package ordersettler

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"math"
	"math/big"
	"strconv"
	"strings"
	"sync"
	"time"

	dbtypes "github.com/skip-mev/go-fast-solver/db"
	"github.com/skip-mev/go-fast-solver/ordersettler/types"
	"github.com/skip-mev/go-fast-solver/shared/contracts/fast_transfer_gateway"
	"github.com/skip-mev/go-fast-solver/shared/metrics"
	"golang.org/x/sync/errgroup"

	"github.com/skip-mev/go-fast-solver/shared/clientmanager"

	"github.com/skip-mev/go-fast-solver/db/gen/db"
	"github.com/skip-mev/go-fast-solver/shared/config"
	"github.com/skip-mev/go-fast-solver/shared/lmt"
	"go.uber.org/zap"
)

type Config struct {
	Delay time.Duration
}

var params = Config{
	Delay: 20 * time.Second,
}

type Database interface {
	GetAllOrderSettlementsWithSettlementStatus(ctx context.Context, settlementStatus string) ([]db.OrderSettlement, error)

	GetOrderByOrderID(ctx context.Context, orderID string) (db.Order, error)

	SetSettlementStatus(ctx context.Context, arg db.SetSettlementStatusParams) (db.OrderSettlement, error)

	SetInitiateSettlementTx(ctx context.Context, arg db.SetInitiateSettlementTxParams) (db.OrderSettlement, error)
	SetCompleteSettlementTx(ctx context.Context, arg db.SetCompleteSettlementTxParams) (db.OrderSettlement, error)

	InsertSubmittedTx(ctx context.Context, arg db.InsertSubmittedTxParams) (db.SubmittedTx, error)

	InsertOrderSettlement(ctx context.Context, arg db.InsertOrderSettlementParams) (db.OrderSettlement, error)
	SetOrderStatus(ctx context.Context, arg db.SetOrderStatusParams) (db.Order, error)

	InTx(ctx context.Context, fn func(ctx context.Context, q db.Querier) error, opts *sql.TxOptions) error
}

type Relayer interface {
	SubmitTxToRelay(ctx context.Context, txHash string, sourceChainID string, maxRelayTxFeeUUSDC *big.Int) error
}

type OrderSettler struct {
	db            Database
	clientManager *clientmanager.ClientManager
	relayer       Relayer
	ordersSeen    map[string]bool
}

func NewOrderSettler(
	ctx context.Context,
	db Database,
	clientManager *clientmanager.ClientManager,
	relayer Relayer,
) (*OrderSettler, error) {
	return &OrderSettler{
		db:            db,
		clientManager: clientManager,
		relayer:       relayer,
		ordersSeen:    make(map[string]bool),
	}, nil
}

// Run looks for any newly fulfilled orders and initiates solver funds settlement flow
func (r *OrderSettler) Run(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case <-time.After(params.Delay):
		}

		if err := r.submitInitiatedSettlementsForRelay(ctx); err != nil {
			lmt.Logger(ctx).Error("error submitting settlements for relay", zap.Error(err))
		}

		if err := r.findNewSettlements(ctx); err != nil {
			lmt.Logger(ctx).Error("error finding new settlements", zap.Error(err))
			continue
		}

		if err := r.settleOrders(ctx); err != nil {
			lmt.Logger(ctx).Error("error settling orders", zap.Error(err))
		}

		if err := r.verifyOrderSettlements(ctx); err != nil {
			lmt.Logger(ctx).Error("error verifying settlements", zap.Error(err))
		}
	}
}

// TODO: feels like this is doing too much
// findNewSettlements queries hyperlane for any fulfilled orders found and creates an order settlement job in the db
func (r *OrderSettler) findNewSettlements(ctx context.Context) error {
	var chains []config.ChainConfig
	cosmosChains, err := config.GetConfigReader(ctx).GetAllChainConfigsOfType(config.ChainType_COSMOS)
	if err != nil {
		return fmt.Errorf("error getting Cosmos chains: %w", err)
	}
	for _, chain := range cosmosChains {
		if chain.FastTransferContractAddress != "" {
			chains = append(chains, chain)
		}
	}

	for _, chain := range chains {
		bridgeClient, err := r.clientManager.GetClient(ctx, chain.ChainID)
		if err != nil {
			return fmt.Errorf("failed to get client: %w", err)
		}

		fills, err := bridgeClient.OrderFillsByFiller(ctx, chain.FastTransferContractAddress, chain.SolverAddress)
		if err != nil {
			return fmt.Errorf("getting order fills: %w", err)
		}
		if len(fills) == 0 {
			// solver has not made any fills on this chain, ignore
			continue
		}

		for _, fill := range fills {
			// continue if order has already been seen
			if r.ordersSeen[fill.OrderID] {
				continue
			}

			sourceChainID, err := config.GetConfigReader(ctx).GetChainIDByHyperlaneDomain(strconv.Itoa(int(fill.SourceDomain)))
			if err != nil {
				return fmt.Errorf("getting source chainID: %w", err)
			}
			sourceGatewayAddress, err := config.GetConfigReader(ctx).GetGatewayContractAddress(sourceChainID)
			if err != nil {
				return fmt.Errorf("getting source gateway address: %w", err)
			}
			sourceBridgeClient, err := r.clientManager.GetClient(ctx, sourceChainID)
			if err != nil {
				return fmt.Errorf("getting client for chainID %s: %w", sourceChainID, err)
			}

			height, err := sourceBridgeClient.BlockHeight(ctx)
			if err != nil {
				return fmt.Errorf("fetching current block height on chain %s: %w", sourceChainID, err)
			}

			// ensure order exists on source chain
			exists, amount, err := sourceBridgeClient.OrderExists(ctx, sourceGatewayAddress, fill.OrderID, big.NewInt(int64(height)))
			if err != nil {
				return fmt.Errorf("checking if order %s exists on chainID %s: %w", fill.OrderID, sourceChainID, err)
			}
			if !exists {
				r.ordersSeen[fill.OrderID] = true
				continue
			}

			// ensure order is not already filled (an order is only marked as
			// filled on the source chain once it is settled)
			status, err := sourceBridgeClient.OrderStatus(ctx, sourceGatewayAddress, fill.OrderID)
			if err != nil {
				return fmt.Errorf("getting order %s status on chainID %s: %w", fill.OrderID, sourceChainID, err)
			}
			if status != fast_transfer_gateway.OrderStatusUnfilled {
				r.ordersSeen[fill.OrderID] = true
				continue
			}

			_, err = r.db.InsertOrderSettlement(ctx, db.InsertOrderSettlementParams{
				SourceChainID:                     sourceChainID,
				DestinationChainID:                chain.ChainID,
				SourceChainGatewayContractAddress: sourceGatewayAddress,
				OrderID:                           fill.OrderID,
				SettlementStatus:                  dbtypes.SettlementStatusPending,
				Amount:                            amount.String(),
			})
			if err != nil && !errors.Is(err, sql.ErrNoRows) {
				return fmt.Errorf("failed to insert settlement: %w", err)
			}
			r.ordersSeen[fill.OrderID] = true
		}
	}
	return nil
}

// settleOrders gets pending settlements out of the db and initiates a
// settlement on the settlements destination chain gateway contract, updating
// the settlements status in the db.
func (r *OrderSettler) settleOrders(ctx context.Context) error {
	batches, err := r.PendingSettlementBatches(ctx)
	if err != nil {
		return fmt.Errorf("getting orders to settle: %w", err)
	}

	var toSettle []types.SettlementBatch
	for _, batch := range batches {
		shouldSettle, err := r.ShouldInitiateSettlement(ctx, batch)
		if err != nil {
			return fmt.Errorf("checking if order settlement should be initiated for batch from source chain %s to destination chain %s: %w", batch.SourceChainID(), batch.DestinationChainID(), err)
		}
		if !shouldSettle {
			lmt.Logger(ctx).Debug(
				"settlement batch is not ready for settlement yet",
				zap.String("sourceChainID", batch.SourceChainID()),
				zap.String("destinationChainID", batch.DestinationChainID()),
			)
			continue
		}
		toSettle = append(toSettle, batch)
	}

	if len(toSettle) == 0 {
		lmt.Logger(ctx).Debug("no settlement batches ready to be settled yet")
		return nil
	}

	lmt.Logger(ctx).Info("initiating order settlements", zap.Stringers("batches", toSettle))

	hashes, err := r.SettleBatches(ctx, toSettle)
	if err != nil {
		return fmt.Errorf("initiating order settlements: %w", err)
	}

	lmt.Logger(ctx).Info("order settlements initiated on chain", zap.Any("hashes", hashes))

	return nil
}

// submitInitiatedSettlementsForRelay finds all order settlements that have
// been initiated on chain and submits them for hyperlane relay.
func (r *OrderSettler) submitInitiatedSettlementsForRelay(ctx context.Context) error {
	initiatedSettlements, err := r.db.GetAllOrderSettlementsWithSettlementStatus(ctx, dbtypes.SettlementStatusSettlementInitiated)
	if err != nil {
		return fmt.Errorf("getting initiated order settlements: %w", err)
	}

	batches := types.IntoSettlementBatchesByHash(initiatedSettlements)

	for _, batch := range batches {
		// these batches are grouped by initiation hash, so just choose the
		// first one since they are all the same
		hash := batch[0].InitiateSettlementTx.String
		if err := r.relayBatch(ctx, hash, batch); err != nil {
			// continue to try and relay other settlements if one fails to be
			// submitted
			lmt.Logger(ctx).Error(
				"submitting batch to be relayed",
				zap.Error(err),
				zap.String("txHash", hash),
				zap.String("settlementPayoutChainID", batch.SourceChainID()),
				zap.String("settlementInitiationChainID", batch.DestinationChainID()),
			)
		}
	}

	return nil
}

// relayBatch submits a tx hash for an initiated batch settlement to be relayed
// from the settlements initiation chain (the orders destination chain), to the
// payout chain (the orders source chain).
func (r *OrderSettler) relayBatch(
	ctx context.Context,
	txHash string,
	batch types.SettlementBatch,
) error {
	// the orders destination chain is where the settlement is initiated
	settlementInitiationChainID := batch.DestinationChainID()

	// the orders source chain is where the settlement is paid out to the solver
	settlementPayoutChainID := batch.SourceChainID()

	maxTxFeeUUSDC, err := r.maxBatchTxFeeUUSDC(ctx, batch)
	if err != nil {
		return fmt.Errorf("calculating max batch (hash: %s) tx fee in uusdc: %w", txHash, err)
	}
	if maxTxFeeUUSDC.Cmp(big.NewInt(0)) <= 0 {
		return fmt.Errorf(
			"batch max tx fee in uusdc is <= 0 based on configured profit margin for %s. min profit margin should be lowered based on current batch size and min fee bps",
			settlementPayoutChainID,
		)
	}

	return r.relaySettlement(
		ctx,
		txHash,
		settlementInitiationChainID,
		settlementPayoutChainID,
		maxTxFeeUUSDC,
	)
}

// relaySettlement submits a tx hash for a settlement to be relayed with
// exponential backoff if an error occurs while submitting the tx to be relayed
func (r *OrderSettler) relaySettlement(
	ctx context.Context,
	txHash string,
	settlementInitiationChainID string,
	settlementPayoutChainID string,
	maxTxFeeUUSDC *big.Int,
) error {
	var (
		maxRetries = 5
		baseDelay  = 2 * time.Second
		err        error
	)

	for i := 0; i < maxRetries; i++ {
		if err = r.relayer.SubmitTxToRelay(ctx, txHash, settlementInitiationChainID, maxTxFeeUUSDC); err == nil {
			return nil
		}
		delay := math.Pow(2, float64(i))
		time.Sleep(time.Duration(delay) * baseDelay)
	}

	return fmt.Errorf(
		"submitting settlement tx hash %s to be relayed from chain %s to chain %s: %w",
		txHash, settlementInitiationChainID, settlementPayoutChainID, err,
	)
}

func (r *OrderSettler) maxBatchTxFeeUUSDC(ctx context.Context, batch types.SettlementBatch) (*big.Int, error) {
	profit, err := r.totalBatchProfit(ctx, batch)
	if err != nil {
		return nil, fmt.Errorf("calculating profit for batch: %w", err)
	}

	totalValue, err := batch.TotalValue()
	if err != nil {
		return nil, fmt.Errorf("calculating total value for batch: %w", err)
	}

	settlementPayoutChainConfig, err := config.GetConfigReader(ctx).GetChainConfig(batch.SourceChainID())
	if err != nil {
		return nil, fmt.Errorf("getting chain config for settlement payout chain %s: %w", batch.SourceChainID(), err)
	}

	minProfitMarginBPS := big.NewFloat(float64(settlementPayoutChainConfig.MinProfitMarginBPS))
	minProfitMarginDec := minProfitMarginBPS.Quo(minProfitMarginBPS, big.NewFloat(10000))
	valueMargin := minProfitMarginDec.Mul(minProfitMarginDec, new(big.Float).SetInt(totalValue))
	valueMarginInt, _ := valueMargin.Int(nil)

	return profit.Sub(profit, valueMarginInt), nil
}

func (r *OrderSettler) totalBatchProfit(ctx context.Context, batch types.SettlementBatch) (*big.Int, error) {
	totalAmountIn, err := batch.TotalValue()
	if err != nil {
		return nil, fmt.Errorf("getting batches total value: %w", err)
	}

	totalAmountOut := big.NewInt(0)
	for _, settlement := range batch {
		// settlements dont store the amount out of the order, in order to
		// calculate the profit, we have to look that up from the db
		order, err := r.db.GetOrderByOrderID(ctx, settlement.OrderID)
		if err != nil {
			return nil, fmt.Errorf("getting order %s for settlement: %w", settlement.OrderID, err)
		}

		amountOut, ok := new(big.Int).SetString(order.AmountOut, 10)
		if !ok {
			return nil, fmt.Errorf("converting order %s's amount out %s to *big.Int", order.OrderID, order.AmountOut)
		}

		totalAmountOut.Add(totalAmountOut, amountOut)
	}

	return totalAmountIn.Sub(totalAmountIn, totalAmountOut), nil
}

// verifyOrderSettlements checks on all instated settlements and updates their
// status in the db with their on chain tx results.
func (r *OrderSettler) verifyOrderSettlements(ctx context.Context) error {
	incompleteSettlements, err := r.IncompleteSettlements(ctx)
	if err != nil {
		return fmt.Errorf("getting incomplete settlements: %w", err)
	}

	for _, settlement := range incompleteSettlements {
		if !settlement.InitiateSettlementTx.Valid {
			continue
		}

		if err = r.verifyOrderSettlement(ctx, settlement); err != nil {
			lmt.Logger(ctx).Warn(
				"failed to verify order settlement, will retry verification on next interval",
				zap.Error(err),
				zap.String("orderID", settlement.OrderID),
				zap.String("sourceChainID", settlement.SourceChainID),
			)
			continue
		}

		lmt.Logger(ctx).Info(
			"successfully verified order settlement",
			zap.String("orderID", settlement.OrderID),
			zap.String("sourceChainID", settlement.SourceChainID),
		)
	}

	return nil
}

// PendingSettlementBatches settlement batches for all orders that are
// currently pending settlement in the db.
func (r *OrderSettler) PendingSettlementBatches(ctx context.Context) ([]types.SettlementBatch, error) {
	pendingSettlements, err := r.db.GetAllOrderSettlementsWithSettlementStatus(ctx, dbtypes.SettlementStatusPending)
	if err != nil {
		return nil, fmt.Errorf("getting orders pending settlement: %w", err)
	}
	var uniniatedSettlements []db.OrderSettlement
	for _, settlement := range pendingSettlements {
		if !settlement.InitiateSettlementTx.Valid {
			uniniatedSettlements = append(uniniatedSettlements, settlement)
		}
	}
	return types.IntoSettlementBatchesByChains(uniniatedSettlements), nil
}

// ShouldInitiateSettlement returns true if a settlement should be initiated
// for a batch based on the uusdc settle up threshold set in the order settler
// config.
func (r *OrderSettler) ShouldInitiateSettlement(ctx context.Context, batch types.SettlementBatch) (bool, error) {
	value, err := batch.TotalValue()
	if err != nil {
		return false, fmt.Errorf("getting settlement batch total value: %w", err)
	}

	sourceChainConfig, err := config.GetConfigReader(ctx).GetChainConfig(batch.SourceChainID())
	if err != nil {
		return false, fmt.Errorf("getting source chain config for chainID %s: %w", batch.SourceChainID(), err)
	}
	settlementThreshold, ok := new(big.Int).SetString(sourceChainConfig.BatchUUSDCSettleUpThreshold, 10)
	if !ok {
		return false, fmt.Errorf(
			"could not convert batch uusdc settle up threshold %s for chainID %s to *big.Int: %w",
			sourceChainConfig.BatchUUSDCSettleUpThreshold,
			batch.SourceChainID(),
			err,
		)
	}

	return value.Cmp(settlementThreshold) >= 0, nil
}

// SettleBatches tries to settle a list settlement batches and update the
// individual settlements status's, returning the tx hash for each initiated
// settlement, in the same order as batches.
func (r *OrderSettler) SettleBatches(ctx context.Context, batches []types.SettlementBatch) ([]string, error) {
	g, gCtx := errgroup.WithContext(ctx)
	hashes := make([]string, len(batches))
	hashesLock := new(sync.Mutex)

	for i, batch := range batches {
		i := i
		batch := batch
		g.Go(func() error {
			hash, err := r.SettleBatch(gCtx, batch)
			if err != nil {
				return fmt.Errorf("settling batch from source chain %s to destination chain %s: %w", batch.SourceChainID(), batch.DestinationChainID(), err)
			}

			hashesLock.Lock()
			defer hashesLock.Unlock()
			hashes[i] = hash

			return nil
		})
	}

	if err := g.Wait(); err != nil {
		return nil, err
	}

	return hashes, nil
}

// SettleBatch initiates a settlement on chain for a SettlementBatch.
func (r *OrderSettler) SettleBatch(ctx context.Context, batch types.SettlementBatch) (string, error) {
	destinationBridgeClient, err := r.clientManager.GetClient(ctx, batch.DestinationChainID())
	if err != nil {
		return "", fmt.Errorf("getting destination bridge client: %w", err)
	}
	txHash, rawTx, err := destinationBridgeClient.InitiateBatchSettlement(ctx, batch)
	if err != nil {
		return "", fmt.Errorf("initiating batch settlement on chain %s: %w", batch.DestinationChainID(), err)
	}
	if rawTx == "" {
		lmt.Logger(ctx).Error(
			"batch settlement rawTx is empty",
			zap.String("batchDestinationChainId", batch.DestinationChainID()),
			zap.Any("batchOrderIDs", batch.OrderIDs()),
		)
		return "", fmt.Errorf("empty batch settlement transaction")
	}

	if err = recordBatchSettlementSubmittedMetric(ctx, batch); err != nil {
		return "", fmt.Errorf("recording batch settlement submitted metrics: %w", err)
	}

	err = r.db.InTx(ctx, func(ctx context.Context, q db.Querier) error {
		// First update all settlements with the initiate settlement tx
		for _, settlement := range batch {
			settlementTx := db.SetInitiateSettlementTxParams{
				SourceChainID:                     settlement.SourceChainID,
				OrderID:                           settlement.OrderID,
				SourceChainGatewayContractAddress: settlement.SourceChainGatewayContractAddress,
				InitiateSettlementTx:              sql.NullString{String: txHash, Valid: true},
			}
			if _, err = q.SetInitiateSettlementTx(ctx, settlementTx); err != nil {
				return fmt.Errorf("setting initiate settlement tx for settlement from source chain %s with order id %s: %w", settlement.SourceChainID, settlement.OrderID, err)
			}
		}
		// we do not insert a submitted tx for each settlement, since many
		// settlements are settled by a single tx (batch settlements)

		// technically this can link back to many order settlement ids,
		// since many settlements are being settled by a single tx.
		// However, we are just choosing the first one here.
		submittedTx := db.InsertSubmittedTxParams{
			OrderSettlementID: sql.NullInt64{Int64: batch[0].ID, Valid: true},
			ChainID:           batch.DestinationChainID(),
			TxHash:            txHash,
			RawTx:             rawTx,
			TxType:            dbtypes.TxTypeSettlement,
			TxStatus:          dbtypes.TxStatusPending,
		}
		if _, err = q.InsertSubmittedTx(ctx, submittedTx); err != nil {
			return fmt.Errorf("inserting raw tx for settlement with hash %s: %w", txHash, err)
		}
		return nil
	}, nil)
	if err != nil {
		return "", fmt.Errorf("recording batch settlement result: %w", err)
	}

	return txHash, nil
}

// recordBatchSettlementSubmittedMetric records a transaction submitted metric for a
// batch settlement
func recordBatchSettlementSubmittedMetric(ctx context.Context, batch types.SettlementBatch) error {
	sourceChainConfig, err := batch.SourceChainConfig(ctx)
	if err != nil {
		return fmt.Errorf("getting source chain config for batch: %w", err)
	}
	destinationChainConfig, err := batch.DestinationChainConfig(ctx)
	if err != nil {
		return fmt.Errorf("getting destination chain config for batch: %w", err)
	}

	metrics.FromContext(ctx).AddTransactionSubmitted(
		err == nil,
		batch.SourceChainID(),
		batch.DestinationChainID(),
		sourceChainConfig.ChainName,
		destinationChainConfig.ChainName,
		string(sourceChainConfig.Environment),
	)

	return nil
}

// verifyOrderSettlement checks if an order settlement tx is complete on chain
// and updates the order settlement status in the db accordingly.
func (r *OrderSettler) verifyOrderSettlement(ctx context.Context, settlement db.OrderSettlement) error {
	sourceBridgeClient, err := r.clientManager.GetClient(ctx, settlement.SourceChainID)
	if err != nil {
		return fmt.Errorf("failed to get client: %w", err)
	}
	destinationBridgeClient, err := r.clientManager.GetClient(ctx, settlement.DestinationChainID)
	if err != nil {
		return fmt.Errorf("failed to get client: %w", err)
	}
	if !settlement.InitiateSettlementTx.Valid {
		return errors.New("message received txHash is null")
	}

	if settlement.SettlementStatus == dbtypes.SettlementStatusPending {
		gasCost, failure, err := destinationBridgeClient.GetTxResult(ctx, settlement.InitiateSettlementTx.String)
		if err != nil {
			// Check if the error is due to tx not found
			if strings.Contains(err.Error(), "tx") && strings.Contains(err.Error(), "not found") && strings.Contains(err.Error(), settlement.InitiateSettlementTx.String) {
				// Transaction not yet indexed, we'll check again later
				return fmt.Errorf("transaction not yet indexed. will retry fetching next interval")
			}
			return fmt.Errorf("failed to fetch message received event: %w", err)
		} else if failure != nil {
			lmt.Logger(ctx).Error("tx failed", zap.String("failure", failure.String()))
			if _, err := r.db.SetSettlementStatus(ctx, db.SetSettlementStatusParams{
				SourceChainID:                     settlement.SourceChainID,
				OrderID:                           settlement.OrderID,
				SourceChainGatewayContractAddress: settlement.SourceChainGatewayContractAddress,
				SettlementStatus:                  dbtypes.SettlementStatusFailed,
				SettlementStatusMessage:           sql.NullString{String: failure.String(), Valid: true},
			}); err != nil {
				return fmt.Errorf("failed to set relay status to failed: %w", err)
			}
			if gasCost == nil {
				return fmt.Errorf("gas cost is nil")
			}
			return fmt.Errorf("failed to fetch message received event: %s", failure.String())
		}

		if _, err := r.db.SetSettlementStatus(ctx, db.SetSettlementStatusParams{
			SourceChainID:                     settlement.SourceChainID,
			OrderID:                           settlement.OrderID,
			SourceChainGatewayContractAddress: settlement.SourceChainGatewayContractAddress,
			SettlementStatus:                  dbtypes.SettlementStatusSettlementInitiated,
		}); err != nil {
			return fmt.Errorf("failed to set relay status to complete: %w", err)
		}
	}

	if settlementIsComplete, err := sourceBridgeClient.IsSettlementComplete(ctx, settlement.SourceChainGatewayContractAddress, settlement.OrderID); err != nil {
		return fmt.Errorf("failed to check if settlement is complete: %w", err)
	} else if settlementIsComplete {
		if _, err := r.db.SetSettlementStatus(ctx, db.SetSettlementStatusParams{
			SourceChainID:                     settlement.SourceChainID,
			OrderID:                           settlement.OrderID,
			SourceChainGatewayContractAddress: settlement.SourceChainGatewayContractAddress,
			SettlementStatus:                  dbtypes.SettlementStatusComplete,
		}); err != nil {
			return fmt.Errorf("failed to set relay status to complete: %w", err)
		}
		return nil
	}

	return fmt.Errorf("settlement is not complete")
}

func (r *OrderSettler) IncompleteSettlements(ctx context.Context) ([]db.OrderSettlement, error) {
	pendingSettlements, err := r.db.GetAllOrderSettlementsWithSettlementStatus(ctx, dbtypes.SettlementStatusPending)
	if err != nil {
		return nil, fmt.Errorf("getting pending settlements: %w", err)
	}

	initiatedSettlements, err := r.db.GetAllOrderSettlementsWithSettlementStatus(ctx, dbtypes.SettlementStatusSettlementInitiated)
	if err != nil {
		return nil, fmt.Errorf("getting initiated settlements: %w", err)
	}

	return append(pendingSettlements, initiatedSettlements...), nil
}

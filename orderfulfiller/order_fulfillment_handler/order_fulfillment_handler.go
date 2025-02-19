package order_fulfillment_handler

import (
	"context"
	"database/sql"
	"fmt"
	"math"
	"math/big"
	"strconv"
	"time"

	dbtypes "github.com/skip-mev/go-fast-solver/db"
	"github.com/skip-mev/go-fast-solver/shared/bridges/cctp"
	"github.com/skip-mev/go-fast-solver/shared/clientmanager"
	"github.com/skip-mev/go-fast-solver/shared/metrics"

	"github.com/skip-mev/go-fast-solver/db/gen/db"
	"github.com/skip-mev/go-fast-solver/shared/config"
	"github.com/skip-mev/go-fast-solver/shared/lmt"
	"go.uber.org/zap"
)

const (
	excessiveOrderFillLatencyDuration = 30 * time.Minute
)

type Relayer interface {
	SubmitTxToRelay(ctx context.Context, txHash string, sourceChainID string, maxRelayTxFeeUUSDC *big.Int) (int64, error)
}

type Database interface {
	GetAllOrdersWithOrderStatus(ctx context.Context, orderStatus string) ([]db.Order, error)

	SetFillTx(ctx context.Context, arg db.SetFillTxParams) (db.Order, error)
	SetOrderStatus(ctx context.Context, arg db.SetOrderStatusParams) (db.Order, error)

	InsertSubmittedTx(ctx context.Context, arg db.InsertSubmittedTxParams) (db.SubmittedTx, error)
	GetSubmittedTxsByOrderIdAndType(ctx context.Context, arg db.GetSubmittedTxsByOrderIdAndTypeParams) ([]db.SubmittedTx, error)

	SetRefundTx(ctx context.Context, arg db.SetRefundTxParams) (db.Order, error)
}

type orderFulfillmentHandler struct {
	db            Database
	clientManager *clientmanager.ClientManager
	relayer       Relayer
}

func NewOrderFulfillmentHandler(db Database, clientManager *clientmanager.ClientManager, relayer Relayer) *orderFulfillmentHandler {
	return &orderFulfillmentHandler{
		db:            db,
		clientManager: clientManager,
		relayer:       relayer,
	}
}

// TODO: feels like this functions is doing too many different things and the
// naming is confusing
func (r *orderFulfillmentHandler) UpdateFulfillmentStatus(ctx context.Context, order db.Order) (fulfillmentStatus string, err error) {
	sourceChainBridgeClient, err := r.clientManager.GetClient(ctx, order.SourceChainID)
	if err != nil {
		return "", fmt.Errorf("failed to get client: %w", err)
	}
	destinationChainBridgeClient, err := r.clientManager.GetClient(ctx, order.DestinationChainID)
	if err != nil {
		return "", fmt.Errorf("failed to get client: %w", err)
	}
	destinationChainGatewayContractAddress, err := config.GetConfigReader(ctx).GetGatewayContractAddress(order.DestinationChainID)
	if err != nil {
		return "", fmt.Errorf("getting gateway contract address for destination chainID %s: %w", order.DestinationChainID, err)
	}
	if order.CreatedAt.Add(excessiveOrderFillLatencyDuration).Before(time.Now()) {
		metrics.FromContext(ctx).IncExcessiveOrderFulfillmentLatency(order.SourceChainID, order.DestinationChainID, order.OrderStatus)
	}

	orderFillEvent, timestamp, err := destinationChainBridgeClient.QueryOrderFillEvent(ctx, destinationChainGatewayContractAddress, order.OrderID)
	if err != nil {
		return "", fmt.Errorf("querying for order fill event on chainID %s at contract %s for order %s: %w", order.DestinationChainID, destinationChainGatewayContractAddress, order.OrderID, err)
	}
	if orderFillEvent != nil {
		// if the order is already filled, set the status to filled
		metrics.FromContext(ctx).IncFillOrderStatusChange(order.SourceChainID, order.DestinationChainID, dbtypes.OrderStatusFilled)
		metrics.FromContext(ctx).ObserveFillLatency(order.SourceChainID, order.DestinationChainID, dbtypes.OrderStatusFilled, time.Since(order.CreatedAt))

		if _, err := r.db.SetFillTx(ctx, db.SetFillTxParams{
			FillTx:                            sql.NullString{String: orderFillEvent.TxHash, Valid: true},
			Filler:                            sql.NullString{String: orderFillEvent.Filler, Valid: true},
			SourceChainID:                     order.SourceChainID,
			OrderID:                           order.OrderID,
			SourceChainGatewayContractAddress: order.SourceChainGatewayContractAddress,
			OrderStatus:                       dbtypes.OrderStatusFilled,
		}); err != nil {
			return "", err
		}
		return dbtypes.OrderStatusFilled, nil
	}

	// if the order is timed out, try and refund the order and update its
	// status
	if isOrderExpired(timestamp, order) {
		isRefunded, refundTxHash, err := sourceChainBridgeClient.IsOrderRefunded(ctx, order.SourceChainGatewayContractAddress, order.OrderID)
		if err != nil {
			return "", fmt.Errorf("querying orderID %s has been refunded on chainID %s: %w", order.OrderID, order.SourceChainID, err)
		}
		if isRefunded {
			metrics.FromContext(ctx).IncFillOrderStatusChange(order.SourceChainID, order.DestinationChainID, dbtypes.OrderStatusRefunded)
			metrics.FromContext(ctx).ObserveFillLatency(order.SourceChainID, order.DestinationChainID, dbtypes.OrderStatusRefunded, time.Since(order.CreatedAt))

			_, err = r.db.SetRefundTx(ctx, db.SetRefundTxParams{
				RefundTx: sql.NullString{
					String: refundTxHash,
					Valid:  true,
				},
				SourceChainID:                     order.SourceChainID,
				OrderID:                           order.OrderID,
				SourceChainGatewayContractAddress: order.SourceChainGatewayContractAddress,
				OrderStatus:                       dbtypes.OrderStatusRefunded,
			})
			if err != nil {
				return "", fmt.Errorf("setting refund tx for orderID %s: %w", order.OrderID, err)
			}

			return dbtypes.OrderStatusRefunded, nil
		}

		if _, err := r.db.SetOrderStatus(ctx, db.SetOrderStatusParams{
			SourceChainID:                     order.SourceChainID,
			OrderID:                           order.OrderID,
			SourceChainGatewayContractAddress: order.SourceChainGatewayContractAddress,
			OrderStatus:                       dbtypes.OrderStatusExpiredPendingRefund,
		}); err != nil {
			return "", err
		}

		if order.OrderStatus == dbtypes.OrderStatusPending {
			metrics.FromContext(ctx).IncFillOrderStatusChange(order.SourceChainID, order.DestinationChainID, dbtypes.OrderStatusExpiredPendingRefund)
		}
		return dbtypes.OrderStatusExpiredPendingRefund, nil

	}

	// order not filled and not timed out, return that the status is pending
	// fulfillment
	return dbtypes.OrderStatusPending, nil
}

func isOrderExpired(expirationTs time.Time, order db.Order) bool {
	return expirationTs.UTC().After(order.TimeoutTimestamp.UTC())
}

func (r *orderFulfillmentHandler) FillOrder(
	ctx context.Context,
	order db.Order,
) (string, error) {
	sourceChainBridgeClient, err := r.clientManager.GetClient(ctx, order.SourceChainID)
	if err != nil {
		return "", fmt.Errorf("failed to get client: %w", err)
	}
	destinationChainBridgeClient, err := r.clientManager.GetClient(ctx, order.DestinationChainID)
	if err != nil {
		return "", fmt.Errorf("failed to get client: %w", err)
	}

	sourceChainConfig, err := config.GetConfigReader(ctx).GetChainConfig(order.SourceChainID)
	if err != nil {
		return "", err
	}
	destinationChainConfig, err := config.GetConfigReader(ctx).GetChainConfig(order.DestinationChainID)
	if err != nil {
		return "", err
	}
	destinationChainGatewayContractAddress, err := config.GetConfigReader(ctx).GetGatewayContractAddress(order.DestinationChainID)
	if err != nil {
		return "", err
	}

	if withinTransferLimits, err := r.checkTransferSize(ctx, destinationChainConfig, order); err != nil {
		return "", fmt.Errorf("checking transfer size for order %s: %w", order.OrderID, err)
	} else if !withinTransferLimits {
		return "", nil
	}

	if acceptableFee, err := r.checkFeeAmount(ctx, order); err != nil {
		return "", fmt.Errorf("checking fee amount for order %s: %w", order.OrderID, err)
	} else if !acceptableFee {
		return "", nil
	}

	if adequateBalance, err := r.checkOrderAssetBalance(ctx, destinationChainBridgeClient, destinationChainConfig, order); err != nil {
		return "", fmt.Errorf("failed to check balance: %w", err)
	} else if !adequateBalance {
		return "", fmt.Errorf("insufficient balance")
	}

	if submittedTxs, err := r.db.GetSubmittedTxsByOrderIdAndType(ctx, db.GetSubmittedTxsByOrderIdAndTypeParams{
		OrderID: sql.NullInt64{Int64: order.ID, Valid: true},
		TxType:  dbtypes.TxTypeOrderFill,
	}); err != nil {
		return "", fmt.Errorf("failed to get submitted txs: %w", err)
	} else if len(submittedTxs) > 0 { // TODO will want to add some retry logic where even if this is > 0, we want to execute an order fill
		return "", nil
	}

	confirmed, err := r.checkBlockConfirmations(ctx, sourceChainConfig, sourceChainBridgeClient, order)
	if err != nil {
		return "", fmt.Errorf("failed to check block confirmations: %w", err)
	} else if !confirmed {
		return "", nil
	}

	txHash, rawTx, _, err := destinationChainBridgeClient.FillOrder(ctx, order, destinationChainGatewayContractAddress)
	metrics.FromContext(ctx).IncTransactionSubmitted(err == nil, order.DestinationChainID, dbtypes.TxTypeOrderFill)
	if err != nil {
		return "", fmt.Errorf("filling order on destination chain at address %s: %w", destinationChainGatewayContractAddress, err)
	}

	if _, err := r.db.InsertSubmittedTx(ctx, db.InsertSubmittedTxParams{
		OrderID:  sql.NullInt64{Int64: order.ID, Valid: true},
		ChainID:  order.DestinationChainID,
		TxHash:   txHash,
		RawTx:    rawTx,
		TxType:   dbtypes.TxTypeOrderFill,
		TxStatus: dbtypes.TxStatusPending,
	}); err != nil {
		return "", fmt.Errorf("failed to insert raw tx %w", err)
	}

	return txHash, nil
}

func (r *orderFulfillmentHandler) checkOrderAssetBalance(ctx context.Context, destinationChainBridgeClient cctp.BridgeClient, destinationChainConfig config.ChainConfig, orderFill db.Order) (adequateBalance bool, err error) {
	balance, err := destinationChainBridgeClient.Balance(ctx, destinationChainConfig.SolverAddress, destinationChainConfig.USDCDenom)
	if err != nil {
		return false, err
	}
	transferAmount, err := strconv.ParseUint(orderFill.AmountOut, 10, 64)
	if err != nil {
		return false, err
	}
	if balance.Cmp(new(big.Int).SetUint64(transferAmount)) < 0 {
		lmt.Logger(ctx).Warn("insufficient balance", zap.String("balance", balance.String()), zap.Uint64("transferAmount", transferAmount))
		metrics.FromContext(ctx).ObserveInsufficientBalanceError(
			destinationChainConfig.ChainID,
			new(big.Int).Sub(new(big.Int).SetUint64(transferAmount), balance).Uint64(),
		)
		return false, nil
	}
	return true, nil
}

func (r *orderFulfillmentHandler) checkTransferSize(ctx context.Context, destinationChainConfig config.ChainConfig, orderFill db.Order) (withinTransferLimits bool, err error) {
	amountIn, ok := new(big.Int).SetString(orderFill.AmountIn, 10)
	if !ok {
		return false, fmt.Errorf("could not convert order amount in %s to *big.Int", orderFill.AmountIn)
	}

	var abandonmentReason string
	var amountOutOfRange int64
	switch {
	case amountIn.Cmp(destinationChainConfig.Cosmos.MinFillSize) < 0:
		abandonmentReason = "transfer amount is below configured min fill size for chain " + orderFill.DestinationChainID
		amountOutOfRange = amountIn.Sub(amountIn, destinationChainConfig.Cosmos.MinFillSize).Int64()
	case amountIn.Cmp(destinationChainConfig.Cosmos.MaxFillSize) > 0:
		abandonmentReason = "transfer amount exceeds configured max fill size for chain" + orderFill.DestinationChainID
		amountOutOfRange = amountIn.Sub(amountIn, destinationChainConfig.Cosmos.MaxFillSize).Int64()
	default:
		return true, nil
	}

	lmt.Logger(ctx).Info(
		"abandoning transaction due to amount exceeding max fill size",
		zap.String("orderID", orderFill.OrderID),
		zap.String("sourceChainID", orderFill.SourceChainID),
		zap.String("orderAmountIn", orderFill.AmountIn),
		zap.String("minAllowedFillSize", destinationChainConfig.Cosmos.MinFillSize.String()),
		zap.String("maxAllowedFillSize", destinationChainConfig.Cosmos.MaxFillSize.String()),
	)

	_, err = r.db.SetOrderStatus(ctx, db.SetOrderStatusParams{
		SourceChainID:                     orderFill.SourceChainID,
		OrderID:                           orderFill.OrderID,
		SourceChainGatewayContractAddress: orderFill.SourceChainGatewayContractAddress,
		OrderStatus:                       dbtypes.OrderStatusAbandoned,
		OrderStatusMessage:                sql.NullString{String: abandonmentReason, Valid: true},
	})
	if err != nil {
		return false, fmt.Errorf("failed to set fill status to abandoned: %w", err)
	}

	lmt.Logger(ctx).Info(
		"abandoning transaction, "+abandonmentReason,
		zap.String("orderID", orderFill.OrderID),
		zap.String("sourceChainID", orderFill.SourceChainID),
		zap.String("orderAmountIn", orderFill.AmountIn),
		zap.String("minAllowedFillSize", destinationChainConfig.Cosmos.MinFillSize.String()),
		zap.String("maxAllowedFillSize", destinationChainConfig.Cosmos.MaxFillSize.String()),
	)

	metrics.FromContext(ctx).IncFillOrderStatusChange(orderFill.SourceChainID, destinationChainConfig.ChainID, dbtypes.OrderStatusAbandoned)
	metrics.FromContext(ctx).ObserveFillLatency(orderFill.SourceChainID, orderFill.DestinationChainID, dbtypes.OrderStatusAbandoned, time.Since(orderFill.CreatedAt))
	metrics.FromContext(ctx).ObserveTransferSizeOutOfRange(
		orderFill.SourceChainID,
		destinationChainConfig.ChainID,
		amountOutOfRange,
	)
	return false, nil
}

// checkFeeAmount checks if an order's solver fee is within the acceptable
// limits to be able to be filled by this solver (based on the configured min
// fee bps). If it is not, the orders state will be set to abandoned in the db.
func (r *orderFulfillmentHandler) checkFeeAmount(ctx context.Context, orderFill db.Order) (bool, error) {
	sourceChainID, err := config.GetConfigReader(ctx).GetChainConfig(orderFill.SourceChainID)
	if err != nil {
		return false, fmt.Errorf("getting config for chainID %s: %w", orderFill.SourceChainID, err)
	}

	isWithinBpsRange, bpsDiff, err := IsWithinBpsRange(ctx, int64(sourceChainID.MinFeeBps), orderFill.AmountIn, orderFill.AmountOut)
	if err != nil {
		return false, fmt.Errorf("checking if order fee for orderID %s is within min bps range: %w", orderFill.OrderID, err)
	}
	if isWithinBpsRange {
		return true, nil
	}

	metrics.FromContext(ctx).IncFillOrderStatusChange(orderFill.SourceChainID, orderFill.DestinationChainID, dbtypes.OrderStatusAbandoned)
	metrics.FromContext(ctx).ObserveFillLatency(orderFill.SourceChainID, orderFill.DestinationChainID, dbtypes.OrderStatusAbandoned, time.Since(orderFill.CreatedAt))

	_, err = r.db.SetOrderStatus(ctx, db.SetOrderStatusParams{
		SourceChainID:                     orderFill.SourceChainID,
		OrderID:                           orderFill.OrderID,
		SourceChainGatewayContractAddress: orderFill.SourceChainGatewayContractAddress,
		OrderStatus:                       dbtypes.OrderStatusAbandoned,
		OrderStatusMessage:                sql.NullString{String: fmt.Sprintf("solver fee for order below configured min fee bps of %d", sourceChainID.MinFeeBps), Valid: true},
	})
	if err != nil {
		return false, fmt.Errorf("failed to set fill status to abandoned: %w", err)
	}

	lmt.Logger(ctx).Info(
		"abandoning transaction due to solver fee smaller than configured min fee bps",
		zap.String("orderID", orderFill.OrderID),
		zap.String("sourceChainID", orderFill.SourceChainID),
		zap.String("orderAmountOut", orderFill.AmountOut),
		zap.Int("minFeeBps", sourceChainID.MinFeeBps),
	)

	metrics.FromContext(ctx).ObserveFeeBpsRejection(
		orderFill.SourceChainID,
		orderFill.DestinationChainID,
		bpsDiff,
	)
	return false, nil
}

// IsWithinBpsRange returns true if the % change between amount in and amount
// out is >= min fee bps. If false, also returns the difference in bps.
func IsWithinBpsRange(ctx context.Context, minFeeBps int64, amountIn, amountOut string) (bool, int64, error) {
	minFee := new(big.Int).SetInt64(minFeeBps)
	in, ok := new(big.Int).SetString(amountIn, 10)
	if !ok {
		return false, 0, fmt.Errorf("converting amount in %s to *big.Int", amountIn)
	}
	out, ok := new(big.Int).SetString(amountOut, 10)
	if !ok {
		return false, 0, fmt.Errorf("converting amount out %s to *big.Int", amountOut)
	}

	minAcceptableFeeScaled := new(big.Int).Mul(minFee, in)
	feeAmount := new(big.Int).Sub(in, out)
	feeAmountScaled := new(big.Int).Mul(feeAmount, big.NewInt(10000))

	actualBps := new(big.Int).Div(feeAmountScaled, in).Int64()
	bpsDiff := minFeeBps - actualBps

	return feeAmountScaled.Cmp(minAcceptableFeeScaled) >= 0, bpsDiff, nil
}

func (r *orderFulfillmentHandler) checkBlockConfirmations(ctx context.Context, sourceChainConfig config.ChainConfig, sourceChainBridgeClient cctp.BridgeClient, order db.Order) (confirmed bool, err error) {
	if height, err := sourceChainBridgeClient.BlockHeight(ctx); err != nil {
		return false, fmt.Errorf("failed to get block height: %w", err)
	} else if uint64(order.OrderCreationTxBlockHeight+sourceChainConfig.NumBlockConfirmationsBeforeFill) > height {
		lmt.Logger(ctx).Debug("required block confirmations not met", zap.String("orderId", order.OrderID), zap.String("sourceChainID", order.SourceChainID))
		return false, nil
	} else {
		exists, _, err := sourceChainBridgeClient.OrderExists(ctx, order.SourceChainGatewayContractAddress, order.OrderID, big.NewInt(order.OrderCreationTxBlockHeight))
		if err != nil {
			return false, err
		}
		if !exists {
			metrics.FromContext(ctx).IncFillOrderStatusChange(order.SourceChainID, order.DestinationChainID, dbtypes.OrderStatusAbandoned)
			metrics.FromContext(ctx).ObserveFillLatency(order.SourceChainID, order.DestinationChainID, dbtypes.OrderStatusAbandoned, time.Since(order.CreatedAt))

			if _, err := r.db.SetOrderStatus(ctx, db.SetOrderStatusParams{
				SourceChainID:                     order.SourceChainID,
				OrderID:                           order.OrderID,
				SourceChainGatewayContractAddress: order.SourceChainGatewayContractAddress,
				OrderStatus:                       dbtypes.OrderStatusAbandoned,
				OrderStatusMessage:                sql.NullString{String: "reorged", Valid: true},
			}); err != nil {
				return false, fmt.Errorf("failed to set fill status to abandoned: %w", err)
			}
			lmt.Logger(ctx).Info("abandoning transaction due to reorg", zap.String("orderId", order.OrderID), zap.String("sourceChainID", order.SourceChainID))
		}
		return true, nil
	}
}

func (r *orderFulfillmentHandler) InitiateTimeout(ctx context.Context, order db.Order) (string, error) {
	destinationChainBridgeClient, err := r.clientManager.GetClient(ctx, order.DestinationChainID)
	if err != nil {
		return "", fmt.Errorf("failed to get client: %w", err)
	}

	destinationChainGatewayContractAddress, err := config.GetConfigReader(ctx).GetGatewayContractAddress(order.DestinationChainID)
	if err != nil {
		return "", err
	}

	submittedTxs, err := r.db.GetSubmittedTxsByOrderIdAndType(ctx, db.GetSubmittedTxsByOrderIdAndTypeParams{
		OrderID: sql.NullInt64{Int64: order.ID, Valid: true},
		TxType:  dbtypes.TxTypeInitiateTimeout,
	})
	if err != nil {
		return "", fmt.Errorf("failed to get submitted txs: %w", err)
	}
	if len(submittedTxs) > 1 {
		return "", fmt.Errorf("got more %d submitted tx's for order %s with type %s, expected only 1", len(submittedTxs), order.OrderStatusMessage.String, dbtypes.TxTypeInitiateTimeout)
	}
	if len(submittedTxs) == 1 {
		// the timeout for this order has already been submitted, return the tx
		// hash
		return submittedTxs[0].TxHash, nil
	}

	txHash, rawTx, _, err := destinationChainBridgeClient.InitiateTimeout(ctx, order, destinationChainGatewayContractAddress)
	metrics.FromContext(ctx).IncTransactionSubmitted(err == nil, order.DestinationChainID, dbtypes.TxTypeInitiateTimeout)
	if err != nil {
		return "", fmt.Errorf("initiating timeout: %w", err)
	}
	if txHash == "" {
		return "", fmt.Errorf("empty tx hash after submitting order for timeout to address %s", destinationChainGatewayContractAddress)
	}

	if _, err := r.db.InsertSubmittedTx(ctx, db.InsertSubmittedTxParams{
		OrderID:  sql.NullInt64{Int64: order.ID, Valid: true},
		ChainID:  order.DestinationChainID,
		TxHash:   txHash,
		RawTx:    rawTx,
		TxType:   dbtypes.TxTypeInitiateTimeout,
		TxStatus: dbtypes.TxStatusPending,
	}); err != nil {
		return "", fmt.Errorf("failed to insert raw tx %w", err)
	}

	lmt.Logger(ctx).Info(
		"successfully initiated timeout",
		zap.String("orderID", order.OrderID),
		zap.String("sourceChainID", order.SourceChainID),
		zap.String("destinationChainID", order.DestinationChainID),
	)

	return txHash, nil
}

func (r *orderFulfillmentHandler) SubmitTimeoutForRelay(ctx context.Context, order db.Order, txHash string) error {
	// the source chain for the relay is the chain that the timeout was
	// initiated on, which is the orders destination chain
	initiateTimeoutChain := order.DestinationChainID

	var (
		maxRetries = 5
		baseDelay  = 2 * time.Second
		err        error
	)

	for i := 0; i < maxRetries; i++ {
		if _, err = r.relayer.SubmitTxToRelay(ctx, txHash, initiateTimeoutChain, nil); err == nil {
			return nil
		}
		delay := math.Pow(2, float64(i))
		time.Sleep(time.Duration(delay) * baseDelay)
	}

	return fmt.Errorf(
		"submitting settlement tx hash %s to be relayed from chain %s to chain %s: %w",
		txHash, initiateTimeoutChain, order.SourceChainID, err,
	)
}

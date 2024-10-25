package hyperlane

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"math/big"
	"time"

	dbtypes "github.com/skip-mev/go-fast-solver/db"
	"github.com/skip-mev/go-fast-solver/db/gen/db"
	"github.com/skip-mev/go-fast-solver/shared/config"
	"github.com/skip-mev/go-fast-solver/shared/lmt"
	"go.uber.org/zap"
)

const (
	relayInterval = 10 * time.Second
)

type Database interface {
	GetAllOrderSettlementsWithSettlementStatus(ctx context.Context, settlementStatus string) ([]db.OrderSettlement, error)
	InsertHyperlaneTransfer(ctx context.Context, arg db.InsertHyperlaneTransferParams) (db.HyperlaneTransfer, error)
	SetMessageStatus(ctx context.Context, arg db.SetMessageStatusParams) (db.HyperlaneTransfer, error)
	GetSubmittedTxsByHyperlaneTransferId(ctx context.Context, hyperlaneTransferID sql.NullInt64) ([]db.SubmittedTx, error)
	GetAllHyperlaneTransfersWithTransferStatus(ctx context.Context, transferStatus string) ([]db.HyperlaneTransfer, error)
	InsertSubmittedTx(ctx context.Context, arg db.InsertSubmittedTxParams) (db.SubmittedTx, error)
	GetSubmittedTxsByOrderStatusAndType(ctx context.Context, arg db.GetSubmittedTxsByOrderStatusAndTypeParams) ([]db.SubmittedTx, error)
	GetAllOrdersWithOrderStatus(ctx context.Context, orderStatus string) ([]db.Order, error)
}

type RelayerRunner struct {
	db           Database
	hyperlane    Client
	relayHandler Relayer
}

func NewRelayerRunner(db Database, hyperlaneClient Client, relayer Relayer) *RelayerRunner {
	return &RelayerRunner{
		db:           db,
		hyperlane:    hyperlaneClient,
		relayHandler: relayer,
	}
}

func (r *RelayerRunner) Run(ctx context.Context) error {
	ticker := time.NewTicker(relayInterval)
	for {
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
			// find settlement txns in the db and insert them as pending
			// hyperlane txns in the hyperlane table
			if err := r.findSettlementsToRelay(ctx); err != nil {
				return fmt.Errorf("finding settlements txns to relay: %w", err)
			}

			// find timeout txns in the db and insert them as pending hyperlane
			// txns in the hyperlane table
			if err := r.findTimeoutsToRelay(ctx); err != nil {
				return fmt.Errorf("finding timeout txns to relay: %w", err)
			}

			// grab all pending hyperlane transfers from the db
			transfers, err := r.db.GetAllHyperlaneTransfersWithTransferStatus(ctx, dbtypes.TransferStatusPending)
			if err != nil {
				return fmt.Errorf("getting pending hyperlane transfers: %w", err)
			}

			for _, transfer := range transfers {
				shouldRelay, err := r.checkHyperlaneTransferStatus(ctx, transfer)
				if err != nil {
					lmt.Logger(ctx).Error(
						"error checking hyperlane transfer status",
						zap.Error(err),
						zap.String("sourceChainID", transfer.SourceChainID),
						zap.String("txHash", transfer.MessageSentTx),
					)
					continue
				}
				if !shouldRelay {
					continue
				}

				var opts RelayOpts
				if transfer.MaxGasPricePct.Valid && transfer.TransferValue.Valid {
					totalRelayValue, ok := new(big.Int).SetString(transfer.TransferValue.String, 10)
					if !ok {
						lmt.Logger(ctx).Error("could not convert relay transfer value to *big.Int", zap.String("transferValue", transfer.TransferValue.String))
						continue
					}
					opts.Profitability = &Profitability{
						MaxGasPricePct:  uint8(transfer.MaxGasPricePct.Int64),
						TotalRelayValue: totalRelayValue,
					}
				}
				destinationTxHash, destinationChainID, err := r.relayHandler.Relay(ctx, transfer.SourceChainID, transfer.MessageSentTx, opts)
				if err != nil {
					lmt.Logger(ctx).Error(
						"error relaying pending hyperlane transfer",
						zap.Error(err),
						zap.String("sourceChainID", transfer.SourceChainID),
						zap.String("txHash", transfer.MessageSentTx),
					)
					continue
				}

				if _, err := r.db.InsertSubmittedTx(ctx, db.InsertSubmittedTxParams{
					HyperlaneTransferID: sql.NullInt64{Int64: transfer.ID, Valid: true},
					ChainID:             destinationChainID,
					TxHash:              destinationTxHash,
					RawTx:               "",
					TxType:              dbtypes.TxTypeHyperlaneMessageDelivery,
					TxStatus:            dbtypes.TxStatusPending,
				}); err != nil {
					lmt.Logger(ctx).Error(
						"error inserting submitted tx for hyperlane transfer",
						zap.Error(err),
						zap.String("sourceChainID", transfer.SourceChainID),
						zap.String("txHash", transfer.MessageSentTx),
					)
				}
			}
		}
	}
}

// checkHyperlaneTransferStatus checks if a hyperlane transfer should be
// relayed or not
func (r *RelayerRunner) checkHyperlaneTransferStatus(ctx context.Context, transfer db.HyperlaneTransfer) (shouldRelay bool, err error) {
	destinationChainConfig, err := config.GetConfigReader(ctx).GetChainConfig(transfer.DestinationChainID)
	if err != nil {
		return false, fmt.Errorf("getting destination chain config for chainID %s: %w", transfer.DestinationChainID, err)
	}
	delivered, err := r.hyperlane.HasBeenDelivered(ctx, destinationChainConfig.HyperlaneDomain, transfer.MessageID)
	if err != nil {
		return false, fmt.Errorf("checking if message with id %s has been delivered: %w", transfer.MessageID, err)
	}
	if delivered {
		if _, err := r.db.SetMessageStatus(ctx, db.SetMessageStatusParams{
			TransferStatus:     dbtypes.TransferStatusSuccess,
			SourceChainID:      transfer.SourceChainID,
			DestinationChainID: transfer.DestinationChainID,
			MessageID:          transfer.MessageID,
		}); err != nil {
			return false, fmt.Errorf("setting message status to success: %w", err)
		}
		lmt.Logger(ctx).Info(
			"message has already been delivered",
			zap.String("sourceChainID", transfer.SourceChainID),
			zap.String("destinationChainID", transfer.DestinationChainID),
			zap.String("messageID", transfer.MessageID),
		)
		return false, nil
	}

	txs, err := r.db.GetSubmittedTxsByHyperlaneTransferId(ctx, sql.NullInt64{Int64: transfer.ID, Valid: true})
	if err != nil {
		return false, fmt.Errorf("getting submitted txs by hyperlane transfer id %d: %w", transfer.ID, err)
	}
	if len(txs) > 0 {
		// for now we will not attempt to submit the hyperlane message more than once.
		// this is to avoid the gas cost of repeatedly landing a failed hyperlane delivery tx.
		// in the future we may add more sophistication around retries
		lmt.Logger(ctx).Info(
			"delivery attempt already made for message",
			zap.String("sourceChainID", transfer.SourceChainID),
			zap.String("destinationChainID", transfer.DestinationChainID),
			zap.String("messageID", transfer.MessageID),
			zap.String("deliveryAttemptTxHash", txs[0].TxHash),
		)
		return false, nil
	}

	return true, nil
}

// settlements that have been initiated and creates pending hyperlane transfers
// from them
func (r *RelayerRunner) findSettlementsToRelay(ctx context.Context) error {
	// poll the order settler db table and check if there are any new order
	// settlements that should be relayed
	pendingSettlements, err := r.db.GetAllOrderSettlementsWithSettlementStatus(ctx, dbtypes.SettlementStatusSettlementInitiated)
	if err != nil {
		return fmt.Errorf("getting pending order settlements: %w", err)
	}

	for _, pending := range pendingSettlements {
		if !pending.InitiateSettlementTx.Valid {
			lmt.Logger(ctx).Debug(
				"found pending order settlement without valid initiate settlement tx",
				zap.Int64("ID", pending.ID),
			)
			continue
		}

		destinationChainConfig, err := config.GetConfigReader(ctx).GetChainConfig(pending.DestinationChainID)
		if err != nil {
			return fmt.Errorf("getting destination chain config for chainID %s: %w", pending.DestinationChainID, err)
		}

		dispatch, _, err := r.hyperlane.GetHyperlaneDispatch(ctx, destinationChainConfig.HyperlaneDomain, pending.DestinationChainID, pending.InitiateSettlementTx.String)
		if err != nil {
			return fmt.Errorf("parsing tx results: %w", err)
		}

		destinationChainID, err := config.GetConfigReader(ctx).GetChainIDByHyperlaneDomain(dispatch.DestinationDomain)
		if err != nil {
			return fmt.Errorf("getting destination chainID by hyperlane domain %s: %w", dispatch.DestinationDomain, err)
		}

		if _, err := r.db.InsertHyperlaneTransfer(ctx, db.InsertHyperlaneTransferParams{
			SourceChainID:      pending.DestinationChainID,
			DestinationChainID: destinationChainID,
			MessageID:          dispatch.MessageID,
			MessageSentTx:      pending.InitiateSettlementTx.String,
			TransferStatus:     dbtypes.TransferStatusPending,
		}); err != nil && !errors.Is(err, sql.ErrNoRows) {
			return fmt.Errorf("inserting hyperlane transfer: %w", err)
		}
	}
	return nil
}

func (r *RelayerRunner) findTimeoutsToRelay(ctx context.Context) error {
	// poll the order settler db table and check if there are any new order
	// settlements that should be relayed
	timeoutTxs, err := r.db.GetSubmittedTxsByOrderStatusAndType(ctx, db.GetSubmittedTxsByOrderStatusAndTypeParams{
		OrderStatus: dbtypes.OrderStatusExpiredPendingRefund,
		TxType:      dbtypes.TxTypeInitiateTimeout,
	})
	if err != nil {
		return fmt.Errorf("getting submitted txs for expired orders pending refunds: %w", err)
	}

	for _, timeoutTx := range timeoutTxs {
		sourceChainConfig, err := config.GetConfigReader(ctx).GetChainConfig(timeoutTx.ChainID)
		if err != nil {
			return fmt.Errorf("getting source chain config for chainID %s: %w", timeoutTx.ChainID, err)
		}

		dispatch, _, err := r.hyperlane.GetHyperlaneDispatch(ctx, sourceChainConfig.HyperlaneDomain, timeoutTx.ChainID, timeoutTx.TxHash)
		if err != nil {
			return fmt.Errorf("parsing tx results: %w", err)
		}

		destinationChainID, err := config.GetConfigReader(ctx).GetChainIDByHyperlaneDomain(dispatch.DestinationDomain)
		if err != nil {
			return fmt.Errorf("getting destination chainID by hyperlane domain %s: %w", dispatch.DestinationDomain, err)
		}

		if _, err := r.db.InsertHyperlaneTransfer(ctx, db.InsertHyperlaneTransferParams{
			SourceChainID:      timeoutTx.ChainID,
			DestinationChainID: destinationChainID,
			MessageID:          dispatch.MessageID,
			MessageSentTx:      timeoutTx.TxHash,
			TransferStatus:     dbtypes.TransferStatusPending,
		}); err != nil && !errors.Is(err, sql.ErrNoRows) {
			return fmt.Errorf("inserting hyperlane transfer: %w", err)
		}
	}
	return nil
}

// RelayOpts provides users options for how the relayer should be have when
// relaying a tx.
type RelayOpts struct {
	// Profitability provides relaying options regarding how profitable it is to
	// relay a tx. Typically this would be used when the relayer is relaying a tx
	// bound for itself, and it should only relay that tx under profitable
	// conditions (i.e. not pay too much for gas, relative to the value that it is
	// relaying).
	Profitability *Profitability
}

type Profitability struct {
	// MaxGasPricePct is the max percentage of the total value of the relayed
	// tx that the relayer is willing to pay in gas fees.
	MaxGasPricePct uint8

	// TotalRelayValue is the total value of the relayed tx, denominated in uusdc.
	TotalRelayValue *big.Int
}

func (opts RelayOpts) validate() error {
	if opts.Profitability == nil {
		return nil
	}
	if opts.Profitability.TotalRelayValue == nil {
		return fmt.Errorf("invalid relay options: total relay value undefined")
	}
	if opts.Profitability.MaxGasPricePct <= 0 {
		return fmt.Errorf("invalid relay options: max gas price pct must be > 0")
	}
	if opts.Profitability.MaxGasPricePct > 100 {
		return fmt.Errorf("invalid relay options: max gas price pct must be <= 100")
	}
	return nil
}

// SubmitTxToRelay submits a transaction hash on a source chain to be relayed.
// This transaction must contain a dispatch message/event that can be relayed
// by hyperlane. This tx will not be immediately relayed but will be placed in
// a queue to be eventually relayed.
func (r *RelayerRunner) SubmitTxToRelay(ctx context.Context, txHash string, sourceChainID string, opts RelayOpts) error {
	if err := opts.validate(); err != nil {
		return fmt.Errorf("validating relay opts: %w", err)
	}

	sourceChainConfig, err := config.GetConfigReader(ctx).GetChainConfig(sourceChainID)
	if err != nil {
		return fmt.Errorf("getting source chain config for chainID %s: %w", sourceChainID, err)
	}

	dispatch, _, err := r.hyperlane.GetHyperlaneDispatch(ctx, sourceChainConfig.HyperlaneDomain, sourceChainID, txHash)
	if err != nil {
		return fmt.Errorf("parsing tx results: %w", err)
	}

	destinationChainID, err := config.GetConfigReader(ctx).GetChainIDByHyperlaneDomain(dispatch.DestinationDomain)
	if err != nil {
		return fmt.Errorf("getting destination chainID by hyperlane domain %s: %w", dispatch.DestinationDomain, err)
	}

	insert := db.InsertHyperlaneTransferParams{
		SourceChainID:      sourceChainID,
		DestinationChainID: destinationChainID,
		MessageID:          dispatch.MessageID,
		MessageSentTx:      txHash,
		TransferStatus:     dbtypes.TransferStatusPending,
	}
	if opts.Profitability != nil {
		insert.MaxGasPricePct = sql.NullInt64{Int64: int64(opts.Profitability.MaxGasPricePct), Valid: true}
		insert.TransferValue = sql.NullString{String: opts.Profitability.TotalRelayValue.String(), Valid: true}
	}
	if _, err := r.db.InsertHyperlaneTransfer(ctx, insert); err != nil && !errors.Is(err, sql.ErrNoRows) {
		return fmt.Errorf("inserting hyperlane transfer: %w", err)
	}

	return nil
}

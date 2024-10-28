package hyperlane

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	dbtypes "github.com/skip-mev/go-fast-solver/db"
	"github.com/skip-mev/go-fast-solver/db/gen/db"
	"github.com/skip-mev/go-fast-solver/shared/config"
	"github.com/skip-mev/go-fast-solver/shared/lmt"
	"github.com/skip-mev/go-fast-solver/shared/metrics"
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
				metrics.FromContext(ctx).IncDatabaseErrors(dbtypes.GET)
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

				destinationTxHash, destinationChainID, err := r.relayHandler.Relay(ctx, transfer.SourceChainID, transfer.MessageSentTx)
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
					metrics.FromContext(ctx).IncDatabaseErrors(dbtypes.INSERT)
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
		metrics.FromContext(ctx).IncHyperlaneMessages(transfer.SourceChainID, transfer.DestinationChainID, dbtypes.TransferStatusSuccess)
		metrics.FromContext(ctx).DecHyperlaneMessages(transfer.SourceChainID, transfer.DestinationChainID, dbtypes.TransferStatusPending)
		metrics.FromContext(ctx).ObserveHyperlaneLatency(transfer.SourceChainID, transfer.DestinationChainID, dbtypes.TransferStatusSuccess, time.Since(transfer.CreatedAt))

		if _, err := r.db.SetMessageStatus(ctx, db.SetMessageStatusParams{
			TransferStatus:     dbtypes.TransferStatusSuccess,
			SourceChainID:      transfer.SourceChainID,
			DestinationChainID: transfer.DestinationChainID,
			MessageID:          transfer.MessageID,
		}); err != nil {
			metrics.FromContext(ctx).IncDatabaseErrors(dbtypes.UPDATE)
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
		metrics.FromContext(ctx).IncDatabaseErrors(dbtypes.GET)
		return false, fmt.Errorf("getting submitted txs by hyperlane transfer id %d: %w", transfer.ID, err)
	}
	if len(txs) > 0 {
		metrics.FromContext(ctx).IncHyperlaneMessages(transfer.SourceChainID, transfer.DestinationChainID, dbtypes.TransferStatusAbandoned)
		metrics.FromContext(ctx).DecHyperlaneMessages(transfer.SourceChainID, transfer.DestinationChainID, dbtypes.TransferStatusPending)
		metrics.FromContext(ctx).ObserveHyperlaneLatency(transfer.SourceChainID, transfer.DestinationChainID, dbtypes.TransferStatusAbandoned, time.Since(transfer.CreatedAt))

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

// findSettlementsToRelay checks the order settlement table for any new
// settlements that have been initiated and creates pending hyperlane transfers
// from them
func (r *RelayerRunner) findSettlementsToRelay(ctx context.Context) error {
	// poll the order settler db table and check if there are any new order
	// settlements that should be relayed
	pendingSettlements, err := r.db.GetAllOrderSettlementsWithSettlementStatus(ctx, dbtypes.SettlementStatusSettlementInitiated)
	if err != nil {
		metrics.FromContext(ctx).IncDatabaseErrors(dbtypes.GET)
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

		metrics.FromContext(ctx).IncHyperlaneMessages(pending.SourceChainID, destinationChainID, dbtypes.TransferStatusPending)
		if _, err := r.db.InsertHyperlaneTransfer(ctx, db.InsertHyperlaneTransferParams{
			SourceChainID:      pending.DestinationChainID,
			DestinationChainID: destinationChainID,
			MessageID:          dispatch.MessageID,
			MessageSentTx:      pending.InitiateSettlementTx.String,
			TransferStatus:     dbtypes.TransferStatusPending,
		}); err != nil && !errors.Is(err, sql.ErrNoRows) {
			metrics.FromContext(ctx).IncDatabaseErrors(dbtypes.INSERT)
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
		metrics.FromContext(ctx).IncDatabaseErrors(dbtypes.GET)
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

		metrics.FromContext(ctx).IncHyperlaneMessages(timeoutTx.ChainID, destinationChainID, dbtypes.TransferStatusPending)
		if _, err := r.db.InsertHyperlaneTransfer(ctx, db.InsertHyperlaneTransferParams{
			SourceChainID:      timeoutTx.ChainID,
			DestinationChainID: destinationChainID,
			MessageID:          dispatch.MessageID,
			MessageSentTx:      timeoutTx.TxHash,
			TransferStatus:     dbtypes.TransferStatusPending,
		}); err != nil && !errors.Is(err, sql.ErrNoRows) {
			metrics.FromContext(ctx).IncDatabaseErrors(dbtypes.INSERT)
			return fmt.Errorf("inserting hyperlane transfer: %w", err)
		}
	}
	return nil
}

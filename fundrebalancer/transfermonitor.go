package fundrebalancer

import (
	"context"
	"fmt"
	"time"

	"github.com/skip-mev/go-fast-solver/db"
	genDB "github.com/skip-mev/go-fast-solver/db/gen/db"
	"github.com/skip-mev/go-fast-solver/shared/clients/skipgo"
	"github.com/skip-mev/go-fast-solver/shared/lmt"
	"github.com/skip-mev/go-fast-solver/shared/metrics"
	"go.uber.org/zap"
)

// TransferTracker is responsible for updating the database with the latest status of funds rebalancing txs
// (does not update user transfer txs, that is done by the tx verifier module)
type TransferTracker struct {
	skipgo   skipgo.SkipGoClient
	database Database
}

func NewTransferTracker(skipgo skipgo.SkipGoClient, db Database) *TransferTracker {
	return &TransferTracker{
		skipgo:   skipgo,
		database: db,
	}
}

func (t *TransferTracker) TrackPendingTransfers(ctx context.Context) {
	const pollInterval = 2 * time.Second
	const initialPollInterval = 1 * time.Nanosecond
	ticker := time.NewTicker(initialPollInterval)

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			ticker.Stop()

			if err := t.UpdateTransfers(ctx); err != nil {
				lmt.Logger(ctx).Error("got an error updating transfers", zap.Error(err))
			}

			ticker.Reset(pollInterval)
		}
	}
}

// UpdateTransfers checks all pending rebalance transfers in the database with
// their current status according to Skip Go. If the transfers have completed
// or errored, it updates their status in the db.
func (t *TransferTracker) UpdateTransfers(ctx context.Context) error {
	pendingTransfers, err := t.database.GetAllPendingRebalanceTransfers(ctx)
	if err != nil {
		return fmt.Errorf("getting all pending rebalance transfers: %w", err)
	}

	for _, pendingTransfer := range pendingTransfers {
		err := t.updateTransferStatus(ctx, pendingTransfer.ID, pendingTransfer.CreatedAt, pendingTransfer.TxHash, pendingTransfer.SourceChainID, pendingTransfer.DestinationChainID)
		if err != nil {
			lmt.Logger(ctx).Error(
				"error tracking transfer",
				zap.Error(err),
				zap.Int64("id", pendingTransfer.ID),
				zap.String("hash", pendingTransfer.TxHash),
				zap.String("souceChainID", pendingTransfer.SourceChainID),
				zap.String("destinationChainID", pendingTransfer.DestinationChainID),
			)
			continue
		}
	}

	return nil
}

func (t *TransferTracker) updateTransferStatus(ctx context.Context, transferID int64, createdAt time.Time, hash, sourceChainID, destinationChainID string) error {
	if time.Since(createdAt) > transferTimeout {
		err := t.database.UpdateTransferStatus(ctx, genDB.UpdateTransferStatusParams{
			Status: db.RebalanceTransferStatusAbandoned,
			ID:     transferID,
		})
		if err != nil {
			return fmt.Errorf("updating transfer status to abandoned for hash %s on chain %s: %w", hash, sourceChainID, err)
		}
		return nil
	}

	txHash, err := t.skipgo.TrackTx(ctx, hash, sourceChainID)
	if err != nil {
		return fmt.Errorf("failed to track transaction %s on chain %s: %w", hash, sourceChainID, err)
	}

	currentStatus, err := t.skipgo.Status(ctx, txHash, sourceChainID)
	if err != nil {
		return fmt.Errorf("getting status for transaction %s on chain %s: %w", hash, sourceChainID, err)
	}

	// check if all transfers in the status are done
	allTransfersDone := true
	var latestState skipgo.TransactionState
	for _, transfer := range currentStatus.Transfers {
		if !transfer.State.IsCompleted() {
			allTransfersDone = false
			latestState = transfer.State
			break
		}
	}

	if !allTransfersDone {
		lmt.Logger(ctx).Debug(
			"waiting for transaction to complete",
			zap.String("latestState", string(latestState)),
			zap.String("txnHash", hash),
			zap.String("sourceChainID", sourceChainID),
			zap.String("destinationChainID", destinationChainID),
		)
		return nil
	}

	// all transfers have finished, grab the first error if any
	var transferError string
	for _, transfer := range currentStatus.Transfers {
		// report the first error that occurred, if any
		if transfer.State.IsCompletedError() {
			if transfer.Error != nil {
				transferError = *transfer.Error
			} else {
				transferError = "error occurred during transfer but reason could not be found. state is " + string(transfer.State)
			}
		}
	}

	if transferError != "" {
		lmt.Logger(ctx).Info(
			"rebalance transaction completed with an error",
			zap.String("txnHash", hash),
			zap.String("sourceChainID", sourceChainID),
			zap.String("destinationChainID", destinationChainID),
			zap.String("error", transferError),
		)
		metrics.FromContext(ctx).IncFundsRebalanceTransferStatusChange(sourceChainID, destinationChainID, db.RebalanceTransferStatusFailed)

		err = t.database.UpdateTransferStatus(ctx, genDB.UpdateTransferStatusParams{
			Status: db.RebalanceTransferStatusFailed,
			ID:     transferID,
		})
		if err != nil {
			return fmt.Errorf("updating transfer status to failed for hash %s on chain %s: %w", hash, sourceChainID, err)
		}

		return nil
	}

	lmt.Logger(ctx).Info(
		"rebalance transaction completed successfully",
		zap.String("txnHash", hash),
		zap.String("sourceChainID", sourceChainID),
		zap.String("destinationChainID", destinationChainID),
	)
	metrics.FromContext(ctx).IncFundsRebalanceTransferStatusChange(sourceChainID, destinationChainID, db.RebalanceTransferStatusSuccess)

	err = t.database.UpdateTransferStatus(ctx, genDB.UpdateTransferStatusParams{
		Status: db.RebalanceTransferStatusSuccess,
		ID:     transferID,
	})
	if err != nil {
		return fmt.Errorf("updating transfer status to completed for hash %s on chain %s: %w", hash, sourceChainID, err)
	}

	return nil
}

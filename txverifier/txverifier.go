package txverifier

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	dbtypes "github.com/skip-mev/go-fast-solver/db"
	"github.com/skip-mev/go-fast-solver/shared/bridges/cctp"
	"github.com/skip-mev/go-fast-solver/shared/clientmanager"
	"github.com/skip-mev/go-fast-solver/shared/metrics"

	"go.uber.org/zap"
	"golang.org/x/sync/errgroup"

	"github.com/skip-mev/go-fast-solver/db/gen/db"
	"github.com/skip-mev/go-fast-solver/shared/lmt"
)

const (
	txAbandonedTimeout = 10 * time.Minute
)

type Config struct {
	Delay time.Duration
}

var params = Config{
	Delay: 5 * time.Second,
}

type Database interface {
	GetSubmittedTxsWithStatus(ctx context.Context, txStatus string) ([]db.SubmittedTx, error)
	SetSubmittedTxStatus(ctx context.Context, arg db.SetSubmittedTxStatusParams) (db.SubmittedTx, error)
}

type TxVerifier struct {
	db            Database
	clientManager *clientmanager.ClientManager
}

func NewTxVerifier(ctx context.Context, db Database, clientManager *clientmanager.ClientManager) (*TxVerifier, error) {
	return &TxVerifier{
		db:            db,
		clientManager: clientManager,
	}, nil
}

// Run calls verifyTxs in a loop to update the status of any pending txs in the database
func (r *TxVerifier) Run(ctx context.Context) {
	for {
		r.verifyTxs(ctx)

		select {
		case <-ctx.Done():
			return
		case <-time.After(params.Delay):
		}
	}
}

func (r *TxVerifier) verifyTxs(ctx context.Context) {
	submittedTxs, err := r.db.GetSubmittedTxsWithStatus(ctx, dbtypes.TxStatusPending)
	if err != nil {
		lmt.Logger(ctx).Error("error getting pending txs", zap.Error(err))
		return
	}

	eg, egCtx := errgroup.WithContext(ctx)
	for _, submittedTx := range submittedTxs {
		submittedTx := submittedTx
		eg.Go(func() error {
			if err := r.VerifyTx(egCtx, submittedTx); err != nil {
				lmt.Logger(ctx).Warn(
					"error in VerifyTx stage",
					zap.Error(err),
					zap.String("txHash", submittedTx.TxHash),
					zap.String("chainID", submittedTx.ChainID),
				)
			} else {
				lmt.Logger(ctx).Info("successful VerifyTx stage", zap.String("txHash", submittedTx.TxHash), zap.String("chainID", submittedTx.ChainID))
			}
			return nil
		})
	}

	if err := eg.Wait(); err != nil {
		lmt.Logger(ctx).Error("error processing submittedTxs", zap.Error(err))
	}
}

// VerifyTx retrieves the tx status from the bridge responsible for relaying the tx, and updates the tx in the
// database with the latest status
func (r *TxVerifier) VerifyTx(ctx context.Context, submittedTx db.SubmittedTx) error {
	bridgeClient, err := r.clientManager.GetClient(ctx, submittedTx.ChainID)
	if err != nil {
		return fmt.Errorf("failed to get client: %w", err)
	}
	_, failure, err := bridgeClient.GetTxResult(ctx, submittedTx.TxHash)
	if err != nil {
		if errors.As(err, &cctp.ErrTxResultNotFound{}) {
			return r.handleTxResultNotFound(ctx, submittedTx)
		}

		return fmt.Errorf("failed to get tx result: %w", err)
	} else if failure != nil {
		lmt.Logger(ctx).Error("tx failed", zap.String("failure", failure.String()))
		metrics.FromContext(ctx).IncTransactionVerified(false, submittedTx.ChainID)
		if _, err := r.db.SetSubmittedTxStatus(ctx, db.SetSubmittedTxStatusParams{
			TxStatus:        dbtypes.TxStatusFailed,
			TxHash:          submittedTx.TxHash,
			ChainID:         submittedTx.ChainID,
			TxStatusMessage: sql.NullString{String: failure.String(), Valid: true},
		}); err != nil {
			return fmt.Errorf("failed to set tx status to failed: %w", err)
		}
		return fmt.Errorf("tx failed: %s", failure.String())
	} else {
		metrics.FromContext(ctx).IncTransactionVerified(true, submittedTx.ChainID)
		if _, err := r.db.SetSubmittedTxStatus(ctx, db.SetSubmittedTxStatusParams{
			TxStatus: dbtypes.TxStatusSuccess,
			TxHash:   submittedTx.TxHash,
			ChainID:  submittedTx.ChainID,
		}); err != nil {
			return fmt.Errorf("failed to set tx status to success: %w", err)
		}
	}
	return nil
}

func (r *TxVerifier) handleTxResultNotFound(ctx context.Context, submittedTx db.SubmittedTx) error {
	if time.Since(submittedTx.CreatedAt) > txAbandonedTimeout {
		if _, err := r.db.SetSubmittedTxStatus(ctx, db.SetSubmittedTxStatusParams{
			TxStatus: dbtypes.TxStatusAbandoned,
			TxHash:   submittedTx.TxHash,
			ChainID:  submittedTx.ChainID,
		}); err != nil {
			return fmt.Errorf("failed to set tx status to abandoned: %w", err)
		}
	}

	return nil
}

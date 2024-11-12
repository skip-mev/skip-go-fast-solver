package hyperlane

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"math/big"
	"strings"
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
	RelaySubmitter
	GetAllOrderSettlementsWithSettlementStatus(ctx context.Context, settlementStatus string) ([]db.OrderSettlement, error)
	SetMessageStatus(ctx context.Context, arg db.SetMessageStatusParams) (db.HyperlaneTransfer, error)
	GetSubmittedTxsByHyperlaneTransferId(ctx context.Context, hyperlaneTransferID sql.NullInt64) ([]db.SubmittedTx, error)
	GetAllHyperlaneTransfersWithTransferStatus(ctx context.Context, transferStatus string) ([]db.HyperlaneTransfer, error)
	InsertSubmittedTx(ctx context.Context, arg db.InsertSubmittedTxParams) (db.SubmittedTx, error)
	GetSubmittedTxsByOrderStatusAndType(ctx context.Context, arg db.GetSubmittedTxsByOrderStatusAndTypeParams) ([]db.SubmittedTx, error)
	GetAllOrdersWithOrderStatus(ctx context.Context, orderStatus string) ([]db.Order, error)
}

type RelaySubmitter interface {
	InsertHyperlaneTransfer(ctx context.Context, arg db.InsertHyperlaneTransferParams) (db.HyperlaneTransfer, error)
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

				destinationTxHash, destinationChainID, err := r.relayTransfer(ctx, transfer)
				if err != nil {
					switch {
					case errors.Is(err, ErrRelayTooExpensive):
						lmt.Logger(ctx).Warn(
							"relaying transfer is too expensive, waiting for better conditions",
							zap.String("sourceChainID", transfer.SourceChainID),
							zap.String("txHash", transfer.MessageSentTx),
						)
					case errors.Is(err, ErrNotEnoughSignaturesFound):
						// warning already logged in relayer
						continue
					case strings.Contains(err.Error(), "execution reverted"):
						// Unrecoverable error
						lmt.Logger(ctx).Warn(
							"abandoning hyperlane transfer",
							zap.Int64("transferId", transfer.ID),
							zap.String("txHash", transfer.MessageSentTx),
							zap.Error(err),
						)

						if _, err := r.db.SetMessageStatus(ctx, db.SetMessageStatusParams{
							TransferStatus:        dbtypes.TransferStatusAbandoned,
							SourceChainID:         transfer.SourceChainID,
							DestinationChainID:    transfer.DestinationChainID,
							MessageID:             transfer.MessageID,
							TransferStatusMessage: sql.NullString{String: err.Error(), Valid: true},
						}); err != nil {
							lmt.Logger(ctx).Error(
								"error updating invalid transfer status",
								zap.Int64("transferId", transfer.ID),
								zap.String("txHash", transfer.MessageSentTx),
								zap.Error(err),
							)
						}
						continue
					default:
						lmt.Logger(ctx).Error(
							"error relaying pending hyperlane transfer",
							zap.Error(err),
							zap.String("sourceChainID", transfer.SourceChainID),
							zap.String("txHash", transfer.MessageSentTx),
						)
					}
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

// relayTransfer constructs relay options and calls the relayer to relay
// preform a hyperlane relay on a dispatch message. Returning the destination
// chain tx hash and the destination chain id.
func (r *RelayerRunner) relayTransfer(ctx context.Context, transfer db.HyperlaneTransfer) (string, string, error) {
	var opts RelayOpts
	if transfer.MaxTxFeeUusdc.Valid {
		maxTxFeeUUSDC, ok := new(big.Int).SetString(transfer.MaxTxFeeUusdc.String, 10)
		if !ok {
			return "", "", fmt.Errorf("could not convert relay max tx fee uusdc value %s to *big.Int", transfer.MaxTxFeeUusdc.String)
		}
		opts.MaxTxFeeUUSDC = maxTxFeeUUSDC
	}

	destinationTxHash, destinationChainID, err := r.relayHandler.Relay(ctx, transfer.SourceChainID, transfer.MessageSentTx, opts)
	if err != nil {
		return "", "", fmt.Errorf("relaying pending hyperlane transfer with tx hash %s from chainID %s: %w", transfer.MessageSentTx, transfer.SourceChainID, err)
	}

	return destinationTxHash, destinationChainID, err
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

// RelayOpts provides users options for how the relayer should behave when
// relaying a tx.
type RelayOpts struct {
	MaxTxFeeUUSDC *big.Int

	// Submitter allows for users to customize the back end for how this relay
	// submission is recorded. Typically this is used to allow users to have
	// the relay submission run in a transaction, since submitting tx to be
	// relayed often times must be atomic with some other actions.
	Submitter RelaySubmitter

	// Delay is how long the submitter should wait before checking on chain for
	// the tx hash.
	Delay time.Duration
}

func (opts RelayOpts) validate() error {
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

	time.Sleep(opts.Delay)

	dispatch, _, err := r.hyperlane.GetHyperlaneDispatch(ctx, sourceChainConfig.HyperlaneDomain, sourceChainID, txHash)
	if err != nil {
		return fmt.Errorf("parsing tx results: %w", err)
	}

	destinationChainID, err := config.GetConfigReader(ctx).GetChainIDByHyperlaneDomain(dispatch.DestinationDomain)
	if err != nil {
		return fmt.Errorf("getting destination chainID by hyperlane domain %s: %w", dispatch.DestinationDomain, err)
	}

	var submitter RelaySubmitter = r.db
	if opts.Submitter != nil {
		submitter = opts.Submitter
	}

	insert := db.InsertHyperlaneTransferParams{
		SourceChainID:      sourceChainID,
		DestinationChainID: destinationChainID,
		MessageID:          dispatch.MessageID,
		MessageSentTx:      txHash,
		TransferStatus:     dbtypes.TransferStatusPending,
	}
	if opts.MaxTxFeeUUSDC != nil {
		insert.MaxTxFeeUusdc = sql.NullString{String: opts.MaxTxFeeUUSDC.String(), Valid: true}
	}
	if _, err := submitter.InsertHyperlaneTransfer(ctx, insert); err != nil && !errors.Is(err, sql.ErrNoRows) {
		return fmt.Errorf("inserting hyperlane transfer: %w", err)
	}

	return nil
}

package hyperlane

import (
	"encoding/hex"
	"errors"
	"fmt"
	"math/big"

	dbtypes "github.com/skip-mev/go-fast-solver/db"
	"github.com/skip-mev/go-fast-solver/shared/metrics"

	"strings"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/skip-mev/go-fast-solver/hyperlane/types"

	"github.com/skip-mev/go-fast-solver/shared/config"
	"github.com/skip-mev/go-fast-solver/shared/lmt"
	"go.uber.org/zap"
	"golang.org/x/net/context"
)

type Relayer interface {
	Relay(ctx context.Context, originChainID string, initiateTxHash string, maxTxFeeUUSDC *big.Int) (destinationTxHash string, destinationChainID string, err error)
}

type relayer struct {
	hyperlane                Client
	storageLocationOverrides map[string]string
}

func NewRelayer(hyperlaneClient Client, storageLocationOverrides map[string]string) Relayer {
	return &relayer{
		hyperlane:                hyperlaneClient,
		storageLocationOverrides: storageLocationOverrides,
	}
}

var (
	ErrRelayTooExpensive        = fmt.Errorf("relay is too expensive")
	ErrMessageAlreadyDelivered  = fmt.Errorf("message has already been delivered")
	ErrNotEnoughSignaturesFound = errors.New("number of signatures found in multisig signed checkpoint is below expected threshold")
)

func (r *relayer) Relay(ctx context.Context, originChainID string, initiateTxHash string, maxTxFeeUUSDC *big.Int) (destinationTxHash string, destinationChainID string, err error) {
	originChainConfig, err := config.GetConfigReader(ctx).GetChainConfig(originChainID)
	if err != nil {
		return "", "", fmt.Errorf("getting chain config for chainID %s: %w", originChainID, err)
	}
	dispatch, merkleHookPostDispatch, err := r.hyperlane.GetHyperlaneDispatch(ctx, originChainConfig.HyperlaneDomain, originChainID, initiateTxHash)
	if err != nil {
		return "", "", fmt.Errorf("parsing tx results: %w", err)
	}
	destinationChainID, err = config.GetConfigReader(ctx).GetChainIDByHyperlaneDomain(dispatch.DestinationDomain)
	if err != nil {
		return "", "", fmt.Errorf("getting destination chainID by hyperlane domain %s: %w", dispatch.DestinationDomain, err)
	}
	destinationChainConfig, err := config.GetConfigReader(ctx).GetChainConfig(destinationChainID)
	if err != nil {
		return "", "", fmt.Errorf("getting destination chain config for chainID %s: %w", destinationChainID, err)
	}

	delivered, err := r.hyperlane.HasBeenDelivered(ctx, dispatch.DestinationDomain, dispatch.MessageID)
	if err != nil {
		return "", "", fmt.Errorf("checking if message with id %s has been delivered: %w", dispatch.MessageID, err)
	}
	if delivered {
		return "", "", ErrMessageAlreadyDelivered
	}

	isContract, err := r.hyperlane.IsContract(ctx, dispatch.DestinationDomain, dispatch.Recipient)
	if err != nil {
		return "", "", fmt.Errorf("checking if recipient %s is a contract: %w", dispatch.Recipient, err)
	}
	if !isContract {
		return "", "", fmt.Errorf("recipient %s is not a contract", dispatch.Recipient)
	}

	// fetch all validators that should validate this message according to the
	// destination chains ism, and get how many of them need to validate
	validators, threshold, err := r.hyperlane.ValidatorsAndThreshold(ctx, dispatch.DestinationDomain, dispatch.Recipient, dispatch.Message)
	if err != nil {
		return "", "", fmt.Errorf("getting validators and quorum threshold from doamin %s for recipient %s: %w", dispatch.DestinationDomain, dispatch.Recipient, err)
	}
	if len(validators) == 0 {
		return "", "", fmt.Errorf("no validator set received from multisig ism")
	}

	lmt.Logger(ctx).Debug(
		"got validators and threshold from recipient ism",
		zap.Any("validators", validators),
		zap.Uint8("threshold", threshold),
	)

	// get the checkpoint storage locations for these validators via the origin
	// chains validator announce contract
	validatorStorageLocations, err := r.hyperlane.ValidatorStorageLocations(ctx, originChainConfig.HyperlaneDomain, validators)
	if err != nil {
		return "", "", fmt.Errorf("getting validator storage locations on domain %s for validators %v: %w", originChainConfig.HyperlaneDomain, validators, err)
	}

	lmt.Logger(ctx).Debug(
		"got validator storage locations",
		zap.Any("validatorStorageLocations", validatorStorageLocations),
	)

	// create fetchers for the validators storage locations (either S3 or local
	// files)
	var checkpointFetchers []CheckpointFetcher
	for _, validatorStorageLocation := range validatorStorageLocations {
		validator := validatorStorageLocation.Validator
		storageLocation := validatorStorageLocation.StorageLocation
		if override, ok := r.storageLocationOverrides[validator]; ok {
			storageLocation = override
		}

		fetcher, err := NewCheckpointFetcherFromStorageLocation(storageLocation, validator)
		if err != nil {
			return "", "", fmt.Errorf("creating checkpoint fetcher from storage location %s for validator %s: %w", storageLocation, validator, err)
		}
		checkpointFetchers = append(checkpointFetchers, fetcher)
	}

	// fetch the checkpoint at index if we have reached a quorum of validators
	// there
	quorumCheckpoint, err := r.checkpointAtIndex(ctx, merkleHookPostDispatch.Index, checkpointFetchers, threshold, dispatch.MessageID)
	if err != nil {
		return "", "", fmt.Errorf("getting checkpoint at index %d: %w", merkleHookPostDispatch.Index, err)
	}

	lmt.Logger(ctx).Debug("found checkpoint with quorum", zap.Uint64("index", merkleHookPostDispatch.Index))

	// convert the checkpoint to metadata to be passed to the destination ism
	// for verification
	metadata, err := quorumCheckpoint.ToMetadata()
	if err != nil {
		return "", "", fmt.Errorf("creating message metadata from multisig checkpoint: %w", err)
	}

	// submit the message to the destination mailbox for processing (ism
	// verification, emit events, calling recipient contract)
	message, err := hex.DecodeString(dispatch.Message)
	if err != nil {
		return "", "", fmt.Errorf("hex decoding dispatch message to bytes: %w", err)
	}

	// if the user specified a max tx fee, ensure that the tx fee to relay will
	// be less than this amount
	if maxTxFeeUUSDC != nil {
		isFeeLessThanMax, err := r.isRelayFeeLessThanMax(ctx, dispatch.DestinationDomain, message, metadata, maxTxFeeUUSDC)
		if err != nil {
			return "", "", fmt.Errorf("checking if relay to domain %s is profitable: %w", dispatch.DestinationDomain, err)
		}
		if !isFeeLessThanMax {
			metrics.FromContext(ctx).IncHyperlaneRelayTooExpensive(originChainID, destinationChainID)
			return "", "", ErrRelayTooExpensive
		}
	}

	hash, err := r.hyperlane.Process(ctx, dispatch.DestinationDomain, message, metadata)
	metrics.FromContext(ctx).IncTransactionSubmitted(err == nil, destinationChainID, dbtypes.TxTypeHyperlaneMessageDelivery)
	if err != nil {
		return "", "", fmt.Errorf("processing message on domain %s: %w", dispatch.DestinationDomain, err)
	}

	lmt.Logger(ctx).Info(
		fmt.Sprintf("relayed hyperlane message from %s to %s", originChainConfig.ChainName, destinationChainConfig.ChainName),
		zap.String("originDispatchTxHash", initiateTxHash),
		zap.String("destinationProcessTxHash", hex.EncodeToString(hash)),
	)

	return hex.EncodeToString(hash), destinationChainID, nil
}

func (r *relayer) checkpointAtIndex(
	ctx context.Context,
	index uint64,
	checkpointFetchers []CheckpointFetcher,
	threshold uint8,
	messageID string,
) (types.MultiSigSignedCheckpoint, error) {
	var multiSigCheckpoint types.MultiSigSignedCheckpoint
	signedCheckpointsPerRoot := make(map[string][]types.SignedCheckpoint)
	for _, fetcher := range checkpointFetchers {
		signedCheckpoint, err := fetcher.Checkpoint(ctx, index)
		if errors.Is(err, ErrCheckpointDoesNotExist) {
			// if the validator for this fetcher has not signed the
			// checkpoint, ignore it
			continue
		}
		if err != nil {
			metrics.FromContext(ctx).IncHyperlaneCheckpointingErrors()
			return types.MultiSigSignedCheckpoint{}, fmt.Errorf("fetching checkpoint at index %d: %w", index, err)
		}

		// ensure that the checkpoint is actually for this index
		if uint64(signedCheckpoint.Value.Checkpoint.Index) != index {
			lmt.Logger(ctx).Debug(
				"checkpoint index mismatch",
				zap.Uint64("expected", index),
				zap.Uint32("got", signedCheckpoint.Value.Checkpoint.Index),
			)
			continue
		}

		digest, err := signedCheckpoint.Digest()
		if err != nil {
			return types.MultiSigSignedCheckpoint{}, fmt.Errorf("hex decoding checkpoint root: %w", err)
		}
		pubkey, err := signedCheckpoint.Signature.RecoverPubKey(digest)
		if err != nil {
			return types.MultiSigSignedCheckpoint{}, fmt.Errorf("recovering pubkey from signature: %w", err)
		}
		signature, err := signedCheckpoint.Signature.RSBytes()
		if err != nil {
			return types.MultiSigSignedCheckpoint{}, fmt.Errorf("converting checkpoint signature to bytes: %w", err)
		}
		if !crypto.VerifySignature(pubkey, digest, signature) {
			lmt.Logger(ctx).Warn(
				"checkpoint signature is not from validator",
				zap.String("validator", fetcher.Validator()),
				zap.Uint64("checkpointIndex", index),
			)
			continue
		}

		root := signedCheckpoint.Value.Checkpoint.Root
		if _, ok := signedCheckpointsPerRoot[root]; !ok {
			signedCheckpointsPerRoot[root] = make([]types.SignedCheckpoint, 0)
		}
		signedCheckpointsPerRoot[root] = append(signedCheckpointsPerRoot[root], *signedCheckpoint)

		if len(signedCheckpointsPerRoot[root]) >= int(threshold) {
			multiSigCheckpoint.Checkpoint = signedCheckpoint.Value
			for _, checkpoint := range signedCheckpointsPerRoot[root] {
				multiSigCheckpoint.Signatures = append(multiSigCheckpoint.Signatures, checkpoint.Signature)
			}
			break
		}
	}
	if len(multiSigCheckpoint.Signatures) < int(threshold) {
		lmt.Logger(ctx).Warn("failed to find expected number of signatures in multisig signed checkpoint",
			zap.Uint8("threshold", threshold), zap.Int("num_signatures_found", len(multiSigCheckpoint.Signatures)))

		return types.MultiSigSignedCheckpoint{}, ErrNotEnoughSignaturesFound
	}
	if strings.TrimPrefix(multiSigCheckpoint.Checkpoint.MessageID, "0x") != strings.TrimPrefix(messageID, "0x") {
		return types.MultiSigSignedCheckpoint{}, fmt.Errorf("mismatch message id in checkpoint and dipsatch message. dispatch has %s and checkpoint has %s", messageID, multiSigCheckpoint.Checkpoint.MessageID)
	}
	if uint64(multiSigCheckpoint.Checkpoint.Checkpoint.Index) != index {
		return types.MultiSigSignedCheckpoint{}, fmt.Errorf("mismatch index in checkpoint and merkle root post dispatch. merkle root post dispatch has %d and checkpoint has %d", index, multiSigCheckpoint.Checkpoint.Checkpoint.Index)
	}

	return multiSigCheckpoint, nil
}

var (
	ErrCouldNotDetermineRelayFee = fmt.Errorf("could not determine relay fee")
)

// isRelayFeeLessThanMax simulates a relay of a message and checks that the fee
// to relay the message is less than the users specified max relay fee in uusdc
func (r *relayer) isRelayFeeLessThanMax(
	ctx context.Context,
	domain string,
	message []byte,
	metadata []byte,
	maxTxFeeUUSDC *big.Int,
) (bool, error) {
	txFeeUUSDC, err := r.hyperlane.QuoteProcessUUSDC(ctx, domain, message, metadata)
	if err != nil {
		if strings.Contains(err.Error(), "execution reverted") {
			// if the quote process call has reverted, we return a sentinel
			// error so that callers can specifically handle this case
			return false, ErrCouldNotDetermineRelayFee
		}
		return false, fmt.Errorf("quoting process call in uusdc: %w", err)
	}

	// dont ever expect for a tx fee to be negative or 0, log a warning here
	if txFeeUUSDC.Cmp(big.NewInt(0)) <= 0 {
		lmt.Logger(ctx).Warn("tx fee uusdc was <= 0", zap.String("txFeeUUSDC", txFeeUUSDC.String()))
		return false, nil
	}

	isFeeLessThanMax := txFeeUUSDC.Cmp(maxTxFeeUUSDC) <= 0

	if isFeeLessThanMax {
		lmt.Logger(ctx).Info(
			"tx fee to relay message is less than max allowed fee",
			zap.String("estimatedTxFeeUUSDC", txFeeUUSDC.String()),
			zap.String("maxTxFeeUUSDC", maxTxFeeUUSDC.String()),
		)
	} else {
		lmt.Logger(ctx).Info(
			"tx fee to relay message is more than max allowed fee",
			zap.String("estimatedTxFeeUUSDC", txFeeUUSDC.String()),
			zap.String("maxTxFeeUUSDC", maxTxFeeUUSDC.String()),
		)
	}

	return isFeeLessThanMax, nil
}

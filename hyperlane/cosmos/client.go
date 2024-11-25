package cosmos

import (
	"context"
	"cosmossdk.io/math"
	"crypto/tls"
	"encoding/hex"
	"fmt"
	"github.com/skip-mev/go-fast-solver/shared/lmt"
	"go.uber.org/zap"
	"math/big"
	"strings"
	"time"

	"google.golang.org/grpc/credentials"

	"strconv"

	wasmtypes "github.com/CosmWasm/wasmd/x/wasm/types"
	coretypes "github.com/cometbft/cometbft/rpc/core/types"
	"github.com/ethereum/go-ethereum/common"
	"github.com/skip-mev/go-fast-solver/hyperlane/types"
	"github.com/skip-mev/go-fast-solver/shared/config"
	"github.com/skip-mev/go-fast-solver/shared/tmrpc"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type HyperlaneClient struct {
	client                   wasmtypes.QueryClient
	hyperlaneDomain          string
	validatorAnnounceAddress string
	merkleHookAddress        string
	mailBoxAddress           string
	tmRPCManager             tmrpc.TendermintRPCClientManager
	chainID                  string
}

func NewHyperlaneClient(ctx context.Context, hyperlaneDomain string) (*HyperlaneClient, error) {
	chainID, err := config.GetConfigReader(ctx).GetChainIDByHyperlaneDomain(hyperlaneDomain)
	if err != nil {
		return nil, fmt.Errorf("getting chainID from hyperlane domain %s: %w", hyperlaneDomain, err)
	}

	chainConfig, err := config.GetConfigReader(ctx).GetChainConfig(chainID)
	if err != nil {
		return nil, fmt.Errorf("getting config for chain %s: %w", chainID, err)
	}

	creds := insecure.NewCredentials()
	if chainConfig.Cosmos.GRPCTLSEnabled {
		creds = credentials.NewTLS(&tls.Config{
			InsecureSkipVerify: true,
		})
	}

	conn, err := grpc.DialContext(ctx, chainConfig.Cosmos.GRPC, grpc.WithTransportCredentials(creds))
	if err != nil {
		return nil, fmt.Errorf("dialing grpc address %s: %w", chainConfig.Cosmos.GRPC, err)
	}

	return &HyperlaneClient{
		client:                   wasmtypes.NewQueryClient(conn),
		hyperlaneDomain:          hyperlaneDomain,
		validatorAnnounceAddress: chainConfig.Relayer.ValidatorAnnounceContractAddress,
		merkleHookAddress:        chainConfig.Relayer.MerkleHookContractAddress,
		mailBoxAddress:           chainConfig.Relayer.MailboxAddress,
		tmRPCManager:             tmrpc.NewTendermintRPCClientManager(),
		chainID:                  chainID,
	}, nil
}

func ParseTxResults(tx *coretypes.ResultTx) (*types.MailboxDispatchEvent, *types.MailboxMerkleHookPostDispatchEvent, error) {
	dispatch, err := ParseDispatch(tx)
	if err != nil {
		return nil, nil, fmt.Errorf("parsing dispatch event from tx %s results: %w", tx.Hash.String(), err)
	}

	merkleHookPostDispatch, err := ParseMerkleHookPostDispatch(tx)
	if err != nil {
		return nil, nil, fmt.Errorf("parsing merkle hook post dispatch event from tx %s results: %w", tx.Hash.String(), err)
	}

	return dispatch, merkleHookPostDispatch, nil
}

func ParseMerkleHookPostDispatch(tx *coretypes.ResultTx) (*types.MailboxMerkleHookPostDispatchEvent, error) {
	const merkleHookPostDispatchEventType = "wasm-hpl_hook_merkle::post_dispatch"

	var d types.MailboxMerkleHookPostDispatchEvent
	found := false
	for _, event := range tx.TxResult.Events {
		switch event.Type {
		case merkleHookPostDispatchEventType:
			if found {
				return nil, fmt.Errorf("found multiple merkle hook post dispatch events in tx results")
			}
			found = true
			for _, attribute := range event.Attributes {
				switch attribute.Key {
				case "message_id":
					d.MessageID = attribute.Value
				case "index":
					idx, err := strconv.Atoi(attribute.Value)
					if err != nil {
						return nil, fmt.Errorf("converting index value %s to int: %w", attribute.Value, err)
					}
					d.Index = uint64(idx)
				}
			}
		}
	}
	if !found {
		return nil, fmt.Errorf("could not find merkle hook post dispatch event type %s", merkleHookPostDispatchEventType)
	}

	return &d, nil
}

func ParseDispatch(tx *coretypes.ResultTx) (*types.MailboxDispatchEvent, error) {
	const dispatchEventType = "wasm-mailbox_dispatch"
	const dispatchIDEventType = "wasm-mailbox_dispatch_id"

	var d types.MailboxDispatchEvent
	dispatchFound := false
	dispatchMessageIDFound := false
	for _, event := range tx.TxResult.Events {
		switch event.Type {
		case dispatchEventType:
			if dispatchFound {
				return nil, fmt.Errorf("found multiple dispatch events in tx results")
			}
			dispatchFound = true
			for _, attribute := range event.Attributes {
				switch attribute.Key {
				case "recipient":
					d.Recipient = attribute.Value
				case "sender":
					d.Sender = attribute.Value
				case "destination":
					d.DestinationDomain = attribute.Value
				case "_contract_address":
					d.SenderMailbox = attribute.Value
				case "message":
					d.Message = attribute.Value
				}
			}
		case dispatchIDEventType:
			if dispatchMessageIDFound {
				return nil, fmt.Errorf("found multiple dispatch message id events in tx results")
			}
			dispatchMessageIDFound = true
			for _, attribute := range event.Attributes {
				switch attribute.Key {
				case "message_id":
					d.MessageID = attribute.Value
				}
			}
		}
	}

	if !dispatchFound {
		return nil, fmt.Errorf("could not find dispatch event type %s", dispatchEventType)
	}
	if !dispatchMessageIDFound {
		return nil, fmt.Errorf("could not find dipatch message id event type %s", dispatchIDEventType)
	}

	return &d, nil
}

func (c *HyperlaneClient) GetHyperlaneDispatch(ctx context.Context, domain, originChainID, initiateTxHash string) (*types.MailboxDispatchEvent, *types.MailboxMerkleHookPostDispatchEvent, error) {
	tmRpcClient, err := c.tmRPCManager.GetClient(ctx, originChainID)
	if err != nil {
		return nil, nil, fmt.Errorf("getting tendermint rpc client for chain %s: %w", originChainID, err)
	}
	txHashBytes, err := hex.DecodeString(initiateTxHash)
	if err != nil {
		return nil, nil, fmt.Errorf("decoding tx hash %s: %w", initiateTxHash, err)
	}
	tx, err := tmRpcClient.Tx(ctx, txHashBytes, false)
	if err != nil {
		return nil, nil, fmt.Errorf("fetching tx results, hash: %s: %w", initiateTxHash, err)
	}
	return ParseTxResults(tx)
}

func (c *HyperlaneClient) HasBeenDelivered(ctx context.Context, domain string, messageID string) (bool, error) {
	if domain != c.hyperlaneDomain {
		return false, fmt.Errorf("expected domain %s but got %s", c.hyperlaneDomain, domain)
	}

	panic("not implemented")
}

func (c *HyperlaneClient) ISMType(ctx context.Context, domain string, recipient string) (uint8, error) {
	if domain != c.hyperlaneDomain {
		return 0, fmt.Errorf("expected domain %s but got %s", c.hyperlaneDomain, domain)
	}

	panic("not implemented")
}

func (c *HyperlaneClient) ValidatorsAndThreshold(
	ctx context.Context,
	domain string,
	recipient string,
	message string,
) ([]common.Address, uint8, error) {
	if domain != c.hyperlaneDomain {
		return nil, 0, fmt.Errorf("expected domain %s but got %s", c.hyperlaneDomain, domain)
	}

	panic("not implemented")
}

func (c *HyperlaneClient) ValidatorStorageLocations(
	ctx context.Context,
	domain string,
	validators []common.Address,
) ([]*types.ValidatorStorageLocation, error) {
	if domain != c.hyperlaneDomain {
		return nil, fmt.Errorf("expected domain %s but got %s", c.hyperlaneDomain, domain)
	}

	var validatorStrs []string
	for _, va := range validators {
		validatorStrs = append(validatorStrs, va.String())
	}

	querier := NewValidatorAnnounceQuerier(c.validatorAnnounceAddress, c.client)
	validatorStorageLocations, err := querier.GetAnnouncedValidatorStorageLocations(ctx, validatorStrs)
	if err != nil {
		return nil, fmt.Errorf("getting storage locations for validators %+v: %w", validators, err)
	}

	return validatorStorageLocations, nil
}

func (c *HyperlaneClient) MerkleTreeLeafCount(ctx context.Context, domain string) (uint64, error) {
	if domain != c.hyperlaneDomain {
		return 0, fmt.Errorf("expected domain %s but got %s", c.hyperlaneDomain, domain)
	}
	return NewMerkleTreeHookQuerier(c.merkleHookAddress, c.client).Count(ctx)
}

func (c *HyperlaneClient) Process(ctx context.Context, domain string, message []byte, metadata []byte) ([]byte, error) {
	panic("not implemented")
}

func (c *HyperlaneClient) QuoteProcessUUSDC(ctx context.Context, domain string, message []byte, metadata []byte) (*big.Int, error) {
	panic("not implemented")
}

func (c *HyperlaneClient) IsContract(ctx context.Context, domain, address string) (bool, error) {
	panic("not implemented")
}

func (c *HyperlaneClient) ListHyperlaneMessageSentTxs(ctx context.Context, domain string, startBlockHeight uint) ([]types.HyperlaneMessage, error) {
	client, err := c.tmRPCManager.GetClient(ctx, c.chainID)
	if err != nil {
		return nil, err
	}
	status, err := client.Status(ctx)
	if err != nil {
		lmt.Logger(ctx).Error("Error fetching status", zap.Error(err))
		return nil, err
	}
	//blocksProcessedPerIteration := uint(1000)
	startBlockHeight = math.Max(startBlockHeight, uint(status.SyncInfo.EarliestBlockHeight))
	//endBlockHeight := math.Min(uint(status.SyncInfo.LatestBlockHeight), startBlockHeight+blocksProcessedPerIteration)
	endBlockHeight := uint(status.SyncInfo.LatestBlockHeight)
	seenTxs := make(map[string]coretypes.ResultTx)
	page := 1
	for {
		result, err := client.TxSearch(
			ctx,
			fmt.Sprintf(
				//"wasm-mailbox_dispatch_id._contract_address='%s'",
				"wasm-mailbox_dispatch_id._contract_address='%s' AND tx.height > %d AND tx.height < %d",
				c.mailBoxAddress,
				startBlockHeight,
				endBlockHeight,
			),
			false,
			&[]int{page}[0],
			&[]int{100}[0],
			"",
		)
		if err != nil {
			if strings.Contains(err.Error(), "page should be within") {
				break
			}
			lmt.Logger(ctx).Error("Error fetching txs", zap.Error(err))
			return nil, err
		}
		for _, tx := range result.Txs {
			seenTxs[tx.Hash.String()] = *tx
		}
		select {
		case <-ctx.Done():
			return nil, nil
		case <-time.After(10 * time.Millisecond):
		}
		page++
	}
	var messages []types.HyperlaneMessage
	for txHash, txResult := range seenTxs {
		for i, event := range txResult.TxResult.Events {
			if event.Type == "wasm-mailbox_dispatch_id" {
				hyperlaneMessage := types.HyperlaneMessage{
					TxHash:       txHash,
					SourceDomain: domain,
				}
				txStr := string(txResult.Tx)
				if strings.Contains(txStr, "timeout") {
					hyperlaneMessage.IsTimeout = true
				}
				if strings.Contains(txStr, "settlement") {
					hyperlaneMessage.IsSettlement = true
				}
				for _, attribute := range event.Attributes {
					if attribute.Key == "message_id" {
						hyperlaneMessage.MessageID = attribute.Value
					}
				}
				for _, attribute := range txResult.TxResult.Events[i+1].Attributes {
					if attribute.Key == "destination" {
						hyperlaneMessage.DestinationDomain = attribute.Value
					}
				}
				messages = append(messages, hyperlaneMessage)
			}
		}
	}
	return messages, nil
}

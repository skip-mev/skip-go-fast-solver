package cctp

import (
	"context"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/skip-mev/go-fast-solver/shared/contracts/fast_transfer_gateway"
	"github.com/skip-mev/go-fast-solver/shared/txexecutor/cosmos"
	"math/big"
	"strconv"
	"strings"
	"time"

	sdkgrpc "github.com/cosmos/cosmos-sdk/types/grpc"
	"google.golang.org/grpc/metadata"

	"google.golang.org/grpc"

	"github.com/skip-mev/go-fast-solver/db/gen/db"
	"github.com/skip-mev/go-fast-solver/ordersettler/types"

	"cosmossdk.io/math"
	wasmtypes "github.com/CosmWasm/wasmd/x/wasm/types"
	"github.com/avast/retry-go/v4"
	rpcclient "github.com/cometbft/cometbft/rpc/client"
	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	"github.com/cosmos/cosmos-sdk/std"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/bech32"
	authtx "github.com/cosmos/cosmos-sdk/x/auth/tx"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"

	"github.com/skip-mev/go-fast-solver/shared/config"
	"github.com/skip-mev/go-fast-solver/shared/signing"
)

type CosmosBridgeClient struct {
	rpcClient  rpcclient.Client
	grpcClient grpc.ClientConnInterface
	cdc        *codec.ProtoCodec
	txConfig   client.TxConfig

	chainID    string
	prefix     string
	signer     signing.Signer
	txExecutor cosmos.CosmosTxExecutor

	gasPrice float64
	gasDenom string
}

var _ BridgeClient = (*CosmosBridgeClient)(nil)

func NewCosmosBridgeClient(
	rpcClient rpcclient.Client,
	grpcClient grpc.ClientConnInterface,
	chainID string,
	prefix string,
	signer signing.Signer,
	gasPrice float64,
	gasDenom string,
	txSubmitter cosmos.CosmosTxExecutor,
) (*CosmosBridgeClient, error) {
	registry := codectypes.NewInterfaceRegistry()

	std.RegisterInterfaces(registry)
	authtypes.RegisterInterfaces(registry)
	wasmtypes.RegisterInterfaces(registry)

	cdc := codec.NewProtoCodec(registry)

	txConfig := authtx.NewTxConfig(cdc, authtx.DefaultSignModes)

	if signer == nil {
		signer = signing.NewNopSigner()
	}

	return &CosmosBridgeClient{
		rpcClient:  rpcClient,
		grpcClient: grpcClient,
		cdc:        cdc,
		txConfig:   txConfig,
		chainID:    chainID,
		prefix:     prefix,
		signer:     signer,
		gasPrice:   gasPrice,
		gasDenom:   gasDenom,
		txExecutor: txSubmitter,
	}, nil
}

func (c *CosmosBridgeClient) Balance(
	ctx context.Context,
	address string,
	denom string,
) (*big.Int, error) {
	requestBytes, err := c.cdc.Marshal(&banktypes.QueryBalanceRequest{
		Address: address,
		Denom:   denom,
	})
	if err != nil {
		return nil, err
	}

	abciResponse, err := c.rpcClient.ABCIQuery(
		ctx,
		"/cosmos.bank.v1beta1.Query/Balance",
		requestBytes,
	)
	if err != nil {
		return nil, err
	} else if abciResponse.Response.Code != 0 {
		return nil, abciError(
			abciResponse.Response.Codespace,
			abciResponse.Response.Code,
			abciResponse.Response.Log,
		)
	}

	response := banktypes.QueryBalanceResponse{}
	if err := c.cdc.Unmarshal(abciResponse.Response.Value, &response); err != nil {
		return nil, err
	}

	return response.Balance.Amount.BigInt(), nil
}

func (c *CosmosBridgeClient) SignerGasTokenBalance(ctx context.Context) (*big.Int, error) {
	return nil, errors.New("not implemented")
}

func (c *CosmosBridgeClient) Allowance(ctx context.Context, owner string) (*big.Int, error) {
	return nil, errors.New("allowance is not supported on Noble")
}

func (c *CosmosBridgeClient) IncreaseAllowance(ctx context.Context, amount *big.Int) (string, error) {
	return "", errors.New("allowance is not supported on Noble")
}

func (c *CosmosBridgeClient) RevokeAllowance(ctx context.Context) (string, error) {
	return "", errors.New("allowance is not supported on Noble")
}

func (c *CosmosBridgeClient) GetTxResult(ctx context.Context, txHash string) (*big.Int, *TxFailure, error) {
	txHashBytes, err := hex.DecodeString(txHash)
	if err != nil {
		return nil, nil, err
	}

	result, err := c.rpcClient.Tx(ctx, txHashBytes, false)
	if err != nil {
		return nil, nil, err
	} else if result.TxResult.Code != 0 {
		return big.NewInt(result.TxResult.GasUsed), &TxFailure{fmt.Sprintf("tx failed with code: %d and log: %s", result.TxResult.Code, result.TxResult.Log)}, nil
	}

	return big.NewInt(result.TxResult.GasUsed), nil, nil
}

func (c *CosmosBridgeClient) IsSettlementComplete(ctx context.Context, gatewayContractAddress, orderID string) (bool, error) {
	return false, errors.New("settlement complete event is not supported on Noble")
}

type FillOrderEnvelope struct {
	FillOrder *OrderEnvelope `json:"fill_order"`
}

type OrderEnvelope struct {
	Order  *FastTransferOrder `json:"order"`
	Filler string             `json:"filler"`
}

type FastTransferOrder struct {
	Sender            string `json:"sender"`
	Recipient         string `json:"recipient"`
	AmountIn          string `json:"amount_in"`
	AmountOut         string `json:"amount_out"`
	Nonce             uint32 `json:"nonce"`
	SourceDomain      uint32 `json:"source_domain"`
	DestinationDomain uint32 `json:"destination_domain"`
	TimeoutTimestamp  uint64 `json:"timeout_timestamp"`
	Data              string `json:"data,omitempty"`
}

func (c *CosmosBridgeClient) FillOrder(ctx context.Context, order db.Order, gatewayContractAddress string) (string, string, *uint64, error) {
	fromAddress, err := bech32.ConvertAndEncode(c.prefix, c.signer.Address())
	if err != nil {
		return "", "", nil, err
	}

	sourceChainConfig, err := config.GetConfigReader(ctx).GetChainConfig(order.SourceChainID)
	if err != nil {
		return "", "", nil, fmt.Errorf("getting config for source chainID %s: %w", order.SourceChainID, err)
	}
	sourceHyperlaneDomain, err := strconv.ParseUint(sourceChainConfig.HyperlaneDomain, 10, 64)
	if err != nil {
		return "", "", nil, fmt.Errorf("converting source hyperlane domain %s to uint: %w", sourceChainConfig.HyperlaneDomain, err)
	}

	destChainConfig, err := config.GetConfigReader(ctx).GetChainConfig(order.DestinationChainID)
	if err != nil {
		return "", "", nil, fmt.Errorf("getting config for destination chainID %s: %w", order.DestinationChainID, err)
	}
	destHyperlaneDomain, err := strconv.ParseUint(destChainConfig.HyperlaneDomain, 10, 64)
	if err != nil {
		return "", "", nil, fmt.Errorf("converting destination hyperlane domain %s to uint: %w", destChainConfig.HyperlaneDomain, err)
	}

	fillOrderMsg := &FillOrderEnvelope{
		FillOrder: &OrderEnvelope{
			Filler: fromAddress,
			Order: &FastTransferOrder{
				Sender:            hex.EncodeToString(order.Sender),
				Recipient:         hex.EncodeToString(order.Recipient),
				AmountIn:          order.AmountIn,
				AmountOut:         order.AmountOut,
				Nonce:             uint32(order.Nonce),
				SourceDomain:      uint32(sourceHyperlaneDomain),
				DestinationDomain: uint32(destHyperlaneDomain),
				TimeoutTimestamp:  uint64(order.TimeoutTimestamp.UTC().Unix()),
			},
		},
	}
	if order.Data.Valid {
		fillOrderMsg.FillOrder.Order.Data = order.Data.String
	}

	fillOrderMsgBytes, err := json.Marshal(fillOrderMsg)
	if err != nil {
		return "", "", nil, err
	}

	msgs := []sdk.Msg{}
	amount, ok := math.NewIntFromString(order.AmountOut)
	if !ok {
		return "", "", nil, errors.New("invalid amount")
	}

	wasmExecuteContractMsg := &wasmtypes.MsgExecuteContract{
		Sender:   fromAddress,
		Contract: gatewayContractAddress,
		Msg:      fillOrderMsgBytes,
		Funds: []sdk.Coin{{
			Denom:  "ibc/498A0751C798A0D9A389AA3691123DADA57DAA4FE165D5C75894505B876BA6E4",
			Amount: amount,
		}},
	}
	msgs = append(msgs, wasmExecuteContractMsg)
	txHash, tx, err := c.submitTx(ctx, msgs)
	if err != nil {
		return "", "", nil, err
	}
	txBytes, err := c.txConfig.TxJSONEncoder()(tx)
	if err != nil {
		return "", "", nil, err
	}
	return txHash, base64.StdEncoding.EncodeToString(txBytes), nil, err
}

type InitiateTimeoutEnvelope struct {
	InitiateTimeout *OrdersEnvelope `json:"initiate_timeout"`
}

type OrdersEnvelope struct {
	Orders []FastTransferOrder `json:"orders"`
}

func (c *CosmosBridgeClient) InitiateTimeout(ctx context.Context, order db.Order, gatewayContractAddress string) (string, string, *uint64, error) {
	fromAddress, err := bech32.ConvertAndEncode(c.prefix, c.signer.Address())
	if err != nil {
		return "", "", nil, err
	}

	sourceChainConfig, err := config.GetConfigReader(ctx).GetChainConfig(order.SourceChainID)
	if err != nil {
		return "", "", nil, fmt.Errorf("getting config for source chainID %s: %w", order.SourceChainID, err)
	}
	sourceHyperlaneDomain, err := strconv.ParseUint(sourceChainConfig.HyperlaneDomain, 10, 64)
	if err != nil {
		return "", "", nil, fmt.Errorf("converting source hyperlane domain %s to uint: %w", sourceChainConfig.HyperlaneDomain, err)
	}

	destChainConfig, err := config.GetConfigReader(ctx).GetChainConfig(order.DestinationChainID)
	if err != nil {
		return "", "", nil, fmt.Errorf("getting config for destination chainID %s: %w", order.DestinationChainID, err)
	}
	destHyperlaneDomain, err := strconv.ParseUint(destChainConfig.HyperlaneDomain, 10, 64)
	if err != nil {
		return "", "", nil, fmt.Errorf("converting destination hyperlane domain %s to uint: %w", destChainConfig.HyperlaneDomain, err)
	}

	initiateTimeoutMsg := &InitiateTimeoutEnvelope{
		InitiateTimeout: &OrdersEnvelope{
			Orders: []FastTransferOrder{
				{
					Sender:            hex.EncodeToString(order.Sender),
					Recipient:         hex.EncodeToString(order.Recipient),
					AmountIn:          order.AmountIn,
					AmountOut:         order.AmountOut,
					Nonce:             uint32(order.Nonce),
					SourceDomain:      uint32(sourceHyperlaneDomain),
					DestinationDomain: uint32(destHyperlaneDomain),
					TimeoutTimestamp:  uint64(order.TimeoutTimestamp.UTC().Unix()),
				},
			},
		},
	}
	if order.Data.Valid {
		initiateTimeoutMsg.InitiateTimeout.Orders[0].Data = order.Data.String
	}

	initiateTimeoutMsgBytes, err := json.Marshal(initiateTimeoutMsg)
	if err != nil {
		return "", "", nil, err
	}

	msgs := []sdk.Msg{}
	wasmExecuteContractMsg := &wasmtypes.MsgExecuteContract{
		Sender:   fromAddress,
		Contract: gatewayContractAddress,
		Msg:      initiateTimeoutMsgBytes,
	}
	msgs = append(msgs, wasmExecuteContractMsg)
	txHash, tx, err := c.submitTx(ctx, msgs)
	if err != nil {
		return "", "", nil, err
	}
	txBytes, err := c.txConfig.TxJSONEncoder()(tx)
	if err != nil {
		return "", "", nil, err
	}
	return txHash, base64.StdEncoding.EncodeToString(txBytes), nil, err
}

type InitiateSettlementEnvelope struct {
	InitiateSettlementMessage *InitiateSettlementMessage `json:"initiate_settlement"`
}

type InitiateSettlementMessage struct {
	OrderIDs         []string `json:"order_ids"`
	RepaymentAddress string   `json:"repayment_address"`
}

// InitiateBatchSettlement posts settlements on chain to a gateway contract address
// so that funds can be repayed. All settlements will be sent to the same
// repayment address and to the same gateway contract address. Thus, all
// settlements should have the same source and destination chain.
func (c *CosmosBridgeClient) InitiateBatchSettlement(ctx context.Context, batch types.SettlementBatch) (string, string, error) {
	if len(batch) == 0 {
		return "", "", nil
	}

	ids := batch.OrderIDs()
	repaymentAddress, err := batch.RepaymentAddress(ctx)
	if err != nil {
		return "", "", fmt.Errorf("getting batch repayment address: %w", err)
	}

	initiateSettlementMsg := &InitiateSettlementEnvelope{
		InitiateSettlementMessage: &InitiateSettlementMessage{
			OrderIDs:         ids,
			RepaymentAddress: hex.EncodeToString(repaymentAddress),
		},
	}
	initiateSettlementMsgBytes, err := json.Marshal(initiateSettlementMsg)
	if err != nil {
		return "", "", err
	}

	fromAddress, err := bech32.ConvertAndEncode(c.prefix, c.signer.Address())
	if err != nil {
		return "", "", err
	}
	gatewayContractAddress, err := batch.DestinationGatewayContractAddress(ctx)
	if err != nil {
		return "", "", fmt.Errorf("getting batch gateway contract address: %w", err)
	}

	msgs := []sdk.Msg{}
	wasmExecuteContractMsg := &wasmtypes.MsgExecuteContract{
		Sender:   fromAddress,
		Contract: gatewayContractAddress,
		Msg:      initiateSettlementMsgBytes,
	}

	msgs = append(msgs, wasmExecuteContractMsg)

	txHash, tx, err := c.submitTx(ctx, msgs)
	if err != nil {
		return "", "", fmt.Errorf("submitting tx: %w", err)
	}

	txBytes, err := c.txConfig.TxJSONEncoder()(tx)
	if err != nil {
		return "", "", fmt.Errorf("json encoding tx: %w", err)
	}

	return txHash, base64.StdEncoding.EncodeToString(txBytes), nil
}

func (c *CosmosBridgeClient) QueryOrderFillEvent(ctx context.Context, gatewayContractAddress, orderID string) (*string, *string, time.Time, error) {
	wasmQueryClient := wasmtypes.NewQueryClient(c.grpcClient)
	var header metadata.MD
	resp, err := wasmQueryClient.SmartContractState(ctx, &wasmtypes.QuerySmartContractStateRequest{
		Address:   gatewayContractAddress,
		QueryData: []byte(fmt.Sprintf(`{"order_fill":{"order_id":"%s"}}`, orderID)),
	}, grpc.Header(&header))
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			blockHeight := header.Get(sdkgrpc.GRPCBlockHeightHeader)
			blockHeightInt, err := strconv.ParseInt(blockHeight[0], 10, 64)
			if err != nil {
				return nil, nil, time.Time{}, fmt.Errorf("parsing block height: %w", err)
			}

			headerResp, err := c.rpcClient.Header(ctx, &blockHeightInt)
			if err != nil {
				return nil, nil, time.Time{}, fmt.Errorf("fetching block header at height %d: %w", blockHeightInt, err)
			}

			return nil, nil, headerResp.Header.Time, nil
		}
		return nil, nil, time.Time{}, fmt.Errorf("failed to query smart contract state: %w", err)
	}

	var fill struct {
		Filler  string `json:"filler"`
		OrderID string `json:"order_id"`
	}
	if err := json.Unmarshal(resp.Data, &fill); err != nil {
		return nil, nil, time.Time{}, fmt.Errorf("failed to unmarshal response: %w", err)
	}
	blockHeight := header.Get(sdkgrpc.GRPCBlockHeightHeader)
	blockHeightInt, err := strconv.ParseInt(blockHeight[0], 10, 64)
	if err != nil {
		return nil, nil, time.Time{}, fmt.Errorf("parsing block height: %w", err)
	}
	headerResp, err := c.rpcClient.Header(ctx, &blockHeightInt)
	if err != nil {
		return nil, nil, time.Time{}, fmt.Errorf("fetching block header at height %d: %w", blockHeightInt, err)
	}
	return &[]string{"txhash"}[0], &fill.Filler, headerResp.Header.Time, nil // TODO query for the actual txhash once the event is implemented
}

func (c *CosmosBridgeClient) IsOrderRefunded(ctx context.Context, gatewayContractAddress, orderID string) (bool, string, error) {
	return false, "", errors.New("not implemented")
}

type Fill struct {
	OrderID      string `json:"order_id"`
	SourceDomain uint32 `json:"source_domain"`
}

func (c *CosmosBridgeClient) OrderFillsByFiller(ctx context.Context, gatewayContractAddress, fillerAddress string) ([]Fill, error) {
	wasmQueryClient := wasmtypes.NewQueryClient(c.grpcClient)
	var startAfter *string
	const limit uint64 = 100

	var fills []Fill
	for {
		query := struct {
			OrderFillsByFiller struct {
				Filler     string  `json:"filler"`
				StartAfter *string `json:"start_after,omitempty"`
				Limit      uint64  `json:"limit"`
			} `json:"order_fills_by_filler"`
		}{
			OrderFillsByFiller: struct {
				Filler     string  `json:"filler"`
				StartAfter *string `json:"start_after,omitempty"`
				Limit      uint64  `json:"limit"`
			}{
				Filler:     fillerAddress,
				StartAfter: startAfter,
				Limit:      limit,
			},
		}
		jsonData, err := json.Marshal(query)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal query: %w", err)
		}

		resp, err := wasmQueryClient.SmartContractState(ctx, &wasmtypes.QuerySmartContractStateRequest{
			Address:   gatewayContractAddress,
			QueryData: jsonData,
		})
		if err != nil {
			return nil, fmt.Errorf("failed to query smart contract state: %w", err)
		}

		var page []Fill
		if err := json.Unmarshal(resp.Data, &page); err != nil {
			return nil, fmt.Errorf("failed to unmarshal response: %w", err)
		}

		fills = append(fills, page...)

		// If we received fewer results than the limit, we've reached the end
		if len(page) < int(limit) {
			break
		}

		// Set the startAfter for the next iteration
		startAfter = &page[len(page)-1].OrderID
	}
	return fills, nil
}

func (c *CosmosBridgeClient) WaitForTx(ctx context.Context, txHash string) error {
	return retry.Do(func() error {
		txHashBytes, err := hex.DecodeString(txHash)
		if err != nil {
			return err
		}

		result, err := c.rpcClient.Tx(ctx, txHashBytes, false)
		if err != nil {
			return err
		} else if result.TxResult.Code != 0 {
			return retry.Unrecoverable(abciError(
				result.TxResult.Codespace,
				result.TxResult.Code,
				result.TxResult.Log,
			))
		}

		return nil
	}, retry.Context(ctx), retry.Delay(1*time.Second), retry.MaxDelay(5*time.Second), retry.Attempts(20))
}

func (c *CosmosBridgeClient) OrderExists(ctx context.Context, gatewayContractAddress, orderID string, blockNumber *big.Int) (bool, *big.Int, error) {
	return false, nil, errors.New("not implemented")
}

func (c *CosmosBridgeClient) OrderStatus(ctx context.Context, gatewayContractAddress, orderID string) (uint8, error) {
	return 0, errors.New("not implemented")
}

func (c *CosmosBridgeClient) Close() {}

func (c *CosmosBridgeClient) submitTx(ctx context.Context, msgs []sdk.Msg) (string, sdk.Tx, error) {
	bech32Address, err := bech32.ConvertAndEncode(c.prefix, c.signer.Address())
	if err != nil {
		return "", nil, err
	}

	numRetries := 5
	for i := 0; i < numRetries; i++ {
		result, tx, err := c.txExecutor.ExecuteTx(ctx, c.chainID, bech32Address, msgs, c.txConfig, c.signer, c.gasPrice, c.gasDenom)

		if err != nil {
			return "", nil, err
		} else if result.Code != 0 {
			if strings.Contains(result.Log, "account sequence mismatch") {
				time.Sleep(1 * time.Second)
				continue
			}
			return "", nil, abciError(result.Codespace, result.Code, result.Log)
		}

		return strings.ToUpper(hex.EncodeToString(result.Hash)), tx, nil
	}
	return "", nil, errors.New("failed to submit tx")
}

func abciError(codespace string, code uint32, log string) error {
	return fmt.Errorf("%s error, code: %d, log: %s", codespace, code, log)
}

func (c *CosmosBridgeClient) BlockHeight(ctx context.Context) (uint64, error) {
	resp, err := c.rpcClient.Header(ctx, nil)
	if err != nil {
		return 0, err
	}
	return uint64(resp.Header.Height), nil
}

func (c *CosmosBridgeClient) QueryOrderSubmittedEvent(ctx context.Context, gatewayContractAddress, orderID string) (*fast_transfer_gateway.FastTransferOrder, error) {
	return nil, errors.New("not implemented")
}

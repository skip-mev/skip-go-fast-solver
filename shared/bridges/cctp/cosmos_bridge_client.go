package cctp

import (
	"context"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"strconv"
	"strings"
	"sync"
	"time"

	grpc2 "github.com/cosmos/cosmos-sdk/types/grpc"
	tx2 "github.com/cosmos/cosmos-sdk/types/tx"
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

const (
	simulationGasUsedMultiplier = 1.2
)

type CosmosBridgeClient struct {
	rpcClient  rpcclient.Client
	grpcClient grpc.ClientConnInterface
	cdc        *codec.ProtoCodec
	txConfig   client.TxConfig

	chainID           string
	prefix            string
	signer            signing.Signer
	txSubmissionMutex sync.Mutex

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
	Data              []byte `json:"data,omitempty"`
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
		data, err := hex.DecodeString(order.Data.String)
		if err != nil {
			return "", "", nil, fmt.Errorf("decoding hex order data to string: %w", err)
		}
		fillOrderMsg.FillOrder.Order.Data = data
	}

	fillOrderMsgBytes, err := json.Marshal(fillOrderMsg)
	if err != nil {
		return "", "", nil, err
	}

	txBuilder := c.txConfig.NewTxBuilder()
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
	err = txBuilder.SetMsgs(msgs...)
	if err != nil {
		return "", "", nil, err
	}
	txBytes, err := c.txConfig.TxJSONEncoder()(txBuilder.GetTx())
	if err != nil {
		return "", "", nil, err
	}
	txHash, err := c.submitTx(ctx, txBuilder.GetTx())
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
	initiateTimeoutMsgBytes, err := json.Marshal(initiateTimeoutMsg)
	if err != nil {
		return "", "", nil, err
	}

	txBuilder := c.txConfig.NewTxBuilder()
	msgs := []sdk.Msg{}
	wasmExecuteContractMsg := &wasmtypes.MsgExecuteContract{
		Sender:   fromAddress,
		Contract: gatewayContractAddress,
		Msg:      initiateTimeoutMsgBytes,
	}
	msgs = append(msgs, wasmExecuteContractMsg)
	err = txBuilder.SetMsgs(msgs...)
	if err != nil {
		return "", "", nil, err
	}
	txBytes, err := c.txConfig.TxJSONEncoder()(txBuilder.GetTx())
	if err != nil {
		return "", "", nil, err
	}
	txHash, err := c.submitTx(ctx, txBuilder.GetTx())
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

	txBuilder := c.txConfig.NewTxBuilder()
	msgs := []sdk.Msg{}
	wasmExecuteContractMsg := &wasmtypes.MsgExecuteContract{
		Sender:   fromAddress,
		Contract: gatewayContractAddress,
		Msg:      initiateSettlementMsgBytes,
	}

	msgs = append(msgs, wasmExecuteContractMsg)
	if err = txBuilder.SetMsgs(msgs...); err != nil {
		return "", "", fmt.Errorf("setting tx messages: %w", err)
	}

	txBytes, err := c.txConfig.TxJSONEncoder()(txBuilder.GetTx())
	if err != nil {
		return "", "", fmt.Errorf("json encoding tx: %w", err)
	}

	txHash, err := c.submitTx(ctx, txBuilder.GetTx())
	if err != nil {
		return "", "", fmt.Errorf("submitting tx: %w", err)
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
			blockHeight := header.Get(grpc2.GRPCBlockHeightHeader)
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
	blockHeight := header.Get(grpc2.GRPCBlockHeightHeader)
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

		var fills []Fill
		if err := json.Unmarshal(resp.Data, &fills); err != nil {
			return nil, fmt.Errorf("failed to unmarshal response: %w", err)
		}

		// If we received fewer results than the limit, we've reached the end
		if len(fills) < int(limit) {
			return fills, nil
		}

		// Set the startAfter for the next iteration
		startAfter = &fills[len(fills)-1].OrderID
	}
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

func (c *CosmosBridgeClient) Close() {}

func (c *CosmosBridgeClient) sign(ctx context.Context, tx sdk.Tx, accountNumber uint64, sequence uint64) (sdk.Tx, error) {
	signedTx, err := c.signer.Sign(ctx, c.chainID, signing.NewCosmosTransaction(tx, accountNumber, sequence, c.txConfig))
	if err != nil {
		return nil, err
	}
	return signedTx.(*signing.CosmosTransaction).Tx, nil
}

func (c *CosmosBridgeClient) submitTx(ctx context.Context, tx sdk.Tx) (string, error) {
	bech32Address, err := bech32.ConvertAndEncode(c.prefix, c.signer.Address())
	if err != nil {
		return "", err
	}
	c.txSubmissionMutex.Lock()
	defer c.txSubmissionMutex.Unlock()

	numRetries := 5
	for i := 0; i < numRetries; i++ {
		account, err := c.queryAccount(ctx, bech32Address)
		if err != nil {
			return "", err
		}
		gasEstimate, err := c.estimateGasUsed(ctx, tx, account)
		if err != nil {
			return "", err
		}
		txBuilder, err := c.txConfig.WrapTxBuilder(tx)
		if err != nil {
			return "", err
		}
		gasEstimateDec := math.LegacyNewDec(int64(gasEstimate))
		gasPriceDec, err := math.LegacyNewDecFromStr(strconv.FormatFloat(c.gasPrice, 'f', -1, 64))
		if err != nil {
			return "", err
		}
		txBuilder.SetFeeAmount(sdk.NewCoins(sdk.NewCoin(c.gasDenom, gasPriceDec.Mul(gasEstimateDec).Ceil().RoundInt())))
		txBuilder.SetGasLimit(gasEstimate)

		signedTx, err := c.sign(ctx, txBuilder.GetTx(), account.GetAccountNumber(), account.GetSequence())
		if err != nil {
			return "", err
		}

		signedTxBytes, err := c.txConfig.TxEncoder()(signedTx)
		if err != nil {
			return "", err
		}

		result, err := c.rpcClient.BroadcastTxSync(ctx, signedTxBytes)
		if err != nil {
			return "", err
		} else if result.Code != 0 {
			if strings.Contains(result.Log, "account sequence mismatch") {
				time.Sleep(1 * time.Second)
				continue
			}
			return "", abciError(result.Codespace, result.Code, result.Log)
		}

		return strings.ToUpper(hex.EncodeToString(result.Hash)), nil
	}
	return "", errors.New("failed to submit tx")
}

func (c *CosmosBridgeClient) estimateGasUsed(ctx context.Context, tx sdk.Tx, account sdk.AccountI) (uint64, error) {
	serviceClient := tx2.NewServiceClient(c.grpcClient)
	signedTxForSimulation, err := c.sign(ctx, tx, account.GetAccountNumber(), account.GetSequence())
	if err != nil {
		return 0, err
	}
	txBytes, err := c.txConfig.TxEncoder()(signedTxForSimulation)
	if err != nil {
		return 0, err
	}
	simulateResponse, err := serviceClient.Simulate(ctx, &tx2.SimulateRequest{
		TxBytes: txBytes,
	})
	if err != nil {
		return 0, err
	}
	return uint64(float64(simulateResponse.GasInfo.GasUsed) * simulationGasUsedMultiplier), nil
}

func (c *CosmosBridgeClient) queryAccount(ctx context.Context, address string) (sdk.AccountI, error) {
	requestBytes, err := c.cdc.Marshal(&authtypes.QueryAccountRequest{Address: address})
	if err != nil {
		return nil, fmt.Errorf("account query failed: %w", err)
	}

	abciResponse, err := c.rpcClient.ABCIQuery(ctx, "/cosmos.auth.v1beta1.Query/Account", requestBytes)
	if err != nil {
		return nil, fmt.Errorf("account query failed: %w", err)
	} else if abciResponse.Response.Code != 0 {
		return nil, abciError(abciResponse.Response.Codespace, abciResponse.Response.Code, abciResponse.Response.Log)
	}

	response := authtypes.QueryAccountResponse{}
	if err := c.cdc.Unmarshal(abciResponse.Response.Value, &response); err != nil {
		return nil, fmt.Errorf("account query failed: %w", err)
	}

	var account sdk.AccountI
	if err := c.cdc.UnpackAny(response.Account, &account); err != nil {
		return nil, fmt.Errorf("account query failed: %w", err)
	}

	return account, nil
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

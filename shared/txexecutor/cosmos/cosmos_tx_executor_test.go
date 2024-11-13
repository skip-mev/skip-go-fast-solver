package cosmos

import (
	wasmtypes "github.com/CosmWasm/wasmd/x/wasm/types"
	abcitypes "github.com/cometbft/cometbft/abci/types"
	coretypes "github.com/cometbft/cometbft/rpc/core/types"
	comettypes "github.com/cometbft/cometbft/types"
	client2 "github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	"github.com/cosmos/cosmos-sdk/std"
	"github.com/cosmos/cosmos-sdk/types"
	sdktx "github.com/cosmos/cosmos-sdk/types/tx"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	cometclient "github.com/skip-mev/go-fast-solver/mocks/github.com/cometbft/cometbft/rpc/client"
	"github.com/skip-mev/go-fast-solver/mocks/github.com/cosmos/cosmos-sdk/client"
	mockcosmostypes "github.com/skip-mev/go-fast-solver/mocks/github.com/cosmos/cosmos-sdk/types"
	mockcosmossigning "github.com/skip-mev/go-fast-solver/mocks/github.com/cosmos/cosmos-sdk/x/auth/signing"
	"github.com/skip-mev/go-fast-solver/mocks/github.com/cosmos/gogoproto/grpc"
	"github.com/skip-mev/go-fast-solver/mocks/shared/cosmosgrpc"
	mocksigning "github.com/skip-mev/go-fast-solver/mocks/shared/signing"
	"github.com/skip-mev/go-fast-solver/mocks/shared/tmrpc"
	"github.com/skip-mev/go-fast-solver/shared/signing"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"golang.org/x/net/context"
	"testing"
	"time"
)

func setupExecutor(t *testing.T, txSubmissionDelay time.Duration, chainID, signerAddress string) (CosmosTxExecutor, client2.TxConfig, signing.Signer) {
	rpcClientManager := tmrpc.NewMockTendermintRPCClientManager(t)
	rpcClient := cometclient.NewMockClient(t)
	rpcClientManager.On("GetClient", mock.Anything, chainID).Return(rpcClient, nil)

	grpcClientConnManager := cosmosgrpc.NewMockCosmosGRPCClientConnManager(t)
	grpcClientConn := grpc.NewMockClientConn(t)
	grpcClientConnManager.On("GetClient", mock.Anything, chainID).Return(grpcClientConn, nil)

	registry := codectypes.NewInterfaceRegistry()
	std.RegisterInterfaces(registry)
	authtypes.RegisterInterfaces(registry)
	wasmtypes.RegisterInterfaces(registry)
	cdc := codec.NewProtoCodec(registry)

	account := authtypes.BaseAccount{
		Address:       signerAddress,
		AccountNumber: 999,
		Sequence:      0,
	}
	accountAny, _ := codectypes.NewAnyWithValue(&account)
	queryAccountResponse := authtypes.QueryAccountResponse{Account: accountAny}
	responseBytes, _ := cdc.Marshal(&queryAccountResponse)
	rpcClient.On("ABCIQuery", mock.Anything, "/cosmos.auth.v1beta1.Query/Account", mock.Anything).Return(&coretypes.ResultABCIQuery{
		Response: abcitypes.ResponseQuery{
			Value: responseBytes,
		},
	}, nil)

	txConfig := client.NewMockTxConfig(t)
	tx := mockcosmostypes.NewMockTx(t)
	signer := mocksigning.NewMockSigner(t)
	cosmosTx := signing.NewCosmosTransaction(tx, 999, 0, txConfig)
	signer.On("Sign", mock.Anything, "chainID", mock.Anything).Return(cosmosTx, nil)
	txConfig.On("TxEncoder").Return(types.TxEncoder(func(tx types.Tx) ([]byte, error) { return []byte("0x1234"), nil }), nil)

	simulateRequest := &sdktx.SimulateRequest{
		TxBytes: []byte("0x1234"),
	}
	grpcClientConn.On("Invoke", mock.Anything, "/cosmos.tx.v1beta1.Service/Simulate", simulateRequest, mock.Anything).Run(func(args mock.Arguments) {
		reply := args.Get(3).(*sdktx.SimulateResponse)
		reply.GasInfo = &types.GasInfo{}
		reply.Result = &types.Result{}
	}).Return(nil)

	txBuilder := client.NewMockTxBuilder(t)
	txConfig.On("NewTxBuilder").Return(txBuilder, nil)
	txBuilder.On("SetMsgs", mock.Anything).Return(nil)
	txBuilder.On("SetFeeAmount", mock.Anything)
	txBuilder.On("SetGasLimit", mock.Anything)
	txBuilder.On("GetTx").Return(mockcosmossigning.NewMockTx(t))

	cosmosTxBytes, _ := cosmosTx.MarshalBinary()
	rpcClient.On("BroadcastTxSync", mock.Anything, comettypes.Tx(cosmosTxBytes)).Return(&coretypes.ResultBroadcastTx{}, nil)

	executor := NewSerializedCosmosTxExecutor(rpcClientManager, grpcClientConnManager, txSubmissionDelay, cdc)
	return executor, txConfig, signer
}

func TestSerializedCosmosTxExecutor_ExecuteTx_NoDelay(t *testing.T) {
	executor, txConfig, signer := setupExecutor(t, 2*time.Second, "chainID", "signerAddress")

	// call ExecuteTx and ensure that it returns immediately since it is the first invocation
	start := time.Now()
	response, _, err := executor.ExecuteTx(
		context.Background(),
		"chainID",
		"signerAddress",
		nil,
		txConfig,
		signer,
		1.0,
		"gasDenom",
	)
	require.Nil(t, err)
	require.NotNil(t, response)
	require.WithinDuration(t, time.Now(), start, 100*time.Second)

}

func TestSerializedCosmosTxExecutor_ExecuteTx_WithDelay(t *testing.T) {
	executor, txConfig, signer := setupExecutor(t, 2*time.Second, "chainID", "signerAddress")

	// call ExecuteTx to start delay timer
	response, _, err := executor.ExecuteTx(
		context.Background(),
		"chainID",
		"signerAddress",
		nil,
		txConfig,
		signer,
		1.0,
		"gasDenom",
	)
	require.Nil(t, err)
	require.NotNil(t, response)

	// call ExecuteTx again and ensure that it returns after the configured delay
	start := time.Now()
	response, _, err = executor.ExecuteTx(
		context.Background(),
		"chainID",
		"signerAddress",
		nil,
		txConfig,
		signer,
		1.0,
		"gasDenom",
	)
	require.Nil(t, err)
	require.NotNil(t, response)
	require.GreaterOrEqual(t, time.Since(start), 2*time.Second)
}

func TestSerializedCosmosTxExecutor_ExecuteTx_DelayCancelled(t *testing.T) {
	executor, txConfig, signer := setupExecutor(t, 10*time.Second, "chainID", "signerAddress")

	// call ExecuteTx to start delay timer
	response, _, err := executor.ExecuteTx(
		context.Background(),
		"chainID",
		"signerAddress",
		nil,
		txConfig,
		signer,
		1.0,
		"gasDenom",
	)
	require.Nil(t, err)
	require.NotNil(t, response)

	// call ExecuteTx with a cancelled context and ensure it returns immediately
	start := time.Now()
	cancelCtx, cancelFn := context.WithCancel(context.Background())
	cancelFn()
	_, _, err = executor.ExecuteTx(
		cancelCtx,
		"chainID",
		"signerAddress",
		nil,
		txConfig,
		signer,
		1.0,
		"gasDenom",
	)
	require.NotNil(t, err)
	require.WithinDuration(t, start, time.Now(), 100*time.Millisecond)
}

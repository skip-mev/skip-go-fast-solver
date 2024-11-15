package cosmos

import (
	"cosmossdk.io/math"
	"fmt"
	wasmtypes "github.com/CosmWasm/wasmd/x/wasm/types"
	"github.com/cometbft/cometbft/rpc/client"
	coretypes "github.com/cometbft/cometbft/rpc/core/types"
	sdkclient "github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	"github.com/cosmos/cosmos-sdk/std"
	"github.com/cosmos/cosmos-sdk/types"
	sdktypes "github.com/cosmos/cosmos-sdk/types/tx"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	"github.com/skip-mev/go-fast-solver/shared/cosmosgrpc"
	"github.com/skip-mev/go-fast-solver/shared/signing"
	"github.com/skip-mev/go-fast-solver/shared/tmrpc"
	"golang.org/x/net/context"
	"strconv"
	"sync"
	"time"
)

const (
	simulationGasUsedMultiplier = 1.2
)

type CosmosTxExecutor interface {
	ExecuteTx(
		ctx context.Context,
		chainID string,
		signerAddress string,
		msgs []types.Msg,
		txConfig sdkclient.TxConfig,
		signer signing.Signer,
		gasPrice float64,
		gasDenom string,
	) (*coretypes.ResultBroadcastTx, types.Tx, error)
}

type SerializedCosmosTxExecutor struct {
	rpcClientManager      tmrpc.TendermintRPCClientManager
	grpcClientConnManager cosmosgrpc.CosmosGRPCClientConnManager
	lock                  sync.Mutex
	lastSubmissionTime    time.Time
	txSubmissionDelay     time.Duration
	cdc                   *codec.ProtoCodec
}

func DefaultSerializedCosmosTxExecutor() CosmosTxExecutor {
	registry := codectypes.NewInterfaceRegistry()

	std.RegisterInterfaces(registry)
	authtypes.RegisterInterfaces(registry)
	wasmtypes.RegisterInterfaces(registry)

	cdc := codec.NewProtoCodec(registry)
	return NewSerializedCosmosTxExecutor(
		tmrpc.NewTendermintRPCClientManager(),
		cosmosgrpc.NewCosmosGRPCClientConnManager(),
		500*time.Millisecond,
		cdc,
	)
}

func NewSerializedCosmosTxExecutor(
	rpcClientManager tmrpc.TendermintRPCClientManager,
	grpcClientManager cosmosgrpc.CosmosGRPCClientConnManager,
	txSubmissionDelay time.Duration,
	cdc *codec.ProtoCodec,
) CosmosTxExecutor {
	return &SerializedCosmosTxExecutor{
		rpcClientManager:      rpcClientManager,
		grpcClientConnManager: grpcClientManager,
		txSubmissionDelay:     txSubmissionDelay,
		cdc:                   cdc,
	}
}

func (s *SerializedCosmosTxExecutor) ExecuteTx(
	ctx context.Context,
	chainID string,
	signerAddress string,
	msgs []types.Msg,
	txConfig sdkclient.TxConfig,
	signer signing.Signer,
	gasPrice float64,
	gasDenom string,
) (*coretypes.ResultBroadcastTx, types.Tx, error) {
	client, err := s.rpcClientManager.GetClient(ctx, chainID)
	if err != nil {
		return nil, nil, err
	}
	s.lock.Lock()
	defer func() {
		if err == nil {
			s.lastSubmissionTime = time.Now()
		}
		s.lock.Unlock()
	}()
	select {
	case <-time.After(time.Until(s.lastSubmissionTime.Add(s.txSubmissionDelay))):
	case <-ctx.Done():
		return nil, nil, ctx.Err()
	}

	txBuilder := txConfig.NewTxBuilder()
	err = txBuilder.SetMsgs(msgs...)
	if err != nil {
		return nil, nil, err
	}

	tx := txBuilder.GetTx()

	account, err := s.queryAccount(ctx, client, signerAddress)
	if err != nil {
		return nil, nil, err
	}
	gasEstimate, err := s.estimateGasUsed(ctx, chainID, tx, account, txConfig, signer)
	if err != nil {
		return nil, nil, err
	}
	gasEstimateDec := math.LegacyNewDec(int64(gasEstimate))
	gasPriceDec, err := math.LegacyNewDecFromStr(strconv.FormatFloat(gasPrice, 'f', -1, 64))
	if err != nil {
		return nil, nil, err
	}
	txBuilder.SetFeeAmount(types.NewCoins(types.NewCoin(gasDenom, gasPriceDec.Mul(gasEstimateDec).Ceil().RoundInt())))
	txBuilder.SetGasLimit(gasEstimate)
	signedTx, err := signer.Sign(ctx, chainID, signing.NewCosmosTransaction(txBuilder.GetTx(), account.GetAccountNumber(), account.GetSequence(), txConfig))
	if err != nil {
		return nil, nil, err
	}

	signedTxBytes, err := signedTx.MarshalBinary()
	if err != nil {
		return nil, nil, err
	}

	res, err := client.BroadcastTxSync(ctx, signedTxBytes)
	return res, txBuilder.GetTx(), err
}

func (s *SerializedCosmosTxExecutor) queryAccount(ctx context.Context, client client.Client, address string) (types.AccountI, error) {
	requestBytes, err := s.cdc.Marshal(&authtypes.QueryAccountRequest{Address: address})
	if err != nil {
		return nil, fmt.Errorf("account query failed: %w", err)
	}

	abciResponse, err := client.ABCIQuery(ctx, "/cosmos.auth.v1beta1.Query/Account", requestBytes)
	if err != nil {
		return nil, fmt.Errorf("account query failed: %w", err)
	} else if abciResponse.Response.Code != 0 {
		return nil, fmt.Errorf("%s error, code: %d, log: %s", abciResponse.Response.Codespace, abciResponse.Response.Code, abciResponse.Response.Log)
	}

	response := authtypes.QueryAccountResponse{}
	if err := s.cdc.Unmarshal(abciResponse.Response.Value, &response); err != nil {
		return nil, fmt.Errorf("account query failed: %w", err)
	}

	var account types.AccountI
	if err := s.cdc.UnpackAny(response.Account, &account); err != nil {
		return nil, fmt.Errorf("account query failed: %w", err)
	}

	return account, nil
}

func (s *SerializedCosmosTxExecutor) estimateGasUsed(
	ctx context.Context,
	chainID string,
	tx types.Tx,
	account types.AccountI,
	txConfig sdkclient.TxConfig,
	signer signing.Signer,
) (uint64, error) {
	clientConn, err := s.grpcClientConnManager.GetClient(ctx, chainID)
	if err != nil {
		return 0, err
	}
	serviceClient := sdktypes.NewServiceClient(clientConn)
	signedTxForSimulation, err := signer.Sign(ctx, chainID, signing.NewCosmosTransaction(tx, account.GetAccountNumber(), account.GetSequence(), txConfig))
	if err != nil {
		return 0, err
	}
	txBytes, err := txConfig.TxEncoder()(signedTxForSimulation.(*signing.CosmosTransaction).Tx)
	if err != nil {
		return 0, err
	}
	simulateResponse, err := serviceClient.Simulate(ctx, &sdktypes.SimulateRequest{
		TxBytes: txBytes,
	})
	if err != nil {
		return 0, err
	}
	return uint64(float64(simulateResponse.GasInfo.GasUsed) * simulationGasUsedMultiplier), nil
}

package evm

import (
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/skip-mev/go-fast-solver/mocks/shared/evmrpc"
	mocksigning "github.com/skip-mev/go-fast-solver/mocks/shared/signing"
	"github.com/skip-mev/go-fast-solver/shared/signing"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"golang.org/x/net/context"
	"testing"
	"time"
)

func setupExecutor(t *testing.T, txSubmissionDelay time.Duration, chainID string) (EVMTxExecutor, signing.Signer) {
	rpcClientManager := evmrpc.NewMockEVMRPCClientManager(t)
	rpcClient := evmrpc.NewMockEVMChainRPC(t)
	rpcClientManager.On("GetClient", mock.Anything, chainID).Return(rpcClient, nil)

	signer := mocksigning.NewMockSigner(t)

	rpcClient.On("PendingNonceAt", mock.Anything, mock.Anything).Return(uint64(9), nil)

	tx := types.NewTx(&types.AccessListTx{Nonce: 9})
	signer.On("Sign", mock.Anything, mock.Anything, mock.Anything).Return(tx, nil)

	txBytes, err := tx.MarshalBinary()
	require.NoError(t, err)

	rpcClient.On("SendTx", mock.Anything, txBytes).Return("txHash", nil)

	executor := NewSerializedEVMTxExecutor(rpcClientManager, txSubmissionDelay)
	return executor, signer
}

func TestSerializedEVMTxExecutor_ExecuteTx_NoDelay(t *testing.T) {
	executor, signer := setupExecutor(t, 2*time.Second, "chainID")

	// call ExecuteTx and ensure that it returns immediately since it is the first invocation
	start := time.Now()
	response, err := executor.ExecuteTx(
		context.Background(),
		"chainID",
		"signerAddress",
		nil,
		"value",
		"to",
		signer,
	)
	require.Nil(t, err)
	require.NotNil(t, response)
	require.WithinDuration(t, time.Now(), start, 100*time.Second)
}

func TestSerializedEVMTxExecutor_ExecuteTx_WithDelay(t *testing.T) {
	executor, signer := setupExecutor(t, 2*time.Second, "chainID")

	// call ExecuteTx to start delay timer
	response, err := executor.ExecuteTx(
		context.Background(),
		"chainID",
		"signerAddress",
		nil,
		"value",
		"to",
		signer,
	)
	require.Nil(t, err)
	require.NotNil(t, response)

	// call ExecuteTx again and ensure that it returns after the configured delay\
	start := time.Now()
	response, err = executor.ExecuteTx(
		context.Background(),
		"chainID",
		"signerAddress",
		nil,
		"value",
		"to",
		signer,
	)
	require.Nil(t, err)
	require.NotNil(t, response)
	require.GreaterOrEqual(t, time.Since(start), 2*time.Second)
}

func TestSerializedEVMTxExecutor_ExecuteTx_DelayCancelled(t *testing.T) {
	executor, signer := setupExecutor(t, 10*time.Second, "chainID")

	// call ExecuteTx to start delay timer
	response, err := executor.ExecuteTx(
		context.Background(),
		"chainID",
		"signerAddress",
		nil,
		"value",
		"to",
		signer,
	)
	require.Nil(t, err)
	require.NotNil(t, response)

	// call ExecuteTx with a cancelled context and ensure it returns immediately
	start := time.Now()
	cancelCtx, cancelFn := context.WithCancel(context.Background())
	cancelFn()
	_, err = executor.ExecuteTx(
		cancelCtx,
		"chainID",
		"signerAddress",
		nil,
		"value",
		"to",
		signer,
	)
	require.NotNil(t, err)
	require.WithinDuration(t, start, time.Now(), 100*time.Millisecond)
}

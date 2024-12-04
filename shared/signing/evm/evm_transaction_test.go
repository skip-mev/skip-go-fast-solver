package evm_test

import (
	"context"
	"testing"

	"github.com/ethereum/go-ethereum/core/types"
	"github.com/skip-mev/go-fast-solver/mocks/shared/evmrpc"
	"github.com/skip-mev/go-fast-solver/shared/signing/evm"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestWithEstimatedGasLimit(t *testing.T) {
	t.Run("gas limit is multiplied by 1.2 from what the node returns", func(t *testing.T) {
		mockRPC := evmrpc.NewMockEVMChainRPC(t)
		mockRPC.EXPECT().EstimateGas(mock.Anything, mock.Anything).Return(100_000, nil)

		builder := evm.NewTxBuilder(mockRPC)
		opt := evm.WithEstimatedGasLimit("0xfrom", "0xto", "0", []byte("0xdeadbeef"))

		tx := &types.DynamicFeeTx{}
		err := opt(context.Background(), builder, tx)
		assert.NoError(t, err)

		gasLimit := types.NewTx(tx).Gas()
		assert.Equal(t, uint64(120_000), gasLimit)
	})
}

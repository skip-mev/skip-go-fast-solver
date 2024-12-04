package fundrebalancer

import (
	"math/big"
	"strconv"
	"testing"
	"time"

	dbtypes "github.com/skip-mev/go-fast-solver/db"
	"github.com/skip-mev/go-fast-solver/db/gen/db"
	mock_database "github.com/skip-mev/go-fast-solver/mocks/fundrebalancer"
	mock_skipgo "github.com/skip-mev/go-fast-solver/mocks/shared/clients/skipgo"
	mock_config "github.com/skip-mev/go-fast-solver/mocks/shared/config"
	mock_evmrpc "github.com/skip-mev/go-fast-solver/mocks/shared/evmrpc"
	evm2 "github.com/skip-mev/go-fast-solver/mocks/shared/txexecutor/evm"
	"github.com/skip-mev/go-fast-solver/shared/clients/skipgo"
	"github.com/skip-mev/go-fast-solver/shared/config"
	"github.com/skip-mev/go-fast-solver/shared/keys"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"golang.org/x/net/context"
)

func TestTransferMonitor_UpdateTransfers(t *testing.T) {
	t.Run("pending transaction status is properly updated", func(t *testing.T) {
		ctx := context.Background()
		mockSkipGo := mock_skipgo.NewMockSkipGoClient(t)
		mockDatabase := mock_database.NewMockDatabase(t)
		mockConfigReader := mock_config.NewMockConfigReader(t)

		ctx = config.ConfigReaderContext(ctx, mockConfigReader)

		// two osmosis pending tx's, one will fail and another will complete successfully
		mockDatabase.EXPECT().GetAllPendingRebalanceTransfers(ctx).Return([]db.GetAllPendingRebalanceTransfersRow{
			{ID: 1, TxHash: "hash", SourceChainID: arbitrumChainID, DestinationChainID: osmosisChainID, Amount: strconv.Itoa(osmosisTargetAmount), CreatedAt: time.Now()},
			{ID: 2, TxHash: "hash2", SourceChainID: arbitrumChainID, DestinationChainID: osmosisChainID, Amount: strconv.Itoa(osmosisTargetAmount), CreatedAt: time.Now()},
		}, nil)

		mockSkipGo.EXPECT().TrackTx(ctx, "hash", arbitrumChainID).Return("hash", nil)
		mockSkipGo.EXPECT().TrackTx(ctx, "hash2", arbitrumChainID).Return("hash2", nil)

		mockSkipGo.EXPECT().Status(ctx, skipgo.TxHash("hash"), arbitrumChainID).Return(&skipgo.StatusResponse{
			Transfers: []skipgo.Transfer{
				{State: skipgo.STATE_COMPLETED_SUCCESS, Error: nil},
				{State: skipgo.STATE_COMPLETED_SUCCESS, Error: nil},
				{State: skipgo.STATE_COMPLETED_SUCCESS, Error: nil},
			},
		}, nil)

		transferError := "ahhhh"
		mockSkipGo.EXPECT().Status(ctx, skipgo.TxHash("hash2"), arbitrumChainID).Return(&skipgo.StatusResponse{
			Transfers: []skipgo.Transfer{
				{State: skipgo.STATE_COMPLETED_SUCCESS, Error: nil},
				{State: skipgo.STATE_COMPLETED_SUCCESS, Error: nil},
				{State: skipgo.STATE_COMPLETED_ERROR, Error: &transferError},
			},
		}, nil)

		mockDatabase.EXPECT().UpdateTransferStatus(ctx, db.UpdateTransferStatusParams{
			ID:     1,
			Status: dbtypes.RebalanceTransferStatusSuccess,
		}).Return(nil)
		mockDatabase.EXPECT().UpdateTransferStatus(ctx, db.UpdateTransferStatusParams{
			ID:     2,
			Status: dbtypes.RebalanceTransferStatusFailed,
		}).Return(nil)

		tm := NewTransferTracker(mockSkipGo, mockDatabase)

		assert.NoError(t, tm.UpdateTransfers(ctx))
	})

	t.Run("errored transaction but no error string", func(t *testing.T) {
		ctx := context.Background()
		mockSkipGo := mock_skipgo.NewMockSkipGoClient(t)
		mockDatabse := mock_database.NewMockDatabase(t)

		// two osmosis pending tx's, one will fail and another will complete successfully
		mockDatabse.EXPECT().GetAllPendingRebalanceTransfers(mockContext).Return([]db.GetAllPendingRebalanceTransfersRow{
			{ID: 1, TxHash: "hash", SourceChainID: arbitrumChainID, DestinationChainID: osmosisChainID, Amount: strconv.Itoa(osmosisTargetAmount), CreatedAt: time.Now()},
			{ID: 2, TxHash: "hash2", SourceChainID: arbitrumChainID, DestinationChainID: osmosisChainID, Amount: strconv.Itoa(osmosisTargetAmount), CreatedAt: time.Now()},
		}, nil)

		mockSkipGo.EXPECT().TrackTx(mockContext, "hash", arbitrumChainID).Return("hash", nil)

		mockSkipGo.EXPECT().TrackTx(mockContext, "hash2", arbitrumChainID).Return("hash2", nil)

		mockSkipGo.EXPECT().Status(mockContext, skipgo.TxHash("hash"), arbitrumChainID).Return(&skipgo.StatusResponse{
			Transfers: []skipgo.Transfer{
				{State: skipgo.STATE_COMPLETED_SUCCESS, Error: nil},
				{State: skipgo.STATE_COMPLETED_SUCCESS, Error: nil},
				{State: skipgo.STATE_COMPLETED_SUCCESS, Error: nil},
			},
		}, nil)

		mockSkipGo.EXPECT().Status(mockContext, skipgo.TxHash("hash2"), arbitrumChainID).Return(&skipgo.StatusResponse{
			Transfers: []skipgo.Transfer{
				{State: skipgo.STATE_COMPLETED_SUCCESS, Error: nil},
				{State: skipgo.STATE_COMPLETED_SUCCESS, Error: nil},
				{State: skipgo.STATE_COMPLETED_ERROR, Error: nil},
			},
		}, nil)

		mockDatabse.EXPECT().UpdateTransferStatus(mockContext, db.UpdateTransferStatusParams{
			ID:     1,
			Status: dbtypes.RebalanceTransferStatusSuccess,
		}).Return(nil)
		mockDatabse.EXPECT().UpdateTransferStatus(mockContext, db.UpdateTransferStatusParams{
			ID:     2,
			Status: dbtypes.RebalanceTransferStatusFailed,
		}).Return(nil)

		tm := NewTransferTracker(mockSkipGo, mockDatabse)

		assert.NoError(t, tm.UpdateTransfers(ctx))
	})
}

func TestFundRebalancer_RebalanceWithAbandonedTransfer(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	mockConfigReader := mock_config.NewMockConfigReader(t)
	mockConfigReader.On("Config").Return(config.Config{
		FundRebalancer: map[string]config.FundRebalancerConfig{
			osmosisChainID: {
				TargetAmount:     strconv.Itoa(osmosisTargetAmount),
				MinAllowedAmount: strconv.Itoa(osmosisMinAmount),
			},
			arbitrumChainID: {
				TargetAmount:     strconv.Itoa(arbitrumTargetAmount),
				MinAllowedAmount: strconv.Itoa(arbitrumMinAmount),
			},
		},
	})

	mockConfigReader.On("GetFundRebalancingConfig", arbitrumChainID).Return(
		config.FundRebalancerConfig{
			TargetAmount:               strconv.Itoa(arbitrumTargetAmount),
			MinAllowedAmount:           strconv.Itoa(arbitrumMinAmount),
			MaxRebalancingGasCostUUSDC: "50000000",
			ProfitabilityTimeout:       disabledTimeout,
			TransferCostCapUUSDC:       "10000000",
		},
		nil,
	)

	mockConfigReader.EXPECT().GetUSDCDenom(osmosisChainID).Return(osmosisUSDCDenom, nil)
	mockConfigReader.EXPECT().GetUSDCDenom(arbitrumChainID).Return(arbitrumUSDCDenom, nil)
	mockConfigReader.On("GetChainConfig", osmosisChainID).Return(
		config.ChainConfig{
			Type:          config.ChainType_COSMOS,
			USDCDenom:     osmosisUSDCDenom,
			SolverAddress: osmosisAddress,
		},
		nil,
	)
	mockConfigReader.On("GetChainConfig", arbitrumChainID).Return(
		config.ChainConfig{
			Type:          config.ChainType_EVM,
			USDCDenom:     arbitrumUSDCDenom,
			SolverAddress: arbitrumAddress,
		},
		nil,
	)
	ctx = config.ConfigReaderContext(ctx, mockConfigReader)

	f, err := loadKeysFile(defaultKeys)
	assert.NoError(t, err)

	mockSkipGo := mock_skipgo.NewMockSkipGoClient(t)
	mockEVMClientManager := mock_evmrpc.NewMockEVMRPCClientManager(t)
	mockEVMClient := mock_evmrpc.NewMockEVMChainRPC(t)
	mockEVMClient.EXPECT().SuggestGasPrice(mockContext).Return(big.NewInt(10000), nil)
	mockEVMClientManager.EXPECT().GetClient(ctx, arbitrumChainID).Return(mockEVMClient, nil)
	fakeDatabase := mock_database.NewFakeDatabase()
	mockEVMTxExecutor := evm2.NewMockEVMTxExecutor(t)
	mockTxPriceOracle := mock_evmrpc.NewMockOracle(t)
	mockTxPriceOracle.On("TxFeeUUSDC", ctx, mock.Anything, mock.Anything).Return(big.NewInt(75), nil)
	keystore, err := keys.LoadKeyStoreFromPlaintextFile(f.Name())
	assert.NoError(t, err)

	rebalancer, err := NewFundRebalancer(ctx, keystore, mockSkipGo, mockEVMClientManager, fakeDatabase, mockTxPriceOracle, mockEVMTxExecutor)
	assert.NoError(t, err)

	// Insert an old pending transfer that should be abandoned
	oldTransferID, err := fakeDatabase.InsertRebalanceTransfer(ctx, db.InsertRebalanceTransferParams{
		TxHash:             "old_hash",
		SourceChainID:      arbitrumChainID,
		DestinationChainID: osmosisChainID,
		Amount:             "50",
	})
	assert.NoError(t, err)
	err = fakeDatabase.UpdateTransferCreatedAt(ctx, oldTransferID, time.Now().Add(-2*transferTimeout))
	assert.NoError(t, err)

	mockSkipGo.EXPECT().Balance(ctx, &skipgo.BalancesRequest{
		Chains: map[string]skipgo.ChainRequest{
			osmosisChainID: {
				Address: osmosisAddress,
				Denoms:  []string{osmosisUSDCDenom},
			},
		},
	}).Return(&skipgo.BalancesResponse{
		Chains: map[string]skipgo.ChainResponse{
			osmosisChainID: {
				Address: osmosisAddress,
				Denoms: map[string]skipgo.DenomDetail{
					osmosisUSDCDenom: {
						Amount:          "0",
						Decimals:        6,
						FormattedAmount: "0",
						Price:           "1.0",
						ValueUSD:        "0",
					},
				},
			},
		},
	}, nil)

	mockEVMClient.EXPECT().GetUSDCBalance(ctx, arbitrumUSDCDenom, arbitrumAddress).Return(big.NewInt(1000), nil)

	route := &skipgo.RouteResponse{
		AmountOut:              strconv.Itoa(osmosisTargetAmount),
		Operations:             []any{"opts"},
		RequiredChainAddresses: []string{arbitrumChainID, osmosisChainID},
	}
	mockSkipGo.EXPECT().Route(ctx, arbitrumUSDCDenom, arbitrumChainID, osmosisUSDCDenom, osmosisChainID, big.NewInt(osmosisTargetAmount)).
		Return(route, nil)

	txs := []skipgo.Tx{{EVMTx: &skipgo.EVMTx{ChainID: arbitrumChainID, To: osmosisAddress, Value: "999", SignerAddress: arbitrumAddress}}}
	mockSkipGo.EXPECT().Msgs(ctx, arbitrumUSDCDenom, arbitrumChainID, arbitrumAddress, osmosisUSDCDenom, osmosisChainID, osmosisAddress, big.NewInt(osmosisTargetAmount), big.NewInt(osmosisTargetAmount), []string{arbitrumAddress, osmosisAddress}, route.Operations).
		Return(txs, nil)

	mockEVMClient.On("EstimateGas", mock.Anything, mock.Anything).Return(uint64(100), nil)
	mockEVMTxExecutor.On("ExecuteTx", ctx, arbitrumChainID, arbitrumAddress, []byte{}, "999", osmosisAddress, mock.Anything).Return("new_hash", nil)

	// Rebalancer sees the pending transfer and doesn't create a new one
	rebalancer.Rebalance(ctx)
	assert.NoError(t, err)
	transfers := fakeDatabase.GetDBContents()
	assert.Equal(t, len(transfers), 1)

	// Update transfers to handle the abandoned transfer
	tm := NewTransferTracker(mockSkipGo, fakeDatabase)
	assert.NoError(t, tm.UpdateTransfers(ctx))

	// Call rebalance again after the old transfer is abandoned to create the new transfer
	rebalancer.Rebalance(ctx)
	assert.NoError(t, err)

	transfers = fakeDatabase.GetDBContents()
	var foundOldTransfer bool
	var foundNewTransfer bool
	for _, transfer := range transfers {
		if transfer.ID == oldTransferID {
			assert.Equal(t, dbtypes.RebalanceTransferStatusAbandoned, transfer.Status)
			foundOldTransfer = true
		} else {
			assert.Equal(t, "new_hash", transfer.TxHash)
			assert.Equal(t, "PENDING", transfer.Status)
			foundNewTransfer = true
		}
	}

	assert.True(t, foundOldTransfer, "old transfer should be found and marked as abandoned")
	assert.True(t, foundNewTransfer, "new transfer should be created")
}

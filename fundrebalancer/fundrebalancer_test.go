package fundrebalancer_test

import (
	"context"
	"encoding/json"
	"fmt"
	"math/big"
	"os"
	"strconv"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/skip-mev/go-fast-solver/db/gen/db"
	"github.com/skip-mev/go-fast-solver/fundrebalancer"
	mock_database "github.com/skip-mev/go-fast-solver/mocks/fundrebalancer"
	mock_skipgo "github.com/skip-mev/go-fast-solver/mocks/shared/clients/skipgo"
	mock_config "github.com/skip-mev/go-fast-solver/mocks/shared/config"
	mock_evmrpc "github.com/skip-mev/go-fast-solver/mocks/shared/evmrpc"
	"github.com/skip-mev/go-fast-solver/shared/clients/skipgo"
	"github.com/skip-mev/go-fast-solver/shared/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

const (
	osmosisAddress  = "osmo1abc"
	arbitrumAddress = "0x123"
	ethAddress      = "0x456"

	osmosisPrivateKey  = "0x75adb34f739a13dcdeb12c6cf3ad9d56f278dc5744933a998784b445a1d23c55"
	arbitrumPrivateKey = "0x75adb34f739a13dcdeb12c6cf3ad9d56f278dc5744933a998784b445a1d23c55"
	ethPrivateKey      = "0x75adb34f739a13dcdeb12c6cf3ad9d56f278dc5744933a998784b445a1d23c55"

	osmosisChainID  = "osmosis-1"
	arbitrumChainID = "42161"
	ethChainID      = "1"

	osmosisTargetAmount  = 100
	osmosisMinAmount     = 50
	arbitrumTargetAmount = 50
	arbitrumMinAmount    = 0
	ethTargetAmount      = 50
	ethMinAmount         = 25

	osmosisUSDCDenom  = "ibc/123"
	arbitrumUSDCDenom = "0xusdc"
	ethUSDCDenom      = "0x123usdc"

	nobleChainID    = "noble-1"
	nobleAddress    = "noble1abc"
	noblePrivateKey = "noble123"
)

var (
	mockContext = mock.Anything

	defaultKeys = map[string]interface{}{
		arbitrumChainID: map[string]string{
			"address":     arbitrumAddress,
			"private_key": arbitrumPrivateKey,
		},
		osmosisChainID: map[string]string{
			"address":     osmosisAddress,
			"private_key": osmosisPrivateKey,
		},
		ethChainID: map[string]string{
			"address":     ethAddress,
			"private_key": ethPrivateKey,
		},
		nobleChainID: map[string]string{
			"address":     nobleAddress,
			"private_key": noblePrivateKey,
		},
	}
)

func loadKeysFile(keys any) (*os.File, error) {
	f, err := os.CreateTemp("", "keys.json")
	if err != nil {
		return nil, fmt.Errorf("creating tmp file: %w", err)
	}

	if err := json.NewEncoder(f).Encode(keys); err != nil {
		return nil, fmt.Errorf("encoding keys json into temp file: %w", err)
	}

	return f, nil
}

func TestFundRebalancer_Rebalance(t *testing.T) {
	t.Run("no rebalancing necessary", func(t *testing.T) {
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
		mockConfigReader.EXPECT().GetUSDCDenom(osmosisChainID).Return(osmosisUSDCDenom, nil)
		mockConfigReader.EXPECT().GetUSDCDenom(arbitrumChainID).Return(arbitrumUSDCDenom, nil)
		mockConfigReader.On("GetChainConfig", osmosisChainID).Return(
			config.ChainConfig{
				Type:          config.ChainType_COSMOS,
				Cosmos:        &config.CosmosConfig{USDCDenom: osmosisUSDCDenom},
				SolverAddress: osmosisAddress,
			},
			nil,
		)
		mockConfigReader.On("GetChainConfig", arbitrumChainID).Return(
			config.ChainConfig{
				Type: config.ChainType_EVM,
				EVM: &config.EVMConfig{
					Contracts: config.ContractsConfig{USDCERC20Address: arbitrumUSDCDenom},
				},
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
		mockEVMClientManager.EXPECT().GetClient(mockContext, arbitrumChainID).Return(mockEVMClient, nil)
		mockDatabse := mock_database.NewMockDatabase(t)

		rebalancer, err := fundrebalancer.NewFundRebalancer(ctx, f.Name(), mockSkipGo, mockEVMClientManager, mockDatabse)
		assert.NoError(t, err)

		// setup initial state of mocks

		// no pending txns
		mockDatabse.EXPECT().GetAllPendingRebalanceTransfers(mockContext).Return(nil, nil).Maybe()
		mockDatabse.EXPECT().GetPendingRebalanceTransfersToChain(mockContext, osmosisChainID).Return(nil, nil)
		mockDatabse.EXPECT().GetPendingRebalanceTransfersToChain(mockContext, arbitrumChainID).Return(nil, nil)

		// balances higher than min amount
		mockSkipGo.EXPECT().Balance(mockContext, osmosisChainID, osmosisAddress, osmosisUSDCDenom).Return("1000", nil)
		mockEVMClient.EXPECT().GetUSDCBalance(mockContext, arbitrumUSDCDenom, arbitrumAddress).Return(big.NewInt(1000), nil)

		rebalancer.Rebalance(ctx)

		// this will fail if the rebalancer makes any extranous calls to try and
		// submit a rebalance txn, etc
	})

	t.Run("single arbitrum -> osmosis rebalance necessary", func(t *testing.T) {
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
		mockConfigReader.EXPECT().GetUSDCDenom(osmosisChainID).Return(osmosisUSDCDenom, nil)
		mockConfigReader.EXPECT().GetUSDCDenom(arbitrumChainID).Return(arbitrumUSDCDenom, nil)
		mockConfigReader.On("GetChainConfig", osmosisChainID).Return(
			config.ChainConfig{
				Type:          config.ChainType_COSMOS,
				Cosmos:        &config.CosmosConfig{USDCDenom: osmosisUSDCDenom},
				SolverAddress: osmosisAddress,
			},
			nil,
		)
		mockConfigReader.On("GetChainConfig", arbitrumChainID).Return(
			config.ChainConfig{
				Type: config.ChainType_EVM,
				EVM: &config.EVMConfig{
					Contracts: config.ContractsConfig{USDCERC20Address: arbitrumUSDCDenom},
				},
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
		mockEVMClientManager.EXPECT().GetClient(mockContext, arbitrumChainID).Return(mockEVMClient, nil)
		mockDatabse := mock_database.NewMockDatabase(t)

		rebalancer, err := fundrebalancer.NewFundRebalancer(ctx, f.Name(), mockSkipGo, mockEVMClientManager, mockDatabse)
		assert.NoError(t, err)

		// setup initial state of mocks

		// no pending txns
		mockDatabse.EXPECT().GetAllPendingRebalanceTransfers(mockContext).Return(nil, nil).Maybe()
		mockDatabse.EXPECT().GetPendingRebalanceTransfersToChain(mockContext, osmosisChainID).Return(nil, nil)
		mockDatabse.EXPECT().GetPendingRebalanceTransfersToChain(mockContext, arbitrumChainID).Return(nil, nil)

		// osmosis balance lower than min amount, arbitrum & eth balances higher than target
		mockSkipGo.EXPECT().Balance(mockContext, osmosisChainID, osmosisAddress, osmosisUSDCDenom).Return("0", nil)
		mockEVMClient.EXPECT().GetUSDCBalance(mockContext, arbitrumUSDCDenom, arbitrumAddress).Return(big.NewInt(1000), nil)

		route := &skipgo.RouteResponse{
			AmountOut:              strconv.Itoa(osmosisTargetAmount),
			Operations:             []any{"opts"},
			RequiredChainAddresses: []string{arbitrumChainID, osmosisChainID},
		}
		mockSkipGo.EXPECT().Route(mockContext, arbitrumUSDCDenom, arbitrumChainID, osmosisUSDCDenom, osmosisChainID, big.NewInt(osmosisTargetAmount)).
			Return(route, nil).Once()

		txs := []skipgo.Tx{{EVMTx: &skipgo.EVMTx{ChainID: arbitrumChainID, To: osmosisAddress, Value: "0"}}}
		mockSkipGo.EXPECT().Msgs(mockContext, arbitrumUSDCDenom, arbitrumChainID, arbitrumAddress, osmosisUSDCDenom, osmosisChainID, osmosisAddress, big.NewInt(osmosisTargetAmount), big.NewInt(osmosisTargetAmount), []string{arbitrumAddress, osmosisAddress}, route.Operations).
			Return(txs, nil).Once()

		mockSkipGo.EXPECT().SubmitTx(mockContext, mock.Anything, arbitrumChainID).
			Return(skipgo.TxHash("arbitrum hash"), nil).Once()

		// setup mock evm client txn construction calls
		mockEVMClient.On("EstimateGas", mock.Anything, mock.Anything).Return(uint64(100), nil).Twice()
		mockEVMClient.On("SuggestGasTipCap", mock.Anything).Return(big.NewInt(50), nil).Once()
		mockEVMClient.On("SuggestGasPrice", mock.Anything).Return(big.NewInt(25), nil).Once()
		mockEVMClient.On("PendingNonceAt", mock.Anything, common.HexToAddress(arbitrumAddress)).Return(uint64(1), nil).Once()

		// should insert once rebalance transaction from arbitrum to osmosis
		mockDatabse.EXPECT().InsertRebalanceTransfer(mockContext, db.InsertRebalanceTransferParams{
			TxHash:             "arbitrum hash",
			SourceChainID:      arbitrumChainID,
			DestinationChainID: osmosisChainID,
			Amount:             strconv.Itoa(osmosisTargetAmount),
		}).Return(1, nil).Once()

		rebalancer.Rebalance(ctx)
	})

	t.Run("arbitrum + ethereum -> osmosis rebalance", func(t *testing.T) {
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
				ethChainID: {
					TargetAmount:     strconv.Itoa(ethTargetAmount),
					MinAllowedAmount: strconv.Itoa(ethMinAmount),
				},
			},
		})
		mockConfigReader.EXPECT().GetUSDCDenom(osmosisChainID).Return(osmosisUSDCDenom, nil)
		mockConfigReader.EXPECT().GetUSDCDenom(arbitrumChainID).Return(arbitrumUSDCDenom, nil)
		mockConfigReader.EXPECT().GetUSDCDenom(ethChainID).Return(ethUSDCDenom, nil)
		mockConfigReader.On("GetChainConfig", osmosisChainID).Return(
			config.ChainConfig{
				Type:          config.ChainType_COSMOS,
				Cosmos:        &config.CosmosConfig{USDCDenom: osmosisUSDCDenom},
				SolverAddress: osmosisAddress,
			},
			nil,
		)
		mockConfigReader.On("GetChainConfig", arbitrumChainID).Return(
			config.ChainConfig{
				Type: config.ChainType_EVM,
				EVM: &config.EVMConfig{
					Contracts: config.ContractsConfig{USDCERC20Address: arbitrumUSDCDenom},
				},
				SolverAddress: arbitrumAddress,
			},
			nil,
		)
		mockConfigReader.On("GetChainConfig", ethChainID).Return(
			config.ChainConfig{
				Type: config.ChainType_EVM,
				EVM: &config.EVMConfig{
					Contracts: config.ContractsConfig{USDCERC20Address: ethUSDCDenom},
				},
				SolverAddress: ethAddress,
			},
			nil,
		)
		ctx = config.ConfigReaderContext(ctx, mockConfigReader)

		f, err := loadKeysFile(defaultKeys)
		assert.NoError(t, err)

		mockSkipGo := mock_skipgo.NewMockSkipGoClient(t)
		mockEVMClientManager := mock_evmrpc.NewMockEVMRPCClientManager(t)
		mockEVMClient := mock_evmrpc.NewMockEVMChainRPC(t)
		mockEVMClientManager.EXPECT().GetClient(mockContext, arbitrumChainID).Return(mockEVMClient, nil)
		mockEVMClientManager.EXPECT().GetClient(mockContext, ethChainID).Return(mockEVMClient, nil)

		// using an in memory database for this test
		mockDatabse := mock_database.NewFakeDatabase()

		rebalancer, err := fundrebalancer.NewFundRebalancer(ctx, f.Name(), mockSkipGo, mockEVMClientManager, mockDatabse)
		assert.NoError(t, err)

		// setup initial state of mocks

		// osmosis will need 100 to reach target
		mockSkipGo.EXPECT().Balance(mockContext, osmosisChainID, osmosisAddress, osmosisUSDCDenom).Return("0", nil)
		// arbitrum has 75 usdc to spare
		mockEVMClient.EXPECT().GetUSDCBalance(mockContext, arbitrumUSDCDenom, arbitrumAddress).Return(big.NewInt(125), nil)
		// eth has 25 usdc to spare
		mockEVMClient.EXPECT().GetUSDCBalance(mockContext, ethUSDCDenom, ethAddress).Return(big.NewInt(75), nil)

		// setup skip go routing mocks
		arbRoute := &skipgo.RouteResponse{
			AmountOut:              "75",
			Operations:             []any{"opts"},
			RequiredChainAddresses: []string{arbitrumChainID, osmosisChainID},
		}
		mockSkipGo.EXPECT().Route(mockContext, arbitrumUSDCDenom, arbitrumChainID, osmosisUSDCDenom, osmosisChainID, big.NewInt(75)).
			Return(arbRoute, nil).Once()

		arbTxs := []skipgo.Tx{{EVMTx: &skipgo.EVMTx{ChainID: arbitrumChainID, To: osmosisAddress, Value: "0"}}}
		mockSkipGo.EXPECT().Msgs(mockContext, arbitrumUSDCDenom, arbitrumChainID, arbitrumAddress, osmosisUSDCDenom, osmosisChainID, osmosisAddress, big.NewInt(75), big.NewInt(75), []string{arbitrumAddress, osmosisAddress}, arbRoute.Operations).
			Return(arbTxs, nil).Once()

		mockSkipGo.EXPECT().SubmitTx(mockContext, mock.Anything, arbitrumChainID).
			Return(skipgo.TxHash("arbhash"), nil).Once()

		ethRoute := &skipgo.RouteResponse{
			AmountOut:              "25",
			Operations:             []any{"opts"},
			RequiredChainAddresses: []string{ethChainID, osmosisChainID},
		}
		mockSkipGo.EXPECT().Route(mockContext, ethUSDCDenom, ethChainID, osmosisUSDCDenom, osmosisChainID, big.NewInt(25)).
			Return(ethRoute, nil).Once()

		ethTxs := []skipgo.Tx{{EVMTx: &skipgo.EVMTx{ChainID: ethChainID, To: osmosisAddress, Value: "0"}}}
		mockSkipGo.EXPECT().Msgs(mockContext, ethUSDCDenom, ethChainID, ethAddress, osmosisUSDCDenom, osmosisChainID, osmosisAddress, big.NewInt(25), big.NewInt(25), []string{ethAddress, osmosisAddress}, ethRoute.Operations).
			Return(ethTxs, nil).Once()

		mockSkipGo.EXPECT().SubmitTx(mockContext, mock.Anything, ethChainID).
			Return(skipgo.TxHash("ethhash"), nil).Once()

		// setup mock evm client txn construction calls
		mockEVMClient.On("EstimateGas", mock.Anything, mock.Anything).Return(uint64(100), nil)
		mockEVMClient.On("SuggestGasTipCap", mock.Anything).Return(big.NewInt(50), nil)
		mockEVMClient.On("SuggestGasPrice", mock.Anything).Return(big.NewInt(25), nil)
		mockEVMClient.On("PendingNonceAt", mock.Anything, common.HexToAddress(arbitrumAddress)).Return(uint64(1), nil).Once()
		mockEVMClient.On("PendingNonceAt", mock.Anything, common.HexToAddress(ethAddress)).Return(uint64(1), nil).Once()

		rebalancer.Rebalance(ctx)

		// check that db two pending rebalance txns
		transfers, err := mockDatabse.GetPendingRebalanceTransfersToChain(ctx, osmosisChainID)
		assert.NoError(t, err)
		assert.Len(t, transfers, 2)
		for _, transfer := range transfers {
			switch transfer.TxHash {
			case "ethhash":
				assert.Equal(t, "1", transfer.SourceChainID)
				assert.Equal(t, "osmosis-1", transfer.DestinationChainID)
				assert.Equal(t, "25", transfer.Amount)
			case "arbhash":
				assert.Equal(t, "42161", transfer.SourceChainID)
				assert.Equal(t, "osmosis-1", transfer.DestinationChainID)
				assert.Equal(t, "75", transfer.Amount)
			default:
				assert.FailNow(t, "got unepxected transfer hash in db", transfer.TxHash)
			}
		}
	})

	t.Run("in flight transfers are counted towards balance", func(t *testing.T) {
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
		mockConfigReader.EXPECT().GetUSDCDenom(osmosisChainID).Return(osmosisUSDCDenom, nil)
		mockConfigReader.EXPECT().GetUSDCDenom(arbitrumChainID).Return(arbitrumUSDCDenom, nil)
		mockConfigReader.On("GetChainConfig", osmosisChainID).Return(
			config.ChainConfig{
				Type:          config.ChainType_COSMOS,
				Cosmos:        &config.CosmosConfig{USDCDenom: osmosisUSDCDenom},
				SolverAddress: osmosisAddress,
			},
			nil,
		)
		mockConfigReader.On("GetChainConfig", arbitrumChainID).Return(
			config.ChainConfig{
				Type: config.ChainType_EVM,
				EVM: &config.EVMConfig{
					Contracts: config.ContractsConfig{USDCERC20Address: arbitrumUSDCDenom},
				},
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
		mockEVMClientManager.EXPECT().GetClient(mockContext, arbitrumChainID).Return(mockEVMClient, nil)
		mockDatabse := mock_database.NewMockDatabase(t)

		rebalancer, err := fundrebalancer.NewFundRebalancer(ctx, f.Name(), mockSkipGo, mockEVMClientManager, mockDatabse)
		assert.NoError(t, err)

		// setup initial state of mocks

		// single osmosis pending tx
		mockDatabse.EXPECT().GetAllPendingRebalanceTransfers(mockContext).Return([]db.GetAllPendingRebalanceTransfersRow{
			{ID: 1, TxHash: "hash", SourceChainID: arbitrumChainID, DestinationChainID: osmosisChainID, Amount: strconv.Itoa(osmosisTargetAmount)},
		}, nil).Maybe()
		mockDatabse.EXPECT().GetPendingRebalanceTransfersToChain(mockContext, osmosisChainID).Return([]db.GetPendingRebalanceTransfersToChainRow{
			{ID: 1, TxHash: "hash", SourceChainID: arbitrumChainID, DestinationChainID: osmosisChainID, Amount: strconv.Itoa(osmosisTargetAmount)},
		}, nil)
		mockDatabse.EXPECT().GetPendingRebalanceTransfersToChain(mockContext, arbitrumChainID).Return(nil, nil)

		// osmosis balance lower than min amount, arbitrum & eth balances higher than target
		mockSkipGo.EXPECT().Balance(mockContext, osmosisChainID, osmosisAddress, osmosisUSDCDenom).Return("0", nil)
		mockEVMClient.EXPECT().GetUSDCBalance(mockContext, arbitrumUSDCDenom, arbitrumAddress).Return(big.NewInt(1000), nil)

		// not expecting any calls to create/submit any transactions because a
		// rebaalnce is not necessary with the in flight txn to osmosis

		rebalancer.Rebalance(ctx)
	})

	t.Run("skips rebalance when gas threshold exceeded", func(t *testing.T) {
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
		mockConfigReader.EXPECT().GetUSDCDenom(osmosisChainID).Return(osmosisUSDCDenom, nil)
		mockConfigReader.EXPECT().GetUSDCDenom(arbitrumChainID).Return(arbitrumUSDCDenom, nil)
		mockConfigReader.On("GetChainConfig", osmosisChainID).Return(
			config.ChainConfig{
				Type:          config.ChainType_COSMOS,
				Cosmos:        &config.CosmosConfig{USDCDenom: osmosisUSDCDenom},
				SolverAddress: osmosisAddress,
			},
			nil,
		)
		mockConfigReader.On("GetChainConfig", arbitrumChainID).Return(
			config.ChainConfig{
				Type: config.ChainType_EVM,
				EVM: &config.EVMConfig{
					Contracts: config.ContractsConfig{USDCERC20Address: arbitrumUSDCDenom},
				},
				SolverAddress:              arbitrumAddress,
				MaxRebalancingGasThreshold: 50, // Set low threshold that will be exceeded
			},
			nil,
		)
		ctx = config.ConfigReaderContext(ctx, mockConfigReader)

		f, err := loadKeysFile(defaultKeys)
		assert.NoError(t, err)

		mockSkipGo := mock_skipgo.NewMockSkipGoClient(t)
		mockEVMClientManager := mock_evmrpc.NewMockEVMRPCClientManager(t)
		mockEVMClient := mock_evmrpc.NewMockEVMChainRPC(t)
		mockEVMClientManager.EXPECT().GetClient(mockContext, arbitrumChainID).Return(mockEVMClient, nil)
		mockDatabse := mock_database.NewMockDatabase(t)

		rebalancer, err := fundrebalancer.NewFundRebalancer(ctx, f.Name(), mockSkipGo, mockEVMClientManager, mockDatabse)
		assert.NoError(t, err)

		// No pending txns
		mockDatabse.EXPECT().GetPendingRebalanceTransfersToChain(mockContext, osmosisChainID).Return(nil, nil)
		mockDatabse.EXPECT().GetPendingRebalanceTransfersToChain(mockContext, arbitrumChainID).Return(nil, nil)

		// Osmosis needs funds, Arbitrum has excess
		mockSkipGo.EXPECT().Balance(mockContext, osmosisChainID, osmosisAddress, osmosisUSDCDenom).Return("0", nil)
		mockEVMClient.EXPECT().GetUSDCBalance(mockContext, arbitrumUSDCDenom, arbitrumAddress).Return(big.NewInt(200), nil)

		route := &skipgo.RouteResponse{
			AmountOut:              "100",
			Operations:             []any{"opts"},
			RequiredChainAddresses: []string{arbitrumChainID, osmosisChainID},
		}
		mockSkipGo.EXPECT().Route(mockContext, arbitrumUSDCDenom, arbitrumChainID, osmosisUSDCDenom, osmosisChainID, big.NewInt(100)).
			Return(route, nil)

		// Return transaction that will require more gas than threshold
		txs := []skipgo.Tx{{EVMTx: &skipgo.EVMTx{ChainID: arbitrumChainID, To: osmosisAddress, Value: "0"}}}
		mockSkipGo.EXPECT().Msgs(
			mockContext,
			arbitrumUSDCDenom,
			arbitrumChainID,
			arbitrumAddress,
			osmosisUSDCDenom,
			osmosisChainID,
			osmosisAddress,
			big.NewInt(100),
			big.NewInt(100),
			[]string{arbitrumAddress, osmosisAddress},
			route.Operations,
		).Return(txs, nil)

		// Return gas estimate higher than threshold
		mockEVMClient.On("EstimateGas", mock.Anything, mock.Anything).Return(uint64(100), nil)

		rebalancer.Rebalance(ctx)

		// Verify no transactions were submitted by checking database
		transfers, err := mockDatabse.GetPendingRebalanceTransfersToChain(ctx, osmosisChainID)
		assert.NoError(t, err)
		assert.Len(t, transfers, 0, "expected no transfers to be submitted due to gas threshold")
	})

}

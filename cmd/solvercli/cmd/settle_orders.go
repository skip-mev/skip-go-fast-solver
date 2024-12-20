package cmd

import (
	"fmt"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/skip-mev/go-fast-solver/hyperlane"
	"github.com/skip-mev/go-fast-solver/ordersettler/types"
	"github.com/skip-mev/go-fast-solver/shared/clients/coingecko"
	"github.com/skip-mev/go-fast-solver/shared/clients/utils"
	"github.com/skip-mev/go-fast-solver/shared/contracts/fast_transfer_gateway"
	"github.com/skip-mev/go-fast-solver/shared/evmrpc"
	"github.com/skip-mev/go-fast-solver/shared/txexecutor/evm"

	"github.com/skip-mev/go-fast-solver/db/connect"
	"github.com/skip-mev/go-fast-solver/db/gen/db"
	"github.com/skip-mev/go-fast-solver/ordersettler"
	"github.com/skip-mev/go-fast-solver/shared/clientmanager"
	"github.com/skip-mev/go-fast-solver/shared/config"
	"github.com/skip-mev/go-fast-solver/shared/keys"
	"github.com/skip-mev/go-fast-solver/shared/lmt"
	"github.com/skip-mev/go-fast-solver/shared/txexecutor/cosmos"
	"github.com/spf13/cobra"
	"go.uber.org/zap"
	"golang.org/x/net/context"
)

var settleCmd = &cobra.Command{
	Use:     "settle-orders",
	Short:   "Settle pending order batches",
	Long:    `Settle all pending order batches immediately without any threshold checks (ignoring configured BatchUUSDCSettleUpThreshold).`,
	Example: `solver settle-orders`,
	Run:     settleOrders,
}

func init() {
	rootCmd.AddCommand(settleCmd)
	settleCmd.Flags().String("config", "", "Path to config file")
}

func settleOrders(cmd *cobra.Command, args []string) {
	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	lmt.ConfigureLogger()
	ctx = lmt.LoggerContext(ctx)

	keyStoreType, err := cmd.Flags().GetString("key-store-type")
	if err != nil {
		lmt.Logger(ctx).Error("Failed to get key-store-type", zap.Error(err))
		return
	}

	keysPath, err := cmd.Flags().GetString("keys")
	if err != nil {
		lmt.Logger(ctx).Error("Failed to get keys path", zap.Error(err))
		return
	}

	sqliteDBPath, err := cmd.Flags().GetString("sqlite-db-path")
	if err != nil {
		lmt.Logger(ctx).Error("Failed to get sqlite-db-path", zap.Error(err))
		return
	}

	migrationsPath, err := cmd.Flags().GetString("migrations-path")
	if err != nil {
		lmt.Logger(ctx).Error("Failed to get migrations-path", zap.Error(err))
		return
	}

	configPath, err := cmd.Flags().GetString("config")
	if err != nil {
		lmt.Logger(ctx).Error("Failed to get config path", zap.Error(err))
		return
	}

	cfg, err := config.LoadConfig(configPath)
	if err != nil {
		lmt.Logger(ctx).Error("Unable to load config", zap.Error(err))
		return
	}

	ctx = config.ConfigReaderContext(ctx, config.NewConfigReader(cfg))

	keyStore, err := keys.GetKeyStore(keyStoreType, keys.GetKeyStoreOpts{KeyFilePath: keysPath})
	if err != nil {
		lmt.Logger(ctx).Fatal("Unable to load keystore", zap.Error(err))
	}

	cosmosTxExecutor := cosmos.DefaultSerializedCosmosTxExecutor()
	evmTxExecutor := evm.DefaultEVMTxExecutor()

	clientManager := clientmanager.NewClientManager(keyStore, cosmosTxExecutor)

	dbConn, err := connect.ConnectAndMigrate(ctx, sqliteDBPath, migrationsPath)
	if err != nil {
		lmt.Logger(ctx).Fatal("Unable to connect to db", zap.Error(err))
	}
	defer dbConn.Close()

	evmManager := evmrpc.NewEVMRPCClientManager()
	rateLimitedClient := utils.DefaultRateLimitedHTTPClient(3)
	coingeckoClient := coingecko.NewCoingeckoClient(rateLimitedClient, "https://api.coingecko.com/api/v3/", "")
	cachedCoinGeckoClient := coingecko.NewCachedPriceClient(coingeckoClient, 15*time.Minute)
	evmTxPriceOracle := evmrpc.NewOracle(cachedCoinGeckoClient)

	hype, err := hyperlane.NewMultiClientFromConfig(ctx, evmManager, keyStore, evmTxPriceOracle, evmTxExecutor)
	if err != nil {
		lmt.Logger(ctx).Fatal("creating hyperlane multi client from config", zap.Error(err))
	}

	relayer := hyperlane.NewRelayer(hype, make(map[string]string))
	relayerRunner := hyperlane.NewRelayerRunner(db.New(dbConn), hype, relayer)

	settler, err := ordersettler.NewOrderSettler(ctx, db.New(dbConn), clientManager, relayerRunner)
	if err != nil {
		lmt.Logger(ctx).Error("creating order settler", zap.Error(err))
		return
	}

	chains, err := config.GetConfigReader(ctx).GetAllChainConfigsOfType(config.ChainType_COSMOS)
	if err != nil {
		lmt.Logger(ctx).Error("error getting Cosmos chains", zap.Error(err))
		return
	}

	var pendingSettlements []db.OrderSettlement
	for _, chain := range chains {
		if chain.FastTransferContractAddress == "" {
			continue
		}

		bridgeClient, err := clientManager.GetClient(ctx, chain.ChainID)
		if err != nil {
			lmt.Logger(ctx).Error("failed to get client",
				zap.String("chainID", chain.ChainID),
				zap.Error(err))
			continue
		}

		fills, err := bridgeClient.OrderFillsByFiller(ctx, chain.FastTransferContractAddress, chain.SolverAddress)
		if err != nil {
			lmt.Logger(ctx).Error("getting order fills",
				zap.String("chainID", chain.ChainID),
				zap.Error(err))
			continue
		}

		// For each fill, check if it needs settlement
		for _, fill := range fills {
			sourceChainID, err := config.GetConfigReader(ctx).GetChainIDByHyperlaneDomain(strconv.Itoa(int(fill.SourceDomain)))
			if err != nil {
				lmt.Logger(ctx).Error("failed to get source chain ID",
					zap.Uint32("domain", fill.SourceDomain),
					zap.Error(err))
				continue
			}

			sourceGatewayAddress, err := config.GetConfigReader(ctx).GetGatewayContractAddress(sourceChainID)
			if err != nil {
				lmt.Logger(ctx).Error("getting source gateway address",
					zap.String("chainID", sourceChainID),
					zap.Error(err))
				continue
			}

			sourceBridgeClient, err := clientManager.GetClient(ctx, sourceChainID)
			if err != nil {
				lmt.Logger(ctx).Error("getting source chain client",
					zap.String("chainID", sourceChainID),
					zap.Error(err))
				continue
			}

			status, err := sourceBridgeClient.OrderStatus(ctx, sourceGatewayAddress, fill.OrderID)
			if err != nil {
				lmt.Logger(ctx).Error("getting order status",
					zap.String("orderID", fill.OrderID),
					zap.Error(err))
				continue
			}

			if status != fast_transfer_gateway.OrderStatusUnfilled {
				continue
			}

			pendingSettlements = append(pendingSettlements, db.OrderSettlement{
				SourceChainID:                     sourceChainID,
				DestinationChainID:                chain.ChainID,
				SourceChainGatewayContractAddress: sourceGatewayAddress,
				OrderID:                           fill.OrderID,
			})
		}
	}

	if len(pendingSettlements) == 0 {
		fmt.Println("No pending settlement batches found")
		return
	}

	batches := types.IntoSettlementBatchesByChains(pendingSettlements)
	fmt.Printf("Found %d pending settlement batches\n", len(batches))

	hashes, err := settler.SettleBatches(ctx, batches)
	if err != nil {
		lmt.Logger(ctx).Error("settling pending batches", zap.Error(err))
		return
	}

	for i, batch := range batches {
		hash := hashes[i]

		fmt.Printf("Initiated settlement batch %d:\n", i+1)
		fmt.Printf("Source Chain: %s\n", batch.SourceChainID())
		fmt.Printf("Destination Chain: %s\n", batch.DestinationChainID())
		fmt.Printf("Number of Orders: %d\n", len(batch.OrderIDs()))
		fmt.Printf("Transaction Hash: %s\n", hash)
	}
}
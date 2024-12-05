package cmd

import (
	"fmt"
	"os/signal"
	"syscall"
	"time"

	"github.com/skip-mev/go-fast-solver/hyperlane"
	"github.com/skip-mev/go-fast-solver/shared/clients/coingecko"
	"github.com/skip-mev/go-fast-solver/shared/clients/utils"
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
	Use:   "settle-orders",
	Short: "Settle pending order batches",
	Long: `Settle all pending order batches immediately without any threshold checks (ignoring configured BatchUUSDCSettleUpThreshold).
Example:
  ./build/solvercli settle-orders --config ./config/local/config.yml \
	  --key-store-type plaintext-file \
	  --keys ./config/local/keys.json \
	  --sqlite-db-path ./solver.db \
	  --migrations-path ./db/migrations`,
	Run: settleOrders,
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

	batches, err := settler.PendingSettlementBatches(ctx)
	if err != nil {
		lmt.Logger(ctx).Error("getting pending settlement batches", zap.Error(err))
		return
	}

	if len(batches) == 0 {
		fmt.Println("No pending settlement batches found")
		return
	}

	fmt.Printf("Found %d pending settlement batches\n", len(batches))

	hashes, err := settler.SettleBatches(ctx, batches)
	if err != nil {
		lmt.Logger(ctx).Error("settling pending batches", zap.Error(err))
		return
	}

	// Submit settlements for relay
	for i, batch := range batches {
		hash := hashes[i]
		if err := settler.RelayBatch(ctx, hash, batch); err != nil {
			lmt.Logger(ctx).Error("submitting settlement for relay",
				zap.Error(err),
				zap.String("txHash", hash),
				zap.String("sourceChain", batch.SourceChainID()),
				zap.String("destinationChain", batch.DestinationChainID()),
			)
			continue
		}

		fmt.Printf("Submitted settlement batch %d for relay:\n", i+1)
		fmt.Printf("Source Chain: %s\n", batch.SourceChainID())
		fmt.Printf("Destination Chain: %s\n", batch.DestinationChainID())
		fmt.Printf("Number of Orders: %d\n", len(batch.OrderIDs()))
		fmt.Printf("Transaction Hash: %s\n", hash)
	}
}

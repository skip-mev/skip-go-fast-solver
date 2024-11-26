package cmd

import (
	"fmt"
	"os/signal"
	"syscall"
	"time"

	"github.com/skip-mev/go-fast-solver/db/connect"
	"github.com/skip-mev/go-fast-solver/db/gen/db"
	"github.com/skip-mev/go-fast-solver/orderfulfiller/order_fulfillment_handler"
	"github.com/skip-mev/go-fast-solver/shared/clientmanager"
	"github.com/skip-mev/go-fast-solver/shared/config"
	"github.com/skip-mev/go-fast-solver/shared/keys"
	"github.com/skip-mev/go-fast-solver/shared/lmt"
	"github.com/skip-mev/go-fast-solver/shared/txexecutor/cosmos"
	"github.com/spf13/cobra"
	"go.uber.org/zap"
	"golang.org/x/net/context"
)

var initiateTimeoutCmd = &cobra.Command{
	Use:   "initiate-timeout",
	Short: "Initiate timeout for an expired order",
	Long: `Initiate timeout for an expired order that hasn't been filled.
Example:
  ./build/solvercli initiate-timeout --order-id <order_id> --config ./config/local/config.yml`,
	Run: initiateTimeout,
}

func init() {
	rootCmd.AddCommand(initiateTimeoutCmd)
	initiateTimeoutCmd.Flags().String("order-id", "", "ID of the order to timeout")
	initiateTimeoutCmd.Flags().String("config", "", "Path to config file")
	initiateTimeoutCmd.MarkFlagRequired("order-id")
}

func initiateTimeout(cmd *cobra.Command, args []string) {
	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	lmt.ConfigureLogger()
	ctx = lmt.LoggerContext(ctx)

	orderID, err := cmd.Flags().GetString("order-id")
	if err != nil {
		lmt.Logger(ctx).Error("Failed to get order-id", zap.Error(err))
		return
	}

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

	dbConn, err := connect.ConnectAndMigrate(ctx, sqliteDBPath, migrationsPath)
	if err != nil {
		lmt.Logger(ctx).Fatal("Unable to connect to db", zap.Error(err))
	}
	defer dbConn.Close()

	queries := db.New(dbConn)

	order, err := queries.GetOrderByOrderID(ctx, orderID)
	if err != nil {
		lmt.Logger(ctx).Error("Failed to get order", zap.Error(err), zap.String("orderID", orderID))
		return
	}

	if time.Now().UTC().Before(order.TimeoutTimestamp.UTC()) {
		lmt.Logger(ctx).Error("Order has not expired yet",
			zap.String("orderID", orderID),
			zap.Time("timeoutTimestamp", order.TimeoutTimestamp))
		return
	}

	cosmosTxExecutor := cosmos.DefaultSerializedCosmosTxExecutor()
	clientManager := clientmanager.NewClientManager(keyStore, cosmosTxExecutor)

	fillHandler := order_fulfillment_handler.NewOrderFulfillmentHandler(queries, clientManager, nil)

	txHash, err := fillHandler.InitiateTimeout(ctx, order)
	if err != nil {
		lmt.Logger(ctx).Error("Failed to initiate timeout", zap.Error(err))
		return
	}

	if err := fillHandler.SubmitTimeoutForRelay(ctx, order, txHash); err != nil {
		lmt.Logger(ctx).Error("Failed to submit timeout for relay", zap.Error(err))
		return
	}

	fmt.Printf("Successfully initiated timeout for order %s\n", orderID)
	fmt.Printf("Transaction hash: %s\n", txHash)
}

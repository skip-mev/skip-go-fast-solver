package main

import (
	"context"
	"flag"
	"fmt"
	"github.com/skip-mev/go-fast-solver/txverifier"
	"os/signal"
	"syscall"

	_ "github.com/golang-migrate/migrate/v4/source/file"
	_ "github.com/mattn/go-sqlite3"
	"github.com/skip-mev/go-fast-solver/db/connect"
	"github.com/skip-mev/go-fast-solver/db/gen/db"
	"github.com/skip-mev/go-fast-solver/fundrebalancer"
	"github.com/skip-mev/go-fast-solver/hyperlane"
	"github.com/skip-mev/go-fast-solver/orderfulfiller"
	"github.com/skip-mev/go-fast-solver/orderfulfiller/order_fulfillment_handler"
	"github.com/skip-mev/go-fast-solver/ordersettler"
	"github.com/skip-mev/go-fast-solver/shared/clientmanager"
	"github.com/skip-mev/go-fast-solver/shared/clients/skipgo"
	"github.com/skip-mev/go-fast-solver/shared/config"
	"github.com/skip-mev/go-fast-solver/shared/evmrpc"
	"github.com/skip-mev/go-fast-solver/shared/keys"
	"github.com/skip-mev/go-fast-solver/shared/lmt"
	"github.com/skip-mev/go-fast-solver/shared/metrics"
	"github.com/skip-mev/go-fast-solver/transfermonitor"
	"go.uber.org/zap"
	"golang.org/x/sync/errgroup"
)

var configPath = flag.String("config", "./config/local/config.yml", "path to solver config file")
var keysPath = flag.String("keys", "./config/local/keys.json", "path to solver key file. must be specified if key-store-type is plaintext-file or encrpyted-file")
var keyStoreType = flag.String("key-store-type", "plaintext-file", "where to load the solver keys from. (plaintext-file, encrypted-file, env)")
var aesKeyHex = flag.String("aes-key-hex", "", "hex-encoded AES key used to decrypt keys file. must be specified if key-store-type is encrypted-file")
var sqliteDBPath = flag.String("sqlite-db-path", "./solver.db", "path to sqlite db file")
var migrationsPath = flag.String("migrations-path", "./db/migrations", "path to db migrations directory")
var quickStart = flag.Bool("quickstart", false, "run quick start mode")
var refundOrders = flag.Bool("refund-orders", true, "if the solver should refund timed out order")

func main() {
	flag.Parse()

	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	lmt.ConfigureLogger()
	ctx = lmt.LoggerContext(ctx)

	promMetrics := metrics.NewPromMetrics()
	ctx = metrics.ContextWithMetrics(ctx, promMetrics)

	cfg, err := config.LoadConfig(*configPath)
	if err != nil {
		lmt.Logger(ctx).Fatal("Unable to load config", zap.Error(err))
	}
	ctx = config.ConfigReaderContext(ctx, config.NewConfigReader(cfg))

	keyStore, err := keys.GetKeyStore(*keyStoreType, keys.GetKeyStoreOpts{KeyFilePath: *keysPath, AESKeyHex: *aesKeyHex})
	if err != nil {
		lmt.Logger(ctx).Fatal("Unable to load keystore", zap.Error(err))
	}

	clientManager := clientmanager.NewClientManager(keyStore)

	dbConn, err := connect.ConnectAndMigrate(ctx, *sqliteDBPath, *migrationsPath)
	if err != nil {
		lmt.Logger(ctx).Fatal("Unable to connect to db", zap.Error(err))
	}
	defer dbConn.Close()

	skipgo, err := skipgo.NewSkipGoClient("https://api.skip.build")
	if err != nil {
		lmt.Logger(ctx).Fatal("Unable to create Skip Go client", zap.Error(err))
	}

	evmManager := evmrpc.NewEVMRPCClientManager()

	eg, ctx := errgroup.WithContext(ctx)

	// Uncomment this section to run a prometheus server for metrics collection
	//eg.Go(func() error {
	//	lmt.Logger(ctx).Info("Starting Prometheus")
	//	if err := metrics.StartPrometheus(ctx, fmt.Sprintf(cfg.Metrics.PrometheusAddress)); err != nil {
	//		return err
	//	}
	//	return nil
	//})

	eg.Go(func() error {
		transferMonitor := transfermonitor.NewTransferMonitor(db.New(dbConn), *quickStart)
		err := transferMonitor.Start(ctx)
		if err != nil {
			return fmt.Errorf("creating transfer monitor: %w", err)
		}
		return nil
	})

	eg.Go(func() error {
		orderFillHandler, err := order_fulfillment_handler.NewOrderFulfillmentHandler(ctx, db.New(dbConn), clientManager)
		if err != nil {
			return err
		}
		r, err := orderfulfiller.NewOrderFulfiller(
			ctx,
			db.New(dbConn),
			cfg.OrderFillerConfig.OrderFillWorkerCount,
			orderFillHandler,
			true,
			*refundOrders,
		)
		if err != nil {
			return fmt.Errorf("creating order filler: %w", err)
		}
		r.Run(ctx)
		return nil
	})

	eg.Go(func() error {
		r, err := txverifier.NewTxVerifier(ctx, db.New(dbConn), clientManager)
		if err != nil {
			return err
		}
		r.Run(ctx)
		return nil
	})

	eg.Go(func() error {
		r, err := ordersettler.NewOrderSettler(ctx, db.New(dbConn), clientManager)
		if err != nil {
			return fmt.Errorf("creating order settler: %w", err)
		}
		r.Run(ctx)
		return nil
	})

	eg.Go(func() error {
		r, err := fundrebalancer.NewFundRebalancer(ctx, *keysPath, skipgo, evmManager, db.New(dbConn))
		if err != nil {
			return fmt.Errorf("creating fund rebalancer: %w", err)
		}
		r.Run(ctx)
		return nil
	})

	eg.Go(func() error {
		hype, err := hyperlane.NewMultiClientFromConfig(ctx, evmManager, keyStore)
		if err != nil {
			return fmt.Errorf("creating hyperlane multi client from config: %w", err)
		}

		relayer := hyperlane.NewRelayer(hype, make(map[string]string))
		err = hyperlane.NewRelayerRunner(db.New(dbConn), hype, relayer).Run(ctx)
		if err != nil {
			return fmt.Errorf("relaying message: %v", err)
		}

		return nil
	})

	if err := eg.Wait(); err != nil {
		lmt.Logger(ctx).Fatal("error running solver", zap.Error(err))
	}
}

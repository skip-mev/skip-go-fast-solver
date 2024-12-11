package main

import (
	"context"
	"flag"
	"fmt"
	"os/signal"
	"syscall"
	"time"

	"github.com/skip-mev/go-fast-solver/gasmonitor"

	"github.com/skip-mev/go-fast-solver/shared/oracle"
	"github.com/skip-mev/go-fast-solver/shared/txexecutor/cosmos"
	"github.com/skip-mev/go-fast-solver/shared/txexecutor/evm"

	"github.com/skip-mev/go-fast-solver/db/connect"
	"github.com/skip-mev/go-fast-solver/db/gen/db"
	"github.com/skip-mev/go-fast-solver/txverifier"

	"github.com/skip-mev/go-fast-solver/fundrebalancer"
	"github.com/skip-mev/go-fast-solver/hyperlane"
	"github.com/skip-mev/go-fast-solver/orderfulfiller"
	"github.com/skip-mev/go-fast-solver/orderfulfiller/order_fulfillment_handler"
	"github.com/skip-mev/go-fast-solver/ordersettler"
	"github.com/skip-mev/go-fast-solver/shared/clientmanager"
	"github.com/skip-mev/go-fast-solver/shared/clients/coingecko"
	"github.com/skip-mev/go-fast-solver/shared/clients/skipgo"
	"github.com/skip-mev/go-fast-solver/shared/clients/utils"

	_ "github.com/golang-migrate/migrate/v4/source/file"
	_ "github.com/mattn/go-sqlite3"
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
var sqliteDBPath = flag.String("sqlite-db-path", "./solver.db", "path to sqlite db file")
var migrationsPath = flag.String("migrations-path", "./db/migrations", "path to db migrations directory")
var quickStart = flag.Bool("quickstart", true, "run quick start mode")
var refundOrders = flag.Bool("refund-orders", false, "if the solver should refund timed out order")
var fillOrders = flag.Bool("fill-orders", true, "if the solver should fill orders")

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
	redactedConfig := redactConfig(&cfg)

	lmt.Logger(ctx).Info("starting skip go fast solver",
		zap.Any("config", redactedConfig), zap.Bool("quickstart", *quickStart),
		zap.Bool("shouldRefundOrders", *refundOrders))

	ctx = config.ConfigReaderContext(ctx, config.NewConfigReader(cfg))

	keyStore, err := keys.GetKeyStore(*keyStoreType, keys.GetKeyStoreOpts{KeyFilePath: *keysPath})
	if err != nil {
		lmt.Logger(ctx).Fatal("Unable to load keystore", zap.Error(err))
	}

	cosmosTxExecutor := cosmos.DefaultSerializedCosmosTxExecutor()
	evmTxExecutor := evm.DefaultEVMTxExecutor()

	clientManager := clientmanager.NewClientManager(keyStore, cosmosTxExecutor)

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
	rateLimitedClient := utils.DefaultRateLimitedHTTPClient(3)
	coingeckoClient := coingecko.NewCoingeckoClient(rateLimitedClient, "https://api.coingecko.com/api/v3/", "")
	cachedCoinGeckoClient := coingecko.NewCachedPriceClient(coingeckoClient, 15*time.Minute)
	txPriceOracle := oracle.NewOracle(cachedCoinGeckoClient)

	hype, err := hyperlane.NewMultiClientFromConfig(ctx, evmManager, keyStore, txPriceOracle, evmTxExecutor)
	if err != nil {
		lmt.Logger(ctx).Fatal("creating hyperlane multi client from config", zap.Error(err))
	}

	relayer := hyperlane.NewRelayer(hype, make(map[string]string))
	relayerRunner := hyperlane.NewRelayerRunner(db.New(dbConn), hype, relayer)

	eg, ctx := errgroup.WithContext(ctx)

	eg.Go(func() error {
		lmt.Logger(ctx).Info("Starting Prometheus")
		if err := metrics.StartPrometheus(ctx, cfg.Metrics.PrometheusAddress); err != nil {
			return err
		}
		return nil
	})

	eg.Go(func() error {
		orderFillHandler := order_fulfillment_handler.NewOrderFulfillmentHandler(db.New(dbConn), clientManager, relayerRunner)
		r, err := orderfulfiller.NewOrderFulfiller(
			ctx,
			db.New(dbConn),
			cfg.OrderFillerConfig.OrderFillWorkerCount,
			orderFillHandler,
			*fillOrders,
			*refundOrders,
		)
		if err != nil {
			return fmt.Errorf("creating order filler: %w", err)
		}
		r.Run(ctx)
		return nil
	})

	eg.Go(func() error {
		r, err := ordersettler.NewOrderSettler(ctx, db.New(dbConn), clientManager, relayerRunner)
		if err != nil {
			return fmt.Errorf("creating order settler: %w", err)
		}
		r.Run(ctx)
		return nil
	})

	eg.Go(func() error {
		r, err := fundrebalancer.NewFundRebalancer(ctx, keyStore, skipgo, evmManager, db.New(dbConn), txPriceOracle, evmTxExecutor)
		if err != nil {
			return fmt.Errorf("creating fund rebalancer: %w", err)
		}
		r.Run(ctx)
		return nil
	})

	eg.Go(func() error {
		r, err := txverifier.NewTxVerifier(ctx, db.New(dbConn), clientManager, txPriceOracle)
		if err != nil {
			return err
		}
		r.Run(ctx)
		return nil
	})

	eg.Go(func() error {
		transferMonitor := transfermonitor.NewTransferMonitor(db.New(dbConn), *quickStart, cfg.TransferMonitorConfig.PollInterval)
		err := transferMonitor.Start(ctx)
		if err != nil {
			return fmt.Errorf("creating transfer monitor: %w", err)
		}
		return nil
	})

	eg.Go(func() error {
		gasMonitor := gasmonitor.NewGasMonitor(clientManager)
		err := gasMonitor.Start(ctx)
		if err != nil {
			return fmt.Errorf("creating gas monitor: %w", err)
		}
		return nil
	})

	eg.Go(func() error {
		if err := relayerRunner.Run(ctx); err != nil {
			return fmt.Errorf("relayer runner: %w", err)
		}
		return nil
	})

	if err := eg.Wait(); err != nil {
		lmt.Logger(ctx).Fatal("error running solver", zap.Error(err))
	}
}

func redactConfig(cfg *config.Config) config.Config {
	redactedConfig := *cfg
	redactedConfig.Chains = make(map[string]config.ChainConfig)

	for chainID, chain := range cfg.Chains {
		chainCopy := chain
		if chainCopy.Cosmos != nil {
			cosmosCopy := *chainCopy.Cosmos
			cosmosCopy.RPC = "[redacted]"
			cosmosCopy.GRPC = "[redacted]"
			cosmosCopy.RPCBasicAuthVar = "[redacted]"
			chainCopy.Cosmos = &cosmosCopy
		}
		if chainCopy.EVM != nil {
			evmCopy := *chainCopy.EVM
			evmCopy.RPC = "[redacted]"
			evmCopy.RPCBasicAuthVar = "[redacted]"
			chainCopy.EVM = &evmCopy
		}
		redactedConfig.Chains[chainID] = chainCopy
	}

	return redactedConfig
}

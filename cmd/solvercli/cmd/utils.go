package cmd

import (
	"context"
	"fmt"
	"github.com/skip-mev/go-fast-solver/db/connect"
	"github.com/skip-mev/go-fast-solver/db/gen/db"
	"github.com/skip-mev/go-fast-solver/shared/clientmanager"
	"github.com/skip-mev/go-fast-solver/shared/config"
	"github.com/skip-mev/go-fast-solver/shared/evmrpc"
	"github.com/skip-mev/go-fast-solver/shared/keys"
	"github.com/skip-mev/go-fast-solver/shared/lmt"
	"github.com/skip-mev/go-fast-solver/shared/txexecutor/cosmos"
	"github.com/spf13/cobra"
	"go.uber.org/zap"
	"math/big"
	"os"
	"os/signal"
	"strings"
	"syscall"
)

const (
	CCTP_TOKEN_DECIMALS = 6
)

func setupContext(cmd *cobra.Command) context.Context {
	ctx, cancel := context.WithCancel(context.Background())

	go func() {
		sigCh := make(chan os.Signal, 1)
		signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
		select {
		case <-sigCh:
			cancel()
		case <-ctx.Done():
			return
		}
	}()

	lmt.ConfigureLogger()
	ctx = lmt.LoggerContext(ctx)

	configPath, err := cmd.Flags().GetString("config")
	if err != nil {
		lmt.Logger(ctx).Fatal("Failed to get config path", zap.Error(err))
	}

	cfg, err := config.LoadConfig(configPath)
	if err != nil {
		lmt.Logger(ctx).Fatal("Failed to load config", zap.Error(err))
	}

	return config.ConfigReaderContext(ctx, config.NewConfigReader(cfg))
}

func setupClients(ctx context.Context, cmd *cobra.Command) (evmrpc.EVMRPCClientManager, *clientmanager.ClientManager) {
	evmClientManager := evmrpc.NewEVMRPCClientManager()

	keysPath, err := cmd.Flags().GetString("keys")
	if err != nil {
		lmt.Logger(ctx).Fatal("Error reading keys path", zap.Error(err))
	}

	keyStoreType, err := cmd.Flags().GetString("key-store-type")
	if err != nil {
		lmt.Logger(ctx).Fatal("Error reading key-store-type", zap.Error(err))
	}

	keyStore, err := keys.GetKeyStore(keyStoreType, keys.GetKeyStoreOpts{KeyFilePath: keysPath})
	if err != nil {
		lmt.Logger(ctx).Fatal("Unable to load keystore", zap.Error(err))
	}

	cosmosTxExecutor := cosmos.DefaultSerializedCosmosTxExecutor()
	return evmClientManager, clientmanager.NewClientManager(keyStore, cosmosTxExecutor)
}

func normalizeBalance(balance *big.Int, decimals uint8) string {
	if balance == nil {
		return "0"
	}

	balanceInt := new(big.Int).SetBytes(balance.Bytes())
	balanceFloat := new(big.Float)
	balanceFloat.SetInt(balanceInt)

	tokenPrecision := new(big.Int).SetInt64(10)
	tokenPrecision.Exp(tokenPrecision, big.NewInt(int64(decimals)), nil)

	tokenPrecisionFloat := new(big.Float).SetInt(tokenPrecision)

	normalizedBalance := new(big.Float)
	normalizedBalance = normalizedBalance.SetMode(big.ToNegativeInf).SetPrec(53) // float prec
	normalizedBalance = normalizedBalance.Quo(balanceFloat, tokenPrecisionFloat)

	str := fmt.Sprintf("%.18f", normalizedBalance)
	if strings.Contains(str, ".") {
		str = strings.TrimRight(strings.TrimRight(str, "0"), ".")
	}

	return str
}

func setupDatabase(ctx context.Context, cmd *cobra.Command) (*db.Queries, error) {
	sqliteDBPath, err := cmd.Flags().GetString("sqlite-db-path")
	if err != nil {
		return nil, fmt.Errorf("getting sqlite-db-path: %w", err)
	}

	migrationsPath, err := cmd.Flags().GetString("migrations-path")
	if err != nil {
		return nil, fmt.Errorf("getting migrations-path: %w", err)
	}

	dbConn, err := connect.ConnectAndMigrate(ctx, sqliteDBPath, migrationsPath)
	if err != nil {
		return nil, fmt.Errorf("connecting to database: %w", err)
	}

	return db.New(dbConn), nil
}

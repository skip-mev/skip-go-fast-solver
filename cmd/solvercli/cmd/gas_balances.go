package cmd

import (
	"fmt"
	"math/big"
	"os/signal"
	"strings"
	"syscall"

	"github.com/skip-mev/go-fast-solver/shared/clientmanager"
	"github.com/skip-mev/go-fast-solver/shared/config"
	"github.com/skip-mev/go-fast-solver/shared/keys"
	"github.com/skip-mev/go-fast-solver/shared/lmt"
	"github.com/skip-mev/go-fast-solver/shared/metrics"
	"github.com/skip-mev/go-fast-solver/shared/txexecutor/cosmos"
	"github.com/spf13/cobra"
	"go.uber.org/zap"
	"golang.org/x/net/context"
)

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

var gasBalancesCmd = &cobra.Command{
	Use:   "gas-balances",
	Short: "Display gas balances for all configured chains",
	Long: `Display the current gas balances for all chains configured in config.yml.
	Example: 
	./build/solvercli gas-balances`,
	Run: func(cmd *cobra.Command, args []string) {
		ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
		defer cancel()

		lmt.ConfigureLogger()
		ctx = lmt.LoggerContext(ctx)

		configPath, err := cmd.Flags().GetString("config")
		if err != nil {
			lmt.Logger(ctx).Error("Error reading config command line argument", zap.Error(err))
			return
		}
		keysPath, err := cmd.Flags().GetString("keys")
		if err != nil {
			lmt.Logger(ctx).Error("Error reading keys command line argument", zap.Error(err))
			return
		}
		keyStoreType, err := cmd.Flags().GetString("key-store-type")
		if err != nil {
			lmt.Logger(ctx).Error("Error reading key-store-type command line argument", zap.Error(err))
			return
		}

		promMetrics := metrics.NewPromMetrics()
		ctx = metrics.ContextWithMetrics(ctx, promMetrics)

		cfg, err := config.LoadConfig(configPath)
		if err != nil {
			lmt.Logger(ctx).Error("failed to load config", zap.Error(err))
			return
		}
		ctx = config.ConfigReaderContext(ctx, config.NewConfigReader(cfg))

		keyStore, err := keys.GetKeyStore(keyStoreType, keys.GetKeyStoreOpts{KeyFilePath: keysPath})
		if err != nil {
			lmt.Logger(ctx).Error("Unable to load keystore", zap.Error(err))
			return
		}

		cosmosTxExecutor := cosmos.DefaultSerializedCosmosTxExecutor()
		clientManager := clientmanager.NewClientManager(keyStore, cosmosTxExecutor)

		var chains []config.ChainConfig
		evmChains, err := config.GetConfigReader(ctx).GetAllChainConfigsOfType(config.ChainType_EVM)
		if err != nil {
			lmt.Logger(ctx).Error("error getting EVM chains", zap.Error(err))
			return
		}
		cosmosChains, err := config.GetConfigReader(ctx).GetAllChainConfigsOfType(config.ChainType_COSMOS)
		if err != nil {
			lmt.Logger(ctx).Error("error getting cosmos chains", zap.Error(err))
			return
		}
		chains = append(chains, evmChains...)
		chains = append(chains, cosmosChains...)

		fmt.Printf("\nGas Balances:\n")
		fmt.Printf("%-20s %-15s %-25s %-25s %-25s\n", "Chain", "Symbol", "Balance", "Warning", "Critical")
		fmt.Printf("%s\n", strings.Repeat("-", 110))

		for _, chain := range chains {
			client, err := clientManager.GetClient(ctx, chain.ChainID)
			if err != nil {
				lmt.Logger(ctx).Error("failed to get client", zap.String("chain", chain.ChainName), zap.Error(err))
				continue
			}

			balance, err := client.SignerGasTokenBalance(ctx)
			if err != nil {
				lmt.Logger(ctx).Error("failed to get balance", zap.String("chain", chain.ChainName), zap.Error(err))
				continue
			}

			warning, critical, err := config.GetConfigReader(ctx).GetGasAlertThresholds(chain.ChainID)
			if err != nil {
				lmt.Logger(ctx).Error("failed to get thresholds", zap.String("chain", chain.ChainName), zap.Error(err))
				continue
			}

			fmt.Printf("%-20s %-15s %-25s %-25s %-25s\n",
				chain.ChainName,
				chain.GasTokenSymbol,
				normalizeBalance(balance, chain.GasTokenDecimals),
				normalizeBalance(warning, chain.GasTokenDecimals),
				normalizeBalance(critical, chain.GasTokenDecimals),
			)
		}
	},
}

func init() {
	rootCmd.AddCommand(gasBalancesCmd)
}

package cmd

import (
	"context"
	"fmt"
	"math/big"
	"os/signal"
	"strings"
	"syscall"

	"github.com/skip-mev/go-fast-solver/shared/clientmanager"
	"github.com/skip-mev/go-fast-solver/shared/keys"
	"github.com/skip-mev/go-fast-solver/shared/txexecutor/cosmos"

	"github.com/skip-mev/go-fast-solver/ordersettler"
	"github.com/skip-mev/go-fast-solver/shared/config"
	"github.com/skip-mev/go-fast-solver/shared/evmrpc"
	"github.com/skip-mev/go-fast-solver/shared/lmt"
	"github.com/spf13/cobra"
	"go.uber.org/zap"
)

var inventoryCmd = &cobra.Command{
	Use:   "inventory",
	Short: "Show current solver inventory across all chains (excluding pending fund rebalances)",
	Run: func(cmd *cobra.Command, args []string) {
		ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
		defer cancel()

		lmt.ConfigureLogger()
		ctx = lmt.LoggerContext(ctx)
		lmt.Logger(ctx).Info("entered inventory function")

		configPath, err := cmd.Flags().GetString("config")
		if err != nil {
			lmt.Logger(ctx).Fatal("Failed to get config path", zap.Error(err))
		}

		cfg, err := config.LoadConfig(configPath)
		if err != nil {
			lmt.Logger(ctx).Fatal("Failed to load config", zap.Error(err))
		}

		lmt.Logger(ctx).Info("loaded config")

		ctx = config.ConfigReaderContext(ctx, config.NewConfigReader(cfg))

		evmClientManager := evmrpc.NewEVMRPCClientManager()

		ctx = config.ConfigReaderContext(ctx, config.NewConfigReader(cfg))

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

		keyStore, err := keys.GetKeyStore(keyStoreType, keys.GetKeyStoreOpts{KeyFilePath: keysPath})
		if err != nil {
			lmt.Logger(ctx).Error("Unable to load keystore", zap.Error(err))
			return
		}

		cosmosTxExecutor := cosmos.DefaultSerializedCosmosTxExecutor()
		cctpClientManager := clientmanager.NewClientManager(keyStore, cosmosTxExecutor)

		inventory, err := getInventory(ctx, evmClientManager, cctpClientManager)
		if err != nil {
			lmt.Logger(ctx).Fatal("Failed to get inventory", zap.Error(err))
		}

		totalBalance := new(big.Int)
		totalPendingSettlements := new(big.Int)
		totalGrossProfit := new(big.Int)
		totalPosition := new(big.Int)

		for _, inv := range inventory {
			totalBalance.Add(totalBalance, inv.CurrentBalance)
			totalPendingSettlements.Add(totalPendingSettlements, inv.PendingSettlements)
			totalGrossProfit.Add(totalGrossProfit, inv.GrossProfit)
			totalPosition.Add(totalPosition, inv.TotalPosition)
		}

		fmt.Println("\nSolver Inventory Summary:")
		fmt.Println("------------------------")

		for chainID, inv := range inventory {
			fmt.Printf("\nChain: %s\n", chainID)
			fmt.Printf("  Current Balance: %s USDC\n", normalizeBalance(inv.CurrentBalance, 6))
			fmt.Printf("  Pending Settlements: %s USDC\n", normalizeBalance(inv.PendingSettlements, 6))
			fmt.Printf("  Gross Profit: %s USDC\n", normalizeBalance(inv.GrossProfit, 6))
			fmt.Printf("  Total Position: %s USDC\n", normalizeBalance(inv.TotalPosition, 6))
		}

		fmt.Printf("\nTotals Across All Chains:")
		fmt.Printf("\n------------------------\n")
		fmt.Printf("  Total Balance: %s USDC\n", normalizeBalance(totalBalance, 6))
		fmt.Printf("  Total Pending Settlements: %s USDC\n", normalizeBalance(totalPendingSettlements, 6))
		fmt.Printf("  Total Gross Profit: %s USDC\n", normalizeBalance(totalGrossProfit, 6))
		fmt.Printf("  Total Position: %s USDC\n", normalizeBalance(totalPosition, 6))
	},
}

type ChainInventory struct {
	CurrentBalance     *big.Int
	PendingSettlements *big.Int
	GrossProfit        *big.Int
	TotalPosition      *big.Int
}

func getInventory(ctx context.Context, evmClientManager evmrpc.EVMRPCClientManager,
	cctpClientManager *clientmanager.ClientManager) (map[string]*ChainInventory, error) {
	inventory := make(map[string]*ChainInventory)
	chains := config.GetConfigReader(ctx).Config().Chains

	for _, chain := range chains {
		inventory[chain.ChainID] = &ChainInventory{
			CurrentBalance:     new(big.Int),
			PendingSettlements: new(big.Int),
			GrossProfit:        new(big.Int),
			TotalPosition:      new(big.Int),
		}
	}

	for chainID := range inventory {
		chainConfig, err := config.GetConfigReader(ctx).GetChainConfig(chainID)
		if err != nil {
			return nil, fmt.Errorf("getting chain config for %s: %w", chainID, err)
		}

		var balance *big.Int
		switch chainConfig.Type {
		case config.ChainType_EVM:
			client, err := evmClientManager.GetClient(ctx, chainID)
			if err != nil {
				return nil, fmt.Errorf("getting evm client for chain %s: %w", chainID, err)
			}
			balance, err = client.GetUSDCBalance(ctx, chainConfig.USDCDenom, chainConfig.SolverAddress)
			if err != nil {
				return nil, fmt.Errorf("getting balance for %s: %w", chainID, err)
			}
			lmt.Logger(ctx).Info("Balance",
				zap.String("chain_id", chainID),
				zap.String("type", "EVM"),
				zap.String("balance", balance.String()),
				zap.String("address", chainConfig.SolverAddress),
				zap.String("denom", chainConfig.USDCDenom))

		case config.ChainType_COSMOS:
			client, err := cctpClientManager.GetClient(ctx, chainID)
			if err != nil {
				return nil, fmt.Errorf("getting balance for %s: %w", chainID, err)
			}
			balance, err = client.Balance(ctx, chainConfig.SolverAddress, chainConfig.USDCDenom)
			if err != nil {
				return nil, fmt.Errorf("getting balance for %s: %w", chainID, err)
			}
			lmt.Logger(ctx).Info("Balance", zap.String("chain_id", chainConfig.ChainID), zap.Any("balance", balance))
		}

		inventory[chainID].CurrentBalance = balance
	}

	pendingSettlements, err := ordersettler.DetectPendingSettlements(ctx, cctpClientManager)
	if err != nil {
		return nil, fmt.Errorf("detecting pending settlements: %w", err)
	}

	for _, settlement := range pendingSettlements {
		inv := inventory[settlement.SourceChainID]
		inv.PendingSettlements.Add(inv.PendingSettlements, settlement.Amount)
		inv.GrossProfit.Add(inv.GrossProfit, settlement.Profit)
	}

	for _, inv := range inventory {
		total := new(big.Int).Set(inv.CurrentBalance)
		total.Add(total, inv.PendingSettlements)
		inv.TotalPosition = total
	}

	return inventory, nil
}

func init() {
	rootCmd.AddCommand(inventoryCmd)
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

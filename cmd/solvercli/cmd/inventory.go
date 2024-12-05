package cmd

import (
	"fmt"
	"math/big"

	"github.com/skip-mev/go-fast-solver/ordersettler"
	"github.com/skip-mev/go-fast-solver/shared/lmt"
	"github.com/spf13/cobra"
	"go.uber.org/zap"
)

var inventoryCmd = &cobra.Command{
	Use:   "inventory",
	Short: "Show complete solver inventory including balances, settlements, and rebalances",
	Run: func(cmd *cobra.Command, args []string) {
		ctx := setupContext(cmd)

		database, err := setupDatabase(ctx, cmd)
		if err != nil {
			lmt.Logger(ctx).Fatal("Failed to setup database", zap.Error(err))
		}

		_, cctpClientManager := setupClients(ctx, cmd)

		pendingSettlements, err := ordersettler.DetectPendingSettlements(ctx, cctpClientManager, nil)
		if err != nil {
			lmt.Logger(ctx).Fatal("Failed to get pending settlements", zap.Error(err))
		}

		pendingRebalances, err := database.GetAllPendingRebalanceTransfers(ctx)
		if err != nil {
			lmt.Logger(ctx).Fatal("Failed to get pending rebalances", zap.Error(err))
		}

		totalPendingSettlements := new(big.Int)
		totalPendingRebalances := new(big.Int)
		totalUSDCPosition := new(big.Int)
		usdcBalances, gasBalances, customBalances, totalAvaialbleUSDCBalance, totalCustomAssetsUSDValue, err := getBalances(ctx, cmd)
		if err != nil {
			lmt.Logger(ctx).Fatal("Failed to get existing balances", zap.Error(err))
		}

		fmt.Println("\nComplete Solver Inventory:")
		fmt.Println("-------------------------")

		fmt.Println("\nOn-Chain Balances:")
		fmt.Println("-----------------")
		for chainID, usdc := range usdcBalances {
			gas := gasBalances[chainID]
			fmt.Printf("\nChain: %s\n", chainID)
			fmt.Printf("  USDC Balance: %s USDC\n", normalizeBalance(usdc.Balance, CCTP_TOKEN_DECIMALS))
			fmt.Printf("  Gas Balance: %s %s\n", normalizeBalance(gas.Balance, gas.Decimals), gas.Symbol)

			if gas.Balance.Cmp(gas.CriticalThreshold) < 0 {
				fmt.Printf("  ⚠️  Gas balance below critical threshold!\n")
			} else if gas.Balance.Cmp(gas.WarningThreshold) < 0 {
				fmt.Printf("  ⚠️  Gas balance below warning threshold\n")
			}
		}

		fmt.Println("\nPending Settlements:")
		fmt.Println("-------------------")
		for _, settlement := range pendingSettlements {
			fmt.Printf("\nFrom %s to %s:\n", settlement.SourceChainID, settlement.DestinationChainID)
			fmt.Printf("  Amount: %s USDC\n", normalizeBalance(settlement.Amount, CCTP_TOKEN_DECIMALS))
			totalPendingSettlements.Add(totalPendingSettlements, settlement.Amount)
		}

		fmt.Println("\nPending Rebalance Transfers:")
		fmt.Println("--------------------------")
		for _, transfer := range pendingRebalances {
			amount, _ := new(big.Int).SetString(transfer.Amount, 10)
			fmt.Printf("\nFrom %s to %s:\n", transfer.SourceChainID, transfer.DestinationChainID)
			fmt.Printf("  Amount: %s USDC\n", normalizeBalance(amount, CCTP_TOKEN_DECIMALS))
			fmt.Printf("  Tx Hash: %s\n", transfer.TxHash)
			totalPendingRebalances.Add(totalPendingRebalances, amount)
		}

		totalUSDCPosition.Add(totalUSDCPosition, totalAvaialbleUSDCBalance)
		totalUSDCPosition.Add(totalUSDCPosition, totalPendingSettlements)
		totalUSDCPosition.Add(totalUSDCPosition, totalPendingRebalances)

		fmt.Printf("\nTotals Across All Chains:")
		fmt.Printf("\n------------------------\n")
		fmt.Printf("  Available USDC Inventory: %s USDC\n", normalizeBalance(totalAvaialbleUSDCBalance, CCTP_TOKEN_DECIMALS))
		fmt.Printf("  Pending Settlements: %s USDC\n", normalizeBalance(totalPendingSettlements, CCTP_TOKEN_DECIMALS))
		fmt.Printf("  Pending Rebalances: %s USDC\n", normalizeBalance(totalPendingRebalances, CCTP_TOKEN_DECIMALS))
		fmt.Printf("  Total USDC Position: %s USDC\n", normalizeBalance(totalUSDCPosition, CCTP_TOKEN_DECIMALS))

		if len(customBalances) > 0 {
			fmt.Printf("Total Custom Assets Value: %.2f USD\n", totalCustomAssetsUSDValue)
		}

	},
}

type ChainInventory struct {
	CurrentBalance     *big.Int
	PendingSettlements *big.Int
	TotalPosition      *big.Int
	GasBalance         *big.Int
	GasSymbol          string
	GasDecimals        uint8
}

func init() {
	rootCmd.AddCommand(inventoryCmd)
	inventoryCmd.Flags().String("custom-assets", "", "JSON map of chain IDs to denom arrays, e.g. '{\"osmosis\":[\"uosmo\",\"uion\"],\"celestia\":[\"utia\"]}'")
}

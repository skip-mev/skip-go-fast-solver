package cmd

import (
	"fmt"
	"math/big"

	"github.com/skip-mev/go-fast-solver/shared/lmt"
	"github.com/spf13/cobra"
	"go.uber.org/zap"
)

var rebalancesCmd = &cobra.Command{
	Use:   "rebalances",
	Short: "Show pending rebalance transfers across chains",
	Run: func(cmd *cobra.Command, args []string) {
		ctx := setupContext(cmd)

		database, err := setupDatabase(ctx, cmd)
		if err != nil {
			lmt.Logger(ctx).Fatal("Failed to setup database", zap.Error(err))
		}

		transfers, err := database.GetAllPendingRebalanceTransfers(ctx)
		if err != nil {
			lmt.Logger(ctx).Fatal("Failed to get pending rebalances", zap.Error(err))
		}

		fmt.Println("\nPending Rebalance Transfers:")
		fmt.Println("--------------------------")

		totalRebalancing := new(big.Int)
		for _, transfer := range transfers {
			transferAmount, ok := new(big.Int).SetString(transfer.Amount, 10)
			if !ok {
				lmt.Logger(ctx).Fatal("Failed to get transfer amount big.Int", zap.Error(err))
			}
			fmt.Printf("\nFrom %s to %s:\n", transfer.SourceChainID, transfer.DestinationChainID)
			fmt.Printf("  Amount: %s USDC\n", normalizeBalance(transferAmount, CCTP_TOKEN_DECIMALS))
			fmt.Printf("  Tx Hash: %s\n", transfer.TxHash)
			amount, _ := new(big.Int).SetString(transfer.Amount, 10)
			totalRebalancing.Add(totalRebalancing, amount)
		}

		fmt.Printf("\nTotal Pending Rebalances: %s USDC\n", normalizeBalance(totalRebalancing, CCTP_TOKEN_DECIMALS))
	},
}

func init() {
	rootCmd.AddCommand(rebalancesCmd)
}

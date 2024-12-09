package cmd

import (
	"fmt"
	"math/big"

	"github.com/skip-mev/go-fast-solver/ordersettler"
	"github.com/skip-mev/go-fast-solver/shared/clientmanager"
	"github.com/skip-mev/go-fast-solver/shared/keys"
	"github.com/skip-mev/go-fast-solver/shared/lmt"
	"github.com/skip-mev/go-fast-solver/shared/txexecutor/cosmos"
	"github.com/spf13/cobra"
	"go.uber.org/zap"
)

type chainPair struct {
	source      string
	destination string
}

type groupedSettlement struct {
	amount *big.Int
	count  int
}

var settlementsCmd = &cobra.Command{
	Use:     "settlements",
	Short:   "Show pending settlement amounts across chains",
	Long:    "Show pending settlement amounts across chains",
	Example: "solver settlements",
	Run: func(cmd *cobra.Command, args []string) {
		ctx := setupContext(cmd)
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
		cctpClientManager := clientmanager.NewClientManager(keyStore, cosmosTxExecutor)

		pendingSettlements, err := ordersettler.DetectPendingSettlements(ctx, cctpClientManager, nil)
		if err != nil {
			lmt.Logger(ctx).Fatal("Failed to get pending settlements", zap.Error(err))
		}

		groupedSettlements := make(map[chainPair]*groupedSettlement)
		for _, settlement := range pendingSettlements {
			pair := chainPair{
				source:      settlement.SourceChainID,
				destination: settlement.DestinationChainID,
			}

			if _, exists := groupedSettlements[pair]; !exists {
				groupedSettlements[pair] = &groupedSettlement{
					amount: new(big.Int),
					count:  0,
				}
			}

			groupedSettlements[pair].amount.Add(groupedSettlements[pair].amount, settlement.Amount)
			groupedSettlements[pair].count++
		}

		fmt.Println("\nPending Settlements:")
		fmt.Println("-------------------")

		totalPending := new(big.Int)
		for pair, settlement := range groupedSettlements {
			fmt.Printf("\nFrom %s to %s:\n", pair.source, pair.destination)
			fmt.Printf("  Amount: %s USDC (%d settlements)\n",
				normalizeBalance(settlement.amount, CCTP_TOKEN_DECIMALS),
				settlement.count)
			totalPending.Add(totalPending, settlement.amount)
		}

		fmt.Printf("\nTotal Pending Settlements: %s USDC\n", normalizeBalance(totalPending, CCTP_TOKEN_DECIMALS))
	},
}

func init() {
	rootCmd.AddCommand(settlementsCmd)
	settlementsCmd.Flags().String("evm-address", "", "Optional EVM address to check settlements for instead of config address")
	settlementsCmd.Flags().String("osmosis-address", "", "Optional Osmosis address to check settlements for instead of config address")
}

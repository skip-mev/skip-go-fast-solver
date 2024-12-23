package cmd

import (
	"fmt"
	dbtypes "github.com/skip-mev/go-fast-solver/db"
	"math/big"

	"github.com/skip-mev/go-fast-solver/db/connect"
	"github.com/skip-mev/go-fast-solver/db/gen/db"
	"github.com/skip-mev/go-fast-solver/shared/lmt"
	"github.com/spf13/cobra"
	"go.uber.org/zap"
	"golang.org/x/net/context"
)

var profitCmd = &cobra.Command{
	Use:   "profit",
	Short: "Calculate total solver profit from settlements minus transaction costs",
	Long: `Calculate the total profit made by the solver by summing all settlement profits
and subtracting transaction costs. This includes costs from fill transactions,
settlement transactions, and rebalancing transactions.`,
	Example: `solver profit`,
	Run:     calculateProfit,
}

func calculateProfit(cmd *cobra.Command, args []string) {
	ctx := context.Background()
	lmt.ConfigureLogger()
	ctx = lmt.LoggerContext(ctx)

	dbPath, err := cmd.Flags().GetString("sqlite-db-path")
	if err != nil {
		lmt.Logger(ctx).Error("Failed to get sqlite-db-path flag", zap.Error(err))
		return
	}

	migrationsPath, err := cmd.Flags().GetString("migrations-path")
	if err != nil {
		lmt.Logger(ctx).Error("Failed to get migrations-path flag", zap.Error(err))
		return
	}

	dbConn, err := connect.ConnectAndMigrate(ctx, dbPath, migrationsPath)
	if err != nil {
		lmt.Logger(ctx).Error("Failed to connect to database", zap.Error(err))
		return
	}
	defer dbConn.Close()

	queries := db.New(dbConn)

	settlements, err := queries.GetAllOrderSettlementsWithSettlementStatus(ctx, dbtypes.SettlementStatusComplete)
	if err != nil {
		lmt.Logger(ctx).Error("Failed to get completed settlements", zap.Error(err))
		return
	}

	totalProfit := big.NewInt(0)
	for _, settlement := range settlements {
		profit, ok := new(big.Int).SetString(settlement.Profit, 10)
		if !ok {
			lmt.Logger(ctx).Error("Failed to parse settlement profit",
				zap.String("profit", settlement.Profit),
				zap.String("orderID", settlement.OrderID))
			continue
		}
		totalProfit = totalProfit.Add(totalProfit, profit)
	}

	submittedTxs, err := queries.GetAllSubmittedTxs(ctx)
	if err != nil {
		lmt.Logger(ctx).Error("Failed to get submitted transactions", zap.Error(err))
		return
	}

	totalTxCosts := big.NewInt(0)
	for _, tx := range submittedTxs {
		if tx.TxCostUusdc.Valid {
			cost, ok := new(big.Int).SetString(tx.TxCostUusdc.String, 10)
			if !ok {
				lmt.Logger(ctx).Error("Failed to parse transaction cost",
					zap.String("txHash", tx.TxHash),
					zap.String("cost", tx.TxCostUusdc.String))
				continue
			}
			totalTxCosts = totalTxCosts.Add(totalTxCosts, cost)
		}
	}

	netProfit := new(big.Int).Sub(totalProfit, totalTxCosts)
	usdcMultiplier := new(big.Float).SetInt64(1000000)
	netProfitUsdc := new(big.Float).Quo(
		new(big.Float).SetInt(netProfit),
		usdcMultiplier,
	)

	fmt.Printf("\nSolver Profit Summary:\n")
	fmt.Printf("Total Settlement Profit: %s USDC\n",
		new(big.Float).Quo(new(big.Float).SetInt(totalProfit), usdcMultiplier))
	fmt.Printf("Total Transaction Costs: %s USDC\n",
		new(big.Float).Quo(new(big.Float).SetInt(totalTxCosts), usdcMultiplier))
	fmt.Printf("Net Profit: %s USDC\n", netProfitUsdc.Text('f', 6))
}

func init() {
	rootCmd.AddCommand(profitCmd)
}

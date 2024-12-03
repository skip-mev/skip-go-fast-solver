package cmd

import (
	"context"
	"fmt"
	"math/big"
	"os/signal"
	"syscall"

	"github.com/skip-mev/go-fast-solver/db/connect"
	"github.com/skip-mev/go-fast-solver/db/gen/db"
	"github.com/skip-mev/go-fast-solver/shared/clients/skipgo"
	"github.com/skip-mev/go-fast-solver/shared/config"
	"github.com/skip-mev/go-fast-solver/shared/evmrpc"
	"github.com/skip-mev/go-fast-solver/shared/lmt"
	"github.com/spf13/cobra"
	"go.uber.org/zap"
)

var inventoryCmd = &cobra.Command{
	Use:   "inventory",
	Short: "Show current solver inventory across all chains",
	Run: func(cmd *cobra.Command, args []string) {
		configPath, err := cmd.Flags().GetString("config")
		if err != nil {
			lmt.Logger(context.Background()).Fatal("Failed to get config path", zap.Error(err))
		}

		cfg, err := config.LoadConfig(configPath)
		if err != nil {
			lmt.Logger(context.Background()).Fatal("Failed to load config", zap.Error(err))
		}

		fmt.Println("loaded config:")

		ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
		defer cancel()

		ctx = config.ConfigReaderContext(ctx, config.NewConfigReader(cfg))

		sqliteDBPath, err := cmd.Flags().GetString("sqlite-db-path")
		if err != nil {
			lmt.Logger(ctx).Error("Error reading keys command line argument", zap.Error(err))
			return
		}
		fmt.Println("sqlllittttt:")

		migrationsPath, err := cmd.Flags().GetString("migrations-path")
		if err != nil {
			lmt.Logger(ctx).Error("Error reading migrations command line argument", zap.Error(err))
			return
		}
		fmt.Println("miiiiigrations:")

		dbConn, err := connect.ConnectAndMigrate(ctx, sqliteDBPath, migrationsPath)
		if err != nil {
			lmt.Logger(ctx).Fatal("Unable to connect to db", zap.Error(err))
		}
		defer dbConn.Close()
		queries := db.New(dbConn)
		fmt.Println("created db:")

		skipgoClient, err := skipgo.NewSkipGoClient("https://api.skip.build")
		if err != nil {
			lmt.Logger(ctx).Fatal("Failed to create skip go client", zap.Error(err))
		}

		evmClientManager := evmrpc.NewEVMRPCClientManager()

		fmt.Println("client manager created:")

		inventory, err := getInventory(ctx, queries, skipgoClient, evmClientManager)
		if err != nil {
			fmt.Println("ERROR")
			fmt.Println(err)
			lmt.Logger(ctx).Fatal("Failed to get inventory", zap.Error(err))
		}

		fmt.Println("\nSolver Inventory Summary:")
		fmt.Println("------------------------")

		for chainID, inv := range inventory {
			fmt.Printf("\nChain: %s\n", chainID)
			fmt.Printf("  Current Balance: %s USDC\n", inv.CurrentBalance)
			fmt.Printf("  Pending Settlements: %s USDC\n", inv.PendingSettlements)
			fmt.Printf("  Incoming Rebalances: %s USDC\n", inv.IncomingRebalances)
			fmt.Printf("  Outgoing Rebalances: %s USDC\n", inv.OutgoingRebalances)
			fmt.Printf("  Total Position: %s USDC\n", inv.TotalPosition)
		}
	},
}

type ChainInventory struct {
	CurrentBalance     *big.Int
	PendingSettlements *big.Int
	IncomingRebalances *big.Int
	OutgoingRebalances *big.Int
	TotalPosition      *big.Int
}

func getInventory(ctx context.Context, queries *db.Queries, skipgoClient skipgo.SkipGoClient, evmClientManager evmrpc.EVMRPCClientManager) (map[string]ChainInventory, error) {
	inventory := make(map[string]ChainInventory)

	// Get all chain configs
	chains := config.GetConfigReader(ctx).Config().Chains

	// Initialize inventory for each chain
	for _, chain := range chains {
		inventory[chain.ChainID] = ChainInventory{
			CurrentBalance:     big.NewInt(0),
			PendingSettlements: big.NewInt(0),
			IncomingRebalances: big.NewInt(0),
			OutgoingRebalances: big.NewInt(0),
			TotalPosition:      big.NewInt(0),
		}
	}

	// Get current balances
	for chainID, inv := range inventory {
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

		case config.ChainType_COSMOS:
			balanceStr, err := skipgoClient.Balance(ctx, chainID, chainConfig.SolverAddress, chainConfig.USDCDenom)
			if err != nil {
				return nil, fmt.Errorf("getting balance for %s: %w", chainID, err)
			}
			balance, _ = new(big.Int).SetString(balanceStr, 10)
		}

		inv.CurrentBalance = balance
		inventory[chainID] = inv
	}

	// Get pending settlements
	pendingSettlements, err := queries.GetAllOrderSettlementsWithSettlementStatus(ctx, "PENDING")
	if err != nil {
		return nil, fmt.Errorf("getting pending settlements: %w", err)
	}

	for _, settlement := range pendingSettlements {
		amount, ok := new(big.Int).SetString(settlement.Amount, 10)
		if !ok {
			return nil, fmt.Errorf("invalid amount in settlement: %s", settlement.Amount)
		}

		inv := inventory[settlement.SourceChainID]
		inv.PendingSettlements.Add(inv.PendingSettlements, amount)
		inventory[settlement.SourceChainID] = inv
	}

	// Get pending rebalance transfers
	pendingTransfers, err := queries.GetAllPendingRebalanceTransfers(ctx)
	if err != nil {
		return nil, fmt.Errorf("getting pending rebalance transfers: %w", err)
	}

	for _, transfer := range pendingTransfers {
		amount, ok := new(big.Int).SetString(transfer.Amount, 10)
		if !ok {
			return nil, fmt.Errorf("invalid amount in transfer: %s", transfer.Amount)
		}

		// Add to destination chain's incoming
		destInv := inventory[transfer.DestinationChainID]
		destInv.IncomingRebalances.Add(destInv.IncomingRebalances, amount)
		inventory[transfer.DestinationChainID] = destInv

		// Add to source chain's outgoing
		sourceInv := inventory[transfer.SourceChainID]
		sourceInv.OutgoingRebalances.Add(sourceInv.OutgoingRebalances, amount)
		inventory[transfer.SourceChainID] = sourceInv
	}

	// Calculate total positions
	for chainID, inv := range inventory {
		total := new(big.Int).Set(inv.CurrentBalance)
		total.Add(total, inv.IncomingRebalances)
		total.Sub(total, inv.OutgoingRebalances)
		total.Add(total, inv.PendingSettlements)
		inv.TotalPosition = total
		inventory[chainID] = inv
	}

	return inventory, nil
}

func init() {
	rootCmd.AddCommand(inventoryCmd)
	inventoryCmd.Flags().String("sqlite-db-path", "./solver.db", "path to sqlite db file")
	inventoryCmd.Flags().String("migrations-path", "./db/migrations", "path to db migrations directory")
}

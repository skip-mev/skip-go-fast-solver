package cmd

import (
	"fmt"
	"os/signal"
	"slices"
	"strconv"
	"syscall"
	"time"

	"github.com/skip-mev/go-fast-solver/db/gen/db"
	"github.com/skip-mev/go-fast-solver/ordersettler/types"
	"github.com/skip-mev/go-fast-solver/shared/clientmanager"
	"github.com/skip-mev/go-fast-solver/shared/config"
	"github.com/skip-mev/go-fast-solver/shared/contracts/fast_transfer_gateway"
	"github.com/skip-mev/go-fast-solver/shared/keys"
	"github.com/skip-mev/go-fast-solver/shared/lmt"
	"github.com/skip-mev/go-fast-solver/shared/txexecutor/cosmos"
	"github.com/spf13/cobra"
	"go.uber.org/zap"
	"golang.org/x/net/context"
)

var orders = []string{
	"405a634a74c386ce00e9416b01a4786939720c46e4906e924089a6c5e0bdb8ce",
	"9a5b13ccd1536bd3bfa23b9da09f33a751f21e5777ec05c43c6c6549cef99a26",
	"2bfc4c87599f2a72f22a1842f4f175df817acf8c0b864c4e101ea575fbb5bdd8",
	"c347d99507a22f1fb3edb4b24abe99fb637768ad7ef43cbf8ae75d92e3f452de",
	"4a52a4ed1c4c71193a4c2961c325833efe874de0fea4c3d1fc9249996fa7ce83",
	"5010ef3e16aac1ef7ef7140e56e15e41b0e031ddb129f05609561d90b98aef63",
	"00f0c7f98dc75942c682ea51a37f18f98d32d78d80bf9027f32343cacdb9675d",
}

var settleCmd = &cobra.Command{
	Use:     "settle-orders",
	Short:   "Settle pending order batches",
	Long:    `Settle all pending order batches immediately without any threshold checks (ignoring configured BatchUUSDCSettleUpThreshold).`,
	Example: `solver settle-orders`,
	Run:     settleOrders,
}

func init() {
	rootCmd.AddCommand(settleCmd)
}

func settleOrders(cmd *cobra.Command, args []string) {
	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	lmt.ConfigureLogger()
	ctx = lmt.LoggerContext(ctx)

	keyStoreType, err := cmd.Flags().GetString("key-store-type")
	if err != nil {
		lmt.Logger(ctx).Error("Failed to get key-store-type", zap.Error(err))
		return
	}

	keysPath, err := cmd.Flags().GetString("keys")
	if err != nil {
		lmt.Logger(ctx).Error("Failed to get keys path", zap.Error(err))
		return
	}

	configPath, err := cmd.Flags().GetString("config")
	if err != nil {
		lmt.Logger(ctx).Error("Failed to get config path", zap.Error(err))
		return
	}

	cfg, err := config.LoadConfig(configPath)
	if err != nil {
		lmt.Logger(ctx).Error("Unable to load config", zap.Error(err))
		return
	}

	ctx = config.ConfigReaderContext(ctx, config.NewConfigReader(cfg))

	keyStore, err := keys.GetKeyStore(keyStoreType, keys.GetKeyStoreOpts{KeyFilePath: keysPath})
	if err != nil {
		lmt.Logger(ctx).Fatal("Unable to load keystore", zap.Error(err))
	}

	cosmosTxExecutor := cosmos.DefaultSerializedCosmosTxExecutor()
	clientManager := clientmanager.NewClientManager(keyStore, cosmosTxExecutor)

	chains, err := config.GetConfigReader(ctx).GetAllChainConfigsOfType(config.ChainType_COSMOS)
	if err != nil {
		lmt.Logger(ctx).Error("error getting Cosmos chains", zap.Error(err))
		return
	}

	var pendingSettlements []db.OrderSettlement
	for _, chain := range chains {
		if chain.FastTransferContractAddress == "" {
			continue
		}

		bridgeClient, err := clientManager.GetClient(ctx, chain.ChainID)
		if err != nil {
			lmt.Logger(ctx).Error("failed to get client",
				zap.String("chainID", chain.ChainID),
				zap.Error(err))
			continue
		}

		fills, err := bridgeClient.OrderFillsByFiller(ctx, chain.FastTransferContractAddress, chain.SolverAddress)
		if err != nil {
			lmt.Logger(ctx).Error("getting order fills",
				zap.String("chainID", chain.ChainID),
				zap.Error(err))
			continue
		}

		// For each fill, check if it needs settlement
		for _, fill := range fills {
			if !slices.Contains(orders, fill.OrderID) {
				continue
			}
			sourceChainID, err := config.GetConfigReader(ctx).GetChainIDByHyperlaneDomain(strconv.Itoa(int(fill.SourceDomain)))
			if err != nil {
				lmt.Logger(ctx).Error("failed to get source chain ID",
					zap.Uint32("domain", fill.SourceDomain),
					zap.Error(err))
				continue
			}

			sourceGatewayAddress, err := config.GetConfigReader(ctx).GetGatewayContractAddress(sourceChainID)
			if err != nil {
				lmt.Logger(ctx).Error("getting source gateway address",
					zap.String("chainID", sourceChainID),
					zap.Error(err))
				continue
			}

			sourceBridgeClient, err := clientManager.GetClient(ctx, sourceChainID)
			if err != nil {
				lmt.Logger(ctx).Error("getting source chain client",
					zap.String("chainID", sourceChainID),
					zap.Error(err))
				continue
			}

			status, err := sourceBridgeClient.OrderStatus(ctx, sourceGatewayAddress, fill.OrderID)
			if err != nil {
				lmt.Logger(ctx).Error("getting order status",
					zap.String("orderID", fill.OrderID),
					zap.Error(err))
				continue
			}

			if status != fast_transfer_gateway.OrderStatusUnfilled {
				continue
			}

			pendingSettlements = append(pendingSettlements, db.OrderSettlement{
				SourceChainID:                     sourceChainID,
				DestinationChainID:                chain.ChainID,
				SourceChainGatewayContractAddress: sourceGatewayAddress,
				OrderID:                           fill.OrderID,
			})
		}
	}

	if len(pendingSettlements) == 0 {
		fmt.Println("No pending settlement batches found")
		return
	}

	batches := types.IntoSettlementBatchesByChains(pendingSettlements)
	fmt.Printf("Found %d pending settlement batches\n", len(batches))

	for i, batch := range batches {
		destinationBridgeClient, err := clientManager.GetClient(ctx, batch.DestinationChainID())
		if err != nil {
			lmt.Logger(ctx).Error("getting destination bridge client", zap.Error(err))
			continue
		}

		txHash, _, err := destinationBridgeClient.InitiateBatchSettlement(ctx, batch)
		if err != nil {
			lmt.Logger(ctx).Error("initiating batch settlement", zap.Error(err))
			continue
		}

		fmt.Printf("Initiated settlement batch %d:\n", i+1)
		fmt.Printf("Source Chain: %s\n", batch.SourceChainID())
		fmt.Printf("Destination Chain: %s\n", batch.DestinationChainID())
		fmt.Printf("Number of Orders: %d\n", len(batch.OrderIDs()))
		fmt.Printf("Order IDs:\n")
		for _, orderID := range batch.OrderIDs() {
			fmt.Printf("  - %s\n", orderID)
		}
		fmt.Printf("Transaction Hash: %s\n", txHash)

		// Sleep between batches to avoid sequence number issues
		if i < len(batches)-1 {
			fmt.Println("Waiting 10 seconds before next batch...")
			time.Sleep(10 * time.Second)
		}
	}
}

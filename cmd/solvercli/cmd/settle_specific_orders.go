package cmd

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/skip-mev/go-fast-solver/db/gen/db"
	"github.com/skip-mev/go-fast-solver/ordersettler/types"
	"github.com/skip-mev/go-fast-solver/shared/config"
	"github.com/skip-mev/go-fast-solver/shared/contracts/fast_transfer_gateway"
	"github.com/skip-mev/go-fast-solver/shared/lmt"
	"github.com/spf13/cobra"
	"go.uber.org/zap"
)

var settleSpecificOrdersCmd = &cobra.Command{
	Use:     "settle-specific-orders",
	Short:   "Settle specific orders",
	Long:    `Initiate settlement for specific pending order immediately without any threshold checks (ignoring configured BatchUUSDCSettleUpThreshold).`,
	Example: `solver settle-specific-orders --order-ids orderid1,orderid2,orderid3 --chain-id osmosis-1`,
	Run:     settleSpecificOrders,
}

func init() {
	rootCmd.AddCommand(settleSpecificOrdersCmd)
	settleSpecificOrdersCmd.Flags().StringSlice("order-ids", nil, "list of order ids to settle")
	settleSpecificOrdersCmd.Flags().String("chain-id", "", "chain ids of orders to settle")

	requiredFlags := []string{"order-ids", "chain-id"}
	for _, flag := range requiredFlags {
		if err := settleSpecificOrdersCmd.MarkFlagRequired(flag); err != nil {
			panic(fmt.Sprintf("failed to mark %s flag as required: %v", flag, err))
		}
	}
}

func settleSpecificOrders(cmd *cobra.Command, args []string) {
	ctx := setupContext(cmd)
	_, clientManager := setupClients(ctx, cmd)

	chainID, err := cmd.Flags().GetString("chain-id")
	if err != nil {
		lmt.Logger(ctx).Error("error getting chain id flag", zap.Error(err))
		return
	}
	if chainID != "osmosis-1" {
		lmt.Logger(ctx).Error(fmt.Sprintf("chain id must be osmosis-1, instead got %s", chainID), zap.Error(err))
		return
	}

	orderIDs, err := cmd.Flags().GetStringSlice("order-ids")
	if err != nil {
		lmt.Logger(ctx).Error("error getting order ids from flags", zap.Error(err))
		return
	}
	if len(orderIDs) == 0 {
		lmt.Logger(ctx).Error("empty order ids list")
		return
	}

	chain, err := config.GetConfigReader(ctx).GetChainConfig(chainID)
	if err != nil {
		lmt.Logger(ctx).Error("error getting chain config by chain id", zap.String("chainID", chainID), zap.Error(err))
		return
	}

	var pendingSettlements []db.OrderSettlement
	for _, orderID := range orderIDs {
		if chain.FastTransferContractAddress == "" {
			continue
		}

		bridgeClient, err := clientManager.GetClient(ctx, chain.ChainID)
		if err != nil {
			lmt.Logger(ctx).Error("failed to get client... ignoring order", zap.String("chainID", chain.ChainID), zap.Error(err))
			continue
		}

		fill, _, err := bridgeClient.QueryOrderFillEvent(ctx, chain.FastTransferContractAddress, strings.TrimPrefix(orderID, "0x"))
		if err != nil {
			lmt.Logger(ctx).Error("error getting order fill for order... ignoring order", zap.String("orderID", orderID), zap.String("gateway", chain.FastTransferContractAddress), zap.String("chainID", chain.ChainID), zap.Error(err))
			continue
		}

		// For each order fill, check if it needs settlement
		sourceChainID, err := config.GetConfigReader(ctx).GetChainIDByHyperlaneDomain(strconv.Itoa(int(fill.SourceDomain)))
		if err != nil {
			lmt.Logger(ctx).Error("failed to get source chain ID for order fill source domain... ignoring order", zap.Uint32("domain", fill.SourceDomain), zap.Error(err))
			continue
		}

		sourceGatewayAddress, err := config.GetConfigReader(ctx).GetGatewayContractAddress(sourceChainID)
		if err != nil {
			lmt.Logger(ctx).Error("getting source gateway address... ignoring order", zap.String("chainID", sourceChainID), zap.Error(err))
			continue
		}

		sourceBridgeClient, err := clientManager.GetClient(ctx, sourceChainID)
		if err != nil {
			lmt.Logger(ctx).Error("getting source chain client... ignoring order", zap.String("chainID", sourceChainID), zap.Error(err))
			continue
		}

		status, err := sourceBridgeClient.OrderStatus(ctx, sourceGatewayAddress, orderID)
		if err != nil {
			lmt.Logger(ctx).Error("getting order status... ingoring order", zap.String("orderID", orderID), zap.Error(err))
			continue
		}

		if status != fast_transfer_gateway.OrderStatusUnfilled {
			lmt.Logger(ctx).Error("order is not unfilled... ignoring order", zap.String("orderID", orderID), zap.Error(err))
			continue
		}

		pendingSettlements = append(pendingSettlements, db.OrderSettlement{
			SourceChainID:                     sourceChainID,
			DestinationChainID:                chain.ChainID,
			SourceChainGatewayContractAddress: sourceGatewayAddress,
			OrderID:                           orderID,
		})
	}

	if len(pendingSettlements) == 0 {
		fmt.Println("No valid pending settlements found")
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
		fmt.Printf("Transaction Hash: %s\n", txHash)
	}
}

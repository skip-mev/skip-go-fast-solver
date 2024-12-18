package ordersettler

import (
	"context"
	"fmt"
	"math/big"
	"strconv"

	"github.com/skip-mev/go-fast-solver/shared/bridges/cctp"
	"github.com/skip-mev/go-fast-solver/shared/contracts/fast_transfer_gateway"
	"github.com/skip-mev/go-fast-solver/shared/lmt"
	"go.uber.org/zap"

	"github.com/skip-mev/go-fast-solver/shared/clientmanager"
	"github.com/skip-mev/go-fast-solver/shared/config"
)

type PendingSettlement struct {
	SourceChainID      string
	DestinationChainID string
	OrderID            string
	Amount             *big.Int
	Profit             *big.Int
}

// DetectPendingSettlements scans all chains for pending settlements that need to be processed
func DetectPendingSettlements(
	ctx context.Context,
	clientManager *clientmanager.ClientManager,
	ordersSeen map[string]bool,
) ([]PendingSettlement, error) {
	var pendingSettlements []PendingSettlement

	cosmosChains, err := config.GetConfigReader(ctx).GetAllChainConfigsOfType(config.ChainType_COSMOS)
	if err != nil {
		return nil, fmt.Errorf("error getting Cosmos chains: %w", err)
	}

	for _, chain := range cosmosChains {
		bridgeClient, err := clientManager.GetClient(ctx, chain.ChainID)
		if err != nil {
			return nil, fmt.Errorf("failed to get client: %w", err)
		}

		fills, err := bridgeClient.OrderFillsByFiller(ctx, chain.FastTransferContractAddress, chain.SolverAddress)
		if err != nil {
			return nil, fmt.Errorf("getting order fills: %w", err)
		}

		for _, fill := range fills {
			if ordersSeen != nil && ordersSeen[fill.OrderID] {
				continue
			}

			sourceChainID, err := config.GetConfigReader(ctx).GetChainIDByHyperlaneDomain(strconv.Itoa(int(fill.SourceDomain)))
			if err != nil {
				lmt.Logger(ctx).Warn(
					"failed to get source chain ID by hyperlane domain. skipping order settlement. it may be unsettled.",
					zap.Uint32("hyperlaneDomain", fill.SourceDomain),
					zap.String("orderID", fill.OrderID),
					zap.Error(err),
				)
				ordersSeen[fill.OrderID] = true
				continue
			}

			sourceGatewayAddress, err := config.GetConfigReader(ctx).GetGatewayContractAddress(sourceChainID)
			if err != nil {
				return nil, fmt.Errorf("getting source gateway address: %w", err)
			}

			sourceBridgeClient, err := clientManager.GetClient(ctx, sourceChainID)
			if err != nil {
				return nil, fmt.Errorf("getting client for chainID %s: %w", sourceChainID, err)
			}

			height, err := sourceBridgeClient.BlockHeight(ctx)
			if err != nil {
				return nil, fmt.Errorf("fetching current block height on chain %s: %w", sourceChainID, err)
			}

			// ensure order exists on source chain
			exists, amount, err := sourceBridgeClient.OrderExists(ctx, sourceGatewayAddress, fill.OrderID, big.NewInt(int64(height)))
			if err != nil {
				return nil, fmt.Errorf("checking if order %s exists on chainID %s: %w", fill.OrderID, sourceChainID, err)
			}
			if !exists {
				ordersSeen[fill.OrderID] = true
				continue
			}

			// ensure order is not already filled (an order is only marked as
			// filled on the source chain once it is settled)
			status, err := sourceBridgeClient.OrderStatus(ctx, sourceGatewayAddress, fill.OrderID)
			if err != nil {
				return nil, fmt.Errorf("getting order %s status on chainID %s: %w", fill.OrderID, sourceChainID, err)
			}
			if status != fast_transfer_gateway.OrderStatusUnfilled {
				ordersSeen[fill.OrderID] = true
				continue
			}

			orderFillEvent, _, err := bridgeClient.QueryOrderFillEvent(ctx, chain.FastTransferContractAddress, fill.OrderID)
			if err != nil {
				if _, ok := err.(cctp.ErrOrderFillEventNotFound); ok {
					lmt.Logger(ctx).Warn(
						"failed to find order fill event",
						zap.String("fastTransferGatewayAddress", chain.FastTransferContractAddress),
						zap.String("orderID", fill.OrderID),
						zap.String("chainID", chain.ChainID),
						zap.Error(err),
					)
					continue
				}
				return nil, fmt.Errorf("querying for order fill event on destination chain at address %s for order id %s: %w", chain.FastTransferContractAddress, fill.OrderID, err)
			}
			profit := new(big.Int).Sub(amount, orderFillEvent.FillAmount)

			pendingSettlements = append(pendingSettlements, PendingSettlement{
				SourceChainID:      sourceChainID,
				DestinationChainID: chain.ChainID,
				OrderID:            fill.OrderID,
				Amount:             amount,
				Profit:             profit,
			})
		}
	}

	return pendingSettlements, nil
}

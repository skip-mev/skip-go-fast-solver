package ordersettler

import (
	"context"
	"fmt"
	"github.com/skip-mev/go-fast-solver/shared/contracts/fast_transfer_gateway"
	"math/big"
	"strconv"

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

	var chains []config.ChainConfig
	for _, chain := range cosmosChains {
		if chain.FastTransferContractAddress != "" {
			chains = append(chains, chain)
		}
	}

	for _, chain := range chains {
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

			exists, amount, err := sourceBridgeClient.OrderExists(ctx, sourceGatewayAddress, fill.OrderID, big.NewInt(int64(height)))
			if err != nil {
				return nil, fmt.Errorf("checking if order %s exists on chainID %s: %w", fill.OrderID, sourceChainID, err)
			}
			if !exists {
				continue
			}

			status, err := sourceBridgeClient.OrderStatus(ctx, sourceGatewayAddress, fill.OrderID)
			if err != nil {
				return nil, fmt.Errorf("getting order %s status on chainID %s: %w", fill.OrderID, sourceChainID, err)
			}
			if status != fast_transfer_gateway.OrderStatusUnfilled {
				continue
			}

			orderFillEvent, _, err := bridgeClient.QueryOrderFillEvent(ctx, chain.FastTransferContractAddress, fill.OrderID)
			if err != nil {
				continue
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

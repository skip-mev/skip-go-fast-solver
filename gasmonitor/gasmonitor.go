package gasmonitor

import (
	"context"
	"fmt"
	"github.com/skip-mev/go-fast-solver/shared/bridges/cctp"
	"github.com/skip-mev/go-fast-solver/shared/clientmanager"
	"github.com/skip-mev/go-fast-solver/shared/metrics"
	"go.uber.org/zap"
	"time"

	"github.com/skip-mev/go-fast-solver/shared/config"
	"github.com/skip-mev/go-fast-solver/shared/lmt"
)

type GasMonitor struct {
	clientManager *clientmanager.ClientManager
}

func NewGasMonitor(clientManager *clientmanager.ClientManager) *GasMonitor {
	return &GasMonitor{
		clientManager: clientManager,
	}
}

func (gm *GasMonitor) Start(ctx context.Context) error {
	lmt.Logger(ctx).Info("Starting gas monitor")
	var chains []config.ChainConfig
	evmChains, err := config.GetConfigReader(ctx).GetAllChainConfigsOfType(config.ChainType_EVM)
	if err != nil {
		return fmt.Errorf("error getting EVM chains: %w", err)
	}
	cosmosChains, err := config.GetConfigReader(ctx).GetAllChainConfigsOfType(config.ChainType_COSMOS)
	if err != nil {
		return fmt.Errorf("error getting cosmos chains: %w", err)
	}
	chains = append(chains, evmChains...)
	chains = append(chains, cosmosChains...)

	ticker := time.NewTicker(1 * time.Minute)
	for {
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
			for _, chain := range chains {
				client, err := gm.clientManager.GetClient(ctx, chain.ChainID)
				if err != nil {
					return err
				}
				err = monitorGasBalance(ctx, chain.ChainID, client)
				if err != nil {
					lmt.Logger(ctx).Error("failed to monitor gas balance", zap.String("chain_id", chain.ChainID), zap.Error(err))
				}
			}
		}
	}
}

// monitorGasBalance exports a metric indicating the current gas balance of the relayer signer and whether it is below alerting thresholds
func monitorGasBalance(ctx context.Context, chainID string, chainClient cctp.BridgeClient) error {
	balance, err := chainClient.SignerGasTokenBalance(ctx)
	if err != nil {
		lmt.Logger(ctx).Error("failed to get gas token balance", zap.Error(err), zap.String("chain_id", chainID))
		return err
	}

	chainConfig, err := config.GetConfigReader(ctx).GetChainConfig(chainID)
	if err != nil {
		return err
	}
	warningThreshold, criticalThreshold, err := config.GetConfigReader(ctx).GetGasAlertThresholds(chainID)
	if err != nil {
		return err
	}
	if balance == nil || warningThreshold == nil || criticalThreshold == nil {
		return fmt.Errorf("gas balance or alert thresholds are nil for chain %s", chainID)
	}
	if balance.Cmp(criticalThreshold) < 0 {
		lmt.Logger(ctx).Error("low balance", zap.String("balance", balance.String()), zap.String("chainID", chainID))
	}
	metrics.FromContext(ctx).SetGasBalance(chainID, chainConfig.ChainName, chainConfig.GasTokenSymbol, *balance, *warningThreshold, *criticalThreshold, chainConfig.GasTokenDecimals)
	return nil
}

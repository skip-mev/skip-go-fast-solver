package utils

import (
	"context"
	"fmt"
	"github.com/skip-mev/go-fast-solver/shared/bridges/cctp"
	"github.com/skip-mev/go-fast-solver/shared/config"
	"github.com/skip-mev/go-fast-solver/shared/lmt"
	"github.com/skip-mev/go-fast-solver/shared/metrics"
	"go.uber.org/zap"
)

// MonitorGasBalance exports a metric indicating the current gas balance of the relayer signer and whether it is below alerting thresholds
func MonitorGasBalance(ctx context.Context, chainID string, chainClient cctp.BridgeClient) error {
	balance, err := chainClient.GasTokenBalance(ctx)
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

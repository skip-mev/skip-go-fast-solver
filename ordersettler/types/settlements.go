package types

import (
	"context"
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/skip-mev/go-fast-solver/db/gen/db"
	"github.com/skip-mev/go-fast-solver/shared/config"
)

// SettlementBatch is a slice of OrderSettlement's that all share the same
// source chain and all share the same destination chain.
type SettlementBatch []db.OrderSettlement

// IntoSettlementBatches converts a slice of settlements that may be from
// different source chains and to different destination chains into a list of
// settlement batches where each settlement batch contains settlements from the
// same source chain and to the same destination chain.
func IntoSettlementBatches(settlements []db.OrderSettlement) ([]SettlementBatch, error) {
	type key struct {
		// ensure settlements are from the same source (if not then they would
		// have different repayment addresses and would be in a different
		// batch)
		sourceChainID string

		// ensure settlements are to the same destination (if not they would
		// have different semantics on how to submit the settlement and
		// therefore wouldnt be a batch)
		destChainID string
	}

	batchesSet := make(map[key]SettlementBatch)
	for _, settlement := range settlements {
		k := key{sourceChainID: settlement.SourceChainID, destChainID: settlement.DestinationChainID}
		batchesSet[k] = append(batchesSet[k], settlement)
	}

	var batches []SettlementBatch
	for _, batch := range batchesSet {
		batches = append(batches, batch)
	}
	return batches, nil
}

func (b SettlementBatch) OrderIDs() []string {
	var ids []string
	for _, settlement := range b {
		ids = append(ids, settlement.OrderID)
	}
	return ids
}

func (b SettlementBatch) SourceChainID() string {
	return b[0].SourceChainID
}

func (b SettlementBatch) DestinationChainID() string {
	return b[0].DestinationChainID
}

func (b SettlementBatch) SourceChainConfig(ctx context.Context) (config.ChainConfig, error) {
	sourceChainConfig, err := config.GetConfigReader(ctx).GetChainConfig(b.SourceChainID())
	if err != nil {
		return config.ChainConfig{}, fmt.Errorf("getting chain config for chain %s: %w", b.SourceChainID(), err)
	}
	return sourceChainConfig, nil
}

func (b SettlementBatch) DestinationChainConfig(ctx context.Context) (config.ChainConfig, error) {
	destinationChainConfig, err := config.GetConfigReader(ctx).GetChainConfig(b.DestinationChainID())
	if err != nil {
		return config.ChainConfig{}, fmt.Errorf("getting chain config for chain %s: %w", b.DestinationChainID(), err)
	}
	return destinationChainConfig, nil
}

func (b SettlementBatch) DestinationGatewayContractAddress(ctx context.Context) (string, error) {
	addr, err := config.GetConfigReader(ctx).GetGatewayContractAddress(b.DestinationChainID())
	if err != nil {
		return "", fmt.Errorf("getting gateway contract address for destination chain %s: %w", b.DestinationChainID(), err)
	}
	return addr, nil
}

func (b SettlementBatch) RepaymentAddress(ctx context.Context) ([]byte, error) {
	sourceChainConfig, err := b.SourceChainConfig(ctx)
	if err != nil {
		return nil, fmt.Errorf("getting source chain config: %w", err)
	}

	var repaymentAddress []byte
	switch sourceChainConfig.Type {
	case config.ChainType_EVM:
		if sourceChainConfig.SolverAddress == "" {
			return nil, fmt.Errorf("solver address not set for chain %s", sourceChainConfig.ChainID)
		}
		repaymentAddress = common.BytesToHash(common.HexToAddress(sourceChainConfig.SolverAddress).Bytes()).Bytes()
	default:
		return nil, fmt.Errorf("unsupported destination chain type %s for settlement", sourceChainConfig.Type)
	}

	return repaymentAddress, nil
}

func (b SettlementBatch) TotalValue() (*big.Int, error) {
	sum := big.NewInt(0)
	for _, settlement := range b {
		value, ok := new(big.Int).SetString(settlement.Amount, 10)
		if !ok {
			return nil, fmt.Errorf("converting settlement amount %s to *big.Int", settlement.Amount)
		}
		sum = sum.Add(sum, value)
	}
	return sum, nil
}

func (b SettlementBatch) String() string {
	return fmt.Sprintf(
		"SourceChainID: %s, DestinationChainID: %s, NumOrdersInBatch: %d",
		b.SourceChainID(),
		b.DestinationChainID(),
		len(b),
	)
}

package evm

import (
	"context"
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/skip-mev/go-fast-solver/shared/evmrpc"
)

type EVMTransaction struct {
	raw *types.Transaction
}

func (tx *EVMTransaction) Raw() interface{} {
	return tx.raw
}

func (tx *EVMTransaction) Bytes() ([]byte, error) {
	return tx.raw.MarshalBinary()
}

func (tx *EVMTransaction) Marshal() ([]byte, error) {
	return tx.raw.MarshalBinary()
}

type Builder interface {
	Build(ctx context.Context, opts ...TxBuildOption) (*types.Transaction, error)
}

type TxBuilder struct {
	rpc evmrpc.EVMChainRPC
}

func NewTxBuilder(rpc evmrpc.EVMChainRPC) TxBuilder {
	return TxBuilder{rpc: rpc}
}

type TxBuildOption func(context.Context, TxBuilder, *types.DynamicFeeTx) error

func (b TxBuilder) Build(ctx context.Context, opts ...TxBuildOption) (*types.Transaction, error) {
	var tx types.DynamicFeeTx
	for _, opt := range opts {
		if err := opt(ctx, b, &tx); err != nil {
			return nil, fmt.Errorf("building tx: %w", err)
		}
	}

	return types.NewTx(&tx), nil
}

func WithData(data []byte) TxBuildOption {
	return func(ctx context.Context, b TxBuilder, tx *types.DynamicFeeTx) error {
		tx.Data = data
		return nil
	}
}

func WithTo(address string) TxBuildOption {
	return func(ctx context.Context, b TxBuilder, tx *types.DynamicFeeTx) error {
		address := common.HexToAddress(address)
		tx.To = &address
		return nil
	}
}

func WithValue(value string) TxBuildOption {
	return func(ctx context.Context, b TxBuilder, tx *types.DynamicFeeTx) error {
		value, ok := new(big.Int).SetString(value, 10)
		if !ok {
			return fmt.Errorf("could not convert value %s to *big.Int", value)
		}
		tx.Value = value
		return nil
	}
}

func WithChainID(chainID string) TxBuildOption {
	return func(ctx context.Context, b TxBuilder, tx *types.DynamicFeeTx) error {
		id, ok := new(big.Int).SetString(chainID, 10)
		if !ok {
			return fmt.Errorf("could not convert chain id %s to *big.Int", chainID)
		}
		tx.ChainID = id
		return nil
	}
}

func WithNonceOfSigner(address string) TxBuildOption {
	return func(ctx context.Context, b TxBuilder, tx *types.DynamicFeeTx) error {
		nonce, err := b.rpc.PendingNonceAt(ctx, common.HexToAddress(address))
		if err != nil {
			return fmt.Errorf("fetching next nonce for %s: %w", address, err)
		}

		tx.Nonce = nonce
		return nil
	}
}

func WithNonce(nonce uint64) TxBuildOption {
	return func(ctx context.Context, b TxBuilder, tx *types.DynamicFeeTx) error {
		tx.Nonce = nonce
		return nil
	}
}

// EstimateGasForTx estimates the gas needed for a transaction with the given parameters
func (b TxBuilder) EstimateGasForTx(ctx context.Context, from, to, value string, data []byte) (uint64, error) {
	toAddr := common.HexToAddress(to)

	valueInt, ok := new(big.Int).SetString(value, 10)
	if !ok {
		return 0, fmt.Errorf("could not convert value %s to *big.Int", value)
	}

	gasLimit, err := b.rpc.EstimateGas(ctx, ethereum.CallMsg{
		From:  common.HexToAddress(from),
		To:    &toAddr,
		Value: valueInt,
		Data:  data,
	})
	if err != nil {
		return 0, fmt.Errorf("estimating gas limit: %w", err)
	}

	return gasLimit, nil
}

func WithEstimatedGasLimit(from, to, value string, data []byte) TxBuildOption {
	return func(ctx context.Context, b TxBuilder, tx *types.DynamicFeeTx) error {
		to := common.HexToAddress(to)

		value, ok := new(big.Int).SetString(value, 10)
		if !ok {
			return fmt.Errorf("could not convert value %s to *big.Int", value)
		}

		gasLimit, err := b.rpc.EstimateGas(ctx, ethereum.CallMsg{
			From:  common.HexToAddress(from),
			To:    &to,
			Value: value,
			Data:  data,
		})
		if err != nil {
			return fmt.Errorf("estimating gas limit: %w", err)
		}

		tx.Gas = gasLimit
		return nil
	}
}

func WithEstimatedGasTipCap() TxBuildOption {
	return func(ctx context.Context, b TxBuilder, tx *types.DynamicFeeTx) error {
		tipCap, err := b.rpc.SuggestGasTipCap(ctx)
		if err != nil {
			return fmt.Errorf("getting suggested gas tip cap: %w", err)
		}

		tx.GasTipCap = tipCap
		return nil
	}
}

func WithEstimatedGasFeeCap() TxBuildOption {
	return func(ctx context.Context, b TxBuilder, tx *types.DynamicFeeTx) error {
		if tx.GasTipCap == nil {
			if err := WithEstimatedGasTipCap()(ctx, b, tx); err != nil {
				return fmt.Errorf("getting estimated gas tip cap: %w", err)
			}
		}

		head, err := b.rpc.HeaderByNumber(ctx, nil)
		if err != nil {
			return fmt.Errorf("getting latest block header: %w", err)
		}

		tx.GasFeeCap = new(big.Int).Add(
			tx.GasTipCap,
			new(big.Int).Mul(head.BaseFee, big.NewInt(2)),
		)

		return nil
	}
}

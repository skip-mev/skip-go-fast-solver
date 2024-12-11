package evmrpc

import (
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/skip-mev/go-fast-solver/shared/contracts/usdc"
	"golang.org/x/net/context"
)

const (
	transferSignature = "0xddf252ad1be2c89b69c2b068fc378daa952ba7f163c4a11628f55a4df523b3ef"
)

type EVMChainRPC interface {
	GetERC20Transfers(ctx context.Context, txHash string) ([]ERC20Transfer, error)
	SendTx(ctx context.Context, txBytes []byte) (string, error)
	GetTxReceipt(ctx context.Context, txHash string) (*types.Receipt, error)
	GetTxByHash(ctx context.Context, txHash string) (*types.Transaction, bool, error)
	GetLogs(ctx context.Context, topics [][]common.Hash, addresses []common.Address) ([]types.Log, error)
	BlockByHash(ctx context.Context, hash common.Hash) (*types.Block, error)
	HeaderByHash(ctx context.Context, hash common.Hash) (*types.Header, error)
	HeaderByNumber(ctx context.Context, number *big.Int) (*types.Header, error)
	PendingNonceAt(ctx context.Context, address common.Address) (uint64, error)
	EstimateGas(ctx context.Context, msg ethereum.CallMsg) (uint64, error)
	SuggestGasPrice(ctx context.Context) (*big.Int, error)
	SuggestGasTipCap(ctx context.Context) (*big.Int, error)
	GetUSDCBalance(ctx context.Context, contractAddress string, account string) (*big.Int, error)
	Client() *ethclient.Client
	CodeAt(ctx context.Context, address common.Address, blockHeight *big.Int) ([]byte, error)
	CallContract(ctx context.Context, call ethereum.CallMsg, blockNumber *big.Int) ([]byte, error)
}

type ERC20Transfer struct {
	Source string
	Dest   string
}

type chainRPCImpl struct {
	cc *ethclient.Client
}

func NewEVMChainRPC(cc *ethclient.Client) EVMChainRPC {
	return &chainRPCImpl{cc: cc}
}

func (cr *chainRPCImpl) Client() *ethclient.Client {
	return cr.cc
}

func (cr *chainRPCImpl) GetUSDCBalance(ctx context.Context, contractAddress string, account string) (*big.Int, error) {
	token, err := usdc.NewUsdcCaller(common.HexToAddress(contractAddress), cr.cc)
	if err != nil {
		return nil, fmt.Errorf("creating usdc contract binding: %w", err)
	}

	currentBalance, err := token.BalanceOf(&bind.CallOpts{Context: ctx}, common.HexToAddress(account))
	if err != nil {
		return nil, fmt.Errorf("getting balance from usdc erc20 contract for account %s: %w", account, err)
	}

	return currentBalance, nil
}

func (cr *chainRPCImpl) GetERC20Transfers(ctx context.Context, txHash string) ([]ERC20Transfer, error) {
	var transfers []ERC20Transfer

	receipt, err := cr.cc.TransactionReceipt(context.Background(), common.HexToHash(txHash))
	if err != nil {
		return nil, err
	}
	for _, log := range receipt.Logs {
		logSignature := log.Topics[0].Hex()
		// check if there is a token transfer and store the recipient address if so
		if logSignature == transferSignature {
			sourceAddress := log.Topics[1].Hex()
			destAddress := log.Topics[2].Hex()
			transfers = append(transfers, ERC20Transfer{
				Source: "0x" + sourceAddress[len(sourceAddress)-40:],
				Dest:   "0x" + destAddress[len(destAddress)-40:],
			})
		}
	}
	return transfers, nil
}

func (cr *chainRPCImpl) SendTx(ctx context.Context, txBytes []byte) (string, error) {
	tx := &types.Transaction{}
	err := tx.UnmarshalBinary(txBytes)
	if err != nil {
		return "", err
	}
	err = cr.cc.SendTransaction(ctx, tx)
	if err != nil {
		return "", err
	}
	return tx.Hash().Hex(), nil
}

func (cr *chainRPCImpl) GetTxReceipt(ctx context.Context, txHash string) (*types.Receipt, error) {
	receipt, err := cr.cc.TransactionReceipt(ctx, common.HexToHash(txHash))
	if err != nil {
		return nil, err
	}
	return receipt, nil
}

func (cr *chainRPCImpl) GetTxByHash(ctx context.Context, txHash string) (*types.Transaction, bool, error) {
	tx, isPending, err := cr.cc.TransactionByHash(ctx, common.HexToHash(txHash))
	if err != nil {
		return nil, false, err
	}
	return tx, isPending, nil
}

func (cr *chainRPCImpl) GetLogs(ctx context.Context, topics [][]common.Hash, addresses []common.Address) ([]types.Log, error) {
	logsTopic, err := cr.cc.FilterLogs(ctx, ethereum.FilterQuery{Topics: topics, Addresses: addresses})
	if err != nil {
		return nil, err
	}
	return logsTopic, nil
}

func (cr *chainRPCImpl) BlockByHash(ctx context.Context, hash common.Hash) (*types.Block, error) {
	return cr.cc.BlockByHash(ctx, hash)
}

func (cr *chainRPCImpl) HeaderByHash(ctx context.Context, hash common.Hash) (*types.Header, error) {
	return cr.cc.HeaderByHash(ctx, hash)
}

func (cr *chainRPCImpl) HeaderByNumber(ctx context.Context, number *big.Int) (*types.Header, error) {
	return cr.cc.HeaderByNumber(ctx, number)
}

func (cr *chainRPCImpl) PendingNonceAt(ctx context.Context, address common.Address) (uint64, error) {
	return cr.cc.PendingNonceAt(ctx, address)
}

func (cr *chainRPCImpl) EstimateGas(ctx context.Context, msg ethereum.CallMsg) (uint64, error) {
	return cr.cc.EstimateGas(ctx, msg)
}

func (cr *chainRPCImpl) SuggestGasPrice(ctx context.Context) (*big.Int, error) {
	return cr.cc.SuggestGasPrice(ctx)
}

func (cr *chainRPCImpl) SuggestGasTipCap(ctx context.Context) (*big.Int, error) {
	return cr.cc.SuggestGasTipCap(ctx)
}

func (cr *chainRPCImpl) CodeAt(ctx context.Context, address common.Address, blockHeight *big.Int) ([]byte, error) {
	return cr.cc.CodeAt(ctx, address, blockHeight)
}

func (cr *chainRPCImpl) CallContract(ctx context.Context, call ethereum.CallMsg, blockNumber *big.Int) ([]byte, error) {
	return cr.cc.CallContract(ctx, call, blockNumber)
}

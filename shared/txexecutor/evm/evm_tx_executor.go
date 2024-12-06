package evm

import (
	"encoding/base64"
	"sync"
	"time"

	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/skip-mev/go-fast-solver/shared/config"
	"github.com/skip-mev/go-fast-solver/shared/evmrpc"
	"github.com/skip-mev/go-fast-solver/shared/signing"
	"github.com/skip-mev/go-fast-solver/shared/signing/evm"
	"golang.org/x/net/context"
)

type EVMTxExecutor interface {
	ExecuteTx(ctx context.Context, chainID string, signerAddress string, data []byte, value string, to string, signer signing.Signer) (txHash string, rawTxB64 string, err error)
}

type SerializedEVMTxExecutor struct {
	lock               sync.Mutex
	lastSubmissionTime time.Time
	txSubmissionDelay  time.Duration
	clientManager      evmrpc.EVMRPCClientManager
}

func DefaultEVMTxExecutor() EVMTxExecutor {
	return NewSerializedEVMTxExecutor(evmrpc.NewEVMRPCClientManager(), 500*time.Millisecond)
}

func NewSerializedEVMTxExecutor(clientManager evmrpc.EVMRPCClientManager, txSubmissionDelay time.Duration) EVMTxExecutor {
	return &SerializedEVMTxExecutor{
		clientManager:     clientManager,
		txSubmissionDelay: txSubmissionDelay,
	}
}

func (s *SerializedEVMTxExecutor) ExecuteTx(ctx context.Context, chainID string, signerAddress string, data []byte, value string, to string, signer signing.Signer) (txHash string, rawTxB64 string, err error) {
	client, err := s.clientManager.GetClient(ctx, chainID)
	if err != nil {
		return "", "", err
	}
	s.lock.Lock()
	defer func() {
		if err == nil {
			s.lastSubmissionTime = time.Now()
		}
		s.lock.Unlock()
	}()
	select {
	case <-time.After(time.Until(s.lastSubmissionTime.Add(s.txSubmissionDelay))):
	case <-ctx.Done():
		return "", "", ctx.Err()
	}

	chainCfg, err := config.GetConfigReader(ctx).GetChainConfig(chainID)
	if err != nil {
		return "", "", err
	}
	if chainCfg.EVM == nil {
		return "", "", fmt.Errorf("EVM chain config is null for chain id %s", chainID)
	}
	var minGasTipCap *big.Int
	if chainCfg.EVM.MinGasTipCap != nil {
		minGasTipCap = big.NewInt(*chainCfg.EVM.MinGasTipCap)
	}

	nonce, err := client.PendingNonceAt(ctx, common.HexToAddress(signerAddress))
	if err != nil {
		return "", "", err
	}
	tx, err := evm.NewTxBuilder(client).Build(
		ctx,
		evm.WithData(data),
		evm.WithValue(value),
		evm.WithTo(to),
		evm.WithChainID(chainID),
		evm.WithNonce(nonce),
		evm.WithEstimatedGasLimit(signerAddress, to, value, data),
		evm.WithEstimatedGasTipCap(minGasTipCap),
		evm.WithEstimatedGasFeeCap(minGasTipCap, big.NewFloat(2)),
	)
	signedTx, err := signer.Sign(ctx, chainID, tx)
	if err != nil {
		return "", "", err
	}
	signedTxBytes, err := signedTx.MarshalBinary()
	if err != nil {
		return "", "", err
	}
	txJsonBytes, err := signedTx.MarshalJSON()
	if err != nil {
		return "", "", err
	}
	txHash, err = client.SendTx(ctx, signedTxBytes)
	if err != nil {
		return "", "", err
	}
	return txHash, base64.StdEncoding.EncodeToString(txJsonBytes), nil
}

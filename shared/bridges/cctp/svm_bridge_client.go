package cctp

import (
	"context"
	"errors"
	"fmt"
	"math/big"
	"time"

	"github.com/gagliardetto/solana-go"
	"github.com/gagliardetto/solana-go/rpc"
	"github.com/gagliardetto/solana-go/rpc/ws"
	"github.com/skip-mev/go-fast-solver/db/gen/db"
	settlement "github.com/skip-mev/go-fast-solver/ordersettler/types"
	"github.com/skip-mev/go-fast-solver/shared/signing"
)

type SvmBridgeClient struct {
	rpcClient        *rpc.Client
	wsClient         *ws.Client
	signer           signing.Signer
	signerAddress    solana.PublicKey
	priorityFee      uint64
	submitRPCClients []*rpc.Client
}

func NewSvmBridgeClient(
	rpcUrl string,
	wsUrl string,
	signer signing.Signer,
	priorityFee uint64,
	submitRPCs []string,
) (*SvmBridgeClient, error) {
	signerAddress := solana.PublicKeyFromBytes(signer.Address())

	wsClient, err := ws.Connect(context.Background(), wsUrl)
	if err != nil {
		return nil, err
	}

	var submitRPCClients []*rpc.Client
	for _, rpcUrl := range submitRPCs {
		submitRPCClients = append(submitRPCClients, rpc.New(rpcUrl))
	}

	return &SvmBridgeClient{
		rpcClient:        rpc.New(rpcUrl),
		wsClient:         wsClient,
		signer:           signer,
		signerAddress:    signerAddress,
		priorityFee:      priorityFee,
		submitRPCClients: submitRPCClients,
	}, nil
}

func (c *SvmBridgeClient) FillOrder(ctx context.Context, order db.Order, gatewayContractAddress string) (string, string, *uint64, error) {
	return "", "", nil, errors.New("not implemented")
}

func (c *SvmBridgeClient) InitiateBatchSettlement(ctx context.Context, batch settlement.SettlementBatch) (string, string, error) {
	return "", "", errors.New("not implemented")
}

func (c *SvmBridgeClient) IsSettlementComplete(ctx context.Context, gatewayContractAddress, orderID string) (bool, error) {
	return false, errors.New("settlement complete event is not supported on Noble")
}

// Queries
func (c *SvmBridgeClient) SignerGasTokenBalance(ctx context.Context) (*big.Int, error) {
	balanceResult, err := c.rpcClient.GetBalance(ctx, c.signerAddress, rpc.CommitmentConfirmed)
	if err != nil {
		return nil, fmt.Errorf("failed to get signer balance: %w", err)
	}

	return big.NewInt(int64(balanceResult.Value)), nil
}

func (c *SvmBridgeClient) GetTxResult(ctx context.Context, txHash string) (*big.Int, *TxFailure, error) {
	return nil, nil, errors.New("not implemented")
}

func (c *SvmBridgeClient) InitiateTimeout(ctx context.Context, order db.Order, gatewayContractAddress string) (string, string, *uint64, error) {
	return "", "", nil, errors.New("not implemented")
}

// Submissions
func IncreaseAllowance(ctx context.Context, amount *big.Int) (string, error) {
	return "", errors.New("allowance is not supported on SVM chains")
}

func RevokeAllowance(ctx context.Context) (string, error) {
	return "", errors.New("allowance is not supported on SVM chains")
}

func (c *SvmBridgeClient) QueryOrderFillEvent(ctx context.Context, gatewayContractAddress, orderID string) (*string, *string, time.Time, error) {
	return nil, nil, time.Time{}, errors.New("not implemented")
}

func (c *SvmBridgeClient) OrderFillsByFiller(ctx context.Context, gatewayContractAddress, fillerAddress string) ([]Fill, error) {
	return nil, errors.New("not implemented")
}

func (c *SvmBridgeClient) Balance(ctx context.Context, address, denom string) (*big.Int, error) {
	return nil, errors.New("not implemented")
}

func (c *SvmBridgeClient) OrderExists(ctx context.Context, gatewayContractAddress, orderID string, blockNumber *big.Int) (bool, *big.Int, error) {
	return false, nil, errors.New("not implemented")
}

func (c *SvmBridgeClient) IsOrderRefunded(ctx context.Context, gatewayContractAddress, orderID string) (bool, error) {
	return false, errors.New("not implemented")
}

// Utils
func WaitForTx(ctx context.Context, txHash string) error {
	return nil
}
func Close() {}

func (c *SvmBridgeClient) BlockHeight(ctx context.Context) (uint64, error) {
	return 0, errors.New("not implemented")
}

package cctp

import (
	"context"
	"encoding/hex"
	"errors"
	"fmt"
	"math/big"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/skip-mev/go-fast-solver/db/gen/db"
	settlement "github.com/skip-mev/go-fast-solver/ordersettler/types"

	"github.com/avast/retry-go/v4"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"

	"github.com/skip-mev/go-fast-solver/shared/contracts/fast_transfer_gateway"
	"github.com/skip-mev/go-fast-solver/shared/contracts/usdc"
	"github.com/skip-mev/go-fast-solver/shared/signing"
	signingevm "github.com/skip-mev/go-fast-solver/shared/signing/evm"
)

type EVMClient interface {
	bind.DeployBackend
	bind.ContractBackend

	BalanceAt(ctx context.Context, account common.Address, blockNumber *big.Int) (*big.Int, error)
	Close()
}

type EVMBridgeClient struct {
	client EVMClient

	chainID string

	usdcERC20Contract *usdc.Usdc

	fromAddress common.Address
	signer      bind.SignerFn

	minGasTipCap *big.Int
}

var _ BridgeClient = (*EVMBridgeClient)(nil)

func NewEVMBridgeClient(
	client EVMClient,
	chainID string,
	signer signing.Signer,
	minGasTipCap *big.Int,
) (*EVMBridgeClient, error) {
	if signer == nil {
		signer = signing.NewNopSigner()
	}

	return &EVMBridgeClient{
		client:       client,
		chainID:      chainID,
		fromAddress:  common.BytesToAddress(signer.Address()),
		signer:       signingevm.EthereumSignerToBindSignerFn(signer, chainID),
		minGasTipCap: minGasTipCap,
	}, nil
}

func (c *EVMBridgeClient) USDCBalance(ctx context.Context, address string) (*big.Int, error) {
	balance, err := c.usdcERC20Contract.BalanceOf(
		&bind.CallOpts{Context: ctx},
		common.HexToAddress(address),
	)
	if err != nil {
		return nil, err
	}
	return balance, nil
}

func (c *EVMBridgeClient) SignerGasTokenBalance(ctx context.Context) (*big.Int, error) {
	balance, err := c.client.BalanceAt(ctx, c.fromAddress, nil)
	if err != nil {
		return nil, err
	}
	return balance, nil
}

func (c *EVMBridgeClient) FillOrder(ctx context.Context, order db.Order, gatewayContractAddress string) (string, string, *uint64, error) {
	return "", "", nil, errors.New("not implemented")
}

func (c *EVMBridgeClient) InitiateTimeout(ctx context.Context, order db.Order, gatewayContractAddress string) (string, string, *uint64, error) {
	return "", "", nil, errors.New("not implemented")

}

func (c *EVMBridgeClient) GetTxResult(ctx context.Context, txHash string) (*big.Int, *TxFailure, error) {
	receipt, err := c.client.TransactionReceipt(ctx, common.HexToHash(txHash))
	if err != nil {
		return nil, nil, err
	}
	if receipt == nil {
		return nil, nil, errors.New("receipt is nil")
	}
	if receipt.EffectiveGasPrice == nil {
		return nil, nil, errors.New("effective gas price is nil")
	}
	gasCost := new(big.Int).Mul(receipt.EffectiveGasPrice, big.NewInt(int64(receipt.GasUsed)))
	if receipt.Status == types.ReceiptStatusFailed {
		return gasCost, &TxFailure{"transaction failed"}, nil
	}
	return gasCost, nil, nil
}

func (c *EVMBridgeClient) InitiateBatchSettlement(ctx context.Context, batch settlement.SettlementBatch) (string, string, error) {
	return "", "", errors.New("not implemented")
}

func (c *EVMBridgeClient) IsSettlementComplete(ctx context.Context, gatewayContractAddress, orderID string) (bool, error) {
	fastTransferGateway, err := fast_transfer_gateway.NewFastTransferGateway(
		common.HexToAddress(gatewayContractAddress),
		c.client,
	)
	if err != nil {
		return false, err
	}
	orderIDBytes, err := hex.DecodeString(orderID)
	if err != nil {
		return false, err
	}
	orderStatus, err := fastTransferGateway.OrderStatuses(&bind.CallOpts{Context: ctx}, [32]byte(orderIDBytes))
	if err != nil {
		return false, err
	}
	return orderStatus == 1, nil // TODO: is this right?
}

type SettlementDetails struct {
	Sender            [32]byte
	Nonce             *big.Int
	DestinationDomain uint32
	Amount            *big.Int
}

func (c *EVMBridgeClient) OrderExists(ctx context.Context, gatewayContractAddress, orderID string, blockNumber *big.Int) (bool, *big.Int, error) {
	fastTransferGateway, err := fast_transfer_gateway.NewFastTransferGateway(
		common.HexToAddress(gatewayContractAddress),
		c.client,
	)
	if err != nil {
		return false, nil, err
	}
	orderIDBytes, err := hex.DecodeString(orderID)
	if err != nil {
		return false, nil, err
	}
	settlementDetails, err := fastTransferGateway.SettlementDetails(&bind.CallOpts{Context: ctx, BlockNumber: blockNumber}, [32]byte(orderIDBytes))
	if err != nil {
		return false, nil, fmt.Errorf("querying fast transfer gateway for orders settlement details: %w", err)
	}

	return settlementDetails.Nonce != nil && settlementDetails.DestinationDomain != 0 && settlementDetails.Amount != nil, settlementDetails.Amount, nil
}

func (c *EVMBridgeClient) IsOrderRefunded(ctx context.Context, gatewayContractAddress, orderID string) (bool, string, error) {
	fastTransferGateway, err := fast_transfer_gateway.NewFastTransferGateway(
		common.HexToAddress(gatewayContractAddress),
		c.client,
	)
	if err != nil {
		return false, "", err
	}

	orderIDBytes, err := hex.DecodeString(orderID)
	if err != nil {
		return false, "", err
	}

	status, err := fastTransferGateway.OrderStatuses(&bind.CallOpts{Context: ctx}, [32]byte(orderIDBytes))
	if err != nil {
		return false, "", fmt.Errorf("querying orderID %s status: %w", orderID, err)
	}

	if status == fast_transfer_gateway.OrderStatusRefunded {
		// Create topic for OrderRefunded event to filter logs for OrderRefunded events with this orderID
		orderRefundedTopic := [][32]byte{[32]byte(orderIDBytes)}
		filterOpts := &bind.FilterOpts{
			Context: ctx,
		}

		iterator, err := fastTransferGateway.FilterOrderRefunded(filterOpts, orderRefundedTopic)
		if err != nil {
			return false, "", fmt.Errorf("filtering OrderRefunded events: %w", err)
		}

		// Find the most recent OrderRefunded event for this orderID
		var refundEvent *types.Log
		for iterator.Next() {
			if iterator.Event != nil {
				refundEvent = &iterator.Event.Raw
			}
		}

		if refundEvent == nil {
			return false, "", fmt.Errorf("no refund event found for orderID %s, but the order is reported as refunded from fast gateway contract", orderID)
		}

		return true, refundEvent.TxHash.Hex(), nil
	}

	return false, "", nil
}

func (c *EVMBridgeClient) QueryOrderFillEvent(ctx context.Context, gatewayContractAddress, orderID string) (*string, *string, time.Time, error) {
	return nil, nil, time.Time{}, errors.New("not implemented")
}

func (c *EVMBridgeClient) ShouldRetryTx(ctx context.Context, txHash string, submitTime pgtype.Timestamp, txExpirationHeight *uint64) (bool, error) {
	return false, nil
}

func (c *EVMBridgeClient) WaitForTx(ctx context.Context, txHash string) error {
	_, err := retry.DoWithData(func() (*types.Receipt, error) {
		return c.client.TransactionReceipt(ctx, common.HexToHash(txHash))
	}, retry.Context(ctx), retry.Delay(1*time.Second), retry.MaxDelay(5*time.Second), retry.Attempts(20))
	return err
}

func (c *EVMBridgeClient) Close() {
	c.client.Close()
}

func (c *EVMBridgeClient) BlockHeight(ctx context.Context) (uint64, error) {
	resp, err := c.client.HeaderByNumber(ctx, nil)
	if err != nil {
		return 0, err
	}
	return resp.Number.Uint64(), nil
}

func (c *EVMBridgeClient) OrderFillsByFiller(ctx context.Context, gatewayContractAddress, fillerAddress string) ([]Fill, error) {
	return nil, errors.New("not implemented")
}

func (c *EVMBridgeClient) Balance(ctx context.Context, address, denom string) (*big.Int, error) {
	return nil, errors.New("not implemented")
}

package cctp

import (
	"context"
	"fmt"
	"math/big"
	"time"

	"github.com/skip-mev/go-fast-solver/shared/contracts/fast_transfer_gateway"

	"github.com/skip-mev/go-fast-solver/db/gen/db"
	"github.com/skip-mev/go-fast-solver/ordersettler/types"
)

type MessageSentEvent struct {
	Message []byte
}

type MessageReceivedEvent struct {
	Caller       string   `yaml:"caller"`
	SourceDomain uint32   `yaml:"source_domain"`
	Nonce        uint64   `yaml:"nonce"`
	Sender       [32]byte `yaml:"sender"`
	MessageBody  []byte   `yaml:"message_body"`
}

type TxFailure struct {
	Message string
}

func (t *TxFailure) String() string {
	return fmt.Sprintf("tx failed: %s", t.Message)

}

type ErrTxFailed struct {
	Code uint64
	Log  string
}

func (e ErrTxFailed) Error() string {
	return fmt.Sprintf("tx failed with code: %d and log: %s", e.Code, e.Log)
}

type ErrReceiveNotFound struct {
	TxHash string
}

func (e ErrReceiveNotFound) Error() string {
	return fmt.Sprintf("receive not found for tx: %s", e.TxHash)
}

type ErrTxResultNotFound struct {
	TxHash string
}

func (e ErrTxResultNotFound) Error() string {
	return fmt.Sprintf("tx result not found for tx: %s", e.TxHash)
}

type ErrOrderFillEventNotFound struct {
	OrderID string
}

func (e ErrOrderFillEventNotFound) Error() string {
	return fmt.Sprintf("order fill event not found for order: %s", e.OrderID)
}

type BridgeClient interface {
	BlockHeight(ctx context.Context) (uint64, error)
	SignerGasTokenBalance(ctx context.Context) (*big.Int, error)
	FillOrder(ctx context.Context, order db.Order, gatewayContractAddress string) (string, string, *uint64, error)
	GetTxResult(ctx context.Context, txHash string) (*big.Int, *TxFailure, error)
	InitiateBatchSettlement(ctx context.Context, batch types.SettlementBatch) (string, string, error)
	IsSettlementComplete(ctx context.Context, gatewayContractAddress, orderID string) (bool, error)
	OrderFillsByFiller(ctx context.Context, gatewayContractAddress, fillerAddress string) ([]Fill, error)
	QueryOrderFillEvent(ctx context.Context, gatewayContractAddress, orderID string) (*OrderFillEvent, time.Time, error)
	Balance(ctx context.Context, address, denom string) (*big.Int, error)
	OrderExists(ctx context.Context, gatewayContractAddress, orderID string, blockNumber *big.Int) (exists bool, amount *big.Int, err error)
	IsOrderRefunded(ctx context.Context, gatewayContractAddress, orderID string) (bool, string, error)
	InitiateTimeout(ctx context.Context, order db.Order, gatewayContractAddress string) (string, string, *uint64, error)
	OrderStatus(ctx context.Context, gatewayContractAddress, orderID string) (uint8, error)
	QueryOrderSubmittedEvent(ctx context.Context, gatewayContractAddress, orderID string) (*fast_transfer_gateway.FastTransferOrder, error)
}

// Code generated by sqlc. DO NOT EDIT.
// versions:
//   sqlc v1.27.0

package db

import (
	"context"
	"database/sql"
)

type Querier interface {
	GetAllHyperlaneTransfersWithTransferStatus(ctx context.Context, transferStatus string) ([]HyperlaneTransfer, error)
	GetAllOrderSettlementsWithSettlementStatus(ctx context.Context, settlementStatus string) ([]OrderSettlement, error)
	GetAllOrdersWithOrderStatus(ctx context.Context, orderStatus string) ([]Order, error)
	GetAllPendingRebalanceTransfers(ctx context.Context) ([]GetAllPendingRebalanceTransfersRow, error)
	GetHyperlaneTransferByMessageSentTx(ctx context.Context, arg GetHyperlaneTransferByMessageSentTxParams) (HyperlaneTransfer, error)
	GetOrderByOrderID(ctx context.Context, orderID string) (Order, error)
	GetOrderSettlement(ctx context.Context, arg GetOrderSettlementParams) (OrderSettlement, error)
	GetPendingRebalanceTransfersToChain(ctx context.Context, destinationChainID string) ([]GetPendingRebalanceTransfersToChainRow, error)
	GetSubmittedTxsByHyperlaneTransferId(ctx context.Context, hyperlaneTransferID sql.NullInt64) ([]SubmittedTx, error)
	GetSubmittedTxsByOrderIdAndType(ctx context.Context, arg GetSubmittedTxsByOrderIdAndTypeParams) ([]SubmittedTx, error)
	GetSubmittedTxsByOrderStatusAndType(ctx context.Context, arg GetSubmittedTxsByOrderStatusAndTypeParams) ([]SubmittedTx, error)
	GetSubmittedTxsWithStatus(ctx context.Context, txStatus string) ([]SubmittedTx, error)
	GetTransferMonitorMetadata(ctx context.Context, chainID string) (TransferMonitorMetadatum, error)
	InsertHyperlaneTransfer(ctx context.Context, arg InsertHyperlaneTransferParams) (HyperlaneTransfer, error)
	InsertOrder(ctx context.Context, arg InsertOrderParams) (Order, error)
	InsertOrderSettlement(ctx context.Context, arg InsertOrderSettlementParams) (OrderSettlement, error)
	InsertRebalanceTransfer(ctx context.Context, arg InsertRebalanceTransferParams) (int64, error)
	InsertSubmittedTx(ctx context.Context, arg InsertSubmittedTxParams) (SubmittedTx, error)
	InsertTransferMonitorMetadata(ctx context.Context, arg InsertTransferMonitorMetadataParams) (TransferMonitorMetadatum, error)
	InsertUnsentRebalanceTransfer(ctx context.Context, arg InsertUnsentRebalanceTransferParams) (int64, error)
	SetCompleteSettlementTx(ctx context.Context, arg SetCompleteSettlementTxParams) (OrderSettlement, error)
	SetFillTx(ctx context.Context, arg SetFillTxParams) (Order, error)
	SetInitiateSettlementTx(ctx context.Context, arg SetInitiateSettlementTxParams) (OrderSettlement, error)
	SetMessageStatus(ctx context.Context, arg SetMessageStatusParams) (HyperlaneTransfer, error)
	SetOrderStatus(ctx context.Context, arg SetOrderStatusParams) (Order, error)
	SetRefundTx(ctx context.Context, arg SetRefundTxParams) (Order, error)
	SetSettlementStatus(ctx context.Context, arg SetSettlementStatusParams) (OrderSettlement, error)
	SetSubmittedTxStatus(ctx context.Context, arg SetSubmittedTxStatusParams) (SubmittedTx, error)
	UpdateTransferStatus(ctx context.Context, arg UpdateTransferStatusParams) error
	UpdateTransferTxHash(ctx context.Context, arg UpdateTransferTxHashParams) error
}

var _ Querier = (*Queries)(nil)

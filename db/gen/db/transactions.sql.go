// Code generated by sqlc. DO NOT EDIT.
// versions:
//   sqlc v1.27.0
// source: transactions.sql

package db

import (
	"context"
	"database/sql"
)

const getSubmittedTxsByHyperlaneTransferId = `-- name: GetSubmittedTxsByHyperlaneTransferId :many
SELECT id, created_at, updated_at, order_id, order_settlement_id, hyperlane_transfer_id, chain_id, tx_hash, raw_tx, tx_type, tx_status, tx_status_message, rebalance_transfer_id FROM submitted_txs WHERE hyperlane_transfer_id = ?
`

func (q *Queries) GetSubmittedTxsByHyperlaneTransferId(ctx context.Context, hyperlaneTransferID sql.NullInt64) ([]SubmittedTx, error) {
	rows, err := q.db.QueryContext(ctx, getSubmittedTxsByHyperlaneTransferId, hyperlaneTransferID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []SubmittedTx
	for rows.Next() {
		var i SubmittedTx
		if err := rows.Scan(
			&i.ID,
			&i.CreatedAt,
			&i.UpdatedAt,
			&i.OrderID,
			&i.OrderSettlementID,
			&i.HyperlaneTransferID,
			&i.ChainID,
			&i.TxHash,
			&i.RawTx,
			&i.TxType,
			&i.TxStatus,
			&i.TxStatusMessage,
			&i.RebalanceTransferID,
		); err != nil {
			return nil, err
		}
		items = append(items, i)
	}
	if err := rows.Close(); err != nil {
		return nil, err
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}

const getSubmittedTxsByOrderIdAndType = `-- name: GetSubmittedTxsByOrderIdAndType :many
SELECT id, created_at, updated_at, order_id, order_settlement_id, hyperlane_transfer_id, chain_id, tx_hash, raw_tx, tx_type, tx_status, tx_status_message, rebalance_transfer_id FROM submitted_txs WHERE order_id = ? AND tx_type = ?
`

type GetSubmittedTxsByOrderIdAndTypeParams struct {
	OrderID sql.NullInt64
	TxType  string
}

func (q *Queries) GetSubmittedTxsByOrderIdAndType(ctx context.Context, arg GetSubmittedTxsByOrderIdAndTypeParams) ([]SubmittedTx, error) {
	rows, err := q.db.QueryContext(ctx, getSubmittedTxsByOrderIdAndType, arg.OrderID, arg.TxType)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []SubmittedTx
	for rows.Next() {
		var i SubmittedTx
		if err := rows.Scan(
			&i.ID,
			&i.CreatedAt,
			&i.UpdatedAt,
			&i.OrderID,
			&i.OrderSettlementID,
			&i.HyperlaneTransferID,
			&i.ChainID,
			&i.TxHash,
			&i.RawTx,
			&i.TxType,
			&i.TxStatus,
			&i.TxStatusMessage,
			&i.RebalanceTransferID,
		); err != nil {
			return nil, err
		}
		items = append(items, i)
	}
	if err := rows.Close(); err != nil {
		return nil, err
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}

const getSubmittedTxsByOrderStatusAndType = `-- name: GetSubmittedTxsByOrderStatusAndType :many
SELECT submitted_txs.id, submitted_txs.created_at, submitted_txs.updated_at, submitted_txs.order_id, submitted_txs.order_settlement_id, submitted_txs.hyperlane_transfer_id, submitted_txs.chain_id, submitted_txs.tx_hash, submitted_txs.raw_tx, submitted_txs.tx_type, submitted_txs.tx_status, submitted_txs.tx_status_message, submitted_txs.rebalance_transfer_id FROM submitted_txs INNER JOIN orders on submitted_txs.order_id = orders.id WHERE orders.order_status = ? AND submitted_txs.tx_type = ?
`

type GetSubmittedTxsByOrderStatusAndTypeParams struct {
	OrderStatus string
	TxType      string
}

func (q *Queries) GetSubmittedTxsByOrderStatusAndType(ctx context.Context, arg GetSubmittedTxsByOrderStatusAndTypeParams) ([]SubmittedTx, error) {
	rows, err := q.db.QueryContext(ctx, getSubmittedTxsByOrderStatusAndType, arg.OrderStatus, arg.TxType)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []SubmittedTx
	for rows.Next() {
		var i SubmittedTx
		if err := rows.Scan(
			&i.ID,
			&i.CreatedAt,
			&i.UpdatedAt,
			&i.OrderID,
			&i.OrderSettlementID,
			&i.HyperlaneTransferID,
			&i.ChainID,
			&i.TxHash,
			&i.RawTx,
			&i.TxType,
			&i.TxStatus,
			&i.TxStatusMessage,
			&i.RebalanceTransferID,
		); err != nil {
			return nil, err
		}
		items = append(items, i)
	}
	if err := rows.Close(); err != nil {
		return nil, err
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}

const getSubmittedTxsWithStatus = `-- name: GetSubmittedTxsWithStatus :many
SELECT id, created_at, updated_at, order_id, order_settlement_id, hyperlane_transfer_id, chain_id, tx_hash, raw_tx, tx_type, tx_status, tx_status_message, rebalance_transfer_id FROM submitted_txs WHERE tx_status = ?
`

func (q *Queries) GetSubmittedTxsWithStatus(ctx context.Context, txStatus string) ([]SubmittedTx, error) {
	rows, err := q.db.QueryContext(ctx, getSubmittedTxsWithStatus, txStatus)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []SubmittedTx
	for rows.Next() {
		var i SubmittedTx
		if err := rows.Scan(
			&i.ID,
			&i.CreatedAt,
			&i.UpdatedAt,
			&i.OrderID,
			&i.OrderSettlementID,
			&i.HyperlaneTransferID,
			&i.ChainID,
			&i.TxHash,
			&i.RawTx,
			&i.TxType,
			&i.TxStatus,
			&i.TxStatusMessage,
			&i.RebalanceTransferID,
		); err != nil {
			return nil, err
		}
		items = append(items, i)
	}
	if err := rows.Close(); err != nil {
		return nil, err
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}

const insertSubmittedTx = `-- name: InsertSubmittedTx :one
INSERT INTO submitted_txs (order_id, order_settlement_id, hyperlane_transfer_id, rebalance_transfer_id, chain_id, tx_hash, raw_tx, tx_type, tx_status) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?) RETURNING id, created_at, updated_at, order_id, order_settlement_id, hyperlane_transfer_id, chain_id, tx_hash, raw_tx, tx_type, tx_status, tx_status_message, rebalance_transfer_id
`

type InsertSubmittedTxParams struct {
	OrderID             sql.NullInt64
	OrderSettlementID   sql.NullInt64
	HyperlaneTransferID sql.NullInt64
	RebalanceTransferID sql.NullInt64
	ChainID             string
	TxHash              string
	RawTx               string
	TxType              string
	TxStatus            string
}

func (q *Queries) InsertSubmittedTx(ctx context.Context, arg InsertSubmittedTxParams) (SubmittedTx, error) {
	row := q.db.QueryRowContext(ctx, insertSubmittedTx,
		arg.OrderID,
		arg.OrderSettlementID,
		arg.HyperlaneTransferID,
		arg.RebalanceTransferID,
		arg.ChainID,
		arg.TxHash,
		arg.RawTx,
		arg.TxType,
		arg.TxStatus,
	)
	var i SubmittedTx
	err := row.Scan(
		&i.ID,
		&i.CreatedAt,
		&i.UpdatedAt,
		&i.OrderID,
		&i.OrderSettlementID,
		&i.HyperlaneTransferID,
		&i.ChainID,
		&i.TxHash,
		&i.RawTx,
		&i.TxType,
		&i.TxStatus,
		&i.TxStatusMessage,
		&i.RebalanceTransferID,
	)
	return i, err
}

const setSubmittedTxStatus = `-- name: SetSubmittedTxStatus :one
UPDATE submitted_txs SET tx_status = ?, tx_status_message = ?, updated_at = CURRENT_TIMESTAMP WHERE tx_hash = ? AND chain_id = ? RETURNING id, created_at, updated_at, order_id, order_settlement_id, hyperlane_transfer_id, chain_id, tx_hash, raw_tx, tx_type, tx_status, tx_status_message, rebalance_transfer_id
`

type SetSubmittedTxStatusParams struct {
	TxStatus        string
	TxStatusMessage sql.NullString
	TxHash          string
	ChainID         string
}

func (q *Queries) SetSubmittedTxStatus(ctx context.Context, arg SetSubmittedTxStatusParams) (SubmittedTx, error) {
	row := q.db.QueryRowContext(ctx, setSubmittedTxStatus,
		arg.TxStatus,
		arg.TxStatusMessage,
		arg.TxHash,
		arg.ChainID,
	)
	var i SubmittedTx
	err := row.Scan(
		&i.ID,
		&i.CreatedAt,
		&i.UpdatedAt,
		&i.OrderID,
		&i.OrderSettlementID,
		&i.HyperlaneTransferID,
		&i.ChainID,
		&i.TxHash,
		&i.RawTx,
		&i.TxType,
		&i.TxStatus,
		&i.TxStatusMessage,
		&i.RebalanceTransferID,
	)
	return i, err
}

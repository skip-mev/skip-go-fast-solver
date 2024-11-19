-- name: InsertSubmittedTx :one
INSERT INTO submitted_txs (order_id, order_settlement_id, hyperlane_transfer_id, chain_id, tx_hash, raw_tx, tx_type, tx_status) VALUES (?, ?, ?, ?, ?, ?, ?, ?) RETURNING *;

-- name: GetSubmittedTxsByOrderIdAndType :many
SELECT * FROM submitted_txs WHERE order_id = ? AND tx_type = ?;

-- name: GetSubmittedTxsByHyperlaneTransferId :many
SELECT * FROM submitted_txs WHERE hyperlane_transfer_id = ?;

-- name: GetSubmittedTxsWithStatus :many
SELECT * FROM submitted_txs WHERE tx_status = ?;

-- name: SetSubmittedTxStatus :one
UPDATE submitted_txs SET 
    tx_status = ?, tx_status_message = ?, tx_cost_uusdc = ?, updated_at = CURRENT_TIMESTAMP 
WHERE tx_hash = ? AND chain_id = ? RETURNING *;

-- name: GetSubmittedTxsByOrderStatusAndType :many
SELECT submitted_txs.* FROM submitted_txs INNER JOIN orders on submitted_txs.order_id = orders.id WHERE orders.order_status = ? AND submitted_txs.tx_type = ?;

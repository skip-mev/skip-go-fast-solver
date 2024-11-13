-- name: InsertHyperlaneTransfer :one
INSERT INTO hyperlane_transfers (
    source_chain_id,
    destination_chain_id,
    message_id,
    message_sent_tx,
    transfer_status,
    max_tx_fee_uusdc
) VALUES (?, ?, ?, ?, ?, ?) ON CONFLICT DO NOTHING RETURNING *;

-- name: GetAllHyperlaneTransfersWithTransferStatus :many
SELECT * FROM hyperlane_transfers WHERE transfer_status = ?;

-- name: SetMessageStatus :one
UPDATE hyperlane_transfers
SET updated_at=CURRENT_TIMESTAMP, transfer_status = ?, transfer_status_message = ?
WHERE source_chain_id = ? AND destination_chain_id = ? AND message_id = ?
    RETURNING *;

-- name: GetHyperlaneTransferByMessageSentTx :one
SELECT * FROM hyperlane_transfers WHERE message_sent_tx = ? AND source_chain_id = ?;

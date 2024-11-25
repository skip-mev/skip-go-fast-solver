-- name: InsertRebalanceTransfer :one
INSERT INTO rebalance_transfers (
    tx_hash,
    source_chain_id,
    destination_chain_id,
    amount
) VALUES (?, ?, ?, ?) RETURNING id;

-- name: GetPendingRebalanceTransfersToChain :many
SELECT 
    id,
    tx_hash,
    source_chain_id,
    destination_chain_id,
    amount
FROM rebalance_transfers
WHERE destination_chain_id = ? AND status = 'PENDING';

-- name: GetAllPendingRebalanceTransfers :many
SELECT 
    id,
    tx_hash,
    source_chain_id,
    destination_chain_id,
    amount 
FROM rebalance_transfers 
WHERE status = 'PENDING';


-- name: UpdateTransferStatus :exec
UPDATE rebalance_transfers
SET updated_at=CURRENT_TIMESTAMP, status = ?
WHERE id = ?;

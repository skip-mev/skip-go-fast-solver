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
    amount,
    created_at
FROM rebalance_transfers 
WHERE status = 'PENDING';


-- name: UpdateTransferStatus :exec
UPDATE rebalance_transfers
SET updated_at=CURRENT_TIMESTAMP, status = ?
WHERE id = ?;

-- name: UpdateTransfer :exec
UPDATE rebalance_transfers
SET updated_at=CURRENT_TIMESTAMP, tx_hash = ?, amount = ?
WHERE id = ?;

-- name: InitializeRebalanceTransfer :one
INSERT INTO rebalance_transfers (
    tx_hash,
    source_chain_id,
    destination_chain_id,
    amount
) VALUES ('', ?, ?, '0') RETURNING id;

-- name: GetPendingRebalanceTransfersBetweenChains :many
SELECT
    id,
    tx_hash,
    source_chain_id,
    destination_chain_id,
    amount,
    created_at
FROM rebalance_transfers
WHERE status = 'PENDING' AND source_chain_id = ? AND destination_chain_id = ?;

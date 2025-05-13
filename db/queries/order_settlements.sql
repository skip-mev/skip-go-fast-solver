-- name: InsertOrderSettlement :one
INSERT INTO order_settlements (
    source_chain_id,
    destination_chain_id,
    source_chain_gateway_contract_address,
    amount,
    profit,
    order_id,
    settlement_status
) VALUES (?, ?, ?, ?, ?, ?, ?) ON CONFLICT DO NOTHING RETURNING *;

-- name: GetAllOrderSettlementsWithSettlementStatus :many
SELECT * FROM order_settlements WHERE settlement_status = ?;

-- name: GetOrderSettlement :one
SELECT * FROM order_settlements WHERE source_chain_id = ? AND source_chain_gateway_contract_address = ? AND order_id = ?;

-- name: SetInitiateSettlementTx :one
UPDATE order_settlements
SET updated_at=CURRENT_TIMESTAMP, initiate_settlement_tx_time=CURRENT_TIMESTAMP, initiate_settlement_tx = ?
WHERE source_chain_id = ? AND order_id = ? AND source_chain_gateway_contract_address = ?
    RETURNING *;

-- name: SetCompleteSettlementTx :one
UPDATE order_settlements
SET updated_at=CURRENT_TIMESTAMP, complete_settlement_tx = ?
WHERE source_chain_id = ? AND order_id = ? AND source_chain_gateway_contract_address = ?
    RETURNING *;

-- name: SetSettlementStatus :one
UPDATE order_settlements
SET updated_at=CURRENT_TIMESTAMP, settlement_status = ?, settlement_status_message = ?
WHERE source_chain_id = ? AND order_id = ? AND source_chain_gateway_contract_address = ?
    RETURNING *;

-- name: SetHyperlaneTransferID :one
UPDATE order_settlements
SET updated_at=CURRENT_TIMESTAMP, hyperlane_transfer_id = ?
WHERE source_chain_id = ? AND order_id = ? AND source_chain_gateway_contract_address = ?
    RETURNING *;

-- name: ClearInitiateSettlement :many
UPDATE order_settlements
SET updated_at=CURRENT_TIMESTAMP, initiate_settlement_tx = null, hyperlane_transfer_id = null, initiate_settlement_tx_time = null, settlement_status = ?
WHERE destination_chain_id=? AND initiate_settlement_tx=?
    RETURNING *;
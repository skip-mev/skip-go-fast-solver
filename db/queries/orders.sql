-- name: InsertOrder :one
INSERT INTO orders (
    source_chain_id,
    destination_chain_id,
    source_chain_gateway_contract_address,
    sender,
    recipient,
    amount_in,
    amount_out,
    nonce,
    data,
    order_creation_tx,
    order_creation_tx_block_height,
    order_id,
    order_status,
    timeout_timestamp
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?) ON CONFLICT DO NOTHING RETURNING *;

-- name: GetAllOrdersWithOrderStatus :many
SELECT * FROM orders WHERE order_status = ?;

-- name: SetFillTx :one
UPDATE orders
SET updated_at=CURRENT_TIMESTAMP, fill_tx = ?, filler = ?, order_status = ?
WHERE source_chain_id = ? AND order_id = ? AND source_chain_gateway_contract_address = ?
    RETURNING *;

-- name: SetOrderStatus :one
UPDATE orders
SET updated_at=CURRENT_TIMESTAMP, order_status = ?, order_status_message = ?
WHERE source_chain_id = ? AND order_id = ? AND source_chain_gateway_contract_address = ?
    RETURNING *;

-- name: SetRefundTx :one
UPDATE orders
SET updated_at=CURRENT_TIMESTAMP, refund_tx = ?, order_status = ?
WHERE source_chain_id = ? AND order_id = ? AND source_chain_gateway_contract_address = ?
    RETURNING *;

-- name: GetOrderByOrderID :one
SELECT * FROM orders WHERE order_id = ?;

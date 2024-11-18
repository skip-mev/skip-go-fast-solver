CREATE TABLE IF NOT EXISTS order_settlements (
    id INTEGER PRIMARY KEY AUTOINCREMENT,

    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,

    source_chain_id      TEXT NOT NULL,
    destination_chain_id TEXT NOT NULL,
    source_chain_gateway_contract_address TEXT NOT NULL,
    amount TEXT NOT NULL,
    fee TEXT NOT NULL,
    order_id TEXT NOT NULL,

    initiate_settlement_tx TEXT,
    complete_settlement_tx TEXT,
    settlement_status         TEXT NOT NULL,
    settlement_status_message TEXT,

    UNIQUE(source_chain_id, source_chain_gateway_contract_address, order_id)
);

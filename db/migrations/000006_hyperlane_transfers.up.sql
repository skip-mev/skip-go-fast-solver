CREATE TABLE IF NOT EXISTS hyperlane_transfers (
    id INTEGER PRIMARY KEY AUTOINCREMENT,

    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,

    source_chain_id      TEXT NOT NULL,
    destination_chain_id TEXT NOT NULL,
    message_id TEXT NOT NULL,
    message_sent_tx    TEXT NOT NULL,

    transfer_status         TEXT NOT NULL,
    transfer_status_message TEXT,

    transfer_value TEXT,
    max_gas_price_pct INTEGER,

    UNIQUE(source_chain_id, destination_chain_id, message_id)
);

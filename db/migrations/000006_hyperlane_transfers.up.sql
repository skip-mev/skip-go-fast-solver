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

    max_tx_fee_uusdc TEXT,

    UNIQUE(source_chain_id, destination_chain_id, message_id)
);

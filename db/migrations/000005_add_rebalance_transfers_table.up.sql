CREATE TABLE IF NOT EXISTS rebalance_transfers (
    id INTEGER PRIMARY KEY AUTOINCREMENT,

    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,

    tx_hash TEXT NOT NULL,
    source_chain_id      TEXT NOT NULL,
    destination_chain_id TEXT NOT NULL,
    amount TEXT NOT NULL,
    status         TEXT NOT NULL DEFAULT 'PENDING',

    CHECK (status IN ('PENDING', 'SUCCESS', 'FAILED', 'ABANDONED'))
);

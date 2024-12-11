CREATE TABLE IF NOT EXISTS submitted_txs (
     id INTEGER PRIMARY KEY AUTOINCREMENT,
     created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
     updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,

     order_id INT,
     order_settlement_id INT,
     hyperlane_transfer_id INT,

     chain_id TEXT NOT NULL,
     tx_hash         TEXT NOT NULL,
     raw_tx          TEXT NOT NULL,
     tx_type TEXT NOT NULL,
     tx_status         TEXT NOT NULL,
     tx_status_message TEXT,

     FOREIGN KEY (order_id) REFERENCES orders(id),
     FOREIGN KEY (order_settlement_id) REFERENCES order_settlements(id),
     FOREIGN KEY (hyperlane_transfer_id) REFERENCES hyperlane_transfers(id)
);

CREATE UNIQUE INDEX submitted_txs_settlement_chain_tx_key
ON submitted_txs(order_settlement_id, chain_id, tx_hash)
WHERE order_settlement_id IS NOT NULL;

CREATE UNIQUE INDEX submitted_txs_chain_tx_key
ON submitted_txs(chain_id, tx_hash)
WHERE order_settlement_id IS NULL;

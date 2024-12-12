ALTER TABLE submitted_txs ADD COLUMN rebalance_transfer_id INT REFERENCES rebalance_transfers(id);

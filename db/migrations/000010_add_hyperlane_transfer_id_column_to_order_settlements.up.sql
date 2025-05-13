ALTER TABLE order_settlements ADD COLUMN hyperlane_transfer_id INT REFERENCES hyperlane_transfers(id);
ALTER TABLE order_settlements ADD COLUMN initiate_settlement_tx_time TIMESTAMP;

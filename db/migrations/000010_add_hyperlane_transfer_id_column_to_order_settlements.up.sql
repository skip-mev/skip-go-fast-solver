ALTER TABLE order_settlements ADD COLUMN hyperlane_transfer_id INT REFERENCES hyperlane_transfers(id);

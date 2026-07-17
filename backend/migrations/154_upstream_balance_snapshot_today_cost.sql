ALTER TABLE upstream_balance_snapshots
    ADD COLUMN IF NOT EXISTS today_cost NUMERIC(20, 8) NOT NULL DEFAULT 0;

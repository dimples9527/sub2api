CREATE TABLE IF NOT EXISTS upstream_balance_snapshots (
    id BIGSERIAL PRIMARY KEY,
    provider_slug VARCHAR(64) NOT NULL,
    provider_name VARCHAR(100) NOT NULL DEFAULT '',
    provider_type VARCHAR(32) NOT NULL DEFAULT '',
    balance NUMERIC(20, 8) NOT NULL DEFAULT 0,
    amount_scale NUMERIC(20, 10) NOT NULL DEFAULT 1,
    status VARCHAR(20) NOT NULL DEFAULT 'success',
    error TEXT NOT NULL DEFAULT '',
    captured_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_upstream_balance_snapshots_provider_captured
    ON upstream_balance_snapshots (provider_slug, captured_at DESC, id DESC);

CREATE INDEX IF NOT EXISTS idx_upstream_balance_snapshots_captured
    ON upstream_balance_snapshots (captured_at DESC);

CREATE TABLE IF NOT EXISTS upstream_balance_recharges (
    id BIGSERIAL PRIMARY KEY,
    provider_slug VARCHAR(64) NOT NULL,
    provider_name VARCHAR(100) NOT NULL DEFAULT '',
    amount NUMERIC(20, 8) NOT NULL,
    amount_scale NUMERIC(20, 10) NOT NULL DEFAULT 1,
    note TEXT NOT NULL DEFAULT '',
    occurred_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_upstream_balance_recharges_provider_occurred
    ON upstream_balance_recharges (provider_slug, occurred_at DESC, id DESC);

CREATE INDEX IF NOT EXISTS idx_upstream_balance_recharges_occurred
    ON upstream_balance_recharges (occurred_at DESC);

CREATE TABLE IF NOT EXISTS supplier_providers (
    id BIGSERIAL PRIMARY KEY,
    code VARCHAR(64) NOT NULL,
    name VARCHAR(120) NOT NULL,
    provider_type VARCHAR(32) NOT NULL,
    base_url TEXT NOT NULL,
    login_url TEXT NOT NULL DEFAULT '',
    api_keys_url TEXT NOT NULL DEFAULT '',
    groups_url TEXT NOT NULL DEFAULT '',
    available_groups_url TEXT NOT NULL DEFAULT '',
    balance_url TEXT NOT NULL DEFAULT '',
    usage_cost_url TEXT NOT NULL DEFAULT '',
    account_name_prefix VARCHAR(100) NOT NULL DEFAULT '',
    temp_disable_minutes INT NOT NULL DEFAULT 0 CHECK (temp_disable_minutes >= 0),
    account_rate_multiplier_scale NUMERIC(12, 6) NOT NULL DEFAULT 1 CHECK (account_rate_multiplier_scale > 0),
    sort_order INT NOT NULL DEFAULT 0,
    enabled BOOLEAN NOT NULL DEFAULT TRUE,
    is_default BOOLEAN NOT NULL DEFAULT FALSE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ NULL
);

CREATE UNIQUE INDEX IF NOT EXISTS uq_supplier_providers_code_active
    ON supplier_providers (code) WHERE deleted_at IS NULL;
CREATE UNIQUE INDEX IF NOT EXISTS uq_supplier_providers_default_active
    ON supplier_providers ((is_default)) WHERE is_default = TRUE AND deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_supplier_providers_enabled_sort
    ON supplier_providers (enabled, sort_order, id) WHERE deleted_at IS NULL;

CREATE TABLE IF NOT EXISTS supplier_provider_credentials (
    provider_id BIGINT PRIMARY KEY REFERENCES supplier_providers(id) ON DELETE CASCADE,
    email VARCHAR(255) NOT NULL DEFAULT '',
    username VARCHAR(255) NOT NULL DEFAULT '',
    password_encrypted TEXT NOT NULL DEFAULT '',
    encryption_version SMALLINT NOT NULL DEFAULT 1,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS supplier_provider_runtime_stats (
    provider_id BIGINT PRIMARY KEY REFERENCES supplier_providers(id) ON DELETE CASCADE,
    status VARCHAR(32) NOT NULL DEFAULT 'unknown',
    risk_level VARCHAR(32) NOT NULL DEFAULT 'normal',
    valid_account_count INT NOT NULL DEFAULT 0,
    schedulable_account_count INT NOT NULL DEFAULT 0,
    request_count BIGINT NOT NULL DEFAULT 0,
    success_rate NUMERIC(8, 4) NOT NULL DEFAULT 0,
    period_cost NUMERIC(20, 6) NOT NULL DEFAULT 0,
    current_balance NUMERIC(20, 6) NOT NULL DEFAULT 0,
    today_cost NUMERIC(20, 6) NOT NULL DEFAULT 0,
    estimated_days NUMERIC(12, 4) NULL,
    rate_risk_count INT NOT NULL DEFAULT 0,
    sync_status VARCHAR(32) NOT NULL DEFAULT 'never',
    sync_message TEXT NOT NULL DEFAULT '',
    last_sync_at TIMESTAMPTZ NULL,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS supplier_provider_metric_snapshots (
    id BIGSERIAL PRIMARY KEY,
    provider_id BIGINT NOT NULL REFERENCES supplier_providers(id) ON DELETE CASCADE,
    status VARCHAR(32) NOT NULL DEFAULT 'unknown',
    valid_account_count INT NOT NULL DEFAULT 0,
    schedulable_account_count INT NOT NULL DEFAULT 0,
    request_count BIGINT NOT NULL DEFAULT 0,
    success_rate NUMERIC(8, 4) NOT NULL DEFAULT 0,
    period_cost NUMERIC(20, 6) NOT NULL DEFAULT 0,
    current_balance NUMERIC(20, 6) NOT NULL DEFAULT 0,
    today_cost NUMERIC(20, 6) NOT NULL DEFAULT 0,
    rate_risk_count INT NOT NULL DEFAULT 0,
    sample_status VARCHAR(32) NOT NULL DEFAULT 'success',
    error_message TEXT NOT NULL DEFAULT '',
    captured_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_supplier_metric_snapshots_provider_time
    ON supplier_provider_metric_snapshots (provider_id, captured_at DESC);

CREATE TABLE IF NOT EXISTS supplier_provider_sync_runs (
    id BIGSERIAL PRIMARY KEY,
    provider_id BIGINT NULL REFERENCES supplier_providers(id) ON DELETE SET NULL,
    trigger_source VARCHAR(32) NOT NULL DEFAULT 'manual',
    status VARCHAR(32) NOT NULL,
    checked_count INT NOT NULL DEFAULT 0,
    created_count INT NOT NULL DEFAULT 0,
    updated_count INT NOT NULL DEFAULT 0,
    skipped_count INT NOT NULL DEFAULT 0,
    conflict_count INT NOT NULL DEFAULT 0,
    error_message TEXT NOT NULL DEFAULT '',
    started_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    finished_at TIMESTAMPTZ NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_supplier_sync_runs_provider_time
    ON supplier_provider_sync_runs (provider_id, started_at DESC);
CREATE INDEX IF NOT EXISTS idx_supplier_sync_runs_status_time
    ON supplier_provider_sync_runs (status, started_at DESC);

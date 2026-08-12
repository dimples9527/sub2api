CREATE TABLE IF NOT EXISTS supplier_provider_monitor_targets (
    id BIGSERIAL PRIMARY KEY,
    provider_id BIGINT NOT NULL REFERENCES supplier_providers(id) ON DELETE CASCADE,
    monitor_key VARCHAR(255) NOT NULL,
    monitor_name VARCHAR(255) NOT NULL,
    monitor_provider VARCHAR(64) NOT NULL DEFAULT '',
    primary_model VARCHAR(255) NOT NULL DEFAULT '',
    availability_7d NUMERIC(10,4) NOT NULL DEFAULT 0,
    active BOOLEAN NOT NULL DEFAULT TRUE,
    first_seen_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    last_seen_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(provider_id, monitor_key)
);

CREATE INDEX IF NOT EXISTS idx_supplier_monitor_targets_provider_active
    ON supplier_provider_monitor_targets(provider_id, active, last_seen_at DESC);

CREATE TABLE IF NOT EXISTS supplier_provider_monitor_bindings (
    id BIGSERIAL PRIMARY KEY,
    provider_id BIGINT NOT NULL REFERENCES supplier_providers(id) ON DELETE CASCADE,
    monitor_target_id BIGINT NOT NULL REFERENCES supplier_provider_monitor_targets(id) ON DELETE CASCADE,
    local_account_id BIGINT NOT NULL REFERENCES accounts(id) ON DELETE CASCADE,
    supplier_provider_account_id BIGINT REFERENCES supplier_provider_accounts(id) ON DELETE SET NULL,
    match_source VARCHAR(32) NOT NULL DEFAULT 'manual',
    match_status VARCHAR(32) NOT NULL DEFAULT 'active',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(provider_id, monitor_target_id),
    UNIQUE(provider_id, local_account_id, monitor_target_id)
);

CREATE INDEX IF NOT EXISTS idx_supplier_monitor_bindings_local_account
    ON supplier_provider_monitor_bindings(local_account_id, match_status);

CREATE TABLE IF NOT EXISTS supplier_provider_monitor_samples (
    id BIGSERIAL PRIMARY KEY,
    monitor_target_id BIGINT NOT NULL REFERENCES supplier_provider_monitor_targets(id) ON DELETE CASCADE,
    checked_at TIMESTAMPTZ NOT NULL,
    status VARCHAR(32) NOT NULL,
    raw_status VARCHAR(64) NOT NULL DEFAULT '',
    latency_ms BIGINT NOT NULL DEFAULT 0,
    ping_latency_ms BIGINT NOT NULL DEFAULT 0,
    availability_7d NUMERIC(10,4) NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(monitor_target_id, checked_at)
);

CREATE INDEX IF NOT EXISTS idx_supplier_monitor_samples_target_checked
    ON supplier_provider_monitor_samples(monitor_target_id, checked_at DESC);

CREATE TABLE IF NOT EXISTS supplier_account_health_history (
    id BIGSERIAL PRIMARY KEY,
    local_account_id BIGINT NOT NULL REFERENCES accounts(id) ON DELETE CASCADE,
    local_account_name VARCHAR(255) NOT NULL DEFAULT '',
    provider_id BIGINT NULL REFERENCES supplier_providers(id) ON DELETE SET NULL,
    provider_name VARCHAR(255) NOT NULL DEFAULT '',
    platform VARCHAR(64) NOT NULL DEFAULT '',
    checked_at TIMESTAMPTZ NOT NULL,
    started_at TIMESTAMPTZ NOT NULL,
    finished_at TIMESTAMPTZ NOT NULL,
    status VARCHAR(16) NOT NULL,
    latency_ms BIGINT NULL,
    latency_limit_ms BIGINT NOT NULL DEFAULT 0 CHECK (latency_limit_ms >= 0),
    model_id VARCHAR(255) NOT NULL DEFAULT '',
    schedulable_before BOOLEAN NOT NULL DEFAULT TRUE,
    schedulable_after BOOLEAN NOT NULL DEFAULT TRUE,
    action VARCHAR(32) NOT NULL DEFAULT 'none',
    consecutive_failed INTEGER NOT NULL DEFAULT 0 CHECK (consecutive_failed >= 0),
    consecutive_slow INTEGER NOT NULL DEFAULT 0 CHECK (consecutive_slow >= 0),
    consecutive_healthy INTEGER NOT NULL DEFAULT 0 CHECK (consecutive_healthy >= 0),
    reason TEXT NOT NULL DEFAULT '',
    error_message TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT ck_supplier_account_health_history_status CHECK (status IN ('healthy', 'slow', 'failed')),
    CONSTRAINT ck_supplier_account_health_history_latency CHECK (
        (status = 'failed' AND latency_ms IS NULL)
        OR (status IN ('healthy', 'slow') AND latency_ms IS NOT NULL AND latency_ms >= 0)
    )
);

CREATE INDEX IF NOT EXISTS idx_supplier_account_health_history_account_checked
    ON supplier_account_health_history (local_account_id, checked_at DESC, id DESC);
CREATE INDEX IF NOT EXISTS idx_supplier_account_health_history_status_latency
    ON supplier_account_health_history (status, checked_at DESC, latency_ms);

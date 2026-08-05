ALTER TABLE supplier_provider_runtime_stats
    ADD COLUMN IF NOT EXISTS auth_login_count BIGINT NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS auth_login_success_count BIGINT NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS auth_login_failure_count BIGINT NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS auth_cache_hit_count BIGINT NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS auth_cache_miss_count BIGINT NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS auth_last_login_at TIMESTAMPTZ NULL,
    ADD COLUMN IF NOT EXISTS auth_last_login_status VARCHAR(32) NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS auth_last_login_error TEXT NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS auth_last_cache_hit_at TIMESTAMPTZ NULL,
    ADD COLUMN IF NOT EXISTS auth_last_cache_error TEXT NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS auth_last_token_expires_at TIMESTAMPTZ NULL,
    ADD COLUMN IF NOT EXISTS auth_last_token_fingerprint VARCHAR(64) NOT NULL DEFAULT '';

CREATE TABLE IF NOT EXISTS supplier_provider_auth_events (
    id BIGSERIAL PRIMARY KEY,
    provider_id BIGINT NOT NULL REFERENCES supplier_providers(id) ON DELETE CASCADE,
    event_type VARCHAR(32) NOT NULL CHECK (event_type IN (
        'cache_hit',
        'cache_miss',
        'login_success',
        'login_failed',
        'cache_invalidated',
        'cache_error'
    )),
    source VARCHAR(32) NOT NULL DEFAULT 'unknown' CHECK (source IN (
        'sync',
        'endpoint_test',
        'manual',
        'unknown'
    )),
    status VARCHAR(32) NOT NULL DEFAULT '',
    started_at TIMESTAMPTZ NOT NULL,
    finished_at TIMESTAMPTZ NOT NULL,
    duration_ms BIGINT NOT NULL DEFAULT 0 CHECK (duration_ms >= 0),
    http_status INT NULL,
    error_message TEXT NOT NULL DEFAULT '',
    token_fingerprint VARCHAR(64) NOT NULL DEFAULT '',
    token_length INT NOT NULL DEFAULT 0 CHECK (token_length >= 0),
    token_expires_at TIMESTAMPTZ NULL,
    cookie_present BOOLEAN NOT NULL DEFAULT FALSE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_supplier_provider_auth_events_provider_time
    ON supplier_provider_auth_events (provider_id, created_at DESC, id DESC);
CREATE INDEX IF NOT EXISTS idx_supplier_provider_auth_events_provider_type_time
    ON supplier_provider_auth_events (provider_id, event_type, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_supplier_provider_auth_events_created_at
    ON supplier_provider_auth_events (created_at);
CREATE TABLE IF NOT EXISTS usage_log_latency_phases (
    id BIGSERIAL PRIMARY KEY,
    request_id VARCHAR(64) NOT NULL,
    api_key_id BIGINT NOT NULL DEFAULT 0,
    build_ms INTEGER NULL CHECK (build_ms >= 0),
    slot_wait_ms INTEGER NULL CHECK (slot_wait_ms >= 0),
    connect_ms INTEGER NULL CHECK (connect_ms >= 0),
    tls_ms INTEGER NULL CHECK (tls_ms >= 0),
    first_byte_ms INTEGER NULL CHECK (first_byte_ms >= 0),
    conn_reused BOOLEAN NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_usage_log_latency_phases_request
    ON usage_log_latency_phases (request_id, api_key_id, id DESC);
CREATE INDEX IF NOT EXISTS idx_usage_log_latency_phases_created
    ON usage_log_latency_phases (created_at);

CREATE TABLE IF NOT EXISTS supplier_rate_guard_change_logs (
    id BIGSERIAL PRIMARY KEY,
    mapping_id BIGINT NOT NULL,
    local_group_id BIGINT NOT NULL,
    local_group_name TEXT NOT NULL,
    upstream_group_key VARCHAR(255) NOT NULL DEFAULT '',
    upstream_group_name TEXT NOT NULL,
    old_rate DOUBLE PRECISION NOT NULL,
    new_rate DOUBLE PRECISION NOT NULL,
    status VARCHAR(16) NOT NULL DEFAULT 'pending',
    changed_at TIMESTAMPTZ NOT NULL,
    handled_at TIMESTAMPTZ NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_supplier_rate_guard_change_logs_status_changed_at
    ON supplier_rate_guard_change_logs (status, changed_at DESC, id DESC);

CREATE INDEX IF NOT EXISTS idx_supplier_rate_guard_change_logs_changed_at
    ON supplier_rate_guard_change_logs (changed_at DESC, id DESC);

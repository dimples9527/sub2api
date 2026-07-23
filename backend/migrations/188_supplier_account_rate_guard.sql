ALTER TABLE supplier_provider_accounts
    ADD COLUMN IF NOT EXISTS rate_sync_status VARCHAR(32) NOT NULL DEFAULT 'never',
    ADD COLUMN IF NOT EXISTS rate_sync_message TEXT NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS last_rate_sync_at TIMESTAMPTZ NULL;

CREATE TABLE IF NOT EXISTS supplier_account_rate_guard_unbind_logs (
    id BIGSERIAL PRIMARY KEY,
    run_id BIGINT NOT NULL REFERENCES supplier_automation_runs(id) ON DELETE CASCADE,
    provider_id BIGINT NOT NULL,
    provider_name VARCHAR(255) NOT NULL DEFAULT '',
    supplier_provider_account_id BIGINT NULL,
    upstream_account_key VARCHAR(255) NOT NULL DEFAULT '',
    upstream_account_name VARCHAR(255) NOT NULL DEFAULT '',
    local_account_id BIGINT NULL,
    local_account_name VARCHAR(255) NOT NULL DEFAULT '',
    local_group_id BIGINT NULL,
    local_group_name VARCHAR(255) NOT NULL DEFAULT '',
    raw_upstream_rate NUMERIC(20, 8) NOT NULL DEFAULT 0,
    rate_scale NUMERIC(20, 8) NOT NULL DEFAULT 1,
    effective_upstream_rate NUMERIC(20, 8) NOT NULL DEFAULT 0,
    local_group_rate NUMERIC(20, 8) NOT NULL DEFAULT 0,
    mode VARCHAR(16) NOT NULL,
    result VARCHAR(16) NOT NULL,
    before_bound BOOLEAN NOT NULL DEFAULT FALSE,
    after_bound BOOLEAN NOT NULL DEFAULT FALSE,
    before_schedulable BOOLEAN NULL,
    after_schedulable BOOLEAN NULL,
    error_message TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_supplier_account_rate_guard_logs_run
    ON supplier_account_rate_guard_unbind_logs (run_id, id DESC);
CREATE INDEX IF NOT EXISTS idx_supplier_account_rate_guard_logs_created
    ON supplier_account_rate_guard_unbind_logs (created_at DESC, id DESC);
CREATE INDEX IF NOT EXISTS idx_supplier_account_rate_guard_logs_provider
    ON supplier_account_rate_guard_unbind_logs (provider_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_supplier_account_rate_guard_logs_local_account
    ON supplier_account_rate_guard_unbind_logs (local_account_id, created_at DESC)
    WHERE local_account_id IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_supplier_account_rate_guard_logs_result
    ON supplier_account_rate_guard_unbind_logs (result, created_at DESC);

INSERT INTO supplier_automation_tasks (
    task_code,
    name,
    enabled,
    cron_expression,
    timeout_seconds,
    config_json
) VALUES (
    'supplier_account_rate_guard',
    '供应商账号倍率守护',
    FALSE,
    '@every 300s',
    600,
    '{}'::JSONB
) ON CONFLICT (task_code) DO NOTHING;

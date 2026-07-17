ALTER TABLE supplier_provider_sync_runs
    ADD COLUMN IF NOT EXISTS sync_scope VARCHAR(32) NOT NULL DEFAULT 'all';

CREATE INDEX IF NOT EXISTS idx_supplier_sync_runs_scope_time
    ON supplier_provider_sync_runs (sync_scope, started_at DESC);

CREATE TABLE IF NOT EXISTS supplier_provider_accounts (
    id BIGSERIAL PRIMARY KEY,
    provider_id BIGINT NOT NULL REFERENCES supplier_providers(id) ON DELETE CASCADE,
    upstream_account_key VARCHAR(255) NOT NULL,
    name VARCHAR(255) NOT NULL,
    status VARCHAR(32) NOT NULL DEFAULT 'unknown',
    group_key VARCHAR(255) NOT NULL DEFAULT '',
    group_name VARCHAR(255) NOT NULL DEFAULT '',
    rate_multiplier NUMERIC(20, 8) NOT NULL DEFAULT 1,
    raw_status VARCHAR(64) NOT NULL DEFAULT '',
    active BOOLEAN NOT NULL DEFAULT TRUE,
    first_seen_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    last_seen_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    inactive_at TIMESTAMPTZ NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT uq_supplier_provider_accounts_key UNIQUE (provider_id, upstream_account_key)
);

CREATE INDEX IF NOT EXISTS idx_supplier_provider_accounts_provider_active
    ON supplier_provider_accounts (provider_id, active, updated_at DESC);
CREATE INDEX IF NOT EXISTS idx_supplier_provider_accounts_group
    ON supplier_provider_accounts (provider_id, group_key) WHERE active = TRUE;
CREATE INDEX IF NOT EXISTS idx_supplier_provider_accounts_inactive
    ON supplier_provider_accounts (inactive_at) WHERE active = FALSE;

CREATE TABLE IF NOT EXISTS supplier_provider_groups (
    id BIGSERIAL PRIMARY KEY,
    provider_id BIGINT NOT NULL REFERENCES supplier_providers(id) ON DELETE CASCADE,
    upstream_group_key VARCHAR(255) NOT NULL,
    name VARCHAR(255) NOT NULL,
    rate_multiplier NUMERIC(20, 8) NOT NULL DEFAULT 1,
    raw_status VARCHAR(64) NOT NULL DEFAULT '',
    active BOOLEAN NOT NULL DEFAULT TRUE,
    first_seen_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    last_seen_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    inactive_at TIMESTAMPTZ NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT uq_supplier_provider_groups_key UNIQUE (provider_id, upstream_group_key)
);

CREATE INDEX IF NOT EXISTS idx_supplier_provider_groups_provider_active
    ON supplier_provider_groups (provider_id, active, updated_at DESC);
CREATE INDEX IF NOT EXISTS idx_supplier_provider_groups_inactive
    ON supplier_provider_groups (inactive_at) WHERE active = FALSE;

CREATE TABLE IF NOT EXISTS supplier_provider_daily_stats (
    id BIGSERIAL PRIMARY KEY,
    provider_id BIGINT NOT NULL REFERENCES supplier_providers(id) ON DELETE CASCADE,
    stat_date DATE NOT NULL,
    current_balance NUMERIC(20, 6) NOT NULL DEFAULT 0,
    today_cost NUMERIC(20, 6) NOT NULL DEFAULT 0,
    valid_account_count INT NOT NULL DEFAULT 0,
    schedulable_account_count INT NOT NULL DEFAULT 0,
    sync_status VARCHAR(32) NOT NULL DEFAULT 'never',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT uq_supplier_provider_daily_stats_date UNIQUE (provider_id, stat_date)
);

CREATE INDEX IF NOT EXISTS idx_supplier_provider_daily_stats_date
    ON supplier_provider_daily_stats (stat_date DESC, provider_id);

CREATE TABLE IF NOT EXISTS supplier_automation_tasks (
    id BIGSERIAL PRIMARY KEY,
    task_code VARCHAR(64) NOT NULL UNIQUE,
    name VARCHAR(120) NOT NULL,
    enabled BOOLEAN NOT NULL DEFAULT TRUE,
    cron_expression VARCHAR(120) NOT NULL,
    timeout_seconds INT NOT NULL DEFAULT 600 CHECK (timeout_seconds > 0),
    config_json JSONB NOT NULL DEFAULT '{}'::JSONB,
    last_status VARCHAR(32) NOT NULL DEFAULT 'never',
    last_message TEXT NOT NULL DEFAULT '',
    last_run_at TIMESTAMPTZ NULL,
    next_run_at TIMESTAMPTZ NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_supplier_automation_tasks_enabled
    ON supplier_automation_tasks (enabled, task_code);

CREATE TABLE IF NOT EXISTS supplier_automation_runs (
    id BIGSERIAL PRIMARY KEY,
    task_code VARCHAR(64) NOT NULL,
    trigger_source VARCHAR(32) NOT NULL DEFAULT 'manual',
    status VARCHAR(32) NOT NULL,
    message TEXT NOT NULL DEFAULT '',
    processed_count INT NOT NULL DEFAULT 0,
    success_count INT NOT NULL DEFAULT 0,
    failed_count INT NOT NULL DEFAULT 0,
    started_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    finished_at TIMESTAMPTZ NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_supplier_automation_runs_task_time
    ON supplier_automation_runs (task_code, started_at DESC);
CREATE INDEX IF NOT EXISTS idx_supplier_automation_runs_status_time
    ON supplier_automation_runs (status, started_at DESC);

INSERT INTO supplier_automation_tasks (
    task_code,
    name,
    enabled,
    cron_expression,
    timeout_seconds,
    config_json
) VALUES
    (
        'supplier_data_sync',
        '供应商数据同步',
        TRUE,
        '*/15 * * * *',
        600,
        '{}'::JSONB
    ),
    (
        'supplier_data_cleanup',
        '供应商数据清理',
        TRUE,
        '30 3 * * *',
        600,
        '{
          "automation_run_retention_days": 30,
          "sync_run_retention_days": 30,
          "metric_snapshot_retention_days": 30,
          "daily_stat_retention_days": 365,
          "inactive_account_retention_days": 90,
          "inactive_group_retention_days": 90
        }'::JSONB
    )
ON CONFLICT (task_code) DO NOTHING;

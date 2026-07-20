ALTER TABLE supplier_provider_groups
    ADD COLUMN IF NOT EXISTS rate_guard_selected BOOLEAN NOT NULL DEFAULT FALSE,
    ADD COLUMN IF NOT EXISTS rate_guard_selection_mode VARCHAR(16) NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS rate_guard_last_snapshot_at TIMESTAMPTZ NULL,
    ADD COLUMN IF NOT EXISTS rate_guard_last_checked_at TIMESTAMPTZ NULL;

CREATE UNIQUE INDEX IF NOT EXISTS uq_supplier_group_rate_guard
    ON supplier_provider_groups (local_group_id)
    WHERE rate_guard_selected = TRUE;

ALTER TABLE supplier_provider_runtime_stats
    ADD COLUMN IF NOT EXISTS group_sync_status VARCHAR(32) NOT NULL DEFAULT 'never',
    ADD COLUMN IF NOT EXISTS group_sync_message TEXT NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS last_group_sync_at TIMESTAMPTZ NULL;

INSERT INTO supplier_automation_tasks (
    task_code,
    name,
    enabled,
    cron_expression,
    timeout_seconds,
    config_json
) VALUES (
    'supplier_rate_guard',
    '供应商分组倍率守护',
    FALSE,
    '2-59/5 * * * *',
    300,
    '{
      "rate_guard_safety_multiplier": 1.1,
      "rate_guard_max_snapshot_age_seconds": 1800
    }'::JSONB
) ON CONFLICT (task_code) DO NOTHING;

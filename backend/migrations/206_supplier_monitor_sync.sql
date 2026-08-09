ALTER TABLE supplier_providers
    ADD COLUMN IF NOT EXISTS monitor_url TEXT NOT NULL DEFAULT '';

ALTER TABLE supplier_provider_types
    ADD COLUMN IF NOT EXISTS monitor_url TEXT NOT NULL DEFAULT '';

UPDATE supplier_provider_types
SET monitor_url = '/api/v1/channel-monitors?timezone=Asia%2FShanghai',
    updated_at = NOW()
WHERE code = 'sub2api'
  AND COALESCE(monitor_url, '') = ''
  AND deleted_at IS NULL;

UPDATE supplier_providers
SET monitor_url = '/api/v1/channel-monitors?timezone=Asia%2FShanghai',
    updated_at = NOW()
WHERE provider_type = 'sub2api'
  AND COALESCE(monitor_url, '') = ''
  AND deleted_at IS NULL;

INSERT INTO supplier_automation_tasks (
    task_code,
    name,
    enabled,
    cron_expression,
    timeout_seconds,
    config_json
) VALUES (
    'supplier_monitor_sync',
    '供应商监控数据同步',
    TRUE,
    '@every 30s',
    600,
    '{}'::JSONB
) ON CONFLICT (task_code) DO UPDATE SET
    name = EXCLUDED.name,
    enabled = supplier_automation_tasks.enabled,
    cron_expression = CASE
        WHEN supplier_automation_tasks.cron_expression IS NULL OR supplier_automation_tasks.cron_expression = ''
        THEN EXCLUDED.cron_expression
        ELSE supplier_automation_tasks.cron_expression
    END,
    timeout_seconds = CASE
        WHEN supplier_automation_tasks.timeout_seconds <= 0
        THEN EXCLUDED.timeout_seconds
        ELSE supplier_automation_tasks.timeout_seconds
    END,
    updated_at = NOW();

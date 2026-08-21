INSERT INTO supplier_automation_tasks (
    task_code,
    name,
    enabled,
    cron_expression,
    timeout_seconds,
    config_json
) VALUES (
    'supplier_provider_recharge_sync',
    '供应商充值记录同步',
    TRUE,
    '@every 30m',
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

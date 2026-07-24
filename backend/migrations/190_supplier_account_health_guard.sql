INSERT INTO supplier_automation_tasks (
    task_code,
    name,
    enabled,
    cron_expression,
    timeout_seconds,
    config_json
) VALUES (
    'supplier_account_health_guard',
    '供应商账号健康守护',
    FALSE,
    '@every 3600s',
    1800,
    '{
      "account_health_guard_max_accounts_per_run": 200,
      "account_health_guard_concurrency": 3,
      "account_health_guard_timeout_per_account_seconds": 90,
      "account_health_guard_failure_threshold": 3,
      "account_health_guard_slow_threshold": 3,
      "account_health_guard_recovery_threshold": 2,
      "account_health_guard_healthy_latency_ms": 15000,
      "account_health_guard_ignored_account_ids": [],
      "account_health_guard_account_models": {},
      "account_health_guard_platform_models": {},
      "account_health_guard_platform_latency_ms": {},
      "account_health_guard_cursor_account_id": 0
    }'::JSONB
) ON CONFLICT (task_code) DO NOTHING;

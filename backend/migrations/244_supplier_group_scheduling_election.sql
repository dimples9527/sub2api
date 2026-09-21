-- 分组择优调度：按本地分组扫描账号，关闭"测试失败仍开着调度"的账号，
-- 并为每个分组的成功账号中连续成功次数最多的前 N 个开启调度（默认单活 N=1）。
-- 复用 last_test_status 与 supplier_health_guard_healthy_count，不重测，做"组内最终裁决"。
-- 频率用间隔式（@every Ns），与健康守护一致，可在自动化任务中心的"执行间隔"里改。
-- 默认每小时一次、默认关闭，交由管理员显式启用（建议与健康守护同频、排其之后）。
INSERT INTO supplier_automation_tasks (
    task_code,
    name,
    enabled,
    cron_expression,
    timeout_seconds,
    config_json
) VALUES (
    'supplier_group_scheduling_election',
    '分组择优调度',
    FALSE,
    '@every 3600s',
    600,
    '{
      "group_scheduling_election_top_n": 1,
      "group_scheduling_election_disabled_group_ids": []
    }'::JSONB
) ON CONFLICT (task_code) DO NOTHING;

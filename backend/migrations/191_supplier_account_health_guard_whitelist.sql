-- 将供应商账号健康守护从忽略列表迁移为检查账号白名单。
-- 迁移后白名单为空，因此同步停用任务，避免调度器执行无配置任务。
UPDATE supplier_automation_tasks
SET enabled = FALSE,
    config_json = (config_json - 'account_health_guard_ignored_account_ids')
        || jsonb_build_object(
            'account_health_guard_account_ids',
            '[]'::jsonb
        )
WHERE task_code = 'supplier_account_health_guard';
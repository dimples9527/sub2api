-- 加速模型监控按任务和完成时间查询最近执行记录。
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_supplier_automation_runs_task_finished
    ON supplier_automation_runs (task_code, finished_at DESC, id DESC)
    WHERE finished_at IS NOT NULL;

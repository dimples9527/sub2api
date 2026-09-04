-- 监控项快照原先只做 upsert，上游删掉或换了 id 的监控项会永久留在 active = TRUE，
-- last_seen_at 冻结在最后一次出现的时刻，「仅显示活跃监控」筛不掉这些幽灵项。
-- 对齐 supplier_provider_accounts / supplier_provider_groups 的 inactive_at 语义，
-- 由同步侧在快照里未出现时置 active = FALSE, inactive_at = 同步时刻。
-- 历史幽灵行不做回填：部署后第一次监控同步（@every 30s）就会把它们全部标记出来。
ALTER TABLE supplier_provider_monitor_targets
    ADD COLUMN IF NOT EXISTS inactive_at TIMESTAMPTZ;

CREATE INDEX IF NOT EXISTS idx_supplier_monitor_targets_inactive
    ON supplier_provider_monitor_targets (inactive_at) WHERE active = FALSE;

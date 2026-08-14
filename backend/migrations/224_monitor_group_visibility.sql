ALTER TABLE monitor_group_platform_overrides
    ADD COLUMN IF NOT EXISTS show_in_monitor BOOLEAN NOT NULL DEFAULT TRUE;

CREATE INDEX IF NOT EXISTS idx_monitor_group_platform_overrides_show
    ON monitor_group_platform_overrides(show_in_monitor);

CREATE TABLE IF NOT EXISTS monitor_group_platform_overrides (
    id BIGSERIAL PRIMARY KEY,
    group_id BIGINT NOT NULL,
    actual_platform VARCHAR(50) NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (group_id),
    CONSTRAINT fk_monitor_group_platform_overrides_group
        FOREIGN KEY (group_id)
        REFERENCES groups(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_monitor_group_platform_overrides_platform
    ON monitor_group_platform_overrides(actual_platform);

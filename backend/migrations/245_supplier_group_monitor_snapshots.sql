-- 分组监控快照：只记「某时刻该分组正在调度的那一条账号」的监控结果。
-- 模型监控的「最新一刻」要的是当时真正在接客的那条账号，而不是全组账号的平均值；
-- 但 accounts.schedulable 只反映当前状态，没有历史，换人之后就还原不出"那时是谁在跑"。
-- 本表在每次采样时把当时的调度账号连同采样结果一起落下来，于是历史时刻各归各的账号，
-- 换人之后旧时刻的数据不会被现在新上来的账号覆盖。
-- 唯一键带 source：同一时刻监控同步与健康守护各记一条，互不覆盖。
CREATE TABLE IF NOT EXISTS supplier_group_monitor_snapshots (
    id BIGSERIAL PRIMARY KEY,
    group_id BIGINT NOT NULL REFERENCES groups(id) ON DELETE CASCADE,
    local_account_id BIGINT NOT NULL REFERENCES accounts(id) ON DELETE CASCADE,
    source VARCHAR(32) NOT NULL DEFAULT '',
    status VARCHAR(16) NOT NULL,
    latency_ms BIGINT NOT NULL DEFAULT 0 CHECK (latency_ms >= 0),
    availability NUMERIC(10, 4) NOT NULL DEFAULT 0,
    checked_at TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT ck_supplier_group_monitor_snapshots_status
        CHECK (status IN ('healthy', 'slow', 'failed', 'unavailable')),
    UNIQUE (group_id, source, checked_at)
);

CREATE INDEX IF NOT EXISTS idx_supplier_group_monitor_snapshots_group_checked
    ON supplier_group_monitor_snapshots (group_id, checked_at DESC);
CREATE INDEX IF NOT EXISTS idx_supplier_group_monitor_snapshots_account_checked
    ON supplier_group_monitor_snapshots (local_account_id, checked_at DESC);

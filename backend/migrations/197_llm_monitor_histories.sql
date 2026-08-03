-- Migration: 197_llm_monitor_histories
-- 远程模型监控快照：用于上游重启或短暂返回空 timeline 时恢复最近一次有效结果。
-- 同一监控地址、时间窗口和看板只保留一行最新快照，避免高频轮询导致数据库无限增长；
-- 过期地址或旧窗口由每日运维清理任务按批次删除。

CREATE TABLE IF NOT EXISTS llm_monitor_histories (
    id           BIGSERIAL PRIMARY KEY,
    source_key   CHAR(64) NOT NULL,
    period       VARCHAR(20) NOT NULL,
    board        VARCHAR(50) NOT NULL,
    payload      JSONB NOT NULL,
    payload_hash CHAR(64) NOT NULL,
    captured_at  TIMESTAMPTZ NOT NULL,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT llm_monitor_histories_source_period_board_key
        UNIQUE (source_key, period, board)
);

CREATE INDEX IF NOT EXISTS idx_llm_monitor_histories_captured_at
    ON llm_monitor_histories (captured_at);

CREATE INDEX IF NOT EXISTS idx_llm_monitor_histories_lookup
    ON llm_monitor_histories (source_key, period, board, captured_at DESC);

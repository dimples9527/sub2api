-- 上游模型降智检测：检测历史区域的「隐藏目标」配置（管理端维护，只对普通用户生效）。
--
-- 历史墙的归类桶只有两种口径（与前端 groupBucketOf 一致）：
--   1) 命中平台分组的记录按分组归桶（group_id 是发起时的快照）；
--   2) 没命中分组的第三方上游记录按 base_url 的主机名归桶。
-- 隐藏清单分成两张表一一对应：管理员隐藏某个分组 / 上游主机后，
-- 对应记录不再下发给普通用户；管理员不受影响，仍能看到全量数据。
--
-- 两处刻意不建外键：
--   - 分组会被改名或删除，历史记录要保留当时的语义，配置行也要跟着保留，
--     否则一次「删除分组」就会把此前隐藏的历史记录重新放出来；
--   - 主机名本来就是自由文本（按主机名匹配，端口不参与）。
CREATE TABLE IF NOT EXISTS model_audit_history_hidden_groups (
    group_id BIGINT PRIMARY KEY,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS model_audit_history_hidden_hosts (
    host TEXT PRIMARY KEY CHECK (host <> ''),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

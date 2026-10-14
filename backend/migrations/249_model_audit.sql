-- 上游模型降智检测（用户侧自助检测）。
--
-- 为什么不复用 accounts 的账号测试记录：本功能的凭据由用户自己提交
-- （Base URL + API Key），检测对象不是平台自己的上游账号，归属与生命周期都不同；
-- 混进 accounts 体系会让「平台的账号」和「用户填的一次性凭据」共享一套权限语义。
--
-- 为什么 API Key 不落库：凭据只在当次检测的上游调用中使用，用完即弃。
-- 落库等于把用户的上游密钥长期留在平台侧，是不必要的风险敞口。
CREATE TABLE IF NOT EXISTS model_audit_runs (
    id             BIGSERIAL   PRIMARY KEY,
    user_id        BIGINT      NOT NULL,
    base_url       TEXT        NOT NULL,
    model_id       TEXT        NOT NULL,
    -- running | completed | failed
    status         TEXT        NOT NULL DEFAULT 'running',
    -- normal | degraded | unknown
    verdict        TEXT        NOT NULL DEFAULT 'unknown',
    logic_passed   INTEGER     NOT NULL DEFAULT 0,
    logic_total    INTEGER     NOT NULL DEFAULT 0,
    -- '' | ok | degraded | unknown
    render_verdict TEXT        NOT NULL DEFAULT '',
    error_message  TEXT        NOT NULL DEFAULT '',
    duration_ms    INTEGER     NOT NULL DEFAULT 0,
    created_at     TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    completed_at   TIMESTAMPTZ
);

CREATE INDEX IF NOT EXISTS idx_model_audit_runs_user_created
    ON model_audit_runs (user_id, created_at DESC);

CREATE TABLE IF NOT EXISTS model_audit_items (
    id          BIGSERIAL   PRIMARY KEY,
    run_id      BIGINT      NOT NULL REFERENCES model_audit_runs (id) ON DELETE CASCADE,
    -- logic | render
    task_type   TEXT        NOT NULL,
    task_key    TEXT        NOT NULL DEFAULT '',
    prompt      TEXT        NOT NULL DEFAULT '',
    expected    TEXT        NOT NULL DEFAULT '',
    response    TEXT        NOT NULL DEFAULT '',
    passed      BOOLEAN     NOT NULL DEFAULT FALSE,
    verdict     TEXT        NOT NULL DEFAULT '',
    detail      TEXT        NOT NULL DEFAULT '',
    -- 生成题的归一化源码指纹，用于识别「多次检测返回同一份源码」。
    source_hash TEXT        NOT NULL DEFAULT '',
    duration_ms INTEGER     NOT NULL DEFAULT 0,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_model_audit_items_run
    ON model_audit_items (run_id);

-- 源码指纹查重只关心有指纹的行，用部分索引避免给逻辑题占位。
CREATE INDEX IF NOT EXISTS idx_model_audit_items_source_hash
    ON model_audit_items (source_hash) WHERE source_hash <> '';

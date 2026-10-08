-- 分组账号异常通知：订阅支持「分组」维度，事件类型新增 group_account_abnormal。
--
-- 为什么单独建分组冷却表（而不是复用 supplier_notification_cooldowns）：
-- 那张表的 provider_id 是 NOT NULL、唯一键是 (channel_id, provider_id, event_type)，
-- 而分组异常没有供应商。改它会让「余额 / 成本 / 分组变化」三条既有链路的运行时依赖一起变化，
-- 属于框架级风险；分组冷却的语义（按分组而非按供应商隔离）本来就与它不同，单独一张表更安全。
--
-- 投递表 provider_id 必须改为可空：分组事件没有供应商，而查询侧原先是
-- JOIN supplier_providers（内连接），非空 + 内连接会让分组投递记录直接查不出来。

-- 1) 订阅表：新增分组维度
ALTER TABLE supplier_notification_subscriptions
    ADD COLUMN IF NOT EXISTS group_id BIGINT NULL REFERENCES groups(id) ON DELETE CASCADE;

-- 唯一索引必须带上 group_id：否则「同渠道 + 同事件」下，按分组的订阅与按供应商的订阅会互相顶掉。
DROP INDEX IF EXISTS uq_supplier_notification_subscriptions_scope;
CREATE UNIQUE INDEX IF NOT EXISTS uq_supplier_notification_subscriptions_scope
    ON supplier_notification_subscriptions(channel_id, event_type, COALESCE(provider_id, 0), COALESCE(group_id, 0));

CREATE INDEX IF NOT EXISTS idx_supplier_notification_subscriptions_group
    ON supplier_notification_subscriptions(group_id);

-- 2) 事件类型约束：订阅表
-- 用 DO 块包住 DROP + ADD，与 236 / 238 改写同一对约束时的写法保持一致。
DO $$
BEGIN
    ALTER TABLE supplier_notification_subscriptions
        DROP CONSTRAINT IF EXISTS supplier_notification_subscriptions_type_ck;
    ALTER TABLE supplier_notification_subscriptions
        ADD CONSTRAINT supplier_notification_subscriptions_type_ck CHECK (
            event_type IN ('balance_low', 'balance_recovered', 'cost_overrun', 'cost_recovered',
                           'group_changed', 'group_account_abnormal')
        );
END $$;

-- 3) 投递表：允许无供应商（分组事件），并记录 group_id 便于排查
ALTER TABLE supplier_notification_deliveries
    ALTER COLUMN provider_id DROP NOT NULL;
ALTER TABLE supplier_notification_deliveries
    ADD COLUMN IF NOT EXISTS group_id BIGINT NULL REFERENCES groups(id) ON DELETE CASCADE;

DO $$
BEGIN
    ALTER TABLE supplier_notification_deliveries
        DROP CONSTRAINT IF EXISTS supplier_notification_deliveries_type_ck;
    ALTER TABLE supplier_notification_deliveries
        ADD CONSTRAINT supplier_notification_deliveries_type_ck CHECK (
            event_type IN ('balance_low', 'balance_recovered', 'cost_overrun', 'cost_recovered',
                           'group_changed', 'group_account_abnormal')
        );
END $$;

CREATE INDEX IF NOT EXISTS idx_supplier_notification_deliveries_group
    ON supplier_notification_deliveries(group_id);

-- 4) 分组维度的通知冷却（独立表，不碰 supplier_notification_cooldowns）
CREATE TABLE IF NOT EXISTS supplier_group_notification_cooldowns (
    id BIGSERIAL PRIMARY KEY,
    channel_id BIGINT NOT NULL REFERENCES supplier_notification_channels(id) ON DELETE CASCADE,
    group_id BIGINT NOT NULL REFERENCES groups(id) ON DELETE CASCADE,
    event_type VARCHAR(32) NOT NULL,
    expires_at TIMESTAMPTZ NOT NULL,
    claimed_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT supplier_group_notification_cooldowns_type_ck CHECK (event_type IN ('group_account_abnormal')),
    UNIQUE(channel_id, group_id, event_type)
);

CREATE INDEX IF NOT EXISTS idx_supplier_group_notification_cooldowns_expiry
    ON supplier_group_notification_cooldowns(expires_at);

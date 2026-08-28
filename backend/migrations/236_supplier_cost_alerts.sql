-- 供应商成本超额预警：全局配置、供应商覆盖配置与预警事件。
-- 阈值为绝对差额金额；0 表示未启用。
ALTER TABLE supplier_cost_deviation_settings
    ADD COLUMN IF NOT EXISTS alert_amount NUMERIC(20, 6) NOT NULL DEFAULT 0;

CREATE TABLE IF NOT EXISTS supplier_cost_alert_configs (
    id BIGSERIAL PRIMARY KEY,
    provider_id BIGINT NOT NULL UNIQUE REFERENCES supplier_providers(id) ON DELETE CASCADE,
    enabled BOOLEAN NOT NULL DEFAULT FALSE,
    threshold_amount NUMERIC(20, 6) NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT supplier_cost_alert_configs_amount_ck CHECK (threshold_amount >= 0)
);

CREATE INDEX IF NOT EXISTS idx_supplier_cost_alert_configs_enabled
    ON supplier_cost_alert_configs(enabled, provider_id);

CREATE TABLE IF NOT EXISTS supplier_cost_alert_events (
    id BIGSERIAL PRIMARY KEY,
    provider_id BIGINT NOT NULL REFERENCES supplier_providers(id) ON DELETE CASCADE,
    event_type VARCHAR(32) NOT NULL,
    status VARCHAR(16) NOT NULL,
    stat_date DATE NOT NULL,
    upstream_cost NUMERIC(20, 6) NOT NULL,
    local_cost NUMERIC(20, 6) NOT NULL,
    overrun_amount NUMERIC(20, 6) NOT NULL,
    threshold_amount NUMERIC(20, 6) NOT NULL,
    observed_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    resolved_at TIMESTAMPTZ NULL,
    last_seen_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT supplier_cost_alert_events_type_ck CHECK (event_type IN ('cost_overrun', 'cost_recovered')),
    CONSTRAINT supplier_cost_alert_events_status_ck CHECK (status IN ('active', 'resolved'))
);

CREATE UNIQUE INDEX IF NOT EXISTS uq_supplier_cost_alert_events_active_overrun
    ON supplier_cost_alert_events(provider_id)
    WHERE event_type = 'cost_overrun' AND status = 'active';
CREATE INDEX IF NOT EXISTS idx_supplier_cost_alert_events_provider_time
    ON supplier_cost_alert_events(provider_id, observed_at DESC);
CREATE INDEX IF NOT EXISTS idx_supplier_cost_alert_events_type_time
    ON supplier_cost_alert_events(event_type, observed_at DESC);

-- 扩展现有供应商通知事件约束，支持成本超额与成本恢复。
DO $$
BEGIN
    ALTER TABLE supplier_notification_subscriptions
        DROP CONSTRAINT IF EXISTS supplier_notification_subscriptions_type_ck;
    ALTER TABLE supplier_notification_subscriptions
        ADD CONSTRAINT supplier_notification_subscriptions_type_ck CHECK (
            event_type IN ('balance_low', 'balance_recovered', 'cost_overrun', 'cost_recovered')
        );
END $$;

DO $$
BEGIN
    ALTER TABLE supplier_notification_cooldowns
        DROP CONSTRAINT IF EXISTS supplier_notification_cooldowns_type_ck;
    ALTER TABLE supplier_notification_cooldowns
        ADD CONSTRAINT supplier_notification_cooldowns_type_ck CHECK (
            event_type IN ('balance_low', 'balance_recovered', 'cost_overrun', 'cost_recovered')
        );
END $$;

DO $$
BEGIN
    ALTER TABLE supplier_notification_deliveries
        DROP CONSTRAINT IF EXISTS supplier_notification_deliveries_type_ck;
    ALTER TABLE supplier_notification_deliveries
        ADD CONSTRAINT supplier_notification_deliveries_type_ck CHECK (
            event_type IN ('balance_low', 'balance_recovered', 'cost_overrun', 'cost_recovered')
        );
END $$;

-- 为成本通知保留独立事件关联，避免影响既有余额事件语义。
ALTER TABLE supplier_notification_deliveries
    ADD COLUMN IF NOT EXISTS cost_alert_event_id BIGINT NULL REFERENCES supplier_cost_alert_events(id) ON DELETE SET NULL;

CREATE INDEX IF NOT EXISTS idx_supplier_notification_deliveries_cost_alert_event
    ON supplier_notification_deliveries(cost_alert_event_id);

-- 供应商分组变化通知事件及其投递关联。
CREATE TABLE IF NOT EXISTS supplier_group_change_events (
    id BIGSERIAL PRIMARY KEY,
    provider_id BIGINT NOT NULL REFERENCES supplier_providers(id) ON DELETE CASCADE,
    provider_code VARCHAR(128) NOT NULL DEFAULT '',
    provider_name VARCHAR(256) NOT NULL DEFAULT '',
    sync_run_id BIGINT NULL REFERENCES supplier_provider_sync_runs(id) ON DELETE SET NULL,
    event_type VARCHAR(32) NOT NULL DEFAULT 'group_changed',
    observed_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    change_count INTEGER NOT NULL,
    payload_json JSONB NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT supplier_group_change_events_type_ck CHECK (event_type = 'group_changed'),
    CONSTRAINT supplier_group_change_events_count_ck CHECK (change_count > 0)
);

CREATE INDEX IF NOT EXISTS idx_supplier_group_change_events_provider_time
    ON supplier_group_change_events(provider_id, observed_at DESC);
CREATE INDEX IF NOT EXISTS idx_supplier_group_change_events_created
    ON supplier_group_change_events(created_at DESC);

-- 扩展供应商通知事件约束，支持分组变化通知。
DO $$
BEGIN
    ALTER TABLE supplier_notification_subscriptions
        DROP CONSTRAINT IF EXISTS supplier_notification_subscriptions_type_ck;
    ALTER TABLE supplier_notification_subscriptions
        ADD CONSTRAINT supplier_notification_subscriptions_type_ck CHECK (
            event_type IN ('balance_low', 'balance_recovered', 'cost_overrun', 'cost_recovered', 'group_changed')
        );
END $$;

DO $$
BEGIN
    ALTER TABLE supplier_notification_cooldowns
        DROP CONSTRAINT IF EXISTS supplier_notification_cooldowns_type_ck;
    ALTER TABLE supplier_notification_cooldowns
        ADD CONSTRAINT supplier_notification_cooldowns_type_ck CHECK (
            event_type IN ('balance_low', 'balance_recovered', 'cost_overrun', 'cost_recovered', 'group_changed')
        );
END $$;

DO $$
BEGIN
    ALTER TABLE supplier_notification_deliveries
        DROP CONSTRAINT IF EXISTS supplier_notification_deliveries_type_ck;
    ALTER TABLE supplier_notification_deliveries
        ADD CONSTRAINT supplier_notification_deliveries_type_ck CHECK (
            event_type IN ('balance_low', 'balance_recovered', 'cost_overrun', 'cost_recovered', 'group_changed')
        );
END $$;

ALTER TABLE supplier_notification_deliveries
    ADD COLUMN IF NOT EXISTS group_change_event_id BIGINT NULL
        REFERENCES supplier_group_change_events(id) ON DELETE SET NULL;

CREATE INDEX IF NOT EXISTS idx_supplier_notification_deliveries_group_change_event
    ON supplier_notification_deliveries(group_change_event_id);

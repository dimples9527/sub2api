-- 供应商余额预警与多渠道通知模块的独立数据表。
CREATE TABLE IF NOT EXISTS supplier_balance_alert_configs (
    id BIGSERIAL PRIMARY KEY,
    provider_id BIGINT NOT NULL UNIQUE REFERENCES supplier_providers(id) ON DELETE CASCADE,
    enabled BOOLEAN NOT NULL DEFAULT FALSE,
    threshold NUMERIC(20, 8) NOT NULL DEFAULT 0,
    cooldown_seconds INTEGER NOT NULL DEFAULT 3600,
    last_scan_at TIMESTAMPTZ NULL,
    last_balance NUMERIC(20, 8) NULL,
    last_scan_status VARCHAR(32) NOT NULL DEFAULT 'never',
    last_scan_error TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT supplier_balance_alert_configs_threshold_ck CHECK (threshold >= 0),
    CONSTRAINT supplier_balance_alert_configs_cooldown_ck CHECK (cooldown_seconds >= 0)
);

CREATE INDEX IF NOT EXISTS idx_supplier_balance_alert_configs_enabled
    ON supplier_balance_alert_configs(enabled, provider_id);

CREATE TABLE IF NOT EXISTS supplier_balance_alert_events (
    id BIGSERIAL PRIMARY KEY,
    provider_id BIGINT NOT NULL REFERENCES supplier_providers(id) ON DELETE CASCADE,
    event_type VARCHAR(32) NOT NULL,
    status VARCHAR(16) NOT NULL,
    balance NUMERIC(20, 8) NOT NULL,
    threshold NUMERIC(20, 8) NOT NULL,
    observed_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    resolved_at TIMESTAMPTZ NULL,
    last_seen_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT supplier_balance_alert_events_type_ck CHECK (event_type IN ('balance_low', 'balance_recovered')),
    CONSTRAINT supplier_balance_alert_events_status_ck CHECK (status IN ('active', 'resolved'))
);

CREATE UNIQUE INDEX IF NOT EXISTS uq_supplier_balance_alert_events_active_low
    ON supplier_balance_alert_events(provider_id, event_type)
    WHERE event_type = 'balance_low' AND status = 'active';
CREATE INDEX IF NOT EXISTS idx_supplier_balance_alert_events_provider_time
    ON supplier_balance_alert_events(provider_id, observed_at DESC);
CREATE INDEX IF NOT EXISTS idx_supplier_balance_alert_events_type_time
    ON supplier_balance_alert_events(event_type, observed_at DESC);

CREATE TABLE IF NOT EXISTS supplier_notification_channels (
    id BIGSERIAL PRIMARY KEY,
    name VARCHAR(128) NOT NULL,
    channel_type VARCHAR(32) NOT NULL,
    enabled BOOLEAN NOT NULL DEFAULT FALSE,
    config_encrypted TEXT NOT NULL DEFAULT '',
    proxy_encrypted TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT supplier_notification_channels_type_ck CHECK (channel_type IN ('feishu', 'email'))
);

CREATE INDEX IF NOT EXISTS idx_supplier_notification_channels_enabled
    ON supplier_notification_channels(enabled, id);

CREATE TABLE IF NOT EXISTS supplier_notification_subscriptions (
    id BIGSERIAL PRIMARY KEY,
    channel_id BIGINT NOT NULL REFERENCES supplier_notification_channels(id) ON DELETE CASCADE,
    provider_id BIGINT NULL REFERENCES supplier_providers(id) ON DELETE CASCADE,
    event_type VARCHAR(32) NOT NULL,
    enabled BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT supplier_notification_subscriptions_type_ck CHECK (event_type IN ('balance_low', 'balance_recovered'))
);

CREATE UNIQUE INDEX IF NOT EXISTS uq_supplier_notification_subscriptions_scope
    ON supplier_notification_subscriptions(channel_id, event_type, COALESCE(provider_id, 0));
CREATE INDEX IF NOT EXISTS idx_supplier_notification_subscriptions_event
    ON supplier_notification_subscriptions(channel_id, event_type, provider_id);

CREATE TABLE IF NOT EXISTS supplier_notification_cooldowns (
    id BIGSERIAL PRIMARY KEY,
    channel_id BIGINT NOT NULL REFERENCES supplier_notification_channels(id) ON DELETE CASCADE,
    provider_id BIGINT NOT NULL REFERENCES supplier_providers(id) ON DELETE CASCADE,
    event_type VARCHAR(32) NOT NULL,
    expires_at TIMESTAMPTZ NOT NULL,
    claimed_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT supplier_notification_cooldowns_type_ck CHECK (event_type IN ('balance_low', 'balance_recovered')),
    UNIQUE(channel_id, provider_id, event_type)
);

CREATE INDEX IF NOT EXISTS idx_supplier_notification_cooldowns_expiry
    ON supplier_notification_cooldowns(expires_at);

CREATE TABLE IF NOT EXISTS supplier_notification_deliveries (
    id BIGSERIAL PRIMARY KEY,
    channel_id BIGINT NOT NULL REFERENCES supplier_notification_channels(id) ON DELETE CASCADE,
    event_id BIGINT NULL REFERENCES supplier_balance_alert_events(id) ON DELETE SET NULL,
    provider_id BIGINT NOT NULL REFERENCES supplier_providers(id) ON DELETE CASCADE,
    event_type VARCHAR(32) NOT NULL,
    status VARCHAR(16) NOT NULL DEFAULT 'pending',
    payload_json JSONB NOT NULL,
    attempt_count INTEGER NOT NULL DEFAULT 0,
    next_attempt_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    last_error TEXT NOT NULL DEFAULT '',
    sent_at TIMESTAMPTZ NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT supplier_notification_deliveries_type_ck CHECK (event_type IN ('balance_low', 'balance_recovered')),
    CONSTRAINT supplier_notification_deliveries_status_ck CHECK (status IN ('pending', 'sending', 'delivered', 'failed')),
    CONSTRAINT supplier_notification_deliveries_attempt_ck CHECK (attempt_count >= 0)
);

CREATE INDEX IF NOT EXISTS idx_supplier_notification_deliveries_due
    ON supplier_notification_deliveries(status, next_attempt_at);
CREATE INDEX IF NOT EXISTS idx_supplier_notification_deliveries_created
    ON supplier_notification_deliveries(created_at DESC);

CREATE TABLE IF NOT EXISTS supplier_notification_delivery_attempts (
    id BIGSERIAL PRIMARY KEY,
    delivery_id BIGINT NOT NULL REFERENCES supplier_notification_deliveries(id) ON DELETE CASCADE,
    attempt_number INTEGER NOT NULL,
    status VARCHAR(16) NOT NULL,
    http_status INTEGER NOT NULL DEFAULT 0,
    error_message TEXT NOT NULL DEFAULT '',
    response_body TEXT NOT NULL DEFAULT '',
    attempted_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    finished_at TIMESTAMPTZ NULL,
    CONSTRAINT supplier_notification_delivery_attempts_status_ck CHECK (status IN ('sending', 'succeeded', 'failed')),
    UNIQUE(delivery_id, attempt_number)
);

CREATE INDEX IF NOT EXISTS idx_supplier_notification_delivery_attempts_delivery
    ON supplier_notification_delivery_attempts(delivery_id, attempt_number);
CREATE TABLE IF NOT EXISTS supplier_provider_types (
    id BIGSERIAL PRIMARY KEY,
    code VARCHAR(64) NOT NULL,
    name VARCHAR(120) NOT NULL,
    login_url TEXT NOT NULL DEFAULT '',
    api_keys_url TEXT NOT NULL DEFAULT '',
    groups_url TEXT NOT NULL DEFAULT '',
    available_groups_url TEXT NOT NULL DEFAULT '',
    balance_url TEXT NOT NULL DEFAULT '',
    usage_cost_url TEXT NOT NULL DEFAULT '',
    enabled BOOLEAN NOT NULL DEFAULT TRUE,
    sort_order INT NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ NULL
);

CREATE UNIQUE INDEX IF NOT EXISTS uq_supplier_provider_types_code_active
    ON supplier_provider_types (code) WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_supplier_provider_types_enabled_sort
    ON supplier_provider_types (enabled, sort_order, id) WHERE deleted_at IS NULL;

INSERT INTO supplier_provider_types (
    code, name, login_url, api_keys_url, groups_url, available_groups_url,
    balance_url, usage_cost_url, enabled, sort_order
) VALUES
    ('sub2api', 'Sub2API', '', '', '', '', '', '', TRUE, 10),
    ('newapi', 'NewAPI', '', '', '', '', '', '', TRUE, 20)
ON CONFLICT DO NOTHING;

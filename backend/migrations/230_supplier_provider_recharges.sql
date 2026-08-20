ALTER TABLE supplier_provider_types ADD COLUMN IF NOT EXISTS recharge_url TEXT NOT NULL DEFAULT '';
ALTER TABLE supplier_providers ADD COLUMN IF NOT EXISTS recharge_url TEXT NOT NULL DEFAULT '';

UPDATE supplier_provider_types
SET recharge_url = CASE code
    WHEN 'sub2api' THEN '/v1/redeem/history?timezone=Asia%2FShanghai'
    WHEN 'newapi' THEN '/api/log/self?p={page}&page_size={page_size}&type=1&start_timestamp={start_timestamp}&end_timestamp={end_timestamp}'
    ELSE recharge_url
END
WHERE recharge_url = ''
  AND code IN ('sub2api', 'newapi');

UPDATE supplier_providers p
SET recharge_url = t.recharge_url
FROM supplier_provider_types t
WHERE p.provider_type = t.code
  AND p.recharge_url = ''
  AND t.recharge_url <> '';

CREATE TABLE IF NOT EXISTS supplier_provider_recharges (
    id BIGSERIAL PRIMARY KEY,
    provider_id BIGINT NOT NULL REFERENCES supplier_providers(id) ON DELETE CASCADE,
    external_id VARCHAR(255) NOT NULL,
    external_code VARCHAR(255) NOT NULL DEFAULT '',
    recharge_type VARCHAR(64) NOT NULL DEFAULT '',
    amount NUMERIC(20, 6) NOT NULL DEFAULT 0,
    status VARCHAR(32) NOT NULL DEFAULT '',
    occurred_at TIMESTAMPTZ NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    source_payload JSONB NOT NULL DEFAULT '{}'::jsonb,
    synced_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT ck_supplier_provider_recharges_amount_nonnegative CHECK (amount >= 0),
    CONSTRAINT uq_supplier_provider_recharges_provider_external UNIQUE (provider_id, external_id)
);

CREATE INDEX IF NOT EXISTS idx_supplier_provider_recharges_provider_occurred
    ON supplier_provider_recharges (provider_id, occurred_at DESC, id DESC);
CREATE INDEX IF NOT EXISTS idx_supplier_provider_recharges_occurred
    ON supplier_provider_recharges (occurred_at DESC, id DESC);

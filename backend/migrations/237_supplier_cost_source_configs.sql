-- 供应商成本来源配置：全局默认来源 + 供应商级覆盖。
-- 来源模式：
--   auto       智能模式（现状行为）：待审批核对记录默认采用计算成本，
--              同步写入以上游接口成本为准，偏差超过阈值时改写为充值修正余额成本或本地成本。
--   upstream   始终以上游接口成本为准，不做偏差改写。
--   calculated 始终以本地计算成本为准，不做偏差改写。
ALTER TABLE supplier_cost_deviation_settings
    ADD COLUMN IF NOT EXISTS cost_source VARCHAR(16) NOT NULL DEFAULT 'auto';

DO $$
BEGIN
    ALTER TABLE supplier_cost_deviation_settings
        DROP CONSTRAINT IF EXISTS supplier_cost_deviation_settings_cost_source_ck;
    ALTER TABLE supplier_cost_deviation_settings
        ADD CONSTRAINT supplier_cost_deviation_settings_cost_source_ck CHECK (
            cost_source IN ('auto', 'upstream', 'calculated')
        );
END $$;

CREATE TABLE IF NOT EXISTS supplier_cost_source_configs (
    id BIGSERIAL PRIMARY KEY,
    provider_id BIGINT NOT NULL UNIQUE REFERENCES supplier_providers(id) ON DELETE CASCADE,
    cost_source VARCHAR(16) NOT NULL,
    threshold NUMERIC(20, 6) NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT supplier_cost_source_configs_source_ck CHECK (cost_source IN ('auto', 'upstream', 'calculated'))
);

CREATE INDEX IF NOT EXISTS idx_supplier_cost_source_configs_provider
    ON supplier_cost_source_configs (provider_id);

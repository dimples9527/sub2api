-- 供应商成本偏差覆盖阈值配置。
-- 独立于框架通用 settings 表，由供应商模块自己管理。
CREATE TABLE IF NOT EXISTS supplier_cost_deviation_settings (
    id INTEGER PRIMARY KEY CHECK (id = 1),
    threshold NUMERIC(6,4) NOT NULL DEFAULT 0.5000,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

INSERT INTO supplier_cost_deviation_settings (id, threshold)
VALUES (1, 0.5000)
ON CONFLICT (id) DO NOTHING;

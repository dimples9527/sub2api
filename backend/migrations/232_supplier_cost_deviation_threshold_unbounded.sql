-- 允许供应商成本偏差阈值保存 100% 以上的任意有限非负数。
ALTER TABLE supplier_cost_deviation_settings
    ALTER COLUMN threshold TYPE NUMERIC
    USING threshold;

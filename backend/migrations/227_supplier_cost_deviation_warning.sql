-- 成本偏差过大时记录原始上游成本与覆盖提示，便于展示时兜底改写。
ALTER TABLE supplier_provider_daily_stats
  ADD COLUMN IF NOT EXISTS raw_upstream_cost NUMERIC(20,6) NULL,
  ADD COLUMN IF NOT EXISTS cost_warning TEXT NULL;

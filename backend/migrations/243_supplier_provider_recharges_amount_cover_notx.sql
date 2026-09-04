CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_supplier_provider_recharges_provider_occurred_amount
    ON supplier_provider_recharges (provider_id, occurred_at DESC, id DESC)
    INCLUDE (amount);

DROP INDEX CONCURRENTLY IF EXISTS idx_supplier_provider_recharges_provider_occurred;

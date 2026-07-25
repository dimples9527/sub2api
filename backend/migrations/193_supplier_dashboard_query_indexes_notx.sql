CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_supplier_dashboard_accounts_normalized_name
    ON accounts ((regexp_replace(lower(name), '[^[:alnum:]]', '', 'g')))
    WHERE deleted_at IS NULL;

CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_supplier_dashboard_supplier_accounts_normalized_active
    ON supplier_provider_accounts (supplier_dashboard_normalized_effective_name, id)
    WHERE active = TRUE AND supplier_dashboard_normalized_effective_name IS NOT NULL;

CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_supplier_dashboard_supplier_accounts_provider_group
    ON supplier_provider_accounts (provider_id, group_key, id);

CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_supplier_dashboard_health_items_account_finished
    ON upstream_account_health_guard_run_items (account_id, finished_at DESC, id DESC);

CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_supplier_dashboard_sync_runs_provider_scope_finished
    ON supplier_provider_sync_runs (provider_id, sync_scope, finished_at DESC, id DESC)
    WHERE finished_at IS NOT NULL;

CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_supplier_dashboard_rate_changes_mapping_time
    ON supplier_rate_guard_change_logs (mapping_id, changed_at DESC, id DESC);
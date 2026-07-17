ALTER TABLE upstream_account_health_guard_runs
    ADD COLUMN IF NOT EXISTS skip_reasons JSONB NOT NULL DEFAULT '[]'::jsonb;

ALTER TABLE supplier_provider_groups
    ADD COLUMN IF NOT EXISTS rate_guard_ignored BOOLEAN NOT NULL DEFAULT FALSE;

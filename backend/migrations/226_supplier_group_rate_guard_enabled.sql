ALTER TABLE supplier_provider_groups
    ADD COLUMN IF NOT EXISTS rate_guard_enabled BOOLEAN NOT NULL DEFAULT TRUE;

UPDATE supplier_provider_groups
SET rate_guard_enabled = FALSE
WHERE rate_guard_ignored = TRUE;

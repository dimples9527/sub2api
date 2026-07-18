ALTER TABLE supplier_provider_groups
    ADD COLUMN IF NOT EXISTS local_group_id BIGINT NULL REFERENCES groups(id) ON DELETE SET NULL;

CREATE INDEX IF NOT EXISTS idx_supplier_provider_groups_local_group
    ON supplier_provider_groups (local_group_id)
    WHERE local_group_id IS NOT NULL;

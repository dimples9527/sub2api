ALTER TABLE supplier_provider_groups
    ADD COLUMN IF NOT EXISTS auto_match_ignored BOOLEAN NOT NULL DEFAULT FALSE,
    ADD COLUMN IF NOT EXISTS auto_match_status VARCHAR(32) NOT NULL DEFAULT 'unmatched',
    ADD COLUMN IF NOT EXISTS matched_upstream_name VARCHAR(255) NULL,
    ADD COLUMN IF NOT EXISTS name_change_pending BOOLEAN NOT NULL DEFAULT FALSE;

UPDATE supplier_provider_groups
SET auto_match_status = CASE
        WHEN local_group_id IS NOT NULL THEN 'manual'
        ELSE 'unmatched'
    END,
    matched_upstream_name = CASE
        WHEN local_group_id IS NOT NULL THEN name
        ELSE NULL
    END,
    name_change_pending = FALSE;

CREATE INDEX IF NOT EXISTS idx_supplier_provider_groups_auto_match
    ON supplier_provider_groups (provider_id, active, auto_match_ignored, auto_match_status)
    WHERE local_group_id IS NULL;

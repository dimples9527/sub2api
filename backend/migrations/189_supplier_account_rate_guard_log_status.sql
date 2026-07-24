ALTER TABLE supplier_account_rate_guard_unbind_logs
    ADD COLUMN IF NOT EXISTS status VARCHAR(16) NOT NULL DEFAULT 'handled',
    ADD COLUMN IF NOT EXISTS handled_at TIMESTAMPTZ NULL;

UPDATE supplier_account_rate_guard_unbind_logs
SET status = CASE
        WHEN result = 'unbound' THEN 'pending'
        ELSE 'handled'
    END,
    handled_at = CASE
        WHEN result = 'unbound' THEN NULL
        ELSE COALESCE(handled_at, NOW())
    END
WHERE status = 'handled' AND handled_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_supplier_account_rate_guard_logs_status_created
    ON supplier_account_rate_guard_unbind_logs (status, created_at DESC, id DESC);

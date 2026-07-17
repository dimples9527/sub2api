ALTER TABLE supplier_automation_runs
    ADD COLUMN IF NOT EXISTS result_detail JSONB NULL;

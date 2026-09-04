ALTER TABLE supplier_provider_cost_reviews
    ADD COLUMN IF NOT EXISTS local_cost NUMERIC(20, 6) NULL;

ALTER TABLE supplier_provider_cost_review_histories
    ADD COLUMN IF NOT EXISTS local_cost NUMERIC(20, 6) NULL;

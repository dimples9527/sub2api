CREATE TABLE IF NOT EXISTS supplier_provider_cost_reviews (
    id BIGSERIAL PRIMARY KEY,
    provider_id BIGINT NOT NULL REFERENCES supplier_providers(id) ON DELETE CASCADE,
    stat_date DATE NOT NULL,
    upstream_cost NUMERIC(20, 6) NULL,
    calculated_cost NUMERIC(20, 6) NULL,
    auto_adopted_cost NUMERIC(20, 6) NULL,
    final_cost NUMERIC(20, 6) NULL,
    effective_cost NUMERIC(20, 6) NOT NULL DEFAULT 0,
    cost_delta NUMERIC(20, 6) NULL,
    effective_delta NUMERIC(20, 6) NULL,
    status VARCHAR(32) NOT NULL DEFAULT 'pending_review',
    decision_type VARCHAR(32) NOT NULL DEFAULT 'none',
    approved_by BIGINT NULL,
    approved_at TIMESTAMPTZ NULL,
    sync_count INTEGER NOT NULL DEFAULT 0 CHECK (sync_count >= 0),
    last_sync_run_id BIGINT NULL,
    last_synced_at TIMESTAMPTZ NULL,
    version BIGINT NOT NULL DEFAULT 1 CHECK (version > 0),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT uq_supplier_provider_cost_reviews_date UNIQUE (provider_id, stat_date),
    CONSTRAINT ck_supplier_provider_cost_reviews_status CHECK (status IN ('pending_review', 'approved', 'changed_after_approval')),
    CONSTRAINT ck_supplier_provider_cost_reviews_decision CHECK (decision_type IN ('none', 'upstream', 'calculated', 'manual'))
);

CREATE INDEX IF NOT EXISTS idx_supplier_provider_cost_reviews_date
    ON supplier_provider_cost_reviews (stat_date DESC, provider_id);
CREATE INDEX IF NOT EXISTS idx_supplier_provider_cost_reviews_status
    ON supplier_provider_cost_reviews (status, stat_date DESC);

CREATE TABLE IF NOT EXISTS supplier_provider_cost_review_histories (
    id BIGSERIAL PRIMARY KEY,
    review_id BIGINT NULL REFERENCES supplier_provider_cost_reviews(id) ON DELETE SET NULL,
    provider_id BIGINT NOT NULL REFERENCES supplier_providers(id) ON DELETE CASCADE,
    stat_date DATE NOT NULL,
    event_type VARCHAR(16) NOT NULL,
    sync_run_id BIGINT NULL,
    upstream_cost NUMERIC(20, 6) NULL,
    calculated_cost NUMERIC(20, 6) NULL,
    auto_adopted_cost NUMERIC(20, 6) NULL,
    final_cost NUMERIC(20, 6) NULL,
    cost_delta NUMERIC(20, 6) NULL,
    effective_delta NUMERIC(20, 6) NULL,
    status VARCHAR(32) NOT NULL,
    decision_type VARCHAR(32) NOT NULL,
    manual_cost NUMERIC(20, 6) NULL,
    operator_id BIGINT NULL,
    operated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT ck_supplier_provider_cost_review_histories_event CHECK (event_type IN ('sync', 'approve'))
);

CREATE INDEX IF NOT EXISTS idx_supplier_provider_cost_review_histories_review
    ON supplier_provider_cost_review_histories (review_id, operated_at DESC);
CREATE INDEX IF NOT EXISTS idx_supplier_provider_cost_review_histories_provider_date
    ON supplier_provider_cost_review_histories (provider_id, stat_date, operated_at DESC);

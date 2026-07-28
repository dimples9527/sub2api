CREATE TABLE IF NOT EXISTS supplier_local_account_platform_overrides (
    id BIGSERIAL PRIMARY KEY,
    local_account_id BIGINT NOT NULL,
    platform VARCHAR(50) NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (local_account_id),
    CONSTRAINT fk_supplier_local_account_platform_overrides_local_account
        FOREIGN KEY (local_account_id)
        REFERENCES accounts(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_supplier_local_account_platform_overrides_platform
    ON supplier_local_account_platform_overrides(platform);

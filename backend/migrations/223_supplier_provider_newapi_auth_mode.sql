ALTER TABLE supplier_providers
    ADD COLUMN IF NOT EXISTS newapi_auth_mode VARCHAR(32) NOT NULL DEFAULT 'auto';

ALTER TABLE supplier_providers
    DROP CONSTRAINT IF EXISTS supplier_providers_newapi_auth_mode_check;

ALTER TABLE supplier_providers
    ADD CONSTRAINT supplier_providers_newapi_auth_mode_check
    CHECK (newapi_auth_mode IN ('auto', 'cookie_session', 'access_token_refresh'));

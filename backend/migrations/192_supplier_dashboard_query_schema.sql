ALTER TABLE supplier_provider_accounts
    ADD COLUMN IF NOT EXISTS supplier_dashboard_normalized_effective_name TEXT NULL;

ALTER TABLE supplier_provider_runtime_stats
    ADD COLUMN IF NOT EXISTS risk_updated_at TIMESTAMPTZ NULL;

CREATE OR REPLACE FUNCTION public.supplier_dashboard_refresh_account_normalized_name()
RETURNS TRIGGER
LANGUAGE plpgsql
AS $$
DECLARE
    provider_prefix TEXT;
BEGIN
    IF NEW.active IS DISTINCT FROM TRUE THEN
        NEW.supplier_dashboard_normalized_effective_name := NULL;
        RETURN NEW;
    END IF;

    SELECT provider.account_name_prefix
    INTO provider_prefix
    FROM supplier_providers AS provider
    WHERE provider.id = NEW.provider_id
      AND provider.deleted_at IS NULL;

    IF NOT FOUND THEN
        NEW.supplier_dashboard_normalized_effective_name := NULL;
    ELSE
        NEW.supplier_dashboard_normalized_effective_name := regexp_replace(
            lower(COALESCE(provider_prefix, '') || COALESCE(NEW.name, '')),
            '[^[:alnum:]]',
            '',
            'g'
        );
    END IF;
    RETURN NEW;
END;
$$;

CREATE OR REPLACE FUNCTION public.supplier_dashboard_refresh_provider_account_names()
RETURNS TRIGGER
LANGUAGE plpgsql
AS $$
BEGIN
    UPDATE supplier_provider_accounts AS account
    SET supplier_dashboard_normalized_effective_name = CASE
        WHEN NEW.deleted_at IS NULL AND account.active = TRUE THEN regexp_replace(
            lower(COALESCE(NEW.account_name_prefix, '') || COALESCE(account.name, '')),
            '[^[:alnum:]]',
            '',
            'g'
        )
        ELSE NULL
    END
    WHERE account.provider_id = NEW.id;
    RETURN NULL;
END;
$$;

CREATE OR REPLACE FUNCTION public.supplier_dashboard_track_provider_risk_updated_at()
RETURNS TRIGGER
LANGUAGE plpgsql
AS $$
BEGIN
    IF TG_OP = 'INSERT' THEN
        NEW.risk_updated_at := COALESCE(NEW.risk_updated_at, NEW.updated_at, CURRENT_TIMESTAMP);
    ELSIF NEW.risk_level IS DISTINCT FROM OLD.risk_level
       OR NEW.rate_risk_count IS DISTINCT FROM OLD.rate_risk_count THEN
        NEW.risk_updated_at := clock_timestamp();
    ELSE
        NEW.risk_updated_at := OLD.risk_updated_at;
    END IF;
    RETURN NEW;
END;
$$;

DROP TRIGGER IF EXISTS supplier_dashboard_refresh_account_normalized_name ON supplier_provider_accounts;
CREATE TRIGGER supplier_dashboard_refresh_account_normalized_name
BEFORE INSERT OR UPDATE OF provider_id, name, active
ON supplier_provider_accounts
FOR EACH ROW
EXECUTE FUNCTION public.supplier_dashboard_refresh_account_normalized_name();

DROP TRIGGER IF EXISTS supplier_dashboard_refresh_provider_account_names ON supplier_providers;
CREATE TRIGGER supplier_dashboard_refresh_provider_account_names
AFTER UPDATE OF account_name_prefix, deleted_at
ON supplier_providers
FOR EACH ROW
WHEN (
    OLD.account_name_prefix IS DISTINCT FROM NEW.account_name_prefix
    OR OLD.deleted_at IS DISTINCT FROM NEW.deleted_at
)
EXECUTE FUNCTION public.supplier_dashboard_refresh_provider_account_names();

DROP TRIGGER IF EXISTS supplier_dashboard_track_provider_risk_updated_at ON supplier_provider_runtime_stats;
CREATE TRIGGER supplier_dashboard_track_provider_risk_updated_at
BEFORE INSERT OR UPDATE OF risk_level, rate_risk_count
ON supplier_provider_runtime_stats
FOR EACH ROW
EXECUTE FUNCTION public.supplier_dashboard_track_provider_risk_updated_at();

UPDATE supplier_provider_accounts AS account
SET supplier_dashboard_normalized_effective_name = CASE
    WHEN provider.deleted_at IS NULL AND account.active = TRUE THEN regexp_replace(
        lower(COALESCE(provider.account_name_prefix, '') || COALESCE(account.name, '')),
        '[^[:alnum:]]',
        '',
        'g'
    )
    ELSE NULL
END
FROM supplier_providers AS provider
WHERE provider.id = account.provider_id
  AND account.supplier_dashboard_normalized_effective_name IS DISTINCT FROM CASE
      WHEN provider.deleted_at IS NULL AND account.active = TRUE THEN regexp_replace(
          lower(COALESCE(provider.account_name_prefix, '') || COALESCE(account.name, '')),
          '[^[:alnum:]]',
          '',
          'g'
      )
      ELSE NULL
  END;

UPDATE supplier_provider_runtime_stats
SET risk_updated_at = updated_at
WHERE risk_updated_at IS NULL;
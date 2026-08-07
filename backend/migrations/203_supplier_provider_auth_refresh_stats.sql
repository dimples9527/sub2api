ALTER TABLE supplier_provider_runtime_stats
    ADD COLUMN IF NOT EXISTS auth_refresh_count BIGINT NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS auth_refresh_success_count BIGINT NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS auth_refresh_failure_count BIGINT NOT NULL DEFAULT 0;

WITH event_summary AS (
    SELECT
        provider_id,
        COUNT(*) FILTER (WHERE event_type IN ('refresh_success', 'refresh_failed'))::BIGINT AS refresh_count,
        COUNT(*) FILTER (WHERE event_type = 'refresh_success')::BIGINT AS refresh_success_count,
        COUNT(*) FILTER (WHERE event_type = 'refresh_failed')::BIGINT AS refresh_failure_count
    FROM supplier_provider_auth_events
    GROUP BY provider_id
)
UPDATE supplier_provider_runtime_stats stats
SET auth_refresh_count = event_summary.refresh_count,
    auth_refresh_success_count = event_summary.refresh_success_count,
    auth_refresh_failure_count = event_summary.refresh_failure_count
FROM event_summary
WHERE stats.provider_id = event_summary.provider_id;

WITH cache_hit_summary AS (
    SELECT
        provider_id,
        COUNT(*)::BIGINT AS cache_hit_count,
        MAX(created_at) AS last_cache_hit_at
    FROM supplier_provider_auth_events
    WHERE event_type = 'cache_hit'
    GROUP BY provider_id
)
UPDATE supplier_provider_runtime_stats stats
SET auth_login_count = stats.auth_login_count + cache_hit_summary.cache_hit_count,
    auth_login_success_count = stats.auth_login_success_count + cache_hit_summary.cache_hit_count,
    auth_last_login_at = CASE
        WHEN cache_hit_summary.last_cache_hit_at > COALESCE(stats.auth_last_login_at, '-infinity'::timestamptz)
            THEN cache_hit_summary.last_cache_hit_at
        ELSE stats.auth_last_login_at
    END,
    auth_last_login_status = CASE
        WHEN cache_hit_summary.last_cache_hit_at > COALESCE(stats.auth_last_login_at, '-infinity'::timestamptz)
            THEN 'success'
        ELSE stats.auth_last_login_status
    END,
    auth_last_login_error = CASE
        WHEN cache_hit_summary.last_cache_hit_at > COALESCE(stats.auth_last_login_at, '-infinity'::timestamptz)
            THEN ''
        ELSE stats.auth_last_login_error
    END
FROM cache_hit_summary
WHERE stats.provider_id = cache_hit_summary.provider_id;

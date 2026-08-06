-- 按事件明细重建供应商认证汇总统计。
-- 登录次数只统计 login_success / login_failed；
-- 缓存命中/未命中只统计 cache_hit / cache_miss。
-- 用于纠正 200 迁移把 cache_hit 回填进登录次数的错误语义。
WITH event_summary AS (
    SELECT
        provider_id,
        COUNT(*) FILTER (WHERE event_type IN ('login_success', 'login_failed'))::BIGINT AS login_count,
        COUNT(*) FILTER (WHERE event_type = 'login_success')::BIGINT AS login_success_count,
        COUNT(*) FILTER (WHERE event_type = 'login_failed')::BIGINT AS login_failure_count,
        COUNT(*) FILTER (WHERE event_type = 'cache_hit')::BIGINT AS cache_hit_count,
        COUNT(*) FILTER (WHERE event_type = 'cache_miss')::BIGINT AS cache_miss_count,
        MAX(finished_at) FILTER (WHERE event_type IN ('login_success', 'login_failed')) AS last_login_at,
        (ARRAY_AGG(status ORDER BY finished_at DESC, id DESC) FILTER (WHERE event_type IN ('login_success', 'login_failed')))[1] AS last_login_status,
        COALESCE((ARRAY_AGG(NULLIF(error_message, '') ORDER BY finished_at DESC, id DESC) FILTER (WHERE event_type IN ('login_success', 'login_failed')))[1], '') AS last_login_error,
        MAX(finished_at) FILTER (WHERE event_type = 'cache_hit') AS last_cache_hit_at,
        COALESCE((ARRAY_AGG(NULLIF(error_message, '') ORDER BY finished_at DESC, id DESC) FILTER (WHERE event_type = 'cache_error'))[1], '') AS last_cache_error,
        MAX(token_expires_at) FILTER (WHERE token_expires_at IS NOT NULL) AS last_token_expires_at,
        COALESCE((ARRAY_AGG(NULLIF(token_fingerprint, '') ORDER BY finished_at DESC, id DESC) FILTER (WHERE COALESCE(token_fingerprint, '') <> ''))[1], '') AS last_token_fingerprint
    FROM supplier_provider_auth_events
    GROUP BY provider_id
)
UPDATE supplier_provider_runtime_stats stats
SET auth_login_count = event_summary.login_count,
    auth_login_success_count = event_summary.login_success_count,
    auth_login_failure_count = event_summary.login_failure_count,
    auth_cache_hit_count = event_summary.cache_hit_count,
    auth_cache_miss_count = event_summary.cache_miss_count,
    auth_last_login_at = event_summary.last_login_at,
    auth_last_login_status = COALESCE(event_summary.last_login_status, ''),
    auth_last_login_error = COALESCE(event_summary.last_login_error, ''),
    auth_last_cache_hit_at = event_summary.last_cache_hit_at,
    auth_last_cache_error = COALESCE(event_summary.last_cache_error, ''),
    auth_last_token_expires_at = COALESCE(event_summary.last_token_expires_at, stats.auth_last_token_expires_at),
    auth_last_token_fingerprint = CASE
        WHEN COALESCE(event_summary.last_token_fingerprint, '') <> '' THEN event_summary.last_token_fingerprint
        ELSE stats.auth_last_token_fingerprint
    END,
    updated_at = NOW()
FROM event_summary
WHERE stats.provider_id = event_summary.provider_id;

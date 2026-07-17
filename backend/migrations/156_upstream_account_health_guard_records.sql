CREATE TABLE IF NOT EXISTS upstream_account_health_guard_runs (
    id TEXT PRIMARY KEY,
    trigger VARCHAR(32) NOT NULL,
    status VARCHAR(32) NOT NULL,
    message TEXT NOT NULL DEFAULT '',
    started_at TIMESTAMPTZ NOT NULL,
    finished_at TIMESTAMPTZ NOT NULL,
    total_accounts INTEGER NOT NULL DEFAULT 0,
    checked_count INTEGER NOT NULL DEFAULT 0,
    healthy_count INTEGER NOT NULL DEFAULT 0,
    slow_count INTEGER NOT NULL DEFAULT 0,
    failed_count INTEGER NOT NULL DEFAULT 0,
    skipped_count INTEGER NOT NULL DEFAULT 0,
    disabled_count INTEGER NOT NULL DEFAULT 0,
    recovered_count INTEGER NOT NULL DEFAULT 0,
    unchanged_count INTEGER NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_upstream_account_health_guard_runs_finished
    ON upstream_account_health_guard_runs (finished_at DESC, created_at DESC);

CREATE TABLE IF NOT EXISTS upstream_account_health_guard_run_items (
    id BIGSERIAL PRIMARY KEY,
    run_id TEXT NOT NULL REFERENCES upstream_account_health_guard_runs(id) ON DELETE CASCADE,
    account_id BIGINT NOT NULL,
    account_name VARCHAR(200) NOT NULL DEFAULT '',
    platform VARCHAR(32) NOT NULL DEFAULT '',
    provider_slug VARCHAR(64) NOT NULL DEFAULT '',
    provider_name VARCHAR(100) NOT NULL DEFAULT '',
    model_id VARCHAR(200) NOT NULL DEFAULT '',
    schedulable_before BOOLEAN NOT NULL DEFAULT FALSE,
    schedulable_after BOOLEAN NOT NULL DEFAULT FALSE,
    status VARCHAR(32) NOT NULL DEFAULT '',
    test_status VARCHAR(32) NOT NULL DEFAULT '',
    latency_ms BIGINT NOT NULL DEFAULT 0,
    latency_limit_ms BIGINT NOT NULL DEFAULT 0,
    consecutive_failed INTEGER NOT NULL DEFAULT 0,
    consecutive_slow INTEGER NOT NULL DEFAULT 0,
    consecutive_healthy INTEGER NOT NULL DEFAULT 0,
    action VARCHAR(32) NOT NULL DEFAULT '',
    reason TEXT NOT NULL DEFAULT '',
    error_message TEXT NOT NULL DEFAULT '',
    started_at TIMESTAMPTZ NOT NULL,
    finished_at TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (run_id, account_id, finished_at)
);

CREATE INDEX IF NOT EXISTS idx_upstream_account_health_guard_items_run
    ON upstream_account_health_guard_run_items (run_id, provider_slug, account_id);

CREATE INDEX IF NOT EXISTS idx_upstream_account_health_guard_items_status
    ON upstream_account_health_guard_run_items (status, action);

WITH legacy_source AS (
    SELECT value::jsonb AS data
    FROM settings
    WHERE key = 'upstream_account_health_guard_records'
),
legacy_records AS (
    SELECT record
    FROM legacy_source,
         LATERAL jsonb_array_elements(
             CASE
                 WHEN jsonb_typeof(data) = 'array' THEN data
                 ELSE '[]'::jsonb
             END
         ) AS record
)
INSERT INTO upstream_account_health_guard_runs (
    id, trigger, status, message, started_at, finished_at,
    total_accounts, checked_count, healthy_count, slow_count, failed_count,
    skipped_count, disabled_count, recovered_count, unchanged_count, created_at
)
SELECT
    record->>'id',
    COALESCE(NULLIF(record->>'trigger', ''), 'scheduled'),
    COALESCE(NULLIF(record->>'status', ''), 'success'),
    COALESCE(record->>'message', ''),
    COALESCE(NULLIF(record->>'started_at', '')::TIMESTAMPTZ, NOW()),
    COALESCE(NULLIF(record->>'finished_at', '')::TIMESTAMPTZ, NOW()),
    COALESCE(NULLIF(record->'summary'->>'total_accounts', '')::INTEGER, 0),
    COALESCE(NULLIF(record->'summary'->>'checked_count', '')::INTEGER, 0),
    COALESCE(NULLIF(record->'summary'->>'healthy_count', '')::INTEGER, 0),
    COALESCE(NULLIF(record->'summary'->>'slow_count', '')::INTEGER, 0),
    COALESCE(NULLIF(record->'summary'->>'failed_count', '')::INTEGER, 0),
    COALESCE(NULLIF(record->'summary'->>'skipped_count', '')::INTEGER, 0),
    COALESCE(NULLIF(record->'summary'->>'disabled_count', '')::INTEGER, 0),
    COALESCE(NULLIF(record->'summary'->>'recovered_count', '')::INTEGER, 0),
    COALESCE(NULLIF(record->'summary'->>'unchanged_count', '')::INTEGER, 0),
    NOW()
FROM legacy_records
WHERE NULLIF(record->>'id', '') IS NOT NULL
ON CONFLICT (id) DO NOTHING;

WITH legacy_source AS (
    SELECT value::jsonb AS data
    FROM settings
    WHERE key = 'upstream_account_health_guard_records'
),
legacy_records AS (
    SELECT record
    FROM legacy_source,
         LATERAL jsonb_array_elements(
             CASE
                 WHEN jsonb_typeof(data) = 'array' THEN data
                 ELSE '[]'::jsonb
             END
         ) AS record
),
legacy_items AS (
    SELECT record->>'id' AS run_id, item
    FROM legacy_records,
         LATERAL jsonb_array_elements(
             CASE
                 WHEN jsonb_typeof(record->'items') = 'array' THEN record->'items'
                 ELSE '[]'::jsonb
             END
         ) AS item
    WHERE NULLIF(record->>'id', '') IS NOT NULL
)
INSERT INTO upstream_account_health_guard_run_items (
    run_id, account_id, account_name, platform, provider_slug, provider_name,
    model_id, schedulable_before, schedulable_after, status, test_status,
    latency_ms, latency_limit_ms, consecutive_failed, consecutive_slow,
    consecutive_healthy, action, reason, error_message, started_at, finished_at, created_at
)
SELECT
    run_id,
    COALESCE(NULLIF(item->>'account_id', '')::BIGINT, 0),
    LEFT(COALESCE(item->>'account_name', ''), 200),
    LEFT(COALESCE(item->>'platform', ''), 32),
    LEFT(COALESCE(item->>'provider_slug', ''), 64),
    LEFT(COALESCE(item->>'provider_name', ''), 100),
    LEFT(COALESCE(item->>'model_id', ''), 200),
    COALESCE(NULLIF(item->>'schedulable_before', '')::BOOLEAN, FALSE),
    COALESCE(NULLIF(item->>'schedulable_after', '')::BOOLEAN, FALSE),
    LEFT(COALESCE(item->>'status', ''), 32),
    LEFT(COALESCE(item->>'test_status', ''), 32),
    COALESCE(NULLIF(item->>'latency_ms', '')::BIGINT, 0),
    COALESCE(NULLIF(item->>'latency_limit_ms', '')::BIGINT, 0),
    COALESCE(NULLIF(item->>'consecutive_failed', '')::INTEGER, 0),
    COALESCE(NULLIF(item->>'consecutive_slow', '')::INTEGER, 0),
    COALESCE(NULLIF(item->>'consecutive_healthy', '')::INTEGER, 0),
    LEFT(COALESCE(item->>'action', ''), 32),
    COALESCE(item->>'reason', ''),
    COALESCE(item->>'error_message', ''),
    COALESCE(NULLIF(item->>'started_at', '')::TIMESTAMPTZ, NOW()),
    COALESCE(NULLIF(item->>'finished_at', '')::TIMESTAMPTZ, NOW()),
    NOW()
FROM legacy_items
ON CONFLICT (run_id, account_id, finished_at) DO NOTHING;

DELETE FROM settings
WHERE key = 'upstream_account_health_guard_records';

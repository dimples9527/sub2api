package repository

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/lib/pq"
)

// modelMonitorGroupHealthRepository 查询模型监控分组健康趋势所需的原始聚合数据。
type modelMonitorGroupHealthRepository struct {
	db *sql.DB
}

// NewModelMonitorGroupHealthRepository 创建模型监控分组健康趋势仓储。
func NewModelMonitorGroupHealthRepository(db *sql.DB) service.ModelMonitorGroupHealthRepository {
	return &modelMonitorGroupHealthRepository{db: db}
}

const modelMonitorGroupHealthEffectivePlatformSQL = `COALESCE(NULLIF(LOWER(TRIM(o.actual_platform)), ''), LOWER(TRIM(g.platform)))`

func (r *modelMonitorGroupHealthRepository) ListGroups(ctx context.Context, groupIDs []int64, platform string) ([]service.ModelMonitorGroupHealthGroup, error) {
	if r == nil || r.db == nil {
		return nil, fmt.Errorf("model monitor group health repository database is not initialized")
	}

	where := []string{
		"g.status = 'active'",
		"g.deleted_at IS NULL",
		"COALESCE(o.show_in_monitor, TRUE)",
	}
	args := make([]any, 0, 2)
	if len(groupIDs) > 0 {
		args = append(args, pq.Array(groupIDs))
		where = append(where, "g.id = ANY($1)")
	}
	if platform != "" {
		args = append(args, platform)
		where = append(where, fmt.Sprintf("%s = $%d", modelMonitorGroupHealthEffectivePlatformSQL, len(args)))
	}

	query := `
SELECT g.id,
       COALESCE(g.name, ''),
       LOWER(COALESCE(NULLIF(TRIM(g.platform), ''), '')),
       ` + modelMonitorGroupHealthEffectivePlatformSQL + `
FROM groups g
LEFT JOIN monitor_group_platform_overrides o ON o.group_id = g.id
WHERE ` + strings.Join(where, " AND ") + `
ORDER BY g.id`

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	groups := make([]service.ModelMonitorGroupHealthGroup, 0)
	for rows.Next() {
		var group service.ModelMonitorGroupHealthGroup
		if err := rows.Scan(&group.ID, &group.Name, &group.Platform, &group.EffectivePlatform); err != nil {
			return nil, err
		}
		groups = append(groups, group)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return groups, nil
}

func (r *modelMonitorGroupHealthRepository) ListUsageBuckets(ctx context.Context, startTime, endTime time.Time, bucketInterval time.Duration, groupIDs []int64, platform string) ([]service.ModelMonitorGroupHealthBucket, error) {
	if r == nil || r.db == nil {
		return nil, fmt.Errorf("model monitor group health repository database is not initialized")
	}
	interval := modelMonitorGroupHealthInterval(bucketInterval)

	query := `
SELECT ul.group_id,
       date_bin($1::interval, ul.created_at, TIMESTAMPTZ '1970-01-01') AS bucket_start,
       COUNT(*)::bigint AS success_count,
       COUNT(ul.duration_ms)::bigint AS latency_sample_count,
       COALESCE(AVG(ul.duration_ms) FILTER (WHERE ul.duration_ms IS NOT NULL), 0)::double precision AS avg_latency_ms,
       COALESCE(percentile_cont(0.95) WITHIN GROUP (ORDER BY ul.duration_ms) FILTER (WHERE ul.duration_ms IS NOT NULL), 0)::double precision AS p95_latency_ms,
       COALESCE(percentile_cont(0.95) WITHIN GROUP (ORDER BY ul.first_token_ms) FILTER (WHERE ul.first_token_ms IS NOT NULL), 0)::double precision AS p95_first_token_ms,
       MAX(ul.created_at) AS last_request_at
FROM usage_logs ul
JOIN groups g ON g.id = ul.group_id
LEFT JOIN monitor_group_platform_overrides o ON o.group_id = ul.group_id
WHERE ul.created_at >= $2
  AND ul.created_at < $3
  AND ul.group_id = ANY($4)
  AND g.status = 'active'
  AND g.deleted_at IS NULL
  AND COALESCE(o.show_in_monitor, TRUE)
  AND ($5 = '' OR ` + modelMonitorGroupHealthEffectivePlatformSQL + ` = $5)
GROUP BY ul.group_id, date_bin($1::interval, ul.created_at, TIMESTAMPTZ '1970-01-01')
ORDER BY ul.group_id, bucket_start`

	rows, err := r.db.QueryContext(ctx, query, interval, startTime.UTC(), endTime.UTC(), pq.Array(groupIDs), platform)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	buckets := make([]service.ModelMonitorGroupHealthBucket, 0)
	for rows.Next() {
		var bucket service.ModelMonitorGroupHealthBucket
		var lastRequestAt sql.NullTime
		if err := rows.Scan(
			&bucket.GroupID,
			&bucket.BucketStart,
			&bucket.SuccessCount,
			&bucket.LatencySampleCount,
			&bucket.AvgLatencyMS,
			&bucket.P95LatencyMS,
			&bucket.P95FirstTokenMS,
			&lastRequestAt,
		); err != nil {
			return nil, err
		}
		if lastRequestAt.Valid {
			lastRequest := lastRequestAt.Time.UTC()
			bucket.LastRequestAt = &lastRequest
		}
		buckets = append(buckets, bucket)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return buckets, nil
}

func (r *modelMonitorGroupHealthRepository) ListErrorBuckets(ctx context.Context, startTime, endTime time.Time, bucketInterval time.Duration, groupIDs []int64, platform string) ([]service.ModelMonitorGroupHealthBucket, error) {
	if r == nil || r.db == nil {
		return nil, fmt.Errorf("model monitor group health repository database is not initialized")
	}
	interval := modelMonitorGroupHealthInterval(bucketInterval)

	query := `
SELECT oel.group_id,
       date_bin($1::interval, oel.created_at, TIMESTAMPTZ '1970-01-01') AS bucket_start,
       COUNT(*)::bigint AS error_count,
       COUNT(*) FILTER (WHERE oel.is_business_limited)::bigint AS business_limited_count,
       MAX(oel.created_at) AS last_request_at
FROM ops_error_logs oel
JOIN groups g ON g.id = oel.group_id
LEFT JOIN monitor_group_platform_overrides o ON o.group_id = oel.group_id
WHERE oel.created_at >= $2
  AND oel.created_at < $3
  AND oel.group_id = ANY($4)
  AND oel.is_count_tokens = FALSE
  AND COALESCE(oel.status_code, 0) >= 400
  AND g.status = 'active'
  AND g.deleted_at IS NULL
  AND COALESCE(o.show_in_monitor, TRUE)
  AND ($5 = '' OR ` + modelMonitorGroupHealthEffectivePlatformSQL + ` = $5)
GROUP BY oel.group_id, date_bin($1::interval, oel.created_at, TIMESTAMPTZ '1970-01-01')
ORDER BY oel.group_id, bucket_start`

	rows, err := r.db.QueryContext(ctx, query, interval, startTime.UTC(), endTime.UTC(), pq.Array(groupIDs), platform)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	buckets := make([]service.ModelMonitorGroupHealthBucket, 0)
	for rows.Next() {
		var bucket service.ModelMonitorGroupHealthBucket
		var lastRequestAt sql.NullTime
		if err := rows.Scan(&bucket.GroupID, &bucket.BucketStart, &bucket.ErrorCount, &bucket.BusinessLimitedCount, &lastRequestAt); err != nil {
			return nil, err
		}
		if lastRequestAt.Valid {
			lastRequest := lastRequestAt.Time.UTC()
			bucket.LastRequestAt = &lastRequest
		}
		buckets = append(buckets, bucket)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return buckets, nil
}

func (r *modelMonitorGroupHealthRepository) ListErrorCategories(ctx context.Context, startTime, endTime time.Time, groupIDs []int64, platform string) ([]service.ModelMonitorGroupHealthErrorCount, error) {
	if r == nil || r.db == nil {
		return nil, fmt.Errorf("model monitor group health repository database is not initialized")
	}

	query := `
WITH classified AS (
	SELECT oel.group_id,
	       CASE
			WHEN oel.is_business_limited THEN 'business_limited'
			WHEN LOWER(COALESCE(oel.network_error_type, '')) LIKE '%timeout%'
				OR LOWER(COALESCE(oel.error_phase, '')) = 'network'
				OR LOWER(COALESCE(oel.error_type, '')) IN ('timeout', 'network_timeout', 'context_deadline_exceeded')
				THEN 'network_timeout'
			WHEN (LOWER(COALESCE(oel.error_phase, '')) IN ('auth', 'account_auth')
					AND LOWER(COALESCE(oel.error_owner, '')) = 'provider')
				OR LOWER(COALESCE(oel.error_owner, '')) = 'account'
				OR LOWER(COALESCE(oel.error_type, '')) IN ('account_auth', 'authentication_error', 'invalid_api_key')
				THEN 'account_auth'
			WHEN LOWER(COALESCE(oel.error_phase, '')) = 'routing'
				OR LOWER(COALESCE(oel.error_type, '')) LIKE '%routing%'
				OR LOWER(COALESCE(oel.error_type, '')) LIKE '%route%'
				THEN 'routing'
			WHEN COALESCE(oel.upstream_status_code, oel.status_code, 0) = 429
				OR LOWER(COALESCE(oel.error_type, '')) LIKE '%rate_limit%'
				OR LOWER(COALESCE(oel.error_type, '')) LIKE '%ratelimit%'
				THEN 'upstream_rate_limit'
			WHEN LOWER(COALESCE(oel.error_phase, '')) = 'upstream'
				OR LOWER(COALESCE(oel.error_owner, '')) = 'provider'
				OR LOWER(COALESCE(oel.error_source, '')) = 'upstream_http'
				THEN 'upstream_error'
			WHEN LOWER(COALESCE(oel.error_phase, '')) IN ('request', 'auth')
				OR LOWER(COALESCE(oel.error_owner, '')) = 'client'
				THEN 'client_request'
			ELSE 'other'
		       END AS category
	FROM ops_error_logs oel
	JOIN groups g ON g.id = oel.group_id
	LEFT JOIN monitor_group_platform_overrides o ON o.group_id = oel.group_id
	WHERE oel.created_at >= $1
	  AND oel.created_at < $2
  AND oel.group_id = ANY($3)
  AND oel.is_count_tokens = FALSE
  AND COALESCE(oel.status_code, 0) >= 400
	  AND g.status = 'active'
	  AND g.deleted_at IS NULL
	  AND COALESCE(o.show_in_monitor, TRUE)
	  AND ($4 = '' OR ` + modelMonitorGroupHealthEffectivePlatformSQL + ` = $4)
)
SELECT group_id, category, COUNT(*)::bigint
FROM classified
GROUP BY group_id, category
ORDER BY group_id, COUNT(*) DESC, category`

	rows, err := r.db.QueryContext(ctx, query, startTime.UTC(), endTime.UTC(), pq.Array(groupIDs), platform)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	items := make([]service.ModelMonitorGroupHealthErrorCount, 0)
	for rows.Next() {
		var item service.ModelMonitorGroupHealthErrorCount
		if err := rows.Scan(&item.GroupID, &item.Category, &item.Count); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}

func modelMonitorGroupHealthUnitString(value int64, unit string) string {
	if value == 1 {
		return fmt.Sprintf("1 %s", unit)
	}
	return fmt.Sprintf("%d %ss", value, unit)
}

func modelMonitorGroupHealthInterval(interval time.Duration) string {
	if interval <= 0 {
		interval = time.Hour
	}
	seconds := int64(interval / time.Second)
	if seconds <= 0 {
		seconds = 1
	}
	if seconds%86400 == 0 {
		return modelMonitorGroupHealthUnitString(seconds/86400, "day")
	}
	if seconds%3600 == 0 {
		return modelMonitorGroupHealthUnitString(seconds/3600, "hour")
	}
	if seconds%60 == 0 {
		return modelMonitorGroupHealthUnitString(seconds/60, "minute")
	}
	return modelMonitorGroupHealthUnitString(seconds, "second")
}

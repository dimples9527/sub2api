package repository

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/Wei-Shaw/sub2api/internal/service"
)

func newModelMonitorGroupHealthRepoTestDB(t *testing.T) (*sql.DB, sqlmock.Sqlmock, *modelMonitorGroupHealthRepository) {
	t.Helper()
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("create sqlmock database: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	return db, mock, &modelMonitorGroupHealthRepository{db: db}
}

func TestModelMonitorGroupHealthRepositoryListGroups(t *testing.T) {
	_, mock, repo := newModelMonitorGroupHealthRepoTestDB(t)
	mock.ExpectQuery(`SELECT[[:space:]]+g\.id`).
		WithArgs(sqlmock.AnyArg(), "openai").
		WillReturnRows(sqlmock.NewRows([]string{"id", "name", "platform", "effective_platform"}).
			AddRow(int64(1), "main", "composite", "openai"))

	groups, err := repo.ListGroups(context.Background(), []int64{1, 2}, "openai")
	if err != nil {
		t.Fatalf("ListGroups() error = %v", err)
	}
	if len(groups) != 1 || groups[0].ID != 1 || groups[0].EffectivePlatform != "openai" {
		t.Fatalf("groups = %+v", groups)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet database expectation: %v", err)
	}
}

func TestModelMonitorGroupHealthRepositoryListUsageBuckets(t *testing.T) {
	_, mock, repo := newModelMonitorGroupHealthRepoTestDB(t)
	start := time.Date(2026, 8, 28, 11, 0, 0, 0, time.UTC)
	end := start.Add(time.Hour)
	bucket := start.Add(-5 * time.Minute)
	mock.ExpectQuery(`date_bin\(\$1::interval`).
		WithArgs("5 minutes", start, end, sqlmock.AnyArg(), "openai").
		WillReturnRows(sqlmock.NewRows([]string{
			"group_id", "bucket_start", "success_count", "latency_sample_count", "avg_latency_ms", "p95_latency_ms", "p95_first_token_ms", "last_request_at",
		}).AddRow(int64(1), bucket, int64(8), int64(7), 930.5, 1400.0, 380.0, start.Add(30*time.Minute)))

	buckets, err := repo.ListUsageBuckets(context.Background(), start, end, 5*time.Minute, []int64{1}, "openai")
	if err != nil {
		t.Fatalf("ListUsageBuckets() error = %v", err)
	}
	if len(buckets) != 1 || buckets[0].SuccessCount != 8 || buckets[0].LatencySampleCount != 7 || buckets[0].P95FirstTokenMS != 380 || buckets[0].LastRequestAt == nil {
		t.Fatalf("buckets = %+v", buckets)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet database expectation: %v", err)
	}
}

func TestModelMonitorGroupHealthRepositoryListErrorBuckets(t *testing.T) {
	_, mock, repo := newModelMonitorGroupHealthRepoTestDB(t)
	start := time.Date(2026, 8, 28, 11, 0, 0, 0, time.UTC)
	end := start.Add(time.Hour)
	bucket := start.Add(10 * time.Minute)
	mock.ExpectQuery(`date_bin\(\$1::interval`).
		WithArgs("1 hour", start, end, sqlmock.AnyArg(), "openai").
		WillReturnRows(sqlmock.NewRows([]string{"group_id", "bucket_start", "error_count", "business_limited_count", "last_request_at"}).
			AddRow(int64(1), bucket, int64(5), int64(2), start.Add(40*time.Minute)))

	buckets, err := repo.ListErrorBuckets(context.Background(), start, end, time.Hour, []int64{1}, "openai")
	if err != nil {
		t.Fatalf("ListErrorBuckets() error = %v", err)
	}
	if len(buckets) != 1 || buckets[0].ErrorCount != 5 || buckets[0].BusinessLimitedCount != 2 || buckets[0].LastRequestAt == nil {
		t.Fatalf("buckets = %+v", buckets)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet database expectation: %v", err)
	}
}

func TestModelMonitorGroupHealthRepositoryListErrorCategories(t *testing.T) {
	_, mock, repo := newModelMonitorGroupHealthRepoTestDB(t)
	start := time.Date(2026, 8, 28, 11, 0, 0, 0, time.UTC)
	end := start.Add(time.Hour)
	mock.ExpectQuery(`CASE[[:space:]]+WHEN`).
		WithArgs(start, end, sqlmock.AnyArg(), "openai").
		WillReturnRows(sqlmock.NewRows([]string{"group_id", "category", "count"}).
			AddRow(int64(1), service.ModelMonitorGroupHealthErrorUpstreamRateLimit, int64(3)))

	items, err := repo.ListErrorCategories(context.Background(), start, end, []int64{1}, "openai")
	if err != nil {
		t.Fatalf("ListErrorCategories() error = %v", err)
	}
	if len(items) != 1 || items[0].Category != service.ModelMonitorGroupHealthErrorUpstreamRateLimit || items[0].Count != 3 {
		t.Fatalf("items = %+v", items)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet database expectation: %v", err)
	}
}

package repository

import (
	"context"
	"database/sql"
	"regexp"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/stretchr/testify/require"
)

func TestSupplierProviderAuthAuditRepositoryRecordsEventAndUpdatesSummary(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	startedAt := time.Date(2026, time.August, 5, 10, 0, 0, 0, time.UTC)
	finishedAt := startedAt.Add(275 * time.Millisecond)
	httpStatus := 401
	event := service.SupplierProviderAuthEventRecord{
		ProviderID:       42,
		EventType:        service.SupplierProviderAuthEventLoginFailed,
		Source:           service.SupplierProviderAuthSourceSync,
		Status:           service.SupplierProviderAuthStatusFailed,
		StartedAt:        startedAt,
		FinishedAt:       finishedAt,
		DurationMS:       275,
		HTTPStatus:       &httpStatus,
		ErrorMessage:     "unauthorized",
		TokenFingerprint: "fingerprint",
		TokenLength:      32,
		CookiePresent:    true,
		CreatedAt:        finishedAt,
	}

	mock.ExpectBegin()
	mock.ExpectExec(regexp.QuoteMeta("INSERT INTO supplier_provider_auth_events (")).
		WithArgs(
			event.ProviderID, event.EventType, event.Source, event.Status,
			event.StartedAt, event.FinishedAt, event.DurationMS, event.HTTPStatus,
			event.ErrorMessage, event.TokenFingerprint, event.TokenLength,
			event.TokenExpiresAt, event.CookiePresent, event.CreatedAt,
		).
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectExec(regexp.QuoteMeta("INSERT INTO supplier_provider_runtime_stats (")).
		WithArgs(
			event.ProviderID, event.EventType, event.Status, event.FinishedAt,
			event.ErrorMessage, event.TokenExpiresAt, event.TokenFingerprint,
		).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()

	repo := NewSupplierProviderAuthAuditRepository(db)
	require.NoError(t, repo.Record(context.Background(), event))
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestSupplierProviderAuthAuditRepositoryTreatsCacheHitAsSuccessfulLogin(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	finishedAt := time.Date(2026, time.August, 5, 10, 0, 0, 0, time.UTC)
	event := service.SupplierProviderAuthEventRecord{
		ProviderID:  42,
		EventType:   service.SupplierProviderAuthEventCacheHit,
		Source:      service.SupplierProviderAuthSourceSync,
		Status:      service.SupplierProviderAuthStatusSuccess,
		StartedAt:   finishedAt,
		FinishedAt:  finishedAt,
		TokenLength: 16,
		CreatedAt:   finishedAt,
	}

	mock.ExpectBegin()
	mock.ExpectExec(regexp.QuoteMeta("INSERT INTO supplier_provider_auth_events (")).
		WithArgs(
			event.ProviderID, event.EventType, event.Source, event.Status,
			event.StartedAt, event.FinishedAt, int64(0), (*int)(nil), "", "", event.TokenLength,
			event.TokenExpiresAt, event.CookiePresent, event.CreatedAt,
		).
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectExec(`CASE WHEN \$2 IN \('cache_hit', 'login_success', 'login_failed'\) THEN 1 ELSE 0 END,\s+CASE WHEN \$2 IN \('cache_hit', 'login_success'\) THEN 1 ELSE 0 END[\s\S]+auth_last_login_at = CASE WHEN \$2 IN \('cache_hit', 'login_success', 'login_failed'\) THEN EXCLUDED\.auth_last_login_at`).
		WithArgs(
			event.ProviderID, event.EventType, event.Status, event.FinishedAt,
			event.ErrorMessage, event.TokenExpiresAt, event.TokenFingerprint,
		).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()

	repo := NewSupplierProviderAuthAuditRepository(db)
	require.NoError(t, repo.Record(context.Background(), event))
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestSupplierProviderAuthAuditRepositoryGetsSummary(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	lastLoginAt := time.Date(2026, time.August, 5, 10, 0, 0, 0, time.UTC)
	lastCacheHitAt := lastLoginAt.Add(-time.Minute)
	lastTokenExpiresAt := lastLoginAt.Add(time.Hour)
	mock.ExpectQuery(`SELECT COALESCE\(s\.auth_login_count, 0\)`).
		WithArgs(int64(42)).
		WillReturnRows(sqlmock.NewRows([]string{
			"auth_login_count", "auth_login_success_count", "auth_login_failure_count",
			"auth_cache_hit_count", "auth_cache_miss_count", "auth_last_login_at",
			"auth_last_login_status", "auth_last_login_error", "auth_last_cache_hit_at",
			"auth_last_cache_error", "auth_last_token_expires_at", "auth_last_token_fingerprint",
		}).AddRow(
			int64(5), int64(4), int64(1), int64(9), int64(2), lastLoginAt,
			"success", "", lastCacheHitAt, "", lastTokenExpiresAt, "fingerprint",
		))

	repo := NewSupplierProviderAuthAuditRepository(db)
	summary, err := repo.GetSummary(context.Background(), 42)
	require.NoError(t, err)
	require.Equal(t, int64(5), summary.LoginCount)
	require.Equal(t, int64(4), summary.LoginSuccessCount)
	require.Equal(t, int64(1), summary.LoginFailureCount)
	require.Equal(t, int64(9), summary.CacheHitCount)
	require.Equal(t, int64(2), summary.CacheMissCount)
	require.Equal(t, "success", summary.LastLoginStatus)
	require.Equal(t, "fingerprint", summary.LastTokenFingerprint)
	require.NotNil(t, summary.LastLoginAt)
	require.NotNil(t, summary.LastCacheHitAt)
	require.NotNil(t, summary.LastTokenExpiresAt)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestSupplierProviderAuthAuditRepositoryReturnsNotFoundForMissingProvider(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	mock.ExpectQuery(`SELECT COALESCE\(s\.auth_login_count, 0\)`).
		WithArgs(int64(99)).
		WillReturnError(sql.ErrNoRows)

	repo := NewSupplierProviderAuthAuditRepository(db)
	_, err = repo.GetSummary(context.Background(), 99)
	require.ErrorIs(t, err, service.ErrSupplierProviderNotFound)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestSupplierProviderAuthAuditRepositoryListsFilteredHistory(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	createdAt := time.Date(2026, time.August, 5, 10, 0, 0, 0, time.UTC)
	startedAt := createdAt.Add(-150 * time.Millisecond)
	finishedAt := createdAt
	expiresAt := createdAt.Add(time.Hour)
	params := service.SupplierProviderAuthHistoryParams{
		Page:      2,
		PageSize:  20,
		EventType: service.SupplierProviderAuthEventLoginSuccess,
	}

	mock.ExpectQuery(regexp.QuoteMeta("SELECT COUNT(*) FROM supplier_provider_auth_events WHERE provider_id = $1 AND event_type = $2")).
		WithArgs(int64(42), service.SupplierProviderAuthEventLoginSuccess).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(int64(21)))
	mock.ExpectQuery(`SELECT id, provider_id, event_type, source, status,[\s\S]*ORDER BY created_at DESC, id DESC LIMIT \$3 OFFSET \$4`).
		WithArgs(int64(42), service.SupplierProviderAuthEventLoginSuccess, 20, 20).
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "provider_id", "event_type", "source", "status", "started_at", "finished_at",
			"duration_ms", "http_status", "error_message", "token_fingerprint", "token_length",
			"token_expires_at", "cookie_present", "created_at",
		}).AddRow(
			int64(7), int64(42), "login_success", "sync", "success", startedAt, finishedAt,
			int64(150), 200, "", "fingerprint", 32, expiresAt, true, createdAt,
		))

	repo := NewSupplierProviderAuthAuditRepository(db)
	result, err := repo.ListHistory(context.Background(), 42, params)
	require.NoError(t, err)
	require.Equal(t, int64(21), result.Total)
	require.Equal(t, 2, result.Page)
	require.Equal(t, 20, result.PageSize)
	require.Len(t, result.Items, 1)
	require.Equal(t, service.SupplierProviderAuthEventLoginSuccess, result.Items[0].EventType)
	require.Equal(t, 200, *result.Items[0].HTTPStatus)
	require.Equal(t, "fingerprint", result.Items[0].TokenFingerprint)
	require.NotNil(t, result.Items[0].TokenExpiresAt)
	require.NoError(t, mock.ExpectationsWereMet())
}

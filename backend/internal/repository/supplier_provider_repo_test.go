package repository

import (
	"context"
	"regexp"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/Wei-Shaw/sub2api/internal/pkg/timezone"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/stretchr/testify/require"
)

func TestSupplierProviderRepositoryDisableAfterAuthFailureUpdatesProviderAndRuntimeStatus(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	repo := NewSupplierProviderRepository(db)
	syncedAt := time.Date(2026, 8, 5, 12, 0, 0, 0, time.UTC)
	message := "供应商登录失败，已自动停用。"

	mock.ExpectBegin()
	mock.ExpectExec(regexp.QuoteMeta(`
UPDATE supplier_providers
SET enabled=FALSE, updated_at=NOW()
WHERE id=$1 AND deleted_at IS NULL`)).
		WithArgs(int64(42)).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec(regexp.QuoteMeta(`
INSERT INTO supplier_provider_runtime_stats (
  provider_id, sync_status, sync_message, last_sync_at, updated_at
) VALUES ($1,$2,$3,$4,$4)
ON CONFLICT (provider_id) DO UPDATE SET
  sync_status=EXCLUDED.sync_status,
  sync_message=EXCLUDED.sync_message,
  last_sync_at=EXCLUDED.last_sync_at,
  updated_at=EXCLUDED.updated_at`)).
		WithArgs(int64(42), service.SupplierSyncStatusFailed, message, syncedAt).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()

	err = repo.DisableAfterAuthFailure(context.Background(), 42, message, syncedAt)

	require.NoError(t, err)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestSupplierProviderRepositoryCreateUsesAllProviderFieldArguments(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	repo := NewSupplierProviderRepository(db)
	provider := &service.SupplierProvider{
		Code:                       "claudenb",
		Name:                       "claudenb",
		ProviderType:               "sub2api",
		NewAPIAuthMode:             service.SupplierNewAPIAuthModeAuto,
		BaseURL:                    "https://ai.claudenb.com",
		LoginURL:                   "/api/v1/auth/login",
		APIKeysURL:                 "/api/v1/keys?page=1&page_size=100&timezone=Asia%2FShanghai",
		GroupsURL:                  "/api/v1/groups/available?timezone=Asia%2FShanghai",
		AvailableGroupsURL:         "/api/v1/groups/available?timezone=Asia%2FShanghai",
		BalanceURL:                 "/api/v1/auth/me?timezone=Asia%2FShanghai",
		UsageCostURL:               "/api/v1/usage/dashboard/stats?timezone=Asia%2FShanghai",
		MonitorURL:                 "/api/v1/channel-monitors?timezone=Asia%2FShanghai",
		AccountNamePrefix:          "claudenb-",
		TempDisableMinutes:         0,
		AccountRateMultiplierScale: 1,
		SortOrder:                  0,
		Enabled:                    true,
		TurnstileEnabled:           false,
		IsDefault:                  false,
		Email:                      "test@example.com",
		PasswordEncrypted:          "test-password",
	}

	mock.ExpectBegin()
	mock.ExpectQuery(regexp.QuoteMeta("SELECT EXISTS(SELECT 1 FROM supplier_providers WHERE deleted_at IS NULL)")).
		WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(true))
	mock.ExpectQuery(regexp.QuoteMeta(`
INSERT INTO supplier_providers (
  code, name, provider_type, newapi_auth_mode, base_url, login_url, api_keys_url, groups_url,
  available_groups_url, balance_url, usage_cost_url, recharge_url, monitor_url, account_name_prefix,
  temp_disable_minutes, account_rate_multiplier_scale, sort_order, enabled, turnstile_enabled, is_default
) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18,$19,$20)
RETURNING id, created_at, updated_at`)).
		WithArgs(
			"claudenb", "claudenb", "sub2api", "auto", "https://ai.claudenb.com", "/api/v1/auth/login",
			"/api/v1/keys?page=1&page_size=100&timezone=Asia%2FShanghai",
			"/api/v1/groups/available?timezone=Asia%2FShanghai",
			"/api/v1/groups/available?timezone=Asia%2FShanghai",
			"/api/v1/auth/me?timezone=Asia%2FShanghai",
			"/api/v1/usage/dashboard/stats?timezone=Asia%2FShanghai",
			"", "/api/v1/channel-monitors?timezone=Asia%2FShanghai", "claudenb-", 0, 1.0, 0, true, false, false,
		).
		WillReturnRows(sqlmock.NewRows([]string{"id", "created_at", "updated_at"}).AddRow(42, time.Now(), time.Now()))
	mock.ExpectExec(regexp.QuoteMeta(`
INSERT INTO supplier_provider_credentials (provider_id, email, username, password_encrypted) VALUES ($1,$2,$3,$4)`)).
		WithArgs(int64(42), "test@example.com", "", "test-password").
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec(regexp.QuoteMeta(`
INSERT INTO supplier_provider_runtime_stats (provider_id) VALUES ($1)`)).
		WithArgs(int64(42)).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()

	err = repo.Create(context.Background(), provider)

	require.NoError(t, err)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestSupplierProviderRepositoryDisableAfterAuthFailureReturnsNotFound(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	repo := NewSupplierProviderRepository(db)
	mock.ExpectBegin()
	mock.ExpectExec(regexp.QuoteMeta(`
UPDATE supplier_providers
SET enabled=FALSE, updated_at=NOW()
WHERE id=$1 AND deleted_at IS NULL`)).
		WithArgs(int64(404)).
		WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectRollback()

	err = repo.DisableAfterAuthFailure(context.Background(), 404, "登录失败", time.Now())

	require.ErrorIs(t, err, service.ErrSupplierProviderNotFound)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestSupplierProviderRepositoryUpdateUsesDistinctArgumentsForProviderFields(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	repo := NewSupplierProviderRepository(db)
	provider := &service.SupplierProvider{
		ID:                         42,
		Code:                       "beikun",
		Name:                       "备坤",
		ProviderType:               "newapi",
		NewAPIAuthMode:             service.SupplierNewAPIAuthModeAuto,
		BaseURL:                    "https://supplier.example.com",
		LoginURL:                   "/api/login",
		APIKeysURL:                 "/api/keys",
		GroupsURL:                  "/api/groups",
		AvailableGroupsURL:         "/api/groups/available",
		BalanceURL:                 "/api/balance",
		UsageCostURL:               "/api/cost",
		RechargeURL:                "/api/recharge",
		MonitorURL:                 "/api/monitor",
		AccountNamePrefix:          "bk-",
		TempDisableMinutes:         15,
		AccountRateMultiplierScale: 1.25,
		SortOrder:                  7,
		Enabled:                    true,
		TurnstileEnabled:           false,
		IsDefault:                  false,
		Email:                      "admin@example.com",
		Username:                   "admin",
		PasswordEncrypted:          "encrypted-password",
	}

	mock.ExpectBegin()
	mock.ExpectExec(regexp.QuoteMeta(`
UPDATE supplier_providers SET
  code=$2, name=$3, provider_type=$4, newapi_auth_mode=$5, base_url=$6, login_url=$7,
  api_keys_url=$8, groups_url=$9, available_groups_url=$10, balance_url=$11,
  usage_cost_url=$12, recharge_url=$13, monitor_url=$14, account_name_prefix=$15, temp_disable_minutes=$16,
  account_rate_multiplier_scale=$17, sort_order=$18, enabled=$19,
  turnstile_enabled=$20, is_default=$21, updated_at=NOW()
WHERE id=$1 AND deleted_at IS NULL`)).
		WithArgs(
			int64(42), "beikun", "备坤", "newapi", "auto", "https://supplier.example.com", "/api/login",
			"/api/keys", "/api/groups", "/api/groups/available", "/api/balance", "/api/cost", "/api/recharge",
			"/api/monitor", "bk-", 15, 1.25, 7, true, false, false,
		).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec(regexp.QuoteMeta(`
INSERT INTO supplier_provider_credentials (provider_id, email, username, password_encrypted, updated_at)
VALUES ($1,$2,$3,$4,NOW())
ON CONFLICT (provider_id) DO UPDATE SET email=EXCLUDED.email, username=EXCLUDED.username,
password_encrypted=EXCLUDED.password_encrypted, updated_at=NOW()`)).
		WithArgs(int64(42), "admin@example.com", "admin", "encrypted-password").
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()

	err = repo.Update(context.Background(), provider)

	require.NoError(t, err)
	require.NoError(t, mock.ExpectationsWereMet())
}

// 余额汇总卡要同时给出生效成本与上游原始成本两个口径，列的顺序与 Scan 必须对齐 ——
// 这里钉住查询列，避免日后加列时只改 SQL、忘了改 Scan（那会静默把上游成本扫成 0）。
func TestSupplierProviderRepositoryListBalanceSummaryDaysScansUpstreamCost(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	repo := NewSupplierProviderRepository(db)

	mock.ExpectQuery(regexp.QuoteMeta(`
SELECT TO_CHAR(d.stat_date, 'YYYY-MM-DD') AS date,
       COALESCE(SUM(d.current_balance), 0) AS balance,
       COALESCE(SUM(d.today_cost), 0) AS effective_cost,
       COALESCE(SUM(COALESCE(d.raw_upstream_cost, d.today_cost)), 0) AS upstream_cost
FROM supplier_provider_daily_stats d
JOIN supplier_providers p ON p.id = d.provider_id AND p.deleted_at IS NULL
GROUP BY d.stat_date
ORDER BY d.stat_date`)).
		WillReturnRows(sqlmock.NewRows([]string{"date", "balance", "effective_cost", "upstream_cost"}).
			AddRow("2026-08-14", 170.0, 60.0, 72.5))

	days, err := repo.ListBalanceSummaryDays(context.Background())

	require.NoError(t, err)
	require.Len(t, days, 1)
	require.Equal(t, "2026-08-14", days[0].Date)
	require.Equal(t, 170.0, days[0].Balance)
	require.Equal(t, 60.0, days[0].EffectiveCost)
	require.Equal(t, 72.5, days[0].UpstreamCost)
	require.NoError(t, mock.ExpectationsWereMet())
}

// 「今日总成本」卡要取本地成本口径（唯一匹配本地账号的用户扣费），
// 与成本分析弹窗一致；这里钉住查询与日期边界，避免退化成 daily_stats 的生效成本。
func TestSupplierProviderRepositoryListLocalCostsForDay(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	repo := NewSupplierProviderRepository(db)

	day := time.Date(2026, 8, 14, 0, 0, 0, 0, time.Local)
	mock.ExpectQuery(`(?s)SELECT COALESCE\(SUM\(ul\.actual_cost\), 0\) AS local_cost,\s+COUNT\(\*\) AS usage_count\s+FROM usage_logs ul`).
		WithArgs(timezone.StartOfDay(day), timezone.StartOfDay(day).AddDate(0, 0, 1)).
		WillReturnRows(sqlmock.NewRows([]string{"local_cost", "usage_count"}).AddRow(52.2, 7))

	localCost, ok, err := repo.ListLocalCostsForDay(context.Background(), day)

	require.NoError(t, err)
	require.True(t, ok)
	require.Equal(t, 52.2, localCost)
	require.NoError(t, mock.ExpectationsWereMet())
}

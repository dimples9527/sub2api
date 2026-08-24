package repository

import (
	"context"
	"regexp"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
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

package repository

import (
	"context"
	"encoding/json"
	"regexp"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/stretchr/testify/require"
)

func TestSupplierProviderRechargeRepositoryUpsertOnlyRewritesChangedBusinessFields(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	repo := NewSupplierProviderRechargeRepository(db)
	occurredAt := time.Date(2026, 8, 5, 12, 0, 0, 0, time.UTC)

	mock.ExpectBegin()
	mock.ExpectExec(regexp.QuoteMeta(`
INSERT INTO supplier_provider_recharges (
  provider_id, external_id, external_code, recharge_type, amount, status,
  occurred_at, description, source_payload, synced_at, updated_at
) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$10)
ON CONFLICT (provider_id, external_id) DO UPDATE SET
  external_code = EXCLUDED.external_code,
  recharge_type = EXCLUDED.recharge_type,
  amount = EXCLUDED.amount,
  status = EXCLUDED.status,
  occurred_at = EXCLUDED.occurred_at,
  description = EXCLUDED.description,
  source_payload = EXCLUDED.source_payload,
  synced_at = EXCLUDED.synced_at,
  updated_at = EXCLUDED.updated_at
WHERE supplier_provider_recharges.external_code IS DISTINCT FROM EXCLUDED.external_code
   OR supplier_provider_recharges.recharge_type IS DISTINCT FROM EXCLUDED.recharge_type
   OR supplier_provider_recharges.amount IS DISTINCT FROM EXCLUDED.amount
   OR supplier_provider_recharges.status IS DISTINCT FROM EXCLUDED.status
   OR supplier_provider_recharges.occurred_at IS DISTINCT FROM EXCLUDED.occurred_at
   OR supplier_provider_recharges.description IS DISTINCT FROM EXCLUDED.description
   OR supplier_provider_recharges.source_payload IS DISTINCT FROM EXCLUDED.source_payload`)).
		WithArgs(int64(3), "ext-7", "code-7", "redeem", 12.5, "success", occurredAt,
			"充值 12.5", []byte(`{"amount":12.5}`), sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectCommit()

	err = repo.Upsert(context.Background(), 3, []service.SupplierProviderRechargeRecord{{
		ExternalID:   " ext-7 ",
		ExternalCode: "code-7",
		RechargeType: "redeem",
		Amount:       12.5,
		Status:       "success",
		OccurredAt:   occurredAt,
		Description:  "充值 12.5",
		RawPayload:   json.RawMessage(`{"amount":12.5}`),
	}})

	require.NoError(t, err)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestSupplierProviderRechargeRepositoryListDoesNotSelectSourcePayload(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	repo := NewSupplierProviderRechargeRepository(db)
	occurredAt := time.Date(2026, 8, 5, 12, 0, 0, 0, time.UTC)

	mock.ExpectQuery(regexp.QuoteMeta(`SELECT COUNT(*), COALESCE(SUM(r.amount), 0) FROM supplier_provider_recharges r JOIN supplier_providers p ON p.id = r.provider_id WHERE p.deleted_at IS NULL`)).
		WillReturnRows(sqlmock.NewRows([]string{"count", "sum"}).AddRow(int64(1), 12.5))
	mock.ExpectQuery(regexp.QuoteMeta(`
SELECT r.id, r.provider_id, p.name, p.provider_type, r.external_id, r.external_code,
       r.recharge_type, r.amount, r.status, r.occurred_at, r.description,
       r.synced_at, r.created_at, r.updated_at
FROM supplier_provider_recharges r
JOIN supplier_providers p ON p.id = r.provider_id
WHERE p.deleted_at IS NULL ORDER BY r.occurred_at DESC, r.id DESC LIMIT $1 OFFSET $2`)).
		WithArgs(20, 0).
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "provider_id", "name", "provider_type", "external_id", "external_code",
			"recharge_type", "amount", "status", "occurred_at", "description",
			"synced_at", "created_at", "updated_at",
		}).AddRow(int64(7), int64(3), "供应商A", "sub2api", "ext-7", "code-7",
			"redeem", 12.5, "success", occurredAt, "充值 12.5",
			occurredAt, occurredAt, occurredAt))

	result, err := repo.List(context.Background(), service.SupplierProviderRechargeListParams{Page: 1, PageSize: 20})

	require.NoError(t, err)
	require.NoError(t, mock.ExpectationsWereMet())
	require.Equal(t, int64(1), result.Total)
	require.InDelta(t, 12.5, result.TotalAmount, 1e-9)
	require.Len(t, result.Items, 1)
	require.Equal(t, "供应商A", result.Items[0].ProviderName)
}

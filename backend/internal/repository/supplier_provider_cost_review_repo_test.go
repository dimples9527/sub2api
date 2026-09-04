package repository

import (
	"context"
	"database/sql"
	"errors"
	"regexp"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/stretchr/testify/require"
)

func newSupplierProviderCostReviewRepoMock(t *testing.T) (*supplierProviderCostReviewRepository, sqlmock.Sqlmock) {
	t.Helper()
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })
	return NewSupplierProviderCostReviewRepository(db).(*supplierProviderCostReviewRepository), mock
}

func TestSupplierProviderCostReviewRepositoryList(t *testing.T) {
	repo, mock := newSupplierProviderCostReviewRepoMock(t)
	mock.ExpectQuery(`SELECT COUNT\(1\) FROM supplier_provider_cost_reviews`).
		WithArgs(int64(8), "2026-08-01", "2026-08-31", service.CostReviewStatusPending, "%alpha%").
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(int64(1)))
	mock.ExpectQuery(`SELECT r\.id, r\.provider_id, p\.name AS provider_name`).
		WithArgs(int64(8), "2026-08-01", "2026-08-31", service.CostReviewStatusPending, "%alpha%", 20, 20).
		WillReturnRows(costReviewRows().AddRow(
			int64(10), int64(8), "supplier-a", time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC),
			1.2, 1.0, 0.95, 1.0, nil, 1.0, 0.2, 0.0, "pending_review", "none", nil, nil, int64(1), nil, nil, int64(1),
			time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC), time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC),
		))

	result, err := repo.List(context.Background(), service.SupplierProviderCostReviewListParams{ProviderID: 8, Keyword: "alpha", StartDate: "2026-08-01", EndDate: "2026-08-31", Status: service.CostReviewStatusPending, Page: 2, PageSize: 20})
	require.NoError(t, err)
	require.Equal(t, int64(1), result.Total)
	require.Len(t, result.Items, 1)
	require.Equal(t, "supplier-a", result.Items[0].ProviderName)
	// 本地成本紧跟计算成本，列顺序错位时这里会读成 auto_adopted_cost 的值
	require.NotNil(t, result.Items[0].LocalCost)
	require.Equal(t, 0.95, *result.Items[0].LocalCost)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestSupplierProviderCostReviewRepositoryHistoryOrdersNewestFirst(t *testing.T) {
	repo, mock := newSupplierProviderCostReviewRepoMock(t)
	mock.ExpectQuery(`SELECT id, review_id, provider_id, stat_date`).
		WithArgs(int64(10)).
		WillReturnRows(historyRows().AddRow(
			int64(2), int64(10), int64(8), time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC), "approve", nil,
			1.2, 1.0, 0.95, 1.0, 1.2, 0.2, 0.2, "approved", "upstream", nil, int64(9), time.Date(2026, 8, 1, 2, 0, 0, 0, time.UTC),
		))
	got, err := repo.History(context.Background(), 10)
	require.NoError(t, err)
	require.Len(t, got, 1)
	require.Equal(t, service.CostReviewDecisionUpstream, got[0].DecisionType)
	require.NotNil(t, got[0].LocalCost)
	require.Equal(t, 0.95, *got[0].LocalCost)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestSupplierProviderCostReviewRepositoryApproveRollsBackWhenDailyStatsFails(t *testing.T) {
	repo, mock := newSupplierProviderCostReviewRepoMock(t)
	mock.ExpectBegin()
	mock.ExpectQuery(`SELECT r\.id, r\.provider_id, p\.name AS provider_name.*FOR UPDATE`).
		WithArgs(int64(10)).
		WillReturnRows(costReviewRows().AddRow(
			int64(10), int64(8), "supplier-a", time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC),
			1.2, 1.0, 0.95, 1.0, nil, 1.0, 0.2, 0.0, "pending_review", "none", nil, nil, int64(1), nil, nil, int64(3),
			time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC), time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC),
		))
	mock.ExpectExec(`UPDATE supplier_provider_cost_reviews`).WithArgs(
		1.2, 0.2, service.CostReviewDecisionUpstream, int64(9), int64(10), int64(3),
	).WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectExec(`INSERT INTO supplier_provider_daily_stats`).WillReturnError(errors.New("daily stats down"))
	mock.ExpectRollback()

	_, err := repo.Approve(context.Background(), 10, service.SupplierProviderCostReviewApproveInput{DecisionType: service.CostReviewDecisionUpstream, Version: 3, OperatorID: 9})
	require.Error(t, err)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestSupplierProviderCostReviewRepositoryApproveRejectsVersionConflict(t *testing.T) {
	repo, mock := newSupplierProviderCostReviewRepoMock(t)
	mock.ExpectBegin()
	mock.ExpectQuery(`SELECT r\.id, r\.provider_id, p\.name AS provider_name.*FOR UPDATE`).
		WithArgs(int64(10)).
		WillReturnRows(costReviewRows().AddRow(
			int64(10), int64(8), "supplier-a", time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC),
			1.2, 1.0, 0.95, 1.0, nil, 1.0, 0.2, 0.0, "pending_review", "none", nil, nil, int64(1), nil, nil, int64(4),
			time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC), time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC),
		))
	mock.ExpectRollback()
	_, err := repo.Approve(context.Background(), 10, service.SupplierProviderCostReviewApproveInput{DecisionType: service.CostReviewDecisionUpstream, Version: 3})
	require.ErrorIs(t, err, service.ErrSupplierProviderCostReviewVersionConflict)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestSupplierProviderCostReviewRepositoryApproveManyUpdatesAllRecordsInOneTransaction(t *testing.T) {
	repo, mock := newSupplierProviderCostReviewRepoMock(t)
	now := time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC)
	mock.ExpectBegin()
	mock.ExpectQuery(`SELECT r\.id, r\.provider_id, p\.name AS provider_name.*FOR UPDATE`).WithArgs(int64(10)).
		WillReturnRows(costReviewRows().AddRow(
			int64(10), int64(8), "supplier-a", now, 1.2, 1.0, 0.95, 1.0, nil, 1.0, 0.2, 0.0, service.CostReviewStatusPending, service.CostReviewDecisionNone, nil, nil, int64(1), nil, now, int64(3), now, now,
		))
	mock.ExpectExec(`UPDATE supplier_provider_cost_reviews`).WithArgs(1.2, 0.2, service.CostReviewDecisionUpstream, int64(9), int64(10), int64(3)).WillReturnResult(sqlmock.NewResult(1, 1))
	// 审批要把人工结论写进 cost_warning，否则成本分析继续显示同步时的自动取值提示。
	mock.ExpectExec(`INSERT INTO supplier_provider_daily_stats`).WithArgs(int64(8), "2026-08-01", 1.2, "已人工审批：生效成本取上游成本 1.20").WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectExec(`INSERT INTO supplier_provider_cost_review_histories`).WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectQuery(`SELECT r\.id, r\.provider_id, p\.name AS provider_name.*FOR UPDATE`).WithArgs(int64(11)).
		WillReturnRows(costReviewRows().AddRow(
			int64(11), int64(9), "supplier-b", now, 2.2, 2.0, 1.95, 2.0, nil, 2.0, 0.2, 0.0, service.CostReviewStatusChanged, service.CostReviewDecisionCalculated, nil, nil, int64(2), nil, now, int64(7), now, now,
		))
	mock.ExpectExec(`UPDATE supplier_provider_cost_reviews`).WithArgs(2.2, 0.2, service.CostReviewDecisionUpstream, int64(9), int64(11), int64(7)).WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectExec(`INSERT INTO supplier_provider_daily_stats`).WithArgs(int64(9), "2026-08-01", 2.2, "已人工审批：生效成本取上游成本 2.20").WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectExec(`INSERT INTO supplier_provider_cost_review_histories`).WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit()

	got, err := repo.ApproveMany(context.Background(), service.SupplierProviderCostReviewBulkApproveInput{
		Items: []service.SupplierProviderCostReviewApproveItem{{ID: 10, Version: 3}, {ID: 11, Version: 7}}, DecisionType: service.CostReviewDecisionUpstream, OperatorID: 9,
	})
	require.NoError(t, err)
	require.Len(t, got, 2)
	require.Equal(t, int64(10), got[0].ID)
	require.Equal(t, int64(11), got[1].ID)
	require.Equal(t, int64(4), got[0].Version)
	require.Equal(t, int64(8), got[1].Version)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestSupplierProviderCostReviewRepositoryApproveManyRollsBackWhenSecondVersionConflicts(t *testing.T) {
	repo, mock := newSupplierProviderCostReviewRepoMock(t)
	now := time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC)
	mock.ExpectBegin()
	mock.ExpectQuery(`SELECT r\.id, r\.provider_id, p\.name AS provider_name.*FOR UPDATE`).WithArgs(int64(10)).
		WillReturnRows(costReviewRows().AddRow(
			int64(10), int64(8), "supplier-a", now, 1.2, 1.0, 0.95, 1.0, nil, 1.0, 0.2, 0.0, service.CostReviewStatusPending, service.CostReviewDecisionNone, nil, nil, int64(1), nil, now, int64(3), now, now,
		))
	mock.ExpectExec(`UPDATE supplier_provider_cost_reviews`).WithArgs(1.2, 0.2, service.CostReviewDecisionUpstream, int64(9), int64(10), int64(3)).WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectExec(`INSERT INTO supplier_provider_daily_stats`).WithArgs(int64(8), "2026-08-01", 1.2, "已人工审批：生效成本取上游成本 1.20").WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectExec(`INSERT INTO supplier_provider_cost_review_histories`).WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectQuery(`SELECT r\.id, r\.provider_id, p\.name AS provider_name.*FOR UPDATE`).WithArgs(int64(11)).
		WillReturnRows(costReviewRows().AddRow(
			int64(11), int64(9), "supplier-b", now, 2.2, 2.0, 1.95, 2.0, nil, 2.0, 0.2, 0.0, service.CostReviewStatusPending, service.CostReviewDecisionNone, nil, nil, int64(2), nil, now, int64(8), now, now,
		))
	mock.ExpectRollback()

	_, err := repo.ApproveMany(context.Background(), service.SupplierProviderCostReviewBulkApproveInput{
		Items: []service.SupplierProviderCostReviewApproveItem{{ID: 10, Version: 3}, {ID: 11, Version: 7}}, DecisionType: service.CostReviewDecisionUpstream, OperatorID: 9,
	})
	require.ErrorIs(t, err, service.ErrSupplierProviderCostReviewVersionConflict)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestSupplierProviderCostReviewRepositorySyncCreatesReviewAndUpdatesDailyStatsInTransaction(t *testing.T) {
	repo, mock := newSupplierProviderCostReviewRepoMock(t)
	now := time.Date(2026, 8, 24, 3, 0, 0, 0, time.UTC)
	mock.ExpectBegin()
	mock.ExpectQuery(`SELECT r\.id, r\.provider_id, p\.name AS provider_name.*FOR UPDATE`).WithArgs(int64(8), "2026-08-24").WillReturnError(sql.ErrNoRows)
	mock.ExpectQuery(`INSERT INTO supplier_provider_cost_reviews`).WithArgs(
		int64(8), "2026-08-24", 1.2, 1.0, 0.95, 1.0, 1.0, 0.2, 0.0, int64(7), now,
	).WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(int64(10)))
	mock.ExpectExec(`INSERT INTO supplier_provider_daily_stats`).WithArgs(int64(8), "2026-08-24", 1.0).WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectExec(`INSERT INTO supplier_provider_cost_review_histories`).WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit()

	got, err := repo.Sync(context.Background(), service.SupplierProviderCostReviewSyncInput{ProviderID: 8, StatDate: now, UpstreamCost: ptr(1.2), CalculatedCost: ptr(1.0), LocalCost: ptr(0.95), AutoAdoptedCost: ptr(1.0), EffectiveCost: 1.0, SyncRunID: ptrInt64(7), SyncedAt: now})
	require.NoError(t, err)
	require.Equal(t, int64(10), got.ID)
	require.NotNil(t, got.LocalCost)
	require.Equal(t, 0.95, *got.LocalCost)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestSupplierProviderCostReviewRepositorySyncUsesLocalDateWithoutUTCOffset(t *testing.T) {
	repo, mock := newSupplierProviderCostReviewRepoMock(t)
	loc, err := time.LoadLocation("Asia/Shanghai")
	require.NoError(t, err)
	// 业务日期为 2026-08-24（Asia/Shanghai 本地午夜）；修复前 .UTC() 会偏移成 2026-08-23
	statDate := time.Date(2026, 8, 24, 0, 0, 0, 0, loc)
	syncedAt := time.Date(2026, 8, 24, 3, 0, 0, 0, time.UTC)
	mock.ExpectBegin()
	mock.ExpectQuery(`SELECT r\.id, r\.provider_id, p\.name AS provider_name.*FOR UPDATE`).WithArgs(int64(8), "2026-08-24").WillReturnError(sql.ErrNoRows)
	mock.ExpectQuery(`INSERT INTO supplier_provider_cost_reviews`).WithArgs(
		int64(8), "2026-08-24", 1.2, 1.0, 0.95, 1.0, 1.0, 0.2, 0.0, int64(7), syncedAt,
	).WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(int64(10)))
	mock.ExpectExec(`INSERT INTO supplier_provider_daily_stats`).WithArgs(int64(8), "2026-08-24", 1.0).WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectExec(`INSERT INTO supplier_provider_cost_review_histories`).WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit()

	got, err := repo.Sync(context.Background(), service.SupplierProviderCostReviewSyncInput{ProviderID: 8, StatDate: statDate, UpstreamCost: ptr(1.2), CalculatedCost: ptr(1.0), LocalCost: ptr(0.95), AutoAdoptedCost: ptr(1.0), EffectiveCost: 1.0, SyncRunID: ptrInt64(7), SyncedAt: syncedAt})
	require.NoError(t, err)
	require.Equal(t, int64(10), got.ID)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestSupplierProviderCostReviewRepositorySyncPreservesApprovedCost(t *testing.T) {
	repo, mock := newSupplierProviderCostReviewRepoMock(t)
	now := time.Date(2026, 8, 24, 3, 0, 0, 0, time.UTC)
	mock.ExpectBegin()
	mock.ExpectQuery(`SELECT r\.id, r\.provider_id, p\.name AS provider_name.*FOR UPDATE`).WithArgs(int64(8), "2026-08-24").
		WillReturnRows(costReviewRows().AddRow(
			int64(10), int64(8), "supplier-a", now,
			1.2, 1.0, 0.95, 1.0, 1.2, 1.2, 0.2, 0.2, service.CostReviewStatusApproved, service.CostReviewDecisionUpstream, int64(9), now, int64(1), nil, now, int64(3), now, now,
		))
	mock.ExpectExec(`UPDATE supplier_provider_cost_reviews`).WithArgs(
		1.4, 1.1, 0.9, 1.1, 0.3, 1.2, 0.2, service.CostReviewStatusChanged, int64(7), now, int64(10),
	).WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectExec(`INSERT INTO supplier_provider_cost_review_histories`).WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit()

	got, err := repo.Sync(context.Background(), service.SupplierProviderCostReviewSyncInput{
		ProviderID: 8, StatDate: now, UpstreamCost: ptr(1.4), CalculatedCost: ptr(1.1), LocalCost: ptr(0.9), AutoAdoptedCost: ptr(1.1), EffectiveCost: 1.1, SyncRunID: ptrInt64(7), SyncedAt: now,
	})
	require.NoError(t, err)
	require.Equal(t, service.CostReviewStatusChanged, got.Status)
	require.Equal(t, 1.2, got.EffectiveCost)
	require.NotNil(t, got.FinalCost)
	require.Equal(t, 1.2, *got.FinalCost)
	require.Equal(t, int64(4), got.Version)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestSupplierProviderCostReviewRepositorySyncRefreshesPendingEffectiveCost(t *testing.T) {
	repo, mock := newSupplierProviderCostReviewRepoMock(t)
	now := time.Date(2026, 8, 24, 3, 0, 0, 0, time.UTC)
	mock.ExpectBegin()
	mock.ExpectQuery(`SELECT r\.id, r\.provider_id, p\.name AS provider_name.*FOR UPDATE`).WithArgs(int64(8), "2026-08-24").
		WillReturnRows(costReviewRows().AddRow(
			int64(10), int64(8), "supplier-a", now,
			1.2, 1.0, 0.95, 1.0, nil, 1.0, 0.2, 0.0, service.CostReviewStatusPending, service.CostReviewDecisionNone, nil, nil, int64(1), nil, now, int64(3), now, now,
		))
	mock.ExpectExec(`UPDATE supplier_provider_cost_reviews`).WithArgs(
		1.4, 1.1, 0.9, 1.1, 0.3, 1.1, 0.0, service.CostReviewStatusPending, int64(7), now, int64(10),
	).WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectExec(`INSERT INTO supplier_provider_daily_stats`).WithArgs(int64(8), "2026-08-24", 1.1).WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectExec(`INSERT INTO supplier_provider_cost_review_histories`).WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit()

	got, err := repo.Sync(context.Background(), service.SupplierProviderCostReviewSyncInput{
		ProviderID: 8, StatDate: now, UpstreamCost: ptr(1.4), CalculatedCost: ptr(1.1), LocalCost: ptr(0.9), AutoAdoptedCost: ptr(1.1), EffectiveCost: 1.1, SyncRunID: ptrInt64(7), SyncedAt: now,
	})
	require.NoError(t, err)
	require.Equal(t, 1.1, got.EffectiveCost)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestApplySyncToReviewKeepsApprovedStatusWhenLatestDataIsUnchanged(t *testing.T) {
	now := time.Date(2026, 8, 24, 3, 0, 0, 0, time.UTC)
	upstream, calculated, autoAdopted := 1.2, 1.0, 1.0
	review := service.SupplierProviderCostReview{
		Status:          service.CostReviewStatusApproved,
		UpstreamCost:    ptr(upstream),
		CalculatedCost:  ptr(calculated),
		LocalCost:       ptr(0.95),
		AutoAdoptedCost: ptr(autoAdopted),
		FinalCost:       ptr(1.2),
		EffectiveCost:   1.2,
		Version:         3,
	}

	// 本地成本只是参考值，变动不参与"数据是否变化"的判定，不能把已审批记录打回复核
	got := applySyncToReview(review, service.SupplierProviderCostReviewSyncInput{
		UpstreamCost: upstreamPtr(upstream), CalculatedCost: upstreamPtr(calculated), LocalCost: upstreamPtr(0.42), AutoAdoptedCost: upstreamPtr(autoAdopted), EffectiveCost: 1.0, SyncedAt: now,
	})

	require.Equal(t, service.CostReviewStatusApproved, got.Status)
	require.Equal(t, 1.2, got.EffectiveCost)
	require.NotNil(t, got.LocalCost)
	require.Equal(t, 0.42, *got.LocalCost)
}

func costReviewRows() *sqlmock.Rows {
	return sqlmock.NewRows([]string{"id", "provider_id", "provider_name", "stat_date", "upstream_cost", "calculated_cost", "local_cost", "auto_adopted_cost", "final_cost", "effective_cost", "cost_delta", "effective_delta", "status", "decision_type", "approved_by", "approved_at", "sync_count", "last_sync_run_id", "last_synced_at", "version", "created_at", "updated_at"})
}
func historyRows() *sqlmock.Rows {
	return sqlmock.NewRows([]string{"id", "review_id", "provider_id", "stat_date", "event_type", "sync_run_id", "upstream_cost", "calculated_cost", "local_cost", "auto_adopted_cost", "final_cost", "cost_delta", "effective_delta", "status", "decision_type", "manual_cost", "operator_id", "operated_at"})
}
func ptr(v float64) *float64         { return &v }
func ptrInt64(v int64) *int64        { return &v }
func upstreamPtr(v float64) *float64 { return &v }

var _ = regexp.QuoteMeta

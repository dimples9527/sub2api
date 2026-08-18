package repository

import (
	"context"
	"regexp"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/require"
)

func TestSupplierProviderRepositoryListCostTrendsMergesUpstreamAndLocal(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	repo := NewSupplierProviderRepository(db)
	start := time.Date(2026, 7, 28, 0, 0, 0, 0, time.UTC)
	end := time.Date(2026, 7, 31, 0, 0, 0, 0, time.UTC)

	mock.ExpectQuery(regexp.QuoteMeta(`SELECT TO_CHAR(d.stat_date, 'YYYY-MM-DD') AS date,
       COALESCE(SUM(d.today_cost), 0) AS upstream_cost,
       SUM(d.raw_upstream_cost) AS raw_upstream_cost,
       COALESCE(string_agg(DISTINCT d.cost_warning, '；') FILTER (WHERE d.cost_warning IS NOT NULL), '') AS cost_warning
FROM supplier_provider_daily_stats d
JOIN supplier_providers p ON p.id = d.provider_id AND p.deleted_at IS NULL
WHERE d.stat_date >= $1::date
  AND d.stat_date < $2::date
GROUP BY d.stat_date
ORDER BY d.stat_date`)).
		WithArgs(start, end).
		WillReturnRows(sqlmock.NewRows([]string{"date", "upstream_cost", "raw_upstream_cost", "cost_warning"}).
			AddRow("2026-07-28", 12.5, 12.5, "上游成本 12.50 与本地成本 10.00 偏差 20%，已按本地成本展示").
			AddRow("2026-07-29", 7, nil, ""))

	mock.ExpectQuery("matched_accounts").
		WithArgs(start, end, sqlmock.AnyArg()).
		WillReturnRows(sqlmock.NewRows([]string{"date", "local_cost"}).
			AddRow("2026-07-28", 10.0).
			AddRow("2026-07-30", 4.5))

	points, err := repo.ListCostTrends(context.Background(), start, end, 0)
	require.NoError(t, err)
	require.Len(t, points, 3)
	require.Equal(t, "2026-07-28", points[0].Date)
	require.Equal(t, 12.5, points[0].UpstreamCost)
	require.Equal(t, 10.0, points[0].LocalCost)
	require.NotNil(t, points[0].RawUpstreamCost)
	require.Equal(t, 12.5, *points[0].RawUpstreamCost)
	require.Contains(t, points[0].Warning, "已按本地成本展示")
	require.Equal(t, "2026-07-29", points[1].Date)
	require.Equal(t, 7.0, points[1].UpstreamCost)
	require.Equal(t, 0.0, points[1].LocalCost)
	require.Nil(t, points[1].RawUpstreamCost)
	require.Equal(t, "2026-07-30", points[2].Date)
	require.Equal(t, 0.0, points[2].UpstreamCost)
	require.Equal(t, 4.5, points[2].LocalCost)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestSupplierProviderRepositoryListCostTrendsFiltersByProvider(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	repo := NewSupplierProviderRepository(db)
	start := time.Date(2026, 7, 28, 0, 0, 0, 0, time.UTC)
	end := time.Date(2026, 7, 31, 0, 0, 0, 0, time.UTC)
	providerID := int64(9)

	mock.ExpectQuery("supplier_provider_daily_stats").
		WithArgs(start, end, providerID).
		WillReturnRows(sqlmock.NewRows([]string{"date", "upstream_cost", "raw_upstream_cost", "cost_warning"}).
			AddRow("2026-07-28", 3.5, nil, ""))

	mock.ExpectQuery("matched_accounts").
		WithArgs(start, end, sqlmock.AnyArg(), providerID).
		WillReturnRows(sqlmock.NewRows([]string{"date", "local_cost"}).
			AddRow("2026-07-28", 2.0))

	points, err := repo.ListCostTrends(context.Background(), start, end, providerID)
	require.NoError(t, err)
	require.Len(t, points, 1)
	require.Equal(t, "2026-07-28", points[0].Date)
	require.Equal(t, 3.5, points[0].UpstreamCost)
	require.Equal(t, 2.0, points[0].LocalCost)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestSupplierProviderRepositoryListBalanceCostsScansRows(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	repo := NewSupplierProviderRepository(db)
	start := time.Date(2026, 7, 28, 0, 0, 0, 0, time.UTC)
	end := time.Date(2026, 7, 30, 0, 0, 0, 0, time.UTC)

	mock.ExpectQuery("supplier_provider_metric_snapshots").
		WithArgs(start, end, sqlmock.AnyArg()).
		WillReturnRows(sqlmock.NewRows([]string{"stat_date", "provider_id", "cost"}).
			AddRow("2026-07-28", int64(7), 20.0).
			AddRow("2026-07-29", int64(7), 5.0))

	costs, err := repo.ListBalanceCosts(context.Background(), start, end, 0)
	require.NoError(t, err)
	require.Len(t, costs, 2)
	require.Equal(t, "2026-07-28", costs[0].Date)
	require.Equal(t, int64(7), costs[0].ProviderID)
	require.Equal(t, 20.0, costs[0].BalanceCost)
	require.Equal(t, "2026-07-29", costs[1].Date)
	require.Equal(t, int64(7), costs[1].ProviderID)
	require.Equal(t, 5.0, costs[1].BalanceCost)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestSupplierProviderRepositoryListBalanceCostsFiltersByProvider(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	repo := NewSupplierProviderRepository(db)
	start := time.Date(2026, 7, 28, 0, 0, 0, 0, time.UTC)
	end := time.Date(2026, 7, 30, 0, 0, 0, 0, time.UTC)
	providerID := int64(7)

	mock.ExpectQuery("supplier_provider_metric_snapshots").
		WithArgs(start, end, sqlmock.AnyArg(), providerID).
		WillReturnRows(sqlmock.NewRows([]string{"stat_date", "provider_id", "cost"}).
			AddRow("2026-07-28", int64(7), 20.0))

	costs, err := repo.ListBalanceCosts(context.Background(), start, end, providerID)
	require.NoError(t, err)
	require.Len(t, costs, 1)
	require.Equal(t, "2026-07-28", costs[0].Date)
	require.Equal(t, int64(7), costs[0].ProviderID)
	require.Equal(t, 20.0, costs[0].BalanceCost)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestSupplierProviderRepositoryListBalanceCostsNoRows(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	repo := NewSupplierProviderRepository(db)
	start := time.Date(2026, 7, 28, 0, 0, 0, 0, time.UTC)
	end := time.Date(2026, 7, 30, 0, 0, 0, 0, time.UTC)

	mock.ExpectQuery("supplier_provider_metric_snapshots").
		WithArgs(start, end, sqlmock.AnyArg()).
		WillReturnRows(sqlmock.NewRows([]string{"stat_date", "provider_id", "cost"}))

	costs, err := repo.ListBalanceCosts(context.Background(), start, end, 0)
	require.NoError(t, err)
	require.Empty(t, costs)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestSupplierProviderRepositoryListCostBreakdownsAggregatesByProvider(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	repo := NewSupplierProviderRepository(db)
	start := time.Date(2026, 7, 28, 0, 0, 0, 0, time.UTC)
	end := time.Date(2026, 7, 31, 0, 0, 0, 0, time.UTC)

	mock.ExpectQuery("WITH provider_account_matches").
		WithArgs(start, end).
		WillReturnRows(sqlmock.NewRows([]string{"id", "name", "provider_type", "upstream_cost", "local_cost", "raw_upstream_cost", "cost_warning"}).
			AddRow(int64(7), "主供应商", "sub2api", 42.5, 17.25, 42.5, "上游成本 42.50 与本地成本 17.25 偏差 59%，已按本地成本展示").
			AddRow(int64(9), "备用供应商", "newapi", 8.0, 3.5, 0.0, ""))

	breakdowns, err := repo.ListCostBreakdowns(context.Background(), start, end, 0)
	require.NoError(t, err)
	require.Len(t, breakdowns, 2)
	require.Equal(t, int64(7), breakdowns[0].ProviderID)
	require.Equal(t, "主供应商", breakdowns[0].ProviderName)
	require.Equal(t, 42.5, breakdowns[0].UpstreamCost)
	require.Equal(t, 17.25, breakdowns[0].LocalCost)
	require.Equal(t, 42.5, breakdowns[0].RawUpstreamCost)
	require.Contains(t, breakdowns[0].CostWarning, "已按本地成本展示")
	require.Equal(t, int64(9), breakdowns[1].ProviderID)
	require.Equal(t, 0.0, breakdowns[1].RawUpstreamCost)
	require.Equal(t, "", breakdowns[1].CostWarning)
	require.NoError(t, mock.ExpectationsWereMet())
}

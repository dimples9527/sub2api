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
       COALESCE(SUM(d.today_cost), 0) AS upstream_cost
FROM supplier_provider_daily_stats d
JOIN supplier_providers p ON p.id = d.provider_id AND p.deleted_at IS NULL
WHERE d.stat_date >= $1::date
  AND d.stat_date < $2::date
GROUP BY d.stat_date
ORDER BY d.stat_date`)).
		WithArgs(start, end).
		WillReturnRows(sqlmock.NewRows([]string{"date", "upstream_cost"}).
			AddRow("2026-07-28", 12.5).
			AddRow("2026-07-29", 7))

	mock.ExpectQuery("matched_accounts").
		WithArgs(start, end, sqlmock.AnyArg()).
		WillReturnRows(sqlmock.NewRows([]string{"date", "local_cost"}).
			AddRow("2026-07-28", 10.0).
			AddRow("2026-07-30", 4.5))

	points, err := repo.ListCostTrends(context.Background(), start, end)
	require.NoError(t, err)
	require.Len(t, points, 3)
	require.Equal(t, "2026-07-28", points[0].Date)
	require.Equal(t, 12.5, points[0].UpstreamCost)
	require.Equal(t, 10.0, points[0].LocalCost)
	require.Equal(t, "2026-07-29", points[1].Date)
	require.Equal(t, 7.0, points[1].UpstreamCost)
	require.Equal(t, 0.0, points[1].LocalCost)
	require.Equal(t, "2026-07-30", points[2].Date)
	require.Equal(t, 0.0, points[2].UpstreamCost)
	require.Equal(t, 4.5, points[2].LocalCost)
	require.NoError(t, mock.ExpectationsWereMet())
}

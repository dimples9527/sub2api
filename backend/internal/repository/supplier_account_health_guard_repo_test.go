package repository

import (
	"context"
	"regexp"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/stretchr/testify/require"
)

func TestSupplierAccountHealthGuardRepositoryListsEnabledProviderAccounts(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	rows := supplierAccountHealthGuardRows().
		AddRow(int64(11), int64(1), "供应商甲", "key-1", "账号一", 1, int64(21), "本地账号一", "openai", "active", true, []byte(`{"supplier_health_guard_failure_count":2}`)).
		AddRow(int64(12), int64(2), "供应商乙", "key-2", "账号二", 0, nil, "", "", "", false, []byte(`{}`)).
		AddRow(int64(13), int64(3), "供应商丙", "key-3", "账号三", 2, nil, "", "", "", false, []byte(`{}`)).
		AddRow(int64(14), int64(4), "供应商丁", "key-4", "账号四", 1, int64(21), "本地账号一", "openai", "active", true, []byte(`{"supplier_health_guard_failure_count":2}`))

	mock.ExpectQuery(regexp.MustCompile(`(?s)a\.active\s*=\s*TRUE.*p\.enabled\s*=\s*TRUE`).String()).WillReturnRows(rows)

	repo := NewSupplierAccountHealthGuardRepository(db)
	items, err := repo.ListAccountHealthGuardCandidates(context.Background())

	require.NoError(t, err)
	require.Len(t, items, 4)
	require.Equal(t, service.SupplierAccountHealthGuardMatchMatched, items[0].MatchStatus)
	require.Equal(t, service.SupplierAccountHealthGuardMatchUnmatched, items[1].MatchStatus)
	require.Equal(t, service.SupplierAccountHealthGuardMatchConflict, items[2].MatchStatus)
	require.Equal(t, int64(21), items[0].LocalAccountID)
	require.NotNil(t, items[0].LocalAccount)
	require.Equal(t, "本地账号一", items[0].LocalAccount.Name)
	require.Equal(t, "openai", items[0].LocalAccount.Platform)
	require.Equal(t, "active", items[0].LocalAccount.Status)
	require.True(t, items[0].LocalAccount.Schedulable)
	require.Equal(t, 2, parseRepositoryTestInt(items[0].LocalAccount.Extra["supplier_health_guard_failure_count"]))
	require.Equal(t, int64(21), items[3].LocalAccountID, "仓储层必须保留映射到同一本地账号的多个供应商来源")
	require.Equal(t, int64(4), items[3].Source.ProviderID)
	require.NoError(t, mock.ExpectationsWereMet())
}

func supplierAccountHealthGuardRows() *sqlmock.Rows {
	return sqlmock.NewRows([]string{
		"provider_account_id", "provider_id", "provider_name", "upstream_account_key", "upstream_account_name",
		"match_count", "local_account_id", "local_account_name", "local_account_platform", "local_account_status",
		"local_account_schedulable", "local_account_extra",
	})
}

func parseRepositoryTestInt(value any) int {
	switch number := value.(type) {
	case float64:
		return int(number)
	case int:
		return number
	default:
		return 0
	}
}

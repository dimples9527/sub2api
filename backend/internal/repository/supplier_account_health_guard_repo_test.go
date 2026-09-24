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
		AddRow(int64(11), int64(1), "供应商甲", "key-1", "账号一", 1, int64(21), "本地账号一", "openai", "grok", "grok", "active", true, []byte(`{"supplier_health_guard_failure_count":2}`)).
		AddRow(int64(12), int64(2), "供应商乙", "key-2", "账号二", 0, nil, "", "", "", "", "", false, []byte(`{}`)).
		AddRow(int64(13), int64(3), "供应商丙", "key-3", "账号三", 2, nil, "", "", "", "", "", false, []byte(`{}`)).
		AddRow(int64(14), int64(4), "供应商丁", "key-4", "账号四", 1, int64(21), "本地账号一", "openai", "grok", "grok", "active", true, []byte(`{"supplier_health_guard_failure_count":2}`))

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
	require.Equal(t, "grok", items[0].PlatformOverride)
	require.Equal(t, "grok", items[0].EffectivePlatform)
	require.Equal(t, "active", items[0].LocalAccount.Status)
	require.True(t, items[0].LocalAccount.Schedulable)
	require.Equal(t, 2, parseRepositoryTestInt(items[0].LocalAccount.Extra["supplier_health_guard_failure_count"]))
	require.Equal(t, int64(21), items[3].LocalAccountID, "仓储层必须保留映射到同一本地账号的多个供应商来源")
	require.Equal(t, int64(4), items[3].Source.ProviderID)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestSupplierAccountHealthGuardRepositoryExplainsUnavailableReasons(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	rows := sqlmock.NewRows([]string{"local_account_id", "cause"}).
		AddRow(int64(21), "provider_disabled").
		AddRow(int64(22), "provider_account_inactive").
		AddRow(int64(23), "no_provider_account").
		AddRow(int64(24), "local_missing").
		AddRow(int64(25), "match_conflict").
		AddRow(int64(26), "")

	mock.ExpectQuery(regexp.MustCompile(`(?s)CASE.*provider_disabled.*FROM local_account`).String()).WillReturnRows(rows)

	repo := NewSupplierAccountHealthGuardRepository(db)
	causes, err := repo.ListAccountHealthGuardUnavailableReasons(context.Background(), []int64{21, 22, 23, 24, 25, 26})

	require.NoError(t, err)
	require.Len(t, causes, 6)
	require.Equal(t, service.SupplierAccountHealthGuardCauseProviderDisabled, causes[21])
	require.Equal(t, service.SupplierAccountHealthGuardCauseProviderAccountInactive, causes[22])
	require.Equal(t, service.SupplierAccountHealthGuardCauseNoProviderAccount, causes[23])
	require.Equal(t, service.SupplierAccountHealthGuardCauseLocalMissing, causes[24])
	require.Equal(t, service.SupplierAccountHealthGuardCauseMatchConflict, causes[25])
	require.Equal(t, service.SupplierAccountHealthGuardCauseUnknown, causes[26], "空成因要落回 Unknown，明细行才会沿用原来的文案")
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestSupplierAccountHealthGuardRepositorySkipsReasonQueryWithoutAccountIDs(t *testing.T) {
	// 没有账号要诊断时一次库都不该查 —— 正常路径（全部账号可用）会走到这里。
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	repo := NewSupplierAccountHealthGuardRepository(db)
	causes, err := repo.ListAccountHealthGuardUnavailableReasons(context.Background(), nil)

	require.NoError(t, err)
	require.NotNil(t, causes)
	require.Empty(t, causes)
	require.NoError(t, mock.ExpectationsWereMet())
}

func supplierAccountHealthGuardRows() *sqlmock.Rows {
	return sqlmock.NewRows([]string{
		"provider_account_id", "provider_id", "provider_name", "upstream_account_key", "upstream_account_name",
		"match_count", "local_account_id", "local_account_name", "local_account_platform", "platform_override", "effective_platform", "local_account_status",
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

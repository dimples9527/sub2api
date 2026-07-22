package repository

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

type supplierProviderGroupSQLExecutorStub struct {
	query     string
	args      []any
	execErr   error
	execCalls int
}

func (s *supplierProviderGroupSQLExecutorStub) ExecContext(_ context.Context, query string, args ...any) (sql.Result, error) {
	s.execCalls++
	s.query = query
	s.args = args
	return nil, s.execErr
}

func (s *supplierProviderGroupSQLExecutorStub) QueryContext(context.Context, string, ...any) (*sql.Rows, error) {
	return nil, errors.New("unexpected QueryContext call")
}

func TestClearSupplierProviderGroupLocalMapping(t *testing.T) {
	t.Run("清理本地分组映射", func(t *testing.T) {
		exec := &supplierProviderGroupSQLExecutorStub{}

		err := clearSupplierProviderGroupLocalMapping(context.Background(), exec, int64(42))

		expectedSQL := `
UPDATE supplier_provider_groups
SET local_group_id = NULL,
    auto_match_status = 'unmatched',
    auto_match_ignored = TRUE,
    matched_upstream_name = NULL,
    name_change_pending = FALSE,
    rate_guard_selected = FALSE,
    rate_guard_selection_mode = '',
    updated_at = NOW()
WHERE local_group_id = $1
`
		require.NoError(t, err)
		require.Equal(t, strings.Join(strings.Fields(expectedSQL), " "), strings.Join(strings.Fields(exec.query), " "))
		require.Equal(t, []any{int64(42)}, exec.args)
		require.Equal(t, 1, exec.execCalls)
	})

	t.Run("原样传播执行错误", func(t *testing.T) {
		execErr := errors.New("exec failed")
		exec := &supplierProviderGroupSQLExecutorStub{execErr: execErr}

		err := clearSupplierProviderGroupLocalMapping(context.Background(), exec, int64(42))

		require.ErrorIs(t, err, execErr)
		require.Equal(t, 1, exec.execCalls)
	})
}

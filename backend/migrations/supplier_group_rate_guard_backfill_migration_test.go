package migrations

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestSupplierGroupRateGuardBackfillSelectsSoleActiveMapping(t *testing.T) {
	content, err := FS.ReadFile("185_supplier_group_rate_guard_backfill.sql")
	require.NoError(t, err)

	sqlText := string(content)
	require.Contains(t, sqlText, "COUNT(*) FILTER (WHERE active = TRUE) = 1")
	require.Contains(t, sqlText, "COUNT(*) FILTER (WHERE rate_guard_selected = TRUE AND active = FALSE) = 0")
	require.Contains(t, sqlText, "rate_guard_selected = TRUE")
	require.Contains(t, sqlText, "rate_guard_selection_mode = 'auto'")
}

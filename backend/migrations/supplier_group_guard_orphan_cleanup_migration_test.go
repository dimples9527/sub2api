package migrations

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestSupplierGroupGuardOrphanCleanupClearsDeletedLocalGroupMappings(t *testing.T) {
	content, err := FS.ReadFile("186_supplier_group_guard_orphan_cleanup.sql")
	require.NoError(t, err)

	sqlText := string(content)
	require.Contains(t, sqlText, "local_group_id = NULL")
	require.Contains(t, sqlText, "rate_guard_selected = FALSE")
	require.Contains(t, sqlText, "rate_guard_selection_mode = ''")
	require.Contains(t, sqlText, "lg.deleted_at IS NULL")
}

package migrations

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestSupplierGroupRateGuardIgnoreMigrationDefinesIgnoreFlag(t *testing.T) {
	content, err := FS.ReadFile("194_supplier_group_rate_guard_ignore.sql")
	require.NoError(t, err)

	require.Contains(t, string(content), "rate_guard_ignored BOOLEAN NOT NULL DEFAULT FALSE")
}

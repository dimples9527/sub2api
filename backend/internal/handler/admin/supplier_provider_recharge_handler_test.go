package admin

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestParseSupplierProviderRechargeDateRangeUsesShanghaiDayRange(t *testing.T) {
	start, end, ok := parseSupplierProviderRechargeDateRange("2026-08-18", "2026-08-18")

	require.True(t, ok)
	require.Equal(t, "Asia/Shanghai", start.Location().String())
	require.Equal(t, time.Date(2026, 8, 18, 0, 0, 0, 0, start.Location()), start)
	require.Equal(t, time.Date(2026, 8, 19, 0, 0, 0, 0, end.Location()), end)
}

func TestParseSupplierProviderRechargeDateRangeRejectsIncompleteAndReverseRanges(t *testing.T) {
	_, _, ok := parseSupplierProviderRechargeDateRange("2026-08-18", "")
	require.False(t, ok)

	_, _, ok = parseSupplierProviderRechargeDateRange("2026-08-19", "2026-08-18")
	require.False(t, ok)
}

func TestParseSupplierProviderRechargeOptionalID(t *testing.T) {
	id, ok := parseSupplierProviderRechargeOptionalID(" 42 ")
	require.True(t, ok)
	require.EqualValues(t, 42, id)

	_, ok = parseSupplierProviderRechargeOptionalID("0")
	require.False(t, ok)
}

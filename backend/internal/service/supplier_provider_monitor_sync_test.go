package service

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestSupplierMonitorAccountMatchCarriesLocalGroups(t *testing.T) {
	account := SupplierProviderAccount{
		ID:               11,
		LocalAccountID:   ptrInt64(101),
		LocalAccountName: "本地账号",
		BindingGroups: []SupplierProviderAccountBindingGroup{
			{ID: 7, Name: "主分组"},
			{ID: 8, Name: "备用分组"},
		},
	}

	match := supplierMonitorAccountMatchFromAccount(account)

	require.Equal(t, int64(101), match.localAccountID)
	require.Equal(t, "本地账号", match.localAccountName)
	require.Equal(t, []int64{7, 8}, match.localGroupIDs)
	require.Equal(t, []string{"主分组", "备用分组"}, match.localGroupNames)
}

func TestSupplierMonitorAccountMatchKeysAllowProviderNamePrefix(t *testing.T) {
	provider := &SupplierProvider{Name: "\u5fd8\u5ddd", AccountNamePrefix: "\u5fd8\u5ddd-"}
	account := SupplierProviderAccount{Name: "\u5fd8\u5ddd-gpt\u7834\u7532\u0031", LocalAccountName: "\u5fd8\u5ddd-gpt\u7834\u7532\u0031"}

	keys := supplierMonitorAccountMatchKeys(provider, account)

	require.Contains(t, keys, "gpt\u7834\u7532\u0031")
}

func TestSupplierMonitorAccountMatchKeysAllowReorderedLatinAndChineseParts(t *testing.T) {
	provider := &SupplierProvider{Name: "\u5fd8\u5ddd", AccountNamePrefix: "\u5fd8\u5ddd-"}
	account := SupplierProviderAccount{Name: "\u7834\u7532gpt1", LocalAccountName: "\u5fd8\u5ddd-\u7834\u7532gpt1"}

	keys := supplierMonitorAccountMatchKeys(provider, account)

	require.Contains(t, keys, "gpt\u7834\u75321")
}

func TestSupplierMonitorMatchPrefersExplicitBindingByMonitorKey(t *testing.T) {
	monitor := SupplierProviderMonitorItem{Key: "2", Name: "Plus-\u7a33\u5b9a"}
	explicitBinding := SupplierProviderMonitorBinding{
		MonitorKey:       "2",
		MonitorName:      "Plus-\u7a33\u5b9a",
		LocalAccountID:   7,
		LocalAccountName: "\u7693\u60a6-\u798f\u5229-Codex\u9ad8\u5e76\u53d1",
		BindingGroups: []SupplierProviderAccountBindingGroup{
			{ID: 81, Name: "AAA"},
		},
	}
	nameMatch := supplierMonitorAccountMatch{localAccountID: 9, localAccountName: "\u7693\u60a6-Plus-\u7a33\u5b9a"}

	match := supplierMonitorMatchForMonitor(
		monitor,
		map[string]supplierMonitorAccountMatch{normalizeSupplierMonitorMatchKey(monitor.Name): nameMatch},
		supplierMonitorBindingMatchIndex([]SupplierProviderMonitorBinding{explicitBinding}),
	)

	require.Equal(t, int64(7), match.localAccountID)
	require.Equal(t, "\u7693\u60a6-\u798f\u5229-Codex\u9ad8\u5e76\u53d1", match.localAccountName)
	require.Equal(t, []int64{81}, match.localGroupIDs)
	require.Equal(t, []string{"AAA"}, match.localGroupNames)
}

func ptrInt64(value int64) *int64 {
	return &value
}

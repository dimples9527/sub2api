package service

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestSupplierProviderServiceCostTrendCache(t *testing.T) {
	repo := &supplierProviderRepoStub{costTrends: []SupplierProviderCostTrendPoint{
		{Date: "2026-09-01", UpstreamCost: 5, LocalCost: 3},
	}}
	svc := NewSupplierProviderService(repo, supplierEncryptorStub{})

	first, err := svc.ListCostTrendsByDateRange(context.Background(), "2026-09-01", "2026-09-02", 0)
	require.NoError(t, err)
	require.Equal(t, 1, repo.costTrendCalls)

	// 同一范围第二次应命中缓存，不再查询仓储。
	second, err := svc.ListCostTrendsByDateRange(context.Background(), "2026-09-01", "2026-09-02", 0)
	require.NoError(t, err)
	require.Equal(t, 1, repo.costTrendCalls)
	require.Equal(t, first, second)

	// 成本写入后缓存失效，下一次查询重新走仓储。
	invalidateSupplierCostTrendCache()
	_, err = svc.ListCostTrendsByDateRange(context.Background(), "2026-09-01", "2026-09-02", 0)
	require.NoError(t, err)
	require.Equal(t, 2, repo.costTrendCalls)

	// 清理缓存，避免影响同包其他用例。
	invalidateSupplierCostTrendCache()
}

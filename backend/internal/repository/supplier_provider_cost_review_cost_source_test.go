package repository

import (
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/stretchr/testify/require"
)

func TestPendingEffectiveCostFollowsCostSource(t *testing.T) {
	upstream := 12.3456789
	calculated := 10.1234567
	fallbackEffective := 8.0

	tests := []struct {
		name     string
		input    service.SupplierProviderCostReviewSyncInput
		expected float64
	}{
		{
			name: "接口成本优先模式采用上游值",
			input: service.SupplierProviderCostReviewSyncInput{
				CostSource:     service.SupplierCostSourceUpstream,
				UpstreamCost:   &upstream,
				CalculatedCost: &calculated,
			},
			expected: 12.345679,
		},
		{
			name: "智能模式默认采用计算值",
			input: service.SupplierProviderCostReviewSyncInput{
				CostSource:     service.SupplierCostSourceAuto,
				UpstreamCost:   &upstream,
				CalculatedCost: &calculated,
			},
			expected: 10.123457,
		},
		{
			name: "计算成本优先模式采用计算值",
			input: service.SupplierProviderCostReviewSyncInput{
				CostSource:     service.SupplierCostSourceCalculated,
				UpstreamCost:   &upstream,
				CalculatedCost: &calculated,
			},
			expected: 10.123457,
		},
		{
			name: "计算值缺失时回退本次生效成本",
			input: service.SupplierProviderCostReviewSyncInput{
				CostSource:    service.SupplierCostSourceCalculated,
				UpstreamCost:  &upstream,
				EffectiveCost: fallbackEffective,
			},
			expected: fallbackEffective,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.InDelta(t, tt.expected, pendingEffectiveCost(tt.input), 0.0000001)
		})
	}
}

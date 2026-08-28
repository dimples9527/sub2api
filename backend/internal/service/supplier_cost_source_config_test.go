package service

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

type supplierCostSourceRepoStub struct {
	globalSource     string
	savedGlobal      string
	resolution       SupplierCostSourceResolution
	resolveCallCount int
}

func (s *supplierCostSourceRepoStub) GetGlobalCostSource(context.Context) (string, error) {
	return s.globalSource, nil
}

func (s *supplierCostSourceRepoStub) SetGlobalCostSource(_ context.Context, source string) error {
	s.savedGlobal = source
	return nil
}

func (s *supplierCostSourceRepoStub) ListConfigs(context.Context) ([]SupplierCostSourceConfig, error) {
	return nil, nil
}

func (s *supplierCostSourceRepoStub) GetConfigByProviderID(context.Context, int64) (*SupplierCostSourceConfig, error) {
	return nil, nil
}

func (s *supplierCostSourceRepoStub) UpsertConfig(context.Context, SupplierCostSourceOverrideInput) (*SupplierCostSourceConfig, error) {
	return nil, nil
}

func (s *supplierCostSourceRepoStub) DeleteConfig(context.Context, int64) error {
	return nil
}

func (s *supplierCostSourceRepoStub) Resolve(context.Context, int64) (SupplierCostSourceResolution, error) {
	s.resolveCallCount++
	return s.resolution, nil
}

func TestSupplierCostSourceConfigService_CachesAndInvalidatesResolution(t *testing.T) {
	invalidateSupplierCostTrendCache()
	defer invalidateSupplierCostTrendCache()

	repo := &supplierCostSourceRepoStub{
		globalSource: SupplierCostSourceAuto,
		resolution: SupplierCostSourceResolution{
			Source:     SupplierCostSourceUpstream,
			Threshold:  0.18,
			Overridden: true,
		},
	}
	svc := NewSupplierCostSourceConfigService(repo)
	ctx := context.Background()

	first, err := svc.ResolveCostSource(ctx, 8)
	require.NoError(t, err)
	second, err := svc.ResolveCostSource(ctx, 8)
	require.NoError(t, err)
	require.Equal(t, repo.resolution, first)
	require.Equal(t, first, second)
	require.Equal(t, 1, repo.resolveCallCount)

	_, err = svc.UpdateGlobalCostSource(ctx, SupplierCostSourceCalculated)
	require.NoError(t, err)
	require.Equal(t, SupplierCostSourceCalculated, repo.savedGlobal)

	third, err := svc.ResolveCostSource(ctx, 8)
	require.NoError(t, err)
	require.Equal(t, repo.resolution, third)
	require.Equal(t, 2, repo.resolveCallCount)
}

func TestSupplierCostSourceConfigService_Validation(t *testing.T) {
	repo := &supplierCostSourceRepoStub{globalSource: "invalid"}
	svc := NewSupplierCostSourceConfigService(repo)
	ctx := context.Background()

	settings, err := svc.GetSettings(ctx)
	require.NoError(t, err)
	require.Equal(t, SupplierCostSourceAuto, settings.CostSource)

	invalidThreshold := 0.0
	_, err = svc.UpsertOverride(ctx, SupplierCostSourceOverrideInput{ProviderID: 8, CostSource: "manual", Threshold: &invalidThreshold})
	require.Error(t, err)
	_, err = svc.UpsertOverride(ctx, SupplierCostSourceOverrideInput{ProviderID: 8, CostSource: SupplierCostSourceAuto, Threshold: &invalidThreshold})
	require.NoError(t, err)
}

package service

import (
	"context"
	"errors"
	"math"
	"testing"

	"github.com/stretchr/testify/require"
)

type supplierCostDeviationSettingRepoStub struct {
	threshold float64
	err       error
}

func (s *supplierCostDeviationSettingRepoStub) GetThreshold(_ context.Context) (float64, error) {
	if s.err != nil {
		return 0, s.err
	}
	return s.threshold, nil
}

func (s *supplierCostDeviationSettingRepoStub) SetThreshold(_ context.Context, threshold float64) error {
	if s.err != nil {
		return s.err
	}
	s.threshold = threshold
	return nil
}

func TestSupplierCostDeviationThreshold_Defaults(t *testing.T) {
	repo := &supplierCostDeviationSettingRepoStub{threshold: DefaultSupplierCostDeviationThreshold}
	svc := NewSupplierCostDeviationSettingsService(repo)
	got, err := svc.GetSupplierCostDeviationSettings(context.Background())
	require.NoError(t, err)
	require.Equal(t, DefaultSupplierCostDeviationThreshold, got.Threshold)
	require.Equal(t, DefaultSupplierCostDeviationThreshold, svc.SupplierCostDeviationThreshold(context.Background()))
}

func TestSupplierCostDeviationThreshold_ReadsConfiguredValue(t *testing.T) {
	repo := &supplierCostDeviationSettingRepoStub{threshold: 0.3}
	svc := NewSupplierCostDeviationSettingsService(repo)

	require.Equal(t, 0.3, svc.SupplierCostDeviationThreshold(context.Background()))

	got, err := svc.GetSupplierCostDeviationSettings(context.Background())
	require.NoError(t, err)
	require.Equal(t, 0.3, got.Threshold)
}

func TestSupplierCostDeviationThreshold_FallsBackOnInvalidValue(t *testing.T) {
	cases := []struct {
		threshold float64
		expected  float64
	}{
		{-1, DefaultSupplierCostDeviationThreshold},
		{0, 0},
		{1.5, 1.5},
		{0.95, 0.95},
	}
	for _, tc := range cases {
		repo := &supplierCostDeviationSettingRepoStub{threshold: tc.threshold}
		svc := NewSupplierCostDeviationSettingsService(repo)
		require.Equalf(t, tc.expected, svc.SupplierCostDeviationThreshold(context.Background()), "threshold=%v", tc.threshold)
	}

	repo := &supplierCostDeviationSettingRepoStub{err: errors.New("db down")}
	svc := NewSupplierCostDeviationSettingsService(repo)
	require.Equal(t, DefaultSupplierCostDeviationThreshold, svc.SupplierCostDeviationThreshold(context.Background()))
}

func TestUpdateSupplierCostDeviationSettings_PersistsAndValidates(t *testing.T) {
	repo := &supplierCostDeviationSettingRepoStub{}
	svc := NewSupplierCostDeviationSettingsService(repo)

	got, err := svc.UpdateSupplierCostDeviationSettings(context.Background(), 0.7)
	require.NoError(t, err)
	require.Equal(t, 0.7, got.Threshold)
	require.Equal(t, 0.7, repo.threshold)

	// 大于 100% 的阈值按原值保存。
	got, err = svc.UpdateSupplierCostDeviationSettings(context.Background(), 2.0)
	require.NoError(t, err)
	require.Equal(t, 2.0, got.Threshold)
	require.Equal(t, 2.0, svc.SupplierCostDeviationThreshold(context.Background()))

	// 0% 是有效阈值。
	got, err = svc.UpdateSupplierCostDeviationSettings(context.Background(), 0)
	require.NoError(t, err)
	require.Equal(t, float64(0), got.Threshold)
	require.Equal(t, float64(0), svc.SupplierCostDeviationThreshold(context.Background()))

	// 负数会被拒绝，且不会修改已保存的值。
	_, err = svc.UpdateSupplierCostDeviationSettings(context.Background(), -0.1)
	require.Error(t, err)
	require.Equal(t, float64(0), repo.threshold)
	repo.err = errors.New("db down")
	_, err = svc.UpdateSupplierCostDeviationSettings(context.Background(), 0.6)
	require.Error(t, err)
	require.Equal(t, float64(0), repo.threshold)
}

func TestValidateSupplierCostDeviationThreshold(t *testing.T) {
	require.NoError(t, validateSupplierCostDeviationThreshold(0.5))
	require.NoError(t, validateSupplierCostDeviationThreshold(0))
	require.NoError(t, validateSupplierCostDeviationThreshold(1.5))
	require.Error(t, validateSupplierCostDeviationThreshold(-0.1))
	require.Error(t, validateSupplierCostDeviationThreshold(math.NaN()))
	require.Error(t, validateSupplierCostDeviationThreshold(math.Inf(1)))
}

package service

import (
	"context"
	"errors"
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
	svc := NewSupplierCostDeviationSettingsService(&supplierCostDeviationSettingRepoStub{})

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
		{0, DefaultSupplierCostDeviationThreshold},
		{1.5, supplierCostDeviationThresholdMax},
		{0.01, supplierCostDeviationThresholdMin},
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

func TestUpdateSupplierCostDeviationSettings_PersistsClamped(t *testing.T) {
	repo := &supplierCostDeviationSettingRepoStub{}
	svc := NewSupplierCostDeviationSettingsService(repo)

	got, err := svc.UpdateSupplierCostDeviationSettings(context.Background(), 0.7)
	require.NoError(t, err)
	require.Equal(t, 0.7, got.Threshold)
	require.Equal(t, 0.7, repo.threshold)

	// 超上限 clamp 到 0.95。
	got, err = svc.UpdateSupplierCostDeviationSettings(context.Background(), 2.0)
	require.NoError(t, err)
	require.Equal(t, supplierCostDeviationThresholdMax, got.Threshold)
	require.Equal(t, supplierCostDeviationThresholdMax, svc.SupplierCostDeviationThreshold(context.Background()))

	// 低于下限 clamp 到 0.05。
	got, err = svc.UpdateSupplierCostDeviationSettings(context.Background(), 0.01)
	require.NoError(t, err)
	require.Equal(t, supplierCostDeviationThresholdMin, got.Threshold)
	require.Equal(t, supplierCostDeviationThresholdMin, svc.SupplierCostDeviationThreshold(context.Background()))

	// 更新失败时返回错误且不修改阈值。
	repo.err = errors.New("db down")
	_, err = svc.UpdateSupplierCostDeviationSettings(context.Background(), 0.6)
	require.Error(t, err)
	require.Equal(t, supplierCostDeviationThresholdMin, repo.threshold)
}

func TestClampSupplierCostDeviationThreshold(t *testing.T) {
	require.Equal(t, 0.5, clampSupplierCostDeviationThreshold(0.5))
	require.Equal(t, supplierCostDeviationThresholdMin, clampSupplierCostDeviationThreshold(0.01))
	require.Equal(t, supplierCostDeviationThresholdMax, clampSupplierCostDeviationThreshold(1.5))
}

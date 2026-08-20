package service

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestBuildSupplierProviderGroupHealthTrendsUsesLatestAccountResultPerBucket(t *testing.T) {
	now := time.Date(2026, 7, 27, 10, 30, 0, 0, time.UTC)
	samples := []SupplierProviderGroupHealthSample{
		{GroupID: 101, AccountID: 11, Status: SupplierAccountHealthGuardStatusFailed, Latency: 410, FinishedAt: now.Add(-4 * time.Minute)},
		{GroupID: 101, AccountID: 11, Status: SupplierAccountHealthGuardStatusHealthy, Latency: 120, FinishedAt: now.Add(-2 * time.Minute)},
		{GroupID: 101, AccountID: 12, Status: SupplierAccountHealthGuardStatusSlow, Latency: 18000, FinishedAt: now.Add(-3 * time.Minute)},
		{GroupID: 102, AccountID: 21, Status: SupplierAccountHealthGuardStatusUnavailable, Latency: 0, FinishedAt: now.Add(-1 * time.Minute)},
	}

	trends := BuildSupplierProviderGroupHealthTrends(samples, SupplierProviderGroupHealthTrendParams{
		GroupIDs: []int64{101, 102}, Period: 90 * time.Minute, BucketCount: 18, Now: now,
	})

	require.Equal(t, 100.0, trends[101].Trend[17].Availability)
	require.Equal(t, "green", trends[101].Trend[17].Tone)
	require.Equal(t, 2, trends[101].Trend[17].TestedAccountCount)
	require.Equal(t, int64(9060), trends[101].Trend[17].Latency)
	require.Equal(t, 0.0, trends[102].Trend[17].Availability)
	require.Equal(t, "red", trends[102].Trend[17].Tone)
}

func TestBuildSupplierProviderGroupHealthTrendsSkipsSkippedSamplesAndLeavesEmptyBucketsOut(t *testing.T) {
	now := time.Date(2026, 7, 27, 10, 30, 0, 0, time.UTC)
	samples := []SupplierProviderGroupHealthSample{
		{GroupID: 101, AccountID: 11, Status: SupplierAccountHealthGuardStatusHealthy, Latency: 100, FinishedAt: now.Add(-12 * time.Minute)},
		{GroupID: 101, AccountID: 12, Status: SupplierAccountHealthGuardStatusFailed, Latency: 300, FinishedAt: now.Add(-12 * time.Minute)},
		{GroupID: 101, AccountID: 13, Status: SupplierAccountHealthGuardStatusSkipped, Latency: 0, FinishedAt: now.Add(-12 * time.Minute)},
	}

	trends := BuildSupplierProviderGroupHealthTrends(samples, SupplierProviderGroupHealthTrendParams{
		GroupIDs: []int64{101}, Period: 90 * time.Minute, BucketCount: 18, Now: now,
	})

	require.Contains(t, trends, int64(101))
	require.Equal(t, 50.0, trends[101].Availability)
	require.Equal(t, "green", trends[101].Trend[15].Tone)
	require.Equal(t, 2, trends[101].Trend[15].TestedAccountCount)
	require.Len(t, trends[101].Trend, 18)
	require.Equal(t, "gray", trends[101].Trend[0].Tone)
	require.Zero(t, trends[101].Trend[0].TestedAccountCount)
}

func TestSupplierProviderGroupHealthTrendToneUsesFortyPercentGreenThreshold(t *testing.T) {
	require.Equal(t, "green", supplierProviderGroupHealthTrendTone(40))
	require.Equal(t, "yellow", supplierProviderGroupHealthTrendTone(39.99))
	require.Equal(t, "red", supplierProviderGroupHealthTrendTone(0))
}

func TestBuildSupplierProviderGroupHealthTrendsOmitsGroupsWithoutUsableSamples(t *testing.T) {
	now := time.Date(2026, 7, 27, 10, 30, 0, 0, time.UTC)

	trends := BuildSupplierProviderGroupHealthTrends([]SupplierProviderGroupHealthSample{
		{GroupID: 101, AccountID: 11, Status: SupplierAccountHealthGuardStatusSkipped, FinishedAt: now.Add(-time.Minute)},
		{GroupID: 102, AccountID: 12, Status: SupplierAccountHealthGuardStatusHealthy, FinishedAt: now.Add(-91 * time.Minute)},
	}, SupplierProviderGroupHealthTrendParams{
		GroupIDs: []int64{101, 102, 103}, Period: 90 * time.Minute, BucketCount: 18, Now: now,
	})

	require.Empty(t, trends)
}

func TestBuildSupplierProviderGroupHealthTrendsPrefersLatestEighteenRawMonitorTimelinePoints(t *testing.T) {
	now := time.Date(2026, 8, 9, 12, 0, 0, 0, time.UTC)
	rawStart := now.Add(-25 * time.Hour)
	samples := []SupplierProviderGroupHealthSample{
		{GroupID: 101, AccountID: 11, Source: SupplierProviderGroupHealthTrendSource, Status: SupplierAccountHealthGuardStatusFailed, Latency: 30000, FinishedAt: now.Add(-time.Minute)},
		{GroupID: 102, AccountID: 21, Source: SupplierProviderGroupHealthTrendSource, Status: SupplierAccountHealthGuardStatusHealthy, Latency: 100, FinishedAt: now.Add(-time.Minute)},
	}
	for index := 0; index < 20; index++ {
		checkedAt := rawStart.Add(time.Duration(index) * time.Hour)
		status := SupplierAccountHealthGuardStatusSlow
		latency := int64(1000 + index)
		if index == 19 {
			status = SupplierAccountHealthGuardStatusHealthy
			latency = 1234
		}
		samples = append(samples, SupplierProviderGroupHealthSample{
			GroupID: 101, AccountID: 11, Source: SupplierProviderGroupHealthTrendMonitorSource,
			Status: status, Latency: latency, FinishedAt: checkedAt,
		})
	}
	// 同一账号、同一分钟的重入历史点只保留最后一次采样。
	samples = append(samples, SupplierProviderGroupHealthSample{
		GroupID: 101, AccountID: 11, Source: SupplierProviderGroupHealthTrendMonitorSource,
		Status: SupplierAccountHealthGuardStatusFailed, Latency: 30000, FinishedAt: rawStart.Add(19*time.Hour + 20*time.Second),
	})

	trends := BuildSupplierProviderGroupHealthTrends(samples, SupplierProviderGroupHealthTrendParams{
		GroupIDs: []int64{101, 102}, Period: 90 * time.Minute, BucketCount: 18, Now: now,
		PreferRawMonitorTimeline: true,
	})

	rawTrend := trends[101]
	require.Equal(t, SupplierProviderGroupHealthTrendMonitorSource, rawTrend.Source)
	require.Len(t, rawTrend.Trend, 18)
	require.Equal(t, rawStart.Add(2*time.Hour).Truncate(time.Minute), rawTrend.Trend[0].Time)
	require.Equal(t, rawStart.Add(19*time.Hour).Truncate(time.Minute), rawTrend.Trend[17].Time)
	require.Equal(t, 0.0, rawTrend.Trend[17].Availability)
	require.Equal(t, int64(30000), rawTrend.Trend[17].Latency)

	fallbackTrend := trends[102]
	require.Equal(t, SupplierProviderGroupHealthTrendSource, fallbackTrend.Source)
	require.Len(t, fallbackTrend.Trend, 18)
	require.Equal(t, 100.0, fallbackTrend.Availability)
}

func TestBuildSupplierProviderGroupHealthTrendsKeepsHealthyAccountWhenRawMonitorTimelineIsPreferred(t *testing.T) {
	now := time.Date(2026, 8, 20, 12, 0, 0, 0, time.UTC)
	trends := BuildSupplierProviderGroupHealthTrends([]SupplierProviderGroupHealthSample{
		{
			GroupID:    101,
			AccountID:  11,
			Source:     SupplierProviderGroupHealthTrendSource,
			Status:     SupplierAccountHealthGuardStatusHealthy,
			Latency:    120,
			FinishedAt: now,
		},
		{
			GroupID:    101,
			AccountID:  11,
			Source:     SupplierProviderGroupHealthTrendMonitorSource,
			Status:     SupplierAccountHealthGuardStatusFailed,
			Latency:    30000,
			FinishedAt: now.Add(-time.Minute),
		},
	}, SupplierProviderGroupHealthTrendParams{
		GroupIDs:                 []int64{101},
		Period:                   90 * time.Minute,
		BucketCount:              18,
		Now:                      now,
		PreferRawMonitorTimeline: true,
	})

	require.Contains(t, trends, int64(101))
	require.Equal(t, 100.0, trends[101].Availability)
	require.Equal(t, "green", trends[101].Trend[len(trends[101].Trend)-1].Tone)
}

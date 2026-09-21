package service

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestSupplierGroupMonitorSnapshotAvailability(t *testing.T) {
	require.Equal(t, 100.0, SupplierGroupMonitorSnapshotAvailability(SupplierAccountHealthGuardStatusHealthy))
	require.Equal(t, 100.0, SupplierGroupMonitorSnapshotAvailability(SupplierAccountHealthGuardStatusSlow))
	require.Equal(t, 0.0, SupplierGroupMonitorSnapshotAvailability(SupplierAccountHealthGuardStatusFailed))
	require.Equal(t, 0.0, SupplierGroupMonitorSnapshotAvailability(SupplierAccountHealthGuardStatusUnavailable))
}

func TestSupplierGroupMonitorSnapshotTone(t *testing.T) {
	require.Equal(t, "green", SupplierGroupMonitorSnapshotTone(SupplierAccountHealthGuardStatusHealthy))
	// slow 在可用率里算 100（与分组聚合一致），但灯色必须显黄：接客的那条账号慢了就是降级。
	require.Equal(t, "yellow", SupplierGroupMonitorSnapshotTone(SupplierAccountHealthGuardStatusSlow))
	require.Equal(t, "red", SupplierGroupMonitorSnapshotTone(SupplierAccountHealthGuardStatusFailed))
	require.Equal(t, "red", SupplierGroupMonitorSnapshotTone(SupplierAccountHealthGuardStatusUnavailable))
	// 认不出来的状态不猜：留空让调用方回退到按趋势点判断。
	require.Equal(t, "", SupplierGroupMonitorSnapshotTone("operational"))
}

func TestApplyLatestGroupMonitorSnapshotsOverridesOnlyLatestMoment(t *testing.T) {
	now := time.Date(2026, 9, 21, 12, 0, 0, 0, time.UTC)
	trend := SupplierProviderGroupHealthTrend{
		GroupID:      7,
		Availability: 50,
		Latency:      900,
		Time:         now.Add(-5 * time.Minute),
		Trend: []SupplierProviderGroupHealthTrendPoint{
			{Time: now.Add(-10 * time.Minute), Availability: 0, Latency: 1000, Tone: "red"},
			{Time: now.Add(-5 * time.Minute), Availability: 50, Latency: 900, Tone: "yellow"},
		},
	}
	snapshot := SupplierGroupMonitorSnapshot{
		GroupID:        7,
		LocalAccountID: 42,
		Source:         SupplierGroupMonitorSnapshotSourceMonitor,
		Status:         SupplierAccountHealthGuardStatusHealthy,
		LatencyMS:      120,
		Availability:   100,
		CheckedAt:      now.Add(-30 * time.Second),
	}

	got := ApplyLatestGroupMonitorSnapshots([]SupplierProviderGroupHealthTrend{trend}, []SupplierGroupMonitorSnapshot{snapshot})

	require.Len(t, got, 1)
	// 最新一刻换成调度账号自己的值。
	require.Equal(t, 100.0, got[0].Availability)
	require.Equal(t, int64(120), got[0].Latency)
	require.Equal(t, now.Add(-30*time.Second), got[0].Time)
	// 灯色跟着一起换，否则状态灯会停留在全组聚合算出来的黄灯上。
	require.Equal(t, "green", got[0].LatestTone)
	// 历史桶一格不动：那一刻本来是谁参与算的，就还是谁。
	require.Len(t, got[0].Trend, 2)
	require.Equal(t, 50.0, got[0].Trend[1].Availability)
	require.Equal(t, int64(900), got[0].Trend[1].Latency)
	require.Equal(t, now.Add(-5*time.Minute), got[0].Trend[1].Time)
}

func TestApplyLatestGroupMonitorSnapshotsKeepsAggregateWhenSnapshotMissing(t *testing.T) {
	now := time.Date(2026, 9, 21, 12, 0, 0, 0, time.UTC)
	// 分组 7 有快照，分组 8 没有：分组 8 保持原有聚合值，不能因为拿不到调度账号就变空。
	trends := []SupplierProviderGroupHealthTrend{
		{GroupID: 7, Availability: 50, Latency: 900, Time: now.Add(-5 * time.Minute)},
		{GroupID: 8, Availability: 50, Latency: 900, Time: now.Add(-5 * time.Minute)},
	}
	snapshots := []SupplierGroupMonitorSnapshot{{
		GroupID:      7,
		Source:       SupplierGroupMonitorSnapshotSourceHealthGuard,
		Status:       SupplierAccountHealthGuardStatusFailed,
		Availability: 0,
		LatencyMS:    0,
		CheckedAt:    now.Add(-time.Minute),
	}}

	got := ApplyLatestGroupMonitorSnapshots(trends, snapshots)

	require.Len(t, got, 2)
	require.Equal(t, 0.0, got[0].Availability)
	require.Equal(t, 50.0, got[1].Availability)
	require.Equal(t, int64(900), got[1].Latency)
	require.Equal(t, now.Add(-5*time.Minute), got[1].Time)
	// 有快照的分组给出调度账号的灯色，没快照的分组留空让调用方回退到按趋势点判断。
	require.Equal(t, "red", got[0].LatestTone)
	require.Equal(t, "", got[1].LatestTone)
}

func TestApplyLatestGroupMonitorSnapshotsPicksNewestSnapshotPerGroup(t *testing.T) {
	now := time.Date(2026, 9, 21, 12, 0, 0, 0, time.UTC)
	trends := []SupplierProviderGroupHealthTrend{{GroupID: 7, Availability: 50, Latency: 900, Time: now}}
	// 同一分组两个来源各一条：取时间更新的那条，而不是先出现的那条。
	snapshots := []SupplierGroupMonitorSnapshot{
		{GroupID: 7, Source: SupplierGroupMonitorSnapshotSourceMonitor, Status: SupplierAccountHealthGuardStatusHealthy, Availability: 100, LatencyMS: 111, CheckedAt: now.Add(-2 * time.Minute)},
		{GroupID: 7, Source: SupplierGroupMonitorSnapshotSourceHealthGuard, Status: SupplierAccountHealthGuardStatusSlow, Availability: 100, LatencyMS: 222, CheckedAt: now.Add(-time.Minute)},
	}

	got := ApplyLatestGroupMonitorSnapshots(trends, snapshots)

	require.Len(t, got, 1)
	require.Equal(t, int64(222), got[0].Latency)
	require.Equal(t, now.Add(-time.Minute), got[0].Time)
}

func TestApplyLatestGroupMonitorSnapshotsIgnoresUnrecognizedStatus(t *testing.T) {
	now := time.Date(2026, 9, 21, 12, 0, 0, 0, time.UTC)
	trends := []SupplierProviderGroupHealthTrend{{GroupID: 7, Availability: 50, Latency: 900, Time: now.Add(-5 * time.Minute)}}

	// 状态认不出来时整条不用，而不是"可用率换掉、灯色留空"——那样又会变成两个时刻拼出来的值。
	got := ApplyLatestGroupMonitorSnapshots(trends, []SupplierGroupMonitorSnapshot{{
		GroupID:      7,
		Source:       SupplierGroupMonitorSnapshotSourceMonitor,
		Status:       "operational",
		Availability: 0,
		CheckedAt:    now.Add(-time.Minute),
	}})

	require.Len(t, got, 1)
	require.Equal(t, 50.0, got[0].Availability)
	require.Equal(t, int64(900), got[0].Latency)
	require.Equal(t, "", got[0].LatestTone)
}

func TestApplyLatestGroupMonitorSnapshotsIgnoresEmptySnapshots(t *testing.T) {
	now := time.Date(2026, 9, 21, 12, 0, 0, 0, time.UTC)
	trends := []SupplierProviderGroupHealthTrend{{GroupID: 7, Availability: 50, Latency: 900, Time: now}}

	got := ApplyLatestGroupMonitorSnapshots(trends, nil)

	require.Equal(t, trends, got)
}

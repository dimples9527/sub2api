package service

import (
	"context"
	"time"
)

// 分组监控快照的来源，落库时用于区分两条写入链路。
const (
	SupplierGroupMonitorSnapshotSourceMonitor     = "supplier_monitor"
	SupplierGroupMonitorSnapshotSourceHealthGuard = "supplier_account_health_guard"
)

// SupplierGroupMonitorSnapshot 是某一时刻某分组「正在调度的那一条账号」的监控结果。
//
// 为什么要单独记一张表：accounts.schedulable 只反映当前状态，没有历史。
// 分组换人之后，从样本本身反推不出"这一刻到底是谁在跑"，只能看到"全组哪些账号有样本"。
// 于是在采样时把当时的调度账号连同结果一起落下来，之后查任何时刻都能拿到当时那条，
// 换人不会让新账号的数据盖掉旧时刻。
type SupplierGroupMonitorSnapshot struct {
	GroupID        int64
	LocalAccountID int64
	Source         string
	Status         string
	LatencyMS      int64
	Availability   float64
	CheckedAt      time.Time
}

// SupplierGroupMonitorSnapshotRepository 提供分组监控快照的写入与最新值读取。
type SupplierGroupMonitorSnapshotRepository interface {
	UpsertGroupMonitorSnapshots(ctx context.Context, snapshots []SupplierGroupMonitorSnapshot) error
	ListLatestGroupMonitorSnapshots(ctx context.Context, groupIDs []int64) ([]SupplierGroupMonitorSnapshot, error)
}

// SupplierGroupMonitorSnapshotAvailability 把一条采样的状态换算成可用率，
// 与趋势点按样本统计的口径保持一致：单个样本时 healthy/slow 记 100，其余记 0。
func SupplierGroupMonitorSnapshotAvailability(status string) float64 {
	switch status {
	case SupplierAccountHealthGuardStatusHealthy, SupplierAccountHealthGuardStatusSlow:
		return 100
	default:
		return 0
	}
}

// SupplierGroupMonitorSnapshotTone 把一条采样的状态换算成趋势点口径的灯色。
//
// 与 SupplierGroupMonitorSnapshotAvailability 分开是有意的：可用率把 slow 记成 100，
// 因为分组聚合口径里 healthy/slow 都算可用；但「当前正在接客的这条账号慢了」本身就是降级信号，
// 最新一刻的灯色要显黄，否则页面会在账号已经慢到被守护盯上时继续亮绿灯。
func SupplierGroupMonitorSnapshotTone(status string) string {
	switch status {
	case SupplierAccountHealthGuardStatusHealthy:
		return supplierProviderGroupHealthTrendToneGreen
	case SupplierAccountHealthGuardStatusSlow:
		return supplierProviderGroupHealthTrendToneYellow
	case SupplierAccountHealthGuardStatusFailed, SupplierAccountHealthGuardStatusUnavailable:
		return supplierProviderGroupHealthTrendToneRed
	default:
		return ""
	}
}

// ApplyLatestGroupMonitorSnapshots 用快照覆盖趋势的「最新一刻」，趋势线本身一格不动。
//
// 只替换 Availability / Latency / Time 三个汇总字段，历史桶保持原样：
// 之前是 A 在跑的时候，那些点仍然是 A 参与算出来的，不会因为现在换成了 B 就被改写。
// 没有快照的分组（尚未采集到、或该分组当前没有开启调度的账号）保持原聚合值，
// 这样分组被择优调度全关时页面不会突然变灰。
func ApplyLatestGroupMonitorSnapshots(
	trends []SupplierProviderGroupHealthTrend,
	snapshots []SupplierGroupMonitorSnapshot,
) []SupplierProviderGroupHealthTrend {
	if len(snapshots) == 0 {
		return trends
	}
	latestByGroup := make(map[int64]SupplierGroupMonitorSnapshot, len(snapshots))
	for _, snapshot := range snapshots {
		if snapshot.GroupID <= 0 || snapshot.CheckedAt.IsZero() {
			continue
		}
		if current, exists := latestByGroup[snapshot.GroupID]; exists && !snapshot.CheckedAt.After(current.CheckedAt) {
			continue
		}
		latestByGroup[snapshot.GroupID] = snapshot
	}

	for index := range trends {
		snapshot, exists := latestByGroup[trends[index].GroupID]
		if !exists {
			continue
		}
		// 灯色必须跟着一起换：可用率/耗时/时间已经换成调度账号的了，
		// 状态灯还按趋势最后一个点（全组聚合）算的话，三者就不是同一刻——
		// 会出现「调度账号已经挂了、灯还是绿的」。
		tone := SupplierGroupMonitorSnapshotTone(snapshot.Status)
		if tone == "" {
			// 认不出的状态整条不用：否则可用率被换成 0、灯色却还按聚合算，又变回不同源。
			continue
		}
		trends[index].Availability = snapshot.Availability
		trends[index].Latency = snapshot.LatencyMS
		trends[index].Time = snapshot.CheckedAt.UTC()
		trends[index].LatestTone = tone
	}
	return trends
}

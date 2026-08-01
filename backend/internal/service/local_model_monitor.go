package service

import "time"

// LocalModelMonitorTrendPoint 表示本地模型监控的一个趋势采样点。
type LocalModelMonitorTrendPoint struct {
	Time         time.Time `json:"time"`
	Availability float64   `json:"availability"`
	Latency      int64     `json:"latency"`
	Tone         string    `json:"tone,omitempty"`
	Valid        bool      `json:"-"`
}

// LocalModelMonitorTrend 表示某个本地分组的一条可用率趋势。
type LocalModelMonitorTrend struct {
	Availability float64                       `json:"availability"`
	Latency      int64                         `json:"latency"`
	Time         time.Time                     `json:"time"`
	Trend        []LocalModelMonitorTrendPoint `json:"trend"`
}

// AverageAvailability 返回趋势中有效采样点的平均可用率。
func (t LocalModelMonitorTrend) AverageAvailability() (float64, bool) {
	var total float64
	count := 0
	for _, point := range t.Trend {
		if !point.Valid {
			continue
		}
		total += point.Availability
		count++
	}
	if count == 0 {
		return 0, false
	}
	return total / float64(count), true
}

// LatestTime 返回趋势中最新有效采样点的时间。
func (t LocalModelMonitorTrend) LatestTime() time.Time {
	var latest time.Time
	for _, point := range t.Trend {
		if !point.Valid || point.Time.IsZero() {
			continue
		}
		if latest.IsZero() || point.Time.After(latest) {
			latest = point.Time
		}
	}
	return latest
}

// SelectLocalModelMonitorTrend 按平均可用率选择一条完整趋势，平均值相同时优先健康守护。
func SelectLocalModelMonitorTrend(upstream, healthGuard *LocalModelMonitorTrend) (LocalModelMonitorTrend, bool) {
	upstreamAverage, upstreamOK := localModelMonitorTrendAverage(upstream)
	healthGuardAverage, healthGuardOK := localModelMonitorTrendAverage(healthGuard)

	switch {
	case !upstreamOK && !healthGuardOK:
		return LocalModelMonitorTrend{}, false
	case !upstreamOK:
		return *healthGuard, true
	case !healthGuardOK:
		return *upstream, true
	case healthGuardAverage >= upstreamAverage:
		return *healthGuard, true
	default:
		return *upstream, true
	}
}

func localModelMonitorTrendAverage(trend *LocalModelMonitorTrend) (float64, bool) {
	if trend == nil {
		return 0, false
	}
	return trend.AverageAvailability()
}

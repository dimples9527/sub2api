package service

import (
	"sort"
	"time"
)

const (
	SupplierProviderGroupHealthTrendSource        = "supplier_account_health_guard"
	SupplierProviderGroupHealthTrendMonitorSource = "supplier_monitor"
)

const (
	supplierProviderGroupHealthTrendToneGreen  = "green"
	supplierProviderGroupHealthTrendToneYellow = "yellow"
	supplierProviderGroupHealthTrendToneRed    = "red"
	supplierProviderGroupHealthTrendToneGray   = "gray"
)

type SupplierProviderGroupHealthTrendParams struct {
	GroupIDs                 []int64
	Period                   time.Duration
	BucketCount              int
	Now                      time.Time
	AllHistory               bool
	PreferRawMonitorTimeline bool
}

type SupplierProviderGroupHealthSample struct {
	GroupID    int64
	AccountID  int64
	Source     string
	Status     string
	Latency    int64
	FinishedAt time.Time
}

type SupplierProviderGroupHealthTrendPoint struct {
	Time               time.Time `json:"time"`
	Availability       float64   `json:"availability"`
	Latency            int64     `json:"latency"`
	TestedAccountCount int       `json:"tested_account_count"`
	Tone               string    `json:"tone"`
}

type SupplierProviderGroupHealthTrend struct {
	GroupID      int64                                   `json:"group_id"`
	Source       string                                  `json:"source"`
	Availability float64                                 `json:"availability"`
	Latency      int64                                   `json:"latency"`
	Time         time.Time                               `json:"time"`
	Trend        []SupplierProviderGroupHealthTrendPoint `json:"trend"`
}

type supplierProviderGroupHealthSampleKey struct {
	groupID     int64
	accountID   int64
	bucketIndex int
}

type supplierProviderGroupHealthRawSampleKey struct {
	groupID   int64
	accountID int64
	minute    time.Time
}

func BuildSupplierProviderGroupHealthTrends(samples []SupplierProviderGroupHealthSample, params SupplierProviderGroupHealthTrendParams) map[int64]SupplierProviderGroupHealthTrend {
	params = normalizeSupplierProviderGroupHealthTrendParams(params)
	requestedGroups := supplierProviderGroupHealthRequestedGroups(params.GroupIDs)
	if len(requestedGroups) == 0 {
		return map[int64]SupplierProviderGroupHealthTrend{}
	}

	fallbackSamples := make([]SupplierProviderGroupHealthSample, 0, len(samples))
	monitorSamples := make([]SupplierProviderGroupHealthSample, 0)
	for _, sample := range samples {
		if sample.Source == SupplierProviderGroupHealthTrendMonitorSource {
			monitorSamples = append(monitorSamples, sample)
			continue
		}
		fallbackSamples = append(fallbackSamples, sample)
	}

	trends := buildSupplierProviderGroupHealthBucketTrends(fallbackSamples, params, requestedGroups)
	if !params.PreferRawMonitorTimeline {
		return trends
	}
	monitorGroups := make(map[int64]struct{}, len(monitorSamples))
	for _, sample := range monitorSamples {
		monitorGroups[sample.GroupID] = struct{}{}
	}
	rawSamples := make([]SupplierProviderGroupHealthSample, 0, len(fallbackSamples)+len(monitorSamples))
	for _, sample := range fallbackSamples {
		if sample.Status == SupplierAccountHealthGuardStatusHealthy || sample.Status == SupplierAccountHealthGuardStatusSlow {
			rawSamples = append(rawSamples, sample)
		}
	}
	rawSamples = append(rawSamples, monitorSamples...)
	for groupID, trend := range buildSupplierProviderGroupHealthRawMonitorTrends(rawSamples, params, requestedGroups) {
		if _, ok := monitorGroups[groupID]; !ok {
			continue
		}
		trends[groupID] = trend
	}
	return trends
}

func supplierProviderGroupHealthRequestedGroups(groupIDs []int64) map[int64]struct{} {
	requestedGroups := make(map[int64]struct{}, len(groupIDs))
	for _, groupID := range groupIDs {
		if groupID > 0 {
			requestedGroups[groupID] = struct{}{}
		}
	}
	return requestedGroups
}

func buildSupplierProviderGroupHealthBucketTrends(samples []SupplierProviderGroupHealthSample, params SupplierProviderGroupHealthTrendParams, requestedGroups map[int64]struct{}) map[int64]SupplierProviderGroupHealthTrend {
	windowStart := params.Now.Add(-params.Period)
	if params.AllHistory {
		var earliestSample time.Time
		for _, sample := range samples {
			if _, ok := requestedGroups[sample.GroupID]; !ok || !supplierProviderGroupHealthSampleUsable(sample) {
				continue
			}
			finishedAt := sample.FinishedAt.UTC()
			if finishedAt.After(params.Now) {
				continue
			}
			if earliestSample.IsZero() || finishedAt.Before(earliestSample) {
				earliestSample = finishedAt
			}
		}
		if !earliestSample.IsZero() {
			windowStart = earliestSample
		}
		params.Period = params.Now.Sub(windowStart)
		if params.Period <= 0 {
			params.Period = time.Minute
			windowStart = params.Now.Add(-params.Period)
		}
	}
	bucketDuration := params.Period / time.Duration(params.BucketCount)
	if bucketDuration <= 0 {
		bucketDuration = time.Nanosecond
	}
	latestSamples := make(map[supplierProviderGroupHealthSampleKey]SupplierProviderGroupHealthSample)
	for _, sample := range samples {
		if _, ok := requestedGroups[sample.GroupID]; !ok || !supplierProviderGroupHealthSampleUsable(sample) {
			continue
		}
		bucketIndex, ok := supplierProviderGroupHealthTrendBucketIndex(sample.FinishedAt, windowStart, params.Now, bucketDuration, params.BucketCount)
		if !ok {
			continue
		}
		key := supplierProviderGroupHealthSampleKey{groupID: sample.GroupID, accountID: sample.AccountID, bucketIndex: bucketIndex}
		if current, exists := latestSamples[key]; !exists || sample.FinishedAt.After(current.FinishedAt) {
			latestSamples[key] = sample
		}
	}

	bucketsByGroup := make(map[int64]map[int][]SupplierProviderGroupHealthSample)
	for key, sample := range latestSamples {
		if bucketsByGroup[key.groupID] == nil {
			bucketsByGroup[key.groupID] = make(map[int][]SupplierProviderGroupHealthSample)
		}
		bucketsByGroup[key.groupID][key.bucketIndex] = append(bucketsByGroup[key.groupID][key.bucketIndex], sample)
	}

	trends := make(map[int64]SupplierProviderGroupHealthTrend, len(bucketsByGroup))
	for groupID, buckets := range bucketsByGroup {
		trend := SupplierProviderGroupHealthTrend{
			GroupID: groupID,
			Source:  SupplierProviderGroupHealthTrendSource,
			Trend:   make([]SupplierProviderGroupHealthTrendPoint, 0, params.BucketCount),
		}
		var latestPoint *SupplierProviderGroupHealthTrendPoint
		for bucketIndex := 0; bucketIndex < params.BucketCount; bucketIndex++ {
			pointTime := windowStart.Add(time.Duration(bucketIndex+1) * bucketDuration)
			point := buildSupplierProviderGroupHealthTrendPoint(pointTime, buckets[bucketIndex])
			trend.Trend = append(trend.Trend, point)
			if point.TestedAccountCount > 0 {
				pointCopy := point
				latestPoint = &pointCopy
			}
		}
		if latestPoint == nil {
			continue
		}
		trend.Availability = latestPoint.Availability
		trend.Latency = latestPoint.Latency
		trend.Time = latestPoint.Time
		trends[groupID] = trend
	}
	return trends
}

func buildSupplierProviderGroupHealthRawMonitorTrends(samples []SupplierProviderGroupHealthSample, params SupplierProviderGroupHealthTrendParams, requestedGroups map[int64]struct{}) map[int64]SupplierProviderGroupHealthTrend {
	latestSamples := make(map[supplierProviderGroupHealthRawSampleKey]SupplierProviderGroupHealthSample)
	for _, sample := range samples {
		if _, ok := requestedGroups[sample.GroupID]; !ok || !supplierProviderGroupHealthSampleUsable(sample) {
			continue
		}
		finishedAt := sample.FinishedAt.UTC()
		if finishedAt.After(params.Now) {
			continue
		}
		key := supplierProviderGroupHealthRawSampleKey{
			groupID:   sample.GroupID,
			accountID: sample.AccountID,
			minute:    finishedAt.Truncate(time.Minute),
		}
		if current, exists := latestSamples[key]; !exists || finishedAt.After(current.FinishedAt) {
			latestSamples[key] = sample
		}
	}

	pointsByGroup := make(map[int64]map[time.Time][]SupplierProviderGroupHealthSample)
	for key, sample := range latestSamples {
		if pointsByGroup[key.groupID] == nil {
			pointsByGroup[key.groupID] = make(map[time.Time][]SupplierProviderGroupHealthSample)
		}
		pointsByGroup[key.groupID][key.minute] = append(pointsByGroup[key.groupID][key.minute], sample)
	}

	trends := make(map[int64]SupplierProviderGroupHealthTrend, len(pointsByGroup))
	for groupID, pointsByMinute := range pointsByGroup {
		minutes := make([]time.Time, 0, len(pointsByMinute))
		for minute := range pointsByMinute {
			minutes = append(minutes, minute)
		}
		sort.Slice(minutes, func(i, j int) bool { return minutes[i].Before(minutes[j]) })
		if len(minutes) > params.BucketCount {
			minutes = minutes[len(minutes)-params.BucketCount:]
		}

		trend := SupplierProviderGroupHealthTrend{
			GroupID: groupID,
			Source:  SupplierProviderGroupHealthTrendMonitorSource,
			Trend:   make([]SupplierProviderGroupHealthTrendPoint, 0, len(minutes)),
		}
		for _, minute := range minutes {
			trend.Trend = append(trend.Trend, buildSupplierProviderGroupHealthTrendPoint(minute, pointsByMinute[minute]))
		}
		if len(trend.Trend) == 0 {
			continue
		}
		latestPoint := trend.Trend[len(trend.Trend)-1]
		trend.Availability = latestPoint.Availability
		trend.Latency = latestPoint.Latency
		trend.Time = latestPoint.Time
		trends[groupID] = trend
	}
	return trends
}

func normalizeSupplierProviderGroupHealthTrendParams(params SupplierProviderGroupHealthTrendParams) SupplierProviderGroupHealthTrendParams {
	if params.Period <= 0 {
		params.Period = 90 * time.Minute
	}
	if params.BucketCount <= 0 {
		params.BucketCount = 18
	}
	if params.Now.IsZero() {
		params.Now = time.Now().UTC()
	} else {
		params.Now = params.Now.UTC()
	}
	return params
}

func supplierProviderGroupHealthSampleUsable(sample SupplierProviderGroupHealthSample) bool {
	if sample.GroupID <= 0 || sample.AccountID <= 0 || sample.FinishedAt.IsZero() {
		return false
	}
	switch sample.Status {
	case SupplierAccountHealthGuardStatusHealthy, SupplierAccountHealthGuardStatusSlow, SupplierAccountHealthGuardStatusFailed, SupplierAccountHealthGuardStatusUnavailable:
		return true
	default:
		return false
	}
}

func supplierProviderGroupHealthTrendBucketIndex(finishedAt, windowStart, now time.Time, bucketDuration time.Duration, bucketCount int) (int, bool) {
	finishedAt = finishedAt.UTC()
	if finishedAt.Before(windowStart) || finishedAt.After(now) {
		return 0, false
	}
	if finishedAt.Equal(now) {
		return bucketCount - 1, true
	}
	bucketIndex := int(finishedAt.Sub(windowStart) / bucketDuration)
	if bucketIndex < 0 || bucketIndex >= bucketCount {
		return 0, false
	}
	return bucketIndex, true
}

func buildSupplierProviderGroupHealthTrendPoint(timeValue time.Time, samples []SupplierProviderGroupHealthSample) SupplierProviderGroupHealthTrendPoint {
	if len(samples) == 0 {
		return SupplierProviderGroupHealthTrendPoint{Time: timeValue, Tone: supplierProviderGroupHealthTrendToneGray}
	}

	availableCount := 0
	latencyTotal := int64(0)
	latencyCount := 0
	for _, sample := range samples {
		if sample.Status == SupplierAccountHealthGuardStatusHealthy || sample.Status == SupplierAccountHealthGuardStatusSlow {
			availableCount++
		}
		if sample.Latency > 0 {
			latencyTotal += sample.Latency
			latencyCount++
		}
	}
	availability := float64(availableCount) * 100 / float64(len(samples))
	latency := int64(0)
	if latencyCount > 0 {
		latency = latencyTotal / int64(latencyCount)
	}
	return SupplierProviderGroupHealthTrendPoint{
		Time:               timeValue,
		Availability:       availability,
		Latency:            latency,
		TestedAccountCount: len(samples),
		Tone:               supplierProviderGroupHealthTrendTone(availability),
	}
}

func supplierProviderGroupHealthTrendTone(availability float64) string {
	switch {
	case availability <= 0:
		return supplierProviderGroupHealthTrendToneRed
	case availability >= 40:
		return supplierProviderGroupHealthTrendToneGreen
	default:
		return supplierProviderGroupHealthTrendToneYellow
	}
}

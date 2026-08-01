package service

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestSelectLocalModelMonitorTrendPrefersHigherAverageAvailability(t *testing.T) {
	upstream := LocalModelMonitorTrend{
		Trend: []LocalModelMonitorTrendPoint{
			{Availability: 70, Valid: true},
			{Availability: 80, Valid: true},
		},
	}
	healthGuard := LocalModelMonitorTrend{
		Trend: []LocalModelMonitorTrendPoint{
			{Availability: 95, Valid: true},
			{Availability: 90, Valid: true},
		},
	}

	selected, ok := SelectLocalModelMonitorTrend(&upstream, &healthGuard)

	require.True(t, ok)
	require.Equal(t, healthGuard, selected)
}

func TestSelectLocalModelMonitorTrendUsesAvailableSourceAndHealthTieBreaker(t *testing.T) {
	upstream := LocalModelMonitorTrend{
		Trend: []LocalModelMonitorTrendPoint{{Availability: 80, Valid: true}},
	}
	healthGuard := LocalModelMonitorTrend{
		Trend: []LocalModelMonitorTrendPoint{{Availability: 80, Valid: true}},
	}

	selected, ok := SelectLocalModelMonitorTrend(&upstream, &healthGuard)
	require.True(t, ok)
	require.Equal(t, healthGuard, selected)

	selected, ok = SelectLocalModelMonitorTrend(&upstream, nil)
	require.True(t, ok)
	require.Equal(t, upstream, selected)

	selected, ok = SelectLocalModelMonitorTrend(nil, &healthGuard)
	require.True(t, ok)
	require.Equal(t, healthGuard, selected)
}

func TestSelectLocalModelMonitorTrendIgnoresEmptyBuckets(t *testing.T) {
	trend := LocalModelMonitorTrend{
		Trend: []LocalModelMonitorTrendPoint{
			{Availability: 100, Valid: true},
			{Tone: "gray", Valid: false},
		},
	}
	other := LocalModelMonitorTrend{
		Trend: []LocalModelMonitorTrendPoint{{Availability: 99, Valid: true}},
	}

	selected, ok := SelectLocalModelMonitorTrend(&trend, &other)

	require.True(t, ok)
	require.Equal(t, trend, selected)
}

func TestLocalModelMonitorPointAvailabilityUsesLatestValidPoint(t *testing.T) {
	checkedAt := time.Date(2026, 8, 1, 12, 0, 0, 0, time.UTC)
	trend := LocalModelMonitorTrend{
		Trend: []LocalModelMonitorTrendPoint{
			{Time: checkedAt.Add(-time.Minute), Availability: 60, Valid: true},
			{Time: checkedAt, Availability: 100, Valid: true},
		},
	}

	availability, ok := trend.AverageAvailability()
	require.True(t, ok)
	require.Equal(t, 80.0, availability)
	require.Equal(t, checkedAt, trend.LatestTime())
}

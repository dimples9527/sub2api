package service

import "time"

// successfulAttemptTimer 记录最终成功上游请求的开始时间。
// 失败请求、重试退避与切号等待不应计入成功 usage 的首字和总耗时。
type successfulAttemptTimer struct {
	fallback time.Time
	start    time.Time
}

func newSuccessfulAttemptTimer(fallback time.Time) successfulAttemptTimer {
	return successfulAttemptTimer{fallback: fallback}
}

// Mark 只记录第一次成功 attempt 的开始时间，避免后续收口逻辑误覆盖。
func (t *successfulAttemptTimer) Mark(start time.Time) {
	if start.IsZero() || !t.start.IsZero() {
		return
	}
	t.start = start
}

func (t successfulAttemptTimer) Start() time.Time {
	if !t.start.IsZero() {
		return t.start
	}
	return t.fallback
}

func (t successfulAttemptTimer) Since() time.Duration {
	return time.Since(t.Start())
}

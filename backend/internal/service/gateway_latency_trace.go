package service

import (
	"context"
	"crypto/tls"
	"net/http/httptrace"
	"strings"
	"sync"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/logger"
)

// LatencyPhases 拆解「最终成功的那次上游 attempt」内部的耗时构成。
//
// successfulAttemptTimer 只能剔除 attempt 之前的等待（失败重试、退避、切号、串行锁），
// attempt 内部的连接池排队 / 建连 / TLS / 等上游首字此前完全不可见。当上游本身也是
// 一层网关时，它自报的耗时同样被它自己的 attempt 重基过，两侧数字无法通过继续做减法
// 对齐，只能靠拆解定位差额落在本地还是上游。
type LatencyPhases struct {
	// BuildMs 组装上游请求（读取/改写请求体、取 token、签名）。
	BuildMs *int `json:"build_ms"`
	// SlotWaitMs 等待连接池空位，已扣除建连与 TLS。account/account_proxy 隔离下
	// maxConnsPerHost 被夹到账号并发数，此项变大即为排队。
	SlotWaitMs *int `json:"slot_wait_ms"`
	// ConnectMs DNS 解析 + TCP 握手。
	ConnectMs *int `json:"connect_ms"`
	// TLSMs TLS 握手。
	TLSMs *int `json:"tls_ms"`
	// FirstByteMs 请求写完到收到首个响应字节，即上游真实处理时间。
	FirstByteMs *int `json:"first_byte_ms"`
	// ConnReused 是否复用了空闲连接。
	ConnReused *bool `json:"conn_reused"`
}

// usageLogLatencyPhaseWriter / usageLogLatencyPhaseReader 是 UsageLogRepository 的
// 可选扩展，运行时断言获取。诊断用的旁路数据不值得改动主接口——那会牵动全部测试桩。
type usageLogLatencyPhaseWriter interface {
	CreateLatencyPhases(ctx context.Context, requestID string, apiKeyID int64, phases *LatencyPhases) error
}

type usageLogLatencyPhaseReader interface {
	GetLatencyPhases(ctx context.Context, requestID string, apiKeyID int64) (*LatencyPhases, error)
}

// writeLatencyPhasesBestEffort 把耗时分解写入侧边表。失败只记日志：这是诊断数据，
// 绝不能影响计费或请求结果。
func writeLatencyPhasesBestEffort(ctx context.Context, repo UsageLogRepository, requestID string, apiKeyID int64, phases *LatencyPhases, logKey string) {
	if repo == nil || phases == nil || strings.TrimSpace(requestID) == "" {
		return
	}
	writer, ok := repo.(usageLogLatencyPhaseWriter)
	if !ok {
		return
	}
	writeCtx, cancel := detachedBillingContext(ctx)
	defer cancel()
	if err := writer.CreateLatencyPhases(writeCtx, requestID, apiKeyID, phases); err != nil {
		logger.LegacyPrintf(logKey, "Create usage log latency phases failed: %v", err)
	}
}

// GetLatencyPhases 读取某条用量记录的耗时分解，缺失时返回 nil。
// 仅管理员接口调用：分解暴露了本地连接池与上游的内部时序。
func (s *UsageService) GetLatencyPhases(ctx context.Context, requestID string, apiKeyID int64) (*LatencyPhases, error) {
	if s == nil || s.usageRepo == nil {
		return nil, nil
	}
	reader, ok := s.usageRepo.(usageLogLatencyPhaseReader)
	if !ok {
		return nil, nil
	}
	return reader.GetLatencyPhases(ctx, requestID, apiKeyID)
}

// latencyTrace 收集单次 attempt 的 httptrace 时间点。
// httptrace 回调运行在 transport goroutine 上，所有字段访问必须持锁。
type latencyTrace struct {
	mu sync.Mutex

	attemptStart time.Time
	built        time.Time
	getConn      time.Time
	gotConn      time.Time
	connectStart time.Time
	connectDone  time.Time
	tlsStart     time.Time
	tlsDone      time.Time
	wroteRequest time.Time
	firstByte    time.Time
	connReused   bool
	gotConnSeen  bool
}

func newLatencyTrace(attemptStart time.Time) *latencyTrace {
	if attemptStart.IsZero() {
		attemptStart = time.Now()
	}
	return &latencyTrace{attemptStart: attemptStart}
}

// attach 把采集钩子挂到 context 上。detachStreamUpstreamContext 使用
// context.WithoutCancel，保留 values，因此 trace 能穿透 detach 存活。
func (t *latencyTrace) attach(ctx context.Context) context.Context {
	if t == nil {
		return ctx
	}
	if ctx == nil {
		ctx = context.Background()
	}
	return httptrace.WithClientTrace(ctx, &httptrace.ClientTrace{
		GetConn:              func(string) { t.stamp(&t.getConn) },
		ConnectStart:         func(string, string) { t.stamp(&t.connectStart) },
		ConnectDone:          func(string, string, error) { t.stamp(&t.connectDone) },
		TLSHandshakeStart:    func() { t.stamp(&t.tlsStart) },
		TLSHandshakeDone:     func(tls.ConnectionState, error) { t.stamp(&t.tlsDone) },
		WroteRequest:         func(httptrace.WroteRequestInfo) { t.stamp(&t.wroteRequest) },
		GotFirstResponseByte: func() { t.stamp(&t.firstByte) },
		GotConn: func(info httptrace.GotConnInfo) {
			t.mu.Lock()
			defer t.mu.Unlock()
			if t.gotConnSeen {
				return
			}
			t.gotConnSeen = true
			t.gotConn = time.Now()
			t.connReused = info.Reused
		},
	})
}

// markBuilt 记录上游请求组装完成的时刻。
func (t *latencyTrace) markBuilt() {
	if t == nil {
		return
	}
	t.stamp(&t.built)
}

// stamp 只记录首次触发，避免 happy-eyeballs 双栈拨号或 transport 内部重试覆盖。
func (t *latencyTrace) stamp(dst *time.Time) {
	t.mu.Lock()
	defer t.mu.Unlock()
	if dst.IsZero() {
		*dst = time.Now()
	}
}

func (t *latencyTrace) phases() *LatencyPhases {
	if t == nil {
		return nil
	}
	t.mu.Lock()
	defer t.mu.Unlock()

	if !t.gotConnSeen && t.firstByte.IsZero() {
		return nil
	}

	connect := latencySpan(t.connectStart, t.connectDone)
	handshake := latencySpan(t.tlsStart, t.tlsDone)
	// GotConn 在建连与 TLS 之后才触发，扣掉两者才是纯排队等待。
	slotWait := latencySpan(t.getConn, t.gotConn)
	if slotWait != nil {
		remaining := *slotWait - latencyDur(connect) - latencyDur(handshake)
		if remaining < 0 {
			remaining = 0
		}
		slotWait = &remaining
	}

	reused := t.connReused
	return &LatencyPhases{
		BuildMs:     latencyMillis(latencySpan(t.attemptStart, t.built)),
		SlotWaitMs:  latencyMillis(slotWait),
		ConnectMs:   latencyMillis(connect),
		TLSMs:       latencyMillis(handshake),
		FirstByteMs: latencyMillis(latencySpan(t.wroteRequest, t.firstByte)),
		ConnReused:  &reused,
	}
}

func latencySpan(start, end time.Time) *time.Duration {
	if start.IsZero() || end.IsZero() || end.Before(start) {
		return nil
	}
	d := end.Sub(start)
	return &d
}

func latencyDur(d *time.Duration) time.Duration {
	if d == nil {
		return 0
	}
	return *d
}

func latencyMillis(d *time.Duration) *int {
	if d == nil {
		return nil
	}
	ms := int(d.Milliseconds())
	return &ms
}

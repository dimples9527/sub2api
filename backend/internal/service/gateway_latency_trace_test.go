package service

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestLatencyTracePhasesNilWhenNothingObserved(t *testing.T) {
	require.Nil(t, (*latencyTrace)(nil).phases())
	require.Nil(t, newLatencyTrace(time.Now()).phases())
}

func TestLatencyTracePhasesExcludesConnectAndTLSFromSlotWait(t *testing.T) {
	base := time.Now()
	trace := newLatencyTrace(base)
	trace.built = base.Add(10 * time.Millisecond)
	trace.getConn = base.Add(10 * time.Millisecond)
	// 排队 200ms，建连 30ms，TLS 70ms → GotConn 落在 310ms。
	trace.connectStart = base.Add(210 * time.Millisecond)
	trace.connectDone = base.Add(240 * time.Millisecond)
	trace.tlsStart = base.Add(240 * time.Millisecond)
	trace.tlsDone = base.Add(310 * time.Millisecond)
	trace.gotConn = base.Add(310 * time.Millisecond)
	trace.gotConnSeen = true
	trace.wroteRequest = base.Add(320 * time.Millisecond)
	trace.firstByte = base.Add(6320 * time.Millisecond)

	phases := trace.phases()
	require.NotNil(t, phases)
	require.Equal(t, 10, *phases.BuildMs)
	require.Equal(t, 200, *phases.SlotWaitMs)
	require.Equal(t, 30, *phases.ConnectMs)
	require.Equal(t, 70, *phases.TLSMs)
	require.Equal(t, 6000, *phases.FirstByteMs)
	require.False(t, *phases.ConnReused)
}

func TestLatencyTracePhasesReusedConnectionKeepsFullSlotWait(t *testing.T) {
	base := time.Now()
	trace := newLatencyTrace(base)
	trace.getConn = base
	trace.gotConn = base.Add(90 * time.Second)
	trace.gotConnSeen = true
	trace.connReused = true
	trace.wroteRequest = base.Add(90 * time.Second)
	trace.firstByte = base.Add(96 * time.Second)

	phases := trace.phases()
	require.NotNil(t, phases)
	// 复用连接时没有建连/TLS，90s 全部是等空位——正是并发被夹到账号并发数时的表现。
	require.Equal(t, 90_000, *phases.SlotWaitMs)
	require.Nil(t, phases.ConnectMs)
	require.Nil(t, phases.TLSMs)
	require.Equal(t, 6_000, *phases.FirstByteMs)
	require.True(t, *phases.ConnReused)
	require.Nil(t, phases.BuildMs)
}

func TestLatencyTracePhasesClampsNegativeSlotWait(t *testing.T) {
	base := time.Now()
	trace := newLatencyTrace(base)
	trace.getConn = base
	trace.gotConn = base.Add(50 * time.Millisecond)
	trace.gotConnSeen = true
	// 双栈拨号下 ConnectDone 可能晚于 GotConn，扣减后不能变负数。
	trace.connectStart = base
	trace.connectDone = base.Add(80 * time.Millisecond)

	phases := trace.phases()
	require.NotNil(t, phases)
	require.Equal(t, 0, *phases.SlotWaitMs)
}

// 钩子挂在 context 上，而请求实际发送前 context 会经 WithoutCancel 转手，
// 这里验证采集确实穿透到 transport。
func TestLatencyTraceAttachCapturesRealRequest(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		time.Sleep(30 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	trace := newLatencyTrace(time.Now())
	ctx := trace.attach(context.Background())
	ctx, _ = detachStreamUpstreamContext(ctx, true)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, server.URL, nil)
	require.NoError(t, err)
	trace.markBuilt()

	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	require.NoError(t, resp.Body.Close())

	phases := trace.phases()
	require.NotNil(t, phases)
	require.NotNil(t, phases.SlotWaitMs)
	require.NotNil(t, phases.ConnectMs)
	require.NotNil(t, phases.BuildMs)
	require.NotNil(t, phases.ConnReused)
	require.NotNil(t, phases.FirstByteMs)
	require.GreaterOrEqual(t, *phases.FirstByteMs, 25)
}

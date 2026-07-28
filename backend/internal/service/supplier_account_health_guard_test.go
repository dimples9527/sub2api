package service

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestNormalizeSupplierAccountHealthGuardConfigUsesDefaultsAndCleansValues(t *testing.T) {
	config := normalizeSupplierAccountHealthGuardConfig(SupplierAccountHealthGuardConfig{
		AccountIDs: []int64{3, 0, -1, 3, 2},
		AccountModels: map[int64]string{
			0: "invalid",
			2: "  gpt-account  ",
			3: " ",
		},
		PlatformModels: map[string]string{
			" OpenAI ": "  gpt-platform ",
			"":         "invalid",
		},
		PlatformLatencyMs: map[string]int64{
			" OpenAI ": 1200,
			"claude":   0,
		},
	})

	require.Equal(t, 200, config.MaxAccountsPerRun)
	require.Equal(t, 3, config.Concurrency)
	require.Equal(t, 90, config.TimeoutPerAccountSeconds)
	require.Equal(t, 3, config.FailureThreshold)
	require.Equal(t, 3, config.SlowThreshold)
	require.Equal(t, 2, config.RecoveryThreshold)
	require.Equal(t, int64(15000), config.HealthyLatencyMs)
	require.Equal(t, []int64{2, 3}, config.AccountIDs)
	require.Equal(t, map[int64]string{2: "gpt-account"}, config.AccountModels)
	require.Equal(t, map[string]string{"openai": "gpt-platform"}, config.PlatformModels)
	require.Equal(t, map[string]int64{"openai": 1200}, config.PlatformLatencyMs)
}

func TestSupplierAccountHealthGuardEvaluateResult(t *testing.T) {
	tests := []struct {
		name       string
		contextErr error
		runErr     error
		result     *ScheduledTestResult
		latencyMs  int64
		limitMs    int64
		wantStatus string
	}{
		{name: "健康", result: &ScheduledTestResult{Status: "success"}, latencyMs: 100, limitMs: 1000, wantStatus: SupplierAccountHealthGuardStatusHealthy},
		{name: "慢响应", result: &ScheduledTestResult{Status: "success"}, latencyMs: 1501, limitMs: 1500, wantStatus: SupplierAccountHealthGuardStatusSlow},
		{name: "测试失败", result: &ScheduledTestResult{Status: "failed", ErrorMessage: "上游错误"}, wantStatus: SupplierAccountHealthGuardStatusFailed},
		{name: "运行错误", runErr: errors.New("连接失败"), wantStatus: SupplierAccountHealthGuardStatusFailed},
		{name: "账号超时", contextErr: context.DeadlineExceeded, wantStatus: SupplierAccountHealthGuardStatusFailed},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			status, _ := supplierAccountHealthGuardEvaluateResult(tt.contextErr, tt.runErr, tt.result, tt.latencyMs, tt.limitMs)
			require.Equal(t, tt.wantStatus, status)
		})
	}
}

func TestSupplierAccountHealthGuardNextSchedulingState(t *testing.T) {
	config := SupplierAccountHealthGuardConfig{FailureThreshold: 3, SlowThreshold: 2, RecoveryThreshold: 2}
	tests := []struct {
		name            string
		item            SupplierAccountHealthGuardRunItem
		wantSchedulable bool
		wantAction      string
	}{
		{name: "连续失败后暂停", item: SupplierAccountHealthGuardRunItem{Status: SupplierAccountHealthGuardStatusFailed, SchedulableBefore: true, ConsecutiveFailed: 3}, wantSchedulable: false, wantAction: SupplierAccountHealthGuardActionDisabled},
		{name: "连续慢响应后暂停", item: SupplierAccountHealthGuardRunItem{Status: SupplierAccountHealthGuardStatusSlow, SchedulableBefore: true, ConsecutiveSlow: 2}, wantSchedulable: false, wantAction: SupplierAccountHealthGuardActionDisabled},
		{name: "连续健康后恢复", item: SupplierAccountHealthGuardRunItem{Status: SupplierAccountHealthGuardStatusHealthy, SchedulableBefore: false, ConsecutiveHealthy: 2}, wantSchedulable: true, wantAction: SupplierAccountHealthGuardActionRecovered},
		{name: "未达到阈值保持状态", item: SupplierAccountHealthGuardRunItem{Status: SupplierAccountHealthGuardStatusFailed, SchedulableBefore: true, ConsecutiveFailed: 2}, wantSchedulable: true, wantAction: SupplierAccountHealthGuardActionNone},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			schedulable, action, _ := supplierAccountHealthGuardNextSchedulingState(config, tt.item)
			require.Equal(t, tt.wantSchedulable, schedulable)
			require.Equal(t, tt.wantAction, action)
		})
	}
}

// 以下桩对象只暴露健康守护运行所需的最小依赖。
type supplierAccountHealthGuardRepoStub struct {
	candidates []SupplierAccountHealthGuardCandidate
	err        error
}

func (s *supplierAccountHealthGuardRepoStub) ListAccountHealthGuardCandidates(context.Context) ([]SupplierAccountHealthGuardCandidate, error) {
	return append([]SupplierAccountHealthGuardCandidate(nil), s.candidates...), s.err
}

type supplierAccountHealthGuardAccountStoreStub struct {
	mu           sync.Mutex
	setCalls     []supplierAccountHealthGuardSetCall
	extraUpdates map[int64]map[string]any
	setErr       error
	updateErr    error
}

type supplierAccountHealthGuardSetCall struct {
	accountID   int64
	schedulable bool
}

func (s *supplierAccountHealthGuardAccountStoreStub) UpdateExtra(_ context.Context, accountID int64, updates map[string]any) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.updateErr != nil {
		return s.updateErr
	}
	if s.extraUpdates == nil {
		s.extraUpdates = map[int64]map[string]any{}
	}
	copied := make(map[string]any, len(updates))
	for key, value := range updates {
		copied[key] = value
	}
	s.extraUpdates[accountID] = copied
	return nil
}

func (s *supplierAccountHealthGuardAccountStoreStub) SetSchedulable(_ context.Context, accountID int64, schedulable bool) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.setCalls = append(s.setCalls, supplierAccountHealthGuardSetCall{accountID: accountID, schedulable: schedulable})
	return s.setErr
}

type supplierAccountHealthGuardTesterStub struct {
	mu      sync.Mutex
	results map[int64]*ScheduledTestResult
	errs    map[int64]error
	calls   []supplierAccountHealthGuardTestCall
	fn      func(context.Context, int64, string) (*ScheduledTestResult, error)
}

type supplierAccountHealthGuardTestCall struct {
	accountID int64
	modelID   string
}

func (s *supplierAccountHealthGuardTesterStub) runTestBackground(ctx context.Context, accountID int64, modelID string) (*ScheduledTestResult, error) {
	s.mu.Lock()
	s.calls = append(s.calls, supplierAccountHealthGuardTestCall{accountID: accountID, modelID: modelID})
	s.mu.Unlock()
	if s.fn != nil {
		return s.fn(ctx, accountID, modelID)
	}
	return s.results[accountID], s.errs[accountID]
}

func TestSupplierAccountHealthGuardRunRejectsEmptyAccountWhitelist(t *testing.T) {
	tester := &supplierAccountHealthGuardTesterStub{results: map[int64]*ScheduledTestResult{}, errs: map[int64]error{}}
	guard := NewSupplierAccountHealthGuardService(
		&supplierAccountHealthGuardRepoStub{},
		&supplierAccountHealthGuardAccountStoreStub{},
		tester,
	)

	_, err := guard.Run(context.Background(), SupplierAccountHealthGuardConfig{}, time.Now())

	require.EqualError(t, err, "请至少选择一个需要检查的账号")
	require.Empty(t, tester.calls)
}

func TestSupplierAccountHealthGuardRunOnlyChecksSelectedAccounts(t *testing.T) {
	repo := &supplierAccountHealthGuardRepoStub{candidates: []SupplierAccountHealthGuardCandidate{
		{Source: SupplierAccountHealthGuardSource{ProviderAccountID: 1, UpstreamAccountName: "未匹配"}, MatchStatus: SupplierAccountHealthGuardMatchUnmatched},
		{Source: SupplierAccountHealthGuardSource{ProviderAccountID: 2, UpstreamAccountName: "冲突"}, MatchStatus: SupplierAccountHealthGuardMatchConflict, MatchCount: 2},
		newSupplierAccountHealthGuardCandidate(40, "未选择账号", "openai", true, SupplierAccountHealthGuardSource{ProviderAccountID: 4}),
		newSupplierAccountHealthGuardCandidate(50, "已选择账号", "claude", true, SupplierAccountHealthGuardSource{ProviderAccountID: 5}),
	}}
	tester := &supplierAccountHealthGuardTesterStub{
		results: map[int64]*ScheduledTestResult{50: {Status: "success", LatencyMs: 20}},
		errs:    map[int64]error{},
	}
	guard := NewSupplierAccountHealthGuardService(repo, &supplierAccountHealthGuardAccountStoreStub{}, tester)

	result, err := guard.Run(context.Background(), SupplierAccountHealthGuardConfig{
		AccountIDs:     []int64{50},
		PlatformModels: map[string]string{"claude": "claude-sonnet"},
	}, time.Date(2026, 7, 24, 10, 0, 0, 0, time.UTC))

	require.NoError(t, err)
	require.Equal(t, []supplierAccountHealthGuardTestCall{{accountID: 50, modelID: "claude-sonnet"}}, tester.calls)
	require.Equal(t, 1, result.TotalAccounts)
	require.Equal(t, 1, result.SelectedCount)
	require.Equal(t, 1, result.CheckedCount)
	require.Zero(t, result.SkippedCount)
	require.Len(t, result.Items, 1)
}

func TestSupplierAccountHealthGuardRunRejectsMissingModelsBeforeTesting(t *testing.T) {
	repo := &supplierAccountHealthGuardRepoStub{candidates: []SupplierAccountHealthGuardCandidate{
		newSupplierAccountHealthGuardCandidate(51, "账号甲", "openai", true, SupplierAccountHealthGuardSource{ProviderAccountID: 51}),
		newSupplierAccountHealthGuardCandidate(52, "账号乙", "claude", true, SupplierAccountHealthGuardSource{ProviderAccountID: 52}),
	}}
	tester := &supplierAccountHealthGuardTesterStub{results: map[int64]*ScheduledTestResult{}, errs: map[int64]error{}}
	guard := NewSupplierAccountHealthGuardService(repo, &supplierAccountHealthGuardAccountStoreStub{}, tester)

	_, err := guard.Run(context.Background(), SupplierAccountHealthGuardConfig{
		AccountIDs:     []int64{51, 52},
		PlatformModels: map[string]string{"openai": "gpt-4o-mini"},
	}, time.Now())

	require.EqualError(t, err, "以下账号尚未配置测试模型：账号乙")
	require.Empty(t, tester.calls)
}

func TestSupplierAccountHealthGuardRunRecordsUnavailableSelectedAccountsAndContinues(t *testing.T) {
	disabled := newSupplierAccountHealthGuardCandidate(61, "已停用账号", "openai", true, SupplierAccountHealthGuardSource{ProviderAccountID: 61})
	disabled.LocalAccount.Status = StatusDisabled
	repo := &supplierAccountHealthGuardRepoStub{candidates: []SupplierAccountHealthGuardCandidate{
		disabled,
		newSupplierAccountHealthGuardCandidate(62, "可用账号", "openai", true, SupplierAccountHealthGuardSource{ProviderAccountID: 62}),
	}}
	tester := &supplierAccountHealthGuardTesterStub{
		results: map[int64]*ScheduledTestResult{62: {Status: "success", LatencyMs: 20}},
		errs:    map[int64]error{},
	}
	guard := NewSupplierAccountHealthGuardService(repo, &supplierAccountHealthGuardAccountStoreStub{}, tester)

	result, err := guard.Run(context.Background(), SupplierAccountHealthGuardConfig{
		AccountIDs:     []int64{61, 62, 63},
		PlatformModels: map[string]string{"openai": "gpt-4o-mini"},
	}, time.Now())

	require.NoError(t, err)
	require.Equal(t, []supplierAccountHealthGuardTestCall{{accountID: 62, modelID: "gpt-4o-mini"}}, tester.calls)
	require.Equal(t, 3, result.TotalAccounts)
	require.Equal(t, 1, result.CheckedCount)
	require.Equal(t, 2, result.UnavailableCount)
	require.Len(t, result.Items, 3)
	require.Equal(t, SupplierAccountHealthGuardStatusUnavailable, result.Items[0].Status)
	require.Equal(t, SupplierAccountHealthGuardStatusHealthy, result.Items[1].Status)
	require.Equal(t, SupplierAccountHealthGuardStatusUnavailable, result.Items[2].Status)
}

func TestSupplierAccountHealthGuardRunDeduplicatesLocalAccountSourcesAndRotatesCursor(t *testing.T) {
	repo := &supplierAccountHealthGuardRepoStub{candidates: []SupplierAccountHealthGuardCandidate{
		newSupplierAccountHealthGuardCandidate(10, "账号十", "openai", true, SupplierAccountHealthGuardSource{ProviderID: 1, ProviderName: "供应商甲", ProviderAccountID: 11}),
		newSupplierAccountHealthGuardCandidate(10, "账号十", "openai", true, SupplierAccountHealthGuardSource{ProviderID: 2, ProviderName: "供应商乙", ProviderAccountID: 12}),
		newSupplierAccountHealthGuardCandidate(20, "账号二十", "openai", true, SupplierAccountHealthGuardSource{ProviderID: 3, ProviderName: "供应商丙", ProviderAccountID: 13}),
		newSupplierAccountHealthGuardCandidate(30, "账号三十", "openai", true, SupplierAccountHealthGuardSource{ProviderID: 4, ProviderName: "供应商丁", ProviderAccountID: 14}),
	}}
	tester := &supplierAccountHealthGuardTesterStub{results: map[int64]*ScheduledTestResult{
		10: {Status: "success", LatencyMs: 10},
		20: {Status: "success", LatencyMs: 20},
		30: {Status: "success", LatencyMs: 30},
	}, errs: map[int64]error{}}
	guard := NewSupplierAccountHealthGuardService(repo, &supplierAccountHealthGuardAccountStoreStub{}, tester)
	config := SupplierAccountHealthGuardConfig{AccountIDs: []int64{10, 20, 30}, MaxAccountsPerRun: 2, Concurrency: 1, PlatformModels: map[string]string{"openai": "gpt-4o-mini"}}

	first, err := guard.Run(context.Background(), config, time.Now())
	require.NoError(t, err)
	require.Equal(t, []supplierAccountHealthGuardTestCall{{accountID: 10, modelID: "gpt-4o-mini"}, {accountID: 20, modelID: "gpt-4o-mini"}}, tester.calls)
	require.Len(t, first.Items[0].Sources, 2)
	require.Equal(t, int64(20), first.CursorAccountID)
	require.Equal(t, 1, first.PendingCount)

	tester.calls = nil
	config.CursorAccountID = first.CursorAccountID
	second, err := guard.Run(context.Background(), config, time.Now())
	require.NoError(t, err)
	require.Equal(t, []supplierAccountHealthGuardTestCall{{accountID: 30, modelID: "gpt-4o-mini"}, {accountID: 10, modelID: "gpt-4o-mini"}}, tester.calls)
	require.Equal(t, int64(10), second.CursorAccountID)
}

func TestSupplierAccountHealthGuardRunDisablesAfterFailureThresholdAndWritesSupplierKeys(t *testing.T) {
	candidate := newSupplierAccountHealthGuardCandidate(21, "故障账号", "openai", true, SupplierAccountHealthGuardSource{ProviderAccountID: 11})
	candidate.LocalAccount.Extra = map[string]any{"supplier_health_guard_failure_count": 2, "upstream_health_guard_failure_count": 99}
	store := &supplierAccountHealthGuardAccountStoreStub{}
	tester := &supplierAccountHealthGuardTesterStub{results: map[int64]*ScheduledTestResult{21: {Status: "failed", ErrorMessage: "401"}}, errs: map[int64]error{}}
	guard := NewSupplierAccountHealthGuardService(&supplierAccountHealthGuardRepoStub{candidates: []SupplierAccountHealthGuardCandidate{candidate}}, store, tester)

	result, err := guard.Run(context.Background(), SupplierAccountHealthGuardConfig{
		AccountIDs:       []int64{21},
		FailureThreshold: 3, SlowThreshold: 3, RecoveryThreshold: 2,
		PlatformModels: map[string]string{"openai": "gpt-4o-mini"},
	}, time.Now())

	require.NoError(t, err)
	require.Equal(t, 1, result.FailedCount)
	require.Equal(t, 1, result.DisabledCount)
	require.Equal(t, []supplierAccountHealthGuardSetCall{{accountID: 21, schedulable: false}}, store.setCalls)
	require.Equal(t, 3, result.Items[0].ConsecutiveFailed)
	for key := range store.extraUpdates[21] {
		require.Contains(t, key, "supplier_health_guard_")
		require.NotContains(t, key, "upstream_health_guard_")
	}
}

func TestSupplierAccountHealthGuardRunRecoversAndUsesOverrides(t *testing.T) {
	candidate := newSupplierAccountHealthGuardCandidate(22, "恢复账号", "openai", false, SupplierAccountHealthGuardSource{ProviderAccountID: 12})
	candidate.LocalAccount.Extra = map[string]any{"supplier_health_guard_healthy_count": 1}
	store := &supplierAccountHealthGuardAccountStoreStub{}
	tester := &supplierAccountHealthGuardTesterStub{results: map[int64]*ScheduledTestResult{22: {Status: "success", LatencyMs: 900}}, errs: map[int64]error{}}
	guard := NewSupplierAccountHealthGuardService(&supplierAccountHealthGuardRepoStub{candidates: []SupplierAccountHealthGuardCandidate{candidate}}, store, tester)

	result, err := guard.Run(context.Background(), SupplierAccountHealthGuardConfig{
		AccountIDs:        []int64{22},
		RecoveryThreshold: 2,
		HealthyLatencyMs:  500,
		AccountModels:     map[int64]string{22: "account-model"},
		PlatformModels:    map[string]string{"openai": "platform-model"},
		PlatformLatencyMs: map[string]int64{"openai": 1000},
	}, time.Now())

	require.NoError(t, err)
	require.Equal(t, 1, result.HealthyCount)
	require.Equal(t, 1, result.RecoveredCount)
	require.Equal(t, "account-model", tester.calls[0].modelID)
	require.Equal(t, int64(1000), result.Items[0].LatencyLimitMs)
	require.Equal(t, []supplierAccountHealthGuardSetCall{{accountID: 22, schedulable: true}}, store.setCalls)
}

func TestSupplierAccountHealthGuardRunMarksSchedulableWriteFailureAsFailed(t *testing.T) {
	candidate := newSupplierAccountHealthGuardCandidate(41, "schedulable-write-failure", "openai", true, SupplierAccountHealthGuardSource{ProviderAccountID: 41})
	candidate.LocalAccount.Extra = map[string]any{supplierHealthGuardSlowCountExtraKey: 1}
	store := &supplierAccountHealthGuardAccountStoreStub{setErr: errors.New("schedulable write failed")}
	tester := &supplierAccountHealthGuardTesterStub{
		results: map[int64]*ScheduledTestResult{41: {Status: "success", LatencyMs: 1501}},
		errs:    map[int64]error{},
	}
	guard := NewSupplierAccountHealthGuardService(&supplierAccountHealthGuardRepoStub{candidates: []SupplierAccountHealthGuardCandidate{candidate}}, store, tester)

	result, err := guard.Run(context.Background(), SupplierAccountHealthGuardConfig{
		AccountIDs:       []int64{41},
		SlowThreshold:    2,
		HealthyLatencyMs: 1500,
		PlatformModels:   map[string]string{"openai": "gpt-4o-mini"},
	}, time.Now())

	require.NoError(t, err)
	require.Equal(t, 1, result.FailedCount)
	require.Zero(t, result.SlowCount)
	require.Equal(t, SupplierAccountHealthGuardStatusFailed, result.Items[0].Status)
	require.Equal(t, SupplierAccountHealthGuardActionNone, result.Items[0].Action)
	require.True(t, result.Items[0].SchedulableAfter)
	require.Contains(t, result.Items[0].ErrorMessage, "schedulable write failed")
}

func TestSupplierAccountHealthGuardRunMarksExtraWriteFailureAsFailed(t *testing.T) {
	candidate := newSupplierAccountHealthGuardCandidate(42, "extra-write-failure", "openai", true, SupplierAccountHealthGuardSource{ProviderAccountID: 42})
	store := &supplierAccountHealthGuardAccountStoreStub{updateErr: errors.New("extra write failed")}
	tester := &supplierAccountHealthGuardTesterStub{
		results: map[int64]*ScheduledTestResult{42: {Status: "success", LatencyMs: 20}},
		errs:    map[int64]error{},
	}
	guard := NewSupplierAccountHealthGuardService(&supplierAccountHealthGuardRepoStub{candidates: []SupplierAccountHealthGuardCandidate{candidate}}, store, tester)

	result, err := guard.Run(context.Background(), SupplierAccountHealthGuardConfig{
		AccountIDs:     []int64{42},
		PlatformModels: map[string]string{"openai": "gpt-4o-mini"},
	}, time.Now())

	require.NoError(t, err)
	require.Equal(t, 1, result.FailedCount)
	require.Zero(t, result.HealthyCount)
	require.Equal(t, SupplierAccountHealthGuardStatusFailed, result.Items[0].Status)
	require.Contains(t, result.Items[0].ErrorMessage, "extra write failed")
}

type supplierAccountHealthGuardConcurrencyTester struct {
	active    atomic.Int32
	maxActive atomic.Int32
	started   chan int64
	release   <-chan struct{}
}

func (s *supplierAccountHealthGuardConcurrencyTester) runTestBackground(ctx context.Context, accountID int64, _ string) (*ScheduledTestResult, error) {
	active := s.active.Add(1)
	defer s.active.Add(-1)
	for {
		currentMax := s.maxActive.Load()
		if active <= currentMax || s.maxActive.CompareAndSwap(currentMax, active) {
			break
		}
	}
	s.started <- accountID
	select {
	case <-s.release:
		return &ScheduledTestResult{Status: "success", LatencyMs: 10}, nil
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

func TestSupplierAccountHealthGuardRunRespectsConcurrencyLimit(t *testing.T) {
	candidates := make([]SupplierAccountHealthGuardCandidate, 0, 4)
	for accountID := int64(1); accountID <= 4; accountID++ {
		candidates = append(candidates, newSupplierAccountHealthGuardCandidate(
			accountID,
			fmt.Sprintf("account-%d", accountID),
			"openai",
			true,
			SupplierAccountHealthGuardSource{ProviderAccountID: accountID},
		))
	}
	release := make(chan struct{})
	tester := &supplierAccountHealthGuardConcurrencyTester{
		started: make(chan int64, len(candidates)),
		release: release,
	}
	guard := NewSupplierAccountHealthGuardService(
		&supplierAccountHealthGuardRepoStub{candidates: candidates},
		&supplierAccountHealthGuardAccountStoreStub{},
		tester,
	)
	type runOutcome struct {
		result SupplierAccountHealthGuardResult
		err    error
	}
	outcome := make(chan runOutcome, 1)
	go func() {
		result, err := guard.Run(context.Background(), SupplierAccountHealthGuardConfig{
			AccountIDs:     []int64{1, 2, 3, 4},
			Concurrency:    2,
			PlatformModels: map[string]string{"openai": "gpt-4o-mini"},
		}, time.Now())
		outcome <- runOutcome{result: result, err: err}
	}()

	for index := 0; index < 2; index++ {
		select {
		case <-tester.started:
		case <-time.After(time.Second):
			t.Fatal("timed out waiting for concurrent checks to start")
		}
	}
	select {
	case accountID := <-tester.started:
		t.Fatalf("concurrency limit 2 exceeded by account %d", accountID)
	case <-time.After(100 * time.Millisecond):
	}
	close(release)

	select {
	case finished := <-outcome:
		require.NoError(t, finished.err)
		require.Equal(t, 4, finished.result.CheckedCount)
		require.Equal(t, int32(2), tester.maxActive.Load())
	case <-time.After(time.Second):
		t.Fatal("health guard concurrency test did not finish")
	}
}

func TestSupplierAccountHealthGuardRunContinuesAfterSingleAccountTimeout(t *testing.T) {
	repo := &supplierAccountHealthGuardRepoStub{candidates: []SupplierAccountHealthGuardCandidate{
		newSupplierAccountHealthGuardCandidate(1, "timeout-account", "openai", true, SupplierAccountHealthGuardSource{ProviderAccountID: 1}),
		newSupplierAccountHealthGuardCandidate(2, "healthy-account", "openai", true, SupplierAccountHealthGuardSource{ProviderAccountID: 2}),
	}}
	tester := &supplierAccountHealthGuardTesterStub{fn: func(ctx context.Context, accountID int64, _ string) (*ScheduledTestResult, error) {
		if accountID == 1 {
			<-ctx.Done()
			return nil, ctx.Err()
		}
		return &ScheduledTestResult{Status: "success", LatencyMs: 20}, nil
	}}
	guard := NewSupplierAccountHealthGuardService(repo, &supplierAccountHealthGuardAccountStoreStub{}, tester)

	result, err := guard.Run(context.Background(), SupplierAccountHealthGuardConfig{
		AccountIDs:               []int64{1, 2},
		Concurrency:              1,
		TimeoutPerAccountSeconds: 1,
		PlatformModels:           map[string]string{"openai": "gpt-4o-mini"},
	}, time.Now())

	require.NoError(t, err)
	require.Equal(t, 2, result.CheckedCount)
	require.Equal(t, 1, result.FailedCount)
	require.Equal(t, 1, result.HealthyCount)
	require.Equal(t, SupplierAccountHealthGuardStatusFailed, result.Items[0].Status)
	require.Equal(t, "\u6d4b\u8bd5\u8d85\u65f6", result.Items[0].Reason)
	require.Equal(t, SupplierAccountHealthGuardStatusHealthy, result.Items[1].Status)
}

func TestSupplierAccountHealthGuardRunDisablesAfterSlowThreshold(t *testing.T) {
	candidate := newSupplierAccountHealthGuardCandidate(31, "slow-account", "openai", true, SupplierAccountHealthGuardSource{ProviderAccountID: 31})
	candidate.LocalAccount.Extra = map[string]any{supplierHealthGuardSlowCountExtraKey: 1}
	store := &supplierAccountHealthGuardAccountStoreStub{}
	tester := &supplierAccountHealthGuardTesterStub{
		results: map[int64]*ScheduledTestResult{31: {Status: "success", LatencyMs: 1501}},
		errs:    map[int64]error{},
	}
	guard := NewSupplierAccountHealthGuardService(&supplierAccountHealthGuardRepoStub{candidates: []SupplierAccountHealthGuardCandidate{candidate}}, store, tester)

	result, err := guard.Run(context.Background(), SupplierAccountHealthGuardConfig{
		AccountIDs:       []int64{31},
		SlowThreshold:    2,
		HealthyLatencyMs: 1500,
		PlatformModels:   map[string]string{"openai": "gpt-4o-mini"},
	}, time.Now())

	require.NoError(t, err)
	require.Equal(t, 1, result.SlowCount)
	require.Equal(t, 1, result.DisabledCount)
	require.Equal(t, 2, result.Items[0].ConsecutiveSlow)
	require.Equal(t, []supplierAccountHealthGuardSetCall{{accountID: 31, schedulable: false}}, store.setCalls)
}

func TestSupplierAccountHealthGuardRunKeepsSchedulingBeforeThreshold(t *testing.T) {
	candidate := newSupplierAccountHealthGuardCandidate(32, "before-threshold-account", "openai", true, SupplierAccountHealthGuardSource{ProviderAccountID: 32})
	store := &supplierAccountHealthGuardAccountStoreStub{}
	tester := &supplierAccountHealthGuardTesterStub{
		results: map[int64]*ScheduledTestResult{32: {Status: "failed", ErrorMessage: "temporary error"}},
		errs:    map[int64]error{},
	}
	guard := NewSupplierAccountHealthGuardService(&supplierAccountHealthGuardRepoStub{candidates: []SupplierAccountHealthGuardCandidate{candidate}}, store, tester)

	result, err := guard.Run(context.Background(), SupplierAccountHealthGuardConfig{
		AccountIDs:       []int64{32},
		FailureThreshold: 2,
		PlatformModels:   map[string]string{"openai": "gpt-4o-mini"},
	}, time.Now())

	require.NoError(t, err)
	require.Equal(t, 1, result.FailedCount)
	require.Zero(t, result.DisabledCount)
	require.True(t, result.Items[0].SchedulableAfter)
	require.Equal(t, SupplierAccountHealthGuardActionNone, result.Items[0].Action)
	require.Empty(t, store.setCalls)
}

func TestSupplierAccountHealthGuardRunUsesEffectivePlatformForDefaults(t *testing.T) {
	candidate := newSupplierAccountHealthGuardCandidate(23, "覆盖平台账号", "openai", true, SupplierAccountHealthGuardSource{ProviderAccountID: 13})
	candidate.PlatformOverride = "grok"
	candidate.EffectivePlatform = "grok"
	store := &supplierAccountHealthGuardAccountStoreStub{}
	tester := &supplierAccountHealthGuardTesterStub{
		results: map[int64]*ScheduledTestResult{23: {Status: "success", LatencyMs: 800}},
		errs:    map[int64]error{},
	}
	guard := NewSupplierAccountHealthGuardService(&supplierAccountHealthGuardRepoStub{candidates: []SupplierAccountHealthGuardCandidate{candidate}}, store, tester)

	result, err := guard.Run(context.Background(), SupplierAccountHealthGuardConfig{
		AccountIDs:        []int64{23},
		HealthyLatencyMs:  5000,
		PlatformModels:    map[string]string{"openai": "gpt-platform", "grok": "grok-platform"},
		PlatformLatencyMs: map[string]int64{"openai": 2000, "grok": 1200},
	}, time.Now())

	require.NoError(t, err)
	require.Equal(t, 1, result.HealthyCount)
	require.Equal(t, "grok-platform", tester.calls[0].modelID)
	require.Equal(t, "grok", result.Items[0].Platform)
	require.Equal(t, int64(1200), result.Items[0].LatencyLimitMs)
}

func newSupplierAccountHealthGuardCandidate(accountID int64, name, platform string, schedulable bool, source SupplierAccountHealthGuardSource) SupplierAccountHealthGuardCandidate {
	return SupplierAccountHealthGuardCandidate{
		Source: source, MatchStatus: SupplierAccountHealthGuardMatchMatched, MatchCount: 1, LocalAccountID: accountID,
		EffectivePlatform: platform,
		LocalAccount:      &Account{ID: accountID, Name: name, Platform: platform, Status: StatusActive, Schedulable: schedulable, Extra: map[string]any{}},
	}
}

func supplierAccountHealthGuardReasonNames(reasons []SupplierAccountHealthGuardSkipReason) []string {
	out := make([]string, 0, len(reasons))
	for _, reason := range reasons {
		out = append(out, reason.Reason)
	}
	return out
}

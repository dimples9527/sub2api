package service

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/pagination"
)

func TestAccountTestServiceBatchTestAccountsLimitsConcurrencyAndKeepsOrder(t *testing.T) {
	repo := &batchAccountTestRepo{
		accounts: map[int64]*Account{
			1: {ID: 1, Name: "one", Platform: PlatformOpenAI},
			2: {ID: 2, Name: "two", Platform: PlatformGemini},
			3: {ID: 3, Name: "three", Platform: PlatformAnthropic},
		},
	}
	var mu sync.Mutex
	running := 0
	maxRunning := 0

	svc := &AccountTestService{
		accountRepo: repo,
		runTestBackgroundFunc: func(ctx context.Context, accountID int64, modelID string) (*ScheduledTestResult, error) {
			mu.Lock()
			running++
			if running > maxRunning {
				maxRunning = running
			}
			mu.Unlock()

			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(20 * time.Millisecond):
			}

			mu.Lock()
			running--
			mu.Unlock()
			return &ScheduledTestResult{
				Status:     "success",
				LatencyMs:  20,
				StartedAt:  time.Now().UTC(),
				FinishedAt: time.Now().UTC(),
			}, nil
		},
	}

	result, err := svc.BatchTestAccounts(context.Background(), BatchAccountTestInput{
		AccountIDs:        []int64{1, 2, 3},
		ModelID:           "test-model",
		Concurrency:       2,
		TimeoutPerAccount: time.Second,
	})
	if err != nil {
		t.Fatalf("BatchTestAccounts returned error: %v", err)
	}
	if result.Total != 3 || result.Success != 3 || result.Failed != 0 {
		t.Fatalf("summary = %+v, want 3 total, 3 success, 0 failed", result)
	}
	if maxRunning > 2 {
		t.Fatalf("maxRunning = %d, want <= 2", maxRunning)
	}
	gotIDs := []int64{result.Results[0].AccountID, result.Results[1].AccountID, result.Results[2].AccountID}
	wantIDs := []int64{1, 2, 3}
	for i := range wantIDs {
		if gotIDs[i] != wantIDs[i] {
			t.Fatalf("result order = %v, want %v", gotIDs, wantIDs)
		}
	}
}

func TestAccountTestServiceBatchTestAccountsUsesPlatformModelsAndIncludesSchedulable(t *testing.T) {
	repo := &batchAccountTestRepo{
		accounts: map[int64]*Account{
			1: {ID: 1, Name: "openai-account", Platform: PlatformOpenAI, Schedulable: true},
			2: {ID: 2, Name: "gemini-account", Platform: PlatformGemini, Schedulable: false},
		},
	}
	var seenMu sync.Mutex
	seenModels := make(map[int64]string)
	svc := &AccountTestService{
		accountRepo: repo,
		runTestBackgroundFunc: func(ctx context.Context, accountID int64, modelID string) (*ScheduledTestResult, error) {
			seenMu.Lock()
			seenModels[accountID] = modelID
			seenMu.Unlock()
			return &ScheduledTestResult{
				Status:     "success",
				LatencyMs:  3,
				StartedAt:  time.Now().UTC(),
				FinishedAt: time.Now().UTC(),
			}, nil
		},
	}

	result, err := svc.BatchTestAccounts(context.Background(), BatchAccountTestInput{
		AccountIDs: []int64{1, 2},
		ModelID:    "fallback-model",
		ModelIDsByPlatform: map[string]string{
			PlatformOpenAI: "gpt-4.1-mini",
			PlatformGemini: "gemini-2.5-flash",
		},
		Concurrency:       2,
		TimeoutPerAccount: time.Second,
	})
	if err != nil {
		t.Fatalf("BatchTestAccounts returned error: %v", err)
	}

	seenMu.Lock()
	openAIModel := seenModels[1]
	geminiModel := seenModels[2]
	seenMu.Unlock()
	if openAIModel != "gpt-4.1-mini" || geminiModel != "gemini-2.5-flash" {
		t.Fatalf("seen models = %#v, want platform-specific models", seenModels)
	}
	if !result.Results[0].Schedulable {
		t.Fatalf("first result schedulable = false, want true")
	}
	if result.Results[1].Schedulable {
		t.Fatalf("second result schedulable = true, want false")
	}
}

func TestAccountTestServiceBatchTestAccountsIsolatesFailuresAndTimeouts(t *testing.T) {
	repo := &batchAccountTestRepo{
		accounts: map[int64]*Account{
			1: {ID: 1, Name: "ok", Platform: PlatformOpenAI},
			2: {ID: 2, Name: "bad", Platform: PlatformOpenAI},
			3: {ID: 3, Name: "slow", Platform: PlatformOpenAI},
		},
	}
	svc := &AccountTestService{
		accountRepo: repo,
		runTestBackgroundFunc: func(ctx context.Context, accountID int64, modelID string) (*ScheduledTestResult, error) {
			switch accountID {
			case 1:
				return &ScheduledTestResult{Status: "success", LatencyMs: 3, StartedAt: time.Now().UTC(), FinishedAt: time.Now().UTC()}, nil
			case 2:
				return &ScheduledTestResult{Status: "failed", ErrorMessage: "upstream 401", LatencyMs: 4, StartedAt: time.Now().UTC(), FinishedAt: time.Now().UTC()}, errors.New("upstream 401")
			case 3:
				<-ctx.Done()
				return nil, ctx.Err()
			default:
				return nil, errors.New("unexpected account")
			}
		},
	}

	result, err := svc.BatchTestAccounts(context.Background(), BatchAccountTestInput{
		AccountIDs:        []int64{1, 2, 3, 404},
		Concurrency:       3,
		TimeoutPerAccount: 10 * time.Millisecond,
	})
	if err != nil {
		t.Fatalf("BatchTestAccounts returned error: %v", err)
	}
	if result.Total != 4 || result.Success != 1 || result.Failed != 3 {
		t.Fatalf("summary = %+v, want 4 total, 1 success, 3 failed", result)
	}
	if result.Results[1].Status != "failed" || result.Results[1].ErrorMessage == "" {
		t.Fatalf("failure item = %+v, want failed with error", result.Results[1])
	}
	if result.Results[2].Status != "timeout" {
		t.Fatalf("timeout item = %+v, want timeout", result.Results[2])
	}
	if result.Results[3].Status != "not_found" {
		t.Fatalf("missing item = %+v, want not_found", result.Results[3])
	}
}

func TestAccountTestServiceBatchTestAccountsPersistsTimeoutStatusIndependently(t *testing.T) {
	repo := &batchAccountTestRepo{
		accounts: map[int64]*Account{
			1: {ID: 1, Name: "slow", Platform: PlatformOpenAI},
		},
	}
	svc := &AccountTestService{
		accountRepo: repo,
		runTestBackgroundFunc: func(ctx context.Context, accountID int64, modelID string) (*ScheduledTestResult, error) {
			<-ctx.Done()
			return nil, ctx.Err()
		},
	}

	result, err := svc.BatchTestAccounts(context.Background(), BatchAccountTestInput{
		AccountIDs:        []int64{1},
		TimeoutPerAccount: 10 * time.Millisecond,
	})
	if err != nil {
		t.Fatalf("BatchTestAccounts returned error: %v", err)
	}
	if result.Results[0].Status != "timeout" {
		t.Fatalf("status = %s, want timeout", result.Results[0].Status)
	}
	repo.mu.Lock()
	updatedExtra := repo.updatedExtra[1]
	updateCtxErr := repo.updateCtxErr[1]
	repo.mu.Unlock()
	if updateCtxErr != nil {
		t.Fatalf("UpdateExtra context error = %v, want nil", updateCtxErr)
	}
	if updatedExtra["last_test_status"] != "failed" {
		t.Fatalf("last_test_status = %v, want failed", updatedExtra["last_test_status"])
	}
	if updatedExtra["last_tested_at"] == "" {
		t.Fatal("last_tested_at is empty")
	}
	if updatedExtra["last_test_error"] != "account test timed out" {
		t.Fatalf("last_test_error = %v, want timeout message", updatedExtra["last_test_error"])
	}
}

func TestAccountTestServiceBatchTestAccountsRejectsTooManyAccounts(t *testing.T) {
	ids := make([]int64, maxBatchAccountTestAccounts+1)
	for i := range ids {
		ids[i] = int64(i + 1)
	}
	svc := &AccountTestService{accountRepo: &batchAccountTestRepo{}}
	_, err := svc.BatchTestAccounts(context.Background(), BatchAccountTestInput{AccountIDs: ids})
	if err == nil {
		t.Fatal("BatchTestAccounts error = nil, want too many accounts error")
	}
}

func TestAccountTestServiceStartBatchTestAccountsReturnsJobBeforeCompletion(t *testing.T) {
	release := make(chan struct{})
	started := make(chan struct{})
	repo := &batchAccountTestRepo{
		accounts: map[int64]*Account{
			1: {ID: 1, Name: "one", Platform: PlatformOpenAI},
		},
	}
	svc := &AccountTestService{
		accountRepo: repo,
		runTestBackgroundFunc: func(ctx context.Context, accountID int64, modelID string) (*ScheduledTestResult, error) {
			close(started)
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-release:
				return &ScheduledTestResult{
					Status:     "success",
					LatencyMs:  5,
					StartedAt:  time.Now().UTC(),
					FinishedAt: time.Now().UTC(),
				}, nil
			}
		},
	}

	job, err := svc.StartBatchTestAccounts(context.Background(), BatchAccountTestInput{
		AccountIDs:        []int64{1},
		TimeoutPerAccount: time.Second,
	})
	if err != nil {
		t.Fatalf("StartBatchTestAccounts returned error: %v", err)
	}
	if job.JobID == "" {
		t.Fatal("job_id is empty")
	}
	if job.Total != 1 || job.Completed != 0 {
		t.Fatalf("initial job = %+v, want total 1 and completed 0", job)
	}

	select {
	case <-started:
	case <-time.After(time.Second):
		t.Fatal("runner did not start")
	}

	pending, err := svc.GetBatchTestJob(context.Background(), job.JobID)
	if err != nil {
		t.Fatalf("GetBatchTestJob returned error: %v", err)
	}
	if pending.Status != "running" || pending.Completed != 0 {
		t.Fatalf("pending job = %+v, want running with 0 completed", pending)
	}

	close(release)
	completed := waitForBatchTestJobStatus(t, svc, job.JobID, "completed")
	if completed.Completed != 1 || completed.Success != 1 || completed.Failed != 0 {
		t.Fatalf("completed job = %+v, want 1 success", completed)
	}
	if len(completed.Results) != 1 || completed.Results[0].Status != "success" {
		t.Fatalf("results = %+v, want success item", completed.Results)
	}
}

func TestAccountTestServiceCancelBatchTestJobCancelsPendingWork(t *testing.T) {
	started := make(chan struct{})
	repo := &batchAccountTestRepo{
		accounts: map[int64]*Account{
			1: {ID: 1, Name: "slow", Platform: PlatformOpenAI},
			2: {ID: 2, Name: "queued", Platform: PlatformOpenAI},
		},
	}
	svc := &AccountTestService{
		accountRepo: repo,
		runTestBackgroundFunc: func(ctx context.Context, accountID int64, modelID string) (*ScheduledTestResult, error) {
			if accountID == 1 {
				close(started)
			}
			<-ctx.Done()
			return nil, ctx.Err()
		},
	}

	job, err := svc.StartBatchTestAccounts(context.Background(), BatchAccountTestInput{
		AccountIDs:        []int64{1, 2},
		Concurrency:       1,
		TimeoutPerAccount: time.Second,
	})
	if err != nil {
		t.Fatalf("StartBatchTestAccounts returned error: %v", err)
	}
	select {
	case <-started:
	case <-time.After(time.Second):
		t.Fatal("runner did not start")
	}

	cancelling, err := svc.CancelBatchTestJob(context.Background(), job.JobID)
	if err != nil {
		t.Fatalf("CancelBatchTestJob returned error: %v", err)
	}
	if cancelling.Status != "cancelling" {
		t.Fatalf("status = %s, want cancelling", cancelling.Status)
	}

	cancelled := waitForBatchTestJobStatus(t, svc, job.JobID, "cancelled")
	if cancelled.Completed != 2 || cancelled.Success != 0 || cancelled.Failed != 2 {
		t.Fatalf("cancelled job = %+v, want all failed/cancelled", cancelled)
	}
	for _, item := range cancelled.Results {
		if item.Status != "cancelled" {
			t.Fatalf("result item = %+v, want cancelled", item)
		}
	}
}

func TestAccountTestServiceStartBatchTestAccountsTimesOutPendingWork(t *testing.T) {
	started := make(chan struct{})
	repo := &batchAccountTestRepo{
		accounts: map[int64]*Account{
			1: {ID: 1, Name: "slow", Platform: PlatformOpenAI},
			2: {ID: 2, Name: "queued", Platform: PlatformOpenAI},
		},
	}
	svc := &AccountTestService{
		accountRepo: repo,
		runTestBackgroundFunc: func(ctx context.Context, accountID int64, modelID string) (*ScheduledTestResult, error) {
			if accountID == 1 {
				close(started)
			}
			<-ctx.Done()
			return nil, ctx.Err()
		},
	}

	job, err := svc.StartBatchTestAccounts(context.Background(), BatchAccountTestInput{
		AccountIDs:        []int64{1, 2},
		Concurrency:       1,
		TimeoutPerAccount: time.Second,
		Timeout:           20 * time.Millisecond,
	})
	if err != nil {
		t.Fatalf("StartBatchTestAccounts returned error: %v", err)
	}
	select {
	case <-started:
	case <-time.After(time.Second):
		t.Fatal("runner did not start")
	}

	completed := waitForBatchTestJobStatus(t, svc, job.JobID, "completed")
	if completed.Completed != 2 || completed.Success != 0 || completed.Failed != 2 {
		t.Fatalf("completed job = %+v, want all failed/timeouts", completed)
	}
	if completed.ErrorMessage != "batch account test timed out" {
		t.Fatalf("error message = %q, want batch timeout", completed.ErrorMessage)
	}
	for _, item := range completed.Results {
		if item.Status != "timeout" || item.ErrorMessage != "batch account test timed out" {
			t.Fatalf("result item = %+v, want batch timeout", item)
		}
	}
}

func TestAccountTestServiceStartBatchTestAccountsPrunesExpiredFinishedJobs(t *testing.T) {
	repo := &batchAccountTestRepo{
		accounts: map[int64]*Account{
			1: {ID: 1, Name: "one", Platform: PlatformOpenAI},
		},
	}
	svc := &AccountTestService{
		accountRepo: repo,
		batchTestJobs: map[string]*BatchAccountTestJob{
			"old": {
				JobID:          "old",
				Status:         "completed",
				Total:          1,
				Completed:      1,
				CreatedAt:      time.Now().Add(-2 * batchAccountTestJobRetention).UTC().Format(time.RFC3339Nano),
				FinishedAt:     time.Now().Add(-2 * batchAccountTestJobRetention).UTC().Format(time.RFC3339Nano),
				createdAtTime:  time.Now().Add(-2 * batchAccountTestJobRetention),
				finishedAtTime: time.Now().Add(-2 * batchAccountTestJobRetention),
			},
		},
		runTestBackgroundFunc: func(ctx context.Context, accountID int64, modelID string) (*ScheduledTestResult, error) {
			return &ScheduledTestResult{Status: "success", LatencyMs: 1, StartedAt: time.Now().UTC(), FinishedAt: time.Now().UTC()}, nil
		},
	}

	job, err := svc.StartBatchTestAccounts(context.Background(), BatchAccountTestInput{AccountIDs: []int64{1}})
	if err != nil {
		t.Fatalf("StartBatchTestAccounts returned error: %v", err)
	}
	if job.JobID == "" {
		t.Fatal("job_id is empty")
	}
	svc.batchTestJobsMu.RLock()
	_, oldExists := svc.batchTestJobs["old"]
	svc.batchTestJobsMu.RUnlock()
	if oldExists {
		t.Fatal("expired finished job was not pruned")
	}
}

func waitForBatchTestJobStatus(t *testing.T, svc *AccountTestService, jobID string, status string) BatchAccountTestJob {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	var last BatchAccountTestJob
	for time.Now().Before(deadline) {
		job, err := svc.GetBatchTestJob(context.Background(), jobID)
		if err != nil {
			t.Fatalf("GetBatchTestJob returned error: %v", err)
		}
		last = *job
		if job.Status == status {
			return *job
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("job status = %s, want %s, last = %+v", last.Status, status, last)
	return last
}

type batchAccountTestRepo struct {
	noopAccountRepo
	mu           sync.Mutex
	accounts     map[int64]*Account
	updatedExtra map[int64]map[string]any
	updateCtxErr map[int64]error
}

func (r *batchAccountTestRepo) GetByIDs(_ context.Context, ids []int64) ([]*Account, error) {
	out := make([]*Account, 0, len(ids))
	for _, id := range ids {
		if account, ok := r.accounts[id]; ok {
			cp := *account
			out = append(out, &cp)
		}
	}
	return out, nil
}

func (r *batchAccountTestRepo) UpdateExtra(ctx context.Context, accountID int64, extra map[string]any) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.updatedExtra == nil {
		r.updatedExtra = make(map[int64]map[string]any)
	}
	if r.updateCtxErr == nil {
		r.updateCtxErr = make(map[int64]error)
	}
	cp := make(map[string]any, len(extra))
	for k, v := range extra {
		cp[k] = v
	}
	r.updatedExtra[accountID] = cp
	r.updateCtxErr[accountID] = ctx.Err()
	return nil
}

type noopAccountRepo struct{}

func (noopAccountRepo) Create(context.Context, *Account) error { return nil }
func (noopAccountRepo) GetByID(context.Context, int64) (*Account, error) {
	return nil, ErrAccountNotFound
}
func (noopAccountRepo) GetByIDs(context.Context, []int64) ([]*Account, error)       { return nil, nil }
func (noopAccountRepo) ExistsByID(context.Context, int64) (bool, error)             { return false, nil }
func (noopAccountRepo) GetByCRSAccountID(context.Context, string) (*Account, error) { return nil, nil }
func (noopAccountRepo) FindByExtraField(context.Context, string, any) ([]Account, error) {
	return nil, nil
}
func (noopAccountRepo) ListCRSAccountIDs(context.Context) (map[string]int64, error) { return nil, nil }
func (noopAccountRepo) Update(context.Context, *Account) error                      { return nil }
func (noopAccountRepo) Delete(context.Context, int64) error                         { return nil }
func (noopAccountRepo) List(context.Context, pagination.PaginationParams) ([]Account, *pagination.PaginationResult, error) {
	return nil, nil, nil
}
func (noopAccountRepo) ListWithFilters(context.Context, pagination.PaginationParams, string, string, string, string, int64, string) ([]Account, *pagination.PaginationResult, error) {
	return nil, nil, nil
}
func (noopAccountRepo) ListAllWithFilters(context.Context, string, string, string, string, int64, string) ([]Account, error) {
	return nil, nil
}
func (noopAccountRepo) ListByGroup(context.Context, int64) ([]Account, error) { return nil, nil }
func (noopAccountRepo) ListActive(context.Context) ([]Account, error)         { return nil, nil }
func (noopAccountRepo) ListOAuthRefreshCandidates(context.Context) ([]Account, error) {
	return nil, nil
}
func (noopAccountRepo) ListByPlatform(context.Context, string) ([]Account, error)      { return nil, nil }
func (noopAccountRepo) UpdateLastUsed(context.Context, int64) error                    { return nil }
func (noopAccountRepo) BatchUpdateLastUsed(context.Context, map[int64]time.Time) error { return nil }
func (noopAccountRepo) SetError(context.Context, int64, string) error                  { return nil }
func (noopAccountRepo) ClearError(context.Context, int64) error                        { return nil }
func (noopAccountRepo) SetSchedulable(context.Context, int64, bool) error              { return nil }
func (noopAccountRepo) AutoPauseExpiredAccounts(context.Context, time.Time) (int64, error) {
	return 0, nil
}
func (noopAccountRepo) BindGroups(context.Context, int64, []int64) error   { return nil }
func (noopAccountRepo) ListSchedulable(context.Context) ([]Account, error) { return nil, nil }
func (noopAccountRepo) ListSchedulableByGroupID(context.Context, int64) ([]Account, error) {
	return nil, nil
}
func (noopAccountRepo) ListSchedulableByPlatform(context.Context, string) ([]Account, error) {
	return nil, nil
}
func (noopAccountRepo) ListSchedulableByGroupIDAndPlatform(context.Context, int64, string) ([]Account, error) {
	return nil, nil
}
func (noopAccountRepo) ListSchedulableByPlatforms(context.Context, []string) ([]Account, error) {
	return nil, nil
}
func (noopAccountRepo) ListSchedulableByGroupIDAndPlatforms(context.Context, int64, []string) ([]Account, error) {
	return nil, nil
}
func (noopAccountRepo) ListSchedulableUngroupedByPlatform(context.Context, string) ([]Account, error) {
	return nil, nil
}
func (noopAccountRepo) ListSchedulableUngroupedByPlatforms(context.Context, []string) ([]Account, error) {
	return nil, nil
}
func (noopAccountRepo) SetRateLimited(context.Context, int64, time.Time) error { return nil }
func (noopAccountRepo) SetModelRateLimit(context.Context, int64, string, time.Time, ...string) error {
	return nil
}
func (noopAccountRepo) SetOverloaded(context.Context, int64, time.Time) error { return nil }
func (noopAccountRepo) SetTempUnschedulable(context.Context, int64, time.Time, string) error {
	return nil
}
func (noopAccountRepo) ClearTempUnschedulable(context.Context, int64) error      { return nil }
func (noopAccountRepo) ClearRateLimit(context.Context, int64) error              { return nil }
func (noopAccountRepo) ClearAntigravityQuotaScopes(context.Context, int64) error { return nil }
func (noopAccountRepo) ClearModelRateLimits(context.Context, int64) error        { return nil }
func (noopAccountRepo) UpdateSessionWindow(context.Context, int64, *time.Time, *time.Time, string) error {
	return nil
}
func (noopAccountRepo) UpdateSessionWindowEnd(context.Context, int64, time.Time) error { return nil }
func (noopAccountRepo) UpdateExtra(context.Context, int64, map[string]any) error       { return nil }
func (noopAccountRepo) BulkUpdate(context.Context, []int64, AccountBulkUpdate) (int64, error) {
	return 0, nil
}
func (noopAccountRepo) IncrementQuotaUsed(context.Context, int64, float64) error { return nil }
func (noopAccountRepo) ResetQuotaUsed(context.Context, int64) error              { return nil }
func (noopAccountRepo) RevertProxyFallback(context.Context, int64) error         { return nil }
func (noopAccountRepo) ListShadowsByParent(context.Context, int64) ([]*Account, error) {
	return nil, nil
}

var _ AccountRepository = (*batchAccountTestRepo)(nil)

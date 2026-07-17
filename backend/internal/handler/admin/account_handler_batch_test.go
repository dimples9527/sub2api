package admin

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
)

func TestAccountHandlerBatchTestStartsJobAndPollsResult(t *testing.T) {
	gin.SetMode(gin.TestMode)
	repo := &batchTestHandlerAccountRepo{
		accounts: map[int64]*service.Account{
			1: {ID: 1, Name: "alpha", Platform: service.PlatformOpenAI},
			2: {ID: 2, Name: "beta", Platform: service.PlatformGemini},
		},
	}
	var seenMu sync.Mutex
	seenModels := make(map[int64]string)
	testSvc := service.NewAccountTestServiceWithRunner(
		repo,
		func(ctx context.Context, accountID int64, modelID string) (*service.ScheduledTestResult, error) {
			seenMu.Lock()
			seenModels[accountID] = modelID
			seenMu.Unlock()
			return &service.ScheduledTestResult{
				Status:     "success",
				LatencyMs:  7,
				StartedAt:  time.Now().UTC(),
				FinishedAt: time.Now().UTC(),
			}, nil
		},
	)
	handler := NewAccountHandler(
		newStubAdminService(),
		nil,
		nil,
		nil,
		nil,
		nil,
		nil,
		testSvc,
		nil,
		nil,
		nil,
		nil,
		nil,
	)
	router := gin.New()
	router.POST("/api/v1/admin/accounts/batch-test", handler.BatchTest)
	router.GET("/api/v1/admin/accounts/batch-test/:job_id", handler.GetBatchTest)

	body, _ := json.Marshal(BatchTestAccountsRequest{
		AccountIDs:            []int64{1, 2},
		ModelID:               "probe-model",
		ModelIDsByPlatform:    map[string]string{service.PlatformGemini: "gemini-2.5-flash"},
		Concurrency:           2,
		TimeoutPerAccountSecs: 1,
	})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/accounts/batch-test", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", w.Code, w.Body.String())
	}
	var payload struct {
		Data service.BatchAccountTestJob `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &payload); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if payload.Data.JobID == "" || payload.Data.Total != 2 {
		t.Fatalf("data = %+v, want job_id and total 2", payload.Data)
	}

	var polled service.BatchAccountTestJob
	deadline := time.Now().Add(time.Second)
	for time.Now().Before(deadline) {
		getReq := httptest.NewRequest(http.MethodGet, "/api/v1/admin/accounts/batch-test/"+payload.Data.JobID, nil)
		getW := httptest.NewRecorder()
		router.ServeHTTP(getW, getReq)
		if getW.Code != http.StatusOK {
			t.Fatalf("poll status = %d, body = %s", getW.Code, getW.Body.String())
		}
		var getPayload struct {
			Data service.BatchAccountTestJob `json:"data"`
		}
		if err := json.Unmarshal(getW.Body.Bytes(), &getPayload); err != nil {
			t.Fatalf("unmarshal poll response: %v", err)
		}
		polled = getPayload.Data
		if polled.Status == "completed" {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	if polled.Status != "completed" || polled.Success != 2 || polled.Failed != 0 {
		t.Fatalf("polled job = %+v, want completed all success", polled)
	}
	if polled.Results[0].AccountName != "alpha" || polled.Results[1].AccountName != "beta" {
		t.Fatalf("results = %+v, want account names", polled.Results)
	}
	seenMu.Lock()
	gotGeminiModel := seenModels[2]
	seenMu.Unlock()
	if gotGeminiModel != "gemini-2.5-flash" {
		t.Fatalf("gemini model = %q, want platform-specific model", gotGeminiModel)
	}
}

func TestAccountHandlerBatchTestRejectsTooManyAccounts(t *testing.T) {
	gin.SetMode(gin.TestMode)
	testSvc := service.NewAccountTestServiceWithRunner(
		&batchTestHandlerAccountRepo{accounts: map[int64]*service.Account{}},
		func(ctx context.Context, accountID int64, modelID string) (*service.ScheduledTestResult, error) {
			return nil, nil
		},
	)
	handler := NewAccountHandler(
		newStubAdminService(),
		nil,
		nil,
		nil,
		nil,
		nil,
		nil,
		testSvc,
		nil,
		nil,
		nil,
		nil,
		nil,
	)
	router := gin.New()
	router.POST("/api/v1/admin/accounts/batch-test", handler.BatchTest)

	accountIDs := make([]int64, 201)
	for i := range accountIDs {
		accountIDs[i] = int64(i + 1)
	}
	body, _ := json.Marshal(BatchTestAccountsRequest{AccountIDs: accountIDs})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/accounts/batch-test", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, body = %s, want 400", w.Code, w.Body.String())
	}
}

type batchTestHandlerAccountRepo struct {
	service.AccountRepository
	accounts map[int64]*service.Account
}

func (r *batchTestHandlerAccountRepo) GetByIDs(_ context.Context, ids []int64) ([]*service.Account, error) {
	out := make([]*service.Account, 0, len(ids))
	for _, id := range ids {
		if account, ok := r.accounts[id]; ok {
			cp := *account
			out = append(out, &cp)
		}
	}
	return out, nil
}

//go:build unit

package admin

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

type batchSyncAccountRepo struct {
	service.AccountRepository
	accounts     []*service.Account
	credUpdates  map[int64]map[string]any
	extraUpdates map[int64]map[string]any
}

func (r *batchSyncAccountRepo) GetByIDs(_ context.Context, ids []int64) ([]*service.Account, error) {
	out := make([]*service.Account, 0, len(ids))
	for _, id := range ids {
		for _, account := range r.accounts {
			if account.ID == id {
				out = append(out, account)
				break
			}
		}
	}
	return out, nil
}

func (r *batchSyncAccountRepo) UpdateCredentials(_ context.Context, id int64, credentials map[string]any) error {
	if r.credUpdates == nil {
		r.credUpdates = map[int64]map[string]any{}
	}
	r.credUpdates[id] = credentials
	return nil
}

func (r *batchSyncAccountRepo) UpdateExtra(_ context.Context, id int64, updates map[string]any) error {
	if r.extraUpdates == nil {
		r.extraUpdates = map[int64]map[string]any{}
	}
	r.extraUpdates[id] = updates
	return nil
}

func jsonModelsResponse(body string) *http.Response {
	return &http.Response{
		StatusCode: http.StatusOK,
		Header:     http.Header{"Content-Type": []string{"application/json"}},
		Body:       io.NopCloser(strings.NewReader(body)),
	}
}

// batchSyncCompleteModelsBody 带齐完整能力字段，避免同步时转去 models.dev 拉真实数据。
const batchSyncCompleteModelsBody = `{"data":[` +
	`{"id":"gpt-5","reasoning":false,"input_modalities":["text"],"context_window":128000},` +
	`{"id":"gpt-4o","reasoning":false,"input_modalities":["text"],"context_window":128000}]}`

// setupSupplierModelSyncRouter 按供应商账号页的真实布局注册：批量同步与同页的
// batch-test / batch-bind-groups 都是静态段，注册期冲突会直接 panic。
func setupSupplierModelSyncRouter(repo *batchSyncAccountRepo, upstream service.HTTPUpstream) *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	accountTestSvc := service.NewAccountTestService(
		repo,
		nil,
		nil,
		nil,
		nil,
		upstream,
		&config.Config{Security: config.SecurityConfig{URLAllowlist: config.URLAllowlistConfig{Enabled: false}}},
		nil,
	)
	handler := NewAccountHandler(newStubAdminService(), nil, nil, nil, nil, nil, nil, nil, accountTestSvc, nil, nil, nil, nil, nil)
	router.POST("/api/v1/admin/supplier-management/accounts/batch-test", handler.SupplierBatchTest)
	router.POST("/api/v1/admin/supplier-management/accounts/batch-bind-groups", handler.BatchBindSupplierAccountGroups)
	router.POST("/api/v1/admin/supplier-management/accounts/models/sync-upstream/batch", handler.SyncUpstreamModelsBatch)
	return router
}

func TestSupplierSyncUpstreamModelsBatchRouteRegistersWithoutPanic(t *testing.T) {
	// 供应商账号页的批量入口都是静态段，必须能共存：注册期冲突会直接 panic，
	// 且只在启动阶段暴露（路由注册本身没有别的测试覆盖）。
	require.NotPanics(t, func() {
		setupSupplierModelSyncRouter(&batchSyncAccountRepo{}, &syncUpstreamHTTPUpstream{})
	})
}

func TestSupplierSyncUpstreamModelsBatchRequiresAccountIDs(t *testing.T) {
	router := setupSupplierModelSyncRouter(&batchSyncAccountRepo{}, &syncUpstreamHTTPUpstream{
		resp: jsonModelsResponse(batchSyncCompleteModelsBody),
	})

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/supplier-management/accounts/models/sync-upstream/batch",
		strings.NewReader(`{"mode":"merge"}`))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestSupplierSyncUpstreamModelsBatchAppliesWhitelist(t *testing.T) {
	repo := &batchSyncAccountRepo{accounts: []*service.Account{
		{
			ID:       11,
			Name:     "batch-target",
			Platform: service.PlatformOpenAI,
			Type:     service.AccountTypeAPIKey,
			Credentials: map[string]any{
				"api_key":       "openai-key",
				"base_url":      "https://openai.example.com/v1",
				"model_mapping": map[string]any{"gpt-4o": "gpt-4o"},
			},
		},
	}}
	router := setupSupplierModelSyncRouter(repo, &syncUpstreamHTTPUpstream{
		resp: jsonModelsResponse(batchSyncCompleteModelsBody),
	})

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/supplier-management/accounts/models/sync-upstream/batch",
		strings.NewReader(`{"account_ids":[11],"mode":"merge","apply":true}`))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)

	// response.Success 统一包装成 { data: ... }，断言要从 data 里取，不能解到顶层。
	var payload struct {
		Data struct {
			Total   int `json:"total"`
			Success int `json:"success"`
			Results []struct {
				AccountID   int64  `json:"account_id"`
				AccountName string `json:"account_name"`
				Status      string `json:"status"`
				Added       int    `json:"added"`
				FinalCount  int    `json:"final_count"`
			} `json:"results"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &payload))
	require.Equal(t, 1, payload.Data.Total)
	require.Equal(t, 1, payload.Data.Success)
	require.Equal(t, int64(11), payload.Data.Results[0].AccountID)
	require.Equal(t, "batch-target", payload.Data.Results[0].AccountName)
	require.Equal(t, service.UpstreamModelBatchSyncStatusSuccess, payload.Data.Results[0].Status)
	require.Equal(t, 1, payload.Data.Results[0].Added)
	require.Equal(t, 2, payload.Data.Results[0].FinalCount)

	mapping, ok := repo.credUpdates[11]["model_mapping"].(map[string]any)
	require.True(t, ok)
	require.Contains(t, mapping, "gpt-5")
	require.Contains(t, mapping, "gpt-4o")
}

func TestSupplierSyncUpstreamModelsBatchRejectsInvalidMode(t *testing.T) {
	repo := &batchSyncAccountRepo{accounts: []*service.Account{
		{
			ID:          12,
			Platform:    service.PlatformOpenAI,
			Type:        service.AccountTypeAPIKey,
			Credentials: map[string]any{"api_key": "openai-key", "base_url": "https://openai.example.com/v1"},
		},
	}}
	router := setupSupplierModelSyncRouter(repo, &syncUpstreamHTTPUpstream{
		resp: jsonModelsResponse(batchSyncCompleteModelsBody),
	})

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/supplier-management/accounts/models/sync-upstream/batch",
		strings.NewReader(`{"account_ids":[12],"mode":"overwrite"}`))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(rec, req)

	require.GreaterOrEqual(t, rec.Code, http.StatusBadRequest)
	// 非法模式不该走到落库那一步。
	require.Empty(t, repo.credUpdates)
}

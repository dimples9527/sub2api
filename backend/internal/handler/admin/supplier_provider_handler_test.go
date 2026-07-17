package admin

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

type supplierProviderHandlerRepoStub struct {
	items []*service.SupplierProvider
	next  int64
}

type supplierProviderTypeHandlerRepoStub struct {
	items []*service.SupplierProviderType
	next  int64
}

func (r *supplierProviderHandlerRepoStub) List(_ context.Context, params service.SupplierProviderListParams) ([]*service.SupplierProvider, int64, error) {
	items := make([]*service.SupplierProvider, 0, len(r.items))
	for _, item := range r.items {
		if params.Enabled != nil && item.Enabled != *params.Enabled {
			continue
		}
		clone := *item
		items = append(items, &clone)
	}
	return items, int64(len(items)), nil
}

func (r *supplierProviderHandlerRepoStub) Summary(_ context.Context, params service.SupplierProviderListParams) (service.SupplierProviderSummary, error) {
	var summary service.SupplierProviderSummary
	for _, item := range r.items {
		if params.Enabled != nil && item.Enabled != *params.Enabled {
			continue
		}
		summary.TotalCount++
		if item.Enabled {
			summary.EnabledCount++
		}
		if item.RiskLevel == "high" || item.RiskLevel == "critical" {
			summary.HighRiskCount++
		}
		if item.EstimatedDays != nil && *item.EstimatedDays < 3 {
			summary.LowBalanceCount++
		}
		if item.SyncStatus == "failed" {
			summary.SyncFailureCount++
		}
		summary.RateRiskCount += item.RateRiskCount
	}
	return summary, nil
}

func (r *supplierProviderHandlerRepoStub) GetByID(_ context.Context, id int64) (*service.SupplierProvider, error) {
	for _, item := range r.items {
		if item.ID == id {
			clone := *item
			return &clone, nil
		}
	}
	return nil, service.ErrSupplierProviderNotFound
}

func (r *supplierProviderHandlerRepoStub) Create(_ context.Context, item *service.SupplierProvider) error {
	r.next++
	item.ID = r.next
	if len(r.items) == 0 || item.IsDefault {
		for _, existing := range r.items {
			existing.IsDefault = false
		}
		item.IsDefault = true
	}
	clone := *item
	r.items = append(r.items, &clone)
	return nil
}

func (r *supplierProviderHandlerRepoStub) Update(_ context.Context, item *service.SupplierProvider) error {
	for index := range r.items {
		if r.items[index].ID == item.ID {
			if item.IsDefault {
				for _, existing := range r.items {
					existing.IsDefault = false
				}
			}
			clone := *item
			r.items[index] = &clone
			return nil
		}
	}
	return service.ErrSupplierProviderNotFound
}

func (r *supplierProviderHandlerRepoStub) Delete(_ context.Context, id int64) error {
	for index := range r.items {
		if r.items[index].ID == id {
			r.items = append(r.items[:index], r.items[index+1:]...)
			return nil
		}
	}
	return service.ErrSupplierProviderNotFound
}

func (r *supplierProviderHandlerRepoStub) SetDefault(ctx context.Context, id int64) (*service.SupplierProvider, error) {
	for _, item := range r.items {
		item.IsDefault = item.ID == id
	}
	return r.GetByID(ctx, id)
}

func (r *supplierProviderTypeHandlerRepoStub) List(_ context.Context, enabledOnly bool) ([]*service.SupplierProviderType, error) {
	items := make([]*service.SupplierProviderType, 0, len(r.items))
	for _, item := range r.items {
		if enabledOnly && !item.Enabled {
			continue
		}
		clone := *item
		items = append(items, &clone)
	}
	return items, nil
}

func (r *supplierProviderTypeHandlerRepoStub) GetByID(_ context.Context, id int64) (*service.SupplierProviderType, error) {
	for _, item := range r.items {
		if item.ID == id {
			clone := *item
			return &clone, nil
		}
	}
	return nil, service.ErrSupplierProviderTypeNotFound
}

func (r *supplierProviderTypeHandlerRepoStub) GetByCode(_ context.Context, code string) (*service.SupplierProviderType, error) {
	for _, item := range r.items {
		if item.Code == code {
			clone := *item
			return &clone, nil
		}
	}
	return nil, service.ErrSupplierProviderTypeNotFound
}

func (r *supplierProviderTypeHandlerRepoStub) Create(_ context.Context, item *service.SupplierProviderType) error {
	r.next++
	item.ID = r.next
	clone := *item
	r.items = append(r.items, &clone)
	return nil
}

func (r *supplierProviderTypeHandlerRepoStub) Update(_ context.Context, item *service.SupplierProviderType) error {
	for index := range r.items {
		if r.items[index].ID == item.ID {
			clone := *item
			r.items[index] = &clone
			return nil
		}
	}
	return service.ErrSupplierProviderTypeNotFound
}

func (r *supplierProviderTypeHandlerRepoStub) Delete(_ context.Context, id int64) error {
	for index := range r.items {
		if r.items[index].ID == id {
			r.items = append(r.items[:index], r.items[index+1:]...)
			return nil
		}
	}
	return service.ErrSupplierProviderTypeNotFound
}

type supplierProviderHandlerEncryptorStub struct{}

func (supplierProviderHandlerEncryptorStub) Encrypt(value string) (string, error) {
	return "cipher:" + value, nil
}
func (supplierProviderHandlerEncryptorStub) Decrypt(value string) (string, error) { return value, nil }

func newSupplierProviderHandlerTestRouter(repo *supplierProviderHandlerRepoStub) *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	svc := service.NewSupplierProviderService(repo, supplierProviderHandlerEncryptorStub{})
	handler := NewSupplierProviderHandler(svc)
	router.GET("/admin/supplier-management/providers", handler.List)
	router.GET("/admin/supplier-management/providers/:id", handler.Get)
	router.POST("/admin/supplier-management/providers", handler.Create)
	router.PUT("/admin/supplier-management/providers/:id", handler.Update)
	router.DELETE("/admin/supplier-management/providers/:id", handler.Delete)
	router.PUT("/admin/supplier-management/providers/:id/default", handler.SetDefault)
	return router
}

func newSupplierProviderTypeHandlerTestRouter(repo *supplierProviderTypeHandlerRepoStub) *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	svc := service.NewSupplierProviderTypeService(repo)
	handler := NewSupplierProviderTypeHandler(svc)
	router.GET("/admin/supplier-management/provider-types", handler.List)
	router.GET("/admin/supplier-management/provider-types/:id", handler.Get)
	router.POST("/admin/supplier-management/provider-types", handler.Create)
	router.PUT("/admin/supplier-management/provider-types/:id", handler.Update)
	router.DELETE("/admin/supplier-management/provider-types/:id", handler.Delete)
	return router
}

func TestSupplierProviderHandlerCreateListAndUpdate(t *testing.T) {
	repo := &supplierProviderHandlerRepoStub{}
	router := newSupplierProviderHandlerTestRouter(repo)

	body := []byte(`{
		"code":"primary",
		"name":"主供应商",
		"provider_type":"sub2api",
		"base_url":"https://supplier.example.com",
		"email":"admin@example.com",
		"password":"secret",
		"enabled":true,
		"account_rate_multiplier_scale":1
	}`)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/admin/supplier-management/providers", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(rec, req)
	require.Equal(t, http.StatusCreated, rec.Code)

	var createResp struct {
		Code int                      `json:"code"`
		Data service.SupplierProvider `json:"data"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &createResp))
	require.Equal(t, "primary", createResp.Data.Code)
	require.True(t, createResp.Data.CredentialConfigured)
	require.Empty(t, createResp.Data.PasswordEncrypted)
	require.Equal(t, "cipher:secret", repo.items[0].PasswordEncrypted)

	update := []byte(`{
		"code":"primary",
		"name":"主供应商更新",
		"provider_type":"sub2api",
		"base_url":"https://supplier.example.com",
		"email":"ops@example.com",
		"enabled":false,
		"account_rate_multiplier_scale":1
	}`)
	rec = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodPut, "/admin/supplier-management/providers/1", bytes.NewReader(update))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(rec, req)
	require.Equal(t, http.StatusOK, rec.Code)
	require.Equal(t, "cipher:secret", repo.items[0].PasswordEncrypted)

	rec = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodGet, "/admin/supplier-management/providers", nil)
	router.ServeHTTP(rec, req)
	require.Equal(t, http.StatusOK, rec.Code)
	var listResp struct {
		Code int                                `json:"code"`
		Data service.SupplierProviderListResult `json:"data"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &listResp))
	require.Len(t, listResp.Data.Items, 1)
	require.Equal(t, "主供应商更新", listResp.Data.Items[0].Name)
	require.True(t, listResp.Data.Items[0].CredentialConfigured)
}

func TestSupplierProviderHandlerSetDefaultAndDelete(t *testing.T) {
	repo := &supplierProviderHandlerRepoStub{next: 2, items: []*service.SupplierProvider{
		{ID: 1, Code: "one", Name: "供应商一", ProviderType: "sub2api", BaseURL: "https://one.example.com", Enabled: true, IsDefault: true},
		{ID: 2, Code: "two", Name: "供应商二", ProviderType: "sub2api", BaseURL: "https://two.example.com", Enabled: true},
	}}
	router := newSupplierProviderHandlerTestRouter(repo)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPut, "/admin/supplier-management/providers/2/default", nil)
	router.ServeHTTP(rec, req)
	require.Equal(t, http.StatusOK, rec.Code)
	require.False(t, repo.items[0].IsDefault)
	require.True(t, repo.items[1].IsDefault)

	rec = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodDelete, "/admin/supplier-management/providers/1", nil)
	router.ServeHTTP(rec, req)
	require.Equal(t, http.StatusOK, rec.Code)
	require.Len(t, repo.items, 1)
	require.Equal(t, int64(2), repo.items[0].ID)
}

func TestSupplierProviderHandlerRejectsInvalidID(t *testing.T) {
	router := newSupplierProviderHandlerTestRouter(&supplierProviderHandlerRepoStub{})
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/admin/supplier-management/providers/bad", nil)
	router.ServeHTTP(rec, req)
	require.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestSupplierProviderTypeHandlerCreateListUpdateAndDelete(t *testing.T) {
	repo := &supplierProviderTypeHandlerRepoStub{}
	router := newSupplierProviderTypeHandlerTestRouter(repo)

	body := []byte(`{
		"code":"sub2api",
		"name":"Sub2API",
		"login_url":"https://supplier.example.com/login",
		"api_keys_url":"https://supplier.example.com/keys",
		"groups_url":"https://supplier.example.com/groups",
		"available_groups_url":"https://supplier.example.com/available-groups",
		"balance_url":"https://supplier.example.com/balance",
		"usage_cost_url":"https://supplier.example.com/cost",
		"enabled":true,
		"sort_order":10
	}`)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/admin/supplier-management/provider-types", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(rec, req)
	require.Equal(t, http.StatusCreated, rec.Code)

	var createResp struct {
		Code int                          `json:"code"`
		Data service.SupplierProviderType `json:"data"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &createResp))
	require.Equal(t, "sub2api", createResp.Data.Code)
	require.Equal(t, "Sub2API", repo.items[0].Name)

	update := []byte(`{
		"code":"sub2api",
		"name":"Sub2API 新版",
		"login_url":"https://supplier.example.com/login",
		"enabled":false,
		"sort_order":20
	}`)
	rec = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodPut, "/admin/supplier-management/provider-types/1", bytes.NewReader(update))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(rec, req)
	require.Equal(t, http.StatusOK, rec.Code)
	require.Equal(t, "Sub2API 新版", repo.items[0].Name)
	require.False(t, repo.items[0].Enabled)

	rec = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodGet, "/admin/supplier-management/provider-types", nil)
	router.ServeHTTP(rec, req)
	require.Equal(t, http.StatusOK, rec.Code)
	var listResp struct {
		Code int                            `json:"code"`
		Data []service.SupplierProviderType `json:"data"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &listResp))
	require.Len(t, listResp.Data, 1)

	rec = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodDelete, "/admin/supplier-management/provider-types/1", nil)
	router.ServeHTTP(rec, req)
	require.Equal(t, http.StatusOK, rec.Code)
	require.Empty(t, repo.items)
}

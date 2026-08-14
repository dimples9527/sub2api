package admin

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestAccountHandlerListModelSquareSyncAccountsUsesGroupEffectivePlatform(t *testing.T) {
	gin.SetMode(gin.TestMode)

	handler := NewAccountHandler(
		&stubAdminService{
			groups: []service.Group{
				{ID: 101, Name: "glm-group", Platform: service.PlatformOpenAI, Status: service.StatusActive},
				{ID: 102, Name: "openai-group", Platform: service.PlatformOpenAI, Status: service.StatusActive},
			},
			accounts: []service.Account{
				{ID: 11, Name: "glm-sync-account", Platform: service.PlatformOpenAI, Type: service.AccountTypeAPIKey, Status: service.StatusActive, GroupIDs: []int64{101}},
				{ID: 12, Name: "openai-account", Platform: service.PlatformOpenAI, Type: service.AccountTypeAPIKey, Status: service.StatusActive, GroupIDs: []int64{102}},
			},
		},
		nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil,
	)
	handler.SetMonitorGroupPlatformOverrideService(&monitorGroupPlatformOverrideHandlerStub{platforms: map[int64]string{101: "glm"}})

	router := gin.New()
	router.GET("/admin/upstream-management/model-square/sync-accounts", handler.ListModelSquareSyncAccounts)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/admin/upstream-management/model-square/sync-accounts?platform=glm", nil)
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	var body struct {
		Data []struct {
			ID                int64    `json:"id"`
			Name              string   `json:"name"`
			Platform          string   `json:"platform"`
			EffectivePlatform string   `json:"effective_platform"`
			GroupIDs          []int64  `json:"group_ids"`
			GroupNames        []string `json:"group_names"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
	require.Len(t, body.Data, 1)
	require.Equal(t, int64(11), body.Data[0].ID)
	require.Equal(t, "glm-sync-account", body.Data[0].Name)
	require.Equal(t, service.PlatformOpenAI, body.Data[0].Platform)
	require.Equal(t, "glm", body.Data[0].EffectivePlatform)
	require.Equal(t, []int64{101}, body.Data[0].GroupIDs)
	require.Equal(t, []string{"glm-group"}, body.Data[0].GroupNames)
}

func TestAccountHandlerListModelSquareSyncAccountsDoesNotUseRawPlatformForGroupedAccount(t *testing.T) {
	gin.SetMode(gin.TestMode)

	handler := NewAccountHandler(
		&stubAdminService{
			groups: []service.Group{
				{ID: 101, Name: "glm-group", Platform: service.PlatformOpenAI, Status: service.StatusActive},
			},
			accounts: []service.Account{
				{ID: 11, Name: "glm-account", Platform: service.PlatformOpenAI, Type: service.AccountTypeAPIKey, Status: service.StatusActive, GroupIDs: []int64{101}},
			},
		},
		nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil,
	)
	handler.SetMonitorGroupPlatformOverrideService(&monitorGroupPlatformOverrideHandlerStub{platforms: map[int64]string{101: "glm"}})

	router := gin.New()
	router.GET("/admin/upstream-management/model-square/sync-accounts", handler.ListModelSquareSyncAccounts)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/admin/upstream-management/model-square/sync-accounts?platform=openai", nil)
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	var body struct {
		Data []ModelSquareSyncAccountCandidate `json:"data"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
	require.Empty(t, body.Data)
}

func TestAccountHandlerListModelSquareSyncAccountsRejectsEmptyPlatform(t *testing.T) {
	gin.SetMode(gin.TestMode)
	handler := NewAccountHandler(&stubAdminService{}, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
	router := gin.New()
	router.GET("/admin/upstream-management/model-square/sync-accounts", handler.ListModelSquareSyncAccounts)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/admin/upstream-management/model-square/sync-accounts", nil)
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusBadRequest, rec.Code)
}

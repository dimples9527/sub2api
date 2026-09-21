package admin

import (
	"bytes"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func newBatchBindHandlerRouter(t *testing.T, stub *stubAdminService) *gin.Engine {
	t.Helper()
	gin.SetMode(gin.TestMode)
	handler := NewAccountHandler(stub, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)

	router := gin.New()
	// 供应商账号页的批量绑定是静态段，同页还有 /accounts/:id 系列通配段。
	// gin 在注册期就会为真正的路由冲突 panic，这里把邻居一起注册上钉死这件事。
	require.NotPanics(t, func() {
		router.POST("/accounts/batch-bind-groups", handler.BatchBindSupplierAccountGroups)
		router.POST("/accounts/batch-test", handler.SupplierBatchTest)
		router.GET("/accounts/batch-test/:job_id", handler.GetSupplierBatchTest)
		router.PUT("/accounts/:local_account_id/platform-override", func(c *gin.Context) {
			c.Status(http.StatusNoContent)
		})
		router.DELETE("/accounts/:id", func(c *gin.Context) { c.Status(http.StatusNoContent) })
	})
	return router
}

func postBatchBindGroups(router *gin.Engine, body string) *httptest.ResponseRecorder {
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/accounts/batch-bind-groups", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(rec, req)
	return rec
}

func TestBatchBindSupplierAccountGroupsRouteCoexistsWithAccountWildcards(t *testing.T) {
	stub := &stubAdminService{}
	router := newBatchBindHandlerRouter(t, stub)

	rec := postBatchBindGroups(router, `{"account_ids":[11,12],"group_ids":[7,8]}`)

	require.Equal(t, http.StatusOK, rec.Code)
	require.NotNil(t, stub.lastBatchBindGroupsInput)
	require.Equal(t, []int64{11, 12}, stub.lastBatchBindGroupsInput.AccountIDs)
	require.Equal(t, []int64{7, 8}, stub.lastBatchBindGroupsInput.GroupIDs)
	require.False(t, stub.lastBatchBindGroupsInput.SkipMixedChannelCheck)
}

func TestBatchBindSupplierAccountGroupsForwardsSkipMixedChannelCheck(t *testing.T) {
	stub := &stubAdminService{}
	router := newBatchBindHandlerRouter(t, stub)

	rec := postBatchBindGroups(router, `{"account_ids":[11],"group_ids":[7],"skip_mixed_channel_check":true}`)

	require.Equal(t, http.StatusOK, rec.Code)
	require.NotNil(t, stub.lastBatchBindGroupsInput)
	require.True(t, stub.lastBatchBindGroupsInput.SkipMixedChannelCheck)
}

func TestBatchBindSupplierAccountGroupsRejectsMissingFields(t *testing.T) {
	stub := &stubAdminService{}
	router := newBatchBindHandlerRouter(t, stub)

	// group_ids 缺失时必须在进 service 之前就被 binding 拦下，避免空分组被当成「清空绑定」。
	rec := postBatchBindGroups(router, `{"account_ids":[11]}`)

	require.Equal(t, http.StatusBadRequest, rec.Code)
	require.Nil(t, stub.lastBatchBindGroupsInput)
}

func TestBatchBindSupplierAccountGroupsReturnsServiceError(t *testing.T) {
	stub := &stubAdminService{batchBindGroupsErr: errors.New("boom")}
	router := newBatchBindHandlerRouter(t, stub)

	rec := postBatchBindGroups(router, `{"account_ids":[11],"group_ids":[7]}`)

	require.Equal(t, http.StatusInternalServerError, rec.Code)
}

func TestBatchBindSupplierAccountGroupsWithoutAdminService(t *testing.T) {
	gin.SetMode(gin.TestMode)
	handler := NewAccountHandler(nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)

	router := gin.New()
	router.POST("/accounts/batch-bind-groups", handler.BatchBindSupplierAccountGroups)

	rec := postBatchBindGroups(router, `{"account_ids":[11],"group_ids":[7]}`)
	require.Equal(t, http.StatusServiceUnavailable, rec.Code)
}

func TestBatchBindSupplierAccountGroupsSerializesPerAccountResults(t *testing.T) {
	stub := &stubAdminService{}
	router := newBatchBindHandlerRouter(t, stub)

	rec := postBatchBindGroups(router, `{"account_ids":[11,12],"group_ids":[7]}`)

	require.Equal(t, http.StatusOK, rec.Code)
	require.Contains(t, rec.Body.String(), `"bound":2`)
	require.Contains(t, rec.Body.String(), `"results":[]`)
}

var _ service.AdminService = (*stubAdminService)(nil)

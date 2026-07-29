package admin

import (
	"net/http"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
)

// SupplierBatchTestAccountsRequest 定义供应商账号页批量测试的独立请求参数。
type SupplierBatchTestAccountsRequest struct {
	AccountIDs            []int64           `json:"account_ids" binding:"required"`
	ModelID               string            `json:"model_id"`
	ModelIDsByPlatform    map[string]string `json:"model_ids_by_platform"`
	ModelIDsByAccount     map[int64]string  `json:"model_ids_by_account"`
	Concurrency           int               `json:"concurrency"`
	TimeoutPerAccountSecs int               `json:"timeout_per_account_seconds"`
	TimeoutSecs           int               `json:"timeout_seconds"`
}

// SupplierBatchTest 启动供应商账号页的批量连接测试任务。
// POST /api/v1/admin/supplier-management/accounts/batch-test
func (h *AccountHandler) SupplierBatchTest(c *gin.Context) {
	if h.accountTestService == nil {
		response.Error(c, http.StatusServiceUnavailable, "Account test service unavailable")
		return
	}

	var req SupplierBatchTestAccountsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request: "+err.Error())
		return
	}

	timeout := time.Duration(req.TimeoutPerAccountSecs) * time.Second
	if req.TimeoutPerAccountSecs <= 0 {
		timeout = defaultBatchAccountTestHTTPTimeout()
	}
	job, err := h.accountTestService.StartBatchTestAccounts(c.Request.Context(), service.BatchAccountTestInput{
		AccountIDs:         req.AccountIDs,
		ModelID:            req.ModelID,
		ModelIDsByPlatform: req.ModelIDsByPlatform,
		ModelIDsByAccount:  req.ModelIDsByAccount,
		Concurrency:        req.Concurrency,
		TimeoutPerAccount:  timeout,
		Timeout:            time.Duration(req.TimeoutSecs) * time.Second,
	})
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}

	response.Success(c, job)
}

// GetSupplierBatchTest 查询供应商账号页批量测试任务状态。
// GET /api/v1/admin/supplier-management/accounts/batch-test/:job_id
func (h *AccountHandler) GetSupplierBatchTest(c *gin.Context) {
	if h.accountTestService == nil {
		response.Error(c, http.StatusServiceUnavailable, "Account test service unavailable")
		return
	}

	job, err := h.accountTestService.GetBatchTestJob(c.Request.Context(), c.Param("job_id"))
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, job)
}

// CancelSupplierBatchTest 取消供应商账号页正在执行的批量测试任务。
// POST /api/v1/admin/supplier-management/accounts/batch-test/:job_id/cancel
func (h *AccountHandler) CancelSupplierBatchTest(c *gin.Context) {
	if h.accountTestService == nil {
		response.Error(c, http.StatusServiceUnavailable, "Account test service unavailable")
		return
	}

	job, err := h.accountTestService.CancelBatchTestJob(c.Request.Context(), c.Param("job_id"))
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, job)
}

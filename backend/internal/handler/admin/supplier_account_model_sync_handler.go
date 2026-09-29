package admin

import (
	"net/http"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
)

// 本文件按「供应商管理模块」业务代码管理（见 AGENTS.md）：批量同步上游模型的入口在
// 供应商账号页，与 supplier_account_batch_test_handler.go、
// supplier_account_group_binding_handler.go 同构 —— 方法挂在 AccountHandler 上，
// 但文件与路由都归供应商模块。批量编排复用的是框架既有的单账号同步能力
// （AccountTestService.SyncUpstreamModelCatalog），没有新增框架侧扩展点。

// SyncUpstreamModelsBatchRequest 批量同步上游模型的请求参数。
type SyncUpstreamModelsBatchRequest struct {
	AccountIDs            []int64 `json:"account_ids"`
	Mode                  string  `json:"mode"`
	Apply                 bool    `json:"apply"`
	Concurrency           int     `json:"concurrency"`
	TimeoutPerAccountSecs int     `json:"timeout_per_account_secs"`
	TimeoutSecs           int     `json:"timeout_secs"`
}

// SyncUpstreamModelsBatch 批量同步所选账号的上游模型。
//
// POST /api/v1/admin/supplier-management/accounts/models/sync-upstream/batch
//
// Apply=false 只返回「应用后会长什么样」，不写白名单；Apply=true 才落库。
func (h *AccountHandler) SyncUpstreamModelsBatch(c *gin.Context) {
	if h.accountTestService == nil {
		response.Error(c, http.StatusServiceUnavailable, "Account test service unavailable")
		return
	}

	var req SyncUpstreamModelsBatchRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request: "+err.Error())
		return
	}
	if len(req.AccountIDs) == 0 {
		response.BadRequest(c, "account_ids is required")
		return
	}

	result, err := h.accountTestService.SyncUpstreamModelCatalogBatch(c.Request.Context(), service.UpstreamModelBatchSyncInput{
		AccountIDs:        req.AccountIDs,
		Mode:              req.Mode,
		Apply:             req.Apply,
		Concurrency:       req.Concurrency,
		TimeoutPerAccount: time.Duration(req.TimeoutPerAccountSecs) * time.Second,
		Timeout:           time.Duration(req.TimeoutSecs) * time.Second,
	})
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}

	response.Success(c, result)
}

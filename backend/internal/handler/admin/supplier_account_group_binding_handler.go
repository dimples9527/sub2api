package admin

import (
	"net/http"

	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
)

// BatchBindSupplierAccountGroupsRequest 定义供应商账号页「批量绑定分组」的请求参数。
type BatchBindSupplierAccountGroupsRequest struct {
	AccountIDs []int64 `json:"account_ids" binding:"required"`
	GroupIDs   []int64 `json:"group_ids" binding:"required"`
	// SkipMixedChannelCheck 由前端在用户确认混合渠道风险后置位。
	SkipMixedChannelCheck bool `json:"skip_mixed_channel_check"`
}

// BatchBindSupplierAccountGroups 把所选分组追加绑定到所选账号上。
//
// POST /api/v1/admin/supplier-management/accounts/batch-bind-groups
//
// 走的是「并集」语义：账号原有分组保留，只补齐缺少的那些。供应商账号页的批量入口
// 是「让这批账号也加入这个分组」，替换语义会把它们从其它分组里摘掉。
func (h *AccountHandler) BatchBindSupplierAccountGroups(c *gin.Context) {
	if h.adminService == nil {
		response.Error(c, http.StatusServiceUnavailable, "Admin service unavailable")
		return
	}

	var req BatchBindSupplierAccountGroupsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request: "+err.Error())
		return
	}

	result, err := h.adminService.BatchBindAccountGroups(c.Request.Context(), &service.BatchBindAccountGroupsInput{
		AccountIDs:            req.AccountIDs,
		GroupIDs:              req.GroupIDs,
		SkipMixedChannelCheck: req.SkipMixedChannelCheck,
	})
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}

	response.Success(c, result)
}

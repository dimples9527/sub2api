package admin

import (
	"context"
	"strconv"
	"strings"

	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
)

type SupplierAutomationServicePort interface {
	ListTasks(ctx context.Context) ([]service.SupplierAutomationTask, error)
	UpdateTask(ctx context.Context, task *service.SupplierAutomationTask) error
	RunWithMode(ctx context.Context, taskCode, trigger, mode string) (service.SupplierAutomationRun, error)
	ListRuns(ctx context.Context, params service.SupplierAutomationRunListParams) (service.SupplierAutomationRunListResult, error)
	ListRateGuardChangeLogs(ctx context.Context, params service.SupplierRateGuardChangeLogListParams) (service.SupplierRateGuardChangeLogListResult, error)
	ListAccountRateGuardUnbindLogs(ctx context.Context, params service.SupplierAccountRateGuardUnbindLogListParams) (service.SupplierAccountRateGuardUnbindLogListResult, error)
	MarkRateGuardChangeLogHandled(ctx context.Context, id int64) (service.SupplierRateGuardChangeLog, error)
}

type SupplierAutomationHandler struct {
	service SupplierAutomationServicePort
}

func NewSupplierAutomationHandler(service SupplierAutomationServicePort) *SupplierAutomationHandler {
	return &SupplierAutomationHandler{service: service}
}

func (h *SupplierAutomationHandler) ListTasks(c *gin.Context) {
	tasks, err := h.service.ListTasks(c.Request.Context())
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, tasks)
}

func (h *SupplierAutomationHandler) UpdateTask(c *gin.Context) {
	taskCode := strings.TrimSpace(c.Param("task_code"))
	if taskCode == "" {
		response.ErrorFrom(c, badRequest("任务编码不能为空"))
		return
	}
	var req service.SupplierAutomationTask
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ErrorFrom(c, badRequest(err.Error()))
		return
	}
	req.TaskCode = taskCode
	if strings.TrimSpace(req.Name) == "" {
		req.Name = taskCode
	}
	if err := h.service.UpdateTask(c.Request.Context(), &req); err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, req)
}

type supplierAutomationRunRequest struct {
	Mode string `json:"mode"`
}

func (h *SupplierAutomationHandler) RunTask(c *gin.Context) {
	taskCode := strings.TrimSpace(c.Param("task_code"))
	if taskCode == "" {
		response.ErrorFrom(c, badRequest("任务编码不能为空"))
		return
	}
	req := supplierAutomationRunRequest{Mode: service.SupplierAutomationRunModeExecute}
	if c.Request.ContentLength != 0 {
		if err := c.ShouldBindJSON(&req); err != nil {
			response.ErrorFrom(c, badRequest(err.Error()))
			return
		}
	}
	req.Mode = strings.ToLower(strings.TrimSpace(req.Mode))
	if req.Mode == "" {
		req.Mode = service.SupplierAutomationRunModeExecute
	}
	if req.Mode != service.SupplierAutomationRunModePreview && req.Mode != service.SupplierAutomationRunModeExecute {
		response.ErrorFrom(c, badRequest("运行模式无效"))
		return
	}
	run, err := h.service.RunWithMode(c.Request.Context(), taskCode, service.SupplierSyncTriggerManual, req.Mode)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, run)
}

func (h *SupplierAutomationHandler) ListRuns(c *gin.Context) {
	page, pageSize := response.ParsePagination(c)
	if pageSize > supplierProviderMaxPageSize {
		pageSize = supplierProviderMaxPageSize
	}
	result, err := h.service.ListRuns(c.Request.Context(), service.SupplierAutomationRunListParams{
		TaskCode: strings.TrimSpace(c.Query("task_code")),
		Status:   strings.TrimSpace(c.Query("status")),
		Page:     page,
		PageSize: pageSize,
	})
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, result)
}

func (h *SupplierAutomationHandler) ListRateGuardChangeLogs(c *gin.Context) {
	page, pageSize := response.ParsePagination(c)
	if pageSize > supplierProviderMaxPageSize {
		pageSize = supplierProviderMaxPageSize
	}
	result, err := h.service.ListRateGuardChangeLogs(c.Request.Context(), service.SupplierRateGuardChangeLogListParams{
		Page: page, PageSize: pageSize,
	})
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, result)
}

func (h *SupplierAutomationHandler) ListAccountRateGuardUnbindLogs(c *gin.Context) {
	page, pageSize := response.ParsePagination(c)
	if pageSize > supplierProviderMaxPageSize {
		pageSize = supplierProviderMaxPageSize
	}
	result, err := h.service.ListAccountRateGuardUnbindLogs(c.Request.Context(), service.SupplierAccountRateGuardUnbindLogListParams{
		RunID:          parseOptionalInt64(c.Query("run_id")),
		ProviderID:     parseOptionalInt64(c.Query("provider_id")),
		LocalAccountID: parseOptionalInt64(c.Query("local_account_id")),
		Search:         strings.TrimSpace(c.Query("search")),
		Result:         strings.TrimSpace(c.Query("result")),
		Page:           page,
		PageSize:       pageSize,
	})
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, result)
}

func (h *SupplierAutomationHandler) MarkRateGuardChangeLogHandled(c *gin.Context) {
	id, err := strconv.ParseInt(strings.TrimSpace(c.Param("id")), 10, 64)
	if err != nil || id <= 0 {
		response.ErrorFrom(c, badRequest("日志编号无效"))
		return
	}
	item, err := h.service.MarkRateGuardChangeLogHandled(c.Request.Context(), id)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, item)
}

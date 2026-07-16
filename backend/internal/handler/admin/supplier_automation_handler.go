package admin

import (
	"context"
	"strings"

	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
)

type SupplierAutomationServicePort interface {
	ListTasks(ctx context.Context) ([]service.SupplierAutomationTask, error)
	UpdateTask(ctx context.Context, task *service.SupplierAutomationTask) error
	Run(ctx context.Context, taskCode, trigger string) (service.SupplierAutomationRun, error)
	ListRuns(ctx context.Context, params service.SupplierAutomationRunListParams) (service.SupplierAutomationRunListResult, error)
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

func (h *SupplierAutomationHandler) RunTask(c *gin.Context) {
	taskCode := strings.TrimSpace(c.Param("task_code"))
	if taskCode == "" {
		response.ErrorFrom(c, badRequest("任务编码不能为空"))
		return
	}
	run, err := h.service.Run(c.Request.Context(), taskCode, service.SupplierSyncTriggerManual)
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

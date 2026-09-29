package admin

import (
	"context"
	"strconv"
	"strings"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	"github.com/Wei-Shaw/sub2api/internal/pkg/timezone"
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
	MarkAccountRateGuardUnbindLogHandled(ctx context.Context, id int64) (service.SupplierAccountRateGuardUnbindLog, error)
	MarkAccountRateGuardUnbindLogsHandled(ctx context.Context, params service.SupplierAccountRateGuardUnbindLogListParams) (service.SupplierAccountRateGuardUnbindLogBatchHandledResult, error)
	ListGroupSchedulingElectionChangeLogs(ctx context.Context, params service.SupplierGroupSchedulingElectionChangeLogListParams) (service.SupplierGroupSchedulingElectionChangeLogListResult, error)
	BuildGroupSchedulingElectionDiagnostics(ctx context.Context, runLimit int) (string, error)
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
		Mode:           strings.TrimSpace(c.Query("mode")),
		Status:         strings.TrimSpace(c.Query("status")),
		OnlyUnbound:    parseOptionalBool(c.Query("only_unbound")),
		Page:           page,
		PageSize:       pageSize,
	})
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, result)
}

// parseOptionalDayBoundary 把 YYYY-MM-DD 解析成时间范围的边界。
// 结束日要 +24h：用户选「到 9 月 20 日」时想包含 20 日当天，
// 直接用当天 00:00 做上界会把整日都排除掉 —— 这是日期区间筛选最常见的静默偏差。
func parseOptionalDayBoundary(c *gin.Context, name string, endOfDay bool) *time.Time {
	raw := strings.TrimSpace(c.Query(name))
	if raw == "" {
		return nil
	}
	parsed, err := timezone.ParseInUserLocation("2006-01-02", raw, c.Query("timezone"))
	if err != nil {
		return nil
	}
	if endOfDay {
		parsed = parsed.Add(24 * time.Hour)
	}
	return &parsed
}

func (h *SupplierAutomationHandler) ListGroupSchedulingElectionChangeLogs(c *gin.Context) {
	page, pageSize := response.ParsePagination(c)
	if pageSize > supplierProviderMaxPageSize {
		pageSize = supplierProviderMaxPageSize
	}
	result, err := h.service.ListGroupSchedulingElectionChangeLogs(c.Request.Context(), service.SupplierGroupSchedulingElectionChangeLogListParams{
		GroupID:     parseOptionalInt64(c.Query("group_id")),
		AccountID:   parseOptionalInt64(c.Query("account_id")),
		RunIDs:      parseOptionalInt64List(c.Query("run_ids")),
		Search:      strings.TrimSpace(c.Query("search")),
		Platform:    strings.TrimSpace(c.Query("platform")),
		Direction:   strings.TrimSpace(c.Query("direction")),
		StartedFrom: parseOptionalDayBoundary(c, "started_from", false),
		StartedTo:   parseOptionalDayBoundary(c, "started_to", true),
		// include_skipped=true 时把「本该动却没动」的记录也带上（在任者锁定 / 无备选 / 待观察 / 写库失败）。
		IncludeSkipped: parseOptionalBool(c.Query("include_skipped")),
		Page:           page,
		PageSize:       pageSize,
	})
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, result)
}

// GetGroupSchedulingElectionDiagnostics 导出「分组择优调度」的诊断快照文本。
//
// 存在的理由：这个任务的业务执行过程一行日志都不打，排查「某组为什么开着多个账号」
// 只能翻 result_detail，而那里的 items 又不含「没发生变化的账号」——
// 「某组当前开着几个」这个数只存在于 accounts.schedulable。这里把它连同配置与最近几轮决策
// 一次性拼好，管理员点一下就能拷走，不必再手写 SQL。
// 返回 JSON 而不是 text/plain，是为了跟本模块其它接口共用同一套响应包装与错误处理。
func (h *SupplierAutomationHandler) GetGroupSchedulingElectionDiagnostics(c *gin.Context) {
	runLimit := 0
	if raw := strings.TrimSpace(c.Query("run_limit")); raw != "" {
		parsed, err := strconv.Atoi(raw)
		if err != nil || parsed <= 0 {
			response.ErrorFrom(c, badRequest("回溯轮数必须是正整数"))
			return
		}
		runLimit = parsed
	}
	text, err := h.service.BuildGroupSchedulingElectionDiagnostics(c.Request.Context(), runLimit)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, gin.H{
		"generated_at": time.Now(),
		"text":         text,
	})
}

func (h *SupplierAutomationHandler) MarkAccountRateGuardUnbindLogHandled(c *gin.Context) {
	id, err := strconv.ParseInt(strings.TrimSpace(c.Param("id")), 10, 64)
	if err != nil || id <= 0 {
		response.ErrorFrom(c, badRequest("日志编号无效"))
		return
	}
	item, err := h.service.MarkAccountRateGuardUnbindLogHandled(c.Request.Context(), id)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, item)
}

// MarkAccountRateGuardUnbindLogsHandled 一键处理：把「当前筛选条件下的全部待处理」标记为已处理。
// 筛选走 query 而不是 body，与列表接口共用同一套参数名 —— 前端不必做两套映射，
// 也不会出现"列表带了这个筛选、批量忘了带"的口径不一致。
// Page/PageSize 有意不解析：批量不分页，条数上限由仓库层的 batchLimit 控制。
func (h *SupplierAutomationHandler) MarkAccountRateGuardUnbindLogsHandled(c *gin.Context) {
	result, err := h.service.MarkAccountRateGuardUnbindLogsHandled(c.Request.Context(), service.SupplierAccountRateGuardUnbindLogListParams{
		RunID:          parseOptionalInt64(c.Query("run_id")),
		ProviderID:     parseOptionalInt64(c.Query("provider_id")),
		LocalAccountID: parseOptionalInt64(c.Query("local_account_id")),
		Search:         strings.TrimSpace(c.Query("search")),
		Result:         strings.TrimSpace(c.Query("result")),
		Mode:           strings.TrimSpace(c.Query("mode")),
		Status:         strings.TrimSpace(c.Query("status")),
		OnlyUnbound:    parseOptionalBool(c.Query("only_unbound")),
	})
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, result)
}

func (h *SupplierAutomationHandler) MarkRateGuardChangeLogHandled(c *gin.Context) {	id, err := strconv.ParseInt(strings.TrimSpace(c.Param("id")), 10, 64)
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

func parseOptionalBool(raw string) bool {
	value, err := strconv.ParseBool(strings.TrimSpace(raw))
	return err == nil && value
}

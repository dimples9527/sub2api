package admin

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

type supplierAutomationHandlerServiceStub struct {
	updated                   bool
	ranCode                   string
	ranMode                   string
	handledID                 int64
	accountRateGuardHandledID int64
	accountRateGuardLogParams service.SupplierAccountRateGuardUnbindLogListParams
	batchCalled               bool
	batchParams               service.SupplierAccountRateGuardUnbindLogListParams
	groupElectionLogParams    service.SupplierGroupSchedulingElectionChangeLogListParams
}

func (s *supplierAutomationHandlerServiceStub) ListTasks(context.Context) ([]service.SupplierAutomationTask, error) {
	return []service.SupplierAutomationTask{{TaskCode: service.SupplierAutomationTaskSync, Name: "同步", Enabled: true, CronExpression: "*/15 * * * *", TimeoutSeconds: 600}}, nil
}
func (s *supplierAutomationHandlerServiceStub) UpdateTask(_ context.Context, task *service.SupplierAutomationTask) error {
	s.updated = true
	return nil
}
func (s *supplierAutomationHandlerServiceStub) Run(_ context.Context, taskCode, trigger string) (service.SupplierAutomationRun, error) {
	s.ranCode = taskCode
	now := time.Now()
	return service.SupplierAutomationRun{TaskCode: taskCode, TriggerSource: trigger, Status: service.SupplierAutomationStatusSuccess, StartedAt: now, FinishedAt: &now}, nil
}

func (s *supplierAutomationHandlerServiceStub) RunWithMode(_ context.Context, taskCode, trigger, mode string) (service.SupplierAutomationRun, error) {
	s.ranMode = mode
	return s.Run(context.Background(), taskCode, trigger)
}
func (s *supplierAutomationHandlerServiceStub) ListRuns(context.Context, service.SupplierAutomationRunListParams) (service.SupplierAutomationRunListResult, error) {
	return service.SupplierAutomationRunListResult{Items: []service.SupplierAutomationRun{{ID: 1, TaskCode: service.SupplierAutomationTaskSync, Status: service.SupplierAutomationStatusSuccess}}, Total: 1, Page: 1, PageSize: 20}, nil
}
func (s *supplierAutomationHandlerServiceStub) ListRateGuardChangeLogs(context.Context, service.SupplierRateGuardChangeLogListParams) (service.SupplierRateGuardChangeLogListResult, error) {
	return service.SupplierRateGuardChangeLogListResult{
		Items: []service.SupplierRateGuardChangeLog{{ID: 9, Status: service.SupplierRateGuardChangeLogStatusPending}},
		Total: 1, PendingCount: 1, Page: 1, PageSize: 20,
	}, nil
}
func (s *supplierAutomationHandlerServiceStub) ListAccountRateGuardUnbindLogs(_ context.Context, params service.SupplierAccountRateGuardUnbindLogListParams) (service.SupplierAccountRateGuardUnbindLogListResult, error) {
	s.accountRateGuardLogParams = params
	return service.SupplierAccountRateGuardUnbindLogListResult{
		Items: []service.SupplierAccountRateGuardUnbindLog{{ID: 11, Result: service.SupplierAccountRateGuardLogResultUnbound}},
		Total: 1, Page: params.Page, PageSize: params.PageSize,
	}, nil
}
func (s *supplierAutomationHandlerServiceStub) MarkRateGuardChangeLogHandled(_ context.Context, id int64) (service.SupplierRateGuardChangeLog, error) {
	s.handledID = id
	return service.SupplierRateGuardChangeLog{ID: id, Status: service.SupplierRateGuardChangeLogStatusHandled}, nil
}
func (s *supplierAutomationHandlerServiceStub) MarkAccountRateGuardUnbindLogHandled(_ context.Context, id int64) (service.SupplierAccountRateGuardUnbindLog, error) {
	s.accountRateGuardHandledID = id
	return service.SupplierAccountRateGuardUnbindLog{ID: id, Status: service.SupplierAccountRateGuardLogStatusHandled}, nil
}

func (s *supplierAutomationHandlerServiceStub) MarkAccountRateGuardUnbindLogsHandled(_ context.Context, params service.SupplierAccountRateGuardUnbindLogListParams) (service.SupplierAccountRateGuardUnbindLogBatchHandledResult, error) {
	s.batchCalled = true
	s.batchParams = params
	return service.SupplierAccountRateGuardUnbindLogBatchHandledResult{Handled: 3, Batch: 500, HasMore: false}, nil
}

func (s *supplierAutomationHandlerServiceStub) ListGroupSchedulingElectionChangeLogs(_ context.Context, params service.SupplierGroupSchedulingElectionChangeLogListParams) (service.SupplierGroupSchedulingElectionChangeLogListResult, error) {
	s.groupElectionLogParams = params
	return service.SupplierGroupSchedulingElectionChangeLogListResult{
		Items: []service.SupplierGroupSchedulingElectionChangeLog{{
			RunID: 5, AccountID: 8, Direction: service.SupplierGroupSchedulingElectionChangeDirectionDisabled,
			SchedulableBefore: true, SchedulableAfter: false,
		}},
		Total: 1, Page: params.Page, PageSize: params.PageSize,
	}, nil
}

func TestSupplierAutomationHandlerRoutes(t *testing.T) {
	gin.SetMode(gin.TestMode)
	stub := &supplierAutomationHandlerServiceStub{}
	handler := NewSupplierAutomationHandler(stub)
	router := gin.New()
	router.GET("/automation/tasks", handler.ListTasks)
	router.PUT("/automation/tasks/:task_code", handler.UpdateTask)
	router.POST("/automation/tasks/:task_code/run", handler.RunTask)
	router.GET("/automation/runs", handler.ListRuns)
	router.GET("/automation/rate-guard-change-logs", handler.ListRateGuardChangeLogs)
	router.GET("/automation/account-rate-guard-unbind-logs", handler.ListAccountRateGuardUnbindLogs)
	router.POST("/automation/account-rate-guard-unbind-logs/:id/handled", handler.MarkAccountRateGuardUnbindLogHandled)
	router.POST("/automation/account-rate-guard-unbind-logs/handled-batch", handler.MarkAccountRateGuardUnbindLogsHandled)
	router.POST("/automation/rate-guard-change-logs/:id/handled", handler.MarkRateGuardChangeLogHandled)
	router.GET("/automation/group-election-change-logs", handler.ListGroupSchedulingElectionChangeLogs)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/automation/tasks", nil)
	router.ServeHTTP(rec, req)
	require.Equal(t, http.StatusOK, rec.Code)

	rec = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodGet, "/automation/account-rate-guard-unbind-logs?run_id=3&provider_id=4&local_account_id=5&search=alpha&mode=execute&result=failed&status=pending&only_unbound=true&page=2&page_size=30", nil)
	router.ServeHTTP(rec, req)
	require.Equal(t, http.StatusOK, rec.Code)
	require.Equal(t, service.SupplierAccountRateGuardUnbindLogListParams{
		RunID: 3, ProviderID: 4, LocalAccountID: 5, Search: "alpha", Mode: "execute", Result: "failed",
		Status: "pending", OnlyUnbound: true, Page: 2, PageSize: 30,
	}, stub.accountRateGuardLogParams)

	rec = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodPost, "/automation/account-rate-guard-unbind-logs/11/handled", nil)
	router.ServeHTTP(rec, req)
	require.Equal(t, http.StatusOK, rec.Code)
	require.Equal(t, int64(11), stub.accountRateGuardHandledID)

	// 一键处理：筛选走 query，与列表同名；Page/PageSize 有意不解析（批量不分页）。
	rec = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodPost, "/automation/account-rate-guard-unbind-logs/handled-batch?provider_id=4&local_account_id=5&search=alpha&mode=execute&only_unbound=true", nil)
	router.ServeHTTP(rec, req)
	require.Equal(t, http.StatusOK, rec.Code)
	require.True(t, stub.batchCalled)
	require.Equal(t, service.SupplierAccountRateGuardUnbindLogListParams{
		ProviderID: 4, LocalAccountID: 5, Search: "alpha", Mode: "execute", OnlyUnbound: true,
	}, stub.batchParams)

	rec = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodPut, "/automation/tasks/supplier_data_sync", bytes.NewBufferString(`{"enabled":true,"cron_expression":"*/30 * * * *","timeout_seconds":600}`))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(rec, req)
	require.Equal(t, http.StatusOK, rec.Code)
	require.True(t, stub.updated)

	rec = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodPost, "/automation/tasks/supplier_data_sync/run", nil)
	router.ServeHTTP(rec, req)
	require.Equal(t, http.StatusOK, rec.Code)
	require.Equal(t, service.SupplierAutomationTaskSync, stub.ranCode)

	rec = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodGet, "/automation/runs?task_code=supplier_data_sync", nil)
	router.ServeHTTP(rec, req)
	require.Equal(t, http.StatusOK, rec.Code)

	rec = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodGet, "/automation/rate-guard-change-logs?page=1&page_size=20", nil)
	router.ServeHTTP(rec, req)
	require.Equal(t, http.StatusOK, rec.Code)

	rec = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodPost, "/automation/rate-guard-change-logs/9/handled", nil)
	router.ServeHTTP(rec, req)
	require.Equal(t, http.StatusOK, rec.Code)
	require.Equal(t, int64(9), stub.handledID)

	rec = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodGet, "/automation/group-election-change-logs?group_id=7&account_id=8&run_ids=5001,5002&platform=anthropic&search=alpha&direction=disabled&started_from=2026-09-01&started_to=2026-09-20&page=2&page_size=30", nil)
	router.ServeHTTP(rec, req)
	require.Equal(t, http.StatusOK, rec.Code)
	require.Equal(t, int64(7), stub.groupElectionLogParams.GroupID)
	require.Equal(t, int64(8), stub.groupElectionLogParams.AccountID)
	// 批次是多选：逗号分隔解析成列表，顺序照传（前端按点选顺序拼）。
	require.Equal(t, []int64{5001, 5002}, stub.groupElectionLogParams.RunIDs)
	require.Equal(t, "anthropic", stub.groupElectionLogParams.Platform)
	require.Equal(t, "alpha", stub.groupElectionLogParams.Search)
	require.Equal(t, service.SupplierGroupSchedulingElectionChangeDirectionDisabled, stub.groupElectionLogParams.Direction)
	require.Equal(t, 2, stub.groupElectionLogParams.Page)
	require.Equal(t, 30, stub.groupElectionLogParams.PageSize)
	// 结束日必须 +24h：选「到 9/20」要包含 20 日当天，否则整天都被排除。
	// 用区间长度断言而不是具体时刻，避免测试结果依赖运行环境的时区。
	require.NotNil(t, stub.groupElectionLogParams.StartedFrom)
	require.NotNil(t, stub.groupElectionLogParams.StartedTo)
	require.Equal(t, 20*24*time.Hour, stub.groupElectionLogParams.StartedTo.Sub(*stub.groupElectionLogParams.StartedFrom))
}

// 一键处理的路径少一段（没有 :id），与单条处理的 /:id/handled 并存。
// gin 对「同层级既有通配段又有静态段」是允许的，但一旦把路径改回带 :id 的形式，
// 冲突会在**服务启动注册路由时 panic** —— 那是最难排查的一类故障，
// 因此在测试里把它钉死：注册不 panic，且两条路径各自命中正确的处理器。
func TestSupplierAutomationHandlerBatchRouteDoesNotConflict(t *testing.T) {
	gin.SetMode(gin.TestMode)
	stub := &supplierAutomationHandlerServiceStub{}
	handler := NewSupplierAutomationHandler(stub)
	router := gin.New()
	require.NotPanics(t, func() {
		router.POST("/automation/account-rate-guard-unbind-logs/:id/handled", handler.MarkAccountRateGuardUnbindLogHandled)
		router.POST("/automation/account-rate-guard-unbind-logs/handled-batch", handler.MarkAccountRateGuardUnbindLogsHandled)
	})

	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/automation/account-rate-guard-unbind-logs/77/handled", nil))
	require.Equal(t, http.StatusOK, rec.Code)
	require.Equal(t, int64(77), stub.accountRateGuardHandledID)
	require.False(t, stub.batchCalled)

	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/automation/account-rate-guard-unbind-logs/handled-batch", nil))
	require.Equal(t, http.StatusOK, rec.Code)
	require.True(t, stub.batchCalled)
}

func TestSupplierAutomationHandlerPassesRunMode(t *testing.T) {	gin.SetMode(gin.TestMode)
	stub := &supplierAutomationHandlerServiceStub{}
	handler := NewSupplierAutomationHandler(stub)
	router := gin.New()
	router.POST("/automation/tasks/:task_code/run", handler.RunTask)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/automation/tasks/supplier_account_rate_guard/run", bytes.NewBufferString(`{"mode":"preview"}`))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	require.Equal(t, service.SupplierAutomationRunModePreview, stub.ranMode)
}

func TestSupplierAutomationHandlerRejectsInvalidRunMode(t *testing.T) {
	gin.SetMode(gin.TestMode)
	handler := NewSupplierAutomationHandler(&supplierAutomationHandlerServiceStub{})
	router := gin.New()
	router.POST("/automation/tasks/:task_code/run", handler.RunTask)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/automation/tasks/supplier_account_rate_guard/run", bytes.NewBufferString(`{"mode":"invalid"}`))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusBadRequest, rec.Code)
}
func TestSupplierAutomationHandlerRejectsBadJSON(t *testing.T) {
	gin.SetMode(gin.TestMode)
	handler := NewSupplierAutomationHandler(&supplierAutomationHandlerServiceStub{})
	router := gin.New()
	router.PUT("/automation/tasks/:task_code", handler.UpdateTask)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPut, "/automation/tasks/supplier_data_sync", bytes.NewBufferString(`{bad`))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusBadRequest, rec.Code)
}

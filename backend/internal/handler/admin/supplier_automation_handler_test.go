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
	accountRateGuardLogParams service.SupplierAccountRateGuardUnbindLogListParams
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
	router.POST("/automation/rate-guard-change-logs/:id/handled", handler.MarkRateGuardChangeLogHandled)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/automation/tasks", nil)
	router.ServeHTTP(rec, req)
	require.Equal(t, http.StatusOK, rec.Code)

	rec = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodGet, "/automation/account-rate-guard-unbind-logs?run_id=3&provider_id=4&local_account_id=5&search=alpha&result=failed&page=2&page_size=30", nil)
	router.ServeHTTP(rec, req)
	require.Equal(t, http.StatusOK, rec.Code)
	require.Equal(t, service.SupplierAccountRateGuardUnbindLogListParams{
		RunID: 3, ProviderID: 4, LocalAccountID: 5, Search: "alpha", Result: "failed", Page: 2, PageSize: 30,
	}, stub.accountRateGuardLogParams)

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
}

func TestSupplierAutomationHandlerPassesRunMode(t *testing.T) {
	gin.SetMode(gin.TestMode)
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

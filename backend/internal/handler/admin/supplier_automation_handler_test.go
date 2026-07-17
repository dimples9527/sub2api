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
	updated bool
	ranCode string
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
func (s *supplierAutomationHandlerServiceStub) ListRuns(context.Context, service.SupplierAutomationRunListParams) (service.SupplierAutomationRunListResult, error) {
	return service.SupplierAutomationRunListResult{Items: []service.SupplierAutomationRun{{ID: 1, TaskCode: service.SupplierAutomationTaskSync, Status: service.SupplierAutomationStatusSuccess}}, Total: 1, Page: 1, PageSize: 20}, nil
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

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/automation/tasks", nil)
	router.ServeHTTP(rec, req)
	require.Equal(t, http.StatusOK, rec.Code)

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

package admin

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/require"
)

func TestSupplierBalanceAlertHandlerDeletesEvent(t *testing.T) {
	gin.SetMode(gin.TestMode)
	repo := &supplierBalanceAlertDeleteRepoStub{}
	handler := NewSupplierBalanceAlertHandler(service.NewSupplierBalanceAlertService(repo, nil, nil))
	router := gin.New()
	router.DELETE("/balance-alert/events/:id", handler.DeleteEvent)

	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, httptest.NewRequest(http.MethodDelete, "/balance-alert/events/12", nil))

	require.Equal(t, http.StatusOK, recorder.Code)
	require.Contains(t, recorder.Body.String(), `"event_id":12`)
	require.Equal(t, int64(12), repo.deletedID)
}

func TestSupplierBalanceAlertHandlerRejectsActiveEventDeletion(t *testing.T) {
	gin.SetMode(gin.TestMode)
	repo := &supplierBalanceAlertDeleteRepoStub{deleteErr: service.ErrSupplierBalanceAlertEventActive}
	handler := NewSupplierBalanceAlertHandler(service.NewSupplierBalanceAlertService(repo, nil, nil))
	router := gin.New()
	router.DELETE("/balance-alert/events/:id", handler.DeleteEvent)

	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, httptest.NewRequest(http.MethodDelete, "/balance-alert/events/12", nil))

	require.Equal(t, http.StatusConflict, recorder.Code)
	require.Contains(t, recorder.Body.String(), "请等待余额恢复后再删除")
}

type supplierBalanceAlertDeleteRepoStub struct {
	deletedID int64
	deleteErr error
}

func (r *supplierBalanceAlertDeleteRepoStub) ListConfigs(context.Context, int64) ([]service.SupplierBalanceAlertConfig, error) {
	return nil, nil
}

func (r *supplierBalanceAlertDeleteRepoStub) GetConfig(context.Context, int64) (*service.SupplierBalanceAlertConfig, error) {
	return nil, nil
}

func (r *supplierBalanceAlertDeleteRepoStub) UpsertConfig(context.Context, int64, bool, decimal.Decimal, int) (*service.SupplierBalanceAlertConfig, error) {
	return nil, nil
}

func (r *supplierBalanceAlertDeleteRepoStub) UpdateScanState(context.Context, int64, time.Time, *decimal.Decimal, string, string) error {
	return nil
}

func (r *supplierBalanceAlertDeleteRepoStub) GetActiveLowEvent(context.Context, int64) (*service.SupplierBalanceAlertEvent, error) {
	return nil, nil
}

func (r *supplierBalanceAlertDeleteRepoStub) CreateEvent(context.Context, *service.SupplierBalanceAlertEvent) error {
	return nil
}

func (r *supplierBalanceAlertDeleteRepoStub) TouchActiveLowEvent(context.Context, int64, decimal.Decimal, time.Time) error {
	return nil
}

func (r *supplierBalanceAlertDeleteRepoStub) ResolveActiveLowEvent(context.Context, int64, time.Time, decimal.Decimal) error {
	return nil
}

func (r *supplierBalanceAlertDeleteRepoStub) ListEvents(context.Context, service.SupplierBalanceAlertEventListParams) (service.SupplierBalanceAlertEventListResult, error) {
	return service.SupplierBalanceAlertEventListResult{}, nil
}

func (r *supplierBalanceAlertDeleteRepoStub) DeleteEvent(_ context.Context, eventID int64) error {
	r.deletedID = eventID
	return r.deleteErr
}

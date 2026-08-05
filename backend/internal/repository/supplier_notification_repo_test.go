package repository

import (
	"context"
	"regexp"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/stretchr/testify/require"
)

func TestSupplierNotificationRepositoryListDeliveriesScansPayloadJSON(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	now := time.Date(2026, time.August, 5, 12, 0, 0, 0, time.UTC)
	mock.ExpectQuery(regexp.QuoteMeta("SELECT COUNT(*) FROM supplier_notification_deliveries d WHERE 1 = 1")).WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(int64(1)))
	mock.ExpectQuery(`SELECT d\.id, d\.channel_id, c\.name, d\.event_id`).WithArgs(50, 0).WillReturnRows(sqlmock.NewRows([]string{
		"id", "channel_id", "channel_name", "event_id", "provider_id", "provider_name", "event_type",
		"status", "payload_json", "attempt_count", "next_attempt_at", "last_error", "sent_at", "created_at", "updated_at",
	}).AddRow(
		int64(11), int64(1), "channel", nil, int64(10), "provider", "balance_low", "pending",
		[]byte(`{"provider_id":10}`), 0, now, "", nil, now, now,
	))

	repo := &supplierNotificationRepository{db: db}
	result, err := repo.ListDeliveries(context.Background(), service.SupplierNotificationDeliveryListParams{})
	require.NoError(t, err)
	require.Equal(t, int64(1), result.Total)
	require.Len(t, result.Items, 1)
	require.Equal(t, int64(11), result.Items[0].ID)
	require.Nil(t, result.Items[0].EventID)
	require.Nil(t, result.Items[0].SentAt)
	require.NoError(t, mock.ExpectationsWereMet())
}

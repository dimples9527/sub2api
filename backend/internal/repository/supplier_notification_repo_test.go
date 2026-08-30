package repository

import (
	"context"
	"database/sql/driver"
	"encoding/json"
	"regexp"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/stretchr/testify/require"
)

type supplierNotificationJSONArgument struct {
	check func([]byte) bool
}

func (a supplierNotificationJSONArgument) Match(value driver.Value) bool {
	var raw []byte
	switch value := value.(type) {
	case string:
		raw = []byte(value)
	case []byte:
		raw = value
	default:
		return false
	}
	return a.check(raw)
}

func TestSupplierNotificationRepositoryCreateGroupChangeEventStoresPayload(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	observedAt := time.Date(2026, time.August, 30, 12, 0, 0, 0, time.UTC)
	createdAt := observedAt.Add(time.Minute)
	syncRunID := int64(77)
	event := &service.SupplierGroupChangeEvent{
		ProviderID:   10,
		ProviderCode: "provider-a",
		ProviderName: "供应商甲",
		SyncRunID:    &syncRunID,
		Changes:      SupplierProviderGroupChangeSummaryForRepositoryTest(),
		ObservedAt:   observedAt,
		CreatedAt:    createdAt,
	}

	payloadMatcher := supplierNotificationJSONArgument{check: func(raw []byte) bool {
		var payload supplierGroupChangeEventPayload
		if err := json.Unmarshal(raw, &payload); err != nil {
			return false
		}
		return payload.ProviderID == event.ProviderID &&
			payload.ProviderCode == event.ProviderCode &&
			payload.ProviderName == event.ProviderName &&
			payload.EventType == service.SupplierGroupChangeEventType &&
			payload.ChangeCount == event.Changes.Count() &&
			payload.ObservedAt.Equal(event.ObservedAt) &&
			len(payload.Changes.Added) == 1 &&
			payload.Changes.Added[0].UpstreamKey == "vip-new"
	}}
	mock.ExpectQuery(regexp.QuoteMeta("INSERT INTO supplier_group_change_events")).
		WithArgs(int64(10), "provider-a", "供应商甲", syncRunID, service.SupplierGroupChangeEventType,
			observedAt, int64(1), payloadMatcher, createdAt).
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(int64(901)))

	repo := &supplierNotificationRepository{db: db}
	require.NoError(t, repo.CreateGroupChangeEvent(context.Background(), event))
	require.Equal(t, int64(901), event.ID)
	require.Equal(t, service.SupplierGroupChangeEventType, event.EventType)
	require.Equal(t, event.Changes.Count(), event.ChangeCount)
	require.NoError(t, mock.ExpectationsWereMet())
}

func SupplierProviderGroupChangeSummaryForRepositoryTest() service.SupplierProviderGroupChangeSummary {
	return service.SupplierProviderGroupChangeSummary{
		Added: []service.SupplierProviderGroupChange{{
			ChangeType:        service.SupplierProviderGroupChangeTypeAdded,
			UpstreamKey:       "vip-new",
			NewName:           "新分组",
			NewRateMultiplier: 1.2,
		}},
	}
}

func TestSupplierNotificationRepositoryCreateGroupChangeEventRejectsInvalidInput(t *testing.T) {
	cases := []struct {
		name  string
		event *service.SupplierGroupChangeEvent
	}{
		{name: "nil event"},
		{name: "missing provider", event: &service.SupplierGroupChangeEvent{Changes: SupplierProviderGroupChangeSummaryForRepositoryTest()}},
		{name: "empty changes", event: &service.SupplierGroupChangeEvent{ProviderID: 10}},
		{name: "invalid event type", event: &service.SupplierGroupChangeEvent{
			ProviderID: 10,
			EventType:  service.SupplierBalanceAlertEventLow,
			Changes:    SupplierProviderGroupChangeSummaryForRepositoryTest(),
		}},
	}

	repo := &supplierNotificationRepository{}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			require.ErrorIs(t, repo.CreateGroupChangeEvent(context.Background(), tc.event), service.ErrSupplierNotificationInvalid)
		})
	}
}

func TestSupplierNotificationRepositoryCreateDeliveryStoresGroupChangeEventID(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	now := time.Date(2026, time.August, 30, 12, 0, 0, 0, time.UTC)
	groupChangeEventID := int64(901)
	delivery := &service.SupplierNotificationDeliveryRecord{
		ChannelID:          1,
		GroupChangeEventID: &groupChangeEventID,
		ProviderID:         10,
		EventType:          service.SupplierGroupChangeEventType,
		PayloadJSON:        []byte(`{"event_type":"group_changed"}`),
		NextAttemptAt:      now,
	}
	mock.ExpectQuery(regexp.QuoteMeta("INSERT INTO supplier_notification_deliveries")).
		WithArgs(int64(1), nil, groupChangeEventID, int64(10), service.SupplierGroupChangeEventType,
			service.SupplierNotificationDeliveryPending, string(delivery.PayloadJSON), int64(0), now, "", nil).
		WillReturnRows(sqlmock.NewRows([]string{"id", "created_at", "updated_at"}).AddRow(int64(902), now, now))

	repo := &supplierNotificationRepository{db: db}
	require.NoError(t, repo.CreateDelivery(context.Background(), delivery))
	require.Equal(t, int64(902), delivery.ID)
	require.Nil(t, delivery.EventID)
	require.NotNil(t, delivery.GroupChangeEventID)
	require.Equal(t, groupChangeEventID, *delivery.GroupChangeEventID)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestSupplierNotificationRepositoryListDeliveriesScansGroupChangeEventID(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	now := time.Date(2026, time.August, 30, 12, 0, 0, 0, time.UTC)
	mock.ExpectQuery(regexp.QuoteMeta("SELECT COUNT(*) FROM supplier_notification_deliveries d WHERE 1 = 1")).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(int64(1)))
	mock.ExpectQuery(`SELECT d\.id, d\.channel_id, c\.name, d\.event_id`).
		WithArgs(50, 0).
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "channel_id", "channel_name", "event_id", "group_change_event_id", "provider_id", "provider_name", "event_type",
			"status", "payload_json", "attempt_count", "next_attempt_at", "last_error", "sent_at", "created_at", "updated_at",
		}).AddRow(
			int64(11), int64(1), "channel", nil, int64(901), int64(10), "provider", service.SupplierGroupChangeEventType, "pending",
			[]byte(`{"event_type":"group_changed"}`), 0, now, "", nil, now, now,
		))

	repo := &supplierNotificationRepository{db: db}
	result, err := repo.ListDeliveries(context.Background(), service.SupplierNotificationDeliveryListParams{})
	require.NoError(t, err)
	require.Len(t, result.Items, 1)
	require.Nil(t, result.Items[0].EventID)
	require.NotNil(t, result.Items[0].GroupChangeEventID)
	require.Equal(t, int64(901), *result.Items[0].GroupChangeEventID)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestSupplierNotificationRepositoryListDeliveriesScansPayloadJSON(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	now := time.Date(2026, time.August, 5, 12, 0, 0, 0, time.UTC)
	mock.ExpectQuery(regexp.QuoteMeta("SELECT COUNT(*) FROM supplier_notification_deliveries d WHERE 1 = 1")).WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(int64(1)))
	mock.ExpectQuery(`SELECT d\.id, d\.channel_id, c\.name, d\.event_id`).WithArgs(50, 0).WillReturnRows(sqlmock.NewRows([]string{
		"id", "channel_id", "channel_name", "event_id", "group_change_event_id", "provider_id", "provider_name", "event_type",
		"status", "payload_json", "attempt_count", "next_attempt_at", "last_error", "sent_at", "created_at", "updated_at",
	}).AddRow(
		int64(11), int64(1), "channel", nil, nil, int64(10), "provider", "balance_low", "pending",
		[]byte(`{"provider_id":10}`), 0, now, "", nil, now, now,
	))

	repo := &supplierNotificationRepository{db: db}
	result, err := repo.ListDeliveries(context.Background(), service.SupplierNotificationDeliveryListParams{})
	require.NoError(t, err)
	require.Equal(t, int64(1), result.Total)
	require.Len(t, result.Items, 1)
	require.Equal(t, int64(11), result.Items[0].ID)
	require.Nil(t, result.Items[0].EventID)
	require.Nil(t, result.Items[0].GroupChangeEventID)
	require.Nil(t, result.Items[0].SentAt)
	require.NoError(t, mock.ExpectationsWereMet())
}

package service

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestSupplierNotificationAcceptsCostAlertEventTypes(t *testing.T) {
	require.True(t, isValidSupplierNotificationEventType(SupplierCostAlertEventOverrun))
	require.True(t, isValidSupplierNotificationEventType(SupplierCostAlertEventRecovered))
	require.False(t, isValidSupplierNotificationEventType("invalid"))
}

func TestSupplierNotificationSubscriptionValidationAllowsCostAlertTypes(t *testing.T) {
	repo := &supplierNotificationRepoStub{channels: map[int64]SupplierNotificationChannel{1: {ID: 1, Enabled: true}}}
	service := NewSupplierNotificationService(repo, nil, nil)
	for _, eventType := range []string{SupplierCostAlertEventOverrun, SupplierCostAlertEventRecovered} {
		item, err := service.SaveSubscription(context.Background(), 0, SupplierNotificationSubscriptionInput{
			ChannelID: 1,
			EventType: eventType,
			Enabled:   true,
		})
		require.NoError(t, err)
		require.Equal(t, eventType, item.EventType)
	}
}

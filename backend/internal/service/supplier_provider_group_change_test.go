package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestBuildSupplierProviderGroupChangeSummary(t *testing.T) {
	previous := []SupplierProviderGroupSnapshot{
		{UpstreamKey: "stable", Name: "企业客户组", RateMultiplier: 1, Active: true},
		{UpstreamKey: "removed", Name: "旧分组", RateMultiplier: 1.5, Active: true},
		{UpstreamKey: "inactive", Name: "历史分组", RateMultiplier: 2, Active: false},
	}
	current := []SupplierProviderGroupSnapshot{
		{UpstreamKey: "stable", Name: "企业高级客户组", RateMultiplier: 1.25, Active: true},
		{UpstreamKey: "added", Name: "新分组", RateMultiplier: 1.8, Active: true},
		{UpstreamKey: "inactive", Name: "历史分组", RateMultiplier: 2, Active: true},
	}

	changes := BuildSupplierProviderGroupChangeSummary(previous, current)

	require.Equal(t, []SupplierProviderGroupChange{{
		ChangeType:        SupplierProviderGroupChangeTypeAdded,
		UpstreamKey:       "added",
		NewName:           "新分组",
		NewRateMultiplier: 1.8,
	}, {
		ChangeType:        SupplierProviderGroupChangeTypeAdded,
		UpstreamKey:       "inactive",
		NewName:           "历史分组",
		NewRateMultiplier: 2,
	}}, changes.Added)
	require.Equal(t, []SupplierProviderGroupChange{{
		ChangeType:        SupplierProviderGroupChangeTypeRemoved,
		UpstreamKey:       "removed",
		OldName:           "旧分组",
		OldRateMultiplier: 1.5,
	}}, changes.Removed)
	require.Equal(t, []SupplierProviderGroupChange{{
		ChangeType:        SupplierProviderGroupChangeTypeRateChanged,
		UpstreamKey:       "stable",
		OldName:           "企业客户组",
		NewName:           "企业高级客户组",
		OldRateMultiplier: 1,
		NewRateMultiplier: 1.25,
	}}, changes.RateChanged)
	require.Equal(t, []SupplierProviderGroupChange{{
		ChangeType:        SupplierProviderGroupChangeTypeNameChanged,
		UpstreamKey:       "stable",
		OldName:           "企业客户组",
		NewName:           "企业高级客户组",
		OldRateMultiplier: 1,
		NewRateMultiplier: 1.25,
	}}, changes.NameChanged)
	require.Equal(t, 5, changes.Count())
}

func TestBuildSupplierProviderGroupChangeSummaryIgnoresTrimOnlyNameChange(t *testing.T) {
	changes := BuildSupplierProviderGroupChangeSummary(
		[]SupplierProviderGroupSnapshot{{UpstreamKey: "group", Name: "企业客户组", RateMultiplier: 1, Active: true}},
		[]SupplierProviderGroupSnapshot{{UpstreamKey: "group", Name: " 企业客户组 ", RateMultiplier: 1 + 1e-10, Active: true}},
	)

	require.True(t, changes.Empty())
}

type supplierGroupChangeNotifierStub struct {
	events []SupplierGroupChangeEvent
	err    error
}

func (n *supplierGroupChangeNotifierStub) DispatchGroupChanged(_ context.Context, event SupplierGroupChangeEvent) error {
	n.events = append(n.events, event)
	return n.err
}

func TestSupplierProviderSyncServiceGroupChangeNotificationIsBestEffort(t *testing.T) {
	providerRepo := &supplierProviderRepoStub{items: []*SupplierProvider{{
		ID: 42, Name: "供应商甲", Code: "provider-a", ProviderType: SupplierProviderTypeSub2API, Enabled: true, PasswordEncrypted: "secret",
	}}}
	dataRepo := &supplierProviderDataRepoStub{groupChanges: SupplierProviderGroupChangeSummary{
		Added: []SupplierProviderGroupChange{{UpstreamKey: "new", NewName: "新分组", NewRateMultiplier: 1.2}},
	}}
	remote := &supplierRemoteClientStub{}
	notifier := &supplierGroupChangeNotifierStub{err: errors.New("通知服务暂不可用")}
	svc := NewSupplierProviderSyncService(providerRepo, dataRepo, remote, supplierEncryptorStub{}, &supplierSyncLockStub{acquired: true})
	svc.SetGroupChangeNotifier(notifier)

	result, err := svc.SyncGroups(context.Background(), 42, SupplierSyncTriggerManual)

	require.NoError(t, err)
	require.Equal(t, SupplierSyncStatusSuccess, result.Status)
	require.Len(t, notifier.events, 1)
	require.Equal(t, int64(42), notifier.events[0].ProviderID)
	require.Equal(t, "provider-a", notifier.events[0].ProviderCode)
	require.Contains(t, result.Message, "通知")
}

func TestSupplierNotificationDispatcherDispatchesGroupChangedOnce(t *testing.T) {
	repo := newSupplierNotificationRepoStub()
	repo.channels[1] = SupplierNotificationChannel{ID: 1, Name: "飞书", ChannelType: SupplierNotificationChannelFeishu, Enabled: true}
	repo.subscriptions = []SupplierNotificationSubscription{{ChannelID: 1, EventType: SupplierGroupChangeEventType, Enabled: true}}
	sender := &supplierNotificationSenderStub{}
	dispatcher := NewSupplierNotificationDispatcher(repo, sender)

	event := SupplierGroupChangeEvent{
		ProviderID:   42,
		ProviderCode: "provider-a",
		ProviderName: "供应商甲",
		Changes: SupplierProviderGroupChangeSummary{
			Added: []SupplierProviderGroupChange{{UpstreamKey: "new", NewName: "新分组", NewRateMultiplier: 1.2}},
		},
		ObservedAt: time.Now(),
	}

	require.NoError(t, dispatcher.DispatchGroupChanged(context.Background(), event))
	require.Len(t, repo.groupEvents, 1)
	require.Len(t, repo.deliveries, 1)
	for _, delivery := range repo.deliveries {
		require.Equal(t, SupplierGroupChangeEventType, delivery.EventType)
		require.NotNil(t, delivery.GroupChangeEventID)
		require.Nil(t, delivery.EventID)
	}
}

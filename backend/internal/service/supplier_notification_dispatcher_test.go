package service

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/shopspring/decimal"
)

type supplierNotificationRepoStub struct {
	channels       map[int64]SupplierNotificationChannel
	subscriptions  []SupplierNotificationSubscription
	cooldowns      map[string]bool
	deliveries     map[int64]*SupplierNotificationDeliveryRecord
	attempts       map[int64][]SupplierNotificationDeliveryAttempt
	nextDeliveryID int64
	dueIDs         []int64
	claimCount     int
}

func (r *supplierNotificationRepoStub) ListChannels(context.Context) ([]SupplierNotificationChannel, error) {
	items := make([]SupplierNotificationChannel, 0, len(r.channels))
	for _, item := range r.channels {
		items = append(items, item)
	}
	return items, nil
}

func (r *supplierNotificationRepoStub) GetChannel(_ context.Context, id int64) (*SupplierNotificationChannel, error) {
	item, ok := r.channels[id]
	if !ok {
		return nil, ErrSupplierNotificationChannelNotFound
	}
	return &item, nil
}

func (r *supplierNotificationRepoStub) SaveChannel(_ context.Context, channel *SupplierNotificationChannel) error {
	if channel.ID == 0 {
		r.nextDeliveryID++
		channel.ID = r.nextDeliveryID
	}
	r.channels[channel.ID] = *channel
	return nil
}

func (r *supplierNotificationRepoStub) DeleteChannel(_ context.Context, id int64) error {
	delete(r.channels, id)
	return nil
}

func (r *supplierNotificationRepoStub) ListSubscriptions(_ context.Context, channelID int64) ([]SupplierNotificationSubscription, error) {
	items := make([]SupplierNotificationSubscription, 0)
	for _, item := range r.subscriptions {
		if channelID <= 0 || item.ChannelID == channelID {
			items = append(items, item)
		}
	}
	return items, nil
}

func (r *supplierNotificationRepoStub) GetSubscription(_ context.Context, id int64) (*SupplierNotificationSubscription, error) {
	for _, item := range r.subscriptions {
		if item.ID == id {
			copy := item
			return &copy, nil
		}
	}
	return nil, ErrSupplierNotificationSubscriptionNotFound
}

func (r *supplierNotificationRepoStub) UpsertSubscription(_ context.Context, subscription *SupplierNotificationSubscription) error {
	if subscription.ID > 0 {
		for i := range r.subscriptions {
			if r.subscriptions[i].ID == subscription.ID {
				r.subscriptions[i] = *subscription
				return nil
			}
		}
	}
	subscription.ID = int64(len(r.subscriptions) + 1)
	r.subscriptions = append(r.subscriptions, *subscription)
	return nil
}

func (r *supplierNotificationRepoStub) DeleteSubscription(_ context.Context, id int64) error {
	for i := range r.subscriptions {
		if r.subscriptions[i].ID == id {
			r.subscriptions = append(r.subscriptions[:i], r.subscriptions[i+1:]...)
			return nil
		}
	}
	return ErrSupplierNotificationSubscriptionNotFound
}

func (r *supplierNotificationRepoStub) ListMatchingSubscriptions(_ context.Context, channelID, providerID int64, eventType string) ([]SupplierNotificationSubscription, error) {
	items := make([]SupplierNotificationSubscription, 0)
	for _, item := range r.subscriptions {
		if item.ChannelID == channelID && item.Enabled && item.EventType == eventType && (item.ProviderID == nil || *item.ProviderID == providerID) {
			items = append(items, item)
		}
	}
	return items, nil
}

func supplierNotificationCooldownKey(channelID, providerID int64, eventType string) string {
	return string(rune(channelID)) + ":" + string(rune(providerID)) + ":" + eventType
}

func (r *supplierNotificationRepoStub) ClaimCooldown(_ context.Context, channelID, providerID int64, eventType string, _, _ time.Time) (bool, error) {
	key := supplierNotificationCooldownKey(channelID, providerID, eventType)
	if r.cooldowns[key] {
		return false, nil
	}
	r.cooldowns[key] = true
	r.claimCount++
	return true, nil
}

func (r *supplierNotificationRepoStub) CreateDelivery(_ context.Context, delivery *SupplierNotificationDeliveryRecord) error {
	r.nextDeliveryID++
	delivery.ID = r.nextDeliveryID
	if delivery.Status == "" {
		delivery.Status = SupplierNotificationDeliveryPending
	}
	if delivery.NextAttemptAt.IsZero() {
		delivery.NextAttemptAt = time.Now()
	}
	copy := *delivery
	r.deliveries[delivery.ID] = &copy
	return nil
}

func (r *supplierNotificationRepoStub) GetDelivery(_ context.Context, id int64) (*SupplierNotificationDeliveryRecord, error) {
	item, ok := r.deliveries[id]
	if !ok {
		return nil, ErrSupplierNotificationDeliveryNotFound
	}
	copy := *item
	return &copy, nil
}

func (r *supplierNotificationRepoStub) ListDueDeliveries(_ context.Context, _ time.Time, _ int) ([]SupplierNotificationDeliveryRecord, error) {
	items := make([]SupplierNotificationDeliveryRecord, 0)
	if len(r.dueIDs) == 0 {
		for _, item := range r.deliveries {
			if item.Status == SupplierNotificationDeliveryPending {
				items = append(items, *item)
			}
		}
		return items, nil
	}
	for _, id := range r.dueIDs {
		if item, ok := r.deliveries[id]; ok && item.Status == SupplierNotificationDeliveryPending {
			items = append(items, *item)
		}
	}
	return items, nil
}

func (r *supplierNotificationRepoStub) ClaimDelivery(_ context.Context, deliveryID int64) (bool, error) {
	item, ok := r.deliveries[deliveryID]
	if !ok || item.Status != SupplierNotificationDeliveryPending {
		return false, nil
	}
	item.Status = SupplierNotificationDeliverySending
	item.AttemptCount++
	r.claimCount++
	return true, nil
}

func (r *supplierNotificationRepoStub) UpdateDelivery(_ context.Context, delivery *SupplierNotificationDeliveryRecord) error {
	copy := *delivery
	r.deliveries[delivery.ID] = &copy
	return nil
}

func (r *supplierNotificationRepoStub) CreateAttempt(_ context.Context, attempt *SupplierNotificationDeliveryAttempt) error {
	attempt.ID = int64(len(r.attempts[attempt.DeliveryID]) + 1)
	copy := *attempt
	r.attempts[attempt.DeliveryID] = append(r.attempts[attempt.DeliveryID], copy)
	return nil
}

func (r *supplierNotificationRepoStub) UpdateAttempt(_ context.Context, attempt *SupplierNotificationDeliveryAttempt) error {
	items := r.attempts[attempt.DeliveryID]
	for i := range items {
		if items[i].AttemptNumber == attempt.AttemptNumber {
			items[i] = *attempt
		}
	}
	r.attempts[attempt.DeliveryID] = items
	return nil
}

func (r *supplierNotificationRepoStub) ListDeliveries(context.Context, SupplierNotificationDeliveryListParams) (SupplierNotificationDeliveryListResult, error) {
	return SupplierNotificationDeliveryListResult{}, nil
}

func (r *supplierNotificationRepoStub) ListAttempts(_ context.Context, deliveryID int64) ([]SupplierNotificationDeliveryAttempt, error) {
	return r.attempts[deliveryID], nil
}

type supplierNotificationSenderStub struct {
	calls   int
	results []error
}

func (s *supplierNotificationSenderStub) Send(context.Context, SupplierNotificationChannel, SupplierNotificationEventPayload) (SupplierNotificationSendResult, error) {
	index := s.calls
	s.calls++
	if index < len(s.results) && s.results[index] != nil {
		return SupplierNotificationSendResult{HTTPStatus: 500, ResponseBody: "failed"}, s.results[index]
	}
	return SupplierNotificationSendResult{HTTPStatus: 200, ResponseBody: `{"code":0}`}, nil
}

func newSupplierNotificationRepoStub() *supplierNotificationRepoStub {
	return &supplierNotificationRepoStub{
		channels:   map[int64]SupplierNotificationChannel{},
		cooldowns:  map[string]bool{},
		deliveries: map[int64]*SupplierNotificationDeliveryRecord{},
		attempts:   map[int64][]SupplierNotificationDeliveryAttempt{},
	}
}

func TestSupplierNotificationDispatcherSkipsChannelWithoutMatchingSubscription(t *testing.T) {
	repo := newSupplierNotificationRepoStub()
	repo.channels[1] = SupplierNotificationChannel{ID: 1, Name: "飞书", ChannelType: SupplierNotificationChannelFeishu, Enabled: true}
	repo.subscriptions = []SupplierNotificationSubscription{{ChannelID: 1, EventType: SupplierBalanceAlertEventRecovered, Enabled: true}}
	dispatcher := NewSupplierNotificationDispatcher(repo, &supplierNotificationSenderStub{})

	err := dispatcher.Dispatch(context.Background(), SupplierBalanceAlertEvent{ProviderID: 10, EventType: SupplierBalanceAlertEventLow, Balance: decimal.NewFromInt(1)})
	if err != nil {
		t.Fatalf("Dispatch returned error: %v", err)
	}
	if repo.claimCount != 0 || len(repo.deliveries) != 0 {
		t.Fatalf("claim/delivery = %d/%d, want 0/0", repo.claimCount, len(repo.deliveries))
	}
}

func TestSupplierNotificationDispatcherCreatesIndependentDeliveryAfterCooldownClaim(t *testing.T) {
	repo := newSupplierNotificationRepoStub()
	repo.channels[1] = SupplierNotificationChannel{ID: 1, Name: "飞书", ChannelType: SupplierNotificationChannelFeishu, Enabled: true}
	providerID := int64(10)
	repo.subscriptions = []SupplierNotificationSubscription{{ChannelID: 1, ProviderID: &providerID, EventType: SupplierBalanceAlertEventLow, Enabled: true}}
	dispatcher := NewSupplierNotificationDispatcher(repo, &supplierNotificationSenderStub{})
	event := SupplierBalanceAlertEvent{ID: 99, ProviderID: providerID, ProviderCode: "demo", ProviderName: "供应商", EventType: SupplierBalanceAlertEventLow, Status: SupplierBalanceAlertEventActive, Balance: decimal.NewFromInt(1), Threshold: decimal.NewFromInt(2), ObservedAt: time.Now()}

	if err := dispatcher.Dispatch(context.Background(), event); err != nil {
		t.Fatalf("Dispatch returned error: %v", err)
	}
	if repo.claimCount != 1 || len(repo.deliveries) != 1 {
		t.Fatalf("claim/delivery = %d/%d, want 1/1", repo.claimCount, len(repo.deliveries))
	}
	for _, delivery := range repo.deliveries {
		var payload SupplierNotificationEventPayload
		if err := json.Unmarshal(delivery.PayloadJSON, &payload); err != nil {
			t.Fatalf("unmarshal payload: %v", err)
		}
		if payload.EventID == nil || *payload.EventID != event.ID || payload.ProviderID != providerID {
			t.Fatalf("payload = %+v", payload)
		}
	}
}

func TestSupplierNotificationDispatcherRetriesAndMarksDelivered(t *testing.T) {
	repo := newSupplierNotificationRepoStub()
	repo.channels[1] = SupplierNotificationChannel{ID: 1, Name: "飞书", ChannelType: SupplierNotificationChannelFeishu, Enabled: true, ConfigEncrypted: "{}"}
	repo.deliveries[7] = &SupplierNotificationDeliveryRecord{ID: 7, ChannelID: 1, ProviderID: 10, ProviderName: "供应商", EventType: SupplierBalanceAlertEventLow, Status: SupplierNotificationDeliveryPending, PayloadJSON: []byte(`{"provider_id":10,"event_type":"balance_low","balance":"1","threshold":"2"}`), NextAttemptAt: time.Now()}
	repo.dueIDs = []int64{7}
	sender := &supplierNotificationSenderStub{results: []error{errors.New("temporary"), nil}}
	dispatcher := NewSupplierNotificationDispatcher(repo, sender)
	dispatcher.retryDelays = []time.Duration{0, 0, 0}

	if err := dispatcher.RunDue(context.Background()); err != nil {
		t.Fatalf("first RunDue returned error: %v", err)
	}
	if repo.deliveries[7].Status != SupplierNotificationDeliveryPending || repo.deliveries[7].AttemptCount != 1 {
		t.Fatalf("after first attempt = %+v", repo.deliveries[7])
	}
	if err := dispatcher.RunDue(context.Background()); err != nil {
		t.Fatalf("second RunDue returned error: %v", err)
	}
	if repo.deliveries[7].Status != SupplierNotificationDeliveryDelivered || repo.deliveries[7].AttemptCount != 2 {
		t.Fatalf("after second attempt = %+v", repo.deliveries[7])
	}
	if len(repo.attempts[7]) != 2 || repo.attempts[7][0].Status != "failed" || repo.attempts[7][1].Status != "succeeded" {
		t.Fatalf("attempts = %+v", repo.attempts[7])
	}
}

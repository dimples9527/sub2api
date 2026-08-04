package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"time"
)

const (
	supplierNotificationDispatcherInterval = 30 * time.Second
	supplierNotificationDispatcherBatch    = 50
	supplierNotificationMaxAttempts        = 4
)

type SupplierNotificationDispatcher struct {
	repo        SupplierNotificationRepository
	sender      SupplierNotificationSender
	interval    time.Duration
	batchSize   int
	retryDelays []time.Duration
	runGuard    chan struct{}
	lifeMu      sync.Mutex
	stopCh      chan struct{}
	doneCh      chan struct{}
}

func NewSupplierNotificationDispatcher(repo SupplierNotificationRepository, sender SupplierNotificationSender) *SupplierNotificationDispatcher {
	return &SupplierNotificationDispatcher{
		repo:        repo,
		sender:      sender,
		interval:    supplierNotificationDispatcherInterval,
		batchSize:   supplierNotificationDispatcherBatch,
		retryDelays: []time.Duration{time.Second, 5 * time.Second, 30 * time.Second},
		runGuard:    make(chan struct{}, 1),
	}
}

func (d *SupplierNotificationDispatcher) SetInterval(interval time.Duration) {
	if d == nil || interval <= 0 {
		return
	}
	d.interval = interval
}

func (d *SupplierNotificationDispatcher) Dispatch(ctx context.Context, event SupplierBalanceAlertEvent) error {
	if d == nil || d.repo == nil {
		return ErrSupplierNotificationInvalid
	}
	if event.ProviderID <= 0 || (event.EventType != SupplierBalanceAlertEventLow && event.EventType != SupplierBalanceAlertEventRecovered) {
		return ErrSupplierNotificationInvalid
	}
	channels, err := d.repo.ListChannels(ctx)
	if err != nil {
		return fmt.Errorf("查询启用供应商通知渠道失败: %w", err)
	}
	payload := supplierNotificationPayloadFromEvent(event)
	payloadJSON, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("编码供应商通知载荷失败: %w", err)
	}
	now := time.Now()
	cooldown := event.Cooldown
	if cooldown <= 0 {
		cooldown = SupplierBalanceAlertDefaultCooldown
	}
	var dispatchErr error
	for _, channel := range channels {
		if !channel.Enabled || channel.ID <= 0 {
			continue
		}
		subscriptions, listErr := d.repo.ListMatchingSubscriptions(ctx, channel.ID, event.ProviderID, event.EventType)
		if listErr != nil {
			dispatchErr = errors.Join(dispatchErr, fmt.Errorf("查询渠道 %d 的通知订阅失败: %w", channel.ID, listErr))
			continue
		}
		if len(subscriptions) == 0 {
			continue
		}
		claimed, claimErr := d.repo.ClaimCooldown(ctx, channel.ID, event.ProviderID, event.EventType, now, now.Add(cooldown))
		if claimErr != nil {
			dispatchErr = errors.Join(dispatchErr, fmt.Errorf("占用渠道 %d 的通知冷却失败: %w", channel.ID, claimErr))
			continue
		}
		if !claimed {
			continue
		}
		eventID := event.ID
		var eventIDPtr *int64
		if eventID > 0 {
			eventIDPtr = &eventID
		}
		delivery := &SupplierNotificationDeliveryRecord{
			ChannelID:     channel.ID,
			ChannelName:   channel.Name,
			EventID:       eventIDPtr,
			ProviderID:    event.ProviderID,
			ProviderName:  event.ProviderName,
			EventType:     event.EventType,
			Status:        SupplierNotificationDeliveryPending,
			PayloadJSON:   payloadJSON,
			NextAttemptAt: now,
		}
		if createErr := d.repo.CreateDelivery(ctx, delivery); createErr != nil {
			dispatchErr = errors.Join(dispatchErr, fmt.Errorf("创建渠道 %d 的通知投递失败: %w", channel.ID, createErr))
		}
	}
	return dispatchErr
}

func supplierNotificationPayloadFromEvent(event SupplierBalanceAlertEvent) SupplierNotificationEventPayload {
	observedAt := event.ObservedAt
	if observedAt.IsZero() {
		observedAt = time.Now()
	}
	var eventID *int64
	if event.ID > 0 {
		id := event.ID
		eventID = &id
	}
	return SupplierNotificationEventPayload{
		EventID:      eventID,
		ProviderID:   event.ProviderID,
		ProviderCode: event.ProviderCode,
		ProviderName: event.ProviderName,
		EventType:    event.EventType,
		Status:       event.Status,
		Balance:      event.Balance,
		Threshold:    event.Threshold,
		ObservedAt:   observedAt,
		ResolvedAt:   event.ResolvedAt,
	}
}

func (d *SupplierNotificationDispatcher) RunDue(ctx context.Context) error {
	if d == nil || d.repo == nil || d.sender == nil {
		return ErrSupplierNotificationInvalid
	}
	select {
	case d.runGuard <- struct{}{}:
		defer func() { <-d.runGuard }()
	default:
		return nil
	}
	items, err := d.repo.ListDueDeliveries(ctx, time.Now(), d.batchSize)
	if err != nil {
		return fmt.Errorf("查询待处理供应商通知失败: %w", err)
	}
	var runErr error
	for i := range items {
		if err := d.processDelivery(ctx, items[i].ID); err != nil {
			runErr = errors.Join(runErr, err)
		}
	}
	return runErr
}

func (d *SupplierNotificationDispatcher) processDelivery(ctx context.Context, deliveryID int64) error {
	claimed, err := d.repo.ClaimDelivery(ctx, deliveryID)
	if err != nil {
		return err
	}
	if !claimed {
		return nil
	}
	delivery, err := d.repo.GetDelivery(ctx, deliveryID)
	if err != nil {
		return err
	}
	if delivery == nil {
		return ErrSupplierNotificationDeliveryNotFound
	}
	attempt := &SupplierNotificationDeliveryAttempt{
		DeliveryID:    delivery.ID,
		AttemptNumber: delivery.AttemptCount,
		Status:        "sending",
		AttemptedAt:   time.Now(),
	}
	if attempt.AttemptNumber <= 0 {
		attempt.AttemptNumber = 1
	}
	if err := d.repo.CreateAttempt(ctx, attempt); err != nil {
		return err
	}
	channel, err := d.repo.GetChannel(ctx, delivery.ChannelID)
	if err != nil {
		return d.finishDeliveryFailure(ctx, delivery, attempt, 0, "查询通知渠道失败", "")
	}
	var payload SupplierNotificationEventPayload
	if err := json.Unmarshal(delivery.PayloadJSON, &payload); err != nil {
		return d.finishDeliveryFailure(ctx, delivery, attempt, 0, "解析通知载荷失败", "")
	}
	result, sendErr := d.sender.Send(ctx, *channel, payload)
	if sendErr != nil {
		return d.finishDeliveryFailure(ctx, delivery, attempt, result.HTTPStatus, sendErr.Error(), result.ResponseBody)
	}
	finishedAt := time.Now()
	attempt.Status = "succeeded"
	attempt.HTTPStatus = result.HTTPStatus
	attempt.ResponseBody = sanitizeSupplierNotificationText(result.ResponseBody)
	attempt.FinishedAt = &finishedAt
	if err := d.repo.UpdateAttempt(ctx, attempt); err != nil {
		return err
	}
	delivery.Status = SupplierNotificationDeliveryDelivered
	delivery.LastError = ""
	delivery.SentAt = &finishedAt
	delivery.NextAttemptAt = finishedAt
	if err := d.repo.UpdateDelivery(ctx, delivery); err != nil {
		return err
	}
	return nil
}

func (d *SupplierNotificationDispatcher) finishDeliveryFailure(ctx context.Context, delivery *SupplierNotificationDeliveryRecord, attempt *SupplierNotificationDeliveryAttempt, httpStatus int, message, responseBody string) error {
	finishedAt := time.Now()
	attempt.Status = "failed"
	attempt.HTTPStatus = httpStatus
	attempt.ErrorMessage = truncateSupplierNotificationText(message, 2000)
	attempt.ResponseBody = sanitizeSupplierNotificationText(responseBody)
	attempt.FinishedAt = &finishedAt
	if err := d.repo.UpdateAttempt(ctx, attempt); err != nil {
		return err
	}
	delivery.LastError = truncateSupplierNotificationText(message, 2000)
	if delivery.AttemptCount < supplierNotificationMaxAttempts {
		delivery.Status = SupplierNotificationDeliveryPending
		delivery.NextAttemptAt = finishedAt.Add(d.retryDelay(delivery.AttemptCount))
	} else {
		delivery.Status = SupplierNotificationDeliveryFailed
		delivery.NextAttemptAt = finishedAt
	}
	return d.repo.UpdateDelivery(ctx, delivery)
}

func (d *SupplierNotificationDispatcher) retryDelay(attemptCount int) time.Duration {
	index := attemptCount - 1
	if index < 0 || index >= len(d.retryDelays) {
		return 0
	}
	return d.retryDelays[index]
}

func (d *SupplierNotificationDispatcher) Start() {
	if d == nil || d.repo == nil || d.sender == nil || d.interval <= 0 {
		return
	}
	d.lifeMu.Lock()
	if d.stopCh != nil {
		d.lifeMu.Unlock()
		return
	}
	stopCh := make(chan struct{})
	doneCh := make(chan struct{})
	d.stopCh = stopCh
	d.doneCh = doneCh
	interval := d.interval
	d.lifeMu.Unlock()
	go func() {
		defer close(doneCh)
		_ = d.RunDue(context.Background())
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				_ = d.RunDue(context.Background())
			case <-stopCh:
				return
			}
		}
	}()
}

func (d *SupplierNotificationDispatcher) Stop() {
	if d == nil {
		return
	}
	d.lifeMu.Lock()
	stopCh := d.stopCh
	doneCh := d.doneCh
	d.stopCh = nil
	d.doneCh = nil
	d.lifeMu.Unlock()
	if stopCh != nil {
		close(stopCh)
	}
	if doneCh != nil {
		<-doneCh
	}
}

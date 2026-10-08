package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/service"
)

type supplierNotificationRepository struct {
	db *sql.DB
}

func NewSupplierNotificationRepository(db *sql.DB) service.SupplierNotificationRepository {
	return &supplierNotificationRepository{db: db}
}

const supplierNotificationChannelSelect = `
SELECT id, name, channel_type, enabled, config_encrypted, proxy_encrypted, created_at, updated_at
FROM supplier_notification_channels`

func (r *supplierNotificationRepository) ListChannels(ctx context.Context) ([]service.SupplierNotificationChannel, error) {
	rows, err := r.db.QueryContext(ctx, supplierNotificationChannelSelect+" ORDER BY id ASC")
	if err != nil {
		return nil, fmt.Errorf("查询供应商通知渠道失败: %w", err)
	}
	defer rows.Close()
	items := make([]service.SupplierNotificationChannel, 0)
	for rows.Next() {
		item, scanErr := scanSupplierNotificationChannel(rows)
		if scanErr != nil {
			return nil, scanErr
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("遍历供应商通知渠道失败: %w", err)
	}
	return items, nil
}

func (r *supplierNotificationRepository) GetChannel(ctx context.Context, id int64) (*service.SupplierNotificationChannel, error) {
	item, err := scanSupplierNotificationChannel(r.db.QueryRowContext(ctx, supplierNotificationChannelSelect+" WHERE id = $1", id))
	if errors.Is(err, sql.ErrNoRows) {
		return nil, service.ErrSupplierNotificationChannelNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("查询供应商通知渠道失败: %w", err)
	}
	return &item, nil
}

func (r *supplierNotificationRepository) SaveChannel(ctx context.Context, channel *service.SupplierNotificationChannel) error {
	if channel == nil {
		return service.ErrSupplierNotificationInvalid
	}
	if channel.ID == 0 {
		err := r.db.QueryRowContext(ctx, `
INSERT INTO supplier_notification_channels (name, channel_type, enabled, config_encrypted, proxy_encrypted)
VALUES ($1, $2, $3, $4, $5)
RETURNING id, created_at, updated_at`, channel.Name, channel.ChannelType, channel.Enabled,
			channel.ConfigEncrypted, channel.ProxyEncrypted).
			Scan(&channel.ID, &channel.CreatedAt, &channel.UpdatedAt)
		if err != nil {
			return fmt.Errorf("创建供应商通知渠道失败: %w", err)
		}
		return nil
	}
	result, err := r.db.ExecContext(ctx, `
UPDATE supplier_notification_channels
SET name = $2, channel_type = $3, enabled = $4, config_encrypted = $5, proxy_encrypted = $6, updated_at = NOW()
WHERE id = $1`, channel.ID, channel.Name, channel.ChannelType, channel.Enabled,
		channel.ConfigEncrypted, channel.ProxyEncrypted)
	if err != nil {
		return fmt.Errorf("更新供应商通知渠道失败: %w", err)
	}
	if affected, _ := result.RowsAffected(); affected == 0 {
		return service.ErrSupplierNotificationChannelNotFound
	}
	return nil
}

func (r *supplierNotificationRepository) DeleteChannel(ctx context.Context, id int64) error {
	result, err := r.db.ExecContext(ctx, "DELETE FROM supplier_notification_channels WHERE id = $1", id)
	if err != nil {
		return fmt.Errorf("删除供应商通知渠道失败: %w", err)
	}
	if affected, _ := result.RowsAffected(); affected == 0 {
		return service.ErrSupplierNotificationChannelNotFound
	}
	return nil
}

const supplierNotificationSubscriptionSelect = `
SELECT id, channel_id, provider_id, group_id, event_type, enabled, created_at, updated_at
FROM supplier_notification_subscriptions`

func (r *supplierNotificationRepository) ListSubscriptions(ctx context.Context, channelID int64) ([]service.SupplierNotificationSubscription, error) {
	query := supplierNotificationSubscriptionSelect
	args := make([]any, 0, 1)
	if channelID > 0 {
		query += " WHERE channel_id = $1"
		args = append(args, channelID)
	}
	query += " ORDER BY id ASC"
	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("查询供应商通知订阅失败: %w", err)
	}
	defer rows.Close()
	items := make([]service.SupplierNotificationSubscription, 0)
	for rows.Next() {
		item, scanErr := scanSupplierNotificationSubscription(rows)
		if scanErr != nil {
			return nil, scanErr
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("遍历供应商通知订阅失败: %w", err)
	}
	return items, nil
}

func (r *supplierNotificationRepository) GetSubscription(ctx context.Context, id int64) (*service.SupplierNotificationSubscription, error) {
	item, err := scanSupplierNotificationSubscription(r.db.QueryRowContext(ctx, supplierNotificationSubscriptionSelect+" WHERE id = $1", id))
	if errors.Is(err, sql.ErrNoRows) {
		return nil, service.ErrSupplierNotificationSubscriptionNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("查询供应商通知订阅失败: %w", err)
	}
	return &item, nil
}

func (r *supplierNotificationRepository) UpsertSubscription(ctx context.Context, subscription *service.SupplierNotificationSubscription) error {
	if subscription == nil {
		return service.ErrSupplierNotificationInvalid
	}
	var existingID int64
	// 冲突键必须与迁移 246 的唯一索引逐字一致（channel_id, event_type, COALESCE(provider_id,0), COALESCE(group_id,0)）：
	// 少了 group_id，同渠道同事件下「按供应商」与「按分组」两条订阅会互相顶掉。
	err := r.db.QueryRowContext(ctx, `
SELECT id FROM supplier_notification_subscriptions
WHERE channel_id = $1 AND event_type = $2
  AND provider_id IS NOT DISTINCT FROM $3
  AND group_id IS NOT DISTINCT FROM $4`,
		subscription.ChannelID, subscription.EventType, subscription.ProviderID, subscription.GroupID).Scan(&existingID)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return fmt.Errorf("查询供应商通知订阅冲突失败: %w", err)
	}
	if existingID > 0 {
		subscription.ID = existingID
		result, updateErr := r.db.ExecContext(ctx, `
UPDATE supplier_notification_subscriptions
SET provider_id = $2, group_id = $3, event_type = $4, enabled = $5, updated_at = NOW()
WHERE id = $1`, subscription.ID, subscription.ProviderID, subscription.GroupID,
			subscription.EventType, subscription.Enabled)
		if updateErr != nil {
			return fmt.Errorf("更新供应商通知订阅失败: %w", updateErr)
		}
		if affected, _ := result.RowsAffected(); affected == 0 {
			return service.ErrSupplierNotificationSubscriptionNotFound
		}
		return nil
	}
	err = r.db.QueryRowContext(ctx, `
INSERT INTO supplier_notification_subscriptions (channel_id, provider_id, group_id, event_type, enabled)
VALUES ($1, $2, $3, $4, $5)
RETURNING id, created_at, updated_at`, subscription.ChannelID, subscription.ProviderID,
		subscription.GroupID, subscription.EventType, subscription.Enabled).
		Scan(&subscription.ID, &subscription.CreatedAt, &subscription.UpdatedAt)
	if err != nil {
		return fmt.Errorf("创建供应商通知订阅失败: %w", err)
	}
	return nil
}

func (r *supplierNotificationRepository) DeleteSubscription(ctx context.Context, id int64) error {
	result, err := r.db.ExecContext(ctx, "DELETE FROM supplier_notification_subscriptions WHERE id = $1", id)
	if err != nil {
		return fmt.Errorf("删除供应商通知订阅失败: %w", err)
	}
	if affected, _ := result.RowsAffected(); affected == 0 {
		return service.ErrSupplierNotificationSubscriptionNotFound
	}
	return nil
}

func (r *supplierNotificationRepository) ListMatchingSubscriptions(ctx context.Context, channelID int64, providerID int64, eventType string) ([]service.SupplierNotificationSubscription, error) {
	rows, err := r.db.QueryContext(ctx, supplierNotificationSubscriptionSelect+`
WHERE channel_id = $1 AND enabled = TRUE AND event_type = $2
  AND (provider_id = $3 OR provider_id IS NULL)
ORDER BY provider_id NULLS FIRST, id ASC`, channelID, eventType, providerID)
	if err != nil {
		return nil, fmt.Errorf("查询匹配供应商通知订阅失败: %w", err)
	}
	defer rows.Close()
	items := make([]service.SupplierNotificationSubscription, 0)
	for rows.Next() {
		item, scanErr := scanSupplierNotificationSubscription(rows)
		if scanErr != nil {
			return nil, scanErr
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("遍历匹配供应商通知订阅失败: %w", err)
	}
	return items, nil
}

// ListMatchingGroupSubscriptions 按分组维度匹配订阅：命中「指定分组」或「通配分组」（group_id IS NULL）。
// 与 ListMatchingSubscriptions 同构，区别只在维度 —— 分组异常事件没有供应商。
func (r *supplierNotificationRepository) ListMatchingGroupSubscriptions(ctx context.Context, channelID int64, groupID int64, eventType string) ([]service.SupplierNotificationSubscription, error) {
	rows, err := r.db.QueryContext(ctx, supplierNotificationSubscriptionSelect+`
WHERE channel_id = $1 AND enabled = TRUE AND event_type = $2
  AND (group_id = $3 OR group_id IS NULL)
ORDER BY group_id NULLS FIRST, id ASC`, channelID, eventType, groupID)
	if err != nil {
		return nil, fmt.Errorf("查询匹配分组通知订阅失败: %w", err)
	}
	defer rows.Close()
	items := make([]service.SupplierNotificationSubscription, 0)
	for rows.Next() {
		item, scanErr := scanSupplierNotificationSubscription(rows)
		if scanErr != nil {
			return nil, scanErr
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("遍历匹配分组通知订阅失败: %w", err)
	}
	return items, nil
}

func (r *supplierNotificationRepository) ClaimCooldown(ctx context.Context, channelID, providerID int64, eventType string, now, expiresAt time.Time) (bool, error) {
	var id int64
	err := r.db.QueryRowContext(ctx, `
INSERT INTO supplier_notification_cooldowns (channel_id, provider_id, event_type, expires_at, claimed_at, updated_at)
VALUES ($1, $2, $3, $5, $4, $4)
ON CONFLICT (channel_id, provider_id, event_type) DO UPDATE SET
  expires_at = EXCLUDED.expires_at,
  claimed_at = EXCLUDED.claimed_at,
  updated_at = EXCLUDED.updated_at
WHERE supplier_notification_cooldowns.expires_at <= $4
RETURNING id`, channelID, providerID, eventType, now, expiresAt).Scan(&id)
	if errors.Is(err, sql.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("占用供应商通知冷却失败: %w", err)
	}
	return id > 0, nil
}

// ClaimGroupCooldown 是分组维度的冷却占用，落在独立表 supplier_group_notification_cooldowns。
// 刻意不复用 ClaimCooldown：那张表的 provider_id 非空，分组事件没有供应商。
func (r *supplierNotificationRepository) ClaimGroupCooldown(ctx context.Context, channelID, groupID int64, eventType string, now, expiresAt time.Time) (bool, error) {
	var id int64
	err := r.db.QueryRowContext(ctx, `
INSERT INTO supplier_group_notification_cooldowns (channel_id, group_id, event_type, expires_at, claimed_at, updated_at)
VALUES ($1, $2, $3, $5, $4, $4)
ON CONFLICT (channel_id, group_id, event_type) DO UPDATE SET
  expires_at = EXCLUDED.expires_at,
  claimed_at = EXCLUDED.claimed_at,
  updated_at = EXCLUDED.updated_at
WHERE supplier_group_notification_cooldowns.expires_at <= $4
RETURNING id`, channelID, groupID, eventType, now, expiresAt).Scan(&id)
	if errors.Is(err, sql.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("占用分组通知冷却失败: %w", err)
	}
	return id > 0, nil
}

type supplierGroupChangeEventPayload struct {
	ProviderID   int64                                      `json:"provider_id"`
	ProviderCode string                                     `json:"provider_code"`
	ProviderName string                                     `json:"provider_name"`
	EventType    string                                     `json:"event_type"`
	ChangeCount  int                                        `json:"change_count"`
	ObservedAt   time.Time                                  `json:"observed_at"`
	Changes      service.SupplierProviderGroupChangeSummary `json:"changes"`
}

func (r *supplierNotificationRepository) CreateGroupChangeEvent(ctx context.Context, event *service.SupplierGroupChangeEvent) error {
	if event == nil || event.ProviderID <= 0 || event.Changes.Empty() {
		return service.ErrSupplierNotificationInvalid
	}
	if event.EventType == "" {
		event.EventType = service.SupplierGroupChangeEventType
	}
	if event.EventType != service.SupplierGroupChangeEventType {
		return service.ErrSupplierNotificationInvalid
	}
	if event.ChangeCount <= 0 {
		event.ChangeCount = event.Changes.Count()
	}
	if event.ChangeCount <= 0 {
		return service.ErrSupplierNotificationInvalid
	}
	if event.ObservedAt.IsZero() {
		event.ObservedAt = time.Now()
	}
	if event.CreatedAt.IsZero() {
		event.CreatedAt = time.Now()
	}
	payloadJSON, err := json.Marshal(supplierGroupChangeEventPayload{
		ProviderID:   event.ProviderID,
		ProviderCode: event.ProviderCode,
		ProviderName: event.ProviderName,
		EventType:    event.EventType,
		ChangeCount:  event.ChangeCount,
		ObservedAt:   event.ObservedAt,
		Changes:      event.Changes,
	})
	if err != nil {
		return fmt.Errorf("编码供应商分组变化事件载荷失败: %w", err)
	}
	err = r.db.QueryRowContext(ctx, `
INSERT INTO supplier_group_change_events (
  provider_id, provider_code, provider_name, sync_run_id, event_type, observed_at,
  change_count, payload_json, created_at
) VALUES ($1, $2, $3, $4, $5, $6, $7, $8::jsonb, $9)
RETURNING id`, event.ProviderID, event.ProviderCode, event.ProviderName, event.SyncRunID,
		event.EventType, event.ObservedAt, event.ChangeCount, string(payloadJSON), event.CreatedAt).
		Scan(&event.ID)
	if err != nil {
		return fmt.Errorf("创建供应商分组变化事件失败: %w", err)
	}
	return nil
}

func (r *supplierNotificationRepository) CreateDelivery(ctx context.Context, delivery *service.SupplierNotificationDeliveryRecord) error {
	if delivery == nil {
		return service.ErrSupplierNotificationInvalid
	}
	if delivery.NextAttemptAt.IsZero() {
		delivery.NextAttemptAt = time.Now()
	}
	if delivery.Status == "" {
		delivery.Status = service.SupplierNotificationDeliveryPending
	}
	// 分组维度的事件（如分组账号异常）没有供应商：ProviderID 为 0 时必须写 NULL，
	// 否则会撞上 provider_id 的外键。迁移 246 已把该列改为可空。
	var providerID any
	if delivery.ProviderID > 0 {
		providerID = delivery.ProviderID
	}
	err := r.db.QueryRowContext(ctx, `
INSERT INTO supplier_notification_deliveries (
  channel_id, event_id, cost_alert_event_id, group_change_event_id, provider_id, group_id, event_type, status, payload_json, attempt_count,
  next_attempt_at, last_error, sent_at
) VALUES ($1, $2, NULL, $3, $4, $5, $6, $7, $8::jsonb, $9, $10, $11, $12)
RETURNING id, created_at, updated_at`, delivery.ChannelID, delivery.EventID, delivery.GroupChangeEventID, providerID, delivery.GroupID,
		delivery.EventType, delivery.Status, string(delivery.PayloadJSON), delivery.AttemptCount,
		delivery.NextAttemptAt, delivery.LastError, delivery.SentAt).
		Scan(&delivery.ID, &delivery.CreatedAt, &delivery.UpdatedAt)
	if err != nil {
		return fmt.Errorf("创建供应商通知投递记录失败: %w", err)
	}
	return nil
}

// supplierNotificationDeliverySelect 同时服务投递记录（发送用）与投递列表（展示用）。
// 供应商与分组都必须用 LEFT JOIN：分组账号异常事件没有供应商，内连接会把这类记录整条丢掉。
// 两个名字列必须 COALESCE：LEFT JOIN 未命中时是 NULL，而 Go 侧是 string，直接 Scan 会报错。
const supplierNotificationDeliverySelect = `
SELECT d.id, d.channel_id, c.name, d.event_id, d.group_change_event_id, d.provider_id, COALESCE(p.name, ''), d.group_id, COALESCE(g.name, ''), d.event_type,
       d.status, d.payload_json, d.attempt_count, d.next_attempt_at, d.last_error,
       d.sent_at, d.created_at, d.updated_at
FROM supplier_notification_deliveries d
JOIN supplier_notification_channels c ON c.id = d.channel_id
LEFT JOIN supplier_providers p ON p.id = d.provider_id
LEFT JOIN groups g ON g.id = d.group_id`

func (r *supplierNotificationRepository) GetDelivery(ctx context.Context, id int64) (*service.SupplierNotificationDeliveryRecord, error) {
	item, err := scanSupplierNotificationDelivery(r.db.QueryRowContext(ctx, supplierNotificationDeliverySelect+" WHERE d.id = $1", id))
	if errors.Is(err, sql.ErrNoRows) {
		return nil, service.ErrSupplierNotificationDeliveryNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("查询供应商通知投递记录失败: %w", err)
	}
	return &item, nil
}

func (r *supplierNotificationRepository) ListDueDeliveries(ctx context.Context, now time.Time, limit int) ([]service.SupplierNotificationDeliveryRecord, error) {
	if limit < 1 || limit > 200 {
		limit = 50
	}
	if _, err := r.db.ExecContext(ctx, `
UPDATE supplier_notification_deliveries
SET status = 'pending', updated_at = NOW()
WHERE status = 'sending' AND updated_at <= $1`, now.Add(-10*time.Minute)); err != nil {
		return nil, fmt.Errorf("恢复超时供应商通知投递状态失败: %w", err)
	}
	rows, err := r.db.QueryContext(ctx, supplierNotificationDeliverySelect+`
WHERE d.status = 'pending' AND d.next_attempt_at <= $1
ORDER BY d.next_attempt_at ASC, d.id ASC LIMIT $2`, now, limit)
	if err != nil {
		return nil, fmt.Errorf("查询待处理供应商通知投递失败: %w", err)
	}
	defer rows.Close()
	items := make([]service.SupplierNotificationDeliveryRecord, 0)
	for rows.Next() {
		item, scanErr := scanSupplierNotificationDelivery(rows)
		if scanErr != nil {
			return nil, scanErr
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("遍历待处理供应商通知投递失败: %w", err)
	}
	return items, nil
}

func (r *supplierNotificationRepository) ClaimDelivery(ctx context.Context, deliveryID int64) (bool, error) {
	result, err := r.db.ExecContext(ctx, `
UPDATE supplier_notification_deliveries
SET status = 'sending', attempt_count = attempt_count + 1, updated_at = NOW()
WHERE id = $1 AND status = 'pending'`, deliveryID)
	if err != nil {
		return false, fmt.Errorf("占用供应商通知投递失败: %w", err)
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return false, fmt.Errorf("获取供应商通知投递影响行数失败: %w", err)
	}
	return affected > 0, nil
}

func (r *supplierNotificationRepository) UpdateDelivery(ctx context.Context, delivery *service.SupplierNotificationDeliveryRecord) error {
	if delivery == nil || delivery.ID <= 0 {
		return service.ErrSupplierNotificationInvalid
	}
	result, err := r.db.ExecContext(ctx, `
UPDATE supplier_notification_deliveries
SET status = $2, attempt_count = $3, next_attempt_at = $4, last_error = $5,
    sent_at = $6, updated_at = NOW()
WHERE id = $1`, delivery.ID, delivery.Status, delivery.AttemptCount, delivery.NextAttemptAt,
		truncateSupplierNotificationText(delivery.LastError, 2000), delivery.SentAt)
	if err != nil {
		return fmt.Errorf("更新供应商通知投递记录失败: %w", err)
	}
	if affected, _ := result.RowsAffected(); affected == 0 {
		return service.ErrSupplierNotificationDeliveryNotFound
	}
	return nil
}

func (r *supplierNotificationRepository) CreateAttempt(ctx context.Context, attempt *service.SupplierNotificationDeliveryAttempt) error {
	if attempt == nil {
		return service.ErrSupplierNotificationInvalid
	}
	if attempt.AttemptedAt.IsZero() {
		attempt.AttemptedAt = time.Now()
	}
	err := r.db.QueryRowContext(ctx, `
INSERT INTO supplier_notification_delivery_attempts (
  delivery_id, attempt_number, status, http_status, error_message, response_body, attempted_at, finished_at
) VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
RETURNING id`, attempt.DeliveryID, attempt.AttemptNumber, attempt.Status, attempt.HTTPStatus,
		truncateSupplierNotificationText(attempt.ErrorMessage, 2000), truncateSupplierNotificationText(attempt.ResponseBody, 4000),
		attempt.AttemptedAt, attempt.FinishedAt).Scan(&attempt.ID)
	if err != nil {
		return fmt.Errorf("创建供应商通知投递尝试记录失败: %w", err)
	}
	return nil
}

func (r *supplierNotificationRepository) UpdateAttempt(ctx context.Context, attempt *service.SupplierNotificationDeliveryAttempt) error {
	if attempt == nil || attempt.ID <= 0 {
		return service.ErrSupplierNotificationInvalid
	}
	result, err := r.db.ExecContext(ctx, `
UPDATE supplier_notification_delivery_attempts
SET status = $2, http_status = $3, error_message = $4, response_body = $5, finished_at = $6
WHERE id = $1`, attempt.ID, attempt.Status, attempt.HTTPStatus,
		truncateSupplierNotificationText(attempt.ErrorMessage, 2000), truncateSupplierNotificationText(attempt.ResponseBody, 4000), attempt.FinishedAt)
	if err != nil {
		return fmt.Errorf("更新供应商通知投递尝试记录失败: %w", err)
	}
	if affected, _ := result.RowsAffected(); affected == 0 {
		return service.ErrSupplierNotificationDeliveryNotFound
	}
	return nil
}

func (r *supplierNotificationRepository) ListDeliveries(ctx context.Context, params service.SupplierNotificationDeliveryListParams) (service.SupplierNotificationDeliveryListResult, error) {
	page, pageSize := normalizeSupplierNotificationPage(params.Page, params.PageSize)
	where := []string{"1 = 1"}
	args := make([]any, 0, 4)
	if params.ChannelID > 0 {
		args = append(args, params.ChannelID)
		where = append(where, fmt.Sprintf("d.channel_id = $%d", len(args)))
	}
	if params.ProviderID > 0 {
		args = append(args, params.ProviderID)
		where = append(where, fmt.Sprintf("d.provider_id = $%d", len(args)))
	}
	if params.GroupID > 0 {
		args = append(args, params.GroupID)
		where = append(where, fmt.Sprintf("d.group_id = $%d", len(args)))
	}
	if strings.TrimSpace(params.EventType) != "" {
		args = append(args, params.EventType)
		where = append(where, fmt.Sprintf("d.event_type = $%d", len(args)))
	}
	if strings.TrimSpace(params.Status) != "" {
		args = append(args, params.Status)
		where = append(where, fmt.Sprintf("d.status = $%d", len(args)))
	}
	whereSQL := strings.Join(where, " AND ")
	var total int64
	if err := r.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM supplier_notification_deliveries d WHERE "+whereSQL, args...).Scan(&total); err != nil {
		return service.SupplierNotificationDeliveryListResult{}, fmt.Errorf("统计供应商通知投递记录失败: %w", err)
	}
	queryArgs := append(append([]any{}, args...), pageSize, (page-1)*pageSize)
	query := supplierNotificationDeliverySelect + " WHERE " + whereSQL + fmt.Sprintf(" ORDER BY d.created_at DESC, d.id DESC LIMIT $%d OFFSET $%d", len(args)+1, len(args)+2)
	rows, err := r.db.QueryContext(ctx, query, queryArgs...)
	if err != nil {
		return service.SupplierNotificationDeliveryListResult{}, fmt.Errorf("查询供应商通知投递记录失败: %w", err)
	}
	defer rows.Close()
	items := make([]service.SupplierNotificationDelivery, 0)
	for rows.Next() {
		item, scanErr := scanSupplierNotificationDeliveryView(rows)
		if scanErr != nil {
			return service.SupplierNotificationDeliveryListResult{}, scanErr
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return service.SupplierNotificationDeliveryListResult{}, fmt.Errorf("遍历供应商通知投递记录失败: %w", err)
	}
	return service.SupplierNotificationDeliveryListResult{Items: items, Total: total, Page: page, PageSize: pageSize}, nil
}

func (r *supplierNotificationRepository) ListAttempts(ctx context.Context, deliveryID int64) ([]service.SupplierNotificationDeliveryAttempt, error) {
	rows, err := r.db.QueryContext(ctx, `
SELECT id, delivery_id, attempt_number, status, http_status, error_message, response_body, attempted_at, finished_at
FROM supplier_notification_delivery_attempts
WHERE delivery_id = $1 ORDER BY attempt_number ASC`, deliveryID)
	if err != nil {
		return nil, fmt.Errorf("查询供应商通知投递尝试失败: %w", err)
	}
	defer rows.Close()
	items := make([]service.SupplierNotificationDeliveryAttempt, 0)
	for rows.Next() {
		item, scanErr := scanSupplierNotificationDeliveryAttempt(rows)
		if scanErr != nil {
			return nil, scanErr
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("遍历供应商通知投递尝试失败: %w", err)
	}
	return items, nil
}

type supplierNotificationScanner interface {
	Scan(dest ...any) error
}

func scanSupplierNotificationChannel(scanner supplierNotificationScanner) (service.SupplierNotificationChannel, error) {
	var item service.SupplierNotificationChannel
	err := scanner.Scan(&item.ID, &item.Name, &item.ChannelType, &item.Enabled, &item.ConfigEncrypted,
		&item.ProxyEncrypted, &item.CreatedAt, &item.UpdatedAt)
	return item, err
}

func scanSupplierNotificationSubscription(scanner supplierNotificationScanner) (service.SupplierNotificationSubscription, error) {
	var item service.SupplierNotificationSubscription
	err := scanner.Scan(&item.ID, &item.ChannelID, &item.ProviderID, &item.GroupID, &item.EventType, &item.Enabled,
		&item.CreatedAt, &item.UpdatedAt)
	return item, err
}

func scanSupplierNotificationDelivery(scanner supplierNotificationScanner) (service.SupplierNotificationDeliveryRecord, error) {
	var item service.SupplierNotificationDeliveryRecord
	// provider_id 自迁移 246 起可空（分组事件无供应商）；直接 Scan 到 int64 会在 NULL 上报错。
	var providerID sql.NullInt64
	err := scanner.Scan(&item.ID, &item.ChannelID, &item.ChannelName, &item.EventID, &item.GroupChangeEventID, &providerID,
		&item.ProviderName, &item.GroupID, &item.GroupName, &item.EventType, &item.Status, &item.PayloadJSON, &item.AttemptCount,
		&item.NextAttemptAt, &item.LastError, &item.SentAt, &item.CreatedAt, &item.UpdatedAt)
	if providerID.Valid {
		item.ProviderID = providerID.Int64
	}
	return item, err
}

func scanSupplierNotificationDeliveryView(scanner supplierNotificationScanner) (service.SupplierNotificationDelivery, error) {
	var item service.SupplierNotificationDelivery
	var payloadJSON []byte
	var providerID sql.NullInt64
	err := scanner.Scan(&item.ID, &item.ChannelID, &item.ChannelName, &item.EventID, &item.GroupChangeEventID, &providerID,
		&item.ProviderName, &item.GroupID, &item.GroupName, &item.EventType, &item.Status, &payloadJSON, &item.AttemptCount, &item.NextAttemptAt,
		&item.LastError, &item.SentAt, &item.CreatedAt, &item.UpdatedAt)
	if providerID.Valid {
		item.ProviderID = providerID.Int64
	}
	return item, err
}

func scanSupplierNotificationDeliveryAttempt(scanner supplierNotificationScanner) (service.SupplierNotificationDeliveryAttempt, error) {
	var item service.SupplierNotificationDeliveryAttempt
	err := scanner.Scan(&item.ID, &item.DeliveryID, &item.AttemptNumber, &item.Status, &item.HTTPStatus,
		&item.ErrorMessage, &item.ResponseBody, &item.AttemptedAt, &item.FinishedAt)
	return item, err
}

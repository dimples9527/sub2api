package service

import (
	"context"
	"time"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"github.com/shopspring/decimal"
)

var (
	ErrSupplierNotificationChannelNotFound      = infraerrors.NotFound("SUPPLIER_NOTIFICATION_CHANNEL_NOT_FOUND", "供应商通知渠道不存在")
	ErrSupplierNotificationSubscriptionNotFound = infraerrors.NotFound("SUPPLIER_NOTIFICATION_SUBSCRIPTION_NOT_FOUND", "供应商通知订阅不存在")
	ErrSupplierNotificationDeliveryNotFound     = infraerrors.NotFound("SUPPLIER_NOTIFICATION_DELIVERY_NOT_FOUND", "供应商通知投递记录不存在")
	ErrSupplierNotificationInvalid              = infraerrors.BadRequest("SUPPLIER_NOTIFICATION_INVALID", "供应商通知参数无效")
)

const (
	SupplierNotificationChannelFeishu = "feishu"
	SupplierNotificationChannelEmail  = "email"
)

const (
	SupplierNotificationDeliveryPending   = "pending"
	SupplierNotificationDeliverySending   = "sending"
	SupplierNotificationDeliveryDelivered = "delivered"
	SupplierNotificationDeliveryFailed    = "failed"
)

type SupplierNotificationFeishuConfig struct {
	WebhookURL string `json:"webhook_url"`
	Secret     string `json:"secret"`
}

type SupplierNotificationEmailConfig struct {
	Host     string   `json:"host"`
	Port     int      `json:"port"`
	Username string   `json:"username"`
	Password string   `json:"password"`
	From     string   `json:"from"`
	To       []string `json:"to"`
	StartTLS bool     `json:"starttls"`
}

type SupplierNotificationProxyConfig struct {
	URL      string `json:"url"`
	Username string `json:"username"`
	Password string `json:"password"`
}

type SupplierNotificationChannel struct {
	ID              int64     `json:"id"`
	Name            string    `json:"name"`
	ChannelType     string    `json:"channel_type"`
	Enabled         bool      `json:"enabled"`
	ConfigEncrypted string    `json:"-"`
	ProxyEncrypted  string    `json:"-"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}

type SupplierNotificationChannelView struct {
	ID                      int64     `json:"id"`
	Name                    string    `json:"name"`
	ChannelType             string    `json:"channel_type"`
	Enabled                 bool      `json:"enabled"`
	Configured              bool      `json:"configured"`
	FeishuWebhookConfigured bool      `json:"feishu_webhook_configured,omitempty"`
	FeishuSecretConfigured  bool      `json:"feishu_secret_configured,omitempty"`
	EmailHost               string    `json:"email_host,omitempty"`
	EmailPort               int       `json:"email_port,omitempty"`
	EmailUsername           string    `json:"email_username,omitempty"`
	EmailFrom               string    `json:"email_from,omitempty"`
	EmailTo                 []string  `json:"email_to,omitempty"`
	EmailStartTLS           bool      `json:"email_starttls,omitempty"`
	ProxyURL                string    `json:"proxy_url,omitempty"`
	ProxyConfigured         bool      `json:"proxy_configured,omitempty"`
	CreatedAt               time.Time `json:"created_at"`
	UpdatedAt               time.Time `json:"updated_at"`
}

type SupplierNotificationChannelInput struct {
	Name        string                            `json:"name"`
	ChannelType string                            `json:"channel_type"`
	Enabled     bool                              `json:"enabled"`
	Feishu      *SupplierNotificationFeishuConfig `json:"feishu,omitempty"`
	Email       *SupplierNotificationEmailConfig  `json:"email,omitempty"`
	Proxy       *SupplierNotificationProxyConfig  `json:"proxy,omitempty"`
}

type SupplierNotificationSubscription struct {
	ID         int64     `json:"id"`
	ChannelID  int64     `json:"channel_id"`
	ProviderID *int64    `json:"provider_id,omitempty"`
	EventType  string    `json:"event_type"`
	Enabled    bool      `json:"enabled"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}

type SupplierNotificationSubscriptionInput struct {
	ChannelID  int64  `json:"channel_id"`
	ProviderID *int64 `json:"provider_id,omitempty"`
	EventType  string `json:"event_type"`
	Enabled    bool   `json:"enabled"`
}

type SupplierNotificationDelivery struct {
	ID            int64      `json:"id"`
	ChannelID     int64      `json:"channel_id"`
	ChannelName   string     `json:"channel_name"`
	EventID       *int64     `json:"event_id,omitempty"`
	ProviderID    int64      `json:"provider_id"`
	ProviderName  string     `json:"provider_name"`
	EventType     string     `json:"event_type"`
	Status        string     `json:"status"`
	AttemptCount  int        `json:"attempt_count"`
	NextAttemptAt time.Time  `json:"next_attempt_at"`
	LastError     string     `json:"last_error,omitempty"`
	SentAt        *time.Time `json:"sent_at,omitempty"`
	CreatedAt     time.Time  `json:"created_at"`
	UpdatedAt     time.Time  `json:"updated_at"`
}

// SupplierNotificationEventPayload 是余额预警通知的统一消息载荷。
// 载荷会写入投递日志，但不会包含渠道密钥、SMTP 密码或代理密码。
type SupplierNotificationEventPayload struct {
	EventID      *int64          `json:"event_id,omitempty"`
	ProviderID   int64           `json:"provider_id"`
	ProviderCode string          `json:"provider_code"`
	ProviderName string          `json:"provider_name"`
	EventType    string          `json:"event_type"`
	Status       string          `json:"status"`
	Balance      decimal.Decimal `json:"balance"`
	Threshold    decimal.Decimal `json:"threshold"`
	ObservedAt   time.Time       `json:"observed_at"`
	ResolvedAt   *time.Time      `json:"resolved_at,omitempty"`
	Test         bool            `json:"test,omitempty"`
}

type SupplierNotificationDeliveryAttempt struct {
	ID            int64      `json:"id"`
	DeliveryID    int64      `json:"delivery_id"`
	AttemptNumber int        `json:"attempt_number"`
	Status        string     `json:"status"`
	HTTPStatus    int        `json:"http_status"`
	ErrorMessage  string     `json:"error_message,omitempty"`
	ResponseBody  string     `json:"response_body,omitempty"`
	AttemptedAt   time.Time  `json:"attempted_at"`
	FinishedAt    *time.Time `json:"finished_at,omitempty"`
}

type SupplierNotificationDeliveryListParams struct {
	ChannelID  int64
	ProviderID int64
	EventType  string
	Status     string
	Page       int
	PageSize   int
}

type SupplierNotificationDeliveryListResult struct {
	Items    []SupplierNotificationDelivery `json:"items"`
	Total    int64                          `json:"total"`
	Page     int                            `json:"page"`
	PageSize int                            `json:"page_size"`
}

type SupplierNotificationDeliveryRecord struct {
	ID            int64
	ChannelID     int64
	ChannelName   string
	EventID       *int64
	ProviderID    int64
	ProviderName  string
	EventType     string
	Status        string
	PayloadJSON   []byte
	AttemptCount  int
	NextAttemptAt time.Time
	LastError     string
	SentAt        *time.Time
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

type SupplierNotificationRepository interface {
	ListChannels(ctx context.Context) ([]SupplierNotificationChannel, error)
	GetChannel(ctx context.Context, id int64) (*SupplierNotificationChannel, error)
	SaveChannel(ctx context.Context, channel *SupplierNotificationChannel) error
	DeleteChannel(ctx context.Context, id int64) error
	ListSubscriptions(ctx context.Context, channelID int64) ([]SupplierNotificationSubscription, error)
	GetSubscription(ctx context.Context, id int64) (*SupplierNotificationSubscription, error)
	UpsertSubscription(ctx context.Context, subscription *SupplierNotificationSubscription) error
	DeleteSubscription(ctx context.Context, id int64) error
	ListMatchingSubscriptions(ctx context.Context, channelID int64, providerID int64, eventType string) ([]SupplierNotificationSubscription, error)
	ClaimCooldown(ctx context.Context, channelID, providerID int64, eventType string, now, expiresAt time.Time) (bool, error)
	CreateDelivery(ctx context.Context, delivery *SupplierNotificationDeliveryRecord) error
	GetDelivery(ctx context.Context, id int64) (*SupplierNotificationDeliveryRecord, error)
	ListDueDeliveries(ctx context.Context, now time.Time, limit int) ([]SupplierNotificationDeliveryRecord, error)
	ClaimDelivery(ctx context.Context, deliveryID int64) (bool, error)
	UpdateDelivery(ctx context.Context, delivery *SupplierNotificationDeliveryRecord) error
	CreateAttempt(ctx context.Context, attempt *SupplierNotificationDeliveryAttempt) error
	UpdateAttempt(ctx context.Context, attempt *SupplierNotificationDeliveryAttempt) error
	ListDeliveries(ctx context.Context, params SupplierNotificationDeliveryListParams) (SupplierNotificationDeliveryListResult, error)
	ListAttempts(ctx context.Context, deliveryID int64) ([]SupplierNotificationDeliveryAttempt, error)
}

type SupplierNotificationSendResult struct {
	HTTPStatus   int
	ResponseBody string
}

// SupplierNotificationSender 负责向单个已启用渠道发送消息。
type SupplierNotificationSender interface {
	Send(ctx context.Context, channel SupplierNotificationChannel, payload SupplierNotificationEventPayload) (SupplierNotificationSendResult, error)
}

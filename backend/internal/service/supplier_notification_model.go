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

// SupplierGroupAccountAbnormalEventType 是「分组账号异常」通知事件。
//
// 触发口径：某个分组**当前开启调度**（accounts.schedulable）的账号里，存在延迟窗口内
// 成功样本数不足（LatencySuccessCount < LatencyMinSamples）的账号 —— 即择优任务无法确认它是否健康。
// 只推异常、不推恢复：恢复属于「少收一条」而不是「错过告警」，不需要配对事件。
//
// 它没有供应商（一个分组的成员可能来自多个供应商），所以投递记录的 provider_id 为 NULL，
// 订阅与冷却都走 group_id 维度，见迁移 246。
const SupplierGroupAccountAbnormalEventType = "group_account_abnormal"

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
	ID         int64  `json:"id"`
	ChannelID  int64  `json:"channel_id"`
	ProviderID *int64 `json:"provider_id,omitempty"`
	// GroupID 是「分组账号异常」这类分组维度订阅的归属分组；与 ProviderID 互斥，
	// 两者都为 nil 表示「通配」——按供应商的通知适用所有供应商，按分组的通知适用所有分组。
	GroupID   *int64    `json:"group_id,omitempty"`
	EventType string    `json:"event_type"`
	Enabled   bool      `json:"enabled"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type SupplierNotificationSubscriptionInput struct {
	ChannelID  int64  `json:"channel_id"`
	ProviderID *int64 `json:"provider_id,omitempty"`
	GroupID    *int64 `json:"group_id,omitempty"`
	EventType  string `json:"event_type"`
	Enabled    bool   `json:"enabled"`
}

type SupplierNotificationDelivery struct {
	ID                 int64  `json:"id"`
	ChannelID          int64  `json:"channel_id"`
	ChannelName        string `json:"channel_name"`
	EventID            *int64 `json:"event_id,omitempty"`
	GroupChangeEventID *int64 `json:"group_change_event_id,omitempty"`
	ProviderID         int64  `json:"provider_id"`
	ProviderName       string `json:"provider_name"`
	// GroupID / GroupName 只对分组维度的事件（如分组账号异常）有值。
	GroupID       *int64     `json:"group_id,omitempty"`
	GroupName     string     `json:"group_name,omitempty"`
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
	EventID            *int64                              `json:"event_id,omitempty"`
	GroupChangeEventID *int64                              `json:"group_change_event_id,omitempty"`
	ProviderID         int64                               `json:"provider_id"`
	ProviderCode       string                              `json:"provider_code"`
	ProviderName       string                              `json:"provider_name"`
	EventType          string                              `json:"event_type"`
	Status             string                              `json:"status"`
	Balance            decimal.Decimal                     `json:"balance"`
	Threshold          decimal.Decimal                     `json:"threshold"`
	ObservedAt         time.Time                           `json:"observed_at"`
	ResolvedAt         *time.Time                          `json:"resolved_at,omitempty"`
	GroupChanges       *SupplierProviderGroupChangeSummary `json:"group_changes,omitempty"`
	// 以下三项只服务于「分组账号异常」事件：它没有供应商，靠 GroupID/GroupName 定位，
	// AbnormalAccounts 列出本轮样本不足的在任账号，让消息本身就能回答「是哪几个账号」。
	GroupID          int64                                 `json:"group_id,omitempty"`
	GroupName        string                                `json:"group_name,omitempty"`
	AbnormalAccounts []SupplierGroupAccountAbnormalAccount `json:"abnormal_accounts,omitempty"`
	Test             bool                                  `json:"test,omitempty"`
}

// SupplierGroupAccountAbnormalAccount 是「分组账号异常」通知里的单个账号摘要。
type SupplierGroupAccountAbnormalAccount struct {
	AccountID   int64  `json:"account_id"`
	AccountName string `json:"account_name"`
	// SuccessCount 是延迟窗口内的成功样本数，RequiredCount 是配置要求的最少样本数。
	// 两个数一起给，运维才能判断是「完全没数据」还是「差一点」。
	SuccessCount  int `json:"success_count"`
	RequiredCount int `json:"required_count"`
}

// SupplierGroupAccountAbnormalEvent 是一次「分组账号异常」事件。
//
// 只描述「哪个分组、哪些在任账号样本不足」，不携带恢复语义 —— 本事件刻意不配对 recovered。
type SupplierGroupAccountAbnormalEvent struct {
	GroupID    int64
	GroupName  string
	Accounts   []SupplierGroupAccountAbnormalAccount
	ObservedAt time.Time
}

// SupplierGroupChangeEvent 表示一次供应商分组同步产生的汇总变化事件。
type SupplierGroupChangeEvent struct {
	ID           int64
	ProviderID   int64
	ProviderCode string
	ProviderName string
	SyncRunID    *int64
	EventType    string
	Changes      SupplierProviderGroupChangeSummary
	ChangeCount  int
	ObservedAt   time.Time
	CreatedAt    time.Time
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
	// GroupID 用于按分组维度筛选投递记录（分组账号异常事件没有供应商）。
	GroupID   int64
	EventType string
	Status    string
	Page      int
	PageSize  int
}

type SupplierNotificationDeliveryListResult struct {
	Items    []SupplierNotificationDelivery `json:"items"`
	Total    int64                          `json:"total"`
	Page     int                            `json:"page"`
	PageSize int                            `json:"page_size"`
}

type SupplierNotificationDeliveryRecord struct {
	ID                 int64
	ChannelID          int64
	ChannelName        string
	EventID            *int64
	GroupChangeEventID *int64
	ProviderID         int64
	ProviderName       string
	// GroupID 非空表示这是分组维度的事件（分组账号异常），此时 ProviderID 为 0、落库为 NULL。
	// GroupName 与 ProviderName 一样只是查询期投影出来的展示名，不参与发送逻辑。
	GroupID       *int64
	GroupName     string
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
	// ListMatchingGroupSubscriptions 按分组维度匹配订阅：命中「指定分组」或「通配分组」（group_id IS NULL）。
	ListMatchingGroupSubscriptions(ctx context.Context, channelID int64, groupID int64, eventType string) ([]SupplierNotificationSubscription, error)
	ClaimCooldown(ctx context.Context, channelID, providerID int64, eventType string, now, expiresAt time.Time) (bool, error)
	// ClaimGroupCooldown 是分组维度的冷却占用，独立于按供应商的 ClaimCooldown。
	ClaimGroupCooldown(ctx context.Context, channelID, groupID int64, eventType string, now, expiresAt time.Time) (bool, error)
	CreateGroupChangeEvent(ctx context.Context, event *SupplierGroupChangeEvent) error
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

// SupplierGroupChangeNotifier 负责发送供应商分组变化通知。
type SupplierGroupChangeNotifier interface {
	DispatchGroupChanged(ctx context.Context, event SupplierGroupChangeEvent) error
}

// SupplierGroupAccountAbnormalNotifier 负责发送「分组账号异常」通知。
// 由分组择优任务在每轮检测后调用，走与其它供应商通知相同的渠道、投递与重试链路。
type SupplierGroupAccountAbnormalNotifier interface {
	DispatchGroupAccountAbnormal(ctx context.Context, event SupplierGroupAccountAbnormalEvent) error
}

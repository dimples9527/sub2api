package service

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/shopspring/decimal"
	"net/url"
	"strings"
	"time"
)

type SupplierNotificationService struct {
	repo      SupplierNotificationRepository
	encryptor SecretEncryptor
	sender    SupplierNotificationSender
}

func NewSupplierNotificationService(repo SupplierNotificationRepository, encryptor SecretEncryptor, sender SupplierNotificationSender) *SupplierNotificationService {
	return &SupplierNotificationService{repo: repo, encryptor: encryptor, sender: sender}
}

func (s *SupplierNotificationService) ListChannels(ctx context.Context) ([]SupplierNotificationChannelView, error) {
	if s == nil || s.repo == nil {
		return nil, ErrSupplierNotificationInvalid
	}
	channels, err := s.repo.ListChannels(ctx)
	if err != nil {
		return nil, err
	}
	views := make([]SupplierNotificationChannelView, 0, len(channels))
	for i := range channels {
		view, viewErr := s.channelView(&channels[i])
		if viewErr != nil {
			return nil, viewErr
		}
		views = append(views, view)
	}
	return views, nil
}

func (s *SupplierNotificationService) GetChannel(ctx context.Context, id int64) (*SupplierNotificationChannelView, error) {
	channel, err := s.getChannel(ctx, id)
	if err != nil {
		return nil, err
	}
	view, err := s.channelView(channel)
	if err != nil {
		return nil, err
	}
	return &view, nil
}

func (s *SupplierNotificationService) SaveChannel(ctx context.Context, id int64, input SupplierNotificationChannelInput) (*SupplierNotificationChannelView, error) {
	if s == nil || s.repo == nil || s.encryptor == nil || id < 0 {
		return nil, ErrSupplierNotificationInvalid
	}
	var existing *SupplierNotificationChannel
	var err error
	if id > 0 {
		existing, err = s.repo.GetChannel(ctx, id)
		if err != nil {
			return nil, err
		}
	}
	name := strings.TrimSpace(input.Name)
	if name == "" && existing != nil {
		name = existing.Name
	}
	if name == "" || len(name) > 128 {
		return nil, ErrSupplierNotificationInvalid
	}
	channelType := strings.ToLower(strings.TrimSpace(input.ChannelType))
	if channelType == "" && existing != nil {
		channelType = existing.ChannelType
	}
	if channelType != SupplierNotificationChannelFeishu && channelType != SupplierNotificationChannelEmail {
		return nil, ErrSupplierNotificationInvalid
	}
	configJSON, err := s.mergeChannelConfig(existing, channelType, input)
	if err != nil {
		return nil, err
	}
	proxyJSON, err := s.mergeProxyConfig(existing, input.Proxy)
	if err != nil {
		return nil, err
	}
	configEncrypted, err := s.encryptJSON(configJSON)
	if err != nil {
		return nil, err
	}
	proxyEncrypted, err := s.encryptJSON(proxyJSON)
	if err != nil {
		return nil, err
	}
	channel := SupplierNotificationChannel{
		ID:              id,
		Name:            name,
		ChannelType:     channelType,
		Enabled:         input.Enabled,
		ConfigEncrypted: configEncrypted,
		ProxyEncrypted:  proxyEncrypted,
	}
	if existing != nil {
		channel.CreatedAt = existing.CreatedAt
	}
	if err := s.repo.SaveChannel(ctx, &channel); err != nil {
		return nil, err
	}
	view, err := s.channelView(&channel)
	if err != nil {
		return nil, err
	}
	return &view, nil
}

func (s *SupplierNotificationService) DeleteChannel(ctx context.Context, id int64) error {
	if s == nil || s.repo == nil || id <= 0 {
		return ErrSupplierNotificationInvalid
	}
	return s.repo.DeleteChannel(ctx, id)
}

func (s *SupplierNotificationService) ListSubscriptions(ctx context.Context, channelID int64) ([]SupplierNotificationSubscription, error) {
	if s == nil || s.repo == nil || channelID < 0 {
		return nil, ErrSupplierNotificationInvalid
	}
	return s.repo.ListSubscriptions(ctx, channelID)
}

func (s *SupplierNotificationService) GetSubscription(ctx context.Context, id int64) (*SupplierNotificationSubscription, error) {
	if s == nil || s.repo == nil || id <= 0 {
		return nil, ErrSupplierNotificationInvalid
	}
	return s.repo.GetSubscription(ctx, id)
}

func (s *SupplierNotificationService) SaveSubscription(ctx context.Context, id int64, input SupplierNotificationSubscriptionInput) (*SupplierNotificationSubscription, error) {
	if s == nil || s.repo == nil || id < 0 || input.ChannelID <= 0 {
		return nil, ErrSupplierNotificationInvalid
	}
	if !isValidSupplierNotificationEventType(input.EventType) {
		return nil, ErrSupplierNotificationInvalid
	}
	if input.ProviderID != nil && *input.ProviderID <= 0 {
		return nil, ErrSupplierNotificationInvalid
	}
	if _, err := s.repo.GetChannel(ctx, input.ChannelID); err != nil {
		return nil, err
	}
	subscription := &SupplierNotificationSubscription{
		ID:         id,
		ChannelID:  input.ChannelID,
		ProviderID: input.ProviderID,
		EventType:  input.EventType,
		Enabled:    input.Enabled,
	}
	if err := s.repo.UpsertSubscription(ctx, subscription); err != nil {
		return nil, err
	}
	if subscription.ID > 0 {
		if saved, err := s.repo.GetSubscription(ctx, subscription.ID); err == nil && saved != nil {
			return saved, nil
		}
	}
	return subscription, nil
}

func (s *SupplierNotificationService) DeleteSubscription(ctx context.Context, id int64) error {
	if s == nil || s.repo == nil || id <= 0 {
		return ErrSupplierNotificationInvalid
	}
	return s.repo.DeleteSubscription(ctx, id)
}

func (s *SupplierNotificationService) TestChannel(ctx context.Context, id int64) (SupplierNotificationSendResult, error) {
	if s == nil || s.repo == nil || s.sender == nil || id <= 0 {
		return SupplierNotificationSendResult{}, ErrSupplierNotificationInvalid
	}
	channel, err := s.repo.GetChannel(ctx, id)
	if err != nil {
		return SupplierNotificationSendResult{}, err
	}
	now := time.Now()
	payload := SupplierNotificationEventPayload{
		ProviderID:   0,
		ProviderCode: "test",
		ProviderName: "测试供应商",
		EventType:    SupplierBalanceAlertEventLow,
		Status:       SupplierBalanceAlertEventActive,
		Balance:      decimal.Zero,
		Threshold:    decimal.NewFromInt(1),
		ObservedAt:   now,
		Test:         true,
	}
	return s.sender.Send(ctx, *channel, payload)
}

func (s *SupplierNotificationService) ListDeliveries(ctx context.Context, params SupplierNotificationDeliveryListParams) (SupplierNotificationDeliveryListResult, error) {
	if s == nil || s.repo == nil {
		return SupplierNotificationDeliveryListResult{}, ErrSupplierNotificationInvalid
	}
	return s.repo.ListDeliveries(ctx, params)
}

func (s *SupplierNotificationService) GetDelivery(ctx context.Context, id int64) (*SupplierNotificationDelivery, error) {
	if s == nil || s.repo == nil || id <= 0 {
		return nil, ErrSupplierNotificationInvalid
	}
	record, err := s.repo.GetDelivery(ctx, id)
	if err != nil {
		return nil, err
	}
	view := supplierNotificationDeliveryViewFromRecord(record)
	return &view, nil
}

func (s *SupplierNotificationService) ListAttempts(ctx context.Context, deliveryID int64) ([]SupplierNotificationDeliveryAttempt, error) {
	if s == nil || s.repo == nil || deliveryID <= 0 {
		return nil, ErrSupplierNotificationInvalid
	}
	return s.repo.ListAttempts(ctx, deliveryID)
}

func (s *SupplierNotificationService) getChannel(ctx context.Context, id int64) (*SupplierNotificationChannel, error) {
	if s == nil || s.repo == nil || id <= 0 {
		return nil, ErrSupplierNotificationInvalid
	}
	return s.repo.GetChannel(ctx, id)
}

func (s *SupplierNotificationService) mergeChannelConfig(existing *SupplierNotificationChannel, channelType string, input SupplierNotificationChannelInput) ([]byte, error) {
	switch channelType {
	case SupplierNotificationChannelFeishu:
		var config SupplierNotificationFeishuConfig
		if existing != nil && existing.ChannelType == channelType && existing.ConfigEncrypted != "" {
			if err := s.decryptInto(existing.ConfigEncrypted, &config); err != nil {
				return nil, err
			}
		}
		if input.Feishu != nil {
			if strings.TrimSpace(input.Feishu.WebhookURL) != "" {
				config.WebhookURL = strings.TrimSpace(input.Feishu.WebhookURL)
			}
			if input.Feishu.Secret != "" {
				config.Secret = input.Feishu.Secret
			}
		}
		if err := validateFeishuConfig(config); err != nil {
			return nil, err
		}
		return json.Marshal(config)
	case SupplierNotificationChannelEmail:
		var config SupplierNotificationEmailConfig
		if existing != nil && existing.ChannelType == channelType && existing.ConfigEncrypted != "" {
			if err := s.decryptInto(existing.ConfigEncrypted, &config); err != nil {
				return nil, err
			}
		}
		if input.Email != nil {
			if input.Email.Host != "" {
				config.Host = strings.TrimSpace(input.Email.Host)
			}
			if input.Email.Port > 0 {
				config.Port = input.Email.Port
			}
			if input.Email.Username != "" {
				config.Username = input.Email.Username
			}
			if input.Email.Password != "" {
				config.Password = input.Email.Password
			}
			if input.Email.From != "" {
				config.From = strings.TrimSpace(input.Email.From)
			}
			if input.Email.To != nil {
				config.To = append([]string(nil), input.Email.To...)
			}
			config.StartTLS = input.Email.StartTLS
		}
		if err := validateEmailConfig(config); err != nil {
			return nil, err
		}
		return json.Marshal(config)
	default:
		return nil, ErrSupplierNotificationInvalid
	}
}

func (s *SupplierNotificationService) mergeProxyConfig(existing *SupplierNotificationChannel, input *SupplierNotificationProxyConfig) ([]byte, error) {
	var config SupplierNotificationProxyConfig
	configured := false
	if existing != nil && existing.ProxyEncrypted != "" {
		if err := s.decryptInto(existing.ProxyEncrypted, &config); err != nil {
			return nil, err
		}
		configured = config.URL != ""
	}
	if input != nil {
		if strings.TrimSpace(input.URL) != "" {
			config.URL = strings.TrimSpace(input.URL)
			configured = true
		}
		if input.Username != "" {
			config.Username = input.Username
		}
		if input.Password != "" {
			config.Password = input.Password
		}
	}
	if !configured {
		return nil, nil
	}
	if err := validateProxyConfig(config); err != nil {
		return nil, err
	}
	return json.Marshal(config)
}

func (s *SupplierNotificationService) channelView(channel *SupplierNotificationChannel) (SupplierNotificationChannelView, error) {
	if channel == nil {
		return SupplierNotificationChannelView{}, ErrSupplierNotificationChannelNotFound
	}
	view := SupplierNotificationChannelView{
		ID:          channel.ID,
		Name:        channel.Name,
		ChannelType: channel.ChannelType,
		Enabled:     channel.Enabled,
		CreatedAt:   channel.CreatedAt,
		UpdatedAt:   channel.UpdatedAt,
	}
	switch channel.ChannelType {
	case SupplierNotificationChannelFeishu:
		var config SupplierNotificationFeishuConfig
		if channel.ConfigEncrypted != "" {
			if err := s.decryptInto(channel.ConfigEncrypted, &config); err != nil {
				return view, err
			}
		}
		view.FeishuWebhookConfigured = strings.TrimSpace(config.WebhookURL) != ""
		view.FeishuSecretConfigured = config.Secret != ""
		view.Configured = view.FeishuWebhookConfigured
	case SupplierNotificationChannelEmail:
		var config SupplierNotificationEmailConfig
		if channel.ConfigEncrypted != "" {
			if err := s.decryptInto(channel.ConfigEncrypted, &config); err != nil {
				return view, err
			}
		}
		view.EmailHost = config.Host
		view.EmailPort = config.Port
		view.EmailUsername = config.Username
		view.EmailFrom = config.From
		view.EmailTo = append([]string(nil), config.To...)
		view.EmailStartTLS = config.StartTLS
		view.Configured = config.Host != "" && config.Port > 0 && config.From != "" && len(config.To) > 0
	default:
		return view, ErrSupplierNotificationInvalid
	}
	if channel.ProxyEncrypted != "" {
		var proxy SupplierNotificationProxyConfig
		if err := s.decryptInto(channel.ProxyEncrypted, &proxy); err != nil {
			return view, err
		}
		if proxy.URL != "" {
			parsed, parseErr := url.Parse(proxy.URL)
			if parseErr == nil {
				parsed.User = nil
				view.ProxyURL = parsed.String()
			}
			view.ProxyConfigured = true
		}
	}
	return view, nil
}

func (s *SupplierNotificationService) decryptInto(ciphertext string, target any) error {
	if s == nil || s.encryptor == nil {
		return ErrSupplierNotificationInvalid
	}
	plaintext, err := s.encryptor.Decrypt(ciphertext)
	if err != nil {
		return fmt.Errorf("解密供应商通知配置失败: %w", err)
	}
	if err := json.Unmarshal([]byte(plaintext), target); err != nil {
		return fmt.Errorf("解析供应商通知配置失败: %w", err)
	}
	return nil
}

func (s *SupplierNotificationService) encryptJSON(raw []byte) (string, error) {
	if len(raw) == 0 {
		return "", nil
	}
	value, err := s.encryptor.Encrypt(string(raw))
	if err != nil {
		return "", fmt.Errorf("加密供应商通知配置失败: %w", err)
	}
	return value, nil
}

func supplierNotificationDeliveryViewFromRecord(record *SupplierNotificationDeliveryRecord) SupplierNotificationDelivery {
	if record == nil {
		return SupplierNotificationDelivery{}
	}
	return SupplierNotificationDelivery{
		ID:            record.ID,
		ChannelID:     record.ChannelID,
		ChannelName:   record.ChannelName,
		EventID:       record.EventID,
		ProviderID:    record.ProviderID,
		ProviderName:  record.ProviderName,
		EventType:     record.EventType,
		Status:        record.Status,
		AttemptCount:  record.AttemptCount,
		NextAttemptAt: record.NextAttemptAt,
		LastError:     record.LastError,
		SentAt:        record.SentAt,
		CreatedAt:     record.CreatedAt,
		UpdatedAt:     record.UpdatedAt,
	}
}

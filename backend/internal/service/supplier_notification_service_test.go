package service

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
)

type supplierNotificationServiceSenderStub struct {
	calls   int
	channel SupplierNotificationChannel
	payload SupplierNotificationEventPayload
}

func (s *supplierNotificationServiceSenderStub) Send(_ context.Context, channel SupplierNotificationChannel, payload SupplierNotificationEventPayload) (SupplierNotificationSendResult, error) {
	s.calls++
	s.channel = channel
	s.payload = payload
	return SupplierNotificationSendResult{HTTPStatus: 200}, nil
}

func TestSupplierNotificationServiceMasksSensitiveChannelConfiguration(t *testing.T) {
	repo := newSupplierNotificationRepoStub()
	encryptor := supplierNotificationEncryptorStub{}
	svc := NewSupplierNotificationService(repo, encryptor, &supplierNotificationServiceSenderStub{})

	view, err := svc.SaveChannel(context.Background(), 0, SupplierNotificationChannelInput{
		Name:        "飞书余额通知",
		ChannelType: SupplierNotificationChannelFeishu,
		Enabled:     true,
		Feishu:      &SupplierNotificationFeishuConfig{WebhookURL: "https://example.com/hook", Secret: "top-secret"},
		Proxy:       &SupplierNotificationProxyConfig{URL: "http://proxy.example.com:8080", Username: "proxy-user", Password: "proxy-pass"},
	})
	if err != nil {
		t.Fatalf("SaveChannel returned error: %v", err)
	}
	if !view.Configured || !view.FeishuWebhookConfigured || !view.FeishuSecretConfigured || !view.ProxyConfigured || view.ProxyURL == "" {
		t.Fatalf("view = %+v", view)
	}
	if viewJSON, _ := json.Marshal(view); string(viewJSON) == "" || supplierNotificationContainsAny(string(viewJSON), "top-secret", "proxy-pass", "example.com/hook") {
		t.Fatalf("sensitive data leaked in view: %s", viewJSON)
	}
	stored := repo.channels[view.ID]
	if stored.ConfigEncrypted == "" || stored.ProxyEncrypted == "" {
		t.Fatalf("stored channel = %+v", stored)
	}
}

func TestSupplierNotificationServicePreservesSecretWhenUpdateOmitsIt(t *testing.T) {
	repo := newSupplierNotificationRepoStub()
	encryptor := supplierNotificationEncryptorStub{}
	svc := NewSupplierNotificationService(repo, encryptor, &supplierNotificationServiceSenderStub{})
	created, err := svc.SaveChannel(context.Background(), 0, SupplierNotificationChannelInput{
		Name:        "邮件通知",
		ChannelType: SupplierNotificationChannelEmail,
		Email:       &SupplierNotificationEmailConfig{Host: "smtp.example.com", Port: 587, Username: "user", Password: "old-password", From: "from@example.com", To: []string{"to@example.com"}, StartTLS: true},
	})
	if err != nil {
		t.Fatalf("create channel: %v", err)
	}
	updated, err := svc.SaveChannel(context.Background(), created.ID, SupplierNotificationChannelInput{
		Name:        "邮件通知-更新",
		ChannelType: SupplierNotificationChannelEmail,
		Enabled:     true,
		Email:       &SupplierNotificationEmailConfig{Host: "smtp2.example.com", Port: 587, Username: "user", From: "from@example.com", To: []string{"to@example.com"}, StartTLS: true},
	})
	if err != nil {
		t.Fatalf("update channel: %v", err)
	}
	if updated.Name != "邮件通知-更新" || updated.EmailHost != "smtp2.example.com" || !updated.Configured {
		t.Fatalf("updated view = %+v", updated)
	}
	configJSON, err := encryptor.Decrypt(repo.channels[created.ID].ConfigEncrypted)
	if err != nil {
		t.Fatalf("decrypt stored config: %v", err)
	}
	var config SupplierNotificationEmailConfig
	if err := json.Unmarshal([]byte(configJSON), &config); err != nil {
		t.Fatalf("decode stored config: %v", err)
	}
	if config.Password != "old-password" {
		t.Fatalf("password = %q, want preserved password", config.Password)
	}
}

func TestSupplierNotificationServiceTestChannelDoesNotCreateDelivery(t *testing.T) {
	repo := newSupplierNotificationRepoStub()
	repo.channels[1] = SupplierNotificationChannel{ID: 1, Name: "飞书", ChannelType: SupplierNotificationChannelFeishu, ConfigEncrypted: "enc:{}"}
	sender := &supplierNotificationServiceSenderStub{}
	svc := NewSupplierNotificationService(repo, supplierNotificationEncryptorStub{}, sender)

	if _, err := svc.TestChannel(context.Background(), 1); err != nil {
		t.Fatalf("TestChannel returned error: %v", err)
	}
	if sender.calls != 1 || len(repo.deliveries) != 0 || !sender.payload.Test {
		t.Fatalf("calls/deliveries/payload = %d/%d/%+v", sender.calls, len(repo.deliveries), sender.payload)
	}
}

func TestSupplierNotificationServiceValidatesSubscriptionEventType(t *testing.T) {
	repo := newSupplierNotificationRepoStub()
	repo.channels[1] = SupplierNotificationChannel{ID: 1, Name: "飞书", ChannelType: SupplierNotificationChannelFeishu}
	svc := NewSupplierNotificationService(repo, supplierNotificationEncryptorStub{}, &supplierNotificationServiceSenderStub{})

	_, err := svc.SaveSubscription(context.Background(), 0, SupplierNotificationSubscriptionInput{ChannelID: 1, EventType: "unknown", Enabled: true})
	if err == nil {
		t.Fatal("SaveSubscription returned nil error for invalid event type")
	}
}

func supplierNotificationContainsAny(value string, parts ...string) bool {
	for _, part := range parts {
		if len(part) > 0 && strings.Contains(value, part) {
			return true
		}
	}
	return false
}

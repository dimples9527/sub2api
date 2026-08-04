package service

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/shopspring/decimal"
)

type supplierNotificationEncryptorStub struct{}

func (supplierNotificationEncryptorStub) Encrypt(plaintext string) (string, error) {
	return "enc:" + plaintext, nil
}

func (supplierNotificationEncryptorStub) Decrypt(ciphertext string) (string, error) {
	return strings.TrimPrefix(ciphertext, "enc:"), nil
}

func TestSupplierNotificationSenderSendsSignedFeishuMessage(t *testing.T) {
	requests := make(chan map[string]any, 1)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Errorf("read body: %v", err)
			return
		}
		var payload map[string]any
		if err := json.Unmarshal(body, &payload); err != nil {
			t.Errorf("decode body: %v", err)
			return
		}
		requests <- payload
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"code":0,"msg":"ok"}`))
	}))
	defer server.Close()

	encryptor := supplierNotificationEncryptorStub{}
	config, _ := json.Marshal(SupplierNotificationFeishuConfig{WebhookURL: server.URL, Secret: "secret"})
	sender := NewSupplierNotificationSender(encryptor)
	channel := SupplierNotificationChannel{ID: 1, ChannelType: SupplierNotificationChannelFeishu, ConfigEncrypted: "enc:" + string(config)}
	payload := SupplierNotificationEventPayload{ProviderID: 7, ProviderName: "供应商", EventType: SupplierBalanceAlertEventLow, Balance: decimal.NewFromInt(1), Threshold: decimal.NewFromInt(2), ObservedAt: time.Now()}

	result, err := sender.Send(context.Background(), channel, payload)
	if err != nil {
		t.Fatalf("Send returned error: %v", err)
	}
	if result.HTTPStatus != http.StatusOK {
		t.Fatalf("HTTP status = %d, want %d", result.HTTPStatus, http.StatusOK)
	}
	select {
	case body := <-requests:
		if body["msg_type"] != "text" || body["timestamp"] == nil || body["sign"] == nil {
			t.Fatalf("request body = %+v", body)
		}
		timestamp, ok := body["timestamp"].(string)
		if !ok || timestamp == "" {
			t.Fatalf("timestamp = %#v, want non-empty string", body["timestamp"])
		}
		stringToSign := timestamp + "\nsecret"
		mac := hmac.New(sha256.New, []byte(stringToSign))
		expectedSign := base64.StdEncoding.EncodeToString(mac.Sum(nil))
		if body["sign"] != expectedSign {
			t.Fatalf("sign = %v, want %s", body["sign"], expectedSign)
		}
	case <-time.After(time.Second):
		t.Fatal("did not receive Feishu request")
	}
}

func TestSupplierNotificationSenderRejectsFeishuBusinessError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"code":19001,"msg":"invalid"}`))
	}))
	defer server.Close()

	config, _ := json.Marshal(SupplierNotificationFeishuConfig{WebhookURL: server.URL})
	sender := NewSupplierNotificationSender(supplierNotificationEncryptorStub{})
	channel := SupplierNotificationChannel{ChannelType: SupplierNotificationChannelFeishu, ConfigEncrypted: "enc:" + string(config)}
	_, err := sender.Send(context.Background(), channel, SupplierNotificationEventPayload{ProviderID: 1, EventType: SupplierBalanceAlertEventLow})
	if err == nil || !strings.Contains(err.Error(), "19001") {
		t.Fatalf("error = %v, want Feishu business error", err)
	}
}

func TestSupplierNotificationSenderDoesNotLeakSecretsInError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadGateway)
		_, _ = w.Write([]byte(`{"msg":"secret"}`))
	}))
	defer server.Close()

	config, _ := json.Marshal(SupplierNotificationFeishuConfig{WebhookURL: server.URL, Secret: "secret"})
	sender := NewSupplierNotificationSender(supplierNotificationEncryptorStub{})
	channel := SupplierNotificationChannel{ChannelType: SupplierNotificationChannelFeishu, ConfigEncrypted: "enc:" + string(config)}
	_, err := sender.Send(context.Background(), channel, SupplierNotificationEventPayload{ProviderID: 1, EventType: SupplierBalanceAlertEventLow})
	if err == nil || strings.Contains(err.Error(), "secret") {
		t.Fatalf("error leaks secret: %v", err)
	}
}

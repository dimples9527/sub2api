package service

import (
	"context"
	"database/sql"
	"fmt"
	"testing"

	dbent "github.com/Wei-Shaw/sub2api/ent"
	"github.com/Wei-Shaw/sub2api/ent/enttest"
	"github.com/stretchr/testify/require"

	"entgo.io/ent/dialect"
	entsql "entgo.io/ent/dialect/sql"
	_ "modernc.org/sqlite"
)

const testPaymentConfigEncryptionKey = "12345678901234567890123456789012"

func newPaymentConfigProviderTestService(t *testing.T) (*PaymentConfigService, *dbent.Client) {
	t.Helper()

	dsn := fmt.Sprintf("file:%s?mode=memory&cache=shared", t.Name())
	db, err := sql.Open("sqlite", dsn)
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })

	_, err = db.Exec("PRAGMA foreign_keys = ON")
	require.NoError(t, err)

	drv := entsql.OpenDB(dialect.SQLite, db)
	client := enttest.NewClient(t, enttest.WithOptions(dbent.Driver(drv)))
	t.Cleanup(func() { _ = client.Close() })

	return &PaymentConfigService{
		entClient:     client,
		encryptionKey: []byte(testPaymentConfigEncryptionKey),
	}, client
}

func createTestProviderInstance(t *testing.T, ctx context.Context, svc *PaymentConfigService, cfg map[string]string) *dbent.PaymentProviderInstance {
	t.Helper()

	enc, err := svc.encryptConfig(cfg)
	require.NoError(t, err)

	inst, err := svc.entClient.PaymentProviderInstance.Create().
		SetProviderKey("alipay").
		SetName("test-provider").
		SetConfig(enc).
		SetSupportedTypes("alipay").
		SetEnabled(true).
		SetSortOrder(1).
		SetLimits("").
		SetRefundEnabled(false).
		Save(ctx)
	require.NoError(t, err)
	return inst
}

func TestDecryptAndMaskConfig_MasksSensitiveFields(t *testing.T) {
	svc, _ := newPaymentConfigProviderTestService(t)

	enc, err := svc.encryptConfig(map[string]string{
		"appId":      "app-123",
		"privateKey": "secret-value",
		"notifyUrl":  "https://example.com/callback",
	})
	require.NoError(t, err)

	got, err := svc.decryptAndMaskConfig(enc)
	require.NoError(t, err)
	require.Equal(t, "app-123", got["appId"])
	require.Equal(t, "https://example.com/callback", got["notifyUrl"])
	require.Equal(t, maskedSensitiveConfigValue, got["privateKey"])
}

func TestHasSensitiveConfigChanges_IgnoresMaskedOrUnchangedValues(t *testing.T) {
	svc, _ := newPaymentConfigProviderTestService(t)
	ctx := context.Background()
	inst := createTestProviderInstance(t, ctx, svc, map[string]string{
		"appId":      "app-123",
		"privateKey": "secret-value",
	})

	changed, err := svc.hasSensitiveConfigChanges(ctx, int64(inst.ID), map[string]string{
		"privateKey": maskedSensitiveConfigValue,
	})
	require.NoError(t, err)
	require.False(t, changed)

	changed, err = svc.hasSensitiveConfigChanges(ctx, int64(inst.ID), map[string]string{
		"privateKey": "secret-value",
	})
	require.NoError(t, err)
	require.False(t, changed)

	changed, err = svc.hasSensitiveConfigChanges(ctx, int64(inst.ID), map[string]string{
		"privateKey": "new-secret",
	})
	require.NoError(t, err)
	require.True(t, changed)
}

func TestMergeConfig_IgnoresMaskedSensitiveValues(t *testing.T) {
	svc, _ := newPaymentConfigProviderTestService(t)
	ctx := context.Background()
	inst := createTestProviderInstance(t, ctx, svc, map[string]string{
		"appId":      "app-123",
		"privateKey": "secret-value",
		"notifyUrl":  "https://old.example.com/callback",
	})

	merged, err := svc.mergeConfig(ctx, int64(inst.ID), map[string]string{
		"privateKey": maskedSensitiveConfigValue,
		"notifyUrl":  "https://new.example.com/callback",
	})
	require.NoError(t, err)
	require.Equal(t, "secret-value", merged["privateKey"])
	require.Equal(t, "https://new.example.com/callback", merged["notifyUrl"])
	require.Equal(t, "app-123", merged["appId"])
}

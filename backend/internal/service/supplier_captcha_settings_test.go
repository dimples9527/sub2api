//go:build unit

package service

import (
	"context"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/Wei-Shaw/sub2api/internal/pkg/captcha"
	"github.com/stretchr/testify/require"
)

func TestGetSupplierCaptchaSettings_Defaults(t *testing.T) {
	repo := newMockSettingRepo()
	svc := NewSettingService(repo, &config.Config{})

	got, err := svc.GetSupplierCaptchaSettings(context.Background())
	require.NoError(t, err)
	require.Equal(t, captcha.ProviderTwoCaptcha, got.Provider)
	require.False(t, got.APIKeyConfigured)
	require.Equal(t, "", got.Endpoint)
	require.Equal(t, int64(0), got.CallTotal)
	require.Equal(t, int64(0), got.CallSuccess)
	require.Equal(t, int64(0), got.CallFailed)
	require.Equal(t, "", got.LastCalledAt)
}

func TestUpdateSupplierCaptchaSettings_KeepAPIKeyWhenEmpty(t *testing.T) {
	repo := newMockSettingRepo()
	svc := NewSettingService(repo, &config.Config{})

	_, err := svc.UpdateSupplierCaptchaSettings(context.Background(), &UpdateSupplierCaptchaSettingsInput{
		Provider: captcha.ProviderTwoCaptcha,
		APIKey:   "secret-key",
		Endpoint: "https://api.example.com",
	})
	require.NoError(t, err)

	got, err := svc.UpdateSupplierCaptchaSettings(context.Background(), &UpdateSupplierCaptchaSettingsInput{
		Provider: captcha.ProviderTwoCaptcha,
		APIKey:   "",
		Endpoint: "https://api.example.com/v2",
	})
	require.NoError(t, err)
	require.True(t, got.APIKeyConfigured)
	require.Equal(t, "https://api.example.com/v2", got.Endpoint)

	cfg, err := svc.loadSupplierCaptchaRuntimeConfig(context.Background())
	require.NoError(t, err)
	require.Equal(t, "secret-key", cfg.APIKey)
}

func TestUpdateSupplierCaptchaSettings_ClearAPIKey(t *testing.T) {
	repo := newMockSettingRepo()
	svc := NewSettingService(repo, &config.Config{})

	_, err := svc.UpdateSupplierCaptchaSettings(context.Background(), &UpdateSupplierCaptchaSettingsInput{
		Provider: captcha.ProviderTwoCaptcha,
		APIKey:   "secret-key",
	})
	require.NoError(t, err)

	got, err := svc.UpdateSupplierCaptchaSettings(context.Background(), &UpdateSupplierCaptchaSettingsInput{
		Provider:    captcha.ProviderTwoCaptcha,
		ClearAPIKey: true,
	})
	require.NoError(t, err)
	require.False(t, got.APIKeyConfigured)
}

func TestUpdateSupplierCaptchaSettings_RejectUnsupportedProvider(t *testing.T) {
	repo := newMockSettingRepo()
	svc := NewSettingService(repo, &config.Config{})

	_, err := svc.UpdateSupplierCaptchaSettings(context.Background(), &UpdateSupplierCaptchaSettingsInput{
		Provider: "capsolver",
	})
	require.Error(t, err)
}

func TestUpdateSupplierCaptchaSettings_AllowsYesCaptcha(t *testing.T) {
	repo := newMockSettingRepo()
	svc := NewSettingService(repo, &config.Config{})

	got, err := svc.UpdateSupplierCaptchaSettings(context.Background(), &UpdateSupplierCaptchaSettingsInput{
		Provider: captcha.ProviderYesCaptcha,
		APIKey:   "yes-secret",
		Endpoint: "https://captcha.example.com",
	})
	require.NoError(t, err)
	require.Equal(t, captcha.ProviderYesCaptcha, got.Provider)
	require.True(t, got.APIKeyConfigured)
	require.Equal(t, "https://captcha.example.com", got.Endpoint)
}

func TestUpdateSupplierCaptchaSettings_RequiresNewAPIKeyWhenProviderChanges(t *testing.T) {
	repo := newMockSettingRepo()
	svc := NewSettingService(repo, &config.Config{})

	_, err := svc.UpdateSupplierCaptchaSettings(context.Background(), &UpdateSupplierCaptchaSettingsInput{
		Provider: captcha.ProviderTwoCaptcha,
		APIKey:   "old-secret",
		Endpoint: "https://old-captcha.example.com",
	})
	require.NoError(t, err)

	_, err = svc.UpdateSupplierCaptchaSettings(context.Background(), &UpdateSupplierCaptchaSettingsInput{
		Provider: captcha.ProviderYesCaptcha,
		Endpoint: "https://old-captcha.example.com",
	})
	require.EqualError(t, err, "switching captcha provider requires a new api key")

	cfg, err := svc.loadSupplierCaptchaRuntimeConfig(context.Background())
	require.NoError(t, err)
	require.Equal(t, captcha.ProviderTwoCaptcha, cfg.Provider)
	require.Equal(t, "old-secret", cfg.APIKey)
	require.Equal(t, "https://old-captcha.example.com", cfg.Endpoint)
}

func TestUpdateSupplierCaptchaSettings_DoesNotReuseEndpointWhenProviderChanges(t *testing.T) {
	repo := newMockSettingRepo()
	svc := NewSettingService(repo, &config.Config{})

	_, err := svc.UpdateSupplierCaptchaSettings(context.Background(), &UpdateSupplierCaptchaSettingsInput{
		Provider: captcha.ProviderTwoCaptcha,
		APIKey:   "old-secret",
		Endpoint: "https://old-captcha.example.com",
	})
	require.NoError(t, err)

	got, err := svc.UpdateSupplierCaptchaSettings(context.Background(), &UpdateSupplierCaptchaSettingsInput{
		Provider: captcha.ProviderYesCaptcha,
		APIKey:   "new-secret",
		Endpoint: "https://old-captcha.example.com",
	})
	require.NoError(t, err)
	require.Equal(t, captcha.ProviderYesCaptcha, got.Provider)
	require.True(t, got.APIKeyConfigured)
	require.Equal(t, "", got.Endpoint)
}

func TestRecordSupplierCaptchaCall_IncrementsStats(t *testing.T) {
	repo := newMockSettingRepo()
	svc := NewSettingService(repo, &config.Config{})

	require.NoError(t, svc.RecordSupplierCaptchaCall(context.Background(), true))
	require.NoError(t, svc.RecordSupplierCaptchaCall(context.Background(), false))
	require.NoError(t, svc.RecordSupplierCaptchaCall(context.Background(), true))

	got, err := svc.GetSupplierCaptchaSettings(context.Background())
	require.NoError(t, err)
	require.Equal(t, int64(3), got.CallTotal)
	require.Equal(t, int64(2), got.CallSuccess)
	require.Equal(t, int64(1), got.CallFailed)
	require.NotEmpty(t, got.LastCalledAt)
}

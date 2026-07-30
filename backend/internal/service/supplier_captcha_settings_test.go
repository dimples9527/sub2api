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

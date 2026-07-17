//go:build unit

package service

import (
	"context"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/stretchr/testify/require"
)

func TestSettingServiceGetPublicSettingsUsesConfiguredLLMMonitorURL(t *testing.T) {
	svc := NewSettingService(&settingPublicRepoStub{values: map[string]string{}}, &config.Config{
		LLMMonitor: config.LLMMonitorConfig{StatusAPIURL: "https://example.com/status"},
	})

	settings, err := svc.GetPublicSettings(context.Background())
	require.NoError(t, err)
	require.Equal(t, "https://example.com/status", settings.LLMMonitorStatusAPIURL)
}

func TestSettingServiceGetPublicSettingsUsesConfiguredLLMMonitorTitle(t *testing.T) {
	svc := NewSettingService(&settingPublicRepoStub{values: map[string]string{}}, &config.Config{
		LLMMonitor: config.LLMMonitorConfig{Title: "Configured Monitor Title"},
	})

	settings, err := svc.GetPublicSettings(context.Background())
	require.NoError(t, err)
	require.Equal(t, "Configured Monitor Title", settings.LLMMonitorTitle)
}

func TestSettingServiceGetPublicSettingsUsesConfiguredLLMMonitorProviderURL(t *testing.T) {
	svc := NewSettingService(&settingPublicRepoStub{values: map[string]string{}}, &config.Config{
		LLMMonitor: config.LLMMonitorConfig{ProviderURL: "https://provider.example.com/"},
	})

	settings, err := svc.GetPublicSettings(context.Background())
	require.NoError(t, err)
	require.Equal(t, "https://provider.example.com/", settings.LLMMonitorProviderURL)
}

func TestSettingServiceGetPublicSettingsAllowsLLMMonitorOverrides(t *testing.T) {
	svc := NewSettingService(&settingPublicRepoStub{values: map[string]string{
		SettingKeyLLMMonitorStatusAPIURL: " https://override.example.com/api/status ",
		SettingKeyLLMMonitorTitle:        " Override Monitor Title ",
		SettingKeyLLMMonitorProviderURL:  " https://override.example.com/provider ",
	}}, &config.Config{
		LLMMonitor: config.LLMMonitorConfig{
			StatusAPIURL: "https://example.com/status",
			Title:        "Configured Monitor Title",
			ProviderURL:  "https://provider.example.com/",
		},
	})

	settings, err := svc.GetPublicSettings(context.Background())
	require.NoError(t, err)
	require.Equal(t, "https://override.example.com/api/status", settings.LLMMonitorStatusAPIURL)
	require.Equal(t, "Override Monitor Title", settings.LLMMonitorTitle)
	require.Equal(t, "https://override.example.com/provider", settings.LLMMonitorProviderURL)
}

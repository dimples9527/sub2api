package service

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/stretchr/testify/require"
)

type llmMonitorRepoStub struct {
	values map[string]string
}

type llmMonitorDefaultsRepoStub struct {
	*llmMonitorRepoStub
	defaults map[string]string
}

func (s *llmMonitorRepoStub) Get(context.Context, string) (*Setting, error) {
	panic("unexpected Get call")
}

func (s *llmMonitorRepoStub) GetValue(context.Context, string) (string, error) {
	panic("unexpected GetValue call")
}

func (s *llmMonitorRepoStub) Set(context.Context, string, string) error {
	panic("unexpected Set call")
}

func (s *llmMonitorRepoStub) GetMultiple(_ context.Context, keys []string) (map[string]string, error) {
	values := make(map[string]string, len(keys))
	for _, key := range keys {
		if value, ok := s.values[key]; ok {
			values[key] = value
		}
	}
	return values, nil
}

func (s *llmMonitorRepoStub) SetMultiple(context.Context, map[string]string) error {
	panic("unexpected SetMultiple call")
}

func (s *llmMonitorRepoStub) GetAll(context.Context) (map[string]string, error) {
	panic("unexpected GetAll call")
}

func (s *llmMonitorRepoStub) Delete(context.Context, string) error {
	panic("unexpected Delete call")
}

func (s *llmMonitorDefaultsRepoStub) GetValue(context.Context, string) (string, error) {
	return "", ErrSettingNotFound
}

func (s *llmMonitorDefaultsRepoStub) SetMultiple(_ context.Context, values map[string]string) error {
	s.defaults = values
	return nil
}

func TestGetLLMMonitorSettingsUsesConfigFallback(t *testing.T) {
	svc := NewSettingService(&llmMonitorRepoStub{values: map[string]string{}}, &config.Config{
		LLMMonitor: config.LLMMonitorConfig{
			StatusAPIURL: " https://status.example.com/api/status ",
			Title:        " Monitor Title ",
			ProviderURL:  " https://provider.example.com/ ",
		},
	})

	settings, err := svc.GetLLMMonitorSettings(context.Background())
	require.NoError(t, err)
	require.Equal(t, "https://status.example.com/api/status", settings.StatusAPIURL)
	require.Equal(t, "Monitor Title", settings.Title)
	require.Equal(t, "https://provider.example.com/", settings.ProviderURL)
}

func TestGetLLMMonitorSettingsPrefersDatabaseOverrides(t *testing.T) {
	svc := NewSettingService(&llmMonitorRepoStub{values: map[string]string{
		SettingKeyLLMMonitorStatusAPIURL: " https://override.example.com/status ",
		SettingKeyLLMMonitorTitle:        " Override Title ",
		SettingKeyLLMMonitorProviderURL:  " https://override.example.com/ ",
	}}, &config.Config{
		LLMMonitor: config.LLMMonitorConfig{
			StatusAPIURL: "https://config.example.com/status",
			Title:        "Config Title",
			ProviderURL:  "https://config.example.com/",
		},
	})

	settings, err := svc.GetLLMMonitorSettings(context.Background())
	require.NoError(t, err)
	require.Equal(t, "https://override.example.com/status", settings.StatusAPIURL)
	require.Equal(t, "Override Title", settings.Title)
	require.Equal(t, "https://override.example.com/", settings.ProviderURL)
}

func TestGetPublicSettingsIncludesResolvedLLMMonitorSettings(t *testing.T) {
	svc := NewSettingService(&llmMonitorRepoStub{values: map[string]string{
		SettingKeyLLMMonitorTitle: " Database Monitor Title ",
	}}, &config.Config{
		LLMMonitor: config.LLMMonitorConfig{
			StatusAPIURL: " https://status.example.com/api/status ",
			Title:        "Config Monitor Title",
			ProviderURL:  " https://provider.example.com/ ",
		},
	})

	settings, err := svc.GetPublicSettings(context.Background())
	require.NoError(t, err)
	require.Equal(t, "https://status.example.com/api/status", settings.LLMMonitorStatusAPIURL)
	require.Equal(t, "Database Monitor Title", settings.LLMMonitorTitle)
	require.Equal(t, "https://provider.example.com/", settings.LLMMonitorProviderURL)
}

func TestPublicSettingsInjectionIncludesLLMMonitorSettings(t *testing.T) {
	svc := NewSettingService(&llmMonitorRepoStub{values: map[string]string{}}, &config.Config{
		LLMMonitor: config.LLMMonitorConfig{
			StatusAPIURL: "https://status.example.com/api/status",
			Title:        "Monitor Title",
			ProviderURL:  "https://provider.example.com/",
		},
	})

	payload, err := svc.GetPublicSettingsForInjection(context.Background())
	require.NoError(t, err)
	encoded, err := json.Marshal(payload)
	require.NoError(t, err)

	var values map[string]any
	require.NoError(t, json.Unmarshal(encoded, &values))
	require.Equal(t, "https://status.example.com/api/status", values["llm_monitor_status_api_url"])
	require.Equal(t, "Monitor Title", values["llm_monitor_title"])
	require.Equal(t, "https://provider.example.com/", values["llm_monitor_provider_url"])
}

func TestInitializeDefaultSettingsIncludesLLMMonitorKeys(t *testing.T) {
	repo := &llmMonitorDefaultsRepoStub{
		llmMonitorRepoStub: &llmMonitorRepoStub{values: map[string]string{}},
	}
	svc := NewSettingService(repo, &config.Config{})

	require.NoError(t, svc.InitializeDefaultSettings(context.Background()))
	for _, key := range []string{
		SettingKeyLLMMonitorStatusAPIURL,
		SettingKeyLLMMonitorTitle,
		SettingKeyLLMMonitorProviderURL,
	} {
		value, ok := repo.defaults[key]
		require.True(t, ok, "missing default for %s", key)
		require.Empty(t, value)
	}
}

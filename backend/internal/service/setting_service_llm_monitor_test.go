package service

import (
	"context"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/config"
)

type llmMonitorSettingRepoStub struct {
	values map[string]string
}

func (s *llmMonitorSettingRepoStub) Get(ctx context.Context, key string) (*Setting, error) {
	t := s.values[key]
	return &Setting{Key: key, Value: t}, nil
}

func (s *llmMonitorSettingRepoStub) GetValue(ctx context.Context, key string) (string, error) {
	return s.values[key], nil
}

func (s *llmMonitorSettingRepoStub) Set(ctx context.Context, key, value string) error {
	s.values[key] = value
	return nil
}

func (s *llmMonitorSettingRepoStub) GetMultiple(ctx context.Context, keys []string) (map[string]string, error) {
	out := make(map[string]string, len(keys))
	for _, key := range keys {
		if value, ok := s.values[key]; ok {
			out[key] = value
		}
	}
	return out, nil
}

func (s *llmMonitorSettingRepoStub) SetMultiple(ctx context.Context, settings map[string]string) error {
	for key, value := range settings {
		s.values[key] = value
	}
	return nil
}

func (s *llmMonitorSettingRepoStub) GetAll(ctx context.Context) (map[string]string, error) {
	out := make(map[string]string, len(s.values))
	for key, value := range s.values {
		out[key] = value
	}
	return out, nil
}

func (s *llmMonitorSettingRepoStub) Delete(ctx context.Context, key string) error {
	delete(s.values, key)
	return nil
}

func TestSettingServiceGetPublicSettingsUsesConfiguredLLMMonitorURL(t *testing.T) {
	svc := NewSettingService(&llmMonitorSettingRepoStub{values: map[string]string{}}, &config.Config{
		LLMMonitor: config.LLMMonitorConfig{StatusAPIURL: "https://example.com/status"},
	})

	settings, err := svc.GetPublicSettings(context.Background())
	if err != nil {
		t.Fatalf("GetPublicSettings() error = %v", err)
	}
	if settings.LLMMonitorStatusAPIURL != "https://example.com/status" {
		t.Fatalf("LLMMonitorStatusAPIURL = %q", settings.LLMMonitorStatusAPIURL)
	}
}

func TestSettingServiceGetPublicSettingsUsesConfiguredLLMMonitorTitle(t *testing.T) {
	svc := NewSettingService(&llmMonitorSettingRepoStub{values: map[string]string{}}, &config.Config{
		LLMMonitor: config.LLMMonitorConfig{Title: "Configured Monitor Title"},
	})

	settings, err := svc.GetPublicSettings(context.Background())
	if err != nil {
		t.Fatalf("GetPublicSettings() error = %v", err)
	}
	if settings.LLMMonitorTitle != "Configured Monitor Title" {
		t.Fatalf("LLMMonitorTitle = %q", settings.LLMMonitorTitle)
	}
}

func TestSettingServiceGetPublicSettingsUsesConfiguredLLMMonitorProviderURL(t *testing.T) {
	svc := NewSettingService(&llmMonitorSettingRepoStub{values: map[string]string{}}, &config.Config{
		LLMMonitor: config.LLMMonitorConfig{ProviderURL: "https://provider.example.com/"},
	})

	settings, err := svc.GetPublicSettings(context.Background())
	if err != nil {
		t.Fatalf("GetPublicSettings() error = %v", err)
	}
	if settings.LLMMonitorProviderURL != "https://provider.example.com/" {
		t.Fatalf("LLMMonitorProviderURL = %q", settings.LLMMonitorProviderURL)
	}
}

func TestSettingServiceGetPublicSettingsAllowsLLMMonitorURLOverride(t *testing.T) {
	svc := NewSettingService(&llmMonitorSettingRepoStub{values: map[string]string{
		SettingKeyLLMMonitorStatusAPIURL: " https://override.example.com/api/status ",
	}}, &config.Config{
		LLMMonitor: config.LLMMonitorConfig{StatusAPIURL: "https://example.com/status"},
	})

	settings, err := svc.GetPublicSettings(context.Background())
	if err != nil {
		t.Fatalf("GetPublicSettings() error = %v", err)
	}
	if settings.LLMMonitorStatusAPIURL != "https://override.example.com/api/status" {
		t.Fatalf("LLMMonitorStatusAPIURL = %q", settings.LLMMonitorStatusAPIURL)
	}
}

func TestSettingServiceGetPublicSettingsAllowsLLMMonitorTitleOverride(t *testing.T) {
	svc := NewSettingService(&llmMonitorSettingRepoStub{values: map[string]string{
		SettingKeyLLMMonitorTitle: " Override Monitor Title ",
	}}, &config.Config{
		LLMMonitor: config.LLMMonitorConfig{Title: "Configured Monitor Title"},
	})

	settings, err := svc.GetPublicSettings(context.Background())
	if err != nil {
		t.Fatalf("GetPublicSettings() error = %v", err)
	}
	if settings.LLMMonitorTitle != "Override Monitor Title" {
		t.Fatalf("LLMMonitorTitle = %q", settings.LLMMonitorTitle)
	}
}

func TestSettingServiceGetPublicSettingsAllowsLLMMonitorProviderURLOverride(t *testing.T) {
	svc := NewSettingService(&llmMonitorSettingRepoStub{values: map[string]string{
		SettingKeyLLMMonitorProviderURL: " https://override.example.com/provider ",
	}}, &config.Config{
		LLMMonitor: config.LLMMonitorConfig{ProviderURL: "https://provider.example.com/"},
	})

	settings, err := svc.GetPublicSettings(context.Background())
	if err != nil {
		t.Fatalf("GetPublicSettings() error = %v", err)
	}
	if settings.LLMMonitorProviderURL != "https://override.example.com/provider" {
		t.Fatalf("LLMMonitorProviderURL = %q", settings.LLMMonitorProviderURL)
	}
}

package service

import (
	"context"
	"fmt"
	"strings"
)

// LLMMonitorSettings contains the runtime settings required by monitor consumers.
type LLMMonitorSettings struct {
	StatusAPIURL string
	Title        string
	ProviderURL  string
}

func llmMonitorSettingKeys() []string {
	return []string{
		SettingKeyLLMMonitorStatusAPIURL,
		SettingKeyLLMMonitorTitle,
		SettingKeyLLMMonitorProviderURL,
	}
}

func addLLMMonitorDefaultSettings(defaults map[string]string) {
	for _, key := range llmMonitorSettingKeys() {
		defaults[key] = ""
	}
}

// GetLLMMonitorSettings reads monitor-specific overrides without loading all public settings.
func (s *SettingService) GetLLMMonitorSettings(ctx context.Context) (*LLMMonitorSettings, error) {
	values, err := s.settingRepo.GetMultiple(ctx, llmMonitorSettingKeys())
	if err != nil {
		return nil, fmt.Errorf("get llm monitor settings: %w", err)
	}

	settings := s.resolveLLMMonitorSettings(values)
	return &settings, nil
}

func (s *SettingService) resolveLLMMonitorSettings(values map[string]string) LLMMonitorSettings {
	settings := LLMMonitorSettings{}
	if s != nil && s.cfg != nil {
		settings.StatusAPIURL = strings.TrimSpace(s.cfg.LLMMonitor.StatusAPIURL)
		settings.Title = strings.TrimSpace(s.cfg.LLMMonitor.Title)
		settings.ProviderURL = strings.TrimSpace(s.cfg.LLMMonitor.ProviderURL)
	}

	if value := strings.TrimSpace(values[SettingKeyLLMMonitorStatusAPIURL]); value != "" {
		settings.StatusAPIURL = value
	}
	if value := strings.TrimSpace(values[SettingKeyLLMMonitorTitle]); value != "" {
		settings.Title = value
	}
	if value := strings.TrimSpace(values[SettingKeyLLMMonitorProviderURL]); value != "" {
		settings.ProviderURL = value
	}
	return settings
}

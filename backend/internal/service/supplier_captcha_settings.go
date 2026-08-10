package service

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/captcha"
)

// 供应商上游打码 settings key（独立维护，避免侵入通用 SystemSettings）。
const (
	SettingKeySupplierCaptchaProvider     = "supplier_captcha_provider"       // 打码平台
	SettingKeySupplierCaptchaAPIKey       = "supplier_captcha_api_key"        // 打码平台 API Key（敏感）
	SettingKeySupplierCaptchaEndpoint     = "supplier_captcha_endpoint"       // 可选自定义 endpoint
	SettingKeySupplierCaptchaCallTotal    = "supplier_captcha_call_total"     // 累计调用次数
	SettingKeySupplierCaptchaCallSuccess  = "supplier_captcha_call_success"   // 成功次数
	SettingKeySupplierCaptchaCallFailed   = "supplier_captcha_call_failed"    // 失败次数
	SettingKeySupplierCaptchaLastCalledAt = "supplier_captcha_last_called_at" // 最近一次调用时间（RFC3339）
)

// SupplierCaptchaSettings 供应商上游打码全局配置（对外只暴露是否已配置 API Key 与调用统计）。
type SupplierCaptchaSettings struct {
	Provider         string `json:"provider"`
	APIKeyConfigured bool   `json:"api_key_configured"`
	Endpoint         string `json:"endpoint"`
	CallTotal        int64  `json:"call_total"`
	CallSuccess      int64  `json:"call_success"`
	CallFailed       int64  `json:"call_failed"`
	LastCalledAt     string `json:"last_called_at,omitempty"`
}

// supplierCaptchaRuntimeConfig 内部运行时配置，包含 API Key。
type supplierCaptchaRuntimeConfig struct {
	Provider string
	APIKey   string
	Endpoint string
}

// UpdateSupplierCaptchaSettingsInput 更新请求。
// APIKey 留空表示保留原值；ClearAPIKey=true 时清空。
// 切换打码平台时必须提供新 API Key，避免复用旧平台凭据。
type UpdateSupplierCaptchaSettingsInput struct {
	Provider    string `json:"provider"`
	APIKey      string `json:"api_key"`
	Endpoint    string `json:"endpoint"`
	ClearAPIKey bool   `json:"clear_api_key"`
}

func normalizeSupplierCaptchaProvider(provider string) string {
	provider = strings.ToLower(strings.TrimSpace(provider))
	if provider == "" {
		return captcha.ProviderTwoCaptcha
	}
	return provider
}

func parseSupplierCaptchaCounter(raw string) int64 {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return 0
	}
	n, err := strconv.ParseInt(raw, 10, 64)
	if err != nil || n < 0 {
		return 0
	}
	return n
}

func (s *SettingService) loadSupplierCaptchaRuntimeConfig(ctx context.Context) (*supplierCaptchaRuntimeConfig, error) {
	if s == nil || s.settingRepo == nil {
		return nil, fmt.Errorf("setting service is not configured")
	}
	values, err := s.settingRepo.GetMultiple(ctx, []string{
		SettingKeySupplierCaptchaProvider,
		SettingKeySupplierCaptchaAPIKey,
		SettingKeySupplierCaptchaEndpoint,
	})
	if err != nil {
		return nil, fmt.Errorf("load supplier captcha settings: %w", err)
	}
	if values == nil {
		values = map[string]string{}
	}
	return &supplierCaptchaRuntimeConfig{
		Provider: normalizeSupplierCaptchaProvider(values[SettingKeySupplierCaptchaProvider]),
		APIKey:   strings.TrimSpace(values[SettingKeySupplierCaptchaAPIKey]),
		Endpoint: strings.TrimSpace(values[SettingKeySupplierCaptchaEndpoint]),
	}, nil
}

func (s *SettingService) loadSupplierCaptchaCallStats(ctx context.Context) (total, success, failed int64, lastCalledAt string, err error) {
	if s == nil || s.settingRepo == nil {
		return 0, 0, 0, "", fmt.Errorf("setting service is not configured")
	}
	values, err := s.settingRepo.GetMultiple(ctx, []string{
		SettingKeySupplierCaptchaCallTotal,
		SettingKeySupplierCaptchaCallSuccess,
		SettingKeySupplierCaptchaCallFailed,
		SettingKeySupplierCaptchaLastCalledAt,
	})
	if err != nil {
		return 0, 0, 0, "", fmt.Errorf("load supplier captcha call stats: %w", err)
	}
	if values == nil {
		values = map[string]string{}
	}
	return parseSupplierCaptchaCounter(values[SettingKeySupplierCaptchaCallTotal]),
		parseSupplierCaptchaCounter(values[SettingKeySupplierCaptchaCallSuccess]),
		parseSupplierCaptchaCounter(values[SettingKeySupplierCaptchaCallFailed]),
		strings.TrimSpace(values[SettingKeySupplierCaptchaLastCalledAt]),
		nil
}

// RecordSupplierCaptchaCall 记录一次实际上游打码平台调用结果。
// 仅统计真实发起 SolveTurnstile 的次数；配置校验或 site key 获取失败不计入。
// 统计写入失败不影响主流程，由调用方决定是否忽略错误。
func (s *SettingService) RecordSupplierCaptchaCall(ctx context.Context, success bool) error {
	if s == nil || s.settingRepo == nil {
		return fmt.Errorf("setting service is not configured")
	}
	total, okCount, failCount, _, err := s.loadSupplierCaptchaCallStats(ctx)
	if err != nil {
		return err
	}
	total++
	updates := map[string]string{
		SettingKeySupplierCaptchaCallTotal:    strconv.FormatInt(total, 10),
		SettingKeySupplierCaptchaLastCalledAt: time.Now().UTC().Format(time.RFC3339),
	}
	if success {
		okCount++
		updates[SettingKeySupplierCaptchaCallSuccess] = strconv.FormatInt(okCount, 10)
	} else {
		failCount++
		updates[SettingKeySupplierCaptchaCallFailed] = strconv.FormatInt(failCount, 10)
	}
	if err := s.settingRepo.SetMultiple(ctx, updates); err != nil {
		return fmt.Errorf("record supplier captcha call: %w", err)
	}
	return nil
}

// GetSupplierCaptchaSettings 读取供应商上游打码全局配置。
func (s *SettingService) GetSupplierCaptchaSettings(ctx context.Context) (*SupplierCaptchaSettings, error) {
	cfg, err := s.loadSupplierCaptchaRuntimeConfig(ctx)
	if err != nil {
		return nil, err
	}
	total, success, failed, lastCalledAt, err := s.loadSupplierCaptchaCallStats(ctx)
	if err != nil {
		return nil, err
	}
	return &SupplierCaptchaSettings{
		Provider:         cfg.Provider,
		APIKeyConfigured: cfg.APIKey != "",
		Endpoint:         cfg.Endpoint,
		CallTotal:        total,
		CallSuccess:      success,
		CallFailed:       failed,
		LastCalledAt:     lastCalledAt,
	}, nil
}

// UpdateSupplierCaptchaSettings 更新供应商上游打码全局配置。
func (s *SettingService) UpdateSupplierCaptchaSettings(ctx context.Context, input *UpdateSupplierCaptchaSettingsInput) (*SupplierCaptchaSettings, error) {
	if input == nil {
		return nil, fmt.Errorf("settings cannot be nil")
	}
	if s == nil || s.settingRepo == nil {
		return nil, fmt.Errorf("setting service is not configured")
	}

	provider := normalizeSupplierCaptchaProvider(input.Provider)
	if provider != captcha.ProviderTwoCaptcha && provider != captcha.ProviderYesCaptcha {
		return nil, fmt.Errorf("unsupported captcha provider: %s", provider)
	}

	endpoint := strings.TrimSpace(input.Endpoint)
	apiKey := strings.TrimSpace(input.APIKey)
	current, err := s.loadSupplierCaptchaRuntimeConfig(ctx)
	if err != nil {
		return nil, err
	}
	providerChanged := provider != current.Provider
	if providerChanged && apiKey == "" && !input.ClearAPIKey {
		return nil, fmt.Errorf("switching captcha provider requires a new api key")
	}
	if providerChanged && endpoint == current.Endpoint {
		endpoint = ""
	}

	updates := map[string]string{
		SettingKeySupplierCaptchaProvider: provider,
		SettingKeySupplierCaptchaEndpoint: endpoint,
	}
	if input.ClearAPIKey {
		updates[SettingKeySupplierCaptchaAPIKey] = ""
	} else if apiKey != "" {
		updates[SettingKeySupplierCaptchaAPIKey] = apiKey
	}

	if err := s.settingRepo.SetMultiple(ctx, updates); err != nil {
		return nil, fmt.Errorf("update supplier captcha settings: %w", err)
	}
	if s.onUpdate != nil {
		s.onUpdate()
	}
	return s.GetSupplierCaptchaSettings(ctx)
}

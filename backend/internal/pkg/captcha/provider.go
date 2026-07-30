package captcha

import (
	"context"
	"errors"
	"fmt"
	"strings"
)

// Provider 打码平台抽象，返回可直接用于上游登录的 Turnstile token。
type Provider interface {
	SolveTurnstile(ctx context.Context, siteKey, pageURL string) (string, error)
}

// Config 构造 Provider 所需配置。
type Config struct {
	Provider string
	APIKey   string
	Endpoint string
}

const (
	ProviderTwoCaptcha       = "2captcha"
	defaultTwoCaptchaEndpoint = "https://api.2captcha.com"
)

// New 根据配置构造打码 Provider。当前仅支持 2Captcha。
func New(cfg Config) (Provider, error) {
	providerType := strings.ToLower(strings.TrimSpace(cfg.Provider))
	if providerType == "" {
		providerType = ProviderTwoCaptcha
	}
	switch providerType {
	case ProviderTwoCaptcha:
		return newTwoCaptcha(cfg), nil
	default:
		return nil, fmt.Errorf("unsupported captcha provider: %s", cfg.Provider)
	}
}

// ValidateConfig 校验打码配置是否可直接使用。
func ValidateConfig(cfg Config) error {
	if strings.TrimSpace(cfg.APIKey) == "" {
		return errors.New("captcha api key is empty")
	}
	providerType := strings.ToLower(strings.TrimSpace(cfg.Provider))
	if providerType == "" {
		providerType = ProviderTwoCaptcha
	}
	if providerType != ProviderTwoCaptcha {
		return fmt.Errorf("unsupported captcha provider: %s", cfg.Provider)
	}
	return nil
}

package service

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/Wei-Shaw/sub2api/internal/pkg/captcha"
)

// SupplierTurnstileSolver 负责在供应商上游登录前求解 Turnstile token。
// 开关关闭时返回空 token；开启后失败直接返回错误（不降级、不空 token 重试）。
type SupplierTurnstileSolver interface {
	PrepareToken(ctx context.Context, provider *SupplierProvider, pageURL string, fetchSiteKey func(context.Context) (string, error)) (string, error)
}

type settingBackedSupplierTurnstileSolver struct {
	settings   *SettingService
	httpClient *http.Client
}

// NewSettingBackedSupplierTurnstileSolver 使用 settings 表中的全局打码配置构造求解器。
func NewSettingBackedSupplierTurnstileSolver(settings *SettingService, httpClient *http.Client) SupplierTurnstileSolver {
	if httpClient == nil {
		httpClient = &http.Client{Timeout: defaultSupplierSub2APIHTTPTimeout}
	}
	return &settingBackedSupplierTurnstileSolver{settings: settings, httpClient: httpClient}
}

func (s *settingBackedSupplierTurnstileSolver) PrepareToken(
	ctx context.Context,
	provider *SupplierProvider,
	pageURL string,
	fetchSiteKey func(context.Context) (string, error),
) (string, error) {
	if provider == nil || !provider.TurnstileEnabled {
		return "", nil
	}
	SupplierSyncProgress(ctx, SupplierSyncProgressStageCaptcha, "正在准备 Turnstile 打码", nil)
	fail := func(err error) (string, error) {
		SupplierSyncProgressFail(ctx, SupplierSyncProgressStageCaptcha, err)
		return "", err
	}
	if s == nil || s.settings == nil {
		return fail(fmt.Errorf("supplier turnstile solver is not configured"))
	}
	if fetchSiteKey == nil {
		return fail(fmt.Errorf("supplier turnstile site key fetcher is nil"))
	}

	cfg, err := s.loadCaptchaConfig(ctx)
	if err != nil {
		return fail(err)
	}
	if err := captcha.ValidateConfig(cfg); err != nil {
		return fail(fmt.Errorf("supplier captcha config invalid: %w", err))
	}

	SupplierSyncProgress(ctx, SupplierSyncProgressStageCaptcha, "正在获取上游 Turnstile 配置", nil)
	siteKey, err := fetchSiteKey(ctx)
	if err != nil {
		return fail(fmt.Errorf("fetch supplier turnstile site key: %w", err))
	}
	siteKey = strings.TrimSpace(siteKey)
	if siteKey == "" {
		return fail(fmt.Errorf("supplier turnstile is enabled but upstream site key is empty"))
	}
	SupplierSyncProgressOK(ctx, SupplierSyncProgressStageCaptcha, "已获取上游 Turnstile 配置")

	pageURL = strings.TrimSpace(pageURL)
	if pageURL == "" {
		pageURL = strings.TrimRight(strings.TrimSpace(provider.BaseURL), "/")
	}
	if pageURL == "" {
		return fail(fmt.Errorf("supplier turnstile page url is empty"))
	}

	providerImpl, err := captcha.New(cfg)
	if err != nil {
		return fail(err)
	}
	providerLabel := strings.ToLower(strings.TrimSpace(cfg.Provider))
	if providerLabel == captcha.ProviderYesCaptcha {
		SupplierSyncProgress(ctx, SupplierSyncProgressStageCaptcha, "正在调用 YesCaptcha", nil)
	} else {
		SupplierSyncProgress(ctx, SupplierSyncProgressStageCaptcha, "正在调用打码平台", nil)
	}
	token, err := providerImpl.SolveTurnstile(ctx, siteKey, pageURL)
	if err != nil {
		return fail(fmt.Errorf("solve supplier turnstile: %w", err))
	}
	token = strings.TrimSpace(token)
	if token == "" {
		return fail(fmt.Errorf("solve supplier turnstile: empty token"))
	}
	SupplierSyncProgressOK(ctx, SupplierSyncProgressStageCaptcha, "打码成功")
	return token, nil
}

func (s *settingBackedSupplierTurnstileSolver) loadCaptchaConfig(ctx context.Context) (captcha.Config, error) {
	cfg, err := s.settings.loadSupplierCaptchaRuntimeConfig(ctx)
	if err != nil {
		return captcha.Config{}, err
	}
	return captcha.Config{
		Provider: cfg.Provider,
		APIKey:   cfg.APIKey,
		Endpoint: cfg.Endpoint,
	}, nil
}

// noopSupplierTurnstileSolver 用于未注入 settings 的测试场景。
type noopSupplierTurnstileSolver struct{}

func (noopSupplierTurnstileSolver) PrepareToken(context.Context, *SupplierProvider, string, func(context.Context) (string, error)) (string, error) {
	return "", nil
}

func supplierPublicGET(ctx context.Context, httpClient *http.Client, baseURL, path string) ([]byte, int, error) {
	if httpClient == nil {
		httpClient = &http.Client{Timeout: defaultSupplierSub2APIHTTPTimeout}
	}
	target, err := supplierSub2APIURL(baseURL, path)
	if err != nil {
		return nil, 0, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, target.String(), nil)
	if err != nil {
		return nil, 0, err
	}
	req.Header.Set("Accept", "application/json")
	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, 0, err
	}
	defer resp.Body.Close()
	raw, err := io.ReadAll(io.LimitReader(resp.Body, supplierSub2APIMaxResponseBytes))
	if err != nil {
		return nil, resp.StatusCode, err
	}
	return raw, resp.StatusCode, nil
}

func fetchSupplierSub2APITurnstileSiteKey(ctx context.Context, httpClient *http.Client, provider *SupplierProvider) (string, error) {
	raw, status, err := supplierPublicGET(ctx, httpClient, provider.BaseURL, "/api/v1/settings/public")
	if err != nil {
		return "", err
	}
	if status < 200 || status >= 300 {
		return "", fmt.Errorf("supplier sub2api public settings status %d: %s", status, strings.TrimSpace(string(raw)))
	}
	var envelope struct {
		Data json.RawMessage `json:"data"`
	}
	// 兼容 {data:{...}} 与直接对象
	if err := json.Unmarshal(raw, &envelope); err == nil && len(envelope.Data) > 0 && string(envelope.Data) != "null" {
		raw = envelope.Data
	}
	var settings struct {
		TurnstileEnabled bool   `json:"turnstile_enabled"`
		TurnstileSiteKey string `json:"turnstile_site_key"`
	}
	if err := json.Unmarshal(raw, &settings); err != nil {
		return "", fmt.Errorf("decode supplier sub2api public settings: %w", err)
	}
	if !settings.TurnstileEnabled {
		return "", nil
	}
	return strings.TrimSpace(settings.TurnstileSiteKey), nil
}

func fetchSupplierNewAPITurnstileSiteKey(ctx context.Context, httpClient *http.Client, provider *SupplierProvider) (string, error) {
	raw, status, err := supplierPublicGET(ctx, httpClient, provider.BaseURL, "/api/status")
	if err != nil {
		return "", err
	}
	if status < 200 || status >= 300 {
		return "", fmt.Errorf("supplier newapi status %d: %s", status, strings.TrimSpace(string(raw)))
	}
	var envelope struct {
		Data json.RawMessage `json:"data"`
	}
	if err := json.Unmarshal(raw, &envelope); err == nil && len(envelope.Data) > 0 && string(envelope.Data) != "null" {
		raw = envelope.Data
	}
	var statusBody struct {
		TurnstileCheck   bool   `json:"turnstile_check"`
		TurnstileSiteKey string `json:"turnstile_site_key"`
	}
	if err := json.Unmarshal(raw, &statusBody); err != nil {
		return "", fmt.Errorf("decode supplier newapi status: %w", err)
	}
	if !statusBody.TurnstileCheck {
		return "", nil
	}
	return strings.TrimSpace(statusBody.TurnstileSiteKey), nil
}

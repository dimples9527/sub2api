package admin

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	"github.com/Wei-Shaw/sub2api/internal/service"

	"github.com/gin-gonic/gin"
)

const (
	defaultFindCGBaseURL = "https://www.findcg.com"
	findCGLoginPath      = "/api/v1/auth/login"
	findCGModelPath      = "/api/v1/model-square"
)

// ModelSquareHandler proxies the external model square API for admin users.
type ModelSquareHandler struct {
	baseURL        string
	httpClient     *http.Client
	email          string
	password       string
	settingService *service.SettingService

	mu    sync.Mutex
	token cachedFindCGToken
}

type cachedFindCGToken struct {
	AccessToken string
	TokenType   string
	ExpiresAt   time.Time
}

// ModelSquareHandlerConfig configures ModelSquareHandler, mainly for tests.
type ModelSquareHandlerConfig struct {
	BaseURL        string
	Email          string
	Password       string
	AppConfig      *config.Config
	SettingService *service.SettingService
	HTTPClient     *http.Client
}

type findCGLoginResponse struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    struct {
		AccessToken string `json:"access_token"`
		ExpiresIn   int64  `json:"expires_in"`
		TokenType   string `json:"token_type"`
	} `json:"data"`
}

// NewModelSquareHandler creates a model square proxy handler.
func NewModelSquareHandler(cfg *config.Config, settingService *service.SettingService) *ModelSquareHandler {
	return newModelSquareHandler(ModelSquareHandlerConfig{AppConfig: cfg, SettingService: settingService})
}

func newModelSquareHandler(cfg ModelSquareHandlerConfig) *ModelSquareHandler {
	baseURL := strings.TrimRight(cfg.BaseURL, "/")
	email := strings.TrimSpace(cfg.Email)
	password := cfg.Password

	if cfg.AppConfig != nil {
		if baseURL == "" {
			baseURL = cfg.AppConfig.ModelSquare.BaseURL
		}
		if email == "" {
			email = cfg.AppConfig.ModelSquare.Email
		}
		if password == "" {
			password = cfg.AppConfig.ModelSquare.Password
		}
	}

	if baseURL == "" {
		baseURL = strings.TrimRight(os.Getenv("MODEL_SQUARE_BASE_URL"), "/")
	}
	if baseURL == "" {
		baseURL = strings.TrimRight(os.Getenv("FINDCG_BASE_URL"), "/")
	}
	if baseURL == "" {
		baseURL = defaultFindCGBaseURL
	}
	if email == "" {
		email = strings.TrimSpace(os.Getenv("MODEL_SQUARE_EMAIL"))
	}
	if email == "" {
		email = strings.TrimSpace(os.Getenv("FINDCG_EMAIL"))
	}
	if password == "" {
		password = os.Getenv("MODEL_SQUARE_PASSWORD")
	}
	if password == "" {
		password = os.Getenv("FINDCG_PASSWORD")
	}

	client := cfg.HTTPClient
	if client == nil {
		client = &http.Client{Timeout: 15 * time.Second}
	}

	return &ModelSquareHandler{
		baseURL:        baseURL,
		httpClient:     client,
		email:          email,
		password:       password,
		settingService: cfg.SettingService,
	}
}

// Get handles GET /api/v1/admin/model-square.
func (h *ModelSquareHandler) Get(c *gin.Context) {
	payload, err := h.fetchModelSquare(c.Request.Context(), false)
	if err != nil {
		response.InternalError(c, err.Error())
		return
	}

	c.Data(http.StatusOK, "application/json; charset=utf-8", payload)
}

func (h *ModelSquareHandler) fetchModelSquare(ctx context.Context, forceLogin bool) ([]byte, error) {
	token, err := h.getToken(ctx, forceLogin)
	if err != nil {
		return nil, err
	}

	payload, status, err := h.requestModelSquare(ctx, token)
	if err != nil {
		return nil, err
	}
	if status != http.StatusUnauthorized {
		if status < 200 || status >= 300 {
			return nil, fmt.Errorf("findcg model-square failed: HTTP %d: %s", status, string(payload))
		}
		return payload, nil
	}

	token, err = h.getToken(ctx, true)
	if err != nil {
		return nil, err
	}
	payload, status, err = h.requestModelSquare(ctx, token)
	if err != nil {
		return nil, err
	}
	if status < 200 || status >= 300 {
		return nil, fmt.Errorf("findcg model-square failed after relogin: HTTP %d: %s", status, string(payload))
	}

	return payload, nil
}

func (h *ModelSquareHandler) getToken(ctx context.Context, force bool) (cachedFindCGToken, error) {
	h.mu.Lock()
	defer h.mu.Unlock()

	baseURL, email, password := h.resolveConfigLocked(ctx)
	if baseURL != h.baseURL || email != h.email || password != h.password {
		h.baseURL = baseURL
		h.email = email
		h.password = password
		h.token = cachedFindCGToken{}
	}

	if !force && h.token.AccessToken != "" && time.Now().Before(h.token.ExpiresAt.Add(-time.Minute)) {
		return h.token, nil
	}

	if email == "" || password == "" {
		return cachedFindCGToken{}, fmt.Errorf("missing findcg credentials: set model_square.email/model_square.password or MODEL_SQUARE_EMAIL/MODEL_SQUARE_PASSWORD")
	}

	body, err := json.Marshal(map[string]string{
		"email":    email,
		"password": password,
	})
	if err != nil {
		return cachedFindCGToken{}, fmt.Errorf("marshal login payload: %w", err)
	}

	req, err := http.NewRequest(http.MethodPost, h.baseURL+findCGLoginPath, bytes.NewReader(body))
	if err != nil {
		return cachedFindCGToken{}, fmt.Errorf("create findcg login request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := h.httpClient.Do(req)
	if err != nil {
		return cachedFindCGToken{}, fmt.Errorf("findcg login request failed: %w", err)
	}
	defer resp.Body.Close()

	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return cachedFindCGToken{}, fmt.Errorf("read findcg login response: %w", err)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return cachedFindCGToken{}, fmt.Errorf("findcg login failed: HTTP %d: %s", resp.StatusCode, string(raw))
	}

	var loginResp findCGLoginResponse
	if err := json.Unmarshal(raw, &loginResp); err != nil {
		return cachedFindCGToken{}, fmt.Errorf("decode findcg login response: %w", err)
	}
	if loginResp.Code != 0 || loginResp.Data.AccessToken == "" {
		if loginResp.Message == "" {
			loginResp.Message = "missing access_token"
		}
		return cachedFindCGToken{}, fmt.Errorf("findcg login failed: %s", loginResp.Message)
	}

	expiresIn := loginResp.Data.ExpiresIn
	if expiresIn <= 0 {
		expiresIn = 3600
	}
	tokenType := loginResp.Data.TokenType
	if tokenType == "" {
		tokenType = "Bearer"
	}

	h.token = cachedFindCGToken{
		AccessToken: loginResp.Data.AccessToken,
		TokenType:   tokenType,
		ExpiresAt:   time.Now().Add(time.Duration(expiresIn) * time.Second),
	}

	return h.token, nil
}

func (h *ModelSquareHandler) resolveConfigLocked(ctx context.Context) (string, string, string) {
	baseURL := h.baseURL
	email := h.email
	password := h.password

	if h.settingService != nil {
		if settings, err := h.settingService.GetAllSettings(ctx); err == nil {
			if settings.ModelSquareBaseURL != "" {
				baseURL = settings.ModelSquareBaseURL
			}
			if settings.ModelSquareEmail != "" {
				email = settings.ModelSquareEmail
			}
			if settings.ModelSquarePassword != "" {
				password = settings.ModelSquarePassword
			}
		}
	}

	if baseURL == "" {
		baseURL = defaultFindCGBaseURL
	}
	return strings.TrimRight(baseURL, "/"), strings.TrimSpace(email), password
}

func (h *ModelSquareHandler) requestModelSquare(ctx context.Context, token cachedFindCGToken) ([]byte, int, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, h.baseURL+findCGModelPath, nil)
	if err != nil {
		return nil, 0, fmt.Errorf("create findcg model-square request: %w", err)
	}
	if token.TokenType == "" {
		token.TokenType = "Bearer"
	}
	req.Header.Set("Authorization", token.TokenType+" "+token.AccessToken)

	resp, err := h.httpClient.Do(req)
	if err != nil {
		return nil, 0, fmt.Errorf("findcg model-square request failed: %w", err)
	}
	defer resp.Body.Close()

	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, 0, fmt.Errorf("read findcg model-square response: %w", err)
	}

	return raw, resp.StatusCode, nil
}

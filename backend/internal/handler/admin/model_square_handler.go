package admin

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/Wei-Shaw/sub2api/internal/handler/dto"
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
	groupProvider  modelSquareGroupProvider

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
	GroupProvider  modelSquareGroupProvider
	HTTPClient     *http.Client
}

type modelSquareGroupProvider interface {
	GetAllGroups(ctx context.Context) ([]service.Group, error)
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
func NewModelSquareHandler(cfg *config.Config, settingService *service.SettingService, groupProvider service.AdminService) *ModelSquareHandler {
	return newModelSquareHandler(ModelSquareHandlerConfig{AppConfig: cfg, SettingService: settingService, GroupProvider: groupProvider})
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
		groupProvider:  cfg.GroupProvider,
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
		return h.mergeGroups(ctx, payload)
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

	return h.mergeGroups(ctx, payload)
}

func (h *ModelSquareHandler) getToken(ctx context.Context, force bool) (cachedFindCGToken, error) {
	h.mu.Lock()
	defer h.mu.Unlock()

	baseURL, email, password, source := h.resolveConfigLocked(ctx)
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
	logFindCGLoginAttempt(h.baseURL, email, password, source)

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
		logFindCGLoginFailure(resp.StatusCode, raw, h.baseURL, email, password, source)
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
		logFindCGLoginFailure(resp.StatusCode, raw, h.baseURL, email, password, source)
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

func (h *ModelSquareHandler) resolveConfigLocked(ctx context.Context) (string, string, string, string) {
	baseURL := h.baseURL
	email := h.email
	password := h.password
	source := "config_or_env"

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
			if settings.ModelSquareBaseURL != "" || settings.ModelSquareEmail != "" || settings.ModelSquarePassword != "" {
				source = "settings"
			}
		} else {
			slog.Warn("findcg model square settings lookup failed, using handler config", "error", err)
		}
	}

	if baseURL == "" {
		baseURL = defaultFindCGBaseURL
	}
	return strings.TrimRight(baseURL, "/"), strings.TrimSpace(email), password, source
}

func logFindCGLoginAttempt(baseURL, email, password, source string) {
	slog.Info(
		"findcg login attempt",
		"base_url", strings.TrimRight(baseURL, "/"),
		"email", maskFindCGEmail(email),
		"password_configured", password != "",
		"password_length", len(password),
		"config_source", source,
	)
}

func logFindCGLoginFailure(status int, body []byte, baseURL, email, password, source string) {
	slog.Warn(
		"findcg login failed",
		"status", status,
		"body", string(body),
		"base_url", strings.TrimRight(baseURL, "/"),
		"email", maskFindCGEmail(email),
		"password_configured", password != "",
		"password_length", len(password),
		"config_source", source,
	)
}

func maskFindCGEmail(email string) string {
	email = strings.TrimSpace(email)
	if email == "" {
		return "***"
	}
	parts := strings.SplitN(email, "@", 2)
	if len(parts) != 2 {
		if len(email) <= 2 {
			return "***"
		}
		return email[:1] + "***" + email[len(email)-1:]
	}
	local := parts[0]
	if len(local) <= 1 {
		return local + "***@" + parts[1]
	}
	if len(local) <= 4 {
		return local[:2] + "***" + local[len(local)-2:] + "@" + parts[1]
	}
	return local[:2] + "***" + local[len(local)-2:] + "@" + parts[1]
}

func (h *ModelSquareHandler) mergeGroups(ctx context.Context, payload []byte) ([]byte, error) {
	if h.groupProvider == nil {
		return payload, nil
	}

	var body map[string]any
	if err := json.Unmarshal(payload, &body); err != nil {
		return nil, fmt.Errorf("decode findcg model-square response: %w", err)
	}

	rawGroups, ok := body["groups"].([]any)
	if !ok || len(rawGroups) == 0 {
		return payload, nil
	}

	localGroups, err := h.groupProvider.GetAllGroups(ctx)
	if err != nil {
		return nil, fmt.Errorf("load local groups for model square: %w", err)
	}
	localByName := make(map[string]service.Group, len(localGroups))
	for i := range localGroups {
		key := normalizeModelSquareGroupName(localGroups[i].Name)
		if key != "" {
			localByName[key] = localGroups[i]
		}
	}

	keptGroupIDs := make(map[string]struct{}, len(rawGroups))
	mergedGroups := make([]any, 0, len(rawGroups))
	for _, rawGroup := range rawGroups {
		groupMap, ok := rawGroup.(map[string]any)
		if !ok {
			continue
		}

		remoteID, hasID := groupMap["id"]
		remoteName, _ := groupMap["name"].(string)
		localGroup, matched := localByName[normalizeModelSquareGroupName(remoteName)]
		if !matched || !hasID {
			continue
		}

		mergedGroup, err := modelSquareGroupMapFromLocal(localGroup)
		if err != nil {
			return nil, err
		}
		mergedGroup["id"] = remoteID
		keptGroupIDs[modelSquareIDKey(remoteID)] = struct{}{}
		mergedGroups = append(mergedGroups, mergedGroup)
	}

	body["groups"] = mergedGroups
	filterModelSquareGroupIDs(body, keptGroupIDs)
	mergedPayload, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("encode merged model-square response: %w", err)
	}
	return mergedPayload, nil
}

func filterModelSquareGroupIDs(body map[string]any, keptGroupIDs map[string]struct{}) {
	rawModels, ok := body["models"].([]any)
	if !ok {
		return
	}
	for _, rawModel := range rawModels {
		modelMap, ok := rawModel.(map[string]any)
		if !ok {
			continue
		}
		rawGroupIDs, ok := modelMap["group_ids"].([]any)
		if !ok {
			continue
		}
		filtered := make([]any, 0, len(rawGroupIDs))
		for _, groupID := range rawGroupIDs {
			if _, ok := keptGroupIDs[modelSquareIDKey(groupID)]; ok {
				filtered = append(filtered, groupID)
			}
		}
		modelMap["group_ids"] = filtered
	}
}

func modelSquareGroupMapFromLocal(group service.Group) (map[string]any, error) {
	raw, err := json.Marshal(dto.GroupFromServiceAdmin(&group))
	if err != nil {
		return nil, fmt.Errorf("encode local group for model square: %w", err)
	}
	var out map[string]any
	if err := json.Unmarshal(raw, &out); err != nil {
		return nil, fmt.Errorf("decode local group for model square: %w", err)
	}
	return out, nil
}

func normalizeModelSquareGroupName(name string) string {
	return strings.ToLower(strings.Join(strings.Fields(name), ""))
}

func modelSquareIDKey(value any) string {
	return fmt.Sprint(value)
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

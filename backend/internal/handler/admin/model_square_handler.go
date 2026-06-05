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
	findCGKeysPath       = "/api/v1/keys?page=1&page_size=100&timezone=Asia%2FShanghai"
	findCGGroupsPath     = "/api/v1/groups/available?timezone=Asia%2FShanghai"

	modelSquareKeysSyncErrorBackoff = 30 * time.Second
	modelSquareRateCompareEpsilon   = 1e-9
)

// ModelSquareHandler proxies the external model square API for admin users.
type ModelSquareHandler struct {
	baseURL        string
	loginURL       string
	modelURL       string
	keysURL        string
	groupsURL      string
	httpClient     *http.Client
	email          string
	password       string
	settingService *service.SettingService
	groupProvider  modelSquareGroupProvider
	syncInterval   time.Duration

	mu    sync.Mutex
	token cachedFindCGToken

	syncRunMu sync.Mutex

	backgroundMu      sync.Mutex
	backgroundStop    chan struct{}
	backgroundStopped chan struct{}
}

type cachedFindCGToken struct {
	AccessToken string
	TokenType   string
	ExpiresAt   time.Time
}

// ModelSquareHandlerConfig configures ModelSquareHandler, mainly for tests.
type ModelSquareHandlerConfig struct {
	BaseURL        string
	LoginURL       string
	ModelURL       string
	KeysURL        string
	GroupsURL      string
	Email          string
	Password       string
	AppConfig      *config.Config
	SettingService *service.SettingService
	GroupProvider  modelSquareGroupProvider
	HTTPClient     *http.Client
	SyncInterval   time.Duration
}

type modelSquareGroupProvider interface {
	GetAllGroups(ctx context.Context) ([]service.Group, error)
	UpdateGroup(ctx context.Context, id int64, input *service.UpdateGroupInput) (*service.Group, error)
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

type findCGKeysResponse struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    struct {
		Items []findCGKeyItem `json:"items"`
	} `json:"data"`
}

type findCGKeyItem struct {
	Name  string          `json:"name"`
	Group *findCGKeyGroup `json:"group"`
}

type findCGKeyGroup struct {
	Name           string   `json:"name"`
	RateMultiplier *float64 `json:"rate_multiplier"`
}

type modelSquareKeysSyncResult struct {
	CheckedCount int                           `json:"checked_count"`
	MatchedCount int                           `json:"matched_count"`
	UpdatedCount int                           `json:"updated_count"`
	RateWarnings []modelSquareRateWarningEntry `json:"rate_warnings,omitempty"`
}

type modelSquareRateWarningEntry struct {
	GroupID                int64   `json:"group_id"`
	GroupName              string  `json:"group_name"`
	LocalRateMultiplier    float64 `json:"local_rate_multiplier"`
	UpstreamRateMultiplier float64 `json:"upstream_rate_multiplier"`
}

// NewModelSquareHandler creates a model square proxy handler.
func NewModelSquareHandler(cfg *config.Config, settingService *service.SettingService, groupProvider service.AdminService) *ModelSquareHandler {
	return newModelSquareHandler(ModelSquareHandlerConfig{AppConfig: cfg, SettingService: settingService, GroupProvider: groupProvider})
}

func newModelSquareHandler(cfg ModelSquareHandlerConfig) *ModelSquareHandler {
	baseURL := strings.TrimRight(cfg.BaseURL, "/")
	loginURL := strings.TrimSpace(cfg.LoginURL)
	modelURL := strings.TrimSpace(cfg.ModelURL)
	keysURL := strings.TrimSpace(cfg.KeysURL)
	groupsURL := strings.TrimSpace(cfg.GroupsURL)
	email := strings.TrimSpace(cfg.Email)
	password := cfg.Password
	syncInterval := cfg.SyncInterval

	if cfg.AppConfig != nil {
		upstreamCfg := cfg.AppConfig.UpstreamManagement
		legacyCfg := cfg.AppConfig.ModelSquare
		if baseURL == "" {
			baseURL = firstNonEmpty(upstreamCfg.BaseURL, legacyCfg.BaseURL)
		}
		if email == "" {
			email = firstNonEmpty(upstreamCfg.Email, legacyCfg.Email)
		}
		if password == "" {
			password = firstNonEmpty(upstreamCfg.Password, legacyCfg.Password)
		}
		if loginURL == "" {
			loginURL = firstNonEmpty(upstreamCfg.LoginURL, legacyCfg.LoginURL)
		}
		if modelURL == "" {
			modelURL = firstNonEmpty(upstreamCfg.ModelURL, upstreamCfg.ModelSquareURL, legacyCfg.ModelURL, legacyCfg.ModelSquareURL)
		}
		if keysURL == "" {
			keysURL = firstNonEmpty(upstreamCfg.APIKeysURL, upstreamCfg.KeysURL, legacyCfg.APIKeysURL, legacyCfg.KeysURL)
		}
		if groupsURL == "" {
			groupsURL = firstNonEmpty(upstreamCfg.GroupsURL, legacyCfg.GroupsURL)
		}
		if syncInterval == 0 {
			if interval := firstPositiveInt(upstreamCfg.KeysSyncIntervalSeconds, legacyCfg.KeysSyncIntervalSeconds); interval > 0 {
				syncInterval = time.Duration(interval) * time.Second
			}
		}
	}

	if baseURL == "" {
		baseURL = strings.TrimRight(os.Getenv("UPSTREAM_MANAGEMENT_BASE_URL"), "/")
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
		email = strings.TrimSpace(os.Getenv("UPSTREAM_MANAGEMENT_EMAIL"))
	}
	if email == "" {
		email = strings.TrimSpace(os.Getenv("MODEL_SQUARE_EMAIL"))
	}
	if email == "" {
		email = strings.TrimSpace(os.Getenv("FINDCG_EMAIL"))
	}
	if password == "" {
		password = os.Getenv("UPSTREAM_MANAGEMENT_PASSWORD")
	}
	if password == "" {
		password = os.Getenv("MODEL_SQUARE_PASSWORD")
	}
	if password == "" {
		password = os.Getenv("FINDCG_PASSWORD")
	}
	if loginURL == "" {
		loginURL = strings.TrimSpace(os.Getenv("UPSTREAM_MANAGEMENT_LOGIN_URL"))
	}
	if loginURL == "" {
		loginURL = strings.TrimSpace(os.Getenv("MODEL_SQUARE_LOGIN_URL"))
	}
	if modelURL == "" {
		modelURL = strings.TrimSpace(os.Getenv("UPSTREAM_MANAGEMENT_MODEL_URL"))
	}
	if modelURL == "" {
		modelURL = strings.TrimSpace(os.Getenv("UPSTREAM_MANAGEMENT_MODEL_SQUARE_URL"))
	}
	if modelURL == "" {
		modelURL = strings.TrimSpace(os.Getenv("MODEL_SQUARE_MODEL_URL"))
	}
	if keysURL == "" {
		keysURL = strings.TrimSpace(os.Getenv("UPSTREAM_MANAGEMENT_API_KEYS_URL"))
	}
	if keysURL == "" {
		keysURL = strings.TrimSpace(os.Getenv("MODEL_SQUARE_KEYS_URL"))
	}
	if groupsURL == "" {
		groupsURL = strings.TrimSpace(os.Getenv("UPSTREAM_MANAGEMENT_GROUPS_URL"))
	}
	if groupsURL == "" {
		groupsURL = strings.TrimSpace(os.Getenv("MODEL_SQUARE_GROUPS_URL"))
	}

	client := cfg.HTTPClient
	if client == nil {
		client = &http.Client{Timeout: 15 * time.Second}
	}

	return &ModelSquareHandler{
		baseURL:        baseURL,
		loginURL:       loginURL,
		modelURL:       modelURL,
		keysURL:        keysURL,
		groupsURL:      groupsURL,
		httpClient:     client,
		email:          email,
		password:       password,
		settingService: cfg.SettingService,
		groupProvider:  cfg.GroupProvider,
		syncInterval:   syncInterval,
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

// GetAvailableGroups handles GET /api/v1/admin/upstream-management/groups.
func (h *ModelSquareHandler) GetAvailableGroups(c *gin.Context) {
	payload, err := h.fetchAvailableGroups(c.Request.Context(), false)
	if err != nil {
		response.InternalError(c, err.Error())
		return
	}

	c.Data(http.StatusOK, "application/json; charset=utf-8", payload)
}

// SyncKeys handles POST /api/v1/admin/upstream-management/sync.
func (h *ModelSquareHandler) SyncKeys(c *gin.Context) {
	result, err := h.syncKeysOnce(c.Request.Context(), false)
	if err != nil {
		response.InternalError(c, err.Error())
		return
	}

	response.Success(c, result)
}

// RateWarnings handles GET /api/v1/admin/upstream-management/rate-warnings.
func (h *ModelSquareHandler) RateWarnings(c *gin.Context) {
	result, err := h.collectGroupRateWarningsFromKeys(c.Request.Context(), false)
	if err != nil {
		response.InternalError(c, err.Error())
		return
	}

	response.Success(c, result)
}

func (h *ModelSquareHandler) fetchModelSquare(ctx context.Context, forceLogin bool) ([]byte, error) {
	payload, status, err := h.requestModelSquareAuthenticated(ctx, findCGModelPath, "model square upstream", forceLogin)
	if err != nil {
		return nil, err
	}
	if status < 200 || status >= 300 {
		return nil, fmt.Errorf("model square upstream failed: HTTP %d: %s", status, string(payload))
	}

	return h.mergeGroups(ctx, payload)
}

func (h *ModelSquareHandler) fetchAvailableGroups(ctx context.Context, forceLogin bool) ([]byte, error) {
	payload, status, err := h.requestModelSquareAuthenticated(ctx, findCGGroupsPath, "model square groups", forceLogin)
	if err != nil {
		return nil, err
	}
	if status < 200 || status >= 300 {
		return nil, fmt.Errorf("model square groups upstream failed: HTTP %d: %s", status, string(payload))
	}
	return h.mergeAvailableGroupsWithLocal(ctx, payload)
}

func (h *ModelSquareHandler) StartBackgroundSync() {
	if h == nil || h.groupProvider == nil || h.syncInterval <= 0 {
		return
	}

	h.backgroundMu.Lock()
	defer h.backgroundMu.Unlock()
	if h.backgroundStop != nil {
		return
	}

	stop := make(chan struct{})
	stopped := make(chan struct{})
	h.backgroundStop = stop
	h.backgroundStopped = stopped
	go h.runBackgroundSync(stop, stopped)
}

func (h *ModelSquareHandler) StopBackgroundSync() {
	if h == nil {
		return
	}

	h.backgroundMu.Lock()
	stop := h.backgroundStop
	stopped := h.backgroundStopped
	h.backgroundStop = nil
	h.backgroundStopped = nil
	h.backgroundMu.Unlock()

	if stop == nil {
		return
	}
	close(stop)
	<-stopped
}

func (h *ModelSquareHandler) runBackgroundSync(stop <-chan struct{}, stopped chan<- struct{}) {
	defer close(stopped)

	for {
		if _, err := h.syncKeysOnce(context.Background(), false); err != nil {
			slog.Warn("model square background key rate sync failed", "error", err)
			if !waitModelSquareSyncInterval(modelSquareKeysSyncErrorBackoff, stop) {
				return
			}
			continue
		}
		if !waitModelSquareSyncInterval(h.currentSyncInterval(context.Background()), stop) {
			return
		}
	}
}

func (h *ModelSquareHandler) currentSyncInterval(ctx context.Context) time.Duration {
	if h == nil {
		return 0
	}
	if h.settingService != nil {
		if settings, err := h.settingService.GetAllSettings(ctx); err == nil && settings != nil {
			if settings.UpstreamManagementKeysSyncIntervalSeconds > 0 {
				return time.Duration(settings.UpstreamManagementKeysSyncIntervalSeconds) * time.Second
			}
		}
	}
	return h.syncInterval
}

func waitModelSquareSyncInterval(interval time.Duration, stop <-chan struct{}) bool {
	if interval <= 0 {
		interval = 5 * time.Second
	}
	timer := time.NewTimer(interval)
	defer timer.Stop()

	select {
	case <-timer.C:
		return true
	case <-stop:
		return false
	}
}

func (h *ModelSquareHandler) syncKeysOnce(ctx context.Context, forceLogin bool) (modelSquareKeysSyncResult, error) {
	h.syncRunMu.Lock()
	defer h.syncRunMu.Unlock()

	return h.syncGroupRateMultipliersFromKeys(ctx, forceLogin)
}

func (h *ModelSquareHandler) syncGroupRateMultipliersFromKeys(ctx context.Context, forceLogin bool) (modelSquareKeysSyncResult, error) {
	result, maxRemoteRateByGroupID, err := h.collectGroupRateWarningsAndUpdatesFromKeys(ctx, forceLogin)
	if err != nil {
		return result, err
	}

	for groupID, remoteRate := range maxRemoteRateByGroupID {
		rate := remoteRate
		if _, err := h.groupProvider.UpdateGroup(ctx, groupID, &service.UpdateGroupInput{RateMultiplier: &rate}); err != nil {
			return result, fmt.Errorf("update local group rate multiplier for model square key sync: %w", err)
		}
		result.UpdatedCount++
	}

	return result, nil
}

func (h *ModelSquareHandler) collectGroupRateWarningsFromKeys(ctx context.Context, forceLogin bool) (modelSquareKeysSyncResult, error) {
	result, _, err := h.collectGroupRateWarningsAndUpdatesFromKeys(ctx, forceLogin)
	return result, err
}

func (h *ModelSquareHandler) collectGroupRateWarningsAndUpdatesFromKeys(ctx context.Context, forceLogin bool) (modelSquareKeysSyncResult, map[int64]float64, error) {
	result := modelSquareKeysSyncResult{}
	if h.groupProvider == nil {
		return result, nil, nil
	}

	localGroups, err := h.groupProvider.GetAllGroups(ctx)
	if err != nil {
		return result, nil, fmt.Errorf("load local groups for model square key rate sync: %w", err)
	}
	if len(localGroups) == 0 {
		return result, nil, nil
	}

	localByName := make(map[string]service.Group, len(localGroups))
	for i := range localGroups {
		key := normalizeModelSquareGroupName(localGroups[i].Name)
		if key != "" {
			localByName[key] = localGroups[i]
		}
	}

	payload, status, err := h.requestModelSquareAuthenticated(ctx, findCGKeysPath, "model square keys", forceLogin)
	if err != nil {
		return result, nil, err
	}
	if status < 200 || status >= 300 {
		return result, nil, fmt.Errorf("model square keys upstream failed: HTTP %d: %s", status, string(payload))
	}

	var keysResp findCGKeysResponse
	if err := json.Unmarshal(payload, &keysResp); err != nil {
		return result, nil, fmt.Errorf("decode model square keys response: %w", err)
	}
	if keysResp.Code != 0 {
		if keysResp.Message == "" {
			keysResp.Message = "unknown error"
		}
		return result, nil, fmt.Errorf("model square keys upstream failed: %s", keysResp.Message)
	}

	maxRemoteRateByGroupID := make(map[int64]float64)
	warningIndexByGroupID := make(map[int64]int)
	for _, item := range keysResp.Data.Items {
		if item.Group == nil || item.Group.RateMultiplier == nil || *item.Group.RateMultiplier < 0 {
			continue
		}
		result.CheckedCount++
		localGroup, ok := modelSquareLocalGroupForKeyItem(item, localByName)
		if !ok {
			continue
		}
		result.MatchedCount++
		if modelSquareRemoteRateNotLower(*item.Group.RateMultiplier, localGroup.RateMultiplier) {
			warning := modelSquareRateWarningEntry{
				GroupID:                localGroup.ID,
				GroupName:              localGroup.Name,
				LocalRateMultiplier:    localGroup.RateMultiplier,
				UpstreamRateMultiplier: *item.Group.RateMultiplier,
			}
			if index, ok := warningIndexByGroupID[localGroup.ID]; ok {
				if modelSquareRemoteRateGreater(warning.UpstreamRateMultiplier, result.RateWarnings[index].UpstreamRateMultiplier) {
					result.RateWarnings[index] = warning
				}
			} else {
				warningIndexByGroupID[localGroup.ID] = len(result.RateWarnings)
				result.RateWarnings = append(result.RateWarnings, warning)
			}
		}
		if !modelSquareRemoteRateGreater(*item.Group.RateMultiplier, localGroup.RateMultiplier) {
			continue
		}
		if current, ok := maxRemoteRateByGroupID[localGroup.ID]; !ok || modelSquareRemoteRateGreater(*item.Group.RateMultiplier, current) {
			maxRemoteRateByGroupID[localGroup.ID] = *item.Group.RateMultiplier
		}
	}

	return result, maxRemoteRateByGroupID, nil
}

func modelSquareRemoteRateGreater(remoteRate, localRate float64) bool {
	return remoteRate-localRate > modelSquareRateCompareEpsilon
}

func modelSquareRemoteRateNotLower(remoteRate, localRate float64) bool {
	return localRate-remoteRate <= modelSquareRateCompareEpsilon
}

func modelSquareLocalGroupForKeyItem(item findCGKeyItem, localByName map[string]service.Group) (service.Group, bool) {
	if item.Group != nil {
		if group, ok := localByName[normalizeModelSquareGroupName(item.Group.Name)]; ok {
			return group, true
		}
	}
	if group, ok := localByName[normalizeModelSquareGroupName(item.Name)]; ok {
		return group, true
	}
	return service.Group{}, false
}

func (h *ModelSquareHandler) getToken(ctx context.Context, force bool) (cachedFindCGToken, error) {
	h.mu.Lock()
	defer h.mu.Unlock()

	baseURL, loginURL, modelURL, keysURL, groupsURL, email, password, source := h.resolveConfigLocked(ctx)
	if baseURL != h.baseURL || loginURL != h.loginURL || modelURL != h.modelURL || keysURL != h.keysURL || groupsURL != h.groupsURL || email != h.email || password != h.password {
		h.baseURL = baseURL
		h.loginURL = loginURL
		h.modelURL = modelURL
		h.keysURL = keysURL
		h.groupsURL = groupsURL
		h.email = email
		h.password = password
		h.token = cachedFindCGToken{}
	}

	if !force && h.token.AccessToken != "" && time.Now().Before(h.token.ExpiresAt.Add(-time.Minute)) {
		return h.token, nil
	}

	if email == "" || password == "" {
		return cachedFindCGToken{}, fmt.Errorf("missing upstream management credentials: set upstream_management.email/upstream_management.password or UPSTREAM_MANAGEMENT_EMAIL/UPSTREAM_MANAGEMENT_PASSWORD")
	}
	logFindCGLoginAttempt(h.baseURL, email, password, source)

	body, err := json.Marshal(map[string]string{
		"email":    email,
		"password": password,
	})
	if err != nil {
		return cachedFindCGToken{}, fmt.Errorf("marshal login payload: %w", err)
	}

	req, err := http.NewRequest(http.MethodPost, h.upstreamURLLocked(findCGLoginPath, h.loginURL), bytes.NewReader(body))
	if err != nil {
		return cachedFindCGToken{}, fmt.Errorf("create model square login request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := h.httpClient.Do(req)
	if err != nil {
		return cachedFindCGToken{}, fmt.Errorf("model square login request failed: %w", err)
	}
	defer resp.Body.Close()

	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return cachedFindCGToken{}, fmt.Errorf("read model square login response: %w", err)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		logFindCGLoginFailure(resp.StatusCode, raw, h.baseURL, email, password, source)
		return cachedFindCGToken{}, fmt.Errorf("model square login failed: HTTP %d: %s", resp.StatusCode, string(raw))
	}

	var loginResp findCGLoginResponse
	if err := json.Unmarshal(raw, &loginResp); err != nil {
		return cachedFindCGToken{}, fmt.Errorf("decode model square login response: %w", err)
	}
	if loginResp.Code != 0 || loginResp.Data.AccessToken == "" {
		if loginResp.Message == "" {
			loginResp.Message = "missing access_token"
		}
		logFindCGLoginFailure(resp.StatusCode, raw, h.baseURL, email, password, source)
		return cachedFindCGToken{}, fmt.Errorf("model square login failed: %s", loginResp.Message)
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

func (h *ModelSquareHandler) resolveConfigLocked(ctx context.Context) (string, string, string, string, string, string, string, string) {
	baseURL := h.baseURL
	loginURL := h.loginURL
	modelURL := h.modelURL
	keysURL := h.keysURL
	groupsURL := h.groupsURL
	email := h.email
	password := h.password
	source := "config_or_env"

	if h.settingService != nil {
		if settings, err := h.settingService.GetAllSettings(ctx); err == nil {
			if settings.UpstreamManagementBaseURL != "" {
				baseURL = settings.UpstreamManagementBaseURL
			}
			if settings.UpstreamManagementEmail != "" {
				email = settings.UpstreamManagementEmail
			}
			if settings.UpstreamManagementPassword != "" {
				password = settings.UpstreamManagementPassword
			}
			if settings.UpstreamManagementLoginURL != "" {
				loginURL = settings.UpstreamManagementLoginURL
			}
			if settings.UpstreamManagementModelURL != "" {
				modelURL = settings.UpstreamManagementModelURL
			}
			if settings.UpstreamManagementAPIKeysURL != "" {
				keysURL = settings.UpstreamManagementAPIKeysURL
			}
			if settings.UpstreamManagementGroupsURL != "" {
				groupsURL = settings.UpstreamManagementGroupsURL
			}
			if settings.UpstreamManagementBaseURL != "" || settings.UpstreamManagementEmail != "" || settings.UpstreamManagementPassword != "" || settings.UpstreamManagementLoginURL != "" || settings.UpstreamManagementModelURL != "" || settings.UpstreamManagementAPIKeysURL != "" || settings.UpstreamManagementGroupsURL != "" {
				source = "settings"
			}
		} else {
			slog.Warn("model square settings lookup failed, using handler config", "error", err)
		}
	}

	if baseURL == "" {
		baseURL = defaultFindCGBaseURL
	}
	return strings.TrimRight(baseURL, "/"), strings.TrimSpace(loginURL), strings.TrimSpace(modelURL), strings.TrimSpace(keysURL), strings.TrimSpace(groupsURL), strings.TrimSpace(email), password, source
}

func logFindCGLoginAttempt(baseURL, email, password, source string) {
	slog.Info(
		"model square login attempt",
		"base_url", strings.TrimRight(baseURL, "/"),
		"email", maskFindCGEmail(email),
		"password_configured", password != "",
		"password_length", len(password),
		"config_source", source,
	)
}

func logFindCGLoginFailure(status int, body []byte, baseURL, email, password, source string) {
	slog.Warn(
		"model square login failed",
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
		return nil, fmt.Errorf("decode model square upstream response: %w", err)
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

func (h *ModelSquareHandler) mergeAvailableGroupsWithLocal(ctx context.Context, payload []byte) ([]byte, error) {
	var body map[string]any
	if err := json.Unmarshal(payload, &body); err != nil {
		return nil, fmt.Errorf("decode model square groups upstream response: %w", err)
	}

	rawGroups, ok := body["data"].([]any)
	if !ok {
		return payload, nil
	}

	localByName := map[string]service.Group{}
	if h.groupProvider != nil {
		localGroups, err := h.groupProvider.GetAllGroups(ctx)
		if err != nil {
			return nil, fmt.Errorf("load local groups for model square groups: %w", err)
		}
		localByName = make(map[string]service.Group, len(localGroups))
		for i := range localGroups {
			key := normalizeModelSquareGroupName(localGroups[i].Name)
			if key != "" {
				localByName[key] = localGroups[i]
			}
		}
	}

	for _, rawGroup := range rawGroups {
		groupMap, ok := rawGroup.(map[string]any)
		if !ok {
			continue
		}
		remoteName, _ := groupMap["name"].(string)
		if localGroup, matched := localByName[normalizeModelSquareGroupName(remoteName)]; matched {
			groupMap["local_group_id"] = localGroup.ID
			groupMap["local_group_name"] = localGroup.Name
			groupMap["local_rate_multiplier"] = localGroup.RateMultiplier
		} else {
			groupMap["local_group_id"] = nil
			groupMap["local_group_name"] = ""
			groupMap["local_rate_multiplier"] = nil
		}
	}

	mergedPayload, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("encode merged model-square groups response: %w", err)
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

func (h *ModelSquareHandler) requestModelSquareAuthenticated(ctx context.Context, path, label string, forceLogin bool) ([]byte, int, error) {
	token, err := h.getToken(ctx, forceLogin)
	if err != nil {
		return nil, 0, err
	}

	payload, status, err := h.requestModelSquareWithToken(ctx, token, path, label)
	if err != nil || status != http.StatusUnauthorized {
		return payload, status, err
	}

	token, err = h.getToken(ctx, true)
	if err != nil {
		return nil, 0, err
	}
	return h.requestModelSquareWithToken(ctx, token, path, label)
}

func (h *ModelSquareHandler) requestModelSquareWithToken(ctx context.Context, token cachedFindCGToken, path, label string) ([]byte, int, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, h.configuredUpstreamURL(path), nil)
	if err != nil {
		return nil, 0, fmt.Errorf("create %s request: %w", label, err)
	}
	if token.TokenType == "" {
		token.TokenType = "Bearer"
	}
	req.Header.Set("Authorization", token.TokenType+" "+token.AccessToken)

	resp, err := h.httpClient.Do(req)
	if err != nil {
		return nil, 0, fmt.Errorf("%s request failed: %w", label, err)
	}
	defer resp.Body.Close()

	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, 0, fmt.Errorf("read %s response: %w", label, err)
	}

	return raw, resp.StatusCode, nil
}

func (h *ModelSquareHandler) configuredUpstreamURL(path string) string {
	h.mu.Lock()
	defer h.mu.Unlock()

	configuredURL := ""
	switch path {
	case findCGModelPath:
		configuredURL = h.modelURL
	case findCGKeysPath:
		configuredURL = h.keysURL
	case findCGGroupsPath:
		configuredURL = h.groupsURL
	case findCGLoginPath:
		configuredURL = h.loginURL
	}
	return h.upstreamURLLocked(path, configuredURL)
}

func (h *ModelSquareHandler) upstreamURLLocked(path, configuredURL string) string {
	if strings.TrimSpace(configuredURL) != "" {
		return strings.TrimSpace(configuredURL)
	}
	return h.baseURL + path
}

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
	accountRateGuardMaxAuditEntries = 100
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
	providers      []upstreamProviderRuntime

	mu    sync.Mutex
	token cachedFindCGToken

	syncRunMu             sync.Mutex
	accountRateGuardRunMu sync.Mutex

	backgroundMu      sync.Mutex
	backgroundStop    chan struct{}
	backgroundStopped chan struct{}

	accountRateGuardInterval time.Duration
	accountRateGuardMu       sync.Mutex
	accountRateGuardStop     chan struct{}
	accountRateGuardStopped  chan struct{}

	accountRateGuardAuditMu sync.Mutex
	accountRateGuardRunID   int64
	accountRateGuardLastRun *accountRateGuardRunRecord
	accountRateGuardAudits  []accountRateGuardAuditEntry
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

	AccountRateGuardInterval time.Duration
}

type modelSquareGroupProvider interface {
	GetAllGroups(ctx context.Context) ([]service.Group, error)
	UpdateGroup(ctx context.Context, id int64, input *service.UpdateGroupInput) (*service.Group, error)
	ListAccounts(ctx context.Context, page, pageSize int, platform, accountType, status, search string, groupID int64, privacyMode string, sortBy, sortOrder string) ([]service.Account, int64, error)
	UpdateAccount(ctx context.Context, id int64, input *service.UpdateAccountInput) (*service.Account, error)
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

type modelSquareKeySummaryResult struct {
	Groups []modelSquareKeySummaryGroup `json:"groups"`
}

type modelSquareKeySummaryGroup struct {
	Name           string                     `json:"name"`
	NormalizedName string                     `json:"normalized_name"`
	KeyCount       int                        `json:"key_count"`
	Keys           []modelSquareKeySummaryKey `json:"keys"`
}

type modelSquareKeySummaryKey struct {
	Name string `json:"name"`
}

type modelSquareRateWarningEntry struct {
	GroupID                int64   `json:"group_id"`
	GroupName              string  `json:"group_name"`
	LocalRateMultiplier    float64 `json:"local_rate_multiplier"`
	UpstreamRateMultiplier float64 `json:"upstream_rate_multiplier"`
}

type accountRateGuardRunRequest struct {
	DryRun bool `json:"dry_run"`
}

type accountRateGuardResult struct {
	DryRun              bool                        `json:"dry_run"`
	CheckedKeyCount     int                         `json:"checked_key_count"`
	MatchedAccountCount int                         `json:"matched_account_count"`
	ViolationCount      int                         `json:"violation_count"`
	UnboundCount        int                         `json:"unbound_count"`
	Violations          []accountRateGuardViolation `json:"violations"`
	Providers           []accountRateGuardProvider  `json:"providers"`
}

type accountRateGuardStatus struct {
	LastRun *accountRateGuardRunRecord   `json:"last_run,omitempty"`
	Audits  []accountRateGuardAuditEntry `json:"audits"`
}

type accountRateGuardRunRecord struct {
	RunID       int64                  `json:"run_id"`
	StartedAt   time.Time              `json:"started_at"`
	CompletedAt time.Time              `json:"completed_at"`
	Result      accountRateGuardResult `json:"result"`
	Error       string                 `json:"error,omitempty"`
}

type accountRateGuardAuditEntry struct {
	RunID                   int64     `json:"run_id"`
	CreatedAt               time.Time `json:"created_at"`
	ProviderSlug            string    `json:"provider_slug"`
	ProviderName            string    `json:"provider_name"`
	UpstreamKeyName         string    `json:"upstream_key_name"`
	MatchedLocalAccountID   int64     `json:"matched_local_account_id"`
	MatchedLocalAccountName string    `json:"matched_local_account_name"`
	UpstreamGroupName       string    `json:"upstream_group_name"`
	UpstreamRateMultiplier  float64   `json:"upstream_rate_multiplier"`
	LocalMinRateMultiplier  float64   `json:"local_min_rate_multiplier"`
	UnboundGroupIDs         []int64   `json:"unbound_group_ids"`
	UnboundGroupNames       []string  `json:"unbound_group_names"`
	RemainingGroupIDs       []int64   `json:"remaining_group_ids"`
}

type accountRateGuardProvider struct {
	Slug              string `json:"slug"`
	Name              string `json:"name"`
	AccountNamePrefix string `json:"account_name_prefix"`
	CheckedKeyCount   int    `json:"checked_key_count"`
	Error             string `json:"error,omitempty"`
}

type accountRateGuardViolation struct {
	ProviderSlug            string   `json:"provider_slug"`
	ProviderName            string   `json:"provider_name"`
	UpstreamKeyName         string   `json:"upstream_key_name"`
	MatchedLocalAccountID   int64    `json:"matched_local_account_id"`
	MatchedLocalAccountName string   `json:"matched_local_account_name"`
	UpstreamGroupName       string   `json:"upstream_group_name"`
	UpstreamRateMultiplier  float64  `json:"upstream_rate_multiplier"`
	LocalMinRateMultiplier  float64  `json:"local_min_rate_multiplier"`
	UnboundGroupIDs         []int64  `json:"unbound_group_ids,omitempty"`
	UnboundGroupNames       []string `json:"unbound_group_names,omitempty"`
}

type upstreamProviderRuntime struct {
	Slug               string
	Name               string
	BaseURL            string
	LoginURL           string
	KeysURL            string
	Email              string
	Password           string
	AccountNamePrefix  string
	TempDisableMinutes int
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
	accountRateGuardInterval := cfg.AccountRateGuardInterval
	providers := []upstreamProviderRuntime{}

	if cfg.AppConfig != nil {
		upstreamCfg := cfg.AppConfig.UpstreamManagement
		legacyCfg := cfg.AppConfig.ModelSquare
		providers = upstreamProviderRuntimesFromConfig(upstreamCfg, legacyCfg)
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
		if accountRateGuardInterval == 0 {
			if interval := firstPositiveInt(upstreamCfg.AccountRateGuardIntervalSeconds, legacyCfg.AccountRateGuardIntervalSeconds); interval > 0 {
				accountRateGuardInterval = time.Duration(interval) * time.Second
			}
		}
	}
	if accountRateGuardInterval == 0 {
		accountRateGuardInterval = 5 * time.Second
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
		providers:      providers,

		accountRateGuardInterval: accountRateGuardInterval,
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

// KeySummary handles GET /api/v1/admin/upstream-management/key-summary.
func (h *ModelSquareHandler) KeySummary(c *gin.Context) {
	result, err := h.collectKeySummaryFromKeys(c.Request.Context(), false)
	if err != nil {
		response.InternalError(c, err.Error())
		return
	}

	response.Success(c, result)
}

// RunAccountRateGuard checks upstream key group rates against same-name local accounts.
func (h *ModelSquareHandler) RunAccountRateGuard(c *gin.Context) {
	var req accountRateGuardRunRequest
	if c.Request.Body != nil {
		if err := c.ShouldBindJSON(&req); err != nil && err != io.EOF {
			response.BadRequest(c, "Invalid request: "+err.Error())
			return
		}
	}

	result, err := h.runAccountRateGuard(c.Request.Context(), req.DryRun)
	if err != nil {
		response.InternalError(c, err.Error())
		return
	}

	response.Success(c, result)
}

// GetAccountRateGuardStatus returns the most recent guard run and recent audit entries.
func (h *ModelSquareHandler) GetAccountRateGuardStatus(c *gin.Context) {
	response.Success(c, h.accountRateGuardStatusSnapshot())
}

// ListAccountRateGuardAudits returns recent guard unbind audit entries.
func (h *ModelSquareHandler) ListAccountRateGuardAudits(c *gin.Context) {
	response.Success(c, h.accountRateGuardAuditsSnapshot())
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

func (h *ModelSquareHandler) StartAccountRateGuard() {
	if h == nil || h.groupProvider == nil || h.accountRateGuardInterval <= 0 {
		return
	}

	h.accountRateGuardMu.Lock()
	defer h.accountRateGuardMu.Unlock()
	if h.accountRateGuardStop != nil {
		return
	}

	stop := make(chan struct{})
	stopped := make(chan struct{})
	h.accountRateGuardStop = stop
	h.accountRateGuardStopped = stopped
	go h.runAccountRateGuardBackground(stop, stopped)
}

func (h *ModelSquareHandler) StopAccountRateGuard() {
	if h == nil {
		return
	}

	h.accountRateGuardMu.Lock()
	stop := h.accountRateGuardStop
	stopped := h.accountRateGuardStopped
	h.accountRateGuardStop = nil
	h.accountRateGuardStopped = nil
	h.accountRateGuardMu.Unlock()

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

func (h *ModelSquareHandler) runAccountRateGuardBackground(stop <-chan struct{}, stopped chan<- struct{}) {
	defer close(stopped)

	for {
		ctx := context.Background()
		if providers, explicit := h.configuredAccountRateGuardProviders(ctx); explicit && len(providers) > 0 {
			if _, err := h.runAccountRateGuard(ctx, false); err != nil {
				slog.Warn("upstream account rate guard failed", "error", err)
			}
		}
		if !waitModelSquareSyncInterval(h.currentAccountRateGuardInterval(context.Background()), stop) {
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

func (h *ModelSquareHandler) currentAccountRateGuardInterval(ctx context.Context) time.Duration {
	if h == nil {
		return 0
	}
	if h.settingService != nil {
		if settings, err := h.settingService.GetAllSettings(ctx); err == nil && settings != nil {
			if settings.UpstreamManagementAccountRateGuardIntervalSeconds > 0 {
				return time.Duration(settings.UpstreamManagementAccountRateGuardIntervalSeconds) * time.Second
			}
		}
	}
	return h.accountRateGuardInterval
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

func (h *ModelSquareHandler) collectKeySummaryFromKeys(ctx context.Context, forceLogin bool) (modelSquareKeySummaryResult, error) {
	result := modelSquareKeySummaryResult{Groups: []modelSquareKeySummaryGroup{}}
	payload, status, err := h.requestModelSquareAuthenticated(ctx, findCGKeysPath, "model square keys", forceLogin)
	if err != nil {
		return result, err
	}
	if status < 200 || status >= 300 {
		return result, fmt.Errorf("model square keys upstream failed: HTTP %d: %s", status, string(payload))
	}

	var keysResp findCGKeysResponse
	if err := json.Unmarshal(payload, &keysResp); err != nil {
		return result, fmt.Errorf("decode model square keys response: %w", err)
	}
	if keysResp.Code != 0 {
		if keysResp.Message == "" {
			keysResp.Message = "unknown error"
		}
		return result, fmt.Errorf("model square keys upstream failed: %s", keysResp.Message)
	}

	indexByName := make(map[string]int)
	for _, item := range keysResp.Data.Items {
		if item.Group == nil {
			continue
		}
		name := strings.TrimSpace(item.Group.Name)
		normalizedName := normalizeModelSquareGroupName(name)
		if normalizedName == "" {
			continue
		}
		if index, ok := indexByName[normalizedName]; ok {
			result.Groups[index].KeyCount++
			result.Groups[index].Keys = append(result.Groups[index].Keys, modelSquareKeySummaryKey{Name: item.Name})
			continue
		}
		indexByName[normalizedName] = len(result.Groups)
		result.Groups = append(result.Groups, modelSquareKeySummaryGroup{
			Name:           name,
			NormalizedName: normalizedName,
			KeyCount:       1,
			Keys:           []modelSquareKeySummaryKey{{Name: item.Name}},
		})
	}

	return result, nil
}

func (h *ModelSquareHandler) runAccountRateGuard(ctx context.Context, dryRun bool) (accountRateGuardResult, error) {
	h.accountRateGuardRunMu.Lock()
	defer h.accountRateGuardRunMu.Unlock()

	runID := h.nextAccountRateGuardRunID()
	startedAt := time.Now()
	result := accountRateGuardResult{DryRun: dryRun, Violations: []accountRateGuardViolation{}, Providers: []accountRateGuardProvider{}}
	var runErr error
	defer func() {
		h.recordAccountRateGuardRun(accountRateGuardRunRecord{
			RunID:       runID,
			StartedAt:   startedAt,
			CompletedAt: time.Now(),
			Result:      result,
			Error:       errorString(runErr),
		})
	}()
	if h.groupProvider == nil {
		return result, nil
	}

	providers := h.accountRateGuardProviders(ctx)
	if len(providers) == 0 {
		return result, nil
	}

	accounts, err := h.loadAccountRateGuardAccounts(ctx)
	if err != nil {
		runErr = err
		return result, err
	}
	accountsByName := make(map[string][]service.Account, len(accounts))
	for _, account := range accounts {
		key := normalizeModelSquareGroupName(account.Name)
		if key == "" {
			continue
		}
		accountsByName[key] = append(accountsByName[key], account)
	}

	unboundAccounts := map[int64]struct{}{}
	for _, provider := range providers {
		providerResult := accountRateGuardProvider{
			Slug:              provider.Slug,
			Name:              provider.Name,
			AccountNamePrefix: provider.AccountNamePrefix,
		}
		keys, err := h.fetchKeysForProvider(ctx, provider)
		if err != nil {
			providerResult.Error = err.Error()
			result.Providers = append(result.Providers, providerResult)
			continue
		}
		providerResult.CheckedKeyCount = len(keys)
		result.Providers = append(result.Providers, providerResult)
		result.CheckedKeyCount += len(keys)

		for _, item := range keys {
			if item.Group == nil || item.Group.RateMultiplier == nil || *item.Group.RateMultiplier < 0 {
				continue
			}
			localNameKey := normalizeModelSquareGroupName(provider.AccountNamePrefix + item.Name)
			if localNameKey == "" {
				continue
			}
			matches := accountsByName[localNameKey]
			if len(matches) == 0 {
				continue
			}
			result.MatchedAccountCount += len(matches)
			for _, account := range matches {
				localMinRate, ok := accountMinGroupRate(account)
				if !ok || localMinRate+modelSquareRateCompareEpsilon >= *item.Group.RateMultiplier {
					continue
				}
				lowGroupIDs, lowGroupNames, remainingGroupIDs := accountRateGuardGroupChanges(account, *item.Group.RateMultiplier)
				if len(lowGroupIDs) == 0 {
					continue
				}
				violation := accountRateGuardViolation{
					ProviderSlug:            provider.Slug,
					ProviderName:            provider.Name,
					UpstreamKeyName:         item.Name,
					MatchedLocalAccountID:   account.ID,
					MatchedLocalAccountName: account.Name,
					UpstreamGroupName:       item.Group.Name,
					UpstreamRateMultiplier:  *item.Group.RateMultiplier,
					LocalMinRateMultiplier:  localMinRate,
					UnboundGroupIDs:         lowGroupIDs,
					UnboundGroupNames:       lowGroupNames,
				}
				result.Violations = append(result.Violations, violation)
				result.ViolationCount++
				if dryRun {
					continue
				}
				if _, alreadyUnbound := unboundAccounts[account.ID]; alreadyUnbound {
					continue
				}
				remaining := remainingGroupIDs
				if _, err := h.groupProvider.UpdateAccount(ctx, account.ID, &service.UpdateAccountInput{
					GroupIDs:              &remaining,
					SkipMixedChannelCheck: true,
				}); err != nil {
					runErr = fmt.Errorf("unbind low-rate groups for account %d: %w", account.ID, err)
					return result, runErr
				}
				h.recordAccountRateGuardAudit(accountRateGuardAuditEntry{
					RunID:                   runID,
					CreatedAt:               time.Now(),
					ProviderSlug:            violation.ProviderSlug,
					ProviderName:            violation.ProviderName,
					UpstreamKeyName:         violation.UpstreamKeyName,
					MatchedLocalAccountID:   violation.MatchedLocalAccountID,
					MatchedLocalAccountName: violation.MatchedLocalAccountName,
					UpstreamGroupName:       violation.UpstreamGroupName,
					UpstreamRateMultiplier:  violation.UpstreamRateMultiplier,
					LocalMinRateMultiplier:  violation.LocalMinRateMultiplier,
					UnboundGroupIDs:         append([]int64(nil), lowGroupIDs...),
					UnboundGroupNames:       append([]string(nil), lowGroupNames...),
					RemainingGroupIDs:       append([]int64(nil), remainingGroupIDs...),
				})
				unboundAccounts[account.ID] = struct{}{}
				result.UnboundCount += len(lowGroupIDs)
			}
		}
	}

	return result, nil
}

func (h *ModelSquareHandler) loadAccountRateGuardAccounts(ctx context.Context) ([]service.Account, error) {
	const pageSize = 1000
	all := []service.Account{}
	for page := 1; ; page++ {
		accounts, total, err := h.groupProvider.ListAccounts(ctx, page, pageSize, "", "", "", "", 0, "", "name", "asc")
		if err != nil {
			return nil, fmt.Errorf("load local accounts for upstream rate guard: %w", err)
		}
		all = append(all, accounts...)
		if len(all) >= int(total) || len(accounts) == 0 {
			return all, nil
		}
	}
}

func (h *ModelSquareHandler) nextAccountRateGuardRunID() int64 {
	h.accountRateGuardAuditMu.Lock()
	defer h.accountRateGuardAuditMu.Unlock()
	h.accountRateGuardRunID++
	return h.accountRateGuardRunID
}

func (h *ModelSquareHandler) recordAccountRateGuardRun(record accountRateGuardRunRecord) {
	h.accountRateGuardAuditMu.Lock()
	defer h.accountRateGuardAuditMu.Unlock()
	record.Result = cloneAccountRateGuardResult(record.Result)
	h.accountRateGuardLastRun = &record
}

func (h *ModelSquareHandler) recordAccountRateGuardAudit(entry accountRateGuardAuditEntry) {
	h.accountRateGuardAuditMu.Lock()
	defer h.accountRateGuardAuditMu.Unlock()
	entry.UnboundGroupIDs = append([]int64(nil), entry.UnboundGroupIDs...)
	entry.UnboundGroupNames = append([]string(nil), entry.UnboundGroupNames...)
	entry.RemainingGroupIDs = append([]int64(nil), entry.RemainingGroupIDs...)
	h.accountRateGuardAudits = append([]accountRateGuardAuditEntry{entry}, h.accountRateGuardAudits...)
	if len(h.accountRateGuardAudits) > accountRateGuardMaxAuditEntries {
		h.accountRateGuardAudits = h.accountRateGuardAudits[:accountRateGuardMaxAuditEntries]
	}
}

func (h *ModelSquareHandler) accountRateGuardStatusSnapshot() accountRateGuardStatus {
	h.accountRateGuardAuditMu.Lock()
	defer h.accountRateGuardAuditMu.Unlock()
	status := accountRateGuardStatus{Audits: cloneAccountRateGuardAudits(h.accountRateGuardAudits)}
	if h.accountRateGuardLastRun != nil {
		last := *h.accountRateGuardLastRun
		last.Result = cloneAccountRateGuardResult(last.Result)
		status.LastRun = &last
	}
	return status
}

func (h *ModelSquareHandler) accountRateGuardAuditsSnapshot() []accountRateGuardAuditEntry {
	h.accountRateGuardAuditMu.Lock()
	defer h.accountRateGuardAuditMu.Unlock()
	return cloneAccountRateGuardAudits(h.accountRateGuardAudits)
}

func cloneAccountRateGuardResult(result accountRateGuardResult) accountRateGuardResult {
	result.Violations = append([]accountRateGuardViolation(nil), result.Violations...)
	for i := range result.Violations {
		result.Violations[i].UnboundGroupIDs = append([]int64(nil), result.Violations[i].UnboundGroupIDs...)
		result.Violations[i].UnboundGroupNames = append([]string(nil), result.Violations[i].UnboundGroupNames...)
	}
	result.Providers = append([]accountRateGuardProvider(nil), result.Providers...)
	return result
}

func cloneAccountRateGuardAudits(entries []accountRateGuardAuditEntry) []accountRateGuardAuditEntry {
	out := make([]accountRateGuardAuditEntry, 0, len(entries))
	for _, entry := range entries {
		copied := entry
		copied.UnboundGroupIDs = append([]int64(nil), entry.UnboundGroupIDs...)
		copied.UnboundGroupNames = append([]string(nil), entry.UnboundGroupNames...)
		copied.RemainingGroupIDs = append([]int64(nil), entry.RemainingGroupIDs...)
		out = append(out, copied)
	}
	return out
}

func errorString(err error) string {
	if err == nil {
		return ""
	}
	return err.Error()
}

func (h *ModelSquareHandler) accountRateGuardProviders(ctx context.Context) []upstreamProviderRuntime {
	providers := []upstreamProviderRuntime{}
	if h == nil {
		return providers
	}
	if configured, explicit := h.configuredAccountRateGuardProviders(ctx); explicit {
		return configured
	}
	baseURL, loginURL, _, keysURL, _, email, password, _ := h.resolveConfigLocked(ctx)
	if strings.TrimSpace(keysURL) == "" {
		keysURL = findCGKeysPath
	}
	providers = append(providers, normalizeUpstreamProviderRuntime(upstreamProviderRuntime{
		Slug:               "default",
		Name:               "Default Upstream",
		BaseURL:            baseURL,
		LoginURL:           loginURL,
		KeysURL:            keysURL,
		Email:              email,
		Password:           password,
		AccountNamePrefix:  "",
		TempDisableMinutes: 1440,
	}))
	return providers
}

func (h *ModelSquareHandler) configuredAccountRateGuardProviders(ctx context.Context) ([]upstreamProviderRuntime, bool) {
	providers := []upstreamProviderRuntime{}
	if h == nil {
		return providers, false
	}
	if h.settingService != nil {
		if settings, err := h.settingService.GetAllSettings(ctx); err == nil && len(settings.UpstreamManagementProviders) > 0 {
			for _, provider := range settings.UpstreamManagementProviders {
				if !provider.Enabled {
					continue
				}
				providers = append(providers, normalizeUpstreamProviderRuntime(upstreamProviderRuntime{
					Slug:               provider.Slug,
					Name:               provider.Name,
					BaseURL:            firstNonEmpty(provider.BaseURL, settings.UpstreamManagementBaseURL),
					LoginURL:           firstNonEmpty(provider.LoginURL, settings.UpstreamManagementLoginURL),
					KeysURL:            firstNonEmpty(provider.APIKeysURL, provider.KeysURL, settings.UpstreamManagementAPIKeysURL),
					Email:              firstNonEmpty(provider.Email, settings.UpstreamManagementEmail),
					Password:           firstNonEmpty(provider.Password, settings.UpstreamManagementPassword),
					AccountNamePrefix:  provider.AccountNamePrefix,
					TempDisableMinutes: provider.TempDisableMinutes,
				}))
			}
			return providers, true
		}
	}
	if len(h.providers) > 0 {
		return append(providers, h.providers...), true
	}
	return providers, false
}

func upstreamProviderRuntimesFromConfig(upstreamCfg config.UpstreamManagementConfig, legacyCfg config.ModelSquareConfig) []upstreamProviderRuntime {
	providers := make([]upstreamProviderRuntime, 0, len(upstreamCfg.Providers)+len(legacyCfg.Providers))
	addProvider := func(providerCfg config.UpstreamManagementProviderConfig) {
		if !providerCfg.Enabled {
			return
		}
		provider := normalizeUpstreamProviderRuntime(upstreamProviderRuntime{
			Slug:               providerCfg.Slug,
			Name:               providerCfg.Name,
			BaseURL:            firstNonEmpty(providerCfg.BaseURL, upstreamCfg.BaseURL, legacyCfg.BaseURL),
			LoginURL:           firstNonEmpty(providerCfg.LoginURL, upstreamCfg.LoginURL, legacyCfg.LoginURL),
			KeysURL:            firstNonEmpty(providerCfg.APIKeysURL, providerCfg.KeysURL, upstreamCfg.APIKeysURL, upstreamCfg.KeysURL, legacyCfg.APIKeysURL, legacyCfg.KeysURL),
			Email:              firstNonEmpty(providerCfg.Email, upstreamCfg.Email, legacyCfg.Email),
			Password:           firstNonEmpty(providerCfg.Password, upstreamCfg.Password, legacyCfg.Password),
			AccountNamePrefix:  providerCfg.AccountNamePrefix,
			TempDisableMinutes: providerCfg.TempDisableMinutes,
		})
		providers = append(providers, provider)
	}
	for _, providerCfg := range upstreamCfg.Providers {
		addProvider(providerCfg)
	}
	if len(legacyCfg.Providers) > 0 && len(providers) == 0 {
		for _, providerCfg := range legacyCfg.Providers {
			addProvider(providerCfg)
		}
	}
	return providers
}

func normalizeUpstreamProviderRuntime(provider upstreamProviderRuntime) upstreamProviderRuntime {
	provider.Slug = strings.TrimSpace(provider.Slug)
	if provider.Slug == "" {
		provider.Slug = normalizeModelSquareGroupName(provider.Name)
	}
	if provider.Slug == "" {
		provider.Slug = "default"
	}
	provider.Name = strings.TrimSpace(provider.Name)
	if provider.Name == "" {
		provider.Name = provider.Slug
	}
	provider.BaseURL = strings.TrimRight(strings.TrimSpace(provider.BaseURL), "/")
	if provider.BaseURL == "" {
		provider.BaseURL = defaultFindCGBaseURL
	}
	provider.LoginURL = strings.TrimSpace(provider.LoginURL)
	provider.KeysURL = strings.TrimSpace(provider.KeysURL)
	if provider.KeysURL == "" {
		provider.KeysURL = findCGKeysPath
	}
	provider.Email = strings.TrimSpace(provider.Email)
	if provider.TempDisableMinutes <= 0 {
		provider.TempDisableMinutes = 1440
	}
	return provider
}

func accountMinGroupRate(account service.Account) (float64, bool) {
	minRate := 0.0
	found := false
	for _, group := range account.Groups {
		if group == nil {
			continue
		}
		if !found || group.RateMultiplier < minRate {
			minRate = group.RateMultiplier
			found = true
		}
	}
	return minRate, found
}

func accountRateGuardGroupChanges(account service.Account, upstreamRate float64) ([]int64, []string, []int64) {
	lowGroupIDs := []int64{}
	lowGroupNames := []string{}
	lowGroupIDSet := map[int64]struct{}{}
	for _, group := range account.Groups {
		if group == nil || group.ID <= 0 {
			continue
		}
		if modelSquareRemoteRateGreater(upstreamRate, group.RateMultiplier) {
			if _, ok := lowGroupIDSet[group.ID]; ok {
				continue
			}
			lowGroupIDSet[group.ID] = struct{}{}
			lowGroupIDs = append(lowGroupIDs, group.ID)
			lowGroupNames = append(lowGroupNames, group.Name)
		}
	}

	currentGroupIDs := account.GroupIDs
	if len(currentGroupIDs) == 0 {
		currentGroupIDs = make([]int64, 0, len(account.Groups))
		seen := map[int64]struct{}{}
		for _, group := range account.Groups {
			if group == nil || group.ID <= 0 {
				continue
			}
			if _, ok := seen[group.ID]; ok {
				continue
			}
			seen[group.ID] = struct{}{}
			currentGroupIDs = append(currentGroupIDs, group.ID)
		}
	}

	remainingGroupIDs := make([]int64, 0, len(currentGroupIDs))
	remainingSeen := map[int64]struct{}{}
	for _, groupID := range currentGroupIDs {
		if groupID <= 0 {
			continue
		}
		if _, low := lowGroupIDSet[groupID]; low {
			continue
		}
		if _, ok := remainingSeen[groupID]; ok {
			continue
		}
		remainingSeen[groupID] = struct{}{}
		remainingGroupIDs = append(remainingGroupIDs, groupID)
	}
	return lowGroupIDs, lowGroupNames, remainingGroupIDs
}

func (h *ModelSquareHandler) fetchKeysForProvider(ctx context.Context, provider upstreamProviderRuntime) ([]findCGKeyItem, error) {
	token := cachedFindCGToken{}
	if provider.Email != "" || provider.Password != "" {
		nextToken, err := h.loginProvider(ctx, provider)
		if err != nil {
			return nil, err
		}
		token = nextToken
	}
	payload, status, err := h.requestProviderWithToken(ctx, provider, token, provider.KeysURL, "upstream provider keys")
	if err != nil {
		return nil, err
	}
	if status < 200 || status >= 300 {
		return nil, fmt.Errorf("upstream provider keys failed: HTTP %d: %s", status, string(payload))
	}
	var keysResp findCGKeysResponse
	if err := json.Unmarshal(payload, &keysResp); err != nil {
		return nil, fmt.Errorf("decode upstream provider keys response: %w", err)
	}
	if keysResp.Code != 0 {
		if keysResp.Message == "" {
			keysResp.Message = "unknown error"
		}
		return nil, fmt.Errorf("upstream provider keys failed: %s", keysResp.Message)
	}
	return keysResp.Data.Items, nil
}

func (h *ModelSquareHandler) loginProvider(ctx context.Context, provider upstreamProviderRuntime) (cachedFindCGToken, error) {
	body, err := json.Marshal(map[string]string{
		"email":    provider.Email,
		"password": provider.Password,
	})
	if err != nil {
		return cachedFindCGToken{}, fmt.Errorf("marshal upstream provider login payload: %w", err)
	}
	loginURL := provider.LoginURL
	if loginURL == "" {
		loginURL = findCGLoginPath
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, providerURL(provider, loginURL), bytes.NewReader(body))
	if err != nil {
		return cachedFindCGToken{}, fmt.Errorf("create upstream provider login request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := h.httpClient.Do(req)
	if err != nil {
		return cachedFindCGToken{}, fmt.Errorf("upstream provider login request failed: %w", err)
	}
	defer resp.Body.Close()
	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return cachedFindCGToken{}, fmt.Errorf("read upstream provider login response: %w", err)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return cachedFindCGToken{}, fmt.Errorf("upstream provider login failed: HTTP %d: %s", resp.StatusCode, string(raw))
	}
	var loginResp findCGLoginResponse
	if err := json.Unmarshal(raw, &loginResp); err != nil {
		return cachedFindCGToken{}, fmt.Errorf("decode upstream provider login response: %w", err)
	}
	if loginResp.Code != 0 || loginResp.Data.AccessToken == "" {
		if loginResp.Message == "" {
			loginResp.Message = "missing access_token"
		}
		return cachedFindCGToken{}, fmt.Errorf("upstream provider login failed: %s", loginResp.Message)
	}
	tokenType := loginResp.Data.TokenType
	if tokenType == "" {
		tokenType = "Bearer"
	}
	return cachedFindCGToken{AccessToken: loginResp.Data.AccessToken, TokenType: tokenType}, nil
}

func (h *ModelSquareHandler) requestProviderWithToken(ctx context.Context, provider upstreamProviderRuntime, token cachedFindCGToken, path, label string) ([]byte, int, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, providerURL(provider, path), nil)
	if err != nil {
		return nil, 0, fmt.Errorf("create %s request: %w", label, err)
	}
	if token.AccessToken != "" {
		if token.TokenType == "" {
			token.TokenType = "Bearer"
		}
		req.Header.Set("Authorization", token.TokenType+" "+token.AccessToken)
	}
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

func providerURL(provider upstreamProviderRuntime, path string) string {
	if strings.HasPrefix(path, "http://") || strings.HasPrefix(path, "https://") {
		return path
	}
	if strings.HasPrefix(path, "/") {
		return strings.TrimRight(provider.BaseURL, "/") + path
	}
	return strings.TrimRight(provider.BaseURL, "/") + "/" + path
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

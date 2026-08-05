package service

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"regexp"
	"strings"
	"time"
)

const (
	SupplierProviderAuthEventCacheHit         SupplierProviderAuthEventType = "cache_hit"
	SupplierProviderAuthEventCacheMiss        SupplierProviderAuthEventType = "cache_miss"
	SupplierProviderAuthEventLoginSuccess     SupplierProviderAuthEventType = "login_success"
	SupplierProviderAuthEventLoginFailed      SupplierProviderAuthEventType = "login_failed"
	SupplierProviderAuthEventCacheInvalidated SupplierProviderAuthEventType = "cache_invalidated"
	SupplierProviderAuthEventCacheError       SupplierProviderAuthEventType = "cache_error"

	SupplierProviderAuthSourceSync         SupplierProviderAuthSource = "sync"
	SupplierProviderAuthSourceEndpointTest SupplierProviderAuthSource = "endpoint_test"
	SupplierProviderAuthSourceManual       SupplierProviderAuthSource = "manual"
	SupplierProviderAuthSourceUnknown      SupplierProviderAuthSource = "unknown"

	SupplierProviderAuthStatusSuccess     SupplierProviderAuthStatus = "success"
	SupplierProviderAuthStatusMiss        SupplierProviderAuthStatus = "miss"
	SupplierProviderAuthStatusFailed      SupplierProviderAuthStatus = "failed"
	SupplierProviderAuthStatusInvalidated SupplierProviderAuthStatus = "invalidated"
	SupplierProviderAuthStatusUnavailable SupplierProviderAuthStatus = "unavailable"

	SupplierProviderAuthCacheCached  = "cached"
	SupplierProviderAuthCacheMissing = "missing"
	SupplierProviderAuthCacheExpired = "expired"
	SupplierProviderAuthCacheError   = "error"

	supplierProviderAuthErrorLimit = 512
)

type SupplierProviderAuthEventType string
type SupplierProviderAuthSource string
type SupplierProviderAuthStatus string

type supplierProviderAuthSourceContextKey struct{}

func WithSupplierProviderAuthSource(ctx context.Context, source SupplierProviderAuthSource) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	return context.WithValue(ctx, supplierProviderAuthSourceContextKey{}, source)
}

func supplierProviderAuthSourceFromContext(ctx context.Context) SupplierProviderAuthSource {
	if ctx != nil {
		if source, ok := ctx.Value(supplierProviderAuthSourceContextKey{}).(SupplierProviderAuthSource); ok && source != "" {
			return source
		}
	}
	return SupplierProviderAuthSourceUnknown
}

type SupplierProviderAuthEventInput struct {
	ProviderID int64
	EventType  SupplierProviderAuthEventType
	Source     SupplierProviderAuthSource
	Status     SupplierProviderAuthStatus
	StartedAt  time.Time
	FinishedAt time.Time
	HTTPStatus *int
	Error      error
	Token      *SupplierProviderAuthToken
}

type SupplierProviderAuthEventRecord struct {
	ID               int64
	ProviderID       int64
	EventType        SupplierProviderAuthEventType
	Source           SupplierProviderAuthSource
	Status           SupplierProviderAuthStatus
	StartedAt        time.Time
	FinishedAt       time.Time
	DurationMS       int64
	HTTPStatus       *int
	ErrorMessage     string
	TokenFingerprint string
	TokenLength      int
	TokenExpiresAt   *time.Time
	CookiePresent    bool
	CreatedAt        time.Time
}

type SupplierProviderAuthSummary struct {
	LoginCount           int64      `json:"login_count"`
	LoginSuccessCount    int64      `json:"login_success_count"`
	LoginFailureCount    int64      `json:"login_failure_count"`
	CacheHitCount        int64      `json:"cache_hit_count"`
	CacheMissCount       int64      `json:"cache_miss_count"`
	LastLoginAt          *time.Time `json:"last_login_at,omitempty"`
	LastLoginStatus      string     `json:"last_login_status"`
	LastLoginError       string     `json:"last_login_error"`
	LastCacheHitAt       *time.Time `json:"last_cache_hit_at,omitempty"`
	LastCacheError       string     `json:"last_cache_error"`
	LastTokenExpiresAt   *time.Time `json:"last_token_expires_at,omitempty"`
	LastTokenFingerprint string     `json:"last_token_fingerprint"`
}

type SupplierProviderAuthHistoryParams struct {
	Page      int
	PageSize  int
	EventType SupplierProviderAuthEventType
}

type SupplierProviderAuthHistoryItem struct {
	ID               int64                         `json:"id"`
	ProviderID       int64                         `json:"provider_id"`
	EventType        SupplierProviderAuthEventType `json:"event_type"`
	Source           SupplierProviderAuthSource    `json:"source"`
	Status           SupplierProviderAuthStatus    `json:"status"`
	StartedAt        time.Time                     `json:"started_at"`
	FinishedAt       time.Time                     `json:"finished_at"`
	DurationMS       int64                         `json:"duration_ms"`
	HTTPStatus       *int                          `json:"http_status,omitempty"`
	ErrorMessage     string                        `json:"error_message,omitempty"`
	TokenFingerprint string                        `json:"token_fingerprint,omitempty"`
	TokenExpiresAt   *time.Time                    `json:"token_expires_at,omitempty"`
	TokenLength      int                           `json:"token_length,omitempty"`
	CookiePresent    bool                          `json:"cookie_present"`
	CreatedAt        time.Time                     `json:"created_at"`
}

type SupplierProviderAuthHistoryResult struct {
	Items    []SupplierProviderAuthHistoryItem `json:"items"`
	Total    int64                             `json:"total"`
	Page     int                               `json:"page"`
	PageSize int                               `json:"page_size"`
}

type SupplierProviderAuthTokenSnapshot struct {
	Status           string     `json:"status"`
	Cached           bool       `json:"cached"`
	TokenType        string     `json:"token_type,omitempty"`
	TokenSummary     string     `json:"token_summary,omitempty"`
	TokenLength      int        `json:"token_length,omitempty"`
	TokenFingerprint string     `json:"token_fingerprint,omitempty"`
	TokenExpiresAt   *time.Time `json:"token_expires_at,omitempty"`
	RemainingSeconds int64      `json:"remaining_seconds"`
	TTLSeconds       int64      `json:"ttl_seconds"`
	CookiePresent    bool       `json:"cookie_present"`
	Error            string     `json:"error,omitempty"`
}

type SupplierProviderAuthLockSnapshot struct {
	Held             bool   `json:"held"`
	Status           string `json:"status"`
	RemainingSeconds int64  `json:"remaining_seconds"`
	Error            string `json:"error,omitempty"`
}

type SupplierProviderAuthStatusResult struct {
	ProviderID int64                             `json:"provider_id"`
	Summary    SupplierProviderAuthSummary       `json:"summary"`
	Cache      SupplierProviderAuthTokenSnapshot `json:"cache"`
	LoginLock  SupplierProviderAuthLockSnapshot  `json:"login_lock"`
	CheckedAt  time.Time                         `json:"checked_at"`
}

type SupplierProviderAuthAuditor interface {
	Record(ctx context.Context, event SupplierProviderAuthEventInput) error
}

type SupplierProviderAuthAuditRepository interface {
	Record(ctx context.Context, event SupplierProviderAuthEventRecord) error
	GetSummary(ctx context.Context, providerID int64) (SupplierProviderAuthSummary, error)
	ListHistory(ctx context.Context, providerID int64, params SupplierProviderAuthHistoryParams) (SupplierProviderAuthHistoryResult, error)
}

type SupplierProviderAuthAuditService struct {
	repo       SupplierProviderAuthAuditRepository
	tokenCache SupplierProviderTokenCache
}

func NewSupplierProviderAuthAuditService(repo SupplierProviderAuthAuditRepository, tokenCache SupplierProviderTokenCache) *SupplierProviderAuthAuditService {
	return &SupplierProviderAuthAuditService{repo: repo, tokenCache: tokenCache}
}

func (s *SupplierProviderAuthAuditService) Record(ctx context.Context, event SupplierProviderAuthEventInput) error {
	if s == nil || s.repo == nil {
		return errors.New("supplier provider auth audit repository is unavailable")
	}
	if event.ProviderID <= 0 {
		return errors.New("supplier provider id must be positive")
	}
	return s.repo.Record(ctx, normalizeSupplierProviderAuthEvent(event))
}

func (s *SupplierProviderAuthAuditService) GetStatus(ctx context.Context, providerID int64) (SupplierProviderAuthStatusResult, error) {
	if providerID <= 0 {
		return SupplierProviderAuthStatusResult{}, errors.New("supplier provider id must be positive")
	}
	if s == nil || s.repo == nil {
		return SupplierProviderAuthStatusResult{}, errors.New("supplier provider auth audit repository is unavailable")
	}
	summary, err := s.repo.GetSummary(ctx, providerID)
	if err != nil {
		return SupplierProviderAuthStatusResult{}, err
	}
	status := SupplierProviderAuthStatusResult{
		ProviderID: providerID,
		Summary:    summary,
		CheckedAt:  time.Now(),
	}
	if s.tokenCache == nil {
		status.Cache = SupplierProviderAuthTokenSnapshot{Status: SupplierProviderAuthCacheError, Error: "token cache is unavailable"}
		status.LoginLock = SupplierProviderAuthLockSnapshot{Status: SupplierProviderAuthCacheError, Error: "token cache is unavailable"}
		return status, nil
	}
	inspector, ok := s.tokenCache.(SupplierProviderTokenCacheInspector)
	if !ok {
		status.Cache = SupplierProviderAuthTokenSnapshot{Status: SupplierProviderAuthCacheError, Error: "token cache status is unavailable"}
		status.LoginLock = SupplierProviderAuthLockSnapshot{Status: SupplierProviderAuthCacheError, Error: "token cache status is unavailable"}
		return status, nil
	}
	snapshot, err := inspector.Inspect(ctx, providerID)
	if err != nil {
		message := sanitizeSupplierProviderAuthError(err)
		status.Cache = SupplierProviderAuthTokenSnapshot{Status: SupplierProviderAuthCacheError, Error: message}
		status.LoginLock = SupplierProviderAuthLockSnapshot{Status: SupplierProviderAuthCacheError, Error: message}
		return status, nil
	}
	now := status.CheckedAt
	if snapshot.Found {
		status.Cache = publicSupplierProviderAuthTokenSnapshot(snapshot.Token, snapshot.TTL, now)
	} else {
		status.Cache = SupplierProviderAuthTokenSnapshot{Status: SupplierProviderAuthCacheMissing, TTLSeconds: positiveDurationSeconds(snapshot.TTL)}
	}
	status.LoginLock = SupplierProviderAuthLockSnapshot{
		Held:             snapshot.LockHeld,
		Status:           "available",
		RemainingSeconds: positiveDurationSeconds(snapshot.LockTTL),
	}
	if snapshot.LockHeld {
		status.LoginLock.Status = "locked"
	}
	return status, nil
}

func (s *SupplierProviderAuthAuditService) ListHistory(ctx context.Context, providerID int64, params SupplierProviderAuthHistoryParams) (SupplierProviderAuthHistoryResult, error) {
	if providerID <= 0 {
		return SupplierProviderAuthHistoryResult{}, errors.New("supplier provider id must be positive")
	}
	if s == nil || s.repo == nil {
		return SupplierProviderAuthHistoryResult{}, errors.New("supplier provider auth audit repository is unavailable")
	}
	if params.Page < 1 {
		params.Page = 1
	}
	if params.PageSize < 1 || params.PageSize > 100 {
		params.PageSize = 20
	}
	return s.repo.ListHistory(ctx, providerID, params)
}

func normalizeSupplierProviderAuthEvent(event SupplierProviderAuthEventInput) SupplierProviderAuthEventRecord {
	startedAt := event.StartedAt
	if startedAt.IsZero() {
		startedAt = time.Now()
	}
	finishedAt := event.FinishedAt
	if finishedAt.IsZero() {
		finishedAt = startedAt
	}
	if finishedAt.Before(startedAt) {
		finishedAt = startedAt
	}
	source := event.Source
	if source == "" {
		source = SupplierProviderAuthSourceUnknown
	}
	status := event.Status
	if status == "" {
		switch event.EventType {
		case SupplierProviderAuthEventCacheHit, SupplierProviderAuthEventLoginSuccess:
			status = SupplierProviderAuthStatusSuccess
		case SupplierProviderAuthEventCacheMiss:
			status = SupplierProviderAuthStatusMiss
		case SupplierProviderAuthEventLoginFailed:
			status = SupplierProviderAuthStatusFailed
		case SupplierProviderAuthEventCacheInvalidated:
			status = SupplierProviderAuthStatusInvalidated
		case SupplierProviderAuthEventCacheError:
			status = SupplierProviderAuthStatusUnavailable
		}
	}
	record := SupplierProviderAuthEventRecord{
		ProviderID:   event.ProviderID,
		EventType:    event.EventType,
		Source:       source,
		Status:       status,
		StartedAt:    startedAt,
		FinishedAt:   finishedAt,
		DurationMS:   finishedAt.Sub(startedAt).Milliseconds(),
		HTTPStatus:   event.HTTPStatus,
		ErrorMessage: sanitizeSupplierProviderAuthError(event.Error),
		CreatedAt:    finishedAt,
	}
	if event.Token != nil {
		token := *event.Token
		accessToken := strings.TrimSpace(token.AccessToken)
		record.TokenFingerprint = supplierProviderAuthTokenFingerprint(accessToken)
		record.TokenLength = len(accessToken)
		record.CookiePresent = strings.TrimSpace(token.CookieHeader) != ""
		if !token.ExpiresAt.IsZero() {
			expiresAt := token.ExpiresAt
			record.TokenExpiresAt = &expiresAt
		}
	}
	return record
}

func publicSupplierProviderAuthTokenSnapshot(token SupplierProviderAuthToken, ttl time.Duration, now time.Time) SupplierProviderAuthTokenSnapshot {
	accessToken := strings.TrimSpace(token.AccessToken)
	cookiePresent := strings.TrimSpace(token.CookieHeader) != ""
	authPresent := accessToken != "" || cookiePresent
	snapshot := SupplierProviderAuthTokenSnapshot{
		Status:           SupplierProviderAuthCacheMissing,
		Cached:           authPresent,
		TokenType:        strings.TrimSpace(token.TokenType),
		TokenSummary:     supplierProviderAuthTokenSummary(accessToken),
		TokenLength:      len(accessToken),
		TokenFingerprint: supplierProviderAuthTokenFingerprint(accessToken),
		RemainingSeconds: 0,
		TTLSeconds:       positiveDurationSeconds(ttl),
		CookiePresent:    cookiePresent,
	}
	if snapshot.TokenType == "" && accessToken != "" {
		snapshot.TokenType = "Bearer"
	}
	if !token.ExpiresAt.IsZero() {
		expiresAt := token.ExpiresAt
		snapshot.TokenExpiresAt = &expiresAt
		if token.ExpiresAt.After(now) && ttl > 0 {
			snapshot.Status = SupplierProviderAuthCacheCached
			snapshot.RemainingSeconds = positiveDurationSeconds(token.ExpiresAt.Sub(now))
		} else if authPresent {
			snapshot.Status = SupplierProviderAuthCacheExpired
			snapshot.TTLSeconds = 0
		}
	} else if authPresent && ttl > 0 {
		snapshot.Status = SupplierProviderAuthCacheCached
	} else if authPresent {
		snapshot.Status = SupplierProviderAuthCacheExpired
		snapshot.TTLSeconds = 0
	}
	return snapshot
}

func supplierProviderAuthTokenSummary(token string) string {
	if token == "" {
		return ""
	}
	if len(token) <= 8 {
		return strings.Repeat("•", len(token))
	}
	return token[:4] + "…" + token[len(token)-4:]
}

func supplierProviderAuthTokenFingerprint(token string) string {
	if token == "" {
		return ""
	}
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}

func positiveDurationSeconds(value time.Duration) int64 {
	if value <= 0 {
		return 0
	}
	return int64(value / time.Second)
}

var supplierProviderAuthAuthorizationPattern = regexp.MustCompile(`(?i)(authorization\s*[:=]\s*)(?:bearer\s+)?[^\s,;]+`)
var supplierProviderAuthCookiePattern = regexp.MustCompile(`(?i)(cookie(?:[_-]?header)?\s*[:=]\s*)[^\r\n]+`)
var supplierProviderAuthSecretPattern = regexp.MustCompile(`(?i)(["']?(?:access[_-]?token|refresh[_-]?token|token|password|secret|api[_-]?key)["']?\s*[:=]\s*)(?:"[^"]*"|'[^']*'|[^\s,;}]+)`)
var supplierProviderAuthURLPattern = regexp.MustCompile(`(?i)(?:https?|redis)://[^\s,;]+`)

func sanitizeSupplierProviderAuthError(err error) string {
	if err == nil {
		return ""
	}
	message := strings.TrimSpace(err.Error())
	message = supplierProviderAuthURLPattern.ReplaceAllString(message, "[REDACTED_URL]")
	message = supplierProviderAuthAuthorizationPattern.ReplaceAllString(message, "$1[REDACTED]")
	message = supplierProviderAuthSecretPattern.ReplaceAllString(message, "$1[REDACTED]")
	message = supplierProviderAuthCookiePattern.ReplaceAllString(message, "$1[REDACTED]")
	if len(message) > supplierProviderAuthErrorLimit {
		message = message[:supplierProviderAuthErrorLimit]
	}
	return message
}

func supplierProviderAuthHTTPStatus(err error) *int {
	if err == nil {
		return nil
	}
	var statusErr supplierSub2APIHTTPStatusError
	if !errors.As(err, &statusErr) || statusErr.status <= 0 {
		return nil
	}
	status := statusErr.status
	return &status
}

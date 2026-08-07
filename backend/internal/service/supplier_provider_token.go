package service

import (
	"context"
	"time"
)

type SupplierProviderAuthToken struct {
	AccessToken  string    `json:"access_token"`
	RefreshToken string    `json:"refresh_token,omitempty"`
	TokenType    string    `json:"token_type"`
	ExpiresAt    time.Time `json:"expires_at"`
	UserID       int64     `json:"user_id,omitempty"`
	CookieHeader string    `json:"cookie_header,omitempty"`
}

type SupplierProviderTokenCache interface {
	Get(ctx context.Context, providerID int64) (SupplierProviderAuthToken, bool, error)
	Set(ctx context.Context, providerID int64, token SupplierProviderAuthToken, ttl time.Duration) error
	Delete(ctx context.Context, providerID int64) error
	TryAcquireLoginLock(ctx context.Context, providerID int64, owner string, ttl time.Duration) (bool, error)
	ReleaseLoginLock(ctx context.Context, providerID int64, owner string) error
}

// SupplierProviderTokenCacheInspector 只用于管理端读取当前缓存状态，完整 Token 不应直接暴露给 HTTP 层。
type SupplierProviderTokenCacheInspector interface {
	Inspect(ctx context.Context, providerID int64) (SupplierProviderTokenCacheSnapshot, error)
}

type SupplierProviderTokenCacheSnapshot struct {
	Token    SupplierProviderAuthToken
	Found    bool
	TTL      time.Duration
	LockHeld bool
	LockTTL  time.Duration
}

type SupplierProviderSyncLock interface {
	TryAcquireSyncLock(ctx context.Context, providerID int64, owner string, ttl time.Duration) (bool, error)
	ReleaseSyncLock(ctx context.Context, providerID int64, owner string) error
}

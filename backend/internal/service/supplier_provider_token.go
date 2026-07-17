package service

import (
	"context"
	"time"
)

type SupplierProviderAuthToken struct {
	AccessToken string    `json:"access_token"`
	TokenType   string    `json:"token_type"`
	ExpiresAt   time.Time `json:"expires_at"`
}

type SupplierProviderTokenCache interface {
	Get(ctx context.Context, providerID int64) (SupplierProviderAuthToken, bool, error)
	Set(ctx context.Context, providerID int64, token SupplierProviderAuthToken, ttl time.Duration) error
	Delete(ctx context.Context, providerID int64) error
	TryAcquireLoginLock(ctx context.Context, providerID int64, owner string, ttl time.Duration) (bool, error)
	ReleaseLoginLock(ctx context.Context, providerID int64, owner string) error
}

type SupplierProviderSyncLock interface {
	TryAcquireSyncLock(ctx context.Context, providerID int64, owner string, ttl time.Duration) (bool, error)
	ReleaseSyncLock(ctx context.Context, providerID int64, owner string) error
}

func SupplierProviderTokenTTL(expiresIn time.Duration) time.Duration {
	if expiresIn <= 0 {
		return 30 * time.Minute
	}

	safetyWindow := time.Minute
	if expiresIn <= 2*time.Minute {
		safetyWindow = expiresIn / 10
	}
	return expiresIn - safetyWindow
}

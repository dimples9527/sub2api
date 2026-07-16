package repository

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/redis/go-redis/v9"
)

const (
	supplierProviderTokenKeyPrefix     = "supplier:provider:auth:"
	supplierProviderLoginLockKeyPrefix = "supplier:provider:auth-lock:"
	supplierProviderSyncLockKeyPrefix  = "supplier:provider:sync-lock:"
)

var supplierProviderLockReleaseScript = redis.NewScript(`
if redis.call("GET", KEYS[1]) == ARGV[1] then
  return redis.call("DEL", KEYS[1])
end
return 0
`)

type SupplierProviderTokenRedisCache struct {
	rdb *redis.Client
}

func NewSupplierProviderTokenCache(rdb *redis.Client) *SupplierProviderTokenRedisCache {
	return &SupplierProviderTokenRedisCache{rdb: rdb}
}

func (c *SupplierProviderTokenRedisCache) Get(ctx context.Context, providerID int64) (service.SupplierProviderAuthToken, bool, error) {
	if err := c.validateProvider(providerID); err != nil {
		return service.SupplierProviderAuthToken{}, false, err
	}
	payload, err := c.rdb.Get(ctx, supplierProviderTokenKey(providerID)).Bytes()
	if errors.Is(err, redis.Nil) {
		return service.SupplierProviderAuthToken{}, false, nil
	}
	if err != nil {
		return service.SupplierProviderAuthToken{}, false, fmt.Errorf("get supplier provider token: %w", err)
	}

	var token service.SupplierProviderAuthToken
	if err := json.Unmarshal(payload, &token); err != nil {
		return service.SupplierProviderAuthToken{}, false, fmt.Errorf("decode supplier provider token: %w", err)
	}
	return token, true, nil
}

func (c *SupplierProviderTokenRedisCache) Set(ctx context.Context, providerID int64, token service.SupplierProviderAuthToken, ttl time.Duration) error {
	if err := c.validateProvider(providerID); err != nil {
		return err
	}
	if ttl <= 0 {
		ttl = service.SupplierProviderTokenTTL(ttl)
	}
	payload, err := json.Marshal(token)
	if err != nil {
		return fmt.Errorf("encode supplier provider token: %w", err)
	}
	if err := c.rdb.Set(ctx, supplierProviderTokenKey(providerID), payload, ttl).Err(); err != nil {
		return fmt.Errorf("set supplier provider token: %w", err)
	}
	return nil
}

func (c *SupplierProviderTokenRedisCache) Delete(ctx context.Context, providerID int64) error {
	if err := c.validateProvider(providerID); err != nil {
		return err
	}
	if err := c.rdb.Del(ctx, supplierProviderTokenKey(providerID)).Err(); err != nil {
		return fmt.Errorf("delete supplier provider token: %w", err)
	}
	return nil
}

func (c *SupplierProviderTokenRedisCache) TryAcquireLoginLock(ctx context.Context, providerID int64, owner string, ttl time.Duration) (bool, error) {
	if err := c.validateLock(providerID, owner, ttl); err != nil {
		return false, err
	}
	acquired, err := c.rdb.SetNX(ctx, supplierProviderLoginLockKey(providerID), owner, ttl).Result()
	if err != nil {
		return false, fmt.Errorf("acquire supplier provider login lock: %w", err)
	}
	return acquired, nil
}

func (c *SupplierProviderTokenRedisCache) ReleaseLoginLock(ctx context.Context, providerID int64, owner string) error {
	if err := c.validateLockOwner(providerID, owner); err != nil {
		return err
	}
	if err := supplierProviderLockReleaseScript.Run(ctx, c.rdb, []string{supplierProviderLoginLockKey(providerID)}, owner).Err(); err != nil {
		return fmt.Errorf("release supplier provider login lock: %w", err)
	}
	return nil
}

func (c *SupplierProviderTokenRedisCache) TryAcquireSyncLock(ctx context.Context, providerID int64, owner string, ttl time.Duration) (bool, error) {
	if err := c.validateLock(providerID, owner, ttl); err != nil {
		return false, err
	}
	acquired, err := c.rdb.SetNX(ctx, supplierProviderSyncLockKey(providerID), owner, ttl).Result()
	if err != nil {
		return false, fmt.Errorf("acquire supplier provider sync lock: %w", err)
	}
	return acquired, nil
}

func (c *SupplierProviderTokenRedisCache) ReleaseSyncLock(ctx context.Context, providerID int64, owner string) error {
	if err := c.validateLockOwner(providerID, owner); err != nil {
		return err
	}
	if err := supplierProviderLockReleaseScript.Run(ctx, c.rdb, []string{supplierProviderSyncLockKey(providerID)}, owner).Err(); err != nil {
		return fmt.Errorf("release supplier provider sync lock: %w", err)
	}
	return nil
}

func (c *SupplierProviderTokenRedisCache) validateProvider(providerID int64) error {
	if c == nil || c.rdb == nil {
		return errors.New("supplier provider token cache is unavailable")
	}
	if providerID <= 0 {
		return errors.New("supplier provider id must be positive")
	}
	return nil
}

func (c *SupplierProviderTokenRedisCache) validateLock(providerID int64, owner string, ttl time.Duration) error {
	if err := c.validateLockOwner(providerID, owner); err != nil {
		return err
	}
	if ttl <= 0 {
		return errors.New("supplier provider lock ttl must be positive")
	}
	return nil
}

func (c *SupplierProviderTokenRedisCache) validateLockOwner(providerID int64, owner string) error {
	if err := c.validateProvider(providerID); err != nil {
		return err
	}
	if strings.TrimSpace(owner) == "" {
		return errors.New("supplier provider lock owner is required")
	}
	return nil
}

func supplierProviderTokenKey(providerID int64) string {
	return supplierProviderTokenKeyPrefix + strconv.FormatInt(providerID, 10)
}

func supplierProviderLoginLockKey(providerID int64) string {
	return supplierProviderLoginLockKeyPrefix + strconv.FormatInt(providerID, 10)
}

func supplierProviderSyncLockKey(providerID int64) string {
	return supplierProviderSyncLockKeyPrefix + strconv.FormatInt(providerID, 10)
}

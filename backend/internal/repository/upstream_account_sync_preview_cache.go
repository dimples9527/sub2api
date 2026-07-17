package repository

import (
	"context"
	"encoding/json"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/redis/go-redis/v9"
)

const upstreamAccountSyncPreviewCacheKey = "upstream:accounts:sync-preview:v3"

type upstreamAccountSyncPreviewCache struct {
	rdb *redis.Client
}

func NewUpstreamAccountSyncPreviewCache(rdb *redis.Client) service.UpstreamAccountSyncPreviewCache {
	return &upstreamAccountSyncPreviewCache{rdb: rdb}
}

func (c *upstreamAccountSyncPreviewCache) Get(ctx context.Context) (service.UpstreamAccountSyncResult, bool, error) {
	if c == nil || c.rdb == nil {
		return service.UpstreamAccountSyncResult{}, false, nil
	}
	raw, err := c.rdb.Get(ctx, upstreamAccountSyncPreviewCacheKey).Bytes()
	if err != nil {
		if err == redis.Nil {
			return service.UpstreamAccountSyncResult{}, false, nil
		}
		return service.UpstreamAccountSyncResult{}, false, err
	}
	var result service.UpstreamAccountSyncResult
	if err := json.Unmarshal(raw, &result); err != nil {
		return service.UpstreamAccountSyncResult{}, false, err
	}
	return result, true, nil
}

func (c *upstreamAccountSyncPreviewCache) Set(ctx context.Context, result service.UpstreamAccountSyncResult) error {
	if c == nil || c.rdb == nil {
		return nil
	}
	result.Records = nil
	raw, err := json.Marshal(result)
	if err != nil {
		return err
	}
	return c.rdb.Set(ctx, upstreamAccountSyncPreviewCacheKey, raw, 0).Err()
}

func (c *upstreamAccountSyncPreviewCache) Delete(ctx context.Context) error {
	if c == nil || c.rdb == nil {
		return nil
	}
	return c.rdb.Del(ctx, upstreamAccountSyncPreviewCacheKey).Err()
}

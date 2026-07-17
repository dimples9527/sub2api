package repository

import (
	"context"
	"strings"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/redis/go-redis/v9"
)

const supplierAutomationLockKeyPrefix = "supplier:automation:lock:"

var supplierAutomationReleaseScript = redis.NewScript(`
if redis.call("GET", KEYS[1]) == ARGV[1] then
  return redis.call("DEL", KEYS[1])
end
return 0
`)

type supplierAutomationRedisLock struct {
	rdb *redis.Client
}

func NewSupplierAutomationLock(rdb *redis.Client) service.SupplierAutomationLock {
	return &supplierAutomationRedisLock{rdb: rdb}
}

func (l *supplierAutomationRedisLock) TryAcquireAutomationLock(ctx context.Context, taskCode, owner string, ttl time.Duration) (bool, error) {
	return l.rdb.SetNX(ctx, supplierAutomationLockKey(taskCode), owner, ttl).Result()
}

func (l *supplierAutomationRedisLock) ReleaseAutomationLock(ctx context.Context, taskCode, owner string) error {
	return supplierAutomationReleaseScript.Run(ctx, l.rdb, []string{supplierAutomationLockKey(taskCode)}, owner).Err()
}

func supplierAutomationLockKey(taskCode string) string {
	return supplierAutomationLockKeyPrefix + strings.TrimSpace(taskCode)
}

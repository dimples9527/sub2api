package service

import (
	"fmt"
	"sync"
	"time"
)

// supplierCostTrendCacheTTL 成本趋势结果缓存有效期。
// 上游成本与本地成本都由定时同步任务写入，短期缓存可明显减少重复聚合开销，又不至于长期陈旧。
const supplierCostTrendCacheTTL = 30 * time.Second

// supplierCostTrendCacheMaxEntries 缓存条数上限，超过后整体重建，避免长时间运行内存持续增长。
const supplierCostTrendCacheMaxEntries = 256

type supplierCostTrendCacheEntry struct {
	result    SupplierProviderCostTrendResult
	expiresAt time.Time
}

var supplierCostTrendCache = struct {
	sync.RWMutex
	entries map[string]supplierCostTrendCacheEntry
}{entries: make(map[string]supplierCostTrendCacheEntry)}

func supplierCostTrendCacheKey(start, endInclusive time.Time, providerID int64) string {
	return fmt.Sprintf("%s|%s|%d", start.Format("2006-01-02"), endInclusive.Format("2006-01-02"), providerID)
}

func getSupplierCostTrendCache(key string) (SupplierProviderCostTrendResult, bool) {
	supplierCostTrendCache.RLock()
	defer supplierCostTrendCache.RUnlock()
	entry, ok := supplierCostTrendCache.entries[key]
	if !ok {
		return SupplierProviderCostTrendResult{}, false
	}
	if time.Now().After(entry.expiresAt) {
		return SupplierProviderCostTrendResult{}, false
	}
	return entry.result, true
}

func setSupplierCostTrendCache(key string, result SupplierProviderCostTrendResult) {
	supplierCostTrendCache.Lock()
	defer supplierCostTrendCache.Unlock()
	if len(supplierCostTrendCache.entries) >= supplierCostTrendCacheMaxEntries {
		supplierCostTrendCache.entries = make(map[string]supplierCostTrendCacheEntry)
	}
	supplierCostTrendCache.entries[key] = supplierCostTrendCacheEntry{
		result:    result,
		expiresAt: time.Now().Add(supplierCostTrendCacheTTL),
	}
}

// invalidateSupplierCostTrendCache 在成本数据写入（回补、定时同步）后清空缓存，
// 保证「重新获取」等操作后立即读到新数据。
func invalidateSupplierCostTrendCache() {
	supplierCostTrendCache.Lock()
	defer supplierCostTrendCache.Unlock()
	supplierCostTrendCache.entries = make(map[string]supplierCostTrendCacheEntry)
}

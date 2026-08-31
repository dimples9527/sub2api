import { readFileSync } from 'node:fs'
import { dirname, resolve } from 'node:path'
import { fileURLToPath } from 'node:url'
import { describe, expect, it } from 'vitest'

const currentDirectory = dirname(fileURLToPath(import.meta.url))
const source = readFileSync(resolve(currentDirectory, 'SupplierAccountHealthView.vue'), 'utf8')

describe('SupplierAccountHealthView 账号健康趋势批量加载', () => {
  it('列表页使用批量趋势接口，而不是逐账号请求', () => {
    expect(source).toContain('getSupplierAccountHealthTrends')
    expect(source).toContain('await getSupplierAccountHealthTrends(ids, range)')
    expect(source).toContain('void loadAccountTrends(accounts.value, selectedRange.value)')
    expect(source).not.toContain('TREND_LOAD_CONCURRENCY')
    expect(source).not.toContain('loadNextTrend')
  })

  it('选中账号的详情趋势单独缓存完整数据', () => {
    // 守护点位、最近一次检测、上游序列与绑定信息按 accountId:range 整份缓存，避免多张并行 map
    expect(source).toContain('const detailTrendCache = ref<Record<string, SupplierAccountHealthTrend>>({})')
    expect(source).toContain('const cacheKey = `${accountId}:${range}`')
    expect(source).toContain('applyDetailTrend(detailTrendCache.value[cacheKey])')
    expect(source).toContain('await loadTrendRequest(accountId, range)')
  })
})

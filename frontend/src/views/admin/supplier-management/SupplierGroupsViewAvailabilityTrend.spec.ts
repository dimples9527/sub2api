import { readFileSync } from 'node:fs'
import { dirname, resolve } from 'node:path'
import { fileURLToPath } from 'node:url'
import { describe, expect, it } from 'vitest'

const currentDirectory = dirname(fileURLToPath(import.meta.url))
const source = readFileSync(resolve(currentDirectory, 'SupplierGroupsView.vue'), 'utf8')

describe('SupplierGroupsView 可用率趋势', () => {
  it('在上游状态后使用供应商专属趋势列', () => {
    expect(source).toContain("{ key: 'raw_status', label: '上游状态'")
    expect(source).toContain("{ key: 'monitor_trend', label: '可用率趋势'")
    expect(source.indexOf("{ key: 'raw_status', label: '上游状态'"))
      .toBeLessThan(source.indexOf("{ key: 'monitor_trend', label: '可用率趋势'"))
    expect(source).toContain('<SupplierGroupAvailabilityTrend')
    expect(source).toContain(':row="availabilityTrendFor(group)"')
    expect(source).toContain(':loading="availabilityTrendLoading(group)"')
    expect(source).toContain(':source-label="availabilityTrendSourceLabel(group)"')
  })

  it('独立加载监控数据，且不导入上游趋势代码', () => {
    expect(source).toContain("from '@/components/admin/supplier-management/SupplierGroupAvailabilityTrend.vue'")
    expect(source).toContain("from '@/utils/supplierGroupMonitorTrend'")
    expect(source).toContain('listSupplierGroupHealthTrends')
    expect(source).toContain("from '@/utils/supplierGroupHealthTrend'")
    expect(source).toContain("getLocalMonitorStatus({ period: '90m', board: 'hot' })")
    expect(source).not.toContain("adminAPI.groups.getLocalMonitorStatus({ period: '90m', board: 'hot' })")
    expect(source).toContain('void loadMonitorTrend(')
    expect(source).toContain('void loadHealthTrend(result.items.map(group => group.id))')
    expect(source).toContain('return unifiedMonitorTrendFor(group) ?? healthTrendIndex.value.get(group.id)')
    expect(source).not.toContain("from '@/components/admin/upstream/UpstreamGroupAvailabilityTrend.vue'")
    expect(source).not.toContain("from '@/utils/upstreamMonitorTrend'")
  })
})

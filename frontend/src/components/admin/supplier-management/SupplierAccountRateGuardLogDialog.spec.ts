import { readFileSync } from 'node:fs'
import { resolve } from 'node:path'
import { describe, expect, it } from 'vitest'

const source = readFileSync(
  resolve(process.cwd(), 'src/components/admin/supplier-management/SupplierAccountRateGuardLogDialog.vue'),
  'utf8'
)

describe('SupplierAccountRateGuardLogDialog', () => {
  it('默认只显示已解绑记录并支持切换全部记录', () => {
    expect(source).toContain("result: 'unbound'")
    expect(source).toContain("showingAllRecords ? '仅看已解绑' : '显示所有记录'")
    expect(source).toContain('async function toggleAllRecords()')
    expect(source).toContain('filters.result = showingAllRecords.value ? \'unbound\' : \'\'')
    expect(source).toContain("filters.result = 'unbound'")
  })

  it('提供处理状态和确认处理能力', () => {
    expect(source).toContain("{ value: 'pending', label: '待处理' }")
    expect(source).toContain('markAccountRateGuardUnbindLogHandled(log.id)')
    expect(source).toContain("log.status === 'pending'")
  })

  it('使用更宽弹窗并在桌面端隐藏表格横向滚动', () => {
    expect(source).toContain('width="full"')
    expect(source).toContain(':global(.modal-content:has(.account-rate-log-dialog))')
    expect(source).toContain('width: min(1800px, calc(100vw - 32px))')
    expect(source).toContain('overflow-x: hidden')
    expect(source).toContain('table-layout: fixed')
    expect(source).toContain('white-space: normal')
    expect(source).toContain(':sticky-first-column="false"')
  })

  it('直接展示筛选和列表，不重复弹窗标题', () => {
    expect(source).toContain('title="账号倍率守护解除绑定日志"')
    expect(source).not.toContain('Account Rate Guard Audit')
    expect(source).not.toContain('账号与分组解绑轨迹')
  })

  it('使用平台色区分状态、实体和倍率信息', () => {
    expect(source).toContain('account-rate-log-status-pending')
    expect(source).toContain('account-rate-log-status-handled')
    expect(source).toContain('account-rate-log-provider')
    expect(source).toContain('account-rate-log-account')
    expect(source).toContain('account-rate-log-group')
    expect(source).toContain('account-rate-log-scheduling')
    expect(source).toContain('platformBadgeLightClass(log.platform)')
    expect(source).toContain('platformBorderClass(log.platform)')
    expect(source).toContain("from '@/utils/platformColors'")
    expect(source).toContain('account-rate-log-rate-upstream')
    expect(source).toContain('account-rate-log-rate-local')
    expect(source).toContain("`account-rate-log-result-${log.result}`")
    expect(source).toContain('color-mix(in srgb, var(--sp-green)')
    expect(source).toContain('color-mix(in srgb, var(--sp-blue)')
    expect(source).toContain('color-mix(in srgb, var(--sp-violet)')
    expect(source).toContain(':global(.dark) .account-rate-log-dialog')
  })

  it('将上游和本地分组倍率横向展示', () => {
    expect(source).toContain('class="account-rate-log-rate-compare"')
    expect(source).toContain('grid-template-columns: minmax(0, 1fr) auto minmax(0, 1fr)')
    expect(source).toContain('class="account-rate-log-rate-divider"')
  })

  it('为日志表格提供斑马纹和悬停高亮', () => {
    expect(source).toContain('tbody tr:nth-child(even)')
    expect(source).toContain('tbody tr:hover')
  })
})

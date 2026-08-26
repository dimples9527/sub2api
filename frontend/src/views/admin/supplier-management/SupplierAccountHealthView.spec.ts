import { readFileSync } from 'node:fs'
import { resolve } from 'node:path'
import { describe, expect, it } from 'vitest'

const source = readFileSync(
  resolve(process.cwd(), 'src/views/admin/supplier-management/SupplierAccountHealthView.vue'),
  'utf8'
)

const apiSource = readFileSync(
  resolve(process.cwd(), 'src/api/admin/supplierAccountHealth.ts'),
  'utf8'
)

const routerSource = readFileSync(resolve(process.cwd(), 'src/router/index.ts'), 'utf8')

describe('SupplierAccountHealthView', () => {
  it('provides an independent account health trend route and API', () => {
    expect(routerSource).toContain("name: 'SupplierAccountHealth'")
    expect(routerSource).toContain("path: '/admin/supplier-management/account-health'")
    expect(apiSource).toContain("'/admin/supplier-management/account-health/accounts'")
    expect(apiSource).toContain("'/admin/supplier-management/account-health/trend'")
  })

  it('renders account filters, health status, latency charts, and range switching', () => {
    expect(source).toContain('<SupplierModuleLayout>')
    expect(source).toContain('供应商筛选')
    expect(source).toContain('当前健康状态')
    expect(source).toContain('账号名称或 ID')
    expect(source).toContain("'24h'")
    expect(source).toContain("'7d'")
    expect(source).toContain("'30d'")
    expect(source).toContain('健康状态趋势')
    expect(source).toContain('响应时间趋势')
    expect(source).toContain('<Line')
  })

  it('selects the account from account_id and avoids rendering failed latency as zero', () => {
    expect(source).toContain("route.query.account_id")
    expect(source).toContain('selectedAccountId')
    expect(source).toContain('latency_ms === null')
    expect(source).toContain('尚无健康检测记录')
  })

  it('exposes diagnostic details and uses the global error toast', () => {
    expect(source).toContain('失败原因')
    expect(source).toContain('动作')
    expect(source).toContain('错误详情')
    expect(source).toContain('useAppStore')
    expect(source).toContain('showError')
    expect(source).toContain('extractApiErrorMessage')
    expect(source).not.toContain('sp-alert')
  })
})

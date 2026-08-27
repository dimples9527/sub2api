import { readFileSync } from 'node:fs'
import { resolve } from 'node:path'
import { describe, expect, it } from 'vitest'

const source = readFileSync(
  resolve(process.cwd(), 'src/views/admin/supplier-management/SupplierAccountsView.vue'),
  'utf8'
)

describe('SupplierAccountsView 测试时间展示', () => {
  it('在原上次测试时间列内上下展示手动测试和守护检测时间', () => {
    expect(source).toContain('#cell-local_account_last_tested_at')
    expect(source).toContain('local_account_last_tested_at')
    expect(source).toContain('local_account_health_guard_last_checked_at')
    expect(source).not.toContain("{ key: 'local_account_health_guard_last_checked_at'")
    expect(source).toContain('上次测试')
    expect(source).toContain('守护检测')
    expect(source).toContain('sp-account-test-times')
  })
})

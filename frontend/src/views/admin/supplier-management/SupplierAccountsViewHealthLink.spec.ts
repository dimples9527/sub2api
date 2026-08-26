import { readFileSync } from 'node:fs'
import { resolve } from 'node:path'
import { describe, expect, it } from 'vitest'

const source = readFileSync(
  resolve(process.cwd(), 'src/views/admin/supplier-management/SupplierAccountsView.vue'),
  'utf8'
)

describe('SupplierAccountsView 健康趋势入口', () => {
  it('仅为已匹配本地账号提供健康趋势入口', () => {
    expect(source).toContain('supplier-account-health-')
    expect(source).toContain("account.local_account_match_status === 'matched'")
    expect(source).toContain("name: 'SupplierAccountHealth'")
    expect(source).toContain('查看健康趋势')
  })
})

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

describe('SupplierAccountsView 配置健康守护弹窗', () => {
  it('每次打开都重新拉取候选账号，不复用上次缓存', () => {
    const accountsLoader = source.match(/async function ensureDialogAccountsLoaded\(\) \{([\s\S]*?)\n\}/)?.[1] || ''
    expect(accountsLoader).not.toContain('dialogEligibleAccounts.value.length > 0')
    expect(accountsLoader).toContain('dialogEligibleAccounts.value = []')
    expect(accountsLoader).toContain('dialogEligibleAccounts.value = await fetchEligibleSupplierAccounts()')
    expect(accountsLoader).toContain('dialogDisabledProviderIds.value = await providersPromise')
  })

  it('分组候选同样每次打开重新拉取', () => {
    const groupsLoader = source.match(/async function ensureDialogGroupsLoaded\(\) \{([\s\S]*?)\n\}/)?.[1] || ''
    expect(groupsLoader).not.toContain('dialogAllGroups.value.length > 0')
    expect(groupsLoader).toContain('dialogAllGroups.value = await listAllGroupsIncludingInactive()')
  })
})

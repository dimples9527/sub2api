import { readFileSync } from 'node:fs'
import { dirname, resolve } from 'node:path'
import { fileURLToPath } from 'node:url'
import { describe, expect, it } from 'vitest'

const currentDirectory = dirname(fileURLToPath(import.meta.url))
const source = readFileSync(resolve(currentDirectory, 'SupplierAccountsView.vue'), 'utf8')
const apiSource = readFileSync(
  resolve(currentDirectory, '../../../api/admin/supplierProviderData.ts'),
  'utf8'
)

describe('SupplierAccountsView 失效上游分组记录删除', () => {
  it('为账号列表返回失效分组记录 ID 和删除资格', () => {
    expect(apiSource).toContain('group_record_id?: number')
    expect(apiSource).toContain('group_record_delete_eligible: boolean')
  })

  it('为上游账号返回删除字段并提供删除接口', () => {
    expect(apiSource).toContain('account_record_delete_eligible: boolean')
    expect(apiSource).toContain('deleteSupplierAccount,')
    expect(source).toContain('deleteSupplierAccount(target.id)')
  })

  it('不在上游账号页面提供分组记录删除入口', () => {
    expect(source).not.toContain('deleteSupplierGroup,')
    expect(source).not.toContain('删除失效分组记录')
    expect(source).not.toContain('deleteSupplierGroup(target.group_record_id)')
  })

  it('为已删除上游账号和上游分组已删除的账号显示清理入口', () => {
    expect(source).toContain("account.status === 'deleted'")
    expect(source).toContain("account.group_status === 'missing'")
    expect(source).toContain('删除上游账号记录')
  })

  it('为每条上游账号记录显示删除按钮，不按删除资格字段隐藏', () => {
    expect(source).not.toContain('<template v-if="account.account_record_delete_eligible">')
    expect(source).not.toContain('!account.account_record_delete_eligible')
  })

  it('删除前二次确认，成功后刷新账号列表并清理当前详情', () => {
    expect(source).not.toContain('title="删除失效上游分组记录"')
    expect(source).not.toContain('仅删除本地保存的失效上游分组记录')
    expect(source).not.toContain('删除时会解除绑定，但不会删除本地分组')
    expect(source).toContain('await loadAccounts()')
    expect(source).toContain('selected.value = null')
  })
})


describe('供应商关闭账号整行灰化', () => {
  it('供应商关闭时为供应商单元格添加行状态标记', () => {
    expect(source).toContain("'provider-disabled': isProviderDisabled(account)")
    expect(source).toContain('function isProviderDisabled(account: SupplierProviderAccount)')
  })

  it('通过整行选择器将供应商关闭账号显示为灰色', () => {
    expect(source).toContain('tbody tr:has(.sp-provider-cell.provider-disabled)')
    expect(source).toContain('filter: grayscale(1)')
  })
})


describe('供应商状态筛选', () => {
  it('默认展示已启用供应商，并保留全部和已关闭供应商状态选项', () => {
    expect(source).toContain("const providerStatusFilter = ref('enabled')")
    expect(source).toContain("{ value: 'disabled', label: '已关闭供应商' }")
    expect(source).toContain('providerStatusFilterOptions')
  })

  it('按供应商状态过滤账号并参与分页', () => {
    expect(source).toContain('accountMatchesProviderStatusFilter(account, providerStatusFilter.value)')
    expect(source).toContain('watch([providerID, providerStatusFilter, groupID')
    expect(source).toContain('applyAccountQuickFilterPage()')
  })
})

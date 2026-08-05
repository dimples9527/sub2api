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

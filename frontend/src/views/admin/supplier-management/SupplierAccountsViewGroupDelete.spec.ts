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

  it('在上游账号操作栏为失效分组显示删除入口并调用删除接口', () => {
    expect(apiSource).toContain('deleteSupplierGroup,')
    expect(source).toContain('v-if="account.group_status === \'inactive\'"')
    expect(source).toContain('删除失效分组记录')
    expect(source).toContain('deleteSupplierGroup(target.group_record_id)')
  })

  it('删除前二次确认，成功后刷新账号列表并清理当前详情', () => {
    expect(source).toContain('title="删除失效上游分组记录"')
    expect(source).toContain('仅删除本地保存的失效上游分组记录')
    expect(source).toContain('await loadAccounts()')
    expect(source).toContain('selected.value = null')
  })
})

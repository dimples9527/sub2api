import { readFileSync } from 'node:fs'
import { dirname, resolve } from 'node:path'
import { fileURLToPath } from 'node:url'
import { describe, expect, it } from 'vitest'

const currentDirectory = dirname(fileURLToPath(import.meta.url))
const source = readFileSync(resolve(currentDirectory, 'SupplierGroupsView.vue'), 'utf8')
const apiSource = readFileSync(
  resolve(currentDirectory, '../../../api/admin/supplierProviderData.ts'),
  'utf8'
)

describe('SupplierGroupsView 失效分组记录删除', () => {
  it('通过 DELETE API 删除指定分组记录', () => {
    expect(apiSource).toContain('export async function deleteSupplierGroup(')
    expect(apiSource).toContain('apiClient.delete<{ group_id: number }>')
    expect(apiSource).toContain('/admin/supplier-management/groups/${id}')
    expect(apiSource).toContain('deleteSupplierGroup,')
  })

  it('仅为失效分组显示删除操作并使用二次确认', () => {
    expect(source).toContain('v-if="!group.active"')
    expect(source).toContain('删除记录')
    expect(source).toContain('deleteTarget')
    expect(source).toContain('title="删除失效分组记录"')
    expect(source).toContain('仅删除本地保存的失效上游分组记录')
    expect(source).toContain('await deleteSupplierGroup(target.id)')
  })

  it('加载分组时不再固定过滤有效记录，删除成功后刷新列表', () => {
    expect(source).not.toContain('active: true')
    expect(source).toContain('await loadGroups()')
    expect(source).toContain("appStore.showSuccess('失效分组记录已删除')")
  })
})

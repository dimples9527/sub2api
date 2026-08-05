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

describe('SupplierGroupsView 分组记录删除', () => {
  it('通过 DELETE API 删除指定分组记录', () => {
    expect(apiSource).toContain('export async function deleteSupplierGroup(')
    expect(apiSource).toContain('apiClient.delete<{ group_id: number }>')
    expect(apiSource).toContain('/admin/supplier-management/groups/${id}')
    expect(apiSource).toContain('deleteSupplierGroup,')
  })

  it('为每条分组记录显示删除操作并使用二次确认', () => {
    expect(source).toContain('删除分组记录')
    expect(source).not.toContain('v-if="!group.active"')
    expect(source).not.toContain('canDeleteSupplierGroup(group)')
    expect(source).toContain('删除记录')
    expect(source).toContain('deleteTarget')
    expect(source).toContain('title="删除分组记录"')
    expect(source).toContain('仅删除本地保存的上游分组记录')
    expect(source).toContain('await deleteSupplierGroup(target.id)')
  })

  it('显示上游分组状态和失效时间，删除成功后刷新列表', () => {
    expect(source).toContain('group.active')
    expect(source).toContain('group.inactive_at')
    expect(source).toContain("if (!group.active) return '已失效'")
    expect(source).toContain("return '正常'")
    expect(source).not.toContain('active: true')
    expect(source).toContain('await loadGroups()')
    expect(source).toContain("appStore.showSuccess('分组记录已删除')")
  })
})

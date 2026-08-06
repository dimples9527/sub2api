import { readFileSync } from 'node:fs'
import { resolve } from 'node:path'
import { describe, expect, it } from 'vitest'

const source = readFileSync(
  resolve(process.cwd(), 'src/views/admin/supplier-management/SupplierGroupsView.vue'),
  'utf8',
)

describe('供应商分组密钥状态展示', () => {
  it('应包含密钥状态列和三种状态文案', () => {
    expect(source).toContain("key: 'key_status'")
    expect(source).toContain('密钥状态')
    expect(source).toContain("group.key_status === 'created'")
    expect(source).toContain("group.key_status === 'not_created'")
    expect(source).toContain('已创建')
    expect(source).toContain('未创建')
    expect(source).toContain('无法确认')
  })

  it('已创建状态应展示当前分组匹配的密钥数量', () => {
    expect(source).toContain('group.account_count')
    expect(source).toContain('个密钥')
  })

  it('本地已有密钥但最近同步失败时应保留已创建状态并提示同步失败', () => {
    expect(source).toContain('`${group.account_count} 个密钥 · 最近同步失败`')
  })
  it('已创建密钥快捷入口只筛选当前分组', () => {
    expect(source).toContain("keyStatusFilter.value = 'created'")
    expect(source).toContain('active: groupScope.value === \'all\' ? undefined : groupScope.value === \'active\'')
    expect(source).toContain('key_status: keyStatusFilter.value || undefined')
    expect(source).toContain('已创建密钥')
  })
})

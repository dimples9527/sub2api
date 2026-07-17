import { readFileSync } from 'node:fs'
import { dirname, resolve } from 'node:path'
import { fileURLToPath } from 'node:url'
import { describe, expect, it } from 'vitest'

const currentDir = dirname(fileURLToPath(import.meta.url))

const groupsSource = readFileSync(resolve(currentDir, 'SupplierGroupsView.vue'), 'utf-8')
const accountsSource = readFileSync(resolve(currentDir, 'SupplierAccountsView.vue'), 'utf-8')

describe('supplier local data views component usage', () => {
  it.each([
    ['SupplierGroupsView', groupsSource],
    ['SupplierAccountsView', accountsSource],
  ])('%s uses existing framework controls for filters, tables, and pagination', (_name, source) => {
    expect(source).toContain("import DataTable from '@/components/common/DataTable.vue'")
    expect(source).toContain("import Input from '@/components/common/Input.vue'")
    expect(source).toContain("import Pagination from '@/components/common/Pagination.vue'")
    expect(source).toContain("import Select, { type SelectOption } from '@/components/common/Select.vue'")
    expect(source).toContain('<DataTable')
    expect(source).toContain('<Input')
    expect(source).toContain('<Pagination')
    expect(source).toContain('<Select')
    expect(source).not.toContain('<table')
    expect(source).not.toContain('<select')
    expect(source).not.toContain('<input')
  })

  it('makes SupplierGroupsView easier to scan and reset', () => {
    expect(groupsSource).toContain('resetGroupFilters')
    expect(groupsSource).toContain('canResetFilters')
    expect(groupsSource).toContain('handleGroupPageSizeChange')
    expect(groupsSource).toContain('pageSize = ref(20)')
    expect(groupsSource).toContain('@update:pageSize="handleGroupPageSizeChange"')
    expect(groupsSource).toContain(':show-page-size-selector="true"')
    expect(groupsSource).toContain('有效分组')
    expect(groupsSource).toContain('已失效')
    expect(groupsSource).toContain('空分组')
    expect(groupsSource).toContain('未命名')
    expect(groupsSource).toContain('activeGroupCount')
    expect(groupsSource).toContain('emptyGroupCount')
    expect(groupsSource).toContain("{ key: 'active', label: '本地状态'")
    expect(groupsSource).toContain("{ key: 'account_count', label: '账号数'")
    expect(groupsSource.indexOf("{ key: 'active', label: '本地状态'")).toBeLessThan(
      groupsSource.indexOf("{ key: 'name', label: '上游分组'")
    )
  })
})

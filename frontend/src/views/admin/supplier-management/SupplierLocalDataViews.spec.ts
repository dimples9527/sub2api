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

  it('uses full-filter summary cards and keeps group controls easy to scan', () => {
    expect(groupsSource).toContain('resetGroupFilters')
    expect(groupsSource).toContain('canResetFilters')
    expect(groupsSource).toContain('handleGroupPageSizeChange')
    expect(groupsSource).toContain('pageSize = ref(20)')
    expect(groupsSource).toContain('@update:pageSize="handleGroupPageSizeChange"')
    expect(groupsSource).toContain(':show-page-size-selector="true"')
    expect(groupsSource).toContain("import StatCard from '@/components/common/StatCard.vue'")
    expect(groupsSource).toContain('<StatCard')
    expect(groupsSource).toContain('筛选结果')
    expect(groupsSource).toContain('关联账号')
    expect(groupsSource).toContain('已关联分组')
    expect(groupsSource).toContain('未关联分组')
    expect(groupsSource).toContain('groupSummary')
    expect(groupsSource).toContain('result.summary')
    expect(groupsSource).toContain('linkedGroupRate')
    expect(groupsSource).toContain('unlinkedGroupRate')
    expect(groupsSource).toContain('当前页摘要')
    expect(groupsSource).toContain('本页 {{ items.length }} 条记录')
    expect(groupsSource).not.toContain('sp-stat-box"><span>已关联账号')
    expect(groupsSource).not.toContain('sp-stat-box"><span>未关联账号')
    expect(groupsSource).not.toContain('var(--sp-bg)')
    expect(groupsSource).not.toContain('<article')
    expect(groupsSource).not.toContain('signalCards')
    expect(groupsSource).toContain("{ key: 'active', label: '记录状态'")
    expect(groupsSource).toContain("{ key: 'account_count', label: '关联账号数'")
    expect(groupsSource.indexOf("{ key: 'active', label: '记录状态'")).toBeLessThan(
      groupsSource.indexOf("{ key: 'name', label: '上游分组'")
    )
    expect(groupsSource).toContain('sp-console-shell')
    expect(groupsSource).toContain('sp-summary-grid')
    expect(groupsSource).toContain('sp-console-panel')
    expect(groupsSource).toContain('sp-table-shell')
  })
})

import { readFileSync } from 'node:fs'
import { dirname, resolve } from 'node:path'
import { fileURLToPath } from 'node:url'
import { describe, expect, it } from 'vitest'

const source = readFileSync(resolve(dirname(fileURLToPath(import.meta.url)), 'SupplierAccountsView.vue'), 'utf8')

describe('供应商上游账号工具栏与排序', () => {
  it('手机端筛选默认收起，按钮能切换展开且不清空现有筛选值，桌面不隐藏', () => {
    expect(source).toContain('const mobileFiltersExpanded = ref(false)')
    expect(source).toContain('data-test="supplier-account-mobile-filter-toggle"')
    expect(source).toContain(':aria-expanded="mobileFiltersExpanded"')
    expect(source).toContain('aria-controls="supplier-account-filter-fields"')
    expect(source).toContain('id="supplier-account-filter-fields"')
    expect(source).toContain('mobileFiltersExpanded = !mobileFiltersExpanded')
    expect(source).toMatch(/@media \(max-width: 767px\) \{[\s\S]*?\.sp-account-filter-fields:not\(\.is-expanded\)\s*\{\s*display: none;/)
  })

  it('四个指定操作在电脑和手机共用更多菜单，保留禁用条件与处理器', () => {
    expect(source).toContain('data-test="supplier-account-toolbar-more"')
    expect(source).toContain(':aria-expanded="toolbarMoreOpen"')
    const menu = source.match(/<div[^>]*class="sp-account-toolbar-more-group"[\s\S]*?<\/div>\s*<\/div>/)?.[0] || ''
    for (const name of ['supplier-account-create', 'supplier-account-sync-upstream-models', 'supplier-account-batch-bind-groups', 'supplier-account-bind-by-group']) {
      expect(menu).toContain(`data-test="${name}"`)
      expect(source.split(`data-test="${name}"`)).toHaveLength(2)
    }
    expect(menu).toContain('openCreateAccountDialog')
    expect(menu).toContain('openSyncModelsDialog')
    expect(menu).toContain('openBatchBindGroupsDialog')
    expect(menu).toContain('openBindByGroupDialog')
    expect(menu).toContain('syncModelsTargets.length === 0 || syncModelsSubmitting')
    expect(menu).toContain('selectedBindableAccounts.length === 0 || batchBindSubmitting')
    expect(source).toContain('closeToolbarMore')
    expect(source).toContain('return action()')
    expect(source).toContain('Escape')
  })

  it('电脑工具栏也显示菜单，展开面板不会被吸顶容器裁切', () => {
    expect(source).toContain('ref="toolbarMoreGroup" class="sp-account-toolbar-more-group"')
    expect(source).not.toContain('v-if="isMobileAccountViewport" ref="toolbarMoreGroup"')
    expect(source).toMatch(/\.sp-account-toolbar \{[\s\S]*?overflow: visible;/)
    expect(source).toMatch(/\.sp-account-toolbar-more-group \{[\s\S]*?display: block;/)
  })
  it('卡片列表有独立排序字段与方向，使用现有服务端排序且同步桌面表头', () => {
    expect(source).toContain('data-test="supplier-account-mobile-sort"')
    expect(source).toContain('data-test="supplier-account-mobile-sort-order"')
    expect(source).toContain('accountColumns.filter(column => column.sortable)')
    expect(source).toContain('handleAccountSort(key, sortOrder.value)')
    expect(source).toContain("handleAccountSort(sortBy, sortOrder === 'asc' ? 'desc' : 'asc')")
    expect(source).toContain(':default-sort-key="sortBy"')
    expect(source).toContain(':default-sort-order="sortOrder"')
    expect(source).toContain(':key="accountTableSortKey"')
    expect(source).toMatch(/@media \(min-width: 768px\) \{[\s\S]*?\.sp-account-mobile-sort\s*\{\s*display: none;/)
  })
})

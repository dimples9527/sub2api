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

  it('uses a compact account workbench with standard pagination', () => {
    expect(accountsSource).not.toContain('Local Supplier Accounts')
    expect(accountsSource).not.toContain('<h1>上游账号</h1>')
    expect(accountsSource).not.toContain('只展示已同步到本地数据库的供应商账号')
    expect(accountsSource).toContain('sp-account-toolbar')
    expect(accountsSource).toContain('sp-account-table-shell')
    expect(accountsSource).toContain('sp-account-pagination')
    expect(accountsSource).toContain('pageSize = ref(20)')
    expect(accountsSource).toContain(':options="pageSizeOptions"')
    expect(accountsSource).toContain('v-model="platformFilter"')
    expect(accountsSource).toContain(':options="platformFilterOptions"')
    expect(accountsSource).toContain('platform: platformFilter.value || undefined')
    expect(accountsSource).toContain("import { platformBadgeClass, platformLabel } from '@/utils/platformColors'")
    expect(accountsSource).toContain("{ key: 'platform', label:")
    expect(accountsSource).toContain('platformBadgeClass(account.platform)')
    expect(accountsSource).toContain('platformLabel(account.platform)')
    expect(accountsSource).toContain('handlePageSizeChange')
    expect(accountsSource).toContain(':show-page-size-selector="false"')
    expect(accountsSource).toContain('@update:page="handlePageChange"')
    expect(accountsSource).not.toContain('查询说明')
    expect(accountsSource).not.toContain('sp-grid-2')
  })
  it('uses full-filter summary cards and keeps group controls easy to scan', () => {
		expect(groupsSource).not.toContain('Supplier Group Matching')
		expect(groupsSource).not.toContain('<h1>分组管理</h1>')
		expect(groupsSource).not.toContain('对照最近一次采集到的上游分组与本地分组')
		expect(groupsSource).toContain('sp-filter-toolbar')
		expect(groupsSource).toContain('sp-filter-fields')
		expect(groupsSource).toContain('sp-filter-actions')
		expect(groupsSource).toContain('v-model="platformFilter"')
		expect(groupsSource).toContain('v-model="matchStatusFilter"')
		expect(groupsSource).toContain('v-model="rateStatusFilter"')
		expect(groupsSource).toContain(':searchable="true"')
    expect(groupsSource).toContain('resetGroupFilters')
    expect(groupsSource).toContain('canResetFilters')
    expect(groupsSource).toContain('handleGroupPageSizeChange')
    expect(groupsSource).toContain('pageSize = ref(20)')
    expect(groupsSource).toContain('@update:pageSize="handleGroupPageSizeChange"')
    expect(groupsSource).toContain(':show-page-size-selector="true"')
    expect(groupsSource).toContain("import StatCard from '@/components/common/StatCard.vue'")
    expect(groupsSource).toContain('<StatCard')
    expect(groupsSource).toContain('上游分组')
    expect(groupsSource).toContain('已匹配')
    expect(groupsSource).toContain('待匹配')
    expect(groupsSource).toContain('倒挂风险')
    expect(groupsSource).toContain('groupSummary')
    expect(groupsSource).toContain('result.summary')
    expect(groupsSource).toContain('matchedGroupRate')
    expect(groupsSource).toContain('unmatchedGroupRate')
    expect(groupsSource).not.toContain('var(--sp-bg)')
    expect(groupsSource).not.toContain('<article')
    expect(groupsSource).not.toContain('signalCards')
    const columns = [
      "{ key: 'provider_name', label: '供应商'",
      "{ key: 'name', label: '上游分组'",
      "{ key: 'rate_multiplier', label: '上游倍率'",
      "{ key: 'raw_status', label: '上游状态'",
      "{ key: 'local_group_name', label: '匹配本地分组'",
      "{ key: 'local_rate_multiplier', label: '本地分组倍率'",
      "{ key: 'rate_delta', label: '价差'",
      "{ key: 'account_count', label: '绑定账号'",
      "{ key: 'rate_status', label: '倍率状态'",
      "{ key: 'actions', label: '操作'",
    ]
    columns.forEach(column => expect(groupsSource).toContain(column))
    expect(groupsSource).toContain('修改后价差')
    expect(groupsSource).not.toContain('收益倍率')
    expect(groupsSource).not.toContain('formatProfitRate')
    columns.slice(1).forEach((column, index) => {
      expect(groupsSource.indexOf(columns[index])).toBeLessThan(groupsSource.indexOf(column))
    })
    expect(groupsSource).toContain('sp-console-shell')
    expect(groupsSource).toContain('sp-summary-grid')
		expect(groupsSource).toContain('sp-summary-filter')
		expect(groupsSource).toContain('applySummaryFilter')
		expect(groupsSource).toContain(':aria-pressed="isSummaryFilterActive')
    expect(groupsSource).toContain('sp-console-panel')
    expect(groupsSource).toContain('sp-table-shell')
    expect(groupsSource).toContain('height: min(64vh, 680px)')
    const mobileMediaIndex = groupsSource.indexOf('@media (max-width: 760px)')
    expect(groupsSource.indexOf('.sp-table-shell { height: auto;', mobileMediaIndex)).toBeGreaterThan(mobileMediaIndex)
		const reducedMotionIndex = groupsSource.indexOf('@media (prefers-reduced-motion: reduce)')
		const mobileStyles = groupsSource.slice(mobileMediaIndex, reducedMotionIndex)
		expect(mobileStyles).toContain('.sp-summary-grid { grid-template-columns: repeat(4, minmax(0, 1fr)); }')
		expect(mobileStyles).not.toContain('.sp-summary-grid { grid-template-columns: 1fr; }')
  })

  it('shows only groups available in the latest collection', () => {
    expect(groupsSource).not.toContain('activeFilter')
    expect(groupsSource).not.toContain("{ key: 'active'")
    expect(groupsSource).not.toContain('selected.active')
    expect(groupsSource).not.toContain('inactiveGroupCount')
    expect(groupsSource).not.toContain('失效记录')
    expect(groupsSource).toContain('active: true')
  })

  it('uses existing dialogs and APIs for local-group operations', () => {
    expect(groupsSource).toContain("import BaseDialog from '@/components/common/BaseDialog.vue'")
    expect(groupsSource).toContain("import ConfirmDialog from '@/components/common/ConfirmDialog.vue'")
    expect(groupsSource).toContain('updateSupplierGroupMapping')
		expect(groupsSource).toContain('updateSupplierGroupRateGuard')
    expect(groupsSource).toContain('adminAPI.groups.create')
    expect(groupsSource).toContain('adminAPI.groups.update')
    expect(groupsSource).toContain('匹配分组')
    expect(groupsSource).toContain('新建分组')
    expect(groupsSource).toContain('调倍率')
    expect(groupsSource).toContain('更换本地分组')
    expect(groupsSource).toContain('取消关联')
		expect(groupsSource).toContain("{ key: 'rate_guard_status', label: '倍率守护'")
		expect(groupsSource).toContain('toggleRateGuard')
		expect(groupsSource).toContain('设为守护')
  })

  it('uses distinct supplier tones and the existing group platform palette', () => {
    expect(groupsSource).toContain("import GroupBadge from '@/components/common/GroupBadge.vue'")
    expect(groupsSource).toContain('<GroupBadge')
    expect(groupsSource).toContain('SUPPLIER_TONES')
    expect(groupsSource).toContain('supplierTone(group.provider_id)')
    expect(groupsSource).toContain('sp-supplier-chip')
    expect(groupsSource).toContain('groupPlatform(group.local_group_platform)')
    expect(groupsSource).toContain('UPSTREAM_GROUP_TONES')
    expect(groupsSource).toContain('upstreamGroupTone(group.upstream_group_key)')
    expect(groupsSource).toContain('sp-upstream-group-chip')
    expect(groupsSource).toContain(`<span :class="['sp-rate-value', upstreamRateTone(group.rate_multiplier)]">`)
    expect(groupsSource).toContain('UPSTREAM_RATE_TONES')
    expect(groupsSource).not.toContain('.sp-rate-value.upstream')
    expect(groupsSource).toContain('{{ supplierTypeLabel(group.provider_id) }}</span>')
    expect(groupsSource).toContain('<span>#{{ group.provider_id }}</span>')
    expect(groupsSource).not.toContain('【{{ supplierTypeLabel(group.provider_id) }}】')
    expect(groupsSource).toContain('SUPPLIER_TYPE_TONES')
    expect(groupsSource).toContain('supplierTypeTone(group.provider_id)')
    expect(groupsSource).toContain('sp-provider-type')
    expect(groupsSource).toContain('【{{ upstreamPlatformLabel(group) }}】')
  })

  it('adds supplier quick filters to the group table header', () => {
    expect(groupsSource).toContain('sp-provider-shortcuts')
    expect(groupsSource).toContain('quickProviderOptions')
    expect(groupsSource).toContain('selectProviderShortcut')
    expect(groupsSource).toContain(':aria-pressed="providerID === option.value"')
    expect(groupsSource).toContain('@click="selectProviderShortcut(option.value)"')
  })
})

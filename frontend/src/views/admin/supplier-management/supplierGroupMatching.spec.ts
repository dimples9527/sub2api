import fs from 'node:fs'
import path from 'node:path'
import { describe, expect, it } from 'vitest'

const apiPath = path.resolve(process.cwd(), 'src/api/admin/supplierProviderData.ts')
const viewPath = path.resolve(process.cwd(), 'src/views/admin/supplier-management/SupplierGroupsView.vue')
const apiSource = fs.readFileSync(apiPath, 'utf8')
const viewSource = fs.readFileSync(viewPath, 'utf8')

describe('supplier group automatic matching workflow', () => {
  it('exposes automatic matching state and management APIs', () => {
    expect(apiSource).toContain("auto_match_status: 'unmatched' | 'auto_matched' | 'manual' | 'ambiguous'")
    expect(apiSource).toContain('auto_match_ignored: boolean')
    expect(apiSource).toContain('name_change_pending: boolean')
		expect(apiSource).toContain('rate_guard_selected: boolean')
		expect(apiSource).toContain("rate_guard_selection_mode: '' | 'auto' | 'manual'")
		expect(apiSource).toContain('local_group_active_mapping_count: number')
		expect(apiSource).toContain('local_group_rate_guard_group_id?: number')
		expect(apiSource).toContain('local_group_rate_guard_group_name?: string')
		expect(apiSource).toContain('local_group_rate_guard_provider_name?: string')
		expect(apiSource).toContain("group_sync_status: 'never' | 'running' | 'success' | 'failed'")
		expect(apiSource).toContain('export async function updateSupplierGroupRateGuard')
    expect(apiSource).toContain('export async function autoMatchSupplierGroups')
    expect(apiSource).toContain('export async function updateSupplierGroupAutoMatchPolicy')
    expect(apiSource).toContain('export async function resolveSupplierGroupNameChange')
		expect(apiSource).toContain('platform?: string')
		expect(apiSource).toContain('match_status?: string')
		expect(apiSource).toContain('rate_status?: string')
  })

  it('provides matching, ignore and name-change controls on the group page', () => {
    expect(viewSource).toContain('自动匹配')
    expect(viewSource).toContain("{ key: 'auto_match_status', label: '匹配状态'")
    expect(viewSource).toContain('toggleAutoMatchIgnored')
    expect(viewSource).toContain('保持本地名称')
    expect(viewSource).toContain('同步本地名称')
    expect(viewSource).toContain('resolveNameChange')
    expect(viewSource).toContain("if (group.local_group_id && group.auto_match_status === 'manual') return '人工匹配'")
		expect(viewSource).toContain('自动守护')
		expect(viewSource).toContain('人工守护')
		expect(viewSource).toContain('可设守护')
		expect(viewSource).toContain('上游分组不可用')
		expect(viewSource).toContain('本地分组不可用')
		expect(viewSource).toContain('等待首次同步')
		expect(viewSource).toContain('分组同步中')
		expect(viewSource).toContain('分组同步失败')
		expect(viewSource).not.toContain("label: '守护异常'")
		expect(viewSource).toContain('非守护源')
		expect(viewSource).toContain('未匹配')
		expect(viewSource).toContain('canManageManualRateGuard(group)')
		expect(viewSource).toContain('group.local_group_active_mapping_count > 1')
		expect(viewSource).not.toContain('items.value.some(item =>')
		expect(viewSource).toContain('!group.rate_guard_selected && !rateGuardEligible(group)')
		expect(viewSource).toContain('更换本地分组')
		expect(viewSource).toContain('取消关联')
		expect(viewSource).not.toContain('重新匹配')
		expect(viewSource).not.toContain('解除匹配')
	})

	it('identifies the selected guard source for other mappings of the same local group', () => {
		expect(viewSource).toContain('已由其它分组守护')
		expect(viewSource).toContain('group.local_group_rate_guard_provider_name')
		expect(viewSource).toContain('group.local_group_rate_guard_group_name')
		expect(viewSource).toContain("filter(Boolean).join(' / ')")
		expect(viewSource).toContain('当前本地分组由该上游分组执行倍率守护，本分组不会参与守护')
		expect(viewSource).toContain("hasOtherRateGuard(group) ? '切换为守护' : '设为守护'")
	})

	it('provides server-backed platform, match and rate filters through clickable summary cards', () => {
		expect(viewSource).toContain("{ value: 'openai', label: 'OpenAI' }")
		expect(viewSource).toContain("{ value: 'name_changed', label: '名称变化' }")
		expect(viewSource).toContain("{ value: 'inverted', label: '倒挂风险' }")
		expect(viewSource).toContain('platform: platformFilter.value || undefined')
		expect(viewSource).toContain('match_status: matchStatusFilter.value || undefined')
		expect(viewSource).toContain('rate_status: rateStatusFilter.value || undefined')
		expect(viewSource).toContain("applySummaryFilter('linked')")
		expect(viewSource).toContain("applySummaryFilter('unlinked')")
		expect(viewSource).toContain("applySummaryFilter('inverted')")
	})

	it('sorts the six core group columns through the server', () => {
		expect(apiSource).toContain('sort_by?: string')
		expect(apiSource).toContain("sort_order?: 'asc' | 'desc'")
		expect(viewSource).toContain('server-side-sort')
		expect(viewSource).toContain('@sort="handleGroupSort"')
		expect(viewSource).toContain('sort_by: sortBy.value || undefined')
		expect(viewSource).toContain('sort_order: sortBy.value ? sortOrder.value : undefined')
		expect(viewSource).toContain("function handleGroupSort(key: string, order: 'asc' | 'desc')")

		const columnsStart = viewSource.indexOf('const groupColumns: Column[] = [')
		const columnsEnd = viewSource.indexOf('\n]', columnsStart)
		const columnsSource = viewSource.slice(columnsStart, columnsEnd)
		const sortableColumns = [
			'provider_name',
			'name',
			'rate_multiplier',
			'local_group_name',
			'local_rate_multiplier',
			'account_count',
		]
		for (const key of sortableColumns) {
			expect(columnsSource).toContain(`{ key: '${key}',`)
			expect(columnsSource).toMatch(new RegExp(`key: '${key}'[^\\n]+sortable: true`))
		}
		expect(columnsSource.match(/sortable: true/g)).toHaveLength(sortableColumns.length)
	})

	it('provides attention shortcuts backed by existing group filters', () => {
		expect(viewSource).toContain("{ label: '名称冲突', matchStatus: 'ambiguous', rateStatus: '' }")
		expect(viewSource).toContain("{ label: '名称变化', matchStatus: 'name_changed', rateStatus: '' }")
		expect(viewSource).toContain("{ label: '已忽略', matchStatus: 'ignored', rateStatus: '' }")
		expect(viewSource).toContain("{ label: '收益偏低', matchStatus: '', rateStatus: 'low' }")
		expect(viewSource).toContain("{ label: '倒挂风险', matchStatus: '', rateStatus: 'inverted' }")
		expect(viewSource).toContain('isAttentionShortcutActive(shortcut)')
		expect(viewSource).toContain('applyAttentionShortcut(shortcut)')
		expect(viewSource).toContain('const active = isAttentionShortcutActive(shortcut)')
		expect(viewSource).toContain("matchStatusFilter.value = active ? '' : shortcut.matchStatus")
		expect(viewSource).toContain("rateStatusFilter.value = active ? '' : shortcut.rateStatus")
		expect(viewSource).toContain('suppressFilterWatch = true')
		expect(viewSource).toContain('page.value = 1')
		expect(viewSource).toContain('sp-attention-shortcuts')
	})

	it('labels current-page group signals with their actual scope', () => {
		expect(viewSource).toContain('本页已关联')
		expect(viewSource).toContain('本页需关注')
		expect(viewSource).not.toContain('currentPageMatchedCount }} 已匹配')
		expect(viewSource).not.toContain('currentPageAttentionCount }} 待处理')
	})
})

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
})

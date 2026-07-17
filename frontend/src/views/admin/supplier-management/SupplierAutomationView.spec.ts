import { readFileSync } from 'node:fs'
import { dirname, resolve } from 'node:path'
import { fileURLToPath } from 'node:url'
import { describe, expect, it } from 'vitest'

const supplierAutomationSource = readFileSync(
  resolve(dirname(fileURLToPath(import.meta.url)), 'SupplierAutomationView.vue'),
  'utf-8'
)

describe('SupplierAutomationView edit dialog', () => {
  it('uses a switch control for the enabled state instead of a raw checkbox', () => {
    expect(supplierAutomationSource).toContain("import Toggle from '@/components/common/Toggle.vue'")
    expect(supplierAutomationSource).toContain('<Toggle v-model="editForm.enabled"')
    expect(supplierAutomationSource).not.toContain('class="sp-switch"')
    expect(supplierAutomationSource).not.toContain('v-model="editForm.enabled" type="checkbox"')
  })

  it('renders result messages as previews with a details dialog', () => {
    expect(supplierAutomationSource).toContain('compactMessage(task.last_message ||')
    expect(supplierAutomationSource).toContain('openResultDetail(')
    expect(supplierAutomationSource).toContain('detailVisible')
    expect(supplierAutomationSource).toContain('sp-message-preview')
    expect(supplierAutomationSource).toContain('结果详情')
  })

  it('refreshes data without auto-opening a dialog after executing a task', () => {
    expect(supplierAutomationSource).toContain('const run = await runTask(taskCode)')
    expect(supplierAutomationSource).toContain('showToast(`任务执行完成：${statusText(run.status)}`)')
    expect(supplierAutomationSource).not.toContain('openResultDetail(`${taskCode} 执行结果')
  })

  it('opens structured run details from latest result and run history rows', () => {
    expect(supplierAutomationSource).toContain('latestRunByTask.value[task.task_code]')
    expect(supplierAutomationSource).toContain('@click.stop="openTaskLatestResult(task)"')
    expect(supplierAutomationSource).toContain('@click="openRunDetail(run)"')
    expect(supplierAutomationSource).toContain('const detailRun = ref<SupplierAutomationRun | null>(null)')
    expect(supplierAutomationSource).toContain('v-if="detailRun"')
    expect(supplierAutomationSource).toContain('sp-run-detail-summary')
    expect(supplierAutomationSource).toContain('sp-provider-card')
    expect(supplierAutomationSource).toContain('sp-stage-card')
    expect(supplierAutomationSource).toContain('providerStagesByCategory(selectedDetailProvider)')
    expect(supplierAutomationSource).toContain('响应摘要')
  })

  it('uses an indexed provider detail layout and defaults to the first failed provider', () => {
    expect(supplierAutomationSource).toContain('sp-provider-detail-layout')
    expect(supplierAutomationSource).toContain('sp-provider-index')
    expect(supplierAutomationSource).toContain('selectedDetailProvider')
    expect(supplierAutomationSource).toContain('selectDetailProvider(provider.provider_id)')
    expect(supplierAutomationSource).toContain('selectInitialDetailProvider(run)')
    expect(supplierAutomationSource).toContain("providers.find(provider => provider.status === 'failed')")
    expect(supplierAutomationSource).toContain('v-if="selectedDetailProvider"')
    expect(supplierAutomationSource).not.toContain('v-for="provider in detailRun.result_detail.providers" :key="provider.provider_id" class="sp-provider-card"')
  })

  it('uses light result detail cards instead of dark log panels', () => {
    expect(supplierAutomationSource).toContain('sp-response-summary')
    expect(supplierAutomationSource).not.toContain('background: rgba(15, 23, 42, 0.86)')
    expect(supplierAutomationSource).not.toContain('background: rgba(2, 6, 23, 0.34)')
    expect(supplierAutomationSource).not.toContain('linear-gradient(145deg, rgba(15, 23, 42, 0.82)')
  })

  it('adds status-aware visual hierarchy to the structured result dialog', () => {
    expect(supplierAutomationSource).toContain(":class=\"['sp-run-detail', statusTone(detailRun.status)]\"")
    expect(supplierAutomationSource).toContain(":class=\"['sp-provider-card', 'sp-provider-detail-card', statusTone(selectedDetailProvider.status)]\"")
    expect(supplierAutomationSource).toContain('.sp-run-detail.bad')
    expect(supplierAutomationSource).toContain('.sp-provider-detail-card.bad')
    expect(supplierAutomationSource).toContain('.sp-run-detail .sp-status.bad')
    expect(supplierAutomationSource).toContain('.sp-tag {')
    expect(supplierAutomationSource).toContain('<span class="sp-tag neutral">处理 {{ selectedDetailProvider.counts.checked_count }}</span>')
    expect(supplierAutomationSource).toContain('<span v-if="stage.http_status" class="sp-tag http">HTTP {{ stage.http_status }}</span>')
  })

  it('keeps the result modal palette while flattening the outer detail surface', () => {
    expect(supplierAutomationSource).toContain(':global(.modal-content:has(.sp-run-detail)) {')
    expect(supplierAutomationSource).toContain('--sp-panel: #ffffff;')
    expect(supplierAutomationSource).toContain('--sp-result-blue-soft: #eaf2ff;')
    expect(supplierAutomationSource).toContain(':global(.modal-content:has(.sp-run-detail) .modal-body) {')
    expect(supplierAutomationSource).toContain('.sp-run-detail {')
    expect(supplierAutomationSource).toContain('padding: 4px 2px 12px;')
    expect(supplierAutomationSource).not.toContain('.sp-run-detail::before {')
    expect(supplierAutomationSource).not.toContain('box-shadow: 0 18px 42px rgba(15, 23, 42, 0.08);')
  })

  it('uses restrained color accents across summary, provider, and stage content', () => {
    expect(supplierAutomationSource).toContain('sp-summary-task')
    expect(supplierAutomationSource).toContain('sp-summary-trigger')
    expect(supplierAutomationSource).toContain('sp-summary-status')
    expect(supplierAutomationSource).toContain('sp-summary-counts')
    expect(supplierAutomationSource).toContain('sp-summary-start')
    expect(supplierAutomationSource).toContain('sp-summary-end')
    expect(supplierAutomationSource).toContain('.sp-provider-detail-card.good')
    expect(supplierAutomationSource).toContain('.sp-provider-detail-card.warn')
    expect(supplierAutomationSource).toContain('.sp-provider-detail-card.bad')
    expect(supplierAutomationSource).toContain('.sp-stage-category.identity')
    expect(supplierAutomationSource).toContain('.sp-stage-category.metrics')
    expect(supplierAutomationSource).toContain('.sp-stage-category.other')
    expect(supplierAutomationSource).toContain('.sp-provider-index-item.bad:not(.active)')
    expect(supplierAutomationSource).toContain('.sp-stage-card.bad')
    expect(supplierAutomationSource).toContain('class="sp-tag success">新增')
    expect(supplierAutomationSource).toContain('class="sp-tag primary">更新')
    expect(supplierAutomationSource).toContain('class="sp-tag warning">跳过')
    expect(supplierAutomationSource).toContain('class="sp-tag http">HTTP')
    expect(supplierAutomationSource).toContain('.sp-tag.success')
    expect(supplierAutomationSource).toContain('.sp-tag.warning')
  })

  it('uses separators and flat rows instead of nested cards in the result detail', () => {
    expect(supplierAutomationSource).toContain('class="sp-summary-item sp-summary-task"')
    expect(supplierAutomationSource).toContain('.sp-summary-item {')
    expect(supplierAutomationSource).toContain('border-right: 1px solid var(--sp-line);')
    expect(supplierAutomationSource).toContain('.sp-stage-card + .sp-stage-card {')
    expect(supplierAutomationSource).toContain('.sp-response-summary {')
    expect(supplierAutomationSource).not.toContain('sp-summary-card')
  })

  it('shows raw response summaries in a wider dialog with a split stage layout', () => {
    expect(supplierAutomationSource).toContain('sp-stage-body')
    expect(supplierAutomationSource).toContain('sp-stage-main')
    expect(supplierAutomationSource).toContain('sp-response-panel')
    expect(supplierAutomationSource).toContain('<pre class="sp-response-summary">{{ stage.response_summary }}</pre>')
    expect(supplierAutomationSource).not.toContain('parseResponseSummaryItems')
    expect(supplierAutomationSource).not.toContain('sp-response-summary-list')
    expect(supplierAutomationSource).not.toContain('sp-response-summary-item')
    expect(supplierAutomationSource).toContain('width="extra-wide"')
    expect(supplierAutomationSource).toContain("import BaseDialog from '@/components/common/BaseDialog.vue'")
    expect(supplierAutomationSource).toContain('<BaseDialog :show="detailVisible"')
  })

  it('adds clear spacing between flat result detail sections', () => {
    expect(supplierAutomationSource).toContain('padding: 18px 0')
    expect(supplierAutomationSource).toContain('padding: 16px 0')
    expect(supplierAutomationSource).toContain('gap: 16px')
    expect(supplierAutomationSource).toContain('margin-top: 16px')
  })

  it('paginates automation run history', () => {
    expect(supplierAutomationSource).toContain("import Pagination from '@/components/common/Pagination.vue'")
    expect(supplierAutomationSource).toContain('<Pagination')
    expect(supplierAutomationSource).toContain('const runPage = ref(1)')
    expect(supplierAutomationSource).toContain('const runPageSize = ref(10)')
    expect(supplierAutomationSource).toContain('page: runPage.value')
    expect(supplierAutomationSource).toContain('page_size: runPageSize.value')
    expect(supplierAutomationSource).toContain('@update:page="changeRunPage"')
    expect(supplierAutomationSource).not.toContain('sp-run-pager')
  })

  it('filters automation run history with server-side task and status params', () => {
    expect(supplierAutomationSource).toContain('const runTaskFilter = ref(\'\')')
    expect(supplierAutomationSource).toContain('const runStatusFilter = ref(\'\')')
    expect(supplierAutomationSource).toContain('data-test="run-task-filter"')
    expect(supplierAutomationSource).toContain('data-test="run-status-filter"')
    expect(supplierAutomationSource).toContain('task_code: runTaskFilter.value || undefined')
    expect(supplierAutomationSource).toContain('status: runStatusFilter.value || undefined')
    expect(supplierAutomationSource).toContain('resetRunFilters')
  })

  it('uses common framework form controls instead of native select and input elements', () => {
    expect(supplierAutomationSource).toContain("import Select, { type SelectOption } from '@/components/common/Select.vue'")
    expect(supplierAutomationSource).toContain("import Input from '@/components/common/Input.vue'")
    expect(supplierAutomationSource).toContain('<Select')
    expect(supplierAutomationSource).toContain('<Input')
    expect(supplierAutomationSource).not.toContain('<select')
    expect(supplierAutomationSource).not.toContain('<input')
  })

  it('uses common framework table and dialog components instead of local table and modal markup', () => {
    expect(supplierAutomationSource).toContain("import DataTable from '@/components/common/DataTable.vue'")
    expect(supplierAutomationSource).toContain("import BaseDialog from '@/components/common/BaseDialog.vue'")
    expect(supplierAutomationSource).toContain('<DataTable')
    expect(supplierAutomationSource).toContain('<BaseDialog')
    expect(supplierAutomationSource).not.toContain('<table')
    expect(supplierAutomationSource).not.toContain('SupplierModal')
  })

  it('resets run history pagination when filters change', () => {
    expect(supplierAutomationSource).toContain('@change="applyRunFilters"')
    expect(supplierAutomationSource).toContain('async function applyRunFilters()')
    expect(supplierAutomationSource).toContain('runPage.value = 1')
    expect(supplierAutomationSource).toContain('await refreshRuns()')
  })

  it('shows run time in automation run history rows', () => {
    expect(supplierAutomationSource).toContain("{ key: 'started_at', label: '运行时间'")
    expect(supplierAutomationSource).toContain('{{ formatTime(run.started_at) }}')
    expect(supplierAutomationSource).toContain(':columns="runColumns"')
  })

  it('displays automation trigger and status values in Chinese', () => {
    expect(supplierAutomationSource).toContain('triggerText(run.trigger_source)')
    expect(supplierAutomationSource).toContain("if (trigger === 'scheduled') return '定时执行'")
    expect(supplierAutomationSource).toContain("if (trigger === 'manual') return '手动执行'")
    expect(supplierAutomationSource).toContain('statusText(run.status)')
  })
})

import { readFileSync } from 'node:fs'
import { createRequire } from 'node:module'
import { dirname, resolve } from 'node:path'
import { fileURLToPath } from 'node:url'
import { describe, expect, it } from 'vitest'

const supplierAutomationSource = readFileSync(
  resolve(dirname(fileURLToPath(import.meta.url)), 'SupplierAutomationView.vue'),
  'utf-8'
)
const supplierAutomationAPISource = readFileSync(
  resolve(dirname(fileURLToPath(import.meta.url)), '../../../api/admin/supplierAutomation.ts'),
  'utf-8'
)
const testRequire = createRequire(import.meta.url)
const compilerSfcPath = testRequire.resolve('@vue/compiler-sfc', {
  paths: [dirname(testRequire.resolve('@vitejs/plugin-vue'))],
})
const { compileStyle } = testRequire(compilerSfcPath) as {
  compileStyle: (options: {
    id: string
    source: string
    scoped: boolean
    filename: string
  }) => { code: string; errors: unknown[] }
}

describe('SupplierAutomationView second-level intervals', () => {
  it('reuses the shared account rate guard log dialog without duplicate headings', () => {
    expect(supplierAutomationSource).toContain('<SupplierAccountRateGuardLogDialog')
    expect(supplierAutomationSource).not.toContain('Account Rate Guard Audit')
    expect(supplierAutomationSource).not.toContain('账号与分组解绑轨迹')
  })

  it('stores positive integer seconds as @every descriptors', () => {
    expect(supplierAutomationSource).toContain('if (!Number.isInteger(seconds) || seconds < 1) return null')
    expect(supplierAutomationSource).toContain('return `@every ${seconds}s`')
    expect(supplierAutomationSource).toContain("appStore.showError('执行间隔必须是正整数秒')")
    expect(supplierAutomationSource).toContain('执行间隔最小为 1 秒，可按正整数秒配置。')
  })

  it('reads @every seconds and keeps legacy five-field cron compatibility', () => {
    expect(supplierAutomationSource).toContain("cronExpression.match(/^@every\\s+(\\d+)s$/)")
    expect(supplierAutomationSource).toContain('if (everyMatch) return Number(everyMatch[1])')
    expect(supplierAutomationSource).toContain('if (parts.length !== 5) return null')
    expect(supplierAutomationSource).toContain('editIntervalSeconds.value = cronToIntervalSeconds(task.cron_expression) || 300')
  })
})

describe('SupplierAutomationView edit dialog', () => {
  it('shows the task name and code in each run history row', () => {
    expect(supplierAutomationSource).toContain('<template #cell-task_code="{ row: run }">')
    expect(supplierAutomationSource).toContain('{{ taskName(run.task_code) }}')
    expect(supplierAutomationSource).toContain('<div class="sp-sub">{{ run.task_code }}</div>')
    expect(supplierAutomationSource).toContain('const taskNameByCode = computed<Record<string, string>>')
    expect(supplierAutomationSource).toContain('function taskName(taskCode: string): string')
  })

  it('assigns every task code a stable generated name color in tasks and run history', () => {
    expect(supplierAutomationSource).toContain(
      'class="sp-entity sp-task-name" :style="taskColorStyle(task.task_code)"'
    )
    expect(supplierAutomationSource).toContain(
      'class="sp-entity sp-task-name" :style="taskColorStyle(run.task_code)"'
    )
    expect(supplierAutomationSource).toContain('function stableTaskColorHash(value: string): number')
    expect(supplierAutomationSource).toContain('function taskColorStyle(taskCode: string): Record<string, string>')
    expect(supplierAutomationSource).toContain("'--sp-task-hue'")
    expect(supplierAutomationSource).toContain("'--sp-task-saturation'")
    expect(supplierAutomationSource).toContain('.sp-task-name {')
    expect(supplierAutomationSource).toContain(':global(.dark .sp-automation-console .sp-task-name) {')
  })

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

  it('opens structured run detail after executing a task and surfaces API error messages', () => {
    expect(supplierAutomationSource).toContain('const run = await runTask(taskCode)')
    expect(supplierAutomationSource).toContain("appStore.showSuccess('任务已保存')")
    expect(supplierAutomationSource).toContain('appStore.showSuccess(`任务执行完成：${statusText(run.status)}`)')
    expect(supplierAutomationSource).toContain("import { useAppStore } from '@/stores/app'")
    expect(supplierAutomationSource).not.toContain('class="sp-toast"')
    expect(supplierAutomationSource).toContain('openRunDetail(run)')
    expect(supplierAutomationSource).toContain("import { extractApiErrorMessage } from '@/utils/apiError'")
    expect(supplierAutomationSource).toContain("extractApiErrorMessage(err, '运行任务失败')")
    expect(supplierAutomationSource).toContain('await openTaskLatestResult(task)')
    expect(supplierAutomationSource).not.toContain('openResultDetail(`${taskCode} 执行结果')
  })

  it('opens structured run details from latest result and run history rows', () => {
    expect(supplierAutomationSource).toContain('async function openTaskLatestResult')
    expect(supplierAutomationSource).toContain('task_code: task.task_code')
    expect(supplierAutomationSource).toContain('page_size: 1')
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

  it('renders latest-result details in a separate task table column', () => {
    const latestResultCell = supplierAutomationSource.match(
      /<template #cell-last_status="\{ row: task \}">[\s\S]*?<\/template>/
    )?.[0] || ''
    const detailsCell = supplierAutomationSource.match(
      /<template #cell-details="\{ row: task \}">[\s\S]*?<\/template>/
    )?.[0] || ''

    expect(supplierAutomationSource).toContain("{ key: 'details',")
    expect(detailsCell).toContain('@click.stop="openTaskLatestResult(task)"')
    expect(latestResultCell).not.toContain('openTaskLatestResult(task)')
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

  it('separates execution outcome from result detail while preserving every detail branch', () => {
    const detailDialogSource = supplierAutomationSource.match(
      /<BaseDialog :show="detailVisible"[\s\S]*?<\/BaseDialog>/
    )?.[0] || ''

    expect(detailDialogSource).toContain('<section class="sp-detail-outcome">')
    expect(detailDialogSource).toContain('<section class="sp-detail-content">')
    expect(detailDialogSource.match(/class="sp-detail-section-head"/g)).toHaveLength(2)
    expect(detailDialogSource).toContain('Execution Outcome')
    expect(detailDialogSource).toContain('执行结论')
    expect(detailDialogSource).toContain('Result Detail')
    expect(detailDialogSource).toContain('结果明细')
    expect(detailDialogSource).toContain(
      '<span class="sp-status" :class="statusTone(detailRun.status)">{{ statusText(detailRun.status) }}</span>'
    )

    const outcomeIndex = detailDialogSource.indexOf('<section class="sp-detail-outcome">')
    const summaryIndex = detailDialogSource.indexOf('<section class="sp-run-detail-summary">')
    const messageIndex = detailDialogSource.indexOf(
      '<div v-if="detailRun.message" class="sp-run-message">{{ detailRun.message }}</div>'
    )
    const contentIndex = detailDialogSource.indexOf('<section class="sp-detail-content">')
    const rateGuardIndex = detailDialogSource.indexOf(
      '<section v-if="detailRun.result_detail?.rate_guard && rateGuardResult" class="sp-rate-guard-detail">'
    )
    const providersIndex = detailDialogSource.indexOf(
      '<section v-else-if="detailRun.result_detail?.providers?.length" class="sp-provider-detail-layout">'
    )
    const cleanupIndex = detailDialogSource.indexOf(
      '<section v-else-if="detailRun.result_detail?.cleanup" class="sp-cleanup-grid">'
    )
    const textFallbackIndex = detailDialogSource.indexOf(
      '<pre v-else class="sp-message-detail">{{ detailMessage }}</pre>'
    )

    expect(outcomeIndex).toBeGreaterThanOrEqual(0)
    expect(summaryIndex).toBeGreaterThan(outcomeIndex)
    expect(messageIndex).toBeGreaterThan(summaryIndex)
    expect(contentIndex).toBeGreaterThan(messageIndex)
    expect(rateGuardIndex).toBeGreaterThan(contentIndex)
    expect(providersIndex).toBeGreaterThan(rateGuardIndex)
    expect(cleanupIndex).toBeGreaterThan(providersIndex)
    expect(textFallbackIndex).toBeGreaterThan(cleanupIndex)
    expect(detailDialogSource).toContain(
      '</div>\n        <pre v-else class="sp-message-detail">{{ detailMessage }}</pre>'
    )
  })

  it('uses restrained outcome emphasis, bounded response scrolling, and edge-only failure states', () => {
    expect(supplierAutomationSource).toMatch(
      /\.sp-detail-outcome,\r?\n\.sp-detail-content \{\r?\n\s+display: grid;\r?\n\s+gap: 14px;\r?\n\}/
    )
    expect(supplierAutomationSource).toContain('.sp-detail-section-head {')
    expect(supplierAutomationSource).toContain('.sp-detail-section-head h3 {')
    expect(supplierAutomationSource).toContain('text-transform: uppercase;')
    expect(supplierAutomationSource).toContain('border-left: 3px solid var(--sp-result-accent);')

    expect(supplierAutomationSource).toMatch(
      /\.sp-detail-content \{\r?\n\s+border-top: 1px solid var\(--sp-line\);\r?\n\s+margin-top: [^;]+;\r?\n\s+padding-top: [^;]+;\r?\n\}/
    )

    const responseSummaryStyles = supplierAutomationSource.match(
      /\.sp-response-summary \{[\s\S]*?\n\}/
    )?.[0] || ''
    expect(responseSummaryStyles).toContain('max-height: 240px;')
    expect(responseSummaryStyles).toContain('overflow: auto;')

    const failedStageCardStyles = supplierAutomationSource.match(
      /\.sp-stage-card\.bad \{[\s\S]*?\n\}/
    )?.[0] || ''
    expect(failedStageCardStyles).toContain('border-left: 3px solid var(--sp-red);')
    expect(failedStageCardStyles).not.toContain('background: var(--sp-result-red-soft);')
    expect(failedStageCardStyles).not.toContain('padding: 8px 10px;')
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

  it('uses common framework controls for form fields and reserves native input for account checkboxes', () => {
    const nativeInputs = supplierAutomationSource.match(/<input\b[\s\S]*?\/>/g) || []

    expect(supplierAutomationSource).toContain("import Select, { type SelectOption } from '@/components/common/Select.vue'")
    expect(supplierAutomationSource).toContain("import Input from '@/components/common/Input.vue'")
    expect(supplierAutomationSource).toContain('<Select')
    expect(supplierAutomationSource).toContain('<Input')
    expect(supplierAutomationSource).not.toContain('<select')
    expect(nativeInputs).toHaveLength(1)
    expect(nativeInputs[0]).toContain('type="checkbox"')
    expect(nativeInputs[0]).toContain('@change="toggleHealthGuardAccount(mapping.localAccountID)"')
  })

  it('shows rate guard settings only for the rate guard task', () => {
    expect(supplierAutomationSource).toContain("editForm.task_code === 'supplier_rate_guard'")
    expect(supplierAutomationSource).not.toContain('rate_guard_safety_multiplier')
    expect(supplierAutomationSource).toContain('editForm.config.rate_guard_max_snapshot_age_seconds')
    expect(supplierAutomationSource).toContain('label="快照最大有效期（秒）"')
  })

  it('validates rate guard settings before saving the task', () => {
    expect(supplierAutomationSource).not.toContain('rate_guard_safety_multiplier')
    expect(supplierAutomationSource).toContain('editForm.config.rate_guard_max_snapshot_age_seconds < 60')
    expect(supplierAutomationSource).toContain("appStore.showError('快照最大有效期不能少于 60 秒')")
  })

  it('declares supplier account health guard configuration and result types', () => {
    const configFields = [
      'account_health_guard_max_accounts_per_run',
      'account_health_guard_concurrency',
      'account_health_guard_timeout_per_account_seconds',
      'account_health_guard_failure_threshold',
      'account_health_guard_slow_threshold',
      'account_health_guard_recovery_threshold',
      'account_health_guard_healthy_latency_ms',
      'account_health_guard_account_ids',
      'account_health_guard_account_models',
      'account_health_guard_platform_models',
      'account_health_guard_platform_latency_ms',
      'account_health_guard_cursor_account_id',
    ]

    for (const field of configFields) {
      expect(supplierAutomationAPISource).toContain(field)
    }
    expect(supplierAutomationAPISource).toContain('export interface SupplierAccountHealthGuardResult')
    expect(supplierAutomationAPISource).toContain('export interface SupplierAccountHealthGuardItem')
    expect(supplierAutomationAPISource).toContain('export interface SupplierAccountHealthGuardSource')
    expect(supplierAutomationAPISource).toContain('export interface SupplierAccountHealthGuardSkipReason')
    expect(supplierAutomationAPISource).toContain('account_health_guard?: SupplierAccountHealthGuardResult')
    expect(supplierAutomationAPISource).toContain('selected_count: number')
    expect(supplierAutomationAPISource).toContain('unavailable_count: number')
    expect(supplierAutomationAPISource).toContain('pending_count: number')
    expect(supplierAutomationAPISource).toContain("status: 'healthy' | 'slow' | 'failed' | 'skipped' | 'unavailable' | string")
    expect(supplierAutomationAPISource).toContain("taskCode === 'supplier_account_health_guard'")
    expect(supplierAutomationAPISource).toContain('35 * 60 * 1000')
    expect(supplierAutomationAPISource).toContain('timeout ? { timeout } : undefined')
  })

  it('uses the ordinary immediate run action for account health guard', () => {
    expect(supplierAutomationSource).toContain("task.task_code === 'supplier_account_rate_guard'")
    expect(supplierAutomationSource).toContain('v-else class="sp-button small primary sp-task-primary"')
    expect(supplierAutomationSource).toContain('@click.stop="runNow(task.task_code)"')
    expect(supplierAutomationSource).not.toContain("runPreview('supplier_account_health_guard')")
    expect(supplierAutomationSource).not.toContain("openAccountRateGuardExecute('supplier_account_health_guard')")
  })

  it('renders and validates the account health guard policy form', () => {
    expect(supplierAutomationSource).toContain("editForm.task_code === 'supplier_account_health_guard'")
    expect(supplierAutomationSource).toContain('健康守护策略')
    expect(supplierAutomationSource).toContain('单次检查账号数')
    expect(supplierAutomationSource).toContain('并发数')
    expect(supplierAutomationSource).toContain('单账号超时（秒）')
    expect(supplierAutomationSource).toContain('连续失败暂停阈值')
    expect(supplierAutomationSource).toContain('连续慢响应暂停阈值')
    expect(supplierAutomationSource).toContain('连续健康恢复阈值')
    expect(supplierAutomationSource).toContain('默认健康延迟（毫秒）')
    expect(supplierAutomationSource).toContain('validateAccountHealthGuardConfig')
  })

  it('configures checked accounts with platform filters and searchable model overrides', () => {
    expect(supplierAutomationSource).toContain('listSupplierAccounts,')
    expect(supplierAutomationSource).toContain("from '@/api/admin/supplierProviderData'")
    expect(supplierAutomationSource).toContain("match_status: 'matched'")
    expect(supplierAutomationSource).toContain('page_size: 200')
    expect(supplierAutomationSource).toContain('while (items.length < result.total && result.items.length > 0)')
    expect(supplierAutomationSource).toContain('account_health_guard_account_ids')
    expect(supplierAutomationSource).not.toContain('account_health_guard_ignored_account_ids')
    expect(supplierAutomationSource).toContain('openHealthGuardAccounts')
    expect(supplierAutomationSource).toContain('healthGuardAccountsVisible')
    expect(supplierAutomationSource).toContain('healthGuardAccountPlatformFilter')
    expect(supplierAutomationSource).toContain('healthGuardAccountProviderFilter')
    expect(supplierAutomationSource).toContain('healthGuardProviderFilterOptions')
    expect(supplierAutomationSource).toContain('healthGuardAccountSearch')
    expect(supplierAutomationSource).toContain('供应商过滤')
    expect(supplierAutomationSource).toContain('平台默认测试模型')
    expect(supplierAutomationSource).toContain('需要检查的账号')
    expect(supplierAutomationSource).toContain('当前不可用')
    expect(supplierAutomationSource).toContain('getSupplierHealthGuardModels')
    expect(supplierAutomationSource).not.toContain('adminAPI.accounts.getAvailableModels')
    expect(supplierAutomationSource).toContain('searchable')
    expect(supplierAutomationSource).toContain('normalizePositiveAccountIDs')
    expect(supplierAutomationSource).toContain('type="checkbox"')
    expect(supplierAutomationSource).toContain(':checked="healthGuardAccountIDs.includes(mapping.localAccountID)"')
    expect(supplierAutomationSource).toContain('@change="toggleHealthGuardAccount(mapping.localAccountID)"')
  })

  it('uses the supplier effective business platform for health guard account grouping', () => {
    expect(supplierAutomationSource).toContain(
      'function effectiveHealthGuardPlatform(account: SupplierProviderAccount): string {'
    )
    expect(supplierAutomationSource).toContain(
      'account.effective_platform || account.local_account_platform || account.platform'
    )
    expect(supplierAutomationSource).toContain('getSupplierHealthGuardModels,')
    expect(supplierAutomationSource).toContain('await getSupplierHealthGuardModels(summary.representativeAccountID)')
    expect(supplierAutomationSource).toContain('current.platform = effectiveHealthGuardPlatform(account)')
    expect(supplierAutomationSource).toContain('platform: effectiveHealthGuardPlatform(account)')
  })

  it('filters health guard accounts by their bound local group platforms', () => {
    expect(supplierAutomationSource).toContain(
      'function healthGuardLocalGroupPlatforms(account: SupplierProviderAccount): string[] {'
    )
    expect(supplierAutomationSource).toContain(
      'account.binding_groups.map(group => normalizeHealthGuardPlatform(group.platform))'
    )
    expect(supplierAutomationSource).toContain('localGroupPlatforms: string[]')
    expect(supplierAutomationSource).toContain('mapping.localGroupPlatforms.includes(platform)')
    expect(supplierAutomationSource).toContain('for (const platform of mapping.localGroupPlatforms)')
  })

  it('renders health guard accounts as a unified selectable workspace', () => {
    expect(supplierAutomationSource).toContain('healthGuardSelectedOnly')
    expect(supplierAutomationSource).toContain('healthGuardWorkspaceAccounts')
    expect(supplierAutomationSource).toContain('healthGuardSelectionSummary')
    expect(supplierAutomationSource).toContain('仅看已选')
    expect(supplierAutomationSource).toContain('使用平台默认')
    expect(supplierAutomationSource).toContain('>覆盖</span>')
    expect(supplierAutomationSource).toContain('.sp-health-guard-account-row.selected')
    expect(supplierAutomationSource).toContain('.sp-health-guard-account-row.missing-model')
    expect(supplierAutomationSource).toContain('color: var(--sp-green)')
    expect(supplierAutomationSource).not.toContain('--sp-good')
    expect(supplierAutomationSource).not.toContain('sp-health-guard-account-columns')
  })

  it('keeps the selected-account filter from overlapping its result and colors account platforms', () => {
    expect(supplierAutomationSource).toContain("import { platformBadgeClass, platformLabel, platformTextClass } from '@/utils/platformColors'")
    expect(supplierAutomationSource).toContain(":class=\"['sp-health-guard-account-platform', platformBadgeClass(mapping.platform)]\"")
    expect(supplierAutomationSource).toContain('.sp-health-guard-account-toolbar')
    expect(supplierAutomationSource).toContain('flex-wrap: wrap')
    expect(supplierAutomationSource).toContain('flex: 1 1 640px')
    expect(supplierAutomationSource).toContain('.sp-health-guard-account-source')
  })

  it('applies the automation dialog palette to the teleported health guard workspace', () => {
    expect(supplierAutomationSource).toContain(':global(.modal-content:has(.sp-health-guard-account-dialog))')
    expect(supplierAutomationSource).toContain(':global(.dark .modal-content:has(.sp-health-guard-account-dialog))')
    expect(supplierAutomationSource).toContain(':global(.modal-content:has(.sp-health-guard-account-dialog) .modal-body)')
    expect(supplierAutomationSource).toContain(':global(.modal-content:has(.sp-health-guard-account-dialog) .modal-footer)')
  })

  it('validates checked accounts and effective models before saving or running', () => {
    expect(supplierAutomationSource).toContain('请至少选择一个需要检查的账号')
    expect(supplierAutomationSource).toContain('以下账号尚未配置测试模型：')
    expect(supplierAutomationSource).toContain('supplierAccountHealthGuardModelForMapping')
    expect(supplierAutomationSource).toContain("if (taskCode === 'supplier_account_health_guard')")
    expect(supplierAutomationSource).toContain('await ensureHealthGuardAccountCandidatesLoaded()')
  })
  it('provides dedicated preview, execute confirmation, and unbind log actions for account rate guard', () => {
    expect(supplierAutomationSource).toContain("task.task_code === 'supplier_account_rate_guard'")
    expect(supplierAutomationSource).toContain('检测预览')
    expect(supplierAutomationSource).toContain('立即执行')
    expect(supplierAutomationSource).toContain('解除绑定日志')
    expect(supplierAutomationSource).toContain("await runTask(taskCode, 'preview')")
    expect(supplierAutomationSource).toContain("await runTask(pendingExecuteTask.value.task_code, 'execute')")
    expect(supplierAutomationSource).toContain('accountRateGuardExecuteVisible')
    expect(supplierAutomationSource).toContain('执行前会重新同步账号倍率，并解除所有不合格的账号与分组绑定')
  })

  it('renders account health guard summaries, filters, and account-level details', () => {
    expect(supplierAutomationSource).toContain('const accountHealthGuardResult = computed')
    expect(supplierAutomationSource).toContain('const healthGuardStatusFilter = ref')
    expect(supplierAutomationSource).toContain('const accountHealthGuardSummaryMetrics = computed')
    expect(supplierAutomationSource).toContain('function setHealthGuardStatusFilter(filter: string)')
    expect(supplierAutomationSource).toContain("value: 'checked'")
    expect(supplierAutomationSource).toContain("value: 'healthy'")
    expect(supplierAutomationSource).toContain("value: 'slow'")
    expect(supplierAutomationSource).toContain("value: 'failed'")
    expect(supplierAutomationSource).toContain("value: 'skipped'")
    expect(supplierAutomationSource).toContain("value: 'unavailable'")
    expect(supplierAutomationSource).toContain("value: 'disabled'")
    expect(supplierAutomationSource).toContain("value: 'recovered'")
    expect(supplierAutomationSource).toContain('health-guard-summary-filter-')
    expect(supplierAutomationSource).toContain('@click="setHealthGuardStatusFilter(metric.filter)"')
    expect(supplierAutomationSource).toContain("filter === 'checked'")
    expect(supplierAutomationSource).toContain("filter === 'disabled' || filter === 'recovered'")
    expect(supplierAutomationSource).toContain('健康守护明细')
    for (const label of ['健康', '慢响应', '失败', '不可用', '待下轮', '暂停', '恢复']) {
      expect(supplierAutomationSource).toContain(label)
    }
    for (const field of [
      'item.local_account_name',
      'item.sources',
      'item.platform',
      'item.model_id',
      'item.latency_ms',
      'item.consecutive_failed',
      'item.consecutive_slow',
      'item.consecutive_healthy',
      'item.schedulable_before',
      'item.schedulable_after',
      'item.action',
      'item.reason',
      'item.error_message',
    ]) {
      expect(supplierAutomationSource).toContain(field)
    }
  })
  it('opens the shared account rate guard log dialog', () => {
    expect(supplierAutomationSource).toContain('SupplierAccountRateGuardLogDialog')
    expect(supplierAutomationSource).toContain('accountRateGuardLogsVisible')
    expect(supplierAutomationSource).toContain('@close="closeAccountRateGuardLogs"')
    expect(supplierAutomationSource).toContain('@pending-count-change="updateAccountRateGuardPendingCount"')
    expect(supplierAutomationSource).not.toContain('accountRateGuardLogColumns')
    expect(supplierAutomationSource).not.toContain('openAccountRateGuardLogError')
  })

  it('shows the current pending account rate guard log count on the action button', () => {
    expect(supplierAutomationSource).toContain('listAccountRateGuardUnbindLogs')
    expect(supplierAutomationSource).toContain('const accountRateGuardPendingCount = ref(0)')
    expect(supplierAutomationSource).toContain('await loadAccountRateGuardPendingCount()')
    expect(supplierAutomationSource).toContain("status: 'pending'")
    expect(supplierAutomationSource).toContain('accountRateGuardPendingCount.value = result.pending_count')
    expect(supplierAutomationSource).toContain('v-if="accountRateGuardPendingCount > 0"')
    expect(supplierAutomationSource).toContain('class="sp-unbind-log-count"')
    expect(supplierAutomationSource).toContain(
      'async function updateAccountRateGuardPendingCount() {\n  await loadAccountRateGuardPendingCount()\n}'
    )
  })

  it('renders rate guard summary counters and item details', () => {
    expect(supplierAutomationSource).toContain('detailRun.result_detail?.rate_guard')
    expect(supplierAutomationSource).toContain('rateGuardResult.checked')
    expect(supplierAutomationSource).toContain('rateGuardResult.raised')
    expect(supplierAutomationSource).toContain('rateGuardResult.unchanged')
    expect(supplierAutomationSource).toContain('rateGuardResult.duplicate')
    expect(supplierAutomationSource).toContain('rateGuardResult.stale')
    expect(supplierAutomationSource).toContain('rateGuardResult.invalid')
    expect(supplierAutomationSource).toContain('rateGuardResult.failed')
    expect(supplierAutomationSource).toContain('rateGuardResult.items')
    expect(supplierAutomationSource).toContain('item.old_rate')
    expect(supplierAutomationSource).toContain('item.target_rate')
    expect(supplierAutomationSource).toContain('rateGuardActionText(item.action)')
    expect(supplierAutomationSource).toContain('rateGuardReasonText(item.reason)')
  })

  it('surfaces rate guard warnings before the complete inspection list', () => {
    expect(supplierAutomationSource).toContain(
      "const rateGuardAlertActions = new Set(['invalid', 'stale', 'failed'])"
    )
    expect(supplierAutomationSource).toContain(
      'rateGuardResult.value?.items.filter(item => rateGuardAlertActions.has(item.action)) || []'
    )
    expect(supplierAutomationSource).toContain('<h4>告警记录</h4>')
    expect(supplierAutomationSource).toContain('v-for="item in rateGuardAlertItems"')
    expect(supplierAutomationSource).toContain('rateGuardReasonText(item.reason)')
    expect(supplierAutomationSource).toContain("'未关联本地分组'")

    const alertsIndex = supplierAutomationSource.indexOf('<section v-if="rateGuardAlertItems.length" class="sp-rate-guard-alerts">')
    const changesIndex = supplierAutomationSource.indexOf('<section class="sp-rate-guard-changes">')
    expect(alertsIndex).toBeGreaterThanOrEqual(0)
    expect(changesIndex).toBeGreaterThan(alertsIndex)
  })

  it('shows rate guard raised and warning counts in run history', () => {
    expect(supplierAutomationSource).toContain('rateGuardWarningCount(run)')
    expect(supplierAutomationSource).toContain('调高 {{ run.result_detail.rate_guard.raised }}')
    expect(supplierAutomationSource).toContain('告警 {{ rateGuardWarningCount(run) }}')
  })

  it('preserves meaningful rate precision in result details', () => {
    expect(supplierAutomationSource).toContain("rate.toFixed(4).replace(/\\.?0+$/, '')")
    expect(supplierAutomationSource).not.toContain('rate.toFixed(2)')
  })

  it('shows actual rate changes before the complete rate guard inspection results', () => {
    expect(supplierAutomationSource).toContain(
      "rateGuardResult.value?.items.filter(item => item.action === 'raised') || []"
    )
    expect(supplierAutomationSource).toContain('<h4>倍率变更记录</h4>')
    expect(supplierAutomationSource).toContain('v-for="item in rateGuardRaisedItems"')
    expect(supplierAutomationSource).toContain('本次未调整本地分组倍率。')
    expect(supplierAutomationSource).toContain('<h4>全部检查结果</h4>')

    const changesIndex = supplierAutomationSource.indexOf('<section class="sp-rate-guard-changes">')
    const allItemsIndex = supplierAutomationSource.indexOf('<section class="sp-rate-guard-inspections">')
    expect(changesIndex).toBeGreaterThanOrEqual(0)
    expect(allItemsIndex).toBeGreaterThan(changesIndex)
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
    expect(supplierAutomationSource).toContain("if (status === 'skipped') return '已跳过'")
  })
})

describe('SupplierAutomationView operations console composition', () => {
  it('uses a vertical overview, task control, and run history hierarchy', () => {
    expect(supplierAutomationSource).toContain('class="sp-automation-console"')
    expect(supplierAutomationSource).toContain('class="sp-overview-strip"')
    expect(supplierAutomationSource).toContain('class="sp-console-stack"')
    expect(supplierAutomationSource).toContain('class="sp-console-panel sp-task-panel"')
    expect(supplierAutomationSource).toContain('class="sp-console-panel sp-history-panel"')
    expect(supplierAutomationSource).not.toContain('class="sp-grid-2"')

    expect(supplierAutomationSource).toMatch(
      /<div\b[^>]*class\s*=\s*["'][^"']*\bsp-console-stack\b[^"']*["'][^>]*>[\s\S]*?<section\b[^>]*class\s*=\s*["'][^"']*\bsp-task-panel\b[^"']*["'][^>]*>[\s\S]*?<\/section>[\s\S]*?<section\b[^>]*class\s*=\s*["'][^"']*\bsp-history-panel\b[^"']*["'][^>]*>[\s\S]*?<\/section>\s*<\/div>/
    )

    const classTokens = [...supplierAutomationSource.matchAll(/\sclass\s*=\s*["']([^"']*)["']/g)]
      .flatMap(([, classNames]) => classNames.trim().split(/\s+/))
    expect(classTokens).not.toContain('sp-grid-2')
  })

  it('shows operational metrics and refresh context without row-selection metrics', () => {
    expect(supplierAutomationSource).toContain("label: '任务总数'")
    expect(supplierAutomationSource).toContain("label: '已启用'")
    expect(supplierAutomationSource).toContain("label: '最近异常'")
    expect(supplierAutomationSource).toContain("label: '正在运行'")
    expect(supplierAutomationSource).toContain('lastRefreshLabel')
    expect(supplierAutomationSource).not.toContain("label: '当前选中'")
  })

  it('removes task row-selection state and interactions', () => {
    expect(supplierAutomationSource).not.toContain('const selectedCode')
    expect(supplierAutomationSource).not.toContain('const selectedTask')
    expect(supplierAutomationSource).not.toContain('clickable-rows')
    expect(supplierAutomationSource).not.toContain('@row-click')
  })

  it('records the refresh time only after run history is loaded', () => {
    const loadDataIndex = supplierAutomationSource.indexOf('async function loadData()')
    const loadRunsIndex = supplierAutomationSource.indexOf('await loadRuns()', loadDataIndex)
    const refreshedAtIndex = supplierAutomationSource.indexOf(
      'lastRefreshedAt.value = new Date().toISOString()',
      loadDataIndex
    )

    expect(loadDataIndex).toBeGreaterThanOrEqual(0)
    expect(loadRunsIndex).toBeGreaterThan(loadDataIndex)
    expect(refreshedAtIndex).toBeGreaterThan(loadRunsIndex)
  })

  it('keeps new console children visually nested in the source', () => {
    expect(supplierAutomationSource).toMatch(
      /^ {8}<section\b[^>]*class="[^"]*\bsp-task-panel\b[^"]*">\n {10}<header/m
    )
    expect(supplierAutomationSource).toMatch(
      /^ {8}<section\b[^>]*class="[^"]*\bsp-history-panel\b[^"]*">\n {10}<header/m
    )
    expect(supplierAutomationSource.match(/^ {6}<BaseDialog\b/gm)).toHaveLength(4)
    expect(supplierAutomationSource).toMatch(/^ {6}<SupplierAccountRateGuardLogDialog\b/m)
    expect(supplierAutomationSource).not.toContain('<Transition name="sp-fade">')
    expect(supplierAutomationSource).not.toContain('class="sp-toast"')
  })

  it('keeps shared table and dialog components unchanged', () => {
    expect(supplierAutomationSource).toContain('<DataTable')
    expect(supplierAutomationSource).toContain('<BaseDialog')
    expect(supplierAutomationSource).not.toContain('<table')
  })
})
describe('SupplierAutomationView task and history panels', () => {
  it('presents task signals and a primary run action', () => {
    expect(supplierAutomationSource).toContain('class="sp-panel-signals"')
    expect(supplierAutomationSource).toContain('已启用 {{ enabledTaskCount }} / {{ tasks.length }}')
    expect(supplierAutomationSource).toContain('class="sp-button small primary sp-task-primary"')
    expect(supplierAutomationSource).toContain("{ key: 'actions', label: '操作'")
  })

  it('uses a full-width history toolbar with a visible result count', () => {
    expect(supplierAutomationSource).toContain('class="sp-history-toolbar"')
    expect(supplierAutomationSource).toContain('class="sp-history-count"')
    expect(supplierAutomationSource).toContain('{{ runTotal }} 条记录')
    expect(supplierAutomationSource).toContain('data-test="run-task-filter"')
    expect(supplierAutomationSource).toContain('data-test="run-status-filter"')
  })

  it('removes secondary task columns that belong in the edit dialog', () => {
    expect(supplierAutomationSource).not.toContain("{ key: 'timeout_seconds', label: '超时' }")
    expect(supplierAutomationSource).not.toContain("{ key: 'next_run_at', label: '下次运行'")
  })
})
describe('SupplierAutomationView Task 5 visual system', () => {
  const styleSource = (supplierAutomationSource.match(/<style scoped>([\s\S]*?)<\/style>/)?.[1] || '').replace(/\r\n/g, '\n')
  const tabletBreakpoint = styleSource.indexOf('@media (max-width: 1024px)')
  const mobileBreakpoint = styleSource.indexOf('@media (max-width: 760px)')
  const tableCardBreakpoint = styleSource.indexOf('@media (max-width: 767px)')
  const darkOverrideIndex = styleSource.indexOf(':global(.dark', Math.max(tableCardBreakpoint, 0))
  const tabletStyles = tabletBreakpoint >= 0 && mobileBreakpoint > tabletBreakpoint
    ? styleSource.slice(tabletBreakpoint, mobileBreakpoint)
    : ''
  const mobileStyles = mobileBreakpoint >= 0
    ? styleSource.slice(mobileBreakpoint, tableCardBreakpoint > mobileBreakpoint ? tableCardBreakpoint : undefined)
    : ''
  const tableCardStyles = tableCardBreakpoint >= 0
    ? styleSource.slice(tableCardBreakpoint, darkOverrideIndex > tableCardBreakpoint ? darkOverrideIndex : undefined)
    : ''
  const compiledStyle = compileStyle({
    id: 'data-v-supplier-automation-task-5',
    source: styleSource,
    scoped: true,
    filename: 'SupplierAutomationView.vue',
  })

  it('defines the compact operations console hierarchy with page-private selectors', () => {
    const selectors = [
      '.sp-automation-console {',
      '.sp-console-head {',
      '.sp-overview-strip {',
      '.sp-overview-item {',
      '.sp-console-stack {',
      '.sp-console-panel {',
      '.sp-panel-head {',
      '.sp-panel-kicker {',
      '.sp-panel-title h2 {',
      '.sp-panel-title p {',
      '.sp-panel-signals,\n.sp-history-count {',
      '.sp-panel-signals span {',
      '.sp-panel-signals .bad {',
      '.sp-history-toolbar {',
      '.sp-task-primary {',
    ]

    for (const selector of selectors) {
      expect(styleSource).toContain(selector)
    }

    expect(styleSource).toMatch(/\.sp-automation-console\s*\{[^}]*display:\s*grid;[^}]*gap:\s*18px;[^}]*min-width:\s*0;/s)
    expect(styleSource).toMatch(/\.sp-overview-strip\s*\{[^}]*grid-template-columns:\s*repeat\(4,\s*minmax\(0,\s*1fr\)\);[^}]*gap:\s*12px;[^}]*background:\s*transparent;/s)
    expect(styleSource).toMatch(/\.sp-overview-item\s*\{[^}]*position:\s*relative;[^}]*overflow:\s*hidden;[^}]*border:\s*1px solid color-mix\(in srgb, var\(--sp-metric-accent\) 18%, var\(--sp-soft\)\);[^}]*border-radius:\s*14px;/s)
    expect(styleSource).toMatch(/\.sp-console-panel\s*\{[^}]*border:\s*1px solid var\(--sp-soft\);[^}]*border-radius:\s*14px;[^}]*background:\s*var\(--sp-panel\);/s)
    expect(styleSource).toMatch(/\.sp-task-primary\s*\{[^}]*min-width:\s*76px;/s)
  })

  it('uses restrained semantic accents on independent overview cards', () => {
    for (const token of ['sp-metric-grid', 'sp-metric-card', 'sp-grid-2', 'sp-section-index']) {
      expect(supplierAutomationSource).not.toContain(token)
    }

    expect(supplierAutomationSource).toContain('class="sp-metric-head"')
    expect(supplierAutomationSource).toContain('class="sp-metric-signal" aria-hidden="true"')
    expect(styleSource).toMatch(/\.sp-overview-item::before\s*\{[^}]*background:\s*var\(--sp-metric-accent\);/s)
    expect(styleSource).toMatch(/\.sp-metric-signal\s*\{[^}]*background:\s*var\(--sp-metric-accent\);/s)
    expect(styleSource).toMatch(/\.sp-overview-item:hover\s*\{[^}]*transform:\s*translateY\(-2px\);/s)
    expect(styleSource).toMatch(/\.sp-overview-item\.sp-neutral\s*\{[^}]*--sp-metric-accent:\s*var\(--sp-muted\);/s)
    expect(styleSource).toMatch(/\.sp-overview-item\.sp-green\s*\{[^}]*--sp-metric-accent:\s*var\(--sp-green\);/s)
    expect(styleSource).toMatch(/\.sp-overview-item\.sp-red\s*\{[^}]*--sp-metric-accent:\s*var\(--sp-red\);/s)
    expect(styleSource).toMatch(/\.sp-overview-item\.sp-blue\s*\{[^}]*--sp-metric-accent:\s*var\(--sp-blue\);/s)
    expect(styleSource).toContain(':global(.dark .sp-automation-console .sp-overview-item) {')
    expect(styleSource).toMatch(/@media \(prefers-reduced-motion: reduce\)\s*\{[^}]*\.sp-overview-item\s*\{[^}]*transition:\s*none;/s)
    expect(styleSource).not.toMatch(/\.sp-overview-item\.sp-(?:red|green|blue|amber|orange|violet)\s*\{[^}]*background:/s)
    expect(supplierAutomationSource).not.toMatch(/<table\b/i)
  })

  it('keeps history filters compact inside the toolbar without changing shared components', () => {
    expect(styleSource).toMatch(/\.sp-history-panel \.sp-panel-body\s*\{[^}]*padding:\s*0;/s)
    expect(styleSource).toMatch(/\.sp-history-toolbar \.sp-run-filters\s*\{[^}]*margin-bottom:\s*0;/s)
    expect(supplierAutomationSource).toContain('<DataTable')
    expect(supplierAutomationSource).toContain('<Pagination')
    expect(supplierAutomationSource).toContain('<BaseDialog')
  })

  it('adapts overview and detail summaries at the 1024px breakpoint', () => {
    expect(tabletBreakpoint).toBeGreaterThanOrEqual(0)
    expect(tabletStyles).toContain('grid-template-columns: repeat(2, minmax(0, 1fr));')
    expect(tabletStyles).not.toContain('.sp-overview-item:nth-child(odd) {')
    expect(tabletStyles).not.toContain('.sp-overview-item:nth-child(n + 3) {')
    expect(tabletStyles).toContain('.sp-retention-grid,\n  .sp-run-detail-summary {')

    const summaryResetIndex = tabletStyles.indexOf('.sp-summary-item,\n  .sp-summary-item:nth-child(3n + 1) {')
    const oddSummaryIndex = tabletStyles.indexOf('.sp-summary-item:nth-child(odd) {')
    expect(summaryResetIndex).toBeGreaterThanOrEqual(0)
    expect(oddSummaryIndex).toBeGreaterThan(summaryResetIndex)
    expect(tabletStyles).toMatch(
      /\.sp-summary-item,\n {2}\.sp-summary-item:nth-child\(3n \+ 1\)\s*\{[^}]*border-left:\s*1px solid var\(--sp-line\);[^}]*padding-left:\s*18px;/s
    )
    expect(tabletStyles).toMatch(
      /\.sp-summary-item:nth-child\(odd\)\s*\{[^}]*border-left:\s*0;[^}]*padding-left:\s*0;/s
    )
  })

  it('uses one 760px breakpoint for single-column 390px layouts', () => {
    expect(styleSource.match(/@media \(max-width: 760px\)/g)).toHaveLength(1)
    expect(mobileStyles).toContain('.sp-console-head,\n  .sp-panel-head,\n  .sp-detail-section-head {')
    expect(mobileStyles).toContain('align-items: stretch;')
    expect(mobileStyles).toContain('flex-direction: column;')
    expect(mobileStyles).toContain('.sp-head-actions,\n  .sp-run-filters {')
    expect(mobileStyles).toContain('width: 100%;')
    expect(mobileStyles).toContain('.sp-edit-summary,\n  .sp-form-grid,\n  .sp-retention-grid,\n  .sp-run-detail-summary,\n  .sp-provider-detail-layout,\n  .sp-cleanup-grid,\n  .sp-rate-guard-summary,\n  .sp-stage-body {')
    expect(mobileStyles).toMatch(/\.sp-overview-strip\s*\{[^}]*grid-template-columns:\s*repeat\(2,\s*minmax\(0,\s*1fr\)\);/s)
    expect(mobileStyles).toMatch(/\.sp-overview-item\s*\{[^}]*min-height:\s*0;/s)
    expect(mobileStyles).toContain('grid-template-columns: 1fr;')
    expect(mobileStyles).not.toContain('.sp-overview-item:first-child {')
    expect(mobileStyles).toMatch(/\.sp-head-actions \.sp-button\s*\{[^}]*min-width:\s*96px;/s)
    expect(mobileStyles).not.toContain('.sp-task-actions')
  })

  it('wraps both DataTable regions without adding desktop table padding', () => {
    const tableRegions = supplierAutomationSource.match(
      /class="sp-table-region sp-(?:task|history)-table-region"/g
    ) || []
    const taskPanelSource = supplierAutomationSource.match(
      /<section class="sp-console-panel sp-task-panel">[\s\S]*?<\/section>/
    )?.[0] || ''
    const historyPanelSource = supplierAutomationSource.match(
      /<section class="sp-console-panel sp-history-panel">[\s\S]*?<\/section>/
    )?.[0] || ''
    const baseTableRegionRule = styleSource.match(/\.sp-table-region\s*\{([^}]*)\}/)?.[1] || ''

    expect(tableRegions).toHaveLength(2)
    expect(taskPanelSource).toMatch(
      /<div class="sp-table-region sp-task-table-region">\s*<DataTable[\s\S]*?<\/DataTable>\s*<\/div>/
    )
    expect(historyPanelSource.indexOf('class="sp-history-toolbar"')).toBeLessThan(
      historyPanelSource.indexOf('class="sp-table-region sp-history-table-region"')
    )
    expect(historyPanelSource).toMatch(
      /<div class="sp-table-region sp-history-table-region">\s*<DataTable[\s\S]*?<\/DataTable>\s*<Pagination[\s\S]*?<\/div>/
    )
    expect(baseTableRegionRule).toContain('min-width: 0;')
    expect(baseTableRegionRule).not.toContain('padding:')
  })

  it('aligns table card spacing and task actions with the DataTable 767px breakpoint', () => {
    expect(styleSource.match(/@media \(max-width: 767px\)/g)).toHaveLength(1)
    expect(tableCardStyles).toMatch(/\.sp-table-region\s*\{[^}]*padding:\s*12px;/s)
    expect(tableCardStyles).toMatch(/\.sp-task-actions\s*\{[^}]*width:\s*100%;/s)
    expect(tableCardStyles).toMatch(
      /\.sp-task-actions \.sp-button\s*\{[^}]*flex:\s*1 1 0;[^}]*min-height:\s*40px;/s
    )
  })

  it('compiles the complete dark modal descendant selector for edit-section numbers', () => {
    expect(compiledStyle.errors).toEqual([])
    expect(styleSource).toContain(
      ':global(.dark .modal-content:has(.sp-edit-dialog) .sp-form-section-head > span) {'
    )
    expect(compiledStyle.code).toContain(
      '.dark .modal-content:has(.sp-edit-dialog) .sp-form-section-head > span'
    )
    expect(compiledStyle.code).not.toMatch(/(^|})\s*\.dark\s*\{/m)
  })
})

describe('SupplierAutomationView edit dialog composition', () => {
  const editDialogSource = supplierAutomationSource.match(
    /<BaseDialog :show="editVisible"[\s\S]*?<\/BaseDialog>/
  )?.[0] || ''
  const saveTaskSource = supplierAutomationSource.match(
    /async function saveTask\(\) \{[\s\S]*?\n\}\n\nasync function runNow/
  )?.[0] || ''

  it('keeps the wide BaseDialog and separates task identity, state, and scheduling', () => {
    expect(editDialogSource).toContain(
      '<BaseDialog :show="editVisible" :title="editingTask?.name || \'编辑任务\'" width="wide" @close="closeEdit">'
    )
    expect(editDialogSource).toContain('class="sp-edit-dialog"')
    expect(editDialogSource).toContain(":class=\"{ 'is-health-guard': editForm.task_code === 'supplier_account_health_guard' }\"")
    expect(editDialogSource).toContain('@submit.prevent="saveTask"')
    expect(editDialogSource).toContain('class="sp-form-section sp-state-section"')
    expect(editDialogSource).toContain('class="sp-form-section sp-schedule-section"')
    expect(editDialogSource).toContain('<Toggle v-model="editForm.enabled" />')
  })

  it('shows task code, enabled state, and formatted interval in the summary', () => {
    const summarySource = editDialogSource.match(
      /<section class="sp-edit-summary"[\s\S]*?<\/section>/
    )?.[0] || ''

    expect(summarySource).toContain('<span>任务编码</span><strong>{{ editForm.task_code }}</strong>')
    expect(summarySource).toContain(
      "<span>当前状态</span><strong>{{ editForm.enabled ? '已启用' : '已停用' }}</strong>"
    )
    expect(summarySource).toContain(
      '<span>当前周期</span><strong>{{ formatInterval(editForm.cron_expression) }}</strong>'
    )
  })

  it('uses independent conditional policy sections without an unconditional fallback', () => {
    const policyConditions = [...editDialogSource.matchAll(
      /<section v-if="editForm\.task_code === '(supplier_rate_guard|supplier_account_health_guard|supplier_data_cleanup)'" class="sp-form-section sp-policy-section">/g
    )].map(([, taskCode]) => taskCode)
    const policySectionCount = editDialogSource.match(
      /class="sp-form-section sp-policy-section"/g
    )?.length || 0

    expect(policyConditions).toEqual([
      'supplier_rate_guard',
      'supplier_account_health_guard',
      'supplier_data_cleanup',
    ])
    expect(policySectionCount).toBe(3)
    expect(editDialogSource).not.toMatch(
      /<section(?![^>]*v-if=)[^>]*class="sp-form-section sp-policy-section"/
    )
    expect(editDialogSource).not.toContain('<section v-else class="sp-form-section sp-policy-section">')
  })

  it('keeps every scheduling, rate guard, and retention input binding', () => {
    const bindings = [
      ['editIntervalSeconds', 'editIntervalSeconds = toNumber($event, editIntervalSeconds)'],
      ['editForm.timeout_seconds', 'editForm.timeout_seconds = toNumber($event, editForm.timeout_seconds)'],
      [
        'editForm.config.rate_guard_max_snapshot_age_seconds',
        'editForm.config.rate_guard_max_snapshot_age_seconds = toNumber($event, editForm.config.rate_guard_max_snapshot_age_seconds)',
      ],
      [
        'editForm.config.automation_run_retention_days',
        'editForm.config.automation_run_retention_days = toNumber($event, editForm.config.automation_run_retention_days)',
      ],
      [
        'editForm.config.sync_run_retention_days',
        'editForm.config.sync_run_retention_days = toNumber($event, editForm.config.sync_run_retention_days)',
      ],
      [
        'editForm.config.metric_snapshot_retention_days',
        'editForm.config.metric_snapshot_retention_days = toNumber($event, editForm.config.metric_snapshot_retention_days)',
      ],
      [
        'editForm.config.daily_stat_retention_days',
        'editForm.config.daily_stat_retention_days = toNumber($event, editForm.config.daily_stat_retention_days)',
      ],
      [
        'editForm.config.inactive_account_retention_days',
        'editForm.config.inactive_account_retention_days = toNumber($event, editForm.config.inactive_account_retention_days)',
      ],
      [
        'editForm.config.inactive_group_retention_days',
        'editForm.config.inactive_group_retention_days = toNumber($event, editForm.config.inactive_group_retention_days)',
      ],
    ]

    for (const [modelValue, updateBinding] of bindings) {
      expect(editDialogSource).toContain(`:model-value="${modelValue}"`)
      expect(editDialogSource).toContain(`@update:model-value="${updateBinding}"`)
    }
  })

  it('keeps cancel and save footer actions disabled while saving', () => {
    expect(editDialogSource).toContain('<template #footer>')
    expect(editDialogSource).toContain(
      '<button class="sp-button ghost" type="button" :disabled="Boolean(savingCode)" @click="closeEdit">取消</button>'
    )
    expect(editDialogSource).toContain(
      '<button class="sp-button primary" type="button" :disabled="Boolean(savingCode)" @click="saveTask">{{ savingCode ? \'保存中\' : \'保存任务\' }}</button>'
    )
  })

  it('keeps closeEdit and the existing saveTask validation branches and error assignments', () => {
    expect(supplierAutomationSource).toMatch(
      /function closeEdit\(\) \{\r?\n\s+editVisible\.value = false\r?\n\}/
    )
    expect(saveTaskSource).toContain('async function saveTask()')
    expect(saveTaskSource).toContain(
      'const cronExpression = intervalSecondsToCron(editIntervalSeconds.value)'
    )
    expect(saveTaskSource).toContain('if (!cronExpression) {')
    expect(saveTaskSource).toContain(
      "appStore.showError('执行间隔必须是正整数秒')"
    )
    expect(saveTaskSource).toContain("if (editForm.task_code === 'supplier_rate_guard') {")
    expect(saveTaskSource).not.toContain('rate_guard_safety_multiplier')
    expect(saveTaskSource).toContain(
      'if (editForm.config.rate_guard_max_snapshot_age_seconds < 60) {'
    )
    expect(saveTaskSource).toContain("appStore.showError('快照最大有效期不能少于 60 秒')")
    expect(saveTaskSource).toContain('editForm.cron_expression = cronExpression')
  })

  it('extends every result-dialog modal surface selector to the edit dialog', () => {
    const selectorPairs = [
      [
        ':global(.modal-content:has(.sp-edit-dialog)),',
        ':global(.modal-content:has(.sp-run-detail)) {',
      ],
      [
        ':global(.dark .modal-content:has(.sp-edit-dialog)),',
        ':global(.dark .modal-content:has(.sp-run-detail)) {',
      ],
      [
        ':global(.modal-content:has(.sp-edit-dialog) .modal-header),',
        ':global(.modal-content:has(.sp-run-detail) .modal-header) {',
      ],
      [
        ':global(.modal-content:has(.sp-edit-dialog) .modal-title),',
        ':global(.modal-content:has(.sp-run-detail) .modal-title) {',
      ],
      [
        ':global(.modal-content:has(.sp-edit-dialog) .modal-body),',
        ':global(.modal-content:has(.sp-run-detail) .modal-body) {',
      ],
      [
        ':global(.modal-content:has(.sp-edit-dialog) .modal-footer),',
        ':global(.modal-content:has(.sp-run-detail) .modal-footer) {',
      ],
    ]

    for (const [editSelector, detailSelector] of selectorPairs) {
      expect(supplierAutomationSource).toContain(`${editSelector}\n${detailSelector}`)
    }
  })
})

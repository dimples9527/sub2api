import { readFileSync } from 'node:fs'
import { dirname, resolve } from 'node:path'
import { fileURLToPath } from 'node:url'
import { describe, expect, it } from 'vitest'

const supplierAutomationSource = readFileSync(
  resolve(dirname(fileURLToPath(import.meta.url)), 'SupplierAutomationView.vue'),
  'utf-8'
)
const supplierModalSource = readFileSync(
  resolve(dirname(fileURLToPath(import.meta.url)), '../../../components/admin/supplier-management/SupplierModal.vue'),
  'utf-8'
)
const supplierManagementStyleSource = readFileSync(
  resolve(dirname(fileURLToPath(import.meta.url)), '../../../components/admin/supplier-management/supplier-management.css'),
  'utf-8'
)

describe('SupplierAutomationView edit dialog', () => {
  it('uses a switch control for the enabled state instead of a raw checkbox', () => {
    expect(supplierAutomationSource).toContain('role="switch"')
    expect(supplierAutomationSource).toContain(':aria-checked="editForm.enabled"')
    expect(supplierAutomationSource).toContain('editForm.enabled = !editForm.enabled')
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
    expect(supplierAutomationSource).toContain('providerStagesByCategory(provider)')
    expect(supplierAutomationSource).toContain('响应摘要')
  })

  it('uses light result detail cards instead of dark log panels', () => {
    expect(supplierAutomationSource).toContain('sp-response-summary')
    expect(supplierAutomationSource).not.toContain('background: rgba(15, 23, 42, 0.86)')
    expect(supplierAutomationSource).not.toContain('background: rgba(2, 6, 23, 0.34)')
    expect(supplierAutomationSource).not.toContain('linear-gradient(145deg, rgba(15, 23, 42, 0.82)')
  })

  it('shows raw response summaries in a wider dialog with a split stage layout', () => {
    expect(supplierAutomationSource).toContain('sp-stage-body')
    expect(supplierAutomationSource).toContain('sp-stage-main')
    expect(supplierAutomationSource).toContain('sp-response-panel')
    expect(supplierAutomationSource).toContain('<pre class="sp-response-summary">{{ stage.response_summary }}</pre>')
    expect(supplierAutomationSource).not.toContain('parseResponseSummaryItems')
    expect(supplierAutomationSource).not.toContain('sp-response-summary-list')
    expect(supplierAutomationSource).not.toContain('sp-response-summary-item')
    expect(supplierAutomationSource).toContain('modal-class="sp-modal-wide"')
    expect(supplierModalSource).toContain('modalClass')
    expect(supplierManagementStyleSource).toContain('.sp-modal.sp-modal-wide')
  })

  it('adds clear spacing between result detail sections', () => {
    expect(supplierAutomationSource).toContain('gap: 20px')
    expect(supplierAutomationSource).toContain('gap: 18px')
    expect(supplierAutomationSource).toContain('padding: 18px')
    expect(supplierAutomationSource).toContain('gap: 16px')
    expect(supplierAutomationSource).toContain('margin-top: 16px')
  })

  it('paginates automation run history', () => {
    expect(supplierAutomationSource).toContain('const runPage = ref(1)')
    expect(supplierAutomationSource).toContain('const runPageSize = ref(10)')
    expect(supplierAutomationSource).toContain('listRuns({ page: runPage.value, page_size: runPageSize.value })')
    expect(supplierAutomationSource).toContain('上一页')
    expect(supplierAutomationSource).toContain('下一页')
  })

  it('displays automation trigger and status values in Chinese', () => {
    expect(supplierAutomationSource).toContain('triggerText(run.trigger_source)')
    expect(supplierAutomationSource).toContain("if (trigger === 'scheduled') return '定时执行'")
    expect(supplierAutomationSource).toContain("if (trigger === 'manual') return '手动执行'")
    expect(supplierAutomationSource).toContain('statusText(run.status)')
  })
})

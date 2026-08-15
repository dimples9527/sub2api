import { readFileSync } from 'node:fs'
import { resolve } from 'node:path'
import { describe, expect, it } from 'vitest'

const source = readFileSync(
  resolve(process.cwd(), 'src/views/admin/supplier-management/SupplierBalanceAlertView.vue'),
  'utf8'
)

describe('SupplierBalanceAlertView', () => {
  it('uses the supplier layout and shared controls for the balance alert workflow', () => {
    expect(source).toContain('<SupplierModuleLayout>')
    expect(source).toContain('<DataTable')
    expect(source).toContain('<Pagination')
    expect(source).toContain('<Toggle')
    expect(source).toContain('<BaseDialog')
    expect(source).toContain('scanSupplierBalanceAlerts')
    expect(source).toContain('updateSupplierBalanceAlertConfig')
  })

  it('exposes provider configuration, event history, and scan feedback', () => {
    expect(source).toContain('供应商余额预警')
    expect(source).toContain('余额预警事件')
    expect(source).toContain('手动扫描')
    expect(source).toContain('cooldown_seconds')
    expect(source).toContain('last_scan_status')
  })

  it('provides resolved-event deletion with an active-event guard', () => {
    expect(source).toContain('deleteSupplierBalanceAlertEvent')
    expect(source).toContain('删除余额预警事件')
    expect(source).toContain("row.status === 'resolved'")
    expect(source).toContain(":disabled=\"row.status !== 'resolved' || deletingEventId !== null\"")
    expect(source).toContain('window.confirm')
    expect(source).toContain('if (!(await loadEvents())) return')
    expect(source).toContain('Math.ceil(eventTotal.value / eventPageSize.value)')
  })

  it('does not report a scan as fully refreshed when event loading fails', () => {
    expect(source).toContain('const [configResult, eventsLoaded]')
    expect(source).toContain('if (!eventsLoaded) return false')
    expect(source).toContain('if (!(await loadAll())) return')
  })

  it('applies supplier type preset colors and alert preset tones', () => {
    expect(source).toContain("sub2api: 'type-sub2api'")
    expect(source).toContain('providerNameTypeClass(row.provider_type)')
    expect(source).toContain('providerNameTypeStyle(row.provider_type)')
    expect(source).toContain('function providerNameTypeStyle')
    expect(source).toContain("function eventBalanceTone(event: SupplierBalanceAlertEvent): string")
    expect(source).toContain("return event.event_type === 'balance_recovered' ? 'sp-balance-alert-recovered' : 'sp-balance-alert-low'")
    expect(source).toContain("function thresholdTone(config: SupplierBalanceAlertConfig): string")
    expect(source).toContain("return isActiveLowConfig(config) ? 'sp-balance-alert-low' : 'sp-balance-alert-muted'")
    expect(source).toContain("function eventThresholdTone(event: SupplierBalanceAlertEvent): string")
    expect(source).toContain('.sp-balance-alert-recovered { color: var(--sp-green)')
    expect(source).toContain('.sp-balance-alert-muted { color: var(--sp-muted)')
  })

  it('keeps manual close guarded and uses the global success toast after closing', () => {
    const closeStart = source.indexOf('function closeConfigDialog')
    const saveStart = source.indexOf('async function saveConfig')
    const closeBody = source.slice(closeStart, saveStart)
    const saveEnd = source.indexOf('async function toggleConfig', saveStart)
    const saveBody = source.slice(saveStart, saveEnd)
    expect(closeBody).toContain('if (savingProviderId.value !== null) return')
    const closeAt = saveBody.indexOf('forceCloseConfigDialog()')
    const toastAt = saveBody.indexOf('appStore.showSuccess(')
    expect(closeAt).toBeGreaterThan(-1)
    expect(toastAt).toBeGreaterThan(closeAt)
    expect(source).toContain("import { useAppStore } from '@/stores/app'")
    expect(source).not.toContain('data-test="balance-alert-toast"')
  })
})

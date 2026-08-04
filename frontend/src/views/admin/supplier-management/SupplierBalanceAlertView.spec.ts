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

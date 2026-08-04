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
})

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

  it('displays automation trigger and status values in Chinese', () => {
    expect(supplierAutomationSource).toContain('triggerText(run.trigger_source)')
    expect(supplierAutomationSource).toContain("if (trigger === 'scheduled') return '定时执行'")
    expect(supplierAutomationSource).toContain("if (trigger === 'manual') return '手动执行'")
    expect(supplierAutomationSource).toContain('statusText(run.status)')
  })
})

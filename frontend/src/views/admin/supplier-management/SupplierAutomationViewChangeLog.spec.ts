import { readFileSync } from 'node:fs'
import { dirname, resolve } from 'node:path'
import { fileURLToPath } from 'node:url'
import { describe, expect, it } from 'vitest'

const supplierAutomationSource = readFileSync(
  resolve(dirname(fileURLToPath(import.meta.url)), 'SupplierAutomationView.vue'),
  'utf8'
)

describe('SupplierAutomationView 倍率守护变更日志', () => {
  it('提供带未处理数量提示的日志入口和六列表格', () => {
    expect(supplierAutomationSource).toContain('变更日志')
    expect(supplierAutomationSource).toContain('pendingRateGuardChangeLogCount')
    expect(supplierAutomationSource).toContain('rateGuardChangeLogColumns')
    expect(supplierAutomationSource).toContain("{ key: 'status', label: '处理状态'")
    expect(supplierAutomationSource).toContain("{ key: 'local_group_name', label: '本地分组'")
    expect(supplierAutomationSource).toContain("{ key: 'upstream_group_name', label: '上游分组'")
    expect(supplierAutomationSource).toContain("{ key: 'old_rate', label: '原倍率'")
    expect(supplierAutomationSource).toContain("{ key: 'new_rate', label: '新倍率'")
    expect(supplierAutomationSource).toContain("{ key: 'changed_at', label: '修改时间'")
  })

  it('允许将待处理记录逐条确认为已处理并刷新数量', () => {
    expect(supplierAutomationSource).toContain('markRateGuardChangeLogHandled')
    expect(supplierAutomationSource).toContain("changeLog.status === 'pending'")
    expect(supplierAutomationSource).toContain('await markRateGuardChangeLogHandled(changeLog.id)')
    expect(supplierAutomationSource).toContain('await loadRateGuardChangeLogs()')
  })
})

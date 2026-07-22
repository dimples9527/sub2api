import { readFileSync } from 'node:fs'
import { resolve } from 'node:path'
import { describe, expect, it } from 'vitest'

const apiSource = readFileSync(resolve(process.cwd(), 'src/api/admin/supplierAutomation.ts'), 'utf8')

describe('supplierAutomation 倍率守护变更日志接口', () => {
  it('提供日志列表和确认处理请求', () => {
    expect(apiSource).toContain('export interface SupplierRateGuardChangeLog')
    expect(apiSource).toContain("'/admin/supplier-management/automation/rate-guard-change-logs'")
    expect(apiSource).toContain('function listRateGuardChangeLogs')
    expect(apiSource).toContain('function markRateGuardChangeLogHandled')
    expect(apiSource).toContain('rate-guard-change-logs/${id}/handled')
  })
})

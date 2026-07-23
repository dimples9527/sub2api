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

  it('支持账号倍率守护运行模式和独立解绑日志', () => {
    expect(apiSource).toContain("export type SupplierAccountRateGuardRunMode = 'preview' | 'execute'")
    expect(apiSource).toContain('mode: SupplierAccountRateGuardRunMode = \'execute\'')
    expect(apiSource).toContain('{ mode }')
    expect(apiSource).toContain('export interface SupplierAccountRateGuardUnbindLog')
    expect(apiSource).toContain('function listAccountRateGuardUnbindLogs')
    expect(apiSource).toContain("'/admin/supplier-management/automation/account-rate-guard-unbind-logs'")
  })
})

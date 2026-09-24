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
    expect(apiSource).toContain('function markAccountRateGuardUnbindLogHandled')
    expect(apiSource).toContain('account-rate-guard-unbind-logs/${id}/handled')
    expect(apiSource).toContain('only_unbound?: boolean')
    expect(apiSource).toContain("status: 'pending' | 'handled'")
  })

  it('批次多选以逗号串传参，不用 axios 的数组默认序列化', () => {
    // axios 默认把数组序列化成 `run_ids[]=1&run_ids[]=2`，后端得用 c.QueryArray("run_ids[]")
    // 才取得到 —— 又脆又丑。这里统一 join 成逗号串，与同模块的 account-health/trends 一致。
    expect(apiSource).toContain('const { run_ids: runIDs, ...rest } = params')
    expect(apiSource).toContain("query.run_ids = runIDs.join(',')")
    // 空数组不能传：后端会把它拼成 IN () 这种语法错。
    expect(apiSource).toContain('if (runIDs && runIDs.length > 0)')
    expect(apiSource).toContain('run_ids?: number[]')
  })
})

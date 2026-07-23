import { readFileSync } from 'node:fs'
import { dirname, resolve } from 'node:path'
import { fileURLToPath } from 'node:url'
import { describe, expect, it } from 'vitest'

const currentDirectory = dirname(fileURLToPath(import.meta.url))
const supplierAutomationSource = readFileSync(
  resolve(currentDirectory, 'SupplierAutomationView.vue'),
  'utf8'
)
const supplierGroupsSource = readFileSync(
  resolve(currentDirectory, 'SupplierGroupsView.vue'),
  'utf8'
)
const repositoryRulesSource = readFileSync(
  resolve(currentDirectory, '../../../../../AGENTS.md'),
  'utf8'
)

describe('SupplierGroupsView 分组倍率变更日志', () => {
  it('在分组管理页面提供带未处理数量提示的日志入口和六列表格', () => {
    expect(supplierGroupsSource).toContain('分组倍率变更日志')
    expect(supplierGroupsSource).toContain('pendingRateGuardChangeLogCount')
    expect(supplierGroupsSource).toContain('rateGuardChangeLogColumns')
    expect(supplierGroupsSource).toContain("{ key: 'status', label: '处理状态'")
    expect(supplierGroupsSource).toContain("{ key: 'local_group_name', label: '本地分组'")
    expect(supplierGroupsSource).toContain("{ key: 'upstream_group_name', label: '上游分组'")
    expect(supplierGroupsSource).toContain("{ key: 'old_rate', label: '原倍率'")
    expect(supplierGroupsSource).toContain("{ key: 'new_rate', label: '新倍率'")
    expect(supplierGroupsSource).toContain("{ key: 'changed_at', label: '修改时间'")
  })

  it('允许将待处理记录逐条确认为已处理并刷新数量', () => {
    expect(supplierGroupsSource).toContain('markRateGuardChangeLogHandled')
    expect(supplierGroupsSource).toContain("changeLog.status === 'pending'")
    expect(supplierGroupsSource).toContain('await markRateGuardChangeLogHandled(changeLog.id)')
    expect(supplierGroupsSource).toContain('await loadRateGuardChangeLogs()')
  })

  it('精简弹窗重复说明并为列表关键信息提供颜色层级', () => {
    expect(supplierGroupsSource).not.toContain('Rate Guard Todo')
    expect(supplierGroupsSource).not.toContain('倍率变更待办')
    expect(supplierGroupsSource).not.toContain('调高本地分组倍率后自动生成记录')
    expect(supplierGroupsSource).toContain('sp-change-log-local-group')
    expect(supplierGroupsSource).toContain('sp-change-log-upstream-group')
    expect(supplierGroupsSource).toContain('sp-change-log-old-rate')
    expect(supplierGroupsSource).toContain('sp-change-log-new-rate')
    expect(supplierGroupsSource).toContain('sp-change-log-time')
    expect(supplierGroupsSource).toContain('--sp-change-log-local: #0891b2')
    expect(supplierGroupsSource).toContain('--sp-change-log-old-rate: #d97706')
    expect(supplierGroupsSource).toContain('--sp-change-log-new-rate: #16a34a')
    expect(supplierGroupsSource).toContain('--sp-change-log-time: #2563eb')
  })

  it('将上游分组标识拼接在名称后而不是另起一行', () => {
    expect(supplierGroupsSource).toContain('class="sp-change-log-key">#{{ changeLog.upstream_group_key }}</span>')
    expect(supplierGroupsSource).not.toContain('class="sp-sub sp-change-log-key"')
  })

  it('从自动化页面移除重复入口，并区分分组页顶部操作按钮颜色', () => {
    expect(supplierAutomationSource).not.toContain('openRateGuardChangeLogs')
    expect(supplierAutomationSource).not.toContain('rateGuardChangeLogsVisible')
    expect(supplierAutomationSource).not.toContain('倍率守护变更日志')
    expect(supplierGroupsSource).toContain('sp-control-button-reset')
    expect(supplierGroupsSource).toContain('sp-control-button-match')
    expect(supplierGroupsSource).toContain('sp-control-button-log')
    expect(supplierGroupsSource).toContain('sp-control-button-refresh')
  })

  it('仓库规则禁止在已有标题的供应商弹窗左上角重复显示标题说明', () => {
    expect(repositoryRulesSource).toContain('## 供应商模块弹窗内容')
    expect(repositoryRulesSource).toContain('弹窗内容区左上角不得重复显示英文眉题、二级标题或说明文案')
  })
})

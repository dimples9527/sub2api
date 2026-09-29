import { readFileSync } from 'node:fs'
import { dirname, resolve } from 'node:path'
import { fileURLToPath } from 'node:url'
import { describe, expect, it } from 'vitest'

const currentDirectory = dirname(fileURLToPath(import.meta.url))
const source = readFileSync(resolve(currentDirectory, 'SupplierAccountsView.vue'), 'utf8')
// 本功能的 API 归供应商模块；框架的账号 API 只作为「不该被改」的对照。
const supplierApiSource = readFileSync(
  resolve(currentDirectory, '../../../api/admin/supplierProviderData.ts'),
  'utf8'
)
const frameworkAccountsApiSource = readFileSync(
  resolve(currentDirectory, '../../../api/admin/accounts.ts'),
  'utf8'
)

describe('SupplierAccountsView 批量同步上游模型', () => {
  it('工具栏提供独立入口，并把可同步数量写在按钮上', () => {
    expect(source).toContain('data-test="supplier-account-sync-upstream-models"')
    expect(source).toContain('data-test="supplier-account-sync-upstream-models-count"')
    // 数量取自与提交同源的 syncModelsTargets，不让用户点开弹窗才发现要同步几个。
    expect(source).toContain('>{{ syncModelsTargets.length }}</span>')
    expect(source).toContain(':disabled="syncModelsTargets.length === 0 || syncModelsSubmitting"')
  })

  it('只同步匹配到本地账号的勾选行，未匹配的不混进请求', () => {
    // 批量接口要的是本地 account ID；上游账号表里「未匹配 / 匹配冲突」的行拿不到。
    expect(source).toContain('const syncModelsTargets = computed(() => bindableLocalAccountIDs(selectedBindableAccounts.value))')
    expect(source).toContain('data-test="supplier-account-sync-upstream-models-skip-note"')
    expect(source).toContain('个未匹配账号会被跳过')
  })

  it('两个模式的叫法直白，不用「覆盖」这种读不出作用范围的词', () => {
    // 「用上游列表覆盖」会被理解成覆盖整份映射（其实只动白名单），
    // 所以统一改成「保留现有并增加新的」/「去除现有并增加新的」。
    expect(source).toContain("{ value: 'merge', label: '保留现有并增加新的' }")
    expect(source).toContain("{ value: 'replace', label: '去除现有并增加新的' }")
    const optionsBlock = source.match(/const upstreamSyncModeOptions: SelectOption\[\] = \[([\s\S]*?)\n\]/)?.[1] || ''
    expect(optionsBlock).not.toContain('覆盖')
    // 按钮与确认框的动词也要跟着走，别只改下拉
    expect(source).toContain("syncModelsMode === 'replace' ? '去除并增加' : '保留并增加'")
    expect(source).toContain("syncModelsMode.value === 'replace' ? '去除现有白名单并增加新的' : '保留现有并增加新的'")
  })

  it('写入方式用框架 Select，不引入原生表单控件', () => {
    // 本页有条既有的产品决策守卫禁止 <input>：模式选择必须继续走 Select。
    expect(source).toContain('const upstreamSyncModeOptions: SelectOption[]')
    expect(source).toContain('v-model="syncModelsMode"')
    const dialogBlock = source.match(/<div class="sp-account-sync-dialog"[\s\S]*?<\/div>\s*<template #footer>/)?.[0] || ''
    expect(dialogBlock).not.toContain('<input')
    expect(dialogBlock).not.toContain('<table')
  })

  it('提供「预览」与「应用」两个入口，apply 参数分别传 false / true', () => {
    expect(source).toContain('data-test="supplier-account-sync-upstream-models-preview"')
    expect(source).toContain('data-test="supplier-account-sync-upstream-models-apply"')
    expect(source).toContain('async function runSyncUpstreamModels(apply: boolean)')
    expect(source).toContain('@click="runSyncUpstreamModels(false)"')
    expect(source).toContain('@click="runSyncUpstreamModels(true)"')
    expect(source).toContain('apply,')
  })

  it('「去除现有并增加新的」是不可逆写入，多于一个账号时必须二次确认', () => {
    const runBlock = source.match(/async function runSyncUpstreamModels\(apply: boolean\) \{([\s\S]*?)\n\}/)?.[1] || ''
    expect(runBlock).toContain('window.confirm')
    // 单个账号的操作认知成本低，不打断；预览不写入，也不该弹确认。
    expect(runBlock).toContain('syncModelsTargets.value.length > 1')
    expect(runBlock).toContain('apply &&')
  })

  it('明示手写映射不会被改动：两种模式都只作用于白名单条目', () => {
    // 白名单与别名映射在存储层是同一个 model_mapping，界面上必须把这条边界讲清楚，
    // 否则用户会以为写入方式会把他的手写映射一起清掉。
    expect(source).toContain('手写的别名映射与通配符规则不受影响，两种模式都原样保留。')
    expect(source).toContain('手写别名映射保留')
    // 结果里如实回显保留了多少条手写映射，让用户能确认自己的映射还在
    expect(source).toContain('item.custom_mapping_kept')
    expect(source).toContain('保留手写映射')
  })

  it('结果逐条列出成功与失败账号，而不是只弹一句总数', () => {
    expect(source).toContain('data-test="supplier-account-sync-upstream-models-result"')
    expect(source).toContain('`supplier-account-sync-upstream-models-item-${item.account_id}`')
    expect(source).toContain("item.status === 'success' ? 'ok' : 'failed'")
    expect(source).toContain('item.error_message')
    expect(source).toContain('appStore.showError(')
  })

  it('弹窗自行声明 --sp-* 兜底变量（Teleport 后拿不到页面根节点变量）', () => {
    expect(source).toContain(':global(.modal-content:has(.sp-account-sync-dialog))')
    expect(source).toContain(':global(.dark .modal-content:has(.sp-account-sync-dialog))')
    const block = source.match(/:global\(\.modal-content:has\(\.sp-account-sync-dialog\)\) \{([\s\S]*?)\n\}/)?.[1] || ''
    for (const token of ['--sp-panel:', '--sp-panel-2:', '--sp-line:', '--sp-text:', '--sp-muted:']) {
      expect(block).toContain(token)
    }
  })

  it('与「测试当前筛选」同族共用蓝色，不另造色号', () => {
    // 工具栏配色已排满（7 个 --sp-* 全占用），新增按钮按既有做法走「同族共用色」。
    expect(source).toContain('.sp-account-toolbar-sync-models')
    const rule = source.match(/\.sp-account-toolbar-sync-models \{([\s\S]*?)\n\}/)?.[1] || ''
    expect(rule).toContain('var(--sp-blue)')
    expect(rule).not.toContain('--sp-cyan')
  })

  it('接口契约：批量同步支持 merge / replace 与预览开关', () => {
    expect(supplierApiSource).toContain('export async function syncUpstreamModelsBatch')
    expect(supplierApiSource).toContain(
      "'/admin/supplier-management/accounts/models/sync-upstream/batch'"
    )
    expect(supplierApiSource).toContain("export type UpstreamModelBatchSyncMode = 'merge' | 'replace'")
  })

  it('接口与类型落在供应商模块，框架的账号 API 一行不改', () => {
    // AGENTS.md：供应商管理相关的 API 属非框架业务代码，不该写进框架的账号 API。
    // 页面也不该再从 accounts.ts 取这批类型（否则又把它耦合回框架）。
    expect(frameworkAccountsApiSource).not.toContain('syncUpstreamModelsBatch')
    expect(source).toContain('type UpstreamModelBatchSyncItem,')
    expect(source).not.toContain("} from '@/api/admin/accounts'")
  })
})

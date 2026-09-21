import { readFileSync } from 'node:fs'
import { dirname, resolve } from 'node:path'
import { fileURLToPath } from 'node:url'
import { describe, expect, it } from 'vitest'

const currentDirectory = dirname(fileURLToPath(import.meta.url))
const source = readFileSync(resolve(currentDirectory, 'SupplierAccountsView.vue'), 'utf8')
const apiSource = readFileSync(
  resolve(currentDirectory, '../../../api/admin/supplierProviderData.ts'),
  'utf8'
)

describe('SupplierAccountsView 批量绑定分组', () => {
  it('用框架 DataTable 的多选能力，不自己画勾选列', () => {
    expect(source).toContain('selectable')
    expect(source).toContain(':selected-keys="selectedAccountKeys"')
    expect(source).toContain('@selection-change="selectedAccountKeys = $event"')
    // 页面必须继续只用框架控件：自己写 checkbox 会绕过 DataTable 的分页/虚拟化选择语义。
    expect(source).not.toContain('<input')
    expect(source).not.toContain('<table')
  })

  it('提供「勾选账号后绑定分组」和「先选分组再选账号」两个独立入口', () => {
    expect(source).toContain('data-test="supplier-account-batch-bind-groups"')
    expect(source).toContain('data-test="supplier-account-batch-bind-count"')
    expect(source).toContain('data-test="supplier-account-bind-by-group"')
    expect(source).toContain('data-test="supplier-account-batch-bind-submit"')
    expect(source).toContain('data-test="supplier-account-bind-by-group-submit"')
  })

  it('未匹配 / 匹配冲突的账号不参与绑定，并如实告知跳过数量', () => {
    expect(source).toContain('selectedBindableAccounts')
    expect(source).toContain('selectedUnbindableAccountCount')
    expect(source).toContain('data-test="supplier-account-batch-bind-skip-note"')
    expect(source).toContain('manageableLocalAccountID(account) !== null')
  })

  it('「将被跳过」的账号逐条列出，而不是只给一个数字', () => {
    // 只报数量的话用户不知道该去处理哪一条上游账号。
    // 产品决定是不阻断勾选，那就必须在提交前把「跳过的到底是谁、为什么」讲清楚。
    expect(source).toContain('const selectedUnbindableAccounts = computed')
    expect(source).toContain('v-for="account in selectedUnbindableAccounts"')
    expect(source).toContain('`supplier-account-batch-bind-skip-${account.id}`')
    expect(source).toContain('unbindableAccountReason')
  })

  it('匹配状态徽标与勾选框无障碍标签都点明「无法绑定分组」', () => {
    expect(source).toContain('const UNBINDABLE_ACCOUNT_HINT =')
    expect(source).toContain(':title="UNBINDABLE_ACCOUNT_HINT"')
    // 读屏用户只听到「选择 X」会以为这行能正常绑定分组
    expect(source).toContain('return `选择 ${name}（${UNBINDABLE_ACCOUNT_HINT}）`')
    // 按钮 title 也要预告会被跳过的数量，让人在打开弹窗前就知道
    expect(source).toContain('个未匹配账号会被跳过')
  })

  it('两个入口共用同一套提交逻辑，避免语义分叉', () => {
    expect(source).toContain('async function runBatchBindGroups(accountIDs: number[], groupIDs: number[])')
    expect(source).toContain('await runBatchBindGroups(')
    expect(source).toContain('await batchBindSupplierAccountGroups({')
    expect(source).toContain('account_ids: accountIDs')
    expect(source).toContain('group_ids: groupIDs')
  })

  it('有失败时单独列出失败账号，而不是只弹一句总数', () => {
    expect(source).toContain('batchBindFailures')
    expect(source).toContain('data-test="supplier-account-batch-bind-result"')
    expect(source).toContain('if (batchBindFailures.value.length > 0) showBatchBindResultDialog.value = true')
  })

  it('弹窗自行声明 --sp-* 兜底变量（Teleport 后拿不到页面根节点变量）', () => {
    expect(source).toContain(':global(.modal-content:has(.sp-batch-bind-dialog))')
    expect(source).toContain(':global(.modal-content:has(.sp-bind-by-group-dialog))')
    expect(source).toContain(':global(.dark .modal-content:has(.sp-batch-bind-dialog))')
  })

  it('筛选或排序变化时清空勾选，避免「已选 N 个」里混进看不见的行', () => {
    const resetBlock = source.match(/function resetPageAndLoad\(\) \{([\s\S]*?)\n\}/)?.[1] || ''
    expect(resetBlock).toContain('selectedAccountKeys.value = []')
    const sortBlock = source.match(/function handleAccountSort\([^)]*\) \{([\s\S]*?)\n\}/)?.[1] || ''
    expect(sortBlock).toContain('selectedAccountKeys.value = []')
  })

  it('接口契约：追加绑定语义与逐账号结果', () => {
    expect(apiSource).toContain('export async function batchBindSupplierAccountGroups')
    expect(apiSource).toContain("'/admin/supplier-management/accounts/batch-bind-groups'")
    expect(apiSource).toContain("export type SupplierAccountGroupBindStatus = 'bound' | 'unchanged' | 'failed'")
    expect(apiSource).toContain('batchBindSupplierAccountGroups,')
  })

  it('「按分组绑定账号」弹窗提供供应商筛选，复用列表页同一份供应商选项', () => {
    expect(source).toContain('v-model="bindByGroupProviderFilter"')
    // 不另造一份选项：与顶部主列表筛选共用 providerOptions，避免两处口径漂移
    expect(source).toContain(':options="providerOptions"')
    expect(source).toContain('const bindByGroupProviderFilter = ref(0)')
  })

  it('供应商筛选进入请求参数（服务端筛选），并在打开弹窗时重置', () => {
    expect(source).toContain('provider_id: bindByGroupProviderFilter.value || undefined')
    // 变化即重新拉取候选账号，与平台筛选走同一条路径
    expect(source).toContain('watch([bindByGroupPlatformFilter, bindByGroupProviderFilter]')
    // 打开弹窗必须回到「全部供应商」，否则上次的筛选会静默残留
    const prepareBlock = source.match(/async function prepareBindByGroupDialog\(\) \{([\s\S]*?)\n\}/)?.[1] || ''
    expect(prepareBlock).toContain('bindByGroupProviderFilter.value = 0')
  })

  it('接口契约：可绑定账号列表支持按供应商过滤', () => {
    expect(apiSource).toContain('provider_id?: number')
  })

  it('候选账号列表丢弃过期响应，避免并发筛选互相覆盖', () => {
    // 搜索防抖、平台下拉、供应商下拉都能触发重新加载；没有序号保护时，
    // 先发后到的旧响应会把新筛选的结果盖掉（下拉显示已筛、列表却是全量）。
    expect(source).toContain('let bindByGroupLoadToken = 0')
    expect(source).toContain('const loadToken = ++bindByGroupLoadToken')
    expect(source).toContain('if (loadToken !== bindByGroupLoadToken) return')
    // loading 只允许最新一次请求关掉，否则先返回的会让列表提前脱离加载态
    expect(source).toContain('if (loadToken === bindByGroupLoadToken) bindByGroupAccountsLoading.value = false')
    // 关闭弹窗让在途请求作废，避免下次打开闪出上一次筛选的列表
    const closeBlock = source.match(/function closeBindByGroupDialog\(\) \{([\s\S]*?)\n\}/)?.[1] || ''
    expect(closeBlock).toContain('bindByGroupLoadToken += 1')
  })
})

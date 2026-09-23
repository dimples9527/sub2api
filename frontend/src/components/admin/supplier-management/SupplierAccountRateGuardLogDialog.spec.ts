import { readFileSync } from 'node:fs'
import { resolve } from 'node:path'
import { describe, expect, it } from 'vitest'

const source = readFileSync(
  resolve(process.cwd(), 'src/components/admin/supplier-management/SupplierAccountRateGuardLogDialog.vue'),
  'utf8'
)

describe('SupplierAccountRateGuardLogDialog', () => {
  it('默认只显示已解绑记录并支持切换全部记录', () => {
    expect(source).toContain("result: 'unbound'")
    expect(source).toContain("showingAllRecords ? '仅看已解绑' : '显示所有记录'")
    expect(source).toContain('async function toggleAllRecords()')
    expect(source).toContain('filters.result = showingAllRecords.value ? \'unbound\' : \'\'')
    expect(source).toContain("filters.result = 'unbound'")
  })

  it('提供处理状态和确认处理能力', () => {
    expect(source).toContain("{ value: 'pending', label: '待处理' }")
    expect(source).toContain('markAccountRateGuardUnbindLogHandled(log.id)')
    expect(source).toContain("log.status === 'pending'")
  })

  it('使用更宽弹窗并在桌面端隐藏表格横向滚动', () => {
    expect(source).toContain('width="full"')
    expect(source).toContain(':global(.modal-content:has(.account-rate-log-dialog))')
    // 1520px 是实测下限：降到 1360 时长账号名与错误信息会被折成 4 行、单行高度翻倍，
    // 反而比留白更难扫读。这条断言防止以后有人觉得"太宽"就随手改小。
    expect(source).toContain('width: min(1520px, calc(100vw - 32px))')
    expect(source).toContain('overflow-x: hidden')
    expect(source).toContain('table-layout: fixed')
    expect(source).toContain('white-space: normal')
    expect(source).toContain(':sticky-first-column="false"')
  })

  it('直接展示筛选和列表，不重复弹窗标题', () => {
    expect(source).toContain('title="账号倍率守护解除绑定日志"')
    expect(source).not.toContain('Account Rate Guard Audit')
    expect(source).not.toContain('账号与分组解绑轨迹')
  })

  it('使用平台色区分状态、实体和倍率信息', () => {
    expect(source).toContain('account-rate-log-status-pending')
    expect(source).toContain('account-rate-log-status-handled')
    expect(source).toContain('account-rate-log-provider')
    expect(source).toContain('account-rate-log-account')
    expect(source).toContain('account-rate-log-group')
    expect(source).toContain('account-rate-log-scheduling')
    expect(source).toContain('platformBadgeLightClass(log.platform)')
    expect(source).toContain('platformBorderClass(log.platform)')
    expect(source).toContain("from '@/utils/platformColors'")
    expect(source).toContain('account-rate-log-rate-upstream')
    expect(source).toContain('account-rate-log-rate-local')
    expect(source).toContain("`account-rate-log-result-${log.result}`")
    expect(source).toContain('color-mix(in srgb, var(--sp-green)')
    expect(source).toContain('color-mix(in srgb, var(--sp-blue)')
    expect(source).toContain('color-mix(in srgb, var(--sp-violet)')
    // 暗色覆盖必须写普通的 `.dark X`：scoped 块里的 `:global(.dark) X` 会被编译成裸 `.dark`，
    // 后代选择器和 data-v 属性一起丢 ⇒ 暗色静默失效（编译、运行都不报错）。
    // 编译产物可自证：`.dark .foo` → `.dark .foo[data-v-xxx]`。
    expect(source).toContain('.dark .account-rate-log-dialog')
    expect(source.replace(/\/\*[\s\S]*?\*\//g, '')).not.toContain(':global(.dark)')
  })

  it('将上游和本地分组倍率横向展示', () => {
    expect(source).toContain('class="account-rate-log-rate-compare"')
    expect(source).toContain('grid-template-columns: minmax(0, 1fr) auto minmax(0, 1fr)')
    expect(source).toContain('class="account-rate-log-rate-divider"')
  })

  it('为日志表格提供斑马纹和悬停高亮', () => {
    expect(source).toContain('tbody tr:nth-child(even)')
    expect(source).toContain('tbody tr:hover')
  })

  // 一键处理清的是「当前筛选条件下的全部待处理」。
  // 这里的每一条断言都对应一个会静默出错的点：筛选漏传会打到范围外、
  // 不做二次确认会误清几百条、无待处理时按钮可点会白跑一次、并发会互相覆盖列表。
  it('提供一键处理，且与列表共用同一套筛选口径', () => {
    expect(source).toContain('markAccountRateGuardUnbindLogsHandled')
    expect(source).toContain('async function markAllPendingHandled()')
    // 列表与批量走同一个参数构造器 —— 分成两份写迟早漂移。
    expect(source).toContain('function buildListParams(withPagination: boolean)')
    expect(source).toContain('listAccountRateGuardUnbindLogs(buildListParams(true))')
    expect(source).toContain('markAccountRateGuardUnbindLogsHandled(buildListParams(false))')
    // 批量不该透传分页参数。
    expect(source).toContain('...(withPagination ? { page: page.value, page_size: pageSize.value } : {})')
  })

  it('一键处理前二次确认条数，且只在有待处理时可用', () => {
    expect(source).toContain('window.confirm(')
    expect(source).toContain('条待处理记录全部标记为已处理？此操作不可撤销。')
    // 确认文案里的条数必须取自 pendingCount（后端用同一套 where 统计出来的），
    // 不能另算一份 —— 否则"按钮上显示 30、确认框说 12"。
    expect(source).toContain('const target = pendingCount.value')
    expect(source).toContain(':disabled="operationBusy || pendingCount <= 0"')
    expect(source).toContain(":title=\"pendingCount > 0 ? `处理当前筛选下的 ${pendingCount} 条待处理记录` : '当前筛选下没有待处理记录'\"")
  })

  it('一键处理按后端单批上限循环提交，并给出成功/失败反馈', () => {
    // has_more 由后端返回，前端不硬编码上限值 —— 后端调批次大小不影响这里。
    expect(source).toContain('hasMore = result.has_more && result.handled > 0')
    expect(source).toContain('while (hasMore && guard < 40)')
    // AGENTS.md：业务成功/失败必须走全局 Toast，不能自建提示条。
    expect(source).toContain('appStore.showSuccess(`已标记 ${handled} 条待处理记录为已处理`)')
    expect(source).toContain("appStore.showError(err instanceof Error ? err.message : '一键处理失败')")
    expect(source).toContain("import { useAppStore } from '@/stores/app'")
  })

  it('单条处理与一键处理互斥，避免并发刷新互相覆盖', () => {
    expect(source).toContain('const operationBusy = computed(() => loading.value || batchHandling.value || handlingID.value > 0)')
    expect(source).toContain("if (log.status !== 'pending' || operationBusy.value) return")
    expect(source).toContain('if (batchHandling.value || pendingCount.value <= 0) return')
  })

  it('把一键处理与普通筛选按钮在视觉上区分开', () => {
    expect(source).toContain('account-rate-log-batch-action')
    // 琥珀与表格里"待处理"同色，语义一致。
    expect(source).toContain('border: 1px solid color-mix(in srgb, var(--sp-amber) 40%, var(--sp-line))')
    expect(source).toContain('.account-rate-log-batch-action:disabled')
  })
})

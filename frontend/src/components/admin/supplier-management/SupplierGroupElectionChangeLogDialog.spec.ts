import { readFileSync } from 'node:fs'
import { resolve } from 'node:path'
import { describe, expect, it } from 'vitest'

const source = readFileSync(
  resolve(process.cwd(), 'src/components/admin/supplier-management/SupplierGroupElectionChangeLogDialog.vue'),
  'utf8'
)
const groupsSource = readFileSync(
  resolve(process.cwd(), 'src/views/admin/supplier-management/SupplierGroupsView.vue'),
  'utf8'
)
const automationSource = readFileSync(
  resolve(process.cwd(), 'src/views/admin/supplier-management/SupplierAutomationView.vue'),
  'utf8'
)
const accountsSource = readFileSync(
  resolve(process.cwd(), 'src/views/admin/supplier-management/SupplierAccountsView.vue'),
  'utf8'
)

/** 取出某个 scoped 规则块的声明体，用来断言「这块样式里到底有没有某条属性」。 */
function cssBlock(selector: string): string {
  const escaped = selector.replace(/[.*+?^${}()|[\]\\]/g, '\\$&')
  const match = source.match(new RegExp(`${escaped}\\s*\\{([^}]*)\\}`))
  return match ? match[1] : ''
}

describe('SupplierGroupElectionChangeLogDialog', () => {
  it('说明文案写清「只显示开关被拨动的记录」', () => {
    expect(source).toContain('只显示调度开关真的被拨动的记录')
    expect(source).toContain('未变更、未测试、写库失败的记录不在这里')
  })

  it('行键用 run_id + account_id 复合键', () => {
    // 同一次运行里多个账号可能同时被拨动，只按 run_id 会撞键：
    // 表现是翻页/筛选后某几行内容错乱，而且不报错。
    expect(source).toContain('function rowKey(log: SupplierGroupElectionChangeLog)')
    expect(source).toContain('${log.run_id}-${log.account_id}')
    expect(source).toContain(':row-key="rowKey"')
  })

  it('方向筛选传后端枚举，全部时不传', () => {
    expect(source).toContain("filters.value.direction === 'all' ? undefined : filters.value.direction")
    expect(source).toContain("{ value: 'disabled', label: '只看关闭' }")
    expect(source).toContain("{ value: 'enabled', label: '只看开启' }")
    // 非法值会被后端当「不过滤」静默吞掉，所以类型必须收成字面量联合。
    expect(source).toContain("direction: 'all' | 'enabled' | 'disabled'")
  })

  it('时间范围与分页都交给后端', () => {
    expect(source).toContain('started_from: filters.value.startedFrom || undefined')
    expect(source).toContain('started_to: filters.value.startedTo || undefined')
    expect(source).toContain('page: page.value')
    expect(source).toContain('page_size: pageSize.value')
    expect(source).toContain('@update:page="changePage"')
  })

  it('重置筛选保留分组锁定，换分组要走查看全部', () => {
    // 用户是冲着某个分组点开弹窗的，重置时把分组一起清掉会让人误以为看到的是全局日志。
    expect(source).toContain("filters.value = { direction: 'all', search: '', startedFrom: '', startedTo: '' }")
    expect(source).toContain('function clearGroup()')
    expect(source).toContain('clearedGroup.value = true')
    expect(source).toContain('const activeGroupID = computed(() => (clearedGroup.value ? null : props.groupId ?? null))')
    expect(source).toContain('title="查看全部分组"')
  })

  it('空态区分「还没发生过切换」与「筛选太窄」', () => {
    // 用同一句话会让人以为功能坏了，直接去翻后端。
    expect(source).toContain("hasActiveFilters ? '当前筛选条件下没有调度切换记录，试试放宽时间范围或换个方向。' : '最近还没有发生调度切换。'")
  })

  it('平台色交给工具类，样式块里不写 color', () => {
    expect(source).toContain("from '@/utils/platformColors'")
    expect(source).toContain(':class="platformTextClass(log.platform)"')
    // 这条规则块一旦写了 color，就与 Tailwind 工具类同权重（0,1,0），
    // 由构建顺序决定谁赢，平台色会被静默压掉。
    expect(cssBlock('.sp-election-log-platform')).not.toContain('color')
  })

  it('弹窗自带深色变量（BaseDialog 会 Teleport 到 body）', () => {
    expect(source).toContain('--sp-election-log-accent: #ea580c')
    expect(source).toContain(':global(.dark) .sp-election-log-dialog')
    expect(cssBlock('.sp-election-log-dialog')).toContain('--sp-election-log-accent')
  })

  it('支持锁定到单个账号，语义与分组锁定一致', () => {
    expect(source).toContain('account_id: activeAccountID.value ?? undefined')
    expect(source).toContain('const activeAccountID = computed(() => (clearedAccount.value ? null : props.accountId ?? null))')
    expect(source).toContain('function clearAccount()')
    expect(source).toContain('clearedAccount.value = false')
    expect(source).toContain('title="查看全部账号"')
    // 重置不能把账号锁定一起清掉（同分组锁定的理由）
    expect(source).toContain('|| activeAccountID.value !== null')
  })
})

describe('调度切换日志的两个入口', () => {
  it('分组管理页：工具栏看全部，行菜单锁定本分组', () => {
    expect(groupsSource).toContain('调度切换日志')
    expect(groupsSource).toContain('本分组调度切换')
    expect(groupsSource).toContain('openElectionChangeLogs(group.local_group_id ?? null, group.local_group_name || \'\')')
    expect(groupsSource).toContain('<SupplierGroupElectionChangeLogDialog')
    expect(groupsSource).toContain(':group-id="electionChangeLogGroupID"')
    expect(groupsSource).toContain(':group-label="electionChangeLogGroupLabel"')
    expect(groupsSource).toContain('@close="closeElectionChangeLogs"')
  })

  it('自动化任务中心：全局视角，不锁定分组', () => {
    expect(automationSource).toContain('data-test="open-election-change-logs"')
    expect(automationSource).toContain('调度切换日志')
    expect(automationSource).toContain('<SupplierGroupElectionChangeLogDialog')
    // 不带 group-id = 看全部分组；带上了就要有对应 prop，否则点击后永远只显示一个分组。
    expect(automationSource).not.toContain(':group-id=')
    expect(automationSource).not.toContain(':account-id=')
  })

  it('上游账号页：工具栏看全部，行内锁定本账号', () => {
    expect(accountsSource).toContain('data-test="supplier-account-election-change-logs"')
    expect(accountsSource).toContain('本账号调度切换')
    expect(accountsSource).toContain('<SupplierGroupElectionChangeLogDialog')
    expect(accountsSource).toContain(':account-id="electionChangeLogAccountID"')
    expect(accountsSource).toContain(':account-label="electionChangeLogAccountLabel"')
    // 日志里的 account_id 是**本地账号 ID**：上游账号记录自己的 id 对不上，
    // 传错不会报错、只会静默查不到数据，所以必须走 manageableLocalAccountID 取。
    expect(accountsSource).toContain('const localAccountID = account ? manageableLocalAccountID(account) : null')
    expect(accountsSource).toContain('v-if="canManageLocalAccount(account)"')
  })

  it('上游账号页的行内入口沿用该页既有按钮色板', () => {
    // 行内按钮不是工具栏按钮：本页给行内动作统一用 `.sp-account-row-actions .sp-account-action-*`
    // 前缀给色，并且 hover 用的是 currentColor 的公共规则 —— 漏了这两处按钮会变成默认灰。
    expect(accountsSource).toContain('.sp-account-row-actions .sp-account-action-election-log {')
    expect(accountsSource).toContain('color: var(--sp-orange)')
    expect(accountsSource).toContain('.sp-account-row-actions .sp-account-action-election-log:hover,')
  })

  it('三处入口同色（橙），且与琥珀色的倍率日志按钮不相邻', () => {
    expect(groupsSource).toContain('sp-control-button-election-log')
    expect(groupsSource).toContain('--sp-orange')
    expect(accountsSource).toContain('sp-account-toolbar-election-logs')
    expect(accountsSource).toContain('--sp-orange')
    expect(automationSource).toContain('sp-election-log-entry')
    expect(automationSource).toContain('--sp-orange')
    // 橙与琥珀是邻近色，两个日志按钮挨着会更难分辨 —— 位置刻意排开。
    const groupsToolbar = groupsSource.slice(groupsSource.indexOf('sp-filter-actions'))
    expect(groupsToolbar.indexOf('sp-control-button-log')).toBeLessThan(groupsToolbar.indexOf('sp-control-button-columns'))
    expect(groupsToolbar.indexOf('sp-control-button-columns')).toBeLessThan(groupsToolbar.indexOf('sp-control-button-election-log'))
  })
})

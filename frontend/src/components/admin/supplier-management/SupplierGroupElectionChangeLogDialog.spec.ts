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

/** 去掉 CSS 注释后的样式文本：注释里会为了解释「为什么不能这么写」而引用坏写法本身。 */
const cssSource = source.replace(/\/\*[\s\S]*?\*\//g, '')

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
    // 按分组分节后不再走 DataTable 的 :row-key prop，行键挪到了 :key 上。
    // 同一个账号会出现在它所属的每个分节里，所以键还要再拼一层分组，
    // 否则同一父节点下照样撞键 —— 实现可以换，这个坑不能重新踩回去。
    expect(source).toContain(':key="`${batch.key}-${rowKey(log)}`"')
    // batch.key 必须含分组：只按 run_id 建键的话，同一个批次在两个分组下会共用一个块，
    // 后一个分组会把前一个的行「吃掉」（少渲染，不报错）。
    expect(source).toContain('const batchKey = `${groupKey}::${log.run_id}`')
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
    // 平台也要一起清：它是筛选条件，不清就会留下一个看不见的收窄。
    expect(source).toContain("filters.value = { direction: 'all', platform: '', search: '', startedFrom: '', startedTo: '' }")
    expect(source).toContain('function clearGroup()')
    expect(source).toContain('clearedGroup.value = true')
    expect(source).toContain('const activeGroupID = computed(() => (clearedGroup.value ? null : props.groupId ?? null))')
    expect(source).toContain('title="查看全部分组"')
  })

  it('空态区分「还没发生过切换」与「筛选太窄」', () => {
    // 用同一句话会让人以为功能坏了，直接去翻后端。
    expect(source).toContain("hasActiveFilters ? '当前筛选条件下没有调度切换记录，试试放宽时间范围或换个方向。' : '最近还没有发生调度切换。'")
  })

  it('按分组分节，且分组名查不到时不丢记录', () => {
    // 一行一个账号时同组的几行是打散的，得自己脑补「这个组整体发生了什么」。
    expect(source).toContain('const sections = computed<LogSection[]>(() => {')
    expect(source).toContain('for (const name of names) push(name, name, log)')
    expect(source).toContain('section.enabled += 1')
    expect(source).toContain('section.disabled += 1')
    // 用分组名当键的前提是「未删除的分组名唯一」（groups_name_unique_active），
    // 后端取名字的 JOIN 也带了 deleted_at IS NULL —— 两边必须同时成立。
    // 分组被删掉 / 历史数据没记名字时，这条变更仍要出现，不能静默消失。
    expect(source).toContain("push(NO_GROUP_KEY, '未记录分组', log)")
    // 分组标题行必须整行贯通，所以走 colspan 而不是 DataTable（它每行固定 N 个 <td>）。
    expect(source).toContain('scope="colgroup"')
    expect(source).toContain(':colspan="columns.length"')
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
    // 暗色覆盖必须写普通的 `.dark xxx`。
    // 实测本仓 Vue 版本里 scoped 样式中的 `:global(.dark) .foo` 会被编译成裸 `.dark {}`：
    // 后代选择器和 data-v 属性一起丢掉，声明落到 <html> 上；而弹窗根元素自己声明了浅色变量，
    // 元素自身声明压过继承 ⇒ 暗色永远不生效、且编译和运行都不报错。
    // 编译产物可自证：`.dark .foo` → `.dark .foo[data-v-xxx]`。
    expect(source).toContain('.dark .sp-election-log-dialog')
    expect(cssSource).not.toContain(':global(.dark)')
    expect(cssBlock('.sp-election-log-dialog')).toContain('--sp-election-log-accent')
    // 批次状态徽标必须用弹窗自己声明的变量，不能复用页面级 `.sp-status`：
    // 后者的 --sp-line/--sp-panel-2/--sp-green 来自 `.supplier-management-page`，
    // Teleport 到 body 后取不到 ⇒ 边框、底色、颜色全部失效，只剩裸文字，且不报错。
    expect(source).toContain('class="sp-election-log-batch-status"')
    expect(source).toContain('.sp-election-log-batch-status.is-good')
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

  it('平台筛选走精确匹配，选项复用全局目录', () => {
    // 平台单独一个入参：search 是「账号名或平台」的模糊匹配，拿它当平台筛会把
    // 「名字里含 openai 的账号」一起捞进来，看着像筛选失灵。
    expect(source).toContain('platform: filters.value.platform || undefined')
    expect(source).toContain('v-model="filters.platform"')
    // 平台清单不在这里另写一份：那份目录是「有哪些平台」的唯一前端来源，
    // 抄一份出来迟早会跟新增平台 / 自定义平台脱节。
    expect(source).toContain("import { CORE_PLATFORM_OPTIONS } from '@/utils/platformOptions'")
    expect(source).toContain('...CORE_PLATFORM_OPTIONS.map((option) => ({ value: option.value, label: option.label }))')
    // 「全部平台」必须是空串（不传参），传个 'all' 会被后端当成平台名精确匹配、一条都查不到。
    expect(source).toContain("{ value: '', label: '全部平台' }")
    // 有平台条件才算「筛选生效」：否则空态会显示「最近还没有发生调度切换」，误导成功能坏了。
    expect(source).toContain("|| filters.value.platform !== ''")
  })

  it('顶部给出最近批次快捷标签，且它不随筛选收窄', () => {
    // 每次都要手动填批次号才能回看某一批，等于没有入口。
    expect(source).toContain('v-if="recentRunIDs.length > 0"')
    expect(source).toContain('v-for="runID in recentRunIDs"')
    expect(source).toContain('class="sp-election-log-recent-run"')
    // 与表格里的批次号按钮同一行为：点一下加入对照，再点移出（复用 filterByRun）。
    expect(source).toContain('@click="filterByRun(runID)"')
    // 批次号来自后端且**不随筛选变化**：跟着筛选一起收窄的话，
    // 点一个标签其余标签就没了，没法来回切换着对比 —— 那样这个入口就废了。
    expect(source).toContain('recentRunIDs.value = result.recent_run_ids || []')
    expect(source).toContain('const recentRunIDs = ref<number[]>([])')
    expect(cssBlock('.sp-election-log-recent')).toContain('flex-wrap')
    expect(cssBlock('.sp-election-log-recent-run.is-active')).toContain('background')
  })

  it('批次可多选：顶部标签与行内批次号共用同一个「对照集合」', () => {
    // 单值锁一批时，想对照「这批动了谁、上批又动了谁」只能来回点，看完记不住。
    // 收成数组后组内本来就按批次分块，选中的几批会各自成块并排。
    expect(source).toContain('const runFilters = ref<number[]>([])')
    // 顶部标签与行内按钮必须是同一个集合、同一套行为；两套状态会互相覆盖。
    expect(source).toContain(":class=\"{ 'is-active': runFilters.includes(runID) }\"")
    expect(source).toContain(":class=\"{ 'is-active': runFilters.includes(log.run_id) }\"")
    expect(source).toContain("'已加入对照，再点移出'")
    // 再点一次是「移出这一批」，不是清空整个选择 —— 否则选了三批想取消一批就得从头再来。
    expect(source).toContain('runFilters.value.includes(runId)')
    expect(source).toContain('? runFilters.value.filter((id) => id !== runId)')
    expect(source).toContain(': [...runFilters.value, runId]')
    // 传参是列表；空列表不传 —— 传空数组会让后端拼出 IN () 这种语法错。
    expect(source).toContain('run_ids: runFilters.value.length > 0 ? runFilters.value : undefined')
    // 选了批次也算「筛选生效」，否则空态会显示「最近还没有发生调度切换」、误导成功能坏了。
    expect(source).toContain('|| runFilters.value.length > 0')
    // 已选批次要在顶部 chip 上列出来：多选之后光看标签高亮，不知道一共选了哪几批。
    expect(source).toContain("runFilters.map((id) => `#${id}`).join('、')")
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

  it('演练产生的建议条目要标出来，不能和真实切换混在一起', () => {
    // 这份日志的取数条件就是 before <> after，演练模式下它描述的是「想改成什么」而不是「改成了什么」。
    // 不标出来，事后回看就会把根本没发生过的切换当成既成事实。
    expect(source).toContain('<span v-if="log.suggested" class="sp-election-log-suggested"')
    expect(source).toContain('title="演练模式：本条只是建议，调度开关没有被修改"')
    // 徽标用虚线边框而不是实心块：方向已经有绿（开启）/红（关闭）两色，
    // 再来一个实心色块会被读成第三种方向，虚线才表达「还没落地」。
    expect(cssBlock('.sp-election-log-suggested')).toContain('dashed')
  })

  it('原因列摊开「为什么是它」：评分、名次、入选线、必需模型补选都要能看见', () => {
    // 一句话原因（「综合分入选」）没法复核：评分怎么算的、组内第几名、入选线多少、
    // 是不是因为必需模型被补选，都得摊出来，否则管理员只能回头翻配置和源码。
    expect(source).toContain('v-if="whyFacts(section, log).length > 0"')
    expect(source).toContain('<summary>依据</summary>')
    // 依据按分组存，必须挑当前分节那一条 —— 一个账号跨多个分组时各组结论可以不同，
    // 把 A 组的评分解释到 B 组的行上是纯误导。
    expect(source).toContain('decisions.find((decision) => decision.group_name === section.groupName)')
    // 旧运行记录里没有 group_decisions：取不到依据就整块不渲染，降级回原来的一行原因。
    expect(source).toContain('if (!decisions || decisions.length === 0) return undefined')
    // 必需模型补选要单独说明：它的综合分不一定进前 N，不写清楚看起来像择优算错了。
    expect(source).toContain('在赢家中无人支持，被按综合分补选开启')
    // 用时项取中性值时必须标明，否则管理员会拿这个 0.5 去反推配置。
    expect(source).toContain('用时项取中性值 0.5')
    expect(cssBlock('.sp-election-log-why')).toContain('margin-top')
  })

  it('「原因」列给本组结论，不照抄账号级的 union 原因', () => {
    // 账号的调度开关是单一字段：它在 A 组当选、在 B 组落选时，账号级 reason 仍记「分组内最优」。
    // 每个分组分节下照抄这句话，就会把「本组落选」写成「分组内最优」。
    expect(source).toContain("{{ groupReasonText(section, log) || log.reason || '—' }}")
    expect(source).toContain('function groupReasonText(section: LogSection, log: SupplierGroupElectionChangeLog): string')
    // 本组没选它、账号却开着（靠别的分组当选）时必须点破，
    // 否则「开启调度」与「本组未入选」并列会被读成自相矛盾。
    expect(source).toContain("if (log.direction === 'enabled') return '本组未入选（该账号在其它分组当选）'")
    // 因必需模型补选要能与「择优入选」区分开。
    expect(source).toContain("return `本组因必需模型 ${decision.required_models.join('、')} 补选`")
    // 旧记录没有依据 ⇒ 返回空串，由调用方降级回 log.reason。
    expect(source).toContain("if (!decision) return ''")
  })
})

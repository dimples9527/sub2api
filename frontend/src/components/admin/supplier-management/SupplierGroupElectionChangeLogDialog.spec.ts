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
    // 「含未切换」同理，是个放宽范围的开关，重置也要收回默认（false）。
    expect(source).toContain("filters.value = { direction: 'all', platform: '', search: '', startedFrom: '', startedTo: '', includeSkipped: false }")
    expect(source).toContain('function clearGroup()')
    expect(source).toContain('clearedGroup.value = true')
    // 分组筛选有三个来源（下拉 > 查看全部 > props 锁定），props 锁定的优先级最低那一档。
    expect(source).toContain('if (pickedGroup.value) return pickedGroupID.value')
    expect(source).toContain('return clearedGroup.value ? null : props.groupId ?? null')
    expect(source).toContain('title="查看全部分组"')
  })

  it('筛选区提供分组下拉，并接管「进来时的分组锁定」', () => {
    // 日志按分组分节展示，但筛选区原先只有方向 / 平台 / 账号名 / 日期 ——
    // 想「只看某个分组」只能从分组页点进来，在弹窗里换不了。
    expect(source).toContain("import { getAllIncludingInactive } from '@/api/admin/groups'")
    expect(source).toContain('v-model="groupFilterValue"')
    expect(source).toContain(':options="groupSelectOptions"')
    // 「全部分组」必须是空串（不传参），与平台下拉同一约定 ——
    // 传个 'all' 会被后端当成分组名，一条都查不到。
    expect(source).toContain("{ value: '', label: '全部分组' }")
    // 用 include_inactive 版本：历史日志里的分组可能已经停用，
    // 只查启用分组会让它在下拉里查不到，看着像日志丢了。
    expect(source).toContain('const list = await getAllIncludingInactive()')
    // 分组目录拉不到不能连累日志本身：异常必须吞掉，否则下拉一挂整个日志都看不成。
    expect(source).toContain('void loadGroupOptions()')
    // 下拉一旦被用过就完全接管分组筛选：否则「下拉显示 B、chip 还写着 A」，两处自相矛盾。
    expect(source).toContain('const showGroupChip = computed(() => !pickedGroup.value && activeGroupID.value !== null)')
    // 下拉选的分组属于筛选条件，重置要清掉（与 props 的锁定对象不同，后者保留）。
    expect(source).toContain('pickedGroup.value = false')
    // 查询由模板统一触发，与方向 / 平台两个下拉保持同一套行为。
    expect(source).toContain('@update:model-value="applyFilters"')
  })

  it('切换时间显示完整日期，不只是时分秒', () => {
    // 只给时分秒的话，跨天回看时分不清是哪天的变更。
    expect(source).toContain("import { formatDateTime } from '@/utils/format'")
    expect(source).toContain('{{ formatDateTime(log.changed_at) }}')
    expect(source).toContain('{{ formatDateTime(batch.changedAt) }}')
    // 列宽不够会静默折成两行、把整行撑高（不报错、不溢出）—— nowrap 与列宽必须同时给。
    expect(cssBlock('.sp-election-log-time')).toContain('white-space: nowrap')
    expect(source).toContain("class: 'min-w-[170px]', sortable: true")
  })

  it('弹窗宽度放宽到 95vw', () => {
    // full 档上限只有 max-w-7xl(1280px)，8 列 + 完整日期时间挤不下。
    expect(source).toContain('max-width: 95vw')
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

  it('切换时间列可排序，默认倒序（最新在前）', () => {
    // 排序只作用于「组内批次块」：这张表按分组分节，分组顺序是「变更条数多的先看」，
    // 跨分组做时间排序会把归类打散 —— 所以排序控件不能去动分组顺序。
    expect(source).toContain("const sortOrder = ref<'asc' | 'desc'>('desc')")
    expect(source).toContain("if (key !== 'changed_at') return")
    // 表头那一列要真的渲染成可点的按钮，并带方向指示。
    expect(source).toContain('class="sp-election-log-sort"')
    expect(source).toContain(':title="sortTitle"')
    expect(source).toContain('`is-${sortOrder}`')
    // 方向必须真的落到批次排序上，只改箭头图标等于点了没反应。
    expect(source).toContain("return sortOrder.value === 'asc' ? ascending : -ascending")
    // aria-sort 属于 columnheader 角色，挂在 <th> 上而不是按钮上。
    expect(source).toContain(':aria-sort="column.sortable ? ariaSortFor(column.key) : undefined"')
    // 每次打开回到默认倒序，否则「默认按时间倒序」不成立。
    expect(source).toContain("sortOrder.value = 'desc'")
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
    expect(source).toContain('v-if="recentRuns.length > 0"')
    expect(source).toContain('v-for="run in recentRuns"')
    expect(source).toContain('class="sp-election-log-recent-run"')
    // 与表格里的批次号按钮同一行为：点一下加入对照，再点移出（复用 filterByRun）。
    expect(source).toContain('@click="filterByRun(run.run_id)"')
    // 批次来自后端且**不随筛选变化**：跟着筛选一起收窄的话，
    // 点一个标签其余标签就没了，没法来回切换着对比 —— 那样这个入口就废了。
    expect(source).toContain('recentRuns.value = result.recent_runs || []')
    expect(source).toContain('const recentRuns = ref<SupplierGroupElectionRecentRun[]>([])')
    expect(cssBlock('.sp-election-log-recent')).toContain('flex-wrap')
    expect(cssBlock('.sp-election-log-recent-run.is-active')).toContain('background')
  })

  it('批次统一显示时间流水号，不是裸的 run_id', () => {
    // run_id 是自增主键，多 worker 并发时会交错 —— 光看号码既不知道是哪一批、
    // 也读不出先后；时间流水号一眼能看出批次时间。
    expect(source).toContain('function runSerial(changedAt: string): string')
    // 四处必须统一：顶部快捷标签、批次小标题、表格行内按钮、已选批次 chip。
    expect(source).toContain('{{ runSerial(run.changed_at) }}')
    expect(source).toContain('批次 {{ runSerial(batch.changedAt) }}')
    expect(source).toContain('{{ runSerial(log.changed_at) }}')
    expect(source).toContain("runFilters.map((id) => runLabel(id)).join('、')")
    // chip 只有批次号、拿不到时间，所以要回头查一次；查不到时退回裸号，不显示空串。
    expect(source).toContain('function runLabel(runID: number): string')
    expect(source).toContain('return changedAt ? runSerial(changedAt) : `#${runID}`')
    // 显示格式换了，传后端的值不能跟着换：筛选仍然用真实的 run_id。
    expect(source).toContain('@click="filterByRun(log.run_id)"')
    expect(source).toContain('run_ids: runFilters.value.length > 0 ? runFilters.value : undefined')
  })

  it('批次可多选：顶部标签与行内批次号共用同一个「对照集合」', () => {
    // 单值锁一批时，想对照「这批动了谁、上批又动了谁」只能来回点，看完记不住。
    // 收成数组后组内本来就按批次分块，选中的几批会各自成块并排。
    expect(source).toContain('const runFilters = ref<number[]>([])')
    // 顶部标签与行内按钮必须是同一个集合、同一套行为；两套状态会互相覆盖。
    expect(source).toContain(":class=\"{ 'is-active': runFilters.includes(run.run_id) }\"")
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
    // chip 显示的也是时间流水号，与表格 / 标签保持同一套称呼。
    expect(source).toContain("runFilters.map((id) => runLabel(id)).join('、')")
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

  it('批次口径只写一次：上提到批次小标题行，不再逐行重复', () => {
    // 取前 N 名、入选线、三项权重、次数封顶这些「同组同批次每行都一样」的参数，
    // 逐行铺正是这份日志显得啰嗦的根源 —— 上提到批次小标题行，一组一批只写一次。
    expect(source).toContain('function batchCalibration(')
    expect(source).toContain('class="sp-election-log-batch-calibration"')
    expect(source).toContain('取前 ${anyScored.top_n} 名')
    expect(source).toContain('入选线 ${anyScored.winner_cutoff.toFixed(3)}')
    expect(source).toContain('次数封顶 ${anyScored.count_score_cap}')
    // 该批次没人参与评分就没有口径可言，整块不渲染。
    expect(source).toContain('if (!anyScored) return')
  })

  it('「本组不计优先级」要能看出来，不能把配置权重照抄成 0.5', () => {
    // priority_weight 恒为配置值（本组禁用优先级时后端也照给 0.5），
    // 无条件列进权重会把「不计」写成「计了 0.5」——所以禁用时改列一句显式口径。
    expect(source).toContain('anyScored.priority_enabled !== false')
    expect(source).toContain("parts.push('本组不计优先级')")
    // 判定依据必须来自后端那一条事实，不能拿 priority_score 去猜：
    // 0 分既可能是「本组不计」，也可能是「计了、但这账号就是最低分」，前端分不出来。
    expect(source).toContain('decision.priority_enabled !== false')
  })

  it('评分贡献条只对 scored 行画，且只画真有贡献的段', () => {
    // 锁定无评分、测试失败、无备选保留这些行本来就没参与打分，没有构成可言，不画条。
    expect(source).toContain('function scoreBreakdown(')
    expect(source).toContain('if (!decision || !decision.scored) return null')
    // 段宽 = 加权贡献 / 综合分。
    expect(source).toContain('const widthPercent = total > 0 ? ((scoreValue * weight) / total) * 100 : 0')
    // 0 贡献的段（本组不计优先级、或该项就是最低分）画出来只剩 1px 空条 + 被裁掉的标签，
    // 会被读成「这项有值、只是很小」，一律不画；全空就整块不渲染。
    expect(source).toContain('if (widthPercent <= 0) return')
    expect(source).toContain('if (segments.length === 0 && flags.length === 0) return null')
    // 段内标「次0.900」而不是居中：段窄了居中的字会被裁掉，段首至少和色块起点对得上。
    expect(cssBlock('.sp-election-log-score-segment')).toContain('justify-content: flex-start')
    // 段内要直接读得到这一项算出来的分，不能只给颜色和宽度、让管理员去悬停看 title。
    expect(source).toContain('{{ segment.shortLabel }}{{ segment.value }}')
    expect(source).toContain('value: score(scoreValue)')
    // 段窄时横向裁尾巴，不折行 —— 条只有 18px 高，折成两行哪一行都读不全。
    expect(cssBlock('.sp-election-log-score-segment-label')).toContain('white-space: nowrap')
    // 标签色必须是弹窗自己声明的变量：段底色是「色相 50% 混面板」，
    // 靠继承拿到的字色会随主题漂，浅底深字 / 深底浅字都不一定成立。
    expect(source).toContain('--sp-election-log-ink: #0f172a')
    expect(cssBlock('.sp-election-log-dialog')).toContain('--sp-election-log-ink')
  })

  it('单行旗标只在该行命中时出现，不再逐行铺封顶 / 中性 / 入选线', () => {
    // 封顶命中、用时中性只跟这一行有关：命中才显示，不命中不出现。
    expect(source).toContain("flags.push('次数封顶命中')")
    expect(source).toContain("flags.push('用时中性')")
    expect(source).toContain("{{ scoreBreakdown(section, log)!.flags.join(' · ') }}")
    expect(cssBlock('.sp-election-log-score-flags')).toContain('margin-top')
  })

  it('原因列摊开「为什么是它」：评分、名次、必需模型补选都要能看见', () => {
    // 一句话原因（「综合分入选」）没法复核：评分怎么算的、组内第几名、
    // 是不是因为必需模型被补选，都得摊出来，否则管理员只能回头翻配置和源码。
    // 依据按分组存，必须挑当前分节那一条 —— 一个账号跨多个分组时各组结论可以不同，
    // 把 A 组的评分解释到 B 组的行上是纯误导。
    expect(source).toContain('decisions.find((decision) => decision.group_name === section.groupName)')
    // 旧运行记录里没有 group_decisions：取不到依据就整块不渲染，降级回原来的一行原因。
    expect(source).toContain('if (!decisions || decisions.length === 0) return undefined')
    // 必需模型补选要单独分类：它的综合分不一定进前 N，混进「择优入选」看起来像择优算错了。
    expect(source).toContain("kind: 'required'")
    expect(source).toContain("note: decision.required_models.join('、')")
    // 用时项取中性值时必须标明，否则管理员会拿这个 0.5 去反推配置。
    expect(source).toContain("flags.push('用时中性')")
    expect(cssBlock('.sp-election-log-score-breakdown')).toContain('margin-top')
    // 综合分要能手工复核：段条只标分项得分（窄了还会被裁），原始值又散在别的列，
    // 必须把「原始值 → 分项得分 × 权重 → 求和」整式写进 DOM。
    expect(source).toContain('sp-election-log-score-formula')
    expect(source).toContain('= ${score(total)}')
    expect(cssBlock('.sp-election-log-score-formula')).toContain('word-break')
  })

  it('「原因」列按本组那条依据分类，不照抄账号级的 union 原因', () => {
    // 账号的调度开关是单一字段：它在 A 组当选、在 B 组落选时，账号级 reason 仍记「分组内最优」。
    // 每个分组分节下照抄这句话就会把「本组落选」写成「分组内最优」——所以按本组那条依据重新分类。
    expect(source).toContain('function classifyReason(')
    expect(source).toContain('decisions.find((decision) => decision.group_name === section.groupName)')
    // 本组没选它、账号却开着（靠别的分组当选）时必须点破，
    // 否则「开启调度」与「未入选」并列会被读成自相矛盾。
    expect(source).toContain("note: '该账号在其它分组当选'")
    // 因必需模型补选要能与「择优入选」区分开：两者是不同的徽标分类。
    expect(source).toContain("kind: 'required'")
    expect(source).toContain("kind: 'elected'")
    // 上游停用必须排在最前判定：这类账号的测试状态与健康计数是「上游停用那一刻」冻结的旧数据，
    // 日志上看起来一切正常，落进后面任何一档都会给出误导结论（实测会落到「未参与择优」，
    // 读起来像评分没算上它，实际是请求打过去必然失败）。
    expect(source).toContain("kind: 'upstream'")
    expect(source.indexOf("kind: 'upstream'")).toBeLessThan(source.indexOf("kind: 'converged'"))
    // 收敛关闭与在任者健康锁定是两回事（一个关、一个留），必须分开分类；
    // 且收敛判断排在锁定之前 —— 被收敛的在任者也带 locked，先判 locked 会把「关闭」标成「保留」。
    expect(source.indexOf("kind: 'converged'")).toBeLessThan(source.indexOf("kind: 'locked'"))
    // 叠加了必需模型这一层时必须说全：只写「超过上限」，用户解释不了
    // 「我手动开的那个账号为什么被换掉、留下的是另一个」——那正是这条日志要回答的问题。
    expect(source).toContain('over_capacity_required_model')
    expect(source).toContain('且必需模型已由保留的账号覆盖')
    // 必需模型补选的顺序同理（方向相反）：补选跑在收敛之后、锁定组也执行，
    // 补选进来的账号同样带 locked，先判 locked 会把「刚被开启」标成「本轮未换人」。
    expect(source.indexOf("kind: 'required'")).toBeLessThan(source.indexOf("kind: 'locked'"))
    // 分组保底同理，但方向是「先判保底、后判入选」：保底账号同样带 elected（保底也是一种入选），
    // 可它的综合分不是前 N（多数情况压根没参选——全失败/全未测），
    // 先判 elected 会把「兜底开启」标成「择优入选」，看起来像择优算错了。
    expect(source).toContain("kind: 'keep-alive'")
    expect(source.indexOf("kind: 'keep-alive'")).toBeLessThan(source.indexOf("kind: 'elected'"))
    // 旧记录没有依据 ⇒ 不归类，降级回账号级原文。
    expect(source).toContain("if (!decision) return { badge: null, note: '' }")
    expect(source).toContain("fallback: log.reason || '—'")
  })

  it('原因列用分类色标签 + 行内名次/综合分，替代灰色散文', () => {
    // 原来结论是全表最弱的灰色 11px 小字，扫读和对比都费劲：
    // 现在收敛成一组带色徽标（配色由业务语义驱动），名次/综合分从折叠的「依据」提到行内。
    expect(source).toContain('class="sp-election-log-reason-badge"')
    expect(source).toContain(':class="reasonBadgeClass(reasonView(section, log).badge)"')
    expect(source).toContain('class="sp-election-log-reason-rank"')
    expect(source).toContain('function reasonRankText(')
    expect(source).toContain('名次 ${decision.rank}/${decision.rank_total}')
    expect(source).toContain('综合分 ${decision.score.toFixed(3)}')
    // 分类色必须真的落到样式上（扫读靠的就是颜色区分），不能只有类名没有配色。
    expect(cssBlock('.sp-election-log-reason-badge.is-elected')).toContain('color')
    expect(cssBlock('.sp-election-log-reason-badge.is-converged')).toContain('color')
    // 分组保底是「兜底开启」而不是「凭成绩入选」，配色必须与 is-elected 的绿分开，
    // 否则同一列里扫过去会把保底读成择优结果。
    expect(cssBlock('.sp-election-log-reason-badge.is-keep-alive')).toContain('color')
    // 上游停用刻意避开 is-test-failed 的红：两者都要人去处理，但动作完全不同 ——
    // 测试失败等它自己翻盘（失败闸门给缓冲轮次），上游停用要先去恢复供应商。同色会让人按前者处置、白等。
    expect(cssBlock('.sp-election-log-reason-badge.is-upstream')).toContain('color')
    // 没归到分类的行（旧记录、账号级跳过原因）降级回原来的灰色原文，不硬塞徽标。
    expect(source).toContain('class="sp-election-log-reason"')
    expect(source).toContain('reasonView(section, log).fallback')
  })
})

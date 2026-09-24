import { readFileSync } from 'node:fs'
import { resolve } from 'node:path'
import { describe, expect, it } from 'vitest'
import { mount, type VueWrapper } from '@vue/test-utils'
import { nextTick } from 'vue'
import Input from '@/components/common/Input.vue'
import Select from '@/components/common/Select.vue'
import i18n from '@/i18n'
import AccountHealthGuardResult from './SupplierAccountHealthGuardResult.vue'
import type {
  SupplierAccountHealthGuardItem,
  SupplierAccountHealthGuardResult as AccountHealthGuardResultData,
} from '@/api/admin/supplierAutomation'

const source = readFileSync(
  resolve(process.cwd(), 'src/components/admin/supplier-management/SupplierAccountHealthGuardResult.vue'),
  'utf8'
)

/** 去掉 HTML 与 CSS 注释后的源码：注释里会为了说明「为什么」而引用坏写法本身。 */
const codeSource = source
  .replace(/<!--[\s\S]*?-->/g, '')
  .replace(/\/\*[\s\S]*?\*\//g, '')

describe('SupplierAccountHealthGuardResult', () => {
  it('结果由 props 传入，组件内不再读 run', () => {
    expect(source).toContain('const props = defineProps<{')
    expect(source).toContain('result: SupplierAccountHealthGuardResult')
    // 明细组件只负责渲染一份结果，取数留在页面，避免两处都去碰 detailRun。
    expect(codeSource).not.toContain('detailRun')
  })

  it('保留十项统计格，八个可筛值一个不少', () => {
    for (const key of [
      'total',
      'selected',
      'checked',
      'healthy',
      'slow',
      'failed',
      'unavailable',
      'pending',
      'disabled',
      'recovered',
    ]) {
      expect(source).toContain(`key: '${key}'`)
    }
    for (const value of [
      'checked',
      'healthy',
      'slow',
      'failed',
      'skipped',
      'unavailable',
      'disabled',
      'recovered',
    ]) {
      expect(source).toContain(`{ value: '${value}'`)
    }
  })

  it('统计格可点筛选，再点一次取消', () => {
    expect(source).toContain('health-guard-summary-filter-')
    expect(source).toContain(':aria-pressed="statusFilter === metric.filter"')
    expect(source).toContain('@click="setStatusFilter(metric.filter)"')
    expect(source).toContain('function setStatusFilter(filter: string)')
    expect(source).toContain("statusFilter.value = statusFilter.value === filter ? 'all' : filter")
    expect(source).toContain("metric.filter === statusFilter ? '取消筛选' : `筛选${metric.label}明细`")
  })

  it('筛选口径与页面一致：checked 是三种已测状态，disabled / recovered 看 action', () => {
    expect(source).toContain('function matchesStatus(item: SupplierAccountHealthGuardItem, filter: string): boolean')
    expect(source).toContain("if (filter === 'checked')")
    expect(source).toContain("item.status === 'healthy' || item.status === 'slow' || item.status === 'failed'")
    expect(source).toContain("if (filter === 'disabled' || filter === 'recovered') return item.action === filter")
  })

  it('明细卡片保留每个账号的完整字段', () => {
    for (const field of [
      'item.local_account_name',
      'item.sources',
      'item.platform',
      'item.model_id',
      'item.latency_ms',
      'item.consecutive_failed',
      'item.consecutive_slow',
      'item.consecutive_healthy',
      'item.schedulable_before',
      'item.schedulable_after',
      'item.action',
      'item.reason',
      'item.error_message',
      'item.interval_seconds',
      'item.next_check_at',
    ]) {
      expect(source).toContain(field)
    }
    expect(source).toContain('健康守护明细')
    for (const label of ['健康', '慢响应', '失败', '不可用', '待下轮', '暂停', '恢复']) {
      expect(source).toContain(label)
    }
  })

  it('跳过原因按原因聚合并给出样例账号', () => {
    expect(source).toContain('result.skip_reasons')
    expect(source).toContain('跳过原因')
    expect(source).toContain('function sampleAccountNames(reason: SupplierAccountHealthGuardSkipReason): string')
    expect(source).toContain('reason.sample_accounts')
  })

  it('说清本轮只测了一批账号，不是白名单全量', () => {
    // 后端按游标取 max_accounts_per_run 个账号测试并推进游标（supplier_account_health_guard.go），
    // 页面上不写这句，「实际检查 20」会被读成「156 个账号全测过了」。
    expect(source).toContain('const progressSummary = computed(')
    expect(source).toContain('白名单 ${props.result.total_accounts} 个账号')
    expect(source).toContain('本轮选中 ${props.result.selected_count} 个')
    expect(source).toContain('实际检查 ${props.result.checked_count} 个')
    expect(source).toContain('每轮按游标顺序取一批账号检查')
    expect(source).toContain('未出现在本轮的账号由后续轮次继续')
  })

  it('异常优先：只有健康且未拨动调度才算正常', () => {
    expect(source).toContain('function isHealthyItem(item: SupplierAccountHealthGuardItem): boolean')
    expect(source).toContain("return item.status === 'healthy' && item.action === 'none'")
    expect(source).toContain('const statusSeverity: Record<string, number> = {')
    expect(source).toContain('function sortItems(items: SupplierAccountHealthGuardItem[])')
    expect(source).toContain('const bySeverity = (statusSeverity[a.status] ?? 5) - (statusSeverity[b.status] ?? 5)')
  })

  it('健康账号默认折叠成一行，可展开', () => {
    expect(source).toContain('const healthyExpanded = ref(false)')
    expect(source).toContain('const expandHealthy = computed(')
    // 任何「收窄」条件都必须让健康账号自动展开：只看到一行折叠提示会被读成筛选没生效。
    expect(source).toContain("healthyExpanded.value\n  || statusFilter.value === 'healthy'\n  || Boolean(keyword.value.trim())")
    expect(source).toContain("|| platformFilter.value !== 'all'")
    expect(source).toContain("|| providerFilter.value !== 'all'")
    expect(source).toContain('class="sp-ahg-collapsed"')
    expect(source).toContain('健康 {{ section.healthyCount }} 个账号，本轮无调度变更')
    expect(source).toContain('function isItemVisible(item: SupplierAccountHealthGuardItem): boolean')
    expect(source).toContain('const healthyTotal = computed(')
  })

  it('可按平台或供应商聚合，且说明多来源账号会重复出现', () => {
    expect(source).toContain("const groupMode = ref<GroupMode>('none')")
    expect(source).toContain("{ value: 'platform', label: '按平台聚合' }")
    expect(source).toContain("{ value: 'provider', label: '按供应商聚合' }")
    expect(source).toContain('function groupRefs(item: SupplierAccountHealthGuardItem): GroupRef[]')
    expect(source).toContain("if (groupMode.value === 'platform')")
    expect(source).toContain("if (groupMode.value === 'provider')")
    // 健康守护的结果里没有本地分组字段，聚合只能落在 platform 与 sources[].provider_name 上。
    expect(codeSource).not.toContain('local_group')
    expect(source).toContain('一个本地账号来自多个供应商时，会在每个供应商下各出现一次')
  })

  it('支持按账号名、平台、供应商搜索', () => {
    expect(source).toContain("const keyword = ref('')")
    expect(source).toContain('function itemHaystack(item: SupplierAccountHealthGuardItem): string')
    expect(source).toContain('item.local_account_name, item.platform, item.model_id, item.match_status')
    expect(source).toContain('source.provider_name')
    expect(source).toContain('source.upstream_account_name')
    expect(source).toContain('const matchedItems = computed(')
    expect(source).toContain('placeholder="搜索账号 / 平台 / 供应商"')
  })

  it('支持异常优先、延迟、倍率、账号名四种排序', () => {
    expect(source).toContain("const sortMode = ref<SortMode>('abnormal')")
    expect(source).toContain("{ value: 'abnormal', label: '异常优先' }")
    expect(source).toContain("{ value: 'latency', label: '按延迟降序' }")
    expect(source).toContain("{ value: 'multiplier', label: '按倍率升序' }")
    expect(source).toContain("{ value: 'name', label: '按账号名' }")
    expect(source).toContain("if (sortMode.value === 'latency')")
    expect(source).toContain("if (sortMode.value === 'multiplier')")
    expect(source).toContain("if (sortMode.value === 'name')")
    // 未知倍率排最后，不能当成 0 —— 0 是合法的「计费为 0」，混在一起就分不出「免费」和「查不到」。
    expect(source).toContain('function itemMultiplierSortKey(item: SupplierAccountHealthGuardItem): number')
    expect(source).toContain('Number.POSITIVE_INFINITY')
  })

  it('平台与供应商两个筛选互相收窄，且失效的选中项会被收回', () => {
    // 候选只应用对方那一个筛选，不能复用 matchedItems（它同时应用两个筛选，
    // 会让当前选中项永远留在候选里，联动退化成两个独立筛选）。
    expect(source).toContain('const platformFacetItems = computed(')
    expect(source).toContain('const providerFacetItems = computed(')
    expect(source).toContain('matchesProviderFilter(item, providerFilter.value)')
    expect(source).toContain('matchesPlatformFilter(item, platformFilter.value)')
    expect(source).toContain('const platformFilterOptions = computed<SelectOption[]>')
    expect(source).toContain('const providerFilterOptions = computed<SelectOption[]>')
    // 联动必须把选不到的项收回去，否则会留下一个下拉里已不存在、又取消不掉的选中项。
    // 收回的是**被挤掉的那个**（watch 挂在「值」上），不是刚改的那个 —— 否则表现是「点了没反应」。
    expect(source).toContain('watch(platformFilter, () => {')
    expect(source).toContain('watch(providerFilter, () => {')
    expect(source).toContain("if (!providerFilterOptions.value.some(option => option.value === providerFilter.value)) {")
    expect(source).toContain("if (!platformFilterOptions.value.some(option => option.value === platformFilter.value)) {")
    // 缺省平台口径与 groupRefs 的聚合口径必须一致，否则「聚合里叫未知平台、筛选里选不到」。
    expect(source).toContain("return item.platform || '未知平台'")
    expect(source).toContain('if (!matchesPlatformFilter(item, platformFilter.value)) return false')
    expect(source).toContain('if (!matchesProviderFilter(item, providerFilter.value)) return false')
  })

  it('显式标注联合类型，避免字面量被推成 string', () => {
    // ref('none') 会被推成 string，赋给 Select 的联合类型时报 TS2322，写错的值后端当「不过滤」静默吞掉。
    expect(source).toContain("const groupMode = ref<GroupMode>('none')")
    expect(source).toContain("const sortMode = ref<SortMode>('abnormal')")
    expect(source).toContain("type GroupMode = 'none' | 'platform' | 'provider'")
    expect(source).toContain("type SortMode = 'abnormal' | 'latency' | 'name' | 'multiplier'")
  })

  it('复用框架通用组件与模块全局类，不自造控件', () => {
    expect(source).toContain("import Select, { type SelectOption } from '@/components/common/Select.vue'")
    expect(source).toContain('<Input v-model="keyword"')
    expect(source).toContain('<Select v-model="platformFilter"')
    expect(source).toContain('<Select')
    expect(source).toContain('v-model="providerFilter"')
    expect(source).toContain('<Select v-model="groupMode"')
    expect(source).toContain('<Select v-model="sortMode"')
    expect(source).toContain('<Select v-model="statusFilter"')
    expect(source).toContain('class="sp-status"')
    expect(source).toContain('class="sp-link-button"')
    expect(source).toContain('sp-button small ghost sp-ahg-healthy-toggle')
  })

  it('不引用页面 scoped 专有类名', () => {
    // 这些类只存在于 SupplierAutomationView 的 <style scoped> 里，
    // 组件里写它们会静默失效（scoped 不跨组件），表现为「改了颜色但没生效」。
    expect(codeSource).not.toContain('sp-rate-guard-summary')
    expect(codeSource).not.toContain('sp-rate-guard-detail')
    expect(codeSource).not.toContain('sp-detail-label')
    expect(codeSource).not.toContain('sp-health-guard-item')
    expect(codeSource).not.toContain('sp-account-health-guard-summary')
  })

  it('不重复弹窗标题，也不加英文眉题', () => {
    expect(codeSource).not.toContain('Execution Outcome')
    expect(codeSource).not.toContain('Result Detail')
    expect(codeSource).not.toContain('执行结论')
    expect(codeSource).not.toContain('运行详情')
  })

  it('变量依赖写在注释里，且暗色只用普通 .dark 前缀', () => {
    // 本组件嵌在 .sp-run-detail 内，--sp-* 由页面挂在 .modal-content:has(.sp-run-detail) 上。
    expect(source).toContain(':global(.modal-content:has(.sp-run-detail))')
    // scoped 里的 :global(.dark) X 会被编译成裸 .dark，后代选择器整条丢弃，暗色静默失效。
    expect(codeSource).not.toContain(':global(.dark)')
  })

  it('窄屏下统计格纵向堆叠，明细表改为横向滚动', () => {
    expect(source).toContain('@media (max-width: 1200px)')
    expect(source).toContain('@media (max-width: 720px)')
    // 明细行是 9 列网格：窄屏再堆叠会把「一行一个账号」拆成九行，看不出同一行的
    // 几个数字是一组，所以改成容器横向滚动（靠行上的 min-width 撑出滚动宽度）。
    expect(source).toContain('.sp-ahg-table {')
    expect(source).toContain('overflow-x: auto;')
    expect(source).toContain('.sp-ahg-row {')
    // 1138px = 九列最小值之和。这个数不能随意下调：它小于弹窗最宽时的内容宽（约 1213px），
    // 所以最宽的弹窗不出现横向滚动；而各列最小值都按「不折行所需宽度」实测得出，
    // 任何一列被挤到折行都会让**整行**变高（实测 57px → 73px）。
    expect(source).toContain('min-width: 1138px;')
  })

  it('测试间隔列上下两行：上行间隔、下行距下次检查还剩多久', () => {
    expect(source).toContain('<span>测试间隔</span>')
    expect(source).toContain('function intervalText(seconds: number | undefined): string')
    expect(source).toContain('function nextCheckText(item: SupplierAccountHealthGuardItem): string')
    // 0 是「每轮都测」而不是「没配置」；字段缺失（不可用/未匹配/已停用的账号）才是不适用。
    expect(source).toContain("if (seconds <= 0) return '每轮都测'")
    // 基准必须是当前时间：以记录结束时间为基准会让历史记录给出早就过期的答案。
    expect(source).toContain('Date.now()')
    expect(source).toContain("return '已可检查'")
  })
})

// 挂载真实组件验证「倍率升序」与「平台 / 供应商联动」的实际行为。
// 上面那些源码字符串断言证明不了联动方向、watch 会不会成环、排序有没有真的生效 ——
// 这三件事只有真跑一遍才能证明。
describe('SupplierAccountHealthGuardResult 交互', () => {
  // 甲只在 openai 有账号、乙横跨 openai 与 anthropic —— 这样才能验证「互相收窄」而不是单向。
  // status 用 slow 而不是 healthy：健康账号默认折叠，用 healthy 会一行都渲染不出来。
  const items: SupplierAccountHealthGuardItem[] = [
    guardItem(1, 'openai', [{ id: 1, name: '供应商甲' }], 0.5),
    guardItem(2, 'anthropic', [{ id: 2, name: '供应商乙' }], 2),
    guardItem(3, 'openai', [{ id: 2, name: '供应商乙' }], 1),
    // 倍率未知（账号已查不到）：升序时排最后，不能当成 0 排到最前。
    guardItem(4, 'openai', [{ id: 1, name: '供应商甲' }]),
  ]

  function mountResult(): VueWrapper {
    return mount(AccountHealthGuardResult, {
      props: { result: guardResult(items) },
      // 真实 Select 在 setup 里调 useI18n，必须装 i18n 插件（i18n 是已创建的实例，可同步安装）。
      // Input 是全局注册的组件，挂载时要显式提供，否则 Vue 会报解析失败。
      global: { components: { Input }, plugins: [i18n] },
    })
  }

  it('选平台后，供应商候选只剩该平台下有的', async () => {
    const wrapper = mountResult()
    await pickOption(wrapper, '全部平台', 'anthropic')
    expect(optionLabels(selectByOptionLabel(wrapper, '全部供应商'))).toEqual(['全部供应商', '供应商乙'])
  })

  it('选供应商后，平台候选只剩该供应商下有的', async () => {
    const wrapper = mountResult()
    await pickOption(wrapper, '全部供应商', '1')
    expect(optionLabels(selectByOptionLabel(wrapper, '全部平台'))).toEqual(['全部平台', 'openai'])
  })

  it('后改的筛选生效，被挤掉的那个自动清空', async () => {
    const wrapper = mountResult()
    // 先选甲（只有 openai），再选 anthropic —— 两者冲突。
    await pickOption(wrapper, '全部供应商', '1')
    await pickOption(wrapper, '全部平台', 'anthropic')
    // 刚选的平台必须生效：若把平台回退掉，用户看到的就是「点了没反应」。
    expect(selectByOptionLabel(wrapper, '全部平台').props('modelValue')).toBe('anthropic')
    expect(selectByOptionLabel(wrapper, '全部供应商').props('modelValue')).toBe('all')
    expect(rowAccountIDs(wrapper)).toEqual([2])
  })

  it('按倍率升序排序，未知倍率排最后', async () => {
    const wrapper = mountResult()
    await pickOption(wrapper, '异常优先', 'multiplier')
    expect(rowAccountIDs(wrapper)).toEqual([1, 3, 2, 4])
  })

  it('平台与供应商两个筛选同时生效', async () => {
    const wrapper = mountResult()
    await pickOption(wrapper, '全部平台', 'openai')
    await pickOption(wrapper, '全部供应商', '2')
    expect(rowAccountIDs(wrapper)).toEqual([3])
  })
})

function guardItem(
  localAccountID: number,
  platform: string,
  providers: Array<{ id: number; name: string }>,
  multiplier?: number,
): SupplierAccountHealthGuardItem {
  return {
    local_account_id: localAccountID,
    local_account_name: `账号 ${localAccountID}`,
    platform,
    sources: providers.map((provider, index) => ({
      provider_id: provider.id,
      provider_name: provider.name,
      supplier_provider_account_id: localAccountID * 10 + index,
      upstream_account_key: `key-${localAccountID}-${index}`,
      upstream_account_name: `上游 ${localAccountID}-${index}`,
    })),
    schedulable_before: true,
    schedulable_after: true,
    status: 'slow',
    latency_ms: 10,
    latency_limit_ms: 15000,
    consecutive_failed: 0,
    consecutive_slow: 0,
    consecutive_healthy: 0,
    action: 'none',
    billing_rate_multiplier: multiplier,
    started_at: '2026-09-24T08:00:00Z',
    finished_at: '2026-09-24T08:00:01Z',
  }
}

function guardResult(items: SupplierAccountHealthGuardItem[]): AccountHealthGuardResultData {
  return {
    total_accounts: items.length,
    selected_count: items.length,
    checked_count: items.length,
    healthy_count: 0,
    slow_count: items.length,
    failed_count: 0,
    skipped_count: 0,
    unavailable_count: 0,
    pending_count: 0,
    disabled_count: 0,
    recovered_count: 0,
    unchanged_count: 0,
    cursor_account_id: 0,
    items,
  }
}

/** 按「某个选项文案」定位 Select：比按下标取更抗模板顺序调整。 */
function selectByOptionLabel(wrapper: VueWrapper, label: string): VueWrapper {
  const found = wrapper.findAllComponents(Select).find(candidate => (
    (candidate.props('options') as Array<{ label: string }>).some(option => option.label === label)
  ))
  if (!found) throw new Error(`未找到含选项「${label}」的 Select`)
  return found
}

function optionLabels(select: VueWrapper): string[] {
  return (select.props('options') as Array<{ label: string }>).map(option => option.label)
}

/** 直接派发 v-model 事件：真实 Select 的下拉需要点击展开，这里只需要驱动值。 */
async function pickOption(wrapper: VueWrapper, selectLabel: string, value: string): Promise<void> {
  selectByOptionLabel(wrapper, selectLabel).vm.$emit('update:modelValue', value)
  await nextTick()
}

/** 明细行的账号 ID，按渲染顺序。第一格里的 `<small>` 就是「本地账号 #N」。 */
function rowAccountIDs(wrapper: VueWrapper): number[] {
  return wrapper.findAll('.sp-ahg-row:not(.sp-ahg-row-head)').map(row => (
    Number(row.find('small').text().replace(/\D+/g, ''))
  ))
}

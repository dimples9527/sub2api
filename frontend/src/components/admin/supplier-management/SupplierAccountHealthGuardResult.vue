<template>
  <div class="sp-ahg">
    <!-- 分批进度：健康守护每轮只按游标取一批账号测试，不说清就会被读成「白名单全量都测过了」。 -->
    <div class="sp-ahg-progress">
      <span class="sp-ahg-progress-main">{{ progressSummary }}</span>
      <span class="sp-ahg-progress-note">
        每轮按游标顺序取一批账号检查，未出现在本轮的账号由后续轮次继续，因此这里不是白名单全量。
      </span>
    </div>

    <div class="sp-ahg-metrics" role="group" aria-label="健康守护统计筛选">
      <div
        v-for="metric in summaryMetrics"
        :key="metric.key"
        class="sp-ahg-metric"
        :class="[
          metric.filter ? 'is-filterable' : 'is-static',
          metric.tone,
          { active: metric.filter && statusFilter === metric.filter },
        ]"
      >
        <button
          v-if="metric.filter"
          type="button"
          class="sp-ahg-metric-button"
          :data-test="`health-guard-summary-filter-${metric.filter}`"
          :aria-pressed="statusFilter === metric.filter"
          :title="metric.filter === statusFilter ? '取消筛选' : `筛选${metric.label}明细`"
          @click="setStatusFilter(metric.filter)"
        >
          <span>{{ metric.label }}</span>
          <strong>{{ metric.count }}</strong>
        </button>
        <template v-else>
          <span>{{ metric.label }}</span>
          <strong>{{ metric.count }}</strong>
        </template>
      </div>
    </div>

    <section v-if="result.skip_reasons?.length" class="sp-ahg-block">
      <header class="sp-ahg-block-head">
        <div><h4>跳过原因</h4></div>
        <strong>{{ result.skipped_count }} 项</strong>
      </header>
      <div class="sp-ahg-reasons">
        <article v-for="reason in result.skip_reasons" :key="reason.reason">
          <strong>{{ reason.reason }}</strong>
          <span>{{ reason.count }}</span>
          <small v-if="reason.sample_accounts?.length">{{ sampleAccountNames(reason) }}</small>
        </article>
      </div>
    </section>

    <section class="sp-ahg-block">
      <header class="sp-ahg-block-head">
        <div><h4>健康守护明细</h4></div>
        <strong>已显示 {{ visibleCount }} / {{ result.items.length }} 个账号</strong>
      </header>

      <div class="sp-ahg-toolbar">
        <div class="sp-ahg-field sp-ahg-search">
          <span class="sr-only">搜索账号</span>
          <Input v-model="keyword" placeholder="搜索账号 / 平台 / 供应商" />
        </div>
        <div class="sp-ahg-field">
          <span class="sr-only">平台筛选</span>
          <Select v-model="platformFilter" :options="platformFilterOptions" :searchable="false" />
        </div>
        <div class="sp-ahg-field">
          <span class="sr-only">供应商筛选</span>
          <Select
            v-model="providerFilter"
            :options="providerFilterOptions"
            searchable
            empty-text="该平台下暂无供应商"
          />
        </div>
        <div class="sp-ahg-field">
          <span class="sr-only">聚合方式</span>
          <Select v-model="groupMode" :options="groupOptions" :searchable="false" />
        </div>
        <div class="sp-ahg-field">
          <span class="sr-only">排序方式</span>
          <Select v-model="sortMode" :options="sortOptions" :searchable="false" />
        </div>
        <div class="sp-ahg-field">
          <span class="sr-only">状态筛选</span>
          <Select v-model="statusFilter" :options="statusFilterOptions" :searchable="false" />
        </div>
        <button
          class="sp-button small ghost sp-ahg-healthy-toggle"
          type="button"
          :aria-pressed="healthyExpanded"
          :disabled="healthyTotal === 0"
          @click="healthyExpanded = !healthyExpanded"
        >
          {{ healthyExpanded ? '收起健康账号' : `展开健康账号（${healthyTotal}）` }}
        </button>
      </div>

      <p v-if="groupMode === 'provider'" class="sp-ahg-hint">
        一个本地账号来自多个供应商时，会在每个供应商下各出现一次。
      </p>

      <div v-if="sections.length" class="sp-ahg-sections">
        <article v-for="section in sections" :key="section.key" class="sp-ahg-section">
          <header v-if="groupMode !== 'none'" class="sp-ahg-section-head">
            <h5>{{ section.title }}</h5>
            <span>异常 {{ section.abnormalCount }} · 健康 {{ section.healthyCount }}</span>
          </header>

          <div class="sp-ahg-items">
            <div class="sp-ahg-table">
              <div class="sp-ahg-row sp-ahg-row-head" aria-hidden="true">
                <span>账号</span>
                <span>状态</span>
                <span>供应商来源</span>
                <span>平台 / 模型</span>
                <span>延迟</span>
                <span>连续计数</span>
                <span>调度状态</span>
                <span>测试间隔</span>
                <span>原因 / 错误</span>
              </div>
              <template v-for="item in section.ordered" :key="itemKey(item)">
                <div v-if="isItemVisible(item)" class="sp-ahg-row" :class="statusTone(item.status)">
                  <span>
                    <strong>{{ item.local_account_name || `本地账号 ${item.local_account_id}` }}</strong>
                    <small>本地账号 #{{ item.local_account_id }}</small>
                  </span>
                  <span>
                    <span class="sp-status" :class="statusTone(item.status)">{{ statusText(item.status) }}</span>
                  </span>
                  <span class="sp-ahg-sources">
                    <template v-if="item.sources?.length">
                      <small v-for="source in item.sources" :key="source.supplier_provider_account_id">
                        {{ source.provider_name || `供应商 ${source.provider_id}` }} · {{ source.upstream_account_name || source.upstream_account_key }}
                      </small>
                    </template>
                    <small v-else>无来源信息</small>
                  </span>
                  <span>
                    <strong>{{ item.platform || '-' }}</strong>
                    <small>{{ item.model_id || '默认模型' }}</small>
                  </span>
                  <span>
                    <strong>{{ item.latency_ms }}ms</strong>
                    <small>阈值 {{ item.latency_limit_ms }}ms</small>
                  </span>
                  <span>
                    <strong>失败 {{ item.consecutive_failed }} / 慢 {{ item.consecutive_slow }}</strong>
                    <small>健康 {{ item.consecutive_healthy }}</small>
                  </span>
                  <span>
                    <strong>{{ schedulableText(item.schedulable_before) }} → {{ schedulableText(item.schedulable_after) }}</strong>
                    <small>{{ actionText(item.action) }}</small>
                  </span>
                  <span>
                    <strong>{{ intervalText(item.interval_seconds) }}</strong>
                    <small v-if="nextCheckText(item)">{{ nextCheckText(item) }}</small>
                  </span>
                  <span class="sp-ahg-message" :class="{ bad: Boolean(item.error_message) }">
                    <span v-if="item.reason">原因：{{ item.reason }}</span>
                    <span v-if="item.error_message">错误：{{ item.error_message }}</span>
                  </span>
                </div>
              </template>
            </div>

            <div v-if="section.healthyCount > 0 && !expandHealthy" class="sp-ahg-collapsed">
              <span>健康 {{ section.healthyCount }} 个账号，本轮无调度变更</span>
              <button class="sp-link-button" type="button" @click="healthyExpanded = true">展开查看</button>
            </div>
          </div>
        </article>
      </div>
      <div v-else class="sp-ahg-empty">{{ emptyText }}</div>
    </section>
  </div>
</template>

<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import Select, { type SelectOption } from '@/components/common/Select.vue'
import type {
  SupplierAccountHealthGuardItem,
  SupplierAccountHealthGuardResult,
  SupplierAccountHealthGuardSkipReason,
} from '@/api/admin/supplierAutomation'

const props = defineProps<{
  result: SupplierAccountHealthGuardResult
}>()

type GroupMode = 'none' | 'platform' | 'provider'
type SortMode = 'abnormal' | 'latency' | 'name' | 'multiplier'

const statusFilter = ref('all')
const keyword = ref('')
// 平台 / 供应商用字符串值而不是数字：'all' 是「不过滤」的哨兵值，和两个 Select 的默认项同源。
const platformFilter = ref('all')
const providerFilter = ref('all')
// 显式标注字面量联合：直接 ref('none') 会被推成 string，赋给 Select 的联合类型时报 TS2322。
const groupMode = ref<GroupMode>('none')
const sortMode = ref<SortMode>('abnormal')
const healthyExpanded = ref(false)

const groupOptions: SelectOption[] = [
  { value: 'none', label: '不聚合' },
  { value: 'platform', label: '按平台聚合' },
  { value: 'provider', label: '按供应商聚合' },
]

const sortOptions: SelectOption[] = [
  { value: 'abnormal', label: '异常优先' },
  { value: 'latency', label: '按延迟降序' },
  { value: 'multiplier', label: '按倍率升序' },
  { value: 'name', label: '按账号名' },
]

const statusFilterOptions: SelectOption[] = [
  { value: 'all', label: '全部状态' },
  { value: 'checked', label: '检查' },
  { value: 'healthy', label: '健康' },
  { value: 'slow', label: '慢响应' },
  { value: 'failed', label: '失败' },
  { value: 'skipped', label: '跳过' },
  { value: 'unavailable', label: '不可用' },
  { value: 'disabled', label: '暂停' },
  { value: 'recovered', label: '恢复' },
]

const progressSummary = computed(() => {
  const parts = [`白名单 ${props.result.total_accounts} 个账号`]
  if (props.result.selected_count !== props.result.total_accounts) {
    parts.push(`本轮选中 ${props.result.selected_count} 个`)
  }
  parts.push(`实际检查 ${props.result.checked_count} 个`)
  return parts.join(' · ')
})

const summaryMetrics = computed(() => {
  const result = props.result
  return [
    { key: 'total', label: '白名单账号', count: result.total_accounts, filter: '', tone: '' },
    { key: 'selected', label: '本轮选择', count: result.selected_count, filter: '', tone: '' },
    { key: 'checked', label: '检查', count: result.checked_count, filter: 'checked', tone: 'neutral' },
    { key: 'healthy', label: '健康', count: result.healthy_count, filter: 'healthy', tone: 'good' },
    { key: 'slow', label: '慢响应', count: result.slow_count, filter: 'slow', tone: 'warn' },
    { key: 'failed', label: '失败', count: result.failed_count, filter: 'failed', tone: 'bad' },
    { key: 'unavailable', label: '不可用', count: result.unavailable_count, filter: 'unavailable', tone: 'warn' },
    { key: 'pending', label: '待下轮', count: result.pending_count, filter: '', tone: '' },
    { key: 'disabled', label: '暂停', count: result.disabled_count, filter: 'disabled', tone: 'bad' },
    { key: 'recovered', label: '恢复', count: result.recovered_count, filter: 'recovered', tone: 'good' },
  ]
})

// 只有「健康且未拨动调度」才算正常；恢复/暂停过调度的账号即使 status 是 healthy 也仍需人工过目。
function isHealthyItem(item: SupplierAccountHealthGuardItem): boolean {
  return item.status === 'healthy' && item.action === 'none'
}

const expandHealthy = computed(() => (
  healthyExpanded.value
  || statusFilter.value === 'healthy'
  || Boolean(keyword.value.trim())
  // 平台 / 供应商筛选也是收窄条件：不展开的话，选了个「账号全健康」的平台会只看到一行折叠提示，
  // 看起来像筛选没生效。与关键字搜索同一处理。
  || platformFilter.value !== 'all'
  || providerFilter.value !== 'all'
))

function matchesStatus(item: SupplierAccountHealthGuardItem, filter: string): boolean {
  if (filter === 'all') return true
  if (filter === 'checked') {
    return item.status === 'healthy' || item.status === 'slow' || item.status === 'failed'
  }
  if (filter === 'disabled' || filter === 'recovered') return item.action === filter
  return item.status === filter
}

// 平台缺省归到「未知平台」，与 groupRefs 的聚合口径保持一致：
// 两边用不同的缺省值会出现「聚合里叫未知平台、筛选里却选不到」。
function itemPlatform(item: SupplierAccountHealthGuardItem): string {
  return item.platform || '未知平台'
}

function itemProviderIDs(item: SupplierAccountHealthGuardItem): string[] {
  return (item.sources || []).map(source => String(source.provider_id))
}

function matchesPlatformFilter(item: SupplierAccountHealthGuardItem, filter: string): boolean {
  return filter === 'all' || itemPlatform(item) === filter
}

function matchesProviderFilter(item: SupplierAccountHealthGuardItem, filter: string): boolean {
  return filter === 'all' || itemProviderIDs(item).includes(filter)
}

// 平台与供应商两个筛选互相收窄（联动）：选了平台，供应商下拉里只剩该平台有的供应商；
// 选了供应商，平台下拉里只剩该供应商有的平台。
// 两边的候选各自只应用**对方**那一个筛选，不能复用 matchedItems —— 它同时应用了两个筛选，
// 拿它算候选会让「当前选中项」永远留在候选里，联动退化成两个独立筛选。
const platformFacetItems = computed(() => (
  props.result.items.filter(item => matchesProviderFilter(item, providerFilter.value))
))

const providerFacetItems = computed(() => (
  props.result.items.filter(item => matchesPlatformFilter(item, platformFilter.value))
))

const platformFilterOptions = computed<SelectOption[]>(() => {
  const platforms = new Set<string>()
  for (const item of platformFacetItems.value) platforms.add(itemPlatform(item))
  return [
    { value: 'all', label: '全部平台' },
    ...Array.from(platforms)
      .sort((a, b) => a.localeCompare(b))
      .map(platform => ({ value: platform, label: platform })),
  ]
})

const providerFilterOptions = computed<SelectOption[]>(() => {
  const providers = new Map<string, string>()
  for (const item of providerFacetItems.value) {
    for (const source of item.sources || []) {
      const value = String(source.provider_id)
      if (!providers.has(value)) {
        providers.set(value, source.provider_name || `供应商 ${source.provider_id}`)
      }
    }
  }
  return [
    { value: 'all', label: '全部供应商' },
    ...Array.from(providers, ([value, label]) => ({ value, label }))
      .sort((a, b) => a.label.localeCompare(b.label)),
  ]
})

// 联动必须把「已经选不到了」的那一项收回去：否则两个筛选交叉后会得到空列表，
// 而用户看到的是一个在下拉里已经不存在、又无法取消的选中项。
//
// 收回的是**被挤掉的那个**，而不是刚改的那个：用户点平台就是想看该平台，
// 若把平台回退成「全部平台」，表现就是「点了没反应」。
// 所以两个 watch 都挂在「值」上（谁变了就检查另一个），语义是「后改的生效」。
// 不会成环：收回只会让对方的候选变多（'all' 一定在候选里），不会把对方也挤掉。
watch(platformFilter, () => {
  if (!providerFilterOptions.value.some(option => option.value === providerFilter.value)) {
    providerFilter.value = 'all'
  }
})
watch(providerFilter, () => {
  if (!platformFilterOptions.value.some(option => option.value === platformFilter.value)) {
    platformFilter.value = 'all'
  }
})

function itemHaystack(item: SupplierAccountHealthGuardItem): string {
  const sources = (item.sources || []).flatMap(source => [
    source.provider_name,
    source.upstream_account_name,
    source.upstream_account_key,
  ])
  return [item.local_account_name, item.platform, item.model_id, item.match_status, ...sources]
    .filter(Boolean)
    .join(' ')
    .toLowerCase()
}

const matchedItems = computed(() => {
  const needle = keyword.value.trim().toLowerCase()
  return props.result.items.filter(item => {
    if (!matchesStatus(item, statusFilter.value)) return false
    if (!matchesPlatformFilter(item, platformFilter.value)) return false
    if (!matchesProviderFilter(item, providerFilter.value)) return false
    if (needle && !itemHaystack(item).includes(needle)) return false
    return true
  })
})

const statusSeverity: Record<string, number> = {
  failed: 0,
  unavailable: 1,
  slow: 2,
  skipped: 3,
  healthy: 4,
}

function sortItems(items: SupplierAccountHealthGuardItem[]): SupplierAccountHealthGuardItem[] {
  const sorted = [...items]
  if (sortMode.value === 'latency') {
    return sorted.sort((a, b) => (b.latency_ms || 0) - (a.latency_ms || 0))
  }
  if (sortMode.value === 'multiplier') {
    return sorted.sort((a, b) => itemMultiplierSortKey(a) - itemMultiplierSortKey(b))
  }
  if (sortMode.value === 'name') {
    return sorted.sort((a, b) => (a.local_account_name || '').localeCompare(b.local_account_name || ''))
  }
  return sorted.sort((a, b) => {
    const bySeverity = (statusSeverity[a.status] ?? 5) - (statusSeverity[b.status] ?? 5)
    if (bySeverity !== 0) return bySeverity
    return (b.latency_ms || 0) - (a.latency_ms || 0)
  })
}

/**
 * 倍率升序的排序键。
 * 未知倍率（账号已查不到）用 Infinity 排到最后 —— 不能用 0 代替：
 * 0 是合法的「计费为 0」，混在一起会让免费的账号和查不到的账号分不出来。
 */
function itemMultiplierSortKey(item: SupplierAccountHealthGuardItem): number {
  const multiplier = item.billing_rate_multiplier
  return typeof multiplier === 'number' && Number.isFinite(multiplier) ? multiplier : Number.POSITIVE_INFINITY
}

interface GroupRef {
  key: string
  title: string
}

// 按供应商聚合时一个账号可能落进多个分组（sources 是多来源），
// 这是有意为之：与调度切换日志「同属多个分组时在每个分组下各出现一次」保持一致。
function groupRefs(item: SupplierAccountHealthGuardItem): GroupRef[] {
  if (groupMode.value === 'platform') {
    const platform = item.platform || '未知平台'
    return [{ key: `platform-${platform}`, title: platform }]
  }
  if (groupMode.value === 'provider') {
    const sources = item.sources || []
    if (!sources.length) return [{ key: 'provider-unknown', title: '无供应商来源' }]
    const seen = new Set<string>()
    const refs: GroupRef[] = []
    for (const source of sources) {
      const title = source.provider_name || `供应商 ${source.provider_id}`
      if (seen.has(title)) continue
      seen.add(title)
      refs.push({ key: `provider-${title}`, title })
    }
    return refs
  }
  return [{ key: 'all', title: '全部账号' }]
}

interface HealthGuardSection extends GroupRef {
  ordered: SupplierAccountHealthGuardItem[]
  abnormalCount: number
  healthyCount: number
}

const sections = computed<HealthGuardSection[]>(() => {
  const buckets = new Map<string, { title: string; items: SupplierAccountHealthGuardItem[] }>()
  for (const item of matchedItems.value) {
    for (const ref of groupRefs(item)) {
      const bucket = buckets.get(ref.key)
      if (bucket) bucket.items.push(item)
      else buckets.set(ref.key, { title: ref.title, items: [item] })
    }
  }
  return [...buckets.entries()].map(([key, bucket]) => {
    const ordered = sortItems(bucket.items)
    const healthyCount = ordered.filter(isHealthyItem).length
    return {
      key,
      title: bucket.title,
      ordered,
      abnormalCount: ordered.length - healthyCount,
      healthyCount,
    }
  })
})

const healthyTotal = computed(() => (
  sections.value.reduce((total, section) => total + section.healthyCount, 0)
))

const visibleCount = computed(() => (
  sections.value.reduce(
    (total, section) => total + section.abnormalCount + (expandHealthy.value ? section.healthyCount : 0),
    0,
  )
))

const emptyText = computed(() => (
  props.result.items.length ? '当前筛选条件下没有账号明细。' : '本次运行没有返回账号明细。'
))

function itemKey(item: SupplierAccountHealthGuardItem): string {
  return `${item.local_account_id}-${item.started_at}`
}

function isItemVisible(item: SupplierAccountHealthGuardItem): boolean {
  return !isHealthyItem(item) || expandHealthy.value
}

function sampleAccountNames(reason: SupplierAccountHealthGuardSkipReason): string {
  return (reason.sample_accounts || [])
    .map(account => account.local_account_name
      || account.upstream_account_name
      || `账号 ${account.local_account_id || account.supplier_provider_account_id}`)
    .join('、')
}

function setStatusFilter(filter: string) {
  statusFilter.value = statusFilter.value === filter ? 'all' : filter
}

function statusText(status: string): string {
  const labels: Record<string, string> = {
    healthy: '健康',
    slow: '慢响应',
    failed: '失败',
    skipped: '跳过',
    unavailable: '不可用',
  }
  return labels[status] || status || '-'
}

function statusTone(status: string): string {
  if (status === 'healthy') return 'good'
  if (status === 'slow' || status === 'skipped' || status === 'unavailable') return 'warn'
  if (status === 'failed') return 'bad'
  return ''
}

function actionText(action: string): string {
  const labels: Record<string, string> = {
    none: '无变更',
    disabled: '暂停调度',
    recovered: '恢复调度',
  }
  return labels[action] || action || '-'
}

function schedulableText(value: boolean): string {
  return value ? '可调度' : '已暂停'
}

/** 检查间隔的展示文案。0 表示每轮都测；不参与本轮检查的账号后端不返回该字段。 */
function intervalText(seconds: number | undefined): string {
  if (seconds === undefined || seconds === null) return '-'
  if (seconds <= 0) return '每轮都测'
  return `${seconds}s`
}

/**
 * 距下次检查还剩多久。
 * 基准取「当前时间」而不是这条记录的结束时间：结果详情是只读快照，用户想知道的是此刻还要等多久；
 * 若以本轮结束时间为基准，翻看历史记录时会给出一个早就过期的答案。
 * 向上取整是为了不夸大剩余时间 —— 显示「还需 1 分」时实际至少还有 1 分钟。
 */
function nextCheckText(item: SupplierAccountHealthGuardItem): string {
  if (!item.next_check_at) {
    // 每轮都测没有「下次」；不参与本轮检查的账号（不可用/未匹配/已停用）也没有间隔。两者都不显示第二行。
    return item.interval_seconds && item.interval_seconds > 0 ? '尚未检查过' : ''
  }
  const target = Date.parse(item.next_check_at)
  if (Number.isNaN(target)) return ''
  const remainingSeconds = Math.ceil((target - Date.now()) / 1000)
  if (remainingSeconds <= 0) return '已可检查'
  if (remainingSeconds >= 3600) return `还需 ${Math.ceil(remainingSeconds / 3600)}小时`
  if (remainingSeconds >= 60) return `还需 ${Math.ceil(remainingSeconds / 60)}分`
  return `还需 ${remainingSeconds}秒`
}
</script>

<style scoped>
/* 变量不在组件内声明：本组件只嵌在 .sp-run-detail 内，那套 --sp-* 由页面挂在
   :global(.modal-content:has(.sp-run-detail)) 上（浅/暗各一份）。在这里再声明一份
   会和同弹窗内其它分支的配色对不上。若将来把本组件挪到 .sp-run-detail 之外，
   必须自己补一份兜底变量，否则颜色会静默失效。 */
.sp-ahg {
  display: grid;
  gap: 16px;
  margin-top: 16px;
  min-width: 0;
}

.sp-ahg-progress {
  display: grid;
  gap: 4px;
  margin: 0;
  border-left: 3px solid var(--sp-blue);
  border-radius: 0 10px 10px 0;
  padding: 10px 14px;
  background: var(--sp-result-blue-soft);
}

.sp-ahg-progress-main {
  color: var(--sp-text);
  font-size: 13px;
  font-weight: 700;
}

.sp-ahg-progress-note {
  color: var(--sp-muted);
  font-size: 11px;
  line-height: 1.6;
}

.sp-ahg-metrics {
  display: grid;
  grid-template-columns: repeat(10, minmax(0, 1fr));
  border-top: 1px solid var(--sp-line);
  border-bottom: 1px solid var(--sp-line);
}

.sp-ahg-metric {
  min-width: 0;
  border-left: 1px solid var(--sp-line);
  padding: 12px;
}

.sp-ahg-metric:first-child {
  border-left: 0;
}

.sp-ahg-metric > span,
.sp-ahg-metric-button > span {
  display: block;
  color: var(--sp-muted);
  font-size: 11px;
}

.sp-ahg-metric > strong,
.sp-ahg-metric-button > strong {
  display: block;
  margin-top: 5px;
  color: var(--sp-text);
  font-size: 18px;
}

.sp-ahg-metric-button {
  display: block;
  width: 100%;
  margin: -12px;
  padding: 12px;
  border: 0;
  background: transparent;
  color: inherit;
  font: inherit;
  text-align: left;
  cursor: pointer;
  transition: background-color 0.15s ease;
}

.sp-ahg-metric-button:hover {
  background: color-mix(in srgb, var(--sp-blue) 8%, transparent);
}

.sp-ahg-metric-button:focus-visible {
  outline: 2px solid color-mix(in srgb, var(--sp-blue) 55%, transparent);
  outline-offset: -2px;
}

.sp-ahg-metric.is-filterable {
  position: relative;
}

.sp-ahg-metric.is-filterable::after {
  content: '';
  position: absolute;
  inset: auto 10px 8px 10px;
  height: 2px;
  border-radius: 999px;
  background: transparent;
  transition: background-color 0.15s ease;
}

.sp-ahg-metric.is-filterable.good.active,
.sp-ahg-metric.is-filterable.good:hover {
  background: color-mix(in srgb, var(--sp-green) 8%, transparent);
}

.sp-ahg-metric.is-filterable.warn.active,
.sp-ahg-metric.is-filterable.warn:hover {
  background: color-mix(in srgb, var(--sp-amber) 8%, transparent);
}

.sp-ahg-metric.is-filterable.bad.active,
.sp-ahg-metric.is-filterable.bad:hover {
  background: color-mix(in srgb, var(--sp-red) 8%, transparent);
}

.sp-ahg-metric.is-filterable.neutral.active,
.sp-ahg-metric.is-filterable.neutral:hover {
  background: color-mix(in srgb, var(--sp-blue) 8%, transparent);
}

.sp-ahg-metric.good.active .sp-ahg-metric-button > strong {
  color: var(--sp-green);
}

.sp-ahg-metric.warn.active .sp-ahg-metric-button > strong {
  color: var(--sp-amber);
}

.sp-ahg-metric.bad.active .sp-ahg-metric-button > strong {
  color: var(--sp-red);
}

.sp-ahg-metric.neutral.active .sp-ahg-metric-button > strong {
  color: var(--sp-blue);
}

.sp-ahg-metric.is-filterable.active::after {
  background: currentColor;
}

.sp-ahg-metric.good.active::after {
  color: var(--sp-green);
}

.sp-ahg-metric.warn.active::after {
  color: var(--sp-amber);
}

.sp-ahg-metric.bad.active::after {
  color: var(--sp-red);
}

.sp-ahg-metric.neutral.active::after {
  color: var(--sp-blue);
}

.sp-ahg-block {
  display: grid;
  gap: 12px;
  min-width: 0;
  border-top: 1px solid var(--sp-line);
  padding-top: 16px;
}

.sp-ahg-block-head {
  display: flex;
  align-items: flex-end;
  justify-content: space-between;
  gap: 8px;
}

.sp-ahg-block-head h4 {
  margin: 0;
  color: var(--sp-text);
  font-size: 14px;
}

.sp-ahg-block-head > strong {
  color: var(--sp-muted);
  font-size: 12px;
  font-weight: 400;
}

.sp-ahg-reasons {
  display: grid;
  grid-template-columns: repeat(3, minmax(0, 1fr));
  gap: 10px;
}

.sp-ahg-reasons article {
  display: grid;
  grid-template-columns: minmax(0, 1fr) auto;
  gap: 4px 10px;
  border: 1px solid var(--sp-line);
  border-radius: 10px;
  padding: 12px;
  background: var(--sp-panel-2);
}

.sp-ahg-reasons strong,
.sp-ahg-reasons span {
  color: var(--sp-text);
  font-size: 12px;
}

.sp-ahg-reasons small {
  grid-column: 1 / -1;
  color: var(--sp-muted);
  font-size: 11px;
  line-height: 1.5;
}

.sp-ahg-toolbar {
  display: flex;
  flex-wrap: wrap;
  gap: 10px;
  align-items: center;
}

.sp-ahg-field {
  min-width: 0;
}

.sp-ahg-search {
  flex: 1 1 220px;
}

.sp-ahg-toolbar .sp-ahg-field:not(.sp-ahg-search) {
  flex: 0 0 150px;
}

.sp-ahg-healthy-toggle {
  flex: 0 0 auto;
}

.sp-ahg-hint {
  margin: 0;
  color: var(--sp-muted);
  font-size: 11px;
  line-height: 1.6;
}

.sp-ahg-sections {
  display: grid;
  gap: 16px;
  min-width: 0;
}

.sp-ahg-section {
  display: grid;
  gap: 10px;
  min-width: 0;
}

.sp-ahg-section-head {
  display: flex;
  align-items: baseline;
  justify-content: space-between;
  gap: 8px;
  border-bottom: 1px solid var(--sp-line);
  padding-bottom: 8px;
}

.sp-ahg-section-head h5 {
  margin: 0;
  color: var(--sp-text);
  font-size: 13px;
}

.sp-ahg-section-head span {
  color: var(--sp-muted);
  font-size: 11px;
}

.sp-ahg-items {
  display: grid;
  gap: 8px;
  min-width: 0;
}

/* 明细是一张 9 列表格：行内不再套卡片，用 border-bottom 分隔即可。
   窄屏靠这里横向滚动，而不是把一行账号堆成九行 —— 堆叠后完全看不出
   同一行的几个数字是一组。 */
.sp-ahg-table {
  min-width: 0;
  overflow-x: auto;
  border-top: 1px solid var(--sp-line);
}

.sp-ahg-row {
  display: grid;
  grid-template-columns:
    minmax(130px, 1fr)
    88px
    minmax(150px, 1.15fr)
    minmax(124px, 0.85fr)
    108px
    minmax(104px, 0.85fr)
    minmax(132px, 1.05fr)
    92px
    minmax(210px, 1.75fr);
  /* 列宽之和：窄于此就横向滚动。弹窗上限是 1280px，内容区约 1213px，
     所以列宽之和必须留在 1213px 以内，否则在**最宽**的弹窗上也要横向滚。
     ⚠️ 定宽度必须给够，否则会静默把行撑高 —— 实测「延迟」列给 88px 时，
     11px 的「阈值 15000ms」在 64px 的内容宽里折成两行，于是**每一行**都至少 73px，
     比同模块的 .sp-rate-guard-row（约 52px）高出一大截。
     各列最小值都按「不折行所需宽度」实测得来（量法：把单元格文字复制进一个 nowrap 的
     隐藏标尺读宽度，再加两侧各 12px 内边距），不是按字数估的：
       · 「平台 / 模型」实测最长的 model_id「claude-sonnet-4.6」需 95px ⇒ 119px，
         这里给 124px。⚠️ 这一列最容易被挤到折行：它只差 0.7px 就会从 1 行变 2 行，
         而折行的代价是**整行**从 57px 涨到 73px。加「测试间隔」列时曾把它挤到
         118px，两行账号因此变高 —— 补宽度补的就是这里。
       · 「延迟」实测「阈值 15000ms」需 73px ⇒ 97px，给 108px。
       · 「测试间隔」需 78px（上行「每轮都测」48px、下行「还需 12小时」61px）⇒ 给 92px。
       最右侧的「原因 / 错误」是唯一承载自由文本的列，所以分到的份额最大：
       它是唯一会随内容长高的列，给窄了会把单行撑到 160px 以上。
       「供应商来源」会随上游账号名折行（实测最长一条需 180px），这是内容本身决定的，
       与列宽无关 —— 上游名长的账号本来就该多占一行。 */
  min-width: 1138px;
  border-bottom: 1px solid var(--sp-line);
}

.sp-ahg-row > span {
  display: block;
  min-width: 0;
  padding: 9px 12px;
  color: var(--sp-text);
  font-size: 12px;
  word-break: break-word;
}

.sp-ahg-row > span strong {
  display: block;
  color: var(--sp-text);
  font-size: 12px;
}

.sp-ahg-row > span small {
  display: block;
  color: var(--sp-muted);
  font-size: 11px;
}

/* 单元格里第二行起统一 3px，省得每格各写一套 margin。 */
.sp-ahg-row > span > strong + small,
.sp-ahg-row > span > small + small {
  margin-top: 3px;
}

.sp-ahg-row-head > span {
  color: var(--sp-muted);
  font-size: 11px;
  font-weight: 700;
}

/* 异常行整行着色。原来卡片是 border-left 竖条，表格行连成一排后
   竖条会把列切开，改用整行底色区分。 */
.sp-ahg-row.bad {
  background: color-mix(in srgb, var(--sp-red) 4%, transparent);
}

.sp-ahg-row.warn {
  background: color-mix(in srgb, var(--sp-amber) 4%, transparent);
}

/* 来源是识别账号的关键信息，用正文色而不是弱化色。
   选择器要带上 .sp-ahg-row，否则压不过上面那条 .sp-ahg-row > span small。 */
.sp-ahg-row > .sp-ahg-sources small {
  color: var(--sp-text);
}

/* 原因 / 错误是行内最长的一列，折行不截断 —— 截断后关键错误信息会看不全。
   这两条同样要带 .sp-ahg-row 前缀才能压过单元格基色。 */
.sp-ahg-row > .sp-ahg-message {
  display: grid;
  gap: 3px;
  color: var(--sp-amber);
}

.sp-ahg-row > .sp-ahg-message.bad {
  color: var(--sp-red);
}

.sp-ahg-collapsed {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 10px;
  border: 1px dashed var(--sp-line);
  border-radius: 10px;
  padding: 10px 14px;
  color: var(--sp-muted);
  font-size: 12px;
}

.sp-ahg-empty {
  border-top: 1px solid var(--sp-line);
  border-bottom: 1px solid var(--sp-line);
  color: var(--sp-muted);
  padding: 16px 0;
}

@media (max-width: 1200px) {
  .sp-ahg-metrics {
    grid-template-columns: repeat(5, minmax(0, 1fr));
  }

  .sp-ahg-metric:nth-child(5n + 1) {
    border-left: 0;
  }
}

@media (max-width: 720px) {
  .sp-ahg-metrics {
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }

  .sp-ahg-metric:nth-child(2n + 1) {
    border-left: 0;
  }

  .sp-ahg-reasons {
    grid-template-columns: minmax(0, 1fr);
  }
}
</style>

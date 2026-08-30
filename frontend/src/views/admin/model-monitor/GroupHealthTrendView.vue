<template>
  <AppLayout>
    <main class="group-health-page">
      <section class="gh-toolbar" aria-label="分组健康趋势筛选">
        <div class="gh-range-switcher" role="group" aria-label="时间范围">
          <button
            v-for="item in rangeOptions"
            :key="item.value"
            type="button"
            class="gh-range-button"
            :class="{ 'gh-range-button-active': selectedRange === item.value }"
            :data-test="`range-${item.value}`"
            @click="changeRange(item.value)"
          >
            {{ item.label }}
          </button>
        </div>
        <div class="gh-filter-group">
          <Select
            v-model="selectedPlatform"
            :options="platformOptions"
            placeholder="全部平台"
            class="w-40"
            :clearable="false"
            aria-label="平台筛选"
            @change="onPlatformChange"
          />
          <Select
            v-model="selectedGroupId"
            :options="groupOptions"
            placeholder="全部分组"
            class="w-48"
            :clearable="false"
            aria-label="分组筛选"
            @change="onGroupChange"
          />
          <button type="button" class="gh-refresh-button" :disabled="loading" data-test="refresh" @click="loadData">
            <svg :class="{ 'gh-spin': loading }" viewBox="0 0 24 24" aria-hidden="true">
              <path d="M20 11a8 8 0 1 0 2 5.3" />
              <path d="M20 4v7h-7" />
            </svg>
            {{ loading ? '读取中' : '刷新' }}
          </button>
        </div>
      </section>

      <p v-if="errorMessage" class="gh-load-error" role="status">{{ errorMessage }}</p>

      <section class="gh-kpi-grid" aria-label="整体健康指标">
        <article class="gh-kpi gh-kpi-primary">
          <div class="gh-kpi-icon"><svg viewBox="0 0 24 24" aria-hidden="true"><path d="M4 19V5m0 14h16M7 15l3-4 3 2 5-7" /></svg></div>
          <div><span>服务健康成功率</span><strong>{{ formatPercent(summary.serviceSuccessRate) }}</strong><small>排除业务限制</small></div>
        </article>
        <article class="gh-kpi">
          <div class="gh-kpi-icon gh-kpi-icon-blue"><svg viewBox="0 0 24 24" aria-hidden="true"><path d="M4 6h16M4 12h16M4 18h10" /></svg></div>
          <div><span>请求总量</span><strong>{{ formatCount(summary.requestCount) }}</strong><small>{{ healthItems.length }} 个活跃分组</small></div>
        </article>
        <article class="gh-kpi">
          <div class="gh-kpi-icon gh-kpi-icon-amber"><svg viewBox="0 0 24 24" aria-hidden="true"><path d="M12 3 2.8 19h18.4L12 3Zm0 6v4m0 3h.01" /></svg></div>
          <div><span>整体错误率</span><strong>{{ formatPercent(summary.errorRate) }}</strong><small>{{ formatCount(summary.serviceErrorCount) }} 次服务错误</small></div>
        </article>
        <article class="gh-kpi">
          <div class="gh-kpi-icon gh-kpi-icon-violet"><svg viewBox="0 0 24 24" aria-hidden="true"><path d="M4 17h3V7H4v10Zm6 0h3V4h-3v13Zm6 0h3v-6h-3v6Z" /></svg></div>
          <div><span>最高 P95 延迟</span><strong>{{ formatLatency(summary.p95Latency) }}</strong><small>分组尾部响应时间</small></div>
        </article>
        <article class="gh-kpi gh-kpi-business">
          <div class="gh-kpi-icon gh-kpi-icon-rose"><svg viewBox="0 0 24 24" aria-hidden="true"><path d="M12 3v18M17 7.5c0-1.7-2.2-3-5-3S7 5.8 7 7.5s2.2 3 5 3 5 1.3 5 3-2.2 3-5 3-5-1.3-5-3" /></svg></div>
          <div><span>业务限制</span><strong>{{ formatCount(summary.businessLimitedCount) }}</strong><small>单独观察，不计入健康失败</small></div>
        </article>
      </section>

      <section class="gh-section-heading">
        <div>
          <p class="gh-section-kicker">GROUP PULSE</p>
          <h2>分组状态</h2>
        </div>
        <span class="gh-section-caption">{{ rangeCaption }} · {{ lastUpdatedLabel }}</span>
      </section>

      <section v-if="loading && !hasLoadedOnce" class="gh-loading-grid" aria-label="正在加载">
        <div v-for="index in 3" :key="index" class="gh-skeleton-card"><span /><span /><span /></div>
      </section>

      <section v-else-if="healthItems.length" class="gh-groups-list">
        <article
          v-for="item in healthItems"
          :key="item.group_id"
          class="gh-group-card"
          :class="{ 'gh-group-card-expanded': expandedGroupId === item.group_id }"
        >
          <button type="button" class="gh-group-summary" @click="toggleGroup(item.group_id)">
            <span class="gh-group-status" :class="`gh-status-${statusTone(item.status)}`"><i />{{ statusLabel(item.status) }}</span>
            <span class="gh-group-identity">
              <strong>{{ item.group_name }}</strong>
              <small>#{{ item.group_id }} · {{ displayPlatformLabel(item.effective_platform || item.platform) }}</small>
            </span>
            <span class="gh-group-stat"><small>健康成功率</small><strong>{{ formatPercent(item.service_success_rate) }}</strong></span>
            <span class="gh-group-stat gh-group-stat-muted"><small>请求</small><strong>{{ formatCount(item.request_count) }}</strong></span>
            <span class="gh-group-stat gh-group-stat-muted"><small>P95</small><strong>{{ formatLatency(item.p95_latency_ms) }}</strong></span>
            <span class="gh-group-chevron" :class="{ 'gh-group-chevron-open': expandedGroupId === item.group_id }">⌄</span>
          </button>

          <div v-if="expandedGroupId === item.group_id" class="gh-group-detail">
            <p v-if="item.status === 'no_data'" class="gh-no-data-hint">
              当前范围暂无请求记录，先展示分组状态；切换时间范围可查看已有记录。
            </p>
            <div class="gh-detail-meta">
              <span>成功 {{ formatCount(item.success_count) }}</span>
              <span>服务错误 {{ formatCount(item.service_error_count) }}</span>
              <span>业务限制 {{ formatCount(item.business_limited_count) }}</span>
              <span>平均 {{ formatLatency(item.avg_latency_ms) }}</span>
              <span>首 token P95 {{ formatLatency(item.p95_first_token_ms) }}</span>
              <span>最近请求 {{ formatDateTime(item.last_request_at) }}</span>
            </div>
            <div class="gh-chart-grid">
              <GroupHealthSuccessTrendChart :points="item.trend" :loading="loading" :time-range="selectedRange" />
              <GroupHealthLatencyTrendChart :points="item.trend" :loading="loading" :time-range="selectedRange" />
              <GroupHealthErrorDistributionChart :errors="item.top_errors" :loading="loading" />
            </div>
          </div>
        </article>
      </section>

      <section v-else class="gh-empty-state">
        <div class="gh-empty-orbit"><span /><span /><span /></div>
        <h3>{{ errorMessage ? '暂时无法读取分组数据' : '当前范围暂无分组数据' }}</h3>
        <p>{{ errorMessage ? '请确认服务状态后重试。' : '可以切换时间范围，或检查分组是否已开启模型监控展示。' }}</p>
        <button type="button" class="gh-empty-button" @click="loadData">重新读取</button>
      </section>
    </main>
  </AppLayout>
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import Select from '@/components/common/Select.vue'
import AppLayout from '@/components/layout/AppLayout.vue'
import { getAll } from '@/api/admin/groups'
import {
  getModelMonitorGroupHealth,
  type ModelMonitorGroupHealthItem,
  type ModelMonitorGroupHealthRange,
} from '@/api/admin/modelMonitorGroupHealth'
import {
  listLLMMonitorGroupPlatformOverrides,
  type LLMMonitorGroupPlatformOverride,
} from '@/api/admin/modelMonitor'
import type { AdminGroup } from '@/types'
import { useAppStore } from '@/stores/app'
import { extractApiErrorMessage } from '@/utils/apiError'
import GroupHealthSuccessTrendChart from './components/GroupHealthSuccessTrendChart.vue'
import GroupHealthLatencyTrendChart from './components/GroupHealthLatencyTrendChart.vue'
import GroupHealthErrorDistributionChart from './components/GroupHealthErrorDistributionChart.vue'

const appStore = useAppStore()
const selectedRange = ref<ModelMonitorGroupHealthRange>('24h')
const selectedPlatform = ref<string | null>(null)
const selectedGroupId = ref<number | null>(null)
const groups = ref<AdminGroup[]>([])
const platformOverrides = ref<LLMMonitorGroupPlatformOverride[]>([])
const healthItems = ref<ModelMonitorGroupHealthItem[]>([])
const expandedGroupId = ref<number | null>(null)
const loading = ref(false)
const hasLoadedOnce = ref(false)
const errorMessage = ref('')
const lastUpdated = ref<Date | null>(null)

const rangeOptions: Array<{ value: ModelMonitorGroupHealthRange; label: string }> = [
  { value: '1h', label: '1H' },
  { value: '24h', label: '24H' },
  { value: '7d', label: '7D' },
  { value: '30d', label: '30D' },
]

const platformLabels: Record<string, string> = {
  openai: 'OpenAI',
  anthropic: 'Anthropic',
  gemini: 'Gemini',
  antigravity: 'Antigravity',
  grok: 'Grok',
  deepseek: 'DeepSeek',
}

const platformOverrideByGroupId = computed(() => new Map(
  platformOverrides.value.map((override) => [override.id, override]),
))

function effectivePlatformForGroup(group: AdminGroup) {
  return String(platformOverrideByGroupId.value.get(group.id)?.effective_platform || group.platform || '').trim().toLowerCase()
}

function isVisibleInModelMonitor(group: AdminGroup) {
  return platformOverrideByGroupId.value.get(group.id)?.show_in_monitor !== false
}

const platformLabelByValue = computed(() => {
  const labels = new Map<string, string>()
  for (const group of groups.value) {
    const platform = effectivePlatformForGroup(group)
    if (!platform) continue
    const override = platformOverrideByGroupId.value.get(group.id)
    labels.set(platform, override?.effective_platform_name || platformLabel(platform))
  }
  return labels
})

const platformOptions = computed(() => [
  { value: null, label: '全部平台' },
  ...Array.from(new Set(groups.value.filter(isVisibleInModelMonitor).map(effectivePlatformForGroup).filter(Boolean)))
    .sort()
    .map((platform) => ({ value: platform, label: displayPlatformLabel(platform) })),
])

const groupOptions = computed(() => [
  { value: null, label: '全部分组' },
  ...groups.value
    .filter(isVisibleInModelMonitor)
    .filter((group) => !selectedPlatform.value || effectivePlatformForGroup(group) === selectedPlatform.value)
    .sort((a, b) => a.name.localeCompare(b.name, 'zh-CN'))
    .map((group) => ({ value: group.id, label: `${group.name} #${group.id}` })),
])

function buildNoDataHealthItems(groupList: AdminGroup[]) {
  return groupList
    .filter(isVisibleInModelMonitor)
    .filter((group) => !selectedPlatform.value || effectivePlatformForGroup(group) === selectedPlatform.value)
    .filter((group) => selectedGroupId.value === null || group.id === selectedGroupId.value)
    .map((group) => ({
      group_id: group.id,
      group_name: group.name,
      platform: group.platform || '',
      effective_platform: effectivePlatformForGroup(group),
      request_count: 0,
      success_count: 0,
      error_count: 0,
      business_limited_count: 0,
      service_error_count: 0,
      success_rate: 0,
      service_success_rate: 0,
      error_rate: 0,
      avg_latency_ms: 0,
      p95_latency_ms: 0,
      p95_first_token_ms: 0,
      status: 'no_data' as const,
      last_request_at: null,
      trend: [],
      top_errors: [],
    }))
}

const summary = computed(() => {
  const requestCount = healthItems.value.reduce((sum, item) => sum + item.request_count, 0)
  const serviceErrorCount = healthItems.value.reduce((sum, item) => sum + item.service_error_count, 0)
  const serviceSuccessCount = healthItems.value.reduce((sum, item) => sum + item.success_count, 0)
  const businessLimitedCount = healthItems.value.reduce((sum, item) => sum + item.business_limited_count, 0)
  const errorCount = healthItems.value.reduce((sum, item) => sum + item.error_count, 0)
  const p95Latency = healthItems.value.reduce((max, item) => Math.max(max, item.p95_latency_ms), 0)
  const serviceSamples = serviceSuccessCount + serviceErrorCount
  return {
    requestCount,
    serviceErrorCount,
    businessLimitedCount,
    serviceSuccessRate: serviceSamples ? (serviceSuccessCount / serviceSamples) * 100 : 0,
    errorRate: requestCount ? (errorCount / requestCount) * 100 : 0,
    p95Latency,
  }
})

const rangeCaption = computed(() => ({ '1h': '最近 1 小时', '24h': '最近 24 小时', '7d': '最近 7 天', '30d': '最近 30 天' })[selectedRange.value])
const lastUpdatedLabel = computed(() => lastUpdated.value ? `更新于 ${lastUpdated.value.toLocaleTimeString('zh-CN', { hour: '2-digit', minute: '2-digit' })}` : '等待首次读取')

async function loadData() {
  loading.value = true
  errorMessage.value = ''
  try {
    const [groupResult, overrideResult, healthResult] = await Promise.allSettled([
      getAll(),
      listLLMMonitorGroupPlatformOverrides(),
      getModelMonitorGroupHealth({
        range: selectedRange.value,
        groupIds: selectedGroupId.value === null ? [] : [selectedGroupId.value],
        platform: selectedPlatform.value ?? undefined,
      }),
    ])

    if (groupResult.status === 'fulfilled') groups.value = groupResult.value
    if (overrideResult.status === 'fulfilled') platformOverrides.value = overrideResult.value

    const fallbackGroups = groupResult.status === 'fulfilled' ? groupResult.value : groups.value
    if (healthResult.status === 'fulfilled') {
      const items = Array.isArray(healthResult.value) ? healthResult.value : []
      healthItems.value = items.length ? items : buildNoDataHealthItems(fallbackGroups)
      lastUpdated.value = new Date()
      hasLoadedOnce.value = true
    } else {
      // 健康趋势接口暂不可用时，仍展示已成功读取的分组，避免整个页面变成空状态。
      // 卡片上的“无数据”只代表当前无法拿到趋势，不把接口失败误判为健康。
      healthItems.value = buildNoDataHealthItems(fallbackGroups)
      errorMessage.value = '加载分组健康趋势失败'
      appStore.showError(extractApiErrorMessage(healthResult.reason, errorMessage.value))
      hasLoadedOnce.value = true
    }

    if (expandedGroupId.value === null || !healthItems.value.some((item) => item.group_id === expandedGroupId.value)) {
      expandedGroupId.value = healthItems.value[0]?.group_id ?? null
    }
  } catch (error) {
    healthItems.value = []
    expandedGroupId.value = null
    errorMessage.value = '加载分组健康趋势失败'
    appStore.showError(extractApiErrorMessage(error, errorMessage.value))
  } finally {
    loading.value = false
  }
}

async function changeRange(range: ModelMonitorGroupHealthRange) {
  if (range === selectedRange.value) return
  selectedRange.value = range
  await loadData()
}

async function onPlatformChange(value: string | number | boolean | null) {
  selectedPlatform.value = typeof value === 'string' ? value : null
  selectedGroupId.value = null
  await loadData()
}

async function onGroupChange(value: string | number | boolean | null) {
  selectedGroupId.value = typeof value === 'number' ? value : value ? Number(value) : null
  await loadData()
}

function toggleGroup(groupId: number) {
  expandedGroupId.value = expandedGroupId.value === groupId ? null : groupId
}

function platformLabel(platform: string) {
  return platformLabels[platform] ?? (platform || '未知平台')
}

function displayPlatformLabel(platform: string) {
  return platformLabelByValue.value.get(platform) || platformLabel(platform)
}

function statusTone(status: string) {
  if (status === 'healthy') return 'healthy'
  if (status === 'critical') return 'critical'
  if (status === 'warning') return 'warning'
  if (status === 'low_sample') return 'low-sample'
  return 'no-data'
}

function statusLabel(status: string) {
  return ({ healthy: '健康', warning: '关注', critical: '告警', low_sample: '样本偏少', no_data: '无数据' } as Record<string, string>)[status] ?? '未知'
}

function formatCount(value: number) {
  return new Intl.NumberFormat('zh-CN', { maximumFractionDigits: 0 }).format(value || 0)
}

function formatPercent(value: number) {
  return `${(value || 0).toFixed(2)}%`
}

function formatLatency(value: number) {
  if (!value) return '—'
  return value >= 1000 ? `${(value / 1000).toFixed(2)} s` : `${Math.round(value)} ms`
}

function formatDateTime(value: string | null) {
  if (!value) return '暂无'
  const date = new Date(value)
  if (Number.isNaN(date.getTime())) return '暂无'
  return date.toLocaleString('zh-CN', { month: '2-digit', day: '2-digit', hour: '2-digit', minute: '2-digit' })
}

onMounted(loadData)
</script>

<style scoped>
.group-health-page {
  --gh-bg: #f5f8fb;
  --gh-surface: rgba(255, 255, 255, .86);
  --gh-surface-muted: rgba(248, 251, 253, .84);
  --gh-line: #dce6ee;
  --gh-ink: #102a43;
  --gh-muted: #6b8296;
  --gh-teal: #0f9f93;
  --gh-blue: #2684ff;
  --gh-amber: #e49a1b;
  --gh-rose: #ed6a7e;
  background:
    radial-gradient(circle at 8% 0%, rgba(45, 212, 191, .10), transparent 25rem),
    radial-gradient(circle at 100% 17%, rgba(56, 189, 248, .11), transparent 24rem),
    var(--gh-bg);
  border-radius: 28px;
  color: var(--gh-ink);
  min-height: calc(100vh - 2rem);
  overflow: hidden;
  padding: 20px;
}

:global(.dark) .group-health-page {
  --gh-bg: #09131f;
  --gh-surface: rgba(17, 31, 48, .88);
  --gh-surface-muted: rgba(13, 26, 42, .82);
  --gh-line: #23364c;
  --gh-ink: #e5f1f6;
  --gh-muted: #8ba1b5;
  --gh-teal: #2dd4bf;
  --gh-blue: #60a5fa;
  --gh-amber: #fbbf24;
  --gh-rose: #fb7185;
  background:
    radial-gradient(circle at 8% 0%, rgba(20, 184, 166, .13), transparent 25rem),
    radial-gradient(circle at 100% 17%, rgba(37, 99, 235, .13), transparent 24rem),
    var(--gh-bg);
}

.gh-toolbar,
.gh-kpi,
.gh-group-card,
.gh-empty-state {
  border: 1px solid var(--gh-line);
  background: var(--gh-surface);
  box-shadow: 0 18px 50px rgba(15, 42, 67, .055);
}

:global(.dark) .gh-toolbar,
:global(.dark) .gh-kpi,
:global(.dark) .gh-group-card,
:global(.dark) .gh-empty-state {
  box-shadow: 0 20px 60px rgba(0, 0, 0, .18);
}

.gh-toolbar { align-items: center; border-radius: 18px; display: flex; justify-content: space-between; margin-top: 16px; padding: 12px; }
.gh-range-switcher { background: color-mix(in srgb, var(--gh-line) 35%, transparent); border-radius: 11px; display: flex; gap: 3px; padding: 3px; }
.gh-range-button { background: transparent; border: 0; border-radius: 8px; color: var(--gh-muted); cursor: pointer; font-size: 11px; font-weight: 800; letter-spacing: .05em; min-width: 49px; padding: 9px 11px; transition: color .2s, background .2s, transform .2s; }
.gh-range-button:hover { color: var(--gh-ink); transform: translateY(-1px); }
.gh-range-button-active { background: var(--gh-surface); box-shadow: 0 3px 10px rgba(15, 42, 67, .10); color: var(--gh-teal); }
.gh-filter-group { align-items: center; display: flex; gap: 9px; }
.gh-refresh-button, .gh-empty-button { align-items: center; background: var(--gh-ink); border: 0; border-radius: 10px; color: var(--gh-bg); cursor: pointer; display: inline-flex; font-size: 12px; font-weight: 800; gap: 7px; justify-content: center; padding: 10px 14px; transition: transform .2s, opacity .2s; }
.gh-refresh-button:hover:not(:disabled), .gh-empty-button:hover { transform: translateY(-1px); }
.gh-refresh-button:disabled { cursor: wait; opacity: .55; }
.gh-refresh-button svg { fill: none; height: 14px; stroke: currentColor; stroke-linecap: round; stroke-linejoin: round; stroke-width: 1.8; width: 14px; }
.gh-spin { animation: gh-spin 1s linear infinite; }
.gh-load-error { background: color-mix(in srgb, var(--gh-rose) 10%, transparent); border: 1px solid color-mix(in srgb, var(--gh-rose) 28%, transparent); border-radius: 12px; color: var(--gh-rose); font-size: 12px; margin: 14px 0 0; padding: 11px 14px; }

.gh-kpi-grid { display: grid; gap: 12px; grid-template-columns: repeat(5, minmax(0, 1fr)); margin-top: 16px; }
.gh-kpi { align-items: flex-start; border-radius: 18px; display: flex; gap: 12px; min-width: 0; padding: 16px; }
.gh-kpi-primary { border-color: color-mix(in srgb, var(--gh-teal) 35%, var(--gh-line)); background: linear-gradient(135deg, color-mix(in srgb, var(--gh-teal) 10%, var(--gh-surface)), var(--gh-surface)); }
.gh-kpi-icon { align-items: center; background: color-mix(in srgb, var(--gh-teal) 13%, transparent); border-radius: 11px; color: var(--gh-teal); display: flex; flex: 0 0 auto; height: 34px; justify-content: center; width: 34px; }
.gh-kpi-icon-blue { background: color-mix(in srgb, var(--gh-blue) 13%, transparent); color: var(--gh-blue); }
.gh-kpi-icon-amber { background: color-mix(in srgb, var(--gh-amber) 13%, transparent); color: var(--gh-amber); }
.gh-kpi-icon-violet { background: rgba(167, 139, 250, .13); color: #8b5cf6; }
.gh-kpi-icon-rose { background: color-mix(in srgb, var(--gh-rose) 13%, transparent); color: var(--gh-rose); }
.gh-kpi-icon svg { fill: none; height: 17px; stroke: currentColor; stroke-linecap: round; stroke-linejoin: round; stroke-width: 1.7; width: 17px; }
.gh-kpi span, .gh-kpi small { color: var(--gh-muted); display: block; font-size: 10px; white-space: nowrap; }
.gh-kpi strong { display: block; font-size: 20px; letter-spacing: -.035em; line-height: 1.25; margin: 6px 0 2px; overflow: hidden; text-overflow: ellipsis; }
.gh-kpi small { font-size: 10px; overflow: hidden; text-overflow: ellipsis; }

.gh-section-heading { align-items: flex-end; display: flex; justify-content: space-between; margin: 30px 2px 13px; }
.gh-section-kicker { color: var(--gh-teal); font-size: 10px; font-weight: 800; letter-spacing: .16em; margin: 0 0 5px; }
.gh-section-heading h2 { font-size: 21px; letter-spacing: -.03em; margin: 0; }
.gh-section-caption { color: var(--gh-muted); font-size: 11px; }
.gh-groups-list { display: grid; gap: 10px; }
.gh-group-card { border-radius: 18px; overflow: hidden; transition: border-color .2s, box-shadow .2s; }
.gh-group-card-expanded { border-color: color-mix(in srgb, var(--gh-teal) 42%, var(--gh-line)); box-shadow: 0 16px 40px color-mix(in srgb, var(--gh-teal) 9%, transparent); }
.gh-group-summary { align-items: center; background: transparent; border: 0; color: inherit; cursor: pointer; display: grid; gap: 14px; grid-template-columns: 92px minmax(170px, 1fr) 150px 90px 100px 20px; padding: 16px 18px; text-align: left; width: 100%; }
.gh-group-status { align-items: center; border-radius: 999px; display: inline-flex; font-size: 10px; font-weight: 800; gap: 6px; justify-self: start; padding: 6px 9px; white-space: nowrap; }
.gh-group-status i { background: currentColor; border-radius: 50%; height: 6px; width: 6px; }
.gh-status-healthy { background: rgba(20, 184, 166, .12); color: var(--gh-teal); }
.gh-status-warning { background: rgba(245, 158, 11, .13); color: var(--gh-amber); }
.gh-status-critical { background: rgba(244, 63, 94, .13); color: var(--gh-rose); }
.gh-status-low-sample, .gh-status-no-data { background: color-mix(in srgb, var(--gh-muted) 13%, transparent); color: var(--gh-muted); }
.gh-group-identity { min-width: 0; }
.gh-group-identity strong { display: block; font-size: 14px; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.gh-group-identity small, .gh-group-stat small { color: var(--gh-muted); display: block; font-size: 10px; margin-top: 4px; }
.gh-group-stat strong { display: block; font-size: 15px; margin-top: 4px; }
.gh-group-stat-muted strong { color: var(--gh-muted); font-size: 13px; }
.gh-group-chevron { color: var(--gh-muted); font-size: 22px; line-height: 1; transform: rotate(0); transition: transform .2s; }
.gh-group-chevron-open { transform: rotate(180deg); }
.gh-group-detail { border-top: 1px solid var(--gh-line); padding: 14px 18px 18px; }
.gh-no-data-hint { background: color-mix(in srgb, var(--gh-muted) 8%, transparent); border: 1px dashed color-mix(in srgb, var(--gh-muted) 32%, transparent); border-radius: 10px; color: var(--gh-muted); font-size: 11px; margin: 0 0 12px; padding: 9px 11px; }
.gh-detail-meta { color: var(--gh-muted); display: flex; flex-wrap: wrap; font-size: 11px; gap: 8px 18px; margin-bottom: 14px; }
.gh-detail-meta span + span::before { color: var(--gh-line); content: '·'; margin-right: 18px; }
.gh-chart-grid { display: grid; gap: 12px; grid-template-columns: repeat(3, minmax(0, 1fr)); }
.gh-loading-grid { display: grid; gap: 10px; }
.gh-skeleton-card { animation: gh-pulse 1.5s ease-in-out infinite; background: color-mix(in srgb, var(--gh-line) 46%, transparent); border-radius: 18px; height: 76px; padding: 18px; }
.gh-skeleton-card span { background: color-mix(in srgb, var(--gh-line) 70%, transparent); border-radius: 5px; display: block; height: 9px; margin-bottom: 10px; width: 20%; }
.gh-skeleton-card span:nth-child(2) { width: 42%; }
.gh-skeleton-card span:nth-child(3) { width: 28%; }
.gh-empty-state { align-items: center; border-radius: 20px; display: flex; flex-direction: column; padding: 68px 20px; text-align: center; }
.gh-empty-orbit { align-items: center; border: 1px solid color-mix(in srgb, var(--gh-teal) 32%, transparent); border-radius: 50%; display: flex; height: 70px; justify-content: center; margin-bottom: 18px; position: relative; width: 70px; }
.gh-empty-orbit::before { border: 1px dashed color-mix(in srgb, var(--gh-blue) 40%, transparent); border-radius: 50%; content: ''; height: 44px; position: absolute; width: 44px; }
.gh-empty-orbit span { background: var(--gh-teal); border-radius: 50%; height: 6px; position: absolute; width: 6px; }
.gh-empty-orbit span:nth-child(1) { transform: translateX(27px); }
.gh-empty-orbit span:nth-child(2) { background: var(--gh-blue); transform: translate(-19px, -19px); }
.gh-empty-orbit span:nth-child(3) { background: var(--gh-amber); transform: translate(-18px, 21px); }
.gh-empty-state h3 { font-size: 16px; margin: 0 0 8px; }
.gh-empty-state p { color: var(--gh-muted); font-size: 12px; margin: 0 0 18px; }

@keyframes gh-spin { to { transform: rotate(360deg); } }
@keyframes gh-pulse { 0%, 100% { opacity: .55; } 50% { opacity: .95; } }

@media (max-width: 1120px) {
  .gh-kpi-grid { grid-template-columns: repeat(3, minmax(0, 1fr)); }
  .gh-kpi-business { grid-column: span 1; }
  .gh-chart-grid { grid-template-columns: repeat(2, minmax(0, 1fr)); }
  .gh-chart-grid > :last-child { grid-column: span 2; }
}

@media (max-width: 760px) {
  .group-health-page { border-radius: 18px; padding: 12px; }
  .gh-toolbar { align-items: stretch; flex-direction: column; gap: 12px; }
  .gh-filter-group { flex-wrap: wrap; }
  .gh-filter-group :deep(.select-container) { flex: 1 1 130px; }
  .gh-refresh-button { flex: 1 0 90px; }
  .gh-kpi-grid { grid-template-columns: repeat(2, minmax(0, 1fr)); }
  .gh-kpi { padding: 13px; }
  .gh-kpi strong { font-size: 18px; }
  .gh-section-heading { align-items: flex-start; flex-direction: column; gap: 8px; }
  .gh-group-summary { gap: 9px; grid-template-columns: 1fr auto; padding: 15px; }
  .gh-group-status { grid-column: 1; grid-row: 1; }
  .gh-group-identity { grid-column: 1; grid-row: 2; }
  .gh-group-stat { display: none; }
  .gh-group-chevron { grid-column: 2; grid-row: 1 / span 2; }
  .gh-chart-grid { grid-template-columns: 1fr; }
  .gh-chart-grid > :last-child { grid-column: auto; }
  .gh-detail-meta { gap: 6px 12px; }
  .gh-detail-meta span + span::before { margin-right: 12px; }
}
</style>

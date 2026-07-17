<template>
  <AppLayout>
    <div class="automation-page">
      <header class="automation-hero">
        <div>
          <span class="automation-kicker">{{ t('admin.upstreamAutomations.eyebrow') }}</span>
          <h1>{{ t('admin.upstreamAutomations.title') }}</h1>
          <p>{{ t('admin.upstreamAutomations.description') }}</p>
        </div>
        <button type="button" class="automation-refresh" :disabled="loading" @click="loadTasks">
          <Icon name="refresh" size="sm" :class="loading ? 'animate-spin' : ''" />
          {{ t('common.refresh') }}
        </button>
      </header>

      <section class="automation-summary">
        <div><span>{{ t('admin.upstreamAutomations.totalTasks') }}</span><strong>{{ tasks.length }}</strong></div>
        <div><span>{{ t('admin.upstreamAutomations.enabledTasks') }}</span><strong>{{ enabledCount }}</strong></div>
        <div><span>{{ t('admin.upstreamAutomations.failedTasks') }}</span><strong :class="failedCount ? 'is-danger' : 'is-good'">{{ failedCount }}</strong></div>
        <div><span>{{ t('admin.upstreamAutomations.runningTasks') }}</span><strong>{{ runningKeys.size }}</strong></div>
      </section>

      <div v-if="warnings.length" class="automation-warning">
        <Icon name="exclamationTriangle" size="sm" />
        {{ t('admin.upstreamAutomations.partialLoad', { count: warnings.length }) }}
      </div>

      <div v-if="loading && !tasks.length" class="automation-loading">
        <span></span>{{ t('common.loading') }}
      </div>

      <section v-else class="automation-grid">
        <article
          v-for="task in tasks"
          :key="task.key"
          :data-test="`automation-card-${task.key}`"
          :class="['automation-card', `tone-${task.tone}`]"
        >
          <div class="card-rail"></div>
          <div class="card-header">
            <span class="task-icon"><Icon :name="task.icon" size="md" /></span>
            <span :class="['task-state', statusClass(task)]">{{ statusLabel(task) }}</span>
          </div>
          <div class="card-copy">
            <span class="task-kind">{{ task.kind }}</span>
            <h2>{{ task.name }}</h2>
            <p>{{ task.description }}</p>
          </div>
          <dl class="task-facts">
            <div><dt>{{ t('admin.upstreamAutomations.schedule') }}</dt><dd>{{ scheduleLabel(task) }}</dd></div>
            <div><dt>{{ t('admin.upstreamAutomations.lastRun') }}</dt><dd>{{ lastRunLabel(task) }}</dd></div>
            <div><dt>{{ t('admin.upstreamAutomations.lastResult') }}</dt><dd>{{ task.lastMessage || statusLabel(task) }}</dd></div>
          </dl>
          <div class="card-actions">
            <button
              :data-test="`automation-action-${task.key}`"
              type="button"
              class="task-primary"
              :disabled="runningKeys.has(task.key)"
              @click="runTask(task)"
            >
              <Icon :name="task.directRun ? 'play' : 'arrowRight'" size="sm" />
              {{ runningKeys.has(task.key) ? t('admin.upstreamAutomations.running') : task.actionLabel }}
            </button>
            <button type="button" class="task-secondary" @click="openTaskSettings(task)">
              <Icon name="cog" size="sm" />{{ t('common.settings') }}
            </button>
          </div>
        </article>
      </section>

      <section class="history-panel">
        <div class="history-header">
          <div>
            <span class="automation-kicker">{{ t('admin.upstreamAutomations.historyEyebrow') }}</span>
            <h2>{{ t('admin.upstreamAutomations.historyTitle') }}</h2>
            <p>{{ t('admin.upstreamAutomations.historyDescription') }}</p>
          </div>
          <span class="history-count">{{ filteredHistory.length }}</span>
        </div>

        <div class="history-filters">
          <select v-model="historyTaskFilter" data-test="history-task-filter">
            <option value="">{{ t('admin.upstreamAutomations.allTasks') }}</option>
            <option v-for="task in tasks" :key="task.key" :value="task.key">{{ task.name }}</option>
          </select>
          <select v-model="historyStatusFilter" data-test="history-status-filter">
            <option value="">{{ t('admin.upstreamAutomations.allStatuses') }}</option>
            <option value="success">{{ t('admin.upstreamAutomations.healthy') }}</option>
            <option value="failed">{{ t('admin.upstreamAutomations.failed') }}</option>
            <option value="skipped">{{ t('admin.upstreamAutomations.skipped') }}</option>
          </select>
          <select v-model="historyTriggerFilter" data-test="history-trigger-filter">
            <option value="">{{ t('admin.upstreamAutomations.allTriggers') }}</option>
            <option value="manual">{{ t('admin.upstreamAutomations.manualTrigger') }}</option>
            <option value="scheduled">{{ t('admin.upstreamAutomations.scheduledTrigger') }}</option>
            <option value="system">{{ t('admin.upstreamAutomations.systemTrigger') }}</option>
          </select>
        </div>

        <div v-if="filteredHistory.length" class="history-list">
          <button
            v-for="entry in filteredHistory"
            :key="entry.id"
            :data-test="`history-row-${entry.id}`"
            type="button"
            class="history-row"
            @click="selectedHistory = entry"
          >
            <span :class="['history-dot', `history-${entry.status}`]"></span>
            <span class="history-task"><strong>{{ taskName(entry.taskKey) }}</strong><small>{{ triggerLabel(entry.trigger) }}</small></span>
            <span class="history-message">{{ entry.message }}</span>
            <span class="history-impact">{{ entry.impact }}</span>
            <time>{{ formatDateTime(entry.time) }}</time>
            <Icon name="chevronRight" size="sm" />
          </button>
        </div>
        <div v-else class="history-empty">{{ t('admin.upstreamAutomations.noHistory') }}</div>
      </section>

      <div v-if="selectedHistory" class="history-detail-overlay" @click.self="selectedHistory = null">
        <aside data-test="history-detail" class="history-detail">
          <div class="history-detail-header">
            <div><span class="automation-kicker">{{ triggerLabel(selectedHistory.trigger) }}</span><h2>{{ taskName(selectedHistory.taskKey) }}</h2></div>
            <button type="button" :aria-label="t('common.close')" @click="selectedHistory = null"><Icon name="x" size="md" /></button>
          </div>
          <dl>
            <div><dt>{{ t('admin.upstreamAutomations.lastRun') }}</dt><dd>{{ formatDateTime(selectedHistory.time) }}</dd></div>
            <div><dt>{{ t('common.status') }}</dt><dd>{{ historyStatusLabel(selectedHistory.status) }}</dd></div>
            <div><dt>{{ t('admin.upstreamAutomations.impact') }}</dt><dd>{{ selectedHistory.impact }}</dd></div>
            <div><dt>{{ t('admin.upstreamAutomations.details') }}</dt><dd>{{ selectedHistory.message }}</dd></div>
          </dl>
          <div class="history-detail-actions">
            <button
              v-if="selectedHistory.status === 'failed'"
              data-test="history-retry"
              type="button"
              class="task-primary"
              :disabled="runningKeys.has(selectedHistory.taskKey)"
              @click="retryHistory(selectedHistory)"
            >
              <Icon name="refresh" size="sm" />{{ t('admin.upstreamAutomations.retry') }}
            </button>
            <button type="button" class="task-secondary" @click="openHistorySettings(selectedHistory)">
              <Icon name="cog" size="sm" />{{ t('common.settings') }}
            </button>
          </div>
        </aside>
      </div>
    </div>
  </AppLayout>
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { useRouter } from 'vue-router'
import { adminAPI } from '@/api/admin'
import AppLayout from '@/components/layout/AppLayout.vue'
import Icon from '@/components/icons/Icon.vue'
import { useAppStore } from '@/stores/app'

type AutomationKey = 'account-sync' | 'group-rate-fix' | 'account-rate-guard' | 'balance-sampler' | 'health-guard'
type AutomationTone = 'cyan' | 'amber' | 'violet' | 'emerald' | 'blue'

type AutomationIcon = 'sync' | 'shield' | 'chart'

interface AutomationTask {
  key: AutomationKey
  name: string
  kind: string
  description: string
  icon: AutomationIcon
  tone: AutomationTone
  enabled: boolean
  intervalSeconds?: number
  lastRunAt?: string
  lastStatus?: string
  lastMessage?: string
  directRun: boolean
  actionLabel: string
  actionPath: string
  settingsPath: string
}

type HistoryStatus = 'success' | 'failed' | 'skipped'
type HistoryTrigger = 'manual' | 'scheduled' | 'system'

interface AutomationHistoryEntry {
  id: string
  taskKey: AutomationKey
  status: HistoryStatus
  trigger: HistoryTrigger
  time: string
  message: string
  impact: string
}

const { t } = useI18n()
const router = useRouter()
const appStore = useAppStore()
const tasks = ref<AutomationTask[]>([])
const warnings = ref<string[]>([])
const loading = ref(false)
const runningKeys = ref(new Set<AutomationKey>())
const history = ref<AutomationHistoryEntry[]>([])
const historyTaskFilter = ref<AutomationKey | ''>('')
const historyStatusFilter = ref<HistoryStatus | ''>('')
const historyTriggerFilter = ref<HistoryTrigger | ''>('')
const selectedHistory = ref<AutomationHistoryEntry | null>(null)
const directTaskRunners: Partial<Record<AutomationKey, () => Promise<unknown>>> = {
  'account-rate-guard': () => adminAPI.upstreamAccountSync.runRateGuardNow(),
  'balance-sampler': () => adminAPI.upstreamAccountSync.runBalanceSampleNow(),
  'health-guard': () => adminAPI.upstreamAccountSync.runHealthGuardNow(),
}

const enabledCount = computed(() => tasks.value.filter(task => task.enabled).length)
const failedCount = computed(() => tasks.value.filter(task => task.lastStatus === 'failed').length)
const filteredHistory = computed(() => history.value.filter(entry => {
  if (historyTaskFilter.value && entry.taskKey !== historyTaskFilter.value) return false
  if (historyStatusFilter.value && entry.status !== historyStatusFilter.value) return false
  if (historyTriggerFilter.value && entry.trigger !== historyTriggerFilter.value) return false
  return true
}))

onMounted(loadTasks)

async function loadTasks() {
  loading.value = true
  warnings.value = []
  const results = await Promise.allSettled([
    adminAPI.upstreamAccountSync.getRecords(),
    adminAPI.upstreamManagement.getRateFixConfig(),
    adminAPI.upstreamAccountSync.getRateGuardConfig(),
    adminAPI.upstreamAccountSync.getRateGuardPollLogs(),
    adminAPI.upstreamAccountSync.getBalanceSamplerConfig(),
    adminAPI.upstreamAccountSync.getBalanceSamplerPollLogs(),
    adminAPI.upstreamAccountSync.getHealthGuardConfig(),
    adminAPI.upstreamAccountSync.getHealthGuardPollLogs(),
    adminAPI.upstreamManagement.getGroups(),
    adminAPI.upstreamAccountSync.getHealthGuardRecords(),
  ])

  const value = <T,>(index: number): T | undefined => {
    const result = results[index]
    if (result.status === 'fulfilled') return result.value as T
    warnings.value.push(String(result.reason instanceof Error ? result.reason.message : result.reason))
    return undefined
  }

  const syncRecords = value<any[]>(0) || []
  const rateFix = value<any>(1)
  const rateGuard = value<any>(2)
  const rateGuardLogs = value<any[]>(3) || []
  const balance = value<any>(4)
  const balanceLogs = value<any[]>(5) || []
  const health = value<any>(6)
  const healthLogs = value<any[]>(7) || []
  const groupResult = value<any>(8)
  const healthRecords = value<any[]>(9) || []
  const latestSync = syncRecords[0]

  tasks.value = [
    makeTask('account-sync', 'cyan', 'sync', false, true, latestSync?.created_at, latestSync?.error ? 'failed' : latestSync ? 'success' : '', latestSync?.error),
    makeTask('group-rate-fix', 'amber', 'sync', false, Boolean(rateFix?.enabled), rateFix?.last_run_at, rateFix?.last_run_status, rateFix?.last_run_message, rateFix?.interval_seconds),
    makeTask('account-rate-guard', 'violet', 'shield', true, Boolean(rateGuard?.enabled), rateGuard?.last_run_at || rateGuardLogs[0]?.checked_at, rateGuard?.last_run_status || rateGuardLogs[0]?.status, rateGuard?.last_run_message || rateGuardLogs[0]?.message, rateGuard?.interval_seconds),
    makeTask('balance-sampler', 'emerald', 'chart', true, Boolean(balance?.enabled), balance?.last_run_at || balanceLogs[0]?.checked_at, balance?.last_run_status || balanceLogs[0]?.status, balance?.last_run_message || balanceLogs[0]?.message, balance?.interval_seconds),
    makeTask('health-guard', 'blue', 'shield', true, Boolean(health?.enabled), health?.last_run_at || healthLogs[0]?.checked_at, health?.last_run_status || healthLogs[0]?.status, health?.last_run_message || healthLogs[0]?.message, health?.interval_seconds),
  ]
  history.value = buildHistory(syncRecords, groupResult?.records || [], rateGuardLogs, balanceLogs, healthRecords, healthLogs)
  loading.value = false
}

function buildHistory(syncRecords: any[], groupRecords: any[], rateGuardLogs: any[], balanceLogs: any[], healthRecords: any[], healthLogs: any[]): AutomationHistoryEntry[] {
  const entries: AutomationHistoryEntry[] = []
  syncRecords.forEach((record, index) => entries.push({
    id: `account-sync-${record.created_at}-${index}`,
    taskKey: 'account-sync',
    status: record.error ? 'failed' : 'success',
    trigger: normalizeTrigger(record.trigger_source),
    time: record.created_at,
    message: record.error || t('admin.upstreamAutomations.syncHistoryMessage', { created: record.created_count || 0, updated: record.updated_count || 0 }),
    impact: String((record.created_count || 0) + (record.updated_count || 0) + (record.conflict_count || 0)),
  }))

  const groupBatches = new Map<string, any[]>()
  groupRecords.forEach(record => {
    const key = record.changed_at || 'unknown'
    groupBatches.set(key, [...(groupBatches.get(key) || []), record])
  })
  groupBatches.forEach((records, time) => entries.push({
    id: `group-rate-fix-${time}`,
    taskKey: 'group-rate-fix', status: 'success', trigger: 'system', time,
    message: records.map(record => record.group_name).filter(Boolean).join(', ') || t('admin.upstreamAutomations.rateFixHistoryMessage'),
    impact: String(records.length),
  }))

  appendPollHistory(entries, 'account-rate-guard', rateGuardLogs)
  appendPollHistory(entries, 'balance-sampler', balanceLogs)
  healthRecords.forEach(record => entries.push({
    id: `health-guard-${record.id}`,
    taskKey: 'health-guard', status: normalizeStatus(record.status), trigger: normalizeTrigger(record.trigger),
    time: record.finished_at || record.started_at,
    message: record.message || t('admin.upstreamAutomations.healthHistoryMessage'),
    impact: String((record.summary?.failed_count || 0) + (record.summary?.slow_count || 0)),
  }))
  if (!healthRecords.length) appendPollHistory(entries, 'health-guard', healthLogs)
  return entries.sort((left, right) => new Date(right.time).getTime() - new Date(left.time).getTime())
}

function appendPollHistory(entries: AutomationHistoryEntry[], taskKey: AutomationKey, logs: any[]) {
  logs.forEach((log, index) => entries.push({
    id: `${taskKey}-${log.checked_at}-${index}`,
    taskKey, status: normalizeStatus(log.status), trigger: normalizeTrigger(log.trigger), time: log.checked_at,
    message: log.message || historyStatusLabel(normalizeStatus(log.status)), impact: '-',
  }))
}

function normalizeStatus(value: string): HistoryStatus {
  if (value === 'failed') return 'failed'
  if (value === 'skipped') return 'skipped'
  return 'success'
}

function normalizeTrigger(value?: string): HistoryTrigger {
  if (value?.includes('manual')) return 'manual'
  if (value?.includes('scheduled')) return 'scheduled'
  return 'system'
}

function makeTask(key: AutomationKey, tone: AutomationTone, icon: AutomationIcon, directRun: boolean, enabled: boolean, lastRunAt?: string, lastStatus?: string, lastMessage?: string, intervalSeconds?: number): AutomationTask {
  const actionPaths: Record<AutomationKey, string> = {
    'account-sync': '/admin/upstream-management/accounts',
    'group-rate-fix': '/admin/upstream-management/groups?rateRisk=true',
    'account-rate-guard': '/admin/upstream-management/accounts',
    'balance-sampler': '/admin/upstream-management/providers',
    'health-guard': '/admin/upstream-management/providers',
  }
  const settingsPaths: Record<AutomationKey, string> = {
    'account-sync': '/admin/upstream-management/accounts',
    'group-rate-fix': '/admin/upstream-management/groups?rateRisk=true',
    'account-rate-guard': '/admin/upstream-management/accounts?automation=rate-guard',
    'balance-sampler': '/admin/upstream-management/providers?automation=balance-sampler',
    'health-guard': '/admin/upstream-management/providers?automation=health-guard',
  }
  return {
    key, tone, icon, directRun, enabled, lastRunAt, lastStatus, lastMessage, intervalSeconds,
    name: t(`admin.upstreamAutomations.tasks.${key}.name`),
    kind: t(`admin.upstreamAutomations.tasks.${key}.kind`),
    description: t(`admin.upstreamAutomations.tasks.${key}.description`),
    actionLabel: t(`admin.upstreamAutomations.tasks.${key}.action`),
    actionPath: actionPaths[key], settingsPath: settingsPaths[key],
  }
}

async function runTask(task: AutomationTask) {
  if (!task.directRun) {
    await router.push(task.actionPath)
    return
  }
  runningKeys.value = new Set(runningKeys.value).add(task.key)
  try {
    const runner = directTaskRunners[task.key]
    if (!runner) throw new Error(`Missing automation runner: ${task.key}`)
    await runner()
    appStore.showSuccess(t('admin.upstreamAutomations.runSuccess', { task: task.name }))
    await loadTasks()
  } catch (error) {
    appStore.showError(error instanceof Error ? error.message : t('admin.upstreamAutomations.runFailed', { task: task.name }))
  } finally {
    const next = new Set(runningKeys.value)
    next.delete(task.key)
    runningKeys.value = next
  }
}

async function retryHistory(entry: AutomationHistoryEntry) {
  const task = tasks.value.find(item => item.key === entry.taskKey)
  if (!task) return
  selectedHistory.value = null
  await runTask(task)
}

function openHistorySettings(entry: AutomationHistoryEntry) {
  const task = tasks.value.find(item => item.key === entry.taskKey)
  if (task) openTaskSettings(task)
}

function openTaskSettings(task: AutomationTask) {
  return router.push(task.settingsPath)
}

function taskName(key: AutomationKey) {
  return tasks.value.find(task => task.key === key)?.name || key
}

function triggerLabel(trigger: HistoryTrigger) {
  return t(`admin.upstreamAutomations.${trigger}Trigger`)
}

function historyStatusLabel(status: HistoryStatus) {
  if (status === 'failed') return t('admin.upstreamAutomations.failed')
  if (status === 'skipped') return t('admin.upstreamAutomations.skipped')
  return t('admin.upstreamAutomations.healthy')
}

function statusLabel(task: AutomationTask) {
  if (!task.enabled) return t('admin.upstreamAutomations.disabled')
  if (task.lastStatus === 'failed') return t('admin.upstreamAutomations.failed')
  if (task.lastStatus === 'success') return t('admin.upstreamAutomations.healthy')
  return t('admin.upstreamAutomations.waiting')
}

function statusClass(task: AutomationTask) {
  if (!task.enabled) return 'state-muted'
  if (task.lastStatus === 'failed') return 'state-danger'
  if (task.lastStatus === 'success') return 'state-good'
  return 'state-waiting'
}

function scheduleLabel(task: AutomationTask) {
  if (!task.enabled) return t('admin.upstreamAutomations.manualOnly')
  if (!task.intervalSeconds) return t('admin.upstreamAutomations.onDemand')
  if (task.intervalSeconds < 3600) return t('admin.upstreamAutomations.everyMinutes', { count: Math.round(task.intervalSeconds / 60) })
  return t('admin.upstreamAutomations.everyHours', { count: Number((task.intervalSeconds / 3600).toFixed(1)) })
}

function lastRunLabel(task: AutomationTask) {
  if (!task.lastRunAt) return t('admin.upstreamAutomations.neverRun')
  const date = new Date(task.lastRunAt)
  return Number.isNaN(date.getTime()) ? task.lastRunAt : date.toLocaleString()
}

function formatDateTime(value: string) {
  const date = new Date(value)
  return Number.isNaN(date.getTime()) ? value : date.toLocaleString()
}
</script>

<style scoped>
.automation-page { @apply min-h-full bg-gray-50 px-4 py-6 dark:bg-gray-950 sm:px-6 lg:px-8; }
.automation-hero { @apply mx-auto flex max-w-[1680px] flex-col gap-4 border-b border-gray-200 pb-6 dark:border-gray-800 sm:flex-row sm:items-end sm:justify-between; }
.automation-kicker { @apply text-xs font-black uppercase tracking-[0.2em] text-cyan-600 dark:text-cyan-400; }
.automation-hero h1 { @apply mt-2 text-3xl font-black tracking-tight text-gray-950 dark:text-white; }
.automation-hero p { @apply mt-2 max-w-3xl text-sm text-gray-500 dark:text-gray-400; }
.automation-refresh { @apply inline-flex min-h-11 items-center justify-center gap-2 rounded-xl border border-gray-200 bg-white px-4 text-sm font-semibold text-gray-700 shadow-sm hover:bg-gray-50 disabled:opacity-50 dark:border-gray-700 dark:bg-gray-800 dark:text-gray-200 dark:hover:bg-gray-700; }
.automation-summary { @apply mx-auto mt-5 grid max-w-[1680px] grid-cols-2 overflow-hidden rounded-2xl border border-gray-200 bg-white shadow-sm dark:border-gray-700 dark:bg-gray-800 lg:grid-cols-4; }
.automation-summary div { @apply border-b border-r border-gray-100 px-5 py-4 last:border-r-0 dark:border-gray-700 lg:border-b-0; }
.automation-summary span { @apply block text-xs font-semibold uppercase tracking-wider text-gray-500 dark:text-gray-400; }
.automation-summary strong { @apply mt-1 block text-2xl font-black text-gray-950 dark:text-white; }
.automation-summary .is-danger { @apply text-red-600 dark:text-red-300; }
.automation-summary .is-good { @apply text-emerald-600 dark:text-emerald-300; }
.automation-warning { @apply mx-auto mt-4 flex max-w-[1680px] items-center gap-2 rounded-xl border border-amber-200 bg-amber-50 px-4 py-3 text-sm text-amber-800 dark:border-amber-800 dark:bg-amber-950/30 dark:text-amber-200; }
.automation-loading { @apply mx-auto mt-8 flex max-w-[1680px] items-center justify-center gap-3 py-24 text-sm text-gray-500; }
.automation-loading span { @apply h-5 w-5 animate-spin rounded-full border-2 border-gray-200 border-t-cyan-500; }
.automation-grid { @apply mx-auto mt-5 grid max-w-[1680px] gap-4 lg:grid-cols-2 2xl:grid-cols-3; }
.automation-card { @apply relative overflow-hidden rounded-2xl border border-gray-200 bg-white p-5 shadow-sm transition-transform hover:-translate-y-0.5 hover:shadow-lg dark:border-gray-700 dark:bg-gray-800; }
.card-rail { @apply absolute inset-y-0 left-0 w-1 bg-cyan-500; }
.tone-amber .card-rail { @apply bg-amber-500; }.tone-violet .card-rail { @apply bg-violet-500; }.tone-emerald .card-rail { @apply bg-emerald-500; }.tone-blue .card-rail { @apply bg-blue-500; }
.card-header { @apply flex items-center justify-between; }.task-icon { @apply flex h-11 w-11 items-center justify-center rounded-xl bg-gray-950 text-white dark:bg-white dark:text-gray-950; }
.task-state { @apply rounded-full px-2.5 py-1 text-xs font-bold; }.state-good { @apply bg-emerald-50 text-emerald-700 dark:bg-emerald-900/30 dark:text-emerald-300; }.state-danger { @apply bg-red-50 text-red-700 dark:bg-red-900/30 dark:text-red-300; }.state-muted { @apply bg-gray-100 text-gray-500 dark:bg-gray-700 dark:text-gray-300; }.state-waiting { @apply bg-amber-50 text-amber-700 dark:bg-amber-900/30 dark:text-amber-300; }
.card-copy { @apply mt-5; }.task-kind { @apply text-[11px] font-black uppercase tracking-[0.18em] text-gray-400; }.card-copy h2 { @apply mt-1 text-xl font-black text-gray-950 dark:text-white; }.card-copy p { @apply mt-2 min-h-10 text-sm leading-5 text-gray-500 dark:text-gray-400; }
.task-facts { @apply mt-5 divide-y divide-gray-100 border-y border-gray-100 dark:divide-gray-700 dark:border-gray-700; }.task-facts div { @apply grid grid-cols-[92px_1fr] gap-3 py-2.5 text-xs; }.task-facts dt { @apply font-semibold text-gray-400; }.task-facts dd { @apply truncate text-right font-medium text-gray-700 dark:text-gray-200; }
.card-actions { @apply mt-5 flex gap-2; }.task-primary,.task-secondary { @apply inline-flex min-h-10 items-center justify-center gap-2 rounded-xl px-4 text-sm font-bold focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-cyan-500 disabled:opacity-50; }.task-primary { @apply flex-1 bg-gray-950 text-white hover:bg-gray-800 dark:bg-white dark:text-gray-950 dark:hover:bg-gray-100; }.task-secondary { @apply border border-gray-200 bg-white text-gray-600 hover:bg-gray-50 dark:border-gray-700 dark:bg-gray-800 dark:text-gray-300 dark:hover:bg-gray-700; }
.history-panel { @apply mx-auto mt-5 max-w-[1680px] rounded-2xl border border-gray-200 bg-white p-5 shadow-sm dark:border-gray-700 dark:bg-gray-800; }
.history-header { @apply flex items-start justify-between gap-4; }.history-header h2 { @apply mt-1 text-xl font-black text-gray-950 dark:text-white; }.history-header p { @apply mt-1 text-sm text-gray-500 dark:text-gray-400; }.history-count { @apply flex h-10 min-w-10 items-center justify-center rounded-xl bg-gray-950 px-3 text-sm font-black text-white dark:bg-white dark:text-gray-950; }
.history-filters { @apply mt-5 grid gap-2 sm:grid-cols-3; }.history-filters select { @apply min-h-10 rounded-xl border border-gray-200 bg-white px-3 text-sm text-gray-700 focus:border-cyan-500 focus:outline-none focus:ring-2 focus:ring-cyan-500/20 dark:border-gray-700 dark:bg-gray-900 dark:text-gray-200; }
.history-list { @apply mt-4 divide-y divide-gray-100 dark:divide-gray-700; }.history-row { @apply grid min-h-16 w-full grid-cols-[10px_minmax(150px,0.8fr)_minmax(160px,1.5fr)_60px_145px_18px] items-center gap-3 rounded-lg px-2 text-left text-sm hover:bg-gray-50 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-cyan-500 dark:hover:bg-gray-700/50; }.history-dot { @apply h-2.5 w-2.5 rounded-full bg-gray-400; }.history-success { @apply bg-emerald-500; }.history-failed { @apply bg-red-500; }.history-skipped { @apply bg-amber-500; }.history-task strong { @apply block text-gray-900 dark:text-white; }.history-task small { @apply mt-1 block text-xs text-gray-400; }.history-message { @apply truncate text-gray-600 dark:text-gray-300; }.history-impact { @apply text-center font-mono font-bold text-gray-700 dark:text-gray-200; }.history-row time { @apply text-xs text-gray-400; }.history-empty { @apply py-16 text-center text-sm text-gray-400; }
.history-detail-overlay { @apply fixed inset-0 z-50 flex justify-end bg-black/45; }.history-detail { @apply h-full w-full max-w-lg overflow-y-auto bg-white p-6 shadow-2xl dark:bg-gray-900; }.history-detail-header { @apply flex items-start justify-between gap-4 border-b border-gray-200 pb-5 dark:border-gray-700; }.history-detail-header h2 { @apply mt-1 text-2xl font-black text-gray-950 dark:text-white; }.history-detail-header button { @apply rounded-xl p-2 text-gray-400 hover:bg-gray-100 dark:hover:bg-gray-800; }.history-detail dl { @apply mt-5 divide-y divide-gray-100 dark:divide-gray-800; }.history-detail dl div { @apply grid grid-cols-[110px_1fr] gap-4 py-4; }.history-detail dt { @apply text-sm font-semibold text-gray-400; }.history-detail dd { @apply whitespace-pre-wrap break-words text-sm font-medium text-gray-800 dark:text-gray-200; }.history-detail-actions { @apply mt-6 flex gap-2; }
@media (max-width: 767px) { .history-row { @apply grid-cols-[10px_1fr_18px] gap-2 py-3; }.history-message,.history-impact,.history-row time { @apply col-start-2; }.history-row time { @apply row-start-4; }.history-row > svg { @apply col-start-3 row-start-1; } }
@media (prefers-reduced-motion: reduce) { .automation-card { @apply transition-none hover:translate-y-0; } }
</style>

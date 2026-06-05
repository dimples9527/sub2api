<template>
  <AppLayout>
    <div class="w-full min-w-0 space-y-5">
      <div class="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
        <div>
          <h1 class="text-2xl font-semibold text-gray-900 dark:text-white">上游分组</h1>
          <p class="mt-1 text-sm text-gray-500 dark:text-gray-400">
            展示上游可用分组，并标记与本地分组的匹配关系。
          </p>
        </div>
        <button type="button" class="btn btn-secondary btn-sm" :disabled="loading" @click="loadGroups">
          <Icon name="refresh" size="sm" :class="loading ? 'animate-spin' : ''" />
          刷新
        </button>
      </div>

      <div class="grid gap-3 md:grid-cols-[1fr_auto_auto]">
        <div class="relative">
          <Icon name="search" size="sm" class="pointer-events-none absolute left-3 top-1/2 -translate-y-1/2 text-gray-400" />
          <input v-model="search" type="search" class="input pl-9" placeholder="搜索上游分组、本地分组或描述" />
        </div>
        <select v-model="platform" class="input min-w-40">
          <option value="">全部平台</option>
          <option v-for="item in platforms" :key="item" :value="item">{{ item }}</option>
        </select>
        <select v-model="matchStatus" class="input min-w-36">
          <option value="">全部匹配</option>
          <option value="matched">已匹配</option>
          <option value="unmatched">未匹配</option>
        </select>
      </div>

      <div class="grid gap-3 sm:grid-cols-3">
        <div class="rounded-lg border border-gray-200 bg-white px-4 py-3 dark:border-dark-700 dark:bg-dark-800">
          <div class="text-xs font-medium text-gray-500 dark:text-gray-400">上游分组</div>
          <div class="mt-1 text-2xl font-semibold text-gray-950 dark:text-white">{{ groups.length }}</div>
        </div>
        <div class="rounded-lg border border-gray-200 bg-white px-4 py-3 dark:border-dark-700 dark:bg-dark-800">
          <div class="text-xs font-medium text-gray-500 dark:text-gray-400">已匹配本地</div>
          <div class="mt-1 text-2xl font-semibold text-emerald-600 dark:text-emerald-300">{{ matchedCount }}</div>
        </div>
        <div class="rounded-lg border border-gray-200 bg-white px-4 py-3 dark:border-dark-700 dark:bg-dark-800">
          <div class="text-xs font-medium text-gray-500 dark:text-gray-400">未匹配</div>
          <div class="mt-1 text-2xl font-semibold text-amber-600 dark:text-amber-300">{{ unmatchedCount }}</div>
        </div>
      </div>

      <div v-if="loading" class="grid min-h-64 place-items-center rounded-lg border border-gray-200 bg-white dark:border-dark-700 dark:bg-dark-800">
        <div class="flex items-center gap-2 text-sm text-gray-500 dark:text-gray-400">
          <span class="h-5 w-5 animate-spin rounded-full border-2 border-primary-500 border-t-transparent"></span>
          正在加载上游分组...
        </div>
      </div>

      <div v-else-if="error" class="rounded-lg border border-red-200 bg-red-50 p-5 text-sm text-red-700 dark:border-red-900/60 dark:bg-red-900/20 dark:text-red-300">
        {{ error }}
      </div>

      <div v-else class="space-y-3">
        <div v-if="syncSuccess" class="rounded-lg border border-emerald-200 bg-emerald-50 px-4 py-3 text-sm text-emerald-700 dark:border-emerald-900/60 dark:bg-emerald-900/20 dark:text-emerald-300">
          {{ syncSuccess }}
        </div>

        <div class="overflow-hidden rounded-lg border border-gray-200 bg-white dark:border-dark-700 dark:bg-dark-800">
          <div class="flex items-center justify-between border-b border-gray-100 px-4 py-3 text-sm text-gray-500 dark:border-dark-700 dark:text-gray-400">
            <span>共 {{ filteredGroups.length }} 条记录</span>
            <span>
              数据来自配置的上游分组 URL
              <span v-if="monitorTrendError" class="ml-2 text-amber-600 dark:text-amber-300">
                {{ monitorTrendError }}
              </span>
              <span v-if="keySummaryError" class="ml-2 text-amber-600 dark:text-amber-300">
                {{ keySummaryError }}
              </span>
            </span>
          </div>
          <div v-if="filteredGroups.length === 0" class="grid min-h-56 place-items-center text-sm text-gray-500 dark:text-gray-400">
            暂无匹配记录
          </div>
          <div v-else class="overflow-x-auto">
            <table class="min-w-full text-left text-sm">
            <thead class="bg-gray-50 text-xs font-semibold text-gray-500 dark:bg-dark-700/60 dark:text-gray-400">
              <tr>
                <th class="whitespace-nowrap px-4 py-3">操作</th>
                <th class="whitespace-nowrap px-4 py-3">上游 ID</th>
                <th class="min-w-56 px-4 py-3">上游分组名称</th>
                <th class="whitespace-nowrap px-4 py-3">平台</th>
                <th class="whitespace-nowrap px-4 py-3">上游倍率</th>
                <th class="whitespace-nowrap px-4 py-3">状态</th>
                <th class="whitespace-nowrap px-4 py-3">上游秘钥</th>
                <th class="min-w-52 px-4 py-3">本地分组名称</th>
                <th class="min-w-56 px-4 py-3">近90分钟趋势</th>
                <th class="whitespace-nowrap px-4 py-3">本地倍率</th>
                <th class="min-w-72 px-4 py-3">描述</th>
                <th class="whitespace-nowrap px-4 py-3">更新时间</th>
              </tr>
            </thead>
            <tbody class="divide-y divide-gray-100 dark:divide-dark-700">
              <tr v-for="group in filteredGroups" :key="String(group.id)" class="transition hover:bg-gray-50 dark:hover:bg-dark-700/60">
                <td class="whitespace-nowrap px-4 py-4">
                  <button
                    v-if="!hasLocalMatch(group)"
                    type="button"
                    class="btn btn-primary btn-xs"
                    :data-test="`sync-local-group-${group.id}`"
                    @click="openSyncDialog(group)"
                  >
                    同步
                  </button>
                  <span v-else class="text-xs font-medium text-gray-400 dark:text-gray-500">已匹配</span>
                </td>
                <td class="whitespace-nowrap px-4 py-4 font-mono text-gray-600 dark:text-gray-300">{{ group.id }}</td>
                <td class="px-4 py-4 font-medium text-gray-950 dark:text-white">
                  <span class="break-words">{{ group.name || '-' }}</span>
                </td>
                <td class="whitespace-nowrap px-4 py-4">
                  <span class="rounded bg-slate-100 px-2 py-1 text-xs font-semibold text-slate-700 dark:bg-dark-700 dark:text-gray-200">
                    {{ group.platform || 'unknown' }}
                  </span>
                </td>
                <td class="whitespace-nowrap px-4 py-4 font-mono text-gray-900 dark:text-gray-100">{{ formatRate(group.rate_multiplier) }}</td>
                <td class="whitespace-nowrap px-4 py-4">
                  <span class="rounded px-2 py-1 text-xs font-semibold" :class="statusClass(group.status)">
                    {{ group.status || '-' }}
                  </span>
                </td>
                <td class="whitespace-nowrap px-4 py-4">
                  <button
                    v-if="hasUpstreamKeys(group)"
                    type="button"
                    class="rounded px-2 py-1 text-xs font-semibold transition hover:ring-2 hover:ring-emerald-300"
                    :class="upstreamKeyStatusClass(group)"
                    :data-test="`upstream-key-status-${group.id}`"
                    @click="openKeySummaryDialog(group)"
                  >
                    {{ upstreamKeyStatusText(group) }}
                  </button>
                  <span
                    v-else
                    class="rounded px-2 py-1 text-xs font-semibold"
                    :class="upstreamKeyStatusClass(group)"
                    :data-test="`upstream-key-status-${group.id}`"
                  >
                    {{ upstreamKeyStatusText(group) }}
                  </span>
                </td>
                <td class="px-4 py-4">
                  <span v-if="group.local_group_name" class="break-words font-medium text-gray-900 dark:text-white">
                    {{ group.local_group_name }}
                  </span>
                  <span v-else class="text-gray-400 dark:text-gray-500">未匹配</span>
                </td>
                <td class="px-4 py-4">
                  <div
                    v-if="monitorTrendForGroup(group)"
                    class="flex min-w-0 items-center gap-2"
                    :title="monitorTrendTitle(monitorTrendForGroup(group)!)"
                  >
                    <div class="monitor-trend-row" aria-label="近90分钟可用率趋势">
                      <span
                        v-for="(point, index) in monitorTrendForGroup(group)!.points"
                        :key="index"
                        class="monitor-trend-block"
                        :class="`monitor-trend-${point.tone}`"
                      />
                    </div>
                    <span
                      class="shrink-0 font-mono text-xs font-semibold"
                      :class="availabilityTextClass(monitorTrendForGroup(group)!.availability)"
                    >
                      {{ formatAvailability(monitorTrendForGroup(group)!.availability) }}
                    </span>
                  </div>
                  <span v-else class="text-gray-400 dark:text-gray-500">-</span>
                </td>
                <td class="whitespace-nowrap px-4 py-4 font-mono text-gray-900 dark:text-gray-100">
                  {{ group.local_rate_multiplier == null ? '-' : formatRate(group.local_rate_multiplier) }}
                </td>
                <td class="px-4 py-4 text-gray-600 dark:text-gray-300">
                  <span class="line-clamp-2 break-words">{{ group.description || '-' }}</span>
                </td>
                <td class="whitespace-nowrap px-4 py-4 text-gray-500 dark:text-gray-400">{{ formatDate(group.updated_at) }}</td>
              </tr>
            </tbody>
            </table>
          </div>
        </div>
      </div>

      <div
        v-if="syncDialogGroup"
        class="fixed inset-0 z-50 flex items-center justify-center bg-black/40 px-4 py-6"
        @click.self="closeSyncDialog"
      >
        <div class="w-full max-w-lg overflow-hidden rounded-lg bg-white shadow-xl dark:bg-dark-800">
          <div class="border-b border-gray-100 px-5 py-4 dark:border-dark-700">
            <h2 class="text-lg font-semibold text-gray-950 dark:text-white">同步上游分组到本地</h2>
            <p class="mt-1 text-sm text-gray-500 dark:text-gray-400">
              将未匹配的上游分组创建为本地分组，确认前可以调整本地倍率。
            </p>
          </div>
          <div class="space-y-4 px-5 py-4">
            <div class="grid gap-3 sm:grid-cols-2">
              <div>
                <div class="text-xs font-medium text-gray-500 dark:text-gray-400">上游分组</div>
                <div class="mt-1 break-words text-sm font-semibold text-gray-950 dark:text-white">{{ syncDialogGroup.name || '-' }}</div>
              </div>
              <div>
                <div class="text-xs font-medium text-gray-500 dark:text-gray-400">平台</div>
                <div class="mt-1 text-sm font-semibold text-gray-950 dark:text-white">{{ syncDialogGroup.platform || 'unknown' }}</div>
              </div>
              <div>
                <div class="text-xs font-medium text-gray-500 dark:text-gray-400">上游倍率</div>
                <div class="mt-1 font-mono text-sm font-semibold text-gray-950 dark:text-white">{{ formatRate(syncDialogGroup.rate_multiplier) }}</div>
              </div>
              <div>
                <label class="text-xs font-medium text-gray-500 dark:text-gray-400" for="sync-rate-multiplier">本地倍率</label>
                <input
                  id="sync-rate-multiplier"
                  v-model.number="syncRateMultiplier"
                  data-test="sync-rate-multiplier"
                  type="number"
                  min="0.0001"
                  step="0.0001"
                  class="input mt-1"
                />
              </div>
            </div>
            <div>
              <div class="text-xs font-medium text-gray-500 dark:text-gray-400">描述</div>
              <div class="mt-1 max-h-24 overflow-auto rounded border border-gray-200 bg-gray-50 px-3 py-2 text-sm text-gray-600 dark:border-dark-700 dark:bg-dark-900 dark:text-gray-300">
                {{ syncDialogGroup.description || '-' }}
              </div>
            </div>
            <div v-if="syncError" class="rounded border border-red-200 bg-red-50 px-3 py-2 text-sm text-red-700 dark:border-red-900/60 dark:bg-red-900/20 dark:text-red-300">
              {{ syncError }}
            </div>
          </div>
          <div class="flex justify-end gap-2 border-t border-gray-100 px-5 py-4 dark:border-dark-700">
            <button type="button" class="btn btn-secondary btn-sm" :disabled="syncSubmitting" @click="closeSyncDialog">取消</button>
            <button
              type="button"
              class="btn btn-primary btn-sm"
              data-test="confirm-sync-local-group"
              :disabled="syncSubmitting"
              @click="submitSyncLocalGroup"
            >
              <span v-if="syncSubmitting" class="mr-1 h-4 w-4 animate-spin rounded-full border-2 border-white/70 border-t-transparent"></span>
              确认同步
            </button>
          </div>
        </div>
      </div>

      <div
        v-if="keySummaryDialogGroup"
        class="fixed inset-0 z-50 flex items-center justify-center bg-black/40 px-4 py-6"
        @click.self="closeKeySummaryDialog"
      >
        <div class="w-full max-w-lg overflow-hidden rounded-lg bg-white shadow-xl dark:bg-dark-800" data-test="upstream-key-dialog">
          <div class="border-b border-gray-100 px-5 py-4 dark:border-dark-700">
            <h2 class="text-lg font-semibold text-gray-950 dark:text-white">上游秘钥</h2>
            <p class="mt-1 text-sm text-gray-500 dark:text-gray-400">
              {{ keySummaryDialogGroup.name || '-' }} 共 {{ keySummaryDialogKeys.length }} 个
            </p>
          </div>
          <div class="max-h-80 overflow-auto px-5 py-4">
            <div v-if="keySummaryDialogKeys.length === 0" class="text-sm text-gray-500 dark:text-gray-400">
              暂无秘钥
            </div>
            <ul v-else class="space-y-2">
              <li
                v-for="item in keySummaryDialogKeys"
                :key="item.name"
                class="rounded border border-gray-200 bg-gray-50 px-3 py-2 text-sm font-medium text-gray-900 dark:border-dark-700 dark:bg-dark-900 dark:text-gray-100"
              >
                {{ item.name || '-' }}
              </li>
            </ul>
          </div>
          <div class="flex justify-end border-t border-gray-100 px-5 py-4 dark:border-dark-700">
            <button type="button" class="btn btn-secondary btn-sm" @click="closeKeySummaryDialog">关闭</button>
          </div>
        </div>
      </div>
    </div>
  </AppLayout>
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import AppLayout from '@/components/layout/AppLayout.vue'
import Icon from '@/components/icons/Icon.vue'
import {
  create as createLocalGroup,
  getUpstreamAvailableGroups,
  getUpstreamKeySummary,
  getUpstreamMonitorStatus,
  type UpstreamKeySummary,
  type UpstreamAvailableGroup
} from '@/api/admin/groups'
import type { CreateGroupRequest, GroupPlatform, SubscriptionType } from '@/types'

type MatchStatus = '' | 'matched' | 'unmatched'
type TrendTone = 'green' | 'yellow' | 'red'

interface MonitorTrendPoint {
  tone: TrendTone
  latency: number
  time: string
  statusText: string
  availability: number
}

interface MonitorTrend {
  name: string
  availability: number
  latency: number
  time: string
  points: MonitorTrendPoint[]
}

interface UpstreamKeySummaryEntry {
  name: string
  count: number
  keys: Array<{ name: string }>
}

const loading = ref(false)
const error = ref('')
const monitorTrendError = ref('')
const keySummaryError = ref('')
const search = ref('')
const platform = ref('')
const matchStatus = ref<MatchStatus>('')
const groups = ref<UpstreamAvailableGroup[]>([])
const monitorTrends = ref<Map<string, MonitorTrend>>(new Map())
const upstreamKeySummary = ref<Map<string, UpstreamKeySummaryEntry> | null>(null)
const keySummaryDialogGroup = ref<UpstreamAvailableGroup | null>(null)
const syncDialogGroup = ref<UpstreamAvailableGroup | null>(null)
const syncRateMultiplier = ref<number>(1)
const syncSubmitting = ref(false)
const syncError = ref('')
const syncSuccess = ref('')

const platforms = computed(() =>
  Array.from(new Set(groups.value.map((group) => group.platform).filter(Boolean) as string[]))
    .sort((a, b) => a.localeCompare(b))
)

const matchedCount = computed(() => groups.value.filter((group) => hasLocalMatch(group)).length)
const unmatchedCount = computed(() => groups.value.length - matchedCount.value)

const filteredGroups = computed(() => {
  const keyword = search.value.trim().toLowerCase()
  return groups.value.filter((group) => {
    if (platform.value && group.platform !== platform.value) return false
    const matched = hasLocalMatch(group)
    if (matchStatus.value === 'matched' && !matched) return false
    if (matchStatus.value === 'unmatched' && matched) return false
    if (!keyword) return true
    return [
      group.name,
      group.description,
      group.platform,
      group.status,
      group.local_group_name,
      String(group.id)
    ].some((value) => String(value || '').toLowerCase().includes(keyword))
  })
})

async function loadGroups() {
  loading.value = true
  error.value = ''
  monitorTrendError.value = ''
  keySummaryError.value = ''
  try {
    const [nextGroups] = await Promise.all([
      getUpstreamAvailableGroups(),
      loadMonitorTrends(),
      loadUpstreamKeySummary(),
    ])
    groups.value = nextGroups
  } catch (err) {
    const e = err as { message?: string }
    error.value = e.message || '上游分组加载失败'
  } finally {
    loading.value = false
  }
}

async function loadUpstreamKeySummary() {
  try {
    const payload = await getUpstreamKeySummary()
    upstreamKeySummary.value = buildUpstreamKeySummaryMap(payload)
  } catch {
    upstreamKeySummary.value = null
    keySummaryError.value = '上游秘钥摘要加载失败'
  }
}

async function loadMonitorTrends() {
  try {
    const payload = await getUpstreamMonitorStatus({ period: '90m', board: 'hot' })
    monitorTrends.value = buildMonitorTrendMap(normalizeMonitorPayload(payload))
  } catch {
    monitorTrends.value = new Map()
    monitorTrendError.value = '近90分钟趋势加载失败'
  }
}

function hasLocalMatch(group: UpstreamAvailableGroup) {
  return group.local_group_id != null || Boolean(group.local_group_name)
}

function openSyncDialog(group: UpstreamAvailableGroup) {
  syncDialogGroup.value = group
  syncRateMultiplier.value = normalizePositiveRate(group.rate_multiplier, 1)
  syncError.value = ''
  syncSuccess.value = ''
}

function closeSyncDialog() {
  if (syncSubmitting.value) return
  syncDialogGroup.value = null
  syncError.value = ''
}

async function submitSyncLocalGroup() {
  if (!syncDialogGroup.value) return
  const rate = Number(syncRateMultiplier.value)
  if (!Number.isFinite(rate) || rate <= 0) {
    syncError.value = '本地倍率必须大于 0'
    return
  }

  syncSubmitting.value = true
  syncError.value = ''
  try {
    await createLocalGroup(buildLocalGroupPayload(syncDialogGroup.value, rate))
    syncDialogGroup.value = null
    syncSuccess.value = '同步成功'
    await loadGroups()
  } catch (err) {
    const e = err as { message?: string }
    syncError.value = e.message || '同步到本地失败'
  } finally {
    syncSubmitting.value = false
  }
}

function buildLocalGroupPayload(group: UpstreamAvailableGroup, rateMultiplier: number): CreateGroupRequest {
  const payload: CreateGroupRequest = {
    name: group.name,
    description: group.description || '',
    rate_multiplier: rateMultiplier,
  }

  const platformValue = normalizeGroupPlatform(group.platform)
  if (platformValue) payload.platform = platformValue

  const subscriptionType = normalizeSubscriptionType(group.subscription_type)
  if (subscriptionType) payload.subscription_type = subscriptionType

  copyNullableNumber(payload, 'daily_limit_usd', group.daily_limit_usd)
  copyNullableNumber(payload, 'weekly_limit_usd', group.weekly_limit_usd)
  copyNullableNumber(payload, 'monthly_limit_usd', group.monthly_limit_usd)
  copyNullableNumber(payload, 'image_price_1k', group.image_price_1k)
  copyNullableNumber(payload, 'image_price_2k', group.image_price_2k)
  copyNullableNumber(payload, 'image_price_4k', group.image_price_4k)

  if (typeof group.claude_code_only === 'boolean') payload.claude_code_only = group.claude_code_only
  if (typeof group.allow_messages_dispatch === 'boolean') payload.allow_messages_dispatch = group.allow_messages_dispatch

  return payload
}

function copyNullableNumber<K extends keyof CreateGroupRequest>(payload: CreateGroupRequest, key: K, value: number | null | undefined) {
  if (value !== undefined) {
    ;(payload[key] as number | null | undefined) = value
  }
}

function normalizePositiveRate(value: unknown, fallback: number) {
  const numeric = Number(value)
  return Number.isFinite(numeric) && numeric > 0 ? numeric : fallback
}

function normalizeGroupPlatform(value?: string): GroupPlatform | undefined {
  if (value === 'anthropic' || value === 'openai' || value === 'gemini' || value === 'antigravity') {
    return value
  }
  return undefined
}

function normalizeSubscriptionType(value?: string): SubscriptionType | undefined {
  if (value === 'standard' || value === 'subscription') return value
  return undefined
}

function monitorTrendForGroup(group: UpstreamAvailableGroup) {
  const keys = [group.local_group_name, group.name]
    .map((value) => normalizeMonitorKey(value))
    .filter(Boolean)
  for (const key of keys) {
    const trend = monitorTrends.value.get(key)
    if (trend) return trend
  }
  return undefined
}

function upstreamKeyCountForGroup(group: UpstreamAvailableGroup) {
  const entry = upstreamKeySummaryForGroup(group)
  if (!upstreamKeySummary.value) return undefined
  return entry?.count ?? 0
}

function upstreamKeySummaryForGroup(group: UpstreamAvailableGroup) {
  if (!upstreamKeySummary.value) return undefined
  const keys = [group.name, group.local_group_name]
    .map((value) => normalizeMonitorKey(value))
    .filter(Boolean)
  for (const key of keys) {
    const entry = upstreamKeySummary.value.get(key)
    if (entry) return entry
  }
  return undefined
}

function upstreamKeyStatusText(group: UpstreamAvailableGroup) {
  const count = upstreamKeyCountForGroup(group)
  if (count == null) return '未知'
  if (count > 0) return `有秘钥 ${count}`
  return '无秘钥'
}

function upstreamKeyStatusClass(group: UpstreamAvailableGroup) {
  const count = upstreamKeyCountForGroup(group)
  if (count == null) return 'bg-gray-100 text-gray-600 dark:bg-dark-700 dark:text-gray-300'
  if (count > 0) return 'bg-emerald-100 text-emerald-700 dark:bg-emerald-900/40 dark:text-emerald-300'
  return 'bg-amber-100 text-amber-700 dark:bg-amber-900/40 dark:text-amber-300'
}

function hasUpstreamKeys(group: UpstreamAvailableGroup) {
  return (upstreamKeyCountForGroup(group) || 0) > 0
}

const keySummaryDialogKeys = computed(() => {
  if (!keySummaryDialogGroup.value) return []
  return upstreamKeySummaryForGroup(keySummaryDialogGroup.value)?.keys || []
})

function openKeySummaryDialog(group: UpstreamAvailableGroup) {
  if ((upstreamKeyCountForGroup(group) || 0) <= 0) return
  keySummaryDialogGroup.value = group
}

function closeKeySummaryDialog() {
  keySummaryDialogGroup.value = null
}

function formatRate(value?: number | null) {
  const n = Number(value)
  if (!Number.isFinite(n)) return '-'
  return `${n.toFixed(4).replace(/0+$/, '').replace(/\.$/, '')}x`
}

function formatDate(value?: string) {
  if (!value) return '-'
  const date = new Date(value)
  if (Number.isNaN(date.getTime())) return value
  return date.toLocaleString('zh-CN', { hour12: false })
}

function formatAvailability(value: number) {
  return `${value.toFixed(value % 1 ? 2 : 0)}%`
}

function availabilityTextClass(value: number) {
  if (value >= 75) return 'text-emerald-600 dark:text-emerald-300'
  if (value >= 30) return 'text-amber-600 dark:text-amber-300'
  return 'text-red-600 dark:text-red-300'
}

function monitorTrendTitle(trend: MonitorTrend) {
  return `${trend.name} 近90分钟可用率 ${formatAvailability(trend.availability)}，最后监测 ${trend.latency}ms ${trend.time}`
}

function statusClass(status?: string) {
  if (status === 'active') {
    return 'bg-emerald-100 text-emerald-700 dark:bg-emerald-900/40 dark:text-emerald-300'
  }
  return 'bg-gray-100 text-gray-600 dark:bg-dark-700 dark:text-gray-300'
}

function normalizeMonitorKey(value: unknown) {
  return String(value || '').replace(/\s+/g, '').toLocaleLowerCase()
}

function buildUpstreamKeySummaryMap(payload: UpstreamKeySummary) {
  const map = new Map<string, UpstreamKeySummaryEntry>()
  for (const item of payload.groups || []) {
    const key = normalizeMonitorKey(item.normalized_name || item.name)
    if (!key) continue
    map.set(key, {
      name: item.name,
      count: Math.max(0, Number(item.key_count) || 0),
      keys: (item.keys || []).map((entry) => ({ name: String(entry.name || '') })).filter((entry) => entry.name),
    })
  }
  return map
}

function pickValue(source: unknown, keys: string[], fallback: unknown = undefined) {
  if (!source || typeof source !== 'object') return fallback
  const record = source as Record<string, unknown>
  for (const key of keys) {
    const value = record[key]
    if (value !== undefined && value !== null) return value
  }
  return fallback
}

function asNumber(value: unknown, fallback = 0) {
  if (typeof value === 'number' && Number.isFinite(value)) return value
  if (typeof value === 'string') {
    const parsed = Number(value.replace('%', '').replace('×', '').replace('x', '').trim())
    if (Number.isFinite(parsed)) return parsed
  }
  return fallback
}

function normalizeMonitorPayload(payload: unknown): unknown[] {
  if (Array.isArray(payload)) return payload
  if (!payload || typeof payload !== 'object') return []
  const record = payload as Record<string, unknown>
  const candidates = [
    record.groups,
    record.data,
    record.items,
    record.list,
    record.status,
    record.statuses,
    record.providers,
    record.services,
    record.result,
  ]
  let emptyArray: unknown[] | null = null
  for (const candidate of candidates) {
    if (Array.isArray(candidate)) {
      if (candidate.length) return candidate
      emptyArray ||= candidate
      continue
    }
    if (candidate && typeof candidate === 'object') {
      const nested = normalizeMonitorPayload(candidate)
      if (nested.length) return nested
    }
  }
  return emptyArray || []
}

function statusAvailability(status: unknown) {
  if (Number(status) === 1) return 100
  if (Number(status) === 2) return 70
  return 0
}

function statusText(status: unknown) {
  switch (Number(status)) {
    case 1:
      return '可用'
    case 2:
      return '降级'
    case 0:
      return '不可用'
    default:
      return '未知'
  }
}

function pointTone(point: unknown): TrendTone {
  const status = asNumber(pickValue(point, ['status'], undefined), Number.NaN)
  const availability = asNumber(pickValue(point, ['availability'], statusAvailability(status)), 0)
  if (status === 1 || availability >= 75) return 'green'
  if (status === 2 || availability >= 30) return 'yellow'
  return 'red'
}

function pointTimestamp(point: unknown) {
  return asNumber(pickValue(point, ['timestamp', 'checked_at', 'checkedAt'], 0), 0)
}

function formatCheckedTime(value: unknown, fallback = '--:--') {
  const numeric = asNumber(value, 0)
  if (numeric > 0) {
    const milliseconds = numeric < 10000000000 ? numeric * 1000 : numeric
    return new Date(milliseconds).toLocaleTimeString('zh-CN', {
      hour: '2-digit',
      minute: '2-digit',
    })
  }
  const text = String(value || '').trim()
  return text ? text.slice(-5) : fallback
}

function trendBucketStepMs(monitorStepMs: number) {
  return Math.max(1000, monitorStepMs || 300000)
}

function monitorIntervalMs(item: unknown, layer: unknown) {
  const interval = asNumber(
    pickValue(item, ['interval_ms', 'intervalMs'], pickValue(layer, ['interval_ms', 'intervalMs'], 300000)),
    300000,
  )
  return Math.max(1000, interval)
}

function timestampToMs(value: unknown) {
  const numeric = asNumber(value, 0)
  if (numeric <= 0) return 0
  return numeric < 10000000000 ? numeric * 1000 : numeric
}

function normalizeTrendPoint(point: unknown, row: { availability: number; latency: number; time: string }, fallbackIndex: number): MonitorTrendPoint {
  if (point && typeof point === 'object') {
    const status = asNumber(pickValue(point, ['status'], undefined), Number.NaN)
    const latency = asNumber(pickValue(point, ['latency', 'latency_ms', 'latencyMs'], row.latency))
    return {
      tone: pointTone(point),
      latency,
      time: formatCheckedTime(pickValue(point, ['displayTimestamp', 'time', 'timestamp', 'checked_at', 'checkedAt'], ''), row.time),
      statusText: statusText(status),
      availability: asNumber(pickValue(point, ['availability'], statusAvailability(status)), 0),
    }
  }
  const tone = String(point || 'red') as TrendTone
  const status = tone === 'green' ? 1 : tone === 'yellow' ? 2 : 0
  return {
    tone: tone === 'green' || tone === 'yellow' ? tone : 'red',
    latency: row.latency,
    time: row.time || `#${fallbackIndex + 1}`,
    statusText: statusText(status),
    availability: row.availability,
  }
}

function fallbackTrend(availability: number, seed: number) {
  const bars: TrendTone[] = []
  const normalized = Math.max(0, Math.min(availability, 100)) / 100
  for (let i = 0; i < 18; i += 1) {
    if (availability <= 0) bars.push('red')
    else if (availability >= 99) bars.push(i === 6 || i === 13 ? 'yellow' : 'green')
    else if (normalized >= 0.85) bars.push(i % 8 === seed % 8 ? 'yellow' : 'green')
    else if (normalized >= 0.6) bars.push(i % 6 === seed % 6 || (i > 9 && i % 4 === 1) ? 'yellow' : 'green')
    else if (normalized >= 0.3) bars.push(i < 4 || i > 13 ? 'red' : i % 5 === seed % 5 ? 'yellow' : 'green')
    else bars.push(i < 10 ? 'red' : i % 4 === seed % 4 ? 'yellow' : 'red')
  }
  return bars
}

function normalizeMonitorTrend(item: unknown, index: number): MonitorTrend | null {
  if (!item || typeof item !== 'object') return null
  const record = item as Record<string, unknown>
  const layers = Array.isArray(record.layers) ? record.layers : []
  const layer = layers.length ? layers[0] : {}
  const current = (layer && typeof layer === 'object' ? (layer as Record<string, unknown>).current_status : {}) || {}
  const timeline = (Array.isArray((layer as Record<string, unknown>).timeline)
    ? [...((layer as Record<string, unknown>).timeline as unknown[])]
    : []
  ).sort((a, b) => pointTimestamp(a) - pointTimestamp(b))
  const latestPoint = timeline.length ? timeline[timeline.length - 1] : {}
  const name = String(pickValue(item, ['provider', 'provider_name', 'providerName', 'name', 'title', 'service_provider'], '')).trim()
  if (!name) return null

  const availabilityValues = timeline.map((point) => asNumber(pickValue(point, ['availability'], statusAvailability(pickValue(point, ['status'], undefined)))))
  const availability = availabilityValues.length
    ? Math.round((availabilityValues.reduce((sum, value) => sum + value, 0) / availabilityValues.length) * 100) / 100
    : asNumber(pickValue(item, ['availability', 'available_rate', 'availableRate', 'success_rate', 'successRate', 'rate_percent', 'uptime'], statusAvailability(pickValue(current, ['status'], record.current_status))), 0)
  const latency = asNumber(pickValue(current, ['latency'], pickValue(latestPoint, ['latency'], pickValue(item, ['latency', 'latency_ms', 'latencyMs', 'response_time', 'responseTime', 'last_latency'], 0))))
  const time = formatCheckedTime(pickValue(current, ['timestamp'], pickValue(latestPoint, ['time', 'timestamp'], pickValue(item, ['time', 'checked_at', 'checkedAt', 'last_check', 'lastCheck', 'last_monitor', 'lastMonitor'], ''))))
  let trendSource: unknown[] = timeline.length
    ? timeline.slice(-18)
    : fallbackTrend(availability, index)
  const currentTimestamp = asNumber(pickValue(current, ['timestamp'], 0), 0)
  const latestTimestamp = asNumber(pickValue(latestPoint, ['timestamp'], 0), 0)
  if (currentTimestamp > 0 && currentTimestamp >= latestTimestamp) {
    const currentPoint = {
      status: pickValue(current, ['status'], pickValue(item, ['current_status'], undefined)),
      latency,
      timestamp: currentTimestamp,
      availability: statusAvailability(pickValue(current, ['status'], undefined)),
    }
    if (trendSource.length) trendSource[trendSource.length - 1] = currentPoint
    else trendSource.push(currentPoint)
  }

  const anchorTimestamp = currentTimestamp > 0
    ? currentTimestamp
    : asNumber(pickValue(trendSource[trendSource.length - 1], ['timestamp', 'displayTimestamp'], 0), 0)
  const anchorMs = timestampToMs(anchorTimestamp)
  if (anchorMs > 0) {
    const intervalStep = trendBucketStepMs(monitorIntervalMs(item, layer))
    trendSource = trendSource.map((point, pointIndex) => {
      const displayTimestamp = anchorMs - ((trendSource.length - 1 - pointIndex) * intervalStep)
      if (point && typeof point === 'object') return { ...(point as Record<string, unknown>), displayTimestamp }
      return { tone: point, displayTimestamp }
    })
  }

  return {
    name,
    availability,
    latency,
    time,
    points: trendSource.map((point, pointIndex) => normalizeTrendPoint(point, { availability, latency, time }, pointIndex)),
  }
}

function buildMonitorTrendMap(items: unknown[]) {
  const map = new Map<string, MonitorTrend>()
  items.forEach((item, index) => {
    const trend = normalizeMonitorTrend(item, index)
    if (!trend) return
    const key = normalizeMonitorKey(trend.name)
    const existing = map.get(key)
    if (!existing || trend.availability < existing.availability) {
      map.set(key, trend)
    }
  })
  return map
}

onMounted(loadGroups)
</script>

<style scoped>
.monitor-trend-row {
  display: inline-flex;
  align-items: center;
  gap: 3px;
  min-width: 0;
}

.monitor-trend-block {
  display: inline-block;
  height: 16px;
  width: 8px;
  flex: 0 0 auto;
  border-radius: 3px;
  background: #94a3b8;
  box-shadow: inset 0 -1px 0 rgba(0, 0, 0, 0.18);
}

.monitor-trend-green {
  background: #22c55e;
}

.monitor-trend-yellow {
  background: #eab308;
}

.monitor-trend-red {
  background: #ef4444;
}
</style>

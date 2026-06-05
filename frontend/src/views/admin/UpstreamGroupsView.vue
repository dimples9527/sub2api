<template>
  <AppLayout>
    <div class="mx-auto max-w-7xl space-y-5">
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

      <div v-else class="overflow-hidden rounded-lg border border-gray-200 bg-white dark:border-dark-700 dark:bg-dark-800">
        <div class="flex items-center justify-between border-b border-gray-100 px-4 py-3 text-sm text-gray-500 dark:border-dark-700 dark:text-gray-400">
          <span>共 {{ filteredGroups.length }} 条记录</span>
          <span>数据来自配置的上游分组 URL</span>
        </div>
        <div v-if="filteredGroups.length === 0" class="grid min-h-56 place-items-center text-sm text-gray-500 dark:text-gray-400">
          暂无匹配记录
        </div>
        <div v-else class="overflow-x-auto">
          <table class="min-w-full text-left text-sm">
            <thead class="bg-gray-50 text-xs font-semibold text-gray-500 dark:bg-dark-700/60 dark:text-gray-400">
              <tr>
                <th class="whitespace-nowrap px-4 py-3">上游 ID</th>
                <th class="min-w-56 px-4 py-3">上游分组名称</th>
                <th class="whitespace-nowrap px-4 py-3">平台</th>
                <th class="whitespace-nowrap px-4 py-3">上游倍率</th>
                <th class="whitespace-nowrap px-4 py-3">状态</th>
                <th class="min-w-52 px-4 py-3">本地分组名称</th>
                <th class="whitespace-nowrap px-4 py-3">本地倍率</th>
                <th class="min-w-72 px-4 py-3">描述</th>
                <th class="whitespace-nowrap px-4 py-3">更新时间</th>
              </tr>
            </thead>
            <tbody class="divide-y divide-gray-100 dark:divide-dark-700">
              <tr v-for="group in filteredGroups" :key="String(group.id)" class="transition hover:bg-gray-50 dark:hover:bg-dark-700/60">
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
                <td class="px-4 py-4">
                  <span v-if="group.local_group_name" class="break-words font-medium text-gray-900 dark:text-white">
                    {{ group.local_group_name }}
                  </span>
                  <span v-else class="text-gray-400 dark:text-gray-500">未匹配</span>
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
  </AppLayout>
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import AppLayout from '@/components/layout/AppLayout.vue'
import Icon from '@/components/icons/Icon.vue'
import { getUpstreamAvailableGroups, type UpstreamAvailableGroup } from '@/api/admin/groups'

type MatchStatus = '' | 'matched' | 'unmatched'

const loading = ref(false)
const error = ref('')
const search = ref('')
const platform = ref('')
const matchStatus = ref<MatchStatus>('')
const groups = ref<UpstreamAvailableGroup[]>([])

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
  try {
    groups.value = await getUpstreamAvailableGroups()
  } catch (err) {
    const e = err as { message?: string }
    error.value = e.message || '上游分组加载失败'
  } finally {
    loading.value = false
  }
}

function hasLocalMatch(group: UpstreamAvailableGroup) {
  return group.local_group_id != null || Boolean(group.local_group_name)
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

function statusClass(status?: string) {
  if (status === 'active') {
    return 'bg-emerald-100 text-emerald-700 dark:bg-emerald-900/40 dark:text-emerald-300'
  }
  return 'bg-gray-100 text-gray-600 dark:bg-dark-700 dark:text-gray-300'
}

onMounted(loadGroups)
</script>

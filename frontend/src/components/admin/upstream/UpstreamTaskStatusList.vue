<template>
  <section class="panel">
    <div class="panel-header"><h2>{{ title }}</h2></div>
    <div v-if="tasks.length" class="task-list">
      <button v-for="task in tasks" :key="task.key" type="button" class="task-row" @click="$emit('open', task)">
        <span class="task-main"><strong>{{ task.name }}</strong><small>{{ taskHint(task) }}</small></span>
        <span :class="['task-status', statusClass(task)]">{{ statusLabel(task) }}</span>
      </button>
    </div>
    <div v-else class="empty-copy">{{ emptyText }}</div>
  </section>
</template>

<script setup lang="ts">
import type { UpstreamDashboardTask } from '@/api/admin/upstreamDashboard'

defineProps<{ title: string; emptyText: string; tasks: UpstreamDashboardTask[] }>()
defineEmits<{ open: [task: UpstreamDashboardTask] }>()

function statusLabel(task: UpstreamDashboardTask) {
  if (!task.enabled) return '未启用'
  if (task.last_run_status === 'failed') return '运行失败'
  if (task.affected_count > 0) return `影响 ${task.affected_count}`
  return '正常'
}

function statusClass(task: UpstreamDashboardTask) {
  if (!task.enabled) return 'is-muted'
  if (task.last_run_status === 'failed') return 'is-danger'
  if (task.affected_count > 0) return 'is-warning'
  return 'is-good'
}

function taskHint(task: UpstreamDashboardTask) {
  if (task.next_run_at) return `下次 ${new Date(task.next_run_at).toLocaleString()}`
  if (task.last_run_message) return task.last_run_message
  return task.last_run_at ? `上次 ${new Date(task.last_run_at).toLocaleString()}` : '暂无运行记录'
}
</script>

<style scoped>
.panel { @apply rounded-2xl border border-gray-200 bg-white p-5 shadow-sm dark:border-gray-700 dark:bg-gray-800; }
.panel-header { @apply mb-3 flex items-center justify-between; }
.panel-header h2 { @apply text-base font-bold text-gray-950 dark:text-white; }
.task-list { @apply divide-y divide-gray-100 dark:divide-gray-700; }
.task-row { @apply flex min-h-16 w-full items-center justify-between gap-3 rounded-lg px-2 text-left transition-colors hover:bg-gray-50 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary-500 dark:hover:bg-gray-700/60; }
.task-main { @apply min-w-0; }
.task-main strong { @apply block text-sm text-gray-900 dark:text-white; }
.task-main small { @apply mt-1 block truncate text-xs text-gray-500 dark:text-gray-400; }
.task-status { @apply shrink-0 rounded-full px-2.5 py-1 text-xs font-semibold; }
.is-good { @apply bg-emerald-50 text-emerald-700 dark:bg-emerald-900/30 dark:text-emerald-300; }
.is-warning { @apply bg-amber-50 text-amber-700 dark:bg-amber-900/30 dark:text-amber-300; }
.is-danger { @apply bg-red-50 text-red-700 dark:bg-red-900/30 dark:text-red-300; }
.is-muted { @apply bg-gray-100 text-gray-500 dark:bg-gray-700 dark:text-gray-300; }
.empty-copy { @apply py-10 text-center text-sm text-gray-400; }
</style>

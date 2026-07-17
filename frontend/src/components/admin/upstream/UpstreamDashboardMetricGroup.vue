<template>
  <section class="metric-group" :class="`metric-group-${tone}`">
    <div class="metric-group-label">{{ title }}</div>
    <div class="metric-grid">
      <div v-for="item in items" :key="item.label" class="metric-item">
        <strong>{{ item.value }}</strong>
        <span>{{ item.label }}</span>
        <small v-if="item.hint" :class="item.hintTone ? `hint-${item.hintTone}` : ''">{{ item.hint }}</small>
      </div>
    </div>
  </section>
</template>

<script setup lang="ts">
export interface DashboardMetricItem {
  label: string
  value: string
  hint?: string
  hintTone?: 'good' | 'warning' | 'danger'
}

defineProps<{
  title: string
  tone: 'blue' | 'green' | 'amber'
  items: DashboardMetricItem[]
}>()
</script>

<style scoped>
.metric-group { @apply rounded-2xl border border-gray-200 bg-white p-5 shadow-sm dark:border-gray-700 dark:bg-gray-800; }
.metric-group-blue { border-top: 3px solid #315efb; }
.metric-group-green { border-top: 3px solid #16a36a; }
.metric-group-amber { border-top: 3px solid #f59e0b; }
.metric-group-label { @apply mb-4 text-xs font-semibold uppercase tracking-[0.16em] text-gray-500 dark:text-gray-400; }
.metric-grid { @apply grid grid-cols-3 gap-4; }
.metric-item { @apply min-w-0; }
.metric-item strong { @apply block truncate text-2xl font-bold tabular-nums text-gray-950 dark:text-white; }
.metric-item span { @apply mt-1 block text-xs font-medium text-gray-600 dark:text-gray-300; }
.metric-item small { @apply mt-1 block truncate text-xs text-gray-400; }
.hint-good { @apply text-emerald-600 dark:text-emerald-400; }
.hint-warning { @apply text-amber-600 dark:text-amber-400; }
.hint-danger { @apply text-red-600 dark:text-red-400; }
@media (max-width: 640px) { .metric-group { @apply p-4; } .metric-grid { @apply gap-2; } .metric-item strong { @apply text-xl; } }
</style>

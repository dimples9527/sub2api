<template>
  <section class="panel">
    <div class="panel-header"><h2>{{ title }}</h2><span>{{ subtitle }}</span></div>
    <div v-if="items.length" class="ranking-list">
      <button v-for="(item, index) in items" :key="item.key" type="button" class="ranking-row" @click="$emit('open', item)">
        <span class="rank">{{ index + 1 }}</span>
        <span class="ranking-name"><strong>{{ item.name }}</strong><small>{{ item.meta }}</small></span>
        <strong class="ranking-value">{{ item.value }}</strong>
      </button>
    </div>
    <div v-else class="empty-copy">{{ emptyText }}</div>
  </section>
</template>

<script setup lang="ts">
export interface DashboardRankingItem {
  key: string
  name: string
  meta: string
  value: string
  target?: string
}

defineProps<{ title: string; subtitle: string; emptyText: string; items: DashboardRankingItem[] }>()
defineEmits<{ open: [item: DashboardRankingItem] }>()
</script>

<style scoped>
.panel { @apply rounded-2xl border border-gray-200 bg-white p-5 shadow-sm dark:border-gray-700 dark:bg-gray-800; }
.panel-header { @apply mb-3 flex items-end justify-between gap-3; }
.panel-header h2 { @apply text-base font-bold text-gray-950 dark:text-white; }
.panel-header span { @apply text-xs text-gray-400; }
.ranking-list { @apply divide-y divide-gray-100 dark:divide-gray-700; }
.ranking-row { @apply grid min-h-14 w-full grid-cols-[28px_1fr_auto] items-center gap-3 rounded-lg px-2 text-left transition-colors hover:bg-gray-50 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary-500 dark:hover:bg-gray-700/60; }
.rank { @apply flex h-6 w-6 items-center justify-center rounded-md bg-gray-100 text-xs font-bold text-gray-500 dark:bg-gray-700 dark:text-gray-300; }
.ranking-name { @apply min-w-0; }
.ranking-name strong { @apply block truncate text-sm text-gray-900 dark:text-white; }
.ranking-name small { @apply mt-0.5 block truncate text-xs text-gray-400; }
.ranking-value { @apply text-sm tabular-nums text-gray-900 dark:text-white; }
.empty-copy { @apply py-10 text-center text-sm text-gray-400; }
</style>

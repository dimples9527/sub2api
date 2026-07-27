<template>
  <div class="supplier-group-trend" :title="containerTitle">
    <div class="supplier-group-trend__meta">
      <span class="supplier-group-trend__rate">{{ rateText }}</span>
      <span class="supplier-group-trend__time">{{ timeText }}</span>
    </div>
    <div class="supplier-group-trend__bars" :aria-label="label">
      <template v-if="loading">
        <span
          v-for="index in BAR_COUNT"
          :key="index"
          class="supplier-group-trend__bar supplier-group-trend__bar--loading"
        />
      </template>
      <template v-else-if="row?.trend?.length">
        <span
          v-for="(point, index) in visibleTrend"
          :key="`${point.time}-${index}`"
          :class="['supplier-group-trend__bar', `supplier-group-trend__bar--${point.tone}`]"
          :title="pointTitle(point)"
        />
      </template>
      <span v-else class="supplier-group-trend__empty">{{ emptyText }}</span>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'

import type { SupplierGroupMonitorTrendPoint, SupplierGroupMonitorTrendRow } from '@/utils/supplierGroupMonitorTrend'

const BAR_COUNT = 18

const props = withDefaults(defineProps<{
  row?: SupplierGroupMonitorTrendRow
  loading?: boolean
  error?: string
  emptyText?: string
  loadingText?: string
  label?: string
}>(), {
  row: undefined,
  loading: false,
  error: '',
  emptyText: '-',
  loadingText: '加载中',
  label: '可用率趋势'
})

const visibleTrend = computed(() => props.row?.trend?.slice(-BAR_COUNT) || [])
const rateText = computed(() => {
  if (props.loading) return props.loadingText
  if (!props.row) return props.error || props.emptyText
  return `${formatPercent(props.row.availability)}%`
})
const timeText = computed(() => {
  if (props.loading || !props.row?.time || props.row.time === '--:--') return ''
  return props.row.time
})
const containerTitle = computed(() => {
  if (props.loading) return props.loadingText
  if (props.error && !props.row) return props.error
  if (!props.row) return props.emptyText
  return `${props.row.provider} ${formatPercent(props.row.availability)}% ${props.row.latency || 0}ms ${props.row.time}`
})

function pointTitle(point: SupplierGroupMonitorTrendPoint) {
  return `${point.time} ${point.statusText} ${formatPercent(point.availability)}% ${point.latency || 0}ms`
}

function formatPercent(value: number) {
  const n = Number(value)
  if (!Number.isFinite(n)) return '0'
  return n % 1 === 0 ? n.toFixed(0) : n.toFixed(2)
}
</script>

<style scoped>
.supplier-group-trend {
  @apply flex min-w-[9.5rem] max-w-[10.5rem] flex-col gap-1;
}

.supplier-group-trend__meta {
  @apply flex h-4 items-center justify-between gap-2 text-[11px] leading-4;
}

.supplier-group-trend__rate {
  @apply truncate font-mono font-semibold text-gray-800 dark:text-gray-100;
}

.supplier-group-trend__time {
  @apply shrink-0 font-mono text-gray-400 dark:text-gray-500;
}

.supplier-group-trend__bars {
  @apply flex h-7 w-full items-end gap-[2px];
}

.supplier-group-trend__bar {
  @apply block h-6 flex-1 rounded-[2px];
}

.supplier-group-trend__bar--green {
  background: #2fa84f;
}

.supplier-group-trend__bar--yellow {
  background: #caa51d;
}

.supplier-group-trend__bar--red {
  background: #de4b52;
}

.supplier-group-trend__bar--loading {
  @apply animate-pulse bg-gray-200 dark:bg-dark-700;
}

.supplier-group-trend__empty {
  @apply flex h-6 w-full items-center text-xs text-gray-400 dark:text-gray-500;
}
</style>

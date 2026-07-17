<template>
  <div class="upstream-trend" :title="containerTitle">
    <div class="upstream-trend__meta">
      <span class="upstream-trend__rate">{{ rateText }}</span>
      <span class="upstream-trend__time">{{ timeText }}</span>
    </div>
    <div class="upstream-trend__bars" :aria-label="label">
      <template v-if="loading">
        <span
          v-for="index in BAR_COUNT"
          :key="index"
          class="upstream-trend__bar upstream-trend__bar--loading"
        />
      </template>
      <template v-else-if="row?.trend?.length">
        <span
          v-for="(point, index) in visibleTrend"
          :key="`${point.time}-${index}`"
          :class="['upstream-trend__bar', `upstream-trend__bar--${point.tone}`]"
          :title="pointTitle(point)"
        />
      </template>
      <span v-else class="upstream-trend__empty">{{ emptyText }}</span>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'

import type { UpstreamMonitorTrendPoint, UpstreamMonitorTrendRow } from '@/utils/upstreamMonitorTrend'

const BAR_COUNT = 18

const props = withDefaults(defineProps<{
  row?: UpstreamMonitorTrendRow
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
  loadingText: 'Loading',
  label: 'Availability trend'
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

function pointTitle(point: UpstreamMonitorTrendPoint) {
  return `${point.time} ${point.statusText} ${formatPercent(point.availability)}% ${point.latency || 0}ms`
}

function formatPercent(value: number) {
  const n = Number(value)
  if (!Number.isFinite(n)) return '0'
  return n % 1 === 0 ? n.toFixed(0) : n.toFixed(2)
}
</script>

<style scoped>
.upstream-trend {
  @apply flex min-w-[9.5rem] max-w-[10.5rem] flex-col gap-1;
}

.upstream-trend__meta {
  @apply flex h-4 items-center justify-between gap-2 text-[11px] leading-4;
}

.upstream-trend__rate {
  @apply truncate font-mono font-semibold text-gray-800 dark:text-gray-100;
}

.upstream-trend__time {
  @apply shrink-0 font-mono text-gray-400 dark:text-gray-500;
}

.upstream-trend__bars {
  @apply flex h-7 w-full items-end gap-[2px];
}

.upstream-trend__bar {
  @apply block h-6 flex-1 rounded-[2px];
}

.upstream-trend__bar--green {
  background: #2fa84f;
}

.upstream-trend__bar--yellow {
  background: #caa51d;
}

.upstream-trend__bar--red {
  background: #de4b52;
}

.upstream-trend__bar--loading {
  @apply animate-pulse bg-gray-200 dark:bg-dark-700;
}

.upstream-trend__empty {
  @apply flex h-6 w-full items-center text-xs text-gray-400 dark:text-gray-500;
}
</style>

<template>
  <div class="space-y-4">
    <div class="flex h-3 w-full overflow-hidden rounded-full bg-gray-100 dark:bg-dark-700">
      <div
        v-for="seg in segments"
        :key="seg.key"
        class="h-full"
        :class="SEGMENT_BAR_CLASSES[seg.key]"
        :style="{ width: seg.percent + '%' }"
        :title="`${label(seg.key)} ${formatMs(seg.ms)}`"
      ></div>
    </div>

    <div class="grid gap-x-8 gap-y-1.5 sm:grid-cols-2">
      <div v-for="seg in segments" :key="seg.key" class="flex items-center gap-2 text-xs">
        <span class="h-2 w-2 shrink-0 rounded-full" :class="SEGMENT_BAR_CLASSES[seg.key]"></span>
        <span class="flex-1 truncate text-gray-500 dark:text-gray-400">{{ label(seg.key) }}</span>
        <span class="font-medium tabular-nums text-gray-900 dark:text-white">{{ formatMs(seg.ms) }}</span>
        <span class="w-9 text-right tabular-nums text-gray-400 dark:text-gray-500">{{ Math.round(seg.percent) }}%</span>
      </div>
    </div>

    <div class="flex flex-wrap items-center gap-2 border-t border-gray-200 pt-3 text-xs dark:border-dark-700">
      <span
        data-testid="latency-local-overhead"
        class="rounded px-2 py-0.5 font-medium bg-rose-50 text-rose-600 ring-1 ring-inset ring-rose-200 dark:bg-rose-500/10 dark:text-rose-300 dark:ring-rose-500/30"
      >
        {{ t('usage.latencyLocalOverhead') }} {{ formatMs(localOverheadMs) }}
      </span>
      <span
        class="rounded px-2 py-0.5 font-medium bg-sky-50 text-sky-600 ring-1 ring-inset ring-sky-200 dark:bg-sky-500/10 dark:text-sky-300 dark:ring-sky-500/30"
      >
        {{ t('usage.latencyUpstreamWait') }} {{ formatMs(phases.first_byte_ms ?? 0) }}
      </span>
      <span
        v-if="phases.conn_reused != null"
        class="rounded bg-gray-100 px-2 py-0.5 text-gray-600 dark:bg-dark-700 dark:text-gray-300"
      >
        {{ phases.conn_reused ? t('usage.latencyConnReused') : t('usage.latencyConnNew') }}
      </span>
    </div>

    <p class="text-[11px] leading-relaxed text-gray-400 dark:text-gray-500">
      {{ t('usage.latencyBreakdownHint') }}
    </p>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import type { LatencyPhases } from '@/api/admin/usage'

type SegmentKey = 'build' | 'slotWait' | 'connect' | 'tls' | 'firstByte' | 'other' | 'stream'

const props = defineProps<{
  phases: LatencyPhases
  durationMs?: number | null
  firstTokenMs?: number | null
}>()

const { t } = useI18n()

// 配色由归因语义驱动：排队=本地连接池争抢(红)、等上游首字=上游(蓝)、流传输=正常产出(绿)。
const SEGMENT_BAR_CLASSES: Record<SegmentKey, string> = {
  build: 'bg-slate-400 dark:bg-slate-500',
  slotWait: 'bg-rose-500',
  connect: 'bg-amber-500',
  tls: 'bg-violet-500',
  firstByte: 'bg-sky-500',
  other: 'bg-gray-300 dark:bg-gray-600',
  stream: 'bg-emerald-500',
}

const SEGMENT_LABEL_KEYS: Record<SegmentKey, string> = {
  build: 'usage.latencyPhaseBuild',
  slotWait: 'usage.latencyPhaseSlotWait',
  connect: 'usage.latencyPhaseConnect',
  tls: 'usage.latencyPhaseTls',
  firstByte: 'usage.latencyPhaseFirstByte',
  other: 'usage.latencyPhaseOther',
  stream: 'usage.latencyPhaseStream',
}

const label = (key: SegmentKey): string => t(SEGMENT_LABEL_KEYS[key])

const formatMs = (ms: number): string => (ms < 1000 ? `${ms}ms` : `${(ms / 1000).toFixed(2)}s`)

const localOverheadMs = computed(
  () =>
    (props.phases.build_ms ?? 0) +
    (props.phases.slot_wait_ms ?? 0) +
    (props.phases.connect_ms ?? 0) +
    (props.phases.tls_ms ?? 0)
)

const segments = computed(() => {
  const p = props.phases
  const list: { key: SegmentKey; ms: number }[] = []
  const push = (key: SegmentKey, ms: number | null | undefined) => {
    if (ms != null && ms > 0) list.push({ key, ms })
  }

  push('build', p.build_ms)
  push('slotWait', p.slot_wait_ms)
  push('connect', p.connect_ms)
  push('tls', p.tls_ms)
  push('firstByte', p.first_byte_ms)

  // 分解只覆盖最终成功的那次 attempt。首字与首个 token 之间的差额（重试前置耗时、
  // 上游首字节到首个可用 token）无法归到任何一段，单列出来才不会被静默吞掉。
  const attempted = list.reduce((sum, seg) => sum + seg.ms, 0)
  const firstTokenBoundary = props.firstTokenMs ?? props.durationMs ?? attempted
  push('other', firstTokenBoundary - attempted)

  // 流传输由总耗时减首字派生，不落库。
  if (props.durationMs != null && props.firstTokenMs != null) {
    push('stream', props.durationMs - props.firstTokenMs)
  }

  const total = list.reduce((sum, seg) => sum + seg.ms, 0) || 1
  return list.map((seg) => ({ ...seg, percent: (seg.ms / total) * 100 }))
})
</script>

<template>
  <BaseDialog
    :show="show"
    :title="t('usage.latencyBreakdown')"
    width="wide"
    close-on-click-outside
    @close="emit('close')"
  >
    <div class="mb-4 space-y-1 text-xs text-gray-500 dark:text-gray-400">
      <div class="flex items-center gap-2">
        <span>{{ t('usage.latencyFirstToken') }}</span>
        <span class="font-medium tabular-nums text-gray-900 dark:text-white">{{ formatMs(row?.first_token_ms) }}</span>
        <span class="text-gray-300 dark:text-dark-600">/</span>
        <span>{{ t('usage.latencyDuration') }}</span>
        <span class="font-medium tabular-nums text-gray-900 dark:text-white">{{ formatMs(row?.duration_ms) }}</span>
      </div>
      <div v-if="row?.request_id" class="break-all font-mono">{{ row.request_id }}</div>
    </div>

    <div v-if="loading" class="py-8 text-center text-sm text-gray-500 dark:text-gray-400">
      {{ t('common.loading') }}
    </div>
    <div v-else-if="failed" class="py-8 text-center text-sm text-red-500 dark:text-red-400">
      {{ t('usage.latencyBreakdownFailed') }}
    </div>
    <div v-else-if="!phases" class="py-8 text-center text-sm text-gray-500 dark:text-gray-400">
      {{ t('usage.latencyBreakdownEmpty') }}
    </div>
    <LatencyWaterfall
      v-else
      :phases="phases"
      :duration-ms="row?.duration_ms ?? null"
      :first-token-ms="row?.first_token_ms ?? null"
    />
  </BaseDialog>
</template>

<script setup lang="ts">
import { ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import BaseDialog from '@/components/common/BaseDialog.vue'
import LatencyWaterfall from './LatencyWaterfall.vue'
import { getLatencyBreakdown, type LatencyPhases } from '@/api/admin/usage'
import type { AdminUsageLog } from '@/types'

const props = defineProps<{
  show: boolean
  row: AdminUsageLog | null
}>()

const emit = defineEmits<{ close: [] }>()

const { t } = useI18n()

const phases = ref<LatencyPhases | null>(null)
const loading = ref(false)
const failed = ref(false)
// 弹窗可以在上一次请求返回前被关掉再换行打开，用序号丢弃过期响应。
let requestSeq = 0

const formatMs = (ms: number | null | undefined): string => {
  if (ms == null) return '-'
  return ms < 1000 ? `${ms}ms` : `${(ms / 1000).toFixed(2)}s`
}

const load = async (requestId: string, apiKeyId: number) => {
  const seq = ++requestSeq
  loading.value = true
  failed.value = false
  phases.value = null
  try {
    const data = await getLatencyBreakdown(requestId, apiKeyId)
    if (seq !== requestSeq) return
    phases.value = data
  } catch {
    if (seq !== requestSeq) return
    failed.value = true
  } finally {
    if (seq === requestSeq) loading.value = false
  }
}

watch(
  () => [props.show, props.row?.request_id, props.row?.api_key_id] as const,
  ([show, requestId, apiKeyId]) => {
    if (!show || !requestId || !apiKeyId) {
      requestSeq++
      loading.value = false
      failed.value = false
      phases.value = null
      return
    }
    void load(requestId, apiKeyId)
  },
  { immediate: true }
)
</script>

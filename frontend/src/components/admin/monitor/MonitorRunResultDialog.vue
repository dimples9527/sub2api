<template>
  <BaseDialog
    :show="show"
    :title="t('admin.channelMonitor.runResultTitle')"
    width="normal"
    @close="$emit('close')"
  >
    <div class="run-result-list space-y-2">
      <div
        v-for="r in results"
        :key="r.model"
        class="run-result-item flex items-center justify-between rounded-lg border border-gray-200 px-3 py-2 text-sm dark:border-dark-600"
      >
        <div class="run-result-copy flex flex-col">
          <span class="font-medium text-gray-900 dark:text-white">{{ r.model }}</span>
          <span v-if="r.message" class="text-xs text-gray-500 dark:text-gray-400">{{ r.message }}</span>
        </div>
        <div class="run-result-meta flex items-center gap-2">
          <span
            class="inline-flex items-center rounded-full px-2 py-0.5 text-[11px]"
            :class="statusBadgeClass(r.status)"
          >
            {{ statusLabel(r.status) }}
          </span>
          <span class="text-xs text-gray-500 dark:text-gray-400">{{ formatLatency(r.latency_ms) }} ms</span>
        </div>
      </div>
    </div>
    <template #footer>
      <div class="flex justify-end">
        <button @click="$emit('close')" class="btn btn-primary">
          {{ t('common.close') }}
        </button>
      </div>
    </template>
  </BaseDialog>
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import type { CheckResult } from '@/api/admin/channelMonitor'
import BaseDialog from '@/components/common/BaseDialog.vue'
import { useChannelMonitorFormat } from '@/composables/useChannelMonitorFormat'

defineProps<{
  show: boolean
  results: CheckResult[]
}>()

defineEmits<{
  (e: 'close'): void
}>()

const { t } = useI18n()
const { statusLabel, statusBadgeClass, formatLatency } = useChannelMonitorFormat()
</script>

<style scoped>
@media (max-width: 520px) {
  .run-result-item {
    align-items: flex-start;
    gap: 8px;
    padding: 10px;
  }

  .run-result-copy {
    min-width: 0;
  }

  .run-result-copy span {
    overflow-wrap: anywhere;
  }

  .run-result-meta {
    flex: 0 0 auto;
    flex-direction: column;
    align-items: flex-end;
    gap: 4px;
  }
}
</style>

<template>
  <div class="monitor-actions-cell flex flex-wrap items-center justify-end gap-1 md:justify-start">
    <button
      @click="$emit('run', row)"
      :disabled="running"
      class="flex flex-col items-center gap-0.5 rounded-lg p-1.5 text-gray-500 transition-colors hover:bg-gray-100 hover:text-primary-600 dark:hover:bg-dark-700 dark:hover:text-primary-400"
    >
      <Icon name="refresh" size="sm" :class="running ? 'animate-spin' : ''" />
      <span class="text-xs">{{ t('admin.channelMonitor.runNow') }}</span>
    </button>
    <button
      @click="$emit('edit', row)"
      class="flex flex-col items-center gap-0.5 rounded-lg p-1.5 text-gray-500 transition-colors hover:bg-gray-100 hover:text-primary-600 dark:hover:bg-dark-700 dark:hover:text-primary-400"
    >
      <Icon name="edit" size="sm" />
      <span class="text-xs">{{ t('common.edit') }}</span>
    </button>
    <button
      @click="$emit('delete', row)"
      class="flex flex-col items-center gap-0.5 rounded-lg p-1.5 text-gray-500 transition-colors hover:bg-red-50 hover:text-red-600 dark:hover:bg-red-900/20 dark:hover:text-red-400"
    >
      <Icon name="trash" size="sm" />
      <span class="text-xs">{{ t('common.delete') }}</span>
    </button>
  </div>
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import type { ChannelMonitor } from '@/api/admin/channelMonitor'
import Icon from '@/components/icons/Icon.vue'

defineProps<{
  row: ChannelMonitor
  running: boolean
}>()

defineEmits<{
  (e: 'run', row: ChannelMonitor): void
  (e: 'edit', row: ChannelMonitor): void
  (e: 'delete', row: ChannelMonitor): void
}>()

const { t } = useI18n()
</script>

<style scoped>
@media (max-width: 767px) {
  .monitor-actions-cell {
    justify-content: flex-start;
    gap: 6px;
  }

  .monitor-actions-cell button {
    min-height: 30px;
    flex: 0 0 auto;
    flex-direction: row;
    gap: 4px;
    border: 1px solid #e2e8f0;
    border-radius: 7px;
    background: #fff;
    padding: 0 8px;
    color: #475569;
  }

  .monitor-actions-cell button span {
    font-size: 12px;
    line-height: 1;
  }

  :global(.dark) .monitor-actions-cell button {
    border-color: #334155;
    background: #111827;
    color: #cbd5e1;
  }
}

@media (max-width: 420px) {
  .monitor-actions-cell {
    justify-content: flex-start;
  }

  .monitor-actions-cell button {
    min-width: 0;
  }
}
</style>

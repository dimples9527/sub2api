<template>
  <div
    v-if="show"
    class="ug-rate-fix-preview-dialog fixed inset-0 z-50 flex items-center justify-center bg-black/45 px-4 py-6"
    role="dialog"
    aria-modal="true"
    :aria-labelledby="titleId"
    @click.self="cancel"
  >
    <div class="w-full max-w-2xl overflow-hidden rounded-xl border border-gray-200 bg-white shadow-2xl dark:border-dark-700 dark:bg-dark-800">
      <div class="border-b border-gray-100 px-5 py-4 dark:border-dark-700">
        <h3 :id="titleId" class="text-lg font-semibold text-gray-950 dark:text-white">
          {{ t('admin.upstreamGroups.rateFixPreviewTitle') }}
        </h3>
        <p class="mt-1 text-sm text-gray-500 dark:text-gray-400">
          {{ t('admin.upstreamGroups.rateFixPreviewDescription', { count: items.length }) }}
        </p>
      </div>

      <div class="max-h-[55vh] overflow-y-auto px-5 py-4">
        <div class="mb-3 rounded-lg border border-amber-200 bg-amber-50 px-3 py-2 text-sm text-amber-800 dark:border-amber-800/60 dark:bg-amber-950/30 dark:text-amber-200">
          {{ t('admin.upstreamGroups.rateFixPreviewWarning', { count: items.length }) }}
        </div>
        <div class="overflow-hidden rounded-lg border border-gray-200 dark:border-dark-700">
          <div
            v-for="item in items"
            :key="`${item.provider_slug}:${item.upstream_group_key}:${item.local_group_id || 0}`"
            class="grid gap-2 border-b border-gray-100 px-4 py-3 last:border-b-0 sm:grid-cols-[minmax(0,1fr)_auto] dark:border-dark-700"
          >
            <div class="min-w-0">
              <div class="truncate text-sm font-semibold text-gray-950 dark:text-white">
                {{ item.local_group_name || item.upstream_group_name }}
              </div>
              <div class="mt-1 truncate text-xs text-gray-500 dark:text-gray-400">
                {{ item.provider_name || item.provider_slug }} · {{ item.upstream_group_name }}
              </div>
            </div>
            <div class="flex items-center gap-2 font-mono text-sm">
              <span class="text-gray-500 dark:text-gray-400">{{ formatRate(item.local_rate) }}</span>
              <span aria-hidden="true" class="text-gray-400">→</span>
              <span class="font-semibold text-amber-700 dark:text-amber-300">{{ formatRate(item.upstream_rate) }}</span>
            </div>
          </div>
        </div>
      </div>

      <div class="flex justify-end gap-2 border-t border-gray-100 px-5 py-4 dark:border-dark-700">
        <button type="button" class="ug-rate-fix-preview-cancel btn btn-secondary btn-sm" :disabled="loading" @click="cancel">
          {{ t('common.cancel') }}
        </button>
        <button type="button" class="ug-rate-fix-preview-confirm btn btn-primary btn-sm" :disabled="loading || items.length === 0" @click="emit('confirm')">
          {{ loading ? t('admin.upstreamGroups.rateFixApplying') : t('admin.upstreamGroups.rateFixConfirmAll', { count: items.length }) }}
        </button>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import type { UpstreamGroupComparison } from '@/api/admin/upstreamManagement'

defineProps<{
  show: boolean
  items: UpstreamGroupComparison[]
  loading: boolean
}>()

const emit = defineEmits<{
  confirm: []
  cancel: []
}>()

const { t } = useI18n()
const titleId = 'upstream-rate-fix-preview-title'

function cancel() {
  emit('cancel')
}

function formatRate(value?: number) {
  if (!Number.isFinite(value)) return '-'
  return Number(value).toFixed(4).replace(/\.?0+$/, '')
}
</script>

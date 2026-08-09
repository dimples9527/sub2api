<template>
  <Teleport to="body">
    <div
      v-if="open && position"
      ref="rootRef"
      class="animate-in fade-in slide-in-from-top-2 fixed z-[100000020] w-max max-w-[calc(100vw-16px)] overflow-visible rounded-xl bg-white shadow-lg ring-1 ring-black/5 duration-200 sm:min-w-[380px] dark:bg-dark-800 dark:ring-white/10"
      style="pointer-events: auto !important;"
      :style="{
        top: position.top !== undefined ? position.top + 'px' : undefined,
        bottom: position.bottom !== undefined ? position.bottom + 'px' : undefined,
        left: position.left + 'px'
      }"
      data-tour="key-group-selector-popover"
    >
      <!-- 平台过滤 + 搜索：复用通用 Select，和创建密钥弹窗保持一致 -->
      <div class="space-y-2 rounded-t-xl border-b border-gray-100 p-2 dark:border-dark-700">
        <div data-tour="key-list-group-platform-wrap">
          <Select
            v-model="platformFilter"
            :options="platformOptions"
            :placeholder="t('keys.allPlatforms')"
            :aria-label="t('keys.platformLabel')"
            data-tour="key-list-group-platform"
          />
        </div>

        <div class="relative">
          <svg
            class="pointer-events-none absolute left-2.5 top-1/2 h-4 w-4 -translate-y-1/2 text-gray-400 dark:text-dark-400"
            fill="none"
            stroke="currentColor"
            viewBox="0 0 24 24"
            stroke-width="2"
            aria-hidden="true"
          >
            <path stroke-linecap="round" stroke-linejoin="round" d="M21 21l-6-6m2-5a7 7 0 11-14 0 7 7 0 0114 0z" />
          </svg>
          <input
            v-model="searchQuery"
            type="text"
            class="group-selector-search-input"
            :placeholder="t('keys.searchGroup')"
            @click.stop
          />
        </div>
      </div>

      <!-- 分组列表 -->
      <div class="max-h-80 overflow-y-auto rounded-b-xl p-1.5">
        <button
          v-for="option in filteredOptions"
          :key="option.value ?? 'null'"
          type="button"
          :class="[
            'flex w-full items-center justify-between rounded-lg px-3 py-2.5 text-sm transition-colors',
            'border-b border-gray-100 last:border-0 dark:border-dark-700',
            isOptionSelected(option)
              ? 'bg-primary-50 dark:bg-primary-900/20'
              : 'hover:bg-gray-100 dark:hover:bg-dark-700'
          ]"
          :title="option.description || undefined"
          @click="emit('select', option.value)"
        >
          <GroupOptionItem
            :name="option.label"
            :platform="option.platform"
            :subscription-type="option.subscriptionType"
            :rate-multiplier="option.rate"
            :user-rate-multiplier="option.userRate"
            :peak-rate-enabled="option.peakRateEnabled"
            :peak-start="option.peakStart"
            :peak-end="option.peakEnd"
            :peak-rate-multiplier="option.peakRateMultiplier"
            :description="option.description"
            :selected="isOptionSelected(option)"
          />
        </button>

        <div
          v-if="filteredOptions.length === 0"
          class="py-4 text-center text-sm text-gray-400 dark:text-gray-500"
        >
          {{ t('keys.noGroupFound') }}
        </div>
      </div>
    </div>
  </Teleport>
</template>

<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import Select from '@/components/common/Select.vue'
import GroupOptionItem from '@/components/common/GroupOptionItem.vue'
import {
  buildGroupBusinessPlatformOptions as buildKeyFormPlatformOptions,
  filterAndSortGroupsByBusinessPlatform as filterAndSortKeyFormGroupOptions,
  type BusinessPlatformFilterValue as KeyFormPlatformFilter
} from '@/features/model-monitor/groupBusinessPlatformFilter'
import type { GroupPlatform, SubscriptionType } from '@/types'

/** 列表切换分组浮层中的分组选项 */
export interface KeyGroupSelectorOption {
  value: number | null
  label: string
  description: string | null
  rate: number
  userRate: number | null
  peakRateEnabled: boolean
  peakStart: string
  peakEnd: string
  peakRateMultiplier: number
  subscriptionType: SubscriptionType
  platform: GroupPlatform
  businessPlatform?: string | null
  businessPlatformName?: string | null
  effectivePlatform?: string | null
  effective_platform?: string | null
  effectivePlatformName?: string | null
  effective_platform_name?: string | null
  actualPlatform?: string | null
  actual_platform?: string | null
}

export interface KeyGroupSelectorPosition {
  top?: number
  bottom?: number
  left: number
}

const props = defineProps<{
  open: boolean
  /** 当前编辑的密钥 ID；切换密钥时重置过滤条件 */
  activeKeyId?: number | null
  position: KeyGroupSelectorPosition | null
  options: KeyGroupSelectorOption[]
  selectedGroupId: number | null
}>()

const emit = defineEmits<{
  select: [groupId: number | null]
}>()

const { t } = useI18n()

const rootRef = ref<HTMLElement | null>(null)
const searchQuery = ref('')
const platformFilter = ref<KeyFormPlatformFilter>('')

const platformOptions = computed(() =>
  buildKeyFormPlatformOptions(props.options, {
    all: t('keys.allPlatforms'),
    platformLabel: (platform) => t(`admin.groups.platforms.${platform}`)
  })
)

const filteredOptions = computed(() => {
  const platformFiltered = filterAndSortKeyFormGroupOptions(props.options, platformFilter.value)
  const query = searchQuery.value.trim().toLowerCase()
  if (!query) return platformFiltered
  return platformFiltered.filter((option) => {
    return (
      option.label.toLowerCase().includes(query) ||
      (option.description && option.description.toLowerCase().includes(query))
    )
  })
})

const resetFilters = () => {
  searchQuery.value = ''
  platformFilter.value = ''
}

const isOptionSelected = (option: KeyGroupSelectorOption) => {
  return (
    props.selectedGroupId === option.value ||
    (props.selectedGroupId == null && option.value === null)
  )
}

/** 供父组件判断点击是否落在浮层内 */
const containsElement = (target: Node | null) => {
  if (!target || !rootRef.value) return false
  return rootRef.value.contains(target)
}

watch(
  () => props.open,
  (open) => {
    if (!open) resetFilters()
  }
)

watch(
  () => props.activeKeyId,
  () => {
    // 在浮层保持打开时切换到另一把密钥，也要清空平台/搜索
    if (props.open) resetFilters()
  }
)

defineExpose({
  containsElement,
  resetFilters
})
</script>

<style scoped>
/* 列表切换分组浮层：搜索框保持和通用 Select 接近的视觉 */
.group-selector-search-input {
  @apply w-full rounded-lg border border-gray-200 bg-gray-50 text-sm leading-5;
  @apply text-gray-900 outline-none transition-colors duration-150;
  @apply hover:border-gray-300 focus:border-primary-300 focus:ring-1 focus:ring-primary-300;
  @apply dark:border-dark-600 dark:bg-dark-700 dark:text-white;
  @apply dark:hover:border-dark-500 dark:focus:border-primary-600 dark:focus:ring-primary-600;
  min-height: 2.125rem;
}

.group-selector-search-input {
  @apply py-1.5 pl-8 pr-3 placeholder-gray-400 dark:placeholder-gray-500;
}
</style>

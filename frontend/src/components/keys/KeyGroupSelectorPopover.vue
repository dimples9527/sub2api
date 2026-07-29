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
      <!-- 平台过滤 + 搜索：平台用自定义下拉，避免原生 option 列表无法美化 -->
      <div class="space-y-2 rounded-t-xl border-b border-gray-100 p-2 dark:border-dark-700">
        <div class="relative" data-tour="key-list-group-platform-wrap">
          <button
            type="button"
            class="group-platform-trigger"
            :class="[
              platformFilter ? 'group-platform-trigger--active' : '',
              platformMenuOpen ? 'group-platform-trigger--open' : ''
            ]"
            :aria-label="t('keys.platformLabel')"
            :aria-expanded="platformMenuOpen"
            aria-haspopup="listbox"
            data-tour="key-list-group-platform"
            @click.stop="platformMenuOpen = !platformMenuOpen"
          >
            <svg
              class="h-4 w-4 flex-shrink-0 text-gray-400 dark:text-dark-400"
              viewBox="0 0 24 24"
              fill="none"
              stroke="currentColor"
              stroke-width="2"
              aria-hidden="true"
            >
              <path stroke-linecap="round" stroke-linejoin="round" d="M4 6h16M4 12h10M4 18h7" />
            </svg>
            <span class="min-w-0 flex-1 truncate text-left">
              {{ platformLabel }}
            </span>
            <svg
              class="h-4 w-4 flex-shrink-0 text-gray-400 transition-transform duration-150 dark:text-dark-400"
              :class="platformMenuOpen ? 'rotate-180' : ''"
              viewBox="0 0 24 24"
              fill="none"
              stroke="currentColor"
              stroke-width="2"
              aria-hidden="true"
            >
              <path stroke-linecap="round" stroke-linejoin="round" d="M19 9l-7 7-7-7" />
            </svg>
          </button>

          <div
            v-if="platformMenuOpen"
            class="group-platform-menu"
            role="listbox"
            :aria-label="t('keys.platformLabel')"
            data-tour="key-list-group-platform-menu"
            @click.stop
          >
            <button
              v-for="option in platformOptions"
              :key="String(option.value)"
              type="button"
              role="option"
              class="group-platform-option"
              :class="platformFilter === option.value ? 'group-platform-option--selected' : ''"
              :aria-selected="platformFilter === option.value"
              @click.stop="selectPlatform(option.value)"
            >
              <span class="min-w-0 flex-1 truncate text-left">{{ option.label }}</span>
              <svg
                v-if="platformFilter === option.value"
                class="h-4 w-4 flex-shrink-0 text-primary-500"
                viewBox="0 0 24 24"
                fill="none"
                stroke="currentColor"
                stroke-width="2"
                aria-hidden="true"
              >
                <path stroke-linecap="round" stroke-linejoin="round" d="M5 13l4 4L19 7" />
              </svg>
            </button>
          </div>
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
            @click.stop="platformMenuOpen = false"
            @focus="platformMenuOpen = false"
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
import GroupOptionItem from '@/components/common/GroupOptionItem.vue'
import {
  buildKeyFormPlatformOptions,
  filterAndSortKeyFormGroupOptions,
  type KeyFormPlatformFilter
} from '@/utils/keyFormGroupOptions'
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
const platformMenuOpen = ref(false)

const platformOptions = computed(() =>
  buildKeyFormPlatformOptions(props.options, {
    all: t('keys.allPlatforms'),
    platformLabel: (platform) => t(`admin.groups.platforms.${platform}`)
  })
)

const platformLabel = computed(() => {
  const current = platformOptions.value.find((option) => option.value === platformFilter.value)
  return current?.label ?? t('keys.allPlatforms')
})

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
  platformMenuOpen.value = false
}

const selectPlatform = (value: KeyFormPlatformFilter) => {
  platformFilter.value = value
  platformMenuOpen.value = false
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

/** 点击浮层内部但非平台下拉区域时，收起平台菜单 */
const handleInsideClick = (target: HTMLElement) => {
  if (!platformMenuOpen.value) return
  if (target.closest('[data-tour="key-list-group-platform-wrap"]')) return
  platformMenuOpen.value = false
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
  handleInsideClick,
  resetFilters
})
</script>

<style scoped>
/* 列表切换分组浮层：平台自定义下拉 + 搜索框统一视觉 */
.group-platform-trigger,
.group-selector-search-input {
  @apply w-full rounded-lg border border-gray-200 bg-gray-50 text-sm leading-5;
  @apply text-gray-900 outline-none transition-colors duration-150;
  @apply hover:border-gray-300 focus:border-primary-300 focus:ring-1 focus:ring-primary-300;
  @apply dark:border-dark-600 dark:bg-dark-700 dark:text-white;
  @apply dark:hover:border-dark-500 dark:focus:border-primary-600 dark:focus:ring-primary-600;
  min-height: 2.125rem;
}

.group-platform-trigger {
  @apply flex cursor-pointer items-center gap-2 px-2.5 py-1.5 text-left;
}

.group-platform-trigger--active {
  @apply border-primary-200 bg-primary-50/70 text-primary-800;
  @apply dark:border-primary-700/60 dark:bg-primary-900/25 dark:text-primary-100;
}

.group-platform-trigger--open {
  @apply border-primary-300 ring-1 ring-primary-300;
  @apply dark:border-primary-600 dark:ring-primary-600;
}

.group-platform-menu {
  @apply absolute left-0 right-0 top-[calc(100%+4px)] z-20;
  @apply max-h-56 overflow-y-auto rounded-xl border border-gray-200 bg-white py-1;
  @apply shadow-lg shadow-black/10;
  @apply dark:border-dark-700 dark:bg-dark-800 dark:shadow-black/30;
}

.group-platform-option {
  @apply flex w-full cursor-pointer items-center justify-between gap-2 px-3 py-2 text-sm;
  @apply text-gray-700 transition-colors duration-150;
  @apply hover:bg-gray-50 dark:text-gray-300 dark:hover:bg-dark-700;
}

.group-platform-option--selected {
  @apply bg-primary-50 text-primary-700;
  @apply dark:bg-primary-900/20 dark:text-primary-300;
}

.group-selector-search-input {
  @apply py-1.5 pl-8 pr-3 placeholder-gray-400 dark:placeholder-gray-500;
}
</style>

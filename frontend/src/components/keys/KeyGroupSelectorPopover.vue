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
      <!-- 厂商过滤 + 搜索：厂商卡片与创建密钥弹窗同一套实现，两处选择体验保持一致 -->
      <div class="space-y-2 rounded-t-xl border-b border-gray-100 p-2 dark:border-dark-700">
        <fieldset data-tour="key-list-group-provider-wrap">
          <!-- 浮层没有标题栏，用 sr-only legend 给单选组一个无障碍名称，不额外占视觉空间 -->
          <legend class="sr-only">{{ t('keys.providerLabel') }}</legend>
          <div class="grid grid-cols-4 gap-1.5" data-tour="key-list-group-provider">
            <label
              v-for="provider in providerOptions"
              :key="provider.value"
              class="relative min-w-0"
              :class="provider.count === 0 ? 'cursor-not-allowed' : 'cursor-pointer'"
            >
              <input
                type="radio"
                name="key-list-group-provider"
                :value="provider.value"
                :checked="selectedProvider === provider.value"
                :disabled="provider.count === 0"
                class="peer sr-only"
                @change="selectProvider(provider.value)"
              />
              <span
                class="flex h-full flex-col items-center gap-1.5 rounded-xl border border-gray-200 bg-white px-1.5 py-2 text-center transition-colors peer-checked:border-primary-500 peer-checked:bg-primary-50/60 peer-checked:ring-1 peer-checked:ring-primary-500 peer-focus-visible:outline peer-focus-visible:outline-2 peer-focus-visible:outline-offset-2 peer-focus-visible:outline-primary-500 peer-disabled:opacity-40 dark:border-dark-600 dark:bg-dark-800 dark:peer-checked:border-primary-500 dark:peer-checked:bg-primary-500/10"
                :class="provider.count > 0 && 'hover:border-primary-300 dark:hover:border-primary-700'"
              >
                <span class="flex h-7 items-center justify-center gap-1" aria-hidden="true">
                  <span
                    v-for="platform in KEY_GROUP_PROVIDER_ICONS[provider.value]"
                    :key="platform"
                    class="flex h-7 w-7 items-center justify-center rounded-lg"
                    :class="platformBadgeLightClass(platform)"
                  >
                    <PlatformIcon :platform="platform" size="lg" />
                  </span>
                </span>
                <span class="w-full truncate text-xs font-semibold text-gray-800 dark:text-gray-100">
                  {{ provider.label }}
                </span>
              </span>
              <span
                v-if="selectedProvider === provider.value"
                class="absolute right-1 top-1 flex h-3.5 w-3.5 items-center justify-center rounded-full bg-primary-500 text-white"
                aria-hidden="true"
              >
                <Icon name="check" size="xs" :stroke-width="3" />
              </span>
            </label>
          </div>
        </fieldset>

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
import GroupOptionItem from '@/components/common/GroupOptionItem.vue'
import PlatformIcon from '@/components/common/PlatformIcon.vue'
import Icon from '@/components/icons/Icon.vue'
import { platformBadgeLightClass } from '@/utils/platformColors'
import {
  KEY_GROUP_PROVIDERS,
  KEY_GROUP_PROVIDER_ICONS,
  getKeyGroupProvider,
  type KeyGroupProvider
} from '@/utils/keyGroupProviders'
import {
  resolveGroupBusinessPlatform,
  sortGroupsByRateAsc
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
const selectedProvider = ref<KeyGroupProvider>('anthropic')

/**
 * 分类按「解析后的业务平台」，而不是分组的 platform 原始字段：
 * composite 分组、或被监控层覆写过上游的分组，其真实上游与 platform 字段并不一致，
 * 用原始字段会把它们一律错归到「其他」，用户按厂商就找不到它们。
 */
const providerOf = (option: KeyGroupSelectorOption): KeyGroupProvider =>
  getKeyGroupProvider(resolveGroupBusinessPlatform(option))

const providerOptions = computed(() =>
  KEY_GROUP_PROVIDERS.map((value) => ({
    value,
    label: t(`keys.providers.${value}`),
    count: props.options.filter((option) => providerOf(option) === value).length
  }))
)

/** 厂商卡片是单选，没有「全部」项，需要一个默认落点：取第一个有分组的厂商 */
const firstProviderWithGroups = (): KeyGroupProvider =>
  providerOptions.value.find((provider) => provider.count > 0)?.value ?? 'anthropic'

const selectProvider = (provider: KeyGroupProvider) => {
  selectedProvider.value = provider
}

const filteredOptions = computed(() => {
  // 按倍率升序，与创建密钥弹窗的分组下拉同一套排序，避免两处顺序不一致。
  const providerFiltered = sortGroupsByRateAsc(
    props.options.filter((option) => providerOf(option) === selectedProvider.value)
  )
  const query = searchQuery.value.trim().toLowerCase()
  if (!query) return providerFiltered
  return providerFiltered.filter((option) => {
    return (
      option.label.toLowerCase().includes(query) ||
      (option.description && option.description.toLowerCase().includes(query))
    )
  })
})

const resetFilters = () => {
  searchQuery.value = ''
  selectProvider(firstProviderWithGroups())
}

// 分组数据可能晚于组件挂载到达。当前厂商一个分组都没有时，落到第一个有分组的厂商，
// 否则用户打开浮层只会看到空列表。
watch(
  providerOptions,
  (providers) => {
    const current = providers.find((provider) => provider.value === selectedProvider.value)
    if (!current || current.count === 0) {
      selectProvider(providers.find((provider) => provider.count > 0)?.value ?? 'anthropic')
    }
  },
  { immediate: true }
)

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
/* 列表切换分组浮层：搜索框视觉与创建密钥弹窗的搜索框保持一致 */
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

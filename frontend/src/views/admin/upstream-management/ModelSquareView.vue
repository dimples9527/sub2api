<template>
  <AppLayout class="model-square-root">
    <TablePageLayout>
      <template #filters>
        <div class="flex flex-wrap items-center gap-3">
          <div class="flex min-w-0 flex-1 flex-wrap items-center gap-3">
            <div class="summary-pill">
              <span>{{ t('admin.modelSquare.modelCount') }}</span>
              <strong>{{ models.length }}</strong>
            </div>
            <div class="summary-pill">
              <span>{{ t('admin.modelSquare.availableCount') }}</span>
              <strong>{{ availableCount }}</strong>
            </div>
            <div class="summary-pill">
              <span>{{ t('admin.modelSquare.groupCount') }}</span>
              <strong>{{ groups.length }}</strong>
            </div>
          </div>

          <div class="flex flex-wrap items-center justify-end gap-2">
            <button
              type="button"
              class="btn btn-secondary"
              :disabled="loading"
              :title="t('common.refresh')"
              @click="reload"
            >
              <Icon name="refresh" size="md" :class="loading ? 'animate-spin' : ''" />
            </button>
          </div>
        </div>

        <div class="mt-3 flex flex-wrap items-center gap-3">
          <div class="relative w-full sm:w-72">
            <Icon
              name="search"
              size="md"
              class="absolute left-3 top-1/2 -translate-y-1/2 text-gray-400 dark:text-gray-500"
            />
            <input
              v-model="searchQuery"
              type="search"
              class="input pl-10"
              :placeholder="t('admin.modelSquare.searchPlaceholder')"
            />
          </div>

          <Select
            v-model="groupFilter"
            :options="groupFilterOptions"
            class="w-full sm:w-44"
            :aria-label="t('admin.modelSquare.allGroups')"
          />

          <Select
            v-model="providerFilter"
            :options="providerFilterOptions"
            class="w-full sm:w-44"
            :aria-label="t('admin.modelSquare.allProviders')"
          />

          <div class="ml-auto inline-grid grid-cols-2 gap-1 rounded-lg border border-gray-200 bg-gray-100 p-1 dark:border-dark-700 dark:bg-dark-800">
            <button
              type="button"
              class="view-toggle-btn"
              :class="{ active: viewMode === 'grid' }"
              :title="t('admin.modelSquare.gridView')"
              @click="viewMode = 'grid'"
            >
              <Icon name="grid" size="sm" />
            </button>
            <button
              type="button"
              class="view-toggle-btn"
              :class="{ active: viewMode === 'list' }"
              :title="t('admin.modelSquare.listView')"
              @click="viewMode = 'list'"
            >
              <Icon name="menu" size="sm" />
            </button>
          </div>
        </div>
      </template>

      <template #table>
        <div v-if="loading" class="grid min-h-64 place-items-center text-sm text-gray-500 dark:text-gray-400">
          <div class="flex items-center gap-2">
            <Icon name="refresh" size="sm" class="animate-spin" />
            <span>{{ t('admin.modelSquare.loading') }}</span>
          </div>
        </div>

        <div v-else-if="loadError" class="rounded-lg border border-red-200 bg-red-50 p-4 text-sm text-red-700 dark:border-red-900/60 dark:bg-red-900/20 dark:text-red-300">
          {{ loadError }}
        </div>

        <EmptyState
          v-else-if="filteredModels.length === 0"
          :title="t('admin.modelSquare.emptyTitle')"
          :description="emptyDescription"
          :action-text="t('common.refresh')"
          @action="reload"
        />

        <div v-else-if="viewMode === 'grid'" class="model-square-board">
          <section
            v-for="section in providerSections"
            :key="section.provider"
            class="provider-section"
            :style="providerAccent(section.provider)"
          >
            <div class="provider-section-header">
              <div class="min-w-0">
                <div class="provider-title">
                  <span class="provider-dot"></span>
                  <span class="truncate">{{ providerLabel(section.provider) }}</span>
                </div>
                <div class="provider-meta">
                  {{ t('admin.modelSquare.providerSummary', { count: section.models.length, rate: formatRate(section.lowestRate) }) }}
                </div>
              </div>
              <span class="provider-count">{{ section.models.length }}</span>
            </div>

            <div class="model-card-grid">
              <article
                v-for="(model, index) in section.models"
                :key="modelKey(model, index)"
                data-test="model-card"
                class="model-card"
                role="button"
                tabindex="0"
                :title="modelCardTitle(model)"
                :style="providerAccent(model.provider)"
                @click="copyModelId(model)"
                @keydown.enter.prevent="copyModelId(model)"
              >
                <div class="model-card-top">
                  <span class="model-provider">
                    <span class="model-provider-dot"></span>
                    <span class="truncate">{{ providerLabel(model.provider) }}</span>
                  </span>
                </div>

                <div class="model-title-row">
                  <h3 class="model-title">
                    {{ modelDisplayName(model) }}
                  </h3>
                  <button
                    type="button"
                    class="copy-button"
                    :title="t('admin.modelSquare.copyTitle')"
                    @click.stop="copyModelId(model)"
                  >
                    <Icon :name="copiedModelId === model.id ? 'check' : 'copy'" size="sm" />
                  </button>
                </div>

                <div class="price-grid">
                  <div
                    v-for="slot in modelPriceSlots(model)"
                    :key="slot.key"
                    :class="['price-box', slot.toneClass]"
                  >
                    <span>{{ slot.label }}</span>
                    <strong>{{ formatPriceOrZero(slot.value) }}</strong>
                    <s v-if="slot.originalValue != null" class="price-original">{{ formatPrice(slot.originalValue) }}</s>
                    <small v-if="slot.unit">{{ slot.unit }}</small>
                  </div>
                </div>

                <div class="model-card-footer">
                  <span class="model-rate-chip">{{ formatRate(modelEffectiveRate(model)) }}</span>
                  <button
                    type="button"
                    v-if="modelGroups(model).length > 0"
                    class="primary-group-chip"
                    @click.stop="openGroupDialog(model)"
                  >
                    <span class="truncate">{{ primaryGroup(model)?.name }}</span>
                    <span v-if="modelGroupOverflowCount(model) > 0" class="group-overflow">+{{ modelGroupOverflowCount(model) }}</span>
                  </button>
                  <button
                    type="button"
                    class="model-detail-button"
                    title="详情"
                    @click.stop="openModelDetails(model)"
                  >
                    <Icon name="eye" size="xs" />
                    <span>详情</span>
                  </button>
                </div>
              </article>
            </div>
          </section>
        </div>

        <div v-else class="overflow-x-auto">
          <table class="w-full min-w-[1100px] divide-y divide-gray-100 text-sm dark:divide-dark-700">
            <thead class="bg-gray-50 dark:bg-dark-800">
              <tr>
                <th class="px-4 py-3 text-left font-medium">{{ t('admin.modelSquare.columns.provider') }}</th>
                <th class="px-4 py-3 text-left font-medium">{{ t('admin.modelSquare.columns.modelId') }}</th>
                <th class="px-4 py-3 text-left font-medium">{{ t('admin.modelSquare.columns.input') }}</th>
                <th class="px-4 py-3 text-left font-medium">{{ t('admin.modelSquare.columns.output') }}</th>
                <th class="px-4 py-3 text-left font-medium">{{ t('admin.modelSquare.columns.cacheRead') }}</th>
                <th class="px-4 py-3 text-left font-medium">{{ t('admin.modelSquare.columns.cacheWrite') }}</th>
                <th class="px-4 py-3 text-left font-medium">{{ t('admin.modelSquare.columns.groups') }}</th>
                <th class="px-4 py-3 text-right font-medium">{{ t('admin.modelSquare.columns.actions') }}</th>
              </tr>
            </thead>
            <tbody class="divide-y divide-gray-100 dark:divide-dark-700">
              <tr
                v-for="(model, index) in filteredModels"
                :key="modelKey(model, index)"
                data-test="model-row"
                class="cursor-pointer transition hover:bg-gray-50 dark:hover:bg-dark-700/60"
                :title="modelCardTitle(model)"
                @click="copyModelId(model)"
              >
                <td class="whitespace-nowrap px-4 py-3">
                  <span class="model-provider">
                    <span class="model-provider-dot" :style="providerAccent(model.provider)"></span>
                    <span class="truncate">{{ providerLabel(model.provider) }}</span>
                  </span>
                </td>
                <td class="max-w-72 px-4 py-3 font-medium text-gray-950 dark:text-white">
                  <span class="break-words">{{ modelDisplayName(model) }}</span>
                </td>
                <td
                  v-for="slot in modelPriceSlots(model)"
                  :key="slot.key"
                  class="whitespace-nowrap px-4 py-3"
                >
                  <span :class="['price-cell', priceCellClass(slot.toneClass)]">{{ formatPriceOrZero(slot.value) }}</span>
                </td>
                <td class="px-4 py-3">
                  <div class="flex min-w-56 flex-wrap gap-1.5">
                    <button
                      v-for="group in modelGroups(model)"
                      :key="String(group.id)"
                      type="button"
                      class="group-chip"
                      @click.stop="openGroupDialog(model)"
                    >
                      {{ group.name }}
                      <b>{{ formatRate(group.rate_multiplier) }}</b>
                    </button>
                    <span v-if="modelGroups(model).length === 0" class="text-xs text-gray-400 dark:text-gray-500">—</span>
                  </div>
                </td>
                <td class="whitespace-nowrap px-4 py-3 text-right">
                  <button
                    type="button"
                    class="model-detail-button"
                    title="详情"
                    @click.stop="openModelDetails(model)"
                  >
                    <Icon name="eye" size="xs" />
                    <span>详情</span>
                  </button>
                </td>
              </tr>
            </tbody>
          </table>
        </div>
      </template>
    </TablePageLayout>

    <BaseDialog
      :show="Boolean(detailModel)"
      :title="detailDialogTitle"
      width="extra-wide"
      @close="closeModelDetails"
    >
      <div v-if="detailModel" class="model-square-root space-y-5">
        <div class="flex flex-wrap items-start justify-between gap-3 border-b border-gray-100 pb-4 dark:border-dark-700">
          <div class="min-w-0">
            <div class="break-words text-lg font-bold text-gray-950 dark:text-white">{{ modelDisplayName(detailModel) }}</div>
            <code class="mt-1 block break-all text-xs text-gray-400">{{ detailModel.id }}</code>
          </div>
          <div class="flex shrink-0 flex-wrap items-center gap-2 text-xs">
            <span class="model-rate-chip">{{ formatRate(detailRate) }}</span>
          </div>
        </div>

        <div>
          <div class="mb-2 flex items-center justify-between gap-3">
            <span class="text-sm font-semibold text-gray-950 dark:text-white">分组倍率</span>
            <span class="text-xs text-gray-500 dark:text-gray-400">平台: {{ providerLabel(detailModel.provider) }}</span>
          </div>
          <div v-if="detailGroups.length > 0" class="grid gap-2 sm:grid-cols-2 lg:grid-cols-3">
            <button
              v-for="group in detailGroups"
              :key="String(group.id)"
              type="button"
              data-test="detail-group-option"
              :class="['detail-group-option', { active: String(group.id) === detailGroupId }]"
              @click="selectDetailGroup(group)"
            >
              <span class="min-w-0 text-left">
                <span class="block truncate font-semibold">{{ group.name }}</span>
                <code class="mt-1 block text-[11px] text-gray-400">#{{ group.id }}</code>
              </span>
              <span class="shrink-0 font-mono text-xs font-bold text-orange-600 dark:text-orange-300">{{ formatRate(group.rate_multiplier) }}</span>
            </button>
          </div>
          <div v-else class="rounded-lg border border-dashed border-gray-200 px-3 py-4 text-sm text-gray-500 dark:border-dark-700 dark:text-gray-400">
            暂无可切换的分组
          </div>
        </div>

        <div class="detail-price-section">
          <div class="mb-3 flex items-center justify-between gap-3">
            <span class="text-sm font-semibold text-gray-950 dark:text-white">当前价格</span>
            <span v-if="selectedDetailGroup" class="text-xs text-gray-500 dark:text-gray-400">
              {{ selectedDetailGroup.name }} · {{ formatRate(detailRate) }}
            </span>
          </div>
          <div class="price-grid !mt-0">
            <div
              v-for="slot in detailPriceSlots"
              :key="slot.key"
              :class="['price-box', slot.toneClass]"
            >
              <span>{{ slot.label }}</span>
              <strong>{{ formatPriceOrZero(slot.value) }}</strong>
              <s v-if="slot.originalValue != null" class="price-original">{{ formatPrice(slot.originalValue) }}</s>
              <small v-if="slot.unit">{{ slot.unit }}</small>
            </div>
          </div>
        </div>
      </div>

      <template #footer>
        <button type="button" class="btn btn-secondary" @click="closeModelDetails">
          {{ t('common.close') }}
        </button>
      </template>
    </BaseDialog>

    <BaseDialog
      :show="Boolean(groupDialogModel)"
      :title="groupDialogTitle"
      width="wide"
      @close="closeGroupDialog"
    >
      <div class="model-square-root max-h-[56vh] space-y-2 overflow-y-auto">
        <div
          v-for="group in groupDialogGroups"
          :key="String(group.id)"
          class="flex items-center justify-between gap-3 rounded-lg border border-gray-100 bg-gray-50 px-3 py-3 dark:border-dark-700 dark:bg-dark-700/50"
        >
          <div class="min-w-0">
            <div class="break-words text-sm font-medium text-gray-950 dark:text-white">{{ group.name }}</div>
            <code class="text-xs text-gray-400">#{{ group.id }}</code>
          </div>
          <div class="shrink-0 text-xs text-gray-500 dark:text-gray-400">
            {{ t('admin.modelSquare.rate') }}
            <span class="ml-2 rounded bg-amber-100 px-2 py-1 font-semibold text-orange-600 dark:bg-amber-900/40 dark:text-amber-300">
              {{ formatRate(group.rate_multiplier) }}
            </span>
          </div>
        </div>
      </div>

      <template #footer>
        <button type="button" class="btn btn-secondary" @click="closeGroupDialog">
          {{ t('common.close') }}
        </button>
      </template>
    </BaseDialog>
  </AppLayout>
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { modelSquareAPI } from '@/api/modelSquare'
import type { AdminModelSquareResult, ModelSquareGroup, ModelSquareModel } from '@/api/admin'
import { useAppStore } from '@/stores/app'
import { extractApiErrorMessage } from '@/utils/apiError'
import { useRouteQueryFilters } from '@/composables/useRouteQueryFilters'
import AppLayout from '@/components/layout/AppLayout.vue'
import TablePageLayout from '@/components/layout/TablePageLayout.vue'
import BaseDialog from '@/components/common/BaseDialog.vue'
import EmptyState from '@/components/common/EmptyState.vue'
import Icon from '@/components/icons/Icon.vue'
import Select, { type SelectOption } from '@/components/common/Select.vue'
import { platformAccentColor } from '@/utils/platformColors'

type PriceField =
  | 'input_price'
  | 'output_price'
  | 'cache_read_price'
  | 'cache_write_price'
  | 'cache_write_1h_price'
  | 'input_price_priority'
  | 'output_price_priority'
  | 'cache_write_price_priority'
  | 'cache_read_price_priority'
  | 'image_input_price'
  | 'image_output_price'
  | 'per_request_price'
type PriceDescriptor = {
  key: PriceField
  label: string
  unit: string
  toneClass: string
}
type ModelPriceSlot = PriceDescriptor & {
  value?: number
  originalValue?: number
}
type ModelSquareProviderSection = {
  provider: string
  models: ModelSquareModel[]
  lowestRate: number
}

const { t } = useI18n()
const appStore = useAppStore()

const result = ref<AdminModelSquareResult | null>(null)
const loading = ref(false)
const loadError = ref('')
const searchQuery = ref('')
const providerFilter = ref('')
const groupFilter = ref('')
useRouteQueryFilters([
  { queryKey: 'provider', state: providerFilter },
  { queryKey: 'group', state: groupFilter },
  { queryKey: 'model', state: searchQuery },
])
const viewMode = ref<'grid' | 'list'>('grid')
const groupDialogModel = ref<ModelSquareModel | null>(null)
const detailModel = ref<ModelSquareModel | null>(null)
const detailGroupId = ref('')
const copiedModelId = ref('')
let copiedTimer: ReturnType<typeof setTimeout> | undefined

const defaultPriceDescriptors: PriceDescriptor[] = [
  { key: 'input_price', label: '\u8f93\u5165', unit: '$/\u767e\u4e07 tokens', toneClass: 'price-box-teal' },
  { key: 'output_price', label: '\u8f93\u51fa', unit: '$/\u767e\u4e07 tokens', toneClass: 'price-box-orange' },
  { key: 'cache_read_price', label: '\u7f13\u5b58\u8bfb\u53d6', unit: '', toneClass: 'price-box-blue' },
  { key: 'cache_write_price', label: '\u7f13\u5b58\u5199\u5165', unit: '', toneClass: 'price-box-violet' },
]

const priceDescriptors: PriceDescriptor[] = [
  ...defaultPriceDescriptors,
  { key: 'cache_write_1h_price', label: '\u7f13\u5b58\u5199\u5165 1h', unit: '$/\u767e\u4e07 tokens', toneClass: 'price-box-violet' },
  { key: 'input_price_priority', label: '\u4f18\u5148\u7ea7\u8f93\u5165', unit: '$/\u767e\u4e07 tokens', toneClass: 'price-box-neutral' },
  { key: 'output_price_priority', label: '\u4f18\u5148\u7ea7\u8f93\u51fa', unit: '$/\u767e\u4e07 tokens', toneClass: 'price-box-neutral' },
  { key: 'cache_write_price_priority', label: '\u4f18\u5148\u7ea7\u7f13\u5b58\u5199\u5165', unit: '$/\u767e\u4e07 tokens', toneClass: 'price-box-violet' },
  { key: 'cache_read_price_priority', label: '\u4f18\u5148\u7ea7\u7f13\u5b58\u8bfb\u53d6', unit: '$/\u767e\u4e07 tokens', toneClass: 'price-box-blue' },
  { key: 'image_input_price', label: '\u56fe\u50cf\u8f93\u5165', unit: '$/\u767e\u4e07 tokens', toneClass: 'price-box-neutral' },
  { key: 'image_output_price', label: '\u56fe\u50cf\u8f93\u51fa', unit: '$/\u767e\u4e07 tokens', toneClass: 'price-box-neutral' },
  { key: 'per_request_price', label: '\u6309\u8bf7\u6c42', unit: '$/\u6b21', toneClass: 'price-box-neutral' },
]
const emptyDescription = '\u5c1a\u672a\u914d\u7f6e\u6a21\u578b\u5e7f\u573a\u5e73\u53f0\u6216\u6a21\u578b\uff0c\u8bf7\u5148\u5728\u201c\u6a21\u578b\u5e7f\u573a\u914d\u7f6e\u201d\u4e2d\u7ef4\u62a4\u5c55\u793a\u76ee\u5f55\u3002'

const payload = computed(() => result.value?.payload?.data || result.value?.payload || {})
const models = computed<ModelSquareModel[]>(() => Array.isArray(payload.value.models) ? payload.value.models : [])
const groups = computed<ModelSquareGroup[]>(() => Array.isArray(payload.value.groups) ? payload.value.groups : [])
const groupById = computed(() => new Map(groups.value.map(group => [String(group.id), group])))
const providers = computed(() => unique(models.value.map(model => model.provider).filter(Boolean) as string[]))
// 分组筛选下拉选项（含“全部”占位项）
const groupFilterOptions = computed<SelectOption[]>(() => [
  { value: '', label: t('admin.modelSquare.allGroups') },
  ...groups.value.map(group => ({ value: String(group.id), label: group.name })),
])
// 平台筛选下拉选项（含“全部”占位项）
const providerFilterOptions = computed<SelectOption[]>(() => [
  { value: '', label: t('admin.modelSquare.allProviders') },
  ...providers.value.map(item => ({ value: item, label: providerLabel(item) })),
])
const availableCount = computed(() => models.value.filter(isAvailable).length)
const groupDialogGroups = computed(() => groupDialogModel.value ? modelGroups(groupDialogModel.value) : [])
const groupDialogTitle = computed(() => {
  const id = groupDialogModel.value?.id || t('admin.modelSquare.unnamedModel')
  return t('admin.modelSquare.groupDialogTitle', { id })
})
const detailGroups = computed(() => detailModel.value ? modelDetailGroups(detailModel.value) : [])
const selectedDetailGroup = computed(() => detailGroups.value.find(group => String(group.id) === detailGroupId.value))
const detailRate = computed(() => {
  const rate = groupRate(selectedDetailGroup.value)
  if (Number.isFinite(rate)) return rate
  return detailModel.value ? modelEffectiveRate(detailModel.value) : 1
})
const detailPriceSlots = computed(() => {
  if (!detailModel.value) return []
  return modelPriceSlots(detailModel.value, detailRate.value)
})
const detailDialogTitle = computed(() => {
  const id = detailModel.value?.id || t('admin.modelSquare.unnamedModel')
  return `${id} 详情`
})

const filteredModels = computed(() => {
  const keyword = searchQuery.value.trim().toLowerCase()
  return models.value.filter(model => {
    if (keyword && !modelSearchText(model).includes(keyword)) return false
    if (providerFilter.value && model.provider !== providerFilter.value) return false
    if (groupFilter.value && !(model.group_ids || []).some(id => String(id) === groupFilter.value)) return false
    return true
  })
})
const providerSections = computed<ModelSquareProviderSection[]>(() => {
  const sections = new Map<string, ModelSquareModel[]>()
  for (const model of filteredModels.value) {
    const provider = model.provider || ''
    const list = sections.get(provider) || []
    list.push(model)
    sections.set(provider, list)
  }

  return Array.from(sections.entries())
    .map(([provider, sectionModels]) => {
      const sortedModels = [...sectionModels].sort((a, b) => {
        if (isAvailable(a) !== isAvailable(b)) return isAvailable(a) ? -1 : 1
        const rateDiff = primaryGroupRate(a) - primaryGroupRate(b)
        if (rateDiff !== 0) return rateDiff
        return (a.id || '').localeCompare(b.id || '')
      })
      return {
        provider,
        models: sortedModels,
        lowestRate: Math.min(...sortedModels.map(primaryGroupRate))
      }
    })
    .sort((a, b) => {
      const rateDiff = a.lowestRate - b.lowestRate
      if (rateDiff !== 0) return rateDiff
      return a.provider.localeCompare(b.provider)
    })
})

async function reload() {
  loading.value = true
  loadError.value = ''
  try {
    result.value = await modelSquareAPI.get()
  } catch (err) {
    const message = extractApiErrorMessage(err, t('admin.modelSquare.loadFailed'))
    loadError.value = message
    result.value = null
    appStore.showError(message)
  } finally {
    loading.value = false
  }
}

function modelGroups(model: ModelSquareModel): ModelSquareGroup[] {
  return (model.group_ids || [])
    .map(id => groupById.value.get(String(id)))
    .filter(Boolean)
    .sort((a, b) => groupRate(a) - groupRate(b)) as ModelSquareGroup[]
}

function modelDetailGroups(model: ModelSquareModel): ModelSquareGroup[] {
  const directGroupIds = new Set((model.group_ids || []).map(id => String(id)))
  const candidates = groups.value.filter(group => {
    if (directGroupIds.has(String(group.id))) return true
    const platform = group.platform?.trim().toLowerCase()
    const modelPlatform = model.platform?.trim().toLowerCase()
    return !platform || platform === 'composite' || (Boolean(modelPlatform) && platform === modelPlatform)
  })
  return candidates.sort((a, b) => {
    const rateDiff = groupRate(a) - groupRate(b)
    if (rateDiff !== 0) return rateDiff
    return String(a.name).localeCompare(String(b.name))
  })
}

function primaryGroupRate(model: ModelSquareModel) {
  return modelEffectiveRate(model)
}

function primaryGroup(model: ModelSquareModel) {
  return modelGroups(model)[0]
}

function modelGroupOverflowCount(model: ModelSquareModel) {
  return Math.max(0, modelGroups(model).length - 1)
}

function groupRate(group?: ModelSquareGroup) {
  const rate = Number(group?.rate_multiplier)
  return Number.isFinite(rate) ? rate : Number.POSITIVE_INFINITY
}

function modelEffectiveRate(model: ModelSquareModel) {
  const rate = Number(model.rate_multiplier)
  if (Number.isFinite(rate)) return rate

  const groupRateValue = groupRate(primaryGroup(model))
  return Number.isFinite(groupRateValue) ? groupRateValue : 1
}

function modelPriceValue(model: ModelSquareModel, field: PriceField, multiplier?: number) {
  const value = model[field]
  if (value == null || value === '') return undefined

  const price = Number(value)
  if (!Number.isFinite(price)) return undefined
  if (multiplier == null) return price

  const baseRate = modelEffectiveRate(model)
  if (!Number.isFinite(baseRate) || baseRate === 0) return price === 0 ? 0 : undefined
  return (price / baseRate) * multiplier
}

function modelDisplayName(model: ModelSquareModel) {
  const displayName = model.display_name?.trim()
  if (displayName && model.id && displayName !== model.id) return `${displayName} (${model.id})`
  return displayName || model.id || t('admin.modelSquare.unnamedModel')
}

function modelPriceSlots(model: ModelSquareModel, multiplier?: number): ModelPriceSlot[] {
  // 固定展示输入、输出、缓存读取、缓存写入四个价格位，不被优先级/图片/按请求等额外价格顶替。
  return defaultPriceDescriptors.map(descriptor => {
    const value = modelPriceValue(model, descriptor.key, multiplier)
    const original = value == null ? undefined : modelPriceValue(model, descriptor.key, 1)
    const originalValue = original != null && value != null && value < original - 1e-9 ? original : undefined
    return {
      ...descriptor,
      value,
      originalValue,
      toneClass: value == null ? 'price-box-unset' : descriptor.toneClass,
    }
  })
}

function modelCardTitle(model: ModelSquareModel) {
  const items = modelConfiguredPriceLines(model)
  return [t('admin.modelSquare.copyTitle'), model.id, ...items].filter(Boolean).join('\n')
}

function modelConfiguredPriceLines(model: ModelSquareModel) {
  return priceDescriptors.flatMap(descriptor => {
    const value = modelPriceValue(model, descriptor.key)
    const unit = descriptor.unit ? ` ${descriptor.unit}` : ''
    return value == null ? [] : [`${descriptor.label}: ${formatPrice(value)}${unit}`]
  })
}

function formatPriceOrZero(value?: number | string) {
  return value == null || value === '' ? '$0' : formatPrice(value)
}

function priceCellClass(toneClass: string) {
  return toneClass === 'price-box-unset' ? 'price-cell-unset' : `price-cell-${toneClass.replace('price-box-', '')}`
}

function isAvailable(model: ModelSquareModel) {
  return model.available !== false
}

function modelSearchText(model: ModelSquareModel) {
  return [model.id, model.display_name, model.provider, model.platform, model.mode]
    .filter(Boolean)
    .join(' ')
    .toLowerCase()
}

function modelKey(model: ModelSquareModel, index: number) {
  return `${model.platform || model.provider || 'unknown'}:${model.id || index}`
}

function providerLabel(value?: string) {
  return value || t('admin.modelSquare.unknownProvider')
}

// 平台强调色：已知平台复用全站 platformColors，未知平台按名称哈希取本地深色，保证不同平台有辨识度。
const providerAccentFallbackColors = ['#0d9488', '#2563eb', '#7c3aed', '#d97706', '#dc2626', '#0891b2', '#16a34a', '#db2777']
const providerAccentKnownPlatforms = new Set(['anthropic', 'openai', 'antigravity', 'gemini', 'grok', 'composite'])

function providerAccent(provider?: string) {
  const key = (provider || '').trim().toLowerCase()
  if (providerAccentKnownPlatforms.has(key)) {
    return { '--ms-provider': platformAccentColor(key) }
  }

  let hash = 0
  for (let i = 0; i < key.length; i++) hash = (hash * 31 + key.charCodeAt(i)) >>> 0
  return { '--ms-provider': providerAccentFallbackColors[hash % providerAccentFallbackColors.length] }
}

function formatRate(value?: number) {
  const n = Number(value)
  if (!Number.isFinite(n)) return '-'
  return `${n.toFixed(3).replace(/0+$/, '').replace(/\.$/, '')}x`
}

function formatPrice(value?: number | string) {
  if (value == null || value === '') return '-'

  const n = Number(value)
  if (!Number.isFinite(n)) return '-'
  return '$' + n.toFixed(n >= 10 ? 2 : 3).replace(/0+$/, '').replace(/\.$/, '')
}

function unique(values: string[]) {
  return Array.from(new Set(values)).sort((a, b) => a.localeCompare(b))
}

function openGroupDialog(model: ModelSquareModel) {
  groupDialogModel.value = model
}

function closeGroupDialog() {
  groupDialogModel.value = null
}

function openModelDetails(model: ModelSquareModel) {
  detailModel.value = model
  const firstGroup = modelDetailGroups(model)[0]
  detailGroupId.value = firstGroup ? String(firstGroup.id) : ''
}

function closeModelDetails() {
  detailModel.value = null
  detailGroupId.value = ''
}

function selectDetailGroup(group: ModelSquareGroup) {
  detailGroupId.value = String(group.id)
}

async function copyModelId(model: ModelSquareModel) {
  if (!model.id) return
  try {
    if (navigator.clipboard?.writeText) {
      await navigator.clipboard.writeText(model.id)
    } else {
      fallbackCopy(model.id)
    }
    appStore.showSuccess(t('admin.modelSquare.copied'))
  } catch {
    fallbackCopy(model.id)
    appStore.showSuccess(t('admin.modelSquare.copied'))
  }

  copiedModelId.value = model.id
  if (copiedTimer) clearTimeout(copiedTimer)
  copiedTimer = setTimeout(() => {
    copiedModelId.value = ''
  }, 1500)
}

function fallbackCopy(value: string) {
  const textarea = document.createElement('textarea')
  textarea.value = value
  textarea.style.position = 'fixed'
  textarea.style.opacity = '0'
  document.body.appendChild(textarea)
  textarea.select()
  document.execCommand('copy')
  textarea.remove()
}

onMounted(reload)
</script>

<style scoped>
.summary-pill {
  @apply flex h-11 items-center gap-3 rounded-lg border px-3 text-sm;
  border-color: var(--ms-line);
  color: var(--ms-muted);
}

.summary-pill strong {
  @apply font-mono text-base;
  color: var(--ms-text);
}

.view-toggle-btn {
  @apply grid h-8 w-8 place-items-center rounded-md transition-colors;
  color: var(--ms-muted);
}

.view-toggle-btn:hover {
  background: var(--ms-panel-2);
  color: var(--ms-text);
}

.view-toggle-btn.active {
  background: color-mix(in srgb, var(--ms-brand) 10%, var(--ms-panel));
  color: var(--ms-brand-strong);
  box-shadow: inset 0 0 0 1px color-mix(in srgb, var(--ms-brand) 45%, transparent);
}

.model-square-board {
  @apply h-full space-y-5 overflow-y-auto p-4;
}

.provider-section {
  @apply relative space-y-3 pl-3;
}

.provider-section::before {
  content: '';
  position: absolute;
  left: 0;
  top: 0.4rem;
  bottom: 0.4rem;
  width: 3px;
  border-radius: 999px;
  background: var(--ms-provider, var(--ms-brand));
  opacity: 0.9;
}

.provider-section-header {
  @apply flex items-center justify-between gap-3 px-1;
}

.provider-title {
  @apply flex min-w-0 items-center gap-2 text-sm font-semibold;
  color: var(--ms-text);
}

.provider-dot,
.model-provider-dot {
  @apply h-2 w-2 shrink-0 rounded-full;
  background: var(--ms-provider, var(--ms-brand));
  box-shadow: 0 0 0 3px color-mix(in srgb, var(--ms-provider, var(--ms-brand)) 16%, transparent);
}

.provider-meta {
  @apply mt-0.5 text-xs;
  color: var(--ms-muted);
}

.provider-count {
  @apply grid h-7 min-w-7 place-items-center rounded-md px-2 font-mono text-xs font-semibold;
  background: var(--ms-panel-3);
  color: var(--ms-muted);
}

.model-card-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(320px, 1fr));
  @apply gap-4;
}

.model-card {
  @apply relative flex min-h-[16.5rem] cursor-pointer flex-col rounded-lg border p-5 transition duration-200;
  border-color: var(--ms-line);
  background: var(--ms-panel);
  box-shadow: var(--ms-shadow);
}

.model-card:hover,
.model-card:focus-visible {
  @apply -translate-y-0.5 outline-none;
  border-color: var(--ms-brand);
  box-shadow: 0 16px 38px rgba(15, 23, 42, 0.12), 0 0 0 1px color-mix(in srgb, var(--ms-brand) 55%, transparent), 0 0 28px var(--ms-glow);
}

.model-card-top {
  @apply flex items-start justify-between gap-3;
}

.model-provider {
  @apply inline-flex min-w-0 items-center gap-2 text-xs font-semibold;
  color: var(--ms-muted);
}

.model-title-row {
  @apply mt-3 flex min-h-[2.5rem] items-start gap-2;
}

.model-title {
  @apply min-w-0 flex-1 text-base font-bold leading-snug transition-colors;
  color: var(--ms-text);
  overflow-wrap: anywhere;
}

.model-card:hover .model-title,
.model-card:focus-visible .model-title {
  color: var(--ms-brand-strong);
}

.copy-button {
  @apply grid h-8 w-8 shrink-0 place-items-center rounded-md opacity-0 transition focus:opacity-100;
  color: var(--ms-dim);
}

.copy-button:hover {
  background: var(--ms-panel-3);
  color: var(--ms-brand-strong);
}

.model-card:hover .copy-button,
.model-card:focus-within .copy-button {
  @apply opacity-100;
}

.price-grid {
  @apply mt-4 grid grid-cols-2 gap-3;
}

.price-box {
  @apply min-h-[4.6rem] rounded-lg border p-3;
  border-color: var(--ms-line);
  background: var(--ms-panel-2);
}

.price-box-neutral {
  border-color: var(--ms-line);
  background: var(--ms-panel-2);
}

.price-box-teal {
  border-color: color-mix(in srgb, var(--ms-brand) 40%, var(--ms-line));
  background: color-mix(in srgb, var(--ms-brand) 12%, var(--ms-panel));
}

.price-box-orange {
  border-color: color-mix(in srgb, var(--ms-orange) 40%, var(--ms-line));
  background: color-mix(in srgb, var(--ms-orange) 12%, var(--ms-panel));
}

.price-box-blue {
  border-color: color-mix(in srgb, var(--ms-blue) 40%, var(--ms-line));
  background: color-mix(in srgb, var(--ms-blue) 12%, var(--ms-panel));
}

.price-box-violet {
  border-color: color-mix(in srgb, var(--ms-violet) 40%, var(--ms-line));
  background: color-mix(in srgb, var(--ms-violet) 12%, var(--ms-panel));
}

.price-box-unset {
  border-style: dashed;
  border-color: var(--ms-line);
  background: var(--ms-panel-2);
}

.price-box span {
  @apply block text-xs font-medium;
  color: var(--ms-muted);
}

.price-box-teal span {
  color: var(--ms-brand);
}

.price-box-orange span {
  color: var(--ms-orange);
}

.price-box-blue span {
  color: var(--ms-blue);
}

.price-box-violet span {
  color: var(--ms-violet);
}

.price-box strong {
  @apply mt-1 block font-mono text-sm font-bold;
  color: var(--ms-text);
}

.price-box-teal strong {
  color: var(--ms-brand-strong);
}

.price-box-orange strong {
  color: var(--ms-orange);
}

.price-box-blue strong {
  color: var(--ms-blue);
}

.price-box-violet strong {
  color: var(--ms-violet);
}

.price-box-unset strong {
  color: var(--ms-dim);
}

.price-box small {
  @apply mt-0.5 block text-[11px] font-medium;
  color: var(--ms-dim);
}

.price-original {
  @apply mt-0.5 block text-[11px] font-medium leading-none line-through;
  color: var(--ms-dim);
  opacity: 0.85;
}

.price-cell {
  @apply inline-flex items-center rounded border px-1.5 py-0.5 font-mono text-xs font-semibold;
}

.price-cell-teal {
  background: color-mix(in srgb, var(--ms-brand) 12%, var(--ms-panel));
  color: var(--ms-brand-strong);
  border-color: color-mix(in srgb, var(--ms-brand) 30%, var(--ms-line));
}

.price-cell-orange {
  background: color-mix(in srgb, var(--ms-orange) 12%, var(--ms-panel));
  color: var(--ms-orange);
  border-color: color-mix(in srgb, var(--ms-orange) 30%, var(--ms-line));
}

.price-cell-blue {
  background: color-mix(in srgb, var(--ms-blue) 12%, var(--ms-panel));
  color: var(--ms-blue);
  border-color: color-mix(in srgb, var(--ms-blue) 30%, var(--ms-line));
}

.price-cell-violet {
  background: color-mix(in srgb, var(--ms-violet) 12%, var(--ms-panel));
  color: var(--ms-violet);
  border-color: color-mix(in srgb, var(--ms-violet) 30%, var(--ms-line));
}

.price-cell-unset {
  border-style: dashed;
  border-color: var(--ms-line);
  background: var(--ms-panel-2);
  color: var(--ms-dim);
}

.model-card-footer {
  @apply mt-auto flex flex-wrap items-end justify-between gap-3 pt-4;
}

.primary-group-chip {
  @apply inline-flex min-w-0 max-w-[68%] items-center gap-1 rounded-md px-2 py-1 text-xs font-medium transition;
  color: var(--ms-muted);
}

.primary-group-chip:hover {
  background: color-mix(in srgb, var(--ms-amber) 10%, var(--ms-panel));
  color: var(--ms-amber);
}

.primary-group-chip b {
  @apply shrink-0 font-bold;
  color: var(--ms-amber);
}

.model-rate-chip {
  @apply inline-flex h-7 shrink-0 items-center rounded-md px-2.5 font-mono text-xs font-bold;
  background: color-mix(in srgb, var(--ms-amber) 12%, var(--ms-panel));
  color: var(--ms-amber);
  border: 1px solid color-mix(in srgb, var(--ms-amber) 30%, transparent);
}

.model-detail-button {
  @apply ml-auto inline-flex h-7 shrink-0 items-center gap-1 rounded-md border px-2 text-xs font-semibold transition;
  border-color: var(--ms-line);
  color: var(--ms-muted);
}

.model-detail-button:hover {
  border-color: var(--ms-brand);
  background: color-mix(in srgb, var(--ms-brand) 9%, var(--ms-panel));
  color: var(--ms-brand-strong);
}

.detail-group-option {
  @apply flex min-w-0 items-center justify-between gap-3 rounded-lg border px-3 py-2.5 text-sm transition;
  border-color: var(--ms-line);
  background: var(--ms-panel);
  color: var(--ms-muted);
}

.detail-group-option:hover {
  border-color: color-mix(in srgb, var(--ms-amber) 55%, var(--ms-line));
  background: color-mix(in srgb, var(--ms-amber) 8%, var(--ms-panel));
}

.detail-group-option.active {
  border-color: var(--ms-amber);
  background: color-mix(in srgb, var(--ms-amber) 12%, var(--ms-panel));
  color: var(--ms-amber);
  box-shadow: var(--ms-shadow);
}

.detail-price-section {
  @apply rounded-lg border p-3;
  border-color: var(--ms-line);
  background: color-mix(in srgb, var(--ms-brand) 5%, var(--ms-panel-2));
}

.group-overflow {
  @apply shrink-0 rounded px-1 font-mono text-[10px];
  background: var(--ms-panel-3);
  color: var(--ms-muted);
}

.group-chip {
  @apply inline-flex max-w-full cursor-pointer items-center gap-1 rounded px-2 py-1 text-xs;
  background: var(--ms-panel-3);
  color: var(--ms-muted);
}

.group-chip:hover {
  background: color-mix(in srgb, var(--ms-amber) 10%, var(--ms-panel));
  color: var(--ms-amber);
}

.group-chip b {
  @apply font-semibold;
  color: var(--ms-amber);
}

.group-more {
  @apply rounded px-2 py-1 text-xs font-semibold transition;
  background: color-mix(in srgb, var(--ms-brand) 10%, var(--ms-panel));
  color: var(--ms-brand-strong);
}

.group-more:hover {
  background: color-mix(in srgb, var(--ms-brand) 16%, var(--ms-panel));
}

@media (max-width: 480px) {
  .model-square-board {
    @apply p-3;
  }

  .model-card-grid {
    grid-template-columns: 1fr;
  }

  .model-card {
    @apply p-4;
  }
}
</style>

<style>
/* 模型广场主题变量：明/暗两套，挂在 .model-square-root 上，覆盖 Teleport 到 body 的弹窗内容。 */
.model-square-root {
  --ms-panel: #ffffff;
  --ms-panel-2: #f9fafb;
  --ms-panel-3: #f3f4f6;
  --ms-line: #e5e7eb;
  --ms-soft: #f1f5f9;
  --ms-text: #111827;
  --ms-muted: #64748b;
  --ms-dim: #94a3b8;
  --ms-brand: #0f766e;
  --ms-brand-strong: #115e59;
  --ms-glow: rgba(20, 184, 166, 0.3);
  --ms-green: #047857;
  --ms-amber: #b45309;
  --ms-orange: #ea580c;
  --ms-blue: #1d4ed8;
  --ms-violet: #6d28d9;
  --ms-red: #b91c1c;
  --ms-shadow: 0 1px 3px rgba(15, 23, 42, 0.06);
}

.dark .model-square-root {
  --ms-panel: #1e293b;
  --ms-panel-2: #334155;
  --ms-panel-3: #374151;
  --ms-line: #334155;
  --ms-soft: #374151;
  --ms-text: #f9fafb;
  --ms-muted: #9ca3af;
  --ms-dim: #6b7280;
  --ms-brand: #5eead4;
  --ms-brand-strong: #2dd4bf;
  --ms-glow: rgba(45, 212, 191, 0.32);
  --ms-green: #34d399;
  --ms-amber: #fbbf24;
  --ms-orange: #fb923c;
  --ms-blue: #60a5fa;
  --ms-violet: #a78bfa;
  --ms-red: #f87171;
  --ms-shadow: 0 1px 3px rgba(0, 0, 0, 0.3);
}
</style>

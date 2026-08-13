<template>
  <AppLayout>
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

          <select v-model="groupFilter" class="input w-full sm:w-44">
            <option value="">{{ t('admin.modelSquare.allGroups') }}</option>
            <option v-for="group in groups" :key="String(group.id)" :value="String(group.id)">
              {{ group.name }}
            </option>
          </select>

          <select v-model="providerFilter" class="input w-full sm:w-44">
            <option value="">{{ t('admin.modelSquare.allProviders') }}</option>
            <option v-for="item in providers" :key="item" :value="item">{{ providerLabel(item) }}</option>
          </select>

          <select v-model="modeFilter" class="input w-full sm:w-40">
            <option value="">{{ t('admin.modelSquare.allModes') }}</option>
            <option v-for="item in modes" :key="item" :value="item">{{ modeLabel(item) }}</option>
          </select>

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
                @click="copyModelId(model)"
                @keydown.enter.prevent="copyModelId(model)"
              >
                <div class="model-card-top">
                  <span class="model-provider">
                    <span class="model-provider-dot"></span>
                    <span class="truncate">{{ providerLabel(model.provider) }}</span>
                  </span>
                  <span :class="['model-status', isAvailable(model) ? 'model-status-available' : 'model-status-muted']">
                    <span class="status-dot"></span>
                    {{ copiedModelId === model.id ? t('admin.modelSquare.copied') : availabilityLabel(model) }}
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
                    <strong>{{ formatPriceOrUnset(slot.value) }}</strong>
                    <small v-if="slot.unit">{{ slot.unit }}</small>
                  </div>
                </div>

                <div class="model-card-footer">
                  <span class="mode-chip">{{ modeLabel(model.mode) }}</span>
                  <button
                    type="button"
                    class="model-detail-button"
                    title="详情"
                    @click.stop="openModelDetails(model)"
                  >
                    <Icon name="eye" size="xs" />
                    <span>详情</span>
                  </button>
                  <button
                    v-if="modelGroups(model).length > 0"
                    type="button"
                    class="primary-group-chip"
                    @click.stop="openGroupDialog(model)"
                  >
                    <span class="truncate">{{ primaryGroup(model)?.name }}</span>
                    <span v-if="modelGroupOverflowCount(model) > 0" class="group-overflow">+{{ modelGroupOverflowCount(model) }}</span>
                  </button>
                  <span class="model-rate-chip">{{ formatRate(modelEffectiveRate(model)) }}</span>
                </div>
              </article>
            </div>
          </section>
        </div>

        <div v-else class="overflow-x-auto">
          <table class="w-full min-w-[980px] divide-y divide-gray-100 text-sm dark:divide-dark-700">
            <thead class="bg-gray-50 dark:bg-dark-800">
              <tr>
                <th class="px-4 py-3 text-left font-medium">{{ t('admin.modelSquare.columns.status') }}</th>
                <th class="px-4 py-3 text-left font-medium">{{ t('admin.modelSquare.columns.provider') }}</th>
                <th class="px-4 py-3 text-left font-medium">{{ t('admin.modelSquare.columns.modelId') }}</th>
                <th class="px-4 py-3 text-left font-medium">{{ t('admin.modelSquare.columns.input') }}</th>
                <th class="px-4 py-3 text-left font-medium">{{ t('admin.modelSquare.columns.output') }}</th>
                <th class="px-4 py-3 text-left font-medium">{{ t('admin.modelSquare.columns.cacheRead') }}</th>
                <th class="px-4 py-3 text-left font-medium">{{ t('admin.modelSquare.columns.cacheWrite') }}</th>
                <th class="px-4 py-3 text-left font-medium">{{ t('admin.modelSquare.columns.mode') }}</th>
                <th class="px-4 py-3 text-left font-medium">{{ t('admin.modelSquare.columns.groups') }}</th>
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
                  <span :class="['badge', isAvailable(model) ? 'badge-success' : 'badge-gray']">
                    {{ copiedModelId === model.id ? t('admin.modelSquare.copied') : availabilityLabel(model) }}
                  </span>
                </td>
                <td class="whitespace-nowrap px-4 py-3">{{ providerLabel(model.provider) }}</td>
                <td class="max-w-72 px-4 py-3 font-medium text-gray-950 dark:text-white">
                  <span class="break-words">{{ modelDisplayName(model) }}</span>
                </td>
                <td class="whitespace-nowrap px-4 py-3 font-mono">{{ formatPriceOrUnset(modelPriceValue(model, 'input_price')) }}</td>
                <td class="whitespace-nowrap px-4 py-3 font-mono">{{ formatPriceOrUnset(modelPriceValue(model, 'output_price')) }}</td>
                <td class="whitespace-nowrap px-4 py-3 font-mono">{{ formatPriceOrUnset(modelPriceValue(model, 'cache_read_price')) }}</td>
                <td class="whitespace-nowrap px-4 py-3 font-mono">{{ formatPriceOrUnset(modelPriceValue(model, 'cache_write_price')) }}</td>
                <td class="whitespace-nowrap px-4 py-3">{{ modeLabel(model.mode) }}</td>
                <td class="px-4 py-3">
                  <div class="flex min-w-72 flex-wrap gap-1.5">
                    <span
                      v-for="group in modelGroups(model)"
                      :key="String(group.id)"
                      class="group-chip"
                    >
                      {{ group.name }}
                      <b>{{ formatRate(group.rate_multiplier) }}</b>
                    </span>
                  </div>
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
      <div v-if="detailModel" class="space-y-5">
        <div class="flex flex-wrap items-start justify-between gap-3 border-b border-gray-100 pb-4 dark:border-dark-700">
          <div class="min-w-0">
            <div class="break-words text-lg font-bold text-gray-950 dark:text-white">{{ modelDisplayName(detailModel) }}</div>
            <code class="mt-1 block break-all text-xs text-gray-400">{{ detailModel.id }}</code>
          </div>
          <div class="flex shrink-0 flex-wrap items-center gap-2 text-xs">
            <span class="mode-chip">{{ modeLabel(detailModel.mode) }}</span>
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
              <strong>{{ formatPriceOrUnset(slot.value) }}</strong>
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
      <div class="max-h-[56vh] space-y-2 overflow-y-auto">
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
import { adminAPI } from '@/api/admin'
import type { AdminModelSquareResult, ModelSquareGroup, ModelSquareModel } from '@/api/admin'
import { useAppStore } from '@/stores/app'
import { extractApiErrorMessage } from '@/utils/apiError'
import { useRouteQueryFilters } from '@/composables/useRouteQueryFilters'
import AppLayout from '@/components/layout/AppLayout.vue'
import TablePageLayout from '@/components/layout/TablePageLayout.vue'
import BaseDialog from '@/components/common/BaseDialog.vue'
import EmptyState from '@/components/common/EmptyState.vue'
import Icon from '@/components/icons/Icon.vue'

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
const modeFilter = ref('')
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
  { key: 'input_price', label: '\u8f93\u5165', unit: '$/\u767e\u4e07 tokens', toneClass: 'price-box-neutral' },
  { key: 'output_price', label: '\u8f93\u51fa', unit: '$/\u767e\u4e07 tokens', toneClass: 'price-box-neutral' },
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
const modes = computed(() => unique(models.value.map(model => model.mode || 'chat')))
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
    if (modeFilter.value && (model.mode || 'chat') !== modeFilter.value) return false
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
    result.value = await adminAPI.modelSquare.get()
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
  const slots = defaultPriceDescriptors.map(descriptor => ({ ...descriptor, value: modelPriceValue(model, descriptor.key, multiplier) }))
  const extraSlots = priceDescriptors
    .filter(descriptor => !defaultPriceDescriptors.some(defaultDescriptor => defaultDescriptor.key === descriptor.key))
    .map(descriptor => ({ ...descriptor, value: modelPriceValue(model, descriptor.key, multiplier) }))
    .filter(slot => slot.value != null)

  for (const extraSlot of extraSlots) {
    const emptyIndex = slots.findIndex(slot => slot.value == null)
    if (emptyIndex < 0) break
    slots[emptyIndex] = extraSlot
  }

  return slots
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

function formatPriceOrUnset(value?: number | string) {
  return value == null || value === '' ? '\u672a\u8bbe\u7f6e' : formatPrice(value)
}

function isAvailable(model: ModelSquareModel) {
  return model.available !== false
}

function availabilityLabel(model: ModelSquareModel) {
  return isAvailable(model) ? t('admin.modelSquare.available') : t('admin.modelSquare.unavailable')
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

function modeLabel(value?: string) {
  if (value === 'image_generation') return t('admin.modelSquare.modes.image')
  if (value === 'embedding') return t('admin.modelSquare.modes.embedding')
  if (value === 'responses') return t('admin.modelSquare.modes.responses')
  return value || t('admin.modelSquare.modes.chat')
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
  @apply flex h-11 items-center gap-3 rounded-lg border border-gray-200 px-3 text-sm text-gray-600 dark:border-dark-600 dark:text-gray-300;
}

.summary-pill strong {
  @apply font-mono text-base text-gray-900 dark:text-white;
}

.view-toggle-btn {
  @apply grid h-8 w-8 place-items-center rounded-md text-gray-500 transition-colors hover:bg-white hover:text-gray-900 dark:hover:bg-dark-700 dark:hover:text-white;
}

.view-toggle-btn.active {
  @apply bg-white text-primary-600 shadow-sm dark:bg-dark-700 dark:text-primary-400;
}

.model-square-board {
  @apply h-full space-y-5 overflow-y-auto p-4;
}

.provider-section {
  @apply space-y-3;
}

.provider-section-header {
  @apply flex items-center justify-between gap-3 px-1;
}

.provider-title {
  @apply flex min-w-0 items-center gap-2 text-sm font-semibold text-gray-900 dark:text-white;
}

.provider-dot,
.model-provider-dot {
  @apply h-2 w-2 shrink-0 rounded-full bg-slate-400;
}

.provider-meta {
  @apply mt-0.5 text-xs text-gray-500 dark:text-gray-400;
}

.provider-count {
  @apply grid h-7 min-w-7 place-items-center rounded-md bg-gray-100 px-2 font-mono text-xs font-semibold text-gray-600 dark:bg-dark-700 dark:text-gray-300;
}

.model-card-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(320px, 1fr));
  @apply gap-4;
}

.model-card {
  @apply relative flex min-h-[16.5rem] cursor-pointer flex-col rounded-lg border border-gray-200 bg-white p-5 transition duration-200 dark:border-dark-700 dark:bg-dark-800;
}

.model-card:hover,
.model-card:focus-visible {
  @apply -translate-y-0.5 border-cyan-400 outline-none dark:border-cyan-500;
  box-shadow: 0 16px 38px rgba(15, 23, 42, 0.08), 0 0 0 1px rgba(6, 182, 212, 0.18);
}

.model-card-top {
  @apply flex items-start justify-between gap-3;
}

.model-provider {
  @apply inline-flex min-w-0 items-center gap-2 text-xs font-semibold text-gray-500 dark:text-gray-400;
}

.model-status {
  @apply inline-flex shrink-0 items-center gap-1 rounded-md px-2.5 py-1 text-xs font-semibold;
}

.model-status-available {
  @apply bg-emerald-100 text-emerald-700 dark:bg-emerald-500/15 dark:text-emerald-300;
}

.model-status-muted {
  @apply bg-gray-100 text-gray-500 dark:bg-dark-700 dark:text-gray-300;
}

.status-dot {
  @apply h-1.5 w-1.5 rounded-full bg-current opacity-70;
}

.model-title-row {
  @apply mt-3 flex min-h-[2.5rem] items-start gap-2;
}

.model-title {
  @apply min-w-0 flex-1 text-base font-bold leading-snug text-gray-950 transition-colors dark:text-white;
  overflow-wrap: anywhere;
}

.model-card:hover .model-title,
.model-card:focus-visible .model-title {
  @apply text-teal-700 dark:text-teal-300;
}

.copy-button {
  @apply grid h-8 w-8 shrink-0 place-items-center rounded-md text-gray-300 opacity-0 transition hover:bg-gray-100 hover:text-teal-700 focus:opacity-100 dark:hover:bg-dark-700 dark:hover:text-teal-300;
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
}

.price-box-neutral {
  @apply border-gray-100 bg-gray-50 dark:border-dark-700 dark:bg-dark-700/50;
}

.price-box-blue {
  @apply border-blue-100 bg-blue-50 text-blue-700 dark:border-blue-900/50 dark:bg-blue-950/30 dark:text-blue-300;
}

.price-box-violet {
  @apply border-violet-100 bg-violet-50 text-violet-700 dark:border-violet-900/50 dark:bg-violet-950/30 dark:text-violet-300;
}

.price-box span {
  @apply block text-xs font-medium text-gray-500 dark:text-gray-400;
}

.price-box strong {
  @apply mt-1 block font-mono text-sm font-bold text-gray-950 dark:text-white;
}

.price-box-blue strong {
  @apply text-blue-700 dark:text-blue-300;
}

.price-box-violet strong {
  @apply text-violet-700 dark:text-violet-300;
}

.price-box small {
  @apply mt-0.5 block text-[11px] font-medium text-gray-400 dark:text-gray-500;
}

.model-card-footer {
  @apply mt-auto flex flex-wrap items-end justify-between gap-3 pt-4;
}

.mode-chip {
  @apply inline-flex h-7 items-center rounded-md bg-blue-50 px-2.5 text-xs font-semibold text-blue-700 dark:bg-blue-950/40 dark:text-blue-300;
}

.primary-group-chip {
  @apply inline-flex min-w-0 max-w-[68%] items-center gap-1 rounded-md px-2 py-1 text-xs font-medium text-gray-500 transition hover:bg-amber-50 hover:text-amber-700 dark:text-gray-400 dark:hover:bg-amber-950/30 dark:hover:text-amber-300;
}

.primary-group-chip b {
  @apply shrink-0 font-bold text-orange-500;
}

.model-rate-chip {
  @apply ml-auto inline-flex h-7 shrink-0 items-center rounded-md bg-amber-50 px-2.5 font-mono text-xs font-bold text-orange-600 dark:bg-amber-950/30 dark:text-orange-300;
}

.model-detail-button {
  @apply inline-flex h-7 shrink-0 items-center gap-1 rounded-md border border-gray-200 px-2 text-xs font-semibold text-gray-500 transition hover:border-cyan-300 hover:bg-cyan-50 hover:text-cyan-700 dark:border-dark-600 dark:text-gray-400 dark:hover:border-cyan-700 dark:hover:bg-cyan-950/30 dark:hover:text-cyan-300;
}

.detail-group-option {
  @apply flex min-w-0 items-center justify-between gap-3 rounded-lg border border-gray-200 bg-white px-3 py-2.5 text-sm text-gray-700 transition hover:border-orange-300 hover:bg-orange-50 dark:border-dark-600 dark:bg-dark-800 dark:text-gray-300 dark:hover:border-orange-700 dark:hover:bg-orange-950/20;
}

.detail-group-option.active {
  @apply border-orange-400 bg-orange-50 text-orange-800 shadow-sm dark:border-orange-500 dark:bg-orange-950/30 dark:text-orange-200;
}

.detail-price-section {
  @apply rounded-lg border border-gray-100 bg-gray-50/70 p-3 dark:border-dark-700 dark:bg-dark-800/60;
}

.group-overflow {
  @apply shrink-0 rounded bg-gray-100 px-1 font-mono text-[10px] text-gray-500 dark:bg-dark-700 dark:text-gray-300;
}

.group-chip {
  @apply inline-flex max-w-full items-center gap-1 rounded bg-gray-100 px-2 py-1 text-xs text-gray-600 dark:bg-dark-700 dark:text-gray-300;
}

.group-chip b {
  @apply font-semibold text-orange-500;
}

.group-more {
  @apply rounded bg-primary-50 px-2 py-1 text-xs font-semibold text-primary-700 transition hover:bg-primary-100 dark:bg-primary-900/30 dark:text-primary-300 dark:hover:bg-primary-900/50;
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

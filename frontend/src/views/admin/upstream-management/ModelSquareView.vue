<template>
  <AppLayout>
    <TablePageLayout>
      <template #filters>
        <div class="flex flex-wrap items-center gap-3">
          <div class="flex min-w-0 flex-1 flex-wrap items-center gap-3">
            <div class="rounded-lg border border-gray-200 px-3 py-2 dark:border-dark-600">
              <div class="text-xs text-gray-500 dark:text-gray-400">{{ t('admin.modelSquare.defaultProvider') }}</div>
              <div class="mt-0.5 flex items-center gap-2 text-sm font-semibold text-gray-900 dark:text-white">
                <span>{{ result?.provider_name || '-' }}</span>
                <code v-if="result?.provider_slug" class="text-xs font-normal text-gray-500 dark:text-gray-400">
                  {{ result.provider_slug }}
                </code>
                <span v-if="result?.provider_type" class="badge badge-gray">{{ providerTypeLabel(result.provider_type) }}</span>
              </div>
            </div>

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
            <option v-for="item in providers" :key="item" :value="item">{{ item }}</option>
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
          :description="t('admin.modelSquare.emptyDescription')"
          :action-text="t('common.refresh')"
          @action="reload"
        />

        <div v-else-if="viewMode === 'grid'" class="grid gap-4 p-1 lg:grid-cols-2 xl:grid-cols-3">
          <article
            v-for="(model, index) in filteredModels"
            :key="modelKey(model, index)"
            data-test="model-card"
            class="model-card"
            role="button"
            tabindex="0"
            :title="t('admin.modelSquare.copyTitle')"
            @click="copyModelId(model)"
            @keydown.enter.prevent="copyModelId(model)"
          >
            <div class="flex items-start justify-between gap-3">
              <span class="inline-flex min-w-0 items-center gap-2 text-xs font-semibold text-gray-500 dark:text-gray-400">
                <span class="h-2 w-2 shrink-0 rounded-full bg-slate-400"></span>
                <span class="truncate">{{ model.provider || 'unknown' }}</span>
              </span>
              <span :class="['badge', isAvailable(model) ? 'badge-success' : 'badge-gray']">
                {{ copiedModelId === model.id ? t('admin.modelSquare.copied') : availabilityLabel(model) }}
              </span>
            </div>

            <div class="mt-4 flex items-start gap-2">
              <h3 class="min-w-0 flex-1 break-words text-base font-semibold text-gray-950 dark:text-white">
                {{ model.id || t('admin.modelSquare.unnamedModel') }}
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

            <div class="mt-4 grid grid-cols-2 gap-3">
              <div class="price-box">
                <span>{{ t('admin.modelSquare.inputPrice') }}</span>
                <strong>{{ formatPrice(modelDisplayPrice(model, 'input_price')) }}</strong>
              </div>
              <div class="price-box">
                <span>{{ t('admin.modelSquare.outputPrice') }}</span>
                <strong>{{ formatPrice(modelDisplayPrice(model, 'output_price')) }}</strong>
              </div>
              <div class="price-box">
                <span>{{ t('admin.modelSquare.cacheReadPrice') }}</span>
                <strong>{{ formatPrice(modelDisplayPrice(model, 'cache_read_price')) }}</strong>
              </div>
              <div class="price-box">
                <span>{{ t('admin.modelSquare.cacheWritePrice') }}</span>
                <strong>{{ formatPrice(modelDisplayPrice(model, 'cache_create_price')) }}</strong>
              </div>
            </div>

            <div class="mt-4 flex items-center justify-between gap-3 border-t border-gray-100 pt-4 dark:border-dark-700">
              <span class="badge badge-primary">{{ modeLabel(model.mode) }}</span>
              <span class="text-xs text-gray-400 dark:text-gray-500">{{ t('admin.modelSquare.perMillionTokens') }}</span>
            </div>

            <div class="mt-3 flex flex-wrap gap-1.5">
              <span
                v-for="group in modelGroups(model).slice(0, 3)"
                :key="String(group.id)"
                class="group-chip"
              >
                {{ group.name }}
                <b>{{ formatRate(group.rate_multiplier) }}</b>
              </span>
              <button
                v-if="modelGroups(model).length > 3"
                type="button"
                class="group-more"
                @click.stop="openGroupDialog(model)"
              >
                +{{ modelGroups(model).length - 3 }}
              </button>
            </div>
          </article>
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
                class="cursor-pointer transition hover:bg-gray-50 dark:hover:bg-dark-700/60"
                :title="t('admin.modelSquare.copyTitle')"
                @click="copyModelId(model)"
              >
                <td class="whitespace-nowrap px-4 py-3">
                  <span :class="['badge', isAvailable(model) ? 'badge-success' : 'badge-gray']">
                    {{ copiedModelId === model.id ? t('admin.modelSquare.copied') : availabilityLabel(model) }}
                  </span>
                </td>
                <td class="whitespace-nowrap px-4 py-3">{{ model.provider || 'unknown' }}</td>
                <td class="max-w-72 px-4 py-3 font-medium text-gray-950 dark:text-white">
                  <span class="break-words">{{ model.id || t('admin.modelSquare.unnamedModel') }}</span>
                </td>
                <td class="whitespace-nowrap px-4 py-3 font-mono">{{ formatPrice(modelDisplayPrice(model, 'input_price')) }}</td>
                <td class="whitespace-nowrap px-4 py-3 font-mono">{{ formatPrice(modelDisplayPrice(model, 'output_price')) }}</td>
                <td class="whitespace-nowrap px-4 py-3 font-mono">{{ formatPrice(modelDisplayPrice(model, 'cache_read_price')) }}</td>
                <td class="whitespace-nowrap px-4 py-3 font-mono">{{ formatPrice(modelDisplayPrice(model, 'cache_create_price')) }}</td>
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
import AppLayout from '@/components/layout/AppLayout.vue'
import TablePageLayout from '@/components/layout/TablePageLayout.vue'
import BaseDialog from '@/components/common/BaseDialog.vue'
import EmptyState from '@/components/common/EmptyState.vue'
import Icon from '@/components/icons/Icon.vue'

type PriceField = 'input_price' | 'output_price' | 'cache_read_price' | 'cache_create_price'

const { t } = useI18n()
const appStore = useAppStore()

const result = ref<AdminModelSquareResult | null>(null)
const loading = ref(false)
const loadError = ref('')
const searchQuery = ref('')
const providerFilter = ref('')
const modeFilter = ref('')
const groupFilter = ref('')
const viewMode = ref<'grid' | 'list'>('grid')
const groupDialogModel = ref<ModelSquareModel | null>(null)
const copiedModelId = ref('')
let copiedTimer: ReturnType<typeof setTimeout> | undefined

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

function primaryGroupRate(model: ModelSquareModel) {
  const rates = modelGroups(model)
    .map(groupRate)
    .filter(rate => Number.isFinite(rate))
  return rates.length > 0 ? Math.min(...rates) : 1
}

function groupRate(group?: ModelSquareGroup) {
  const rate = Number(group?.rate_multiplier)
  return Number.isFinite(rate) ? rate : Number.POSITIVE_INFINITY
}

function modelDisplayPrice(model: ModelSquareModel, field: PriceField) {
  const price = Number(model[field] ?? 0)
  if (!Number.isFinite(price)) return 0
  return price * primaryGroupRate(model)
}

function isAvailable(model: ModelSquareModel) {
  return model.available !== false
}

function availabilityLabel(model: ModelSquareModel) {
  return isAvailable(model) ? t('admin.modelSquare.available') : t('admin.modelSquare.unavailable')
}

function modelSearchText(model: ModelSquareModel) {
  return [model.id, model.provider, model.mode]
    .filter(Boolean)
    .join(' ')
    .toLowerCase()
}

function modelKey(model: ModelSquareModel, index: number) {
  return `${model.provider || 'unknown'}:${model.id || index}`
}

function providerTypeLabel(type: string) {
  if (type === 'newapi') return 'NewAPI'
  return 'Sub2API'
}

function modeLabel(value?: string) {
  if (value === 'image_generation') return 'Image'
  if (value === 'embedding') return 'Embedding'
  if (value === 'responses') return 'Responses'
  return value || 'Chat'
}

function formatRate(value?: number) {
  const n = Number(value)
  if (!Number.isFinite(n)) return '-'
  return `${n.toFixed(3).replace(/0+$/, '').replace(/\.$/, '')}x`
}

function formatPrice(value?: number | string) {
  const n = Number(value ?? 0)
  if (!Number.isFinite(n)) return '$0'
  return `$${n.toFixed(n >= 10 ? 2 : 3).replace(/0+$/, '').replace(/\.$/, '')}`
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

.model-card {
  @apply flex min-h-[18rem] cursor-pointer flex-col rounded-lg border border-gray-200 bg-white p-4 transition hover:border-primary-300 hover:shadow-sm dark:border-dark-700 dark:bg-dark-800 dark:hover:border-primary-700;
}

.copy-button {
  @apply grid h-8 w-8 shrink-0 place-items-center rounded-md text-gray-400 transition hover:bg-gray-100 hover:text-primary-600 dark:hover:bg-dark-700 dark:hover:text-primary-400;
}

.price-box {
  @apply rounded-lg border border-gray-100 bg-gray-50 p-3 dark:border-dark-700 dark:bg-dark-700/60;
}

.price-box span {
  @apply block text-xs text-gray-500 dark:text-gray-400;
}

.price-box strong {
  @apply mt-1 block font-mono text-sm text-gray-950 dark:text-white;
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
</style>

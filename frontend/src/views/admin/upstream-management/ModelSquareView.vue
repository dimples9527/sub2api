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
            <span v-if="hasActiveFilters" class="result-count">
              {{ t('admin.modelSquare.filteredCount', { filtered: sortedModels.length, total: models.length }) }}
            </span>
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

          <Select
            v-model="sortBy"
            :options="sortOptions"
            class="w-full sm:w-40"
            :aria-label="t('admin.modelSquare.sortBy')"
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
          :title="hasActiveFilters ? t('admin.modelSquare.noMatchTitle') : t('admin.modelSquare.emptyTitle')"
          :description="hasActiveFilters ? t('admin.modelSquare.noMatchDescription') : emptyDescription"
        >
          <template #action>
            <button v-if="hasActiveFilters" type="button" class="btn btn-primary" @click="clearFilters">
              <Icon name="x" size="sm" />
              {{ t('admin.modelSquare.clearFilters') }}
            </button>
            <button v-else type="button" class="btn btn-primary" @click="reload">
              <Icon name="refresh" size="sm" />
              {{ t('common.refresh') }}
            </button>
          </template>
        </EmptyState>

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
                    v-for="slot in modelPriceSlots(model, modelEffectiveRate(model))"
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
                    :class="primaryGroupBadgeClass(model)"
                    :title="primaryGroupTitle(model)"
                    :disabled="!!activeFilterGroup"
                    @click.stop="openGroupDialog(model)"
                  >
                    <span class="truncate">{{ primaryGroup(model)?.name }}</span>
                    <span v-if="modelGroupOverflowCount(model) > 0" class="group-overflow">+{{ modelGroupOverflowCount(model) }}</span>
                  </button>
                  <!-- 已按分组过滤时：卡片价已是该分组价，详情（列全部分组）就多余了，隐藏。 -->
                  <button
                    type="button"
                    v-if="!activeFilterGroup"
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
          <!--
            ⚠️ 这里的 min-w-* 是失效的：TablePageLayout 的 `.table-scroll-container :deep(table)`
            设了 `min-width: max-content`（选择器权重更高），表格宽度一律由内容撑开、超出即横向滚动。
            所以价格列一改宽，整表宽度就跟着涨，只能从列内容本身下手。
          -->
          <table class="w-full min-w-[1100px] divide-y divide-gray-100 text-sm dark:divide-dark-700">
            <thead class="bg-gray-50 dark:bg-dark-800">
              <tr>
                <th class="px-4 py-3 text-left font-medium">{{ t('admin.modelSquare.columns.provider') }}</th>
                <th class="px-4 py-3 text-left font-medium">{{ t('admin.modelSquare.columns.modelId') }}</th>
                <!--
                  四个价格位合并成一列：卡片自带标签，列头再写「输入/输出/缓存读取/缓存写入」就是重复信息，
                  而且四列只有数字、没有单位，横向扫读时分不清哪个是哪个 —— 这正是「看不出层级」的来源。
                -->
                <th class="px-4 py-3 text-left font-medium">{{ t('admin.modelSquare.columns.price') }}</th>
                <th class="px-4 py-3 text-left font-medium">{{ t('admin.modelSquare.columns.groups') }}</th>
                <th class="px-4 py-3 text-right font-medium">{{ t('admin.modelSquare.columns.actions') }}</th>
              </tr>
            </thead>
            <tbody class="divide-y divide-gray-100 dark:divide-dark-700">
              <tr
                v-for="(model, index) in sortedModels"
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
                <td class="px-4 py-3">
                  <!--
                    与卡片视图共用同一套 price-box 卡片 —— 两个视图之间切换时，价格的语言必须一致，
                    否则管理员会以为换了个视图就换了套价格。

                    这段 markup 刻意与上方卡片视图的 .price-grid 保持镜像（唯一差异是外层容器类），
                    因为两个视图在 DOM 里没有共同的祖先可以挂组件：卡片在 <article> 里，这里在 <td> 里。
                    视觉规则全部在 .price-box* 样式里，改一边就要改另一边。
                  -->
                  <div class="list-price-grid">
                    <div
                      v-for="slot in modelPriceSlots(model, modelEffectiveRate(model))"
                      :key="slot.key"
                      :class="['price-box', slot.toneClass]"
                    >
                      <span>{{ slot.label }}</span>
                      <strong>{{ formatPriceOrZero(slot.value) }}</strong>
                      <s v-if="slot.originalValue != null" class="price-original">{{ formatPrice(slot.originalValue) }}</s>
                      <small v-if="slot.unit">{{ slot.unit }}</small>
                    </div>
                  </div>
                </td>
                <td class="px-4 py-3">
                  <div class="flex min-w-56 flex-wrap gap-1.5">
                    <button
                      v-for="group in modelGroups(model)"
                      :key="String(group.id)"
                      type="button"
                      class="group-chip"
                      :class="groupPlatformBadgeClass(group)"
                      :title="groupChipTitle(group)"
                      :disabled="!!activeFilterGroup"
                      @click.stop="openGroupDialog(model)"
                    >
                      {{ group.name }}
                      <b>{{ formatRate(group.rate_multiplier) }}</b>
                    </button>
                    <span v-if="modelGroups(model).length === 0" class="text-xs text-gray-400 dark:text-gray-500">—</span>
                  </div>
                </td>
                <td class="whitespace-nowrap px-4 py-3 text-right">
                  <!-- 已按分组过滤时隐藏详情，理由同网格视图。 -->
                  <button
                    type="button"
                    v-if="!activeFilterGroup"
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
          <div class="mb-3 flex items-center justify-between gap-3">
            <span class="text-sm font-semibold text-gray-950 dark:text-white">{{ t('admin.modelSquare.groupPricingTitle') }}</span>
            <span class="provider-inline text-xs" :style="providerAccent(detailModel.provider)">平台: {{ providerLabel(detailModel.provider) }}</span>
          </div>

          <div v-if="detailGroups.length > 0" class="group-price-table-wrap">
            <table class="group-price-table">
              <thead>
                <tr>
                  <th scope="col">{{ t('admin.modelSquare.columns.groups') }}</th>
                  <th scope="col">{{ t('admin.modelSquare.rate') }}</th>
                  <!--
                    四个价格位合并成一列，与列表视图同一处理：卡片自带标签，
                    再摆四个「输入/输出/缓存读取/缓存写入」的列头就是重复信息。
                  -->
                  <th scope="col">{{ t('admin.modelSquare.columns.price') }}</th>
                </tr>
              </thead>
              <tbody>
                <tr
                  v-for="group in detailGroups"
                  :key="String(group.id)"
                  data-test="detail-group-row"
                  :class="{ active: isCurrentRateGroup(group) }"
                >
                  <td class="group-cell">
                    <!--
                      分组名与倍率都做成胶囊：这一行读的是「哪个分组、什么倍率、多少钱」，
                      前两样是标识而不是数字，胶囊才能把它们和右边的价格块分开。
                      分组名在这里不可点，所以用 span 复用列表视图那枚胶囊 ——
                      hover 反馈只挂在 button 上，挂在 span 上会让它看起来能点。
                      配色也复用同一枚：同一个分组在卡片、列表、弹窗里必须是同一种颜色，
                      否则管理员会以为它们是两个不同的分组。
                    -->
                    <span class="group-chip" :class="groupPlatformBadgeClass(group)" :title="groupChipTitle(group)">
                      <span class="truncate">{{ group.name }}</span>
                    </span>
                    <code class="text-[11px] text-gray-400">#{{ group.id }}</code>
                  </td>
                  <td class="rate-cell">
                    <span class="model-rate-chip">{{ formatRate(group.rate_multiplier) }}</span>
                  </td>
                  <td>
                    <!--
                      价格用卡片而不是裸数字，样式与列表视图、配置页同一套（见 .detail-price-grid）。
                      刻意不渲染划线原价：每一行都是一个分组，而「1 倍率原价」是按模型算的、与行无关，
                      逐行重复同一个数字只是噪音 —— 这一行打了多少折，左边的倍率胶囊已经说清楚了。
                    -->
                    <div class="detail-price-grid">
                      <div
                        v-for="slot in detailPriceSlotsFor(group)"
                        :key="slot.key"
                        :class="['price-box', slot.toneClass]"
                      >
                        <span>{{ slot.label }}</span>
                        <strong>{{ formatPriceOrZero(slot.value) }}</strong>
                        <small v-if="slot.unit">{{ slot.unit }}</small>
                      </div>
                    </div>
                  </td>
                </tr>
              </tbody>
            </table>
          </div>
          <div v-else class="rounded-lg border border-dashed border-gray-200 px-3 py-4 text-sm text-gray-500 dark:border-dark-700 dark:text-gray-400">
            {{ t('admin.modelSquare.groupPricingEmpty') }}
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
            <!--
              分组名同样跟随平台配色：这个弹窗列的是同一个模型下的全部分组，
              名字若在这里是黑的、在列表里是平台色，同一行数据就有两种读法。
              tooltip 带上平台名 —— 颜色不能是唯一的信息载体。
            -->
            <div
              class="break-words text-sm font-medium"
              :class="groupPlatformTextClass(group)"
              :title="groupPlatformLabel(group)"
              data-test="group-dialog-name"
            >{{ group.name }}</div>
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
import { groupPlatformBadgeClass, groupPlatformLabel, groupPlatformTextClass } from '../model-square/groupPlatformStyle'

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
type SortOption = 'name' | 'price-asc' | 'price-desc'

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
const sortBy = ref<SortOption>('name')
const groupDialogModel = ref<ModelSquareModel | null>(null)
const detailModel = ref<ModelSquareModel | null>(null)
const copiedModelId = ref('')
let copiedTimer: ReturnType<typeof setTimeout> | undefined

const defaultPriceDescriptors: PriceDescriptor[] = [
  { key: 'input_price', label: '\u8f93\u5165', unit: '$/\u767e\u4e07 tokens', toneClass: 'price-box-teal' },
  { key: 'output_price', label: '\u8f93\u51fa', unit: '$/\u767e\u4e07 tokens', toneClass: 'price-box-orange' },
  { key: 'cache_read_price', label: '\u7f13\u5b58\u8bfb\u53d6', unit: '', toneClass: 'price-box-blue' },
  { key: 'cache_write_price', label: '\u7f13\u5b58\u5199\u5165', unit: '', toneClass: 'price-box-violet' },
]

// \u6309\u5f20\uff08\u56fe/\u5f20\uff09\u8ba1\u8d39\u4ef7\u683c\u4f4d\uff1a\u6a21\u578b\u5e7f\u573a\u914d\u7f6e\u91cc\u914d\u4e86 per_request_price \u7684\u6a21\u578b\u8d70\u8fd9\u4e2a\u3002
// \u4e0e token \u4ef7\u683c\u4e92\u65a5\uff08\u89c1 modelPriceSlots\uff09\u2014\u2014\u56fe/\u5f20\u6a21\u578b\u53ea\u5c55\u793a\u8fd9\u4e00\u5f20\u5361\uff0c\u4e0d\u6446 token \u4f4d\u3002
const perRequestDescriptor: PriceDescriptor = { key: 'per_request_price', label: '\u6309\u5f20', unit: '$/\u5f20', toneClass: 'price-box-teal' }

const priceDescriptors: PriceDescriptor[] = [
  ...defaultPriceDescriptors,
  { key: 'cache_write_1h_price', label: '\u7f13\u5b58\u5199\u5165 1h', unit: '$/\u767e\u4e07 tokens', toneClass: 'price-box-violet' },
  { key: 'input_price_priority', label: '\u4f18\u5148\u7ea7\u8f93\u5165', unit: '$/\u767e\u4e07 tokens', toneClass: 'price-box-neutral' },
  { key: 'output_price_priority', label: '\u4f18\u5148\u7ea7\u8f93\u51fa', unit: '$/\u767e\u4e07 tokens', toneClass: 'price-box-neutral' },
  { key: 'cache_write_price_priority', label: '\u4f18\u5148\u7ea7\u7f13\u5b58\u5199\u5165', unit: '$/\u767e\u4e07 tokens', toneClass: 'price-box-violet' },
  { key: 'cache_read_price_priority', label: '\u4f18\u5148\u7ea7\u7f13\u5b58\u8bfb\u53d6', unit: '$/\u767e\u4e07 tokens', toneClass: 'price-box-blue' },
  { key: 'image_input_price', label: '\u56fe\u50cf\u8f93\u5165', unit: '$/\u767e\u4e07 tokens', toneClass: 'price-box-neutral' },
  { key: 'image_output_price', label: '\u56fe\u50cf\u8f93\u51fa', unit: '$/\u767e\u4e07 tokens', toneClass: 'price-box-neutral' },
  perRequestDescriptor,
]
const emptyDescription = '\u5c1a\u672a\u914d\u7f6e\u6a21\u578b\u5e7f\u573a\u5e73\u53f0\u6216\u6a21\u578b\uff0c\u8bf7\u5148\u5728\u201c\u6a21\u578b\u5e7f\u573a\u914d\u7f6e\u201d\u4e2d\u7ef4\u62a4\u5c55\u793a\u76ee\u5f55\u3002'

const payload = computed(() => result.value?.payload?.data || result.value?.payload || {})
const models = computed<ModelSquareModel[]>(() => Array.isArray(payload.value.models) ? payload.value.models : [])
const groups = computed<ModelSquareGroup[]>(() => Array.isArray(payload.value.groups) ? payload.value.groups : [])
const groupById = computed(() => new Map(groups.value.map(group => [String(group.id), group])))
/*
  当前筛选中的分组对象。

  筛选生效时卡片上的分组名、倍率、价格都要跟着它走：这个页面的价格本来就是按分组倍率
  折算的，管理员筛了分组 B、卡片却标着分组 A 的倍率，那么「看到的分组」和「算出来的
  价格」就是两回事，比不改还容易误导。

  只取 groups 里能查得到的：分组被删、或下拉里是一个本页加载不出来的 ID 时，
  回退到「倍率最低的分组」那套老逻辑，而不是显示一个空名字。
*/
const activeFilterGroup = computed<ModelSquareGroup | null>(() =>
  groupFilter.value ? groupById.value.get(groupFilter.value) || null : null
)
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
const sortOptions = computed<SelectOption[]>(() => [
  { value: 'name', label: t('admin.modelSquare.sortName') },
  { value: 'price-asc', label: t('admin.modelSquare.sortPriceAsc') },
  { value: 'price-desc', label: t('admin.modelSquare.sortPriceDesc') },
])
const groupDialogGroups = computed(() => groupDialogModel.value ? modelGroups(groupDialogModel.value) : [])
const groupDialogTitle = computed(() => {
  const id = groupDialogModel.value?.id || t('admin.modelSquare.unnamedModel')
  return t('admin.modelSquare.groupDialogTitle', { id })
})
const detailGroups = computed(() => detailModel.value ? modelDetailGroups(detailModel.value) : [])
const detailRate = computed(() => detailModel.value ? modelEffectiveRate(detailModel.value) : 1)
/*
  detailPriceColumns 随「价格列改成卡片」一并删除：卡片自带标签，列头不再需要四个价格名。
  「只取四个语义价格位、不被优先级/图片/按请求价格顶掉」这条约束现在由 modelPriceSlots 承担
  （它固定遍历 defaultPriceDescriptors），不必在这里再留一份。
*/
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
const hasActiveFilters = computed(() => Boolean(
  searchQuery.value.trim() || providerFilter.value || groupFilter.value
))
// 排序键：名称用模型 ID；价格用卡片展示的输入价（已按模型有效倍率换算）。
// 未配置价格的模型不参与价格比较，始终排在末尾。
function modelSortPrice(model: ModelSquareModel): number | null {
  // 必须带上当前倍率：卡片上的价格已经按它折算，排序若还用原价，
  // 顺序会和眼睛看到的数字对不上（筛了分组后尤其明显）。
  const value = modelPriceValue(model, 'input_price', modelEffectiveRate(model))
  return typeof value === 'number' && Number.isFinite(value) ? value : null
}
function compareModels(left: ModelSquareModel, right: ModelSquareModel): number {
  const byId = (left.id || '').localeCompare(right.id || '')
  if (sortBy.value === 'name') return byId

  const leftPrice = modelSortPrice(left)
  const rightPrice = modelSortPrice(right)
  if (leftPrice == null && rightPrice == null) return byId
  if (leftPrice == null) return 1
  if (rightPrice == null) return -1
  if (leftPrice !== rightPrice) {
    return sortBy.value === 'price-asc' ? leftPrice - rightPrice : rightPrice - leftPrice
  }
  return byId
}
// 可用模型优先，其余按当前排序键；网格与列表共用同一顺序，避免两种视图结果不一致。
const sortedModels = computed(() => {
  return [...filteredModels.value].sort((a, b) => {
    if (isAvailable(a) !== isAvailable(b)) return isAvailable(a) ? -1 : 1
    return compareModels(a, b)
  })
})
const providerSections = computed<ModelSquareProviderSection[]>(() => {
  const sections = new Map<string, ModelSquareModel[]>()
  for (const model of sortedModels.value) {
    const provider = model.provider || ''
    const list = sections.get(provider) || []
    list.push(model)
    sections.set(provider, list)
  }

  return Array.from(sections.entries())
    .map(([provider, sectionModels]) => ({
      provider,
      models: sectionModels,
      lowestRate: Math.min(...sectionModels.map(primaryGroupRate))
    }))
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

function clearFilters() {
  searchQuery.value = ''
  providerFilter.value = ''
  groupFilter.value = ''
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

/*
  卡片上代表这个模型的分组：默认取倍率最低的那个（管理员最关心的就是最低价），
  筛选生效时改取被筛的那个 —— 否则卡片上的分组与倍率跟筛选条件对不上。

  必须确认模型真的绑了它才生效：filteredModels 保证了列表里的模型都绑着，
  但详情弹窗、复制等路径拿到的模型不受过滤约束，直接取会显示出一个它根本没绑的分组。
*/
function primaryGroup(model: ModelSquareModel): ModelSquareGroup | undefined {
  const filtered = activeFilterGroup.value
  if (filtered && (model.group_ids || []).some(id => String(id) === String(filtered.id))) {
    return filtered
  }
  return modelGroups(model)[0]
}

/*
  分组名跟随分组自己的平台配色，取色与配置页共用一处（groupPlatformStyle）：
  同一个分组在配置页、卡片、列表、详情弹窗里必须是同一种颜色 —— 取色一旦分叉，
  管理员会以为它们是两个不同的分组。

  平台色直接取 group.platform。展示页的数据由配置接口产出，那里的 platform 已经是
  「有效平台」（平台覆盖已解析完），所以不必再传 platformOverrides。

  它和卡片上的平台标签不是一回事，两者本来就允许不同：分组归属是手动绑定的，
  一个 openai 模型完全可以绑在 gemini 的分组上，那时两种颜色正是要传达的信息。
*/
function groupChipTitle(group: ModelSquareGroup) {
  // 颜色不能是唯一的信息载体：色觉障碍下 gemini 蓝 / zhipu 靛几乎分不出来，
  // tooltip 里带上平台名，色块才不至于退化成纯装饰。
  return `${group.name} · ${groupPlatformLabel(group)}`
}

function primaryGroupBadgeClass(model: ModelSquareModel) {
  const group = primaryGroup(model)
  return group ? groupPlatformBadgeClass(group) : ''
}

function primaryGroupTitle(model: ModelSquareModel) {
  const group = primaryGroup(model)
  return group ? groupChipTitle(group) : ''
}

function modelGroupOverflowCount(model: ModelSquareModel) {
  return Math.max(0, modelGroups(model).length - 1)
}

function groupRate(group?: ModelSquareGroup) {
  const rate = Number(group?.rate_multiplier)
  return Number.isFinite(rate) ? rate : Number.POSITIVE_INFINITY
}

/*
  价格折算的基准倍率。**必须不随筛选变化**：价格先归一化到这个固定基准，再乘上
  「当前展示的倍率」，筛选时价格才会跟着动；基准若也跟着筛选变，分子分母一起变，
  卡片上的价格就永远等于原价，看起来就像没生效。

  取模型自带的倍率；没有就取「最低倍率分组」—— 也就是不筛选时卡片代表的那个。
*/
function modelBaseRate(model: ModelSquareModel): number {
  const own = Number(model.rate_multiplier)
  if (Number.isFinite(own)) return own

  const fallback = groupRate(modelGroups(model)[0])
  return Number.isFinite(fallback) ? fallback : 1
}

/*
  卡片当前展示的倍率：筛选生效时取被筛分组的，否则回退到基准。

  被筛分组优先于「模型自带的 rate_multiplier」：管理员筛了分组 B 就是想看 B 下的表现，
  卡片标着 B 的名字却显示模型自己的倍率，倍率与分组就对不上 —— 这正是先前
  「分组名变了、倍率没变」的成因。
*/
function modelEffectiveRate(model: ModelSquareModel): number {
  const filtered = activeFilterGroup.value
  if (filtered && (model.group_ids || []).some(id => String(id) === String(filtered.id))) {
    const rate = groupRate(filtered)
    if (Number.isFinite(rate)) return rate
  }
  return modelBaseRate(model)
}

function modelPriceValue(model: ModelSquareModel, field: PriceField, multiplier?: number) {
  const value = model[field]
  if (value == null || value === '') return undefined

  const price = Number(value)
  if (!Number.isFinite(price)) return undefined
  if (multiplier == null) return price

  // 基准必须固定：见 modelBaseRate 的注释。用 modelEffectiveRate 会让分子分母同变，
  // 价格恒等于原价，筛选看起来就没效果。
  const baseRate = modelBaseRate(model)
  if (!Number.isFinite(baseRate) || baseRate === 0) return price === 0 ? 0 : undefined
  return (price / baseRate) * multiplier
}

function modelDisplayName(model: ModelSquareModel) {
  const displayName = model.display_name?.trim()
  if (displayName && model.id && displayName !== model.id) return `${displayName} (${model.id})`
  return displayName || model.id || t('admin.modelSquare.unnamedModel')
}

// 图/张计费模型：配了 per_request_price 就按这个类型展示。它没有 token 价，
// token 位会回退成官方参考价，摆出来会被读成「按 token 计费」——所以要单独识别。
function modelIsPerImage(model: ModelSquareModel): boolean {
  const value = model.per_request_price
  return value != null && value !== '' && Number.isFinite(Number(value))
}

function modelPriceSlots(model: ModelSquareModel, multiplier?: number): ModelPriceSlot[] {
  // 图/张模型只展示一张「按张」卡，不摆 token 价格位。
  // 按张是固定单价，不乘分组倍率 —— 这里刻意不传 multiplier，原样展示；也没有划线原价。
  if (modelIsPerImage(model)) {
    const value = modelPriceValue(model, 'per_request_price')
    return [{
      ...perRequestDescriptor,
      value,
      originalValue: undefined,
      toneClass: value == null ? 'price-box-unset' : perRequestDescriptor.toneClass,
    }]
  }
  // 固定展示输入、输出、缓存读取、缓存写入四个价格位，不被优先级/图片等额外价格顶替。
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
  // 图/张模型只列按张价：它的 token 位是官方参考回退价，列出来会与「按张计费」自相矛盾。
  const descriptors = modelIsPerImage(model) ? [perRequestDescriptor] : priceDescriptors
  return descriptors.flatMap(descriptor => {
    const value = modelPriceValue(model, descriptor.key)
    const unit = descriptor.unit ? ` ${descriptor.unit}` : ''
    return value == null ? [] : [`${descriptor.label}: ${formatPrice(value)}${unit}`]
  })
}

function formatPriceOrZero(value?: number | string) {
  return value == null || value === '' ? '$0' : formatPrice(value)
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
  // 已按分组过滤时不再打开分组弹窗：当前就锁定在这一个分组，弹窗（切换分组归属视角）没有意义。
  if (activeFilterGroup.value) return
  groupDialogModel.value = model
}

function closeGroupDialog() {
  groupDialogModel.value = null
}

function openModelDetails(model: ModelSquareModel) {
  detailModel.value = model
}

function closeModelDetails() {
  detailModel.value = null
}

// 详情表格按分组各算一遍价格：分组倍率 × 模型基础价，一次列全，用户不必逐个切换。
function detailPriceSlotsFor(group: ModelSquareGroup): ModelPriceSlot[] {
  if (!detailModel.value) return []
  return modelPriceSlots(detailModel.value, groupRate(group))
}

// 卡片倍率 chip 取模型有效倍率，这里高亮对应分组，让用户能把卡片上的价格对回具体分组。
function isCurrentRateGroup(group: ModelSquareGroup): boolean {
  if (!detailModel.value) return false
  return groupRate(group) === detailRate.value
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

/* 只在筛选生效时出现，用品牌色与静态统计 pill 区分，提示当前看到的是子集。 */
.result-count {
  @apply inline-flex h-7 items-center rounded-md px-2.5 font-mono text-xs font-semibold;
  background: color-mix(in srgb, var(--ms-brand) 10%, var(--ms-panel));
  color: var(--ms-brand-strong);
  border: 1px solid color-mix(in srgb, var(--ms-brand) 30%, transparent);
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
  color: var(--ms-provider, var(--ms-brand));
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
  color: var(--ms-provider, var(--ms-brand));
}

.provider-inline {
  color: var(--ms-provider, var(--ms-brand));
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

/*
  列表视图与详情弹窗的价格卡片组。

  与卡片视图共用 .price-box 的整套视觉语言（色调边框、标签、单位、划线原价），
  只压掉卡片视图的固定高度与内边距 —— 列表的价值是「一屏比对多个模型」，
  照搬 min-h-[4.6rem] 会让行高翻倍，那还不如直接切回卡片视图。
  四格横排而不是 2×2，同理：两行会把行高再抬一倍。

  原先这里是四个只显示数字的 .price-cell 药丸，既没有单位也没有划线原价，
  四列并排看不出主次 —— 改成卡片后标签随价格走，不必再依赖列头。

  两处共用同一组排布规则：卡片视图、列表视图、详情弹窗三个地方的价格必须长得一样，
  管理员来回切的时候不能换一副样子。
*/
.list-price-grid,
.detail-price-grid {
  @apply grid grid-cols-4 gap-2;
}

/*
  弹窗宽度是列表价格列的两倍多，卡片若跟着铺满会被拉到 160px 上下，比配置页那张（83px）宽一倍 ——
  「一样的卡片」首先得是一样大。356px = 83 × 4 + 间距 8 × 3，正好是配置页那一组卡片的宽度。

  刻意不用 width: max-content 让它自己缩：那是按每一行自己的内容算宽度，分组名长的行会把整组
  顶宽，于是第二行的「输出」卡与第一行错开 —— 纵向比对各分组的价格正是这张表存在的理由。
  余下的空间留在价格列右侧，行尾留白在表格里看不出来。
*/
.detail-price-grid {
  max-width: 356px;
}

.list-price-grid .price-box,
.detail-price-grid .price-box {
  @apply min-h-0 rounded-md px-2 py-1.5;
}

/* 紧凑档比卡片视图小一档，字号各降一档，但要保住「标签 < 数字」的层级差。 */
.list-price-grid .price-box strong,
.detail-price-grid .price-box strong {
  @apply text-[13px];
}

.list-price-grid .price-box small,
.detail-price-grid .price-box small {
  @apply text-[10px];
}

.list-price-grid .price-original {
  @apply text-[10px];
}

.model-card-footer {
  @apply mt-auto flex flex-wrap items-end justify-between gap-3 pt-4;
}

/*
  分组胶囊。这里只给形状与边框宽度，背景/文字/边框色由 groupPlatformBadgeClass 按
  分组自己的平台给出（与配置页同一套取色）。在这里写死颜色会盖掉平台色。
*/
.primary-group-chip {
  @apply inline-flex min-w-0 max-w-[68%] items-center gap-1 rounded-md border px-2 py-1 text-xs font-medium transition;
}

/*
  hover 把这枚胶囊加深一档，而不是换成另一种颜色：分组已经有平台身份，
  换色会被读成「换了个分组」。currentColor 就是平台文字色，所以这套 hover
  对每个平台都成立，不必逐个平台写一遍。
*/
.primary-group-chip:not(:disabled):hover {
  border-color: currentColor;
  background: color-mix(in srgb, currentColor 16%, transparent);
}

/* 已按分组过滤时胶囊不可点：去掉手型与 hover 反馈，避免看起来还能点开弹窗。 */
.primary-group-chip:disabled {
  @apply cursor-default;
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

/* 详情弹窗的分组价格表：一次列全所有分组，并高亮卡片倍率对应的那一行。 */
.group-price-table-wrap {
  @apply overflow-x-auto rounded-lg border;
  border-color: var(--ms-line);
}

.group-price-table {
  @apply w-full border-collapse text-sm;
}

.group-price-table th {
  @apply whitespace-nowrap px-3 py-2 text-left text-xs font-semibold;
  background: var(--ms-panel-2);
  color: var(--ms-muted);
  border-bottom: 1px solid var(--ms-line);
}

/*
  三列一律左对齐。原先「分组左、倍率与价格右」是为了让裸数字上下对齐；现在倍率是胶囊、
  价格是卡片，都是自带边框的块，右对齐会把它们推离行的起点，各行卡片的左边缘也会参差。
  纵向比对改由卡片位置保证（见 .detail-price-grid），不再依赖文本对齐。
*/

.group-price-table td {
  @apply px-3 py-2.5 align-middle;
  border-bottom: 1px solid color-mix(in srgb, var(--ms-line) 60%, transparent);
  color: var(--ms-text);
}

.group-price-table tbody tr:last-child td {
  border-bottom: 0;
}

.group-price-table tbody tr:hover td {
  background: var(--ms-panel-2);
}

/* 卡片倍率 chip 取自模型的有效倍率，这里同步高亮，让卡片价格能对回具体行。
   12% 与改动前的分组按钮激活态一致，保证浅色下也能和 hover 的灰底区分开。 */
.group-price-table tbody tr.active td {
  background: color-mix(in srgb, var(--ms-amber) 12%, var(--ms-panel));
}

.group-price-table tbody tr.active .group-cell {
  box-shadow: inset 3px 0 0 var(--ms-amber);
}

.group-cell {
  @apply min-w-[9rem] max-w-[14rem];
}

/* 倍率列只剩一枚胶囊：字重、颜色、底色都由 .model-rate-chip 承担，这里不再重复。 */
.rate-cell {
  @apply whitespace-nowrap;
}

.group-overflow {
  @apply shrink-0 rounded px-1 font-mono text-[10px];
  background: var(--ms-panel-3);
  color: var(--ms-muted);
}

/*
  分组胶囊：与 .primary-group-chip 同一处理 —— 只给形状与边框宽度，
  背景/文字/边框色由 groupPlatformBadgeClass 按分组自己的平台给出。
*/
.group-chip {
  @apply inline-flex max-w-full items-center gap-1 rounded border px-2 py-1 text-xs;
}

/* 手型与 hover 反馈只挂在按钮上：详情弹窗里的分组名不可点，用 span 复用同一枚胶囊，
   挂上这两样会让它看起来能点。 */
button.group-chip {
  @apply cursor-pointer;
}

/* 已按分组过滤时不可点，理由同 .primary-group-chip:disabled。 */
button.group-chip:disabled {
  @apply cursor-default;
}

/* 与卡片视图的胶囊同一套 hover：把当前这枚胶囊加深，不换色。 */
button.group-chip:not(:disabled):hover {
  border-color: currentColor;
  background: color-mix(in srgb, currentColor 18%, transparent);
}

/*
  倍率留在琥珀色，不跟着分组名走平台色：全页「琥珀 = 倍率/折扣」这条约定
  （卡片上的 .model-rate-chip 同样是琥珀）不该在胶囊内部断掉，
  而且列表里一串胶囊的倍率同色才扫得动。
*/
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

<template>
  <AppLayout class="model-square-root">
    <TablePageLayout>
      <template #filters>
        <div class="flex flex-wrap items-center gap-3">
          <!--
            顶部统计从原来的灰色小 pill 提升为 KPI 卡组：三个数（模型 / 可用 / 分组）
            各带图标与语义色顶条，页面顶部才有「头」——原先只有 h-11 的灰边框 pill，
            视觉重量几乎为零，整页看起来没有起点。
            形态对齐同模块 PlatformOverridesView 的 po-kpi（图标块 + 标签 + 数值 + 脚注），
            但配色走本页 --ms-* 主题变量，深色下自动跟随。
          -->
          <div class="ms-kpi-grid">
            <div class="ms-kpi ms-kpi-models">
              <span class="ms-kpi-icon"><Icon name="grid" size="md" aria-hidden="true" /></span>
              <span class="ms-kpi-body">
                <span class="ms-kpi-label">{{ t('admin.modelSquare.modelCount') }}</span>
                <strong class="ms-kpi-value">{{ models.length }}</strong>
                <span class="ms-kpi-foot">配置进展示目录的模型</span>
              </span>
            </div>
            <div class="ms-kpi ms-kpi-available">
              <span class="ms-kpi-icon"><Icon name="checkCircle" size="md" aria-hidden="true" /></span>
              <span class="ms-kpi-body">
                <span class="ms-kpi-label">{{ t('admin.modelSquare.availableCount') }}</span>
                <strong class="ms-kpi-value">{{ availableCount }}</strong>
                <span class="ms-kpi-foot">已绑定分组，能被筛到</span>
              </span>
            </div>
            <div class="ms-kpi ms-kpi-groups">
              <span class="ms-kpi-icon"><Icon name="users" size="md" aria-hidden="true" /></span>
              <span class="ms-kpi-body">
                <span class="ms-kpi-label">{{ t('admin.modelSquare.groupCount') }}</span>
                <strong class="ms-kpi-value">{{ groups.length }}</strong>
                <span class="ms-kpi-foot">参与展示的分组</span>
              </span>
            </div>
            <span v-if="hasActiveFilters" class="result-count">
              {{ t('admin.modelSquare.filteredCount', { filtered: sortedModels.length, total: models.length }) }}
            </span>
          </div>

          <div class="ml-auto flex flex-wrap items-center justify-end gap-2">
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
        <!--
          平台筛选从顶部下拉移到左侧目录卡片。

          下拉的问题是「看不见有多少个平台、各自几个模型」—— 要挨个展开才知道有什么，
          而且筛完平台之后顶部那一行看不出还剩哪些可选项。目录把全部平台平铺出来，
          每个带自己的平台色圆点与模型数，当前选中项直接高亮，一眼就能横向比较。

          目录是 aside，其余状态块（加载 / 报错 / 空态 / 网格 / 列表）是它的兄弟节点，
          靠 .ms-shell > :not(.ms-catalog) 统一撑满剩余宽度 —— 这样不必把它们再包一层。
        -->
        <div class="ms-shell">
          <aside class="ms-catalog" :aria-label="t('admin.modelSquare.catalogTitle')">
            <div class="ms-catalog-head">
              <span class="ms-catalog-title">{{ t('admin.modelSquare.catalogTitle') }}</span>
            </div>
            <nav class="ms-catalog-list">
              <button
                type="button"
                class="ms-catalog-item ms-catalog-item-all"
                :class="{ active: !providerFilter }"
                :aria-pressed="!providerFilter"
                @click="providerFilter = ''"
              >
                <span class="ms-catalog-dot"></span>
                <span class="ms-catalog-main">
                  <span class="ms-catalog-name">{{ t('admin.modelSquare.allProviders') }}</span>
                  <span class="ms-catalog-sub">{{ catalogSubtitle(catalogStats) }}</span>
                </span>
                <span class="ms-catalog-count">{{ catalogTotal }}</span>
              </button>
              <button
                v-for="item in providerCatalog"
                :key="item.provider"
                type="button"
                class="ms-catalog-item"
                :class="{ active: providerFilter === item.provider }"
                :aria-pressed="providerFilter === item.provider"
                :style="providerAccent(item.provider)"
                :title="providerLabel(item.provider)"
                @click="toggleProviderFilter(item.provider)"
              >
                <span class="ms-catalog-dot"></span>
                <span class="ms-catalog-main">
                  <span class="ms-catalog-name">{{ providerLabel(item.provider) }}</span>
                  <span class="ms-catalog-sub">{{ catalogSubtitle(item) }}</span>
                </span>
                <span class="ms-catalog-count">{{ item.count }}</span>
              </button>
            </nav>
          </aside>

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
                    <!-- 已按分组过滤时：卡片价已是该分组价，详情里也只有这一个分组，重复，隐藏。 -->
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
import { computed, onMounted, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { modelSquareAPI } from '@/api/modelSquare'
import { listLLMMonitorGroupPlatformOverrides } from '@/api/admin/modelMonitor'
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
// 模型监控里「监控显示」关掉的分组 ID：详情/分组弹窗的分组列表按它过滤，只留打开的。
const monitorHiddenGroupIds = ref<Set<string>>(new Set())
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
/*
  左侧平台目录的数据。

  catalogTotal 用 models 的长度而不是「各平台数量之和」：平台为空的模型（provider 缺失）
  不会出现在 providers 里，但它们在「全部」下确实是可见的，总数必须把它们算进去，
  否则「全部」上的数字会比实际看到的卡片少。

  数量按「模型广场配置里配了什么」算，与卡片区一致，不受其它筛选影响 —— 目录本身
  就是用来横向比较平台规模的，跟着分组筛选一起变会让人以为平台模型变少了。
*/
const catalogTotal = computed(() => models.value.length)
/*
  目录卡片副标题的两项：该范围内去重后的分组数、以及最低倍率。

  倍率取 modelBaseRate 而不是 primaryGroupRate —— 后者跟着分组筛选走，
  而目录刻意不随筛选变化（见上面的注释），用它会让「筛一下目录上的数字就变」。
*/
function catalogStatsOf(list: ModelSquareModel[]) {
  const groupIds = new Set<string>()
  let lowestRate = Number.POSITIVE_INFINITY
  for (const model of list) {
    for (const id of model.group_ids || []) groupIds.add(String(id))
    lowestRate = Math.min(lowestRate, modelBaseRate(model))
  }
  return { groupCount: groupIds.size, lowestRate: Number.isFinite(lowestRate) ? lowestRate : 0 }
}
// 「全部提供商」那一条的副信息：全局分组数与全局最低倍率（与顶部统计卡的口径一致）。
const catalogStats = computed(() => catalogStatsOf(models.value))
const providerCatalog = computed(() => {
  const grouped = new Map<string, ModelSquareModel[]>()
  for (const model of models.value) {
    const provider = model.provider || ''
    const list = grouped.get(provider) || []
    list.push(model)
    grouped.set(provider, list)
  }
  return providers.value.map(provider => {
    const list = grouped.get(provider) || []
    return { provider, count: list.length, ...catalogStatsOf(list) }
  })
})
function catalogSubtitle(stats: { groupCount: number; lowestRate: number }) {
  return t('admin.modelSquare.catalogSummary', {
    count: stats.groupCount,
    rate: formatRate(stats.lowestRate),
  })
}
// 再点一次当前项 = 取消筛选：目录里高亮项本身就是「已选」的视觉锚点，
// 回到全部最自然的手势就是点它，不必回头去找顶上那条「全部」。
function toggleProviderFilter(provider: string) {
  providerFilter.value = providerFilter.value === provider ? '' : provider
}
/*
  筛了平台之后，该平台下的模型实际绑定过的分组 ID 集合。
  null = 未筛平台（不做限制），与「筛了但一个分组都没绑上」区分开：
  后者应该是只剩“全部”的空下拉，而不是退回全量分组。
*/
const providerScopedGroupIds = computed<Set<string> | null>(() => {
  if (!providerFilter.value) return null
  const ids = new Set<string>()
  for (const model of models.value) {
    if (model.provider !== providerFilter.value) continue
    for (const id of model.group_ids || []) ids.add(String(id))
  }
  return ids
})
// 分组筛选下拉选项（含“全部”占位项）。同样按「监控显示」过滤：监控里关掉的分组不进下拉。
// 平台筛选生效时只列该平台用得到的分组 —— 否则能选到一个本平台压根没有的分组，选完列表直接空掉。
const groupFilterOptions = computed<SelectOption[]>(() => {
  const scoped = providerScopedGroupIds.value
  return [
    { value: '', label: t('admin.modelSquare.allGroups') },
    ...filterMonitorVisibleGroups(groups.value)
      .filter(group => !scoped || scoped.has(String(group.id)))
      .map(group => ({ value: String(group.id), label: group.name })),
  ]
})
/*
  平台一变，原先选中的分组可能已不在候选里，必须主动收回。
  下拉里没有这一项时用户连“取消选择”的入口都没有，只能看着空列表猜原因。
  只挂在平台值上：挂在候选列表上会把用户刚改的分组一起回退掉，表现为“点了没反应”。
*/
watch(providerFilter, () => {
  if (groupFilter.value && !groupFilterOptions.value.some(option => option.value === groupFilter.value)) {
    groupFilter.value = ''
  }
})
const availableCount = computed(() => models.value.filter(isAvailable).length)
const sortOptions = computed<SelectOption[]>(() => [
  { value: 'name', label: t('admin.modelSquare.sortName') },
  { value: 'price-asc', label: t('admin.modelSquare.sortPriceAsc') },
  { value: 'price-desc', label: t('admin.modelSquare.sortPriceDesc') },
])
// 弹窗里的分组列表只保留「监控显示」为开的分组（模型监控平台配置页里的开关）。
function filterMonitorVisibleGroups(list: ModelSquareGroup[]): ModelSquareGroup[] {
  if (monitorHiddenGroupIds.value.size === 0) return list
  return list.filter(group => !monitorHiddenGroupIds.value.has(String(group.id)))
}

const groupDialogGroups = computed(() => groupDialogModel.value ? filterMonitorVisibleGroups(modelGroups(groupDialogModel.value)) : [])
const groupDialogTitle = computed(() => {
  const id = groupDialogModel.value?.id || t('admin.modelSquare.unnamedModel')
  return t('admin.modelSquare.groupDialogTitle', { id })
})
/*
  详情弹窗的分组只取模型自己绑定的那些，与卡片胶囊、列表胶囊、分组弹窗同源（modelGroups）。

  这里曾经按「模型平台」兜底展开：分组平台与模型平台一致、或无平台/composite，就一并列出来。
  那套口径是分组归属还「由渠道反推」时代的遗留 —— 现在归属的唯一来源是配置页手动绑定的
  group_ids（见 buildConfiguredModelSquareResult）。继续兜底会把模型根本没绑的分组摆进详情，
  管理员读到的是「这个模型在这个分组下可用」，而它在那个分组里其实用不了。
*/
const detailGroups = computed(() => detailModel.value ? filterMonitorVisibleGroups(modelGroups(detailModel.value)) : [])
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
    // 监控可见性单独兜底：拉不到就当作全部可见（空集合），不因它拖垮整个列表。
    const [square, overrides] = await Promise.all([
      modelSquareAPI.get(),
      listLLMMonitorGroupPlatformOverrides().catch(() => []),
    ])
    result.value = square
    monitorHiddenGroupIds.value = new Set(
      overrides.filter(item => !item.show_in_monitor).map(item => String(item.id))
    )
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
/*
  顶部 KPI 卡组：三个数各带图标块与语义色顶条。

  图标块刻意用「淡染底 + 混色图标」而不是实色底 + 白图标 —— 深色主题下
  --ms-green 这类语义色本身是亮色（#34d399），实色底配白图标对比度不到 2:1。
  混色写法与页面既有的 .chip-tinted 同源，深浅两套主题共用一条规则。
*/
.ms-kpi-grid {
  @apply flex min-w-0 flex-1 flex-wrap items-stretch gap-3;
}

.ms-kpi {
  @apply relative flex min-w-[11rem] flex-1 items-start gap-3 overflow-hidden rounded-xl border px-3.5 py-3;
  border-color: var(--ms-line);
  /* 卡面自带极淡的同色渐变 —— 纯白卡 + 彩色顶条会像「贴上去的」，融不进同一张卡 */
  background: linear-gradient(
    180deg,
    color-mix(in srgb, var(--ms-kpi-accent) 11%, var(--ms-panel)) 0%,
    var(--ms-panel) 64%
  );
  transition: transform 0.15s ease, border-color 0.15s ease, box-shadow 0.15s ease;
}

/*
  顶条：三张卡各用自己的语义色，扫一眼就能把「总量 / 可用 / 分组」区分开。

  光晕（box-shadow 用同色半透明）是「灵动」的主要来源：亮色细条本身只有 3px，
  发光之后才会在浅色背景上浮起来，而不是一条压死的色带。
*/
.ms-kpi::before {
  content: '';
  position: absolute;
  top: 0;
  left: 0;
  right: 0;
  height: 3px;
  background: linear-gradient(90deg, var(--ms-kpi-accent), color-mix(in srgb, var(--ms-kpi-accent) 35%, transparent));
  box-shadow: 0 0 14px color-mix(in srgb, var(--ms-kpi-accent) 65%, transparent);
}

.ms-kpi:hover {
  transform: translateY(-2px);
  border-color: color-mix(in srgb, var(--ms-kpi-accent) 55%, var(--ms-line));
  box-shadow: 0 12px 26px -12px color-mix(in srgb, var(--ms-kpi-accent) 75%, transparent);
}

/*
  实色渐变图标块：淡染底太「平」。
  图标块是这三张卡上唯一能承载饱和色的地方，色彩活力由它承担。
*/
.ms-kpi-icon {
  @apply grid h-9 w-9 shrink-0 place-items-center rounded-lg text-white;
  background: linear-gradient(
    135deg,
    color-mix(in srgb, var(--ms-kpi-accent) 70%, black),
    var(--ms-kpi-accent)
  );
  box-shadow: 0 6px 14px -6px color-mix(in srgb, var(--ms-kpi-accent) 90%, transparent);
}

/* 深色主题下亮色本身就是高亮色（#2dd4bf 这类），白图标会糊在底上 ⇒ 翻成深色图标 */
.dark .ms-kpi-icon {
  color: #0f172a;
  background: linear-gradient(
    135deg,
    var(--ms-kpi-accent),
    color-mix(in srgb, var(--ms-kpi-accent) 72%, white)
  );
}

.ms-kpi-body {
  @apply flex min-w-0 flex-col;
}

.ms-kpi-label {
  @apply text-xs font-medium;
  color: var(--ms-muted);
}

.ms-kpi-value {
  @apply font-mono text-xl font-bold leading-tight;
  color: var(--ms-text);
}

.ms-kpi-foot {
  @apply mt-0.5 truncate text-[11px];
  color: var(--ms-dim);
}

/*
  三张卡的色相刻意拉开（蓝 205° / 绿 160° / 紫 262°）。
  原先「模型数」用品牌青绿（174°）与「可用数」的绿（160°）只差 14°，
  并排看几乎同色，三张卡退化成一整片 —— 色相距离才是这三张卡能被区分开的原因。
*/
.ms-kpi-models {
  --ms-kpi-accent: var(--ms-blue-bright);
}

.ms-kpi-available {
  --ms-kpi-accent: var(--ms-green-bright);
}

.ms-kpi-groups {
  --ms-kpi-accent: var(--ms-violet-bright);
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

/*
  左侧平台目录 + 右侧内容的两栏骨架。

  目录是 #table 槽里唯一新增的结构，其余状态块（加载 / 报错 / 空态 / 网格 / 列表）
  原样搬进来当它的兄弟节点 —— 用 :not(.ms-catalog) 统一给它们 min-w-0 + flex-1，
  省掉再包一层 div。

  min-w-0 是必须的：列表视图的表格写死了 min-w-[1100px]，没有它，flex 项的最小宽度
  会被内容顶开，整个 shell 被撑宽、外层出现横向滚动条，而不是让表格自己滚。
*/
.ms-shell {
  @apply flex min-h-0 flex-1 gap-3 p-3;
}

.ms-shell > :not(.ms-catalog) {
  @apply min-w-0 flex-1;
}

/*
  目录容器本身**不画卡片** —— 它只是一列排布容器，真正的卡片是下面每一条。
  定宽、高度按内容走（不拉满），窄屏时改成整宽横排。

  ⚠️ self-start 不能省：不加就会拉伸到 shell 全高，一列卡片会被拉成一根贴边的柱子。
  代价是高度不再由容器兜底，所以要自己用 max-height 兜住平台特别多的情形
  （超出部分交给列表内部滚动）。
  ⚠️ 配套给 .ms-catalog-list 补 min-h-0：列向 flex 项默认 min-height: auto，
  不允许收缩到内容以下，只写 overflow-y-auto 是不会滚的。
*/
.ms-catalog {
  @apply flex w-64 shrink-0 self-start flex-col;
  max-height: 100%;
}

/*
  头部不放「总数」徽标：下面的「全部提供商」那一条已经在显示同一个数字，
  两个相同的数并排只会让人多问一句「这两个 12 是同一件事吗」。
  也不画分隔线 —— 下面每一条自带边框，再压一条线会读成「标题也有个框」。
  px-2 与 .ms-catalog-list 的左右内边距对齐，标题才和卡片的边框落在同一条竖线上。
*/
.ms-catalog-head {
  @apply flex items-center px-2 pb-1.5;
}

.ms-catalog-title {
  @apply text-sm font-semibold tracking-wide;
  color: var(--ms-muted);
}

/*
  px-2 / py-1.5 不只是留白：这个容器 overflow-y-auto（计算后 overflow-x 也是 auto），
  会把子元素超出内容盒的部分裁掉。卡片的投影横向半径约 5px，padding 不够就会被切出一条直边。
*/
.ms-catalog-list {
  @apply min-h-0 flex-1 space-y-2 overflow-y-auto px-2 py-1.5;
}

/*
  每个平台各自一张小卡片：受光面 + 边框 + 三层投影 + 顶沿内高光（--ms-sheen），
  与模型卡、价格卡同一套语言，只是缩一档。

  阴影用「平台色混 transparent」而不是分主题写 rgba：混色自带明暗适配，
  浅色下是淡淡的彩影、深色下自动变重，不必再为 .dark 定一组 alpha
  （价格卡的 --pb-float 用的也是这个办法）。
  --ms-provider 由 providerAccent 内联注入；「全部」那条没有 → 回落到品牌色。
*/
.ms-catalog-item {
  --cat-accent: var(--ms-provider, var(--ms-brand));
  --cat-float:
    0 1px 2px -1px color-mix(in srgb, var(--cat-accent) 45%, transparent),
    0 3px 7px -2px color-mix(in srgb, var(--cat-accent) 50%, transparent),
    inset 0 1px 0 var(--ms-sheen);
  @apply flex w-full items-center gap-2.5 rounded-xl border px-3 py-3 text-left text-[15px] transition;
  color: var(--ms-text);
  border-color: color-mix(in srgb, var(--cat-accent) 30%, var(--ms-line));
  background: linear-gradient(
    180deg,
    color-mix(in srgb, var(--cat-accent) 8%, var(--ms-panel)) 0%,
    var(--ms-panel) 100%
  );
  box-shadow: var(--cat-float);
}

/*
  悬停 = 抬起 1px + 投影拉长 + 底色加深，与模型卡 hover 同一套动作。

  悬停与选中都取条目自己的平台色。选中比悬停再重一档，保证「已选」始终比「鼠标划过」更明确。
  两条规则特异性相同（都是两个 class / 一个 class + 伪类），靠源码顺序定胜负 ——
  .active 必须写在 :hover 之后，否则划过选中项时底色会从 20% 退回 14%，看着像「选中掉了」。
*/
.ms-catalog-item:hover {
  @apply -translate-y-px;
  border-color: color-mix(in srgb, var(--cat-accent) 55%, var(--ms-line));
  background: linear-gradient(
    180deg,
    color-mix(in srgb, var(--cat-accent) 14%, var(--ms-panel)) 0%,
    color-mix(in srgb, var(--cat-accent) 5%, var(--ms-panel)) 100%
  );
  box-shadow:
    0 2px 4px -2px color-mix(in srgb, var(--cat-accent) 50%, transparent),
    0 9px 18px -7px color-mix(in srgb, var(--cat-accent) 55%, transparent),
    inset 0 1px 0 var(--ms-sheen);
}

.ms-catalog-item.active {
  @apply font-semibold;
  border-color: color-mix(in srgb, var(--cat-accent) 70%, var(--ms-line));
  background: linear-gradient(
    180deg,
    color-mix(in srgb, var(--cat-accent) 20%, var(--ms-panel)) 0%,
    color-mix(in srgb, var(--cat-accent) 9%, var(--ms-panel)) 100%
  );
  box-shadow:
    var(--cat-float),
    0 0 0 1px color-mix(in srgb, var(--cat-accent) 40%, transparent);
}

.ms-catalog-dot {
  @apply h-2.5 w-2.5 shrink-0 rounded-full;
  background: var(--ms-provider, var(--ms-brand));
  box-shadow: 0 0 0 3px color-mix(in srgb, var(--ms-provider, var(--ms-brand)) 16%, transparent);
}

/* 名称与副标题包成一列，右侧的计数徽标才能贴着卡片右边、不被长平台名挤走。 */
.ms-catalog-main {
  @apply min-w-0 flex-1;
}

.ms-catalog-name {
  @apply block truncate;
}

/* 副标题：分组数与最低倍率。压到 muted 色，避免和平台名抢视线。 */
.ms-catalog-sub {
  @apply mt-0.5 block truncate text-xs;
  color: var(--ms-muted);
}

.ms-catalog-count {
  @apply shrink-0 rounded px-2 py-0.5 font-mono text-xs font-semibold tabular-nums;
  background: var(--ms-panel-3);
  color: color-mix(in srgb, var(--ms-provider, var(--ms-brand)) 72%, var(--ms-text));
}

/*
  窄屏（与 TablePageLayout 的 mobile-mode 同一条断点）改成上下排、目录横排成一串 chip。

  竖排目录在 400px 宽的手机上会吃掉 256px，内容只剩几十像素 —— 而目录此时又删不掉：
  顶部那个平台下拉已经拿掉，它是唯一的平台筛选入口，必须保持可用。
*/
@media (max-width: 1023px) {
  .ms-shell {
    /* 外层此时是 h-auto，flex-1 的 0 基准高度会让整块塌掉，退回 auto */
    @apply flex-none flex-col;
  }

  /*
    self-stretch 抵消基础规则的 self-start：这里已经是列向排列，
    留着 flex-start 会让卡片按内容缩宽、不再整宽。max-h-none 同理 ——
    外层高度是 auto，百分比上限没有参照，留着不如去掉。
  */
  .ms-catalog {
    @apply w-full self-stretch max-h-none;
  }

  .ms-catalog-list {
    /* space-y-0 抵消基础规则的 space-y-2：横排时行间距会变成上边距，卡片会错位 */
    @apply flex gap-2 space-y-0 overflow-x-auto px-2 py-1.5;
  }

  .ms-catalog-item {
    @apply w-auto shrink-0;
  }

  /* 横排时名称按内容宽走：基础规则的 flex-1 基准是 0，shrink-to-fit 下会被压没 */
  .ms-catalog-item .ms-catalog-main {
    @apply flex-none;
  }

  .ms-catalog-item .ms-catalog-name {
    @apply max-w-[8rem];
  }

  /* chip 里塞第二行会把每片撑得又高又宽，副标题只留在大屏的竖排列表里看 */
  .ms-catalog-sub {
    @apply hidden;
  }
}

.model-square-board {
  @apply h-full space-y-5 overflow-y-auto p-4;
}

.provider-section {
  @apply relative space-y-3 pl-3;
}

/*
  分区左脊：加宽到 4px 并向下渐隐。

  它和 KPI 顶条、卡片顶条是同一个平台色（都取 --ms-provider）——
  三层同源，扫一眼就知道「这一片属于哪个平台」；先前只有一条 3px 实色细线，
  在白色背景上几乎读不出平台差异。
*/
.provider-section::before {
  content: '';
  position: absolute;
  left: 0;
  top: 0.4rem;
  bottom: 0.4rem;
  width: 4px;
  border-radius: 999px;
  background: linear-gradient(
    180deg,
    var(--ms-provider, var(--ms-brand)),
    color-mix(in srgb, var(--ms-provider, var(--ms-brand)) 35%, transparent)
  );
  /* 光晕替代原先的 opacity: 0.9 —— 柔化交给同色发光，色条本身保持饱和 */
  box-shadow: 0 0 14px color-mix(in srgb, var(--ms-provider, var(--ms-brand)) 55%, transparent);
}

.provider-section-header {
  @apply flex items-center justify-between gap-3 px-1;
}

.provider-title {
  @apply flex min-w-0 items-center gap-2 text-base font-bold tracking-tight;
  /* 与 --ms-text 混色：平台主色在浅色下偏浅、深色下偏暗，直接当文字色两头都不稳 */
  color: color-mix(in srgb, var(--ms-provider, var(--ms-brand)) 78%, var(--ms-text));
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

/* 计数徽标改用分区自己的平台色，与左脊、标题同色，不再是一枚灰 chip */
.provider-count {
  @apply grid h-7 min-w-7 place-items-center rounded-md px-2 font-mono text-xs font-semibold;
  background: color-mix(in srgb, var(--ms-provider, var(--ms-brand)) 14%, var(--ms-panel));
  color: color-mix(in srgb, var(--ms-provider, var(--ms-brand)) 70%, var(--ms-text));
  box-shadow: inset 0 0 0 1px color-mix(in srgb, var(--ms-provider, var(--ms-brand)) 26%, transparent);
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

/*
  卡片顶条：与分区左脊、KPI 顶条同一个平台色。

  卡片本体原先只有顶部一个小圆点带平台色，网格一铺开就读不出「哪几张是同一个平台」。
  顶条把平台归属变成卡片自身的结构特征，不必读文字。
  圆角单独写是因为卡片没有 overflow: hidden，直角色条会在圆角处露头。
*/
.model-card::before {
  content: '';
  position: absolute;
  top: 0;
  left: 0;
  right: 0;
  height: 3px;
  border-radius: 0.5rem 0.5rem 0 0;
  background: linear-gradient(
    90deg,
    var(--ms-provider, var(--ms-brand)),
    color-mix(in srgb, var(--ms-provider, var(--ms-brand)) 30%, transparent)
  );
  /* 与 KPI 顶条同一套光晕语言：细条靠发光浮起来，而不是压成一条色带 */
  box-shadow: 0 0 12px color-mix(in srgb, var(--ms-provider, var(--ms-brand)) 60%, transparent);
}

/*
  hover 光晕改用自己的平台色（原先固定用品牌色 --ms-brand）。
  网格里同时铺着几个平台，hover 哪张就该是哪张的颜色 —— 固定品牌色会让
  「平台色骨架」在唯一需要反馈的瞬间断掉。

  阴影换成 --ms-shadow-hover（与静止态同结构、整体加强），而不是另写一串：
  静止态已经有三层了，hover 只加强其中一层的话，抬起量会被吃掉。
*/
.model-card:hover,
.model-card:focus-visible {
  @apply -translate-y-0.5 outline-none;
  border-color: color-mix(in srgb, var(--ms-provider, var(--ms-brand)) 70%, var(--ms-line));
  box-shadow:
    var(--ms-shadow-hover),
    0 0 0 1px color-mix(in srgb, var(--ms-provider, var(--ms-brand)) 50%, transparent),
    0 0 28px color-mix(in srgb, var(--ms-provider, var(--ms-brand)) 35%, transparent);
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

/*
  四个价格位。每个色位两个变量：

    --pb-accent  装饰亮色（边框 / 底色 / 光晕）
    --pb-ink     同色系的「落墨色」—— 浅色主题下取深饱和档，深色主题下取浅亮档

  这样「往 ink 方向混」在任何主题下都等价于「提高对比度」，标签与数值共用一条公式，
  不必为深色主题单独写一套文字色。

  先前的边框取「文字档」深色（#0f766e 一类），3px 细线压在白底上几乎不发色；
  数值更是直接用深色档 —— teal 那档是 #115e59，接近黑，四张卡看起来只有橙的一张有颜色。

  ink 取 800/900 档而不是 700 档：700 档（#0f766e / #ea580c 一类）混出来仍偏亮，
  橙色尤其明显 —— 只有它一张「跳」出来，四张卡的重量对不齐。深一档后四色重量接近，
  同时因为数值里掺了 22% 亮色，色相还在，不至于退回「一张有色 + 三张黑」。
*/
.price-box-teal {
  --pb-accent: var(--ms-brand-bright);
  --pb-ink: #115e59;
}

.price-box-orange {
  --pb-accent: var(--ms-orange-bright);
  --pb-ink: #9a3412;
}

.price-box-blue {
  --pb-accent: var(--ms-blue-bright);
  --pb-ink: #1e40af;
}

.price-box-violet {
  --pb-accent: var(--ms-violet-bright);
  --pb-ink: #5b21b6;
}

/*
  深色主题下 ink 反过来要比 accent 更亮，公式不动，「往 ink 混」依然是提高对比度。

  同样要「再深一档」：上一版的 ink（#99f6e4 这类 200 档）在深底上发白，
  往 accent 靠 60% 后仍比 accent 亮，既压住了亮度，又保住了「数值比标签更抢眼」的层级。
  不直接取 accent —— 那样标签与数值会完全同色，层级只剩字号和字重。
*/
.dark .price-box-teal {
  --pb-ink: color-mix(in srgb, var(--ms-brand-bright) 60%, #99f6e4);
}

.dark .price-box-orange {
  --pb-ink: color-mix(in srgb, var(--ms-orange-bright) 60%, #fdba74);
}

.dark .price-box-blue {
  --pb-ink: color-mix(in srgb, var(--ms-blue-bright) 60%, #93c5fd);
}

.dark .price-box-violet {
  --pb-ink: color-mix(in srgb, var(--ms-violet-bright) 60%, #c4b5fd);
}

.price-box-teal,
.price-box-orange,
.price-box-blue,
.price-box-violet {
  border-color: color-mix(in srgb, var(--pb-accent) 55%, var(--ms-line));
  background: linear-gradient(
    180deg,
    color-mix(in srgb, var(--pb-accent) 20%, var(--ms-panel)) 0%,
    color-mix(in srgb, var(--pb-accent) 8%, var(--ms-panel)) 100%
  );
  /*
    悬浮感 = 三层阴影 + 一条顶沿高光，缺一层就塌：

      ① 接触阴影：几乎不模糊的一小条，给出卡片与底面的贴合点。少了它，卡片会像贴纸；
      ② 近距投影：卡片的实际厚度；
      ③ 远距环境光遮蔽：大扩散 + 低浓度 —— 悬浮感主要来自这一层；
      ④ 顶沿内高光（--ms-sheen）：给卡面一条受光的上沿，卡片才有厚度而不是一块色。

    原先只有一条 `0 2px 10px -5px` 的色晕，负 spread 把它压得紧贴边框，
    既没有贴合点也没有环境遮蔽 —— 看着就是「平铺在底面上的一块色」。

    整串挂在 --pb-float 上：紧凑档（列表 / 详情弹窗）只换这一个变量就能整体缩一档，
    不必把四条阴影抄第二遍。--pb-accent 未定义时（未配置的 price-box-unset）整串失效，
    卡片自动保持平面 —— 那是「缺失」状态，本来就不该浮起来。
  */
  --pb-float:
    0 1px 2px -1px color-mix(in srgb, var(--pb-accent) 55%, transparent),
    0 4px 9px -3px color-mix(in srgb, var(--pb-accent) 60%, transparent),
    0 12px 22px -10px color-mix(in srgb, var(--pb-accent) 70%, transparent),
    inset 0 1px 0 var(--ms-sheen);
  box-shadow: var(--pb-float);
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

/*
  标签（输入 / 输出 / 缓存读取 / 缓存写入）：亮色档 58% + ink 42%。

  比数值浅一档 —— 白底上越深越抢眼，标签必须退到数值后面；
  深色主题下 ink 更亮，混色结果也随之变浅，方向自动一致。
  纯亮色档（#14b8a6）在浅底上只有 2.3:1，12px 小字糊成一片，所以不取满。
*/
.price-box-teal span,
.price-box-orange span,
.price-box-blue span,
.price-box-violet span {
  color: color-mix(in srgb, var(--pb-accent) 58%, var(--pb-ink));
}

.price-box strong {
  @apply mt-1 block font-mono text-sm font-bold;
  color: var(--ms-text);
}

/*
  数值：亮色档 22% + ink 78%。

  用 ink 当主体而不是直接用 ink，是为了让四张卡各自带着自己的色相；
  22% 是「看得出颜色」与「沉得住」的交点 —— 浅色主题下四色实测 5.8 / 5.8 / 7.1 / 7.6 : 1，
  比上一版（4.5 / 4.5 / 5.9 / 6.4）更深，也离 AA 门槛更远。
*/
.price-box-teal strong,
.price-box-orange strong,
.price-box-blue strong,
.price-box-violet strong {
  color: color-mix(in srgb, var(--pb-accent) 22%, var(--pb-ink));
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
  /*
    紧凑档只有 83px 宽，上面那套（12px 偏移 + 22px 扩散）是按卡片视图 130px 宽的卡片定的 ——
    照搬进 8px 间距的表格里，阴影会糊到邻格上，反而显脏。按卡片尺寸缩一档，三层结构不变。
  */
  --pb-float:
    0 1px 2px -1px color-mix(in srgb, var(--pb-accent) 50%, transparent),
    0 3px 6px -2px color-mix(in srgb, var(--pb-accent) 55%, transparent),
    0 7px 14px -7px color-mix(in srgb, var(--pb-accent) 65%, transparent),
    inset 0 1px 0 var(--ms-sheen);
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
  /*
    模型卡静止态的悬浮阴影，与价格卡同一套三层分工（接触 / 厚度 / 环境遮蔽），
    尺度按卡片尺寸放大 —— 卡片宽 320px 起，用价格卡那组 12px 偏移会像「没浮起来」。

    原先是单条 `0 1px 3px rgba(15,23,42,.06)`，几乎等于没有，整张卡平铺在底面上。
    hover 必须同结构整体加强（--ms-shadow-hover），否则静止态一厚，hover 就没有「抬起」的余量了。
  */
  --ms-shadow:
    0 1px 2px -1px rgba(15, 23, 42, 0.1),
    0 8px 14px -6px rgba(15, 23, 42, 0.13),
    0 18px 32px -16px rgba(15, 23, 42, 0.18);
  --ms-shadow-hover:
    0 2px 4px -2px rgba(15, 23, 42, 0.14),
    0 14px 28px -10px rgba(15, 23, 42, 0.2),
    0 32px 56px -22px rgba(15, 23, 42, 0.26);
  /*
    顶沿高光：卡片上沿一条 1px 的受光边。没有它，卡面就是一块纯色，
    有阴影也只会像「剪下来贴在底面上」；有它才有厚度。
    浅色下用高不透明度白（卡面本身接近白，低了看不见），深色下只留一点点。
  */
  --ms-sheen: rgba(255, 255, 255, 0.8);

  /*
    装饰亮色：同一个语义色需要两档 —— 做文字/边框要深（保对比度），
    做顶条/图标块/光晕要亮（保活力）。

    先前的顶条与图标块直接拿文字档的深色（--ms-brand 是 #0f766e），
    3px 细条压在白底上几乎不发色，这是整页看着「闷」的主因。
  */
  --ms-brand-bright: #14b8a6;
  --ms-green-bright: #10b981;
  --ms-blue-bright: #3b82f6;
  --ms-violet-bright: #8b5cf6;
  --ms-orange-bright: #f97316;
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
  /*
    深底上黑色阴影的可视度天然比白底低（背景本身就是暗的），所以 alpha 要拉到 0.4~0.55
    才看得出卡片下方压着一片更暗的区域；照搬浅色那组 0.08~0.14 会完全消失。
  */
  --ms-shadow:
    0 1px 2px -1px rgba(0, 0, 0, 0.55),
    0 8px 14px -6px rgba(0, 0, 0, 0.48),
    0 18px 32px -16px rgba(0, 0, 0, 0.55);
  --ms-shadow-hover:
    0 2px 4px -2px rgba(0, 0, 0, 0.6),
    0 14px 28px -10px rgba(0, 0, 0, 0.55),
    0 32px 56px -22px rgba(0, 0, 0, 0.6);
  /* 深底上卡面本来就是暗的，高光只留一条几乎看不出的亮边，多了会发灰 */
  --ms-sheen: rgba(255, 255, 255, 0.08);

  /* 深色下再提一档：深色卡面上的装饰色要比浅色主题更亮才「跳」得出来 */
  --ms-brand-bright: #2dd4bf;
  --ms-green-bright: #34d399;
  --ms-blue-bright: #60a5fa;
  --ms-violet-bright: #a78bfa;
  --ms-orange-bright: #fb923c;
}
</style>

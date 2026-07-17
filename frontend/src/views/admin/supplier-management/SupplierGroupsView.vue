<template>
  <SupplierModuleLayout>
    <header class="sp-page-head">
      <div>
        <div class="sp-eyebrow">Local Supplier Groups</div>
        <h1>分组管理</h1>
        <p class="sp-subtitle">展示供应商同步后的可用分组、关联账号数和最近同步状态。</p>
      </div>
      <div class="sp-controls">
        <Select v-model="providerID" class="sp-search" :options="providerOptions" :searchable="false" />
        <Select v-model="activeFilter" class="sp-search" :options="activeFilterOptions" :searchable="false" />
        <Input v-model="search" class="sp-search" placeholder="搜索分组或上游 Key" />
        <button class="sp-button small" type="button" :disabled="loading || !canResetFilters" @click="resetGroupFilters">重置筛选</button>
        <button class="sp-button" type="button" :disabled="loading" @click="loadGroups">刷新</button>
      </div>
    </header>

    <div v-if="error" class="sp-alert sp-error-line">{{ error }}</div>

    <div class="sp-console-shell">
      <div class="sp-summary-grid" aria-label="分组汇总">
        <StatCard
          title="筛选结果"
          :value="groupSummary.group_count"
          :icon="FilteredGroupsIcon"
          icon-variant="primary"
        />
        <StatCard
          title="关联账号"
          :value="groupSummary.account_count"
          :icon="AccountsIcon"
          icon-variant="success"
        />
        <StatCard
          title="已关联分组"
          :value="groupSummary.linked_group_count"
          :icon="LinkedGroupsIcon"
          icon-variant="success"
          :change="linkedGroupRate"
          change-type="neutral"
        />
        <StatCard
          title="未关联分组"
          :value="groupSummary.unlinked_group_count"
          :icon="UnlinkedGroupsIcon"
          :icon-variant="groupSummary.unlinked_group_count > 0 ? 'warning' : 'primary'"
          :change="unlinkedGroupRate"
          change-type="neutral"
        />
      </div>

      <div class="sp-grid-2">
        <div class="sp-console-panel sp-panel">
          <header class="sp-panel-head">
            <div class="sp-panel-title"><span class="sp-section-index">01</span><div><h2>本地分组表</h2><span>共 {{ total }} 条同步记录</span></div></div>
          </header>
          <div class="sp-table-shell">
            <DataTable
              :columns="groupColumns"
              :data="items"
              :loading="loading"
              row-key="id"
              clickable-rows
              @row-click="selected = $event"
            >
              <template #cell-provider_name="{ row: group }">
                <div class="sp-entity">{{ group.provider_name }}</div>
                <div class="sp-sub">ID {{ group.provider_id }}</div>
              </template>
              <template #cell-name="{ row: group }">
                <div class="sp-entity">{{ group.name || '未命名分组' }}</div>
                <div class="sp-sub">{{ group.upstream_group_key }}</div>
              </template>
              <template #cell-account_count="{ row: group }">
                <span class="sp-num">{{ group.account_count }}</span>
              </template>
              <template #cell-rate_multiplier="{ row: group }">
                <span class="sp-num">{{ group.rate_multiplier || 0 }}</span>
              </template>
              <template #cell-raw_status="{ row: group }">
                {{ group.raw_status || 'unknown' }}
              </template>
              <template #cell-active="{ row: group }">
                <span class="sp-status" :class="group.active ? 'good' : ''">{{ group.active ? '可用' : '失效记录' }}</span>
              </template>
              <template #cell-last_seen_at="{ row: group }">
                {{ formatTime(group.last_seen_at) }}
              </template>
              <template #empty>
                暂无本地分组数据，请先在供应商列表执行同步。
              </template>
            </DataTable>
          </div>
          <Pagination
            v-if="total > 0"
            class="sp-data-pagination"
            :page="page"
            :total="total"
            :page-size="pageSize"
            :show-page-size-selector="true"
            @update:page="handleGroupPageChange"
            @update:pageSize="handleGroupPageSizeChange"
          />
        </div>

        <div class="sp-console-panel sp-panel">
          <header class="sp-panel-head"><div class="sp-panel-title"><span class="sp-section-index">02</span><div><h2>当前页摘要</h2><span>本页 {{ items.length }} 条记录</span></div></div></header>
          <div class="sp-panel-body">
            <div class="sp-stat-list">
              <div class="sp-stat-box"><span>可用记录</span><b>{{ activeGroupCount }}</b></div>
              <div class="sp-stat-box"><span>失效记录</span><b>{{ inactiveGroupCount }}</b></div>
              <div class="sp-stat-box"><span>已关联分组</span><b>{{ groupsWithAccountsCount }}</b></div>
              <div class="sp-stat-box"><span>未关联分组</span><b>{{ emptyGroupCount }}</b></div>
              <div class="sp-stat-box"><span>未命名记录</span><b>{{ unnamedGroupCount }}</b></div>
              <div class="sp-stat-box"><span>当前页 / 匹配总数</span><b>{{ items.length }} / {{ total }}</b></div>
            </div>
          </div>
        </div>
      </div>
    </div>

    <SupplierDrawer :show="Boolean(selected)" :title="selected?.name || selected?.upstream_group_key || ''" eyebrow="GROUP DETAIL" @close="selected = null">
      <template v-if="selected">
        <div class="sp-detail-grid">
          <div class="sp-detail-cell"><span>供应商</span><b>{{ selected.provider_name }}</b></div>
          <div class="sp-detail-cell"><span>上游 Key</span><b>{{ selected.upstream_group_key }}</b></div>
          <div class="sp-detail-cell"><span>分组名称</span><b>{{ selected.name || '未命名分组' }}</b></div>
          <div class="sp-detail-cell"><span>倍率</span><b>{{ selected.rate_multiplier }}</b></div>
          <div class="sp-detail-cell"><span>关联账号数</span><b>{{ selected.account_count }}</b></div>
          <div class="sp-detail-cell"><span>上游状态</span><b>{{ selected.raw_status || 'unknown' }}</b></div>
          <div class="sp-detail-cell"><span>记录状态</span><b>{{ selected.active ? '可用' : '失效记录' }}</b></div>
          <div class="sp-detail-cell"><span>最近同步</span><b>{{ formatTime(selected.last_seen_at) }}</b></div>
        </div>
      </template>
    </SupplierDrawer>
  </SupplierModuleLayout>
</template>

<script setup lang="ts">
import { computed, h, nextTick, onMounted, ref, watch } from 'vue'
import { SupplierDrawer, SupplierModuleLayout } from '@/components/admin/supplier-management'
import DataTable from '@/components/common/DataTable.vue'
import Input from '@/components/common/Input.vue'
import Pagination from '@/components/common/Pagination.vue'
import Select, { type SelectOption } from '@/components/common/Select.vue'
import StatCard from '@/components/common/StatCard.vue'
import Icon from '@/components/icons/Icon.vue'
import supplierProvidersAPI, { type SupplierProvider } from '@/api/admin/supplierProviders'
import {
  listSupplierGroups,
  type SupplierProviderGroup,
  type SupplierProviderGroupSummary,
} from '@/api/admin/supplierProviderData'
import type { Column } from '@/components/common/types'

const EMPTY_GROUP_SUMMARY: SupplierProviderGroupSummary = {
  group_count: 0,
  account_count: 0,
  linked_group_count: 0,
  unlinked_group_count: 0,
}

const FilteredGroupsIcon = () => h(Icon, { name: 'filter', size: 'lg' })
const AccountsIcon = () => h(Icon, { name: 'users', size: 'lg' })
const LinkedGroupsIcon = () => h(Icon, { name: 'link', size: 'lg' })
const UnlinkedGroupsIcon = () => h(Icon, { name: 'exclamationCircle', size: 'lg' })

const providers = ref<SupplierProvider[]>([])
const items = ref<SupplierProviderGroup[]>([])
const groupSummary = ref<SupplierProviderGroupSummary>({ ...EMPTY_GROUP_SUMMARY })
const selected = ref<SupplierProviderGroup | null>(null)
const total = ref(0)
const loading = ref(false)
const error = ref('')
const page = ref(1)
const pageSize = ref(20)
const providerID = ref(0)
const activeFilter = ref('true')
const search = ref('')
let searchTimer: number | undefined
let suppressFilterWatch = false

const DEFAULT_PROVIDER_ID = 0
const DEFAULT_ACTIVE_FILTER = 'true'
const providerOptions = computed<SelectOption[]>(() => [
  { value: 0, label: '全部供应商' },
  ...providers.value.map(provider => ({ value: provider.id, label: provider.name })),
])
const activeFilterOptions: SelectOption[] = [
  { value: 'true', label: '仅可用' },
  { value: '', label: '全部状态' },
  { value: 'false', label: '失效记录' },
]
const groupColumns: Column[] = [
  { key: 'provider_name', label: '供应商', class: 'min-w-[150px]' },
  { key: 'active', label: '记录状态' },
  { key: 'account_count', label: '关联账号数' },
  { key: 'name', label: '上游分组', class: 'min-w-[180px]' },
  { key: 'rate_multiplier', label: '倍率' },
  { key: 'last_seen_at', label: '最近同步', class: 'min-w-[150px]' },
  { key: 'raw_status', label: '上游状态' },
]
const canResetFilters = computed(() =>
  providerID.value !== DEFAULT_PROVIDER_ID
  || activeFilter.value !== DEFAULT_ACTIVE_FILTER
  || search.value.trim() !== ''
)
const activeGroupCount = computed(() => items.value.filter(group => group.active).length)
const inactiveGroupCount = computed(() => items.value.filter(group => !group.active).length)
const groupsWithAccountsCount = computed(() => items.value.filter(group => group.account_count > 0).length)
const emptyGroupCount = computed(() => items.value.filter(group => group.account_count <= 0).length)
const unnamedGroupCount = computed(() => items.value.filter(group => !group.name?.trim()).length)
const linkedGroupRate = computed(() => percentage(groupSummary.value.linked_group_count, groupSummary.value.group_count))
const unlinkedGroupRate = computed(() => percentage(groupSummary.value.unlinked_group_count, groupSummary.value.group_count))

onMounted(async () => {
  await loadProviders()
  await loadGroups()
})

watch([providerID, activeFilter], () => {
  if (suppressFilterWatch) return
  page.value = 1
  void loadGroups()
})

watch(search, () => {
  if (suppressFilterWatch) return
  window.clearTimeout(searchTimer)
  searchTimer = window.setTimeout(() => {
    page.value = 1
    void loadGroups()
  }, 350)
})

function resetGroupFilters() {
  window.clearTimeout(searchTimer)
  suppressFilterWatch = true
  providerID.value = DEFAULT_PROVIDER_ID
  activeFilter.value = DEFAULT_ACTIVE_FILTER
  search.value = ''
  page.value = 1
  void nextTick(() => {
    suppressFilterWatch = false
    void loadGroups()
  })
}

function handleGroupPageChange(nextPage: number) {
  if (nextPage === page.value) return
  page.value = nextPage
  void loadGroups()
}

function handleGroupPageSizeChange(nextPageSize: number) {
  if (nextPageSize === pageSize.value) return
  pageSize.value = nextPageSize
  page.value = 1
  void loadGroups()
}

async function loadProviders() {
  const result = await supplierProvidersAPI.list({ page: 1, page_size: 200 })
  providers.value = result.items
}

async function loadGroups() {
  loading.value = true
  error.value = ''
  try {
    const result = await listSupplierGroups({
      provider_id: providerID.value || undefined,
      active: activeFilter.value === '' ? undefined : activeFilter.value === 'true',
      search: search.value.trim() || undefined,
      page: page.value,
      page_size: pageSize.value,
    })
    items.value = result.items
    total.value = result.total
    page.value = result.page
    pageSize.value = result.page_size
    groupSummary.value = result.summary
  } catch (err) {
    error.value = err instanceof Error ? err.message : '加载分组数据失败'
  } finally {
    loading.value = false
  }
}

function formatTime(value?: string): string {
  if (!value) return '—'
  const date = new Date(value)
  if (Number.isNaN(date.getTime())) return '—'
  return date.toLocaleString('zh-CN')
}

function percentage(value: number, totalValue: number): number {
  if (totalValue <= 0) return 0
  return Math.round((value / totalValue) * 100)
}
</script>

<style scoped>
.sp-console-shell {
  display: grid;
  gap: 1rem;
}

.sp-summary-grid {
  display: grid;
  grid-template-columns: repeat(4, minmax(0, 1fr));
  gap: 0.75rem;
}

.sp-summary-grid :deep(.stat-card) {
  position: relative;
  min-height: 6.25rem;
  align-items: center;
  overflow: hidden;
  padding: 1rem;
  border: 1px solid color-mix(in srgb, var(--sp-line) 82%, var(--sp-text));
  border-radius: 0.5rem;
  background: var(--sp-panel);
  box-shadow: 0 4px 14px rgba(15, 23, 42, 0.06);
  transition: border-color 160ms ease, box-shadow 160ms ease, transform 160ms ease;
  animation: sp-summary-enter 180ms ease-out both;
}

.sp-summary-grid :deep(.stat-card)::before {
  position: absolute;
  inset: 0 auto 0 0;
  width: 3px;
  background: var(--sp-cyan);
  content: '';
}

.sp-summary-grid :deep(.stat-card:nth-child(2))::before,
.sp-summary-grid :deep(.stat-card:nth-child(3))::before {
  background: var(--sp-green);
}

.sp-summary-grid :deep(.stat-card:nth-child(4))::before {
  background: var(--sp-amber);
}

.sp-summary-grid :deep(.stat-icon) {
  width: 2.75rem;
  height: 2.75rem;
  border: 1px solid currentColor;
  border-radius: 0.5rem;
  opacity: 0.92;
}

.sp-summary-grid :deep(.stat-label) {
  color: var(--sp-muted);
  font-size: 0.78rem;
  font-weight: 600;
  letter-spacing: 0;
}

.sp-summary-grid :deep(.stat-value) {
  color: var(--sp-text);
  font-size: 1.75rem;
  font-weight: 750;
  line-height: 1.1;
}

.sp-summary-grid :deep(.stat-trend) {
  padding: 0.15rem 0.4rem;
  border: 1px solid var(--sp-line);
  border-radius: 0.25rem;
  background: color-mix(in srgb, var(--sp-panel-2) 72%, transparent);
  color: var(--sp-muted);
  font-size: 0.75rem;
  font-weight: 600;
}

.sp-summary-grid :deep(.stat-card:nth-child(2)) {
  animation-delay: 30ms;
}

.sp-summary-grid :deep(.stat-card:nth-child(3)) {
  animation-delay: 60ms;
}

.sp-summary-grid :deep(.stat-card:nth-child(4)) {
  animation-delay: 90ms;
}

@media (hover: hover) {
  .sp-summary-grid :deep(.stat-card:hover) {
    border-color: color-mix(in srgb, var(--sp-cyan) 34%, var(--sp-line));
    box-shadow: 0 10px 24px rgba(15, 23, 42, 0.1);
    transform: translateY(-2px);
  }
}

.sp-console-panel {
  background:
    linear-gradient(180deg, rgba(15, 23, 42, 0.02), transparent 18%),
    var(--sp-panel);
}

.sp-table-shell {
  display: flex;
  flex-direction: column;
  min-height: 0;
  height: min(70vh, 760px);
  overflow: hidden;
}

.sp-table-shell :deep(.table-wrapper) {
  min-height: 0;
  flex: 1;
}

.sp-data-pagination {
  flex-shrink: 0;
}

@media (max-width: 1050px) {
  .sp-summary-grid {
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }
}

@media (max-width: 760px) {
  .sp-summary-grid {
    grid-template-columns: 1fr;
  }
}

@media (prefers-reduced-motion: reduce) {
  .sp-summary-grid :deep(.stat-card) {
    transition: none;
    animation: none;
  }
}

@keyframes sp-summary-enter {
  from {
    opacity: 0;
    transform: translateY(5px);
  }

  to {
    opacity: 1;
    transform: translateY(0);
  }
}
</style>

<template>
  <SupplierModuleLayout>
    <section class="sp-account-toolbar" aria-label="账号筛选与操作">
      <div class="sp-account-filter-fields">
        <Input
          v-model="search"
          class="sp-search sp-account-search"
          placeholder="搜索账号名称或上游 Key"
        />
        <Select
          v-model="providerID"
          class="sp-search sp-account-select"
          :options="providerOptions"
          :searchable="false"
        />
        <Select
          v-model="platformFilter"
          class="sp-search sp-account-select"
          :options="platformFilterOptions"
          :searchable="false"
        />
        <Select
          v-model="activeFilter"
          class="sp-search sp-account-select"
          :options="activeFilterOptions"
          :searchable="false"
        />
      </div>
      <button class="sp-button sp-account-refresh" type="button" :disabled="loading" @click="loadAccounts">
        {{ loading ? '刷新中…' : '刷新' }}
      </button>
    </section>

    <div v-if="error" class="sp-alert sp-error-line">{{ error }}</div>

    <section class="sp-panel sp-account-workbench">
      <header class="sp-panel-head sp-account-panel-head">
        <div class="sp-panel-title">
          <span class="sp-section-index">01</span>
          <div>
            <h2>本地账号表</h2>
            <span>当前筛选共 {{ total }} 条同步记录</span>
          </div>
        </div>
        <div class="sp-account-legend" aria-label="账号状态图例">
          <span><i class="good"></i>有效</span>
          <span><i class="bad"></i>已失效</span>
        </div>
      </header>

      <div class="sp-account-table-shell">
        <DataTable
          :columns="accountColumns"
          :data="items"
          :loading="loading"
          row-key="id"
          clickable-rows
          @row-click="selected = $event"
        >
          <template #cell-name="{ row: account }">
            <div class="sp-account-identity">
              <span class="sp-account-avatar" aria-hidden="true">{{ accountInitial(account) }}</span>
              <div class="sp-account-copy">
                <div class="sp-entity">{{ account.name || '未命名账号' }}</div>
                <div class="sp-sub sp-account-key" :title="account.upstream_account_key">
                  {{ account.upstream_account_key }}
                </div>
              </div>
            </div>
          </template>
          <template #cell-provider_name="{ row: account }">
            <div class="sp-provider-cell">
              <span class="sp-provider-dot" aria-hidden="true"></span>
              <div>
                <div class="sp-entity">{{ account.provider_name }}</div>
                <div class="sp-sub">供应商 #{{ account.provider_id }}</div>
              </div>
            </div>
          </template>
          <template #cell-group_name="{ row: account }">
            <span v-if="account.group_name || account.group_key" class="sp-account-group">
              {{ account.group_name || account.group_key }}
            </span>
            <span v-else class="sp-account-muted">未分组</span>
          </template>
          <template #cell-platform="{ row: account }">
            <span
              v-if="account.platform"
              :class="['inline-flex items-center rounded-md border px-2 py-0.5 text-[11px] font-semibold', platformBadgeClass(account.platform)]"
            >
              {{ platformLabel(account.platform) }}
            </span>
            <span v-else class="sp-account-muted">--</span>
          </template>
          <template #cell-rate_multiplier="{ row: account }">
            <span class="sp-account-rate">× {{ formatRate(account.rate_multiplier) }}</span>
          </template>
          <template #cell-raw_status="{ row: account }">
            <span class="sp-account-upstream-status">
              {{ account.raw_status || account.status || 'unknown' }}
            </span>
          </template>
          <template #cell-active="{ row: account }">
            <span class="sp-status" :class="account.active ? 'good' : 'bad'">
              <i class="sp-status-dot" aria-hidden="true"></i>
              {{ account.active ? '有效' : '已失效' }}
            </span>
          </template>
          <template #cell-last_seen_at="{ row: account }">
            <div class="sp-account-time">{{ formatTime(account.last_seen_at) }}</div>
          </template>
          <template #empty>
            <div class="sp-account-empty">
              <strong>暂无本地账号数据</strong>
              <span>请先在供应商列表执行同步，或调整当前筛选条件。</span>
            </div>
          </template>
        </DataTable>
      </div>

      <footer v-if="total > 0" class="sp-account-pagination">
        <div class="sp-account-page-size">
          <span>每页显示</span>
          <Select
            :model-value="pageSize"
            class="sp-account-page-size-select"
            :options="pageSizeOptions"
            :searchable="false"
            @update:model-value="handlePageSizeChange"
          />
          <span>条</span>
        </div>
        <Pagination
          class="sp-data-pagination"
          :page="page"
          :total="total"
          :page-size="pageSize"
          :show-page-size-selector="false"
          @update:page="handlePageChange"
        />
      </footer>
    </section>

    <SupplierDrawer
      :show="Boolean(selected)"
      :title="selected?.name || selected?.upstream_account_key || ''"
      eyebrow="ACCOUNT DETAIL"
      @close="selected = null"
    >
      <template v-if="selected">
        <div class="sp-detail-grid">
          <div class="sp-detail-cell"><span>供应商</span><b>{{ selected.provider_name }}</b></div>
          <div class="sp-detail-cell"><span>上游 Key</span><b>{{ selected.upstream_account_key }}</b></div>
          <div class="sp-detail-cell"><span>分组</span><b>{{ selected.group_name || selected.group_key || '未分组' }}</b></div>
          <div class="sp-detail-cell"><span>倍率</span><b>{{ selected.rate_multiplier }}</b></div>
          <div class="sp-detail-cell"><span>上游状态</span><b>{{ selected.raw_status || selected.status }}</b></div>
          <div class="sp-detail-cell"><span>本地状态</span><b>{{ selected.active ? '有效' : '已失效' }}</b></div>
          <div class="sp-detail-cell"><span>最近同步</span><b>{{ formatTime(selected.last_seen_at) }}</b></div>
          <div class="sp-detail-cell"><span>失效时间</span><b>{{ selected.inactive_at ? formatTime(selected.inactive_at) : '—' }}</b></div>
        </div>
      </template>
    </SupplierDrawer>
  </SupplierModuleLayout>
</template>

<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import { SupplierDrawer, SupplierModuleLayout } from '@/components/admin/supplier-management'
import DataTable from '@/components/common/DataTable.vue'
import Input from '@/components/common/Input.vue'
import Pagination from '@/components/common/Pagination.vue'
import Select, { type SelectOption } from '@/components/common/Select.vue'
import supplierProvidersAPI, { type SupplierProvider } from '@/api/admin/supplierProviders'
import { listSupplierAccounts, type SupplierProviderAccount } from '@/api/admin/supplierProviderData'
import type { Column } from '@/components/common/types'
import { platformBadgeClass, platformLabel } from '@/utils/platformColors'

const providers = ref<SupplierProvider[]>([])
const items = ref<SupplierProviderAccount[]>([])
const selected = ref<SupplierProviderAccount | null>(null)
const total = ref(0)
const loading = ref(false)
const error = ref('')
const page = ref(1)
const pageSize = ref(20)
const providerID = ref(0)
const platformFilter = ref('')
const activeFilter = ref('true')
const search = ref('')
let searchTimer: number | undefined

const providerOptions = computed<SelectOption[]>(() => [
  { value: 0, label: '全部供应商' },
  ...providers.value.map(provider => ({ value: provider.id, label: provider.name })),
])
const activeFilterOptions: SelectOption[] = [
  { value: 'true', label: '仅有效' },
  { value: '', label: '全部状态' },
  { value: 'false', label: '已失效' },
]
const platformFilterOptions: SelectOption[] = [
  { value: '', label: '全部平台' },
  { value: 'anthropic', label: 'Anthropic' },
  { value: 'openai', label: 'OpenAI' },
  { value: 'gemini', label: 'Gemini' },
  { value: 'antigravity', label: 'Antigravity' },
  { value: 'grok', label: 'Grok' },
]
const pageSizeOptions: SelectOption[] = [
  { value: 20, label: '20' },
  { value: 50, label: '50' },
  { value: 100, label: '100' },
]
const accountColumns: Column[] = [
  { key: 'name', label: '账号 / 上游 Key', class: 'min-w-[240px]' },
  { key: 'provider_name', label: '供应商', class: 'min-w-[170px]' },
  { key: 'group_name', label: '分组', class: 'min-w-[140px]' },
  { key: 'platform', label: '平台', class: 'min-w-[112px]' },
  { key: 'rate_multiplier', label: '倍率', class: 'min-w-[96px]' },
  { key: 'raw_status', label: '上游状态', class: 'min-w-[120px]' },
  { key: 'active', label: '本地状态', class: 'min-w-[112px]' },
  { key: 'last_seen_at', label: '最近同步', class: 'min-w-[180px]' },
]

onMounted(async () => {
  await loadProviders()
  await loadAccounts()
})

onBeforeUnmount(() => {
  window.clearTimeout(searchTimer)
})

watch([providerID, platformFilter, activeFilter], () => {
  resetPageAndLoad()
})

watch(search, () => {
  window.clearTimeout(searchTimer)
  searchTimer = window.setTimeout(resetPageAndLoad, 350)
})

async function loadProviders() {
  const result = await supplierProvidersAPI.list({ page: 1, page_size: 200 })
  providers.value = result.items
}

async function loadAccounts() {
  loading.value = true
  error.value = ''
  try {
    const result = await listSupplierAccounts({
      provider_id: providerID.value || undefined,
      platform: platformFilter.value || undefined,
      active: activeFilter.value === '' ? undefined : activeFilter.value === 'true',
      search: search.value.trim() || undefined,
      page: page.value,
      page_size: pageSize.value,
    })
    items.value = result.items
    total.value = result.total
    page.value = result.page
    pageSize.value = result.page_size
  } catch (err) {
    error.value = err instanceof Error ? err.message : '加载账号数据失败'
  } finally {
    loading.value = false
  }
}

function resetPageAndLoad() {
  page.value = 1
  void loadAccounts()
}

function handlePageChange(nextPage: number) {
  if (nextPage === page.value) return
  page.value = nextPage
  void loadAccounts()
}

function handlePageSizeChange(value: string | number | boolean | null) {
  if (value === null || typeof value === 'boolean') return
  const nextPageSize = Number(value)
  if (![20, 50, 100].includes(nextPageSize) || nextPageSize === pageSize.value) return
  pageSize.value = nextPageSize
  resetPageAndLoad()
}

function accountInitial(account: SupplierProviderAccount): string {
  const value = account.name?.trim() || account.upstream_account_key?.trim() || '?'
  return value.slice(0, 1).toUpperCase()
}

function formatRate(value?: number): string {
  return Number(value || 0).toFixed(2)
}

function formatTime(value?: string): string {
  if (!value) return '—'
  const date = new Date(value)
  if (Number.isNaN(date.getTime())) return '—'
  return date.toLocaleString('zh-CN', {
    year: 'numeric',
    month: '2-digit',
    day: '2-digit',
    hour: '2-digit',
    minute: '2-digit',
  })
}
</script>

<style scoped>
.sp-account-toolbar {
  display: flex;
  align-items: center;
  gap: 0.75rem;
  margin-bottom: 1rem;
  padding: 0.875rem;
  border: 1px solid var(--sp-line);
  border-radius: 0.75rem;
  background: var(--sp-panel);
  box-shadow: var(--sp-shadow);
}

.sp-account-filter-fields {
  display: grid;
  min-width: 0;
  flex: 1 1 auto;
  grid-template-columns: minmax(16rem, 1fr) minmax(10rem, 0.36fr) minmax(9rem, 0.3fr) minmax(9rem, 0.3fr);
  gap: 0.625rem;
}

.sp-account-toolbar .sp-search {
  width: 100%;
  min-width: 0;
}

.sp-account-refresh {
  min-width: 5.5rem;
  min-height: 2.625rem;
}

.sp-account-workbench {
  border-color: color-mix(in srgb, var(--sp-cyan) 14%, var(--sp-line));
}

.sp-account-panel-head {
  min-height: 4.5rem;
  background:
    linear-gradient(90deg, color-mix(in srgb, var(--sp-cyan) 5%, transparent), transparent 34%),
    var(--sp-panel);
}

.sp-account-legend {
  display: flex;
  align-items: center;
  gap: 1rem;
  color: var(--sp-muted);
  font-size: 0.75rem;
}

.sp-account-legend span {
  display: inline-flex;
  align-items: center;
  gap: 0.4rem;
}

.sp-account-legend i,
.sp-status-dot,
.sp-provider-dot {
  display: inline-block;
  width: 0.45rem;
  height: 0.45rem;
  flex: 0 0 auto;
  border-radius: 9999px;
}

.sp-account-legend i.good,
.sp-status.good .sp-status-dot {
  background: var(--sp-green);
  box-shadow: 0 0 0 3px color-mix(in srgb, var(--sp-green) 14%, transparent);
}

.sp-account-legend i.bad,
.sp-status.bad .sp-status-dot {
  background: var(--sp-red);
  box-shadow: 0 0 0 3px color-mix(in srgb, var(--sp-red) 12%, transparent);
}

.sp-account-table-shell {
  min-height: 23rem;
  overflow: hidden;
  background: var(--sp-panel);
}

.sp-account-table-shell :deep(.table-wrapper) {
  border: 0;
  border-radius: 0;
}

.sp-account-table-shell :deep(.table-header) {
  background: var(--sp-panel-2);
}

.sp-account-table-shell :deep(th) {
  height: 3.25rem;
  border-bottom-color: var(--sp-line);
  color: var(--sp-muted);
  font-size: 0.6875rem;
  letter-spacing: 0.08em;
}

.sp-account-table-shell :deep(td) {
  height: 4.5rem;
  border-color: var(--sp-soft);
}

.sp-account-table-shell :deep(tbody tr) {
  transition: background-color 140ms ease, box-shadow 140ms ease;
}

.sp-account-table-shell :deep(tbody tr:hover) {
  background: color-mix(in srgb, var(--sp-cyan) 5%, var(--sp-panel));
  box-shadow: inset 3px 0 0 color-mix(in srgb, var(--sp-cyan) 72%, transparent);
}

.sp-account-identity,
.sp-provider-cell {
  display: flex;
  min-width: 0;
  align-items: center;
  gap: 0.75rem;
}

.sp-account-avatar {
  display: inline-flex;
  width: 2.25rem;
  height: 2.25rem;
  flex: 0 0 auto;
  align-items: center;
  justify-content: center;
  border: 1px solid color-mix(in srgb, var(--sp-cyan) 24%, var(--sp-line));
  border-radius: 0.625rem;
  background: color-mix(in srgb, var(--sp-cyan) 8%, var(--sp-panel-2));
  color: var(--sp-cyan);
  font-size: 0.8125rem;
  font-weight: 800;
}

.sp-account-copy {
  min-width: 0;
}

.sp-account-key {
  max-width: 17rem;
  overflow: hidden;
  font-family: ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, monospace;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.sp-provider-dot {
  background: var(--sp-cyan);
  box-shadow: 0 0 0 4px color-mix(in srgb, var(--sp-cyan) 10%, transparent);
}

.sp-account-group,
.sp-account-upstream-status {
  display: inline-flex;
  align-items: center;
  padding: 0.25rem 0.55rem;
  border: 1px solid var(--sp-soft);
  border-radius: 0.4rem;
  background: var(--sp-panel-2);
  color: var(--sp-muted);
  font-size: 0.75rem;
}

.sp-account-upstream-status {
  font-family: ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, monospace;
  text-transform: lowercase;
}

.sp-account-rate {
  color: var(--sp-text);
  font-family: ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, monospace;
  font-size: 0.8125rem;
  font-weight: 700;
}

.sp-account-time,
.sp-account-muted {
  color: var(--sp-muted);
  font-size: 0.8125rem;
}

.sp-status {
  font-weight: 600;
}

.sp-account-empty {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 0.375rem;
}

.sp-account-empty strong {
  color: var(--sp-text);
  font-size: 0.9375rem;
}

.sp-account-empty span {
  color: var(--sp-muted);
  font-size: 0.8125rem;
}

.sp-account-pagination {
  display: flex;
  min-height: 4.25rem;
  align-items: center;
  justify-content: space-between;
  gap: 1rem;
  border-top: 1px solid var(--sp-line);
  background: var(--sp-panel);
}

.sp-account-page-size {
  display: flex;
  flex: 0 0 auto;
  align-items: center;
  gap: 0.5rem;
  padding-left: 1rem;
  color: var(--sp-muted);
  font-size: 0.8125rem;
}

.sp-account-page-size-select {
  width: 5.25rem;
}

.sp-account-page-size-select :deep(.select-trigger) {
  min-height: 2.25rem;
  padding-block: 0.375rem;
}

.sp-account-pagination :deep(.sp-data-pagination),
.sp-account-pagination :deep(> div) {
  border-top: 0;
  background: transparent;
}

@media (max-width: 900px) {
  .sp-account-toolbar {
    align-items: stretch;
    flex-direction: column;
  }

  .sp-account-filter-fields {
    grid-template-columns: 1fr 1fr;
  }

  .sp-account-search {
    grid-column: 1 / -1;
  }

  .sp-account-refresh {
    width: 100%;
  }

  .sp-account-pagination {
    align-items: stretch;
    flex-direction: column;
    gap: 0;
  }

  .sp-account-page-size {
    justify-content: center;
    padding: 0.875rem 1rem 0;
  }
}

@media (max-width: 760px) {
  .sp-account-toolbar {
    margin-bottom: 0.75rem;
    padding: 0.75rem;
  }

  .sp-account-panel-head {
    min-height: auto;
  }

  .sp-account-legend {
    width: 100%;
  }

  .sp-account-table-shell {
    min-height: 0;
    padding: 0.75rem;
  }

  .sp-account-table-shell :deep(> .space-y-3 > div) {
    border-color: var(--sp-line);
    background: var(--sp-panel-2);
  }
}

@media (max-width: 520px) {
  .sp-account-filter-fields {
    grid-template-columns: 1fr;
  }

  .sp-account-search {
    grid-column: auto;
  }
}

@media (prefers-reduced-motion: reduce) {
  .sp-account-table-shell :deep(tbody tr) {
    transition: none;
  }
}
</style>

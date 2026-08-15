<template>
  <SupplierModuleLayout>
    <!-- 筛选工具栏 -->
    <section class="sp-filter-toolbar">
      <header class="sp-filter-card-head">
        <div class="sp-filter-card-head-content">
          <span class="sp-filter-card-kicker">监控数据关联</span>
          <h2>供应商监控绑定</h2>
          <p>把供应商监控项绑定到本地账号，模型监控会沿用账号所属的全部分组展示数据。</p>
        </div>
        <div class="sp-filter-card-head-decor" aria-hidden="true"></div>
      </header>
      <div class="sp-filter-card-body">
        <div class="sp-filter-fields">
          <div class="sp-filter-control sp-filter-search-control">
            <span class="sr-only">监控项搜索</span>
            <Input v-model="search" class="w-full" placeholder="搜索监控名称、Key 或模型" />
          </div>
          <div class="sp-filter-control">
            <span class="sr-only">供应商</span>
            <Select
              v-model="providerID"
              class="w-full"
              :options="providerOptions"
              :searchable="true"
              search-placeholder="搜索供应商"
            />
          </div>
          <div class="sp-filter-control">
            <span class="sr-only">监控状态</span>
            <Select v-model="activeFilter" class="w-full" :options="activeOptions" :searchable="false" />
          </div>
        </div>
        <div class="sp-filter-actions">
          <button class="sp-button small sp-control-button sp-control-button-reset" type="button" :disabled="loading" @click="resetFilters">
            <Icon name="x" size="sm" />
            <span>重置筛选</span>
          </button>
          <button class="sp-button sp-control-button sp-control-button-refresh" type="button" :disabled="loading" @click="loadTargets">
            <Icon name="refresh" size="sm" :class="loading ? 'sp-spin' : ''" />
            <span>刷新</span>
          </button>
        </div>
      </div>
    </section>

    <!-- 概览统计栏 -->
    <div class="sp-monitor-overview">
      <div class="sp-monitor-overview-stat">
        <div class="sp-monitor-overview-icon">
          <Icon name="chartBar" size="md" />
        </div>
        <div class="sp-monitor-overview-text">
          <span class="sp-monitor-overview-label">显式关系</span>
          <strong>{{ total }}</strong>
          <span class="sp-monitor-overview-unit">个监控项</span>
        </div>
      </div>
      <div class="sp-monitor-overview-divider" aria-hidden="true"></div>
      <div class="sp-monitor-overview-note">
        <Icon name="link" size="sm" />
        <span>绑定关系独立于供应商账号名称和本地分组名称。</span>
      </div>
    </div>

    <!-- 数据表格 -->
    <section class="sp-monitor-table-card">
      <div class="sp-monitor-table-inner">
        <DataTable :columns="columns" :data="targets" :loading="loading" row-key="id">
          <template #cell-provider_name="{ row }">
            <div class="sp-monitor-provider">
              <span class="sp-provider-dot" :class="providerDotClass(row.provider_id)" aria-hidden="true"></span>
              <div class="sp-monitor-provider-text">
                <strong>{{ row.provider_name || `供应商 #${row.provider_id}` }}</strong>
                <span>#{{ row.provider_id }}</span>
              </div>
            </div>
          </template>
          <template #cell-monitor_name="{ row }">
            <div class="sp-monitor-target-cell">
              <strong>{{ row.monitor_name || '未命名监控项' }}</strong>
              <span>{{ row.monitor_key }} · {{ row.monitor_provider || '未知探针' }}</span>
            </div>
          </template>
          <template #cell-primary_model="{ row }">
            <code class="sp-monitor-model-code">{{ row.primary_model || '—' }}</code>
          </template>
          <template #cell-availability_7d="{ row }">
            <span :class="['sp-monitor-availability', availabilityClass(row.availability_7d)]">
              {{ formatAvailability(row.availability_7d) }}
            </span>
          </template>
          <template #cell-local_account_name="{ row }">
            <div v-if="row.local_account_id" class="sp-monitor-binding-cell">
              <div class="sp-monitor-binding-name">
                <Icon name="link" size="xs" class="sp-monitor-binding-icon" />
                <strong>{{ row.local_account_name }}</strong>
              </div>
              <span v-if="row.binding_groups.length" class="sp-monitor-binding-groups">{{ groupNames(row.binding_groups) }}</span>
              <span v-else class="sp-monitor-muted">未加入本地分组</span>
            </div>
            <span v-else class="sp-monitor-unbound">
              <span class="sp-monitor-unbound-dot" aria-hidden="true"></span>
              未绑定本地账号
            </span>
          </template>
          <template #cell-last_seen_at="{ row }">
            <span class="sp-monitor-time">{{ formatTime(row.last_seen_at) }}</span>
          </template>
          <template #cell-actions="{ row }">
            <div class="sp-monitor-actions">
              <button class="sp-button small primary" type="button" @click="openBinding(row)">
                <Icon name="link" size="sm" />
                <span>{{ row.local_account_id ? '更换' : '绑定' }}</span>
              </button>
              <button v-if="row.local_account_id" class="sp-button small ghost" type="button" @click="unbind(row)">
                <Icon name="x" size="sm" />
                <span>解除</span>
              </button>
            </div>
          </template>
        </DataTable>
        <div v-if="!loading && !targets.length" class="sp-monitor-empty">
          <div class="sp-monitor-empty-icon">
            <Icon name="chartBar" size="xl" />
          </div>
          <strong>暂无监控目标</strong>
          <span>先在供应商管理中同步一次监控数据。</span>
        </div>
      </div>
      <Pagination
        v-if="total > 0"
        class="sp-data-pagination"
        :page="page"
        :total="total"
        :page-size="pageSize"
        :show-page-size-selector="false"
        @update:page="handlePageChange"
      />
    </section>

    <!-- 绑定弹窗 -->
    <BaseDialog :show="Boolean(bindingTarget)" title="绑定本地账号" width="wide" @close="closeBinding">
      <div class="supplier-monitor-dialog">
        <div v-if="bindingTarget" class="sp-binding-summary">
          <div class="sp-binding-summary-card">
            <div class="sp-binding-summary-icon sp-binding-summary-icon-monitor">
              <Icon name="chartBar" size="sm" />
            </div>
            <div class="sp-binding-summary-body">
              <span>监控项</span>
              <strong>{{ bindingTarget.monitor_name }}</strong>
            </div>
          </div>
          <div class="sp-binding-summary-card">
            <div class="sp-binding-summary-icon sp-binding-summary-icon-provider">
              <Icon name="server" size="sm" />
            </div>
            <div class="sp-binding-summary-body">
              <span>供应商</span>
              <strong>{{ bindingTarget.provider_name }}</strong>
            </div>
          </div>
          <div class="sp-binding-summary-card">
            <div class="sp-binding-summary-icon sp-binding-summary-icon-current">
              <Icon name="link" size="sm" />
            </div>
            <div class="sp-binding-summary-body">
              <span>当前绑定</span>
              <strong :class="{ 'sp-binding-summary-unbound': !bindingTarget.local_account_name }">
                {{ bindingTarget.local_account_name || '未绑定' }}
              </strong>
            </div>
          </div>
        </div>
        <div class="sp-binding-search">
          <span class="sr-only">搜索本地账号</span>
          <Input v-model="accountSearch" class="w-full" placeholder="搜索本地账号、备注或模型" />
        </div>
        <div class="sp-binding-account-list" :class="{ loading: accountsLoading }">
          <button
            v-for="account in localAccounts"
            :key="account.id"
            class="sp-binding-account"
            :class="{ selected: selectedAccountID === account.id }"
            type="button"
            @click="selectedAccountID = account.id"
          >
            <span class="sp-binding-account-mark"><Icon name="check" size="sm" /></span>
            <span class="sp-binding-account-copy">
              <strong>{{ account.name }}</strong>
              <span>{{ account.platform }} · {{ account.groups?.map(group => group.name).join('、') || '未加入分组' }}</span>
            </span>
            <span v-if="account.id === bindingTarget?.local_account_id" class="sp-binding-current">当前</span>
          </button>
          <div v-if="accountsLoading" class="sp-binding-list-state">
            <Icon name="refresh" size="sm" class="sp-spin" />
            <span>正在加载本地账号…</span>
          </div>
          <div v-else-if="!localAccounts.length" class="sp-binding-list-state">没有找到可绑定的本地账号。</div>
        </div>
      </div>
      <template #footer>
        <div class="sp-dialog-actions">
          <button class="sp-button ghost" type="button" @click="closeBinding">取消</button>
          <button class="sp-button primary" type="button" :disabled="!selectedAccountID || saving" @click="saveBinding">
            <Icon v-if="!saving" name="check" size="sm" />
            <Icon v-else name="refresh" size="sm" class="sp-spin" />
            <span>{{ saving ? '保存中' : '保存绑定' }}</span>
          </button>
        </div>
      </template>
    </BaseDialog>
  </SupplierModuleLayout>
</template>

<script setup lang="ts">
import { computed, onMounted, ref, watch } from 'vue'
import { adminAPI } from '@/api/admin'
import {
  bindSupplierMonitorTarget,
  listSupplierMonitorTargets,
  unbindSupplierMonitorTarget,
  type SupplierProviderMonitorTarget,
} from '@/api/admin/supplierProviderData'
import supplierProvidersAPI, { type SupplierProvider } from '@/api/admin/supplierProviders'
import BaseDialog from '@/components/common/BaseDialog.vue'
import DataTable from '@/components/common/DataTable.vue'
import Input from '@/components/common/Input.vue'
import Pagination from '@/components/common/Pagination.vue'
import Select, { type SelectOption } from '@/components/common/Select.vue'
import type { Column } from '@/components/common/types'
import Icon from '@/components/icons/Icon.vue'
import { SupplierModuleLayout } from '@/components/admin/supplier-management'
import { useAppStore } from '@/stores/app'
import type { Account } from '@/types'

const appStore = useAppStore()
const targets = ref<SupplierProviderMonitorTarget[]>([])
const providers = ref<SupplierProvider[]>([])
const localAccounts = ref<Account[]>([])
const bindingTarget = ref<SupplierProviderMonitorTarget | null>(null)
const selectedAccountID = ref<number | null>(null)
const search = ref('')
const accountSearch = ref('')
const providerID = ref<number | null>(null)
const activeFilter = ref<boolean | null>(true)
const page = ref(1)
const pageSize = ref(20)
const total = ref(0)
const loading = ref(false)
const accountsLoading = ref(false)
const saving = ref(false)

const columns: Column[] = [
  { key: 'provider_name', label: '供应商', class: 'min-w-[170px]' },
  { key: 'monitor_name', label: '监控项', class: 'min-w-[250px]' },
  { key: 'primary_model', label: '主模型', class: 'min-w-[145px]' },
  { key: 'availability_7d', label: '7 天可用率', class: 'min-w-[120px]' },
  { key: 'local_account_name', label: '绑定本地账号', class: 'min-w-[250px]' },
  { key: 'last_seen_at', label: '最近同步', class: 'min-w-[165px]' },
  { key: 'actions', label: '操作', class: 'min-w-[180px]' },
]

const providerDotColors = ['#3b82f6', '#16a34a', '#7c3aed', '#d97706', '#dc2626', '#0891b2', '#4f46e5', '#db2777']
function providerDotClass(providerId: number) {
  const idx = (providerId || 0) % providerDotColors.length
  return `sp-provider-dot-${idx}`
}

const providerOptions = computed<SelectOption[]>(() => [
  { value: null, label: '全部供应商' },
  ...providers.value.map(provider => ({ value: provider.id, label: provider.name }))
])

const activeOptions: SelectOption[] = [
  { value: true, label: '仅显示活跃监控' },
  { value: null, label: '全部监控项' },
]

let searchTimer: number | undefined
let accountSearchTimer: number | undefined

async function loadProviders() {
  try {
    const result = await supplierProvidersAPI.list({ page: 1, page_size: 200 })
    providers.value = result.items
  } catch (error) {
    appStore.showError(errorMessage(error, '加载供应商失败'))
  }
}

async function loadTargets() {
  loading.value = true
  try {
    const result = await listSupplierMonitorTargets({
      provider_id: providerID.value || undefined,
      active: activeFilter.value === null ? undefined : activeFilter.value,
      search: search.value.trim() || undefined,
      page: page.value,
      page_size: pageSize.value,
    })
    targets.value = result.items
    total.value = result.total
  } catch (error) {
    appStore.showError(errorMessage(error, '加载监控项失败'))
  } finally {
    loading.value = false
  }
}

async function loadLocalAccounts() {
  accountsLoading.value = true
  try {
    const result = await adminAPI.accounts.list(1, 200, {
      search: accountSearch.value.trim() || undefined,
      status: 'active',
      sort_by: 'name',
      sort_order: 'asc',
    })
    localAccounts.value = result.items
  } catch (error) {
    appStore.showError(errorMessage(error, '加载本地账号失败'))
  } finally {
    accountsLoading.value = false
  }
}

function openBinding(target: SupplierProviderMonitorTarget) {
  bindingTarget.value = target
  selectedAccountID.value = target.local_account_id || null
  accountSearch.value = ''
  void loadLocalAccounts()
}

function closeBinding() {
  if (saving.value) return
  bindingTarget.value = null
  selectedAccountID.value = null
}

async function saveBinding() {
  if (!bindingTarget.value || !selectedAccountID.value) return
  saving.value = true
  try {
    await bindSupplierMonitorTarget(bindingTarget.value.id, selectedAccountID.value)
    appStore.showSuccess('监控项绑定已保存')
    closeBinding()
    await loadTargets()
  } catch (error) {
    appStore.showError(errorMessage(error, '保存监控项绑定失败'))
  } finally {
    saving.value = false
  }
}

async function unbind(target: SupplierProviderMonitorTarget) {
  try {
    await unbindSupplierMonitorTarget(target.id)
    appStore.showSuccess('监控项绑定已解除')
    await loadTargets()
  } catch (error) {
    appStore.showError(errorMessage(error, '解除监控项绑定失败'))
  }
}

function resetFilters() {
  search.value = ''
  providerID.value = null
  activeFilter.value = true
  page.value = 1
  void loadTargets()
}

function handlePageChange(nextPage: number) {
  page.value = nextPage
  void loadTargets()
}

function groupNames(groups: SupplierProviderMonitorTarget['binding_groups']) {
  return groups.map(group => group.name).join('、')
}

function formatAvailability(value: number) {
  return `${Number(value || 0).toFixed(2)}%`
}

function availabilityClass(value: number) {
  if (value >= 99) return 'healthy'
  if (value >= 95) return 'warning'
  return 'danger'
}

function formatTime(value: string) {
  if (!value) return '—'
  return new Intl.DateTimeFormat('zh-CN', { dateStyle: 'short', timeStyle: 'short' }).format(new Date(value))
}

function errorMessage(error: unknown, fallback: string) {
  return error instanceof Error && error.message ? error.message : fallback
}

watch([search, providerID, activeFilter], () => {
  window.clearTimeout(searchTimer)
  searchTimer = window.setTimeout(() => {
    page.value = 1
    void loadTargets()
  }, 260)
})

watch(accountSearch, () => {
  if (!bindingTarget.value) return
  window.clearTimeout(accountSearchTimer)
  accountSearchTimer = window.setTimeout(() => void loadLocalAccounts(), 260)
})

onMounted(async () => {
  await loadProviders()
  await loadTargets()
})
</script>

<style scoped>
/* ===== 筛选工具栏 ===== */
.sp-filter-toolbar {
  overflow: hidden;
  border: 1px solid var(--sp-line);
  border-radius: 0.875rem;
  background: var(--sp-panel);
  box-shadow: var(--sp-shadow);
}

.sp-filter-card-head {
  position: relative;
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 1.25rem 1.25rem 1.1rem;
  border-bottom: 1px solid var(--sp-line);
  background: linear-gradient(135deg, color-mix(in srgb, var(--sp-cyan) 4%, transparent) 0%, transparent 60%);
}

.sp-filter-card-head-content {
  position: relative;
  z-index: 1;
}

.sp-filter-card-head-content h2 {
  margin: 0.15rem 0 0.35rem;
  color: var(--sp-text);
  font-size: 1.25rem;
  font-weight: 700;
  letter-spacing: -0.01em;
  line-height: 1.35;
}

.sp-filter-card-head-content p {
  margin: 0;
  color: var(--sp-muted);
  font-size: 0.8125rem;
  line-height: 1.5;
  max-width: 42rem;
}

.sp-filter-card-kicker {
  display: inline-block;
  color: var(--sp-cyan);
  font-size: 0.6875rem;
  font-weight: 700;
  letter-spacing: 0.06em;
  text-transform: uppercase;
}

.sp-filter-card-head-decor {
  position: absolute;
  right: -1.5rem;
  top: -1.5rem;
  width: 10rem;
  height: 10rem;
  border-radius: 50%;
  background: radial-gradient(circle, color-mix(in srgb, var(--sp-cyan) 6%, transparent) 0%, transparent 70%);
  pointer-events: none;
}

.sp-filter-card-body {
  display: flex;
  align-items: flex-end;
  gap: 0.875rem;
  padding: 0.875rem 1rem 1rem;
}

.sp-filter-fields {
  display: grid;
  grid-template-columns: minmax(15rem, 1fr) repeat(2, minmax(9rem, 0.55fr));
  gap: 0.75rem;
  flex: 1;
  min-width: 0;
}

.sp-filter-control {
  min-width: 0;
}

.sp-filter-actions {
  display: flex;
  flex-shrink: 0;
  align-items: center;
  gap: 0.5rem;
  padding-left: 0.875rem;
  border-left: 1px solid var(--sp-line);
}

.sp-control-button {
  display: inline-flex;
  align-items: center;
  gap: 0.375rem;
  border-color: transparent;
  background: transparent;
  font-weight: 500;
  transition: background 0.15s ease, color 0.15s ease;
}

.sp-control-button-reset {
  color: var(--sp-muted);
}

.sp-control-button-reset:hover {
  background: color-mix(in srgb, var(--sp-muted) 8%, transparent);
  color: var(--sp-text);
}

.sp-control-button-refresh {
  color: var(--sp-green);
}

.sp-control-button-refresh:hover {
  background: color-mix(in srgb, var(--sp-green) 8%, transparent);
}

/* ===== 概览统计栏 ===== */
.sp-monitor-overview {
  display: flex;
  align-items: center;
  gap: 1rem;
  margin: 1rem 0;
  padding: 1rem 1.25rem;
  border: 1px solid var(--sp-line);
  border-radius: 0.75rem;
  background: var(--sp-panel);
  box-shadow: var(--sp-shadow);
}

.sp-monitor-overview-stat {
  display: flex;
  align-items: center;
  gap: 0.75rem;
}

.sp-monitor-overview-icon {
  display: grid;
  width: 2.5rem;
  height: 2.5rem;
  flex-shrink: 0;
  place-items: center;
  border-radius: 0.625rem;
  background: color-mix(in srgb, var(--sp-cyan) 10%, transparent);
  color: var(--sp-cyan);
}

.sp-monitor-overview-text {
  display: flex;
  align-items: baseline;
  gap: 0.4rem;
  color: var(--sp-muted);
}

.sp-monitor-overview-label {
  font-size: 0.75rem;
  font-weight: 500;
}

.sp-monitor-overview-text strong {
  color: var(--sp-text);
  font-size: 1.5rem;
  font-weight: 700;
  font-variant-numeric: tabular-nums;
  line-height: 1;
}

.sp-monitor-overview-unit {
  font-size: 0.8125rem;
}

.sp-monitor-overview-divider {
  width: 1px;
  height: 2rem;
  flex-shrink: 0;
  background: var(--sp-line);
}

.sp-monitor-overview-note {
  display: flex;
  align-items: center;
  gap: 0.45rem;
  color: var(--sp-cyan);
  font-size: 0.8rem;
  opacity: 0.85;
}

/* ===== 表格卡片 ===== */
.sp-monitor-table-card {
  overflow: hidden;
  border: 1px solid var(--sp-line);
  border-radius: 0.875rem;
  background: var(--sp-panel);
  box-shadow: var(--sp-shadow);
}

.sp-monitor-table-inner {
  overflow: auto;
}

/* ===== 表格单元格 ===== */
.sp-monitor-provider {
  display: flex;
  align-items: center;
  gap: 0.6rem;
}

.sp-provider-dot {
  width: 0.5rem;
  height: 0.5rem;
  flex-shrink: 0;
  border-radius: 50%;
  box-shadow: 0 0 0 2px color-mix(in srgb, currentColor 20%, transparent);
}

.sp-provider-dot-0 { background: #3b82f6; color: #3b82f6; }
.sp-provider-dot-1 { background: #16a34a; color: #16a34a; }
.sp-provider-dot-2 { background: #7c3aed; color: #7c3aed; }
.sp-provider-dot-3 { background: #d97706; color: #d97706; }
.sp-provider-dot-4 { background: #dc2626; color: #dc2626; }
.sp-provider-dot-5 { background: #0891b2; color: #0891b2; }
.sp-provider-dot-6 { background: #4f46e5; color: #4f46e5; }
.sp-provider-dot-7 { background: #db2777; color: #db2777; }

.sp-monitor-provider-text {
  display: flex;
  flex-direction: column;
  gap: 0.2rem;
  min-width: 0;
}

.sp-monitor-provider-text strong {
  color: var(--sp-text);
  font-size: 0.875rem;
  font-weight: 600;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.sp-monitor-provider-text span {
  color: var(--sp-muted);
  font-size: 0.7rem;
}

.sp-monitor-target-cell {
  display: flex;
  flex-direction: column;
  gap: 0.25rem;
  min-width: 0;
}

.sp-monitor-target-cell strong {
  color: var(--sp-text);
  font-size: 0.875rem;
  font-weight: 600;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.sp-monitor-target-cell span {
  color: var(--sp-muted);
  font-size: 0.72rem;
}

.sp-monitor-model-code {
  display: inline-block;
  padding: 0.2rem 0.5rem;
  border: 1px solid var(--sp-line);
  border-radius: 0.35rem;
  background: var(--sp-panel-2);
  color: var(--sp-text);
  font-size: 0.75rem;
  font-family: ui-monospace, SFMono-Regular, Consolas, "Liberation Mono", monospace;
  max-width: 100%;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

/* 可用率 */
.sp-monitor-availability {
  display: inline-flex;
  align-items: center;
  gap: 0.35rem;
  padding: 0.25rem 0.625rem;
  border-radius: 9999px;
  font-size: 0.8rem;
  font-weight: 700;
  font-variant-numeric: tabular-nums;
  line-height: 1.4;
}

.sp-monitor-availability.healthy {
  color: var(--sp-green);
  background: color-mix(in srgb, var(--sp-green) 8%, transparent);
}

.sp-monitor-availability.warning {
  color: var(--sp-amber);
  background: color-mix(in srgb, var(--sp-amber) 8%, transparent);
}

.sp-monitor-availability.danger {
  color: var(--sp-red);
  background: color-mix(in srgb, var(--sp-red) 8%, transparent);
}

/* 绑定 */
.sp-monitor-binding-cell {
  display: flex;
  flex-direction: column;
  gap: 0.3rem;
  min-width: 0;
}

.sp-monitor-binding-name {
  display: flex;
  align-items: center;
  gap: 0.4rem;
  color: var(--sp-text);
  font-size: 0.875rem;
  font-weight: 600;
}

.sp-monitor-binding-name strong {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.sp-monitor-binding-icon {
  color: var(--sp-cyan);
  flex-shrink: 0;
}

.sp-monitor-binding-groups {
  color: var(--sp-muted);
  font-size: 0.72rem;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.sp-monitor-muted {
  color: var(--sp-dim);
  font-size: 0.75rem;
}

.sp-monitor-unbound {
  display: inline-flex;
  align-items: center;
  gap: 0.4rem;
  color: var(--sp-amber);
  font-size: 0.8rem;
  font-weight: 500;
}

.sp-monitor-unbound-dot {
  width: 0.375rem;
  height: 0.375rem;
  border-radius: 50%;
  background: var(--sp-amber);
  flex-shrink: 0;
}

.sp-monitor-time {
  color: var(--sp-muted);
  font-size: 0.8rem;
  font-variant-numeric: tabular-nums;
}

/* 操作 */
.sp-monitor-actions {
  display: flex;
  flex-wrap: wrap;
  gap: 0.4rem;
}

.sp-monitor-actions .sp-button {
  display: inline-flex;
  align-items: center;
  gap: 0.375rem;
  white-space: nowrap;
}

/* 空状态 */
.sp-monitor-empty {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 0.5rem;
  padding: 4rem 1rem;
}

.sp-monitor-empty-icon {
  display: grid;
  width: 3.5rem;
  height: 3.5rem;
  place-items: center;
  border-radius: 50%;
  background: var(--sp-panel-2);
  color: var(--sp-dim);
  margin-bottom: 0.25rem;
}

.sp-monitor-empty strong {
  color: var(--sp-text);
  font-size: 0.9375rem;
}

.sp-monitor-empty span {
  color: var(--sp-muted);
  font-size: 0.8125rem;
}

/* 分页 */
.sp-data-pagination {
  flex-shrink: 0;
  padding: 0.75rem 1rem;
  border-top: 1px solid var(--sp-soft);
}

/* ===== 绑定弹窗 ===== */
.supplier-monitor-dialog {
  --sp-panel: #ffffff;
  --sp-panel-2: #f8fafc;
  --sp-line: #dbe3ec;
  --sp-ink: #172033;
  --sp-text: #172033;
  --sp-muted: #6b778c;
  --sp-dim: #94a3b8;
  --sp-cyan: #0891b2;
  --sp-green: #16a34a;
  --sp-amber: #d97706;
  --sp-red: #dc2626;
  --sp-soft: #f1f5f9;
  color: var(--sp-text);
}

.sp-binding-summary {
  display: grid;
  grid-template-columns: repeat(3, minmax(0, 1fr));
  gap: 0.75rem;
  margin-bottom: 1.125rem;
}

.sp-binding-summary-card {
  display: flex;
  align-items: flex-start;
  gap: 0.7rem;
  padding: 0.85rem;
  border: 1px solid var(--sp-line);
  border-radius: 0.625rem;
  background: var(--sp-panel-2);
  transition: border-color 0.15s ease;
}

.sp-binding-summary-icon {
  display: grid;
  width: 2rem;
  height: 2rem;
  flex-shrink: 0;
  place-items: center;
  border-radius: 0.5rem;
}

.sp-binding-summary-icon-monitor {
  background: color-mix(in srgb, var(--sp-cyan) 12%, transparent);
  color: var(--sp-cyan);
}

.sp-binding-summary-icon-provider {
  background: color-mix(in srgb, var(--sp-green) 12%, transparent);
  color: var(--sp-green);
}

.sp-binding-summary-icon-current {
  background: color-mix(in srgb, var(--sp-amber) 12%, transparent);
  color: var(--sp-amber);
}

.sp-binding-summary-body {
  display: flex;
  flex-direction: column;
  gap: 0.25rem;
  min-width: 0;
}

.sp-binding-summary-body span {
  color: var(--sp-muted);
  font-size: 0.6875rem;
  font-weight: 500;
  letter-spacing: 0.03em;
  text-transform: uppercase;
}

.sp-binding-summary-body strong {
  overflow: hidden;
  color: var(--sp-text);
  font-size: 0.85rem;
  font-weight: 600;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.sp-binding-summary-unbound {
  color: var(--sp-muted) !important;
  font-weight: 400 !important;
}

.sp-binding-search {
  margin-bottom: 0.75rem;
}

.sp-binding-account-list {
  display: grid;
  max-height: 22rem;
  gap: 0.5rem;
  overflow-y: auto;
  padding-right: 0.2rem;
}

.sp-binding-account-list.loading {
  opacity: 0.6;
  pointer-events: none;
}

.sp-binding-account {
  display: flex;
  align-items: center;
  gap: 0.65rem;
  min-width: 0;
  padding: 0.8rem 0.75rem;
  border: 1px solid var(--sp-line);
  border-radius: 0.5rem;
  background: var(--sp-panel);
  color: var(--sp-text);
  text-align: left;
  cursor: pointer;
  transition: border-color 0.16s ease, background 0.16s ease, box-shadow 0.16s ease;
}

.sp-binding-account:hover {
  border-color: color-mix(in srgb, var(--sp-cyan) 50%, var(--sp-line));
  background: color-mix(in srgb, var(--sp-cyan) 4%, var(--sp-panel));
}

.sp-binding-account.selected {
  border-color: var(--sp-cyan);
  background: color-mix(in srgb, var(--sp-cyan) 7%, var(--sp-panel));
  box-shadow: 0 0 0 2px color-mix(in srgb, var(--sp-cyan) 15%, transparent);
}

.sp-binding-account-mark {
  display: grid;
  width: 1.25rem;
  height: 1.25rem;
  flex: 0 0 auto;
  place-items: center;
  border: 1.5px solid var(--sp-line);
  border-radius: 0.3rem;
  color: transparent;
  transition: border-color 0.16s ease, background 0.16s ease, color 0.16s ease;
}

.sp-binding-account.selected .sp-binding-account-mark {
  border-color: var(--sp-cyan);
  background: var(--sp-cyan);
  color: #fff;
}

.sp-binding-account-copy {
  display: flex;
  min-width: 0;
  flex: 1;
  flex-direction: column;
  gap: 0.2rem;
}

.sp-binding-account-copy strong {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  font-size: 0.875rem;
  font-weight: 600;
}

.sp-binding-account-copy span {
  color: var(--sp-muted);
  font-size: 0.72rem;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.sp-binding-current {
  display: inline-flex;
  align-items: center;
  padding: 0.15rem 0.5rem;
  border: 1px solid color-mix(in srgb, var(--sp-cyan) 30%, var(--sp-line));
  border-radius: 9999px;
  background: color-mix(in srgb, var(--sp-cyan) 8%, transparent);
  color: var(--sp-cyan);
  font-size: 0.6875rem;
  font-weight: 600;
  flex-shrink: 0;
}

.sp-binding-list-state {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 0.5rem;
  padding: 2.5rem 1rem;
  color: var(--sp-muted);
  font-size: 0.8125rem;
  text-align: center;
}

.sp-dialog-actions {
  display: flex;
  justify-content: flex-end;
  gap: 0.5rem;
}

.sp-dialog-actions .sp-button {
  display: inline-flex;
  align-items: center;
  gap: 0.375rem;
  white-space: nowrap;
}

/* 旋转动画 */
.sp-spin {
  animation: sp-spin 0.7s linear infinite;
}

@keyframes sp-spin {
  to { transform: rotate(360deg); }
}

/* ===== 响应式 ===== */
@media (max-width: 900px) {
  .sp-filter-fields {
    grid-template-columns: 1fr;
  }

  .sp-filter-card-body {
    flex-direction: column;
    align-items: stretch;
  }

  .sp-filter-actions {
    padding-left: 0;
    padding-top: 0.75rem;
    border-left: 0;
    border-top: 1px solid var(--sp-line);
  }

  .sp-monitor-overview {
    flex-direction: column;
    align-items: flex-start;
  }

  .sp-monitor-overview-divider {
    width: 100%;
    height: 1px;
  }

  .sp-binding-summary {
    grid-template-columns: 1fr;
  }
}

@media (max-width: 700px) {
  .sp-filter-card-head {
    padding: 1rem;
  }

  .sp-filter-card-head-content h2 {
    font-size: 1.125rem;
  }

  .sp-monitor-overview {
    padding: 0.85rem 1rem;
  }
}
</style>
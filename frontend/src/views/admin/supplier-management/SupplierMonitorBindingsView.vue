<template>
  <SupplierModuleLayout>
    <section class="sp-filter-toolbar">
      <header class="sp-filter-card-head">
        <div>
          <span class="sp-filter-card-kicker">监控数据关联</span>
          <h2>供应商监控绑定</h2>
          <p>把供应商监控项绑定到本地账号，模型监控会沿用账号所属的全部分组展示数据。</p>
        </div>
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

    <div class="sp-monitor-overview">
      <div>
        <span class="sp-filter-card-kicker">显式关系</span>
        <strong>{{ total }}</strong>
        <span>个监控项</span>
      </div>
      <div class="sp-monitor-overview-note">
        <Icon name="link" size="sm" />
        <span>绑定关系独立于供应商账号名称和本地分组名称。</span>
      </div>
    </div>

    <section class="sp-table-shell sp-monitor-table-shell">
      <DataTable :columns="columns" :data="targets" :loading="loading" row-key="id">
        <template #cell-provider_name="{ row }">
          <div class="sp-monitor-provider">
            <span class="sp-provider-dot sp-provider-dot-cyan" aria-hidden="true"></span>
            <div>
              <strong>{{ row.provider_name || `供应商 #${row.provider_id}` }}</strong>
              <span>#{{ row.provider_id }}</span>
            </div>
          </div>
        </template>
        <template #cell-monitor_name="{ row }">
          <div class="sp-monitor-target-cell">
            <strong>{{ row.monitor_name || '未命名监控项' }}</strong>
            <span>Key {{ row.monitor_key }} · {{ row.monitor_provider || '未知探针' }}</span>
          </div>
        </template>
        <template #cell-primary_model="{ row }">
          <code>{{ row.primary_model || '—' }}</code>
        </template>
        <template #cell-availability_7d="{ row }">
          <span :class="['sp-monitor-availability', availabilityClass(row.availability_7d)]">
            {{ formatAvailability(row.availability_7d) }}
          </span>
        </template>
        <template #cell-local_account_name="{ row }">
          <div v-if="row.local_account_id" class="sp-monitor-binding-cell">
            <strong>{{ row.local_account_name }}</strong>
            <span v-if="row.binding_groups.length">{{ groupNames(row.binding_groups) }}</span>
            <span v-else class="sp-monitor-muted">未加入本地分组</span>
          </div>
          <span v-else class="sp-monitor-unbound">未绑定本地账号</span>
        </template>
        <template #cell-last_seen_at="{ row }">
          <span class="sp-monitor-time">{{ formatTime(row.last_seen_at) }}</span>
        </template>
        <template #cell-actions="{ row }">
          <div class="sp-monitor-actions">
            <button class="sp-button small primary" type="button" @click="openBinding(row)">
              <Icon name="link" size="sm" />
              <span>{{ row.local_account_id ? '更换账号' : '绑定账号' }}</span>
            </button>
            <button v-if="row.local_account_id" class="sp-button small ghost" type="button" @click="unbind(row)">
              <Icon name="x" size="sm" />
              <span>解除</span>
            </button>
          </div>
        </template>
      </DataTable>
      <div v-if="!loading && !targets.length" class="sp-monitor-empty">
        <Icon name="chartBar" size="xl" />
        <strong>暂无监控目标</strong>
        <span>先在供应商管理中同步一次监控数据。</span>
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

    <BaseDialog :show="Boolean(bindingTarget)" title="绑定本地账号" width="wide" @close="closeBinding">
      <div class="supplier-monitor-dialog">
        <div v-if="bindingTarget" class="sp-binding-summary">
          <div>
            <span>监控项</span>
            <strong>{{ bindingTarget.monitor_name }}</strong>
          </div>
          <div>
            <span>供应商</span>
            <strong>{{ bindingTarget.provider_name }}</strong>
          </div>
          <div>
            <span>当前绑定</span>
            <strong>{{ bindingTarget.local_account_name || '未绑定' }}</strong>
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
          <div v-if="accountsLoading" class="sp-binding-list-state">正在加载本地账号…</div>
          <div v-else-if="!localAccounts.length" class="sp-binding-list-state">没有找到可绑定的本地账号。</div>
        </div>
      </div>
      <template #footer>
        <div class="sp-dialog-actions">
          <button class="sp-button ghost" type="button" @click="closeBinding">取消</button>
          <button class="sp-button primary" type="button" :disabled="!selectedAccountID || saving" @click="saveBinding">
            <Icon name="check" size="sm" :class="saving ? 'sp-spin' : ''" />
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
  { key: 'actions', label: '操作', class: 'min-w-[210px]' },
]

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
.sp-monitor-overview {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 1rem;
  margin: 1rem 0;
  padding: 0.95rem 1.15rem;
  border: 1px solid var(--sp-line);
  background: var(--sp-panel);
}

.sp-monitor-overview > div:first-child { display: flex; align-items: baseline; gap: 0.55rem; color: var(--sp-muted); }
.sp-monitor-overview strong { color: var(--sp-ink); font-size: 1.35rem; }
.sp-monitor-overview-note { display: flex; align-items: center; gap: 0.45rem; color: var(--sp-cyan); font-size: 0.78rem; }
.sp-monitor-table-shell { overflow: hidden; }
.sp-monitor-provider, .sp-monitor-binding-cell, .sp-monitor-target-cell { display: flex; flex-direction: column; gap: 0.25rem; }
.sp-monitor-provider { flex-direction: row; align-items: center; gap: 0.6rem; }
.sp-monitor-provider span:not(.sp-provider-dot), .sp-monitor-target-cell span, .sp-monitor-binding-cell span, .sp-monitor-time { color: var(--sp-muted); font-size: 0.75rem; }
.sp-provider-dot-cyan { background: var(--sp-cyan); }
.sp-monitor-availability { font-variant-numeric: tabular-nums; font-weight: 700; }
.sp-monitor-availability.healthy { color: var(--sp-green); }
.sp-monitor-availability.warning { color: var(--sp-amber); }
.sp-monitor-availability.danger { color: var(--sp-red); }
.sp-monitor-unbound { color: var(--sp-amber); font-size: 0.8rem; }
.sp-monitor-muted { color: var(--sp-muted); }
.sp-monitor-actions { display: flex; flex-wrap: wrap; gap: 0.45rem; }
.sp-monitor-empty { display: flex; flex-direction: column; align-items: center; gap: 0.4rem; padding: 3.5rem 1rem; color: var(--sp-muted); }
.sp-monitor-empty strong { color: var(--sp-ink); }
.sp-binding-summary { display: grid; grid-template-columns: repeat(3, minmax(0, 1fr)); gap: 0.75rem; margin-bottom: 1rem; }
.sp-binding-summary > div { display: flex; flex-direction: column; gap: 0.3rem; padding: 0.8rem; border: 1px solid var(--sp-line); background: var(--sp-panel-2); }
.sp-binding-summary span { color: var(--sp-muted); font-size: 0.72rem; }
.sp-binding-summary strong { overflow: hidden; color: var(--sp-ink); font-size: 0.85rem; text-overflow: ellipsis; white-space: nowrap; }
.sp-binding-search { margin-bottom: 0.75rem; }
.sp-binding-account-list { display: grid; max-height: 22rem; gap: 0.5rem; overflow-y: auto; padding-right: 0.2rem; }
.sp-binding-account { display: flex; align-items: center; gap: 0.65rem; min-width: 0; padding: 0.75rem; border: 1px solid var(--sp-line); background: var(--sp-panel); color: var(--sp-ink); text-align: left; transition: border-color 160ms ease, background 160ms ease; }
.sp-binding-account:hover, .sp-binding-account.selected { border-color: var(--sp-cyan); background: color-mix(in srgb, var(--sp-cyan) 7%, var(--sp-panel)); }
.sp-binding-account-mark { display: grid; width: 1.25rem; height: 1.25rem; flex: 0 0 auto; place-items: center; border: 1px solid var(--sp-line); color: transparent; }
.sp-binding-account.selected .sp-binding-account-mark { border-color: var(--sp-cyan); background: var(--sp-cyan); color: white; }
.sp-binding-account-copy { display: flex; min-width: 0; flex: 1; flex-direction: column; gap: 0.2rem; }
.sp-binding-account-copy strong { overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.sp-binding-account-copy span, .sp-binding-current { color: var(--sp-muted); font-size: 0.75rem; }
.sp-binding-current { color: var(--sp-cyan); }
.sp-binding-list-state { padding: 2rem; color: var(--sp-muted); text-align: center; }
.supplier-monitor-dialog { --sp-panel: #ffffff; --sp-panel-2: #f8fafc; --sp-line: #dbe3ec; --sp-ink: #172033; --sp-muted: #6b778c; --sp-cyan: #0891b2; }
@media (max-width: 700px) {
  .sp-monitor-overview { align-items: flex-start; flex-direction: column; }
  .sp-binding-summary { grid-template-columns: 1fr; }
}
</style>

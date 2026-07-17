<template>
  <SupplierModuleLayout>
    <header class="sp-page-head">
      <div>
        <div class="sp-eyebrow">Local Supplier Accounts</div>
        <h1>上游账号</h1>
        <p class="sp-subtitle">只展示已同步到本地数据库的供应商账号，不实时调用上游接口。</p>
      </div>
      <div class="sp-controls">
        <Select v-model="providerID" class="sp-search" :options="providerOptions" :searchable="false" />
        <Select v-model="activeFilter" class="sp-search" :options="activeFilterOptions" :searchable="false" />
        <Input v-model="search" class="sp-search" placeholder="搜索账号或上游 Key" />
        <button class="sp-button" type="button" :disabled="loading" @click="loadAccounts">刷新</button>
      </div>
    </header>

    <div v-if="error" class="sp-alert sp-error-line">{{ error }}</div>

    <section class="sp-grid-2">
      <div class="sp-panel">
        <header class="sp-panel-head">
          <div class="sp-panel-title"><span class="sp-section-index">01</span><div><h2>本地账号表</h2><span>共 {{ total }} 条同步记录</span></div></div>
        </header>
        <DataTable
          :columns="accountColumns"
          :data="items"
          :loading="loading"
          row-key="id"
          clickable-rows
          @row-click="selected = $event"
        >
          <template #cell-provider_name="{ row: account }">
            <div class="sp-entity">{{ account.provider_name }}</div>
            <div class="sp-sub">ID {{ account.provider_id }}</div>
          </template>
          <template #cell-name="{ row: account }">
            <div class="sp-entity">{{ account.name || '未命名账号' }}</div>
            <div class="sp-sub">{{ account.upstream_account_key }}</div>
          </template>
          <template #cell-group_name="{ row: account }">
            {{ account.group_name || account.group_key || '未分组' }}
          </template>
          <template #cell-rate_multiplier="{ row: account }">
            <span class="sp-num">{{ account.rate_multiplier || 0 }}</span>
          </template>
          <template #cell-raw_status="{ row: account }">
            {{ account.raw_status || account.status || 'unknown' }}
          </template>
          <template #cell-active="{ row: account }">
            <span class="sp-status" :class="account.active ? 'good' : ''">{{ account.active ? '有效' : '已失效' }}</span>
          </template>
          <template #cell-last_seen_at="{ row: account }">
            {{ formatTime(account.last_seen_at) }}
          </template>
          <template #empty>
            暂无本地账号数据，请先在供应商列表执行同步。
          </template>
        </DataTable>
        <Pagination
          v-if="total > 0"
          class="sp-data-pagination"
          :page="page"
          :total="total"
          :page-size="pageSize"
          :show-page-size-selector="false"
          @update:page="page = $event"
        />
      </div>

      <aside class="sp-panel">
        <header class="sp-panel-head"><div class="sp-panel-title"><span class="sp-section-index">02</span><div><h2>查询说明</h2><span>页面只读取本地同步结果</span></div></div></header>
        <div class="sp-panel-body">
          <div class="sp-stat-list">
            <div class="sp-stat-box"><span>当前筛选记录</span><b>{{ items.length }}</b></div>
            <div class="sp-stat-box"><span>总记录</span><b>{{ total }}</b></div>
            <div class="sp-stat-box"><span>页大小</span><b>{{ pageSize }}</b></div>
          </div>
        </div>
      </aside>
    </section>

    <SupplierDrawer :show="Boolean(selected)" :title="selected?.name || selected?.upstream_account_key || ''" eyebrow="ACCOUNT DETAIL" @close="selected = null">
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
import { computed, onMounted, ref, watch } from 'vue'
import { SupplierDrawer, SupplierModuleLayout } from '@/components/admin/supplier-management'
import DataTable from '@/components/common/DataTable.vue'
import Input from '@/components/common/Input.vue'
import Pagination from '@/components/common/Pagination.vue'
import Select, { type SelectOption } from '@/components/common/Select.vue'
import supplierProvidersAPI, { type SupplierProvider } from '@/api/admin/supplierProviders'
import { listSupplierAccounts, type SupplierProviderAccount } from '@/api/admin/supplierProviderData'
import type { Column } from '@/components/common/types'

const providers = ref<SupplierProvider[]>([])
const items = ref<SupplierProviderAccount[]>([])
const selected = ref<SupplierProviderAccount | null>(null)
const total = ref(0)
const loading = ref(false)
const error = ref('')
const page = ref(1)
const pageSize = ref(50)
const providerID = ref(0)
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
const accountColumns: Column[] = [
  { key: 'provider_name', label: '供应商', class: 'min-w-[150px]' },
  { key: 'name', label: '上游 Key / 名称', class: 'min-w-[190px]' },
  { key: 'group_name', label: '分组', class: 'min-w-[130px]' },
  { key: 'rate_multiplier', label: '倍率' },
  { key: 'raw_status', label: '上游状态' },
  { key: 'active', label: '本地状态' },
  { key: 'last_seen_at', label: '最近同步', class: 'min-w-[150px]' },
]

onMounted(async () => {
  await loadProviders()
  await loadAccounts()
})

watch([providerID, activeFilter], () => {
  page.value = 1
  void loadAccounts()
})

watch(page, () => { void loadAccounts() })

watch(search, () => {
  window.clearTimeout(searchTimer)
  searchTimer = window.setTimeout(() => {
    page.value = 1
    void loadAccounts()
  }, 350)
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

function formatTime(value?: string): string {
  if (!value) return '—'
  const date = new Date(value)
  if (Number.isNaN(date.getTime())) return '—'
  return date.toLocaleString('zh-CN')
}
</script>

<template>
  <SupplierModuleLayout>
    <header class="sp-page-head">
      <div>
        <div class="sp-eyebrow">Local Supplier Accounts</div>
        <h1>上游账号</h1>
        <p class="sp-subtitle">只展示已同步到本地数据库的供应商账号，不实时调用上游接口。</p>
      </div>
      <div class="sp-controls">
        <select v-model.number="providerID" class="sp-search">
          <option :value="0">全部供应商</option>
          <option v-for="provider in providers" :key="provider.id" :value="provider.id">{{ provider.name }}</option>
        </select>
        <select v-model="activeFilter" class="sp-search">
          <option value="true">仅有效</option>
          <option value="">全部状态</option>
          <option value="false">已失效</option>
        </select>
        <input v-model="search" class="sp-search" placeholder="搜索账号或上游 Key" />
        <button class="sp-button" type="button" :disabled="loading" @click="loadAccounts">刷新</button>
      </div>
    </header>

    <div v-if="error" class="sp-alert sp-error-line">{{ error }}</div>

    <section class="sp-grid-2">
      <div class="sp-panel">
        <header class="sp-panel-head">
          <div class="sp-panel-title"><span class="sp-section-index">01</span><div><h2>本地账号表</h2><span>共 {{ total }} 条同步记录</span></div></div>
        </header>
        <div class="sp-table-wrap">
          <table class="sp-table">
            <thead><tr><th>供应商</th><th>上游 Key / 名称</th><th>分组</th><th>倍率</th><th>上游状态</th><th>本地状态</th><th>最近同步</th></tr></thead>
            <tbody>
              <tr v-if="loading"><td colspan="7">正在加载账号数据...</td></tr>
              <tr v-for="account in items" :key="account.id" class="clickable" @click="selected = account">
                <td><div class="sp-entity">{{ account.provider_name }}</div><div class="sp-sub">ID {{ account.provider_id }}</div></td>
                <td><div class="sp-entity">{{ account.name || '未命名账号' }}</div><div class="sp-sub">{{ account.upstream_account_key }}</div></td>
                <td>{{ account.group_name || account.group_key || '未分组' }}</td>
                <td><span class="sp-num">{{ account.rate_multiplier || 0 }}</span></td>
                <td>{{ account.raw_status || account.status || 'unknown' }}</td>
                <td><span class="sp-status" :class="account.active ? 'good' : ''">{{ account.active ? '有效' : '已失效' }}</span></td>
                <td>{{ formatTime(account.last_seen_at) }}</td>
              </tr>
              <tr v-if="!loading && !items.length"><td colspan="7">暂无本地账号数据，请先在供应商列表执行同步。</td></tr>
            </tbody>
          </table>
        </div>
        <div class="sp-footer-note">
          <button class="sp-button small" type="button" :disabled="page <= 1 || loading" @click="page--">上一页</button>
          <span>第 {{ page }} 页 / {{ totalPages }} 页</span>
          <button class="sp-button small" type="button" :disabled="page >= totalPages || loading" @click="page++">下一页</button>
        </div>
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
import supplierProvidersAPI, { type SupplierProvider } from '@/api/admin/supplierProviders'
import { listSupplierAccounts, type SupplierProviderAccount } from '@/api/admin/supplierProviderData'

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

const totalPages = computed(() => Math.max(1, Math.ceil(total.value / pageSize.value)))

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

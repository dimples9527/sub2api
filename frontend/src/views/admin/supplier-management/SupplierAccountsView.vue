<template>
  <SupplierModuleLayout>
    <section class="sp-account-toolbar" aria-label="账号筛选与操作">
      <div class="sp-account-filter-fields">
        <div
          ref="searchFilterControl"
          class="sp-account-filter-control sp-account-search"
          role="group"
          aria-labelledby="supplier-account-search-label"
        >
          <span id="supplier-account-search-label" class="sp-account-filter-label">账号搜索</span>
          <Input
            v-model="search"
            class="sp-search"
            placeholder="搜索账号名称或上游 Key"
          />
        </div>
        <div
          ref="providerFilterControl"
          class="sp-account-filter-control"
          role="group"
          aria-labelledby="supplier-account-provider-label"
        >
          <span id="supplier-account-provider-label" class="sp-account-filter-label">供应商</span>
          <Select
            v-model="providerID"
            class="sp-search sp-account-select"
            :options="providerOptions"
            :searchable="false"
          />
        </div>
        <div
          ref="platformFilterControl"
          class="sp-account-filter-control"
          role="group"
          aria-labelledby="supplier-account-platform-label"
        >
          <span id="supplier-account-platform-label" class="sp-account-filter-label">平台</span>
          <Select
            v-model="platformFilter"
            class="sp-search sp-account-select"
            :options="platformFilterOptions"
            :searchable="false"
          />
        </div>
        <div
          ref="activeFilterControl"
          class="sp-account-filter-control"
          role="group"
          aria-labelledby="supplier-account-active-label"
        >
          <span id="supplier-account-active-label" class="sp-account-filter-label">账号状态</span>
          <Select
            v-model="activeFilter"
            class="sp-search sp-account-select"
            :options="activeFilterOptions"
            :searchable="false"
          />
        </div>
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
            <h2>上游账号表</h2>
            <span>当前筛选共 {{ total }} 个上游账号</span>
          </div>
        </div>
        <div class="sp-account-legend" aria-label="本地账号匹配状态图例">
          <span><i class="matched"></i>已匹配</span>
          <span><i class="unmatched"></i>未匹配</span>
          <span><i class="conflict"></i>匹配冲突</span>
        </div>
      </header>

      <div class="sp-account-table-shell">
        <DataTable
          :columns="accountColumns"
          :data="items"
          :loading="loading"
          row-key="id"
          clickable-rows
          @row-click="openDrawer"
        >
          <template #cell-provider_name="{ row: account }">
            <div :class="['sp-provider-cell', supplierTone(account.provider_id).chip]">
              <span
                :class="['sp-provider-dot', supplierTone(account.provider_id).dot]"
                aria-hidden="true"
              ></span>
              <div class="sp-account-copy">
                <div class="sp-entity">{{ account.provider_name || '—' }}</div>
                <div class="sp-sub sp-account-meta">
                  <span>供应商 #{{ account.provider_id }}</span>
                </div>
              </div>
            </div>
          </template>

          <template #cell-upstream_account_key="{ row: account }">
            <div class="sp-account-identity">
              <span class="sp-account-avatar" aria-hidden="true">{{ accountInitial(account) }}</span>
              <div class="sp-account-copy">
                <div class="sp-entity">{{ account.name || '—' }}</div>
                <div class="sp-sub sp-account-meta">
                  <span
                    v-if="account.platform"
                    :class="['sp-platform-badge', platformBadgeClass(account.platform)]"
                  >
                    {{ platformLabel(account.platform) }}
                  </span>
                  <span v-else class="sp-account-muted">—</span>
                  <span class="sp-account-key" :title="account.upstream_account_key || '—'">
                    {{ account.upstream_account_key || '—' }}
                  </span>
                </div>
              </div>
            </div>
          </template>

          <template #cell-local_account_name="{ row: account }">
            <span
              v-if="account.local_account_match_status === 'unmatched'"
              class="sp-match-badge unmatched"
            >
              未匹配
            </span>
            <span
              v-else-if="account.local_account_match_status === 'conflict'"
              class="sp-match-badge conflict"
            >
              匹配冲突（{{ account.local_account_match_count }}）
            </span>
            <div
              v-else-if="account.local_account_match_status === 'matched'"
              class="sp-local-account-cell"
            >
              <span class="sp-match-badge matched">已匹配</span>
              <strong>{{ displayValue(account.local_account_name) }}</strong>
            </div>
            <span v-else class="sp-account-muted">—</span>
          </template>

          <template #cell-local_account_priority="{ row: account }">
            <div v-if="canEditPriority(account)" class="sp-priority-cell" @click.stop>
              <Input
                v-if="editingPriorityAccountID === account.local_account_id"
                ref="priorityInput"
                v-model="priorityDraft"
                type="number"
                class="sp-priority-input"
                :disabled="savingPriorityAccountID === account.local_account_id"
                @click.stop
                @enter="savePriority(account)"
                @keydown.esc="cancelPriorityEdit"
                @blur="savePriority(account)"
              />
              <button
                v-else
                type="button"
                class="sp-account-number sp-priority-trigger"
                :disabled="savingPriorityAccountID === account.local_account_id"
                title="点击编辑账号优先级"
                @click.stop="startPriorityEdit(account)"
              >
                {{ displayValue(account.local_account_priority) }}
              </button>
            </div>
            <span v-else class="sp-account-muted">—</span>
          </template>

          <template #cell-rate_multiplier="{ row: account }">
            <span :class="['sp-account-rate', platformTextClass(account.platform || '')]">
              {{ formatRate(account.rate_multiplier) }}
            </span>
          </template>

          <template #cell-group_name="{ row: account }">
            <span v-if="account.group_name || account.group_key" class="sp-account-group">
              {{ account.group_name || account.group_key }}
            </span>
            <span v-else class="sp-account-muted">—</span>
          </template>

          <template #cell-local_account_status="{ row: account }">
            <span
              v-if="isMatchedLocalAccount(account)"
              :class="['sp-local-status', localAccountStatusTone(account.local_account_status)]"
            >
              {{ localAccountStatusLabel(account.local_account_status) }}
            </span>
            <span v-else class="sp-account-muted">—</span>
          </template>

          <template #cell-local_account_schedulable="{ row: account }">
            <button
              v-if="canToggleSchedulable(account)"
              type="button"
              class="relative inline-flex h-5 w-9 flex-shrink-0 cursor-pointer rounded-full border-2 border-transparent transition-colors duration-200 ease-in-out focus:outline-none focus:ring-2 focus:ring-primary-500 focus:ring-offset-2 disabled:cursor-not-allowed disabled:opacity-50 dark:focus:ring-offset-dark-800"
              :class="account.local_account_schedulable ? 'bg-primary-500 hover:bg-primary-600' : 'bg-gray-200 hover:bg-gray-300 dark:bg-dark-600 dark:hover:bg-dark-500'"
              :disabled="togglingSchedulableID === account.local_account_id"
              :aria-pressed="account.local_account_schedulable"
              :title="schedulableToggleTitle(account)"
              @click.stop="handleToggleSchedulable(account)"
            >
              <span
                class="pointer-events-none inline-block h-4 w-4 transform rounded-full bg-white shadow ring-0 transition duration-200 ease-in-out"
                :class="account.local_account_schedulable ? 'translate-x-4' : 'translate-x-0'"
              />
            </button>
            <span v-else class="sp-account-muted">—</span>
          </template>

          <template #cell-local_account_last_test_status="{ row: account }">
            <button
              v-if="isFailedTest(account)"
              type="button"
              class="sp-test-status failed"
              title="查看测试失败详情"
              @click.stop="openTestErrorDialog(account)"
            >
              {{ accountTestStatusLabel(account.local_account_last_test_status) }}
            </button>
            <span
              v-else-if="isMatchedLocalAccount(account) && account.local_account_last_test_status"
              :class="['sp-test-status', accountTestStatusTone(account.local_account_last_test_status)]"
            >
              {{ accountTestStatusLabel(account.local_account_last_test_status) }}
            </span>
            <span v-else class="sp-account-muted">—</span>
          </template>

          <template #cell-local_account_last_tested_at="{ row: account }">
            <span v-if="isMatchedLocalAccount(account)" class="sp-account-time">
              {{ formatTime(account.local_account_last_tested_at) }}
            </span>
            <span v-else class="sp-account-muted">—</span>
          </template>

          <template #cell-supplier_current_balance="{ row: account }">
            <div class="sp-money-cell">
              <strong>{{ formatCNY(account.supplier_current_balance) }}</strong>
              <small>供应商汇总</small>
            </div>
          </template>

          <template #cell-supplier_today_cost="{ row: account }">
            <div class="sp-money-cell cost">
              <strong>{{ formatCNY(account.supplier_today_cost) }}</strong>
              <small>供应商汇总</small>
            </div>
          </template>

          <template #cell-actions="{ row: account }">
            <button
              class="sp-button small ghost sp-account-view-button"
              type="button"
              @click.stop="openDrawer(account)"
            >查看</button>
          </template>

          <template #empty>
            <div class="sp-account-empty">
              <strong>暂无上游账号数据</strong>
              <span>请先同步供应商上游账号，或调整当前筛选条件。</span>
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
      :title="selected?.name || selected?.upstream_account_key || '上游账号详情'"
      eyebrow="ACCOUNT DETAIL"
      @close="selected = null"
    >
      <template v-if="selected">
        <div class="sp-detail-grid">
          <div class="sp-detail-cell"><span>供应商</span><b>{{ displayValue(selected.provider_name) }}</b></div>
          <div class="sp-detail-cell"><span>平台</span><b>{{ selected.platform ? platformLabel(selected.platform) : '—' }}</b></div>
          <div class="sp-detail-cell"><span>上游账号名称</span><b>{{ displayValue(selected.name) }}</b></div>
          <div class="sp-detail-cell"><span>上游 Key</span><b>{{ displayValue(selected.upstream_account_key) }}</b></div>
          <div class="sp-detail-cell"><span>匹配状态</span><b>{{ localAccountMatchLabel(selected) }}</b></div>
          <div class="sp-detail-cell"><span>本地账号</span><b>{{ localAccountDisplayName(selected) }}</b></div>
          <div class="sp-detail-cell"><span>优先级</span><b>{{ localDetailValue(selected, selected.local_account_priority) }}</b></div>
          <div class="sp-detail-cell"><span>上游倍率</span><b>{{ formatRate(selected.rate_multiplier) }}</b></div>
          <div class="sp-detail-cell"><span>账号绑定的分组</span><b>{{ selected.group_name || selected.group_key || '—' }}</b></div>
          <div class="sp-detail-cell"><span>本地账号状态</span><b>{{ isMatchedLocalAccount(selected) ? localAccountStatusLabel(selected.local_account_status) : '—' }}</b></div>
          <div class="sp-detail-cell"><span>是否调度</span><b>{{ localSchedulableLabel(selected) }}</b></div>
          <div class="sp-detail-cell"><span>测试结果</span><b>{{ isMatchedLocalAccount(selected) ? accountTestStatusLabel(selected.local_account_last_test_status) : '—' }}</b></div>
          <div class="sp-detail-cell"><span>上次测试时间</span><b>{{ isMatchedLocalAccount(selected) ? formatTime(selected.local_account_last_tested_at) : '—' }}</b></div>
          <div class="sp-detail-cell"><span>余额（供应商汇总）</span><b>{{ formatCNY(selected.supplier_current_balance) }}</b></div>
          <div class="sp-detail-cell"><span>今日消费（供应商汇总）</span><b>{{ formatCNY(selected.supplier_today_cost) }}</b></div>
          <div class="sp-detail-cell"><span>上游状态</span><b>{{ displayValue(selected.raw_status || selected.status) }}</b></div>
          <div class="sp-detail-cell"><span>最近同步</span><b>{{ formatTime(selected.last_seen_at) }}</b></div>
          <div class="sp-detail-cell"><span>失效时间</span><b>{{ formatTime(selected.inactive_at) }}</b></div>
        </div>
      </template>
    </SupplierDrawer>

    <BaseDialog
      :show="Boolean(testErrorAccount)"
      title="测试失败详情"
      width="normal"
      @close="testErrorAccount = null"
    >
      <div v-if="testErrorAccount" class="sp-test-error-dialog">
        <div class="sp-test-error-meta">
          <span>本地账号</span>
          <strong>{{ displayValue(testErrorAccount.local_account_name) }}</strong>
        </div>
        <div class="sp-test-error-meta">
          <span>上次测试时间</span>
          <strong>{{ formatTime(testErrorAccount.local_account_last_tested_at) }}</strong>
        </div>
        <div class="sp-test-error-message">
          {{ testErrorAccount.local_account_last_test_error || '暂无错误详情' }}
        </div>
      </div>
    </BaseDialog>
  </SupplierModuleLayout>
</template>
<script setup lang="ts">
import { computed, nextTick, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import { SupplierDrawer, SupplierModuleLayout } from '@/components/admin/supplier-management'
import DataTable from '@/components/common/DataTable.vue'
import BaseDialog from '@/components/common/BaseDialog.vue'
import Input from '@/components/common/Input.vue'
import Pagination from '@/components/common/Pagination.vue'
import Select, { type SelectOption } from '@/components/common/Select.vue'
import supplierProvidersAPI, { type SupplierProvider } from '@/api/admin/supplierProviders'
import { listSupplierAccounts, type SupplierProviderAccount } from '@/api/admin/supplierProviderData'
import { adminAPI } from '@/api/admin'
import { useAppStore } from '@/stores/app'
import type { Column } from '@/components/common/types'
import { platformBadgeClass, platformLabel, platformTextClass } from '@/utils/platformColors'
import { extractApiErrorMessage } from '@/utils/apiError'

const appStore = useAppStore()

const providers = ref<SupplierProvider[]>([])
const items = ref<SupplierProviderAccount[]>([])
const selected = ref<SupplierProviderAccount | null>(null)
const testErrorAccount = ref<SupplierProviderAccount | null>(null)
const togglingSchedulableID = ref<number | null>(null)
const editingPriorityAccountID = ref<number | null>(null)
const savingPriorityAccountID = ref<number | null>(null)
const priorityDraft = ref('')
const priorityInput = ref<InstanceType<typeof Input> | null>(null)
const total = ref(0)
const loading = ref(false)
const error = ref('')
const page = ref(1)
const pageSize = ref(20)
const providerID = ref(0)
const platformFilter = ref('')
const activeFilter = ref('true')
const search = ref('')
const searchFilterControl = ref<HTMLElement | null>(null)
const providerFilterControl = ref<HTMLElement | null>(null)
const platformFilterControl = ref<HTMLElement | null>(null)
const activeFilterControl = ref<HTMLElement | null>(null)
let searchTimer: number | undefined

const cnyFormatter = new Intl.NumberFormat('zh-CN', {
  style: 'currency',
  currency: 'CNY',
  minimumFractionDigits: 2,
  maximumFractionDigits: 2,
})

const providerOptions = computed<SelectOption[]>(() => [
  { value: 0, label: '全部供应商' },
  ...providers.value.map(provider => ({ value: provider.id, label: provider.name })),
])
const supplierIDs = computed(() => [...new Set([
  ...providers.value.map(provider => provider.id),
  ...items.value.map(account => account.provider_id),
])].sort((left, right) => left - right))
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
const SUPPLIER_TONES = [
  { chip: 'border-sky-500/30 bg-sky-500/10 text-sky-700 dark:text-sky-300', dot: 'bg-sky-500' },
  { chip: 'border-orange-500/30 bg-orange-500/10 text-orange-700 dark:text-orange-300', dot: 'bg-orange-500' },
  { chip: 'border-emerald-500/30 bg-emerald-500/10 text-emerald-700 dark:text-emerald-300', dot: 'bg-emerald-500' },
  { chip: 'border-violet-500/30 bg-violet-500/10 text-violet-700 dark:text-violet-300', dot: 'bg-violet-500' },
  { chip: 'border-rose-500/30 bg-rose-500/10 text-rose-700 dark:text-rose-300', dot: 'bg-rose-500' },
  { chip: 'border-cyan-500/30 bg-cyan-500/10 text-cyan-700 dark:text-cyan-300', dot: 'bg-cyan-500' },
  { chip: 'border-amber-500/30 bg-amber-500/10 text-amber-700 dark:text-amber-300', dot: 'bg-amber-500' },
  { chip: 'border-indigo-500/30 bg-indigo-500/10 text-indigo-700 dark:text-indigo-300', dot: 'bg-indigo-500' },
  { chip: 'border-lime-500/30 bg-lime-500/10 text-lime-700 dark:text-lime-300', dot: 'bg-lime-500' },
  { chip: 'border-fuchsia-500/30 bg-fuchsia-500/10 text-fuchsia-700 dark:text-fuchsia-300', dot: 'bg-fuchsia-500' },
  { chip: 'border-teal-500/30 bg-teal-500/10 text-teal-700 dark:text-teal-300', dot: 'bg-teal-500' },
  { chip: 'border-red-500/30 bg-red-500/10 text-red-700 dark:text-red-300', dot: 'bg-red-500' },
]
const accountColumns: Column[] = [
  { key: 'provider_name', label: '供应商', class: 'min-w-[190px]' },
  { key: 'upstream_account_key', label: '上游账号', class: 'min-w-[260px]' },
  { key: 'local_account_name', label: '本地账号', class: 'min-w-[190px]' },
  { key: 'local_account_priority', label: '优先级', class: 'min-w-[88px]' },
  { key: 'rate_multiplier', label: '上游倍率', class: 'min-w-[104px]' },
  { key: 'group_name', label: '账号绑定的分组', class: 'min-w-[160px]' },
  { key: 'local_account_status', label: '本地账号状态', class: 'min-w-[136px]' },
  { key: 'local_account_schedulable', label: '是否调度', class: 'min-w-[104px]' },
  { key: 'local_account_last_test_status', label: '测试结果', class: 'min-w-[120px]' },
  { key: 'local_account_last_tested_at', label: '上次测试时间', class: 'min-w-[172px]' },
  { key: 'supplier_current_balance', label: '余额', class: 'min-w-[142px]' },
  { key: 'supplier_today_cost', label: '今日消费', class: 'min-w-[142px]' },
  { key: 'actions', label: '操作', class: 'min-w-[88px]' },
]

onMounted(async () => {
  applyFilterControlLabels()
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

function openDrawer(account: SupplierProviderAccount) {
  selected.value = account
}

function applyFilterControlLabels() {
  setFilterControlLabel(searchFilterControl.value, 'input', 'supplier-account-search-label')
  setFilterControlLabel(providerFilterControl.value, '.select-trigger', 'supplier-account-provider-label')
  setFilterControlLabel(platformFilterControl.value, '.select-trigger', 'supplier-account-platform-label')
  setFilterControlLabel(activeFilterControl.value, '.select-trigger', 'supplier-account-active-label')
}

function setFilterControlLabel(container: HTMLElement | null, selector: string, labelID: string) {
  container?.querySelector<HTMLElement>(selector)?.setAttribute('aria-labelledby', labelID)
}

function isMatchedLocalAccount(account: SupplierProviderAccount): boolean {
  return account.local_account_match_status === 'matched'
}

function supplierTone(providerID: number) {
  const providerIndex = supplierIDs.value.indexOf(providerID)
  const toneIndex = providerIndex >= 0 ? providerIndex : Math.abs(Math.trunc(providerID || 0))
  return SUPPLIER_TONES[toneIndex % SUPPLIER_TONES.length]
}

function localAccountStatusLabel(status?: string): string {
  if (status === 'active') return '正常'
  if (status === 'inactive' || status === 'disabled') return '停用'
  if (status === 'error') return '异常'
  return displayValue(status)
}

function localAccountStatusTone(status?: string): string {
  if (status === 'active') return 'good'
  if (status === 'error') return 'bad'
  return 'neutral'
}

function canEditPriority(account: SupplierProviderAccount): boolean {
  return isMatchedLocalAccount(account)
    && Number.isInteger(account.local_account_id)
    && Number(account.local_account_id) > 0
    && typeof account.local_account_priority === 'number'
}

function startPriorityEdit(account: SupplierProviderAccount) {
  if (!canEditPriority(account)) return
  const localAccountID = Number(account.local_account_id)
  if (savingPriorityAccountID.value === localAccountID) return

  editingPriorityAccountID.value = localAccountID
  priorityDraft.value = String(account.local_account_priority)
  void nextTick(() => {
    priorityInput.value?.focus()
    priorityInput.value?.select()
  })
}

function cancelPriorityEdit() {
  editingPriorityAccountID.value = null
  priorityDraft.value = ''
}

async function savePriority(account: SupplierProviderAccount) {
  if (!canEditPriority(account)) return
  const localAccountID = Number(account.local_account_id)
  if (editingPriorityAccountID.value !== localAccountID) return
  if (savingPriorityAccountID.value === localAccountID) return

  const draft = priorityDraft.value.trim()
  const nextPriority = Number(draft)
  if (draft === '' || !Number.isInteger(nextPriority) || nextPriority < 0) {
    appStore.showError('请输入有效的整数优先级')
    return
  }
  if (nextPriority === account.local_account_priority) {
    cancelPriorityEdit()
    return
  }

  savingPriorityAccountID.value = localAccountID
  try {
    const updated = await adminAPI.accounts.update(localAccountID, { priority: nextPriority })
    const priority = typeof updated?.priority === 'number' ? updated.priority : nextPriority
    items.value = items.value.map(item => item.local_account_id === localAccountID
      ? { ...item, local_account_priority: priority }
      : item)
    if (selected.value?.local_account_id === localAccountID) {
      selected.value = { ...selected.value, local_account_priority: priority }
    }
    cancelPriorityEdit()
    appStore.showSuccess('账号优先级已保存')
  } catch (err) {
    appStore.showError(extractApiErrorMessage(err, '修改账号优先级失败'))
  } finally {
    savingPriorityAccountID.value = null
  }
}

function canToggleSchedulable(account: SupplierProviderAccount): boolean {
  return isMatchedLocalAccount(account)
    && Number.isInteger(account.local_account_id)
    && Number(account.local_account_id) > 0
    && typeof account.local_account_schedulable === 'boolean'
}

function schedulableToggleTitle(account: SupplierProviderAccount): string {
  return account.local_account_schedulable
    ? '当前参与调度，点击停用'
    : '当前不参与调度，点击启用'
}

async function handleToggleSchedulable(account: SupplierProviderAccount) {
  if (!canToggleSchedulable(account)) return
  const localAccountID = Number(account.local_account_id)
  if (togglingSchedulableID.value === localAccountID) return

  const nextSchedulable = !account.local_account_schedulable
  togglingSchedulableID.value = localAccountID
  try {
    const updated = await adminAPI.accounts.setSchedulable(localAccountID, nextSchedulable)
    const schedulable = updated?.schedulable ?? nextSchedulable
    items.value = items.value.map(item => item.local_account_id === localAccountID
      ? { ...item, local_account_schedulable: schedulable }
      : item)
    if (selected.value?.local_account_id === localAccountID) {
      selected.value = { ...selected.value, local_account_schedulable: schedulable }
    }
  } catch (err) {
    appStore.showError(extractApiErrorMessage(err, '修改账号调度状态失败'))
  } finally {
    togglingSchedulableID.value = null
  }
}

function accountTestStatusLabel(status?: string): string {
  if (status === 'testing') return '测试中'
  if (status === 'success') return '成功'
  if (status === 'failed') return '失败'
  return '—'
}

function accountTestStatusTone(status?: string): string {
  if (status === 'testing') return 'testing'
  if (status === 'success') return 'success'
  if (status === 'failed') return 'failed'
  return 'neutral'
}

function isFailedTest(account: SupplierProviderAccount): boolean {
  return isMatchedLocalAccount(account) && account.local_account_last_test_status === 'failed'
}

function openTestErrorDialog(account: SupplierProviderAccount) {
  if (!isFailedTest(account)) return
  testErrorAccount.value = account
}

function displayValue(value?: string | number | null): string {
  if (value === null || value === undefined || value === '') return '—'
  return String(value)
}

function localAccountMatchLabel(account: SupplierProviderAccount): string {
  if (account.local_account_match_status === 'unmatched') return '未匹配'
  if (account.local_account_match_status === 'conflict') {
    return `匹配冲突（${account.local_account_match_count}）`
  }
  if (account.local_account_match_status === 'matched') return '已匹配'
  return '—'
}

function localAccountDisplayName(account: SupplierProviderAccount): string {
  if (!isMatchedLocalAccount(account)) return localAccountMatchLabel(account)
  return displayValue(account.local_account_name)
}

function localDetailValue(
  account: SupplierProviderAccount,
  value?: string | number | null
): string {
  return isMatchedLocalAccount(account) ? displayValue(value) : '—'
}

function localSchedulableLabel(account: SupplierProviderAccount): string {
  if (!isMatchedLocalAccount(account)) return '—'
  if (account.local_account_schedulable === true) return '是'
  if (account.local_account_schedulable === false) return '否'
  return '—'
}

function accountInitial(account: SupplierProviderAccount): string {
  const value = account.name?.trim() || account.upstream_account_key?.trim() || '?'
  return value.slice(0, 1).toUpperCase()
}

function formatRate(value?: number | null): string {
  if (value === null || value === undefined || !Number.isFinite(Number(value))) return '—'
  return `× ${Number(value).toFixed(2)}`
}

function formatCNY(value?: number | null): string {
  if (value === null || value === undefined || !Number.isFinite(Number(value))) return '—'
  return cnyFormatter.format(Number(value))
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

.sp-account-filter-control {
  display: flex;
  min-width: 0;
  flex-direction: column;
  gap: 0.3rem;
}

.sp-account-filter-label {
  color: var(--sp-muted);
  font-size: 0.6875rem;
  font-weight: 700;
  letter-spacing: 0.04em;
  line-height: 1rem;
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
.sp-provider-dot {
  display: inline-block;
  width: 0.45rem;
  height: 0.45rem;
  flex: 0 0 auto;
  border-radius: 9999px;
}

.sp-account-legend i.matched {
  background: var(--sp-green);
  box-shadow: 0 0 0 3px color-mix(in srgb, var(--sp-green) 14%, transparent);
}

.sp-account-legend i.unmatched {
  background: var(--sp-muted);
  box-shadow: 0 0 0 3px color-mix(in srgb, var(--sp-muted) 12%, transparent);
}

.sp-account-legend i.conflict {
  background: var(--sp-amber);
  box-shadow: 0 0 0 3px color-mix(in srgb, var(--sp-amber) 14%, transparent);
}

.sp-account-table-shell {
  min-height: 23rem;
  overflow: hidden;
  background: var(--sp-panel);
}

.sp-account-table-shell :deep(.table-wrapper) {
  overflow-x: auto;
  border: 0;
  border-radius: 0;
  scrollbar-gutter: stable both-edges;
}

.sp-account-table-shell :deep(table) {
  min-width: 118rem;
}

.sp-account-table-shell :deep(.table-header) {
  background: var(--sp-panel-2);
}

.sp-account-table-shell :deep(th) {
  height: 3.25rem;
  border-bottom-color: var(--sp-line);
  color: var(--sp-muted);
  font-size: 0.6875rem;
  letter-spacing: 0.06em;
}

.sp-account-table-shell :deep(td) {
  height: 4.25rem;
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
  gap: 0.625rem;
}

.sp-provider-cell {
  width: fit-content;
  border-width: 1px;
  border-radius: 0.55rem;
  padding: 0.35rem 0.5rem;
}

.sp-provider-cell .sp-entity {
  color: inherit;
}

.sp-account-avatar {
  display: inline-flex;
  width: 2rem;
  height: 2rem;
  flex: 0 0 auto;
  align-items: center;
  justify-content: center;
  border: 1px solid color-mix(in srgb, var(--sp-cyan) 24%, var(--sp-line));
  border-radius: 0.5rem;
  background: color-mix(in srgb, var(--sp-cyan) 8%, var(--sp-panel-2));
  color: var(--sp-cyan);
  font-size: 0.75rem;
  font-weight: 800;
}

.sp-account-copy {
  min-width: 0;
}

.sp-account-meta {
  display: flex;
  min-width: 0;
  align-items: center;
  gap: 0.375rem;
  margin-top: 0.2rem;
}

.sp-platform-badge {
  display: inline-flex;
  flex: 0 0 auto;
  align-items: center;
  border-width: 1px;
  border-radius: 0.3rem;
  padding: 0.08rem 0.35rem;
  font-size: 0.625rem;
  font-weight: 700;
  line-height: 1rem;
}

.sp-account-key {
  max-width: 14rem;
  overflow: hidden;
  font-family: ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, monospace;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.sp-provider-dot {
  box-shadow: 0 0 0 4px color-mix(in srgb, currentColor 10%, transparent);
}

.sp-account-group,
.sp-account-code,
.sp-local-status,
.sp-test-status,
.sp-match-badge {
  display: inline-flex;
  align-items: center;
  border: 1px solid var(--sp-soft);
  border-radius: 0.4rem;
  padding: 0.22rem 0.5rem;
  background: var(--sp-panel-2);
  color: var(--sp-muted);
  font-size: 0.75rem;
  line-height: 1.2;
}

.sp-account-code,
.sp-account-number,
.sp-account-rate,
.sp-money-cell strong {
  font-family: ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, monospace;
}

.sp-account-code {
  text-transform: lowercase;
}

.sp-match-badge.matched {
  border-color: color-mix(in srgb, var(--sp-green) 28%, var(--sp-line));
  background: color-mix(in srgb, var(--sp-green) 8%, var(--sp-panel));
  color: var(--sp-green);
}

.sp-match-badge.unmatched {
  color: var(--sp-muted);
}

.sp-match-badge.conflict {
  border-color: color-mix(in srgb, var(--sp-amber) 32%, var(--sp-line));
  background: color-mix(in srgb, var(--sp-amber) 8%, var(--sp-panel));
  color: var(--sp-amber);
}

.sp-local-account-cell,
.sp-money-cell {
  display: flex;
  flex-direction: column;
  align-items: flex-start;
  gap: 0.25rem;
}

.sp-local-account-cell strong {
  max-width: 11rem;
  overflow: hidden;
  color: var(--sp-text);
  font-size: 0.8125rem;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.sp-account-number,
.sp-account-rate {
  font-size: 0.8125rem;
  font-weight: 700;
}

.sp-account-number {
  color: var(--sp-text);
}

.sp-priority-cell {
  display: inline-flex;
  min-width: 3.75rem;
  align-items: center;
}

.sp-priority-trigger {
  min-width: 2.75rem;
  border: 1px solid transparent;
  border-radius: 0.4rem;
  padding: 0.25rem 0.45rem;
  background: transparent;
  cursor: pointer;
  text-align: center;
  transition: border-color 140ms ease, background-color 140ms ease, color 140ms ease;
}

.sp-priority-trigger:hover,
.sp-priority-trigger:focus-visible {
  border-color: color-mix(in srgb, var(--sp-green) 32%, var(--sp-line));
  background: color-mix(in srgb, var(--sp-green) 8%, var(--sp-panel));
  color: var(--sp-green);
  outline: none;
}

.sp-priority-trigger:disabled {
  cursor: wait;
  opacity: 0.55;
}

.sp-priority-input {
  width: 4.5rem;
}

.sp-priority-input :deep(.input) {
  min-height: 2rem;
  padding: 0.3rem 0.45rem;
  font-family: ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, monospace;
  font-size: 0.8125rem;
  font-weight: 700;
  text-align: center;
}

.sp-local-status.good {
  border-color: color-mix(in srgb, var(--sp-green) 28%, var(--sp-line));
  background: color-mix(in srgb, var(--sp-green) 8%, var(--sp-panel));
  color: var(--sp-green);
}

.sp-local-status.bad {
  border-color: color-mix(in srgb, var(--sp-red) 24%, var(--sp-line));
  background: color-mix(in srgb, var(--sp-red) 8%, var(--sp-panel));
  color: var(--sp-red);
}

.sp-local-status.neutral,
.sp-test-status.neutral {
  color: var(--sp-muted);
}

.sp-test-status {
  font-weight: 700;
}

button.sp-test-status.failed {
  cursor: pointer;
  transition: filter 140ms ease, transform 140ms ease;
}

button.sp-test-status.failed:hover {
  filter: brightness(0.96);
  transform: translateY(-1px);
}

.sp-test-status.testing {
  border-color: color-mix(in srgb, #2563eb 28%, var(--sp-line));
  background: color-mix(in srgb, #2563eb 8%, var(--sp-panel));
  color: #2563eb;
}

.sp-test-status.success {
  border-color: color-mix(in srgb, var(--sp-green) 28%, var(--sp-line));
  background: color-mix(in srgb, var(--sp-green) 8%, var(--sp-panel));
  color: var(--sp-green);
}

.sp-test-status.failed {
  border-color: color-mix(in srgb, var(--sp-red) 28%, var(--sp-line));
  background: color-mix(in srgb, var(--sp-red) 8%, var(--sp-panel));
  color: var(--sp-red);
}

.sp-test-error-dialog {
  display: grid;
  gap: 0.875rem;
}

.sp-test-error-meta {
  display: grid;
  grid-template-columns: 6.5rem 1fr;
  gap: 0.75rem;
  align-items: baseline;
  color: var(--sp-muted);
  font-size: 0.8125rem;
}

.sp-test-error-meta strong {
  color: var(--sp-text);
  font-weight: 700;
}

.sp-test-error-message {
  max-height: 20rem;
  overflow: auto;
  border: 1px solid color-mix(in srgb, var(--sp-red) 24%, var(--sp-line));
  border-radius: 0.625rem;
  padding: 0.875rem;
  background: color-mix(in srgb, var(--sp-red) 6%, var(--sp-panel-2));
  color: var(--sp-text);
  font-family: ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, monospace;
  font-size: 0.75rem;
  line-height: 1.65;
  white-space: pre-wrap;
  word-break: break-word;
}

.sp-account-time,
.sp-account-muted {
  color: var(--sp-muted);
  font-size: 0.8125rem;
}

.sp-money-cell strong {
  color: var(--sp-green);
  font-size: 0.8125rem;
}

.sp-money-cell.cost strong {
  color: var(--sp-amber);
}

.sp-money-cell small {
  color: var(--sp-muted);
  font-size: 0.625rem;
  letter-spacing: 0.04em;
}

.sp-account-view-button {
  min-width: 3.5rem;
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

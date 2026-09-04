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
        <Icon name="link" size="sm" class="sp-monitor-overview-note-icon" />
        <span>绑定关系独立于供应商账号名称和本地分组名称。</span>
      </div>
      <button class="sp-button small sp-overview-auto-match" type="button" :disabled="autoMatching" @click="autoMatch">
        <Icon name="sparkles" size="sm" :class="autoMatching ? 'sp-spin' : ''" />
        <span>自动匹配</span>
      </button>
    </div>

    <!-- 数据表格 -->
    <section class="sp-monitor-table-card">
      <div class="sp-monitor-table-inner">
        <DataTable
          :columns="columns"
          :data="targets"
          :loading="loading"
          row-key="id"
          server-side-sort
          @sort="handleSort"
        >
          <template #cell-provider_name="{ row }">
            <div class="sp-monitor-provider" :class="rowHueClass(row.provider_id)">
              <span class="sp-monitor-provider-tag">
                {{ row.provider_name || `供应商 #${row.provider_id}` }}
              </span>
              <span class="sp-monitor-provider-id">#{{ row.provider_id }}</span>
            </div>
          </template>
          <template #cell-monitor_name="{ row }">
            <div
              class="sp-monitor-target-cell"
              :class="[rowHueClass(row.provider_id), { 'sp-monitor-target-inactive': !!inactiveReason(row) }]"
            >
              <div class="sp-monitor-target-name">
                <span class="sp-monitor-target-chip">{{ row.monitor_name || '未命名监控项' }}</span>
                <span
                  v-if="inactiveReason(row)"
                  class="sp-monitor-inactive-chip"
                  :class="{ 'sp-monitor-inactive-chip-provider': !row.provider_enabled }"
                >{{ inactiveReason(row) }}</span>
              </div>
              <span>
                {{ row.monitor_key }} ·
                <span :class="{ 'sp-monitor-probe-unknown': !row.monitor_provider }">{{ row.monitor_provider || '未知探针' }}</span>
              </span>
            </div>
          </template>
          <template #cell-primary_model="{ row }">
            <span v-if="row.primary_model" class="sp-monitor-model-tag">
              <span class="sp-monitor-model-logo" aria-hidden="true">
                <ModelIcon :model="row.primary_model" size="13px" />
              </span>
              <span class="sp-monitor-model-name">{{ row.primary_model }}</span>
            </span>
            <span v-else class="sp-monitor-muted">—</span>
          </template>
          <template #cell-availability_7d="{ row }">
            <span v-if="row.availability_7d === null" class="sp-monitor-availability-missing">上游未上报</span>
            <div v-else class="sp-monitor-availability-cell">
              <span :class="['sp-monitor-availability', availabilityClass(row.availability_7d)]">
                {{ formatAvailability(row.availability_7d) }}
              </span>
              <span class="sp-monitor-availability-track" title="条形以 95%~100% 为量程">
                <span
                  :class="['sp-monitor-availability-fill', availabilityClass(row.availability_7d)]"
                  :style="{ width: availabilityBarWidth(row.availability_7d) }"
                ></span>
              </span>
            </div>
          </template>
          <template #cell-local_account_name="{ row }">
            <div
              class="sp-monitor-binding"
              :class="[rowHueClass(row.provider_id), { 'sp-monitor-binding-empty': !row.local_account_id }]"
            >
              <Icon name="arrowRight" size="xs" class="sp-monitor-binding-arrow" />
              <div v-if="row.local_account_id" class="sp-monitor-binding-cell">
                <strong class="sp-monitor-binding-name">{{ row.local_account_name }}</strong>
                <span v-if="row.binding_groups.length" class="sp-monitor-binding-groups">{{ groupNames(row.binding_groups) }}</span>
                <span v-else class="sp-monitor-muted">未加入本地分组</span>
              </div>
              <span v-else class="sp-monitor-unbound">未绑定本地账号</span>
            </div>
          </template>
          <template #cell-last_seen_at="{ row }">
            <span
              class="sp-monitor-time"
              :class="timeStatusClass(row.last_seen_at)"
              :title="formatTime(row.last_seen_at)"
            >{{ formatRelativeTime(row.last_seen_at) }}</span>
          </template>
          <template #cell-actions="{ row }">
            <div class="sp-monitor-actions">
              <button class="sp-button small primary" type="button" @click="openBinding(row)">
                <Icon name="link" size="sm" />
                <span>{{ row.local_account_id ? '更换' : '绑定' }}</span>
              </button>
              <button v-if="row.local_account_id" class="sp-button small danger" type="button" @click="unbind(row)">
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
        <div class="sp-binding-filters">
          <div class="sp-binding-filter sp-binding-filter-search">
            <span class="sr-only">搜索本地账号</span>
            <Input v-model="accountSearch" class="w-full" placeholder="搜索本地账号名称或 ID" />
          </div>
          <div class="sp-binding-filter">
            <span class="sr-only">供应商</span>
            <Select
              v-model="accountProviderID"
              class="w-full"
              :options="providerOptions"
              :searchable="true"
              search-placeholder="搜索供应商"
            />
          </div>
          <div class="sp-binding-filter">
            <span class="sr-only">平台</span>
            <Select
              v-model="accountPlatform"
              class="w-full"
              :options="accountPlatformOptions"
              :searchable="false"
            />
          </div>
        </div>
        <p v-if="bindingMatchVariants.length" class="sp-binding-list-caption">
          <Icon name="sort" size="sm" />
          <span>按与监控项名称的相似度排序，越靠前越可能是同一个账号；100% 即自动匹配会直接采用的那一个。</span>
        </p>
        <div class="sp-binding-account-list" :class="{ loading: accountsLoading }">
          <button
            v-for="{ account, score } in rankedLocalAccounts"
            :key="account.id"
            class="sp-binding-account"
            :class="{ selected: selectedAccountID === account.id }"
            type="button"
            @click="selectedAccountID = account.id"
          >
            <span class="sp-binding-account-mark"><Icon name="check" size="sm" /></span>
            <span class="sp-binding-account-copy">
              <strong>{{ account.name }}</strong>
              <span>{{ account.platform || '未设置平台' }} · {{ account.provider_name || '未匹配供应商' }} · {{ account.groups.map(group => group.name).join('、') || '未加入分组' }}</span>
            </span>
            <span v-if="score > 0" class="sp-binding-similarity" :class="`tier-${similarityTierClass(score)}`">{{ similarityLabel(score) }}</span>
            <span v-if="account.id === bindingTarget?.local_account_id" class="sp-binding-current">当前</span>
          </button>
          <div v-if="accountsLoading" class="sp-binding-list-state">
            <Icon name="refresh" size="sm" class="sp-spin" />
            <span>正在加载本地账号…</span>
          </div>
          <div v-else-if="!localAccounts.length" class="sp-binding-list-state">
            {{ accountProviderID ? '该供应商下没有匹配到本地账号，可切换为“全部供应商”再查找。' : '没有找到可绑定的本地账号。' }}
          </div>
          <div v-else-if="accountsTotal > localAccounts.length" class="sp-binding-list-state">
            共 {{ accountsTotal }} 个账号，当前只按名称顺序列出前 {{ localAccounts.length }} 个；相似度只在这些账号之间比较，最匹配的那个可能还没被加载，请用筛选或搜索缩小范围。
          </div>
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
import {
  autoMatchSupplierMonitorTargets,
  bindSupplierMonitorTarget,
  listSupplierBindableLocalAccounts,
  listSupplierMonitorTargets,
  unbindSupplierMonitorTarget,
  type SupplierBindableLocalAccount,
  type SupplierProviderMonitorTarget,
  type SupplierProviderMonitorTargetSort,
} from '@/api/admin/supplierProviderData'
import { customPlatformsAPI, type CustomPlatform } from '@/api/admin/customPlatforms'
import supplierProvidersAPI, { type SupplierProvider } from '@/api/admin/supplierProviders'
import BaseDialog from '@/components/common/BaseDialog.vue'
import DataTable from '@/components/common/DataTable.vue'
import Input from '@/components/common/Input.vue'
import ModelIcon from '@/components/common/ModelIcon.vue'
import Pagination from '@/components/common/Pagination.vue'
import Select, { type SelectOption } from '@/components/common/Select.vue'
import type { Column } from '@/components/common/types'
import Icon from '@/components/icons/Icon.vue'
import { SupplierModuleLayout } from '@/components/admin/supplier-management'
import { useAppStore } from '@/stores/app'
import { formatRelativeTime } from '@/utils/format'
import { buildPlatformOptions, loadPlatformCatalog } from '@/utils/platformOptions'

const appStore = useAppStore()
const targets = ref<SupplierProviderMonitorTarget[]>([])
const providers = ref<SupplierProvider[]>([])
const customPlatforms = ref<CustomPlatform[]>([])
const localAccounts = ref<SupplierBindableLocalAccount[]>([])
const accountsTotal = ref(0)
const bindingTarget = ref<SupplierProviderMonitorTarget | null>(null)
const selectedAccountID = ref<number | null>(null)
const search = ref('')
const accountSearch = ref('')
const accountProviderID = ref<number | null>(null)
const accountPlatform = ref('')
const providerID = ref<number | null>(null)
const activeFilter = ref<boolean | null>(true)
const page = ref(1)
const pageSize = ref(20)
const total = ref(0)
const sortColumn = ref('')
const sortOrder = ref<'asc' | 'desc'>('asc')
const loading = ref(false)
const accountsLoading = ref(false)
const saving = ref(false)

// 列序刻意把「监控项 → 绑定本地账号」放在一起，这两列才是这个页面要对照的一对；
// 主模型 / 可用率 / 最近同步 排在后面作为判断绑定是否合理的证据列。
// 左三列不上列身份色：它们按「行」上色，同一行同色相＝同一条关系链，见样式里的行色相说明。
// 右三列的 sp-monitor-col-* 才是列身份色。
const columns: Column[] = [
  { key: 'provider_name', label: '供应商', sortable: true, class: 'min-w-[170px]' },
  { key: 'monitor_name', label: '监控项', sortable: true, class: 'min-w-[250px]' },
  { key: 'local_account_name', label: '绑定本地账号', sortable: true, class: 'min-w-[260px]' },
  { key: 'primary_model', label: '主模型', class: 'min-w-[145px] sp-monitor-col sp-monitor-col-model' },
  { key: 'availability_7d', label: '7 天可用率', sortable: true, class: 'min-w-[155px] sp-monitor-col sp-monitor-col-availability' },
  { key: 'last_seen_at', label: '最近同步', sortable: true, class: 'min-w-[130px] sp-monitor-col sp-monitor-col-time' },
  { key: 'actions', label: '操作', class: 'min-w-[180px]' },
]

// DataTable 用列 key 发排序事件，后端只认白名单里的排序字段，这里做一层映射。
const sortFieldByColumn: Record<string, SupplierProviderMonitorTargetSort> = {
  provider_name: 'provider',
  monitor_name: 'monitor_name',
  local_account_name: 'binding',
  availability_7d: 'availability',
  last_seen_at: 'last_seen',
}

// 同一行的 供应商标签 / 监控项胶囊 / 绑定账号 共用一个色相，三格连成一条关系链。
// 色相按供应商 ID 取模，样式里的 .sp-monitor-hue-* 与之一一对应。
const ROW_HUE_COUNT = 6
function rowHueClass(providerId: number) {
  return `sp-monitor-hue-${(providerId || 0) % ROW_HUE_COUNT}`
}

const providerOptions = computed<SelectOption[]>(() => [
  { value: null, label: '全部供应商' },
  ...providers.value.map(provider => ({ value: provider.id, label: provider.name }))
])

const activeOptions: SelectOption[] = [
  { value: true, label: '仅显示活跃监控' },
  { value: null, label: '全部监控项' },
]

const accountPlatformOptions = computed<SelectOption[]>(() => [
  { value: '', label: '全部平台' },
  ...buildPlatformOptions(customPlatforms.value),
])

// ===== 候选账号相似度 =====
// 归一化规则和后端自动匹配的 normalizeSupplierMonitorMatchKey 保持一致（只留小写字母、数字、汉字），
// 这样这里算到 100% 的账号，就是自动匹配会直接认定的那一个。
function normalizeMatchKey(value: string) {
  return (value || '').toLowerCase().replace(/[^a-z0-9一-鿿]/g, '')
}

// 监控项名和本地账号名往往只差一个供应商前缀，两边都补一个去前缀的变体再两两比。
// 后端用的是供应商配置里的账号名前缀，接口没下发，这里用供应商名尽量覆盖。
function matchVariants(value: string, prefix: string) {
  const normalized = normalizeMatchKey(value)
  if (!normalized) return []
  const variants = [normalized]
  const normalizedPrefix = normalizeMatchKey(prefix)
  if (normalizedPrefix && normalized.startsWith(normalizedPrefix)) {
    const stripped = normalized.slice(normalizedPrefix.length)
    if (stripped) variants.push(stripped)
  }
  return variants
}

// 二元组 Dice 系数：账号名里常见「破甲gpt1 / gpt破甲1」这种片段重排，
// 编辑距离对重排惩罚过重，二元组重合度更稳。
function bigramSimilarity(left: string, right: string) {
  if (!left || !right) return 0
  if (left === right) return 1
  if (left.length < 2 || right.length < 2) return 0
  const pool = new Map<string, number>()
  for (let i = 0; i < left.length - 1; i++) {
    const gram = left.slice(i, i + 2)
    pool.set(gram, (pool.get(gram) ?? 0) + 1)
  }
  let shared = 0
  for (let i = 0; i < right.length - 1; i++) {
    const gram = right.slice(i, i + 2)
    const available = pool.get(gram) ?? 0
    if (available > 0) {
      pool.set(gram, available - 1)
      shared++
    }
  }
  return (2 * shared) / (left.length + right.length - 2)
}

const bindingMatchVariants = computed(() => {
  const target = bindingTarget.value
  if (!target) return []
  return matchVariants(target.monitor_name, target.provider_name)
})

// 相似度降序把最可能的账号顶到最前；同分再按名称升序，低分区仍然是可浏览的字母序。
const rankedLocalAccounts = computed(() => {
  const monitorKeys = bindingMatchVariants.value
  const prefix = bindingTarget.value?.provider_name || ''
  return localAccounts.value
    .map(account => {
      let score = 0
      for (const accountKey of matchVariants(account.name, prefix)) {
        for (const monitorKey of monitorKeys) {
          score = Math.max(score, bigramSimilarity(monitorKey, accountKey))
        }
      }
      return { account, score }
    })
    .sort(
      (a, b) =>
        b.score - a.score ||
        a.account.name.localeCompare(b.account.name, 'zh-Hans-CN') ||
        a.account.id - b.account.id
    )
})

function similarityLabel(score: number) {
  return `${Math.round(score * 100)}%`
}

function similarityTierClass(score: number) {
  if (score >= 0.8) return 'strong'
  if (score >= 0.5) return 'medium'
  return 'weak'
}

let searchTimer: number | undefined
let accountSearchTimer: number | undefined
let loadedAccountQuery = ''

function accountQuerySignature() {
  return `${accountProviderID.value ?? ''}|${accountPlatform.value}|${accountSearch.value.trim()}`
}

async function loadFilterOptions() {
  try {
    const [result, platformItems] = await Promise.all([
      supplierProvidersAPI.list({ page: 1, page_size: 200 }),
      customPlatformsAPI.list(),
      loadPlatformCatalog(),
    ])
    providers.value = result.items
    customPlatforms.value = platformItems
  } catch (error) {
    appStore.showError(errorMessage(error, '加载筛选选项失败'))
  }
}

async function loadTargets() {
  loading.value = true
  try {
    const result = await listSupplierMonitorTargets({
      provider_id: providerID.value || undefined,
      active: activeFilter.value === null ? undefined : activeFilter.value,
      search: search.value.trim() || undefined,
      sort: sortFieldByColumn[sortColumn.value] || undefined,
      order: sortColumn.value ? sortOrder.value : undefined,
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
  loadedAccountQuery = accountQuerySignature()
  accountsLoading.value = true
  try {
    const result = await listSupplierBindableLocalAccounts({
      provider_id: accountProviderID.value || undefined,
      platform: accountPlatform.value || undefined,
      search: accountSearch.value.trim() || undefined,
      page: 1,
      page_size: 200,
    })
    localAccounts.value = result.items
    accountsTotal.value = result.total
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
  accountProviderID.value = target.provider_id || null
  accountPlatform.value = ''
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
    appStore.showSuccess('账号监控项绑定已保存')
    saving.value = false
    closeBinding()
    await loadTargets()
  } catch (error) {
    appStore.showError(errorMessage(error, '保存账号监控项绑定失败'))
  } finally {
    saving.value = false
  }
}

async function unbind(target: SupplierProviderMonitorTarget) {
  try {
    await unbindSupplierMonitorTarget(target.id)
    appStore.showSuccess('账号监控项绑定已解除')
    await loadTargets()
  } catch (error) {
    appStore.showError(errorMessage(error, '解除账号监控项绑定失败'))
  }
}

function resetFilters() {
  search.value = ''
  providerID.value = null
  activeFilter.value = true
  page.value = 1
  void loadTargets()
}

// 排序在后端做：客户端排序只能重排当前一页，和分页一起用会给出错误的「最差/最旧」。
function handleSort(key: string, order: 'asc' | 'desc') {
  sortColumn.value = sortFieldByColumn[key] ? key : ''
  sortOrder.value = order
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
  return `${Number(value).toFixed(2)}%`
}

function availabilityClass(value: number) {
  if (value >= 99) return 'healthy'
  if (value >= 95) return 'warning'
  return 'danger'
}

// 可用率几乎都挤在 95%~100%，线性 0~100 的条形里 99.2 和 99.9 看起来一样宽。
// 这里把 95~100 拉满整条宽度，低于 95 一律留空，让真正需要对比的区间产生差异。
function availabilityBarWidth(value: number) {
  const ratio = (value - 95) / 5
  return `${Math.min(Math.max(ratio, 0), 1) * 100}%`
}

function formatTime(value: string) {
  if (!value) return '—'
  return new Intl.DateTimeFormat('zh-CN', { dateStyle: 'short', timeStyle: 'short' }).format(new Date(value))
}

// 监控同步任务是 @every 30s，所以「几分钟没动」就已经是异常，阈值必须比人的直觉紧得多，
// 否则整片冻结的行看起来和正常行没区别。
function timeStatusClass(value: string) {
  if (!value) return 'never'
  const timestamp = new Date(value).getTime()
  if (Number.isNaN(timestamp)) return 'never'
  const minutes = (Date.now() - timestamp) / 6e4
  if (minutes > 60) return 'stale'
  if (minutes > 10) return 'warning'
  return ''
}

// 「对不上上游」有两种成因，页面必须分开说：供应商整体不同步了，还是单个监控项被上游删了。
function inactiveReason(row: SupplierProviderMonitorTarget) {
  if (!row.provider_enabled) return '供应商已停用'
  if (!row.active) return '上游已移除'
  return ''
}

const autoMatching = ref(false)

async function autoMatch() {
  if (autoMatching.value) return
  autoMatching.value = true
  try {
    const result = await autoMatchSupplierMonitorTargets(providerID.value || undefined)
    appStore.showSuccess(`自动匹配完成：${result.matched} 个已绑定，${result.ambiguous} 个存在歧义，${result.skipped} 个未匹配`)
    await loadTargets()
  } catch (error) {
    appStore.showError(errorMessage(error, '自动匹配失败'))
  } finally {
    autoMatching.value = false
  }
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

watch([accountSearch, accountProviderID, accountPlatform], () => {
  if (!bindingTarget.value) return
  // openBinding 已按行内供应商拉过一次，跳过初始化筛选带来的重复查询
  if (accountQuerySignature() === loadedAccountQuery) return
  window.clearTimeout(accountSearchTimer)
  accountSearchTimer = window.setTimeout(() => void loadLocalAccounts(), 260)
})

onMounted(async () => {
  await loadFilterOptions()
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
  background: color-mix(in srgb, var(--sp-muted) 10%, transparent);
}

.sp-control-button-reset:hover {
  background: color-mix(in srgb, var(--sp-muted) 16%, transparent);
  color: var(--sp-text);
}

.sp-control-button-refresh {
  color: var(--sp-green);
  background: color-mix(in srgb, var(--sp-green) 10%, transparent);
}

.sp-control-button-refresh:hover {
  background: color-mix(in srgb, var(--sp-green) 16%, transparent);
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
  margin-left: auto;
}

.sp-monitor-overview-note {
  display: flex;
  align-items: center;
  gap: 0.45rem;
  padding: 0.35rem 0.75rem;
  border: 1px solid color-mix(in srgb, var(--sp-cyan) 16%, var(--sp-line));
  border-radius: 9999px;
  background: color-mix(in srgb, var(--sp-cyan) 5%, var(--sp-panel));
  color: var(--sp-muted);
  font-size: 0.8rem;
  line-height: 1.5;
}

.sp-monitor-overview-note .sp-monitor-overview-note-icon {
  color: var(--sp-cyan);
  display: inline-flex;
}

/* ===== 表格卡片 ===== */

/* 概览栏自动匹配按钮 */
/* 自动匹配是「系统按名称批量推算绑定」的动作，用紫色和逐行手动绑定（蓝）区分开 */
.sp-overview-auto-match {
  display: inline-flex;
  align-items: center;
  gap: 0.375rem;
  white-space: nowrap;
  padding: 0.35rem 0.85rem;
  border: 1px solid var(--sp-violet);
  border-radius: 0.5rem;
  background: var(--sp-violet);
  color: #fff;
  font-size: 0.8rem;
  font-weight: 600;
  cursor: pointer;
  transition: border-color 0.16s ease, background 0.16s ease;
}
.sp-overview-auto-match:hover {
  border-color: #6d28d9;
  background: #6d28d9;
}
.dark .sp-overview-auto-match:hover {
  border-color: #8b5cf6;
  background: #8b5cf6;
}
.sp-overview-auto-match:disabled {
  opacity: 0.6;
  pointer-events: none;
}

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

/* ===== 列身份色（只给右侧证据列）=====
   主模型青 / 可用率青绿 / 最近同步灰。左侧三列是关系链本身，颜色按行走而不是按列走。
   th/td 由 DataTable 渲染，不带本组件的 scope 属性，必须走 :deep()。 */
.sp-monitor-table-card :deep(.sp-monitor-col-model) { --col-accent: #0891b2; }
.sp-monitor-table-card :deep(.sp-monitor-col-availability) { --col-accent: #0f766e; }
.sp-monitor-table-card :deep(.sp-monitor-col-time) { --col-accent: #64748b; }
.dark .sp-monitor-table-card :deep(.sp-monitor-col-model) { --col-accent: #22d3ee; }
.dark .sp-monitor-table-card :deep(.sp-monitor-col-availability) { --col-accent: #5eead4; }
.dark .sp-monitor-table-card :deep(.sp-monitor-col-time) { --col-accent: #94a3b8; }

/* 列身份只体现在表头文字和左侧细分隔条上，避免整行变成色块 */
.sp-monitor-table-card :deep(th.sp-monitor-col),
.sp-monitor-table-card :deep(td.sp-monitor-col) {
  box-shadow: inset 1px 0 0 color-mix(in srgb, var(--col-accent) 16%, transparent);
}
.sp-monitor-table-card :deep(th.sp-monitor-col) { color: var(--col-accent); }

/* 主模型是「关系区 → 证据区」的分界，这一道分隔条用实线画满 */
.sp-monitor-table-card :deep(th.sp-monitor-col-model),
.sp-monitor-table-card :deep(td.sp-monitor-col-model) {
  box-shadow: inset 1px 0 0 var(--sp-line);
}

/* ===== 行色相 =====
   同一行的 供应商标签 / 监控项胶囊 / 绑定账号 共用 --row-hue，扫一眼就知道哪三格是一条链；
   色相按供应商 ID 取模，同一个供应商的多个监控项自然同色。
   这里刻意不用 --sp-amber / --sp-red / --sp-green：那三个色在本页有语义（待处理、异常、健康），
   拿来当身份色会和状态色打架，所以身份色单独取一组互不相邻的色相。 */
.sp-monitor-hue-0 { --row-hue: #2563eb; }
.sp-monitor-hue-1 { --row-hue: #db2777; }
.sp-monitor-hue-2 { --row-hue: #0d9488; }
.sp-monitor-hue-3 { --row-hue: #7c3aed; }
.sp-monitor-hue-4 { --row-hue: #0891b2; }
.sp-monitor-hue-5 { --row-hue: #c026d3; }
.dark .sp-monitor-hue-0 { --row-hue: #60a5fa; }
.dark .sp-monitor-hue-1 { --row-hue: #f472b6; }
.dark .sp-monitor-hue-2 { --row-hue: #2dd4bf; }
.dark .sp-monitor-hue-3 { --row-hue: #a78bfa; }
.dark .sp-monitor-hue-4 { --row-hue: #22d3ee; }
.dark .sp-monitor-hue-5 { --row-hue: #e879f9; }

/* 未绑定行整行淡琥珀：分组后未绑定的行连成一片，扫一眼就知道还剩哪些要处理。
   用 background-image 而不是 background-color，DataTable 的 hover 底色才不会被压掉。 */
.sp-monitor-table-card :deep(tr:has(.sp-monitor-binding-empty)) {
  background-image: linear-gradient(
    color-mix(in srgb, var(--sp-amber) 5%, transparent),
    color-mix(in srgb, var(--sp-amber) 5%, transparent)
  );
}

/* ===== 表格单元格 ===== */
.sp-monitor-provider {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  min-width: 0;
}

.sp-monitor-provider-tag {
  display: inline-flex;
  align-items: center;
  max-width: 100%;
  padding: 0.15rem 0.5rem;
  border: 1px solid color-mix(in srgb, var(--row-hue) 26%, var(--sp-line));
  border-radius: 9999px;
  background: color-mix(in srgb, var(--row-hue) 9%, var(--sp-panel));
  color: var(--row-hue);
  font-size: 0.8125rem;
  font-weight: 600;
  line-height: 1.45;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.sp-monitor-provider-id {
  flex-shrink: 0;
  color: var(--sp-muted);
  font-size: 0.7rem;
}

.sp-monitor-target-cell {
  display: flex;
  flex-direction: column;
  gap: 0.25rem;
  min-width: 0;
}

.sp-monitor-target-name {
  display: flex;
  align-items: center;
  gap: 0.4rem;
  min-width: 0;
}

.sp-monitor-target-cell .sp-monitor-target-chip {
  display: inline-flex;
  align-items: center;
  max-width: 100%;
  padding: 0.15rem 0.5rem;
  border: 1px solid color-mix(in srgb, var(--row-hue) 24%, var(--sp-line));
  border-radius: 9999px;
  background: color-mix(in srgb, var(--row-hue) 8%, var(--sp-panel));
  color: var(--row-hue);
  font-size: 0.8125rem;
  font-weight: 600;
  line-height: 1.45;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.sp-monitor-target-inactive .sp-monitor-target-chip {
  border-color: var(--sp-line);
  background: var(--sp-panel-2);
  color: var(--sp-dim);
}

.sp-monitor-target-cell .sp-monitor-inactive-chip {
  flex-shrink: 0;
  padding: 0.1rem 0.45rem;
  border: 1px solid var(--sp-line);
  border-radius: 9999px;
  background: var(--sp-panel-2);
  color: var(--sp-dim);
  font-size: 0.6875rem;
  font-weight: 500;
  line-height: 1.4;
}

/* 上游删掉监控项是正常生命周期，灰色即可；供应商整体停用需要人去处理，用本页的「待处理」琥珀色。 */
.sp-monitor-target-cell .sp-monitor-inactive-chip-provider {
  border-color: color-mix(in srgb, var(--sp-amber) 30%, var(--sp-line));
  background: color-mix(in srgb, var(--sp-amber) 8%, transparent);
  color: var(--sp-amber);
}

.sp-monitor-target-cell span {
  color: var(--sp-muted);
  font-size: 0.72rem;
}

.sp-monitor-target-cell .sp-monitor-probe-unknown {
  color: var(--sp-amber);
  font-weight: 500;
}

/* 主模型：胶囊 + 品牌图标。胶囊底色跟随列身份色（青＝上游探针配置），
   模型家族靠 ModelIcon 的品牌图标区分，不在这里另造一套模型名到颜色的映射。 */
.sp-monitor-model-tag {
  display: inline-flex;
  align-items: center;
  gap: 0.35rem;
  max-width: 100%;
  padding: 0.2rem 0.5rem 0.2rem 0.25rem;
  border: 1px solid color-mix(in srgb, var(--col-accent) 26%, var(--sp-line));
  border-radius: 9999px;
  background: color-mix(in srgb, var(--col-accent) 8%, var(--sp-panel));
  min-width: 0;
}

/* 品牌图标用固定亮底托一下：OpenAI 一类的品牌色是纯黑，直接放在深色面板上会看不见 */
.sp-monitor-model-logo {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  flex-shrink: 0;
  width: 1.125rem;
  height: 1.125rem;
  border-radius: 9999px;
  background: #fff;
}

.sp-monitor-model-name {
  overflow: hidden;
  color: var(--sp-text);
  font-size: 0.75rem;
  font-family: ui-monospace, SFMono-Regular, Consolas, "Liberation Mono", monospace;
  text-overflow: ellipsis;
  white-space: nowrap;
}

/* 可用率 */
.sp-monitor-availability-cell {
  display: flex;
  flex-direction: column;
  gap: 0.3rem;
  min-width: 0;
}

.sp-monitor-availability-track {
  display: block;
  width: 100%;
  max-width: 6rem;
  height: 0.25rem;
  overflow: hidden;
  border-radius: 9999px;
  background: var(--sp-panel-3);
}

.sp-monitor-availability-fill {
  display: block;
  height: 100%;
  border-radius: 9999px;
  transition: width 0.2s ease;
}

.sp-monitor-availability-fill.healthy {
  background: var(--sp-green);
}

.sp-monitor-availability-fill.warning {
  background: var(--sp-amber);
}

.sp-monitor-availability-fill.danger {
  background: var(--sp-red);
}

.sp-monitor-availability-missing {
  color: var(--sp-dim);
  font-size: 0.78rem;
}

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

.sp-monitor-availability::before {
  content: '';
  width: 0.375rem;
  height: 0.375rem;
  flex-shrink: 0;
  border-radius: 50%;
  background: currentColor;
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

/* 绑定：左侧色轨 + 箭头 + 账号名共用行色相，和同一行的监控项胶囊读成「监控项 → 本地账号」一组。
   未绑定时整格翻成琥珀（--row-hue 被 sp-monitor-binding-empty 改写），表示这行还等着处理。 */
.sp-monitor-binding {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  min-width: 0;
  padding-left: 0.55rem;
  border-left: 2px solid color-mix(in srgb, var(--row-hue) 45%, transparent);
}

/* 两个类名一起写是为了压过 .dark .sp-monitor-hue-*（同为两级选择器，靠后声明取胜），
   否则深色模式下未绑定的格子会拿回行色相而不是琥珀。 */
.sp-monitor-binding.sp-monitor-binding-empty {
  --row-hue: var(--sp-amber);
}

.sp-monitor-binding-arrow {
  flex-shrink: 0;
  color: var(--row-hue);
}

.sp-monitor-binding-cell {
  display: flex;
  flex-direction: column;
  gap: 0.2rem;
  min-width: 0;
}

.sp-monitor-binding-name {
  overflow: hidden;
  color: var(--row-hue);
  font-size: 0.875rem;
  font-weight: 600;
  text-overflow: ellipsis;
  white-space: nowrap;
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
  color: var(--sp-amber);
  font-size: 0.8rem;
  font-weight: 500;
  white-space: nowrap;
}

.sp-monitor-time {
  color: var(--sp-muted);
  font-size: 0.8rem;
  font-variant-numeric: tabular-nums;
}

.sp-monitor-time.warning {
  color: var(--sp-amber);
}

.sp-monitor-time.stale {
  color: var(--sp-red);
}

.sp-monitor-time.never {
  color: var(--sp-dim);
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
  --sp-panel-2: #f9fafb;
  --sp-panel-3: #f3f4f6;
  --sp-line: #e5e7eb;
  --sp-soft: #f1f5f9;
  --sp-text: #111827;
  --sp-muted: #64748b;
  --sp-dim: #94a3b8;
  --sp-cyan: #3b82f6;
  --sp-green: #16a34a;
  --sp-amber: #d97706;
  --sp-orange: #ea580c;
  --sp-red: #dc2626;
  --sp-blue: #2563eb;
  --sp-violet: #7c3aed;
  --sp-shadow: 0 1px 2px rgba(15, 23, 42, 0.04);
  color: var(--sp-text);
}

/* 弹窗经 Teleport 挂到 body，暗色主题需在此自行兜底 */
.dark .supplier-monitor-dialog {
  --sp-panel: #1f2937;
  --sp-panel-2: #1f2937;
  --sp-panel-3: #374151;
  --sp-line: #374151;
  --sp-soft: #374151;
  --sp-text: #f9fafb;
  --sp-muted: #9ca3af;
  --sp-dim: #6b7280;
  --sp-shadow: 0 1px 2px rgba(0, 0, 0, 0.18);
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

.sp-binding-filters {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 0.5rem;
  margin-bottom: 0.75rem;
}

.sp-binding-filter {
  min-width: 0;
}

.sp-binding-filter-search {
  grid-column: 1 / -1;
}

@media (max-width: 640px) {
  .sp-binding-filters {
    grid-template-columns: minmax(0, 1fr);
  }
}

.sp-binding-list-caption {
  display: flex;
  align-items: center;
  gap: 0.35rem;
  margin: 0 0 0.55rem;
  color: var(--sp-muted);
  font-size: 0.75rem;
  line-height: 1.5;
}

.sp-binding-list-caption :deep(svg) {
  flex-shrink: 0;
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

.sp-binding-similarity {
  display: inline-flex;
  align-items: center;
  padding: 0.15rem 0.5rem;
  border: 1px solid var(--sp-line);
  border-radius: 9999px;
  font-size: 0.6875rem;
  font-weight: 600;
  flex-shrink: 0;
  font-variant-numeric: tabular-nums;
}

.sp-binding-similarity.tier-strong {
  border-color: color-mix(in srgb, var(--sp-green) 30%, var(--sp-line));
  background: color-mix(in srgb, var(--sp-green) 8%, transparent);
  color: var(--sp-green);
}

.sp-binding-similarity.tier-medium {
  border-color: color-mix(in srgb, var(--sp-amber) 30%, var(--sp-line));
  background: color-mix(in srgb, var(--sp-amber) 8%, transparent);
  color: var(--sp-amber);
}

.sp-binding-similarity.tier-weak {
  border-color: var(--sp-line);
  background: color-mix(in srgb, var(--sp-muted) 6%, transparent);
  color: var(--sp-muted);
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

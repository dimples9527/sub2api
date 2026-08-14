<template>
  <AppLayout>
    <TablePageLayout>
      <template #actions>
        <div class="model-square-config-hero">
          <div class="min-w-0 flex-1">
            <div class="flex items-center gap-3">
              <span class="hero-icon"><Icon name="cube" size="lg" /></span>
              <div>
                <p class="text-xs font-semibold uppercase tracking-[0.25em] text-emerald-600 dark:text-emerald-300">模型配置中心</p>
                <h1 class="mt-1 text-2xl font-semibold text-gray-950 dark:text-white">模型广场配置</h1>
              </div>
            </div>
            <p class="mt-3 max-w-3xl text-sm leading-6 text-gray-600 dark:text-dark-300">
              按平台维护模型广场展示的模型清单。你可以手动录入模型，也可以选择同平台账号同步上游模型后再保存。
            </p>
          </div>
          <div class="grid w-full grid-cols-1 gap-3 sm:grid-cols-3 lg:w-auto lg:min-w-[34rem]">
            <div class="metric-card">
              <span>已配置平台</span>
              <strong>{{ configuredPlatformCount }}</strong>
            </div>
            <div class="metric-card accent-blue">
              <span>模型总数</span>
              <strong>{{ totalModelCount }}</strong>
            </div>
            <div class="metric-card accent-amber">
              <span>最近保存</span>
              <strong class="text-base">{{ formatTime(configUpdatedAt) }}</strong>
            </div>
          </div>
        </div>
      </template>

      <template #filters>
        <div class="space-y-3 rounded-2xl border border-gray-200 bg-white p-4 shadow-sm dark:border-dark-700 dark:bg-dark-800">
          <div class="flex flex-col gap-3 xl:flex-row xl:items-center">
            <div class="grid flex-1 grid-cols-1 gap-3 md:grid-cols-[minmax(14rem,18rem)_minmax(14rem,1fr)]">
              <Select
                v-model="selectedPlatform"
                :options="platformSelectOptions"
                searchable
                :clearable="false"
                placeholder="选择平台"
                aria-label="选择平台"
              >
                <template #selected="{ option }">
                  <span class="inline-flex min-w-0 items-center gap-2">
                    <PlatformIcon :platform="platformIconKey(String(option?.value || selectedPlatform))" size="md" />
                    <span class="truncate">{{ option?.label || currentPlatformLabel }}</span>
                  </span>
                </template>
                <template #option="{ option }">
                  <span class="inline-flex min-w-0 items-center gap-2">
                    <PlatformIcon :platform="platformIconKey(String(option.value))" size="sm" />
                    <span class="truncate">{{ option.label }}</span>
                  </span>
                </template>
              </Select>
              <SearchInput v-model="searchQuery" placeholder="搜索模型 ID 或展示名称" />
            </div>

            <div class="flex flex-wrap items-center gap-2 xl:justify-end">
              <button type="button" class="btn btn-secondary" :disabled="loading" @click="reload">
                <Icon name="refresh" size="sm" :class="loading ? 'animate-spin' : ''" />
                刷新
              </button>
              <button type="button" class="btn btn-secondary" @click="openBatchDialog">
                <Icon name="upload" size="sm" />
                批量添加
              </button>
              <button type="button" class="btn btn-secondary" @click="openSyncDialog">
                <Icon name="sync" size="sm" />
                同步账号模型
              </button>
              <button type="button" class="btn btn-primary" @click="openModelDialog()">
                <Icon name="plus" size="sm" />
                添加模型
              </button>
              <button type="button" class="btn btn-primary" :disabled="saving" @click="saveConfig">
                <Icon name="check" size="sm" />
                {{ saving ? '保存中…' : '保存配置' }}
              </button>
            </div>
          </div>

          <div class="platform-strip">
            <button
              v-for="platform in platformCards"
              :key="platform.platform"
              type="button"
              class="platform-chip"
              :class="{ active: selectedPlatform === platform.platform }"
              @click="selectedPlatform = platform.platform"
            >
              <PlatformIcon :platform="platformIconKey(platform.platform)" size="sm" />
              <span class="truncate">{{ platform.label }}</span>
              <b>{{ platform.modelCount }}</b>
            </button>
          </div>
          <div v-if="referencePricingLoading" class="reference-pricing-status">
            <Icon name="refresh" size="sm" class="animate-spin" />
            正在加载官方参考价格
          </div>
        </div>
      </template>

      <template #table>
        <div v-if="loadError" class="m-4 rounded-xl border border-red-200 bg-red-50 px-4 py-3 text-sm text-red-700 dark:border-red-900/60 dark:bg-red-900/20 dark:text-red-300">
          {{ loadError }}
        </div>

        <DataTable
          v-else
          :columns="columns"
          :data="filteredModels"
          :loading="loading"
          row-key="id"
          sticky-actions-column
        >
          <template #empty>
            <EmptyState
              title="当前平台还没有模型"
              description="可以手动添加模型，或从一个同平台上游账号同步模型列表。"
              action-text="添加模型"
              @action="openModelDialog()"
            />
          </template>

          <template #cell-id="{ value }">
            <code class="model-code">{{ value }}</code>
          </template>

          <template #cell-display_name="{ row, value }">
            <div class="min-w-0">
              <div class="truncate font-medium text-gray-950 dark:text-white">{{ value || row.id }}</div>
              <div v-if="value && value !== row.id" class="mt-1 truncate text-xs text-gray-500 dark:text-dark-400">{{ row.id }}</div>
            </div>
          </template>

          <template #cell-price_summary="{ row }">
            <div v-if="modelPriceGroups(row).length" class="price-groups">
              <div v-for="group in modelPriceGroups(row)" :key="group.title" class="price-group">
                <span class="price-group-title">{{ group.title }}</span>
                <span
                  v-for="item in group.items"
                  :key="item.label"
                  :class="['price-pill', item.source === 'official' ? 'price-pill-reference' : '']"
                >
                  <span>{{ item.label }}</span>
                  <strong>{{ formatPriceValue(item) }}</strong>
                </span>
                <span v-if="group.hasOfficialReference" class="price-reference-badge">官方参考</span>
              </div>
            </div>
            <span v-else class="price-empty">{{ modelPriceEmptyText(row) }}</span>
          </template>

          <template #cell-source="{ value }">
            <span :class="['source-badge', value === 'sync' ? 'source-sync' : 'source-manual']">
              {{ value === 'sync' ? '上游同步' : '手动维护' }}
            </span>
          </template>

          <template #cell-actions="{ row }">
            <div class="flex justify-end gap-2">
              <button type="button" class="btn btn-secondary btn-sm" @click="openModelDialog(row)">
                编辑
              </button>
              <button type="button" class="btn btn-danger btn-sm" @click="askRemoveModel(row)">
                删除
              </button>
            </div>
          </template>
        </DataTable>
      </template>
    </TablePageLayout>

    <BaseDialog :show="modelDialogVisible" :title="editingModelId ? '编辑模型' : '添加模型'" width="wide" @close="closeModelDialog">
      <div class="space-y-4">
        <Input v-model="modelForm.id" label="模型 ID" placeholder="例如：gpt-5.2" required />
        <Input v-model="modelForm.display_name" label="展示名称" placeholder="不填则使用模型 ID" />
        <div class="rounded-xl border border-sky-200 bg-sky-50 px-4 py-3 text-sm text-sky-800 dark:border-sky-500/30 dark:bg-sky-500/10 dark:text-sky-100">
          官方参考价格来自项目动态价格目录：优先同步远程模型价格数据，远程不可用时使用项目内置回退目录。Token 价格在本页面按 USD / 1M Tokens 展示和录入，保存时会转换为计费使用的 USD / Token；获取默认价格只会回填当前未填写的价格字段，编辑时优先回显已维护价格，空字段会使用已加载的官方参考价。
        </div>
        <div class="grid grid-cols-1 gap-3 sm:grid-cols-2">
          <Input
            v-for="price in PRICE_FIELDS"
            :key="price.key"
            v-model="modelForm[price.key]"
            type="number"
            :label="priceInputLabel(price)"
            :placeholder="priceInputPlaceholder(price)"
          />
        </div>
        <div class="flex justify-end">
          <button type="button" class="btn btn-secondary btn-sm" :disabled="defaultPricingLoading" @click="applyDefaultPricing">
            <Icon name="refresh" size="sm" :class="defaultPricingLoading ? 'animate-spin' : ''" />
            {{ defaultPricingLoading ? '查询中…' : '获取默认价格' }}
          </button>
        </div>
      </div>
      <template #footer>
        <button type="button" class="btn btn-secondary" @click="closeModelDialog">取消</button>
        <button type="button" class="btn btn-primary" @click="submitModelDialog">保存</button>
      </template>
    </BaseDialog>

    <BaseDialog :show="batchDialogVisible" title="批量添加模型" width="wide" @close="closeBatchDialog">
      <div class="space-y-4">
        <TextArea
          v-model="batchText"
          label="模型列表"
          rows="10"
          placeholder="每行一个模型 ID，也支持用逗号或空格分隔"
          hint="重复模型会自动跳过，新增模型来源会标记为手动维护。"
        />
      </div>
      <template #footer>
        <button type="button" class="btn btn-secondary" @click="closeBatchDialog">取消</button>
        <button type="button" class="btn btn-primary" @click="submitBatchDialog">添加</button>
      </template>
    </BaseDialog>

    <BaseDialog :show="syncDialogVisible" title="同步上游账号模型" width="wide" @close="closeSyncDialog">
      <div class="space-y-4">
          <div class="rounded-xl border border-emerald-200 bg-emerald-50 px-4 py-3 text-sm text-emerald-800 dark:border-emerald-500/30 dark:bg-emerald-500/10 dark:text-emerald-100">
            将调用所选账号的上游模型列表接口，并把返回模型合并到当前平台配置中。已有模型不会重复添加。
          </div>
          <Select
            v-model="syncAccountId"
            :options="syncAccountOptions"
            :disabled="syncAccountsLoading"
            searchable
            clearable
            :placeholder="syncAccountsLoading ? '正在加载账号' : '选择同平台账号'"
            empty-text="当前平台暂无可选账号"
          aria-label="选择同步账号"
        />
        <div class="grid grid-cols-1 gap-3 sm:grid-cols-3">
          <div class="sync-meta-card">
            <span>当前平台</span>
            <strong>{{ currentPlatformLabel }}</strong>
          </div>
          <div class="sync-meta-card">
            <span>可选账号</span>
            <strong>{{ syncAccountOptions.length }}</strong>
          </div>
          <div class="sync-meta-card">
            <span>上次同步</span>
            <strong>{{ formatTime(currentConfig.synced_at || null) }}</strong>
          </div>
        </div>
      </div>
      <template #footer>
        <button type="button" class="btn btn-secondary" @click="closeSyncDialog">取消</button>
        <button type="button" class="btn btn-primary" :disabled="syncing || !syncAccountId" @click="submitSyncDialog">
          {{ syncing ? '同步中…' : '开始同步' }}
        </button>
      </template>
    </BaseDialog>

    <ConfirmDialog
      :show="Boolean(modelPendingRemove)"
      title="删除模型"
      :message="`确认从 ${currentPlatformLabel} 配置中删除模型 ${modelPendingRemove?.id || ''} 吗？`"
      confirm-text="删除"
      cancel-text="取消"
      danger
      @confirm="confirmRemoveModel"
      @cancel="modelPendingRemove = null"
    />
  </AppLayout>
</template>

<script setup lang="ts">
import { computed, onMounted, ref, watch } from 'vue'
import { adminAPI } from '@/api/admin'
import type {
  ModelSquareConfigPayload,
  ModelSquareOfficialPricing,
  ModelSquarePlatformConfig,
  ModelSquarePlatformModelConfig,
  ModelSquareSyncAccountCandidate,
} from '@/api/admin/modelSquareConfig'
import type { CustomPlatform } from '@/api/admin/customPlatforms'
import type { Account } from '@/types'
import type { Column } from '@/components/common/types'
import type { SelectOption } from '@/components/common/Select.vue'
import { useAppStore } from '@/stores/app'
import { extractApiErrorMessage } from '@/utils/apiError'
import { resolvePlatformDisplayLabel, setCustomPlatformLabels } from '@/utils/customPlatformLabels'
import AppLayout from '@/components/layout/AppLayout.vue'
import TablePageLayout from '@/components/layout/TablePageLayout.vue'
import DataTable from '@/components/common/DataTable.vue'
import EmptyState from '@/components/common/EmptyState.vue'
import BaseDialog from '@/components/common/BaseDialog.vue'
import ConfirmDialog from '@/components/common/ConfirmDialog.vue'
import Input from '@/components/common/Input.vue'
import SearchInput from '@/components/common/SearchInput.vue'
import Select from '@/components/common/Select.vue'
import TextArea from '@/components/common/TextArea.vue'
import PlatformIcon from '@/components/common/PlatformIcon.vue'
import Icon from '@/components/icons/Icon.vue'

const BUILTIN_PLATFORMS = [
  { platform: 'anthropic', name: 'Anthropic' },
  { platform: 'openai', name: 'OpenAI' },
  { platform: 'gemini', name: 'Gemini' },
  { platform: 'antigravity', name: 'Antigravity' },
  { platform: 'grok', name: 'Grok' },
]

const PRICE_FIELDS = [
  { key: 'input_price', label: '输入价格' },
  { key: 'output_price', label: '输出价格' },
  { key: 'cache_write_price', label: '缓存写入价格（5 分钟）' },
  { key: 'cache_write_1h_price', label: '缓存写入价格（1 小时）' },
  { key: 'cache_read_price', label: '缓存读取价格' },
  { key: 'input_price_priority', label: '优先级输入价格' },
  { key: 'output_price_priority', label: '优先级输出价格' },
  { key: 'cache_write_price_priority', label: '优先级缓存写入价格' },
  { key: 'cache_read_price_priority', label: '优先级缓存读取价格' },
  { key: 'image_input_price', label: '图像输入价格' },
  { key: 'image_output_price', label: '图像输出价格' },
  { key: 'per_request_price', label: '按请求价格' },
] as const

const PRICE_PER_MILLION_TOKENS = 1_000_000
const PER_REQUEST_PRICE_FIELD = 'per_request_price'

type PriceField = typeof PRICE_FIELDS[number]['key']
type ModelForm = { id: string; display_name: string } & Record<PriceField, string>
type PriceFieldMeta = typeof PRICE_FIELDS[number]
type PriceItem = { key: PriceField; label: string; value: number; source: 'configured' | 'official' }
type PriceGroup = { title: string; items: PriceItem[]; hasOfficialReference: boolean }
type OfficialPricingStatus = 'loading' | 'found' | 'not_found' | 'error'

const createEmptyModelForm = (): ModelForm => ({
  id: '',
  display_name: '',
  input_price: '',
  output_price: '',
  cache_write_price: '',
  cache_write_1h_price: '',
  cache_read_price: '',
  input_price_priority: '',
  output_price_priority: '',
  cache_write_price_priority: '',
  cache_read_price_priority: '',
  image_input_price: '',
  image_output_price: '',
  per_request_price: '',
})

const columns: Column[] = [
  { key: 'id', label: '模型 ID', sortable: true },
  { key: 'display_name', label: '展示名称', sortable: true },
  { key: 'price_summary', label: '价格（每 1M Tokens）', sortable: false },
  { key: 'source', label: '来源', sortable: true },
  { key: 'actions', label: '操作', sortable: false },
]

const appStore = useAppStore()
const loading = ref(false)
const saving = ref(false)
const syncing = ref(false)
const loadError = ref('')
const selectedPlatform = ref('openai')
const searchQuery = ref('')
const configUpdatedAt = ref<string | null>(null)
const platformConfigs = ref<ModelSquarePlatformConfig[]>([])
const customPlatforms = ref<CustomPlatform[]>([])
const accounts = ref<Account[]>([])
const syncAccounts = ref<ModelSquareSyncAccountCandidate[]>([])
const syncAccountsLoading = ref(false)

const modelDialogVisible = ref(false)
const editingModelId = ref<string | null>(null)
const modelForm = ref<ModelForm>(createEmptyModelForm())
const defaultPricingLoading = ref(false)
const referencePricingLoadingCount = ref(0)
const referencePricingLoading = computed(() => referencePricingLoadingCount.value > 0)
const officialPricingMap = ref<Record<string, ModelSquareOfficialPricing>>({})
const officialPricingStatusMap = ref<Record<string, OfficialPricingStatus>>({})
const batchDialogVisible = ref(false)
const batchText = ref('')
const syncDialogVisible = ref(false)
const syncAccountId = ref<number | null>(null)
const modelPendingRemove = ref<ModelSquarePlatformModelConfig | null>(null)

const normalizePlatform = (platform?: string | null) => (platform || '').trim().toLowerCase()
const normalizeModelId = (id?: string | null) => (id || '').trim()
const modelKey = (id: string) => id.trim().toLowerCase()
const platformIconKey = (platform: string) => platform as any
const referencePricingInFlightKeys = new Set<string>()
const referencePricingPromises = new Map<string, Promise<void>>()

const configuredPlatformCount = computed(() => platformConfigs.value.filter(item => item.models?.length || item.synced_from_account_id).length)
const totalModelCount = computed(() => platformConfigs.value.reduce((sum, item) => sum + (item.models?.length || 0), 0))

const platformLabelMap = computed(() => {
  const labels = new Map<string, string>()
  for (const item of BUILTIN_PLATFORMS) labels.set(item.platform, item.name)
  for (const item of customPlatforms.value) {
    const platform = normalizePlatform(item.code)
    if (!platform || labels.has(platform)) continue
    labels.set(platform, item.name.trim() || resolvePlatformDisplayLabel(platform))
  }
  for (const account of accounts.value) labels.set(normalizePlatform(account.platform), resolvePlatformDisplayLabel(account.platform))
  for (const config of platformConfigs.value) {
    const platform = normalizePlatform(config.platform)
    if (!platform) continue
    labels.set(platform, config.name?.trim() || resolvePlatformDisplayLabel(platform))
  }
  return labels
})

const platformCards = computed(() => {
  const platforms = new Map<string, { platform: string; label: string; modelCount: number; rank: number; order: number }>()
  for (const [index, item] of BUILTIN_PLATFORMS.entries()) {
    platforms.set(item.platform, { platform: item.platform, label: item.name, modelCount: 0, rank: 0, order: index })
  }
  for (const item of [...customPlatforms.value].sort((left, right) => {
    const sortOrderDiff = (left.sort_order ?? 0) - (right.sort_order ?? 0)
    if (sortOrderDiff !== 0) return sortOrderDiff
    return left.name.localeCompare(right.name, 'zh-Hans-CN')
  })) {
    const platform = normalizePlatform(item.code)
    if (!platform || platforms.has(platform)) continue
    platforms.set(platform, {
      platform,
      label: item.name.trim() || resolvePlatformDisplayLabel(platform),
      modelCount: 0,
      rank: 1,
      order: item.sort_order ?? 0,
    })
  }
  for (const account of accounts.value) {
    const platform = normalizePlatform(account.platform)
    if (!platform || platforms.has(platform)) continue
    platforms.set(platform, { platform, label: resolvePlatformDisplayLabel(platform), modelCount: 0, rank: 2, order: 0 })
  }
  for (const config of platformConfigs.value) {
    const platform = normalizePlatform(config.platform)
    if (!platform) continue
    const existing = platforms.get(platform)
    platforms.set(platform, {
      platform,
      label: config.name?.trim() || existing?.label || resolvePlatformDisplayLabel(platform),
      modelCount: config.models?.length || 0,
      rank: existing?.rank ?? 3,
      order: existing?.order ?? 0,
    })
  }
  return Array.from(platforms.values()).sort((left, right) => {
    if (left.rank !== right.rank) return left.rank - right.rank
    if (left.order !== right.order) return left.order - right.order
    return left.label.localeCompare(right.label, 'zh-Hans-CN')
  })
})

const platformSelectOptions = computed<SelectOption[]>(() => platformCards.value.map(item => ({
  value: item.platform,
  label: `${item.label}（${item.modelCount}）`,
})))

const currentPlatformLabel = computed(() => platformLabelMap.value.get(selectedPlatform.value) || resolvePlatformDisplayLabel(selectedPlatform.value))
const currentConfig = computed<ModelSquarePlatformConfig>(() => {
  return platformConfigs.value.find(item => normalizePlatform(item.platform) === selectedPlatform.value) || createPlatformConfig(selectedPlatform.value)
})
const currentModels = computed(() => currentConfig.value.models || [])
const filteredModels = computed(() => {
  const keyword = searchQuery.value.trim().toLowerCase()
  if (!keyword) return currentModels.value
  return currentModels.value.filter(model => `${model.id} ${model.display_name || ''}`.toLowerCase().includes(keyword))
})
const syncAccountOptions = computed<SelectOption[]>(() => syncAccounts.value
  .map(account => ({
    value: account.id,
    label: [
      account.name || `#${account.id}`,
      account.type,
      account.status,
      ...(account.group_names || []),
    ].filter(Boolean).join(' · '),
  })))

function createPlatformConfig(platform: string): ModelSquarePlatformConfig {
  return {
    platform,
    name: platformLabelMap.value.get(platform) || resolvePlatformDisplayLabel(platform),
    synced_from_account_id: null,
    synced_from_account_name: '',
    synced_at: null,
    models: [],
  }
}

function modelPriceValues(model: ModelSquarePlatformModelConfig): Pick<ModelSquarePlatformModelConfig, PriceField> {
  return {
    input_price: model.input_price ?? null,
    output_price: model.output_price ?? null,
    cache_write_price: model.cache_write_price ?? null,
    cache_write_1h_price: model.cache_write_1h_price ?? null,
    cache_read_price: model.cache_read_price ?? null,
    input_price_priority: model.input_price_priority ?? null,
    output_price_priority: model.output_price_priority ?? null,
    cache_write_price_priority: model.cache_write_price_priority ?? null,
    cache_read_price_priority: model.cache_read_price_priority ?? null,
    image_input_price: model.image_input_price ?? null,
    image_output_price: model.image_output_price ?? null,
    per_request_price: model.per_request_price ?? null,
  }
}

function isPerRequestPriceField(key: PriceField): boolean {
  return key === PER_REQUEST_PRICE_FIELD
}

function storedPriceToDisplayPrice(value?: number | null, key?: PriceField): string {
  if (value == null || !Number.isFinite(value)) return ''
  const displayValue = key && isPerRequestPriceField(key) ? value : value * PRICE_PER_MILLION_TOKENS
  return formatPlainPriceNumber(displayValue)
}

function displayPriceToStoredPrice(value: number, key: PriceField): number {
  return isPerRequestPriceField(key) ? value : value / PRICE_PER_MILLION_TOKENS
}

function priceInputLabel(price: PriceFieldMeta): string {
  return isPerRequestPriceField(price.key) ? `${price.label}（USD / 请求）` : `${price.label}（USD / 1M Tokens）`
}

function priceInputPlaceholder(price: PriceFieldMeta): string {
  return isPerRequestPriceField(price.key) ? '例如：0.02' : '例如：5'
}

function formatPlainPriceNumber(value: number): string {
  if (!Number.isFinite(value)) return ''
  return new Intl.NumberFormat('en-US', {
    minimumFractionDigits: Number.isInteger(value) ? 0 : 2,
    maximumFractionDigits: 6,
    useGrouping: false,
  }).format(value)
}

function ensureCurrentConfig(): ModelSquarePlatformConfig {
  const platform = normalizePlatform(selectedPlatform.value)
  let config = platformConfigs.value.find(item => normalizePlatform(item.platform) === platform)
  if (!config) {
    config = createPlatformConfig(platform)
    platformConfigs.value.push(config)
  }
  config.platform = platform
  if (!config.name) config.name = currentPlatformLabel.value
  if (!Array.isArray(config.models)) config.models = []
  return config
}

function normalizeConfig(input: ModelSquareConfigPayload): void {
  configUpdatedAt.value = input.updated_at || null
  platformConfigs.value = (input.platforms || []).map(item => ({
    platform: normalizePlatform(item.platform),
    name: item.name || resolvePlatformDisplayLabel(item.platform),
    synced_from_account_id: item.synced_from_account_id ?? null,
    synced_from_account_name: item.synced_from_account_name || '',
    synced_at: item.synced_at || null,
    models: dedupeModels(item.models || []),
  })).filter(item => item.platform)
}

function dedupeModels(models: ModelSquarePlatformModelConfig[]): ModelSquarePlatformModelConfig[] {
  const seen = new Set<string>()
  const result: ModelSquarePlatformModelConfig[] = []
  for (const model of models) {
    const id = normalizeModelId(model.id)
    if (!id) continue
    const key = modelKey(id)
    if (seen.has(key)) continue
    seen.add(key)
    result.push({
      id,
      display_name: (model.display_name || id).trim(),
      source: model.source === 'sync' ? 'sync' : 'manual',
      ...modelPriceValues(model),
    })
  }
  return result
}

async function reload(): Promise<void> {
  loading.value = true
  loadError.value = ''
  try {
    const [config, customPlatformList, accountPage] = await Promise.all([
      adminAPI.modelSquareConfig.get(),
      adminAPI.customPlatforms.list(false),
      adminAPI.accounts.list(1, 500, { lite: 'true' }),
    ])
    customPlatforms.value = Array.isArray(customPlatformList) ? customPlatformList : []
    setCustomPlatformLabels(customPlatformList)
    normalizeConfig(config)
    accounts.value = Array.isArray(accountPage.items) ? accountPage.items : []
    if (!platformCards.value.some(item => item.platform === selectedPlatform.value)) {
      selectedPlatform.value = platformCards.value[0]?.platform || 'openai'
    }
    void loadCurrentPlatformReferencePricing()
  } catch (err) {
    loadError.value = extractApiErrorMessage(err, '加载模型广场配置失败')
    appStore.showError(loadError.value)
  } finally {
    loading.value = false
  }
}

async function loadSyncAccountsForCurrentPlatform(): Promise<void> {
  const platform = normalizePlatform(selectedPlatform.value)
  syncAccounts.value = []
  if (!platform) return

  syncAccountsLoading.value = true
  try {
    const candidates = await adminAPI.modelSquareConfig.listSyncAccounts(platform)
    if (normalizePlatform(selectedPlatform.value) === platform) {
      syncAccounts.value = Array.isArray(candidates) ? candidates : []
    }
  } catch (err) {
    if (normalizePlatform(selectedPlatform.value) === platform) {
      appStore.showError(extractApiErrorMessage(err, '加载可同步账号失败'))
    }
  } finally {
    if (normalizePlatform(selectedPlatform.value) === platform) {
      syncAccountsLoading.value = false
    }
  }
}

async function saveConfig(): Promise<void> {
  saving.value = true
  try {
    const payload: ModelSquareConfigPayload = {
      platforms: platformConfigs.value
        .map(config => ({
          ...config,
          platform: normalizePlatform(config.platform),
          name: config.name?.trim() || resolvePlatformDisplayLabel(config.platform),
          models: dedupeModels(config.models || []),
        }))
        .filter(config => config.platform),
    }
    const updated = await adminAPI.modelSquareConfig.update(payload)
    normalizeConfig(updated)
    appStore.showSuccess('模型广场配置已保存')
  } catch (err) {
    appStore.showError(extractApiErrorMessage(err, '保存模型广场配置失败'))
  } finally {
    saving.value = false
  }
}

function fillModelFormFromOfficialPricing(pricing: ModelSquareOfficialPricing): void {
  for (const price of PRICE_FIELDS) {
    if (modelForm.value[price.key].trim()) continue
    const officialValue = pricing[price.key as keyof ModelSquareOfficialPricing]
    if (isOfficialReferencePriceValue(officialValue)) {
      modelForm.value[price.key] = storedPriceToDisplayPrice(officialValue, price.key)
    }
  }
}

function openModelDialog(model?: ModelSquarePlatformModelConfig): void {
  editingModelId.value = model?.id || null
  const officialPricing = model ? officialPricingForModel(model) : null
  modelForm.value = {
    id: model?.id || '',
    display_name: model?.display_name && model.display_name !== model.id ? model.display_name : '',
    ...Object.fromEntries(PRICE_FIELDS.map(({ key }) => {
      const configuredValue = model?.[key]
      const officialValue = officialPricing?.[key as keyof ModelSquareOfficialPricing]
      const value = configuredValue != null && Number.isFinite(configuredValue)
        ? configuredValue
        : isOfficialReferencePriceValue(officialValue)
          ? officialValue
          : null
      return [key, storedPriceToDisplayPrice(value, key)]
    })) as Pick<ModelForm, PriceField>,
  }
  modelDialogVisible.value = true

  if (model && !officialPricing && hasMissingConfiguredPrice(model)) {
    void waitForOfficialPricing(model).then(() => {
      if (!modelDialogVisible.value || modelKey(editingModelId.value || '') !== modelKey(model.id)) return
      if (modelKey(normalizeModelId(modelForm.value.id)) !== modelKey(model.id)) return
      const loadedPricing = officialPricingForModel(model)
      if (loadedPricing) fillModelFormFromOfficialPricing(loadedPricing)
    })
  }
}

function closeModelDialog(): void {
  modelDialogVisible.value = false
  editingModelId.value = null
  modelForm.value = createEmptyModelForm()
}

function parseModelFormPrices(): Pick<ModelSquarePlatformModelConfig, PriceField> | null {
  const prices = {} as Pick<ModelSquarePlatformModelConfig, PriceField>
  for (const price of PRICE_FIELDS) {
    const raw = modelForm.value[price.key].trim()
    if (!raw) {
      prices[price.key] = null
      continue
    }
    const value = Number(raw)
    if (!Number.isFinite(value) || value < 0) {
      appStore.showError(`${price.label}必须是非负数字`)
      return null
    }
    prices[price.key] = displayPriceToStoredPrice(value, price.key)
  }
  return prices
}

function isOfficialReferencePriceValue(value: unknown): value is number {
  return typeof value === 'number' && Number.isFinite(value) && value > 0
}

function officialPricingKey(id: string): string {
  return modelKey(id)
}

function officialPricingLookupCandidates(id: string): string[] {
  const normalized = normalizeModelId(id)
  if (!normalized) return []

  const candidates = [normalized]
  const lastSegment = normalized.split('/').pop()?.trim() || ''
  if (lastSegment && modelKey(lastSegment) !== modelKey(normalized)) candidates.push(lastSegment)
  return candidates
}

function officialPricingForModel(model: ModelSquarePlatformModelConfig): ModelSquareOfficialPricing | null {
  return officialPricingMap.value[officialPricingKey(model.id)] || null
}

function officialPricingStatusForModel(model: ModelSquarePlatformModelConfig): OfficialPricingStatus | null {
  return officialPricingStatusMap.value[officialPricingKey(model.id)] || null
}

function hasMissingConfiguredPrice(model: ModelSquarePlatformModelConfig): boolean {
  return PRICE_FIELDS.some(price => model[price.key] == null)
}

function shouldLoadOfficialPricing(model: ModelSquarePlatformModelConfig): boolean {
  const id = normalizeModelId(model.id)
  if (!id || !hasMissingConfiguredPrice(model)) return false
  const key = officialPricingKey(id)
  return !officialPricingMap.value[key] && !referencePricingInFlightKeys.has(key)
}

function waitForOfficialPricing(model: ModelSquarePlatformModelConfig): Promise<void> {
  const key = officialPricingKey(model.id)
  if (officialPricingMap.value[key] || !shouldLoadOfficialPricing(model)) {
    return referencePricingPromises.get(key) || Promise.resolve()
  }
  return loadOfficialPricingForModels([model])
}

async function loadOfficialPricingForModels(models: ModelSquarePlatformModelConfig[]): Promise<void> {
  const requestModels = models
    .map(model => ({ ...model, id: normalizeModelId(model.id) }))
    .filter(model => shouldLoadOfficialPricing(model))
  if (requestModels.length === 0) return

  const requestKeys = requestModels.map(model => officialPricingKey(model.id))
  for (const key of requestKeys) referencePricingInFlightKeys.add(key)
  officialPricingStatusMap.value = {
    ...officialPricingStatusMap.value,
    ...Object.fromEntries(requestKeys.map(key => [key, 'loading' as OfficialPricingStatus])),
  }
  referencePricingLoadingCount.value += 1

  const pricingPromise = (async () => {
    try {
      const pricingResults = await Promise.all(requestModels.map(async model => {
        try {
          let lastPricing: ModelSquareOfficialPricing | null = null
          for (const candidate of officialPricingLookupCandidates(model.id)) {
            const pricing = await adminAPI.modelSquareConfig.getModelPricing(candidate)
            lastPricing = pricing
            if (pricing.found) return { id: model.id, pricing, failed: false }
          }
          return { id: model.id, pricing: lastPricing, failed: false }
        } catch {
          return { id: model.id, pricing: null, failed: true }
        }
      }))

      const nextPricingMap = { ...officialPricingMap.value }
      const nextStatusMap = { ...officialPricingStatusMap.value }
      for (const { id, pricing, failed } of pricingResults) {
        const key = officialPricingKey(id)
        if (pricing?.found) {
          nextPricingMap[key] = pricing
          nextStatusMap[key] = 'found'
        } else {
          delete nextPricingMap[key]
          nextStatusMap[key] = failed ? 'error' : 'not_found'
        }
      }
      officialPricingMap.value = nextPricingMap
      officialPricingStatusMap.value = nextStatusMap
    } finally {
      for (const key of requestKeys) {
        referencePricingInFlightKeys.delete(key)
        referencePricingPromises.delete(key)
      }
      referencePricingLoadingCount.value = Math.max(0, referencePricingLoadingCount.value - 1)
    }
  })()
  for (const key of requestKeys) referencePricingPromises.set(key, pricingPromise)
  await pricingPromise
}

async function loadCurrentPlatformReferencePricing(): Promise<void> {
  const config = platformConfigs.value.find(item => normalizePlatform(item.platform) === selectedPlatform.value)
  if (!config) return
  await loadOfficialPricingForModels(config.models)
}

async function applyDefaultPricing(): Promise<void> {
  const id = normalizeModelId(modelForm.value.id)
  if (!id) {
    appStore.showError('请先填写模型 ID')
    return
  }
  defaultPricingLoading.value = true
  try {
    const pricing = await adminAPI.modelSquareConfig.getModelPricing(id)
    if (!pricing.found) {
      appStore.showError('未找到该模型的默认价格')
      return
    }
    let filled = 0
    for (const price of PRICE_FIELDS) {
      const current = modelForm.value[price.key].trim()
      const defaultValue = pricing[price.key as keyof ModelSquareOfficialPricing]
      if (!current && isOfficialReferencePriceValue(defaultValue)) {
        modelForm.value[price.key] = storedPriceToDisplayPrice(defaultValue, price.key)
        filled += 1
      }
    }
    appStore.showSuccess(filled > 0 ? `已回填 ${filled} 项默认价格` : '当前价格字段已全部填写')
  } catch (err) {
    appStore.showError(extractApiErrorMessage(err, '获取默认价格失败'))
  } finally {
    defaultPricingLoading.value = false
  }
}

function submitModelDialog(): void {
  const id = normalizeModelId(modelForm.value.id)
  if (!id) {
    appStore.showError('请填写模型 ID')
    return
  }
  const config = ensureCurrentConfig()
  const duplicate = config.models.find(model => modelKey(model.id) === modelKey(id) && modelKey(model.id) !== modelKey(editingModelId.value || ''))
  if (duplicate) {
    appStore.showError('当前平台已存在相同模型')
    return
  }
  const displayName = normalizeModelId(modelForm.value.display_name) || id
  const prices = parseModelFormPrices()
  if (!prices) return
  const nextModel: ModelSquarePlatformModelConfig = { id, display_name: displayName, source: 'manual', ...prices }
  const index = config.models.findIndex(model => modelKey(model.id) === modelKey(editingModelId.value || ''))
  if (index >= 0) config.models.splice(index, 1, nextModel)
  else config.models.push(nextModel)
  closeModelDialog()
  void loadCurrentPlatformReferencePricing()
}

function openBatchDialog(): void {
  batchText.value = ''
  batchDialogVisible.value = true
}

function closeBatchDialog(): void {
  batchDialogVisible.value = false
  batchText.value = ''
}

function submitBatchDialog(): void {
  const ids = Array.from(new Set(batchText.value.split(/[\s,，]+/).map(normalizeModelId).filter(Boolean)))
  if (ids.length === 0) {
    appStore.showError('请至少填写一个模型 ID')
    return
  }
  const config = ensureCurrentConfig()
  const existing = new Set(config.models.map(model => modelKey(model.id)))
  let added = 0
  for (const id of ids) {
    if (existing.has(modelKey(id))) continue
    config.models.push({ id, display_name: id, source: 'manual', ...modelPriceValues({ id }) })
    existing.add(modelKey(id))
    added += 1
  }
  appStore.showSuccess(`已添加 ${added} 个模型`)
  closeBatchDialog()
  void loadCurrentPlatformReferencePricing()
}

function openSyncDialog(): void {
  syncAccountId.value = null
  syncDialogVisible.value = true
  void loadSyncAccountsForCurrentPlatform()
}

function closeSyncDialog(): void {
  if (syncing.value) return
  syncDialogVisible.value = false
  syncAccountId.value = null
  syncAccounts.value = []
}

async function submitSyncDialog(): Promise<void> {
  if (!syncAccountId.value) {
    appStore.showError('请选择要同步的账号')
    return
  }
  const account = syncAccounts.value.find(item => item.id === syncAccountId.value)
  syncing.value = true
  try {
    const result = await adminAPI.accounts.syncUpstreamModels(syncAccountId.value)
    const models = Array.isArray(result.models) ? result.models : []
    const config = ensureCurrentConfig()
    const existing = new Set(config.models.map(model => modelKey(model.id)))
    const addedModelIds: string[] = []
    let added = 0
    for (const raw of models) {
      const id = normalizeModelId(raw)
      if (!id || existing.has(modelKey(id))) continue
      config.models.push({ id, display_name: id, source: 'sync', ...modelPriceValues({ id }) })
      existing.add(modelKey(id))
      addedModelIds.push(id)
      added += 1
    }
    await loadOfficialPricingForModels(addedModelIds.map(id => ({ id, source: 'sync', ...modelPriceValues({ id }) })))
    config.synced_from_account_id = syncAccountId.value
    config.synced_from_account_name = account?.name || ''
    config.synced_at = new Date().toISOString()
    appStore.showSuccess(`同步完成，新增 ${added} 个模型`)
    closeSyncDialog()
  } catch (err) {
    appStore.showError(extractApiErrorMessage(err, '同步上游模型失败'))
  } finally {
    syncing.value = false
  }
}

function askRemoveModel(model: ModelSquarePlatformModelConfig): void {
  modelPendingRemove.value = model
}

function confirmRemoveModel(): void {
  if (!modelPendingRemove.value) return
  const config = ensureCurrentConfig()
  config.models = config.models.filter(model => modelKey(model.id) !== modelKey(modelPendingRemove.value?.id || ''))
  modelPendingRemove.value = null
}

function formatPriceValue(item: PriceItem): string {
  if (item.value == null || !Number.isFinite(item.value)) return ''
  if (isPerRequestPriceField(item.key)) return formatPlainPriceNumber(item.value)
  return `$${formatPlainPriceNumber(item.value * PRICE_PER_MILLION_TOKENS)}`
}

function pricingValue(model: ModelSquarePlatformModelConfig, officialPricing: ModelSquareOfficialPricing | null, key: PriceField): PriceItem | null {
  const configuredValue = model[key]
  if (configuredValue != null && Number.isFinite(configuredValue)) {
    return { key, value: configuredValue, label: '', source: 'configured' }
  }
  const officialValue = officialPricing?.[key as keyof ModelSquareOfficialPricing]
  if (isOfficialReferencePriceValue(officialValue)) {
    return { key, value: officialValue, label: '', source: 'official' }
  }
  return null
}

function pricedItems(model: ModelSquarePlatformModelConfig, officialPricing: ModelSquareOfficialPricing | null, items: Array<{ label: string; key: PriceField }>): PriceItem[] {
  return items.flatMap(item => {
    const price = pricingValue(model, officialPricing, item.key)
    return price ? [{ ...price, label: item.label }] : []
  })
}

function modelPriceGroups(model: ModelSquarePlatformModelConfig): PriceGroup[] {
  const officialPricing = officialPricingForModel(model)
  return [
    {
      title: '基础',
      items: pricedItems(model, officialPricing, [
        { label: '输入', key: 'input_price' },
        { label: '输出', key: 'output_price' },
      ]),
    },
    {
      title: '缓存',
      items: pricedItems(model, officialPricing, [
        { label: '写 5m', key: 'cache_write_price' },
        { label: '写 1h', key: 'cache_write_1h_price' },
        { label: '读取', key: 'cache_read_price' },
      ]),
    },
    {
      title: '优先级',
      items: pricedItems(model, officialPricing, [
        { label: '输入', key: 'input_price_priority' },
        { label: '输出', key: 'output_price_priority' },
        { label: '写入', key: 'cache_write_price_priority' },
        { label: '读取', key: 'cache_read_price_priority' },
      ]),
    },
    {
      title: '图像/请求',
      items: pricedItems(model, officialPricing, [
        { label: '图入', key: 'image_input_price' },
        { label: '图出', key: 'image_output_price' },
        { label: '请求', key: 'per_request_price' },
      ]),
    },
  ].map(group => ({
    ...group,
    hasOfficialReference: group.items.some(item => item.source === 'official'),
  })).filter(group => group.items.length > 0)
}

function modelPriceEmptyText(model: ModelSquarePlatformModelConfig): string {
  switch (officialPricingStatusForModel(model)) {
    case 'loading':
      return '正在查询官方参考价'
    case 'not_found':
      return '官方目录无价格'
    case 'error':
      return '官方价格查询失败'
    default:
      return '未设置'
  }
}

function formatTime(value?: string | null): string {
  if (!value) return '未保存'
  const date = new Date(value)
  if (Number.isNaN(date.getTime())) return '未保存'
  return date.toLocaleString('zh-CN', { hour12: false })
}

watch(selectedPlatform, () => {
  searchQuery.value = ''
  syncAccountId.value = null
  syncAccounts.value = []
  if (syncDialogVisible.value) {
    void loadSyncAccountsForCurrentPlatform()
  }
  void loadCurrentPlatformReferencePricing()
})

onMounted(() => {
  void reload()
})
</script>

<style scoped>
.model-square-config-hero {
  @apply flex flex-col gap-4 rounded-3xl border border-emerald-200/70 bg-gradient-to-br from-white via-emerald-50/80 to-sky-50/80 p-5 shadow-sm dark:border-emerald-500/20 dark:from-dark-800 dark:via-emerald-950/20 dark:to-sky-950/20 lg:flex-row lg:items-stretch lg:justify-between;
}

.hero-icon {
  @apply inline-flex h-12 w-12 items-center justify-center rounded-2xl bg-emerald-500 text-white shadow-lg shadow-emerald-500/20;
}

.metric-card {
  @apply rounded-2xl border border-gray-200 bg-white/85 px-4 py-3 shadow-sm backdrop-blur dark:border-dark-700 dark:bg-dark-900/70;
}

.metric-card span,
.sync-meta-card span {
  @apply text-xs font-medium text-gray-500 dark:text-dark-400;
}

.metric-card strong {
  @apply mt-2 block text-2xl font-semibold text-gray-950 dark:text-white;
}

.metric-card.accent-blue {
  @apply border-sky-200 bg-sky-50/80 dark:border-sky-500/30 dark:bg-sky-500/10;
}

.metric-card.accent-amber {
  @apply border-amber-200 bg-amber-50/80 dark:border-amber-500/30 dark:bg-amber-500/10;
}

.platform-strip {
  @apply flex gap-2 overflow-x-auto pb-1;
}

.platform-chip {
  @apply inline-flex max-w-56 shrink-0 items-center gap-2 rounded-full border border-gray-200 bg-gray-50 px-3 py-2 text-sm text-gray-700 transition hover:border-emerald-300 hover:bg-emerald-50 dark:border-dark-700 dark:bg-dark-900 dark:text-dark-200 dark:hover:border-emerald-500/40 dark:hover:bg-emerald-500/10;
}

.platform-chip.active {
  @apply border-emerald-400 bg-emerald-100 text-emerald-900 shadow-sm dark:border-emerald-500/60 dark:bg-emerald-500/20 dark:text-emerald-100;
}

.platform-chip b {
  @apply rounded-full bg-white px-2 py-0.5 text-xs text-gray-600 dark:bg-dark-800 dark:text-dark-300;
}

.reference-pricing-status {
  @apply inline-flex items-center gap-2 rounded-xl border border-sky-200 bg-sky-50 px-3 py-2 text-xs font-medium text-sky-700 dark:border-sky-500/30 dark:bg-sky-500/10 dark:text-sky-200;
}

.model-code {
  @apply inline-flex max-w-md rounded-lg border border-gray-200 bg-gray-50 px-2 py-1 font-mono text-xs text-gray-800 dark:border-dark-700 dark:bg-dark-900 dark:text-gray-100;
}

.price-groups {
  @apply flex max-w-3xl flex-wrap gap-2;
}

.price-group {
  @apply inline-flex max-w-full flex-wrap items-center gap-1 rounded-xl border border-gray-200 bg-gray-50 px-2 py-1 dark:border-dark-700 dark:bg-dark-900/70;
}

.price-group-title {
  @apply mr-1 text-xs font-semibold text-gray-500 dark:text-dark-400;
}

.price-pill {
  @apply inline-flex items-center gap-1 rounded-lg bg-white px-2 py-0.5 text-xs text-gray-600 shadow-sm dark:bg-dark-800 dark:text-dark-300;
}

.price-pill-reference {
  @apply bg-sky-50 text-sky-700 ring-1 ring-inset ring-sky-200 dark:bg-sky-500/10 dark:text-sky-200 dark:ring-sky-500/30;
}

.price-pill strong {
  @apply font-mono font-semibold text-gray-950 dark:text-white;
}

.price-reference-badge {
  @apply rounded-full bg-sky-100 px-2 py-0.5 text-[11px] font-medium text-sky-700 dark:bg-sky-500/15 dark:text-sky-200;
}

.price-empty {
  @apply text-xs text-gray-400 dark:text-dark-500;
}

.source-badge {
  @apply inline-flex rounded-full px-2.5 py-1 text-xs font-medium;
}

.source-sync {
  @apply bg-sky-500/10 text-sky-700 dark:bg-sky-500/15 dark:text-sky-200;
}

.source-manual {
  @apply bg-emerald-500/10 text-emerald-700 dark:bg-emerald-500/15 dark:text-emerald-200;
}

.sync-meta-card {
  @apply rounded-xl border border-gray-200 bg-gray-50 px-4 py-3 dark:border-dark-700 dark:bg-dark-900/60;
}

.sync-meta-card strong {
  @apply mt-1 block truncate text-sm font-semibold text-gray-950 dark:text-white;
}
</style>

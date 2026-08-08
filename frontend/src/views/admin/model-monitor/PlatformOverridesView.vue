<template>
  <AppLayout>
    <TablePageLayout>
      <template #actions>
        <div class="flex flex-col gap-4 rounded-2xl border border-gray-200 bg-white p-4 shadow-sm dark:border-dark-700 dark:bg-dark-800 lg:flex-row lg:items-stretch lg:justify-between">
          <div class="grid flex-1 grid-cols-1 gap-3 sm:grid-cols-3">
            <div class="rounded-xl border border-gray-200 bg-gray-50/80 px-4 py-3 dark:border-dark-700 dark:bg-dark-900/50">
              <p class="text-xs font-medium uppercase tracking-wide text-gray-500 dark:text-dark-400">分组总数</p>
              <p class="mt-2 text-2xl font-semibold text-gray-900 dark:text-white">{{ totalCount }}</p>
            </div>
            <div class="rounded-xl border border-amber-200 bg-amber-50/80 px-4 py-3 dark:border-amber-500/30 dark:bg-amber-500/10">
              <p class="text-xs font-medium uppercase tracking-wide text-amber-700 dark:text-amber-300">已配置实际平台</p>
              <p class="mt-2 text-2xl font-semibold text-amber-800 dark:text-amber-200">{{ overrideCount }}</p>
            </div>
            <div class="rounded-xl border border-emerald-200 bg-emerald-50/80 px-4 py-3 dark:border-emerald-500/30 dark:bg-emerald-500/10">
              <p class="text-xs font-medium uppercase tracking-wide text-emerald-700 dark:text-emerald-300">默认继承原平台</p>
              <p class="mt-2 text-2xl font-semibold text-emerald-800 dark:text-emerald-200">{{ inheritedCount }}</p>
            </div>
          </div>

          <div class="flex flex-wrap items-center gap-2 lg:flex-shrink-0 lg:justify-end">
            <button class="btn btn-secondary" :disabled="loading" @click="reload">
              {{ loading ? '刷新中…' : '刷新' }}
            </button>
          </div>
        </div>
      </template>

      <template #filters>
        <div class="flex flex-col gap-3 lg:flex-row lg:items-center">
          <div class="w-full lg:max-w-md">
            <SearchInput
              v-model="searchQuery"
              placeholder="搜索分组名称、原平台或实际平台"
            />
          </div>
          <Select
            v-model="platformFilter"
            :options="platformFilterOptions"
            placeholder="全部平台"
            class="w-44"
            :clearable="false"
            @change="applyFilters"
          />
          <div class="text-sm text-gray-500 dark:text-dark-400">
            监控页展示优先使用实际平台；未配置时回退原分组平台。
          </div>
        </div>
      </template>

      <template #table>
        <DataTable
          :columns="columns"
          :data="filteredGroups"
          :loading="loading"
          row-key="id"
        >
          <template #cell-name="{ row, value }">
            <div class="min-w-0">
              <div class="truncate font-medium text-gray-900 dark:text-white">{{ value }}</div>
              <div class="mt-1 text-xs text-gray-500 dark:text-dark-400">ID：{{ row.id }}</div>
            </div>
          </template>

          <template #cell-platform="{ value }">
            <span :class="['inline-flex items-center rounded-md border px-2 py-0.5 text-xs font-medium', platformBadgeClass(value)]">
              {{ platformText(value) }}
            </span>
          </template>

          <template #cell-actual_platform="{ row, value }">
            <span v-if="value" :class="['inline-flex items-center rounded-md border px-2 py-0.5 text-xs font-medium', platformBadgeClass(value)]">
              {{ platformText(value) }}
            </span>
            <span v-else class="text-sm text-gray-400 dark:text-dark-500">未配置</span>
            <div v-if="row.actual_platform" class="mt-1 text-xs text-gray-500 dark:text-dark-400">
              覆盖原平台：{{ platformText(row.platform) }}
            </div>
          </template>

          <template #cell-effective_platform="{ row, value }">
            <span :class="['inline-flex items-center rounded-md border px-2 py-0.5 text-xs font-medium', platformBadgeClass(value)]">
              {{ row.effective_platform_name || platformText(value) }}
            </span>
          </template>

          <template #cell-override_state="{ row }">
            <span
              :class="[
                'inline-flex items-center rounded-full px-2 py-0.5 text-xs font-medium',
                row.actual_platform
                  ? 'bg-amber-500/10 text-amber-700 dark:bg-amber-500/15 dark:text-amber-200'
                  : 'bg-gray-500/10 text-gray-600 dark:bg-dark-700 dark:text-dark-300'
              ]"
            >
              {{ row.actual_platform ? '已配置' : '默认继承' }}
            </span>
          </template>

          <template #cell-rate_multiplier="{ value }">
            <span class="font-mono text-sm text-gray-900 dark:text-gray-100">{{ formatRateMultiplier(value) }}</span>
          </template>

          <template #cell-actions="{ row }">
            <div class="flex flex-wrap justify-end gap-2">
              <button class="btn btn-secondary btn-sm" :disabled="savingGroupId === row.id" @click="openEditDialog(row)">
                编辑
              </button>
              <button
                class="btn btn-secondary btn-sm text-rose-600 hover:text-rose-700 disabled:opacity-50 dark:text-rose-300 dark:hover:text-rose-200"
                :disabled="savingGroupId === row.id || !row.actual_platform"
                @click="askClearOverride(row)"
              >
                清除
              </button>
            </div>
          </template>

          <template #empty>
            <EmptyState
              title="没有匹配的分组"
              description="请尝试调整搜索词或平台筛选条件。"
            />
          </template>
        </DataTable>
      </template>
    </TablePageLayout>

    <BaseDialog
      :show="showEditDialog"
      title="设置分组实际平台"
      width="normal"
      @close="closeEditDialog"
    >
      <div v-if="editingGroup" class="space-y-4">
        <div class="rounded-xl border border-gray-200 bg-gray-50 px-4 py-3 text-sm text-gray-700 dark:border-dark-700 dark:bg-dark-900 dark:text-dark-300">
          <div class="flex items-center justify-between gap-3">
            <span class="font-medium text-gray-900 dark:text-white">{{ editingGroup.name }}</span>
            <span :class="['inline-flex items-center rounded-md border px-2 py-0.5 text-xs font-medium', platformBadgeClass(editingGroup.effective_platform)]">
              当前展示：{{ platformText(editingGroup.effective_platform) }}
            </span>
          </div>
          <div class="mt-2 grid gap-1 text-xs text-gray-500 dark:text-dark-400">
            <div>原平台：{{ platformText(editingGroup.platform) }}</div>
            <div>当前实际平台：{{ editingGroup.actual_platform ? platformText(editingGroup.actual_platform) : '未配置' }}</div>
          </div>
        </div>

        <div>
          <label class="input-label">实际平台</label>
          <Select
            v-model="platformDraft"
            :options="platformOptions"
            placeholder="选择实际平台，不选则沿用原平台"
            class="w-full"
            :clearable="true"
          />
        </div>

        <p class="text-xs leading-5 text-gray-500 dark:text-dark-400">
          该配置只影响模型监控页面的展示和筛选，不会修改原有分组平台字段。
        </p>
      </div>

      <template #footer>
        <div class="flex items-center justify-between gap-3">
          <button
            v-if="editingGroup?.actual_platform"
            class="btn btn-secondary text-rose-600 hover:text-rose-700 dark:text-rose-300 dark:hover:text-rose-200"
            :disabled="savingGroupId === editingGroup.id"
            @click="askClearOverride(editingGroup)"
          >
            清除配置
          </button>
          <div class="ml-auto flex items-center gap-3">
            <button class="btn btn-secondary" :disabled="savingGroupId !== null" @click="closeEditDialog">取消</button>
            <button class="btn btn-primary" :disabled="savingGroupId !== null" @click="saveOverride">
              {{ savingGroupId !== null ? '保存中…' : '保存' }}
            </button>
          </div>
        </div>
      </template>
    </BaseDialog>

    <ConfirmDialog
      :show="showClearConfirm"
      title="清除实际平台配置"
      :message="clearConfirmMessage"
      confirm-text="清除"
      cancel-text="取消"
      :danger="true"
      @confirm="confirmClearOverride"
      @cancel="closeClearConfirm"
    />
  </AppLayout>
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { useAppStore } from '@/stores/app'
import { extractApiErrorMessage } from '@/utils/apiError'
import { platformBadgeClass } from '@/utils/platformColors'
import { adminAPI } from '@/api/admin'
import { resolvePlatformDisplayLabel, setCustomPlatformLabels } from '@/utils/customPlatformLabels'
import type { CustomPlatform } from '@/api/admin/customPlatforms'
import type { Column } from '@/components/common/types'
import AppLayout from '@/components/layout/AppLayout.vue'
import TablePageLayout from '@/components/layout/TablePageLayout.vue'
import DataTable from '@/components/common/DataTable.vue'
import SearchInput from '@/components/common/SearchInput.vue'
import Select from '@/components/common/Select.vue'
import EmptyState from '@/components/common/EmptyState.vue'
import BaseDialog from '@/components/common/BaseDialog.vue'
import ConfirmDialog from '@/components/common/ConfirmDialog.vue'
import type { LLMMonitorGroupPlatformOverride } from '@/api/admin/modelMonitor'

const appStore = useAppStore()

const loading = ref(false)
const groups = ref<LLMMonitorGroupPlatformOverride[]>([])
const customPlatforms = ref<CustomPlatform[]>([])
const searchQuery = ref('')
const platformFilter = ref<'all' | string>('all')
const savingGroupId = ref<number | null>(null)
const showEditDialog = ref(false)
const editingGroup = ref<LLMMonitorGroupPlatformOverride | null>(null)
const platformDraft = ref<string>('')
const showClearConfirm = ref(false)
const clearTargetGroup = ref<LLMMonitorGroupPlatformOverride | null>(null)

const corePlatformOptions = [
  { value: 'anthropic', label: 'Anthropic' },
  { value: 'openai', label: 'OpenAI' },
  { value: 'gemini', label: 'Gemini' },
  { value: 'antigravity', label: 'Antigravity' },
  { value: 'grok', label: 'Grok' },
  { value: 'composite', label: 'Composite' },
]

const platformOptions = computed(() => [
  ...corePlatformOptions,
  ...customPlatforms.value
    .filter((platform) => platform.enabled)
    .map((platform) => ({
      value: platform.code,
      label: platform.name,
    })),
])

const platformFilterOptions = computed(() => [
  { value: 'all', label: '全部平台' },
  ...platformOptions.value,
])

const columns = computed<Column[]>(() => [
  { key: 'name', label: '分组名称', sortable: false },
  { key: 'platform', label: '原平台', sortable: false },
  { key: 'actual_platform', label: '实际平台', sortable: false },
  { key: 'effective_platform', label: '展示平台', sortable: false },
  { key: 'override_state', label: '配置状态', sortable: false },
  { key: 'rate_multiplier', label: '倍率', sortable: false },
  { key: 'actions', label: '操作', sortable: false, class: 'text-right' },
])

const totalCount = computed(() => groups.value.length)
const overrideCount = computed(() => groups.value.filter((group) => !!group.actual_platform).length)
const inheritedCount = computed(() => totalCount.value - overrideCount.value)

const filteredGroups = computed(() => {
  const keyword = searchQuery.value.trim().toLowerCase()
  const platform = platformFilter.value

  return [...groups.value]
    .filter((group) => {
      if (platform !== 'all' && group.effective_platform !== platform) return false
      if (!keyword) return true
      const haystack = [group.name, group.platform, group.actual_platform, group.effective_platform]
        .filter((item) => item)
        .join(' ')
        .toLowerCase()
      return haystack.includes(keyword)
    })
    .sort((left, right) => left.name.localeCompare(right.name, 'zh-Hans-CN'))
})

const clearConfirmMessage = computed(() => {
  if (!clearTargetGroup.value) return ''
  return `确认清除「${clearTargetGroup.value.name}」的实际平台配置吗？清除后，模型监控页面会回退到原分组平台。`
})

async function reload() {
  loading.value = true
  try {
    const [items, platforms] = await Promise.all([
      adminAPI.modelMonitor.listLLMMonitorGroupPlatformOverrides(),
      adminAPI.customPlatforms.list(false),
    ])
    groups.value = items
    customPlatforms.value = platforms
    setCustomPlatformLabels(platforms)
  } catch (error) {
    appStore.showError(extractApiErrorMessage(error, '加载分组平台配置失败'))
  } finally {
    loading.value = false
  }
}

function applyFilters() {
  // 纯前端筛选，保留该方法以便 Select 的 change 事件统一收敛。
}

function platformText(value: string) {
  return resolvePlatformDisplayLabel(value)
}

function formatRateMultiplier(value: unknown) {
  const num = typeof value === 'number' ? value : Number(value)
  if (!Number.isFinite(num) || num <= 0) return '1x'
  const normalized = num.toFixed(3).replace(/\.0+$/, '').replace(/(\.[0-9]*?)0+$/, '$1')
  return `${normalized}x`
}

function openEditDialog(group: LLMMonitorGroupPlatformOverride) {
  editingGroup.value = group
  platformDraft.value = group.actual_platform
  showEditDialog.value = true
}

function closeEditDialog() {
  if (savingGroupId.value !== null) return
  showEditDialog.value = false
  editingGroup.value = null
  platformDraft.value = ''
}

async function saveOverride() {
  const group = editingGroup.value
  if (!group || savingGroupId.value !== null) return

  const platform = platformDraft.value
  savingGroupId.value = group.id
  try {
    if (platform) {
      await adminAPI.modelMonitor.setLLMMonitorGroupPlatformOverride(group.id, platform)
      appStore.showSuccess(`已将「${group.name}」的实际平台设置为 ${platformText(platform)}`)
    } else {
      await adminAPI.modelMonitor.clearLLMMonitorGroupPlatformOverride(group.id)
      appStore.showSuccess(`已清除「${group.name}」的实际平台配置`)
    }
    showEditDialog.value = false
    editingGroup.value = null
    platformDraft.value = ''
    await reload()
  } catch (error) {
    appStore.showError(extractApiErrorMessage(error, '保存分组平台配置失败'))
  } finally {
    savingGroupId.value = null
  }
}

function askClearOverride(group: LLMMonitorGroupPlatformOverride) {
  if (!group.actual_platform || savingGroupId.value !== null) return
  clearTargetGroup.value = group
  showClearConfirm.value = true
}

function closeClearConfirm() {
  if (savingGroupId.value !== null) return
  showClearConfirm.value = false
  clearTargetGroup.value = null
}

async function confirmClearOverride() {
  const group = clearTargetGroup.value
  if (!group || savingGroupId.value !== null) return

  savingGroupId.value = group.id
  try {
    await adminAPI.modelMonitor.clearLLMMonitorGroupPlatformOverride(group.id)
    appStore.showSuccess(`已清除「${group.name}」的实际平台配置`)
    if (editingGroup.value?.id === group.id) {
      showEditDialog.value = false
      editingGroup.value = null
      platformDraft.value = ''
    }
    showClearConfirm.value = false
    clearTargetGroup.value = null
    await reload()
  } catch (error) {
    appStore.showError(extractApiErrorMessage(error, '清除分组平台配置失败'))
  } finally {
    savingGroupId.value = null
  }
}

onMounted(() => {
  void reload()
})
</script>

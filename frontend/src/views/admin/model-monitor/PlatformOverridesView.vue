<template>
  <AppLayout>
    <TablePageLayout>
      <template #actions>
        <div class="po-actions-card">
          <div class="po-kpi-grid">
            <div class="po-kpi po-kpi-default">
              <div class="po-kpi-icon po-kpi-icon-default">
                <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><rect x="3" y="3" width="7" height="7" rx="1"/><rect x="14" y="3" width="7" height="7" rx="1"/><rect x="3" y="14" width="7" height="7" rx="1"/><rect x="14" y="14" width="7" height="7" rx="1"/></svg>
              </div>
              <div class="po-kpi-body">
                <span class="po-kpi-label">分组总数</span>
                <strong class="po-kpi-value">{{ totalCount }}</strong>
              </div>
            </div>
            <div class="po-kpi po-kpi-warning">
              <div class="po-kpi-icon po-kpi-icon-warning">
                <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M12 2L2 22h20L12 2z"/><line x1="12" y1="9" x2="12" y2="13"/><circle cx="12" cy="17" r="0.5" fill="currentColor"/></svg>
              </div>
              <div class="po-kpi-body">
                <span class="po-kpi-label">已配置实际平台</span>
                <strong class="po-kpi-value">{{ overrideCount }}</strong>
              </div>
            </div>
            <div class="po-kpi po-kpi-success">
              <div class="po-kpi-icon po-kpi-icon-success">
                <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M20 6L9 17l-5-5"/></svg>
              </div>
              <div class="po-kpi-body">
                <span class="po-kpi-label">默认继承原平台</span>
                <strong class="po-kpi-value">{{ inheritedCount }}</strong>
              </div>
            </div>
          </div>

          <div class="po-actions-right">
            <button class="po-btn po-btn-refresh" :disabled="loading" @click="reload">
              <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" :class="loading ? 'po-spin' : ''"><polyline points="23 4 23 10 17 10"/><path d="M20.49 15a9 9 0 1 1-2.12-9.36L23 10"/></svg>
              <span>{{ loading ? '刷新中…' : '刷新' }}</span>
            </button>
          </div>
        </div>
      </template>

      <template #filters>
        <div class="po-filter-bar">
          <div class="po-filter-search">
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
          <div class="po-filter-hint">
            监控页展示优先使用实际平台；未配置时回退原分组平台。
          </div>
        </div>
      </template>

      <template #table>
        <div class="po-table-wrapper">
          <DataTable
            :columns="columns"
            :data="filteredGroups"
            :loading="loading"
            row-key="id"
          >
            <template #cell-name="{ row, value }">
              <div class="po-cell-name">
                <strong>{{ value }}</strong>
                <span>ID：{{ row.id }}</span>
              </div>
            </template>

            <template #cell-platform="{ value }">
              <span :class="['po-badge', platformBadgeClass(value)]">
                {{ platformText(value) }}
              </span>
            </template>

            <template #cell-actual_platform="{ row, value }">
              <span v-if="value" :class="['po-badge', platformBadgeClass(value)]">
                {{ platformText(value) }}
              </span>
              <span v-else class="po-cell-dim">未配置</span>
              <div v-if="row.actual_platform" class="po-cell-note">
                覆盖原平台：{{ platformText(row.platform) }}
              </div>
            </template>

            <template #cell-effective_platform="{ row, value }">
              <span :class="['po-badge', platformBadgeClass(value)]">
                {{ row.effective_platform_name || platformText(value) }}
              </span>
            </template>

            <template #cell-override_state="{ row }">
              <span
                :class="[
                  'po-status-pill',
                  row.actual_platform ? 'po-status-pill-warning' : 'po-status-pill-muted'
                ]"
              >
                {{ row.actual_platform ? '已配置' : '默认继承' }}
              </span>
              <span
                :class="['po-status-pill', row.show_in_monitor ? 'po-status-pill-success' : 'po-status-pill-muted']"
              >
                {{ row.show_in_monitor ? '监控页显示' : '监控页隐藏' }}
              </span>
            </template>

            <template #cell-rate_multiplier="{ value }">
              <span class="po-cell-mono">{{ formatRateMultiplier(value) }}</span>
            </template>

            <template #cell-actions="{ row }">
              <div class="po-cell-actions">
                <Toggle
                  :model-value="row.show_in_monitor"
                  :disabled="savingGroupId === row.id"
                  :aria-label="row.show_in_monitor ? `隐藏「${row.name}」的模型监控展示` : `显示「${row.name}」到模型监控页面`"
                  @update:model-value="(value) => toggleGroupVisibility(row, value)"
                />
                <button class="po-btn po-btn-sm po-btn-edit" :disabled="savingGroupId === row.id" @click="openEditDialog(row)">
                  编辑
                </button>
                <button
                  class="po-btn po-btn-sm po-btn-clear"
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
        </div>
      </template>
    </TablePageLayout>

    <BaseDialog
      :show="showEditDialog"
      title="设置分组实际平台"
      width="normal"
      @close="closeEditDialog"
    >
      <div v-if="editingGroup" class="po-dialog-body">
        <div class="po-dialog-summary">
          <div class="po-dialog-summary-head">
            <span class="po-dialog-summary-name">{{ editingGroup.name }}</span>
            <span :class="['po-badge', platformBadgeClass(editingGroup.effective_platform)]">
              当前展示：{{ platformText(editingGroup.effective_platform) }}
            </span>
          </div>
          <div class="po-dialog-summary-meta">
            <span>原平台：{{ platformText(editingGroup.platform) }}</span>
            <span>当前实际平台：{{ editingGroup.actual_platform ? platformText(editingGroup.actual_platform) : '未配置' }}</span>
          </div>
        </div>

        <div class="po-dialog-field">
          <label class="po-dialog-label">实际平台</label>
          <Select
            v-model="platformDraft"
            :options="platformOptions"
            placeholder="选择实际平台，不选则沿用原平台"
            class="w-full"
            :clearable="true"
          />
        </div>

        <p class="po-dialog-hint">
          该配置只影响模型监控页面的展示和筛选，不会修改原有分组平台字段。
        </p>
      </div>

      <template #footer>
        <div class="po-dialog-footer">
          <button
            v-if="editingGroup?.actual_platform"
            class="po-btn po-btn-clear-foot"
            :disabled="savingGroupId === editingGroup.id"
            @click="askClearOverride(editingGroup)"
          >
            清除配置
          </button>
          <div class="po-dialog-footer-right">
            <button class="po-btn po-btn-cancel" :disabled="savingGroupId !== null" @click="closeEditDialog">取消</button>
            <button class="po-btn po-btn-primary" :disabled="savingGroupId !== null" @click="saveOverride">
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
import Toggle from '@/components/common/Toggle.vue'
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
    groups.value = items.map((item) => ({
      ...item,
      show_in_monitor: item.show_in_monitor !== false,
    }))
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

async function toggleGroupVisibility(group: LLMMonitorGroupPlatformOverride, showInMonitor: boolean) {
  if (savingGroupId.value !== null || group.show_in_monitor === showInMonitor) return

  const previous = group.show_in_monitor
  group.show_in_monitor = showInMonitor
  savingGroupId.value = group.id
  try {
    await adminAPI.modelMonitor.setLLMMonitorGroupVisibility(group.id, showInMonitor)
    appStore.showSuccess(showInMonitor
      ? `「${group.name}」将在模型监控页面显示`
      : `「${group.name}」已从模型监控页面隐藏`)
    await reload()
  } catch (error) {
    group.show_in_monitor = previous
    appStore.showError(extractApiErrorMessage(error, '保存模型监控显示配置失败'))
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

<style scoped>
/* ===== KPI 卡片区域 ===== */
.po-actions-card {
  display: flex;
  flex-direction: column;
  gap: 1rem;
  padding: 1.125rem 1.25rem;
  border: 1px solid #e5e7eb;
  border-radius: 0.875rem;
  background: linear-gradient(135deg, #f8fafc 0%, #fff 50%);
  box-shadow: 0 1px 2px rgba(15, 23, 42, 0.04);
}

:global(.dark) .po-actions-card {
  border-color: #374151;
  background: linear-gradient(135deg, #1f2937 0%, #111827 50%);
}

.po-kpi-grid {
  display: grid;
  grid-template-columns: repeat(3, minmax(0, 1fr));
  gap: 0.75rem;
  flex: 1;
}

.po-kpi {
  display: flex;
  align-items: flex-start;
  gap: 0.75rem;
  padding: 0.875rem 1rem;
  border-radius: 0.75rem;
  border: 1px solid transparent;
  transition: border-color 0.15s ease, box-shadow 0.15s ease;
}

.po-kpi-default {
  background: #f9fafb;
  border-color: #e5e7eb;
}

:global(.dark) .po-kpi-default {
  background: #1f2937;
  border-color: #374151;
}

.po-kpi-warning {
  background: #fffbeb;
  border-color: #fde68a;
}

:global(.dark) .po-kpi-warning {
  background: rgba(217, 119, 6, 0.08);
  border-color: rgba(217, 119, 6, 0.25);
}

.po-kpi-success {
  background: #ecfdf5;
  border-color: #a7f3d0;
}

:global(.dark) .po-kpi-success {
  background: rgba(22, 163, 74, 0.08);
  border-color: rgba(22, 163, 74, 0.25);
}

.po-kpi-icon {
  display: grid;
  width: 2.25rem;
  height: 2.25rem;
  flex-shrink: 0;
  place-items: center;
  border-radius: 0.625rem;
}

.po-kpi-icon-default {
  background: #e5e7eb;
  color: #6b7280;
}

:global(.dark) .po-kpi-icon-default {
  background: #374151;
  color: #9ca3af;
}

.po-kpi-icon-warning {
  background: #fef3c7;
  color: #d97706;
}

:global(.dark) .po-kpi-icon-warning {
  background: rgba(217, 119, 6, 0.2);
  color: #fbbf24;
}

.po-kpi-icon-success {
  background: #d1fae5;
  color: #16a34a;
}

:global(.dark) .po-kpi-icon-success {
  background: rgba(22, 163, 74, 0.2);
  color: #4ade80;
}

.po-kpi-body {
  display: flex;
  flex-direction: column;
  gap: 0.25rem;
  min-width: 0;
}

.po-kpi-label {
  font-size: 0.6875rem;
  font-weight: 600;
  letter-spacing: 0.04em;
  text-transform: uppercase;
  color: #6b7280;
}

:global(.dark) .po-kpi-label {
  color: #9ca3af;
}

.po-kpi-warning .po-kpi-label {
  color: #92400e;
}

:global(.dark) .po-kpi-warning .po-kpi-label {
  color: #fcd34d;
}

.po-kpi-success .po-kpi-label {
  color: #166534;
}

:global(.dark) .po-kpi-success .po-kpi-label {
  color: #6ee7b7;
}

.po-kpi-value {
  font-size: 1.5rem;
  font-weight: 700;
  font-variant-numeric: tabular-nums;
  line-height: 1;
  color: #111827;
}

:global(.dark) .po-kpi-value {
  color: #f9fafb;
}

.po-actions-right {
  display: flex;
  flex-shrink: 0;
  align-items: center;
  gap: 0.5rem;
}

@media (min-width: 1024px) {
  .po-actions-card {
    flex-direction: row;
    align-items: stretch;
  }
}

/* ===== 按钮 ===== */
.po-btn {
  display: inline-flex;
  align-items: center;
  gap: 0.375rem;
  min-height: 2.25rem;
  padding: 0.45rem 0.8rem;
  border: 1px solid #e5e7eb;
  border-radius: 0.5rem;
  background: #fff;
  color: #374151;
  font-size: 0.8125rem;
  font-weight: 500;
  cursor: pointer;
  transition: background 0.15s ease, border-color 0.15s ease, color 0.15s ease;
}

:global(.dark) .po-btn {
  background: #1f2937;
  border-color: #374151;
  color: #e5e7eb;
}

.po-btn:hover {
  background: #f9fafb;
}

:global(.dark) .po-btn:hover {
  background: #374151;
}

.po-btn:disabled {
  cursor: not-allowed;
  opacity: 0.5;
}

.po-btn-refresh {
  color: #16a34a;
  border-color: #d1fae5;
  background: #f0fdf4;
}

:global(.dark) .po-btn-refresh {
  color: #4ade80;
  border-color: rgba(22, 163, 74, 0.3);
  background: rgba(22, 163, 74, 0.1);
}

.po-btn-refresh:hover {
  background: #dcfce7;
}

:global(.dark) .po-btn-refresh:hover {
  background: rgba(22, 163, 74, 0.18);
}

.po-btn-sm {
  min-height: 1.75rem;
  padding: 0.25rem 0.6rem;
  font-size: 0.75rem;
}

.po-btn-edit {
  color: #3b82f6;
  border-color: #bfdbfe;
  background: #eff6ff;
}

:global(.dark) .po-btn-edit {
  color: #93c5fd;
  border-color: rgba(59, 130, 246, 0.3);
  background: rgba(59, 130, 246, 0.1);
}

.po-btn-edit:hover {
  background: #dbeafe;
}

.po-btn-clear {
  color: #dc2626;
  border-color: #fecaca;
  background: #fef2f2;
}

:global(.dark) .po-btn-clear {
  color: #fca5a5;
  border-color: rgba(220, 38, 38, 0.3);
  background: rgba(220, 38, 38, 0.1);
}

.po-btn-clear:hover {
  background: #fee2e2;
}

.po-btn-primary {
  color: #fff;
  border-color: #3b82f6;
  background: #3b82f6;
}

.po-btn-primary:hover {
  background: #2563eb;
  border-color: #2563eb;
}

.po-btn-cancel {
  color: #6b7280;
  border-color: #e5e7eb;
  background: #fff;
}

:global(.dark) .po-btn-cancel {
  color: #9ca3af;
  border-color: #374151;
  background: #1f2937;
}

.po-btn-clear-foot {
  color: #dc2626;
  border-color: transparent;
  background: transparent;
}

.po-btn-clear-foot:hover {
  background: #fef2f2;
}

:global(.dark) .po-btn-clear-foot:hover {
  background: rgba(220, 38, 38, 0.1);
}

/* ===== 筛选栏 ===== */
.po-filter-bar {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 0.75rem;
  padding: 0.875rem 1rem;
  border: 1px solid #e5e7eb;
  border-radius: 0.75rem;
  background: #fff;
  box-shadow: 0 1px 2px rgba(15, 23, 42, 0.04);
}

:global(.dark) .po-filter-bar {
  border-color: #374151;
  background: #1f2937;
}

.po-filter-search {
  flex: 1;
  min-width: 0;
  max-width: 24rem;
}

.po-filter-hint {
  color: #9ca3af;
  font-size: 0.8125rem;
  margin-left: auto;
}

:global(.dark) .po-filter-hint {
  color: #6b7280;
}

/* ===== 表格 ===== */
.po-table-wrapper {
  overflow: hidden;
  border: 1px solid #e5e7eb;
  border-radius: 0.875rem;
  background: #fff;
  box-shadow: 0 1px 2px rgba(15, 23, 42, 0.04);
}

:global(.dark) .po-table-wrapper {
  border-color: #374151;
  background: #1f2937;
}

.po-cell-name {
  display: flex;
  flex-direction: column;
  gap: 0.2rem;
}

.po-cell-name strong {
  color: #111827;
  font-size: 0.875rem;
  font-weight: 600;
}

:global(.dark) .po-cell-name strong {
  color: #f9fafb;
}

.po-cell-name span {
  color: #9ca3af;
  font-size: 0.7rem;
}

:global(.dark) .po-cell-name span {
  color: #6b7280;
}

.po-badge {
  display: inline-flex;
  align-items: center;
  padding: 0.2rem 0.55rem;
  border-radius: 0.35rem;
  font-size: 0.75rem;
  font-weight: 600;
  border: 1px solid #e5e7eb;
  background: #f3f4f6;
  color: #374151;
}

:global(.dark) .po-badge {
  border-color: #374151;
  background: #374151;
  color: #e5e7eb;
}

.po-status-pill {
  display: inline-flex;
  align-items: center;
  padding: 0.2rem 0.6rem;
  border-radius: 9999px;
  font-size: 0.75rem;
  font-weight: 600;
}

.po-status-pill-warning {
  background: #fef3c7;
  color: #92400e;
}

:global(.dark) .po-status-pill-warning {
  background: rgba(217, 119, 6, 0.15);
  color: #fcd34d;
}

.po-status-pill-success {
  margin-left: 0.35rem;
  background: #dcfce7;
  color: #166534;
}

:global(.dark) .po-status-pill-success {
  background: rgba(22, 163, 74, 0.16);
  color: #86efac;
}

.po-status-pill-muted {
  background: #f3f4f6;
  color: #6b7280;
}

:global(.dark) .po-status-pill-muted {
  background: #374151;
  color: #9ca3af;
}

.po-cell-dim {
  color: #9ca3af;
  font-size: 0.8125rem;
}

.po-cell-note {
  margin-top: 0.25rem;
  color: #9ca3af;
  font-size: 0.72rem;
}

:global(.dark) .po-cell-note {
  color: #6b7280;
}

.po-cell-mono {
  font-family: ui-monospace, SFMono-Regular, Consolas, "Liberation Mono", monospace;
  font-size: 0.8125rem;
  color: #374151;
}

:global(.dark) .po-cell-mono {
  color: #e5e7eb;
}

.po-cell-actions {
  display: flex;
  align-items: center;
  justify-content: flex-end;
  gap: 0.4rem;
}

/* ===== 弹窗 ===== */
.po-dialog-body {
  display: flex;
  flex-direction: column;
  gap: 1rem;
}

.po-dialog-summary {
  padding: 0.875rem 1rem;
  border: 1px solid #e5e7eb;
  border-radius: 0.75rem;
  background: #f9fafb;
}

:global(.dark) .po-dialog-summary {
  border-color: #374151;
  background: #1f2937;
}

.po-dialog-summary-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 0.75rem;
}

.po-dialog-summary-name {
  font-weight: 600;
  color: #111827;
  font-size: 0.9375rem;
}

:global(.dark) .po-dialog-summary-name {
  color: #f9fafb;
}

.po-dialog-summary-meta {
  display: flex;
  flex-direction: column;
  gap: 0.25rem;
  margin-top: 0.5rem;
  color: #9ca3af;
  font-size: 0.75rem;
}

:global(.dark) .po-dialog-summary-meta {
  color: #6b7280;
}

.po-dialog-field {
  display: flex;
  flex-direction: column;
  gap: 0.4rem;
}

.po-dialog-label {
  font-size: 0.875rem;
  font-weight: 600;
  color: #111827;
}

:global(.dark) .po-dialog-label {
  color: #f9fafb;
}

.po-dialog-hint {
  margin: 0;
  color: #9ca3af;
  font-size: 0.75rem;
  line-height: 1.5;
}

.po-dialog-footer {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 0.75rem;
}

.po-dialog-footer-right {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  margin-left: auto;
}

/* 旋转动画 */
.po-spin {
  animation: po-spin 0.7s linear infinite;
}

@keyframes po-spin {
  to { transform: rotate(360deg); }
}

/* ===== 响应式 ===== */
@media (max-width: 640px) {
  .po-kpi-grid {
    grid-template-columns: 1fr;
  }

  .po-filter-bar {
    flex-direction: column;
    align-items: stretch;
  }

  .po-filter-search {
    max-width: none;
  }

  .po-filter-hint {
    margin-left: 0;
  }
}
</style>

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
                <span class="po-kpi-foot">全部模型监控分组</span>
              </div>
            </div>
            <div class="po-kpi po-kpi-warning">
              <div class="po-kpi-icon po-kpi-icon-warning">
                <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M12 2L2 22h20L12 2z"/><line x1="12" y1="9" x2="12" y2="13"/><circle cx="12" cy="17" r="0.5" fill="currentColor"/></svg>
              </div>
              <div class="po-kpi-body">
                <span class="po-kpi-label">已配置实际平台</span>
                <strong class="po-kpi-value">{{ overrideCount }}</strong>
                <span class="po-kpi-foot">已设置独立展示平台</span>
              </div>
            </div>
            <div class="po-kpi po-kpi-success">
              <div class="po-kpi-icon po-kpi-icon-success">
                <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M20 6L9 17l-5-5"/></svg>
              </div>
              <div class="po-kpi-body">
                <span class="po-kpi-label">默认继承原平台</span>
                <strong class="po-kpi-value">{{ inheritedCount }}</strong>
                <span class="po-kpi-foot">沿用原平台展示</span>
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
        <div class="po-filter-panel">
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
          <div class="po-legend-bar" aria-label="平台颜色图例">
            <span class="po-legend-title">平台配色</span>
            <span v-for="item in legendItems" :key="item.value" class="po-legend-item">
              <i class="po-legend-dot" :style="{ background: resolvePlatformColor(item.value) }"></i>
              {{ item.label }}
            </span>
          </div>
        </div>
      </template>

      <template #table>
        <div class="po-table-wrapper">
          <DataTable
            :columns="columns"
            :data="pagedGroups"
            :loading="loading"
            row-key="id"
          >
            <template #cell-name="{ row, value }">
              <div class="po-cell-name">
                <div class="po-cell-name-main">
                  <i class="po-cell-name-dot" :style="{ background: resolvePlatformColor(row.effective_platform) }"></i>
                  <strong :style="{ color: resolvePlatformColor(row.effective_platform) }">{{ value }}</strong>
                </div>
                <span>ID：{{ row.id }}</span>
              </div>
            </template>

            <template #cell-platform="{ value }">
              <span :class="['po-badge', platformBadgeClass(value)]" :style="customPlatformBadgeStyle(value)">
                <i class="po-badge-dot"></i>
                {{ platformText(value) }}
              </span>
            </template>

            <template #cell-actual_platform="{ row, value }">
              <span v-if="value" :class="['po-badge', platformBadgeClass(value)]" :style="customPlatformBadgeStyle(value)">
                <i class="po-badge-dot"></i>
                {{ platformText(value) }}
              </span>
              <span v-else class="po-cell-dim">未配置</span>
              <div v-if="row.actual_platform" class="po-cell-note">
                覆盖原平台：{{ platformText(row.platform) }}
              </div>
            </template>

            <template #cell-effective_platform="{ row, value }">
              <span :class="['po-badge', platformBadgeClass(value)]" :style="customPlatformBadgeStyle(value)">
                <i class="po-badge-dot"></i>
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
            </template>

            <template #cell-show_in_monitor="{ row }">
              <Toggle
                :model-value="row.show_in_monitor"
                :disabled="savingGroupId === row.id"
                :aria-label="row.show_in_monitor ? `隐藏「${row.name}」的模型监控展示` : `显示「${row.name}」到模型监控页面`"
                @update:model-value="(value) => toggleGroupVisibility(row, value)"
              />
            </template>

            <template #cell-rate_multiplier="{ value }">
              <span class="po-rate-chip">{{ formatRateMultiplier(value) }}</span>
            </template>

            <template #cell-actions="{ row }">
              <div class="po-cell-actions">
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

      <template #pagination>
        <Pagination
          v-if="filteredGroups.length > 0"
          :page="page"
          :total="filteredGroups.length"
          :page-size="pageSize"
          @update:page="page = $event"
          @update:pageSize="changePageSize"
        />
      </template>
    </TablePageLayout>

    <BaseDialog
      :show="showEditDialog"
      title="设置分组实际平台"
      width="normal"
      @close="closeEditDialog"
    >
      <div v-if="editingGroup" class="po-dialog-body">
        <div class="po-dialog-summary" :style="{ borderLeftColor: resolvePlatformColor(editingGroup.effective_platform) }">
          <div class="po-dialog-summary-head">
            <span class="po-dialog-summary-name">{{ editingGroup.name }}</span>
            <span :class="['po-badge', platformBadgeClass(editingGroup.effective_platform)]" :style="customPlatformBadgeStyle(editingGroup.effective_platform)">
              <i class="po-badge-dot"></i>
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
          <div v-if="platformDraft" class="po-draft-preview">
            <span :class="['po-badge', platformBadgeClass(platformDraft)]" :style="customPlatformBadgeStyle(platformDraft)">
              <i class="po-badge-dot"></i>
              {{ platformText(platformDraft) }}
            </span>
            <span class="po-draft-preview-hint">保存后，模型监控页面将以该平台展示并筛选此分组。</span>
          </div>
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
import { computed, onMounted, ref, watch } from 'vue'
import { useAppStore } from '@/stores/app'
import { extractApiErrorMessage } from '@/utils/apiError'
import { platformBadgeClass } from '@/utils/platformColors'
import { adminAPI } from '@/api/admin'
import { getPersistedPageSize } from '@/composables/usePersistedPageSize'
import { resolvePlatformDisplayLabel, setCustomPlatformLabels } from '@/utils/customPlatformLabels'
import { customPlatformBadgeStyle, resolvePlatformColor, updateCustomPlatformColors } from '@/utils/customPlatformColors'
import { buildPlatformOptions } from '@/utils/platformOptions'
import type { CustomPlatform } from '@/api/admin/customPlatforms'
import type { Column } from '@/components/common/types'
import AppLayout from '@/components/layout/AppLayout.vue'
import TablePageLayout from '@/components/layout/TablePageLayout.vue'
import DataTable from '@/components/common/DataTable.vue'
import SearchInput from '@/components/common/SearchInput.vue'
import Select from '@/components/common/Select.vue'
import Toggle from '@/components/common/Toggle.vue'
import EmptyState from '@/components/common/EmptyState.vue'
import Pagination from '@/components/common/Pagination.vue'
import BaseDialog from '@/components/common/BaseDialog.vue'
import ConfirmDialog from '@/components/common/ConfirmDialog.vue'
import type { LLMMonitorGroupPlatformOverride } from '@/api/admin/modelMonitor'

const appStore = useAppStore()

const loading = ref(false)
const groups = ref<LLMMonitorGroupPlatformOverride[]>([])
const customPlatforms = ref<CustomPlatform[]>([])
const searchQuery = ref('')
const platformFilter = ref<'all' | string>('all')
const page = ref(1)
const pageSize = ref(getPersistedPageSize())
const savingGroupId = ref<number | null>(null)
const showEditDialog = ref(false)
const editingGroup = ref<LLMMonitorGroupPlatformOverride | null>(null)
const platformDraft = ref<string>('')
const showClearConfirm = ref(false)
const clearTargetGroup = ref<LLMMonitorGroupPlatformOverride | null>(null)

const platformOptions = computed(() => buildPlatformOptions(customPlatforms.value))

const platformFilterOptions = computed(() => [
  { value: 'all', label: '全部平台' },
  ...platformOptions.value,
])

const legendItems = computed(() => [
  ...platformOptions.value,
])

const columns = computed<Column[]>(() => [
  { key: 'name', label: '分组名称', sortable: false },
  { key: 'platform', label: '原平台', sortable: false },
  { key: 'actual_platform', label: '实际平台', sortable: false },
  { key: 'effective_platform', label: '展示平台', sortable: false },
  { key: 'override_state', label: '配置状态', sortable: false },
  { key: 'show_in_monitor', label: '监控显示', sortable: false },
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

const pagedGroups = computed(() => {
  const start = (page.value - 1) * pageSize.value
  return filteredGroups.value.slice(start, start + pageSize.value)
})

watch([searchQuery, platformFilter], () => {
  page.value = 1
})

watch([() => filteredGroups.value.length, pageSize], ([total, size]) => {
  const lastPage = Math.max(1, Math.ceil(total / size))
  if (page.value > lastPage) page.value = lastPage
})

function changePageSize(size: number) {
  pageSize.value = size
  page.value = 1
}

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
    updateCustomPlatformColors(platforms)
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
  position: relative;
  overflow: hidden;
  display: flex;
  align-items: flex-start;
  gap: 0.75rem;
  padding: 1rem;
  border-radius: 0.75rem;
  border: 1px solid transparent;
  transition: border-color 0.15s ease, box-shadow 0.15s ease, transform 0.15s ease;
}

/* 顶部平台色渐变条 */
.po-kpi::before {
  content: '';
  position: absolute;
  top: 0;
  left: 0;
  right: 0;
  height: 3px;
}

.po-kpi:hover {
  transform: translateY(-1px);
  box-shadow: 0 6px 16px -6px rgba(15, 23, 42, 0.14);
}

.po-kpi-default {
  background: linear-gradient(180deg, #eef2ff 0%, #f9fafb 55%);
  border-color: #e0e7ff;
}

.po-kpi-default::before {
  background: linear-gradient(90deg, #6366f1, #a855f7);
}

:global(.dark) .po-kpi-default {
  background: linear-gradient(180deg, rgba(99, 102, 241, 0.12) 0%, #1f2937 55%);
  border-color: rgba(99, 102, 241, 0.3);
}

.po-kpi-warning {
  background: linear-gradient(180deg, #fffbeb 0%, #fefce8 60%);
  border-color: #fde68a;
}

.po-kpi-warning::before {
  background: linear-gradient(90deg, #f59e0b, #f97316);
}

:global(.dark) .po-kpi-warning {
  background: linear-gradient(180deg, rgba(217, 119, 6, 0.14) 0%, #1f2937 55%);
  border-color: rgba(217, 119, 6, 0.3);
}

.po-kpi-success {
  background: linear-gradient(180deg, #ecfdf5 0%, #f0fdf4 60%);
  border-color: #a7f3d0;
}

.po-kpi-success::before {
  background: linear-gradient(90deg, #10b981, #14b8a6);
}

:global(.dark) .po-kpi-success {
  background: linear-gradient(180deg, rgba(22, 163, 74, 0.12) 0%, #1f2937 55%);
  border-color: rgba(22, 163, 74, 0.3);
}

.po-kpi-icon {
  display: grid;
  width: 2.25rem;
  height: 2.25rem;
  flex-shrink: 0;
  place-items: center;
  border-radius: 0.625rem;
  box-shadow: 0 1px 2px rgba(15, 23, 42, 0.08);
}

.po-kpi-icon-default {
  background: linear-gradient(135deg, #6366f1, #8b5cf6);
  color: #fff;
}

:global(.dark) .po-kpi-icon-default {
  background: linear-gradient(135deg, #818cf8, #a78bfa);
}

.po-kpi-icon-warning {
  background: linear-gradient(135deg, #f59e0b, #f97316);
  color: #fff;
}

:global(.dark) .po-kpi-icon-warning {
  background: linear-gradient(135deg, #fbbf24, #fb923c);
}

.po-kpi-icon-success {
  background: linear-gradient(135deg, #10b981, #14b8a6);
  color: #fff;
}

:global(.dark) .po-kpi-icon-success {
  background: linear-gradient(135deg, #34d399, #2dd4bf);
}

.po-kpi-body {
  display: flex;
  flex-direction: column;
  gap: 0.3rem;
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

.po-kpi-default .po-kpi-label {
  color: #4f46e5;
}

:global(.dark) .po-kpi-default .po-kpi-label {
  color: #a5b4fc;
}

.po-kpi-warning .po-kpi-label {
  color: #b45309;
}

:global(.dark) .po-kpi-warning .po-kpi-label {
  color: #fcd34d;
}

.po-kpi-success .po-kpi-label {
  color: #047857;
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

.po-kpi-foot {
  margin-top: 0.1rem;
  font-size: 0.6875rem;
  color: #9ca3af;
}

:global(.dark) .po-kpi-foot {
  color: #6b7280;
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

/* ===== 筛选面板 ===== */
.po-filter-panel {
  display: flex;
  flex-direction: column;
  gap: 0.625rem;
}

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

/* 平台配色图例 */
.po-legend-bar {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 0.5rem 1rem;
  padding: 0.5rem 0.875rem;
  border: 1px dashed #d1d5db;
  border-radius: 0.625rem;
  background: rgba(249, 250, 251, 0.7);
}

:global(.dark) .po-legend-bar {
  border-color: #4b5563;
  background: rgba(31, 41, 55, 0.5);
}

.po-legend-title {
  display: inline-flex;
  align-items: center;
  gap: 0.375rem;
  font-size: 0.75rem;
  font-weight: 600;
  letter-spacing: 0.04em;
  color: #6b7280;
}

:global(.dark) .po-legend-title {
  color: #9ca3af;
}

.po-legend-title::before {
  content: '';
  width: 0.625rem;
  height: 0.625rem;
  border-radius: 9999px;
  background: conic-gradient(#f97316, #22c55e, #a855f7, #3b82f6, #71717a, #06b6d4, #f97316);
}

.po-legend-item {
  display: inline-flex;
  align-items: center;
  gap: 0.35rem;
  font-size: 0.75rem;
  font-weight: 500;
  color: #4b5563;
}

:global(.dark) .po-legend-item {
  color: #d1d5db;
}

.po-legend-dot {
  width: 0.5rem;
  height: 0.5rem;
  flex-shrink: 0;
  border-radius: 9999px;
  box-shadow: 0 0 0 3px rgba(15, 23, 42, 0.06);
}

/* ===== 表格 ===== */
.po-table-wrapper {
  display: flex;
  flex: 1 1 auto;
  min-height: 0;
  flex-direction: column;
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

.po-cell-name-main {
  display: flex;
  align-items: center;
  gap: 0.45rem;
  min-width: 0;
}

.po-cell-name-dot {
  width: 0.5rem;
  height: 0.5rem;
  flex-shrink: 0;
  border-radius: 9999px;
  box-shadow: 0 0 0 3px rgba(15, 23, 42, 0.05);
}

.po-cell-name strong {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
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

/* 平台徽章：只声明布局与边框宽度/样式，背景与配色交给
   platformBadgeClass 的 Tailwind 工具类，避免 scoped 基础样式
   因优先级覆盖平台品牌色。 */
.po-badge {
  display: inline-flex;
  align-items: center;
  gap: 0.35rem;
  padding: 0.15rem 0.6rem;
  border-width: 1px;
  border-style: solid;
  border-radius: 9999px;
  font-size: 0.75rem;
  font-weight: 600;
  line-height: 1.375rem;
  white-space: nowrap;
}

.po-badge-dot {
  width: 0.375rem;
  height: 0.375rem;
  flex-shrink: 0;
  border-radius: 9999px;
  background: currentColor;
  opacity: 0.85;
}

.po-status-pill {
  display: inline-flex;
  align-items: center;
  gap: 0.375rem;
  padding: 0.2rem 0.6rem;
  border-radius: 9999px;
  font-size: 0.75rem;
  font-weight: 600;
}

.po-status-pill::before {
  content: '';
  width: 0.375rem;
  height: 0.375rem;
  flex-shrink: 0;
  border-radius: 9999px;
  background: currentColor;
  opacity: 0.85;
}

.po-status-pill-warning {
  background: #fef3c7;
  color: #b45309;
}

:global(.dark) .po-status-pill-warning {
  background: rgba(217, 119, 6, 0.15);
  color: #fcd34d;
}

.po-status-pill-success {
  margin-left: 0.35rem;
  background: #dcfce7;
  color: #047857;
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

/* 倍率胶囊 */
.po-rate-chip {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  min-width: 3.25rem;
  padding: 0.15rem 0.5rem;
  border: 1px solid #e5e7eb;
  border-radius: 0.5rem;
  background: #f9fafb;
  font-family: ui-monospace, SFMono-Regular, Consolas, "Liberation Mono", monospace;
  font-size: 0.75rem;
  font-weight: 600;
  color: #374151;
}

:global(.dark) .po-rate-chip {
  border-color: #374151;
  background: #1f2937;
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
  border-left-width: 3px;
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

/* 平台草稿预览 */
.po-draft-preview {
  display: flex;
  align-items: center;
  gap: 0.6rem;
  padding: 0.5rem 0.75rem;
  border: 1px solid #e5e7eb;
  border-radius: 0.625rem;
  background: #f9fafb;
}

:global(.dark) .po-draft-preview {
  border-color: #374151;
  background: #1f2937;
}

.po-draft-preview-hint {
  color: #9ca3af;
  font-size: 0.75rem;
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

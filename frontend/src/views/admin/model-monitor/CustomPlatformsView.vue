<template>
  <AppLayout>
    <TablePageLayout>
      <template #actions>
        <div class="cp-actions-card">
          <div class="cp-kpi-grid">
            <div class="cp-kpi cp-kpi-default">
              <div class="cp-kpi-icon cp-kpi-icon-default">
                <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><rect x="3" y="3" width="7" height="7" rx="1"/><rect x="14" y="3" width="7" height="7" rx="1"/><rect x="3" y="14" width="7" height="7" rx="1"/><rect x="14" y="14" width="7" height="7" rx="1"/></svg>
              </div>
              <div class="cp-kpi-body">
                <span class="cp-kpi-label">平台总数</span>
                <strong class="cp-kpi-value">{{ totalCount }}</strong>
              </div>
            </div>
            <div class="cp-kpi cp-kpi-success">
              <div class="cp-kpi-icon cp-kpi-icon-success">
                <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M20 6L9 17l-5-5"/></svg>
              </div>
              <div class="cp-kpi-body">
                <span class="cp-kpi-label">启用中</span>
                <strong class="cp-kpi-value">{{ enabledCount }}</strong>
              </div>
            </div>
            <div class="cp-kpi cp-kpi-danger">
              <div class="cp-kpi-icon cp-kpi-icon-danger">
                <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><line x1="18" y1="6" x2="6" y2="18"/><line x1="6" y1="6" x2="18" y2="18"/></svg>
              </div>
              <div class="cp-kpi-body">
                <span class="cp-kpi-label">停用中</span>
                <strong class="cp-kpi-value">{{ disabledCount }}</strong>
              </div>
            </div>
          </div>

          <div class="cp-actions-right">
            <button class="cp-btn cp-btn-refresh" :disabled="loading" @click="loadCustomPlatforms">
              <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" :class="loading ? 'cp-spin' : ''"><polyline points="23 4 23 10 17 10"/><path d="M20.49 15a9 9 0 1 1-2.12-9.36L23 10"/></svg>
              <span>{{ loading ? '刷新中…' : '刷新' }}</span>
            </button>
            <button class="cp-btn cp-btn-primary" @click="openCreateDialog">
              <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><line x1="12" y1="5" x2="12" y2="19"/><line x1="5" y1="12" x2="19" y2="12"/></svg>
              <span>新增平台</span>
            </button>
          </div>
        </div>
      </template>

      <template #filters>
        <div class="cp-filter-bar">
          <div class="cp-filter-search">
            <SearchInput
              v-model="searchQuery"
              placeholder="搜索平台代号或名称"
            />
          </div>
          <Select
            v-model="statusFilter"
            :options="statusOptions"
            placeholder="全部状态"
            class="w-44"
            :clearable="false"
          />
          <div class="cp-filter-hint">
            这里维护的是模型监控和供应商模块共用的独立平台字典，不影响原有分组平台字段。
          </div>
        </div>
      </template>

      <template #table>
        <div class="cp-table-wrapper">
          <DataTable
            :columns="columns"
            :data="filteredPlatforms"
            :loading="loading"
            row-key="id"
          >
            <template #cell-code="{ value }">
              <span class="cp-code-badge">{{ value }}</span>
            </template>

            <template #cell-name="{ value }">
              <span class="cp-cell-name">{{ value }}</span>
            </template>

            <template #cell-enabled="{ value }">
              <span
                :class="[
                  'cp-status-pill',
                  value ? 'cp-status-pill-success' : 'cp-status-pill-muted'
                ]"
              >
                <span class="cp-status-dot" :class="value ? 'cp-status-dot-success' : 'cp-status-dot-muted'" aria-hidden="true"></span>
                {{ value ? '启用' : '停用' }}
              </span>
            </template>

            <template #cell-sort_order="{ value }">
              <span class="cp-cell-mono">{{ value }}</span>
            </template>

            <template #cell-created_at="{ value }">
              <span class="cp-cell-time">{{ formatTime(value) }}</span>
            </template>

            <template #cell-updated_at="{ value }">
              <span class="cp-cell-time">{{ formatTime(value) }}</span>
            </template>

            <template #cell-actions="{ row }">
              <div class="cp-cell-actions">
                <button class="cp-btn cp-btn-sm cp-btn-edit" @click="openEditDialog(row)">编辑</button>
                <button class="cp-btn cp-btn-sm cp-btn-delete" @click="askDeletePlatform(row)">删除</button>
              </div>
            </template>
          </DataTable>
        </div>
      </template>
    </TablePageLayout>

    <BaseDialog
      :show="showDialog"
      :title="dialogTitle"
      width="wide"
      @close="closeDialog"
    >
      <div class="cp-dialog-body">
        <div class="cp-dialog-row">
          <div class="cp-dialog-field">
            <label class="cp-dialog-label">平台代号</label>
            <Input
              v-model="form.code"
              placeholder="例如 glm、deepseek、kimi"
              :disabled="saving"
              required
            />
          </div>
          <div class="cp-dialog-field">
            <label class="cp-dialog-label">显示名称</label>
            <Input
              v-model="form.name"
              placeholder="例如 GLM、DeepSeek、Kimi"
              :disabled="saving"
              required
            />
          </div>
        </div>

        <div class="cp-dialog-row">
          <div class="cp-dialog-field">
            <label class="cp-dialog-label">排序</label>
            <Input
              v-model="form.sortOrder"
              type="number"
              placeholder="数字越小越靠前"
              :disabled="saving"
            />
          </div>
          <div class="cp-dialog-field">
            <label class="cp-dialog-label">启用状态</label>
            <div class="cp-dialog-toggle">
              <Toggle v-model="form.enabled" :disabled="saving" />
              <div class="cp-dialog-toggle-text">
                <span class="cp-dialog-toggle-state">{{ form.enabled ? '启用' : '停用' }}</span>
                <span class="cp-dialog-toggle-hint">停用后不会出现在新配置的可选项中，但历史记录仍会保留。</span>
              </div>
            </div>
          </div>
        </div>
      </div>

      <template #footer>
        <div class="cp-dialog-footer">
          <button class="cp-btn cp-btn-cancel" :disabled="saving" @click="closeDialog">取消</button>
          <button class="cp-btn cp-btn-primary" :disabled="saving" @click="savePlatform">
            {{ saving ? '保存中…' : '保存' }}
          </button>
        </div>
      </template>
    </BaseDialog>

    <ConfirmDialog
      :show="showDeleteConfirm"
      title="删除自定义平台"
      :message="deleteConfirmMessage"
      confirm-text="删除"
      cancel-text="取消"
      :danger="true"
      @confirm="confirmDeletePlatform"
      @cancel="closeDeleteConfirm"
    />
  </AppLayout>
</template>

<script setup lang="ts">
import { computed, onMounted, reactive, ref } from 'vue'
import AppLayout from '@/components/layout/AppLayout.vue'
import TablePageLayout from '@/components/layout/TablePageLayout.vue'
import DataTable from '@/components/common/DataTable.vue'
import SearchInput from '@/components/common/SearchInput.vue'
import Select from '@/components/common/Select.vue'
import Input from '@/components/common/Input.vue'
import Toggle from '@/components/common/Toggle.vue'
import BaseDialog from '@/components/common/BaseDialog.vue'
import ConfirmDialog from '@/components/common/ConfirmDialog.vue'
import type { Column } from '@/components/common/types'
import { useAppStore } from '@/stores/app'
import { extractApiErrorMessage } from '@/utils/apiError'
import { customPlatformsAPI, type CustomPlatform, type CustomPlatformUpsertPayload } from '@/api/admin/customPlatforms'
import { setCustomPlatformLabels } from '@/utils/customPlatformLabels'

const appStore = useAppStore()

const loading = ref(false)
const saving = ref(false)
const platforms = ref<CustomPlatform[]>([])
const searchQuery = ref('')
const statusFilter = ref<'all' | 'enabled' | 'disabled'>('all')
const showDialog = ref(false)
const editingPlatform = ref<CustomPlatform | null>(null)
const showDeleteConfirm = ref(false)
const deleteTarget = ref<CustomPlatform | null>(null)

const form = reactive({
  code: '',
  name: '',
  enabled: true,
  sortOrder: '0',
})

const statusOptions = [
  { value: 'all', label: '全部状态' },
  { value: 'enabled', label: '仅启用' },
  { value: 'disabled', label: '仅停用' },
]

const columns = computed<Column[]>(() => [
  { key: 'code', label: '代号', sortable: false },
  { key: 'name', label: '名称', sortable: false },
  { key: 'enabled', label: '状态', sortable: false },
  { key: 'sort_order', label: '排序', sortable: false },
  { key: 'created_at', label: '创建时间', sortable: false },
  { key: 'updated_at', label: '更新时间', sortable: false },
  { key: 'actions', label: '操作', sortable: false, class: 'text-right' },
])

const totalCount = computed(() => platforms.value.length)
const enabledCount = computed(() => platforms.value.filter(item => item.enabled).length)
const disabledCount = computed(() => totalCount.value - enabledCount.value)

const filteredPlatforms = computed(() => {
  const keyword = searchQuery.value.trim().toLowerCase()
  const status = statusFilter.value
  return [...platforms.value]
    .filter((item) => {
      if (status === 'enabled' && !item.enabled) return false
      if (status === 'disabled' && item.enabled) return false
      if (!keyword) return true
      return [item.code, item.name].some(value => value.toLowerCase().includes(keyword))
    })
    .sort((left, right) => (left.sort_order - right.sort_order) || left.id - right.id)
})

const dialogTitle = computed(() => editingPlatform.value ? `编辑平台：${editingPlatform.value.name}` : '新增自定义平台')
const deleteConfirmMessage = computed(() => {
  if (!deleteTarget.value) return ''
  return `确认删除「${deleteTarget.value.name}」吗？删除后仅作为历史记录保留，不会影响已保存的数据。`
})

async function loadCustomPlatforms() {
  loading.value = true
  try {
    const items = await customPlatformsAPI.list(false)
    platforms.value = items
    setCustomPlatformLabels(items)
  } catch (error) {
    appStore.showError(extractApiErrorMessage(error, '加载自定义平台失败'))
  } finally {
    loading.value = false
  }
}

function openCreateDialog() {
  editingPlatform.value = null
  form.code = ''
  form.name = ''
  form.enabled = true
  form.sortOrder = '0'
  showDialog.value = true
}

function openEditDialog(item: CustomPlatform) {
  editingPlatform.value = item
  form.code = item.code
  form.name = item.name
  form.enabled = item.enabled
  form.sortOrder = String(item.sort_order ?? 0)
  showDialog.value = true
}

function closeDialog() {
  if (saving.value) return
  showDialog.value = false
  editingPlatform.value = null
}

async function savePlatform() {
  if (saving.value) return
  const code = form.code.trim().toLowerCase()
  const name = form.name.trim()
  const sortOrder = Number(form.sortOrder)
  if (!code || !name) {
    appStore.showError('请填写平台代号和名称')
    return
  }

  const payload: CustomPlatformUpsertPayload = {
    code,
    name,
    enabled: form.enabled,
    sort_order: Number.isFinite(sortOrder) ? sortOrder : 0,
  }

  saving.value = true
  try {
    if (editingPlatform.value) {
      await customPlatformsAPI.update(editingPlatform.value.id, payload)
      appStore.showSuccess(`已更新平台「${name}」`)
    } else {
      await customPlatformsAPI.create(payload)
      appStore.showSuccess(`已新增平台「${name}」`)
    }
    showDialog.value = false
    editingPlatform.value = null
    await loadCustomPlatforms()
  } catch (error) {
    appStore.showError(extractApiErrorMessage(error, '保存自定义平台失败'))
  } finally {
    saving.value = false
  }
}

function askDeletePlatform(item: CustomPlatform) {
  deleteTarget.value = item
  showDeleteConfirm.value = true
}

function closeDeleteConfirm() {
  if (saving.value) return
  showDeleteConfirm.value = false
  deleteTarget.value = null
}

async function confirmDeletePlatform() {
  const item = deleteTarget.value
  if (!item || saving.value) return

  saving.value = true
  try {
    await customPlatformsAPI.delete(item.id)
    appStore.showSuccess(`已删除平台「${item.name}」`)
    showDeleteConfirm.value = false
    deleteTarget.value = null
    await loadCustomPlatforms()
  } catch (error) {
    appStore.showError(extractApiErrorMessage(error, '删除自定义平台失败'))
  } finally {
    saving.value = false
  }
}

function formatTime(value: string) {
  if (!value) return '-'
  const date = new Date(value)
  if (Number.isNaN(date.getTime())) return '-'
  return date.toLocaleString('zh-CN')
}

onMounted(() => {
  void loadCustomPlatforms()
})
</script>

<style scoped>
/* ===== KPI 卡片区域 ===== */
.cp-actions-card {
  display: flex;
  flex-direction: column;
  gap: 1rem;
  padding: 1.125rem 1.25rem;
  border: 1px solid #e5e7eb;
  border-radius: 0.875rem;
  background: linear-gradient(135deg, #f8fafc 0%, #fff 50%);
  box-shadow: 0 1px 2px rgba(15, 23, 42, 0.04);
}

:global(.dark) .cp-actions-card {
  border-color: #374151;
  background: linear-gradient(135deg, #1f2937 0%, #111827 50%);
}

.cp-kpi-grid {
  display: grid;
  grid-template-columns: repeat(3, minmax(0, 1fr));
  gap: 0.75rem;
  flex: 1;
}

.cp-kpi {
  display: flex;
  align-items: flex-start;
  gap: 0.75rem;
  padding: 0.875rem 1rem;
  border-radius: 0.75rem;
  border: 1px solid transparent;
  transition: border-color 0.15s ease, box-shadow 0.15s ease;
}

.cp-kpi-default {
  background: #f9fafb;
  border-color: #e5e7eb;
}

:global(.dark) .cp-kpi-default {
  background: #1f2937;
  border-color: #374151;
}

.cp-kpi-success {
  background: #ecfdf5;
  border-color: #a7f3d0;
}

:global(.dark) .cp-kpi-success {
  background: rgba(22, 163, 74, 0.08);
  border-color: rgba(22, 163, 74, 0.25);
}

.cp-kpi-danger {
  background: #fef2f2;
  border-color: #fecaca;
}

:global(.dark) .cp-kpi-danger {
  background: rgba(220, 38, 38, 0.08);
  border-color: rgba(220, 38, 38, 0.25);
}

.cp-kpi-icon {
  display: grid;
  width: 2.25rem;
  height: 2.25rem;
  flex-shrink: 0;
  place-items: center;
  border-radius: 0.625rem;
}

.cp-kpi-icon-default {
  background: #e5e7eb;
  color: #6b7280;
}

:global(.dark) .cp-kpi-icon-default {
  background: #374151;
  color: #9ca3af;
}

.cp-kpi-icon-success {
  background: #d1fae5;
  color: #16a34a;
}

:global(.dark) .cp-kpi-icon-success {
  background: rgba(22, 163, 74, 0.2);
  color: #4ade80;
}

.cp-kpi-icon-danger {
  background: #fee2e2;
  color: #dc2626;
}

:global(.dark) .cp-kpi-icon-danger {
  background: rgba(220, 38, 38, 0.2);
  color: #fca5a5;
}

.cp-kpi-body {
  display: flex;
  flex-direction: column;
  gap: 0.25rem;
  min-width: 0;
}

.cp-kpi-label {
  font-size: 0.6875rem;
  font-weight: 600;
  letter-spacing: 0.04em;
  text-transform: uppercase;
  color: #6b7280;
}

:global(.dark) .cp-kpi-label {
  color: #9ca3af;
}

.cp-kpi-success .cp-kpi-label {
  color: #166534;
}

:global(.dark) .cp-kpi-success .cp-kpi-label {
  color: #6ee7b7;
}

.cp-kpi-danger .cp-kpi-label {
  color: #991b1b;
}

:global(.dark) .cp-kpi-danger .cp-kpi-label {
  color: #fca5a5;
}

.cp-kpi-value {
  font-size: 1.5rem;
  font-weight: 700;
  font-variant-numeric: tabular-nums;
  line-height: 1;
  color: #111827;
}

:global(.dark) .cp-kpi-value {
  color: #f9fafb;
}

.cp-actions-right {
  display: flex;
  flex-shrink: 0;
  align-items: center;
  gap: 0.5rem;
}

@media (min-width: 1024px) {
  .cp-actions-card {
    flex-direction: row;
    align-items: stretch;
  }
}

/* ===== 按钮 ===== */
.cp-btn {
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

:global(.dark) .cp-btn {
  background: #1f2937;
  border-color: #374151;
  color: #e5e7eb;
}

.cp-btn:hover {
  background: #f9fafb;
}

:global(.dark) .cp-btn:hover {
  background: #374151;
}

.cp-btn:disabled {
  cursor: not-allowed;
  opacity: 0.5;
}

.cp-btn-primary {
  color: #fff;
  border-color: #3b82f6;
  background: #3b82f6;
}

.cp-btn-primary:hover {
  background: #2563eb;
  border-color: #2563eb;
}

.cp-btn-refresh {
  color: #16a34a;
  border-color: #d1fae5;
  background: #f0fdf4;
}

:global(.dark) .cp-btn-refresh {
  color: #4ade80;
  border-color: rgba(22, 163, 74, 0.3);
  background: rgba(22, 163, 74, 0.1);
}

.cp-btn-refresh:hover {
  background: #dcfce7;
}

:global(.dark) .cp-btn-refresh:hover {
  background: rgba(22, 163, 74, 0.18);
}

.cp-btn-sm {
  min-height: 1.75rem;
  padding: 0.25rem 0.6rem;
  font-size: 0.75rem;
}

.cp-btn-edit {
  color: #3b82f6;
  border-color: #bfdbfe;
  background: #eff6ff;
}

:global(.dark) .cp-btn-edit {
  color: #93c5fd;
  border-color: rgba(59, 130, 246, 0.3);
  background: rgba(59, 130, 246, 0.1);
}

.cp-btn-edit:hover {
  background: #dbeafe;
}

.cp-btn-delete {
  color: #dc2626;
  border-color: #fecaca;
  background: #fef2f2;
}

:global(.dark) .cp-btn-delete {
  color: #fca5a5;
  border-color: rgba(220, 38, 38, 0.3);
  background: rgba(220, 38, 38, 0.1);
}

.cp-btn-delete:hover {
  background: #fee2e2;
}

.cp-btn-cancel {
  color: #6b7280;
  border-color: #e5e7eb;
  background: #fff;
}

:global(.dark) .cp-btn-cancel {
  color: #9ca3af;
  border-color: #374151;
  background: #1f2937;
}

/* ===== 筛选栏 ===== */
.cp-filter-bar {
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

:global(.dark) .cp-filter-bar {
  border-color: #374151;
  background: #1f2937;
}

.cp-filter-search {
  flex: 1;
  min-width: 0;
  max-width: 24rem;
}

.cp-filter-hint {
  color: #9ca3af;
  font-size: 0.8125rem;
  margin-left: auto;
}

:global(.dark) .cp-filter-hint {
  color: #6b7280;
}

/* ===== 表格 ===== */
.cp-table-wrapper {
  overflow: hidden;
  border: 1px solid #e5e7eb;
  border-radius: 0.875rem;
  background: #fff;
  box-shadow: 0 1px 2px rgba(15, 23, 42, 0.04);
}

:global(.dark) .cp-table-wrapper {
  border-color: #374151;
  background: #1f2937;
}

.cp-code-badge {
  display: inline-flex;
  align-items: center;
  padding: 0.2rem 0.55rem;
  border-radius: 0.35rem;
  font-size: 0.75rem;
  font-weight: 600;
  font-family: ui-monospace, SFMono-Regular, Consolas, "Liberation Mono", monospace;
  border: 1px solid #e5e7eb;
  background: #f3f4f6;
  color: #374151;
}

:global(.dark) .cp-code-badge {
  border-color: #374151;
  background: #374151;
  color: #e5e7eb;
}

.cp-cell-name {
  color: #111827;
  font-size: 0.875rem;
  font-weight: 600;
}

:global(.dark) .cp-cell-name {
  color: #f9fafb;
}

.cp-status-pill {
  display: inline-flex;
  align-items: center;
  gap: 0.35rem;
  padding: 0.2rem 0.6rem;
  border-radius: 9999px;
  font-size: 0.75rem;
  font-weight: 600;
}

.cp-status-pill-success {
  background: #ecfdf5;
  color: #166534;
}

:global(.dark) .cp-status-pill-success {
  background: rgba(22, 163, 74, 0.15);
  color: #6ee7b7;
}

.cp-status-pill-muted {
  background: #f3f4f6;
  color: #6b7280;
}

:global(.dark) .cp-status-pill-muted {
  background: #374151;
  color: #9ca3af;
}

.cp-status-dot {
  width: 0.375rem;
  height: 0.375rem;
  border-radius: 50%;
  flex-shrink: 0;
}

.cp-status-dot-success {
  background: #16a34a;
}

:global(.dark) .cp-status-dot-success {
  background: #4ade80;
}

.cp-status-dot-muted {
  background: #9ca3af;
}

:global(.dark) .cp-status-dot-muted {
  background: #6b7280;
}

.cp-cell-mono {
  font-family: ui-monospace, SFMono-Regular, Consolas, "Liberation Mono", monospace;
  font-size: 0.8125rem;
  color: #374151;
}

:global(.dark) .cp-cell-mono {
  color: #e5e7eb;
}

.cp-cell-time {
  color: #6b7280;
  font-size: 0.8125rem;
}

:global(.dark) .cp-cell-time {
  color: #9ca3af;
}

.cp-cell-actions {
  display: flex;
  justify-content: flex-end;
  gap: 0.4rem;
}

/* ===== 弹窗 ===== */
.cp-dialog-body {
  display: flex;
  flex-direction: column;
  gap: 1rem;
}

.cp-dialog-row {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 1rem;
}

.cp-dialog-field {
  display: flex;
  flex-direction: column;
  gap: 0.4rem;
}

.cp-dialog-label {
  font-size: 0.875rem;
  font-weight: 600;
  color: #111827;
}

:global(.dark) .cp-dialog-label {
  color: #f9fafb;
}

.cp-dialog-toggle {
  display: flex;
  align-items: center;
  gap: 0.75rem;
  padding: 0.75rem 1rem;
  border: 1px solid #e5e7eb;
  border-radius: 0.75rem;
  background: #f9fafb;
}

:global(.dark) .cp-dialog-toggle {
  border-color: #374151;
  background: #1f2937;
}

.cp-dialog-toggle-text {
  display: flex;
  flex-direction: column;
  gap: 0.15rem;
}

.cp-dialog-toggle-state {
  font-size: 0.875rem;
  font-weight: 600;
  color: #111827;
}

:global(.dark) .cp-dialog-toggle-state {
  color: #f9fafb;
}

.cp-dialog-toggle-hint {
  font-size: 0.75rem;
  color: #9ca3af;
}

:global(.dark) .cp-dialog-toggle-hint {
  color: #6b7280;
}

.cp-dialog-footer {
  display: flex;
  justify-content: flex-end;
  gap: 0.5rem;
}

/* 旋转动画 */
.cp-spin {
  animation: cp-spin 0.7s linear infinite;
}

@keyframes cp-spin {
  to { transform: rotate(360deg); }
}

/* ===== 响应式 ===== */
@media (max-width: 640px) {
  .cp-kpi-grid {
    grid-template-columns: 1fr;
  }

  .cp-filter-bar {
    flex-direction: column;
    align-items: stretch;
  }

  .cp-filter-search {
    max-width: none;
  }

  .cp-filter-hint {
    margin-left: 0;
  }

  .cp-dialog-row {
    grid-template-columns: 1fr;
  }
}
</style>
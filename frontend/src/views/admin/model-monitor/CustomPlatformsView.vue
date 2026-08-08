<template>
  <AppLayout>
    <TablePageLayout>
      <template #actions>
        <div class="flex flex-col gap-4 rounded-2xl border border-gray-200 bg-white p-4 shadow-sm dark:border-dark-700 dark:bg-dark-800 lg:flex-row lg:items-stretch lg:justify-between">
          <div class="grid flex-1 grid-cols-1 gap-3 sm:grid-cols-3">
            <div class="rounded-xl border border-gray-200 bg-gray-50/80 px-4 py-3 dark:border-dark-700 dark:bg-dark-900/50">
              <p class="text-xs font-medium uppercase tracking-wide text-gray-500 dark:text-dark-400">平台总数</p>
              <p class="mt-2 text-2xl font-semibold text-gray-900 dark:text-white">{{ totalCount }}</p>
            </div>
            <div class="rounded-xl border border-emerald-200 bg-emerald-50/80 px-4 py-3 dark:border-emerald-500/30 dark:bg-emerald-500/10">
              <p class="text-xs font-medium uppercase tracking-wide text-emerald-700 dark:text-emerald-300">启用中</p>
              <p class="mt-2 text-2xl font-semibold text-emerald-800 dark:text-emerald-200">{{ enabledCount }}</p>
            </div>
            <div class="rounded-xl border border-rose-200 bg-rose-50/80 px-4 py-3 dark:border-rose-500/30 dark:bg-rose-500/10">
              <p class="text-xs font-medium uppercase tracking-wide text-rose-700 dark:text-rose-300">停用中</p>
              <p class="mt-2 text-2xl font-semibold text-rose-800 dark:text-rose-200">{{ disabledCount }}</p>
            </div>
          </div>

          <div class="flex flex-wrap items-center gap-2 lg:flex-shrink-0 lg:justify-end">
            <button class="btn btn-secondary" :disabled="loading" @click="loadCustomPlatforms">
              {{ loading ? '刷新中…' : '刷新' }}
            </button>
            <button class="btn btn-primary" @click="openCreateDialog">
              新增平台
            </button>
          </div>
        </div>
      </template>

      <template #filters>
        <div class="flex flex-col gap-3 lg:flex-row lg:items-center">
          <div class="w-full lg:max-w-md">
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
          <div class="text-sm text-gray-500 dark:text-dark-400">
            这里维护的是模型监控和供应商模块共用的独立平台字典，不影响原有分组平台字段。
          </div>
        </div>
      </template>

      <template #table>
        <DataTable
          :columns="columns"
          :data="filteredPlatforms"
          :loading="loading"
          row-key="id"
        >
          <template #cell-code="{ value }">
            <span class="inline-flex items-center rounded-md border border-gray-200 bg-gray-50 px-2 py-0.5 text-xs font-mono font-medium text-gray-700 dark:border-dark-700 dark:bg-dark-900/50 dark:text-gray-200">
              {{ value }}
            </span>
          </template>

          <template #cell-name="{ value }">
            <span class="font-medium text-gray-900 dark:text-white">{{ value }}</span>
          </template>

          <template #cell-enabled="{ value }">
            <span
              :class="[
                'inline-flex items-center rounded-full px-2 py-0.5 text-xs font-medium',
                value
                  ? 'bg-emerald-500/10 text-emerald-700 dark:bg-emerald-500/15 dark:text-emerald-200'
                  : 'bg-gray-500/10 text-gray-600 dark:bg-dark-700 dark:text-dark-300'
              ]"
            >
              {{ value ? '启用' : '停用' }}
            </span>
          </template>

          <template #cell-sort_order="{ value }">
            <span class="font-mono text-sm text-gray-700 dark:text-gray-300">{{ value }}</span>
          </template>

          <template #cell-created_at="{ value }">
            <span class="text-sm text-gray-600 dark:text-dark-300">{{ formatTime(value) }}</span>
          </template>

          <template #cell-updated_at="{ value }">
            <span class="text-sm text-gray-600 dark:text-dark-300">{{ formatTime(value) }}</span>
          </template>

          <template #cell-actions="{ row }">
            <div class="flex justify-end gap-2">
              <button class="btn btn-secondary btn-sm" @click="openEditDialog(row)">编辑</button>
              <button class="btn btn-danger btn-sm" @click="askDeletePlatform(row)">删除</button>
            </div>
          </template>
        </DataTable>
      </template>
    </TablePageLayout>

    <BaseDialog
      :show="showDialog"
      :title="dialogTitle"
      width="wide"
      @close="closeDialog"
    >
      <div class="grid gap-4">
        <div class="grid gap-4 md:grid-cols-2">
          <Input
            v-model="form.code"
            label="平台代号"
            placeholder="例如 glm、deepseek、kimi"
            :disabled="saving"
            required
          />
          <Input
            v-model="form.name"
            label="显示名称"
            placeholder="例如 GLM、DeepSeek、Kimi"
            :disabled="saving"
            required
          />
        </div>

        <div class="grid gap-4 md:grid-cols-2">
          <Input
            v-model="form.sortOrder"
            type="number"
            label="排序"
            placeholder="数字越小越靠前"
            :disabled="saving"
          />
          <div>
            <label class="input-label mb-1.5 block">启用状态</label>
            <div class="flex items-center gap-3 rounded-xl border border-gray-200 bg-gray-50 px-4 py-3 dark:border-dark-700 dark:bg-dark-900/40">
              <Toggle v-model="form.enabled" :disabled="saving" />
              <div>
                <div class="text-sm font-medium text-gray-900 dark:text-white">
                  {{ form.enabled ? '启用' : '停用' }}
                </div>
                <div class="text-xs text-gray-500 dark:text-dark-400">
                  停用后不会出现在新配置的可选项中，但历史记录仍会保留。
                </div>
              </div>
            </div>
          </div>
        </div>
      </div>

      <template #footer>
        <div class="flex items-center justify-end gap-3">
          <button class="btn btn-secondary" :disabled="saving" @click="closeDialog">取消</button>
          <button class="btn btn-primary" :disabled="saving" @click="savePlatform">
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

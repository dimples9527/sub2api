<template>
  <AppLayout>
    <TablePageLayout>
      <template #actions>
        <div class="hv-actions-card">
          <div class="hv-kpi-grid">
            <div class="hv-kpi hv-kpi-default">
              <div class="hv-kpi-icon hv-kpi-icon-default">
                <Icon name="grid" size="md" aria-hidden="true" />
              </div>
              <div class="hv-kpi-body">
                <span class="hv-kpi-label">平台分组</span>
                <strong class="hv-kpi-value">{{ groupTotal }}</strong>
                <span class="hv-kpi-foot">检测历史里的分组桶</span>
              </div>
            </div>
            <div class="hv-kpi hv-kpi-warning">
              <div class="hv-kpi-icon hv-kpi-icon-warning">
                <Icon name="eyeOff" size="md" aria-hidden="true" />
              </div>
              <div class="hv-kpi-body">
                <span class="hv-kpi-label">已隐藏分组</span>
                <strong class="hv-kpi-value">{{ hiddenGroupCount }}</strong>
                <span class="hv-kpi-foot">普通用户看不到这些分组的记录</span>
              </div>
            </div>
            <div class="hv-kpi hv-kpi-violet">
              <div class="hv-kpi-icon hv-kpi-icon-violet">
                <Icon name="server" size="md" aria-hidden="true" />
              </div>
              <div class="hv-kpi-body">
                <span class="hv-kpi-label">已隐藏上游主机</span>
                <strong class="hv-kpi-value">{{ hiddenHostCount }}</strong>
                <span class="hv-kpi-foot">第三方上游主机桶</span>
              </div>
            </div>
          </div>

          <div class="hv-actions-right">
            <button class="hv-btn" :disabled="loading" @click="reload">
              <Icon name="refresh" size="sm" aria-hidden="true" :class="loading ? 'animate-spin' : ''" />
              <span>{{ loading ? '刷新中…' : '刷新' }}</span>
            </button>
          </div>
        </div>
      </template>

      <template #filters>
        <div class="hv-filter-panel">
          <div class="hv-filter-bar">
            <div class="hv-filter-search">
              <SearchInput v-model="searchQuery" placeholder="搜索分组名称、ID 或上游主机" />
            </div>
            <Select
              v-model="kindFilter"
              :options="kindOptions"
              placeholder="全部类型"
              class="w-40"
              :clearable="false"
            />
            <div class="hv-filter-hint">
              隐藏的分组 / 上游主机不会出现在普通用户的检测历史里；管理员不受影响，仍能看到全部记录。
            </div>
          </div>
        </div>
      </template>

      <template #table>
        <div class="hv-table-wrapper">
          <DataTable :columns="columns" :data="pagedRows" :loading="loading" row-key="key">
            <template #cell-kind="{ row }">
              <span :class="['hv-kind', row.kind === 'group' ? 'hv-kind-group' : 'hv-kind-host']">
                {{ row.kind === 'group' ? '平台分组' : '上游主机' }}
              </span>
            </template>

            <template #cell-title="{ row }">
              <div class="hv-cell-title">
                <strong :class="{ 'is-unavailable': row.unavailable }">{{ row.title }}</strong>
                <span v-if="row.unavailable" class="hv-cell-note">分组已不存在，历史记录仍按此配置隐藏</span>
              </div>
            </template>

            <template #cell-meta="{ row }">
              <span class="hv-cell-meta">{{ row.meta }}</span>
            </template>

            <template #cell-hidden="{ row }">
              <div class="hv-cell-toggle">
                <Toggle
                  :model-value="!row.hidden"
                  :disabled="savingKey !== null"
                  :aria-label="
                    row.hidden
                      ? `恢复「${row.title}」在检测历史中的显示`
                      : `在检测历史中隐藏「${row.title}」`
                  "
                  @update:model-value="(value) => toggleRow(row, !value)"
                />
                <span :class="['hv-state', row.hidden ? 'hv-state-hidden' : 'hv-state-visible']">
                  {{ row.hidden ? '已隐藏' : '显示中' }}
                </span>
              </div>
            </template>

            <template #empty>
              <EmptyState
                title="没有匹配的分组或上游主机"
                description="请尝试调整搜索词或类型筛选。"
              />
            </template>
          </DataTable>
        </div>
      </template>

      <template #pagination>
        <Pagination
          v-if="filteredRows.length > 0"
          :page="page"
          :total="filteredRows.length"
          :page-size="pageSize"
          @update:page="page = $event"
          @update:pageSize="changePageSize"
        />
      </template>
    </TablePageLayout>
  </AppLayout>
</template>

<script setup lang="ts">
import { computed, onMounted, ref, watch } from 'vue'
import { useAppStore } from '@/stores/app'
import { extractApiErrorMessage } from '@/utils/apiError'
import { adminAPI } from '@/api/admin'
import { getPersistedPageSize } from '@/composables/usePersistedPageSize'
import type { Column } from '@/components/common/types'
import AppLayout from '@/components/layout/AppLayout.vue'
import TablePageLayout from '@/components/layout/TablePageLayout.vue'
import DataTable from '@/components/common/DataTable.vue'
import SearchInput from '@/components/common/SearchInput.vue'
import Select from '@/components/common/Select.vue'
import Toggle from '@/components/common/Toggle.vue'
import EmptyState from '@/components/common/EmptyState.vue'
import Icon from '@/components/icons/Icon.vue'
import Pagination from '@/components/common/Pagination.vue'

/**
 * 检测历史可见性配置页。
 *
 * 检测历史墙的归类桶只有两种（与用户端 groupBucketOf 一致）：
 *   1) 命中平台分组的记录 → 分组桶（group_id 是发起时的快照）；
 *   2) 没命中分组的第三方上游记录 → 主机名桶。
 * 本页把两类桶铺成一张表，逐项开关「普通用户是否可见」；管理员账号不受影响。
 * 分组清单来自管理端分组接口（含已停用分组），主机清单来自历史里实际出现过的
 * 未命中分组记录（服务端按主机名前缀归一统计，含端口剥离）。
 */

type VisibilityKind = 'group' | 'host'

interface VisibilityRow {
  key: string
  kind: VisibilityKind
  title: string
  meta: string
  hidden: boolean
  /** 分组行：分组 ID（含已不存在的历史分组）。 */
  groupId?: number
  /** 主机行：小写主机名。 */
  host?: string
  /** 分组已不存在（只剩历史记录被隐藏）。 */
  unavailable?: boolean
  /** 主机行排序用：未命中分组的记录数。 */
  runCount?: number
}

const appStore = useAppStore()

const loading = ref(false)
const rows = ref<VisibilityRow[]>([])
const searchQuery = ref('')
const kindFilter = ref<'all' | VisibilityKind>('all')
const page = ref(1)
const pageSize = ref(getPersistedPageSize())
const savingKey = ref<string | null>(null)

const kindOptions = [
  { value: 'all', label: '全部类型' },
  { value: 'group', label: '平台分组' },
  { value: 'host', label: '上游主机' },
]

const columns: Column[] = [
  { key: 'kind', label: '类型', sortable: false },
  { key: 'title', label: '名称', sortable: false },
  { key: 'meta', label: '说明', sortable: false },
  { key: 'hidden', label: '用户端显示', sortable: false },
]

const groupTotal = computed(() => rows.value.filter((row) => row.kind === 'group' && !row.unavailable).length)
const hiddenGroupCount = computed(() => rows.value.filter((row) => row.kind === 'group' && row.hidden).length)
const hiddenHostCount = computed(() => rows.value.filter((row) => row.kind === 'host' && row.hidden).length)

const filteredRows = computed(() => {
  const keyword = searchQuery.value.trim().toLowerCase()
  return rows.value.filter((row) => {
    if (kindFilter.value !== 'all' && row.kind !== kindFilter.value) return false
    if (!keyword) return true
    return `${row.title} ${row.meta}`.toLowerCase().includes(keyword)
  })
})

const pagedRows = computed(() => {
  const start = (page.value - 1) * pageSize.value
  return filteredRows.value.slice(start, start + pageSize.value)
})

watch([searchQuery, kindFilter], () => {
  page.value = 1
})

watch([() => filteredRows.value.length, pageSize], ([total, size]) => {
  const lastPage = Math.max(1, Math.ceil(total / size))
  if (page.value > lastPage) page.value = lastPage
})

function changePageSize(size: number) {
  pageSize.value = size
  page.value = 1
}

async function reload() {
  loading.value = true
  try {
    const [groups, visibility] = await Promise.all([
      adminAPI.groups.getAllIncludingInactive(),
      adminAPI.modelDegradation.getModelDegradationHistoryVisibility(),
    ])

    const hiddenGroupIds = new Set(visibility.hidden_group_ids)
    const hiddenHosts = new Set(visibility.hidden_hosts)
    const next: VisibilityRow[] = []

    const knownGroupIds = new Set<number>()
    const sortedGroups = [...groups].sort((left, right) => left.name.localeCompare(right.name, 'zh-Hans-CN'))
    for (const group of sortedGroups) {
      knownGroupIds.add(group.id)
      next.push({
        key: `g:${group.id}`,
        kind: 'group',
        title: group.name,
        meta: `分组 ID：${group.id}`,
        hidden: hiddenGroupIds.has(group.id),
        groupId: group.id,
      })
    }
    // 分组被删除后隐藏配置仍然保留（历史记录里的 group_id 是发起时的快照），
    // 补一行失效入口，让管理员可以手动取消隐藏。
    for (const groupId of visibility.hidden_group_ids) {
      if (knownGroupIds.has(groupId)) continue
      next.push({
        key: `g:${groupId}`,
        kind: 'group',
        title: `分组 #${groupId}`,
        meta: `分组 ID：${groupId}（已不存在）`,
        hidden: true,
        groupId,
        unavailable: true,
      })
    }

    const sortedHosts = [...visibility.hosts].sort(
      (left, right) => right.run_count - left.run_count || left.host.localeCompare(right.host)
    )
    const statHosts = new Set<string>()
    for (const stat of sortedHosts) {
      statHosts.add(stat.host)
      next.push({
        key: `h:${stat.host}`,
        kind: 'host',
        title: stat.host,
        meta: `未命中分组的记录：${stat.run_count} 条`,
        hidden: hiddenHosts.has(stat.host),
        host: stat.host,
        runCount: stat.run_count,
      })
    }
    // 隐藏但历史记录已被清理的主机：保留行，方便取消隐藏。
    for (const host of visibility.hidden_hosts) {
      if (statHosts.has(host)) continue
      next.push({
        key: `h:${host}`,
        kind: 'host',
        title: host,
        meta: '暂无历史记录',
        hidden: true,
        host,
        runCount: 0,
      })
    }

    rows.value = next
  } catch (error) {
    appStore.showError(extractApiErrorMessage(error, '加载检测历史可见性配置失败'))
  } finally {
    loading.value = false
  }
}

async function toggleRow(row: VisibilityRow, nextHidden: boolean) {
  if (savingKey.value !== null || row.hidden === nextHidden) return

  const previous = row.hidden
  row.hidden = nextHidden
  savingKey.value = row.key
  try {
    if (row.kind === 'group' && row.groupId) {
      await adminAPI.modelDegradation.setModelDegradationHistoryGroupHidden(row.groupId, nextHidden)
    } else if (row.kind === 'host' && row.host) {
      await adminAPI.modelDegradation.setModelDegradationHistoryHostHidden(row.host, nextHidden)
    } else {
      return
    }
    appStore.showSuccess(
      nextHidden
        ? `「${row.title}」已从普通用户的检测历史中隐藏`
        : `「${row.title}」已恢复在检测历史中的显示`
    )
  } catch (error) {
    row.hidden = previous
    appStore.showError(extractApiErrorMessage(error, '保存检测历史可见性配置失败'))
  } finally {
    savingKey.value = null
  }
}

onMounted(reload)
</script>

<style scoped>
/* ===== KPI 卡片区域 ===== */
.hv-actions-card {
  display: flex;
  flex-direction: column;
  gap: 1rem;
  padding: 1.125rem 1.25rem;
  border: 1px solid #e5e7eb;
  border-radius: 0.875rem;
  background: linear-gradient(135deg, #f8fafc 0%, #fff 50%);
  box-shadow: 0 1px 2px rgba(15, 23, 42, 0.04);
}

.dark .hv-actions-card {
  border-color: #374151;
  background: linear-gradient(135deg, #1f2937 0%, #111827 50%);
}

.hv-kpi-grid {
  display: grid;
  grid-template-columns: repeat(3, minmax(0, 1fr));
  gap: 0.75rem;
  flex: 1;
}

.hv-kpi {
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

.hv-kpi::before {
  content: '';
  position: absolute;
  top: 0;
  left: 0;
  right: 0;
  height: 3px;
}

.hv-kpi:hover {
  transform: translateY(-1px);
  box-shadow: 0 6px 16px -6px rgba(15, 23, 42, 0.14);
}

.hv-kpi-default {
  background: linear-gradient(180deg, #f0fdfa 0%, #f9fafb 55%);
  border-color: #ccfbf1;
}

.hv-kpi-default::before {
  background: linear-gradient(90deg, #0d9488, #2dd4bf);
}

.dark .hv-kpi-default {
  background: linear-gradient(180deg, rgba(20, 184, 166, 0.12) 0%, #1f2937 55%);
  border-color: rgba(20, 184, 166, 0.3);
}

.hv-kpi-warning {
  background: linear-gradient(180deg, #fffbeb 0%, #fefce8 60%);
  border-color: #fde68a;
}

.hv-kpi-warning::before {
  background: linear-gradient(90deg, #f59e0b, #f97316);
}

.dark .hv-kpi-warning {
  background: linear-gradient(180deg, rgba(217, 119, 6, 0.14) 0%, #1f2937 55%);
  border-color: rgba(217, 119, 6, 0.3);
}

.hv-kpi-violet {
  background: linear-gradient(180deg, #f5f3ff 0%, #faf9ff 60%);
  border-color: #ddd6fe;
}

.hv-kpi-violet::before {
  background: linear-gradient(90deg, #7c3aed, #a78bfa);
}

.dark .hv-kpi-violet {
  background: linear-gradient(180deg, rgba(124, 58, 237, 0.14) 0%, #1f2937 55%);
  border-color: rgba(124, 58, 237, 0.3);
}

.hv-kpi-icon {
  display: grid;
  width: 2.25rem;
  height: 2.25rem;
  flex-shrink: 0;
  place-items: center;
  border-radius: 0.625rem;
  box-shadow: 0 1px 2px rgba(15, 23, 42, 0.08);
}

.hv-kpi-icon-default {
  background: linear-gradient(135deg, #0d9488, #14b8a6);
  color: #fff;
}

.dark .hv-kpi-icon-default {
  background: linear-gradient(135deg, #14b8a6, #2dd4bf);
}

.hv-kpi-icon-warning {
  background: linear-gradient(135deg, #f59e0b, #f97316);
  color: #fff;
}

.dark .hv-kpi-icon-warning {
  background: linear-gradient(135deg, #fbbf24, #fb923c);
}

.hv-kpi-icon-violet {
  background: linear-gradient(135deg, #7c3aed, #8b5cf6);
  color: #fff;
}

.dark .hv-kpi-icon-violet {
  background: linear-gradient(135deg, #8b5cf6, #a78bfa);
}

.hv-kpi-body {
  display: flex;
  flex-direction: column;
  gap: 0.25rem;
  min-width: 0;
}

.hv-kpi-label {
  font-size: 0.75rem;
  font-weight: 600;
  letter-spacing: 0.02em;
  color: #0f766e;
}

.dark .hv-kpi-label {
  color: #5eead4;
}

.hv-kpi-warning .hv-kpi-label {
  color: #b45309;
}

.dark .hv-kpi-warning .hv-kpi-label {
  color: #fbbf24;
}

.hv-kpi-violet .hv-kpi-label {
  color: #6d28d9;
}

.dark .hv-kpi-violet .hv-kpi-label {
  color: #c4b5fd;
}

.hv-kpi-value {
  font-size: 1.5rem;
  font-weight: 700;
  font-variant-numeric: tabular-nums;
  line-height: 1;
  color: #111827;
}

.dark .hv-kpi-value {
  color: #f9fafb;
}

.hv-kpi-foot {
  margin-top: 0.1rem;
  font-size: 0.6875rem;
  color: #9ca3af;
}

.dark .hv-kpi-foot {
  color: #6b7280;
}

.hv-actions-right {
  display: flex;
  flex-shrink: 0;
  align-items: center;
  gap: 0.5rem;
}

@media (min-width: 1024px) {
  .hv-actions-card {
    flex-direction: row;
    align-items: stretch;
  }
}

/* ===== 按钮 ===== */
.hv-btn {
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

.dark .hv-btn {
  background: #1f2937;
  border-color: #374151;
  color: #e5e7eb;
}

.hv-btn:hover {
  background: #f9fafb;
}

.dark .hv-btn:hover {
  background: #374151;
}

.hv-btn:disabled {
  cursor: not-allowed;
  opacity: 0.5;
}

.hv-btn:focus-visible {
  outline: none;
  box-shadow: 0 0 0 2px rgba(124, 58, 237, 0.4);
}

/* ===== 筛选面板 ===== */
.hv-filter-panel {
  display: flex;
  flex-direction: column;
  gap: 0.625rem;
}

.hv-filter-bar {
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

.dark .hv-filter-bar {
  border-color: #374151;
  background: #1f2937;
}

.hv-filter-search {
  flex: 1 1 16rem;
  min-width: 12rem;
  max-width: 24rem;
}

.hv-filter-hint {
  flex: 1 1 100%;
  font-size: 0.75rem;
  color: #6b7280;
}

.dark .hv-filter-hint {
  color: #9ca3af;
}

/* ===== 表格 ===== */
.hv-table-wrapper {
  display: flex;
  /* 让表格区吃掉布局剩余高度并由内部 .table-wrapper 滚动；
     缺了 flex / min-height 时超出部分会被外层 overflow-hidden 裁掉。 */
  flex: 1 1 auto;
  min-height: 0;
  flex-direction: column;
}

.hv-kind {
  display: inline-flex;
  align-items: center;
  padding: 0.15rem 0.55rem;
  border-radius: 9999px;
  font-size: 0.75rem;
  font-weight: 500;
  white-space: nowrap;
}

.hv-kind-group {
  color: #0f766e;
  background: #ccfbf1;
  border: 1px solid #99f6e4;
}

.dark .hv-kind-group {
  color: #5eead4;
  background: rgba(20, 184, 166, 0.12);
  border-color: rgba(20, 184, 166, 0.3);
}

.hv-kind-host {
  color: #6d28d9;
  background: #ede9fe;
  border: 1px solid #ddd6fe;
}

.dark .hv-kind-host {
  color: #c4b5fd;
  background: rgba(124, 58, 237, 0.12);
  border-color: rgba(124, 58, 237, 0.3);
}

.hv-cell-title {
  display: flex;
  flex-direction: column;
  gap: 0.15rem;
  min-width: 0;
}

.hv-cell-title strong {
  color: #111827;
  font-weight: 600;
  word-break: break-all;
}

.dark .hv-cell-title strong {
  color: #f9fafb;
}

.hv-cell-title strong.is-unavailable {
  color: #9ca3af;
}

.dark .hv-cell-title strong.is-unavailable {
  color: #6b7280;
}

.hv-cell-note {
  font-size: 0.6875rem;
  color: #b45309;
}

.dark .hv-cell-note {
  color: #fbbf24;
}

.hv-cell-meta {
  font-size: 0.8125rem;
  color: #6b7280;
  font-variant-numeric: tabular-nums;
}

.dark .hv-cell-meta {
  color: #9ca3af;
}

.hv-cell-toggle {
  display: inline-flex;
  align-items: center;
  gap: 0.5rem;
}

.hv-state {
  font-size: 0.75rem;
  font-weight: 500;
}

.hv-state-visible {
  color: #059669;
}

.dark .hv-state-visible {
  color: #34d399;
}

.hv-state-hidden {
  color: #b45309;
}

.dark .hv-state-hidden {
  color: #fbbf24;
}
</style>

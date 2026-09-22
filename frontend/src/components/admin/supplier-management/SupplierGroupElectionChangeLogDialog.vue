<template>
  <BaseDialog :show="show" :title="dialogTitle" width="full" @close="emit('close')">
    <div class="sp-election-log-dialog">
      <p class="sp-election-log-hint">
        只显示调度开关真的被拨动的记录：择优调度把某个账号从「开」改成「关」，或重新选回「开」。
        未变更、未测试、写库失败的记录不在这里。
      </p>

      <div class="sp-election-log-filters">
        <span v-if="activeGroupID" class="sp-election-log-group-chip">
          仅看分组：{{ activeGroupLabel || `分组 ${activeGroupID}` }}
          <button type="button" class="sp-election-log-chip-clear" title="查看全部分组" @click="clearGroup">
            查看全部
          </button>
        </span>
        <span v-if="activeAccountID" class="sp-election-log-group-chip">
          仅看账号：{{ activeAccountLabel || `账号 ${activeAccountID}` }}
          <button type="button" class="sp-election-log-chip-clear" title="查看全部账号" @click="clearAccount">
            查看全部
          </button>
        </span>
        <span v-if="runFilter" class="sp-election-log-group-chip">
          仅看批次：#{{ runFilter }}
          <button type="button" class="sp-election-log-chip-clear" title="查看全部批次" @click="clearRun">
            查看全部
          </button>
        </span>
        <div class="sp-election-log-filter">
          <span class="sr-only">开关方向</span>
          <Select v-model="filters.direction" :options="directionOptions" :searchable="false" @update:model-value="applyFilters" />
        </div>
        <div class="sp-election-log-filter sp-election-log-search">
          <span class="sr-only">按账号名搜索</span>
          <Input v-model="filters.search" placeholder="按账号名搜索" @enter="applyFilters" />
        </div>
        <div class="sp-election-log-filter">
          <span class="sr-only">起始日期</span>
          <Input v-model="filters.startedFrom" type="date" />
        </div>
        <span class="sp-election-log-range-sep">至</span>
        <div class="sp-election-log-filter">
          <span class="sr-only">截止日期</span>
          <Input v-model="filters.startedTo" type="date" />
        </div>
        <button class="sp-button small primary" type="button" :disabled="loading" @click="applyFilters">
          {{ loading ? '查询中' : '查询' }}
        </button>
        <button class="sp-button small ghost" type="button" :disabled="loading || !hasActiveFilters" @click="resetFilters">
          重置
        </button>
      </div>

      <div class="sp-election-log-table-region">
        <DataTable
          :columns="columns"
          :data="items"
          :loading="loading"
          :row-key="rowKey"
          :sticky-actions-column="false"
        >
          <template #cell-run="{ row: log }">
            <button
              type="button"
              class="sp-election-log-run"
              :class="{ 'is-active': runFilter === log.run_id }"
              :title="runFilter === log.run_id ? '已锁定该批次' : '只看这一批次的切换'"
              @click="filterByRun(log.run_id)"
            >
              #{{ log.run_id }}
            </button>
          </template>
          <template #cell-changed_at="{ row: log }">
            <span class="sp-election-log-time">{{ formatTime(log.changed_at) }}</span>
          </template>
          <template #cell-account="{ row: log }">
            <strong class="sp-election-log-account">{{ log.account_name || `账号 ${log.account_id}` }}</strong>
            <span v-if="log.platform" class="sp-election-log-platform" :class="platformTextClass(log.platform)">{{ log.platform }}</span>
            <small class="sp-election-log-groups">{{ groupText(log) }}</small>
          </template>
          <template #cell-direction="{ row: log }">
            <span class="sp-election-log-direction" :class="log.direction === 'enabled' ? 'good' : 'bad'">
              {{ directionText(log) }}
            </span>
            <small class="sp-election-log-switch">{{ log.schedulable_before ? '开' : '关' }} → {{ log.schedulable_after ? '开' : '关' }}</small>
          </template>
          <template #cell-test_status="{ row: log }">
            <span class="sp-status" :class="log.test_status === 'success' ? 'good' : 'bad'">{{ testStatusText(log.test_status) }}</span>
          </template>
          <template #cell-latency_ms="{ row: log }">
            <span class="sp-election-log-latency" :class="{ 'is-empty': !log.latency_ms }">{{ latencyText(log.latency_ms) }}</span>
          </template>
          <template #cell-reason="{ row: log }">
            <small class="sp-election-log-reason">{{ log.reason || '—' }}</small>
            <small v-if="log.error_message" class="sp-election-log-error">{{ log.error_message }}</small>
          </template>
          <!-- 空态要区分「真的没有切换」和「筛选太窄」—— 用同一句话会让人以为功能坏了。 -->
          <template #empty>
            {{ hasActiveFilters ? '当前筛选条件下没有调度切换记录，试试放宽时间范围或换个方向。' : '最近还没有发生调度切换。' }}
          </template>
        </DataTable>
        <Pagination
          v-if="total > 0"
          class="sp-election-log-pagination"
          :page="page"
          :total="total"
          :page-size="pageSize"
          :show-page-size-selector="true"
          @update:page="changePage"
          @update:page-size="changePageSize"
        />
      </div>
    </div>
    <template #footer>
      <button class="sp-button primary" type="button" @click="emit('close')">关闭</button>
    </template>
  </BaseDialog>
</template>

<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { listGroupElectionChangeLogs, type SupplierGroupElectionChangeLog } from '@/api/admin/supplierAutomation'
import BaseDialog from '@/components/common/BaseDialog.vue'
import DataTable from '@/components/common/DataTable.vue'
import Input from '@/components/common/Input.vue'
import Pagination from '@/components/common/Pagination.vue'
import Select, { type SelectOption } from '@/components/common/Select.vue'
import type { Column } from '@/components/common/types'
import { useAppStore } from '@/stores/app'
import { formatTime } from '@/utils/format'
import { platformTextClass } from '@/utils/platformColors'

interface Props {
  show: boolean
  /** 锁定到某个本地分组；为空表示看全部分组。 */
  groupId?: number | null
  groupLabel?: string
  /** 锁定到某个本地账号；为空表示看全部账号。日志里的 account_id 是本地账号 ID。 */
  accountId?: number | null
  accountLabel?: string
}

const props = withDefaults(defineProps<Props>(), {
  groupId: null,
  groupLabel: '',
  accountId: null,
  accountLabel: '',
})

const emit = defineEmits<{ close: [] }>()

const appStore = useAppStore()

const loading = ref(false)
const items = ref<SupplierGroupElectionChangeLog[]>([])
const total = ref(0)
const page = ref(1)
const pageSize = ref(20)
// direction 用字面量联合而不是 string：API 只认 enabled/disabled，写错的值会被后端当成"不过滤"静默吞掉。
const filters = ref<{ direction: 'all' | 'enabled' | 'disabled'; search: string; startedFrom: string; startedTo: string }>({
  direction: 'all',
  search: '',
  startedFrom: '',
  startedTo: '',
})
// 从某个分组/账号进来后又点了「查看全部」，此时以组件内部状态为准。
const clearedGroup = ref(false)
const clearedAccount = ref(false)
// 任务批次筛选纯内部：从表格里点某条记录的批次号锁进来，再点「查看全部」清掉。
const runFilter = ref<number | null>(null)

const activeGroupID = computed(() => (clearedGroup.value ? null : props.groupId ?? null))
const activeGroupLabel = computed(() => (clearedGroup.value ? '' : props.groupLabel))
const activeAccountID = computed(() => (clearedAccount.value ? null : props.accountId ?? null))
const activeAccountLabel = computed(() => (clearedAccount.value ? '' : props.accountLabel))
const dialogTitle = computed(() => {
  if (activeGroupID.value) return `调度切换日志 · ${activeGroupLabel.value || `分组 ${activeGroupID.value}`}`
  if (activeAccountID.value) return `调度切换日志 · ${activeAccountLabel.value || `账号 ${activeAccountID.value}`}`
  return '分组调度切换日志'
})
const hasActiveFilters = computed(() => (
  filters.value.direction !== 'all'
  || filters.value.search.trim() !== ''
  || filters.value.startedFrom !== ''
  || filters.value.startedTo !== ''
  || activeGroupID.value !== null
  || activeAccountID.value !== null
  || runFilter.value !== null
))

const directionOptions: SelectOption[] = [
  { value: 'all', label: '全部方向' },
  { value: 'disabled', label: '只看关闭' },
  { value: 'enabled', label: '只看开启' },
]

const columns: Column[] = [
  { key: 'run', label: '任务批次', class: 'min-w-[100px]' },
  { key: 'changed_at', label: '切换时间', class: 'min-w-[150px]' },
  { key: 'account', label: '账号', class: 'min-w-[150px]' },
  { key: 'direction', label: '调度变更', class: 'min-w-[110px]' },
  { key: 'test_status', label: '测试状态', class: 'min-w-[90px]' },
  { key: 'healthy_count', label: '连续成功', class: 'min-w-[80px]' },
  { key: 'latency_ms', label: '测试用时', class: 'min-w-[90px]' },
  { key: 'reason', label: '原因', class: 'min-w-[200px]' },
]

// 同一次运行里可能有多个账号同时变更，run_id 会重复 —— 键必须用 run_id + account_id。
function rowKey(log: SupplierGroupElectionChangeLog) {
  return `${log.run_id}-${log.account_id}`
}

function directionText(log: SupplierGroupElectionChangeLog) {
  if (log.direction === 'enabled') return '开启调度'
  if (log.direction === 'disabled') return '关闭调度'
  return log.schedulable_after ? '开启调度' : '关闭调度'
}

function testStatusText(status?: string) {
  if (status === 'success') return '成功'
  if (status === 'failed') return '失败'
  if (status === 'slow') return '慢响应'
  return '未测试'
}

function latencyText(latencyMs?: number) {
  if (!latencyMs || latencyMs <= 0) return '—'
  if (latencyMs < 1000) return `${latencyMs}ms`
  return `${(latencyMs / 1000).toFixed(1)}s`
}

function groupText(log: SupplierGroupElectionChangeLog) {
  const names = log.group_names?.filter(Boolean) || []
  if (names.length) return names.join('、')
  const ids = log.group_ids?.filter(Boolean) || []
  if (ids.length) return ids.map(id => `分组 ${id}`).join('、')
  return '—'
}

async function load() {
  loading.value = true
  try {
    const result = await listGroupElectionChangeLogs({
      group_id: activeGroupID.value ?? undefined,
      account_id: activeAccountID.value ?? undefined,
      run_id: runFilter.value ?? undefined,
      search: filters.value.search.trim() || undefined,
      direction: filters.value.direction === 'all' ? undefined : filters.value.direction,
      started_from: filters.value.startedFrom || undefined,
      started_to: filters.value.startedTo || undefined,
      page: page.value,
      page_size: pageSize.value,
    })
    items.value = result.items
    total.value = result.total
    page.value = result.page
    pageSize.value = result.page_size
  } catch (err) {
    appStore.showError(err instanceof Error ? err.message : '加载调度切换日志失败')
  } finally {
    loading.value = false
  }
}

function applyFilters() {
  page.value = 1
  return load()
}

function resetFilters() {
  // 重置清筛选条件，但把「锁定到某个分组 / 账号」保留 —— 用户是冲着它点开弹窗的，
  // 一起清掉会让人误以为看到的是全局日志。换对象要点「查看全部」。
  filters.value = { direction: 'all', search: '', startedFrom: '', startedTo: '' }
  // 批次锁定是弹窗内点出来的临时筛选，不属于「进来时的对象」，重置一并清掉。
  runFilter.value = null
  page.value = 1
  return load()
}

function clearGroup() {
  clearedGroup.value = true
  page.value = 1
  return load()
}

function clearAccount() {
  clearedAccount.value = true
  page.value = 1
  return load()
}

function filterByRun(runId: number) {
  // 再点已锁定的批次号 = 取消锁定，省得非要移到 chip 上点「查看全部」。
  runFilter.value = runFilter.value === runId ? null : runId
  page.value = 1
  return load()
}

function clearRun() {
  runFilter.value = null
  page.value = 1
  return load()
}

function changePage(next: number) {
  const maxPage = Math.max(1, Math.ceil(total.value / pageSize.value))
  page.value = Math.min(Math.max(1, next), maxPage)
  return load()
}

function changePageSize(size: number) {
  pageSize.value = size
  page.value = 1
  return load()
}

watch(() => props.show, (visible) => {
  if (!visible) return
  clearedGroup.value = false
  clearedAccount.value = false
  runFilter.value = null
  page.value = 1
  void load()
})

watch(() => props.groupId, () => {
  if (!props.show) return
  clearedGroup.value = false
  page.value = 1
  void load()
})

watch(() => props.accountId, () => {
  if (!props.show) return
  clearedAccount.value = false
  page.value = 1
  void load()
})
</script>

<style scoped>
/* BaseDialog 会 Teleport 到 body，拿不到页面根节点上的 --sp-*，
   所以这份日志用到的变量必须在弹窗根元素上重新声明一份。 */
.sp-election-log-dialog {
  /* 与三个入口按钮同色：一个功能一个颜色，点开后才不会觉得跳到了别的功能。 */
  --sp-election-log-accent: #ea580c;
  --sp-election-log-muted: #64748b;
  --sp-election-log-line: #e5e7eb;
  --sp-election-log-panel: #ffffff;
  --sp-election-log-soft: #f1f5f9;
  display: grid;
  gap: 12px;
}

/* BaseDialog 的 full 预设最宽到 max-w-7xl(1280px)，这份日志列多，再放宽到接近整屏。
   只命中「装着本日志」的那一个 modal-content，不动其他用 full 的弹窗。 */
:global(.modal-content:has(.sp-election-log-dialog)) {
  max-width: min(1600px, 95vw);
}

:global(.dark) .sp-election-log-dialog {
  --sp-election-log-accent: #fb923c;
  --sp-election-log-muted: #94a3b8;
  --sp-election-log-line: #374151;
  --sp-election-log-panel: #1f2937;
  --sp-election-log-soft: #374151;
}

.sp-election-log-hint {
  margin: 0;
  color: var(--sp-election-log-muted);
  font-size: 12px;
  line-height: 1.6;
}

.sp-election-log-filters {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
  align-items: center;
}

.sp-election-log-filter {
  min-width: 120px;
}

.sp-election-log-search {
  min-width: 160px;
}

.sp-election-log-group-chip {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  border: 1px solid color-mix(in srgb, var(--sp-election-log-accent) 35%, var(--sp-election-log-line));
  border-radius: 999px;
  padding: 3px 4px 3px 10px;
  background: color-mix(in srgb, var(--sp-election-log-accent) 10%, var(--sp-election-log-panel));
  color: var(--sp-election-log-accent);
  font-size: 12px;
  font-weight: 700;
}

.sp-election-log-chip-clear {
  border: 0;
  border-radius: 999px;
  padding: 2px 8px;
  background: transparent;
  color: inherit;
  font-size: 11px;
  font-weight: 700;
  cursor: pointer;
}

.sp-election-log-chip-clear:hover {
  background: color-mix(in srgb, var(--sp-election-log-accent) 18%, var(--sp-election-log-panel));
}

.sp-election-log-range-sep {
  color: var(--sp-election-log-muted);
  font-size: 12px;
}

.sp-election-log-table-region {
  overflow: hidden;
  border: 1px solid color-mix(in srgb, var(--sp-election-log-accent) 18%, var(--sp-election-log-soft));
  border-radius: 10px;
}

.sp-election-log-run {
  border: 1px solid color-mix(in srgb, var(--sp-election-log-accent) 30%, var(--sp-election-log-line));
  border-radius: 6px;
  padding: 2px 8px;
  background: transparent;
  color: var(--sp-election-log-accent);
  font-size: 12px;
  font-weight: 700;
  font-variant-numeric: tabular-nums;
  cursor: pointer;
}

.sp-election-log-run:hover {
  background: color-mix(in srgb, var(--sp-election-log-accent) 12%, var(--sp-election-log-panel));
}

.sp-election-log-run.is-active {
  background: var(--sp-election-log-accent);
  border-color: var(--sp-election-log-accent);
  color: #fff;
}

.sp-election-log-account {
  font-weight: 750;
}

/* 只给版式不给 color：颜色由 platformTextClass 的 Tailwind 类决定。
   这里写 color 会与工具类同权重（0,1,0），顺序由构建决定，平台色会被静默压掉。 */
.sp-election-log-platform {
  margin-left: 6px;
  font-size: 11px;
  font-weight: 700;
}

.sp-election-log-groups {
  display: block;
  color: var(--sp-election-log-muted);
  font-size: 11px;
}

.sp-election-log-direction {
  font-weight: 750;
}

.sp-election-log-direction.good {
  color: #16a34a;
}

.sp-election-log-direction.bad {
  color: #dc2626;
}

:global(.dark) .sp-election-log-direction.good {
  color: #4ade80;
}

:global(.dark) .sp-election-log-direction.bad {
  color: #f87171;
}

.sp-election-log-switch {
  display: block;
  color: var(--sp-election-log-muted);
  font-size: 11px;
  font-variant-numeric: tabular-nums;
}

.sp-election-log-latency.is-empty {
  color: var(--sp-election-log-muted);
}

.sp-election-log-reason {
  display: block;
  color: var(--sp-election-log-muted);
  font-size: 11px;
  line-height: 1.5;
}

.sp-election-log-error {
  display: block;
  color: #dc2626;
  font-size: 11px;
  line-height: 1.5;
}

:global(.dark) .sp-election-log-error {
  color: #f87171;
}
</style>

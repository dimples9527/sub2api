<template>
  <BaseDialog :show="show" :title="dialogTitle" width="full" @close="emit('close')">
    <div class="sp-election-log-dialog">
      <p class="sp-election-log-hint">
        只显示调度开关真的被拨动的记录：择优调度把某个账号从「开」改成「关」，或重新选回「开」。
        未变更、未测试、写库失败的记录不在这里。
        按分组归类展示，每组标出该组的开 / 关条数；组内再按批次分块，不同批次不混在一起。
        一个账号同属多个分组时，会在它所属的每个分组下各出现一次。
        分页按记录切分，所以同一个分组或同一个批次可能被分到相邻两页，上方条数也只统计当前这一页。
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
        <div class="sp-election-log-scroll">
          <table class="sp-election-log-table">
            <thead>
              <tr>
                <th v-for="column in columns" :key="column.key" scope="col" :class="column.class">{{ column.label }}</th>
              </tr>
            </thead>
            <tbody v-if="loading">
              <tr v-for="i in 5" :key="`skeleton-${i}`">
                <td v-for="column in columns" :key="column.key" :class="column.class">
                  <span class="sp-election-log-skeleton"></span>
                </td>
              </tr>
            </tbody>
            <tbody v-else-if="sections.length === 0">
              <tr>
                <!-- 空态要区分「真的没有切换」和「筛选太窄」—— 用同一句话会让人以为功能坏了。 -->
                <td :colspan="columns.length" class="sp-election-log-empty">
                  {{ hasActiveFilters ? '当前筛选条件下没有调度切换记录，试试放宽时间范围或换个方向。' : '最近还没有发生调度切换。' }}
                </td>
              </tr>
            </tbody>
            <tbody v-else>
              <template v-for="section in sections" :key="section.key">
                <tr class="sp-election-log-group-row">
                  <th scope="colgroup" :colspan="columns.length">
                    <strong class="sp-election-log-group-name">{{ section.groupName }}</strong>
                    <span class="sp-election-log-group-stat">
                      <em class="good">{{ section.enabled }} 开</em>
                      <em class="bad">{{ section.disabled }} 关</em>
                    </span>
                    <small class="sp-election-log-group-count">共 {{ section.total }} 条变更 · {{ section.batches.length }} 个批次</small>
                  </th>
                </tr>
                <!-- 组内再按批次分块。不同批次的变更不混在同一段里 ——
                     混在一起会把两次运行读成一次，也没法按批次对照前后变化。 -->
                <template v-for="batch in section.batches" :key="batch.key">
                  <tr class="sp-election-log-batch-row">
                    <td :colspan="columns.length">
                      <span class="sp-election-log-batch-id">批次 #{{ batch.runID }}</span>
                      <span class="sp-election-log-batch-time">{{ formatTime(batch.changedAt) }}</span>
                      <span class="sp-election-log-batch-status" :class="runStatusClass(batch.runStatus)">{{ runStatusText(batch.runStatus) }}</span>
                      <small class="sp-election-log-batch-count">本批次 {{ batch.rows.length }} 条变更</small>
                    </td>
                  </tr>
                  <tr v-for="log in batch.rows" :key="`${batch.key}-${rowKey(log)}`" class="sp-election-log-row">
                    <td v-for="column in columns" :key="column.key" :class="column.class">
                      <button
                        v-if="column.key === 'run'"
                        type="button"
                        class="sp-election-log-run"
                        :class="{ 'is-active': runFilter === log.run_id }"
                        :title="runFilter === log.run_id ? '已锁定该批次' : '只看这一批次的切换'"
                        @click="filterByRun(log.run_id)"
                      >
                        #{{ log.run_id }}
                      </button>
                      <span v-else-if="column.key === 'changed_at'" class="sp-election-log-time">{{ formatTime(log.changed_at) }}</span>
                      <template v-else-if="column.key === 'account'">
                        <strong class="sp-election-log-account">{{ log.account_name || `账号 ${log.account_id}` }}</strong>
                        <span v-if="log.platform" class="sp-election-log-platform" :class="platformTextClass(log.platform)">{{ log.platform }}</span>
                      </template>
                      <template v-else-if="column.key === 'direction'">
                        <span class="sp-election-log-direction" :class="log.direction === 'enabled' ? 'good' : 'bad'">
                          {{ directionText(log) }}
                        </span>
                        <!-- 演练产生的条目没有真的写库，必须标出来：
                             这份日志的取数条件就是「前后不一致」，不标的话建议会被读成已发生的切换。 -->
                        <span v-if="log.suggested" class="sp-election-log-suggested" title="演练模式：本条只是建议，调度开关没有被修改">建议</span>
                        <small class="sp-election-log-switch">{{ log.schedulable_before ? '开' : '关' }} → {{ log.schedulable_after ? '开' : '关' }}</small>
                      </template>
                      <span v-else-if="column.key === 'test_status'" class="sp-status" :class="log.test_status === 'success' ? 'good' : 'bad'">{{ testStatusText(log.test_status) }}</span>
                      <span v-else-if="column.key === 'healthy_count'">{{ log.healthy_count }}</span>
                      <span v-else-if="column.key === 'latency_ms'" class="sp-election-log-latency" :class="{ 'is-empty': !log.latency_ms }">{{ latencyText(log.latency_ms) }}</span>
                      <template v-else-if="column.key === 'reason'">
                        <small class="sp-election-log-reason">{{ log.reason || '—' }}</small>
                        <small v-if="log.error_message" class="sp-election-log-error">{{ log.error_message }}</small>
                      </template>
                    </td>
                  </tr>
                </template>
              </template>
            </tbody>
          </table>
        </div>
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

// 批次（一次运行）的整体状态，取值见后端 SupplierAutomationStatus*：running/success/partial/failed。
function runStatusText(status?: string) {
  switch (status) {
    case 'success': return '成功'
    case 'partial': return '部分成功'
    case 'failed': return '失败'
    case 'running': return '运行中'
    default: return status || '—'
  }
}

// 「部分成功」用黄而不是绿或红：它既不是全好也不是全坏，染成任一种都会误导。
// 刻意不复用页面级 `.sp-status`：它依赖 `.supplier-management-page` 上的 --sp-*，
// 而本弹窗被 Teleport 到 body，那些变量在这里未定义 ⇒ 会退化成裸文字（无边框/底色/颜色）。
function runStatusClass(status?: string) {
  if (status === 'success') return 'is-good'
  if (status === 'partial') return 'is-warn'
  if (status === 'failed') return 'is-bad'
  return ''
}

function latencyText(latencyMs?: number) {
  if (!latencyMs || latencyMs <= 0) return '—'
  if (latencyMs < 1000) return `${latencyMs}ms`
  return `${(latencyMs / 1000).toFixed(1)}s`
}

/** 组内的一块批次：同一次运行里的变更，以及该批次自己的开 / 关条数。 */
interface LogBatchBlock {
  key: string
  runID: number
  runStatus: string
  changedAt: string
  enabled: number
  disabled: number
  rows: SupplierGroupElectionChangeLog[]
}

/** 一个分组分节：该组下被拨动过的账号，组内再按批次切块。 */
interface LogSection {
  key: string
  groupName: string
  enabled: number
  disabled: number
  /** 该组跨批次的变更总条数。 */
  total: number
  batches: LogBatchBlock[]
}

// 分组名缺失时的兜底桶（分组已被删除，或历史数据没记名字）。
const NO_GROUP_KEY = '__no_group__'

// 按分组分节、组内再按批次分块。
// 分两层是因为一份日志里混着多次运行：只按分组归档的话，同一个组下面会把不同批次的行混在一起，
// 读起来像一次运行，也没法按批次对照「这一批动了谁、下一批又动了谁」。
// 一个账号同属多个分组时，它会在**每个**分节下各出现一次 —— 这不是重复：
// 它的开关确实同时影响了那几个组（后端 group_ids 本身就是数组）。
// 用分组名当键是安全的：groups 上有 name 的部分唯一索引（WHERE deleted_at IS NULL），
// 后端取名字时也带了同样的过滤，所以同一批日志里不会出现两个同名的组。
const sections = computed<LogSection[]>(() => {
  const groupMap = new Map<string, LogSection>()
  // 批次块按「分组 + 批次」建键：同一个批次出现在两个分组下时是两块，不能共用一个对象。
  const batchMap = new Map<string, LogBatchBlock>()
  const push = (groupKey: string, groupName: string, log: SupplierGroupElectionChangeLog) => {
    let section = groupMap.get(groupKey)
    if (!section) {
      section = { key: groupKey, groupName, enabled: 0, disabled: 0, total: 0, batches: [] }
      groupMap.set(groupKey, section)
    }
    const batchKey = `${groupKey}::${log.run_id}`
    let block = batchMap.get(batchKey)
    if (!block) {
      block = {
        key: batchKey,
        runID: log.run_id,
        runStatus: log.run_status,
        changedAt: log.changed_at,
        enabled: 0,
        disabled: 0,
        rows: [],
      }
      batchMap.set(batchKey, block)
      section.batches.push(block)
    }
    block.rows.push(log)
    section.total += 1
    if (log.schedulable_after) {
      block.enabled += 1
      section.enabled += 1
    } else {
      block.disabled += 1
      section.disabled += 1
    }
  }
  for (const log of items.value) {
    const names = (log.group_names || []).filter(Boolean)
    if (names.length === 0) {
      // 不丢掉这条变更：它确实拨动过，只是名字查不到了。
      push(NO_GROUP_KEY, '未记录分组', log)
      continue
    }
    for (const name of names) push(name, name, log)
  }
  // 变更条数多的组排前面（动静大的先看），条数相同按名字排，避免刷新后顺序漂移。
  const list = Array.from(groupMap.values()).sort((a, b) =>
    b.total - a.total || a.groupName.localeCompare(b.groupName, 'zh-Hans-CN')
  )
  // 组内批次新的在前；时间解析失败时用批次号兜底，保证顺序稳定不跳。
  for (const section of list) {
    section.batches.sort((a, b) => {
      const diff = Date.parse(b.changedAt) - Date.parse(a.changedAt)
      return Number.isFinite(diff) && diff !== 0 ? diff : b.runID - a.runID
    })
  }
  return list
})

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

/* 暗色覆盖一律写普通的 `.dark xxx`，不要写 `:global(.dark) xxx`。
   实测本仓 Vue 版本里 scoped 样式中的 `:global(.dark) .foo` 会被编译成裸 `.dark {}`
   —— 后代选择器和 data-v 属性一起丢掉，声明落到 <html> 上，
   而弹窗根元素自己声明了浅色变量（元素自身声明压过继承），结果就是暗色永远不生效且不报错。
   `.dark .foo` 才会编译成 `.dark .foo[data-v-xxx]`。 */
.dark .sp-election-log-dialog {
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

/* ── 分节表格 ─────────────────────────────────────────────────────────
   这里没有用 DataTable：它每行固定渲染 columns.length 个 <td>，不支持 colspan，
   而「分组标题行」必须整行贯通。页面层自建表格是既有组件能力之外的正解。 */
.sp-election-log-scroll {
  max-height: min(60vh, 680px);
  overflow: auto;
}

.sp-election-log-table {
  width: 100%;
  min-width: max-content;
  border-collapse: collapse;
}

.sp-election-log-table thead th {
  position: sticky;
  top: 0;
  z-index: 2;
  padding: 9px 12px;
  border-bottom: 1px solid var(--sp-election-log-line);
  background: var(--sp-election-log-soft);
  color: var(--sp-election-log-muted);
  font-size: 11px;
  font-weight: 700;
  letter-spacing: 0.03em;
  text-align: left;
  white-space: nowrap;
}

.sp-election-log-row > td {
  padding: 10px 12px;
  border-bottom: 1px solid color-mix(in srgb, var(--sp-election-log-line) 55%, transparent);
  font-size: 13px;
  vertical-align: top;
}

.sp-election-log-row:hover > td {
  background: color-mix(in srgb, var(--sp-election-log-accent) 4%, var(--sp-election-log-panel));
}

/* 分组标题行：左侧 accent 竖条 + 浅橙底，把「下面几行属于同一组」变成一眼可辨的块。 */
.sp-election-log-group-row > th {
  padding: 8px 12px 8px 0;
  border-bottom: 1px solid color-mix(in srgb, var(--sp-election-log-accent) 24%, var(--sp-election-log-line));
  background: color-mix(in srgb, var(--sp-election-log-accent) 9%, var(--sp-election-log-panel));
  text-align: left;
}

.sp-election-log-group-row > th::before {
  content: '';
  display: inline-block;
  width: 3px;
  height: 13px;
  margin: 0 8px 0 12px;
  border-radius: 2px;
  background: var(--sp-election-log-accent);
  vertical-align: -2px;
}

.sp-election-log-group-name {
  color: var(--sp-election-log-accent);
  font-size: 13px;
  font-weight: 750;
}

.sp-election-log-group-stat {
  margin-left: 10px;
  font-variant-numeric: tabular-nums;
}

.sp-election-log-group-stat em {
  margin-right: 8px;
  font-style: normal;
  font-weight: 700;
}

.sp-election-log-group-stat em.good { color: #16a34a; }
.sp-election-log-group-stat em.bad { color: #dc2626; }

.sp-election-log-group-count {
  color: var(--sp-election-log-muted);
  font-size: 11px;
}

/* 批次小标题行：比分组标题轻一档（无 accent 竖条、底色更淡、左缩进更多），
   让「分组 > 批次 > 行」三级在视觉上分得开。 */
.sp-election-log-batch-row > td {
  padding: 6px 12px 6px 30px;
  border-bottom: 1px solid color-mix(in srgb, var(--sp-election-log-line) 75%, transparent);
  background: color-mix(in srgb, var(--sp-election-log-accent) 4%, var(--sp-election-log-panel));
  text-align: left;
}

.sp-election-log-batch-id {
  font-size: 12px;
  font-weight: 700;
  font-variant-numeric: tabular-nums;
}

.sp-election-log-batch-time {
  margin-left: 10px;
  color: var(--sp-election-log-muted);
  font-size: 11px;
  font-variant-numeric: tabular-nums;
}

.sp-election-log-batch-count {
  margin-left: 10px;
  color: var(--sp-election-log-muted);
  font-size: 11px;
}

/* 批次状态徽标。刻意不复用页面级 `.sp-status`：它依赖 `.supplier-management-page` 上的 --sp-*，
   而本弹窗被 Teleport 到 body，那些变量在这里未定义 ⇒ 边框/底色/颜色全部失效、只剩裸文字。
   这里用弹窗自己声明的 --sp-election-log-*，配色与数据行的方向列对齐（绿 / 黄 / 红）。 */
.sp-election-log-batch-status {
  display: inline-flex;
  align-items: center;
  margin-left: 10px;
  padding: 1px 7px;
  border: 1px solid var(--sp-election-log-line);
  border-radius: 9999px;
  color: var(--sp-election-log-muted);
  font-size: 11px;
}

.sp-election-log-batch-status.is-good {
  border-color: color-mix(in srgb, #16a34a 35%, var(--sp-election-log-line));
  background: color-mix(in srgb, #16a34a 8%, var(--sp-election-log-panel));
  color: #16a34a;
}

.sp-election-log-batch-status.is-warn {
  border-color: color-mix(in srgb, #d97706 35%, var(--sp-election-log-line));
  background: color-mix(in srgb, #d97706 8%, var(--sp-election-log-panel));
  color: #d97706;
}

.sp-election-log-batch-status.is-bad {
  border-color: color-mix(in srgb, #dc2626 35%, var(--sp-election-log-line));
  background: color-mix(in srgb, #dc2626 8%, var(--sp-election-log-panel));
  color: #dc2626;
}

.sp-election-log-skeleton {
  display: block;
  height: 14px;
  width: 72%;
  border-radius: 4px;
  background: var(--sp-election-log-soft);
}

.sp-election-log-empty {
  padding: 48px 12px;
  color: var(--sp-election-log-muted);
  text-align: center;
}

.dark .sp-election-log-group-row > th {
  background: color-mix(in srgb, var(--sp-election-log-accent) 15%, var(--sp-election-log-panel));
}

.dark .sp-election-log-batch-row > td {
  background: color-mix(in srgb, var(--sp-election-log-accent) 8%, var(--sp-election-log-panel));
}

/* 暗色下亮一档，否则深底上的绿 / 黄 / 红对比不足。 */
.dark .sp-election-log-batch-status.is-good { color: #4ade80; }
.dark .sp-election-log-batch-status.is-warn { color: #fbbf24; }
.dark .sp-election-log-batch-status.is-bad { color: #f87171; }

.dark .sp-election-log-group-stat em.good { color: #4ade80; }
.dark .sp-election-log-group-stat em.bad { color: #f87171; }

.sp-election-log-direction {
  font-weight: 750;
}

.sp-election-log-direction.good {
  color: #16a34a;
}

.sp-election-log-direction.bad {
  color: #dc2626;
}

.dark .sp-election-log-direction.good {
  color: #4ade80;
}

.dark .sp-election-log-direction.bad {
  color: #f87171;
}

/* 「建议」徽标用虚线边框而不是实心块：它与方向色（绿=开启/红=关闭）是两套维度，
   实心块会被当成第三种方向；虚线则明确表达「这一条还没落地」。 */
.sp-election-log-suggested {
  display: inline-block;
  margin-left: 6px;
  padding: 0 5px;
  border: 1px dashed var(--sp-election-log-accent, #ea580c);
  border-radius: 4px;
  color: var(--sp-election-log-accent, #ea580c);
  font-size: 11px;
  font-weight: 700;
  line-height: 16px;
  vertical-align: 1px;
}

.dark .sp-election-log-suggested {
  border-color: #fb923c;
  color: #fb923c;
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

.dark .sp-election-log-error {
  color: #f87171;
}
</style>

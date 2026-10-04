<template>
  <BaseDialog :show="show" :title="dialogTitle" width="full" @close="emit('close')">
    <div class="sp-election-log-dialog">
      <p class="sp-election-log-hint">
        <template v-if="filters.includeSkipped">
          已含「本该动却没动」的记录：在任者健康锁定、分组只剩它保留、连续失败未达阈值、写库失败，
          各自标注原因并显示「本次跳过」。仍不含未测试、未参与择优的账号（它们不产生调度意图）。
        </template>
        <template v-else>
          只显示调度开关真的被拨动的记录：择优调度把某个账号从「开」改成「关」，或重新选回「开」。
          未变更、未测试、写库失败的记录不在这里，勾选「含未切换」可把带原因的跳过记录一并带出。
        </template>
        按分组归类展示，每组标出该组的开 / 关条数；组内再按批次分块，不同批次不混在一起。
        顶部「最近批次」标签可多选，选中的几批会各自成块并排，方便对照「这批动了谁、那批又动了谁」。
        一个账号同属多个分组时，会在它所属的每个分组下各出现一次。
        分页按记录切分，所以同一个分组或同一个批次可能被分到相邻两页，上方条数也只统计当前这一页。
      </p>

      <div class="sp-election-log-filters">
        <!-- chip 只在「进来时被锁定到某个分组」时显示，见 showGroupChip。 -->
        <span v-if="showGroupChip" class="sp-election-log-group-chip">
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
        <span v-if="runFilters.length > 0" class="sp-election-log-group-chip">
          仅看批次：{{ runFilters.map((id) => runLabel(id)).join('、') }}
          <button type="button" class="sp-election-log-chip-clear" title="查看全部批次" @click="clearRun">
            查看全部
          </button>
        </span>
        <div class="sp-election-log-filter">
          <span class="sr-only">开关方向</span>
          <Select v-model="filters.direction" :options="directionOptions" :searchable="false" @update:model-value="applyFilters" />
        </div>
        <div class="sp-election-log-filter">
          <span class="sr-only">平台</span>
          <Select v-model="filters.platform" :options="platformOptions" @update:model-value="applyFilters" />
        </div>
        <!-- 分组筛选跟在平台后面：一个分组必属于某个平台（Group.platform），
             先平台后分组才符合「先圈大范围、再挑里面的组」的选取顺序。 -->
        <div class="sp-election-log-filter sp-election-log-filter-group">
          <span class="sr-only">分组</span>
          <Select v-model="groupFilterValue" :options="groupSelectOptions" placeholder="全部分组" @update:model-value="applyFilters" />
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
        <!-- 含未切换：默认只看开关真被拨动的记录，打开后把「本该动却没动」的也带出来
             （在任者健康锁定 / 分组只剩它保留 / 连续失败未达阈值 / 写库失败），各自标注原因。 -->
        <label class="sp-election-log-toggle" :class="{ 'is-on': filters.includeSkipped }">
          <Toggle :model-value="filters.includeSkipped" @update:model-value="onToggleIncludeSkipped" />
          <span>含未切换（带原因）</span>
        </label>
        <button class="sp-button small primary" type="button" :disabled="loading" @click="applyFilters">
          {{ loading ? '查询中' : '查询' }}
        </button>
        <button class="sp-button small ghost" type="button" :disabled="loading || !hasActiveFilters" @click="resetFilters">
          重置
        </button>
      </div>

      <!-- 最近批次快捷标签：点一下只看那一批，再点一下取消（与表格里的批次号按钮同一套行为）。
           这批数据由后端给（最近 5 个真有变更的批次），**不随上面的筛选变化** ——
           否则点一个标签其余标签就消失了，没法来回切换着对比。 -->
      <div v-if="recentRuns.length > 0" class="sp-election-log-recent">
        <span class="sp-election-log-recent-label">最近批次</span>
        <button
          v-for="run in recentRuns"
          :key="run.run_id"
          type="button"
          class="sp-election-log-recent-run"
          :class="{ 'is-active': runFilters.includes(run.run_id) }"
          :title="runFilters.includes(run.run_id) ? '已加入对照，再点移出' : `加入对照：批次 ${runSerial(run.changed_at)}`"
          @click="filterByRun(run.run_id)"
        >
          {{ runSerial(run.changed_at) }}
        </button>
      </div>

      <div class="sp-election-log-table-region">
        <div class="sp-election-log-scroll">
          <table class="sp-election-log-table">
            <thead>
              <tr>
                <th
                  v-for="column in columns"
                  :key="column.key"
                  scope="col"
                  :class="column.class"
                  :aria-sort="column.sortable ? ariaSortFor(column.key) : undefined"
                >
                  <button
                    v-if="column.sortable"
                    type="button"
                    class="sp-election-log-sort"
                    :title="sortTitle"
                    @click="toggleSort(column.key)"
                  >
                    {{ column.label }}
                    <span class="sp-election-log-sort-icon" :class="`is-${sortOrder}`" aria-hidden="true">
                      <svg viewBox="0 0 10 6" width="8" height="6"><path d="M5 6 0 0h10L5 6Z" /></svg>
                    </span>
                  </button>
                  <template v-else>{{ column.label }}</template>
                </th>
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
                      <em v-if="section.skipped" class="skip">{{ section.skipped }} 跳过</em>
                    </span>
                    <small class="sp-election-log-group-count">共 {{ section.total }} 条 · {{ section.batches.length }} 个批次</small>
                  </th>
                </tr>
                <!-- 组内再按批次分块。不同批次的变更不混在同一段里 ——
                     混在一起会把两次运行读成一次，也没法按批次对照前后变化。 -->
                <template v-for="batch in section.batches" :key="batch.key">
                  <tr class="sp-election-log-batch-row">
                    <td :colspan="columns.length">
                      <span class="sp-election-log-batch-id">批次 {{ runSerial(batch.changedAt) }}</span>
                      <span class="sp-election-log-batch-time">{{ formatDateTime(batch.changedAt) }}</span>
                      <span class="sp-election-log-batch-status" :class="runStatusClass(batch.runStatus)">{{ runStatusText(batch.runStatus) }}</span>
                      <small class="sp-election-log-batch-count">本批次 {{ batch.rows.length }} 条</small>
                      <!-- 批次口径：取前 N 名、入选线、权重、封顶等组级固定参数，同批次同组每行都一样，只在标题写一次。 -->
                      <small v-if="batchCalibration(section, batch)" class="sp-election-log-batch-calibration">{{ batchCalibration(section, batch) }}</small>
                    </td>
                  </tr>
                  <tr v-for="log in batch.rows" :key="`${batch.key}-${rowKey(log)}`" class="sp-election-log-row">
                    <td v-for="column in columns" :key="column.key" :class="column.class">
                      <button
                        v-if="column.key === 'run'"
                        type="button"
                        class="sp-election-log-run"
                        :class="{ 'is-active': runFilters.includes(log.run_id) }"
                        :title="runFilters.includes(log.run_id) ? '已加入对照，再点移出' : '加入对照：这一批次的切换'"
                        @click="filterByRun(log.run_id)"
                      >
                        {{ runSerial(log.changed_at) }}
                      </button>
                      <span v-else-if="column.key === 'changed_at'" class="sp-election-log-time">{{ formatDateTime(log.changed_at) }}</span>
                      <template v-else-if="column.key === 'account'">
                        <strong class="sp-election-log-account">{{ log.account_name || `账号 ${log.account_id}` }}</strong>
                        <span v-if="log.platform" class="sp-election-log-platform" :class="platformTextClass(log.platform)">{{ log.platform }}</span>
                      </template>
                      <template v-else-if="column.key === 'direction'">
                        <span class="sp-election-log-direction" :class="directionClass(log)">
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
                        <!-- 结论按本组那条依据分类上色（reasonView），扫一列色块就能分堆，不必逐句读散文。
                             用本组结论而不是账号级 log.reason：后者是 union 语义，
                             照抄到别的分组分节下会把「本组落选」写成「分组内最优」。 -->
                        <template v-if="reasonView(section, log).badge">
                          <div class="sp-election-log-reason-head">
                            <span class="sp-election-log-reason-badge" :class="reasonBadgeClass(reasonView(section, log).badge)">{{ reasonView(section, log).badge?.label }}</span>
                            <!-- 名次/综合分：对比时最常看的两个数，从折叠的「依据」里提到行内。 -->
                            <span v-if="reasonView(section, log).rank" class="sp-election-log-reason-rank">{{ reasonView(section, log).rank }}</span>
                          </div>
                          <small v-if="reasonView(section, log).note" class="sp-election-log-reason-note">{{ reasonView(section, log).note }}</small>
                          <!-- 评分贡献条：次数/用时/优先三段按加权贡献上色，上下对齐扫一列就看出谁靠哪项赢的。
                               段宽 = 加权贡献 / 综合分；段内标「次0.900」= 项名 + 该项算出来的分项得分，
                               既是图例也是读数，不用再悬停看 title。
                               0 贡献的段（如本组不计优先级）不画，只对 scored 行显示。 -->
                          <div v-if="scoreBreakdown(section, log)" class="sp-election-log-score-breakdown">
                            <div v-if="scoreBreakdown(section, log)!.segments.length > 0" class="sp-election-log-score-bar">
                              <span
                                v-for="segment in scoreBreakdown(section, log)!.segments"
                                :key="segment.label"
                                class="sp-election-log-score-segment"
                                :class="`is-${segment.kind}`"
                                :style="{ width: `${segment.widthPercent}%` }"
                                :title="`${segment.label} ${segment.value}`"
                              >
                                <small class="sp-election-log-score-segment-label">{{ segment.shortLabel }}{{ segment.value }}</small>
                              </span>
                            </div>
                            <!-- 综合分的计算式：每项「原始值 → 分项得分 × 权重」再求和。
                                 段条只标分项得分且窄了会被裁，原始值又散在别的列 ⇒
                                 给一行不依赖布局的完整式，综合分可手工复核。 -->
                            <small v-if="scoreBreakdown(section, log)!.formula" class="sp-election-log-score-formula">
                              {{ scoreBreakdown(section, log)!.formula }}
                            </small>
                            <!-- 单行旗标：只跟这一行相关的标记（封顶命中、用时中性）才显示，不命中就不出现。 -->
                            <small v-if="scoreBreakdown(section, log)!.flags.length > 0" class="sp-election-log-score-flags">
                              {{ scoreBreakdown(section, log)!.flags.join(' · ') }}
                            </small>
                          </div>
                        </template>
                        <!-- 没归到分类的行（旧记录无依据、账号级跳过原因）降级回原文。 -->
                        <small v-else class="sp-election-log-reason">{{ reasonView(section, log).fallback }}</small>
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
import { getAllIncludingInactive } from '@/api/admin/groups'
import {
  listGroupElectionChangeLogs,
  type SupplierGroupElectionChangeLog,
  type SupplierGroupElectionDecisionDetail,
  type SupplierGroupElectionRecentRun,
} from '@/api/admin/supplierAutomation'
import BaseDialog from '@/components/common/BaseDialog.vue'
import Input from '@/components/common/Input.vue'
import Pagination from '@/components/common/Pagination.vue'
import Select, { type SelectOption } from '@/components/common/Select.vue'
import Toggle from '@/components/common/Toggle.vue'
import type { Column } from '@/components/common/types'
import { useAppStore } from '@/stores/app'
import { formatDateTime } from '@/utils/format'
import { platformTextClass } from '@/utils/platformColors'
import { CORE_PLATFORM_OPTIONS } from '@/utils/platformOptions'

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
// platform 是 string：平台集合可配置（还能加自定义平台），写死联合类型会把新平台挡在类型层。
const filters = ref<{
  direction: 'all' | 'enabled' | 'disabled'
  platform: string
  search: string
  startedFrom: string
  startedTo: string
  includeSkipped: boolean
}>({
  direction: 'all',
  platform: '',
  search: '',
  startedFrom: '',
  startedTo: '',
  includeSkipped: false,
})
// 从某个分组/账号进来后又点了「查看全部」，此时以组件内部状态为准。
const clearedGroup = ref(false)
const clearedAccount = ref(false)
// 用户在「分组」下拉里主动选过的值。pickedGroup 标记「下拉被用过」，
// pickedGroupID 为 null 表示选了「全部分组」。
// 一旦用过下拉就由它完全接管分组筛选 —— 否则会出现「下拉显示 B、chip 显示 A」的自相矛盾。
const pickedGroup = ref(false)
const pickedGroupID = ref<number | null>(null)
// 任务批次筛选纯内部：从顶部标签或行内批次号点进来，再点「查看全部」清掉。
// 收成数组而不是单值 —— 多选后组内按批次分块，几批会各占一块并排，正好用来对照。
const runFilters = ref<number[]>([])
// 顶部快捷批次标签：由后端给出（最近 5 个真有变更的批次，按变更时间新到旧）。
// 带 changed_at 才能显示时间流水号；它不随当前筛选变化 —— 否则点一个标签，其余标签就没了，
// 没法来回切换着看。
const recentRuns = ref<SupplierGroupElectionRecentRun[]>([])
// 分组目录，给筛选下拉用（加载逻辑见 loadGroupOptions）。
const groupOptions = ref<SelectOption[]>([])
let groupOptionsLoaded = false

// 分组筛选有三个来源，优先级从高到低：
//   ① 用户在筛选下拉里选的值（pickedGroup 为真时）；
//   ② 「查看全部」清掉的锁定（clearedGroup）；
//   ③ 进来时 props 带的分组锁定。
// ① 必须最高：用户刚在下拉里改完，任何回填都会把这次操作静默抹掉。
const activeGroupID = computed(() => {
  if (pickedGroup.value) return pickedGroupID.value
  return clearedGroup.value ? null : props.groupId ?? null
})
const activeGroupLabel = computed(() => {
  if (pickedGroup.value) {
    if (pickedGroupID.value === null) return ''
    // 下拉里选出来的分组用目录里的名字；目录没拉到就留空，由调用方回退成「分组 N」。
    return groupOptions.value.find((option) => Number(option.value) === pickedGroupID.value)?.label ?? ''
  }
  return clearedGroup.value ? '' : props.groupLabel
})
const activeAccountID = computed(() => (clearedAccount.value ? null : props.accountId ?? null))
const activeAccountLabel = computed(() => (clearedAccount.value ? '' : props.accountLabel))
const dialogTitle = computed(() => {
  if (activeGroupID.value) return `调度切换日志 · ${activeGroupLabel.value || `分组 ${activeGroupID.value}`}`
  if (activeAccountID.value) return `调度切换日志 · ${activeAccountLabel.value || `账号 ${activeAccountID.value}`}`
  return '分组调度切换日志'
})
const hasActiveFilters = computed(() => (
  filters.value.direction !== 'all'
  || filters.value.platform !== ''
  || filters.value.search.trim() !== ''
  || filters.value.startedFrom !== ''
  || filters.value.startedTo !== ''
  || filters.value.includeSkipped
  || activeGroupID.value !== null
  || activeAccountID.value !== null
  || runFilters.value.length > 0
))

const directionOptions: SelectOption[] = [
  { value: 'all', label: '全部方向' },
  { value: 'disabled', label: '只看关闭' },
  { value: 'enabled', label: '只看开启' },
]

// 平台选项复用全局目录，不在这里另写一份：那份目录是「有哪些平台」的唯一前端来源，
// 抄一份出来迟早会跟新增平台 / 自定义平台脱节。
const platformOptions = computed<SelectOption[]>(() => [
  { value: '', label: '全部平台' },
  ...CORE_PLATFORM_OPTIONS.map((option) => ({ value: option.value, label: option.label })),
])

// 分组目录。用 include_inactive 版本 —— 日志里的分组可能已经被停用，
// 用只返回启用分组的接口会让那些分组在下拉里查不到，看着像日志丢了。
async function loadGroupOptions() {
  if (groupOptionsLoaded) return
  try {
    const list = await getAllIncludingInactive()
    groupOptions.value = list.map((group) => ({ value: String(group.id), label: group.name }))
    groupOptionsLoaded = true
  } catch {
    // 分组目录拉不到不该连累日志本身：下拉退化成只有「全部分组」，日志照常可看。
    // 不置 groupOptionsLoaded，下次打开弹窗会再试一次。
  }
}

// 「全部分组」用空串，与平台下拉同一约定（空串 = 不传参）。
const groupSelectOptions = computed<SelectOption[]>(() => [
  { value: '', label: '全部分组' },
  ...groupOptions.value,
])

// 下拉的当前值。getter 反映 activeGroupID —— 从分组页点进来时下拉也要显示那个分组，
// 否则下拉写着「全部分组」而列表其实被锁在一个分组上，两处自相矛盾。
// setter 只记录选择、不自己查：查询统一交给模板的 @update:model-value，
// 与方向 / 平台两个下拉保持同一套行为。
const groupFilterValue = computed<string>({
  get: () => (activeGroupID.value === null ? '' : String(activeGroupID.value)),
  set: (value) => {
    pickedGroup.value = true
    const parsed = Number(value)
    pickedGroupID.value = value === '' || !Number.isFinite(parsed) ? null : parsed
  },
})

// chip 只在「进来时被锁定到某个分组」时出现：用户自己在下拉里选的分组
// 已经在下拉里看得见，再冒一个 chip 是重复信息。
const showGroupChip = computed(() => !pickedGroup.value && activeGroupID.value !== null)

const columns: Column[] = [
  { key: 'run', label: '任务批次', class: 'min-w-[100px]' },
  // sortable：只排「组内批次块」的先后，不改分组顺序 —— 分组是按变更条数排的（见 sections 注释），
  // 而分组分节是这张表的主结构，跨分组做时间排序会把归类打散。
  // 宽度按「2026/09/23 10:00:00」这种完整日期时间给足：只给到时间的宽度会静默折行、把整行撑高。
  { key: 'changed_at', label: '切换时间', class: 'min-w-[170px]', sortable: true },
  { key: 'account', label: '账号', class: 'min-w-[150px]' },
  { key: 'direction', label: '调度变更', class: 'min-w-[110px]' },
  { key: 'test_status', label: '测试状态', class: 'min-w-[90px]' },
  { key: 'healthy_count', label: '连续成功', class: 'min-w-[80px]' },
  { key: 'latency_ms', label: '测试用时', class: 'min-w-[90px]' },
  { key: 'reason', label: '原因', class: 'min-w-[200px]' },
]

// 「切换时间」列当前的排序方向，默认倒序（最新在前）—— 与后端 ORDER BY changed_at DESC 同向，
// 所以「默认」和升级前的观感完全一致，排序控件只是把这件事变得可调。
// 作用范围是**每组内部的批次块**：表格按分组分节，组的顺序按变更条数排，
// 跨分组做时间排序会把归类打散，所以排序只重排组内批次。
const sortOrder = ref<'asc' | 'desc'>('desc')

// 只有「切换时间」一列可排序；其余列点不动。
function toggleSort(key: string) {
  if (key !== 'changed_at') return
  sortOrder.value = sortOrder.value === 'desc' ? 'asc' : 'desc'
}

// aria-sort 属于 columnheader 角色，挂到 <th> 上而不是按钮上。
function ariaSortFor(key: string): 'ascending' | 'descending' | 'none' {
  if (key !== 'changed_at') return 'none'
  return sortOrder.value === 'asc' ? 'ascending' : 'descending'
}

// 按钮的 title 说清「点下去会变成什么」，而不是只描述当前状态。
const sortTitle = computed(() => (
  sortOrder.value === 'desc'
    ? '每组内按切换时间倒序（最新在前），点击改为正序'
    : '每组内按切换时间正序（最早在前），点击改为倒序'
))

// 同一次运行里可能有多个账号同时变更，run_id 会重复 —— 键必须用 run_id + account_id。
function rowKey(log: SupplierGroupElectionChangeLog) {
  return `${log.run_id}-${log.account_id}`
}

function directionText(log: SupplierGroupElectionChangeLog) {
  // 含未切换视图里会出现 before==after 的行：它「本该动却没动」，不是开也不是关。
  if (log.schedulable_before === log.schedulable_after) return '本次跳过'
  if (log.direction === 'enabled') return '开启调度'
  if (log.direction === 'disabled') return '关闭调度'
  return log.schedulable_after ? '开启调度' : '关闭调度'
}

// 方向色：开=绿、关=红，跳过=中性灰（不能染成开/关，那会把「没动」读成一次切换）。
function directionClass(log: SupplierGroupElectionChangeLog) {
  if (log.schedulable_before === log.schedulable_after) return 'skip'
  return log.direction === 'enabled' ? 'good' : 'bad'
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

// 时间流水号：把批次时间压成 20260923-100000。
// 用它代替裸的 run_id 显示 —— run_id 是自增主键，多 worker 并发时会交错，
// 光看号码既不知道是哪一批、也读不出先后；时间流水号一眼就能看出批次时间。
// 用本地时间：管理员看到的其它时间列也是本地时间，两者必须能对上。
function runSerial(changedAt: string): string {
  const date = new Date(changedAt)
  if (Number.isNaN(date.getTime())) return changedAt
  const pad = (value: number) => String(value).padStart(2, '0')
  return `${date.getFullYear()}${pad(date.getMonth() + 1)}${pad(date.getDate())}`
    + `-${pad(date.getHours())}${pad(date.getMinutes())}${pad(date.getSeconds())}`
}

// run_id → 变更时间。表格行与批次块自己带着时间，但「仅看批次」chip 只有批次号，
// 得回头查它对应的时间。数据来源就是当前页 items：选中批次后 run_ids 会带上它，
// 后端必然返回该批次的记录。
const runChangedAt = computed(() => {
  const map = new Map<number, string>()
  for (const log of items.value) map.set(log.run_id, log.changed_at)
  return map
})

// 批次在 UI 上的统一称呼。查不到时间时（该批次被其它筛选条件排空）退回裸批次号，
// 至少不丢信息，也好过显示一个空串。
function runLabel(runID: number): string {
  const changedAt = runChangedAt.value.get(runID)
  return changedAt ? runSerial(changedAt) : `#${runID}`
}

/** 组内的一块批次：同一次运行里的变更，以及该批次自己的开 / 关 / 跳过条数。 */
interface LogBatchBlock {
  key: string
  runID: number
  runStatus: string
  changedAt: string
  enabled: number
  disabled: number
  // skipped：含未切换视图下「本该动却没动」的条数（before === after）。
  skipped: number
  rows: SupplierGroupElectionChangeLog[]
}

/** 一个分组分节：该组下被拨动过的账号，组内再按批次切块。 */
interface LogSection {
  key: string
  groupName: string
  enabled: number
  disabled: number
  skipped: number
  /** 该组跨批次的记录总条数（含未切换视图下也含跳过条目）。 */
  total: number
  batches: LogBatchBlock[]
}

// 分组名缺失时的兜底桶（分组已被删除，或历史数据没记名字）。
const NO_GROUP_KEY = '__no_group__'

// 批次口径：取前 N 名、入选线、权重、封顶等组级固定参数，同批次同组每行都一样，只在标题写一次。
// 只对有 scored 行的批次显示（没人参与评分就没有口径可言）。
function batchCalibration(section: LogSection, batch: LogBatchBlock): string {
  // 从本批次任一 scored 行拿口径（同批次同组口径一致）。
  const anyScored = batch.rows.map(log => decisionFor(section, log)).find(d => d?.scored)
  if (!anyScored) return ''
  const parts: string[] = []
  if (anyScored.top_n) {
    parts.push(`取前 ${anyScored.top_n} 名`)
  }
  if (typeof anyScored.winner_cutoff === 'number') {
    parts.push(`入选线 ${anyScored.winner_cutoff.toFixed(3)}`)
  }
  const weights: string[] = []
  if (typeof anyScored.count_weight === 'number') weights.push(`次${anyScored.count_weight.toFixed(1)}`)
  if (typeof anyScored.latency_weight === 'number') weights.push(`时${anyScored.latency_weight.toFixed(1)}`)
  // 级只在「本组真的计优先级」时才列进权重：priority_weight 恒为配置值（本组禁用时也是 0.5），
  // 无条件列出来会把「不计」写成「计了 0.5」。旧记录没有 priority_enabled（undefined），保持原样。
  if (anyScored.priority_enabled !== false && typeof anyScored.priority_weight === 'number') {
    weights.push(`级${anyScored.priority_weight.toFixed(1)}`)
  }
  if (weights.length > 0) parts.push(`权重 ${weights.join('/')}`)
  if (anyScored.count_score_cap) parts.push(`次数封顶 ${anyScored.count_score_cap}`)
  // 不计优先级是一句「有或没有」的口径，放在权重后面单列，比在权重里塞个 0 更好读。
  if (anyScored.priority_enabled === false) parts.push('本组不计优先级')
  return parts.join(' · ')
}

// 评分贡献条：次数/用时/优先三段按加权贡献上色，上下对齐扫一列就看出谁靠哪项赢的。
// 只对 scored 行显示（没参与评分就没有构成可言）。
interface ScoreSegment {
  kind: 'count' | 'latency' | 'priority'
  label: string
  shortLabel: string
  value: string
  widthPercent: number
}
interface ScoreBreakdown {
  segments: ScoreSegment[]
  flags: string[]
  /** 综合分的计算式：每项「原始值 → 分项得分 × 权重」再求和。没有可写的项时为空串。 */
  formula: string
}
function scoreBreakdown(section: LogSection, log: SupplierGroupElectionChangeLog): ScoreBreakdown | null {
  const decision = decisionFor(section, log)
  if (!decision || !decision.scored) return null
  const segments: ScoreSegment[] = []
  const flags: string[] = []
  const terms: string[] = []
  const score = (value?: number) => (typeof value === 'number' ? value.toFixed(3) : '—')
  // 权重按原值写、只给整数补一位小数（1 → 1.0，0.5 → 0.5，1.25 → 1.25）：
  // 统一 toFixed(1) 会把 1.25 截成 1.3，式子当场算不平 —— 复核用的式子宁可长一点也要准。
  const weightText = (value: number) => {
    const text = String(value)
    return text.includes('.') ? text : `${text}.0`
  }

  // 加权贡献 = 分项得分 × 权重；段宽 = 加权贡献 / 综合分。
  const total = decision.score ?? 0
  // rawText 是该分项的**原始输入**（次数、参与评分的用时），只进计算式、不进段条 ——
  // 段条按贡献比例分宽，塞原始值只会被裁掉。
  const pushSegment = (
    kind: ScoreSegment['kind'],
    label: string,
    shortLabel: string,
    rawText: string,
    scoreValue?: number,
    weight?: number,
  ) => {
    if (typeof scoreValue !== 'number' || typeof weight !== 'number') return
    const widthPercent = total > 0 ? ((scoreValue * weight) / total) * 100 : 0
    // 段宽为 0 的项在这一行上没有任何贡献：画出来只剩一个 1px 空条加一个被裁掉的标签，
    // 反而会被读成「这项有值、只是很小」。0 贡献的段一律不画。
    if (widthPercent <= 0) return
    segments.push({ kind, label, shortLabel, value: score(scoreValue), widthPercent })
    // 计算式与段条**同口径**（0 贡献的项两边都不出现），否则式子的项与条的颜色对不上。
    terms.push(`${shortLabel}${rawText}=${score(scoreValue)}×${weightText(weight)}`)
  }

  // 次数原始值写「12/10」：分母是封顶值，超过即封顶命中（批次口径已列封顶值，这里再给一份自解释）。
  const countCap = decision.count_score_cap
  const countRaw = typeof countCap === 'number' && countCap > 0 ? `${log.healthy_count}/${countCap}` : `${log.healthy_count}`
  // 用时原始值取 effective_latency_ms —— 它**不是**表格里的「测试用时」：含成功率惩罚，
  // 且在任者还要按迟滞死区折算，真正进公式的是这个数。
  // 取中性值时直接标「中性」，免得管理员拿一个没参与计算的值去反推分项得分。
  const latencyRaw = decision.latency_fallback
    ? '中性'
    : typeof decision.effective_latency_ms === 'number' && decision.effective_latency_ms > 0
      ? `${decision.effective_latency_ms}ms`
      : ''

  pushSegment('count', '次数', '次', countRaw, decision.count_score, decision.count_weight)
  pushSegment('latency', '用时', '时', latencyRaw, decision.latency_score, decision.latency_weight)
  // 本组不计优先级时这一项必然 0 贡献（后端已把 priority_score 置 0），直接不参与画条；
  // 旧运行记录没有 priority_enabled，由上面「0 贡献不画」兜住。
  // 优先级没有原始值可写（后端只给归一化后的 priority_score），留空。
  if (decision.priority_enabled !== false) {
    pushSegment('priority', '优先级', '级', '', decision.priority_score, decision.priority_weight)
  }

  // 单行旗标：只跟这一行相关的标记才显示，不命中就不出现。
  if (decision.count_score_cap && (decision.count_score ?? 0) >= 1.0) {
    flags.push('次数封顶命中')
  }
  if (decision.latency_fallback) {
    flags.push('用时中性')
  }

  // 既没构成也没旗标就整块不渲染，别留一条空条。
  if (segments.length === 0 && flags.length === 0) return null
  // 计算式：段条只标分项得分、窄了还会被裁，原始值又散在别的列（且用时列显示的不是参与计算的那个数），
  // 光看条子推不出综合分。这里给一行不依赖布局的完整式，综合分可手工复核。
  const formula = terms.length > 0 ? `${terms.join(' + ')} = ${score(total)}` : ''
  return { segments, flags, formula }
}

// 取「本行在本分节这个分组下」的那条裁决依据。
// 依据是按分组存的（一个账号跨多个分组时各组结论可能不同），所以必须按当前分节挑一条，
// 不能把该行的所有依据堆在一起 —— 那会把 A 组的评分解释到 B 组的行上。
function decisionFor(section: LogSection, log: SupplierGroupElectionChangeLog) {
  const decisions = log.group_decisions
  if (!decisions || decisions.length === 0) return undefined
  if (section.key === NO_GROUP_KEY) return undefined
  const matched = decisions.find((decision) => decision.group_name === section.groupName)
  // 只有一个分组结论时不存在歧义（名字缺失也认），多个却没匹配上就不猜。
  return matched || (decisions.length === 1 ? decisions[0] : undefined)
}

// 原因列的分类结论：把散文收敛成一眼可辨的色标签。kind 决定颜色，label 是短标签。
type ReasonBadgeKind =
  | 'elected' | 'required' | 'keep-alive' | 'locked' | 'converged' | 'upstream' | 'not-elected' | 'test-failed' | 'no-election'
interface ReasonBadge {
  kind: ReasonBadgeKind
  label: string
}
interface ReasonView {
  badge: ReasonBadge | null
  // 标签说不完、但对比时又必须点破的补充（如「该账号在其它分组当选」）。无则为空串。
  note: string
  // 名次/综合分：对比时最常看的两个数，从折叠的「依据」里提到行内。无评分时为空串。
  rank: string
  // 没归到任何分类时（旧记录无依据、账号级跳过原因）降级回的原文。
  fallback: string
}

// 本行在**当前分组**下的分类结论 —— 原因列的色标签就按它上色。
//
// 为什么不直接按账号级 log.reason 分类：那是账号级结论，而账号的调度开关是单一字段（union 语义）——
// 它在 A 组当选、在 B 组落选时整体仍记「分组内最优」，照抄到 B 组的分节下就是错的。
// 这里只认「本组那条依据」，所以同一账号在不同分组分节下可以是不同的标签；
// 旧运行记录没有依据（decision 为空）时不归类，由调用方降级回 log.reason。
function classifyReason(
  log: SupplierGroupElectionChangeLog,
  decision?: SupplierGroupElectionDecisionDetail,
): { badge: ReasonBadge | null; note: string } {
  if (!decision) return { badge: null, note: '' }
  // 上游停用排在最前：这类账号的测试状态与健康计数是「上游停用那一刻」冻结下来的旧数据，
  // 日志上看起来一切正常 —— 它不参选、不被补选、也不是锁定在任者，落进后面任何一档都会给出误导结论
  // （实测会落到「未参与择优」，读起来像评分没算上它，实际是请求打过去必然失败）。
  if (decision.upstream_unavailable) {
    return {
      badge: { kind: 'upstream', label: '上游停用' },
      note: '该账号匹配的上游账号已不可用，因此关闭调度',
    }
  }
  // 收敛关闭必须排在锁定之前：被收敛掉的在任者也带 locked（本组本轮没做择优），
  // 但它是被「关闭」的，标成「健康锁定（保留）」会与同一行的「关闭调度」自相矛盾。
  if (decision.over_capacity) {
    // 叠加了必需模型这一层时必须说全：只写「超过上限」，用户解释不了
    // 「我手动开的那个账号为什么被换掉、留下的是另一个」——那正是这条日志要回答的问题。
    const note = decision.over_capacity_required_model
      ? '本组在任账号数超过上限，且必需模型已由保留的账号覆盖'
      : '本组在任账号数超过上限'
    return { badge: { kind: 'converged', label: '收敛关闭' }, note }
  }
  // 必需模型补选同理，只是方向相反：补选跑在收敛之后、锁定组也执行（必需模型是硬底线），
  // 所以被补选进来的账号同样带 locked —— 先判 locked 会把「刚被开启」标成「本轮未换人」，
  // 与同一行的「开启调度」自相矛盾。
  if (decision.required_models && decision.required_models.length > 0) {
    return { badge: { kind: 'required', label: '必需模型补选' }, note: decision.required_models.join('、') }
  }
  if (decision.locked) return { badge: { kind: 'locked', label: '健康锁定' }, note: '在任者正常，本轮未换人' }
  // 保底开启必须排在 elected 之前：它也带 elected（保底同样是一种入选），但综合分不是前 N ——
  // 多数情况它压根没参选（全失败/全未测），标成「择优入选」会让日志看起来像择优算错了。
  if (decision.keep_alive) return { badge: { kind: 'keep-alive', label: '分组保底' }, note: '本组无开启账号，兜底开启' }
  if (decision.elected) return { badge: { kind: 'elected', label: '择优入选' }, note: '' }
  // 「本该动却没动」行（before === after 且非锁定）：结论落在账号级 —— 无备选保留 /
  // 连续失败待观察 / 写库失败，这些是 union 语义下账号整体的裁决，交回 log.reason 才准，这里不硬归类。
  if (log.schedulable_before === log.schedulable_after) return { badge: null, note: '' }
  // 本组没选它、账号却开着（靠别的分组当选）：必须点破，否则「开启调度」与「未入选」并列会读成自相矛盾。
  if (log.direction === 'enabled') return { badge: { kind: 'not-elected', label: '本组未入选' }, note: '该账号在其它分组当选' }
  if (decision.test_failed) return { badge: { kind: 'test-failed', label: '测试失败' }, note: '' }
  if (decision.scored) return { badge: { kind: 'not-elected', label: '未入选' }, note: '' }
  return { badge: { kind: 'no-election', label: '未参与择优' }, note: '' }
}

// 名次/综合分：对比时最常看的两个数，原先埋在折叠的「依据」里、逐行点开才看得到，提到行内。
// 只有真正参与评分（scored）才给：锁定/收敛的行虽也带名次，但那是评过分后才被规则收走的。
function reasonRankText(decision?: SupplierGroupElectionDecisionDetail): string {
  if (!decision || !decision.scored) return ''
  const parts: string[] = []
  if (decision.rank && decision.rank_total) parts.push(`名次 ${decision.rank}/${decision.rank_total}`)
  if (typeof decision.score === 'number') parts.push(`综合分 ${decision.score.toFixed(3)}`)
  return parts.join(' · ')
}

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
      section = { key: groupKey, groupName, enabled: 0, disabled: 0, skipped: 0, total: 0, batches: [] }
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
        skipped: 0,
        rows: [],
      }
      batchMap.set(batchKey, block)
      section.batches.push(block)
    }
    block.rows.push(log)
    section.total += 1
    // before === after 的行是「本该动却没动」：单独计跳过，不能混进开 / 关 —— 那会把没发生的切换算成一次。
    if (log.schedulable_before === log.schedulable_after) {
      block.skipped += 1
      section.skipped += 1
    } else if (log.schedulable_after) {
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
  // 组内批次按「切换时间」排，方向由表头那一列控制（默认倒序 = 最新在前）。
  // 时间解析失败时用批次号兜底，保证顺序稳定不跳。
  for (const section of list) {
    section.batches.sort((a, b) => {
      const diff = Date.parse(a.changedAt) - Date.parse(b.changedAt)
      const ascending = Number.isFinite(diff) && diff !== 0 ? diff : a.runID - b.runID
      return sortOrder.value === 'asc' ? ascending : -ascending
    })
  }
  return list
})

// 每行的原因视图预先算好一份，模板里多处引用同一份，省得反复分类。
// 键要拼分组：同一个账号在它所属的每个分组分节下各出现一次，本组结论可以不同。
function buildReasonView(section: LogSection, log: SupplierGroupElectionChangeLog): ReasonView {
  const decision = decisionFor(section, log)
  const { badge, note } = classifyReason(log, decision)
  return { badge, note, rank: reasonRankText(decision), fallback: log.reason || '—' }
}

const reasonViewMap = computed(() => {
  const map = new Map<string, ReasonView>()
  for (const section of sections.value) {
    for (const batch of section.batches) {
      for (const log of batch.rows) map.set(`${section.key}::${rowKey(log)}`, buildReasonView(section, log))
    }
  }
  return map
})

function reasonView(section: LogSection, log: SupplierGroupElectionChangeLog): ReasonView {
  return reasonViewMap.value.get(`${section.key}::${rowKey(log)}`)
    ?? { badge: null, note: '', rank: '', fallback: log.reason || '—' }
}

function reasonBadgeClass(badge: ReasonBadge | null): string {
  return badge ? `is-${badge.kind}` : ''
}

async function load() {
  loading.value = true
  try {
    const result = await listGroupElectionChangeLogs({
      group_id: activeGroupID.value ?? undefined,
      account_id: activeAccountID.value ?? undefined,
      run_ids: runFilters.value.length > 0 ? runFilters.value : undefined,
      platform: filters.value.platform || undefined,
      search: filters.value.search.trim() || undefined,
      direction: filters.value.direction === 'all' ? undefined : filters.value.direction,
      started_from: filters.value.startedFrom || undefined,
      started_to: filters.value.startedTo || undefined,
      include_skipped: filters.value.includeSkipped || undefined,
      page: page.value,
      page_size: pageSize.value,
    })
    items.value = result.items
    total.value = result.total
    page.value = result.page
    pageSize.value = result.page_size
    // 后端每次都回全量最近批次（不随筛选变），直接整体替换即可。
    recentRuns.value = result.recent_runs || []
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

// 「含未切换」是个放宽范围的开关，切一下就该立即重查 —— 与 direction/platform 的 Select 一致。
function onToggleIncludeSkipped(value: boolean) {
  filters.value.includeSkipped = value
  return applyFilters()
}

function resetFilters() {
  // 重置清筛选条件，但把「锁定到某个分组 / 账号」保留 —— 用户是冲着它点开弹窗的，
  // 一起清掉会让人误以为看到的是全局日志。换对象要点「查看全部」。
  filters.value = { direction: 'all', platform: '', search: '', startedFrom: '', startedTo: '', includeSkipped: false }
  // 下拉里主动选的分组是筛选条件，重置一并清掉；清掉后 activeGroupID 回落到 props 锁定
  // （进来时锁定的对象仍保留 —— 用户是冲着它点开弹窗的）。
  pickedGroup.value = false
  pickedGroupID.value = null
  // 批次锁定是弹窗内点出来的临时筛选，不属于「进来时的对象」，重置一并清掉。
  runFilters.value = []
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

// 点批次号 = 把它加入 / 移出「对照集合」。多选是刻意的：组内本来就按批次分块，
// 选中几批就会各自成块并排显示，一眼能看出「这批动了谁、那批又动了谁」。
function filterByRun(runId: number) {
  runFilters.value = runFilters.value.includes(runId)
    ? runFilters.value.filter((id) => id !== runId)
    : [...runFilters.value, runId]
  page.value = 1
  return load()
}

function clearRun() {
  runFilters.value = []
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
  // 下拉里选的分组同样是本次查看的临时筛选，下次打开回到「进来时的锁定」。
  pickedGroup.value = false
  pickedGroupID.value = null
  runFilters.value = []
  page.value = 1
  // 每次打开都回到默认的「最新在前」：排序方向是本次查看的临时选择，
  // 留到下次会让「默认按时间倒序」这句话不成立。
  sortOrder.value = 'desc'
  void loadGroupOptions()
  void load()
})

watch(() => props.groupId, () => {
  if (!props.show) return
  clearedGroup.value = false
  pickedGroup.value = false
  pickedGroupID.value = null
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
  /* 贡献条段首标签用的正文字色。段底色是「色相 50% 混面板」，深浅不定，
     标签压在上面必须自己带色，不能靠继承（继承到的颜色随主题漂）。 */
  --sp-election-log-ink: #0f172a;
  display: grid;
  gap: 12px;
}

/* BaseDialog 的 full 预设最宽到 max-w-7xl(1280px)，这份日志列多（8 列 + 完整日期时间），
   按需求放宽到 95vw。只命中「装着本日志」的那一个 modal-content，不动其他用 full 的弹窗。 */
:global(.modal-content:has(.sp-election-log-dialog)) {
  max-width: 95vw;
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
  --sp-election-log-ink: #e2e8f0;
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

/* 分组下拉比另外两个下拉宽一档：分组名普遍比平台名长（「TKAPI2-Codex｜应急专用｜pro号池」这种），
   与平台下拉同宽会把名字截得看不出是哪个组。 */
.sp-election-log-filter-group {
  min-width: 180px;
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

/* 「含未切换」开关：一个开关一句标签并排，切一下即重查。
   打开后标签转为 accent 色，让「现在多看了跳过记录」这件事在筛选区一眼可辨。 */
.sp-election-log-toggle {
  display: inline-flex;
  align-items: center;
  gap: 8px;
  color: var(--sp-election-log-muted);
  font-size: 12px;
  font-weight: 700;
  cursor: pointer;
  user-select: none;
}

.sp-election-log-toggle.is-on {
  color: var(--sp-election-log-accent);
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

/* 最近批次快捷标签行。与上面的筛选区并排但不混在一起：它是「换着看」的入口，
   不是筛选条件本身，所以单独一行、更紧凑（标签比下拉框小一档）。
   换行时保持左对齐，标签多了也不会把行撑破。 */
.sp-election-log-recent {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 6px;
}

.sp-election-log-recent-label {
  color: var(--sp-election-log-muted);
  font-size: 12px;
}

/* 与表格里的批次号按钮同一套视觉（同色、同圆角、同 active 态），
   点起来才像是同一个动作 —— 只是位置从行内挪到了顶部。 */
.sp-election-log-recent-run {
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

.sp-election-log-recent-run:hover {
  background: color-mix(in srgb, var(--sp-election-log-accent) 12%, var(--sp-election-log-panel));
}

.sp-election-log-recent-run.is-active {
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

/* 「切换时间」表头的排序控件。排版（字号 / 字重 / 字距 / 颜色）全部继承表头，
   只多一个方向箭头 —— 表头其余列是纯文字，这一列多出箭头就足以说明「可以点」，
   再加底色或边框反而会和分组标题行抢层级。 */
.sp-election-log-sort {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  border: 0;
  padding: 0;
  background: transparent;
  color: inherit;
  font: inherit;
  letter-spacing: inherit;
  cursor: pointer;
}

.sp-election-log-sort:hover {
  color: var(--sp-election-log-accent);
}

/* 键盘可达：outline 而不是 box-shadow，避免在 sticky 表头上糊出一块底色。 */
.sp-election-log-sort:focus-visible {
  outline: 2px solid var(--sp-election-log-accent);
  outline-offset: 2px;
  border-radius: 2px;
}

/* 箭头指向当前排序方向：朝下 = 最新在前（默认），朝上 = 最早在前。
   用 accent 色标出「这一列正在生效」，与页面其他可交互元素的取色一致。 */
.sp-election-log-sort-icon {
  display: inline-flex;
  color: var(--sp-election-log-accent);
}

.sp-election-log-sort-icon > svg {
  display: block;
  fill: currentColor;
}

.sp-election-log-sort-icon.is-asc {
  transform: rotate(180deg);
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
.sp-election-log-group-stat em.skip { color: var(--sp-election-log-muted); }

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

/* 数据行的「切换时间」：完整日期 + 时间（年月日 时:分:秒）。
   只给时间的话，跨天回看时分不清是哪天的变更。
   nowrap 是必须的 —— 列宽不够时它会静默折成两行、把整行撑高（不报错、不溢出）。 */
.sp-election-log-time {
  white-space: nowrap;
  font-variant-numeric: tabular-nums;
}

.sp-election-log-batch-count {
  margin-left: 10px;
  color: var(--sp-election-log-muted);
  font-size: 11px;
}

/* 批次口径：取前 N 名、入选线、权重、封顶等组级固定参数，同批次同组每行都一样，只在标题写一次。 */
.sp-election-log-batch-calibration {
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

/* 跳过：中性灰。它是「本该动却没动」，与开（绿）/ 关（红）是两套维度，染成任一色都会误读成切换。 */
.sp-election-log-direction.skip {
  color: var(--sp-election-log-muted);
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

/* 原因列的分类色标签：把结论从灰色小字提成一眼可辨的色块，扫一列就能分堆
   （入选 / 收敛关闭 / 健康锁定 / 必需模型补选 / 分组保底 / 未入选 / 测试失败 / 未参与）。
   配色由业务语义驱动，与方向列、批次状态徽标同一套取色（绿好 / 琥珀让位 / 蓝保留 / 紫补选 / 红失败 / 青兜底）。 */
.sp-election-log-reason-head {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 6px;
}

.sp-election-log-reason-badge {
  display: inline-flex;
  align-items: center;
  padding: 1px 8px;
  border: 1px solid var(--sp-election-log-line);
  border-radius: 9999px;
  background: var(--sp-election-log-panel);
  color: var(--sp-election-log-muted);
  font-size: 11px;
  font-weight: 700;
  white-space: nowrap;
}

.sp-election-log-reason-badge.is-elected {
  border-color: color-mix(in srgb, #16a34a 35%, var(--sp-election-log-line));
  background: color-mix(in srgb, #16a34a 8%, var(--sp-election-log-panel));
  color: #16a34a;
}

.sp-election-log-reason-badge.is-converged {
  border-color: color-mix(in srgb, #d97706 35%, var(--sp-election-log-line));
  background: color-mix(in srgb, #d97706 8%, var(--sp-election-log-panel));
  color: #d97706;
}

.sp-election-log-reason-badge.is-locked {
  border-color: color-mix(in srgb, #2563eb 35%, var(--sp-election-log-line));
  background: color-mix(in srgb, #2563eb 8%, var(--sp-election-log-panel));
  color: #2563eb;
}

.sp-election-log-reason-badge.is-required {
  border-color: color-mix(in srgb, #7c3aed 35%, var(--sp-election-log-line));
  background: color-mix(in srgb, #7c3aed 8%, var(--sp-election-log-panel));
  color: #7c3aed;
}

/* 分组保底：青。它是「兜底开启」而不是「凭成绩入选」，所以刻意避开 is-elected 的绿——
   同一列里两者混在一起，扫过去会把保底读成择优结果。 */
.sp-election-log-reason-badge.is-keep-alive {
  border-color: color-mix(in srgb, #0d9488 35%, var(--sp-election-log-line));
  background: color-mix(in srgb, #0d9488 8%, var(--sp-election-log-panel));
  color: #0d9488;
}

.sp-election-log-reason-badge.is-test-failed {
  border-color: color-mix(in srgb, #dc2626 35%, var(--sp-election-log-line));
  background: color-mix(in srgb, #dc2626 8%, var(--sp-election-log-panel));
  color: #dc2626;
}

/* 上游停用：玫红。刻意避开 is-test-failed 的红 —— 两者都要人去处理，但动作完全不同：
   测试失败等它自己翻盘（失败闸门会给缓冲轮次），上游停用则要先去恢复供应商，
   账号自身再"健康"也没用。同色会让人按前者处置，白等。 */
.sp-election-log-reason-badge.is-upstream {
  border-color: color-mix(in srgb, #db2777 35%, var(--sp-election-log-line));
  background: color-mix(in srgb, #db2777 8%, var(--sp-election-log-panel));
  color: #db2777;
}

/* 未入选 / 未参与择优：中性灰。它们不是错误也不是成绩，只是「这次没轮到」，
   染成红或绿都会误读，用默认的灰底灰字即可，不再单独上色。 */

/* 暗色下把语义色亮一档，否则深底上对比不足（与批次状态徽标同样处理）。 */
.dark .sp-election-log-reason-badge.is-elected { color: #4ade80; border-color: color-mix(in srgb, #4ade80 35%, var(--sp-election-log-line)); }
.dark .sp-election-log-reason-badge.is-converged { color: #fbbf24; border-color: color-mix(in srgb, #fbbf24 35%, var(--sp-election-log-line)); }
.dark .sp-election-log-reason-badge.is-locked { color: #60a5fa; border-color: color-mix(in srgb, #60a5fa 35%, var(--sp-election-log-line)); }
.dark .sp-election-log-reason-badge.is-required { color: #a78bfa; border-color: color-mix(in srgb, #a78bfa 35%, var(--sp-election-log-line)); }
.dark .sp-election-log-reason-badge.is-keep-alive { color: #2dd4bf; border-color: color-mix(in srgb, #2dd4bf 35%, var(--sp-election-log-line)); }
.dark .sp-election-log-reason-badge.is-test-failed { color: #f87171; border-color: color-mix(in srgb, #f87171 35%, var(--sp-election-log-line)); }
.dark .sp-election-log-reason-badge.is-upstream { color: #f472b6; border-color: color-mix(in srgb, #f472b6 35%, var(--sp-election-log-line)); }

/* 名次/综合分：等宽数字，方便上下行对齐着比大小。 */
.sp-election-log-reason-rank {
  color: var(--sp-election-log-muted);
  font-size: 11px;
  font-weight: 600;
  font-variant-numeric: tabular-nums;
  white-space: nowrap;
}

/* 标签说不完、但对比时又必须点破的补充（如「该账号在其它分组当选」）。 */
.sp-election-log-reason-note {
  display: block;
  margin-top: 3px;
  color: var(--sp-election-log-muted);
  font-size: 11px;
  line-height: 1.5;
}

/* 评分贡献条：次数/用时/优先三段按加权贡献上色，上下对齐扫一列就看出谁靠哪项赢的。
   只对 scored 行显示（没参与评分就没有构成可言）。 */
.sp-election-log-score-breakdown {
  margin-top: 6px;
}

.sp-election-log-score-bar {
  display: flex;
  height: 18px;
  border-radius: 4px;
  overflow: hidden;
  background: color-mix(in srgb, var(--sp-election-log-line) 30%, transparent);
}

/* 段内标「次0.900」：项名放段首（「次▇▇▇」），不居中 —— 段窄了居中的字会被裁，
   段首至少能和色块起点对上。字宽随段宽变化，**窄段会把读数裁掉**（这是「段宽=比例」的固有代价，
   0 贡献的段已经在 pushSegment 里滤掉，剩下的都是真有贡献的）。 */
.sp-election-log-score-segment {
  display: flex;
  align-items: center;
  justify-content: flex-start;
  position: relative;
  min-width: 1px;
  padding-left: 3px;
}

.sp-election-log-score-segment.is-count {
  background: color-mix(in srgb, #16a34a 50%, var(--sp-election-log-panel));
}

.sp-election-log-score-segment.is-latency {
  background: color-mix(in srgb, #2563eb 50%, var(--sp-election-log-panel));
}

.sp-election-log-score-segment.is-priority {
  background: color-mix(in srgb, #7c3aed 50%, var(--sp-election-log-panel));
}

.dark .sp-election-log-score-segment.is-count {
  background: color-mix(in srgb, #4ade80 40%, var(--sp-election-log-panel));
}

.dark .sp-election-log-score-segment.is-latency {
  background: color-mix(in srgb, #60a5fa 40%, var(--sp-election-log-panel));
}

.dark .sp-election-log-score-segment.is-priority {
  background: color-mix(in srgb, #a78bfa 40%, var(--sp-election-log-panel));
}

/* 段内读数：项名 + 该项的分项得分（如「次0.900」）。 */
.sp-election-log-score-segment-label {
  color: var(--sp-election-log-ink);
  font-size: 10px;
  font-weight: 700;
  opacity: 0.7;
  /* 段窄时宁可横向裁掉尾巴，也不要折成两行 —— 条只有 18px 高，折行后两行都读不全。 */
  white-space: nowrap;
}

/* 单行旗标：只跟这一行相关的标记（封顶命中、用时中性）才显示，不命中就不出现。 */
.sp-election-log-score-flags {
  display: block;
  margin-top: 3px;
  color: var(--sp-election-log-muted);
  font-size: 10px;
  line-height: 1.5;
}

/* 综合分的计算式：原始值 → 分项得分 × 权重 → 求和。
   排在旗标之前（紧贴段条），因为它是段条上那几个读数的来源。
   原因列最窄 200px，长式子必然折行 —— 宁可折行也不省略项，否则综合分复核不了。 */
.sp-election-log-score-formula {
  display: block;
  margin-top: 3px;
  color: var(--sp-election-log-muted);
  font-size: 10px;
  line-height: 1.5;
  word-break: break-word;
}
</style>

<template>
  <SupplierModuleLayout>
    <header class="sp-page-head">
      <div>
        <div class="sp-eyebrow">Automation Tasks</div>
        <h1>自动化任务中心</h1>
        <p class="sp-subtitle">维护供应商同步与数据清理任务，只依赖后台真实任务记录。</p>
      </div>
      <div class="sp-controls">
        <button class="sp-button" type="button" :disabled="loading" @click="loadData">刷新</button>
      </div>
    </header>

    <div v-if="error" class="sp-alert sp-error-line">{{ error }}</div>

    <section class="sp-metric-grid">
      <article v-for="metric in metrics" :key="metric.label" class="sp-metric-card" :class="`sp-${metric.tone}`">
        <div class="sp-metric-label">{{ metric.label }}</div>
        <div class="sp-metric-value">{{ metric.value }}</div>
        <div class="sp-metric-foot">{{ metric.foot }}</div>
      </article>
    </section>

    <section class="sp-grid-2">
      <div class="sp-panel">
        <header class="sp-panel-head">
          <div class="sp-panel-title">
            <span class="sp-section-index">01</span>
            <div>
              <h2>任务配置</h2>
              <span>启用、Cron、超时和保留策略</span>
            </div>
          </div>
        </header>
        <div class="sp-table-wrap">
          <table class="sp-table">
            <thead>
              <tr>
                <th>任务</th>
                <th>启用</th>
                <th>执行间隔</th>
                <th>超时</th>
                <th>上次运行</th>
                <th>最近结果</th>
                <th>下次运行</th>
                <th>操作</th>
              </tr>
            </thead>
            <tbody>
              <tr v-if="loading"><td colspan="8">正在加载任务数据...</td></tr>
              <tr v-for="task in tasks" :key="task.task_code" class="clickable" :class="{ selected: selectedCode === task.task_code }" @click="selectedCode = task.task_code">
                <td><div class="sp-entity">{{ task.name }}</div><div class="sp-sub">{{ task.task_code }}</div></td>
                <td><span class="sp-status" :class="task.enabled ? 'good' : ''">{{ task.enabled ? '已启用' : '已停用' }}</span></td>
                <td><span class="sp-status info">{{ formatInterval(task.cron_expression) }}</span><div class="sp-sub">{{ task.cron_expression }}</div></td>
                <td>{{ task.timeout_seconds }}s</td>
                <td>{{ formatTime(task.last_run_at) }}</td>
                <td>
                  <span class="sp-status" :class="statusTone(task.last_status)">{{ statusText(task.last_status) }}</span>
                  <div class="sp-result-cell">
                    <span class="sp-sub sp-message-preview">{{ taskResultSummary(task) }}</span>
                    <button
                      v-if="task.last_message || latestRunByTask[task.task_code]"
                      class="sp-link-button"
                      type="button"
                      @click.stop="openTaskLatestResult(task)"
                    >
                      查看详情
                    </button>
                  </div>
                </td>
                <td>{{ formatTime(task.next_run_at) }}</td>
                <td>
                  <div class="sp-inline">
                    <button class="sp-button small" type="button" :disabled="savingCode === task.task_code" @click.stop="openEdit(task)">{{ savingCode === task.task_code ? '保存中' : '编辑' }}</button>
                    <button class="sp-button small" type="button" :disabled="runningCode === task.task_code" @click.stop="runNow(task.task_code)">{{ runningCode === task.task_code ? '运行中' : '立即运行' }}</button>
                  </div>
                </td>
              </tr>
              <tr v-if="!loading && !tasks.length"><td colspan="8">暂无自动化任务。</td></tr>
            </tbody>
          </table>
        </div>
      </div>

      <aside class="sp-panel">
        <header class="sp-panel-head">
          <div class="sp-panel-title">
            <span class="sp-section-index">02</span>
            <div>
              <h2>运行历史</h2>
              <span>最近自动化执行记录</span>
            </div>
          </div>
        </header>
        <div class="sp-panel-body">
          <div class="sp-table-wrap">
            <table class="sp-table">
              <thead>
                <tr>
                  <th>任务</th>
                  <th>触发</th>
                  <th>状态</th>
                  <th>处理 / 成功 / 失败</th>
                </tr>
              </thead>
              <tbody>
                <tr v-if="loading"><td colspan="4">正在加载运行记录...</td></tr>
                <tr v-for="run in runs" :key="run.id">
                  <td>{{ run.task_code }}</td>
                  <td>{{ triggerText(run.trigger_source) }}</td>
                  <td>
                    <span class="sp-status" :class="statusTone(run.status)">{{ statusText(run.status) }}</span>
                    <button class="sp-link-button sp-message-preview" type="button" @click="openRunDetail(run)">
                      {{ compactMessage(run.message || '查看详情') }}
                    </button>
                  </td>
                  <td>{{ run.processed_count }} / {{ run.success_count }} / {{ run.failed_count }}</td>
                </tr>
                <tr v-if="!loading && !runs.length"><td colspan="4">暂无运行历史。</td></tr>
              </tbody>
            </table>
          </div>
          <div class="sp-run-pager">
            <button class="sp-button small" type="button" :disabled="runPage <= 1 || loading" @click="changeRunPage(runPage - 1)">上一页</button>
            <span>第 {{ runPage }} / {{ runTotalPages }} 页，共 {{ runTotal }} 条</span>
            <button class="sp-button small" type="button" :disabled="runPage >= runTotalPages || loading" @click="changeRunPage(runPage + 1)">下一页</button>
          </div>
        </div>
      </aside>
    </section>

    <SupplierModal :show="editVisible" :title="editingTask?.name || '编辑任务'" confirm-text="保存任务" @close="closeEdit" @confirm="saveTask">
      <form class="sp-form" @submit.prevent="saveTask">
        <label class="sp-switch-field">
          <span>启用</span>
          <button
            class="sp-switch"
            type="button"
            role="switch"
            :aria-checked="editForm.enabled"
            :class="{ active: editForm.enabled }"
            @click="editForm.enabled = !editForm.enabled"
          >
            <span class="sp-switch-track"><span class="sp-switch-thumb"></span></span>
            <em>{{ editForm.enabled ? '已启用' : '已停用' }}</em>
          </button>
        </label>
        <label>
          <span>执行间隔（秒）</span>
          <input v-model.number="editIntervalSeconds" type="number" min="60" step="60" />
        </label>
        <label><span>超时秒数</span><input v-model.number="editForm.timeout_seconds" type="number" min="1" /></label>
        <div class="sp-form-note">当前调度器按分钟执行，执行间隔必须不少于 60 秒，并且是 60 秒的整数倍。</div>
        <template v-if="editForm.task_code === 'supplier_data_cleanup'">
          <label><span>自动化运行保留天数</span><input v-model.number="editForm.config.automation_run_retention_days" type="number" min="0" /></label>
          <label><span>同步记录保留天数</span><input v-model.number="editForm.config.sync_run_retention_days" type="number" min="0" /></label>
          <label><span>快照保留天数</span><input v-model.number="editForm.config.metric_snapshot_retention_days" type="number" min="0" /></label>
          <label><span>每日统计保留天数</span><input v-model.number="editForm.config.daily_stat_retention_days" type="number" min="0" /></label>
          <label><span>失效账号保留天数</span><input v-model.number="editForm.config.inactive_account_retention_days" type="number" min="0" /></label>
          <label><span>失效分组保留天数</span><input v-model.number="editForm.config.inactive_group_retention_days" type="number" min="0" /></label>
        </template>
      </form>
    </SupplierModal>

    <SupplierModal :show="detailVisible" :title="detailTitle || '结果详情'" confirm-text="关闭" modal-class="sp-modal-wide" @close="closeResultDetail" @confirm="closeResultDetail">
      <div v-if="detailRun" class="sp-run-detail">
        <section class="sp-run-detail-summary">
          <div>
            <span class="sp-detail-label">任务</span>
            <strong>{{ detailRun.task_code }}</strong>
          </div>
          <div>
            <span class="sp-detail-label">触发</span>
            <strong>{{ triggerText(detailRun.trigger_source) }}</strong>
          </div>
          <div>
            <span class="sp-detail-label">状态</span>
            <span class="sp-status" :class="statusTone(detailRun.status)">{{ statusText(detailRun.status) }}</span>
          </div>
          <div>
            <span class="sp-detail-label">处理 / 成功 / 失败</span>
            <strong>{{ detailRun.processed_count }} / {{ detailRun.success_count }} / {{ detailRun.failed_count }}</strong>
          </div>
          <div>
            <span class="sp-detail-label">开始</span>
            <strong>{{ formatTime(detailRun.started_at) }}</strong>
          </div>
          <div>
            <span class="sp-detail-label">结束</span>
            <strong>{{ formatTime(detailRun.finished_at) }}</strong>
          </div>
        </section>

        <div v-if="detailRun.message" class="sp-run-message">{{ detailRun.message }}</div>

        <section v-if="detailRun.result_detail?.providers?.length" class="sp-provider-list">
          <article v-for="provider in detailRun.result_detail.providers" :key="provider.provider_id" class="sp-provider-card">
            <header class="sp-provider-head">
              <div>
                <span class="sp-detail-label">供应商 {{ provider.provider_id }}</span>
                <h3>{{ provider.provider_name || `供应商 ${provider.provider_id}` }}</h3>
              </div>
              <span class="sp-status" :class="statusTone(provider.status)">{{ statusText(provider.status) }}</span>
            </header>
            <div class="sp-provider-stats">
              <span>处理 {{ provider.counts.checked_count }}</span>
              <span>新增 {{ provider.counts.created_count }}</span>
              <span>更新 {{ provider.counts.updated_count }}</span>
              <span>跳过 {{ provider.counts.skipped_count }}</span>
            </div>
            <p v-if="provider.message" class="sp-provider-message">{{ provider.message }}</p>

            <div class="sp-stage-groups">
              <section v-for="category in providerStagesByCategory(provider)" :key="category.key" class="sp-stage-category">
                <h4>{{ category.title }}</h4>
                <article v-for="stage in category.stages" :key="`${provider.provider_id}-${stage.scope}`" class="sp-stage-card" :class="statusTone(stage.status)">
                  <div class="sp-stage-head">
                    <strong>{{ scopeText(stage.scope) }}</strong>
                    <span class="sp-status" :class="statusTone(stage.status)">{{ statusText(stage.status) }}</span>
                  </div>
                  <div class="sp-stage-metrics">
                    <span v-if="stage.http_status">HTTP {{ stage.http_status }}</span>
                    <span v-if="stage.duration_ms !== undefined">{{ stage.duration_ms }}ms</span>
                    <span v-if="stage.response_bytes !== undefined">{{ stage.response_bytes }} bytes</span>
                    <span>处理 {{ stage.counts.checked_count }}</span>
                    <span>更新 {{ stage.counts.updated_count }}</span>
                  </div>
                  <div class="sp-stage-body">
                    <div class="sp-stage-main">
                      <div v-if="stage.endpoint" class="sp-stage-row"><em>接口</em><span>{{ stage.endpoint }}</span></div>
                      <div v-if="stage.parsed_summary" class="sp-stage-row"><em>解析</em><span>{{ stage.parsed_summary }}</span></div>
                      <div v-if="stage.error" class="sp-stage-row bad"><em>错误</em><span>{{ stage.error }}</span></div>
                      <div v-if="stage.parse_error" class="sp-stage-row bad"><em>解析错误</em><span>{{ stage.parse_error }}</span></div>
                      <div v-if="stage.message && stage.message !== '同步成功'" class="sp-stage-row"><em>结果</em><span>{{ stage.message }}</span></div>
                    </div>
                    <aside v-if="stage.response_summary" class="sp-response-panel">
                      <div class="sp-response-panel-head">
                        <span>响应摘要</span>
                        <small>原始返回</small>
                      </div>
                      <pre class="sp-response-summary">{{ stage.response_summary }}</pre>
                    </aside>
                  </div>
                </article>
              </section>
            </div>
          </article>
        </section>

        <section v-else-if="detailRun.result_detail?.cleanup" class="sp-cleanup-grid">
          <article><span>自动化运行</span><strong>{{ detailRun.result_detail.cleanup.automation_runs }}</strong></article>
          <article><span>同步记录</span><strong>{{ detailRun.result_detail.cleanup.sync_runs }}</strong></article>
          <article><span>指标快照</span><strong>{{ detailRun.result_detail.cleanup.metric_snapshots }}</strong></article>
          <article><span>每日统计</span><strong>{{ detailRun.result_detail.cleanup.daily_stats }}</strong></article>
          <article><span>供应商账号</span><strong>{{ detailRun.result_detail.cleanup.accounts }}</strong></article>
          <article><span>供应商分组</span><strong>{{ detailRun.result_detail.cleanup.groups }}</strong></article>
        </section>

        <pre v-else class="sp-message-detail">{{ detailMessage }}</pre>
      </div>
      <pre v-else class="sp-message-detail">{{ detailMessage }}</pre>
    </SupplierModal>

    <Transition name="sp-fade"><div v-if="toast" class="sp-toast">{{ toast }}</div></Transition>
  </SupplierModuleLayout>
</template>

<script setup lang="ts">
import { computed, onMounted, reactive, ref } from 'vue'
import { SupplierModal, SupplierModuleLayout } from '@/components/admin/supplier-management'
import {
  listRuns,
  listTasks,
  runTask,
  updateTask,
  type SupplierAutomationProviderRunDetail,
  type SupplierAutomationRun,
  type SupplierAutomationStageRunDetail,
  type SupplierAutomationTask,
} from '@/api/admin/supplierAutomation'

const tasks = ref<SupplierAutomationTask[]>([])
const runs = ref<SupplierAutomationRun[]>([])
const selectedCode = ref('')
const loading = ref(false)
const savingCode = ref('')
const runningCode = ref('')
const editVisible = ref(false)
const editingTask = ref<SupplierAutomationTask | null>(null)
const editIntervalSeconds = ref(900)
const detailVisible = ref(false)
const detailTitle = ref('')
const detailMessage = ref('')
const detailRun = ref<SupplierAutomationRun | null>(null)
const error = ref('')
const toast = ref('')
const runPage = ref(1)
const runPageSize = ref(10)
const runTotal = ref(0)
let toastTimer: number | undefined

const editForm = reactive<SupplierAutomationTask>({
  id: 0,
  task_code: '',
  name: '',
  enabled: true,
  cron_expression: '',
  timeout_seconds: 600,
  config: {
    automation_run_retention_days: 30,
    sync_run_retention_days: 30,
    metric_snapshot_retention_days: 30,
    daily_stat_retention_days: 365,
    inactive_account_retention_days: 90,
    inactive_group_retention_days: 90,
  },
  last_status: '',
  last_message: '',
})

const selectedTask = computed(() => tasks.value.find(task => task.task_code === selectedCode.value) || tasks.value[0])
const latestRunByTask = computed<Record<string, SupplierAutomationRun>>(() => {
  const latest: Record<string, SupplierAutomationRun> = {}
  for (const run of runs.value) {
    if (!latest[run.task_code]) latest[run.task_code] = run
  }
  return latest
})
const runTotalPages = computed(() => Math.max(1, Math.ceil(runTotal.value / runPageSize.value)))
const metrics = computed(() => [
  { tone: 'green', label: '启用任务', value: String(tasks.value.filter(task => task.enabled).length), foot: '当前可自动执行的任务' },
  { tone: 'blue', label: '最近成功', value: String(runs.value.filter(run => run.status === 'success').length), foot: '最近加载的运行历史' },
  { tone: 'amber', label: '最近失败', value: String(runs.value.filter(run => run.status === 'failed').length), foot: '需要关注的运行记录' },
  { tone: 'red', label: '当前选中', value: selectedTask.value?.name || '无', foot: '点击任务行切换' },
])

onMounted(async () => {
  await loadData()
})

async function loadData() {
  loading.value = true
  error.value = ''
  try {
    tasks.value = await listTasks()
    await loadRuns()
    if (!selectedCode.value && tasks.value[0]) selectedCode.value = tasks.value[0].task_code
  } catch (err) {
    error.value = err instanceof Error ? err.message : '加载自动化任务失败'
  } finally {
    loading.value = false
  }
}

async function loadRuns() {
  const result = await listRuns({ page: runPage.value, page_size: runPageSize.value })
  runs.value = result.items
  runTotal.value = result.total
}

function openEdit(task: SupplierAutomationTask) {
  editingTask.value = task
  selectedCode.value = task.task_code
  Object.assign(editForm, JSON.parse(JSON.stringify(task)))
  editIntervalSeconds.value = cronToIntervalSeconds(task.cron_expression) || 900
  editVisible.value = true
}

function closeEdit() {
  editVisible.value = false
}

async function saveTask() {
  if (!editingTask.value) return
  const cronExpression = intervalSecondsToCron(editIntervalSeconds.value)
  if (!cronExpression) {
    error.value = '执行间隔必须不少于 60 秒，并且是 60 秒的整数倍'
    return
  }
  editForm.cron_expression = cronExpression
  savingCode.value = editingTask.value.task_code
  try {
    await updateTask(editingTask.value.task_code, editForm)
    showToast('任务已保存')
    editVisible.value = false
    await loadData()
  } catch (err) {
    error.value = err instanceof Error ? err.message : '保存任务失败'
  } finally {
    savingCode.value = ''
  }
}

async function runNow(taskCode: string) {
  runningCode.value = taskCode
  try {
    const run = await runTask(taskCode)
    showToast(`任务执行完成：${statusText(run.status)}`)
    runPage.value = 1
    await loadData()
  } catch (err) {
    error.value = err instanceof Error ? err.message : '运行任务失败'
  } finally {
    runningCode.value = ''
  }
}

function openResultDetail(title: string, message: string) {
  detailRun.value = null
  detailTitle.value = title || '结果详情'
  detailMessage.value = message || '暂无结果'
  detailVisible.value = true
}

function openTaskLatestResult(task: SupplierAutomationTask) {
  const run = latestRunByTask.value[task.task_code]
  if (run) {
    openRunDetail(run)
    return
  }
  openResultDetail(`${task.name} 最近结果`, task.last_message)
}

function openRunDetail(run: SupplierAutomationRun) {
  detailRun.value = run
  detailTitle.value = `${run.task_code} 运行详情：${statusText(run.status)}`
  detailMessage.value = formatRunDetail(run)
  detailVisible.value = true
}

function formatRunDetail(run: SupplierAutomationRun): string {
  const lines = [
    `任务：${run.task_code}`,
    `触发：${triggerText(run.trigger_source)}`,
    `状态：${statusText(run.status)}`,
    `处理 / 成功 / 失败：${run.processed_count} / ${run.success_count} / ${run.failed_count}`,
    `开始时间：${formatTime(run.started_at)}`,
    `结束时间：${formatTime(run.finished_at)}`,
    '',
    run.message || '暂无结果',
  ]
  const providers = run.result_detail?.providers || []
  if (providers.length) {
    lines.push('', '接口明细：')
    for (const provider of providers) {
      lines.push(...formatProviderRunDetail(provider))
    }
  } else if (run.result_detail?.cleanup) {
    const cleanup = run.result_detail.cleanup
    lines.push(
      '',
      '清理明细：',
      `- 自动化运行：${cleanup.automation_runs}`,
      `- 同步记录：${cleanup.sync_runs}`,
      `- 指标快照：${cleanup.metric_snapshots}`,
      `- 每日统计：${cleanup.daily_stats}`,
      `- 供应商账号：${cleanup.accounts}`,
      `- 供应商分组：${cleanup.groups}`
    )
  }
  return lines.join('\n')
}

function formatProviderRunDetail(provider: SupplierAutomationProviderRunDetail): string[] {
  const title = provider.provider_name || `供应商 ${provider.provider_id}`
  const lines = [
    '',
    `供应商 ${provider.provider_id}：${title}`,
    `状态：${statusText(provider.status)}；处理 / 新增 / 更新 / 跳过：${provider.counts.checked_count} / ${provider.counts.created_count} / ${provider.counts.updated_count} / ${provider.counts.skipped_count}`,
  ]
  if (provider.message) lines.push(`结果：${provider.message}`)
  for (const stage of provider.stages || []) {
    lines.push(...formatStageRunDetail(stage))
  }
  return lines
}

function formatStageRunDetail(stage: SupplierAutomationStageRunDetail): string[] {
  const lines = [
    `  - ${scopeText(stage.scope)}：${statusText(stage.status)}`,
    `    计数：${stage.counts.checked_count} / ${stage.counts.created_count} / ${stage.counts.updated_count} / ${stage.counts.skipped_count}`,
  ]
  if (stage.endpoint) lines.push(`    接口：${stage.endpoint}`)
  if (stage.http_status) lines.push(`    HTTP：${stage.http_status}`)
  if (stage.duration_ms !== undefined) lines.push(`    耗时：${stage.duration_ms}ms`)
  if (stage.response_bytes !== undefined) lines.push(`    返回大小：${stage.response_bytes} bytes`)
  if (stage.parsed_summary) lines.push(`    解析摘要：${stage.parsed_summary}`)
  if (stage.error) lines.push(`    错误：${stage.error}`)
  if (stage.parse_error) lines.push(`    解析错误：${stage.parse_error}`)
  if (stage.response_summary) lines.push(`    响应摘要：${stage.response_summary}`)
  if (stage.message && stage.message !== '同步成功') lines.push(`    结果：${stage.message}`)
  return lines
}

function providerStagesByCategory(provider: SupplierAutomationProviderRunDetail) {
  const stages = provider.stages || []
  const categories = [
    { key: 'identity', title: '账号与分组', scopes: ['accounts', 'groups'] },
    { key: 'metrics', title: '余额与成本', scopes: ['balance', 'cost'] },
    { key: 'other', title: '其他接口', scopes: [] },
  ]
  return categories
    .map(category => ({
      key: category.key,
      title: category.title,
      stages: category.key === 'other'
        ? stages.filter(stage => !['accounts', 'groups', 'balance', 'cost'].includes(stage.scope))
        : stages.filter(stage => category.scopes.includes(stage.scope)),
    }))
    .filter(category => category.stages.length > 0)
}

function closeResultDetail() {
  detailVisible.value = false
  detailRun.value = null
}

function taskResultSummary(task: SupplierAutomationTask): string {
  const run = latestRunByTask.value[task.task_code]
  if (run) return runSummary(run)
  return compactMessage(task.last_message || '暂无结果')
}

function runSummary(run: SupplierAutomationRun): string {
  if (!run.processed_count && !run.success_count && !run.failed_count) {
    return compactMessage(run.message || '暂无结果')
  }
  return `${run.processed_count} 个对象，${run.success_count} 成功，${run.failed_count} 失败`
}

async function changeRunPage(page: number) {
  runPage.value = Math.min(Math.max(1, page), runTotalPages.value)
  loading.value = true
  error.value = ''
  try {
    await loadRuns()
  } catch (err) {
    error.value = err instanceof Error ? err.message : '加载运行历史失败'
  } finally {
    loading.value = false
  }
}

function compactMessage(message: string): string {
  const normalized = String(message || '').replace(/\s+/g, ' ').trim()
  if (!normalized) return '暂无结果'
  return normalized.length > 80 ? `${normalized.slice(0, 80)}...` : normalized
}

function statusTone(status?: string): string {
  if (status === 'failed') return 'bad'
  if (status === 'partial') return 'warn'
  if (status === 'success') return 'good'
  return ''
}

function statusText(status?: string): string {
  if (status === 'failed') return '失败'
  if (status === 'partial') return '部分成功'
  if (status === 'success') return '成功'
  if (status === 'running') return '运行中'
  return '未运行'
}

function triggerText(trigger?: string): string {
  if (trigger === 'scheduled') return '定时执行'
  if (trigger === 'manual') return '手动执行'
  return trigger || '未知'
}

function scopeText(scope?: string): string {
  if (scope === 'accounts') return '账号接口'
  if (scope === 'groups') return '分组接口'
  if (scope === 'balance') return '余额接口'
  if (scope === 'cost') return '成本接口'
  if (scope === 'all') return '全量同步'
  return scope || '未知接口'
}

function formatTime(value?: string): string {
  if (!value) return '—'
  const date = new Date(value)
  if (Number.isNaN(date.getTime())) return '—'
  return date.toLocaleString('zh-CN')
}

function formatInterval(cronExpression: string): string {
  const seconds = cronToIntervalSeconds(cronExpression)
  if (!seconds) return cronExpression || '未配置'
  if (seconds % 86400 === 0) return `每 ${seconds / 86400} 天`
  if (seconds % 3600 === 0) return `每 ${seconds / 3600} 小时`
  if (seconds % 60 === 0) return `每 ${seconds / 60} 分钟`
  return `每 ${seconds} 秒`
}

function cronToIntervalSeconds(cronExpression: string): number | null {
  const parts = cronExpression.trim().split(/\s+/)
  if (parts.length !== 5) return null
  const [minute, hour, dayOfMonth, month, dayOfWeek] = parts
  if (hour === '*' && dayOfMonth === '*' && month === '*' && dayOfWeek === '*') {
    const minuteMatch = minute.match(/^\*\/(\d+)$/)
    if (minuteMatch) return Number(minuteMatch[1]) * 60
    if (minute === '*') return 60
  }
  if (minute === '0' && dayOfMonth === '*' && month === '*' && dayOfWeek === '*') {
    const hourMatch = hour.match(/^\*\/(\d+)$/)
    if (hourMatch) return Number(hourMatch[1]) * 3600
    if (hour === '0') return 86400
  }
  if (dayOfMonth === '*' && month === '*' && dayOfWeek === '*' && minute !== '*' && hour !== '*') {
    return 86400
  }
  return null
}

function intervalSecondsToCron(seconds: number): string | null {
  if (!Number.isFinite(seconds) || seconds < 60 || seconds % 60 !== 0) return null
  if (seconds === 86400) return '0 0 * * *'
  if (seconds % 3600 === 0) {
    const hours = seconds / 3600
    if (hours >= 1 && hours < 24) return `0 */${hours} * * *`
  }
  const minutes = seconds / 60
  if (minutes >= 1 && minutes < 60) return `*/${minutes} * * * *`
  return null
}

function showToast(message: string) {
  toast.value = message
  window.clearTimeout(toastTimer)
  toastTimer = window.setTimeout(() => { toast.value = '' }, 1800)
}
</script>

<style scoped>
.sp-result-cell {
  display: grid;
  gap: 6px;
  max-width: 220px;
}

.sp-run-pager {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  margin-top: 12px;
  color: rgba(148, 163, 184, 0.95);
  font-size: 12px;
}

.sp-message-detail {
  max-width: min(780px, 78vw);
  max-height: 68vh;
  white-space: pre-wrap;
  word-break: break-word;
  overflow: auto;
}

.sp-run-detail {
  display: grid;
  gap: 20px;
  max-width: min(920px, 82vw);
  max-height: 72vh;
  overflow: auto;
  padding-right: 4px;
}

.sp-run-detail-summary {
  display: grid;
  grid-template-columns: repeat(3, minmax(0, 1fr));
  gap: 14px;
}

.sp-run-detail-summary > div,
.sp-cleanup-grid > article {
  border: 1px solid var(--sp-soft);
  border-radius: 14px;
  background: var(--sp-panel-2);
  padding: 12px;
}

.sp-detail-label {
  display: block;
  margin-bottom: 5px;
  color: var(--sp-muted);
  font-size: 12px;
}

.sp-run-message,
.sp-provider-message {
  border-left: 3px solid var(--sp-cyan);
  color: var(--sp-text);
  background: color-mix(in srgb, var(--sp-cyan) 7%, var(--sp-panel));
  border-radius: 12px;
  padding: 10px 12px;
  line-height: 1.65;
}

.sp-provider-list {
  display: grid;
  gap: 18px;
}

.sp-provider-card {
  display: grid;
  gap: 16px;
  border: 1px solid var(--sp-line);
  border-radius: 18px;
  background: var(--sp-panel);
  box-shadow: 0 10px 28px rgba(15, 23, 42, 0.06);
  padding: 18px;
}

.sp-provider-head,
.sp-stage-head {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 12px;
}

.sp-provider-head h3 {
  margin: 0;
  color: var(--sp-text);
  font-size: 16px;
}

.sp-provider-stats,
.sp-stage-metrics {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
  margin-top: 0;
}

.sp-provider-stats span,
.sp-stage-metrics span {
  border: 1px solid var(--sp-line);
  border-radius: 999px;
  color: var(--sp-muted);
  background: var(--sp-panel-2);
  padding: 4px 9px;
  font-size: 12px;
}

.sp-stage-groups {
  display: grid;
  gap: 18px;
  margin-top: 0;
}

.sp-stage-category h4 {
  margin: 0 0 12px;
  color: var(--sp-text);
  font-size: 13px;
}

.sp-stage-card {
  display: grid;
  gap: 14px;
  border: 1px solid var(--sp-soft);
  border-radius: 14px;
  background: var(--sp-panel-2);
  padding: 16px;
}

.sp-stage-card.good {
  border-color: color-mix(in srgb, var(--sp-green) 28%, var(--sp-line));
}

.sp-stage-card.warn {
  border-color: color-mix(in srgb, var(--sp-amber) 32%, var(--sp-line));
}

.sp-stage-card.bad {
  border-color: color-mix(in srgb, var(--sp-red) 32%, var(--sp-line));
}

.sp-stage-body {
  display: grid;
  grid-template-columns: minmax(0, 1fr) minmax(280px, 0.8fr);
  gap: 16px;
  align-items: stretch;
}

.sp-stage-main {
  display: grid;
  align-content: start;
  gap: 9px;
}

.sp-stage-row {
  display: grid;
  grid-template-columns: 76px minmax(0, 1fr);
  gap: 10px;
  color: var(--sp-text);
  font-size: 12px;
  line-height: 1.55;
}

.sp-stage-row em {
  color: var(--sp-muted);
  font-style: normal;
}

.sp-stage-row.bad span {
  color: var(--sp-red);
}

.sp-response-panel {
  display: grid;
  grid-template-rows: auto minmax(0, 1fr);
  min-width: 0;
  border: 1px solid var(--sp-soft);
  border-radius: 12px;
  background: var(--sp-panel);
  overflow: hidden;
}

.sp-response-panel-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 10px;
  border-bottom: 1px solid var(--sp-soft);
  background: var(--sp-panel-2);
  padding: 8px 10px;
}

.sp-response-panel-head span {
  color: var(--sp-text);
  font-size: 12px;
  font-weight: 700;
}

.sp-response-panel-head small {
  color: var(--sp-muted);
  font-size: 11px;
}

.sp-response-summary {
  min-height: 130px;
  max-height: 260px;
  margin: 0;
  overflow: auto;
  white-space: pre-wrap;
  word-break: break-word;
  border: 0;
  border-radius: 0;
  background: var(--sp-panel);
  color: var(--sp-text);
  padding: 10px;
  font-size: 12px;
  line-height: 1.6;
}

.sp-cleanup-grid {
  display: grid;
  grid-template-columns: repeat(3, minmax(0, 1fr));
  gap: 10px;
  margin-top: 16px;
}

.sp-cleanup-grid span {
  display: block;
  color: var(--sp-muted);
  font-size: 12px;
}

.sp-cleanup-grid strong {
  display: block;
  margin-top: 6px;
  color: var(--sp-text);
  font-size: 20px;
}

@media (max-width: 760px) {
  .sp-run-detail-summary,
  .sp-cleanup-grid,
  .sp-stage-body {
    grid-template-columns: 1fr;
  }
}
</style>

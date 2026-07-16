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
                  <button
                    v-if="task.last_message"
                    class="sp-link-button sp-message-preview"
                    type="button"
                    @click.stop="openResultDetail(`${task.name} 最近结果`, task.last_message)"
                  >
                    {{ compactMessage(task.last_message || '暂无结果') }}
                  </button>
                  <div v-else class="sp-sub sp-message-preview">{{ compactMessage(task.last_message || '暂无结果') }}</div>
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
                    <button
                      v-if="run.message"
                      class="sp-link-button sp-message-preview"
                      type="button"
                      @click="openResultDetail(`${run.task_code} 运行详情`, run.message)"
                    >
                      {{ compactMessage(run.message || '—') }}
                    </button>
                    <div v-else class="sp-sub sp-message-preview">{{ compactMessage(run.message || '—') }}</div>
                  </td>
                  <td>{{ run.processed_count }} / {{ run.success_count }} / {{ run.failed_count }}</td>
                </tr>
                <tr v-if="!loading && !runs.length"><td colspan="4">暂无运行历史。</td></tr>
              </tbody>
            </table>
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

    <SupplierModal :show="detailVisible" :title="detailTitle || '结果详情'" confirm-text="关闭" @close="closeResultDetail" @confirm="closeResultDetail">
      <pre class="sp-message-detail">{{ detailMessage }}</pre>
    </SupplierModal>

    <Transition name="sp-fade"><div v-if="toast" class="sp-toast">{{ toast }}</div></Transition>
  </SupplierModuleLayout>
</template>

<script setup lang="ts">
import { computed, onMounted, reactive, ref } from 'vue'
import { SupplierModal, SupplierModuleLayout } from '@/components/admin/supplier-management'
import { listRuns, listTasks, runTask, updateTask, type SupplierAutomationRun, type SupplierAutomationTask } from '@/api/admin/supplierAutomation'

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
const error = ref('')
const toast = ref('')
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
    runs.value = (await listRuns({ page: 1, page_size: 20 })).items
    if (!selectedCode.value && tasks.value[0]) selectedCode.value = tasks.value[0].task_code
  } catch (err) {
    error.value = err instanceof Error ? err.message : '加载自动化任务失败'
  } finally {
    loading.value = false
  }
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
    await runTask(taskCode)
    showToast('任务已提交执行')
    await loadData()
  } catch (err) {
    error.value = err instanceof Error ? err.message : '运行任务失败'
  } finally {
    runningCode.value = ''
  }
}

function openResultDetail(title: string, message: string) {
  detailTitle.value = title || '结果详情'
  detailMessage.value = message || '暂无结果'
  detailVisible.value = true
}

function closeResultDetail() {
  detailVisible.value = false
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

<template>
  <BaseDialog
    :show="show"
    title="健康守护任务详情"
    width="extra-wide"
    @close="emit('close')"
  >
    <div v-if="loading" class="sp-health-guard-run-state">正在加载最近一次运行…</div>
    <div v-else-if="!run" class="sp-health-guard-run-state">暂无健康守护运行记录。</div>
    <div v-else-if="run" :class="['sp-run-detail', statusTone(run.status)]">
      <!-- 执行结论：与自动化页「运行详情」弹窗同一结构。
           ⚠️ 那边是**内联实现**，被 SupplierAutomationView.spec.ts 的源码断言钉死
           （要求弹窗内「执行结论」与「结果明细」是两个独立分区、且各 detail 分支顺序固定），
           因此无法抽成共享组件复用 —— 这里按同一结构复刻，class 名刻意保持同名。
           改一处务必同步另一处（含下面四个纯映射函数的文案）。 -->
      <section class="sp-detail-outcome">
        <div class="sp-detail-section-head">
          <div>
            <h3>执行结论</h3>
          </div>
          <span class="sp-status" :class="statusTone(run.status)">{{ statusText(run.status) }}</span>
        </div>

        <section class="sp-run-detail-summary">
          <div class="sp-summary-item sp-summary-task">
            <span class="sp-detail-label">任务</span>
            <!-- 自动化页此处是 taskName(task_code)（依赖任务列表接口）。
                 本弹窗只服务健康守护一个任务，故直接写死；取值须与
                 migrations/190 里 supplier_automation_tasks.name 保持一致。 -->
            <strong>供应商账号健康守护</strong>
          </div>
          <div class="sp-summary-item sp-summary-trigger">
            <span class="sp-detail-label">触发</span>
            <strong>{{ triggerText(run.trigger_source) }}</strong>
          </div>
          <div class="sp-summary-item sp-summary-status" :class="statusTone(run.status)">
            <span class="sp-detail-label">状态</span>
            <span class="sp-status" :class="statusTone(run.status)">{{ statusText(run.status) }}</span>
          </div>
          <div class="sp-summary-item sp-summary-counts">
            <span class="sp-detail-label">处理 / 成功 / 失败</span>
            <strong>{{ run.processed_count }} / {{ run.success_count }} / {{ run.failed_count }}</strong>
          </div>
          <div class="sp-summary-item sp-summary-start">
            <span class="sp-detail-label">开始</span>
            <strong>{{ formatTime(run.started_at) }}</strong>
          </div>
          <div class="sp-summary-item sp-summary-end">
            <span class="sp-detail-label">结束</span>
            <strong>{{ formatTime(run.finished_at) }}</strong>
          </div>
        </section>

        <div v-if="run.message" class="sp-run-message">{{ run.message }}</div>
      </section>

      <section class="sp-detail-content">
        <div class="sp-detail-section-head">
          <div>
            <h3>结果明细</h3>
          </div>
        </div>

        <SupplierAccountHealthGuardResult
          v-if="result"
          :key="run.id"
          :result="result"
        />
      </section>
    </div>
    <template #footer>
      <button class="sp-button primary" type="button" @click="emit('close')">关闭</button>
    </template>
  </BaseDialog>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import BaseDialog from '@/components/common/BaseDialog.vue'
import type { SupplierAutomationRun } from '@/api/admin/supplierAutomation'
import SupplierAccountHealthGuardResult from './SupplierAccountHealthGuardResult.vue'

const props = defineProps<{
  show: boolean
  run: SupplierAutomationRun | null
  loading?: boolean
}>()

const emit = defineEmits<{ (e: 'close'): void }>()

const result = computed(() => props.run?.result_detail?.account_health_guard || null)

// 以下四个是纯映射函数，与 SupplierAutomationView 里的同名实现保持一字不差
// （那边被 spec 钉死为内联实现，抽不出来）—— 改文案时两处要一起改。
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
  if (status === 'skipped') return '已跳过'
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
</script>

<style scoped>
/* ⚠️ 本弹窗经 BaseDialog Teleport 到 body，页面上的 --sp-* 变量取不到，必须在这里就地声明一整套
   （AGENTS.md：弹窗内配色不得只依赖页面根节点变量）。
   取值与自动化页结果详情弹窗逐字相同 —— 两处不一致会让同一个「执行结论」区块在两个页面颜色不同。 */
:global(.modal-content:has(.sp-run-detail)) {
  --sp-panel: #ffffff;
  --sp-panel-2: #f8fafc;
  --sp-panel-3: #eef2f7;
  --sp-line: #d7e0ea;
  --sp-soft: #e8eef5;
  --sp-text: #172033;
  --sp-muted: #607089;
  --sp-cyan: #0284c7;
  --sp-green: #16835d;
  --sp-amber: #c56a0a;
  --sp-orange: #dd5f16;
  --sp-red: #d14343;
  --sp-blue: #2563eb;
  --sp-violet: #6d5bd0;
  --sp-result-blue-soft: #eaf2ff;
  --sp-result-cyan-soft: #e6f6fb;
  --sp-result-green-soft: #e8f7ef;
  --sp-result-amber-soft: #fff3dc;
  --sp-result-red-soft: #fff0f0;
  --sp-result-violet-soft: #f1efff;
  --sp-result-neutral-soft: #f3f6fa;
  overflow: hidden;
  border-color: #cbd7e5;
  background: var(--sp-panel);
  color: var(--sp-text);
}

:global(.dark .modal-content:has(.sp-run-detail)) {
  --sp-panel: #172033;
  --sp-panel-2: #1d293d;
  --sp-panel-3: #243249;
  --sp-line: #35445c;
  --sp-soft: #2c3a51;
  --sp-text: #edf3fb;
  --sp-muted: #a8b6ca;
  --sp-result-blue-soft: #1b3155;
  --sp-result-cyan-soft: #153947;
  --sp-result-green-soft: #173a31;
  --sp-result-amber-soft: #432f1d;
  --sp-result-red-soft: #48272d;
  --sp-result-violet-soft: #302b51;
  --sp-result-neutral-soft: #202d42;
  border-color: #3b4b64;
}

:global(.modal-content:has(.sp-run-detail) .modal-header) {
  border-bottom-color: var(--sp-line);
  background: var(--sp-panel);
}

:global(.modal-content:has(.sp-run-detail) .modal-title) {
  color: var(--sp-text);
}

:global(.modal-content:has(.sp-run-detail)) {
  max-height: min(95vh, calc(100dvh - 16px));
}

/* 明细表是 9 列、内容宽约 1213px，extra-wide 档（xl:max-w-6xl = 1152px）放不下会横向滚动
   ⇒ 与自动化页一样按视口给 80vw。权重：本选择器 = .modal-content + :has() 内的 .sp-run-detail
   两个类，高于 Tailwind 的 .xl\:max-w-6xl（一个类），不依赖源码顺序。 */
@media (min-width: 640px) {
  :global(.modal-content:has(.sp-run-detail)) {
    width: 80vw;
    max-width: 80vw;
  }
}

.sp-health-guard-run-state {
  padding: 2.5rem 1rem;
  color: var(--sp-muted);
  text-align: center;
  font-size: 0.875rem;
}

.sp-run-detail {
  --sp-result-accent: var(--sp-cyan);
  display: grid;
  /* 宽度跟随父容器（.modal-body 是 flex column + stretch），显式 100% 是为了父容器布局变化时兜住。 */
  max-width: 100%;
  /* 高度同样交给父容器：父容器高度确定时正好等于它；为 auto 时百分比无法解析、等价于取消上限，
     此时靠 flex-shrink + overflow: auto 把内容压回弹窗内滚动。 */
  max-height: 100%;
  overflow: auto;
  border: 0;
  background: transparent;
  box-shadow: none;
  padding: 4px 2px 12px;
}

.sp-detail-outcome,
.sp-detail-content {
  display: grid;
  gap: 14px;
}

.sp-detail-content {
  gap: 8px;
  border-top: 1px solid var(--sp-line);
  margin-top: 8px;
  padding-top: 10px;
}

.sp-detail-section-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 16px;
}

.sp-detail-section-head h3 {
  margin: 0;
  color: var(--sp-text);
  font-size: 16px;
  line-height: 1.25;
}

.sp-run-detail.good {
  --sp-result-accent: var(--sp-green);
}

.sp-run-detail.warn {
  --sp-result-accent: var(--sp-amber);
}

.sp-run-detail.bad {
  --sp-result-accent: var(--sp-red);
}

.sp-run-detail .sp-status {
  border-width: 1px;
  border-style: solid;
  font-weight: 700;
}

.sp-run-detail .sp-status.good {
  border-color: color-mix(in srgb, var(--sp-green) 38%, var(--sp-line));
  background: var(--sp-result-green-soft);
  color: var(--sp-green);
}

.sp-run-detail .sp-status.warn {
  border-color: color-mix(in srgb, var(--sp-amber) 42%, var(--sp-line));
  background: var(--sp-result-amber-soft);
  color: var(--sp-amber);
}

.sp-run-detail .sp-status.bad {
  border-color: color-mix(in srgb, var(--sp-red) 44%, var(--sp-line));
  background: var(--sp-result-red-soft);
  color: var(--sp-red);
}

.sp-run-detail-summary {
  display: grid;
  /* 一行 6 格。最长的是时间戳（19 字符约 152px）与任务中文名，等分列都放得下。 */
  grid-template-columns: repeat(6, minmax(0, 1fr));
  row-gap: 18px;
  border-bottom: 1px solid var(--sp-line);
  padding: 4px 0 18px;
}

.sp-summary-item {
  min-width: 0;
  border-left: 1px solid var(--sp-line);
  padding: 2px 18px 4px;
}

.sp-summary-item:first-child {
  border-left: 0;
  padding-left: 0;
}

.sp-summary-task {
  --sp-summary-accent: var(--sp-blue);
}

.sp-summary-trigger {
  --sp-summary-accent: var(--sp-violet);
}

.sp-summary-status {
  --sp-summary-accent: var(--sp-amber);
}

.sp-summary-status.good {
  --sp-summary-accent: var(--sp-green);
}

.sp-summary-status.warn {
  --sp-summary-accent: var(--sp-amber);
}

.sp-summary-status.bad {
  --sp-summary-accent: var(--sp-red);
}

.sp-summary-counts {
  --sp-summary-accent: var(--sp-cyan);
}

.sp-summary-start {
  --sp-summary-accent: var(--sp-green);
}

.sp-summary-end {
  --sp-summary-accent: var(--sp-amber);
}

.sp-run-detail-summary strong {
  color: color-mix(in srgb, var(--sp-summary-accent, var(--sp-text)) 38%, var(--sp-text));
  font-weight: 800;
}

.sp-detail-label {
  display: block;
  margin-bottom: 5px;
  color: var(--sp-muted);
  font-size: 12px;
  font-weight: 700;
  letter-spacing: 0;
}

.sp-run-message {
  margin: 0;
  border: 0;
  border-left: 3px solid var(--sp-result-accent);
  color: var(--sp-text);
  background: color-mix(in srgb, var(--sp-result-accent, var(--sp-cyan)) 7%, var(--sp-panel));
  border-radius: 0;
  padding: 10px 12px;
  line-height: 1.65;
}

/* 窄屏把 6 格摘要切回 2 列 / 1 列，与自动化页的两处媒体查询一致（断点 1024 / 760）。 */
@media (max-width: 1024px) {
  .sp-run-detail-summary {
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }

  .sp-summary-item,
  .sp-summary-item:nth-child(3n + 1) {
    border-left: 1px solid var(--sp-line);
    padding-left: 18px;
  }

  .sp-summary-item:nth-child(odd) {
    border-left: 0;
    padding-left: 0;
  }
}

@media (max-width: 760px) {
  .sp-run-detail-summary {
    grid-template-columns: minmax(0, 1fr);
    row-gap: 0;
  }

  .sp-summary-item,
  .sp-summary-item:nth-child(3n + 1) {
    border-top: 1px solid var(--sp-line);
    border-left: 0;
    padding: 12px 0;
  }

  .sp-summary-item:first-child {
    border-top: 0;
  }
}
</style>

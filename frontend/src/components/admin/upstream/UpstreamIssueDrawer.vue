<template>
  <Teleport to="body">
    <div v-if="issue" class="drawer-overlay" @click.self="$emit('close')">
      <aside data-test="issue-drawer" class="drawer" role="dialog" aria-modal="true" :aria-label="issue.title">
        <header class="drawer-header">
          <div>
            <span :class="['severity', `severity-${issue.severity}`]">{{ severityLabel(issue.severity) }}</span>
            <h2>{{ issue.title }}</h2>
            <p>{{ formatDate(issue.detected_at) }} · {{ issue.source }}</p>
          </div>
          <button type="button" class="close-button" aria-label="关闭" @click="$emit('close')">×</button>
        </header>
        <section class="explanation">
          <strong>发生了什么？</strong>
          <p>{{ issue.description || '系统检测到需要管理员关注的上游异常。' }}</p>
        </section>
        <div class="impact-grid">
          <div><span>影响数量</span><strong>{{ issue.impact_count }}</strong></div>
          <div><span>异常类型</span><strong>{{ issue.type }}</strong></div>
        </div>
        <section class="next-step">
          <strong>建议下一步</strong>
          <p>{{ actionDescription(issue.action) }}</p>
        </section>
        <footer class="drawer-footer">
          <button type="button" class="secondary-button" @click="$emit('close')">稍后处理</button>
          <button v-if="issue.target_path" data-test="issue-primary-action" type="button" class="primary-button" @click="$emit('primary', issue)">
            前往处理
          </button>
        </footer>
      </aside>
    </div>
  </Teleport>
</template>

<script setup lang="ts">
import type { UpstreamDashboardIssue, UpstreamDashboardSeverity } from '@/api/admin/upstreamDashboard'

defineProps<{ issue: UpstreamDashboardIssue | null }>()
defineEmits<{ close: []; primary: [issue: UpstreamDashboardIssue] }>()

function severityLabel(severity: UpstreamDashboardSeverity) {
  return { critical: '严重', high: '高优先级', medium: '中优先级', low: '低优先级' }[severity]
}

function formatDate(value: string) {
  const date = new Date(value)
  return Number.isNaN(date.getTime()) ? value : date.toLocaleString()
}

function actionDescription(action?: string) {
  if (action === 'preview_rate_fix') return '查看最新倍率差异，在确认影响范围后执行修复。'
  if (action === 'resolve_conflicts') return '进入账号页核对冲突账号，确认正确的本地匹配关系。'
  if (action === 'view_balance') return '检查供应商余额接口和最近采样记录。'
  if (action === 'view_health') return '查看健康巡检记录和受影响账号。'
  return '进入对应业务页查看详细状态。'
}
</script>

<style scoped>
.drawer-overlay { @apply fixed inset-0 z-[70] flex justify-end bg-gray-950/45 backdrop-blur-[2px]; }
.drawer { @apply flex h-full w-full max-w-2xl flex-col overflow-y-auto bg-white p-6 shadow-2xl dark:bg-gray-900; animation: drawer-in 180ms ease-out; }
.drawer-header { @apply flex items-start justify-between gap-4 border-b border-gray-200 pb-5 dark:border-gray-700; }
.drawer-header h2 { @apply mt-3 text-xl font-bold text-gray-950 dark:text-white; }
.drawer-header p { @apply mt-1 text-xs text-gray-500; }
.severity { @apply inline-flex rounded-full px-2.5 py-1 text-xs font-semibold; }
.severity-critical { @apply bg-red-100 text-red-700 dark:bg-red-900/40 dark:text-red-300; }
.severity-high { @apply bg-orange-100 text-orange-700 dark:bg-orange-900/40 dark:text-orange-300; }
.severity-medium { @apply bg-amber-100 text-amber-700 dark:bg-amber-900/40 dark:text-amber-300; }
.severity-low { @apply bg-gray-100 text-gray-600 dark:bg-gray-700 dark:text-gray-300; }
.close-button { @apply flex h-11 w-11 items-center justify-center rounded-xl text-2xl text-gray-400 hover:bg-gray-100 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary-500 dark:hover:bg-gray-800; }
.explanation { @apply mt-6 rounded-xl border border-amber-200 bg-amber-50 p-4 dark:border-amber-800 dark:bg-amber-950/30; }
.explanation strong { @apply text-sm text-amber-900 dark:text-amber-200; }
.explanation p, .next-step p { @apply mt-2 text-sm leading-6 text-gray-600 dark:text-gray-300; }
.impact-grid { @apply mt-5 grid grid-cols-2 gap-3; }
.impact-grid div { @apply rounded-xl bg-gray-50 p-4 dark:bg-gray-800; }
.impact-grid span { @apply block text-xs text-gray-500; }
.impact-grid strong { @apply mt-1 block text-lg text-gray-950 dark:text-white; }
.next-step { @apply mt-6; }
.next-step strong { @apply text-sm text-gray-950 dark:text-white; }
.drawer-footer { @apply mt-auto flex justify-end gap-3 border-t border-gray-200 pt-5 dark:border-gray-700; }
.primary-button, .secondary-button { @apply min-h-11 rounded-xl px-4 text-sm font-semibold focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary-500; }
.primary-button { @apply bg-primary-600 text-white hover:bg-primary-700; }
.secondary-button { @apply border border-gray-300 text-gray-700 hover:bg-gray-50 dark:border-gray-600 dark:text-gray-200 dark:hover:bg-gray-800; }
@keyframes drawer-in { from { transform: translateX(24px); opacity: .6; } to { transform: translateX(0); opacity: 1; } }
@media (prefers-reduced-motion: reduce) { .drawer { animation: none; } }
</style>

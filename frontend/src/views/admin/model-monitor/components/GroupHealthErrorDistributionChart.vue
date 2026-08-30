<template>
  <section class="gh-chart-card" aria-labelledby="error-distribution-title">
    <div class="gh-chart-heading">
      <div>
        <p class="gh-chart-kicker">故障画像</p>
        <h4 id="error-distribution-title">错误分类分布</h4>
      </div>
      <span class="gh-chart-unit">{{ totalErrors }} 次</span>
    </div>
    <div class="gh-chart-stage">
      <Bar v-if="chartData" :data="chartData" :options="chartOptions" />
      <div v-else class="gh-chart-empty">{{ loading ? '正在读取错误分类…' : '当前范围暂无错误记录' }}</div>
    </div>
  </section>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { BarElement, CategoryScale, Chart as ChartJS, Legend, LinearScale, Title, Tooltip } from 'chart.js'
import { Bar } from 'vue-chartjs'
import type { ModelMonitorGroupHealthErrorItem } from '@/api/admin/modelMonitorGroupHealth'

ChartJS.register(Title, Tooltip, Legend, BarElement, CategoryScale, LinearScale)

const props = defineProps<{
  errors: ModelMonitorGroupHealthErrorItem[] | null
  loading?: boolean
}>()

const isDarkMode = computed(() => typeof document !== 'undefined' && document.documentElement.classList.contains('dark'))
const textColor = computed(() => isDarkMode.value ? '#9ca3af' : '#64748b')
const errorLabels: Record<string, string> = {
  upstream_rate_limit: '上游限流',
  upstream_error: '上游错误',
  network_timeout: '网络超时',
  account_auth: '账号认证',
  routing: '路由失败',
  business_limited: '业务限制',
  client_request: '客户端请求',
  other: '其他',
}

const normalizedErrors = computed(() => Array.isArray(props.errors) ? props.errors : [])
const totalErrors = computed(() => normalizedErrors.value.reduce((sum, item) => sum + item.count, 0))
const chartData = computed(() => {
  if (!normalizedErrors.value.length) return null
  const errors = [...normalizedErrors.value].sort((a, b) => b.count - a.count)
  return {
    labels: errors.map((item) => errorLabels[item.category] ?? item.category),
    datasets: [{
      label: '错误次数',
      data: errors.map((item) => item.count),
      backgroundColor: ['#fb7185', '#f97316', '#f59e0b', '#a78bfa', '#38bdf8', '#64748b'],
      borderRadius: 7,
      borderSkipped: false,
      barThickness: 13,
    }],
  }
})

const chartOptions = computed(() => ({
  indexAxis: 'y' as const,
  responsive: true,
  maintainAspectRatio: false,
  scales: {
    x: {
      beginAtZero: true,
      grid: { color: isDarkMode.value ? '#263449' : '#e2e8f0', borderDash: [4, 4] },
      ticks: { color: textColor.value, precision: 0, font: { size: 10 } },
    },
    y: {
      grid: { display: false },
      ticks: { color: textColor.value, font: { size: 10 } },
    },
  },
  plugins: {
    legend: { display: false },
    tooltip: {
      displayColors: false,
      callbacks: {
        label: (context: any) => `错误次数：${context.parsed.x}`,
      },
    },
  },
}))
</script>

<style scoped>
.gh-chart-card {
  min-width: 0;
  border: 1px solid var(--gh-line);
  border-radius: 18px;
  background: var(--gh-surface-muted);
  padding: 16px;
}

.gh-chart-heading {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 12px;
}

.gh-chart-kicker {
  margin: 0 0 4px;
  color: var(--gh-muted);
  font-size: 10px;
  font-weight: 700;
  letter-spacing: .12em;
  text-transform: uppercase;
}

h4 {
  margin: 0;
  color: var(--gh-ink);
  font-size: 14px;
  font-weight: 700;
}

.gh-chart-unit {
  border: 1px solid color-mix(in srgb, #fb7185 30%, transparent);
  border-radius: 999px;
  color: #fb7185;
  font-size: 11px;
  font-weight: 700;
  padding: 3px 8px;
}

.gh-chart-stage {
  height: 210px;
  margin-top: 14px;
  position: relative;
}

.gh-chart-empty {
  align-items: center;
  color: var(--gh-muted);
  display: flex;
  height: 100%;
  justify-content: center;
  font-size: 12px;
}
</style>

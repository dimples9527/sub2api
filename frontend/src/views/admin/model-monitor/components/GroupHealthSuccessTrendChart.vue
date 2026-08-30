<template>
  <section class="gh-chart-card" aria-labelledby="success-trend-title">
    <div class="gh-chart-heading">
      <div>
        <p class="gh-chart-kicker">稳定性信号</p>
        <h4 id="success-trend-title">服务健康成功率</h4>
      </div>
      <span class="gh-chart-unit">%</span>
    </div>
    <div class="gh-chart-stage">
      <Line v-if="chartData" :data="chartData" :options="chartOptions" />
      <div v-else class="gh-chart-empty">{{ loading ? '正在读取趋势…' : '暂无足够请求样本' }}</div>
    </div>
  </section>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { CategoryScale, Chart as ChartJS, Filler, Legend, LineElement, LinearScale, PointElement, Title, Tooltip } from 'chart.js'
import { Line } from 'vue-chartjs'
import type { ModelMonitorGroupHealthPoint, ModelMonitorGroupHealthRange } from '@/api/admin/modelMonitorGroupHealth'

ChartJS.register(Title, Tooltip, Legend, LineElement, LinearScale, PointElement, CategoryScale, Filler)

const props = defineProps<{
  points: ModelMonitorGroupHealthPoint[]
  loading?: boolean
  timeRange: ModelMonitorGroupHealthRange
}>()

const isDarkMode = computed(() => typeof document !== 'undefined' && document.documentElement.classList.contains('dark'))
const palette = computed(() => ({
  text: isDarkMode.value ? '#9ca3af' : '#64748b',
  grid: isDarkMode.value ? '#263449' : '#e2e8f0',
  line: '#14b8a6',
  fill: 'rgba(20, 184, 166, 0.14)',
}))

function formatTime(value: string) {
  const date = new Date(value)
  if (Number.isNaN(date.getTime())) return value
  if (props.timeRange === '1h' || props.timeRange === '24h') {
    return date.toLocaleTimeString('zh-CN', { hour: '2-digit', minute: '2-digit' })
  }
  return date.toLocaleDateString('zh-CN', { month: '2-digit', day: '2-digit' })
}

const chartData = computed(() => {
  if (!props.points.length) return null
  return {
    labels: props.points.map((point) => formatTime(point.time)),
    datasets: [{
      label: '服务健康成功率',
      data: props.points.map((point) => point.service_success_rate),
      borderColor: palette.value.line,
      backgroundColor: palette.value.fill,
      fill: true,
      tension: 0.36,
      pointRadius: 0,
      pointHitRadius: 12,
      borderWidth: 2,
    }],
  }
})

const chartOptions = computed(() => ({
  responsive: true,
  maintainAspectRatio: false,
  interaction: { intersect: false, mode: 'index' as const },
  scales: {
    x: {
      grid: { display: false },
      ticks: { color: palette.value.text, maxTicksLimit: 7, font: { size: 10 } },
    },
    y: {
      min: 0,
      max: 100,
      grid: { color: palette.value.grid, borderDash: [4, 4] },
      ticks: {
        color: palette.value.text,
        font: { size: 10 },
        callback: (value: string | number) => `${value}%`,
      },
    },
  },
  plugins: {
    legend: { display: false },
    tooltip: {
      displayColors: false,
      callbacks: {
        label: (context: any) => `健康成功率：${context.parsed.y.toFixed(2)}%`,
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
  border: 1px solid color-mix(in srgb, var(--gh-teal) 30%, transparent);
  border-radius: 999px;
  color: var(--gh-teal);
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


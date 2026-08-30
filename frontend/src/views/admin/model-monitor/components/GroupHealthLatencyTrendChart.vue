<template>
  <section class="gh-chart-card" aria-labelledby="latency-trend-title">
    <div class="gh-chart-heading">
      <div>
        <p class="gh-chart-kicker">响应速度</p>
        <h4 id="latency-trend-title">延迟趋势</h4>
      </div>
      <div class="gh-chart-legend" aria-label="延迟图例">
        <span><i class="gh-dot gh-dot-avg" />平均</span>
        <span><i class="gh-dot gh-dot-p95" />P95</span>
      </div>
    </div>
    <div class="gh-chart-stage">
      <Line v-if="chartData" :data="chartData" :options="chartOptions" />
      <div v-else class="gh-chart-empty">{{ loading ? '正在读取趋势…' : '暂无足够延迟样本' }}</div>
    </div>
  </section>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { CategoryScale, Chart as ChartJS, Legend, LineElement, LinearScale, PointElement, Title, Tooltip } from 'chart.js'
import { Line } from 'vue-chartjs'
import type { ModelMonitorGroupHealthPoint, ModelMonitorGroupHealthRange } from '@/api/admin/modelMonitorGroupHealth'

ChartJS.register(Title, Tooltip, Legend, LineElement, LinearScale, PointElement, CategoryScale)

const props = defineProps<{
  points: ModelMonitorGroupHealthPoint[]
  loading?: boolean
  timeRange: ModelMonitorGroupHealthRange
}>()

const isDarkMode = computed(() => typeof document !== 'undefined' && document.documentElement.classList.contains('dark'))
const palette = computed(() => ({
  text: isDarkMode.value ? '#9ca3af' : '#64748b',
  grid: isDarkMode.value ? '#263449' : '#e2e8f0',
  avg: '#38bdf8',
  p95: '#f59e0b',
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
    datasets: [
      {
        label: '平均延迟',
        data: props.points.map((point) => point.avg_latency_ms),
        borderColor: palette.value.avg,
        backgroundColor: palette.value.avg,
        tension: 0.36,
        pointRadius: 0,
        pointHitRadius: 12,
        borderWidth: 2,
      },
      {
        label: 'P95 延迟',
        data: props.points.map((point) => point.p95_latency_ms),
        borderColor: palette.value.p95,
        backgroundColor: palette.value.p95,
        tension: 0.36,
        pointRadius: 0,
        pointHitRadius: 12,
        borderWidth: 2,
        borderDash: [5, 4],
      },
    ],
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
      beginAtZero: true,
      grid: { color: palette.value.grid, borderDash: [4, 4] },
      ticks: {
        color: palette.value.text,
        font: { size: 10 },
        callback: (value: string | number) => `${value} ms`,
      },
    },
  },
  plugins: {
    legend: { display: false },
    tooltip: {
      displayColors: true,
      callbacks: {
        label: (context: any) => `${context.dataset.label ?? ''}：${context.parsed.y.toFixed(0)} ms`,
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

.gh-chart-legend {
  display: flex;
  flex-wrap: wrap;
  gap: 10px;
  color: var(--gh-muted);
  font-size: 11px;
}

.gh-chart-legend span {
  align-items: center;
  display: inline-flex;
  gap: 5px;
}

.gh-dot {
  border-radius: 50%;
  display: inline-block;
  height: 7px;
  width: 7px;
}

.gh-dot-avg { background: #38bdf8; }
.gh-dot-p95 { background: #f59e0b; }

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


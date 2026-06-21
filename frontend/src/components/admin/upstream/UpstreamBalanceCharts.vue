<template>
  <div class="balance-charts-container">
    <!-- 总览统计卡片 -->
    <div class="stats-overview">
      <div class="stat-card stat-card-primary">
        <div class="stat-icon">
          <Icon name="creditCard" size="md" :stroke-width="2" />
        </div>
        <div class="stat-content">
          <div class="stat-label">总余额</div>
          <div class="stat-value">${{ formatMoney(totalBalance) }}</div>
          <div v-if="totalBalance > 0" class="stat-hint">{{ providerCount }} 个上游</div>
        </div>
        <button
          type="button"
          class="stat-action-btn stat-action-primary"
          title="查看余额详情"
          @click="scrollToBalanceChart"
        >
          <Icon name="chart" size="sm" :stroke-width="2" />
        </button>
      </div>
      <div class="stat-card stat-card-warning">
        <div class="stat-icon">
          <Icon name="arrowDown" size="md" :stroke-width="2" />
        </div>
        <div class="stat-content">
          <div class="stat-label">今日消费</div>
          <div class="stat-value">${{ formatMoney(todayConsumption) }}</div>
          <div v-if="estimatedDaysLeft !== null" class="stat-hint">预计剩余 {{ estimatedDaysLeft }} 天</div>
        </div>
        <button
          type="button"
          class="stat-action-btn stat-action-warning"
          title="查看消费趋势"
          @click="scrollToConsumptionChart"
        >
          <Icon name="trendingUp" size="sm" :stroke-width="2" />
        </button>
      </div>
      <div class="stat-card stat-card-success">
        <div class="stat-icon">
          <Icon name="plus" size="md" :stroke-width="2" />
        </div>
        <div class="stat-content">
          <div class="stat-label">累计充值</div>
          <div class="stat-value">${{ formatMoney(totalRecharge) }}</div>
          <div class="stat-hint">最近 {{ days }} 天</div>
        </div>
        <button
          type="button"
          class="stat-action-btn stat-action-success"
          title="记录充值"
          @click="emit('add-recharge')"
        >
          <Icon name="plus" size="sm" :stroke-width="2" />
        </button>
      </div>
      <div class="stat-card stat-card-danger">
        <div class="stat-icon">
          <Icon name="exclamationTriangle" size="md" :stroke-width="2" />
        </div>
        <div class="stat-content">
          <div class="stat-label">异常提醒</div>
          <div class="stat-value">{{ anomalyCount }}</div>
          <div class="stat-hint">{{ lowBalanceCount }} 个余额不足</div>
        </div>
        <button
          v-if="anomalyCount > 0 || lowBalanceCount > 0"
          type="button"
          class="stat-action-btn stat-action-danger"
          title="查看异常详情"
          @click="emit('show-anomalies')"
        >
          <Icon name="eye" size="sm" :stroke-width="2" />
        </button>
      </div>
    </div>

    <!-- 图表区域 -->
    <div class="charts-grid">
      <!-- 余额分布饼图 -->
      <div ref="balanceChartRef" class="chart-card">
        <div class="chart-header">
          <h3>各上游余额分布</h3>
          <div class="chart-actions">
            <button
              type="button"
              class="chart-toggle"
              :class="{ active: balanceChartView === 'pie' }"
              @click="balanceChartView = 'pie'"
              title="饼图"
            >
              <Icon name="chart" size="sm" :stroke-width="2" />
            </button>
            <button
              type="button"
              class="chart-toggle"
              :class="{ active: balanceChartView === 'bar' }"
              @click="balanceChartView = 'bar'"
              title="柱状图"
            >
              <Icon name="chartBar" size="sm" :stroke-width="2" />
            </button>
          </div>
        </div>
        <div v-if="loading" class="chart-loading">
          <LoadingSpinner />
        </div>
        <div v-else-if="providerBalanceData.length === 0" class="chart-empty">
          <Icon name="inbox" size="xl" :stroke-width="1.5" />
          <span>暂无数据</span>
        </div>
        <div v-else class="chart-content">
          <div v-if="balanceChartView === 'pie'" class="chart-canvas-wrapper">
            <Doughnut :data="balanceChartData" :options="balanceChartOptions" />
          </div>
          <div v-else class="chart-canvas-wrapper chart-canvas-bar">
            <Bar :data="balanceBarChartData" :options="balanceBarChartOptions" />
          </div>
          <div class="chart-legend">
            <div
              v-for="(item, index) in providerBalanceData.slice(0, 8)"
              :key="item.provider_slug"
              class="legend-item"
            >
              <span class="legend-color" :style="{ backgroundColor: chartColors[index] }"></span>
              <span class="legend-label" :title="item.provider_name || item.provider_slug">
                {{ item.provider_name || item.provider_slug }}
              </span>
              <span class="legend-value">${{ formatMoney(item.current_balance) }}</span>
            </div>
            <div v-if="providerBalanceData.length > 8" class="legend-item legend-item-more">
              <span class="legend-color" style="background-color: #94a3b8"></span>
              <span class="legend-label">其他 ({{ providerBalanceData.length - 8 }})</span>
              <span class="legend-value">${{ formatMoney(otherProvidersBalance) }}</span>
            </div>
          </div>
        </div>
      </div>

      <!-- 每日消费趋势图 -->
      <div ref="consumptionChartRef" class="chart-card">
        <div class="chart-header">
          <h3>每日消费趋势</h3>
          <div class="chart-info">最近 {{ days }} 天</div>
        </div>
        <div v-if="loading" class="chart-loading">
          <LoadingSpinner />
        </div>
        <div v-else-if="dailyConsumptionData.length === 0" class="chart-empty">
          <Icon name="inbox" size="xl" :stroke-width="1.5" />
          <span>暂无数据</span>
        </div>
        <div v-else class="chart-content chart-content-full">
          <div class="chart-canvas-wrapper chart-canvas-line">
            <Line :data="consumptionChartData" :options="consumptionChartOptions" />
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, ref } from 'vue'
import { Chart as ChartJS, ArcElement, BarElement, CategoryScale, LinearScale, PointElement, LineElement, Title, Tooltip, Legend } from 'chart.js'
import { Doughnut, Bar, Line } from 'vue-chartjs'
import Icon from '@/components/icons/Icon.vue'
import LoadingSpinner from '@/components/common/LoadingSpinner.vue'
import type { UpstreamBalanceConsumptionOverview, UpstreamBalanceProviderSummary } from '@/api/admin/upstreamAccountSync'

ChartJS.register(ArcElement, BarElement, CategoryScale, LinearScale, PointElement, LineElement, Title, Tooltip, Legend)

const props = withDefaults(defineProps<{
  overview?: UpstreamBalanceConsumptionOverview | null
  loading?: boolean
  days?: number
}>(), {
  overview: null,
  loading: false,
  days: 30
})

const emit = defineEmits<{
  'add-recharge': []
  'show-anomalies': []
}>()

const balanceChartView = ref<'pie' | 'bar'>('pie')
const balanceChartRef = ref<HTMLElement>()
const consumptionChartRef = ref<HTMLElement>()

function scrollToBalanceChart() {
  balanceChartRef.value?.scrollIntoView({ behavior: 'smooth', block: 'center' })
}

function scrollToConsumptionChart() {
  consumptionChartRef.value?.scrollIntoView({ behavior: 'smooth', block: 'center' })
}

const chartColors = [
  '#3b82f6', // blue
  '#10b981', // green
  '#f59e0b', // amber
  '#ef4444', // red
  '#8b5cf6', // violet
  '#ec4899', // pink
  '#14b8a6', // teal
  '#f97316'  // orange
]

// 提取各上游余额数据
const providerBalanceData = computed<UpstreamBalanceProviderSummary[]>(() => {
  if (!props.overview?.summaries) return []
  return Object.values(props.overview.summaries)
    .filter(s => s.current_balance > 0)
    .sort((a, b) => b.current_balance - a.current_balance)
})

// 总余额
const totalBalance = computed(() => {
  return providerBalanceData.value.reduce((sum, item) => sum + item.current_balance, 0)
})

// 今日总消费
const todayConsumption = computed(() => {
  return providerBalanceData.value.reduce((sum, item) => sum + item.today_consumption, 0)
})

// 上游数量
const providerCount = computed(() => providerBalanceData.value.length)

// 异常数量
const anomalyCount = computed(() => {
  return providerBalanceData.value.filter(item => item.anomaly || item.last_snapshot_error).length
})

// 低余额数量 (余额不足今日消费的2倍)
const lowBalanceCount = computed(() => {
  return providerBalanceData.value.filter(item => {
    if (item.today_consumption <= 0) return false
    return item.current_balance < item.today_consumption * 2
  }).length
})

// 预计剩余天数
const estimatedDaysLeft = computed<number | null>(() => {
  if (todayConsumption.value <= 0 || totalBalance.value <= 0) return null
  const days = Math.floor(totalBalance.value / todayConsumption.value)
  return days > 999 ? 999 : days
})

// 累计充值 (最近N天)
const totalRecharge = computed(() => {
  if (!props.overview?.rows) return 0
  return props.overview.rows.reduce((sum, row) => sum + row.recharge_amount, 0)
})

// 其他上游总余额 (排名8之后的)
const otherProvidersBalance = computed(() => {
  if (providerBalanceData.value.length <= 8) return 0
  return providerBalanceData.value.slice(8).reduce((sum, item) => sum + item.current_balance, 0)
})

// 余额饼图数据
const balanceChartData = computed(() => {
  const topProviders = providerBalanceData.value.slice(0, 8)
  const labels = topProviders.map(p => p.provider_name || p.provider_slug)
  const data = topProviders.map(p => p.current_balance)

  if (otherProvidersBalance.value > 0) {
    labels.push('其他')
    data.push(otherProvidersBalance.value)
  }

  return {
    labels,
    datasets: [{
      data,
      backgroundColor: [...chartColors.slice(0, topProviders.length), '#94a3b8'],
      borderWidth: 0
    }]
  }
})

const balanceChartOptions = computed(() => ({
  responsive: true,
  maintainAspectRatio: false,
  plugins: {
    legend: { display: false },
    tooltip: {
      callbacks: {
        label: (context: any) => {
          const value = context.raw as number
          const total = context.dataset.data.reduce((a: number, b: number) => a + b, 0)
          const percentage = total > 0 ? ((value / total) * 100).toFixed(1) : '0.0'
          return `${context.label}: $${formatMoney(value)} (${percentage}%)`
        }
      }
    }
  }
}))

// 余额柱状图数据
const balanceBarChartData = computed(() => {
  const topProviders = providerBalanceData.value.slice(0, 10)
  return {
    labels: topProviders.map(p => p.provider_name || p.provider_slug),
    datasets: [{
      label: '余额 ($)',
      data: topProviders.map(p => p.current_balance),
      backgroundColor: chartColors.slice(0, topProviders.length),
      borderWidth: 0
    }]
  }
})

const balanceBarChartOptions = computed(() => ({
  responsive: true,
  maintainAspectRatio: false,
  plugins: {
    legend: { display: false },
    tooltip: {
      callbacks: {
        label: (context: any) => `余额: $${formatMoney(context.raw)}`
      }
    }
  },
  scales: {
    y: {
      beginAtZero: true,
      ticks: {
        callback: (value: any) => `$${formatMoney(value)}`
      }
    }
  }
}))

// 每日消费数据 (按日期聚合所有上游)
const dailyConsumptionData = computed<Array<{ date: string; consumption: number; recharge: number; localConsumption: number }>>(() => {
  if (!props.overview) return []

  const dateMap = new Map<string, { consumption: number; recharge: number; localConsumption: number }>()

  props.overview.rows?.forEach(row => {
    if (!row.complete) return // 只统计完整的数据

    const existing = dateMap.get(row.date) || { consumption: 0, recharge: 0, localConsumption: 0 }
    existing.consumption += row.consumption_amount
    existing.recharge += row.recharge_amount
    dateMap.set(row.date, existing)
  })

  props.overview.local_daily_consumptions?.forEach(row => {
    const existing = dateMap.get(row.date) || { consumption: 0, recharge: 0, localConsumption: 0 }
    existing.localConsumption += row.actual_cost
    dateMap.set(row.date, existing)
  })

  return Array.from(dateMap.entries())
    .map(([date, values]) => ({ date, ...values }))
    .sort((a, b) => a.date.localeCompare(b.date))
})

const consumptionChartData = computed(() => ({
  labels: dailyConsumptionData.value.map(d => {
    const date = new Date(d.date)
    return `${date.getMonth() + 1}/${date.getDate()}`
  }),
  datasets: [
    {
      label: '消费',
      data: dailyConsumptionData.value.map(d => d.consumption),
      borderColor: '#ef4444',
      backgroundColor: 'rgba(239, 68, 68, 0.1)',
      tension: 0.3,
      fill: true
    },
    {
      label: '充值',
      data: dailyConsumptionData.value.map(d => d.recharge),
      borderColor: '#10b981',
      backgroundColor: 'rgba(16, 185, 129, 0.1)',
      tension: 0.3,
      fill: true
    },
    {
      label: '本地消费',
      data: dailyConsumptionData.value.map(d => d.localConsumption),
      borderColor: '#2563eb',
      backgroundColor: 'rgba(37, 99, 235, 0.08)',
      tension: 0.3,
      fill: false
    }
  ]
}))

const consumptionChartOptions = computed(() => ({
  responsive: true,
  maintainAspectRatio: false,
  interaction: {
    mode: 'index' as const,
    intersect: false
  },
  plugins: {
    legend: {
      display: true,
      position: 'top' as const,
      align: 'end' as const,
      labels: {
        boxWidth: 12,
        boxHeight: 12,
        padding: 10,
        font: { size: 11 }
      }
    },
    tooltip: {
      callbacks: {
        label: (context: any) => `${context.dataset.label}: $${formatMoney(context.raw)}`
      }
    }
  },
  scales: {
    y: {
      beginAtZero: true,
      ticks: {
        callback: (value: any) => `$${formatMoney(value)}`
      }
    }
  }
}))

function formatMoney(value: number): string {
  if (value >= 1000000) {
    return (value / 1000000).toFixed(2) + 'M'
  } else if (value >= 1000) {
    return (value / 1000).toFixed(2) + 'K'
  } else if (value >= 1) {
    return value.toFixed(2)
  } else if (value >= 0.01) {
    return value.toFixed(3)
  }
  return value.toFixed(4)
}
</script>

<style scoped>
.balance-charts-container {
  display: grid;
  gap: 16px;
}

.stats-overview {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(240px, 1fr));
  gap: 12px;
}

.stat-card {
  position: relative;
  display: flex;
  align-items: center;
  gap: 14px;
  border: 1px solid #e5e7eb;
  border-radius: 8px;
  background: #fff;
  padding: 16px;
  box-shadow: 0 4px 12px rgba(15, 23, 42, 0.04);
  transition: box-shadow 150ms ease, border-color 150ms ease;
}

.stat-card:hover {
  box-shadow: 0 8px 24px rgba(15, 23, 42, 0.08);
}

.stat-icon {
  display: grid;
  width: 44px;
  height: 44px;
  flex: none;
  place-items: center;
  border-radius: 10px;
}

.stat-card-primary .stat-icon {
  background: #eff6ff;
  color: #2563eb;
}

.stat-card-warning .stat-icon {
  background: #fff7ed;
  color: #ea580c;
}

.stat-card-success .stat-icon {
  background: #ecfdf5;
  color: #059669;
}

.stat-card-danger .stat-icon {
  background: #fef2f2;
  color: #dc2626;
}

.stat-content {
  display: flex;
  min-width: 0;
  flex: 1;
  flex-direction: column;
  gap: 2px;
}

.stat-label {
  color: #64748b;
  font-size: 12px;
  font-weight: 600;
}

.stat-value {
  color: #111827;
  font-size: 24px;
  font-weight: 750;
  line-height: 1.2;
}

.stat-hint {
  margin-top: 2px;
  color: #94a3b8;
  font-size: 11px;
}

.stat-action-btn {
  display: grid;
  width: 36px;
  height: 36px;
  flex: none;
  place-items: center;
  border: 1px solid;
  border-radius: 8px;
  background: #fff;
  cursor: pointer;
  transition: all 150ms ease;
}

.stat-action-btn:hover {
  transform: translateY(-2px);
  box-shadow: 0 4px 12px rgba(0, 0, 0, 0.12);
}

.stat-action-btn:active {
  transform: translateY(0);
}

.stat-action-primary {
  border-color: #bfdbfe;
  color: #2563eb;
}

.stat-action-primary:hover {
  border-color: #2563eb;
  background: #eff6ff;
}

.stat-action-warning {
  border-color: #fed7aa;
  color: #ea580c;
}

.stat-action-warning:hover {
  border-color: #ea580c;
  background: #fff7ed;
}

.stat-action-success {
  border-color: #a7f3d0;
  color: #059669;
}

.stat-action-success:hover {
  border-color: #059669;
  background: #ecfdf5;
}

.stat-action-danger {
  border-color: #fecaca;
  color: #dc2626;
}

.stat-action-danger:hover {
  border-color: #dc2626;
  background: #fef2f2;
}

.charts-grid {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(400px, 1fr));
  gap: 16px;
}

.chart-card {
  display: flex;
  min-height: 320px;
  flex-direction: column;
  border: 1px solid #e5e7eb;
  border-radius: 8px;
  background: #fff;
  padding: 16px;
  box-shadow: 0 4px 12px rgba(15, 23, 42, 0.04);
}

.chart-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  margin-bottom: 16px;
}

.chart-header h3 {
  margin: 0;
  color: #111827;
  font-size: 14px;
  font-weight: 700;
}

.chart-info {
  color: #64748b;
  font-size: 12px;
}

.chart-actions {
  display: flex;
  gap: 4px;
}

.chart-toggle {
  display: grid;
  width: 28px;
  height: 28px;
  place-items: center;
  border: 1px solid #e5e7eb;
  border-radius: 6px;
  background: #fff;
  color: #64748b;
  cursor: pointer;
  transition: all 150ms ease;
}

.chart-toggle:hover {
  border-color: #2563eb;
  color: #2563eb;
}

.chart-toggle.active {
  border-color: #2563eb;
  background: #eff6ff;
  color: #2563eb;
}

.chart-loading,
.chart-empty {
  display: grid;
  flex: 1;
  place-items: center;
  color: #94a3b8;
  font-size: 13px;
  gap: 8px;
}

.chart-content {
  display: flex;
  flex: 1;
  align-items: center;
  gap: 16px;
}

.chart-content-full {
  display: block;
}

.chart-canvas-wrapper {
  position: relative;
  width: 180px;
  height: 180px;
  flex: none;
}

.chart-canvas-bar,
.chart-canvas-line {
  width: 100%;
  height: 240px;
}

.chart-legend {
  display: flex;
  flex: 1;
  flex-direction: column;
  gap: 8px;
  overflow-y: auto;
  max-height: 220px;
}

.legend-item {
  display: flex;
  align-items: center;
  gap: 8px;
  font-size: 12px;
}

.legend-item-more {
  margin-top: 4px;
  padding-top: 8px;
  border-top: 1px solid #e5e7eb;
}

.legend-color {
  width: 12px;
  height: 12px;
  flex: none;
  border-radius: 2px;
}

.legend-label {
  overflow: hidden;
  flex: 1;
  color: #334155;
  font-weight: 500;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.legend-value {
  color: #64748b;
  font-variant-numeric: tabular-nums;
  font-weight: 650;
}

@media (max-width: 768px) {
  .stats-overview {
    grid-template-columns: 1fr;
  }

  .charts-grid {
    grid-template-columns: 1fr;
  }

  .chart-content {
    flex-direction: column;
  }

  .chart-canvas-wrapper {
    width: 100%;
  }

  .chart-legend {
    max-height: none;
  }
}
</style>

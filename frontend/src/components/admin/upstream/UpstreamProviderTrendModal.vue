<template>
  <div v-if="show" class="trend-modal-overlay" @click.self="$emit('close')">
    <div class="trend-modal">
      <div class="trend-modal-header">
        <div class="trend-modal-title">
          <Icon name="chart" size="md" :stroke-width="2" />
          <h3>{{ providerName || providerSlug }} - 趋势分析</h3>
        </div>
        <button type="button" class="trend-modal-close" @click="$emit('close')">
          <Icon name="x" size="md" :stroke-width="2" />
        </button>
      </div>

      <div class="trend-modal-body">
        <div v-if="loading" class="trend-loading">
          <LoadingSpinner />
          <span>加载数据中...</span>
        </div>

        <div v-else-if="error" class="trend-error">
          <Icon name="exclamationCircle" size="xl" :stroke-width="1.5" />
          <span>{{ error }}</span>
        </div>

        <div v-else class="trend-content">
          <!-- 统计摘要 -->
          <div class="trend-summary">
            <div class="summary-item">
              <span class="summary-label">当前余额</span>
              <span class="summary-value summary-value-primary">${{ formatMoney(currentBalance) }}</span>
            </div>
            <div class="summary-item">
              <span class="summary-label">今日消费</span>
              <span class="summary-value summary-value-danger">${{ formatMoney(todayConsumption) }}</span>
            </div>
            <div class="summary-item">
              <span class="summary-label">平均日消费</span>
              <span class="summary-value summary-value-warning">${{ formatMoney(avgDailyConsumption) }}</span>
            </div>
            <div class="summary-item">
              <span class="summary-label">累计充值</span>
              <span class="summary-value summary-value-success">${{ formatMoney(totalRecharge) }}</span>
            </div>
          </div>

          <!-- 余额趋势图 -->
          <div class="trend-chart-section">
            <h4 class="trend-chart-title">余额变化趋势</h4>
            <div class="trend-chart-wrapper">
              <Line :data="balanceChartData" :options="balanceChartOptions" />
            </div>
          </div>

          <!-- 消费趋势图 -->
          <div class="trend-chart-section">
            <h4 class="trend-chart-title">每日消费趋势</h4>
            <div class="trend-chart-wrapper">
              <Bar :data="consumptionChartData" :options="consumptionChartOptions" />
            </div>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, watch } from 'vue'
import { Chart as ChartJS, CategoryScale, LinearScale, PointElement, LineElement, BarElement, Title, Tooltip, Legend, Filler } from 'chart.js'
import { Line, Bar } from 'vue-chartjs'
import Icon from '@/components/icons/Icon.vue'
import LoadingSpinner from '@/components/common/LoadingSpinner.vue'
import type { UpstreamBalanceDailyRow } from '@/api/admin/upstreamAccountSync'

ChartJS.register(CategoryScale, LinearScale, PointElement, LineElement, BarElement, Title, Tooltip, Legend, Filler)

const props = withDefaults(defineProps<{
  show: boolean
  providerSlug: string
  providerName?: string
  rows: UpstreamBalanceDailyRow[]
  loading?: boolean
  error?: string
}>(), {
  loading: false,
  error: ''
})

defineEmits<{
  'close': []
}>()

// 筛选当前上游的数据
const providerRows = computed(() => {
  return props.rows
    .filter(row => row.provider_slug === props.providerSlug && row.complete)
    .sort((a, b) => a.date.localeCompare(b.date))
})

const currentBalance = computed(() => {
  if (!providerRows.value.length) return 0
  return providerRows.value[providerRows.value.length - 1].current_balance
})

const todayConsumption = computed(() => {
  if (!providerRows.value.length) return 0
  return providerRows.value[providerRows.value.length - 1].consumption_amount
})

const avgDailyConsumption = computed(() => {
  if (!providerRows.value.length) return 0
  const total = providerRows.value.reduce((sum, row) => sum + row.consumption_amount, 0)
  return total / providerRows.value.length
})

const totalRecharge = computed(() => {
  if (!providerRows.value.length) return 0
  return providerRows.value.reduce((sum, row) => sum + row.recharge_amount, 0)
})

// 余额趋势图数据
const balanceChartData = computed(() => ({
  labels: providerRows.value.map(row => {
    const date = new Date(row.date)
    return `${date.getMonth() + 1}/${date.getDate()}`
  }),
  datasets: [
    {
      label: '期初余额',
      data: providerRows.value.map(row => row.opening_balance),
      borderColor: '#94a3b8',
      backgroundColor: 'rgba(148, 163, 184, 0.1)',
      borderWidth: 1,
      borderDash: [5, 5],
      tension: 0.3,
      fill: false,
      pointRadius: 2
    },
    {
      label: '期末余额',
      data: providerRows.value.map(row => row.closing_balance),
      borderColor: '#2563eb',
      backgroundColor: 'rgba(37, 99, 235, 0.1)',
      borderWidth: 2,
      tension: 0.3,
      fill: true,
      pointRadius: 3,
      pointHoverRadius: 5
    }
  ]
}))

const balanceChartOptions = computed(() => ({
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
      beginAtZero: false,
      ticks: {
        callback: (value: any) => `$${formatMoney(value)}`
      }
    }
  }
}))

// 消费趋势图数据
const consumptionChartData = computed(() => ({
  labels: providerRows.value.map(row => {
    const date = new Date(row.date)
    return `${date.getMonth() + 1}/${date.getDate()}`
  }),
  datasets: [
    {
      label: '消费',
      data: providerRows.value.map(row => row.consumption_amount),
      backgroundColor: '#ef4444',
      borderWidth: 0
    },
    {
      label: '充值',
      data: providerRows.value.map(row => row.recharge_amount),
      backgroundColor: '#10b981',
      borderWidth: 0
    }
  ]
}))

const consumptionChartOptions = computed(() => ({
  responsive: true,
  maintainAspectRatio: false,
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
      stacked: false,
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

// 监听显示状态，自动滚动到顶部
watch(() => props.show, (show) => {
  if (show) {
    document.body.style.overflow = 'hidden'
  } else {
    document.body.style.overflow = ''
  }
}, { immediate: true })
</script>

<style scoped>
.trend-modal-overlay {
  position: fixed;
  top: 0;
  right: 0;
  bottom: 0;
  left: 0;
  z-index: 1000;
  display: flex;
  align-items: center;
  justify-content: center;
  background: rgba(0, 0, 0, 0.5);
  padding: 16px;
  backdrop-filter: blur(4px);
}

.trend-modal {
  display: flex;
  width: 100%;
  max-width: 900px;
  max-height: 90vh;
  flex-direction: column;
  overflow: hidden;
  border-radius: 12px;
  background: #fff;
  box-shadow: 0 20px 60px rgba(0, 0, 0, 0.3);
}

.trend-modal-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  border-bottom: 1px solid #e5e7eb;
  padding: 20px 24px;
}

.trend-modal-title {
  display: flex;
  align-items: center;
  gap: 12px;
  color: #2563eb;
}

.trend-modal-title h3 {
  margin: 0;
  color: #111827;
  font-size: 18px;
  font-weight: 700;
}

.trend-modal-close {
  display: grid;
  width: 32px;
  height: 32px;
  place-items: center;
  border: 0;
  border-radius: 6px;
  background: transparent;
  color: #64748b;
  cursor: pointer;
  transition: all 150ms ease;
}

.trend-modal-close:hover {
  background: #f1f5f9;
  color: #111827;
}

.trend-modal-body {
  flex: 1;
  overflow-y: auto;
  padding: 24px;
}

.trend-loading,
.trend-error {
  display: grid;
  min-height: 300px;
  place-items: center;
  gap: 12px;
  color: #94a3b8;
  font-size: 14px;
  text-align: center;
}

.trend-error {
  color: #ef4444;
}

.trend-summary {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(180px, 1fr));
  gap: 16px;
  margin-bottom: 32px;
}

.summary-item {
  display: flex;
  flex-direction: column;
  gap: 8px;
  border: 1px solid #e5e7eb;
  border-radius: 8px;
  padding: 16px;
  background: #f9fafb;
}

.summary-label {
  color: #64748b;
  font-size: 12px;
  font-weight: 600;
  text-transform: uppercase;
}

.summary-value {
  font-size: 24px;
  font-variant-numeric: tabular-nums;
  font-weight: 750;
  line-height: 1.2;
}

.summary-value-primary {
  color: #2563eb;
}

.summary-value-danger {
  color: #ef4444;
}

.summary-value-warning {
  color: #ea580c;
}

.summary-value-success {
  color: #059669;
}

.trend-chart-section {
  margin-bottom: 32px;
}

.trend-chart-section:last-child {
  margin-bottom: 0;
}

.trend-chart-title {
  margin: 0 0 16px;
  color: #111827;
  font-size: 14px;
  font-weight: 700;
}

.trend-chart-wrapper {
  position: relative;
  height: 280px;
  border: 1px solid #e5e7eb;
  border-radius: 8px;
  background: #fff;
  padding: 16px;
}

@media (max-width: 768px) {
  .trend-modal {
    max-width: 100%;
    max-height: 100vh;
    border-radius: 0;
  }

  .trend-summary {
    grid-template-columns: 1fr;
  }

  .trend-chart-wrapper {
    height: 240px;
  }
}
</style>

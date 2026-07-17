import { mount } from '@vue/test-utils'
import { describe, expect, it, vi } from 'vitest'
import UpstreamBalanceCharts from './UpstreamBalanceCharts.vue'

const renderedCharts: Array<{ data: any }> = []

vi.mock('vue-chartjs', () => ({
  Doughnut: {
    name: 'Doughnut',
    props: ['data', 'options'],
    template: '<div />'
  },
  Bar: {
    name: 'Bar',
    props: ['data', 'options'],
    template: '<div />'
  },
  Line: {
    name: 'Line',
    props: ['data', 'options'],
    setup(props: any) {
      renderedCharts.push(props)
      return {}
    },
    template: '<div data-test="line-chart" />'
  }
}))

vi.mock('chart.js', () => ({
  Chart: {
    register: vi.fn()
  },
  ArcElement: {},
  BarElement: {},
  CategoryScale: {},
  LinearScale: {},
  PointElement: {},
  LineElement: {},
  Title: {},
  Tooltip: {},
  Legend: {}
}))

vi.mock('@/components/icons/Icon.vue', () => ({
  default: {
    name: 'Icon',
    template: '<span />'
  }
}))

vi.mock('@/components/common/LoadingSpinner.vue', () => ({
  default: {
    name: 'LoadingSpinner',
    template: '<span />'
  }
}))

describe('UpstreamBalanceCharts', () => {
  it('adds local daily consumption to the daily consumption trend chart', () => {
    renderedCharts.length = 0

    mount(UpstreamBalanceCharts, {
      props: {
        days: 2,
        overview: {
          config: { enabled: true, interval_seconds: 3600 },
          summaries: {},
          snapshots: [],
          rows: [
            {
              provider_slug: 'sub-main',
              date: '2026-06-16',
              amount_scale: 1,
              opening_balance: 100,
              closing_balance: 80,
              current_balance: 80,
              recharge_amount: 5,
              consumption_amount: 20,
              snapshot_count: 2,
              complete: true,
              anomaly: false
            },
            {
              provider_slug: 'sub-main',
              date: '2026-06-17',
              amount_scale: 1,
              opening_balance: 80,
              closing_balance: 70,
              current_balance: 70,
              recharge_amount: 0,
              consumption_amount: 10,
              snapshot_count: 2,
              complete: true,
              anomaly: false
            }
          ],
          local_daily_consumptions: [
            { date: '2026-06-16', actual_cost: 7.5 },
            { date: '2026-06-17', actual_cost: 9.25 }
          ]
        }
      }
    })

    const lineChart = renderedCharts[0]
    expect(lineChart.data.datasets).toEqual(
      expect.arrayContaining([
        expect.objectContaining({
          label: '本地消费',
          data: [7.5, 9.25]
        })
      ])
    )
  })
})

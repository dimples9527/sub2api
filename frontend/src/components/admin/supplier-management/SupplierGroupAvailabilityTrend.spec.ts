import { mount } from '@vue/test-utils'
import { describe, expect, it } from 'vitest'
import SupplierGroupAvailabilityTrend from './SupplierGroupAvailabilityTrend.vue'

describe('SupplierGroupAvailabilityTrend', () => {
  it('展示可用率、检查时间和全部趋势点', () => {
    const wrapper = mount(SupplierGroupAvailabilityTrend, {
      props: {
        label: '可用率趋势',
        row: {
          provider: 'VIP', availability: 66.67, latency: 120, time: '10:30',
          trend: [
            { tone: 'red', statusText: 'Down', time: '10:00', latency: 320, availability: 0 },
            { tone: 'green', statusText: 'OK', time: '10:30', latency: 120, availability: 100 },
          ],
        },
      },
    })

    expect(wrapper.text()).toContain('66.67%')
    expect(wrapper.text()).toContain('10:30')
    expect(wrapper.findAll('.supplier-group-trend__bar')).toHaveLength(2)
    expect(wrapper.find('.supplier-group-trend__bar--green').exists()).toBe(true)
  })

  it('在加载和无数据时分别展示加载文案与空状态', async () => {
    const wrapper = mount(SupplierGroupAvailabilityTrend, {
      props: { loading: true, loadingText: '加载中', emptyText: '-' },
    })
    expect(wrapper.text()).toContain('加载中')
    expect(wrapper.findAll('.supplier-group-trend__bar--loading')).toHaveLength(18)
    await wrapper.setProps({ loading: false })
    expect(wrapper.text()).toContain('-')
  })
})

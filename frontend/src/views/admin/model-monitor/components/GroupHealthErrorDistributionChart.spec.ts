import { mount } from '@vue/test-utils'
import { describe, expect, it } from 'vitest'

import GroupHealthErrorDistributionChart from './GroupHealthErrorDistributionChart.vue'

describe('GroupHealthErrorDistributionChart', () => {
  it('接口返回空错误列表时仍能渲染空状态', () => {
    const wrapper = mount(GroupHealthErrorDistributionChart, {
      props: { errors: null },
      global: {
        stubs: {
          Bar: { template: '<div data-test="bar-chart" />' },
        },
      },
    })

    expect(wrapper.text()).toContain('当前范围暂无错误记录')
    expect(wrapper.text()).toContain('0 次')
  })
})

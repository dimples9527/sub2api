import { mount } from '@vue/test-utils'
import { describe, expect, it, vi } from 'vitest'

import AmountInput from './AmountInput.vue'

vi.mock('vue-i18n', () => ({
  useI18n: () => ({
    t: (key: string, params?: Record<string, unknown>) => {
      if (key === 'payment.rechargePayAmount') return '充值金额'
      if (key === 'payment.rechargeCreditAmount') return '到账余额'
      if (key === 'payment.quickAmounts') return '快捷金额'
      if (key === 'payment.customAmount') return '自定义金额'
      if (key === 'payment.enterAmount') return '输入金额'
      if (params) return `${key}:${JSON.stringify(params)}`
      return key
    },
  }),
}))

describe('AmountInput', () => {
  it('renders recharge amount options and emits selected pay amount', async () => {
    const wrapper = mount(AmountInput, {
      props: {
        modelValue: null,
        options: [
          { pay_amount: 2, credit_amount: 2 },
          { pay_amount: 5, credit_amount: 5 },
        ],
      },
    })

    expect(wrapper.text()).toContain('充值金额')
    expect(wrapper.text()).toContain('到账余额')
    expect(wrapper.text()).toContain('$2')
    expect(wrapper.text()).toContain('$5')

    await wrapper.find('button[data-testid="amount-option-2"]').trigger('click')

    expect(wrapper.emitted('update:modelValue')?.[0]).toEqual([2])
  })

  it('filters quick options by min and max amount', () => {
    const wrapper = mount(AmountInput, {
      props: {
        modelValue: null,
        min: 5,
        max: 20,
        options: [
          { pay_amount: 2, credit_amount: 2 },
          { pay_amount: 5, credit_amount: 5 },
          { pay_amount: 50, credit_amount: 50 },
        ],
      },
    })

    expect(wrapper.text()).not.toContain('$2')
    expect(wrapper.text()).toContain('$5')
    expect(wrapper.text()).not.toContain('$50')
  })
})

import { mount } from '@vue/test-utils'
import { describe, expect, it, vi } from 'vitest'
import UpstreamBalanceSamplerDialog from './UpstreamBalanceSamplerDialog.vue'

vi.mock('vue-i18n', () => ({ useI18n: () => ({ t: (key: string) => key }) }))
vi.mock('@/components/common/BaseDialog.vue', () => ({ default: { props: ['show'], template: '<div v-if="show"><slot /><slot name="footer" /></div>' } }))

describe('UpstreamBalanceSamplerDialog', () => {
  it('emits configuration updates and actions', async () => {
    const wrapper = mount(UpstreamBalanceSamplerDialog, { props: { show: true, enabled: false, intervalSeconds: 3600, providerAmountScales: { main: 1 }, providers: [{ slug: 'main', name: 'Main' }], defaultScales: { main: 2 }, saving: false } })
    await wrapper.get('[data-test="balance-sampler-enabled"]').setValue(true)
    await wrapper.get('[data-test="balance-sampler-interval"]').setValue('1800')
    await wrapper.get('[data-test="balance-sampler-scale-main"]').setValue('3')
    await wrapper.get('[data-test="balance-sampler-cancel"]').trigger('click')
    await wrapper.get('[data-test="balance-sampler-save"]').trigger('click')
    expect(wrapper.emitted('update:enabled')).toEqual([[true]])
    expect(wrapper.emitted('update:intervalSeconds')).toEqual([[1800]])
    expect(wrapper.emitted('update:providerScale')).toEqual([['main', 3]])
    expect(wrapper.emitted('close')).toHaveLength(1)
    expect(wrapper.emitted('save')).toHaveLength(1)
  })
})

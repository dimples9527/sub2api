import { mount } from '@vue/test-utils'
import { describe, expect, it, vi } from 'vitest'
import UpstreamHealthGuardPolicyFields from './UpstreamHealthGuardPolicyFields.vue'
vi.mock('vue-i18n', () => ({ useI18n: () => ({ t: (key: string) => key }) }))
describe('UpstreamHealthGuardPolicyFields', () => {
  it('emits field updates and ignored-account management', async () => {
    const wrapper = mount(UpstreamHealthGuardPolicyFields, { props: { values: { interval_seconds: 3600, max_accounts_per_run: 100, concurrency: 4, timeout_per_account_seconds: 30, failure_threshold: 3, slow_threshold: 2, recovery_threshold: 2, healthy_latency_ms: 5000 }, ignoredSummaryText: '2 accounts', ignoredInputInvalid: false } })
    await wrapper.get('[data-test="health-guard-field-interval_seconds"]').setValue('1800')
    await wrapper.get('[data-test="health-guard-ignored-manage"]').trigger('click')
    expect(wrapper.emitted('update:field')).toEqual([['interval_seconds', 1800]])
    expect(wrapper.emitted('manage-ignored')).toHaveLength(1)
  })
})

import { mount } from '@vue/test-utils'
import { describe, expect, it, vi } from 'vitest'
import UpstreamAccountRateGuardPanel from './UpstreamAccountRateGuardPanel.vue'

vi.mock('vue-i18n', () => ({
  useI18n: () => ({
    t: (key: string, params?: Record<string, unknown> | string) => {
      if (key.endsWith('rateGuardIgnoredSummary') && typeof params === 'object') {
        return `ignored:${params.count}`
      }
      return typeof params === 'string' ? params : key
    },
  }),
}))

vi.mock('@/components/icons/Icon.vue', () => ({
  default: {
    name: 'Icon',
    template: '<span />',
  },
}))

function mountPanel(overrides: Record<string, unknown> = {}) {
  return mount(UpstreamAccountRateGuardPanel, {
    props: {
      enabled: true,
      intervalSeconds: 3600,
      config: {
        enabled: true,
        interval_seconds: 3600,
        last_run_status: 'failed',
        last_run_message: 'rate mismatch',
      },
      lastRunText: '2026-07-13 10:00',
      dailyRunsText: '24 runs/day',
      ignoredCount: 2,
      ignoredSummaryText: 'key-a, key-b',
      ignoredInputInvalid: false,
      loading: false,
      saving: false,
      running: false,
      unhandledSyncLogCount: 3,
      automationTarget: true,
      ...overrides,
    },
  })
}

describe('UpstreamAccountRateGuardPanel', () => {
  it('renders the current status and keeps the automation target marker', () => {
    const wrapper = mountPanel()

    expect(wrapper.get('[data-test=rate-guard-panel]').classes()).toContain('is-automation-target')
    expect(wrapper.text()).toContain('2026-07-13 10:00')
    expect(wrapper.text()).toContain('rate mismatch')
    expect(wrapper.text()).toContain('key-a, key-b')
    expect(wrapper.text()).toContain('3')
  })

  it('emits form updates and all panel actions', async () => {
    const wrapper = mountPanel()

    await wrapper.get('[data-test=rate-guard-enabled]').setValue(false)
    await wrapper.get('[data-test=rate-guard-interval]').setValue('1800')
    await wrapper.get('[data-test=rate-guard-ignored-manage]').trigger('click')
    await wrapper.get('[data-test=rate-guard-open-logs]').trigger('click')
    await wrapper.get('[data-test=rate-guard-save]').trigger('click')
    await wrapper.get('[data-test=rate-guard-run]').trigger('click')

    expect(wrapper.emitted('update:enabled')).toEqual([[false]])
    expect(wrapper.emitted('update:intervalSeconds')).toEqual([[1800]])
    expect(wrapper.emitted('manage-ignored')).toHaveLength(1)
    expect(wrapper.emitted('open-logs')).toHaveLength(1)
    expect(wrapper.emitted('save')).toHaveLength(1)
    expect(wrapper.emitted('run')).toHaveLength(1)
  })
})

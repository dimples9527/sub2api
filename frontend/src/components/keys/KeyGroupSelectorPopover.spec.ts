import { mount } from '@vue/test-utils'
import { describe, expect, it, vi } from 'vitest'
import KeyGroupSelectorPopover from './KeyGroupSelectorPopover.vue'
import type { KeyGroupSelectorOption } from './KeyGroupSelectorPopover.vue'

vi.mock('vue-i18n', () => ({
  createI18n: () => ({
    install: vi.fn(),
    global: { t: (key: string) => key, locale: { value: 'zh-CN' }, setLocaleMessage: vi.fn() },
  }),
  useI18n: () => ({ t: (key: string) => key }),
}))

const options: KeyGroupSelectorOption[] = [
  {
    value: 1,
    label: 'OpenAI 分组',
    description: 'openai group',
    rate: 1,
    userRate: null,
    peakRateEnabled: false,
    peakStart: '',
    peakEnd: '',
    peakRateMultiplier: 1,
    subscriptionType: 'standard',
    platform: 'openai',
  },
  {
    value: 2,
    label: 'Gemini 分组',
    description: 'gemini group',
    rate: 1,
    userRate: null,
    peakRateEnabled: false,
    peakStart: '',
    peakEnd: '',
    peakRateMultiplier: 1,
    subscriptionType: 'standard',
    platform: 'gemini',
  },
]

function mountPopover(popoverOptions = options) {
  return mount(KeyGroupSelectorPopover, {
    props: {
      open: true,
      activeKeyId: 1,
      position: { top: 20, left: 30 },
      options: popoverOptions,
      selectedGroupId: 1,
    },
    global: {
      stubs: {
        Teleport: true,
        GroupOptionItem: {
          props: ['name'],
          template: '<span data-test="group-option">{{ name }}</span>',
        },
      },
    },
  })
}

describe('KeyGroupSelectorPopover', () => {
  it('renders the same provider cards as the create-key dialog', () => {
    const wrapper = mountPopover()

    const radios = wrapper.findAll('input[name="key-list-group-provider"]')
    expect(radios.map((radio) => (radio.element as HTMLInputElement).value)).toEqual([
      'anthropic',
      'openai',
      'domestic',
      'other',
    ])

    // 厂商卡片是单选，没有「全部」项，默认落在第一个有分组的厂商上
    expect((wrapper.get('input[value="openai"]').element as HTMLInputElement).checked).toBe(true)
    expect(
      wrapper.findAll('[data-test="group-option"]').map((item) => item.text())
    ).toEqual(['OpenAI 分组'])
  })

  it('filters groups when switching provider', async () => {
    const wrapper = mountPopover()

    await wrapper.get('input[value="other"]').setValue()

    expect(
      wrapper.findAll('[data-test="group-option"]').map((item) => item.text())
    ).toEqual(['Gemini 分组'])
  })

  it('classifies groups by resolved business platform rather than the raw platform field', async () => {
    const wrapper = mountPopover([
      ...options,
      {
        value: 3,
        label: '智谱分组',
        description: 'custom platform group',
        rate: 0.8,
        userRate: null,
        peakRateEnabled: false,
        peakStart: '',
        peakEnd: '',
        peakRateMultiplier: 1,
        subscriptionType: 'standard',
        platform: 'composite',
        businessPlatform: 'zhipu',
        businessPlatformName: '智谱 GLM',
      },
    ])

    // 原始 platform 是 composite（会归入 other），解析后的业务平台 zhipu 才归入 domestic
    await wrapper.get('input[value="domestic"]').setValue()

    expect(
      wrapper.findAll('[data-test="group-option"]').map((item) => item.text())
    ).toEqual(['智谱分组'])
  })
})

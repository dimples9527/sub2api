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

function mountPopover(popoverOptions = options, emitValue = 'gemini') {
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
        Select: {
          name: 'Select',
          props: ['modelValue', 'options', 'placeholder', 'ariaLabel'],
          emits: ['update:modelValue'],
          template: `<button data-test="platform-select" @click="$emit('update:modelValue', '${emitValue}')">{{ placeholder }}</button>`,
        },
        GroupOptionItem: {
          props: ['name'],
          template: '<span data-test="group-option">{{ name }}</span>',
        },
      },
    },
  })
}

describe('KeyGroupSelectorPopover', () => {
  it('uses the shared Select component to filter groups by platform', async () => {
    const wrapper = mountPopover()

    const select = wrapper.findComponent({ name: 'Select' })
    expect(select.exists()).toBe(true)
    expect(select.props('options')).toEqual([
      { value: '', label: 'keys.allPlatforms' },
      { value: 'openai', label: 'admin.groups.platforms.openai' },
      { value: 'gemini', label: 'admin.groups.platforms.gemini' },
    ])
    expect(wrapper.find('[data-tour="key-list-group-platform-menu"]').exists()).toBe(false)

    await wrapper.get('[data-test="platform-select"]').trigger('click')

    const renderedGroups = wrapper.findAll('[data-test="group-option"]').map((item) => item.text())
    expect(renderedGroups).toEqual(['Gemini 分组'])
  })

  it('builds and filters platform options from bound business platform', async () => {
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
        businessPlatform: 'glm',
        businessPlatformName: '智谱 GLM',
      },
    ], 'glm')

    const select = wrapper.findComponent({ name: 'Select' })
    expect(select.props('options')).toEqual([
      { value: '', label: 'keys.allPlatforms' },
      { value: 'openai', label: 'admin.groups.platforms.openai' },
      { value: 'gemini', label: 'admin.groups.platforms.gemini' },
      { value: 'glm', label: '智谱 GLM' },
    ])

    await wrapper.get('[data-test="platform-select"]').trigger('click')

    const renderedGroups = wrapper.findAll('[data-test="group-option"]').map((item) => item.text())
    expect(renderedGroups).toEqual(['智谱分组'])
  })
})

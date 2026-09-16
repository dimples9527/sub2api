import { mount } from '@vue/test-utils'
import { nextTick } from 'vue'
import { describe, expect, it, vi } from 'vitest'

import KeyGroupSelectorPopover from '../KeyGroupSelectorPopover.vue'
import type { KeyGroupSelectorOption } from '../KeyGroupSelectorPopover.vue'

vi.mock('vue-i18n', () => ({
  createI18n: () => ({
    install: vi.fn(),
    global: { t: (key: string) => key, locale: { value: 'zh-CN' }, setLocaleMessage: vi.fn() },
  }),
  useI18n: () => ({ t: (key: string) => key }),
}))

/**
 * 同厂商下故意乱序给出倍率，用来验证浮层按倍率升序展示。
 * 另外放一个 composite 分组：它解析后归入 other，用来验证厂商切换会真正收窄列表。
 */
const options: KeyGroupSelectorOption[] = [
  {
    value: 21,
    label: '贵分组',
    description: 'expensive',
    rate: 3,
    userRate: null,
    peakRateEnabled: false,
    peakStart: '',
    peakEnd: '',
    peakRateMultiplier: 1,
    subscriptionType: 'standard',
    platform: 'openai',
  },
  {
    value: 22,
    label: '便宜分组',
    description: 'cheap',
    rate: 0.5,
    userRate: null,
    peakRateEnabled: false,
    peakStart: '',
    peakEnd: '',
    peakRateMultiplier: 1,
    subscriptionType: 'standard',
    platform: 'openai',
  },
  {
    value: 23,
    label: '中等分组',
    description: 'middle',
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
    value: 31,
    label: '复合组',
    description: 'multi',
    rate: 1.5,
    userRate: null,
    peakRateEnabled: false,
    peakStart: '',
    peakEnd: '',
    peakRateMultiplier: 1,
    subscriptionType: 'standard',
    platform: 'composite',
  },
]

function mountPopover(popoverOptions: KeyGroupSelectorOption[] = options) {
  return mount(KeyGroupSelectorPopover, {
    props: {
      open: true,
      activeKeyId: 10,
      position: { top: 100, left: 20 },
      options: popoverOptions,
      selectedGroupId: 21,
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

/** 浮层里当前可见的分组名称，顺序即渲染顺序。 */
function optionLabels(wrapper: ReturnType<typeof mountPopover>) {
  return wrapper.findAll('[data-test="group-option"]').map((item) => item.text())
}

describe('KeyGroupSelectorPopover 列表切换浮层', () => {
  it('按倍率升序展示当前厂商的分组', async () => {
    const wrapper = mountPopover()
    await nextTick()

    // 厂商卡片是单选、没有「全部」项，默认落在第一个有分组的厂商（openai）
    expect((wrapper.get('input[value="openai"]').element as HTMLInputElement).checked).toBe(true)
    // fixture 里 openai 三个分组按倍率 3 / 0.5 / 1 给出，渲染时须升序
    expect(optionLabels(wrapper)).toEqual(['便宜分组', '中等分组', '贵分组'])
  })

  it('选择分组时向外抛出 select 事件', async () => {
    const wrapper = mountPopover()
    await nextTick()

    const target = wrapper.findAll('button').filter((button) => button.text().includes('贵分组'))
    expect(target).toHaveLength(1)

    await target[0].trigger('click')

    expect(wrapper.emitted('select')?.[0]).toEqual([21])
  })

  it('关闭或切换密钥时重置厂商过滤和搜索', async () => {
    const wrapper = mountPopover()
    await nextTick()

    await wrapper.get('input[value="other"]').setValue()
    expect(optionLabels(wrapper)).toEqual(['复合组'])

    // 搜索词在当前厂商下匹配不到任何分组，列表须变空 —— 这样才证明搜索确实生效
    const search = wrapper.get('input.group-selector-search-input')
    await search.setValue('贵')
    await nextTick()
    expect(optionLabels(wrapper)).toEqual([])

    await wrapper.setProps({ open: false })
    await nextTick()
    await wrapper.setProps({ open: true, activeKeyId: 11 })
    await nextTick()

    expect((wrapper.get('input[value="openai"]').element as HTMLInputElement).checked).toBe(true)
    expect((wrapper.get('input.group-selector-search-input').element as HTMLInputElement).value).toBe('')
    expect(optionLabels(wrapper)).toEqual(['便宜分组', '中等分组', '贵分组'])
  })
})

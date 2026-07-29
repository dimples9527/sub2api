import { describe, expect, it, vi } from 'vitest'
import { mount } from '@vue/test-utils'
import { nextTick } from 'vue'

const messages: Record<string, string> = {
  'keys.platformLabel': '平台',
  'keys.allPlatforms': '全部平台',
  'keys.searchGroup': '搜索分组',
  'keys.noGroupFound': '未找到分组',
  'admin.groups.platforms.anthropic': 'Anthropic',
  'admin.groups.platforms.openai': 'OpenAI',
  'admin.groups.platforms.composite': '复合'
}

vi.mock('vue-i18n', async (importOriginal) => {
  const actual = await importOriginal<typeof import('vue-i18n')>()
  return {
    ...actual,
    useI18n: () => ({
      t: (key: string) => messages[key] ?? key
    })
  }
})

import KeyGroupSelectorPopover from '../KeyGroupSelectorPopover.vue'
import type { KeyGroupSelectorOption } from '../KeyGroupSelectorPopover.vue'

const options: KeyGroupSelectorOption[] = [
  {
    value: 2,
    label: '高倍率组',
    description: 'high',
    rate: 2,
    userRate: null,
    peakRateEnabled: false,
    peakStart: '',
    peakEnd: '',
    peakRateMultiplier: 1,
    subscriptionType: 'pay_per_use' as any,
    platform: 'openai'
  },
  {
    value: 1,
    label: '低倍率组',
    description: 'low',
    rate: 1,
    userRate: null,
    peakRateEnabled: false,
    peakStart: '',
    peakEnd: '',
    peakRateMultiplier: 1,
    subscriptionType: 'pay_per_use' as any,
    platform: 'anthropic'
  },
  {
    value: 3,
    label: '复合组',
    description: 'multi',
    rate: 1.5,
    userRate: null,
    peakRateEnabled: false,
    peakStart: '',
    peakEnd: '',
    peakRateMultiplier: 1,
    subscriptionType: 'pay_per_use' as any,
    platform: 'composite'
  }
]

function mountPopover(props: Record<string, unknown> = {}) {
  return mount(KeyGroupSelectorPopover, {
    props: {
      open: true,
      activeKeyId: 10,
      position: { top: 100, left: 20 },
      options,
      selectedGroupId: 1,
      ...props
    },
    global: {
      stubs: {
        Teleport: true,
        GroupOptionItem: {
          props: ['name', 'selected'],
          template: '<div class="stub-group-option">{{ name }}</div>'
        }
      }
    }
  })
}

describe('KeyGroupSelectorPopover', () => {
  it('按倍率升序展示分组，并支持平台过滤包含 composite', async () => {
    const wrapper = mountPopover()
    await nextTick()

    const names = wrapper.findAll('.stub-group-option').map((node) => node.text())
    expect(names).toEqual(['低倍率组', '复合组', '高倍率组'])

    await wrapper.get('button.group-platform-trigger').trigger('click')
    const platformButtons = wrapper.findAll('.group-platform-menu button')
    const openaiBtn = platformButtons.find((btn) => btn.text().includes('OpenAI'))
    expect(openaiBtn).toBeTruthy()
    await openaiBtn!.trigger('click')
    await nextTick()

    const filteredNames = wrapper.findAll('.stub-group-option').map((node) => node.text())
    // openai + composite，并按倍率升序
    expect(filteredNames).toEqual(['复合组', '高倍率组'])
  })

  it('选择分组时向外抛出 select 事件', async () => {
    const wrapper = mountPopover()
    await nextTick()

    const buttons = wrapper.findAll('button').filter((btn) => btn.text().includes('高倍率组'))
    expect(buttons.length).toBeGreaterThan(0)
    await buttons[0].trigger('click')

    expect(wrapper.emitted('select')?.[0]).toEqual([2])
  })

  it('关闭或切换密钥时重置平台过滤和搜索', async () => {
    const wrapper = mountPopover()
    await nextTick()

    await wrapper.get('button.group-platform-trigger').trigger('click')
    const platformButtons = wrapper.findAll('.group-platform-menu button')
    const openaiBtn = platformButtons.find((btn) => btn.text().includes('OpenAI'))
    await openaiBtn!.trigger('click')

    const search = wrapper.get('input.group-selector-search-input')
    await search.setValue('高倍率')
    await nextTick()
    expect(wrapper.findAll('.stub-group-option')).toHaveLength(1)

    await wrapper.setProps({ open: false })
    await nextTick()
    await wrapper.setProps({ open: true, activeKeyId: 11 })
    await nextTick()

    const names = wrapper.findAll('.stub-group-option').map((node) => node.text())
    expect(names).toEqual(['低倍率组', '复合组', '高倍率组'])
    expect((wrapper.get('input.group-selector-search-input').element as HTMLInputElement).value).toBe('')
  })
})

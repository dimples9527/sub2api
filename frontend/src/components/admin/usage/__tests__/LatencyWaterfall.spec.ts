import { describe, expect, it, vi } from 'vitest'
import { mount } from '@vue/test-utils'

vi.mock('vue-i18n', async () => {
  const actual = await vi.importActual<typeof import('vue-i18n')>('vue-i18n')
  return { ...actual, useI18n: () => ({ t: (key: string) => key }) }
})

import LatencyWaterfall from '../LatencyWaterfall.vue'

const widths = (wrapper: ReturnType<typeof mount>): string[] =>
  wrapper.findAll('.flex.h-3 > div').map((el) => (el.element as HTMLElement).style.width)

describe('LatencyWaterfall', () => {
  // 排队 85s 是本地连接池争抢，上游首字 6.29s 才是上游真实处理——分解的全部意义在这个对比上。
  it('splits local overhead from upstream wait and derives the streaming tail', () => {
    const wrapper = mount(LatencyWaterfall, {
      props: {
        phases: {
          build_ms: 10,
          slot_wait_ms: 85_000,
          connect_ms: 30,
          tls_ms: 70,
          first_byte_ms: 6_290,
          conn_reused: false,
        },
        durationMs: 96_000,
        firstTokenMs: 92_000,
      },
    })

    // 组装+排队+建连+TLS
    expect(wrapper.get('[data-testid="latency-local-overhead"]').text()).toContain('85.11s')
    // 6 段：组装/排队/建连/TLS/等上游首字/其他 + 流传输
    expect(widths(wrapper)).toHaveLength(7)
    expect(wrapper.text()).toContain('usage.latencyPhaseStream')
  })

  it('omits absent phases and the streaming tail for non-streaming requests', () => {
    const wrapper = mount(LatencyWaterfall, {
      props: {
        phases: {
          build_ms: null,
          slot_wait_ms: 0,
          connect_ms: null,
          tls_ms: null,
          first_byte_ms: 1_200,
          conn_reused: true,
        },
        durationMs: 1_260,
        firstTokenMs: null,
      },
    })

    // 只剩「等上游首字 + 其他」，零值与 null 段都不占位。
    expect(widths(wrapper)).toHaveLength(2)
    expect(wrapper.text()).not.toContain('usage.latencyPhaseStream')
    expect(wrapper.text()).toContain('usage.latencyConnReused')
  })
})

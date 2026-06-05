import { beforeEach, describe, expect, it, vi } from 'vitest'
import { flushPromises, mount } from '@vue/test-utils'

import UpstreamGroupsView from '../UpstreamGroupsView.vue'

const {
  createGroupMock,
  getUpstreamAvailableGroupsMock,
  getUpstreamMonitorStatusMock,
} = vi.hoisted(() => ({
  createGroupMock: vi.fn(),
  getUpstreamAvailableGroupsMock: vi.fn(),
  getUpstreamMonitorStatusMock: vi.fn(),
}))

vi.mock('@/api/admin/groups', () => ({
  create: createGroupMock,
  default: {
    create: createGroupMock,
    getUpstreamAvailableGroups: getUpstreamAvailableGroupsMock,
    getUpstreamMonitorStatus: getUpstreamMonitorStatusMock,
  },
  getUpstreamAvailableGroups: getUpstreamAvailableGroupsMock,
  getUpstreamMonitorStatus: getUpstreamMonitorStatusMock,
}))

function mountView() {
  return mount(UpstreamGroupsView, {
    global: {
      stubs: {
        AppLayout: { template: '<div><slot /></div>' },
        Icon: { template: '<span />' },
      },
    },
  })
}

describe('UpstreamGroupsView', () => {
  beforeEach(() => {
    createGroupMock.mockReset()
    getUpstreamAvailableGroupsMock.mockReset()
    getUpstreamMonitorStatusMock.mockReset()
    getUpstreamMonitorStatusMock.mockResolvedValue({ groups: [] })
  })

  it('syncs an unmatched upstream group to a local group with an editable rate multiplier', async () => {
    getUpstreamAvailableGroupsMock
      .mockResolvedValueOnce([
        {
          id: 5,
          name: 'claude福利',
          description: 'upstream desc',
          platform: 'anthropic',
          rate_multiplier: 0.15,
          status: 'active',
          subscription_type: 'standard',
          daily_limit_usd: 10,
          weekly_limit_usd: null,
          monthly_limit_usd: 100,
          claude_code_only: true,
          local_group_id: null,
          local_group_name: '',
          local_rate_multiplier: null,
        },
      ])
      .mockResolvedValueOnce([
        {
          id: 5,
          name: 'claude福利',
          platform: 'anthropic',
          rate_multiplier: 0.15,
          status: 'active',
          local_group_id: 20,
          local_group_name: 'claude福利',
          local_rate_multiplier: 0.2,
        },
      ])
    createGroupMock.mockResolvedValue({ id: 20, name: 'claude福利' })

    const wrapper = mountView()
    await flushPromises()

    await wrapper.find('[data-test="sync-local-group-5"]').trigger('click')
    await wrapper.find('[data-test="sync-rate-multiplier"]').setValue('0.2')
    await wrapper.find('[data-test="confirm-sync-local-group"]').trigger('click')
    await flushPromises()

    expect(createGroupMock).toHaveBeenCalledWith({
      name: 'claude福利',
      description: 'upstream desc',
      platform: 'anthropic',
      rate_multiplier: 0.2,
      subscription_type: 'standard',
      daily_limit_usd: 10,
      weekly_limit_usd: null,
      monthly_limit_usd: 100,
      claude_code_only: true,
    })
    expect(getUpstreamAvailableGroupsMock).toHaveBeenCalledTimes(2)
    expect(wrapper.text()).toContain('同步成功')
  })
})

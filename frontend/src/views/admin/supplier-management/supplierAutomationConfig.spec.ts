import { describe, expect, it } from 'vitest'
import type { SupplierProviderAccount } from '../../../api/admin/supplierProviderData'
import {
  buildHealthGuardAccountMappings,
  healthGuardAccountUnavailableReason,
} from './supplierAutomationConfig'

function makeAccount(overrides: Partial<SupplierProviderAccount> = {}): SupplierProviderAccount {
  return {
    id: 1,
    provider_id: 1,
    provider_name: '供应商',
    upstream_account_key: 'key',
    name: 'upstream',
    status: 'active',
    group_key: 'g',
    group_name: 'G',
    rate_multiplier: 1,
    binding_groups: [],
    raw_status: 'active',
    active: true,
    last_seen_at: '',
    local_account_match_status: 'matched',
    local_account_match_count: 1,
    local_account_id: 7,
    local_account_name: '皓悦-claude-ccmax',
    local_account_platform: 'anthropic',
    local_account_status: 'active',
    supplier_current_balance: 0,
    supplier_today_cost: 0,
    group_record_delete_eligible: false,
    account_record_delete_eligible: false,
    ...overrides,
  }
}

describe('healthGuardAccountUnavailableReason', () => {
  it('可用账号不给原因', () => {
    expect(healthGuardAccountUnavailableReason(makeAccount())).toBe('')
  })

  it('按「上游停用 → 匹配失效 → 本地账号状态」的顺序给出原因', () => {
    expect(healthGuardAccountUnavailableReason(makeAccount({ active: false }))).toBe('上游账号已停用')
    expect(healthGuardAccountUnavailableReason(makeAccount({ local_account_match_status: 'conflict' })))
      .toBe('匹配到多个本地账号')
    expect(healthGuardAccountUnavailableReason(makeAccount({ local_account_match_status: 'unmatched' })))
      .toBe('未匹配本地账号')
    // 查询侧按 deleted_at 过滤，本地账号删除后 join 不到行，状态只剩空串。
    expect(healthGuardAccountUnavailableReason(makeAccount({ local_account_status: '' }))).toBe('本地账号已删除')
  })

  it('本地账号状态异常时带出上游错误里的 message，而不是整段 JSON', () => {
    const reason = healthGuardAccountUnavailableReason(makeAccount({
      local_account_status: 'error',
      local_account_last_test_error:
        'API returned 403: {"code":"INSUFFICIENT_BALANCE","message":"Insufficient account balance"}',
    }))
    expect(reason).toBe('本地账号状态异常（error）：Insufficient account balance')
  })

  it('没有可读错误时只给状态，不拼空冒号', () => {
    expect(healthGuardAccountUnavailableReason(makeAccount({ local_account_status: 'disabled' })))
      .toBe('本地账号状态异常（disabled）')
  })
})

describe('buildHealthGuardAccountMappings 的不可用原因', () => {
  it('同一本地账号只要有一个可用来源，就清空原因', () => {
    const [mapping] = buildHealthGuardAccountMappings([
      makeAccount({ id: 1, local_account_status: 'error' }),
      makeAccount({ id: 2 }),
    ])
    expect(mapping.available).toBe(true)
    expect(mapping.unavailableReason).toBe('')
  })

  it('全部来源都不可用时保留首个原因', () => {
    const [mapping] = buildHealthGuardAccountMappings([
      makeAccount({ id: 1, active: false }),
      makeAccount({ id: 2, local_account_status: 'error' }),
    ])
    expect(mapping.available).toBe(false)
    expect(mapping.unavailableReason).toBe('上游账号已停用')
  })
})

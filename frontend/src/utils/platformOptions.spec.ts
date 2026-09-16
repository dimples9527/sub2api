import { describe, expect, it } from 'vitest'
import { applyPlatformCatalog, buildPlatformOptions, CORE_PLATFORM_CODES, CORE_PLATFORM_OPTIONS } from './platformOptions'

describe('platform options', () => {
  it('contains all framework core platforms in one stable order', () => {
    expect(CORE_PLATFORM_CODES).toEqual([
      'anthropic',
      'openai',
      'gemini',
      'antigravity',
      'grok',
      'kimi',
      'zhipu',
      'deepseek',
      'minimax',
      'opencode_go',
      'composite',
    ])
  })

  it('covers every concrete account platform plus composite before the API responds', () => {
    // 这条兜底列表会在 platformsAPI.list() 返回后被 applyPlatformCatalog 覆盖，
    // 但接口失败或尚未返回时它就是唯一来源。漏登记的平台会在所有筛选下拉里消失，
    // minimax 与 opencode_go 就漏过一次，所以这里把清单钉死。
    // ⚠️ 必须在下面 applyPlatformCatalog 那条用例之前跑 —— 它会整体覆盖 CORE_PLATFORM_OPTIONS。
    const concretePlatforms = [
      'anthropic',
      'openai',
      'gemini',
      'antigravity',
      'grok',
      'kimi',
      'zhipu',
      'deepseek',
      'minimax',
      'opencode_go',
    ]

    expect(CORE_PLATFORM_OPTIONS.map((option) => option.value)).toEqual([
      ...concretePlatforms,
      'composite',
    ])
  })

  it('merges enabled custom platforms without duplicating core codes', () => {
    expect(buildPlatformOptions([
      { id: 1, code: 'deepseek', name: '旧 DeepSeek', color: '#000000', enabled: true, sort_order: 1, created_at: '', updated_at: '' },
      { id: 2, code: 'provider_x', name: 'Provider X', color: '#111111', enabled: true, sort_order: 2, created_at: '', updated_at: '' },
      { id: 3, code: 'disabled_x', name: 'Disabled', color: '#222222', enabled: false, sort_order: 3, created_at: '', updated_at: '' },
    ]).filter(option => option.value === 'deepseek' || option.value === 'provider_x')).toEqual([
      { value: 'deepseek', label: 'DeepSeek' },
      { value: 'provider_x', label: 'Provider X' },
    ])
  })

  it('applies a framework platform added after the frontend was built', () => {
    applyPlatformCatalog([
      { code: 'openai', name: 'OpenAI', color: '#22c55e', health_guard: true },
      { code: 'new-framework-platform', name: 'New Framework Platform', color: '#123456', health_guard: true },
    ])

    expect(CORE_PLATFORM_OPTIONS).toContainEqual({
      value: 'new-framework-platform',
      label: 'New Framework Platform',
    })
  })
})

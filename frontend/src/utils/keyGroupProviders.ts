import type { GroupPlatform } from '@/types'

export type KeyGroupProvider = 'anthropic' | 'openai' | 'domestic' | 'other'

export const KEY_GROUP_PROVIDERS = ['anthropic', 'openai', 'domestic', 'other'] as const

// Classify by the configured upstream platform, never by a group's display name.
const PROVIDER_BY_PLATFORM: Record<GroupPlatform, KeyGroupProvider> = {
  anthropic: 'anthropic',
  openai: 'openai',
  kimi: 'domestic',
  zhipu: 'domestic',
  deepseek: 'domestic',
  minimax: 'domestic',
  gemini: 'other',
  grok: 'other',
  antigravity: 'other',
  composite: 'other',
  opencode_go: 'other'
}

/**
 * 入参是「解析后的业务平台」，而不是分组的 platform 原始字段。
 * composite 分组、或被监控层覆写过上游的分组，其真实上游与 platform 字段并不一致，
 * 调用方应先用 resolveGroupBusinessPlatform 解析后再传进来。
 * 未知平台一律归入 other，保证以后新增平台不会让分组在选择器里凭空消失。
 */
export function getKeyGroupProvider(businessPlatform: string | null | undefined): KeyGroupProvider {
  return PROVIDER_BY_PLATFORM[(businessPlatform ?? '') as GroupPlatform] ?? 'other'
}

// Collections use representative provider marks rather than an invented brand logo.
export const KEY_GROUP_PROVIDER_ICONS: Record<KeyGroupProvider, GroupPlatform[]> = {
  anthropic: ['anthropic'],
  openai: ['openai'],
  domestic: ['deepseek', 'kimi'],
  other: ['gemini', 'grok']
}

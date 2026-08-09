import type { GroupPlatform } from '@/types'

/** ?????????????????? */
export type BusinessPlatformFilterValue = string | ''

export interface GroupBusinessPlatformSource {
  platform: string
  /** ?????????????????? */
  businessPlatform?: string | null
  businessPlatformName?: string | null
  /** ???? DTO / ???? camelCase ? snake_case ?? */
  effectivePlatform?: string | null
  effective_platform?: string | null
  effectivePlatformName?: string | null
  effective_platform_name?: string | null
  actualPlatform?: string | null
  actual_platform?: string | null
}

export interface GroupBusinessPlatformSortable extends GroupBusinessPlatformSource {
  label: string
  rate: number
}

/** ???????????????????? */
export const BUSINESS_PLATFORM_CORE_ORDER = [
  'anthropic',
  'openai',
  'gemini',
  'antigravity',
  'grok',
  'composite'
] as const

const BUSINESS_PLATFORM_CORE_SET = new Set<string>(BUSINESS_PLATFORM_CORE_ORDER)

function normalizeBusinessPlatform(platform: string | null | undefined): string {
  return (platform || '').trim().toLowerCase()
}

function firstPlatformValue(...values: Array<string | null | undefined>): string {
  for (const value of values) {
    const normalized = normalizeBusinessPlatform(value)
    if (normalized) return normalized
  }
  return ''
}

function firstPlatformName(...values: Array<string | null | undefined>): string {
  for (const value of values) {
    const name = (value || '').trim()
    if (name) return name
  }
  return ''
}

/** ?????????????????????????? */
export function resolveGroupBusinessPlatform(group: GroupBusinessPlatformSource): string {
  return firstPlatformValue(
    group.businessPlatform,
    group.effectivePlatform,
    group.effective_platform,
    group.actualPlatform,
    group.actual_platform,
    group.platform
  )
}

/** ??????????????/???????? platform ?? */
function hasExplicitGroupBusinessPlatform(group: GroupBusinessPlatformSource): boolean {
  return !!firstPlatformValue(
    group.businessPlatform,
    group.effectivePlatform,
    group.effective_platform,
    group.actualPlatform,
    group.actual_platform
  )
}

function resolveGroupBusinessPlatformName(group: GroupBusinessPlatformSource): string {
  return firstPlatformName(
    group.businessPlatformName,
    group.effectivePlatformName,
    group.effective_platform_name
  )
}

function businessPlatformOptionLabel(
  platform: string,
  explicitName: string | undefined,
  platformLabel: (platform: string) => string
): string {
  if (explicitName) return explicitName
  if (!BUSINESS_PLATFORM_CORE_SET.has(platform)) return platform
  return platformLabel(platform)
}

/**
 * ???????????????
 * ???????????????????????? composite ???????????????
 */
export function matchesGroupBusinessPlatformFilter(
  groupPlatform: string | null | undefined,
  platformFilter: BusinessPlatformFilterValue | null | undefined,
  group?: Partial<GroupBusinessPlatformSource>
): boolean {
  const filter = normalizeBusinessPlatform(platformFilter)
  if (!filter) return true

  if (group) {
    const businessPlatform = resolveGroupBusinessPlatform({
      platform: group.platform || groupPlatform || '',
      ...group
    })
    if (businessPlatform) {
      if (businessPlatform === filter) return true
      if (hasExplicitGroupBusinessPlatform(group as GroupBusinessPlatformSource)) return false
      return businessPlatform === 'composite' && BUSINESS_PLATFORM_CORE_SET.has(filter)
    }
  }

  const rawPlatform = normalizeBusinessPlatform(groupPlatform)
  return rawPlatform === filter || (rawPlatform === 'composite' && BUSINESS_PLATFORM_CORE_SET.has(filter))
}

/** ???????????????????? */
export function sortGroupsByRateAsc<T extends GroupBusinessPlatformSortable>(items: T[]): T[] {
  return [...items].sort((a, b) => {
    if (a.rate !== b.rate) return a.rate - b.rate
    return a.label.localeCompare(b.label, 'zh-CN')
  })
}

/** ??????????????? */
export function filterAndSortGroupsByBusinessPlatform<T extends GroupBusinessPlatformSortable>(
  options: T[],
  platformFilter: BusinessPlatformFilterValue | null | undefined
): T[] {
  const filtered = options.filter((option) =>
    matchesGroupBusinessPlatformFilter(option.platform, platformFilter, option)
  )
  return sortGroupsByRateAsc(filtered)
}

/**
 * ?????????????
 * ???????????????????????????
 */
export function buildGroupBusinessPlatformOptions(
  groups: GroupBusinessPlatformSource[],
  labels: {
    all: string
    platformLabel: (platform: string) => string
  }
): Array<{ value: BusinessPlatformFilterValue; label: string }> {
  const present = new Map<string, string>()
  for (const group of groups) {
    const platform = resolveGroupBusinessPlatform(group)
    if (!platform || present.has(platform)) continue
    present.set(platform, resolveGroupBusinessPlatformName(group))
  }
  const options: Array<{ value: BusinessPlatformFilterValue; label: string }> = [
    { value: '', label: labels.all }
  ]

  for (const platform of BUSINESS_PLATFORM_CORE_ORDER) {
    if (!present.has(platform)) continue
    options.push({
      value: platform,
      label: businessPlatformOptionLabel(platform, present.get(platform), labels.platformLabel)
    })
  }

  // ???????????????????
  for (const [platform, explicitName] of present) {
    if (BUSINESS_PLATFORM_CORE_SET.has(platform)) continue
    options.push({
      value: platform,
      label: businessPlatformOptionLabel(platform, explicitName, labels.platformLabel)
    })
  }

  return options
}

/** ??????????????????? */
export function isGroupValidForBusinessPlatformFilter(
  selectedGroupId: number | null | undefined,
  selectedGroupPlatform: GroupPlatform | null | undefined,
  platformFilter: BusinessPlatformFilterValue | null | undefined,
  selectedGroup?: Partial<GroupBusinessPlatformSource>
): boolean {
  if (selectedGroupId == null) return true
  if (!selectedGroupPlatform) return false
  return matchesGroupBusinessPlatformFilter(selectedGroupPlatform, platformFilter, selectedGroup)
}

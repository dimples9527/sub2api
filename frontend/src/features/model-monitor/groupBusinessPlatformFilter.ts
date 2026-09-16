import type { GroupPlatform } from '@/types'

/** 业务平台筛选值；空字符串表示「全部平台」。 */
export type BusinessPlatformFilterValue = string | ''

export interface GroupBusinessPlatformSource {
  platform: string
  /** 监控层解析出的业务平台，存在时优先于 platform 使用。 */
  businessPlatform?: string | null
  businessPlatformName?: string | null
  /** 同时兼容 DTO 的 camelCase 与 snake_case 两种命名。 */
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

/** 展示顺序固定的核心平台；其余平台排在它们之后。 */
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

/** 按优先级解析分组真实归属的业务平台，全部缺失时退回 platform。 */
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

/** 是否由监控层显式指定了业务平台，而不是回退到 platform。 */
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
 * 判断分组是否命中平台筛选。
 * 显式指定了业务平台的分组只按该平台匹配；composite 分组额外视为命中任一核心平台。
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

/** 按倍率升序排列，倍率相同时按名称排序。 */
export function sortGroupsByRateAsc<T extends GroupBusinessPlatformSortable>(items: T[]): T[] {
  return [...items].sort((a, b) => {
    if (a.rate !== b.rate) return a.rate - b.rate
    return a.label.localeCompare(b.label, 'zh-CN')
  })
}

/** 先按业务平台筛选，再按倍率升序排列。 */
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
 * 从分组数据里提取实际出现过的业务平台，生成筛选下拉项。
 * 核心平台按固定顺序排在前面，其余平台按出现顺序追加。
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

  // 核心平台之外的平台追加在后面，保持数据里出现的顺序。
  for (const [platform, explicitName] of present) {
    if (BUSINESS_PLATFORM_CORE_SET.has(platform)) continue
    options.push({
      value: platform,
      label: businessPlatformOptionLabel(platform, explicitName, labels.platformLabel)
    })
  }

  return options
}

/** 切换平台筛选后，判断已选分组是否仍在新筛选范围内。 */
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

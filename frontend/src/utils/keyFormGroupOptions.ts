import type { GroupPlatform } from '@/types'

/** 密钥表单中用于平台过滤的平台值；空字符串表示全部平台 */
export type KeyFormPlatformFilter = GroupPlatform | ''

export interface KeyFormGroupSortable {
  label: string
  rate: number
  platform: GroupPlatform
}

/** 平台下拉展示顺序（仅展示实际出现的平台） */
export const KEY_FORM_PLATFORM_ORDER: GroupPlatform[] = [
  'anthropic',
  'openai',
  'gemini',
  'antigravity',
  'grok',
  'composite'
]

/**
 * 判断分组是否匹配密钥表单的平台过滤。
 * 选择具体平台时，同时包含 composite 复合分组。
 */
export function matchesKeyFormPlatformFilter(
  groupPlatform: GroupPlatform,
  platformFilter: KeyFormPlatformFilter | null | undefined
): boolean {
  if (!platformFilter) return true
  return groupPlatform === platformFilter || groupPlatform === 'composite'
}

/** 按倍率升序排序；倍率相同再按名称稳定排序 */
export function sortKeyFormGroupsByRateAsc<T extends KeyFormGroupSortable>(items: T[]): T[] {
  return [...items].sort((a, b) => {
    if (a.rate !== b.rate) return a.rate - b.rate
    return a.label.localeCompare(b.label, 'zh-CN')
  })
}

/** 按平台过滤并按倍率升序排序 */
export function filterAndSortKeyFormGroupOptions<T extends KeyFormGroupSortable>(
  options: T[],
  platformFilter: KeyFormPlatformFilter | null | undefined
): T[] {
  const filtered = options.filter((option) =>
    matchesKeyFormPlatformFilter(option.platform, platformFilter)
  )
  return sortKeyFormGroupsByRateAsc(filtered)
}

/**
 * 从可用分组中推导平台过滤选项。
 * 始终包含“全部平台”，其余仅展示当前数据中出现的平台。
 */
export function buildKeyFormPlatformOptions(
  groups: Array<{ platform: GroupPlatform }>,
  labels: {
    all: string
    platformLabel: (platform: GroupPlatform) => string
  }
): Array<{ value: KeyFormPlatformFilter; label: string }> {
  const present = new Set(groups.map((group) => group.platform))
  const options: Array<{ value: KeyFormPlatformFilter; label: string }> = [
    { value: '', label: labels.all }
  ]

  for (const platform of KEY_FORM_PLATFORM_ORDER) {
    if (!present.has(platform)) continue
    options.push({
      value: platform,
      label: labels.platformLabel(platform)
    })
  }

  // 兜底：未在预设顺序中的平台也展示
  for (const platform of present) {
    if (KEY_FORM_PLATFORM_ORDER.includes(platform)) continue
    options.push({
      value: platform,
      label: labels.platformLabel(platform)
    })
  }

  return options
}

/** 切换平台后，判断当前已选分组是否仍有效 */
export function isSelectedGroupValidForPlatformFilter(
  selectedGroupId: number | null | undefined,
  selectedGroupPlatform: GroupPlatform | null | undefined,
  platformFilter: KeyFormPlatformFilter | null | undefined
): boolean {
  if (selectedGroupId == null) return true
  if (!selectedGroupPlatform) return false
  return matchesKeyFormPlatformFilter(selectedGroupPlatform, platformFilter)
}

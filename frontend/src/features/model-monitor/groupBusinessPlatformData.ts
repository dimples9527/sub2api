import { apiClient, buildGatewayUrl } from '@/api/client'

interface LLMMonitorGroupPlatformData {
  id: number
  platform?: string | null
  actual_platform?: string | null
  effective_platform?: string | null
  effective_platform_name?: string | null
}

/** ?????????? */
export interface GroupBusinessPlatformInfo {
  businessPlatform: string
  businessPlatformName: string | null
  effectivePlatform: string | null
  actualPlatform: string | null
}

function normalizeBusinessPlatform(value: string | null | undefined): string {
  return (value || '').trim().toLowerCase()
}

function normalizeBusinessPlatformName(value: string | null | undefined): string | null {
  const name = (value || '').trim()
  return name || null
}

/**
 * ??????????????????? ID ??????
 * ????????????????????? API Key / Group ?? DTO?
 */
export function buildGroupBusinessPlatformMap(
  groups: LLMMonitorGroupPlatformData[]
): Map<number, GroupBusinessPlatformInfo> {
  const map = new Map<number, GroupBusinessPlatformInfo>()

  for (const group of groups) {
    if (!group.id) continue

    const effectivePlatform = normalizeBusinessPlatform(group.effective_platform)
    const actualPlatform = normalizeBusinessPlatform(group.actual_platform)
    const fallbackPlatform = normalizeBusinessPlatform(group.platform)
    const businessPlatform = effectivePlatform || actualPlatform || fallbackPlatform
    if (!businessPlatform) continue

    map.set(group.id, {
      businessPlatform,
      businessPlatformName: normalizeBusinessPlatformName(group.effective_platform_name),
      effectivePlatform: effectivePlatform || null,
      actualPlatform: actualPlatform || null,
    })
  }

  return map
}

/**
 * ????????????????????
 * ??????????????????????? platform?
 */
export async function loadGroupBusinessPlatformMap(): Promise<Map<number, GroupBusinessPlatformInfo>> {
  try {
    const { data } = await apiClient.get<LLMMonitorGroupPlatformData[]>(
      buildGatewayUrl('/api/llm-monitor/groups')
    )
    return buildGroupBusinessPlatformMap(Array.isArray(data) ? data : [])
  } catch (error) {
    console.error('Failed to load group business platform data:', error)
    return new Map()
  }
}

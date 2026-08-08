/**
 * 模型监控相关的管理端接口。
 * 这里只处理“分组实际平台”这类只给模型监控使用的独立配置。
 */

import { apiClient } from '../client'
import type { GroupPlatform } from '@/types'

export interface LLMMonitorGroupPlatformOverride {
  id: number
  name: string
  platform: GroupPlatform
  actual_platform: GroupPlatform | ''
  effective_platform: GroupPlatform
  rate_multiplier: number
}

export interface LLMMonitorGroupPlatformOverrideUpdateResult {
  message: string
}

export async function listLLMMonitorGroupPlatformOverrides(): Promise<LLMMonitorGroupPlatformOverride[]> {
  const { data } = await apiClient.get<LLMMonitorGroupPlatformOverride[]>('/admin/model-monitor/platform-overrides')
  return data
}

export async function setLLMMonitorGroupPlatformOverride(
  groupId: number,
  actualPlatform: GroupPlatform,
): Promise<LLMMonitorGroupPlatformOverrideUpdateResult> {
  const { data } = await apiClient.put<LLMMonitorGroupPlatformOverrideUpdateResult>(
    `/admin/model-monitor/platform-overrides/${groupId}`,
    { actual_platform: actualPlatform },
  )
  return data
}

export async function clearLLMMonitorGroupPlatformOverride(
  groupId: number,
): Promise<LLMMonitorGroupPlatformOverrideUpdateResult> {
  const { data } = await apiClient.delete<LLMMonitorGroupPlatformOverrideUpdateResult>(
    `/admin/model-monitor/platform-overrides/${groupId}`,
  )
  return data
}

export const modelMonitorAPI = {
  listLLMMonitorGroupPlatformOverrides,
  setLLMMonitorGroupPlatformOverride,
  clearLLMMonitorGroupPlatformOverride,
}

export default modelMonitorAPI

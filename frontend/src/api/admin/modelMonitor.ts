/**
 * 模型监控相关的管理端接口。
 * 这里只处理“分组实际平台”这类只给模型监控使用的独立配置。
 */

import { apiClient } from '../client'


export interface LLMMonitorGroupPlatformOverride {
  id: number
  name: string
  platform: string
  actual_platform: string | ''
  effective_platform: string
  effective_platform_name: string
  rate_multiplier: number
  show_in_monitor: boolean
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
  actualPlatform: string,
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

export async function setLLMMonitorGroupVisibility(
  groupId: number,
  showInMonitor: boolean,
): Promise<LLMMonitorGroupPlatformOverrideUpdateResult> {
  const { data } = await apiClient.put<LLMMonitorGroupPlatformOverrideUpdateResult>(
    `/admin/model-monitor/visibility/${groupId}`,
    { show_in_monitor: showInMonitor },
  )
  return data
}

export const modelMonitorAPI = {
  listLLMMonitorGroupPlatformOverrides,
  setLLMMonitorGroupPlatformOverride,
  setLLMMonitorGroupVisibility,
  clearLLMMonitorGroupPlatformOverride,
}

export default modelMonitorAPI

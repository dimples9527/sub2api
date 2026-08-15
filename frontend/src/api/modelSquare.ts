/**
 * 模型广场用户只读 API
 * 后端聚合配置、渠道、分组、分组平台覆盖与参考价后，前端复用
 * buildConfiguredModelSquareResult 生成展示目录，供普通用户查看。
 */

import { apiClient } from './client'
import {
  buildConfiguredModelSquareResult,
  type AdminModelSquareResult,
  type ModelSquareUserChannel,
  type ModelSquareUserGroup,
} from './admin/modelSquare'
import type { ModelSquareConfigPayload, ModelSquareOfficialPricing } from './admin/modelSquareConfig'

/** 模型广场用户接口返回的聚合数据（字段与后端 DTO 一一对应）。 */
export interface ModelSquareUserPayload {
  config: ModelSquareConfigPayload
  channels: ModelSquareUserChannel[]
  groups: ModelSquareUserGroup[]
  platform_overrides: Array<{ id: number | string; effective_platform?: string }>
  reference_prices: Record<string, ModelSquareOfficialPricing>
}

/** 获取模型广场用户只读聚合数据并生成展示目录。 */
export async function getModelSquare(): Promise<AdminModelSquareResult> {
  const { data } = await apiClient.get<ModelSquareUserPayload>('/model-square')

  // 分组平台配置覆盖：分组 ID -> 展示平台，与模型监控保持一致。
  const platformOverrides = new Map(
    (data.platform_overrides || []).map(item => [
      String(item.id),
      normalizePlatform(item.effective_platform),
    ])
  )

  // 参考价 Map 的 key 与聚合函数的 referencePricingKey 保持一致（小写模型 ID）。
  const referencePrices = new Map(
    Object.entries(data.reference_prices || {}).map(([key, price]) => [
      key.trim().toLowerCase(),
      price,
    ])
  )

  return buildConfiguredModelSquareResult(
    data.config,
    data.channels || [],
    data.groups || [],
    referencePrices,
    platformOverrides
  )
}

function normalizePlatform(value: string | undefined) {
  return (value || '').trim().toLowerCase()
}

export const modelSquareAPI = {
  get: getModelSquare,
}

export default modelSquareAPI
import { getAllIncludingInactive } from './groups'
import { listLLMMonitorGroupPlatformOverrides } from './modelMonitor'
import {
  get as getModelSquareConfig,
  getModelPricing as getModelSquareReferencePricing,
  type ModelSquareConfigPayload,
  type ModelSquareOfficialPricing,
  type ModelSquarePlatformModelConfig,
} from './modelSquareConfig'

export type { ModelSquareConfigPayload } from './modelSquareConfig'

export interface ModelSquareGroup {
  id: number | string
  name: string
  platform?: string
  rate_multiplier?: number
}

export interface ModelSquareModel {
  id: string
  display_name?: string
  provider?: string
  platform?: string
  available?: boolean
  mode?: string
  input_price?: number | string
  output_price?: number | string
  cache_write_price?: number | string
  cache_write_1h_price?: number | string
  cache_read_price?: number | string
  input_price_priority?: number | string
  output_price_priority?: number | string
  cache_write_price_priority?: number | string
  cache_read_price_priority?: number | string
  image_input_price?: number | string
  image_output_price?: number | string
  per_request_price?: number | string
  rate_multiplier?: number
  group_ids?: Array<number | string>
}

export interface ModelSquarePayload {
  groups?: ModelSquareGroup[]
  models?: ModelSquareModel[]
  data?: {
    groups?: ModelSquareGroup[]
    models?: ModelSquareModel[]
  }
  code?: number
  message?: string
}

/**
 * 模型广场用户接口返回的最小分组结构。
 */
export interface ModelSquareUserGroup {
  id: number | string
  name: string
  platform?: string
  rate_multiplier?: number
}

export interface AdminModelSquareResult {
  provider_slug: string
  provider_name: string
  provider_type: string
  payload: ModelSquarePayload
}

const PER_MILLION = 1_000_000
const TOKEN_PRICE_FIELDS = [
  'input_price',
  'output_price',
  'cache_write_price',
  'cache_write_1h_price',
  'cache_read_price',
  'input_price_priority',
  'output_price_priority',
  'cache_write_price_priority',
  'cache_read_price_priority',
  'image_input_price',
  'image_output_price',
] as const
const REQUEST_PRICE_FIELDS = ['per_request_price'] as const

type TokenPriceField = typeof TOKEN_PRICE_FIELDS[number]
type RequestPriceField = typeof REQUEST_PRICE_FIELDS[number]
type PriceField = TokenPriceField | RequestPriceField

/**
 * 使用模型广场配置生成展示目录。
 *
 * 分组归属直接取配置里为模型手动绑定的 group_ids —— 它是唯一来源，不再由渠道反推：
 * 反推出来的归属既不能收窄也不能指定，管理员在配置页看不到、也改不动，
 * 一旦渠道配置变化就会让展示页的分组筛选结果跟着漂移。
 *
 * 可用性同样只看这份绑定（详见下方 available 处的说明）。渠道数据不再参与这里，
 * 因此本函数也不再需要 channels 入参。
 */
export function buildConfiguredModelSquareResult(
  config: ModelSquareConfigPayload,
  localGroups: ModelSquareUserGroup[],
  referencePrices = new Map<string, ModelSquareOfficialPricing>(),
  platformOverrides = new Map<string, string>()
): AdminModelSquareResult {
  const configuredPlatforms = new Set(
    (config.platforms || [])
      .map(item => normalizePlatform(item.platform))
      .filter(Boolean)
  )
  const models: ModelSquareModel[] = []

  for (const platformConfig of config.platforms || []) {
    const platform = normalizePlatform(platformConfig.platform)
    if (!platform) continue
    const providerName = platformConfig.name?.trim() || platform

    for (const modelConfig of platformConfig.models || []) {
      const modelID = modelConfig.id?.trim()
      if (!modelID) continue

      const groupIDs = normalizeModelGroupIDs(modelConfig.group_ids)
      const minimumRateMultiplier = minimumGroupRateMultiplier(localGroups, platform, platformOverrides)

      models.push({
        id: modelID,
        display_name: modelConfig.display_name?.trim() || undefined,
        provider: providerName,
        platform,
        /*
          可用性只由「有没有绑定分组」决定。
          
          这里曾经要求「存在 active 渠道、渠道包含该模型、且渠道自己绑了与平台兼容的分组」，
          但那套判据与真实调度路径不符：请求路由走的是分组→账号（account_groups），
          渠道只负责定价、模型映射与模型限制，分组没配渠道时请求照跑（用分组自身的定价）。
          结果就是有账号支撑的分组被误判成不可用，管理员看到「渠道未启用」却找不到可修的地方。
          
          绑定分组是配置页里唯一需要管理员维护的动作，可用性就以它为准 ——
          绑了即视为可用，也顺带保证展示页按分组筛选时不会筛出「标着不可用」的模型。
        */
        available: groupIDs.length > 0,
        mode: modelMode(modelID, modelConfig, referencePrices.get(referencePricingKey(modelID))),
        rate_multiplier: minimumRateMultiplier,
        ...configuredPrices(modelConfig, referencePrices.get(referencePricingKey(modelID)), minimumRateMultiplier),
        group_ids: groupIDs,
      })
    }
  }

  const groups: ModelSquareGroup[] = localGroups
    .filter(group => {
      const platform = effectiveGroupPlatform(group, platformOverrides)
      return configuredPlatforms.has(platform) || platform === 'composite'
    })
    .map(group => ({
      id: group.id,
      name: group.name,
      platform: effectiveGroupPlatform(group, platformOverrides),
      rate_multiplier: group.rate_multiplier,
    }))

  return {
    provider_slug: 'configured',
    provider_name: '模型广场配置',
    provider_type: 'local',
    payload: { groups, models },
  }
}

/**
 * 规范化手动绑定的分组 ID：丢弃非法值、去重、升序。
 *
 * 口径必须与后端 normalizeModelSquareGroupIDs 保持一致。三处（本函数、配置页保存载荷、
 * 分组选择控件）必须共用它：任一处的形态不同，同一组 ID 就会序列化成不同字符串，
 * 配置页的「未保存改动」比对随之误报 —— 管理员会看到明明没改却提示有改动。
 */
export function normalizeModelGroupIDs(input?: Array<number | string> | null): number[] {
  if (!Array.isArray(input) || input.length === 0) return []
  const ids = new Set<number>()
  for (const raw of input) {
    const id = typeof raw === 'number' ? raw : Number(String(raw ?? '').trim())
    if (!Number.isInteger(id) || id <= 0) continue
    ids.add(id)
  }
  return Array.from(ids).sort((left, right) => left - right)
}

/**
 * 计算分组的展示平台：优先使用“分组平台配置”中的实际平台覆盖，否则沿用分组原始平台。
 *
 * 导出给配置页做分组平台配色用。配色必须取「有效平台」而不是 `group.platform`：
 * 平台覆盖会把分组挂到另一个平台下，按原始平台着色会与 listBindableGroups 的过滤
 * 结果自相矛盾 —— 分组能出现在候选列表里，却标着另一个平台的颜色。
 */
export function effectiveGroupPlatform(group: ModelSquareUserGroup, platformOverrides: Map<string, string>) {
  const overridden = platformOverrides.get(String(group.id))
  return normalizePlatform(overridden || group.platform)
}

function isGroupCompatible(group: ModelSquareUserGroup | undefined, platform: string, platformOverrides: Map<string, string>) {
  if (!group) return false
  const groupPlatform = effectiveGroupPlatform(group, platformOverrides)
  return groupPlatform === platform || groupPlatform === 'composite'
}

/**
 * 列出某个平台可以绑定的分组。
 *
 * 配置页的分组绑定控件用它，而不是在页面里另写一套平台判断：这里与展示页
 * 复用同一个 isGroupCompatible，两边口径一旦分叉，就会出现「配置页能绑、
 * 展示页却筛不到」这种只在保存后才暴露的错配。
 */
export function listBindableGroups(
  groups: ModelSquareUserGroup[],
  platform: string,
  platformOverrides = new Map<string, string>()
): ModelSquareUserGroup[] {
  const normalized = normalizePlatform(platform)
  if (!normalized) return []
  return groups.filter(group => isGroupCompatible(group, normalized, platformOverrides))
}

function configuredPrices(
  model: ModelSquarePlatformModelConfig,
  referencePrice?: ModelSquareOfficialPricing,
  rateMultiplier = 1
): Partial<ModelSquareModel> {
  const prices: Partial<ModelSquareModel> = {}
  for (const field of TOKEN_PRICE_FIELDS) {
    const configuredPrice = displayTokenPrice(model[field])
    const officialPrice = displayOfficialReferencePrice(referencePrice?.[field])
    assignPrice(prices, field, multiplyPrice(configuredPrice ?? officialPrice, rateMultiplier))
  }
  for (const field of REQUEST_PRICE_FIELDS) {
    // 按张（每张图一口价）不随分组倍率缩放：它是固定单价，乘倍率没有业务含义，直接原样展示。
    assignPrice(prices, field, displayRequestPrice(model[field]))
  }
  return prices
}

function minimumGroupRateMultiplier(groups: ModelSquareUserGroup[], platform: string, platformOverrides: Map<string, string>) {
  const rates = groups
    .filter(group => {
      const groupPlatform = effectiveGroupPlatform(group, platformOverrides)
      return groupPlatform === platform || groupPlatform === 'composite'
    })
    .map(group => toFiniteNumber(group.rate_multiplier))
    .filter((rate): rate is number => rate != null && rate >= 0)
  return rates.length > 0 ? Math.min(...rates) : 1
}

function multiplyPrice(price: number | undefined, multiplier: number) {
  return price == null ? undefined : price * multiplier
}

function assignPrice(target: Partial<ModelSquareModel>, field: PriceField, value: number | null | undefined) {
  if (value != null) target[field] = value
}

function displayTokenPrice(value: unknown): number | undefined {
  const price = toFiniteNumber(value)
  return price == null ? undefined : price * PER_MILLION
}

function displayOfficialReferencePrice(value: unknown): number | undefined {
  const price = toFiniteNumber(value)
  return price != null && price > 0 ? price * PER_MILLION : undefined
}

function displayRequestPrice(value: unknown): number | undefined {
  return toFiniteNumber(value)
}

function toFiniteNumber(value: unknown): number | undefined {
  if (value == null || value === '') return undefined
  const price = Number(value)
  return Number.isFinite(price) ? price : undefined
}

function normalizePlatform(value: string | undefined) {
  return (value || '').trim().toLowerCase()
}

function modelMode(
  modelName: string,
  modelConfig?: ModelSquarePlatformModelConfig,
  referencePrice?: ModelSquareOfficialPricing
): string {
  if (hasAnyPrice(modelConfig, ['image_input_price', 'image_output_price']) || hasReferenceImagePrice(referencePrice)) return 'image_generation'
  const name = modelName.toLowerCase()
  if (name.includes('embedding')) return 'embedding'
  if (name.includes('response')) return 'responses'
  return 'chat'
}

function hasAnyPrice(modelConfig: ModelSquarePlatformModelConfig | undefined, fields: PriceField[]) {
  return fields.some(field => modelConfig?.[field] != null)
}

function hasReferenceImagePrice(referencePrice?: ModelSquareOfficialPricing) {
  return ['image_input_price', 'image_output_price'].some(field => displayOfficialReferencePrice(referencePrice?.[field as TokenPriceField]) != null)
}

export async function getModelSquare(): Promise<AdminModelSquareResult> {
  const config = await getModelSquareConfig()
  if (!hasConfiguredModels(config)) {
    return buildConfiguredModelSquareResult(config, [])
  }

  const [localGroups, referencePrices, platformOverrides] = await Promise.all([
    getAllIncludingInactive(),
    listReferencePrices(config),
    listGroupPlatformOverrides(),
  ])
  return buildConfiguredModelSquareResult(config, localGroups, referencePrices, platformOverrides)
}

function hasConfiguredModels(config: ModelSquareConfigPayload) {
  return (config.platforms || []).some(platform => (platform.models || []).some(model => model.id?.trim()))
}

/**
 * 加载“分组平台配置”中的实际平台覆盖（分组 ID -> 展示平台）。
 * 模型广场按覆盖后的平台过滤分组归属，与模型监控保持一致；加载失败时沿用分组原始平台。
 */
async function listGroupPlatformOverrides(): Promise<Map<string, string>> {
  try {
    const items = await listLLMMonitorGroupPlatformOverrides()
    return new Map(items.map(item => [String(item.id), normalizePlatform(item.effective_platform)]))
  } catch {
    return new Map()
  }
}

async function listReferencePrices(config: ModelSquareConfigPayload): Promise<Map<string, ModelSquareOfficialPricing>> {
  const modelIDs = new Set<string>()
  for (const platform of config.platforms || []) {
    for (const model of platform.models || []) {
      const id = model.id?.trim()
      if (id && hasMissingConfiguredTokenPrice(model)) modelIDs.add(id)
    }
  }

  const entries = await Promise.all(Array.from(modelIDs).map(async id => {
    try {
      const price = await getModelSquareReferencePricing(id)
      return price.found ? [referencePricingKey(id), price] as const : undefined
    } catch {
      return undefined
    }
  }))

  return new Map(entries.filter((entry): entry is readonly [string, ModelSquareOfficialPricing] => entry != null))
}

function hasMissingConfiguredTokenPrice(model: ModelSquarePlatformModelConfig) {
  return TOKEN_PRICE_FIELDS.some(field => toFiniteNumber(model[field]) == null)
}

function referencePricingKey(modelID: string) {
  return modelID.trim().toLowerCase()
}

/**
 * 计算「模型属于哪些分组」所需的外部上下文。
 *
 * 分组归属是配置里存的字段，但展示它需要分组名单（把 ID 解析成名字、平台与倍率），
 * 所以配置页要单独拿到这份数据。这里刻意不复用 getModelSquare()：
 * 它内部会重新 GET 一次已保存的配置，拿到的永远是上次保存的状态，
 * 看不到正在编辑、尚未保存的模型。
 *
 * 渠道数据不在这里 —— 可用性已改为只看分组绑定，不再需要渠道。
 * 参考价也不在这里拉：配置页自己按模型 ID 查询并缓存，避免每个模型多发一次请求。
 */
export interface ModelSquareGroupContext {
  groups: ModelSquareUserGroup[]
  platformOverrides: Map<string, string>
}

export async function loadModelSquareGroupContext(): Promise<ModelSquareGroupContext> {
  const [groups, platformOverrides] = await Promise.all([
    getAllIncludingInactive(),
    listGroupPlatformOverrides(),
  ])
  return { groups, platformOverrides }
}

export const modelSquareAPI = {
  get: getModelSquare,
  loadGroupContext: loadModelSquareGroupContext,
}

export default modelSquareAPI

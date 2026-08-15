import { getAllIncludingInactive } from './groups'
import { list, type Channel } from './channels'
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
 * 模型广场用户接口返回的最小渠道结构（只读聚合数据，不含管理端敏感字段）。
 * buildConfiguredModelSquareResult 只需这些字段即可完成可用性与分组归属计算。
 */
export interface ModelSquareUserChannel {
  id?: number | string
  status?: string
  group_ids?: Array<number | string>
  model_pricing?: Array<{
    platform?: string
    models?: string[]
  }>
  model_mapping?: Record<string, Record<string, string>>
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

type ModelMatch = {
  available: boolean
  groupIDs: Array<number | string>
}

/**
 * 使用模型广场配置生成展示目录，并仅用渠道与分组补充可用状态和倍率信息。
 */
export function buildConfiguredModelSquareResult(
  config: ModelSquareConfigPayload,
  channels: ModelSquareUserChannel[],
  localGroups: ModelSquareUserGroup[],
  referencePrices = new Map<string, ModelSquareOfficialPricing>(),
  platformOverrides = new Map<string, string>()
): AdminModelSquareResult {
  const groupById = new Map(localGroups.map(group => [String(group.id), group]))
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

      const match = findConfiguredModelMatch(platform, modelID, channels, groupById, platformOverrides)
      const minimumRateMultiplier = minimumGroupRateMultiplier(localGroups, platform, platformOverrides)

      models.push({
        id: modelID,
        display_name: modelConfig.display_name?.trim() || undefined,
        provider: providerName,
        platform,
        available: match.available,
        mode: modelMode(modelID, modelConfig, referencePrices.get(referencePricingKey(modelID))),
        rate_multiplier: minimumRateMultiplier,
        ...configuredPrices(modelConfig, referencePrices.get(referencePricingKey(modelID)), minimumRateMultiplier),
        group_ids: match.groupIDs,
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

function findConfiguredModelMatch(
  platform: string,
  modelID: string,
  channels: ModelSquareUserChannel[],
  groupById: Map<string, ModelSquareUserGroup>,
  platformOverrides: Map<string, string>
): ModelMatch {
  const usedGroupIDs = new Set<number | string>()
  let available = false

  for (const channel of channels) {
    if (!channelSupportsModel(channel, platform, modelID)) continue

    const compatibleGroupIDs = (channel.group_ids || []).filter(groupID => isGroupCompatible(groupById.get(String(groupID)), platform, platformOverrides))
    if (compatibleGroupIDs.length === 0) continue

    for (const groupID of compatibleGroupIDs) usedGroupIDs.add(groupID)
    if (channel.status === 'active') available = true
  }

  return { available, groupIDs: Array.from(usedGroupIDs) }
}

function channelSupportsModel(channel: ModelSquareUserChannel, platform: string, modelID: string) {
  const key = modelKey(platform, modelID)

  for (const pricing of channel.model_pricing || []) {
    const pricingPlatform = normalizePlatform(pricing.platform || 'anthropic')
    for (const modelName of pricing.models || []) {
      if (modelKey(pricingPlatform, modelName) === key) return true
    }
  }

  for (const [mappingPlatform, mapping] of Object.entries(channel.model_mapping || {})) {
    if (normalizePlatform(mappingPlatform) !== platform) continue
    for (const [sourceModel, targetModel] of Object.entries(mapping || {})) {
      if (modelKey(platform, sourceModel) === key || modelKey(platform, targetModel) === key) return true
    }
  }

  return false
}

/**
 * 计算分组的展示平台：优先使用“分组平台配置”中的实际平台覆盖，否则沿用分组原始平台。
 */
function effectiveGroupPlatform(group: ModelSquareUserGroup, platformOverrides: Map<string, string>) {
  const overridden = platformOverrides.get(String(group.id))
  return normalizePlatform(overridden || group.platform)
}

function isGroupCompatible(group: ModelSquareUserGroup | undefined, platform: string, platformOverrides: Map<string, string>) {
  if (!group) return false
  const groupPlatform = effectiveGroupPlatform(group, platformOverrides)
  return groupPlatform === platform || groupPlatform === 'composite'
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
    assignPrice(prices, field, multiplyPrice(displayRequestPrice(model[field]), rateMultiplier))
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

function modelKey(platform: string, modelName: string): string {
  return `${normalizePlatform(platform)}:${modelName.trim()}`.toLowerCase()
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
    return buildConfiguredModelSquareResult(config, [], [])
  }

  const [channels, localGroups, referencePrices, platformOverrides] = await Promise.all([
    listAllChannels(),
    getAllIncludingInactive(),
    listReferencePrices(config),
    listGroupPlatformOverrides(),
  ])
  return buildConfiguredModelSquareResult(config, channels, localGroups, referencePrices, platformOverrides)
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

async function listAllChannels(): Promise<Channel[]> {
  const pageSize = 1000
  const firstPage = await list(1, pageSize)
  const channels = [...firstPage.items]
  const total = Number(firstPage.total)
  const pageCount = Number.isFinite(total) && total > 0 ? Math.ceil(total / pageSize) : 1

  for (let page = 2; page <= pageCount; page += 1) {
    const nextPage = await list(page, pageSize)
    channels.push(...nextPage.items)
  }

  return channels
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

export const modelSquareAPI = {
  get: getModelSquare,
}

export default modelSquareAPI

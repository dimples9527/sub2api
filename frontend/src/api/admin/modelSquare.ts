import { getAllIncludingInactive, getModelsListCandidates } from './groups'
import { list, type Channel, type ChannelModelPricing } from './channels'
import type { AdminGroup } from '@/types'

export interface ModelSquareGroup {
  id: number | string
  name: string
  rate_multiplier?: number
}

export interface ModelSquareModel {
  id: string
  provider?: string
  available?: boolean
  mode?: string
  input_price?: number | string
  output_price?: number | string
  cache_read_price?: number | string
  cache_create_price?: number | string
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

export interface AdminModelSquareResult {
  provider_slug: string
  provider_name: string
  provider_type: string
  payload: ModelSquarePayload
}

const PER_MILLION = 1_000_000

type AggregatedModel = ModelSquareModel & {
  pricingRate: number
  hasPricing: boolean
  pricingChannelActive: boolean
}

type CandidateModelsByGroup = Map<string, Set<string>>

/**
 * 将本地渠道与分组配置压平为管理端模型广场的旧数据结构。
 * 模型来自渠道的 model_pricing / model_mapping，价格来自渠道定价配置。
 */
export function buildLocalModelSquareResult(
  channels: Channel[],
  localGroups: AdminGroup[],
  candidateModelsByGroup?: CandidateModelsByGroup
): AdminModelSquareResult {
  const groupById = new Map(localGroups.map(group => [String(group.id), group]))
  const usedGroupIDs = new Set<string>()
  const models = new Map<string, AggregatedModel>()

  for (const channel of channels) {
    const channelActive = channel.status === 'active'
    const pricingByModel = pricingIndex(channel.model_pricing)

    for (const pricing of channel.model_pricing || []) {
      const platform = pricing.platform || 'anthropic'
      for (const modelName of pricing.models || []) {
        addModel(
          modelName,
          platform,
          pricing,
          channelActive,
          channel.group_ids,
          groupById,
          usedGroupIDs,
          models,
          candidateModelsByGroup
        )
      }
    }

    for (const [platform, mapping] of Object.entries(channel.model_mapping || {})) {
      for (const [sourceModel, targetModel] of Object.entries(mapping || {})) {
        const pricing = pricingByModel.get(modelKey(platform, targetModel)) || pricingByModel.get(modelKey(platform, sourceModel))
        addModel(
          sourceModel,
          platform,
          pricing,
          channelActive,
          channel.group_ids,
          groupById,
          usedGroupIDs,
          models,
          candidateModelsByGroup
        )
      }
    }
  }

  const groups: ModelSquareGroup[] = localGroups
    .filter(group => usedGroupIDs.has(String(group.id)))
    .map(group => ({
    id: group.id,
    name: group.name,
    rate_multiplier: group.rate_multiplier
  }))

  const publicModels = Array.from(models.values()).map(({ pricingRate: _pricingRate, hasPricing: _hasPricing, pricingChannelActive: _pricingChannelActive, ...model }) => model)
  return {
    provider_slug: 'local',
    provider_name: '本地渠道配置',
    provider_type: 'local',
    payload: { groups, models: publicModels }
  }
}

function modelKey(platform: string, modelName: string): string {
  return `${platform}:${modelName}`.toLowerCase()
}

function pricingIndex(entries: ChannelModelPricing[]): Map<string, ChannelModelPricing> {
  const index = new Map<string, ChannelModelPricing>()
  for (const entry of entries || []) {
    const platform = entry.platform || 'anthropic'
    for (const modelName of entry.models || []) {
      index.set(modelKey(platform, modelName), entry)
    }
  }
  return index
}

function addModel(
  modelName: string,
  platform: string,
  pricing: ChannelModelPricing | undefined,
  channelActive: boolean,
  channelGroupIDs: number[],
  groupById: Map<string, AdminGroup>,
  usedGroupIDs: Set<string>,
  models: Map<string, AggregatedModel>,
  candidateModelsByGroup?: CandidateModelsByGroup
) {
  const normalizedName = modelName.trim()
  const normalizedPlatform = platform.trim() || 'anthropic'
  if (!normalizedName) return

  const groupIDs = channelGroupIDs.filter(groupID => {
    const group = groupById.get(String(groupID))
    return !group || group.platform === normalizedPlatform || group.platform === 'composite'
  })
  if (groupIDs.length === 0) return

  const candidateGroupIDs = groupIDs.filter(groupID => {
    const candidates = candidateModelsByGroup?.get(String(groupID))
    return candidates === undefined || candidates.has(normalizedName)
  })
  if (candidateGroupIDs.length === 0) return

  for (const groupID of candidateGroupIDs) usedGroupIDs.add(String(groupID))

  const key = modelKey(normalizedPlatform, normalizedName)
  const current = models.get(key)
  const hasPricing = pricing != null
  const pricingRate = candidateGroupIDs.reduce((lowest, groupID) => {
    const rate = Number(groupById.get(String(groupID))?.rate_multiplier)
    return Number.isFinite(rate) ? Math.min(lowest, rate) : lowest
  }, Number.POSITIVE_INFINITY)
  const shouldUsePricing = hasPricing && (
    !current?.hasPricing ||
    (channelActive && !current.pricingChannelActive) ||
    (channelActive === current.pricingChannelActive && pricingRate < current.pricingRate)
  )

  if (!current) {
    models.set(key, {
      id: normalizedName,
      provider: normalizedPlatform,
      available: channelActive,
      mode: modelMode(normalizedName, pricing),
      input_price: perMillion(pricing?.input_price),
      output_price: perMillion(pricing?.output_price),
      cache_read_price: perMillion(pricing?.cache_read_price),
      cache_create_price: perMillion(pricing?.cache_write_price),
      group_ids: candidateGroupIDs,
      pricingRate,
      hasPricing,
      pricingChannelActive: channelActive
    })
    return
  }

  current.available = current.available !== false || channelActive
  current.group_ids = Array.from(new Set([...(current.group_ids || []), ...candidateGroupIDs]))
  if (!shouldUsePricing) return

  current.input_price = perMillion(pricing?.input_price)
  current.output_price = perMillion(pricing?.output_price)
  current.cache_read_price = perMillion(pricing?.cache_read_price)
  current.cache_create_price = perMillion(pricing?.cache_write_price)
  current.pricingRate = pricingRate
  current.hasPricing = true
  current.pricingChannelActive = channelActive
}

function normalizeModelNames(models: string[] | undefined): Set<string> {
  const normalized = new Set<string>()
  for (const model of models || []) {
    const value = model.trim()
    if (!value) continue
    normalized.add(value)
  }
  return normalized
}

async function buildCandidateModelsByGroup(localGroups: AdminGroup[]): Promise<CandidateModelsByGroup> {
  const index: CandidateModelsByGroup = new Map()
  await Promise.all(localGroups.map(async group => {
    const key = String(group.id)
    const config = group.models_list_config
    const configuredModels = config?.enabled ? normalizeModelNames(config.models) : new Set<string>()
    if (configuredModels.size > 0) {
      index.set(key, configuredModels)
      return
    }

    const models = await getModelsListCandidates(group.id, group.platform).catch(() => undefined)
    if (models) index.set(key, normalizeModelNames(models))
  }))
  return index
}

function perMillion(value: number | null | undefined): number | undefined {
  return value == null ? undefined : value * PER_MILLION
}

function modelMode(modelName: string, pricing?: ChannelModelPricing): string {
  const billingMode = pricing?.billing_mode
  if (billingMode === 'image') return 'image_generation'

  const name = modelName.toLowerCase()
  if (name.includes('embedding')) return 'embedding'
  if (name.includes('response')) return 'responses'
  return 'chat'
}

export async function getModelSquare(): Promise<AdminModelSquareResult> {
  const [channelResult, localGroups] = await Promise.all([
    listAllChannels(),
    getAllIncludingInactive()
  ])
  const candidateModelsByGroup = await buildCandidateModelsByGroup(localGroups)
  return buildLocalModelSquareResult(channelResult, localGroups, candidateModelsByGroup)
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

export const modelSquareAPI = {
  get: getModelSquare
}

export default modelSquareAPI

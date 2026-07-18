export type SupplierGroupRateCode =
  | 'unmatched'
  | 'inactive'
  | 'invalid'
  | 'inverted'
  | 'equal'
  | 'low'
  | 'normal'

export type SupplierUpstreamRateBand = 'invalid' | 'low' | 'standard' | 'elevated' | 'high'

export interface SupplierGroupRateInput {
  localGroupID?: number
  upstreamRate: number
  localRate?: number
  localStatus: string
}

export interface SupplierGroupRateInsight {
  code: SupplierGroupRateCode
  label: string
  ratio: number | null
}

const RATE_EPSILON = 1e-9
const LOW_MARGIN_THRESHOLD = 1.1

export function getSupplierUpstreamRateBand(upstreamRate: number): SupplierUpstreamRateBand {
  if (!Number.isFinite(upstreamRate) || upstreamRate <= 0) return 'invalid'
  if (upstreamRate <= 1) return 'low'
  if (upstreamRate <= 2) return 'standard'
  if (upstreamRate <= 5) return 'elevated'
  return 'high'
}

export function getSupplierGroupRateInsight(input: SupplierGroupRateInput): SupplierGroupRateInsight {
  if (!input.localGroupID) {
    return { code: 'unmatched', label: '未匹配', ratio: null }
  }

  const validRates = Number.isFinite(input.upstreamRate)
    && input.upstreamRate > 0
    && Number.isFinite(input.localRate)
    && Number(input.localRate) > 0
  const ratio = validRates ? Number(input.localRate) / input.upstreamRate : null

  if (input.localStatus === 'inactive') {
    return { code: 'inactive', label: '本地已停用', ratio }
  }
  if (!validRates || ratio === null) {
    return { code: 'invalid', label: '数据异常', ratio: null }
  }
  if (Number(input.localRate) < input.upstreamRate - RATE_EPSILON) {
    return { code: 'inverted', label: '倒挂风险', ratio }
  }
  if (Math.abs(Number(input.localRate) - input.upstreamRate) <= RATE_EPSILON) {
    return { code: 'equal', label: '倍率持平', ratio }
  }
  if (ratio < LOW_MARGIN_THRESHOLD) {
    return { code: 'low', label: '收益偏低', ratio }
  }
  return { code: 'normal', label: '正常', ratio }
}

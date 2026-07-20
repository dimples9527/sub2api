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
  delta: number | null
}

const RATE_EPSILON = 1e-9
const LOW_MARGIN_MULTIPLIER = 1.1

type DecimalRate = number | string

interface DecimalParts {
  coefficient: bigint
  scale: number
}

export function getSupplierUpstreamRateBand(upstreamRate: number): SupplierUpstreamRateBand {
  if (!Number.isFinite(upstreamRate) || upstreamRate <= 0) return 'invalid'
  if (upstreamRate <= 1) return 'low'
  if (upstreamRate <= 2) return 'standard'
  if (upstreamRate <= 5) return 'elevated'
  return 'high'
}

export function getSupplierGroupRateInsight(input: SupplierGroupRateInput): SupplierGroupRateInsight {
  if (!input.localGroupID) {
    return { code: 'unmatched', label: '未匹配', delta: null }
  }

  const validRates = Number.isFinite(input.upstreamRate)
    && input.upstreamRate > 0
    && Number.isFinite(input.localRate)
    && Number(input.localRate) > 0
  const delta = validRates ? Number(input.localRate) - input.upstreamRate : null

  if (input.localStatus === 'inactive') {
    return { code: 'inactive', label: '本地已停用', delta }
  }
  if (!validRates || delta === null) {
    return { code: 'invalid', label: '数据异常', delta: null }
  }
  if (delta < -RATE_EPSILON) {
    return { code: 'inverted', label: '倒挂风险', delta }
  }
  if (Math.abs(delta) <= RATE_EPSILON) {
    return { code: 'equal', label: '倍率持平', delta }
  }
  if (Number(input.localRate) < input.upstreamRate * LOW_MARGIN_MULTIPLIER) {
    return { code: 'low', label: '收益偏低', delta }
  }
  return { code: 'normal', label: '正常', delta }
}

export function formatSupplierGroupRateDelta(localRate: DecimalRate | undefined, upstreamRate: DecimalRate | undefined): string {
  const local = parseDecimalRate(localRate)
  const upstream = parseDecimalRate(upstreamRate)
  if (!local || !upstream) return '-'

  const scale = Math.max(local.scale, upstream.scale)
  const localCoefficient = local.coefficient * 10n ** BigInt(scale - local.scale)
  const upstreamCoefficient = upstream.coefficient * 10n ** BigInt(scale - upstream.scale)
  const delta = localCoefficient - upstreamCoefficient
  if (delta === 0n) return '0'

  const sign = delta < 0n ? '-' : '+'
  const digits = (delta < 0n ? -delta : delta).toString().padStart(scale + 1, '0')
  if (scale === 0) return `${sign}${digits}`

  const integerPart = digits.slice(0, -scale) || '0'
  const fractionPart = digits.slice(-scale).replace(/0+$/, '')
  return `${sign}${integerPart}${fractionPart ? `.${fractionPart}` : ''}`
}

function parseDecimalRate(value: DecimalRate | undefined): DecimalParts | null {
  if (value == null) return null
  const raw = String(value).trim()
  const match = raw.match(/^([+-]?)(\d+(?:\.\d*)?|\.\d+)(?:e([+-]?\d+))?$/i)
  if (!match) return null

  const exponent = Number(match[3] || 0)
  if (!Number.isSafeInteger(exponent) || Math.abs(exponent) > 1000) return null

  const [integerPart, fractionalPart = ''] = match[2].split('.')
  const digits = `${integerPart}${fractionalPart}`.replace(/^0+(?=\d)/, '') || '0'
  const unscaled = BigInt(digits) * (match[1] === '-' ? -1n : 1n)
  const scaleBeforeExponent = fractionalPart.length - exponent

  return {
    coefficient: unscaled * 10n ** BigInt(Math.max(0, -scaleBeforeExponent)),
    scale: Math.max(0, scaleBeforeExponent),
  }
}

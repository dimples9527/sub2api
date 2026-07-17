export type UpstreamMonitorTone = 'green' | 'yellow' | 'red'

export interface UpstreamMonitorTrendPoint {
  tone: UpstreamMonitorTone
  statusText: string
  time: string
  latency: number
  availability: number
}

export interface UpstreamMonitorTrendRow {
  provider: string
  availability: number
  latency: number
  time: string
  trend: UpstreamMonitorTrendPoint[]
}

export function normalizeUpstreamMonitorGroupKey(value: unknown): string {
  return String(value ?? '').trim().toLowerCase().replace(/[\s_-]+/g, '')
}

export function buildUpstreamMonitorTrendIndex(payload: unknown): Map<string, UpstreamMonitorTrendRow> {
  const rows = normalizeMonitorPayload(payload)
  const index = new Map<string, UpstreamMonitorTrendRow>()
  rows.forEach((item, itemIndex) => {
    const row = normalizeMonitorItem(item, itemIndex)
    if (!row.provider) return
    for (const key of rowKeys(row.provider, item)) {
      if (key && !index.has(key)) {
        index.set(key, row)
      }
    }
  })
  return index
}

function normalizeMonitorPayload(payload: unknown): unknown[] {
  if (Array.isArray(payload)) return payload
  if (!isRecord(payload)) return []
  const candidates = [
    payload.groups,
    payload.data,
    payload.items,
    payload.list,
    payload.status,
    payload.statuses,
    payload.providers,
    payload.services,
    payload.result
  ]
  for (const candidate of candidates) {
    if (Array.isArray(candidate)) return candidate
    const nested = normalizeMonitorPayload(candidate)
    if (nested.length) return nested
  }
  return []
}

function normalizeMonitorItem(item: unknown, itemIndex: number): UpstreamMonitorTrendRow {
  const object = isRecord(item) ? item : {}
  const layer = firstRecord(object.layers)
  const current = asRecord(layer.current_status) || {}
  const timeline = Array.isArray(layer.timeline) ? layer.timeline : []
  const latestPoint = asRecord(timeline[timeline.length - 1]) || {}
  const provider = stringValue(pickValue(object, ['provider', 'provider_name', 'providerName', 'name', 'title', 'service_provider']), `provider ${itemIndex + 1}`)
  const availabilityValues = timeline.map((point) => {
    const row = asRecord(point) || {}
    return numberValue(row.availability, statusAvailability(row.status))
  })
  const availability = availabilityValues.length
    ? round2(availabilityValues.reduce((sum, value) => sum + value, 0) / availabilityValues.length)
    : numberValue(
      pickValue(object, ['availability', 'available_rate', 'availableRate', 'success_rate', 'successRate', 'rate_percent', 'uptime']),
      statusAvailability(pickValue(current, ['status']) ?? object.current_status)
    )
  const latency = numberValue(
    pickValue(current, ['latency'])
      ?? pickValue(latestPoint, ['latency'])
      ?? pickValue(object, ['latency', 'latency_ms', 'latencyMs', 'response_time', 'responseTime', 'last_latency']),
    0
  )
  const time = formatCheckedTime(
    pickValue(current, ['timestamp'])
      ?? pickValue(latestPoint, ['time', 'timestamp'])
      ?? pickValue(object, ['time', 'checked_at', 'checkedAt', 'last_check', 'lastCheck', 'last_monitor', 'lastMonitor'])
  )
  const rawTrend = timeline.length
    ? [...timeline].sort((a, b) => pointTimestamp(a) - pointTimestamp(b)).slice(-18)
    : fallbackTrend(object, availability)
  const trend = rawTrend.map((point, index) => normalizeTrendPoint(point, { availability, latency, time }, index))
  return { provider, availability, latency, time, trend }
}

function rowKeys(provider: string, item: unknown): string[] {
  const object = isRecord(item) ? item : {}
  const keys = [
    provider,
    pickValue(object, ['group_name', 'groupName', 'group']),
    pickValue(object, ['provider_name', 'providerName'])
  ]
  return keys.map(normalizeUpstreamMonitorGroupKey).filter(Boolean)
}

function normalizeTrendPoint(point: unknown, row: { availability: number; latency: number; time: string }, fallbackIndex: number): UpstreamMonitorTrendPoint {
  if (isRecord(point)) {
    if ('tone' in point && !('status' in point) && !('availability' in point)) {
      const tone = normalizeTone(point.tone)
      return {
        tone,
        statusText: statusTextFromTone(tone),
        time: pointTime(point, row.time),
        latency: row.latency,
        availability: row.availability
      }
    }
    const status = pickValue(point, ['status'])
    const latency = numberValue(pickValue(point, ['latency', 'latency_ms', 'latencyMs']), row.latency)
    return {
      tone: pointTone(point),
      statusText: statusText(status),
      time: pointTime(point, row.time),
      latency,
      availability: numberValue(pickValue(point, ['availability']), statusAvailability(status))
    }
  }
  const tone = normalizeTone(point)
  return {
    tone,
    statusText: statusTextFromTone(tone),
    time: row.time || `#${fallbackIndex + 1}`,
    latency: row.latency,
    availability: row.availability
  }
}

function fallbackTrend(item: Record<string, unknown>, availability: number): unknown[] {
  const rawTrend = pickValue(item, ['trend', 'history', 'availability_trend', 'availabilityTrend', 'status_history', 'statusHistory'])
  if (Array.isArray(rawTrend) && rawTrend.length) return rawTrend.slice(-18)
  const tone: UpstreamMonitorTone = availability >= 75 ? 'green' : availability >= 30 ? 'yellow' : 'red'
  return Array.from({ length: 18 }, () => tone)
}

function pointTone(point: Record<string, unknown>): UpstreamMonitorTone {
  const status = numberValue(pickValue(point, ['status']), Number.NaN)
  const availability = numberValue(pickValue(point, ['availability']), statusAvailability(status))
  if (status === 1 || availability >= 75) return 'green'
  if (status === 2 || availability >= 30) return 'yellow'
  return 'red'
}

function normalizeTone(value: unknown): UpstreamMonitorTone {
  const text = String(value ?? '').toLowerCase()
  if (text.includes('green') || text.includes('ok') || text.includes('success') || text === '1' || text === 'true') return 'green'
  if (text.includes('yellow') || text.includes('warn') || text.includes('partial') || text === '2') return 'yellow'
  return 'red'
}

function statusAvailability(status: unknown): number {
  const value = Number(status)
  if (value === 1) return 100
  if (value === 2) return 70
  return 0
}

function statusText(status: unknown): string {
  switch (Number(status)) {
    case 1:
      return 'OK'
    case 2:
      return 'Degraded'
    case 0:
      return 'Down'
    default:
      return 'Unknown'
  }
}

function statusTextFromTone(tone: UpstreamMonitorTone): string {
  if (tone === 'green') return 'OK'
  if (tone === 'yellow') return 'Degraded'
  return 'Down'
}

function pointTime(point: Record<string, unknown>, fallback: string): string {
  const value = pickValue(point, ['displayTimestamp', 'time', 'timestamp', 'checked_at', 'checkedAt'])
  if (typeof value === 'string' && value.trim() && !/^\d+$/.test(value.trim())) {
    return value.trim()
  }
  return formatCheckedTime(value, fallback)
}

function pointTimestamp(point: unknown): number {
  return numberValue(pickValue(asRecord(point) || {}, ['timestamp', 'checked_at', 'checkedAt']), 0)
}

function formatCheckedTime(value: unknown, fallback = '--:--'): string {
  const numeric = numberValue(value, 0)
  if (numeric > 0) {
    const milliseconds = numeric < 10000000000 ? numeric * 1000 : numeric
    return new Date(milliseconds).toLocaleTimeString('zh-CN', {
      hour: '2-digit',
      minute: '2-digit'
    })
  }
  const text = String(value ?? '').trim()
  return text ? text.slice(-5) : fallback
}

function pickValue(object: Record<string, unknown>, keys: string[]): unknown
function pickValue(object: unknown, keys: string[]): unknown
function pickValue(object: unknown, keys: string[]): unknown {
  if (!isRecord(object)) return undefined
  for (const key of keys) {
    if (object[key] !== undefined && object[key] !== null && object[key] !== '') return object[key]
  }
  return undefined
}

function firstRecord(value: unknown): Record<string, unknown> {
  if (Array.isArray(value) && isRecord(value[0])) return value[0]
  return {}
}

function asRecord(value: unknown): Record<string, unknown> | null {
  return isRecord(value) ? value : null
}

function isRecord(value: unknown): value is Record<string, unknown> {
  return typeof value === 'object' && value !== null && !Array.isArray(value)
}

function stringValue(value: unknown, fallback = ''): string {
  const text = String(value ?? '').trim()
  return text || fallback
}

function numberValue(value: unknown, fallback: number): number {
  const n = Number(value)
  return Number.isFinite(n) ? n : fallback
}

function round2(value: number): number {
  return Math.round(value * 100) / 100
}

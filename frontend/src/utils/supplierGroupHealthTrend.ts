import type {
  SupplierProviderGroupHealthTrend,
  SupplierProviderGroupHealthTrendPoint,
} from '@/api/admin/supplierProviderData'
import type { SupplierGroupMonitorTrendRow } from '@/utils/supplierGroupMonitorTrend'

export function buildSupplierGroupHealthTrendIndex(
  trends: SupplierProviderGroupHealthTrend[]
): Map<number, SupplierGroupMonitorTrendRow> {
  return new Map(trends.map(trend => [trend.group_id, toSupplierGroupHealthTrendRow(trend)]))
}

function toSupplierGroupHealthTrendRow(trend: SupplierProviderGroupHealthTrend): SupplierGroupMonitorTrendRow {
  return {
    provider: '供应商账号健康守护',
    availability: trend.availability,
    latency: trend.latency,
    time: formatTrendTime(trend.time),
    trend: trend.trend.map(toTrendPoint),
  }
}

function toTrendPoint(point: SupplierProviderGroupHealthTrendPoint) {
  return {
    tone: point.tone,
    statusText: point.tested_account_count > 0 ? `已测 ${point.tested_account_count} 个账号` : '暂无检测',
    time: formatTrendTime(point.time),
    latency: point.latency,
    availability: point.availability,
  }
}

function formatTrendTime(value: string): string {
  const date = new Date(value)
  if (Number.isNaN(date.getTime())) return '--:--'
  return date.toLocaleTimeString('zh-CN', { hour: '2-digit', minute: '2-digit' })
}
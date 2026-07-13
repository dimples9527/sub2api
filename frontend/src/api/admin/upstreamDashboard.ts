import { apiClient } from '../client'

export type UpstreamDashboardRange = '24h' | '7d'
export type UpstreamDashboardSeverity = 'critical' | 'high' | 'medium' | 'low'

export interface UpstreamDashboardSummary {
  provider_count: number
  disabled_provider_count: number
  matched_account_count: number
  pending_account_count: number
  rate_risk_count: number
  model_count: number
}

export interface UpstreamDashboardStability {
  request_count: number
  success_count: number
  error_count: number
  success_rate: number
  error_rate: number
  p95_latency_ms: number
  health_score: number
}

export interface UpstreamDashboardCost {
  period_cost: number
  total_balance: number
  estimated_days?: number
  anomaly_providers: number
}

export interface UpstreamDashboardIssue {
  id: string
  type: string
  source: string
  severity: UpstreamDashboardSeverity
  entity_key: string
  title: string
  description: string
  impact_count: number
  action?: string
  target_path?: string
  detected_at: string
}

export interface UpstreamDashboardTask {
  key: string
  name: string
  enabled: boolean
  last_run_at?: string
  last_run_status?: string
  last_run_message?: string
  next_run_at?: string
  affected_count: number
  settings_path: string
}

export interface UpstreamDashboardProviderRanking {
  provider_slug: string
  provider_name: string
  balance: number
  cost: number
}

export interface UpstreamDashboardModelRanking {
  model: string
  requests: number
  cost: number
}

export interface UpstreamDashboardTrendPoint {
  date: string
  cost: number
}

export interface UpstreamDashboardWarning {
  source: string
  message: string
}

export interface UpstreamDashboardResponse {
  range: UpstreamDashboardRange
  summary: UpstreamDashboardSummary
  stability: UpstreamDashboardStability
  cost: UpstreamDashboardCost
  issues: UpstreamDashboardIssue[]
  tasks: UpstreamDashboardTask[]
  provider_rankings: UpstreamDashboardProviderRanking[]
  model_rankings: UpstreamDashboardModelRanking[]
  trends: UpstreamDashboardTrendPoint[]
  warnings?: UpstreamDashboardWarning[]
  generated_at: string
}

export async function get(range: UpstreamDashboardRange = '24h'): Promise<UpstreamDashboardResponse> {
  const { data } = await apiClient.get<UpstreamDashboardResponse>(
    '/admin/upstream-management/dashboard',
    { params: { range } }
  )
  return data
}

export const upstreamDashboardAPI = { get }

export default upstreamDashboardAPI

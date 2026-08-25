import { apiClient } from '../client'

export type SupplierCostReviewStatus = 'pending_review' | 'approved' | 'changed_after_approval'
export type SupplierCostReviewDecision = 'none' | 'upstream' | 'calculated' | 'manual'

export interface SupplierProviderCostReview {
  id: number
  provider_id: number
  provider_name: string
  stat_date: string
  upstream_cost?: number | null
  calculated_cost?: number | null
  auto_adopted_cost?: number | null
  final_cost?: number | null
  effective_cost: number
  cost_delta?: number | null
  effective_delta?: number | null
  status: SupplierCostReviewStatus
  decision_type: SupplierCostReviewDecision
  approved_by?: number | null
  approved_at?: string | null
  sync_count: number
  last_sync_run_id?: number | null
  last_synced_at?: string | null
  version: number
  created_at: string
  updated_at: string
}

export interface SupplierProviderCostReviewHistory {
  id: number
  review_id?: number | null
  provider_id: number
  stat_date: string
  event_type: 'sync' | 'approve'
  sync_run_id?: number | null
  upstream_cost?: number | null
  calculated_cost?: number | null
  auto_adopted_cost?: number | null
  final_cost?: number | null
  cost_delta?: number | null
  effective_delta?: number | null
  status: SupplierCostReviewStatus
  decision_type: SupplierCostReviewDecision
  manual_cost?: number | null
  operator_id?: number | null
  operated_at: string
}

export interface SupplierProviderCostReviewListParams {
  provider_id?: number
  keyword?: string
  start_date?: string
  end_date?: string
  status?: SupplierCostReviewStatus | ''
  page?: number
  page_size?: number
}

export interface SupplierProviderCostReviewListResult {
  items: SupplierProviderCostReview[]
  total: number
  page: number
  page_size: number
}

export interface SupplierProviderCostReviewApprovePayload {
  decision_type: Exclude<SupplierCostReviewDecision, 'none'>
  manual_cost?: number
  version: number
}

export interface SupplierProviderCostReviewBulkApprovePayload {
  items: Array<{ id: number; version: number }>
  decision_type: Exclude<SupplierCostReviewDecision, 'none'>
  manual_cost?: number
}

export interface SupplierProviderCostReviewBulkApproveResult {
  items: SupplierProviderCostReview[]
  count: number
}

export async function listSupplierProviderCostReviews(
  params: SupplierProviderCostReviewListParams = {}
): Promise<SupplierProviderCostReviewListResult> {
  const { data } = await apiClient.get<SupplierProviderCostReviewListResult>(
    '/admin/supplier-management/cost-reviews',
    { params }
  )
  return data
}

export async function listSupplierProviderCostReviewHistory(
  reviewId: number
): Promise<SupplierProviderCostReviewHistory[]> {
  const { data } = await apiClient.get<SupplierProviderCostReviewHistory[]>(
    `/admin/supplier-management/cost-reviews/${reviewId}/history`
  )
  return data
}

export async function approveSupplierProviderCostReview(
  reviewId: number,
  payload: SupplierProviderCostReviewApprovePayload
): Promise<SupplierProviderCostReview> {
  const { data } = await apiClient.post<SupplierProviderCostReview>(
    `/admin/supplier-management/cost-reviews/${reviewId}/approve`,
    payload
  )
  return data
}

export async function bulkApproveSupplierProviderCostReviews(
  payload: SupplierProviderCostReviewBulkApprovePayload
): Promise<SupplierProviderCostReviewBulkApproveResult> {
  const { data } = await apiClient.post<SupplierProviderCostReviewBulkApproveResult>(
    '/admin/supplier-management/cost-reviews/bulk-approve',
    payload
  )
  return data
}

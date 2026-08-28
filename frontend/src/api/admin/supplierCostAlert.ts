import { apiClient } from '../client'
import type { BasePaginationResponse } from '@/types'

export type SupplierCostAlertEventType = 'cost_overrun' | 'cost_recovered'
export type SupplierCostAlertEventStatus = 'active' | 'resolved'

export interface SupplierCostAlertSettings {
  amount: string
}

export interface SupplierCostAlertSettingsInput {
  amount: string
}

export interface SupplierCostAlertOverride {
  id: number
  provider_id: number
  enabled: boolean
  amount: string
  created_at: string
  updated_at: string
}

export interface SupplierCostAlertOverrideInput {
  provider_id: number
  enabled: boolean
  amount: string
}

export interface SupplierCostAlertEvent {
  id: number
  provider_id: number
  provider_code: string
  provider_name: string
  event_type: SupplierCostAlertEventType | string
  status: SupplierCostAlertEventStatus | string
  stat_date: string
  upstream_cost: string
  local_cost: string
  overrun_amount: string
  threshold: string
  observed_at: string
  resolved_at?: string
  last_seen_at: string
  created_at: string
  updated_at: string
}

export interface SupplierCostAlertEventListParams {
  provider_id?: number
  event_type?: SupplierCostAlertEventType | ''
  status?: SupplierCostAlertEventStatus | ''
  page?: number
  page_size?: number
}

function compactParams(params: object): Record<string, unknown> | undefined {
  const entries = Object.entries(params).filter(([, value]) => value !== undefined && value !== '')
  return entries.length > 0 ? Object.fromEntries(entries) : undefined
}

export async function getSupplierCostAlertSettings(): Promise<SupplierCostAlertSettings> {
  const { data } = await apiClient.get<SupplierCostAlertSettings>(
    '/admin/supplier-management/cost-alert/settings'
  )
  return data
}

export async function updateSupplierCostAlertSettings(
  input: SupplierCostAlertSettingsInput
): Promise<SupplierCostAlertSettings> {
  const { data } = await apiClient.put<SupplierCostAlertSettings>(
    '/admin/supplier-management/cost-alert/settings',
    input
  )
  return data
}

export async function listSupplierCostAlertOverrides(): Promise<{ items: SupplierCostAlertOverride[] }> {
  const { data } = await apiClient.get<{ items: SupplierCostAlertOverride[] }>(
    '/admin/supplier-management/cost-alert/overrides'
  )
  return data
}

export async function createSupplierCostAlertOverride(
  input: SupplierCostAlertOverrideInput
): Promise<SupplierCostAlertOverride> {
  const { data } = await apiClient.post<SupplierCostAlertOverride>(
    '/admin/supplier-management/cost-alert/overrides',
    input
  )
  return data
}

export async function updateSupplierCostAlertOverride(
  id: number,
  input: Omit<SupplierCostAlertOverrideInput, 'provider_id'>
): Promise<SupplierCostAlertOverride> {
  const { data } = await apiClient.put<SupplierCostAlertOverride>(
    `/admin/supplier-management/cost-alert/overrides/${id}`,
    input
  )
  return data
}

export async function deleteSupplierCostAlertOverride(id: number): Promise<{ message: string }> {
  const { data } = await apiClient.delete<{ message: string }>(
    `/admin/supplier-management/cost-alert/overrides/${id}`
  )
  return data
}

export async function listSupplierCostAlertEvents(
  params: SupplierCostAlertEventListParams = {}
): Promise<BasePaginationResponse<SupplierCostAlertEvent>> {
  const { data } = await apiClient.get<BasePaginationResponse<SupplierCostAlertEvent>>(
    '/admin/supplier-management/cost-alert/events',
    { params: compactParams(params) }
  )
  return data
}

export const supplierCostAlertAPI = {
  getSupplierCostAlertSettings,
  updateSupplierCostAlertSettings,
  listSupplierCostAlertOverrides,
  createSupplierCostAlertOverride,
  updateSupplierCostAlertOverride,
  deleteSupplierCostAlertOverride,
  listSupplierCostAlertEvents,
}

export default supplierCostAlertAPI

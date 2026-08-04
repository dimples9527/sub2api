import { apiClient } from '../client'
import type { BasePaginationResponse } from '@/types'

export type SupplierBalanceAlertEventType = 'balance_low' | 'balance_recovered'
export type SupplierBalanceAlertEventStatus = 'active' | 'resolved'
export type SupplierBalanceAlertScanStatus = 'never' | 'ok' | 'skipped' | 'error'

export interface SupplierBalanceAlertConfig {
  id: number
  provider_id: number
  provider_code: string
  provider_name: string
  provider_type: string
  provider_enabled: boolean
  enabled: boolean
  threshold: string
  cooldown_seconds: number
  last_scan_at?: string
  last_balance?: string
  last_scan_status: SupplierBalanceAlertScanStatus | string
  last_scan_error?: string
  created_at: string
  updated_at: string
}

export interface SupplierBalanceAlertConfigInput {
  enabled: boolean
  threshold: string
  cooldown_seconds: number
}

export interface SupplierBalanceAlertEvent {
  id: number
  provider_id: number
  provider_code: string
  provider_name: string
  event_type: SupplierBalanceAlertEventType | string
  status: SupplierBalanceAlertEventStatus | string
  balance: string
  threshold: string
  observed_at: string
  resolved_at?: string
  last_seen_at: string
  created_at: string
  updated_at: string
}

export interface SupplierBalanceAlertEventListParams {
  provider_id?: number
  event_type?: SupplierBalanceAlertEventType | ''
  status?: SupplierBalanceAlertEventStatus | ''
  page?: number
  page_size?: number
}

export interface SupplierBalanceAlertScanProviderResult {
  provider_id: number
  provider_name: string
  status: string
  balance?: string
  event_type?: string
  message?: string
}

export interface SupplierBalanceAlertScanResult {
  started_at: string
  finished_at: string
  checked: number
  skipped: number
  triggered: number
  recovered: number
  failed: number
  providers: SupplierBalanceAlertScanProviderResult[]
}

function compactParams(params: object): Record<string, unknown> | undefined {
  const entries = Object.entries(params).filter(([, value]) => value !== undefined && value !== '')
  return entries.length > 0 ? Object.fromEntries(entries) : undefined
}

export async function listSupplierBalanceAlertConfigs(providerId?: number): Promise<{ items: SupplierBalanceAlertConfig[] }> {
  const { data } = await apiClient.get<{ items: SupplierBalanceAlertConfig[] }>(
    '/admin/supplier-management/balance-alert/configs',
    { params: providerId ? { provider_id: providerId } : undefined }
  )
  return data
}

export async function updateSupplierBalanceAlertConfig(
  providerId: number,
  input: SupplierBalanceAlertConfigInput
): Promise<SupplierBalanceAlertConfig> {
  const { data } = await apiClient.put<SupplierBalanceAlertConfig>(
    `/admin/supplier-management/balance-alert/configs/${providerId}`,
    input
  )
  return data
}

export async function scanSupplierBalanceAlerts(): Promise<SupplierBalanceAlertScanResult> {
  const { data } = await apiClient.post<SupplierBalanceAlertScanResult>(
    '/admin/supplier-management/balance-alert/scan'
  )
  return data
}

export async function listSupplierBalanceAlertEvents(
  params: SupplierBalanceAlertEventListParams = {}
): Promise<BasePaginationResponse<SupplierBalanceAlertEvent>> {
  const { data } = await apiClient.get<BasePaginationResponse<SupplierBalanceAlertEvent>>(
    '/admin/supplier-management/balance-alert/events',
    { params: compactParams(params) }
  )
  return data
}

export const supplierBalanceAlertAPI = {
  listSupplierBalanceAlertConfigs,
  updateSupplierBalanceAlertConfig,
  scanSupplierBalanceAlerts,
  listSupplierBalanceAlertEvents,
}

export default supplierBalanceAlertAPI

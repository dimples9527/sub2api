import { apiClient } from '../client'

export type SupplierCostSourceMode = 'auto' | 'upstream' | 'calculated'

export interface SupplierCostSourceSettings {
  cost_source: SupplierCostSourceMode
}

export interface SupplierCostSourceOverride {
  id: number
  provider_id: number
  provider_name: string
  cost_source: SupplierCostSourceMode
  threshold?: number | null
  created_at: string
  updated_at: string
}

export interface SupplierCostSourceOverrideInput {
  provider_id: number
  cost_source: SupplierCostSourceMode
  threshold?: number | null
}

export async function getSupplierCostSourceSettings(): Promise<SupplierCostSourceSettings> {
  const { data } = await apiClient.get<SupplierCostSourceSettings>(
    '/admin/supplier-management/cost-source/settings'
  )
  return data
}

export async function updateSupplierCostSourceSettings(
  input: SupplierCostSourceSettings
): Promise<SupplierCostSourceSettings> {
  const { data } = await apiClient.put<SupplierCostSourceSettings>(
    '/admin/supplier-management/cost-source/settings',
    input
  )
  return data
}

export async function listSupplierCostSourceOverrides(): Promise<{
  items: SupplierCostSourceOverride[]
}> {
  const { data } = await apiClient.get<{ items: SupplierCostSourceOverride[] }>(
    '/admin/supplier-management/cost-source/overrides'
  )
  return data
}

export async function createSupplierCostSourceOverride(
  input: SupplierCostSourceOverrideInput
): Promise<SupplierCostSourceOverride> {
  const { data } = await apiClient.post<SupplierCostSourceOverride>(
    '/admin/supplier-management/cost-source/overrides',
    input
  )
  return data
}

export async function updateSupplierCostSourceOverride(
  id: number,
  input: Omit<SupplierCostSourceOverrideInput, 'provider_id'>
): Promise<SupplierCostSourceOverride> {
  const { data } = await apiClient.put<SupplierCostSourceOverride>(
    `/admin/supplier-management/cost-source/overrides/${id}`,
    input
  )
  return data
}

export async function deleteSupplierCostSourceOverride(id: number): Promise<{ message: string }> {
  const { data } = await apiClient.delete<{ message: string }>(
    `/admin/supplier-management/cost-source/overrides/${id}`
  )
  return data
}

export const supplierCostSourceAPI = {
  getSupplierCostSourceSettings,
  updateSupplierCostSourceSettings,
  listSupplierCostSourceOverrides,
  createSupplierCostSourceOverride,
  updateSupplierCostSourceOverride,
  deleteSupplierCostSourceOverride,
}

export default supplierCostSourceAPI

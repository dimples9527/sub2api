import { apiClient } from '../client'

export interface SupplierProviderType {
  id: number
  code: string
  name: string
  login_url: string
  api_keys_url: string
  groups_url: string
  available_groups_url: string
  balance_url: string
  usage_cost_url: string
  enabled: boolean
  sort_order: number
  created_at: string
  updated_at: string
}

export interface SupplierProviderTypeUpsertPayload {
  code: string
  name: string
  login_url?: string
  api_keys_url?: string
  groups_url?: string
  available_groups_url?: string
  balance_url?: string
  usage_cost_url?: string
  enabled: boolean
  sort_order?: number
}

export async function list(enabledOnly = false): Promise<SupplierProviderType[]> {
  const { data } = await apiClient.get<SupplierProviderType[]>(
    '/admin/supplier-management/provider-types',
    { params: { enabled_only: enabledOnly } }
  )
  return data
}

export async function create(payload: SupplierProviderTypeUpsertPayload): Promise<SupplierProviderType> {
  const { data } = await apiClient.post<SupplierProviderType>(
    '/admin/supplier-management/provider-types',
    payload
  )
  return data
}

export async function update(id: number, payload: SupplierProviderTypeUpsertPayload): Promise<SupplierProviderType> {
  const { data } = await apiClient.put<SupplierProviderType>(
    `/admin/supplier-management/provider-types/${id}`,
    payload
  )
  return data
}

export async function deleteType(id: number): Promise<{ message: string }> {
  const { data } = await apiClient.delete<{ message: string }>(
    `/admin/supplier-management/provider-types/${id}`
  )
  return data
}

export const supplierProviderTypesAPI = {
  list,
  create,
  update,
  delete: deleteType
}

export default supplierProviderTypesAPI

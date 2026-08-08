import { apiClient } from '../client'

export interface CustomPlatform {
  id: number
  code: string
  name: string
  enabled: boolean
  sort_order: number
  created_at: string
  updated_at: string
}

export interface CustomPlatformUpsertPayload {
  code: string
  name: string
  enabled: boolean
  sort_order?: number
}

export async function list(enabledOnly = false): Promise<CustomPlatform[]> {
  const { data } = await apiClient.get<CustomPlatform[]>('/admin/custom-platforms', {
    params: { enabled_only: enabledOnly },
  })
  return data
}

export async function get(id: number): Promise<CustomPlatform> {
  const { data } = await apiClient.get<CustomPlatform>(`/admin/custom-platforms/${id}`)
  return data
}

export async function create(payload: CustomPlatformUpsertPayload): Promise<CustomPlatform> {
  const { data } = await apiClient.post<CustomPlatform>('/admin/custom-platforms', payload)
  return data
}

export async function update(id: number, payload: CustomPlatformUpsertPayload): Promise<CustomPlatform> {
  const { data } = await apiClient.put<CustomPlatform>(`/admin/custom-platforms/${id}`, payload)
  return data
}

export async function remove(id: number): Promise<{ message: string }> {
  const { data } = await apiClient.delete<{ message: string }>(`/admin/custom-platforms/${id}`)
  return data
}

export const customPlatformsAPI = {
  list,
  get,
  create,
  update,
  delete: remove,
}

export default customPlatformsAPI

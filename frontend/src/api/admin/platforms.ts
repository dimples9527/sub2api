import { apiClient } from '../client'

export interface PlatformDefinition {
  code: string
  name: string
  color: string
  health_guard: boolean
}

export async function list(): Promise<PlatformDefinition[]> {
  const { data } = await apiClient.get<PlatformDefinition[]>('/admin/platforms')
  return data
}

export const platformsAPI = { list }

export default platformsAPI

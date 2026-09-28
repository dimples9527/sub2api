import { apiClient } from '../client'

export interface RechargePromoTier {
  threshold: number
  bonus_rate: number
}

export interface RechargePromoConfig {
  enabled: boolean
  start_at?: number | null
  end_at?: number | null
  tiers: RechargePromoTier[]
}

export async function getConfig(): Promise<RechargePromoConfig> {
  const { data } = await apiClient.get<RechargePromoConfig>(
    '/admin/settings/recharge-holiday-promo',
  )
  return data
}

export async function updateConfig(
  payload: RechargePromoConfig,
): Promise<RechargePromoConfig> {
  const { data } = await apiClient.put<RechargePromoConfig>(
    '/admin/settings/recharge-holiday-promo',
    payload,
  )
  return data
}

export const rechargePromoAPI = {
  getConfig,
  updateConfig,
}

export default rechargePromoAPI

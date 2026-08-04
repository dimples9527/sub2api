import { apiClient } from '../client'
import type { BasePaginationResponse } from '@/types'

export type SupplierNotificationChannelType = 'feishu' | 'email'
export type SupplierNotificationEventType = 'balance_low' | 'balance_recovered'
export type SupplierNotificationDeliveryStatus = 'pending' | 'sending' | 'delivered' | 'failed'

export interface SupplierNotificationFeishuConfigInput {
  webhook_url?: string
  secret?: string
}

export interface SupplierNotificationEmailConfigInput {
  host: string
  port: number
  username: string
  password?: string
  from: string
  to: string[]
  starttls: boolean
}

export interface SupplierNotificationProxyConfigInput {
  url: string
  username?: string
  password?: string
}

export interface SupplierNotificationChannelInput {
  name: string
  channel_type: SupplierNotificationChannelType
  enabled: boolean
  feishu?: SupplierNotificationFeishuConfigInput
  email?: SupplierNotificationEmailConfigInput
  proxy?: SupplierNotificationProxyConfigInput
}

export interface SupplierNotificationChannelView {
  id: number
  name: string
  channel_type: SupplierNotificationChannelType | string
  enabled: boolean
  configured: boolean
  feishu_webhook_configured?: boolean
  feishu_secret_configured?: boolean
  email_host?: string
  email_port?: number
  email_username?: string
  email_from?: string
  email_to?: string[]
  email_starttls?: boolean
  proxy_url?: string
  proxy_configured?: boolean
  created_at: string
  updated_at: string
}

export interface SupplierNotificationSubscription {
  id: number
  channel_id: number
  provider_id?: number | null
  event_type: SupplierNotificationEventType | string
  enabled: boolean
  created_at: string
  updated_at: string
}

export interface SupplierNotificationSubscriptionInput {
  channel_id: number
  provider_id?: number | null
  event_type: SupplierNotificationEventType
  enabled: boolean
}

export interface SupplierNotificationDelivery {
  id: number
  channel_id: number
  channel_name: string
  event_id?: number | null
  provider_id: number
  provider_name: string
  event_type: SupplierNotificationEventType | string
  status: SupplierNotificationDeliveryStatus | string
  attempt_count: number
  next_attempt_at: string
  last_error?: string
  sent_at?: string
  created_at: string
  updated_at: string
}

export interface SupplierNotificationDeliveryListParams {
  channel_id?: number
  provider_id?: number
  event_type?: SupplierNotificationEventType | ''
  status?: SupplierNotificationDeliveryStatus | ''
  page?: number
  page_size?: number
}

export interface SupplierNotificationDeliveryAttempt {
  id: number
  delivery_id: number
  attempt_number: number
  status: string
  http_status: number
  error_message?: string
  response_body?: string
  attempted_at: string
  finished_at?: string
}

export interface SupplierNotificationDeliveryDetail extends SupplierNotificationDelivery {
  payload?: Record<string, unknown>
}

export interface SupplierNotificationSendResult {
  http_status: number
  response_body: string
}

function compactParams(params: object): Record<string, unknown> | undefined {
  const entries = Object.entries(params).filter(([, value]) => value !== undefined && value !== '')
  return entries.length > 0 ? Object.fromEntries(entries) : undefined
}

export async function listSupplierNotificationChannels(): Promise<{ items: SupplierNotificationChannelView[] }> {
  const { data } = await apiClient.get<{ items: SupplierNotificationChannelView[] }>(
    '/admin/supplier-management/notification-channels'
  )
  return data
}

export async function createSupplierNotificationChannel(
  input: SupplierNotificationChannelInput
): Promise<SupplierNotificationChannelView> {
  const { data } = await apiClient.post<SupplierNotificationChannelView>(
    '/admin/supplier-management/notification-channels',
    input
  )
  return data
}

export async function updateSupplierNotificationChannel(
  id: number,
  input: SupplierNotificationChannelInput
): Promise<SupplierNotificationChannelView> {
  const { data } = await apiClient.put<SupplierNotificationChannelView>(
    `/admin/supplier-management/notification-channels/${id}`,
    input
  )
  return data
}

export async function deleteSupplierNotificationChannel(id: number): Promise<{ message: string }> {
  const { data } = await apiClient.delete<{ message: string }>(
    `/admin/supplier-management/notification-channels/${id}`
  )
  return data
}

export async function sendSupplierNotificationChannelTest(id: number): Promise<SupplierNotificationSendResult> {
  const { data } = await apiClient.post<SupplierNotificationSendResult>(
    `/admin/supplier-management/notification-channels/${id}/test`
  )
  return data
}

export async function listSupplierNotificationSubscriptions(
  channelId?: number
): Promise<{ items: SupplierNotificationSubscription[] }> {
  const { data } = await apiClient.get<{ items: SupplierNotificationSubscription[] }>(
    '/admin/supplier-management/notification-subscriptions',
    { params: channelId ? { channel_id: channelId } : undefined }
  )
  return data
}

export async function createSupplierNotificationSubscription(
  input: SupplierNotificationSubscriptionInput
): Promise<SupplierNotificationSubscription> {
  const { data } = await apiClient.post<SupplierNotificationSubscription>(
    '/admin/supplier-management/notification-subscriptions',
    input
  )
  return data
}

export async function updateSupplierNotificationSubscription(
  id: number,
  input: SupplierNotificationSubscriptionInput
): Promise<SupplierNotificationSubscription> {
  const { data } = await apiClient.put<SupplierNotificationSubscription>(
    `/admin/supplier-management/notification-subscriptions/${id}`,
    input
  )
  return data
}

export async function deleteSupplierNotificationSubscription(id: number): Promise<{ message: string }> {
  const { data } = await apiClient.delete<{ message: string }>(
    `/admin/supplier-management/notification-subscriptions/${id}`
  )
  return data
}

export async function listSupplierNotificationDeliveries(
  params: SupplierNotificationDeliveryListParams = {}
): Promise<BasePaginationResponse<SupplierNotificationDelivery>> {
  const { data } = await apiClient.get<BasePaginationResponse<SupplierNotificationDelivery>>(
    '/admin/supplier-management/notification-deliveries',
    { params: compactParams(params) }
  )
  return data
}

export async function getSupplierNotificationDelivery(id: number): Promise<SupplierNotificationDeliveryDetail> {
  const { data } = await apiClient.get<SupplierNotificationDeliveryDetail>(
    `/admin/supplier-management/notification-deliveries/${id}`
  )
  return data
}

export async function listSupplierNotificationDeliveryAttempts(
  id: number
): Promise<{ items: SupplierNotificationDeliveryAttempt[] }> {
  const { data } = await apiClient.get<{ items: SupplierNotificationDeliveryAttempt[] }>(
    `/admin/supplier-management/notification-deliveries/${id}/attempts`
  )
  return data
}

export const supplierNotificationsAPI = {
  listSupplierNotificationChannels,
  createSupplierNotificationChannel,
  updateSupplierNotificationChannel,
  deleteSupplierNotificationChannel,
  sendSupplierNotificationChannelTest,
  listSupplierNotificationSubscriptions,
  createSupplierNotificationSubscription,
  updateSupplierNotificationSubscription,
  deleteSupplierNotificationSubscription,
  listSupplierNotificationDeliveries,
  getSupplierNotificationDelivery,
  listSupplierNotificationDeliveryAttempts,
}

export default supplierNotificationsAPI

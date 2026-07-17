/**
 * Admin API barrel export
 * Centralized exports for all admin API modules
 */

import dashboardAPI from './dashboard'
import usersAPI from './users'
import groupsAPI from './groups'
import accountsAPI from './accounts'
import proxiesAPI from './proxies'
import redeemAPI from './redeem'
import promoAPI from './promo'
import announcementsAPI from './announcements'
import settingsAPI from './settings'
import systemAPI from './system'
import subscriptionsAPI from './subscriptions'
import usageAPI from './usage'
import geminiAPI from './gemini'
import antigravityAPI from './antigravity'
import grokAPI from './grok'
import userAttributesAPI from './userAttributes'
import opsAPI from './ops'
import errorPassthroughAPI from './errorPassthrough'
import dataManagementAPI from './dataManagement'
import apiKeysAPI from './apiKeys'
import scheduledTestsAPI from './scheduledTests'
import backupAPI from './backup'
import tlsFingerprintProfileAPI from './tlsFingerprintProfile'
import channelsAPI from './channels'
import channelMonitorAPI from './channelMonitor'
import channelMonitorTemplateAPI from './channelMonitorTemplate'
import adminPaymentAPI from './payment'
import affiliatesAPI from './affiliates'
import riskControlAPI from './riskControl'
import adminComplianceAPI from './compliance'
import auditAPI from './audit'
import upstreamProvidersAPI from './upstreamProviders'
import supplierProvidersAPI from './supplierProviders'
import supplierProviderTypesAPI from './supplierProviderTypes'
import supplierProviderDataAPI from './supplierProviderData'
import supplierAutomationAPI from './supplierAutomation'
import upstreamDashboardAPI from './upstreamDashboard'
import upstreamManagementAPI from './upstreamManagement'
import upstreamAccountSyncAPI from './upstreamAccountSync'
import modelSquareAPI from './modelSquare'

/**
 * Unified admin API object for convenient access
 */
export const adminAPI = {
  dashboard: dashboardAPI,
  users: usersAPI,
  groups: groupsAPI,
  accounts: accountsAPI,
  proxies: proxiesAPI,
  redeem: redeemAPI,
  promo: promoAPI,
  announcements: announcementsAPI,
  settings: settingsAPI,
  system: systemAPI,
  subscriptions: subscriptionsAPI,
  usage: usageAPI,
  gemini: geminiAPI,
  antigravity: antigravityAPI,
  grok: grokAPI,
  userAttributes: userAttributesAPI,
  ops: opsAPI,
  errorPassthrough: errorPassthroughAPI,
  dataManagement: dataManagementAPI,
  apiKeys: apiKeysAPI,
  scheduledTests: scheduledTestsAPI,
  backup: backupAPI,
  tlsFingerprintProfiles: tlsFingerprintProfileAPI,
  channels: channelsAPI,
  channelMonitor: channelMonitorAPI,
  channelMonitorTemplate: channelMonitorTemplateAPI,
  payment: adminPaymentAPI,
  affiliates: affiliatesAPI,
  riskControl: riskControlAPI,
  compliance: adminComplianceAPI,
  audit: auditAPI,
  upstreamProviders: upstreamProvidersAPI,
  supplierProviders: supplierProvidersAPI,
  supplierProviderTypes: supplierProviderTypesAPI,
  supplierProviderData: supplierProviderDataAPI,
  supplierAutomation: supplierAutomationAPI,
  upstreamDashboard: upstreamDashboardAPI,
  upstreamManagement: upstreamManagementAPI,
  upstreamAccountSync: upstreamAccountSyncAPI,
  modelSquare: modelSquareAPI
}

export {
  dashboardAPI,
  usersAPI,
  groupsAPI,
  accountsAPI,
  proxiesAPI,
  redeemAPI,
  promoAPI,
  announcementsAPI,
  settingsAPI,
  systemAPI,
  subscriptionsAPI,
  usageAPI,
  geminiAPI,
  antigravityAPI,
  grokAPI,
  userAttributesAPI,
  opsAPI,
  errorPassthroughAPI,
  dataManagementAPI,
  apiKeysAPI,
  scheduledTestsAPI,
  backupAPI,
  tlsFingerprintProfileAPI,
  channelsAPI,
  channelMonitorAPI,
  channelMonitorTemplateAPI,
  adminPaymentAPI,
  affiliatesAPI,
  riskControlAPI,
  adminComplianceAPI,
  auditAPI,
  upstreamProvidersAPI,
  supplierProvidersAPI,
  supplierProviderTypesAPI,
  supplierProviderDataAPI,
  supplierAutomationAPI,
  upstreamDashboardAPI,
  upstreamManagementAPI,
  upstreamAccountSyncAPI,
  modelSquareAPI
}

export default adminAPI

// Re-export types used by components
export type { AuditLog, AuditLogQuery, AuditLogListResponse } from './audit'
export type { BalanceHistoryItem } from './users'
export type { ErrorPassthroughRule, CreateRuleRequest, UpdateRuleRequest } from './errorPassthrough'
export type { BackupAgentHealth, DataManagementConfig } from './dataManagement'
export type { TLSFingerprintProfile, CreateProfileRequest, UpdateProfileRequest } from './tlsFingerprintProfile'
export type { ContentModerationConfig, ContentModerationLog, ModerationMode } from './riskControl'
export type {
  UpstreamDashboardCost,
  UpstreamDashboardIssue,
  UpstreamDashboardModelRanking,
  UpstreamDashboardProviderRanking,
  UpstreamDashboardRange,
  UpstreamDashboardResponse,
  UpstreamDashboardStability,
  UpstreamDashboardSummary,
  UpstreamDashboardTask,
  UpstreamDashboardTrendPoint,
  UpstreamDashboardWarning
} from './upstreamDashboard'
export type {
  SupplierProvider,
  SupplierProviderListParams,
  SupplierProviderListResult,
  SupplierProviderSummary,
  SupplierProviderUpsertPayload
} from './supplierProviders'
export type {
  SupplierProviderType,
  SupplierProviderTypeUpsertPayload
} from './supplierProviderTypes'
export type {
  SupplierProviderAccount,
  SupplierProviderAccountListResult,
  SupplierProviderDataListParams,
  SupplierProviderGroup,
  SupplierProviderGroupListResult,
  SupplierProviderSyncResult,
  SupplierSyncCounts,
  SupplierSyncScope,
  SupplierSyncStatus
} from './supplierProviderData'
export type {
  SupplierAutomationConfig,
  SupplierAutomationRun,
  SupplierAutomationRunListParams,
  SupplierAutomationRunListResult,
  SupplierAutomationTask
} from './supplierAutomation'
export type {
  UpstreamProviderBalance,
  UpstreamProviderConfig,
  UpstreamProviderKey,
  UpstreamProviderTestResult,
  UpstreamProviderTestStage
} from './upstreamProviders'
export type {
  UpstreamGroupAutoRateFixConfig,
  UpstreamGroupCompareResult,
  UpstreamGroupComparison,
  UpstreamGroupLocalCreateRequest,
  UpstreamGroupRateFixRecord
} from './upstreamManagement'
export type {
  UpstreamAccountHealthGuardConfig,
  UpstreamAccountHealthGuardPollLog,
  UpstreamAccountHealthGuardRunItem,
  UpstreamAccountHealthGuardRunRecord,
  UpstreamAccountHealthGuardRunResponse,
  UpstreamAccountHealthGuardRunSummary,
  UpstreamAccountRateGuardConfig,
  UpstreamAccountRateGuardPollLog,
  UpstreamAccountSyncConflictAccount,
  UpstreamAccountSyncItem,
  UpstreamAccountSyncRecord,
  UpstreamAccountSyncRequest,
  UpstreamAccountSyncResult,
  UpstreamAccountSyncSummary,
  UpstreamAccountSyncUnbindDetail
} from './upstreamAccountSync'
export type {
  AdminModelSquareResult,
  ModelSquareGroup,
  ModelSquareModel,
  ModelSquarePayload
} from './modelSquare'

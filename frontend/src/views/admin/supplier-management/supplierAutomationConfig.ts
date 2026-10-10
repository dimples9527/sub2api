// 供应商自动化任务配置的**纯函数**层：归一化、校验、平台筛选、健康守护账号映射。
//
// 这些函数原先内联在 SupplierAutomationView.vue 里。健康守护账号弹窗与「配置参与择优的分组」弹窗
// 抽成共享组件后（上游账号页也要用），两边都需要同一套规则：
// 归一化必须一致，否则「同一个配置在两处保存出不同结果」；校验必须一致，
// 否则「页面上校验通过的配置，弹窗里存不进去」。
//
// 约定：本文件不引入任何 Vue 响应式 API，也不依赖任何页面级状态。
// 所有需要页面数据的地方（账号映射、分组列表）一律由调用方作为参数传入。

import type { SupplierProviderAccount } from '@/api/admin/supplierProviderData'
import type {
  SupplierAccountHealthGuardMultiplierInterval,
  SupplierAutomationConfig,
} from '@/api/admin/supplierAutomation'
import { resolvePlatformDisplayLabel as platformLabel } from '@/utils/customPlatformLabels'

/** 平台筛选标签：平台列表从当前数据现算，只列出「此刻确实有内容的平台」。 */
export type PlatformFacet = { platform: string; count: number; label: string }

/** 健康守护账号工作区的一行：同一本地账号可能被多个上游账号来源命中，聚合成一条。 */
export interface HealthGuardAccountMapping {
  localAccountID: number
  localAccountName: string
  platform: string
  localGroupPlatforms: string[]
  available: boolean
  sources: SupplierProviderAccount[]
}

export interface HealthGuardPlatformSummary {
  platform: string
  accountCount: number
}

export type HealthGuardAccountThresholdField =
  | 'account_health_guard_account_failure_thresholds'
  | 'account_health_guard_account_slow_thresholds'
  | 'account_health_guard_account_recovery_thresholds'

// 批量设置里「留空即不改」的哨兵值。三处（守卫 / 调度 / 模型）各用一份，值相同但语义独立。
export const HEALTH_GUARD_BATCH_GUARD_KEEP = 'keep'
export const HEALTH_GUARD_BATCH_MODEL_KEEP = 'keep'
export const HEALTH_GUARD_BATCH_MODEL_CLEAR = 'clear'
export const HEALTH_GUARD_BATCH_SCHEDULING_KEEP = 'keep'

export function toNumber(value: string | number, fallback: number): number {
  const next = Number(value)
  return Number.isFinite(next) ? next : fallback
}

export function positiveIntegerOr(value: unknown, fallback: number): number {
  const parsed = Number(value)
  return Number.isInteger(parsed) && parsed > 0 ? parsed : fallback
}

export function normalizePositiveAccountIDs(value: unknown): number[] {
  if (!Array.isArray(value)) return []
  return Array.from(new Set(
    value
      .map(item => Number(item))
      .filter(item => Number.isSafeInteger(item) && item > 0)
  )).sort((a, b) => a - b)
}

export function normalizeStringMap(value: unknown): Record<string, string> {
  if (!value || typeof value !== 'object' || Array.isArray(value)) return {}
  return Object.fromEntries(
    Object.entries(value as Record<string, unknown>)
      .map(([key, item]) => [key.trim(), String(item || '').trim()])
      .filter(([key, item]) => key && item)
  )
}

export function normalizePositiveNumberMap(value: unknown): Record<string, number> {
  if (!value || typeof value !== 'object' || Array.isArray(value)) return {}
  return Object.fromEntries(
    Object.entries(value as Record<string, unknown>)
      .map(([key, item]) => [key.trim(), Math.floor(Number(item))])
      .filter(([key, item]) => Boolean(key) && Number.isFinite(item) && Number(item) > 0)
  ) as Record<string, number>
}

export function normalizeRequiredModelList(models: unknown): string[] {
  if (!Array.isArray(models)) return []
  const seen = new Set<string>()
  const out: string[] = []
  for (const raw of models) {
    const model = String(raw ?? '').trim()
    if (!model) continue
    const key = model.toLowerCase()
    if (seen.has(key)) continue
    seen.add(key)
    out.push(model)
  }
  return out
}

// 传数组而不是 Map，调用方不必先自己聚合；分组弹窗按分组数统计，账号弹窗按可用账号数统计。
export function buildPlatformFacets(items: Array<{ platform: string }>): PlatformFacet[] {
  const counts = new Map<string, number>()
  for (const item of items) {
    counts.set(item.platform, (counts.get(item.platform) ?? 0) + 1)
  }
  return Array.from(counts, ([platform, count]) => ({
    platform,
    count,
    label: platformLabel(platform),
  })).sort((a, b) => a.label.localeCompare(b.label, 'zh-CN'))
}

/** 空选即不过滤：存「已选中」而不是「已排除」。 */
export function matchesPlatformFilter(platform: string, selected: string[]): boolean {
  return selected.length === 0 || selected.includes(platform)
}

/** 平台标签是多选切换：再点一下能移除，而不是被下一个平台顶掉。 */
export function togglePlatformFilter(current: string[], platform: string): string[] {
  return current.includes(platform)
    ? current.filter(item => item !== platform)
    : [...current, platform]
}

export function normalizeHealthGuardPlatform(platform?: string): string {
  return platform?.trim().toLowerCase() || 'unknown'
}

export function effectiveHealthGuardPlatform(account: SupplierProviderAccount): string {
  return normalizeHealthGuardPlatform(
    account.effective_platform || account.local_account_platform || account.platform
  )
}

export function healthGuardLocalGroupPlatforms(account: SupplierProviderAccount): string[] {
  const platforms = Array.from(new Set(
    account.binding_groups.map(group => normalizeHealthGuardPlatform(group.platform))
      .filter(platform => platform !== 'unknown')
  ))
  return platforms.length ? platforms : [effectiveHealthGuardPlatform(account)]
}

export function isHealthGuardAccountAvailable(account: SupplierProviderAccount): boolean {
  return account.active
    && account.local_account_match_status === 'matched'
    && account.local_account_status === 'active'
}

/**
 * 把上游账号列表聚合成「本地账号」维度的映射表。
 * 同一本地账号可能命中多个上游来源：只要有一个可用就算可用，平台以可用来源为准。
 */
export function buildHealthGuardAccountMappings(
  accounts: SupplierProviderAccount[]
): HealthGuardAccountMapping[] {
  const grouped = new Map<number, HealthGuardAccountMapping>()
  for (const account of accounts) {
    const localAccountID = Number(account.local_account_id)
    if (!Number.isSafeInteger(localAccountID) || localAccountID <= 0) continue

    const available = isHealthGuardAccountAvailable(account)
    const current = grouped.get(localAccountID)
    if (current) {
      current.sources.push(account)
      current.localGroupPlatforms = Array.from(new Set([
        ...current.localGroupPlatforms,
        ...healthGuardLocalGroupPlatforms(account),
      ]))
      if (available) {
        current.available = true
        current.platform = effectiveHealthGuardPlatform(account)
        current.localAccountName = account.local_account_name || current.localAccountName
      }
      continue
    }

    grouped.set(localAccountID, {
      localAccountID,
      localAccountName: account.local_account_name || `账号 #${localAccountID}`,
      platform: effectiveHealthGuardPlatform(account),
      localGroupPlatforms: healthGuardLocalGroupPlatforms(account),
      available,
      sources: [account],
    })
  }
  return Array.from(grouped.values()).sort((a, b) => a.localAccountName.localeCompare(b.localAccountName, 'zh-CN'))
}

export function normalizeAccountHealthGuardAccountIntervals(value: unknown): Record<string, number> {
  if (!value || typeof value !== 'object' || Array.isArray(value)) return {}
  return Object.fromEntries(
    Object.entries(value as Record<string, unknown>)
      .map(([key, item]) => [key.trim(), Math.floor(Number(item))])
      .filter(([key, item]) => Boolean(key) && Number.isFinite(item) && Number(item) >= 60)
  ) as Record<string, number>
}

export function normalizeAccountHealthGuardAccountThresholds(value: unknown): Record<string, number> {
  if (!value || typeof value !== 'object' || Array.isArray(value)) return {}
  return Object.fromEntries(
    Object.entries(value as Record<string, unknown>)
      .map(([key, item]) => [key.trim(), Math.floor(Number(item))])
      .filter(([key, item]) => Boolean(key) && Number.isSafeInteger(item) && Number(item) > 0)
  ) as Record<string, number>
}

/** 只保留「明确关掉」（false）的账号：与「未配置 = 跟随全局」区分开。 */
export function normalizeAccountHealthGuardSchedulingChange(value: unknown): Record<string, boolean> {
  if (!value || typeof value !== 'object' || Array.isArray(value)) return {}
  return Object.fromEntries(
    Object.entries(value as Record<string, unknown>)
      .filter(([key, item]) => Boolean(key) && item === false)
  ) as Record<string, boolean>
}

export function normalizeHealthGuardPlatformMultiplierIntervals(
  value: unknown
): Record<string, SupplierAccountHealthGuardMultiplierInterval[]> {
  if (!value || typeof value !== 'object' || Array.isArray(value)) return {}
  const out: Record<string, SupplierAccountHealthGuardMultiplierInterval[]> = {}
  for (const [rawPlatform, rawRules] of Object.entries(value as Record<string, unknown>)) {
    const platform = rawPlatform.trim().toLowerCase()
    if (!platform || !Array.isArray(rawRules)) continue
    const cleaned: SupplierAccountHealthGuardMultiplierInterval[] = []
    for (const rawRule of rawRules) {
      if (!rawRule || typeof rawRule !== 'object') continue
      const rule = rawRule as Record<string, unknown>
      const min = Number(rule.min_multiplier)
      const max = Number(rule.max_multiplier)
      const interval = Math.floor(Number(rule.interval_seconds))
      if (!Number.isFinite(min) || !Number.isFinite(max) || !Number.isFinite(interval)) continue
      if (interval < 60 || min < 0) continue
      if (max > 0 && min >= max) continue
      cleaned.push({ min_multiplier: min, max_multiplier: max > 0 ? max : 0, interval_seconds: interval })
    }
    if (cleaned.length === 0) continue
    cleaned.sort((a, b) => a.min_multiplier - b.min_multiplier)
    out[platform] = cleaned
  }
  return out
}

/** 账号级测试模型的生效值：账号覆盖优先，其次平台默认。 */
export function supplierAccountHealthGuardModelForMapping(
  config: SupplierAutomationConfig,
  mapping: HealthGuardAccountMapping
): string {
  const accountModels = normalizeStringMap(config.account_health_guard_account_models)
  const platformModels = normalizeStringMap(config.account_health_guard_platform_models)
  return accountModels[String(mapping.localAccountID)]?.trim()
    || platformModels[mapping.platform]?.trim()
    || ''
}

/**
 * 取「配置里勾选参与守护」的账号行。
 * mappings 由调用方传入（页面/组件各自持有同一份账号数据），本函数不做数据获取。
 * 配置里存在但映射表里查不到的账号仍要返回一行（不可用占位），否则「勾了但看不见」会变成幽灵配置。
 */
export function healthGuardSelectedRowsForConfig(
  config: SupplierAutomationConfig,
  mappings: HealthGuardAccountMapping[]
): HealthGuardAccountMapping[] {
  const byID = new Map(mappings.map(mapping => [mapping.localAccountID, mapping]))
  return normalizePositiveAccountIDs(config.account_health_guard_account_ids).map(id => byID.get(id) || {
    localAccountID: id,
    localAccountName: `账号 #${id}`,
    platform: 'unknown',
    localGroupPlatforms: ['unknown'],
    available: false,
    sources: [],
  })
}

export function validateAccountHealthGuardSelection(
  config: SupplierAutomationConfig,
  mappings: HealthGuardAccountMapping[]
): string {
  const accountIDs = normalizePositiveAccountIDs(config.account_health_guard_account_ids)
  if (!accountIDs.length) return '请至少选择一个需要检查的账号'

  const missingModelAccounts = healthGuardSelectedRowsForConfig(config, mappings)
    .filter(mapping => mapping.available && !supplierAccountHealthGuardModelForMapping(config, mapping))
    .map(mapping => mapping.localAccountName)
  if (missingModelAccounts.length) {
    return `以下账号尚未配置测试模型：${missingModelAccounts.join('、')}`
  }
  return ''
}

/**
 * 健康守护配置的整体校验：先做**就地归一化**（原地改写 config），再返回第一条错误。
 * 归一化与校验必须同一趟完成 —— 校验通过的 config 直接拿去保存，不能留下未清洗的字段。
 */
export function validateAccountHealthGuardConfig(
  config: SupplierAutomationConfig,
  mappings: HealthGuardAccountMapping[]
): string {
  const rules: Array<[number, number, number, string]> = [
    [config.account_health_guard_max_accounts_per_run, 1, 1000, '单次检查账号数必须在 1 到 1000 之间'],
    [config.account_health_guard_concurrency, 1, 32, '并发数必须在 1 到 32 之间'],
    [config.account_health_guard_timeout_per_account_seconds, 5, 300, '单账号超时必须在 5 到 300 秒之间'],
    [config.account_health_guard_failure_threshold, 1, Number.MAX_SAFE_INTEGER, '连续失败暂停阈值必须是正整数'],
    [config.account_health_guard_slow_threshold, 1, Number.MAX_SAFE_INTEGER, '连续慢响应暂停阈值必须是正整数'],
    [config.account_health_guard_recovery_threshold, 1, Number.MAX_SAFE_INTEGER, '连续健康恢复阈值必须是正整数'],
    [config.account_health_guard_healthy_latency_ms, 1, Number.MAX_SAFE_INTEGER, '默认健康延迟必须是正整数毫秒'],
  ]
  for (const [value, min, max, message] of rules) {
    if (!Number.isInteger(value) || value < min || value > max) return message
  }
  config.account_health_guard_account_ids = normalizePositiveAccountIDs(config.account_health_guard_account_ids)
  config.account_health_guard_account_models = normalizeStringMap(config.account_health_guard_account_models)
  config.account_health_guard_platform_models = normalizeStringMap(config.account_health_guard_platform_models)
  const rawIntervals = config.account_health_guard_account_intervals || {}
  for (const [accountID, interval] of Object.entries(rawIntervals)) {
    const parsed = Math.floor(Number(interval))
    if (Number.isFinite(parsed) && parsed > 0 && parsed < 60) {
      return `账号 #${accountID} 的检查间隔不能小于 60 秒`
    }
  }
  config.account_health_guard_account_intervals = normalizeAccountHealthGuardAccountIntervals(config.account_health_guard_account_intervals)
  const accountThresholdRules: Array<[Record<string, number>, string]> = [
    [config.account_health_guard_account_failure_thresholds || {}, '连续失败暂停阈值'],
    [config.account_health_guard_account_slow_thresholds || {}, '连续慢响应暂停阈值'],
    [config.account_health_guard_account_recovery_thresholds || {}, '连续健康恢复阈值'],
  ]
  for (const [thresholds, label] of accountThresholdRules) {
    for (const [accountID, threshold] of Object.entries(thresholds)) {
      if (!Number.isInteger(threshold) || threshold < 1) {
        return `账号 #${accountID} 的${label}必须是正整数`
      }
    }
  }
  config.account_health_guard_account_failure_thresholds = normalizeAccountHealthGuardAccountThresholds(config.account_health_guard_account_failure_thresholds)
  config.account_health_guard_account_slow_thresholds = normalizeAccountHealthGuardAccountThresholds(config.account_health_guard_account_slow_thresholds)
  config.account_health_guard_account_recovery_thresholds = normalizeAccountHealthGuardAccountThresholds(config.account_health_guard_account_recovery_thresholds)
  return validateAccountHealthGuardSelection(config, mappings)
}

// 平台筛选可能把结果筛空。这时沿用「当前没有可配置的分组」会误导 ——
// 用户会以为分组被删光了，实际上只是筛选条件太窄。
// 提示里只列出**真正生效**的筛选条件，别让用户去关一个本来就开着的开关。
// 没按平台筛时保持原有两种文案不变（「当前没有已关闭的…」比通用文案更具体，不要丢）。
export function groupFilterEmptyHint(
  selectedPlatformCount: number,
  disabledOnly: boolean,
  disabledEmptyText: string
): string {
  if (selectedPlatformCount === 0) {
    return disabledOnly ? disabledEmptyText : '当前没有可配置的分组。'
  }
  const actions = ['减少选中的平台']
  if (disabledOnly) {
    actions.push('关闭「仅看已关闭」')
  }
  return `当前筛选条件下没有分组，试试${actions.join('或')}。`
}

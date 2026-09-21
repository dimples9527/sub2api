import type {
  SupplierAccountGroupBindResult,
  SupplierBindableLocalAccount,
} from '@/api/admin/supplierProviderData'
import type { GroupPlatform } from '@/types'

/**
 * 归一化「要绑定分组的本地账号 ID」：丢掉空值、非正整数与重复项。
 *
 * 上游账号表里「未匹配」「匹配冲突」的行没有本地账号，会被勾选但绑定不落地；
 * 调用方先把 manageableLocalAccountID 的结果映射过来，这里负责去重与丢弃空值。
 */
export function uniqueBatchBindAccountIDs(values: Array<number | null | undefined>): number[] {
  const ids: number[] = []
  for (const value of values) {
    const id = Number(value)
    if (!Number.isInteger(id) || id <= 0 || ids.includes(id)) continue
    ids.push(id)
  }
  return ids
}

/**
 * 勾选的账号同属一个平台时返回该平台，用于过滤候选分组；跨平台混选时返回 undefined。
 *
 * 跨平台时不做预过滤：混合渠道风险由后端按账号逐条判定，逐条给出失败原因，
 * 在前端提前合并成「一刀切」的过滤反而会让用户找不到本该可选的分组。
 */
export function commonBatchBindPlatform(
  platforms: Array<string | undefined | null>
): GroupPlatform | undefined {
  const distinct = new Set<string>()
  for (const platform of platforms) {
    const normalized = String(platform ?? '').trim().toLowerCase()
    if (normalized === '' || normalized === 'unknown') continue
    distinct.add(normalized)
  }
  if (distinct.size !== 1) return undefined
  return [...distinct][0] as GroupPlatform
}

/**
 * 批量绑定结果的提示文案。
 *
 * failed 为 true 时调用方应改用 showError；文案里必须带上「无需变更」的数量，
 * 否则用户看到「已为 0 个账号绑定」会以为操作失败。
 */
export function batchBindResultSummary(
  result: SupplierAccountGroupBindResult | null
): { text: string; failed: boolean } {
  if (!result) {
    return { text: '批量绑定分组没有返回结果，请刷新后重试', failed: true }
  }
  const bound = Number(result.bound) || 0
  const unchanged = Number(result.unchanged) || 0
  const failed = Number(result.failed) || 0
  const parts: string[] = []

  if (bound > 0) {
    parts.push(`已为 ${bound} 个账号追加绑定分组`)
  }
  if (unchanged > 0) {
    parts.push(`${unchanged} 个账号本就在所选分组内`)
  }
  if (failed > 0) {
    parts.push(`${failed} 个账号失败`)
  }
  if (parts.length === 0) {
    return { text: '没有需要变更的账号', failed: false }
  }
  return { text: parts.join('，'), failed: failed > 0 }
}

/** 候选账号行里「已加入的分组」文案。 */
export function bindableAccountGroupsLabel(
  account: Pick<SupplierBindableLocalAccount, 'groups'>
): string {
  const groups = account.groups ?? []
  if (groups.length === 0) return '未加入分组'
  return groups.map(group => group.name).join('、')
}

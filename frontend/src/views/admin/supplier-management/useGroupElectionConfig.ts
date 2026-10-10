import { computed, ref } from 'vue'
import type { AdminGroup } from '@/types'
import type { SupplierProviderAccount } from '@/api/admin/supplierProviderData'
import type { SelectOption } from '@/components/common/Select.vue'
import type { SupplierAutomationConfig } from '@/api/admin/supplierAutomation'
import { useAppStore } from '@/stores/app'
import {
  buildHealthGuardAccountMappings,
  buildPlatformFacets,
  groupFilterEmptyHint,
  matchesPlatformFilter,
  normalizePositiveAccountIDs,
  normalizeRequiredModelList,
  type HealthGuardAccountMapping,
} from './supplierAutomationConfig'

/**
 * 「分组择优调度」配置的派生值与分组级操作。
 *
 * 这些成员原先只长在自动化页里，但「配置参与择优的分组」弹窗搬到共享组件、
 * 上游账号页也要复用它之后，页面与弹窗必须共用同一份口径 ——
 * 否则「已关闭 N 个分组」的摘要、保存时的归一化、弹窗里的开关会各算一套，
 * 出现「摘要说 3 个、弹窗显示 4 个」这种没人能一眼看出是谁错的分叉。
 *
 * 不持有任何数据：config / 分组 / 账号都由调用方给（页面给编辑弹窗的草稿，
 * 弹窗给 v-model 进来的那份），所以同一个 config 在哪读都一样。
 */
export function useGroupElectionConfig(
  getConfig: () => SupplierAutomationConfig,
  getGroups: () => AdminGroup[],
  getAccounts: () => SupplierProviderAccount[]
) {
  const appStore = useAppStore()

  // 用 computed 而不是把 getConfig() 存成常量：调用方会整体替换 config 对象
  // （自动化页 openEdit 里 Object.assign(editForm, ...) 就是），存常量会读到旧对象。
  const configRef = computed(() => getConfig())

  /** 分组 → 可用成员账号的候选映射，与健康守护账号弹窗共用同一份账号列表。 */
  const accountMappings = computed<HealthGuardAccountMapping[]>(() =>
    buildHealthGuardAccountMappings(getAccounts())
  )

  /** 平台标签从当前分组现算，只列出此刻确实有内容的平台。 */
  const groupPlatformFacets = computed(() => buildPlatformFacets(getGroups()))

  const electionGroupSearch = ref('')

  const electionGroupDisabledOnly = ref(false)

  const electionGroupPlatformFilter = ref<string[]>([])

  const electionGroupCheckedIDs = ref<number[]>([])

  const groupElectionDisabledGroupIDs = computed(() =>
    normalizePositiveAccountIDs(configRef.value.group_scheduling_election_disabled_group_ids)
  )

  const groupElectionKeepHealthyGroupIDs = computed(() =>
    normalizePositiveAccountIDs(configRef.value.group_scheduling_election_keep_healthy_incumbent_group_ids)
  )

  const groupElectionKeepHealthyExcludedGroupIDs = computed(() =>
    normalizePositiveAccountIDs(configRef.value.group_scheduling_election_keep_healthy_incumbent_excluded_group_ids)
  )

  const groupElectionDryRunGroupIDs = computed(() =>
    normalizePositiveAccountIDs(configRef.value.group_scheduling_election_dry_run_group_ids)
  )

  const electionDryRunAll = computed(() => configRef.value.group_scheduling_election_dry_run === true)

  const electionKeepHealthyAll = computed(
    () => configRef.value.group_scheduling_election_keep_healthy_incumbent_global === true
  )

  const groupElectionPriorityEnabledGroupIDs = computed(() =>
    normalizePositiveAccountIDs(configRef.value.group_scheduling_election_priority_enabled_group_ids)
  )

  const groupElectionPriorityDisabledGroupIDs = computed(() =>
    normalizePositiveAccountIDs(configRef.value.group_scheduling_election_priority_disabled_group_ids)
  )

  const electionPriorityAll = computed(
    () => configRef.value.group_scheduling_election_priority_enabled_global === true
  )

  const electionAlertEnabled = computed(
    () => configRef.value.group_scheduling_election_alert_enabled === true
  )

  const electionAlertOverrides = computed<Record<number, boolean>>(() => {
    const raw = configRef.value.group_scheduling_election_alert_group_overrides
    if (!raw || typeof raw !== 'object') return {}
    const out: Record<number, boolean> = {}
    for (const [key, value] of Object.entries(raw)) {
      const groupID = Number(key)
      if (!Number.isSafeInteger(groupID) || groupID <= 0) continue
      out[groupID] = value === true
    }
    return out
  })

  const electionAlertOverrideCount = computed(() => Object.keys(electionAlertOverrides.value).length)

  const electionAlertMutedCount = computed(
    () => Object.values(electionAlertOverrides.value).filter(enabled => !enabled).length
  )

  const electionAlertForcedCount = computed(
    () => Object.values(electionAlertOverrides.value).filter(enabled => enabled).length
  )

  const groupElectionDefaultAccountMap = computed<Record<string, number>>(() => {
    const raw = configRef.value.group_scheduling_election_default_account_by_group
    const out: Record<string, number> = {}
    if (raw && typeof raw === 'object') {
      for (const [key, value] of Object.entries(raw)) {
        const groupID = Number(key)
        const accountID = Number(value)
        if (!Number.isSafeInteger(groupID) || groupID <= 0) continue
        if (!Number.isSafeInteger(accountID) || accountID <= 0) continue
        out[String(groupID)] = accountID
      }
    }
    return out
  })

  const groupElectionDefaultAccountCount = computed(
    () => Object.keys(groupElectionDefaultAccountMap.value).length
  )

  const groupElectionRequiredModelsMap = computed<Record<string, string[]>>(() => {
    const raw = configRef.value.group_scheduling_election_required_models
    const out: Record<string, string[]> = {}
    if (raw && typeof raw === 'object') {
      for (const [key, models] of Object.entries(raw)) {
        const groupID = Number(key)
        if (!Number.isFinite(groupID) || groupID <= 0) continue
        const cleaned = normalizeRequiredModelList(models)
        if (cleaned.length) out[String(groupID)] = cleaned
      }
    }
    return out
  })

  const groupElectionRequiredModelsCount = computed(() => Object.keys(groupElectionRequiredModelsMap.value).length)

  const groupElectionTopNByGroupMap = computed<Record<string, number>>(() => {
    const raw = configRef.value.group_scheduling_election_top_n_by_group
    const out: Record<string, number> = {}
    if (raw && typeof raw === 'object') {
      for (const [key, value] of Object.entries(raw)) {
        const groupID = Number(key)
        if (!Number.isFinite(groupID) || groupID <= 0) continue
        const topN = Number(value)
        if (!Number.isInteger(topN) || topN <= 0) continue
        out[String(groupID)] = topN
      }
    }
    return out
  })

  const groupElectionTopNOverridesCount = computed(() => Object.keys(groupElectionTopNByGroupMap.value).length)

  const electionGroupScopeSummary = computed(() => {
    const disabled = groupElectionDisabledGroupIDs.value.length
    return {
      disabled,
      enabled: Math.max(getGroups().length - disabled, 0),
    }
  })

  const electionFilteredGroups = computed(() => {
    const keyword = electionGroupSearch.value.trim().toLowerCase()
    let result = getGroups()
    if (electionGroupPlatformFilter.value.length > 0) {
      result = result.filter(group =>
        matchesPlatformFilter(group.platform, electionGroupPlatformFilter.value)
      )
    }
    if (electionGroupDisabledOnly.value) {
      result = result.filter(group => electionGroupIsDisabled(group.id))
    }
    if (keyword) {
      result = result.filter(group =>
        group.name.toLowerCase().includes(keyword) || String(group.id).includes(keyword)
      )
    }
    // 筛选之后再排倍率序：排序基准固定，列表顺序不随筛选/开关翻转跳变。
    return [...result].sort((a, b) => groupRateMultiplierSortKey(a) - groupRateMultiplierSortKey(b))
  })

  const electionGroupAllChecked = computed(
    () =>
      electionFilteredGroups.value.length > 0 &&
      electionFilteredGroups.value.every(group => electionGroupCheckedIDs.value.includes(group.id))
  )

  const electionGroupEmptyHint = computed(() =>
    groupFilterEmptyHint(
      electionGroupPlatformFilter.value.length,
      electionGroupDisabledOnly.value,
      '当前没有已关闭择优的分组。'
    )
  )

  const electionGroupMemberAccountsMap = computed<Map<number, Array<{ id: number; name: string }>>>(() => {
    const map = new Map<number, Array<{ id: number; name: string }>>()
    for (const mapping of accountMappings.value) {
      if (!mapping.available) continue
      const seen = new Set<number>()
      for (const source of mapping.sources) {
        for (const group of source.binding_groups || []) {
          const groupID = Number(group.id)
          if (!Number.isSafeInteger(groupID) || groupID <= 0 || seen.has(groupID)) continue
          seen.add(groupID)
          const list = map.get(groupID) ?? []
          list.push({ id: mapping.localAccountID, name: mapping.localAccountName })
          map.set(groupID, list)
        }
      }
    }
    for (const list of map.values()) list.sort((a, b) => a.name.localeCompare(b.name, 'zh-CN'))
    return map
  })

  function groupRateMultiplierSortKey(group: AdminGroup): number {
    const rate = Number(group.rate_multiplier)
    return Number.isFinite(rate) ? rate : Number.POSITIVE_INFINITY
  }

  /** 打开弹窗时把上一次操作的残留清干净：筛选条件与勾选都不能跨次带入。 */
  function resetElectionDialogFilters() {
    electionGroupSearch.value = ''
    electionGroupDisabledOnly.value = false
    electionGroupPlatformFilter.value = []
    // 勾选是上一次批量操作的选择，带着旧勾选打开会让用户看到「已选 3 个」却不知道是哪 3 个
    // （尤其是被筛选挡住的行）。
    electionGroupCheckedIDs.value = []
  }

  function electionGroupIsDisabled(groupID: number): boolean {
    return groupElectionDisabledGroupIDs.value.includes(groupID)
  }

  function electionGroupParticipates(groupID: number): boolean {
    return !electionGroupIsDisabled(groupID)
  }

  function toggleElectionGroup(groupID: number) {
    // 勾选 = 参与择优；勾掉的才进配置，因此写回的是"关闭列表"。
    const disabled = groupElectionDisabledGroupIDs.value
    const next = disabled.includes(groupID)
      ? disabled.filter(id => id !== groupID)
      : [...disabled, groupID]
    configRef.value.group_scheduling_election_disabled_group_ids = normalizePositiveAccountIDs(next)
    // 关闭择优的分组不该再留在"健康锁定"列表里（锁定对不择优的分组无意义）；
    // 演练名单同理——不参与择优的分组根本不会产生切换建议，留着只会让摘要多算一个。
    if (!disabled.includes(groupID)) {
      const keep = groupElectionKeepHealthyGroupIDs.value
      if (keep.includes(groupID)) {
        configRef.value.group_scheduling_election_keep_healthy_incumbent_group_ids =
          normalizePositiveAccountIDs(keep.filter(id => id !== groupID))
      }
      const keepExcluded = groupElectionKeepHealthyExcludedGroupIDs.value
      if (keepExcluded.includes(groupID)) {
        configRef.value.group_scheduling_election_keep_healthy_incumbent_excluded_group_ids =
          normalizePositiveAccountIDs(keepExcluded.filter(id => id !== groupID))
      }
      const dryRun = groupElectionDryRunGroupIDs.value
      if (dryRun.includes(groupID)) {
        configRef.value.group_scheduling_election_dry_run_group_ids =
          normalizePositiveAccountIDs(dryRun.filter(id => id !== groupID))
      }
      // 优先级计分名单同理：不参与择优的分组不该留在"计优先级"列表里。
      if (groupElectionPriorityEnabledGroupIDs.value.includes(groupID)) {
        configRef.value.group_scheduling_election_priority_enabled_group_ids = normalizePositiveAccountIDs(
          groupElectionPriorityEnabledGroupIDs.value.filter(id => id !== groupID)
        )
      }
      if (groupElectionPriorityDisabledGroupIDs.value.includes(groupID)) {
        configRef.value.group_scheduling_election_priority_disabled_group_ids = normalizePositiveAccountIDs(
          groupElectionPriorityDisabledGroupIDs.value.filter(id => id !== groupID)
        )
      }
      // 每组开启数的分组级覆盖同理：不参与择优的分组留着覆盖没意义，清掉、回落全局默认。
      if (groupElectionTopNByGroupMap.value[String(groupID)]) {
        const next: Record<string, number> = { ...groupElectionTopNByGroupMap.value }
        delete next[String(groupID)]
        configRef.value.group_scheduling_election_top_n_by_group = next as unknown as Record<number, number>
      }
    }
  }

  function electionGroupKeepHealthy(groupID: number): boolean {
    // 生效值口径与后端 keepHealthyForGroup 一致：强制不锁定→false（优先）、强制锁定→true、都不在→跟随全局。
    if (groupElectionKeepHealthyExcludedGroupIDs.value.includes(groupID)) return false
    if (groupElectionKeepHealthyGroupIDs.value.includes(groupID)) return true
    return electionKeepHealthyAll.value
  }

  function setElectionDryRunAll(value: unknown) {
    configRef.value.group_scheduling_election_dry_run = Boolean(value)
  }

  function setElectionKeepHealthyAll(value: unknown) {
    configRef.value.group_scheduling_election_keep_healthy_incumbent_global = Boolean(value)
  }

  function setElectionAlertEnabled(value: unknown) {
    configRef.value.group_scheduling_election_alert_enabled = Boolean(value)
  }

  function electionAlertOverrideValue(groupID: number): boolean | undefined {
    return electionAlertOverrides.value[groupID]
  }

  function setElectionAlertOverride(groupID: number, value: boolean | undefined) {
    if (!Number.isSafeInteger(groupID) || groupID <= 0) return
    const next = { ...electionAlertOverrides.value }
    if (value === undefined) delete next[groupID]
    else next[groupID] = value
    configRef.value.group_scheduling_election_alert_group_overrides = next
  }

  function electionGroupDefaultAccount(groupID: number): number | undefined {
    const value = groupElectionDefaultAccountMap.value[String(groupID)]
    return Number.isSafeInteger(value) && value > 0 ? value : undefined
  }

  function electionGroupDefaultAccountValue(groupID: number): string {
    const current = electionGroupDefaultAccount(groupID)
    return current === undefined ? '' : String(current)
  }

  function electionGroupDefaultAccountOptions(groupID: number): SelectOption[] {
    const options: SelectOption[] = [{ value: '', label: '不指定（按择优结果）' }]
    for (const account of electionGroupMemberAccounts(groupID)) {
      options.push({ value: String(account.id), label: account.name })
    }
    const current = electionGroupDefaultAccount(groupID)
    if (current !== undefined && !options.some(option => option.value === String(current))) {
      options.push({ value: String(current), label: `#${current}（不在候选里）` })
    }
    return options
  }

  function setElectionGroupDefaultAccount(groupID: number, raw: string | number | boolean | null) {
    if (!Number.isSafeInteger(groupID) || groupID <= 0) return
    const accountID = Number(raw)
    const next = { ...groupElectionDefaultAccountMap.value }
    if (!Number.isSafeInteger(accountID) || accountID <= 0) delete next[String(groupID)]
    else next[String(groupID)] = accountID
    configRef.value.group_scheduling_election_default_account_by_group =
      next as unknown as Record<number, number>
  }

  function electionGroupMemberAccounts(groupID: number): Array<{ id: number; name: string }> {
    return electionGroupMemberAccountsMap.value.get(groupID) ?? []
  }

  function electionGroupDryRun(groupID: number): boolean {
    // 总开关打开时全部分组都演练，行内开关显示成打开（此时改它不会让该分组真的生效）。
    return electionDryRunAll.value || groupElectionDryRunGroupIDs.value.includes(groupID)
  }

  function toggleElectionGroupDryRun(groupID: number) {
    const list = groupElectionDryRunGroupIDs.value
    const next = list.includes(groupID)
      ? list.filter(id => id !== groupID)
      : [...list, groupID]
    configRef.value.group_scheduling_election_dry_run_group_ids = normalizePositiveAccountIDs(next)
  }

  function toggleElectionKeepHealthyGroup(groupID: number) {
    // 覆盖语义：目标 = 翻转当前生效值。与全局不一致才写显式覆盖，一致则清掉覆盖、回落继承全局。
    const desired = !electionGroupKeepHealthy(groupID)
    let included = groupElectionKeepHealthyGroupIDs.value.filter(id => id !== groupID)
    let excluded = groupElectionKeepHealthyExcludedGroupIDs.value.filter(id => id !== groupID)
    if (desired !== electionKeepHealthyAll.value) {
      if (desired) included = [...included, groupID]
      else excluded = [...excluded, groupID]
    }
    configRef.value.group_scheduling_election_keep_healthy_incumbent_group_ids = normalizePositiveAccountIDs(included)
    configRef.value.group_scheduling_election_keep_healthy_incumbent_excluded_group_ids =
      normalizePositiveAccountIDs(excluded)
  }

  function electionGroupPriority(groupID: number): boolean {
    if (groupElectionPriorityDisabledGroupIDs.value.includes(groupID)) return false
    if (groupElectionPriorityEnabledGroupIDs.value.includes(groupID)) return true
    return electionPriorityAll.value
  }

  function setElectionPriorityAll(value: unknown) {
    configRef.value.group_scheduling_election_priority_enabled_global = Boolean(value)
  }

  function toggleElectionGroupPriority(groupID: number) {
    // 覆盖语义与健康锁一致：目标 = 翻转当前生效值。与全局不一致才写显式覆盖，一致则清掉覆盖、回落继承全局。
    const desired = !electionGroupPriority(groupID)
    let included = groupElectionPriorityEnabledGroupIDs.value.filter(id => id !== groupID)
    let excluded = groupElectionPriorityDisabledGroupIDs.value.filter(id => id !== groupID)
    if (desired !== electionPriorityAll.value) {
      if (desired) included = [...included, groupID]
      else excluded = [...excluded, groupID]
    }
    configRef.value.group_scheduling_election_priority_enabled_group_ids = normalizePositiveAccountIDs(included)
    configRef.value.group_scheduling_election_priority_disabled_group_ids = normalizePositiveAccountIDs(excluded)
  }

  function electionGroupRequiredModelsText(groupID: number): string {
    return (groupElectionRequiredModelsMap.value[String(groupID)] || []).join(', ')
  }

  function setElectionGroupRequiredModels(groupID: number, text: string) {
    const models = normalizeRequiredModelList(String(text ?? '').split(/[,，\s]+/))
    const next: Record<string, string[]> = { ...groupElectionRequiredModelsMap.value }
    if (models.length) next[String(groupID)] = models
    else delete next[String(groupID)]
    configRef.value.group_scheduling_election_required_models = next as unknown as Record<number, string[]>
  }

  function electionGroupTopNText(groupID: number): string {
    const value = groupElectionTopNByGroupMap.value[String(groupID)]
    return value ? String(value) : ''
  }

  function setElectionGroupTopN(groupID: number, text: string) {
    const next: Record<string, number> = { ...groupElectionTopNByGroupMap.value }
    const topN = Math.floor(Number(String(text ?? '').trim()))
    if (Number.isFinite(topN) && topN > 0) next[String(groupID)] = Math.min(topN, 100)
    else delete next[String(groupID)]
    configRef.value.group_scheduling_election_top_n_by_group = next as unknown as Record<number, number>
  }

  function enableAllElectionGroups() {
    const current = groupElectionDisabledGroupIDs.value
    if (current.length > 1) {
      const confirmed = window.confirm(`将 ${current.length} 个分组全部恢复为参与择优？此操作在保存任务后生效。`)
      if (!confirmed) {
        return
      }
    }
    configRef.value.group_scheduling_election_disabled_group_ids = []
    if (current.length > 0) {
      appStore.showSuccess(`已将 ${current.length} 个分组恢复为参与择优，保存任务后生效`)
    }
  }

  function electionGroupIsChecked(groupID: number): boolean {
    return electionGroupCheckedIDs.value.includes(groupID)
  }

  function toggleElectionGroupCheck(groupID: number) {
    electionGroupCheckedIDs.value = electionGroupIsChecked(groupID)
      ? electionGroupCheckedIDs.value.filter(id => id !== groupID)
      : [...electionGroupCheckedIDs.value, groupID]
  }

  function toggleElectionGroupCheckAll() {
    const ids = electionFilteredGroups.value.map(group => group.id)
    if (ids.length === 0) return
    if (ids.every(id => electionGroupCheckedIDs.value.includes(id))) {
      electionGroupCheckedIDs.value = electionGroupCheckedIDs.value.filter(id => !ids.includes(id))
      return
    }
    electionGroupCheckedIDs.value = [...new Set([...electionGroupCheckedIDs.value, ...ids])]
  }

  function clearElectionGroupChecks() {
    electionGroupCheckedIDs.value = []
  }

  function applyElectionGroupBatch(
    label: string,
    desired: boolean,
    readCurrent: (groupID: number) => boolean,
    apply: (groupID: number) => void,
    requireParticipating = true
  ) {
    const targets = [...electionGroupCheckedIDs.value]
    let changed = 0
    let skipped = 0
    for (const groupID of targets) {
      if (requireParticipating && !electionGroupParticipates(groupID)) {
        skipped += 1
        continue
      }
      if (readCurrent(groupID) === desired) continue
      apply(groupID)
      changed += 1
    }
    const skipNote = skipped > 0 ? `，跳过 ${skipped} 个未参与择优的分组` : ''
    if (changed > 0) {
      appStore.showSuccess(
        `已将 ${changed} 个分组的「${label}」设为${desired ? '开启' : '关闭'}${skipNote}，保存任务后生效`
      )
      return
    }
    if (skipped > 0) {
      appStore.showError(`所选分组都不参与择优，无法设置「${label}」`)
      return
    }
    appStore.showSuccess(`所选分组的「${label}」已是${desired ? '开启' : '关闭'}，无需修改`)
  }

  function batchElectionGroupsSelect(participates: boolean) {
    applyElectionGroupBatch(
      '参与择优',
      participates,
      groupID => electionGroupParticipates(groupID),
      groupID => toggleElectionGroup(groupID),
      false
    )
  }

  function batchElectionGroupsKeepHealthy(enabled: boolean) {
    applyElectionGroupBatch(
      '健康锁定',
      enabled,
      groupID => electionGroupKeepHealthy(groupID),
      groupID => toggleElectionKeepHealthyGroup(groupID)
    )
  }

  function batchElectionGroupsDryRun(enabled: boolean) {
    // 读「名单里的原始状态」而不是行上显示的生效值：总开关打开时全部分组都显示成演练中，
    // 用生效值判断会让「批量关闭」对一个不在名单里的分组调 toggle，反倒把它加进名单。
    applyElectionGroupBatch(
      '演练',
      enabled,
      groupID => groupElectionDryRunGroupIDs.value.includes(groupID),
      groupID => toggleElectionGroupDryRun(groupID)
    )
  }

  function batchElectionGroupsPriority(enabled: boolean) {
    applyElectionGroupBatch(
      '计优先级',
      enabled,
      groupID => electionGroupPriority(groupID),
      groupID => toggleElectionGroupPriority(groupID)
    )
  }

  return {
    groupPlatformFacets,
    electionGroupSearch,
    electionGroupDisabledOnly,
    electionGroupPlatformFilter,
    electionGroupCheckedIDs,
    groupElectionDisabledGroupIDs,
    groupElectionKeepHealthyGroupIDs,
    groupElectionKeepHealthyExcludedGroupIDs,
    groupElectionDryRunGroupIDs,
    electionDryRunAll,
    electionKeepHealthyAll,
    groupElectionPriorityEnabledGroupIDs,
    groupElectionPriorityDisabledGroupIDs,
    electionPriorityAll,
    electionAlertEnabled,
    electionAlertOverrides,
    electionAlertOverrideCount,
    electionAlertMutedCount,
    electionAlertForcedCount,
    groupElectionDefaultAccountMap,
    groupElectionDefaultAccountCount,
    groupElectionRequiredModelsMap,
    groupElectionRequiredModelsCount,
    groupElectionTopNByGroupMap,
    groupElectionTopNOverridesCount,
    electionGroupScopeSummary,
    electionFilteredGroups,
    electionGroupAllChecked,
    electionGroupEmptyHint,
    electionGroupMemberAccountsMap,
    groupRateMultiplierSortKey,
    resetElectionDialogFilters,
    electionGroupIsDisabled,
    electionGroupParticipates,
    toggleElectionGroup,
    electionGroupKeepHealthy,
    setElectionDryRunAll,
    setElectionKeepHealthyAll,
    setElectionAlertEnabled,
    electionAlertOverrideValue,
    setElectionAlertOverride,
    electionGroupDefaultAccount,
    electionGroupDefaultAccountValue,
    electionGroupDefaultAccountOptions,
    setElectionGroupDefaultAccount,
    electionGroupMemberAccounts,
    electionGroupDryRun,
    toggleElectionGroupDryRun,
    toggleElectionKeepHealthyGroup,
    electionGroupPriority,
    setElectionPriorityAll,
    toggleElectionGroupPriority,
    electionGroupRequiredModelsText,
    setElectionGroupRequiredModels,
    electionGroupTopNText,
    setElectionGroupTopN,
    enableAllElectionGroups,
    electionGroupIsChecked,
    toggleElectionGroupCheck,
    toggleElectionGroupCheckAll,
    clearElectionGroupChecks,
    applyElectionGroupBatch,
    batchElectionGroupsSelect,
    batchElectionGroupsKeepHealthy,
    batchElectionGroupsDryRun,
    batchElectionGroupsPriority,
  }
}

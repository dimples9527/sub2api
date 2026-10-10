<template>
  <BaseDialog
    :show="show"
    title="配置参与择优的分组"
    width="full"
    :z-index="60"
    @close="close"
  >
    <div class="sp-rate-guard-group-dialog">
      <section class="sp-rate-guard-group-workspace">
        <div class="sp-rate-guard-group-summary" aria-label="分组择优调度分组配置摘要">
          <article>
            <span>参与择优</span>
            <strong>{{ electionGroupScopeSummary.enabled }}</strong>
          </article>
          <article :class="{ warning: electionGroupScopeSummary.disabled > 0 }">
            <span>已关闭</span>
            <strong>{{ electionGroupScopeSummary.disabled }}</strong>
          </article>
          <article>
            <span>可选分组</span>
            <strong>{{ groups.length }}</strong>
          </article>
        </div>

        <div class="sp-rate-guard-group-toolbar">
          <div class="sp-rate-guard-group-filters">
            <Input v-model="electionGroupSearch" placeholder="搜索分组名称或 ID" />
            <button
              class="sp-rate-guard-group-selected-toggle"
              :class="{ active: electionGroupDisabledOnly }"
              type="button"
              :aria-pressed="electionGroupDisabledOnly"
              @click="electionGroupDisabledOnly = !electionGroupDisabledOnly"
            >
              <span class="sp-rate-guard-group-selected-toggle-mark" aria-hidden="true"></span>
              仅看已关闭
              <strong>{{ electionGroupScopeSummary.disabled }}</strong>
            </button>
          </div>
          <span class="sp-rate-guard-group-filter-result">
            筛选结果 <strong>{{ electionFilteredGroups.length }}</strong> 个
          </span>
          <!-- 与「配置参与守护的分组」保持同一套筛选控件与配色语言。 -->
          <div
            v-if="groupPlatformFacets.length"
            class="sp-platform-chip-row"
            role="group"
            aria-label="按平台筛选分组"
          >
            <button
              v-for="facet in groupPlatformFacets"
              :key="facet.platform"
              class="sp-platform-chip"
              type="button"
              :aria-pressed="electionGroupPlatformFilter.includes(facet.platform)"
              :style="{ '--sp-chip-platform-color': platformAccentColor(facet.platform) }"
              @click="electionGroupPlatformFilter = togglePlatformFilter(electionGroupPlatformFilter, facet.platform)"
            >
              {{ facet.label }}
              <strong>{{ facet.count }}</strong>
            </button>
          </div>
        </div>

        <div v-if="loadingGroups" class="sp-rate-guard-empty">正在加载分组...</div>
        <template v-else-if="electionFilteredGroups.length">
          <!-- 批量操作条：勾选框专门用来多选，配合这里的批量开关整批改状态。
               全选只作用于「当前筛选」的结果（与工具栏筛选联动）；
               只在有勾选时展开可执行的批量动作，避免空架子占地方。 -->
          <div class="sp-group-election-batch-bar">
            <label class="sp-group-election-batch-master">
              <input
                type="checkbox"
                :checked="electionGroupAllChecked"
                :aria-label="`全选当前筛选的 ${electionFilteredGroups.length} 个分组`"
                @change="toggleElectionGroupCheckAll"
              />
              <span>全选</span>
            </label>
            <span class="sp-group-election-batch-count">已选 <strong>{{ electionGroupCheckedIDs.length }}</strong> 个</span>
            <template v-if="electionGroupCheckedIDs.length">
              <span class="sp-group-election-batch-divider"></span>
              <span class="sp-group-election-batch-label">参与择优</span>
              <button type="button" class="sp-group-election-batch-btn" @click="batchElectionGroupsSelect(true)">批量开启</button>
              <button type="button" class="sp-group-election-batch-btn" @click="batchElectionGroupsSelect(false)">批量关闭</button>
              <span class="sp-group-election-batch-label">健康锁定</span>
              <button type="button" class="sp-group-election-batch-btn" @click="batchElectionGroupsKeepHealthy(true)">批量开锁</button>
              <button type="button" class="sp-group-election-batch-btn" @click="batchElectionGroupsKeepHealthy(false)">批量解锁</button>
              <span class="sp-group-election-batch-label">演练</span>
              <button type="button" class="sp-group-election-batch-btn" @click="batchElectionGroupsDryRun(true)">批量开启</button>
              <button type="button" class="sp-group-election-batch-btn" @click="batchElectionGroupsDryRun(false)">批量关闭</button>
              <span class="sp-group-election-batch-label">计优先级</span>
              <button type="button" class="sp-group-election-batch-btn" @click="batchElectionGroupsPriority(true)">批量开启</button>
              <button type="button" class="sp-group-election-batch-btn" @click="batchElectionGroupsPriority(false)">批量关闭</button>
              <button type="button" class="sp-group-election-batch-clear" @click="clearElectionGroupChecks">清空勾选</button>
            </template>
          </div>
          <div class="sp-rate-guard-group-list">
          <article
            v-for="group in electionFilteredGroups"
            :key="group.id"
            class="sp-rate-guard-group-row"
            :class="{ disabled: electionGroupIsDisabled(group.id) }"
            :style="{ '--sp-group-platform-color': platformAccentColor(group.platform) }"
          >
            <label class="sp-rate-guard-group-choice">
              <!-- 勾选框只负责「多选 + 批量操作」：是否参与择优改由行内「参与择优」开关控制，
                   两者不再共用同一个框，选中不等同于参与，避免误改。 -->
              <input
                type="checkbox"
                :checked="electionGroupIsChecked(group.id)"
                :aria-label="`选择分组 ${group.name} 用于批量操作`"
                @change="toggleElectionGroupCheck(group.id)"
              />
              <span class="sp-rate-guard-group-choice-copy">
                <strong :class="platformTextClass(group.platform)">{{ group.name }}</strong>
                <span class="sp-rate-guard-group-platform" :class="platformBadgeClass(group.platform)">
                  {{ platformLabel(group.platform) }}
                </span>
                <span class="sp-rate-guard-group-id">#{{ group.id }}</span>
                <span class="sp-rate-guard-group-rate">倍率 {{ group.rate_multiplier }}</span>
              </span>
            </label>
            <span
              v-if="electionGroupIsDisabled(group.id)"
              class="sp-rate-guard-group-status off"
            >
              已关闭择优
            </span>
            <label
              v-else
              class="sp-health-guard-account-scheduling-toggle sp-election-keep-healthy-toggle sp-election-toggle-health"
              :title="`打开后：分组「${group.name}」当前开着调度的账号测试都正常时，保留现状、跳过择优与换人；有开着的账号失败仍走正常择优`"
            >
              <Toggle
                :model-value="electionGroupKeepHealthy(group.id)"
                :aria-label="`分组 ${group.name} 在任者健康时是否锁定`"
                @update:model-value="toggleElectionKeepHealthyGroup(group.id)"
              />
              <span>健康锁定</span>
            </label>
            <label
              v-if="!electionGroupIsDisabled(group.id)"
              class="sp-health-guard-account-scheduling-toggle sp-election-dry-run-toggle sp-election-toggle-dryrun"
              :title="`打开后：分组「${group.name}」只给出建议、不真的改调度（总开关打开时全部分组都演练）`"
            >
              <Toggle
                :model-value="electionGroupDryRun(group.id)"
                :aria-label="`分组 ${group.name} 是否只演练不生效`"
                @update:model-value="toggleElectionGroupDryRun(group.id)"
              />
              <span>演练</span>
            </label>
            <label
              v-if="!electionGroupIsDisabled(group.id)"
              class="sp-health-guard-account-scheduling-toggle sp-election-keep-healthy-toggle sp-election-toggle-priority"
              :title="`打开后：分组「${group.name}」的综合分计入账号优先级（数值越小优先级越高），都健康时高优先级更占优势；默认跟随全局开关，逐个分组可覆盖`"
            >
              <Toggle
                :model-value="electionGroupPriority(group.id)"
                :aria-label="`分组 ${group.name} 是否计入账号优先级`"
                @update:model-value="toggleElectionGroupPriority(group.id)"
              />
              <span>计优先级</span>
            </label>
            <!-- 参与择优开关列：不参与的分组也要能直接勾回来，所以不像其它开关那样用 v-if 隐藏 ——
                 位置固定在网格第 5 列，行与行对齐不随显隐摇晃。 -->
            <label
              class="sp-health-guard-account-scheduling-toggle sp-election-participate-toggle"
              :title="`打开后：分组「${group.name}」参与择优调度；关闭则不参与（同时清空其健康锁定/演练/计优先级配置）`"
            >
              <Toggle
                :model-value="electionGroupParticipates(group.id)"
                :aria-label="`分组 ${group.name} 是否参与择优调度`"
                @update:model-value="toggleElectionGroup(group.id)"
              />
              <span>参与择优</span>
            </label>
            <div v-if="!electionGroupIsDisabled(group.id)" class="sp-election-group-extra">
              <label
                class="sp-election-top-n-override"
                :title="`分组「${group.name}」单独设置「每组开启账号数」：留空=沿用全局默认（当前 ${config.group_scheduling_election_top_n ?? 1}），填 1–100 的正整数则覆盖全局。`"
              >
                <span class="sp-election-top-n-override-label">每组开启数</span>
                <input
                  class="sp-election-top-n-override-input"
                  type="number"
                  min="1"
                  max="100"
                  :placeholder="`全局 ${config.group_scheduling_election_top_n ?? 1}`"
                  :value="electionGroupTopNText(group.id)"
                  :aria-label="`分组 ${group.name} 单独设置的每组开启账号数`"
                  @change="setElectionGroupTopN(group.id, ($event.target as HTMLInputElement).value)"
                />
              </label>
              <label
                class="sp-election-required-models"
                :title="`分组「${group.name}」的必需模型：择优后若赢家没覆盖这些模型，会补选一个健康支持者开启；支持它的账号全失败时不硬留、只告警待恢复。逗号或空格分隔。`"
              >
                <span class="sp-election-required-models-label">必需模型</span>
                <input
                  class="sp-election-required-models-input"
                  type="text"
                  placeholder="留空=无强制要求，如 gpt-5.6, claude-opus-5"
                  :value="electionGroupRequiredModelsText(group.id)"
                  :aria-label="`分组 ${group.name} 的必需模型`"
                  @change="setElectionGroupRequiredModels(group.id, ($event.target as HTMLInputElement).value)"
                />
              </label>
              <label
                class="sp-election-default-account"
                :title="`分组「${group.name}」的默认账号：它健康（可参选）时本组只开它一个、关闭其它成员；它失败或不可用时回退正常择优。若它不支持本组必需模型，会额外补选一个支持者。`"
              >
                <span class="sp-election-default-account-label">默认账号</span>
                <Select
                  class="sp-election-default-account-select"
                  :model-value="electionGroupDefaultAccountValue(group.id)"
                  :options="electionGroupDefaultAccountOptions(group.id)"
                  :searchable="false"
                  :aria-label="`分组 ${group.name} 的默认账号`"
                  @update:model-value="setElectionGroupDefaultAccount(group.id, $event)"
                />
              </label>
              <!-- 异常推送覆盖：三段式（跟随全局 / 强制推送 / 静音），不用开关 ——
                   开关只有两态，表达不了「这个分组没配过覆盖」，而那正是绝大多数分组的状态。 -->
              <span
                class="sp-election-alert-override"
                role="group"
                :aria-label="`分组 ${group.name} 的异常推送覆盖`"
                :title="`分组「${group.name}」的异常推送：跟随全局=沿用上方总开关；强制推送=即使总开关关闭，本组样本不足时也推送；静音=本组永不推送。需先在通知页订阅「分组账号异常」事件。`"
              >
                <span class="sp-election-alert-override-label">异常推送</span>
                <span class="sp-election-alert-override-choice">
                  <button
                    type="button"
                    :class="{ active: electionAlertOverrideValue(group.id) === undefined }"
                    @click="setElectionAlertOverride(group.id, undefined)"
                  >跟随全局</button>
                  <button
                    type="button"
                    :class="{ active: electionAlertOverrideValue(group.id) === true }"
                    @click="setElectionAlertOverride(group.id, true)"
                  >强制推送</button>
                  <button
                    type="button"
                    :class="{ active: electionAlertOverrideValue(group.id) === false }"
                    @click="setElectionAlertOverride(group.id, false)"
                  >静音</button>
                </span>
              </span>
            </div>
          </article>
          </div>
        </template>
        <div v-else class="sp-rate-guard-empty">{{ electionGroupEmptyHint }}</div>
      </section>
    </div>
    <template #footer>
      <span class="sp-rate-guard-group-hint">取消勾选的分组会被跳过；「健康锁定」默认跟随全局开关，逐个分组可覆盖（关掉=强制不锁定、打开=强制锁定），开着的账号正常时保留现状、不换人；「计优先级」默认跟随全局开关，逐个分组可覆盖（打开=强制计入、关掉=强制不计）；「演练」只记录建议、不改调度。</span>
      <button class="sp-button ghost" type="button" @click="enableAllElectionGroups">全部参与</button>
      <button class="sp-button primary" type="button" @click="emit('confirm')">完成</button>
    </template>
  </BaseDialog>
</template>

<script setup lang="ts">
import { watch } from 'vue'
import BaseDialog from '@/components/common/BaseDialog.vue'
import Input from '@/components/common/Input.vue'
import Select from '@/components/common/Select.vue'
import Toggle from '@/components/common/Toggle.vue'
import type { AdminGroup } from '@/types'
import type { SupplierProviderAccount } from '@/api/admin/supplierProviderData'
import type { SupplierAutomationConfig } from '@/api/admin/supplierAutomation'
import { resolvePlatformDisplayLabel as platformLabel } from '@/utils/customPlatformLabels'
import { platformAccentColor, platformBadgeClass, platformTextClass } from '@/utils/platformColors'
import { togglePlatformFilter } from '@/views/admin/supplier-management/supplierAutomationConfig'
import { useGroupElectionConfig } from '@/views/admin/supplier-management/useGroupElectionConfig'

const props = defineProps<{
  show: boolean
  /** 可选分组（含停用分组）——「已关闭择优」的分组若此刻正被停用，也要能看见它的开关状态。 */
  groups: AdminGroup[]
  loadingGroups: boolean
  /** 账号候选：只服务「默认账号」下拉里的「分组 → 成员账号」。 */
  accounts: SupplierProviderAccount[]
}>()

// 配置草稿用 v-model 双向绑定：本弹窗没有自己的「保存」，它做的就是就地修改调用方持有的那份 config
// （自动化页的任务编辑弹窗、或上游账号页的配置草稿）。用 defineModel 既能直接改草稿，
// 又不会触发 vue/no-mutating-props —— 改的是 ref 里的那个对象，不是 prop 本身。
const config = defineModel<SupplierAutomationConfig>('config', { required: true })

// 「完成」与「关闭」必须分开：本弹窗没有自己的保存动作，改的是调用方持有的 config。
// 在自动化页两者等价（都只是收起弹窗，由任务编辑弹窗的「保存」统一提交）；
// 但上游账号页是原地打开、没有父级「保存」兜底，只能靠「完成」落库 ——
// 若把它和 ✕ / 点遮罩 / Esc 混成一个事件，就会变成「点一下遮罩就把配置写进库」。
const emit = defineEmits<{ close: []; confirm: [] }>()

// 派生值与分组级操作全在共享 composable 里：自动化页的「分组择优调度策略」摘要、保存时的归一化、
// 以及本弹窗，三处读同一份口径；上游账号页复用本弹窗时也走同一份。
const {
  groupPlatformFacets,
  electionGroupSearch,
  electionGroupDisabledOnly,
  electionGroupPlatformFilter,
  electionGroupCheckedIDs,
  electionGroupScopeSummary,
  electionFilteredGroups,
  electionGroupAllChecked,
  electionGroupEmptyHint,
  electionGroupIsDisabled,
  electionGroupParticipates,
  toggleElectionGroup,
  electionGroupKeepHealthy,
  electionAlertOverrideValue,
  setElectionAlertOverride,
  electionGroupDefaultAccountValue,
  electionGroupDefaultAccountOptions,
  setElectionGroupDefaultAccount,
  electionGroupDryRun,
  toggleElectionGroupDryRun,
  toggleElectionKeepHealthyGroup,
  electionGroupPriority,
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
  batchElectionGroupsSelect,
  batchElectionGroupsKeepHealthy,
  batchElectionGroupsDryRun,
  batchElectionGroupsPriority,
  resetElectionDialogFilters,
} = useGroupElectionConfig(
  () => config.value,
  () => props.groups,
  () => props.accounts
)

function close() {
  emit('close')
}

// 每次打开都把上一次的筛选与勾选清掉（口径与原页面 openElectionGroups 里的重置一致，只是搬了家）。
// 数据加载仍由调用方负责 —— 只有它知道该不该重新拉分组与账号。
watch(
  () => props.show,
  visible => {
    if (visible) resetElectionDialogFilters()
  }
)
</script>

<style scoped>
:global(.modal-content:has(.sp-health-guard-account-dialog) .modal-body),
:global(.modal-content:has(.sp-rate-guard-group-dialog) .modal-body) {
  overflow: hidden;
}

:global(.modal-content:has(.sp-rate-guard-group-dialog)) {
  --sp-panel: #ffffff;
  --sp-panel-2: #f8fafc;
  --sp-panel-3: #eef2f7;
  --sp-line: #d7e0ea;
  --sp-soft: #e8eef5;
  --sp-text: #172033;
  --sp-muted: #607089;
  --sp-cyan: #0284c7;
  --sp-green: #16835d;
  --sp-amber: #c56a0a;
  --sp-orange: #dd5f16;
  --sp-red: #d14343;
  --sp-blue: #2563eb;
  --sp-violet: #6d5bd0;
  --sp-result-blue-soft: #eaf2ff;
  --sp-result-cyan-soft: #e6f6fb;
  --sp-result-green-soft: #e8f7ef;
  --sp-result-amber-soft: #fff3dc;
  --sp-result-red-soft: #fff0f0;
  --sp-result-violet-soft: #f1efff;
  --sp-result-neutral-soft: #f3f6fa;
  overflow: hidden;
  border-color: #cbd7e5;
  background: var(--sp-panel);
  color: var(--sp-text);
  height: min(760px, calc(100dvh - 1rem));
  max-height: min(760px, calc(100dvh - 1rem));
}

:global(.dark .modal-content:has(.sp-rate-guard-group-dialog)) {
  --sp-panel: #172033;
  --sp-panel-2: #1d293d;
  --sp-panel-3: #243249;
  --sp-line: #35445c;
  --sp-soft: #2c3a51;
  --sp-text: #edf3fb;
  --sp-muted: #a8b6ca;
  --sp-result-blue-soft: #1b3155;
  --sp-result-cyan-soft: #153947;
  --sp-result-green-soft: #173a31;
  --sp-result-amber-soft: #432f1d;
  --sp-result-red-soft: #48272d;
  --sp-result-violet-soft: #302b51;
  --sp-result-neutral-soft: #202d42;
  border-color: #3b4b64;
}

:global(.modal-content:has(.sp-rate-guard-group-dialog) .modal-header) {
  border-bottom-color: var(--sp-line);
  background: var(--sp-panel);
}

:global(.modal-content:has(.sp-rate-guard-group-dialog) .modal-title) {
  color: var(--sp-text);
}

:global(.modal-content:has(.sp-rate-guard-group-dialog) .modal-body) {
  display: flex;
  min-height: 0;
  flex: 1 1 auto;
  flex-direction: column;
  overflow: hidden;
  background: var(--sp-panel);
}

:global(.modal-content:has(.sp-rate-guard-group-dialog) .modal-footer) {
  border-top-color: var(--sp-line);
  background: var(--sp-panel);
}

@media (min-width: 768px) {
  :global(.modal-content:has(.sp-rate-guard-group-dialog)) {
      width: 90vw;
      max-width: 90vw;
    }
}

@media (min-width: 640px) {
  :global(.modal-content:has(.sp-rate-guard-group-dialog)) {
      height: 95vh;
      max-height: 95vh;
    }
}

.sp-rate-guard-group-dialog {
  display: flex;
  flex: 1 1 auto;
  flex-direction: column;
  min-height: 0;
}

.sp-rate-guard-group-workspace {
  display: flex;
  flex: 1 1 auto;
  flex-direction: column;
  min-height: 0;
  border: 1px solid var(--sp-line);
  border-radius: 12px;
  overflow: hidden;
  background: var(--sp-panel);
}

.sp-rate-guard-group-summary {
  display: grid;
  flex: 0 0 auto;
  grid-template-columns: repeat(3, minmax(0, 1fr));
  border-bottom: 1px solid var(--sp-line);
  background: color-mix(in srgb, var(--sp-soft) 22%, transparent);
}

.sp-rate-guard-group-summary article {
  --sp-summary-accent: var(--sp-cyan);

  position: relative;
  display: flex;
  align-items: baseline;
  justify-content: flex-start;
  min-width: 0;
  gap: 8px;
  border-left: 1px solid var(--sp-line);
  padding: 12px 14px 11px;
}

.sp-rate-guard-group-summary article:first-child {
  border-left: 0;
}

.sp-rate-guard-group-summary article:nth-child(1) {
  --sp-summary-accent: var(--sp-cyan);
}

.sp-rate-guard-group-summary article:nth-child(2) {
  --sp-summary-accent: var(--sp-muted);
}

.sp-rate-guard-group-summary article:nth-child(3) {
  --sp-summary-accent: var(--sp-blue);
}

.sp-rate-guard-group-summary article::before {
  content: '';
  position: absolute;
  top: 0;
  right: 0;
  left: 0;
  height: 3px;
  background: var(--sp-summary-accent);
  opacity: 0.85;
}

.sp-rate-guard-group-summary article span {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  color: color-mix(in srgb, var(--sp-summary-accent) 64%, var(--sp-muted));
  font-size: 12px;
}

.sp-rate-guard-group-summary article span::before {
  content: '';
  width: 6px;
  height: 6px;
  border-radius: 999px;
  background: var(--sp-summary-accent);
  opacity: 0.9;
}

.sp-rate-guard-group-summary article strong {
  color: var(--sp-summary-accent);
  font-size: 18px;
  font-weight: 750;
  font-variant-numeric: tabular-nums;
  line-height: 1;
}

.sp-rate-guard-group-toolbar {
  display: flex;
  flex: 0 0 auto;
  flex-wrap: wrap;
  align-items: center;
  justify-content: space-between;
  gap: 8px 12px;
  padding: 10px 14px;
}

.sp-rate-guard-group-filters {
  display: flex;
  flex: 1 1 auto;
  align-items: center;
  gap: 10px;
  flex-wrap: wrap;
}

.sp-rate-guard-group-filters :deep(.form-field),
.sp-rate-guard-group-filters > div {
  width: 16rem;
}

.sp-rate-guard-group-selected-toggle {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  border: 1px solid var(--sp-line);
  border-radius: 8px;
  padding: 0.5rem 0.75rem;
  background: var(--sp-panel);
  color: var(--sp-muted);
  font-size: 12px;
  cursor: pointer;
  transition: border-color 160ms ease, color 160ms ease, background-color 160ms ease;
}

.sp-rate-guard-group-selected-toggle:hover {
  border-color: color-mix(in srgb, var(--sp-cyan) 30%, var(--sp-line));
  color: var(--sp-text);
}

.sp-rate-guard-group-selected-toggle strong {
  color: var(--sp-text);
  font-variant-numeric: tabular-nums;
}

.sp-rate-guard-group-selected-toggle:focus-visible {
  outline: 2px solid color-mix(in srgb, var(--sp-cyan) 40%, transparent);
  outline-offset: 2px;
}

.sp-rate-guard-group-selected-toggle-mark {
  width: 8px;
  height: 8px;
  border-radius: 50%;
  background: currentColor;
  opacity: 0.5;
}

.sp-rate-guard-group-filter-result {
  flex: 0 0 auto;
  color: var(--sp-muted);
  font-size: 12px;
}

.sp-rate-guard-group-filter-result strong {
  color: var(--sp-text);
  font-variant-numeric: tabular-nums;
}

.sp-group-election-batch-bar {
  display: flex;
  flex: 0 0 auto;
  flex-wrap: wrap;
  align-items: center;
  gap: 6px 10px;
  padding: 8px 14px;
  border-top: 1px solid var(--sp-line);
  background: color-mix(in srgb, var(--sp-cyan) 5%, var(--sp-panel));
}

.sp-group-election-batch-master {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  color: var(--sp-text);
  font-size: 12px;
  font-weight: 650;
  cursor: pointer;
}

.sp-group-election-batch-master input[type='checkbox'] {
  width: 16px;
  height: 16px;
  margin: 0;
  border-radius: 4px;
  /* 与列表行里的勾选框同一套外观：两者是同一个多选模型的两个入口，
     样式分叉会让人以为是两套互不相干的选中。 */
  accent-color: var(--sp-cyan);
  cursor: pointer;
}

.sp-group-election-batch-master input[type='checkbox']:focus-visible {
  outline: 2px solid color-mix(in srgb, var(--sp-cyan) 40%, transparent);
  outline-offset: 2px;
}

.sp-group-election-batch-count {
  color: var(--sp-muted);
  font-size: 12px;
}

.sp-group-election-batch-count strong {
  color: var(--sp-text);
  font-variant-numeric: tabular-nums;
}

.sp-group-election-batch-divider {
  width: 1px;
  height: 16px;
  background: var(--sp-line);
}

.sp-group-election-batch-label {
  color: var(--sp-muted);
  font-size: 12px;
  font-weight: 650;
}

.sp-group-election-batch-btn {
  border: 1px solid var(--sp-line);
  border-radius: 8px;
  padding: 0.2rem 0.6rem;
  background: var(--sp-panel);
  color: var(--sp-text);
  font-size: 12px;
  cursor: pointer;
  transition: border-color 160ms ease, color 160ms ease, background-color 160ms ease;
}

.sp-group-election-batch-btn:hover {
  border-color: color-mix(in srgb, var(--sp-cyan) 40%, var(--sp-line));
  background: color-mix(in srgb, var(--sp-cyan) 8%, var(--sp-panel));
  color: var(--sp-cyan);
}

.sp-group-election-batch-btn:focus-visible {
  outline: 2px solid color-mix(in srgb, var(--sp-cyan) 40%, transparent);
  outline-offset: 2px;
}

.sp-group-election-batch-clear {
  margin-left: auto;
  border: 0;
  padding: 0.2rem 0.2rem;
  background: transparent;
  color: var(--sp-muted);
  font-size: 12px;
  text-decoration: underline;
  cursor: pointer;
}

.sp-group-election-batch-clear:hover {
  color: var(--sp-text);
}

.sp-platform-chip-row {
  display: flex;
  flex: 1 1 100%;
  flex-wrap: wrap;
  align-items: center;
  gap: 6px;
  min-width: 0;
}

.sp-platform-chip {
  display: inline-flex;
  align-items: center;
  gap: 5px;
  border: 1px solid var(--sp-line);
  border-radius: 8px;
  padding: 0.28rem 0.6rem;
  background: var(--sp-panel);
  color: var(--sp-muted);
  font-size: 12px;
  cursor: pointer;
  transition: border-color 160ms ease, color 160ms ease, background-color 160ms ease;
}

.sp-platform-chip:hover {
  border-color: color-mix(in srgb, var(--sp-chip-platform-color) 30%, var(--sp-line));
  color: var(--sp-text);
}

.sp-platform-chip[aria-pressed='true'] {
  border-color: color-mix(in srgb, var(--sp-chip-platform-color) 55%, var(--sp-line));
  background: color-mix(in srgb, var(--sp-chip-platform-color) 16%, var(--sp-panel));
  color: var(--sp-text);
  font-weight: 650;
}

.sp-platform-chip:focus-visible {
  outline: 2px solid color-mix(in srgb, var(--sp-chip-platform-color) 40%, transparent);
  outline-offset: 2px;
}

.sp-platform-chip strong {
  color: inherit;
  font-variant-numeric: tabular-nums;
  opacity: 0.7;
}

.sp-rate-guard-group-list {
  flex: 1 1 auto;
  min-height: 0;
  overflow: auto;
  overscroll-behavior: contain;
  border-top: 1px solid var(--sp-line);
  background: color-mix(in srgb, var(--sp-soft) 12%, var(--sp-panel));
  scrollbar-width: thin;
  scrollbar-color: color-mix(in srgb, var(--sp-line) 82%, transparent) transparent;
}

.sp-rate-guard-group-list::-webkit-scrollbar {
  width: 8px;
}

.sp-rate-guard-group-list::-webkit-scrollbar-thumb {
  border: 2px solid transparent;
  border-radius: 999px;
  background: color-mix(in srgb, var(--sp-line) 85%, transparent);
  background-clip: padding-box;
}

.sp-rate-guard-group-row {
  position: relative;
  display: flex;
  align-items: center;
  justify-content: space-between;
  min-width: 0;
  gap: 8px 12px;
  border-bottom: 1px solid var(--sp-line);
  padding: 10px 14px;
  background: var(--sp-panel);
  transition: background-color 160ms ease, box-shadow 160ms ease;
}

.sp-rate-guard-group-row:has(.sp-election-participate-toggle) {
  display: grid;
  grid-template-columns: minmax(0, 1fr) auto auto auto auto;
  align-items: center;
  column-gap: 14px;
}

.sp-rate-guard-group-row:has(.sp-election-participate-toggle) > .sp-rate-guard-group-choice {
  grid-column: 1;
}

.sp-rate-guard-group-row:has(.sp-election-participate-toggle) > .sp-rate-guard-group-status,
.sp-rate-guard-group-row:has(.sp-election-participate-toggle) > .sp-election-toggle-health {
  grid-column: 2;
}

.sp-rate-guard-group-row:has(.sp-election-participate-toggle) > .sp-election-toggle-dryrun {
  grid-column: 3;
}

.sp-rate-guard-group-row:has(.sp-election-participate-toggle) > .sp-election-toggle-priority {
  grid-column: 4;
}

.sp-rate-guard-group-row:has(.sp-election-participate-toggle) > .sp-election-participate-toggle {
  grid-column: 5;
}

.sp-rate-guard-group-row:has(.sp-election-participate-toggle) > .sp-election-group-extra {
  grid-column: 1 / -1;
}

.sp-rate-guard-group-row .sp-election-participate-toggle {
  color: var(--sp-text);
  font-weight: 700;
}

.sp-rate-guard-group-row:last-child {
  border-bottom: 0;
}

.sp-rate-guard-group-row:hover {
  background: color-mix(in srgb, var(--sp-cyan) 4%, var(--sp-panel));
}

.sp-rate-guard-group-row::before {
  content: '';
  position: absolute;
  top: 0;
  bottom: 0;
  left: 0;
  width: 3px;
  background: var(--sp-group-platform-color, transparent);
}

.sp-rate-guard-group-choice {
  display: grid;
  min-width: 0;
  flex: 1 1 auto;
  grid-template-columns: 16px minmax(0, 1fr);
  align-items: center;
  gap: 9px;
  cursor: pointer;
}

.sp-rate-guard-group-choice input[type='checkbox'] {
  width: 16px;
  height: 16px;
  /* 不依赖浏览器默认外观：否则各平台未勾选态的方框粗细、圆角都不一样，
     而这一列正是用户判断"这个分组到底参不参与"的唯一依据，必须稳定可辨。 */
  accent-color: var(--sp-cyan);
  border-radius: 4px;
  cursor: pointer;
  margin: 0;
}

.sp-rate-guard-group-choice input[type='checkbox']:focus-visible {
  outline: 2px solid color-mix(in srgb, var(--sp-cyan) 40%, transparent);
  outline-offset: 2px;
}

.sp-rate-guard-group-choice-copy {
  display: flex;
  align-items: center;
  gap: 8px;
  min-width: 0;
  flex-wrap: wrap;
}

.sp-rate-guard-group-choice-copy strong {
  font-size: 13px;
  max-width: 100%;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.sp-rate-guard-group-id,
.sp-rate-guard-group-rate {
  color: var(--sp-muted);
  font-size: 12px;
  font-variant-numeric: tabular-nums;
  white-space: nowrap;
}

.sp-rate-guard-group-rate {
  border-radius: 6px;
  padding: 1px 6px;
  background: color-mix(in srgb, var(--sp-line) 55%, transparent);
}

.sp-rate-guard-group-platform {
  display: inline-flex;
  flex: 0 0 auto;
  align-items: center;
  border-width: 1px;
  border-radius: 4px;
  padding: 0 5px;
  font-size: 10px;
  font-weight: 700;
  line-height: 1.4;
  white-space: nowrap;
}

.sp-rate-guard-group-status {
  flex: 0 0 auto;
  border: 1px solid color-mix(in srgb, var(--sp-amber) 38%, var(--sp-line));
  border-radius: 6px;
  padding: 2px 8px;
  background: color-mix(in srgb, var(--sp-amber) 8%, transparent);
  color: var(--sp-amber);
  font-size: 12px;
  font-weight: 600;
  white-space: nowrap;
}

.sp-rate-guard-group-hint {
  margin-right: auto;
  color: var(--sp-muted);
  font-size: 12px;
}

.sp-health-guard-account-scheduling-toggle,
.sp-health-guard-account-guard-toggle {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  color: var(--sp-muted);
  cursor: pointer;
  font-size: 11px;
  font-weight: 650;
  line-height: 1.2;
  white-space: nowrap;
}

.sp-election-group-extra {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 16px;
  min-width: 0;
  padding-left: 24px;
}

.sp-election-required-models {
  flex: 1 1 auto;
  min-width: 180px;
  display: flex;
  align-items: center;
  gap: 8px;
  min-width: 0;
}

.sp-election-required-models-label {
  flex: 0 0 auto;
  font-size: 12px;
  color: var(--sp-muted);
}

.sp-election-required-models-input {
  flex: 1 1 auto;
  min-width: 0;
  padding: 4px 8px;
  font-size: 12px;
  color: var(--sp-text);
  background: var(--sp-panel);
  border: 1px solid var(--sp-line);
  border-radius: 6px;
}

.sp-election-required-models-input:focus {
  outline: none;
  border-color: var(--sp-cyan);
}

.sp-election-top-n-override {
  flex: 0 0 auto;
  display: flex;
  align-items: center;
  gap: 8px;
}

.sp-election-top-n-override-label {
  flex: 0 0 auto;
  font-size: 12px;
  color: var(--sp-muted);
}

.sp-election-top-n-override-input {
  flex: 0 0 auto;
  width: 120px;
  padding: 4px 8px;
  font-size: 12px;
  color: var(--sp-text);
  background: var(--sp-panel);
  border: 1px solid var(--sp-line);
  border-radius: 6px;
}

.sp-election-top-n-override-input:focus {
  outline: none;
  border-color: var(--sp-cyan);
}

.sp-election-default-account {
  flex: 0 0 auto;
  display: flex;
  align-items: center;
  gap: 8px;
}

.sp-election-default-account-label {
  flex: 0 0 auto;
  font-size: 12px;
  color: var(--sp-muted);
}

.sp-election-default-account-select {
  width: 180px;
}

.sp-election-alert-override {
  flex: 0 0 auto;
  display: flex;
  align-items: center;
  gap: 8px;
}

.sp-election-alert-override-label {
  flex: 0 0 auto;
  font-size: 12px;
  color: var(--sp-muted);
}

.sp-election-alert-override-choice {
  display: inline-flex;
  flex: 0 0 auto;
  gap: 4px;
  padding: 2px;
  border: 1px solid var(--sp-line);
  border-radius: 10px;
  background: color-mix(in srgb, var(--sp-soft) 18%, var(--sp-panel));
}

.sp-election-alert-override-choice button {
  border: none;
  border-radius: 8px;
  padding: 4px 10px;
  background: transparent;
  color: var(--sp-muted);
  font-size: 12px;
  cursor: pointer;
}

.sp-rate-guard-empty {
  border-top: 1px solid var(--sp-line);
  border-bottom: 1px solid var(--sp-line);
  color: var(--sp-muted);
  padding: 16px 0;
}

.sp-election-dry-run-toggle {
  display: inline-flex;
  align-items: center;
  gap: 8px;
  flex: 0 0 auto;
  font-size: 12px;
  color: var(--sp-muted, #64748b);
}
</style>

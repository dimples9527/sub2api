<template>
  <SupplierModuleLayout>
    <header class="sp-page-head">
      <div>
        <div class="sp-eyebrow">Supplier Group Matching</div>
        <h1>分组管理</h1>
        <p class="sp-subtitle">对照最近一次采集到的上游分组与本地分组，集中处理倍率和匹配关系。</p>
      </div>
      <div class="sp-controls">
        <Select v-model="providerID" class="sp-search" :options="providerOptions" :searchable="false" />
        <Input v-model="search" class="sp-search" placeholder="搜索上游分组或 Key" />
        <button class="sp-button small sp-control-button" type="button" :disabled="loading || !canResetFilters" @click="resetGroupFilters">
          <Icon name="x" size="sm" />
          <span>重置筛选</span>
        </button>
        <button class="sp-button sp-control-button" type="button" :disabled="loading" @click="refreshAll">
          <Icon name="refresh" size="sm" :class="loading ? 'sp-spin' : ''" />
          <span>刷新</span>
        </button>
      </div>
    </header>

    <div v-if="error" class="sp-alert sp-error-line">{{ error }}</div>

    <div class="sp-console-shell">
      <div class="sp-summary-grid" aria-label="分组匹配汇总">
        <StatCard
          title="上游分组"
          :value="groupSummary.group_count"
          :icon="UpstreamGroupsIcon"
          icon-variant="primary"
        />
        <StatCard
          title="已匹配"
          :value="groupSummary.linked_group_count"
          :icon="MatchedGroupsIcon"
          icon-variant="success"
          :change="matchedGroupRate"
          change-type="neutral"
        />
        <StatCard
          title="待匹配"
          :value="groupSummary.unlinked_group_count"
          :icon="UnmatchedGroupsIcon"
          :icon-variant="groupSummary.unlinked_group_count > 0 ? 'warning' : 'primary'"
          :change="unmatchedGroupRate"
          change-type="neutral"
        />
        <StatCard
          title="倒挂风险"
          :value="groupSummary.rate_risk_count"
          :icon="RateRiskIcon"
          :icon-variant="groupSummary.rate_risk_count > 0 ? 'danger' : 'success'"
        />
      </div>

      <div class="sp-console-panel sp-panel sp-groups-panel">
        <header class="sp-panel-head">
          <div class="sp-panel-title">
            <span class="sp-section-index">01</span>
            <div>
              <h2>上游与本地分组对照</h2>
              <span>共 {{ total }} 条，当前页 {{ items.length }} 条</span>
            </div>
          </div>
          <div class="sp-panel-signals" aria-label="当前页状态">
            <span><i class="good"></i>{{ currentPageMatchedCount }} 已匹配</span>
            <span><i class="warn"></i>{{ currentPageAttentionCount }} 待处理</span>
          </div>
        </header>

        <div class="sp-table-shell">
          <DataTable
            :columns="groupColumns"
            :data="items"
            :loading="loading"
            row-key="id"
            clickable-rows
            @row-click="selected = $event"
          >
            <template #cell-provider_name="{ row: group }">
              <div :class="['sp-supplier-chip', supplierTone(group.provider_id).chip]">
                <i :class="['sp-supplier-dot', supplierTone(group.provider_id).dot]"></i>
                <span class="sp-supplier-name">{{ group.provider_name }}</span>
              </div>
              <div class="sp-sub sp-provider-meta">
                <span :class="['sp-provider-type', supplierTypeTone(group.provider_id)]">{{ supplierTypeLabel(group.provider_id) }}</span>
                <span>#{{ group.provider_id }}</span>
              </div>
            </template>

            <template #cell-name="{ row: group }">
              <div :class="['sp-upstream-group-chip', upstreamGroupTone(group.upstream_group_key).chip]">
                <i
                  :class="['sp-upstream-group-accent', upstreamGroupTone(group.upstream_group_key).accent]"
                  aria-hidden="true"
                ></i>
                <span class="sp-group-name">{{ group.name || '未命名分组' }}</span>
                <span
                  :class="[
                    'sp-inline-platform',
                    group.local_group_platform
                      ? platformTextClass(group.local_group_platform)
                      : 'sp-inline-platform-muted',
                  ]"
                >【{{ upstreamPlatformLabel(group) }}】</span>
              </div>
              <div :class="['sp-sub', 'sp-key', upstreamGroupTone(group.upstream_group_key).meta]">
                {{ group.upstream_group_key }}
              </div>
            </template>

            <template #cell-rate_multiplier="{ row: group }">
              <span :class="['sp-rate-value', upstreamRateTone(group.rate_multiplier)]">{{ formatRate(group.rate_multiplier) }}</span>
            </template>

            <template #cell-raw_status="{ row: group }">
              <span class="sp-status" :class="upstreamStatusTone(group)">
                <i></i>{{ upstreamStatusLabel(group) }}
              </span>
            </template>

            <template #cell-local_group_name="{ row: group }">
              <div v-if="group.local_group_id" class="sp-local-group">
                <GroupBadge
                  :name="group.local_group_name || `本地分组 #${group.local_group_id}`"
                  :platform="groupPlatform(group.local_group_platform)"
                  :show-rate="false"
                  class="sp-local-group-badge"
                />
                <div class="sp-sub">
                  <span :class="platformTextClass(group.local_group_platform || '')">
                    {{ platformLabel(group.local_group_platform) }}
                  </span>
                  <span>#{{ group.local_group_id }}</span>
                </div>
              </div>
              <button v-else type="button" class="sp-inline-empty" @click.stop="openMappingDialog(group)">
                <Icon name="link" size="sm" />
                <span>未匹配</span>
              </button>
            </template>

            <template #cell-local_rate_multiplier="{ row: group }">
              <span v-if="group.local_rate_multiplier != null" class="sp-rate-value local">
                {{ formatRate(group.local_rate_multiplier) }}
              </span>
              <span v-else class="sp-empty-value">-</span>
            </template>

            <template #cell-profit_rate="{ row: group }">
              <div v-if="rateInsight(group).ratio != null" class="sp-profit-cell" :class="rateInsight(group).code">
                <strong>{{ formatProfitRate(rateInsight(group).ratio) }}</strong>
                <span>{{ formatRateDelta(group) }}</span>
              </div>
              <span v-else class="sp-empty-value">-</span>
            </template>

            <template #cell-account_count="{ row: group }">
              <button
                type="button"
                class="sp-account-count"
                :title="`查看 ${group.account_count} 个绑定账号`"
                @click.stop="selected = group"
              >
                <Icon name="users" size="sm" />
                <strong>{{ group.account_count }}</strong>
              </button>
            </template>

            <template #cell-rate_status="{ row: group }">
              <span class="sp-rate-status" :class="rateInsight(group).code">
                <i></i>{{ rateInsight(group).label }}
              </span>
            </template>

            <template #cell-actions="{ row: group }">
              <div class="sp-row-actions" @click.stop>
                <template v-if="!group.local_group_id">
                  <button type="button" class="sp-row-action primary" title="匹配本地分组" @click="openMappingDialog(group)">
                    <Icon name="link" size="sm" />
                    <span>匹配分组</span>
                  </button>
                  <button type="button" class="sp-row-action" title="新建本地分组" @click="openCreateDialog(group)">
                    <Icon name="plus" size="sm" />
                    <span>新建分组</span>
                  </button>
                </template>
                <template v-else>
                  <button type="button" class="sp-row-action primary" title="修改本地分组倍率" @click="openRateDialog(group)">
                    <Icon name="edit" size="sm" />
                    <span>调倍率</span>
                  </button>
                  <button type="button" class="sp-row-action" title="重新匹配本地分组" @click="openMappingDialog(group)">
                    <Icon name="refresh" size="sm" />
                    <span>重新匹配</span>
                  </button>
                  <button type="button" class="sp-row-action danger" title="解除匹配" @click="unmatchTarget = group">
                    <Icon name="x" size="sm" />
                    <span>解除匹配</span>
                  </button>
                </template>
              </div>
            </template>

            <template #empty>
              暂无可用上游分组，请先在供应商列表执行分组同步。
            </template>
          </DataTable>
        </div>

        <Pagination
          v-if="total > 0"
          class="sp-data-pagination"
          :page="page"
          :total="total"
          :page-size="pageSize"
          :show-page-size-selector="true"
          @update:page="handleGroupPageChange"
          @update:pageSize="handleGroupPageSizeChange"
        />
      </div>
    </div>

    <SupplierDrawer
      :show="Boolean(selected)"
      :title="selected?.name || selected?.upstream_group_key || ''"
      eyebrow="GROUP COMPARISON"
      @close="selected = null"
    >
      <template v-if="selected">
        <div class="sp-drawer-summary">
          <span class="sp-rate-status" :class="rateInsight(selected).code">
            <i></i>{{ rateInsight(selected).label }}
          </span>
          <strong>{{ rateInsight(selected).ratio == null ? '-' : formatProfitRate(rateInsight(selected).ratio) }}</strong>
          <small>收益倍率</small>
        </div>
        <div class="sp-detail-grid">
          <div class="sp-detail-cell"><span>供应商</span><b>{{ selected.provider_name }}</b></div>
          <div class="sp-detail-cell"><span>绑定账号</span><b>{{ selected.account_count }}</b></div>
          <div class="sp-detail-cell"><span>上游分组</span><b>{{ selected.name || '未命名分组' }}</b></div>
          <div class="sp-detail-cell"><span>上游 Key</span><b>{{ selected.upstream_group_key }}</b></div>
          <div class="sp-detail-cell"><span>上游倍率</span><b>{{ formatRate(selected.rate_multiplier) }}</b></div>
          <div class="sp-detail-cell"><span>上游状态</span><b>{{ upstreamStatusLabel(selected) }}</b></div>
          <div class="sp-detail-cell">
            <span>本地分组</span>
            <GroupBadge
              v-if="selected.local_group_id"
              :name="selected.local_group_name || `本地分组 #${selected.local_group_id}`"
              :platform="groupPlatform(selected.local_group_platform)"
              :show-rate="false"
              class="sp-detail-group-badge"
            />
            <b v-else>未匹配</b>
          </div>
          <div class="sp-detail-cell"><span>本地倍率</span><b>{{ formatRate(selected.local_rate_multiplier) }}</b></div>
          <div class="sp-detail-cell sp-detail-wide"><span>最近同步</span><b>{{ formatTime(selected.last_seen_at) }}</b></div>
        </div>
      </template>
    </SupplierDrawer>

    <BaseDialog :show="Boolean(mappingTarget)" title="匹配本地分组" width="normal" @close="closeMappingDialog">
      <template v-if="mappingTarget">
        <div class="sp-dialog-context">
          <span>{{ mappingTarget.provider_name }}</span>
          <strong>{{ mappingTarget.name || mappingTarget.upstream_group_key }}</strong>
          <small>上游倍率 {{ formatRate(mappingTarget.rate_multiplier) }}</small>
        </div>
        <div class="sp-field">
          <span>本地分组</span>
          <Select
            v-model="mappingLocalGroupID"
            :options="localGroupOptions"
            :searchable="true"
            search-placeholder="搜索本地分组"
            placeholder="请选择本地分组"
          />
        </div>
        <div v-if="mappingPreview" class="sp-match-preview">
          <div><span>平台</span><strong :class="platformTextClass(mappingPreview.platform)">{{ platformLabel(mappingPreview.platform) }}</strong></div>
          <div><span>当前倍率</span><strong>{{ formatRate(mappingPreview.rate_multiplier) }}</strong></div>
          <div><span>收益倍率</span><strong>{{ mappingProfitPreview }}</strong></div>
        </div>
      </template>
      <template #footer>
        <div class="sp-dialog-actions">
          <button type="button" class="sp-button secondary" :disabled="savingMapping" @click="closeMappingDialog">取消</button>
          <button type="button" class="sp-button" :disabled="savingMapping || !mappingLocalGroupID" @click="saveMapping">
            <Icon name="link" size="sm" />
            <span>{{ savingMapping ? '保存中' : '保存匹配' }}</span>
          </button>
        </div>
      </template>
    </BaseDialog>

    <BaseDialog :show="Boolean(createTarget)" title="新建本地分组" width="normal" @close="closeCreateDialog">
      <template v-if="createTarget">
        <div class="sp-dialog-context">
          <span>{{ createTarget.provider_name }}</span>
          <strong>{{ createTarget.name || createTarget.upstream_group_key }}</strong>
          <small>创建成功后将自动完成匹配</small>
        </div>
        <div class="sp-dialog-form">
          <Input v-model="newGroupName" label="本地分组名称" placeholder="输入本地分组名称" />
          <div class="sp-field">
            <span>平台</span>
            <Select v-model="newGroupPlatform" :options="platformOptions" :searchable="false" />
          </div>
          <Input v-model="newGroupRate" type="number" label="本地分组倍率" placeholder="输入大于 0 的倍率" />
        </div>
        <div class="sp-rate-recommendation">
          <span>上游 {{ formatRate(createTarget.rate_multiplier) }}</span>
          <Icon name="chevronRight" size="sm" />
          <strong>建议不低于 {{ formatRate(suggestedLocalRate(createTarget)) }}</strong>
        </div>
      </template>
      <template #footer>
        <div class="sp-dialog-actions">
          <button type="button" class="sp-button secondary" :disabled="creatingLocalGroup" @click="closeCreateDialog">取消</button>
          <button type="button" class="sp-button" :disabled="creatingLocalGroup" @click="createLocalGroup">
            <Icon name="plus" size="sm" />
            <span>{{ creatingLocalGroup ? '创建中' : '创建并匹配' }}</span>
          </button>
        </div>
      </template>
    </BaseDialog>

    <BaseDialog :show="Boolean(rateTarget)" title="修改本地分组倍率" width="narrow" @close="closeRateDialog">
      <template v-if="rateTarget">
        <div class="sp-dialog-context">
          <span>{{ rateTarget.local_group_name }}</span>
          <strong>当前 {{ formatRate(rateTarget.local_rate_multiplier) }}</strong>
          <small>对应上游倍率 {{ formatRate(rateTarget.rate_multiplier) }}</small>
        </div>
        <Input v-model="localRateInput" type="number" label="新倍率" placeholder="输入大于 0 的倍率" @enter="saveLocalRate" />
        <div class="sp-rate-recommendation">
          <span>修改后收益倍率</span>
          <strong>{{ localRatePreview }}</strong>
        </div>
      </template>
      <template #footer>
        <div class="sp-dialog-actions">
          <button type="button" class="sp-button secondary" :disabled="savingLocalRate" @click="closeRateDialog">取消</button>
          <button type="button" class="sp-button" :disabled="savingLocalRate" @click="saveLocalRate">
            <Icon name="edit" size="sm" />
            <span>{{ savingLocalRate ? '保存中' : '保存倍率' }}</span>
          </button>
        </div>
      </template>
    </BaseDialog>

    <ConfirmDialog
      :show="Boolean(unmatchTarget)"
      title="解除本地分组匹配"
      :message="`解除后，${unmatchTarget?.name || unmatchTarget?.upstream_group_key || '该上游分组'} 将回到待匹配状态。`"
      confirm-text="解除匹配"
      cancel-text="取消"
      danger
      @confirm="removeMapping"
      @cancel="unmatchTarget = null"
    />
  </SupplierModuleLayout>
</template>

<script setup lang="ts">
import { computed, h, nextTick, onMounted, ref, watch } from 'vue'
import { adminAPI } from '@/api/admin'
import {
  listSupplierGroups,
  updateSupplierGroupMapping,
  type SupplierProviderGroup,
  type SupplierProviderGroupSummary,
} from '@/api/admin/supplierProviderData'
import supplierProvidersAPI, { type SupplierProvider } from '@/api/admin/supplierProviders'
import { SupplierDrawer, SupplierModuleLayout } from '@/components/admin/supplier-management'
import BaseDialog from '@/components/common/BaseDialog.vue'
import ConfirmDialog from '@/components/common/ConfirmDialog.vue'
import DataTable from '@/components/common/DataTable.vue'
import GroupBadge from '@/components/common/GroupBadge.vue'
import Input from '@/components/common/Input.vue'
import Pagination from '@/components/common/Pagination.vue'
import Select, { type SelectOption } from '@/components/common/Select.vue'
import StatCard from '@/components/common/StatCard.vue'
import type { Column } from '@/components/common/types'
import Icon from '@/components/icons/Icon.vue'
import { useAppStore } from '@/stores/app'
import type { AdminGroup, GroupPlatform } from '@/types'
import { platformTextClass } from '@/utils/platformColors'
import {
  getSupplierGroupRateInsight,
  getSupplierUpstreamRateBand,
  type SupplierGroupRateInsight,
  type SupplierUpstreamRateBand,
} from './supplierGroupRates'

const EMPTY_GROUP_SUMMARY: SupplierProviderGroupSummary = {
  group_count: 0,
  account_count: 0,
  linked_group_count: 0,
  unlinked_group_count: 0,
  rate_risk_count: 0,
}

const PLATFORM_LABELS: Record<string, string> = {
  anthropic: 'Anthropic',
  openai: 'OpenAI',
  gemini: 'Gemini',
  antigravity: 'Antigravity',
  grok: 'Grok',
}

const SUPPLIER_TYPE_LABELS: Record<string, string> = {
  sub2api: 'Sub2API',
  newapi: 'NewAPI',
}

const SUPPLIER_TYPE_TONES: Record<string, string> = {
  sub2api: 'bg-teal-500/10 text-teal-600 dark:text-teal-400',
  newapi: 'bg-blue-500/10 text-blue-600 dark:text-blue-400',
}
const SUPPLIER_TYPE_TONE_DEFAULT = 'bg-slate-500/10 text-slate-600 dark:text-slate-400'

const SUPPLIER_TONES = [
  { chip: 'border-sky-500/30 bg-sky-500/10 text-sky-700 dark:text-sky-300', dot: 'bg-sky-500', meta: 'text-sky-600 dark:text-sky-400' },
  { chip: 'border-orange-500/30 bg-orange-500/10 text-orange-700 dark:text-orange-300', dot: 'bg-orange-500', meta: 'text-orange-600 dark:text-orange-400' },
  { chip: 'border-emerald-500/30 bg-emerald-500/10 text-emerald-700 dark:text-emerald-300', dot: 'bg-emerald-500', meta: 'text-emerald-600 dark:text-emerald-400' },
  { chip: 'border-violet-500/30 bg-violet-500/10 text-violet-700 dark:text-violet-300', dot: 'bg-violet-500', meta: 'text-violet-600 dark:text-violet-400' },
  { chip: 'border-rose-500/30 bg-rose-500/10 text-rose-700 dark:text-rose-300', dot: 'bg-rose-500', meta: 'text-rose-600 dark:text-rose-400' },
  { chip: 'border-cyan-500/30 bg-cyan-500/10 text-cyan-700 dark:text-cyan-300', dot: 'bg-cyan-500', meta: 'text-cyan-600 dark:text-cyan-400' },
  { chip: 'border-amber-500/30 bg-amber-500/10 text-amber-700 dark:text-amber-300', dot: 'bg-amber-500', meta: 'text-amber-600 dark:text-amber-400' },
  { chip: 'border-indigo-500/30 bg-indigo-500/10 text-indigo-700 dark:text-indigo-300', dot: 'bg-indigo-500', meta: 'text-indigo-600 dark:text-indigo-400' },
  { chip: 'border-lime-500/30 bg-lime-500/10 text-lime-700 dark:text-lime-300', dot: 'bg-lime-500', meta: 'text-lime-600 dark:text-lime-400' },
  { chip: 'border-fuchsia-500/30 bg-fuchsia-500/10 text-fuchsia-700 dark:text-fuchsia-300', dot: 'bg-fuchsia-500', meta: 'text-fuchsia-600 dark:text-fuchsia-400' },
  { chip: 'border-teal-500/30 bg-teal-500/10 text-teal-700 dark:text-teal-300', dot: 'bg-teal-500', meta: 'text-teal-600 dark:text-teal-400' },
  { chip: 'border-red-500/30 bg-red-500/10 text-red-700 dark:text-red-300', dot: 'bg-red-500', meta: 'text-red-600 dark:text-red-400' },
] as const

const UPSTREAM_GROUP_TONES = [
  { chip: 'bg-blue-500/10 text-blue-800 dark:text-blue-200', accent: 'bg-blue-500', meta: 'text-blue-600 dark:text-blue-400' },
  { chip: 'bg-orange-500/10 text-orange-800 dark:text-orange-200', accent: 'bg-orange-500', meta: 'text-orange-600 dark:text-orange-400' },
  { chip: 'bg-emerald-500/10 text-emerald-800 dark:text-emerald-200', accent: 'bg-emerald-500', meta: 'text-emerald-600 dark:text-emerald-400' },
  { chip: 'bg-violet-500/10 text-violet-800 dark:text-violet-200', accent: 'bg-violet-500', meta: 'text-violet-600 dark:text-violet-400' },
  { chip: 'bg-rose-500/10 text-rose-800 dark:text-rose-200', accent: 'bg-rose-500', meta: 'text-rose-600 dark:text-rose-400' },
  { chip: 'bg-cyan-500/10 text-cyan-800 dark:text-cyan-200', accent: 'bg-cyan-500', meta: 'text-cyan-600 dark:text-cyan-400' },
  { chip: 'bg-amber-500/10 text-amber-800 dark:text-amber-200', accent: 'bg-amber-500', meta: 'text-amber-600 dark:text-amber-400' },
  { chip: 'bg-indigo-500/10 text-indigo-800 dark:text-indigo-200', accent: 'bg-indigo-500', meta: 'text-indigo-600 dark:text-indigo-400' },
  { chip: 'bg-lime-500/10 text-lime-800 dark:text-lime-200', accent: 'bg-lime-500', meta: 'text-lime-600 dark:text-lime-400' },
  { chip: 'bg-fuchsia-500/10 text-fuchsia-800 dark:text-fuchsia-200', accent: 'bg-fuchsia-500', meta: 'text-fuchsia-600 dark:text-fuchsia-400' },
  { chip: 'bg-teal-500/10 text-teal-800 dark:text-teal-200', accent: 'bg-teal-500', meta: 'text-teal-600 dark:text-teal-400' },
  { chip: 'bg-red-500/10 text-red-800 dark:text-red-200', accent: 'bg-red-500', meta: 'text-red-600 dark:text-red-400' },
] as const

const UPSTREAM_RATE_TONES: Record<SupplierUpstreamRateBand, string> = {
  invalid: 'text-slate-500 dark:text-slate-400',
  low: 'text-emerald-600 dark:text-emerald-400',
  standard: 'text-sky-600 dark:text-sky-400',
  elevated: 'text-amber-600 dark:text-amber-400',
  high: 'text-rose-600 dark:text-rose-400',
}

const UpstreamGroupsIcon = () => h(Icon, { name: 'filter', size: 'lg' })
const MatchedGroupsIcon = () => h(Icon, { name: 'link', size: 'lg' })
const UnmatchedGroupsIcon = () => h(Icon, { name: 'exclamationCircle', size: 'lg' })
const RateRiskIcon = () => h(Icon, { name: 'exclamationCircle', size: 'lg' })

const appStore = useAppStore()
const providers = ref<SupplierProvider[]>([])
const localGroups = ref<AdminGroup[]>([])
const items = ref<SupplierProviderGroup[]>([])
const groupSummary = ref<SupplierProviderGroupSummary>({ ...EMPTY_GROUP_SUMMARY })
const selected = ref<SupplierProviderGroup | null>(null)
const mappingTarget = ref<SupplierProviderGroup | null>(null)
const createTarget = ref<SupplierProviderGroup | null>(null)
const rateTarget = ref<SupplierProviderGroup | null>(null)
const unmatchTarget = ref<SupplierProviderGroup | null>(null)
const mappingLocalGroupID = ref<number | null>(null)
const newGroupName = ref('')
const newGroupPlatform = ref<string>('openai')
const newGroupRate = ref('')
const localRateInput = ref('')
const total = ref(0)
const loading = ref(false)
const savingMapping = ref(false)
const creatingLocalGroup = ref(false)
const savingLocalRate = ref(false)
const error = ref('')
const page = ref(1)
const pageSize = ref(20)
const providerID = ref(0)
const search = ref('')
let searchTimer: number | undefined
let suppressFilterWatch = false

const DEFAULT_PROVIDER_ID = 0
const providerOptions = computed<SelectOption[]>(() => [
  { value: 0, label: '全部供应商' },
  ...providers.value.map(provider => ({ value: provider.id, label: provider.name })),
])
const supplierIDs = computed(() => [...new Set(providers.value.map(provider => provider.id))].sort((left, right) => left - right))
const localGroupOptions = computed<SelectOption[]>(() => localGroups.value.map(group => ({
  value: group.id,
  label: `${group.name} · ${platformLabel(group.platform)} · ${formatRate(group.rate_multiplier)}`,
})))
const platformOptions: SelectOption[] = Object.entries(PLATFORM_LABELS).map(([value, label]) => ({ value, label }))
const groupColumns: Column[] = [
  { key: 'provider_name', label: '供应商', class: 'min-w-[150px]' },
  { key: 'name', label: '上游分组', class: 'min-w-[190px]' },
  { key: 'rate_multiplier', label: '上游倍率', class: 'min-w-[96px]' },
  { key: 'raw_status', label: '上游状态', class: 'min-w-[105px]' },
  { key: 'local_group_name', label: '匹配本地分组', class: 'min-w-[190px]' },
  { key: 'local_rate_multiplier', label: '本地分组倍率', class: 'min-w-[110px]' },
  { key: 'profit_rate', label: '收益倍率', class: 'min-w-[110px]' },
  { key: 'account_count', label: '绑定账号', class: 'min-w-[90px]' },
  { key: 'rate_status', label: '倍率状态', class: 'min-w-[110px]' },
  { key: 'actions', label: '操作', class: 'min-w-[270px]' },
]
const canResetFilters = computed(() => providerID.value !== DEFAULT_PROVIDER_ID || search.value.trim() !== '')
const matchedGroupRate = computed(() => percentage(groupSummary.value.linked_group_count, groupSummary.value.group_count))
const unmatchedGroupRate = computed(() => percentage(groupSummary.value.unlinked_group_count, groupSummary.value.group_count))
const currentPageMatchedCount = computed(() => items.value.filter(group => group.local_group_id).length)
const currentPageAttentionCount = computed(() => items.value.filter(group => {
  const code = rateInsight(group).code
  return code !== 'normal'
}).length)
const mappingPreview = computed(() => localGroups.value.find(group => group.id === Number(mappingLocalGroupID.value)) ?? null)
const mappingProfitPreview = computed(() => {
  if (!mappingTarget.value || !mappingPreview.value) return '-'
  const upstreamRate = Number(mappingTarget.value.rate_multiplier)
  if (!Number.isFinite(upstreamRate) || upstreamRate <= 0) return '-'
  return formatProfitRate(mappingPreview.value.rate_multiplier / upstreamRate)
})
const localRatePreview = computed(() => {
  if (!rateTarget.value) return '-'
  const upstreamRate = Number(rateTarget.value.rate_multiplier)
  const localRate = Number(localRateInput.value)
  if (!Number.isFinite(upstreamRate) || upstreamRate <= 0 || !Number.isFinite(localRate) || localRate <= 0) return '-'
  return formatProfitRate(localRate / upstreamRate)
})

onMounted(async () => {
  try {
    await Promise.all([loadProviders(), loadLocalGroups()])
  } catch (err) {
    appStore.showError(errorMessage(err, '加载筛选选项失败'))
  }
  await loadGroups()
})

watch(providerID, () => {
  if (suppressFilterWatch) return
  page.value = 1
  void loadGroups()
})

watch(search, () => {
  if (suppressFilterWatch) return
  window.clearTimeout(searchTimer)
  searchTimer = window.setTimeout(() => {
    page.value = 1
    void loadGroups()
  }, 350)
})

function resetGroupFilters() {
  window.clearTimeout(searchTimer)
  suppressFilterWatch = true
  providerID.value = DEFAULT_PROVIDER_ID
  search.value = ''
  page.value = 1
  void nextTick(() => {
    suppressFilterWatch = false
    void loadGroups()
  })
}

function handleGroupPageChange(nextPage: number) {
  if (nextPage === page.value) return
  page.value = nextPage
  void loadGroups()
}

function handleGroupPageSizeChange(nextPageSize: number) {
  if (nextPageSize === pageSize.value) return
  pageSize.value = nextPageSize
  page.value = 1
  void loadGroups()
}

async function loadProviders() {
  const result = await supplierProvidersAPI.list({ page: 1, page_size: 200 })
  providers.value = result.items
}

async function loadLocalGroups() {
  localGroups.value = await adminAPI.groups.getAll()
}

async function refreshAll() {
  try {
    await Promise.all([loadLocalGroups(), loadGroups()])
  } catch (err) {
    appStore.showError(errorMessage(err, '刷新分组数据失败'))
  }
}

async function loadGroups() {
  loading.value = true
  error.value = ''
  try {
    const result = await listSupplierGroups({
      provider_id: providerID.value || undefined,
      active: true,
      search: search.value.trim() || undefined,
      page: page.value,
      page_size: pageSize.value,
    })
    items.value = result.items
    total.value = result.total
    page.value = result.page
    pageSize.value = result.page_size
    groupSummary.value = result.summary
    if (selected.value) {
      selected.value = result.items.find(group => group.id === selected.value?.id) ?? selected.value
    }
  } catch (err) {
    error.value = errorMessage(err, '加载分组数据失败')
  } finally {
    loading.value = false
  }
}

function openMappingDialog(group: SupplierProviderGroup) {
  mappingTarget.value = group
  mappingLocalGroupID.value = group.local_group_id ?? null
}

function closeMappingDialog() {
  if (savingMapping.value) return
  mappingTarget.value = null
  mappingLocalGroupID.value = null
}

async function saveMapping() {
  const target = mappingTarget.value
  const localGroupID = Number(mappingLocalGroupID.value)
  if (!target || !Number.isInteger(localGroupID) || localGroupID <= 0) {
    appStore.showError('请选择要匹配的本地分组')
    return
  }
  savingMapping.value = true
  try {
    await updateSupplierGroupMapping(target.id, localGroupID)
    mappingTarget.value = null
    mappingLocalGroupID.value = null
    await loadGroups()
    appStore.showSuccess('本地分组匹配已保存')
  } catch (err) {
    appStore.showError(errorMessage(err, '保存分组匹配失败'))
  } finally {
    savingMapping.value = false
  }
}

function openCreateDialog(group: SupplierProviderGroup) {
  createTarget.value = group
  newGroupName.value = group.name?.trim() || group.upstream_group_key
  newGroupPlatform.value = 'openai'
  newGroupRate.value = String(suggestedLocalRate(group))
}

function closeCreateDialog() {
  if (creatingLocalGroup.value) return
  createTarget.value = null
}

async function createLocalGroup() {
  const target = createTarget.value
  const name = newGroupName.value.trim()
  const rate = Number(newGroupRate.value)
  if (!target || !name) {
    appStore.showError('请输入本地分组名称')
    return
  }
  if (!Number.isFinite(rate) || rate <= 0) {
    appStore.showError('本地分组倍率必须大于 0')
    return
  }
  creatingLocalGroup.value = true
  try {
    const created = await adminAPI.groups.create({
      name,
      platform: newGroupPlatform.value as GroupPlatform,
      rate_multiplier: rate,
    })
    await loadLocalGroups()
    try {
      await updateSupplierGroupMapping(target.id, created.id)
    } catch (err) {
      appStore.showError(errorMessage(err, '本地分组已创建，但自动匹配失败，请使用重新匹配'))
      createTarget.value = null
      await loadGroups()
      return
    }
    createTarget.value = null
    await loadGroups()
    appStore.showSuccess('本地分组已创建并完成匹配')
  } catch (err) {
    appStore.showError(errorMessage(err, '创建本地分组失败'))
  } finally {
    creatingLocalGroup.value = false
  }
}

function openRateDialog(group: SupplierProviderGroup) {
  rateTarget.value = group
  localRateInput.value = String(group.local_rate_multiplier ?? '')
}

function closeRateDialog() {
  if (savingLocalRate.value) return
  rateTarget.value = null
  localRateInput.value = ''
}

async function saveLocalRate() {
  const target = rateTarget.value
  const localGroupID = Number(target?.local_group_id)
  const rate = Number(localRateInput.value)
  if (!target || !Number.isInteger(localGroupID) || localGroupID <= 0) return
  if (!Number.isFinite(rate) || rate <= 0) {
    appStore.showError('本地分组倍率必须大于 0')
    return
  }
  savingLocalRate.value = true
  try {
    await adminAPI.groups.update(localGroupID, { rate_multiplier: rate })
    rateTarget.value = null
    localRateInput.value = ''
    await Promise.all([loadLocalGroups(), loadGroups()])
    appStore.showSuccess('本地分组倍率已更新')
  } catch (err) {
    appStore.showError(errorMessage(err, '修改本地分组倍率失败'))
  } finally {
    savingLocalRate.value = false
  }
}

async function removeMapping() {
  const target = unmatchTarget.value
  if (!target) return
  unmatchTarget.value = null
  try {
    await updateSupplierGroupMapping(target.id, null)
    await loadGroups()
    appStore.showSuccess('本地分组匹配已解除')
  } catch (err) {
    appStore.showError(errorMessage(err, '解除分组匹配失败'))
  }
}

function rateInsight(group: SupplierProviderGroup): SupplierGroupRateInsight {
  return getSupplierGroupRateInsight({
    localGroupID: group.local_group_id,
    upstreamRate: Number(group.rate_multiplier),
    localRate: group.local_rate_multiplier,
    localStatus: group.local_group_status || '',
  })
}

function supplierTone(providerID: number) {
  const providerIndex = supplierIDs.value.indexOf(providerID)
  const toneIndex = providerIndex >= 0 ? providerIndex : Math.abs(providerID)
  return SUPPLIER_TONES[toneIndex % SUPPLIER_TONES.length]
}

function upstreamGroupTone(groupKey: string) {
  let hash = 0
  for (const char of groupKey) {
    hash = ((hash << 5) - hash + char.charCodeAt(0)) | 0
  }
  return UPSTREAM_GROUP_TONES[(hash >>> 0) % UPSTREAM_GROUP_TONES.length]
}

function upstreamRateTone(rate: number): string {
  return UPSTREAM_RATE_TONES[getSupplierUpstreamRateBand(rate)]
}

function supplierTypeLabel(providerID: number): string {
  const providerType = providers.value.find(provider => provider.id === providerID)?.provider_type?.trim()
  if (!providerType) return '未知类型'
  return SUPPLIER_TYPE_LABELS[providerType.toLowerCase()] || providerType
}

function supplierTypeTone(providerID: number): string {
  const providerType = providers.value.find(provider => provider.id === providerID)?.provider_type?.trim().toLowerCase() || ''
  return SUPPLIER_TYPE_TONES[providerType] || SUPPLIER_TYPE_TONE_DEFAULT
}

function upstreamPlatformLabel(group: SupplierProviderGroup): string {
  return group.local_group_platform ? platformLabel(group.local_group_platform) : '待匹配'
}

function groupPlatform(platform?: string): GroupPlatform | undefined {
  if (!platform || !(platform in PLATFORM_LABELS)) return undefined
  return platform as GroupPlatform
}

function upstreamStatusLabel(group: SupplierProviderGroup): string {
  return group.raw_status?.trim() || (group.active ? '有效' : '失效')
}

function upstreamStatusTone(group: SupplierProviderGroup): string {
  const status = upstreamStatusLabel(group).toLowerCase()
  if (group.active && ['active', 'enabled', 'normal', 'success', '有效'].includes(status)) return 'good'
  if (!group.active || ['inactive', 'disabled', 'failed', '失效'].includes(status)) return 'bad'
  return 'info'
}

function suggestedLocalRate(group: SupplierProviderGroup): number {
  const upstreamRate = Number(group.rate_multiplier)
  if (!Number.isFinite(upstreamRate) || upstreamRate <= 0) return 1
  return Math.ceil(upstreamRate * 1.1 * 100) / 100
}

function formatRate(value?: number): string {
  const rate = Number(value)
  if (!Number.isFinite(rate)) return '-'
  return `${rate.toFixed(4).replace(/\.?0+$/, '')}x`
}

function formatProfitRate(value: number | null): string {
  if (value == null || !Number.isFinite(value)) return '-'
  return `${value.toFixed(2)}x`
}

function formatRateDelta(group: SupplierProviderGroup): string {
  const upstreamRate = Number(group.rate_multiplier)
  const localRate = Number(group.local_rate_multiplier)
  if (!Number.isFinite(upstreamRate) || !Number.isFinite(localRate)) return ''
  const delta = localRate - upstreamRate
  return `价差 ${delta > 0 ? '+' : ''}${delta.toFixed(2)}`
}

function platformLabel(platform?: string): string {
  if (!platform) return '未设置平台'
  return PLATFORM_LABELS[platform] || platform
}

function formatTime(value?: string): string {
  if (!value) return '-'
  const date = new Date(value)
  if (Number.isNaN(date.getTime())) return '-'
  return date.toLocaleString('zh-CN')
}

function percentage(value: number, totalValue: number): number {
  if (totalValue <= 0) return 0
  return Math.round((value / totalValue) * 100)
}

function errorMessage(err: unknown, fallback: string): string {
  if (typeof err === 'object' && err && 'message' in err) {
    return String((err as { message?: unknown }).message || fallback)
  }
  return fallback
}
</script>

<style scoped>
.sp-console-shell {
  display: grid;
  gap: 1rem;
}

.sp-control-button,
.sp-dialog-actions .sp-button {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  gap: 0.4rem;
}

.sp-summary-grid {
  display: grid;
  grid-template-columns: repeat(4, minmax(0, 1fr));
  gap: 0.75rem;
}

.sp-summary-grid :deep(.stat-card) {
  position: relative;
  min-height: 6.25rem;
  align-items: center;
  overflow: hidden;
  padding: 1rem;
  border: 1px solid color-mix(in srgb, var(--sp-line) 82%, var(--sp-text));
  border-radius: 0.5rem;
  background: var(--sp-panel);
  box-shadow: 0 4px 14px rgba(15, 23, 42, 0.06);
  transition: border-color 160ms ease, box-shadow 160ms ease, transform 160ms ease;
  animation: sp-summary-enter 180ms ease-out both;
}

.sp-summary-grid :deep(.stat-card)::before {
  position: absolute;
  inset: 0 auto 0 0;
  width: 3px;
  background: var(--sp-cyan);
  content: '';
}

.sp-summary-grid :deep(.stat-card:nth-child(2))::before {
  background: var(--sp-green);
}

.sp-summary-grid :deep(.stat-card:nth-child(3))::before {
  background: var(--sp-amber);
}

.sp-summary-grid :deep(.stat-card:nth-child(4))::before {
  background: var(--sp-red);
}

.sp-summary-grid :deep(.stat-icon) {
  width: 2.75rem;
  height: 2.75rem;
  border: 1px solid currentColor;
  border-radius: 0.5rem;
  opacity: 0.92;
}

.sp-summary-grid :deep(.stat-label) {
  color: var(--sp-muted);
  font-size: 0.78rem;
  font-weight: 600;
  letter-spacing: 0;
}

.sp-summary-grid :deep(.stat-value) {
  color: var(--sp-text);
  font-size: 1.75rem;
  font-weight: 750;
  line-height: 1.1;
}

.sp-summary-grid :deep(.stat-trend) {
  padding: 0.15rem 0.4rem;
  border: 1px solid var(--sp-line);
  border-radius: 0.25rem;
  background: var(--sp-panel-2);
  color: var(--sp-muted);
  font-size: 0.75rem;
  font-weight: 600;
}

.sp-summary-grid :deep(.stat-card:nth-child(2)) { animation-delay: 30ms; }
.sp-summary-grid :deep(.stat-card:nth-child(3)) { animation-delay: 60ms; }
.sp-summary-grid :deep(.stat-card:nth-child(4)) { animation-delay: 90ms; }

.sp-groups-panel {
  min-width: 0;
  margin-bottom: 0;
}

.sp-panel-signals {
  display: flex;
  align-items: center;
  gap: 1rem;
  color: var(--sp-muted);
  font-size: 0.75rem;
}

.sp-panel-signals span {
  display: inline-flex;
  align-items: center;
  gap: 0.375rem;
}

.sp-panel-signals i,
.sp-status i,
.sp-rate-status i {
  width: 0.42rem;
  height: 0.42rem;
  flex: 0 0 auto;
  border-radius: 50%;
  background: currentColor;
}

.sp-panel-signals .good { color: var(--sp-green); }
.sp-panel-signals .warn { color: var(--sp-amber); }

.sp-table-shell {
  display: flex;
  height: min(64vh, 680px);
  min-height: 24rem;
  flex-direction: column;
  overflow: hidden;
}

.sp-table-shell :deep(.table-wrapper) {
  min-height: 0;
  flex: 1;
}

.sp-table-shell :deep(td) {
  vertical-align: middle;
}

.sp-data-pagination {
  flex-shrink: 0;
  border-top: 1px solid var(--sp-soft);
}

.sp-group-name {
  max-width: 14rem;
  overflow: hidden;
  font-weight: 650;
  text-overflow: ellipsis;
}

.sp-upstream-group-chip {
  display: inline-flex;
  max-width: 14rem;
  min-height: 1.75rem;
  align-items: center;
  gap: 0.5rem;
  padding: 0.25rem 0.55rem 0.25rem 0.35rem;
  overflow: hidden;
  border-radius: 0.25rem;
  line-height: 1.2;
  white-space: nowrap;
}

.sp-upstream-group-accent {
  width: 0.2rem;
  height: 1rem;
  flex: 0 0 auto;
  border-radius: 0.125rem;
}

.sp-supplier-chip {
  display: inline-flex;
  max-width: 10rem;
  align-items: center;
  gap: 0.45rem;
  padding: 0.3rem 0.55rem;
  overflow: hidden;
  border-width: 1px;
  border-style: solid;
  border-radius: 0.375rem;
  font-size: 0.8rem;
  font-weight: 700;
  line-height: 1.2;
}

.sp-supplier-name {
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
}

.sp-provider-meta {
  display: flex;
  align-items: center;
  gap: 0.35rem;
  color: var(--sp-dim);
}

.sp-provider-type {
  display: inline-flex;
  align-items: center;
  padding: 0.1rem 0.35rem;
  border-radius: 0.25rem;
  font-size: 0.68rem;
  font-weight: 700;
  line-height: 1.2;
}

.sp-inline-platform {
  flex: 0 0 auto;
  font-size: 0.7rem;
  font-weight: 650;
  opacity: 0.9;
  white-space: nowrap;
}

.sp-inline-platform-muted {
  color: var(--sp-muted);
}

.sp-supplier-dot {
  width: 0.48rem;
  height: 0.48rem;
  flex: 0 0 auto;
  border-radius: 50%;
  box-shadow: 0 0 0 2px color-mix(in srgb, currentColor 14%, transparent);
}

.sp-key {
  max-width: 14rem;
  overflow: hidden;
  font-family: ui-monospace, SFMono-Regular, Consolas, monospace;
  text-overflow: ellipsis;
}

.sp-rate-value {
  display: inline-flex;
  min-width: 3.5rem;
  align-items: center;
  font-variant-numeric: tabular-nums;
  font-weight: 750;
}

.sp-rate-value.local { color: var(--sp-text); }

.sp-status,
.sp-rate-status {
  display: inline-flex;
  align-items: center;
  gap: 0.4rem;
  padding: 0.25rem 0.55rem;
  border: 1px solid currentColor;
  border-radius: 9999px;
  background: var(--sp-panel);
  font-size: 0.75rem;
  font-weight: 650;
  white-space: nowrap;
}

.sp-status.good,
.sp-rate-status.normal { color: var(--sp-green); }
.sp-status.bad,
.sp-rate-status.inverted { color: var(--sp-red); }
.sp-status.info { color: var(--sp-blue); }
.sp-rate-status.low,
.sp-rate-status.equal { color: var(--sp-amber); }
.sp-rate-status.unmatched,
.sp-rate-status.inactive,
.sp-rate-status.invalid { color: var(--sp-muted); }

.sp-local-group .sp-sub {
  display: flex;
  align-items: center;
  gap: 0.5rem;
}

.sp-local-group-badge {
  max-width: 12rem;
}

.sp-detail-group-badge {
  width: fit-content;
  max-width: 100%;
  margin-top: 0.375rem;
}

.sp-inline-empty {
  display: inline-flex;
  align-items: center;
  gap: 0.35rem;
  padding: 0;
  border: 0;
  background: transparent;
  color: var(--sp-muted);
  cursor: pointer;
}

.sp-inline-empty:hover { color: var(--sp-cyan); }
.sp-empty-value { color: var(--sp-dim); }

.sp-profit-cell {
  display: grid;
  gap: 0.15rem;
  font-variant-numeric: tabular-nums;
}

.sp-profit-cell strong {
  color: var(--sp-text);
  font-size: 0.9rem;
}

.sp-profit-cell span {
  color: var(--sp-muted);
  font-size: 0.7rem;
}

.sp-profit-cell.normal strong { color: var(--sp-green); }
.sp-profit-cell.inverted strong { color: var(--sp-red); }
.sp-profit-cell.low strong,
.sp-profit-cell.equal strong { color: var(--sp-amber); }

.sp-account-count {
  display: inline-flex;
  min-width: 3.2rem;
  align-items: center;
  justify-content: center;
  gap: 0.35rem;
  padding: 0.3rem 0.5rem;
  border: 1px solid var(--sp-line);
  border-radius: 0.375rem;
  background: var(--sp-panel-2);
  color: var(--sp-blue);
  cursor: pointer;
}

.sp-account-count:hover {
  border-color: var(--sp-cyan);
  color: var(--sp-cyan);
}

.sp-row-actions {
  display: flex;
  align-items: center;
  gap: 0.4rem;
}

.sp-row-action {
  display: inline-flex;
  min-height: 2rem;
  align-items: center;
  justify-content: center;
  gap: 0.3rem;
  padding: 0.35rem 0.55rem;
  border: 1px solid var(--sp-line);
  border-radius: 0.375rem;
  background: var(--sp-panel);
  color: var(--sp-muted);
  font-size: 0.75rem;
  font-weight: 650;
  cursor: pointer;
  transition: border-color 140ms ease, background 140ms ease, color 140ms ease;
}

.sp-row-action:hover {
  border-color: var(--sp-cyan);
  color: var(--sp-cyan);
}

.sp-row-action.primary {
  border-color: color-mix(in srgb, var(--sp-cyan) 40%, var(--sp-line));
  background: color-mix(in srgb, var(--sp-cyan) 7%, var(--sp-panel));
  color: var(--sp-cyan);
}

.sp-row-action.danger:hover {
  border-color: var(--sp-red);
  color: var(--sp-red);
}

.sp-drawer-summary {
  display: grid;
  grid-template-columns: 1fr auto;
  align-items: end;
  gap: 0.3rem 1rem;
  padding: 0 0 1rem;
  border-bottom: 1px solid var(--sp-soft);
}

.sp-drawer-summary .sp-rate-status { width: fit-content; }
.sp-drawer-summary strong { font-size: 2rem; line-height: 1; }
.sp-drawer-summary small { grid-column: 2; color: var(--sp-muted); text-align: right; }
.sp-detail-wide { grid-column: 1 / -1; }

.sp-dialog-context {
  display: grid;
  gap: 0.25rem;
  margin-bottom: 1rem;
  padding: 0 0 1rem;
  border-bottom: 1px solid var(--sp-soft);
}

.sp-dialog-context span,
.sp-dialog-context small {
  color: var(--sp-muted);
  font-size: 0.78rem;
}

.sp-dialog-context strong { color: var(--sp-text); font-size: 1rem; }

.sp-field {
  display: grid;
  gap: 0.375rem;
  color: var(--sp-text);
  font-size: 0.875rem;
  font-weight: 500;
}

.sp-dialog-form {
  display: grid;
  gap: 1rem;
}

.sp-match-preview {
  display: grid;
  grid-template-columns: repeat(3, minmax(0, 1fr));
  gap: 0.75rem;
  margin-top: 1rem;
  padding-top: 1rem;
  border-top: 1px solid var(--sp-soft);
}

.sp-match-preview div {
  display: grid;
  gap: 0.25rem;
}

.sp-match-preview span { color: var(--sp-muted); font-size: 0.75rem; }
.sp-match-preview strong { color: var(--sp-text); font-size: 0.9rem; }

.sp-rate-recommendation {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 0.75rem;
  margin-top: 1rem;
  padding: 0.75rem 0;
  border-top: 1px solid var(--sp-soft);
  border-bottom: 1px solid var(--sp-soft);
  color: var(--sp-muted);
  font-size: 0.8rem;
}

.sp-rate-recommendation strong { color: var(--sp-green); }

.sp-dialog-actions {
  display: flex;
  justify-content: flex-end;
  gap: 0.75rem;
}

.sp-button.secondary {
  border-color: var(--sp-line);
  background: var(--sp-panel);
  color: var(--sp-text);
}

.sp-spin { animation: sp-spin 800ms linear infinite; }

@media (hover: hover) {
  .sp-summary-grid :deep(.stat-card:hover) {
    border-color: color-mix(in srgb, var(--sp-cyan) 34%, var(--sp-line));
    box-shadow: 0 10px 24px rgba(15, 23, 42, 0.1);
    transform: translateY(-2px);
  }
}

@media (max-width: 1050px) {
  .sp-summary-grid { grid-template-columns: repeat(2, minmax(0, 1fr)); }
}

@media (max-width: 760px) {
  .sp-summary-grid { grid-template-columns: 1fr; }
  .sp-table-shell { height: auto; min-height: 0; overflow: visible; }
  .sp-panel-signals { width: 100%; justify-content: space-between; }
  .sp-row-actions { flex-wrap: wrap; justify-content: flex-end; }
  .sp-match-preview { grid-template-columns: 1fr; }
}

@media (prefers-reduced-motion: reduce) {
  .sp-summary-grid :deep(.stat-card) { transition: none; animation: none; }
  .sp-spin { animation: none; }
}

@keyframes sp-summary-enter {
  from { opacity: 0; transform: translateY(5px); }
  to { opacity: 1; transform: translateY(0); }
}

@keyframes sp-spin {
  to { transform: rotate(360deg); }
}
</style>

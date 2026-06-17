<template>
  <AppLayout>
    <TablePageLayout class="upstream-accounts-page">
      <template #filters>
        <div class="accounts-shell">
          <section class="accounts-topbar">
            <div class="stats-strip">
              <article
                v-for="card in statCards"
                :key="card.key"
                :class="['stat-card', `stat-card-${card.tone}`]"
              >
                <span v-if="card.key === 'update' && summary.update_count > 0" class="stat-alert-dot"></span>
                <span class="stat-icon">
                  <Icon :name="card.icon" size="md" :stroke-width="2" />
                </span>
                <span class="stat-copy">
                  <strong>{{ card.value }}</strong>
                  <span>{{ card.label }}</span>
                </span>
              </article>
            </div>
            <div class="accounts-actions">
              <div class="provider-summary">
                <span>{{ t('admin.upstreamAccounts.syncProviders') }}</span>
                <strong>{{ syncProviderLabel }}</strong>
                <code v-if="syncProviderCode">{{ syncProviderCode }}</code>
              </div>
              <button
                type="button"
                class="ui-button ui-button-icon"
                :disabled="loading || syncing"
                :title="t('common.refresh')"
                @click="reload"
              >
                <Icon name="refresh" size="md" :stroke-width="2" :class="loading ? 'animate-spin' : ''" />
              </button>
              <button
                type="button"
                class="ui-button ui-button-primary"
                :disabled="loading || syncing || !canSync"
                @click="runSync"
              >
                <Icon name="sync" size="sm" :stroke-width="2" :class="syncing ? 'animate-spin' : ''" />
                {{ t('admin.upstreamAccounts.syncNow') }}
              </button>
            </div>
          </section>

          <section class="rate-guard-panel">
            <div class="guard-left">
              <label class="guard-switch" :class="{ 'is-on': rateGuardForm.enabled }">
                <input v-model="rateGuardForm.enabled" type="checkbox" />
                <span></span>
              </label>
              <div class="guard-copy">
                <div class="guard-title">{{ t('admin.upstreamAccounts.rateGuardTitle') }}</div>
                <div class="guard-description">{{ t('admin.upstreamAccounts.rateGuardDescription') }}</div>
                <div class="guard-status-line">
                  <span :class="['status-pill', rateGuardForm.enabled ? 'status-pill-on' : 'status-pill-muted']">
                    {{ rateGuardForm.enabled ? t('admin.upstreamAccounts.rateGuardEnabled') : t('admin.upstreamAccounts.rateGuardDisabled') }}
                  </span>
                  <span>
                    {{ t('admin.upstreamAccounts.rateGuardLastRun') }}:
                    {{ rateGuardLastRunText }}
                  </span>
                  <span
                    v-if="rateGuardConfig?.last_run_status"
                    :class="['record-status', rateGuardConfig.last_run_status === 'failed' ? 'record-status-error' : 'record-status-success']"
                  >
                    {{ rateGuardConfig.last_run_status === 'failed' ? t('admin.upstreamAccounts.rateGuardStatusFailed') : t('admin.upstreamAccounts.rateGuardStatusSuccess') }}
                  </span>
                  <span v-if="rateGuardConfig?.last_run_message" class="status-error-message">
                    {{ rateGuardConfig.last_run_message }}
                  </span>
                </div>
              </div>
            </div>
            <div class="guard-controls">
              <span class="control-label">{{ t('admin.upstreamAccounts.rateGuardAutoRun') }} {{ t('admin.upstreamAccounts.rateGuardIntervalSeconds') }}</span>
              <input
                v-model.number="rateGuardForm.interval_seconds"
                type="number"
                min="1"
                class="ui-input interval-input"
              />
              <span class="guard-hint">{{ rateGuardDailyRunsText }}</span>
              <button
                type="button"
                class="ui-button"
                :disabled="loadingRateGuardConfig || savingRateGuardConfig"
                @click="saveRateGuardConfig"
              >
                {{ t('common.save') }}
              </button>
              <button
                type="button"
                class="ui-button ui-button-primary"
                :disabled="loadingRateGuardConfig || savingRateGuardConfig || runningRateGuardNow"
                @click="runRateGuardNow"
              >
                <Icon name="play" size="sm" :stroke-width="2" :class="runningRateGuardNow ? 'animate-pulse' : ''" />
                {{ t('admin.upstreamAccounts.rateGuardRunNow') }}
              </button>
            </div>
          </section>

          <section class="filter-row">
            <Select
              v-model="providerFilter"
              class="filter-select"
              :options="providerOptions"
            />
            <Select
              v-model="sourceFilter"
              class="filter-select"
              :options="sourceOptions"
            />
            <div class="search-wrap">
              <Icon name="search" size="sm" :stroke-width="2" />
              <input
                v-model.trim="searchQuery"
                type="search"
                class="ui-input filter-search"
                :placeholder="t('admin.upstreamAccounts.searchPlaceholder')"
              />
            </div>
            <div class="filtered-count">
              <span>{{ t('admin.upstreamAccounts.filteredCount') }}</span>
              <strong>{{ filteredItems.length }}</strong>
            </div>
          </section>

          <nav class="quick-tags" aria-label="quick filters">
            <button
              v-for="(tag, index) in quickFilterTags"
              :key="tag"
              type="button"
              :class="['quick-tag', { active: index === 0 }]"
            >
              {{ tag }}
            </button>
          </nav>
        </div>
      </template>

      <template #table>
        <div class="accounts-table-content">
          <div v-if="warnings.length" class="warning-banner">
            <Icon name="exclamationTriangle" size="sm" :stroke-width="2" />
            <div>
              <div v-for="warning in warnings" :key="warning">{{ warning }}</div>
            </div>
          </div>

          <section class="accounts-table-card">
            <DataTable
              :columns="columns"
              :data="filteredItems"
              :loading="loading"
              :row-class="accountRowClass"
              :estimate-row-height="92"
            >
              <template #cell-source="{ row }">
                <div class="source-cell">
                  <span :class="['source-line', sourceToneClass(row)]"></span>
                  <div class="source-main">
                    <div class="source-title">
                      <Icon v-if="row.rate_violation" name="exclamationTriangle" size="sm" :stroke-width="2" class="source-warning-icon" />
                      <span :class="['table-tag', providerToneClass(row.provider_slug, 'tag')]">
                        {{ row.provider_name || row.provider_slug }}
                      </span>
                    </div>
                    <code class="source-id">{{ row.provider_slug || '-' }}</code>
                  </div>
                  <a
                    v-if="row.provider_base_url"
                    :href="row.provider_base_url"
                    target="_blank"
                    rel="noopener noreferrer"
                    class="table-tag home-tag"
                    :title="t('admin.upstreamProviders.openHomepage')"
                  >
                    <Icon name="home" size="xs" :stroke-width="2" />
                    {{ t('admin.upstreamProviders.homepageShort') }}
                  </a>
                </div>
              </template>

              <template #cell-upstream_key_name="{ row }">
                <div class="two-line-cell">
                  <span :class="['main-text', matchedAccountPlatformTextClass(row)]">{{ row.upstream_key_name }}</span>
                  <span class="sub-text">{{ row.upstream_group_name || '-' }}</span>
                </div>
              </template>

              <template #cell-local_account_name="{ row }">
                <div class="two-line-cell">
                  <span :class="['main-text', matchedAccountPlatformTextClass(row)]">{{ row.local_account_name || row.matched_account_name || '-' }}</span>
                  <span v-if="row.matched_account_id" :class="['table-tag', 'tag-account', 'account-id-tag', matchedAccountPlatformTagClass(row)]">
                    #{{ row.matched_account_id }} {{ row.matched_account_name || row.local_account_name }}
                  </span>
                  <div v-else-if="row.conflict_accounts?.length" class="tag-list">
                    <span
                      v-for="account in row.conflict_accounts"
                      :key="`${row.provider_slug}-${row.upstream_key_name}-conflict-${account.id}`"
                      class="group-chip group-chip-warning"
                      :title="conflictAccountTitle(account)"
                    >
                      #{{ account.id }} {{ account.name }}
                      <span v-if="account.bound_groups?.length" class="font-mono">
                        {{ conflictAccountRates(account) }}
                      </span>
                    </span>
                  </div>
                  <span v-else-if="row.conflict_account_ids?.length" class="sub-text sub-text-warning">
                    {{ t('admin.upstreamAccounts.conflictIds', { ids: row.conflict_account_ids.join(', ') }) }}
                  </span>
                  <span v-else class="table-tag tag-account account-id-tag">-</span>
                </div>
              </template>

              <template #cell-upstream_rate_multiplier="{ value }">
                <div class="rate-cell">
                  <span :class="['rate-value', rateToneClass(value)]">{{ formatRate(value) }}</span>
                  <span :class="['rate-bar', rateToneClass(value)]">
                    <span :style="{ width: rateProgressWidth(value) }"></span>
                  </span>
                </div>
              </template>

              <template #cell-local_group_name="{ row }">
                <div v-if="row.bound_groups?.length" class="tag-list group-list">
                  <span
                    v-for="group in row.bound_groups"
                    :key="`${row.provider_slug}-${row.upstream_key_name}-${group.id}`"
                    :class="['group-chip', matchedAccountPlatformTagClass(row)]"
                    :title="`${group.name} ${formatRate(group.rate_multiplier)}`"
                  >
                    {{ group.name }}
                    <span class="font-mono">{{ formatRate(group.rate_multiplier) }}</span>
                  </span>
                </div>
                <div v-else class="two-line-cell">
                  <span class="dash">{{ row.local_group_name || '-' }}</span>
                  <span v-if="row.local_rate_multiplier !== undefined" class="sub-text">
                    {{ formatRate(row.local_rate_multiplier) }}
                  </span>
                </div>
              </template>

              <template #cell-balance="{ row }">
                <div class="balance-cell">
                  <span v-if="getProviderBalance(row.provider_slug) !== null" class="balance-value">
                    ${{ formatMoney(getProviderBalance(row.provider_slug) || 0) }}
                  </span>
                  <span v-else class="dash">-</span>
                  <button
                    v-if="getProviderBalance(row.provider_slug) !== null"
                    type="button"
                    class="trend-btn"
                    title="查看余额趋势"
                    @click="openTrendModal(row.provider_slug, row.provider_name)"
                  >
                    <Icon name="chart" size="xs" :stroke-width="2" />
                  </button>
                </div>
              </template>

              <template #cell-today_consumption="{ row }">
                <div class="balance-cell">
                  <span v-if="getProviderConsumption(row.provider_slug) !== null" class="consumption-value">
                    ${{ formatMoney(getProviderConsumption(row.provider_slug) || 0) }}
                  </span>
                  <span v-else class="dash">-</span>
                  <button
                    v-if="getProviderConsumption(row.provider_slug) !== null"
                    type="button"
                    class="trend-btn"
                    title="查看消费趋势"
                    @click="openTrendModal(row.provider_slug, row.provider_name)"
                  >
                    <Icon name="trendingUp" size="xs" :stroke-width="2" />
                  </button>
                </div>
              </template>

              <template #cell-status="{ row }">
                <div v-if="getMatchedAccount(row)" class="status-cell">
                  <AccountStatusIndicator
                    :account="getMatchedAccount(row)!"
                    @show-temp-unsched="handleShowTempUnsched"
                  />
                </div>
                <span v-else class="dash">-</span>
              </template>

              <template #cell-schedulable="{ row }">
                <div v-if="getMatchedAccount(row)" class="status-cell">
                  <button
                    type="button"
                    class="schedulable-toggle"
                    :disabled="togglingSchedulableId === getMatchedAccount(row)!.id"
                    :class="[getMatchedAccount(row)!.schedulable ? 'schedulable-on' : 'schedulable-off']"
                    :title="getMatchedAccount(row)!.schedulable ? t('admin.accounts.schedulableEnabled') : t('admin.accounts.schedulableDisabled')"
                    @click="handleToggleSchedulable(getMatchedAccount(row)!)"
                  >
                    <span class="schedulable-track">
                      <span
                        class="schedulable-thumb"
                        :class="[getMatchedAccount(row)!.schedulable ? 'schedulable-thumb-on' : 'schedulable-thumb-off']"
                      />
                    </span>
                  </button>
                </div>
                <span v-else class="dash">-</span>
              </template>

              <template #cell-test_status="{ row }">
                <div class="test-status-cell">
                  <span
                    v-if="accountTestStatusById[row.matched_account_id || 0]"
                    :class="[
                      'test-status-pill',
                      `test-status-${accountTestStatusById[row.matched_account_id || 0]}`
                    ]"
                  >
                    <Icon
                      :name="accountTestStatusById[row.matched_account_id || 0] === 'success'
                        ? 'checkCircle'
                        : accountTestStatusById[row.matched_account_id || 0] === 'failed'
                          ? 'xCircle'
                          : 'clock'"
                      size="sm"
                      :stroke-width="2"
                    />
                    {{ accountTestStatusLabel(accountTestStatusById[row.matched_account_id || 0]) }}
                  </span>
                  <span v-else class="dash">-</span>
                </div>
              </template>

              <template #cell-actions="{ row }">
                <div v-if="row.matched_account_id" class="action-cell">
                  <button
                    type="button"
                    :class="['text-action', row.rate_violation ? 'text-action-danger' : '']"
                    :disabled="savingAccountGroupId === row.matched_account_id || testingAccountId === row.matched_account_id"
                    @click="openAccountGroupDialog(row)"
                  >
                    <Icon :name="row.rate_violation ? 'exclamationTriangle' : 'edit'" size="sm" :stroke-width="2" :class="savingAccountGroupId === row.matched_account_id ? 'animate-spin' : ''" />
                    {{ row.rate_violation ? '\u5904\u7406\u98ce\u9669' : t('admin.upstreamAccounts.editBoundGroups') }}
                  </button>
                  <button
                    type="button"
                    class="text-action text-action-muted"
                    :disabled="testingAccountId === row.matched_account_id || savingAccountGroupId === row.matched_account_id"
                    @click="openAccountTestDialog(row)"
                  >
                    <Icon name="play" size="sm" :stroke-width="2" :class="testingAccountId === row.matched_account_id ? 'animate-spin' : ''" />
                    {{ t('admin.upstreamAccounts.testConnection') }}
                  </button>
                </div>
                <span v-else class="dash action-dash">-</span>
              </template>

              <template #empty>
                <EmptyState
                  :title="emptyTitle"
                  :description="emptyDescription"
                  :action-text="t('common.refresh')"
                  @action="reload"
                />
              </template>
            </DataTable>
          </section>

          <section class="records-panel">
            <div class="records-header">
              <h3>{{ t('admin.upstreamAccounts.syncLogs') }}</h3>
              <span>{{ t('admin.upstreamAccounts.latestRecords', { count: syncLogEntries.length }) }} {{ syncLogEntries.length }}</span>
            </div>
            <div class="records-info">{{ t('admin.upstreamAccounts.syncLogsDescription') }}</div>
            <div v-if="syncLogEntries.length" class="records-table-wrap">
              <table class="records-table">
                <thead>
                  <tr>
                    <th>{{ t('admin.upstreamAccounts.logTime') }}</th>
                    <th>{{ t('admin.upstreamAccounts.logTriggerSource') }}</th>
                    <th>{{ t('admin.upstreamAccounts.logAccount') }}</th>
                    <th>{{ t('admin.upstreamAccounts.logUpstream') }}</th>
                    <th>{{ t('admin.upstreamAccounts.logRateCompare') }}</th>
                    <th>{{ t('admin.upstreamAccounts.logUnboundGroups') }}</th>
                    <th>{{ t('admin.upstreamAccounts.logRemainingGroups') }}</th>
                  </tr>
                </thead>
                <tbody>
                  <tr v-for="entry in syncLogEntries" :key="entry.key" class="records-row">
                    <td>{{ formatDateTime(entry.created_at) }}</td>
                    <td>
                      <span :class="['trigger-chip', triggerClass(entry.trigger_source)]">
                        {{ upstreamAccountSyncTriggerSourceLabel(entry.trigger_source) }}
                      </span>
                    </td>
                    <td>
                      <div class="two-line-cell">
                        <span class="main-text">{{ entry.matched_local_account_name }}</span>
                        <code class="table-tag tag-account">#{{ entry.matched_local_account_id }}</code>
                      </div>
                    </td>
                    <td>
                      <div class="two-line-cell">
                        <span class="main-text">{{ entry.upstream_key_name }}</span>
                        <div class="tag-list">
                          <span :class="['table-tag', providerToneClass(entry.provider_slug, 'tag')]">{{ entry.provider_name || entry.provider_slug }}</span>
                          <span class="table-tag tag-gray">{{ entry.upstream_group_name }}</span>
                        </div>
                      </div>
                    </td>
                    <td>
                      <div class="rate-compare">
                        <span class="rate-compare-upstream">{{ formatRate(entry.upstream_rate_multiplier) }}</span>
                        <span>/</span>
                        <span class="rate-compare-local">{{ formatRate(entry.local_min_rate_multiplier) }}</span>
                      </div>
                    </td>
                    <td>
                      <div class="tag-list">
                        <span v-for="group in entry.unbound_group_names" :key="`${entry.key}-${group}`" class="log-chip log-chip-warning">{{ group }}</span>
                      </div>
                    </td>
                    <td>
                      <div class="tag-list">
                        <span v-if="!entry.remaining_group_ids.length" class="dash">-</span>
                        <code v-for="groupID in entry.remaining_group_ids" :key="`${entry.key}-${groupID}`" class="log-chip">#{{ groupID }}</code>
                      </div>
                    </td>
                  </tr>
                </tbody>
              </table>
            </div>
            <div v-else class="records-empty">
              <Icon name="document" size="xl" :stroke-width="2" />
              <span>{{ t('admin.upstreamAccounts.noSyncLogs') }}</span>
              <button type="button" class="ui-button" :disabled="loading || syncing || !canSync" @click="runSync">
                {{ t('admin.upstreamAccounts.syncNow') }}
              </button>
            </div>
          </section>
        </div>

        <div v-if="accountGroupDialogItem" class="account-group-dialog fixed inset-0 z-50 flex items-center justify-center bg-black/40 px-4 py-6" @click.self="closeAccountGroupDialog">
          <div class="w-full max-w-lg overflow-hidden rounded-lg bg-white shadow-xl dark:bg-dark-800">
            <div class="border-b border-gray-100 px-5 py-4 dark:border-dark-700">
              <h3 class="text-lg font-semibold text-gray-950 dark:text-white">{{ t('admin.upstreamAccounts.editBoundGroupsTitle') }}</h3>
              <p class="mt-1 text-sm text-gray-500 dark:text-gray-400">{{ t('admin.upstreamAccounts.editBoundGroupsDescription') }}</p>
            </div>
            <div class="space-y-4 px-5 py-4">
              <div>
                <div class="text-xs font-medium text-gray-500 dark:text-gray-400">{{ t('admin.upstreamAccounts.columns.localAccount') }}</div>
                <div class="mt-1 text-sm font-semibold text-gray-950 dark:text-white">{{ accountGroupDialogItem.matched_account_name || accountGroupDialogItem.local_account_name }}</div>
              </div>
              <GroupSelector
                v-model="accountGroupIds"
                :groups="accountGroupOptions"
                :platform="accountGroupPlatform"
                searchable
              />
              <div v-if="accountGroupIds.length" class="tag-list">
                <span v-for="groupID in accountGroupIds" :key="groupID" :class="['group-chip', accountGroupPlatformTagClass]">
                  {{ groupNameById(groupID) }}
                </span>
              </div>
            </div>
            <div class="flex justify-between gap-2 border-t border-gray-100 px-5 py-4 dark:border-dark-700">
              <button type="button" class="btn btn-danger btn-sm" :disabled="savingAccountGroupId === accountGroupDialogItem.matched_account_id" @click="clearAccountGroups">
                {{ t('admin.upstreamAccounts.clearBoundGroups') }}
              </button>
              <div class="flex gap-2">
                <button type="button" class="btn btn-secondary btn-sm" :disabled="savingAccountGroupId === accountGroupDialogItem.matched_account_id" @click="closeAccountGroupDialog">
                  {{ t('common.cancel') }}
                </button>
                <button type="button" class="btn btn-primary btn-sm" :disabled="savingAccountGroupId === accountGroupDialogItem.matched_account_id" @click="saveAccountGroups">
                  <Icon name="cog" size="sm" class="mr-1" :class="savingAccountGroupId === accountGroupDialogItem.matched_account_id ? 'animate-spin' : ''" />
                  {{ t('common.save') }}
                </button>
              </div>
            </div>
          </div>
        </div>

        <AccountTestModal
          :show="showTestModal"
          :account="testingAccount"
          @close="closeTestModal"
          @test-result="handleAccountTestResult"
        />
        <TempUnschedStatusModal
          :show="showTempUnsched"
          :account="tempUnschedAccount"
          @close="closeTempUnschedModal"
          @reset="handleTempUnschedReset"
        />
        <UpstreamProviderTrendModal
          :show="showTrendModal"
          :provider-slug="trendProviderSlug"
          :provider-name="trendProviderName"
          :rows="balanceOverview?.rows || []"
          @close="closeTrendModal"
        />
      </template>
    </TablePageLayout>
  </AppLayout>
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { adminAPI } from '@/api/admin'
import type {
  UpstreamAccountRateGuardConfig,
  UpstreamAccountSyncConflictAccount,
  UpstreamAccountSyncBoundGroup,
  UpstreamAccountSyncItem,
  UpstreamAccountSyncRecord,
  UpstreamAccountSyncResult,
  UpstreamAccountSyncUnbindDetail,
  UpstreamBalanceConsumptionOverview,
} from '@/api/admin/upstreamAccountSync'
import type { Account, AdminGroup, GroupPlatform } from '@/types'
import { useAppStore } from '@/stores/app'
import { extractApiErrorMessage } from '@/utils/apiError'
import { formatDateTime } from '@/utils/format'
import type { Column } from '@/components/common/types'
import Select, { type SelectOption } from '@/components/common/Select.vue'
import AppLayout from '@/components/layout/AppLayout.vue'
import TablePageLayout from '@/components/layout/TablePageLayout.vue'
import DataTable from '@/components/common/DataTable.vue'
import EmptyState from '@/components/common/EmptyState.vue'
import Icon from '@/components/icons/Icon.vue'
import GroupSelector from '@/components/common/GroupSelector.vue'
import AccountTestModal from '@/components/admin/account/AccountTestModal.vue'
import AccountStatusIndicator from '@/components/account/AccountStatusIndicator.vue'
import TempUnschedStatusModal from '@/components/account/TempUnschedStatusModal.vue'
import UpstreamProviderTrendModal from '@/components/admin/upstream/UpstreamProviderTrendModal.vue'

const { t } = useI18n()
const appStore = useAppStore()

type AccountTestStatus = 'testing' | 'success' | 'failed'

const result = ref<UpstreamAccountSyncResult | null>(null)
const loading = ref(false)
const syncing = ref(false)
const loadingRateGuardConfig = ref(false)
const savingRateGuardConfig = ref(false)
const runningRateGuardNow = ref(false)
const savingAccountGroupId = ref<number | null>(null)
const testingAccountId = ref<number | null>(null)
const togglingSchedulableId = ref<number | null>(null)
const showTestModal = ref(false)
const showTempUnsched = ref(false)
const testingAccount = ref<Account | null>(null)
const tempUnschedAccount = ref<Account | null>(null)
const accountTestStatusById = ref<Record<number, AccountTestStatus>>({})
const matchedAccountsById = ref<Record<number, Account>>({})
const localGroups = ref<AdminGroup[]>([])
const loadError = ref('')
const searchQuery = ref('')
const providerFilter = ref('')
const sourceFilter = ref('')
const rateGuardConfig = ref<UpstreamAccountRateGuardConfig | null>(null)
const rateGuardForm = ref({
  enabled: false,
  interval_seconds: 3600
})
const accountGroupDialogItem = ref<UpstreamAccountSyncItem | null>(null)
const accountGroupIds = ref<number[]>([])
const accountGroupPlatform = ref<GroupPlatform | undefined>()
const balanceOverview = ref<UpstreamBalanceConsumptionOverview | null>(null)
const showTrendModal = ref(false)
const trendProviderSlug = ref('')
const trendProviderName = ref('')

type UpstreamAccountSyncLogEntry = UpstreamAccountSyncUnbindDetail & {
  created_at: string
  key: string
}

const columns = computed<Column[]>(() => [
  { key: 'source', label: t('admin.upstreamAccounts.columns.source'), class: 'upstream-center-column' },
  { key: 'upstream_key_name', label: t('admin.upstreamAccounts.columns.upstreamKey'), class: 'upstream-center-column' },
  { key: 'local_account_name', label: t('admin.upstreamAccounts.columns.localAccount'), class: 'upstream-center-column' },
  { key: 'upstream_rate_multiplier', label: t('admin.upstreamAccounts.columns.upstreamRate'), sortable: true, class: 'upstream-center-column upstream-rate-column' },
  { key: 'local_group_name', label: t('admin.upstreamAccounts.columns.boundGroups'), class: 'upstream-center-column' },
  { key: 'balance', label: '余额', class: 'upstream-center-column' },
  { key: 'today_consumption', label: '今日消费', class: 'upstream-center-column' },
  { key: 'status', label: t('admin.accounts.columns.status'), class: 'upstream-center-column' },
  { key: 'schedulable', label: t('admin.accounts.columns.schedulable'), class: 'upstream-center-column' },
  { key: 'test_status', label: t('admin.upstreamAccounts.columns.testStatus'), class: 'upstream-center-column' },
  { key: 'actions', label: t('common.actions'), class: 'upstream-center-column' }
])

const emptySummary = {
  upstream_key_count: 0,
  matched_account_count: 0,
  create_count: 0,
  update_count: 0,
  skip_count: 0,
  conflict_count: 0,
  rate_violation_count: 0,
  unbound_group_count: 0
}

const summary = computed(() => result.value?.summary || emptySummary)
const syncProviders = computed(() => result.value?.providers || [])
const items = computed<UpstreamAccountSyncItem[]>(() => result.value?.items || [])
const warnings = computed(() => result.value?.warnings || [])
const records = computed<UpstreamAccountSyncRecord[]>(() => result.value?.records || [])
const statCards = computed(() => [
  {
    key: 'total',
    label: t('admin.upstreamAccounts.upstreamKeys'),
    value: summary.value.upstream_key_count,
    icon: 'database' as const,
    tone: 'emerald'
  },
  {
    key: 'create',
    label: t('admin.upstreamAccounts.toCreate'),
    value: summary.value.create_count,
    icon: 'plus' as const,
    tone: 'gray'
  },
  {
    key: 'update',
    label: t('admin.upstreamAccounts.toUpdate'),
    value: summary.value.update_count,
    icon: 'refresh' as const,
    tone: 'orange'
  },
  {
    key: 'risk',
    label: t('admin.upstreamAccounts.rateRisks'),
    value: summary.value.rate_violation_count,
    icon: 'exclamationTriangle' as const,
    tone: 'red'
  }
])
const quickFilterTags = computed(() => [
  '\u5168\u90e8',
  'Happiness',
  'NikoAPI',
  t('admin.upstreamAccounts.toUpdate'),
  '\u65e0\u7ed1\u5b9a\u5206\u7ec4',
  '\u500d\u7387\u5f02\u5e38'
])
const syncLogEntries = computed<UpstreamAccountSyncLogEntry[]>(() => {
  const entries: UpstreamAccountSyncLogEntry[] = []
  for (const record of records.value) {
    for (const detail of record.unbind_details || []) {
      const unboundGroupIDs = numberArray(detail.unbound_group_ids)
      entries.push({
        ...detail,
        unbound_group_ids: unboundGroupIDs,
        unbound_group_names: stringArray(detail.unbound_group_names),
        remaining_group_ids: numberArray(detail.remaining_group_ids),
        created_at: record.created_at,
        key: `${record.created_at}-${detail.matched_local_account_id}-${detail.upstream_key_name}-${unboundGroupIDs.join('_')}`
      })
    }
  }
  return entries
})
const canSync = computed(() => summary.value.create_count > 0 || summary.value.update_count > 0 || summary.value.rate_violation_count > 0)
const syncProviderLabel = computed(() => {
  if (syncProviders.value.length === 1) return syncProviders.value[0].name || syncProviders.value[0].slug
  if (syncProviders.value.length > 1) return t('admin.upstreamAccounts.multipleProviders', { count: syncProviders.value.length })
  return '-'
})
const syncProviderCode = computed(() => {
  if (syncProviders.value.length === 1) return syncProviders.value[0].slug
  return ''
})
const rateGuardLastRunText = computed(() => {
  if (!rateGuardConfig.value?.last_run_at) {
    return t('admin.upstreamAccounts.rateGuardNeverRun')
  }
  return formatDateTime(rateGuardConfig.value.last_run_at)
})
const rateGuardDailyRunsText = computed(() => {
  const seconds = Number(rateGuardForm.value.interval_seconds)
  if (!Number.isFinite(seconds) || seconds <= 0) return '\u7ea6\u6bcf\u65e5\u6267\u884c - \u6b21'
  return `\u7ea6\u6bcf\u65e5\u6267\u884c ${Math.floor(86400 / seconds)} \u6b21`
})
const providerOptions = computed<SelectOption[]>(() => [
  { value: '', label: t('admin.upstreamAccounts.allProviders') },
  ...Array.from(new Map(items.value.map(item => [
    item.provider_slug,
    {
      value: item.provider_slug,
      label: item.provider_name ? `${item.provider_name} (${item.provider_slug})` : item.provider_slug
    }
  ])).values())
])
const sourceOptions = computed<SelectOption[]>(() => [
  { value: '', label: t('admin.upstreamAccounts.allSources') },
  { value: 'synced', label: t('admin.upstreamAccounts.sourceSynced') },
  { value: 'unsynced', label: t('admin.upstreamAccounts.sourceUnsynced') }
])
const filteredItems = computed(() => {
  const keyword = searchQuery.value.trim().toLowerCase()
  return items.value.filter((item) => {
    if (providerFilter.value && item.provider_slug !== providerFilter.value) return false
    if (sourceFilter.value === 'synced' && !item.matched_account_id) return false
    if (sourceFilter.value === 'unsynced' && item.matched_account_id) return false
    if (!keyword) return true
    const haystack = [
      item.provider_name,
      item.provider_slug,
      item.upstream_key_name,
      item.upstream_group_name,
      item.local_account_name,
      item.matched_account_name,
      item.local_group_name
    ].filter(Boolean).join(' ').toLowerCase()
    return haystack.includes(keyword)
  })
})
const accountGroupOptions = computed(() => {
  if (!accountGroupDialogItem.value) return []
  const platform = accountGroupPlatform.value
  return localGroups.value.filter(group => (!platform || group.platform === platform) && group.status === 'active')
})
const accountGroupPlatformTagClass = computed(() => platformTagClass(accountGroupPlatform.value))

const emptyTitle = computed(() => {
  return loadError.value ? t('admin.upstreamAccounts.emptyNoDefaultTitle') : t('admin.upstreamAccounts.emptyTitle')
})

const emptyDescription = computed(() => {
  return loadError.value || t('admin.upstreamAccounts.emptyDescription')
})

async function reload() {
  loading.value = true
  loadingRateGuardConfig.value = true
  loadError.value = ''
  try {
    const [preview, config, balance] = await Promise.all([
      adminAPI.upstreamAccountSync.getPreview(),
      adminAPI.upstreamAccountSync.getRateGuardConfig(),
      adminAPI.upstreamAccountSync.getBalanceConsumption(30)
    ])
    result.value = preview
    await syncMatchedAccounts(preview.items || [])
    applyRateGuardConfig(config)
    balanceOverview.value = balance
    void loadLocalGroups()
  } catch (err) {
    const message = extractApiErrorMessage(err, t('admin.upstreamAccounts.loadFailed'))
    loadError.value = message
    result.value = null
    appStore.showError(message)
  } finally {
    loading.value = false
    loadingRateGuardConfig.value = false
  }
}

async function runSync() {
  const confirmed = window.confirm(
    t('admin.upstreamAccounts.syncConfirm', {
      create: summary.value.create_count,
      update: summary.value.update_count,
      unbind: summary.value.unbound_group_count
    })
  )
  if (!confirmed) {
    return
  }

  syncing.value = true
  try {
    result.value = await adminAPI.upstreamAccountSync.runSync({
      create_missing: true,
      update_existing: true,
      apply_rate_guard: true
    })
    await syncMatchedAccounts(result.value?.items || [])
    appStore.showSuccess(t('admin.upstreamAccounts.syncSuccess'))
  } catch (err) {
    appStore.showError(extractApiErrorMessage(err, t('admin.upstreamAccounts.syncFailed')))
  } finally {
    syncing.value = false
  }
}

function applyRateGuardConfig(config: UpstreamAccountRateGuardConfig) {
  rateGuardConfig.value = config
  rateGuardForm.value = {
    enabled: Boolean(config.enabled),
    interval_seconds: Number(config.interval_seconds) > 0 ? Number(config.interval_seconds) : 3600
  }
}

async function saveRateGuardConfig() {
  if (!Number.isInteger(rateGuardForm.value.interval_seconds) || rateGuardForm.value.interval_seconds <= 0) {
    appStore.showError(t('admin.upstreamAccounts.invalidRateGuardInterval'))
    return
  }
  savingRateGuardConfig.value = true
  try {
    const base = rateGuardConfig.value || {
      enabled: false,
      interval_seconds: 3600
    }
    const config = await adminAPI.upstreamAccountSync.updateRateGuardConfig({
      ...base,
      enabled: rateGuardForm.value.enabled,
      interval_seconds: rateGuardForm.value.interval_seconds
    })
    applyRateGuardConfig(config)
    appStore.showSuccess(t('admin.upstreamAccounts.rateGuardSaved'))
  } catch (err) {
    appStore.showError(extractApiErrorMessage(err, t('admin.upstreamAccounts.rateGuardSaveFailed')))
  } finally {
    savingRateGuardConfig.value = false
  }
}

async function runRateGuardNow() {
  runningRateGuardNow.value = true
  try {
    const config = await adminAPI.upstreamAccountSync.runRateGuardNow()
    applyRateGuardConfig(config)
    const preview = await adminAPI.upstreamAccountSync.getPreview()
    result.value = preview
    const remainingRisks = preview.summary?.rate_violation_count || 0
    if (remainingRisks > 0) {
      appStore.showWarning(t('admin.upstreamAccounts.rateGuardRunCompletedWithRisks', { count: remainingRisks }))
    } else {
      appStore.showSuccess(t('admin.upstreamAccounts.rateGuardRunSuccess'))
    }
  } catch (err) {
    appStore.showError(extractApiErrorMessage(err, t('admin.upstreamAccounts.rateGuardRunFailed')))
  } finally {
    runningRateGuardNow.value = false
  }
}

function formatRate(value: number | undefined) {
  const n = Number(value)
  return Number.isFinite(n) ? `${n.toFixed(2)}x` : '-'
}

async function loadLocalGroups() {
  try {
    localGroups.value = await adminAPI.groups.getAllIncludingInactive()
  } catch {
    localGroups.value = []
  }
}

async function syncMatchedAccounts(syncItems: UpstreamAccountSyncItem[]) {
  const matchedIDs = Array.from(
    new Set(
      syncItems
        .map(item => Number(item.matched_account_id))
        .filter(id => Number.isFinite(id) && id > 0)
    )
  )
  if (!matchedIDs.length) {
    matchedAccountsById.value = {}
    return
  }

  const entries = await Promise.allSettled(
    matchedIDs.map(async (accountId) => {
      const account = await adminAPI.accounts.getById(accountId)
      return [accountId, account] as const
    })
  )

  const nextMap: Record<number, Account> = {}
  const nextTestStatusMap: Record<number, AccountTestStatus> = {}
  for (const entry of entries) {
    if (entry.status !== 'fulfilled') continue
    const [accountId, account] = entry.value
    nextMap[accountId] = account
    if (account.last_test_status === 'success' || account.last_test_status === 'failed') {
      nextTestStatusMap[accountId] = account.last_test_status
    }
  }
  matchedAccountsById.value = nextMap
  const currentTestStatuses = accountTestStatusById.value
  for (const [accountId, status] of Object.entries(currentTestStatuses)) {
    const numericId = Number(accountId)
    if (!Number.isFinite(numericId) || numericId <= 0) continue
    if (!nextTestStatusMap[numericId]) {
      nextTestStatusMap[numericId] = status
    }
  }
  accountTestStatusById.value = nextTestStatusMap
}

async function ensureMatchedAccount(row: UpstreamAccountSyncItem) {
  const accountId = Number(row.matched_account_id)
  if (!Number.isFinite(accountId) || accountId <= 0) return null
  const cached = matchedAccountsById.value[accountId]
  if (cached) return cached
  try {
    const account = await adminAPI.accounts.getById(accountId)
    matchedAccountsById.value = {
      ...matchedAccountsById.value,
      [accountId]: account
    }
    return account
  } catch (err) {
    appStore.showError(extractApiErrorMessage(err, t('admin.upstreamAccounts.loadAccountFailed')))
    return null
  }
}

function getMatchedAccount(row: UpstreamAccountSyncItem) {
  const accountId = Number(row.matched_account_id)
  if (!Number.isFinite(accountId) || accountId <= 0) return null
  return matchedAccountsById.value[accountId] || null
}

function accountRowClass(row: UpstreamAccountSyncItem) {
  if (row.rate_violation) return 'risk-row'
  return ''
}

function accountTestStatusLabel(status: AccountTestStatus | undefined) {
  if (status === 'testing') return t('admin.upstreamAccounts.testStatusTesting')
  if (status === 'failed') return t('admin.upstreamAccounts.testStatusFailed')
  if (status === 'success') return t('admin.upstreamAccounts.testStatusSuccess')
  return '-'
}

function sourceToneClass(row: UpstreamAccountSyncItem) {
  if (row.rate_violation) return 'source-line-red'
  const slug = (row.provider_slug || row.provider_name || '').toLowerCase()
  if (slug.includes('niko')) return 'source-line-blue'
  return 'source-line-emerald'
}

function providerToneClass(providerSlug: string | undefined, target: 'card' | 'tag') {
  const tones = [
    'sky',
    'emerald',
    'violet',
    'cyan',
    'rose',
    'amber',
    'indigo',
    'teal'
  ]
  const slug = providerSlug?.trim() || 'default'
  let hash = 0
  for (let i = 0; i < slug.length; i++) {
    hash = (hash * 31 + slug.charCodeAt(i)) >>> 0
  }
  const tone = tones[hash % tones.length]
  return `${target === 'card' ? 'source-card' : 'tag-provider'}-${tone}`
}

function matchedAccountPlatform(row: UpstreamAccountSyncItem) {
  return getMatchedAccount(row)?.platform || inferAccountGroupPlatform(row) || ''
}

function matchedAccountPlatformTagClass(row: UpstreamAccountSyncItem) {
  return platformTagClass(matchedAccountPlatform(row))
}

function matchedAccountPlatformTextClass(row: UpstreamAccountSyncItem) {
  return platformTextToneClass(matchedAccountPlatform(row))
}

function platformTagClass(platform: string | undefined) {
  if (platform === 'anthropic') return 'platform-tag-anthropic'
  if (platform === 'openai') return 'platform-tag-openai'
  if (platform === 'antigravity') return 'platform-tag-antigravity'
  if (platform === 'gemini') return 'platform-tag-gemini'
  return 'platform-tag-default'
}

function platformTextToneClass(platform: string | undefined) {
  if (platform === 'anthropic') return 'platform-text-anthropic'
  if (platform === 'openai') return 'platform-text-openai'
  if (platform === 'antigravity') return 'platform-text-antigravity'
  if (platform === 'gemini') return 'platform-text-gemini'
  return 'platform-text-default'
}

async function openAccountGroupDialog(row: UpstreamAccountSyncItem) {
  accountGroupDialogItem.value = row
  accountGroupIds.value = (row.bound_groups || [])
    .map((group: UpstreamAccountSyncBoundGroup) => Number(group.id))
    .filter((id) => Number.isFinite(id))
  accountGroupPlatform.value = inferAccountGroupPlatform(row)

  const account = await ensureMatchedAccount(row)
  if (account) {
    accountGroupPlatform.value = account.platform
  }
}

function closeAccountGroupDialog() {
  if (savingAccountGroupId.value) return
  accountGroupDialogItem.value = null
  accountGroupIds.value = []
  accountGroupPlatform.value = undefined
}

async function saveAccountGroups() {
  const row = accountGroupDialogItem.value
  if (!row?.matched_account_id) return
  savingAccountGroupId.value = row.matched_account_id
  try {
    const updated = await adminAPI.accounts.update(row.matched_account_id, { group_ids: accountGroupIds.value })
    if (updated) updateMatchedAccount(updated)
    accountGroupDialogItem.value = null
    accountGroupIds.value = []
    accountGroupPlatform.value = undefined
    appStore.showSuccess(t('admin.upstreamAccounts.boundGroupsSaved'))
  } catch (err) {
    appStore.showError(extractApiErrorMessage(err, t('admin.upstreamAccounts.boundGroupsSaveFailed')))
  } finally {
    savingAccountGroupId.value = null
  }
}

async function clearAccountGroups() {
  accountGroupIds.value = []
  await saveAccountGroups()
}

async function openAccountTestDialog(row: UpstreamAccountSyncItem) {
  const accountId = Number(row.matched_account_id)
  if (!Number.isFinite(accountId) || accountId <= 0) return
  testingAccountId.value = accountId
  try {
    testingAccount.value = await ensureMatchedAccount(row)
    if (!testingAccount.value) return
    showTestModal.value = true
  } catch (err) {
    appStore.showError(extractApiErrorMessage(err, t('admin.upstreamAccounts.testConnectionFailed')))
  } finally {
    testingAccountId.value = null
  }
}

function closeTestModal() {
  showTestModal.value = false
  testingAccount.value = null
}

function handleAccountTestResult(payload: { accountId: number; status: AccountTestStatus }) {
  accountTestStatusById.value = {
    ...accountTestStatusById.value,
    [payload.accountId]: payload.status
  }
}

function updateMatchedAccount(account: Account) {
  matchedAccountsById.value = {
    ...matchedAccountsById.value,
    [account.id]: account
  }
}

async function handleToggleSchedulable(account: Account) {
  togglingSchedulableId.value = account.id
  try {
    const updated = await adminAPI.accounts.setSchedulable(account.id, !account.schedulable)
    updateMatchedAccount(updated || { ...account, schedulable: !account.schedulable })
  } catch (err) {
    appStore.showError(extractApiErrorMessage(err, t('admin.accounts.failedToToggleSchedulable')))
  } finally {
    togglingSchedulableId.value = null
  }
}

function handleShowTempUnsched(account: Account) {
  tempUnschedAccount.value = account
  showTempUnsched.value = true
}

function handleTempUnschedReset(updated: Account) {
  tempUnschedAccount.value = null
  showTempUnsched.value = false
  updateMatchedAccount(updated)
}

function closeTempUnschedModal() {
  tempUnschedAccount.value = null
  showTempUnsched.value = false
}

function groupNameById(groupID: number) {
  const group = localGroups.value.find(item => item.id === groupID)
  return group ? `${group.name} ${formatRate(group.rate_multiplier)}` : `#${groupID}`
}

function inferAccountGroupPlatform(row: UpstreamAccountSyncItem): GroupPlatform | undefined {
  const groupIDs = [
    row.local_group_id,
    ...(row.bound_groups || []).map((group) => group.id)
  ]
    .map((id) => Number(id))
    .filter((id) => Number.isFinite(id))
  for (const groupID of groupIDs) {
    const group = localGroups.value.find(item => item.id === groupID)
    if (group?.platform) return group.platform
  }
  return undefined
}

function rateToneClass(value: number | undefined) {
  const n = Number(value)
  if (!Number.isFinite(n)) return 'rate-muted'
  if (n >= 0.4) return 'rate-red'
  if (n >= 0.3) return 'rate-deep-orange'
  if (n >= 0.2) return 'rate-orange'
  if (n > 0.1) return 'rate-green'
  return 'rate-deep-green'
}

function rateProgressWidth(value: number | undefined) {
  const n = Number(value)
  if (!Number.isFinite(n) || n <= 0) return '0%'
  return `${Math.min(100, Math.max(8, (n / 0.5) * 100))}%`
}

function triggerClass(triggerSource: string | undefined) {
  if (triggerSource === 'scheduled_rate_guard') return 'trigger-scheduled'
  if (triggerSource === 'manual_rate_guard') return 'trigger-guard'
  return 'trigger-sync'
}

function conflictAccountRates(account: UpstreamAccountSyncConflictAccount) {
  return (account.bound_groups || [])
    .map(group => formatRate(group.rate_multiplier))
    .join(' / ')
}

function numberArray(value: unknown): number[] {
  return Array.isArray(value) ? value.filter((item): item is number => Number.isFinite(Number(item))).map(Number) : []
}

function stringArray(value: unknown): string[] {
  return Array.isArray(value) ? value.map(String).filter(Boolean) : []
}

function conflictAccountTitle(account: UpstreamAccountSyncConflictAccount) {
  const groups = (account.bound_groups || [])
    .map(group => `${group.name} ${formatRate(group.rate_multiplier)}`)
    .join(', ')
  return groups ? `${account.name}: ${groups}` : account.name
}

function upstreamAccountSyncTriggerSourceLabel(triggerSource: string | undefined) {
  if (triggerSource === 'scheduled_rate_guard') return t('admin.upstreamAccounts.triggerScheduledRateGuard')
  if (triggerSource === 'manual_rate_guard') return t('admin.upstreamAccounts.triggerManualRateGuard')
  return t('admin.upstreamAccounts.triggerManualSync')
}

function getProviderBalance(providerSlug: string): number | null {
  if (!balanceOverview.value?.summaries) return null
  const summary = balanceOverview.value.summaries[providerSlug]
  return summary ? summary.current_balance : null
}

function getProviderConsumption(providerSlug: string): number | null {
  if (!balanceOverview.value?.summaries) return null
  const summary = balanceOverview.value.summaries[providerSlug]
  return summary ? summary.today_consumption : null
}

function formatMoney(value: number): string {
  if (value >= 1000000) {
    return (value / 1000000).toFixed(2) + 'M'
  } else if (value >= 1000) {
    return (value / 1000).toFixed(2) + 'K'
  } else if (value >= 1) {
    return value.toFixed(2)
  } else if (value >= 0.01) {
    return value.toFixed(3)
  }
  return value.toFixed(4)
}

function openTrendModal(providerSlug: string, providerName: string) {
  trendProviderSlug.value = providerSlug
  trendProviderName.value = providerName
  showTrendModal.value = true
}

function closeTrendModal() {
  showTrendModal.value = false
}

onMounted(reload)
</script>

<style scoped>
.upstream-accounts-page {
  width: 100%;
  max-width: none;
  margin: 0;
}

.upstream-accounts-page :deep(.table-page-layout) {
  gap: 16px;
  width: 100%;
  max-width: none;
  height: auto;
  min-height: calc(100vh - 64px - 4rem);
}

.upstream-accounts-page :deep(.layout-section-scrollable) {
  overflow: visible;
}

.upstream-accounts-page :deep(.table-scroll-container) {
  border: 0;
  border-radius: 0;
  background: transparent;
  box-shadow: none;
  overflow: auto;
}

.accounts-shell {
  display: grid;
  gap: 16px;
}

.accounts-topbar {
  display: grid;
  grid-template-columns: minmax(0, 1fr) auto;
  gap: 16px;
  align-items: center;
}

.stats-strip {
  display: grid;
  grid-template-columns: repeat(4, minmax(0, 1fr));
  gap: 12px;
}

.stat-card,
.rate-guard-panel,
.accounts-table-card,
.records-panel {
  border: 1px solid #e5e7eb;
  border-radius: 8px;
  background: #fff;
  box-shadow: 0 8px 24px rgba(15, 23, 42, 0.04);
}

.stat-card {
  position: relative;
  display: flex;
  min-height: 82px;
  align-items: center;
  gap: 12px;
  padding: 16px;
}

.stat-alert-dot {
  position: absolute;
  top: 12px;
  right: 12px;
  width: 8px;
  height: 8px;
  border-radius: 999px;
  background: #dc2626;
}

.stat-icon {
  display: grid;
  width: 40px;
  height: 40px;
  flex: none;
  place-items: center;
  border-radius: 999px;
}

.stat-copy {
  display: flex;
  min-width: 0;
  flex-direction: column;
}

.stat-copy strong {
  font-size: 24px;
  font-weight: 750;
  line-height: 1.1;
  color: #111827;
}

.stat-copy span {
  margin-top: 4px;
  color: #64748b;
  font-size: 12px;
  font-weight: 500;
}

.stat-card-emerald .stat-icon {
  background: #ecfdf5;
  color: #059669;
}

.stat-card-gray .stat-icon {
  background: #f1f5f9;
  color: #64748b;
}

.stat-card-orange .stat-icon {
  background: #fff7ed;
  color: #d97706;
}

.stat-card-red .stat-icon {
  background: #fef2f2;
  color: #dc2626;
}

.accounts-actions {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  justify-content: flex-end;
  gap: 10px;
}

.provider-summary {
  display: grid;
  min-width: 150px;
  gap: 2px;
  color: #64748b;
  font-size: 12px;
}

.provider-summary strong {
  overflow: hidden;
  color: #111827;
  font-size: 13px;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.provider-summary code {
  color: #64748b;
  font-size: 11px;
}

.ui-button {
  display: inline-flex;
  min-height: 38px;
  align-items: center;
  justify-content: center;
  gap: 8px;
  border: 1px solid #e5e7eb;
  border-radius: 8px;
  background: #fff;
  padding: 0 14px;
  color: #374151;
  font-weight: 600;
  transition: border-color 150ms ease, background 150ms ease, color 150ms ease, box-shadow 150ms ease;
}

.ui-button:hover:not(:disabled) {
  border-color: #a7f3d0;
  color: #059669;
  box-shadow: 0 0 0 3px rgba(5, 150, 105, 0.08);
}

.ui-button:disabled {
  cursor: not-allowed;
  opacity: 0.55;
}

.ui-button-primary {
  border-color: #059669;
  background: #059669;
  color: #fff;
}

.ui-button-primary:hover:not(:disabled) {
  border-color: #047857;
  background: #047857;
  color: #fff;
}

.ui-button-icon {
  width: 38px;
  padding: 0;
}

.rate-guard-panel {
  display: grid;
  grid-template-columns: minmax(280px, 1fr) auto;
  gap: 20px;
  align-items: center;
  padding: 18px;
}

.guard-left {
  display: flex;
  min-width: 0;
  align-items: flex-start;
  gap: 14px;
}

.guard-switch {
  position: relative;
  width: 42px;
  height: 24px;
  flex: none;
  margin-top: 2px;
  cursor: pointer;
}

.guard-switch input {
  position: absolute;
  opacity: 0;
}

.guard-switch span {
  display: block;
  width: 42px;
  height: 24px;
  border-radius: 999px;
  background: #cbd5e1;
  transition: background 150ms ease;
}

.guard-switch span::after {
  content: "";
  position: absolute;
  top: 3px;
  left: 3px;
  width: 18px;
  height: 18px;
  border-radius: 999px;
  background: #fff;
  box-shadow: 0 1px 3px rgba(15, 23, 42, 0.2);
  transition: transform 150ms ease;
}

.guard-switch.is-on span {
  background: #059669;
}

.guard-switch.is-on span::after {
  transform: translateX(18px);
}

.guard-title {
  color: #111827;
  font-size: 15px;
  font-weight: 700;
}

.guard-description {
  margin-top: 4px;
  color: #64748b;
  font-size: 12px;
}

.guard-status-line {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 8px;
  margin-top: 10px;
  color: #64748b;
  font-size: 12px;
}

.guard-controls {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  justify-content: flex-end;
  gap: 10px;
}

.control-label {
  color: #475569;
  font-size: 13px;
  font-weight: 600;
  white-space: nowrap;
}

.guard-hint {
  color: #64748b;
  font-size: 12px;
}

.ui-input {
  height: 38px;
  border: 1px solid #d1d5db;
  border-radius: 8px;
  background: #fff;
  color: #111827;
  outline: none;
  padding: 0 12px;
  transition: border-color 150ms ease, box-shadow 150ms ease;
}

.ui-input:focus {
  border-color: #059669;
  box-shadow: 0 0 0 3px rgba(5, 150, 105, 0.12);
}

.interval-input {
  width: 92px;
}

.filter-row {
  display: grid;
  grid-template-columns: 156px 172px minmax(260px, 1fr) auto;
  gap: 12px;
  align-items: center;
}

.filter-select {
  width: 100%;
}

.filter-select :deep(select),
.filter-select :deep(button) {
  min-height: 38px;
  border-radius: 8px;
  border-color: #d1d5db;
  background: #fff;
}

.search-wrap {
  position: relative;
  min-width: 0;
}

.search-wrap > svg {
  position: absolute;
  top: 50%;
  left: 12px;
  color: #94a3b8;
  transform: translateY(-50%);
}

.filter-search {
  width: 100%;
  padding-left: 38px;
}

.filtered-count {
  display: inline-flex;
  height: 34px;
  align-items: center;
  gap: 6px;
  border-radius: 8px;
  background: #f1f5f9;
  padding: 0 12px;
  color: #64748b;
  white-space: nowrap;
}

.filtered-count strong {
  color: #111827;
  font-weight: 750;
}

.quick-tags {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
}

.quick-tag {
  height: 30px;
  border: 1px solid #e5e7eb;
  border-radius: 8px;
  background: #fff;
  padding: 0 12px;
  color: #334155;
  font-size: 12px;
  font-weight: 650;
  transition: border-color 150ms ease, background 150ms ease, color 150ms ease;
}

.quick-tag:hover {
  border-color: #059669;
  color: #059669;
}

.quick-tag.active {
  border-color: #059669;
  background: #059669;
  color: #fff;
}

.accounts-table-content {
  display: flex;
  min-height: 0;
  flex-direction: column;
  gap: 16px;
  overflow: visible;
}

.warning-banner {
  display: flex;
  gap: 8px;
  border: 1px solid #fed7aa;
  border-radius: 8px;
  background: #fff7ed;
  padding: 12px;
  color: #c2410c;
  font-size: 13px;
}

.accounts-table-card {
  display: flex;
  flex: none;
  min-height: 0;
  overflow: auto;
  height: auto;
  max-height: 42rem;
}

.accounts-table-card :deep(.table-wrapper) {
  display: flex;
  flex: 1;
  min-height: 0;
}

.accounts-table-card :deep(table) {
  border-collapse: collapse;
}

.accounts-table-card :deep(thead),
.accounts-table-card :deep(.table-header),
.accounts-table-card :deep(.sticky-header-cell) {
  background: #f8fafc;
}

.accounts-table-card :deep(th) {
  border-bottom: 1px solid #e5e7eb;
  color: #64748b;
  font-size: 12px;
  font-weight: 600;
  text-transform: none;
  letter-spacing: 0;
}

.accounts-table-card :deep(th.upstream-center-column),
.accounts-table-card :deep(td.upstream-center-column) {
  text-align: center;
}

.accounts-table-card :deep(th.upstream-center-column > div) {
  justify-content: center;
}

.accounts-table-card :deep(td) {
  border-bottom: 1px solid #eef2f7;
  color: #334155;
}

.accounts-table-card :deep(.data-table-row) {
  transition: background 150ms ease;
}

.accounts-table-card :deep(.data-table-row:hover) {
  background: #f8fafc;
}

.accounts-table-card :deep(.data-table-row.risk-row),
.accounts-table-card :deep(.data-table-row.risk-row .sticky-col) {
  background: #fff7f7;
}

.accounts-table-card :deep(.data-table-row.risk-row:hover),
.accounts-table-card :deep(.data-table-row.risk-row:hover .sticky-col) {
  background: #fef2f2;
}

.source-cell {
  display: grid;
  min-width: 13rem;
  grid-template-columns: 2px minmax(0, 1fr) auto;
  align-items: center;
  gap: 10px;
}

.source-line {
  width: 2px;
  height: 42px;
  border-radius: 999px;
}

.source-line-emerald {
  background: #059669;
}

.source-line-blue {
  background: #2563eb;
}

.source-line-red {
  background: #dc2626;
}

.source-title {
  display: flex;
  min-width: 0;
  align-items: center;
  gap: 6px;
}

.source-warning-icon {
  color: #dc2626;
}

.source-id,
.sub-text {
  display: block;
  overflow: hidden;
  margin-top: 5px;
  color: #64748b;
  font-size: 12px;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.source-main,
.two-line-cell {
  min-width: 0;
}

.main-text {
  display: block;
  overflow: hidden;
  color: #1f2937;
  font-weight: 500;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.table-tag,
.group-chip,
.log-chip,
.trigger-chip,
.status-pill {
  display: inline-flex;
  max-width: 100%;
  align-items: center;
  gap: 6px;
  border-radius: 6px;
  padding: 2px 8px;
  font-size: 12px;
  font-weight: 600;
  line-height: 18px;
  white-space: nowrap;
}

.table-tag {
  overflow: hidden;
  text-overflow: ellipsis;
}

.status-pill::before {
  content: "";
  width: 6px;
  height: 6px;
  border-radius: 999px;
  background: currentColor;
}

.status-pill-on {
  background: #ecfdf5;
  color: #059669;
}

.status-pill-muted,
.tag-gray,
.home-tag,
.tag-account {
  background: #f1f5f9;
  color: #64748b;
}

.record-status {
  border-radius: 6px;
  padding: 2px 8px;
  font-size: 12px;
  font-weight: 600;
}

.record-status-success {
  background: #ecfdf5;
  color: #047857;
}

.record-status-error,
.status-error-message,
.sub-text-warning {
  color: #dc2626;
}

.tag-provider-sky,
.tag-provider-cyan,
.tag-provider-indigo {
  background: #eff6ff;
  color: #1d4ed8;
}

.tag-provider-emerald,
.tag-provider-teal {
  background: #ecfdf5;
  color: #047857;
}

.tag-provider-violet {
  background: #f5f3ff;
  color: #6d28d9;
}

.tag-provider-rose {
  background: #fef2f2;
  color: #b91c1c;
}

.tag-provider-amber,
.tag-warning {
  background: #fff7ed;
  color: #c2410c;
}

.account-id-tag {
  margin-top: 6px;
}

.balance-cell {
  display: inline-flex;
  align-items: center;
  gap: 8px;
}

.balance-value {
  color: #059669;
  font-variant-numeric: tabular-nums;
  font-weight: 650;
}

.consumption-value {
  color: #ea580c;
  font-variant-numeric: tabular-nums;
  font-weight: 650;
}

.trend-btn {
  display: grid;
  width: 24px;
  height: 24px;
  place-items: center;
  border: 1px solid #e5e7eb;
  border-radius: 4px;
  background: #fff;
  color: #64748b;
  cursor: pointer;
  transition: all 150ms ease;
}

.trend-btn:hover {
  border-color: #2563eb;
  background: #eff6ff;
  color: #2563eb;
  transform: translateY(-1px);
  box-shadow: 0 2px 8px rgba(37, 99, 235, 0.2);
}

.trend-btn:active {
  transform: translateY(0);
}

.platform-text-openai {
  color: #047857;
}

.platform-text-anthropic {
  color: #c2410c;
}

.platform-text-gemini {
  color: #1d4ed8;
}

.platform-text-antigravity {
  color: #6d28d9;
}

.platform-text-default {
  color: #1f2937;
}

.platform-tag-openai {
  background: #ecfdf5;
  color: #047857;
}

.platform-tag-anthropic {
  background: #fff7ed;
  color: #c2410c;
}

.platform-tag-gemini {
  background: #eff6ff;
  color: #1d4ed8;
}

.platform-tag-antigravity {
  background: #f5f3ff;
  color: #6d28d9;
}

.platform-tag-default {
  background: #f1f5f9;
  color: #64748b;
}

.rate-cell {
  display: grid;
  justify-items: end;
  gap: 7px;
}

.rate-value {
  min-width: 62px;
  justify-content: center;
  font-variant-numeric: tabular-nums;
}

.rate-bar {
  display: block;
  width: 76px;
  height: 4px;
  overflow: hidden;
  border-radius: 999px;
  background: #e5e7eb;
}

.rate-bar span {
  display: block;
  height: 100%;
  border-radius: inherit;
  background: currentColor;
}

.rate-deep-green {
  background: #d1fae5;
  color: #065f46;
}

.rate-green {
  background: #ecfdf5;
  color: #047857;
}

.rate-orange {
  background: #ffedd5;
  color: #c2410c;
}

.rate-deep-orange {
  background: #fed7aa;
  color: #9a3412;
}

.rate-red {
  background: #fef2f2;
  color: #b91c1c;
}

.rate-muted {
  background: #f1f5f9;
  color: #64748b;
}

.tag-list {
  display: flex;
  max-width: none;
  flex-wrap: wrap;
  gap: 6px;
}

.group-list {
  min-width: 16rem;
  white-space: normal;
}

.group-chip-blue {
  background: #eff6ff;
  color: #1d4ed8;
}

.group-chip-emerald {
  background: #ecfdf5;
  color: #047857;
}

.group-chip-violet {
  background: #f5f3ff;
  color: #6d28d9;
}

.group-chip-warning,
.log-chip-warning {
  background: #fef2f2;
  color: #b91c1c;
}

.dash {
  color: #94a3b8;
}

.action-cell {
  display: flex;
  align-items: center;
  justify-content: flex-end;
  gap: 10px;
}

.test-status-cell {
  display: flex;
  align-items: center;
  min-height: 32px;
}

.test-status-pill {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  border-radius: 6px;
  padding: 2px 8px;
  font-size: 12px;
  font-weight: 600;
  line-height: 18px;
}

.test-status-testing {
  background: #eff6ff;
  color: #2563eb;
}

.test-status-success {
  background: #ecfdf5;
  color: #059669;
}

.test-status-failed {
  background: #fef2f2;
  color: #dc2626;
}

.status-cell {
  display: flex;
  align-items: center;
  justify-content: center;
  min-height: 32px;
}

.schedulable-toggle {
  display: inline-flex;
  align-items: center;
  border: 0;
  background: transparent;
  padding: 0;
}

.schedulable-track {
  display: inline-flex;
  width: 38px;
  height: 22px;
  align-items: center;
  border-radius: 999px;
  padding: 2px;
  transition: background 150ms ease;
}

.schedulable-on .schedulable-track {
  background: #059669;
}

.schedulable-off .schedulable-track {
  background: #cbd5e1;
}

.schedulable-thumb {
  display: block;
  width: 16px;
  height: 16px;
  border-radius: 999px;
  background: #fff;
  box-shadow: 0 1px 2px rgba(15, 23, 42, 0.24);
  transition: transform 150ms ease;
}

.schedulable-thumb-on {
  transform: translateX(16px);
}

.schedulable-thumb-off {
  transform: translateX(0);
}

.accounts-table-card :deep(.upstream-rate-column .rate-cell) {
  justify-items: center;
}

.accounts-table-card :deep(.upstream-rate-column .rate-value) {
  min-width: 72px;
}

.accounts-table-card :deep(.upstream-rate-column .rate-bar) {
  margin-inline: auto;
}

.action-dash {
  display: flex;
  justify-content: flex-end;
}

.text-action {
  display: inline-flex;
  align-items: center;
  gap: 5px;
  border: 0;
  background: transparent;
  padding: 4px 0;
  color: #059669;
  font-weight: 650;
  transition: color 150ms ease;
}

.text-action:hover:not(:disabled) {
  color: #047857;
}

.text-action:disabled {
  cursor: not-allowed;
  opacity: 0.55;
}

.text-action-danger {
  color: #dc2626;
}

.text-action-muted {
  color: #64748b;
}

.records-panel {
  overflow: hidden;
}

.records-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  border-bottom: 1px solid #e5e7eb;
  padding: 16px 18px;
}

.records-header h3 {
  margin: 0;
  color: #111827;
  font-size: 15px;
  font-weight: 750;
}

.records-header span {
  color: #64748b;
  font-size: 12px;
}

.records-info {
  margin: 14px 18px 0;
  border-radius: 8px;
  background: #f8fafc;
  padding: 10px 12px;
  color: #64748b;
  font-size: 12px;
}

.records-table-wrap {
  max-height: 20rem;
  overflow: auto;
}

.records-table {
  min-width: 1080px;
  width: 100%;
  border-collapse: collapse;
  font-size: 13px;
}

.records-table th {
  border-bottom: 1px solid #e5e7eb;
  background: #f8fafc;
  padding: 10px 16px;
  color: #64748b;
  font-weight: 600;
  text-align: left;
}

.records-table td {
  border-bottom: 1px solid #eef2f7;
  padding: 12px 16px;
  color: #334155;
  vertical-align: top;
}

.records-table tbody tr {
  transition: background 150ms ease;
}

.records-table tbody tr:hover {
  background: #f8fafc;
}

.trigger-sync {
  background: #ecfdf5;
  color: #047857;
}

.trigger-scheduled {
  background: #f5f3ff;
  color: #6d28d9;
}

.trigger-guard {
  background: #fff7ed;
  color: #c2410c;
}

.log-chip {
  background: #f1f5f9;
  color: #475569;
}

.rate-compare {
  display: inline-flex;
  align-items: center;
  gap: 8px;
  border-radius: 6px;
  background: #f1f5f9;
  padding: 4px 8px;
  font-family: ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, monospace;
  font-size: 12px;
  font-weight: 650;
}

.rate-compare-upstream {
  color: #c2410c;
}

.rate-compare-local {
  color: #047857;
}

.records-empty {
  display: grid;
  place-items: center;
  gap: 10px;
  padding: 42px 16px 46px;
  color: #64748b;
  text-align: center;
}

.records-empty svg {
  color: #cbd5e1;
}

@media (max-width: 1023px) {
  .accounts-topbar,
  .rate-guard-panel,
  .filter-row {
    grid-template-columns: 1fr;
  }

  .accounts-actions,
  .guard-controls {
    justify-content: flex-start;
  }

  .accounts-table-content {
    height: auto;
    overflow: visible;
  }

  .accounts-table-card {
    height: auto;
    min-height: 0;
    overflow: visible;
  }
}

@media (max-width: 768px) {
  .stats-strip {
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }
}

@media (max-width: 520px) {
  .stats-strip {
    grid-template-columns: 1fr;
  }
}
</style>

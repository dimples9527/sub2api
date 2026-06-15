<template>
  <AppLayout>
    <TablePageLayout>
      <template #filters>
        <div class="accounts-toolbar">
          <div class="provider-panel">
            <div class="min-w-0">
              <div class="meta-label">{{ t('admin.upstreamAccounts.syncProviders') }}</div>
              <div class="mt-1 flex min-w-0 items-center gap-2 text-sm font-semibold text-gray-900 dark:text-white">
                <span class="truncate">{{ syncProviderLabel }}</span>
                <code v-if="syncProviderCode" class="text-xs font-normal text-gray-500 dark:text-gray-400">
                  {{ syncProviderCode }}
                </code>
              </div>
            </div>
            <div class="provider-count">{{ syncProviders.length }}</div>
          </div>

          <div class="stats-strip">
            <div class="stat-tile">
              <span>{{ t('admin.upstreamAccounts.upstreamKeys') }}</span>
              <strong>{{ summary.upstream_key_count }}</strong>
            </div>
            <div class="stat-tile stat-tile-create">
              <span>{{ t('admin.upstreamAccounts.toCreate') }}</span>
              <strong>{{ summary.create_count }}</strong>
            </div>
            <div class="stat-tile stat-tile-update">
              <span>{{ t('admin.upstreamAccounts.toUpdate') }}</span>
              <strong>{{ summary.update_count }}</strong>
            </div>
            <div class="stat-tile" :class="summary.rate_violation_count > 0 ? 'stat-tile-warning' : ''">
              <span>{{ t('admin.upstreamAccounts.rateRisks') }}</span>
              <strong>{{ summary.rate_violation_count }}</strong>
            </div>
          </div>

          <div class="accounts-actions">
            <button
              type="button"
              class="btn btn-secondary"
              :disabled="loading || syncing"
              :title="t('common.refresh')"
              @click="reload"
            >
              <Icon name="refresh" size="md" :class="loading ? 'animate-spin' : ''" />
            </button>
            <button
              type="button"
              class="btn btn-primary"
              :disabled="loading || syncing || !canSync"
              @click="runSync"
            >
              <Icon name="sync" size="sm" class="mr-2" :class="syncing ? 'animate-spin' : ''" />
              {{ t('admin.upstreamAccounts.syncNow') }}
            </button>
          </div>
        </div>

        <div class="filter-row">
          <div class="relative min-w-0">
            <Icon
              name="search"
              size="md"
              class="absolute left-3 top-1/2 -translate-y-1/2 text-gray-400 dark:text-gray-500"
            />
            <input
              v-model.trim="searchQuery"
              type="search"
              class="input filter-search pl-10"
              :placeholder="t('admin.upstreamAccounts.searchPlaceholder')"
            />
          </div>
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
          <div class="filtered-count">
            <span>{{ t('admin.upstreamAccounts.filteredCount') }}</span>
            <strong>{{ filteredItems.length }}</strong>
          </div>
        </div>

        <div class="rate-guard-panel">
          <div class="min-w-0">
            <div class="meta-label">{{ t('admin.upstreamAccounts.rateGuardTitle') }}</div>
            <div class="mt-1 text-sm text-gray-600 dark:text-gray-300">
              {{ t('admin.upstreamAccounts.rateGuardDescription') }}
            </div>
            <div class="mt-2 flex flex-wrap items-center gap-2 text-xs">
              <span :class="['badge', rateGuardForm.enabled ? 'badge-success' : 'badge-gray']">
                {{ rateGuardForm.enabled ? t('admin.upstreamAccounts.rateGuardEnabled') : t('admin.upstreamAccounts.rateGuardDisabled') }}
              </span>
              <span class="text-gray-500 dark:text-gray-400">
                {{ t('admin.upstreamAccounts.rateGuardLastRun') }}:
                {{ rateGuardLastRunText }}
              </span>
              <span
                v-if="rateGuardConfig?.last_run_status"
                :class="['record-status', rateGuardConfig.last_run_status === 'failed' ? 'record-status-error' : 'record-status-success']"
              >
                {{ rateGuardConfig.last_run_status === 'failed' ? t('admin.upstreamAccounts.rateGuardStatusFailed') : t('admin.upstreamAccounts.rateGuardStatusSuccess') }}
              </span>
              <span v-if="rateGuardConfig?.last_run_message" class="text-red-600 dark:text-red-300">
                {{ rateGuardConfig.last_run_message }}
              </span>
            </div>
          </div>
          <label class="guard-toggle">
            <input v-model="rateGuardForm.enabled" type="checkbox" class="h-4 w-4 rounded border-gray-300 text-primary-600 focus:ring-primary-500" />
            <span>{{ t('admin.upstreamAccounts.rateGuardAutoRun') }}</span>
          </label>
          <label class="guard-interval">
            <span>{{ t('admin.upstreamAccounts.rateGuardIntervalSeconds') }}</span>
            <input
              v-model.number="rateGuardForm.interval_seconds"
              type="number"
              min="1"
              class="input h-9 w-28"
            />
          </label>
          <button
            type="button"
            class="btn btn-secondary"
            :disabled="loadingRateGuardConfig || savingRateGuardConfig"
            @click="saveRateGuardConfig"
          >
            <Icon name="cog" size="sm" class="mr-2" :class="savingRateGuardConfig ? 'animate-spin' : ''" />
            {{ t('common.save') }}
          </button>
          <button
            type="button"
            class="btn btn-primary"
            :disabled="loadingRateGuardConfig || savingRateGuardConfig || runningRateGuardNow"
            @click="runRateGuardNow"
          >
            <Icon name="play" size="sm" class="mr-2" :class="runningRateGuardNow ? 'animate-pulse' : ''" />
            {{ t('admin.upstreamAccounts.rateGuardRunNow') }}
          </button>
        </div>
      </template>

      <template #table>
        <div class="accounts-table-content">
          <div v-if="warnings.length" class="warning-banner">
            <div v-for="warning in warnings" :key="warning">{{ warning }}</div>
          </div>

          <div class="accounts-table-primary">
            <DataTable :columns="columns" :data="filteredItems" :loading="loading">
              <template #cell-source="{ row }">
                <div :class="['source-card min-w-[12rem]', providerToneClass(row.provider_slug, 'card')]">
                  <div class="flex items-center gap-2">
                    <span class="min-w-0 flex-1 truncate font-semibold text-gray-950 dark:text-white">{{ row.provider_name || row.provider_slug }}</span>
                    <a
                      v-if="row.provider_base_url"
                      :href="row.provider_base_url"
                      target="_blank"
                      rel="noopener noreferrer"
                      class="source-home-link"
                      :title="t('admin.upstreamProviders.openHomepage')"
                    >
                      <Icon name="home" size="sm" />
                      <span>{{ t('admin.upstreamProviders.homepageShort') }}</span>
                    </a>
                  </div>
                  <code :class="['table-tag', providerToneClass(row.provider_slug, 'tag')]">{{ row.provider_slug }}</code>
                </div>
              </template>

              <template #cell-upstream_key_name="{ row }">
                <div class="key-card min-w-[15rem]">
                  <span class="font-semibold text-gray-950 dark:text-white">{{ row.upstream_key_name }}</span>
                  <div class="tag-list max-w-[20rem]">
                    <span class="table-tag tag-group">{{ row.upstream_group_name }}</span>
                    <span v-if="row.rate_violation" class="table-tag tag-warning">
                      {{ t('admin.upstreamAccounts.rateRisks') }}
                    </span>
                  </div>
                </div>
              </template>

              <template #cell-local_account_name="{ row }">
                <div :class="['account-card min-w-[14rem]', accountCardClass(row)]">
                  <span class="font-semibold text-gray-950 dark:text-white">{{ row.local_account_name || '-' }}</span>
                  <span v-if="row.matched_account_id" class="text-xs text-gray-500 dark:text-gray-400">
                    <span class="table-tag tag-account">#{{ row.matched_account_id }} {{ row.matched_account_name }}</span>
                  </span>
                  <div v-else-if="row.conflict_accounts?.length" class="tag-list max-w-[24rem]">
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
                  <span v-else-if="row.conflict_account_ids?.length" class="text-xs text-amber-600 dark:text-amber-300">
                    {{ t('admin.upstreamAccounts.conflictIds', { ids: row.conflict_account_ids.join(', ') }) }}
                  </span>
                </div>
              </template>

              <template #cell-local_group_name="{ row }">
                <div class="table-main-cell min-w-[16rem]">
                  <div v-if="row.bound_groups?.length" class="tag-list max-w-[22rem]">
                    <span
                      v-for="(group, index) in row.bound_groups"
                      :key="`${row.provider_slug}-${row.upstream_key_name}-${group.id}`"
                      :class="['group-chip', groupChipClass(group.rate_violation, index)]"
                      :title="`${group.name} ${formatRate(group.rate_multiplier)}`"
                    >
                      {{ group.name }}
                      <span class="font-mono">{{ formatRate(group.rate_multiplier) }}</span>
                    </span>
                  </div>
                  <template v-else>
                    <span>{{ row.local_group_name || '-' }}</span>
                    <span v-if="row.local_rate_multiplier !== undefined" class="text-xs font-mono text-gray-500 dark:text-gray-400">
                      {{ formatRate(row.local_rate_multiplier) }}
                    </span>
                  </template>
                </div>
              </template>

              <template #cell-upstream_rate_multiplier="{ value }">
                <span :class="['rate-value', rateToneClass(value)]">{{ formatRate(value) }}</span>
              </template>

              <template #cell-balance_consumption="{ row }">
                <div class="balance-cost-cell">
                  <div class="font-mono text-sm font-semibold text-gray-950 dark:text-white">
                    {{ formatMoney(balanceSummaryFor(row.provider_slug)?.today_consumption) }}
                  </div>
                  <div class="mt-1 flex flex-wrap gap-1">
                    <span :class="['table-tag', balanceSummaryFor(row.provider_slug)?.complete ? 'tag-success' : 'tag-muted']">
                      {{ balanceSummaryFor(row.provider_slug)?.complete ? t('admin.upstreamAccounts.balanceComplete') : t('admin.upstreamAccounts.balanceIncomplete') }}
                    </span>
                    <span v-if="balanceSummaryFor(row.provider_slug)?.anomaly" class="table-tag tag-warning">
                      {{ t('admin.upstreamAccounts.balanceAnomaly') }}
                    </span>
                  </div>
                </div>
              </template>

              <template #cell-actions="{ row }">
                <button
                  type="button"
                  class="btn btn-secondary btn-sm"
                  @click="openBalanceDetails(row.provider_slug)"
                >
                  <Icon name="more" size="sm" class="mr-1" />
                  {{ t('common.more') }}
                </button>
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
          </div>

          <div
            v-if="balanceDetailsOpen"
            class="fixed inset-0 z-50 flex items-end justify-center bg-black/40 p-4 sm:items-center"
            @click.self="closeBalanceDetails"
          >
            <div class="balance-dialog">
              <div class="balance-dialog-header">
                <div class="min-w-0">
                  <h3 class="truncate text-base font-semibold text-gray-950 dark:text-white">
                    {{ selectedBalanceProviderLabel }}
                  </h3>
                  <p class="mt-1 text-sm text-gray-500 dark:text-gray-400">
                    {{ t('admin.upstreamAccounts.balanceDialogDescription') }}
                  </p>
                </div>
                <button type="button" class="btn btn-secondary btn-sm" @click="closeBalanceDetails">
                  {{ t('common.close') }}
                </button>
              </div>

              <div class="balance-dialog-body">
                <div class="balance-summary-grid">
                  <div class="balance-metric">
                    <span>{{ t('admin.upstreamAccounts.currentBalance') }}</span>
                    <strong>{{ formatMoney(selectedBalanceSummary?.current_balance) }}</strong>
                  </div>
                  <div class="balance-metric">
                    <span>{{ t('admin.upstreamAccounts.todayConsumption') }}</span>
                    <strong>{{ formatMoney(selectedBalanceSummary?.today_consumption) }}</strong>
                  </div>
                  <div class="balance-metric">
                    <span>{{ t('admin.upstreamAccounts.amountScale') }}</span>
                    <strong>{{ formatScale(selectedBalanceScale) }}</strong>
                  </div>
                  <div class="balance-metric">
                    <span>{{ t('admin.upstreamAccounts.lastSnapshot') }}</span>
                    <strong class="text-sm">{{ selectedBalanceSummary?.last_snapshot_at ? formatDateTime(selectedBalanceSummary.last_snapshot_at) : '-' }}</strong>
                  </div>
                </div>

                <div class="balance-config-panel">
                  <label class="guard-toggle">
                    <input v-model="balanceSamplerForm.enabled" type="checkbox" class="h-4 w-4 rounded border-gray-300 text-primary-600 focus:ring-primary-500" />
                    <span>{{ t('admin.upstreamAccounts.balanceSamplerAutoRun') }}</span>
                  </label>
                  <label class="guard-interval">
                    <span>{{ t('admin.upstreamAccounts.balanceSamplerIntervalSeconds') }}</span>
                    <input v-model.number="balanceSamplerForm.interval_seconds" type="number" min="60" class="input h-9 w-28" />
                  </label>
                  <label class="guard-interval">
                    <span>{{ t('admin.upstreamAccounts.amountScale') }}</span>
                    <input v-model.number="selectedProviderScaleInput" type="number" min="0.000001" step="0.000001" class="input h-9 w-28" />
                  </label>
                  <button type="button" class="btn btn-secondary" :disabled="savingBalanceSamplerConfig" @click="saveBalanceSamplerConfig">
                    <Icon name="cog" size="sm" class="mr-2" :class="savingBalanceSamplerConfig ? 'animate-spin' : ''" />
                    {{ t('common.save') }}
                  </button>
                  <button type="button" class="btn btn-primary" :disabled="runningBalanceSampleNow" @click="runBalanceSampleNow">
                    <Icon name="play" size="sm" class="mr-2" :class="runningBalanceSampleNow ? 'animate-pulse' : ''" />
                    {{ t('admin.upstreamAccounts.balanceSampleNow') }}
                  </button>
                </div>

                <div class="balance-recharge-panel">
                  <div class="balance-section-title">{{ t('admin.upstreamAccounts.addRecharge') }}</div>
                  <div class="balance-recharge-form">
                    <input v-model.number="rechargeForm.amount" type="number" min="0" step="0.000001" class="input" :placeholder="t('admin.upstreamAccounts.rechargeAmount')" />
                    <input v-model="rechargeForm.note" type="text" class="input" :placeholder="t('admin.upstreamAccounts.rechargeNote')" />
                    <button type="button" class="btn btn-secondary" :disabled="addingRecharge" @click="addBalanceRecharge">
                      <Icon name="plus" size="sm" class="mr-2" />
                      {{ t('common.add') }}
                    </button>
                  </div>
                </div>

                <div>
                  <div class="balance-section-title">{{ t('admin.upstreamAccounts.balanceHistory') }}</div>
                  <div class="max-h-72 overflow-auto rounded-lg border border-gray-200 dark:border-dark-600">
                    <table class="records-table min-w-[760px]">
                      <thead class="bg-gray-50 dark:bg-dark-800">
                        <tr>
                          <th class="px-4 py-2 text-left font-medium">{{ t('admin.upstreamAccounts.balanceDate') }}</th>
                          <th class="px-4 py-2 text-left font-medium">{{ t('admin.upstreamAccounts.openingBalance') }}</th>
                          <th class="px-4 py-2 text-left font-medium">{{ t('admin.upstreamAccounts.rechargeAmount') }}</th>
                          <th class="px-4 py-2 text-left font-medium">{{ t('admin.upstreamAccounts.closingBalance') }}</th>
                          <th class="px-4 py-2 text-left font-medium">{{ t('admin.upstreamAccounts.consumptionAmount') }}</th>
                          <th class="px-4 py-2 text-left font-medium">{{ t('common.status') }}</th>
                        </tr>
                      </thead>
                      <tbody>
                        <tr v-for="row in selectedBalanceRows" :key="`${row.provider_slug}-${row.date}`" class="records-row">
                          <td class="px-4 py-3 font-mono text-gray-600 dark:text-gray-300">{{ row.date }}</td>
                          <td class="px-4 py-3 font-mono">{{ formatMoney(row.opening_balance) }}</td>
                          <td class="px-4 py-3 font-mono">{{ formatMoney(row.recharge_amount) }}</td>
                          <td class="px-4 py-3 font-mono">{{ formatMoney(row.closing_balance) }}</td>
                          <td class="px-4 py-3 font-mono">{{ formatMoney(row.consumption_amount) }}</td>
                          <td class="px-4 py-3">
                            <span :class="['record-status', row.anomaly ? 'record-status-error' : row.complete ? 'record-status-success' : 'record-status-muted']">
                              {{ balanceRowStatus(row) }}
                            </span>
                          </td>
                        </tr>
                        <tr v-if="!selectedBalanceRows.length">
                          <td colspan="6" class="px-4 py-8 text-center text-gray-400">{{ t('admin.upstreamAccounts.noBalanceHistory') }}</td>
                        </tr>
                      </tbody>
                    </table>
                  </div>
                </div>
              </div>
            </div>
          </div>

          <div class="records-panel">
            <div class="records-header">
              <div>
                <h3 class="text-sm font-semibold text-gray-900 dark:text-white">{{ t('admin.upstreamAccounts.syncLogs') }}</h3>
                <span class="text-xs text-gray-500 dark:text-gray-400">{{ t('admin.upstreamAccounts.syncLogsDescription') }}</span>
              </div>
              <div class="records-total">{{ syncLogEntries.length }}</div>
            </div>
            <div class="max-h-80 overflow-auto">
              <table class="records-table min-w-[1080px]">
                <thead class="bg-primary-50/80 dark:bg-primary-950/30">
                  <tr>
                    <th class="px-4 py-2 text-left font-medium">{{ t('admin.upstreamAccounts.logTime') }}</th>
                    <th class="px-4 py-2 text-left font-medium">{{ t('admin.upstreamAccounts.logTriggerSource') }}</th>
                    <th class="px-4 py-2 text-left font-medium">{{ t('admin.upstreamAccounts.logAccount') }}</th>
                    <th class="px-4 py-2 text-left font-medium">{{ t('admin.upstreamAccounts.logUpstream') }}</th>
                    <th class="px-4 py-2 text-left font-medium">{{ t('admin.upstreamAccounts.logRateCompare') }}</th>
                    <th class="px-4 py-2 text-left font-medium">{{ t('admin.upstreamAccounts.logUnboundGroups') }}</th>
                    <th class="px-4 py-2 text-left font-medium">{{ t('admin.upstreamAccounts.logRemainingGroups') }}</th>
                  </tr>
                </thead>
                <tbody class="divide-y divide-gray-100 dark:divide-dark-700">
                  <tr v-for="entry in syncLogEntries" :key="entry.key" class="records-row">
                    <td class="px-4 py-3 text-gray-600 dark:text-gray-300">{{ formatDateTime(entry.created_at) }}</td>
                    <td class="px-4 py-3">
                      <span :class="['trigger-chip', triggerClass(entry.trigger_source)]">
                        {{ upstreamAccountSyncTriggerSourceLabel(entry.trigger_source) }}
                      </span>
                    </td>
                    <td class="px-4 py-3">
                      <div class="table-main-cell min-w-[12rem]">
                        <span class="font-medium text-gray-900 dark:text-white">{{ entry.matched_local_account_name }}</span>
                        <code class="table-tag tag-account">#{{ entry.matched_local_account_id }}</code>
                      </div>
                    </td>
                    <td class="px-4 py-3">
                      <div class="table-main-cell min-w-[14rem]">
                        <span class="font-medium text-gray-900 dark:text-white">{{ entry.upstream_key_name }}</span>
                        <div class="tag-list max-w-[22rem]">
                          <span :class="['table-tag', providerToneClass(entry.provider_slug, 'tag')]">{{ entry.provider_name || entry.provider_slug }}</span>
                          <span class="table-tag tag-group">{{ entry.upstream_group_name }}</span>
                        </div>
                      </div>
                    </td>
                    <td class="px-4 py-3">
                      <div class="rate-compare">
                        <span class="rate-compare-upstream">{{ formatRate(entry.upstream_rate_multiplier) }}</span>
                        <span class="text-gray-400">/</span>
                        <span class="rate-compare-local">{{ formatRate(entry.local_min_rate_multiplier) }}</span>
                      </div>
                    </td>
                    <td class="px-4 py-3">
                      <div class="tag-list">
                        <span v-for="group in entry.unbound_group_names" :key="`${entry.key}-${group}`" class="log-chip log-chip-warning">{{ group }}</span>
                      </div>
                    </td>
                    <td class="px-4 py-3">
                      <div class="tag-list">
                        <span v-if="!entry.remaining_group_ids.length" class="text-xs text-gray-400">-</span>
                        <code v-for="groupID in entry.remaining_group_ids" :key="`${entry.key}-${groupID}`" class="log-chip">#{{ groupID }}</code>
                      </div>
                    </td>
                  </tr>
                  <tr v-if="!syncLogEntries.length">
                    <td colspan="7" class="px-4 py-8 text-center text-gray-400">{{ t('admin.upstreamAccounts.noSyncLogs') }}</td>
                  </tr>
                </tbody>
              </table>
            </div>
          </div>
        </div>
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
  UpstreamAccountSyncItem,
  UpstreamAccountSyncRecord,
  UpstreamAccountSyncResult,
  UpstreamAccountSyncUnbindDetail,
  UpstreamBalanceConsumptionOverview,
  UpstreamBalanceDailyRow,
  UpstreamBalanceProviderSummary,
  UpstreamBalanceSamplerConfig
} from '@/api/admin/upstreamAccountSync'
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

const { t } = useI18n()
const appStore = useAppStore()

const result = ref<UpstreamAccountSyncResult | null>(null)
const loading = ref(false)
const syncing = ref(false)
const loadingRateGuardConfig = ref(false)
const savingRateGuardConfig = ref(false)
const runningRateGuardNow = ref(false)
const savingBalanceSamplerConfig = ref(false)
const runningBalanceSampleNow = ref(false)
const addingRecharge = ref(false)
const loadError = ref('')
const searchQuery = ref('')
const providerFilter = ref('')
const sourceFilter = ref('')
const rateGuardConfig = ref<UpstreamAccountRateGuardConfig | null>(null)
const balanceOverview = ref<UpstreamBalanceConsumptionOverview | null>(null)
const balanceDetailsOpen = ref(false)
const selectedBalanceProviderSlug = ref('')
const selectedProviderScaleInput = ref(1)
const rateGuardForm = ref({
  enabled: false,
  interval_seconds: 3600
})
const balanceSamplerForm = ref({
  enabled: false,
  interval_seconds: 3600,
  provider_amount_scales: {} as Record<string, number>
})
const rechargeForm = ref({
  amount: null as number | null,
  note: ''
})

type UpstreamAccountSyncLogEntry = UpstreamAccountSyncUnbindDetail & {
  created_at: string
  key: string
}

const columns = computed<Column[]>(() => [
  { key: 'source', label: t('admin.upstreamAccounts.columns.source') },
  { key: 'upstream_key_name', label: t('admin.upstreamAccounts.columns.upstreamKey') },
  { key: 'upstream_rate_multiplier', label: t('admin.upstreamAccounts.columns.upstreamRate') },
  { key: 'balance_consumption', label: t('admin.upstreamAccounts.columns.balanceConsumption') },
  { key: 'local_account_name', label: t('admin.upstreamAccounts.columns.localAccount') },
  { key: 'local_group_name', label: t('admin.upstreamAccounts.columns.boundGroups') },
  { key: 'actions', label: t('common.actions') }
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
const balanceSummaries = computed<Record<string, UpstreamBalanceProviderSummary>>(() => balanceOverview.value?.summaries || {})
const balanceRows = computed<UpstreamBalanceDailyRow[]>(() => balanceOverview.value?.rows || [])
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
const selectedBalanceSummary = computed(() => selectedBalanceProviderSlug.value ? balanceSummaries.value[selectedBalanceProviderSlug.value] : undefined)
const selectedBalanceRows = computed(() => balanceRows.value.filter(row => row.provider_slug === selectedBalanceProviderSlug.value))
const selectedBalanceScale = computed(() => {
  const configured = balanceSamplerForm.value.provider_amount_scales[selectedBalanceProviderSlug.value]
  if (Number(configured) > 0) return Number(configured)
  if (Number(selectedBalanceSummary.value?.amount_scale) > 0) return Number(selectedBalanceSummary.value?.amount_scale)
  return 1
})
const selectedBalanceProviderLabel = computed(() => {
  const slug = selectedBalanceProviderSlug.value
  if (!slug) return '-'
  const provider = syncProviders.value.find(item => item.slug === slug)
  const summary = selectedBalanceSummary.value
  return provider?.name || summary?.provider_name || slug
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
    applyRateGuardConfig(config)
    applyBalanceOverview(balance)
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

function applyBalanceOverview(overview: UpstreamBalanceConsumptionOverview) {
  balanceOverview.value = overview
  applyBalanceSamplerConfig(overview.config)
}

function applyBalanceSamplerConfig(config: UpstreamBalanceSamplerConfig) {
  balanceSamplerForm.value = {
    enabled: Boolean(config.enabled),
    interval_seconds: Number(config.interval_seconds) > 0 ? Number(config.interval_seconds) : 3600,
    provider_amount_scales: { ...(config.provider_amount_scales || {}) }
  }
  if (selectedBalanceProviderSlug.value) {
    selectedProviderScaleInput.value = selectedBalanceScale.value
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

async function saveBalanceSamplerConfig() {
  if (!Number.isInteger(balanceSamplerForm.value.interval_seconds) || balanceSamplerForm.value.interval_seconds < 60) {
    appStore.showError(t('admin.upstreamAccounts.invalidBalanceSamplerInterval'))
    return
  }
  if (!selectedBalanceProviderSlug.value) return
  const scale = Number(selectedProviderScaleInput.value)
  if (!Number.isFinite(scale) || scale <= 0) {
    appStore.showError(t('admin.upstreamAccounts.invalidAmountScale'))
    return
  }
  savingBalanceSamplerConfig.value = true
  try {
    const providerAmountScales = {
      ...balanceSamplerForm.value.provider_amount_scales,
      [selectedBalanceProviderSlug.value]: scale
    }
    const base = balanceOverview.value?.config || { enabled: false, interval_seconds: 3600 }
    const config = await adminAPI.upstreamAccountSync.updateBalanceSamplerConfig({
      ...base,
      enabled: balanceSamplerForm.value.enabled,
      interval_seconds: balanceSamplerForm.value.interval_seconds,
      provider_amount_scales: providerAmountScales
    })
    applyBalanceSamplerConfig(config)
    appStore.showSuccess(t('admin.upstreamAccounts.balanceSamplerSaved'))
  } catch (err) {
    appStore.showError(extractApiErrorMessage(err, t('admin.upstreamAccounts.balanceSamplerSaveFailed')))
  } finally {
    savingBalanceSamplerConfig.value = false
  }
}

async function runBalanceSampleNow() {
  runningBalanceSampleNow.value = true
  try {
    const config = await adminAPI.upstreamAccountSync.runBalanceSampleNow()
    applyBalanceSamplerConfig(config)
    const balance = await adminAPI.upstreamAccountSync.getBalanceConsumption(30)
    applyBalanceOverview(balance)
    appStore.showSuccess(t('admin.upstreamAccounts.balanceSampleSuccess'))
  } catch (err) {
    appStore.showError(extractApiErrorMessage(err, t('admin.upstreamAccounts.balanceSampleFailed')))
  } finally {
    runningBalanceSampleNow.value = false
  }
}

async function addBalanceRecharge() {
  if (!selectedBalanceProviderSlug.value) return
  const amount = Number(rechargeForm.value.amount)
  if (!Number.isFinite(amount) || amount <= 0) {
    appStore.showError(t('admin.upstreamAccounts.invalidRechargeAmount'))
    return
  }
  addingRecharge.value = true
  try {
    await adminAPI.upstreamAccountSync.addBalanceRecharge({
      provider_slug: selectedBalanceProviderSlug.value,
      amount,
      amount_scale: selectedBalanceScale.value,
      note: rechargeForm.value.note.trim() || undefined,
      occurred_at: new Date().toISOString()
    })
    rechargeForm.value = { amount: null, note: '' }
    const balance = await adminAPI.upstreamAccountSync.getBalanceConsumption(30)
    applyBalanceOverview(balance)
    appStore.showSuccess(t('admin.upstreamAccounts.rechargeAdded'))
  } catch (err) {
    appStore.showError(extractApiErrorMessage(err, t('admin.upstreamAccounts.rechargeAddFailed')))
  } finally {
    addingRecharge.value = false
  }
}

function openBalanceDetails(providerSlug: string) {
  selectedBalanceProviderSlug.value = providerSlug
  selectedProviderScaleInput.value = selectedBalanceScale.value
  balanceDetailsOpen.value = true
}

function closeBalanceDetails() {
  balanceDetailsOpen.value = false
}

function balanceSummaryFor(providerSlug: string | undefined) {
  if (!providerSlug) return undefined
  return balanceSummaries.value[providerSlug]
}

function balanceRowStatus(row: UpstreamBalanceDailyRow) {
  if (row.anomaly) return t('admin.upstreamAccounts.balanceAnomaly')
  if (row.complete) return t('admin.upstreamAccounts.balanceComplete')
  return t('admin.upstreamAccounts.balanceIncomplete')
}

function formatRate(value: number | undefined) {
  const n = Number(value)
  return Number.isFinite(n) ? `${n.toFixed(2)}x` : '-'
}

function formatScale(value: number | undefined) {
  const n = Number(value)
  return Number.isFinite(n) && n > 0 ? `${n.toFixed(6).replace(/0+$/, '').replace(/\.$/, '')}x` : '-'
}

function formatMoney(value: number | undefined) {
  const n = Number(value)
  if (!Number.isFinite(n)) return '-'
  return n.toLocaleString(undefined, {
    minimumFractionDigits: 2,
    maximumFractionDigits: 6
  })
}

function accountCardClass(row: UpstreamAccountSyncItem) {
  if (row.conflict_accounts?.length || row.conflict_account_ids?.length) return 'account-card-conflict'
  if (!row.matched_account_id) return 'account-card-new'
  if (row.rate_violation) return 'account-card-warning'
  return 'account-card-matched'
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

function groupChipClass(rateViolation: boolean, index: number) {
  if (rateViolation) return 'group-chip-warning'
  const tones = ['group-chip-blue', 'group-chip-emerald', 'group-chip-violet', 'group-chip-cyan']
  return tones[index % tones.length]
}

function rateToneClass(value: number | undefined) {
  const n = Number(value)
  if (!Number.isFinite(n)) return ''
  if (n >= 2) return 'rate-purple'
  if (n > 1) return 'rate-primary'
  return 'rate-success'
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

onMounted(reload)
</script>

<style scoped>
.accounts-toolbar {
  @apply grid gap-3 xl:grid-cols-[minmax(14rem,18rem)_1fr_auto];
}

.provider-panel {
  @apply flex min-h-16 items-center justify-between gap-3 rounded-lg border border-gray-200 bg-white px-4 py-3 dark:border-dark-600 dark:bg-dark-800/40;
}

.meta-label {
  @apply text-xs font-medium text-gray-500 dark:text-gray-400;
}

.provider-count {
  @apply flex h-9 w-9 shrink-0 items-center justify-center rounded-md bg-gray-100 font-mono text-sm font-semibold text-gray-700 dark:bg-dark-700 dark:text-gray-200;
}

.stats-strip {
  @apply grid grid-cols-2 gap-2 sm:grid-cols-4;
}

.stat-tile {
  @apply flex min-h-16 flex-col justify-center rounded-lg border border-gray-200 bg-white px-3 py-2 text-sm text-gray-600 dark:border-dark-600 dark:bg-dark-800/40 dark:text-gray-300;
}

.stat-tile span {
  @apply text-xs font-medium text-gray-500 dark:text-gray-400;
}

.stat-tile strong {
  @apply mt-1 font-mono text-xl text-gray-950 dark:text-white;
}

.stat-tile-create {
  @apply border-sky-200 bg-sky-50/60 dark:border-sky-800/50 dark:bg-sky-950/20;
}

.stat-tile-update {
  @apply border-emerald-200 bg-emerald-50/60 dark:border-emerald-800/50 dark:bg-emerald-950/20;
}

.stat-tile-warning {
  @apply border-amber-200 bg-amber-50 text-amber-800 dark:border-amber-700/40 dark:bg-amber-900/20 dark:text-amber-200;
}

.accounts-actions {
  @apply flex flex-wrap items-center justify-end gap-2 xl:min-h-16;
}

.filter-row {
  @apply mt-3 grid gap-3 md:grid-cols-[minmax(14rem,1fr)_12rem_11rem_11rem_auto];
}

.filter-search {
  @apply w-full;
}

.filter-select {
  @apply w-full;
}

.filtered-count {
  @apply flex h-10 items-center justify-between gap-3 rounded-lg border border-gray-200 px-3 text-sm text-gray-600 dark:border-dark-600 dark:text-gray-300;
}

.filtered-count strong {
  @apply font-mono text-base text-gray-900 dark:text-white;
}

.rate-guard-panel {
  @apply mt-3 grid items-center gap-3 rounded-lg border border-gray-200 bg-white px-4 py-3 dark:border-dark-600 dark:bg-dark-800/40 lg:grid-cols-[minmax(16rem,1fr)_auto_auto_auto_auto];
}

.guard-toggle {
  @apply inline-flex items-center gap-2 text-sm font-medium text-gray-700 dark:text-gray-200;
}

.guard-interval {
  @apply flex items-center gap-2 text-sm text-gray-600 dark:text-gray-300;
}

.accounts-table-content {
  @apply flex h-full min-h-0 flex-col overflow-y-auto;
}

.warning-banner {
  @apply mb-4 rounded-lg border border-amber-200 bg-amber-50 p-3 text-sm text-amber-800 dark:border-amber-700/40 dark:bg-amber-900/20 dark:text-amber-200;
}

.accounts-table-primary {
  @apply flex flex-none flex-col overflow-hidden;
  height: clamp(28rem, 52vh, 42rem);
  min-height: 28rem;
}

.accounts-table-primary :deep(.table-wrapper) {
  @apply min-h-0;
}

.accounts-table-primary :deep(tbody tr) {
  @apply transition-colors;
}

.records-panel {
  @apply mt-4 overflow-hidden rounded-lg border border-gray-200 bg-white dark:border-dark-600 dark:bg-dark-800/30;
}

.records-header {
  @apply flex items-center justify-between gap-3 border-b border-gray-200 px-4 py-3 dark:border-dark-600;
}

.records-total {
  @apply flex h-8 min-w-8 items-center justify-center rounded-md bg-gray-100 px-2 font-mono text-sm font-semibold text-gray-700 dark:bg-dark-700 dark:text-gray-200;
}

.records-row {
  @apply align-top;
}

.records-table {
  @apply w-full divide-y divide-gray-100 text-sm dark:divide-dark-700;
}

.records-table tbody {
  @apply divide-y divide-gray-100 dark:divide-dark-700;
}

.records-table tbody tr {
  @apply transition-colors hover:bg-gray-50 dark:hover:bg-dark-700/40;
}

.table-main-cell {
  @apply flex flex-col gap-1 leading-tight;
}

.source-card,
.key-card,
.account-card {
  @apply flex flex-col gap-2 rounded-md border px-3 py-2 leading-tight;
}

.source-home-link {
  @apply inline-flex shrink-0 flex-col items-center gap-0.5 rounded-md p-1 text-xs font-medium text-gray-500 transition-colors hover:bg-white/70 hover:text-sky-600 dark:text-gray-400 dark:hover:bg-dark-700/70 dark:hover:text-sky-300;
}

.account-card-new {
  @apply border-sky-200 bg-sky-50/70 dark:border-sky-800/50 dark:bg-sky-950/25;
}

.account-card-matched {
  @apply border-emerald-200 bg-emerald-50/70 dark:border-emerald-800/50 dark:bg-emerald-950/25;
}

.account-card-warning {
  @apply border-amber-200 bg-amber-50/80 dark:border-amber-700/40 dark:bg-amber-950/25;
}

.account-card-conflict {
  @apply border-violet-200 bg-violet-50/70 dark:border-violet-800/50 dark:bg-violet-950/25;
}

.source-card-sky {
  @apply border-sky-200 bg-sky-50/70 dark:border-sky-800/50 dark:bg-sky-950/25;
}

.source-card-emerald {
  @apply border-emerald-200 bg-emerald-50/70 dark:border-emerald-800/50 dark:bg-emerald-950/25;
}

.source-card-violet {
  @apply border-violet-200 bg-violet-50/70 dark:border-violet-800/50 dark:bg-violet-950/25;
}

.source-card-cyan {
  @apply border-cyan-200 bg-cyan-50/70 dark:border-cyan-800/50 dark:bg-cyan-950/25;
}

.source-card-rose {
  @apply border-rose-200 bg-rose-50/70 dark:border-rose-800/50 dark:bg-rose-950/25;
}

.source-card-amber {
  @apply border-amber-200 bg-amber-50/70 dark:border-amber-800/50 dark:bg-amber-950/25;
}

.source-card-indigo {
  @apply border-indigo-200 bg-indigo-50/70 dark:border-indigo-800/50 dark:bg-indigo-950/25;
}

.source-card-teal {
  @apply border-teal-200 bg-teal-50/70 dark:border-teal-800/50 dark:bg-teal-950/25;
}

.key-card {
  @apply border-primary-100 bg-primary-50/60 dark:border-primary-800/40 dark:bg-primary-950/20;
}

.table-tag {
  @apply inline-flex max-w-full items-center gap-1 truncate rounded-md px-2 py-1 text-xs font-semibold ring-1;
}

.tag-provider-sky {
  @apply bg-sky-50 text-sky-700 ring-sky-200 dark:bg-sky-950/40 dark:text-sky-300 dark:ring-sky-800/60;
}

.tag-provider-emerald {
  @apply bg-emerald-50 text-emerald-700 ring-emerald-200 dark:bg-emerald-950/40 dark:text-emerald-300 dark:ring-emerald-800/60;
}

.tag-provider-violet {
  @apply bg-violet-50 text-violet-700 ring-violet-200 dark:bg-violet-950/40 dark:text-violet-300 dark:ring-violet-800/60;
}

.tag-provider-cyan {
  @apply bg-cyan-50 text-cyan-700 ring-cyan-200 dark:bg-cyan-950/40 dark:text-cyan-300 dark:ring-cyan-800/60;
}

.tag-provider-rose {
  @apply bg-rose-50 text-rose-700 ring-rose-200 dark:bg-rose-950/40 dark:text-rose-300 dark:ring-rose-800/60;
}

.tag-provider-amber {
  @apply bg-amber-50 text-amber-700 ring-amber-200 dark:bg-amber-950/40 dark:text-amber-300 dark:ring-amber-800/60;
}

.tag-provider-indigo {
  @apply bg-indigo-50 text-indigo-700 ring-indigo-200 dark:bg-indigo-950/40 dark:text-indigo-300 dark:ring-indigo-800/60;
}

.tag-provider-teal {
  @apply bg-teal-50 text-teal-700 ring-teal-200 dark:bg-teal-950/40 dark:text-teal-300 dark:ring-teal-800/60;
}

.tag-group {
  @apply bg-primary-50 text-primary-700 ring-primary-200 dark:bg-primary-950/40 dark:text-primary-300 dark:ring-primary-800/60;
}

.tag-warning {
  @apply bg-amber-50 text-amber-700 ring-amber-200 dark:bg-amber-950/40 dark:text-amber-300 dark:ring-amber-800/60;
}

.tag-account {
  @apply bg-indigo-50 font-mono text-indigo-700 ring-indigo-200 dark:bg-indigo-950/40 dark:text-indigo-300 dark:ring-indigo-800/60;
}

.trigger-chip {
  @apply inline-flex rounded-full px-2.5 py-1 text-xs font-bold ring-1;
}

.rate-value {
  @apply inline-flex rounded-md px-2 py-1 font-mono text-sm font-semibold ring-1;
}

.rate-success {
  @apply bg-emerald-50 text-emerald-700 ring-emerald-200 dark:bg-emerald-950/30 dark:text-emerald-300 dark:ring-emerald-800/60;
}

.rate-primary {
  @apply bg-primary-50 text-primary-700 ring-primary-200 dark:bg-primary-950/30 dark:text-primary-300 dark:ring-primary-800/60;
}

.rate-purple {
  @apply bg-violet-50 text-violet-700 ring-violet-200 dark:bg-violet-950/30 dark:text-violet-300 dark:ring-violet-800/60;
}

.record-status {
  @apply inline-flex rounded-md px-2 py-1 text-xs font-medium;
}

.record-status-success {
  @apply bg-emerald-50 text-emerald-700 dark:bg-emerald-900/20 dark:text-emerald-200;
}

.record-status-error {
  @apply bg-red-50 text-red-700 dark:bg-red-900/20 dark:text-red-200;
}

.record-status-muted {
  @apply bg-gray-100 text-gray-600 dark:bg-dark-700 dark:text-gray-300;
}

.balance-cost-cell {
  @apply min-w-[10rem] leading-tight;
}

.balance-dialog {
  @apply flex max-h-[88vh] w-full max-w-5xl flex-col overflow-hidden rounded-lg bg-white shadow-xl dark:bg-dark-800;
}

.balance-dialog-header {
  @apply flex items-start justify-between gap-3 border-b border-gray-200 px-5 py-4 dark:border-dark-600;
}

.balance-dialog-body {
  @apply space-y-4 overflow-y-auto p-5;
}

.balance-summary-grid {
  @apply grid gap-3 sm:grid-cols-2 lg:grid-cols-4;
}

.balance-metric {
  @apply rounded-lg border border-gray-200 bg-gray-50 px-3 py-3 dark:border-dark-600 dark:bg-dark-900/40;
}

.balance-metric span {
  @apply text-xs font-medium text-gray-500 dark:text-gray-400;
}

.balance-metric strong {
  @apply mt-1 block font-mono text-lg text-gray-950 dark:text-white;
}

.balance-config-panel,
.balance-recharge-panel {
  @apply rounded-lg border border-gray-200 bg-white p-3 dark:border-dark-600 dark:bg-dark-800/50;
}

.balance-config-panel {
  @apply grid items-center gap-3 lg:grid-cols-[auto_auto_auto_auto_auto];
}

.balance-recharge-form {
  @apply mt-3 grid gap-3 md:grid-cols-[minmax(10rem,12rem)_1fr_auto];
}

.balance-section-title {
  @apply text-sm font-semibold text-gray-900 dark:text-white;
}

.tag-success {
  @apply bg-emerald-50 text-emerald-700 ring-emerald-200 dark:bg-emerald-950/40 dark:text-emerald-300 dark:ring-emerald-800/60;
}

.tag-muted {
  @apply bg-gray-100 text-gray-600 ring-gray-200 dark:bg-dark-700 dark:text-gray-300 dark:ring-dark-600;
}

.rate-compare {
  @apply inline-flex items-center gap-2 rounded-md bg-slate-100 px-2 py-1 font-mono text-sm font-semibold ring-1 ring-slate-200 dark:bg-slate-900/60 dark:ring-slate-700;
}

.rate-compare-upstream {
  @apply text-amber-700 dark:text-amber-300;
}

.rate-compare-local {
  @apply text-emerald-700 dark:text-emerald-300;
}

.tag-list {
  @apply flex max-w-[18rem] flex-wrap gap-1.5;
}

.group-chip {
  @apply inline-flex items-center gap-1 rounded-md px-2 py-1 text-xs font-semibold ring-1;
}

.group-chip-blue {
  @apply bg-sky-50 text-sky-700 ring-sky-200 dark:bg-sky-950/40 dark:text-sky-300 dark:ring-sky-800/60;
}

.group-chip-emerald {
  @apply bg-emerald-50 text-emerald-700 ring-emerald-200 dark:bg-emerald-950/40 dark:text-emerald-300 dark:ring-emerald-800/60;
}

.group-chip-violet {
  @apply bg-violet-50 text-violet-700 ring-violet-200 dark:bg-violet-950/40 dark:text-violet-300 dark:ring-violet-800/60;
}

.group-chip-cyan {
  @apply bg-cyan-50 text-cyan-700 ring-cyan-200 dark:bg-cyan-950/40 dark:text-cyan-300 dark:ring-cyan-800/60;
}

.group-chip-warning {
  @apply bg-amber-50 text-amber-700 ring-1 ring-amber-200 dark:bg-amber-900/20 dark:text-amber-200 dark:ring-amber-700/40;
}

.trigger-sync {
  @apply bg-primary-50 text-primary-700 ring-primary-200 dark:bg-primary-950/40 dark:text-primary-300 dark:ring-primary-800/60;
}

.trigger-scheduled {
  @apply bg-violet-50 text-violet-700 ring-violet-200 dark:bg-violet-950/40 dark:text-violet-300 dark:ring-violet-800/60;
}

.trigger-guard {
  @apply bg-amber-50 text-amber-700 ring-amber-200 dark:bg-amber-950/40 dark:text-amber-300 dark:ring-amber-800/60;
}

.log-chip {
  @apply inline-flex items-center rounded-md bg-gray-100 px-2 py-1 text-xs font-medium text-gray-700 dark:bg-dark-700 dark:text-gray-200;
}

.log-chip-warning {
  @apply bg-amber-50 text-amber-700 dark:bg-amber-900/20 dark:text-amber-200;
}

@media (max-width: 1023px) {
  .accounts-table-content {
    @apply h-auto overflow-visible;
  }

  .accounts-table-primary {
    @apply h-auto min-h-0 overflow-visible;
  }
}
</style>

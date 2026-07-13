<template>
  <AppLayout>
    <div class="upstream-providers-page">
      <TablePageLayout>
        <template #filters>
        <div class="upstream-toolbar" :class="{ 'upstream-filters-expanded': showProviderAdvancedFilters }">
          <div class="upstream-toolbar-left">
            <div class="upstream-toolbar-title">{{ t('admin.upstreamProviders.title') }}</div>
            <div class="upstream-toolbar-actions">
              <button type="button" class="btn btn-primary upstream-toolbar-action" @click="openCreateDialog">
                {{ t('admin.upstreamProviders.createProvider') }}
              </button>
              <button
                type="button"
                class="btn btn-secondary upstream-toolbar-action"
                :disabled="loading"
                :title="t('common.refresh')"
                @click="reload"
              >
                {{ t('common.refresh') }}
              </button>
              <button
                type="button"
                class="btn btn-secondary upstream-toolbar-action upstream-sample-action"
                :disabled="runningBalanceSampleNow"
                :title="t('admin.upstreamProviders.balanceSampleNow')"
                @click="runBalanceSampleNow"
              >
                <Icon name="play" size="sm" :class="runningBalanceSampleNow ? 'animate-pulse' : ''" />
                <span>{{ t('admin.upstreamProviders.balanceSampleNow') }}</span>
              </button>
              <button
                type="button"
                class="btn btn-secondary upstream-toolbar-action upstream-sampler-settings-action"
                :title="t('admin.upstreamProviders.balanceSamplerSettings')"
                @click="openBalanceSamplerDialog"
              >
                <Icon name="cog" size="sm" />
                <span>{{ t('admin.upstreamProviders.balanceSamplerSettings') }}</span>
              </button>
              <button
                type="button"
                class="btn btn-secondary upstream-toolbar-action upstream-health-run-action"
                :disabled="runningHealthGuardNow"
                :title="t('admin.upstreamProviders.healthGuardRunNow')"
                @click="runHealthGuardNow"
              >
                <Icon name="shield" size="sm" :class="runningHealthGuardNow ? 'animate-pulse' : ''" />
                <span>{{ t('admin.upstreamProviders.healthGuardRunNow') }}</span>
              </button>
              <button
                type="button"
                class="btn btn-secondary upstream-toolbar-action upstream-health-settings-action"
                :title="t('admin.upstreamProviders.healthGuardSettings')"
                @click="openHealthGuardDialog"
              >
                <Icon name="cog" size="sm" />
                <span>{{ t('admin.upstreamProviders.healthGuardSettings') }}</span>
              </button>
            </div>
          </div>

          <div class="upstream-toolbar-filters">
            <div class="upstream-search-row">
              <div class="relative min-w-0 flex-1 sm:w-64">
                <Icon
                  name="search"
                  size="sm"
                  class="absolute left-3 top-1/2 -translate-y-1/2 text-gray-400 dark:text-gray-500"
                />
                <input
                  v-model="searchQuery"
                  type="text"
                  class="input upstream-compact-input pl-9"
                  :placeholder="t('admin.upstreamProviders.searchPlaceholder')"
                />
              </div>
              <button
                type="button"
                class="upstream-filter-toggle"
                :aria-expanded="showProviderAdvancedFilters"
                @click="showProviderAdvancedFilters = !showProviderAdvancedFilters"
              >
                <Icon name="filter" size="sm" />
                <span>{{ t('admin.upstreamProviders.mobileFilterToggle') }}</span>
                <strong v-if="activeProviderFilterCount">{{ activeProviderFilterCount }}</strong>
                <Icon :name="showProviderAdvancedFilters ? 'chevronUp' : 'chevronDown'" size="sm" />
              </button>
            </div>

            <div class="upstream-filter-controls" :class="{ 'is-open': showProviderAdvancedFilters }">
              <Select
                v-model="typeFilter"
                class="upstream-filter-select"
                :options="typeFilterOptions"
                :searchable="false"
                :placeholder="t('admin.upstreamProviders.type')"
              />

              <Select
                v-model="enabledFilter"
                class="upstream-filter-select"
                :options="enabledFilterOptions"
                :searchable="false"
                :placeholder="t('common.status')"
              />
            </div>
          </div>

          <nav class="upstream-quick-tags" aria-label="quick filters">
            <button
              v-for="option in quickProviderFilterOptions"
              :key="option.key"
              type="button"
              :class="['upstream-quick-tag', { active: activeQuickProviderFilter === option.key }, option.tone ? `upstream-quick-tag-${option.tone}` : '']"
              @click="activeQuickProviderFilter = option.key"
            >
              <span>{{ option.label }}</span>
              <strong>{{ option.count }}</strong>
            </button>
          </nav>

          <div class="upstream-toolbar-right">
            <div class="upstream-total">
              <span>{{ t('admin.upstreamProviders.totalBalance') }}</span>
              <strong>{{ formatTotalMoney(totalProviderBalance) }} 元</strong>
            </div>
            <div class="upstream-total">
              <span>{{ t('admin.upstreamProviders.todayConsumption') }}</span>
              <strong class="is-cost">{{ formatTotalMoney(totalTodayConsumption) }} 元</strong>
            </div>
            <div class="relative">
              <button
                type="button"
                class="column-settings-button"
                :title="t('admin.upstreamProviders.columnSettings')"
                @click="showColumnSettings = !showColumnSettings"
              >
                <Icon name="cog" size="sm" />
              </button>
              <div
                v-if="showColumnSettings"
                class="column-settings-panel"
              >
                <label
                  v-for="option in optionalColumnOptions"
                  :key="option.key"
                  class="column-settings-option"
                >
                  <input
                    v-model="visibleOptionalColumns"
                    type="checkbox"
                    class="rounded border-gray-300 text-primary-600 focus:ring-primary-500"
                    :value="option.key"
                  />
                  <span>{{ option.label }}</span>
                </label>
              </div>
            </div>
          </div>
        </div>
        </template>

        <template #table>
        <DataTable
          :columns="columns"
          :data="filteredProviders"
          :loading="loading"
          row-key="slug"
          :row-class="providerRowClass"
          :is-row-detail-visible="isProviderDetailVisible"
          :estimate-row-height="92"
        >
          <template #cell-homepage="{ row }">
            <div class="homepage-control-cell">
              <button
                type="button"
                class="expand-toggle"
                :title="isExpanded(row.slug) ? t('common.collapse') : t('common.expand')"
                :aria-expanded="isExpanded(row.slug)"
                @click="toggleExpanded(row.slug)"
              >
                <Icon name="chevronRight" size="sm" :class="['expand-toggle-icon', isExpanded(row.slug) && 'is-expanded']" />
              </button>
              <a
                v-if="row.base_url"
                :href="row.base_url"
                target="_blank"
                rel="noopener noreferrer"
                class="homepage-button"
                :title="t('admin.upstreamProviders.openHomepage')"
              >
                <Icon name="home" size="sm" />
                <span>{{ t('admin.upstreamProviders.homepageShort') }}</span>
              </a>
              <span v-else class="text-xs text-gray-400">-</span>
            </div>
          </template>

          <template #cell-prefix="{ row }">
            <span class="prefix-value">{{ row.account_name_prefix || '-' }}</span>
          </template>

          <template #cell-sort_order="{ row }">
            <span class="sort-order-value">{{ formatSortOrder(row.sort_order) }}</span>
          </template>

          <template #cell-rate_scale="{ row }">
            <span class="numeric-value">{{ formatRateScale(row.account_rate_multiplier_scale) }}</span>
          </template>

          <template #cell-balance="{ row }">
            <div class="numeric-cell">
              <span
                v-if="providerBalances[row.slug]"
                :class="['numeric-value', 'numeric-balance', isLowBalance(providerBalances[row.slug].balance) && 'numeric-alert']"
                :title="t('admin.upstreamProviders.balance')"
              >
                {{ formatBalance(providerBalances[row.slug].balance) }}
              </span>
              <span v-else class="numeric-muted">-</span>
              <button
                type="button"
                class="balance-action-button"
                :disabled="balanceLoadingSlugs.has(row.slug)"
                :title="t('admin.upstreamProviders.fetchBalance')"
                @click="fetchProviderBalance(row)"
              >
                <Icon name="dollar" size="sm" :class="balanceLoadingSlugs.has(row.slug) ? 'animate-pulse' : ''" />
              </button>
            </div>
          </template>

          <template #cell-today_consumption="{ row }">
            <div class="numeric-cell">
              <span class="numeric-value numeric-cost">
                {{ formatMoney(todayConsumptionForProvider(row.slug)) }}
              </span>
              <button
                type="button"
                class="balance-action-button balance-more-button"
                :title="t('common.more')"
                @click="openBalanceDetails(row.slug)"
              >
                <Icon name="more" size="sm" />
              </button>
            </div>
          </template>

          <template #row-detail="{ row }">
            <div v-if="isExpanded(row.slug)" class="provider-detail-panel">
              <div class="detail-column">
                <div class="detail-title">{{ t('admin.upstreamProviders.baseUrl') }}</div>
                <button
                  type="button"
                  class="copyable-text detail-copy"
                  :title="copyTitle(row.base_url)"
                  @click="copyValue(row.base_url)"
                >
                  <code>{{ row.base_url || '-' }}</code>
                  <span class="copy-hint">{{ copyHint(row.base_url) }}</span>
                </button>
              </div>
              <div class="detail-column">
                <div class="detail-title">{{ t('admin.upstreamProviders.columns.auth') }}</div>
                <button
                  type="button"
                  class="copyable-text detail-copy"
                  :title="copyTitle(accountIdentity(row))"
                  @click="copyValue(accountIdentity(row))"
                >
                  <code>{{ accountIdentity(row) || '-' }}</code>
                  <span class="copy-hint">{{ copyHint(accountIdentity(row)) }}</span>
                </button>
                <span :class="['password-state-tag', row.password_configured ? 'password-state-ok' : 'password-state-muted']">
                  {{ row.password_configured ? t('admin.upstreamProviders.passwordConfigured') : t('admin.upstreamProviders.passwordNotConfigured') }}
                </span>
              </div>
              <div class="detail-column detail-column-wide">
                <div class="detail-title">{{ t('admin.upstreamProviders.columns.endpoints') }}</div>
                <div class="detail-endpoint-list">
                  <button
                    v-for="endpoint in endpointOptions(row)"
                    :key="endpoint.key"
                    type="button"
                    class="copyable-text detail-endpoint"
                    :title="copyTitle(endpoint.value)"
                    @click="copyValue(endpoint.value)"
                  >
                    <span>{{ endpoint.label }}</span>
                    <code>{{ endpoint.value || '-' }}</code>
                    <span class="copy-hint">{{ copyHint(endpoint.value) }}</span>
                  </button>
                </div>
              </div>
            </div>
          </template>

          <template #cell-name="{ row }">
            <div class="provider-name-card">
              <div class="provider-title-line">
                <span class="provider-name">{{ row.name }}</span>
                <span class="provider-type-tag" :class="providerTypeClass(row.type)">{{ providerTypeLabel(row.type) }}</span>
                <span v-if="row.is_default" class="provider-default-tag">
                  {{ t('admin.upstreamProviders.defaultProvider') }}
                </span>
              </div>
              <button
                type="button"
                class="copyable-text provider-inline-url"
                :title="copyTitle(row.base_url)"
                @click="copyValue(row.base_url)"
              >
                <span>{{ row.base_url || '-' }}</span>
                <span class="copy-hint">{{ copyHint(row.base_url) }}</span>
              </button>
            </div>
          </template>

          <template #cell-enabled="{ row }">
            <div class="provider-enabled-cell">
              <Toggle
                :model-value="row.enabled"
                @update:model-value="toggleProviderEnabled(row, $event)"
              />
              <span :class="['provider-enabled-text', row.enabled ? 'is-enabled' : 'is-disabled']">
                {{ row.enabled ? t('common.enabled') : t('common.disabled') }}
              </span>
            </div>
          </template>

          <template #cell-base_url="{ value }">
            <button
              type="button"
              class="copyable-text provider-url-tag"
              :title="copyTitle(value)"
              @click="copyValue(value)"
            >
              <code>{{ value || '-' }}</code>
              <span class="copy-hint">{{ copyHint(value) }}</span>
            </button>
          </template>

          <template #cell-auth="{ row }">
            <div class="tag-list max-w-[14rem]">
              <button
                v-if="row.username || row.email"
                type="button"
                class="copyable-text info-tag tag-auth"
                :title="copyTitle(row.username || row.email)"
                @click="copyValue(row.username || row.email)"
              >
                {{ row.username || row.email }}
                <span class="copy-hint">{{ copyHint(row.username || row.email) }}</span>
              </button>
              <span v-else class="info-tag tag-muted">-</span>
              <span v-if="row.password_configured" class="info-tag tag-success">
                {{ t('admin.upstreamProviders.passwordConfigured') }}
              </span>
            </div>
          </template>

          <template #cell-actions="{ row }">
            <div class="action-button-group">
              <button
                v-if="!row.is_default"
                type="button"
                class="action-button"
                :disabled="defaultingSlug === row.slug"
                :title="t('admin.upstreamProviders.setDefault')"
                @click="setDefaultProvider(row)"
              >
                <Icon name="badge" size="sm" :class="defaultingSlug === row.slug ? 'animate-pulse' : ''" />
                <span>{{ t('admin.upstreamProviders.setDefaultShort') }}</span>
              </button>
              <button
                type="button"
                class="action-button"
                :disabled="testingSlugs.has(row.slug)"
                :title="t('admin.upstreamProviders.testProvider')"
                @click="testSavedProvider(row)"
              >
                <Icon name="play" size="sm" :class="testingSlugs.has(row.slug) ? 'animate-pulse' : ''" />
                <span>{{ t('admin.upstreamProviders.testShort') }}</span>
              </button>
              <button
                type="button"
                class="action-button"
                :disabled="keysLoadingSlug === row.slug"
                :title="t('admin.upstreamProviders.fetchKeys')"
                @click="openKeysDialog(row)"
              >
                <Icon name="key" size="sm" :class="keysLoadingSlug === row.slug ? 'animate-pulse' : ''" />
                <span>{{ t('admin.upstreamProviders.keysShort') }}</span>
              </button>
              <button
                type="button"
                class="action-button"
                :title="t('common.edit')"
                @click="openEditDialog(row)"
              >
                <Icon name="edit" size="sm" />
                <span>{{ t('common.edit') }}</span>
              </button>
              <button
                type="button"
                class="action-button action-danger"
                :title="t('common.delete')"
                @click="openDeleteDialog(row)"
              >
                <Icon name="trash" size="sm" />
                <span>{{ t('common.delete') }}</span>
              </button>
              <button
                type="button"
                class="action-button provider-mobile-detail-toggle"
                :title="isExpanded(row.slug) ? t('admin.upstreamProviders.hideMobileDetails') : t('admin.upstreamProviders.showMobileDetails')"
                @click="toggleExpanded(row.slug)"
              >
                <Icon :name="isExpanded(row.slug) ? 'chevronUp' : 'chevronDown'" size="sm" />
                <span>
                  {{ isExpanded(row.slug)
                    ? t('admin.upstreamProviders.hideMobileDetails')
                    : t('admin.upstreamProviders.showMobileDetails') }}
                </span>
              </button>
            </div>
          </template>

          <template #empty>
            <EmptyState
              :title="t('admin.upstreamProviders.emptyTitle')"
              :description="t('admin.upstreamProviders.emptyDescription')"
              :action-text="t('admin.upstreamProviders.createProvider')"
              @action="openCreateDialog"
            />
          </template>
          </DataTable>
        </template>
      </TablePageLayout>

      <section data-test="provider-balance-charts-section" class="upstream-balance-charts-section">
        <UpstreamBalanceCharts
          :overview="balanceOverview"
          :loading="loading"
          :days="30"
        />
      </section>
    </div>

    <UpstreamBalanceSamplerDialog
      :show="showBalanceSamplerDialog"
      :enabled="balanceSamplerForm.enabled"
      :interval-seconds="balanceSamplerForm.interval_seconds"
      :provider-amount-scales="balanceSamplerForm.provider_amount_scales"
      :providers="providers"
      :default-scales="balanceSamplerDefaultScales"
      :saving="savingBalanceSamplerConfig"
      @close="closeBalanceSamplerDialog"
      @save="saveBalanceSamplerConfig"
      @update:enabled="balanceSamplerForm.enabled = $event"
      @update:interval-seconds="balanceSamplerForm.interval_seconds = $event"
      @update:provider-scale="updateBalanceSamplerProviderScale"
    />

    <BaseDialog
      :show="showHealthGuardDialog"
      :title="t('admin.upstreamProviders.healthGuardSettings')"
      width="extra-wide"
      @close="closeHealthGuardDialog"
    >
      <div class="health-guard-dialog">
        <div class="health-guard-status-panel">
          <label class="health-guard-toggle">
            <input
              v-model="healthGuardForm.enabled"
              type="checkbox"
              class="rounded border-gray-300 text-primary-600 focus:ring-primary-500"
            />
            <span>
              <strong>{{ t('admin.upstreamProviders.healthGuardAutoRun') }}</strong>
              <small>{{ t('admin.upstreamProviders.healthGuardScopeHint') }}</small>
            </span>
          </label>

          <div class="health-guard-run-state">
            <span>{{ t('admin.upstreamProviders.healthGuardLastRun') }}</span>
            <strong>{{ healthGuardLastRunText }}</strong>
            <em
              v-if="healthGuardConfig?.last_run_status"
              :class="['record-status', healthGuardConfig.last_run_status === 'failed' ? 'record-status-error' : 'record-status-success']"
            >
              {{ healthGuardStatusLabel(healthGuardConfig.last_run_status) }}
            </em>
          </div>

          <button
            type="button"
            class="btn btn-secondary health-guard-run-button"
            :disabled="runningHealthGuardNow || loadingHealthGuard"
            @click="runHealthGuardNow"
          >
            <Icon name="play" size="sm" :class="runningHealthGuardNow ? 'animate-pulse' : ''" />
            {{ t('admin.upstreamProviders.healthGuardRunNow') }}
          </button>
        </div>

        <div class="health-guard-content-grid">
          <div class="health-guard-config-column">
            <section class="health-guard-config-section" :class="{ 'is-collapsed': !healthGuardConfigExpanded }">
              <button
                type="button"
                class="health-guard-section-toggle"
                :aria-expanded="healthGuardConfigExpanded"
                data-test="health-guard-config-toggle"
                @click="healthGuardConfigExpanded = !healthGuardConfigExpanded"
              >
                <span>
                  <strong>{{ t('admin.upstreamProviders.healthGuardConfigSection') }}</strong>
                  <small>{{ healthGuardConfigSummary }}</small>
                </span>
                <Icon name="chevronDown" size="sm" :class="{ 'rotate-180': healthGuardConfigExpanded }" />
              </button>

              <div v-show="healthGuardConfigExpanded" class="health-guard-config-body">
                <UpstreamHealthGuardPolicyFields
                  :values="healthGuardForm"
                  :ignored-summary-text="healthGuardIgnoredSummaryText"
                  :ignored-input-invalid="healthGuardIgnoredInputInvalid"
                  @update:field="updateHealthGuardPolicyField"
                  @manage-ignored="openHealthGuardIgnoredDialog"
                />

                <div class="health-guard-platform-panel">
                  <div class="balance-section-title">{{ t('admin.upstreamProviders.healthGuardPlatformModels') }}</div>
                  <div class="health-guard-platform-list">
                    <label
                      v-for="platform in healthGuardPlatformOptions"
                      :key="platform.value"
                      class="health-guard-platform-row"
                    >
                      <span class="health-guard-platform-name">
                        <strong>{{ platform.label }}</strong>
                        <small>{{ t('admin.upstreamProviders.healthGuardPlatformHint') }}</small>
                      </span>
                      <input
                        v-model.trim="healthGuardForm.platform_models[platform.value]"
                        type="text"
                        class="input"
                        :placeholder="platform.placeholder"
                      />
                      <input
                        v-model.number="healthGuardForm.platform_latency_ms[platform.value]"
                        type="number"
                        min="1"
                        step="500"
                        class="input"
                        :placeholder="String(healthGuardForm.healthy_latency_ms || 15000)"
                      />
                    </label>
                  </div>
                </div>

                <div class="health-guard-account-models-panel">
                  <div class="balance-section-title">{{ t('admin.upstreamProviders.healthGuardAccountModels') }}</div>
                  <div class="health-guard-account-models-add">
                    <Select
                      v-model="healthGuardAccountModelAccountToAdd"
                      :options="healthGuardAccountModelOptions"
                      searchable
                      clearable
                      class="health-guard-account-model-select"
                      :placeholder="t('admin.upstreamProviders.healthGuardAccountModelSelectPlaceholder')"
                      :empty-text="t('admin.upstreamProviders.healthGuardIgnoredNoOptions')"
                    >
                      <template #option="{ option }">
                        <span class="health-guard-ignored-option">
                          <strong>{{ option.label }}</strong>
                          <small>{{ option.meta }}</small>
                        </span>
                      </template>
                      <template #selected="{ option }">
                        <span v-if="option" class="health-guard-ignored-option is-selected">
                          <strong>{{ option.label }}</strong>
                          <small>{{ option.meta }}</small>
                        </span>
                        <span v-else>{{ t('admin.upstreamProviders.healthGuardAccountModelSelectPlaceholder') }}</span>
                      </template>
                    </Select>
                    <input
                      v-model.trim="healthGuardAccountModelDraft"
                      type="text"
                      class="input health-guard-account-model-input"
                      :placeholder="t('admin.upstreamProviders.healthGuardAccountModelInputPlaceholder')"
                      @keydown.enter.prevent="addHealthGuardAccountModel"
                    />
                    <button
                      type="button"
                      class="btn btn-primary"
                      :disabled="!healthGuardAccountModelAccountToAdd || !healthGuardAccountModelDraft.trim()"
                      data-test="health-guard-account-model-add"
                      @click="addHealthGuardAccountModel"
                    >
                      <Icon name="plus" size="xs" />
                      <span>{{ t('admin.upstreamProviders.healthGuardAccountModelAdd') }}</span>
                    </button>
                  </div>
                  <div v-if="healthGuardAccountModelRows.length" class="health-guard-account-model-list">
                    <article
                      v-for="row in healthGuardAccountModelRows"
                      :key="row.id"
                      class="health-guard-account-model-row"
                      :class="{ 'is-missing': row.missing }"
                    >
                      <div class="health-guard-account-model-account">
                        <strong>{{ healthGuardAccountModelName(row) }}</strong>
                        <small>{{ healthGuardAccountModelMeta(row) }}</small>
                      </div>
                      <input
                        :value="row.model"
                        type="text"
                        class="input health-guard-account-model-row-input"
                        :placeholder="t('admin.upstreamProviders.healthGuardAccountModelInputPlaceholder')"
                        @input="updateHealthGuardAccountModel(row.id, ($event.target as HTMLInputElement).value)"
                      />
                      <button
                        type="button"
                        class="health-guard-account-model-remove"
                        :title="t('common.delete')"
                        :aria-label="t('admin.upstreamProviders.healthGuardAccountModelRemove', { id: row.id })"
                        :data-test="`health-guard-account-model-remove-${row.id}`"
                        @click="removeHealthGuardAccountModel(row.id)"
                      >
                        <Icon name="x" size="xs" />
                      </button>
                    </article>
                  </div>
                  <div v-else class="health-guard-empty health-guard-account-model-empty">
                    {{ t('admin.upstreamProviders.healthGuardAccountModelEmpty') }}
                  </div>
                  <p class="health-guard-account-models-hint">{{ t('admin.upstreamProviders.healthGuardAccountModelsHint') }}</p>
                </div>
              </div>
            </section>
          </div>

          <div class="health-guard-record-panel">
            <div class="health-guard-record-header">
              <div class="balance-section-title">{{ t('admin.upstreamProviders.healthGuardRecentRuns') }}</div>
              <button type="button" class="btn btn-secondary btn-sm" :disabled="loadingHealthGuard" @click="loadHealthGuardState">
                {{ t('common.refresh') }}
              </button>
            </div>

            <div class="health-guard-summary-grid">
              <div v-for="card in healthGuardSummaryCards" :key="card.key" class="health-guard-summary-card" :class="card.tone">
                <span>{{ card.label }}</span>
                <strong>{{ card.value }}</strong>
              </div>
            </div>

            <div class="health-guard-detail-actions">
              <button
                type="button"
                class="health-guard-detail-action"
                :disabled="!healthGuardSkipReasons.length"
                @click="showHealthGuardSkipReasonsDialog = true"
              >
                <span>{{ t('admin.upstreamProviders.healthGuardSkipReasons') }}</span>
                <strong>{{ healthGuardSkippedCount }}</strong>
                <small>{{ t('admin.upstreamProviders.healthGuardViewDetails') }}</small>
              </button>
              <button
                type="button"
                class="health-guard-detail-action"
                :disabled="!healthGuardAdjustmentLogs.length"
                @click="showHealthGuardAdjustmentDialog = true"
              >
                <span>{{ t('admin.upstreamProviders.healthGuardAdjustmentLogs') }}</span>
                <strong>{{ healthGuardAdjustmentLogs.length }}</strong>
                <small>{{ t('admin.upstreamProviders.healthGuardViewDetails') }}</small>
              </button>
              <button
                type="button"
                class="health-guard-detail-action"
                :disabled="!latestHealthGuardItems.length"
                @click="showHealthGuardResultsDialog = true"
              >
                <span>{{ t('admin.upstreamProviders.healthGuardResultList') }}</span>
                <strong>{{ latestHealthGuardItems.length }}</strong>
                <small>{{ t('admin.upstreamProviders.healthGuardViewDetails') }}</small>
              </button>
            </div>
          </div>
        </div>
      </div>

      <template #footer>
        <div class="flex justify-end gap-3">
          <button type="button" class="btn btn-secondary" :disabled="savingHealthGuardConfig" @click="closeHealthGuardDialog">
            {{ t('common.cancel') }}
          </button>
          <button type="button" class="btn btn-primary" :disabled="savingHealthGuardConfig" @click="saveHealthGuardConfig">
            {{ savingHealthGuardConfig ? t('common.saving') : t('common.save') }}
          </button>
        </div>
      </template>
    </BaseDialog>

    <BaseDialog
      :show="showHealthGuardIgnoredDialog"
      :title="t('admin.upstreamProviders.healthGuardIgnoredManageTitle')"
      width="wide"
      :z-index="60"
      @close="closeHealthGuardIgnoredDialog"
    >
      <div class="health-guard-ignored-dialog">
        <div class="health-guard-ignored-add">
          <Select
            v-model="healthGuardIgnoredAccountToAdd"
            :options="healthGuardIgnoredAccountOptions"
            searchable
            clearable
            class="health-guard-ignored-select"
            :placeholder="t('admin.upstreamProviders.healthGuardIgnoredSelectPlaceholder')"
            :empty-text="t('admin.upstreamProviders.healthGuardIgnoredNoOptions')"
          >
            <template #option="{ option }">
              <span class="health-guard-ignored-option">
                <strong>{{ option.label }}</strong>
                <small>{{ option.meta }}</small>
              </span>
            </template>
            <template #selected="{ option }">
              <span v-if="option" class="health-guard-ignored-option is-selected">
                <strong>{{ option.label }}</strong>
                <small>{{ option.meta }}</small>
              </span>
              <span v-else>{{ t('admin.upstreamProviders.healthGuardIgnoredSelectPlaceholder') }}</span>
            </template>
          </Select>
          <button
            type="button"
            class="btn btn-primary"
            :disabled="!healthGuardIgnoredAccountToAdd"
            data-test="health-guard-ignored-add"
            @click="addHealthGuardIgnoredAccount"
          >
            {{ t('common.add') }}
          </button>
        </div>

        <div v-if="loadingHealthGuardIgnoredOptions" class="health-guard-ignored-loading">
          {{ t('common.loading') }}
        </div>

        <div v-if="healthGuardIgnoredAccountRows.length" class="health-guard-ignored-list">
          <article
            v-for="row in healthGuardIgnoredAccountRows"
            :key="row.id"
            class="health-guard-ignored-account"
            :class="{ 'is-missing': row.missing }"
          >
            <div class="health-guard-ignored-account-main">
              <strong>{{ healthGuardIgnoredAccountName(row) }}</strong>
              <code>#{{ row.id }}</code>
            </div>
            <div class="health-guard-ignored-account-meta">
              <span>{{ healthGuardIgnoredAccountPlatform(row) }}</span>
              <span :class="['record-status', healthGuardIgnoredAccountStatusClass(row)]">
                {{ healthGuardIgnoredAccountStatusLabel(row) }}
              </span>
            </div>
            <button
              type="button"
              class="health-guard-ignored-remove"
              :title="t('common.delete')"
              :aria-label="t('admin.upstreamProviders.healthGuardIgnoredRemoveAccount', { id: row.id })"
              :data-test="`health-guard-ignored-remove-${row.id}`"
              @click="removeHealthGuardIgnoredAccount(row.id)"
            >
              <Icon name="x" size="xs" />
            </button>
          </article>
        </div>
        <div v-else class="health-guard-empty">
          {{ t('admin.upstreamProviders.healthGuardIgnoredEmpty') }}
        </div>
      </div>
      <template #footer>
        <button type="button" class="btn btn-secondary" @click="closeHealthGuardIgnoredDialog">
          {{ t('common.close') }}
        </button>
      </template>
    </BaseDialog>

    <BaseDialog
      :show="showHealthGuardSkipReasonsDialog"
      :title="t('admin.upstreamProviders.healthGuardSkipReasonDetails')"
      width="wide"
      :z-index="60"
      @close="showHealthGuardSkipReasonsDialog = false"
    >
      <div class="health-guard-detail-dialog">
        <div v-if="healthGuardSkipReasons.length" class="health-guard-skip-list health-guard-modal-list">
          <article v-for="reason in healthGuardSkipReasons" :key="reason.reason" class="health-guard-skip-item">
            <div>
              <strong>{{ healthGuardSkipReasonLabel(reason.reason) }}</strong>
              <span>{{ t('admin.upstreamProviders.healthGuardSkipCount', { count: reason.count }) }}</span>
            </div>
            <p v-if="healthGuardSkipReasonSamples(reason)">
              {{ t('admin.upstreamProviders.healthGuardSkipSampleAccounts', { accounts: healthGuardSkipReasonSamples(reason) }) }}
            </p>
          </article>
        </div>
        <div v-else class="health-guard-empty">
          {{ t('admin.upstreamProviders.healthGuardNoSkipReasons') }}
        </div>
      </div>
      <template #footer>
        <button type="button" class="btn btn-secondary" @click="showHealthGuardSkipReasonsDialog = false">
          {{ t('common.close') }}
        </button>
      </template>
    </BaseDialog>

    <BaseDialog
      :show="showHealthGuardAdjustmentDialog"
      :title="t('admin.upstreamProviders.healthGuardAdjustmentLogDetails')"
      width="full"
      :z-index="60"
      @close="showHealthGuardAdjustmentDialog = false"
    >
      <div class="health-guard-detail-dialog health-guard-detail-dialog-large">
        <div class="health-guard-filter-row">
          <button
            v-for="option in healthGuardAdjustmentFilterOptions"
            :key="option.key"
            type="button"
            class="health-guard-filter-button"
            :class="{ 'is-active': activeHealthGuardAdjustmentFilter === option.key }"
            @click="activeHealthGuardAdjustmentFilter = option.key"
          >
            <span>{{ option.label }}</span>
            <strong>{{ option.count }}</strong>
          </button>
        </div>

        <div v-if="filteredHealthGuardAdjustmentLogs.length" class="health-guard-adjustment-list health-guard-modal-list">
          <article
            v-for="log in filteredHealthGuardAdjustmentLogs"
            :key="`${log.record.id}-${log.item.account_id}-${log.item.action}-${log.item.finished_at}`"
            class="health-guard-adjustment-item"
            :class="`is-${log.item.action}`"
          >
            <div class="health-guard-adjustment-main">
              <strong>{{ log.item.account_name || `#${log.item.account_id}` }}</strong>
              <span>{{ healthGuardAdjustmentTime(log) }}</span>
            </div>
            <div class="health-guard-adjustment-metrics">
              <span :class="['record-status', healthGuardActionClass(log.item.action)]">
                {{ healthGuardActionLabel(log.item.action) }}
              </span>
              <span>{{ healthGuardSchedulableChange(log.item) }}</span>
              <span>{{ log.item.provider_name || log.item.provider_slug }} / {{ healthGuardPlatformLabel(log.item.platform) }}</span>
            </div>
            <p v-if="log.item.reason || log.item.error_message">{{ log.item.reason || log.item.error_message }}</p>
          </article>
        </div>
        <div v-else class="health-guard-empty">
          {{ t('admin.upstreamProviders.healthGuardNoAdjustmentLogs') }}
        </div>
      </div>
      <template #footer>
        <button type="button" class="btn btn-secondary" @click="showHealthGuardAdjustmentDialog = false">
          {{ t('common.close') }}
        </button>
      </template>
    </BaseDialog>

    <BaseDialog
      :show="showHealthGuardResultsDialog"
      :title="t('admin.upstreamProviders.healthGuardResultDetails')"
      width="full"
      :z-index="60"
      @close="showHealthGuardResultsDialog = false"
    >
      <div class="health-guard-detail-dialog health-guard-detail-dialog-large">
        <div class="health-guard-filter-row">
          <button
            v-for="option in healthGuardResultFilterOptions"
            :key="option.key"
            type="button"
            class="health-guard-filter-button"
            :class="{ 'is-active': activeHealthGuardResultFilter === option.key }"
            @click="activeHealthGuardResultFilter = option.key"
          >
            <span>{{ option.label }}</span>
            <strong>{{ option.count }}</strong>
          </button>
        </div>

        <div v-if="filteredLatestHealthGuardItems.length" class="health-guard-item-list health-guard-modal-list">
          <article
            v-for="item in filteredLatestHealthGuardItems"
            :key="`${item.account_id}-${item.finished_at}`"
            class="health-guard-item-card"
            :class="healthGuardItemClass(item.status)"
          >
            <div class="health-guard-item-main">
              <strong>{{ item.account_name || `#${item.account_id}` }}</strong>
              <span>{{ item.provider_name || item.provider_slug }} / {{ healthGuardPlatformLabel(item.platform) }}</span>
            </div>
            <div class="health-guard-item-metrics">
              <span :class="['record-status', healthGuardStatusClass(item.status)]">
                {{ healthGuardStatusLabel(item.status) }}
              </span>
              <span>{{ formatLatencyMs(item.latency_ms) }} / {{ formatLatencyMs(item.latency_limit_ms) }}</span>
              <span>{{ healthGuardActionLabel(item.action) }}</span>
            </div>
            <p v-if="item.reason || item.error_message">{{ item.reason || item.error_message }}</p>
          </article>
        </div>
        <div v-else class="health-guard-empty">
          {{ latestHealthGuardRecord ? t('admin.upstreamProviders.healthGuardNoCheckedItems') : t('admin.upstreamProviders.healthGuardNoRecords') }}
        </div>
      </div>
      <template #footer>
        <button type="button" class="btn btn-secondary" @click="showHealthGuardResultsDialog = false">
          {{ t('common.close') }}
        </button>
      </template>
    </BaseDialog>

    <BaseDialog
      :show="showFormDialog"
      :title="formMode === 'create' ? t('admin.upstreamProviders.createProvider') : t('admin.upstreamProviders.editProvider')"
      width="wide"
      @close="closeFormDialog"
    >
      <form id="upstream-provider-form" class="space-y-5" @submit.prevent="submitForm">
        <div class="grid gap-4 md:grid-cols-2">
          <div>
            <label class="input-label">{{ t('admin.upstreamProviders.type') }}</label>
            <select v-model="form.type" class="input" :disabled="formMode === 'edit'">
              <option value="sub2api">Sub2API</option>
              <option value="newapi">NewAPI</option>
            </select>
          </div>
          <div>
            <label class="input-label">{{ t('admin.upstreamProviders.enabled') }}</label>
            <label class="flex h-10 items-center gap-3 rounded-lg border border-gray-200 px-3 dark:border-dark-600">
              <Toggle v-model="form.enabled" />
              <span class="text-sm text-gray-700 dark:text-gray-200">
                {{ form.enabled ? t('common.enabled') : t('common.disabled') }}
              </span>
            </label>
          </div>
          <div>
            <label class="input-label">{{ t('admin.upstreamProviders.defaultProvider') }}</label>
            <label class="flex h-10 items-center gap-2 rounded-lg border border-gray-200 px-3 dark:border-dark-600">
              <input v-model="form.is_default" type="checkbox" class="rounded border-gray-300 text-primary-600 focus:ring-primary-500" />
              <span class="text-sm text-gray-700 dark:text-gray-200">
                {{ form.is_default ? t('admin.upstreamProviders.defaultProvider') : t('admin.upstreamProviders.notDefaultProvider') }}
              </span>
            </label>
          </div>
          <div>
            <label class="input-label">{{ t('admin.upstreamProviders.name') }}</label>
            <input v-model.trim="form.name" required type="text" class="input" />
          </div>
          <div>
            <label class="input-label">{{ t('admin.upstreamProviders.slug') }}</label>
            <input
              v-model.trim="form.slug"
              required
              type="text"
              class="input"
              pattern="[A-Za-z0-9][A-Za-z0-9_-]{0,63}"
              :disabled="formMode === 'edit'"
            />
          </div>
          <div>
            <label class="input-label">{{ t('admin.upstreamProviders.sortOrder') }}</label>
            <input
              v-model.number="form.sort_order"
              type="number"
              min="0"
              step="1"
              class="input"
            />
            <p class="mt-1 text-xs text-gray-500 dark:text-gray-400">
              {{ t('admin.upstreamProviders.sortOrderHint') }}
            </p>
          </div>
        </div>

        <div class="grid gap-4 md:grid-cols-2">
          <div class="md:col-span-2">
            <label class="input-label">{{ t('admin.upstreamProviders.baseUrl') }}</label>
            <input v-model.trim="form.base_url" required type="url" class="input" list="upstream-provider-base-url-options" placeholder="https://example.com" />
          </div>
          <div>
            <label class="input-label">{{ t('admin.upstreamProviders.apiKeysUrl') }}</label>
            <input v-model.trim="form.api_keys_url" required type="text" class="input" list="upstream-provider-api-keys-url-options" placeholder="/api/token/" />
          </div>
          <div>
            <label class="input-label">
              {{ t('admin.upstreamProviders.loginUrl') }}
              <span v-if="form.type === 'sub2api'" class="text-xs font-normal text-gray-400">({{ t('common.optional') }})</span>
            </label>
            <input
              v-model.trim="form.login_url"
              :required="form.type === 'newapi'"
              type="text"
              class="input"
              list="upstream-provider-login-url-options"
              :placeholder="form.type === 'sub2api' ? '/api/v1/auth/login' : '/api/user/login'"
            />
          </div>
          <div v-if="form.type === 'newapi'" class="md:col-span-2">
            <label class="input-label">{{ t('admin.upstreamProviders.groupsUrl') }}</label>
            <input v-model.trim="form.groups_url" required type="text" class="input" list="upstream-provider-groups-url-options" placeholder="/api/group/" />
          </div>
          <div class="md:col-span-2">
            <label class="input-label">
              {{ t('admin.upstreamProviders.availableGroupsUrl') }}
              <span class="text-xs font-normal text-gray-400">({{ t('common.optional') }})</span>
            </label>
            <input
              v-model.trim="form.available_groups_url"
              type="text"
              class="input"
              list="upstream-provider-available-groups-url-options"
              placeholder="/api/v1/groups/available?timezone=Asia%2FShanghai"
            />
          </div>
          <div class="md:col-span-2">
            <label class="input-label">
              {{ t('admin.upstreamProviders.balanceUrl') }}
              <span class="text-xs font-normal text-gray-400">({{ t('common.optional') }})</span>
            </label>
            <input
              v-model.trim="form.balance_url"
              type="text"
              class="input"
              list="upstream-provider-balance-url-options"
              :placeholder="form.type === 'newapi' ? '/api/user/self' : '/api/v1/auth/me?timezone=Asia%2FShanghai'"
            />
          </div>
          <div class="md:col-span-2">
            <label class="input-label">
              {{ t('admin.upstreamProviders.usageCostUrl') }}
              <span class="text-xs font-normal text-gray-400">({{ t('common.optional') }})</span>
            </label>
            <input
              v-model.trim="form.usage_cost_url"
              type="text"
              class="input"
              list="upstream-provider-usage-cost-url-options"
              :placeholder="form.type === 'newapi'
                ? '/api/log/self/stat?type=0&token_name=&model_name=&start_timestamp={start_timestamp}&end_timestamp={end_timestamp}&group='
                : '/api/v1/usage/dashboard/stats?timezone=Asia%2FShanghai'"
            />
          </div>
        </div>

        <datalist id="upstream-provider-base-url-options">
          <option v-for="option in urlOptions.base_url" :key="`base-${option}`" :value="option" />
        </datalist>
        <datalist id="upstream-provider-api-keys-url-options">
          <option v-for="option in urlOptions.api_keys_url" :key="`keys-${option}`" :value="option" />
        </datalist>
        <datalist id="upstream-provider-login-url-options">
          <option v-for="option in urlOptions.login_url" :key="`login-${option}`" :value="option" />
        </datalist>
        <datalist id="upstream-provider-groups-url-options">
          <option v-for="option in urlOptions.groups_url" :key="`groups-${option}`" :value="option" />
        </datalist>
        <datalist id="upstream-provider-available-groups-url-options">
          <option v-for="option in urlOptions.available_groups_url" :key="`available-groups-${option}`" :value="option" />
        </datalist>
        <datalist id="upstream-provider-balance-url-options">
          <option v-for="option in urlOptions.balance_url" :key="`balance-${option}`" :value="option" />
        </datalist>
        <datalist id="upstream-provider-usage-cost-url-options">
          <option v-for="option in urlOptions.usage_cost_url" :key="`usage-cost-${option}`" :value="option" />
        </datalist>

        <div class="grid gap-4 md:grid-cols-2">
          <div>
            <label class="input-label">
              {{ form.type === 'newapi' ? t('admin.upstreamProviders.usernameOrEmail') : t('admin.upstreamProviders.email') }}
            </label>
            <input v-if="form.type === 'sub2api'" v-model.trim="form.email" type="email" class="input" />
            <input v-else v-model.trim="form.username" required type="text" class="input" />
          </div>
          <div>
            <label class="input-label">
              {{ t('admin.upstreamProviders.password') }}
              <span v-if="editingProvider?.password_configured" class="text-xs font-normal text-gray-400">
                {{ t('admin.upstreamProviders.blankKeepsPassword') }}
              </span>
            </label>
            <input
              v-model="form.password"
              :required="form.type === 'newapi' && formMode === 'create'"
              type="password"
              class="input"
              autocomplete="new-password"
            />
          </div>
          <div>
            <label class="input-label">{{ t('admin.upstreamProviders.accountNamePrefix') }}</label>
            <input v-model.trim="form.account_name_prefix" type="text" class="input" />
          </div>
          <div>
            <label class="input-label">{{ t('admin.upstreamProviders.accountRateMultiplierScale') }}</label>
            <input
              v-model.number="form.account_rate_multiplier_scale"
              required
              type="number"
              min="0.000001"
              step="any"
              class="input"
            />
            <p class="mt-1 text-xs text-gray-500 dark:text-gray-400">
              {{ t('admin.upstreamProviders.accountRateMultiplierScaleHint') }}
            </p>
          </div>
          <div>
            <label class="input-label">{{ t('admin.upstreamProviders.tempDisableMinutes') }}</label>
            <input v-model.number="form.temp_disable_minutes" type="number" min="0" class="input" />
          </div>
        </div>
      </form>

      <template #footer>
        <div class="flex flex-wrap justify-between gap-3">
          <button
            type="button"
            class="btn btn-secondary"
            :disabled="testingDraft || submitting"
            @click="testDraftProvider"
          >
            <Icon name="play" size="sm" class="mr-2" :class="testingDraft ? 'animate-pulse' : ''" />
            {{ t('admin.upstreamProviders.testDraft') }}
          </button>
          <div class="flex gap-3">
            <button type="button" class="btn btn-secondary" @click="closeFormDialog">
              {{ t('common.cancel') }}
            </button>
            <button type="submit" form="upstream-provider-form" class="btn btn-primary" :disabled="submitting">
              {{ submitting ? t('common.saving') : t('common.save') }}
            </button>
          </div>
        </div>
      </template>
    </BaseDialog>

    <BaseDialog
      :show="showTestDialog"
      :title="t('admin.upstreamProviders.testResult')"
      width="wide"
      @close="showTestDialog = false"
    >
      <div v-if="testResult" class="space-y-5">
        <div class="grid gap-3 md:grid-cols-3">
          <div v-for="stage in testStages(testResult)" :key="stage.key" class="rounded-lg border border-gray-200 p-4 dark:border-dark-600">
            <div class="mb-2 flex items-center justify-between">
              <span class="text-sm font-medium text-gray-900 dark:text-white">{{ stage.label }}</span>
              <span :class="['badge', stage.stage.ok ? 'badge-success' : 'badge-danger']">
                {{ stage.stage.ok ? t('admin.upstreamProviders.stageOk') : t('admin.upstreamProviders.stageFailed') }}
              </span>
            </div>
            <div class="space-y-1 text-xs text-gray-600 dark:text-gray-300">
              <div v-if="stage.stage.status_code">{{ t('admin.upstreamProviders.statusCode') }}: {{ stage.stage.status_code }}</div>
              <div v-if="stage.stage.item_count !== undefined">{{ t('admin.upstreamProviders.itemCount') }}: {{ stage.stage.item_count }}</div>
              <div v-if="stage.stage.group_count !== undefined">{{ t('admin.upstreamProviders.groupCount') }}: {{ stage.stage.group_count }}</div>
              <div v-if="stage.stage.user_id">{{ t('admin.upstreamProviders.userId') }}: {{ stage.stage.user_id }}</div>
              <div v-if="stage.stage.error" class="text-red-600 dark:text-red-300">{{ stage.stage.error }}</div>
            </div>
          </div>
        </div>

        <div v-if="testResult.warnings?.length" class="rounded-lg border border-amber-200 bg-amber-50 p-3 text-sm text-amber-800 dark:border-amber-700/40 dark:bg-amber-900/20 dark:text-amber-200">
          <div v-for="warning in testResult.warnings" :key="warning">{{ warning }}</div>
        </div>

        <div>
          <h4 class="mb-2 text-sm font-medium text-gray-900 dark:text-white">
            {{ t('admin.upstreamProviders.parsedKeys') }}
          </h4>
          <div class="max-h-72 overflow-auto rounded-lg border border-gray-200 dark:border-dark-600">
            <div class="provider-mobile-record-cards">
              <article
                v-for="item in testResult.parsed_keys"
                :key="`test-card-${item.key_name}-${item.group_name}`"
                class="provider-mobile-record-card"
              >
                <div>
                  <span>{{ t('admin.upstreamProviders.keyName') }}</span>
                  <strong>{{ item.key_name || '-' }}</strong>
                </div>
                <div>
                  <span>{{ t('admin.upstreamProviders.groupName') }}</span>
                  <strong>{{ item.group_name || '-' }}</strong>
                </div>
                <div>
                  <span>{{ t('admin.upstreamProviders.rateMultiplier') }}</span>
                  <strong>{{ formatRate(item.rate_multiplier) }}</strong>
                </div>
                <div>
                  <span>{{ t('admin.upstreamProviders.rawStatus') }}</span>
                  <strong>{{ item.raw_status || '-' }}</strong>
                </div>
              </article>
              <div v-if="!testResult.parsed_keys?.length" class="provider-mobile-record-empty">{{ t('common.noData') }}</div>
            </div>
            <table class="w-full min-w-[720px] divide-y divide-gray-200 text-sm dark:divide-dark-700">
              <thead class="bg-gray-50 dark:bg-dark-800">
                <tr>
                  <th class="px-4 py-2 text-left font-medium">{{ t('admin.upstreamProviders.keyName') }}</th>
                  <th class="px-4 py-2 text-left font-medium">{{ t('admin.upstreamProviders.groupName') }}</th>
                  <th class="px-4 py-2 text-left font-medium">{{ t('admin.upstreamProviders.rateMultiplier') }}</th>
                  <th class="px-4 py-2 text-left font-medium">{{ t('admin.upstreamProviders.rawStatus') }}</th>
                </tr>
              </thead>
              <tbody class="divide-y divide-gray-100 dark:divide-dark-700">
                <tr v-for="item in testResult.parsed_keys" :key="`${item.key_name}-${item.group_name}`">
                  <td class="px-4 py-2">{{ item.key_name || '-' }}</td>
                  <td class="px-4 py-2">{{ item.group_name || '-' }}</td>
                  <td class="px-4 py-2">{{ formatRate(item.rate_multiplier) }}</td>
                  <td class="px-4 py-2">{{ item.raw_status || '-' }}</td>
                </tr>
                <tr v-if="!testResult.parsed_keys?.length">
                  <td colspan="4" class="px-4 py-6 text-center text-gray-400">{{ t('common.noData') }}</td>
                </tr>
              </tbody>
            </table>
          </div>
        </div>
      </div>
    </BaseDialog>

    <BaseDialog
      :show="showKeysDialog"
      :title="keysDialogTitle"
      width="wide"
      @close="closeKeysDialog"
    >
      <div class="space-y-4">
        <div v-if="keysWarnings.length" class="rounded-lg border border-amber-200 bg-amber-50 p-3 text-sm text-amber-800 dark:border-amber-700/40 dark:bg-amber-900/20 dark:text-amber-200">
          <div v-for="warning in keysWarnings" :key="warning">{{ warning }}</div>
        </div>
        <div class="max-h-[60vh] overflow-auto rounded-lg border border-gray-200 dark:border-dark-600">
          <div class="provider-mobile-record-cards">
            <article
              v-for="item in keysItems"
              :key="`keys-card-${item.key_name}-${item.group_name}-${item.raw_group_id}`"
              class="provider-mobile-record-card"
            >
              <div>
                <span>{{ t('admin.upstreamProviders.keyName') }}</span>
                <strong>{{ item.key_name || '-' }}</strong>
              </div>
              <div>
                <span>{{ t('admin.upstreamProviders.groupName') }}</span>
                <strong>{{ item.group_name || '-' }}</strong>
              </div>
              <div>
                <span>{{ t('admin.upstreamProviders.rateMultiplier') }}</span>
                <strong>{{ formatRate(item.rate_multiplier) }}</strong>
              </div>
              <div>
                <span>{{ t('admin.upstreamProviders.rawStatus') }}</span>
                <strong>{{ item.raw_status || '-' }}</strong>
              </div>
              <div>
                <span>{{ t('admin.upstreamProviders.rawGroupId') }}</span>
                <strong>{{ item.raw_group_id || '-' }}</strong>
              </div>
            </article>
            <div v-if="!keysItems.length" class="provider-mobile-record-empty">{{ t('common.noData') }}</div>
          </div>
          <table class="w-full min-w-[760px] divide-y divide-gray-200 text-sm dark:divide-dark-700">
            <thead class="bg-gray-50 dark:bg-dark-800">
              <tr>
                <th class="px-4 py-2 text-left font-medium">{{ t('admin.upstreamProviders.keyName') }}</th>
                <th class="px-4 py-2 text-left font-medium">{{ t('admin.upstreamProviders.groupName') }}</th>
                <th class="px-4 py-2 text-left font-medium">{{ t('admin.upstreamProviders.rateMultiplier') }}</th>
                <th class="px-4 py-2 text-left font-medium">{{ t('admin.upstreamProviders.rawStatus') }}</th>
                <th class="px-4 py-2 text-left font-medium">{{ t('admin.upstreamProviders.rawGroupId') }}</th>
              </tr>
            </thead>
            <tbody class="divide-y divide-gray-100 dark:divide-dark-700">
              <tr v-for="item in keysItems" :key="`${item.key_name}-${item.group_name}-${item.raw_group_id}`">
                <td class="px-4 py-2">{{ item.key_name || '-' }}</td>
                <td class="px-4 py-2">{{ item.group_name || '-' }}</td>
                <td class="px-4 py-2">{{ formatRate(item.rate_multiplier) }}</td>
                <td class="px-4 py-2">{{ item.raw_status || '-' }}</td>
                <td class="px-4 py-2">{{ item.raw_group_id || '-' }}</td>
              </tr>
              <tr v-if="!keysItems.length">
                <td colspan="5" class="px-4 py-8 text-center text-gray-400">{{ t('common.noData') }}</td>
              </tr>
            </tbody>
          </table>
        </div>
      </div>
    </BaseDialog>

    <div
      v-if="balanceDetailsOpen"
      class="balance-dialog-overlay fixed inset-0 z-50 flex items-end justify-center bg-black/40 p-4 sm:items-center"
      @click.self="closeBalanceDetails"
    >
      <div class="balance-dialog">
        <div class="balance-dialog-header">
          <span class="balance-dialog-handle" aria-hidden="true"></span>
          <div class="min-w-0">
            <h3 class="truncate text-base font-semibold text-gray-950 dark:text-white">
              {{ selectedBalanceProviderLabel }}
            </h3>
            <p class="mt-1 text-sm text-gray-500 dark:text-gray-400">
              {{ t('admin.upstreamProviders.balanceDialogDescription') }}
            </p>
          </div>
          <button type="button" class="btn btn-secondary btn-sm balance-dialog-close" @click="closeBalanceDetails">
            <Icon name="x" size="sm" />
            <span>{{ t('common.close') }}</span>
          </button>
        </div>

        <div class="balance-dialog-body">
          <div class="balance-summary-grid">
            <div class="balance-metric">
              <span>{{ t('admin.upstreamProviders.currentBalance') }}</span>
              <strong>{{ formatMoney(selectedBalanceSummary?.current_balance) }}</strong>
            </div>
            <div class="balance-metric">
              <span>{{ t('admin.upstreamProviders.todayConsumption') }}</span>
              <strong>{{ formatMoney(todayConsumptionForProvider(selectedBalanceProviderSlug)) }}</strong>
            </div>
            <div class="balance-metric">
              <span>{{ t('admin.upstreamProviders.amountScale') }}</span>
              <strong>{{ formatScale(selectedBalanceScale) }}</strong>
            </div>
            <div class="balance-metric">
              <span>{{ t('admin.upstreamProviders.lastSnapshot') }}</span>
              <strong
                class="balance-last-snapshot text-sm"
                :title="selectedBalanceSummary?.last_snapshot_at ? formatDateTime(selectedBalanceSummary.last_snapshot_at) : ''"
              >
                <span class="balance-last-snapshot-full">
                  {{ selectedBalanceSummary?.last_snapshot_at ? formatDateTime(selectedBalanceSummary.last_snapshot_at) : '-' }}
                </span>
                <span class="balance-last-snapshot-compact">
                  {{ selectedBalanceSummary?.last_snapshot_at ? formatCompactDateTime(selectedBalanceSummary.last_snapshot_at) : '-' }}
                </span>
              </strong>
            </div>
          </div>

          <div class="balance-recharge-panel balance-dialog-section">
            <div class="balance-section-title">{{ t('admin.upstreamProviders.addRecharge') }}</div>
            <div class="balance-recharge-form">
              <input v-model.number="rechargeForm.amount" type="number" min="0" step="0.000001" class="input" :placeholder="t('admin.upstreamProviders.rechargeAmount')" />
              <input v-model="rechargeForm.note" type="text" class="input" :placeholder="t('admin.upstreamProviders.rechargeNote')" />
              <button type="button" class="btn btn-secondary" :disabled="addingRecharge" @click="addBalanceRecharge">
                <Icon name="plus" size="sm" class="mr-2" />
                {{ t('common.add') }}
              </button>
            </div>
          </div>

          <div class="balance-record-section">
            <div class="balance-record-tabs" role="tablist">
              <button
                type="button"
                :class="['balance-record-tab', activeBalanceRecordTab === 'samples' && 'is-active']"
                :aria-selected="activeBalanceRecordTab === 'samples'"
                @click="activeBalanceRecordTab = 'samples'"
              >
                <span>{{ t('admin.upstreamProviders.balanceSamples') }}</span>
                <strong>{{ selectedBalanceSnapshots.length }}</strong>
              </button>
              <button
                type="button"
                :class="['balance-record-tab', activeBalanceRecordTab === 'history' && 'is-active']"
                :aria-selected="activeBalanceRecordTab === 'history'"
                @click="activeBalanceRecordTab = 'history'"
              >
                <span>{{ t('admin.upstreamProviders.balanceHistory') }}</span>
                <strong>{{ selectedBalanceRows.length }}</strong>
              </button>
            </div>

            <section v-show="activeBalanceRecordTab === 'samples'" class="balance-record-pane">
            <div class="balance-record-header">
              <div class="balance-section-title">{{ t('admin.upstreamProviders.balanceSamples') }}</div>
              <span class="balance-record-count">{{ selectedBalanceSnapshots.length }}</span>
            </div>
            <div class="balance-record-list overflow-auto rounded-lg border border-gray-200 dark:border-dark-600">
              <div class="provider-mobile-record-cards">
                <article
                  v-for="snapshot in selectedBalanceSnapshots"
                  :key="'snapshot-card-' + (snapshot.id || (snapshot.provider_slug + ':' + snapshot.captured_at))"
                  :class="['provider-mobile-record-card', 'balance-record-card', snapshot.status === 'success' ? 'is-success' : 'is-error']"
                >
                  <div>
                    <span>{{ t('admin.upstreamProviders.sampleTime') }}</span>
                    <strong>{{ formatDateTime(snapshot.captured_at) }}</strong>
                  </div>
                  <div>
                    <span>{{ t('admin.upstreamProviders.currentBalance') }}</span>
                    <strong>{{ formatMoney(snapshot.balance) }}</strong>
                  </div>
                  <div>
                    <span>{{ t('admin.upstreamProviders.amountScale') }}</span>
                    <strong>{{ formatScale(snapshot.amount_scale) }}</strong>
                  </div>
                  <div>
                    <span>{{ t('common.status') }}</span>
                    <strong>{{ snapshot.status === 'success' ? t('admin.upstreamProviders.balanceComplete') : t('admin.upstreamProviders.balanceAnomaly') }}</strong>
                  </div>
                  <div>
                    <span>{{ t('admin.upstreamProviders.sampleError') }}</span>
                    <strong>{{ snapshot.error || '-' }}</strong>
                  </div>
                </article>
                <div v-if="!selectedBalanceSnapshots.length" class="provider-mobile-record-empty">{{ t('admin.upstreamProviders.noBalanceSamples') }}</div>
              </div>
              <table class="records-table min-w-[760px]">
                <thead class="bg-gray-50 dark:bg-dark-800">
                  <tr>
                    <th class="px-4 py-2 text-left font-medium">{{ t('admin.upstreamProviders.sampleTime') }}</th>
                    <th class="px-4 py-2 text-left font-medium">{{ t('admin.upstreamProviders.currentBalance') }}</th>
                    <th class="px-4 py-2 text-left font-medium">{{ t('admin.upstreamProviders.amountScale') }}</th>
                    <th class="px-4 py-2 text-left font-medium">{{ t('common.status') }}</th>
                    <th class="px-4 py-2 text-left font-medium">{{ t('admin.upstreamProviders.sampleError') }}</th>
                  </tr>
                </thead>
                <tbody>
                  <tr v-for="snapshot in selectedBalanceSnapshots" :key="snapshot.id || `${snapshot.provider_slug}-${snapshot.captured_at}`" class="records-row snapshot-row">
                    <td class="px-4 py-3 font-mono text-gray-600 dark:text-gray-300">{{ formatDateTime(snapshot.captured_at) }}</td>
                    <td class="px-4 py-3 font-mono">{{ formatMoney(snapshot.balance) }}</td>
                    <td class="px-4 py-3 font-mono">{{ formatScale(snapshot.amount_scale) }}</td>
                    <td class="px-4 py-3">
                      <span :class="['record-status', snapshot.status === 'success' ? 'record-status-success' : 'record-status-error']">
                        {{ snapshot.status === 'success' ? t('admin.upstreamProviders.balanceComplete') : t('admin.upstreamProviders.balanceAnomaly') }}
                      </span>
                    </td>
                    <td class="px-4 py-3 text-gray-500 dark:text-gray-300">{{ snapshot.error || '-' }}</td>
                  </tr>
                  <tr v-if="!selectedBalanceSnapshots.length">
                    <td colspan="5" class="px-4 py-8 text-center text-gray-400">{{ t('admin.upstreamProviders.noBalanceSamples') }}</td>
                  </tr>
                </tbody>
              </table>
            </div>
            </section>

            <section v-show="activeBalanceRecordTab === 'history'" class="balance-record-pane">
            <div class="balance-record-header">
              <div class="balance-section-title">{{ t('admin.upstreamProviders.balanceHistory') }}</div>
              <span class="balance-record-count">{{ selectedBalanceRows.length }}</span>
            </div>
            <div class="balance-record-list overflow-auto rounded-lg border border-gray-200 dark:border-dark-600">
              <div class="provider-mobile-record-cards">
                <article
                  v-for="row in selectedBalanceRows"
                  :key="`daily-card-${row.provider_slug}-${row.date}`"
                  :class="['provider-mobile-record-card', 'balance-record-card', row.anomaly ? 'is-error' : row.complete ? 'is-success' : 'is-muted']"
                >
                  <div>
                    <span>{{ t('admin.upstreamProviders.balanceDate') }}</span>
                    <strong>{{ row.date }}</strong>
                  </div>
                  <div>
                    <span>{{ t('admin.upstreamProviders.openingBalance') }}</span>
                    <strong>{{ formatMoney(row.opening_balance) }}</strong>
                  </div>
                  <div>
                    <span>{{ t('admin.upstreamProviders.rechargeAmount') }}</span>
                    <strong>{{ formatMoney(row.recharge_amount) }}</strong>
                  </div>
                  <div>
                    <span>{{ t('admin.upstreamProviders.closingBalance') }}</span>
                    <strong>{{ formatMoney(row.closing_balance) }}</strong>
                  </div>
                  <div>
                    <span>{{ t('admin.upstreamProviders.consumptionAmount') }}</span>
                    <strong>{{ formatMoney(row.consumption_amount) }}</strong>
                  </div>
                  <div>
                    <span>{{ t('common.status') }}</span>
                    <strong>{{ balanceRowStatus(row) }}</strong>
                  </div>
                </article>
                <div v-if="!selectedBalanceRows.length" class="provider-mobile-record-empty">{{ t('admin.upstreamProviders.noBalanceHistory') }}</div>
              </div>
              <table class="records-table min-w-[760px]">
                <thead class="bg-gray-50 dark:bg-dark-800">
                  <tr>
                    <th class="px-4 py-2 text-left font-medium">{{ t('admin.upstreamProviders.balanceDate') }}</th>
                    <th class="px-4 py-2 text-left font-medium">{{ t('admin.upstreamProviders.openingBalance') }}</th>
                    <th class="px-4 py-2 text-left font-medium">{{ t('admin.upstreamProviders.rechargeAmount') }}</th>
                    <th class="px-4 py-2 text-left font-medium">{{ t('admin.upstreamProviders.closingBalance') }}</th>
                    <th class="px-4 py-2 text-left font-medium">{{ t('admin.upstreamProviders.consumptionAmount') }}</th>
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
                    <td colspan="6" class="px-4 py-8 text-center text-gray-400">{{ t('admin.upstreamProviders.noBalanceHistory') }}</td>
                  </tr>
                </tbody>
              </table>
            </div>
            </section>
          </div>
        </div>
      </div>
    </div>

    <ConfirmDialog
      :show="showDeleteDialog"
      :title="t('common.delete')"
      :message="deleteMessage"
      :confirm-text="t('common.delete')"
      :cancel-text="t('common.cancel')"
      :danger="true"
      @confirm="confirmDelete"
      @cancel="showDeleteDialog = false"
    />
  </AppLayout>
</template>

<script setup lang="ts">
import { computed, onMounted, reactive, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { useRoute, useRouter } from 'vue-router'
import { adminAPI } from '@/api/admin'
import type {
  UpstreamProviderBalance,
  UpstreamProviderConfig,
  UpstreamProviderKey,
  UpstreamProviderTestResult,
  UpstreamProviderTestStage,
} from '@/api/admin/upstreamProviders'
import type {
  UpstreamAccountHealthGuardConfig,
  UpstreamAccountHealthGuardRunItem,
  UpstreamAccountHealthGuardRunRecord,
  UpstreamAccountHealthGuardSkipReason,
  UpstreamBalanceConsumptionOverview,
  UpstreamBalanceDailyRow,
  UpstreamBalanceProviderSummary,
  UpstreamBalanceSamplerConfig,
  UpstreamBalanceSnapshot,
} from '@/api/admin/upstreamAccountSync'
import type { Account } from '@/types'
import { useAppStore } from '@/stores/app'
import { extractApiErrorMessage } from '@/utils/apiError'
import { formatDateTime } from '@/utils/format'
import { useClipboard } from '@/composables/useClipboard'
import { useRouteQueryFilters } from '@/composables/useRouteQueryFilters'
import type { Column } from '@/components/common/types'
import AppLayout from '@/components/layout/AppLayout.vue'
import TablePageLayout from '@/components/layout/TablePageLayout.vue'
import DataTable from '@/components/common/DataTable.vue'
import BaseDialog from '@/components/common/BaseDialog.vue'
import UpstreamBalanceSamplerDialog from '@/components/admin/upstream/UpstreamBalanceSamplerDialog.vue'
import UpstreamHealthGuardPolicyFields from '@/components/admin/upstream/UpstreamHealthGuardPolicyFields.vue'
import ConfirmDialog from '@/components/common/ConfirmDialog.vue'
import EmptyState from '@/components/common/EmptyState.vue'
import Select, { type SelectOption } from '@/components/common/Select.vue'
import Toggle from '@/components/common/Toggle.vue'
import Icon from '@/components/icons/Icon.vue'
import UpstreamBalanceCharts from '@/components/admin/upstream/UpstreamBalanceCharts.vue'

const { t } = useI18n()
const route = useRoute()
const router = useRouter()
const appStore = useAppStore()
const { copyToClipboard } = useClipboard()

type UpstreamProviderTableRow = UpstreamProviderConfig & {
  balance?: number
  today_consumption?: number
}
type QuickProviderFilterKey = 'all' | 'enabled' | 'disabled' | 'default' | 'balance_anomaly'
type HealthGuardForm = UpstreamAccountHealthGuardConfig & {
  account_models: Record<string, string>
  platform_models: Record<string, string>
  platform_latency_ms: Record<string, number>
}
type HealthGuardAdjustmentLogItem = {
  record: UpstreamAccountHealthGuardRunRecord
  item: UpstreamAccountHealthGuardRunItem
}
type HealthGuardIgnoredAccountSummary = {
  id: number
  account?: Account
  loading?: boolean
  missing?: boolean
}
type HealthGuardAccountModelRow = HealthGuardIgnoredAccountSummary & {
  model: string
}
type HealthGuardIgnoredAccountOption = SelectOption & {
  account?: Account
  meta: string
}
type HealthGuardAdjustmentFilterKey = 'all' | 'latest' | 'disabled' | 'recovered'
type HealthGuardResultFilterKey = 'all' | 'failed' | 'slow' | 'healthy' | 'changed' | 'disabled' | 'recovered' | 'unchanged' | 'skipped'

const providers = ref<UpstreamProviderConfig[]>([])
const loading = ref(false)
const searchQuery = ref('')
const typeFilter = ref('')
const enabledFilter = ref('')
const showProviderAdvancedFilters = ref(false)
const activeQuickProviderFilter = ref<QuickProviderFilterKey>('all')
useRouteQueryFilters([
  { queryKey: 'provider', state: searchQuery },
  { queryKey: 'status', state: enabledFilter, fromQuery: value => value === 'disabled' ? 'disabled' : value === 'enabled' ? 'enabled' : '', toQuery: value => value || undefined },
])
const defaultingSlug = ref<string | null>(null)
const showColumnSettings = ref(false)
const visibleOptionalColumns = ref<string[]>([])
const expandedProviderSlugs = ref(new Set<string>())
const copiedValue = ref('')
let copiedTimer: ReturnType<typeof setTimeout> | undefined

const showFormDialog = ref(false)
const formMode = ref<'create' | 'edit'>('create')
const editingProvider = ref<UpstreamProviderConfig | null>(null)
const submitting = ref(false)
const testingDraft = ref(false)

const showBalanceSamplerDialog = ref(false)
const savingBalanceSamplerConfig = ref(false)
const showHealthGuardDialog = ref(false)
const loadingHealthGuard = ref(false)
const savingHealthGuardConfig = ref(false)
const runningHealthGuardNow = ref(false)
const healthGuardConfigExpanded = ref(
  typeof window === 'undefined' || typeof window.matchMedia !== 'function'
    ? true
    : !window.matchMedia('(max-width: 767px)').matches
)
const showHealthGuardIgnoredDialog = ref(false)
const loadingHealthGuardIgnoredOptions = ref(false)
const healthGuardIgnoredAccountOptionsSource = ref<Account[]>([])
const healthGuardIgnoredAccountToAdd = ref<number | null>(null)
const healthGuardAccountModelAccountToAdd = ref<number | null>(null)
const healthGuardAccountModelDraft = ref('')
const showHealthGuardSkipReasonsDialog = ref(false)
const showHealthGuardAdjustmentDialog = ref(false)
const showHealthGuardResultsDialog = ref(false)
const activeHealthGuardAdjustmentFilter = ref<HealthGuardAdjustmentFilterKey>('all')
const activeHealthGuardResultFilter = ref<HealthGuardResultFilterKey>('all')

const showTestDialog = ref(false)
const testResult = ref<UpstreamProviderTestResult | null>(null)
const testingSlugs = ref(new Set<string>())
const balanceLoadingSlugs = ref(new Set<string>())
const providerBalances = ref<Record<string, UpstreamProviderBalance>>({})
const runningBalanceSampleNow = ref(false)
const addingRecharge = ref(false)
const balanceOverview = ref<UpstreamBalanceConsumptionOverview | null>(null)
const balanceDetailsOpen = ref(false)
const selectedBalanceProviderSlug = ref('')
const activeBalanceRecordTab = ref<'samples' | 'history'>('samples')
const balanceSamplerForm = ref({
  enabled: false,
  interval_seconds: 3600,
  provider_amount_scales: {} as Record<string, number>,
})
const balanceSamplerDefaultScales = computed(() => Object.fromEntries(
  providers.value.map(provider => [provider.slug, defaultBalanceSamplerScaleForProvider(provider.slug)])
))
const healthGuardConfig = ref<UpstreamAccountHealthGuardConfig | null>(null)
const healthGuardRecords = ref<UpstreamAccountHealthGuardRunRecord[]>([])
const healthGuardForm = ref<HealthGuardForm>(defaultHealthGuardForm())
const healthGuardIgnoredInput = ref('')
const healthGuardIgnoredAccounts = ref<Record<number, HealthGuardIgnoredAccountSummary>>({})
const rechargeForm = ref({
  amount: null as number | null,
  note: '',
})

const showKeysDialog = ref(false)
const keysProvider = ref<UpstreamProviderConfig | null>(null)
const keysLoadingSlug = ref<string | null>(null)
const keysItems = ref<UpstreamProviderKey[]>([])
const keysWarnings = ref<string[]>([])

const showDeleteDialog = ref(false)
const deletingProvider = ref<UpstreamProviderConfig | null>(null)

const form = reactive<UpstreamProviderConfig>({
  type: 'sub2api',
  slug: '',
  name: '',
  sort_order: 0,
  enabled: true,
  is_default: false,
  base_url: '',
  login_url: '',
  api_keys_url: '',
  groups_url: '',
  available_groups_url: '',
  balance_url: '',
  usage_cost_url: '',
  email: '',
  username: '',
  password: '',
  account_name_prefix: '',
  temp_disable_minutes: 0,
  account_rate_multiplier_scale: 1,
})

const optionalColumnOptions = computed(() => [
  { key: 'base_url', label: t('admin.upstreamProviders.columns.baseUrl') },
  { key: 'auth', label: t('admin.upstreamProviders.columns.auth') },
])

const typeFilterOptions = computed<SelectOption[]>(() => [
  { value: '', label: t('admin.upstreamProviders.allTypes') },
  { value: 'sub2api', label: 'Sub2API' },
  { value: 'newapi', label: 'NewAPI' },
])

const enabledFilterOptions = computed<SelectOption[]>(() => [
  { value: '', label: t('admin.upstreamProviders.allStatus') },
  { value: 'enabled', label: t('common.enabled') },
  { value: 'disabled', label: t('common.disabled') },
])
const healthGuardPlatformOptions = computed(() => [
  { value: 'anthropic', label: 'Anthropic', placeholder: 'claude-3-5-haiku-latest' },
  { value: 'openai', label: 'OpenAI', placeholder: 'gpt-4o-mini' },
  { value: 'gemini', label: 'Gemini', placeholder: 'gemini-2.5-flash' },
  { value: 'antigravity', label: 'Antigravity', placeholder: 'gemini-3-flash' },
  { value: 'grok', label: 'Grok', placeholder: 'grok-3-mini' },
])
const activeProviderFilterCount = computed(() => {
  let count = 0
  if (typeFilter.value) count += 1
  if (enabledFilter.value) count += 1
  if (activeQuickProviderFilter.value !== 'all') count += 1
  return count
})
const quickProviderFilterOptions = computed<Array<{ key: QuickProviderFilterKey; label: string; count: number; tone?: string }>>(() => {
  const list = providers.value
  return [
    { key: 'all', label: t('admin.upstreamProviders.quickFilterAll'), count: list.length },
    {
      key: 'enabled',
      label: t('admin.upstreamProviders.quickFilterEnabled'),
      count: list.filter(provider => provider.enabled).length,
      tone: 'success',
    },
    {
      key: 'disabled',
      label: t('admin.upstreamProviders.quickFilterDisabled'),
      count: list.filter(provider => !provider.enabled).length,
      tone: 'muted',
    },
    {
      key: 'default',
      label: t('admin.upstreamProviders.quickFilterDefault'),
      count: list.filter(provider => provider.is_default).length,
      tone: 'info',
    },
    {
      key: 'balance_anomaly',
      label: t('admin.upstreamProviders.quickFilterBalanceAnomaly'),
      count: list.filter(provider => providerHasBalanceAnomaly(provider.slug)).length,
      tone: 'danger',
    },
  ]
})

const baseColumns = computed<Column[]>(() => [
  { key: 'homepage', label: t('admin.upstreamProviders.columns.homepage'), class: 'upstream-homepage-column' },
  { key: 'name', label: t('admin.upstreamProviders.columns.name'), class: 'upstream-name-column' },
  { key: 'enabled', label: t('admin.upstreamProviders.enabled'), class: 'upstream-enabled-column' },
  { key: 'sort_order', label: t('admin.upstreamProviders.columns.sortOrder'), class: 'upstream-sort-order-column' },
  { key: 'prefix', label: t('admin.upstreamProviders.columns.prefix'), class: 'upstream-prefix-column' },
  { key: 'rate_scale', label: t('admin.upstreamProviders.columns.rateScale'), class: 'upstream-numeric-column' },
  { key: 'balance', label: t('admin.upstreamProviders.columns.balance'), sortable: true, class: 'upstream-numeric-column' },
  { key: 'today_consumption', label: t('admin.upstreamProviders.columns.todayCost'), sortable: true, class: 'upstream-numeric-column' },
])

const optionalColumns = computed<Record<string, Column>>(() => ({
  base_url: { key: 'base_url', label: t('admin.upstreamProviders.columns.baseUrl'), class: 'upstream-url-column' },
  auth: { key: 'auth', label: t('admin.upstreamProviders.columns.auth'), class: 'upstream-auth-column' },
}))

const columns = computed<Column[]>(() => {
  const secondaryColumns = visibleOptionalColumns.value
    .map(key => optionalColumns.value[key])
    .filter((column): column is Column => Boolean(column))

  return [
    ...baseColumns.value,
    ...secondaryColumns,
    { key: 'actions', label: t('common.actions'), class: 'upstream-actions-column' },
  ]
})

const filteredProviders = computed<UpstreamProviderTableRow[]>(() => {
  const keyword = searchQuery.value.trim().toLowerCase()
  return providers.value
    .filter((provider) => {
      if (activeQuickProviderFilter.value === 'enabled' && !provider.enabled) return false
      if (activeQuickProviderFilter.value === 'disabled' && provider.enabled) return false
      if (activeQuickProviderFilter.value === 'default' && !provider.is_default) return false
      if (activeQuickProviderFilter.value === 'balance_anomaly' && !providerHasBalanceAnomaly(provider.slug)) return false
      if (typeFilter.value && provider.type !== typeFilter.value) return false
      if (enabledFilter.value === 'enabled' && !provider.enabled) return false
      if (enabledFilter.value === 'disabled' && provider.enabled) return false
      if (!keyword) return true
      return [
        provider.name,
        provider.slug,
        provider.type,
        provider.base_url,
        provider.api_keys_url,
        provider.login_url,
        provider.groups_url,
        provider.available_groups_url,
        provider.balance_url,
        provider.usage_cost_url,
      ]
        .filter(Boolean)
        .some((value) => String(value).toLowerCase().includes(keyword))
    })
    .map(provider => ({
      ...provider,
      balance: providerBalanceForSort(provider.slug),
      today_consumption: todayConsumptionForProvider(provider.slug),
    }))
})

function providerRowClass(provider: UpstreamProviderConfig) {
  const classes = ['provider-mobile-row-card']
  if (!provider.enabled) classes.push('provider-disabled-row')
  if (isExpanded(provider.slug)) classes.push('provider-mobile-row-expanded')
  if (providerHasBalanceAnomaly(provider.slug)) classes.push('provider-balance-anomaly-row')
  return classes
}

const urlOptions = computed(() => ({
  base_url: uniqueProviderURLs('base_url'),
  api_keys_url: uniqueProviderURLs('api_keys_url'),
  login_url: uniqueProviderURLs('login_url'),
  groups_url: uniqueProviderURLs('groups_url'),
  available_groups_url: uniqueProviderURLs('available_groups_url'),
  balance_url: uniqueProviderURLs('balance_url'),
  usage_cost_url: uniqueProviderURLs('usage_cost_url'),
}))

const keysDialogTitle = computed(() => {
  const name = keysProvider.value?.name || ''
  return name ? t('admin.upstreamProviders.keysDialogTitleWithName', { name }) : t('admin.upstreamProviders.keysDialogTitle')
})

const deleteMessage = computed(() => {
  const name = deletingProvider.value?.name || ''
  return t('admin.upstreamProviders.deleteConfirm', { name })
})
const balanceSummaries = computed<Record<string, UpstreamBalanceProviderSummary>>(() => balanceOverview.value?.summaries || {})
const balanceRows = computed<UpstreamBalanceDailyRow[]>(() => balanceOverview.value?.rows || [])
const balanceSnapshots = computed<UpstreamBalanceSnapshot[]>(() => balanceOverview.value?.snapshots || [])
const selectedBalanceSummary = computed(() => selectedBalanceProviderSlug.value ? balanceSummaries.value[selectedBalanceProviderSlug.value] : undefined)
const selectedBalanceRows = computed(() => balanceRows.value.filter(row => row.provider_slug === selectedBalanceProviderSlug.value))
const selectedBalanceSnapshots = computed(() => balanceSnapshots.value
  .filter(snapshot => snapshot.provider_slug === selectedBalanceProviderSlug.value)
  .slice()
  .sort((left, right) => new Date(right.captured_at).getTime() - new Date(left.captured_at).getTime()))
const totalProviderBalance = computed(() => {
  const seen = new Set<string>()
  let total = 0

  for (const [slug, balance] of Object.entries(providerBalances.value)) {
    const amount = Number(balance.balance)
    if (Number.isFinite(amount)) {
      seen.add(slug)
      total += amount
    }
  }

  for (const summary of Object.values(balanceSummaries.value)) {
    const slug = summary.provider_slug
    if (slug && seen.has(slug)) continue
    const amount = Number(summary.current_balance)
    if (Number.isFinite(amount)) total += amount
  }

  return total
})
const totalTodayConsumption = computed(() => {
  const seen = new Set<string>()
  let total = 0

  for (const [slug, balance] of Object.entries(providerBalances.value)) {
    const amount = Number(balance.today_cost)
    if (Number.isFinite(amount)) {
      seen.add(slug)
      total += amount
    }
  }

  for (const summary of Object.values(balanceSummaries.value)) {
    const slug = summary.provider_slug
    if (slug && seen.has(slug)) continue
    const amount = Number(summary.today_consumption)
    if (Number.isFinite(amount)) total += amount
  }

  return total
})
const selectedBalanceScale = computed(() => {
  const configured = balanceSamplerForm.value.provider_amount_scales[selectedBalanceProviderSlug.value]
  if (Number(configured) > 0) return Number(configured)
  if (Number(selectedBalanceSummary.value?.amount_scale) > 0) return Number(selectedBalanceSummary.value?.amount_scale)
  return 1
})
const selectedBalanceProviderLabel = computed(() => {
  const slug = selectedBalanceProviderSlug.value
  if (!slug) return '-'
  const provider = providers.value.find(item => item.slug === slug)
  const summary = selectedBalanceSummary.value
  return provider?.name || summary?.provider_name || slug
})
const latestHealthGuardRecord = computed(() => healthGuardRecords.value[0])
const latestHealthGuardItems = computed<UpstreamAccountHealthGuardRunItem[]>(() => latestHealthGuardRecord.value?.items || [])
const healthGuardAdjustmentLogs = computed<HealthGuardAdjustmentLogItem[]>(() => {
  const logs: HealthGuardAdjustmentLogItem[] = []
  for (const record of healthGuardRecords.value) {
    for (const item of record.items || []) {
      if (item.action !== 'disabled' && item.action !== 'recovered') continue
      logs.push({ record, item })
    }
  }
  return logs
    .sort((a, b) => healthGuardAdjustmentTimestamp(b) - healthGuardAdjustmentTimestamp(a))
    .slice(0, 30)
})
const healthGuardSkipReasons = computed<UpstreamAccountHealthGuardSkipReason[]>(() => latestHealthGuardRecord.value?.summary?.skip_reasons || [])
const healthGuardSkippedCount = computed(() => healthGuardSkipReasons.value.reduce((sum, reason) => sum + Math.max(0, Number(reason.count) || 0), 0))
const healthGuardAdjustmentFilterOptions = computed<Array<{ key: HealthGuardAdjustmentFilterKey; label: string; count: number }>>(() => {
  const logs = healthGuardAdjustmentLogs.value
  const latestRecordID = latestHealthGuardRecord.value?.id
  return [
    { key: 'all', label: t('admin.upstreamProviders.healthGuardFilterAll'), count: logs.length },
    { key: 'latest', label: t('admin.upstreamProviders.healthGuardFilterLatestRun'), count: latestRecordID ? logs.filter(log => log.record.id === latestRecordID).length : 0 },
    { key: 'disabled', label: t('admin.upstreamProviders.healthGuardDisabled'), count: logs.filter(log => log.item.action === 'disabled').length },
    { key: 'recovered', label: t('admin.upstreamProviders.healthGuardRecovered'), count: logs.filter(log => log.item.action === 'recovered').length },
  ]
})
const filteredHealthGuardAdjustmentLogs = computed(() => {
  const filter = activeHealthGuardAdjustmentFilter.value
  if (filter === 'all') return healthGuardAdjustmentLogs.value
  if (filter === 'latest') {
    const latestRecordID = latestHealthGuardRecord.value?.id
    return latestRecordID ? healthGuardAdjustmentLogs.value.filter(log => log.record.id === latestRecordID) : []
  }
  return healthGuardAdjustmentLogs.value.filter(log => log.item.action === filter)
})
const healthGuardResultFilterOptions = computed<Array<{ key: HealthGuardResultFilterKey; label: string; count: number }>>(() => {
  const items = latestHealthGuardItems.value
  return [
    { key: 'all', label: t('admin.upstreamProviders.healthGuardFilterAll'), count: items.length },
    { key: 'failed', label: t('admin.upstreamProviders.healthGuardFailed'), count: items.filter(item => item.status === 'failed').length },
    { key: 'slow', label: t('admin.upstreamProviders.healthGuardSlow'), count: items.filter(item => item.status === 'slow').length },
    { key: 'healthy', label: t('admin.upstreamProviders.healthGuardHealthy'), count: items.filter(item => item.status === 'healthy').length },
    { key: 'changed', label: t('admin.upstreamProviders.healthGuardAdjusted'), count: items.filter(item => item.action === 'disabled' || item.action === 'recovered').length },
    { key: 'disabled', label: t('admin.upstreamProviders.healthGuardActionDisabled'), count: items.filter(item => item.action === 'disabled').length },
    { key: 'recovered', label: t('admin.upstreamProviders.healthGuardActionRecovered'), count: items.filter(item => item.action === 'recovered').length },
    { key: 'unchanged', label: t('admin.upstreamProviders.healthGuardActionNone'), count: items.filter(healthGuardItemUnchanged).length },
    { key: 'skipped', label: t('admin.upstreamProviders.healthGuardSkipped'), count: items.filter(item => item.status === 'skipped').length },
  ]
})
const filteredLatestHealthGuardItems = computed(() => {
  const filter = activeHealthGuardResultFilter.value
  if (filter === 'all') return latestHealthGuardItems.value
  if (filter === 'changed') return latestHealthGuardItems.value.filter(item => item.action === 'disabled' || item.action === 'recovered')
  if (filter === 'disabled' || filter === 'recovered') return latestHealthGuardItems.value.filter(item => item.action === filter)
  if (filter === 'unchanged') return latestHealthGuardItems.value.filter(healthGuardItemUnchanged)
  return latestHealthGuardItems.value.filter(item => item.status === filter)
})
const healthGuardLastRunText = computed(() => {
  const lastRun = healthGuardConfig.value?.last_run_at
  return lastRun ? formatDateTime(lastRun) : t('admin.upstreamProviders.healthGuardNeverRun')
})
const healthGuardSummaryCards = computed(() => {
  const summary = latestHealthGuardRecord.value?.summary
  return [
    { key: 'total', label: t('admin.upstreamProviders.healthGuardTotal'), value: summary?.total_accounts || 0, tone: '' },
    { key: 'checked', label: t('admin.upstreamProviders.healthGuardChecked'), value: summary?.checked_count || 0, tone: '' },
    { key: 'healthy', label: t('admin.upstreamProviders.healthGuardHealthy'), value: summary?.healthy_count || 0, tone: 'is-success' },
    { key: 'slow', label: t('admin.upstreamProviders.healthGuardSlow'), value: summary?.slow_count || 0, tone: 'is-warning' },
    { key: 'failed', label: t('admin.upstreamProviders.healthGuardFailed'), value: summary?.failed_count || 0, tone: 'is-danger' },
    { key: 'skipped', label: t('admin.upstreamProviders.healthGuardSkipped'), value: summary?.skipped_count || 0, tone: 'is-warning' },
    { key: 'disabled', label: t('admin.upstreamProviders.healthGuardDisabled'), value: summary?.disabled_count || 0, tone: 'is-danger' },
    { key: 'recovered', label: t('admin.upstreamProviders.healthGuardRecovered'), value: summary?.recovered_count || 0, tone: 'is-success' },
  ]
})
const healthGuardIgnoredIDs = computed(() => parseHealthGuardIgnoredInput(healthGuardIgnoredInput.value) || [])
const healthGuardAccountModelIDs = computed(() => (
  Object.keys(normalizeHealthGuardAccountModels(healthGuardForm.value.account_models))
    .map(id => Number(id))
    .filter(id => Number.isSafeInteger(id) && id > 0)
))
const healthGuardVisibleAccountIDs = computed(() => normalizeHealthGuardIgnoredAccountIDs([
  ...healthGuardIgnoredIDs.value,
  ...healthGuardAccountModelIDs.value,
]))
const healthGuardIgnoredAccountRows = computed<HealthGuardIgnoredAccountSummary[]>(() => (
  healthGuardIgnoredIDs.value.map(id => healthGuardIgnoredAccounts.value[id] || { id, loading: true })
))
const healthGuardAccountModelRows = computed<HealthGuardAccountModelRow[]>(() => (
  Object.entries(normalizeHealthGuardAccountModels(healthGuardForm.value.account_models))
    .map(([rawID, model]) => {
      const id = Number(rawID)
      return {
        ...(healthGuardIgnoredAccounts.value[id] || { id, loading: true }),
        id,
        model,
      }
    })
))
const healthGuardIgnoredInputInvalid = computed(() => Boolean(healthGuardIgnoredInput.value.trim()) && parseHealthGuardIgnoredInput(healthGuardIgnoredInput.value) === null)
const healthGuardIgnoredSummaryText = computed(() => {
  const count = healthGuardIgnoredIDs.value.length
  if (!count) return t('admin.upstreamProviders.healthGuardIgnoredNone')
  return t('admin.upstreamProviders.healthGuardIgnoredSummary', { count })
})
const healthGuardConfigSummary = computed(() => t('admin.upstreamProviders.healthGuardConfigSummary', {
  interval: Math.max(60, Math.floor(Number(healthGuardForm.value.interval_seconds) || 0)),
  concurrency: Math.max(1, Math.min(8, Math.floor(Number(healthGuardForm.value.concurrency) || 0))),
  ignored: healthGuardIgnoredIDs.value.length,
}))
const healthGuardIgnoredAccountOptions = computed<HealthGuardIgnoredAccountOption[]>(() => {
  const ignored = new Set(healthGuardIgnoredIDs.value)
  return healthGuardIgnoredAccountOptionsSource.value.map((account) => {
    const meta = `${healthGuardPlatformLabel(account.platform)} / #${account.id}`
    return {
      value: account.id,
      label: account.name || `#${account.id}`,
      meta,
      account,
      disabled: ignored.has(account.id),
    }
  })
})
const healthGuardAccountModelOptions = computed<HealthGuardIgnoredAccountOption[]>(() => {
  const configured = new Set(healthGuardAccountModelIDs.value)
  return healthGuardIgnoredAccountOptionsSource.value.map((account) => {
    const meta = `${healthGuardPlatformLabel(account.platform)} / #${account.id}`
    return {
      value: account.id,
      label: account.name || `#${account.id}`,
      meta,
      account,
      disabled: configured.has(account.id),
    }
  })
})

function resetForm() {
  Object.assign(form, {
    type: 'sub2api',
    slug: '',
    name: '',
    sort_order: 0,
    enabled: true,
    is_default: false,
    base_url: '',
    login_url: '',
    api_keys_url: '',
    groups_url: '',
    available_groups_url: '',
    balance_url: '',
    usage_cost_url: '',
    email: '',
    username: '',
    password: '',
    account_name_prefix: '',
    temp_disable_minutes: 0,
    account_rate_multiplier_scale: 1,
  })
}

function fillForm(provider: UpstreamProviderConfig) {
  Object.assign(form, {
    ...provider,
    login_url: provider.login_url || '',
    groups_url: provider.groups_url || '',
    available_groups_url: provider.available_groups_url || (provider.type === 'newapi' ? '' : provider.groups_url || ''),
    balance_url: provider.balance_url || '',
    usage_cost_url: provider.usage_cost_url || '',
    email: provider.email || '',
    username: provider.username || '',
    password: '',
    sort_order: Math.max(0, Math.floor(Number(provider.sort_order) || 0)),
    is_default: Boolean(provider.is_default),
    account_name_prefix: provider.account_name_prefix || '',
    temp_disable_minutes: provider.temp_disable_minutes || 0,
    account_rate_multiplier_scale: Number(provider.account_rate_multiplier_scale) > 0 ? Number(provider.account_rate_multiplier_scale) : 1,
  })
}

function buildPayload(): UpstreamProviderConfig {
  const payload: UpstreamProviderConfig = {
    type: form.type,
    slug: form.slug.trim(),
    name: form.name.trim(),
    sort_order: Math.max(0, Math.floor(Number(form.sort_order) || 0)),
    enabled: form.enabled,
    is_default: Boolean(form.is_default),
    base_url: form.base_url.trim(),
    login_url: form.login_url?.trim() || '',
    api_keys_url: form.api_keys_url.trim(),
    groups_url: form.type === 'newapi' ? form.groups_url?.trim() || '' : '',
    available_groups_url: form.available_groups_url?.trim() || '',
    balance_url: form.balance_url?.trim() || '',
    usage_cost_url: form.usage_cost_url?.trim() || '',
    email: form.email?.trim() || '',
    username: form.username?.trim() || '',
    account_name_prefix: form.account_name_prefix?.trim() || '',
    temp_disable_minutes: Number(form.temp_disable_minutes) || 0,
    account_rate_multiplier_scale: Number(form.account_rate_multiplier_scale) > 0 ? Number(form.account_rate_multiplier_scale) : 1,
  }
  const password = form.password?.trim()
  if (password) {
    payload.password = password
  }
  return payload
}

async function reload() {
  loading.value = true
  try {
    const [nextProviders, balance] = await Promise.all([
      adminAPI.upstreamProviders.list(),
      adminAPI.upstreamAccountSync.getBalanceConsumption(30),
    ])
    providers.value = nextProviders
    applyBalanceOverview(balance)
    retainBalancesForProviders(nextProviders)
    void fetchProviderBalances(nextProviders)
  } catch (err) {
    appStore.showError(extractApiErrorMessage(err, t('admin.upstreamProviders.loadFailed')))
  } finally {
    loading.value = false
  }
}

function openCreateDialog() {
  formMode.value = 'create'
  editingProvider.value = null
  resetForm()
  showFormDialog.value = true
}

function openEditDialog(provider: UpstreamProviderConfig) {
  formMode.value = 'edit'
  editingProvider.value = provider
  fillForm(provider)
  showFormDialog.value = true
}

function closeFormDialog() {
  showFormDialog.value = false
  editingProvider.value = null
}

function openBalanceSamplerDialog() {
  showBalanceSamplerDialog.value = true
}

function closeBalanceSamplerDialog() {
  showBalanceSamplerDialog.value = false
}

function updateBalanceSamplerProviderScale(providerSlug: string, scale: number) {
  balanceSamplerForm.value.provider_amount_scales[providerSlug] = scale
}

function updateHealthGuardPolicyField(
  field: 'interval_seconds' | 'max_accounts_per_run' | 'concurrency' | 'timeout_per_account_seconds' | 'failure_threshold' | 'slow_threshold' | 'recovery_threshold' | 'healthy_latency_ms',
  value: number,
) {
  healthGuardForm.value[field] = value
}

function closeHealthGuardDetailDialogs() {
  showHealthGuardSkipReasonsDialog.value = false
  showHealthGuardAdjustmentDialog.value = false
  showHealthGuardResultsDialog.value = false
}

async function openHealthGuardDialog() {
  showHealthGuardDialog.value = true
  await loadHealthGuardState()
  void loadHealthGuardIgnoredAccountOptions()
}

function closeHealthGuardDialog() {
  showHealthGuardDialog.value = false
  showHealthGuardIgnoredDialog.value = false
  healthGuardAccountModelAccountToAdd.value = null
  healthGuardAccountModelDraft.value = ''
  closeHealthGuardDetailDialogs()
}

async function openAutomationSettingsFromQuery() {
  const automation = typeof route.query.automation === 'string' ? route.query.automation : ''
  if (automation === 'balance-sampler') {
    openBalanceSamplerDialog()
  } else if (automation === 'health-guard') {
    await openHealthGuardDialog()
  } else {
    return
  }
  const query = { ...route.query }
  delete query.automation
  await router.replace({ query })
}

async function openHealthGuardIgnoredDialog() {
  showHealthGuardIgnoredDialog.value = true
  await Promise.all([
    refreshHealthGuardIgnoredAccounts(),
    loadHealthGuardIgnoredAccountOptions(),
  ])
}

function closeHealthGuardIgnoredDialog() {
  showHealthGuardIgnoredDialog.value = false
  healthGuardIgnoredAccountToAdd.value = null
}

async function submitForm() {
  submitting.value = true
  try {
    const payload = buildPayload()
    if (formMode.value === 'create') {
      await adminAPI.upstreamProviders.create(payload)
      appStore.showSuccess(t('admin.upstreamProviders.createSuccess'))
    } else {
      await adminAPI.upstreamProviders.update(editingProvider.value?.slug || payload.slug, payload)
      appStore.showSuccess(t('admin.upstreamProviders.updateSuccess'))
    }
    closeFormDialog()
    await reload()
  } catch (err) {
    appStore.showError(extractApiErrorMessage(err, t('admin.upstreamProviders.saveFailed')))
  } finally {
    submitting.value = false
  }
}

async function testDraftProvider() {
  testingDraft.value = true
  try {
    testResult.value = await adminAPI.upstreamProviders.testConfig(buildPayload())
    showTestDialog.value = true
  } catch (err) {
    appStore.showError(extractApiErrorMessage(err, t('admin.upstreamProviders.testFailed')))
  } finally {
    testingDraft.value = false
  }
}

async function testSavedProvider(provider: UpstreamProviderConfig) {
  const next = new Set(testingSlugs.value)
  next.add(provider.slug)
  testingSlugs.value = next
  try {
    testResult.value = await adminAPI.upstreamProviders.testSaved(provider.slug)
    showTestDialog.value = true
  } catch (err) {
    appStore.showError(extractApiErrorMessage(err, t('admin.upstreamProviders.testFailed')))
  } finally {
    const done = new Set(testingSlugs.value)
    done.delete(provider.slug)
    testingSlugs.value = done
  }
}

async function setDefaultProvider(provider: UpstreamProviderConfig) {
  defaultingSlug.value = provider.slug
  try {
    await adminAPI.upstreamProviders.setDefault(provider.slug)
    appStore.showSuccess(t('admin.upstreamProviders.setDefaultSuccess'))
    await reload()
  } catch (err) {
    appStore.showError(extractApiErrorMessage(err, t('admin.upstreamProviders.setDefaultFailed')))
  } finally {
    defaultingSlug.value = null
  }
}

async function toggleProviderEnabled(provider: UpstreamProviderConfig, enabled: boolean) {
  if (provider.enabled === enabled) return
  const previous = provider.enabled
  provider.enabled = enabled
  try {
    await adminAPI.upstreamProviders.update(provider.slug, {
      ...provider,
      enabled,
    })
    appStore.showSuccess(t('admin.upstreamProviders.updateSuccess'))
    await reload()
  } catch (err) {
    provider.enabled = previous
    appStore.showError(extractApiErrorMessage(err, t('admin.upstreamProviders.saveFailed')))
  }
}

async function openKeysDialog(provider: UpstreamProviderConfig) {
  keysProvider.value = provider
  keysItems.value = []
  keysWarnings.value = []
  keysLoadingSlug.value = provider.slug
  showKeysDialog.value = true
  try {
    const result = await adminAPI.upstreamProviders.getKeys(provider.slug)
    keysItems.value = result.items || []
    keysWarnings.value = result.warnings || []
  } catch (err) {
    appStore.showError(extractApiErrorMessage(err, t('admin.upstreamProviders.keysFailed')))
  } finally {
    keysLoadingSlug.value = null
  }
}

async function fetchProviderBalances(nextProviders: UpstreamProviderConfig[]) {
  await Promise.allSettled(nextProviders.map(provider => fetchProviderBalance(provider, { silent: true })))
}

async function fetchProviderBalance(provider: UpstreamProviderConfig, options: { silent?: boolean } = {}) {
  const nextLoading = new Set(balanceLoadingSlugs.value)
  nextLoading.add(provider.slug)
  balanceLoadingSlugs.value = nextLoading
  try {
    const balance = await adminAPI.upstreamProviders.getBalance(provider.slug)
    providerBalances.value = {
      ...providerBalances.value,
      [provider.slug]: balance,
    }
    if (!options.silent) {
      appStore.showSuccess(t('admin.upstreamProviders.balanceFetchSuccess'))
    }
  } catch (err) {
    if (!options.silent) {
      appStore.showError(extractApiErrorMessage(err, t('admin.upstreamProviders.balanceFailed')))
    }
  } finally {
    const done = new Set(balanceLoadingSlugs.value)
    done.delete(provider.slug)
    balanceLoadingSlugs.value = done
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
    provider_amount_scales: { ...(config.provider_amount_scales || {}) },
  }
}

function defaultHealthGuardForm(): HealthGuardForm {
  return {
    enabled: false,
    interval_seconds: 3600,
    max_accounts_per_run: 200,
    concurrency: 3,
    timeout_per_account_seconds: 90,
    failure_threshold: 3,
    slow_threshold: 3,
    recovery_threshold: 2,
    healthy_latency_ms: 15000,
    ignored_account_ids: [],
    account_models: {},
    platform_models: {},
    platform_latency_ms: {},
  }
}

function applyHealthGuardConfig(config: UpstreamAccountHealthGuardConfig) {
  const ignoredAccountIDs = normalizeHealthGuardIgnoredAccountIDs(config.ignored_account_ids)
  const accountModels = normalizeHealthGuardAccountModels(config.account_models)
  healthGuardConfig.value = config
  healthGuardForm.value = {
    ...defaultHealthGuardForm(),
    ...config,
    ignored_account_ids: ignoredAccountIDs,
    account_models: accountModels,
    platform_models: { ...(config.platform_models || {}) },
    platform_latency_ms: { ...(config.platform_latency_ms || {}) },
  }
  healthGuardIgnoredInput.value = ignoredAccountIDs.join(', ')
  healthGuardAccountModelAccountToAdd.value = null
  healthGuardAccountModelDraft.value = ''
}

async function loadHealthGuardState() {
  loadingHealthGuard.value = true
  try {
    const [config, records] = await Promise.all([
      adminAPI.upstreamAccountSync.getHealthGuardConfig(),
      adminAPI.upstreamAccountSync.getHealthGuardRecords(),
    ])
    applyHealthGuardConfig(config)
    healthGuardRecords.value = records || []
  } catch (err) {
    appStore.showError(extractApiErrorMessage(err, t('admin.upstreamProviders.healthGuardLoadFailed')))
  } finally {
    loadingHealthGuard.value = false
  }
}

function buildHealthGuardPayload(ignoredAccountIDs: number[], accountModels: Record<string, string>): UpstreamAccountHealthGuardConfig {
  const platformModels = Object.fromEntries(
    Object.entries(healthGuardForm.value.platform_models || {})
      .map(([platform, model]) => [platform, String(model || '').trim()])
      .filter(([, model]) => model)
  ) as Record<string, string>
  const platformLatency = Object.fromEntries(
    Object.entries(healthGuardForm.value.platform_latency_ms || {})
      .map(([platform, latency]) => [platform, Math.floor(Number(latency) || 0)])
      .filter(([, latency]) => Number(latency) > 0)
  ) as Record<string, number>
  return {
    ...defaultHealthGuardForm(),
    enabled: Boolean(healthGuardForm.value.enabled),
    interval_seconds: Math.max(60, Math.floor(Number(healthGuardForm.value.interval_seconds) || 0)),
    max_accounts_per_run: Math.max(1, Math.min(1000, Math.floor(Number(healthGuardForm.value.max_accounts_per_run) || 0))),
    concurrency: Math.max(1, Math.min(8, Math.floor(Number(healthGuardForm.value.concurrency) || 0))),
    timeout_per_account_seconds: Math.max(5, Math.min(300, Math.floor(Number(healthGuardForm.value.timeout_per_account_seconds) || 0))),
    failure_threshold: Math.max(1, Math.floor(Number(healthGuardForm.value.failure_threshold) || 0)),
    slow_threshold: Math.max(1, Math.floor(Number(healthGuardForm.value.slow_threshold) || 0)),
    recovery_threshold: Math.max(1, Math.floor(Number(healthGuardForm.value.recovery_threshold) || 0)),
    healthy_latency_ms: Math.max(1, Math.floor(Number(healthGuardForm.value.healthy_latency_ms) || 0)),
    ignored_account_ids: ignoredAccountIDs,
    account_models: accountModels,
    platform_models: platformModels,
    platform_latency_ms: platformLatency,
  }
}

async function saveHealthGuardConfig() {
  const ignoredAccountIDs = parseHealthGuardIgnoredInput(healthGuardIgnoredInput.value)
  if (!ignoredAccountIDs) {
    appStore.showError(t('admin.upstreamProviders.invalidHealthGuardIgnoredAccounts'))
    return
  }
  const accountModels = normalizeHealthGuardAccountModels(healthGuardForm.value.account_models)
  savingHealthGuardConfig.value = true
  try {
    const config = await adminAPI.upstreamAccountSync.updateHealthGuardConfig(buildHealthGuardPayload(ignoredAccountIDs, accountModels))
    applyHealthGuardConfig(config)
    appStore.showSuccess(t('admin.upstreamProviders.healthGuardSaved'))
    closeHealthGuardDialog()
  } catch (err) {
    appStore.showError(extractApiErrorMessage(err, t('admin.upstreamProviders.healthGuardSaveFailed')))
  } finally {
    savingHealthGuardConfig.value = false
  }
}

async function runHealthGuardNow() {
  runningHealthGuardNow.value = true
  try {
    const response = await adminAPI.upstreamAccountSync.runHealthGuardNow()
    applyHealthGuardConfig(response.config)
    healthGuardRecords.value = [response.record, ...healthGuardRecords.value.filter(record => record.id !== response.record.id)].slice(0, 50)
    appStore.showSuccess(t('admin.upstreamProviders.healthGuardRunSuccess'))
  } catch (err) {
    appStore.showError(extractApiErrorMessage(err, t('admin.upstreamProviders.healthGuardRunFailed')))
  } finally {
    runningHealthGuardNow.value = false
  }
}

function normalizeHealthGuardIgnoredAccountIDs(value: unknown): number[] {
  const raw = Array.isArray(value) ? value : []
  const seen = new Set<number>()
  for (const item of raw) {
    const id = Number(item)
    if (!Number.isSafeInteger(id) || id <= 0) continue
    seen.add(id)
  }
  return Array.from(seen).sort((a, b) => a - b)
}

function parseHealthGuardIgnoredInput(value: string): number[] | null {
  const text = String(value || '').trim()
  if (!text) return []
  const tokens = text.split(/[,\s]+/).map(token => token.trim()).filter(Boolean)
  const ids: number[] = []
  for (const token of tokens) {
    if (!/^\d+$/.test(token)) return null
    const id = Number(token)
    if (!Number.isSafeInteger(id) || id <= 0) return null
    ids.push(id)
  }
  return normalizeHealthGuardIgnoredAccountIDs(ids)
}

async function refreshHealthGuardIgnoredAccounts(ids: number[] = healthGuardVisibleAccountIDs.value) {
  const uniqueIDs = normalizeHealthGuardIgnoredAccountIDs(ids)
  if (!uniqueIDs.length) {
    healthGuardIgnoredAccounts.value = {}
    return
  }

  const current = healthGuardIgnoredAccounts.value
  const next: Record<number, HealthGuardIgnoredAccountSummary> = {}
  const missingIDs: number[] = []
  for (const id of uniqueIDs) {
    const cached = current[id]
    if (cached && !cached.loading) {
      next[id] = cached
    } else {
      next[id] = { id, loading: true }
      missingIDs.push(id)
    }
  }
  healthGuardIgnoredAccounts.value = next
  if (!missingIDs.length) return

  const entries = await Promise.allSettled(missingIDs.map(async (id) => {
    const account = await adminAPI.accounts.getById(id)
    return { id, account }
  }))
  const latestIDs = new Set(healthGuardVisibleAccountIDs.value)
  const updated = { ...healthGuardIgnoredAccounts.value }
  entries.forEach((entry, index) => {
    const id = missingIDs[index]
    if (!latestIDs.has(id)) return
    if (entry.status === 'fulfilled') {
      updated[id] = { id, account: entry.value.account }
    } else {
      updated[id] = { id, missing: true }
    }
  })
  healthGuardIgnoredAccounts.value = updated
}

async function loadHealthGuardIgnoredAccountOptions() {
  if (loadingHealthGuardIgnoredOptions.value) return
  if (healthGuardIgnoredAccountOptionsSource.value.length) return
  loadingHealthGuardIgnoredOptions.value = true
  try {
    const result = await adminAPI.accounts.list(1, 200, {
      lite: 'true',
      sort_by: 'name',
      sort_order: 'asc',
    })
    const accounts = result.items || []
    healthGuardIgnoredAccountOptionsSource.value = accounts
    const next = { ...healthGuardIgnoredAccounts.value }
    const visibleIDs = new Set(healthGuardVisibleAccountIDs.value)
    for (const account of accounts) {
      if (visibleIDs.has(account.id)) {
        next[account.id] = { id: account.id, account }
      }
    }
    healthGuardIgnoredAccounts.value = next
  } catch (err) {
    appStore.showError(extractApiErrorMessage(err, t('admin.upstreamProviders.healthGuardIgnoredLoadFailed')))
  } finally {
    loadingHealthGuardIgnoredOptions.value = false
  }
}

function addHealthGuardIgnoredAccount() {
  const id = Number(healthGuardIgnoredAccountToAdd.value)
  if (!Number.isSafeInteger(id) || id <= 0) return
  const ids = normalizeHealthGuardIgnoredAccountIDs([...healthGuardIgnoredIDs.value, id])
  healthGuardIgnoredInput.value = ids.join(', ')
  const account = healthGuardIgnoredAccountOptionsSource.value.find(item => item.id === id)
  if (account) {
    healthGuardIgnoredAccounts.value = {
      ...healthGuardIgnoredAccounts.value,
      [id]: { id, account },
    }
  }
  healthGuardIgnoredAccountToAdd.value = null
}

function removeHealthGuardIgnoredAccount(id: number) {
  const ids = healthGuardIgnoredIDs.value.filter(item => item !== id)
  healthGuardIgnoredInput.value = ids.join(', ')
}

function addHealthGuardAccountModel() {
  const id = Number(healthGuardAccountModelAccountToAdd.value)
  const model = healthGuardAccountModelDraft.value.trim()
  if (!Number.isSafeInteger(id) || id <= 0 || !model) return

  healthGuardForm.value.account_models = normalizeHealthGuardAccountModels({
    ...(healthGuardForm.value.account_models || {}),
    [String(id)]: model,
  })
  const account = healthGuardIgnoredAccountOptionsSource.value.find(item => item.id === id)
  if (account) {
    healthGuardIgnoredAccounts.value = {
      ...healthGuardIgnoredAccounts.value,
      [id]: { id, account },
    }
  }
  healthGuardAccountModelAccountToAdd.value = null
  healthGuardAccountModelDraft.value = ''
}

function updateHealthGuardAccountModel(id: number, model: string) {
  if (!Number.isSafeInteger(id) || id <= 0) return
  healthGuardForm.value.account_models = normalizeHealthGuardAccountModels({
    ...(healthGuardForm.value.account_models || {}),
    [String(id)]: model,
  })
}

function removeHealthGuardAccountModel(id: number) {
  const next = { ...(healthGuardForm.value.account_models || {}) }
  delete next[String(id)]
  healthGuardForm.value.account_models = normalizeHealthGuardAccountModels(next)
}

function healthGuardIgnoredAccountName(row: HealthGuardIgnoredAccountSummary) {
  if (row.loading) return t('admin.upstreamProviders.healthGuardIgnoredAccountLoading')
  if (row.account?.name) return row.account.name
  return t('admin.upstreamProviders.healthGuardIgnoredAccountMissing')
}

function healthGuardIgnoredAccountPlatform(row: HealthGuardIgnoredAccountSummary) {
  if (row.account?.platform) return healthGuardPlatformLabel(row.account.platform)
  return '-'
}

function healthGuardAccountModelName(row: HealthGuardAccountModelRow) {
  return healthGuardIgnoredAccountName(row)
}

function healthGuardAccountModelMeta(row: HealthGuardAccountModelRow) {
  if (row.account?.platform) return `${healthGuardPlatformLabel(row.account.platform)} / #${row.id}`
  return `#${row.id}`
}

function healthGuardIgnoredAccountStatusLabel(row: HealthGuardIgnoredAccountSummary) {
  if (row.loading) return t('common.loading')
  if (!row.account) return t('admin.upstreamProviders.healthGuardIgnoredAccountMissingStatus')
  if (row.account.status === 'active') return t('common.active')
  if (row.account.status === 'inactive') return t('common.inactive')
  if (row.account.status === 'error') return t('admin.upstreamProviders.healthGuardFailed')
  return row.account.status || '-'
}

function healthGuardIgnoredAccountStatusClass(row: HealthGuardIgnoredAccountSummary) {
  if (row.loading) return 'record-status-muted'
  if (!row.account) return 'record-status-error'
  if (row.account.status === 'active') return 'record-status-success'
  if (row.account.status === 'inactive') return 'record-status-muted'
  if (row.account.status === 'error') return 'record-status-error'
  return 'record-status-muted'
}

function normalizeHealthGuardAccountModels(value: unknown): Record<string, string> {
  if (!value || typeof value !== 'object' || Array.isArray(value)) return {}
  const out: Record<string, string> = {}
  for (const [rawID, rawModel] of Object.entries(value as Record<string, unknown>)) {
    const id = Number(rawID)
    const model = String(rawModel || '').trim()
    if (!Number.isSafeInteger(id) || id <= 0 || !model) continue
    out[String(id)] = model
  }
  return Object.fromEntries(Object.entries(out).sort(([a], [b]) => Number(a) - Number(b)))
}

function defaultBalanceSamplerScaleForProvider(providerSlug: string) {
  const configured = balanceSamplerForm.value.provider_amount_scales[providerSlug]
  if (Number(configured) > 0) return Number(configured)
  const summaryScale = balanceSummaries.value[providerSlug]?.amount_scale
  if (Number(summaryScale) > 0) return Number(summaryScale)
  return 1
}

async function saveBalanceSamplerConfig() {
  savingBalanceSamplerConfig.value = true
  try {
    const payload = {
      enabled: Boolean(balanceSamplerForm.value.enabled),
      interval_seconds: Math.max(60, Math.floor(Number(balanceSamplerForm.value.interval_seconds) || 0)),
      provider_amount_scales: Object.fromEntries(
        providers.value
          .map(provider => {
            const raw = balanceSamplerForm.value.provider_amount_scales[provider.slug]
            const fallback = defaultBalanceSamplerScaleForProvider(provider.slug)
            const parsed = Number(raw)
            const scale = Number.isFinite(parsed) && parsed > 0 ? parsed : fallback
            return [provider.slug, scale]
          })
      ),
    }
    const config = await adminAPI.upstreamAccountSync.updateBalanceSamplerConfig(payload)
    applyBalanceSamplerConfig(config)
    appStore.showSuccess(t('admin.upstreamProviders.balanceSamplerSaved'))
    closeBalanceSamplerDialog()
  } catch (err) {
    appStore.showError(extractApiErrorMessage(err, t('admin.upstreamProviders.balanceSamplerSaveFailed')))
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
    appStore.showSuccess(t('admin.upstreamProviders.balanceSampleSuccess'))
  } catch (err) {
    appStore.showError(extractApiErrorMessage(err, t('admin.upstreamProviders.balanceSampleFailed')))
  } finally {
    runningBalanceSampleNow.value = false
  }
}

async function addBalanceRecharge() {
  if (!selectedBalanceProviderSlug.value) return
  const amount = Number(rechargeForm.value.amount)
  if (!Number.isFinite(amount) || amount <= 0) {
    appStore.showError(t('admin.upstreamProviders.invalidRechargeAmount'))
    return
  }
  addingRecharge.value = true
  try {
    await adminAPI.upstreamAccountSync.addBalanceRecharge({
      provider_slug: selectedBalanceProviderSlug.value,
      amount,
      amount_scale: selectedBalanceScale.value,
      note: rechargeForm.value.note.trim() || undefined,
      occurred_at: new Date().toISOString(),
    })
    rechargeForm.value = { amount: null, note: '' }
    const balance = await adminAPI.upstreamAccountSync.getBalanceConsumption(30)
    applyBalanceOverview(balance)
    appStore.showSuccess(t('admin.upstreamProviders.rechargeAdded'))
  } catch (err) {
    appStore.showError(extractApiErrorMessage(err, t('admin.upstreamProviders.rechargeAddFailed')))
  } finally {
    addingRecharge.value = false
  }
}

function openBalanceDetails(providerSlug: string) {
  selectedBalanceProviderSlug.value = providerSlug
  activeBalanceRecordTab.value = 'samples'
  balanceDetailsOpen.value = true
}

function closeBalanceDetails() {
  balanceDetailsOpen.value = false
}

function todayConsumptionForProvider(providerSlug: string | undefined) {
  if (!providerSlug) return undefined
  const liveCost = Number(providerBalances.value[providerSlug]?.today_cost)
  if (Number.isFinite(liveCost)) return liveCost
  return balanceSummaries.value[providerSlug]?.today_consumption
}

function providerBalanceForSort(providerSlug: string | undefined) {
  if (!providerSlug) return undefined
  const liveBalance = Number(providerBalances.value[providerSlug]?.balance)
  if (Number.isFinite(liveBalance)) return liveBalance
  return balanceSummaries.value[providerSlug]?.current_balance
}

function providerHasBalanceAnomaly(providerSlug: string | undefined) {
  if (!providerSlug) return false
  return Boolean(balanceSummaries.value[providerSlug]?.anomaly)
}

function balanceRowStatus(row: UpstreamBalanceDailyRow) {
  if (row.anomaly) return t('admin.upstreamProviders.balanceAnomaly')
  if (row.complete) return t('admin.upstreamProviders.balanceComplete')
  return t('admin.upstreamProviders.balanceIncomplete')
}

function isExpanded(providerSlug: string | undefined) {
  return Boolean(providerSlug && expandedProviderSlugs.value.has(providerSlug))
}

function isProviderDetailVisible(row: UpstreamProviderConfig | undefined) {
  return isExpanded(row?.slug)
}

function toggleExpanded(providerSlug: string | undefined) {
  if (!providerSlug) return
  const next = new Set(expandedProviderSlugs.value)
  if (next.has(providerSlug)) {
    next.delete(providerSlug)
  } else {
    next.add(providerSlug)
  }
  expandedProviderSlugs.value = next
}

type EndpointOption = {
  key: string
  label: string
  value: string
}

function endpointOptions(provider: UpstreamProviderConfig): EndpointOption[] {
  const options: EndpointOption[] = [
    {
      key: 'keys',
      label: t('admin.upstreamProviders.keysEndpointShort'),
      value: provider.api_keys_url || '',
    },
  ]

  if (provider.login_url) {
    options.push({
      key: 'login',
      label: t('admin.upstreamProviders.loginEndpointShort'),
      value: provider.login_url,
    })
  }

  if (provider.groups_url) {
    options.push({
      key: 'groups',
      label: t('admin.upstreamProviders.groupsEndpointShort'),
      value: provider.groups_url,
    })
  }

  const availableURL = availableGroupsURL(provider)
  if (availableURL && availableURL !== provider.groups_url) {
    options.push({
      key: 'available_groups',
      label: t('admin.upstreamProviders.availableGroupsEndpointShort'),
      value: availableURL,
    })
  }

  const balanceURL = balanceURLForProvider(provider)
  if (balanceURL) {
    options.push({
      key: 'balance',
      label: t('admin.upstreamProviders.balanceEndpointShort'),
      value: balanceURL,
    })
  }

  const usageCostURL = usageCostURLForProvider(provider)
  if (usageCostURL) {
    options.push({
      key: 'usage_cost',
      label: t('admin.upstreamProviders.usageCostEndpointShort'),
      value: usageCostURL,
    })
  }

  return options
}

function accountIdentity(provider: UpstreamProviderConfig) {
  return provider.username || provider.email || ''
}

function copyTitle(value: string | undefined) {
  return value ? t('admin.upstreamProviders.clickToCopy') : ''
}

function copyHint(value: string | undefined) {
  return copiedValue.value && copiedValue.value === value
    ? `${t('common.copied')} ✓`
    : t('admin.upstreamProviders.clickToCopy')
}

async function copyValue(value: string | undefined) {
  const text = String(value || '').trim()
  if (!text) return
  const success = await copyToClipboard(text, t('common.copiedToClipboard'))
  if (!success) return
  copiedValue.value = text
  if (copiedTimer) clearTimeout(copiedTimer)
  copiedTimer = setTimeout(() => {
    copiedValue.value = ''
  }, 1400)
}

function formatScale(value: number | undefined) {
  const n = Number(value)
  return Number.isFinite(n) && n > 0 ? `${n.toFixed(6).replace(/0+$/, '').replace(/\.$/, '')}x` : '-'
}

function formatMoney(value: number | undefined) {
  const n = Number(value)
  if (!Number.isFinite(n)) return '-'
  return n.toLocaleString(undefined, {
    minimumFractionDigits: 4,
    maximumFractionDigits: 4,
  })
}

function formatCompactDateTime(value: string | undefined) {
  if (!value) return '-'
  const date = new Date(value)
  if (Number.isNaN(date.getTime())) return '-'
  const month = String(date.getMonth() + 1).padStart(2, '0')
  const day = String(date.getDate()).padStart(2, '0')
  const hours = String(date.getHours()).padStart(2, '0')
  const minutes = String(date.getMinutes()).padStart(2, '0')
  return `${month}-${day} ${hours}:${minutes}`
}

function closeKeysDialog() {
  showKeysDialog.value = false
  keysProvider.value = null
  keysItems.value = []
  keysWarnings.value = []
}

function openDeleteDialog(provider: UpstreamProviderConfig) {
  deletingProvider.value = provider
  showDeleteDialog.value = true
}

async function confirmDelete() {
  if (!deletingProvider.value) return
  try {
    await adminAPI.upstreamProviders.delete(deletingProvider.value.slug)
    appStore.showSuccess(t('admin.upstreamProviders.deleteSuccess'))
    showDeleteDialog.value = false
    deletingProvider.value = null
    await reload()
  } catch (err) {
    appStore.showError(extractApiErrorMessage(err, t('admin.upstreamProviders.deleteFailed')))
  }
}

function providerTypeLabel(type: string) {
  if (type === 'newapi') return 'NewAPI'
  return 'Sub2API'
}

function providerTypeClass(type: string) {
  if (type === 'newapi') return 'type-newapi'
  return 'type-sub2api'
}

function availableGroupsURL(provider: UpstreamProviderConfig) {
  return provider.available_groups_url || (provider.type === 'newapi' ? '' : provider.groups_url || '')
}

function balanceURLForProvider(provider: UpstreamProviderConfig) {
  return provider.balance_url || ''
}

function usageCostURLForProvider(provider: UpstreamProviderConfig) {
  return provider.usage_cost_url || ''
}

function retainBalancesForProviders(nextProviders: UpstreamProviderConfig[]) {
  const slugs = new Set(nextProviders.map(provider => provider.slug))
  providerBalances.value = Object.fromEntries(
    Object.entries(providerBalances.value).filter(([slug]) => slugs.has(slug))
  )
}

function uniqueProviderURLs(field: 'base_url' | 'api_keys_url' | 'login_url' | 'groups_url' | 'available_groups_url' | 'balance_url' | 'usage_cost_url') {
  const seen = new Set<string>()
  const out: string[] = []
  for (const provider of providers.value) {
    const value = String(provider[field] || '').trim()
    if (!value || seen.has(value)) {
      continue
    }
    seen.add(value)
    out.push(value)
  }
  return out
}

function formatRate(value: number | undefined) {
  const n = Number(value)
  return Number.isFinite(n) ? `${n.toFixed(4)}x` : '-'
}

function formatBalance(value: number | undefined) {
  const n = Number(value)
  return Number.isFinite(n) ? n.toFixed(4) : '-'
}

function formatTotalMoney(value: number | undefined) {
  const n = Number(value)
  if (!Number.isFinite(n)) return '0.00'
  return n.toFixed(2)
}

function formatSortOrder(value: number | undefined) {
  const n = Number(value)
  return Number.isFinite(n) && n >= 0 ? String(Math.floor(n)) : '0'
}

function isLowBalance(value: number | undefined) {
  const n = Number(value)
  return Number.isFinite(n) && n < 10
}

function formatRateScale(value: number | undefined) {
  const n = Number(value)
  return Number.isFinite(n) && n > 0 ? `${n.toFixed(4)}x` : '1.0000x'
}

function formatLatencyMs(value: number | undefined) {
  const n = Number(value)
  if (!Number.isFinite(n) || n <= 0) return '-'
  if (n >= 1000) return `${(n / 1000).toFixed(1)}s`
  return `${Math.floor(n)}ms`
}

function healthGuardStatusLabel(status: string | undefined) {
  switch (status) {
    case 'success':
      return t('common.success')
    case 'healthy':
      return t('admin.upstreamProviders.healthGuardHealthy')
    case 'slow':
      return t('admin.upstreamProviders.healthGuardSlow')
    case 'failed':
      return t('admin.upstreamProviders.healthGuardFailed')
    case 'skipped':
      return t('admin.upstreamProviders.healthGuardSkipped')
    default:
      return status || '-'
  }
}

function healthGuardStatusClass(status: string | undefined) {
  switch (status) {
    case 'success':
    case 'healthy':
      return 'record-status-success'
    case 'slow':
      return 'record-status-warning'
    case 'failed':
      return 'record-status-error'
    default:
      return 'record-status-muted'
  }
}

function healthGuardItemClass(status: string | undefined) {
  switch (status) {
    case 'healthy':
      return 'is-healthy'
    case 'slow':
      return 'is-slow'
    case 'failed':
      return 'is-failed'
    default:
      return ''
  }
}

function healthGuardActionLabel(action: string | undefined) {
  switch (action) {
    case 'disabled':
      return t('admin.upstreamProviders.healthGuardActionDisabled')
    case 'recovered':
      return t('admin.upstreamProviders.healthGuardActionRecovered')
    case 'none':
      return t('admin.upstreamProviders.healthGuardActionNone')
    default:
      return action || '-'
  }
}

function healthGuardActionClass(action: string | undefined) {
  switch (action) {
    case 'disabled':
      return 'record-status-error'
    case 'recovered':
      return 'record-status-success'
    default:
      return 'record-status-muted'
  }
}

function healthGuardItemUnchanged(item: UpstreamAccountHealthGuardRunItem) {
  return !item.action || item.action === 'none'
}

function healthGuardAdjustmentTimestamp(log: HealthGuardAdjustmentLogItem) {
  const raw = log.item.finished_at || log.record.finished_at
  const timestamp = raw ? new Date(raw).getTime() : 0
  return Number.isFinite(timestamp) ? timestamp : 0
}

function healthGuardAdjustmentTime(log: HealthGuardAdjustmentLogItem) {
  return formatDateTime(log.item.finished_at || log.record.finished_at)
}

function healthGuardSchedulableLabel(value: boolean | undefined) {
  return value
    ? t('admin.upstreamProviders.healthGuardSchedulableEnabled')
    : t('admin.upstreamProviders.healthGuardSchedulableDisabled')
}

function healthGuardSchedulableChange(item: UpstreamAccountHealthGuardRunItem) {
  return t('admin.upstreamProviders.healthGuardSchedulableChange', {
    before: healthGuardSchedulableLabel(item.schedulable_before),
    after: healthGuardSchedulableLabel(item.schedulable_after),
  })
}

function healthGuardPlatformLabel(platform: string | undefined) {
  const normalized = String(platform || '').toLowerCase()
  return healthGuardPlatformOptions.value.find(option => option.value === normalized)?.label || platform || '-'
}

function healthGuardSkipReasonLabel(reason: string | undefined) {
  switch (reason) {
    case 'account_ignored':
      return t('admin.upstreamProviders.healthGuardSkipReasonAccountIgnored')
    case 'account_disabled':
      return t('admin.upstreamProviders.healthGuardSkipReasonAccountDisabled')
    case 'missing_provider_slug':
      return t('admin.upstreamProviders.healthGuardSkipReasonMissingProviderSlug')
    case 'provider_disabled':
      return t('admin.upstreamProviders.healthGuardSkipReasonProviderDisabled')
    case 'provider_not_found':
      return t('admin.upstreamProviders.healthGuardSkipReasonProviderNotFound')
    default:
      return t('admin.upstreamProviders.healthGuardSkipReasonUnknown')
  }
}

function healthGuardSkipReasonSamples(reason: UpstreamAccountHealthGuardSkipReason) {
  const samples = reason.sample_accounts || []
  return samples
    .map((account) => {
      const name = account.account_name || `#${account.account_id}`
      const platform = healthGuardPlatformLabel(account.platform)
      const provider = account.provider_slug ? ` / ${account.provider_slug}` : ''
      return `${name} / ${platform}${provider}`
    })
    .join(', ')
}

function testStages(result: UpstreamProviderTestResult) {
  const stages: Array<{ key: string; label: string; stage: UpstreamProviderTestStage }> = [
    { key: 'login', label: t('admin.upstreamProviders.stageLogin'), stage: result.login },
    { key: 'keys', label: t('admin.upstreamProviders.stageKeys'), stage: result.keys },
  ]
  if (result.groups) {
    stages.push({ key: 'groups', label: t('admin.upstreamProviders.stageGroups'), stage: result.groups })
  }
  return stages
}

watch(
  [showHealthGuardDialog, () => healthGuardVisibleAccountIDs.value.join(',')],
  ([visible]) => {
    if (!visible) return
    void refreshHealthGuardIgnoredAccounts(healthGuardVisibleAccountIDs.value)
  }
)

onMounted(async () => {
  await reload()
  await openAutomationSettingsFromQuery()
})
</script>

<style scoped>
.upstream-providers-page {
  @apply space-y-6;
}

.upstream-providers-page :deep(.table-page-layout) {
  height: auto;
  min-height: calc(100vh - 64px - 4rem);
}

.upstream-providers-page :deep(.layout-section-scrollable) {
  overflow: visible;
}

.upstream-balance-charts-section {
  @apply min-w-0;
}

.upstream-toolbar {
  @apply flex min-h-20 flex-wrap items-center gap-4 rounded-lg border border-gray-200 bg-white px-4 py-3 shadow-sm dark:border-dark-700 dark:bg-dark-800;
}

.upstream-toolbar-left {
  @apply flex flex-wrap items-center gap-3;
}

.upstream-toolbar-actions {
  @apply flex flex-wrap items-center gap-2;
}

.upstream-toolbar-title {
  @apply whitespace-nowrap text-sm font-medium text-gray-950 dark:text-white;
}

.upstream-toolbar-action {
  @apply h-9 rounded-md px-4 text-sm;
}

.upstream-sample-action {
  @apply inline-flex items-center gap-2 border-emerald-200 bg-emerald-50 text-emerald-700 hover:border-emerald-300 hover:bg-emerald-100 dark:border-emerald-800 dark:bg-emerald-950/40 dark:text-emerald-300 dark:hover:bg-emerald-900/50;
}

.upstream-sampler-settings-action {
  @apply inline-flex items-center gap-2;
}

.upstream-health-run-action {
  @apply inline-flex items-center gap-2 border-sky-200 bg-sky-50 text-sky-700 hover:border-sky-300 hover:bg-sky-100 dark:border-sky-800 dark:bg-sky-950/40 dark:text-sky-300 dark:hover:bg-sky-900/50;
}

.upstream-health-settings-action {
  @apply inline-flex items-center gap-2;
}

.upstream-toolbar-filters {
  @apply flex flex-1 flex-wrap items-center justify-end gap-3;
}

.upstream-search-row {
  @apply flex min-w-0 items-center gap-2;
}

.upstream-filter-controls {
  @apply flex flex-wrap items-center gap-3;
}

.upstream-filter-toggle {
  display: none;
}

.upstream-compact-input {
  @apply h-9 rounded-md text-sm;
}

.upstream-filter-select {
  @apply w-40;
}

.upstream-quick-tags {
  @apply flex flex-wrap items-center gap-2;
}

.upstream-quick-tag {
  @apply inline-flex h-8 items-center gap-2 rounded-md border border-gray-200 bg-white px-3 text-xs font-semibold text-gray-600 transition-colors hover:border-primary-400 hover:text-primary-600 dark:border-dark-600 dark:bg-dark-800/60 dark:text-gray-300 dark:hover:border-primary-500 dark:hover:text-primary-300;
}

.upstream-quick-tag strong {
  @apply inline-flex min-w-5 items-center justify-center rounded-full bg-gray-100 px-1.5 font-mono text-[11px] text-gray-500 dark:bg-dark-700 dark:text-gray-300;
}

.upstream-quick-tag.active {
  border-color: #00B42A;
  background: #E8FFEA;
  color: #008A22;
}

.upstream-quick-tag.active strong {
  background: rgba(0, 180, 42, 0.14);
  color: #007A1D;
}

.upstream-quick-tag-danger.active {
  border-color: #FCA5A5;
  background: #FEF2F2;
  color: #DC2626;
}

.upstream-quick-tag-muted.active {
  border-color: #CBD5E1;
  background: #F8FAFC;
  color: #475569;
}

.upstream-quick-tag-info.active {
  border-color: #BFDBFE;
  background: #EFF6FF;
  color: #2563EB;
}

.upstream-toolbar-right {
  @apply flex items-center justify-end gap-7;
}

.upstream-total {
  @apply min-w-[5.5rem] whitespace-nowrap;
}

.upstream-total span {
  @apply mb-1 block text-xs text-gray-500 dark:text-gray-400;
}

.upstream-total strong {
  @apply block text-lg font-bold leading-none text-gray-950 dark:text-white;
}

.upstream-total strong.is-cost {
  @apply text-orange-600 dark:text-orange-300;
}

.action-button {
  @apply inline-flex h-7 items-center gap-1 rounded border border-gray-200 bg-white px-2 text-xs text-gray-500 transition-colors disabled:cursor-not-allowed disabled:opacity-50 hover:border-primary-300 hover:bg-blue-50 hover:text-primary-600 dark:border-dark-600 dark:bg-dark-800 dark:text-gray-300 dark:hover:bg-dark-700 dark:hover:text-primary-300;
}

.action-button-group {
  @apply flex min-w-[18rem] flex-wrap items-center justify-center gap-1.5;
}

.provider-mobile-detail-toggle {
  display: none;
}

.action-danger {
  @apply hover:border-red-300 hover:bg-red-50 hover:text-red-600 dark:hover:bg-red-900/20 dark:hover:text-red-300;
}

.homepage-button {
  @apply inline-flex h-7 w-7 items-center justify-center rounded text-gray-400 transition-colors hover:bg-blue-50 hover:text-blue-600 dark:text-gray-500 dark:hover:bg-dark-700 dark:hover:text-primary-300;
}

.homepage-button span {
  @apply sr-only;
}

.homepage-control-cell {
  @apply flex min-w-[4.5rem] items-center justify-center gap-3;
}

.expand-toggle {
  @apply inline-flex h-7 w-7 shrink-0 items-center justify-center rounded text-gray-400 transition-colors hover:bg-blue-50 hover:text-blue-600 dark:text-gray-500 dark:hover:bg-dark-700 dark:hover:text-primary-300;
}

.expand-toggle-icon {
  @apply transition-transform duration-150;
}

.expand-toggle-icon.is-expanded {
  @apply rotate-90;
}

.column-settings-panel {
  @apply absolute right-0 z-30 mt-2 w-52 rounded-md border border-gray-200 bg-white p-2 shadow-lg dark:border-dark-600 dark:bg-dark-800;
}

.column-settings-option {
  @apply flex cursor-pointer items-center gap-2 rounded px-2 py-2 text-xs text-gray-700 hover:bg-gray-50 dark:text-gray-200 dark:hover:bg-dark-700;
}

.column-settings-button {
  @apply inline-flex h-8 w-8 items-center justify-center rounded border border-gray-200 bg-white text-gray-400 transition-colors hover:border-primary-500 hover:text-primary-600 dark:border-dark-600 dark:bg-dark-800 dark:text-gray-500 dark:hover:text-primary-300;
}

.provider-name-card {
  @apply mx-auto min-w-[14rem] max-w-[18rem] space-y-2 text-left;
}

.provider-title-line {
  @apply flex min-w-0 flex-wrap items-center gap-1.5;
}

.provider-name {
  @apply max-w-[9rem] truncate text-sm font-bold text-gray-950 dark:text-white;
}

.provider-type-tag,
.provider-default-tag {
  @apply inline-flex h-6 items-center gap-1 rounded px-1.5 text-xs leading-none;
}

.provider-type-tag.type-newapi {
  @apply bg-blue-50 text-blue-600 dark:bg-blue-950/40 dark:text-blue-300;
}

.provider-type-tag.type-sub2api {
  @apply bg-violet-50 text-violet-600 dark:bg-violet-950/40 dark:text-violet-300;
}

.provider-default-tag {
  @apply bg-blue-50 text-blue-600 dark:bg-blue-950/40 dark:text-blue-300;
}

.provider-inline-url {
  @apply block w-full max-w-full text-xs text-gray-500 hover:text-primary-600 dark:text-gray-400 dark:hover:text-primary-300;
}

.provider-inline-url span:first-child {
  @apply block truncate;
}

.provider-url-tag {
  @apply inline-flex max-w-full items-center rounded bg-gray-50 px-2 py-1 font-mono text-xs text-gray-700 dark:bg-dark-700 dark:text-gray-200;
}

.info-tag {
  @apply inline-flex items-center rounded-md px-2 py-1 text-xs font-medium ring-1;
}

.tag-auth {
  @apply bg-indigo-50 font-mono text-indigo-700 ring-indigo-200 dark:bg-indigo-950/40 dark:text-indigo-300 dark:ring-indigo-800/60;
}

.tag-rate {
  @apply bg-violet-50 font-mono text-violet-700 ring-violet-200 dark:bg-violet-950/40 dark:text-violet-300 dark:ring-violet-800/60;
}

.tag-warning {
  @apply bg-amber-50 text-amber-700 ring-amber-200 dark:bg-amber-900/20 dark:text-amber-200 dark:ring-amber-700/40;
}

.tag-success {
  @apply bg-emerald-50 text-emerald-700 ring-emerald-200 dark:bg-emerald-950/40 dark:text-emerald-300 dark:ring-emerald-800/60;
}

.tag-muted {
  @apply bg-gray-100 text-gray-600 ring-gray-200 dark:bg-dark-700 dark:text-gray-300 dark:ring-dark-600;
}

.endpoint-tag {
  @apply inline-flex max-w-full items-center gap-1 rounded-md bg-gray-100 px-2 py-1 text-xs ring-1 ring-gray-200 dark:bg-dark-700 dark:ring-dark-600;
}

.endpoint-tag span {
  @apply shrink-0 text-gray-500 dark:text-gray-400;
}

.endpoint-tag code {
  @apply truncate font-mono text-gray-700 dark:text-gray-200;
}

.prefix-value {
  @apply inline-flex max-w-[8rem] truncate font-sans text-xs text-gray-950 dark:text-white;
}

.sort-order-value {
  @apply inline-flex min-w-10 justify-center rounded bg-gray-100 px-2 py-1 font-mono text-xs font-semibold text-gray-700 dark:bg-dark-700 dark:text-gray-200;
}

.numeric-cell {
  @apply flex min-w-[7.5rem] items-center justify-center gap-2;
}

.numeric-value,
.numeric-muted {
  @apply block min-w-[6.5rem] text-center font-mono text-sm font-semibold tabular-nums text-gray-950 dark:text-gray-100;
  font-variant-numeric: tabular-nums;
}

.numeric-muted {
  @apply text-gray-400;
}

.numeric-balance {
  @apply text-lg font-bold text-teal-600 dark:text-teal-300;
}

.numeric-cost {
  @apply text-lg font-bold text-emerald-600 dark:text-emerald-300;
}

.numeric-alert {
  @apply rounded-md bg-red-50 font-bold text-red-700 ring-1 ring-red-100 dark:bg-red-950/30 dark:text-red-300 dark:ring-red-900/60;
}

.mini-icon-button {
  @apply inline-flex h-6 w-6 shrink-0 items-center justify-center rounded text-gray-400 transition-colors hover:bg-blue-50 hover:text-blue-600 disabled:cursor-not-allowed disabled:opacity-50 dark:text-gray-500 dark:hover:bg-dark-700 dark:hover:text-primary-300;
}

.balance-action-button {
  @apply inline-flex h-7 w-7 shrink-0 items-center justify-center rounded-md border border-primary-200 bg-primary-50 text-primary-600 shadow-sm transition-all hover:border-primary-300 hover:bg-primary-100 hover:shadow-md disabled:cursor-not-allowed disabled:opacity-50 dark:border-primary-800 dark:bg-primary-950/40 dark:text-primary-400 dark:hover:border-primary-700 dark:hover:bg-primary-900/50;
}

.balance-more-button {
  @apply border-violet-200 bg-violet-50 text-violet-600 hover:border-violet-300 hover:bg-violet-100 dark:border-violet-800 dark:bg-violet-950/40 dark:text-violet-400 dark:hover:border-violet-700 dark:hover:bg-violet-900/50;
}

.center-cell {
  @apply whitespace-nowrap text-center text-xs text-gray-950 dark:text-white;
}

.provider-enabled-cell {
  @apply inline-flex items-center justify-center gap-2;
}

.provider-enabled-text {
  @apply text-xs font-medium;
}

.provider-enabled-text.is-enabled {
  @apply text-emerald-600 dark:text-emerald-300;
}

.provider-enabled-text.is-disabled {
  @apply text-gray-500 dark:text-gray-400;
}

.copyable-text {
  @apply relative min-w-0 cursor-pointer border-0 text-left transition-colors;
}

.copyable-text code {
  @apply min-w-0 truncate;
}

.copy-hint {
  @apply pointer-events-none absolute -top-7 right-0 z-20 whitespace-nowrap rounded bg-gray-900 px-2 py-1 text-[11px] font-medium text-white opacity-0 shadow transition-opacity dark:bg-white dark:text-gray-900;
}

.copyable-text:hover .copy-hint,
.copyable-text:focus-visible .copy-hint {
  @apply opacity-100;
}

.provider-detail-panel {
  @apply grid min-h-[6.25rem] gap-8 border-t border-gray-100 bg-gray-50 py-4 pl-24 pr-6 md:grid-cols-[18rem_18rem_minmax(22rem,1fr)] dark:border-dark-700 dark:bg-dark-800/70;
}

.detail-column {
  @apply min-w-0 bg-transparent p-0;
}

.detail-column-wide {
  @apply min-w-0;
}

.detail-title {
  @apply mb-2 text-xs font-medium text-gray-500 dark:text-gray-400;
}

.detail-copy {
  @apply flex w-full items-center font-mono text-xs text-gray-950 hover:text-primary-600 dark:text-gray-100 dark:hover:text-primary-300;
}

.password-state-tag {
  @apply mt-2 inline-flex rounded-md px-2 py-1 text-xs font-medium ring-1;
}

.password-state-ok {
  @apply bg-emerald-50 text-emerald-700 ring-emerald-200 dark:bg-emerald-950/40 dark:text-emerald-300 dark:ring-emerald-800/60;
}

.password-state-muted {
  @apply bg-gray-100 text-gray-600 ring-gray-200 dark:bg-dark-700 dark:text-gray-300 dark:ring-dark-600;
}

.detail-endpoint-list {
  @apply space-y-1;
}

.detail-endpoint {
  @apply grid w-full grid-cols-[4rem_minmax(0,1fr)] items-center gap-2 rounded text-xs hover:text-primary-600 dark:hover:text-primary-300;
}

.detail-endpoint > span:first-child {
  @apply text-gray-500 dark:text-gray-400;
}

.detail-endpoint code {
  @apply font-mono text-gray-800 dark:text-gray-100;
}

.provider-mobile-record-cards {
  display: none;
}

.provider-mobile-record-card {
  border: 1px solid #e5e7eb;
  border-radius: 8px;
  background: #fff;
  padding: 12px;
}

.provider-mobile-record-card + .provider-mobile-record-card {
  margin-top: 8px;
}

.provider-mobile-record-card > div {
  display: grid;
  grid-template-columns: minmax(76px, auto) minmax(0, 1fr);
  gap: 10px;
  align-items: baseline;
}

.provider-mobile-record-card > div + div {
  margin-top: 8px;
}

.provider-mobile-record-card span {
  color: #64748b;
  font-size: 12px;
  font-weight: 700;
}

.provider-mobile-record-card strong {
  min-width: 0;
  overflow-wrap: anywhere;
  color: #111827;
  font-size: 13px;
  text-align: right;
}

.provider-mobile-record-empty {
  padding: 24px 12px;
  color: #94a3b8;
  font-size: 13px;
  text-align: center;
}

:global(.dark) .provider-mobile-record-card {
  border-color: #334155;
  background: #111827;
}

:global(.dark) .provider-mobile-record-card span {
  color: #94a3b8;
}

:global(.dark) .provider-mobile-record-card strong {
  color: #e5e7eb;
}

:deep(.table-wrapper table) {
  min-width: 1280px;
  border-collapse: separate;
  border-spacing: 0;
}

:deep(.table-body .data-table-row:nth-of-type(4n + 1)) {
  background-color: rgb(252 252 253);
}

:deep(.dark .table-body .data-table-row:nth-of-type(4n + 1)) {
  background-color: rgb(31 41 55 / 0.28);
}

:deep(.table-body .data-table-row:hover) {
  background-color: rgb(248 250 252);
}

:deep(.dark .table-body .data-table-row:hover) {
  background-color: rgb(31 41 55);
}

:deep(.provider-disabled-row),
:deep(.provider-disabled-row .sticky-col) {
  background-color: rgb(249 250 251);
}

:deep(.provider-disabled-row:hover),
:deep(.provider-disabled-row:hover .sticky-col) {
  background-color: rgb(243 244 246);
}

:deep(.provider-disabled-row td) {
  color: rgb(203 213 225);
}

:deep(.provider-disabled-row td > *),
:deep(.provider-disabled-row > .space-y-3) {
  filter: grayscale(1);
  opacity: 0.46;
}

:deep(.dark .provider-disabled-row),
:deep(.dark .provider-disabled-row .sticky-col) {
  background-color: rgb(17 24 39 / 0.72);
}

:deep(.dark .provider-disabled-row:hover),
:deep(.dark .provider-disabled-row:hover .sticky-col) {
  background-color: rgb(31 41 55 / 0.72);
}

:deep(.upstream-homepage-column) {
  width: 5.75rem;
  min-width: 5.75rem;
}

:deep(.upstream-name-column) {
  min-width: 16.25rem;
}

:deep(.upstream-enabled-column) {
  width: 7.5rem;
  min-width: 7.5rem;
}

:deep(.upstream-sort-order-column) {
  width: 5.5rem;
  min-width: 5.5rem;
}

:deep(.upstream-prefix-column) {
  min-width: 8.5rem;
}

:deep(th.upstream-numeric-column),
:deep(td.upstream-numeric-column) {
  min-width: 8.25rem;
}

:deep(th.upstream-actions-column),
:deep(td.upstream-actions-column) {
  min-width: 18rem;
}

:deep(.table-wrapper tbody td) {
  text-align: center;
}

:deep(.table-wrapper thead th) {
  text-align: center !important;
}

:deep(.table-wrapper thead th > div),
:deep(.table-wrapper thead th .sticky-header-cell) {
  justify-content: center !important;
}

:deep(.table-wrapper) {
  border-radius: 0.5rem;
}

:deep(.table-wrapper th) {
  height: 2.5rem;
  padding-top: 0;
  padding-bottom: 0;
  background-color: rgb(246 247 249);
  font-size: 0.75rem;
  font-weight: 600;
  color: rgb(111 127 145);
  text-transform: none;
  letter-spacing: 0;
}

:deep(.dark .table-wrapper th) {
  background-color: rgb(31 41 55);
}

:deep(.table-wrapper td) {
  height: 6rem;
  padding-top: 0.875rem;
  padding-bottom: 0.875rem;
  vertical-align: middle;
}

:deep(.data-table-detail-row td) {
  height: auto;
  padding: 0;
  overflow: visible;
}

:deep(.data-table-detail-row) {
  display: table-row;
}

.balance-cost-cell {
  @apply min-w-[12rem] leading-tight;
}

.balance-dialog {
  @apply flex max-h-[90vh] w-full max-w-5xl flex-col overflow-hidden rounded-xl bg-white shadow-2xl dark:bg-dark-800;
}

.balance-dialog-handle {
  display: none;
}

.balance-dialog-header {
  @apply flex items-start justify-between gap-4 border-b border-gray-200 px-5 py-4 dark:border-dark-600;
}

.balance-dialog-close {
  @apply shrink-0 gap-1.5;
}

.balance-dialog-body {
  @apply flex min-h-0 flex-1 flex-col gap-4 overflow-auto p-5;
}

.balance-summary-grid {
  @apply grid gap-3 md:grid-cols-4;
}

.balance-metric {
  @apply rounded-lg border border-gray-200 bg-gray-50 px-4 py-3 dark:border-dark-600 dark:bg-dark-900/40;
}

.balance-metric span {
  @apply block text-xs font-medium text-gray-500 dark:text-gray-400;
}

.balance-metric strong {
  @apply mt-1 block font-mono text-lg text-gray-950 dark:text-white;
}

.balance-config-panel,
.balance-recharge-panel {
  @apply rounded-lg border border-gray-200 bg-white p-4 dark:border-dark-600 dark:bg-dark-800/70;
}

.balance-dialog-section {
  @apply min-w-0;
}

.balance-config-panel {
  @apply flex flex-wrap items-center gap-3;
}

.balance-sampler-controls {
  @apply grid gap-3 md:grid-cols-[minmax(14rem,1fr)_16rem];
}

.balance-sampler-toggle,
.balance-sampler-interval {
  @apply flex min-h-12 items-center gap-3 rounded-lg border border-gray-200 bg-gray-50 px-4 py-3 text-sm text-gray-700 dark:border-dark-600 dark:bg-dark-900/40 dark:text-gray-200;
}

.balance-sampler-toggle span,
.balance-sampler-interval span {
  @apply font-medium;
}

.balance-sampler-interval {
  @apply justify-between;
}

.balance-sampler-interval .input {
  @apply h-9 w-28 text-right font-mono text-sm;
}

.balance-sampler-provider-panel {
  @apply rounded-lg border border-gray-200 bg-white p-4 dark:border-dark-600 dark:bg-dark-800/70;
}

.balance-sampler-provider-list {
  @apply mt-3 divide-y divide-gray-100 overflow-hidden rounded-lg border border-gray-200 dark:divide-dark-700 dark:border-dark-600;
}

.balance-sampler-provider-row {
  @apply grid min-h-14 grid-cols-[minmax(0,1fr)_9rem] items-center gap-4 bg-white px-3 py-2 dark:bg-dark-800;
}

.balance-sampler-provider-name {
  @apply min-w-0;
}

.balance-sampler-provider-name strong {
  @apply block truncate text-sm font-semibold text-gray-950 dark:text-white;
}

.balance-sampler-provider-name small {
  @apply block truncate font-mono text-xs text-gray-500 dark:text-gray-400;
}

.balance-sampler-provider-row .input {
  @apply h-9 text-right font-mono text-sm;
}

.balance-sampler-empty {
  @apply px-4 py-8 text-center text-sm text-gray-400;
}

.health-guard-dialog {
  @apply flex min-h-0 w-full flex-col gap-3;
}

.health-guard-status-panel {
  @apply grid gap-3 md:grid-cols-[minmax(0,1fr)_13rem_auto];
}

.health-guard-toggle,
.health-guard-run-state {
  @apply flex min-h-12 items-center gap-3 rounded-lg border border-gray-200 bg-gray-50 px-3 py-2 text-sm text-gray-700 dark:border-dark-600 dark:bg-dark-900/40 dark:text-gray-200;
}

.health-guard-toggle span {
  @apply min-w-0;
}

.health-guard-toggle strong,
.health-guard-run-state strong {
  @apply block text-sm font-semibold text-gray-950 dark:text-white;
}

.health-guard-toggle small,
.health-guard-run-state span {
  @apply mt-0.5 block text-xs text-gray-500 dark:text-gray-400;
}

.health-guard-run-state {
  @apply flex-wrap justify-between;
}

.health-guard-run-state em {
  @apply not-italic;
}

.health-guard-run-button {
  @apply inline-flex min-h-12 items-center justify-center gap-2 rounded-lg px-4;
}

.health-guard-content-grid {
  @apply grid min-h-0 gap-3 lg:grid-cols-[minmax(0,1.08fr)_minmax(22rem,0.92fr)];
}

.health-guard-config-column {
  @apply flex min-h-0 min-w-0 flex-col gap-3;
}

.health-guard-config-section {
  @apply rounded-lg border border-gray-200 bg-white dark:border-dark-600 dark:bg-dark-800/70;
}

.health-guard-section-toggle {
  @apply flex min-h-12 w-full items-center justify-between gap-3 rounded-lg px-3 py-2 text-left transition-colors hover:bg-gray-50 focus:outline-none focus-visible:ring-2 focus-visible:ring-primary-500/30 dark:hover:bg-dark-700/50;
}

.health-guard-section-toggle span {
  @apply min-w-0;
}

.health-guard-section-toggle strong {
  @apply block truncate text-sm font-semibold text-gray-950 dark:text-white;
}

.health-guard-section-toggle small {
  @apply mt-0.5 block truncate text-xs text-gray-500 dark:text-gray-400;
}

.health-guard-section-toggle svg {
  @apply shrink-0 transition-transform;
}

.health-guard-config-body {
  @apply space-y-3 border-t border-gray-100 p-3 dark:border-dark-700;
}

.health-guard-config-grid {
  @apply grid gap-2 md:grid-cols-4;
}

.health-guard-field {
  @apply min-w-0 rounded-lg border border-gray-200 bg-white p-2.5 dark:border-dark-600 dark:bg-dark-800/70;
}

.health-guard-field span {
  @apply mb-1.5 block truncate text-xs font-semibold text-gray-500 dark:text-gray-400;
}

.health-guard-field .input {
  @apply h-9 text-right font-mono text-sm;
}

.health-guard-field-wide {
  @apply md:col-span-4;
}

.health-guard-field small {
  @apply mt-2 block text-xs text-gray-500 dark:text-gray-400;
}

.health-guard-ignored-summary {
  @apply flex min-h-12 items-center justify-between gap-3 rounded-lg border border-gray-200 bg-gray-50 px-3 py-2 dark:border-dark-600 dark:bg-dark-900/40;
}

.health-guard-ignored-summary.is-invalid {
  @apply border-red-200 bg-red-50 dark:border-red-500/30 dark:bg-red-500/10;
}

.health-guard-ignored-summary div {
  @apply min-w-0;
}

.health-guard-ignored-summary strong {
  @apply block truncate text-sm font-semibold text-gray-950 dark:text-white;
}

.health-guard-ignored-summary small {
  @apply mt-0.5 block text-xs text-gray-500 dark:text-gray-400;
}

.health-guard-ignored-dialog {
  @apply flex max-h-[70vh] min-h-0 flex-col gap-3 overflow-hidden;
}

.health-guard-ignored-add {
  @apply grid gap-2 sm:grid-cols-[minmax(0,1fr)_auto];
}

.health-guard-ignored-select {
  @apply min-w-0;
}

.health-guard-ignored-option {
  @apply flex min-w-0 flex-col;
}

.health-guard-ignored-option strong {
  @apply truncate text-sm font-semibold text-gray-900 dark:text-white;
}

.health-guard-ignored-option small {
  @apply truncate font-mono text-xs text-gray-500 dark:text-gray-400;
}

.health-guard-ignored-option.is-selected strong,
.health-guard-ignored-option.is-selected small {
  @apply truncate;
}

.health-guard-ignored-loading {
  @apply rounded-md bg-gray-50 px-3 py-2 text-sm text-gray-500 dark:bg-dark-900/40 dark:text-gray-400;
}

.health-guard-ignored-list {
  @apply grid min-h-0 gap-1.5 overflow-auto pr-1 sm:grid-cols-2 xl:grid-cols-3;
}

.health-guard-ignored-account {
  @apply flex min-h-10 flex-wrap items-center gap-x-2 gap-y-1 rounded-md border border-gray-200 bg-gray-50 px-2.5 py-1.5 dark:border-dark-600 dark:bg-dark-900/40;
}

.health-guard-ignored-account.is-missing {
  @apply border-red-200 bg-red-50 dark:border-red-500/30 dark:bg-red-500/10;
}

.health-guard-ignored-account-main {
  @apply flex min-w-0 flex-1 items-baseline gap-1.5;
}

.health-guard-ignored-account-main strong {
  @apply min-w-0 truncate text-sm font-semibold text-gray-950 dark:text-white;
}

.health-guard-ignored-account-main code {
  @apply shrink-0 text-xs text-gray-500 dark:text-gray-400;
}

.health-guard-ignored-account-meta {
  @apply flex shrink-0 items-center gap-1.5 text-xs text-gray-500 dark:text-gray-400;
}

.health-guard-ignored-remove {
  @apply inline-flex h-7 w-7 shrink-0 items-center justify-center rounded-md text-gray-400 transition-colors hover:bg-gray-200 hover:text-gray-700 dark:hover:bg-dark-700 dark:hover:text-gray-200;
}

.health-guard-ignored-list-empty {
  @apply mt-3 rounded-lg border border-red-200 bg-red-50 px-3 py-2 text-xs font-medium text-red-700 dark:border-red-500/30 dark:bg-red-500/10 dark:text-red-300;
}

.health-guard-platform-panel,
.health-guard-account-models-panel,
.health-guard-record-panel {
  @apply rounded-lg border border-gray-200 bg-white p-3 dark:border-dark-600 dark:bg-dark-800/70;
}

.health-guard-config-body .health-guard-platform-panel,
.health-guard-config-body .health-guard-account-models-panel {
  @apply bg-gray-50/50 dark:bg-dark-900/20;
}

.health-guard-record-panel {
  @apply flex min-h-0 min-w-0 flex-col;
}

.health-guard-platform-list {
  @apply mt-2 divide-y divide-gray-100 overflow-hidden rounded-lg border border-gray-200 dark:divide-dark-700 dark:border-dark-600;
}

.health-guard-platform-row {
  @apply grid min-h-12 grid-cols-[minmax(0,1fr)_minmax(9rem,13rem)_7.5rem] items-center gap-2 bg-white px-3 py-2 dark:bg-dark-800;
}

.health-guard-platform-name {
  @apply min-w-0;
}

.health-guard-platform-name strong {
  @apply block truncate text-sm font-semibold text-gray-950 dark:text-white;
}

.health-guard-platform-name small {
  @apply block truncate text-xs text-gray-500 dark:text-gray-400;
}

.health-guard-platform-row .input {
  @apply h-9 text-sm;
}

.health-guard-platform-row .input:last-child {
  @apply text-right font-mono;
}

.health-guard-account-models-add {
  @apply mt-2 grid gap-2 lg:grid-cols-[minmax(0,1fr)_minmax(12rem,18rem)_auto];
}

.health-guard-account-model-select {
  @apply min-w-0;
}

.health-guard-account-model-input {
  @apply h-9 text-left font-mono text-sm;
}

.health-guard-account-models-add .btn {
  @apply h-9 whitespace-nowrap px-3;
}

.health-guard-account-model-list {
  @apply mt-2 divide-y divide-gray-100 overflow-hidden rounded-lg border border-gray-200 dark:divide-dark-700 dark:border-dark-600;
}

.health-guard-account-model-row {
  @apply grid min-h-11 grid-cols-[minmax(0,1fr)_minmax(12rem,18rem)_auto] items-center gap-2 bg-white px-2.5 py-1.5 dark:bg-dark-800;
}

.health-guard-account-model-row.is-missing {
  @apply bg-red-50 dark:bg-red-500/10;
}

.health-guard-account-model-account {
  @apply min-w-0;
}

.health-guard-account-model-account strong {
  @apply block truncate text-sm font-semibold text-gray-950 dark:text-white;
}

.health-guard-account-model-account small {
  @apply block truncate font-mono text-xs text-gray-500 dark:text-gray-400;
}

.health-guard-account-model-row-input {
  @apply h-8 text-left font-mono text-sm;
}

.health-guard-account-model-remove {
  @apply inline-flex h-8 w-8 items-center justify-center rounded-md text-gray-400 transition-colors hover:bg-gray-100 hover:text-gray-700 dark:hover:bg-dark-700 dark:hover:text-gray-200;
}

.health-guard-account-model-empty {
  @apply mt-2 rounded-md py-2 text-xs;
}

.health-guard-account-models-hint {
  @apply mt-2 text-xs text-gray-500 dark:text-gray-400;
}

.health-guard-record-header {
  @apply mb-2 flex items-center justify-between gap-3;
}

.health-guard-summary-grid {
  @apply grid grid-cols-4 gap-1.5 xl:grid-cols-8;
}

.health-guard-summary-card {
  @apply rounded-md border border-gray-200 bg-gray-50 px-2 py-1.5 text-center dark:border-dark-600 dark:bg-dark-900/40;
}

.health-guard-summary-card span {
  @apply block truncate text-xs font-medium text-gray-500 dark:text-gray-400;
}

.health-guard-summary-card strong {
  @apply mt-0.5 block font-mono text-base text-gray-950 dark:text-white;
}

.health-guard-summary-card.is-success strong {
  @apply text-emerald-600 dark:text-emerald-300;
}

.health-guard-summary-card.is-warning strong {
  @apply text-amber-600 dark:text-amber-300;
}

.health-guard-summary-card.is-danger strong {
  @apply text-red-600 dark:text-red-300;
}

.health-guard-detail-actions {
  @apply mt-3 grid gap-2;
}

.health-guard-detail-action {
  @apply grid min-h-16 grid-cols-[minmax(0,1fr)_auto] items-center gap-x-3 rounded-lg border border-gray-200 bg-white px-3 py-2 text-left transition-colors hover:border-primary-200 hover:bg-primary-50/40 focus:outline-none focus-visible:ring-2 focus-visible:ring-primary-500/30 disabled:cursor-not-allowed disabled:opacity-55 dark:border-dark-600 dark:bg-dark-800 dark:hover:border-primary-900/70 dark:hover:bg-primary-950/20;
}

.health-guard-detail-action span {
  @apply min-w-0 truncate text-sm font-semibold text-gray-950 dark:text-white;
}

.health-guard-detail-action strong {
  @apply row-span-2 font-mono text-lg text-gray-900 dark:text-gray-100;
}

.health-guard-detail-action small {
  @apply min-w-0 truncate text-xs text-gray-500 dark:text-gray-400;
}

.health-guard-detail-dialog {
  @apply flex max-h-[70vh] min-h-0 flex-col gap-3 overflow-hidden;
}

.health-guard-detail-dialog-large {
  @apply min-h-0 flex-1;
  max-height: none;
}

:global(.modal-content:has(.health-guard-detail-dialog-large)) {
  height: min(94vh, 980px);
  max-height: 94vh;
  height: min(94dvh, 980px);
  max-height: 94dvh;
}

:global(.modal-content:has(.health-guard-detail-dialog-large) .modal-body) {
  display: flex;
  min-height: 0;
  flex-direction: column;
  overflow: hidden;
}

.health-guard-filter-row {
  @apply flex flex-wrap gap-2;
}

.health-guard-filter-button {
  @apply inline-flex min-h-8 items-center gap-2 rounded-md border border-gray-200 bg-white px-3 text-sm font-semibold text-gray-600 transition-colors hover:border-primary-200 hover:text-primary-600 focus:outline-none focus-visible:ring-2 focus-visible:ring-primary-500/30 dark:border-dark-600 dark:bg-dark-800 dark:text-gray-300 dark:hover:border-primary-900/70 dark:hover:text-primary-300;
}

.health-guard-filter-button strong {
  @apply font-mono text-xs;
}

.health-guard-filter-button.is-active {
  @apply border-primary-200 bg-primary-50 text-primary-700 dark:border-primary-800/70 dark:bg-primary-950/40 dark:text-primary-300;
}

.health-guard-modal-list {
  @apply min-h-0 flex-1 overflow-auto pr-1;
  -webkit-overflow-scrolling: touch;
  overscroll-behavior: contain;
}

.health-guard-adjustment-panel {
  @apply mt-3 border-t border-gray-100 pt-3 dark:border-dark-700;
}

.health-guard-adjustment-header {
  @apply flex items-center justify-between gap-3;
}

.health-guard-adjustment-header span {
  @apply shrink-0 font-mono text-xs font-semibold text-gray-500 dark:text-gray-400;
}

.health-guard-adjustment-list {
  @apply mt-2 space-y-2;
}

.health-guard-adjustment-item {
  @apply rounded-lg border border-gray-200 bg-white p-2.5 dark:border-dark-600 dark:bg-dark-800;
}

.health-guard-adjustment-item.is-disabled {
  @apply border-red-200 bg-red-50/50 dark:border-red-900/70 dark:bg-red-950/20;
}

.health-guard-adjustment-item.is-recovered {
  @apply border-emerald-200 bg-emerald-50/40 dark:border-emerald-900/70 dark:bg-emerald-950/20;
}

.health-guard-adjustment-main {
  @apply flex min-w-0 flex-wrap items-baseline justify-between gap-2;
}

.health-guard-adjustment-main strong {
  @apply min-w-0 truncate text-sm font-semibold text-gray-950 dark:text-white;
}

.health-guard-adjustment-main span {
  @apply shrink-0 font-mono text-xs text-gray-500 dark:text-gray-400;
}

.health-guard-adjustment-metrics {
  @apply mt-2 flex flex-wrap items-center gap-2 text-xs text-gray-500 dark:text-gray-300;
}

.health-guard-adjustment-metrics span:not(.record-status) {
  overflow-wrap: anywhere;
}

.health-guard-adjustment-item p {
  @apply mt-2 break-words text-xs text-gray-600 dark:text-gray-300;
}

.health-guard-skip-panel {
  @apply mt-3 border-t border-gray-100 pt-3 dark:border-dark-700;
}

.health-guard-skip-list {
  @apply mt-2 grid gap-2 sm:grid-cols-2 lg:grid-cols-1 xl:grid-cols-2;
}

.health-guard-skip-item {
  @apply rounded-md bg-amber-50/70 px-3 py-2 text-sm text-gray-700 dark:bg-amber-950/20 dark:text-gray-200;
}

.health-guard-skip-item div {
  @apply flex flex-wrap items-center justify-between gap-2;
}

.health-guard-skip-item strong {
  @apply text-sm font-semibold text-gray-950 dark:text-white;
}

.health-guard-skip-item span {
  @apply font-mono text-xs font-semibold text-amber-700 dark:text-amber-300;
}

.health-guard-skip-item p {
  @apply mt-1 break-words text-xs text-gray-500 dark:text-gray-400;
}

.health-guard-item-list {
  @apply mt-3 min-h-0 flex-1 space-y-2 overflow-auto;
}

.health-guard-item-card {
  @apply rounded-lg border border-gray-200 bg-white p-2.5 dark:border-dark-600 dark:bg-dark-800;
}

.health-guard-item-card.is-healthy {
  @apply border-emerald-200 bg-emerald-50/40 dark:border-emerald-900/70 dark:bg-emerald-950/20;
}

.health-guard-item-card.is-slow {
  @apply border-amber-200 bg-amber-50/50 dark:border-amber-900/70 dark:bg-amber-950/20;
}

.health-guard-item-card.is-failed {
  @apply border-red-200 bg-red-50/50 dark:border-red-900/70 dark:bg-red-950/20;
}

.health-guard-item-main {
  @apply flex min-w-0 flex-wrap items-baseline justify-between gap-2;
}

.health-guard-item-main strong {
  @apply min-w-0 truncate text-sm font-semibold text-gray-950 dark:text-white;
}

.health-guard-item-main span {
  @apply min-w-0 truncate text-xs text-gray-500 dark:text-gray-400;
}

.health-guard-item-metrics {
  @apply mt-2 flex flex-wrap items-center gap-2 text-xs text-gray-500 dark:text-gray-300;
}

.health-guard-item-card p {
  @apply mt-2 break-words text-xs text-gray-600 dark:text-gray-300;
}

.health-guard-empty {
  @apply rounded-lg border border-dashed border-gray-200 px-4 py-8 text-center text-sm text-gray-400 dark:border-dark-600;
}

@media (min-width: 1024px) {
  .health-guard-dialog {
    max-height: calc(90vh - 9rem);
    overflow: hidden;
  }

  .health-guard-content-grid {
    flex: 1 1 auto;
    overflow: hidden;
  }

  .health-guard-config-column,
  .health-guard-record-panel {
    overflow-y: auto;
    overscroll-behavior: contain;
  }

  .health-guard-adjustment-list {
    overflow-y: auto;
    padding-right: 0.125rem;
  }

  .health-guard-item-list {
    min-height: 8rem;
  }
}

@media (max-width: 640px) {
  :global(.modal-content:has(.health-guard-detail-dialog-large)) {
    height: calc(100vh - 1rem);
    max-height: calc(100vh - 1rem);
    height: calc(100dvh - 1rem);
    max-height: calc(100dvh - 1rem);
  }
}

.record-status {
  @apply inline-flex items-center rounded-md px-2 py-1 text-xs font-semibold ring-1;
}

.record-status-success {
  @apply bg-emerald-50 text-emerald-700 ring-emerald-200 dark:bg-emerald-950/40 dark:text-emerald-300 dark:ring-emerald-800/60;
}

.record-status-warning {
  @apply bg-amber-50 text-amber-700 ring-amber-200 dark:bg-amber-950/40 dark:text-amber-300 dark:ring-amber-800/60;
}

.record-status-error {
  @apply bg-red-50 text-red-700 ring-red-200 dark:bg-red-950/40 dark:text-red-300 dark:ring-red-800/60;
}

.record-status-muted {
  @apply bg-gray-100 text-gray-600 ring-gray-200 dark:bg-dark-700 dark:text-gray-300 dark:ring-dark-600;
}

.balance-recharge-form {
  @apply mt-3 grid gap-3 md:grid-cols-[12rem_1fr_auto];
}

.balance-record-section {
  @apply flex min-h-0 min-w-0 flex-1 flex-col gap-3;
}

.balance-record-tabs {
  @apply grid grid-cols-2 gap-2 rounded-lg bg-gray-100 p-1 dark:bg-dark-900/60;
}

.balance-record-tab {
  @apply inline-flex h-9 min-w-0 items-center justify-center gap-2 rounded-md px-3 text-sm font-semibold text-gray-500 transition-colors hover:text-gray-900 dark:text-gray-400 dark:hover:text-gray-100;
}

.balance-record-tab.is-active {
  @apply bg-white text-primary-600 shadow-sm dark:bg-dark-700 dark:text-primary-300;
}

.balance-record-tab strong {
  @apply inline-flex h-5 min-w-5 items-center justify-center rounded-full bg-gray-200 px-1.5 font-mono text-[11px] text-gray-600 dark:bg-dark-600 dark:text-gray-300;
}

.balance-record-tab.is-active strong {
  @apply bg-primary-50 text-primary-600 dark:bg-primary-950/60 dark:text-primary-300;
}

.balance-record-pane {
  @apply flex min-h-0 flex-1 flex-col gap-3;
}

.balance-record-header {
  @apply flex items-center justify-between gap-3;
}

.balance-section-title {
  @apply text-sm font-semibold text-gray-900 dark:text-white;
}

.balance-record-count {
  @apply inline-flex h-6 min-w-6 items-center justify-center rounded-full bg-gray-100 px-2 font-mono text-xs font-semibold text-gray-500 ring-1 ring-gray-200 dark:bg-dark-700 dark:text-gray-300 dark:ring-dark-600;
}

.balance-record-list {
  @apply min-h-[20rem] flex-1 bg-white dark:bg-dark-800;
  max-height: min(52vh, 34rem);
}

.balance-last-snapshot {
  @apply min-w-0;
}

.balance-last-snapshot-compact {
  display: none;
}

.balance-pill {
  @apply inline-flex items-center rounded-md bg-violet-50 px-2 py-1 font-mono text-xs font-semibold text-violet-700 ring-1 ring-violet-200 dark:bg-violet-950/40 dark:text-violet-300 dark:ring-violet-800/60;
}

.balance-pill-warning {
  @apply bg-amber-50 text-amber-700 ring-amber-200 dark:bg-amber-950/40 dark:text-amber-300 dark:ring-amber-800/60;
}

@media (max-width: 767px) {
  .upstream-providers-page :deep(.table-page-layout.mobile-mode) {
    height: auto;
    min-height: calc(100vh - 64px - 2rem);
  }

  .upstream-toolbar {
    @apply items-stretch gap-3 px-3;
  }

  .upstream-toolbar-left,
  .upstream-toolbar-filters,
  .upstream-toolbar-right,
  .upstream-quick-tags {
    @apply w-full justify-start;
  }

  .upstream-toolbar-left {
    @apply gap-2;
  }

  .upstream-toolbar-actions {
    width: 100%;
    gap: 8px;
  }

  .upstream-toolbar-title {
    @apply w-full whitespace-normal;
  }

  .upstream-toolbar-action {
    min-height: 32px;
    width: auto;
    min-width: 0;
    flex: 0 0 auto;
    justify-content: center;
    padding: 0 10px;
    font-size: 12px;
  }

  .upstream-sample-action,
  .upstream-sampler-settings-action,
  .upstream-health-run-action,
  .upstream-health-settings-action {
    width: auto;
  }

  .upstream-toolbar-filters {
    gap: 8px;
  }

  .upstream-search-row {
    display: grid;
    width: 100%;
    grid-template-columns: minmax(0, 1fr) auto;
    gap: 8px;
  }

  .upstream-filter-toggle {
    display: inline-flex;
    min-height: 34px;
    align-items: center;
    justify-content: center;
    gap: 5px;
    border: 1px solid #dbe3ee;
    border-radius: 8px;
    background: #f8fafc;
    padding: 0 9px;
    color: #334155;
    font-size: 12px;
    font-weight: 800;
    white-space: nowrap;
  }

  .upstream-filter-toggle strong {
    display: inline-flex;
    min-width: 18px;
    height: 18px;
    align-items: center;
    justify-content: center;
    border-radius: 999px;
    background: #dc2626;
    color: #fff;
    font-size: 11px;
    line-height: 1;
  }

  .upstream-filter-controls {
    display: none;
    width: 100%;
    grid-template-columns: repeat(2, minmax(0, 1fr));
    gap: 8px;
  }

  .upstream-filter-controls.is-open {
    display: grid;
  }

  .upstream-filter-select {
    @apply w-full;
  }

  .upstream-quick-tags {
    flex-wrap: nowrap;
    overflow-x: auto;
    padding-bottom: 2px;
    scrollbar-width: none;
  }

  .upstream-quick-tags::-webkit-scrollbar {
    display: none;
  }

  .upstream-quick-tag {
    flex: 0 0 auto;
    height: 30px;
    padding: 0 10px;
  }

  .upstream-toolbar-right {
    @apply gap-3;
  }

  .upstream-total {
    @apply min-w-0 flex-1 whitespace-normal rounded-md bg-gray-50 px-3 py-2 dark:bg-dark-900/40;
  }

  .column-settings-panel {
    @apply right-auto left-0 w-[min(13rem,calc(100vw-2rem))];
  }

  .provider-name-card,
  .action-button-group,
  .homepage-control-cell,
  .numeric-cell {
    @apply mx-0 min-w-0 max-w-full justify-end text-right;
  }

  .provider-title-line {
    @apply justify-end;
  }

  .provider-name,
  .provider-inline-url span:first-child,
  .copyable-text code,
  .prefix-value {
    @apply whitespace-normal break-all;
  }

  .action-button-group {
    @apply justify-end;
  }

  .action-button {
    @apply h-8;
  }

  .provider-mobile-detail-toggle {
    display: inline-flex;
  }

  .upstream-providers-page :deep(.provider-mobile-row-card) {
    position: relative;
    overflow: hidden;
    border-color: #e2e8f0;
    border-radius: 8px;
    background: #fff;
    padding: 12px;
    box-shadow: none;
  }

  .upstream-providers-page :deep(.provider-mobile-row-card::before) {
    position: absolute;
    inset: 0 auto 0 0;
    width: 3px;
    background: #00B42A;
    content: "";
  }

  .upstream-providers-page :deep(.provider-mobile-row-card.provider-disabled-row::before) {
    background: #94a3b8;
  }

  .upstream-providers-page :deep(.provider-mobile-row-card.provider-balance-anomaly-row) {
    border-color: #fecaca;
    background: #fffafa;
  }

  .upstream-providers-page :deep(.provider-mobile-row-card.provider-balance-anomaly-row::before) {
    background: #F53F3F;
  }

  .upstream-providers-page :deep(.provider-mobile-row-card > .space-y-3 > .flex) {
    gap: 10px;
  }

  .upstream-providers-page :deep(.provider-mobile-row-card > .space-y-3 > .flex > span) {
    color: #64748b;
    font-size: 11px;
    font-weight: 800;
    letter-spacing: 0;
    text-transform: none;
  }

  .upstream-providers-page :deep(.provider-mobile-row-card > .space-y-3 > .flex > div) {
    color: #0f172a;
    font-size: 13px;
  }

  .upstream-providers-page :deep(.provider-mobile-row-card:not(.provider-mobile-row-expanded) > .space-y-3 > .flex:nth-child(1)),
  .upstream-providers-page :deep(.provider-mobile-row-card:not(.provider-mobile-row-expanded) > .space-y-3 > .flex:nth-child(4)),
  .upstream-providers-page :deep(.provider-mobile-row-card:not(.provider-mobile-row-expanded) > .space-y-3 > .flex:nth-child(5)),
  .upstream-providers-page :deep(.provider-mobile-row-card:not(.provider-mobile-row-expanded) > .space-y-3 > .flex:nth-child(6)),
  .upstream-providers-page :deep(.provider-mobile-row-card:not(.provider-mobile-row-expanded) > .space-y-3 > .flex:nth-child(7)),
  .upstream-providers-page :deep(.provider-mobile-row-card:not(.provider-mobile-row-expanded) > .space-y-3 > .flex:nth-child(8)),
  .upstream-providers-page :deep(.provider-mobile-row-card:not(.provider-mobile-row-expanded) > .space-y-3 > .flex:nth-child(n + 11)) {
    display: none;
  }

  .provider-detail-panel {
    @apply grid-cols-1 gap-4 px-4 py-4;
  }

  .detail-endpoint {
    @apply grid-cols-1 gap-1;
  }

  .detail-endpoint code,
  .detail-copy code {
    @apply whitespace-normal break-all;
  }

  .balance-dialog-overlay {
    padding: 0;
    align-items: flex-end;
  }

  .balance-dialog {
    width: 100%;
    height: min(92dvh, calc(100vh - 0.75rem));
    max-height: min(92dvh, calc(100vh - 0.75rem));
    border-radius: 14px 14px 0 0;
  }

  .balance-dialog-header {
    position: relative;
    align-items: center;
    gap: 10px;
    padding: 18px 12px 10px;
  }

  .balance-dialog-handle {
    position: absolute;
    top: 7px;
    left: 50%;
    display: block;
    width: 38px;
    height: 4px;
    border-radius: 999px;
    background: #cbd5e1;
    transform: translateX(-50%);
  }

  .balance-dialog-header p {
    display: none;
  }

  .balance-dialog-close {
    width: 32px;
    height: 32px;
    padding: 0;
    border-radius: 8px;
  }

  .balance-dialog-close span {
    display: none;
  }

  .balance-dialog-body {
    max-height: none;
    padding: 10px;
    gap: 10px;
  }

  .balance-summary-grid {
    grid-template-columns: repeat(4, minmax(0, 1fr));
    gap: 6px;
  }

  .balance-metric {
    min-width: 0;
    padding: 7px 5px;
    text-align: center;
  }

  .balance-metric span {
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
    font-size: 11px;
  }

  .balance-metric strong {
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
    font-size: 12px;
    line-height: 1.25;
  }

  .balance-last-snapshot-full {
    display: none;
  }

  .balance-last-snapshot-compact {
    display: inline;
  }

  .balance-recharge-panel {
    padding: 10px;
  }

  .balance-recharge-form {
    grid-template-columns: repeat(2, minmax(0, 1fr));
    gap: 8px;
  }

  .balance-recharge-form .input {
    min-width: 0;
    height: 34px;
    font-size: 12px;
  }

  .balance-recharge-form .btn {
    grid-column: 1 / -1;
    width: fit-content;
    min-height: 32px;
    padding: 0 10px;
    font-size: 12px;
  }

  .balance-record-header {
    margin-top: 2px;
    padding: 0 2px;
  }

  .balance-record-tabs {
    gap: 4px;
    border-radius: 8px;
    padding: 4px;
  }

  .balance-record-tab {
    height: 32px;
    gap: 5px;
    padding: 0 6px;
    font-size: 12px;
  }

  .balance-record-tab strong {
    height: 18px;
    min-width: 18px;
    padding: 0 5px;
    font-size: 10px;
  }

  .balance-record-count {
    height: 22px;
    min-width: 22px;
    font-size: 11px;
  }

  .balance-record-list {
    min-height: 18rem;
    max-height: none;
  }

  .balance-record-list.mb-5 {
    margin-bottom: 10px;
  }

  .provider-mobile-record-cards {
    display: block;
    padding: 8px;
  }

  .provider-mobile-record-cards + table,
  .provider-mobile-record-cards + .records-table {
    display: none;
  }

  .balance-record-card {
    position: relative;
    padding: 9px 9px 9px 12px;
  }

  .balance-record-card::before {
    position: absolute;
    inset: 9px auto 9px 6px;
    width: 3px;
    border-radius: 999px;
    background: #94a3b8;
    content: "";
  }

  .balance-record-card.is-success::before {
    background: #10b981;
  }

  .balance-record-card.is-error::before {
    background: #ef4444;
  }

  .provider-mobile-record-card > div {
    grid-template-columns: minmax(70px, auto) minmax(0, 1fr);
    gap: 8px;
  }

  .provider-mobile-record-card > div + div {
    margin-top: 6px;
  }

  .provider-mobile-record-card span {
    font-size: 11px;
  }

  .provider-mobile-record-card strong {
    font-size: 12px;
    line-height: 1.3;
  }

  .balance-sampler-controls {
    @apply grid-cols-1;
  }

  .balance-config-panel {
    @apply items-stretch;
  }

  .balance-config-panel .input,
  .balance-config-panel .btn,
  .balance-sampler-interval .input {
    @apply w-full;
  }

  .balance-sampler-toggle,
  .balance-sampler-interval {
    @apply flex-col items-start;
  }

  .balance-sampler-provider-row {
    @apply grid-cols-1 gap-2;
  }

  .balance-sampler-provider-row .input {
    @apply w-full text-left;
  }

  .health-guard-status-panel,
  .health-guard-config-grid,
  .health-guard-platform-row {
    grid-template-columns: 1fr;
  }

  .health-guard-dialog {
    gap: 0.75rem;
  }

  .health-guard-content-grid {
    grid-template-columns: 1fr;
    overflow: visible;
  }

  .health-guard-config-column,
  .health-guard-record-panel {
    overflow: visible;
  }

  .health-guard-toggle,
  .health-guard-run-state {
    @apply items-start;
  }

  .health-guard-run-state {
    @apply flex-col;
  }

  .health-guard-run-button {
    width: 100%;
    min-height: 40px;
    padding: 0 12px;
  }

  .health-guard-field .input,
  .health-guard-platform-row .input,
  .health-guard-platform-row .input:last-child {
    @apply w-full text-left;
  }

  .health-guard-section-toggle small,
  .health-guard-ignored-summary strong {
    overflow: visible;
    text-overflow: clip;
    white-space: normal;
  }

  .health-guard-ignored-summary {
    @apply flex-col items-stretch;
  }

  .health-guard-ignored-add {
    grid-template-columns: 1fr;
  }

  .health-guard-ignored-add .btn {
    width: 100%;
  }

  .health-guard-account-models-add {
    grid-template-columns: 1fr;
  }

  .health-guard-account-models-add .btn {
    width: 100%;
  }

  .health-guard-account-model-row {
    grid-template-columns: minmax(0, 1fr) auto;
  }

  .health-guard-account-model-row-input {
    grid-column: 1 / -1;
  }

  .health-guard-platform-row {
    @apply items-start;
  }

  .health-guard-platform-name small {
    overflow: visible;
    text-overflow: clip;
    white-space: normal;
  }

  .health-guard-summary-grid {
    grid-template-columns: repeat(4, minmax(0, 1fr));
  }

  .health-guard-detail-action {
    min-height: 56px;
  }

  .health-guard-filter-row {
    display: grid;
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }

  .health-guard-filter-button {
    justify-content: space-between;
    width: 100%;
  }

  .health-guard-adjustment-header {
    @apply items-start;
  }

  .health-guard-adjustment-main,
  .health-guard-item-main {
    @apply block;
  }

  .health-guard-adjustment-main strong,
  .health-guard-item-main strong {
    overflow: visible;
    text-overflow: clip;
    white-space: normal;
    overflow-wrap: anywhere;
  }

  .health-guard-adjustment-main span,
  .health-guard-item-main span {
    @apply mt-1 block;
  }

  .health-guard-adjustment-metrics,
  .health-guard-item-metrics {
    @apply items-start;
  }

  .health-guard-adjustment-metrics span:not(.record-status),
  .health-guard-item-metrics span:not(.record-status) {
    flex-basis: 100%;
  }

  .health-guard-skip-list {
    grid-template-columns: 1fr;
  }

}

@media (max-width: 380px) {
  .upstream-filter-controls {
    grid-template-columns: 1fr;
  }

  .upstream-toolbar-action {
    flex: 1 1 calc(50% - 4px);
  }

  .health-guard-summary-grid {
    grid-template-columns: repeat(4, minmax(0, 1fr));
    gap: 0.25rem;
  }

  .health-guard-summary-card {
    padding: 0.375rem 0.25rem;
  }

  .health-guard-summary-card span {
    font-size: 0.6875rem;
  }

  .health-guard-summary-card strong {
    font-size: 0.875rem;
  }

  .health-guard-filter-row {
    grid-template-columns: 1fr;
  }

  .balance-recharge-form {
    grid-template-columns: 1fr;
  }
}
</style>

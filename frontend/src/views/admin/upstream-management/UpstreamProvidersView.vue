<template>
  <AppLayout>
    <div class="upstream-providers-page">
      <TablePageLayout>
        <template #filters>
        <div class="upstream-toolbar">
          <div class="upstream-toolbar-left">
            <div class="upstream-toolbar-title">{{ t('admin.upstreamProviders.title') }}</div>
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
          </div>

          <div class="upstream-toolbar-filters">
            <div class="relative w-full sm:w-64">
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

          <template #cell-interface="{ row }">
            <div class="interface-switcher">
              <div class="interface-tabs">
                <button
                  v-for="endpoint in endpointOptions(row)"
                  :key="endpoint.key"
                  type="button"
                  :class="['interface-tab', activeEndpointTab(row.slug) === endpoint.key && 'interface-tab-active']"
                  @click="setEndpointTab(row.slug, endpoint.key)"
                >
                  {{ endpoint.label }}
                </button>
              </div>
              <button
                type="button"
                class="copyable-text interface-path"
                :title="copyTitle(endpointValue(row, activeEndpointTab(row.slug)))"
                @click="copyValue(endpointValue(row, activeEndpointTab(row.slug)))"
              >
                <code>{{ endpointValue(row, activeEndpointTab(row.slug)) || '-' }}</code>
                <span class="copy-hint">{{ copyHint(endpointValue(row, activeEndpointTab(row.slug))) }}</span>
              </button>
            </div>
          </template>

          <template #cell-temp_disable_minutes="{ row }">
            <div class="center-cell">{{ formatTempDisable(row.temp_disable_minutes) }}</div>
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
                <span :class="['provider-enabled-tag', row.enabled ? 'is-enabled' : 'is-disabled']">
                  {{ row.enabled ? t('common.enabled') : t('common.disabled') }}
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
            <span :class="['badge', row.enabled ? 'badge-success' : 'badge-gray']">
              {{ row.enabled ? t('common.enabled') : t('common.disabled') }}
            </span>
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

    <BaseDialog
      :show="showBalanceSamplerDialog"
      :title="t('admin.upstreamProviders.balanceSamplerSettings')"
      width="wide"
      @close="closeBalanceSamplerDialog"
    >
      <div class="balance-sampler-dialog space-y-5">
        <div class="balance-sampler-controls">
          <label class="balance-sampler-toggle">
            <input
              v-model="balanceSamplerForm.enabled"
              type="checkbox"
              class="rounded border-gray-300 text-primary-600 focus:ring-primary-500"
            />
            <span>{{ t('admin.upstreamProviders.balanceSamplerAutoRun') }}</span>
          </label>

          <label class="balance-sampler-interval">
            <span>{{ t('admin.upstreamProviders.balanceSamplerIntervalSeconds') }}</span>
            <input
              v-model.number="balanceSamplerForm.interval_seconds"
              data-test="balance-sampler-interval"
              type="number"
              min="60"
              step="60"
              class="input"
            />
          </label>
        </div>

        <div class="balance-sampler-provider-panel">
          <div class="balance-section-title">{{ t('admin.upstreamProviders.amountScale') }}</div>
          <div class="balance-sampler-provider-list">
            <label
              v-for="provider in providers"
              :key="provider.slug"
              class="balance-sampler-provider-row"
            >
              <span class="balance-sampler-provider-name">
                <strong>{{ provider.name }}</strong>
                <small>{{ provider.slug }}</small>
              </span>
              <input
                v-model.number="balanceSamplerForm.provider_amount_scales[provider.slug]"
                :data-test="`balance-sampler-scale-${provider.slug}`"
                type="number"
                min="0.000001"
                step="any"
                class="input"
                :placeholder="formatScale(defaultBalanceSamplerScaleForProvider(provider.slug))"
              />
            </label>
            <div v-if="!providers.length" class="balance-sampler-empty">
              {{ t('common.noData') }}
            </div>
          </div>
        </div>
      </div>

      <template #footer>
        <div class="flex justify-end gap-3">
          <button type="button" class="btn btn-secondary" :disabled="savingBalanceSamplerConfig" @click="closeBalanceSamplerDialog">
            {{ t('common.cancel') }}
          </button>
          <button type="button" class="btn btn-primary" :disabled="savingBalanceSamplerConfig" @click="saveBalanceSamplerConfig">
            {{ savingBalanceSamplerConfig ? t('common.saving') : t('common.save') }}
          </button>
        </div>
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
            <label class="flex h-10 items-center gap-2 rounded-lg border border-gray-200 px-3 dark:border-dark-600">
              <input v-model="form.enabled" type="checkbox" class="rounded border-gray-300 text-primary-600 focus:ring-primary-500" />
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
              {{ t('admin.upstreamProviders.balanceDialogDescription') }}
            </p>
          </div>
          <button type="button" class="btn btn-secondary btn-sm" @click="closeBalanceDetails">
            {{ t('common.close') }}
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
              <strong class="text-sm">{{ selectedBalanceSummary?.last_snapshot_at ? formatDateTime(selectedBalanceSummary.last_snapshot_at) : '-' }}</strong>
            </div>
          </div>

          <div class="balance-recharge-panel">
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

          <div>
            <div class="balance-section-title">{{ t('admin.upstreamProviders.balanceSamples') }}</div>
            <div class="mb-5 max-h-72 overflow-auto rounded-lg border border-gray-200 dark:border-dark-600">
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

            <div class="balance-section-title">{{ t('admin.upstreamProviders.balanceHistory') }}</div>
            <div class="max-h-72 overflow-auto rounded-lg border border-gray-200 dark:border-dark-600">
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
import { computed, onMounted, reactive, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { adminAPI } from '@/api/admin'
import type {
  UpstreamProviderBalance,
  UpstreamProviderConfig,
  UpstreamProviderKey,
  UpstreamProviderTestResult,
  UpstreamProviderTestStage,
} from '@/api/admin/upstreamProviders'
import type {
  UpstreamBalanceConsumptionOverview,
  UpstreamBalanceDailyRow,
  UpstreamBalanceProviderSummary,
  UpstreamBalanceSamplerConfig,
  UpstreamBalanceSnapshot,
} from '@/api/admin/upstreamAccountSync'
import { useAppStore } from '@/stores/app'
import { extractApiErrorMessage } from '@/utils/apiError'
import { formatDateTime } from '@/utils/format'
import { useClipboard } from '@/composables/useClipboard'
import type { Column } from '@/components/common/types'
import AppLayout from '@/components/layout/AppLayout.vue'
import TablePageLayout from '@/components/layout/TablePageLayout.vue'
import DataTable from '@/components/common/DataTable.vue'
import BaseDialog from '@/components/common/BaseDialog.vue'
import ConfirmDialog from '@/components/common/ConfirmDialog.vue'
import EmptyState from '@/components/common/EmptyState.vue'
import Select, { type SelectOption } from '@/components/common/Select.vue'
import Icon from '@/components/icons/Icon.vue'
import UpstreamBalanceCharts from '@/components/admin/upstream/UpstreamBalanceCharts.vue'

const { t } = useI18n()
const appStore = useAppStore()
const { copyToClipboard } = useClipboard()

const providers = ref<UpstreamProviderConfig[]>([])
const loading = ref(false)
const searchQuery = ref('')
const typeFilter = ref('')
const enabledFilter = ref('')
const defaultingSlug = ref<string | null>(null)
const showColumnSettings = ref(false)
const visibleOptionalColumns = ref<string[]>([])
const expandedProviderSlugs = ref(new Set<string>())
const endpointTabs = ref<Record<string, string>>({})
const copiedValue = ref('')
let copiedTimer: ReturnType<typeof setTimeout> | undefined

const showFormDialog = ref(false)
const formMode = ref<'create' | 'edit'>('create')
const editingProvider = ref<UpstreamProviderConfig | null>(null)
const submitting = ref(false)
const testingDraft = ref(false)

const showBalanceSamplerDialog = ref(false)
const savingBalanceSamplerConfig = ref(false)

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
const balanceSamplerForm = ref({
  enabled: false,
  interval_seconds: 3600,
  provider_amount_scales: {} as Record<string, number>,
})
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

const baseColumns = computed<Column[]>(() => [
  { key: 'homepage', label: t('admin.upstreamProviders.columns.homepage'), class: 'upstream-homepage-column' },
  { key: 'name', label: t('admin.upstreamProviders.columns.name'), class: 'upstream-name-column' },
  { key: 'sort_order', label: t('admin.upstreamProviders.columns.sortOrder'), class: 'upstream-sort-order-column' },
  { key: 'interface', label: t('admin.upstreamProviders.columns.interface'), class: 'upstream-interface-column' },
  { key: 'prefix', label: t('admin.upstreamProviders.columns.prefix'), class: 'upstream-prefix-column' },
  { key: 'rate_scale', label: t('admin.upstreamProviders.columns.rateScale'), class: 'upstream-numeric-column' },
  { key: 'temp_disable_minutes', label: t('admin.upstreamProviders.tempDisableMinutes'), class: 'upstream-temp-column' },
  { key: 'balance', label: t('admin.upstreamProviders.columns.balance'), class: 'upstream-numeric-column' },
  { key: 'today_consumption', label: t('admin.upstreamProviders.columns.todayCost'), class: 'upstream-numeric-column' },
])

const optionalColumns = computed<Record<string, Column>>(() => ({
  temp_disable_minutes: { key: 'temp_disable_minutes', label: t('admin.upstreamProviders.tempDisableMinutes'), class: 'upstream-numeric-column' },
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

const filteredProviders = computed(() => {
  const keyword = searchQuery.value.trim().toLowerCase()
  return providers.value.filter((provider) => {
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
})

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

function activeEndpointTab(providerSlug: string | undefined) {
  if (!providerSlug) return 'keys'
  return endpointTabs.value[providerSlug] || 'keys'
}

function setEndpointTab(providerSlug: string | undefined, tab: string) {
  if (!providerSlug) return
  endpointTabs.value = {
    ...endpointTabs.value,
    [providerSlug]: tab,
  }
}

function endpointValue(provider: UpstreamProviderConfig, tab: string) {
  return endpointOptions(provider).find(option => option.key === tab)?.value || provider.api_keys_url || ''
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

function formatTempDisable(value: number | undefined) {
  const n = Number(value)
  if (!Number.isFinite(n) || n <= 0) return '0分钟'
  return `${n}分钟`
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

onMounted(reload)
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

.upstream-toolbar-filters {
  @apply flex flex-1 flex-wrap items-center justify-end gap-3;
}

.upstream-compact-input {
  @apply h-9 rounded-md text-sm;
}

.upstream-filter-select {
  @apply w-40;
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
.provider-default-tag,
.provider-enabled-tag {
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

.provider-enabled-tag.is-enabled {
  @apply bg-emerald-50 text-emerald-600 dark:bg-emerald-950/40 dark:text-emerald-300;
}

.provider-enabled-tag.is-enabled::before {
  @apply h-2 w-2 rounded-full bg-emerald-500;
  content: '';
}

.provider-enabled-tag.is-disabled {
  @apply bg-gray-100 text-gray-500 dark:bg-dark-700 dark:text-gray-300;
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

.interface-switcher {
  @apply mx-auto min-w-[15rem] max-w-[16rem] space-y-2 text-left;
}

.interface-tabs {
  @apply inline-flex gap-1;
}

.interface-tab {
  @apply h-8 rounded border border-gray-200 bg-white px-2.5 text-xs text-gray-500 transition-colors hover:border-primary-500 hover:text-primary-600 dark:border-dark-600 dark:bg-dark-800 dark:text-gray-400 dark:hover:text-primary-300;
}

.interface-tab-active {
  @apply border-primary-600 bg-primary-600 text-white hover:text-white dark:border-primary-500 dark:bg-primary-600 dark:text-white;
}

.copyable-text {
  @apply relative min-w-0 cursor-pointer border-0 text-left transition-colors;
}

.copyable-text code {
  @apply min-w-0 truncate;
}

.interface-path {
  @apply flex w-full items-center rounded bg-gray-50 px-2.5 py-1.5 font-mono text-xs text-gray-950 hover:bg-gray-100 dark:bg-dark-700 dark:text-gray-100 dark:hover:bg-dark-600;
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

:deep(.upstream-homepage-column) {
  width: 5.75rem;
  min-width: 5.75rem;
}

:deep(.upstream-name-column) {
  min-width: 16.25rem;
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

:deep(th.upstream-temp-column),
:deep(td.upstream-temp-column) {
  min-width: 7.25rem;
}

:deep(.upstream-interface-column) {
  min-width: 14.75rem;
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
  @apply max-h-[90vh] w-full max-w-5xl overflow-hidden rounded-xl bg-white shadow-2xl dark:bg-dark-800;
}

.balance-dialog-header {
  @apply flex items-start justify-between gap-4 border-b border-gray-200 px-5 py-4 dark:border-dark-600;
}

.balance-dialog-body {
  @apply max-h-[calc(90vh-5rem)] space-y-4 overflow-auto p-5;
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

.balance-recharge-form {
  @apply mt-3 grid gap-3 md:grid-cols-[12rem_1fr_auto];
}

.balance-section-title {
  @apply text-sm font-semibold text-gray-900 dark:text-white;
}

.balance-pill {
  @apply inline-flex items-center rounded-md bg-violet-50 px-2 py-1 font-mono text-xs font-semibold text-violet-700 ring-1 ring-violet-200 dark:bg-violet-950/40 dark:text-violet-300 dark:ring-violet-800/60;
}

.balance-pill-warning {
  @apply bg-amber-50 text-amber-700 ring-amber-200 dark:bg-amber-950/40 dark:text-amber-300 dark:ring-amber-800/60;
}
</style>

<template>
  <AppLayout>
    <TablePageLayout>
      <template #filters>
        <div class="flex flex-wrap items-center gap-3">
          <div class="relative w-full sm:w-72">
            <Icon
              name="search"
              size="md"
              class="absolute left-3 top-1/2 -translate-y-1/2 text-gray-400 dark:text-gray-500"
            />
            <input
              v-model="searchQuery"
              type="text"
              class="input pl-10"
              :placeholder="t('admin.upstreamProviders.searchPlaceholder')"
            />
          </div>

          <select v-model="typeFilter" class="input w-full sm:w-40">
            <option value="">{{ t('admin.upstreamProviders.allTypes') }}</option>
            <option value="sub2api">Sub2API</option>
            <option value="newapi">NewAPI</option>
          </select>

          <select v-model="enabledFilter" class="input w-full sm:w-40">
            <option value="">{{ t('admin.upstreamProviders.allStatus') }}</option>
            <option value="enabled">{{ t('common.enabled') }}</option>
            <option value="disabled">{{ t('common.disabled') }}</option>
          </select>

          <div class="flex flex-1 flex-wrap items-center justify-end gap-2">
            <button
              type="button"
              class="btn btn-secondary"
              :disabled="loading"
              :title="t('common.refresh')"
              @click="reload"
            >
              <Icon name="refresh" size="md" :class="loading ? 'animate-spin' : ''" />
            </button>
            <button type="button" class="btn btn-primary" @click="openCreateDialog">
              <Icon name="plus" size="md" class="mr-2" />
              {{ t('admin.upstreamProviders.createProvider') }}
            </button>
          </div>
        </div>
      </template>

      <template #table>
        <DataTable :columns="columns" :data="filteredProviders" :loading="loading">
          <template #cell-homepage="{ row }">
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
          </template>

          <template #cell-name="{ row }">
            <div :class="['provider-name-card min-w-[12rem]', providerToneClass(row.slug, 'card')]">
              <div class="flex items-center gap-2">
                <span class="min-w-0 flex-1 truncate font-semibold">{{ row.name }}</span>
                <span v-if="row.is_default" class="badge badge-success">
                  {{ t('admin.upstreamProviders.defaultProvider') }}
                </span>
              </div>
              <div class="mt-1 flex flex-wrap gap-1">
                <span class="badge" :class="providerTypeClass(row.type)">{{ providerTypeLabel(row.type) }}</span>
                <code :class="['provider-slug-tag', providerToneClass(row.slug, 'tag')]">{{ row.slug }}</code>
              </div>
            </div>
          </template>

          <template #cell-enabled="{ row }">
            <span :class="['badge', row.enabled ? 'badge-success' : 'badge-gray']">
              {{ row.enabled ? t('common.enabled') : t('common.disabled') }}
            </span>
          </template>

          <template #cell-base_url="{ value }">
            <code class="provider-url-tag" :title="value">{{ value || '-' }}</code>
          </template>

          <template #cell-auth="{ row }">
            <div class="tag-list max-w-[14rem]">
              <span v-if="row.username || row.email" class="info-tag tag-auth">
                {{ row.username || row.email }}
              </span>
              <span v-else class="info-tag tag-muted">-</span>
              <span v-if="row.password_configured" class="info-tag tag-success">
                {{ t('admin.upstreamProviders.passwordConfigured') }}
              </span>
            </div>
          </template>

          <template #cell-endpoints="{ row }">
            <div class="tag-list max-w-md">
              <span class="endpoint-tag" :title="row.api_keys_url">
                <span>{{ t('admin.upstreamProviders.keysEndpointShort') }}</span>
                <code>{{ row.api_keys_url || '-' }}</code>
              </span>
              <span v-if="row.login_url" class="endpoint-tag" :title="row.login_url">
                <span>{{ t('admin.upstreamProviders.loginEndpointShort') }}</span>
                <code>{{ row.login_url }}</code>
              </span>
              <span v-if="row.type === 'newapi' && row.groups_url" class="endpoint-tag" :title="row.groups_url">
                <span>{{ t('admin.upstreamProviders.groupsEndpointShort') }}</span>
                <code>{{ row.groups_url }}</code>
              </span>
              <span v-if="availableGroupsURL(row)" class="endpoint-tag" :title="availableGroupsURL(row)">
                <span>{{ t('admin.upstreamProviders.availableGroupsEndpointShort') }}</span>
                <code>{{ availableGroupsURL(row) }}</code>
              </span>
            </div>
          </template>

          <template #cell-policy="{ row }">
            <div class="tag-list max-w-[18rem]">
              <span class="info-tag tag-muted">{{ t('admin.upstreamProviders.prefix') }}: {{ row.account_name_prefix || '-' }}</span>
              <span class="info-tag tag-rate">{{ t('admin.upstreamProviders.accountRateMultiplierScaleShort') }}: {{ formatRateScale(row.account_rate_multiplier_scale) }}</span>
              <span class="info-tag tag-muted">{{ t('admin.upstreamProviders.tempDisableMinutes') }}: {{ row.temp_disable_minutes || 0 }}</span>
            </div>
          </template>

          <template #cell-balance_consumption="{ row }">
            <div class="balance-cost-cell">
              <div class="flex flex-wrap items-center gap-2">
                <span
                  v-if="providerBalances[row.slug]"
                  class="balance-pill"
                  :title="t('admin.upstreamProviders.balance')"
                >
                  {{ formatBalance(providerBalances[row.slug].balance) }}
                </span>
                <span class="font-mono text-sm font-semibold text-gray-950 dark:text-white">
                  {{ formatMoney(balanceSummaryFor(row.slug)?.today_consumption) }}
                </span>
              </div>
              <div class="mt-1 flex flex-wrap gap-1">
                <button
                  type="button"
                  class="action-button action-button-inline hover:bg-emerald-50 hover:text-emerald-600 dark:hover:bg-emerald-900/20 dark:hover:text-emerald-300"
                  :disabled="balanceLoadingSlugs.has(row.slug)"
                  :title="t('admin.upstreamProviders.fetchBalance')"
                  @click="fetchProviderBalance(row)"
                >
                  <Icon name="dollar" size="sm" :class="balanceLoadingSlugs.has(row.slug) ? 'animate-pulse' : ''" />
                  <span>{{ t('admin.upstreamProviders.balanceShort') }}</span>
                </button>
                <button
                  type="button"
                  class="action-button action-button-inline hover:bg-violet-50 hover:text-violet-600 dark:hover:bg-violet-900/20 dark:hover:text-violet-300"
                  :title="t('common.more')"
                  @click="openBalanceDetails(row.slug)"
                >
                  <Icon name="more" size="sm" />
                  <span>{{ t('common.more') }}</span>
                </button>
                <span :class="['info-tag', balanceSummaryFor(row.slug)?.complete ? 'tag-success' : 'tag-muted']">
                  {{ balanceSummaryFor(row.slug)?.complete ? t('admin.upstreamProviders.balanceComplete') : t('admin.upstreamProviders.balanceIncomplete') }}
                </span>
                <span v-if="balanceSummaryFor(row.slug)?.anomaly" class="info-tag tag-warning">
                  {{ t('admin.upstreamProviders.balanceAnomaly') }}
                </span>
              </div>
            </div>
          </template>

          <template #cell-actions="{ row }">
            <div class="flex items-center gap-1">
              <button
                v-if="!row.is_default"
                type="button"
                class="action-button hover:bg-amber-50 hover:text-amber-600 dark:hover:bg-amber-900/20 dark:hover:text-amber-300"
                :disabled="defaultingSlug === row.slug"
                :title="t('admin.upstreamProviders.setDefault')"
                @click="setDefaultProvider(row)"
              >
                <Icon name="badge" size="sm" :class="defaultingSlug === row.slug ? 'animate-pulse' : ''" />
                <span>{{ t('admin.upstreamProviders.setDefaultShort') }}</span>
              </button>
              <button
                type="button"
                class="action-button hover:bg-emerald-50 hover:text-emerald-600 dark:hover:bg-emerald-900/20 dark:hover:text-emerald-300"
                :disabled="testingSlugs.has(row.slug)"
                :title="t('admin.upstreamProviders.testProvider')"
                @click="testSavedProvider(row)"
              >
                <Icon name="play" size="sm" :class="testingSlugs.has(row.slug) ? 'animate-pulse' : ''" />
                <span>{{ t('admin.upstreamProviders.testShort') }}</span>
              </button>
              <button
                type="button"
                class="action-button hover:bg-blue-50 hover:text-blue-600 dark:hover:bg-blue-900/20 dark:hover:text-blue-300"
                :disabled="keysLoadingSlug === row.slug"
                :title="t('admin.upstreamProviders.fetchKeys')"
                @click="openKeysDialog(row)"
              >
                <Icon name="key" size="sm" :class="keysLoadingSlug === row.slug ? 'animate-pulse' : ''" />
                <span>{{ t('admin.upstreamProviders.keysShort') }}</span>
              </button>
              <button
                type="button"
                class="action-button hover:bg-gray-100 hover:text-primary-600 dark:hover:bg-dark-700 dark:hover:text-primary-400"
                :title="t('common.edit')"
                @click="openEditDialog(row)"
              >
                <Icon name="edit" size="sm" />
                <span>{{ t('common.edit') }}</span>
              </button>
              <button
                type="button"
                class="action-button hover:bg-red-50 hover:text-red-600 dark:hover:bg-red-900/20 dark:hover:text-red-300"
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
              <strong>{{ formatMoney(selectedBalanceSummary?.today_consumption) }}</strong>
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

          <div class="balance-config-panel">
            <label class="guard-toggle">
              <input v-model="balanceSamplerForm.enabled" type="checkbox" class="h-4 w-4 rounded border-gray-300 text-primary-600 focus:ring-primary-500" />
              <span>{{ t('admin.upstreamProviders.balanceSamplerAutoRun') }}</span>
            </label>
            <label class="guard-interval">
              <span>{{ t('admin.upstreamProviders.balanceSamplerIntervalSeconds') }}</span>
              <input v-model.number="balanceSamplerForm.interval_seconds" type="number" min="60" class="input h-9 w-28" />
            </label>
            <label class="guard-interval">
              <span>{{ t('admin.upstreamProviders.amountScale') }}</span>
              <input v-model.number="selectedProviderScaleInput" type="number" min="0.000001" step="0.000001" class="input h-9 w-28" />
            </label>
            <button type="button" class="btn btn-secondary" :disabled="savingBalanceSamplerConfig" @click="saveBalanceSamplerConfig">
              <Icon name="cog" size="sm" class="mr-2" :class="savingBalanceSamplerConfig ? 'animate-spin' : ''" />
              {{ t('common.save') }}
            </button>
            <button type="button" class="btn btn-primary" :disabled="runningBalanceSampleNow" @click="runBalanceSampleNow">
              <Icon name="play" size="sm" class="mr-2" :class="runningBalanceSampleNow ? 'animate-pulse' : ''" />
              {{ t('admin.upstreamProviders.balanceSampleNow') }}
            </button>
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
import type { Column } from '@/components/common/types'
import AppLayout from '@/components/layout/AppLayout.vue'
import TablePageLayout from '@/components/layout/TablePageLayout.vue'
import DataTable from '@/components/common/DataTable.vue'
import BaseDialog from '@/components/common/BaseDialog.vue'
import ConfirmDialog from '@/components/common/ConfirmDialog.vue'
import EmptyState from '@/components/common/EmptyState.vue'
import Icon from '@/components/icons/Icon.vue'

const { t } = useI18n()
const appStore = useAppStore()

const providers = ref<UpstreamProviderConfig[]>([])
const loading = ref(false)
const searchQuery = ref('')
const typeFilter = ref('')
const enabledFilter = ref('')
const defaultingSlug = ref<string | null>(null)

const showFormDialog = ref(false)
const formMode = ref<'create' | 'edit'>('create')
const editingProvider = ref<UpstreamProviderConfig | null>(null)
const submitting = ref(false)
const testingDraft = ref(false)

const showTestDialog = ref(false)
const testResult = ref<UpstreamProviderTestResult | null>(null)
const testingSlugs = ref(new Set<string>())
const balanceLoadingSlugs = ref(new Set<string>())
const providerBalances = ref<Record<string, UpstreamProviderBalance>>({})
const savingBalanceSamplerConfig = ref(false)
const runningBalanceSampleNow = ref(false)
const addingRecharge = ref(false)
const balanceOverview = ref<UpstreamBalanceConsumptionOverview | null>(null)
const balanceDetailsOpen = ref(false)
const selectedBalanceProviderSlug = ref('')
const selectedProviderScaleInput = ref(1)
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
  enabled: true,
  is_default: false,
  base_url: '',
  login_url: '',
  api_keys_url: '',
  groups_url: '',
  available_groups_url: '',
  email: '',
  username: '',
  password: '',
  account_name_prefix: '',
  temp_disable_minutes: 0,
  account_rate_multiplier_scale: 1,
})

const columns = computed<Column[]>(() => [
  { key: 'homepage', label: t('admin.upstreamProviders.columns.homepage') },
  { key: 'name', label: t('admin.upstreamProviders.columns.name') },
  { key: 'enabled', label: t('admin.upstreamProviders.columns.status') },
  { key: 'base_url', label: t('admin.upstreamProviders.columns.baseUrl') },
  { key: 'auth', label: t('admin.upstreamProviders.columns.auth') },
  { key: 'endpoints', label: t('admin.upstreamProviders.columns.endpoints') },
  { key: 'policy', label: t('admin.upstreamProviders.columns.policy') },
  { key: 'balance_consumption', label: t('admin.upstreamProviders.columns.balanceConsumption') },
  { key: 'actions', label: t('common.actions') },
])

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
    enabled: true,
    is_default: false,
    base_url: '',
    login_url: '',
    api_keys_url: '',
    groups_url: '',
    available_groups_url: '',
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
    email: provider.email || '',
    username: provider.username || '',
    password: '',
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
    enabled: form.enabled,
    is_default: Boolean(form.is_default),
    base_url: form.base_url.trim(),
    login_url: form.login_url?.trim() || '',
    api_keys_url: form.api_keys_url.trim(),
    groups_url: form.type === 'newapi' ? form.groups_url?.trim() || '' : '',
    available_groups_url: form.available_groups_url?.trim() || '',
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
  if (selectedBalanceProviderSlug.value) {
    selectedProviderScaleInput.value = selectedBalanceScale.value
  }
}

async function saveBalanceSamplerConfig() {
  if (!Number.isInteger(balanceSamplerForm.value.interval_seconds) || balanceSamplerForm.value.interval_seconds < 60) {
    appStore.showError(t('admin.upstreamProviders.invalidBalanceSamplerInterval'))
    return
  }
  if (!selectedBalanceProviderSlug.value) return
  const scale = Number(selectedProviderScaleInput.value)
  if (!Number.isFinite(scale) || scale <= 0) {
    appStore.showError(t('admin.upstreamProviders.invalidAmountScale'))
    return
  }
  savingBalanceSamplerConfig.value = true
  try {
    const providerAmountScales = {
      ...balanceSamplerForm.value.provider_amount_scales,
      [selectedBalanceProviderSlug.value]: scale,
    }
    const base = balanceOverview.value?.config || { enabled: false, interval_seconds: 3600 }
    const config = await adminAPI.upstreamAccountSync.updateBalanceSamplerConfig({
      ...base,
      enabled: balanceSamplerForm.value.enabled,
      interval_seconds: balanceSamplerForm.value.interval_seconds,
      provider_amount_scales: providerAmountScales,
    })
    applyBalanceSamplerConfig(config)
    appStore.showSuccess(t('admin.upstreamProviders.balanceSamplerSaved'))
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
  if (row.anomaly) return t('admin.upstreamProviders.balanceAnomaly')
  if (row.complete) return t('admin.upstreamProviders.balanceComplete')
  return t('admin.upstreamProviders.balanceIncomplete')
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
    maximumFractionDigits: 6,
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
  if (type === 'newapi') return 'badge-primary'
  return 'badge-gray'
}

function providerToneClass(providerSlug: string | undefined, target: 'card' | 'tag') {
  const tones = ['sky', 'emerald', 'violet', 'cyan', 'rose', 'amber', 'indigo', 'teal']
  const slug = providerSlug?.trim() || 'default'
  let hash = 0
  for (let i = 0; i < slug.length; i++) {
    hash = (hash * 31 + slug.charCodeAt(i)) >>> 0
  }
  const tone = tones[hash % tones.length]
  return `${target === 'card' ? 'provider-name-card' : 'provider-slug-tag'}-${tone}`
}

function availableGroupsURL(provider: UpstreamProviderConfig) {
  return provider.available_groups_url || (provider.type === 'newapi' ? '' : provider.groups_url || '')
}

function retainBalancesForProviders(nextProviders: UpstreamProviderConfig[]) {
  const slugs = new Set(nextProviders.map(provider => provider.slug))
  providerBalances.value = Object.fromEntries(
    Object.entries(providerBalances.value).filter(([slug]) => slugs.has(slug))
  )
}

function uniqueProviderURLs(field: 'base_url' | 'api_keys_url' | 'login_url' | 'groups_url' | 'available_groups_url') {
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
  return Number.isFinite(n) ? `${n.toFixed(2)}x` : '-'
}

function formatBalance(value: number | undefined) {
  const n = Number(value)
  return Number.isFinite(n) ? n.toFixed(6).replace(/\.?0+$/, '') : '-'
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
.action-button {
  @apply flex flex-col items-center gap-0.5 rounded-lg p-1.5 text-gray-500 transition-colors disabled:cursor-not-allowed disabled:opacity-50;
}

.action-button-inline {
  @apply flex-row items-center gap-1 px-2 py-1 text-xs;
}

.homepage-button {
  @apply inline-flex items-center gap-1.5 rounded-md bg-sky-50 px-2 py-1 text-xs font-semibold text-sky-700 ring-1 ring-sky-200 hover:bg-sky-100 dark:bg-sky-950/30 dark:text-sky-300 dark:ring-sky-800/60 dark:hover:bg-sky-900/40;
}

.provider-name-card {
  @apply rounded-lg border px-3 py-2 shadow-sm;
}

.provider-name-card-sky {
  @apply border-sky-200 bg-sky-50/70 text-sky-950 dark:border-sky-800/50 dark:bg-sky-950/20 dark:text-white;
}

.provider-name-card-emerald {
  @apply border-emerald-200 bg-emerald-50/70 text-emerald-950 dark:border-emerald-800/50 dark:bg-emerald-950/20 dark:text-white;
}

.provider-name-card-violet {
  @apply border-violet-200 bg-violet-50/70 text-violet-950 dark:border-violet-800/50 dark:bg-violet-950/20 dark:text-white;
}

.provider-name-card-cyan {
  @apply border-cyan-200 bg-cyan-50/70 text-cyan-950 dark:border-cyan-800/50 dark:bg-cyan-950/20 dark:text-white;
}

.provider-name-card-rose {
  @apply border-rose-200 bg-rose-50/70 text-rose-950 dark:border-rose-800/50 dark:bg-rose-950/20 dark:text-white;
}

.provider-name-card-amber {
  @apply border-amber-200 bg-amber-50/70 text-amber-950 dark:border-amber-800/50 dark:bg-amber-950/20 dark:text-white;
}

.provider-name-card-indigo {
  @apply border-indigo-200 bg-indigo-50/70 text-indigo-950 dark:border-indigo-800/50 dark:bg-indigo-950/20 dark:text-white;
}

.provider-name-card-teal {
  @apply border-teal-200 bg-teal-50/70 text-teal-950 dark:border-teal-800/50 dark:bg-teal-950/20 dark:text-white;
}

.provider-slug-tag {
  @apply inline-flex items-center rounded-md px-2 py-1 font-mono text-xs font-semibold ring-1;
}

.provider-slug-tag-sky {
  @apply bg-sky-100 text-sky-700 ring-sky-200 dark:bg-sky-950/50 dark:text-sky-300 dark:ring-sky-800/60;
}

.provider-slug-tag-emerald {
  @apply bg-emerald-100 text-emerald-700 ring-emerald-200 dark:bg-emerald-950/50 dark:text-emerald-300 dark:ring-emerald-800/60;
}

.provider-slug-tag-violet {
  @apply bg-violet-100 text-violet-700 ring-violet-200 dark:bg-violet-950/50 dark:text-violet-300 dark:ring-violet-800/60;
}

.provider-slug-tag-cyan {
  @apply bg-cyan-100 text-cyan-700 ring-cyan-200 dark:bg-cyan-950/50 dark:text-cyan-300 dark:ring-cyan-800/60;
}

.provider-slug-tag-rose {
  @apply bg-rose-100 text-rose-700 ring-rose-200 dark:bg-rose-950/50 dark:text-rose-300 dark:ring-rose-800/60;
}

.provider-slug-tag-amber {
  @apply bg-amber-100 text-amber-700 ring-amber-200 dark:bg-amber-950/50 dark:text-amber-300 dark:ring-amber-800/60;
}

.provider-slug-tag-indigo {
  @apply bg-indigo-100 text-indigo-700 ring-indigo-200 dark:bg-indigo-950/50 dark:text-indigo-300 dark:ring-indigo-800/60;
}

.provider-slug-tag-teal {
  @apply bg-teal-100 text-teal-700 ring-teal-200 dark:bg-teal-950/50 dark:text-teal-300 dark:ring-teal-800/60;
}

.provider-url-tag {
  @apply inline-flex max-w-full items-center rounded-md bg-gray-100 px-2 py-1 font-mono text-xs text-gray-700 ring-1 ring-gray-200 dark:bg-dark-700 dark:text-gray-200 dark:ring-dark-600;
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

.balance-recharge-form {
  @apply mt-3 grid gap-3 md:grid-cols-[12rem_1fr_auto];
}

.balance-section-title {
  @apply text-sm font-semibold text-gray-900 dark:text-white;
}

.balance-pill {
  @apply inline-flex items-center rounded-md bg-violet-50 px-2 py-1 font-mono text-xs font-semibold text-violet-700 ring-1 ring-violet-200 dark:bg-violet-950/40 dark:text-violet-300 dark:ring-violet-800/60;
}
</style>

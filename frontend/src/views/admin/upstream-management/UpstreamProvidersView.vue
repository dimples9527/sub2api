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
          <template #cell-name="{ row }">
            <div class="flex min-w-[12rem] flex-col gap-1">
              <div class="flex items-center gap-2">
                <span class="font-medium text-gray-900 dark:text-white">{{ row.name }}</span>
                <span class="badge" :class="providerTypeClass(row.type)">{{ providerTypeLabel(row.type) }}</span>
                <span v-if="row.is_default" class="badge badge-success">
                  {{ t('admin.upstreamProviders.defaultProvider') }}
                </span>
              </div>
              <code class="text-xs text-gray-500 dark:text-gray-400">{{ row.slug }}</code>
            </div>
          </template>

          <template #cell-enabled="{ row }">
            <span :class="['badge', row.enabled ? 'badge-success' : 'badge-gray']">
              {{ row.enabled ? t('common.enabled') : t('common.disabled') }}
            </span>
          </template>

          <template #cell-base_url="{ value }">
            <code class="block max-w-xs truncate text-xs" :title="value">{{ value || '-' }}</code>
          </template>

          <template #cell-auth="{ row }">
            <div class="flex flex-col gap-1 text-xs">
              <span v-if="row.username || row.email" class="text-gray-700 dark:text-gray-200">
                {{ row.username || row.email }}
              </span>
              <span v-else class="text-gray-400">-</span>
              <span
                v-if="row.password_configured"
                class="inline-flex w-fit items-center rounded bg-emerald-50 px-1.5 py-0.5 font-medium text-emerald-700 dark:bg-emerald-900/30 dark:text-emerald-300"
              >
                {{ t('admin.upstreamProviders.passwordConfigured') }}
              </span>
            </div>
          </template>

          <template #cell-endpoints="{ row }">
            <div class="flex max-w-md flex-col gap-1 text-xs">
              <span class="truncate" :title="row.api_keys_url">
                <span class="text-gray-400">{{ t('admin.upstreamProviders.keysEndpointShort') }}:</span>
                <code>{{ row.api_keys_url || '-' }}</code>
              </span>
              <span v-if="row.login_url" class="truncate" :title="row.login_url">
                <span class="text-gray-400">{{ t('admin.upstreamProviders.loginEndpointShort') }}:</span>
                <code>{{ row.login_url }}</code>
              </span>
              <span v-if="row.groups_url" class="truncate" :title="row.groups_url">
                <span class="text-gray-400">{{ t('admin.upstreamProviders.groupsEndpointShort') }}:</span>
                <code>{{ row.groups_url }}</code>
              </span>
            </div>
          </template>

          <template #cell-policy="{ row }">
            <div class="flex flex-col gap-1 text-xs text-gray-600 dark:text-gray-300">
              <span>{{ t('admin.upstreamProviders.prefix') }}: {{ row.account_name_prefix || '-' }}</span>
              <span>{{ t('admin.upstreamProviders.tempDisableMinutes') }}: {{ row.temp_disable_minutes || 0 }}</span>
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
            <input v-model.trim="form.base_url" required type="url" class="input" placeholder="https://example.com" />
          </div>
          <div>
            <label class="input-label">{{ t('admin.upstreamProviders.apiKeysUrl') }}</label>
            <input v-model.trim="form.api_keys_url" required type="text" class="input" placeholder="/api/token/" />
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
              placeholder="/api/user/login"
            />
          </div>
          <div v-if="form.type === 'newapi'" class="md:col-span-2">
            <label class="input-label">{{ t('admin.upstreamProviders.groupsUrl') }}</label>
            <input v-model.trim="form.groups_url" required type="text" class="input" placeholder="/api/group/" />
          </div>
        </div>

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
  UpstreamProviderConfig,
  UpstreamProviderKey,
  UpstreamProviderTestResult,
  UpstreamProviderTestStage,
} from '@/api/admin/upstreamProviders'
import { useAppStore } from '@/stores/app'
import { extractApiErrorMessage } from '@/utils/apiError'
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
  email: '',
  username: '',
  password: '',
  account_name_prefix: '',
  temp_disable_minutes: 0,
})

const columns = computed<Column[]>(() => [
  { key: 'name', label: t('admin.upstreamProviders.columns.name') },
  { key: 'enabled', label: t('admin.upstreamProviders.columns.status') },
  { key: 'base_url', label: t('admin.upstreamProviders.columns.baseUrl') },
  { key: 'auth', label: t('admin.upstreamProviders.columns.auth') },
  { key: 'endpoints', label: t('admin.upstreamProviders.columns.endpoints') },
  { key: 'policy', label: t('admin.upstreamProviders.columns.policy') },
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
    ]
      .filter(Boolean)
      .some((value) => String(value).toLowerCase().includes(keyword))
  })
})

const keysDialogTitle = computed(() => {
  const name = keysProvider.value?.name || ''
  return name ? t('admin.upstreamProviders.keysDialogTitleWithName', { name }) : t('admin.upstreamProviders.keysDialogTitle')
})

const deleteMessage = computed(() => {
  const name = deletingProvider.value?.name || ''
  return t('admin.upstreamProviders.deleteConfirm', { name })
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
    email: '',
    username: '',
    password: '',
    account_name_prefix: '',
    temp_disable_minutes: 0,
  })
}

function fillForm(provider: UpstreamProviderConfig) {
  Object.assign(form, {
    ...provider,
    login_url: provider.login_url || '',
    groups_url: provider.groups_url || '',
    email: provider.email || '',
    username: provider.username || '',
    password: '',
    is_default: Boolean(provider.is_default),
    account_name_prefix: provider.account_name_prefix || '',
    temp_disable_minutes: provider.temp_disable_minutes || 0,
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
    email: form.email?.trim() || '',
    username: form.username?.trim() || '',
    account_name_prefix: form.account_name_prefix?.trim() || '',
    temp_disable_minutes: Number(form.temp_disable_minutes) || 0,
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
    providers.value = await adminAPI.upstreamProviders.list()
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

function formatRate(value: number | undefined) {
  const n = Number(value)
  return Number.isFinite(n) ? `${n.toFixed(2)}x` : '-'
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
</style>

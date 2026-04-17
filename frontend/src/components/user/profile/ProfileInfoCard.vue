<template>
  <div class="card overflow-hidden">
    <div
      class="border-b border-gray-100 bg-gradient-to-r from-primary-500/10 to-primary-600/5 px-6 py-5 dark:border-dark-700 dark:from-primary-500/20 dark:to-primary-600/10"
    >
      <div class="flex items-center gap-4">
        <!-- Avatar -->
        <div
          class="flex h-16 w-16 items-center justify-center rounded-2xl bg-gradient-to-br from-primary-500 to-primary-600 text-2xl font-bold text-white shadow-lg shadow-primary-500/20"
        >
          {{ user?.email?.charAt(0).toUpperCase() || 'U' }}
        </div>
        <div class="min-w-0 flex-1">
          <h2 class="truncate text-lg font-semibold text-gray-900 dark:text-white">
            {{ user?.email }}
          </h2>
          <div class="mt-1 flex items-center gap-2">
            <span :class="['badge', user?.role === 'admin' ? 'badge-primary' : 'badge-gray']">
              {{ user?.role === 'admin' ? t('profile.administrator') : t('profile.user') }}
            </span>
            <span
              :class="['badge', user?.status === 'active' ? 'badge-success' : 'badge-danger']"
            >
              {{ user?.status }}
            </span>
          </div>
        </div>
      </div>
    </div>
    <div class="px-6 py-4">
      <div class="space-y-3">
        <div class="flex items-center gap-3 text-sm text-gray-600 dark:text-gray-400">
          <Icon name="mail" size="sm" class="text-gray-400 dark:text-gray-500" />
          <span class="truncate">{{ user?.email }}</span>
        </div>
        <div
          v-if="user?.username"
          class="flex items-center gap-3 text-sm text-gray-600 dark:text-gray-400"
        >
          <Icon name="user" size="sm" class="text-gray-400 dark:text-gray-500" />
          <span class="truncate">{{ user.username }}</span>
        </div>
        <div
          v-if="user?.invite_code"
          class="rounded-xl border border-primary-100 bg-primary-50/70 p-3 dark:border-primary-900/40 dark:bg-primary-900/10"
        >
          <div class="flex items-start justify-between gap-3">
            <div class="min-w-0">
              <p class="text-xs font-medium uppercase tracking-[0.12em] text-primary-600 dark:text-primary-300">
                {{ t('profile.inviteCode') }}
              </p>
              <p class="mt-1 font-mono text-base font-semibold text-gray-900 dark:text-white">
                {{ user.invite_code }}
              </p>
            </div>
            <div class="flex gap-2">
              <button type="button" class="btn btn-sm btn-secondary" @click="copyInviteCode">
                {{ t('profile.copyCode') }}
              </button>
              <button type="button" class="btn btn-sm btn-secondary" @click="copyInviteLink">
                {{ t('profile.copyInviteLink') }}
              </button>
            </div>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import Icon from '@/components/icons/Icon.vue'
import type { User } from '@/types'
import { useAppStore } from '@/stores'

const props = defineProps<{
  user: User | null
}>()

const { t } = useI18n()
const appStore = useAppStore()

async function copyText(value: string): Promise<void> {
  try {
    await navigator.clipboard.writeText(value)
    appStore.showSuccess(t('common.copiedToClipboard'))
  } catch {
    appStore.showError(t('common.copyFailed'))
  }
}

async function copyInviteCode(): Promise<void> {
  if (!props.user?.invite_code) return
  await copyText(props.user.invite_code)
}

async function copyInviteLink(): Promise<void> {
  if (!props.user?.invite_code) return
  const link = `${window.location.origin}/register?invite=${encodeURIComponent(props.user.invite_code)}`
  await copyText(link)
}
</script>

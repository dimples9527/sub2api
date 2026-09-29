<template>
  <AppLayout>
    <div class="space-y-6">
      <div v-if="loading" class="flex items-center justify-center py-16">
        <div class="h-8 w-8 animate-spin rounded-full border-b-2 border-primary-600"></div>
      </div>

      <template v-else>
        <div class="flex flex-col gap-4 lg:flex-row lg:items-center lg:justify-between">
          <div>
            <h1 class="text-2xl font-semibold text-gray-900 dark:text-white">
              {{ t('admin.rechargePromo.title') }}
            </h1>
            <p class="mt-1 text-sm text-gray-500 dark:text-gray-400">
              {{ t('admin.rechargePromo.description') }}
            </p>
          </div>
          <div class="flex flex-wrap items-center gap-2">
            <button
              type="button"
              class="btn btn-primary inline-flex items-center gap-2"
              :disabled="saving"
              @click="save"
            >
              <Icon name="check" size="sm" />
              {{ t('admin.rechargePromo.save') }}
            </button>
          </div>
        </div>

        <div class="card space-y-4 p-6">
          <label class="input-label flex items-center gap-2">
            <input v-model="enabled" type="checkbox" class="h-4 w-4" />
            {{ t('admin.rechargePromo.enabled') }}
          </label>
          <p class="text-xs text-gray-400">
            {{ t('admin.rechargePromo.enabledHint') }}
          </p>

          <div
            v-if="enabled"
            class="space-y-4 rounded-lg border border-gray-200 p-4 dark:border-dark-700"
          >
            <div class="flex flex-wrap gap-4">
              <div>
                <label class="input-label">{{ t('admin.rechargePromo.start') }}</label>
                <input v-model="startAt" type="datetime-local" class="input w-56" />
              </div>
              <div>
                <label class="input-label">{{ t('admin.rechargePromo.end') }}</label>
                <input v-model="endAt" type="datetime-local" class="input w-56" />
              </div>
            </div>
            <p class="-mt-2 text-xs text-gray-400">
              {{ t('admin.rechargePromo.windowHint') }}
            </p>
            <!-- PLACEHOLDER_TIERS -->
            <div>
              <label class="input-label">{{ t('admin.rechargePromo.tiers') }}</label>
              <div
                v-for="(tier, i) in tiers"
                :key="`promo-tier-${i}`"
                class="mb-2 flex items-center gap-2"
              >
                <input
                  v-model="tier.threshold"
                  type="number"
                  min="0"
                  step="0.01"
                  class="input w-32"
                  :placeholder="t('admin.rechargePromo.thresholdPlaceholder')"
                />
                <div class="flex items-center gap-1">
                  <input
                    v-model="tier.bonusPercent"
                    type="number"
                    min="0"
                    step="0.01"
                    class="input w-28"
                    :placeholder="t('admin.rechargePromo.bonusPlaceholder')"
                  />
                  <span class="text-sm text-gray-400">%</span>
                </div>
                <button
                  type="button"
                  class="btn btn-secondary btn-sm shrink-0 text-red-600 hover:text-red-700 dark:text-red-400"
                  @click="removeTier(i)"
                >
                  {{ t('admin.rechargePromo.removeTier') }}
                </button>
              </div>
              <button type="button" class="btn btn-secondary btn-sm" @click="addTier">
                {{ t('admin.rechargePromo.addTier') }}
              </button>
              <p class="mt-1 text-xs text-gray-400">
                {{ t('admin.rechargePromo.tiersHint') }}
              </p>
              <p class="mt-1 text-xs text-gray-400">
                {{ t('admin.rechargePromo.tiersExample') }}
              </p>
            </div>
          </div>
        </div>
      </template>
    </div>
  </AppLayout>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { useI18n } from 'vue-i18n'
import AppLayout from '@/components/layout/AppLayout.vue'
import Icon from '@/components/icons/Icon.vue'
import { useAppStore } from '@/stores/app'
import { extractApiErrorMessage } from '@/utils/apiError'
import { adminAPI } from '@/api/admin'
import type { RechargePromoConfig } from '@/api/admin/rechargePromo'

interface TierRow {
  threshold: string | number
  bonusPercent: string | number
}

const { t } = useI18n()
const appStore = useAppStore()

const loading = ref(true)
const saving = ref(false)
const enabled = ref(false)
const startAt = ref('')
const endAt = ref('')
const tiers = ref<TierRow[]>([])

// datetime-local <-> unix seconds, interpreted in the admin browser's local timezone;
// the server compares absolute instants so there is no timezone ambiguity.
function unixToDatetimeLocal(unix: number | null | undefined): string {
  if (unix === null || unix === undefined || !Number.isFinite(unix)) return ''
  const d = new Date(unix * 1000)
  const pad = (n: number) => String(n).padStart(2, '0')
  return `${d.getFullYear()}-${pad(d.getMonth() + 1)}-${pad(d.getDate())}T${pad(d.getHours())}:${pad(d.getMinutes())}`
}

function datetimeLocalToUnix(value: string): number | null {
  const trimmed = value.trim()
  if (!trimmed) return null
  const ms = new Date(trimmed).getTime()
  if (Number.isNaN(ms)) return null
  return Math.floor(ms / 1000)
}

function hydrate(promo: RechargePromoConfig | null | undefined) {
  if (!promo) {
    enabled.value = false
    startAt.value = ''
    endAt.value = ''
    tiers.value = []
    return
  }
  enabled.value = !!promo.enabled
  startAt.value = unixToDatetimeLocal(promo.start_at)
  endAt.value = unixToDatetimeLocal(promo.end_at)
  tiers.value = (promo.tiers || []).map((tier) => ({
    threshold: String(tier.threshold),
    bonusPercent: String(Math.round(tier.bonus_rate * 10000) / 100),
  }))
}

function addTier() {
  tiers.value.push({ threshold: '', bonusPercent: '' })
}

function removeTier(index: number) {
  tiers.value.splice(index, 1)
}
// PLACEHOLDER_SCRIPT
async function load() {
  loading.value = true
  try {
    hydrate(await adminAPI.rechargePromo.getConfig())
  } catch (err: unknown) {
    appStore.showError(
      extractApiErrorMessage(err, t('admin.rechargePromo.loadFailed')),
    )
  } finally {
    loading.value = false
  }
}

async function save() {
  const startUnix = datetimeLocalToUnix(startAt.value)
  const endUnix = datetimeLocalToUnix(endAt.value)
  if (startUnix !== null && endUnix !== null && endUnix < startUnix) {
    appStore.showError(t('admin.rechargePromo.invalidRange'))
    return
  }
  const payloadTiers: RechargePromoConfig['tiers'] = []
  const seen = new Set<number>()
  for (const row of tiers.value) {
    const thresholdStr = String(row.threshold).trim()
    const bonusStr = String(row.bonusPercent).trim()
    if (thresholdStr === '' && bonusStr === '') continue
    const threshold = Math.round(Number(thresholdStr) * 100) / 100
    const bonusPercent = Number(bonusStr)
    if (
      !Number.isFinite(threshold) ||
      threshold < 0 ||
      !Number.isFinite(bonusPercent) ||
      bonusPercent < 0 ||
      bonusPercent > 500
    ) {
      appStore.showError(t('admin.rechargePromo.invalidTier'))
      return
    }
    if (seen.has(threshold)) {
      appStore.showError(t('admin.rechargePromo.duplicateThreshold'))
      return
    }
    seen.add(threshold)
    payloadTiers.push({
      threshold,
      bonus_rate: Math.round((bonusPercent / 100) * 1e6) / 1e6,
    })
  }
  if (enabled.value && payloadTiers.length === 0) {
    appStore.showError(t('admin.rechargePromo.noTiers'))
    return
  }
  payloadTiers.sort((a, b) => a.threshold - b.threshold)
  saving.value = true
  try {
    const updated = await adminAPI.rechargePromo.updateConfig({
      enabled: enabled.value,
      start_at: startUnix,
      end_at: endUnix,
      tiers: payloadTiers,
    })
    hydrate(updated)
    appStore.showSuccess(t('admin.rechargePromo.saveSuccess'))
  } catch (err: unknown) {
    appStore.showError(
      extractApiErrorMessage(err, t('admin.rechargePromo.saveFailed')),
    )
  } finally {
    saving.value = false
  }
}

onMounted(() => load())
</script>

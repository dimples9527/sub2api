<script setup lang="ts">
import { computed, onBeforeUnmount, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import BaseDialog from '@/components/common/BaseDialog.vue'
import Select from '@/components/common/Select.vue'
import Icon from '@/components/icons/Icon.vue'
import { KEY_TEST_PROMPT, fetchKeyTestModels, runKeyConnectionTest } from '@/api/keyConnectionTest'
import { maskApiKey } from '@/utils/maskApiKey'
import type { ApiKey } from '@/types'

const props = defineProps<{
  show: boolean
  apiKey: ApiKey | null
}>()

const emit = defineEmits<{
  close: []
}>()

const { t } = useI18n()

const models = ref<string[]>([])
const selectedModel = ref('')
const loadingModels = ref(false)
const modelsError = ref('')
const testing = ref(false)
const reply = ref('')
const firstTokenMs = ref<number | null>(null)
const failureMessage = ref('')

// 关弹窗或连点测试时要能打断上一次请求：否则迟到的流会把已经重置的界面重新写脏。
let inFlight: AbortController | null = null

const modelOptions = computed(() => models.value.map((model) => ({ value: model, label: model })))
const canTest = computed(
  () =>
    Boolean(props.apiKey) && Boolean(selectedModel.value) && !testing.value && !loadingModels.value,
)
const firstTokenText = computed(() =>
  firstTokenMs.value === null
    ? ''
    : t('keys.test.firstToken', { seconds: (firstTokenMs.value / 1000).toFixed(2) }),
)
const statusText = computed(() =>
  failureMessage.value
    ? `${t('keys.test.failed')} · ${failureMessage.value}`
    : t('keys.test.success'),
)

function errorText(error: unknown, fallback: string): string {
  const message = (error as { message?: unknown } | null)?.message
  return typeof message === 'string' && message ? message : fallback
}

function cancelInFlight() {
  inFlight?.abort()
  inFlight = null
}

function resetState() {
  cancelInFlight()
  models.value = []
  selectedModel.value = ''
  loadingModels.value = false
  modelsError.value = ''
  testing.value = false
  reply.value = ''
  firstTokenMs.value = null
  failureMessage.value = ''
}

async function loadModels() {
  const key = props.apiKey
  if (!key) return

  const controller = new AbortController()
  inFlight = controller
  loadingModels.value = true
  modelsError.value = ''

  try {
    const list = await fetchKeyTestModels(key.key, { signal: controller.signal })
    if (controller.signal.aborted) return
    models.value = list
    selectedModel.value = list[0] ?? ''
    if (list.length === 0) modelsError.value = t('keys.test.noModels')
  } catch (error) {
    if (controller.signal.aborted) return
    modelsError.value = t('keys.test.modelsFailed', {
      reason: errorText(error, t('keys.test.requestFailed')),
    })
  } finally {
    if (!controller.signal.aborted) loadingModels.value = false
    if (inFlight === controller) inFlight = null
  }
}

async function runTest() {
  const key = props.apiKey
  if (!key || !selectedModel.value || testing.value) return

  cancelInFlight()
  const controller = new AbortController()
  inFlight = controller

  testing.value = true
  reply.value = ''
  firstTokenMs.value = null
  failureMessage.value = ''

  try {
    const result = await runKeyConnectionTest(
      key.key,
      selectedModel.value,
      {
        onFirstToken: (elapsedMs) => {
          if (!controller.signal.aborted) firstTokenMs.value = elapsedMs
        },
        onDelta: (text) => {
          if (!controller.signal.aborted) reply.value = text
        },
      },
      { signal: controller.signal },
    )
    if (controller.signal.aborted) return

    // 收到 200 却一个文字片段都没有，不能算成功 —— 否则就是「测试成功但没内容」的静默失败。
    if (!result.text.trim()) {
      failureMessage.value = t('keys.test.emptyResponse')
    }
  } catch (error) {
    if (controller.signal.aborted) return
    failureMessage.value = errorText(error, t('keys.test.requestFailed'))
  } finally {
    if (!controller.signal.aborted) testing.value = false
    if (inFlight === controller) inFlight = null
  }
}

function close() {
  cancelInFlight()
  emit('close')
}

watch(
  () => props.show,
  (show) => {
    if (!show) return
    resetState()
    void loadModels()
  },
  { immediate: true },
)

onBeforeUnmount(cancelInFlight)
</script>

<template>
  <BaseDialog :show="show" :title="t('keys.test.title')" width="normal" @close="close">
    <div class="space-y-4">
      <!-- 密钥信息 -->
      <div
        v-if="apiKey"
        class="flex items-center justify-between gap-3 rounded-xl border border-gray-200 bg-gray-50 p-3 dark:border-dark-500 dark:bg-dark-700"
      >
        <div class="min-w-0">
          <div class="truncate text-sm font-semibold text-gray-900 dark:text-gray-100">
            {{ apiKey.name }}
          </div>
          <div
            class="mt-0.5 flex flex-wrap items-center gap-x-2 gap-y-1 text-xs text-gray-500 dark:text-gray-400"
          >
            <code class="font-mono">{{ maskApiKey(apiKey.key) }}</code>
            <span class="text-gray-300 dark:text-dark-500">·</span>
            <span>{{ apiKey.group?.name ?? t('keys.noGroup') }}</span>
          </div>
        </div>
        <!-- 状态徽标与列表页保持一致：文案走 keys.status.*，样式走 badge-*，
             否则「额度耗尽 / 已过期」会被错显成「禁用」。 -->
        <span
          :class="[
            'badge shrink-0',
            apiKey.status === 'active'
              ? 'badge-success'
              : apiKey.status === 'quota_exhausted'
                ? 'badge-warning'
                : apiKey.status === 'expired'
                  ? 'badge-danger'
                  : 'badge-gray',
          ]"
        >
          {{ t('keys.status.' + apiKey.status) }}
        </span>
      </div>

      <p class="text-xs leading-5 text-gray-500 dark:text-gray-400">{{ t('keys.test.hint') }}</p>

      <!-- 测试模型 -->
      <div class="space-y-1.5">
        <label class="text-sm font-medium text-gray-700 dark:text-gray-300">
          {{ t('keys.test.modelLabel') }}
        </label>
        <div class="flex items-center gap-2">
          <Select
            v-model="selectedModel"
            class="flex-1"
            :options="modelOptions"
            :disabled="loadingModels || testing"
            :placeholder="
              loadingModels ? t('keys.test.loadingModels') : t('keys.test.modelPlaceholder')
            "
            :aria-label="t('keys.test.modelLabel')"
            searchable
            data-test="key-test-model"
          />
          <button
            type="button"
            class="btn btn-primary shrink-0"
            :disabled="!canTest"
            data-test="key-test-run"
            @click="runTest"
          >
            {{ testing ? t('keys.test.testing') : t('keys.test.run') }}
          </button>
        </div>
        <p v-if="modelsError" class="text-xs text-amber-600 dark:text-amber-400">
          {{ modelsError }}
        </p>
      </div>

      <!-- 测试结果 -->
      <div
        v-if="testing || reply || failureMessage"
        class="space-y-3 rounded-xl border border-gray-200 p-3 dark:border-dark-500"
      >
        <!-- 实际发出的固定短消息 -->
        <div class="flex justify-end">
          <div
            class="max-w-[80%] break-words rounded-2xl rounded-br-sm bg-primary-50 px-3 py-2 text-sm text-gray-800 dark:bg-primary-900/30 dark:text-gray-100"
          >
            {{ KEY_TEST_PROMPT }}
          </div>
        </div>

        <div v-if="reply" class="space-y-1.5">
          <div class="flex flex-wrap items-center gap-2 text-xs text-gray-500 dark:text-gray-400">
            <span class="font-medium text-gray-700 dark:text-gray-300">{{ selectedModel }}</span>
            <span v-if="firstTokenText" class="rounded bg-gray-100 px-1.5 py-0.5 dark:bg-dark-600">
              {{ firstTokenText }}
            </span>
          </div>
          <div
            class="max-w-[90%] whitespace-pre-wrap break-words rounded-2xl rounded-bl-sm bg-gray-100 px-3 py-2 text-sm text-gray-800 dark:bg-dark-700 dark:text-gray-100"
          >
            {{ reply }}
          </div>
        </div>
        <p v-else-if="testing" class="text-sm text-gray-500 dark:text-gray-400">
          {{ t('keys.test.waitingFirstToken') }}
        </p>

        <p
          v-if="!testing"
          class="flex items-center gap-1.5 text-xs font-medium"
          :class="
            failureMessage
              ? 'text-red-600 dark:text-red-400'
              : 'text-green-600 dark:text-green-400'
          "
        >
          <Icon :name="failureMessage ? 'xCircle' : 'checkCircle'" size="sm" />
          {{ statusText }}
        </p>
      </div>
    </div>

    <template #footer>
      <button type="button" class="btn btn-secondary" @click="close">
        {{ t('common.close') }}
      </button>
    </template>
  </BaseDialog>
</template>

<template>
  <AppLayout>
    <div class="image-page">
      <header class="flex flex-col gap-3 border-b border-gray-200/70 px-4 py-4 dark:border-dark-700/70 sm:px-6 lg:flex-row lg:items-center lg:justify-between">
        <div class="min-w-0">
          <div class="mb-2 flex items-center gap-2 text-xs font-semibold uppercase text-gray-400 dark:text-gray-500">
            <span>MIAOAPI</span>
            <span>/</span>
            <span class="text-primary-600 dark:text-primary-400">{{ t('imageGeneration.navCrumb') }}</span>
          </div>
          <div class="flex items-center gap-3">
            <div class="flex h-10 w-10 items-center justify-center rounded-xl border border-primary-100 bg-primary-50 text-primary-600 shadow-sm dark:border-primary-900/40 dark:bg-primary-900/20 dark:text-primary-300">
              <Icon name="image" size="md" />
            </div>
            <div class="min-w-0">
              <h1 class="truncate text-xl font-bold text-gray-950 dark:text-white">{{ t('imageGeneration.title') }}</h1>
              <p class="mt-1 text-sm text-gray-500 dark:text-gray-400">{{ t('imageGeneration.description') }}</p>
            </div>
          </div>
        </div>

        <div class="flex flex-wrap items-center gap-2">
          <span class="inline-flex items-center gap-2 rounded-full bg-white px-3 py-1.5 text-xs font-medium text-gray-600 shadow-sm ring-1 ring-gray-200 dark:bg-dark-800 dark:text-gray-300 dark:ring-dark-600">
            <span class="h-2 w-2 rounded-full bg-primary-500"></span>
            {{ t('imageGeneration.serviceOnline') }}
          </span>
          <button class="btn btn-secondary btn-sm" type="button" @click="loadKeys" :disabled="loadingKeys">
            <Icon name="refresh" size="sm" :class="loadingKeys ? 'animate-spin' : ''" />
            {{ t('common.refresh') }}
          </button>
        </div>
      </header>

      <main class="image-workspace">
        <section v-if="history.length === 0 && !generating" class="empty-panel">
          <div class="flex h-16 w-16 items-center justify-center rounded-3xl border border-gray-200 bg-white text-primary-600 shadow-lg shadow-primary-500/10 dark:border-dark-700 dark:bg-dark-800 dark:text-primary-300">
            <Icon name="image" size="xl" />
          </div>
          <h2>{{ t('imageGeneration.emptyTitle') }}</h2>
          <p>{{ t('imageGeneration.emptyDescription') }}</p>
          <div class="flex flex-wrap justify-center gap-2">
            <button
              v-for="sample in promptSamples"
              :key="sample"
              class="sample-chip"
              type="button"
              @click="prompt = sample"
            >
              {{ sample }}
            </button>
          </div>
        </section>

        <section v-else class="mx-auto flex w-full max-w-6xl flex-col gap-5 px-4 py-6 sm:px-6">
          <div v-if="generating" class="generation-status">
            <Icon name="sparkles" size="lg" class="animate-pulse text-primary-600 dark:text-primary-300" />
            <div>
              <p class="font-semibold text-gray-900 dark:text-white">{{ t('imageGeneration.generating') }}</p>
              <p class="text-sm text-gray-500 dark:text-gray-400">{{ t('imageGeneration.generatingHint') }}</p>
            </div>
          </div>

          <article v-for="item in history" :key="item.id" class="result-group">
            <div class="flex flex-col gap-2 border-b border-gray-100 px-4 py-3 dark:border-dark-700 sm:flex-row sm:items-center sm:justify-between">
              <div class="min-w-0">
                <p class="truncate text-sm font-semibold text-gray-900 dark:text-white">{{ item.prompt }}</p>
                <p class="mt-1 text-xs text-gray-500 dark:text-gray-400">
                  {{ item.model }} / {{ item.size }} / {{ item.quality }} / {{ formatTime(item.createdAt) }}
                </p>
              </div>
              <button class="btn btn-secondary btn-sm" type="button" @click="reusePrompt(item.prompt)">
                <Icon name="copy" size="sm" />
                {{ t('imageGeneration.reusePrompt') }}
              </button>
            </div>

            <div class="grid gap-4 p-4 sm:grid-cols-2 xl:grid-cols-3">
              <div v-for="image in item.images" :key="image.id" class="image-card">
                <img :src="image.src" :alt="item.prompt" class="h-full w-full object-cover" />
                <div class="image-actions">
                  <button class="icon-action" type="button" :title="t('imageGeneration.copyImageUrl')" @click="copyImageSource(image.src)">
                    <Icon name="copy" size="sm" />
                  </button>
                  <a class="icon-action" :href="image.src" :download="downloadName(item, image.id)" :title="t('imageGeneration.download')">
                    <Icon name="download" size="sm" />
                  </a>
                </div>
                <p v-if="image.revisedPrompt" class="line-clamp-2 border-t border-gray-100 px-3 py-2 text-xs text-gray-500 dark:border-dark-700 dark:text-gray-400">
                  {{ image.revisedPrompt }}
                </p>
              </div>
            </div>
          </article>
        </section>
      </main>

      <footer class="composer-shell">
        <div v-if="errorMessage" class="mb-3 rounded-xl border border-red-200 bg-red-50 px-4 py-3 text-sm text-red-700 dark:border-red-900/60 dark:bg-red-950/30 dark:text-red-300">
          {{ errorMessage }}
        </div>

        <div v-if="availableKeys.length === 0" class="mb-3 rounded-xl border border-amber-200 bg-amber-50 px-4 py-3 text-sm text-amber-800 dark:border-amber-900/60 dark:bg-amber-950/30 dark:text-amber-200">
          {{ loadingKeys ? t('imageGeneration.loadingKeys') : t('imageGeneration.noKeys') }}
        </div>

        <div class="mb-3 flex flex-wrap items-center gap-2">
          <label class="control-pill min-w-[220px]">
            <Icon name="key" size="sm" />
            <select v-model.number="selectedKeyId" :disabled="loadingKeys || generating">
              <option :value="0">{{ t('imageGeneration.selectKey') }}</option>
              <option v-for="key in availableKeys" :key="key.id" :value="key.id">
                {{ key.name }} / {{ key.group?.name || 'OpenAI' }}
              </option>
            </select>
          </label>

          <label class="control-pill">
            <span>{{ t('imageGeneration.model') }}</span>
            <select v-model="form.model" :disabled="generating">
              <option v-for="model in modelOptions" :key="model" :value="model">{{ model }}</option>
            </select>
          </label>

          <label class="control-pill">
            <span>{{ t('imageGeneration.size') }}</span>
            <select v-model="form.size" :disabled="generating">
              <option v-for="size in sizeOptions" :key="size" :value="size">{{ size }}</option>
            </select>
          </label>

          <label class="control-pill">
            <span>{{ t('imageGeneration.count') }}</span>
            <select v-model.number="form.count" :disabled="generating">
              <option :value="1">1</option>
              <option :value="2">2</option>
              <option :value="4">4</option>
            </select>
          </label>

          <label class="control-pill">
            <span>{{ t('imageGeneration.quality') }}</span>
            <select v-model="form.quality" :disabled="generating">
              <option v-for="quality in qualityOptions" :key="quality" :value="quality">
                {{ t(`imageGeneration.qualities.${quality}`) }}
              </option>
            </select>
          </label>
        </div>

        <form class="prompt-box" @submit.prevent="submit">
          <textarea
            v-model="prompt"
            :placeholder="t('imageGeneration.promptPlaceholder')"
            :disabled="generating"
            rows="2"
            @keydown.enter.exact.prevent="submit"
          ></textarea>
          <button v-if="generating" class="send-button cancel" type="button" @click="cancelGeneration" :title="t('common.cancel')">
            <Icon name="x" size="lg" />
          </button>
          <button v-else class="send-button" type="submit" :disabled="!canSubmit" :title="t('imageGeneration.generate')">
            <Icon name="arrowUp" size="lg" />
          </button>
        </form>
        <p class="mt-2 text-xs text-gray-400 dark:text-gray-500">{{ t('imageGeneration.submitHint') }}</p>
      </footer>
    </div>
  </AppLayout>
</template>

<script setup lang="ts">
import { computed, onMounted, reactive, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import AppLayout from '@/components/layout/AppLayout.vue'
import Icon from '@/components/icons/Icon.vue'
import keysAPI from '@/api/keys'
import type { ApiKey } from '@/types'
import { useAppStore } from '@/stores/app'
import { useClipboard } from '@/composables/useClipboard'
import {
  buildImageGenerationPayload,
  generateImages,
  getImageCapableOpenAIKeys,
  type GeneratedImage,
  type ImageQuality,
  type ImageSize,
} from '@/api/imageGeneration'

interface HistoryItem {
  id: string
  prompt: string
  model: string
  size: ImageSize
  quality: ImageQuality
  createdAt: Date
  images: GeneratedImage[]
}

const { t } = useI18n()
const appStore = useAppStore()
const { copyToClipboard } = useClipboard()

const modelOptions = ['gpt-image-2', 'gpt-image-1']
const sizeOptions: ImageSize[] = ['1024x1024', '1024x1536', '1536x1024']
const qualityOptions: ImageQuality[] = ['auto', 'low', 'medium', 'high']
const promptSamples = computed(() => [
  t('imageGeneration.samples.glass'),
  t('imageGeneration.samples.portrait'),
  t('imageGeneration.samples.sunrise'),
  t('imageGeneration.samples.logo'),
])

const keys = ref<ApiKey[]>([])
const selectedKeyId = ref(0)
const loadingKeys = ref(false)
const generating = ref(false)
const errorMessage = ref('')
const history = ref<HistoryItem[]>([])
const prompt = ref('')
let generationController: AbortController | null = null

const form = reactive<{
  model: string
  size: ImageSize
  count: number
  quality: ImageQuality
}>({
  model: modelOptions[0],
  size: sizeOptions[0],
  count: 1,
  quality: 'auto',
})

const availableKeys = computed(() => getImageCapableOpenAIKeys(keys.value))
const selectedKey = computed(() => availableKeys.value.find((key) => key.id === selectedKeyId.value) ?? null)
const canSubmit = computed(() => prompt.value.trim().length > 0 && !!selectedKey.value && !generating.value)

async function loadKeys() {
  loadingKeys.value = true
  errorMessage.value = ''
  try {
    const response = await keysAPI.list(1, 200, { status: 'active' })
    keys.value = response.items
    if (!selectedKey.value && availableKeys.value.length > 0) {
      selectedKeyId.value = availableKeys.value[0].id
    }
  } catch (error) {
    console.error('Failed to load image generation keys:', error)
    appStore.showError(t('imageGeneration.loadKeysFailed'))
  } finally {
    loadingKeys.value = false
  }
}

async function submit() {
  if (!canSubmit.value || !selectedKey.value) return

  const currentPrompt = prompt.value.trim()
  const payload = buildImageGenerationPayload({
    model: form.model,
    prompt: currentPrompt,
    size: form.size,
    count: form.count,
    quality: form.quality,
  })

  generationController?.abort()
  generationController = new AbortController()
  generating.value = true
  errorMessage.value = ''

  try {
    const images = await generateImages(selectedKey.value.key, payload, {
      signal: generationController.signal,
    })
    if (images.length === 0) {
      throw new Error(t('imageGeneration.emptyResponse'))
    }
    history.value.unshift({
      id: `${Date.now()}-${Math.random().toString(36).slice(2, 8)}`,
      prompt: currentPrompt,
      model: form.model,
      size: form.size,
      quality: form.quality,
      createdAt: new Date(),
      images,
    })
    appStore.showSuccess(t('imageGeneration.generateSuccess'))
  } catch (error) {
    if (isAbortError(error)) return
    errorMessage.value = extractErrorMessage(error)
  } finally {
    generating.value = false
    generationController = null
  }
}

function cancelGeneration() {
  generationController?.abort()
  generationController = null
  generating.value = false
}

function reusePrompt(value: string) {
  prompt.value = value
}

async function copyImageSource(src: string) {
  await copyToClipboard(src, t('imageGeneration.copied'))
}

function downloadName(item: HistoryItem, imageId: string) {
  return `image-${item.createdAt.getTime()}-${imageId}.png`
}

function formatTime(date: Date) {
  return new Intl.DateTimeFormat(undefined, {
    hour: '2-digit',
    minute: '2-digit',
  }).format(date)
}

function isAbortError(error: unknown) {
  if (!error || typeof error !== 'object') return false
  const err = error as { name?: string; code?: string }
  return err.name === 'AbortError' || err.code === 'ERR_CANCELED'
}

function extractErrorMessage(error: unknown) {
  if (error instanceof Error && error.message) {
    return error.message
  }
  if (error && typeof error === 'object' && 'message' in error) {
    return String((error as { message?: unknown }).message || t('imageGeneration.generateFailed'))
  }
  return t('imageGeneration.generateFailed')
}

onMounted(loadKeys)
</script>

<style scoped>
.image-page {
  min-height: calc(100vh - 4rem);
  margin: -1rem;
  display: flex;
  flex-direction: column;
  background:
    linear-gradient(180deg, rgba(248, 250, 252, 0.96), rgba(243, 244, 246, 0.94)),
    radial-gradient(circle at 50% 0%, rgba(99, 102, 241, 0.09), transparent 34%);
}

.dark .image-page {
  background:
    linear-gradient(180deg, rgba(9, 13, 22, 0.98), rgba(15, 23, 42, 0.96)),
    radial-gradient(circle at 50% 0%, rgba(99, 102, 241, 0.16), transparent 38%);
}

.image-workspace {
  flex: 1 1 auto;
  display: flex;
  align-items: stretch;
  padding-bottom: 20rem;
}

.empty-panel {
  margin: auto;
  display: flex;
  max-width: 620px;
  flex-direction: column;
  align-items: center;
  gap: 1rem;
  padding: 2rem 1rem;
  text-align: center;
}

.empty-panel h2 {
  font-size: 1.5rem;
  font-weight: 800;
  color: rgb(17 24 39);
}

.dark .empty-panel h2 {
  color: white;
}

.empty-panel p {
  max-width: 460px;
  color: rgb(107 114 128);
  line-height: 1.8;
}

.dark .empty-panel p {
  color: rgb(156 163 175);
}

.sample-chip {
  border-radius: 999px;
  border: 1px solid rgb(229 231 235);
  background: white;
  padding: 0.5rem 0.875rem;
  font-size: 0.8125rem;
  color: rgb(75 85 99);
  box-shadow: 0 1px 2px rgba(15, 23, 42, 0.04);
  transition: all 0.16s ease;
}

.sample-chip:hover {
  border-color: rgb(99 102 241);
  color: rgb(79 70 229);
}

.dark .sample-chip {
  border-color: rgb(55 65 81);
  background: rgb(31 41 55);
  color: rgb(209 213 219);
}

.generation-status,
.result-group {
  overflow: hidden;
  border-radius: 1.25rem;
  border: 1px solid rgba(229, 231, 235, 0.8);
  background: rgba(255, 255, 255, 0.88);
  box-shadow: 0 18px 45px rgba(15, 23, 42, 0.07);
}

.generation-status {
  display: flex;
  align-items: center;
  gap: 1rem;
  padding: 1rem 1.25rem;
}

.dark .generation-status,
.dark .result-group {
  border-color: rgba(55, 65, 81, 0.8);
  background: rgba(31, 41, 55, 0.82);
}

.image-card {
  position: relative;
  overflow: hidden;
  border-radius: 1rem;
  border: 1px solid rgb(229 231 235);
  background: rgb(249 250 251);
  aspect-ratio: 1 / 1;
}

.dark .image-card {
  border-color: rgb(55 65 81);
  background: rgb(17 24 39);
}

.image-actions {
  position: absolute;
  right: 0.625rem;
  top: 0.625rem;
  display: flex;
  gap: 0.5rem;
}

.icon-action {
  display: inline-flex;
  height: 2.25rem;
  width: 2.25rem;
  align-items: center;
  justify-content: center;
  border-radius: 999px;
  background: rgba(255, 255, 255, 0.88);
  color: rgb(55 65 81);
  box-shadow: 0 8px 20px rgba(15, 23, 42, 0.18);
  backdrop-filter: blur(12px);
}

.icon-action:hover {
  color: rgb(79 70 229);
}

.composer-shell {
  position: sticky;
  bottom: 0;
  z-index: 10;
  border-top: 1px solid rgba(229, 231, 235, 0.8);
  background: rgba(249, 250, 251, 0.92);
  padding: 0.75rem 1rem 1rem;
  backdrop-filter: blur(18px);
}

.dark .composer-shell {
  border-color: rgba(55, 65, 81, 0.8);
  background: rgba(9, 13, 22, 0.9);
}

.control-pill {
  display: inline-flex;
  align-items: center;
  gap: 0.5rem;
  border-radius: 999px;
  border: 1px solid rgb(229 231 235);
  background: white;
  padding: 0.5rem 0.75rem;
  font-size: 0.8125rem;
  color: rgb(75 85 99);
  box-shadow: 0 1px 2px rgba(15, 23, 42, 0.04);
}

.dark .control-pill {
  border-color: rgb(55 65 81);
  background: rgb(31 41 55);
  color: rgb(209 213 219);
}

.control-pill select {
  max-width: 16rem;
  min-width: 0;
  border: 0;
  background: transparent;
  color: inherit;
  outline: none;
}

.prompt-box {
  display: flex;
  align-items: flex-end;
  gap: 0.75rem;
  border-radius: 1.25rem;
  border: 1px solid rgb(229 231 235);
  background: white;
  padding: 0.625rem;
  box-shadow: 0 12px 32px rgba(15, 23, 42, 0.1);
}

.dark .prompt-box {
  border-color: rgb(55 65 81);
  background: rgb(31 41 55);
}

.prompt-box textarea {
  min-height: 3rem;
  flex: 1;
  resize: none;
  border: 0;
  background: transparent;
  padding: 0.75rem 0.875rem;
  color: rgb(17 24 39);
  outline: none;
}

.dark .prompt-box textarea {
  color: white;
}

.send-button {
  display: inline-flex;
  height: 3rem;
  width: 3rem;
  flex: 0 0 3rem;
  align-items: center;
  justify-content: center;
  border-radius: 1rem;
  background: linear-gradient(135deg, rgb(99 102 241), rgb(79 70 229));
  color: white;
  box-shadow: 0 12px 24px rgba(79, 70, 229, 0.28);
}

.send-button:disabled {
  cursor: not-allowed;
  opacity: 0.45;
}

.send-button.cancel {
  background: linear-gradient(135deg, rgb(239 68 68), rgb(220 38 38));
  box-shadow: 0 12px 24px rgba(220, 38, 38, 0.24);
}

@media (min-width: 768px) {
  .image-page {
    margin: -1.5rem;
  }

  .composer-shell {
    padding-left: 1.5rem;
    padding-right: 1.5rem;
  }

  .image-workspace {
    padding-bottom: 15rem;
  }
}

@media (min-width: 1024px) {
  .image-page {
    margin: -2rem;
  }

  .composer-shell {
    padding-left: 2rem;
    padding-right: 2rem;
  }

  .image-workspace {
    padding-bottom: 13.5rem;
  }
}
</style>

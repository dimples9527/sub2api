<template>
  <AppLayout>
    <div class="image-page">
      <header class="flex flex-col gap-3 border-b border-gray-200/70 px-4 py-4 dark:border-dark-700/70 sm:px-6 lg:flex-row lg:items-center lg:justify-between">
        <div class="min-w-0">
          <div class="mb-2 flex items-center gap-2 text-xs font-semibold uppercase text-gray-400 dark:text-gray-500">
            <span>SUNSHINE API</span>
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

        <section v-else class="chat-thread">
          <div v-if="generating" class="chat-turn assistant-turn generation-status">
            <div class="status-badge">AI</div>
            <div class="assistant-bubble generation-card">
              <div class="generation-canvas">
                <Icon name="sparkles" size="xl" class="animate-pulse text-primary-500 dark:text-primary-300" />
              </div>
              <div class="generation-progress">
                <span></span>
              </div>
              <p>{{ t('imageGeneration.generating') }} / {{ generationStageLabel }} / {{ t('imageGeneration.elapsed', { seconds: elapsedSeconds }) }}</p>
              <p class="generation-hint">{{ t('imageGeneration.generatingHint') }}</p>
            </div>
          </div>

          <article v-for="item in history" :key="item.id" class="result-group">
            <div class="chat-turn user-turn">
              <div class="user-bubble">
                <p>{{ item.prompt }}</p>
                <p class="mt-1 text-xs text-gray-500 dark:text-gray-400">
                  {{ item.model }} / {{ item.size }} / {{ item.quality }} / {{ t('imageGeneration.elapsed', { seconds: item.elapsedSeconds }) }} / {{ formatTime(item.createdAt) }}
                </p>
              </div>
            </div>

            <div class="chat-turn assistant-turn">
              <div class="status-badge">AI</div>
              <div class="assistant-bubble">
                <div class="assistant-toolbar">
                  <span>{{ item.model }}</span>
                  <div class="flex flex-wrap items-center gap-2">
                    <button
                      class="btn btn-secondary btn-sm"
                      type="button"
                      data-testid="continue-edit-image"
                      @click="continueEdit(item)"
                    >
                      <Icon name="edit" size="sm" />
                      {{ t('imageGeneration.continueEdit') }}
                    </button>
                    <button class="btn btn-secondary btn-sm" type="button" @click="reusePrompt(item.prompt)">
                      <Icon name="copy" size="sm" />
                      {{ t('imageGeneration.reusePrompt') }}
                    </button>
                    <button
                      v-if="item.images.length > 1"
                      class="btn btn-secondary btn-sm"
                      type="button"
                      data-testid="download-all-images"
                      @click="downloadAll(item)"
                    >
                      <Icon name="download" size="sm" />
                      {{ t('imageGeneration.downloadAll') }}
                    </button>
                  </div>
                </div>

                <div class="image-grid">
                  <div v-for="image in item.images" :key="image.id" class="image-card">
                    <button
                      class="image-preview-trigger"
                      type="button"
                      data-testid="generated-image-button"
                      :title="t('imageGeneration.viewImage')"
                      @click="openPreview(item, image)"
                    >
                      <img :src="image.src" :alt="item.prompt" class="h-full w-full object-cover" />
                    </button>
                    <div class="image-actions">
                      <button class="icon-action" type="button" :title="t('imageGeneration.viewImage')" @click="openPreview(item, image)">
                        <Icon name="eye" size="sm" />
                      </button>
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
              </div>
            </div>
          </article>
        </section>
      </main>

      <footer class="composer-shell">
        <div v-if="errorMessage" class="mb-3 rounded-xl border border-red-200 bg-red-50 px-4 py-3 text-sm text-red-700 dark:border-red-900/60 dark:bg-red-950/30 dark:text-red-300">
          <div class="flex flex-col gap-2 sm:flex-row sm:items-center sm:justify-between">
            <span>{{ errorMessage }}</span>
            <button
              v-if="lastFailedRequest"
              class="btn btn-secondary btn-sm"
              type="button"
              data-testid="retry-image-generation"
              :disabled="generating"
              @click="retryLastFailedGeneration"
            >
              <Icon name="refresh" size="sm" />
              {{ t('imageGeneration.retry') }}
            </button>
          </div>
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
            <Icon name="cpu" size="sm" />
            <span>{{ t('imageGeneration.model') }}</span>
            <select v-model="form.model" :disabled="generating || loadingModels">
              <option v-for="model in modelOptions" :key="model" :value="model">{{ model }}</option>
            </select>
          </label>

          <label class="control-pill">
            <Icon name="image" size="sm" />
            <span>{{ t('imageGeneration.size') }}</span>
            <select v-model="form.size" :disabled="generating">
              <option v-for="size in displaySizeOptions" :key="size.value" :value="size.value">{{ size.label }}</option>
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
          <div
            v-if="referenceImage"
            class="reference-image-preview"
            data-testid="reference-image-preview"
          >
            <img :src="referenceImage.src" :alt="referenceImage.prompt" />
            <button
              class="reference-remove"
              type="button"
              :title="t('imageGeneration.removeReferenceImage')"
              data-testid="remove-reference-image"
              @click="clearReferenceImage"
            >
              <Icon name="x" size="sm" />
            </button>
          </div>
          <textarea
            v-model="prompt"
            :placeholder="promptPlaceholder"
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

      <Teleport to="body">
        <div
          v-if="previewImage"
          class="image-preview-backdrop"
          data-testid="image-preview-dialog"
          :data-preview-owner="previewDialogId"
          role="dialog"
          aria-modal="true"
          :aria-label="t('imageGeneration.viewImage')"
          @click.self="closePreview"
        >
          <div class="image-preview-panel">
            <button class="preview-close" type="button" :title="t('imageGeneration.closePreview')" @click="closePreview">
              <Icon name="x" size="lg" />
            </button>
            <button
              v-if="previewImage.total > 1"
              class="preview-nav previous"
              type="button"
              :title="t('imageGeneration.previousImage')"
              @click="showPreviousPreviewImage"
            >
              <Icon name="chevronLeft" size="lg" />
            </button>
            <img :src="previewImage.src" :alt="previewImage.prompt" />
            <button
              v-if="previewImage.total > 1"
              class="preview-nav next"
              type="button"
              :title="t('imageGeneration.nextImage')"
              @click="showNextPreviewImage"
            >
              <Icon name="chevronRight" size="lg" />
            </button>
            <div class="preview-footer">
              <div class="preview-summary">
                <p>
                  <span v-if="previewImage.total > 1">{{ previewImage.index + 1 }} / {{ previewImage.total }} - </span>
                  {{ previewImage.prompt }}
                </p>
                <dl class="preview-details">
                  <div>
                    <dt>{{ t('imageGeneration.previewDetails.model') }}</dt>
                    <dd>{{ previewImage.model }}</dd>
                  </div>
                  <div>
                    <dt>{{ t('imageGeneration.previewDetails.size') }}</dt>
                    <dd>{{ previewImage.size }}</dd>
                  </div>
                  <div>
                    <dt>{{ t('imageGeneration.previewDetails.quality') }}</dt>
                    <dd>{{ previewImage.quality }}</dd>
                  </div>
                  <div>
                    <dt>{{ t('imageGeneration.previewDetails.elapsed') }}</dt>
                    <dd>{{ t('imageGeneration.elapsed', { seconds: previewImage.elapsedSeconds }) }}</dd>
                  </div>
                  <div>
                    <dt>{{ t('imageGeneration.previewDetails.createdAt') }}</dt>
                    <dd>{{ formatTime(previewImage.createdAt) }}</dd>
                  </div>
                </dl>
              </div>
              <a class="preview-download" :href="previewImage.src" :download="previewImage.downloadName">
                <Icon name="download" size="sm" />
                {{ t('imageGeneration.download') }}
              </a>
            </div>
          </div>
        </div>
      </Teleport>
    </div>
  </AppLayout>
</template>

<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, reactive, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import AppLayout from '@/components/layout/AppLayout.vue'
import Icon from '@/components/icons/Icon.vue'
import keysAPI from '@/api/keys'
import type { ApiKey } from '@/types'
import { useAppStore } from '@/stores/app'
import { useClipboard } from '@/composables/useClipboard'
import {
  buildImageGenerationPayload,
  fetchImageModelOptions,
  generateImageEdit,
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
  elapsedSeconds: string
  createdAt: Date
  images: GeneratedImage[]
}

interface StoredHistoryItem {
  id: string
  prompt: string
  model: string
  size: ImageSize
  quality: ImageQuality
  elapsedSeconds: string
  createdAt: string
  images: GeneratedImage[]
}

interface GenerationRequestSnapshot {
  prompt: string
  model: string
  size: ImageSize
  count: number
  quality: ImageQuality
  referenceImage?: ReferenceImage
}

interface ReferenceImage {
  src: string
  prompt: string
  imageId: string
}

const { t } = useI18n()
const appStore = useAppStore()
const { copyToClipboard } = useClipboard()

const HISTORY_STORAGE_KEY = 'sunshine:image-generation-history'
const MAX_STORED_HISTORY_ITEMS = 50
const defaultModelOptions = ['gpt-image-2', 'gpt-image-1']
const sizeOptions: ImageSize[] = ['1920x1088', '2560x1440', '3840x2160']
const displaySizeOptions: Array<{ value: ImageSize; label: string }> = [
  { value: '1920x1088', label: '1K - 1920x1088' },
  { value: '2560x1440', label: '2K - 2560x1440' },
  { value: '3840x2160', label: '4K - 3840x2160' },
]
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
const loadingModels = ref(false)
const generating = ref(false)
const errorMessage = ref('')
const history = ref<HistoryItem[]>([])
const prompt = ref('')
const elapsedMs = ref(0)
const previewDialogId = `image-preview-${Math.random().toString(36).slice(2, 10)}`
const previewState = ref<{ itemId: string; imageIndex: number } | null>(null)
const lastFailedRequest = ref<GenerationRequestSnapshot | null>(null)
const referenceImage = ref<ReferenceImage | null>(null)
let generationController: AbortController | null = null
let generationStartAt = 0
let elapsedTimer: number | null = null

const form = reactive<{
  model: string
  size: ImageSize
  count: number
  quality: ImageQuality
}>({
  model: defaultModelOptions[0],
  size: sizeOptions[0],
  count: 1,
  quality: 'auto',
})

const availableKeys = computed(() => getImageCapableOpenAIKeys(keys.value))
const selectedKey = computed(() => availableKeys.value.find((key) => key.id === selectedKeyId.value) ?? null)
const modelOptions = ref([...defaultModelOptions])
const canSubmit = computed(() => prompt.value.trim().length > 0 && !!selectedKey.value && !generating.value)
const promptPlaceholder = computed(() =>
  referenceImage.value ? t('imageGeneration.editPromptPlaceholder') : t('imageGeneration.promptPlaceholder'),
)
const elapsedSeconds = computed(() => (elapsedMs.value / 1000).toFixed(1))
const generationStageLabel = computed(() => {
  if (elapsedMs.value < 2000) return t('imageGeneration.stage.preparing')
  if (elapsedMs.value < 8000) return t('imageGeneration.stage.queued')
  if (elapsedMs.value < 45000) return t('imageGeneration.stage.generating')
  return t('imageGeneration.stage.finishing')
})
const previewItem = computed(() => {
  if (!previewState.value) return null
  return history.value.find((item) => item.id === previewState.value?.itemId) ?? null
})
const previewImage = computed(() => {
  if (!previewState.value || !previewItem.value) return null

  const imageIndex = normalizeImageIndex(previewState.value.imageIndex, previewItem.value.images.length)
  const image = previewItem.value.images[imageIndex]
  if (!image) return null

  return {
    src: image.src,
    prompt: image.revisedPrompt || previewItem.value.prompt,
    downloadName: downloadName(previewItem.value, image.id),
    index: imageIndex,
    total: previewItem.value.images.length,
    model: previewItem.value.model,
    size: previewItem.value.size,
    quality: previewItem.value.quality,
    elapsedSeconds: previewItem.value.elapsedSeconds,
    createdAt: previewItem.value.createdAt,
  }
})

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

async function loadModelOptionsForSelectedKey() {
  const key = selectedKey.value
  if (!key) {
    setModelOptions(defaultModelOptions)
    return
  }

  loadingModels.value = true
  try {
    const models = await fetchImageModelOptions(key.key)
    setModelOptions(models.length > 0 ? models : defaultModelOptions)
  } catch (error) {
    console.error('Failed to load image model options:', error)
    setModelOptions(defaultModelOptions)
  } finally {
    loadingModels.value = false
  }
}

function setModelOptions(models: string[]) {
  const normalizedModels = models.map((model) => model.trim()).filter(Boolean)
  modelOptions.value = normalizedModels.length > 0 ? normalizedModels : [...defaultModelOptions]
  if (!modelOptions.value.includes(form.model)) {
    form.model = modelOptions.value[0]
  }
}

async function submit() {
  if (!canSubmit.value || !selectedKey.value) return

  await runGeneration({
    prompt: prompt.value.trim(),
    model: form.model,
    size: form.size,
    count: form.count,
    quality: form.quality,
    referenceImage: referenceImage.value ?? undefined,
  })
}

async function runGeneration(request: GenerationRequestSnapshot) {
  if (!selectedKey.value) return

  generationController?.abort()
  generationController = new AbortController()
  generating.value = true
  startElapsedTimer()
  errorMessage.value = ''
  lastFailedRequest.value = null

  try {
    const images = request.referenceImage
      ? await generateImageEdit(
        selectedKey.value.key,
        {
          ...request,
          image: await sourceToImageFile(request.referenceImage.src, request.referenceImage.imageId),
        },
        { signal: generationController.signal },
      )
      : await generateImages(selectedKey.value.key, buildImageGenerationPayload(request), {
        signal: generationController.signal,
      })
    if (images.length === 0) {
      throw new Error(t('imageGeneration.emptyResponse'))
    }
    const finalElapsedSeconds = elapsedSeconds.value
    history.value.unshift({
      id: `${Date.now()}-${Math.random().toString(36).slice(2, 8)}`,
      prompt: request.prompt,
      model: request.model,
      size: request.size,
      quality: request.quality,
      elapsedSeconds: finalElapsedSeconds,
      createdAt: new Date(),
      images,
    })
    persistHistory()
    if (request.referenceImage) {
      clearReferenceImage()
    }
    appStore.showSuccess(t('imageGeneration.generateSuccess'))
  } catch (error) {
    if (isAbortError(error)) return
    errorMessage.value = extractErrorMessage(error)
    lastFailedRequest.value = request
  } finally {
    generating.value = false
    stopElapsedTimer()
    generationController = null
  }
}

function cancelGeneration() {
  generationController?.abort()
  generationController = null
  generating.value = false
  stopElapsedTimer()
}

function startElapsedTimer() {
  stopElapsedTimer()
  generationStartAt = Date.now()
  elapsedMs.value = 0
  elapsedTimer = window.setInterval(() => {
    elapsedMs.value = Date.now() - generationStartAt
  }, 100)
}

function stopElapsedTimer() {
  if (elapsedTimer !== null) {
    window.clearInterval(elapsedTimer)
    elapsedTimer = null
  }
}

function reusePrompt(value: string) {
  prompt.value = value
}

function continueEdit(item: HistoryItem) {
  const firstImage = item.images[0]
  if (!firstImage) return
  referenceImage.value = {
    src: firstImage.src,
    prompt: firstImage.revisedPrompt || item.prompt,
    imageId: firstImage.id,
  }
  prompt.value = ''
}

async function retryLastFailedGeneration() {
  if (!lastFailedRequest.value) return
  prompt.value = lastFailedRequest.value.prompt
  form.model = lastFailedRequest.value.model
  form.size = lastFailedRequest.value.size
  form.count = lastFailedRequest.value.count
  form.quality = lastFailedRequest.value.quality
  referenceImage.value = lastFailedRequest.value.referenceImage ?? null
  await runGeneration(lastFailedRequest.value)
}

function clearReferenceImage() {
  referenceImage.value = null
}

async function copyImageSource(src: string) {
  await copyToClipboard(src, t('imageGeneration.copied'))
}

function downloadName(item: HistoryItem, imageId: string) {
  return `${slugifyPrompt(item.prompt)}-${item.size}-${formatDownloadTimestamp(item.createdAt)}-${imageId}.png`
}

function openPreview(item: HistoryItem, image: GeneratedImage) {
  previewState.value = {
    itemId: item.id,
    imageIndex: Math.max(item.images.findIndex((current) => current.id === image.id), 0),
  }
}

function closePreview() {
  previewState.value = null
}

function showNextPreviewImage() {
  updatePreviewIndex(1)
}

function showPreviousPreviewImage() {
  updatePreviewIndex(-1)
}

function updatePreviewIndex(offset: number) {
  if (!previewState.value || !previewItem.value) return
  previewState.value = {
    ...previewState.value,
    imageIndex: normalizeImageIndex(previewState.value.imageIndex + offset, previewItem.value.images.length),
  }
}

function normalizeImageIndex(index: number, total: number) {
  if (total <= 0) return 0
  return ((index % total) + total) % total
}

function handlePreviewKeydown(event: KeyboardEvent) {
  if (!previewImage.value) return
  const activeDialog = document.querySelector(`[data-preview-owner="${previewDialogId}"]`)
  if (!activeDialog?.isConnected) return

  if (event.key === 'Escape') {
    event.preventDefault()
    closePreview()
  } else if (event.key === 'ArrowRight') {
    event.preventDefault()
    showNextPreviewImage()
  } else if (event.key === 'ArrowLeft') {
    event.preventDefault()
    showPreviousPreviewImage()
  }
}

function downloadAll(item: HistoryItem) {
  item.images.forEach((image) => {
    const link = document.createElement('a')
    link.href = image.src
    link.download = downloadName(item, image.id)
    document.body.appendChild(link)
    link.click()
    link.remove()
  })
}

function persistHistory() {
  const storedItems: StoredHistoryItem[] = history.value.slice(0, MAX_STORED_HISTORY_ITEMS).map((item) => ({
    ...item,
    createdAt: item.createdAt.toISOString(),
  }))
  localStorage.setItem(HISTORY_STORAGE_KEY, JSON.stringify(storedItems))
}

function restoreHistory() {
  const rawHistory = localStorage.getItem(HISTORY_STORAGE_KEY)
  if (!rawHistory) return

  try {
    const storedItems = JSON.parse(rawHistory) as StoredHistoryItem[]
    if (!Array.isArray(storedItems)) return

    history.value = storedItems
      .filter(isStoredHistoryItem)
      .slice(0, MAX_STORED_HISTORY_ITEMS)
      .map((item) => ({
        ...item,
        createdAt: new Date(item.createdAt),
      }))
  } catch {
    localStorage.removeItem(HISTORY_STORAGE_KEY)
  }
}

function isStoredHistoryItem(item: unknown): item is StoredHistoryItem {
  if (!item || typeof item !== 'object') return false
  const candidate = item as Partial<StoredHistoryItem>
  return (
    typeof candidate.id === 'string' &&
    typeof candidate.prompt === 'string' &&
    typeof candidate.model === 'string' &&
    sizeOptions.includes(candidate.size as ImageSize) &&
    qualityOptions.includes(candidate.quality as ImageQuality) &&
    typeof candidate.elapsedSeconds === 'string' &&
    typeof candidate.createdAt === 'string' &&
    Array.isArray(candidate.images)
  )
}

function slugifyPrompt(value: string) {
  const slug = value
    .toLowerCase()
    .replace(/[^a-z0-9]+/g, '-')
    .replace(/^-+|-+$/g, '')
    .slice(0, 48)
  return slug || 'image'
}

function formatDownloadTimestamp(date: Date) {
  const year = date.getFullYear()
  const month = String(date.getMonth() + 1).padStart(2, '0')
  const day = String(date.getDate()).padStart(2, '0')
  const hour = String(date.getHours()).padStart(2, '0')
  const minute = String(date.getMinutes()).padStart(2, '0')
  const second = String(date.getSeconds()).padStart(2, '0')
  return `${year}${month}${day}-${hour}${minute}${second}`
}

async function sourceToImageFile(src: string, fallbackId: string) {
  const response = await fetch(src)
  const blob = await response.blob()
  const extension = imageExtensionFromType(blob.type)
  return new File([blob], `reference-${fallbackId}.${extension}`, {
    type: blob.type || 'image/png',
  })
}

function imageExtensionFromType(type: string) {
  if (type === 'image/jpeg') return 'jpg'
  if (type === 'image/webp') return 'webp'
  return 'png'
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
  const fallback = t('imageGeneration.generateFailed')
  const message = getRawErrorMessage(error)
  const normalizedMessage = message.toLowerCase()

  if (/(401|403|auth|unauthorized|forbidden|api key|token)/.test(normalizedMessage)) {
    return t('imageGeneration.errors.auth')
  }
  if (/(quota|balance|insufficient|billing|credit)/.test(normalizedMessage)) {
    return t('imageGeneration.errors.quota')
  }
  if (/(size|resolution|dimension|invalid_request_error)/.test(normalizedMessage)) {
    return t('imageGeneration.errors.size')
  }
  if (/(timeout|network|fetch|connection)/.test(normalizedMessage)) {
    return t('imageGeneration.errors.network')
  }
  if (/(disabled|group|image generation is not allowed|allow_image_generation)/.test(normalizedMessage)) {
    return t('imageGeneration.errors.groupDisabled')
  }

  return message || fallback
}

function getRawErrorMessage(error: unknown) {
  if (error instanceof Error && error.message) {
    return error.message
  }
  if (error && typeof error === 'object' && 'message' in error) {
    return String((error as { message?: unknown }).message || '')
  }
  return ''
}

onMounted(() => {
  restoreHistory()
  loadKeys()
  window.addEventListener('keydown', handlePreviewKeydown)
})
watch(selectedKeyId, () => {
  loadModelOptionsForSelectedKey()
})
onBeforeUnmount(() => {
  generationController?.abort()
  stopElapsedTimer()
  window.removeEventListener('keydown', handlePreviewKeydown)
})
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

.chat-thread {
  margin: 0 auto;
  display: flex;
  width: 100%;
  max-width: 980px;
  flex-direction: column;
  gap: 1.25rem;
  padding: 1.5rem 1rem;
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

.result-group {
  display: flex;
  flex-direction: column;
  gap: 0.875rem;
}

.chat-turn {
  display: flex;
  width: 100%;
  align-items: flex-end;
  gap: 0.75rem;
}

.assistant-turn {
  justify-content: flex-start;
  align-items: flex-start;
}

.user-turn {
  justify-content: flex-end;
}

.assistant-bubble,
.user-bubble {
  min-width: 0;
  max-width: min(100%, 780px);
  overflow: hidden;
  border: 1px solid rgba(229, 231, 235, 0.9);
  box-shadow: 0 18px 45px rgba(15, 23, 42, 0.07);
}

.assistant-bubble {
  width: fit-content;
  max-width: 100%;
  border-radius: 1.125rem 1.125rem 1.125rem 0.375rem;
  background: rgba(255, 255, 255, 0.9);
}

.user-bubble {
  border-radius: 1.125rem 1.125rem 0.375rem 1.125rem;
  background: rgb(238 242 255);
  padding: 0.875rem 1rem;
  color: rgb(31 41 55);
}

.user-bubble p:first-child {
  white-space: pre-wrap;
  overflow-wrap: anywhere;
  font-size: 0.925rem;
  font-weight: 700;
  line-height: 1.65;
}

.dark .assistant-bubble {
  border-color: rgba(55, 65, 81, 0.84);
  background: rgba(31, 41, 55, 0.86);
}

.dark .user-bubble {
  border-color: rgba(55, 65, 81, 0.9);
  background: rgba(49, 46, 129, 0.42);
  color: rgb(243 244 246);
}

.status-badge {
  display: inline-flex;
  height: 2rem;
  width: 2rem;
  flex: 0 0 2rem;
  align-items: center;
  justify-content: center;
  border-radius: 0.625rem;
  background: linear-gradient(135deg, rgb(79 70 229), rgb(99 102 241));
  color: white;
  font-size: 0.8125rem;
  font-weight: 800;
  box-shadow: 0 10px 22px rgba(79, 70, 229, 0.24);
}

.generation-card {
  width: min(100%, 390px);
  padding: 0.75rem 0.875rem 0.875rem;
}

.generation-canvas {
  display: flex;
  aspect-ratio: 16 / 9;
  align-items: center;
  justify-content: center;
  overflow: hidden;
  border-radius: 0.75rem;
  background:
    linear-gradient(90deg, rgba(238, 242, 255, 0.86), rgba(255, 255, 255, 0.94), rgba(238, 242, 255, 0.86));
  background-size: 220% 100%;
  animation: generation-shimmer 1.7s ease-in-out infinite;
}

.dark .generation-canvas {
  background:
    linear-gradient(90deg, rgba(30, 41, 59, 0.84), rgba(55, 65, 81, 0.88), rgba(30, 41, 59, 0.84));
  background-size: 220% 100%;
}

.generation-progress {
  margin-top: 0.75rem;
  height: 0.25rem;
  overflow: hidden;
  border-radius: 999px;
  background: rgb(229 231 235);
}

.generation-progress span {
  display: block;
  height: 100%;
  width: 68%;
  border-radius: inherit;
  background: linear-gradient(90deg, rgb(99 102 241), rgb(79 70 229));
  animation: generation-progress 1.3s ease-in-out infinite alternate;
}

.generation-card p {
  margin-top: 0.625rem;
  font-size: 0.75rem;
  font-weight: 700;
  color: rgb(107 114 128);
}

.dark .generation-card p {
  color: rgb(156 163 175);
}

.generation-card .generation-hint {
  margin-top: 0.25rem;
  font-weight: 500;
}

.assistant-toolbar {
  display: flex;
  flex-direction: column;
  gap: 0.75rem;
  border-bottom: 1px solid rgb(243 244 246);
  padding: 0.875rem 1rem;
}

.assistant-toolbar > span {
  font-size: 0.75rem;
  font-weight: 800;
  text-transform: uppercase;
  color: rgb(107 114 128);
}

.dark .assistant-toolbar {
  border-color: rgb(55 65 81);
}

.dark .assistant-toolbar > span {
  color: rgb(156 163 175);
}

.image-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(260px, 1fr));
  gap: 0.875rem;
  padding: 0.875rem;
}

.image-card {
  position: relative;
  overflow: hidden;
  border-radius: 0.875rem;
  border: 1px solid rgb(229 231 235);
  background: rgb(249 250 251);
  aspect-ratio: 16 / 9;
  max-width: 100%;
}

.dark .image-card {
  border-color: rgb(55 65 81);
  background: rgb(17 24 39);
}

.image-preview-trigger {
  display: block;
  height: 100%;
  width: 100%;
  cursor: zoom-in;
  background: transparent;
}

.image-preview-trigger img {
  transition: transform 0.22s ease, filter 0.22s ease;
}

.image-card:hover .image-preview-trigger img {
  transform: scale(1.025);
  filter: saturate(1.04);
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

.composer-shell > * {
  margin-left: auto;
  margin-right: auto;
  max-width: 980px;
}

.dark .composer-shell {
  border-color: rgba(55, 65, 81, 0.8);
  background: rgba(9, 13, 22, 0.9);
}

.control-pill {
  display: inline-flex;
  align-items: center;
  gap: 0.5rem;
  min-height: 2.25rem;
  border-radius: 0.7rem;
  border: 1px solid rgb(221 221 255);
  background: white;
  padding: 0.375rem 0.625rem;
  font-size: 0.75rem;
  color: rgb(75 85 99);
  box-shadow: 0 8px 20px rgba(15, 23, 42, 0.04);
  transition: border-color 0.16s ease, box-shadow 0.16s ease;
}

.control-pill:hover,
.control-pill:focus-within {
  border-color: rgb(99 102 241);
  box-shadow: 0 0 0 3px rgba(99, 102, 241, 0.08);
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
  font-weight: 700;
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

.reference-image-preview {
  position: relative;
  height: 3.25rem;
  width: 3.25rem;
  flex: 0 0 3.25rem;
  overflow: visible;
  border-radius: 0.625rem;
  border: 1px solid rgb(229 231 235);
  background: rgb(249 250 251);
}

.reference-image-preview img {
  height: 100%;
  width: 100%;
  border-radius: inherit;
  object-fit: cover;
}

.reference-remove {
  position: absolute;
  right: -0.5rem;
  top: -0.5rem;
  display: inline-flex;
  height: 1.25rem;
  width: 1.25rem;
  align-items: center;
  justify-content: center;
  border-radius: 999px;
  background: rgb(239 68 68);
  color: white;
  box-shadow: 0 6px 14px rgba(185, 28, 28, 0.28);
}

.dark .prompt-box {
  border-color: rgb(55 65 81);
  background: rgb(31 41 55);
}

.dark .reference-image-preview {
  border-color: rgb(55 65 81);
  background: rgb(17 24 39);
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

.image-preview-backdrop {
  position: fixed;
  inset: 0;
  z-index: 80;
  display: flex;
  align-items: center;
  justify-content: center;
  background: rgba(15, 23, 42, 0.72);
  padding: 1.25rem;
  backdrop-filter: blur(16px);
}

.image-preview-panel {
  position: relative;
  display: flex;
  max-height: min(92vh, 900px);
  width: min(96vw, 1120px);
  flex-direction: column;
  overflow: hidden;
  border-radius: 1rem;
  background: rgb(17 24 39);
  box-shadow: 0 30px 80px rgba(0, 0, 0, 0.38);
}

.image-preview-panel img {
  min-height: 0;
  max-height: calc(92vh - 4.5rem);
  width: 100%;
  object-fit: contain;
  background: rgb(3 7 18);
}

.preview-close {
  position: absolute;
  right: 0.75rem;
  top: 0.75rem;
  z-index: 1;
  display: inline-flex;
  height: 2.5rem;
  width: 2.5rem;
  align-items: center;
  justify-content: center;
  border-radius: 999px;
  background: rgba(255, 255, 255, 0.92);
  color: rgb(31 41 55);
  box-shadow: 0 10px 26px rgba(0, 0, 0, 0.24);
}

.preview-nav {
  position: absolute;
  top: 50%;
  z-index: 1;
  display: inline-flex;
  height: 2.75rem;
  width: 2.75rem;
  transform: translateY(-50%);
  align-items: center;
  justify-content: center;
  border-radius: 999px;
  background: rgba(255, 255, 255, 0.9);
  color: rgb(31 41 55);
  box-shadow: 0 14px 30px rgba(0, 0, 0, 0.28);
  transition: transform 0.16s ease, background 0.16s ease;
}

.preview-nav:hover {
  transform: translateY(-50%) scale(1.04);
  background: white;
}

.preview-nav.previous {
  left: 0.875rem;
}

.preview-nav.next {
  right: 0.875rem;
}

.preview-footer {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 1rem;
  padding: 0.875rem 1rem;
  color: white;
}

.preview-summary {
  min-width: 0;
  flex: 1;
}

.preview-summary p {
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  font-size: 0.875rem;
  color: rgb(229 231 235);
}

.preview-details {
  margin-top: 0.625rem;
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 0.375rem 0.75rem;
  font-size: 0.72rem;
}

.preview-details div {
  min-width: 0;
}

.preview-details dt {
  color: rgb(148 163 184);
}

.preview-details dd {
  margin-top: 0.125rem;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  font-weight: 700;
  color: rgb(243 244 246);
}

.preview-download {
  display: inline-flex;
  flex: 0 0 auto;
  align-items: center;
  gap: 0.375rem;
  border-radius: 0.625rem;
  background: white;
  padding: 0.5rem 0.75rem;
  font-size: 0.8125rem;
  font-weight: 700;
  color: rgb(55 65 81);
}

@keyframes generation-shimmer {
  0% {
    background-position: 0% 50%;
  }

  100% {
    background-position: 100% 50%;
  }
}

@keyframes generation-progress {
  from {
    width: 38%;
  }

  to {
    width: 82%;
  }
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

  .chat-thread {
    padding-left: 1.5rem;
    padding-right: 1.5rem;
  }

  .assistant-toolbar {
    flex-direction: row;
    align-items: center;
    justify-content: space-between;
  }

  .image-grid {
    gap: 1rem;
    padding: 1rem;
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

  .chat-thread {
    padding-left: 2rem;
    padding-right: 2rem;
  }

  .image-grid {
    grid-template-columns: repeat(auto-fill, minmax(260px, 1fr));
  }
}
</style>

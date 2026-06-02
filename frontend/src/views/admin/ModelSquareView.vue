<template>
  <AppLayout>
    <div class="mx-auto max-w-7xl space-y-5">
      <div class="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
        <div>
          <h1 class="text-2xl font-semibold text-gray-900 dark:text-white">模型广场</h1>
        </div>
        <button type="button" class="btn btn-secondary btn-sm" :disabled="loading" @click="loadModels">
          <Icon name="refresh" size="sm" :class="loading ? 'animate-spin' : ''" />
          刷新
        </button>
      </div>

      <div class="grid gap-3 md:grid-cols-[1fr_auto_auto_auto]">
        <div class="relative">
          <Icon name="search" size="sm" class="pointer-events-none absolute left-3 top-1/2 -translate-y-1/2 text-gray-400" />
          <input v-model="search" type="search" class="input pl-9" placeholder="搜索模型..." />
        </div>
        <select v-model="groupId" class="input min-w-40">
          <option value="">全部分组</option>
          <option v-for="group in groups" :key="group.id" :value="String(group.id)">{{ group.name }}</option>
        </select>
        <select v-model="provider" class="input min-w-40">
          <option value="">全部提供商</option>
          <option v-for="item in providers" :key="item" :value="item">{{ item }}</option>
        </select>
        <select v-model="mode" class="input min-w-36">
          <option value="">全部类型</option>
          <option v-for="item in modes" :key="item" :value="item">{{ modeLabel(item) }}</option>
        </select>
      </div>

      <div class="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
        <div class="inline-grid w-fit grid-cols-2 gap-1 rounded-lg border border-gray-200 bg-gray-100 p-1 dark:border-dark-700 dark:bg-dark-800">
          <button
            type="button"
            class="view-toggle-btn"
            :class="{ active: viewMode === 'grid' }"
            aria-label="网格视图"
            @click="viewMode = 'grid'"
          >
            <Icon name="grid" size="sm" />
          </button>
          <button
            type="button"
            class="view-toggle-btn"
            :class="{ active: viewMode === 'list' }"
            aria-label="列表视图"
            @click="viewMode = 'list'"
          >
            <Icon name="menu" size="sm" />
          </button>
        </div>
        <span class="text-sm text-gray-500 dark:text-gray-400">{{ availableCount }} 个可用模型</span>
      </div>

      <div v-if="loading" class="grid min-h-64 place-items-center rounded-lg border border-gray-200 bg-white dark:border-dark-700 dark:bg-dark-800">
        <div class="flex items-center gap-2 text-sm text-gray-500 dark:text-gray-400">
          <span class="h-5 w-5 animate-spin rounded-full border-2 border-primary-500 border-t-transparent"></span>
          正在加载模型广场数据...
        </div>
      </div>

      <div v-else-if="error" class="rounded-lg border border-red-200 bg-red-50 p-5 text-sm text-red-700 dark:border-red-900/60 dark:bg-red-900/20 dark:text-red-300">
        {{ error }}
      </div>

      <template v-else>
        <div class="flex items-center justify-between text-sm text-gray-500 dark:text-gray-400">
          <span>{{ filteredModels.length }} 个模型，{{ availableCount }} 个可用</span>
          <span v-if="updatedAt">更新于 {{ updatedAt }}</span>
        </div>

        <div v-if="filteredModels.length === 0" class="grid min-h-56 place-items-center rounded-lg border border-dashed border-gray-300 bg-white text-sm text-gray-500 dark:border-dark-600 dark:bg-dark-800 dark:text-gray-400">
          暂无匹配模型
        </div>

        <div v-else-if="viewMode === 'grid'" class="grid gap-4 lg:grid-cols-2 xl:grid-cols-3">
          <article
            v-for="(model, index) in filteredModels"
            :key="model.id || index"
            class="model-card cursor-pointer"
            :class="{ featured: index === 0 }"
            role="button"
            tabindex="0"
            title="点击复制模型 ID"
            @click="copyModelId(model)"
            @keydown.enter.prevent="copyModelId(model)"
          >
            <div class="flex items-start justify-between gap-3">
              <span class="inline-flex items-center gap-2 text-xs font-semibold lowercase text-gray-500 dark:text-gray-400">
                <span class="h-2 w-2 rounded-full bg-teal-500"></span>
                {{ model.provider || 'unknown' }}
              </span>
              <span
                class="rounded-md px-2 py-1 text-xs font-semibold"
                :class="model.available ? 'bg-emerald-100 text-emerald-700 dark:bg-emerald-900/40 dark:text-emerald-300' : 'bg-gray-100 text-gray-500 dark:bg-dark-700 dark:text-gray-400'"
              >
                {{ copiedModelId === model.id ? '已复制' : (model.available ? '可用' : '不可用') }}
              </span>
            </div>

            <h2 class="mt-4 min-h-10 break-words text-base font-bold text-gray-950 dark:text-white">
              {{ model.id || '未命名模型' }}
            </h2>

            <div class="mt-4 grid grid-cols-2 gap-3">
              <PriceBox label="输入" :value="modelDisplayPrice(model, 'input_price')" />
              <PriceBox label="输出" :value="modelDisplayPrice(model, 'output_price')" />
              <PriceBox label="Cache Read" :value="modelDisplayPrice(model, 'cache_read_price')" tone="blue" />
              <PriceBox label="Cache Write" :value="modelDisplayPrice(model, 'cache_create_price')" tone="purple" />
            </div>

            <div class="mt-auto flex items-center justify-between gap-3 border-t border-gray-100 pt-4 dark:border-dark-700">
              <span class="rounded bg-blue-50 px-2 py-1 text-xs font-medium text-blue-700 dark:bg-blue-900/30 dark:text-blue-300">
                {{ modeLabel(model.mode) }}
              </span>
              <div class="flex min-w-0 flex-wrap justify-end gap-1.5">
                <span
                  v-for="group in modelGroups(model).slice(0, 3)"
                  :key="group.id"
                  class="rounded bg-gray-100 px-2 py-1 text-xs text-gray-600 dark:bg-dark-700 dark:text-gray-300"
                >
                  {{ group.name }} <b class="text-orange-500">{{ formatRate(group.rate_multiplier) }}x</b>
                </span>
                <button
                  v-if="modelGroups(model).length > 3"
                  type="button"
                  class="rounded bg-teal-50 px-2 py-1 text-xs font-semibold text-teal-700 transition hover:bg-teal-100 focus:outline-none focus:ring-2 focus:ring-teal-400 dark:bg-teal-900/30 dark:text-teal-300 dark:hover:bg-teal-900/50"
                  @click.stop="openGroupModal(model)"
                >
                  +{{ modelGroups(model).length - 3 }} &gt;
                </button>
              </div>
            </div>
          </article>
        </div>

        <div v-else class="overflow-hidden rounded-lg border border-gray-200 bg-white dark:border-dark-700 dark:bg-dark-800">
          <div class="overflow-x-auto">
            <table class="min-w-full text-left text-sm">
              <thead class="bg-gray-50 text-xs font-semibold text-gray-500 dark:bg-dark-700/60 dark:text-gray-400">
                <tr>
                  <th class="whitespace-nowrap px-4 py-3">状态</th>
                  <th class="whitespace-nowrap px-4 py-3">提供商</th>
                  <th class="min-w-48 px-4 py-3">Model ID</th>
                  <th class="whitespace-nowrap px-4 py-3">输入 $/M</th>
                  <th class="whitespace-nowrap px-4 py-3">输出 $/M</th>
                  <th class="whitespace-nowrap px-4 py-3">缓存读取 $/M</th>
                  <th class="whitespace-nowrap px-4 py-3">缓存写入 $/M</th>
                  <th class="whitespace-nowrap px-4 py-3">类型</th>
                  <th class="whitespace-nowrap px-4 py-3">按次计费</th>
                  <th class="min-w-72 px-4 py-3">分组</th>
                </tr>
              </thead>
              <tbody class="divide-y divide-gray-100 dark:divide-dark-700">
                <tr
                  v-for="(model, index) in filteredModels"
                  :key="model.id || index"
                  class="cursor-pointer transition hover:bg-teal-50/50 dark:hover:bg-dark-700/60"
                  title="点击复制模型 ID"
                  @click="copyModelId(model)"
                >
                  <td class="whitespace-nowrap px-4 py-4">
                    <span class="inline-flex items-center gap-2 text-xs font-semibold" :class="model.available ? 'text-emerald-600 dark:text-emerald-300' : 'text-gray-500 dark:text-gray-400'">
                      <span class="h-2 w-2 rounded-full" :class="model.available ? 'bg-emerald-300' : 'bg-gray-300 dark:bg-gray-600'"></span>
                      {{ copiedModelId === model.id ? '已复制' : (model.available ? '可用' : '不可用') }}
                    </span>
                  </td>
                  <td class="whitespace-nowrap px-4 py-4 text-gray-600 dark:text-gray-300">
                    <span class="inline-flex items-center gap-2">
                      <span class="h-2 w-2 rounded-full bg-slate-400"></span>
                      {{ model.provider || 'unknown' }}
                    </span>
                  </td>
                  <td class="max-w-64 px-4 py-4 font-medium text-gray-950 dark:text-white">
                    <span class="break-words">{{ model.id || '未命名模型' }}</span>
                  </td>
                  <td class="whitespace-nowrap px-4 py-4 font-mono text-gray-900 dark:text-gray-100">{{ formatPrice(modelDisplayPrice(model, 'input_price')) }}</td>
                  <td class="whitespace-nowrap px-4 py-4 font-mono text-gray-900 dark:text-gray-100">{{ formatPrice(modelDisplayPrice(model, 'output_price')) }}</td>
                  <td class="whitespace-nowrap px-4 py-4 font-mono text-gray-900 dark:text-gray-100">{{ formatPrice(modelDisplayPrice(model, 'cache_read_price')) }}</td>
                  <td class="whitespace-nowrap px-4 py-4 font-mono text-gray-900 dark:text-gray-100">{{ formatPrice(modelDisplayPrice(model, 'cache_create_price')) }}</td>
                  <td class="whitespace-nowrap px-4 py-4">
                    <span class="rounded bg-blue-100 px-2 py-1 text-xs font-medium text-blue-700 dark:bg-blue-900/40 dark:text-blue-300">
                      {{ modeLabel(model.mode) }}
                    </span>
                  </td>
                  <td class="whitespace-nowrap px-4 py-4 text-gray-400">—</td>
                  <td class="px-4 py-4">
                    <div class="flex min-w-72 flex-wrap gap-1.5">
                      <span
                        v-for="group in modelGroups(model)"
                        :key="group.id"
                        class="rounded bg-gray-100 px-2 py-1 text-xs text-gray-600 dark:bg-dark-700 dark:text-gray-300"
                      >
                        {{ group.name }} <b class="text-orange-500">{{ formatRate(group.rate_multiplier) }}x</b>
                      </span>
                    </div>
                  </td>
                </tr>
              </tbody>
            </table>
          </div>
        </div>
      </template>
    </div>

    <div
      v-if="groupModalModel"
      class="fixed inset-0 z-50 grid place-items-center bg-slate-950/45 px-4 py-6 backdrop-blur-sm"
      role="dialog"
      aria-modal="true"
      @click.self="closeGroupModal"
    >
      <div class="w-full max-w-lg overflow-hidden rounded-2xl border border-gray-200 bg-white shadow-2xl dark:border-dark-700 dark:bg-dark-800">
        <div class="flex items-start justify-between gap-4 border-b border-gray-100 px-5 py-4 dark:border-dark-700">
          <div class="min-w-0">
            <h2 class="break-words text-base font-semibold text-gray-950 dark:text-white">
              {{ groupModalModel.id || '未命名模型' }} 的可用分组
            </h2>
            <p class="mt-1 text-sm text-gray-500 dark:text-gray-400">
              共 {{ groupModalGroups.length }} 个分组
            </p>
          </div>
          <button
            type="button"
            class="rounded-md p-1.5 text-gray-400 transition hover:bg-gray-100 hover:text-gray-700 focus:outline-none focus:ring-2 focus:ring-primary-400 dark:hover:bg-dark-700 dark:hover:text-gray-200"
            aria-label="关闭"
            @click="closeGroupModal"
          >
            <Icon name="x" size="sm" />
          </button>
        </div>

        <div class="max-h-[56vh] space-y-2 overflow-y-auto px-5 py-4">
          <div
            v-for="group in groupModalGroups"
            :key="group.id"
            class="flex items-center justify-between gap-3 rounded-lg border border-gray-100 bg-gray-50 px-3 py-3 dark:border-dark-700 dark:bg-dark-700/50"
          >
            <div class="flex min-w-0 items-center gap-3">
              <span class="grid h-6 w-6 shrink-0 place-items-center rounded-md bg-teal-100 text-[10px] font-bold uppercase text-teal-700 dark:bg-teal-900/50 dark:text-teal-200">
                {{ groupInitials(group.name) }}
              </span>
              <span class="min-w-0 break-words text-sm font-medium text-gray-950 dark:text-white">
                {{ group.name }}
              </span>
            </div>
            <div class="shrink-0 text-xs text-gray-500 dark:text-gray-400">
              倍率
              <span class="ml-2 rounded bg-amber-100 px-2 py-1 font-semibold text-orange-600 dark:bg-amber-900/40 dark:text-amber-300">
                {{ formatRate(group.rate_multiplier) }}x
              </span>
            </div>
          </div>
        </div>

        <div class="flex justify-end border-t border-gray-100 px-5 py-3 dark:border-dark-700">
          <button type="button" class="btn btn-secondary btn-sm" @click="closeGroupModal">关闭</button>
        </div>
      </div>
    </div>
  </AppLayout>
</template>

<script setup lang="ts">
import { computed, defineComponent, h, onMounted, ref } from 'vue'
import { apiClient } from '@/api/client'
import AppLayout from '@/components/layout/AppLayout.vue'
import Icon from '@/components/icons/Icon.vue'

interface ModelSquareGroup {
  id: number | string
  name: string
  rate_multiplier?: number
}

interface ModelSquareModel {
  id: string
  provider?: string
  available?: boolean
  mode?: string
  input_price?: number | string
  output_price?: number | string
  cache_read_price?: number | string
  cache_create_price?: number | string
  group_ids?: Array<number | string>
}

interface ModelSquareResponse {
  groups?: ModelSquareGroup[]
  models?: ModelSquareModel[]
  updated_at?: string
}

const loading = ref(false)
const error = ref('')
const search = ref('')
const provider = ref('')
const mode = ref('')
const groupId = ref('')
const viewMode = ref<'grid' | 'list'>('grid')
const models = ref<ModelSquareModel[]>([])
const groups = ref<ModelSquareGroup[]>([])
const updatedAt = ref('')
const groupModalModel = ref<ModelSquareModel | null>(null)
const copiedModelId = ref('')
let copiedTimer: number | undefined

const groupById = computed(() => new Map(groups.value.map((group) => [String(group.id), group])))
const providers = computed(() => unique(models.value.map((model) => model.provider).filter(Boolean) as string[]))
const modes = computed(() => unique(models.value.map((model) => model.mode || 'chat')))
const availableCount = computed(() => models.value.filter((model) => model.available).length)
const groupModalGroups = computed(() => groupModalModel.value ? modelGroups(groupModalModel.value) : [])

const filteredModels = computed(() => {
  const keyword = search.value.trim().toLowerCase()
  return models.value.filter((model) => {
    if (keyword && !String(model.id || '').toLowerCase().includes(keyword)) return false
    if (groupId.value && !(model.group_ids || []).some((id) => String(id) === groupId.value)) return false
    if (provider.value && model.provider !== provider.value) return false
    if (mode.value && (model.mode || 'chat') !== mode.value) return false
    return true
  })
})

async function loadModels() {
  loading.value = true
  error.value = ''
  try {
    const data = await apiClient.get<ModelSquareResponse>('/model-square')
    models.value = Array.isArray(data.data.models) ? data.data.models : []
    groups.value = Array.isArray(data.data.groups) ? data.data.groups : []
    updatedAt.value = data.data.updated_at || ''
  } catch (err) {
    const e = err as { message?: string }
    error.value = e.message || '模型广场数据加载失败'
  } finally {
    loading.value = false
  }
}

function modelGroups(model: ModelSquareModel): ModelSquareGroup[] {
  return (model.group_ids || [])
    .map((id) => groupById.value.get(String(id)))
    .filter(Boolean) as ModelSquareGroup[]
}

function primaryGroupRate(model: ModelSquareModel) {
  const firstGroupId = model.group_ids?.[0]
  if (firstGroupId === undefined || firstGroupId === null) return 1
  const rate = Number(groupById.value.get(String(firstGroupId))?.rate_multiplier ?? 1)
  return Number.isFinite(rate) ? rate : 1
}

function modelDisplayPrice(model: ModelSquareModel, field: 'input_price' | 'output_price' | 'cache_read_price' | 'cache_create_price') {
  const price = Number(model[field] ?? 0)
  if (!Number.isFinite(price)) return 0
  return price * primaryGroupRate(model)
}

function openGroupModal(model: ModelSquareModel) {
  groupModalModel.value = model
}

function closeGroupModal() {
  groupModalModel.value = null
}

async function copyModelId(model: ModelSquareModel) {
  if (!model.id) return
  try {
    await navigator.clipboard.writeText(model.id)
  } catch {
    const textarea = document.createElement('textarea')
    textarea.value = model.id
    textarea.style.position = 'fixed'
    textarea.style.opacity = '0'
    document.body.appendChild(textarea)
    textarea.select()
    document.execCommand('copy')
    textarea.remove()
  }

  copiedModelId.value = model.id
  if (copiedTimer) window.clearTimeout(copiedTimer)
  copiedTimer = window.setTimeout(() => {
    copiedModelId.value = ''
  }, 1500)
}

function groupInitials(name: string) {
  const compact = String(name || '').trim().replace(/\s+/g, '')
  if (!compact) return '--'
  return compact.slice(0, 2).toUpperCase()
}

function modeLabel(value?: string) {
  if (value === 'image_generation') return 'Image'
  if (value === 'embedding') return 'Embedding'
  if (value === 'responses') return 'responses'
  return value || 'Chat'
}

function formatRate(value?: number) {
  const n = Number(value ?? 0)
  return Number.isFinite(n) ? n.toFixed(3).replace(/0+$/, '').replace(/\.$/, '') : '0'
}

function formatPrice(value?: number | string) {
  const n = Number(value ?? 0)
  if (!Number.isFinite(n)) return '$0'
  return `$${n.toFixed(n >= 10 ? 2 : 3).replace(/0+$/, '').replace(/\.$/, '')}`
}

function unique(values: string[]) {
  return Array.from(new Set(values)).sort((a, b) => a.localeCompare(b))
}

const PriceBox = defineComponent({
  props: {
    label: { type: String, required: true },
    value: { type: [Number, String], default: 0 },
    tone: { type: String, default: '' }
  },
  setup(props) {
    return () =>
      h('div', {
        class: [
          'rounded-lg border p-3',
          props.tone === 'blue'
            ? 'border-blue-100 bg-blue-50 text-blue-700 dark:border-blue-900/40 dark:bg-blue-900/20 dark:text-blue-300'
            : props.tone === 'purple'
              ? 'border-purple-100 bg-purple-50 text-purple-700 dark:border-purple-900/40 dark:bg-purple-900/20 dark:text-purple-300'
              : 'border-gray-100 bg-gray-50 text-gray-900 dark:border-dark-700 dark:bg-dark-700/60 dark:text-gray-100'
        ]
      }, [
        h('div', { class: 'text-xs text-gray-500 dark:text-gray-400' }, props.label),
        h('div', { class: 'mt-1 font-mono text-sm font-bold' }, formatPrice(props.value)),
        h('div', { class: 'mt-1 text-[10px] text-gray-400' }, '$/M tokens')
      ])
  }
})

onMounted(loadModels)
</script>

<style scoped>
.model-card {
  min-height: 280px;
  display: flex;
  flex-direction: column;
  border: 1px solid rgb(226 232 240);
  border-radius: 10px;
  background: rgba(255, 255, 255, 0.96);
  padding: 1.25rem;
  box-shadow: 0 12px 30px rgba(15, 23, 42, 0.06);
}

.dark .model-card {
  border-color: rgb(55 65 81);
  background: rgb(31 41 55 / 0.76);
}

.model-card.featured {
  border-color: rgb(45 212 191 / 0.75);
  box-shadow: 0 0 0 1px rgb(45 212 191 / 0.18), 0 16px 34px rgb(13 148 136 / 0.12);
}

.view-toggle-btn {
  display: inline-grid;
  height: 2rem;
  width: 2rem;
  place-items: center;
  border-radius: 0.375rem;
  color: rgb(100 116 139);
  transition: background-color 150ms ease, color 150ms ease, box-shadow 150ms ease;
}

.view-toggle-btn:hover {
  background: rgb(255 255 255 / 0.72);
  color: rgb(15 23 42);
}

.view-toggle-btn.active {
  background: white;
  color: rgb(20 184 166);
  box-shadow: 0 1px 3px rgb(15 23 42 / 0.1);
}

.dark .view-toggle-btn:hover {
  background: rgb(55 65 81 / 0.72);
  color: rgb(229 231 235);
}

.dark .view-toggle-btn.active {
  background: rgb(55 65 81);
  color: rgb(94 234 212);
}
</style>

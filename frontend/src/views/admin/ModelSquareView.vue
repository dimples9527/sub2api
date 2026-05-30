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

        <div v-else class="grid gap-4 lg:grid-cols-2 xl:grid-cols-3">
          <article
            v-for="(model, index) in filteredModels"
            :key="model.id || index"
            class="model-card"
            :class="{ featured: index === 0 }"
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
                {{ model.available ? '可用' : '不可用' }}
              </span>
            </div>

            <h2 class="mt-4 min-h-10 break-words text-base font-bold text-gray-950 dark:text-white">
              {{ model.id || '未命名模型' }}
            </h2>

            <div class="mt-4 grid grid-cols-2 gap-3">
              <PriceBox label="输入" :value="model.input_price" />
              <PriceBox label="输出" :value="model.output_price" />
              <PriceBox label="Cache Read" :value="model.cache_read_price" tone="blue" />
              <PriceBox label="Cache Write" :value="model.cache_create_price" tone="purple" />
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
                <span v-if="modelGroups(model).length > 3" class="rounded bg-teal-50 px-2 py-1 text-xs font-semibold text-teal-700 dark:bg-teal-900/30 dark:text-teal-300">
                  +{{ modelGroups(model).length - 3 }}
                </span>
              </div>
            </div>
          </article>
        </div>
      </template>
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
const models = ref<ModelSquareModel[]>([])
const groups = ref<ModelSquareGroup[]>([])
const updatedAt = ref('')

const groupById = computed(() => new Map(groups.value.map((group) => [String(group.id), group])))
const providers = computed(() => unique(models.value.map((model) => model.provider).filter(Boolean) as string[]))
const modes = computed(() => unique(models.value.map((model) => model.mode || 'chat')))
const availableCount = computed(() => models.value.filter((model) => model.available).length)

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
    const data = await apiClient.get<ModelSquareResponse>('/admin/model-square')
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
</style>

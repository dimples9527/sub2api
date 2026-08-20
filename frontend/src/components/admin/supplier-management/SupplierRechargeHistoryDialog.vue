<template>
  <BaseDialog
    :show="show"
    :title="dialogTitle"
    width="extra-wide"
    :close-on-click-outside="false"
    @close="emit('close')"
  >
    <div class="recharge-history-dialog" data-test="supplier-recharge-history-dialog">
      <div class="recharge-toolbar">
        <div class="recharge-filters">
          <Select
            v-if="!providerId"
            v-model="selectedProviderId"
            class="w-full sm:w-52"
            :options="providerOptions"
            aria-label="筛选供应商"
            placeholder="所有供应商"
            clearable
          />
          <Input v-model="startDate" type="date" class="w-full sm:w-40" aria-label="开始日期" />
          <Input v-model="endDate" type="date" class="w-full sm:w-40" aria-label="结束日期" />
          <button class="sp-button small" type="button" :disabled="loading" @click="loadRecharges">查询</button>
          <button class="sp-button small primary" type="button" :disabled="syncing" @click="syncHistory">
            {{ syncing ? '同步中…' : '同步历史充值' }}
          </button>
        </div>
        <div class="recharge-summary" data-test="supplier-recharge-summary">
          <span>记录 {{ total }}</span>
          <strong>累计 ¥ {{ formatAmount(totalAmount) }}</strong>
        </div>
      </div>

      <div v-if="error" class="recharge-error" data-test="supplier-recharge-error">{{ error }}</div>
      <DataTable
        :columns="columns"
        :data="items"
        :loading="loading"
        row-key="id"
        data-test="supplier-recharge-table"
      >
        <template #cell-provider_name="{ row }">
          <div class="recharge-provider-cell">
            <strong>{{ row.provider_name }}</strong>
            <small>{{ row.provider_type }}</small>
          </div>
        </template>
        <template #cell-amount="{ row }">
          <strong class="recharge-amount">¥ {{ formatAmount(row.amount) }}</strong>
        </template>
        <template #cell-occurred_at="{ row }">
          {{ formatDateTime(row.occurred_at) }}
        </template>
        <template #cell-status="{ row }">
          <span class="recharge-status" :class="statusTone(row.status)">{{ row.status || '—' }}</span>
        </template>
        <template #cell-description="{ row }">
          <span class="recharge-description" :title="row.description">{{ row.description || '—' }}</span>
        </template>
        <template #empty>
          <div class="recharge-empty" data-test="supplier-recharge-empty">暂无充值记录</div>
        </template>
      </DataTable>
      <Pagination
        v-if="total > 0"
        :page="page"
        :total="total"
        :page-size="pageSize"
        :show-page-size-selector="false"
        @update:page="changePage"
      />
    </div>
  </BaseDialog>
</template>

<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import BaseDialog from '@/components/common/BaseDialog.vue'
import DataTable from '@/components/common/DataTable.vue'
import Input from '@/components/common/Input.vue'
import Pagination from '@/components/common/Pagination.vue'
import Select, { type SelectOption } from '@/components/common/Select.vue'
import { useAppStore } from '@/stores/app'
import {
  listSupplierProviderRecharges,
  syncSupplierProviderRecharges,
  type SupplierProviderRecharge,
} from '@/api/admin/supplierProviderRecharges'
import type { Column } from '@/components/common/types'

interface ProviderOption {
  id: number
  name: string
}

interface Props {
  show: boolean
  providerId?: number
  providerName?: string
  providers?: ProviderOption[]
}

const props = withDefaults(defineProps<Props>(), {
  providerName: '',
  providers: () => [],
})

const emit = defineEmits<{ (event: 'close'): void }>()
const appStore = useAppStore()
const items = ref<SupplierProviderRecharge[]>([])
const loading = ref(false)
const syncing = ref(false)
const error = ref('')
const total = ref(0)
const totalAmount = ref(0)
const page = ref(1)
const pageSize = ref(20)
const selectedProviderId = ref<number | null>(null)
const startDate = ref('')
const endDate = ref('')

const dialogTitle = computed(() => props.providerId ? `${props.providerName || '供应商'}充值记录` : '所有供应商充值记录')
const providerOptions = computed<SelectOption[]>(() => [
  { value: null, label: '所有供应商' },
  ...props.providers.map(provider => ({ value: provider.id, label: provider.name })),
])
const columns: Column[] = [
  { key: 'provider_name', label: '供应商', class: 'min-w-[150px]' },
  { key: 'amount', label: '充值金额', class: 'min-w-[120px]' },
  { key: 'recharge_type', label: '类型' },
  { key: 'status', label: '状态' },
  { key: 'occurred_at', label: '发生时间', class: 'min-w-[170px]' },
  { key: 'description', label: '说明', class: 'min-w-[260px]' },
]

function resetState() {
  items.value = []
  error.value = ''
  total.value = 0
  totalAmount.value = 0
  page.value = 1
  selectedProviderId.value = props.providerId ?? null
}

async function loadRecharges() {
  if (!props.show) return
  loading.value = true
  error.value = ''
  try {
    const result = await listSupplierProviderRecharges({
      provider_id: props.providerId ?? selectedProviderId.value ?? undefined,
      start_date: startDate.value || undefined,
      end_date: endDate.value || undefined,
      page: page.value,
      page_size: pageSize.value,
    })
    items.value = result.items || []
    total.value = Number(result.total || 0)
    totalAmount.value = Number(result.total_amount || 0)
  } catch (err) {
    error.value = err instanceof Error ? err.message : '加载充值记录失败'
  } finally {
    loading.value = false
  }
}

async function syncHistory() {
  syncing.value = true
  try {
    const result = await syncSupplierProviderRecharges(props.providerId ?? selectedProviderId.value ?? undefined, true)
    const failed = Number(result.failed_count || 0)
    if (failed > 0) {
      const failedItems = (result.items || [])
        .filter(item => item.status === 'failed')
        .map(item => item.provider_name ? `${item.provider_name}${item.message ? `\uFF08${item.message}\uFF09` : ''}` : item.message || '')
        .filter(Boolean)
      const detail = failedItems.length > 0 ? `\uFF1A${failedItems.join('\u3001')}` : ''
      appStore.showError(`\u5145\u503C\u8BB0\u5F55\u540C\u6B65\u5B8C\u6210\uFF0C\u4F46\u6709 ${failed} \u4E2A\u4F9B\u5E94\u5546\u5931\u8D25${detail}`)
    } else {
      appStore.showSuccess('\u5145\u503C\u5386\u53F2\u540C\u6B65\u5B8C\u6210')
    }
    await loadRecharges()
  } catch (err) {
    appStore.showError(err instanceof Error ? err.message : '同步充值记录失败')
  } finally {
    syncing.value = false
  }
}

function changePage(nextPage: number) {
  page.value = nextPage
  void loadRecharges()
}

function formatAmount(value: number) {
  return Number(value || 0).toLocaleString('zh-CN', { minimumFractionDigits: 2, maximumFractionDigits: 2 })
}

function formatDateTime(value?: string) {
  if (!value) return '—'
  const date = new Date(value)
  return Number.isNaN(date.getTime()) ? value : date.toLocaleString('zh-CN', { hour12: false })
}

function statusTone(status: string) {
  return status === 'used' || status === 'success' ? 'good' : status === 'failed' ? 'bad' : 'neutral'
}

watch(
  () => [props.show, props.providerId] as const,
  ([show]) => {
    if (show) {
      resetState()
      void loadRecharges()
    }
  },
  { immediate: true },
)
</script>

<style scoped>
.recharge-history-dialog {
  --sp-panel: #ffffff;
  --sp-panel-2: #f8fafc;
  --sp-line: #e2e8f0;
  --sp-text: #0f172a;
  --sp-muted: #64748b;
  --sp-green: #16a34a;
  --sp-blue: #2563eb;
  --sp-cyan: #0891b2;
  --sp-violet: #7c3aed;
  --sp-red: #dc2626;
  color: var(--sp-text);
}

.recharge-toolbar {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 1rem;
  margin-bottom: 1rem;
  padding: 0.75rem;
  border: 1px solid var(--sp-line);
  border-radius: 0.8rem;
  background: var(--sp-panel-2);
}

.recharge-filters {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 0.5rem;
}

.recharge-summary {
  display: flex;
  flex-shrink: 0;
  flex-direction: column;
  align-items: flex-end;
  gap: 0.15rem;
  color: var(--sp-muted);
  font-size: 0.75rem;
}

.recharge-summary strong,
.recharge-amount {
  color: var(--sp-green);
  font-weight: 800;
}

.recharge-error {
  margin-bottom: 0.75rem;
  padding: 0.65rem 0.75rem;
  border: 1px solid color-mix(in srgb, var(--sp-red) 28%, var(--sp-line));
  border-radius: 0.6rem;
  background: color-mix(in srgb, var(--sp-red) 7%, var(--sp-panel));
  color: var(--sp-red);
  font-size: 0.8rem;
}

.recharge-provider-cell {
  display: flex;
  flex-direction: column;
  gap: 0.15rem;
}

.recharge-provider-cell small,
.recharge-description {
  overflow: hidden;
  color: var(--sp-muted);
  font-size: 0.75rem;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.recharge-status {
  display: inline-flex;
  border-radius: 999px;
  padding: 0.2rem 0.5rem;
  font-size: 0.7rem;
  font-weight: 700;
}

.recharge-status.good { background: #dcfce7; color: #15803d; }
.recharge-status.bad { background: #fee2e2; color: #b91c1c; }
.recharge-status.neutral { background: #e2e8f0; color: #475569; }

.recharge-empty {
  padding: 2rem 0;
  color: var(--sp-muted);
  text-align: center;
}

@media (max-width: 760px) {
  .recharge-toolbar,
  .recharge-summary {
    align-items: stretch;
  }

  .recharge-toolbar {
    flex-direction: column;
  }

  .recharge-summary {
    align-items: flex-start;
  }
}
</style>
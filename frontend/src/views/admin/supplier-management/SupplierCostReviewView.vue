<template>
  <SupplierModuleLayout>
    <header class="sp-page-head cost-review-head">
      <div>
        <div class="sp-eyebrow">供应商管理 / 成本核对</div>
        <h1>上游成本核对</h1>
        <p class="sp-subtitle">对比接口成本与本地计算成本，逐条确认当前业务生效成本。</p>
      </div>
      <div class="sp-controls">
        <span v-if="lastLoadedAt" class="sp-data-note">更新于 {{ formatDateTime(lastLoadedAt) }}</span>
        <button class="sp-button primary" type="button" :disabled="loading" @click="loadReviews">{{ loading ? '刷新中…' : '刷新数据' }}</button>
      </div>
    </header>

    <section class="sp-panel cost-review-filters" aria-label="成本核对筛选">
      <div class="cost-review-filter-grid">
        <Input v-model="filters.keyword" aria-label="供应商名称" placeholder="搜索供应商名称" @keyup.enter="applyFilters" />
        <Select v-model="filters.providerId" :options="providerOptions" clearable aria-label="供应商" placeholder="全部供应商" @change="applyFilters" />
        <DateRangePicker v-model:start-date="filters.startDate" v-model:end-date="filters.endDate" @change="applyFilters" />
        <Select v-model="filters.status" :options="statusOptions" clearable aria-label="核对状态" placeholder="全部状态" @change="applyFilters" />
        <button class="sp-button ghost" type="button" :disabled="loading" @click="resetFilters">重置筛选</button>
      </div>
    </section>

    <section class="sp-metric-grid cost-review-metrics" aria-label="成本核对摘要">
      <article class="sp-metric-card sp-blue"><div class="sp-metric-label">当前记录</div><div class="sp-metric-value">{{ total }}</div><div class="sp-metric-foot">按当前筛选条件统计</div></article>
      <article class="sp-metric-card sp-amber"><div class="sp-metric-label">待审批</div><div class="sp-metric-value">{{ statusCounts.pending_review }}</div><div class="sp-metric-foot">首次同步默认采用计算成本</div></article>
      <article class="sp-metric-card sp-green"><div class="sp-metric-label">已审批</div><div class="sp-metric-value">{{ statusCounts.approved }}</div><div class="sp-metric-foot">人工决定已写入业务成本</div></article>
      <article class="sp-metric-card sp-violet"><div class="sp-metric-label">审批后有新数据</div><div class="sp-metric-value">{{ statusCounts.changed_after_approval }}</div><div class="sp-metric-foot">需要重新确认最新成本</div></article>
    </section>

      <section class="sp-panel cost-review-table-panel">
      <header class="sp-panel-head">
        <div class="sp-panel-title"><span class="sp-section-index">01</span><div><h2>成本核对列表</h2><span>接口值、计算值与当前生效值均保留 6 位小数语义</span></div></div>
        <div class="cost-review-bulk-actions">
          <span v-if="selectedKeys.length" class="sp-status info">已选 {{ selectedKeys.length }} 条，可审批 {{ bulkApprovableReviews.length }} 条</span>
          <button v-if="bulkApprovableReviews.length" class="sp-button primary" type="button" data-test="bulk-approve" :disabled="bulkApproving" @click="openBulkApproval">一键审批</button>
          <span class="sp-status info">第 {{ page }} / {{ pageCount }} 页</span>
        </div>
      </header>
      <DataTable :columns="columns" :data="reviews" :loading="loading" row-key="id" selectable :selected-keys="selectedKeys" :virtualize-threshold="1000" @update:selected-keys="selectedKeys = $event" @selection-change="selectedKeys = $event">
        <template #cell-provider_name="{ row }"><strong class="sp-entity">{{ row.provider_name }}</strong><span class="sp-sub">供应商 #{{ row.provider_id }}</span></template>
        <template #cell-upstream_cost="{ row }">{{ formatCost(row.upstream_cost) }}</template>
        <template #cell-calculated_cost="{ row }">{{ formatCost(row.calculated_cost) }}</template>
        <template #cell-auto_adopted_cost="{ row }">{{ formatCost(row.auto_adopted_cost) }}</template>
        <template #cell-final_cost="{ row }">{{ formatCost(row.final_cost) }}</template>
        <template #cell-effective_cost="{ row }"><strong>{{ formatCost(row.effective_cost) }}</strong></template>
        <template #cell-cost_delta="{ row }"><span :class="deltaClass(row.cost_delta)">{{ formatSignedCost(row.cost_delta) }}</span></template>
        <template #cell-status="{ row }"><span class="sp-status" :class="statusClass(row.status)">{{ statusLabel(row.status) }}</span></template>
        <template #cell-decision_type="{ row }">{{ decisionLabel(row.decision_type) }}</template>
        <template #cell-approved_at="{ row }">{{ formatDateTime(row.approved_at) }}</template>
        <template #cell-last_synced_at="{ row }">{{ formatDateTime(row.last_synced_at) }}</template>
        <template #cell-actions="{ row }">
          <div class="cost-review-actions">
            <button class="sp-button small" type="button" :data-test="`approve-${row.id}`" @click="openApproval(row)">{{ row.status === 'approved' ? '重新审批' : '审批' }}</button>
            <button class="sp-button small ghost" type="button" :data-test="`history-${row.id}`" @click="openHistory(row)">历史</button>
          </div>
        </template>
        <template #empty><div class="sp-panel-body sp-empty-state">当前筛选条件下暂无成本核对记录</div></template>
      </DataTable>
      <div class="sp-pagination-row"><Pagination v-model:page="page" v-model:page-size="pageSize" :total="total" :show-jump="total > 100" @update:page="onPageChange" @update:page-size="onPageSizeChange" /></div>
    </section>

    <BaseDialog :show="bulkApprovalVisible" title="一键审批上游成本" width="normal" @close="closeBulkApproval">
      <div class="cost-review-dialog supplier-management-page">
        <div class="review-summary"><div><span>已选记录</span><strong>{{ selectedReviews.length }} 条</strong></div><div><span>本次审批</span><strong>{{ bulkApprovableReviews.length }} 条</strong></div><div><span>跳过记录</span><strong>{{ selectedReviews.length - bulkApprovableReviews.length }} 条</strong></div></div>
        <div class="review-choice-grid">
          <button type="button" class="review-choice" :class="{ active: bulkDecisionType === 'upstream' }" data-test="bulk-decision-upstream" @click="bulkDecisionType = 'upstream'"><span>接口成本</span><strong>统一采用接口值</strong><small>按各记录本次上游接口返回值审批</small></button>
          <button type="button" class="review-choice" :class="{ active: bulkDecisionType === 'calculated' }" data-test="bulk-decision-calculated" @click="bulkDecisionType = 'calculated'"><span>计算成本</span><strong>统一采用计算值</strong><small>按各记录本地计算成本审批</small></button>
          <button type="button" class="review-choice" :class="{ active: bulkDecisionType === 'manual' }" data-test="bulk-decision-manual" @click="bulkDecisionType = 'manual'"><span>手动输入</span><strong>统一手动成本</strong><small>为本次选中的记录写入同一金额</small></button>
        </div>
        <Input v-if="bulkDecisionType === 'manual'" v-model="bulkManualCost" type="number" min="0" step="0.000001" label="统一手动成本" placeholder="请输入成本金额" data-test="bulk-manual-cost" />
        <p class="review-dialog-note">仅审批待审批或审批后有新数据的记录，已审批且没有新数据的记录会自动跳过。</p>
      </div>
      <template #footer><div class="dialog-actions"><button class="sp-button ghost" type="button" @click="closeBulkApproval">取消</button><button class="sp-button primary" type="button" data-test="submit-bulk-approval" :disabled="bulkApproving || bulkApprovableReviews.length === 0" @click="submitBulkApproval">{{ bulkApproving ? '提交中…' : '确认一键审批' }}</button></div></template>
    </BaseDialog>

    <BaseDialog :show="approvalVisible" title="审批上游成本" width="normal" @close="closeApproval">
      <div v-if="approvalRow" class="cost-review-dialog supplier-management-page">
        <div class="review-summary"><div><span>供应商</span><strong>{{ approvalRow.provider_name }}</strong></div><div><span>统计日期</span><strong>{{ formatDateOnly(approvalRow.stat_date) }}</strong></div><div><span>当前生效</span><strong>{{ formatCost(approvalRow.effective_cost) }}</strong></div></div>
        <div class="review-choice-grid">
          <button type="button" class="review-choice" :class="{ active: decisionType === 'upstream' }" data-test="decision-upstream" @click="decisionType = 'upstream'"><span>接口成本</span><strong>{{ formatCost(approvalRow.upstream_cost) }}</strong><small>采用本次上游接口返回值</small></button>
          <button type="button" class="review-choice" :class="{ active: decisionType === 'calculated' }" data-test="decision-calculated" @click="decisionType = 'calculated'"><span>计算成本</span><strong>{{ formatCost(approvalRow.calculated_cost) }}</strong><small>采用系统本地计算值</small></button>
          <button type="button" class="review-choice" :class="{ active: decisionType === 'manual' }" data-test="decision-manual" @click="decisionType = 'manual'"><span>手动输入</span><strong>自定义金额</strong><small>输入非负且最多 6 位小数</small></button>
        </div>
        <Input v-if="decisionType === 'manual'" v-model="manualCost" type="number" min="0" step="0.000001" label="手动成本" placeholder="请输入成本金额" data-test="manual-cost" />
        <p class="review-dialog-note">提交时会携带当前版本 {{ approvalRow.version }}，数据发生变化时将提示刷新后重试。</p>
      </div>
      <template #footer><div class="dialog-actions"><button class="sp-button ghost" type="button" @click="closeApproval">取消</button><button class="sp-button primary" type="button" data-test="submit-approval" :disabled="approving" @click="submitApproval">{{ approving ? '提交中…' : '确认审批' }}</button></div></template>
    </BaseDialog>

    <BaseDialog :show="historyVisible" title="成本核对历史" width="wide" @close="closeHistory">
      <div class="history-dialog supplier-management-page">
        <div v-if="historyLoading" class="sp-empty-state">历史加载中…</div>
        <div v-else-if="history.length === 0" class="sp-empty-state">暂无历史记录</div>
        <ol v-else class="history-list">
          <li v-for="item in history" :key="item.id" class="history-item"><div class="history-marker" :class="item.event_type === 'approve' ? 'approve' : 'sync'"></div><div class="history-content"><div class="history-title"><strong>{{ item.event_type === 'approve' ? '人工审批' : '同步' }}</strong><span>{{ formatDateTime(item.operated_at) }}</span></div><div class="history-values"><span>接口 {{ formatCost(item.upstream_cost) }}</span><span>计算 {{ formatCost(item.calculated_cost) }}</span><span>最终 {{ formatCost(item.final_cost) }}</span><span>{{ statusLabel(item.status) }}</span></div></div></li>
        </ol>
      </div>
    </BaseDialog>
  </SupplierModuleLayout>
</template>

<script setup lang="ts">
import { computed, onMounted, reactive, ref } from 'vue'
import { SupplierModuleLayout } from '@/components/admin/supplier-management'
import BaseDialog from '@/components/common/BaseDialog.vue'
import DataTable from '@/components/common/DataTable.vue'
import DateRangePicker from '@/components/common/DateRangePicker.vue'
import Input from '@/components/common/Input.vue'
import Pagination from '@/components/common/Pagination.vue'
import Select, { type SelectOption } from '@/components/common/Select.vue'
import type { Column } from '@/components/common/types'
import { useAppStore } from '@/stores/app'
import { list as listProviders } from '@/api/admin/supplierProviders'
import { extractApiErrorMessage } from '@/utils/apiError'
import {
  approveSupplierProviderCostReview,
  bulkApproveSupplierProviderCostReviews,
  listSupplierProviderCostReviewHistory,
  listSupplierProviderCostReviews,
  type SupplierCostReviewDecision,
  type SupplierCostReviewStatus,
  type SupplierProviderCostReview,
  type SupplierProviderCostReviewHistory,
} from '@/api/admin/supplierProviderCostReviews'

const appStore = useAppStore()
const filters = reactive<{ keyword: string; providerId: number | null; startDate: string; endDate: string; status: SupplierCostReviewStatus | '' }>({ keyword: '', providerId: null, startDate: '', endDate: '', status: '' })
const reviews = ref<SupplierProviderCostReview[]>([])
const providers = ref<Array<{ id: number; name: string }>>([])
const page = ref(1)
const pageSize = ref(20)
const total = ref(0)
const loading = ref(false)
const lastLoadedAt = ref('')
const approvalVisible = ref(false)
const approvalRow = ref<SupplierProviderCostReview | null>(null)
const decisionType = ref<Exclude<SupplierCostReviewDecision, 'none'>>('calculated')
const manualCost = ref('')
const approving = ref(false)
const selectedKeys = ref<Array<string | number>>([])
const bulkApprovalVisible = ref(false)
const bulkDecisionType = ref<Exclude<SupplierCostReviewDecision, 'none'>>('calculated')
const bulkManualCost = ref('')
const bulkApproving = ref(false)
const historyVisible = ref(false)
const historyLoading = ref(false)
const history = ref<SupplierProviderCostReviewHistory[]>([])

const providerOptions = computed<SelectOption[]>(() => providers.value.map(provider => ({ value: provider.id, label: provider.name })))
const statusOptions: SelectOption[] = [
  { value: 'pending_review', label: '待审批' },
  { value: 'approved', label: '已审批' },
  { value: 'changed_after_approval', label: '审批后有新数据' },
]
const columns: Column[] = [
  { key: 'provider_name', label: '供应商', class: 'min-w-36' },
  { key: 'stat_date', label: '统计日期' },
  { key: 'upstream_cost', label: '接口成本' },
  { key: 'calculated_cost', label: '计算成本' },
  { key: 'auto_adopted_cost', label: '自动采用' },
  { key: 'final_cost', label: '最终成本' },
  { key: 'effective_cost', label: '生效成本' },
  { key: 'cost_delta', label: '接口差额' },
  { key: 'status', label: '状态' },
  { key: 'decision_type', label: '审批方式' },
  { key: 'approved_at', label: '审批时间' },
  { key: 'last_synced_at', label: '最近同步' },
  { key: 'actions', label: '操作', class: 'min-w-32' },
]

const pageCount = computed(() => Math.max(1, Math.ceil(total.value / pageSize.value)))
const statusCounts = computed(() => reviews.value.reduce((counts, row) => { counts[row.status] += 1; return counts }, { pending_review: 0, approved: 0, changed_after_approval: 0 } as Record<SupplierCostReviewStatus, number>))
const selectedReviews = computed(() => reviews.value.filter(row => selectedKeys.value.some(key => String(key) === String(row.id))))
const bulkApprovableReviews = computed(() => selectedReviews.value.filter(row => row.status === 'pending_review' || row.status === 'changed_after_approval'))

function formatCost(value: number | null | undefined) { return value === null || value === undefined ? '--' : Number(value).toFixed(6) }
function formatSignedCost(value: number | null | undefined) { return value === null || value === undefined ? '--' : `${value >= 0 ? '+' : ''}${Number(value).toFixed(6)}` }
function formatDateOnly(value: string | null | undefined) { if (!value) return '--'; const date = new Date(value); return Number.isNaN(date.getTime()) ? value.slice(0, 10) : date.toISOString().slice(0, 10) }
function formatDateTime(value: string | null | undefined) { if (!value) return '--'; const date = new Date(value); return Number.isNaN(date.getTime()) ? value : date.toLocaleString('zh-CN', { hour12: false }) }
function statusLabel(status: SupplierCostReviewStatus) { return { pending_review: '待审批', approved: '已审批', changed_after_approval: '审批后有新数据' }[status] }
function statusClass(status: SupplierCostReviewStatus) { return { pending_review: 'warn', approved: 'good', changed_after_approval: 'info' }[status] }
function decisionLabel(decision: SupplierCostReviewDecision) { return { none: '自动采用计算值', upstream: '接口值', calculated: '计算值', manual: '手动输入' }[decision] }
function deltaClass(value: number | null | undefined) { return value !== null && value !== undefined && value > 0 ? 'cost-positive' : value !== null && value !== undefined && value < 0 ? 'cost-negative' : 'cost-neutral' }

async function loadProviders() { try { const result = await listProviders({ page: 1, page_size: 1000 }); providers.value = result.items.map(item => ({ id: item.id, name: item.name })) } catch (error) { appStore.showError(extractApiErrorMessage(error, '加载供应商失败')) } }
async function loadReviews() { loading.value = true; try { const result = await listSupplierProviderCostReviews({ keyword: filters.keyword.trim() || undefined, provider_id: filters.providerId ?? undefined, start_date: filters.startDate || undefined, end_date: filters.endDate || undefined, status: filters.status, page: page.value, page_size: pageSize.value }); reviews.value = result.items; total.value = result.total; lastLoadedAt.value = new Date().toISOString() } catch (error) { appStore.showError(extractApiErrorMessage(error, '加载成本核对列表失败')) } finally { loading.value = false } }
function applyFilters() { page.value = 1; selectedKeys.value = []; void loadReviews() }
function resetFilters() { filters.keyword = ''; filters.providerId = null; filters.startDate = ''; filters.endDate = ''; filters.status = ''; applyFilters() }
function onPageChange() { selectedKeys.value = []; void loadReviews() }
function onPageSizeChange() { page.value = 1; selectedKeys.value = []; void loadReviews() }
function openApproval(row: SupplierProviderCostReview) { approvalRow.value = row; decisionType.value = row.status === 'changed_after_approval' ? 'calculated' : 'calculated'; manualCost.value = ''; approvalVisible.value = true }
function closeApproval() { if (!approving.value) approvalVisible.value = false }
async function submitApproval() { if (!approvalRow.value) return; if (decisionType.value === 'manual') { const value = manualCost.value.trim(); if (!/^\d+(?:\.\d{1,6})?$/.test(value)) { appStore.showError('请输入非负且最多 6 位小数的金额'); return } } const payload = { decision_type: decisionType.value, ...(decisionType.value === 'manual' ? { manual_cost: Number(manualCost.value) } : {}), version: approvalRow.value.version }; approving.value = true; try { await approveSupplierProviderCostReview(approvalRow.value.id, payload); appStore.showSuccess('成本审批已提交'); approvalVisible.value = false; await loadReviews() } catch (error) { appStore.showError(extractApiErrorMessage(error, '提交成本审批失败')) } finally { approving.value = false } }
function openBulkApproval() { if (bulkApprovableReviews.value.length === 0) return; bulkDecisionType.value = 'calculated'; bulkManualCost.value = ''; bulkApprovalVisible.value = true }
function closeBulkApproval() { if (!bulkApproving.value) bulkApprovalVisible.value = false }
async function submitBulkApproval() {
  if (bulkApprovableReviews.value.length === 0) return
  if (bulkDecisionType.value === 'manual') {
    const value = bulkManualCost.value.trim()
    if (!/^\d+(?:\.\d{1,6})?$/.test(value)) {
      appStore.showError('请输入非负且最多 6 位小数的金额')
      return
    }
  }
  const payload = {
    items: bulkApprovableReviews.value.map(row => ({ id: row.id, version: row.version })),
    decision_type: bulkDecisionType.value,
    ...(bulkDecisionType.value === 'manual' ? { manual_cost: Number(bulkManualCost.value) } : {}),
  }
  bulkApproving.value = true
  try {
    const result = await bulkApproveSupplierProviderCostReviews(payload)
    selectedKeys.value = []
    bulkApprovalVisible.value = false
    await loadReviews()
    appStore.showSuccess(`已批量审批 ${result.count} 条成本核对记录`)
  } catch (error) {
    appStore.showError(extractApiErrorMessage(error, '提交批量成本审批失败'))
  } finally {
    bulkApproving.value = false
  }
}
async function openHistory(row: SupplierProviderCostReview) { historyVisible.value = true; historyLoading.value = true; history.value = []; try { history.value = await listSupplierProviderCostReviewHistory(row.id) } catch (error) { appStore.showError(extractApiErrorMessage(error, '加载成本历史失败')) } finally { historyLoading.value = false } }
function closeHistory() { historyVisible.value = false }

onMounted(async () => { await Promise.all([loadProviders(), loadReviews()]) })
</script>

<style scoped>
.cost-review-head { align-items: flex-start; }
.cost-review-filters { margin-bottom: 18px; }
.cost-review-filter-grid { display: grid; grid-template-columns: minmax(180px, 1fr) minmax(180px, 1fr) minmax(280px, 1.5fr) minmax(180px, 1fr) auto; gap: 12px; align-items: center; padding: 16px; }
.cost-review-metrics { margin-bottom: 18px; }
.cost-review-table-panel { overflow: hidden; }
.cost-review-actions { display: flex; gap: 8px; white-space: nowrap; }
.cost-review-bulk-actions { display: flex; align-items: center; gap: 10px; flex-wrap: wrap; justify-content: flex-end; }
.cost-positive { color: #b45309; }
.cost-negative { color: #047857; }
.cost-neutral { color: #64748b; }
.cost-review-dialog, .history-dialog { --sp-panel: #ffffff; --sp-line: #dbe4ee; --sp-blue: #2563eb; --sp-cyan: #0891b2; --sp-green: #059669; --sp-violet: #7c3aed; --sp-ink: #172033; --sp-muted: #64748b; color: var(--sp-ink); }
.review-summary { display: grid; grid-template-columns: repeat(3, 1fr); gap: 12px; margin-bottom: 20px; padding: 16px; border: 1px solid var(--sp-line); border-radius: 14px; background: linear-gradient(135deg, #f8fbff, #f5f3ff); }
.review-summary span, .review-choice span { display: block; color: var(--sp-muted); font-size: 12px; }
.review-summary strong { display: block; margin-top: 6px; font-size: 16px; }
.review-choice-grid { display: grid; grid-template-columns: repeat(3, 1fr); gap: 12px; margin-bottom: 18px; }
.review-choice { min-height: 118px; padding: 16px; border: 1px solid var(--sp-line); border-radius: 14px; background: var(--sp-panel); text-align: left; transition: border-color .2s, box-shadow .2s, transform .2s; }
.review-choice:hover { border-color: var(--sp-blue); transform: translateY(-1px); }
.review-choice.active { border-color: var(--sp-blue); box-shadow: 0 0 0 3px color-mix(in srgb, var(--sp-blue) 14%, transparent); background: #eff6ff; }
.review-choice strong { display: block; margin: 10px 0 6px; font-size: 18px; }
.review-choice small { color: var(--sp-muted); }
.review-dialog-note { margin-top: 14px; color: var(--sp-muted); font-size: 12px; }
.dialog-actions { display: flex; justify-content: flex-end; gap: 10px; }
.history-list { position: relative; margin: 0; padding: 4px 0 4px 24px; list-style: none; }
.history-list::before { position: absolute; top: 12px; bottom: 12px; left: 6px; width: 1px; background: var(--sp-line); content: ''; }
.history-item { position: relative; display: flex; gap: 14px; padding: 10px 0 18px; }
.history-marker { position: absolute; left: -22px; top: 13px; width: 13px; height: 13px; border: 3px solid #fff; border-radius: 50%; box-shadow: 0 0 0 1px var(--sp-line); background: var(--sp-cyan); }
.history-marker.approve { background: var(--sp-violet); }
.history-content { flex: 1; border: 1px solid var(--sp-line); border-radius: 12px; padding: 12px 14px; background: #fbfdff; }
.history-title, .history-values { display: flex; align-items: center; justify-content: space-between; gap: 12px; flex-wrap: wrap; }
.history-title span, .history-values { color: var(--sp-muted); font-size: 12px; }
.history-values { justify-content: flex-start; margin-top: 9px; }
@media (max-width: 900px) { .cost-review-filter-grid, .review-choice-grid, .review-summary { grid-template-columns: 1fr; } }
</style>

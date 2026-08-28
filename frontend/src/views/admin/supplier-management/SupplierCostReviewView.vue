<template>
  <SupplierModuleLayout>
    <header class="sp-page-head cost-review-head">
      <div>
        <div class="sp-eyebrow">供应商管理 / 成本核对</div>
        <h1>上游成本核对</h1>
        <p class="sp-subtitle">对比接口成本与本地计算成本，逐条确认当前业务生效成本。</p>
      </div>
      <div class="sp-controls">
        <button class="sp-button ghost cost-source-button" type="button" data-test="cost-source-config-button" @click="openCostSourceDialog">成本来源</button>
        <button class="sp-button ghost cost-alert-config-button" type="button" data-test="cost-alert-config-button" @click="openCostAlertConfigDialog">预警配置</button>
        <button class="sp-button ghost cost-alert-events-button" type="button" data-test="cost-alert-events-button" @click="openCostAlertEventsDialog">预警事件</button>
        <span v-if="lastLoadedAt" class="sp-data-note">更新于 {{ formatDateTime(lastLoadedAt) }}</span>
        <button class="sp-button primary" type="button" :disabled="loading" @click="loadReviews">{{ loading ? '刷新中…' : '刷新数据' }}</button>
      </div>
    </header>

    <section class="sp-metric-grid cost-review-metrics" aria-label="成本核对摘要">
      <article class="sp-metric-card sp-blue"><div class="sp-metric-label">当前记录</div><div class="sp-metric-value">{{ total }}</div><div class="sp-metric-foot">按当前筛选条件统计</div></article>
      <article class="sp-metric-card sp-amber"><div class="sp-metric-label">待审批</div><div class="sp-metric-value">{{ statusCounts.pending_review }}</div><div class="sp-metric-foot">首次同步默认采用计算成本</div></article>
      <article class="sp-metric-card sp-green"><div class="sp-metric-label">已审批</div><div class="sp-metric-value">{{ statusCounts.approved }}</div><div class="sp-metric-foot">人工决定已写入业务成本</div></article>
      <article class="sp-metric-card sp-violet"><div class="sp-metric-label">审批后有新数据</div><div class="sp-metric-value">{{ statusCounts.changed_after_approval }}</div><div class="sp-metric-foot">需要重新确认最新成本</div></article>
    </section>

    <section class="sp-panel cost-review-filters" aria-label="成本核对筛选">
      <div class="cost-review-filter-grid">
        <Input v-model="filters.keyword" aria-label="供应商名称" placeholder="搜索供应商名称" @keyup.enter="applyFilters" />
        <Select v-model="filters.providerId" :options="providerOptions" clearable aria-label="供应商" placeholder="全部供应商" @change="applyFilters" />
        <DateRangePicker v-model:start-date="filters.startDate" v-model:end-date="filters.endDate" @change="applyFilters" />
        <Select v-model="filters.status" :options="statusOptions" clearable aria-label="核对状态" placeholder="全部状态" @change="applyFilters" />
        <button class="sp-button ghost" type="button" :disabled="loading" @click="resetFilters">重置筛选</button>
      </div>
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
        <template #cell-provider_name="{ row }">
          <span class="provider-badge" :class="providerColorClass(row.provider_id)" :data-test="`provider-identity-${row.provider_id}`">
            <strong>{{ row.provider_name }}</strong>
            <span class="sp-sub">供应商 #{{ row.provider_id }}</span>
          </span>
        </template>
        <template #cell-upstream_cost="{ row }">{{ formatCost(row.upstream_cost) }}</template>
        <template #cell-calculated_cost="{ row }">{{ formatCost(row.calculated_cost) }}</template>
        <template #cell-auto_adopted_cost="{ row }">{{ formatCost(row.auto_adopted_cost) }}</template>
        <template #cell-final_cost="{ row }">{{ formatCost(row.final_cost) }}</template>
        <template #cell-effective_cost="{ row }"><strong>{{ formatCost(row.effective_cost) }}</strong></template>
        <template #cell-cost_delta="{ row }"><span :class="deltaClass(row.cost_delta)">{{ formatSignedCost(row.cost_delta) }}</span></template>
        <template #cell-status="{ row }"><span class="sp-status" :class="statusClass(row.status)">{{ statusLabel(row.status) }}</span></template>
        <template #cell-decision_type="{ row }">{{ decisionLabel(row.decision_type) }}</template>
        <template #cell-approved_at="{ row }"><span class="sp-sub">{{ formatDateTime(row.approved_at) }}</span></template>
        <template #cell-last_synced_at="{ row }"><span class="sp-sub">{{ formatDateTime(row.last_synced_at) }}</span></template>
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

    <BaseDialog :show="costAlertSettingsVisible" title="成本超额预警配置" width="full" @close="closeCostAlertSettingsDialog">
      <div class="cost-alert-dialog cost-settings-large-dialog cost-alert-settings-dialog supplier-management-page" data-test="cost-alert-settings-section">
        <div class="cost-alert-settings-body">
          <div class="cost-alert-global-settings">
            <Input
              v-model="costAlertSettingsForm.amount"
              type="number"
              min="0"
              step="0.000001"
              label="全局差额阈值"
              hint="填写 0 表示不触发成本超额预警。"
              data-test="cost-alert-global-amount"
            />
            <button class="sp-button primary" type="button" data-test="save-cost-alert-settings" :disabled="savingCostAlertSettings" @click="saveCostAlertSettings">
              {{ savingCostAlertSettings ? '保存中…' : '保存全局阈值' }}
            </button>
          </div>
          <DataTable
            :columns="costAlertOverrideColumns"
            :data="costAlertOverrides"
            :loading="costAlertLoading"
            row-key="id"
            :virtualize-threshold="1000"
          >
            <template #cell-provider_id="{ row }">
              <span class="provider-badge" :class="providerColorClass(row.provider_id)" :data-test="`provider-identity-${row.provider_id}`">
                <strong>{{ providerName(row.provider_id) }}</strong>
                <span class="sp-sub">供应商 #{{ row.provider_id }}</span>
              </span>
            </template>
            <template #cell-amount="{ row }">{{ formatDecimalString(row.amount) }}</template>
            <template #cell-enabled="{ row }">
              <div class="sp-inline">
                <Toggle
                  :model-value="row.enabled"
                  :aria-label="`${providerName(row.provider_id)}成本预警${row.enabled ? '已启用' : '已停用'}`"
                  @click.stop
                  @update:model-value="saveCostAlertOverride({ ...row, enabled: $event })"
                />
                <span class="sp-status" :class="row.enabled ? 'good' : 'info'">{{ row.enabled ? '已启用' : '已停用' }}</span>
              </div>
            </template>
            <template #cell-actions="{ row }">
              <div class="cost-alert-actions">
                <button class="sp-button small ghost" type="button" @click="openCostAlertOverrideDialog(row)">编辑</button>
                <button
                  class="sp-button small danger"
                  type="button"
                  :disabled="deletingCostAlertOverrideId !== null"
                  @click="removeCostAlertOverride(row)"
                >
                  {{ deletingCostAlertOverrideId === row.id ? '删除中…' : '删除' }}
                </button>
              </div>
            </template>
            <template #empty><div class="sp-empty-state">暂无供应商覆盖配置，全部使用全局差额阈值。</div></template>
          </DataTable>
          <div class="cost-alert-add-row">
            <Select
              v-model="costAlertOverrideForm.providerId"
              :options="availableCostAlertProviderOptions"
              clearable
              aria-label="新增覆盖供应商"
              placeholder="选择供应商"
              class="cost-alert-provider-select"
            />
            <Input v-model="costAlertOverrideForm.amount" type="number" min="0" step="0.000001" aria-label="覆盖差额阈值" placeholder="差额阈值" />
            <label class="cost-alert-switch">
              <span>启用覆盖</span>
              <Toggle v-model="costAlertOverrideForm.enabled" />
            </label>
            <button class="sp-button primary" type="button" data-test="add-cost-alert-override" :disabled="savingCostAlertOverride" @click="saveCostAlertOverride()">新增覆盖配置</button>
          </div>
        </div>
      </div>
      <template #footer>
        <div class="dialog-actions">
          <button class="sp-button ghost" type="button" @click="closeCostAlertSettingsDialog">关闭</button>
        </div>
      </template>
    </BaseDialog>

    <BaseDialog :show="costSourceVisible" title="成本来源配置" width="full" @close="closeCostSourceDialog">
      <div class="cost-source-dialog cost-settings-large-dialog cost-source-settings-dialog supplier-management-page" data-test="cost-source-settings-section">
        <div class="cost-source-global">
          <div class="cost-source-field">
            <span class="cost-source-label">全局默认成本来源</span>
            <Select
              v-model="costSourceSettingsForm.costSource"
              :options="costSourceModeOptions"
              aria-label="全局默认成本来源"
              class="cost-source-mode-select"
            />
          </div>
          <button class="sp-button primary" type="button" data-test="save-cost-source-settings" :disabled="savingCostSourceSettings" @click="saveCostSourceSettings">
            {{ savingCostSourceSettings ? '保存中…' : '保存全局来源' }}
          </button>
        </div>
        <p class="cost-source-note">智能模式保持现有核对行为；接口成本优先或计算成本优先时，待审批记录与同步写入都固定采用所选来源。</p>

        <DataTable
          :columns="costSourceOverrideColumns"
          :data="costSourceOverrides"
          :loading="costSourceLoading"
          row-key="id"
          :virtualize-threshold="1000"
        >
          <template #cell-provider_id="{ row }">
            <span class="provider-badge" :class="providerColorClass(row.provider_id)">
              <strong>{{ providerName(row.provider_id) }}</strong>
              <span class="sp-sub">供应商 #{{ row.provider_id }}</span>
            </span>
          </template>
          <template #cell-cost_source="{ row }">
            <span class="sp-tag" :class="costSourceTagClass(row.cost_source)">{{ costSourceLabel(row.cost_source) }}</span>
          </template>
          <template #cell-threshold="{ row }">{{ row.cost_source === 'auto' ? formatDecimalString(row.threshold ? String(row.threshold) : null) : '不适用' }}</template>
          <template #cell-actions="{ row }">
            <div class="cost-source-actions">
              <button class="sp-button small ghost" type="button" @click="openCostSourceOverrideDialog(row)">编辑</button>
              <button
                class="sp-button small danger"
                type="button"
                :disabled="deletingCostSourceOverrideId !== null"
                @click="removeCostSourceOverride(row)"
              >
                {{ deletingCostSourceOverrideId === row.id ? '删除中…' : '删除' }}
              </button>
            </div>
          </template>
          <template #empty><div class="sp-empty-state">暂无供应商单独配置，全部跟随全局默认成本来源。</div></template>
        </DataTable>

        <div class="cost-source-add-row">
          <Select
            v-model="costSourceOverrideForm.providerId"
            :options="availableCostSourceProviderOptions"
            clearable
            aria-label="新增成本来源供应商"
            placeholder="选择供应商"
            class="cost-source-provider-select"
          />
          <Select
            v-model="costSourceOverrideForm.costSource"
            :options="costSourceModeOptions"
            aria-label="新增成本来源模式"
            class="cost-source-mode-select"
          />
          <Input
            v-if="costSourceOverrideForm.costSource === 'auto'"
            v-model="costSourceOverrideForm.threshold"
            type="number"
            min="0"
            step="0.000001"
            aria-label="成本来源偏差阈值"
            placeholder="偏差阈值（留空跟随全局）"
          />
          <button class="sp-button primary" type="button" data-test="add-cost-source-override" :disabled="savingCostSourceOverride" @click="saveCostSourceOverride()">新增单独配置</button>
        </div>
      </div>
      <template #footer>
        <div class="dialog-actions">
          <button class="sp-button ghost" type="button" @click="closeCostSourceDialog">关闭</button>
        </div>
      </template>
    </BaseDialog>

    <BaseDialog :show="costSourceOverrideDialogVisible" title="编辑成本来源单独配置" width="normal" @close="closeCostSourceOverrideDialog">
      <div v-if="costSourceOverrideDialogForm" class="cost-source-dialog supplier-management-page">
        <div class="review-summary">
          <div><span>供应商</span><strong>{{ providerName(costSourceOverrideDialogForm.providerId) }}</strong></div>
          <div><span>全局默认来源</span><strong>{{ costSourceLabel(costSourceSettings.cost_source) }}</strong></div>
          <div><span>全局差额阈值</span><strong>{{ formatDecimalString(costAlertSettings.amount) }}</strong></div>
        </div>
        <div class="cost-source-field">
          <span class="cost-source-label">成本来源</span>
          <Select
            v-model="costSourceOverrideDialogForm.costSource"
            :options="costSourceModeOptions"
            aria-label="成本来源"
            class="cost-source-mode-select"
          />
        </div>
        <Input
          v-if="costSourceOverrideDialogForm.costSource === 'auto'"
          v-model="costSourceOverrideDialogForm.threshold"
          type="number"
          min="0"
          step="0.000001"
          label="覆盖偏差阈值"
          hint="留空表示跟随全局差额阈值。"
          data-test="cost-source-override-threshold"
        />
        <p class="review-dialog-note">仅在智能模式下覆盖阈值参与偏差判断；接口成本优先或计算成本优先时固定采用所选来源。</p>
      </div>
      <template #footer>
        <div class="dialog-actions">
          <button class="sp-button ghost" type="button" @click="closeCostSourceOverrideDialog">取消</button>
          <button class="sp-button primary" type="button" :disabled="savingCostSourceOverride" @click="submitCostSourceOverrideDialog">
            {{ savingCostSourceOverride ? '保存中…' : '保存单独配置' }}
          </button>
        </div>
      </template>
    </BaseDialog>

    <BaseDialog :show="costAlertEventsVisible" title="成本超额预警事件" width="full" @close="closeCostAlertEventsDialog">
      <div class="cost-alert-dialog cost-settings-large-dialog cost-alert-events-dialog supplier-management-page" data-test="cost-alert-events-section">
        <div class="cost-alert-event-toolbar">
          <Select
            v-model="costAlertEventFilters.type"
            :options="costAlertEventTypeOptions"
            clearable
            aria-label="预警类型"
            placeholder="全部类型"
            class="cost-alert-filter-control"
            @change="onCostAlertEventFiltersChange"
          />
          <Select
            v-model="costAlertEventFilters.status"
            :options="costAlertEventStatusOptions"
            clearable
            aria-label="预警状态"
            placeholder="全部状态"
            class="cost-alert-filter-control"
            @change="onCostAlertEventFiltersChange"
          />
          <button class="sp-button ghost" type="button" @click="resetCostAlertEventFilters">重置筛选</button>
        </div>
        <DataTable
          :columns="costAlertEventColumns"
          :data="costAlertEvents"
          :loading="costAlertEventsLoading"
          row-key="id"
          :virtualize-threshold="1000"
        >
          <template #cell-provider_name="{ row }">
            <span class="provider-badge" :class="providerColorClass(row.provider_id)" :data-test="`provider-identity-${row.provider_id}`">
              <strong>{{ row.provider_name }}</strong>
              <span class="sp-sub">{{ row.provider_code }}</span>
            </span>
          </template>
          <template #cell-event_type="{ row }">
            <span class="sp-tag" :class="row.event_type === 'cost_recovered' ? 'good' : 'warn'">{{ costAlertEventTypeLabel(row.event_type) }}</span>
          </template>
          <template #cell-status="{ row }">
            <span class="sp-status" :class="row.status === 'active' ? 'warn' : 'good'">{{ costAlertEventStatusLabel(row.status) }}</span>
          </template>
          <template #cell-stat_date="{ row }">{{ formatDateOnly(row.stat_date) }}</template>
          <template #cell-upstream_cost="{ row }">{{ formatDecimalString(row.upstream_cost) }}</template>
          <template #cell-local_cost="{ row }">{{ formatDecimalString(row.local_cost) }}</template>
          <template #cell-overrun_amount="{ row }">
            <span :class="row.event_type === 'cost_recovered' ? 'cost-negative' : 'cost-positive'">{{ formatDecimalString(row.overrun_amount) }}</span>
          </template>
          <template #cell-threshold="{ row }">{{ formatDecimalString(row.threshold) }}</template>
          <template #cell-observed_at="{ row }">{{ formatDateTime(row.observed_at) }}</template>
          <template #empty><div class="sp-empty-state">暂无成本超额预警事件。</div></template>
        </DataTable>
        <div class="sp-pagination-row">
          <Pagination
            v-model:page="costAlertEventPage"
            v-model:page-size="costAlertEventPageSize"
            :total="costAlertEventTotal"
            :show-jump="costAlertEventTotal > 100"
            @update:page="loadCostAlertEvents"
            @update:page-size="onCostAlertEventPageSizeChange"
          />
        </div>
      </div>
      <template #footer>
        <div class="dialog-actions">
          <button class="sp-button ghost" type="button" @click="closeCostAlertEventsDialog">关闭</button>
        </div>
      </template>
    </BaseDialog>
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

    <BaseDialog :show="costAlertOverrideDialogVisible" title="编辑成本预警覆盖" width="normal" @close="closeCostAlertOverrideDialog">
      <div v-if="costAlertOverrideDialogForm" class="cost-review-dialog supplier-management-page">
        <div class="review-summary">
          <div><span>供应商</span><strong>{{ providerName(costAlertOverrideDialogForm.providerId) }}</strong></div>
          <div><span>覆盖状态</span><strong>{{ costAlertOverrideDialogForm.enabled ? '已启用' : '已停用' }}</strong></div>
          <div><span>全局阈值</span><strong>{{ formatDecimalString(costAlertSettings.amount) }}</strong></div>
        </div>
        <Input
          v-model="costAlertOverrideDialogForm.amount"
          type="number"
          min="0"
          step="0.000001"
          label="覆盖差额阈值"
          hint="填写 0 表示该供应商不触发成本超额预警。"
          data-test="cost-alert-override-amount"
        />
        <label class="cost-alert-switch">
          <span>启用成本预警覆盖</span>
          <Toggle v-model="costAlertOverrideDialogForm.enabled" />
        </label>
        <p class="review-dialog-note">覆盖配置会优先于全局阈值；停用后该供应商不参与成本超额预警。</p>
      </div>
      <template #footer>
        <div class="dialog-actions">
          <button class="sp-button ghost" type="button" @click="closeCostAlertOverrideDialog">取消</button>
          <button class="sp-button primary" type="button" :disabled="savingCostAlertOverride" @click="submitCostAlertOverrideDialog">
            {{ savingCostAlertOverride ? '保存中…' : '保存覆盖配置' }}
          </button>
        </div>
      </template>
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
import Toggle from '@/components/common/Toggle.vue'
import type { Column } from '@/components/common/types'
import { useAppStore } from '@/stores/app'
import { list as listProviders } from '@/api/admin/supplierProviders'
import {
  createSupplierCostAlertOverride,
  deleteSupplierCostAlertOverride,
  getSupplierCostAlertSettings,
  listSupplierCostAlertEvents,
  listSupplierCostAlertOverrides,
  updateSupplierCostAlertOverride,
  updateSupplierCostAlertSettings,
  type SupplierCostAlertEvent,
  type SupplierCostAlertEventListParams,
  type SupplierCostAlertEventType,
  type SupplierCostAlertEventStatus,
  type SupplierCostAlertOverride,
  type SupplierCostAlertSettings,
  type SupplierCostAlertSettingsInput,
} from '@/api/admin/supplierCostAlert'
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
import {
  createSupplierCostSourceOverride,
  deleteSupplierCostSourceOverride,
  getSupplierCostSourceSettings,
  listSupplierCostSourceOverrides,
  updateSupplierCostSourceOverride,
  updateSupplierCostSourceSettings,
  type SupplierCostSourceMode,
  type SupplierCostSourceOverride,
  type SupplierCostSourceSettings,
} from '@/api/admin/supplierCostSource'

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
const costAlertSettings = ref<SupplierCostAlertSettings>({ amount: '0' })
const costAlertSettingsForm = ref<SupplierCostAlertSettingsInput>({ amount: '0' })
const costAlertOverrides = ref<SupplierCostAlertOverride[]>([])
const costAlertEvents = ref<SupplierCostAlertEvent[]>([])
const costAlertLoading = ref(false)
const costAlertEventsLoading = ref(false)
const savingCostAlertSettings = ref(false)
const savingCostAlertOverride = ref(false)
const deletingCostAlertOverrideId = ref<number | null>(null)
const costAlertEventPage = ref(1)
const costAlertEventPageSize = ref(10)
const costAlertEventTotal = ref(0)
const costAlertSettingsVisible = ref(false)
const costAlertEventsVisible = ref(false)
const costAlertOverrideDialogVisible = ref(false)
const costAlertOverrideDialogForm = ref<{ id: number; providerId: number; enabled: boolean; amount: string } | null>(null)
const costAlertOverrideForm = ref<{ providerId: number | null; enabled: boolean; amount: string }>({ providerId: null, enabled: true, amount: '' })
const costAlertEventFilters = ref<{ type: SupplierCostAlertEventType | ''; status: SupplierCostAlertEventStatus | '' }>({ type: '', status: '' })
const costSourceSettings = ref<SupplierCostSourceSettings>({ cost_source: 'auto' })
const costSourceSettingsForm = ref<{ costSource: SupplierCostSourceMode }>({ costSource: 'auto' })
const costSourceOverrides = ref<SupplierCostSourceOverride[]>([])
const costSourceLoading = ref(false)
const savingCostSourceSettings = ref(false)
const savingCostSourceOverride = ref(false)
const deletingCostSourceOverrideId = ref<number | null>(null)
const costSourceVisible = ref(false)
const costSourceOverrideDialogVisible = ref(false)
const costSourceOverrideDialogForm = ref<{ id: number; providerId: number; costSource: SupplierCostSourceMode; threshold: string } | null>(null)
const costSourceOverrideForm = ref<{ providerId: number | null; costSource: SupplierCostSourceMode; threshold: string }>({ providerId: null, costSource: 'auto', threshold: '' })

const providerOptions = computed<SelectOption[]>(() => providers.value.map(provider => ({ value: provider.id, label: provider.name })))
const statusOptions: SelectOption[] = [
  { value: 'pending_review', label: '待审批' },
  { value: 'approved', label: '已审批' },
  { value: 'changed_after_approval', label: '审批后有新数据' },
]
const columns: Column[] = [
  { key: 'provider_name', label: '供应商', class: 'min-w-36' },
  { key: 'stat_date', label: '统计日期' },
  { key: 'upstream_cost', label: '接口成本', class: 'text-right tabular-col' },
  { key: 'calculated_cost', label: '计算成本', class: 'text-right tabular-col' },
  { key: 'auto_adopted_cost', label: '自动采用', class: 'text-right tabular-col muted-col' },
  { key: 'final_cost', label: '最终成本', class: 'text-right tabular-col' },
  { key: 'effective_cost', label: '生效成本', class: 'text-right tabular-col effective-col' },
  { key: 'cost_delta', label: '接口差额', class: 'text-right tabular-col' },
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
const availableCostAlertProviderOptions = computed<SelectOption[]>(() => {
  const configured = new Set(costAlertOverrides.value.map(item => item.provider_id))
  return providerOptions.value.filter(option => !configured.has(Number(option.value)))
})
const availableCostSourceProviderOptions = computed<SelectOption[]>(() => {
  const configured = new Set(costSourceOverrides.value.map(item => item.provider_id))
  return providerOptions.value.filter(option => !configured.has(Number(option.value)))
})
const costSourceModeOptions: SelectOption[] = [
  { value: 'auto', label: '智能模式' },
  { value: 'upstream', label: '接口成本优先' },
  { value: 'calculated', label: '计算成本优先' },
]
const costAlertEventTypeOptions: SelectOption[] = [
  { value: 'cost_overrun', label: '成本超额' },
  { value: 'cost_recovered', label: '成本恢复' },
]
const costAlertEventStatusOptions: SelectOption[] = [
  { value: 'active', label: '活动中' },
  { value: 'resolved', label: '已恢复' },
]
const costAlertOverrideColumns: Column[] = [
  { key: 'provider_id', label: '供应商' },
  { key: 'amount', label: '差额阈值' },
  { key: 'enabled', label: '预警开关' },
  { key: 'updated_at', label: '更新时间' },
  { key: 'actions', label: '操作' },
]
const costSourceOverrideColumns: Column[] = [
  { key: 'provider_id', label: '供应商' },
  { key: 'cost_source', label: '成本来源' },
  { key: 'threshold', label: '偏差阈值' },
  { key: 'updated_at', label: '更新时间' },
  { key: 'actions', label: '操作' },
]
const costAlertEventColumns: Column[] = [
  { key: 'provider_name', label: '供应商' },
  { key: 'event_type', label: '事件类型' },
  { key: 'status', label: '状态' },
  { key: 'stat_date', label: '统计日期' },
  { key: 'upstream_cost', label: '上游成本' },
  { key: 'local_cost', label: '本地成本' },
  { key: 'overrun_amount', label: '超额金额' },
  { key: 'threshold', label: '阈值' },
  { key: 'observed_at', label: '发生时间' },
]

function formatCost(value: number | null | undefined) { return value === null || value === undefined ? '--' : Number(value).toFixed(6) }
function formatDecimalString(value: string | null | undefined) { if (value === undefined || value === null || value === '') return '--'; const amount = Number(value); return Number.isFinite(amount) ? amount.toFixed(6) : '--' }
function formatSignedCost(value: number | null | undefined) { return value === null || value === undefined ? '--' : `${value >= 0 ? '+' : ''}${Number(value).toFixed(6)}` }
function formatDateOnly(value: string | null | undefined) { if (!value) return '--'; const date = new Date(value); return Number.isNaN(date.getTime()) ? value.slice(0, 10) : date.toISOString().slice(0, 10) }
function formatDateTime(value: string | null | undefined) { if (!value) return '--'; const date = new Date(value); return Number.isNaN(date.getTime()) ? value : date.toLocaleString('zh-CN', { hour12: false }) }
function statusLabel(status: SupplierCostReviewStatus) { return { pending_review: '待审批', approved: '已审批', changed_after_approval: '审批后有新数据' }[status] }
function statusClass(status: SupplierCostReviewStatus) { return { pending_review: 'warn', approved: 'good', changed_after_approval: 'info' }[status] }
function decisionLabel(decision: SupplierCostReviewDecision) { return { none: '自动采用计算值', upstream: '接口值', calculated: '计算值', manual: '手动输入' }[decision] }
function deltaClass(value: number | null | undefined) { return value !== null && value !== undefined && value > 0 ? 'cost-positive' : value !== null && value !== undefined && value < 0 ? 'cost-negative' : 'cost-neutral' }
function providerColorClass(providerId: number) {
  return `provider-color-${((Math.abs(providerId) - 1) % 8) + 1}`
}
function providerName(providerId: number) { return providers.value.find(provider => provider.id === providerId)?.name ?? `供应商 #${providerId}` }
function costAlertEventTypeLabel(eventType: string) { return eventType === 'cost_recovered' ? '成本恢复' : '成本超额' }
function costAlertEventStatusLabel(status: string) { return status === 'active' ? '活动中' : status === 'resolved' ? '已恢复' : status }
function costSourceLabel(source: SupplierCostSourceMode | string) {
  return { auto: '智能模式', upstream: '接口成本优先', calculated: '计算成本优先' }[source] ?? source
}
function costSourceTagClass(source: SupplierCostSourceMode | string) {
  return source === 'upstream' ? 'info' : source === 'calculated' ? 'good' : 'warn'
}
function validateCostSourceThreshold(value: string) {
  const threshold = value.trim()
  if (!threshold) return null
  if (!/^\d+(?:\.\d{1,6})?$/.test(threshold)) {
    appStore.showError('偏差阈值必须是非负数字，且最多 6 位小数')
    return undefined
  }
  return Number(threshold)
}
function validateCostAlertAmount(value: string, message = '差额阈值必须是大于或等于 0 的数字') {
  const amount = value.trim()
  if (!amount || Number.isNaN(Number(amount)) || Number(amount) < 0) {
    appStore.showError(message)
    return null
  }
  return amount
}

async function loadCostAlertSettings() {
  try {
    costAlertSettings.value = await getSupplierCostAlertSettings()
    costAlertSettingsForm.value = { amount: costAlertSettings.value.amount || '0' }
  } catch (error) {
    appStore.showError(extractApiErrorMessage(error, '加载成本预警配置失败'))
  }
}

async function saveCostAlertSettings() {
  const amount = validateCostAlertAmount(costAlertSettingsForm.value.amount)
  if (amount === null) return
  savingCostAlertSettings.value = true
  try {
    const saved = await updateSupplierCostAlertSettings({ amount })
    costAlertSettings.value = saved
    costAlertSettingsForm.value = { amount: saved.amount || '0' }
    appStore.showSuccess('全局成本预警阈值已保存')
  } catch (error) {
    appStore.showError(extractApiErrorMessage(error, '保存成本预警阈值失败'))
  } finally {
    savingCostAlertSettings.value = false
  }
}

async function loadCostAlertOverrides() {
  try {
    const result = await listSupplierCostAlertOverrides()
    costAlertOverrides.value = result.items ?? []
  } catch (error) {
    appStore.showError(extractApiErrorMessage(error, '加载供应商覆盖配置失败'))
  }
}

async function loadCostSourceSettings() {
  try {
    const settings = await getSupplierCostSourceSettings()
    costSourceSettings.value = settings
    costSourceSettingsForm.value = { costSource: settings.cost_source || 'auto' }
  } catch (error) {
    appStore.showError(extractApiErrorMessage(error, '加载成本来源配置失败'))
  }
}

async function saveCostSourceSettings() {
  savingCostSourceSettings.value = true
  try {
    const saved = await updateSupplierCostSourceSettings({ cost_source: costSourceSettingsForm.value.costSource })
    costSourceSettings.value = saved
    costSourceSettingsForm.value = { costSource: saved.cost_source || 'auto' }
    appStore.showSuccess('全局成本来源已保存')
  } catch (error) {
    appStore.showError(extractApiErrorMessage(error, '保存成本来源配置失败'))
  } finally {
    savingCostSourceSettings.value = false
  }
}

async function loadCostSourceOverrides() {
  try {
    const result = await listSupplierCostSourceOverrides()
    costSourceOverrides.value = result.items ?? []
  } catch (error) {
    appStore.showError(extractApiErrorMessage(error, '加载供应商成本来源配置失败'))
  }
}

async function openCostSourceDialog() {
  costSourceVisible.value = true
  await Promise.all([loadCostSourceSettings(), loadCostSourceOverrides()])
}

function closeCostSourceDialog() {
  costSourceVisible.value = false
}

function openCostSourceOverrideDialog(row: SupplierCostSourceOverride) {
  costSourceOverrideDialogForm.value = {
    id: row.id,
    providerId: row.provider_id,
    costSource: row.cost_source,
    threshold: row.threshold === null || row.threshold === undefined ? '' : String(row.threshold),
  }
  costSourceOverrideDialogVisible.value = true
}

function closeCostSourceOverrideDialog() {
  if (savingCostSourceOverride.value) return
  costSourceOverrideDialogVisible.value = false
  costSourceOverrideDialogForm.value = null
}

async function saveCostSourceOverride(row?: SupplierCostSourceOverride) {
  if (row) {
    await updateCostSourceOverride(row.id, {
      cost_source: row.cost_source,
      threshold: row.threshold ?? null,
    })
    return
  }
  const providerId = costSourceOverrideForm.value.providerId
  if (!providerId) {
    appStore.showError('请选择需要单独配置成本来源的供应商')
    return
  }
  const threshold = costSourceOverrideForm.value.costSource === 'auto'
    ? validateCostSourceThreshold(costSourceOverrideForm.value.threshold)
    : null
  if (threshold === undefined) return
  await createCostSourceOverride({ provider_id: providerId, cost_source: costSourceOverrideForm.value.costSource, threshold })
  costSourceOverrideForm.value = { providerId: null, costSource: 'auto', threshold: '' }
}

async function createCostSourceOverride(input: { provider_id: number; cost_source: SupplierCostSourceMode; threshold: number | null }) {
  savingCostSourceOverride.value = true
  try {
    await createSupplierCostSourceOverride(input)
    await loadCostSourceOverrides()
    appStore.showSuccess('供应商成本来源已保存')
  } catch (error) {
    appStore.showError(extractApiErrorMessage(error, '保存供应商成本来源配置失败'))
  } finally {
    savingCostSourceOverride.value = false
  }
}

async function updateCostSourceOverride(id: number, input: { cost_source: SupplierCostSourceMode; threshold: number | null }) {
  savingCostSourceOverride.value = true
  try {
    await updateSupplierCostSourceOverride(id, input)
    await loadCostSourceOverrides()
    appStore.showSuccess('供应商成本来源已保存')
  } catch (error) {
    appStore.showError(extractApiErrorMessage(error, '保存供应商成本来源配置失败'))
  } finally {
    savingCostSourceOverride.value = false
  }
}

async function submitCostSourceOverrideDialog() {
  const form = costSourceOverrideDialogForm.value
  if (!form) return
  const threshold = form.costSource === 'auto' ? validateCostSourceThreshold(form.threshold) : null
  if (threshold === undefined) return
  await updateCostSourceOverride(form.id, { cost_source: form.costSource, threshold })
  costSourceOverrideDialogVisible.value = false
  costSourceOverrideDialogForm.value = null
}

async function removeCostSourceOverride(row: SupplierCostSourceOverride) {
  if (!window.confirm(`确认删除「${providerName(row.provider_id)}」的成本来源单独配置？删除后将跟随全局默认来源。`)) return
  deletingCostSourceOverrideId.value = row.id
  try {
    await deleteSupplierCostSourceOverride(row.id)
    await loadCostSourceOverrides()
    appStore.showSuccess('供应商成本来源配置已删除')
  } catch (error) {
    appStore.showError(extractApiErrorMessage(error, '删除供应商成本来源配置失败'))
  } finally {
    deletingCostSourceOverrideId.value = null
  }
}

async function loadCostAlertEvents() {
  costAlertEventsLoading.value = true
  try {
    const params: SupplierCostAlertEventListParams = {
      page: costAlertEventPage.value,
      page_size: costAlertEventPageSize.value,
      event_type: costAlertEventFilters.value.type,
      status: costAlertEventFilters.value.status,
    }
    const result = await listSupplierCostAlertEvents(params)
    costAlertEvents.value = result.items ?? []
    costAlertEventTotal.value = result.total ?? 0
  } catch (error) {
    appStore.showError(extractApiErrorMessage(error, '加载成本预警事件失败'))
  } finally {
    costAlertEventsLoading.value = false
  }
}

async function openCostAlertConfigDialog() {
  costAlertSettingsVisible.value = true
  await Promise.all([loadCostAlertSettings(), loadCostAlertOverrides()])
}

function closeCostAlertSettingsDialog() {
  costAlertSettingsVisible.value = false
}

async function openCostAlertEventsDialog() {
  costAlertEventsVisible.value = true
  await loadCostAlertEvents()
}

function closeCostAlertEventsDialog() {
  costAlertEventsVisible.value = false
}

function openCostAlertOverrideDialog(row: SupplierCostAlertOverride) {
  costAlertOverrideDialogForm.value = { id: row.id, providerId: row.provider_id, enabled: row.enabled, amount: row.amount || '0' }
  costAlertOverrideDialogVisible.value = true
}

function closeCostAlertOverrideDialog() {
  if (savingCostAlertOverride.value) return
  costAlertOverrideDialogVisible.value = false
  costAlertOverrideDialogForm.value = null
}

async function saveCostAlertOverride(row?: SupplierCostAlertOverride) {
  if (row) {
    await updateCostAlertOverride(row.id, { enabled: row.enabled, amount: row.amount })
    return
  }
  const providerId = costAlertOverrideForm.value.providerId
  if (!providerId) {
    appStore.showError('请选择需要覆盖预警阈值的供应商')
    return
  }
  const amount = validateCostAlertAmount(costAlertOverrideForm.value.amount, '请输入有效的覆盖差额阈值')
  if (amount === null) return
  await upsertCostAlertOverride({ provider_id: providerId, enabled: costAlertOverrideForm.value.enabled, amount })
  costAlertOverrideForm.value = { providerId: null, enabled: true, amount: '' }
}

async function updateCostAlertOverride(id: number, input: { enabled: boolean; amount: string }) {
  savingCostAlertOverride.value = true
  try {
    await updateSupplierCostAlertOverride(id, input)
    await loadCostAlertOverrides()
    appStore.showSuccess('成本预警覆盖配置已保存')
  } catch (error) {
    appStore.showError(extractApiErrorMessage(error, '保存供应商覆盖配置失败'))
  } finally {
    savingCostAlertOverride.value = false
  }
}

async function upsertCostAlertOverride(input: { provider_id: number; enabled: boolean; amount: string }) {
  savingCostAlertOverride.value = true
  try {
    await createSupplierCostAlertOverride(input)
    await loadCostAlertOverrides()
    appStore.showSuccess('成本预警覆盖配置已保存')
  } catch (error) {
    appStore.showError(extractApiErrorMessage(error, '保存供应商覆盖配置失败'))
  } finally {
    savingCostAlertOverride.value = false
  }
}

async function submitCostAlertOverrideDialog() {
  const form = costAlertOverrideDialogForm.value
  if (!form) return
  const amount = validateCostAlertAmount(form.amount)
  if (amount === null) return
  await updateCostAlertOverride(form.id, { enabled: form.enabled, amount })
  costAlertOverrideDialogVisible.value = false
  costAlertOverrideDialogForm.value = null
}

async function removeCostAlertOverride(row: SupplierCostAlertOverride) {
  if (!window.confirm(`确认删除「${providerName(row.provider_id)}」的成本预警覆盖配置？删除后将回退到全局阈值。`)) return
  deletingCostAlertOverrideId.value = row.id
  try {
    await deleteSupplierCostAlertOverride(row.id)
    await loadCostAlertOverrides()
    appStore.showSuccess('成本预警覆盖配置已删除')
  } catch (error) {
    appStore.showError(extractApiErrorMessage(error, '删除供应商覆盖配置失败'))
  } finally {
    deletingCostAlertOverrideId.value = null
  }
}

function onCostAlertEventFiltersChange() {
  costAlertEventPage.value = 1
  void loadCostAlertEvents()
}

function resetCostAlertEventFilters() {
  costAlertEventFilters.value = { type: '', status: '' }
  onCostAlertEventFiltersChange()
}

function onCostAlertEventPageSizeChange() {
  costAlertEventPage.value = 1
  void loadCostAlertEvents()
}

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

onMounted(async () => { await Promise.all([loadProviders(), loadReviews(), loadCostAlertSettings(), loadCostSourceSettings()]) })
</script>

<style scoped>
.cost-review-head { align-items: flex-start; }
.cost-review-head .cost-source-button,
.cost-review-head .cost-alert-config-button,
.cost-review-head .cost-alert-events-button {
  --button-accent: #2563eb;
  color: var(--button-accent);
  border-color: color-mix(in srgb, var(--button-accent) 34%, transparent);
  background: linear-gradient(150deg, color-mix(in srgb, var(--button-accent) 8%, transparent), transparent 64%), #fff;
  transition: border-color .2s ease, box-shadow .2s ease, transform .2s ease;
}
.cost-review-head .cost-source-button { --button-accent: #2563eb; }
.cost-review-head .cost-alert-config-button { --button-accent: #d97706; }
.cost-review-head .cost-alert-events-button { --button-accent: #7c3aed; }
.cost-review-head .cost-source-button:hover,
.cost-review-head .cost-alert-config-button:hover,
.cost-review-head .cost-alert-events-button:hover {
  border-color: color-mix(in srgb, var(--button-accent) 68%, transparent);
  box-shadow: 0 0 0 3px color-mix(in srgb, var(--button-accent) 11%, transparent), 0 8px 18px color-mix(in srgb, var(--button-accent) 10%, transparent);
  transform: translateY(-1px);
}
.cost-review-head .cost-source-button:focus-visible,
.cost-review-head .cost-alert-config-button:focus-visible,
.cost-review-head .cost-alert-events-button:focus-visible {
  outline: none;
  border-color: var(--button-accent);
  box-shadow: 0 0 0 3px color-mix(in srgb, var(--button-accent) 18%, transparent);
}
.cost-review-filters { margin-bottom: 18px; }
.cost-review-filter-grid { display: grid; grid-template-columns: minmax(180px, 1fr) minmax(180px, 1fr) minmax(280px, 1.5fr) minmax(180px, 1fr) auto; gap: 12px; align-items: center; padding: 16px; }
.cost-review-metrics { grid-template-columns: repeat(4, minmax(145px, 1fr)); gap: 0.75rem; margin-bottom: 18px; }
.cost-review-metrics .sp-metric-card {
  --sp-metric-accent: var(--sp-blue);
  cursor: default;
  border-color: color-mix(in srgb, var(--sp-metric-accent) 24%, var(--sp-line));
  background:
    linear-gradient(150deg, color-mix(in srgb, var(--sp-metric-accent) 7%, transparent), transparent 56%),
    var(--sp-panel);
}
.cost-review-metrics .sp-metric-card::before {
  content: '';
  position: absolute;
  inset: 0 0 auto 0;
  height: 3px;
  background: linear-gradient(90deg, var(--sp-metric-accent), color-mix(in srgb, var(--sp-metric-accent) 30%, transparent));
}
.cost-review-metrics .sp-metric-card.sp-amber { --sp-metric-accent: var(--sp-amber); }
.cost-review-metrics .sp-metric-card.sp-green { --sp-metric-accent: var(--sp-green); }
.cost-review-metrics .sp-metric-card.sp-violet { --sp-metric-accent: var(--sp-violet); }
.cost-review-metrics .sp-metric-card:hover {
  border-color: color-mix(in srgb, var(--sp-metric-accent) 55%, var(--sp-line));
  box-shadow:
    0 0 0 1px color-mix(in srgb, var(--sp-metric-accent) 22%, transparent),
    0 10px 24px color-mix(in srgb, var(--sp-metric-accent) 8%, transparent);
}
.cost-review-metrics .sp-metric-value {
  margin-top: 0.625rem;
  font-variant-numeric: tabular-nums;
  letter-spacing: -0.02em;
}
.cost-review-metrics .sp-metric-foot {
  margin-top: 0.75rem;
  padding-top: 0.625rem;
  border-top: 1px solid color-mix(in srgb, var(--sp-metric-accent) 14%, var(--sp-soft));
}
.cost-review-table-panel { overflow: hidden; }
.cost-alert-dialog {
  --sp-panel: #ffffff;
  --sp-line: #dbe4ee;
  --sp-blue: #2563eb;
  --sp-cyan: #0891b2;
  --sp-green: #059669;
  --sp-violet: #7c3aed;
  --sp-ink: #172033;
  --sp-muted: #64748b;
  min-height: 0;
  color: var(--sp-ink);
}
.cost-source-dialog {
  --sp-panel: #ffffff;
  --sp-line: #dbe4ee;
  --sp-blue: #2563eb;
  --sp-cyan: #0891b2;
  --sp-green: #059669;
  --sp-amber: #d97706;
  --sp-violet: #7c3aed;
  --sp-ink: #172033;
  --sp-muted: #64748b;
  --sp-soft: #f1f5f9;
  min-height: 0;
  color: var(--sp-ink);
}
/* 成本配置大弹窗：宽度使用 full 档，并让内容区撑满更高的弹窗主体。 */
:global(.modal-content:has(.cost-settings-large-dialog)) {
  min-height: min(88vh, 940px);
}
:global(.modal-content:has(.cost-settings-large-dialog) .modal-body) {
  display: flex;
  align-items: stretch;
}
.cost-settings-large-dialog { width: 100%; }
.cost-alert-settings-dialog .cost-alert-settings-body {
  height: 100%;
  align-items: stretch;
  grid-template-rows: auto minmax(0, 1fr) auto;
}
.cost-alert-settings-dialog :deep(.table-wrapper) {
  min-height: 0;
}
.cost-source-settings-dialog {
  display: flex;
  flex-direction: column;
  gap: 18px;
}
.cost-source-settings-dialog :deep(.table-wrapper) {
  flex: 1 1 auto;
  min-height: 0;
}
.cost-source-global { display: flex; align-items: flex-end; gap: 12px; flex-wrap: wrap; }
.cost-source-field { min-width: 240px; }
.cost-source-label { display: block; margin-bottom: 6px; color: var(--sp-muted); font-size: 12px; }
.cost-source-mode-select { width: 220px; }
.cost-source-note { margin: -6px 0 0; color: var(--sp-muted); font-size: 12px; }
.cost-source-actions { display: flex; gap: 8px; white-space: nowrap; }
.cost-source-add-row { display: flex; align-items: flex-end; gap: 12px; flex-wrap: wrap; padding-top: 16px; border-top: 1px solid var(--sp-line); }
.cost-source-provider-select { min-width: 240px; }
.cost-alert-settings-body { display: grid; gap: 18px; }
.cost-alert-event-toolbar { display: flex; align-items: center; gap: 10px; flex-wrap: wrap; padding-bottom: 14px; margin-bottom: 4px; border-bottom: 1px solid var(--sp-line); }
.cost-alert-global-settings { display: flex; align-items: flex-end; gap: 12px; flex-wrap: wrap; }
.cost-alert-actions { display: flex; gap: 8px; white-space: nowrap; }
.cost-alert-add-row { display: flex; align-items: flex-end; gap: 12px; flex-wrap: wrap; padding-top: 16px; border-top: 1px solid var(--sp-line); }
.cost-alert-provider-select { min-width: 240px; }
.cost-alert-switch { display: flex; align-items: center; gap: 8px; }
.cost-alert-filter-control { min-width: 140px; }
.cost-alert-events-dialog {
  display: flex;
  flex-direction: column;
  gap: 18px;
}
.cost-alert-events-dialog :deep(.table-wrapper) {
  flex: 1 1 auto;
  min-height: 0;
}
.cost-alert-events-dialog .sp-pagination-row {
  flex: 0 0 auto;
  margin-bottom: 0;
}
/* 按供应商 ID 稳定分配 8 色徽标，保证同一供应商在不同区域颜色一致。 */
.provider-badge {
  --provider-accent: var(--sp-blue);
  display: inline-grid;
  grid-template-columns: auto minmax(0, 1fr);
  align-items: center;
  column-gap: 8px;
  max-width: 100%;
  padding: 5px 9px;
  border: 1px solid color-mix(in srgb, var(--provider-accent) 26%, var(--sp-line));
  border-radius: 10px;
  background: linear-gradient(150deg, color-mix(in srgb, var(--provider-accent) 8%, transparent), transparent 64%), #fff;
  text-align: left;
}
.provider-badge::before {
  content: '';
  width: 8px;
  height: 8px;
  border-radius: 50%;
  background: var(--provider-accent);
  box-shadow: 0 0 0 3px color-mix(in srgb, var(--provider-accent) 13%, transparent);
}
.provider-badge strong {
  overflow: hidden;
  color: var(--provider-accent);
  font-size: 13px;
  font-weight: 700;
  line-height: 1.3;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.provider-badge .sp-sub { grid-column: 2; font-size: 11px; }
.provider-color-1 { --provider-accent: #2563eb; }
.provider-color-2 { --provider-accent: #0f766e; }
.provider-color-3 { --provider-accent: #7c3aed; }
.provider-color-4 { --provider-accent: #b45309; }
.provider-color-5 { --provider-accent: #be123c; }
.provider-color-6 { --provider-accent: #047857; }
.provider-color-7 { --provider-accent: #4338ca; }
.provider-color-8 { --provider-accent: #a16207; }
.cost-review-actions { display: flex; gap: 8px; white-space: nowrap; }
/* 金额列：右对齐 + 等宽数字，便于逐位比对成本 */
.tabular-col { font-variant-numeric: tabular-nums; }
/* 次要成本列（自动采用）：视觉弱化，突出人工确认后的生效值 */
.muted-col { color: var(--sp-muted); }
/* 生效成本列：浅蓝渐变底作为当前业务生效值的关键锚点 */
.effective-col {
  background: linear-gradient(90deg, color-mix(in srgb, var(--sp-blue) 5%, transparent), transparent 72%);
  box-shadow: inset 2px 0 0 var(--sp-blue);
}
.effective-col strong { font-weight: 700; }
.cost-review-bulk-actions { display: flex; align-items: center; gap: 10px; flex-wrap: wrap; justify-content: flex-end; }
.cost-positive,
.cost-negative,
.cost-neutral {
  display: inline-flex;
  align-items: center;
  gap: 0.375rem;
  padding: 0.25rem 0.5rem;
  border-radius: 0.375rem;
  font-variant-numeric: tabular-nums;
  font-weight: 600;
}
.cost-positive {
  color: #b45309;
  background: rgba(217, 119, 6, 0.08);
}
.cost-negative {
  color: #047857;
  background: rgba(5, 150, 105, 0.08);
}
.cost-neutral { color: #64748b; background: rgba(100, 116, 139, 0.08); }
.cost-positive::before { content: '▲'; font-size: 0.625em; }
.cost-negative::before { content: '▼'; font-size: 0.625em; }
.cost-neutral::before { content: '—'; font-size: 0.75em; }
.cost-review-dialog,
.history-dialog {
  --sp-panel: #ffffff;
  --sp-line: #dbe4ee;
  --sp-blue: #2563eb;
  --sp-cyan: #0891b2;
  --sp-green: #059669;
  --sp-violet: #7c3aed;
  --sp-ink: #172033;
  --sp-muted: #64748b;
  min-height: 0;
  color: var(--sp-ink);
}
.review-summary { display: grid; grid-template-columns: repeat(3, 1fr); gap: 12px; margin-bottom: 20px; padding: 16px; border: 1px solid var(--sp-line); border-radius: 14px; background: linear-gradient(135deg, #f8fbff, #f5f3ff); }
.review-summary span, .review-choice span { display: block; color: var(--sp-muted); font-size: 12px; }
.review-summary strong { display: block; margin-top: 6px; font-size: 16px; }
.review-choice-grid { display: grid; grid-template-columns: repeat(3, 1fr); gap: 12px; margin-bottom: 18px; }
.review-choice { position: relative; min-height: 118px; padding: 16px; border: 1px solid var(--sp-line); border-radius: 14px; background: var(--sp-panel); text-align: left; transition: border-color .2s, box-shadow .2s, transform .2s; }
.review-choice:hover { border-color: color-mix(in srgb, var(--sp-blue) 55%, var(--sp-line)); transform: translateY(-1px); }
.review-choice.active {
  border-color: var(--sp-blue);
  box-shadow: 0 0 0 3px color-mix(in srgb, var(--sp-blue) 14%, transparent), 0 10px 20px color-mix(in srgb, var(--sp-blue) 8%, transparent);
  background: linear-gradient(150deg, color-mix(in srgb, var(--sp-blue) 7%, transparent), transparent 58%), #eff6ff;
}
.review-choice.active::before {
  content: '';
  position: absolute;
  inset: 0 0 auto 0;
  height: 3px;
  border-radius: 14px 14px 0 0;
  background: linear-gradient(90deg, var(--sp-blue), color-mix(in srgb, var(--sp-blue) 30%, transparent));
}
.review-choice:focus-visible { outline: none; box-shadow: 0 0 0 3px color-mix(in srgb, var(--sp-blue) 18%, transparent); }
.review-choice strong { display: block; margin: 10px 0 6px; font-size: 18px; }
.review-choice small { color: var(--sp-muted); }
.review-dialog-note { margin-top: 14px; color: var(--sp-muted); font-size: 12px; }
.dialog-actions { display: flex; justify-content: flex-end; gap: 10px; }
.history-list { position: relative; margin: 0; padding: 4px 0 4px 24px; list-style: none; }
.history-list::before { position: absolute; top: 12px; bottom: 12px; left: 6px; width: 1px; background: var(--sp-line); content: ''; }
.history-item { position: relative; display: flex; gap: 14px; padding: 10px 0 18px; }
.history-marker { position: absolute; left: -22px; top: 13px; width: 13px; height: 13px; border: 3px solid #fff; border-radius: 50%; box-shadow: 0 0 0 1px var(--sp-line); background: var(--sp-cyan); }
.history-marker.approve { background: var(--sp-violet); }
.history-content {
  flex: 1;
  border: 1px solid var(--sp-line);
  border-radius: 12px;
  padding: 12px 14px;
  background: linear-gradient(150deg, color-mix(in srgb, var(--sp-cyan) 4%, transparent), transparent 62%), #fbfdff;
  transition: border-color .2s ease, box-shadow .2s ease;
}
.history-item:hover .history-content {
  border-color: color-mix(in srgb, var(--sp-cyan) 35%, var(--sp-line));
  box-shadow: 0 6px 18px color-mix(in srgb, var(--sp-cyan) 8%, transparent);
}
.history-marker {
  box-shadow: 0 0 0 1px var(--sp-line), 0 0 0 3px color-mix(in srgb, var(--sp-cyan) 10%, transparent);
}
.history-title, .history-values { display: flex; align-items: center; justify-content: space-between; gap: 12px; flex-wrap: wrap; }
.history-title span, .history-values { color: var(--sp-muted); font-size: 12px; }
.history-values { justify-content: flex-start; margin-top: 9px; }
@media (max-width: 900px) { .cost-review-filter-grid, .review-choice-grid, .review-summary { grid-template-columns: 1fr; } }
</style>

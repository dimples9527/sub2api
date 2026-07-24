<template>
  <SupplierModuleLayout>
    <div class="sp-automation-console">
      <header class="sp-page-head sp-console-head">
        <div>
          <div class="sp-eyebrow">Automation Tasks</div>
          <h1>自动化任务中心</h1>
          <p class="sp-subtitle">集中查看供应商同步、倍率守护与数据清理任务的运行状态。</p>
        </div>
        <div class="sp-controls sp-head-actions">
          <span class="sp-refresh-meta">上次刷新：{{ lastRefreshLabel }}</span>
          <button class="sp-button primary" type="button" :disabled="loading" @click="loadData">
            {{ loading ? '刷新中' : '刷新' }}
          </button>
        </div>
      </header>

      <div v-if="error" class="sp-alert sp-error-line" role="alert">{{ error }}</div>

      <section class="sp-overview-strip" aria-label="自动化任务运行概览">
        <article v-for="metric in metrics" :key="metric.label" class="sp-overview-item" :class="`sp-${metric.tone}`">
          <div class="sp-metric-head">
            <div class="sp-metric-label">{{ metric.label }}</div>
            <span class="sp-metric-signal" aria-hidden="true"></span>
          </div>
          <div class="sp-metric-value">{{ metric.value }}</div>
          <div class="sp-metric-foot">{{ metric.foot }}</div>
        </article>
      </section>

      <div class="sp-console-stack">
        <section class="sp-console-panel sp-task-panel">
          <header class="sp-panel-head">
            <div class="sp-panel-title">
              <div>
                <span class="sp-panel-kicker">Task Control</span>
                <h2>任务控制</h2>
                <p>管理任务状态、执行周期与最近一次运行结果。</p>
              </div>
            </div>
            <div class="sp-panel-signals" aria-label="任务状态摘要">
              <span>已启用 {{ enabledTaskCount }} / {{ tasks.length }}</span>
              <span :class="{ bad: recentExceptionCount > 0 }">异常 {{ recentExceptionCount }}</span>
            </div>
          </header>
          <div class="sp-table-region sp-task-table-region">
            <DataTable
              :columns="taskColumns"
              :data="tasks"
              :loading="loading"
              row-key="task_code"
            >
              <template #cell-task="{ row: task }">
                <div class="sp-entity sp-task-name" :style="taskColorStyle(task.task_code)">{{ task.name }}</div>
                <div class="sp-sub">{{ task.task_code }}</div>
              </template>
              <template #cell-enabled="{ row: task }">
                <span class="sp-status" :class="task.enabled ? 'good' : ''">{{ task.enabled ? '已启用' : '已停用' }}</span>
              </template>
              <template #cell-cron_expression="{ row: task }">
                <span class="sp-status info">{{ formatInterval(task.cron_expression) }}</span>
                <div class="sp-sub">{{ task.cron_expression }}</div>
              </template>

              <template #cell-last_run_at="{ row: task }">
                {{ formatTime(task.last_run_at) }}
              </template>
              <template #cell-last_status="{ row: task }">
                <span class="sp-status" :class="statusTone(task.last_status)">{{ statusText(task.last_status) }}</span>
                <div class="sp-result-cell">
                  <span class="sp-sub sp-message-preview">{{ taskResultSummary(task) }}</span>
                </div>
              </template>

              <template #cell-details="{ row: task }">
                <button
                  v-if="task.last_message || latestRunByTask[task.task_code]"
                  class="sp-link-button"
                  type="button"
                  @click.stop="openTaskLatestResult(task)"
                >
                  查看详情
                </button>
              </template>

              <template #cell-actions="{ row: task }">
                <div class="sp-inline sp-task-actions">
                  <button class="sp-button small ghost" type="button" :disabled="savingCode === task.task_code" @click.stop="openEdit(task)">
                    {{ savingCode === task.task_code ? '保存中' : '编辑' }}
                  </button>
                  <template v-if="task.task_code === 'supplier_account_rate_guard'">
                    <button class="sp-button small ghost sp-preview-action" type="button" :disabled="runningCode === task.task_code" @click.stop="runPreview(task.task_code)">
                      {{ runningCode === task.task_code && runningMode === 'preview' ? '检测中' : '检测预览' }}
                    </button>
                    <button class="sp-button small primary sp-task-primary" type="button" :disabled="runningCode === task.task_code" @click.stop="openAccountRateGuardExecute(task)">
                      {{ runningCode === task.task_code && runningMode === 'execute' ? '执行中' : '立即执行' }}
                    </button>
                    <button
                      class="sp-link-button sp-unbind-log-action"
                      :class="{ 'has-pending': accountRateGuardPendingCount > 0 }"
                      type="button"
                      @click.stop="openAccountRateGuardLogs"
                    >
                      <span>解除绑定日志</span>
                      <span v-if="accountRateGuardPendingCount > 0" class="sp-unbind-log-count">{{ accountRateGuardPendingCount }}</span>
                    </button>
                  </template>
                  <button v-else class="sp-button small primary sp-task-primary" type="button" :disabled="runningCode === task.task_code" @click.stop="runNow(task.task_code)">
                    {{ runningCode === task.task_code ? '运行中' : '立即运行' }}
                  </button>
                </div>
              </template>
              <template #empty>
                暂无自动化任务。
              </template>
            </DataTable>
          </div>
        </section>

        <section class="sp-console-panel sp-history-panel">
          <header class="sp-panel-head sp-history-head">
            <div class="sp-panel-title">
              <div>
                <span class="sp-panel-kicker">Run History</span>
                <h2>执行记录</h2>
                <p>按任务和状态定位最近运行结果。</p>
              </div>
            </div>
            <div class="sp-history-count">{{ runTotal }} 条记录</div>
          </header>
          <div class="sp-panel-body">
            <div class="sp-history-toolbar">
              <div class="sp-run-filters">
                <div class="sp-select-field">
                  <span>任务</span>
                  <Select v-model="runTaskFilter" data-test="run-task-filter" :options="runTaskFilterOptions" :disabled="loading" :searchable="false" @change="applyRunFilters" />
                </div>
                <div class="sp-select-field">
                  <span>状态</span>
                  <Select v-model="runStatusFilter" data-test="run-status-filter" :options="runStatusFilterOptions" :disabled="loading" :searchable="false" @change="applyRunFilters" />
                </div>
                <button class="sp-button small ghost" type="button" :disabled="loading || (!runTaskFilter && !runStatusFilter)" @click="resetRunFilters">
                  重置筛选
                </button>
              </div>
            </div>
            <div class="sp-table-region sp-history-table-region">
              <DataTable
                :columns="runColumns"
                :data="runs"
                :loading="loading"
                row-key="id"
                :sticky-actions-column="false"
              >
                <template #cell-task_code="{ row: run }">
                  <div class="sp-entity sp-task-name" :style="taskColorStyle(run.task_code)">{{ taskName(run.task_code) }}</div>
                  <div class="sp-sub">{{ run.task_code }}</div>
                </template>
                <template #cell-started_at="{ row: run }">
                  {{ formatTime(run.started_at) }}
                </template>
                <template #cell-trigger_source="{ row: run }">
                  {{ triggerText(run.trigger_source) }}
                </template>
                <template #cell-status="{ row: run }">
                  <span class="sp-status" :class="statusTone(run.status)">{{ statusText(run.status) }}</span>
                  <button class="sp-link-button sp-message-preview" type="button" @click="openRunDetail(run)">
                    {{ compactMessage(run.message || '查看详情') }}
                  </button>
                </template>
                <template #cell-counts="{ row: run }">
                  {{ run.processed_count }} / {{ run.success_count }} / {{ run.failed_count }}
                  <div v-if="run.result_detail?.rate_guard" class="sp-run-rate-summary">
                    <span v-if="run.result_detail.rate_guard.raised > 0" class="good">
                      调高 {{ run.result_detail.rate_guard.raised }}
                    </span>
                    <span v-if="rateGuardWarningCount(run) > 0" class="warn">
                      告警 {{ rateGuardWarningCount(run) }}
                    </span>
                  </div>
                </template>
                <template #empty>
                  暂无运行历史。
                </template>
              </DataTable>
              <Pagination
                v-if="runTotal > 0"
                class="sp-run-pagination"
                :page="runPage"
                :total="runTotal"
                :page-size="runPageSize"
                :show-page-size-selector="false"
                @update:page="changeRunPage"
              />
            </div>
          </div>
        </section>
      </div>

      <BaseDialog :show="editVisible" :title="editingTask?.name || '编辑任务'" width="wide" @close="closeEdit">
        <form class="sp-edit-dialog" @submit.prevent="saveTask">
          <section class="sp-edit-summary" aria-label="当前任务摘要">
            <div><span>任务编码</span><strong>{{ editForm.task_code }}</strong></div>
            <div><span>当前状态</span><strong>{{ editForm.enabled ? '已启用' : '已停用' }}</strong></div>
            <div><span>当前周期</span><strong>{{ formatInterval(editForm.cron_expression) }}</strong></div>
          </section>

          <section class="sp-form-section sp-state-section">
            <div class="sp-form-section-head">
              <span>01</span>
              <div><h3>运行状态</h3><p>停用后任务不会由调度器自动触发，仍可保留现有配置。</p></div>
            </div>
            <label class="sp-toggle-field">
              <span>启用任务</span>
              <div class="sp-toggle-row">
                <Toggle v-model="editForm.enabled" />
                <em>{{ editForm.enabled ? '已启用' : '已停用' }}</em>
              </div>
            </label>
          </section>

          <section class="sp-form-section sp-schedule-section">
            <div class="sp-form-section-head">
              <span>02</span>
              <div><h3>调度设置</h3><p>配置执行频率和单次任务允许占用的最长时间。</p></div>
            </div>
            <div class="sp-form-grid">
              <Input :model-value="editIntervalSeconds" type="number" label="执行间隔（秒）" @update:model-value="editIntervalSeconds = toNumber($event, editIntervalSeconds)" />
              <Input :model-value="editForm.timeout_seconds" type="number" label="超时秒数" @update:model-value="editForm.timeout_seconds = toNumber($event, editForm.timeout_seconds)" />
            </div>
            <div class="sp-form-note">执行间隔最小为 1 秒，可按正整数秒配置。</div>
          </section>

          <section v-if="editForm.task_code === 'supplier_rate_guard'" class="sp-form-section sp-policy-section">
            <div class="sp-form-section-head">
              <span>03</span>
              <div><h3>倍率守护策略</h3><p>控制安全倍率以及允许参与计算的快照有效期。</p></div>
            </div>
            <div class="sp-form-grid">
              <Input :model-value="editForm.config.rate_guard_safety_multiplier" type="number" label="安全倍率" @update:model-value="editForm.config.rate_guard_safety_multiplier = toNumber($event, editForm.config.rate_guard_safety_multiplier)" />
              <Input :model-value="editForm.config.rate_guard_max_snapshot_age_seconds" type="number" label="快照最大有效期（秒）" @update:model-value="editForm.config.rate_guard_max_snapshot_age_seconds = toNumber($event, editForm.config.rate_guard_max_snapshot_age_seconds)" />
            </div>
          </section>

          <section v-if="editForm.task_code === 'supplier_data_cleanup'" class="sp-form-section sp-policy-section">
            <div class="sp-form-section-head">
              <span>03</span>
              <div><h3>数据保留策略</h3><p>分别设置自动化记录、同步数据和失效对象的保留时间。</p></div>
            </div>
            <div class="sp-form-grid sp-retention-grid">
              <Input :model-value="editForm.config.automation_run_retention_days" type="number" label="自动化运行保留天数" @update:model-value="editForm.config.automation_run_retention_days = toNumber($event, editForm.config.automation_run_retention_days)" />
              <Input :model-value="editForm.config.sync_run_retention_days" type="number" label="同步记录保留天数" @update:model-value="editForm.config.sync_run_retention_days = toNumber($event, editForm.config.sync_run_retention_days)" />
              <Input :model-value="editForm.config.metric_snapshot_retention_days" type="number" label="快照保留天数" @update:model-value="editForm.config.metric_snapshot_retention_days = toNumber($event, editForm.config.metric_snapshot_retention_days)" />
              <Input :model-value="editForm.config.daily_stat_retention_days" type="number" label="每日统计保留天数" @update:model-value="editForm.config.daily_stat_retention_days = toNumber($event, editForm.config.daily_stat_retention_days)" />
              <Input :model-value="editForm.config.inactive_account_retention_days" type="number" label="失效账号保留天数" @update:model-value="editForm.config.inactive_account_retention_days = toNumber($event, editForm.config.inactive_account_retention_days)" />
              <Input :model-value="editForm.config.inactive_group_retention_days" type="number" label="失效分组保留天数" @update:model-value="editForm.config.inactive_group_retention_days = toNumber($event, editForm.config.inactive_group_retention_days)" />
            </div>
          </section>
        </form>
        <template #footer>
          <button class="sp-button ghost" type="button" :disabled="Boolean(savingCode)" @click="closeEdit">取消</button>
          <button class="sp-button primary" type="button" :disabled="Boolean(savingCode)" @click="saveTask">{{ savingCode ? '保存中' : '保存任务' }}</button>
        </template>
      </BaseDialog>

      <BaseDialog :show="detailVisible" :title="detailTitle || '结果详情'" width="extra-wide" @close="closeResultDetail">
        <div v-if="detailRun" :class="['sp-run-detail', statusTone(detailRun.status)]">
          <section class="sp-detail-outcome">
            <div class="sp-detail-section-head">
              <div>
                <span class="sp-detail-section-kicker">Execution Outcome</span>
                <h3>执行结论</h3>
              </div>
              <span class="sp-status" :class="statusTone(detailRun.status)">{{ statusText(detailRun.status) }}</span>
            </div>

            <section class="sp-run-detail-summary">
              <div class="sp-summary-item sp-summary-task">
                <span class="sp-detail-label">任务</span>
                <strong>{{ detailRun.task_code }}</strong>
              </div>
              <div class="sp-summary-item sp-summary-trigger">
                <span class="sp-detail-label">触发</span>
                <strong>{{ triggerText(detailRun.trigger_source) }}</strong>
              </div>
              <div class="sp-summary-item sp-summary-status" :class="statusTone(detailRun.status)">
                <span class="sp-detail-label">状态</span>
                <span class="sp-status" :class="statusTone(detailRun.status)">{{ statusText(detailRun.status) }}</span>
              </div>
              <div class="sp-summary-item sp-summary-counts">
                <span class="sp-detail-label">处理 / 成功 / 失败</span>
                <strong>{{ detailRun.processed_count }} / {{ detailRun.success_count }} / {{ detailRun.failed_count }}</strong>
              </div>
              <div class="sp-summary-item sp-summary-start">
                <span class="sp-detail-label">开始</span>
                <strong>{{ formatTime(detailRun.started_at) }}</strong>
              </div>
              <div class="sp-summary-item sp-summary-end">
                <span class="sp-detail-label">结束</span>
                <strong>{{ formatTime(detailRun.finished_at) }}</strong>
              </div>
            </section>

            <div v-if="detailRun.message" class="sp-run-message">{{ detailRun.message }}</div>
          </section>

          <section class="sp-detail-content">
            <div class="sp-detail-section-head">
              <div>
                <span class="sp-detail-section-kicker">Result Detail</span>
                <h3>结果明细</h3>
              </div>
            </div>

            <section v-if="detailRun.result_detail?.rate_guard && rateGuardResult" class="sp-rate-guard-detail">
              <div class="sp-rate-guard-summary">
                <div><span>检查</span><strong>{{ rateGuardResult.checked }}</strong></div>
                <div><span>调高</span><strong>{{ rateGuardResult.raised }}</strong></div>
                <div><span>无需调整</span><strong>{{ rateGuardResult.unchanged }}</strong></div>
                <div><span>重复快照</span><strong>{{ rateGuardResult.duplicate }}</strong></div>
                <div><span>快照过期</span><strong>{{ rateGuardResult.stale }}</strong></div>
                <div><span>无效</span><strong>{{ rateGuardResult.invalid }}</strong></div>
                <div><span>失败</span><strong>{{ rateGuardResult.failed }}</strong></div>
              </div>

              <section v-if="rateGuardAlertItems.length" class="sp-rate-guard-alerts">
                <header class="sp-rate-guard-section-head">
                  <div>
                    <span>Warnings</span>
                    <h4>告警记录</h4>
                  </div>
                  <strong>{{ rateGuardAlertItems.length }} 项</strong>
                </header>
                <div class="sp-rate-guard-items sp-rate-guard-alert-items">
                  <div class="sp-rate-guard-row sp-rate-guard-row-head" aria-hidden="true">
                    <span>供应商 / 上游分组</span>
                    <span>本地分组</span>
                    <span>原倍率</span>
                    <span>目标倍率</span>
                    <span>告警原因</span>
                    <span>快照时间</span>
                  </div>
                  <div v-for="item in rateGuardAlertItems" :key="`alert-${item.mapping_id}`" class="sp-rate-guard-row">
                    <span>
                      <strong>{{ item.provider_name || `供应商 ${item.provider_id}` }}</strong>
                      <small>{{ item.upstream_group_name || item.upstream_group_key }}</small>
                    </span>
                    <span>
                      <strong>{{ item.local_group_name || (item.local_group_id > 0 ? `本地分组 ${item.local_group_id}` : '未关联本地分组') }}</strong>
                      <small v-if="item.local_group_id > 0">#{{ item.local_group_id }}</small>
                    </span>
                    <span>{{ formatRate(item.old_rate) }}</span>
                    <span>{{ formatRate(item.target_rate) }}</span>
                    <span>
                      <strong>{{ rateGuardActionText(item.action) }}</strong>
                      <small>{{ rateGuardReasonText(item.reason) }}</small>
                    </span>
                    <span>{{ formatTime(item.snapshot_at) }}</span>
                  </div>
                </div>
              </section>

              <section class="sp-rate-guard-changes">
                <header class="sp-rate-guard-section-head">
                  <div>
                    <span>Rate Changes</span>
                    <h4>倍率变更记录</h4>
                  </div>
                  <strong>{{ rateGuardRaisedItems.length }} 项</strong>
                </header>
                <div v-if="rateGuardRaisedItems.length" class="sp-rate-guard-items sp-rate-guard-change-items">
                  <div class="sp-rate-guard-row sp-rate-guard-row-head" aria-hidden="true">
                    <span>供应商 / 上游分组</span>
                    <span>本地分组</span>
                    <span>原倍率</span>
                    <span>调整后倍率</span>
                    <span>结果</span>
                    <span>快照时间</span>
                  </div>
                  <div v-for="item in rateGuardRaisedItems" :key="`change-${item.mapping_id}`" class="sp-rate-guard-row">
                    <span>
                      <strong>{{ item.provider_name || `供应商 ${item.provider_id}` }}</strong>
                      <small>{{ item.upstream_group_name || item.upstream_group_key }}</small>
                    </span>
                    <span>
                      <strong>{{ item.local_group_name || `本地分组 ${item.local_group_id}` }}</strong>
                      <small>#{{ item.local_group_id }}</small>
                    </span>
                    <span>{{ formatRate(item.old_rate) }}</span>
                    <span>{{ formatRate(item.target_rate) }}</span>
                    <span><strong>{{ rateGuardActionText(item.action) }}</strong></span>
                    <span>{{ formatTime(item.snapshot_at) }}</span>
                  </div>
                </div>
                <div v-else class="sp-rate-guard-empty">本次未调整本地分组倍率。</div>
              </section>

              <section class="sp-rate-guard-inspections">
                <header class="sp-rate-guard-section-head">
                  <div>
                    <span>Inspection Results</span>
                    <h4>全部检查结果</h4>
                  </div>
                  <strong>{{ rateGuardResult.items.length }} 项</strong>
                </header>
                <div v-if="rateGuardResult.items.length" class="sp-rate-guard-items">
                  <div class="sp-rate-guard-row sp-rate-guard-row-head" aria-hidden="true">
                    <span>供应商 / 上游分组</span>
                    <span>本地分组</span>
                    <span>原倍率</span>
                    <span>目标倍率</span>
                    <span>结果</span>
                    <span>快照时间</span>
                  </div>
                  <div v-for="item in rateGuardResult.items" :key="item.mapping_id" class="sp-rate-guard-row">
                    <span>
                      <strong>{{ item.provider_name || `供应商 ${item.provider_id}` }}</strong>
                      <small>{{ item.upstream_group_name || item.upstream_group_key }}</small>
                    </span>
                    <span>
                      <strong>{{ item.local_group_name || `本地分组 ${item.local_group_id}` }}</strong>
                      <small>#{{ item.local_group_id }}</small>
                    </span>
                    <span>{{ formatRate(item.old_rate) }}</span>
                    <span>{{ formatRate(item.target_rate) }}</span>
                    <span>
                      <strong>{{ rateGuardActionText(item.action) }}</strong>
                      <small v-if="item.reason">{{ rateGuardReasonText(item.reason) }}</small>
                    </span>
                    <span>{{ formatTime(item.snapshot_at) }}</span>
                  </div>
                </div>
                <div v-else class="sp-rate-guard-empty">本次没有可检查的守护分组。</div>
              </section>
            </section>

            <section v-else-if="detailRun.result_detail?.account_rate_guard && accountRateGuardResult" class="sp-rate-guard-detail">
              <div class="sp-rate-guard-summary sp-account-rate-guard-summary">
                <div><span>运行模式</span><strong>{{ accountRateGuardModeText(accountRateGuardResult.mode) }}</strong></div>
                <div><span>检查供应商</span><strong>{{ accountRateGuardResult.checked_providers }}</strong></div>
                <div><span>同步失败</span><strong>{{ accountRateGuardResult.rate_sync_failed_providers }}</strong></div>
                <div><span>检查账号</span><strong>{{ accountRateGuardResult.checked_accounts }}</strong></div>
                <div><span>风险分组</span><strong>{{ accountRateGuardResult.risk_groups }}</strong></div>
                <div><span>解除绑定</span><strong>{{ accountRateGuardResult.unbound_groups }}</strong></div>
                <div><span>关闭调度</span><strong>{{ accountRateGuardResult.disabled_accounts }}</strong></div>
                <div><span>跳过</span><strong>{{ accountRateGuardResult.skipped }}</strong></div>
                <div><span>失败</span><strong>{{ accountRateGuardResult.failed }}</strong></div>
              </div>
              <div class="sp-run-message">
                {{ accountRateGuardResult.mode === 'preview' ? '预览仅记录计划，不会修改账号分组绑定。' : '实际执行结果已写入独立解除绑定日志。' }}
              </div>
            </section>

            <section v-else-if="detailRun.result_detail?.providers?.length" class="sp-provider-detail-layout">
              <aside class="sp-provider-index" aria-label="供应商结果">
                <button
                  v-for="provider in detailRun.result_detail.providers"
                  :key="provider.provider_id"
                  type="button"
                  class="sp-provider-index-item"
                  :class="[statusTone(provider.status), { active: selectedDetailProvider?.provider_id === provider.provider_id }]"
                  @click="selectDetailProvider(provider.provider_id)"
                >
                  <span class="sp-provider-index-name">{{ provider.provider_name || `供应商 ${provider.provider_id}` }}</span>
                  <span class="sp-status" :class="statusTone(provider.status)">{{ statusText(provider.status) }}</span>
                  <span class="sp-provider-index-meta">
                    {{ provider.counts.checked_count }} / {{ provider.counts.updated_count }} / {{ provider.counts.skipped_count }}
                  </span>
                </button>
              </aside>

              <article v-if="selectedDetailProvider" :class="['sp-provider-card', 'sp-provider-detail-card', statusTone(selectedDetailProvider.status)]">
                <header class="sp-provider-head">
                  <div>
                    <span class="sp-detail-label">供应商 {{ selectedDetailProvider.provider_id }}</span>
                    <h3>{{ selectedDetailProvider.provider_name || `供应商 ${selectedDetailProvider.provider_id}` }}</h3>
                  </div>
                  <span class="sp-status" :class="statusTone(selectedDetailProvider.status)">{{ statusText(selectedDetailProvider.status) }}</span>
                </header>
                <div class="sp-provider-stats">
                  <span class="sp-tag neutral">处理 {{ selectedDetailProvider.counts.checked_count }}</span>
                  <span class="sp-tag success">新增 {{ selectedDetailProvider.counts.created_count }}</span>
                  <span class="sp-tag primary">更新 {{ selectedDetailProvider.counts.updated_count }}</span>
                  <span class="sp-tag warning">跳过 {{ selectedDetailProvider.counts.skipped_count }}</span>
                </div>
                <p v-if="selectedDetailProvider.message" class="sp-provider-message">{{ selectedDetailProvider.message }}</p>

                <div class="sp-stage-groups">
                  <section v-for="category in providerStagesByCategory(selectedDetailProvider)" :key="category.key" class="sp-stage-category" :class="category.key">
                    <h4>{{ category.title }}</h4>
                    <article v-for="stage in category.stages" :key="`${selectedDetailProvider.provider_id}-${stage.scope}`" class="sp-stage-card" :class="statusTone(stage.status)">
                      <div class="sp-stage-head">
                        <strong>{{ scopeText(stage.scope) }}</strong>
                        <span class="sp-status" :class="statusTone(stage.status)">{{ statusText(stage.status) }}</span>
                      </div>
                      <div class="sp-stage-metrics">
                        <span v-if="stage.http_status" class="sp-tag http">HTTP {{ stage.http_status }}</span>
                        <span v-if="stage.duration_ms !== undefined" class="sp-tag timing">{{ stage.duration_ms }}ms</span>
                        <span v-if="stage.response_bytes !== undefined" class="sp-tag neutral">{{ stage.response_bytes }} bytes</span>
                        <span class="sp-tag neutral">处理 {{ stage.counts.checked_count }}</span>
                        <span class="sp-tag success">更新 {{ stage.counts.updated_count }}</span>
                      </div>
                      <div class="sp-stage-body">
                        <div class="sp-stage-main">
                          <div v-if="stage.endpoint" class="sp-stage-row"><em>接口</em><span>{{ stage.endpoint }}</span></div>
                          <div v-if="stage.parsed_summary" class="sp-stage-row"><em>解析</em><span>{{ stage.parsed_summary }}</span></div>
                          <div v-if="stage.error" class="sp-stage-row bad"><em>错误</em><span>{{ stage.error }}</span></div>
                          <div v-if="stage.parse_error" class="sp-stage-row bad"><em>解析错误</em><span>{{ stage.parse_error }}</span></div>
                          <div v-if="stage.message && stage.message !== '同步成功'" class="sp-stage-row"><em>结果</em><span>{{ stage.message }}</span></div>
                        </div>
                        <aside v-if="stage.response_summary" class="sp-response-panel">
                          <div class="sp-response-panel-head">
                            <span>响应摘要</span>
                            <small>原始返回</small>
                          </div>
                          <pre class="sp-response-summary">{{ stage.response_summary }}</pre>
                        </aside>
                      </div>
                    </article>
                  </section>
                </div>
              </article>
            </section>

            <section v-else-if="detailRun.result_detail?.cleanup" class="sp-cleanup-grid">
              <article><span>自动化运行</span><strong>{{ detailRun.result_detail.cleanup.automation_runs }}</strong></article>
              <article><span>同步记录</span><strong>{{ detailRun.result_detail.cleanup.sync_runs }}</strong></article>
              <article><span>指标快照</span><strong>{{ detailRun.result_detail.cleanup.metric_snapshots }}</strong></article>
              <article><span>每日统计</span><strong>{{ detailRun.result_detail.cleanup.daily_stats }}</strong></article>
              <article><span>供应商账号</span><strong>{{ detailRun.result_detail.cleanup.accounts }}</strong></article>
              <article><span>供应商分组</span><strong>{{ detailRun.result_detail.cleanup.groups }}</strong></article>
            </section>

            <pre v-else class="sp-message-detail">{{ detailMessage }}</pre>
          </section>
        </div>
        <pre v-else class="sp-message-detail">{{ detailMessage }}</pre>
        <template #footer>
          <button class="sp-button primary" type="button" @click="closeResultDetail">关闭</button>
        </template>
      </BaseDialog>

      <BaseDialog :show="accountRateGuardExecuteVisible" title="确认执行账号倍率守护" width="wide" @close="closeAccountRateGuardExecute">
        <div class="sp-guard-confirm">
          <span class="sp-guard-confirm-mark" aria-hidden="true">!</span>
          <div>
            <h3>本次操作会修改账号分组绑定</h3>
            <p>执行前会重新同步账号倍率，并解除所有不合格的账号与分组绑定；账号没有剩余分组时会同时关闭调度。</p>
            <p class="sp-guard-confirm-note">该任务不会自动恢复绑定。建议先使用“检测预览”核对计划。</p>
          </div>
        </div>
        <template #footer>
          <button class="sp-button ghost" type="button" :disabled="Boolean(runningCode)" @click="closeAccountRateGuardExecute">取消</button>
          <button class="sp-button primary" type="button" :disabled="Boolean(runningCode)" @click="executeAccountRateGuard">
            {{ runningCode ? '执行中' : '确认解除绑定' }}
          </button>
        </template>
      </BaseDialog>

      <SupplierAccountRateGuardLogDialog
        :show="accountRateGuardLogsVisible"
        @close="closeAccountRateGuardLogs"
        @pending-count-change="updateAccountRateGuardPendingCount"
      />

      <Transition name="sp-fade"><div v-if="toast" class="sp-toast">{{ toast }}</div></Transition>
    </div>
  </SupplierModuleLayout>
</template>

<script setup lang="ts">
import { computed, onMounted, reactive, ref } from 'vue'
import { SupplierAccountRateGuardLogDialog, SupplierModuleLayout } from '@/components/admin/supplier-management'
import BaseDialog from '@/components/common/BaseDialog.vue'
import DataTable from '@/components/common/DataTable.vue'
import Input from '@/components/common/Input.vue'
import Pagination from '@/components/common/Pagination.vue'
import Select, { type SelectOption } from '@/components/common/Select.vue'
import Toggle from '@/components/common/Toggle.vue'
import type { Column } from '@/components/common/types'
import {
  listAccountRateGuardUnbindLogs,
  listRuns,
  listTasks,
  runTask,
  updateTask,
  type SupplierAutomationProviderRunDetail,
  type SupplierAutomationRun,
  type SupplierAutomationStageRunDetail,
  type SupplierAutomationTask,
} from '@/api/admin/supplierAutomation'

const tasks = ref<SupplierAutomationTask[]>([])
const runs = ref<SupplierAutomationRun[]>([])
const lastRefreshedAt = ref('')
const loading = ref(false)
const savingCode = ref('')
const runningCode = ref('')
const runningMode = ref<'preview' | 'execute'>('execute')
const editVisible = ref(false)
const editingTask = ref<SupplierAutomationTask | null>(null)
const editIntervalSeconds = ref(900)
const detailVisible = ref(false)
const detailTitle = ref('')
const detailMessage = ref('')
const detailRun = ref<SupplierAutomationRun | null>(null)
const selectedDetailProviderID = ref<number | null>(null)
const error = ref('')
const toast = ref('')
const runPage = ref(1)
const runPageSize = ref(10)
const runTotal = ref(0)
const runTaskFilter = ref('')
const runStatusFilter = ref('')
const accountRateGuardExecuteVisible = ref(false)
const pendingExecuteTask = ref<SupplierAutomationTask | null>(null)
const accountRateGuardLogsVisible = ref(false)
const accountRateGuardPendingCount = ref(0)

let toastTimer: number | undefined

const editForm = reactive<SupplierAutomationTask>({
  id: 0,
  task_code: '',
  name: '',
  enabled: true,
  cron_expression: '',
  timeout_seconds: 600,
  config: {
    rate_guard_safety_multiplier: 1.1,
    rate_guard_max_snapshot_age_seconds: 1800,
    automation_run_retention_days: 30,
    sync_run_retention_days: 30,
    metric_snapshot_retention_days: 30,
    daily_stat_retention_days: 365,
    inactive_account_retention_days: 90,
    inactive_group_retention_days: 90,
  },
  last_status: '',
  last_message: '',
})

const taskNameByCode = computed<Record<string, string>>(() =>
  Object.fromEntries(tasks.value.map(task => [task.task_code, task.name]))
)

const latestRunByTask = computed<Record<string, SupplierAutomationRun>>(() => {
  const latest: Record<string, SupplierAutomationRun> = {}
  for (const run of runs.value) {
    if (!latest[run.task_code]) latest[run.task_code] = run
  }
  return latest
})
const detailProviders = computed(() => detailRun.value?.result_detail?.providers || [])
const selectedDetailProvider = computed(() => {
  return detailProviders.value.find(provider => provider.provider_id === selectedDetailProviderID.value) || detailProviders.value[0] || null
})
const rateGuardAlertActions = new Set(['invalid', 'stale', 'failed'])
const rateGuardResult = computed(() => detailRun.value?.result_detail?.rate_guard || null)
const accountRateGuardResult = computed(() => detailRun.value?.result_detail?.account_rate_guard || null)
const rateGuardAlertItems = computed(() => (
  rateGuardResult.value?.items.filter(item => rateGuardAlertActions.has(item.action)) || []
))
const rateGuardRaisedItems = computed(() => (
  rateGuardResult.value?.items.filter(item => item.action === 'raised') || []
))
const runTotalPages = computed(() => Math.max(1, Math.ceil(runTotal.value / runPageSize.value)))

const runTaskFilterOptions = computed<SelectOption[]>(() => [
  { value: '', label: '全部任务' },
  ...tasks.value.map(task => ({ value: task.task_code, label: task.name })),
])
const runStatusFilterOptions: SelectOption[] = [
  { value: '', label: '全部状态' },
  { value: 'success', label: '成功' },
  { value: 'partial', label: '部分成功' },
  { value: 'failed', label: '失败' },
  { value: 'running', label: '运行中' },
]
const taskColumns: Column[] = [
  { key: 'task', label: '任务', class: 'min-w-[210px]' },
  { key: 'enabled', label: '状态', class: 'min-w-[90px]' },
  { key: 'cron_expression', label: '执行周期', class: 'min-w-[140px]' },
  { key: 'last_run_at', label: '最近运行', class: 'min-w-[160px]' },
  { key: 'last_status', label: '最近结果', class: 'min-w-[220px]' },
  { key: 'details', label: '详情', class: 'min-w-[90px]' },
  { key: 'actions', label: '操作', class: 'min-w-[310px]' },
]
const runColumns: Column[] = [
  { key: 'task_code', label: '任务', class: 'min-w-[140px]' },
  { key: 'started_at', label: '运行时间', class: 'min-w-[150px]' },
  { key: 'trigger_source', label: '触发' },
  { key: 'status', label: '状态', class: 'min-w-[170px]' },
  { key: 'counts', label: '处理 / 成功 / 失败', class: 'min-w-[150px]' },
]
const lastRefreshLabel = computed(() => (
  lastRefreshedAt.value ? formatTime(lastRefreshedAt.value) : '尚未刷新'
))

const enabledTaskCount = computed(() =>
  tasks.value.filter(task => task.enabled).length
)

const recentExceptionCount = computed(() =>
  runs.value.filter(run => run.status === 'failed' || run.status === 'partial').length
)

const runningTaskCount = computed(() =>
  tasks.value.filter(task =>
    task.last_status === 'running' || task.task_code === runningCode.value
  ).length
)

const metrics = computed(() => [
  { tone: 'neutral', label: '任务总数', value: String(tasks.value.length), foot: '当前已登记的自动化任务' },
  { tone: 'green', label: '已启用', value: String(enabledTaskCount.value), foot: '可由调度器自动执行' },
  { tone: 'red', label: '最近异常', value: String(recentExceptionCount.value), foot: '当前已加载记录中的异常' },
  { tone: 'blue', label: '正在运行', value: String(runningTaskCount.value), foot: runningTaskCount.value ? '已有任务正在执行' : '当前没有运行任务' },
])

onMounted(async () => {
  await loadData()
})

async function loadData() {
  loading.value = true
  error.value = ''
  try {
    tasks.value = await listTasks()
    await loadRuns()
    await loadAccountRateGuardPendingCount()
    lastRefreshedAt.value = new Date().toISOString()
  } catch (err) {
    error.value = err instanceof Error ? err.message : '加载自动化任务失败'
  } finally {
    loading.value = false
  }
}

function openAccountRateGuardLogs() {
  accountRateGuardLogsVisible.value = true
}

function closeAccountRateGuardLogs() {
  accountRateGuardLogsVisible.value = false
}

async function loadAccountRateGuardPendingCount() {
  const result = await listAccountRateGuardUnbindLogs({
    result: 'unbound',
    status: 'pending',
    page: 1,
    page_size: 1,
  })
  accountRateGuardPendingCount.value = result.pending_count
}

async function updateAccountRateGuardPendingCount() {
  await loadAccountRateGuardPendingCount()
}

function taskName(taskCode: string): string {
  return taskNameByCode.value[taskCode] || taskCode
}

function stableTaskColorHash(value: string): number {
  let hash = 2166136261
  for (let index = 0; index < value.length; index += 1) {
    hash ^= value.charCodeAt(index)
    hash = Math.imul(hash, 16777619)
  }
  return hash >>> 0
}

function taskColorStyle(taskCode: string): Record<string, string> {
  const hash = stableTaskColorHash(taskCode || 'supplier_automation_task')
  return {
    '--sp-task-hue': String(hash % 360),
    '--sp-task-saturation': `${58 + ((hash >>> 8) % 18)}%`,
  }
}

async function loadRuns() {
  const result = await listRuns({
    task_code: runTaskFilter.value || undefined,
    status: runStatusFilter.value || undefined,
    page: runPage.value,
    page_size: runPageSize.value,
  })
  runs.value = result.items
  runTotal.value = result.total
}

function openEdit(task: SupplierAutomationTask) {
  editingTask.value = task
  Object.assign(editForm, JSON.parse(JSON.stringify(task)))
  editIntervalSeconds.value = cronToIntervalSeconds(task.cron_expression) || 300
  editVisible.value = true
}

function closeEdit() {
  editVisible.value = false
}

async function saveTask() {
  if (!editingTask.value) return
  const cronExpression = intervalSecondsToCron(editIntervalSeconds.value)
  if (!cronExpression) {
    error.value = '执行间隔必须是正整数秒'
    return
  }
  if (editForm.task_code === 'supplier_rate_guard') {
    if (editForm.config.rate_guard_safety_multiplier <= 0) {
      error.value = '安全倍率必须大于 0'
      return
    }
    if (editForm.config.rate_guard_max_snapshot_age_seconds < 60) {
      error.value = '快照最大有效期不能少于 60 秒'
      return
    }
  }
  editForm.cron_expression = cronExpression
  savingCode.value = editingTask.value.task_code
  try {
    await updateTask(editingTask.value.task_code, editForm)
    showToast('任务已保存')
    editVisible.value = false
    await loadData()
  } catch (err) {
    error.value = err instanceof Error ? err.message : '保存任务失败'
  } finally {
    savingCode.value = ''
  }
}

async function runNow(taskCode: string) {
  runningCode.value = taskCode
  runningMode.value = 'execute'
  try {
    const run = await runTask(taskCode)
    showToast(`任务执行完成：${statusText(run.status)}`)
    runPage.value = 1
    await loadData()
  } catch (err) {
    error.value = err instanceof Error ? err.message : '运行任务失败'
  } finally {
    runningCode.value = ''
  }
}

async function runPreview(taskCode: string) {
  runningCode.value = taskCode
  runningMode.value = 'preview'
  try {
    const run = await runTask(taskCode, 'preview')
    showToast(`检测预览完成：发现 ${run.result_detail?.account_rate_guard?.risk_groups || 0} 个风险分组`)
    runPage.value = 1
    await loadData()
    openRunDetail(run)
  } catch (err) {
    error.value = err instanceof Error ? err.message : '账号倍率守护检测预览失败'
  } finally {
    runningCode.value = ''
  }
}

function openAccountRateGuardExecute(task: SupplierAutomationTask) {
  pendingExecuteTask.value = task
  accountRateGuardExecuteVisible.value = true
}

function closeAccountRateGuardExecute() {
  if (runningCode.value) return
  accountRateGuardExecuteVisible.value = false
  pendingExecuteTask.value = null
}

async function executeAccountRateGuard() {
  if (!pendingExecuteTask.value) return
  runningCode.value = pendingExecuteTask.value.task_code
  runningMode.value = 'execute'
  try {
    const run = await runTask(pendingExecuteTask.value.task_code, 'execute')
    showToast(`账号倍率守护执行完成：解除 ${run.result_detail?.account_rate_guard?.unbound_groups || 0} 个分组绑定`)
    accountRateGuardExecuteVisible.value = false
    pendingExecuteTask.value = null
    runPage.value = 1
    await loadData()
    openRunDetail(run)
  } catch (err) {
    error.value = err instanceof Error ? err.message : '执行账号倍率守护失败'
  } finally {
    runningCode.value = ''
  }
}

function openResultDetail(title: string, message: string) {
  detailRun.value = null
  selectedDetailProviderID.value = null
  detailTitle.value = title || '结果详情'
  detailMessage.value = message || '暂无结果'
  detailVisible.value = true
}

function openTaskLatestResult(task: SupplierAutomationTask) {
  const run = latestRunByTask.value[task.task_code]
  if (run) {
    openRunDetail(run)
    return
  }
  openResultDetail(`${task.name} 最近结果`, task.last_message)
}

function openRunDetail(run: SupplierAutomationRun) {
  detailRun.value = run
  selectInitialDetailProvider(run)
  detailTitle.value = `${run.task_code} 运行详情：${statusText(run.status)}`
  detailMessage.value = formatRunDetail(run)
  detailVisible.value = true
}

function selectInitialDetailProvider(run: SupplierAutomationRun) {
  const providers = run.result_detail?.providers || []
  const failedProvider = providers.find(provider => provider.status === 'failed')
    || providers.find(provider => (provider.stages || []).some(stage => stage.status === 'failed'))
    || providers[0]
  selectedDetailProviderID.value = failedProvider?.provider_id ?? null
}

function selectDetailProvider(providerID: number) {
  selectedDetailProviderID.value = providerID
}

function accountRateGuardModeText(mode: string): string {
  return mode === 'preview' ? '预览' : '执行'
}

function formatRunDetail(run: SupplierAutomationRun): string {
  const lines = [
    `任务：${run.task_code}`,
    `触发：${triggerText(run.trigger_source)}`,
    `状态：${statusText(run.status)}`,
    `处理 / 成功 / 失败：${run.processed_count} / ${run.success_count} / ${run.failed_count}`,
    `开始时间：${formatTime(run.started_at)}`,
    `结束时间：${formatTime(run.finished_at)}`,
    '',
    run.message || '暂无结果',
  ]
  const providers = run.result_detail?.providers || []
  if (providers.length) {
    lines.push('', '接口明细：')
    for (const provider of providers) {
      lines.push(...formatProviderRunDetail(provider))
    }
  } else if (run.result_detail?.account_rate_guard) {
    const guard = run.result_detail.account_rate_guard
    lines.push(
      '',
      '账号倍率守护明细：',
      `- 模式：${accountRateGuardModeText(guard.mode)}`,
      `- 检查供应商：${guard.checked_providers}`,
      `- 检查账号：${guard.checked_accounts}`,
      `- 风险分组：${guard.risk_groups}`,
      `- 解除绑定：${guard.unbound_groups}`,
      `- 关闭调度：${guard.disabled_accounts}`,
      `- 跳过 / 失败：${guard.skipped} / ${guard.failed}`
    )
  } else if (run.result_detail?.cleanup) {
    const cleanup = run.result_detail.cleanup
    lines.push(
      '',
      '清理明细：',
      `- 自动化运行：${cleanup.automation_runs}`,
      `- 同步记录：${cleanup.sync_runs}`,
      `- 指标快照：${cleanup.metric_snapshots}`,
      `- 每日统计：${cleanup.daily_stats}`,
      `- 供应商账号：${cleanup.accounts}`,
      `- 供应商分组：${cleanup.groups}`
    )
  }
  return lines.join('\n')
}

function formatProviderRunDetail(provider: SupplierAutomationProviderRunDetail): string[] {
  const title = provider.provider_name || `供应商 ${provider.provider_id}`
  const lines = [
    '',
    `供应商 ${provider.provider_id}：${title}`,
    `状态：${statusText(provider.status)}；处理 / 新增 / 更新 / 跳过：${provider.counts.checked_count} / ${provider.counts.created_count} / ${provider.counts.updated_count} / ${provider.counts.skipped_count}`,
  ]
  if (provider.message) lines.push(`结果：${provider.message}`)
  for (const stage of provider.stages || []) {
    lines.push(...formatStageRunDetail(stage))
  }
  return lines
}

function formatStageRunDetail(stage: SupplierAutomationStageRunDetail): string[] {
  const lines = [
    `  - ${scopeText(stage.scope)}：${statusText(stage.status)}`,
    `    计数：${stage.counts.checked_count} / ${stage.counts.created_count} / ${stage.counts.updated_count} / ${stage.counts.skipped_count}`,
  ]
  if (stage.endpoint) lines.push(`    接口：${stage.endpoint}`)
  if (stage.http_status) lines.push(`    HTTP：${stage.http_status}`)
  if (stage.duration_ms !== undefined) lines.push(`    耗时：${stage.duration_ms}ms`)
  if (stage.response_bytes !== undefined) lines.push(`    返回大小：${stage.response_bytes} bytes`)
  if (stage.parsed_summary) lines.push(`    解析摘要：${stage.parsed_summary}`)
  if (stage.error) lines.push(`    错误：${stage.error}`)
  if (stage.parse_error) lines.push(`    解析错误：${stage.parse_error}`)
  if (stage.response_summary) lines.push(`    响应摘要：${stage.response_summary}`)
  if (stage.message && stage.message !== '同步成功') lines.push(`    结果：${stage.message}`)
  return lines
}

function formatRate(rate: number): string {
  return Number.isFinite(rate) && rate > 0 ? rate.toFixed(4).replace(/\.?0+$/, '') : '-'
}


function rateGuardWarningCount(run: SupplierAutomationRun): number {
  const result = run.result_detail?.rate_guard
  return result ? result.invalid + result.stale + result.failed : 0
}

function rateGuardActionText(action: string): string {
  const labels: Record<string, string> = {
    raised: '已调高',
    unchanged: '无需调整',
    duplicate: '重复快照',
    stale: '快照过期',
    invalid: '已冻结',
    failed: '执行失败',
  }
  return labels[action] || action || '-'
}

function rateGuardReasonText(reason?: string): string {
  if (!reason) return ''
  const labels: Record<string, string> = {
    provider_inactive: '供应商已停用',
    guardian_inactive: '守护上游分组已失活',
    local_group_inactive: '本地分组已失活',
    group_sync_not_success: '最近一次分组同步未成功',
    snapshot_stale: '上游倍率快照已过期',
    snapshot_duplicate: '该倍率快照已处理',
    rate_invalid: '倍率数据无效',
    selection_changed: '守护分组已变更',
    snapshot_changed: '倍率快照已更新',
  }
  return labels[reason] || reason
}

function providerStagesByCategory(provider: SupplierAutomationProviderRunDetail) {
  const stages = provider.stages || []
  const categories = [
    { key: 'identity', title: '账号与分组', scopes: ['accounts', 'groups'] },
    { key: 'metrics', title: '余额与成本', scopes: ['balance', 'cost'] },
    { key: 'other', title: '其他接口', scopes: [] },
  ]
  return categories
    .map(category => ({
      key: category.key,
      title: category.title,
      stages: category.key === 'other'
        ? stages.filter(stage => !['accounts', 'groups', 'balance', 'cost'].includes(stage.scope))
        : stages.filter(stage => category.scopes.includes(stage.scope)),
    }))
    .filter(category => category.stages.length > 0)
}

function closeResultDetail() {
  detailVisible.value = false
  detailRun.value = null
  selectedDetailProviderID.value = null
}

function taskResultSummary(task: SupplierAutomationTask): string {
  const run = latestRunByTask.value[task.task_code]
  if (run) return runSummary(run)
  return compactMessage(task.last_message || '暂无结果')
}

function runSummary(run: SupplierAutomationRun): string {
  if (!run.processed_count && !run.success_count && !run.failed_count) {
    return compactMessage(run.message || '暂无结果')
  }
  return `${run.processed_count} 个对象，${run.success_count} 成功，${run.failed_count} 失败`
}

function toNumber(value: string | number, fallback: number): number {
  const next = Number(value)
  return Number.isFinite(next) ? next : fallback
}

async function changeRunPage(page: number) {
  runPage.value = Math.min(Math.max(1, page), runTotalPages.value)
  await refreshRuns()
}

async function applyRunFilters() {
  runPage.value = 1
  await refreshRuns()
}

async function resetRunFilters() {
  runTaskFilter.value = ''
  runStatusFilter.value = ''
  runPage.value = 1
  await refreshRuns()
}

async function refreshRuns() {
  loading.value = true
  error.value = ''
  try {
    await loadRuns()
  } catch (err) {
    error.value = err instanceof Error ? err.message : '加载运行历史失败'
  } finally {
    loading.value = false
  }
}

function compactMessage(message: string): string {
  const normalized = String(message || '').replace(/\s+/g, ' ').trim()
  if (!normalized) return '暂无结果'
  return normalized.length > 80 ? `${normalized.slice(0, 80)}...` : normalized
}

function statusTone(status?: string): string {
  if (status === 'failed') return 'bad'
  if (status === 'partial') return 'warn'
  if (status === 'success') return 'good'
  return ''
}

function statusText(status?: string): string {
  if (status === 'failed') return '失败'
  if (status === 'partial') return '部分成功'
  if (status === 'success') return '成功'
  if (status === 'running') return '运行中'
  return '未运行'
}

function triggerText(trigger?: string): string {
  if (trigger === 'scheduled') return '定时执行'
  if (trigger === 'manual') return '手动执行'
  return trigger || '未知'
}

function scopeText(scope?: string): string {
  if (scope === 'accounts') return '账号接口'
  if (scope === 'groups') return '分组接口'
  if (scope === 'balance') return '余额接口'
  if (scope === 'cost') return '成本接口'
  if (scope === 'all') return '全量同步'
  return scope || '未知接口'
}

function formatTime(value?: string): string {
  if (!value) return '—'
  const date = new Date(value)
  if (Number.isNaN(date.getTime())) return '—'
  return date.toLocaleString('zh-CN')
}

function formatInterval(cronExpression: string): string {
  const seconds = cronToIntervalSeconds(cronExpression)
  if (!seconds) return cronExpression || '未配置'
  if (seconds % 86400 === 0) return `每 ${seconds / 86400} 天`
  if (seconds % 3600 === 0) return `每 ${seconds / 3600} 小时`
  if (seconds % 60 === 0) return `每 ${seconds / 60} 分钟`
  return `每 ${seconds} 秒`
}

function cronToIntervalSeconds(cronExpression: string): number | null {
  const everyMatch = cronExpression.match(/^@every\s+(\d+)s$/)
  if (everyMatch) return Number(everyMatch[1])

  const parts = cronExpression.trim().split(/\s+/)
  if (parts.length !== 5) return null
  const [minute, hour, dayOfMonth, month, dayOfWeek] = parts
  if (hour === '*' && dayOfMonth === '*' && month === '*' && dayOfWeek === '*') {
    const minuteMatch = minute.match(/^\*\/(\d+)$/)
    if (minuteMatch) return Number(minuteMatch[1]) * 60
    if (minute === '*') return 60
  }
  if (minute === '0' && dayOfMonth === '*' && month === '*' && dayOfWeek === '*') {
    const hourMatch = hour.match(/^\*\/(\d+)$/)
    if (hourMatch) return Number(hourMatch[1]) * 3600
    if (hour === '0') return 86400
  }
  if (dayOfMonth === '*' && month === '*' && dayOfWeek === '*' && minute !== '*' && hour !== '*') {
    return 86400
  }
  return null
}

function intervalSecondsToCron(seconds: number): string | null {
  if (!Number.isInteger(seconds) || seconds < 1) return null
  return `@every ${seconds}s`
}

function showToast(message: string) {
  toast.value = message
  window.clearTimeout(toastTimer)
  toastTimer = window.setTimeout(() => { toast.value = '' }, 1800)
}
</script>

<style scoped>
.sp-automation-console {
  display: grid;
  gap: 18px;
  min-width: 0;
}

.sp-console-head {
  align-items: flex-end;
  margin-bottom: 0;
}

.sp-head-actions {
  justify-content: flex-end;
}

.sp-refresh-meta {
  color: var(--sp-muted);
  font-size: 12px;
  font-weight: 600;
}

.sp-task-name {
  display: inline-flex;
  width: fit-content;
  max-width: 100%;
  align-items: center;
  border: 1px solid hsl(var(--sp-task-hue, 215), var(--sp-task-saturation, 16%), 82%);
  border-radius: 7px;
  padding: 3px 8px;
  background: hsl(var(--sp-task-hue, 215), var(--sp-task-saturation, 16%), 96%);
  color: hsl(var(--sp-task-hue, 215), var(--sp-task-saturation, 16%), 32%);
  line-height: 1.35;
}

:global(.dark .sp-automation-console .sp-task-name) {
  border-color: hsl(var(--sp-task-hue, 215), var(--sp-task-saturation, 16%), 34%);
  background: hsl(var(--sp-task-hue, 215), 32%, 18%);
  color: hsl(var(--sp-task-hue, 215), var(--sp-task-saturation, 16%), 76%);
}

.sp-overview-strip {
  display: grid;
  grid-template-columns: repeat(4, minmax(0, 1fr));
  gap: 12px;
  background: transparent;
}

.sp-overview-item {
  --sp-metric-accent: var(--sp-muted);

  position: relative;
  isolation: isolate;
  min-width: 0;
  min-height: 132px;
  overflow: hidden;
  border: 1px solid color-mix(in srgb, var(--sp-metric-accent) 18%, var(--sp-soft));
  border-radius: 14px;
  padding: 18px 18px 16px;
  background:
    radial-gradient(circle at 92% 8%, color-mix(in srgb, var(--sp-metric-accent) 12%, transparent) 0, transparent 44%),
    linear-gradient(145deg, color-mix(in srgb, var(--sp-metric-accent) 5%, transparent), transparent 55%),
    var(--sp-panel);
  box-shadow:
    0 1px 2px color-mix(in srgb, var(--sp-text) 5%, transparent),
    0 8px 22px color-mix(in srgb, var(--sp-metric-accent) 7%, transparent);
  transition: transform 180ms ease, border-color 180ms ease, box-shadow 180ms ease;
}

.sp-overview-item::before {
  position: absolute;
  z-index: 1;
  top: 0;
  left: 18px;
  width: 46px;
  height: 3px;
  border-radius: 0 0 999px 999px;
  background: var(--sp-metric-accent);
  box-shadow: 0 2px 10px color-mix(in srgb, var(--sp-metric-accent) 32%, transparent);
  content: '';
}

.sp-overview-item:hover {
  border-color: color-mix(in srgb, var(--sp-metric-accent) 34%, var(--sp-soft));
  box-shadow:
    0 2px 4px color-mix(in srgb, var(--sp-text) 6%, transparent),
    0 14px 30px color-mix(in srgb, var(--sp-metric-accent) 12%, transparent);
  transform: translateY(-2px);
}

.sp-overview-item.sp-neutral {
  --sp-metric-accent: var(--sp-muted);
}

.sp-overview-item.sp-green {
  --sp-metric-accent: var(--sp-green);
}

.sp-overview-item.sp-red {
  --sp-metric-accent: var(--sp-red);
}

.sp-overview-item.sp-blue {
  --sp-metric-accent: var(--sp-blue);
}

.sp-metric-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
}

.sp-metric-label {
  color: var(--sp-muted);
  font-size: 12px;
  font-weight: 750;
  letter-spacing: 0.04em;
}

.sp-metric-signal {
  width: 8px;
  height: 8px;
  flex: 0 0 auto;
  border-radius: 50%;
  background: var(--sp-metric-accent);
  box-shadow: 0 0 0 4px color-mix(in srgb, var(--sp-metric-accent) 12%, transparent);
}

.sp-overview-item .sp-metric-value {
  margin-top: 10px;
  color: var(--sp-text);
  font-size: clamp(28px, 2.3vw, 34px);
  font-variant-numeric: tabular-nums;
  font-weight: 760;
  letter-spacing: -0.035em;
  line-height: 1;
}

.sp-overview-item .sp-metric-foot {
  margin-top: 13px;
  border-top: 1px solid color-mix(in srgb, var(--sp-metric-accent) 10%, var(--sp-soft));
  padding-top: 10px;
  color: var(--sp-muted);
  font-size: 12px;
  line-height: 1.45;
}

:global(.dark .sp-automation-console .sp-overview-item) {
  border-color: color-mix(in srgb, var(--sp-metric-accent) 24%, var(--sp-soft));
  box-shadow:
    0 1px 2px rgb(0 0 0 / 16%),
    0 10px 26px color-mix(in srgb, var(--sp-metric-accent) 9%, transparent);
}

.sp-console-stack {
  display: grid;
  gap: 18px;
  min-width: 0;
}

.sp-console-panel {
  min-width: 0;
  overflow: hidden;
  border: 1px solid var(--sp-soft);
  border-radius: 14px;
  background: var(--sp-panel);
}

.sp-panel-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 18px;
  border-bottom: 1px solid var(--sp-soft);
  padding: 16px 18px;
}

.sp-panel-title {
  min-width: 0;
}

.sp-panel-kicker {
  color: var(--sp-muted);
  font-size: 10px;
  font-weight: 800;
  letter-spacing: 0.12em;
  text-transform: uppercase;
}

.sp-panel-title h2 {
  margin: 3px 0 0;
  color: var(--sp-text);
  font-size: 17px;
  line-height: 1.35;
}

.sp-panel-title p {
  margin: 4px 0 0;
  color: var(--sp-muted);
  font-size: 12px;
  line-height: 1.5;
}

.sp-panel-signals,
.sp-history-count {
  display: inline-flex;
  align-items: center;
  gap: 8px;
  color: var(--sp-muted);
  font-size: 12px;
  font-weight: 700;
}

.sp-panel-signals {
  flex-wrap: wrap;
  justify-content: flex-end;
}

.sp-panel-signals span {
  border: 1px solid var(--sp-soft);
  border-radius: 999px;
  padding: 5px 9px;
  background: var(--sp-panel);
}

.sp-panel-signals .bad {
  border-color: color-mix(in srgb, var(--sp-red) 36%, var(--sp-soft));
  color: var(--sp-red);
}

.sp-change-log-dialog {
  display: grid;
  gap: 14px;
}

.sp-change-log-head {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 14px;
  border: 1px solid var(--sp-soft);
  border-radius: 10px;
  background: color-mix(in srgb, var(--sp-panel-2) 78%, transparent);
  padding: 14px 16px;
}

.sp-change-log-head > div > span {
  color: var(--sp-muted);
  font-size: 10px;
  font-weight: 800;
  letter-spacing: 0.12em;
  text-transform: uppercase;
}

.sp-change-log-head h3 {
  margin: 3px 0 0;
  color: var(--sp-text);
  font-size: 16px;
}

.sp-change-log-head p {
  margin: 5px 0 0;
  color: var(--sp-muted);
  font-size: 12px;
  line-height: 1.55;
}

.sp-change-log-table-region {
  overflow: hidden;
  border: 1px solid var(--sp-soft);
  border-radius: 10px;
}

.sp-change-log-pagination {
  border-top: 1px solid var(--sp-soft);
}

.sp-unbind-log-dialog .sp-change-log-head {
  border-left: 3px solid var(--sp-blue);
}

.sp-log-result-action {
  padding: 0;
}

.sp-guard-confirm {
  display: grid;
  grid-template-columns: 44px minmax(0, 1fr);
  gap: 16px;
  align-items: start;
  border: 1px solid color-mix(in srgb, var(--sp-amber) 36%, var(--sp-soft));
  border-left: 3px solid var(--sp-amber);
  border-radius: 12px;
  background: color-mix(in srgb, var(--sp-amber) 6%, var(--sp-panel));
  padding: 18px;
}

.sp-guard-confirm-mark {
  display: grid;
  width: 40px;
  height: 40px;
  place-items: center;
  border-radius: 50%;
  background: var(--sp-amber);
  color: white;
  font-size: 22px;
  font-weight: 850;
}

.sp-guard-confirm h3 {
  margin: 0;
  color: var(--sp-text);
  font-size: 17px;
}

.sp-guard-confirm p {
  margin: 8px 0 0;
  color: var(--sp-muted);
  font-size: 13px;
  line-height: 1.65;
}

.sp-guard-confirm .sp-guard-confirm-note {
  color: var(--sp-amber);
  font-weight: 700;
}

.sp-log-error-detail {
  display: grid;
  gap: 10px;
}

.sp-log-error-detail > span {
  color: var(--sp-muted);
  font-size: 12px;
  font-weight: 750;
}

.sp-log-error-detail pre {
  max-height: 360px;
  margin: 0;
  overflow: auto;
  border: 1px solid color-mix(in srgb, var(--sp-red) 30%, var(--sp-soft));
  border-left: 3px solid var(--sp-red);
  border-radius: 10px;
  background: color-mix(in srgb, var(--sp-red) 5%, var(--sp-panel));
  padding: 16px;
  color: var(--sp-text);
  font: 12px/1.7 ui-monospace, SFMono-Regular, Consolas, monospace;
  white-space: pre-wrap;
  overflow-wrap: anywhere;
}

.sp-history-count::before {
  width: 6px;
  height: 6px;
  border-radius: 50%;
  background: var(--sp-blue);
  content: '';
}

.sp-history-panel .sp-panel-body {
  padding: 0;
}

.sp-table-region {
  min-width: 0;
}

.sp-history-toolbar {
  border-bottom: 1px solid var(--sp-soft);
  background: color-mix(in srgb, var(--sp-panel-2) 74%, transparent);
  padding: 12px 18px;
}

.sp-history-toolbar .sp-run-filters {
  margin-bottom: 0;
}

.sp-task-actions {
  max-width: 300px;
  flex-wrap: wrap;
}

.sp-task-primary {
  min-width: 76px;
}

.sp-preview-action {
  border-color: color-mix(in srgb, var(--sp-blue) 32%, var(--sp-soft));
  color: var(--sp-blue);
}

.sp-unbind-log-action {
  flex: 1 0 100%;
  justify-content: flex-start;
  gap: 6px;
  color: var(--sp-muted);
  font-size: 11px;
}

.sp-unbind-log-action.has-pending {
  color: var(--sp-amber);
  font-weight: 700;
}

.sp-unbind-log-count {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  min-width: 20px;
  height: 20px;
  padding: 0 6px;
  border: 1px solid color-mix(in srgb, var(--sp-amber) 36%, var(--sp-line));
  border-radius: 999px;
  background: color-mix(in srgb, var(--sp-amber) 10%, var(--sp-panel));
  color: var(--sp-amber);
  font-size: 11px;
  font-variant-numeric: tabular-nums;
  line-height: 1;
}

.sp-result-cell {
  display: grid;
  gap: 6px;
  max-width: 220px;
}

.sp-run-rate-summary {
  display: flex;
  flex-wrap: wrap;
  gap: 4px 10px;
  margin-top: 4px;
  font-size: 11px;
  font-weight: 700;
}

.sp-run-rate-summary .good {
  color: var(--sp-green);
}

.sp-run-rate-summary .warn {
  color: var(--sp-amber);
}

.sp-run-pagination {
  margin-top: 12px;
  overflow: hidden;
  border: 1px solid var(--sp-soft);
  border-radius: 12px;
}

.sp-run-filters {
  display: grid;
  grid-template-columns: minmax(140px, 1fr) minmax(130px, 0.8fr) auto;
  gap: 10px;
  align-items: end;
  margin-bottom: 12px;
}

.sp-select-field {
  display: grid;
  gap: 5px;
  color: var(--sp-muted);
  font-size: 12px;
  font-weight: 600;
}

.sp-toggle-row {
  display: inline-flex;
  align-items: center;
  gap: 10px;
  min-height: 40px;
  color: var(--sp-muted);
}

.sp-toggle-row em {
  font-style: normal;
  font-size: 13px;
  font-weight: 600;
}

.sp-edit-dialog {
  display: grid;
  min-height: 0;
  max-height: 72vh;
  gap: 18px;
  overflow: auto;
  padding: 4px 2px 12px;
}

.sp-edit-summary {
  display: grid;
  grid-template-columns: repeat(3, minmax(0, 1fr));
  border: 1px solid var(--sp-soft);
  border-radius: 14px;
  background: color-mix(in srgb, var(--sp-blue) 4%, var(--sp-panel));
}

.sp-edit-summary > div {
  min-width: 0;
  border-left: 1px solid var(--sp-soft);
  padding: 14px 16px;
}

.sp-edit-summary > div:first-child {
  border-left: 0;
}

.sp-edit-summary span,
.sp-form-section-head p {
  color: var(--sp-muted);
  font-size: 12px;
}

.sp-edit-summary strong {
  display: block;
  margin-top: 5px;
  overflow: hidden;
  color: var(--sp-text);
  font-size: 14px;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.sp-form-section {
  display: grid;
  gap: 14px;
  border-top: 1px solid var(--sp-soft);
  padding-top: 18px;
}

.sp-state-section {
  border-top: 0;
  padding-top: 0;
}

.sp-form-section-head {
  display: flex;
  align-items: flex-start;
  gap: 12px;
}

.sp-form-section-head > span {
  color: var(--sp-blue);
  font-size: 11px;
  font-weight: 800;
  letter-spacing: 0.08em;
}

.sp-form-section-head h3 {
  margin: 0;
  color: var(--sp-text);
  font-size: 15px;
}

.sp-form-section-head p {
  margin: 4px 0 0;
  line-height: 1.6;
}

.sp-form-grid {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 14px;
}

.sp-toggle-field {
  display: grid;
  justify-items: start;
  gap: 6px;
  color: var(--sp-text);
  font-size: 13px;
  font-weight: 600;
}

.sp-message-detail {
  max-width: min(780px, 78vw);
  max-height: 68vh;
  white-space: pre-wrap;
  word-break: break-word;
  overflow: auto;
}

:global(.modal-content:has(.sp-edit-dialog)),
:global(.modal-content:has(.sp-run-detail)) {
  --sp-panel: #ffffff;
  --sp-panel-2: #f8fafc;
  --sp-panel-3: #eef2f7;
  --sp-line: #d7e0ea;
  --sp-soft: #e8eef5;
  --sp-text: #172033;
  --sp-muted: #607089;
  --sp-cyan: #0284c7;
  --sp-green: #16835d;
  --sp-amber: #c56a0a;
  --sp-orange: #dd5f16;
  --sp-red: #d14343;
  --sp-blue: #2563eb;
  --sp-violet: #6d5bd0;
  --sp-result-blue-soft: #eaf2ff;
  --sp-result-cyan-soft: #e6f6fb;
  --sp-result-green-soft: #e8f7ef;
  --sp-result-amber-soft: #fff3dc;
  --sp-result-red-soft: #fff0f0;
  --sp-result-violet-soft: #f1efff;
  --sp-result-neutral-soft: #f3f6fa;
  overflow: hidden;
  border-color: #cbd7e5;
  background: var(--sp-panel);
  color: var(--sp-text);
}

:global(.dark .modal-content:has(.sp-edit-dialog)),
:global(.dark .modal-content:has(.sp-run-detail)) {
  --sp-panel: #172033;
  --sp-panel-2: #1d293d;
  --sp-panel-3: #243249;
  --sp-line: #35445c;
  --sp-soft: #2c3a51;
  --sp-text: #edf3fb;
  --sp-muted: #a8b6ca;
  --sp-result-blue-soft: #1b3155;
  --sp-result-cyan-soft: #153947;
  --sp-result-green-soft: #173a31;
  --sp-result-amber-soft: #432f1d;
  --sp-result-red-soft: #48272d;
  --sp-result-violet-soft: #302b51;
  --sp-result-neutral-soft: #202d42;
  border-color: #3b4b64;
}

:global(.modal-content:has(.sp-edit-dialog) .modal-header),
:global(.modal-content:has(.sp-run-detail) .modal-header) {
  border-bottom-color: var(--sp-line);
  background: var(--sp-panel);
}

:global(.modal-content:has(.sp-edit-dialog) .modal-title),
:global(.modal-content:has(.sp-run-detail) .modal-title) {
  color: var(--sp-text);
}

:global(.modal-content:has(.sp-edit-dialog) .modal-body),
:global(.modal-content:has(.sp-run-detail) .modal-body) {
  display: flex;
  min-height: 0;
  flex-direction: column;
  overflow: hidden;
  background: var(--sp-panel);
}

:global(.modal-content:has(.sp-edit-dialog) .modal-footer),
:global(.modal-content:has(.sp-run-detail) .modal-footer) {
  border-top-color: var(--sp-line);
  background: var(--sp-panel);
}

.sp-run-detail {
  --sp-result-accent: var(--sp-cyan);
  display: grid;
  max-width: min(1120px, 86vw);
  max-height: 72vh;
  overflow: auto;
  border: 0;
  background: transparent;
  box-shadow: none;
  padding: 4px 2px 12px;
}

.sp-detail-outcome,
.sp-detail-content {
  display: grid;
  gap: 14px;
}

.sp-detail-content {
  border-top: 1px solid var(--sp-line);
  margin-top: 16px;
  padding-top: 18px;
}

.sp-detail-section-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 16px;
}

.sp-detail-section-kicker {
  display: block;
  color: var(--sp-result-accent);
  font-size: 10px;
  font-weight: 800;
  letter-spacing: 0.12em;
  line-height: 1.2;
  text-transform: uppercase;
}

.sp-detail-section-head h3 {
  margin: 3px 0 0;
  color: var(--sp-text);
  font-size: 16px;
  line-height: 1.25;
}

.sp-detail-content > .sp-provider-detail-layout,
.sp-detail-content > .sp-cleanup-grid,
.sp-detail-content > .sp-rate-guard-detail {
  margin-top: 0;
}
.sp-run-detail.good {
  --sp-result-accent: var(--sp-green);
}

.sp-run-detail.warn {
  --sp-result-accent: var(--sp-amber);
}

.sp-run-detail.bad {
  --sp-result-accent: var(--sp-red);
}

.sp-run-detail .sp-status {
  border-width: 1px;
  border-style: solid;
  font-weight: 700;
}

.sp-run-detail .sp-status.good {
  border-color: color-mix(in srgb, var(--sp-green) 38%, var(--sp-line));
  background: var(--sp-result-green-soft);
  color: var(--sp-green);
}

.sp-run-detail .sp-status.warn {
  border-color: color-mix(in srgb, var(--sp-amber) 42%, var(--sp-line));
  background: var(--sp-result-amber-soft);
  color: var(--sp-amber);
}

.sp-run-detail .sp-status.bad {
  border-color: color-mix(in srgb, var(--sp-red) 44%, var(--sp-line));
  background: var(--sp-result-red-soft);
  color: var(--sp-red);
}

.sp-run-detail-summary {
  display: grid;
  grid-template-columns: repeat(3, minmax(0, 1fr));
  row-gap: 18px;
  border-bottom: 1px solid var(--sp-line);
  padding: 4px 0 18px;
}

.sp-summary-item {
  min-width: 0;
  border-left: 1px solid var(--sp-line);
  padding: 2px 18px 4px;
}

.sp-summary-item:nth-child(3n + 1) {
  border-left: 0;
  padding-left: 0;
}

.sp-summary-task {
  --sp-summary-accent: var(--sp-blue);
}

.sp-summary-trigger {
  --sp-summary-accent: var(--sp-violet);
}

.sp-summary-status {
  --sp-summary-accent: var(--sp-amber);
}

.sp-summary-status.good {
  --sp-summary-accent: var(--sp-green);
}

.sp-summary-status.warn {
  --sp-summary-accent: var(--sp-amber);
}

.sp-summary-status.bad {
  --sp-summary-accent: var(--sp-red);
}

.sp-summary-counts {
  --sp-summary-accent: var(--sp-cyan);
}

.sp-summary-start {
  --sp-summary-accent: var(--sp-green);
}

.sp-summary-end {
  --sp-summary-accent: var(--sp-amber);
}

.sp-run-detail-summary strong {
  color: color-mix(in srgb, var(--sp-summary-accent, var(--sp-text)) 38%, var(--sp-text));
  font-weight: 800;
}

.sp-detail-label {
  display: block;
  margin-bottom: 5px;
  color: var(--sp-muted);
  font-size: 12px;
  font-weight: 700;
  letter-spacing: 0;
}

.sp-run-message,
.sp-provider-message {
  margin: 0;
  border: 0;
  border-left: 3px solid var(--sp-result-accent);
  color: var(--sp-text);
  background: color-mix(in srgb, var(--sp-result-accent, var(--sp-cyan)) 7%, var(--sp-panel));
  border-radius: 0;
  padding: 10px 12px;
  line-height: 1.65;
}

.sp-provider-message {
  margin: 0;
}

.sp-provider-list {
  display: grid;
  gap: 18px;
}

.sp-provider-detail-layout {
  display: grid;
  grid-template-columns: minmax(220px, 0.36fr) minmax(0, 1fr);
  gap: 16px;
  min-height: 420px;
  margin-top: 18px;
}

.sp-provider-index {
  display: grid;
  align-content: start;
  gap: 0;
  max-height: min(58vh, 620px);
  overflow: auto;
  border: 0;
  border-right: 1px solid var(--sp-line);
  border-radius: 0;
  background: transparent;
  padding: 0 16px 0 0;
}

.sp-provider-index-item {
  display: grid;
  grid-template-columns: minmax(0, 1fr) auto;
  gap: 6px 10px;
  width: 100%;
  border: 0;
  border-bottom: 1px solid var(--sp-line);
  border-left: 3px solid transparent;
  border-radius: 0;
  background: transparent;
  padding: 12px 10px;
  text-align: left;
  cursor: pointer;
  transition: border-left-color 0.16s ease, background-color 0.16s ease;
}

.sp-provider-index-item:hover,
.sp-provider-index-item.active {
  border-left-color: var(--sp-blue);
  background: var(--sp-result-blue-soft);
}

.sp-provider-index-item.bad:not(.active) {
  border-left-color: var(--sp-red);
  background: var(--sp-result-red-soft);
}

.sp-provider-index-item.warn:not(.active) {
  border-left-color: var(--sp-amber);
  background: var(--sp-result-amber-soft);
}

.sp-provider-index-name {
  min-width: 0;
  overflow: hidden;
  color: var(--sp-text);
  font-size: 13px;
  font-weight: 700;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.sp-provider-index-meta {
  grid-column: 1 / -1;
  color: var(--sp-muted);
  font-size: 12px;
  font-weight: 600;
}

.sp-provider-card {
  display: grid;
  gap: 16px;
  border: 0;
  border-radius: 0;
  background: transparent;
  box-shadow: none;
  padding: 0 0 0 4px;
}

.sp-provider-detail-card {
  --sp-result-accent: var(--sp-cyan);
  align-content: start;
  min-width: 0;
  max-height: min(58vh, 620px);
  overflow: auto;
}

.sp-provider-detail-card.good {
  --sp-result-accent: var(--sp-green);
}

.sp-provider-detail-card.warn {
  --sp-result-accent: var(--sp-amber);
}

.sp-provider-detail-card.bad {
  --sp-result-accent: var(--sp-red);
}

.sp-provider-head,
.sp-stage-head {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 12px;
}

.sp-provider-head {
  border-bottom: 1px solid var(--sp-line);
  padding: 0 0 16px;
}

.sp-provider-head h3,
.sp-stage-head strong {
  margin: 0;
  color: color-mix(in srgb, var(--sp-result-accent, var(--sp-cyan)) 34%, var(--sp-text));
  font-size: 16px;
  font-weight: 800;
}

.sp-tag {
  --sp-tag-accent: var(--sp-blue);
  --sp-tag-surface: var(--sp-result-blue-soft);
  display: inline-flex;
  align-items: center;
  gap: 5px;
  border: 1px solid color-mix(in srgb, var(--sp-tag-accent) 34%, var(--sp-line));
  border-radius: 999px;
  background: var(--sp-tag-surface);
  color: color-mix(in srgb, var(--sp-tag-accent) 72%, var(--sp-text));
  padding: 4px 9px;
}

.sp-tag.success {
  --sp-tag-accent: var(--sp-green);
  --sp-tag-surface: var(--sp-result-green-soft);
}

.sp-tag.primary,
.sp-tag.http {
  --sp-tag-accent: var(--sp-blue);
  --sp-tag-surface: var(--sp-result-blue-soft);
}

.sp-tag.warning,
.sp-tag.timing {
  --sp-tag-accent: var(--sp-amber);
  --sp-tag-surface: var(--sp-result-amber-soft);
}

.sp-tag.neutral {
  --sp-tag-accent: var(--sp-muted);
  --sp-tag-surface: var(--sp-result-neutral-soft);
}

.sp-provider-stats,
.sp-stage-metrics {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
  margin-top: 0;
}

.sp-provider-stats .sp-tag,
.sp-stage-metrics .sp-tag {
  font-size: 12px;
  font-weight: 700;
}

.sp-stage-groups {
  display: grid;
  gap: 0;
  margin-top: 0;
}

.sp-stage-category {
  border-top: 1px solid var(--sp-line);
  padding: 18px 0 0;
}

.sp-stage-category:first-child {
  border-top: 0;
  padding-top: 0;
}

.sp-stage-category h4 {
  display: flex;
  align-items: center;
  gap: 8px;
  margin: 0;
  border: 0;
  border-radius: 0;
  background: transparent;
  color: var(--sp-text);
  font-size: 13px;
  font-weight: 800;
  padding: 0 0 12px;
}

.sp-stage-category h4::before {
  content: "";
  width: 7px;
  height: 7px;
  border-radius: 999px;
  background: var(--sp-stage-accent, var(--sp-cyan));
}

.sp-stage-card {
  display: grid;
  gap: 14px;
  border: 0;
  border-top: 1px solid var(--sp-line);
  border-radius: 0;
  background: transparent;
  padding: 16px 0;
}

.sp-stage-card + .sp-stage-card {
  border-top-color: color-mix(in srgb, var(--sp-line) 78%, transparent);
}

.sp-stage-category.identity {
  --sp-stage-accent: var(--sp-blue);
}

.sp-stage-category.metrics {
  --sp-stage-accent: var(--sp-amber);
}

.sp-stage-category.other {
  --sp-stage-accent: var(--sp-violet);
}

.sp-stage-card.good {
  --sp-stage-accent: var(--sp-green);
}

.sp-stage-card.warn {
  --sp-stage-accent: var(--sp-amber);
}

.sp-stage-card.bad {
  --sp-stage-accent: var(--sp-red);
  border-left: 3px solid var(--sp-red);
}

.sp-stage-body {
  display: grid;
  grid-template-columns: minmax(0, 1fr) minmax(280px, 0.8fr);
  gap: 16px;
  align-items: stretch;
}

.sp-stage-main {
  display: grid;
  align-content: start;
  gap: 9px;
}

.sp-stage-row {
  display: grid;
  grid-template-columns: 76px minmax(0, 1fr);
  gap: 10px;
  border: 1px solid transparent;
  border-radius: 8px;
  color: var(--sp-text);
  font-size: 12px;
  line-height: 1.55;
  padding: 5px 0;
}

.sp-stage-row em {
  color: var(--sp-muted);
  font-style: normal;
  font-weight: 700;
}

.sp-stage-row span {
  min-width: 0;
  word-break: break-word;
}

.sp-stage-row.bad {
  border-color: transparent;
  border-left: 3px solid var(--sp-red);
  border-radius: 0;
}

.sp-stage-row.bad em {
  color: var(--sp-red);
}

.sp-stage-row.bad span {
  color: var(--sp-red);
  font-weight: 700;
}

.sp-response-panel {
  display: grid;
  grid-template-rows: auto minmax(0, 1fr);
  min-width: 0;
  border: 0;
  border-left: 1px solid var(--sp-line);
  border-radius: 0;
  background: transparent;
  padding-left: 16px;
}

.sp-response-panel-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 10px;
  border: 0;
  background: transparent;
  padding: 0 0 8px;
}

.sp-response-panel-head span {
  color: var(--sp-text);
  font-size: 12px;
  font-weight: 700;
}

.sp-response-panel-head small {
  color: var(--sp-muted);
  font-size: 11px;
}

.sp-response-summary {
  min-height: 130px;
  max-height: 240px;
  margin: 0;
  overflow: auto;
  white-space: pre-wrap;
  word-break: break-word;
  border: 0;
  border-radius: 4px;
  background: var(--sp-panel-2);
  color: var(--sp-text);
  padding: 10px;
  font-size: 12px;
  line-height: 1.6;
}

.sp-cleanup-grid {
  display: grid;
  grid-template-columns: repeat(3, minmax(0, 1fr));
  gap: 0;
  border-top: 1px solid var(--sp-line);
  border-bottom: 1px solid var(--sp-line);
  margin-top: 16px;
}

.sp-cleanup-grid > article {
  border-left: 1px solid var(--sp-line);
  padding: 14px 16px;
}

.sp-cleanup-grid > article:nth-child(3n + 1) {
  border-left: 0;
}

.sp-cleanup-grid > article:nth-child(n + 4) {
  border-top: 1px solid var(--sp-line);
}

.sp-cleanup-grid span {
  display: block;
  color: var(--sp-muted);
  font-size: 12px;
}

.sp-cleanup-grid strong {
  display: block;
  margin-top: 6px;
  color: var(--sp-text);
  font-size: 20px;
}

.sp-rate-guard-detail {
  display: grid;
  gap: 16px;
  margin-top: 16px;
}

.sp-rate-guard-summary {
  display: grid;
  grid-template-columns: repeat(7, minmax(0, 1fr));
  border-top: 1px solid var(--sp-line);
  border-bottom: 1px solid var(--sp-line);
}

.sp-rate-guard-summary > div {
  min-width: 0;
  border-left: 1px solid var(--sp-line);
  padding: 12px;
}

.sp-rate-guard-summary > div:first-child {
  border-left: 0;
}

.sp-rate-guard-summary span,
.sp-rate-guard-row small {
  display: block;
  color: var(--sp-muted);
  font-size: 11px;
}

.sp-rate-guard-summary strong {
  display: block;
  margin-top: 5px;
  color: var(--sp-text);
  font-size: 18px;
}

.sp-rate-guard-alerts,
.sp-rate-guard-changes,
.sp-rate-guard-inspections {
  min-width: 0;
}

.sp-rate-guard-inspections {
  border-top: 1px solid var(--sp-line);
  padding-top: 16px;
}

.sp-rate-guard-section-head {
  display: flex;
  align-items: flex-end;
  justify-content: space-between;
  gap: 12px;
  padding-bottom: 10px;
}

.sp-rate-guard-section-head span {
  display: block;
  color: var(--sp-muted);
  font-size: 10px;
  font-weight: 800;
  letter-spacing: 0.08em;
  line-height: 1.2;
  text-transform: uppercase;
}

.sp-rate-guard-section-head h4 {
  margin: 3px 0 0;
  color: var(--sp-text);
  font-size: 14px;
  line-height: 1.3;
}

.sp-rate-guard-section-head > strong {
  color: var(--sp-muted);
  font-size: 12px;
}

.sp-rate-guard-items {
  min-width: 0;
  overflow-x: auto;
  border-top: 1px solid var(--sp-line);
}

.sp-rate-guard-change-items {
  border-left: 3px solid var(--sp-green);
  background: color-mix(in srgb, var(--sp-green) 4%, transparent);
}

.sp-rate-guard-alert-items {
  border-left: 3px solid var(--sp-amber);
  background: color-mix(in srgb, var(--sp-amber) 5%, transparent);
}

.sp-rate-guard-alert-items .sp-rate-guard-row:not(.sp-rate-guard-row-head) > span:nth-child(5) {
  color: var(--sp-amber);
  font-weight: 700;
}

.sp-rate-guard-change-items .sp-rate-guard-row:not(.sp-rate-guard-row-head) > span:nth-child(3),
.sp-rate-guard-change-items .sp-rate-guard-row:not(.sp-rate-guard-row-head) > span:nth-child(4),
.sp-rate-guard-change-items .sp-rate-guard-row:not(.sp-rate-guard-row-head) > span:nth-child(5) {
  color: var(--sp-green);
  font-weight: 700;
}

.sp-rate-guard-row {
  display: grid;
  grid-template-columns: minmax(180px, 1.3fr) minmax(160px, 1fr) 90px 90px minmax(130px, 0.8fr) 150px;
  min-width: 880px;
  border-bottom: 1px solid var(--sp-line);
}

.sp-rate-guard-row > span {
  min-width: 0;
  padding: 11px 12px;
  color: var(--sp-text);
  font-size: 12px;
  word-break: break-word;
}

.sp-rate-guard-row strong {
  display: block;
  font-size: 12px;
}

.sp-rate-guard-row small {
  margin-top: 3px;
}

.sp-rate-guard-row-head > span {
  color: var(--sp-muted);
  font-size: 11px;
  font-weight: 700;
}

.sp-rate-guard-empty {
  border-top: 1px solid var(--sp-line);
  border-bottom: 1px solid var(--sp-line);
  color: var(--sp-muted);
  padding: 16px 0;
}

@media (prefers-reduced-motion: reduce) {
  .sp-overview-item {
    transition: none;
  }

  .sp-overview-item:hover {
    transform: none;
  }
}
@media (max-width: 1024px) {
  .sp-overview-strip {
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }



  .sp-retention-grid,
  .sp-run-detail-summary {
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }

  .sp-summary-item,
  .sp-summary-item:nth-child(3n + 1) {
    border-left: 1px solid var(--sp-line);
    padding-left: 18px;
  }

  .sp-summary-item:nth-child(odd) {
    border-left: 0;
    padding-left: 0;
  }
}

@media (max-width: 760px) {
  .sp-console-head,
  .sp-panel-head,
  .sp-detail-section-head {
    align-items: stretch;
    flex-direction: column;
  }

  .sp-head-actions,
  .sp-run-filters {
    width: 100%;
  }

  .sp-head-actions {
    justify-content: space-between;
  }

  .sp-head-actions .sp-refresh-meta {
    flex: 1 1 150px;
  }

  .sp-head-actions .sp-button {
    min-width: 96px;
  }

  .sp-run-filters {
    grid-template-columns: 1fr;
  }

  .sp-run-filters .sp-button {
    width: 100%;
  }

  .sp-overview-strip,
  .sp-edit-summary,
  .sp-form-grid,
  .sp-retention-grid,
  .sp-run-detail-summary,
  .sp-provider-detail-layout,
  .sp-cleanup-grid,
  .sp-rate-guard-summary,
  .sp-stage-body {
    grid-template-columns: 1fr;
  }



  .sp-panel-head {
    gap: 12px;
    padding: 14px;
  }

  .sp-panel-signals {
    justify-content: flex-start;
    width: 100%;
  }

  .sp-history-toolbar {
    padding: 12px 14px;
  }


  .sp-edit-summary > div,
  .sp-edit-summary > div:first-child {
    border-top: 1px solid var(--sp-soft);
    border-left: 0;
  }

  .sp-edit-summary > div:first-child {
    border-top: 0;
  }

  .sp-provider-index,
  .sp-provider-detail-card {
    max-height: none;
  }

  .sp-run-detail {
    max-width: 100%;
  }

  .sp-run-detail-summary {
    row-gap: 0;
  }

  .sp-summary-item,
  .sp-summary-item:nth-child(3n + 1) {
    border-top: 1px solid var(--sp-line);
    border-left: 0;
    padding: 12px 0;
  }

  .sp-summary-item:first-child {
    border-top: 0;
  }

  .sp-provider-index {
    border-right: 0;
    border-bottom: 1px solid var(--sp-line);
    padding: 0 0 12px;
  }

  .sp-provider-card {
    padding: 4px 0 0;
  }

  .sp-response-panel {
    border-top: 1px solid var(--sp-line);
    border-left: 0;
    padding-top: 14px;
    padding-left: 0;
  }

  .sp-cleanup-grid > article,
  .sp-cleanup-grid > article:nth-child(3n + 1) {
    border-top: 1px solid var(--sp-line);
    border-left: 0;
    padding-right: 0;
    padding-left: 0;
  }

  .sp-cleanup-grid > article:first-child {
    border-top: 0;
  }

  .sp-rate-guard-summary > div,
  .sp-rate-guard-summary > div:first-child {
    border-top: 1px solid var(--sp-line);
    border-left: 0;
  }

  .sp-rate-guard-summary > div:first-child {
    border-top: 0;
  }
}

@media (max-width: 767px) {
  .sp-table-region {
    padding: 12px;
  }

  .sp-task-actions {
    width: 100%;
  }

  .sp-task-actions .sp-button {
    flex: 1 1 0;
    min-width: 0;
    min-height: 40px;
  }
}

:global(.dark .modal-content:has(.sp-edit-dialog) .sp-form-section-head > span) {
  color: color-mix(in srgb, var(--sp-blue) 52%, white);
}</style>

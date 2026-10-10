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
                <div class="sp-inline gap-2">
                  <Toggle
                    :model-value="task.enabled"
                    :disabled="updatingTaskIDs.has(task.task_code)"
                    :aria-label="`切换${task.name}任务状态`"
                    @update:model-value="toggleTaskStatus(task.task_code, $event)"
                  />
                  <span class="sp-status" :class="task.enabled ? 'good' : ''">{{ task.enabled ? '已启用' : '已停用' }}</span>
                </div>
              </template>
              <template #cell-cron_expression="{ row: task }">
                <span class="sp-status info">{{ formatInterval(task.cron_expression) }}</span>
                <div class="sp-sub">{{ task.cron_expression }}</div>
              </template>

              <template #cell-last_run_at="{ row: task }">
                {{ formatTime(task.last_run_at) }}
              </template>
              <template #cell-last_status="{ row: task }">
                <!-- 状态标签兼作最近一次任务的详情入口，与「详情」列同一动作、同一可用条件：
                     没有历史结果时不可点，避免打开一个只写着“暂无结果”的空弹窗。 -->
                <button
                  class="sp-status sp-status-action"
                  :class="statusTone(task.last_status)"
                  type="button"
                  :disabled="!(task.last_message || latestRunByTask[task.task_code])"
                  @click.stop="openTaskLatestResult(task)"
                >
                  {{ statusText(task.last_status) }}
                </button>
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
            <div class="sp-history-head-actions">
              <div class="sp-history-count">{{ runTotal }} 条记录</div>
              <button
                class="sp-button small ghost sp-election-log-entry"
                type="button"
                data-test="open-election-change-logs"
                @click="openElectionChangeLogs"
              >
                调度切换日志
              </button>
              <!-- 与「调度切换日志」同色：同属择优调度这一族，不新增色板。
                   区别在用途——那边看「谁被拨动了」，这边看「现在到底开着几个」。 -->
              <button
                class="sp-button small ghost sp-election-log-entry"
                type="button"
                data-test="open-election-diagnostics"
                @click="openElectionDiagnostics"
              >
                诊断快照
              </button>
            </div>
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
        <form class="sp-edit-dialog" :class="{ 'is-health-guard': editForm.task_code === 'supplier_account_health_guard' }" :data-grid-cols="editDialogGridCols" @submit.prevent="saveTask">
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
            <div v-if="editForm.task_code === 'supplier_provider_recharge_sync'" class="sp-form-note">
              供应商充值记录同步采用增量同步，默认每 30 分钟执行一次；可在此处调整周期。
            </div>
          </section>

          <section v-if="editForm.task_code === 'supplier_rate_guard'" class="sp-form-section sp-policy-section">
            <div class="sp-form-section-head">
              <span>03</span>
              <div><h3>倍率守护策略</h3><p>控制允许参与倍率比较的上游快照有效期。</p></div>
            </div>
            <div class="sp-form-grid">
              <Input :model-value="editForm.config.rate_guard_max_snapshot_age_seconds" type="number" label="快照最大有效期（秒）" @update:model-value="editForm.config.rate_guard_max_snapshot_age_seconds = toNumber($event, editForm.config.rate_guard_max_snapshot_age_seconds)" />
            </div>
          </section>

          <section v-if="editForm.task_code === 'supplier_account_rate_guard'" class="sp-form-section sp-policy-section">
            <div class="sp-form-section-head">
              <span>03</span>
              <div><h3>账号倍率守护策略</h3><p>控制哪些本地分组参与守护；关闭的分组在检测与执行时都会被跳过。</p></div>
            </div>
            <div class="sp-rate-guard-scope-card">
              <div>
                <strong>不参与守护的分组</strong>
                <span v-if="accountRateGuardDisabledGroupIDs.length === 0">所有分组都参与守护。新增分组也会自动参与。</span>
                <span v-else>已关闭 <strong class="sp-rate-guard-scope-count">{{ accountRateGuardDisabledGroupIDs.length }}</strong> 个分组，其余分组正常参与守护。</span>
              </div>
              <button
                class="sp-button small ghost sp-rate-guard-scope-config-button"
                type="button"
                @click="openRateGuardGroups"
              >
                <Icon name="cog" size="md" />
                配置参与分组
              </button>
            </div>
          </section>

          <section v-if="editForm.task_code === 'supplier_account_health_guard'" class="sp-form-section sp-policy-section">
            <div class="sp-form-section-head">
              <span>03</span>
              <div><h3>健康守护策略</h3><p>控制每轮检测规模、单账号测试压力以及暂停和恢复调度的连续次数。</p></div>
            </div>
            <div class="sp-health-guard-account-card">
              <div>
                <strong>需要检查的账号</strong>
                <span>已选择 <strong class="sp-health-guard-card-count">{{ healthGuardAccountIDs.length }}</strong> 个本地账号，可按平台设置默认测试模型并为账号单独覆盖。</span>
              </div>
              <button
                class="sp-button small ghost sp-health-guard-config-button"
                type="button"
                @click="openHealthGuardAccounts"
              >
                <Icon name="cog" size="md" />
                配置检查账号
              </button>
            </div>
            <div class="sp-form-grid sp-health-guard-policy-grid">
              <Input :model-value="editForm.config.account_health_guard_max_accounts_per_run" type="number" label="单次检查账号数" @update:model-value="editForm.config.account_health_guard_max_accounts_per_run = toNumber($event, editForm.config.account_health_guard_max_accounts_per_run)" />
              <Input :model-value="editForm.config.account_health_guard_concurrency" type="number" label="并发数" @update:model-value="editForm.config.account_health_guard_concurrency = toNumber($event, editForm.config.account_health_guard_concurrency)" />
              <Input :model-value="editForm.config.account_health_guard_timeout_per_account_seconds" type="number" label="单账号超时（秒）" @update:model-value="editForm.config.account_health_guard_timeout_per_account_seconds = toNumber($event, editForm.config.account_health_guard_timeout_per_account_seconds)" />
              <Input :model-value="editForm.config.account_health_guard_failure_threshold" type="number" label="连续失败暂停阈值" @update:model-value="editForm.config.account_health_guard_failure_threshold = toNumber($event, editForm.config.account_health_guard_failure_threshold)" />
              <Input :model-value="editForm.config.account_health_guard_slow_threshold" type="number" label="连续慢响应暂停阈值" @update:model-value="editForm.config.account_health_guard_slow_threshold = toNumber($event, editForm.config.account_health_guard_slow_threshold)" />
              <Input :model-value="editForm.config.account_health_guard_recovery_threshold" type="number" label="连续健康恢复阈值" @update:model-value="editForm.config.account_health_guard_recovery_threshold = toNumber($event, editForm.config.account_health_guard_recovery_threshold)" />
              <Input :model-value="editForm.config.account_health_guard_healthy_latency_ms" type="number" label="默认健康延迟（毫秒）" @update:model-value="editForm.config.account_health_guard_healthy_latency_ms = toNumber($event, editForm.config.account_health_guard_healthy_latency_ms)" />
            </div>
            <div class="sp-health-guard-multiplier-switch">
              <div>
                <strong>修改调度（全局）</strong>
                <span v-if="editForm.config.account_health_guard_scheduling_change_enabled">
                  已开启：所有检查账号在连续失败/慢响应达到阈值后自动暂停调度，恢复达标后自动恢复；可在「配置检查账号」里为个别账号单独关闭。
                </span>
                <span v-else>已关闭：所有账号只做健康检测，不会自动暂停或恢复调度；可在「配置检查账号」里为个别账号单独开启。</span>
              </div>
              <div class="sp-toggle-row">
                <Toggle v-model="editForm.config.account_health_guard_scheduling_change_enabled" />
                <em>{{ editForm.config.account_health_guard_scheduling_change_enabled ? '已启用' : '已停用' }}</em>
              </div>
            </div>
            <div class="sp-health-guard-multiplier-switch">
              <div>
                <strong>未开调度账号按倍率间隔测试</strong>
                <span v-if="editForm.config.account_health_guard_platform_multiplier_intervals_enabled">
                  已开启：未开调度的账号改由所属平台的倍率区间决定检查间隔（覆盖账号级间隔），可在「配置检查账号」里为各平台设置区间。
                </span>
                <span v-else>关闭时未开调度账号沿用账号级检查间隔；已开调度账号始终不受此规则影响。</span>
              </div>
              <div class="sp-toggle-row">
                <Toggle v-model="editForm.config.account_health_guard_platform_multiplier_intervals_enabled" />
                <em>{{ editForm.config.account_health_guard_platform_multiplier_intervals_enabled ? '已启用' : '已停用' }}</em>
              </div>
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

          <section v-if="editForm.task_code === 'supplier_group_scheduling_election'" class="sp-form-section sp-policy-section">
            <div class="sp-form-section-head">
              <span>03</span>
              <div><h3>分组择优调度策略</h3><p>为每个分组按综合分选出最优账号开启调度：综合分 = 次数权重 × 连续成功次数分 + 用时权重 × 组内相对用时分，两项都先归一到 0~1 再加权，所以两个权重的比值就是它们的话语权之比。测试失败的账号要连续失败达到阈值才关闭，且分组里没有备选账号时一律保留。本任务只读现有测试状态、不重新测试，建议排在健康守护之后运行。</p></div>
            </div>
            <div class="sp-election-dry-run-card" :class="{ 'is-on': electionDryRunAll }">
              <div>
                <strong>演练模式</strong>
                <span v-if="electionDryRunAll">本任务只计算并记下「该开谁、该关谁」，不修改任何账号的调度开关；切换日志里这些条目会标成「建议」。</span>
                <span v-else>按择优结果真实修改调度。改完权重或阈值想先看看它会切谁，就打开演练——计算与日志照常，只是不落地。</span>
              </div>
              <label
                class="sp-election-dry-run-toggle"
                title="打开后本任务进入演练：只出建议、不改调度"
              >
                <Toggle
                  :model-value="electionDryRunAll"
                  aria-label="是否开启演练模式（只记录建议、不修改调度）"
                  @update:model-value="setElectionDryRunAll"
                />
                <span>{{ electionDryRunAll ? '演练中' : '正式执行' }}</span>
              </label>
            </div>
            <div class="sp-election-dry-run-card sp-election-keep-healthy-card" :class="{ 'is-on': electionKeepHealthyAll }">
              <div>
                <strong>在任者健康锁定（全局默认）</strong>
                <span v-if="electionKeepHealthyAll">默认对所有分组开启：开着调度的账号测试都正常、且没有开着却失败的，就保留现状、跳过择优与换人。可在「配置分组」里对个别分组单独关闭。</span>
                <span v-else>默认按择优结果照常换人。打开后所有分组默认锁定健康在任者、压住无谓抖动；仍可在「配置分组」里对个别分组单独开或关。</span>
              </div>
              <label
                class="sp-election-dry-run-toggle"
                title="打开后：默认所有分组在开着的账号都正常时保留现状、不换人；可在配置分组里逐个覆盖"
              >
                <Toggle
                  :model-value="electionKeepHealthyAll"
                  aria-label="是否全局开启在任者健康锁定"
                  @update:model-value="setElectionKeepHealthyAll"
                />
                <span>{{ electionKeepHealthyAll ? '默认锁定' : '默认不锁' }}</span>
              </label>
            </div>
            <div class="sp-election-dry-run-card sp-election-keep-healthy-card" :class="{ 'is-on': electionPriorityAll }">
              <div>
                <strong>账号优先级参与择优（全局默认）</strong>
                <span v-if="electionPriorityAll">默认对所有分组开启：综合分里计入账号优先级（数值越小优先级越高），都健康时高优先级账号更占优势。可在「配置分组」里对个别分组单独关闭。</span>
                <span v-else>默认不把优先级计入综合分（与升级前一致）。打开后所有分组默认计优先级；仍可在「配置分组」里对个别分组单独开或关。</span>
              </div>
              <label
                class="sp-election-dry-run-toggle"
                title="打开后：默认所有分组的综合分都计入账号优先级（数值越小优先级越高）；可在配置分组里逐个覆盖"
              >
                <Toggle
                  :model-value="electionPriorityAll"
                  aria-label="是否全局开启账号优先级计分"
                  @update:model-value="setElectionPriorityAll"
                />
                <span>{{ electionPriorityAll ? '默认计入' : '默认不计' }}</span>
              </label>
            </div>
            <div class="sp-election-dry-run-card sp-election-keep-healthy-card" :class="{ 'is-on': electionAlertEnabled }">
              <div>
                <strong>分组账号异常推送</strong>
                <span v-if="electionAlertEnabled">已开启：分组里当前开着调度的账号，若延迟窗口内的成功样本数不足「平均最少成功样本」，就按通知页订阅的「分组账号异常」事件推一条消息。只推异常、不推恢复，也不影响任何调度裁决。</span>
                <span v-else>默认不推送。打开后会在分组在任账号健康样本不足时推送消息，需先在通知页订阅「分组账号异常」事件。</span>
              </div>
              <label
                class="sp-election-dry-run-toggle"
                title="打开后：分组在任账号健康样本不足时推送消息（只推异常、不推恢复；需先在通知页订阅该事件）"
              >
                <Toggle
                  :model-value="electionAlertEnabled"
                  aria-label="是否开启分组账号异常推送"
                  @update:model-value="setElectionAlertEnabled"
                />
                <span>{{ electionAlertEnabled ? '推送中' : '不推送' }}</span>
              </label>
            </div>
            <div class="sp-form-grid sp-group-election-policy-grid">
              <Input :model-value="editForm.config.group_scheduling_election_top_n" type="number" label="每组开启账号数（默认 1 单活）" hint="每组最多保留几个账号处于开启调度状态，默认 1 = 单活。注意这只是「每组」的目标、不是硬上限：账号的开关记在账号上（不是记在「账号 + 分组」上），判定规则是「在所属的任一分组里最优就开」。所以同属 A、B 两个组的账号，只要它在 B 组里最优就会被开启，它落在 A 组里的那一份也跟着开着 —— A 组实际开着的数量就会超过这里填的值。要真正单活，只能让分组互不重叠、或让重叠的账号只在其中一个组里最优。填 2 以上时较慢的账号也会分摊到请求。" @update:model-value="editForm.config.group_scheduling_election_top_n = toNumber($event, editForm.config.group_scheduling_election_top_n ?? 1)" />
              <Input :model-value="editForm.config.group_scheduling_election_count_weight" type="number" step="0.1" min="0.1" label="连续成功次数权重（默认 1）" hint="连续成功次数在综合分里的话语权。次数分 = min(连续成功次数 ÷ 封顶值, 1)，即达到封顶值后一律按封顶值算——默认封顶 10 时，142 次与 190 次得分完全相同。封顶是为了防止老账号靠资历永久占位。" @update:model-value="editForm.config.group_scheduling_election_count_weight = toNumber($event, editForm.config.group_scheduling_election_count_weight ?? 1)" />
              <Input :model-value="editForm.config.group_scheduling_election_latency_weight" type="number" step="0.1" min="0.1" label="测试用时权重（默认 0.5）" hint="用时在综合分里的话语权。用时只在同一平台内比较（跨平台基线速度差数倍，比了没意义），组内最快的得 1 分、最慢的得 0 分。默认 0.5，即用时满分也只相当于次数满分的一半——封顶 10 时最快也只抵 5 次连续成功；调到比次数权重还大就变成谁快谁上。" @update:model-value="editForm.config.group_scheduling_election_latency_weight = toNumber($event, editForm.config.group_scheduling_election_latency_weight ?? 0.5)" />
              <Input :model-value="editForm.config.group_scheduling_election_priority_weight" type="number" step="0.1" min="0.1" label="账号优先级权重（默认 0.5，需在下方开关分组内开启）" hint="账号优先级在综合分里的话语权。优先级在同一分组内比较（数值越小优先级越高），组内优先级最高的得 1 分、最低的得 0 分；全都设置相同值则这项取中性分、不改变排序。默认 0.5，即全部健康时优先级最高者恰好拿满分。注意：只有在上方/「配置分组」里开启了「优先级计分」的分组，这项才真正计入综合分；想进一步放大「都健康时高优先级占优」，可调大此值或调小次数/用时权重。" @update:model-value="editForm.config.group_scheduling_election_priority_weight = toNumber($event, editForm.config.group_scheduling_election_priority_weight ?? 0.5)" />
              <Input :model-value="editForm.config.group_scheduling_election_failure_threshold" type="number" min="1" label="连续失败关闭阈值（默认 2 次）" hint="测试失败后要连续几个执行周期都失败，才真正关闭调度。默认 2 是给单次网络抖动留翻盘机会，填 1 等于失败一次就关。另有两种情况不关：分组里已无其它成功账号（关了就成空组），或失败次数还没到阈值——都保持原状等下一轮。" @update:model-value="editForm.config.group_scheduling_election_failure_threshold = toNumber($event, editForm.config.group_scheduling_election_failure_threshold ?? 2)" />
              <Input :model-value="editForm.config.group_scheduling_election_switch_margin" type="number" step="0.05" min="0" label="切换迟滞比例（默认 0.15，挑战者要快 15% 才换人）" hint="换人的死区，比较的是延迟而不是总分。挑战者的延迟要比当前正在跑的账号快出这个比例才夺位，否则维持现状，防止两个差不多的账号来回对拍。默认 0.15 = 快 15%。填到 0.95 会让在任者几乎永不被换下。" @update:model-value="editForm.config.group_scheduling_election_switch_margin = toNumber($event, editForm.config.group_scheduling_election_switch_margin ?? 0.15)" />
              <Input :model-value="editForm.config.group_scheduling_election_latency_window_minutes" type="number" min="1" label="延迟平均窗口（分钟，默认 30）" hint="综合分用的延迟取最近这么多分钟内健康检测成功样本的平均值，避免被单次抖动带偏。注意账号列表里显示的「近 1h 均」是另一个窗口，数值可能不同，判断依据以本任务的运行记录为准。" @update:model-value="editForm.config.group_scheduling_election_latency_window_minutes = toNumber($event, editForm.config.group_scheduling_election_latency_window_minutes ?? 30)" />
              <Input :model-value="editForm.config.group_scheduling_election_latency_min_samples" type="number" min="1" label="平均最少成功样本（默认 3，不足则回退单次）" hint="窗口内的成功样本少于这个数时不信任平均值，改回用最近一次测试的耗时。样本太少时均值会被一两次异常带偏，回退单次等于退回旧行为，不会更差。" @update:model-value="editForm.config.group_scheduling_election_latency_min_samples = toNumber($event, editForm.config.group_scheduling_election_latency_min_samples ?? 3)" />
              <Input :model-value="editForm.config.group_scheduling_election_count_score_cap" type="number" min="1" max="100" label="连续成功次数封顶（默认 10）" hint="次数分 = min(连续成功次数, 封顶值) / 封顶值，所以达到封顶值的账号得分完全相同——默认 10 时，连续成功 142 次和 190 次没有任何区别。调大它会让长期稳定的账号更难被更快的账号换掉；填 1 等于只看用时。上限 100。" @update:model-value="editForm.config.group_scheduling_election_count_score_cap = toNumber($event, editForm.config.group_scheduling_election_count_score_cap ?? 10)" />
            </div>
            <div class="sp-rate-guard-scope-card">
              <div>
                <strong>分组参与择优 / 在任者健康锁定 / 优先级计分</strong>
                <span v-if="groupElectionDisabledGroupIDs.length === 0">所有分组都参与择优。新增分组也会自动参与。</span>
                <span v-else>已关闭 <strong class="sp-rate-guard-scope-count">{{ groupElectionDisabledGroupIDs.length }}</strong> 个分组，其余分组正常参与择优。</span>
                <span v-if="electionKeepHealthyAll">已<strong class="sp-election-keep-healthy-count">全局</strong>开启「在任者健康锁定」：开着的账号正常就保留现状、不换人<template v-if="groupElectionKeepHealthyExcludedGroupIDs.length > 0">，其中 <strong class="sp-rate-guard-scope-count">{{ groupElectionKeepHealthyExcludedGroupIDs.length }}</strong> 个分组单独关闭</template>。</span>
                <span v-else-if="groupElectionKeepHealthyGroupIDs.length > 0">已对 <strong class="sp-rate-guard-scope-count">{{ groupElectionKeepHealthyGroupIDs.length }}</strong> 个分组开启「在任者健康则锁定」：开着的账号正常就跳过、不换人。</span>
                <span v-else>暂无分组开启「在任者健康锁定」。</span>
                <span v-if="electionPriorityAll">已<strong class="sp-election-keep-healthy-count">全局</strong>开启「优先级计分」：综合分计入账号优先级，都健康时高优先级更占优势<template v-if="groupElectionPriorityDisabledGroupIDs.length > 0">，其中 <strong class="sp-rate-guard-scope-count">{{ groupElectionPriorityDisabledGroupIDs.length }}</strong> 个分组单独关闭</template>。</span>
                <span v-else-if="groupElectionPriorityEnabledGroupIDs.length > 0">已对 <strong class="sp-rate-guard-scope-count">{{ groupElectionPriorityEnabledGroupIDs.length }}</strong> 个分组开启「优先级计分」：综合分计入账号优先级，都健康时高优先级更占优势。</span>
                <span v-if="groupElectionRequiredModelsCount > 0">已为 <strong class="sp-rate-guard-scope-count">{{ groupElectionRequiredModelsCount }}</strong> 个分组设置必需模型：换人时保证这些模型不断供，支持者全失败则告警待恢复。</span>
                <span v-if="groupElectionTopNOverridesCount > 0">已为 <strong class="sp-rate-guard-scope-count">{{ groupElectionTopNOverridesCount }}</strong> 个分组单独设置「每组开启账号数」，覆盖全局默认值。</span>
                <span v-if="groupElectionDefaultAccountCount > 0">已为 <strong class="sp-rate-guard-scope-count">{{ groupElectionDefaultAccountCount }}</strong> 个分组指定了「默认账号」：该账号健康时只开它一个、关闭其它成员。</span>
                <span v-if="electionAlertOverrideCount > 0">已为 <strong class="sp-rate-guard-scope-count">{{ electionAlertOverrideCount }}</strong> 个分组单独设置「异常推送」：<template v-if="electionAlertMutedCount > 0">{{ electionAlertMutedCount }} 个静音</template><template v-if="electionAlertMutedCount > 0 && electionAlertForcedCount > 0">、</template><template v-if="electionAlertForcedCount > 0">{{ electionAlertForcedCount }} 个强制推送</template>；覆盖与总开关不一致时以覆盖为准。</span>
                <span v-if="electionDryRunAll">演练模式已开启：<strong class="sp-election-dry-run-count">全部分组</strong>只记录建议，不修改调度。</span>
                <span v-else-if="groupElectionDryRunGroupIDs.length > 0">已对 <strong class="sp-election-dry-run-count">{{ groupElectionDryRunGroupIDs.length }}</strong> 个分组开启演练：这些分组只记录建议，不修改调度。</span>
              </div>
              <button
                class="sp-button small ghost sp-rate-guard-scope-config-button"
                type="button"
                @click="openElectionGroups"
              >
                <Icon name="cog" size="md" />
                配置分组
              </button>
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
                <h3>执行结论</h3>
              </div>
              <span class="sp-status" :class="statusTone(detailRun.status)">{{ statusText(detailRun.status) }}</span>
            </div>

            <section class="sp-run-detail-summary">
              <div class="sp-summary-item sp-summary-task">
                <span class="sp-detail-label">任务</span>
                <strong>{{ taskName(detailRun.task_code) }}</strong>
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

            <section v-else-if="detailRun.result_detail?.group_election && groupElectionResult" class="sp-rate-guard-detail">
              <div class="sp-rate-guard-summary sp-account-rate-guard-summary">
                <div><span>每组开启数</span><strong>{{ groupElectionResult.top_n }}</strong></div>
                <div><span>扫描分组</span><strong>{{ groupElectionResult.group_count }}</strong></div>
                <div><span>涉及账号</span><strong>{{ groupElectionResult.account_count }}</strong></div>
                <div><span>开启调度</span><strong>{{ groupElectionResult.enabled_count }}</strong></div>
                <div><span>关闭调度</span><strong>{{ groupElectionResult.disabled_count }}</strong></div>
                <div><span>保持不变</span><strong>{{ groupElectionResult.unchanged_count }}</strong></div>
                <div><span>跳过（未测）</span><strong>{{ groupElectionResult.skipped_count }}</strong></div>
                <div><span>更新失败</span><strong>{{ groupElectionResult.failed_write_count }}</strong></div>
                <div v-if="groupElectionResult.required_model_uncovered_count" class="sp-group-election-warn-stat"><span>必需模型断供</span><strong>{{ groupElectionResult.required_model_uncovered_count }}</strong></div>
                <!-- 演练的建议数单列：它和"开启/关闭"是两回事，混进去会让人以为调度真的被改了。 -->
                <template v-if="groupElectionResult.dry_run">
                  <div class="sp-group-election-suggest-stat"><span>建议开启（未生效）</span><strong>{{ groupElectionResult.suggested_enabled_count ?? 0 }}</strong></div>
                  <div class="sp-group-election-suggest-stat"><span>建议关闭（未生效）</span><strong>{{ groupElectionResult.suggested_disabled_count ?? 0 }}</strong></div>
                </template>
              </div>
              <div v-if="groupElectionResult.required_model_warnings?.length" class="sp-group-election-required-warns">
                <div class="sp-group-election-required-warns-title">必需模型当前无健康提供者，已切到能用账号，待人工恢复：</div>
                <div
                  v-for="(warn, index) in groupElectionResult.required_model_warnings"
                  :key="`${warn.group_id}-${warn.model}-${index}`"
                  class="sp-group-election-required-warn"
                >
                  <span class="sp-group-election-required-warn-group">{{ warn.group_name || `分组 ${warn.group_id}` }}</span>
                  <span class="sp-group-election-required-warn-model">{{ warn.model }}</span>
                  <small v-if="warn.failed_account_ids?.length">失败支持者账号：{{ warn.failed_account_ids.join('、') }}</small>
                  <small v-else>该分组无任何账号映射此模型</small>
                </div>
              </div>
              <div v-if="groupElectionResult.items.length" class="sp-rate-guard-table sp-group-election-table">
                <div class="sp-rate-guard-head sp-group-election-head">
                  <span>账号</span>
                  <span>测试状态</span>
                  <span>连续成功</span>
                  <span>测试用时</span>
                  <span>调度变更</span>
                  <span>原因</span>
                </div>
                <div v-for="item in groupElectionResult.items" :key="item.account_id" class="sp-rate-guard-row sp-group-election-row">
                  <span>
                    <strong>{{ item.account_name || `账号 ${item.account_id}` }}</strong>
                    <small v-if="item.platform">{{ item.platform }}</small>
                  </span>
                  <span>{{ groupElectionTestStatusText(item.test_status) }}</span>
                  <span>{{ item.healthy_count }}</span>
                  <span class="sp-group-election-latency" :class="{ 'is-empty': !item.latency_ms }">{{ groupElectionLatencyText(item.latency_ms) }}</span>
                  <span>
                    <strong>{{ groupElectionActionText(item.action, item.suggested) }}</strong>
                    <small>{{ item.schedulable_before ? '开' : '关' }} → {{ item.schedulable_after ? '开' : '关' }}</small>
                  </span>
                  <span>
                    <small>{{ item.reason }}</small>
                    <small v-if="item.error_message" class="sp-group-election-error">{{ item.error_message }}</small>
                  </span>
                </div>
              </div>
              <div v-else class="sp-rate-guard-empty">本次没有账号发生调度变更。</div>
            </section>

            <SupplierAccountHealthGuardResult
              v-else-if="detailRun.result_detail?.account_health_guard && accountHealthGuardResult"
              :key="detailRun.id"
              :result="accountHealthGuardResult"
            />

            <section v-else-if="detailRun.result_detail?.recharge_sync && rechargeSyncResult" class="sp-rate-guard-detail sp-recharge-sync-detail">
              <div class="sp-rate-guard-summary">
                <div><span>同步供应商</span><strong>{{ rechargeSyncResult.items.length }}</strong></div>
                <div><span>成功</span><strong>{{ rechargeSyncResult.success_count }}</strong></div>
                <div><span>失败</span><strong :class="{ 'sp-monitor-warning-value': rechargeSyncResult.failed_count > 0 }">{{ rechargeSyncResult.failed_count }}</strong></div>
                <div><span>同步记录数</span><strong>{{ rechargeSyncRecordCount }}</strong></div>
              </div>

              <section class="sp-monitor-items">
                <header class="sp-rate-guard-section-head">
                  <div>
                    <h4>供应商充值记录同步</h4>
                  </div>
                  <strong>{{ rechargeSyncResult.items.length }} 个供应商</strong>
                </header>
                <div v-if="rechargeSyncResult.items.length" class="sp-rate-guard-items sp-recharge-sync-items">
                  <div class="sp-monitor-row sp-monitor-row-head" aria-hidden="true">
                    <span>供应商</span>
                    <span>状态</span>
                    <span>同步记录数</span>
                    <span>同步时间</span>
                    <span>结果</span>
                  </div>
                  <div v-for="item in rechargeSyncResult.items" :key="`${item.provider_id}-${item.synced_at}`" class="sp-monitor-row">
                    <span><strong>{{ item.provider_name || `供应商 ${item.provider_id}` }}</strong><small>供应商 #{{ item.provider_id }}</small></span>
                    <span class="sp-status" :class="statusTone(item.status)">{{ statusText(item.status) }}</span>
                    <span>{{ item.record_count }}</span>
                    <span>{{ formatTime(item.synced_at) }}</span>
                    <span>{{ item.message || '-' }}</span>
                  </div>
                </div>
                <div v-else class="sp-rate-guard-empty">本次没有可同步的启用供应商。</div>
              </section>
            </section>

            <section v-else-if="detailRun.result_detail?.supplier_monitor && supplierMonitorResult" class="sp-rate-guard-detail sp-monitor-detail">
              <div class="sp-rate-guard-summary sp-monitor-summary">
                <div><span>监控项</span><strong>{{ supplierMonitorDisplayItems.length }}</strong></div>
                <div><span>已匹配本地账号</span><strong>{{ supplierMonitorMatchedAccountCount }}</strong></div>
                <div><span>未匹配本地账号</span><strong :class="{ 'sp-monitor-warning-value': supplierMonitorUnmatchedAccountCount > 0 }">{{ supplierMonitorUnmatchedAccountCount }}</strong></div>
                <div><span>已归属本地分组</span><strong>{{ supplierMonitorMatchedGroupCount }}</strong></div>
                <div><span>未归属本地分组</span><strong :class="{ 'sp-monitor-warning-value': supplierMonitorUngroupedCount > 0 }">{{ supplierMonitorUngroupedCount }}</strong></div>
                <div><span>健康 / 慢 / 失败</span><strong>{{ supplierMonitorHealthyCount }} / {{ supplierMonitorSlowCount }} / {{ supplierMonitorFailedCount }}</strong></div>
                <div><span>供应商成功</span><strong>{{ supplierMonitorResult.success_count }} / {{ supplierMonitorResult.processed_count }}</strong></div>
              </div>

              <section class="sp-monitor-items">
                <header class="sp-rate-guard-section-head">
                  <div>
                    <h4>监控项 / 本地账号 / 本地分组</h4>
                  </div>
                  <strong>{{ supplierMonitorDisplayItems.length }} 项</strong>
                </header>
                <div v-if="supplierMonitorDisplayItems.length" class="sp-rate-guard-items sp-monitor-items-table">
                  <div class="sp-monitor-row sp-monitor-row-head" aria-hidden="true">
                    <span>供应商 / 监控名称</span>
                    <span>本地账号</span>
                    <span>本地分组</span>
                    <span>状态</span>
                    <span>延迟</span>
                    <span>7 天可用率</span>
                    <span>检查时间</span>
                  </div>
                  <div v-for="item in supplierMonitorDisplayItems" :key="`${item.provider_id}-${item.upstream_name}-${item.checked_at}`" class="sp-monitor-row">
                    <span>
                      <strong>{{ item.provider_name || `供应商 ${item.provider_id}` }}</strong>
                      <small>{{ item.upstream_name }}</small>
                      <small v-if="item.primary_model">模型 {{ item.primary_model }}</small>
                    </span>
                    <span :class="{ 'sp-monitor-unmatched': !item.local_account_id }">
                      <strong>{{ item.local_account_name || '未匹配本地账号' }}</strong>
                      <small v-if="item.local_account_id">账号 #{{ item.local_account_id }}</small>
                    </span>
                    <span :class="{ 'sp-monitor-unmatched': !item.local_group_names?.length }">
                      <strong>{{ item.local_group_names?.length ? item.local_group_names.join('、') : '未归属本地分组' }}</strong>
                      <small v-if="item.local_group_ids?.length">分组 {{ item.local_group_ids.map(groupID => `#${groupID}`).join('、') }}</small>
                    </span>
                    <span><span class="sp-status" :class="statusTone(item.status)">{{ statusText(item.status) }}</span><small v-if="item.raw_status">上游 {{ item.raw_status }}</small></span>
                    <span><strong>{{ item.latency_ms }}ms</strong><small v-if="item.ping_latency_ms !== undefined">Ping {{ item.ping_latency_ms }}ms</small></span>
                    <span>{{ formatAvailability(item.availability_7d) }}</span>
                    <span>{{ formatTime(item.checked_at) }}</span>
                  </div>
                </div>
                <div v-else class="sp-rate-guard-empty">本次没有返回监控项，请检查供应商接口和 token 配置。</div>
              </section>
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
              <article><span>守护历史</span><strong>{{ detailRun.result_detail.cleanup.account_health_history }}</strong></article>
              <article><span>认证事件</span><strong>{{ detailRun.result_detail.cleanup.auth_events }}</strong></article>
              <article><span>监控样本</span><strong>{{ detailRun.result_detail.cleanup.monitor_samples }}</strong></article>
            </section>

            <pre v-else class="sp-message-detail">{{ detailMessage }}</pre>
          </section>
        </div>
        <pre v-else class="sp-message-detail">{{ detailMessage }}</pre>
        <template #footer>
          <button class="sp-button primary" type="button" @click="closeResultDetail">关闭</button>
        </template>
      </BaseDialog>

      <!-- 健康守护账号配置已抽成共享组件：上游账号页用的是同一个组件。
           账号候选数据仍由本页持有并传入，保证两处看到同一份列表、不重复拉取。 -->
      <SupplierHealthGuardAccountsDialog
        v-model:config="editForm.config"
        :show="healthGuardAccountsVisible"
        :accounts="healthGuardSupplierAccounts"
        :loading-accounts="loadingHealthGuardSupplierAccounts"
        :disabled-provider-ids="disabledSupplierProviderIDs"
        @confirm="closeHealthGuardAccounts"
        @close="closeHealthGuardAccounts"
      />

      <!-- 二级弹窗：z-index 必须高于「配置健康守护账号」（60）。BaseDialog 一律 Teleport 到 body，
           这里的 .modal-content 是父弹窗的**兄弟**节点，配色变量一条都继承不到，
           所以 .sp-multiplier-interval-dialog 自己声明了完整的 --sp-* 兜底（见样式块）。 -->

      <!-- 二级弹窗：批量设置账号。z-index 高于「配置健康守护账号」（60）、低于「配置倍率区间」（70）。
           内容区不再重复弹窗标题，作用范围与「留空即不改」的语义统一放在页脚提示里。 -->

      <BaseDialog
        :show="rateGuardGroupsVisible"
        title="配置参与守护的分组"
        width="full"
        :z-index="60"
        @close="closeRateGuardGroups"
      >
        <div class="sp-rate-guard-group-dialog">
          <section class="sp-rate-guard-group-workspace">
            <div class="sp-rate-guard-group-summary" aria-label="账号倍率守护分组配置摘要">
              <article>
                <span>参与守护</span>
                <strong>{{ rateGuardGroupScopeSummary.enabled }}</strong>
              </article>
              <article :class="{ warning: rateGuardGroupScopeSummary.disabled > 0 }">
                <span>已关闭</span>
                <strong>{{ rateGuardGroupScopeSummary.disabled }}</strong>
              </article>
              <article>
                <span>可选分组</span>
                <strong>{{ rateGuardGroups.length }}</strong>
              </article>
            </div>

            <div class="sp-rate-guard-group-toolbar">
              <div class="sp-rate-guard-group-filters">
                <Input v-model="rateGuardGroupSearch" placeholder="搜索分组名称或 ID" />
                <button
                  class="sp-rate-guard-group-selected-toggle"
                  :class="{ active: rateGuardGroupDisabledOnly }"
                  type="button"
                  :aria-pressed="rateGuardGroupDisabledOnly"
                  @click="rateGuardGroupDisabledOnly = !rateGuardGroupDisabledOnly"
                >
                  <span class="sp-rate-guard-group-selected-toggle-mark" aria-hidden="true"></span>
                  仅看已关闭
                  <strong>{{ rateGuardGroupScopeSummary.disabled }}</strong>
                </button>
              </div>
              <span class="sp-rate-guard-group-filter-result">
                筛选结果 <strong>{{ rateGuardFilteredGroups.length }}</strong> 个
              </span>
              <!-- 平台标签独占一行（flex-basis: 100%）：跟搜索框挤同一行会把搜索框压到
                   没法输入，而平台数量还会随分组增长。没有分组时不渲染，省掉一条空行。 -->
              <div
                v-if="groupPlatformFacets.length"
                class="sp-platform-chip-row"
                role="group"
                aria-label="按平台筛选分组"
              >
                <button
                  v-for="facet in groupPlatformFacets"
                  :key="facet.platform"
                  class="sp-platform-chip"
                  type="button"
                  :aria-pressed="rateGuardGroupPlatformFilter.includes(facet.platform)"
                  :style="{ '--sp-chip-platform-color': platformAccentColor(facet.platform) }"
                  @click="rateGuardGroupPlatformFilter = togglePlatformFilter(rateGuardGroupPlatformFilter, facet.platform)"
                >
                  {{ facet.label }}
                  <strong>{{ facet.count }}</strong>
                </button>
              </div>
            </div>

            <div v-if="loadingRateGuardGroups" class="sp-rate-guard-empty">正在加载分组...</div>
            <div v-else-if="rateGuardFilteredGroups.length" class="sp-rate-guard-group-list">
              <article
                v-for="group in rateGuardFilteredGroups"
                :key="group.id"
                class="sp-rate-guard-group-row"
                :class="{ disabled: rateGuardGroupIsDisabled(group.id) }"
                :style="{ '--sp-group-platform-color': platformAccentColor(group.platform) }"
              >
                <label class="sp-rate-guard-group-choice">
                  <input
                    type="checkbox"
                    :checked="!rateGuardGroupIsDisabled(group.id)"
                    :aria-label="`${rateGuardGroupIsDisabled(group.id) ? '开启' : '关闭'}分组 ${group.name} 的账号倍率守护`"
                    @change="toggleRateGuardGroup(group.id)"
                  />
                  <span class="sp-rate-guard-group-choice-copy">
                    <strong :class="platformTextClass(group.platform)">{{ group.name }}</strong>
                    <span class="sp-rate-guard-group-platform" :class="platformBadgeClass(group.platform)">
                      {{ platformLabel(group.platform) }}
                    </span>
                    <span class="sp-rate-guard-group-id">#{{ group.id }}</span>
                    <span class="sp-rate-guard-group-rate">倍率 {{ group.rate_multiplier }}</span>
                  </span>
                </label>
                <!-- 只在"已关闭"时显示标签：勾选行已经用复选框表达了"参与"，
                     再补一句"参与守护"就是把同一件事说两遍，几十行下来全是噪声。
                     把标签收窄成"异常提示"，列表里的离群行才能一眼跳出来。 -->
                <span
                  v-if="rateGuardGroupIsDisabled(group.id)"
                  class="sp-rate-guard-group-status off"
                >
                  已关闭守护
                </span>
              </article>
            </div>
            <div v-else class="sp-rate-guard-empty">{{ rateGuardGroupEmptyHint }}</div>
          </section>
        </div>
        <template #footer>
          <span class="sp-rate-guard-group-hint">取消勾选的分组会被跳过；全部勾选即所有分组都参与守护。</span>
          <button class="sp-button ghost" type="button" @click="enableAllRateGuardGroups">全部参与</button>
          <button class="sp-button primary" type="button" @click="closeRateGuardGroups">完成</button>
        </template>
      </BaseDialog>

      <SupplierGroupElectionGroupsDialog
        v-model:config="editForm.config"
        :show="electionGroupsVisible"
        :groups="rateGuardGroups"
        :loading-groups="loadingRateGuardGroups"
        :accounts="healthGuardSupplierAccounts"
        @confirm="closeElectionGroups"
        @close="closeElectionGroups"
      />

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

      <!-- 不带 group-id：这里是全局视角，看所有分组最近的调度切换。 -->
      <SupplierGroupElectionChangeLogDialog
        :show="electionChangeLogsVisible"
        @close="closeElectionChangeLogs"
      />

      <!-- 诊断快照：把排查所需的上下文（配置 / 各分组开启数 / 开启中的账号 / 最近几轮决策）
           拼成一段纯文本，供管理员一键复制交给他人排查，不必再手写 SQL。 -->
      <SupplierGroupElectionDiagnosticsDialog
        :show="electionDiagnosticsVisible"
        @close="closeElectionDiagnostics"
      />

    </div>
  </SupplierModuleLayout>
</template>

<script setup lang="ts">
import { computed, onMounted, reactive, ref } from 'vue'
import { SupplierAccountHealthGuardResult, SupplierAccountRateGuardLogDialog, SupplierGroupElectionChangeLogDialog, SupplierGroupElectionDiagnosticsDialog, SupplierGroupElectionGroupsDialog, SupplierHealthGuardAccountsDialog, SupplierModuleLayout } from '@/components/admin/supplier-management'
import BaseDialog from '@/components/common/BaseDialog.vue'
import DataTable from '@/components/common/DataTable.vue'
import Input from '@/components/common/Input.vue'
import Pagination from '@/components/common/Pagination.vue'
import Select, { type SelectOption } from '@/components/common/Select.vue'
import Toggle from '@/components/common/Toggle.vue'
import Icon from '@/components/icons/Icon.vue'
import type { Column } from '@/components/common/types'
import type { SupplierProviderAccount } from '@/api/admin/supplierProviderData'
import { getAllIncludingInactive as listAllGroups } from '@/api/admin/groups'
// 两个配置弹窗的候选数据（可用账号 / 已关闭的供应商）：上游账号页原地打开同一个弹窗，
// 也走这一份，「哪些账号算可用」才不会在两个页面里各写一套。
import {
  fetchDisabledSupplierProviderIds,
  fetchEligibleSupplierAccounts,
} from './supplierAutomationAccountCandidates'
import type { AdminGroup } from '@/types'
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
  type SupplierProviderRechargeSyncAllResult,
  type SupplierProviderMonitorSyncItem,
} from '@/api/admin/supplierAutomation'
import { useAppStore } from '@/stores/app'
import { ensureCustomPlatformLabels, resolvePlatformDisplayLabel as platformLabel } from '@/utils/customPlatformLabels'
import { platformAccentColor, platformBadgeClass, platformTextClass } from '@/utils/platformColors'
import { extractApiErrorMessage } from '@/utils/apiError'
import { cronToIntervalSeconds } from './supplierAutomationCron'
// 健康守护账号 / 分组择优配置的纯函数层：页面与弹窗组件共用同一套归一化 / 校验 / 账号映射规则。
import {
  buildHealthGuardAccountMappings,
  buildPlatformFacets,
  groupFilterEmptyHint,
  matchesPlatformFilter,
  normalizeAccountHealthGuardAccountIntervals,
  normalizeAccountHealthGuardSchedulingChange,
  normalizeAccountHealthGuardAccountThresholds,
  normalizeHealthGuardPlatformMultiplierIntervals,
  normalizePositiveAccountIDs,
  normalizePositiveNumberMap,
  normalizeStringMap,
  positiveIntegerOr,
  toNumber,
  togglePlatformFilter,
  validateAccountHealthGuardConfig,
  validateAccountHealthGuardSelection,
  type HealthGuardAccountMapping,
} from './supplierAutomationConfig'
// 分组择优配置的派生值与分组级操作：编辑弹窗摘要、保存归一化、「配置参与择优的分组」弹窗共用同一份口径。
import { useGroupElectionConfig } from './useGroupElectionConfig'

const tasks = ref<SupplierAutomationTask[]>([])
const runs = ref<SupplierAutomationRun[]>([])
const lastRefreshedAt = ref('')
const loading = ref(false)
const appStore = useAppStore()
const savingCode = ref('')
const runningCode = ref('')
const updatingTaskIDs = ref<Set<string>>(new Set())
const runningMode = ref<'preview' | 'execute'>('execute')
const editVisible = ref(false)
const editingTask = ref<SupplierAutomationTask | null>(null)
const editIntervalSeconds = ref(900)
const detailVisible = ref(false)
const detailTitle = ref('')
const detailMessage = ref('')
const detailRun = ref<SupplierAutomationRun | null>(null)
const selectedDetailProviderID = ref<number | null>(null)
 const runPage = ref(1)
const runPageSize = ref(10)
const runTotal = ref(0)
const runTaskFilter = ref('')
const runStatusFilter = ref('')
const accountRateGuardExecuteVisible = ref(false)
const pendingExecuteTask = ref<SupplierAutomationTask | null>(null)
const accountRateGuardLogsVisible = ref(false)
const accountRateGuardPendingCount = ref(0)
// 全局视角的调度切换日志：不锁定分组，分组页那个入口才带 group-id。
const electionChangeLogsVisible = ref(false)
const electionDiagnosticsVisible = ref(false)
const rateGuardGroupsVisible = ref(false)
const rateGuardGroupSearch = ref('')
const rateGuardGroupDisabledOnly = ref(false)
// 多选：空数组 = 不按平台过滤。存「已选中」而不是「已排除」，
// 因为平台数量会随分组增删变化，存已排除会在新增平台时出现"凭空多出一个筛选项"的错觉。
const rateGuardGroupPlatformFilter = ref<string[]>([])
const rateGuardGroups = ref<AdminGroup[]>([])
const loadingRateGuardGroups = ref(false)
const electionGroupsVisible = ref(false)
const healthGuardAccountsVisible = ref(false)
// 与「仅看已选」是同一维度的两个互斥方向：同时开启只会得到空集，所以开一个必须关掉另一个。
// 「供应商是否开启」是另一个维度，与上面两个正交，可以和它们叠加；
// 但开启/关闭本身是一对互斥方向，所以这两个之间互斥。
const healthGuardSupplierAccounts = ref<SupplierProviderAccount[]>([])
// 已关闭（停用）的供应商 ID。账号列表接口不返回这个状态，只能另取一份供应商列表。
const disabledSupplierProviderIDs = ref<Set<number>>(new Set())
const loadingHealthGuardSupplierAccounts = ref(false)

 
const editForm = reactive<SupplierAutomationTask>({
  id: 0,
  task_code: '',
  name: '',
  enabled: true,
  cron_expression: '',
  timeout_seconds: 600,
  config: {
    rate_guard_max_snapshot_age_seconds: 1800,
    account_rate_guard_disabled_group_ids: [],
    automation_run_retention_days: 30,
    sync_run_retention_days: 30,
    metric_snapshot_retention_days: 30,
    daily_stat_retention_days: 365,
    inactive_account_retention_days: 90,
    inactive_group_retention_days: 90,
    account_health_guard_max_accounts_per_run: 200,
    account_health_guard_concurrency: 3,
    account_health_guard_timeout_per_account_seconds: 30,
    account_health_guard_failure_threshold: 3,
    account_health_guard_slow_threshold: 3,
    account_health_guard_recovery_threshold: 2,
    account_health_guard_healthy_latency_ms: 15000,
    account_health_guard_account_ids: [],
    account_health_guard_account_models: {},
    account_health_guard_account_intervals: {},
    account_health_guard_platform_models: {},
    account_health_guard_platform_latency_ms: {},
    account_health_guard_account_scheduling_change: {},
    // 全局修改调度默认开启：升级前的行为就是"检测到异常自动暂停调度"。
    account_health_guard_scheduling_change_enabled: true,
    account_health_guard_account_failure_thresholds: {},
    account_health_guard_account_slow_thresholds: {},
    account_health_guard_account_recovery_thresholds: {},
    account_health_guard_platform_multiplier_intervals_enabled: false,
    account_health_guard_platform_multiplier_intervals: {},
    account_health_guard_cursor_account_id: 0,
    group_scheduling_election_top_n: 1,
    group_scheduling_election_top_n_by_group: {},
    group_scheduling_election_disabled_group_ids: [],
    group_scheduling_election_count_weight: 1,
    group_scheduling_election_latency_weight: 0.5,
    group_scheduling_election_priority_weight: 0.5,
    // 优先级计分默认关闭：升级前的行为就是"优先级不参与择优"。
    group_scheduling_election_priority_enabled_global: false,
    group_scheduling_election_priority_enabled_group_ids: [],
    group_scheduling_election_priority_disabled_group_ids: [],
    group_scheduling_election_failure_threshold: 2,
    group_scheduling_election_switch_margin: 0.15,
    group_scheduling_election_latency_window_minutes: 30,
    group_scheduling_election_latency_min_samples: 3,
    group_scheduling_election_count_score_cap: 10,
    group_scheduling_election_keep_healthy_incumbent_group_ids: [],
    // 在任者健康锁定默认关闭：升级前的行为就是"不锁定、照常择优换人"。
    group_scheduling_election_keep_healthy_incumbent_global: false,
    group_scheduling_election_keep_healthy_incumbent_excluded_group_ids: [],
    group_scheduling_election_required_models: {},
    // 演练默认关闭：默认行为必须是"真的择优"，演练是管理员显式选择的观察模式。
    group_scheduling_election_dry_run: false,
    group_scheduling_election_dry_run_group_ids: [],
    // 异常推送默认关闭：升级前不发任何消息，开启是管理员的显式选择。
    group_scheduling_election_alert_enabled: false,
    group_scheduling_election_alert_group_overrides: {},
    // 默认账号默认不指定：空对象表示所有分组都按择优结果开。
    group_scheduling_election_default_account_by_group: {},
  },
  last_status: '',
  last_message: '',
})

const taskNameByCode = computed<Record<string, string>>(() =>
  Object.fromEntries(tasks.value.map(task => [task.task_code, task.name]))
)
const taskNameFallbackByCode: Record<string, string> = {
  supplier_provider_recharge_sync: '供应商充值记录同步',
  supplier_group_scheduling_election: '分组择优调度',
}

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
const groupElectionResult = computed(() => detailRun.value?.result_detail?.group_election || null)
const accountHealthGuardResult = computed(() => detailRun.value?.result_detail?.account_health_guard || null)
const rechargeSyncResult = computed<SupplierProviderRechargeSyncAllResult | null>(() => detailRun.value?.result_detail?.recharge_sync || null)
const rechargeSyncRecordCount = computed(() => (
  rechargeSyncResult.value?.items.reduce((total, item) => total + (item.record_count || 0), 0) || 0
))
const supplierMonitorResult = computed(() => detailRun.value?.result_detail?.supplier_monitor || null)
const supplierMonitorItems = computed<SupplierProviderMonitorSyncItem[]>(() => supplierMonitorResult.value?.items || [])
const supplierMonitorDisplayItems = computed<SupplierProviderMonitorSyncItem[]>(() => {
  const latestByMonitor = new Map<string, SupplierProviderMonitorSyncItem>()
  for (const item of supplierMonitorItems.value) {
    const key = `${item.provider_id}:${item.upstream_key || item.upstream_name}`
    const previous = latestByMonitor.get(key)
    if (!previous || supplierMonitorCheckedAt(item) > supplierMonitorCheckedAt(previous)) {
      latestByMonitor.set(key, item)
    }
  }
  return Array.from(latestByMonitor.values())
})
const supplierMonitorMatchedAccountCount = computed(() => (
  supplierMonitorDisplayItems.value.filter(item => Boolean(item.local_account_id)).length
))
const supplierMonitorUnmatchedAccountCount = computed(() => (
  supplierMonitorDisplayItems.value.filter(item => !item.local_account_id).length
))
const supplierMonitorMatchedGroupCount = computed(() => (
  supplierMonitorDisplayItems.value.filter(item => Boolean(item.local_group_names?.length)).length
))
const supplierMonitorUngroupedCount = computed(() => (
  supplierMonitorDisplayItems.value.filter(item => !item.local_group_names?.length).length
))
const supplierMonitorHealthyCount = computed(() => (
  supplierMonitorDisplayItems.value.filter(item => item.status === 'healthy').length
))
const supplierMonitorSlowCount = computed(() => (
  supplierMonitorDisplayItems.value.filter(item => item.status === 'slow').length
))
const supplierMonitorFailedCount = computed(() => (
  supplierMonitorDisplayItems.value.filter(item => item.status === 'failed').length
))
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
  try {
    await ensureCustomPlatformLabels()
    tasks.value = await listTasks()
    await loadRuns()
    await loadAccountRateGuardPendingCount()
    lastRefreshedAt.value = new Date().toISOString()
  } catch (err) {
    appStore.showError(extractApiErrorMessage(err, '加载自动化任务失败'))
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

function openElectionChangeLogs() {
  electionChangeLogsVisible.value = true
}

function closeElectionChangeLogs() {
  electionChangeLogsVisible.value = false
}

function openElectionDiagnostics() {
  electionDiagnosticsVisible.value = true
}

function closeElectionDiagnostics() {
  electionDiagnosticsVisible.value = false
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
  return taskNameByCode.value[taskCode] || taskNameFallbackByCode[taskCode] || taskCode
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
  applyAccountHealthGuardDefaults()
  applyAccountRateGuardDefaults()
  applyGroupElectionDefaults()
  editIntervalSeconds.value = cronToIntervalSeconds(task.cron_expression) || 300
  editVisible.value = true
}

function closeEdit() {
  editVisible.value = false
}

async function toggleTaskStatus(taskCode: string, enabled: boolean) {
  const task = tasks.value.find(item => item.task_code === taskCode)
  if (!task || task.enabled === enabled || updatingTaskIDs.value.has(taskCode)) return

  const previousEnabled = task.enabled
  task.enabled = enabled
  updatingTaskIDs.value = new Set(updatingTaskIDs.value).add(taskCode)
  try {
    await updateTask(taskCode, task)
    appStore.showSuccess(enabled ? '任务已启用' : '任务已停用')
    await loadData()
  } catch (err) {
    task.enabled = previousEnabled
    appStore.showError(extractApiErrorMessage(err, '更新任务运行状态失败'))
  } finally {
    const next = new Set(updatingTaskIDs.value)
    next.delete(taskCode)
    updatingTaskIDs.value = next
  }
}

async function saveTask() {
  if (!editingTask.value) return
  const cronExpression = intervalSecondsToCron(editIntervalSeconds.value)
  if (!cronExpression) {
    appStore.showError('执行间隔必须是正整数秒')
    return
  }
  if (editForm.task_code === 'supplier_rate_guard') {
    if (editForm.config.rate_guard_max_snapshot_age_seconds < 60) {
      appStore.showError('快照最大有效期不能少于 60 秒')
      return
    }
  }
  if (editForm.task_code === 'supplier_account_health_guard') {
    try {
      await ensureHealthGuardAccountCandidatesLoaded()
    } catch (err) {
      appStore.showError(extractApiErrorMessage(err, '加载健康守护账号失败'))
      return
    }
    const validationMessage = validateAccountHealthGuardConfig(editForm.config, healthGuardAccountMappings.value)
    if (validationMessage) {
      appStore.showError(validationMessage)
      return
    }
  }
  if (editForm.task_code === 'supplier_account_rate_guard') {
    // 提交前再归一化一次：弹窗里勾选产生的值要保证去重升序，且非法 ID 不入库。
    applyAccountRateGuardDefaults()
  }
  if (editForm.task_code === 'supplier_group_scheduling_election') {
    const topN = Number(editForm.config.group_scheduling_election_top_n)
    if (!Number.isInteger(topN) || topN < 1) {
      appStore.showError('每组开启账号数必须是不小于 1 的整数')
      return
    }
    if (topN > 100) {
      appStore.showError('每组开启账号数不能超过 100')
      return
    }
    const topNByGroup = editForm.config.group_scheduling_election_top_n_by_group ?? {}
    for (const groupTopN of Object.values(topNByGroup)) {
      const n = Number(groupTopN)
      if (!Number.isInteger(n) || n < 1 || n > 100) {
        appStore.showError('分组单独设置的每组开启账号数必须是 1–100 的整数')
        return
      }
    }
    applyGroupElectionDefaults()
  }
  editForm.cron_expression = cronExpression
  savingCode.value = editingTask.value.task_code
  try {
    await updateTask(editingTask.value.task_code, editForm)
    appStore.showSuccess('任务已保存')
    editVisible.value = false
    await loadData()
  } catch (err) {
    appStore.showError(extractApiErrorMessage(err, '保存任务失败'))
  } finally {
    savingCode.value = ''
  }
}

async function runNow(taskCode: string) {
  if (taskCode === 'supplier_account_health_guard') {
    try {
      await ensureHealthGuardAccountCandidatesLoaded()
    } catch (err) {
      appStore.showError(extractApiErrorMessage(err, '加载健康守护账号失败'))
      return
    }
    const task = tasks.value.find(item => item.task_code === taskCode)
    const validationMessage = task
      ? validateAccountHealthGuardSelection(task.config, healthGuardAccountMappings.value)
      : '请至少选择一个需要检查的账号'
    if (validationMessage) {
      appStore.showError(validationMessage)
      return
    }
  }
  runningCode.value = taskCode
  runningMode.value = 'execute'
  try {
    const run = await runTask(taskCode)
    appStore.showSuccess(`任务执行完成：${statusText(run.status)}`)
    runPage.value = 1
    await loadData()
    openRunDetail(run)
  } catch (err) {
    appStore.showError(extractApiErrorMessage(err, '运行任务失败'))
    runPage.value = 1
    try {
      await loadData()
      // 后端可能已落库 run（超时/执行失败），尽量展示最近一次结构化结果
      const task = tasks.value.find(item => item.task_code === taskCode)
      if (task) {
        await openTaskLatestResult(task)
      }
    } catch {
      // 忽略刷新失败，保留上方错误提示
    }
  } finally {
    runningCode.value = ''
  }
}

async function runPreview(taskCode: string) {
  runningCode.value = taskCode
  runningMode.value = 'preview'
  try {
    const run = await runTask(taskCode, 'preview')
    appStore.showSuccess(`检测预览完成：发现 ${run.result_detail?.account_rate_guard?.risk_groups || 0} 个风险分组`)
    runPage.value = 1
    await loadData()
    openRunDetail(run)
  } catch (err) {
    appStore.showError(extractApiErrorMessage(err, '账号倍率守护检测预览失败'))
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
    appStore.showSuccess(`账号倍率守护执行完成：解除 ${run.result_detail?.account_rate_guard?.unbound_groups || 0} 个分组绑定`)
    accountRateGuardExecuteVisible.value = false
    pendingExecuteTask.value = null
    runPage.value = 1
    await loadData()
    openRunDetail(run)
  } catch (err) {
    appStore.showError(extractApiErrorMessage(err, '执行账号倍率守护失败'))
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

async function openTaskLatestResult(task: SupplierAutomationTask) {
  // 不依赖当前历史分页/筛选，按任务拉取最近一次完整运行记录
  try {
    const result = await listRuns({
      task_code: task.task_code,
      page: 1,
      page_size: 1,
    })
    const latest = result.items[0]
    if (latest) {
      openRunDetail(latest)
      return
    }
  } catch (err) {
    appStore.showError(extractApiErrorMessage(err, '加载最近结果失败'))
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

// suggested：演练模式下这个账号只是"本该被拨动"，没真写库，文案必须带上"建议"，
// 否则运行明细里一条"开启调度"会被当成已经生效的切换。
function groupElectionActionText(action: string, suggested?: boolean): string {
  const prefix = suggested ? '建议' : ''
  switch (action) {
    case 'enabled':
      return `${prefix}开启调度`
    case 'disabled':
      return `${prefix}关闭调度`
    default:
      return suggested ? '建议变更' : '保持不变'
  }
}

function groupElectionTestStatusText(status?: string): string {
  switch (status) {
    case 'success':
      return '成功'
    case 'failed':
      return '失败'
    default:
      return '未测试'
  }
}

// 与供应商健康页的 formatLatency 保持同一口径：没有数据显示"—"而不是 0 ms，
// 因为 0 会被读成"极快"，实际上只是这个账号还没测出耗时。
function groupElectionLatencyText(value?: number | null): string {
  const latency = Number(value)
  if (!Number.isFinite(latency) || latency <= 0) return '—'
  return `${Math.round(latency)} ms`
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
  } else if (run.result_detail?.recharge_sync) {
    const rechargeSync = run.result_detail.recharge_sync
    const recordCount = rechargeSync.items.reduce((total, item) => total + (item.record_count || 0), 0)
    lines.push(
      '',
      '供应商充值记录同步明细：',
      `- 同步供应商：${rechargeSync.items.length}`,
      `- 成功 / 失败：${rechargeSync.success_count} / ${rechargeSync.failed_count}`,
      `- 同步记录数：${recordCount}`
    )
    for (const item of rechargeSync.items) {
      lines.push(`- ${item.provider_name || `供应商 ${item.provider_id}`}：${statusText(item.status)}，${item.record_count} 条，${item.message || '无附加信息'}`)
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
  } else if (run.result_detail?.account_health_guard) {
    const guard = run.result_detail.account_health_guard
    lines.push(
      '',
      '健康守护明细：',
      `- 检查：${guard.checked_count}`,
      `- 健康：${guard.healthy_count}`,
      `- 慢响应：${guard.slow_count}`,
      `- 失败：${guard.failed_count}`,
      `- 跳过：${guard.skipped_count}`,
      `- 暂停：${guard.disabled_count}`,
      `- 恢复：${guard.recovered_count}`
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
      `- 供应商分组：${cleanup.groups}`,
      `- 守护历史：${cleanup.account_health_history}`,
      `- 认证事件：${cleanup.auth_events}`,
      `- 监控样本：${cleanup.monitor_samples}`
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
  const healthGuard = run.result_detail?.account_health_guard
  if (healthGuard) {
    return `检查 ${healthGuard.checked_count}，健康 ${healthGuard.healthy_count}，慢响应 ${healthGuard.slow_count}，失败 ${healthGuard.failed_count}，不可用 ${healthGuard.unavailable_count}，待下轮 ${healthGuard.pending_count}，暂停 ${healthGuard.disabled_count}，恢复 ${healthGuard.recovered_count}`
  }
  const groupElection = run.result_detail?.group_election
  if (groupElection) {
    // 被闸门拦住的账号不算失败，但要让人一眼看见：否则「关闭 0 个」会被读成一切正常。
    let summary = `扫描 ${groupElection.group_count} 个分组，开启 ${groupElection.enabled_count}，关闭 ${groupElection.disabled_count}`
    if (groupElection.pending_count) summary += `，${groupElection.pending_count} 个待观察`
    if (groupElection.kept_count) summary += `，${groupElection.kept_count} 个待人工确认`
    if (groupElection.required_model_uncovered_count) summary += `，${groupElection.required_model_uncovered_count} 个必需模型待恢复`
    // 演练必须写在摘要里：演练轮次的"开启/关闭"恒为 0，不说明就会被读成任务空转。
    if (groupElection.dry_run) {
      summary += `（演练：建议开启 ${groupElection.suggested_enabled_count ?? 0}、建议关闭 ${groupElection.suggested_disabled_count ?? 0}，未实际修改调度）`
    }
    return summary
  }
  const rechargeSync = run.result_detail?.recharge_sync
  if (rechargeSync) {
    const recordCount = rechargeSync.items.reduce((total, item) => total + (item.record_count || 0), 0)
    return `同步 ${rechargeSync.items.length} 个供应商，成功 ${rechargeSync.success_count}，失败 ${rechargeSync.failed_count}，充值记录 ${recordCount} 条`
  }
  if (!run.processed_count && !run.success_count && !run.failed_count) {
    return compactMessage(run.message || '暂无结果')
  }
  return `${run.processed_count} 个对象，${run.success_count} 成功，${run.failed_count} 失败`
}







const healthGuardAccountMappings = computed<HealthGuardAccountMapping[]>(() =>
  buildHealthGuardAccountMappings(healthGuardSupplierAccounts.value)
)

const healthGuardAccountIDs = computed(() =>
  normalizePositiveAccountIDs(editForm.config.account_health_guard_account_ids)
)

// 账号倍率守护的分组开关：配置里存的是"被关闭"的分组，空列表即所有分组都参与。
const accountRateGuardDisabledGroupIDs = computed(() =>
  normalizePositiveAccountIDs(editForm.config.account_rate_guard_disabled_group_ids)
)

// ── 平台筛选（「配置参与守护/择优的分组」与「健康守护账号」三个弹窗共用）──
//
// 平台列表从当前数据里现算，而不是取全平台枚举：只列出「此刻确实有内容的平台」，
// 避免用户点到一个筛完空空如也的标签，误以为筛选坏了。
// 传数组而不是 Map，调用方不必先自己聚合；分组弹窗按分组数统计，账号弹窗按可用账号数统计。


const groupPlatformFacets = computed(() => buildPlatformFacets(rateGuardGroups.value))

// 标签是多选：点一下加入、再点一下移除。
// 不做成单选切换 —— 切到下一个平台就会丢掉上一个的选择，而「同时看两个平台」
// 恰恰是这个筛选最常见的用法。

// 空选 = 不按平台过滤。与「仅看已关闭」默认关保持一致：打开弹窗先看到全部。


const rateGuardGroupEmptyHint = computed(() =>
  groupFilterEmptyHint(
    rateGuardGroupPlatformFilter.value.length,
    rateGuardGroupDisabledOnly.value,
    '当前没有已关闭守护的分组。'
  )
)

// 「分组择优调度」的派生值与分组级操作统一收在 useGroupElectionConfig 里 ——
// 编辑弹窗里的「分组择优调度策略」摘要、保存时的归一化、以及「配置参与择优的分组」弹窗，
// 三处读的必须是同一份口径（上游账号页复用那个弹窗时也走同一份），否则会出现
// 「摘要说 3 个分组、弹窗里数出 4 个」这种没人能一眼看出是谁错的分叉。
const {
  groupElectionDisabledGroupIDs,
  groupElectionKeepHealthyGroupIDs,
  groupElectionKeepHealthyExcludedGroupIDs,
  groupElectionDryRunGroupIDs,
  electionDryRunAll,
  electionKeepHealthyAll,
  groupElectionPriorityEnabledGroupIDs,
  groupElectionPriorityDisabledGroupIDs,
  electionPriorityAll,
  electionAlertEnabled,
  electionAlertOverrides,
  electionAlertOverrideCount,
  electionAlertMutedCount,
  electionAlertForcedCount,
  groupElectionDefaultAccountMap,
  groupElectionDefaultAccountCount,
  groupElectionRequiredModelsMap,
  groupElectionRequiredModelsCount,
  groupElectionTopNByGroupMap,
  groupElectionTopNOverridesCount,
  setElectionDryRunAll,
  setElectionKeepHealthyAll,
  setElectionPriorityAll,
  setElectionAlertEnabled,
} = useGroupElectionConfig(
  () => editForm.config,
  () => rateGuardGroups.value,
  () => healthGuardSupplierAccounts.value
)

// 编辑弹窗的宽度由「最宽那一行的列数」决定，不是区块个数 ——
// 3 列网格（健康守护 7 个输入框 / 数据保留 4 个输入框）每列都要放得下「标签 + 输入框」，
// 压窄会把标签折行、区块高度反而涨上去；2 列及以下（其余任务）用窄档即可，
// 铺满只会把内容拉散（实测账号倍率守护铺满后策略卡里从文字到按钮空出 1012px）。
// 窄屏下 3 列会由既有媒体查询自动降级为 2 列，所以这里只表达「宽屏需要多宽」。
//
// 分组择优是唯一的例外：它的网格实际是 2 列（.sp-group-election-policy-grid 要给每个字段
// 放一段说明文字，3 列会把说明挤成 5~6 行），但**仍留在 3 档** —— 窄档是宽度和高度一起收的
// （980px 宽 + 640px 高），而择优实测表单内容总高 1015px —— 光「分组择优调度策略」这一个区块
// 自己就 656px（8 个配置项各带一段说明 + 一张范围卡片），收到 640px 必然滚动、看不完一屏。
// 所以这里的 3 对择优而言表达的是「内容量需要大弹窗」，
// 不是「表单排 3 列」——别看到这个 3 就去把网格改回 3 列。
const editDialogGridCols = computed<2 | 3>(() =>
  editForm.task_code === 'supplier_account_health_guard' ||
  editForm.task_code === 'supplier_data_cleanup' ||
  editForm.task_code === 'supplier_group_scheduling_election'
    ? 3
    : 2
)

const rateGuardGroupScopeSummary = computed(() => {
  const disabled = accountRateGuardDisabledGroupIDs.value.length
  return {
    disabled,
    enabled: Math.max(rateGuardGroups.value.length - disabled, 0),
  }
})

const rateGuardFilteredGroups = computed(() => {
  const keyword = rateGuardGroupSearch.value.trim().toLowerCase()
  let result = rateGuardGroups.value
  if (rateGuardGroupPlatformFilter.value.length > 0) {
    result = result.filter(group =>
      matchesPlatformFilter(group.platform, rateGuardGroupPlatformFilter.value)
    )
  }
  if (rateGuardGroupDisabledOnly.value) {
    result = result.filter(group => rateGuardGroupIsDisabled(group.id))
  }
  if (!keyword) {
    return result
  }
  return result.filter(group =>
    group.name.toLowerCase().includes(keyword) || String(group.id).includes(keyword)
  )
})




// 入口区只显示一行摘要：配置搬进二级弹窗后，用户需要一个信号判断「有没有配过」，
// 否则「未开调度账号按倍率间隔」开着却不知道配没配。



// 账号弹窗的平台标签与分组弹窗共用同一套渲染，只是统计口径换成「可用账号数」——
// 与它替换掉的下拉选项口径一致，筛选行为不变。
//
// 平台与供应商两个筛选互相收窄（联动）：
//   · 平台标签的候选与计数只统计「供应商筛选后」的账号；
//   · 供应商下拉的候选与计数只统计「平台筛选后」的账号。
// 两边各自只应用**对方**那一个条件 —— 不能拿最终列表（healthGuardWorkspaceAccounts）算候选，
// 那会把当前选中项永远留在候选里，联动就退化成两个独立筛选。






/**
 * 倍率升序的排序键。
 * 一个账号可能有多个来源、各自倍率不同（行上显示成「0.5 / 0.8」），取其中最小值。
 * 倍率取值与展示用的是同一套过滤（Number + Number.isFinite），避免排序键和行上显示的数字对不上。
 * 完全没有可用倍率的账号排到最后 —— 用 0 代替会让「免费的」和「查不到倍率的」混在一起。
 */


// 「仅看未开启」按钮上的计数。只数**可用**账号：不可用且未参与的账号根本不进列表
// （列表基集 = 可用账号 + 已选账号），算进来会出现「按钮说 2、点开只有 1 行」。
// 与「仅看已选」一样取全局口径，不跟着平台/供应商/搜索变，否则数字一直跳。

/**
 * 账号的供应商是否已全部关闭。
 *
 * 一个本地账号可能有多个上游来源（`sources` 每项一个供应商），这里用 `every` 而不是 `some`：
 * 只要还有一个来源的供应商是开启的，该账号就仍能正常工作，不该被标成「供应商已关闭」。
 *
 * 供应商列表没拿到、或某个 provider_id 不在列表里时一律按「开启」处理 ——
 * 宁可漏标，也不要误标灰（误标会让人以为这个账号已经废了）。
 */

// 供应商维度的两个计数，口径与另外两个按钮一致（全局可用账号，不随其它筛选变）。
// 「开启数」用「总数 - 关闭数」而不是再 filter 一遍：两者互斥且互补，
// 写成减法不会出现「两个数字加起来不等于总数」的口径漂移。


// 空态文案按当前生效的快捷过滤给出 —— 筛出空列表时必须说清是哪条筛出来的，
// 否则用户看到「没有可配置账号」会去查数据，而不是去关掉刚点的那个过滤。

// 两个快捷过滤互斥：同时开启只会得到空集，所以开一个就关掉另一个。


// 供应商维度的互斥对：开一个就关掉另一个。与上面那对（参与守护）互不干扰 ——
// 两个维度正交，用户要能同时看「未参与守护 + 供应商已关闭」。









// 多选集合：纯界面瞬时状态（不写进 config），只决定「批量设置 / 批量间隔」的作用范围。
// 与上面的「参与守护」是两个概念 —— 那个是业务配置、保存后生效；勾选只是本次批量操作的选择。
// 合并成一个控件会让「想批量设阈值」被迫先把账号纳入守护。






async function ensureHealthGuardAccountCandidatesLoaded() {
  loadingHealthGuardSupplierAccounts.value = true
  try {
    // 供应商状态与账号列表并行取，别串行加一轮等待。
    // 供应商状态拿不到时降级成「没有已关闭的供应商」而不是整体抛错：它是附加信息，
    // 拿不到只影响标灰与那个快捷过滤，不该把账号列表一起拖垮、也不该弹一个
    // 「加载健康守护账号失败」把用户引到错误的方向。
    const providersPromise = fetchDisabledSupplierProviderIds()
    healthGuardSupplierAccounts.value = await fetchEligibleSupplierAccounts()
    disabledSupplierProviderIDs.value = await providersPromise
  } finally {
    loadingHealthGuardSupplierAccounts.value = false
  }
}


function applyAccountHealthGuardDefaults() {
  const config = editForm.config
  config.account_health_guard_max_accounts_per_run = positiveIntegerOr(config.account_health_guard_max_accounts_per_run, 200)
  config.account_health_guard_concurrency = positiveIntegerOr(config.account_health_guard_concurrency, 3)
  config.account_health_guard_timeout_per_account_seconds = positiveIntegerOr(config.account_health_guard_timeout_per_account_seconds, 30)
  config.account_health_guard_failure_threshold = positiveIntegerOr(config.account_health_guard_failure_threshold, 3)
  config.account_health_guard_slow_threshold = positiveIntegerOr(config.account_health_guard_slow_threshold, 3)
  config.account_health_guard_recovery_threshold = positiveIntegerOr(config.account_health_guard_recovery_threshold, 2)
  config.account_health_guard_healthy_latency_ms = positiveIntegerOr(config.account_health_guard_healthy_latency_ms, 15000)
  config.account_health_guard_account_ids = normalizePositiveAccountIDs(config.account_health_guard_account_ids)
  config.account_health_guard_account_models = normalizeStringMap(config.account_health_guard_account_models)
  config.account_health_guard_platform_models = normalizeStringMap(config.account_health_guard_platform_models)
  config.account_health_guard_platform_latency_ms = normalizePositiveNumberMap(config.account_health_guard_platform_latency_ms)
  config.account_health_guard_account_intervals = normalizeAccountHealthGuardAccountIntervals(config.account_health_guard_account_intervals)
  config.account_health_guard_account_scheduling_change = normalizeAccountHealthGuardSchedulingChange(config.account_health_guard_account_scheduling_change)
  // 旧配置里没有这个键，读回来是 undefined —— 只有显式 false 才算关闭，否则一律按开启处理。
  config.account_health_guard_scheduling_change_enabled = config.account_health_guard_scheduling_change_enabled !== false
  config.account_health_guard_account_failure_thresholds = normalizeAccountHealthGuardAccountThresholds(config.account_health_guard_account_failure_thresholds)
  config.account_health_guard_account_slow_thresholds = normalizeAccountHealthGuardAccountThresholds(config.account_health_guard_account_slow_thresholds)
  config.account_health_guard_account_recovery_thresholds = normalizeAccountHealthGuardAccountThresholds(config.account_health_guard_account_recovery_thresholds)
  config.account_health_guard_platform_multiplier_intervals_enabled = Boolean(config.account_health_guard_platform_multiplier_intervals_enabled)
  config.account_health_guard_platform_multiplier_intervals = normalizeHealthGuardPlatformMultiplierIntervals(config.account_health_guard_platform_multiplier_intervals)
  const cursorAccountID = Number(config.account_health_guard_cursor_account_id)
  config.account_health_guard_cursor_account_id = Number.isSafeInteger(cursorAccountID) && cursorAccountID > 0 ? cursorAccountID : 0
}

function applyAccountRateGuardDefaults() {
  // 旧配置里没有这个字段，归一化后得到空数组 —— 正好是"所有分组都参与守护"的默认语义。
  editForm.config.account_rate_guard_disabled_group_ids = normalizePositiveAccountIDs(
    editForm.config.account_rate_guard_disabled_group_ids
  )
}

function applyGroupElectionDefaults() {
  // 旧配置缺字段时回落默认：每组保留 1 个最优账号（严格单活），所有分组都参与择优。
  const topN = Number(editForm.config.group_scheduling_election_top_n)
  editForm.config.group_scheduling_election_top_n = Number.isFinite(topN) && topN > 0 ? topN : 1
  editForm.config.group_scheduling_election_disabled_group_ids = normalizePositiveAccountIDs(
    editForm.config.group_scheduling_election_disabled_group_ids
  )
  editForm.config.group_scheduling_election_keep_healthy_incumbent_group_ids = normalizePositiveAccountIDs(
    editForm.config.group_scheduling_election_keep_healthy_incumbent_group_ids
  )
  // 在任者健康锁定的全局开关收敛成真正的布尔（旧配置没有这个键，读回来是 undefined）；
  // 强制不锁定名单必须序列化成数组而非省略——后端整块覆盖 config_json，省略等于保留旧值。
  editForm.config.group_scheduling_election_keep_healthy_incumbent_global =
    editForm.config.group_scheduling_election_keep_healthy_incumbent_global === true
  editForm.config.group_scheduling_election_keep_healthy_incumbent_excluded_group_ids = normalizePositiveAccountIDs(
    editForm.config.group_scheduling_election_keep_healthy_incumbent_excluded_group_ids
  )
  // 权重必须是正数：0 在后端归一化里被当成"未配置"回落默认，
  // 这里先在前端拦住，免得管理员填了 0 却静默拿到默认值。
  const countWeight = Number(editForm.config.group_scheduling_election_count_weight)
  editForm.config.group_scheduling_election_count_weight =
    Number.isFinite(countWeight) && countWeight > 0 ? countWeight : 1
  const latencyWeight = Number(editForm.config.group_scheduling_election_latency_weight)
  editForm.config.group_scheduling_election_latency_weight =
    Number.isFinite(latencyWeight) && latencyWeight > 0 ? latencyWeight : 0.5
  // 优先级权重同理：正数才有效，0 在后端被当成"未配置"回落默认 0.5。
  const priorityWeight = Number(editForm.config.group_scheduling_election_priority_weight)
  editForm.config.group_scheduling_election_priority_weight =
    Number.isFinite(priorityWeight) && priorityWeight > 0 ? priorityWeight : 0.5
  // 优先级计分的全局开关收敛成真正的布尔（旧配置没有这个键，读回来是 undefined）；
  // 分组名单必须序列化成数组而非省略——后端整块覆盖 config_json，省略等于保留旧值。
  editForm.config.group_scheduling_election_priority_enabled_global =
    editForm.config.group_scheduling_election_priority_enabled_global === true
  editForm.config.group_scheduling_election_priority_enabled_group_ids = normalizePositiveAccountIDs(
    editForm.config.group_scheduling_election_priority_enabled_group_ids
  )
  editForm.config.group_scheduling_election_priority_disabled_group_ids = normalizePositiveAccountIDs(
    editForm.config.group_scheduling_election_priority_disabled_group_ids
  )
  // 阈值同理，且必须是正整数：0 在后端归一化里被当成"未配置"回落默认 2。
  const failureThreshold = Math.floor(Number(editForm.config.group_scheduling_election_failure_threshold))
  editForm.config.group_scheduling_election_failure_threshold =
    Number.isFinite(failureThreshold) && failureThreshold > 0 ? failureThreshold : 2
  // 迟滞比例：正数才有效，0 在后端被当成"未配置"回落默认 0.15；想近乎关闭请填极小正数。
  const switchMargin = Number(editForm.config.group_scheduling_election_switch_margin)
  editForm.config.group_scheduling_election_switch_margin =
    Number.isFinite(switchMargin) && switchMargin > 0 ? switchMargin : 0.15
  // 平均窗口与最少样本必须是正整数：0 在后端被当成"未配置"回落默认 30 / 3。
  const latencyWindow = Math.floor(Number(editForm.config.group_scheduling_election_latency_window_minutes))
  editForm.config.group_scheduling_election_latency_window_minutes =
    Number.isFinite(latencyWindow) && latencyWindow > 0 ? latencyWindow : 30
  const latencyMinSamples = Math.floor(Number(editForm.config.group_scheduling_election_latency_min_samples))
  editForm.config.group_scheduling_election_latency_min_samples =
    Number.isFinite(latencyMinSamples) && latencyMinSamples > 0 ? latencyMinSamples : 3
  // 次数封顶值：正整数才有效，0 在后端被当成"未配置"回落默认 10；上限与后端常量保持一致（100）。
  const countScoreCap = Math.floor(Number(editForm.config.group_scheduling_election_count_score_cap))
  editForm.config.group_scheduling_election_count_score_cap =
    Number.isFinite(countScoreCap) && countScoreCap > 0 ? Math.min(countScoreCap, 100) : 10
  // 必需模型：清洗成 { [groupID]: 模型列表 }，丢弃非正 groupID 与空列表；空 map 也保留（等于无强制要求）。
  editForm.config.group_scheduling_election_required_models =
    groupElectionRequiredModelsMap.value as unknown as Record<number, string[]>
  // 每组开启账号数的分组级覆盖：清洗成 { [groupID]: TopN }，丢弃非正 groupID 与非正 TopN（等于回落全局）；
  // 空 map 也必须序列化而非省略 —— 后端整块覆盖 config_json，省略等于保留旧值，"清掉全部覆盖"就存不成功。
  editForm.config.group_scheduling_election_top_n_by_group =
    groupElectionTopNByGroupMap.value as unknown as Record<number, number>
  // 演练：总开关收敛成真正的布尔（旧配置没有这个键，读回来是 undefined）；
  // 分组名单必须序列化成数组而不是省略 —— 后端是整块覆盖 config_json，
  // 省略等于保留旧值，"清掉全部演练分组"就永远保存不成功。
  editForm.config.group_scheduling_election_dry_run = electionDryRunAll.value
  editForm.config.group_scheduling_election_dry_run_group_ids = normalizePositiveAccountIDs(
    editForm.config.group_scheduling_election_dry_run_group_ids
  )
  // 异常推送：总开关同样收敛成真正的布尔（旧配置没有这个键，读回来是 undefined）；
  // 分组覆盖表必须序列化 —— 后端整块覆盖 config_json，省略等于保留旧值，"清掉全部覆盖"就存不成功。
  editForm.config.group_scheduling_election_alert_enabled = electionAlertEnabled.value
  editForm.config.group_scheduling_election_alert_group_overrides = electionAlertOverrides.value
  // 分组默认账号：同样必须序列化（空对象也要写成 {}），否则"清掉某个分组的默认账号"存不成功。
  editForm.config.group_scheduling_election_default_account_by_group =
    groupElectionDefaultAccountMap.value as unknown as Record<number, number>
}



async function openRateGuardGroups() {
  rateGuardGroupsVisible.value = true
  rateGuardGroupSearch.value = ''
  rateGuardGroupDisabledOnly.value = false
  rateGuardGroupPlatformFilter.value = []
  if (rateGuardGroups.value.length > 0) {
    return
  }
  loadingRateGuardGroups.value = true
  try {
    // 用含停用分组的接口：已关闭的分组如果此时正被停用，仍要能看见开关状态，
    // 否则配置会变成"看不见但依然生效"的幽灵项。
    rateGuardGroups.value = await listAllGroups()
  } catch (err) {
    appStore.showError(extractApiErrorMessage(err, '加载分组失败'))
  } finally {
    loadingRateGuardGroups.value = false
  }
}

function closeRateGuardGroups() {
  rateGuardGroupsVisible.value = false
}

function rateGuardGroupIsDisabled(groupID: number): boolean {
  return accountRateGuardDisabledGroupIDs.value.includes(groupID)
}

function toggleRateGuardGroup(groupID: number) {
  // 面板里勾选 = 参与守护；勾掉的才进配置，因此写回的是"关闭列表"。
  const disabled = accountRateGuardDisabledGroupIDs.value
  const next = disabled.includes(groupID)
    ? disabled.filter(id => id !== groupID)
    : [...disabled, groupID]
  editForm.config.account_rate_guard_disabled_group_ids = normalizePositiveAccountIDs(next)
}

function enableAllRateGuardGroups() {
  const current = accountRateGuardDisabledGroupIDs.value
  // 一次清空多个分组的开关是不可逆的（弹窗内没有撤销入口），
  // 所以先确认再执行：误点一下就把几十个分组全部放回守护，用户很难察觉自己改了什么。
  if (current.length > 1) {
    const confirmed = window.confirm(`将 ${current.length} 个分组全部恢复为参与守护？此操作在保存任务后生效。`)
    if (!confirmed) {
      return
    }
  }
  editForm.config.account_rate_guard_disabled_group_ids = []
  if (current.length > 0) {
    appStore.showSuccess(`已将 ${current.length} 个分组恢复为参与守护，保存任务后生效`)
  }
}

async function openElectionGroups() {
  electionGroupsVisible.value = true
  // 筛选与勾选由弹窗自己在打开时重置 —— 那几项状态已随弹窗搬进共享组件
  // （上游账号页复用同一个弹窗），页面这边不再持有，也就不能在这里清。
  // 分组默认账号的下拉需要「分组 → 成员账号」这份数据，与健康守护账号弹窗共用同一份缓存，
  // 已加载过就不再请求。放在分组加载之外：两者互不依赖，没必要串行等。
  const accountsPromise = ensureHealthGuardAccountCandidatesLoaded().catch(() => {
    // 账号候选是附加信息：拿不到只影响默认账号下拉，不该把整个分组配置弹窗拖垮。
  })
  if (rateGuardGroups.value.length === 0) {
    loadingRateGuardGroups.value = true
    try {
      // 含停用分组：已关闭择优的分组若此刻正被停用，仍要能看见开关状态。
      rateGuardGroups.value = await listAllGroups()
    } catch (err) {
      appStore.showError(extractApiErrorMessage(err, '加载分组失败'))
    } finally {
      loadingRateGuardGroups.value = false
    }
  }
  await accountsPromise
}

function closeElectionGroups() {
  electionGroupsVisible.value = false
}

async function openHealthGuardAccounts() {
  try {
    await ensureHealthGuardAccountCandidatesLoaded()
  } catch (err) {
    appStore.showError(extractApiErrorMessage(err, '加载健康守护账号失败'))
    return
  }
  healthGuardAccountsVisible.value = true
}

function closeHealthGuardAccounts() {
  healthGuardAccountsVisible.value = false
}









// 批量间隔：解析当前输入框的秒数，非法（空/非整数/小于 60）时返回 null，供按钮禁用与函数守卫共用同一判据。
// 批量操作只作用于「当前筛选结果里已勾选（行首勾选框）且可用」的账号：把筛选当子集选择器，
// 就能筛选 A 组→设 300→应用、筛选 B 组→设 600→应用，两批互不覆盖。
// 不可用账号运行时直接记为不可用、不参与检查，故排除在批量作用范围外。

// 全选 / 取消全选只切换「当前筛选结果」里可用账号的**勾选状态**（多选），不动「参与守护」：
// 否则点一下「全选」就会把几十个账号默默纳入守护，而纳入守护是有副作用的业务配置。
// 批量纳入/移出守护改由「批量设置」弹窗的「参与守护」项完成，用户需要显式选一次。

// 取消勾选「当前筛选结果」中的账号：只清勾选，保留账号已有的守护配置
// （要连配置一起清掉请用「批量设置 → 移出守护」）。

// 把批量间隔应用到「当前筛选结果里已勾选且可用」的账号，保存任务后生效。

// 清除「当前筛选结果里已勾选」账号的间隔覆盖，回落到「按倍率区间 / 每轮都测」的默认行为。

// 全局「修改调度」开关：只有显式 false 才算关闭（旧配置里没有这个键），与后端归一化口径一致。

// 账号级开关显示的是「最终生效值」：有显式覆盖就用覆盖值，否则跟随全局默认。

// 与全局默认一致时不写覆盖值（等于「跟随全局」）；只有不一致才落库成显式覆盖。
// 不写 false 占位：留着占位会让这个账号永远跟不上全局开关的后续变更。

// 悬浮提示要能说清「这一行是覆盖还是跟随全局」，否则两个开关状态一样时分不清。

// ── 批量设置（参与守护 / 修改调度 / 账号级测试模型 / 三项阈值）──
//
// 作用范围与批量间隔共用同一个 healthGuardBatchTargetRows：
// 「多选」就是「当前筛选结果里已勾选（行首勾选框）的可用账号」，两处若各算一套口径，
// 会出现「按钮说 12 个、实际只改了 8 个」这类对不上的情况。

// 参与守护：不修改 / 纳入 / 移出。
// 原先工具栏的「全选筛选结果 / 取消全选」顺带承担了批量纳入/移出守护；那两个按钮改成只管勾选后，
// 这条能力平移到这里 —— 用户必须显式选一次，不会因为点「全选」就把几十个账号默默纳入守护。
// 「移出」复用 removeHealthGuardAccount：与单行关掉开关一致，会一并清空该账号的模型/间隔/阈值/调度覆盖。

// 修改调度：不修改 / 跟随全局（清掉账号级覆盖）/ 强制开启 / 强制关闭。
// 「跟随全局」必须单独留一项 —— 全局开关改过之后，被批量写死 true/false 的账号否则再也回不到跟随状态。

// 账号级测试模型：不修改 / 清除覆盖 / 批量目标涉及平台的模型并集。
// 账号级覆盖只能是一个模型值，跨平台批量设置本身就需要人工确认模型可用，
// 因此这里给并集而不是按平台分组（按平台分组等于把「批量」拆成若干次单平台设置）。




// 阈值输入解析：空或非法一律返回 null（= 该项不修改），合法则返回正整数。


// 返回某平台的倍率区间规则数组（不存在则就地建空数组），供模板 v-model 直接编辑。





// 清洗每个平台的倍率区间规则：平台 key 转小写、丢弃间隔<60 或非法区间（min<0、max>0 且 min>=max）的规则，
// 并按下界升序排；某平台没有合法规则则整体丢弃。与后端归一化保持一致。

// 账号级阈值只保留正整数；留空或非法值等于不覆盖，交由全局阈值生效。




function supplierMonitorCheckedAt(item: SupplierProviderMonitorSyncItem): number {
  const timestamp = Date.parse(item.checked_at)
  return Number.isFinite(timestamp) ? timestamp : 0
}
function formatAvailability(value?: number): string {
  return typeof value === 'number' && Number.isFinite(value) ? `${value.toFixed(2)}%` : '-'
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
  try {
    await loadRuns()
  } catch (err) {
    appStore.showError(extractApiErrorMessage(err, '加载运行历史失败'))
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
  if (status === 'skipped') return '已跳过'
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

function intervalSecondsToCron(seconds: number): string | null {
  if (!Number.isInteger(seconds) || seconds < 1) return null
  return `@every ${seconds}s`
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
  gap: 8px;
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
  gap: 8px;
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

.sp-history-head-actions {
  display: inline-flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 10px;
}

/* 与分组管理页、账号页的「调度切换日志」入口同色（橙），一个功能一个颜色。 */
.sp-election-log-entry {
  border-color: color-mix(in srgb, var(--sp-orange) 45%, var(--sp-line));
  background: color-mix(in srgb, var(--sp-orange) 10%, var(--sp-panel));
  color: var(--sp-orange);
}

.sp-election-log-entry:hover:not(:disabled) {
  border-color: color-mix(in srgb, var(--sp-orange) 62%, var(--sp-line));
  background: color-mix(in srgb, var(--sp-orange) 16%, var(--sp-panel));
  color: color-mix(in srgb, var(--sp-orange) 88%, #7c2d12);
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

/* 最近结果列的状态标签兼作详情入口：字体与行高显式归一，避免 button 默认值改变 pill 尺寸
   （字号仍由 .sp-status 提供）；hover 反馈用 currentColor 光环，跟随 good/warn/bad 语义色，
   且不参与边框/底色的特异性竞争。 */
.sp-status-action {
  font-family: inherit;
  line-height: inherit;
  cursor: pointer;
  transition: box-shadow 0.15s ease;
}

.sp-status-action:not(:disabled):hover,
.sp-status-action:not(:disabled):focus-visible {
  box-shadow: 0 0 0 3px color-mix(in srgb, currentColor 16%, transparent);
}

.sp-status-action:disabled {
  cursor: default;
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
  grid-template-columns: repeat(2, minmax(0, 1fr));
  min-height: 0;
  /* 高度交给 .modal-body 的 flex 分配（见下方 `:has(.sp-edit-dialog) .modal-body` 那条）。
     这里原先有一条 max-height: 72vh，是弹窗还「按内容自适应高度」的时代留下的；
     现在 3 档弹窗高度已经是 calc(100dvh - 2rem)，两个上限叠在一起会在内容区下方空出一条：
     实测 1584x1105 视口下 body 可用 897px，表单被 72vh(=795.6px) 卡住 ⇒ 空出约 101px，
     而且表单自己还在滚动，观感就是「内容在滚、下面却还有一块白」。
     同模块的 .sp-health-guard-account-list / .sp-rate-guard-group-list 都已是「不设独立上限、
     吃满父容器剩余空间」的写法，这条跟上它们。 */
  flex: 1 1 auto;
  gap: 14px;
  overflow: auto;
  padding: 4px 2px 12px;
}

.sp-edit-summary {
  grid-column: 1 / -1;
  display: grid;
  grid-template-columns: repeat(3, minmax(0, 1fr));
  border: 1px solid var(--sp-soft);
  border-radius: 14px;
  background: color-mix(in srgb, var(--sp-blue) 4%, var(--sp-panel));
}

.sp-edit-summary > div {
  min-width: 0;
  border-left: 1px solid var(--sp-soft);
  padding: 10px 14px;
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
  margin-top: 3px;
  overflow: hidden;
  color: var(--sp-text);
  font-size: 14px;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.sp-form-section {
  display: grid;
  gap: 12px;
  min-width: 0;
  border: 1px solid color-mix(in srgb, var(--sp-blue) 14%, var(--sp-line));
  border-radius: 14px;
  padding: 14px;
  background: color-mix(in srgb, var(--sp-blue) 3%, var(--sp-panel));
}

.sp-state-section {
  border-top: 0;
  padding-top: 0;
}

.sp-policy-section {
  grid-column: 1 / -1;
}

.sp-form-section-head {
  display: flex;
  align-items: flex-start;
  gap: 10px;
}

.sp-form-section-head > span {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  flex: 0 0 auto;
  min-width: 32px;
  height: 30px;
  border: 1px solid color-mix(in srgb, var(--sp-blue) 26%, var(--sp-line));
  border-radius: 9px;
  padding: 0 7px;
  color: var(--sp-blue);
  background: color-mix(in srgb, var(--sp-blue) 9%, var(--sp-panel));
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
  margin: 3px 0 0;
  line-height: 1.5;
}

.sp-form-grid {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 12px;
}

.sp-health-guard-policy-grid,
.sp-retention-grid {
  grid-template-columns: repeat(3, minmax(0, 1fr));
}

/* 分组择优的每个配置项都带一段说明文字，3 列（约 260px/列）会把说明挤成 5~6 行，
   读起来比不看还累；这里刻意退回 2 列（约 400px/列）。窄屏仍由 media query 收成 1 列。 */
.sp-group-election-policy-grid {
  grid-template-columns: repeat(2, minmax(0, 1fr));
}

.sp-toggle-field {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  min-height: 40px;
  color: var(--sp-text);
  font-size: 13px;
  font-weight: 600;
}

.sp-form-note {
  color: var(--sp-muted);
  font-size: 12px;
  line-height: 1.5;
}

.sp-message-detail {
  max-width: min(780px, 78vw);
  max-height: 68vh;
  white-space: pre-wrap;
  word-break: break-word;
  overflow: auto;
}

:global(.modal-content:has(.sp-health-guard-account-dialog)),
:global(.modal-content:has(.sp-multiplier-interval-dialog)),
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

:global(.dark .modal-content:has(.sp-health-guard-account-dialog)),
:global(.dark .modal-content:has(.sp-multiplier-interval-dialog)),
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

:global(.modal-content:has(.sp-health-guard-account-dialog) .modal-header),
:global(.modal-content:has(.sp-multiplier-interval-dialog) .modal-header),
:global(.modal-content:has(.sp-edit-dialog) .modal-header),
:global(.modal-content:has(.sp-run-detail) .modal-header) {
  border-bottom-color: var(--sp-line);
  background: var(--sp-panel);
}

:global(.modal-content:has(.sp-health-guard-account-dialog) .modal-title),
:global(.modal-content:has(.sp-edit-dialog) .modal-title),
:global(.modal-content:has(.sp-run-detail) .modal-title) {
  color: var(--sp-text);
}

:global(.modal-content:has(.sp-health-guard-account-dialog)),
:global(.modal-content:has(.sp-edit-dialog)),
:global(.modal-content:has(.sp-run-detail)) {
  max-height: min(95vh, calc(100dvh - 16px));
}

/* 结果详情弹窗按视口给到 80% 宽：这个弹窗要同时承载 8 类任务的明细（表格列多、字段杂），
   而 BaseDialog 的 extra-wide 档最宽只到 xl:max-w-6xl = 1152px ——
   在 1584 视口下只占 73%，横向空间明显浪费。
   只命中「装着结果详情」的那一个 modal-content，同样用 extra-wide 的其它弹窗
   （倍率区间、分组切换日志等）不受影响。
   权重说明：本选择器等于 2 个类（.modal-content + :has() 内的 .sp-run-detail），
   高于 Tailwind 的 .xl\:max-w-6xl（1 个类），所以不依赖源码顺序也能生效。
   只在 >=640px 生效：更窄时 BaseDialog 的档位本就是 w-full（铺满可用宽度），
   强行 80vw 反而会把手机上的弹窗改窄。 */
@media (min-width: 640px) {
  :global(.modal-content:has(.sp-run-detail)) {
    width: 80vw;
    max-width: 80vw;
  }
}

/* 电脑端：任务编辑弹窗、健康守护账号配置弹窗、分组配置弹窗横向铺满屏幕（只留 overlay 自带的四周外边距）。
   三者是同一层级的配置入口（后两者由前者内的按钮打开），尺寸保持一致才不会出现层级间的跳变。
   只在 >=768px 生效，窄屏沿用 BaseDialog 的 full 档表现，避免手机上挤掉内容宽度。 */
@media (min-width: 768px) {
  :global(.modal-content:has(.sp-health-guard-account-dialog)),
  :global(.modal-content:has(.sp-edit-dialog)) {
    width: calc(100vw - 2rem);
    max-width: calc(100vw - 2rem);
  }

  /* 编辑弹窗按「最宽那行的列数」分档，不再一律铺满。
     这个弹窗是多任务共用的，不同 task_code 的内容量差一倍：
     账号倍率守护内容总高 481px / 可用高 729px，铺满后底部留 248px 空白（约 34%），
     策略卡片里从文字到按钮更是空出 1012px；健康守护有 7 个输入框排 3 列（内容高 721px，几乎填满）。
     决定宽度的是最宽那一行的列数，不是区块个数 —— 3 列要放得下每列的「标签 + 输入框」，
     2 列则 980px 就够（每列约 450px）。档位值与 data-grid-cols 由脚本同步而来，
     新增任务只要落进这两档就自动适配，不必再逐任务写死。
     这条必须排在上一条之后 —— 同层选择器权重相同，靠源码顺序取胜。 */
  :global(.modal-content:has(.sp-edit-dialog[data-grid-cols="2"])) {
    width: min(980px, calc(100vw - 2rem));
    max-width: min(980px, calc(100vw - 2rem));
  }

  /* 3 档不再「一律铺满」：铺满时宽度只取决于视口，屏幕越宽越散
     （实测 1584 视口下 1552px，每个字段的输入框横跨 700px 却只放一个数字）。
     给一个 1280px 上限（= BaseDialog full 档的 xl:max-w-7xl，回到框架自己的尺度）。
     收窄的代价实测过：健康守护的 3 列区块高度 494px → 494px 完全不变（每列 384px
     仍放得下「标签 + 输入框」，没触发上面担心的折行）；分组择优的说明文字各多占一行，
     表单总高 1224 → 1291（+67px），而它本来就要滚动，边际影响可接受。 */
  :global(.modal-content:has(.sp-edit-dialog[data-grid-cols="3"])) {
    width: min(1280px, calc(100vw - 2rem));
    max-width: min(1280px, calc(100vw - 2rem));
  }

  /* 分组择优调度：编辑弹窗按 90% 视口宽。
     本走 3 列档（最宽 1280px），在宽屏上偏窄、分组择优的列与说明文字排得挤。
     用组内唯一的类精确命中：只有分组择优的 form 里有 .sp-group-election-policy-grid，
     不波及其它任务的编辑弹窗。
     选择器含 :has()（2 个类），权重高于 Tailwind 的 .xl\:max-w-7xl，
     不依赖源码顺序即可生效。只在 >=768px 生效，窄屏沿用 BaseDialog 档位，避免手机上挤掉内容。 */
  :global(.modal-content:has(.sp-edit-dialog .sp-group-election-policy-grid)) {
    width: 90vw;
    max-width: 90vw;
  }
}

/* 高度同步收窄：2 列档宽度降到 980px 后再顶满 100dvh 会显得又瘦又长。
   640px 对 481px 的内容留出约 160px 余量，够容纳策略卡片换行；
   3 列档内容更高，沿用上面的 100dvh 兜底。
   屏幕不够高时仍由 calc(100dvh - 2rem) 兜住。 */
:global(.modal-content:has(.sp-edit-dialog[data-grid-cols="2"])) {
  height: min(640px, calc(100dvh - 1rem));
  max-height: min(640px, calc(100dvh - 1rem));
}

@media (min-width: 640px) {
  :global(.modal-content:has(.sp-edit-dialog[data-grid-cols="2"])) {
    height: min(640px, calc(100dvh - 2rem));
    max-height: min(640px, calc(100dvh - 2rem));
  }
}

:global(.modal-content:has(.sp-health-guard-account-dialog) .modal-body),
:global(.modal-content:has(.sp-multiplier-interval-dialog) .modal-body),
:global(.modal-content:has(.sp-edit-dialog) .modal-body),
:global(.modal-content:has(.sp-run-detail) .modal-body) {
  display: flex;
  min-height: 0;
  flex: 1 1 auto;
  flex-direction: column;
  overflow: hidden;
  background: var(--sp-panel);
}



/* Input 组件的说明文字是全站通用样式（style.css 的 .input-hint = 12px 固定灰字），
   对本弹窗里动辄两三百字的策略说明太挤：12px 中文行距只有 1.33，读起来很累。
   在页面层覆盖，只作用于编辑任务弹窗，不动通用组件与全局样式 ——
   字号提到 13px（与模块 .sp-toggle-field 同档），行高放开到 1.7，
   颜色改用跟主题的 --sp-muted（浅色 #607089 对白底 5.2:1、深色 #a8b6ca 对深底 7:1，
   都优于原 gray-500 的 4.8:1），比原来的固定灰更清楚且不再与主题脱节。 */
:global(.modal-content:has(.sp-edit-dialog) .input-hint) {
  color: var(--sp-muted);
  font-size: 13px;
  line-height: 1.7;
}

/* 字段标签改用模块自己的蓝。原先它跟随全站通用的 .input-label
   （浅色 gray-700 / 深色 gray-300），与本弹窗里同样走灰阶的说明文字只差一档明度 ——
   深色下更是 #d1d5db 对 #a8b6ca，几乎分不出来；而这些字段的说明动辄三四行，
   扫视时找不到「下一个字段从哪开始」。换成饱和蓝后标签成了可跳读的锚点。
   浅色取 --sp-blue（#2563eb，对白底 5.17:1）；深色必须另给亮蓝 ——
   --sp-blue 在深色块里没有重定义，仍是 #2563eb，在 #172033 上只有 3.5:1。
   同样只作用于编辑任务弹窗，不动通用组件与全局样式。 */
:global(.modal-content:has(.sp-edit-dialog) .input-label) {
  color: var(--sp-blue);
}

:global(.dark .modal-content:has(.sp-edit-dialog) .input-label) {
  color: #60a5fa;
}

:global(.modal-content:has(.sp-health-guard-account-dialog) .modal-body),
:global(.modal-content:has(.sp-rate-guard-group-dialog) .modal-body) {
  overflow: hidden;
}

/* 任务编辑弹窗、账号配置弹窗、分组配置弹窗：让内容区吃满可用高度，列表在剩余空间内滚动。
   高度同样顶到 overlay 留白之内，与宽度一起构成「近全屏」。
   这条必须排在上面那条 max-height: min(95vh, …) 之后 —— 两者选择器权重相同，
   靠源码顺序取胜。 */
:global(.modal-content:has(.sp-health-guard-account-dialog)),
:global(.modal-content:has(.sp-edit-dialog)) {
  height: calc(100dvh - 1rem);
  max-height: calc(100dvh - 1rem);
}
@media (min-width: 640px) {
  :global(.modal-content:has(.sp-health-guard-account-dialog)),
  :global(.modal-content:has(.sp-edit-dialog)) {
    height: calc(100dvh - 2rem);
    max-height: calc(100dvh - 2rem);
  }
}

:global(.modal-content:has(.sp-health-guard-account-dialog) .modal-footer),
:global(.modal-content:has(.sp-multiplier-interval-dialog) .modal-footer),
:global(.modal-content:has(.sp-edit-dialog) .modal-footer),
:global(.modal-content:has(.sp-run-detail) .modal-footer) {
  border-top-color: var(--sp-line);
  background: var(--sp-panel);
}

/* 分组配置弹窗是独立 Teleport 出来的 modal-content，不是编辑弹窗的后代，
   因此上面那些按 .sp-edit-dialog 挂的规则全部够不到它 ——
   变量、配色、body 布局、尺寸都会缺失，表现为"文字是默认黑、背景透明、列表不滚动"。
   这里单独给它一份，选择器独立成组，避免改动上面既有的断言锚点。 */
:global(.modal-content:has(.sp-rate-guard-group-dialog)) {
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
  height: min(760px, calc(100dvh - 1rem));
  max-height: min(760px, calc(100dvh - 1rem));
}

:global(.dark .modal-content:has(.sp-rate-guard-group-dialog)) {
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

:global(.modal-content:has(.sp-rate-guard-group-dialog) .modal-header) {
  border-bottom-color: var(--sp-line);
  background: var(--sp-panel);
}

:global(.modal-content:has(.sp-rate-guard-group-dialog) .modal-title) {
  color: var(--sp-text);
}

:global(.modal-content:has(.sp-rate-guard-group-dialog) .modal-body) {
  display: flex;
  min-height: 0;
  flex: 1 1 auto;
  flex-direction: column;
  overflow: hidden;
  background: var(--sp-panel);
}

:global(.modal-content:has(.sp-rate-guard-group-dialog) .modal-footer) {
  border-top-color: var(--sp-line);
  background: var(--sp-panel);
}













/* 宽度按 90% 视口给：用户要求把「配置分组」弹窗放宽到 90%。
   这个弹窗里是"摘要三块 + 搜索 + 每组多列开关"，列宽越大越好排。
   选择器与上方分组择优编辑弹窗那条同权重、同媒体条件，靠源码顺序取胜
   （本块排更后面，故覆盖上面 90vw 同名选择器里不含 .sp-rate-guard-group-dialog 的部分不影响）——
   它是真正给分组弹窗生效的那一条。
   只 <= 100vw 生效、窄屏沿用 BaseDialog 的 full 档表现。 */
@media (min-width: 768px) {
  :global(.modal-content:has(.sp-rate-guard-group-dialog)) {
    width: 90vw;
    max-width: 90vw;
  }
}

/* 高度按 95% 视口给：用户要求把「配置分组」弹窗加高到 95%。
   分组择优这个弹窗内容最多（三块摘要 + 搜索/筛选 + 每组多列开关 + 必需模型输入），
   高度给足才能少滚动、多看几行。列表仍由 .modal-body 内部滚动承接，
   不设固定上限、也不在窄屏缩回，直接铺到视口高度的 95%。 */
@media (min-width: 640px) {
  :global(.modal-content:has(.sp-rate-guard-group-dialog)) {
    height: 95vh;
    max-height: 95vh;
  }
}

.sp-run-detail {
  --sp-result-accent: var(--sp-cyan);
  display: grid;
  /* 宽度跟随父容器，不再自设上限。原来的 max-width: min(1120px, 86vw) 是弹窗还在
     BaseDialog extra-wide 档（最宽 1152px、内容区 1104px）时留下的，那个上限从不生效；
     弹窗按视口放宽到 80vw 后内容区变成 1219px，内容却仍卡在 1120px，
     而 .modal-body 是 flex column + 默认 stretch（子项从交叉轴起点即左边起算），
     表现为「弹窗右侧空出一条」。这里靠 stretch 铺满即可，
     显式写 100% 是为了父容器布局将来变化时仍能兜住。 */
  max-width: 100%;
  /* 高度同样交给父容器：原来的 max-height: 72vh 会在弹窗还有余高时提前截断内容区
     （1105px 视口下只到 795px，而弹窗可用约 1049px），滚动条被提到半空、下方空一截。
     改 100% 后两种情况都不留白：父容器高度确定时（弹窗被自身 max-height 封顶）正好等于
     父容器高度；父容器高度为 auto 时百分比无法解析、等价于取消上限，此时靠
     flex-shrink + overflow: auto 把内容压回弹窗内滚动。
     ⚠️ 这里不能照抄编辑弹窗的 `flex: 1 1 auto` —— 结果详情弹窗没有固定 height，
     `.modal-content` 高度是 auto，flex-grow 拿不到剩余空间，真正起作用的是 shrink。 */
  max-height: 100%;
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
  margin-top: 8px;
  padding-top: 10px;
}

/* 明细区里的区块比执行结论区多，间距再紧一档。
   必须单独成一条规则：上面那块被 spec 的正则钉住了「正好三个声明」的形状
   （border-top / margin-top / padding-top 后直接收尾），并进去会直接打爆那条断言。 */
.sp-detail-content {
  gap: 8px;
}

.sp-detail-section-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 16px;
}

.sp-detail-section-head h3 {
  margin: 0;
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
  /* 一行 6 格（原来是 3 列 × 2 行，省掉一整行）。等分就够：
     最长的是时间戳（19 字符约 152px）和任务中文名（最长 9 字约 126px），
     每列可用 185px（内容区 1219px 减左右 padding）都放得下。
     「任务」格显示的是中文名而非原始代号 —— 代号最长 35 字符（约 280px）在等分列里会折行，
     而完整代号在弹窗标题栏已经显示了，这里不必重复。
     1024px 以下仍由媒体查询切回 2 列 / 1 列，这里只管桌面。 */
  grid-template-columns: repeat(6, minmax(0, 1fr));
  row-gap: 18px;
  border-bottom: 1px solid var(--sp-line);
  padding: 4px 0 18px;
}

.sp-summary-item {
  min-width: 0;
  border-left: 1px solid var(--sp-line);
  padding: 2px 18px 4px;
}

.sp-summary-item:first-child {
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
  gap: 8px;
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

.sp-health-guard-account-card {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 16px;
  border: 1px solid var(--sp-soft);
  border-radius: 12px;
  padding: 14px 16px;
  background: color-mix(in srgb, var(--sp-blue) 4%, var(--sp-panel));
}

.sp-health-guard-account-card > div {
  display: grid;
  gap: 4px;
}

.sp-health-guard-account-card strong,
.sp-health-guard-dialog-section-head strong,
.sp-health-guard-platform-model-grid strong {
  color: var(--sp-text);
  font-size: 13px;
}

.sp-health-guard-account-card span,
.sp-health-guard-dialog-section-head span,
.sp-health-guard-platform-model-grid span {
  color: var(--sp-muted);
  font-size: 12px;
}

.sp-health-guard-account-card strong.sp-health-guard-card-count {
  color: var(--sp-blue);
  font-size: inherit;
  font-variant-numeric: tabular-nums;
}

.sp-health-guard-account-card .sp-button.sp-health-guard-config-button {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  gap: 6px;
  /* 这个按钮是「健康守护策略」区唯一的操作入口，卡片里也只有它一个可点目标，
     继承 .small 的 32px 高 / 12px 字在这一片最容易漏看，因此在此处单点放大。
     量级取常规 .sp-button（40px / 14px）与 .small 之间，并加大左右内边距配合文字。 */
  min-height: 2.5rem;
  padding: 0.5rem 1rem;
  font-size: 0.8125rem;
  border-color: color-mix(in srgb, var(--sp-blue) 45%, var(--sp-line));
  background: color-mix(in srgb, var(--sp-blue) 9%, var(--sp-panel));
  color: var(--sp-blue);
  font-weight: 600;
  transition: border-color 160ms ease, background-color 160ms ease, color 160ms ease, transform 120ms ease;
}

.sp-health-guard-account-card .sp-button.sp-health-guard-config-button:hover {
  border-color: var(--sp-blue);
  background: color-mix(in srgb, var(--sp-blue) 14%, var(--sp-panel));
  color: var(--sp-blue);
}

.sp-health-guard-account-card .sp-button.sp-health-guard-config-button:active {
  transform: translateY(1px);
}

.sp-health-guard-account-card .sp-button.sp-health-guard-config-button:focus-visible {
  outline: 2px solid color-mix(in srgb, var(--sp-blue) 30%, transparent);
  outline-offset: 2px;
}

/* 账号倍率守护的「不参与守护的分组」卡片：与健康守护账号卡片同构，
   只在配色上换成青色系，避免两个卡片在同一弹窗里被误认成同一个入口。 */
.sp-rate-guard-scope-card {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 16px;
  border: 1px solid var(--sp-soft);
  border-radius: 12px;
  padding: 14px 16px;
  background: color-mix(in srgb, var(--sp-cyan) 4%, var(--sp-panel));
}

.sp-rate-guard-scope-card > div {
  display: grid;
  gap: 4px;
}

.sp-rate-guard-scope-card strong {
  color: var(--sp-text);
  font-size: 13px;
}

.sp-rate-guard-scope-card span {
  color: var(--sp-muted);
  font-size: 12px;
}

.sp-rate-guard-scope-card strong.sp-rate-guard-scope-count {
  color: var(--sp-cyan);
  font-size: inherit;
  font-variant-numeric: tabular-nums;
}

.sp-rate-guard-scope-card .sp-button.sp-rate-guard-scope-config-button {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  gap: 6px;
  min-height: 2.5rem;
  padding: 0.5rem 1rem;
  font-size: 0.8125rem;
  border-color: color-mix(in srgb, var(--sp-cyan) 45%, var(--sp-line));
  background: color-mix(in srgb, var(--sp-cyan) 9%, var(--sp-panel));
  color: var(--sp-cyan);
  font-weight: 600;
  transition: border-color 160ms ease, background-color 160ms ease, color 160ms ease, transform 120ms ease;
}

.sp-rate-guard-scope-card .sp-button.sp-rate-guard-scope-config-button:hover {
  border-color: var(--sp-cyan);
  background: color-mix(in srgb, var(--sp-cyan) 14%, var(--sp-panel));
  color: var(--sp-cyan);
}

.sp-rate-guard-scope-card .sp-button.sp-rate-guard-scope-config-button:active {
  transform: translateY(1px);
}

.sp-rate-guard-scope-card .sp-button.sp-rate-guard-scope-config-button:focus-visible {
  outline: 2px solid color-mix(in srgb, var(--sp-cyan) 30%, transparent);
  outline-offset: 2px;
}

.sp-rate-guard-group-dialog {
  display: flex;
  flex: 1 1 auto;
  flex-direction: column;
  min-height: 0;
}

/* 工作区自己管理滚动：摘要和工具栏固定，只有列表滚动，
   否则分组多的时候连摘要都会被滚走，用户看不到"已关闭 N 个"这个最关键的判断依据。 */
.sp-rate-guard-group-workspace {
  display: flex;
  flex: 1 1 auto;
  flex-direction: column;
  min-height: 0;
  border: 1px solid var(--sp-line);
  border-radius: 12px;
  overflow: hidden;
  background: var(--sp-panel);
}

/* 摘要与健康守护弹窗同构：顶部 3px 色条 + 圆点标签 + 大数字，
   两者是同一层级的配置入口，视觉语言必须一致，用户才不会觉得是两个功能。
   标签与数字贴在一起（不 space-between）：三块在近全屏宽度下会被拉开 500px 以上，
   数字散落在半空中反而不成组，读的时候要靠视线来回跳。 */
.sp-rate-guard-group-summary {
  display: grid;
  flex: 0 0 auto;
  grid-template-columns: repeat(3, minmax(0, 1fr));
  border-bottom: 1px solid var(--sp-line);
  background: color-mix(in srgb, var(--sp-soft) 22%, transparent);
}

.sp-rate-guard-group-summary article {
  --sp-summary-accent: var(--sp-cyan);

  position: relative;
  display: flex;
  align-items: baseline;
  justify-content: flex-start;
  min-width: 0;
  gap: 8px;
  border-left: 1px solid var(--sp-line);
  padding: 12px 14px 11px;
}

.sp-rate-guard-group-summary article:first-child {
  border-left: 0;
}

.sp-rate-guard-group-summary article:nth-child(1) {
  --sp-summary-accent: var(--sp-cyan);
}

.sp-rate-guard-group-summary article:nth-child(2) {
  --sp-summary-accent: var(--sp-muted);
}

.sp-rate-guard-group-summary article:nth-child(3) {
  --sp-summary-accent: var(--sp-blue);
}

.sp-rate-guard-group-summary article::before {
  content: '';
  position: absolute;
  top: 0;
  right: 0;
  left: 0;
  height: 3px;
  background: var(--sp-summary-accent);
  opacity: 0.85;
}

.sp-rate-guard-group-summary article span {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  color: color-mix(in srgb, var(--sp-summary-accent) 64%, var(--sp-muted));
  font-size: 12px;
}

.sp-rate-guard-group-summary article span::before {
  content: '';
  width: 6px;
  height: 6px;
  border-radius: 999px;
  background: var(--sp-summary-accent);
  opacity: 0.9;
}

.sp-rate-guard-group-summary article strong {
  color: var(--sp-summary-accent);
  font-size: 18px;
  font-weight: 750;
  font-variant-numeric: tabular-nums;
  line-height: 1;
}

/* 有关闭项时用琥珀色警示：这是"有分组不参与守护"的唯一可见信号。
   但只染顶部色条与数字，不铺满整块底色 —— 铺满会让它比"参与守护 19"更抢眼，
   而那句才是主体数量；附注性质的告警不该喧宾夺主。 */
.sp-rate-guard-group-summary article.warning {
  --sp-summary-accent: var(--sp-amber);

  background: color-mix(in srgb, var(--sp-amber) 6%, transparent);
}

.sp-rate-guard-group-toolbar {
  display: flex;
  flex: 0 0 auto;
  flex-wrap: wrap;
  align-items: center;
  justify-content: space-between;
  gap: 8px 12px;
  padding: 10px 14px;
}

.sp-rate-guard-group-filters {
  display: flex;
  flex: 1 1 auto;
  align-items: center;
  gap: 10px;
  flex-wrap: wrap;
}

.sp-rate-guard-group-filters :deep(.form-field),
.sp-rate-guard-group-filters > div {
  width: 16rem;
}

.sp-rate-guard-group-selected-toggle {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  border: 1px solid var(--sp-line);
  border-radius: 8px;
  padding: 0.5rem 0.75rem;
  background: var(--sp-panel);
  color: var(--sp-muted);
  font-size: 12px;
  cursor: pointer;
  transition: border-color 160ms ease, color 160ms ease, background-color 160ms ease;
}

.sp-rate-guard-group-selected-toggle:hover {
  border-color: color-mix(in srgb, var(--sp-cyan) 30%, var(--sp-line));
  color: var(--sp-text);
}

.sp-rate-guard-group-selected-toggle strong {
  color: var(--sp-text);
  font-variant-numeric: tabular-nums;
}

.sp-rate-guard-group-selected-toggle.active {
  border-color: color-mix(in srgb, var(--sp-cyan) 50%, var(--sp-line));
  background: color-mix(in srgb, var(--sp-cyan) 8%, var(--sp-panel));
  color: var(--sp-cyan);
}

.sp-rate-guard-group-selected-toggle:focus-visible {
  outline: 2px solid color-mix(in srgb, var(--sp-cyan) 40%, transparent);
  outline-offset: 2px;
}

.sp-rate-guard-group-selected-toggle-mark {
  width: 8px;
  height: 8px;
  border-radius: 50%;
  background: currentColor;
  opacity: 0.5;
}

.sp-rate-guard-group-selected-toggle.active .sp-rate-guard-group-selected-toggle-mark {
  opacity: 1;
}

.sp-rate-guard-group-filter-result {
  flex: 0 0 auto;
  color: var(--sp-muted);
  font-size: 12px;
}

.sp-rate-guard-group-filter-result strong {
  color: var(--sp-text);
  font-variant-numeric: tabular-nums;
}

/* 批量操作条：夹在工具栏与列表之间，不随列表滚动 —— 滚到第 50 行还能直接点批量按钮。
   动作只在有勾选时展开：没勾选时先摆一排灰按钮，用户会以为能点。 */
.sp-group-election-batch-bar {
  display: flex;
  flex: 0 0 auto;
  flex-wrap: wrap;
  align-items: center;
  gap: 6px 10px;
  padding: 8px 14px;
  border-top: 1px solid var(--sp-line);
  background: color-mix(in srgb, var(--sp-cyan) 5%, var(--sp-panel));
}

.sp-group-election-batch-master {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  color: var(--sp-text);
  font-size: 12px;
  font-weight: 650;
  cursor: pointer;
}

.sp-group-election-batch-master input[type='checkbox'] {
  width: 16px;
  height: 16px;
  margin: 0;
  border-radius: 4px;
  /* 与列表行里的勾选框同一套外观：两者是同一个多选模型的两个入口，
     样式分叉会让人以为是两套互不相干的选中。 */
  accent-color: var(--sp-cyan);
  cursor: pointer;
}

.sp-group-election-batch-master input[type='checkbox']:focus-visible {
  outline: 2px solid color-mix(in srgb, var(--sp-cyan) 40%, transparent);
  outline-offset: 2px;
}

.sp-group-election-batch-count {
  color: var(--sp-muted);
  font-size: 12px;
}

.sp-group-election-batch-count strong {
  color: var(--sp-text);
  font-variant-numeric: tabular-nums;
}

.sp-group-election-batch-divider {
  width: 1px;
  height: 16px;
  background: var(--sp-line);
}

.sp-group-election-batch-label {
  color: var(--sp-muted);
  font-size: 12px;
  font-weight: 650;
}

.sp-group-election-batch-btn {
  border: 1px solid var(--sp-line);
  border-radius: 8px;
  padding: 0.2rem 0.6rem;
  background: var(--sp-panel);
  color: var(--sp-text);
  font-size: 12px;
  cursor: pointer;
  transition: border-color 160ms ease, color 160ms ease, background-color 160ms ease;
}

.sp-group-election-batch-btn:hover {
  border-color: color-mix(in srgb, var(--sp-cyan) 40%, var(--sp-line));
  background: color-mix(in srgb, var(--sp-cyan) 8%, var(--sp-panel));
  color: var(--sp-cyan);
}

.sp-group-election-batch-btn:focus-visible {
  outline: 2px solid color-mix(in srgb, var(--sp-cyan) 40%, transparent);
  outline-offset: 2px;
}

/* 「清空勾选」是收尾动作、不是一次批量改配置，因此不跟批量按钮抢视觉重量：
   贴右侧、只用文字下划线，避免被误当成第四个批量动作点下去。 */
.sp-group-election-batch-clear {
  margin-left: auto;
  border: 0;
  padding: 0.2rem 0.2rem;
  background: transparent;
  color: var(--sp-muted);
  font-size: 12px;
  text-decoration: underline;
  cursor: pointer;
}

.sp-group-election-batch-clear:hover {
  color: var(--sp-text);
}

/* 平台标签独占一行（flex-basis: 100%）。三个弹窗共用同一套控件：
   分组弹窗里搜索框 + 「仅看已关闭」已经占满第一行，标签挤进去会把搜索框压到没法输入，
   而平台数量还会随分组增长；账号弹窗里更要放在 grid 容器之外（flex-basis 在 grid 里不生效）。 */
.sp-platform-chip-row {
  display: flex;
  flex: 1 1 100%;
  flex-wrap: wrap;
  align-items: center;
  gap: 6px;
  min-width: 0;
}

/* 未选中一律保持中性灰：一屏里同时挂着七八个平台色标签，反而看不出"选中了哪几个"。
   选中态才上平台色，写法与「仅看已关闭」的 active 态同构（色描边 + 浅色底）。
   文字不跟着平台色走 —— ACCENT 取的是 500 系，在深色面板上做正文偏暗，
   改用底色和描边承载平台色，两种主题下都读得清。 */
.sp-platform-chip {
  display: inline-flex;
  align-items: center;
  gap: 5px;
  border: 1px solid var(--sp-line);
  border-radius: 8px;
  padding: 0.28rem 0.6rem;
  background: var(--sp-panel);
  color: var(--sp-muted);
  font-size: 12px;
  cursor: pointer;
  transition: border-color 160ms ease, color 160ms ease, background-color 160ms ease;
}

.sp-platform-chip:hover {
  border-color: color-mix(in srgb, var(--sp-chip-platform-color) 30%, var(--sp-line));
  color: var(--sp-text);
}

.sp-platform-chip[aria-pressed='true'] {
  border-color: color-mix(in srgb, var(--sp-chip-platform-color) 55%, var(--sp-line));
  background: color-mix(in srgb, var(--sp-chip-platform-color) 16%, var(--sp-panel));
  color: var(--sp-text);
  font-weight: 650;
}

.sp-platform-chip:focus-visible {
  outline: 2px solid color-mix(in srgb, var(--sp-chip-platform-color) 40%, transparent);
  outline-offset: 2px;
}

.sp-platform-chip strong {
  color: inherit;
  font-variant-numeric: tabular-nums;
  opacity: 0.7;
}

/* 列表整块滚动 + 行分隔线（与健康守护账号列表同构），
   比"一堆独立小卡片"更耐看，分组多时也能一眼扫完。
   不设 max-height：弹窗已近全屏，列表直接吃满摘要与工具栏之外的剩余空间，
   再叠加一层上限会让内容区下方空出一块，反而显得没铺满。 */
.sp-rate-guard-group-list {
  flex: 1 1 auto;
  min-height: 0;
  overflow: auto;
  overscroll-behavior: contain;
  border-top: 1px solid var(--sp-line);
  background: color-mix(in srgb, var(--sp-soft) 12%, var(--sp-panel));
  scrollbar-width: thin;
  scrollbar-color: color-mix(in srgb, var(--sp-line) 82%, transparent) transparent;
}

.sp-rate-guard-group-list::-webkit-scrollbar {
  width: 8px;
}

.sp-rate-guard-group-list::-webkit-scrollbar-thumb {
  border: 2px solid transparent;
  border-radius: 999px;
  background: color-mix(in srgb, var(--sp-line) 85%, transparent);
  background-clip: padding-box;
}

.sp-rate-guard-group-row {
  position: relative;
  display: flex;
  align-items: center;
  justify-content: space-between;
  min-width: 0;
  gap: 8px 12px;
  border-bottom: 1px solid var(--sp-line);
  padding: 10px 14px;
  background: var(--sp-panel);
  transition: background-color 160ms ease, box-shadow 160ms ease;
}

/* 分组择优调度弹窗的分组行改走网格：它比倍率守护行多了「健康锁定 / 演练 / 计优先级 / 参与择优」
   四个开关和「必需模型」输入。宽弹窗下面用 flex + space-between 时，每组名字长短、
   徽标、ID 占比都不固定，后面的开关列位置被挤得左摇右摆 —— 这行这么多列，
   必须把每一列宽度钉死，行与行才能对齐。
   判据取「参与择优」开关（每行都渲染），不能取「健康锁定」：后者是 v-else 分支，
   已关闭的行上没有它，那些行会退回 flex，同一张列表里出现两种排法。
   这个开关类只出现在择优弹窗，倍率守护弹窗的两列行不受影响。
   列宽：名字区不自设宽度(1fr 吃掉剩余)、四个开关各自 auto（文字/toggle 尺寸恒定，天然等宽），
   必需模型占满整行另起一行(grid-column 1/-1)，不与开关并排。 */
.sp-rate-guard-group-row:has(.sp-election-participate-toggle) {
  display: grid;
  grid-template-columns: minmax(0, 1fr) auto auto auto auto;
  align-items: center;
  column-gap: 14px;
}

/* 每一列都显式定位，不靠自动排布：已关闭的行少渲染「健康锁定 / 演练 / 计优先级」三个开关，
   自动排布会把「参与择优」顶到第 3 列，与参与中的行（第 5 列）横向错开 ——
   而这一列正是用户扫视"哪些分组算数"的锚点，错开就白做了。 */
.sp-rate-guard-group-row:has(.sp-election-participate-toggle) > .sp-rate-guard-group-choice {
  grid-column: 1;
}

.sp-rate-guard-group-row:has(.sp-election-participate-toggle) > .sp-rate-guard-group-status,
.sp-rate-guard-group-row:has(.sp-election-participate-toggle) > .sp-election-toggle-health {
  grid-column: 2;
}

.sp-rate-guard-group-row:has(.sp-election-participate-toggle) > .sp-election-toggle-dryrun {
  grid-column: 3;
}

.sp-rate-guard-group-row:has(.sp-election-participate-toggle) > .sp-election-toggle-priority {
  grid-column: 4;
}

.sp-rate-guard-group-row:has(.sp-election-participate-toggle) > .sp-election-participate-toggle {
  grid-column: 5;
}

/* 「每组开启数 + 必需模型 + 默认账号」合并成一行占满整行、另起一行：
   TopN 是窄数字框、必需模型吃掉剩余宽度、默认账号是窄下拉。 */
.sp-rate-guard-group-row:has(.sp-election-participate-toggle) > .sp-election-group-extra {
  grid-column: 1 / -1;
}

/* 「参与择优」是这一行的主状态，另外三个开关都从属于它 ——
   文字用正文色而不是次要灰，和左侧勾选框一起回答"这一行到底算不算数"。 */
.sp-rate-guard-group-row .sp-election-participate-toggle {
  color: var(--sp-text);
  font-weight: 700;
}

.sp-rate-guard-group-row:last-child {
  border-bottom: 0;
}

.sp-rate-guard-group-row:hover {
  background: color-mix(in srgb, var(--sp-cyan) 4%, var(--sp-panel));
}

/* 平台色条：3px 竖条，色值来自行上的 --sp-group-platform-color。
   与下面「已关闭」的灰条共用同一条竖线 —— 两者互斥，不该并排画两条：
   并排会在左边堆出双色块，反而看不出哪个状态优先。 */
.sp-rate-guard-group-row::before {
  content: '';
  position: absolute;
  top: 0;
  bottom: 0;
  left: 0;
  width: 3px;
  background: var(--sp-group-platform-color, transparent);
}

/* 已关闭：左侧色条 + 灰底。色条是关键 —— 视线扫过整列时，
   靠底色深浅分辨"关没关"太吃力，一条竖线能立刻看出被跳过的行在哪。
   这条竖线改由 ::before 提供（原来是 box-shadow inset），已关闭时置灰：
   对一行被跳过的分组来说"是否参与"比"属于哪个平台"更要紧，
   平台信息仍由行内徽标承担，不会丢。 */
.sp-rate-guard-group-row.disabled {
  background: color-mix(in srgb, var(--sp-muted) 6%, var(--sp-panel));
}

.sp-rate-guard-group-row.disabled::before {
  background: color-mix(in srgb, var(--sp-muted) 55%, var(--sp-line));
}

.sp-rate-guard-group-row.disabled:hover {
  background: color-mix(in srgb, var(--sp-muted) 9%, var(--sp-panel));
}

.sp-rate-guard-group-choice {
  display: grid;
  min-width: 0;
  flex: 1 1 auto;
  grid-template-columns: 16px minmax(0, 1fr);
  align-items: center;
  gap: 9px;
  cursor: pointer;
}

.sp-rate-guard-group-choice input[type='checkbox'] {
  width: 16px;
  height: 16px;
  /* 不依赖浏览器默认外观：否则各平台未勾选态的方框粗细、圆角都不一样，
     而这一列正是用户判断"这个分组到底参不参与"的唯一依据，必须稳定可辨。 */
  accent-color: var(--sp-cyan);
  border-radius: 4px;
  cursor: pointer;
  margin: 0;
}

.sp-rate-guard-group-choice input[type='checkbox']:focus-visible {
  outline: 2px solid color-mix(in srgb, var(--sp-cyan) 40%, transparent);
  outline-offset: 2px;
}

.sp-rate-guard-group-choice-copy {
  display: flex;
  align-items: center;
  gap: 8px;
  min-width: 0;
  flex-wrap: wrap;
}

/* 分组名不设 color：它由行上的 platformTextClass 按平台着色
   （与健康守护账号名同一套做法）。这里若写死 color: var(--sp-text)，
   权重 (0,1,1) 会压过 Tailwind 的平台文字色 (0,1,0)，平台色永远不生效。
   未着色时的兜底由继承提供，值同样是 --sp-text。
   ⚠️「已关闭」行的置灰规则权重更高（(0,2,1)），仍会正常覆盖平台色。 */
.sp-rate-guard-group-choice-copy strong {
  font-size: 13px;
  max-width: 100%;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.sp-rate-guard-group-row.disabled .sp-rate-guard-group-choice-copy strong {
  color: var(--sp-muted);
}

.sp-rate-guard-group-id,
.sp-rate-guard-group-rate {
  color: var(--sp-muted);
  font-size: 12px;
  font-variant-numeric: tabular-nums;
  white-space: nowrap;
}

/* 倍率用底色块而非描边：--sp-soft 在浅灰列表底上几乎看不见边框，
   底色块能在不抢视线的前提下把"倍率"和前面的 ID 区分开。 */
.sp-rate-guard-group-rate {
  border-radius: 6px;
  padding: 1px 6px;
  background: color-mix(in srgb, var(--sp-line) 55%, transparent);
}

/* 平台徽标：规格与健康守护账号行的 .sp-health-guard-account-platform 完全一致 ——
   同一页面里的平台徽标不该出现两种字号或圆角。
   配色由 platformBadgeClass 提供（项目集中式平台配色库），这里不再另写一份映射。 */
.sp-rate-guard-group-platform {
  display: inline-flex;
  flex: 0 0 auto;
  align-items: center;
  border-width: 1px;
  border-radius: 4px;
  padding: 0 5px;
  font-size: 10px;
  font-weight: 700;
  line-height: 1.4;
  white-space: nowrap;
}

/* 状态标签只在"已关闭"行出现，因此直接用琥珀警示色 ——
   它现在表达的是"这个分组被排除了"，不是中性状态，不该用灰色低调处理。 */
.sp-rate-guard-group-status {
  flex: 0 0 auto;
  border: 1px solid color-mix(in srgb, var(--sp-amber) 38%, var(--sp-line));
  border-radius: 6px;
  padding: 2px 8px;
  background: color-mix(in srgb, var(--sp-amber) 8%, transparent);
  color: var(--sp-amber);
  font-size: 12px;
  font-weight: 600;
  white-space: nowrap;
}

.sp-rate-guard-group-hint {
  margin-right: auto;
  color: var(--sp-muted);
  font-size: 12px;
}



































.sp-health-guard-multiplier-switch {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 16px;
  margin-top: 12px;
  padding: 10px 12px;
  border: 1px solid color-mix(in srgb, var(--sp-blue) 12%, var(--sp-line));
  border-radius: 10px;
  background: color-mix(in srgb, var(--sp-soft) 20%, transparent);
}

.sp-health-guard-multiplier-switch > div {
  display: grid;
  gap: 3px;
  min-width: 0;
}

.sp-health-guard-multiplier-switch > div span {
  color: var(--sp-muted);
}





















































































































































/* 行首的「参与守护」开关与行内的「修改调度」开关共用一套规格，
   两者都是「开关 + 说明文字」的小控件，分开写会各自漂移。 */
.sp-health-guard-account-scheduling-toggle,
.sp-health-guard-account-guard-toggle {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  color: var(--sp-muted);
  cursor: pointer;
  font-size: 11px;
  font-weight: 650;
  line-height: 1.2;
  white-space: nowrap;
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
  gap: 8px;
  padding-bottom: 10px;
}

.sp-rate-guard-section-head h4 {
  margin: 0;
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

/* 择优调度明细有自己的列语义：两个评分依据（次数、用时）相邻，然后才是结论与原因。
   不能沿用限速守护那张表的列宽，否则新列会挤进别人的宽度里。
   顺带补上表头 —— 原来 .sp-rate-guard-head 没有任何规则，表头和数据行根本对不齐。 */
.sp-group-election-head,
.sp-group-election-row {
  display: grid;
  grid-template-columns: minmax(170px, 1.25fr) 92px 88px 96px minmax(120px, 0.9fr) minmax(160px, 1.15fr);
  min-width: 726px;
}

.sp-group-election-table {
  overflow-x: auto;
}

.sp-group-election-head {
  border-bottom: 1px solid var(--sp-line);
}

.sp-group-election-head > span {
  padding: 10px 12px;
  color: var(--sp-muted);
  font-size: 11px;
  font-weight: 700;
}

/* 用时是拿来横向比较的数字，等宽数字才不会让相邻行的数字左右跳动。 */
.sp-group-election-latency {
  font-variant-numeric: tabular-nums;
}

.sp-group-election-latency.is-empty {
  color: var(--sp-muted);
}

/* 「每组开启数 + 必需模型 + 默认账号 + 异常推送」同处一行：另起一整行，
   避免和上面的勾选/健康锁定挤在同一水平线上。
   允许换行：四项在窄屏（md 及以下，弹窗只有 768px）排不下时整项折到下一行，
   而不是把「必需模型」压成 0 宽 —— 它是这一行里唯一需要输入的长控件。 */
.sp-election-group-extra {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 16px;
  min-width: 0;
  padding-left: 24px;
}

.sp-election-required-models {
  flex: 1 1 auto;
  min-width: 180px;
  display: flex;
  align-items: center;
  gap: 8px;
  min-width: 0;
}

.sp-election-required-models-label {
  flex: 0 0 auto;
  font-size: 12px;
  color: var(--sp-muted);
}

.sp-election-required-models-input {
  flex: 1 1 auto;
  min-width: 0;
  padding: 4px 8px;
  font-size: 12px;
  color: var(--sp-text);
  background: var(--sp-panel);
  border: 1px solid var(--sp-line);
  border-radius: 6px;
}

.sp-election-required-models-input:focus {
  outline: none;
  border-color: var(--sp-cyan);
}

/* 每组开启数覆盖：窄数字框，不铺满，与左侧必需模型并排。 */
.sp-election-top-n-override {
  flex: 0 0 auto;
  display: flex;
  align-items: center;
  gap: 8px;
}

.sp-election-top-n-override-label {
  flex: 0 0 auto;
  font-size: 12px;
  color: var(--sp-muted);
}

.sp-election-top-n-override-input {
  flex: 0 0 auto;
  width: 120px;
  padding: 4px 8px;
  font-size: 12px;
  color: var(--sp-text);
  background: var(--sp-panel);
  border: 1px solid var(--sp-line);
  border-radius: 6px;
}

.sp-election-top-n-override-input:focus {
  outline: none;
  border-color: var(--sp-cyan);
}

/* 分组默认账号：窄下拉，与「每组开启数」「必需模型」并排。
   不铺满 —— 它多数时候是「不指定」，留白比拉长更像一个可选配置。
   控件本身用框架 Select（边框 / 背景 / 焦点态由它自带），这里只约束宽度。 */
.sp-election-default-account {
  flex: 0 0 auto;
  display: flex;
  align-items: center;
  gap: 8px;
}

.sp-election-default-account-label {
  flex: 0 0 auto;
  font-size: 12px;
  color: var(--sp-muted);
}

.sp-election-default-account-select {
  width: 180px;
}

/* 分组行内的「异常推送」覆盖：三段式（跟随全局 / 强制推送 / 静音）。
   刻意不用开关 —— 开关只有两态，表达不了「这个分组没配过覆盖」，
   而那正是绝大多数分组的状态。 */
.sp-election-alert-override {
  flex: 0 0 auto;
  display: flex;
  align-items: center;
  gap: 8px;
}

.sp-election-alert-override-label {
  flex: 0 0 auto;
  font-size: 12px;
  color: var(--sp-muted);
}

.sp-election-alert-override-choice {
  display: inline-flex;
  flex: 0 0 auto;
  gap: 4px;
  padding: 2px;
  border: 1px solid var(--sp-line);
  border-radius: 10px;
  background: color-mix(in srgb, var(--sp-soft) 18%, var(--sp-panel));
}

.sp-election-alert-override-choice button {
  border: none;
  border-radius: 8px;
  padding: 4px 10px;
  background: transparent;
  color: var(--sp-muted);
  font-size: 12px;
  cursor: pointer;
}

.sp-election-alert-override-choice button.active {
  background: var(--sp-panel);
  color: var(--sp-text);
  font-weight: 600;
}

/* 必需模型断供是需要人工跟进的告警，用红色把统计格与明细都标出来。 */
.sp-group-election-warn-stat strong {
  color: var(--sp-red);
}

/* 演练的建议统计用紫（与编辑弹窗的演练开关同色），
   不用红/绿：它不是故障，也不是"已开启/已关闭"，而是"没落地"。 */
.sp-group-election-suggest-stat strong {
  color: var(--sp-violet, #7c3aed);
}

.dark .sp-group-election-suggest-stat strong {
  color: #a78bfa;
}

.sp-group-election-required-warns {
  margin-top: 10px;
  padding: 10px 12px;
  border: 1px solid color-mix(in srgb, var(--sp-red) 40%, var(--sp-line));
  border-radius: 8px;
  background: color-mix(in srgb, var(--sp-red) 6%, var(--sp-panel));
}

.sp-group-election-required-warns-title {
  font-size: 12px;
  font-weight: 600;
  color: var(--sp-red);
  margin-bottom: 6px;
}

.sp-group-election-required-warn {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 6px 10px;
  padding: 4px 0;
  font-size: 12px;
}

.sp-group-election-required-warn-group {
  font-weight: 600;
  color: var(--sp-text);
}

.sp-group-election-required-warn-model {
  font-variant-numeric: tabular-nums;
  color: var(--sp-red);
}

.sp-group-election-required-warn small {
  color: var(--sp-muted);
}

.sp-monitor-detail {
  --sp-monitor-accent: var(--sp-cyan);
}

.sp-monitor-summary strong.sp-monitor-warning-value,
.sp-monitor-unmatched strong {
  color: var(--sp-red);
}

.sp-monitor-items {
  min-width: 0;
}

.sp-monitor-items-table {
  border-left: 3px solid var(--sp-cyan);
  background: color-mix(in srgb, var(--sp-cyan) 4%, transparent);
}

.sp-monitor-row {
  display: grid;
  grid-template-columns: minmax(190px, 1.25fr) minmax(150px, 1fr) minmax(170px, 1.1fr) 110px 90px 105px 150px;
  min-width: 1080px;
  border-bottom: 1px solid var(--sp-line);
}

.sp-monitor-row > span {
  min-width: 0;
  padding: 11px 12px;
  color: var(--sp-text);
  font-size: 12px;
  word-break: break-word;
}

.sp-monitor-row strong {
  display: block;
  font-size: 12px;
}

.sp-monitor-row small {
  display: block;
  margin-top: 3px;
  color: var(--sp-muted);
  font-size: 11px;
}

.sp-monitor-row-head > span {
  color: var(--sp-muted);
  font-size: 11px;
  font-weight: 700;
}

.sp-monitor-unmatched {
  color: var(--sp-red) !important;
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

  .sp-health-guard-account-row,
  .sp-health-guard-selected-toggle,
  .sp-health-guard-account-card .sp-button.sp-health-guard-config-button,
  .sp-health-guard-platform-model-grid article {
    transition: none;
  }
}
@media (max-width: 1024px) {
  .sp-overview-strip {
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }



  .sp-health-guard-policy-grid,
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

  /* 手机端筛选保持紧凑：任务/状态下拉 2 列，重置按钮通栏 */
  .sp-run-filters {
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }

  .sp-run-filters .sp-button {
    grid-column: 1 / -1;
    width: 100%;
  }

  /* 顶部统计卡：手机端保持 2x2，并压缩高度，避免四张卡各占一整行 */
  .sp-overview-strip {
    grid-template-columns: repeat(2, minmax(0, 1fr));
    gap: 0.5rem;
  }

  .sp-overview-item {
    min-height: 0;
    border-radius: 12px;
    padding: 0.65rem 0.75rem 0.7rem;
    box-shadow: 0 1px 2px color-mix(in srgb, var(--sp-text) 4%, transparent);
  }

  .sp-overview-item::before {
    left: 0.75rem;
    width: 1.75rem;
    height: 2px;
  }

  .sp-overview-item .sp-metric-value {
    margin-top: 0.35rem;
    font-size: clamp(1.25rem, 5.5vw, 1.5rem);
  }

  .sp-overview-item .sp-metric-foot {
    margin-top: 0.45rem;
    padding-top: 0.4rem;
    font-size: 0.6875rem;
    line-height: 1.35;
    display: -webkit-box;
    -webkit-line-clamp: 2;
    -webkit-box-orient: vertical;
    overflow: hidden;
  }

  .sp-metric-head {
    gap: 0.35rem;
  }

  .sp-metric-label {
    font-size: 0.6875rem;
  }

  .sp-metric-signal {
    width: 6px;
    height: 6px;
    box-shadow: 0 0 0 3px color-mix(in srgb, var(--sp-metric-accent) 12%, transparent);
  }

  .sp-edit-dialog,
  .sp-edit-summary,
  .sp-form-grid,
  .sp-health-guard-policy-grid,
  .sp-retention-grid,
  .sp-group-election-policy-grid,
  .sp-run-detail-summary,
  .sp-provider-detail-layout,
  .sp-cleanup-grid,
  .sp-rate-guard-summary,
  .sp-stage-body {
    grid-template-columns: 1fr;
  }



  .sp-panel-head {
    gap: 8px;
    padding: 14px;
  }

  .sp-panel-signals {
    justify-content: flex-start;
    width: 100%;
  }

  .sp-history-toolbar {
    padding: 8px 12px; background: color-mix(in srgb, var(--sp-blue) 5%, var(--sp-panel)); border-left: 3px solid var(--sp-blue); background: color-mix(in srgb, var(--sp-blue) 5%, var(--sp-panel)); border-left: 3px solid var(--sp-blue); background: color-mix(in srgb, var(--sp-blue) 5%, var(--sp-panel)); border-left: 3px solid var(--sp-blue); background: color-mix(in srgb, var(--sp-blue) 5%, var(--sp-panel)); border-left: 3px solid var(--sp-blue);
  }


  .sp-edit-summary > div,
  .sp-edit-summary > div:first-child {
    border-top: 1px solid var(--sp-soft);
    border-left: 0;
  }

  .sp-edit-summary > div:first-child {
    border-top: 0;
  }

  .sp-health-guard-account-card,
  .sp-health-guard-account-toolbar {
    align-items: stretch;
    flex-direction: column;
  }

  

  

  

  

  

  

  

  

  

  

  

  

  

  

  

  .sp-health-guard-account-card .sp-button {
    width: 100%;
  }

  .sp-health-guard-item-grid > div,
  .sp-health-guard-item-grid > div:first-child {
    border-top: 1px solid var(--sp-line);
    border-left: 0;
  }

  .sp-health-guard-item-grid > div:first-child {
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
}

/* 演练模式开关：与「分组参与择优」范围卡片同构，但配色走紫（--sp-violet）——
   演练既不是"开"也不是"关"，用绿/红会误导；而蓝已被分组参与占用、橙被切换日志占用、
   琥珀被倍率日志占用，紫是这套色板里唯一没有歧义的剩余色相。
   弹窗由 BaseDialog Teleport 到 body，--sp-* 可能取不到，故关键色都带明确回退值。 */
.sp-election-dry-run-card {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 16px;
  padding: 12px 16px;
  border: 1px solid var(--sp-line, #e5e7eb);
  border-radius: 12px;
  background: var(--sp-panel-2, #f9fafb);
  margin-bottom: 14px;
}

.sp-election-dry-run-card > div {
  display: grid;
  gap: 4px;
  min-width: 0;
}

.sp-election-dry-run-card strong {
  font-size: 13px;
  color: var(--sp-text, #111827);
}

.sp-election-dry-run-card span {
  font-size: 12px;
  line-height: 1.6;
  color: var(--sp-muted, #64748b);
}

.sp-election-dry-run-card.is-on {
  border-color: color-mix(in srgb, var(--sp-violet, #7c3aed) 45%, var(--sp-line, #e5e7eb));
  background: color-mix(in srgb, var(--sp-violet, #7c3aed) 7%, var(--sp-panel, #ffffff));
}

.sp-election-dry-run-card.is-on strong,
strong.sp-election-dry-run-count {
  color: var(--sp-violet, #7c3aed);
}

.sp-election-dry-run-toggle {
  display: inline-flex;
  align-items: center;
  gap: 8px;
  flex: 0 0 auto;
  font-size: 12px;
  color: var(--sp-muted, #64748b);
}

.sp-election-dry-run-card.is-on .sp-election-dry-run-toggle {
  font-weight: 600;
  color: var(--sp-violet, #7c3aed);
}

.dark .sp-election-dry-run-card {
  border-color: #374151;
  background: #111827;
}

.dark .sp-election-dry-run-card.is-on {
  background: color-mix(in srgb, #a78bfa 10%, #1f2937);
}

.dark .sp-election-dry-run-card.is-on strong,
.dark strong.sp-election-dry-run-count {
  color: #a78bfa;
}

/* 在任者健康锁定卡片复用演练卡片的排版，只把"开启"配色改成绿（--sp-green）：
   锁定健康在任者是"稳住不动"的正向动作，绿在语义上贴切，也和紫色的演练卡片区分开。
   这些覆盖规则与演练版同特指度，靠排在其后取胜。 */
.sp-election-keep-healthy-card.is-on {
  border-color: color-mix(in srgb, var(--sp-green, #16835d) 45%, var(--sp-line, #e5e7eb));
  background: color-mix(in srgb, var(--sp-green, #16835d) 7%, var(--sp-panel, #ffffff));
}

.sp-election-keep-healthy-card.is-on strong,
strong.sp-election-keep-healthy-count {
  color: var(--sp-green, #16835d);
}

.sp-election-keep-healthy-card.is-on .sp-election-dry-run-toggle {
  color: var(--sp-green, #16835d);
}

.dark .sp-election-keep-healthy-card.is-on {
  background: color-mix(in srgb, #34d399 10%, #1f2937);
}

.dark .sp-election-keep-healthy-card.is-on strong,
.dark strong.sp-election-keep-healthy-count {
  color: #34d399;
}</style>

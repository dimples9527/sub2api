<template>
  <SupplierModuleLayout>
    <section class="sp-account-toolbar" aria-label="账号筛选与操作">
      <header class="sp-filter-card-head">
        <div>
          <span class="sp-filter-card-kicker">筛选条件</span>
          <h2>筛选账号</h2>
          <p>按供应商、平台、本地分组、同步有效性和上游状态 / 已删除快速定位账号。</p>
        </div>
        <span class="sp-filter-card-count">{{ total }} 个账号</span>
      </header>

      <div class="sp-account-filter-body">
        <div class="sp-account-filter-fields">
          <div
            ref="searchFilterControl"
            class="sp-account-filter-control sp-account-search"
            role="group"
            aria-labelledby="supplier-account-search-label"
          >
            <span id="supplier-account-search-label" class="sr-only">账号搜索</span>
            <Input
              v-model="search"
              class="w-full"
              placeholder="搜索账号名称或上游 Key"
            />
          </div>
          <div
            ref="providerFilterControl"
            class="sp-account-filter-control"
            role="group"
            aria-labelledby="supplier-account-provider-label"
          >
            <span id="supplier-account-provider-label" class="sr-only">供应商</span>
            <Select
              v-model="providerID"
              class="w-full"
              :options="providerOptions"
              :searchable="false"
            />
          </div>
          <div
            ref="platformFilterControl"
            class="sp-account-filter-control"
            role="group"
            aria-labelledby="supplier-account-platform-label"
          >
            <span id="supplier-account-platform-label" class="sr-only">平台</span>
            <Select
              v-model="platformFilter"
              class="w-full"
              :options="platformFilterOptions"
              :searchable="false"
            />
          </div>
          <div
            ref="groupFilterControl"
            class="sp-account-filter-control"
            role="group"
            aria-labelledby="supplier-account-group-label"
          >
            <span id="supplier-account-group-label" class="sr-only">本地分组</span>
            <Select
              v-model="groupID"
              class="w-full"
              :options="localGroupOptions"
              :searchable="false"
            />
          </div>
          <div
            ref="activeFilterControl"
            class="sp-account-filter-control"
            role="group"
            aria-labelledby="supplier-account-active-label"
          >
            <span id="supplier-account-active-label" class="sr-only">同步有效性</span>
            <Select
              v-model="activeFilter"
              class="w-full"
              :options="activeFilterOptions"
              :searchable="false"
            />
          </div>
          <div
            ref="upstreamStatusFilterControl"
            class="sp-account-filter-control"
            role="group"
            aria-labelledby="supplier-account-upstream-status-label"
          >
            <span id="supplier-account-upstream-status-label" class="sr-only">上游状态</span>
            <Select
              v-model="upstreamStatusFilter"
              class="w-full"
              :options="upstreamStatusFilterOptions"
              :searchable="false"
            />
          </div>
        </div>
        <div class="sp-account-filter-actions" data-test="supplier-account-filter-actions">
          <button
            class="sp-button sp-account-toolbar-btn sp-account-toolbar-create"
            type="button"
            data-test="supplier-account-create"
            @click="openCreateAccountDialog"
          >
            添加账号
          </button>
          <button
            class="sp-button sp-account-toolbar-btn sp-account-toolbar-test"
            type="button"
            data-test="supplier-account-batch-test"
            :disabled="!batchTesting && (loading || batchTestPreparing || total === 0)"
            @click="handleSupplierBatchTestButton"
          >
            {{ batchTesting ? '查看测试进度' : batchTestPreparing ? '准备测试中…' : '测试当前筛选' }}
          </button>
          <button
            class="sp-button sp-account-toolbar-btn sp-account-toolbar-logs"
            type="button"
            data-test="supplier-account-rate-guard-logs"
            :class="{ 'has-pending': accountRateGuardPendingCount > 0 }"
            @click="openAccountRateGuardLogs"
          >
            倍率守护日志
            <span
              v-if="accountRateGuardPendingCount > 0"
              class="sp-account-rate-guard-pending-count"
              data-test="supplier-account-rate-guard-pending-count"
            >{{ accountRateGuardPendingCount }}</span>
          </button>
          <button
            class="sp-button sp-account-toolbar-btn sp-account-toolbar-refresh sp-account-refresh"
            type="button"
            data-test="supplier-account-refresh"
            :disabled="loading"
            @click="refreshAccountsWorkbench"
          >
            {{ loading ? '刷新中…' : '刷新' }}
          </button>
        </div>
      </div>
    </section>
    <div v-if="error" class="sp-alert sp-error-line">{{ error }}</div>

    <section class="sp-panel sp-account-workbench">
      <header class="sp-panel-head sp-account-panel-head">
        <div class="sp-panel-title">
          <span class="sp-section-index">01</span>
          <div>
            <h2>上游账号表</h2>
            <span>当前筛选共 {{ total }} 个上游账号</span>
          </div>
        </div>
        <div class="sp-account-quick-filters" role="group" aria-label="账号快捷过滤">
          <button
            v-for="option in accountQuickFilterOptions"
            :key="option.key"
            :data-test="`supplier-account-quick-filter-${option.key}`"
            :class="['sp-account-quick-filter', { active: accountQuickFilter === option.key }]"
            type="button"
            @click="selectAccountQuickFilter(option.key)"
          >
            <span>{{ option.label }}</span><strong>{{ option.count }}</strong>
          </button>
        </div>
        <div class="sp-account-legend" aria-label="本地账号匹配状态图例">
          <span><i class="matched"></i>已匹配</span>
          <span><i class="unmatched"></i>未匹配</span>
          <span><i class="conflict"></i>匹配冲突</span>
        </div>
      </header>

      <div class="sp-account-table-shell">
        <DataTable
          :columns="accountColumns"
          :data="items"
          :loading="loading"
          row-key="id"
          server-side-sort
          clickable-rows
          @sort="handleAccountSort"
          @row-click="openDrawer"
        >
          <template #cell-provider_name="{ row: account }">
            <div :class="['sp-provider-cell', supplierTone(account.provider_id).chip]">
              <span
                :class="['sp-provider-dot', supplierTone(account.provider_id).dot]"
                aria-hidden="true"
              ></span>
              <div class="sp-account-copy">
                <div class="sp-entity">{{ account.provider_name || '—' }}</div>
                <div class="sp-sub sp-account-meta">
                  <span>供应商 #{{ account.provider_id }}</span>
                </div>
              </div>
            </div>
          </template>

          <template #cell-upstream_account_key="{ row: account }">
            <div class="sp-account-identity">
              <span class="sp-account-avatar" aria-hidden="true">{{ accountInitial(account) }}</span>
              <div class="sp-account-copy">
                <div class="sp-entity">{{ account.name || '—' }}</div>
                <div class="sp-sub sp-account-meta">
                  <span
                    v-if="effectivePlatform(account) !== 'unknown'"
                    :class="['sp-platform-badge', platformBadgeClass(effectivePlatform(account))]"
                  >
                    {{ platformLabel(effectivePlatform(account)) }}
                  </span>
                  <span v-else class="sp-account-muted">—</span>
                  <span class="sp-account-key" :title="account.upstream_account_key || '—'">
                    {{ account.upstream_account_key || '—' }}
                  </span>
                </div>
              </div>
            </div>
          </template>

          <template #cell-local_account_name="{ row: account }">
            <span
              v-if="account.local_account_match_status === 'unmatched'"
              class="sp-match-badge unmatched"
            >
              未匹配
            </span>
            <span
              v-else-if="account.local_account_match_status === 'conflict'"
              class="sp-match-badge conflict"
            >
              匹配冲突（{{ account.local_account_match_count }}）
            </span>
            <div
              v-else-if="account.local_account_match_status === 'matched'"
              class="sp-local-account-cell"
            >
              <span class="sp-match-badge matched">已匹配</span>
              <strong>{{ displayValue(account.local_account_name) }}</strong>
            </div>
            <span v-else class="sp-account-muted">—</span>
          </template>

          <template #cell-local_account_priority="{ row: account }">
            <div v-if="canEditPriority(account)" class="sp-priority-cell" @click.stop>
              <Input
                v-if="editingPriorityAccountID === account.local_account_id"
                ref="priorityInput"
                v-model="priorityDraft"
                type="number"
                class="sp-priority-input"
                :disabled="savingPriorityAccountID === account.local_account_id"
                @click.stop
                @enter="savePriority(account)"
                @keydown.esc="cancelPriorityEdit"
                @blur="savePriority(account)"
              />
              <button
                v-else
                type="button"
                class="sp-account-number sp-priority-trigger"
                :disabled="savingPriorityAccountID === account.local_account_id"
                title="点击编辑账号优先级"
                @click.stop="startPriorityEdit(account)"
              >
                {{ displayValue(account.local_account_priority) }}
              </button>
            </div>
            <span v-else class="sp-account-muted">—</span>
          </template>

          <template #cell-rate_multiplier="{ row: account }">
            <span :class="['sp-account-rate', platformTextClass(effectivePlatform(account))]">
              {{ formatRate(account.rate_multiplier) }}
            </span>
          </template>

          <template #cell-group_name="{ row: account }">
            <div class="sp-account-group-stack">
              <div v-if="account.binding_groups?.length" class="sp-account-groups">
                <GroupBadge
                  v-for="group in account.binding_groups"
                  :key="group.id"
                  :name="group.name"
                  :platform="group.platform"
                  :subscription-type="group.subscription_type"
                  :rate-multiplier="group.rate_multiplier"
                  :show-rate="true"
                  :always-show-rate="true"
                />
              </div>
              <span v-else class="sp-account-muted">—</span>
              <span
                v-if="account.group_status === 'inactive' || account.group_status === 'missing'"
                class="sp-upstream-group-deleted"
                :title="account.group_status === 'inactive' ? '上游分组已失效' : '上游分组已删除'"
              >
                上游分组已{{ account.group_status === 'inactive' ? '失效' : '删除' }}
                <template v-if="account.group_name">（{{ account.group_name }}）</template>
              </span>
            </div>
          </template>

          <template #cell-local_account_status="{ row: account }">
            <span
              v-if="isMatchedLocalAccount(account)"
              :class="['sp-local-status', localAccountStatusTone(account.local_account_status)]"
            >
              {{ localAccountStatusLabel(account.local_account_status) }}
            </span>
            <span v-else class="sp-account-muted">—</span>
          </template>

          <template #cell-upstream_account_status="{ row: account }">
            <span
              v-if="account.status || account.raw_status"
              :class="['sp-local-status', upstreamStatusTone(account.status || account.raw_status)]"
              :title="account.raw_status && account.raw_status !== account.status ? ('原始: ' + account.raw_status) : undefined"
            >
              {{ upstreamStatusLabel(account.status || account.raw_status) }}
            </span>
            <span v-else class="sp-account-muted">—</span>
          </template>

          <template #cell-local_account_schedulable="{ row: account }">
            <button
              v-if="canToggleSchedulable(account)"
              type="button"
              class="relative inline-flex h-5 w-9 flex-shrink-0 cursor-pointer rounded-full border-2 border-transparent transition-colors duration-200 ease-in-out focus:outline-none focus:ring-2 focus:ring-primary-500 focus:ring-offset-2 disabled:cursor-not-allowed disabled:opacity-50 dark:focus:ring-offset-dark-800"
              :class="account.local_account_schedulable ? 'bg-primary-500 hover:bg-primary-600' : 'bg-gray-200 hover:bg-gray-300 dark:bg-dark-600 dark:hover:bg-dark-500'"
              :disabled="togglingSchedulableID === account.local_account_id"
              :aria-pressed="account.local_account_schedulable"
              :title="schedulableToggleTitle(account)"
              @click.stop="handleToggleSchedulable(account)"
            >
              <span
                class="pointer-events-none inline-block h-4 w-4 transform rounded-full bg-white shadow ring-0 transition duration-200 ease-in-out"
                :class="account.local_account_schedulable ? 'translate-x-4' : 'translate-x-0'"
              />
            </button>
            <span v-else class="sp-account-muted">—</span>
          </template>

          <template #cell-local_account_last_test_status="{ row: account }">
            <button
              v-if="isFailedTest(account)"
              type="button"
              class="sp-test-status failed"
              title="查看测试失败详情"
              @click.stop="openTestErrorDialog(account)"
            >
              {{ accountTestStatusLabel(account.local_account_last_test_status) }}
            </button>
            <span
              v-else-if="isMatchedLocalAccount(account) && account.local_account_last_test_status"
              :class="['sp-test-status', accountTestStatusTone(account.local_account_last_test_status)]"
            >
              {{ accountTestStatusLabel(account.local_account_last_test_status) }}
            </span>
            <span v-else class="sp-account-muted">—</span>
          </template>

          <template #cell-local_account_last_tested_at="{ row: account }">
            <span v-if="isMatchedLocalAccount(account)" class="sp-account-time">
              {{ formatTime(account.local_account_last_tested_at) }}
            </span>
            <span v-else class="sp-account-muted">—</span>
          </template>

          <template #cell-supplier_current_balance="{ row: account }">
            <div class="sp-money-cell">
              <strong>{{ formatCNY(account.supplier_current_balance) }}</strong>
              <small>供应商汇总</small>
            </div>
          </template>

          <template #cell-supplier_today_cost="{ row: account }">
            <div class="sp-money-cell cost">
              <strong>{{ formatCNY(account.supplier_today_cost) }}</strong>
              <small>供应商汇总</small>
            </div>
          </template>

          <template #cell-actions="{ row: account }">
            <div class="sp-account-row-actions" @click.stop>
              <button
                class="sp-button small ghost sp-account-view-button sp-account-action-view"
                type="button"
                @click.stop="openDrawer(account)"
              >查看</button>
              <button
                class="sp-button small danger sp-account-action-delete"
                type="button"
                :disabled="deletingSupplierAccountRecordID === account.id"
                title="删除上游账号记录"
                @click.stop="requestDeleteSupplierAccountRecord(account)"
              >{{ deletingSupplierAccountRecordID === account.id ? '删除中' : '删除上游账号记录' }}</button>
              <template v-if="canManageLocalAccount(account)">
                <button
                  class="sp-button small sp-account-action-test"
                  type="button"
                  :disabled="testingAccountID === account.local_account_id || accountActionLoadingID === account.local_account_id || duplicatingAccountID === account.local_account_id"
                  :data-test="`supplier-account-test-${account.local_account_id}`"
                  @click.stop="openLocalAccountTest(account)"
                >{{ testingAccountID === account.local_account_id ? '加载中…' : '测试账号' }}</button>
                <button
                  class="sp-button small sp-account-action-edit"
                  type="button"
                  :disabled="accountActionLoadingID === account.local_account_id || testingAccountID === account.local_account_id || duplicatingAccountID === account.local_account_id"
                  @click.stop="openLocalAccountEditor(account)"
                >编辑</button>
                <button
                  v-if="canDuplicateLocalAccount(account)"
                  class="sp-button small sp-account-action-copy"
                  type="button"
                  :disabled="duplicatingAccountID === account.local_account_id || accountActionLoadingID === account.local_account_id || testingAccountID === account.local_account_id || deletingAccountID === account.local_account_id"
                  @click.stop="requestDuplicateLocalAccount(account)"
                  data-test="supplier-account-duplicate"
                >{{ duplicatingAccountID === account.local_account_id ? '复制中' : '复制账号' }}</button>
                <button
                  class="sp-button small sp-account-action-platform"
                  type="button"
                  :disabled="savingBusinessPlatform || accountActionLoadingID === account.local_account_id || testingAccountID === account.local_account_id || duplicatingAccountID === account.local_account_id"
                  @click.stop="openBusinessPlatformDialog(account)"
                >配置业务平台</button>
                <button
                  class="sp-button small sp-account-action-binding"
                  type="button"
                  :disabled="accountActionLoadingID === account.local_account_id || testingAccountID === account.local_account_id || duplicatingAccountID === account.local_account_id"
                  @click.stop="openAccountBindingEditor(account)"
                >编辑绑定</button>
                <button
                  class="sp-button small danger sp-account-action-delete"
                  type="button"
                  :disabled="deletingAccountID === account.local_account_id || testingAccountID === account.local_account_id || duplicatingAccountID === account.local_account_id"
                  @click.stop="deleteLocalAccount(account)"
                >{{ deletingAccountID === account.local_account_id ? '删除中' : '删除' }}</button>
              </template>
            </div>
          </template>

          <template #empty>
            <div class="sp-account-empty">
              <strong>暂无上游账号数据</strong>
              <span>请先同步供应商上游账号，或调整当前筛选条件。</span>
            </div>
          </template>
        </DataTable>
      </div>

      <footer v-if="total > 0" class="sp-account-pagination">
        <div class="sp-account-page-size">
          <span>每页显示</span>
          <Select
            :model-value="pageSize"
            class="sp-account-page-size-select"
            :options="pageSizeOptions"
            :searchable="false"
            @update:model-value="handlePageSizeChange"
          />
          <span>条</span>
        </div>
        <Pagination
          class="sp-data-pagination"
          :page="page"
          :total="total"
          :page-size="pageSize"
          :show-page-size-selector="false"
          @update:page="handlePageChange"
        />
      </footer>
    </section>

    <SupplierDrawer
      :show="Boolean(selected)"
      :title="selected?.name || selected?.upstream_account_key || '上游账号详情'"
      eyebrow="ACCOUNT DETAIL"
      @close="selected = null"
    >
      <template v-if="selected">
        <div class="sp-detail-grid">
          <div class="sp-detail-cell"><span>供应商</span><b>{{ displayValue(selected.provider_name) }}</b></div>
          <div class="sp-detail-cell"><span>平台</span><b>{{ effectivePlatform(selected) !== 'unknown' ? platformLabel(effectivePlatform(selected)) : '—' }}</b></div>
          <div class="sp-detail-cell"><span>上游账号名称</span><b>{{ displayValue(selected.name) }}</b></div>
          <div class="sp-detail-cell"><span>上游 Key</span><b>{{ displayValue(selected.upstream_account_key) }}</b></div>
          <div class="sp-detail-cell"><span>匹配状态</span><b>{{ localAccountMatchLabel(selected) }}</b></div>
          <div class="sp-detail-cell"><span>本地账号</span><b>{{ localAccountDisplayName(selected) }}</b></div>
          <div class="sp-detail-cell"><span>优先级</span><b>{{ localDetailValue(selected, selected.local_account_priority) }}</b></div>
          <div class="sp-detail-cell"><span>上游倍率</span><b>{{ formatRate(selected.rate_multiplier) }}</b></div>
          <div class="sp-detail-cell">
            <span>上游分组</span>
            <b>
              {{ selected.group_name || selected.group_key || '—' }}
              <template v-if="selected.group_status === 'inactive'">（已失效）</template>
              <template v-else-if="selected.group_status === 'missing'">（已删除）</template>
            </b>
          </div>
          <div class="sp-detail-cell"><span>本地账号状态</span><b>{{ isMatchedLocalAccount(selected) ? localAccountStatusLabel(selected.local_account_status) : '—' }}</b></div>
          <div class="sp-detail-cell"><span>是否调度</span><b>{{ localSchedulableLabel(selected) }}</b></div>
          <div class="sp-detail-cell"><span>测试结果</span><b>{{ isMatchedLocalAccount(selected) ? accountTestStatusLabel(selected.local_account_last_test_status) : '—' }}</b></div>
          <div class="sp-detail-cell"><span>上次测试时间</span><b>{{ isMatchedLocalAccount(selected) ? formatTime(selected.local_account_last_tested_at) : '—' }}</b></div>
          <div class="sp-detail-cell"><span>余额（供应商汇总）</span><b>{{ formatCNY(selected.supplier_current_balance) }}</b></div>
          <div class="sp-detail-cell"><span>今日消费（供应商汇总）</span><b>{{ formatCNY(selected.supplier_today_cost) }}</b></div>
          <div class="sp-detail-cell"><span>上游状态</span><b>{{ upstreamStatusLabel(selected.status || selected.raw_status) }}</b></div>
          <div class="sp-detail-cell"><span>最近同步</span><b>{{ formatTime(selected.last_seen_at) }}</b></div>
          <div class="sp-detail-cell"><span>失效时间</span><b>{{ formatTime(selected.inactive_at) }}</b></div>
        </div>
      </template>
    </SupplierDrawer>

    <BaseDialog
      :show="Boolean(businessPlatformAccount)"
      title="配置业务平台"
      @close="closeBusinessPlatformDialog"
    >
      <div v-if="businessPlatformAccount" class="sp-business-platform-dialog">
        <dl class="sp-business-platform-summary">
          <div>
            <dt>本地账号</dt>
            <dd>{{ businessPlatformAccount.local_account_name || businessPlatformAccount.name || `账号 #${businessPlatformAccount.local_account_id}` }}</dd>
          </div>
          <div>
            <dt>接入平台</dt>
            <dd>{{ platformLabel(normalizePlatform(businessPlatformAccount.local_account_platform) || 'unknown') }}</dd>
          </div>
        </dl>
        <label class="sp-business-platform-field">
          <span>业务平台</span>
          <Select
            v-model="businessPlatformDraft"
            class="w-full"
            :options="businessPlatformOptions"
            :searchable="false"
          />
        </label>
        <p class="sp-business-platform-hint">
          仅影响供应商模块中的平台归类和健康守护模型，不会修改账号接入协议或原始平台。
        </p>
      </div>
      <template #footer>
        <button
          class="sp-business-platform-cancel"
          type="button"
          :disabled="savingBusinessPlatform"
          @click="closeBusinessPlatformDialog"
        >取消</button>
        <button
          class="sp-business-platform-save"
          type="button"
          :disabled="savingBusinessPlatform"
          @click="saveBusinessPlatform"
        >{{ savingBusinessPlatform ? '保存中' : '保存' }}</button>
      </template>
    </BaseDialog>

    <BaseDialog
      :show="Boolean(testErrorAccount)"
      title="测试失败详情"
      width="normal"
      @close="testErrorAccount = null"
    >
      <div v-if="testErrorAccount" class="sp-test-error-dialog">
        <div class="sp-test-error-meta">
          <span>本地账号</span>
          <strong>{{ displayValue(testErrorAccount.local_account_name) }}</strong>
        </div>
        <div class="sp-test-error-meta">
          <span>上次测试时间</span>
          <strong>{{ formatTime(testErrorAccount.local_account_last_tested_at) }}</strong>
        </div>
        <div class="sp-test-error-message">
          {{ testErrorAccount.local_account_last_test_error || '暂无错误详情' }}
        </div>
      </div>
    </BaseDialog>

    <CreateAccountModal
      v-if="showCreateAccountModal"
      :show="showCreateAccountModal"
      :proxies="accountProxies"
      :groups="accountEditGroups"
      @close="closeCreateAccountDialog"
      @created="handleAccountCreated"
    />

    <BaseDialog
      :show="showBatchTestConfigDialog"
      title="供应商账号批量测试"
      width="wide"
      @close="closeBatchTestConfigDialog"
    >
      <div class="sp-batch-test-dialog">
        <div class="sp-batch-test-summary">
          <div class="sp-batch-summary-metric accounts">
            <span>待测试本地账号</span>
            <div><strong>{{ batchTestTargets.length }}</strong><small>个</small></div>
          </div>
          <div class="sp-batch-summary-metric platforms">
            <span>涉及平台</span>
            <div><strong>{{ batchTestPlatformSummaries.length }}</strong><small>个平台</small></div>
          </div>
          <p><span aria-hidden="true"></span>{{ batchTestFilterSummary }}</p>
        </div>

        <section class="sp-batch-test-section model-section">
          <header>
            <span class="sp-batch-section-index">01</span>
            <div>
              <strong>平台测试模型</strong>
              <span>未选择时由账号测试接口自动使用默认模型。</span>
            </div>
          </header>
          <div class="sp-batch-test-platform-list">
            <div
              v-for="summary in batchTestPlatformSummaries"
              :key="summary.platform"
              class="sp-batch-test-platform-row"
            >
              <span :class="['sp-batch-test-platform-accent', platformAccentBarClass(summary.platform)]"></span>
              <div class="sp-batch-test-platform-info">
                <span :class="['sp-platform-badge', platformBadgeClass(summary.platform)]">
                  {{ batchPlatformLabel(summary.platform) }}
                </span>
                <small>{{ summary.count }} 个账号</small>
              </div>
              <div class="sp-batch-test-model-select">
                <span>测试模型</span>
                <Select
                  v-model="batchTestModelByPlatform[summary.platform]"
                  :options="batchTestModelSelectOptions(summary.platform)"
                  :disabled="batchTestModelLoadingByPlatform[summary.platform]"
                  :searchable="true"
                />
              </div>
            </div>
          </div>
        </section>

        <section class="sp-batch-test-section settings-section">
          <header>
            <span class="sp-batch-section-index">02</span>
            <div>
              <strong>执行参数</strong>
              <span>批量任务会在后台执行，可以随时查看进度或取消。</span>
            </div>
          </header>
          <div class="sp-batch-test-settings">
            <label>
              <span>并发数</span>
              <small>同时测试的账号数量</small>
              <Select
                v-model="batchTestConcurrency"
                :options="batchTestConcurrencyOptions"
                :searchable="false"
              />
            </label>
            <label>
              <span>单账号超时</span>
              <small>超过时限后记为超时</small>
              <Select
                v-model="batchTestTimeoutSeconds"
                :options="batchTestTimeoutOptions"
                :searchable="false"
              />
            </label>
          </div>
        </section>
      </div>
      <template #footer>
        <button class="sp-button ghost sp-batch-secondary-button" type="button" :disabled="batchTesting" @click="closeBatchTestConfigDialog">
          取消
        </button>
        <button class="sp-button sp-batch-start-button" type="button" :disabled="batchTesting || batchTestTargets.length === 0" @click="startSupplierBatchTest">
          <span aria-hidden="true">{{ batchTesting ? '◌' : '▶' }}</span>
          {{ batchTesting ? '启动中…' : `开始测试 ${batchTestTargets.length} 个账号` }}
        </button>
      </template>
    </BaseDialog>

    <BaseDialog
      :show="showBatchTestResultDialog"
      title="批量测试结果"
      width="full"
      @close="closeBatchTestResultDialog"
    >
      <div class="sp-batch-result-dialog batch-test-result-modal">
        <p class="batch-result-progress-description">
          {{ batchTestProgressDescription }}
        </p>
        <div class="sync-confirm-summary">
          <div class="sync-result-stat total">
            <span>总数</span>
            <strong>{{ batchTestResult?.total || 0 }}</strong>
          </div>
          <div class="sync-result-stat completed">
            <span>已完成</span>
            <strong>{{ batchTestResult?.completed || 0 }}</strong>
          </div>
          <div class="sync-result-stat success">
            <span>成功</span>
            <strong>{{ batchTestResult?.success || 0 }}</strong>
          </div>
          <div class="sync-result-stat failed">
            <span>失败</span>
            <strong>{{ batchTestResult?.failed || 0 }}</strong>
          </div>
        </div>
        <div class="sync-confirm-body">
          <section class="sync-confirm-section">
            <div class="sync-confirm-section-title">
              <span>测试结果</span>
              <strong>{{ batchTestResultItems.length }}</strong>
            </div>
            <div v-if="batchTestResult?.error_message" class="sp-alert sp-error-line">
              {{ batchTestResult.error_message }}
            </div>
            <div v-if="batchTestResultItems.length" class="batch-result-scroll">
              <div class="batch-result-toolbar">
                <div class="batch-result-tabs" aria-label="批量测试结果筛选">
                  <button
                    v-for="option in batchTestResultFilterOptions"
                    :key="option.key"
                    :data-test="`supplier-batch-result-filter-${option.key}`"
                    :class="[
                      'batch-result-tab',
                      `batch-result-tab-${option.key}`,
                      { active: batchTestResultFilter === option.key },
                    ]"
                    type="button"
                    @click="batchTestResultFilter = option.key"
                  >
                    <span>{{ option.label }}</span>
                    <strong>{{ option.count }}</strong>
                  </button>
                </div>
                <div class="batch-result-hint">
                  <span>异常结果优先展示</span>
                  <span class="batch-result-hint-tag">失败优先</span>
                </div>
              </div>
              <div v-if="filteredBatchTestResultItems.length" class="batch-result-list">
                <article
                  v-for="item in filteredBatchTestResultItems"
                  :key="item.account_id"
                  :class="[
                    'batch-result-card',
                    batchTestItemTone(item.status),
                    { 'failed-schedulable': batchTestIsFailedSchedulable(item) },
                  ]"
                >
                  <div class="batch-result-card-head">
                    <div class="batch-result-account">
                      <strong>{{ item.account_name || batchTestTargetName(item.account_id) }}</strong>
                      <span>#{{ item.account_id }}</span>
                    </div>
                    <div class="batch-result-card-status">
                      <span :class="['sp-test-status', batchTestItemTone(item.status)]">
                        {{ batchTestItemStatusLabel(item.status) }}
                      </span>
                      <span v-if="batchTestIsFailedSchedulable(item)" class="batch-result-risk-tag">
                        失败但仍在调度
                      </span>
                    </div>
                  </div>
                  <div class="batch-result-grid">
                    <div class="batch-result-metric">
                      <span>平台</span>
                      <strong>
                        <span
                          :class="['sp-platform-badge', platformBadgeClass(item.platform || batchTestTargetPlatform(item.account_id))]"
                        >{{ batchPlatformLabel(item.platform || batchTestTargetPlatform(item.account_id)) }}</span>
                      </strong>
                    </div>
                    <div class="batch-result-metric">
                      <span>延迟</span>
                      <strong>{{ item.latency_ms > 0 ? `${item.latency_ms} ms` : '—' }}</strong>
                    </div>
                    <div class="batch-result-metric">
                      <span>账号 ID</span>
                      <strong>{{ item.account_id }}</strong>
                    </div>
                    <div class="batch-result-metric">
                      <span>是否调度</span>
                      <strong
                        :class="[
                          'batch-result-schedule-status',
                          batchTestItemSchedulable(item) ? 'enabled' : 'disabled',
                        ]"
                      >
                        {{ batchTestItemSchedulable(item) ? '调度打开' : '调度关闭' }}
                      </strong>
                    </div>
                    <div class="batch-result-metric">
                      <span>完成时间</span>
                      <strong>{{ item.finished_at ? formatTime(item.finished_at) : '—' }}</strong>
                    </div>
                  </div>
                  <div v-if="item.error_message" class="batch-result-error">{{ item.error_message }}</div>
                  <div class="batch-result-card-actions">
                    <button
                      type="button"
                      :class="[
                        'batch-result-schedule-button',
                        batchTestItemSchedulable(item) ? 'disable-action' : 'enable-action',
                      ]"
                      :disabled="togglingSchedulableID !== null"
                      :data-test="`supplier-batch-result-schedulable-toggle-${item.account_id}`"
                      @click="toggleBatchTestItemSchedulable(item)"
                    >
                      {{ togglingSchedulableID === item.account_id
                        ? '处理中…'
                        : batchTestItemSchedulable(item) ? '关闭调度' : '打开调度' }}
                    </button>
                  </div>
                </article>
              </div>
              <div v-else class="batch-result-empty">当前筛选下暂无测试结果</div>
            </div>
            <div v-else class="batch-result-empty">
              {{ batchTesting ? '测试任务正在排队，请稍候…' : '暂无测试结果' }}
            </div>
          </section>
        </div>
      </div>
      <template #footer>
        <button
          v-if="batchTestCanCancel"
          class="batch-result-button"
          type="button"
          :disabled="batchTestCancelling"
          @click="cancelSupplierBatchTest"
        >{{ batchTestCancelling ? '取消中…' : '取消测试' }}</button>
        <button
          class="batch-result-button batch-result-button-primary"
          type="button"
          @click="closeBatchTestResultDialog"
        >关闭</button>
      </template>
    </BaseDialog>

    <EditAccountModal
      v-if="showEditAccountModal"
      :show="showEditAccountModal"
      :account="editingAccount"
      :proxies="accountProxies"
      :groups="accountEditGroups"
      @close="closeLocalAccountEditor"
      @updated="handleLocalAccountUpdated"
    />

    <AccountTestModal
      :show="showAccountTestModal"
      :account="testingAccount"
      @close="closeAccountTestModal"
      @test-result="handleAccountTestResult"
    />

    <BaseDialog
      :show="Boolean(bindingAccount)"
      title="编辑账号绑定"
      width="wide"
      @close="closeAccountBindingEditor"
    >
      <div
        v-if="bindingAccount"
        class="sp-account-binding-dialog"
      >
        <div class="sp-account-binding-summary">
          <span class="sp-account-binding-accent" :class="platformAccentBarClass(bindingPlatform || '')"></span>
          <div class="sp-account-binding-selected-head">
            <span>已选分组</span>
            <strong :class="platformTextClass(bindingPlatform || '')">
              {{ selectedBindingGroupIDs.length }} 个分组
            </strong>
          </div>
          <div
            v-if="selectedBindingGroups.length > 0"
            class="sp-account-binding-selected-groups"
          >
            <GroupBadge
              v-for="group in selectedBindingGroups"
              :key="group.id"
              :name="group.name"
              :platform="group.platform"
              :subscription-type="group.subscription_type"
              :rate-multiplier="group.rate_multiplier"
              show-rate
            />
          </div>
          <div v-else class="sp-account-binding-empty">
            当前未绑定任何分组，保存后该账号将不会参与分组调度。
          </div>
        </div>
        <GroupSelector
          v-model="selectedBindingGroupIDs"
          :groups="accountEditGroups"
          :platform="bindingPlatform"
          searchable
        />
      </div>
      <template #footer>
        <button
          class="sp-button ghost"
          type="button"
          :disabled="savingBindingAccountID !== null"
          @click="closeAccountBindingEditor"
        >取消</button>
        <button
          :class="['sp-account-binding-primary', platformButtonClass(bindingPlatform || '')]"
          type="button"
          :disabled="savingBindingAccountID !== null"
          @click="saveAccountBinding"
        >{{ savingBindingAccountID !== null ? '保存中' : '保存绑定' }}</button>
      </template>
    </BaseDialog>

    <SupplierAccountRateGuardLogDialog
      :show="accountRateGuardLogsVisible"
      @close="closeAccountRateGuardLogs"
      @pending-count-change="updateAccountRateGuardPendingCount"
    />

    <ConfirmDialog
      :show="Boolean(duplicateConfirmAccount)"
      title="确认复制本地账号"
      :message="duplicateConfirmMessage"
      confirm-text="确认复制"
      cancel-text="取消"
      @confirm="confirmDuplicateLocalAccount"
      @cancel="closeDuplicateConfirm"
    />

    <ConfirmDialog
      :show="Boolean(deleteSupplierAccountRecordTarget)"
      title="删除上游账号记录"
      :message="deleteSupplierAccountRecordMessage"
      confirm-text="删除记录"
      cancel-text="取消"
      danger
      @confirm="confirmDeleteSupplierAccountRecord"
      @cancel="deleteSupplierAccountRecordTarget = null"
    />

    <BaseDialog
      :show="Boolean(duplicateResultAccount)"
      title="账号已复制"
      width="normal"
      @close="closeDuplicateResult"
    >
      <div v-if="duplicateResultAccount" class="space-y-3 text-sm text-gray-600 dark:text-gray-300">
        <p>
          已创建本地账号「<strong class="text-gray-900 dark:text-gray-100">{{ duplicateResultAccount.name }}</strong>」，并已暂停调度，请确认凭据后再启用。
        </p>
        <p>
          供应商上游账号页按「上游账号」列出，并通过本地账号名自动匹配；副本名称带有 (Copy) 后缀，不会作为新的上游行出现在本页。
        </p>
        <p>
          可在账号管理中按名称搜索查看，或直接编辑刚复制的账号。
        </p>
      </div>
      <template #footer>
        <button class="sp-button ghost" type="button" @click="closeDuplicateResult">关闭</button>
        <button class="sp-button" type="button" @click="openDuplicatedAccountEditor">编辑新账号</button>
        <button class="sp-button" type="button" @click="goToAccountManagement">去账号管理</button>
      </template>
    </BaseDialog>
  </SupplierModuleLayout>
</template>
<script setup lang="ts">
import { computed, nextTick, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import { useRouter } from 'vue-router'
import AccountTestModal from '@/components/admin/account/AccountTestModal.vue'
import { SupplierAccountRateGuardLogDialog, SupplierDrawer, SupplierModuleLayout } from '@/components/admin/supplier-management'
import { CreateAccountModal, EditAccountModal } from '@/components/account'
import DataTable from '@/components/common/DataTable.vue'
import BaseDialog from '@/components/common/BaseDialog.vue'
import ConfirmDialog from '@/components/common/ConfirmDialog.vue'
import GroupBadge from '@/components/common/GroupBadge.vue'
import GroupSelector from '@/components/common/GroupSelector.vue'
import Input from '@/components/common/Input.vue'
import Pagination from '@/components/common/Pagination.vue'
import Select, { type SelectOption } from '@/components/common/Select.vue'
import supplierProvidersAPI, { type SupplierProvider } from '@/api/admin/supplierProviders'
import { customPlatformsAPI, type CustomPlatform } from '@/api/admin/customPlatforms'
import {
  cancelSupplierAccountBatchTestJob,
  clearSupplierLocalAccountPlatformOverride,
  deleteSupplierAccount,
  getSupplierAccountBatchTestJob,
  listSupplierAccounts,
  setSupplierLocalAccountPlatformOverride,
  startSupplierAccountBatchTest,
  type SupplierProviderAccount,
} from '@/api/admin/supplierProviderData'
import { listAccountRateGuardUnbindLogs } from '@/api/admin/supplierAutomation'
import { adminAPI } from '@/api/admin'
import { useAppStore } from '@/stores/app'
import type { Column } from '@/components/common/types'
import type {
  Account,
  AdminGroup,
  BatchAccountTestItem,
  BatchAccountTestJob,
  BatchAccountTestStatus,
  ClaudeModel,
  GroupPlatform,
  Proxy as AccountProxy,
} from '@/types'
import { ensureCustomPlatformLabels, resolvePlatformDisplayLabel as platformLabel } from '@/utils/customPlatformLabels'
import {
  platformAccentBarClass,
  platformBadgeClass,
  platformButtonClass,
  platformTextClass,
} from '@/utils/platformColors'
import { extractApiErrorMessage } from '@/utils/apiError'

const appStore = useAppStore()
const router = useRouter()

type SupplierBatchTestTarget = {
  accountID: number
  accountName: string
  platform: string
  schedulable: boolean
  providerEnabled: boolean
}

type SupplierBatchTestPlatformSummary = {
  platform: string
  count: number
  representativeAccountID: number
}

type SupplierAccountFilterSnapshot = {
  providerID: number
  groupID: number
  platform: string
  active?: boolean
  status?: string
  search?: string
  quickFilter: AccountQuickFilterKey
  summary: string
}

type AccountQuickFilterKey = 'all' | 'bound' | 'unbound' | 'schedulable' | 'paused' | 'failed' | 'group_deleted' | 'upstream_deleted'

type AccountQuickFilterOption = {
  key: AccountQuickFilterKey
  label: string
  count: number
}

type SupplierBatchResultFilter =
  | 'all'
  | 'failed'
  | 'failed_schedulable'
  | 'failed_unschedulable'
  | 'success'
  | 'success_unschedulable'
  | 'success_upstream_disabled'
  | 'skipped'

const providers = ref<SupplierProvider[]>([])
const localGroups = ref<AdminGroup[]>([])
const customPlatforms = ref<CustomPlatform[]>([])
const accountSourceItems = ref<SupplierProviderAccount[]>([])
const items = ref<SupplierProviderAccount[]>([])
const selected = ref<SupplierProviderAccount | null>(null)
const testErrorAccount = ref<SupplierProviderAccount | null>(null)
const togglingSchedulableID = ref<number | null>(null)
const editingPriorityAccountID = ref<number | null>(null)
const savingPriorityAccountID = ref<number | null>(null)
const accountActionLoadingID = ref<number | null>(null)
const deletingAccountID = ref<number | null>(null)
const deletingSupplierAccountRecordID = ref<number | null>(null)
const duplicatingAccountID = ref<number | null>(null)
const duplicateConfirmAccount = ref<SupplierProviderAccount | null>(null)
const deleteSupplierAccountRecordTarget = ref<SupplierProviderAccount | null>(null)
const duplicateResultAccount = ref<Account | null>(null)
const testingAccountID = ref<number | null>(null)
const testingAccount = ref<Account | null>(null)
const editingAccount = ref<Account | null>(null)
const showEditAccountModal = ref(false)
const showAccountTestModal = ref(false)
const showCreateAccountModal = ref(false)
const showBatchTestConfigDialog = ref(false)
const showBatchTestResultDialog = ref(false)
const accountRateGuardLogsVisible = ref(false)
const accountRateGuardPendingCount = ref(0)
const businessPlatformAccount = ref<SupplierProviderAccount | null>(null)
const businessPlatformDraft = ref('')
const savingBusinessPlatform = ref(false)
const batchTestPreparing = ref(false)
const batchTesting = ref(false)
const batchTestCancelling = ref(false)
const batchTestTargets = ref<SupplierBatchTestTarget[]>([])
const batchTestResult = ref<BatchAccountTestJob | null>(null)
const batchTestResultFilter = ref<SupplierBatchResultFilter>('all')
const batchTestFilterSummary = ref('')
const batchTestModelByPlatform = ref<Record<string, string>>({})
const batchTestModelOptionsByPlatform = ref<Record<string, ClaudeModel[]>>({})
const batchTestModelLoadingByPlatform = ref<Record<string, boolean>>({})
const batchTestConcurrency = ref(3)
const batchTestTimeoutSeconds = ref(90)
const accountEditGroups = ref<AdminGroup[]>([])
const accountProxies = ref<AccountProxy[]>([])
const bindingAccount = ref<Account | null>(null)
const selectedBindingGroupIDs = ref<number[]>([])
const bindingPlatform = ref<GroupPlatform | undefined>()
const selectedBindingGroups = computed(() => {
  const selectedIDs = new Set(selectedBindingGroupIDs.value)
  return accountEditGroups.value.filter(group => selectedIDs.has(group.id))
})
const sortBy = ref('')
const sortOrder = ref<'asc' | 'desc'>('asc')
const accountQuickFilter = ref<AccountQuickFilterKey>('all')
const savingBindingAccountID = ref<number | null>(null)
const priorityDraft = ref('')
const priorityInput = ref<InstanceType<typeof Input> | null>(null)
const total = ref(0)
const loading = ref(false)
const error = ref('')
const page = ref(1)
const pageSize = ref(20)
const providerID = ref(0)
const groupID = ref(0)
const platformFilter = ref('')
const activeFilter = ref('')
const upstreamStatusFilter = ref('')
const search = ref('')
const searchFilterControl = ref<HTMLElement | null>(null)
const providerFilterControl = ref<HTMLElement | null>(null)
const groupFilterControl = ref<HTMLElement | null>(null)
const platformFilterControl = ref<HTMLElement | null>(null)
const activeFilterControl = ref<HTMLElement | null>(null)
const upstreamStatusFilterControl = ref<HTMLElement | null>(null)
let searchTimer: number | undefined
let batchTestPollTimer: ReturnType<typeof setTimeout> | null = null
let batchTestPollToken = 0

const cnyFormatter = new Intl.NumberFormat('zh-CN', {
  style: 'currency',
  currency: 'CNY',
  minimumFractionDigits: 2,
  maximumFractionDigits: 2,
})

const providerOptions = computed<SelectOption[]>(() => [
  { value: 0, label: '全部供应商' },
  ...providers.value.map(provider => ({ value: provider.id, label: provider.name })),
])
const localGroupOptions = computed<SelectOption[]>(() => [
  { value: 0, label: '全部本地分组' },
  ...localGroups.value
    .filter(group => !platformFilter.value || group.platform === platformFilter.value)
    .map(group => ({
      value: group.id,
      label: `${group.name} #${group.id}`,
    })),
])
const accountQuickFilterOptions = computed<AccountQuickFilterOption[]>(() => [
  { key: 'all', label: '全部', count: accountSourceItems.value.length },
  {
    key: 'bound',
    label: '已绑定分组',
    count: accountSourceItems.value.filter(account => accountMatchesQuickFilter(account, 'bound')).length,
  },
  {
    key: 'unbound',
    label: '未绑定分组',
    count: accountSourceItems.value.filter(account => accountMatchesQuickFilter(account, 'unbound')).length,
  },
  {
    key: 'schedulable',
    label: '可参与调度',
    count: accountSourceItems.value.filter(account => accountMatchesQuickFilter(account, 'schedulable')).length,
  },
  {
    key: 'paused',
    label: '暂停调度',
    count: accountSourceItems.value.filter(account => accountMatchesQuickFilter(account, 'paused')).length,
  },
  {
    key: 'failed',
    label: '测试失败',
    count: accountSourceItems.value.filter(account => accountMatchesQuickFilter(account, 'failed')).length,
  },
  {
    key: 'group_deleted',
    label: '上游分组已删除',
    count: accountSourceItems.value.filter(account => accountMatchesQuickFilter(account, 'group_deleted')).length,
  },
  {
    key: 'upstream_deleted',
    label: '上游密钥已删除',
    count: accountSourceItems.value.filter(account => accountMatchesQuickFilter(account, 'upstream_deleted')).length,
  },
])
const supplierIDs = computed(() => [...new Set([
  ...providers.value.map(provider => provider.id),
  ...accountSourceItems.value.map(account => account.provider_id),
])].sort((left, right) => left - right))
const activeFilterOptions: SelectOption[] = [
  { value: 'true', label: '仅有效' },
  { value: '', label: '全部有效性' },
  { value: 'false', label: '已失效' },
]
const upstreamStatusFilterOptions: SelectOption[] = [
  { value: '', label: '全部上游状态' },
  { value: 'active', label: '正常' },
  { value: 'disabled', label: '停用' },
  { value: 'expired', label: '已过期' },
  { value: 'quota_exhausted', label: '额度耗尽' },
  { value: 'unknown', label: '未知' },
  { value: 'deleted', label: '已删除' },
]
const corePlatformOptions: SelectOption[] = [
  { value: 'anthropic', label: 'Anthropic' },
  { value: 'openai', label: 'OpenAI' },
  { value: 'gemini', label: 'Gemini' },
  { value: 'antigravity', label: 'Antigravity' },
  { value: 'grok', label: 'Grok' },
]
const customPlatformOptions = computed<SelectOption[]>(() => customPlatforms.value
  .filter(platform => platform.enabled)
  .map(platform => ({ value: platform.code, label: platform.name })))
const platformFilterOptions = computed<SelectOption[]>(() => [
  { value: '', label: '全部平台' },
  ...corePlatformOptions,
  ...customPlatformOptions.value,
])
const businessPlatformOptions = computed<SelectOption[]>(() => [
  { value: '', label: '业务平台跟随接入平台' },
  ...corePlatformOptions,
  ...customPlatformOptions.value,
])
const pageSizeOptions: SelectOption[] = [
  { value: 20, label: '20' },
  { value: 50, label: '50' },
  { value: 100, label: '100' },
]
const batchTestConcurrencyOptions: SelectOption[] = [
  { value: 1, label: '1 个' },
  { value: 3, label: '3 个' },
  { value: 5, label: '5 个' },
  { value: 8, label: '8 个' },
]
const batchTestTimeoutOptions: SelectOption[] = [
  { value: 30, label: '30 秒' },
  { value: 60, label: '60 秒' },
  { value: 90, label: '90 秒' },
  { value: 120, label: '120 秒' },
]
const SUPPLIER_ACCOUNT_FILTER_PAGE_SIZE = 200
const SUPPLIER_BATCH_TEST_PAGE_SIZE = 200
const MAX_SUPPLIER_BATCH_TEST_ACCOUNTS = 200
const SUPPLIER_BATCH_TEST_POLL_INTERVAL_MS = 1000
const SUPPLIER_BATCH_TEST_TOTAL_TIMEOUT_SECONDS = 600
const SUPPLIER_TONES = [
  { chip: 'border-sky-500/30 bg-sky-500/10 text-sky-700 dark:text-sky-300', dot: 'bg-sky-500' },
  { chip: 'border-orange-500/30 bg-orange-500/10 text-orange-700 dark:text-orange-300', dot: 'bg-orange-500' },
  { chip: 'border-emerald-500/30 bg-emerald-500/10 text-emerald-700 dark:text-emerald-300', dot: 'bg-emerald-500' },
  { chip: 'border-violet-500/30 bg-violet-500/10 text-violet-700 dark:text-violet-300', dot: 'bg-violet-500' },
  { chip: 'border-rose-500/30 bg-rose-500/10 text-rose-700 dark:text-rose-300', dot: 'bg-rose-500' },
  { chip: 'border-cyan-500/30 bg-cyan-500/10 text-cyan-700 dark:text-cyan-300', dot: 'bg-cyan-500' },
  { chip: 'border-amber-500/30 bg-amber-500/10 text-amber-700 dark:text-amber-300', dot: 'bg-amber-500' },
  { chip: 'border-indigo-500/30 bg-indigo-500/10 text-indigo-700 dark:text-indigo-300', dot: 'bg-indigo-500' },
  { chip: 'border-lime-500/30 bg-lime-500/10 text-lime-700 dark:text-lime-300', dot: 'bg-lime-500' },
  { chip: 'border-fuchsia-500/30 bg-fuchsia-500/10 text-fuchsia-700 dark:text-fuchsia-300', dot: 'bg-fuchsia-500' },
  { chip: 'border-teal-500/30 bg-teal-500/10 text-teal-700 dark:text-teal-300', dot: 'bg-teal-500' },
  { chip: 'border-red-500/30 bg-red-500/10 text-red-700 dark:text-red-300', dot: 'bg-red-500' },
]
const batchTestPlatformSummaries = computed<SupplierBatchTestPlatformSummary[]>(() => {
  const summaries = new Map<string, SupplierBatchTestPlatformSummary>()
  for (const target of batchTestTargets.value) {
    const platform = target.platform || 'unknown'
    const current = summaries.get(platform)
    if (current) {
      current.count += 1
      continue
    }
    summaries.set(platform, {
      platform,
      count: 1,
      representativeAccountID: target.accountID,
    })
  }
  return [...summaries.values()].sort((left, right) => left.platform.localeCompare(right.platform))
})

function buildSupplierFilterSummary(snapshot: Omit<SupplierAccountFilterSnapshot, 'summary'>): string {
  const provider = snapshot.providerID
    ? providers.value.find(item => item.id === snapshot.providerID)?.name || `供应商 #${snapshot.providerID}`
    : '全部供应商'
  const group = snapshot.groupID
    ? localGroups.value.find(item => item.id === snapshot.groupID)?.name || `本地分组 #${snapshot.groupID}`
    : '全部本地分组'
  const platform = snapshot.platform ? platformLabel(snapshot.platform) : '全部平台'
  const active = snapshot.active === undefined
    ? '全部有效性'
    : activeFilterOptions.find(option => option.value === String(snapshot.active))?.label || '全部有效性'
  const status = snapshot.status
    ? upstreamStatusFilterOptions.find(option => option.value === snapshot.status)?.label || snapshot.status
    : '全部上游状态'
  const keyword = snapshot.search?.trim() || ''
  const quickFilter = accountQuickFilterOptions.value.find(option => option.key === snapshot.quickFilter)?.label || '全部'
  return [
    provider,
    group,
    platform,
    active,
    status,
    `快捷过滤：${quickFilter}`,
    keyword ? `关键词：${keyword}` : '无搜索关键词',
  ].join(' · ')
}

function createSupplierAccountFilterSnapshot(): SupplierAccountFilterSnapshot {
  const snapshot = {
    providerID: providerID.value || 0,
    groupID: groupID.value || 0,
    platform: platformFilter.value || '',
    active: activeFilter.value === '' ? undefined : activeFilter.value === 'true',
    status: upstreamStatusFilter.value || undefined,
    search: search.value.trim() || undefined,
    quickFilter: accountQuickFilter.value,
  }
  return {
    ...snapshot,
    summary: buildSupplierFilterSummary(snapshot),
  }
}

const batchTestResultItems = computed<BatchAccountTestItem[]>(() => (batchTestResult.value?.results || [])
  .map((item, index) => ({ item, index }))
  .sort((left, right) => {
    const priorityDifference = batchTestResultPriority(left.item) - batchTestResultPriority(right.item)
    return priorityDifference || left.index - right.index
  })
  .map(entry => entry.item))

const batchTestResultCounts = computed(() => {
  const resultItems = batchTestResultItems.value
  return {
    all: resultItems.length,
    failed: resultItems.filter(batchTestIsFailed).length,
    failedSchedulable: resultItems.filter(batchTestIsFailedSchedulable).length,
    failedUnschedulable: resultItems.filter(batchTestIsFailedUnschedulable).length,
    success: resultItems.filter(item => item.status === 'success').length,
    successUnschedulable: resultItems.filter(batchTestIsSuccessUnschedulable).length,
    successUpstreamDisabled: resultItems.filter(batchTestIsSuccessUpstreamDisabled).length,
    skipped: resultItems.filter(batchTestIsSkipped).length,
  }
})

const batchTestResultFilterOptions = computed(() => {
  const counts = batchTestResultCounts.value
  return [
    { key: 'all' as const, label: '全部', count: counts.all },
    { key: 'failed' as const, label: '失败', count: counts.failed },
    { key: 'failed_schedulable' as const, label: '失败且调度打开', count: counts.failedSchedulable },
    { key: 'failed_unschedulable' as const, label: '失败且调度关闭', count: counts.failedUnschedulable },
    { key: 'success' as const, label: '成功', count: counts.success },
    { key: 'success_unschedulable' as const, label: '成功且调度关闭', count: counts.successUnschedulable },
    { key: 'success_upstream_disabled' as const, label: '成功且上游禁用', count: counts.successUpstreamDisabled },
    { key: 'skipped' as const, label: '跳过', count: counts.skipped },
  ]
})

const filteredBatchTestResultItems = computed<BatchAccountTestItem[]>(() => {
  const resultItems = batchTestResultItems.value
  if (batchTestResultFilter.value === 'failed') return resultItems.filter(batchTestIsFailed)
  if (batchTestResultFilter.value === 'failed_schedulable') return resultItems.filter(batchTestIsFailedSchedulable)
  if (batchTestResultFilter.value === 'failed_unschedulable') return resultItems.filter(batchTestIsFailedUnschedulable)
  if (batchTestResultFilter.value === 'success') return resultItems.filter(item => item.status === 'success')
  if (batchTestResultFilter.value === 'success_unschedulable') return resultItems.filter(batchTestIsSuccessUnschedulable)
  if (batchTestResultFilter.value === 'success_upstream_disabled') return resultItems.filter(batchTestIsSuccessUpstreamDisabled)
  if (batchTestResultFilter.value === 'skipped') return resultItems.filter(batchTestIsSkipped)
  return resultItems
})

const batchTestCanCancel = computed(() => {
  const status = batchTestResult.value?.status
  return status === 'queued' || status === 'running'
})

const batchTestProgressDescription = computed(() => {
  const job = batchTestResult.value
  if (!job) return '正在创建批量测试任务…'
  return `已完成 ${job.completed} / ${job.total}，成功 ${job.success}，失败 ${job.failed}`
})

const accountColumns: Column[] = [
  { key: 'provider_name', label: '供应商', sortable: true, class: 'min-w-[190px]' },
  { key: 'upstream_account_key', label: '上游账号', sortable: true, class: 'min-w-[260px]' },
  { key: 'upstream_account_status', label: '上游状态', sortable: true, class: 'min-w-[110px]' },
  { key: 'local_account_name', label: '本地账号', sortable: true, class: 'min-w-[190px]' },
  { key: 'local_account_priority', label: '优先级', sortable: true, class: 'min-w-[88px]' },
  { key: 'rate_multiplier', label: '上游倍率', sortable: true, class: 'min-w-[104px]' },
  { key: 'group_name', label: '账号绑定的分组', class: 'min-w-[260px]' },
  { key: 'local_account_status', label: '本地账号状态', sortable: true, class: 'min-w-[136px]' },
  { key: 'local_account_schedulable', label: '是否调度', sortable: true, class: 'min-w-[104px]' },
  { key: 'local_account_last_test_status', label: '测试结果', sortable: true, class: 'min-w-[120px]' },
  { key: 'local_account_last_tested_at', label: '上次测试时间', sortable: true, class: 'min-w-[172px]' },
  { key: 'supplier_current_balance', label: '余额', sortable: true, class: 'min-w-[142px]' },
  { key: 'supplier_today_cost', label: '今日消费', sortable: true, class: 'min-w-[142px]' },
  { key: 'actions', label: '操作', class: 'min-w-[300px]' },
]

onMounted(async () => {
  applyFilterControlLabels()
  await Promise.all([ensureCustomPlatformLabels(), loadCustomPlatforms(), loadProviders(), loadLocalGroups(), loadAccountRateGuardPendingCount()])
  await loadAccounts()
})

onBeforeUnmount(() => {
  window.clearTimeout(searchTimer)
  batchTestPollToken += 1
  clearBatchTestPollTimer()
})

watch([providerID, groupID, activeFilter, upstreamStatusFilter], () => {
  resetPageAndLoad()
})

watch(platformFilter, platform => {
  if (groupID.value && platform) {
    const selectedGroup = localGroups.value.find(group => group.id === groupID.value)
    if (selectedGroup?.platform !== platform) {
      groupID.value = 0
      return
    }
  }
  resetPageAndLoad()
})

watch(search, () => {
  window.clearTimeout(searchTimer)
  searchTimer = window.setTimeout(resetPageAndLoad, 350)
})

async function loadProviders() {
  const result = await supplierProvidersAPI.list({ page: 1, page_size: 200 })
  providers.value = result.items
}

async function loadLocalGroups() {
  localGroups.value = await adminAPI.groups.getAll()
}

async function loadCustomPlatforms() {
  customPlatforms.value = await customPlatformsAPI.list(true)
}

function accountMatchesQuickFilter(
  account: SupplierProviderAccount,
  filter: AccountQuickFilterKey
): boolean {
  if (filter === 'all') return true
  if (filter === 'bound') return account.binding_groups.length > 0
  if (filter === 'unbound') return account.binding_groups.length === 0
  if (filter === 'failed') return account.local_account_last_test_status === 'failed'
  if (filter === 'group_deleted') {
    return account.group_status === 'inactive' || account.group_status === 'missing'
  }
  if (filter === 'upstream_deleted') {
    return account.status === 'deleted'
  }
  if (account.local_account_match_status !== 'matched') return false
  if (filter === 'schedulable') return account.local_account_schedulable === true
  return account.local_account_schedulable === false
}

function applyAccountQuickFilterPage() {
  const filteredAccounts = accountSourceItems.value.filter(account => (
    accountMatchesQuickFilter(account, accountQuickFilter.value)
  ))
  total.value = filteredAccounts.length
  const lastPage = Math.max(1, Math.ceil(total.value / pageSize.value))
  if (page.value > lastPage) page.value = lastPage
  const start = (page.value - 1) * pageSize.value
  items.value = filteredAccounts.slice(start, start + pageSize.value)
}

function selectAccountQuickFilter(filter: AccountQuickFilterKey) {
  if (accountQuickFilter.value === filter) return
  accountQuickFilter.value = filter
  page.value = 1
  applyAccountQuickFilterPage()
}

async function loadAccounts() {
  loading.value = true
  error.value = ''
  try {
    const loadedAccounts: SupplierProviderAccount[] = []
    let nextPage = 1

    while (true) {
      const result = await listSupplierAccounts({
        provider_id: providerID.value || undefined,
        group_id: groupID.value || undefined,
        platform: platformFilter.value || undefined,
        active: activeFilter.value === '' ? undefined : activeFilter.value === 'true',
        status: upstreamStatusFilter.value || undefined,
        search: search.value.trim() || undefined,
        sort_by: sortBy.value || undefined,
        sort_order: sortBy.value ? sortOrder.value : undefined,
        page: nextPage,
        page_size: SUPPLIER_ACCOUNT_FILTER_PAGE_SIZE,
      })
      loadedAccounts.push(...result.items)
      if (loadedAccounts.length >= result.total || result.items.length === 0) break
      nextPage += 1
    }

    accountSourceItems.value = loadedAccounts
    applyAccountQuickFilterPage()
  } catch (err) {
    error.value = err instanceof Error ? err.message : '加载账号数据失败'
  } finally {
    loading.value = false
  }
}

async function loadFilteredTestAccounts(snapshot: SupplierAccountFilterSnapshot): Promise<SupplierProviderAccount[]> {
  const filteredAccounts: SupplierProviderAccount[] = []
  let nextPage = 1

  while (true) {
    const result = await listSupplierAccounts({
      provider_id: snapshot.providerID || undefined,
      group_id: snapshot.groupID || undefined,
      platform: snapshot.platform || undefined,
      active: snapshot.active,
      status: snapshot.status,
      search: snapshot.search,
      page: nextPage,
      page_size: SUPPLIER_BATCH_TEST_PAGE_SIZE,
    })
    filteredAccounts.push(...result.items)
    if (filteredAccounts.length >= result.total || result.items.length === 0) break
    nextPage += 1
  }

  return filteredAccounts.filter(account => accountMatchesQuickFilter(account, snapshot.quickFilter))
}

function handleSupplierBatchTestButton() {
  if (batchTesting.value) {
    showBatchTestResultDialog.value = true
    return
  }
  void openSupplierBatchTestDialog(createSupplierAccountFilterSnapshot())
}

async function openSupplierBatchTestDialog(snapshot: SupplierAccountFilterSnapshot) {
  if (batchTestPreparing.value || batchTesting.value) return
  batchTestPreparing.value = true
  try {
    const filteredAccounts = await loadFilteredTestAccounts(snapshot)
    const uniqueTargets = new Map<number, SupplierBatchTestTarget>()
    for (const account of filteredAccounts) {
      if (account.local_account_match_status !== 'matched') continue
      const localAccountID = Number(account.local_account_id)
      if (!Number.isInteger(localAccountID) || localAccountID <= 0 || uniqueTargets.has(localAccountID)) continue
      uniqueTargets.set(localAccountID, {
        accountID: localAccountID,
        accountName: account.local_account_name || account.name || `账号 #${localAccountID}`,
        platform: effectivePlatform(account),
        schedulable: account.local_account_schedulable === true,
        providerEnabled: providers.value.find(provider => provider.id === account.provider_id)?.enabled !== false,
      })
    }

    const targets = [...uniqueTargets.values()]
    if (targets.length === 0) {
      appStore.showWarning('当前筛选条件下没有唯一匹配的本地账号可测试')
      return
    }
    if (targets.length > MAX_SUPPLIER_BATCH_TEST_ACCOUNTS) {
      appStore.showWarning(`当前筛选匹配账号超过 ${MAX_SUPPLIER_BATCH_TEST_ACCOUNTS} 个，请缩小筛选范围后重试`)
      return
    }

    batchTestTargets.value = targets
    batchTestModelByPlatform.value = {}
    batchTestModelOptionsByPlatform.value = {}
    batchTestModelLoadingByPlatform.value = {}
    batchTestResult.value = null
    batchTestResultFilter.value = 'all'
    batchTestFilterSummary.value = snapshot.summary
    showBatchTestConfigDialog.value = true
    await loadSupplierBatchTestModels()
  } catch (err) {
    appStore.showError(extractApiErrorMessage(err, '加载当前筛选账号失败'))
  } finally {
    batchTestPreparing.value = false
  }
}

async function loadSupplierBatchTestModels() {
  await Promise.all(batchTestPlatformSummaries.value.map(async summary => {
    batchTestModelByPlatform.value = {
      ...batchTestModelByPlatform.value,
      [summary.platform]: '',
    }
    if (summary.platform === 'unknown') return

    batchTestModelLoadingByPlatform.value = {
      ...batchTestModelLoadingByPlatform.value,
      [summary.platform]: true,
    }
    try {
      const models = await adminAPI.accounts.getAvailableModels(summary.representativeAccountID)
      batchTestModelOptionsByPlatform.value = {
        ...batchTestModelOptionsByPlatform.value,
        [summary.platform]: models,
      }
    } catch {
      batchTestModelOptionsByPlatform.value = {
        ...batchTestModelOptionsByPlatform.value,
        [summary.platform]: [],
      }
    } finally {
      batchTestModelLoadingByPlatform.value = {
        ...batchTestModelLoadingByPlatform.value,
        [summary.platform]: false,
      }
    }
  }))
}

function batchTestModelSelectOptions(platform: string): SelectOption[] {
  const loadingModels = batchTestModelLoadingByPlatform.value[platform]
  const models = batchTestModelOptionsByPlatform.value[platform] || []
  return [
    { value: '', label: loadingModels ? '加载模型中…' : '自动选择默认模型' },
    ...models.map(model => ({
      value: model.id,
      label: model.display_name || model.id,
    })),
  ]
}

function selectedBatchTestModelsByPlatform(): Record<string, string> {
  const result: Record<string, string> = {}
  for (const summary of batchTestPlatformSummaries.value) {
    const modelID = batchTestModelByPlatform.value[summary.platform]?.trim()
    if (modelID) result[summary.platform] = modelID
  }
  return result
}

// 将业务平台选中的模型展开到每个账号，避免本地账号原始 platform 与业务平台不一致时模型丢失。
function selectedBatchTestModelsByAccount(): Record<number, string> {
  const modelsByPlatform = selectedBatchTestModelsByPlatform()
  const result: Record<number, string> = {}
  for (const target of batchTestTargets.value) {
    const modelID = modelsByPlatform[target.platform]?.trim()
    if (modelID) result[target.accountID] = modelID
  }
  return result
}

async function startSupplierBatchTest() {
  if (batchTesting.value || batchTestTargets.value.length === 0) return
  batchTesting.value = true
  batchTestCancelling.value = false
  batchTestResult.value = null
  batchTestResultFilter.value = 'all'
  showBatchTestConfigDialog.value = false
  showBatchTestResultDialog.value = true
  clearBatchTestPollTimer()
  const pollToken = ++batchTestPollToken

  try {
    const modelIDsByPlatform = selectedBatchTestModelsByPlatform()
    const modelIDsByAccount = selectedBatchTestModelsByAccount()
    const job = await startSupplierAccountBatchTest({
      account_ids: batchTestTargets.value.map(target => target.accountID),
      ...(Object.keys(modelIDsByPlatform).length ? { model_ids_by_platform: modelIDsByPlatform } : {}),
      ...(Object.keys(modelIDsByAccount).length ? { model_ids_by_account: modelIDsByAccount } : {}),
      concurrency: batchTestConcurrency.value,
      timeout_per_account_seconds: batchTestTimeoutSeconds.value,
      timeout_seconds: SUPPLIER_BATCH_TEST_TOTAL_TIMEOUT_SECONDS,
    })
    batchTestResult.value = job
    showBatchTestResultDialog.value = true
    if (isSupplierBatchTestTerminal(job.status)) {
      await finishSupplierBatchTest(job, pollToken)
    } else {
      scheduleSupplierBatchTestPoll(job.job_id, pollToken)
    }
  } catch (err) {
    batchTesting.value = false
    showBatchTestResultDialog.value = false
    appStore.showError(extractApiErrorMessage(err, '启动供应商账号批量测试失败'))
  }
}

function scheduleSupplierBatchTestPoll(jobID: string, pollToken: number) {
  clearBatchTestPollTimer()
  batchTestPollTimer = setTimeout(async () => {
    if (pollToken !== batchTestPollToken) return
    try {
      const job = await getSupplierAccountBatchTestJob(jobID)
      if (pollToken !== batchTestPollToken) return
      batchTestResult.value = job
      if (isSupplierBatchTestTerminal(job.status)) {
        await finishSupplierBatchTest(job, pollToken)
      } else {
        scheduleSupplierBatchTestPoll(jobID, pollToken)
      }
    } catch (err) {
      if (pollToken !== batchTestPollToken) return
      batchTesting.value = false
      clearBatchTestPollTimer()
      appStore.showError(extractApiErrorMessage(err, '查询供应商账号批量测试进度失败'))
    }
  }, SUPPLIER_BATCH_TEST_POLL_INTERVAL_MS)
}

async function cancelSupplierBatchTest() {
  const jobID = batchTestResult.value?.job_id
  if (!jobID || !batchTestCanCancel.value || batchTestCancelling.value) return
  batchTestCancelling.value = true
  try {
    const job = await cancelSupplierAccountBatchTestJob(jobID)
    batchTestResult.value = job
    if (isSupplierBatchTestTerminal(job.status)) {
      await finishSupplierBatchTest(job, batchTestPollToken)
    }
  } catch (err) {
    appStore.showError(extractApiErrorMessage(err, '取消供应商账号批量测试失败'))
  } finally {
    batchTestCancelling.value = false
  }
}

async function finishSupplierBatchTest(job: BatchAccountTestJob, pollToken: number) {
  if (pollToken !== batchTestPollToken) return
  clearBatchTestPollTimer()
  batchTesting.value = false
  batchTestCancelling.value = false
  batchTestResult.value = job
  await loadAccounts()

  if (job.status === 'cancelled') {
    appStore.showWarning('供应商账号批量测试已取消')
  } else if (job.status === 'failed') {
    appStore.showError(job.error_message || '供应商账号批量测试失败')
  } else if (job.failed > 0) {
    appStore.showWarning(`批量测试完成：${job.success} 个成功，${job.failed} 个失败`)
  } else {
    appStore.showSuccess(`批量测试完成：${job.success} 个账号全部成功`)
  }
}

function clearBatchTestPollTimer() {
  if (batchTestPollTimer) {
    clearTimeout(batchTestPollTimer)
    batchTestPollTimer = null
  }
}

function closeBatchTestConfigDialog() {
  if (batchTesting.value) return
  showBatchTestConfigDialog.value = false
}

function closeBatchTestResultDialog() {
  showBatchTestResultDialog.value = false
}

function isSupplierBatchTestTerminal(status: BatchAccountTestJob['status']): boolean {
  return status === 'completed' || status === 'cancelled' || status === 'failed'
}

function batchPlatformLabel(platform?: string): string {
  return !platform || platform === 'unknown' ? '未知平台' : platformLabel(platform)
}

function normalizePlatform(platform?: string): string {
  return platform?.trim().toLowerCase() || ''
}

function effectivePlatform(account: SupplierProviderAccount): string {
  return normalizePlatform(account.effective_platform)
    || normalizePlatform(account.local_account_platform)
    || normalizePlatform(account.platform)
    || 'unknown'
}

function batchTestItemStatusLabel(status: BatchAccountTestStatus): string {
  if (status === 'success') return '成功'
  if (status === 'failed') return '失败'
  if (status === 'timeout') return '超时'
  if (status === 'not_found') return '账号不存在'
  return '已取消'
}

function batchTestIsSkipped(item: BatchAccountTestItem): boolean {
  return item.status === 'cancelled'
}

function batchTestIsFailed(item: BatchAccountTestItem): boolean {
  return item.status !== 'success' && !batchTestIsSkipped(item)
}

function batchTestItemSchedulable(item: BatchAccountTestItem): boolean {
  if (typeof item.schedulable === 'boolean') return item.schedulable
  return batchTestTargets.value.find(target => target.accountID === item.account_id)?.schedulable === true
}

function batchTestIsProviderDisabled(item: BatchAccountTestItem): boolean {
  return batchTestTargets.value.find(target => target.accountID === item.account_id)?.providerEnabled === false
}

function batchTestIsFailedSchedulable(item: BatchAccountTestItem): boolean {
  return batchTestIsFailed(item) && batchTestItemSchedulable(item)
}

function batchTestIsFailedUnschedulable(item: BatchAccountTestItem): boolean {
  return batchTestIsFailed(item) && !batchTestItemSchedulable(item)
}

function batchTestIsSuccessUnschedulable(item: BatchAccountTestItem): boolean {
  return item.status === 'success' && !batchTestItemSchedulable(item) && !batchTestIsProviderDisabled(item)
}

function batchTestIsSuccessUpstreamDisabled(item: BatchAccountTestItem): boolean {
  return item.status === 'success' && batchTestIsProviderDisabled(item)
}


function batchTestResultPriority(item: BatchAccountTestItem): number {
  if (batchTestIsFailedSchedulable(item)) return 0
  if (batchTestIsFailed(item)) return 1
  if (batchTestIsSkipped(item)) return 2
  return 3
}

function batchTestItemTone(status: BatchAccountTestStatus): string {
  if (status === 'success') return 'success'
  if (status === 'cancelled') return 'neutral'
  if (status === 'timeout') return 'warning'
  return 'failed'
}

function batchTestTargetName(accountID: number): string {
  return batchTestTargets.value.find(target => target.accountID === accountID)?.accountName || `账号 #${accountID}`
}

function batchTestTargetPlatform(accountID: number): string {
  return batchTestTargets.value.find(target => target.accountID === accountID)?.platform || 'unknown'
}

async function toggleBatchTestItemSchedulable(item: BatchAccountTestItem) {
  const accountID = item.account_id
  if (!Number.isInteger(accountID) || accountID <= 0 || togglingSchedulableID.value !== null) return

  const nextSchedulable = !batchTestItemSchedulable(item)
  togglingSchedulableID.value = accountID
  try {
    const updated = await adminAPI.accounts.setSchedulable(accountID, nextSchedulable)
    const schedulable = updated?.schedulable ?? nextSchedulable
    if (batchTestResult.value) {
      batchTestResult.value = {
        ...batchTestResult.value,
        results: batchTestResult.value.results.map(resultItem => resultItem.account_id === accountID
          ? { ...resultItem, schedulable }
          : resultItem),
      }
    }
    batchTestTargets.value = batchTestTargets.value.map(target => target.accountID === accountID
      ? { ...target, schedulable }
      : target)
    accountSourceItems.value = accountSourceItems.value.map(account => account.local_account_id === accountID
      ? { ...account, local_account_schedulable: schedulable }
      : account)
    applyAccountQuickFilterPage()
    if (selected.value?.local_account_id === accountID) {
      selected.value = { ...selected.value, local_account_schedulable: schedulable }
    }
    appStore.showSuccess(schedulable ? '账号调度已打开' : '账号调度已关闭')
  } catch (err) {
    appStore.showError(extractApiErrorMessage(err, '修改账号调度状态失败'))
  } finally {
    togglingSchedulableID.value = null
  }
}

function resetPageAndLoad() {
  page.value = 1
  void loadAccounts()
}

function handlePageChange(nextPage: number) {
  if (nextPage === page.value) return
  page.value = nextPage
  applyAccountQuickFilterPage()
}

function handlePageSizeChange(value: string | number | boolean | null) {
  if (value === null || typeof value === 'boolean') return
  const nextPageSize = Number(value)
  if (![20, 50, 100].includes(nextPageSize) || nextPageSize === pageSize.value) return
  pageSize.value = nextPageSize
  page.value = 1
  applyAccountQuickFilterPage()
}

function openDrawer(account: SupplierProviderAccount) {
  selected.value = account
}

function manageableLocalAccountID(account: SupplierProviderAccount): number | null {
  if (!isMatchedLocalAccount(account)) return null
  const localAccountID = Number(account.local_account_id)
  return Number.isInteger(localAccountID) && localAccountID > 0 ? localAccountID : null
}

function canManageLocalAccount(account: SupplierProviderAccount): boolean {
  return manageableLocalAccountID(account) !== null
}

function canDuplicateLocalAccount(account: SupplierProviderAccount): boolean {
  if (manageableLocalAccountID(account) === null) return false
  return ['apikey', 'upstream', 'bedrock', 'service_account'].includes(
    (account.local_account_type || '').trim().toLowerCase()
  )
}

const duplicateConfirmMessage = computed(() => {
  const account = duplicateConfirmAccount.value
  if (!account) return ''
  const accountName = account.local_account_name || account.name || account.upstream_account_key || `账号 #${account.local_account_id}`
  return `确认复制本地账号「${accountName}」？将创建暂停调度的本地副本，名称通常会追加 (Copy) 后缀。供应商上游账号页按上游账号匹配本地名，副本不会作为新的上游行出现在本页，可在账号管理中查看。`
})

const deleteSupplierAccountRecordMessage = computed(() => {
  const account = deleteSupplierAccountRecordTarget.value
  if (!account) return ''
  const accountName = account.name || account.upstream_account_key || `账号 #${account.id}`
  let reason = '当前上游账号记录'
  if (account.status === 'deleted') {
    reason = '上游账号已删除'
  } else if (account.group_status === 'missing') {
    reason = '该账号引用的上游分组已删除'
  } else if (account.group_status === 'inactive') {
    reason = '该账号引用的上游分组已失效'
  }
  return `确认删除本地保存的上游账号记录“${accountName}”？${reason}。本操作不会删除上游系统数据或本地账号，且删除后无法恢复。`
})

function requestDeleteSupplierAccountRecord(account: SupplierProviderAccount) {
  if (deletingSupplierAccountRecordID.value !== null) return

  deleteSupplierAccountRecordTarget.value = account
}

async function confirmDeleteSupplierAccountRecord() {
  const target = deleteSupplierAccountRecordTarget.value
  if (!target || deletingSupplierAccountRecordID.value !== null) return

  deletingSupplierAccountRecordID.value = target.id
  deleteSupplierAccountRecordTarget.value = null

  try {
    await deleteSupplierAccount(target.id)
    if (selected.value?.id === target.id) selected.value = null
    await loadAccounts()
    appStore.showSuccess('上游账号记录已删除')
  } catch (err) {
    appStore.showError(extractApiErrorMessage(err, '删除上游账号记录失败'))
  } finally {
    deletingSupplierAccountRecordID.value = null
  }
}

function requestDuplicateLocalAccount(account: SupplierProviderAccount) {
  if (!canDuplicateLocalAccount(account) || duplicatingAccountID.value !== null) return
  duplicateConfirmAccount.value = account
}

function closeDuplicateConfirm() {
  if (duplicatingAccountID.value !== null) return
  duplicateConfirmAccount.value = null
}

function closeDuplicateResult() {
  duplicateResultAccount.value = null
}

async function openDuplicatedAccountEditor() {
  const account = duplicateResultAccount.value
  if (!account) return
  duplicateResultAccount.value = null
  try {
    await loadAccountEditorOptions()
    editingAccount.value = account
    showEditAccountModal.value = true
  } catch (err) {
    appStore.showError(extractApiErrorMessage(err, '加载账号编辑信息失败'))
  }
}

function goToAccountManagement() {
  const account = duplicateResultAccount.value
  const keyword = account?.name?.trim() || ''
  duplicateResultAccount.value = null
  void router.push({
    path: '/admin/accounts',
    query: keyword ? { search: keyword } : undefined,
  })
}

async function confirmDuplicateLocalAccount() {
  const account = duplicateConfirmAccount.value
  const localAccountID = account ? manageableLocalAccountID(account) : null
  if (localAccountID === null || duplicatingAccountID.value !== null) return

  duplicatingAccountID.value = localAccountID
  try {
    const duplicate = await adminAPI.accounts.duplicate(localAccountID)
    duplicateConfirmAccount.value = null
    duplicateResultAccount.value = duplicate
    appStore.showSuccess(`账号已复制为「${duplicate.name}」，已暂停调度，请确认凭据后再启用`)
    await loadAccounts()
  } catch (err) {
    appStore.showError(extractApiErrorMessage(err, '复制账号失败'))
  } finally {
    duplicatingAccountID.value = null
  }
}

function openBusinessPlatformDialog(account: SupplierProviderAccount) {
  if (manageableLocalAccountID(account) === null || savingBusinessPlatform.value) return
  businessPlatformAccount.value = account
  businessPlatformDraft.value = normalizePlatform(account.platform_override)
}

function closeBusinessPlatformDialog() {
  if (savingBusinessPlatform.value) return
  businessPlatformAccount.value = null
  businessPlatformDraft.value = ''
}

async function saveBusinessPlatform() {
  const account = businessPlatformAccount.value
  const localAccountID = account ? manageableLocalAccountID(account) : null
  if (localAccountID === null || savingBusinessPlatform.value) return

  const platform = normalizePlatform(businessPlatformDraft.value)
  savingBusinessPlatform.value = true
  try {
    if (platform) {
      await setSupplierLocalAccountPlatformOverride(localAccountID, platform)
    } else {
      await clearSupplierLocalAccountPlatformOverride(localAccountID)
    }
    appStore.showSuccess(platform ? '业务平台已保存' : '业务平台已恢复为跟随接入平台')
    businessPlatformAccount.value = null
    businessPlatformDraft.value = ''
    await loadAccounts()
  } catch (err) {
    appStore.showError(extractApiErrorMessage(err, '保存业务平台失败'))
  } finally {
    savingBusinessPlatform.value = false
  }
}

async function openLocalAccountTest(account: SupplierProviderAccount) {
  const localAccountID = manageableLocalAccountID(account)
  if (localAccountID === null || testingAccountID.value !== null || accountActionLoadingID.value !== null) return

  testingAccountID.value = localAccountID
  try {
    testingAccount.value = await adminAPI.accounts.getById(localAccountID)
    showAccountTestModal.value = true
  } catch (err) {
    testingAccount.value = null
    appStore.showError(extractApiErrorMessage(err, '加载测试账号失败'))
  } finally {
    testingAccountID.value = null
  }
}

function closeAccountTestModal() {
  showAccountTestModal.value = false
  testingAccount.value = null
}

async function handleAccountTestResult(payload: {
  accountId: number
  status: 'testing' | 'success' | 'failed'
}) {
  if (payload.status === 'success' || payload.status === 'failed') {
    await loadAccounts()
  }
}

async function loadAccountEditorOptions() {
  const [groups, proxies] = await Promise.all([
    adminAPI.groups.getAll(),
    adminAPI.proxies.getAll(),
  ])
  accountEditGroups.value = groups
  accountProxies.value = proxies
}

async function openCreateAccountDialog() {
  try {
    await loadAccountEditorOptions()
    showCreateAccountModal.value = true
  } catch (err) {
    appStore.showError(extractApiErrorMessage(err, '加载新增账号配置失败'))
  }
}

function closeCreateAccountDialog() {
  showCreateAccountModal.value = false
}

async function handleAccountCreated() {
  showCreateAccountModal.value = false
  await loadAccounts()
}

async function openLocalAccountEditor(account: SupplierProviderAccount) {
  const localAccountID = manageableLocalAccountID(account)
  if (localAccountID === null || accountActionLoadingID.value !== null) return
  accountActionLoadingID.value = localAccountID
  try {
    const [localAccount] = await Promise.all([
      adminAPI.accounts.getById(localAccountID),
      loadAccountEditorOptions(),
    ])
    editingAccount.value = localAccount
    showEditAccountModal.value = true
  } catch (err) {
    appStore.showError(extractApiErrorMessage(err, '加载账号编辑信息失败'))
  } finally {
    accountActionLoadingID.value = null
  }
}

function closeLocalAccountEditor() {
  showEditAccountModal.value = false
  editingAccount.value = null
}

function handleLocalAccountUpdated(updatedAccount: Account) {
  if (editingAccount.value?.id === updatedAccount.id) {
    editingAccount.value = updatedAccount
  }
  void loadAccounts()
}

async function openAccountBindingEditor(account: SupplierProviderAccount) {
  const localAccountID = manageableLocalAccountID(account)
  if (localAccountID === null || accountActionLoadingID.value !== null) return
  accountActionLoadingID.value = localAccountID
  try {
    const [localAccount, groups] = await Promise.all([
      adminAPI.accounts.getById(localAccountID),
      adminAPI.groups.getAll(),
    ])
    accountEditGroups.value = groups
    bindingAccount.value = localAccount
    selectedBindingGroupIDs.value = [
      ...(localAccount.group_ids || localAccount.groups?.map(group => group.id) || []),
    ]
    bindingPlatform.value = localAccount.platform
  } catch (err) {
    appStore.showError(extractApiErrorMessage(err, '加载账号绑定信息失败'))
  } finally {
    accountActionLoadingID.value = null
  }
}

function closeAccountBindingEditor() {
  if (savingBindingAccountID.value !== null) return
  bindingAccount.value = null
  selectedBindingGroupIDs.value = []
  bindingPlatform.value = undefined
}

async function saveAccountBinding() {
  const account = bindingAccount.value
  if (!account || savingBindingAccountID.value !== null) return
  const bindingGroupIDs = [...selectedBindingGroupIDs.value]
  savingBindingAccountID.value = account.id
  try {
    const updated = await adminAPI.accounts.update(account.id, { group_ids: bindingGroupIDs })
    bindingAccount.value = updated
    selectedBindingGroupIDs.value = [...(updated.group_ids || bindingGroupIDs)]
    appStore.showSuccess('账号绑定已保存')
    bindingAccount.value = null
    selectedBindingGroupIDs.value = []
    bindingPlatform.value = undefined
    await loadAccounts()
  } catch (err) {
    appStore.showError(extractApiErrorMessage(err, '保存账号绑定失败'))
  } finally {
    savingBindingAccountID.value = null
  }
}

async function deleteLocalAccount(account: SupplierProviderAccount) {
  const localAccountID = manageableLocalAccountID(account)
  if (localAccountID === null || deletingAccountID.value !== null) return
  const accountName = account.local_account_name || account.name || account.upstream_account_key
  if (!window.confirm('确认删除本地账号「' + accountName + '」？')) return
  deletingAccountID.value = localAccountID
  try {
    await adminAPI.accounts.delete(localAccountID)
    if (selected.value?.local_account_id === localAccountID) selected.value = null
    appStore.showSuccess('账号已删除')
    await loadAccounts()
  } catch (err) {
    appStore.showError(extractApiErrorMessage(err, '删除账号失败'))
  } finally {
    deletingAccountID.value = null
  }
}

function applyFilterControlLabels() {
  setFilterControlLabel(searchFilterControl.value, 'input', 'supplier-account-search-label')
  setFilterControlLabel(providerFilterControl.value, '.select-trigger', 'supplier-account-provider-label')
  setFilterControlLabel(groupFilterControl.value, '.select-trigger', 'supplier-account-group-label')
  setFilterControlLabel(platformFilterControl.value, '.select-trigger', 'supplier-account-platform-label')
  setFilterControlLabel(activeFilterControl.value, '.select-trigger', 'supplier-account-active-label')
  setFilterControlLabel(upstreamStatusFilterControl.value, '.select-trigger', 'supplier-account-upstream-status-label')
}

function setFilterControlLabel(container: HTMLElement | null, selector: string, labelID: string) {
  container?.querySelector<HTMLElement>(selector)?.setAttribute('aria-labelledby', labelID)
}

function isMatchedLocalAccount(account: SupplierProviderAccount): boolean {
  return account.local_account_match_status === 'matched'
}

function supplierTone(providerID: number) {
  const providerIndex = supplierIDs.value.indexOf(providerID)
  const toneIndex = providerIndex >= 0 ? providerIndex : Math.abs(Math.trunc(providerID || 0))
  return SUPPLIER_TONES[toneIndex % SUPPLIER_TONES.length]
}

function upstreamStatusLabel(status?: string): string {
  if (!status) return '—'
  const s = String(status).trim().toLowerCase()
  if (s === 'active' || s === '1') return '正常'
  if (s === 'disabled' || s === 'inactive' || s === '2') return '停用'
  if (s === 'expired' || s === '3') return '已过期'
  if (s === 'quota_exhausted' || s === '4') return '额度耗尽'
  if (s === 'unknown') return '未知'
  if (s === 'deleted') return '已删除'
  return displayValue(status)
}

function upstreamStatusTone(status?: string): string {
  const s = String(status ?? '').trim().toLowerCase()
  if (s === 'active' || s === '1') return 'good'
  if (s === 'disabled' || s === 'inactive' || s === '2') return 'neutral'
  if (s === 'expired' || s === '3' || s === 'quota_exhausted' || s === '4') return 'bad'
  if (s === 'deleted') return 'bad'
  return 'neutral'
}

function localAccountStatusLabel(status?: string): string {
  if (status === 'active') return '正常'
  if (status === 'inactive' || status === 'disabled') return '停用'
  if (status === 'error') return '异常'
  return displayValue(status)
}

function localAccountStatusTone(status?: string): string {
  if (status === 'active') return 'good'
  if (status === 'error') return 'bad'
  return 'neutral'
}

function canEditPriority(account: SupplierProviderAccount): boolean {
  return isMatchedLocalAccount(account)
    && Number.isInteger(account.local_account_id)
    && Number(account.local_account_id) > 0
    && typeof account.local_account_priority === 'number'
}

function startPriorityEdit(account: SupplierProviderAccount) {
  if (!canEditPriority(account)) return
  const localAccountID = Number(account.local_account_id)
  if (savingPriorityAccountID.value === localAccountID) return

  editingPriorityAccountID.value = localAccountID
  priorityDraft.value = String(account.local_account_priority)
  void nextTick(() => {
    priorityInput.value?.focus()
    priorityInput.value?.select()
  })
}

function cancelPriorityEdit() {
  editingPriorityAccountID.value = null
  priorityDraft.value = ''
}

async function savePriority(account: SupplierProviderAccount) {
  if (!canEditPriority(account)) return
  const localAccountID = Number(account.local_account_id)
  if (editingPriorityAccountID.value !== localAccountID) return
  if (savingPriorityAccountID.value === localAccountID) return

  const draft = priorityDraft.value.trim()
  const nextPriority = Number(draft)
  if (draft === '' || !Number.isInteger(nextPriority) || nextPriority < 0) {
    appStore.showError('请输入有效的整数优先级')
    return
  }
  if (nextPriority === account.local_account_priority) {
    cancelPriorityEdit()
    return
  }

  savingPriorityAccountID.value = localAccountID
  try {
    const updated = await adminAPI.accounts.update(localAccountID, { priority: nextPriority })
    const priority = typeof updated?.priority === 'number' ? updated.priority : nextPriority
    accountSourceItems.value = accountSourceItems.value.map(item => item.local_account_id === localAccountID
      ? { ...item, local_account_priority: priority }
      : item)
    applyAccountQuickFilterPage()
    if (selected.value?.local_account_id === localAccountID) {
      selected.value = { ...selected.value, local_account_priority: priority }
    }
    cancelPriorityEdit()
    appStore.showSuccess('账号优先级已保存')
  } catch (err) {
    appStore.showError(extractApiErrorMessage(err, '修改账号优先级失败'))
  } finally {
    savingPriorityAccountID.value = null
  }
}

function canToggleSchedulable(account: SupplierProviderAccount): boolean {
  return isMatchedLocalAccount(account)
    && Number.isInteger(account.local_account_id)
    && Number(account.local_account_id) > 0
    && typeof account.local_account_schedulable === 'boolean'
}

function schedulableToggleTitle(account: SupplierProviderAccount): string {
  return account.local_account_schedulable
    ? '当前参与调度，点击停用'
    : '当前不参与调度，点击启用'
}

async function handleToggleSchedulable(account: SupplierProviderAccount) {
  if (!canToggleSchedulable(account)) return
  const localAccountID = Number(account.local_account_id)
  if (togglingSchedulableID.value === localAccountID) return

  const nextSchedulable = !account.local_account_schedulable
  togglingSchedulableID.value = localAccountID
  try {
    const updated = await adminAPI.accounts.setSchedulable(localAccountID, nextSchedulable)
    const schedulable = updated?.schedulable ?? nextSchedulable
    accountSourceItems.value = accountSourceItems.value.map(item => item.local_account_id === localAccountID
      ? { ...item, local_account_schedulable: schedulable }
      : item)
    applyAccountQuickFilterPage()
    if (selected.value?.local_account_id === localAccountID) {
      selected.value = { ...selected.value, local_account_schedulable: schedulable }
    }
  } catch (err) {
    appStore.showError(extractApiErrorMessage(err, '修改账号调度状态失败'))
  } finally {
    togglingSchedulableID.value = null
  }
}

function accountTestStatusLabel(status?: string): string {
  if (status === 'testing') return '测试中'
  if (status === 'success') return '成功'
  if (status === 'failed') return '失败'
  return '—'
}

function accountTestStatusTone(status?: string): string {
  if (status === 'testing') return 'testing'
  if (status === 'success') return 'success'
  if (status === 'failed') return 'failed'
  return 'neutral'
}

function isFailedTest(account: SupplierProviderAccount): boolean {
  return isMatchedLocalAccount(account) && account.local_account_last_test_status === 'failed'
}

function openTestErrorDialog(account: SupplierProviderAccount) {
  if (!isFailedTest(account)) return
  testErrorAccount.value = account
}

function displayValue(value?: string | number | null): string {
  if (value === null || value === undefined || value === '') return '—'
  return String(value)
}

function localAccountMatchLabel(account: SupplierProviderAccount): string {
  if (account.local_account_match_status === 'unmatched') return '未匹配'
  if (account.local_account_match_status === 'conflict') {
    return `匹配冲突（${account.local_account_match_count}）`
  }
  if (account.local_account_match_status === 'matched') return '已匹配'
  return '—'
}

function localAccountDisplayName(account: SupplierProviderAccount): string {
  if (!isMatchedLocalAccount(account)) return localAccountMatchLabel(account)
  return displayValue(account.local_account_name)
}

function localDetailValue(
  account: SupplierProviderAccount,
  value?: string | number | null
): string {
  return isMatchedLocalAccount(account) ? displayValue(value) : '—'
}

function localSchedulableLabel(account: SupplierProviderAccount): string {
  if (!isMatchedLocalAccount(account)) return '—'
  if (account.local_account_schedulable === true) return '是'
  if (account.local_account_schedulable === false) return '否'
  return '—'
}

function accountInitial(account: SupplierProviderAccount): string {
  const value = account.name?.trim() || account.upstream_account_key?.trim() || '?'
  return value.slice(0, 1).toUpperCase()
}

function formatRate(value?: number | null): string {
  if (value === null || value === undefined || !Number.isFinite(Number(value))) return '—'
  return `× ${String(value)}`
}

function openAccountRateGuardLogs() {
  accountRateGuardLogsVisible.value = true
}

function closeAccountRateGuardLogs() {
  accountRateGuardLogsVisible.value = false
}

async function loadAccountRateGuardPendingCount() {
  try {
    const result = await listAccountRateGuardUnbindLogs({
      result: 'unbound',
      status: 'pending',
      page: 1,
      page_size: 1,
    })
    accountRateGuardPendingCount.value = Number(result.pending_count) || 0
  } catch {
    // 角标加载失败不阻断页面主流程，保留上一次数量。
  }
}

async function updateAccountRateGuardPendingCount() {
  await loadAccountRateGuardPendingCount()
}

async function refreshAccountsWorkbench() {
  await Promise.all([loadAccounts(), loadAccountRateGuardPendingCount()])
}

function handleAccountSort(key: string, order: 'asc' | 'desc') {
  sortBy.value = key
  sortOrder.value = order
  page.value = 1
  void loadAccounts()
}

function formatCNY(value?: number | null): string {
  if (value === null || value === undefined || !Number.isFinite(Number(value))) return '—'
  return cnyFormatter.format(Number(value))
}

function formatTime(value?: string): string {
  if (!value) return '—'
  const date = new Date(value)
  if (Number.isNaN(date.getTime())) return '—'
  return date.toLocaleString('zh-CN', {
    year: 'numeric',
    month: '2-digit',
    day: '2-digit',
    hour: '2-digit',
    minute: '2-digit',
  })
}
</script>
<style scoped>
.sp-account-toolbar {
  margin-bottom: 1rem;
  overflow: hidden;
  border: 1px solid color-mix(in srgb, var(--sp-cyan) 18%, var(--sp-line));
  border-radius: 0.875rem;
  background: var(--sp-panel);
  box-shadow: var(--sp-shadow);
}

.sp-filter-card-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 1rem;
  padding: 0.875rem 1rem 0.75rem;
  border-bottom: 1px solid var(--sp-line);
  background: linear-gradient(90deg, color-mix(in srgb, var(--sp-cyan) 6%, transparent), transparent 38%);
}

.sp-filter-card-head > div {
  min-width: 0;
}

.sp-filter-card-kicker {
  display: block;
  margin-bottom: 0.2rem;
  color: var(--sp-cyan);
  font-size: 0.625rem;
  font-weight: 800;
  letter-spacing: 0.11em;
}

.sp-filter-card-head h2 {
  margin: 0;
  color: var(--sp-text);
  font-size: 0.9375rem;
  font-weight: 800;
  line-height: 1.35;
}

.sp-filter-card-head p {
  margin: 0.2rem 0 0;
  color: var(--sp-muted);
  font-size: 0.75rem;
  line-height: 1.45;
}

.sp-filter-card-count {
  flex: 0 0 auto;
  padding: 0.35rem 0.6rem;
  border: 1px solid color-mix(in srgb, var(--sp-cyan) 20%, var(--sp-line));
  border-radius: 999px;
  background: color-mix(in srgb, var(--sp-cyan) 6%, var(--sp-panel));
  color: var(--sp-cyan);
  font-size: 0.6875rem;
  font-weight: 700;
}

.sp-account-filter-body {
  display: flex;
  align-items: flex-end;
  gap: 0.875rem;
  padding: 0.875rem 1rem 1rem;
}

.sp-account-filter-fields {
  display: grid;
  min-width: 0;
  flex: 1 1 auto;
  grid-template-columns: minmax(14rem, 1fr) minmax(8.5rem, 0.32fr) minmax(9rem, 0.34fr) minmax(7.5rem, 0.26fr) minmax(7.5rem, 0.26fr) minmax(8rem, 0.28fr);
  gap: 0.625rem;
}

.sp-account-filter-control {
  display: flex;
  min-width: 0;
  flex-direction: column;
  gap: 0.3rem;
}


.sp-account-filter-actions {
  display: flex;
  flex: 0 0 auto;
  flex-wrap: wrap;
  align-items: center;
  align-self: center;
  justify-content: flex-end;
  gap: 0.75rem;
  padding-left: 1rem;
  border-left: 1px solid var(--sp-line);
}

.sp-account-toolbar-btn {
  min-height: 2.625rem;
  padding-inline: 0.95rem;
  font-weight: 650;
  white-space: nowrap;
}

.sp-account-toolbar-create {
  border-color: color-mix(in srgb, var(--sp-green) 55%, var(--sp-line));
  background: var(--sp-green);
  color: #fff;
  box-shadow: 0 8px 18px color-mix(in srgb, var(--sp-green) 22%, transparent);
}

.sp-account-toolbar-create:hover:not(:disabled) {
  border-color: color-mix(in srgb, var(--sp-green) 70%, #14532d);
  background: color-mix(in srgb, var(--sp-green) 88%, #14532d);
  color: #fff;
}

.sp-account-toolbar-test {
  border-color: color-mix(in srgb, var(--sp-blue) 42%, var(--sp-line));
  background: color-mix(in srgb, var(--sp-blue) 10%, var(--sp-panel));
  color: var(--sp-blue);
}

.sp-account-toolbar-test:hover:not(:disabled) {
  border-color: color-mix(in srgb, var(--sp-blue) 58%, var(--sp-line));
  background: color-mix(in srgb, var(--sp-blue) 16%, var(--sp-panel));
  color: color-mix(in srgb, var(--sp-blue) 85%, #0f172a);
}

.sp-account-toolbar-logs {
  display: inline-flex;
  align-items: center;
  gap: 0.375rem;
  border-color: color-mix(in srgb, var(--sp-amber) 45%, var(--sp-line));
  background: color-mix(in srgb, var(--sp-amber) 12%, var(--sp-panel));
  color: var(--sp-amber);
}

.sp-account-toolbar-logs.has-pending {
  border-color: color-mix(in srgb, var(--sp-amber) 68%, var(--sp-line));
  background: color-mix(in srgb, var(--sp-amber) 18%, var(--sp-panel));
  font-weight: 700;
}

.sp-account-rate-guard-pending-count {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  min-width: 1.25rem;
  height: 1.25rem;
  padding: 0 0.375rem;
  border: 1px solid color-mix(in srgb, var(--sp-amber) 36%, var(--sp-line));
  border-radius: 999px;
  background: color-mix(in srgb, var(--sp-amber) 14%, var(--sp-panel));
  color: var(--sp-amber);
  font-size: 0.6875rem;
  font-variant-numeric: tabular-nums;
  font-weight: 800;
  line-height: 1;
}

.sp-account-toolbar-logs:hover:not(:disabled) {
  border-color: color-mix(in srgb, var(--sp-amber) 62%, var(--sp-line));
  background: color-mix(in srgb, var(--sp-amber) 18%, var(--sp-panel));
  color: color-mix(in srgb, var(--sp-amber) 88%, #7c2d12);
}

.sp-account-toolbar-refresh,
.sp-account-refresh {
  min-width: 5.25rem;
  min-height: 2.625rem;
  border-color: color-mix(in srgb, var(--sp-violet) 28%, var(--sp-line));
  background: color-mix(in srgb, var(--sp-violet) 8%, var(--sp-panel));
  color: var(--sp-violet);
}

.sp-account-toolbar-refresh:hover:not(:disabled),
.sp-account-refresh:hover:not(:disabled) {
  border-color: color-mix(in srgb, var(--sp-violet) 48%, var(--sp-line));
  background: color-mix(in srgb, var(--sp-violet) 14%, var(--sp-panel));
  color: color-mix(in srgb, var(--sp-violet) 90%, #1e1b4b);
}

.sp-account-workbench {
  border-color: color-mix(in srgb, var(--sp-cyan) 14%, var(--sp-line));
}

.sp-account-panel-head {
  display: grid;
  grid-template-columns: minmax(13rem, auto) minmax(28rem, 1fr) auto;
  align-items: center;
  gap: 1rem;
  min-height: 4.5rem;
  background:
    linear-gradient(90deg, color-mix(in srgb, var(--sp-cyan) 5%, transparent), transparent 34%),
    var(--sp-panel);
}

.sp-account-quick-filters {
  display: flex;
  min-width: 0;
  align-items: center;
  justify-content: center;
  gap: 0.4rem;
}

.sp-account-quick-filter {
  display: inline-flex;
  min-height: 2rem;
  align-items: center;
  justify-content: center;
  gap: 0.38rem;
  border: 1px solid var(--sp-line);
  border-radius: 999px;
  padding: 0.3rem 0.62rem;
  background: color-mix(in srgb, var(--sp-panel-2) 88%, transparent);
  color: var(--sp-muted);
  font-size: 0.72rem;
  font-weight: 750;
  white-space: nowrap;
  transition: border-color 150ms ease, background-color 150ms ease, color 150ms ease, transform 150ms ease;
}

.sp-account-quick-filter:hover {
  border-color: color-mix(in srgb, var(--sp-cyan) 42%, var(--sp-line));
  color: var(--sp-text);
  transform: translateY(-1px);
}

.sp-account-quick-filter strong {
  min-width: 1.25rem;
  border-radius: 999px;
  padding: 0.05rem 0.28rem;
  background: color-mix(in srgb, var(--sp-muted) 12%, transparent);
  color: var(--sp-text);
  font-family: ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, monospace;
  font-size: 0.68rem;
  line-height: 1.1rem;
  text-align: center;
}

.sp-account-quick-filter.active {
  border-color: color-mix(in srgb, var(--sp-cyan) 55%, var(--sp-line));
  background: color-mix(in srgb, var(--sp-cyan) 11%, var(--sp-panel));
  color: var(--sp-cyan);
  box-shadow: 0 0 0 2px color-mix(in srgb, var(--sp-cyan) 8%, transparent);
}

.sp-account-quick-filter:nth-child(2).active,
.sp-account-quick-filter:nth-child(4).active {
  border-color: color-mix(in srgb, var(--sp-green) 52%, var(--sp-line));
  background: color-mix(in srgb, var(--sp-green) 9%, var(--sp-panel));
  color: var(--sp-green);
  box-shadow: 0 0 0 2px color-mix(in srgb, var(--sp-green) 7%, transparent);
}

.sp-account-quick-filter:nth-child(5).active {
  border-color: color-mix(in srgb, var(--sp-amber) 55%, var(--sp-line));
  background: color-mix(in srgb, var(--sp-amber) 9%, var(--sp-panel));
  color: var(--sp-amber);
  box-shadow: 0 0 0 2px color-mix(in srgb, var(--sp-amber) 7%, transparent);
}

.sp-account-quick-filter:nth-child(6).active {
  border-color: color-mix(in srgb, var(--sp-red) 52%, var(--sp-line));
  background: color-mix(in srgb, var(--sp-red) 8%, var(--sp-panel));
  color: var(--sp-red);
  box-shadow: 0 0 0 2px color-mix(in srgb, var(--sp-red) 6%, transparent);
}

.sp-account-quick-filter:nth-child(2) strong,
.sp-account-quick-filter:nth-child(4) strong {
  background: color-mix(in srgb, var(--sp-green) 13%, transparent);
  color: var(--sp-green);
}

.sp-account-quick-filter:nth-child(5) strong {
  background: color-mix(in srgb, var(--sp-amber) 14%, transparent);
  color: var(--sp-amber);
}

.sp-account-quick-filter:nth-child(6) strong {
  background: color-mix(in srgb, var(--sp-red) 12%, transparent);
  color: var(--sp-red);
}

.sp-account-legend {
  display: flex;
  align-items: center;
  gap: 1rem;
  color: var(--sp-muted);
  font-size: 0.75rem;
}

.sp-account-legend span {
  display: inline-flex;
  align-items: center;
  gap: 0.4rem;
}

.sp-account-legend i,
.sp-provider-dot {
  display: inline-block;
  width: 0.45rem;
  height: 0.45rem;
  flex: 0 0 auto;
  border-radius: 9999px;
}

.sp-account-legend i.matched {
  background: var(--sp-green);
  box-shadow: 0 0 0 3px color-mix(in srgb, var(--sp-green) 14%, transparent);
}

.sp-account-legend i.unmatched {
  background: var(--sp-muted);
  box-shadow: 0 0 0 3px color-mix(in srgb, var(--sp-muted) 12%, transparent);
}

.sp-account-legend i.conflict {
  background: var(--sp-amber);
  box-shadow: 0 0 0 3px color-mix(in srgb, var(--sp-amber) 14%, transparent);
}

.sp-account-table-shell {
  min-height: 23rem;
  overflow: hidden;
  background: var(--sp-panel);
}

.sp-account-table-shell :deep(.table-wrapper) {
  overflow-x: auto;
  border: 0;
  border-radius: 0;
  scrollbar-gutter: stable both-edges;
}

.sp-account-table-shell :deep(table) {
  min-width: 118rem;
}

.sp-account-table-shell :deep(.table-header) {
  background: var(--sp-panel-2);
}

.sp-account-table-shell :deep(th) {
  height: 3.25rem;
  border-bottom-color: var(--sp-line);
  color: var(--sp-muted);
  font-size: 0.6875rem;
  letter-spacing: 0.06em;
}

.sp-account-table-shell :deep(td) {
  height: 4.25rem;
  border-color: var(--sp-soft);
}

.sp-account-table-shell :deep(tbody tr) {
  transition: background-color 140ms ease, box-shadow 140ms ease;
}

.sp-account-table-shell :deep(tbody tr:hover) {
  background: color-mix(in srgb, var(--sp-cyan) 5%, var(--sp-panel));
  box-shadow: inset 3px 0 0 color-mix(in srgb, var(--sp-cyan) 72%, transparent);
}

.sp-account-identity,
.sp-provider-cell {
  display: flex;
  min-width: 0;
  align-items: center;
  gap: 0.625rem;
}

.sp-provider-cell {
  width: fit-content;
  border-width: 1px;
  border-radius: 0.55rem;
  padding: 0.35rem 0.5rem;
}

.sp-provider-cell .sp-entity {
  color: inherit;
}

.sp-account-avatar {
  display: inline-flex;
  width: 2rem;
  height: 2rem;
  flex: 0 0 auto;
  align-items: center;
  justify-content: center;
  border: 1px solid color-mix(in srgb, var(--sp-cyan) 24%, var(--sp-line));
  border-radius: 0.5rem;
  background: color-mix(in srgb, var(--sp-cyan) 8%, var(--sp-panel-2));
  color: var(--sp-cyan);
  font-size: 0.75rem;
  font-weight: 800;
}

.sp-account-copy {
  min-width: 0;
}

.sp-account-meta {
  display: flex;
  min-width: 0;
  align-items: center;
  gap: 0.375rem;
  margin-top: 0.2rem;
}

.sp-platform-badge {
  display: inline-flex;
  flex: 0 0 auto;
  align-items: center;
  border-width: 1px;
  border-radius: 0.3rem;
  padding: 0.08rem 0.35rem;
  font-size: 0.625rem;
  font-weight: 700;
  line-height: 1rem;
}

.sp-account-key {
  max-width: 14rem;
  overflow: hidden;
  font-family: ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, monospace;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.sp-provider-dot {
  box-shadow: 0 0 0 4px color-mix(in srgb, currentColor 10%, transparent);
}

.sp-account-code,
.sp-local-status,
.sp-test-status,
.sp-match-badge {
  display: inline-flex;
  align-items: center;
  border: 1px solid var(--sp-soft);
  border-radius: 0.4rem;
  padding: 0.22rem 0.5rem;
  background: var(--sp-panel-2);
  color: var(--sp-muted);
  font-size: 0.75rem;
  line-height: 1.2;
}

.sp-account-groups {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 0.375rem;
}

.sp-account-group-stack {
  display: flex;
  flex-direction: column;
  gap: 0.35rem;
  min-width: 0;
}

.sp-upstream-group-deleted {
  display: inline-flex;
  align-items: center;
  width: fit-content;
  max-width: 100%;
  color: var(--sp-red);
  font-size: 0.72rem;
  font-weight: 700;
  line-height: 1.2;
  padding: 0.18rem 0.5rem;
  border: 1px solid color-mix(in srgb, var(--sp-red) 28%, var(--sp-line));
  border-radius: 999px;
  background: color-mix(in srgb, var(--sp-red) 8%, var(--sp-panel));
}

.sp-account-code,
.sp-account-number,
.sp-account-rate,
.sp-money-cell strong {
  font-family: ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, monospace;
}

.sp-account-code {
  text-transform: lowercase;
}

.sp-match-badge.matched {
  border-color: color-mix(in srgb, var(--sp-green) 28%, var(--sp-line));
  background: color-mix(in srgb, var(--sp-green) 8%, var(--sp-panel));
  color: var(--sp-green);
}

.sp-match-badge.unmatched {
  color: var(--sp-muted);
}

.sp-match-badge.conflict {
  border-color: color-mix(in srgb, var(--sp-amber) 32%, var(--sp-line));
  background: color-mix(in srgb, var(--sp-amber) 8%, var(--sp-panel));
  color: var(--sp-amber);
}

.sp-local-account-cell,
.sp-money-cell {
  display: flex;
  flex-direction: column;
  align-items: flex-start;
  gap: 0.25rem;
}

.sp-local-account-cell strong {
  max-width: 11rem;
  overflow: hidden;
  color: var(--sp-text);
  font-size: 0.8125rem;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.sp-account-number,
.sp-account-rate {
  font-size: 0.8125rem;
  font-weight: 700;
}

.sp-account-number {
  color: var(--sp-text);
}

.sp-priority-cell {
  display: inline-flex;
  min-width: 3.75rem;
  align-items: center;
}

.sp-priority-trigger {
  min-width: 2.75rem;
  border: 1px solid transparent;
  border-radius: 0.4rem;
  padding: 0.25rem 0.45rem;
  background: transparent;
  cursor: pointer;
  text-align: center;
  transition: border-color 140ms ease, background-color 140ms ease, color 140ms ease;
}

.sp-priority-trigger:hover,
.sp-priority-trigger:focus-visible {
  border-color: color-mix(in srgb, var(--sp-green) 32%, var(--sp-line));
  background: color-mix(in srgb, var(--sp-green) 8%, var(--sp-panel));
  color: var(--sp-green);
  outline: none;
}

.sp-priority-trigger:disabled {
  cursor: wait;
  opacity: 0.55;
}

.sp-priority-input {
  width: 4.5rem;
}

.sp-priority-input :deep(.input) {
  min-height: 2rem;
  padding: 0.3rem 0.45rem;
  font-family: ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, monospace;
  font-size: 0.8125rem;
  font-weight: 700;
  text-align: center;
}

.sp-local-status.good {
  border-color: color-mix(in srgb, var(--sp-green) 28%, var(--sp-line));
  background: color-mix(in srgb, var(--sp-green) 8%, var(--sp-panel));
  color: var(--sp-green);
}

.sp-local-status.bad {
  border-color: color-mix(in srgb, var(--sp-red) 24%, var(--sp-line));
  background: color-mix(in srgb, var(--sp-red) 8%, var(--sp-panel));
  color: var(--sp-red);
}

.sp-local-status.neutral,
.sp-test-status.neutral {
  color: var(--sp-muted);
}

.sp-test-status {
  font-weight: 700;
}

button.sp-test-status.failed {
  cursor: pointer;
  transition: filter 140ms ease, transform 140ms ease;
}

button.sp-test-status.failed:hover {
  filter: brightness(0.96);
  transform: translateY(-1px);
}

.sp-test-status.testing {
  border-color: color-mix(in srgb, #2563eb 28%, var(--sp-line));
  background: color-mix(in srgb, #2563eb 8%, var(--sp-panel));
  color: #2563eb;
}

.sp-test-status.success {
  border-color: color-mix(in srgb, var(--sp-green) 28%, var(--sp-line));
  background: color-mix(in srgb, var(--sp-green) 8%, var(--sp-panel));
  color: var(--sp-green);
}

.sp-test-status.failed {
  border-color: color-mix(in srgb, var(--sp-red) 28%, var(--sp-line));
  background: color-mix(in srgb, var(--sp-red) 8%, var(--sp-panel));
  color: var(--sp-red);
}

.sp-test-error-dialog {
  display: grid;
  gap: 0.875rem;
}

.sp-test-error-meta {
  display: grid;
  grid-template-columns: 6.5rem 1fr;
  gap: 0.75rem;
  align-items: baseline;
  color: var(--sp-muted);
  font-size: 0.8125rem;
}

.sp-test-error-meta strong {
  color: var(--sp-text);
  font-weight: 700;
}

.sp-test-error-message {
  max-height: 20rem;
  overflow: auto;
  border: 1px solid color-mix(in srgb, var(--sp-red) 24%, var(--sp-line));
  border-radius: 0.625rem;
  padding: 0.875rem;
  background: color-mix(in srgb, var(--sp-red) 6%, var(--sp-panel-2));
  color: var(--sp-text);
  font-family: ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, monospace;
  font-size: 0.75rem;
  line-height: 1.65;
  white-space: pre-wrap;
  word-break: break-word;
}

.sp-account-time,
.sp-account-muted {
  color: var(--sp-muted);
  font-size: 0.8125rem;
}

.sp-money-cell strong {
  color: var(--sp-green);
  font-size: 0.8125rem;
}

.sp-money-cell.cost strong {
  color: var(--sp-amber);
}

.sp-money-cell small {
  color: var(--sp-muted);
  font-size: 0.625rem;
  letter-spacing: 0.04em;
}

.sp-account-view-button {
  min-width: 3.5rem;
}
.sp-account-row-actions {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 0.375rem;
}

.sp-account-row-actions .sp-button {
  flex: 0 0 auto;
}

.sp-account-row-actions .sp-account-action-view {
  border-color: var(--sp-line);
  color: var(--sp-muted);
}

.sp-account-row-actions .sp-account-action-test {
  border-color: color-mix(in srgb, var(--sp-blue) 42%, var(--sp-line));
  background: color-mix(in srgb, var(--sp-blue) 9%, var(--sp-panel));
  color: var(--sp-blue);
}

.sp-account-row-actions .sp-account-action-edit {
  border-color: color-mix(in srgb, var(--sp-amber) 42%, var(--sp-line));
  background: color-mix(in srgb, var(--sp-amber) 9%, var(--sp-panel));
  color: var(--sp-amber);
}

.sp-account-row-actions .sp-account-action-copy {
  border-color: color-mix(in srgb, var(--sp-violet) 42%, var(--sp-line));
  background: color-mix(in srgb, var(--sp-violet) 9%, var(--sp-panel));
  color: var(--sp-violet);
}

.sp-account-row-actions .sp-account-action-platform {
  border-color: color-mix(in srgb, #4f46e5 42%, var(--sp-line));
  background: color-mix(in srgb, #4f46e5 9%, var(--sp-panel));
  color: #4f46e5;
}

.sp-account-row-actions .sp-account-action-binding {
  border-color: color-mix(in srgb, var(--sp-green) 42%, var(--sp-line));
  background: color-mix(in srgb, var(--sp-green) 9%, var(--sp-panel));
  color: var(--sp-green);
}

.sp-account-row-actions .sp-account-action-delete {
  border-color: color-mix(in srgb, var(--sp-red) 42%, var(--sp-line));
  background: color-mix(in srgb, var(--sp-red) 9%, var(--sp-panel));
  color: var(--sp-red);
}

.sp-account-row-actions .sp-account-action-test:hover,
.sp-account-row-actions .sp-account-action-edit:hover,
.sp-account-row-actions .sp-account-action-copy:hover,
.sp-account-row-actions .sp-account-action-platform:hover,
.sp-account-row-actions .sp-account-action-binding:hover,
.sp-account-row-actions .sp-account-action-delete:hover {
  background: color-mix(in srgb, currentColor 14%, var(--sp-panel));
}

:global(.modal-content:has(.sp-business-platform-dialog)) {
  --sp-panel: #ffffff;
  --sp-panel-2: #f5f8fc;
  --sp-panel-3: #eef3f9;
  --sp-line: #d7e2ef;
  --sp-soft: #e8eef6;
  --sp-text: #152238;
  --sp-muted: #5f7088;
  --sp-dim: #8a99ad;
  --sp-cyan: #2563eb;
  --sp-green: #16835d;
  --sp-amber: #c56a0a;
  --sp-orange: #ea580c;
  --sp-red: #d14343;
  --sp-blue: #1d4ed8;
  --sp-violet: #7c3aed;
  --sp-shadow: 0 1px 2px rgba(15, 23, 42, 0.04);
  border-color: #c9d7e8;
  background:
    radial-gradient(circle at 12% 0%, color-mix(in srgb, #2563eb 10%, transparent), transparent 36%),
    radial-gradient(circle at 96% 8%, color-mix(in srgb, #0ea5e9 8%, transparent), transparent 30%),
    #ffffff;
  color: var(--sp-text);
}

:global(.dark .modal-content:has(.sp-business-platform-dialog)) {
  --sp-panel: #172033;
  --sp-panel-2: #1d293d;
  --sp-panel-3: #243249;
  --sp-line: #35445c;
  --sp-soft: #2c3a51;
  --sp-text: #edf3fb;
  --sp-muted: #a8b6ca;
  --sp-dim: #75849a;
  --sp-cyan: #3b82f6;
  --sp-blue: #60a5fa;
  --sp-shadow: 0 1px 2px rgba(0, 0, 0, 0.2);
  border-color: #3b4b64;
  background:
    radial-gradient(circle at 10% 0%, color-mix(in srgb, #3b82f6 16%, transparent), transparent 34%),
    radial-gradient(circle at 94% 10%, color-mix(in srgb, #0ea5e9 12%, transparent), transparent 30%),
    #172033;
  color: var(--sp-text);
}

:global(.modal-content:has(.sp-business-platform-dialog) .modal-footer) {
  gap: 0.75rem;
  border-top-color: color-mix(in srgb, #2563eb 16%, var(--sp-line));
  background: linear-gradient(
    90deg,
    color-mix(in srgb, #2563eb 7%, var(--sp-panel)),
    color-mix(in srgb, #0ea5e9 5%, var(--sp-panel))
  );
}

:global(.dark .modal-content:has(.sp-business-platform-dialog) .modal-footer) {
  border-top-color: color-mix(in srgb, #60a5fa 18%, var(--sp-line));
  background: linear-gradient(
    90deg,
    color-mix(in srgb, #3b82f6 12%, var(--sp-panel)),
    color-mix(in srgb, #0ea5e9 8%, var(--sp-panel))
  );
}

.sp-business-platform-dialog {
  display: grid;
  gap: 1rem;
  padding: 0.15rem 0.05rem 0.25rem;
}

.sp-business-platform-summary {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 0.75rem;
  margin: 0;
}

.sp-business-platform-summary > div,
.sp-business-platform-field {
  display: grid;
  gap: 0.35rem;
}

.sp-business-platform-summary > div {
  min-width: 0;
  border: 1px solid color-mix(in srgb, #2563eb 14%, var(--sp-line));
  border-radius: 0.75rem;
  padding: 0.8rem 0.9rem;
  background:
    linear-gradient(180deg, color-mix(in srgb, #2563eb 6%, var(--sp-panel)), var(--sp-panel-2));
  box-shadow: inset 0 1px 0 rgba(255, 255, 255, 0.55);
}

.sp-business-platform-summary dt,
.sp-business-platform-field > span {
  color: var(--sp-muted);
  font-size: 0.75rem;
  font-weight: 600;
  letter-spacing: 0.01em;
}

.sp-business-platform-summary dd {
  overflow: hidden;
  margin: 0;
  color: var(--sp-text);
  font-size: 0.92rem;
  font-weight: 700;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.sp-business-platform-hint {
  margin: 0;
  border: 1px solid color-mix(in srgb, #0ea5e9 16%, var(--sp-line));
  border-radius: 0.7rem;
  padding: 0.7rem 0.85rem;
  background: color-mix(in srgb, #0ea5e9 7%, var(--sp-panel));
  color: var(--sp-muted);
  font-size: 0.8rem;
  line-height: 1.55;
}

.sp-business-platform-cancel,
.sp-business-platform-save {
  display: inline-flex;
  min-height: 2.5rem;
  min-width: 5.5rem;
  align-items: center;
  justify-content: center;
  border-radius: 0.55rem;
  padding: 0.5rem 0.95rem;
  font-size: 0.875rem;
  font-weight: 600;
  cursor: pointer;
  transition: background-color 0.15s ease, border-color 0.15s ease, color 0.15s ease, box-shadow 0.15s ease, opacity 0.15s ease;
}

.sp-business-platform-cancel {
  border: 1px solid #c5d0de;
  background: #f8fafc;
  color: #334155;
}

.sp-business-platform-cancel:hover:not(:disabled) {
  border-color: #94a3b8;
  background: #eef2f7;
  color: #0f172a;
}

.sp-business-platform-save {
  border: 1px solid #1d4ed8;
  background: linear-gradient(135deg, #2563eb, #1d4ed8 58%, #0f766e);
  color: #ffffff;
  box-shadow: 0 0.4rem 0.9rem rgba(37, 99, 235, 0.28);
}

.sp-business-platform-save:hover:not(:disabled) {
  border-color: #1e40af;
  background: linear-gradient(135deg, #1d4ed8, #1e40af 58%, #0f766e);
  color: #ffffff;
  box-shadow: 0 0.45rem 1rem rgba(29, 78, 216, 0.34);
}

.sp-business-platform-cancel:disabled,
.sp-business-platform-save:disabled {
  cursor: not-allowed;
  opacity: 0.55;
  box-shadow: none;
}

:global(.dark .modal-content:has(.sp-business-platform-dialog) .sp-business-platform-cancel) {
  border-color: #46576f;
  background: #243249;
  color: #dbe7f5;
}

:global(.dark .modal-content:has(.sp-business-platform-dialog) .sp-business-platform-cancel:hover:not(:disabled)) {
  border-color: #64748b;
  background: #2c3a51;
  color: #ffffff;
}

:global(.dark .modal-content:has(.sp-business-platform-dialog) .sp-business-platform-save) {
  border-color: #3b82f6;
  background: linear-gradient(135deg, #3b82f6, #2563eb 58%, #0e7490);
  color: #ffffff;
}

.sp-account-binding-dialog {
  display: grid;
  gap: 1rem;
}

.sp-account-binding-summary {
  position: relative;
  overflow: hidden;
  display: grid;
  gap: 0.75rem;
  padding: 1rem;
  border: 1px solid var(--sp-line);
  border-radius: 0.65rem;
  background: var(--sp-panel-2);
}

.sp-account-binding-accent {
  position: absolute;
  inset: 0 0 auto;
  height: 0.2rem;
}

.sp-account-binding-selected-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 1rem;
}

.sp-account-binding-selected-head > span {
  color: var(--sp-muted);
  font-size: 0.75rem;
}

.sp-account-binding-selected-groups {
  display: flex;
  min-height: 2rem;
  flex-wrap: wrap;
  align-items: center;
  gap: 0.5rem;
}

.sp-account-binding-primary {
  display: inline-flex;
  min-width: 7rem;
  min-height: 2.5rem;
  align-items: center;
  justify-content: center;
  border: 1px solid transparent;
  border-radius: 0.5rem;
  padding: 0.5rem 0.875rem;
  font-size: 0.875rem;
  font-weight: 600;
  cursor: pointer;
  transition: background-color 0.15s ease, opacity 0.15s ease;
}

.sp-account-binding-primary:disabled {
  cursor: not-allowed;
  opacity: 0.55;
}

.sp-account-binding-empty {
  border: 1px dashed var(--sp-line);
  border-radius: 0.6rem;
  padding: 0.7rem 0.8rem;
  color: var(--sp-muted);
  font-size: 0.75rem;
  line-height: 1.5;
}

.sp-account-empty {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 0.375rem;
}

.sp-account-empty strong {
  color: var(--sp-text);
  font-size: 0.9375rem;
}

.sp-account-empty span {
  color: var(--sp-muted);
  font-size: 0.8125rem;
}

.sp-account-pagination {
  display: flex;
  min-height: 4.25rem;
  align-items: center;
  justify-content: space-between;
  gap: 1rem;
  border-top: 1px solid var(--sp-line);
  background: var(--sp-panel);
}

.sp-account-page-size {
  display: flex;
  flex: 0 0 auto;
  align-items: center;
  gap: 0.5rem;
  padding-left: 1rem;
  color: var(--sp-muted);
  font-size: 0.8125rem;
}

.sp-account-page-size-select {
  width: 5.25rem;
}

.sp-account-page-size-select :deep(.select-trigger) {
  min-height: 2.25rem;
  padding-block: 0.375rem;
}

.sp-account-pagination :deep(.sp-data-pagination),
.sp-account-pagination :deep(> div) {
  border-top: 0;
  background: transparent;
}

:global(.modal-content:has(.sp-batch-test-dialog)),
:global(.modal-content:has(.sp-batch-result-dialog)) {
  --sp-panel: #ffffff;
  --sp-panel-2: #f8fafc;
  --sp-panel-3: #eef2f7;
  --sp-line: #d7e0ea;
  --sp-soft: #e8eef5;
  --sp-text: #172033;
  --sp-muted: #607089;
  --sp-dim: #8a99ad;
  --sp-cyan: #0284c7;
  --sp-green: #16835d;
  --sp-amber: #c56a0a;
  --sp-orange: #ea580c;
  --sp-red: #d14343;
  --sp-blue: #2563eb;
  --sp-violet: #7c3aed;
  --sp-shadow: 0 1px 2px rgba(15, 23, 42, 0.04);
  border-color: #cbd7e5;
  background: var(--sp-panel);
  color: var(--sp-text);
}

:global(.dark .modal-content:has(.sp-batch-test-dialog)),
:global(.dark .modal-content:has(.sp-batch-result-dialog)) {
  --sp-panel: #172033;
  --sp-panel-2: #1d293d;
  --sp-panel-3: #243249;
  --sp-line: #35445c;
  --sp-soft: #2c3a51;
  --sp-text: #edf3fb;
  --sp-muted: #a8b6ca;
  --sp-dim: #75849a;
  --sp-shadow: 0 1px 2px rgba(0, 0, 0, 0.2);
  border-color: #3b4b64;
}

.sp-batch-test-dialog,
.sp-batch-result-dialog {
  display: flex;
  flex-direction: column;
  gap: 1rem;
  overflow: hidden;
  border: 0;
  border-radius: 0;
  padding: 0.15rem 0.1rem 0.35rem;
  background: transparent;
  box-shadow: none;
}

.sp-batch-result-dialog {
  min-height: 0;
  flex: 1 1 auto;
  height: 100%;
  overflow: hidden;
}

.sp-batch-test-dialog {
  /* 弹窗外层已有 BaseDialog 边框，这里不再套一层卡片 */
  background: transparent;
}

.sp-batch-result-dialog {
  background: transparent;
}

.sp-batch-test-summary {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 0.75rem;
  border: 0;
  border-radius: 0;
  padding: 0;
  background: transparent;
  box-shadow: none;
}

.sp-batch-summary-metric {
  position: relative;
  display: flex;
  min-height: 3.75rem;
  align-items: center;
  justify-content: space-between;
  overflow: hidden;
  border: 0;
  border-radius: 0.75rem;
  padding: 0.75rem 0.875rem 0.75rem 1rem;
  background: color-mix(in srgb, var(--sp-panel-2) 88%, transparent);
  box-shadow: none;
}

.sp-batch-summary-metric.accounts {
  background: color-mix(in srgb, var(--sp-cyan) 10%, var(--sp-panel-2));
}

.sp-batch-summary-metric.platforms {
  background: color-mix(in srgb, var(--sp-amber) 10%, var(--sp-panel-2));
}

.sp-batch-summary-metric::before {
  position: absolute;
  inset: 0 auto 0 0;
  width: 0.25rem;
  content: '';
}

.sp-batch-summary-metric.accounts::before {
  background: linear-gradient(180deg, var(--sp-cyan), #2563eb);
}

.sp-batch-summary-metric.platforms::before {
  background: linear-gradient(180deg, var(--sp-amber), #ea580c);
}

.sp-batch-summary-metric > span {
  color: var(--sp-muted);
  font-size: 0.75rem;
  font-weight: 800;
  letter-spacing: 0.02em;
}

.sp-batch-summary-metric > div {
  display: flex;
  align-items: baseline;
  gap: 0.25rem;
}

.sp-batch-summary-metric strong {
  color: var(--sp-text);
  font-family: ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, monospace;
  font-size: 1.65rem;
  line-height: 1;
}

.sp-batch-summary-metric.accounts strong {
  color: var(--sp-cyan);
}

.sp-batch-summary-metric.platforms strong {
  color: var(--sp-amber);
}

.sp-batch-summary-metric small {
  color: var(--sp-muted);
  font-size: 0.7rem;
}

.sp-batch-test-summary p {
  display: flex;
  grid-column: 1 / -1;
  align-items: flex-start;
  gap: 0.5rem;
  margin: 0;
  border-top: 0;
  padding: 0.15rem 0.1rem 0;
  color: var(--sp-muted);
  font-size: 0.75rem;
  line-height: 1.55;
}

.sp-batch-test-summary p > span {
  width: 0.45rem;
  height: 0.45rem;
  flex: 0 0 auto;
  margin-top: 0.35rem;
  border-radius: 999px;
  background: var(--sp-cyan);
  box-shadow: 0 0 0 0.2rem color-mix(in srgb, var(--sp-cyan) 12%, transparent);
}

.sp-batch-test-section {
  overflow: hidden;
  border: 0;
  border-radius: 0;
  background: transparent;
  box-shadow: none;
}

.sp-batch-test-section.settings-section {
  border-color: transparent;
}

.sp-batch-test-section > header {
  display: flex;
  align-items: center;
  gap: 0.7rem;
  padding: 0.15rem 0.1rem 0.7rem;
  border-bottom: 1px solid color-mix(in srgb, var(--sp-line) 88%, transparent);
  background: transparent;
}

.sp-batch-test-section.settings-section > header {
  background: transparent;
}

.sp-batch-section-index {
  display: inline-flex;
  width: 1.8rem;
  height: 1.8rem;
  flex: 0 0 auto;
  align-items: center;
  justify-content: center;
  border: 1px solid color-mix(in srgb, var(--sp-cyan) 24%, var(--sp-line));
  border-radius: 0.5rem;
  background: color-mix(in srgb, var(--sp-cyan) 9%, var(--sp-panel));
  color: var(--sp-cyan);
  font-family: ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, monospace;
  font-size: 0.7rem;
  font-weight: 900;
}

.settings-section .sp-batch-section-index {
  border-color: color-mix(in srgb, var(--sp-amber) 26%, var(--sp-line));
  background: color-mix(in srgb, var(--sp-amber) 9%, var(--sp-panel));
  color: var(--sp-amber);
}

.sp-batch-test-section > header div {
  display: flex;
  min-width: 0;
  flex-direction: column;
  gap: 0.16rem;
}

.sp-batch-test-section > header strong {
  color: var(--sp-text);
  font-size: 0.875rem;
}

.sp-batch-test-section > header span:not(.sp-batch-section-index),
.sp-batch-test-platform-row small,
.sp-batch-result-empty {
  color: var(--sp-muted);
  font-size: 0.72rem;
}

.sp-batch-test-platform-list {
  display: flex;
  flex-direction: column;
  border: 0;
  background: transparent;
}

.sp-batch-test-platform-row {
  position: relative;
  display: grid;
  grid-template-columns: minmax(10rem, 0.75fr) minmax(15rem, 1.25fr);
  align-items: center;
  gap: 1rem;
  padding: 0.85rem 0.25rem 0.85rem 0.95rem;
  border-bottom: 1px solid color-mix(in srgb, var(--sp-line) 90%, transparent);
  background: transparent;
  transition: background 150ms ease;
}

.sp-batch-test-platform-row:hover {
  background: color-mix(in srgb, var(--sp-panel-2) 80%, transparent);
  transform: none;
}

.sp-batch-test-platform-row:last-child {
  border-bottom: 0;
}

.sp-batch-test-platform-accent {
  position: absolute;
  inset: 0.65rem auto 0.65rem 0;
  width: 0.22rem;
  border-radius: 0 999px 999px 0;
}

.sp-batch-test-platform-info {
  display: flex;
  min-width: 0;
  align-items: center;
  gap: 0.6rem;
}

.sp-batch-test-model-select {
  display: grid;
  grid-template-columns: auto minmax(0, 1fr);
  align-items: center;
  gap: 0.75rem;
}

.sp-batch-test-model-select > span {
  color: var(--sp-muted);
  font-size: 0.7rem;
  font-weight: 800;
  white-space: nowrap;
}

.sp-batch-test-settings {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 0.75rem;
  padding: 0.85rem 0.1rem 0.15rem;
}

.sp-batch-test-settings label {
  display: grid;
  grid-template-columns: 1fr auto;
  align-items: center;
  gap: 0.18rem 0.75rem;
  border: 0;
  border-radius: 0.7rem;
  padding: 0.75rem 0.8rem;
  background: color-mix(in srgb, var(--sp-amber) 8%, var(--sp-panel-2));
  box-shadow: inset 3px 0 0 color-mix(in srgb, var(--sp-amber) 55%, transparent);
}

.sp-batch-test-settings label > span {
  color: var(--sp-text);
  font-size: 0.78rem;
  font-weight: 800;
}

.sp-batch-test-settings label > small {
  grid-column: 1;
  color: var(--sp-muted);
  font-size: 0.68rem;
}

.sp-batch-test-settings label > :last-child {
  grid-column: 2;
  grid-row: 1 / 3;
  min-width: 9.5rem;
}

.sp-batch-secondary-button {
  border-color: color-mix(in srgb, var(--sp-muted) 32%, var(--sp-line));
}

.sp-batch-start-button,
.sp-batch-close-button {
  border-color: #059669;
  background: linear-gradient(135deg, #059669, #0d9488 58%, #0891b2);
  color: #fff;
  box-shadow: 0 0.45rem 1rem color-mix(in srgb, var(--sp-green) 24%, transparent);
}

.sp-batch-start-button {
  display: inline-flex;
  min-width: 12.5rem;
  align-items: center;
  justify-content: center;
  gap: 0.45rem;
}

.sp-batch-start-button:hover,
.sp-batch-close-button:hover {
  border-color: #047857;
  background: linear-gradient(135deg, #047857, #0f766e 58%, #0e7490);
  color: #fff;
}

.sp-batch-start-button:disabled {
  border-color: color-mix(in srgb, var(--sp-cyan) 18%, var(--sp-line));
  background: color-mix(in srgb, var(--sp-cyan) 12%, var(--sp-panel-2));
  color: color-mix(in srgb, var(--sp-cyan) 55%, var(--sp-muted));
  box-shadow: none;
}

.batch-result-progress-description {
  margin: 0;
  border: 0;
  border-radius: 0;
  padding: 0.15rem 0.1rem 0.35rem;
  background: transparent;
  color: var(--sp-muted);
  font-size: 0.8rem;
  line-height: 1.5;
}

.sync-confirm-summary {
  display: grid;
  grid-template-columns: repeat(4, minmax(0, 1fr));
  gap: 0.75rem;
  padding: 0;
  background: transparent;
}

.sync-result-stat {
  display: grid;
  gap: 0.25rem;
  border: 0;
  border-radius: 0.7rem;
  padding: 0.7rem 0.8rem;
  background: color-mix(in srgb, var(--sp-panel-2) 92%, transparent);
  box-shadow: none;
}

.sync-result-stat.total {
  background: color-mix(in srgb, var(--sp-blue) 10%, var(--sp-panel-2));
}

.sync-result-stat.completed {
  background: color-mix(in srgb, var(--sp-cyan) 10%, var(--sp-panel-2));
}

.sync-result-stat.success {
  background: color-mix(in srgb, var(--sp-green) 10%, var(--sp-panel-2));
}

.sync-result-stat.failed {
  background: color-mix(in srgb, var(--sp-red) 10%, var(--sp-panel-2));
}

.sync-result-stat span {
  color: var(--sp-muted);
  font-size: 0.7rem;
  font-weight: 700;
}

.sync-result-stat strong {
  color: var(--sp-text);
  font-family: ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, monospace;
  font-size: 1.25rem;
}

.sync-result-stat.total strong {
  color: var(--sp-blue);
}

.sync-result-stat.completed strong {
  color: var(--sp-cyan);
}

.sync-result-stat.success strong {
  color: var(--sp-green);
}

.sync-result-stat.failed strong {
  color: var(--sp-red);
}

.sync-confirm-body {
  display: flex;
  min-height: 0;
  flex: 1 1 auto;
  flex-direction: column;
  padding: 0;
  background: transparent;
}

.sync-confirm-section {
  display: flex;
  min-height: 0;
  flex: 1 1 auto;
  flex-direction: column;
  gap: 0;
  overflow: hidden;
  border: 0;
  border-radius: 0;
  background: transparent;
  box-shadow: none;
}

.sync-confirm-section-title {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 0.75rem;
  border-bottom: 1px solid color-mix(in srgb, var(--sp-line) 90%, transparent);
  padding: 0.2rem 0.1rem 0.7rem;
  background: transparent;
  color: var(--sp-text);
  font-size: 0.8rem;
  font-weight: 800;
}

.sync-confirm-section-title strong {
  display: inline-flex;
  min-width: 1.5rem;
  min-height: 1.5rem;
  align-items: center;
  justify-content: center;
  border-radius: 999px;
  background: var(--sp-panel-3);
  color: var(--sp-muted);
  font-size: 0.7rem;
}

.batch-result-scroll {
  display: flex;
  min-height: 0;
  max-height: none;
  flex: 1 1 auto;
  flex-direction: column;
  overflow: auto;
  overscroll-behavior: contain;
  scroll-padding-bottom: 1.5rem;
  -webkit-overflow-scrolling: touch;
}

.batch-result-list {
  flex: 0 0 auto;
  padding-bottom: 1.5rem;
}

.batch-result-toolbar {
  position: sticky;
  top: 0;
  z-index: 3;
  display: grid;
  gap: 0.5rem;
  border-bottom: 1px solid color-mix(in srgb, var(--sp-line) 90%, transparent);
  padding: 0.55rem 0.1rem 0.65rem;
  background: color-mix(in srgb, var(--sp-panel) 92%, transparent);
  backdrop-filter: blur(10px);
}

.batch-result-tabs {
  display: flex;
  gap: 0.4rem;
  overflow-x: auto;
  padding-bottom: 1px;
}

.batch-result-tab {
  display: inline-flex;
  min-height: 1.9rem;
  flex: 0 0 auto;
  align-items: center;
  gap: 0.4rem;
  border: 1px solid var(--sp-line);
  border-radius: 999px;
  padding: 0 0.65rem;
  background: var(--sp-panel);
  color: var(--sp-muted);
  font-size: 0.72rem;
  font-weight: 800;
  white-space: nowrap;
  transition: border-color 150ms ease, background 150ms ease, color 150ms ease;
}

.batch-result-tab:hover {
  border-color: color-mix(in srgb, var(--sp-blue) 35%, var(--sp-line));
  background: var(--sp-panel-2);
  color: var(--sp-text);
}

.batch-result-tab strong {
  color: var(--sp-text);
  font-size: 0.72rem;
}

.batch-result-tab-all.active {
  border-color: color-mix(in srgb, var(--sp-blue) 55%, var(--sp-line));
  background: var(--sp-blue);
  color: #fff;
}

.batch-result-tab-success.active,
.batch-result-tab-success_unschedulable.active {
  border-color: color-mix(in srgb, var(--sp-green) 55%, var(--sp-line));
  background: var(--sp-green);
  color: #fff;
}

.batch-result-tab-failed.active,
.batch-result-tab-failed_schedulable.active,
.batch-result-tab-failed_unschedulable.active {
  border-color: color-mix(in srgb, var(--sp-red) 55%, var(--sp-line));
  background: var(--sp-red);
  color: #fff;
}

.batch-result-tab-success_upstream_disabled.active {
  border-color: color-mix(in srgb, #7c3aed 58%, var(--sp-line));
  background: #7c3aed;
  color: #fff;
}

.batch-result-tab-skipped.active {
  border-color: color-mix(in srgb, var(--sp-muted) 65%, var(--sp-line));
  background: var(--sp-muted);
  color: #fff;
}

.batch-result-tab.active strong {
  color: inherit;
}

.batch-result-hint {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 0.5rem;
  color: var(--sp-muted);
  font-size: 0.72rem;
}

.batch-result-hint-tag {
  flex: none;
  border: 1px solid color-mix(in srgb, var(--sp-amber) 35%, var(--sp-line));
  border-radius: 999px;
  padding: 0.18rem 0.5rem;
  background: color-mix(in srgb, var(--sp-amber) 8%, var(--sp-panel));
  color: var(--sp-amber);
  font-size: 0.65rem;
  font-weight: 800;
}

.batch-result-list {
  display: grid;
  gap: 0.55rem;
  padding: 0.55rem 0.1rem 1.5rem;
}

.batch-result-card {
  position: relative;
  display: grid;
  gap: 0.65rem;
  overflow: hidden;
  border: 0;
  border-radius: 0.7rem;
  padding: 0.8rem 0.85rem 0.8rem 0.95rem;
  background: color-mix(in srgb, var(--sp-panel-2) 90%, transparent);
  box-shadow: inset 3px 0 0 color-mix(in srgb, var(--sp-line) 90%, transparent);
  transition: background 150ms ease, box-shadow 150ms ease;
}

.batch-result-card:hover {
  box-shadow: inset 3px 0 0 color-mix(in srgb, var(--sp-blue) 45%, var(--sp-line));
  transform: none;
  background: color-mix(in srgb, var(--sp-panel-2) 98%, transparent);
}

.batch-result-card.success {
  border-color: transparent;
  background: color-mix(in srgb, var(--sp-green) 9%, var(--sp-panel-2));
  box-shadow: inset 3px 0 0 color-mix(in srgb, var(--sp-green) 70%, transparent);
}

.batch-result-card.failed {
  border-color: transparent;
  background: color-mix(in srgb, var(--sp-red) 9%, var(--sp-panel-2));
  box-shadow: inset 3px 0 0 color-mix(in srgb, var(--sp-red) 70%, transparent);
}

.batch-result-card.failed-schedulable {
  border-color: transparent;
  box-shadow: inset 3px 0 0 color-mix(in srgb, var(--sp-red) 85%, transparent);
  background: color-mix(in srgb, var(--sp-red) 12%, var(--sp-panel-2));
}

.batch-result-card.warning {
  border-color: transparent;
  background: color-mix(in srgb, var(--sp-amber) 10%, var(--sp-panel-2));
  box-shadow: inset 3px 0 0 color-mix(in srgb, var(--sp-amber) 70%, transparent);
}

.batch-result-card.neutral {
  background: color-mix(in srgb, var(--sp-muted) 6%, var(--sp-panel-2));
  box-shadow: inset 3px 0 0 color-mix(in srgb, var(--sp-muted) 45%, transparent);
}

.batch-result-card::before {
  position: absolute;
  inset: 0 auto 0 0;
  width: 3px;
  background: var(--sp-green);
  content: '';
}

.batch-result-card.failed::before {
  background: var(--sp-red);
}

.batch-result-card.warning::before {
  background: var(--sp-amber);
}

.batch-result-card.neutral::before {
  background: var(--sp-muted);
}

.batch-result-card-head {
  display: flex;
  min-width: 0;
  align-items: flex-start;
  justify-content: space-between;
  gap: 0.65rem;
}

.batch-result-account {
  display: grid;
  min-width: 0;
  gap: 0.2rem;
}

.batch-result-account strong {
  overflow-wrap: anywhere;
  color: var(--sp-text);
  font-size: 0.875rem;
  font-weight: 800;
  line-height: 1.3;
}

.batch-result-account span {
  color: var(--sp-muted);
  font-size: 0.72rem;
}

.batch-result-card-status {
  display: flex;
  flex: none;
  align-items: flex-end;
  flex-direction: column;
  gap: 0.35rem;
}

.batch-result-risk-tag {
  display: inline-flex;
  align-items: center;
  border: 1px solid color-mix(in srgb, var(--sp-red) 42%, var(--sp-line));
  border-radius: 999px;
  padding: 0.16rem 0.48rem;
  background: color-mix(in srgb, var(--sp-red) 12%, var(--sp-panel));
  color: var(--sp-red);
  font-size: 0.64rem;
  font-weight: 850;
  white-space: nowrap;
}

.batch-result-grid {
  display: grid;
  grid-template-columns: repeat(5, minmax(0, 1fr));
  gap: 0.45rem;
}

.batch-result-metric {
  display: grid;
  min-width: 0;
  gap: 0.2rem;
  border: 0;
  border-radius: 0;
  padding: 0.15rem 0.1rem;
  background: transparent;
}

.batch-result-card.success .batch-result-metric,
.batch-result-card.failed .batch-result-metric,
.batch-result-card.warning .batch-result-metric {
  border-color: transparent;
  background: transparent;
}

.batch-result-metric > span {
  color: var(--sp-muted);
  font-size: 0.66rem;
  font-weight: 800;
}

.batch-result-metric > strong {
  min-width: 0;
  overflow-wrap: anywhere;
  color: var(--sp-text);
  font-size: 0.76rem;
  font-weight: 800;
}

.batch-result-schedule-status {
  display: inline-flex;
  width: fit-content;
  align-items: center;
  gap: 0.35rem;
}

.batch-result-schedule-status::before {
  width: 0.42rem;
  height: 0.42rem;
  flex: none;
  border-radius: 999px;
  background: currentColor;
  content: '';
}

.batch-result-schedule-status.enabled {
  color: var(--sp-green);
}

.batch-result-schedule-status.disabled {
  color: var(--sp-amber);
}

.batch-result-error {
  overflow-wrap: anywhere;
  border: 0;
  border-radius: 0.5rem;
  padding: 0.55rem 0.65rem;
  background: color-mix(in srgb, var(--sp-red) 10%, var(--sp-panel-2));
  color: color-mix(in srgb, var(--sp-red) 82%, var(--sp-text));
  font-size: 0.72rem;
  font-weight: 650;
  line-height: 1.45;
  box-shadow: inset 3px 0 0 color-mix(in srgb, var(--sp-red) 70%, transparent);
}

.batch-result-empty {
  padding: 2rem 1rem;
  color: var(--sp-muted);
  font-size: 0.8rem;
  text-align: center;
}

.batch-result-card-actions {
  display: flex;
  justify-content: flex-end;
  border-top: 1px solid color-mix(in srgb, var(--sp-line) 78%, transparent);
  padding-top: 0.6rem;
}

.batch-result-schedule-button {
  display: inline-flex;
  min-height: 2rem;
  align-items: center;
  justify-content: center;
  border: 1px solid;
  border-radius: 0.5rem;
  padding: 0 0.7rem;
  font-size: 0.72rem;
  font-weight: 800;
  transition: border-color 150ms ease, background 150ms ease, color 150ms ease, box-shadow 150ms ease;
}

.batch-result-schedule-button.enable-action {
  border-color: color-mix(in srgb, var(--sp-green) 48%, var(--sp-line));
  background: color-mix(in srgb, var(--sp-green) 12%, var(--sp-panel));
  color: var(--sp-green);
}

.batch-result-schedule-button.disable-action {
  border-color: color-mix(in srgb, var(--sp-amber) 48%, var(--sp-line));
  background: color-mix(in srgb, var(--sp-amber) 12%, var(--sp-panel));
  color: var(--sp-amber);
}

.batch-result-schedule-button:hover:not(:disabled) {
  box-shadow: 0 0 0 3px color-mix(in srgb, currentColor 10%, transparent);
}

.batch-result-schedule-button:disabled {
  cursor: not-allowed;
  opacity: 0.58;
}

.batch-result-button {
  display: inline-flex;
  min-height: 2.375rem;
  align-items: center;
  justify-content: center;
  gap: 0.5rem;
  border: 1px solid var(--sp-line);
  border-radius: 0.5rem;
  padding: 0 0.875rem;
  background: var(--sp-panel);
  color: var(--sp-text);
  font-weight: 600;
  transition: border-color 150ms ease, background 150ms ease, color 150ms ease, box-shadow 150ms ease;
}

.batch-result-button:hover:not(:disabled) {
  border-color: color-mix(in srgb, var(--sp-green) 45%, var(--sp-line));
  color: var(--sp-green);
  box-shadow: 0 0 0 3px color-mix(in srgb, var(--sp-green) 8%, transparent);
}

.batch-result-button:disabled {
  cursor: not-allowed;
  opacity: 0.55;
}

.batch-result-button-primary {
  border-color: #059669;
  background: #059669;
  color: #fff;
}

.batch-result-button-primary:hover:not(:disabled) {
  border-color: #047857;
  background: #047857;
  color: #fff;
}
:global(.modal-content:has(.sp-batch-test-dialog) .modal-footer) {
  border-top-color: color-mix(in srgb, var(--sp-green) 18%, var(--sp-line));
  background: linear-gradient(90deg, color-mix(in srgb, var(--sp-green) 7%, var(--sp-panel)), color-mix(in srgb, var(--sp-cyan) 4%, var(--sp-panel)));
}

:global(.modal-content:has(.sp-batch-result-dialog) .modal-footer) {
  border-top-color: color-mix(in srgb, var(--sp-green) 22%, var(--sp-line));
  background: linear-gradient(90deg, color-mix(in srgb, var(--sp-green) 9%, var(--sp-panel)), color-mix(in srgb, var(--sp-cyan) 5%, var(--sp-panel)));
}

:global(.modal-content:has(.sp-batch-test-dialog)) {
  width: min(1040px, calc(100vw - 32px));
  max-width: none;
  overflow-x: hidden;
}

:global(.modal-content:has(.sp-batch-result-dialog)) {
  width: min(1600px, calc(100vw - 32px));
  max-width: none;
  max-height: min(92vh, 92dvh);
  overflow-x: hidden;
  overflow-y: hidden;
}

:global(.modal-content:has(.sp-batch-result-dialog) .modal-body) {
  display: flex;
  min-height: 0;
  flex: 1 1 auto;
  flex-direction: column;
  overflow: hidden;
  padding-bottom: 0.75rem;
}

:global(.modal-content:has(.sp-batch-result-dialog) .modal-footer) {
  flex: 0 0 auto;
}

@media (max-width: 1280px) {
  .sp-account-panel-head {
    grid-template-columns: minmax(13rem, 1fr) auto;
  }

  .sp-account-quick-filters {
    grid-column: 1 / -1;
    grid-row: 2;
    justify-content: flex-start;
    flex-wrap: wrap;
  }

}

@media (max-width: 900px) {
  .sp-batch-test-platform-row {
    grid-template-columns: 1fr;
    gap: 0.65rem;
  }

  .sp-batch-test-model-select {
    grid-template-columns: minmax(5rem, auto) minmax(0, 1fr);
  }

  .sync-confirm-summary,
  .batch-result-grid {
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }

  .batch-result-list {
    grid-template-columns: 1fr;
  }

  .sp-account-filter-body {
    align-items: stretch;
    flex-direction: column;
  }

  .sp-account-filter-fields {
    grid-template-columns: 1fr 1fr;
  }

  .sp-account-search {
    grid-column: 1 / -1;
  }

  .sp-account-filter-actions {
    width: 100%;
    justify-content: flex-start;
    gap: 0.625rem;
    padding-top: 0.75rem;
    padding-left: 0;
    border-top: 1px solid var(--sp-line);
    border-left: 0;
  }

  .sp-account-toolbar-btn {
    flex: 1 1 calc(50% - 0.625rem);
    justify-content: center;
  }

  .sp-account-pagination {
    align-items: stretch;
    flex-direction: column;
    gap: 0;
  }

  .sp-account-page-size {
    justify-content: center;
    padding: 0.875rem 1rem 0;
  }
}

@media (max-width: 760px) {
  .sp-account-toolbar {
    margin-bottom: 0.75rem;
  }

  .sp-filter-card-head,
  .sp-account-filter-body {
    padding-inline: 0.75rem;
  }

  .sp-account-panel-head {
    grid-template-columns: 1fr;
    min-height: auto;
  }

  .sp-account-quick-filters {
    grid-column: auto;
    grid-row: auto;
  }

  .sp-account-legend {
    width: 100%;
  }

  .sp-account-table-shell {
    min-height: 0;
    padding: 0.75rem;
  }

  .sp-account-table-shell :deep(> .space-y-3 > div) {
    border-color: var(--sp-line);
    background: var(--sp-panel-2);
  }
}

@media (max-width: 520px) {
  .sp-batch-test-summary,
  .sp-batch-test-settings {
    grid-template-columns: 1fr;
  }

  .sp-batch-test-model-select {
    grid-template-columns: 1fr;
  }

  .sp-batch-test-settings label {
    grid-template-columns: 1fr;
  }

  .sp-batch-test-settings label > :last-child {
    grid-column: 1;
    grid-row: auto;
    margin-top: 0.35rem;
  }

  .sync-confirm-summary,
  .batch-result-grid {
    grid-template-columns: 1fr;
  }

  .batch-result-hint {
    align-items: flex-start;
    flex-direction: column;
  }

  .sp-batch-start-button,
  .batch-result-button {
    width: 100%;
    min-width: 0;
  }

  .sp-filter-card-head {
    align-items: flex-start;
    flex-direction: column;
  }

  /* 手机端筛选区保持紧凑：搜索通栏，下拉与按钮 2 列，避免全部整行占高 */
  .sp-account-filter-fields {
    grid-template-columns: repeat(2, minmax(0, 1fr));
    gap: 0.625rem;
  }

  .sp-account-search {
    grid-column: 1 / -1;
  }

  .sp-account-filter-actions {
    display: grid;
    grid-template-columns: repeat(2, minmax(0, 1fr));
    gap: 0.5rem;
  }

  .sp-account-toolbar-btn,
  .sp-account-refresh {
    flex: initial;
    width: 100%;
    min-width: 0;
    min-height: 2.5rem;
    padding-inline: 0.55rem;
    font-size: 0.8125rem;
    white-space: normal;
    line-height: 1.25;
    text-align: center;
  }

  .sp-account-binding-selected-head {
    align-items: flex-start;
    flex-direction: column;
    gap: 0.5rem;
  }
}

@media (prefers-reduced-motion: reduce) {
  .sp-account-table-shell :deep(tbody tr) {
    transition: none;
  }
}


/* 手机端结果弹窗：压缩顶部统计区，列表独立滚动，避免底部账号被裁切 */
@media (max-width: 640px) {
  :global(.modal-overlay:has(.sp-batch-result-dialog)) {
    align-items: stretch;
    justify-content: stretch;
    padding: 0.375rem;
    padding-bottom: max(0.375rem, env(safe-area-inset-bottom, 0px));
  }

  :global(.modal-content:has(.sp-batch-result-dialog)) {
    width: 100%;
    max-width: none;
    max-height: none;
    height: 100%;
    margin: 0;
    border-radius: 1rem;
  }

  :global(.modal-content:has(.sp-batch-result-dialog) .modal-header) {
    flex: 0 0 auto;
    padding: 0.65rem 0.85rem;
  }

  :global(.modal-content:has(.sp-batch-result-dialog) .modal-body) {
    display: flex;
    min-height: 0;
    flex: 1 1 auto;
    flex-direction: column;
    overflow: hidden;
    padding: 0.55rem 0.7rem 0.35rem;
  }

  :global(.modal-content:has(.sp-batch-result-dialog) .modal-footer) {
    flex: 0 0 auto;
    padding: 0.55rem 0.7rem;
    padding-bottom: max(0.55rem, env(safe-area-inset-bottom, 0px));
  }

  .sp-batch-result-dialog {
    display: flex;
    min-height: 0;
    height: 100%;
    flex: 1 1 auto;
    flex-direction: column;
    gap: 0.55rem;
    overflow: hidden;
    padding: 0.1rem 0;
    border: 0;
    background: transparent;
  }

  .batch-result-progress-description {
    flex: 0 0 auto;
    padding: 0.5rem 0.7rem;
    font-size: 0.72rem;
    line-height: 1.4;
  }

  .sp-batch-result-dialog .sync-confirm-summary {
    flex: 0 0 auto;
    grid-template-columns: repeat(2, minmax(0, 1fr));
    gap: 0.45rem;
  }

  .sp-batch-result-dialog .sync-result-stat {
    gap: 0.1rem;
    padding: 0.45rem 0.55rem;
  }

  .sp-batch-result-dialog .sync-result-stat span {
    font-size: 0.62rem;
  }

  .sp-batch-result-dialog .sync-result-stat strong {
    font-size: 1rem;
  }

  .sp-batch-result-dialog .sync-confirm-body,
  .sp-batch-result-dialog .sync-confirm-section {
    display: flex;
    min-height: 0;
    flex: 1 1 auto;
    flex-direction: column;
    overflow: hidden;
  }

  .sp-batch-result-dialog .sync-confirm-section-title {
    flex: 0 0 auto;
    padding: 0.2rem 0.1rem 0.55rem;
    font-size: 0.75rem;
  }

  .sp-batch-result-dialog .batch-result-scroll {
    display: flex;
    min-height: 0;
    max-height: none;
    flex: 1 1 auto;
    flex-direction: column;
    overflow: auto;
    overscroll-behavior: contain;
    scroll-padding-bottom: 2rem;
    -webkit-overflow-scrolling: touch;
  }

  .sp-batch-result-dialog .batch-result-toolbar {
    position: sticky;
    top: 0;
    z-index: 2;
    flex: 0 0 auto;
    gap: 0.35rem;
    padding: 0.45rem;
  }

  .sp-batch-result-dialog .batch-result-hint {
    display: none;
  }

  .sp-batch-result-dialog .batch-result-list {
    display: grid;
    min-height: auto;
    flex: 0 0 auto;
    gap: 0.5rem;
    overflow: visible;
    padding: 0.5rem 0.5rem 2rem;
  }

  .sp-batch-result-dialog .batch-result-card {
    gap: 0.45rem;
    padding: 0.6rem;
  }

  .sp-batch-result-dialog .batch-result-card:last-child {
    margin-bottom: 0.25rem;
  }

  .sp-batch-result-dialog .batch-result-grid {
    grid-template-columns: repeat(2, minmax(0, 1fr));
    gap: 0.35rem 0.5rem;
  }

  .sp-batch-result-dialog .batch-result-metric span {
    font-size: 0.62rem;
  }

  .sp-batch-result-dialog .batch-result-metric strong {
    font-size: 0.75rem;
  }

  .sp-batch-result-dialog .batch-result-empty {
    padding-bottom: 1.5rem;
  }
}

</style>

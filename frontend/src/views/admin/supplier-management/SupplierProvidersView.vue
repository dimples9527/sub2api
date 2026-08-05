<template>
  <SupplierModuleLayout>
    <section class="sp-provider-filter-card" aria-label="供应商筛选与操作">
      <header class="sp-filter-card-head">
        <div>
          <span class="sp-filter-card-kicker">筛选条件</span>
          <h2>筛选供应商</h2>
          <p>集中搜索供应商，并在同一区域完成数据刷新和维护操作。</p>
        </div>
        <span class="sp-filter-card-count">{{ sortedProviders.length }} 个结果</span>
      </header>

      <div class="sp-provider-filter-body">
        <div class="sp-provider-filter-fields">
          <div
            class="sp-provider-filter-control"
            role="group"
            aria-labelledby="supplier-provider-search-label"
          >
            <span id="supplier-provider-search-label" class="sr-only">供应商搜索</span>
            <Input v-model="search" class="w-full" placeholder="搜索供应商" @enter="loadProviders" />
          </div>
          <div class="sp-provider-quick-filters" role="group" aria-label="供应商快捷筛选">
            <button
              v-for="quickFilter in providerQuickFilters"
              :key="quickFilter.key"
              class="sp-button small ghost"
              :class="{ active: providerQuickFilter === quickFilter.key }"
              :data-test="`supplier-provider-filter-${quickFilter.key}`"
              type="button"
              @click="providerQuickFilter = quickFilter.key"
            >
              {{ quickFilter.label }}
            </button>
          </div>
        </div>
        <div class="sp-provider-filter-actions">
          <button class="sp-button" type="button" :disabled="loading || costTrendLoading" @click="refreshProvidersView">刷新数据</button>
          <button class="sp-button" type="button" @click="openTypeManager">类型维护</button>
          <button class="sp-button" type="button" @click="openCreateProviderType">新增供应商类型</button>
          <button class="sp-button primary" type="button" @click="openCreate">新增供应商</button>
        </div>
      </div>
    </section>
    <div v-if="error" class="sp-alert sp-error-line">{{ error }}</div>

    <section class="sp-metric-grid">
      <article
        v-for="metric in metrics"
        :key="metric.key"
        class="sp-metric-card"
        :class="[`sp-${metric.tone}`, { selected: filter === metric.key }]"
        @click="filter = metric.key"
      >
        <div class="sp-metric-label">{{ metric.label }}</div>
        <div class="sp-metric-value">{{ metric.value }}</div>
        <div class="sp-metric-foot">{{ metric.foot }}</div>
      </article>
    </section>

    <section class="sp-grid-2">
      <div class="sp-panel">
        <header class="sp-panel-head">
          <div class="sp-panel-title">
            <span class="sp-section-index">01</span>
            <div>
              <h2>供应商运行列表</h2>
              <span>默认按真实业务风险排序</span>
            </div>
          </div>
        </header>

        <DataTable
          :columns="providerColumns"
          :data="sortedProviders"
          :loading="loading"
          row-key="id"
          server-side-sort
          clickable-rows
          @sort="handleProviderSort"
          @row-click="selectProviderForDetail"
        >
          <template #cell-homepage="{ row: provider }">
            <button
              class="sp-provider-home-button"
              type="button"
              :disabled="!provider.base_url?.trim()"
              :title="`打开${provider.name}主页`"
              :aria-label="`打开${provider.name}主页`"
              :data-test="`supplier-provider-home-${provider.id}`"
              @click.stop="openProviderHomepage(provider)"
            >
              <Icon name="home" size="sm" />
            </button>
          </template>
          <template #cell-name="{ row: provider }">
            <div class="sp-entity">{{ provider.name }}</div>
            <div class="sp-sub">{{ provider.code }} · {{ provider.provider_type }} · {{ provider.base_url }}</div>
          </template>
          <template #cell-status="{ row: provider }">
            <div class="sp-provider-status-toggle">
              <Toggle
                :model-value="provider.enabled"
                :disabled="updatingProviderIDs.has(provider.id)"
                :aria-label="`${provider.name}运行状态：${provider.enabled ? '已启用' : '已停用'}`"
                :data-test="`supplier-provider-enabled-${provider.id}`"
                @click.stop
                @update:model-value="updateProviderEnabled(provider, $event)"
              />
              <span class="sp-status" :class="statusTone(provider)">{{ statusText(provider) }}</span>
            </div>
          </template>
          <template #cell-account_counts="{ row: provider }">
            <span class="sp-num">{{ provider.valid_account_count }} / {{ provider.schedulable_account_count }}</span>
          </template>
          <template #cell-success_rate="{ row: provider }">
            <span :class="{ 'sp-up': provider.success_rate > 0 && provider.success_rate < 95 }">{{ percent(provider.success_rate) }}</span>
          </template>
          <template #cell-today_cost="{ row: provider }">
            <span class="sp-provider-today-cost">{{ currency(provider.today_cost) }}</span>
          </template>
          <template #cell-current_balance="{ row: provider }">
            <span :class="isBalanceWarning(provider) ? 'sp-provider-balance-warning' : 'sp-provider-balance-normal'">
              {{ balanceText(provider) }}
            </span>
          </template>
          <template #cell-rate_risk_count="{ row: provider }">
            <span class="sp-status" :class="rateTone(provider)">{{ rateRiskText(provider) }}</span>
          </template>
          <template #cell-credential_configured="{ row: provider }">
            <span class="sp-status" :class="provider.credential_configured ? 'good' : 'warn'">{{ provider.credential_configured ? '已配置' : '未配置' }}</span>
          </template>
          <template #cell-last_sync_at="{ row: provider }">
            {{ syncText(provider) }}
          </template>
          <template #cell-auth_summary="{ row: provider }">
            <div class="sp-entity">{{ provider.auth_summary?.login_count || 0 }} 次</div>
            <div class="sp-sub">成功 {{ provider.auth_summary?.login_success_count || 0 }} / 失败 {{ provider.auth_summary?.login_failure_count || 0 }}</div>
          </template>
          <template #cell-actions="{ row: provider }">
            <div class="sp-inline" @click.stop>
              <button class="sp-button small" type="button" @click="openAuthHistory(provider)">登录记录</button>
              <button class="sp-button small" type="button" @click="openEdit(provider)">编辑</button>
              <button class="sp-button small" type="button" :disabled="isSyncing(provider, 'all')" @click="syncProviderData(provider, 'all')">{{ isSyncing(provider, 'all') ? '同步中' : '同步全部' }}</button>
              <button class="sp-button small" type="button" :disabled="provider.is_default" @click="makeDefault(provider)">默认</button>
              <button class="sp-button small danger" type="button" @click="removeProvider(provider)">删除</button>
            </div>
          </template>
          <template #empty>
            没有符合条件的供应商
          </template>
        </DataTable>
      </div>

            <aside class="sp-panel sp-health-panel" data-test="supplier-health-panel">
        <header class="sp-panel-head">
          <div class="sp-panel-title">
            <span class="sp-section-index">02</span>
            <div>
              <h2>供应商组合健康</h2>
              <span>综合状态 · 优先待办 · 成本对比</span>
            </div>
          </div>
          <div class="sp-health-head-right">
            <button
              class="sp-button small ghost"
              type="button"
              :disabled="costTrendLoading"
              data-test="supplier-cost-refresh"
              title="按当前时间范围向上游回补成本并刷新曲线（NewAPI 支持历史，Sub2API 仅当天）"
              @click="refreshCostTrends"
            >
              {{ costTrendLoading ? '回补中…' : '重新获取' }}
            </button>
            <span class="sp-status" :class="healthTone" data-test="supplier-health-tone">{{ healthLabel }}</span>
          </div>
        </header>
        <div class="sp-panel-body sp-health-body">
          <div class="sp-health-summary" :class="healthTone" data-test="supplier-health-summary">
            <strong>{{ healthLabel }}</strong>
            <p>{{ healthMessage }}</p>
          </div>

          <section v-if="priorityTodos.length" class="sp-health-section" data-test="supplier-health-todos">
            <div class="sp-health-section-head">
              <span>优先待办</span>
              <small>最多展示 3 项，点击可联动筛选</small>
            </div>
            <div class="sp-health-todo-list">
              <button
                v-for="todo in priorityTodos"
                :key="todo.key"
                class="sp-health-todo"
                :class="todo.tone"
                type="button"
                :data-test="`supplier-health-todo-${todo.key}`"
                @click="applyHealthTodo(todo)"
              >
                <div class="sp-health-todo-main">
                  <strong>{{ todo.title }}</strong>
                  <small>{{ todo.detail }}</small>
                </div>
                <span class="sp-health-todo-action">处理</span>
              </button>
            </div>
          </section>

          <section class="sp-health-section" data-test="supplier-health-completeness">
            <div class="sp-health-section-head">
              <span>配置完整性</span>
              <small>基础配置与调度就绪度</small>
            </div>
            <div class="sp-list sp-health-completeness-list">
              <div class="sp-list-item">
                <div>
                  <strong>默认供应商</strong>
                  <small>{{ defaultProvider ? `${defaultProvider.name} · ${defaultProvider.code}` : '尚未配置' }}</small>
                </div>
                <span class="sp-status" :class="defaultProvider ? 'good' : 'warn'">{{ defaultProvider ? '已设置' : '缺失' }}</span>
              </div>
              <div class="sp-list-item">
                <div>
                  <strong>凭据覆盖</strong>
                  <small>{{ credentialCoverage }}</small>
                </div>
                <span class="sp-status" :class="credentialMissingCount ? 'warn' : 'good'">{{ credentialMissingCount ? '需补充' : '完整' }}</span>
              </div>
              <div class="sp-list-item">
                <div>
                  <strong>倍率风险</strong>
                  <small>当前累计 {{ summary.rate_risk_count }} 个风险项</small>
                </div>
                <span class="sp-status" :class="summary.rate_risk_count ? 'warn' : 'good'">{{ summary.rate_risk_count ? '关注' : '正常' }}</span>
              </div>
              <div class="sp-list-item">
                <div>
                  <strong>零可调度</strong>
                  <small>启用中但无可调度账号 {{ zeroSchedulableCount }} 个</small>
                </div>
                <span class="sp-status" :class="zeroSchedulableCount ? 'warn' : 'good'">{{ zeroSchedulableCount ? '关注' : '正常' }}</span>
              </div>
            </div>
          </section>

          <section class="sp-health-section sp-health-chart-section" data-test="supplier-cost-trend">
            <div class="sp-health-section-head sp-health-cost-section-head">
              <div class="sp-health-section-title">
                <span>成本对比</span>
                <small>{{ costTrendRangeLabel }} · {{ costTrendScopeLabel }} · 上游成本 / 本地成本</small>
              </div>
              <div class="sp-health-date-range-control" data-test="supplier-cost-controls">
                <span class="sp-health-control-label">日期范围</span>
                <div class="sp-health-date-range" data-test="supplier-cost-date-range">
                  <DateRangePicker
                    v-model:start-date="costTrendStartDate"
                    v-model:end-date="costTrendEndDate"
                    @change="onCostTrendDateRangeChange"
                  />
                </div>
              </div>
            </div>

            <div class="sp-health-chart-controls">
              <div class="sp-health-provider-filter w-full sm:w-44 flex items-center gap-2">
                <Select
                  v-model="costTrendProviderId"
                  class="w-full"
                  :options="costTrendProviderOptions"
                  aria-label="成本对比供应商"
                  data-test="supplier-cost-provider"
                  @update:model-value="onCostTrendProviderChange"
                />
                <span class="text-xs text-gray-500 whitespace-nowrap">偏差阈值</span>
                <Select
                  v-model="deviationThreshold"
                  class="w-24"
                  :options="deviationThresholdOptions"
                  aria-label="成本偏差阈值"
                />
                <span class="text-xs font-mono w-12 text-blue-600">{{ (deviationThreshold * 100).toFixed(0) }}%</span>
              </div>
            </div>

            <div class="sp-health-chart-meta">
              <div class="sp-health-chart-legend">
                <span class="sp-health-legend-item upstream"><i></i>上游成本</span>
                <span class="sp-health-legend-item local"><i></i>本地成本</span>
              </div>
              <div class="sp-health-chart-totals">
                <span>上游合计 <b>{{ currency(costTrendTotals.upstream) }}</b></span>
                <span>本地合计 <b>{{ currency(costTrendTotals.local) }}</b></span>
              </div>
            </div>
            <div class="sp-health-chart-canvas">
              <Line
                v-if="costTrendChartData"
                :data="costTrendChartData"
                :options="costTrendChartOptions"
              />
              <div v-else-if="costTrendLoading" class="sp-health-chart-empty">成本曲线加载中…</div>
              <div v-else class="sp-health-chart-empty">暂无按天成本数据，可调整时间范围或供应商后点「重新获取」向上游回补</div>
            </div>
          </section>

        </div>
      </aside>
    </section>

    <section class="sp-panel sp-cost-breakdown-panel" data-test="supplier-cost-breakdown-panel">
      <header class="sp-panel-head">
        <div class="sp-cost-breakdown-head-left" data-test="supplier-cost-breakdown-head-left">
          <div class="sp-health-date-range-control" data-test="supplier-cost-breakdown-controls">
            <span class="sp-health-control-label">日期范围</span>
            <div class="sp-health-date-range" data-test="supplier-cost-breakdown-date-range">
              <DateRangePicker
                v-model:start-date="costBreakdownStartDate"
                v-model:end-date="costBreakdownEndDate"
                @change="onCostBreakdownDateRangeChange"
              />
            </div>
          </div>
          <div class="sp-panel-title">
            <span class="sp-section-index">03</span>
            <div>
              <h2>按供应商拆分成本</h2>
              <span>{{ costBreakdownRangeLabel }} · 每个供应商并排比较上游成本和本地成本</span>
            </div>
          </div>
        </div>
        <span class="sp-cost-breakdown-count">供应商 <b>{{ costBreakdown.length }}</b> 个</span>
      </header>
      <div class="sp-panel-body sp-cost-breakdown-body" data-test="supplier-cost-breakdown">
        <div class="sp-health-chart-meta">
          <div class="sp-health-chart-legend">
            <span class="sp-health-legend-item upstream"><i></i>上游成本</span>
            <span class="sp-health-legend-item local"><i></i>本地成本</span>
          </div>
        </div>
        <div class="sp-health-breakdown-chart" data-test="supplier-cost-breakdown-chart-container">
          <Bar
            v-if="costBreakdownChartData"
            :data="costBreakdownChartData"
            :options="costBreakdownChartOptions"
            data-test="supplier-cost-breakdown-chart"
          />
          <div v-else-if="costBreakdownLoading" class="sp-health-chart-empty">供应商成本加载中…</div>
          <div v-else class="sp-health-chart-empty">暂无供应商成本数据，可调整时间范围后重试</div>
        </div>
      </div>
    </section>

    <div class="sp-footer-note">
      <span>数据来源：新供应商管理接口</span>
      <span>编辑时密码留空会保留原凭据</span>
    </div>

    <SupplierDrawer
      :show="Boolean(selectedProvider)"
      :title="selectedProvider?.name || ''"
      eyebrow="PROVIDER DETAIL"
      @close="selectedProvider = null"
    >
      <template v-if="selectedProvider">
        <div class="sp-alert">{{ selectedProvider.name }} 当前运行统计来自独立供应商数据表，后续同步任务写入后会自动更新。</div>
        <div class="sp-detail-grid">
          <div class="sp-detail-cell"><span>供应商编码</span><b>{{ selectedProvider.code }}</b></div>
          <div class="sp-detail-cell"><span>供应商类型</span><b>{{ selectedProvider.provider_type }}</b></div>
          <div class="sp-detail-cell"><span>有效 / 可调度账号</span><b>{{ selectedProvider.valid_account_count }} / {{ selectedProvider.schedulable_account_count }}</b></div>
          <div class="sp-detail-cell"><span>成功率</span><b>{{ percent(selectedProvider.success_rate) }}</b></div>
          <div class="sp-detail-cell"><span>今日成本</span><b>{{ currency(selectedProvider.today_cost) }}</b></div>
          <div class="sp-detail-cell"><span>当前余额</span><b>{{ currency(selectedProvider.current_balance) }}</b></div>
          <div class="sp-detail-cell"><span>预计可用</span><b :class="{ 'sp-up': isLowBalance(selectedProvider) }">{{ balanceText(selectedProvider) }}</b></div>
          <div class="sp-detail-cell"><span>最近同步</span><b>{{ syncText(selectedProvider) }}</b></div>
        </div>
        <div class="sp-drawer-actions">
          <button class="sp-button primary" type="button" @click="openEdit(selectedProvider)">编辑配置</button>
          <button class="sp-button" type="button" @click="openAuthHistory(selectedProvider)">登录与 Token</button>
          <button class="sp-button" type="button" :disabled="isSyncing(selectedProvider, 'accounts')" @click="syncProviderData(selectedProvider, 'accounts')">同步 API Key</button>
          <button class="sp-button" type="button" :disabled="isSyncing(selectedProvider, 'groups')" @click="syncProviderData(selectedProvider, 'groups')">同步分组</button>
          <button class="sp-button" type="button" :disabled="isSyncing(selectedProvider, 'balance')" @click="syncProviderData(selectedProvider, 'balance')">刷新余额</button>
          <button class="sp-button" type="button" :disabled="isSyncing(selectedProvider, 'cost')" @click="syncProviderData(selectedProvider, 'cost')">刷新成本</button>
          <button class="sp-button" type="button" :disabled="isTesting(selectedProvider, 'accounts')" @click="testProviderEndpointData(selectedProvider, 'accounts')">{{ isTesting(selectedProvider, 'accounts') ? '测试中' : '测试 API Key' }}</button>
          <button class="sp-button" type="button" :disabled="isTesting(selectedProvider, 'groups')" @click="testProviderEndpointData(selectedProvider, 'groups')">{{ isTesting(selectedProvider, 'groups') ? '测试中' : '测试分组' }}</button>
          <button class="sp-button" type="button" :disabled="isTesting(selectedProvider, 'balance')" @click="testProviderEndpointData(selectedProvider, 'balance')">{{ isTesting(selectedProvider, 'balance') ? '测试中' : '测试余额' }}</button>
          <button class="sp-button" type="button" :disabled="isTesting(selectedProvider, 'cost')" @click="testProviderEndpointData(selectedProvider, 'cost')">{{ isTesting(selectedProvider, 'cost') ? '测试中' : '测试成本' }}</button>
          <button class="sp-button" type="button" :disabled="selectedProvider.is_default" @click="makeDefault(selectedProvider)">设为默认</button>
        </div>
        <div class="sp-timeline">
          <h4>接口配置</h4>
          <div class="sp-event"><b>基础地址</b><p>{{ selectedProvider.base_url }}</p></div>
          <div class="sp-event"><b>登录接口</b><p>{{ selectedProvider.login_url || '未配置' }}</p></div>
          <div class="sp-event"><b>API Key 接口</b><p>{{ selectedProvider.api_keys_url || '未配置' }}</p></div>
          <div class="sp-event"><b>同步状态</b><p>{{ selectedProvider.sync_message || syncText(selectedProvider) }}</p></div>
        </div>
      </template>
    </SupplierDrawer>

    <BaseDialog
      :show="authDialogVisible"
      :title="authProvider ? `${authProvider.name} · 登录与 Token` : '登录与 Token'"
      width="wide"
      @close="closeAuthHistory"
    >
      <div v-if="authLoading && !authStatus" class="sp-alert">正在读取认证缓存与历史记录…</div>
      <template v-else-if="authStatus">
        <div class="sp-detail-grid">
          <div class="sp-detail-cell"><span>登录总次数</span><b>{{ authStatus.summary.login_count }}</b></div>
          <div class="sp-detail-cell"><span>成功 / 失败</span><b>{{ authStatus.summary.login_success_count }} / {{ authStatus.summary.login_failure_count }}</b></div>
          <div class="sp-detail-cell"><span>最近登录</span><b>{{ formatAuthTime(authStatus.summary.last_login_at) }}</b></div>
          <div class="sp-detail-cell"><span>最近结果</span><b>{{ authEventLabel(authStatus.summary.last_login_status) }}</b></div>
          <div class="sp-detail-cell"><span>Token 缓存</span><b>{{ authCacheLabel(authStatus.cache.status) }}</b></div>
          <div class="sp-detail-cell"><span>剩余有效期</span><b>{{ formatDuration(authStatus.cache.remaining_seconds) }}</b></div>
          <div class="sp-detail-cell"><span>Redis TTL</span><b>{{ formatDuration(authStatus.cache.ttl_seconds) }}</b></div>
          <div class="sp-detail-cell"><span>Cookie 会话</span><b>{{ authStatus.cache.cookie_present ? '存在' : '不存在' }}</b></div>
          <div class="sp-detail-cell"><span>Token 摘要</span><b>{{ authStatus.cache.token_summary || '—' }}</b></div>
          <div class="sp-detail-cell"><span>Token 长度</span><b>{{ authStatus.cache.token_length || 0 }}</b></div>
          <div class="sp-detail-cell"><span>登录锁</span><b>{{ authStatus.login_lock.held ? `持有（${formatDuration(authStatus.login_lock.remaining_seconds)}）` : '未持有' }}</b></div>
          <div class="sp-detail-cell"><span>检查时间</span><b>{{ formatAuthTime(authStatus.checked_at) }}</b></div>
        </div>
        <div v-if="authStatus.cache.token_fingerprint" class="sp-alert">Token 指纹：{{ authStatus.cache.token_fingerprint }}</div>
        <div v-if="authStatus.summary.last_login_error || authStatus.summary.last_cache_error || authStatus.cache.error" class="sp-alert sp-error-line">
          {{ authStatus.summary.last_login_error || authStatus.summary.last_cache_error || authStatus.cache.error }}
        </div>
        <div class="sp-inline">
          <Select v-model="authEventFilter" class="w-40" :options="authEventOptions" data-test="supplier-auth-event-filter" @update:model-value="reloadAuthHistory" />
          <button class="sp-button" type="button" data-test="supplier-auth-refresh" :disabled="authLoading" @click="reloadAuthData">{{ authLoading ? '刷新中' : '刷新状态' }}</button>
        </div>
        <DataTable :columns="authHistoryColumns" :data="authHistory.items" :loading="authLoading" row-key="id">
          <template #cell-created_at="{ row }">{{ formatAuthTime(row.created_at) }}</template>
          <template #cell-event_type="{ row }">{{ authEventLabel(row.event_type) }}</template>
          <template #cell-source="{ row }">{{ authSourceLabel(row.source) }}</template>
          <template #cell-status="{ row }"><span class="sp-status" :class="row.status === 'success' ? 'good' : row.status === 'failed' || row.status === 'unavailable' ? 'bad' : 'warn'">{{ authEventLabel(row.status) }}</span></template>
          <template #cell-duration_ms="{ row }">{{ row.duration_ms }} ms</template>
          <template #cell-http_status="{ row }">{{ row.http_status || '—' }}</template>
          <template #cell-error_message="{ row }">{{ row.error_message || '—' }}</template>
          <template #empty>暂无认证历史</template>
        </DataTable>
        <div class="sp-inline">
          <button class="sp-button small" type="button" data-test="supplier-auth-prev" :disabled="authHistory.page <= 1 || authLoading" @click="changeAuthPage(-1)">上一页</button>
          <span>第 {{ authHistory.page }} 页 · 共 {{ authHistory.total }} 条</span>
          <button class="sp-button small" type="button" data-test="supplier-auth-next" :disabled="authHistory.page * authHistory.page_size >= authHistory.total || authLoading" @click="changeAuthPage(1)">下一页</button>
        </div>
      </template>
      <template #footer><button class="sp-button primary" type="button" @click="closeAuthHistory">关闭</button></template>
    </BaseDialog>

    <BaseDialog
      :show="modalVisible"
      :title="editingProvider ? '编辑供应商' : '新增供应商'"
      width="wide"
      @close="closeModal"
    >
      <form class="sp-provider-dialog" @submit.prevent="submitProvider">
        <div class="sp-dialog-summary" aria-label="供应商配置摘要">
          <div><span>操作类型</span><strong>{{ editingProvider ? '编辑配置' : '新增配置' }}</strong></div>
          <div><span>供应商类型</span><strong>{{ form.provider_type || '待选择' }}</strong></div>
          <div><span>运行状态</span><strong>{{ form.enabled ? '已启用' : '已停用' }}</strong></div>
        </div>

        <section class="sp-dialog-section">
          <div class="sp-dialog-section-head">
            <span>01</span>
            <div><h4>基础身份</h4><p>定义供应商名称、唯一编码和接口模板类型。</p></div>
          </div>
          <div class="sp-dialog-grid sp-dialog-grid-3">
            <Input v-model="form.name" label="供应商名称" required />
            <Input v-model="form.code" label="供应商编码" required :disabled="Boolean(editingProvider)" />
            <label class="sp-select-field">
              <span>供应商类型</span>
              <Select
                v-model="form.provider_type"
                :options="providerTypeOptions"
                placeholder="请选择供应商类型"
                :searchable="false"
                @change="applySelectedTypeTemplate(true)"
              />
            </label>
          </div>
        </section>

        <section class="sp-dialog-section">
          <div class="sp-dialog-section-head">
            <span>02</span>
            <div><h4>接口模板</h4><p>配置登录、账号、分组、余额和成本数据的访问地址。</p></div>
          </div>
          <div class="sp-dialog-grid sp-dialog-grid-2">
            <Input v-model="form.base_url" label="基础地址" required placeholder="https://supplier.example.com" />
            <Input v-model="form.login_url" label="登录接口" placeholder="https://supplier.example.com/api/v1/auth/login" />
            <Input v-model="form.api_keys_url" label="API Key 接口" placeholder="https://supplier.example.com/api/admin/keys" />
            <Input v-model="form.groups_url" label="分组接口" />
            <Input v-model="form.balance_url" label="余额接口" />
            <Input v-model="form.usage_cost_url" label="成本接口" />
          </div>
          <div class="sp-dialog-note">切换类型会用类型模板覆盖接口字段；覆盖后仍可继续手动编辑。</div>
        </section>

        <section class="sp-dialog-section">
          <div class="sp-dialog-section-head">
            <span>03</span>
            <div><h4>认证与运行策略</h4><p>补充登录凭据、账号命名和调度保护参数。</p></div>
          </div>
          <div class="sp-dialog-grid sp-dialog-grid-3">
            <Input v-if="form.provider_type === 'sub2api'" v-model="form.email" label="登录邮箱" />
            <Input v-else v-model="form.username" label="登录用户名" />
            <Input v-model="form.password" type="password" label="登录密码" :placeholder="editingProvider ? '留空则保留原密码' : ''" />
            <Input v-model="form.account_name_prefix" label="账号名前缀" />
            <Input :model-value="form.temp_disable_minutes" type="number" label="临时禁用分钟" @update:model-value="form.temp_disable_minutes = toNumber($event, form.temp_disable_minutes ?? 0)" />
            <Input :model-value="form.account_rate_multiplier_scale" type="number" label="倍率缩放" @update:model-value="form.account_rate_multiplier_scale = toNumber($event, form.account_rate_multiplier_scale)" />
            <Input :model-value="form.sort_order" type="number" label="排序" @update:model-value="form.sort_order = toNumber($event, form.sort_order ?? 0)" />
            <label class="sp-toggle-field sp-dialog-toggle-card">
              <span>启用供应商</span>
              <div class="sp-toggle-row">
                <Toggle v-model="form.enabled" />
                <em>{{ form.enabled ? '已启用' : '已停用' }}</em>
              </div>
            </label>
            <label class="sp-toggle-field sp-dialog-toggle-card">
              <span>启用 Turnstile 人机校验</span>
              <div class="sp-toggle-row">
                <Toggle v-model="form.turnstile_enabled" />
                <em>{{ form.turnstile_enabled ? '开启' : '关闭' }}</em>
              </div>
              <p class="sp-toggle-hint">开启后，登录该上游前会调用 2Captcha 自动求解验证码；失败将导致登录失败</p>
            </label>
            <label class="sp-toggle-field sp-dialog-toggle-card">
              <span>设为默认供应商</span>
              <div class="sp-toggle-row">
                <Toggle :model-value="Boolean(form.is_default)" @update:model-value="form.is_default = $event" />
                <em>{{ form.is_default ? '默认' : '非默认' }}</em>
              </div>
            </label>
          </div>
        </section>
      </form>
      <template #footer>
        <button class="sp-button ghost" type="button" @click="closeModal">取消</button>
        <button class="sp-button primary" type="button" @click="submitProvider">保存供应商</button>
      </template>
    </BaseDialog>

    <BaseDialog
      :show="createTypeVisible"
      title="新增供应商类型"
      width="wide"
      @close="closeCreateProviderType"
    >
      <form class="sp-type-create-dialog" @submit.prevent="submitNewProviderType">
        <div class="sp-dialog-summary" aria-label="新增供应商类型摘要">
          <div><span>配置用途</span><strong>接口模板</strong></div>
          <div><span>类型编码</span><strong>{{ typeForm.code || '待填写' }}</strong></div>
          <div><span>默认状态</span><strong>{{ typeForm.enabled ? '启用' : '停用' }}</strong></div>
        </div>

        <section class="sp-dialog-section">
          <div class="sp-dialog-section-head">
            <span>01</span>
            <div><h4>类型信息</h4><p>名称用于界面识别，编码用于供应商配置关联。</p></div>
          </div>
          <div class="sp-dialog-grid sp-dialog-grid-3">
            <Input v-model="typeForm.name" label="供应商类型" required placeholder="Sub2API" />
            <Input v-model="typeForm.code" label="类型编码" required placeholder="sub2api" />
            <Input :model-value="typeForm.sort_order" type="number" label="排序" @update:model-value="typeForm.sort_order = toNumber($event, typeForm.sort_order ?? 0)" />
          </div>
        </section>

        <section class="sp-dialog-section">
          <div class="sp-dialog-section-head">
            <span>02</span>
            <div><h4>接口模板</h4><p>新增供应商选择该类型时，可自动带入这些接口地址。</p></div>
          </div>
          <div class="sp-dialog-grid sp-dialog-grid-2">
            <Input v-model="typeForm.login_url" label="登录接口" placeholder="https://supplier.example.com/api/v1/auth/login" />
            <Input v-model="typeForm.api_keys_url" label="API Key 接口" />
            <Input v-model="typeForm.groups_url" label="分组接口" />
            <Input v-model="typeForm.balance_url" label="余额接口" />
            <Input v-model="typeForm.usage_cost_url" label="成本接口" />
            <label class="sp-toggle-field sp-dialog-toggle-card">
              <span>启用类型</span>
              <div class="sp-toggle-row">
                <Toggle v-model="typeForm.enabled" />
                <em>{{ typeForm.enabled ? '启用' : '停用' }}</em>
              </div>
            </label>
          </div>
          <div class="sp-dialog-note">供应商自身接口字段为空时，后台会使用这里的类型模板。</div>
        </section>
      </form>
      <template #footer>
        <button class="sp-button ghost" type="button" @click="closeCreateProviderType">取消</button>
        <button class="sp-button primary" type="button" @click="submitNewProviderType">创建类型</button>
      </template>
    </BaseDialog>

    <BaseDialog
      :show="typeManagerVisible"
      title="供应商类型维护"
      width="wide"
      @close="closeTypeManager"
    >
      <div class="sp-type-manager-dialog">
        <aside class="sp-type-list" aria-label="供应商类型列表">
          <div class="sp-type-list-head">
            <div><span>Type Index</span><strong>已有类型</strong></div>
            <em>{{ providerTypes.length }} 项</em>
          </div>
          <button
            v-for="type in providerTypes"
            :key="type.id"
            class="sp-type-row"
            :class="{ active: editingProviderType?.id === type.id }"
            type="button"
            :aria-pressed="editingProviderType?.id === type.id"
            @click="editProviderType(type)"
          >
            <span><b>{{ type.name }}</b><small>{{ type.code }}</small></span>
            <em :class="type.enabled ? 'good' : 'warn'">{{ type.enabled ? '启用' : '停用' }}</em>
          </button>
          <div v-if="!providerTypes.length" class="sp-type-empty">暂无供应商类型，请从页面顶部新增。</div>
        </aside>

        <form class="sp-type-editor" @submit.prevent="submitProviderType">
          <div class="sp-dialog-summary" aria-label="当前类型摘要">
            <div><span>当前类型</span><strong>{{ editingProviderType?.name || '新类型' }}</strong></div>
            <div><span>类型编码</span><strong>{{ typeForm.code || '待填写' }}</strong></div>
            <div><span>当前状态</span><strong>{{ typeForm.enabled ? '启用' : '停用' }}</strong></div>
          </div>
          <section class="sp-dialog-section">
            <div class="sp-dialog-section-head">
              <span>01</span>
              <div><h4>基础配置</h4><p>维护类型名称、编码、排序和启用状态。</p></div>
            </div>
            <div class="sp-dialog-grid sp-dialog-grid-3">
              <Input v-model="typeForm.name" label="供应商类型" required placeholder="Sub2API" />
              <Input v-model="typeForm.code" label="类型编码" required placeholder="sub2api" :disabled="Boolean(editingProviderType)" />
              <Input :model-value="typeForm.sort_order" type="number" label="排序" @update:model-value="typeForm.sort_order = toNumber($event, typeForm.sort_order ?? 0)" />
            </div>
          </section>
          <section class="sp-dialog-section">
            <div class="sp-dialog-section-head">
              <span>02</span>
              <div><h4>接口模板</h4><p>供应商未单独配置接口时使用该模板。</p></div>
            </div>
            <div class="sp-dialog-grid sp-dialog-grid-2">
              <Input v-model="typeForm.login_url" label="登录接口" placeholder="https://supplier.example.com/api/v1/auth/login" />
              <Input v-model="typeForm.api_keys_url" label="API Key 接口" />
              <Input v-model="typeForm.groups_url" label="分组接口" />
              <Input v-model="typeForm.balance_url" label="余额接口" />
              <Input v-model="typeForm.usage_cost_url" label="成本接口" />
              <label class="sp-toggle-field sp-dialog-toggle-card">
                <span>启用类型</span>
                <div class="sp-toggle-row">
                  <Toggle v-model="typeForm.enabled" />
                  <em>{{ typeForm.enabled ? '启用' : '停用' }}</em>
                </div>
              </label>
            </div>
            <div class="sp-dialog-note">这些接口作为供应商模板使用；供应商自身字段为空时后台会使用这里的配置。</div>
          </section>
          <div class="sp-dialog-danger-zone">
            <div><strong>删除供应商类型</strong><span>删除前请确认没有供应商继续引用该类型。</span></div>
            <button v-if="editingProviderType" class="sp-button danger" type="button" @click="removeProviderType(editingProviderType)">删除当前类型</button>
          </div>
        </form>
      </div>
      <template #footer>
        <button class="sp-button ghost" type="button" @click="closeTypeManager">关闭</button>
        <button class="sp-button primary" type="button" :disabled="!editingProviderType" @click="submitProviderType">保存修改</button>
      </template>
    </BaseDialog>

    <BaseDialog
      :show="testResultVisible"
      title="接口测试结果"
      width="extra-wide"
      @close="closeTestResult"
    >
      <div class="sp-test-dialog">
        <div v-if="testResult" class="sp-test-result" :class="{ bad: Boolean(testResult.error || testResult.parse_error) }">
          <div class="sp-dialog-summary" aria-label="接口测试摘要">
            <div><span>测试接口</span><strong>{{ scopeLabel(testResult.scope) }}</strong></div>
            <div><span>HTTP 状态</span><strong>{{ testResult.http_status || '无' }}</strong></div>
            <div><span>响应耗时</span><strong>{{ testResult.duration_ms }} ms</strong></div>
            <div><span>响应大小</span><strong>{{ testResult.response_bytes }} bytes</strong></div>
          </div>
          <div v-if="testResult.error" class="sp-alert sp-error-line">请求错误：{{ testResult.error }}</div>
          <div v-if="testResult.parse_error" class="sp-alert sp-error-line">解析错误：{{ testResult.parse_error }}</div>

          <section class="sp-dialog-section">
            <div class="sp-dialog-section-head">
              <span>01</span>
              <div><h4>调用尝试</h4><p>按照实际请求顺序展示端点、状态和耗时。</p></div>
            </div>
            <div class="sp-test-attempts">
              <article v-for="(attempt, index) in testResult.attempts" :key="`${attempt.endpoint}:${index}`" class="sp-test-attempt">
                <div><span>{{ String(index + 1).padStart(2, '0') }}</span><strong>{{ attempt.endpoint }}</strong></div>
                <p>HTTP {{ attempt.http_status || '无' }} · {{ attempt.duration_ms }} ms · {{ attempt.response_bytes }} bytes</p>
                <p v-if="attempt.error" class="bad">请求错误：{{ attempt.error }}</p>
                <p v-if="attempt.parse_error" class="bad">解析错误：{{ attempt.parse_error }}</p>
              </article>
            </div>
          </section>

          <section class="sp-dialog-section">
            <div class="sp-dialog-section-head">
              <span>02</span>
              <div><h4>响应内容</h4><p>对照脱敏原始返回与前端解析结果。</p></div>
            </div>
            <div class="sp-test-response-grid">
              <div class="sp-response-panel">
                <div class="sp-response-panel-head"><strong>脱敏原始返回</strong><span>Raw Response</span></div>
                <pre class="sp-message-detail">{{ testResult.response_summary || '无返回内容' }}</pre>
              </div>
              <div class="sp-response-panel">
                <div class="sp-response-panel-head"><strong>解析结果</strong><span>Parsed Data</span></div>
                <pre class="sp-message-detail">{{ formatDiagnosticJSON(testResult.parsed_data) }}</pre>
              </div>
            </div>
          </section>
          <div class="sp-dialog-note">敏感字段已脱敏；该测试只调用接口，不会写入同步记录或本地数据表。</div>
        </div>
      </div>
      <template #footer>
        <button class="sp-button primary" type="button" @click="closeTestResult">关闭</button>
      </template>
    </BaseDialog>

  </SupplierModuleLayout>
</template>

<script setup lang="ts">
import { computed, onMounted, reactive, ref, watch } from 'vue'
import {
  Chart as ChartJS,
  CategoryScale,
  LinearScale,
  PointElement,
  LineElement,
  BarElement,
  Filler,
  Tooltip,
  Legend,
  type ChartOptions,
  type TooltipItem,
} from 'chart.js'
import { Bar, Line } from 'vue-chartjs'
import { SupplierDrawer, SupplierModuleLayout } from '@/components/admin/supplier-management'
import BaseDialog from '@/components/common/BaseDialog.vue'
import DataTable from '@/components/common/DataTable.vue'
import DateRangePicker from '@/components/common/DateRangePicker.vue'
import Icon from '@/components/icons/Icon.vue'
import Input from '@/components/common/Input.vue'
import Select, { type SelectOption } from '@/components/common/Select.vue'
import Toggle from '@/components/common/Toggle.vue'
import supplierProvidersAPI, { type SupplierProvider, type SupplierProviderSummary, type SupplierProviderUpsertPayload, type SupplierProviderCostTrendPoint, type SupplierProviderCostBreakdown, type SupplierProviderCostBackfillResult, type SupplierProviderAuthStatusResult, type SupplierProviderAuthHistoryResult, type SupplierProviderAuthEventType } from '@/api/admin/supplierProviders'
import supplierProviderTypesAPI, { type SupplierProviderType, type SupplierProviderTypeUpsertPayload } from '@/api/admin/supplierProviderTypes'
import { syncProvider, testProviderEndpoint, type SupplierProviderEndpointTestResult, type SupplierSyncScope } from '@/api/admin/supplierProviderData'
import { useAppStore } from '@/stores/app'
import type { Column } from '@/components/common/types'

ChartJS.register(CategoryScale, LinearScale, PointElement, LineElement, BarElement, Filler, Tooltip, Legend)

type Tone = 'good' | 'warn' | 'bad' | 'info' | ''
type ProviderQuickFilter = 'all' | 'enabled' | 'disabled' | 'default'

type HealthTodo = {
  key: string
  title: string
  detail: string
  tone: Tone
  filter?: string
  quickFilter?: ProviderQuickFilter
  selectId?: number
}

const providerQuickFilters: Array<{ key: ProviderQuickFilter; label: string }> = [
  { key: 'all', label: '全部' },
  { key: 'enabled', label: '已启用' },
  { key: 'disabled', label: '已停用' },
  { key: 'default', label: '默认' },
]
type SupplierDiagnosticScope = Exclude<SupplierSyncScope, 'all'>

const emptySummary = (): SupplierProviderSummary => ({
  total_count: 0,
  enabled_count: 0,
  high_risk_count: 0,
  low_balance_count: 0,
  sync_failure_count: 0,
  rate_risk_count: 0,
})

const emptyForm = (): SupplierProviderUpsertPayload => ({
  code: '',
  name: '',
  provider_type: 'sub2api',
  base_url: '',
  login_url: '',
  api_keys_url: '',
  groups_url: '',
  available_groups_url: '',
  balance_url: '',
  usage_cost_url: '',
  email: '',
  username: '',
  password: '',
  account_name_prefix: '',
  temp_disable_minutes: 0,
  account_rate_multiplier_scale: 1,
  sort_order: 0,
  enabled: true,
  turnstile_enabled: false,
  is_default: false,
})

const emptyTypeForm = (): SupplierProviderTypeUpsertPayload => ({
  code: '',
  name: '',
  login_url: '',
  api_keys_url: '',
  groups_url: '',
  available_groups_url: '',
  balance_url: '',
  usage_cost_url: '',
  enabled: true,
  sort_order: 0,
})


function formatLocalDate(date: Date): string {
  const year = date.getFullYear()
  const month = String(date.getMonth() + 1).padStart(2, '0')
  const day = String(date.getDate()).padStart(2, '0')
  return `${year}-${month}-${day}`
}

function createDefaultCostTrendRange() {
  const end = new Date()
  const start = new Date()
  // 默认近 14 天（含今天）
  start.setDate(end.getDate() - 13)
  return {
    start: formatLocalDate(start),
    end: formatLocalDate(end),
  }
}

const providers = ref<SupplierProvider[]>([])
const providerTypes = ref<SupplierProviderType[]>([])
const summary = ref<SupplierProviderSummary>(emptySummary())
const costTrendDefaultRange = createDefaultCostTrendRange()
const costTrendStartDate = ref(costTrendDefaultRange.start)
const costTrendEndDate = ref(costTrendDefaultRange.end)
const costBreakdownDefaultRange = createDefaultCostTrendRange()
const costBreakdownStartDate = ref(costBreakdownDefaultRange.start)
const costBreakdownEndDate = ref(costBreakdownDefaultRange.end)
const costTrendProviderId = ref<number | ''>('')
const costTrendLoading = ref(false)
const costBreakdownLoading = ref(false)
const costTrendPoints = ref<SupplierProviderCostTrendPoint[]>([])
const costTrendBreakdown = ref<SupplierProviderCostBreakdown[]>([])
const deviationThresholdOptions: SelectOption[] = Array.from({ length: 31 }, (_, value) => ({
  value: value / 100,
  label: `${value}%`,
}))
const deviationThreshold = ref(0.15) // 15%
const search = ref('')
const providerQuickFilter = ref<ProviderQuickFilter>('all')
const filter = ref('all')
const providerSortKey = ref('')
const providerSortOrder = ref<'asc' | 'desc'>('asc')
const loading = ref(false)
const error = ref('')
const selectedProvider = ref<SupplierProvider | null>(null)
const authDialogVisible = ref(false)
const authProvider = ref<SupplierProvider | null>(null)
const authStatus = ref<SupplierProviderAuthStatusResult | null>(null)
const authHistory = ref<SupplierProviderAuthHistoryResult>({ items: [], total: 0, page: 1, page_size: 20 })
const authEventFilter = ref<SupplierProviderAuthEventType | ''>('')
const authLoading = ref(false)
const editingProvider = ref<SupplierProvider | null>(null)
const editingProviderType = ref<SupplierProviderType | null>(null)
const modalVisible = ref(false)
const typeManagerVisible = ref(false)
const createTypeVisible = ref(false)
const form = reactive<SupplierProviderUpsertPayload>(emptyForm())
const typeForm = reactive<SupplierProviderTypeUpsertPayload>(emptyTypeForm())
const syncingKeys = ref<Set<string>>(new Set())
const testingKeys = ref<Set<string>>(new Set())
const updatingProviderIDs = ref<Set<number>>(new Set())
const testResultVisible = ref(false)
const testResult = ref<SupplierProviderEndpointTestResult | null>(null)
let searchTimer: number | undefined
const appStore = useAppStore()

const enabledProviderTypes = computed(() => providerTypes.value.filter(type => type.enabled))
const providerTypeOptions = computed<SelectOption[]>(() =>
  enabledProviderTypes.value.map(type => ({ value: type.code, label: `${type.name}（${type.code}）` }))
)
const providerColumns: Column[] = [
  { key: 'homepage', label: '主页', class: 'w-[64px]' },
  { key: 'name', label: '供应商', sortable: true, class: 'min-w-[220px]' },
  { key: 'status', label: '运行状态', sortable: true },
  { key: 'account_counts', label: '有效 / 可调度账号', sortable: true, class: 'min-w-[150px]' },
  { key: 'success_rate', label: '成功率', sortable: true },
  { key: 'today_cost', label: '今日成本', sortable: true },
  { key: 'current_balance', label: '余额可用', sortable: true },
  { key: 'rate_risk_count', label: '倍率风险', sortable: true },
  { key: 'credential_configured', label: '凭据', sortable: true },
  { key: 'last_sync_at', label: '最近同步', sortable: true, class: 'min-w-[110px]' },
  { key: 'auth_summary', label: '登录认证', sortable: true, class: 'min-w-[130px]' },
  { key: 'actions', label: '操作', class: 'min-w-[330px]' },
]

const authHistoryColumns: Column[] = [
  { key: 'created_at', label: '时间', class: 'min-w-[150px]' },
  { key: 'event_type', label: '事件' },
  { key: 'source', label: '来源' },
  { key: 'status', label: '结果' },
  { key: 'duration_ms', label: '耗时' },
  { key: 'http_status', label: 'HTTP' },
  { key: 'error_message', label: '错误摘要', class: 'min-w-[220px]' },
]

const authEventOptions: SelectOption[] = [
  { value: '', label: '全部事件' },
  { value: 'login_success', label: '登录成功' },
  { value: 'login_failed', label: '登录失败' },
  { value: 'cache_hit', label: '缓存命中' },
  { value: 'cache_miss', label: '缓存未命中' },
  { value: 'cache_invalidated', label: '缓存失效' },
  { value: 'cache_error', label: '缓存异常' },
]

const metrics = computed(() => [
  { key: 'all', tone: 'green', label: '启用供应商', value: String(summary.value.enabled_count), foot: `共管理 ${summary.value.total_count} 个供应商` },
  { key: 'risk', tone: 'red', label: '高风险供应商', value: String(summary.value.high_risk_count), foot: '风险等级为 high 或 critical' },
  { key: 'balance', tone: 'orange', label: '余额不足 3 天', value: String(summary.value.low_balance_count), foot: '按预计可用天数判断' },
  { key: 'sync', tone: 'blue', label: '同步异常', value: String(summary.value.sync_failure_count), foot: '最近同步状态失败' },
  { key: 'rate', tone: 'amber', label: '倍率风险项', value: String(summary.value.rate_risk_count), foot: '供应商账号倍率风险累计' },
])

const filteredProviders = computed(() => providers.value.filter(provider => {
  if (providerQuickFilter.value === 'enabled' && !provider.enabled) return false
  if (providerQuickFilter.value === 'disabled' && provider.enabled) return false
  if (providerQuickFilter.value === 'default' && !provider.is_default) return false
  if (filter.value === 'risk' && !['high', 'critical'].includes(provider.risk_level)) return false
  if (filter.value === 'balance' && !isLowBalance(provider)) return false
  if (filter.value === 'sync' && provider.sync_status !== 'failed') return false
  if (filter.value === 'rate' && provider.rate_risk_count <= 0) return false
  return true
}))

const sortedProviders = computed(() => {
  const rows = [...filteredProviders.value]
  if (!providerSortKey.value) {
    return rows.sort((left, right) => riskWeight(right) - riskWeight(left) || left.sort_order - right.sort_order || left.id - right.id)
  }

  return rows.sort((left, right) => compareProviders(left, right, providerSortKey.value, providerSortOrder.value))
})

const defaultProvider = computed(() => providers.value.find(provider => provider.is_default) || null)
const credentialMissingCount = computed(() => providers.value.filter(provider => !provider.credential_configured).length)
const credentialCoverage = computed(() => `${providers.value.length - credentialMissingCount.value} / ${providers.value.length} 个供应商已配置凭据`)
const zeroSchedulableCount = computed(() => providers.value.filter(provider => provider.enabled && Number(provider.schedulable_account_count || 0) <= 0).length)

const healthTone = computed<Tone>(() => {
  if (!providers.value.length) return 'warn'
  if (summary.value.high_risk_count > 0) return 'bad'
  if (
    summary.value.sync_failure_count > 0
    || summary.value.low_balance_count > 0
    || !defaultProvider.value
    || credentialMissingCount.value
    || summary.value.rate_risk_count
    || zeroSchedulableCount.value
  ) return 'warn'
  return 'good'
})

const healthLabel = computed(() => {
  if (healthTone.value === 'bad') return '告警'
  if (healthTone.value === 'warn') return '需关注'
  return '稳定'
})

const healthMessage = computed(() => {
  if (!providers.value.length) return '还没有供应商数据，请先新增供应商配置。'
  if (summary.value.high_risk_count) return `当前有 ${summary.value.high_risk_count} 个高风险供应商，应优先检查凭据、余额和同步结果。`
  if (summary.value.sync_failure_count) return `当前有 ${summary.value.sync_failure_count} 个供应商同步异常，需要查看同步日志。`
  if (summary.value.low_balance_count) return `有 ${summary.value.low_balance_count} 个供应商预计可用不足 3 天，建议及时补充余额。`
  if (!defaultProvider.value) return '尚未设置默认供应商，新建账号时缺少兜底来源。'
  if (credentialMissingCount.value) return `还有 ${credentialMissingCount.value} 个供应商未配置登录凭据，同步与测试会受影响。`
  if (zeroSchedulableCount.value) return `有 ${zeroSchedulableCount.value} 个已启用供应商当前 0 可调度账号，实际无法承接流量。`
  if (summary.value.rate_risk_count) return `累计 ${summary.value.rate_risk_count} 个倍率风险项，建议核对账号倍率配置。`
  return '供应商组合运行平稳，配置完整，暂无高优先级风险。'
})

const priorityTodos = computed<HealthTodo[]>(() => {
  const todos: HealthTodo[] = []
  if (summary.value.high_risk_count > 0) {
    todos.push({
      key: 'high-risk',
      title: '高风险供应商',
      detail: `当前有 ${summary.value.high_risk_count} 个高风险供应商，应优先检查凭据、余额和同步结果。`,
      tone: 'bad',
      filter: 'risk',
    })
  }
  if (summary.value.sync_failure_count > 0) {
    todos.push({
      key: 'sync-failed',
      title: '同步异常',
      detail: `当前有 ${summary.value.sync_failure_count} 个供应商同步异常，需要查看同步日志。`,
      tone: 'warn',
      filter: 'sync',
    })
  }
  if (summary.value.low_balance_count > 0) {
    todos.push({
      key: 'low-balance',
      title: '余额不足 3 天',
      detail: `有 ${summary.value.low_balance_count} 个供应商预计可用不足 3 天，建议及时补充余额。`,
      tone: 'warn',
      filter: 'balance',
    })
  }
  if (!defaultProvider.value) {
    todos.push({
      key: 'no-default',
      title: '缺少默认供应商',
      detail: '尚未设置默认供应商，新建账号时缺少兜底来源。',
      tone: 'warn',
      quickFilter: 'default',
    })
  }
  if (credentialMissingCount.value > 0) {
    todos.push({
      key: 'missing-credential',
      title: '凭据未配置',
      detail: `还有 ${credentialMissingCount.value} 个供应商未配置登录凭据。`,
      tone: 'warn',
    })
  }
  if (zeroSchedulableCount.value > 0) {
    todos.push({
      key: 'zero-schedulable',
      title: '零可调度账号',
      detail: `有 ${zeroSchedulableCount.value} 个已启用供应商当前 0 可调度账号。`,
      tone: 'warn',
      quickFilter: 'enabled',
    })
  }
  if (summary.value.rate_risk_count > 0) {
    todos.push({
      key: 'rate-risk',
      title: '倍率风险',
      detail: `累计 ${summary.value.rate_risk_count} 个倍率风险项，建议核对账号倍率配置。`,
      tone: 'warn',
      filter: 'rate',
    })
  }
  return todos.slice(0, 3)
})

const costTrendProviderOptions = computed<SelectOption[]>(() => [
  { value: '', label: '全部供应商' },
  ...providers.value.map(provider => ({
    value: provider.id,
    label: provider.name,
  })),
])

const costTrendScopeLabel = computed(() => {
  if (!costTrendProviderId.value) return '全部供应商'
  const matched = providers.value.find(provider => provider.id === costTrendProviderId.value)
  return matched ? matched.name : '指定供应商'
})

const costTrendRangeLabel = computed(() => {
  const start = costTrendStartDate.value
  const end = costTrendEndDate.value
  if (!start || !end) return '未选择时间'
  if (start === end) return start
  return `${start} ~ ${end}`
})

const costBreakdownRangeLabel = computed(() => {
  const start = costBreakdownStartDate.value
  const end = costBreakdownEndDate.value
  if (!start || !end) return '未选择时间'
  if (start === end) return start
  return `${start} ~ ${end}`
})

const costTrendTotals = computed(() =>
  costTrendPoints.value.reduce((acc, point) => {
    acc.upstream += Number(point.upstream_cost || 0)
    acc.local += Number(point.local_cost || 0)
    return acc
  }, { upstream: 0, local: 0 }),
)

const costBreakdown = computed(() =>
  costTrendBreakdown.value.map(provider => ({
    id: provider.provider_id,
    name: provider.provider_name,
    upstreamCost: Number(provider.upstream_cost || 0),
    localCost: Number(provider.local_cost || 0),
  })),
)

const costTrendChartData = computed(() => {
  if (!costTrendPoints.value.length) return null
  const hasValue = costTrendPoints.value.some(point => Number(point.upstream_cost || 0) > 0 || Number(point.local_cost || 0) > 0)
  if (!hasValue) return null
  return {
    labels: costTrendPoints.value.map(point => formatCostTrendLabel(point.date)),
    datasets: [
      {
        label: '上游成本',
        data: costTrendPoints.value.map(point => Number(point.upstream_cost || 0)),
        borderColor: '#3b82f6',
        backgroundColor: 'rgba(59, 130, 246, 0.12)',
        fill: true,
        tension: 0.35,
      },
      {
        label: '本地成本',
        data: costTrendPoints.value.map(point => Number(point.local_cost || 0)),
        borderColor: '#d97706',
        backgroundColor: 'rgba(217, 119, 6, 0.10)',
        fill: true,
        tension: 0.35,
      },
    ],
  }
})

const costTrendChartOptions = computed<ChartOptions<'line'>>(() => {
  const isDark = typeof document !== 'undefined' && document.documentElement.classList.contains('dark')
  return {
    responsive: true,
    maintainAspectRatio: false,
    interaction: { mode: 'index' as const, intersect: false },
    plugins: {
      legend: { display: false },
      tooltip: {
        backgroundColor: isDark ? '#1f2937' : '#ffffff',
        titleColor: isDark ? '#f9fafb' : '#111827',
        bodyColor: isDark ? '#d1d5db' : '#334155',
        borderColor: isDark ? '#374151' : '#e2e8f0',
        borderWidth: 1,
        callbacks: {
          label(context: TooltipItem<'line'>) {
            const label = context.dataset.label || ''
            const value = Number(context.parsed.y ?? 0)
            return `${label}: ${currency(value)}`
          },
        },
      },
    },
    scales: {
      x: {
        ticks: { color: isDark ? '#9ca3af' : '#64748b' },
        grid: { display: false },
      },
      y: {
        beginAtZero: true,
        ticks: {
          color: isDark ? '#9ca3af' : '#64748b',
          callback(value: string | number) {
            const num = Number(value)
            if (Math.abs(num) >= 1000) return `¥${(num / 1000).toFixed(1)}k`
            return `¥${num}`
          },
        },
        grid: { display: false },
      },
    },
  }
})

const costBreakdownChartData = computed(() => {
  if (!costBreakdown.value.length) return null
  const hasValue = costBreakdown.value.some(item => item.upstreamCost > 0 || item.localCost > 0)
  if (!hasValue) return null

  return {
    labels: costBreakdown.value.map(item => item.name),
    datasets: [
      {
        label: '上游成本',
        data: costBreakdown.value.map(item => item.upstreamCost),
        backgroundColor: '#3b82f6',
        borderColor: '#2563eb',
        borderWidth: 1,
        borderRadius: 4,
        borderSkipped: false,
        barPercentage: 0.36,
        categoryPercentage: 0.76,
        maxBarThickness: 24,
      },
      {
        label: '本地成本',
        data: costBreakdown.value.map(item => item.localCost),
        backgroundColor: '#d97706',
        borderColor: '#b45309',
        borderWidth: 1,
        borderRadius: 4,
        borderSkipped: false,
        barPercentage: 0.36,
        categoryPercentage: 0.76,
        maxBarThickness: 24,
      },
    ],
  }
})

const costBreakdownChartOptions = computed<ChartOptions<'bar'>>(() => {
  const isDark = typeof document !== 'undefined' && document.documentElement.classList.contains('dark')
  return {
    responsive: true,
    maintainAspectRatio: false,
    interaction: { mode: 'index' as const, intersect: false },
    plugins: {
      legend: { display: false },
      tooltip: {
        backgroundColor: isDark ? '#1f2937' : '#ffffff',
        titleColor: isDark ? '#f9fafb' : '#111827',
        bodyColor: isDark ? '#d1d5db' : '#334155',
        borderColor: isDark ? '#374151' : '#e2e8f0',
        borderWidth: 1,
        callbacks: {
          label(context: TooltipItem<'bar'>) {
            const label = context.dataset.label || ''
            const value = Number(context.parsed.y ?? 0)
            return `${label}: ${currency(value)}`
          },
        },
      },
    },
    scales: {
      x: {
        ticks: {
          color: isDark ? '#9ca3af' : '#64748b',
          maxRotation: 45,
          minRotation: 0,
          autoSkip: false,
          callback(value: string | number) {
            const label = costBreakdown.value[Number(value)]?.name || ''
            return label.length > 12 ? `${label.slice(0, 12)}…` : label
          },
        },
        grid: { display: false },
      },
      y: {
        beginAtZero: true,
        ticks: {
          color: isDark ? '#9ca3af' : '#64748b',
          callback(value: string | number) {
            const num = Number(value)
            if (Math.abs(num) >= 1000) return `¥${(num / 1000).toFixed(1)}k`
            return `¥${num}`
          },
        },
        grid: { color: isDark ? 'rgba(148, 163, 184, 0.16)' : 'rgba(148, 163, 184, 0.22)' },
      },
    },
  }
})

watch(search, () => {
  window.clearTimeout(searchTimer)
  searchTimer = window.setTimeout(loadProviders, 350)
})

onMounted(async () => {
  await loadProviderTypes()
  await Promise.all([loadProviders(), loadCostTrends(), loadCostBreakdown()])
})

async function loadProviderTypes() {
  try {
    providerTypes.value = await supplierProviderTypesAPI.list()
  } catch (err) {
    error.value = errorMessage(err, '加载供应商类型失败')
  }
}

async function loadProviders() {
  loading.value = true
  error.value = ''
  try {
    const result = await supplierProvidersAPI.list({ search: search.value.trim(), page: 1, page_size: 100 })
    providers.value = result.items
    summary.value = result.summary
    if (selectedProvider.value) {
      selectedProvider.value = result.items.find(provider => provider.id === selectedProvider.value?.id) || null
    }
  } catch (err) {
    error.value = errorMessage(err, '加载供应商失败')
  } finally {
    loading.value = false
  }
}

async function loadCostTrends() {
  costTrendLoading.value = true
  try {
    const providerId = typeof costTrendProviderId.value === 'number' ? costTrendProviderId.value : 0
    const result = await supplierProvidersAPI.listCostTrends({
      start_date: costTrendStartDate.value,
      end_date: costTrendEndDate.value,
      provider_id: providerId || undefined,
    })
    costTrendPoints.value = Array.isArray(result?.points) ? result.points : []
  } catch {
    costTrendPoints.value = []
  } finally {
    costTrendLoading.value = false
  }
}

async function loadCostBreakdown() {
  if (costBreakdownLoading.value) return
  costBreakdownLoading.value = true
  try {
    const result = await supplierProvidersAPI.listCostTrends({
      start_date: costBreakdownStartDate.value,
      end_date: costBreakdownEndDate.value,
    })
    costTrendBreakdown.value = Array.isArray(result?.breakdown) ? result.breakdown : []
  } catch {
    costTrendBreakdown.value = []
  } finally {
    costBreakdownLoading.value = false
  }
}

async function onCostTrendDateRangeChange(range: { startDate: string; endDate: string; preset: string | null }) {
  costTrendStartDate.value = range.startDate
  costTrendEndDate.value = range.endDate
  await loadCostTrends()
}

async function onCostTrendProviderChange() {
  await loadCostTrends()
}

async function onCostBreakdownDateRangeChange(range: { startDate: string; endDate: string; preset: string | null }) {
  costBreakdownStartDate.value = range.startDate
  costBreakdownEndDate.value = range.endDate
  await loadCostBreakdown()
}

async function refreshCostTrends() {
  if (costTrendLoading.value) return
  costTrendLoading.value = true
  try {
    const providerId = typeof costTrendProviderId.value === 'number' ? costTrendProviderId.value : 0
    const backfill = await supplierProvidersAPI.backfillCostTrends({
      start_date: costTrendStartDate.value,
      end_date: costTrendEndDate.value,
      provider_id: providerId || undefined,
    })
    const result = await supplierProvidersAPI.listCostTrends({
      start_date: costTrendStartDate.value,
      end_date: costTrendEndDate.value,
      provider_id: providerId || undefined,
    })
    costTrendPoints.value = Array.isArray(result?.points) ? result.points : []
    notifyCostBackfillResult(backfill)
  } catch (err) {
    appStore.showError(errorMessage(err, '回补上游成本失败'))
  } finally {
    costTrendLoading.value = false
  }
}

function notifyCostBackfillResult(result: SupplierProviderCostBackfillResult) {
  const success = Number(result?.success_count || 0)
  const failed = Number(result?.failed_count || 0)
  const skipped = Number(result?.skipped_count || 0)
  const providers = Number(result?.provider_count || 0)
  const days = Number(result?.day_count || 0)
  const rangeText = `${result?.start_date || costTrendStartDate.value} ~ ${result?.end_date || costTrendEndDate.value}`
  const message = `上游成本回补完成（${rangeText}，${providers} 个供应商 × ${days} 天）：成功 ${success}，跳过 ${skipped}，失败 ${failed}`
  if (failed > 0 && success === 0) {
    appStore.showError(message)
    return
  }
  if (failed > 0 || skipped > 0) {
    appStore.showWarning(message)
    return
  }
  appStore.showSuccess(message)
}


function selectProviderForDetail(provider: SupplierProvider) {
  selectedProvider.value = provider
  // 点击列表行时同步成本对比范围，便于按供应商查看
  if (costTrendProviderId.value !== provider.id) {
    costTrendProviderId.value = provider.id
    void loadCostTrends()
  }
}

async function openAuthHistory(provider: SupplierProvider) {
  authProvider.value = provider
  authDialogVisible.value = true
  authEventFilter.value = ''
  authHistory.value = { items: [], total: 0, page: 1, page_size: 20 }
  authStatus.value = null
  await reloadAuthData()
}

function closeAuthHistory() {
  authDialogVisible.value = false
  authProvider.value = null
  authStatus.value = null
  authHistory.value = { items: [], total: 0, page: 1, page_size: 20 }
}

async function reloadAuthData() {
  if (!authProvider.value) return
  authLoading.value = true
  try {
    const [status, history] = await Promise.all([
      supplierProvidersAPI.getAuthStatus(authProvider.value.id),
      supplierProvidersAPI.listAuthHistory(authProvider.value.id, {
        page: authHistory.value.page,
        page_size: authHistory.value.page_size,
        event_type: authEventFilter.value,
      }),
    ])
    authStatus.value = status
    authHistory.value = history
  } catch (err) {
    error.value = errorMessage(err, '加载供应商登录记录失败')
  } finally {
    authLoading.value = false
  }
}

async function reloadAuthHistory() {
  authHistory.value.page = 1
  await reloadAuthData()
}

async function changeAuthPage(offset: number) {
  authHistory.value.page = Math.max(1, authHistory.value.page + offset)
  await reloadAuthData()
}

function formatAuthTime(value?: string): string {
  if (!value) return '—'
  const date = new Date(value)
  return Number.isNaN(date.getTime()) ? value : date.toLocaleString('zh-CN', { hour12: false })
}

function formatDuration(seconds: number): string {
  const value = Math.max(0, Number(seconds || 0))
  if (value <= 0) return '0 秒'
  if (value < 60) return `${Math.floor(value)} 秒`
  if (value < 3600) return `${Math.floor(value / 60)} 分 ${Math.floor(value % 60)} 秒`
  return `${Math.floor(value / 3600)} 小时 ${Math.floor((value % 3600) / 60)} 分`
}

function authCacheLabel(status: string): string {
  return ({ cached: '已缓存', missing: '未缓存', expired: '已过期', error: '缓存异常' } as Record<string, string>)[status] || status || '未知'
}

function authEventLabel(value: string): string {
  return ({
    cache_hit: '缓存命中', cache_miss: '缓存未命中', login_success: '登录成功', login_failed: '登录失败',
    cache_invalidated: '缓存失效', cache_error: '缓存异常', success: '成功', failed: '失败', miss: '未命中',
    invalidated: '已失效', unavailable: '不可用',
  } as Record<string, string>)[value] || value || '—'
}

function authSourceLabel(value: string): string {
  return ({ sync: '同步任务', endpoint_test: '接口测试', manual: '手动操作', unknown: '未知来源' } as Record<string, string>)[value] || value
}

async function refreshProvidersView() {
  await Promise.all([loadProviders(), loadCostTrends(), loadCostBreakdown()])
}

function applyHealthTodo(todo: HealthTodo) {
  if (todo.filter) filter.value = todo.filter
  if (todo.quickFilter) providerQuickFilter.value = todo.quickFilter
  if (todo.selectId) {
    selectedProvider.value = providers.value.find(provider => provider.id === todo.selectId) || null
  }
}

function formatCostTrendLabel(date: string) {
  const parts = String(date || '').split('-')
  if (parts.length >= 3) return `${Number(parts[1])}/${Number(parts[2])}`
  return date
}

function openCreate() {
  editingProvider.value = null
  Object.assign(form, emptyForm())
  if (!form.provider_type && enabledProviderTypes.value[0]) form.provider_type = enabledProviderTypes.value[0].code
  applySelectedTypeTemplate(false)
  modalVisible.value = true
}

function openEdit(provider: SupplierProvider) {
  editingProvider.value = provider
  Object.assign(form, {
    code: provider.code,
    name: provider.name,
    provider_type: provider.provider_type,
    base_url: provider.base_url,
    login_url: provider.login_url,
    api_keys_url: provider.api_keys_url,
    groups_url: provider.groups_url,
    available_groups_url: provider.available_groups_url,
    balance_url: provider.balance_url,
    usage_cost_url: provider.usage_cost_url,
    email: provider.email,
    username: provider.username,
    password: '',
    account_name_prefix: provider.account_name_prefix,
    temp_disable_minutes: provider.temp_disable_minutes,
    account_rate_multiplier_scale: provider.account_rate_multiplier_scale || 1,
    sort_order: provider.sort_order,
    enabled: provider.enabled,
    turnstile_enabled: Boolean(provider.turnstile_enabled),
    is_default: provider.is_default,
  })
  modalVisible.value = true
}

function closeModal() {
  modalVisible.value = false
}

function openCreateProviderType() {
  newProviderType()
  createTypeVisible.value = true
}

function closeCreateProviderType() {
  createTypeVisible.value = false
}

function openTypeManager() {
  typeManagerVisible.value = true
  if (providerTypes.value.length) editProviderType(providerTypes.value[0])
  else newProviderType()
}

function closeTypeManager() {
  typeManagerVisible.value = false
}

function newProviderType() {
  editingProviderType.value = null
  Object.assign(typeForm, emptyTypeForm())
}

function editProviderType(providerType: SupplierProviderType) {
  editingProviderType.value = providerType
  Object.assign(typeForm, {
    code: providerType.code,
    name: providerType.name,
    login_url: providerType.login_url,
    api_keys_url: providerType.api_keys_url,
    groups_url: providerType.groups_url,
    available_groups_url: providerType.available_groups_url,
    balance_url: providerType.balance_url,
    usage_cost_url: providerType.usage_cost_url,
    enabled: providerType.enabled,
    sort_order: providerType.sort_order,
  })
}

async function submitNewProviderType() {
  const payload = normalizeTypePayload(typeForm)
  try {
    await supplierProviderTypesAPI.create(payload)
    appStore.showSuccess('供应商类型已创建')
    await loadProviderTypes()
    createTypeVisible.value = false
  } catch (err) {
    appStore.showError(errorMessage(err, '创建供应商类型失败'))
  }
}

async function submitProviderType() {
  const payload = normalizeTypePayload(typeForm)
  try {
    if (editingProviderType.value) {
      await supplierProviderTypesAPI.update(editingProviderType.value.id, payload)
      appStore.showSuccess('供应商类型已更新')
    } else {
      await supplierProviderTypesAPI.create(payload)
      appStore.showSuccess('供应商类型已创建')
    }
    await loadProviderTypes()
    const refreshed = providerTypes.value.find(type => type.code === payload.code) || null
    if (refreshed) editProviderType(refreshed)
  } catch (err) {
    appStore.showError(errorMessage(err, '保存供应商类型失败'))
  }
}

async function removeProviderType(providerType: SupplierProviderType) {
  if (!window.confirm(`确认删除供应商类型「${providerType.name}」？`)) return
  try {
    await supplierProviderTypesAPI.delete(providerType.id)
    appStore.showSuccess('供应商类型已删除')
    await loadProviderTypes()
    if (providerTypes.value.length) editProviderType(providerTypes.value[0])
    else newProviderType()
  } catch (err) {
    appStore.showError(errorMessage(err, '删除供应商类型失败'))
  }
}

function providerUpdatePayload(provider: SupplierProvider, enabled: boolean): SupplierProviderUpsertPayload {
  return normalizePayload({
    code: provider.code,
    name: provider.name,
    provider_type: provider.provider_type,
    base_url: provider.base_url,
    login_url: provider.login_url,
    api_keys_url: provider.api_keys_url,
    groups_url: provider.groups_url,
    available_groups_url: provider.available_groups_url,
    balance_url: provider.balance_url,
    usage_cost_url: provider.usage_cost_url,
    email: provider.email,
    username: provider.username,
    password: '',
    account_name_prefix: provider.account_name_prefix,
    temp_disable_minutes: provider.temp_disable_minutes,
    account_rate_multiplier_scale: provider.account_rate_multiplier_scale || 1,
    sort_order: provider.sort_order,
    enabled,
    turnstile_enabled: Boolean(provider.turnstile_enabled),
    is_default: provider.is_default,
  })
}

async function updateProviderEnabled(provider: SupplierProvider, enabled: boolean) {
  if (updatingProviderIDs.value.has(provider.id) || provider.enabled === enabled) return

  const previousEnabled = provider.enabled
  provider.enabled = enabled
  updatingProviderIDs.value = new Set(updatingProviderIDs.value).add(provider.id)
  try {
    await supplierProvidersAPI.update(provider.id, providerUpdatePayload(provider, enabled))
    appStore.showSuccess(enabled ? '供应商已启用' : '供应商已停用')
    await loadProviders()
  } catch (err) {
    provider.enabled = previousEnabled
    appStore.showError(errorMessage(err, '更新供应商运行状态失败'))
  } finally {
    const next = new Set(updatingProviderIDs.value)
    next.delete(provider.id)
    updatingProviderIDs.value = next
  }
}

async function submitProvider() {
  const payload = normalizePayload(form)
  try {
    if (editingProvider.value) {
      await supplierProvidersAPI.update(editingProvider.value.id, payload)
      appStore.showSuccess('供应商已更新')
    } else {
      await supplierProvidersAPI.create(payload)
      appStore.showSuccess('供应商已创建')
    }
    modalVisible.value = false
    await loadProviders()
  } catch (err) {
    appStore.showError(errorMessage(err, '保存供应商失败'))
  }
}

async function makeDefault(provider: SupplierProvider) {
  try {
    await supplierProvidersAPI.setDefault(provider.id)
    appStore.showSuccess('默认供应商已更新')
    await loadProviders()
  } catch (err) {
    appStore.showError(errorMessage(err, '设置默认供应商失败'))
  }
}

async function removeProvider(provider: SupplierProvider) {
  if (!window.confirm(`确认删除供应商「${provider.name}」？`)) return
  try {
    await supplierProvidersAPI.delete(provider.id)
    appStore.showSuccess('供应商已删除')
    if (selectedProvider.value?.id === provider.id) selectedProvider.value = null
    await loadProviders()
  } catch (err) {
    appStore.showError(errorMessage(err, '删除供应商失败'))
  }
}

async function syncProviderData(provider: SupplierProvider, scope: SupplierSyncScope) {
  const key = `${provider.id}:${scope}`
  if (syncingKeys.value.has(key)) return
  syncingKeys.value = new Set(syncingKeys.value).add(key)
  try {
    const result = await syncProvider(provider.id, scope)
    showSyncResultFeedback(result.status, scope)
    await loadProviders()
  } catch (err) {
    appStore.showError(errorMessage(err, '同步供应商失败'))
  } finally {
    const next = new Set(syncingKeys.value)
    next.delete(key)
    syncingKeys.value = next
  }
}

function isSyncing(provider: SupplierProvider, scope: SupplierSyncScope): boolean {
  return syncingKeys.value.has(`${provider.id}:${scope}`)
}

async function testProviderEndpointData(provider: SupplierProvider, scope: SupplierDiagnosticScope) {
  const key = `${provider.id}:${scope}`
  if (testingKeys.value.has(key)) return
  testingKeys.value = new Set(testingKeys.value).add(key)
  try {
    testResult.value = await testProviderEndpoint(provider.id, scope)
    testResultVisible.value = true
  } catch (err) {
    appStore.showError(errorMessage(err, '测试供应商接口失败'))
  } finally {
    const next = new Set(testingKeys.value)
    next.delete(key)
    testingKeys.value = next
  }
}

function isTesting(provider: SupplierProvider, scope: SupplierDiagnosticScope): boolean {
  return testingKeys.value.has(`${provider.id}:${scope}`)
}

function closeTestResult() {
  testResultVisible.value = false
}

function syncResultText(status: string, scope: SupplierSyncScope): string {
  const label: Record<SupplierSyncScope, string> = {
    accounts: 'API Key',
    groups: '分组',
    balance: '余额',
    cost: '成本',
    all: '全部数据',
  }
  if (status === 'partial') return `${label[scope]}部分同步失败`
  if (status === 'failed') return `${label[scope]}同步失败`
  return `${label[scope]}同步完成`
}

function scopeLabel(scope: string): string {
  const label: Record<string, string> = {
    accounts: 'API Key',
    groups: '分组',
    balance: '余额',
    cost: '成本',
    all: '全部数据',
  }
  return label[scope] || scope
}

function formatDiagnosticJSON(value: unknown): string {
  if (value === undefined || value === null || value === '') return '无解析结果'
  try {
    return JSON.stringify(value, null, 2)
  } catch {
    return String(value)
  }
}

function toNumber(value: string | number, fallback: number): number {
  const next = Number(value)
  return Number.isFinite(next) ? next : fallback
}

function normalizePayload(payload: SupplierProviderUpsertPayload): SupplierProviderUpsertPayload {
  const normalizedProviderType = payload.provider_type.trim()
  return {
    ...payload,
    code: payload.code.trim(),
    name: payload.name.trim(),
    provider_type: normalizedProviderType,
    base_url: payload.base_url.trim(),
    login_url: payload.login_url?.trim() || '',
    api_keys_url: payload.api_keys_url?.trim() || '',
    groups_url: payload.groups_url?.trim() || '',
    available_groups_url: payload.groups_url?.trim() || '',
    balance_url: payload.balance_url?.trim() || '',
    usage_cost_url: payload.usage_cost_url?.trim() || '',
    email: normalizedProviderType === 'sub2api' ? payload.email?.trim() || '' : '',
    username: normalizedProviderType === 'sub2api' ? '' : payload.username?.trim() || '',
    password: payload.password?.trim() || '',
    account_name_prefix: payload.account_name_prefix?.trim() || '',
    temp_disable_minutes: Number(payload.temp_disable_minutes || 0),
    account_rate_multiplier_scale: Number(payload.account_rate_multiplier_scale || 1),
    sort_order: Number(payload.sort_order || 0),
    enabled: Boolean(payload.enabled),
    turnstile_enabled: Boolean(payload.turnstile_enabled),
    is_default: Boolean(payload.is_default),
  }
}

function normalizeTypePayload(payload: SupplierProviderTypeUpsertPayload): SupplierProviderTypeUpsertPayload {
  return {
    code: payload.code.trim(),
    name: payload.name.trim(),
    login_url: payload.login_url?.trim() || '',
    api_keys_url: payload.api_keys_url?.trim() || '',
    groups_url: payload.groups_url?.trim() || '',
    available_groups_url: payload.groups_url?.trim() || '',
    balance_url: payload.balance_url?.trim() || '',
    usage_cost_url: payload.usage_cost_url?.trim() || '',
    enabled: Boolean(payload.enabled),
    sort_order: Number(payload.sort_order || 0),
  }
}

function applySelectedTypeTemplate(overwrite: boolean) {
  const providerType = providerTypes.value.find(type => type.code === form.provider_type)
  if (!providerType) return
  applyTemplateField('login_url', providerType.login_url, overwrite)
  applyTemplateField('api_keys_url', providerType.api_keys_url, overwrite)
  applyTemplateField('groups_url', providerType.groups_url, overwrite)
  applyTemplateField('balance_url', providerType.balance_url, overwrite)
  applyTemplateField('usage_cost_url', providerType.usage_cost_url, overwrite)
}

function applyTemplateField(field: keyof Pick<SupplierProviderUpsertPayload, 'login_url' | 'api_keys_url' | 'groups_url' | 'balance_url' | 'usage_cost_url'>, value: string, overwrite: boolean) {
  if (!value) return
  if (overwrite || !String(form[field] || '').trim()) form[field] = value
}

function riskWeight(provider: SupplierProvider): number {
  const risk = provider.risk_level === 'critical' ? 400 : provider.risk_level === 'high' ? 300 : provider.risk_level === 'medium' ? 150 : 0
  const balance = isLowBalance(provider) ? 80 : 0
  const sync = provider.sync_status === 'failed' ? 60 : 0
  return risk + balance + sync + provider.rate_risk_count
}

function handleProviderSort(key: string, order: 'asc' | 'desc') {
  providerSortKey.value = key
  providerSortOrder.value = order
}

function compareProviders(left: SupplierProvider, right: SupplierProvider, key: string, order: 'asc' | 'desc'): number {
  if (key === 'last_sync_at') {
    const leftTimestamp = timestampValue(left.last_sync_at)
    const rightTimestamp = timestampValue(right.last_sync_at)
    if (leftTimestamp === null && rightTimestamp === null) return left.id - right.id
    if (leftTimestamp === null) return 1
    if (rightTimestamp === null) return -1
    const comparison = leftTimestamp - rightTimestamp
    return comparison === 0 ? left.id - right.id : comparison * sortDirection(order)
  }

  let comparison = 0
  switch (key) {
    case 'name':
      comparison = left.name.localeCompare(right.name, 'zh-CN')
      break
    case 'status':
      comparison = riskWeight(left) - riskWeight(right)
      break
    case 'account_counts':
      comparison = numericValue(left.valid_account_count) - numericValue(right.valid_account_count)
        || numericValue(left.schedulable_account_count) - numericValue(right.schedulable_account_count)
      break
    case 'success_rate':
      comparison = numericValue(left.success_rate) - numericValue(right.success_rate)
      break
    case 'today_cost':
      comparison = numericValue(left.today_cost) - numericValue(right.today_cost)
      break
    case 'current_balance':
      comparison = numericValue(left.current_balance) - numericValue(right.current_balance)
      break
    case 'rate_risk_count':
      comparison = numericValue(left.rate_risk_count) - numericValue(right.rate_risk_count)
      break
    case 'credential_configured':
      comparison = Number(left.credential_configured) - Number(right.credential_configured)
      break
    case 'auth_summary':
      comparison = numericValue(left.auth_summary?.login_count) - numericValue(right.auth_summary?.login_count)
      break
  }
  return comparison === 0 ? left.id - right.id : comparison * sortDirection(order)
}

function sortDirection(order: 'asc' | 'desc'): number {
  return order === 'asc' ? 1 : -1
}

function timestampValue(value?: string): number | null {
  if (!value) return null
  const timestamp = new Date(value).getTime()
  return Number.isNaN(timestamp) ? null : timestamp
}

function openProviderHomepage(provider: SupplierProvider) {
  const url = provider.base_url?.trim()
  if (!url) return
  window.open(url, '_blank', 'noopener,noreferrer')
}

function statusTone(provider: SupplierProvider): Tone {
  if (!provider.enabled) return ''
  if (['critical', 'high'].includes(provider.risk_level)) return 'bad'
  if (provider.risk_level === 'medium' || isLowBalance(provider)) return 'warn'
  if (provider.sync_status === 'failed') return 'info'
  return 'good'
}

function statusText(provider: SupplierProvider): string {
  if (!provider.enabled) return '已停用'
  if (provider.risk_level === 'critical') return '严重风险'
  if (provider.risk_level === 'high') return '高风险'
  if (provider.risk_level === 'medium') return '需关注'
  if (provider.status && provider.status !== 'unknown') return provider.status
  return provider.is_default ? '默认启用' : '启用'
}

function rateTone(provider: SupplierProvider): Tone {
  return provider.rate_risk_count > 0 ? 'warn' : 'good'
}

function rateRiskText(provider: SupplierProvider): string {
  return provider.rate_risk_count > 0 ? `${provider.rate_risk_count} 个风险` : '无风险'
}

function balanceText(provider: SupplierProvider): string {
  return currency(provider.current_balance)
}

function isLowBalance(provider: SupplierProvider): boolean {
  return typeof provider.estimated_days === 'number' && provider.estimated_days < 3
}

function isBalanceWarning(provider: SupplierProvider): boolean {
  return numericValue(provider.current_balance) < 10
}

function syncText(provider: SupplierProvider): string {
  if (provider.sync_status === 'failed') return '同步失败'
  if (!provider.last_sync_at) return '未同步'
  const timestamp = new Date(provider.last_sync_at).getTime()
  if (Number.isNaN(timestamp)) return '时间异常'
  const minutes = Math.max(0, Math.floor((Date.now() - timestamp) / 60000))
  if (minutes < 1) return '刚刚'
  if (minutes < 60) return `${minutes} 分钟前`
  const hours = Math.floor(minutes / 60)
  if (hours < 24) return `${hours} 小时前`
  return `${Math.floor(hours / 24)} 天前`
}

function percent(value: number): string {
  if (!value) return '0%'
  return `${value.toFixed(1)}%`
}

function currency(value: number): string {
  return `¥ ${numericValue(value).toLocaleString('zh-CN', { maximumFractionDigits: 2 })}`
}

function numericValue(value: unknown): number {
  const number = Number(value)
  return Number.isFinite(number) ? number : 0
}

function showSyncResultFeedback(status: string, scope: SupplierSyncScope) {
  const message = syncResultText(status, scope)
  if (status === 'failed') {
    appStore.showError(message)
    return
  }
  if (status === 'partial') {
    appStore.showWarning(message)
    return
  }
  appStore.showSuccess(message)
}

function errorMessage(err: unknown, fallback: string): string {
  if (typeof err === 'object' && err && 'message' in err) {
    const apiErr = err as { message?: unknown; reason?: unknown; code?: unknown }
    const reason = String(apiErr.reason || '')
    const message = String(apiErr.message || '')
    if (reason === 'SUPPLIER_PROVIDER_INVALID' || message === 'invalid supplier provider configuration') {
      return '供应商配置无效：请检查基础地址是否为完整 http/https 地址，接口路径是否以 / 开头，排序和倍率等数值是否有效。'
    }
    return message || fallback
  }
  return fallback
}

</script>

<style scoped>
.sp-provider-home-button {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 2rem;
  height: 2rem;
  border: 1px solid color-mix(in srgb, var(--sp-cyan) 30%, var(--sp-line));
  border-radius: 0.5rem;
  background: color-mix(in srgb, var(--sp-cyan) 8%, var(--sp-panel));
  color: var(--sp-cyan);
  transition: border-color 0.15s ease, background 0.15s ease, color 0.15s ease;
}

.sp-provider-home-button:hover:not(:disabled) {
  border-color: var(--sp-cyan);
  background: var(--sp-cyan);
  color: #fff;
}

.sp-provider-home-button:disabled {
  cursor: not-allowed;
  opacity: 0.4;
}

.sp-provider-today-cost {
  color: var(--sp-amber);
  font-weight: 700;
}

.sp-provider-balance-normal {
  color: var(--sp-cyan);
  font-weight: 700;
}

.sp-provider-balance-warning {
  color: var(--sp-red);
  font-weight: 800;
}

.sp-provider-filter-card {
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

/* 筛选区：搜索与操作同一行紧挨排列，快捷筛选单独通栏，消除中间大块留白 */
.sp-provider-filter-body {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 0.625rem 0.75rem;
  padding: 0.875rem 1rem 1rem;
}

.sp-provider-filter-fields {
  display: contents;
}

.sp-provider-filter-control {
  display: flex;
  flex: 1 1 16rem;
  min-width: min(14rem, 100%);
  max-width: 24rem;
  flex-direction: column;
  gap: 0.3rem;
}

.sp-provider-quick-filters {
  display: flex;
  flex: 1 1 100%;
  flex-wrap: wrap;
  order: 3;
  gap: 0.5rem;
  margin-top: 0;
}

.sp-provider-quick-filters .sp-button.active {
  border-color: var(--sp-cyan);
  background: color-mix(in srgb, var(--sp-cyan) 14%, var(--sp-panel));
  color: var(--sp-cyan);
}

.sp-provider-status-toggle {
  display: inline-flex;
  align-items: center;
  gap: 0.5rem;
}

.sp-provider-filter-actions {
  display: flex;
  flex: 0 1 auto;
  flex-wrap: wrap;
  align-items: center;
  justify-content: flex-end;
  order: 2;
  gap: 0.5rem;
}

/* 统计卡：页面层压缩窄屏高度，并保持 2 列（覆盖共享样式在 460px 的单列） */
.sp-metric-grid {
  gap: 0.75rem;
}

@media (max-width: 900px) {
  .sp-provider-filter-control {
    flex: 1 1 100%;
    max-width: none;
  }

  .sp-provider-filter-actions {
    width: 100%;
    justify-content: stretch;
    order: 3;
    padding-top: 0.75rem;
    border-top: 1px solid var(--sp-line);
  }

  .sp-provider-quick-filters {
    order: 2;
  }

  .sp-provider-filter-actions .sp-button {
    flex: 1 1 calc(50% - 0.5rem);
    min-width: 0;
  }
}

@media (max-width: 760px) {
  .sp-metric-grid {
    grid-template-columns: repeat(2, minmax(0, 1fr));
    gap: 0.5rem;
    margin-bottom: 0.75rem;
  }

  .sp-metric-card {
    min-height: 0;
    padding: 0.65rem 0.75rem;
    border-radius: 0.65rem;
  }

  .sp-metric-label {
    font-size: 0.6875rem;
  }

  .sp-metric-value {
    margin-top: 0.3rem;
    font-size: 1.35rem;
    line-height: 1.2;
  }

  .sp-metric-foot {
    margin-top: 0.35rem;
    font-size: 0.6875rem;
    line-height: 1.35;
    display: -webkit-box;
    -webkit-line-clamp: 2;
    -webkit-box-orient: vertical;
    overflow: hidden;
  }
}

@media (max-width: 520px) {
  .sp-filter-card-head {
    align-items: flex-start;
    flex-direction: column;
    padding: 0.75rem;
  }

  .sp-provider-filter-body {
    padding: 0.75rem;
  }

  /* 手机端筛选区保持紧凑：搜索通栏，快捷筛选与操作按钮 2 列 */
  .sp-provider-quick-filters {
    display: grid;
    grid-template-columns: repeat(2, minmax(0, 1fr));
    gap: 0.5rem;
  }

  .sp-provider-quick-filters .sp-button {
    width: 100%;
    min-width: 0;
    justify-content: center;
  }

  .sp-provider-filter-actions {
    display: grid;
    grid-template-columns: repeat(2, minmax(0, 1fr));
    gap: 0.5rem;
  }

  .sp-provider-filter-actions .sp-button {
    width: 100%;
    flex: initial;
    min-width: 0;
    min-height: 2.5rem;
    padding-inline: 0.55rem;
    font-size: 0.8125rem;
    white-space: normal;
    line-height: 1.25;
    text-align: center;
  }

  .sp-metric-grid {
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }
}

@media (max-width: 460px) {
  /* 覆盖共享样式把统计卡打成单列的行为 */
  .sp-metric-grid {
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }
}

.sp-provider-dialog,
.sp-type-create-dialog,
.sp-type-editor,
.sp-test-dialog {
  display: grid;
  gap: 16px;
  min-width: 0;
  color: var(--sp-text);
}

.sp-provider-dialog,
.sp-type-create-dialog,
.sp-type-editor {
  max-height: min(70vh, 720px);
  overflow: auto;
  padding: 2px 4px 12px 2px;
}

.sp-dialog-summary {
  display: grid;
  grid-template-columns: repeat(3, minmax(0, 1fr));
  overflow: hidden;
  border: 1px solid var(--sp-line);
  border-radius: 12px;
  background: var(--sp-panel-2);
}

.sp-test-dialog .sp-dialog-summary {
  grid-template-columns: repeat(4, minmax(0, 1fr));
}

.sp-dialog-summary > div {
  min-width: 0;
  padding: 13px 15px;
  border-right: 1px solid var(--sp-line);
}

.sp-dialog-summary > div:last-child {
  border-right: 0;
}

.sp-dialog-summary span,
.sp-dialog-summary strong {
  display: block;
}

.sp-dialog-summary span {
  color: var(--sp-muted);
  font-size: 11px;
  font-weight: 600;
  letter-spacing: 0.02em;
}

.sp-dialog-summary strong {
  margin-top: 6px;
  overflow: hidden;
  color: var(--sp-text);
  font-size: 14px;
  font-weight: 700;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.sp-dialog-section {
  display: grid;
  gap: 14px;
  padding: 16px;
  border: 1px solid var(--sp-line);
  border-radius: 12px;
  background: var(--sp-panel);
}

.sp-dialog-section-head {
  display: flex;
  align-items: flex-start;
  gap: 11px;
}

.sp-dialog-section-head > span {
  display: inline-grid;
  flex: 0 0 auto;
  width: 28px;
  height: 28px;
  place-items: center;
  border: 1px solid color-mix(in srgb, var(--sp-cyan) 30%, var(--sp-line));
  border-radius: 8px;
  background: color-mix(in srgb, var(--sp-cyan) 7%, var(--sp-panel));
  color: var(--sp-cyan);
  font-size: 10px;
  font-weight: 800;
  letter-spacing: 0.04em;
}

.sp-dialog-section-head h4 {
  margin: 0;
  color: var(--sp-text);
  font-size: 14px;
  font-weight: 700;
  line-height: 1.35;
}

.sp-dialog-section-head p {
  margin: 3px 0 0;
  color: var(--sp-muted);
  font-size: 12px;
  line-height: 1.5;
}

.sp-dialog-grid {
  display: grid;
  gap: 14px;
  min-width: 0;
}

.sp-dialog-grid-2 {
  grid-template-columns: repeat(2, minmax(0, 1fr));
}

.sp-dialog-grid-3 {
  grid-template-columns: repeat(3, minmax(0, 1fr));
}

.sp-select-field,
.sp-toggle-field {
  display: grid;
  align-content: start;
  gap: 0.375rem;
  color: var(--sp-text);
  font-size: 0.875rem;
  font-weight: 500;
}

.sp-dialog-toggle-card {
  min-height: 70px;
  padding: 10px 12px;
  border: 1px solid var(--sp-line);
  border-radius: 9px;
  background: var(--sp-panel-2);
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

.sp-toggle-hint {
  margin: 0;
  color: var(--sp-muted);
  font-size: 12px;
  font-weight: 400;
  line-height: 1.5;
}

.sp-dialog-note {
  padding: 10px 12px;
  border-left: 3px solid var(--sp-cyan);
  border-radius: 4px 8px 8px 4px;
  background: color-mix(in srgb, var(--sp-cyan) 6%, var(--sp-panel-2));
  color: var(--sp-muted);
  font-size: 12px;
  line-height: 1.6;
}

.sp-type-manager-dialog {
  display: grid;
  grid-template-columns: minmax(190px, 0.65fr) minmax(0, 2fr);
  gap: 16px;
  min-width: 0;
  max-height: min(70vh, 720px);
  color: var(--sp-text);
}

.sp-type-list {
  display: grid;
  align-content: start;
  gap: 8px;
  min-width: 0;
  overflow: auto;
  padding: 12px;
  border: 1px solid var(--sp-line);
  border-radius: 12px;
  background: var(--sp-panel-2);
}

.sp-type-list-head {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 8px;
  padding: 2px 2px 8px;
  border-bottom: 1px solid var(--sp-line);
}

.sp-type-list-head span,
.sp-type-list-head strong {
  display: block;
}

.sp-type-list-head span {
  color: var(--sp-cyan);
  font-size: 9px;
  font-weight: 800;
  letter-spacing: 0.1em;
  text-transform: uppercase;
}

.sp-type-list-head strong {
  margin-top: 3px;
  color: var(--sp-text);
  font-size: 13px;
}

.sp-type-list-head em {
  color: var(--sp-muted);
  font-size: 11px;
  font-style: normal;
}

.sp-type-row {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 10px;
  width: 100%;
  padding: 11px 12px;
  border: 1px solid var(--sp-line);
  border-left: 3px solid transparent;
  border-radius: 9px;
  background: var(--sp-panel);
  color: var(--sp-text);
  text-align: left;
  cursor: pointer;
  transition: border-color 0.15s ease, background 0.15s ease, transform 0.15s ease;
}

.sp-type-row:hover {
  border-color: color-mix(in srgb, var(--sp-cyan) 36%, var(--sp-line));
  transform: translateY(-1px);
}

.sp-type-row.active {
  border-color: color-mix(in srgb, var(--sp-cyan) 36%, var(--sp-line));
  border-left-color: var(--sp-cyan);
  background: color-mix(in srgb, var(--sp-cyan) 7%, var(--sp-panel));
}

.sp-type-row b,
.sp-type-row small {
  display: block;
}

.sp-type-row b {
  font-size: 13px;
}

.sp-type-row small {
  margin-top: 4px;
  color: var(--sp-muted);
  font-size: 11px;
}

.sp-type-row em {
  flex: 0 0 auto;
  padding: 3px 7px;
  border: 1px solid var(--sp-line);
  border-radius: 999px;
  color: var(--sp-muted);
  font-size: 10px;
  font-style: normal;
  font-weight: 700;
}

.sp-type-row em.good {
  border-color: color-mix(in srgb, var(--sp-green) 32%, var(--sp-line));
  color: var(--sp-green);
}

.sp-type-row em.warn {
  border-color: color-mix(in srgb, var(--sp-amber) 32%, var(--sp-line));
  color: var(--sp-amber);
}

.sp-type-empty {
  padding: 18px 10px;
  color: var(--sp-muted);
  font-size: 12px;
  line-height: 1.6;
  text-align: center;
}

.sp-type-editor {
  min-height: 0;
}

.sp-dialog-danger-zone {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 16px;
  padding: 13px 14px;
  border: 1px solid color-mix(in srgb, var(--sp-red) 25%, var(--sp-line));
  border-radius: 10px;
  background: color-mix(in srgb, var(--sp-red) 4%, var(--sp-panel));
}

.sp-dialog-danger-zone strong,
.sp-dialog-danger-zone span {
  display: block;
}

.sp-dialog-danger-zone strong {
  color: var(--sp-text);
  font-size: 12px;
}

.sp-dialog-danger-zone span {
  margin-top: 3px;
  color: var(--sp-muted);
  font-size: 11px;
}

.sp-test-dialog {
  max-height: 72vh;
  overflow: auto;
  padding: 2px 4px 12px 2px;
}

.sp-test-result {
  --sp-test-accent: var(--sp-green);
  display: grid;
  gap: 16px;
}

.sp-test-result.bad {
  --sp-test-accent: var(--sp-red);
}

.sp-test-result .sp-dialog-summary {
  border-top: 3px solid var(--sp-test-accent);
}

.sp-test-attempts {
  display: grid;
  gap: 8px;
}

.sp-test-attempt {
  padding: 12px 14px;
  border: 1px solid var(--sp-line);
  border-radius: 9px;
  background: var(--sp-panel-2);
}

.sp-test-attempt > div {
  display: flex;
  align-items: center;
  gap: 10px;
  min-width: 0;
}

.sp-test-attempt > div > span {
  display: inline-grid;
  flex: 0 0 auto;
  width: 24px;
  height: 24px;
  place-items: center;
  border-radius: 7px;
  background: var(--sp-panel-3);
  color: var(--sp-muted);
  font-size: 9px;
  font-weight: 800;
}

.sp-test-attempt strong {
  overflow: hidden;
  color: var(--sp-text);
  font-size: 12px;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.sp-test-attempt p {
  margin: 7px 0 0 34px;
  color: var(--sp-muted);
  font-size: 11px;
  line-height: 1.5;
}

.sp-test-attempt p.bad {
  color: var(--sp-red);
}

.sp-test-response-grid {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 12px;
}

.sp-response-panel {
  min-width: 0;
  overflow: hidden;
  border: 1px solid var(--sp-line);
  border-radius: 10px;
  background: var(--sp-panel-2);
}

.sp-response-panel-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 10px;
  padding: 10px 12px;
  border-bottom: 1px solid var(--sp-line);
}

.sp-response-panel-head strong {
  color: var(--sp-text);
  font-size: 12px;
}

.sp-response-panel-head span {
  color: var(--sp-muted);
  font-size: 9px;
  font-weight: 700;
  letter-spacing: 0.08em;
  text-transform: uppercase;
}

.sp-response-panel .sp-message-detail {
  max-height: 300px;
  margin: 0;
  overflow: auto;
  border: 0;
  border-radius: 0;
  background: transparent;
  white-space: pre-wrap;
  word-break: break-word;
}

:global(.modal-content:has(.sp-provider-dialog)),
:global(.modal-content:has(.sp-type-create-dialog)),
:global(.modal-content:has(.sp-type-manager-dialog)),
:global(.modal-content:has(.sp-test-dialog)) {
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
  --sp-red: #d14343;
  overflow: hidden;
  border-color: #cbd7e5;
  background: var(--sp-panel);
  color: var(--sp-text);
}

:global(.dark .modal-content:has(.sp-provider-dialog)),
:global(.dark .modal-content:has(.sp-type-create-dialog)),
:global(.dark .modal-content:has(.sp-type-manager-dialog)),
:global(.dark .modal-content:has(.sp-test-dialog)) {
  --sp-panel: #172033;
  --sp-panel-2: #1d293d;
  --sp-panel-3: #243249;
  --sp-line: #35445c;
  --sp-soft: #2c3a51;
  --sp-text: #edf3fb;
  --sp-muted: #a8b6ca;
  border-color: #3b4b64;
}

:global(.modal-content:has(.sp-provider-dialog) .modal-header),
:global(.modal-content:has(.sp-type-create-dialog) .modal-header),
:global(.modal-content:has(.sp-type-manager-dialog) .modal-header),
:global(.modal-content:has(.sp-test-dialog) .modal-header),
:global(.modal-content:has(.sp-provider-dialog) .modal-footer),
:global(.modal-content:has(.sp-type-create-dialog) .modal-footer),
:global(.modal-content:has(.sp-type-manager-dialog) .modal-footer),
:global(.modal-content:has(.sp-test-dialog) .modal-footer) {
  border-color: var(--sp-line);
  background: var(--sp-panel);
}

:global(.modal-content:has(.sp-provider-dialog) .modal-title),
:global(.modal-content:has(.sp-type-create-dialog) .modal-title),
:global(.modal-content:has(.sp-type-manager-dialog) .modal-title),
:global(.modal-content:has(.sp-test-dialog) .modal-title) {
  color: var(--sp-text);
}

:global(.modal-content:has(.sp-provider-dialog) .modal-body),
:global(.modal-content:has(.sp-type-create-dialog) .modal-body),
:global(.modal-content:has(.sp-type-manager-dialog) .modal-body),
:global(.modal-content:has(.sp-test-dialog) .modal-body) {
  min-height: 0;
  overflow: hidden;
  background: var(--sp-panel);
}

@media (max-width: 920px) {
  .sp-dialog-grid-3,
  .sp-test-dialog .sp-dialog-summary {
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }

  .sp-type-manager-dialog {
    grid-template-columns: minmax(170px, 0.55fr) minmax(0, 1.45fr);
  }
}

@media (max-width: 760px) {
  .sp-provider-dialog,
  .sp-type-create-dialog,
  .sp-type-editor,
  .sp-test-dialog,
  .sp-type-manager-dialog {
    max-height: 68vh;
  }

  .sp-dialog-summary,
  .sp-test-dialog .sp-dialog-summary,
  .sp-dialog-grid-2,
  .sp-dialog-grid-3,
  .sp-test-response-grid,
  .sp-type-manager-dialog {
    grid-template-columns: 1fr;
  }

  .sp-dialog-summary > div {
    border-right: 0;
    border-bottom: 1px solid var(--sp-line);
  }

  .sp-dialog-summary > div:last-child {
    border-bottom: 0;
  }

  .sp-dialog-section {
    padding: 14px;
  }

  .sp-type-list {
    max-height: 210px;
  }

  .sp-dialog-danger-zone,
  .sp-response-panel-head {
    align-items: stretch;
    flex-direction: column;
  }

  .sp-dialog-danger-zone .sp-button {
    width: 100%;
    min-height: 40px;
  }

  .sp-test-attempt strong {
    white-space: normal;
    word-break: break-all;
  }
}

.sp-health-panel {
  display: flex;
  flex-direction: column;
  min-height: 0;
}

.sp-health-head-right {
  display: inline-flex;
  align-items: center;
  gap: 0.5rem;
  flex-shrink: 0;
}

.sp-health-body {
  display: flex;
  flex-direction: column;
  gap: 0.875rem;
}

.sp-health-summary {
  padding: 0.75rem 0.875rem;
  border: 1px solid var(--sp-line);
  border-radius: 0.75rem;
  background: color-mix(in srgb, var(--sp-panel) 92%, var(--sp-cyan));
}

.sp-health-summary.good {
  border-color: color-mix(in srgb, var(--sp-green) 28%, var(--sp-line));
  background: color-mix(in srgb, var(--sp-green) 8%, var(--sp-panel));
}

.sp-health-summary.warn {
  border-color: color-mix(in srgb, var(--sp-amber) 28%, var(--sp-line));
  background: color-mix(in srgb, var(--sp-amber) 8%, var(--sp-panel));
}

.sp-health-summary.bad {
  border-color: color-mix(in srgb, var(--sp-red) 28%, var(--sp-line));
  background: color-mix(in srgb, var(--sp-red) 8%, var(--sp-panel));
}

.sp-health-summary strong {
  display: block;
  color: var(--sp-text);
  font-size: 0.9375rem;
  font-weight: 800;
}

.sp-health-summary p {
  margin: 0.35rem 0 0;
  color: var(--sp-muted);
  font-size: 0.8125rem;
  line-height: 1.5;
}

.sp-health-section {
  display: flex;
  flex-direction: column;
  gap: 0.625rem;
}

.sp-health-section-head {
  display: flex;
  align-items: baseline;
  justify-content: space-between;
  gap: 0.75rem;
}

.sp-health-section-title {
  min-width: 0;
}

.sp-health-section-head > span,
.sp-health-section-title > span {
  color: var(--sp-text);
  font-size: 0.8125rem;
  font-weight: 800;
}

.sp-health-section-head > small {
  color: var(--sp-muted);
  font-size: 0.72rem;
}

.sp-health-section-title > small {
  color: var(--sp-muted);
  font-size: 0.72rem;
  display: block;
  margin-top: 0.2rem;
}

.sp-health-todo-list {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 0.5rem;
}

.sp-health-todo {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 0.75rem;
  width: 100%;
  min-width: 0;
  padding: 0.7rem 0.8rem;
  border: 1px solid var(--sp-line);
  border-radius: 0.75rem;
  background: var(--sp-panel);
  text-align: left;
  cursor: pointer;
  transition: border-color 0.15s ease, background 0.15s ease;
}

.sp-health-todo:hover {
  border-color: color-mix(in srgb, var(--sp-cyan) 35%, var(--sp-line));
  background: color-mix(in srgb, var(--sp-cyan) 6%, var(--sp-panel));
}

.sp-health-todo.bad {
  border-color: color-mix(in srgb, var(--sp-red) 28%, var(--sp-line));
}

.sp-health-todo.warn {
  border-color: color-mix(in srgb, var(--sp-amber) 28%, var(--sp-line));
}

.sp-health-todo-main {
  min-width: 0;
}

.sp-health-todo-main strong {
  display: block;
  color: var(--sp-text);
  font-size: 0.8125rem;
  font-weight: 800;
}

.sp-health-todo-main small {
  display: block;
  margin-top: 0.2rem;
  color: var(--sp-muted);
  font-size: 0.72rem;
  line-height: 1.4;
}

.sp-health-todo-action {
  flex-shrink: 0;
  margin-top: 0.1rem;
  color: var(--sp-cyan);
  font-size: 0.72rem;
  font-weight: 800;
}

.sp-health-completeness-list {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 0.5rem;
  margin-top: 0;
}

.sp-health-completeness-list .sp-list-item {
  align-items: flex-start;
  min-width: 0;
  height: 100%;
  padding: 0.7rem 0.8rem;
  border: 1px solid var(--sp-line);
  border-radius: 0.75rem;
  background: var(--sp-panel);
}

.sp-health-chart-meta {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 0.75rem;
  flex-wrap: wrap;
}

.sp-health-chart-legend {
  display: inline-flex;
  align-items: center;
  gap: 0.75rem;
}

.sp-health-legend-item {
  display: inline-flex;
  align-items: center;
  gap: 0.35rem;
  color: var(--sp-muted);
  font-size: 0.72rem;
  font-weight: 700;
}

.sp-health-legend-item i {
  width: 0.55rem;
  height: 0.55rem;
  border-radius: 999px;
}

.sp-health-legend-item.upstream i {
  background: #3b82f6;
}

.sp-health-legend-item.local i {
  background: #d97706;
}

.sp-health-chart-totals {
  display: inline-flex;
  align-items: center;
  gap: 0.75rem;
  color: var(--sp-muted);
  font-size: 0.72rem;
}

.sp-health-chart-totals b {
  color: var(--sp-text);
  font-weight: 800;
}

.sp-health-chart-canvas {
  height: 180px;
  border: 1px solid var(--sp-line);
  border-radius: 0.75rem;
  padding: 0.5rem 0.35rem 0.25rem;
  background: color-mix(in srgb, var(--sp-panel) 94%, #94a3b8);
}

.sp-health-chart-empty {
  display: flex;
  align-items: center;
  justify-content: center;
  height: 100%;
  color: var(--sp-muted);
  font-size: 0.8125rem;
  text-align: center;
  padding: 0.75rem;
}

.sp-cost-breakdown-panel {
  margin-bottom: 1rem;
}

.sp-cost-breakdown-head-left {
  display: flex;
  min-width: 0;
  align-items: center;
  flex: 1 1 auto;
  gap: 0.75rem;
  flex-wrap: wrap;
}

.sp-cost-breakdown-head-left .sp-health-date-range-control {
  flex: 0 1 auto;
  justify-content: flex-start;
}

.sp-cost-breakdown-head-left .sp-panel-title {
  min-width: 12rem;
  flex: 1 1 18rem;
}

.sp-cost-breakdown-body {
  display: flex;
  flex-direction: column;
  gap: 0.75rem;
}

.sp-cost-breakdown-count {
  flex-shrink: 0;
  color: var(--sp-muted);
  font-size: 0.72rem;
}

.sp-cost-breakdown-count b {
  color: var(--sp-text);
  font-weight: 800;
}

.sp-health-breakdown-chart {
  position: relative;
  width: 100%;
  min-width: 0;
  height: 300px;
  padding: 0.75rem 0.5rem 0.5rem;
  overflow: hidden;
  border: 1px solid var(--sp-line);
  border-radius: 0.75rem;
  background: color-mix(in srgb, var(--sp-panel) 94%, #94a3b8);
}

.sp-health-breakdown-chart :deep(canvas) {
  display: block;
  max-width: 100%;
}


.sp-health-chart-controls {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 0.625rem;
  flex-wrap: wrap;
}

.sp-health-cost-section-head {
  align-items: center;
  flex-wrap: wrap;
}

.sp-health-cost-section-head .sp-health-date-range-control {
  flex: 0 1 auto;
}

.sp-health-date-range-control {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  flex: 1 1 18rem;
  justify-content: flex-end;
}

.sp-health-control-label {
  flex-shrink: 0;
  color: var(--sp-muted);
  font-size: 0.72rem;
  font-weight: 700;
}

.sp-health-date-range {
  min-width: 12rem;
  max-width: 18rem;
  flex: 1 1 12rem;
}

.sp-health-date-range :deep(.date-picker-root) {
  width: 100%;
}

.sp-health-date-range :deep(.date-picker-trigger) {
  width: 100%;
  min-height: 2rem;
  border-color: var(--sp-line);
  background: color-mix(in srgb, var(--sp-panel) 88%, #000);
  color: var(--sp-text);
  box-shadow: none;
}

.sp-health-date-range :deep(.date-picker-trigger-open),
.sp-health-date-range :deep(.date-picker-trigger:hover) {
  border-color: color-mix(in srgb, var(--sp-cyan) 42%, var(--sp-line));
}

.sp-health-date-range :deep(.date-picker-dropdown) {
  z-index: 40;
  border-color: var(--sp-line);
  background: var(--sp-panel);
}

.sp-health-provider-filter {
  min-width: 10rem;
  max-width: 14rem;
}

@media (max-width: 640px) {
  .sp-health-todo-list {
    grid-template-columns: 1fr;
  }

  .sp-health-completeness-list {
    grid-template-columns: 1fr;
  }

  .sp-health-cost-section-head .sp-health-date-range-control {
    width: 100%;
    flex-basis: 100%;
    justify-content: flex-start;
  }

  .sp-health-date-range {
    max-width: none;
  }
}

@media (max-width: 760px) {
  .sp-cost-breakdown-head-left {
    width: 100%;
    align-items: flex-start;
  }

  .sp-cost-breakdown-head-left .sp-health-date-range-control {
    flex: 1 1 100%;
    justify-content: flex-start;
  }

  .sp-cost-breakdown-head-left .sp-health-date-range {
    max-width: none;
  }
}

</style>

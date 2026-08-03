<template>
  <SupplierModuleLayout>
    <section class="sp-filter-toolbar" aria-label="分组筛选与操作">
      <header class="sp-filter-card-head">
        <div>
          <span class="sp-filter-card-kicker">筛选条件</span>
          <h2>筛选分组</h2>
          <p>按供应商、平台、匹配状态和倍率状态定位上游分组。</p>
        </div>
      </header>

      <div class="sp-filter-card-body">
        <div class="sp-filter-fields">
          <div class="sp-filter-control sp-filter-search-control">
            <span class="sr-only">分组搜索</span>
            <Input v-model="search" class="w-full" placeholder="搜索上游分组或 Key" />
          </div>
          <div class="sp-filter-control">
            <span class="sr-only">供应商</span>
            <Select
              v-model="providerID"
              class="w-full"
              :options="providerOptions"
              :searchable="true"
              search-placeholder="搜索供应商"
            />
          </div>
          <div class="sp-filter-control">
            <span class="sr-only">平台</span>
            <Select
              v-model="platformFilter"
              class="w-full"
              :options="platformFilterOptions"
              :searchable="false"
            />
          </div>
          <div class="sp-filter-control">
            <span class="sr-only">匹配状态</span>
            <Select
              v-model="matchStatusFilter"
              class="w-full"
              :options="matchStatusFilterOptions"
              :searchable="false"
            />
          </div>
          <div class="sp-filter-control">
            <span class="sr-only">倍率状态</span>
            <Select
              v-model="rateStatusFilter"
              class="w-full"
              :options="rateStatusFilterOptions"
              :searchable="false"
            />
          </div>
        </div>
        <div class="sp-filter-actions">
          <button class="sp-button small sp-control-button sp-control-button-reset" type="button" :disabled="loading || !canResetFilters" @click="resetGroupFilters">
            <Icon name="x" size="sm" />
            <span>重置筛选</span>
          </button>
          <button class="sp-button sp-control-button sp-control-button-match" type="button" :disabled="loading || autoMatching" @click="runAutoMatch">
            <Icon name="sync" size="sm" :class="autoMatching ? 'sp-spin' : ''" />
            <span>{{ autoMatching ? '匹配中' : '自动匹配' }}</span>
          </button>
          <button
            class="sp-button sp-control-button sp-control-button-log"
            type="button"
            :disabled="rateGuardChangeLogsLoading"
            @click="openRateGuardChangeLogs"
          >
            <Icon name="clock" size="sm" />
            <span>分组倍率变更日志</span>
            <em v-if="pendingRateGuardChangeLogCount > 0" class="sp-pending-count">{{ pendingRateGuardChangeLogCount }}</em>
          </button>
          <button class="sp-button sp-control-button sp-control-button-refresh" type="button" :disabled="loading" @click="refreshAll">
            <Icon name="refresh" size="sm" :class="loading ? 'sp-spin' : ''" />
            <span>刷新</span>
          </button>
        </div>
      </div>
    </section>
    <div v-if="error" class="sp-alert sp-error-line">{{ error }}</div>

    <div class="sp-console-shell">
      <div class="sp-summary-grid" aria-label="分组匹配汇总">
        <button
          type="button"
          class="sp-summary-filter"
          :class="{ active: isSummaryFilterActive('all') }"
          :aria-pressed="isSummaryFilterActive('all')"
          :disabled="loading"
          aria-label="显示全部上游分组"
          @click="applySummaryFilter('all')"
        >
          <StatCard
            title="上游分组"
            :value="groupSummary.group_count"
            :icon="UpstreamGroupsIcon"
            icon-variant="primary"
          />
        </button>
        <button
          type="button"
          class="sp-summary-filter"
          :class="{ active: isSummaryFilterActive('linked') }"
          :aria-pressed="isSummaryFilterActive('linked')"
          :disabled="loading"
          aria-label="过滤已匹配分组"
          @click="applySummaryFilter('linked')"
        >
          <StatCard
            title="已匹配"
            :value="groupSummary.linked_group_count"
            :icon="MatchedGroupsIcon"
            icon-variant="success"
            :change="matchedGroupRate"
            change-type="neutral"
          />
        </button>
        <button
          type="button"
          class="sp-summary-filter"
          :class="{ active: isSummaryFilterActive('unlinked') }"
          :aria-pressed="isSummaryFilterActive('unlinked')"
          :disabled="loading"
          aria-label="过滤待匹配分组"
          @click="applySummaryFilter('unlinked')"
        >
          <StatCard
            title="待匹配"
            :value="groupSummary.unlinked_group_count"
            :icon="UnmatchedGroupsIcon"
            :icon-variant="groupSummary.unlinked_group_count > 0 ? 'warning' : 'primary'"
            :change="unmatchedGroupRate"
            change-type="neutral"
          />
        </button>
        <button
          type="button"
          class="sp-summary-filter"
          :class="{ active: isSummaryFilterActive('inverted') }"
          :aria-pressed="isSummaryFilterActive('inverted')"
          :disabled="loading"
          aria-label="过滤倒挂风险分组"
          @click="applySummaryFilter('inverted')"
        >
          <StatCard
            title="倒挂风险"
            :value="groupSummary.rate_risk_count"
            :icon="RateRiskIcon"
            :icon-variant="groupSummary.rate_risk_count > 0 ? 'danger' : 'success'"
          />
        </button>
      </div>

      <div class="sp-console-panel sp-panel sp-groups-panel">
        <header class="sp-panel-head">
          <div class="sp-panel-title">
            <span class="sp-section-index">01</span>
            <div>
              <h2>上游与本地分组对照</h2>
              <span>共 {{ total }} 条，当前页 {{ items.length }} 条</span>
            </div>
          </div>
          <div class="sp-provider-shortcuts" aria-label="供应商快捷过滤">
            <button
              v-for="option in quickProviderOptions"
              :key="String(option.value)"
              type="button"
              class="sp-provider-shortcut"
              :class="{ active: providerID === option.value }"
              :aria-pressed="providerID === option.value"
              :disabled="loading"
              :title="String(option.label)"
              @click="selectProviderShortcut(option.value)"
            >
              <span>{{ option.label }}</span>
            </button>
          </div>
          <div class="sp-panel-signals" aria-label="当前页状态">
            <span><i class="good"></i>{{ currentPageMatchedCount }} 本页已关联</span>
            <span><i class="warn"></i>{{ currentPageAttentionCount }} 本页需关注</span>
          </div>
        </header>

        <div class="sp-attention-shortcuts" aria-label="待处理快捷过滤">
          <button
            v-for="shortcut in attentionShortcuts"
            :key="shortcut.label"
            type="button"
            class="sp-attention-shortcut"
            :class="{ active: isAttentionShortcutActive(shortcut) }"
            :aria-pressed="isAttentionShortcutActive(shortcut)"
            :disabled="loading"
            @click="applyAttentionShortcut(shortcut)"
          >
            {{ shortcut.label }}
          </button>
        </div>

        <div class="sp-table-shell">
          <DataTable
            :columns="groupColumns"
            :data="items"
            :loading="loading"
            row-key="id"
            server-side-sort
            clickable-rows
            @sort="handleGroupSort"
            @row-click="selected = $event"
          >
            <template #cell-provider_name="{ row: group }">
              <div :class="['sp-supplier-chip', supplierTone(group.provider_id).chip]">
                <i :class="['sp-supplier-dot', supplierTone(group.provider_id).dot]"></i>
                <span class="sp-supplier-name">{{ group.provider_name }}</span>
              </div>
              <div class="sp-sub sp-provider-meta">
                <span :class="['sp-provider-type', supplierTypeTone(group.provider_id)]">{{ supplierTypeLabel(group.provider_id) }}</span>
                <span>#{{ group.provider_id }}</span>
              </div>
            </template>

            <template #cell-name="{ row: group }">
              <div :class="['sp-upstream-group-chip', upstreamGroupTone(group.upstream_group_key).chip]">
                <i
                  :class="['sp-upstream-group-accent', upstreamGroupTone(group.upstream_group_key).accent]"
                  aria-hidden="true"
                ></i>
                <span class="sp-group-name">{{ group.name || '未命名分组' }}</span>
                <span
                  :class="[
                    'sp-inline-platform',
                    group.local_group_platform
                      ? platformTextClass(group.local_group_platform)
                      : 'sp-inline-platform-muted',
                  ]"
                >【{{ upstreamPlatformLabel(group) }}】</span>
              </div>
              <div :class="['sp-sub', 'sp-key', upstreamGroupTone(group.upstream_group_key).meta]">
                {{ group.upstream_group_key }}
              </div>
            </template>

            <template #cell-rate_multiplier="{ row: group }">
              <span :class="['sp-rate-value', platformTextClass(group.local_group_platform || '')]">{{ formatRate(group.rate_multiplier) }}</span>
            </template>

            <template #cell-raw_status="{ row: group }">
              <span class="sp-status" :class="upstreamStatusTone(group)">
                <i></i>{{ upstreamStatusLabel(group) }}
              </span>
            </template>

            <template #cell-monitor_trend="{ row: group }">
              <SupplierGroupAvailabilityTrend
                :row="availabilityTrendFor(group)"
                :loading="availabilityTrendLoading(group)"
                :source-label="availabilityTrendSourceLabel(group)"
                empty-text="-"
                loading-text="加载中"
                label="可用率趋势"
              />
            </template>

            <template #cell-local_group_name="{ row: group }">
              <div v-if="group.local_group_id" class="sp-local-group">
                <GroupBadge
                  :name="group.local_group_name || `本地分组 #${group.local_group_id}`"
                  :platform="groupPlatform(group.local_group_platform)"
                  :show-rate="false"
                  class="sp-local-group-badge"
                />
                <div class="sp-sub">
                  <span :class="platformTextClass(group.local_group_platform || '')">
                    {{ platformLabel(group.local_group_platform) }}
                  </span>
                  <span>#{{ group.local_group_id }}</span>
                </div>
              </div>
              <button v-else type="button" class="sp-inline-empty" @click.stop="openMappingDialog(group)">
                <Icon name="link" size="sm" />
                <span>未匹配</span>
              </button>
            </template>

            <template #cell-local_rate_multiplier="{ row: group }">
              <span v-if="group.local_rate_multiplier != null" class="sp-rate-value" :class="platformTextClass(group.local_group_platform || '')">
                {{ formatRate(group.local_rate_multiplier) }}
              </span>
              <span v-else class="sp-empty-value">-</span>
            </template>

            <template #cell-auto_match_status="{ row: group }">
              <div class="sp-match-state-stack">
                <span class="sp-match-state" :class="matchStatusTone(group)">
                  <i></i>{{ matchStatusLabel(group) }}
                </span>
                <button
                  v-if="group.name_change_pending"
                  type="button"
                  class="sp-name-change-link"
                  @click.stop="selected = group"
                >名称已变化</button>
              </div>
            </template>

			<template #cell-rate_guard_status="{ row: group }">
				<div class="sp-guard-state-stack" :title="rateGuardStatus(group).title">
					<span class="sp-guard-state" :class="rateGuardStatus(group).tone">
						<Icon name="shield" size="xs" />
						{{ rateGuardStatus(group).label }}
					</span>
					<small v-if="rateGuardStatus(group).detail">{{ rateGuardStatus(group).detail }}</small>
				</div>
			</template>

            <template #cell-rate_delta="{ row: group }">
              <div v-if="rateInsight(group).delta != null" class="sp-rate-delta-cell" :class="rateInsight(group).code">
                <strong>{{ formatSupplierGroupRateDelta(group.local_rate_multiplier, group.rate_multiplier) }}</strong>
              </div>
              <span v-else class="sp-empty-value">-</span>
            </template>

            <template #cell-account_count="{ row: group }">
              <button
                type="button"
                class="sp-account-count"
                :title="`查看 ${group.account_count} 个绑定账号`"
                @click.stop="selected = group"
              >
                <Icon name="users" size="sm" />
                <strong>{{ group.account_count }}</strong>
              </button>
            </template>

            <template #cell-rate_status="{ row: group }">
              <span class="sp-rate-status" :class="rateInsight(group).code">
                <i></i>{{ rateInsight(group).label }}
              </span>
            </template>

            <template #cell-actions="{ row: group }">
              <div class="sp-row-actions" @click.stop>
                <template v-if="!group.active">
                  <button
                    type="button"
                    class="sp-row-action danger"
                    :disabled="deletingGroupID === group.id || !canDeleteSupplierGroup(group)"
                    :title="supplierGroupDeleteHint(group)"
                    @click="openDeleteGroupDialog(group)"
                  >
                    <Icon name="x" size="sm" />
                    <span>{{ deletingGroupID === group.id ? '删除中' : '删除记录' }}</span>
                  </button>
                </template>
                <template v-else>
                <template v-if="!group.local_group_id">
                  <button type="button" class="sp-row-action primary" title="匹配本地分组" @click="openMappingDialog(group)">
                    <Icon name="link" size="sm" />
                    <span>匹配分组</span>
                  </button>
                  <button type="button" class="sp-row-action" title="新建本地分组" @click="openCreateDialog(group)">
                    <Icon name="plus" size="sm" />
                    <span>新建分组</span>
                  </button>
                </template>
                <template v-else>
					<button
						v-if="canManageManualRateGuard(group)"
						type="button"
						class="sp-row-action guard"
						:class="{ active: group.rate_guard_selected }"
						:disabled="guardUpdatingGroupID === group.id || (!group.rate_guard_selected && !rateGuardEligible(group))"
						:title="group.rate_guard_selected ? '取消人工倍率守护' : (hasOtherRateGuard(group) ? '切换为该本地分组的倍率守护来源' : '设为该本地分组的倍率守护来源')"
						@click="toggleRateGuard(group)"
					>
						<Icon name="shield" size="sm" />
						<span>{{ group.rate_guard_selected ? '取消守护' : (hasOtherRateGuard(group) ? '切换为守护' : '设为守护') }}</span>
					</button>
					<button
						v-if="group.rate_guard_selected"
						type="button"
						class="sp-row-action"
						:class="{ active: group.rate_guard_ignored }"
						:disabled="guardIgnoreUpdatingGroupID === group.id"
						:title="group.rate_guard_ignored ? '恢复该分组的自动倍率守护' : '忽略该分组的自动倍率守护'"
						@click="group.rate_guard_ignored ? toggleRateGuardIgnored(group) : (rateGuardIgnoreTarget = group)"
					>
						<Icon :name="group.rate_guard_ignored ? 'refresh' : 'x'" size="sm" />
						<span>{{ group.rate_guard_ignored ? '恢复守护' : '忽略守护' }}</span>
					</button>
                  <button type="button" class="sp-row-action primary" title="修改本地分组倍率" @click="openRateDialog(group)">
                    <Icon name="edit" size="sm" />
                    <span>调倍率</span>
                  </button>
                  <button type="button" class="sp-row-action" title="更换关联的本地分组" @click="openMappingDialog(group)">
                    <Icon name="refresh" size="sm" />
                    <span>更换本地分组</span>
                  </button>
                  <button type="button" class="sp-row-action danger" title="取消本地分组关联" @click="unmatchTarget = group">
                    <Icon name="x" size="sm" />
                    <span>取消关联</span>
                  </button>
                </template>
                <button
                  type="button"
                  class="sp-row-action"
                  :class="{ active: group.auto_match_ignored }"
                  :disabled="policyUpdatingGroupID === group.id"
                  :title="group.auto_match_ignored ? '重新允许自动匹配' : '忽略该分组的自动匹配'"
                  @click="toggleAutoMatchIgnored(group)"
                >
                  <Icon :name="group.auto_match_ignored ? 'refresh' : 'x'" size="sm" />
                  <span>{{ group.auto_match_ignored ? '允许自动' : '忽略自动' }}</span>
                </button>
                </template>
              </div>
            </template>

            <template #empty>
              暂无可用上游分组，请先在供应商列表执行分组同步。
            </template>
          </DataTable>
        </div>

        <Pagination
          v-if="total > 0"
          class="sp-data-pagination"
          :page="page"
          :total="total"
          :page-size="pageSize"
          :show-page-size-selector="true"
          @update:page="handleGroupPageChange"
          @update:pageSize="handleGroupPageSizeChange"
        />
      </div>
    </div>

    <SupplierDrawer
      :show="Boolean(selected)"
      :title="selected?.name || selected?.upstream_group_key || ''"
      eyebrow="GROUP COMPARISON"
      @close="selected = null"
    >
      <template v-if="selected">
        <div class="sp-drawer-summary">
          <span class="sp-rate-status" :class="rateInsight(selected).code">
            <i></i>{{ rateInsight(selected).label }}
          </span>
          <strong>{{ formatSupplierGroupRateDelta(selected.local_rate_multiplier, selected.rate_multiplier) }}</strong>
          <small>价差</small>
        </div>
        <div v-if="selected.name_change_pending" class="sp-name-change-alert">
          <div>
            <span>上游名称已变化</span>
            <strong>{{ selected.matched_upstream_name || '未记录' }} → {{ selected.name || selected.upstream_group_key }}</strong>
            <small>当前本地名称：{{ selected.local_group_name || '未匹配' }}</small>
          </div>
          <div class="sp-name-change-actions">
            <button
              type="button"
              class="sp-row-action"
              :disabled="resolvingNameGroupID === selected.id"
              @click="resolveNameChange(selected, 'keep_local')"
            >保持本地名称</button>
            <button
              type="button"
              class="sp-row-action primary"
              :disabled="resolvingNameGroupID === selected.id"
              @click="resolveNameChange(selected, 'sync_local_name')"
            >同步本地名称</button>
          </div>
        </div>
        <div class="sp-detail-grid">
          <div class="sp-detail-cell"><span>供应商</span><b>{{ selected.provider_name }}</b></div>
          <div class="sp-detail-cell"><span>绑定账号</span><b>{{ selected.account_count }}</b></div>
          <div class="sp-detail-cell"><span>上游分组</span><b>{{ selected.name || '未命名分组' }}</b></div>
          <div class="sp-detail-cell"><span>上游 Key</span><b>{{ selected.upstream_group_key }}</b></div>
          <div class="sp-detail-cell"><span>上游倍率</span><b>{{ formatRate(selected.rate_multiplier) }}</b></div>
          <div class="sp-detail-cell"><span>上游状态</span><b>{{ upstreamStatusLabel(selected) }}</b></div>
          <div class="sp-detail-cell">
            <span>本地分组</span>
            <GroupBadge
              v-if="selected.local_group_id"
              :name="selected.local_group_name || `本地分组 #${selected.local_group_id}`"
              :platform="groupPlatform(selected.local_group_platform)"
              :show-rate="false"
              class="sp-detail-group-badge"
            />
            <b v-else>未匹配</b>
          </div>
          <div class="sp-detail-cell"><span>本地倍率</span><b>{{ formatRate(selected.local_rate_multiplier) }}</b></div>
			<div class="sp-detail-cell">
				<span>倍率守护</span>
				<div class="sp-guard-detail" :title="rateGuardStatus(selected).title">
					<b class="sp-guard-state" :class="rateGuardStatus(selected).tone">{{ rateGuardStatus(selected).label }}</b>
					<small v-if="rateGuardStatus(selected).detail">{{ rateGuardStatus(selected).detail }}</small>
				</div>
			</div>
			<div class="sp-detail-cell"><span>分组同步</span><b>{{ selected.group_sync_status || 'never' }}</b></div>
          <div class="sp-detail-cell sp-detail-wide"><span>最近同步</span><b>{{ formatTime(selected.last_seen_at) }}</b></div>
        </div>
      </template>
    </SupplierDrawer>

    <BaseDialog :show="Boolean(mappingTarget)" title="匹配本地分组" width="normal" @close="closeMappingDialog">
      <template v-if="mappingTarget">
        <div class="sp-dialog-context match">
          <span>{{ mappingTarget.provider_name }}</span>
          <strong>{{ mappingTarget.name || mappingTarget.upstream_group_key }}</strong>
          <small>上游倍率 {{ formatRate(mappingTarget.rate_multiplier) }}</small>
        </div>
        <div class="sp-field">
          <span>本地分组</span>
          <Select
            v-model="mappingLocalGroupID"
            :options="localGroupOptions"
            :searchable="true"
            search-placeholder="搜索本地分组"
            placeholder="请选择本地分组"
          />
        </div>
        <div v-if="mappingPreview" class="sp-match-preview">
          <div class="sp-match-preview-card platform"><span>平台</span><strong :class="platformTextClass(mappingPreview.platform)">{{ platformLabel(mappingPreview.platform) }}</strong></div>
          <div class="sp-match-preview-card rate"><span>当前倍率</span><strong>{{ formatRate(mappingPreview.rate_multiplier) }}</strong></div>
          <div class="sp-match-preview-card delta"><span>价差</span><strong>{{ mappingRateDeltaPreview }}</strong></div>
        </div>
      </template>
      <template #footer>
        <div class="sp-dialog-actions">
          <button type="button" class="sp-button secondary" :disabled="savingMapping" @click="closeMappingDialog">取消</button>
          <button type="button" class="sp-button sp-dialog-primary match" :disabled="savingMapping || !mappingLocalGroupID" @click="saveMapping">
            <Icon name="link" size="sm" />
            <span>{{ savingMapping ? '保存中' : '保存匹配' }}</span>
          </button>
        </div>
      </template>
    </BaseDialog>

    <BaseDialog :show="Boolean(createTarget)" title="新建本地分组" width="normal" @close="closeCreateDialog">
      <template v-if="createTarget">
        <div class="sp-dialog-context create">
          <span>{{ createTarget.provider_name }}</span>
          <strong>{{ createTarget.name || createTarget.upstream_group_key }}</strong>
          <small>创建成功后将自动完成匹配</small>
        </div>
        <div class="sp-dialog-form">
          <Input v-model="newGroupName" label="本地分组名称" placeholder="输入本地分组名称" />
          <div class="sp-field">
            <span>平台</span>
            <Select v-model="newGroupPlatform" :options="platformOptions" :searchable="false" />
          </div>
          <Input v-model="newGroupRate" type="number" label="本地分组倍率" placeholder="输入大于 0 的倍率" />
        </div>
        <div class="sp-rate-recommendation create">
          <span>上游 {{ formatRate(createTarget.rate_multiplier) }}</span>
          <Icon name="chevronRight" size="sm" />
          <strong>建议不低于 {{ formatRate(suggestedLocalRate(createTarget)) }}</strong>
        </div>
      </template>
      <template #footer>
        <div class="sp-dialog-actions">
          <button type="button" class="sp-button secondary" :disabled="creatingLocalGroup" @click="closeCreateDialog">取消</button>
          <button type="button" class="sp-button sp-dialog-primary create" :disabled="creatingLocalGroup" @click="createLocalGroup">
            <Icon name="plus" size="sm" />
            <span>{{ creatingLocalGroup ? '创建中' : '创建并匹配' }}</span>
          </button>
        </div>
      </template>
    </BaseDialog>

    <BaseDialog :show="Boolean(rateTarget)" title="修改本地分组倍率" width="narrow" @close="closeRateDialog">
      <template v-if="rateTarget">
        <div class="sp-dialog-context rate">
          <span>{{ rateTarget.local_group_name }}</span>
          <strong>当前 {{ formatRate(rateTarget.local_rate_multiplier) }}</strong>
          <small>对应上游倍率 {{ formatRate(rateTarget.rate_multiplier) }}</small>
        </div>
        <Input v-model="localRateInput" type="number" label="新倍率" placeholder="输入大于 0 的倍率" @enter="saveLocalRate" />
        <div :class="['sp-rate-recommendation', 'rate', localRateDeltaTone]">
          <span>修改后价差</span>
          <strong>{{ localRateDeltaPreview }}</strong>
        </div>
      </template>
      <template #footer>
        <div class="sp-dialog-actions">
          <button type="button" class="sp-button secondary" :disabled="savingLocalRate" @click="closeRateDialog">取消</button>
          <button type="button" class="sp-button sp-dialog-primary rate" :disabled="savingLocalRate" @click="saveLocalRate">
            <Icon name="edit" size="sm" />
            <span>{{ savingLocalRate ? '保存中' : '保存倍率' }}</span>
          </button>
        </div>
      </template>
    </BaseDialog>

    <BaseDialog :show="rateGuardChangeLogsVisible" title="分组倍率变更日志" width="extra-wide" @close="closeRateGuardChangeLogs">
      <div class="sp-change-log-dialog log">
        <div class="sp-table-region sp-change-log-table-region">
          <DataTable
            :columns="rateGuardChangeLogColumns"
            :data="rateGuardChangeLogs"
            :loading="rateGuardChangeLogsLoading"
            row-key="id"
            :sticky-actions-column="false"
          >
            <template #cell-status="{ row: changeLog }">
              <button
                v-if="changeLog.status === 'pending'"
                class="sp-link-button sp-change-log-status-action"
                type="button"
                :disabled="handlingRateGuardChangeLogID === changeLog.id"
                @click="handleRateGuardChangeLog(changeLog)"
              >
                {{ handlingRateGuardChangeLogID === changeLog.id ? '确认中' : '待处理' }}
              </button>
              <span v-else class="sp-status good">已处理</span>
            </template>
            <template #cell-local_group_name="{ row: changeLog }">
              <strong class="sp-change-log-local-group">
                {{ changeLog.local_group_name || `本地分组 ${changeLog.local_group_id}` }}
              </strong>
            </template>
            <template #cell-upstream_group_name="{ row: changeLog }">
              <strong class="sp-change-log-upstream-group">
                {{ changeLog.upstream_group_name || '未命名分组' }}
                <span v-if="changeLog.upstream_group_key" class="sp-change-log-key">#{{ changeLog.upstream_group_key }}</span>
              </strong>
            </template>
            <template #cell-old_rate="{ row: changeLog }">
              <span class="sp-change-log-rate sp-change-log-old-rate">{{ formatRate(changeLog.old_rate) }}</span>
            </template>
            <template #cell-new_rate="{ row: changeLog }">
              <span class="sp-change-log-rate sp-change-log-new-rate">{{ formatRate(changeLog.new_rate) }}</span>
            </template>
            <template #cell-changed_at="{ row: changeLog }">
              <span class="sp-change-log-time">{{ formatTime(changeLog.changed_at) }}</span>
            </template>
            <template #empty>暂无分组倍率变更记录。</template>
          </DataTable>
          <Pagination
            v-if="rateGuardChangeLogTotal > 0"
            class="sp-change-log-pagination"
            :page="rateGuardChangeLogPage"
            :total="rateGuardChangeLogTotal"
            :page-size="rateGuardChangeLogPageSize"
            :show-page-size-selector="false"
            @update:page="changeRateGuardChangeLogPage"
          />
        </div>
      </div>
      <template #footer>
        <button class="sp-button primary sp-dialog-primary log" type="button" @click="closeRateGuardChangeLogs">关闭</button>
      </template>
    </BaseDialog>

    <ConfirmDialog
      :show="Boolean(rateGuardIgnoreTarget)"
      title="忽略自动倍率守护"
      :message="`忽略后，${rateGuardIgnoreTarget?.name || rateGuardIgnoreTarget?.upstream_group_key || '该上游分组'} 将保留守护来源，仅暂停自动倍率调整。`"
      confirm-text="忽略守护"
      cancel-text="取消"
      danger
      @confirm="confirmRateGuardIgnore"
      @cancel="rateGuardIgnoreTarget = null"
    />

    <ConfirmDialog
      :show="Boolean(unmatchTarget)"
      title="取消本地分组关联"
      :message="`取消后，${unmatchTarget?.name || unmatchTarget?.upstream_group_key || '该上游分组'} 将保持未匹配，不再参与自动匹配。`"
      confirm-text="取消关联"
      cancel-text="取消"
      danger
      @confirm="removeMapping"
      @cancel="unmatchTarget = null"
    />

    <ConfirmDialog
      :show="Boolean(deleteTarget)"
      title="删除失效分组记录"
      :message="`仅删除本地保存的失效上游分组记录，不会删除上游系统内容。删除 ${deleteTarget?.name || deleteTarget?.upstream_group_key || '该分组'} 后不可恢复。`"
      confirm-text="删除记录"
      cancel-text="取消"
      danger
      @confirm="confirmDeleteGroup"
      @cancel="deleteTarget = null"
    />
  </SupplierModuleLayout>
</template>

<script setup lang="ts">
import { computed, h, nextTick, onMounted, ref, watch } from 'vue'
import { adminAPI } from '@/api/admin'
import {
  listRateGuardChangeLogs,
  markRateGuardChangeLogHandled,
  type SupplierRateGuardChangeLog,
} from '@/api/admin/supplierAutomation'
import {
  autoMatchSupplierGroups,
  deleteSupplierGroup,
  listSupplierGroupHealthTrends,
  listSupplierGroups,
  resolveSupplierGroupNameChange,
  updateSupplierGroupAutoMatchPolicy,
	updateSupplierGroupRateGuard,
	updateSupplierGroupRateGuardIgnore,
  updateSupplierGroupMapping,
  type SupplierProviderGroup,
  type SupplierProviderGroupSummary,
} from '@/api/admin/supplierProviderData'
import supplierProvidersAPI, { type SupplierProvider } from '@/api/admin/supplierProviders'
import { SupplierDrawer, SupplierModuleLayout } from '@/components/admin/supplier-management'
import SupplierGroupAvailabilityTrend from '@/components/admin/supplier-management/SupplierGroupAvailabilityTrend.vue'
import BaseDialog from '@/components/common/BaseDialog.vue'
import ConfirmDialog from '@/components/common/ConfirmDialog.vue'
import DataTable from '@/components/common/DataTable.vue'
import GroupBadge from '@/components/common/GroupBadge.vue'
import Input from '@/components/common/Input.vue'
import Pagination from '@/components/common/Pagination.vue'
import Select, { type SelectOption } from '@/components/common/Select.vue'
import StatCard from '@/components/common/StatCard.vue'
import type { Column } from '@/components/common/types'
import Icon from '@/components/icons/Icon.vue'
import { useAppStore } from '@/stores/app'
import type { AdminGroup, GroupPlatform } from '@/types'
import { platformTextClass } from '@/utils/platformColors'
import {
  buildSupplierGroupMonitorTrendIndex,
  normalizeSupplierGroupMonitorKey,
  type SupplierGroupMonitorTrendRow,
} from '@/utils/supplierGroupMonitorTrend'
import { buildSupplierGroupHealthTrendIndex } from '@/utils/supplierGroupHealthTrend'
import {
  formatSupplierGroupRateDelta,
  getSupplierGroupRateInsight,
  type SupplierGroupRateInsight,
} from './supplierGroupRates'

const EMPTY_GROUP_SUMMARY: SupplierProviderGroupSummary = {
  group_count: 0,
  account_count: 0,
  linked_group_count: 0,
  unlinked_group_count: 0,
  rate_risk_count: 0,
}

type SummaryFilter = 'all' | 'linked' | 'unlinked' | 'inverted'

interface AttentionShortcut {
  label: string
  matchStatus: string
  rateStatus: string
}

const PLATFORM_LABELS: Record<string, string> = {
  anthropic: 'Anthropic',
  openai: 'OpenAI',
  gemini: 'Gemini',
  antigravity: 'Antigravity',
  grok: 'Grok',
}

const SUPPLIER_TYPE_LABELS: Record<string, string> = {
  sub2api: 'Sub2API',
  newapi: 'NewAPI',
}

const SUPPLIER_TYPE_TONES: Record<string, string> = {
  sub2api: 'bg-teal-500/10 text-teal-600 dark:text-teal-400',
  newapi: 'bg-blue-500/10 text-blue-600 dark:text-blue-400',
}
const SUPPLIER_TYPE_TONE_DEFAULT = 'bg-slate-500/10 text-slate-600 dark:text-slate-400'

const SUPPLIER_TONES = [
  { chip: 'border-sky-500/30 bg-sky-500/10 text-sky-700 dark:text-sky-300', dot: 'bg-sky-500', meta: 'text-sky-600 dark:text-sky-400' },
  { chip: 'border-orange-500/30 bg-orange-500/10 text-orange-700 dark:text-orange-300', dot: 'bg-orange-500', meta: 'text-orange-600 dark:text-orange-400' },
  { chip: 'border-emerald-500/30 bg-emerald-500/10 text-emerald-700 dark:text-emerald-300', dot: 'bg-emerald-500', meta: 'text-emerald-600 dark:text-emerald-400' },
  { chip: 'border-violet-500/30 bg-violet-500/10 text-violet-700 dark:text-violet-300', dot: 'bg-violet-500', meta: 'text-violet-600 dark:text-violet-400' },
  { chip: 'border-rose-500/30 bg-rose-500/10 text-rose-700 dark:text-rose-300', dot: 'bg-rose-500', meta: 'text-rose-600 dark:text-rose-400' },
  { chip: 'border-cyan-500/30 bg-cyan-500/10 text-cyan-700 dark:text-cyan-300', dot: 'bg-cyan-500', meta: 'text-cyan-600 dark:text-cyan-400' },
  { chip: 'border-amber-500/30 bg-amber-500/10 text-amber-700 dark:text-amber-300', dot: 'bg-amber-500', meta: 'text-amber-600 dark:text-amber-400' },
  { chip: 'border-indigo-500/30 bg-indigo-500/10 text-indigo-700 dark:text-indigo-300', dot: 'bg-indigo-500', meta: 'text-indigo-600 dark:text-indigo-400' },
  { chip: 'border-lime-500/30 bg-lime-500/10 text-lime-700 dark:text-lime-300', dot: 'bg-lime-500', meta: 'text-lime-600 dark:text-lime-400' },
  { chip: 'border-fuchsia-500/30 bg-fuchsia-500/10 text-fuchsia-700 dark:text-fuchsia-300', dot: 'bg-fuchsia-500', meta: 'text-fuchsia-600 dark:text-fuchsia-400' },
  { chip: 'border-teal-500/30 bg-teal-500/10 text-teal-700 dark:text-teal-300', dot: 'bg-teal-500', meta: 'text-teal-600 dark:text-teal-400' },
  { chip: 'border-red-500/30 bg-red-500/10 text-red-700 dark:text-red-300', dot: 'bg-red-500', meta: 'text-red-600 dark:text-red-400' },
] as const

const UPSTREAM_GROUP_TONES = [
  { chip: 'bg-blue-500/10 text-blue-800 dark:text-blue-200', accent: 'bg-blue-500', meta: 'text-blue-600 dark:text-blue-400' },
  { chip: 'bg-orange-500/10 text-orange-800 dark:text-orange-200', accent: 'bg-orange-500', meta: 'text-orange-600 dark:text-orange-400' },
  { chip: 'bg-emerald-500/10 text-emerald-800 dark:text-emerald-200', accent: 'bg-emerald-500', meta: 'text-emerald-600 dark:text-emerald-400' },
  { chip: 'bg-violet-500/10 text-violet-800 dark:text-violet-200', accent: 'bg-violet-500', meta: 'text-violet-600 dark:text-violet-400' },
  { chip: 'bg-rose-500/10 text-rose-800 dark:text-rose-200', accent: 'bg-rose-500', meta: 'text-rose-600 dark:text-rose-400' },
  { chip: 'bg-cyan-500/10 text-cyan-800 dark:text-cyan-200', accent: 'bg-cyan-500', meta: 'text-cyan-600 dark:text-cyan-400' },
  { chip: 'bg-amber-500/10 text-amber-800 dark:text-amber-200', accent: 'bg-amber-500', meta: 'text-amber-600 dark:text-amber-400' },
  { chip: 'bg-indigo-500/10 text-indigo-800 dark:text-indigo-200', accent: 'bg-indigo-500', meta: 'text-indigo-600 dark:text-indigo-400' },
  { chip: 'bg-lime-500/10 text-lime-800 dark:text-lime-200', accent: 'bg-lime-500', meta: 'text-lime-600 dark:text-lime-400' },
  { chip: 'bg-fuchsia-500/10 text-fuchsia-800 dark:text-fuchsia-200', accent: 'bg-fuchsia-500', meta: 'text-fuchsia-600 dark:text-fuchsia-400' },
  { chip: 'bg-teal-500/10 text-teal-800 dark:text-teal-200', accent: 'bg-teal-500', meta: 'text-teal-600 dark:text-teal-400' },
  { chip: 'bg-red-500/10 text-red-800 dark:text-red-200', accent: 'bg-red-500', meta: 'text-red-600 dark:text-red-400' },
] as const

const UpstreamGroupsIcon = () => h(Icon, { name: 'filter', size: 'lg' })
const MatchedGroupsIcon = () => h(Icon, { name: 'link', size: 'lg' })
const UnmatchedGroupsIcon = () => h(Icon, { name: 'exclamationCircle', size: 'lg' })
const RateRiskIcon = () => h(Icon, { name: 'exclamationCircle', size: 'lg' })

const appStore = useAppStore()
const providers = ref<SupplierProvider[]>([])
const localGroups = ref<AdminGroup[]>([])
const items = ref<SupplierProviderGroup[]>([])
const groupSummary = ref<SupplierProviderGroupSummary>({ ...EMPTY_GROUP_SUMMARY })
const selected = ref<SupplierProviderGroup | null>(null)
const mappingTarget = ref<SupplierProviderGroup | null>(null)
const createTarget = ref<SupplierProviderGroup | null>(null)
const rateTarget = ref<SupplierProviderGroup | null>(null)
const unmatchTarget = ref<SupplierProviderGroup | null>(null)
const deleteTarget = ref<SupplierProviderGroup | null>(null)
const rateGuardIgnoreTarget = ref<SupplierProviderGroup | null>(null)
const mappingLocalGroupID = ref<number | null>(null)
const newGroupName = ref('')
const newGroupPlatform = ref<string>('openai')
const newGroupRate = ref('')
const localRateInput = ref('')
const total = ref(0)
const loading = ref(false)
const monitorTrendIndex = ref<Map<string, SupplierGroupMonitorTrendRow>>(new Map())
const monitorLoading = ref(false)
const healthTrendIndex = ref<Map<number, SupplierGroupMonitorTrendRow>>(new Map())
const healthTrendLoading = ref(false)
const savingMapping = ref(false)
const creatingLocalGroup = ref(false)
const deletingGroupID = ref<number | null>(null)
const savingLocalRate = ref(false)
const autoMatching = ref(false)
const policyUpdatingGroupID = ref<number | null>(null)
const guardUpdatingGroupID = ref<number | null>(null)
const guardIgnoreUpdatingGroupID = ref<number | null>(null)
const resolvingNameGroupID = ref<number | null>(null)
const rateGuardChangeLogsVisible = ref(false)
const rateGuardChangeLogsLoading = ref(false)
const rateGuardChangeLogs = ref<SupplierRateGuardChangeLog[]>([])
const rateGuardChangeLogTotal = ref(0)
const pendingRateGuardChangeLogCount = ref(0)
const rateGuardChangeLogPage = ref(1)
const rateGuardChangeLogPageSize = ref(20)
const handlingRateGuardChangeLogID = ref<number | null>(null)
const error = ref('')
const page = ref(1)
const pageSize = ref(20)
const sortBy = ref('')
const sortOrder = ref<'asc' | 'desc'>('asc')
const providerID = ref(0)
const search = ref('')
const platformFilter = ref('')
const matchStatusFilter = ref('')
const rateStatusFilter = ref('')
let searchTimer: number | undefined
let monitorTrendRequestID = 0
let healthTrendRequestID = 0
let suppressFilterWatch = false

const DEFAULT_PROVIDER_ID = 0
const providerOptions = computed<SelectOption[]>(() => [
  { value: 0, label: '全部供应商' },
  ...providers.value.map(provider => ({ value: provider.id, label: provider.name })),
])
const quickProviderOptions = computed<SelectOption[]>(() => providerOptions.value)
const platformFilterOptions: SelectOption[] = [
  { value: '', label: '全部平台' },
  { value: 'anthropic', label: 'Anthropic' },
  { value: 'openai', label: 'OpenAI' },
  { value: 'gemini', label: 'Gemini' },
  { value: 'antigravity', label: 'Antigravity' },
  { value: 'grok', label: 'Grok' },
]
const matchStatusFilterOptions: SelectOption[] = [
  { value: '', label: '全部匹配状态' },
  { value: 'linked', label: '已匹配' },
  { value: 'unlinked', label: '待匹配' },
  { value: 'auto_matched', label: '自动匹配' },
  { value: 'manual', label: '人工匹配' },
  { value: 'ambiguous', label: '名称冲突' },
  { value: 'ignored', label: '已忽略' },
  { value: 'name_changed', label: '名称变化' },
]
const rateStatusFilterOptions: SelectOption[] = [
  { value: '', label: '全部倍率状态' },
  { value: 'normal', label: '正常' },
  { value: 'low', label: '收益偏低' },
  { value: 'equal', label: '倍率持平' },
  { value: 'inverted', label: '倒挂风险' },
  { value: 'inactive', label: '本地停用' },
  { value: 'invalid', label: '数据异常' },
]
const attentionShortcuts: AttentionShortcut[] = [
  { label: '名称冲突', matchStatus: 'ambiguous', rateStatus: '' },
  { label: '名称变化', matchStatus: 'name_changed', rateStatus: '' },
  { label: '已忽略', matchStatus: 'ignored', rateStatus: '' },
  { label: '收益偏低', matchStatus: '', rateStatus: 'low' },
  { label: '倒挂风险', matchStatus: '', rateStatus: 'inverted' },
]
const supplierIDs = computed(() => [...new Set(providers.value.map(provider => provider.id))].sort((left, right) => left - right))
const localGroupOptions = computed<SelectOption[]>(() => localGroups.value.map(group => ({
  value: group.id,
  label: `${group.name} · ${platformLabel(group.platform)} · ${formatRate(group.rate_multiplier)}`,
})))
const platformOptions: SelectOption[] = Object.entries(PLATFORM_LABELS).map(([value, label]) => ({ value, label }))
const groupColumns: Column[] = [
  { key: 'provider_name', label: '供应商', sortable: true, class: 'min-w-[150px]' },
  { key: 'name', label: '上游分组', sortable: true, class: 'min-w-[190px]' },
  { key: 'rate_multiplier', label: '上游倍率', sortable: true, class: 'min-w-[96px]' },
  { key: 'raw_status', label: '上游状态', class: 'min-w-[105px]' },
  { key: 'monitor_trend', label: '可用率趋势', class: 'min-w-[160px]' },
  { key: 'local_group_name', label: '匹配本地分组', sortable: true, class: 'min-w-[190px]' },
  { key: 'auto_match_status', label: '匹配状态', class: 'min-w-[120px]' },
	{ key: 'rate_guard_status', label: '倍率守护', class: 'min-w-[180px]' },
  { key: 'local_rate_multiplier', label: '本地分组倍率', sortable: true, class: 'min-w-[110px]' },
  { key: 'rate_delta', label: '价差', class: 'min-w-[110px]' },
  { key: 'account_count', label: '绑定账号', sortable: true, class: 'min-w-[90px]' },
  { key: 'rate_status', label: '倍率状态', class: 'min-w-[110px]' },
  { key: 'actions', label: '操作', class: 'min-w-[270px]' },
]
const rateGuardChangeLogColumns: Column[] = [
  { key: 'status', label: '处理状态', class: 'min-w-[100px]' },
  { key: 'local_group_name', label: '本地分组', class: 'min-w-[150px]' },
  { key: 'upstream_group_name', label: '上游分组', class: 'min-w-[170px]' },
  { key: 'old_rate', label: '原倍率', class: 'min-w-[90px]' },
  { key: 'new_rate', label: '新倍率', class: 'min-w-[90px]' },
  { key: 'changed_at', label: '修改时间', class: 'min-w-[170px]' },
]
const canResetFilters = computed(() => (
  providerID.value !== DEFAULT_PROVIDER_ID
  || search.value.trim() !== ''
  || platformFilter.value !== ''
  || matchStatusFilter.value !== ''
  || rateStatusFilter.value !== ''
))
const matchedGroupRate = computed(() => percentage(groupSummary.value.linked_group_count, groupSummary.value.group_count))
const unmatchedGroupRate = computed(() => percentage(groupSummary.value.unlinked_group_count, groupSummary.value.group_count))
const currentPageMatchedCount = computed(() => items.value.filter(group => group.local_group_id).length)
const currentPageAttentionCount = computed(() => items.value.filter(group => {
  const code = rateInsight(group).code
  return code !== 'normal'
}).length)
const mappingPreview = computed(() => localGroups.value.find(group => group.id === Number(mappingLocalGroupID.value)) ?? null)
const mappingRateDeltaPreview = computed(() => {
  if (!mappingTarget.value || !mappingPreview.value) return '-'
  return formatSupplierGroupRateDelta(mappingPreview.value.rate_multiplier, mappingTarget.value.rate_multiplier)
})
const localRateDeltaPreview = computed(() => {
  if (!rateTarget.value) return '-'
  return formatSupplierGroupRateDelta(localRateInput.value, rateTarget.value.rate_multiplier)
})
const localRateDeltaTone = computed<'neutral' | 'danger' | 'warning' | 'success'>(() => {
  if (!rateTarget.value) return 'neutral'
  const localRate = Number(localRateInput.value)
  const upstreamRate = Number(rateTarget.value.rate_multiplier)
  if (!Number.isFinite(localRate) || localRate <= 0 || !Number.isFinite(upstreamRate)) return 'neutral'
  if (localRate < upstreamRate) return 'danger'
  if (localRate === upstreamRate) return 'warning'
  return 'success'
})
const rateGuardChangeLogTotalPages = computed(() => (
  Math.max(1, Math.ceil(rateGuardChangeLogTotal.value / rateGuardChangeLogPageSize.value))
))

onMounted(async () => {
  try {
    await Promise.all([loadProviders(), loadLocalGroups()])
  } catch (err) {
    appStore.showError(errorMessage(err, '加载筛选选项失败'))
  }
  void loadMonitorTrend()
  await loadGroups()
  await refreshRateGuardChangeLogs()
})

watch([providerID, platformFilter, matchStatusFilter, rateStatusFilter], () => {
  if (suppressFilterWatch) return
  page.value = 1
  void loadGroups()
})

watch(search, () => {
  if (suppressFilterWatch) return
  window.clearTimeout(searchTimer)
  searchTimer = window.setTimeout(() => {
    page.value = 1
    void loadGroups()
  }, 350)
})

function resetGroupFilters() {
  window.clearTimeout(searchTimer)
  suppressFilterWatch = true
  providerID.value = DEFAULT_PROVIDER_ID
  search.value = ''
  platformFilter.value = ''
  matchStatusFilter.value = ''
  rateStatusFilter.value = ''
  page.value = 1
  void nextTick(() => {
    suppressFilterWatch = false
    void loadGroups()
  })
}

function selectProviderShortcut(value: SelectOption['value']) {
  if (typeof value !== 'number' || providerID.value === value) return
  providerID.value = value
}

function isSummaryFilterActive(filter: SummaryFilter): boolean {
  switch (filter) {
    case 'linked':
      return matchStatusFilter.value === 'linked' && rateStatusFilter.value === ''
    case 'unlinked':
      return matchStatusFilter.value === 'unlinked' && rateStatusFilter.value === ''
    case 'inverted':
      return matchStatusFilter.value === '' && rateStatusFilter.value === 'inverted'
    default:
      return matchStatusFilter.value === '' && rateStatusFilter.value === ''
  }
}

function applySummaryFilter(filter: SummaryFilter) {
  const nextFilter = filter !== 'all' && isSummaryFilterActive(filter) ? 'all' : filter
  suppressFilterWatch = true
  matchStatusFilter.value = ''
  rateStatusFilter.value = ''
  if (nextFilter === 'linked' || nextFilter === 'unlinked') {
    matchStatusFilter.value = nextFilter
  } else if (nextFilter === 'inverted') {
    rateStatusFilter.value = nextFilter
  }
  page.value = 1
  void nextTick(() => {
    suppressFilterWatch = false
    void loadGroups()
  })
}

function isAttentionShortcutActive(shortcut: AttentionShortcut): boolean {
  return matchStatusFilter.value === shortcut.matchStatus
    && rateStatusFilter.value === shortcut.rateStatus
}

function applyAttentionShortcut(shortcut: AttentionShortcut) {
  const active = isAttentionShortcutActive(shortcut)
  suppressFilterWatch = true
  matchStatusFilter.value = active ? '' : shortcut.matchStatus
  rateStatusFilter.value = active ? '' : shortcut.rateStatus
  page.value = 1
  void nextTick(() => {
    suppressFilterWatch = false
    void loadGroups()
  })
}

function handleGroupPageChange(nextPage: number) {
  if (nextPage === page.value) return
  page.value = nextPage
  void loadGroups()
}

function handleGroupPageSizeChange(nextPageSize: number) {
  if (nextPageSize === pageSize.value) return
  pageSize.value = nextPageSize
  page.value = 1
  void loadGroups()
}

function handleGroupSort(key: string, order: 'asc' | 'desc') {
  sortBy.value = key
  sortOrder.value = order
  page.value = 1
  void loadGroups()
}

async function loadProviders() {
  const result = await supplierProvidersAPI.list({ page: 1, page_size: 200 })
  providers.value = result.items
}

async function loadLocalGroups() {
  localGroups.value = await adminAPI.groups.getAll()
}

async function refreshAll() {
  try {
    await Promise.all([loadLocalGroups(), loadGroups(), loadRateGuardChangeLogs(), loadMonitorTrend()])
  } catch (err) {
    appStore.showError(errorMessage(err, '刷新分组数据失败'))
  }
}

function unifiedMonitorTrendFor(group: SupplierProviderGroup) {
  const keys = [group.name, group.upstream_group_key, group.local_group_name]
  for (const key of keys) {
    const trend = monitorTrendIndex.value.get(normalizeSupplierGroupMonitorKey(key))
    if (trend) return trend
  }
  return undefined
}

function availabilityTrendFor(group: SupplierProviderGroup) {
  return unifiedMonitorTrendFor(group) ?? healthTrendIndex.value.get(group.id)
}

function availabilityTrendLoading(group: SupplierProviderGroup) {
  return monitorLoading.value || (!unifiedMonitorTrendFor(group) && healthTrendLoading.value)
}

function availabilityTrendSourceLabel(group: SupplierProviderGroup) {
  if (unifiedMonitorTrendFor(group)) return '统一监控'
  return healthTrendIndex.value.has(group.id) ? '供应商账号健康守护' : ''
}

async function loadHealthTrend(groupIDs: number[], requestID = ++healthTrendRequestID) {
  const uniqueIDs = [...new Set(groupIDs.filter(id => Number.isInteger(id) && id > 0))]
  healthTrendIndex.value = new Map()
  if (!uniqueIDs.length) {
    healthTrendLoading.value = false
    return
  }

  healthTrendLoading.value = true
  try {
    const trends = await listSupplierGroupHealthTrends(uniqueIDs, '90m')
    if (requestID !== healthTrendRequestID) return
    healthTrendIndex.value = buildSupplierGroupHealthTrendIndex(trends)
  } catch {
    if (requestID !== healthTrendRequestID) return
    healthTrendIndex.value = new Map()
  } finally {
    if (requestID === healthTrendRequestID) healthTrendLoading.value = false
  }
}

async function loadMonitorTrend(requestID = ++monitorTrendRequestID) {
  monitorLoading.value = true
  try {
    const payload = await adminAPI.groups.getUpstreamMonitorStatus({ period: '90m', board: 'hot' })
    if (requestID !== monitorTrendRequestID) return
    monitorTrendIndex.value = buildSupplierGroupMonitorTrendIndex(payload)
  } catch (err) {
    if (requestID !== monitorTrendRequestID) return
    monitorTrendIndex.value = new Map()
    appStore.showError(errorMessage(err, '加载可用率趋势失败'))
  } finally {
    if (requestID === monitorTrendRequestID) {
      monitorLoading.value = false
    }
  }
}
async function loadRateGuardChangeLogs() {
  const result = await listRateGuardChangeLogs({
    page: rateGuardChangeLogPage.value,
    page_size: rateGuardChangeLogPageSize.value,
  })
  rateGuardChangeLogs.value = result.items
  rateGuardChangeLogTotal.value = result.total
  pendingRateGuardChangeLogCount.value = result.pending_count
  rateGuardChangeLogPage.value = result.page
  rateGuardChangeLogPageSize.value = result.page_size
}

async function openRateGuardChangeLogs() {
  rateGuardChangeLogsVisible.value = true
  await refreshRateGuardChangeLogs()
}

function closeRateGuardChangeLogs() {
  rateGuardChangeLogsVisible.value = false
}

async function changeRateGuardChangeLogPage(page: number) {
  rateGuardChangeLogPage.value = Math.min(Math.max(1, page), rateGuardChangeLogTotalPages.value)
  await refreshRateGuardChangeLogs()
}

async function refreshRateGuardChangeLogs() {
  rateGuardChangeLogsLoading.value = true
  try {
    await loadRateGuardChangeLogs()
  } catch (err) {
    appStore.showError(errorMessage(err, '加载倍率守护变更日志失败'))
  } finally {
    rateGuardChangeLogsLoading.value = false
  }
}

async function handleRateGuardChangeLog(changeLog: SupplierRateGuardChangeLog) {
  if (changeLog.status !== 'pending') return
  handlingRateGuardChangeLogID.value = changeLog.id
  try {
    await markRateGuardChangeLogHandled(changeLog.id)
    appStore.showSuccess('变更日志已标记为已处理')
    await loadRateGuardChangeLogs()
    if (!rateGuardChangeLogs.value.length && rateGuardChangeLogPage.value > 1) {
      rateGuardChangeLogPage.value -= 1
      await loadRateGuardChangeLogs()
    }
  } catch (err) {
    appStore.showError(errorMessage(err, '确认倍率守护变更日志失败'))
  } finally {
    handlingRateGuardChangeLogID.value = null
  }
}

async function runAutoMatch() {
  autoMatching.value = true
  try {
    const result = await autoMatchSupplierGroups(providerID.value || undefined)
    await loadGroups()
    appStore.showSuccess(`自动匹配完成：匹配 ${result.auto_matched} 个，冲突 ${result.ambiguous} 个`)
  } catch (err) {
    appStore.showError(errorMessage(err, '自动匹配失败'))
  } finally {
    autoMatching.value = false
  }
}

async function loadGroups() {
  loading.value = true
  error.value = ''
  try {
    const result = await listSupplierGroups({
      provider_id: providerID.value || undefined,
      search: search.value.trim() || undefined,
      platform: platformFilter.value || undefined,
      match_status: matchStatusFilter.value || undefined,
      rate_status: rateStatusFilter.value || undefined,
      sort_by: sortBy.value || undefined,
      sort_order: sortBy.value ? sortOrder.value : undefined,
      page: page.value,
      page_size: pageSize.value,
    })
    items.value = result.items
    total.value = result.total
    page.value = result.page
    pageSize.value = result.page_size
    groupSummary.value = result.summary
    void loadHealthTrend(result.items.map(group => group.id))
    if (selected.value) {
      selected.value = result.items.find(group => group.id === selected.value?.id) ?? selected.value
    }
  } catch (err) {
    error.value = errorMessage(err, '加载分组数据失败')
  } finally {
    loading.value = false
  }
}

function canDeleteSupplierGroup(group: SupplierProviderGroup): boolean {
  return !group.active
    && !group.local_group_id
    && !group.rate_guard_selected
    && group.account_count === 0
}

function supplierGroupDeleteHint(group: SupplierProviderGroup): string {
  if (canDeleteSupplierGroup(group)) return '删除失效分组记录'
  if (group.local_group_id) return '该分组已关联本地分组，不能删除'
  if (group.rate_guard_selected) return '该分组仍参与倍率守护，不能删除'
  if (group.account_count > 0) return '该分组仍有有效上游账号，不能删除'
  return '仅可删除失效分组记录'
}

function openDeleteGroupDialog(group: SupplierProviderGroup) {
  if (!canDeleteSupplierGroup(group)) return
  deleteTarget.value = group
}

async function confirmDeleteGroup() {
  const target = deleteTarget.value
  if (!target) return

  deleteTarget.value = null
  deletingGroupID.value = target.id
  try {
    await deleteSupplierGroup(target.id)
    if (selected.value?.id === target.id) {
      selected.value = null
    }
    await loadGroups()
    appStore.showSuccess('失效分组记录已删除')
  } catch (err) {
    appStore.showError(errorMessage(err, '删除分组记录失败'))
  } finally {
    deletingGroupID.value = null
  }
}

function openMappingDialog(group: SupplierProviderGroup) {
  mappingTarget.value = group
  mappingLocalGroupID.value = group.local_group_id ?? null
}

function closeMappingDialog() {
  if (savingMapping.value) return
  mappingTarget.value = null
  mappingLocalGroupID.value = null
}

async function saveMapping() {
  const target = mappingTarget.value
  const localGroupID = Number(mappingLocalGroupID.value)
  if (!target || !Number.isInteger(localGroupID) || localGroupID <= 0) {
    appStore.showError('请选择要匹配的本地分组')
    return
  }
  savingMapping.value = true
  try {
    await updateSupplierGroupMapping(target.id, localGroupID)
    mappingTarget.value = null
    mappingLocalGroupID.value = null
    await loadGroups()
    appStore.showSuccess('本地分组匹配已保存')
  } catch (err) {
    appStore.showError(errorMessage(err, '保存分组匹配失败'))
  } finally {
    savingMapping.value = false
  }
}

function openCreateDialog(group: SupplierProviderGroup) {
  createTarget.value = group
  newGroupName.value = group.name?.trim() || group.upstream_group_key
  newGroupPlatform.value = 'openai'
  newGroupRate.value = String(suggestedLocalRate(group))
}

function closeCreateDialog() {
  if (creatingLocalGroup.value) return
  createTarget.value = null
}

async function createLocalGroup() {
  const target = createTarget.value
  const name = newGroupName.value.trim()
  const rate = Number(newGroupRate.value)
  if (!target || !name) {
    appStore.showError('请输入本地分组名称')
    return
  }
  if (!Number.isFinite(rate) || rate <= 0) {
    appStore.showError('本地分组倍率必须大于 0')
    return
  }
  creatingLocalGroup.value = true
  try {
    const created = await adminAPI.groups.create({
      name,
      platform: newGroupPlatform.value as GroupPlatform,
      rate_multiplier: rate,
    })
    await loadLocalGroups()
    try {
      await updateSupplierGroupMapping(target.id, created.id)
    } catch (err) {
      appStore.showError(errorMessage(err, '本地分组已创建，但自动匹配失败，请使用更换本地分组'))
      createTarget.value = null
      await loadGroups()
      return
    }
    createTarget.value = null
    await loadGroups()
    appStore.showSuccess('本地分组已创建并完成匹配')
  } catch (err) {
    appStore.showError(errorMessage(err, '创建本地分组失败'))
  } finally {
    creatingLocalGroup.value = false
  }
}

function openRateDialog(group: SupplierProviderGroup) {
  rateTarget.value = group
  localRateInput.value = String(group.local_rate_multiplier ?? '')
}

function closeRateDialog() {
  if (savingLocalRate.value) return
  rateTarget.value = null
  localRateInput.value = ''
}

async function saveLocalRate() {
  const target = rateTarget.value
  const localGroupID = Number(target?.local_group_id)
  const rate = Number(localRateInput.value)
  if (!target || !Number.isInteger(localGroupID) || localGroupID <= 0) return
  if (!Number.isFinite(rate) || rate <= 0) {
    appStore.showError('本地分组倍率必须大于 0')
    return
  }
  savingLocalRate.value = true
  try {
    await adminAPI.groups.update(localGroupID, { rate_multiplier: rate })
    rateTarget.value = null
    localRateInput.value = ''
    await Promise.all([loadLocalGroups(), loadGroups()])
    appStore.showSuccess('本地分组倍率已更新')
  } catch (err) {
    appStore.showError(errorMessage(err, '修改本地分组倍率失败'))
  } finally {
    savingLocalRate.value = false
  }
}

async function removeMapping() {
  const target = unmatchTarget.value
  if (!target) return
  unmatchTarget.value = null
  try {
    await updateSupplierGroupMapping(target.id, null)
    await loadGroups()
    appStore.showSuccess('本地分组关联已取消')
  } catch (err) {
    appStore.showError(errorMessage(err, '取消分组关联失败'))
  }
}

async function toggleAutoMatchIgnored(group: SupplierProviderGroup) {
  policyUpdatingGroupID.value = group.id
  try {
    await updateSupplierGroupAutoMatchPolicy(group.id, !group.auto_match_ignored)
    await loadGroups()
    appStore.showSuccess(group.auto_match_ignored ? '已重新允许自动匹配' : '已忽略该分组的自动匹配')
  } catch (err) {
    appStore.showError(errorMessage(err, '更新自动匹配策略失败'))
  } finally {
    policyUpdatingGroupID.value = null
  }
}

async function toggleRateGuard(group: SupplierProviderGroup) {
	if (!group.rate_guard_selected && !rateGuardEligible(group)) return
	guardUpdatingGroupID.value = group.id
	const selected = !group.rate_guard_selected
	try {
		await updateSupplierGroupRateGuard(group.id, selected)
		await loadGroups()
		appStore.showSuccess(selected ? '已设为人工守护分组' : '已取消人工守护')
	} catch (err) {
		appStore.showError(errorMessage(err, '更新倍率守护失败'))
	} finally {
		guardUpdatingGroupID.value = null
	}
}

async function toggleRateGuardIgnored(group: SupplierProviderGroup) {
	guardIgnoreUpdatingGroupID.value = group.id
	const ignored = !group.rate_guard_ignored
	try {
		await updateSupplierGroupRateGuardIgnore(group.id, ignored)
		await loadGroups()
		appStore.showSuccess(ignored ? '已忽略该分组的自动倍率守护' : '已恢复该分组的自动倍率守护')
	} catch (err) {
		appStore.showError(errorMessage(err, '更新自动倍率守护忽略状态失败'))
	} finally {
		guardIgnoreUpdatingGroupID.value = null
	}
}

async function confirmRateGuardIgnore() {
	const target = rateGuardIgnoreTarget.value
	if (!target) return
	rateGuardIgnoreTarget.value = null
	await toggleRateGuardIgnored(target)
}

function rateGuardEligible(group: SupplierProviderGroup): boolean {
	return Boolean(group.local_group_id && group.active && group.local_group_status === 'active')
}

function canManageManualRateGuard(group: SupplierProviderGroup): boolean {
	if (group.rate_guard_selected && group.rate_guard_selection_mode === 'manual') {
		return !group.active || group.local_group_active_mapping_count > 1
	}
	return !group.rate_guard_selected && group.local_group_active_mapping_count > 1
}

function hasOtherRateGuard(group: SupplierProviderGroup): boolean {
	return Boolean(group.local_group_rate_guard_group_id && group.local_group_rate_guard_group_id !== group.id)
}

function rateGuardStatus(group: SupplierProviderGroup): { label: string; tone: string; detail?: string; title?: string } {
	if (group.rate_guard_selected && group.rate_guard_ignored) {
		const detail = '保留守护来源，已暂停自动倍率调整'
		return { label: '已忽略自动守护', tone: 'muted', detail, title: detail }
	}
	if (group.rate_guard_selected && !group.active) {
		const detail = '最近同步未返回该分组，或上游已停用'
		return { label: '上游分组不可用', tone: 'danger', detail, title: detail }
	}
	if (group.rate_guard_selected && group.local_group_status !== 'active') {
		const detail = '匹配的本地分组当前未启用'
		return { label: '本地分组不可用', tone: 'danger', detail, title: detail }
	}
	if (group.rate_guard_selected && group.group_sync_status === 'never') {
		const detail = '供应商尚未完成首次分组同步'
		return { label: '等待首次同步', tone: 'pending', detail, title: detail }
	}
	if (group.rate_guard_selected && group.group_sync_status === 'running') {
		const detail = '供应商分组数据正在同步'
		return { label: '分组同步中', tone: 'pending', detail, title: detail }
	}
	if (group.rate_guard_selected && group.group_sync_status === 'failed') {
		const detail = '供应商最近一次分组同步失败'
		return { label: '分组同步失败', tone: 'danger', detail, title: detail }
	}
	if (group.rate_guard_selected && group.group_sync_status !== 'success') {
		const detail = `未知的分组同步状态：${group.group_sync_status || '空'}`
		return { label: '同步状态异常', tone: 'danger', detail, title: detail }
	}
	if (group.rate_guard_selected && group.rate_guard_selection_mode === 'auto') {
		return { label: '自动守护', tone: 'auto' }
	}
	if (group.rate_guard_selected && group.rate_guard_selection_mode === 'manual') {
		return { label: '人工守护', tone: 'manual' }
	}
	if (!group.local_group_id) {
		return { label: '未匹配', tone: 'muted' }
	}
	if (hasOtherRateGuard(group)) {
		const detail = [
			group.local_group_rate_guard_provider_name,
			group.local_group_rate_guard_group_name,
		].filter(Boolean).join(' / ')
		return {
			label: '已由其它分组守护',
			tone: 'muted',
			detail: detail || '其它上游分组',
			title: '当前本地分组由该上游分组执行倍率守护，本分组不会参与守护',
		}
	}
	if (group.local_group_active_mapping_count > 1) {
		return { label: '可设守护', tone: 'pending' }
	}
	return { label: '非守护源', tone: 'muted' }
}

async function resolveNameChange(group: SupplierProviderGroup, action: 'keep_local' | 'sync_local_name') {
  resolvingNameGroupID.value = group.id
  try {
    await resolveSupplierGroupNameChange(group.id, action)
    if (action === 'sync_local_name') {
      await loadLocalGroups()
    }
    await loadGroups()
    appStore.showSuccess(action === 'sync_local_name' ? '本地分组名称已同步' : '已保留本地分组名称')
  } catch (err) {
    appStore.showError(errorMessage(err, '处理名称变化失败'))
  } finally {
    resolvingNameGroupID.value = null
  }
}

function rateInsight(group: SupplierProviderGroup): SupplierGroupRateInsight {
  return getSupplierGroupRateInsight({
    localGroupID: group.local_group_id,
    upstreamRate: Number(group.rate_multiplier),
    localRate: group.local_rate_multiplier,
    localStatus: group.local_group_status || '',
  })
}

function supplierTone(providerID: number) {
  const providerIndex = supplierIDs.value.indexOf(providerID)
  const toneIndex = providerIndex >= 0 ? providerIndex : Math.abs(providerID)
  return SUPPLIER_TONES[toneIndex % SUPPLIER_TONES.length]
}

function upstreamGroupTone(groupKey: string) {
  let hash = 0
  for (const char of groupKey) {
    hash = ((hash << 5) - hash + char.charCodeAt(0)) | 0
  }
  return UPSTREAM_GROUP_TONES[(hash >>> 0) % UPSTREAM_GROUP_TONES.length]
}

function supplierTypeLabel(providerID: number): string {
  const providerType = providers.value.find(provider => provider.id === providerID)?.provider_type?.trim()
  if (!providerType) return '未知类型'
  return SUPPLIER_TYPE_LABELS[providerType.toLowerCase()] || providerType
}

function supplierTypeTone(providerID: number): string {
  const providerType = providers.value.find(provider => provider.id === providerID)?.provider_type?.trim().toLowerCase() || ''
  return SUPPLIER_TYPE_TONES[providerType] || SUPPLIER_TYPE_TONE_DEFAULT
}

function upstreamPlatformLabel(group: SupplierProviderGroup): string {
  return group.local_group_platform ? platformLabel(group.local_group_platform) : '待匹配'
}

function groupPlatform(platform?: string): GroupPlatform | undefined {
  if (!platform || !(platform in PLATFORM_LABELS)) return undefined
  return platform as GroupPlatform
}

function upstreamStatusLabel(group: SupplierProviderGroup): string {
  return group.raw_status?.trim() || (group.active ? '有效' : '失效')
}

function upstreamStatusTone(group: SupplierProviderGroup): string {
  const status = upstreamStatusLabel(group).toLowerCase()
  if (group.active && ['active', 'enabled', 'normal', 'success', '有效'].includes(status)) return 'good'
  if (!group.active || ['inactive', 'disabled', 'failed', '失效'].includes(status)) return 'bad'
  return 'info'
}

function matchStatusLabel(group: SupplierProviderGroup): string {
	if (group.local_group_id && group.auto_match_status === 'manual') return '人工匹配'
	if (group.local_group_id && group.auto_match_status === 'auto_matched') return '自动匹配'
	if (group.auto_match_ignored) return '已忽略'
	if (group.auto_match_status === 'ambiguous') return '名称冲突'
	return '待匹配'
}

function matchStatusTone(group: SupplierProviderGroup): string {
	if (group.name_change_pending || group.auto_match_status === 'ambiguous') return 'warn'
	if (group.local_group_id && group.auto_match_status === 'auto_matched') return 'auto'
	if (group.local_group_id && group.auto_match_status === 'manual') return 'manual'
	if (group.auto_match_ignored) return 'muted'
	return 'pending'
}

function suggestedLocalRate(group: SupplierProviderGroup): number {
  const upstreamRate = Number(group.rate_multiplier)
  if (!Number.isFinite(upstreamRate) || upstreamRate <= 0) return 1
  return Math.ceil(upstreamRate * 1.1 * 100) / 100
}

function formatRate(value?: number): string {
  const rate = Number(value)
  if (!Number.isFinite(rate)) return '-'
  return `${rate.toFixed(4).replace(/\.?0+$/, '')}x`
}

function platformLabel(platform?: string): string {
  if (!platform) return '未设置平台'
  return PLATFORM_LABELS[platform] || platform
}

function formatTime(value?: string): string {
  if (!value) return '-'
  const date = new Date(value)
  if (Number.isNaN(date.getTime())) return '-'
  return date.toLocaleString('zh-CN')
}

function percentage(value: number, totalValue: number): number {
  if (totalValue <= 0) return 0
  return Math.round((value / totalValue) * 100)
}

function errorMessage(err: unknown, fallback: string): string {
  if (typeof err === 'object' && err && 'message' in err) {
    return String((err as { message?: unknown }).message || fallback)
  }
  return fallback
}
</script>

<style scoped>
.sp-console-shell {
  display: grid;
  gap: 1rem;
}

.sp-filter-toolbar {
  width: 100%;
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

.sp-filter-card-body {
  display: flex;
  align-items: flex-end;
  gap: 0.875rem;
  padding: 0.875rem 1rem 1rem;
}

.sp-filter-fields {
  display: grid;
  min-width: 0;
  flex: 1 1 auto;
  grid-template-columns: minmax(15rem, 1fr) repeat(4, minmax(9rem, 0.55fr));
  gap: 0.625rem;
}

.sp-filter-control {
  display: flex;
  min-width: 0;
  flex-direction: column;
  gap: 0.3rem;
}


.sp-filter-actions {
  display: flex;
  flex: 0 0 auto;
  align-items: center;
  align-self: stretch;
  justify-content: flex-end;
  gap: 0.5rem;
  padding-left: 0.875rem;
  border-left: 1px solid var(--sp-line);
}

.sp-control-button,
.sp-dialog-actions .sp-button {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  gap: 0.4rem;
}

.sp-control-button-reset {
  border-color: color-mix(in srgb, var(--sp-muted) 30%, var(--sp-line));
  background: var(--sp-panel);
  color: var(--sp-muted);
}

.sp-control-button-match {
  border-color: color-mix(in srgb, var(--sp-cyan) 38%, var(--sp-line));
  background: color-mix(in srgb, var(--sp-cyan) 9%, var(--sp-panel));
  color: var(--sp-cyan);
}

.sp-control-button-log {
  position: relative;
  border-color: color-mix(in srgb, var(--sp-amber) 42%, var(--sp-line));
  background: color-mix(in srgb, var(--sp-amber) 10%, var(--sp-panel));
  color: var(--sp-amber);
}

.sp-control-button-refresh {
  border-color: color-mix(in srgb, var(--sp-green) 38%, var(--sp-line));
  background: color-mix(in srgb, var(--sp-green) 9%, var(--sp-panel));
  color: var(--sp-green);
}

.sp-pending-count {
  display: inline-grid;
  min-width: 18px;
  height: 18px;
  flex: 0 0 auto;
  place-items: center;
  border-radius: 999px;
  padding: 0 5px;
  background: var(--sp-red);
  color: white;
  font-size: 10px;
  font-style: normal;
  font-weight: 800;
  line-height: 1;
}

.sp-change-log-dialog {
  --sp-change-log-accent: #7c3aed;
  --sp-change-log-local: #0891b2;
  --sp-change-log-upstream: #8b5cf6;
  --sp-change-log-old-rate: #d97706;
  --sp-change-log-new-rate: #16a34a;
  --sp-change-log-time: #2563eb;
  --sp-change-log-muted: #64748b;
  --sp-change-log-line: #e5e7eb;
  --sp-change-log-panel: #ffffff;
  --sp-change-log-soft: #f1f5f9;
  display: grid;
  gap: 14px;
}

.sp-change-log-dialog.log {
  padding: 0.2rem;
  border: 1px solid color-mix(in srgb, var(--sp-change-log-accent) 22%, var(--sp-change-log-line));
  border-radius: 0.75rem;
  background: color-mix(in srgb, var(--sp-change-log-accent) 3%, var(--sp-change-log-panel));
}

:global(.dark) .sp-change-log-dialog {
  --sp-change-log-local: #22d3ee;
  --sp-change-log-upstream: #a78bfa;
  --sp-change-log-old-rate: #fbbf24;
  --sp-change-log-new-rate: #4ade80;
  --sp-change-log-time: #60a5fa;
  --sp-change-log-muted: #94a3b8;
  --sp-change-log-line: #374151;
  --sp-change-log-panel: #1f2937;
  --sp-change-log-soft: #374151;
}

.sp-change-log-table-region {
  overflow: hidden;
  border: 1px solid color-mix(in srgb, var(--sp-change-log-accent) 18%, var(--sp-change-log-soft));
  border-radius: 10px;
}

.sp-change-log-local-group,
.sp-change-log-upstream-group {
  font-weight: 750;
}

.sp-change-log-local-group {
  color: var(--sp-change-log-local);
}

.sp-change-log-upstream-group {
  color: var(--sp-change-log-upstream);
}

.sp-change-log-key {
  margin-left: 6px;
  color: var(--sp-change-log-muted);
  font-family: ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, monospace;
  font-size: 11px;
  font-weight: 650;
  white-space: nowrap;
}

.sp-change-log-rate {
  display: inline-flex;
  min-width: 54px;
  align-items: center;
  justify-content: center;
  border: 1px solid transparent;
  border-radius: 999px;
  padding: 3px 8px;
  font-size: 12px;
  font-weight: 800;
  font-variant-numeric: tabular-nums;
}

.sp-change-log-old-rate {
  border-color: color-mix(in srgb, var(--sp-change-log-old-rate) 38%, var(--sp-change-log-line));
  background: color-mix(in srgb, var(--sp-change-log-old-rate) 14%, var(--sp-change-log-panel));
  color: var(--sp-change-log-old-rate);
}

.sp-change-log-new-rate {
  border-color: color-mix(in srgb, var(--sp-change-log-new-rate) 38%, var(--sp-change-log-line));
  background: color-mix(in srgb, var(--sp-change-log-new-rate) 14%, var(--sp-change-log-panel));
  color: var(--sp-change-log-new-rate);
}

.sp-change-log-time {
  color: var(--sp-change-log-time);
  font-size: 12px;
  font-weight: 650;
  font-variant-numeric: tabular-nums;
  white-space: nowrap;
}

.sp-change-log-status-action {
  color: var(--sp-change-log-accent);
  font-weight: 750;
}

.sp-change-log-status-action:disabled {
  cursor: wait;
  opacity: 0.65;
}

.sp-change-log-pagination {
  border-top: 1px solid var(--sp-change-log-soft);
}

.sp-summary-grid {
  display: grid;
  grid-template-columns: repeat(4, minmax(0, 1fr));
  gap: 0.75rem;
}

.sp-summary-filter {
  min-width: 0;
  padding: 0;
  border: 0;
  outline: 0;
  background: transparent;
  text-align: left;
  cursor: pointer;
}

.sp-summary-filter:disabled {
  cursor: wait;
  opacity: 0.72;
}

.sp-summary-filter:focus-visible :deep(.stat-card) {
  border-color: var(--sp-cyan);
  box-shadow: 0 0 0 3px color-mix(in srgb, var(--sp-cyan) 16%, transparent);
}

.sp-summary-filter.active :deep(.stat-card) {
  border-color: color-mix(in srgb, var(--sp-cyan) 58%, var(--sp-line));
  background: color-mix(in srgb, var(--sp-cyan) 6%, var(--sp-panel));
  box-shadow: 0 6px 18px rgba(15, 23, 42, 0.1), inset 0 0 0 1px color-mix(in srgb, var(--sp-cyan) 10%, transparent);
}

.sp-summary-grid :deep(.stat-card) {
  position: relative;
  min-height: 6.25rem;
  align-items: center;
  overflow: hidden;
  padding: 1rem;
  border: 1px solid color-mix(in srgb, var(--sp-line) 82%, var(--sp-text));
  border-radius: 0.5rem;
  background: var(--sp-panel);
  box-shadow: 0 4px 14px rgba(15, 23, 42, 0.06);
  transition: border-color 160ms ease, box-shadow 160ms ease, transform 160ms ease;
  animation: sp-summary-enter 180ms ease-out both;
}

.sp-summary-grid :deep(.stat-card)::before {
  position: absolute;
  inset: 0 auto 0 0;
  width: 3px;
  background: var(--sp-cyan);
  content: '';
}

.sp-summary-filter:nth-child(2) :deep(.stat-card)::before {
  background: var(--sp-green);
}

.sp-summary-filter:nth-child(3) :deep(.stat-card)::before {
  background: var(--sp-amber);
}

.sp-summary-filter:nth-child(4) :deep(.stat-card)::before {
  background: var(--sp-red);
}

.sp-summary-grid :deep(.stat-icon) {
  width: 2.75rem;
  height: 2.75rem;
  border: 1px solid currentColor;
  border-radius: 0.5rem;
  opacity: 0.92;
}

.sp-summary-grid :deep(.stat-label) {
  color: var(--sp-muted);
  font-size: 0.78rem;
  font-weight: 600;
  letter-spacing: 0;
}

.sp-summary-grid :deep(.stat-value) {
  color: var(--sp-text);
  font-size: 1.75rem;
  font-weight: 750;
  line-height: 1.1;
}

.sp-summary-grid :deep(.stat-trend) {
  padding: 0.15rem 0.4rem;
  border: 1px solid var(--sp-line);
  border-radius: 0.25rem;
  background: var(--sp-panel-2);
  color: var(--sp-muted);
  font-size: 0.75rem;
  font-weight: 600;
}

.sp-summary-filter:nth-child(2) :deep(.stat-card) { animation-delay: 30ms; }
.sp-summary-filter:nth-child(3) :deep(.stat-card) { animation-delay: 60ms; }
.sp-summary-filter:nth-child(4) :deep(.stat-card) { animation-delay: 90ms; }

.sp-groups-panel {
  min-width: 0;
  margin-bottom: 0;
}

.sp-provider-shortcuts {
  display: flex;
  min-width: 10rem;
  max-width: min(58vw, 52rem);
  flex: 1 1 auto;
  align-items: center;
  justify-content: center;
  gap: 0.4rem;
  overflow-x: auto;
  padding: 0.15rem 0.25rem;
  scrollbar-width: thin;
}

.sp-provider-shortcut {
  display: inline-flex;
  max-width: 8.5rem;
  min-height: 2rem;
  flex: 0 0 auto;
  align-items: center;
  justify-content: center;
  padding: 0.3rem 0.65rem;
  border: 1px solid var(--sp-line);
  border-radius: 0.375rem;
  background: var(--sp-panel-2);
  color: var(--sp-muted);
  font-size: 0.75rem;
  font-weight: 650;
  cursor: pointer;
  transition: border-color 140ms ease, background 140ms ease, color 140ms ease;
}

.sp-provider-shortcut span {
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.sp-provider-shortcut:hover {
  border-color: color-mix(in srgb, var(--sp-cyan) 36%, var(--sp-line));
  color: var(--sp-cyan);
}

.sp-provider-shortcut.active {
  border-color: color-mix(in srgb, var(--sp-cyan) 58%, var(--sp-line));
  background: color-mix(in srgb, var(--sp-cyan) 8%, var(--sp-panel));
  color: var(--sp-cyan);
}

.sp-provider-shortcut:disabled {
  cursor: wait;
  opacity: 0.68;
}

.sp-panel-signals {
  display: flex;
  align-items: center;
  gap: 1rem;
  color: var(--sp-muted);
  font-size: 0.75rem;
}

.sp-panel-signals span {
  display: inline-flex;
  align-items: center;
  gap: 0.375rem;
}

.sp-panel-signals i,
.sp-status i,
.sp-rate-status i,
.sp-match-state i {
  width: 0.42rem;
  height: 0.42rem;
  flex: 0 0 auto;
  border-radius: 50%;
  background: currentColor;
}

.sp-panel-signals .good { color: var(--sp-green); }
.sp-panel-signals .warn { color: var(--sp-amber); }

.sp-attention-shortcuts {
  display: flex;
  min-width: 0;
  align-items: center;
  gap: 0.4rem;
  overflow-x: auto;
  padding: 0.55rem 0.75rem;
  border-top: 1px solid var(--sp-soft);
  border-bottom: 1px solid var(--sp-soft);
  background: color-mix(in srgb, var(--sp-panel-2) 72%, var(--sp-panel));
  scrollbar-width: thin;
}

.sp-attention-shortcut {
  display: inline-flex;
  min-width: 4.75rem;
  min-height: 2rem;
  flex: 0 0 auto;
  align-items: center;
  justify-content: center;
  padding: 0.3rem 0.7rem;
  border: 1px solid var(--sp-line);
  border-radius: 0.375rem;
  background: var(--sp-panel);
  color: var(--sp-muted);
  font-size: 0.75rem;
  font-weight: 650;
  cursor: pointer;
  transition: border-color 140ms ease, background 140ms ease, color 140ms ease;
  white-space: nowrap;
}

.sp-attention-shortcut:hover {
  border-color: color-mix(in srgb, var(--sp-amber) 42%, var(--sp-line));
  color: var(--sp-amber);
}

.sp-attention-shortcut:focus-visible {
  outline: 2px solid color-mix(in srgb, var(--sp-cyan) 60%, transparent);
  outline-offset: 2px;
}

.sp-attention-shortcut.active {
  border-color: color-mix(in srgb, var(--sp-amber) 58%, var(--sp-line));
  background: color-mix(in srgb, var(--sp-amber) 9%, var(--sp-panel));
  color: var(--sp-amber);
}

.sp-attention-shortcut:disabled {
  cursor: wait;
  opacity: 0.68;
}

.sp-table-shell {
  display: flex;
  height: min(64vh, 680px);
  min-height: 24rem;
  flex-direction: column;
  overflow: hidden;
}

.sp-table-shell :deep(.table-wrapper) {
  min-height: 0;
  flex: 1;
}

.sp-table-shell :deep(td) {
  vertical-align: middle;
}

.sp-data-pagination {
  flex-shrink: 0;
  border-top: 1px solid var(--sp-soft);
}

.sp-group-name {
  max-width: 14rem;
  overflow: hidden;
  font-weight: 650;
  text-overflow: ellipsis;
}

.sp-upstream-group-chip {
  display: inline-flex;
  max-width: 14rem;
  min-height: 1.75rem;
  align-items: center;
  gap: 0.5rem;
  padding: 0.25rem 0.55rem 0.25rem 0.35rem;
  overflow: hidden;
  border-radius: 0.25rem;
  line-height: 1.2;
  white-space: nowrap;
}

.sp-upstream-group-accent {
  width: 0.2rem;
  height: 1rem;
  flex: 0 0 auto;
  border-radius: 0.125rem;
}

.sp-supplier-chip {
  display: inline-flex;
  max-width: 10rem;
  align-items: center;
  gap: 0.45rem;
  padding: 0.3rem 0.55rem;
  overflow: hidden;
  border-width: 1px;
  border-style: solid;
  border-radius: 0.375rem;
  font-size: 0.8rem;
  font-weight: 700;
  line-height: 1.2;
}

.sp-supplier-name {
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
}

.sp-provider-meta {
  display: flex;
  align-items: center;
  gap: 0.35rem;
  color: var(--sp-dim);
}

.sp-provider-type {
  display: inline-flex;
  align-items: center;
  padding: 0.1rem 0.35rem;
  border-radius: 0.25rem;
  font-size: 0.68rem;
  font-weight: 700;
  line-height: 1.2;
}

.sp-inline-platform {
  flex: 0 0 auto;
  font-size: 0.7rem;
  font-weight: 650;
  opacity: 0.9;
  white-space: nowrap;
}

.sp-inline-platform-muted {
  color: var(--sp-muted);
}

.sp-supplier-dot {
  width: 0.48rem;
  height: 0.48rem;
  flex: 0 0 auto;
  border-radius: 50%;
  box-shadow: 0 0 0 2px color-mix(in srgb, currentColor 14%, transparent);
}

.sp-key {
  max-width: 14rem;
  overflow: hidden;
  font-family: ui-monospace, SFMono-Regular, Consolas, monospace;
  text-overflow: ellipsis;
}

.sp-rate-value {
  display: inline-flex;
  min-width: 3.5rem;
  align-items: center;
  font-variant-numeric: tabular-nums;
  font-weight: 750;
}

.sp-status,
.sp-rate-status {
  display: inline-flex;
  align-items: center;
  gap: 0.4rem;
  padding: 0.25rem 0.55rem;
  border: 1px solid currentColor;
  border-radius: 9999px;
  background: var(--sp-panel);
  font-size: 0.75rem;
  font-weight: 650;
  white-space: nowrap;
}

.sp-status.good,
.sp-rate-status.normal { color: var(--sp-green); }
.sp-status.bad,
.sp-rate-status.inverted { color: var(--sp-red); }
.sp-status.info { color: var(--sp-blue); }
.sp-rate-status.low,
.sp-rate-status.equal { color: var(--sp-amber); }
.sp-rate-status.unmatched,
.sp-rate-status.inactive,
.sp-rate-status.invalid { color: var(--sp-muted); }

.sp-match-state-stack {
  display: grid;
  justify-items: start;
  gap: 0.3rem;
}

.sp-match-state {
  display: inline-flex;
  align-items: center;
  gap: 0.35rem;
  color: var(--sp-muted);
  font-size: 0.75rem;
  font-weight: 650;
  white-space: nowrap;
}

.sp-match-state.auto { color: var(--sp-green); }
.sp-match-state.manual { color: var(--sp-blue); }
.sp-match-state.warn { color: var(--sp-amber); }
.sp-match-state.pending,
.sp-match-state.muted { color: var(--sp-muted); }

.sp-guard-state {
	display: inline-flex;
	width: fit-content;
	align-items: center;
	gap: 0.35rem;
	padding: 0.25rem 0.45rem;
	border: 1px solid currentColor;
	border-radius: 0.375rem;
	font-size: 0.72rem;
	font-weight: 650;
	white-space: nowrap;
}

.sp-guard-state-stack,
.sp-guard-detail {
	display: grid;
	justify-items: start;
	gap: 0.3rem;
}

.sp-guard-state-stack small,
.sp-guard-detail small {
	max-width: 11rem;
	overflow: hidden;
	color: var(--sp-muted);
	font-size: 0.68rem;
	line-height: 1.3;
	text-overflow: ellipsis;
	white-space: nowrap;
}

.sp-guard-state.auto { color: var(--sp-green); }
.sp-guard-state.manual { color: var(--sp-blue); }
.sp-guard-state.pending { color: var(--sp-amber); }
.sp-guard-state.danger { color: var(--sp-red); }
.sp-guard-state.muted { color: var(--sp-muted); }

.sp-name-change-link {
  padding: 0;
  border: 0;
  background: transparent;
  color: var(--sp-amber);
  font-size: 0.7rem;
  cursor: pointer;
}

.sp-name-change-link:hover { text-decoration: underline; }

.sp-local-group .sp-sub {
  display: flex;
  align-items: center;
  gap: 0.5rem;
}

.sp-local-group-badge {
  max-width: 12rem;
}

.sp-detail-group-badge {
  width: fit-content;
  max-width: 100%;
  margin-top: 0.375rem;
}

.sp-inline-empty {
  display: inline-flex;
  align-items: center;
  gap: 0.35rem;
  padding: 0;
  border: 0;
  background: transparent;
  color: var(--sp-muted);
  cursor: pointer;
}

.sp-inline-empty:hover { color: var(--sp-cyan); }
.sp-empty-value { color: var(--sp-dim); }

.sp-rate-delta-cell {
  display: grid;
  gap: 0.15rem;
  font-variant-numeric: tabular-nums;
}

.sp-rate-delta-cell strong {
  color: var(--sp-text);
  font-size: 0.9rem;
}

.sp-rate-delta-cell.normal strong { color: var(--sp-green); }
.sp-rate-delta-cell.inverted strong { color: var(--sp-red); }
.sp-rate-delta-cell.low strong,
.sp-rate-delta-cell.equal strong { color: var(--sp-amber); }

.sp-account-count {
  display: inline-flex;
  min-width: 3.2rem;
  align-items: center;
  justify-content: center;
  gap: 0.35rem;
  padding: 0.3rem 0.5rem;
  border: 1px solid var(--sp-line);
  border-radius: 0.375rem;
  background: var(--sp-panel-2);
  color: var(--sp-blue);
  cursor: pointer;
}

.sp-account-count:hover {
  border-color: var(--sp-cyan);
  color: var(--sp-cyan);
}

.sp-row-actions {
  display: flex;
  align-items: center;
  gap: 0.4rem;
}

.sp-row-action {
  display: inline-flex;
  min-height: 2rem;
  align-items: center;
  justify-content: center;
  gap: 0.3rem;
  padding: 0.35rem 0.55rem;
  border: 1px solid var(--sp-line);
  border-radius: 0.375rem;
  background: var(--sp-panel);
  color: var(--sp-muted);
  font-size: 0.75rem;
  font-weight: 650;
  cursor: pointer;
  transition: border-color 140ms ease, background 140ms ease, color 140ms ease;
}

.sp-row-action:hover {
  border-color: var(--sp-cyan);
  color: var(--sp-cyan);
}

.sp-row-action.primary {
  border-color: color-mix(in srgb, var(--sp-cyan) 40%, var(--sp-line));
  background: color-mix(in srgb, var(--sp-cyan) 7%, var(--sp-panel));
  color: var(--sp-cyan);
}

.sp-row-action.active {
  border-color: color-mix(in srgb, var(--sp-amber) 42%, var(--sp-line));
  color: var(--sp-amber);
}

.sp-row-action.guard {
	border-color: color-mix(in srgb, var(--sp-green) 34%, var(--sp-line));
	color: var(--sp-green);
}

.sp-row-action.guard.active {
	border-color: color-mix(in srgb, var(--sp-amber) 42%, var(--sp-line));
	color: var(--sp-amber);
}

.sp-row-action:disabled {
  cursor: not-allowed;
  opacity: 0.55;
}

.sp-row-action.danger:hover {
  border-color: var(--sp-red);
  color: var(--sp-red);
}

.sp-drawer-summary {
  display: grid;
  grid-template-columns: 1fr auto;
  align-items: end;
  gap: 0.3rem 1rem;
  padding: 0 0 1rem;
  border-bottom: 1px solid var(--sp-soft);
}

.sp-drawer-summary .sp-rate-status { width: fit-content; }
.sp-drawer-summary strong { font-size: 2rem; line-height: 1; }
.sp-drawer-summary small { grid-column: 2; color: var(--sp-muted); text-align: right; }
.sp-detail-wide { grid-column: 1 / -1; }

.sp-name-change-alert {
  display: grid;
  gap: 0.75rem;
  margin: 1rem 0;
  padding: 0.85rem;
  border: 1px solid color-mix(in srgb, var(--sp-amber) 45%, var(--sp-line));
  border-radius: 0.5rem;
  background: color-mix(in srgb, var(--sp-amber) 7%, var(--sp-panel));
}

.sp-name-change-alert > div:first-child {
  display: grid;
  gap: 0.25rem;
}

.sp-name-change-alert span,
.sp-name-change-alert small { color: var(--sp-muted); font-size: 0.75rem; }
.sp-name-change-alert strong { color: var(--sp-text); font-size: 0.9rem; overflow-wrap: anywhere; }

.sp-name-change-actions {
  display: flex;
  flex-wrap: wrap;
  gap: 0.5rem;
}

:global(.modal-content:has(.sp-dialog-context)),
:global(.modal-content:has(.sp-change-log-dialog)) {
  --sp-panel: #ffffff;
  --sp-panel-2: #f9fafb;
  --sp-panel-3: #f3f4f6;
  --sp-line: #e5e7eb;
  --sp-soft: #f1f5f9;
  --sp-text: #111827;
  --sp-muted: #64748b;
  --sp-dim: #94a3b8;
  --sp-cyan: #3b82f6;
  --sp-green: #16a34a;
  --sp-amber: #d97706;
  --sp-orange: #ea580c;
  --sp-red: #dc2626;
  --sp-blue: #2563eb;
  --sp-violet: #7c3aed;
  color: var(--sp-text);
  background: var(--sp-panel);
}

:global(.dark .modal-content:has(.sp-dialog-context)),
:global(.dark .modal-content:has(.sp-change-log-dialog)) {
  --sp-panel: #1f2937;
  --sp-panel-2: #111827;
  --sp-panel-3: #374151;
  --sp-line: #374151;
  --sp-soft: #374151;
  --sp-text: #f9fafb;
  --sp-muted: #9ca3af;
  --sp-dim: #6b7280;
  background: var(--sp-panel);
}

:global(.modal-content:has(.sp-dialog-context.match)) {
  --sp-dialog-shell-accent: var(--sp-cyan);
}

:global(.modal-content:has(.sp-dialog-context.create)) {
  --sp-dialog-shell-accent: var(--sp-green);
}

:global(.modal-content:has(.sp-dialog-context.rate)) {
  --sp-dialog-shell-accent: var(--sp-amber);
}

:global(.modal-content:has(.sp-change-log-dialog)) {
  --sp-dialog-shell-accent: var(--sp-violet);
}

:global(.modal-content:has(.sp-dialog-context)),
:global(.modal-content:has(.sp-change-log-dialog)) {
  border-top: 3px solid var(--sp-dialog-shell-accent);
  box-shadow: 0 16px 36px color-mix(in srgb, var(--sp-dialog-shell-accent) 18%, rgba(15, 23, 42, 0.18));
}

:global(.modal-content:has(.sp-dialog-context) .modal-header),
:global(.modal-content:has(.sp-change-log-dialog) .modal-header) {
  border-bottom-color: color-mix(in srgb, var(--sp-dialog-shell-accent) 24%, var(--sp-line));
  background: linear-gradient(90deg, color-mix(in srgb, var(--sp-dialog-shell-accent) 9%, transparent), transparent 72%);
}

:global(.modal-content:has(.sp-dialog-context) .modal-title),
:global(.modal-content:has(.sp-change-log-dialog) .modal-title) {
  color: var(--sp-dialog-shell-accent);
}

.sp-dialog-context {
  --sp-dialog-accent: var(--sp-cyan);
  display: grid;
  gap: 0.25rem;
  margin-bottom: 1rem;
  padding: 0.85rem 0.95rem;
  border: 1px solid color-mix(in srgb, var(--sp-dialog-accent) 30%, var(--sp-line));
  border-left-width: 4px;
  border-radius: 0.625rem;
  background: color-mix(in srgb, var(--sp-dialog-accent) 7%, var(--sp-panel));
}

.sp-dialog-context.match { --sp-dialog-accent: var(--sp-cyan); }
.sp-dialog-context.create { --sp-dialog-accent: var(--sp-green); }
.sp-dialog-context.rate { --sp-dialog-accent: var(--sp-amber); }

.sp-dialog-context > span { color: var(--sp-dialog-accent); font-weight: 700; }

.sp-dialog-context span,
.sp-dialog-context small {
  font-size: 0.78rem;
}

.sp-dialog-context small { color: var(--sp-muted); }
.sp-dialog-context strong { color: var(--sp-text); font-size: 1rem; }

:global(.dark) .sp-dialog-context {
  background: color-mix(in srgb, var(--sp-dialog-accent) 12%, var(--sp-panel));
}

.sp-field {
  display: grid;
  gap: 0.375rem;
  color: var(--sp-text);
  font-size: 0.875rem;
  font-weight: 500;
}

.sp-dialog-form {
  display: grid;
  gap: 1rem;
}

.sp-match-preview {
  display: grid;
  grid-template-columns: repeat(3, minmax(0, 1fr));
  gap: 0.75rem;
  margin-top: 1rem;
}

.sp-match-preview-card {
  --sp-preview-accent: var(--sp-blue);
  display: grid;
  gap: 0.25rem;
  min-width: 0;
  padding: 0.75rem;
  border: 1px solid color-mix(in srgb, var(--sp-preview-accent) 24%, var(--sp-line));
  border-radius: 0.5rem;
  background: color-mix(in srgb, var(--sp-preview-accent) 6%, var(--sp-panel));
}

.sp-match-preview-card.platform { --sp-preview-accent: var(--sp-blue); }
.sp-match-preview-card.rate { --sp-preview-accent: var(--sp-amber); }
.sp-match-preview-card.delta { --sp-preview-accent: var(--sp-green); }

.sp-match-preview-card span { color: var(--sp-muted); font-size: 0.75rem; }
.sp-match-preview-card strong { color: var(--sp-preview-accent); font-size: 0.9rem; }

:global(.dark) .sp-match-preview-card {
  background: color-mix(in srgb, var(--sp-preview-accent) 11%, var(--sp-panel));
}

.sp-rate-recommendation {
  --sp-recommendation-accent: var(--sp-green);
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 0.75rem;
  margin-top: 1rem;
  padding: 0.75rem 0.85rem;
  border: 1px solid color-mix(in srgb, var(--sp-recommendation-accent) 26%, var(--sp-line));
  border-radius: 0.5rem;
  background: color-mix(in srgb, var(--sp-recommendation-accent) 6%, var(--sp-panel));
  color: var(--sp-muted);
  font-size: 0.8rem;
}

.sp-rate-recommendation.create,
.sp-rate-recommendation.success { --sp-recommendation-accent: var(--sp-green); }
.sp-rate-recommendation.rate,
.sp-rate-recommendation.warning { --sp-recommendation-accent: var(--sp-amber); }
.sp-rate-recommendation.danger { --sp-recommendation-accent: var(--sp-red); }
.sp-rate-recommendation.neutral { --sp-recommendation-accent: var(--sp-muted); }
.sp-rate-recommendation strong { color: var(--sp-recommendation-accent); }

:global(.dark) .sp-rate-recommendation {
  background: color-mix(in srgb, var(--sp-recommendation-accent) 11%, var(--sp-panel));
}

.sp-dialog-primary {
  --sp-dialog-button: var(--sp-cyan);
  border-color: color-mix(in srgb, var(--sp-dialog-button) 72%, var(--sp-line));
  background: var(--sp-dialog-button);
  color: white;
  box-shadow: 0 7px 16px color-mix(in srgb, var(--sp-dialog-button) 20%, transparent);
}

.sp-dialog-primary.match { --sp-dialog-button: var(--sp-cyan); }
.sp-dialog-primary.create { --sp-dialog-button: var(--sp-green); }
.sp-dialog-primary.rate { --sp-dialog-button: var(--sp-amber); }
.sp-dialog-primary.log { --sp-dialog-button: #7c3aed; }
.sp-dialog-primary:disabled { box-shadow: none; }

.sp-dialog-actions {
  display: flex;
  justify-content: flex-end;
  gap: 0.75rem;
}

.sp-button.secondary {
  border-color: var(--sp-line);
  background: var(--sp-panel);
  color: var(--sp-text);
}

.sp-spin { animation: sp-spin 800ms linear infinite; }

@media (hover: hover) {
  .sp-control-button-reset:not(:disabled):hover {
    border-color: color-mix(in srgb, var(--sp-text) 28%, var(--sp-line));
    background: color-mix(in srgb, var(--sp-muted) 8%, var(--sp-panel));
    color: var(--sp-text);
  }

  .sp-control-button-match:not(:disabled):hover {
    border-color: color-mix(in srgb, var(--sp-cyan) 58%, var(--sp-line));
    background: color-mix(in srgb, var(--sp-cyan) 15%, var(--sp-panel));
  }

  .sp-control-button-log:not(:disabled):hover {
    border-color: color-mix(in srgb, var(--sp-amber) 62%, var(--sp-line));
    background: color-mix(in srgb, var(--sp-amber) 16%, var(--sp-panel));
  }

  .sp-control-button-refresh:not(:disabled):hover {
    border-color: color-mix(in srgb, var(--sp-green) 58%, var(--sp-line));
    background: color-mix(in srgb, var(--sp-green) 15%, var(--sp-panel));
  }

  .sp-summary-filter:not(:disabled):hover :deep(.stat-card) {
    border-color: color-mix(in srgb, var(--sp-cyan) 34%, var(--sp-line));
    box-shadow: 0 10px 24px rgba(15, 23, 42, 0.1);
    transform: translateY(-2px);
  }
}

@media (max-width: 1280px) {
  .sp-filter-card-body {
    align-items: stretch;
    flex-direction: column;
  }

  .sp-filter-actions {
    width: 100%;
    padding-top: 0.75rem;
    padding-left: 0;
    border-top: 1px solid var(--sp-line);
    border-left: 0;
  }
}

@media (max-width: 760px) {
  .sp-filter-card-head,
  .sp-filter-card-body { padding-inline: 0.75rem; }
  .sp-filter-fields { grid-template-columns: repeat(2, minmax(0, 1fr)); }
  .sp-filter-search-control { grid-column: 1 / -1; }
  .sp-filter-actions { display: grid; grid-template-columns: repeat(2, minmax(0, 1fr)); }
  .sp-filter-actions .sp-button { width: 100%; min-width: 0; padding-inline: 0.45rem; }
  .sp-summary-grid { grid-template-columns: repeat(4, minmax(0, 1fr)); }
  .sp-summary-grid { gap: 0.35rem; }
  .sp-summary-grid :deep(.stat-card) { min-height: 4.75rem; padding: 0.55rem; }
  .sp-summary-grid :deep(.stat-icon),
  .sp-summary-grid :deep(.stat-trend) { display: none; }
  .sp-summary-grid :deep(.stat-label) { font-size: 0.67rem; }
  .sp-summary-grid :deep(.stat-value) { margin-top: 0.3rem; font-size: 1.25rem; }
  .sp-provider-shortcuts {
    width: 100%;
    max-width: 100%;
    justify-content: flex-start;
    padding-inline: 0;
  }
  .sp-provider-shortcut {
    max-width: 7.5rem;
  }
  .sp-attention-shortcuts { padding-inline: 0.55rem; }
  .sp-table-shell { height: auto; min-height: 0; overflow: visible; }
  .sp-panel-signals { width: 100%; justify-content: space-between; }
  .sp-row-actions { flex-wrap: wrap; justify-content: flex-end; }
  .sp-match-preview { grid-template-columns: 1fr; }
  .sp-dialog-actions { width: 100%; }
  .sp-dialog-actions .sp-button { flex: 1 1 0; }
}

@media (max-width: 520px) {
  /* 手机端筛选区保持紧凑：搜索通栏，下拉与按钮 2 列，避免全部整行占高 */
  .sp-filter-fields {
    grid-template-columns: repeat(2, minmax(0, 1fr));
    gap: 0.625rem;
  }

  .sp-filter-search-control {
    grid-column: 1 / -1;
  }

  .sp-filter-actions {
    gap: 0.5rem;
  }

  .sp-filter-actions .sp-button {
    min-height: 2.5rem;
    padding-inline: 0.55rem;
    font-size: 0.8125rem;
    white-space: normal;
    line-height: 1.25;
    text-align: center;
  }
}

@media (max-width: 390px) {
  .sp-summary-grid { gap: 0.25rem; }
  .sp-summary-grid :deep(.stat-card) { min-height: 4.25rem; padding: 0.45rem; }
  .sp-summary-grid :deep(.stat-label) { font-size: 0.625rem; }
  .sp-summary-grid :deep(.stat-value) { font-size: 1.1rem; }
}

@media (prefers-reduced-motion: reduce) {
  .sp-summary-grid :deep(.stat-card) { transition: none; animation: none; }
  .sp-spin { animation: none; }
}

@keyframes sp-summary-enter {
  from { opacity: 0; transform: translateY(5px); }
  to { opacity: 1; transform: translateY(0); }
}

@keyframes sp-spin {
  to { transform: rotate(360deg); }
}
</style>

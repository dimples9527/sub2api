<template>
  <BaseDialog
    :show="show"
    title="配置健康守护账号"
    width="full"
    :z-index="60"
    @close="closeHealthGuardAccounts"
  >
    <div class="sp-health-guard-account-dialog">
      <section class="sp-health-guard-platform-models">
        <div class="sp-health-guard-dialog-section-head">
          <strong>平台默认测试模型</strong>
          <span>账号未单独设置模型时使用对应平台的默认值。</span>
        </div>
        <div v-if="healthGuardPlatformSummaries.length" class="sp-health-guard-platform-model-grid">
          <article v-for="summary in healthGuardPlatformSummaries" :key="summary.platform">
            <div :class="platformTextClass(summary.platform)">
              <strong>{{ platformLabel(summary.platform) }}</strong>
              <span>{{ summary.accountCount }} 个可选账号</span>
            </div>
            <Select
              v-model="config.account_health_guard_platform_models[summary.platform]"
              :options="healthGuardModelSelectOptions(summary.platform, config.account_health_guard_platform_models[summary.platform])"
              searchable
              clearable
              creatable
              :creatable-prefix="healthGuardModelCreatablePrefix"
              :placeholder="healthGuardModelLoadingByPlatform[summary.platform] ? '加载模型中…' : '选择平台默认模型'"
              empty-text="暂无可用模型"
            />
          </article>
        </div>
        <div v-else class="sp-rate-guard-empty">当前没有可配置默认模型的平台。</div>
      </section>

      <!-- 倍率区间按平台铺开后，7 个平台会把账号工作区挤到看不见（弹窗高度受 100dvh 约束），
           因此这里只放「摘要 + 入口」，实际区间配置放进二级弹窗。 -->
      <section v-if="config.account_health_guard_platform_multiplier_intervals_enabled" class="sp-health-guard-platform-models sp-health-guard-multiplier-entry-section">
        <div class="sp-health-guard-dialog-section-head">
          <strong>未开调度账号按倍率间隔</strong>
          <span>仅对未开启调度的账号生效：按账号所属平台 + 计费倍率落入的区间取检查间隔（覆盖账号级间隔）。区间取 [下限, 上限)，上限留空表示无上界；未命中任何区间的账号每轮都会检查。间隔不得低于 60 秒。</span>
        </div>
        <div class="sp-health-guard-multiplier-entry">
          <span>{{ healthGuardMultiplierSummary }}</span>
          <button
            class="sp-button small ghost sp-health-guard-multiplier-entry-button"
            type="button"
            @click="openMultiplierIntervalDialog"
          >
            <Icon name="cog" size="sm" />
            配置倍率区间
          </button>
        </div>
      </section>

      <section class="sp-health-guard-account-workspace">
        <div class="sp-health-guard-selection-summary" aria-label="健康守护账号配置摘要">
          <article>
            <span>已选账号</span>
            <strong>{{ healthGuardSelectionSummary.selected }}</strong>
          </article>
          <article>
            <span>使用平台默认模型</span>
            <strong>{{ healthGuardSelectionSummary.platformDefault }}</strong>
          </article>
          <article>
            <span>账号模型覆盖</span>
            <strong>{{ healthGuardSelectionSummary.overridden }}</strong>
          </article>
          <article>
            <span>已设检查间隔</span>
            <strong>{{ healthGuardSelectionSummary.intervals }}</strong>
          </article>
          <article :class="{ warning: healthGuardSelectionSummary.missingModel > 0 }">
            <span>缺少有效模型</span>
            <strong>{{ healthGuardSelectionSummary.missingModel }}</strong>
          </article>
        </div>

        <div class="sp-health-guard-account-toolbar">
          <div class="sp-health-guard-account-filters">
            <Select
              v-model="healthGuardAccountProviderFilter"
              :options="healthGuardProviderFilterOptions"
              searchable
              placeholder="供应商过滤"
              empty-text="暂无供应商"
            />
            <Input v-model="healthGuardAccountSearch" placeholder="搜索账号名称或供应商来源" />
            <!-- 两个快捷过滤是同一维度的互斥方向（参与 / 未参与守护），共用一个 flex 容器
                 塞进 grid 的第 3 列。不能把第二个按钮直接铺成 grid 子项：桌面端列数会被
                 撑到 4，多出来的那个只能掉到第二行、被 minmax(160px, 0.42fr) 拉成一整块宽按钮。 -->
            <div class="sp-health-guard-quick-filters" role="group" aria-label="健康守护账号快捷过滤">
              <button
                class="sp-health-guard-selected-toggle"
                :class="{ active: healthGuardSelectedOnly }"
                type="button"
                :aria-pressed="healthGuardSelectedOnly"
                @click="toggleHealthGuardSelectedOnly"
              >
                <span class="sp-health-guard-selected-toggle-mark" aria-hidden="true"></span>
                仅看已选
                <strong>{{ healthGuardSelectionSummary.selected }}</strong>
              </button>
              <button
                class="sp-health-guard-selected-toggle"
                :class="{ active: healthGuardUnselectedOnly }"
                type="button"
                :aria-pressed="healthGuardUnselectedOnly"
                @click="toggleHealthGuardUnselectedOnly"
              >
                <span class="sp-health-guard-selected-toggle-mark" aria-hidden="true"></span>
                仅看未开启
                <strong>{{ healthGuardUnselectedCount }}</strong>
              </button>
              <!-- 分隔线右侧是「供应商是否开启」维度：与左边那两个正交、可以叠加
                   （例如「未参与守护 + 供应商已关闭」一起看），但这一对内部互斥。 -->
              <span class="sp-health-guard-batch-divider" aria-hidden="true"></span>
              <button
                class="sp-health-guard-selected-toggle"
                :class="{ active: healthGuardProviderEnabledOnly }"
                type="button"
                :aria-pressed="healthGuardProviderEnabledOnly"
                title="只显示上游供应商仍在启用的账号"
                @click="toggleHealthGuardProviderEnabledOnly"
              >
                <span class="sp-health-guard-selected-toggle-mark" aria-hidden="true"></span>
                仅看供应商开启
                <strong>{{ healthGuardProviderEnabledCount }}</strong>
              </button>
              <button
                class="sp-health-guard-selected-toggle"
                :class="{ active: healthGuardProviderClosedOnly }"
                type="button"
                :aria-pressed="healthGuardProviderClosedOnly"
                title="只显示上游供应商已停用的账号"
                @click="toggleHealthGuardProviderClosedOnly"
              >
                <span class="sp-health-guard-selected-toggle-mark" aria-hidden="true"></span>
                仅看供应商关闭
                <strong>{{ healthGuardProviderClosedCount }}</strong>
              </button>
            </div>
          </div>
          <span class="sp-health-guard-filter-result">筛选结果 <strong>{{ healthGuardWorkspaceAccounts.length }}</strong> 个</span>
          <!-- 平台标签与两个分组弹窗共用同一套控件与配色语言，独占一行（flex-basis: 100%）。
               必须放在 grid 容器之外：.sp-health-guard-account-filters 是 grid，
               flex-basis 在 grid 里不生效，塞进去只会把那几列挤变形。
               没有可选平台时不渲染，省掉一条空行。 -->
          <div
            v-if="healthGuardAccountPlatformFacets.length"
            class="sp-platform-chip-row"
            role="group"
            aria-label="按平台筛选账号"
          >
            <button
              v-for="facet in healthGuardAccountPlatformFacets"
              :key="facet.platform"
              class="sp-platform-chip"
              type="button"
              :aria-pressed="healthGuardAccountPlatformFilter.includes(facet.platform)"
              :style="{ '--sp-chip-platform-color': platformAccentColor(facet.platform) }"
              @click="healthGuardAccountPlatformFilter = togglePlatformFilter(healthGuardAccountPlatformFilter, facet.platform)"
            >
              {{ facet.label }}
              <strong>{{ facet.count }}</strong>
            </button>
          </div>

          <!-- 批量配置：全选/取消全选作用于「当前筛选结果」；间隔应用/清除作用于「当前筛选结果里已勾选且可用」的账号，
               所以可以按筛选分批设不同间隔而互不覆盖。独占一行（flex-basis: 100%），避免挤压上方 grid 列。 -->
          <div class="sp-health-guard-batch-actions" role="group" aria-label="批量配置账号">
            <button class="sp-button small ghost" type="button" @click="selectAllFilteredHealthGuardAccounts">
              全选筛选结果
            </button>
            <button class="sp-button small ghost" type="button" @click="deselectFilteredHealthGuardAccounts">
              取消全选
            </button>
            <span class="sp-health-guard-batch-divider" aria-hidden="true"></span>
            <div class="sp-health-guard-batch-interval-input">
              <Input
                v-model="healthGuardBatchIntervalInput"
                type="number"
                min="60"
                placeholder="间隔（秒）"
                aria-label="批量检查间隔（秒），不小于 60 秒"
                title="批量为当前筛选结果中已勾选且可用的账号设置检查间隔，不小于 60 秒"
              />
            </div>
            <button
              class="sp-button small"
              type="button"
              :disabled="!healthGuardBatchIntervalValid || healthGuardBatchTargetCount === 0"
              title="为当前筛选结果中已勾选且可用的账号统一设置检查间隔（可按筛选分批设不同值）"
              @click="applyHealthGuardBatchInterval"
            >
              应用到已选
            </button>
            <button
              class="sp-button small ghost danger"
              type="button"
              :disabled="healthGuardBatchTargetCount === 0"
              title="清除当前筛选结果中已勾选账号的检查间隔，回落到默认行为"
              @click="clearHealthGuardSelectedIntervals"
            >
              清除已选间隔
            </button>
            <span class="sp-health-guard-batch-divider" aria-hidden="true"></span>
            <button
              class="sp-button small ghost"
              type="button"
              :disabled="healthGuardBatchTargetCount === 0"
              title="批量为当前筛选结果中已勾选且可用的账号设置修改调度、账号级测试模型和三项阈值"
              @click="openHealthGuardBatchSettings"
            >
              批量设置
            </button>
          </div>
        </div>

        <div v-if="loadingAccounts" class="sp-rate-guard-empty">正在加载账号...</div>
        <div v-else-if="healthGuardWorkspaceAccounts.length" class="sp-health-guard-account-list">
          <article
            v-for="mapping in healthGuardWorkspaceAccounts"
            :key="mapping.localAccountID"
            class="sp-health-guard-account-row"
            :class="{
              selected: healthGuardAccountIsSelected(mapping.localAccountID),
              unavailable: !mapping.available,
              'provider-closed': healthGuardAccountProviderClosed(mapping),
              'missing-model': healthGuardAccountIsSelected(mapping.localAccountID)
                && mapping.available
                && !supplierAccountHealthGuardModelForMapping(config, mapping),
            }"
          >
            <!-- 多选列：只决定「批量设置 / 批量间隔」的作用范围，是纯界面状态（不写进 config）。
                 与右侧「参与守护」开关刻意分开：开关是业务配置、保存后生效；
                 若用一个开关兼作多选，想批量设阈值就得先把账号纳入守护，两件事会被绑死。 -->
            <label class="sp-health-guard-account-check">
              <input
                type="checkbox"
                :checked="healthGuardAccountIsChecked(mapping.localAccountID)"
                :aria-label="`选择账号 ${mapping.localAccountName} 用于批量设置`"
                @change="toggleHealthGuardAccountCheck(mapping.localAccountID)"
              />
            </label>

            <div
              class="sp-health-guard-account-choice"
              :title="mapping.available ? healthGuardSourceSummary(mapping) : '账号已停用、删除或匹配失效，运行时将记录为不可用'"
            >
              <span class="sp-health-guard-account-choice-copy">
                <strong :class="platformTextClass(mapping.platform)">{{ mapping.localAccountName }}{{ healthGuardAccountMultiplierText(mapping) }}</strong>
                <span
                  v-if="mapping.available"
                  :class="['sp-health-guard-account-platform', platformBadgeClass(mapping.platform)]"
                >
                  {{ platformLabel(mapping.platform) }}
                </span>
                <span class="sp-health-guard-account-id">#{{ mapping.localAccountID }}</span>
                <span
                  class="sp-health-guard-account-source"
                  :class="{ unavailable: !mapping.available }"
                >
                  {{ mapping.available ? healthGuardSourceSummary(mapping) : '当前不可用' }}
                </span>
                <span
                  v-if="healthGuardAccountIsSelected(mapping.localAccountID) && !mapping.available"
                  class="sp-health-guard-model-status unavailable"
                >账号不可用</span>
                <span
                  v-else-if="healthGuardAccountIsSelected(mapping.localAccountID) && healthGuardAccountOverrideModel(mapping.localAccountID)"
                  class="sp-health-guard-model-status override"
                >账号覆盖</span>
                <!-- 「平台默认」而不是「默认」：单说「默认」看不出默认的是哪一层的东西；
                     和上一档「账号覆盖」配成一对，才读得出这是在说「测试模型取自哪里」。 -->
                <span
                  v-else-if="healthGuardAccountIsSelected(mapping.localAccountID) && healthGuardPlatformDefaultModel(mapping.platform)"
                  class="sp-health-guard-model-status"
                >平台默认</span>
                <span
                  v-else-if="healthGuardAccountIsSelected(mapping.localAccountID)"
                  class="sp-health-guard-model-status missing"
                >未设模型</span>
                <span
                  v-if="healthGuardAccountIsSelected(mapping.localAccountID) && mapping.available && healthGuardAccountIntervalValue(mapping.localAccountID)"
                  class="sp-health-guard-model-status interval"
                >间隔 {{ healthGuardAccountIntervalValue(mapping.localAccountID) }}s</span>
              </span>
            </div>

            <!-- 开关列：表示「该账号是否参与守护」，是业务配置，保存任务后生效。
                 放在账号信息之后：账号名紧跟勾选框，开关不再抢占行首。 -->
            <label
              class="sp-health-guard-account-guard-toggle"
              :title="mapping.available
                ? '开启后该账号参与健康守护检查'
                : '账号已停用、删除或匹配失效，开启后运行时仍会记录为不可用'"
            >
              <Toggle
                :model-value="healthGuardAccountIsSelected(mapping.localAccountID)"
                :aria-label="`${healthGuardAccountIsSelected(mapping.localAccountID) ? '关闭' : '开启'}账号 ${mapping.localAccountName} 的健康守护`"
                @update:model-value="toggleHealthGuardAccount(mapping.localAccountID)"
              />
              <span>参与守护</span>
            </label>

            <div
              class="sp-health-guard-account-group-summary"
              :title="healthGuardAccountGroupsTitle(mapping)"
              :aria-label="healthGuardAccountGroupsTitle(mapping)"
            >
              <span>绑定分组</span>
              <strong>{{ healthGuardAccountGroupCount(mapping) }} 个</strong>
            </div>

            <div
              v-if="healthGuardAccountIsSelected(mapping.localAccountID) && mapping.available"
              class="sp-health-guard-account-model-editor"
            >
              <Select
                v-model="config.account_health_guard_account_models[String(mapping.localAccountID)]"
                :options="healthGuardModelSelectOptions(mapping.platform, config.account_health_guard_account_models[String(mapping.localAccountID)])"
                searchable
                clearable
                creatable
                :creatable-prefix="healthGuardModelCreatablePrefix"
                placeholder="平台默认模型"
                empty-text="暂无可用模型"
              />
              <Input
                :model-value="healthGuardAccountIntervalValue(mapping.localAccountID)"
                type="number"
                min="60"
                placeholder="检查间隔（秒）"
                :aria-label="`账号 ${mapping.localAccountName} 检查间隔（秒），留空表示按任务全局执行间隔`"
                title="为该账号单独设置检查频率，留空表示按任务全局执行间隔"
                @update:model-value="setHealthGuardAccountInterval(mapping.localAccountID, $event)"
              />
              <Input
                :model-value="healthGuardAccountThresholdValue('account_health_guard_account_failure_thresholds', mapping.localAccountID)"
                type="number"
                min="1"
                :placeholder="`失败阈值（全局 ${config.account_health_guard_failure_threshold}）`"
                :aria-label="`账号 ${mapping.localAccountName} 连续失败暂停阈值，留空表示沿用全局阈值 ${config.account_health_guard_failure_threshold}`"
                :title="`连续失败达到该次数后自动暂停该账号调度，留空表示沿用全局阈值 ${config.account_health_guard_failure_threshold}`"
                @update:model-value="setHealthGuardAccountThreshold('account_health_guard_account_failure_thresholds', mapping.localAccountID, $event)"
              />
              <Input
                :model-value="healthGuardAccountThresholdValue('account_health_guard_account_slow_thresholds', mapping.localAccountID)"
                type="number"
                min="1"
                :placeholder="`慢响应阈值（全局 ${config.account_health_guard_slow_threshold}）`"
                :aria-label="`账号 ${mapping.localAccountName} 连续慢响应暂停阈值，留空表示沿用全局阈值 ${config.account_health_guard_slow_threshold}`"
                :title="`连续慢响应达到该次数后自动暂停该账号调度，留空表示沿用全局阈值 ${config.account_health_guard_slow_threshold}`"
                @update:model-value="setHealthGuardAccountThreshold('account_health_guard_account_slow_thresholds', mapping.localAccountID, $event)"
              />
              <Input
                :model-value="healthGuardAccountThresholdValue('account_health_guard_account_recovery_thresholds', mapping.localAccountID)"
                type="number"
                min="1"
                :placeholder="`恢复阈值（全局 ${config.account_health_guard_recovery_threshold}）`"
                :aria-label="`账号 ${mapping.localAccountName} 连续健康恢复阈值，留空表示沿用全局阈值 ${config.account_health_guard_recovery_threshold}`"
                :title="`连续健康达到该次数后自动恢复该账号调度，留空表示沿用全局阈值 ${config.account_health_guard_recovery_threshold}`"
                @update:model-value="setHealthGuardAccountThreshold('account_health_guard_account_recovery_thresholds', mapping.localAccountID, $event)"
              />
              <label class="sp-health-guard-account-scheduling-toggle">
                <Toggle
                  :model-value="healthGuardAccountSchedulingChangeValue(mapping.localAccountID)"
                  :aria-label="`账号 ${mapping.localAccountName} 是否修改调度`"
                  :title="healthGuardAccountSchedulingChangeTitle(mapping.localAccountID)"
                  @update:model-value="setHealthGuardAccountSchedulingChange(mapping.localAccountID, $event)"
                />
                <span>修改调度</span>
              </label>
            </div>
          </article>
        </div>
        <div v-else class="sp-rate-guard-empty">
          {{ healthGuardEmptyHint }}
        </div>
      </section>
    </div>
    <template #footer>
      <button class="sp-button primary" type="button" @click="emit('confirm')">完成</button>
    </template>
  </BaseDialog>
  <BaseDialog
    :show="multiplierIntervalDialogVisible"
    title="配置倍率区间"
    width="extra-wide"
    :z-index="70"
    @close="closeMultiplierIntervalDialog"
  >
    <div class="sp-multiplier-interval-dialog">
      <div v-if="healthGuardPlatformSummaries.length" class="sp-health-guard-multiplier-grid">
        <article v-for="summary in healthGuardPlatformSummaries" :key="summary.platform">
          <div class="sp-health-guard-multiplier-head" :class="platformTextClass(summary.platform)">
            <strong>{{ platformLabel(summary.platform) }}</strong>
            <button class="sp-button small ghost" type="button" @click="addHealthGuardMultiplierRule(summary.platform)">
              <Icon name="plus" size="sm" />
              新增区间
            </button>
          </div>
          <div v-if="healthGuardMultiplierRules(summary.platform).length" class="sp-health-guard-multiplier-rows">
            <div class="sp-health-guard-multiplier-row sp-health-guard-multiplier-row-head">
              <span>倍率下限（含）</span>
              <span>倍率上限（不含，留空=无上界）</span>
              <span>检查间隔（秒）</span>
              <span></span>
            </div>
            <div v-for="(rule, index) in healthGuardMultiplierRules(summary.platform)" :key="index" class="sp-health-guard-multiplier-row">
              <Input :model-value="rule.min_multiplier" type="number" step="0.1" min="0" @update:model-value="rule.min_multiplier = toNumber($event, rule.min_multiplier)" />
              <Input :model-value="rule.max_multiplier || ''" type="number" step="0.1" min="0" placeholder="无上界" @update:model-value="rule.max_multiplier = toNumber($event, 0)" />
              <Input :model-value="rule.interval_seconds" type="number" min="60" @update:model-value="rule.interval_seconds = toNumber($event, rule.interval_seconds)" />
              <button class="sp-button small ghost danger" type="button" @click="removeHealthGuardMultiplierRule(summary.platform, index)">
                <Icon name="trash" size="sm" />
              </button>
            </div>
          </div>
          <div v-else class="sp-rate-guard-empty">未配置区间：该平台下未开调度的账号每轮都会检查。</div>
        </article>
      </div>
      <div v-else class="sp-rate-guard-empty">当前没有可配置倍率间隔的平台。</div>
    </div>
    <template #footer>
      <span class="sp-multiplier-interval-hint">区间取 [下限, 上限)，上限留空表示无上界；未命中任何区间的账号每轮都会检查。</span>
      <button class="sp-button primary" type="button" @click="closeMultiplierIntervalDialog">完成</button>
    </template>
  </BaseDialog>
  <BaseDialog
    :show="healthGuardBatchSettingsVisible"
    title="批量设置账号"
    width="extra-wide"
    :z-index="65"
    @close="closeHealthGuardBatchSettings"
  >
    <div class="sp-health-guard-batch-dialog">
      <div class="sp-health-guard-batch-rows">
        <div class="sp-health-guard-batch-row">
          <div>
            <strong>参与守护</strong>
            <span>「纳入守护」把勾选账号加入检查名单；「移出守护」会清空该账号的账号级配置，因此选它时本次其余项对该批账号不再生效。</span>
          </div>
          <Select
            v-model="healthGuardBatchGuardInput"
            :options="healthGuardBatchGuardOptions"
            :searchable="false"
            aria-label="批量设置参与守护"
          />
        </div>
        <div class="sp-health-guard-batch-row">
          <div>
            <strong>修改调度</strong>
            <span>「跟随全局」会清掉该账号的账号级覆盖，之后随任务的全局开关一起变。</span>
          </div>
          <Select
            v-model="healthGuardBatchSchedulingChangeInput"
            :options="healthGuardBatchSchedulingChangeOptions"
            :searchable="false"
            aria-label="批量设置修改调度"
          />
        </div>
        <div class="sp-health-guard-batch-row">
          <div>
            <strong>账号级测试模型</strong>
            <span>写入账号级覆盖，优先级高于「平台默认测试模型」；跨平台账号请先确认该模型可用。</span>
          </div>
          <Select
            v-model="healthGuardBatchModelInput"
            :options="healthGuardBatchModelOptions"
            searchable
            creatable
            :creatable-prefix="healthGuardModelCreatablePrefix"
            aria-label="批量设置账号级测试模型"
          />
        </div>
        <div class="sp-health-guard-batch-row">
          <div>
            <strong>连续失败暂停阈值</strong>
            <span>写入账号级覆盖，留空的账号继续沿用全局阈值。</span>
          </div>
          <Input
            v-model="healthGuardBatchFailureThresholdInput"
            type="number"
            min="1"
            placeholder="不修改"
            aria-label="批量设置连续失败暂停阈值"
          />
        </div>
        <div class="sp-health-guard-batch-row">
          <div>
            <strong>连续慢响应暂停阈值</strong>
            <span>写入账号级覆盖，留空的账号继续沿用全局阈值。</span>
          </div>
          <Input
            v-model="healthGuardBatchSlowThresholdInput"
            type="number"
            min="1"
            placeholder="不修改"
            aria-label="批量设置连续慢响应暂停阈值"
          />
        </div>
        <div class="sp-health-guard-batch-row">
          <div>
            <strong>连续健康恢复阈值</strong>
            <span>写入账号级覆盖，留空的账号继续沿用全局阈值。</span>
          </div>
          <Input
            v-model="healthGuardBatchRecoveryThresholdInput"
            type="number"
            min="1"
            placeholder="不修改"
            aria-label="批量设置连续健康恢复阈值"
          />
        </div>
      </div>
    </div>
    <template #footer>
      <span class="sp-health-guard-batch-hint">
        只写入填过的项，留空表示保持原值；作用于当前筛选结果里已勾选的 {{ healthGuardBatchTargetCount }} 个可用账号，保存任务后生效。检查间隔请用工具栏的「应用到已选」。
      </span>
      <button
        class="sp-button primary"
        type="button"
        :disabled="healthGuardBatchTargetCount === 0"
        @click="applyHealthGuardBatchSettings"
      >应用到已选</button>
    </template>
  </BaseDialog>
</template>

<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import BaseDialog from '@/components/common/BaseDialog.vue'
import Input from '@/components/common/Input.vue'
import Select, { type SelectOption } from '@/components/common/Select.vue'
import Toggle from '@/components/common/Toggle.vue'
import Icon from '@/components/icons/Icon.vue'
import type { SupplierProviderAccount } from '@/api/admin/supplierProviderData'
import { adminAPI } from '@/api/admin'
import type {
  SupplierAccountHealthGuardMultiplierInterval,
  SupplierAutomationConfig,
} from '@/api/admin/supplierAutomation'
import { useAppStore } from '@/stores/app'
import { resolvePlatformDisplayLabel as platformLabel } from '@/utils/customPlatformLabels'
import { platformAccentColor, platformBadgeClass, platformTextClass } from '@/utils/platformColors'
import { extractApiErrorMessage } from '@/utils/apiError'
import {
  HEALTH_GUARD_BATCH_GUARD_KEEP,
  HEALTH_GUARD_BATCH_MODEL_CLEAR,
  HEALTH_GUARD_BATCH_MODEL_KEEP,
  HEALTH_GUARD_BATCH_SCHEDULING_KEEP,
  buildHealthGuardAccountMappings,
  healthGuardSelectedRowsForConfig,
  buildPlatformFacets,
  matchesPlatformFilter,
  normalizeAccountHealthGuardAccountThresholds,
  normalizeHealthGuardPlatform,
  normalizePositiveAccountIDs,
  normalizeStringMap,
  supplierAccountHealthGuardModelForMapping,
  toNumber,
  togglePlatformFilter,
  type HealthGuardAccountMapping,
  type HealthGuardAccountThresholdField,
  type HealthGuardPlatformSummary,
} from '@/views/admin/supplier-management/supplierAutomationConfig'

const props = defineProps<{
  show: boolean
  accounts: SupplierProviderAccount[]
  loadingAccounts: boolean
  disabledProviderIds: Set<number>
}>()

// 配置草稿用 v-model 双向绑定而不是普通 prop：本弹窗没有自己的「保存」，
// 它做的就是就地修改调用方（任务编辑弹窗 / 上游账号页）持有的那份 config。
// 用 defineModel 既能直接改草稿，又不会触发 vue/no-mutating-props（改的是 ref 里的对象，不是 prop 本身）。
const config = defineModel<SupplierAutomationConfig>('config', { required: true })

// 「完成」与「关闭」必须分开：本弹窗没有自己的保存动作，改的是调用方持有的 config。
// 在自动化页两者等价（都只是收起子弹窗，由任务编辑弹窗的「保存」统一提交）；
// 但上游账号页是原地打开、没有父级「保存」兜底，只能靠「完成」落库 ——
// 若把它和 ✕ / 点遮罩 / Esc 混成一个事件，就会变成「点一下遮罩就把配置写进库」。
const emit = defineEmits<{ close: []; confirm: [] }>()

const appStore = useAppStore()

/** 账号候选映射由父组件持有的数据派生；页面与弹窗共用同一份账号列表。 */
const mappings = computed<HealthGuardAccountMapping[]>(() =>
  buildHealthGuardAccountMappings(props.accounts)
)

const multiplierIntervalDialogVisible = ref(false)

const healthGuardAccountSearch = ref('')

const healthGuardSelectedOnly = ref(false)

const healthGuardUnselectedOnly = ref(false)

const healthGuardProviderClosedOnly = ref(false)

const healthGuardProviderEnabledOnly = ref(false)

const healthGuardBatchIntervalInput = ref<number | string>('')

const healthGuardModelOptionsByPlatform = ref<Record<string, { id: string; display_name?: string }[]>>({})

const healthGuardModelLoadingByPlatform = ref<Record<string, boolean>>({})

const healthGuardAvailableAccountMappings = computed(() =>
  mappings.value.filter(mapping => mapping.available)
)

const healthGuardSelectedAccountRows = computed(() =>
  healthGuardSelectedRowsForConfig(config.value, mappings.value)
)

const healthGuardPlatformSummaries = computed<HealthGuardPlatformSummary[]>(() => {
  const summaries = new Map<string, HealthGuardPlatformSummary>()
  for (const mapping of healthGuardAvailableAccountMappings.value) {
    const platform = mapping.platform
    const current = summaries.get(platform)
    if (current) {
      current.accountCount += 1
      continue
    }
    summaries.set(platform, {
      platform,
      accountCount: 1,
    })
  }
  return Array.from(summaries.values()).sort((a, b) => platformLabel(a.platform).localeCompare(platformLabel(b.platform), 'zh-CN'))
})

const healthGuardMultiplierSummary = computed(() => {
  const total = healthGuardPlatformSummaries.value.length
  if (total === 0) return '当前没有可配置倍率间隔的平台。'
  const configured = healthGuardPlatformSummaries.value.filter(
    summary => healthGuardMultiplierRules(summary.platform).length > 0
  ).length
  if (configured === 0) return `共 ${total} 个平台，均未配置区间（每轮都会检查）。`
  return `已为 ${configured}/${total} 个平台配置区间，其余平台每轮都会检查。`
})

function openMultiplierIntervalDialog() {
  multiplierIntervalDialogVisible.value = true
}

function closeMultiplierIntervalDialog() {
  multiplierIntervalDialogVisible.value = false
}

const healthGuardProviderScopedMappings = computed(() => {
  const providerID = healthGuardAccountProviderFilter.value
  if (!providerID) return healthGuardAvailableAccountMappings.value
  return healthGuardAvailableAccountMappings.value.filter(mapping =>
    mapping.sources.some(source => String(source.provider_id) === providerID)
  )
})

const healthGuardPlatformScopedMappings = computed(() => {
  const selected = healthGuardAccountPlatformFilter.value
  if (!selected.length) return healthGuardAvailableAccountMappings.value
  return healthGuardAvailableAccountMappings.value.filter(mapping =>
    matchesPlatformFilter(mapping.platform, selected)
  )
})

const healthGuardWorkspaceAccounts = computed(() => {
  const accounts = [...healthGuardAvailableAccountMappings.value]
  const includedAccountIDs = new Set(accounts.map(mapping => mapping.localAccountID))
  for (const mapping of healthGuardSelectedAccountRows.value) {
    if (includedAccountIDs.has(mapping.localAccountID)) continue
    accounts.push(mapping)
    includedAccountIDs.add(mapping.localAccountID)
  }

  const platformFilter = healthGuardAccountPlatformFilter.value
  const providerID = healthGuardAccountProviderFilter.value
  const keyword = healthGuardAccountSearch.value.trim().toLowerCase()
  const filtered = accounts.filter(mapping => {
    if (healthGuardSelectedOnly.value && !healthGuardAccountIDs.value.includes(mapping.localAccountID)) return false
    if (healthGuardUnselectedOnly.value && healthGuardAccountIDs.value.includes(mapping.localAccountID)) return false
    if (healthGuardProviderClosedOnly.value && !healthGuardAccountProviderClosed(mapping)) return false
    if (healthGuardProviderEnabledOnly.value && healthGuardAccountProviderClosed(mapping)) return false
    if (!matchesPlatformFilter(mapping.platform, platformFilter)) return false
    if (providerID && !mapping.sources.some(source => String(source.provider_id) === providerID)) return false
    if (!keyword) return true
    const searchableText = [
      mapping.localAccountName,
      String(mapping.localAccountID),
      mapping.platform,
      ...mapping.localGroupPlatforms,
      ...mapping.sources.flatMap(source => [source.name, source.provider_name, source.upstream_account_key]),
    ].filter(Boolean).join(' ').toLowerCase()
    return searchableText.includes(keyword)
  })
  // 按倍率升序：倍率区间规则是按倍率落桶取间隔的，同一桶的账号相邻才看得出「这批账号走同一条区间」。
  // 排序放在筛选之后，列表顺序不随筛选条件跳变（每次都从同一基准重排）。
  return filtered.sort((a, b) => healthGuardAccountMultiplierSortKey(a) - healthGuardAccountMultiplierSortKey(b))
})

function healthGuardAccountMultiplierSortKey(mapping: HealthGuardAccountMapping): number {
  const rates = mapping.sources
    .map(source => Number(source.rate_multiplier))
    .filter(rate => Number.isFinite(rate))
  return rates.length ? Math.min(...rates) : Number.POSITIVE_INFINITY
}

const healthGuardSelectionSummary = computed(() => {
  const accountModels = normalizeStringMap(config.value.account_health_guard_account_models)
  const platformModels = normalizeStringMap(config.value.account_health_guard_platform_models)
  let platformDefault = 0
  let overridden = 0
  let missingModel = 0
  let intervals = 0

  for (const mapping of healthGuardSelectedAccountRows.value) {
    if (!mapping.available) continue
    if (accountModels[String(mapping.localAccountID)]?.trim()) {
      overridden += 1
    } else if (platformModels[mapping.platform]?.trim()) {
      platformDefault += 1
    } else {
      missingModel += 1
    }
    if (healthGuardAccountIntervalValue(mapping.localAccountID)) intervals += 1
  }

  return {
    selected: healthGuardAccountIDs.value.length,
    platformDefault,
    overridden,
    missingModel,
    intervals,
  }
})

const healthGuardUnselectedCount = computed(() =>
  healthGuardAvailableAccountMappings.value.filter(
    mapping => !healthGuardAccountIDs.value.includes(mapping.localAccountID)
  ).length
)

function healthGuardAccountProviderClosed(mapping: HealthGuardAccountMapping): boolean {
  if (!mapping.sources.length) return false
  return mapping.sources.every(source =>
    props.disabledProviderIds.has(Number(source.provider_id))
  )
}

const healthGuardProviderClosedCount = computed(() =>
  healthGuardAvailableAccountMappings.value.filter(
    mapping => healthGuardAccountProviderClosed(mapping)
  ).length
)

const healthGuardProviderEnabledCount = computed(() =>
  healthGuardAvailableAccountMappings.value.length - healthGuardProviderClosedCount.value
)

const healthGuardEmptyHint = computed(() => {
  if (healthGuardSelectedOnly.value) return '当前筛选条件下没有已选账号。'
  if (healthGuardUnselectedOnly.value) return '当前筛选条件下没有未参与守护的账号。'
  if (healthGuardProviderClosedOnly.value) return '当前筛选条件下没有供应商已关闭的账号。'
  if (healthGuardProviderEnabledOnly.value) return '当前筛选条件下没有供应商开启的账号。'
  return '当前筛选条件下没有可配置账号。'
})

function toggleHealthGuardSelectedOnly() {
  healthGuardSelectedOnly.value = !healthGuardSelectedOnly.value
  if (healthGuardSelectedOnly.value) healthGuardUnselectedOnly.value = false
}

function toggleHealthGuardUnselectedOnly() {
  healthGuardUnselectedOnly.value = !healthGuardUnselectedOnly.value
  if (healthGuardUnselectedOnly.value) healthGuardSelectedOnly.value = false
}

function toggleHealthGuardProviderClosedOnly() {
  healthGuardProviderClosedOnly.value = !healthGuardProviderClosedOnly.value
  if (healthGuardProviderClosedOnly.value) healthGuardProviderEnabledOnly.value = false
}

function toggleHealthGuardProviderEnabledOnly() {
  healthGuardProviderEnabledOnly.value = !healthGuardProviderEnabledOnly.value
  if (healthGuardProviderEnabledOnly.value) healthGuardProviderClosedOnly.value = false
}

function healthGuardSourceSummary(mapping: HealthGuardAccountMapping): string {
  const sources = mapping.sources
    .map(source => `${source.provider_name || `供应商 ${source.provider_id}`} · ${source.name || source.upstream_account_key}`)
    .filter(Boolean)
  return sources.length ? sources.join('；') : '供应商来源不可用'
}

function healthGuardAccountMultiplierText(mapping: HealthGuardAccountMapping): string {
  const rates = Array.from(new Set(
    mapping.sources
      .map(source => Number(source.rate_multiplier))
      .filter(rate => Number.isFinite(rate))
      .map(rate => rate.toFixed(4).replace(/\.?0+$/, ''))
  ))
  if (!rates.length) return ''
  return `（倍率：${rates.join(' / ')}）`
}

function healthGuardAccountGroups(mapping: HealthGuardAccountMapping): SupplierProviderAccount['binding_groups'] {
  const groups = new Map<number, SupplierProviderAccount['binding_groups'][number]>()
  for (const source of mapping.sources) {
    for (const group of source.binding_groups) {
      if (!groups.has(group.id)) groups.set(group.id, group)
    }
  }
  return Array.from(groups.values()).sort((a, b) => a.name.localeCompare(b.name, 'zh-CN'))
}

function healthGuardAccountGroupCount(mapping: HealthGuardAccountMapping): number {
  return healthGuardAccountGroups(mapping).length
}

function healthGuardAccountGroupsTitle(mapping: HealthGuardAccountMapping): string {
  const groups = healthGuardAccountGroups(mapping)
  if (!groups.length) return '未绑定分组'
  const names = groups.map(group => group.name || `分组 #${group.id}`)
  return `绑定分组（${groups.length}）：${names.join('、')}`
}

const healthGuardModelCreatablePrefix = '使用模型'

function healthGuardModelSelectOptions(platform: string, currentModel = ''): SelectOption[] {
  const models = healthGuardModelOptionsByPlatform.value[platform] || []
  const options: SelectOption[] = [
    {
      value: '',
      label: healthGuardModelLoadingByPlatform.value[platform] ? '加载模型中…' : '未配置',
    },
    ...models.map(model => ({
      value: model.id,
      label: model.display_name || model.id,
    })),
  ]
  const normalizedCurrentModel = currentModel?.trim()
  if (normalizedCurrentModel && !models.some(model => model.id === normalizedCurrentModel)) {
    options.splice(1, 0, {
      value: normalizedCurrentModel,
      label: `${normalizedCurrentModel}（当前配置）`,
    })
  }
  return options
}

function healthGuardAccountIsSelected(accountID: number): boolean {
  return healthGuardAccountIDs.value.includes(accountID)
}

const healthGuardCheckedAccountIDs = ref<number[]>([])

function healthGuardAccountIsChecked(accountID: number): boolean {
  return healthGuardCheckedAccountIDs.value.includes(accountID)
}

function toggleHealthGuardAccountCheck(accountID: number) {
  healthGuardCheckedAccountIDs.value = healthGuardAccountIsChecked(accountID)
    ? healthGuardCheckedAccountIDs.value.filter(item => item !== accountID)
    : normalizePositiveAccountIDs([...healthGuardCheckedAccountIDs.value, accountID])
}

function healthGuardAccountOverrideModel(accountID: number): string {
  return normalizeStringMap(config.value.account_health_guard_account_models)[String(accountID)]?.trim() || ''
}

function healthGuardPlatformDefaultModel(platform: string): string {
  return normalizeStringMap(config.value.account_health_guard_platform_models)[platform]?.trim() || ''
}

async function loadHealthGuardModels() {
  const platforms = healthGuardPlatformSummaries.value.map(summary => summary.platform)
  healthGuardModelOptionsByPlatform.value = {}
  healthGuardModelLoadingByPlatform.value = Object.fromEntries(platforms.map(platform => [platform, true]))
  try {
    const config = await adminAPI.modelSquareConfig.get()
    const byPlatform: Record<string, { id: string; display_name?: string }[]> = {}
    for (const platformConfig of config.platforms || []) {
      byPlatform[platformConfig.platform] = (platformConfig.models || []).map(model => ({
        id: model.id,
        display_name: model.display_name,
      }))
    }
    healthGuardModelOptionsByPlatform.value = Object.fromEntries(
      platforms.map(platform => [platform, byPlatform[normalizeHealthGuardPlatform(platform)] || []])
    )
  } catch {
    healthGuardModelOptionsByPlatform.value = Object.fromEntries(platforms.map(platform => [platform, []]))
  } finally {
    healthGuardModelLoadingByPlatform.value = Object.fromEntries(platforms.map(platform => [platform, false]))
  }
}

function addHealthGuardAccount(id: number) {
  config.value.account_health_guard_account_ids = normalizePositiveAccountIDs([
    ...healthGuardAccountIDs.value,
    id,
  ])
}

function toggleHealthGuardAccount(id: number) {
  if (healthGuardAccountIDs.value.includes(id)) {
    removeHealthGuardAccount(id)
    return
  }
  addHealthGuardAccount(id)
}

function removeHealthGuardAccount(id: number) {
  config.value.account_health_guard_account_ids = healthGuardAccountIDs.value.filter(item => item !== id)
  const accountModels = { ...config.value.account_health_guard_account_models }
  delete accountModels[String(id)]
  config.value.account_health_guard_account_models = accountModels
  const accountIntervals = { ...(config.value.account_health_guard_account_intervals || {}) }
  delete accountIntervals[String(id)]
  config.value.account_health_guard_account_intervals = accountIntervals
  const schedulingChange = { ...(config.value.account_health_guard_account_scheduling_change || {}) }
  delete schedulingChange[String(id)]
  config.value.account_health_guard_account_scheduling_change = schedulingChange
  // 阈值覆盖与账号选择强绑定，取消选择后必须一并清掉，否则重新选中会带着旧覆盖值回来。
  config.value.account_health_guard_account_failure_thresholds = deleteHealthGuardAccountThreshold(
    config.value.account_health_guard_account_failure_thresholds, id)
  config.value.account_health_guard_account_slow_thresholds = deleteHealthGuardAccountThreshold(
    config.value.account_health_guard_account_slow_thresholds, id)
  config.value.account_health_guard_account_recovery_thresholds = deleteHealthGuardAccountThreshold(
    config.value.account_health_guard_account_recovery_thresholds, id)
}

function deleteHealthGuardAccountThreshold(value: unknown, accountID: number): Record<string, number> {
  const thresholds = { ...normalizeAccountHealthGuardAccountThresholds(value) }
  delete thresholds[String(accountID)]
  return thresholds
}

function healthGuardAccountThresholdValue(
  field: 'account_health_guard_account_failure_thresholds'
    | 'account_health_guard_account_slow_thresholds'
    | 'account_health_guard_account_recovery_thresholds',
  accountID: number
): number | undefined {
  const threshold = Number(config.value[field]?.[String(accountID)])
  return Number.isSafeInteger(threshold) && threshold > 0 ? threshold : undefined
}

function setHealthGuardAccountThreshold(
  field: 'account_health_guard_account_failure_thresholds'
    | 'account_health_guard_account_slow_thresholds'
    | 'account_health_guard_account_recovery_thresholds',
  accountID: number,
  value: string | number
) {
  const thresholds = { ...normalizeAccountHealthGuardAccountThresholds(config.value[field]) }
  const parsed = Math.floor(Number(value))
  // 留空或非法输入一律回落到全局阈值，与「检查间隔」字段的留空语义保持一致。
  if (Number.isSafeInteger(parsed) && parsed > 0) {
    thresholds[String(accountID)] = parsed
  } else {
    delete thresholds[String(accountID)]
  }
  config.value[field] = thresholds
}

function healthGuardAccountIntervalValue(accountID: number): number | undefined {
  const interval = Number(config.value.account_health_guard_account_intervals?.[String(accountID)])
  return Number.isSafeInteger(interval) && interval >= 60 ? interval : undefined
}

function setHealthGuardAccountInterval(accountID: number, value: string | number) {
  const intervals = { ...(config.value.account_health_guard_account_intervals || {}) }
  const parsed = Math.floor(Number(value))
  if (Number.isSafeInteger(parsed) && parsed >= 60) {
    intervals[String(accountID)] = parsed
  } else {
    delete intervals[String(accountID)]
  }
  config.value.account_health_guard_account_intervals = intervals
}

const healthGuardBatchIntervalSeconds = computed<number | null>(() => {
  const parsed = Math.floor(Number(healthGuardBatchIntervalInput.value))
  return Number.isSafeInteger(parsed) && parsed >= 60 ? parsed : null
})

const healthGuardBatchIntervalValid = computed(() => healthGuardBatchIntervalSeconds.value !== null)

const healthGuardBatchTargetRows = computed(() =>
  healthGuardWorkspaceAccounts.value.filter(
    mapping => mapping.available && healthGuardCheckedAccountIDs.value.includes(mapping.localAccountID)
  )
)

const healthGuardBatchTargetCount = computed(() => healthGuardBatchTargetRows.value.length)

function selectAllFilteredHealthGuardAccounts() {
  const ids = healthGuardWorkspaceAccounts.value
    .filter(mapping => mapping.available)
    .map(mapping => mapping.localAccountID)
  healthGuardCheckedAccountIDs.value = normalizePositiveAccountIDs([
    ...healthGuardCheckedAccountIDs.value,
    ...ids,
  ])
}

function deselectFilteredHealthGuardAccounts() {
  const filtered = new Set(healthGuardWorkspaceAccounts.value.map(mapping => mapping.localAccountID))
  healthGuardCheckedAccountIDs.value = healthGuardCheckedAccountIDs.value.filter(id => !filtered.has(id))
}

function applyHealthGuardBatchInterval() {
  const seconds = healthGuardBatchIntervalSeconds.value
  if (seconds === null) {
    appStore.showError('检查间隔必须是不小于 60 秒的整数')
    return
  }
  const intervals = { ...(config.value.account_health_guard_account_intervals || {}) }
  let count = 0
  for (const mapping of healthGuardBatchTargetRows.value) {
    intervals[String(mapping.localAccountID)] = seconds
    count += 1
  }
  config.value.account_health_guard_account_intervals = intervals
  appStore.showSuccess(`已为 ${count} 个账号设置检查间隔 ${seconds} 秒，保存任务后生效`)
}

function clearHealthGuardSelectedIntervals() {
  const intervals = { ...(config.value.account_health_guard_account_intervals || {}) }
  let count = 0
  for (const mapping of healthGuardBatchTargetRows.value) {
    if (intervals[String(mapping.localAccountID)] !== undefined) {
      delete intervals[String(mapping.localAccountID)]
      count += 1
    }
  }
  config.value.account_health_guard_account_intervals = intervals
  appStore.showSuccess(`已清除 ${count} 个账号的检查间隔`)
}

const healthGuardGlobalSchedulingChange = computed(
  () => config.value.account_health_guard_scheduling_change_enabled !== false
)

function healthGuardAccountSchedulingChangeValue(accountID: number): boolean {
  const override = config.value.account_health_guard_account_scheduling_change?.[String(accountID)]
  return typeof override === 'boolean' ? override : healthGuardGlobalSchedulingChange.value
}

function setHealthGuardAccountSchedulingChange(accountID: number, value: boolean) {
  const schedulingChange = { ...(config.value.account_health_guard_account_scheduling_change || {}) }
  if (value === healthGuardGlobalSchedulingChange.value) {
    delete schedulingChange[String(accountID)]
  } else {
    schedulingChange[String(accountID)] = value
  }
  config.value.account_health_guard_account_scheduling_change = schedulingChange
}

function healthGuardAccountSchedulingChangeTitle(accountID: number): string {
  const override = config.value.account_health_guard_account_scheduling_change?.[String(accountID)]
  const globalHint = healthGuardGlobalSchedulingChange.value ? '全局默认开启' : '全局默认关闭'
  const base = '关闭后仅检测账号健康，不会自动暂停或恢复调度'
  return typeof override === 'boolean'
    ? `${base}（账号级覆盖，${globalHint}）`
    : `${base}（跟随全局：${globalHint}）`
}

const healthGuardBatchSettingsVisible = ref(false)

const healthGuardBatchGuardOptions: SelectOption[] = [
  { value: HEALTH_GUARD_BATCH_GUARD_KEEP, label: '不修改' },
  { value: 'on', label: '纳入守护' },
  { value: 'off', label: '移出守护（清空账号级配置）' },
]

const healthGuardBatchSchedulingChangeOptions: SelectOption[] = [
  { value: HEALTH_GUARD_BATCH_SCHEDULING_KEEP, label: '不修改' },
  { value: 'global', label: '跟随全局（清除账号级覆盖）' },
  { value: 'on', label: '开启' },
  { value: 'off', label: '关闭' },
]

const healthGuardBatchModelInput = ref(HEALTH_GUARD_BATCH_MODEL_KEEP)

const healthGuardBatchModelOptions = computed<SelectOption[]>(() => {
  const platforms = new Set(healthGuardBatchTargetRows.value.map(mapping => mapping.platform))
  const models = new Map<string, string>()
  for (const platform of platforms) {
    for (const model of healthGuardModelOptionsByPlatform.value[platform] || []) {
      if (!models.has(model.id)) models.set(model.id, model.display_name || model.id)
    }
  }
  return [
    { value: HEALTH_GUARD_BATCH_MODEL_KEEP, label: '不修改' },
    { value: HEALTH_GUARD_BATCH_MODEL_CLEAR, label: '清除账号级覆盖（回落平台默认）' },
    ...Array.from(models, ([value, label]) => ({ value, label })),
  ]
})

const healthGuardBatchFailureThresholdInput = ref('')

const healthGuardBatchSlowThresholdInput = ref('')

const healthGuardBatchRecoveryThresholdInput = ref('')

function openHealthGuardBatchSettings() {
  // 每次打开都回到「全部不修改」：上一次的输入不该在下一次静默生效。
  healthGuardBatchGuardInput.value = HEALTH_GUARD_BATCH_GUARD_KEEP
  healthGuardBatchSchedulingChangeInput.value = HEALTH_GUARD_BATCH_SCHEDULING_KEEP
  healthGuardBatchModelInput.value = HEALTH_GUARD_BATCH_MODEL_KEEP
  healthGuardBatchFailureThresholdInput.value = ''
  healthGuardBatchSlowThresholdInput.value = ''
  healthGuardBatchRecoveryThresholdInput.value = ''
  healthGuardBatchSettingsVisible.value = true
}

function closeHealthGuardBatchSettings() {
  healthGuardBatchSettingsVisible.value = false
}

function healthGuardBatchThresholdValue(value: string): number | null {
  const parsed = Math.floor(Number(value))
  return Number.isSafeInteger(parsed) && parsed > 0 ? parsed : null
}

function healthGuardMultiplierRules(platform: string): SupplierAccountHealthGuardMultiplierInterval[] {
  let map = config.value.account_health_guard_platform_multiplier_intervals
  if (!map || typeof map !== 'object' || Array.isArray(map)) {
    map = {}
    config.value.account_health_guard_platform_multiplier_intervals = map
  }
  if (!Array.isArray(map[platform])) {
    map[platform] = []
  }
  return map[platform]
}

function addHealthGuardMultiplierRule(platform: string) {
  healthGuardMultiplierRules(platform).push({ min_multiplier: 0, max_multiplier: 0, interval_seconds: 3600 })
}

function removeHealthGuardMultiplierRule(platform: string, index: number) {
  healthGuardMultiplierRules(platform).splice(index, 1)
}

const healthGuardAccountPlatformFilter = ref<string[]>([])

const healthGuardAccountProviderFilter = ref('')

const healthGuardProviderFilterOptions = computed<SelectOption[]>(() => {
  const providers = new Map<number, { name: string; accountIDs: Set<number> }>()
  for (const mapping of healthGuardPlatformScopedMappings.value) {
    for (const source of mapping.sources) {
      const providerID = Number(source.provider_id)
      if (!Number.isSafeInteger(providerID) || providerID <= 0) continue
      const current = providers.get(providerID)
      if (current) {
        current.accountIDs.add(mapping.localAccountID)
        if (!current.name && source.provider_name) current.name = source.provider_name
        continue
      }
      providers.set(providerID, {
        name: source.provider_name || `供应商 ${providerID}`,
        accountIDs: new Set([mapping.localAccountID]),
      })
    }
  }

  return [
    { value: '', label: '全部供应商' },
    ...Array.from(providers.entries())
      .sort(([, a], [, b]) => a.name.localeCompare(b.name, 'zh-CN'))
      .map(([providerID, provider]) => ({
        value: String(providerID),
        label: `${provider.name}（${provider.accountIDs.size}）`,
      })),
  ]
})

const healthGuardAccountPlatformFacets = computed(() =>
  buildPlatformFacets(healthGuardProviderScopedMappings.value)
)

const healthGuardAccountIDs = computed(() =>
  normalizePositiveAccountIDs(config.value.account_health_guard_account_ids)
)

const healthGuardBatchGuardInput = ref(HEALTH_GUARD_BATCH_GUARD_KEEP)

const healthGuardBatchSchedulingChangeInput = ref(HEALTH_GUARD_BATCH_SCHEDULING_KEEP)

function applyHealthGuardBatchSettings() {
  const targets = healthGuardBatchTargetRows.value
  if (!targets.length) {
    appStore.showError('当前筛选结果里没有已勾选的可用账号')
    return
  }

  const thresholdFields: Array<{ input: string; label: string; field: HealthGuardAccountThresholdField }> = [
    { input: healthGuardBatchFailureThresholdInput.value, label: '连续失败暂停阈值', field: 'account_health_guard_account_failure_thresholds' },
    { input: healthGuardBatchSlowThresholdInput.value, label: '连续慢响应暂停阈值', field: 'account_health_guard_account_slow_thresholds' },
    { input: healthGuardBatchRecoveryThresholdInput.value, label: '连续健康恢复阈值', field: 'account_health_guard_account_recovery_thresholds' },
  ]
  // 先整体校验再落库：任何一项非法就整批不动，不留「改了一半」的中间状态。
  for (const item of thresholdFields) {
    if (item.input.trim() && healthGuardBatchThresholdValue(item.input) === null) {
      appStore.showError(`${item.label}必须是正整数`)
      return
    }
  }

  const changed: string[] = []

  // 「参与守护」先处理，并据此收敛本次作用对象：「移出守护」会清空该账号的账号级配置，
  // 若之后再把阈值/模型写进去就会被反手清掉，出现「提示说设了、实际没设」。
  // 所以移出后不再处理其余项 —— 弹窗里已写明这一点，不是静默忽略。
  const guardChoice = healthGuardBatchGuardInput.value
  let effectiveTargets = targets
  if (guardChoice === 'off') {
    for (const mapping of targets) removeHealthGuardAccount(mapping.localAccountID)
    effectiveTargets = []
    changed.push('移出守护')
  } else if (guardChoice === 'on') {
    config.value.account_health_guard_account_ids = normalizePositiveAccountIDs([
      ...healthGuardAccountIDs.value,
      ...targets.map(mapping => mapping.localAccountID),
    ])
    changed.push('纳入守护')
  }

  if (effectiveTargets.length) {
    const schedulingChoice = healthGuardBatchSchedulingChangeInput.value
    if (schedulingChoice !== HEALTH_GUARD_BATCH_SCHEDULING_KEEP) {
      const schedulingChange = { ...(config.value.account_health_guard_account_scheduling_change || {}) }
      for (const mapping of effectiveTargets) {
        const accountID = String(mapping.localAccountID)
        if (schedulingChoice === 'global') {
          delete schedulingChange[accountID]
        } else {
          schedulingChange[accountID] = schedulingChoice === 'on'
        }
      }
      config.value.account_health_guard_account_scheduling_change = schedulingChange
      changed.push('修改调度')
    }

    const modelChoice = healthGuardBatchModelInput.value
    if (modelChoice !== HEALTH_GUARD_BATCH_MODEL_KEEP) {
      const accountModels = { ...(config.value.account_health_guard_account_models || {}) }
      for (const mapping of effectiveTargets) {
        const accountID = String(mapping.localAccountID)
        if (modelChoice === HEALTH_GUARD_BATCH_MODEL_CLEAR) {
          delete accountModels[accountID]
        } else {
          accountModels[accountID] = modelChoice
        }
      }
      config.value.account_health_guard_account_models = accountModels
      changed.push('测试模型')
    }

    for (const item of thresholdFields) {
      const threshold = healthGuardBatchThresholdValue(item.input)
      if (threshold === null) continue
      const thresholds = { ...normalizeAccountHealthGuardAccountThresholds(config.value[item.field]) }
      for (const mapping of effectiveTargets) thresholds[String(mapping.localAccountID)] = threshold
      config.value[item.field] = thresholds
      changed.push(item.label)
    }
  }

  if (!changed.length) {
    appStore.showError('请先填写至少一项要批量设置的内容')
    return
  }
  appStore.showSuccess(`已为 ${targets.length} 个账号设置${changed.join('、')}，保存任务后生效`)
  closeHealthGuardBatchSettings()
}

// 联动必须把「已经选不到」的那一项收回去，否则会留下一个界面上根本不存在、又取消不掉的选中项：
// 平台标签只在候选里有对应 facet 时才渲染，被挤掉的标签会连点击取消的机会都没有，
// 结果是列表一直空着而用户找不到原因。
//
// 收回的是**被挤掉的那个**，而不是刚改的那个：用户点平台标签就是想看该平台，
// 若把刚点的标签回退掉，表现就是「点了没反应」。所以两个 watch 都挂在「值」上
// （谁变了就检查另一个），语义是「后改的生效」。
// 不会成环：收回只会让对方的候选变多（空选 / 「全部供应商」一定在候选里），不会把对方也挤掉。
watch(healthGuardAccountPlatformFilter, () => {
  if (!healthGuardProviderFilterOptions.value.some(
    option => option.value === healthGuardAccountProviderFilter.value
  )) {
    healthGuardAccountProviderFilter.value = ''
  }
})

watch(healthGuardAccountProviderFilter, () => {
  const available = new Set(healthGuardAccountPlatformFacets.value.map(facet => facet.platform))
  const kept = healthGuardAccountPlatformFilter.value.filter(platform => available.has(platform))
  if (kept.length !== healthGuardAccountPlatformFilter.value.length) {
    healthGuardAccountPlatformFilter.value = kept
  }
})

async function onDialogOpen() {
  healthGuardAccountPlatformFilter.value = []
  healthGuardAccountProviderFilter.value = ''
  healthGuardAccountSearch.value = ''
  healthGuardSelectedOnly.value = false
  healthGuardUnselectedOnly.value = false
  healthGuardProviderClosedOnly.value = false
  healthGuardProviderEnabledOnly.value = false
  healthGuardBatchIntervalInput.value = ''
  // 勾选是「本次批量操作的选择」，每次打开都清空：否则上一次的勾选会静默参与下一次批量设置。
  healthGuardCheckedAccountIDs.value = []
  try {
    await loadHealthGuardModels()
  } catch (err) {
    appStore.showError(extractApiErrorMessage(err, '加载健康守护账号失败'))
  }
}

function closeHealthGuardAccounts() {
  emit('close')
}

// 打开时重置本地筛选并拉取模型清单；账号数据由父组件在打开前准备好。
watch(
  () => props.show,
  visible => {
    if (visible) void onDialogOpen()
  }
)

</script>

<style scoped>

/* 倍率区间弹窗是内容型的，不需要「近全屏」那套 height: calc(100dvh - …)，
   给一个上限让它在内容少时矮、内容多时顶到 70vh 由容器内部滚动。 */
:global(.modal-content:has(.sp-multiplier-interval-dialog)) {
  max-height: 70vh;
}


:global(.modal-content:has(.sp-health-guard-account-dialog) .modal-body),
:global(.modal-content:has(.sp-rate-guard-group-dialog) .modal-body) {
  overflow: hidden;
}


/* 批量设置弹窗是又一个独立 Teleport 出来的 modal-content（与「配置健康守护账号」是兄弟节点），
   同样够不到页面根节点上的 --sp-* 变量，所以单独给它一份变量与布局。
   选择器独立成组：上面那几组已被源码断言逐字钉住，往里面加选择器会直接打爆相邻用例。 */
:global(.modal-content:has(.sp-health-guard-batch-dialog)) {
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
  overflow: hidden;
  border-color: #cbd7e5;
  background: var(--sp-panel);
  color: var(--sp-text);
}


:global(.dark .modal-content:has(.sp-health-guard-batch-dialog)) {
  --sp-panel: #172033;
  --sp-panel-2: #1d293d;
  --sp-panel-3: #243249;
  --sp-line: #35445c;
  --sp-soft: #2c3a51;
  --sp-text: #edf3fb;
  --sp-muted: #a8b6ca;
  border-color: #3b4b64;
}


:global(.modal-content:has(.sp-health-guard-batch-dialog) .modal-header) {
  border-bottom-color: var(--sp-line);
  background: var(--sp-panel);
}


:global(.modal-content:has(.sp-health-guard-batch-dialog) .modal-title) {
  color: var(--sp-text);
}


:global(.modal-content:has(.sp-health-guard-batch-dialog) .modal-body) {
  display: flex;
  min-height: 0;
  flex: 1 1 auto;
  flex-direction: column;
  overflow: hidden;
  background: var(--sp-panel);
}


:global(.modal-content:has(.sp-health-guard-batch-dialog) .modal-footer) {
  border-top-color: var(--sp-line);
  background: var(--sp-panel);
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


.sp-health-guard-account-dialog {
  display: flex;
  flex: 1 1 auto;
  flex-direction: column;
  gap: 12px;
  min-height: 0;
  height: 100%;
  max-height: none;
  overflow: hidden;
}


.sp-health-guard-platform-models,
.sp-health-guard-account-workspace {
  min-width: 0;
  overflow: hidden;
  border: 1px solid var(--sp-line);
  border-radius: 12px;
  background: var(--sp-panel);
}


.sp-health-guard-platform-models {
  flex: 0 0 auto;
  padding: 10px 14px 12px;
  background: color-mix(in srgb, var(--sp-blue) 4%, var(--sp-panel));
}


/* 倍率区间原先按平台铺在主弹窗里，7 个平台会把账号工作区挤没（弹窗高度受 100dvh 约束）。
   现在主弹窗只留「摘要 + 入口」一行，区间配置搬进 .sp-multiplier-interval-dialog 二级弹窗，
   这里因此不再需要可收缩/内部滚动那套（它们正是「grid 被压成 0 高度」的根因）。 */
.sp-health-guard-multiplier-entry {
  display: flex;
  align-items: center;
  justify-content: space-between;
  flex-wrap: wrap;
  gap: 8px 12px;
  padding-top: 10px;
}


.sp-health-guard-multiplier-entry > span {
  color: var(--sp-muted);
  font-size: 12px;
}


.sp-health-guard-multiplier-entry-button {
  flex: 0 0 auto;
}


.sp-health-guard-account-workspace {
  display: flex;
  flex: 1 1 auto;
  flex-direction: column;
  min-height: 0;
}


.sp-health-guard-dialog-section-head {
  display: flex;
  align-items: baseline;
  flex-wrap: wrap;
  justify-content: space-between;
  gap: 4px 12px;
  padding-bottom: 10px;
  border-bottom: 1px solid var(--sp-line);
}


.sp-health-guard-dialog-section-head strong {
  display: inline-flex;
  align-items: center;
  gap: 7px;
  color: color-mix(in srgb, var(--sp-blue) 78%, var(--sp-text));
}


.sp-health-guard-dialog-section-head strong::before {
  content: '';
  width: 4px;
  height: 14px;
  border-radius: 2px;
  background: var(--sp-blue);
}


.sp-health-guard-dialog-section-head > span {
  color: color-mix(in srgb, var(--sp-blue) 52%, var(--sp-muted));
}


.sp-health-guard-platform-model-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(min(100%, 300px), 1fr));
  align-content: start;
  gap: 8px 12px;
  margin-top: 10px;
}


.sp-health-guard-platform-model-grid article {
  display: grid;
  grid-template-columns: minmax(88px, 0.55fr) minmax(150px, 1.45fr);
  align-items: center;
  gap: 8px;
  min-width: 0;
  border: 1px solid color-mix(in srgb, var(--sp-blue) 12%, var(--sp-line));
  border-radius: 10px;
  padding: 8px 10px;
  background: var(--sp-panel);
  transition: border-color 160ms ease, box-shadow 160ms ease, background-color 160ms ease;
}


.sp-health-guard-platform-model-grid article:hover {
  border-color: color-mix(in srgb, var(--sp-blue) 30%, var(--sp-line));
  box-shadow: 0 2px 10px color-mix(in srgb, var(--sp-blue) 9%, transparent);
}


.sp-health-guard-platform-model-grid article > div {
  display: grid;
  min-width: 0;
  gap: 2px;
}


.sp-health-guard-platform-model-grid article > div strong {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  color: inherit;
}


.sp-health-guard-platform-model-grid article > div span {
  color: color-mix(in srgb, currentColor 58%, var(--sp-muted));
  font-variant-numeric: tabular-nums;
}


/* 倍率区间弹窗容器：自己滚，不靠 modal-body（body 是 overflow: hidden，
   内部不滚就会被 overflow 吞掉，正是之前主弹窗里 grid 消失的同一个坑）。 */
.sp-multiplier-interval-dialog {
  flex: 1 1 auto;
  min-height: 0;
  overflow-y: auto;
  padding: 12px 14px;
}


/* 列宽必须保持 460px：卡片内是「下限 / 上限 / 间隔 / 删除」四列 grid，
   列宽一收窄，表头「倍率上限（不含，留空=无上界）」就会折成两行、「新增区间」按钮被压扁。
   弹窗本身用 extra-wide 而不是 wide，就是为了在 460px 列宽下仍能排两列。 */
.sp-multiplier-interval-dialog .sp-health-guard-multiplier-grid {
  grid-template-columns: repeat(auto-fill, minmax(min(100%, 460px), 1fr));
  margin-top: 0;
}


.sp-multiplier-interval-hint {
  color: var(--sp-muted);
  font-size: 12px;
}


/* 批量设置弹窗：逐项一行「左侧说明 + 右侧控件」。
   容器自己滚，不靠 modal-body（body 是 overflow: hidden，内部不滚会被吞掉）。 */
.sp-health-guard-batch-dialog {
  flex: 1 1 auto;
  min-height: 0;
  overflow-y: auto;
  padding: 12px 14px;
}


.sp-health-guard-batch-rows {
  display: grid;
  gap: 10px;
}


.sp-health-guard-batch-row {
  display: grid;
  grid-template-columns: minmax(0, 1fr) minmax(220px, 0.8fr);
  align-items: center;
  gap: 8px 16px;
  border: 1px solid color-mix(in srgb, var(--sp-blue) 12%, var(--sp-line));
  border-radius: 10px;
  padding: 10px 12px;
  background: color-mix(in srgb, var(--sp-soft) 20%, transparent);
}


.sp-health-guard-batch-row > div {
  display: grid;
  gap: 3px;
  min-width: 0;
}


.sp-health-guard-batch-row > div span {
  color: var(--sp-muted);
  font-size: 12px;
}


.sp-health-guard-batch-hint {
  color: var(--sp-muted);
  font-size: 12px;
}


.sp-health-guard-multiplier-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(min(100%, 460px), 1fr));
  align-content: start;
  gap: 10px 14px;
  margin-top: 12px;
}


.sp-health-guard-multiplier-grid article {
  align-self: start;
  border: 1px solid color-mix(in srgb, var(--sp-blue) 12%, var(--sp-line));
  border-radius: 10px;
  padding: 10px 12px;
  background: var(--sp-panel);
}


.sp-health-guard-multiplier-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  margin-bottom: 8px;
}


.sp-health-guard-multiplier-rows {
  display: grid;
  gap: 6px;
}


.sp-health-guard-multiplier-row {
  display: grid;
  grid-template-columns: minmax(0, 1fr) minmax(0, 1.3fr) minmax(0, 1fr) 40px;
  align-items: center;
  gap: 8px;
}


.sp-health-guard-multiplier-row-head span {
  color: var(--sp-muted);
  font-size: 12px;
}


.sp-health-guard-selection-summary {
  display: grid;
  flex: 0 0 auto;
  grid-template-columns: repeat(5, minmax(0, 1fr));
  border-bottom: 1px solid var(--sp-line);
  background: color-mix(in srgb, var(--sp-soft) 22%, transparent);
}


.sp-health-guard-selection-summary article {
  --sp-summary-accent: var(--sp-muted);

  position: relative;
  display: flex;
  align-items: baseline;
  justify-content: space-between;
  min-width: 0;
  gap: 10px;
  border-left: 1px solid var(--sp-line);
  padding: 12px 14px 11px;
}


.sp-health-guard-selection-summary article:first-child {
  border-left: 0;
}


.sp-health-guard-selection-summary article:nth-child(1) {
  --sp-summary-accent: var(--sp-blue);
}


.sp-health-guard-selection-summary article:nth-child(2) {
  --sp-summary-accent: var(--sp-violet);
}


.sp-health-guard-selection-summary article:nth-child(3) {
  --sp-summary-accent: var(--sp-green);
}


.sp-health-guard-selection-summary article:nth-child(4) {
  --sp-summary-accent: var(--sp-cyan);
}


.sp-health-guard-selection-summary article:nth-child(5) {
  --sp-summary-accent: var(--sp-amber);
}


.sp-health-guard-selection-summary article::before {
  content: '';
  position: absolute;
  top: 0;
  right: 0;
  left: 0;
  height: 3px;
  background: var(--sp-summary-accent);
  opacity: 0.85;
}


.sp-health-guard-selection-summary article span {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  color: color-mix(in srgb, var(--sp-summary-accent) 64%, var(--sp-muted));
  font-size: 12px;
}


.sp-health-guard-selection-summary article span::before {
  content: '';
  width: 6px;
  height: 6px;
  border-radius: 999px;
  background: var(--sp-summary-accent);
  opacity: 0.9;
}


.sp-health-guard-selection-summary article strong {
  color: var(--sp-summary-accent);
  font-size: 18px;
  font-weight: 750;
  font-variant-numeric: tabular-nums;
  line-height: 1;
}


.sp-health-guard-selection-summary article.warning {
  background: color-mix(in srgb, var(--sp-amber) 10%, transparent);
}


.sp-health-guard-account-toolbar {
  display: flex;
  flex: 0 0 auto;
  flex-wrap: wrap;
  align-items: center;
  justify-content: space-between;
  gap: 8px 12px;
  padding: 10px 14px;
}


/* 三列：供应商下拉 / 搜索框 / 快捷过滤（仅看已选 + 仅看未开启）。平台筛选已改成独占一行的标签组、
   移出这个 grid（flex-basis 在 grid 里不生效，塞进来只会把列挤变形），所以列数从 4 减到 3。
   两个快捷过滤共用第 3 列的一个 flex 容器，不再各占一列 —— 否则列数会重新变回 4。
   ⚠️ 被移除的是**第 1 个**子节点 ⇒ 移动端跨列阈值 nth-child 的落点会整体前移一位，
   原来「搜索框 + 仅看已选通栏」会变成「只有仅看已选通栏」。移动端规则已按单列重写。 */
.sp-health-guard-account-filters {
  display: grid;
  flex: 1 1 640px;
  grid-template-columns: minmax(160px, 0.42fr) minmax(220px, 1fr) auto;
  gap: 10px;
  min-width: 0;
}


/* 两个快捷过滤按钮的容器：桌面端贴合内容宽度占住 grid 第 3 列，
   移动端 grid 变单列后由 flex-wrap 自己换行。 */
.sp-health-guard-quick-filters {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 8px;
  min-width: 0;
}


.sp-health-guard-selected-toggle {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  gap: 8px;
  min-height: 38px;
  border: 1px solid var(--sp-line);
  border-radius: 9px;
  padding: 0 12px;
  color: var(--sp-muted);
  background: var(--sp-panel);
  cursor: pointer;
  font: inherit;
  font-size: 12px;
  font-weight: 600;
  white-space: nowrap;
  transition: border-color 160ms ease, background-color 160ms ease, color 160ms ease, box-shadow 160ms ease, transform 120ms ease;
}


.sp-health-guard-selected-toggle:hover,
.sp-health-guard-selected-toggle:focus-visible {
  border-color: color-mix(in srgb, var(--sp-blue) 45%, var(--sp-line));
  color: var(--sp-blue);
}


.sp-health-guard-selected-toggle:active {
  transform: translateY(1px);
}


.sp-health-guard-selected-toggle:focus-visible {
  outline: 2px solid color-mix(in srgb, var(--sp-blue) 28%, transparent);
  outline-offset: 2px;
}


.sp-health-guard-selected-toggle.active {
  border-color: color-mix(in srgb, var(--sp-blue) 52%, var(--sp-line));
  color: var(--sp-blue);
  background: color-mix(in srgb, var(--sp-blue) 10%, var(--sp-panel));
  box-shadow: 0 1px 0 color-mix(in srgb, var(--sp-blue) 20%, transparent);
}


.sp-health-guard-selected-toggle-mark {
  width: 8px;
  height: 8px;
  border-radius: 999px;
  background: var(--sp-muted);
  transition: background-color 160ms ease, box-shadow 160ms ease;
}


.sp-health-guard-selected-toggle.active .sp-health-guard-selected-toggle-mark {
  background: var(--sp-blue);
  box-shadow: 0 0 0 3px color-mix(in srgb, var(--sp-blue) 18%, transparent);
}


.sp-health-guard-selected-toggle strong {
  min-width: 20px;
  border-radius: 999px;
  padding: 1px 7px;
  color: inherit;
  background: color-mix(in srgb, currentColor 10%, transparent);
  font-size: 11px;
  font-variant-numeric: tabular-nums;
  font-weight: 700;
}


.sp-health-guard-filter-result {
  display: inline-flex;
  align-items: center;
  flex: 0 0 auto;
  min-height: 24px;
  border-radius: 999px;
  padding: 0 9px;
  color: var(--sp-muted);
  background: color-mix(in srgb, var(--sp-soft) 72%, transparent);
  font-size: 11px;
  font-variant-numeric: tabular-nums;
  white-space: nowrap;
}


.sp-health-guard-filter-result strong {
  color: var(--sp-blue);
  font-weight: 750;
  font-variant-numeric: tabular-nums;
}


.sp-health-guard-batch-actions {
  display: flex;
  flex: 1 1 100%;
  flex-wrap: wrap;
  align-items: center;
  gap: 8px;
}


.sp-health-guard-batch-divider {
  width: 1px;
  align-self: stretch;
  min-height: 24px;
  background: var(--sp-line);
}


.sp-health-guard-batch-interval-input {
  flex: 0 0 auto;
  width: 132px;
}


.sp-health-guard-account-list {
  flex: 1 1 auto;
  min-height: 0;
  max-height: none;
  overflow: auto;
  overscroll-behavior: contain;
  -webkit-overflow-scrolling: touch;
  border-top: 1px solid var(--sp-line);
  background: color-mix(in srgb, var(--sp-soft) 12%, var(--sp-panel));
  scrollbar-width: thin;
  scrollbar-color: color-mix(in srgb, var(--sp-line) 82%, transparent) transparent;
}


.sp-health-guard-account-list::-webkit-scrollbar {
  width: 8px;
}


.sp-health-guard-account-list::-webkit-scrollbar-thumb {
  border: 2px solid transparent;
  border-radius: 999px;
  background: color-mix(in srgb, var(--sp-line) 85%, transparent);
  background-clip: padding-box;
}


.sp-health-guard-account-row {
  position: relative;
  display: grid;
  grid-template-columns: auto minmax(0, 1fr) auto minmax(140px, 0.35fr) minmax(320px, 0.8fr);
  align-items: center;
  min-width: 0;
  gap: 8px 12px;
  border-bottom: 1px solid var(--sp-line);
  padding: 8px 14px;
  background: var(--sp-panel);
  transition: background-color 160ms ease, box-shadow 160ms ease;
}


.sp-health-guard-account-row:last-child {
  border-bottom: 0;
}


.sp-health-guard-account-row:hover {
  background: color-mix(in srgb, var(--sp-blue) 4%, var(--sp-panel));
}


.sp-health-guard-account-row.selected {
  background: color-mix(in srgb, var(--sp-blue) 8%, var(--sp-panel));
  box-shadow: inset 3px 0 0 var(--sp-blue);
}


.sp-health-guard-account-row.selected:hover {
  background: color-mix(in srgb, var(--sp-blue) 11%, var(--sp-panel));
}


.sp-health-guard-account-row.missing-model {
  background: color-mix(in srgb, var(--sp-amber) 6%, var(--sp-panel));
  box-shadow: inset 3px 0 0 var(--sp-amber);
}


/* 供应商已关闭：整行弱化成中性灰。
   刻意不跟 .unavailable 共用琥珀警示色 —— 账号本身没坏（开关、模型、间隔都还能配），
   只是上游供应商被停用了，用警示色会被读成「这个账号出错了」。
   信息列降透明度表达「这条已经不活跃」；勾选框与开关保持原样，
   整行一起变淡会被当成禁用，用户就不敢再点了。 */
.sp-health-guard-account-row.provider-closed {
  background: color-mix(in srgb, var(--sp-muted) 7%, var(--sp-panel));
  box-shadow: inset 3px 0 0 color-mix(in srgb, var(--sp-muted) 55%, var(--sp-line));
}


.sp-health-guard-account-row.provider-closed .sp-health-guard-account-choice-copy,
.sp-health-guard-account-row.provider-closed .sp-health-guard-account-group-summary {
  opacity: 0.55;
}


.sp-health-guard-account-row.unavailable {
  background: color-mix(in srgb, var(--sp-amber) 8%, var(--sp-panel));
  box-shadow: inset 3px 0 0 color-mix(in srgb, var(--sp-amber) 72%, var(--sp-line));
}


.sp-health-guard-account-row:not(:has(.sp-health-guard-account-model-editor)) {
  grid-template-columns: auto minmax(0, 1fr) auto minmax(140px, 0.35fr);
}


/* 多选列：只服务于「批量设置 / 批量间隔」的选择，与右侧「参与守护」开关（业务配置）无关。
   和开关列一样是固定宽度列（auto），不参与伸缩。 */
.sp-health-guard-account-check {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  cursor: pointer;
}


.sp-health-guard-account-check input {
  width: 16px;
  height: 16px;
  margin: 0;
  accent-color: var(--sp-blue);
  cursor: pointer;
}


/* 勾选列与开关列都是固定宽度列，账号信息才是唯一可伸缩的列：
   开关用 fr 会随窗口变宽、把账号名挤成省略号。 */
.sp-health-guard-account-choice {
  min-width: 0;
  cursor: help;
}


.sp-health-guard-account-choice-copy {
  display: flex;
  align-items: center;
  min-width: 0;
  gap: 8px;
  color: var(--sp-muted);
  font-size: 12px;
}


.sp-health-guard-account-choice-copy strong {
  flex: 0 1 auto;
  min-width: 0;
  overflow: hidden;
  font-size: 13px;
  font-weight: 700;
  line-height: 1.25;
  text-overflow: ellipsis;
  white-space: nowrap;
}


.sp-health-guard-account-id {
  flex: 0 0 auto;
  color: color-mix(in srgb, var(--sp-blue) 52%, var(--sp-muted));
  font-size: 11px;
  font-variant-numeric: tabular-nums;
  white-space: nowrap;
}


.sp-health-guard-account-source {
  flex: 1 1 auto;
  min-width: 48px;
  overflow: hidden;
  color: var(--sp-muted);
  font-size: 11px;
  line-height: 1.25;
  text-overflow: ellipsis;
  white-space: nowrap;
}


.sp-health-guard-account-source.unavailable {
  color: var(--sp-amber);
  font-weight: 650;
}


.sp-health-guard-account-platform {
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


.sp-health-guard-account-group-summary {
  display: flex;
  min-width: 0;
  flex-direction: column;
  align-items: flex-start;
  justify-content: center;
  gap: 2px;
  cursor: help;
}


.sp-health-guard-account-group-summary > span {
  color: var(--sp-muted);
  font-size: 10px;
  line-height: 1.2;
}


.sp-health-guard-account-group-summary > strong {
  overflow: hidden;
  max-width: 100%;
  color: var(--sp-blue);
  font-size: 12px;
  font-variant-numeric: tabular-nums;
  line-height: 1.25;
  text-overflow: ellipsis;
  white-space: nowrap;
}


.sp-health-guard-model-status {
  display: inline-flex;
  align-items: center;
  flex: 0 0 auto;
  gap: 4px;
  border: 1px solid color-mix(in srgb, var(--sp-blue) 26%, var(--sp-line));
  border-radius: 999px;
  padding: 1px 7px;
  color: var(--sp-blue);
  background: color-mix(in srgb, var(--sp-blue) 7%, var(--sp-panel));
  font-size: 10px;
  font-weight: 700;
  line-height: 1.4;
  white-space: nowrap;
}


.sp-health-guard-model-status.override {
  color: var(--sp-green);
  border-color: color-mix(in srgb, var(--sp-green) 30%, var(--sp-line));
  background: color-mix(in srgb, var(--sp-green) 8%, var(--sp-panel));
}


.sp-health-guard-model-status.interval {
  color: var(--sp-cyan);
  border-color: color-mix(in srgb, var(--sp-cyan) 32%, var(--sp-line));
  background: color-mix(in srgb, var(--sp-cyan) 8%, var(--sp-panel));
}


.sp-health-guard-model-status.missing,
.sp-health-guard-model-status.unavailable {
  color: var(--sp-amber);
  border-color: color-mix(in srgb, var(--sp-amber) 34%, var(--sp-line));
  background: color-mix(in srgb, var(--sp-amber) 9%, var(--sp-panel));
}


.sp-health-guard-account-model-editor {
  display: grid;
  grid-template-columns: minmax(0, 1fr) minmax(112px, 0.45fr) auto;
  min-width: 0;
  align-items: center;
  gap: 8px;
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


.sp-health-guard-account-model-editor :deep(.select-trigger) {
  min-height: 34px;
  border-radius: 8px;
  padding: 5px 10px;
  font-size: 12px;
  line-height: 1.25;
}


/* 抽组件时这一层的 @media 包装丢了 ⇒ 手机端规则变成「所有宽度都生效」，
   桌面下工具栏/筛选区被压成单列，就是「挤变形」的直接原因。下面按原页面原文恢复各自的媒体条件。 */
@media (prefers-reduced-motion: reduce) {
  .sp-health-guard-account-row,
  .sp-health-guard-selected-toggle,
  .sp-health-guard-account-card .sp-button.sp-health-guard-config-button,
  .sp-health-guard-platform-model-grid article {
    transition: none;
  }
}

@media (max-width: 760px) {
  .sp-health-guard-account-card,
  .sp-health-guard-account-toolbar {
    align-items: stretch;
    flex-direction: column;
  }


  .sp-health-guard-account-dialog {
    /* 高度由 modal-content 约束，避免独立 max-height 把账号列表裁切掉 */
    max-height: none;
    min-height: 0;
  }


  .sp-health-guard-platform-models {
    flex: 0 1 auto;
    max-height: min(24vh, 180px);
    min-height: 0;
    overflow: auto;
    overscroll-behavior: contain;
    -webkit-overflow-scrolling: touch;
  }


  .sp-health-guard-account-workspace {
    flex: 1 1 auto;
    min-height: 0;
    overflow: hidden;
  }


  .sp-health-guard-platform-model-grid,
  .sp-health-guard-platform-model-grid article,
  .sp-health-guard-account-row {
    grid-template-columns: 1fr;
  }


  /* 健康守护账号筛选：手机端纵向堆叠，三个控件各占一行通栏（第 3 行是快捷过滤容器）。
     这里原先是「平台/供应商 2 列，搜索与快捷过滤通栏」—— 平台下拉改成
     独占一行的标签组之后，剩下的下拉只剩一个，2 列配对的前提不再成立；
     若只把跨列阈值从 n+3 改成 n+2，会剩下「下拉占半列、右边空一格」的残缺行。 */
  .sp-health-guard-account-filters {
    /* 手机端工具栏改为纵向布局后，桌面端 flex: 1 1 640px 的 640px 会变成高度基准，
       导致筛选区被撑到约 640px 高、账号列表被挤出可视区域；这里改为按内容自适应 */
    flex: 1 1 auto;
    grid-template-columns: 1fr;
  }


  .sp-health-guard-selection-summary {
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }


  .sp-health-guard-selection-summary article {
    padding: 8px 10px 7px;
  }


  .sp-health-guard-selection-summary article strong {
    font-size: 15px;
  }


  .sp-health-guard-selection-summary article:nth-child(odd) {
    border-left: 0;
  }


  .sp-health-guard-selection-summary article:nth-child(n + 3) {
    border-top: 1px solid var(--sp-line);
  }


  .sp-health-guard-account-toolbar {
    gap: 8px;
    padding: 10px 12px;
  }


  .sp-health-guard-account-list {
    /* 优先占剩余空间，可收缩；列表内部滚动，避免整页被裁切 */
    flex: 1 1 40vh;
    min-height: 0;
    overflow: auto;
    overscroll-behavior: contain;
    -webkit-overflow-scrolling: touch;
  }


  .sp-health-guard-account-model-editor {
    grid-template-columns: minmax(0, 1fr);
    min-width: 0;
  }


  /* 批量设置：手机端「说明 + 控件」改纵向堆叠，控件通栏 */
  .sp-health-guard-batch-row {
    grid-template-columns: 1fr;
  }


  .sp-health-guard-filter-result {
    align-self: flex-start;
  }


  .sp-health-guard-account-card .sp-button {
    width: 100%;
  }
}

:global(.modal-content:has(.sp-health-guard-account-dialog)),
:global(.modal-content:has(.sp-multiplier-interval-dialog)),
:global(.modal-content:has(.sp-run-detail)){
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
:global(.dark .modal-content:has(.sp-run-detail)){
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
:global(.modal-content:has(.sp-run-detail) .modal-header){
  border-bottom-color: var(--sp-line);
  background: var(--sp-panel);
}

:global(.modal-content:has(.sp-health-guard-account-dialog) .modal-title),
:global(.modal-content:has(.sp-run-detail) .modal-title){
  color: var(--sp-text);
}

:global(.modal-content:has(.sp-health-guard-account-dialog)),
:global(.modal-content:has(.sp-run-detail)){
  max-height: min(95vh, calc(100dvh - 16px));
}

:global(.modal-content:has(.sp-health-guard-account-dialog) .modal-body),
:global(.modal-content:has(.sp-multiplier-interval-dialog) .modal-body),
:global(.modal-content:has(.sp-run-detail) .modal-body){
  display: flex;
  min-height: 0;
  flex: 1 1 auto;
  flex-direction: column;
  overflow: hidden;
  background: var(--sp-panel);
}

/* 任务编辑弹窗、账号配置弹窗、分组配置弹窗：让内容区吃满可用高度，列表在剩余空间内滚动。
   高度同样顶到 overlay 留白之内，与宽度一起构成「近全屏」。
   这条必须排在上面那条 max-height: min(95vh, …) 之后 —— 两者选择器权重相同，
   靠源码顺序取胜。 */
:global(.modal-content:has(.sp-health-guard-account-dialog)){
  height: calc(100dvh - 1rem);
  max-height: calc(100dvh - 1rem);
}

/* >=768px：横向铺满（只留 overlay 自带的四周外边距）。
   原页面把「健康守护账号配置」与「任务编辑」写在同一个选择器组里；后者留在自动化页，
   所以这里只带上前者 —— 否则账号页（不加载自动化页的样式）会退回 BaseDialog 的
   full 档 1280px 上限，比原来窄，内容跟着被挤。 */
@media (min-width: 768px) {
  :global(.modal-content:has(.sp-health-guard-account-dialog)) {
    width: calc(100vw - 2rem);
    max-width: calc(100vw - 2rem);
  }
}

/* >=640px：上下各再多留 1rem。抽组件时这条的 @media 也丢了，会无条件盖掉上面那条。 */
@media (min-width: 640px) {
  :global(.modal-content:has(.sp-health-guard-account-dialog)){
    height: calc(100dvh - 2rem);
    max-height: calc(100dvh - 2rem);
  }
}

:global(.modal-content:has(.sp-health-guard-account-dialog) .modal-footer),
:global(.modal-content:has(.sp-multiplier-interval-dialog) .modal-footer),
:global(.modal-content:has(.sp-run-detail) .modal-footer){
  border-top-color: var(--sp-line);
  background: var(--sp-panel);
}

/* ===== 平台标签组与空态：本弹窗模板在用，但规则原先只写在自动化页的 <style scoped> 里
   （那是 scoped，账号页不加载它就等于没有样式）。这里补一份，与页面/分组弹窗逐字相同。 ===== */
.sp-platform-chip-row {
  display: flex;
  flex: 1 1 100%;
  flex-wrap: wrap;
  align-items: center;
  gap: 6px;
  min-width: 0;
}

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

.sp-rate-guard-empty {
  border-top: 1px solid var(--sp-line);
  border-bottom: 1px solid var(--sp-line);
  color: var(--sp-muted);
  padding: 16px 0;
}
</style>
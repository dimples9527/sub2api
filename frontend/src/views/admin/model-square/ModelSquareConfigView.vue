<template>
  <AppLayout>
    <TablePageLayout>
      <!--
        这里原本有一整条 hero 横栏：图标 + 「模型配置中心」眉题 + 「模型广场配置」标题 + 说明
        + 「最近保存」+ 右侧三个指标卡（已配置平台 / 模型总数 / 无渠道支撑）。已按用户要求整条去掉。
        连带影响都补上了，改回来之前先看这几条：
          · 「有未保存改动」文案挪进保存按钮（sr-only）—— 否则脏标记只剩一个 aria-hidden 的圆点，
            无障碍上等于没有；
          · 三个指标卡的 computed（configuredPlatformCount / totalModelCount / uncoveredModelCount）
            与 .metric-card / .hero-* 样式一并删除，避免留下无引用的死代码；
          · 指标卡承载的两类信息都还在：每个模型的渠道状态在表格的「分组绑定」列里作为独立标签，
            未绑定的模型数挂在平台 Select 的「未绑定 N」徽标上。
        不要再把横栏加回来。
      -->
      <template #filters>
        <div class="space-y-3 rounded-2xl border border-gray-200 bg-white p-4 shadow-sm dark:border-dark-700 dark:bg-dark-800">
          <!--
            平台平铺标签：按平台主色着色，替代原来的平台下拉。点一下即切换当前平台。
            激活态用平台主色实心、未激活用同色淡染，配色统一从 --chip-accent（平台主色）
            经 color-mix 派生 —— 与模型广场展示页的 PlazaFilterBar 同一套 chip 语言，
            新增平台无需扩色板。「未绑定 N」告警跟到各标签上，一眼看出问题在哪个平台。
          -->
          <div class="flex flex-wrap items-center gap-2" role="group" aria-label="选择平台">
            <button
              v-for="card in platformCards"
              :key="card.platform"
              type="button"
              class="platform-chip"
              :class="selectedPlatform === card.platform ? 'chip-tinted-active' : 'chip-tinted'"
              :style="{ '--chip-accent': platformAccentColor(card.platform) }"
              :aria-pressed="selectedPlatform === card.platform"
              @click="selectedPlatform = card.platform"
            >
              <PlatformIcon :platform="platformIconKey(card.platform)" size="xs" />
              <span class="truncate">{{ card.label }}</span>
              <span class="platform-chip-count">{{ card.modelCount }}</span>
              <span v-if="card.uncoveredCount" class="platform-option-warn">未绑定 {{ card.uncoveredCount }}</span>
            </button>
          </div>

          <div class="flex flex-col gap-3 xl:flex-row xl:items-center">
            <div class="w-full md:max-w-sm xl:flex-1">
              <SearchInput v-model="searchQuery" placeholder="搜索模型 ID 或展示名称" />
            </div>

            <div class="flex flex-wrap items-center gap-2 xl:justify-end">
              <button type="button" class="btn btn-secondary" :disabled="loading" @click="requestReload">
                <Icon name="refresh" size="sm" :class="loading ? 'animate-spin' : ''" />
                刷新
              </button>
              <button type="button" class="btn btn-secondary" @click="openBatchDialog">
                <Icon name="upload" size="sm" />
                批量添加
              </button>
              <button type="button" class="btn btn-secondary" @click="openSyncDialog">
                <Icon name="sync" size="sm" />
                同步账号模型
              </button>
              <button type="button" class="btn btn-primary" @click="openModelDialog()">
                <Icon name="plus" size="sm" />
                添加模型
              </button>
              <button
                type="button"
                class="btn btn-primary"
                :class="{ 'is-dirty': isDirty }"
                :disabled="saving"
                @click="saveConfig"
              >
                <Icon name="check" size="sm" />
                {{ saving ? '保存中…' : '保存配置' }}
                <!--
                  文案原本在 hero 横栏上，hero 删除后挪进按钮（sr-only）：
                  圆点是 aria-hidden 的纯装饰，只剩它的话无障碍上就完全不知道有未保存改动了。
                  视觉上仍靠 .is-dirty 的琥珀色提示环 + 圆点，不额外占位。
                -->
                <span v-if="isDirty" class="sr-only">有未保存改动</span>
                <!-- 圆点只是把「该点这里了」提示到视线落点上，不带文案。 -->
                <span v-if="isDirty" class="save-dirty-dot" aria-hidden="true"></span>
              </button>
            </div>
          </div>

          <div v-if="referencePricingLoading" class="reference-pricing-status">
            <Icon name="refresh" size="sm" class="animate-spin" />
            正在加载官方参考价格
          </div>
        </div>
      </template>

      <template #table>
        <!--
          批量操作条：勾选后浮出，取消勾选即消失，不额外占常驻位置。
          计数取的是「当前平台里还找得到的选中模型」，不是勾选键的个数 ——
          模型被删掉或换过平台后键还留着，按键数显示会虚报处理数量。
        -->
        <div v-if="!loadError && selectedModels.length" class="batch-bar">
          <span class="batch-bar-count">已选 {{ selectedModels.length }} 个模型</span>
          <div class="batch-bar-actions">
            <button type="button" class="btn btn-secondary btn-sm" @click="openBatchBindDialog">
              <Icon name="plus" size="sm" />
              绑定分组
            </button>
            <button type="button" class="btn btn-secondary btn-sm" @click="requestClearSelectedGroups">
              清空分组
            </button>
            <button type="button" class="btn btn-secondary btn-sm" @click="clearModelSelection">
              取消选择
            </button>
          </div>
        </div>

        <div v-if="loadError" class="m-4 rounded-xl border border-red-200 bg-red-50 px-4 py-3 text-sm text-red-700 dark:border-red-900/60 dark:bg-red-900/20 dark:text-red-300">
          {{ loadError }}
        </div>

        <DataTable
          v-else
          :columns="columns"
          :data="filteredModels"
          :loading="loading"
          row-key="id"
          sticky-actions-column
          selectable
          v-model:selected-keys="selectedModelKeys"
          :selection-label="modelSelectionLabel"
        >
          <template #empty>
            <EmptyState
              title="当前平台还没有模型"
              description="可以手动添加模型，或从一个同平台上游账号同步模型列表。"
              action-text="添加模型"
              @action="openModelDialog()"
            />
          </template>

          <template #cell-id="{ value }">
            <code class="model-code">{{ value }}</code>
          </template>

          <template #cell-display_name="{ row, value }">
            <div class="min-w-0">
              <div class="truncate font-semibold text-gray-950 dark:text-white">{{ value || row.id }}</div>
              <div v-if="value && value !== row.id" class="mt-1 truncate text-xs text-gray-500 dark:text-dark-400">{{ row.id }}</div>
            </div>
          </template>

          <template #cell-price_summary="{ row }">
            <!--
              每个价格位一张卡片：标签在上（小字、按价格位着色），数字在下（大字、等宽）。

              改之前四个价格挤在两个灰底圆角框里，标签和数字同为 text-xs，
              扫读时分不出主次 —— 这就是「没有层级」的来源。而且「官方参考」徽标挂在组尾，
              视觉上不属于任何一个价格，两个价格里有一个是参考价时读不出来是哪个。

              固定渲染四个价格位（缺价显示「—」）而不是「有什么显示什么」：
              价格列要承担的工作是横向比对多个模型，缺项就塌陷的布局做不到这件事。
            -->
            <div v-if="modelHasAnyPrice(row)" class="price-cards">
              <div
                v-for="slot in modelPriceSlots(row)"
                :key="slot.key"
                :class="['price-card', slot.toneClass]"
              >
                <span class="price-card-label">{{ slot.label }}</span>
                <strong class="price-card-value">{{ slot.value == null ? '—' : (slot.raw ? formatRawPriceSlotValue(slot.value) : formatPriceSlotValue(slot.value)) }}</strong>
                <!--
                  第三行二选一：有官方参考价就标来源，否则显示单位。
                  两个都放会把卡片撑到 103px（单位是 65px、标签 52px，横排相加、竖排则多一行），
                  而展示页那张卡片是 83px —— 一列宽出去 120px，整表就得横向滚动。
                  二选一之后第三行宽度恒定是那个更宽的单位，卡片回到 83×76，与展示页一致。
                -->
                <div v-if="slot.source === 'official' || slot.unit" class="price-card-meta">
                  <span v-if="slot.source === 'official'" class="price-card-tag">官方参考</span>
                  <small v-else>{{ slot.unit }}</small>
                </div>
              </div>
            </div>
            <span v-else class="price-empty">{{ modelPriceEmptyText(row) }}</span>
          </template>

          <template #cell-group_binding="{ row }">
            <!--
              分组名要查分组列表才能解析，所以上下文没拉出来时说「—」而不是「未绑定分组」——
              后者会把一次接口抖动读成配置缺失，管理员会去重复绑定。
              整个单元格是按钮：这是表格里的快捷编辑入口，点开即可改绑定，不必进「编辑模型」。
            -->
            <button
              type="button"
              class="group-binding"
              :title="groupBindingCellTitle(row)"
              @click="openGroupBindDialog(row)"
            >
              <span v-if="modelBoundGroupList(row) === null" class="group-binding-pending">—</span>
              <span v-else-if="(modelBoundGroupList(row) || []).length === 0" class="group-binding-empty">
                未绑定分组
              </span>
              <template v-else>
                <span
                  v-for="group in (modelBoundGroupList(row) || []).slice(0, 2)"
                  :key="String(group.id)"
                  class="group-chip"
                  :class="groupPlatformBadgeClass(group, groupContext?.platformOverrides)"
                  :title="`${group.name} · ${groupPlatformLabel(group, groupContext?.platformOverrides)}`"
                >
                  {{ group.name }}
                </span>
                <span v-if="(modelBoundGroupList(row) || []).length > 2" class="group-chip-more">
                  +{{ (modelBoundGroupList(row) || []).length - 2 }}
                </span>
              </template>
            </button>
          </template>

          <template #cell-source="{ value }">
            <span :class="['source-badge', value === 'sync' ? 'source-sync' : 'source-manual']">
              {{ value === 'sync' ? '上游同步' : '手动维护' }}
            </span>
          </template>

          <template #cell-actions="{ row }">
            <div class="flex justify-end gap-2">
              <button type="button" class="btn btn-secondary btn-sm" @click="openModelDialog(row)">
                编辑
              </button>
              <button type="button" class="btn btn-danger btn-sm" @click="askRemoveModel(row)">
                删除
              </button>
            </div>
          </template>
        </DataTable>
      </template>
    </TablePageLayout>

    <BaseDialog :show="modelDialogVisible" :title="editingModelId ? '编辑模型' : '添加模型'" width="extra-wide" @close="closeModelDialog">
      <div class="model-dialog-grid" :class="{ 'is-single': modelBillingMode !== 'token' }">
        <div class="space-y-4">
          <div class="grid grid-cols-1 gap-3 sm:grid-cols-2">
            <Input v-model="modelForm.id" label="模型 ID" placeholder="例如：gpt-5.2" required />
            <Input v-model="modelForm.display_name" label="展示名称" placeholder="不填则使用模型 ID" />
          </div>

          <ModelGroupBindPicker
            v-model="modelForm.group_ids"
            :groups="groupContext?.groups || []"
            :platform="selectedPlatform"
            :platform-overrides="groupContext?.platformOverrides"
            :context-state="groupContextState"
          />

          <!--
            计费类型二选一：按量（token 价格，之前的默认）或图/张（每张一口价）。
            两种量纲互斥 —— 一个模型要么按 token 计费、要么按张计费，切换时只展示并保存对应那套字段，
            另一套在保存时清掉，避免展示页拿到「既有 token 价又有按张价」的混合数据不知道按哪个显示。
          -->
          <div class="model-billing-mode">
            <span class="model-billing-mode-label">计费类型</span>
            <div class="model-billing-mode-options" role="group" aria-label="计费类型">
              <button
                type="button"
                class="model-billing-mode-btn"
                :class="{ 'is-active': modelBillingMode === 'token' }"
                :aria-pressed="modelBillingMode === 'token'"
                @click="modelBillingMode = 'token'"
              >
                按量计费
              </button>
              <button
                type="button"
                class="model-billing-mode-btn"
                :class="{ 'is-active': modelBillingMode === 'per_image' }"
                :aria-pressed="modelBillingMode === 'per_image'"
                @click="modelBillingMode = 'per_image'"
              >
                图 / 张计费
              </button>
            </div>
          </div>

          <template v-if="modelBillingMode === 'token'">
          <div class="model-dialog-section-head">
            <div class="min-w-0">
              <h4 class="model-dialog-section-title">Token 价格</h4>
              <!--
                原来这段是放在天蓝色说明框里的一整段长文案。这里压成一行并去掉装饰容器，
                但两句「完成当前操作不可缺少」的信息必须留着：单位换算（不写就不知道填的是
                每 token 还是每百万）和回填规则（不写会以为点一下就把自己的定价覆盖了）。
              -->
              <p class="model-dialog-section-note">
                Token 价格按 USD / 1M Tokens 录入，保存时换算为计费使用的 USD / Token。
                官方参考价格来自项目动态价格目录（远程不可用时使用内置回退目录），
                获取默认价格只会回填当前未填写的价格字段，已填写的价格不会被覆盖。
              </p>
            </div>
            <button
              type="button"
              class="btn btn-secondary btn-sm ml-auto shrink-0"
              :disabled="defaultPricingLoading"
              @click="applyDefaultPricing"
            >
              <Icon name="refresh" size="sm" :class="defaultPricingLoading ? 'animate-spin' : ''" />
              {{ defaultPricingLoading ? '查询中…' : '获取默认价格' }}
            </button>
          </div>

          <div class="grid grid-cols-1 gap-3 sm:grid-cols-2">
            <div v-for="price in PRICE_FIELDS" :key="price.key" class="model-price-field">
              <!--
                字段名必须继续走 Input 的 label：Input 没有 inheritAttrs / $attrs 透传，
                额外的 aria-label 只会落到外层 div 上，内层 input 拿不到 —— 那样输入框就没有
                无障碍名称了。所以状态徽标只能放在输入框下方自己拼一行，不能挤进标签行。
                注意 Input 的 `id` prop 没有默认值，不传时 label 的 `for` 与 input 的 `id`
                会一起被省略（即标签并未真正关联到输入框）；要修的是在这里补 `:id`，
                而不是去改通用组件。
              -->
              <Input
                v-model="modelForm[price.key]"
                type="number"
                :label="`${price.label}（USD / 1M Tokens）`"
                :placeholder="modelPricePlaceholder(price.key)"
              />
              <div class="model-price-field-foot">
                <!--
                  表格里已经有「官方参考」徽标区分价格来源，弹窗里原本没有，
                  编辑时看不出哪些字段是自己填的、哪些还在吃官方价。
                  注意「没有官方参考价」必须单独说成未设置 —— 那时并没有官方价可跟随。
                -->
                <span :class="['model-price-field-state', `is-${modelPriceFieldState(price.key)}`]">
                  {{ MODEL_PRICE_FIELD_STATE_LABELS[modelPriceFieldState(price.key)] }}
                </span>
                <span class="model-price-field-baseline">{{ modelPriceBaselineText(price.key) }}</span>
              </div>
            </div>
          </div>
          </template>

          <!--
            图/张计费：每生成一张图收取的固定价格。与 token 价格是两套量纲，切到这个类型时
            token 网格与基准价对照都不展示，保存时也会清掉本页的 token 价格字段。
          -->
          <div v-else class="model-request-price">
            <Input
              v-model="modelForm.per_request_price"
              type="number"
              label="按张价格（USD / 张）"
              placeholder="例如：0.04"
            />
            <p class="model-request-price-note">
              每生成一张图片收取的固定价格，直接按 USD 录入，不按 token 换算。留空表示暂未定价。
            </p>
          </div>
        </div>

        <aside v-if="modelBillingMode === 'token'" class="model-baseline-aside">
          <div class="model-baseline-head">
            <h4 class="model-baseline-title">基准价对照</h4>
            <p class="model-baseline-note">官方参考价 · USD / 1M Tokens</p>
          </div>

          <div class="model-baseline-rows">
            <div v-for="row in modelBaselineRows" :key="row.key" :class="['model-baseline-row', row.tone]">
              <div class="model-baseline-row-top">
                <span class="model-baseline-row-name">{{ row.label }}</span>
                <span :class="['model-baseline-row-value', row.valueTone]">{{ row.valueText }}</span>
              </div>
              <div class="model-baseline-row-meta">
                <span>{{ row.meta }}</span>
                <span v-if="row.badge" class="model-baseline-ratio">{{ row.badge }}</span>
              </div>
            </div>
          </div>

          <div class="model-baseline-summary">
            <span class="model-baseline-summary-label">已自定义</span>
            <span class="model-baseline-summary-value">{{ modelConfiguredPriceCount }} / {{ PRICE_FIELDS.length }}</span>
          </div>
          <!--
            展示页对「配置值和官方值都没有」的字段渲染成「未设置」而不是 $0
            （modelSquare.ts 的 assignPrice 只在非 null 时写入），这里必须说准。
          -->
          <p class="model-baseline-foot">
            留空的字段在展示页按官方基准价显示；官方目录也没有时显示为「未设置」，不会写成 $0。
          </p>
          <!--
            预填会把官方价「钉」成自己的定价：保存后该字段就有值了，官方后续调价不会再跟上，
            「获取默认价格」也只回填空字段。不写清楚的话，管理员会以为留空就等于一直跟随官方。
          -->
          <p class="model-baseline-hint">
            表单里未填写的字段已预填当前官方价。一旦保存，这些字段就成为你自己的定价，
            官方后续调价不会自动跟随；要重新同步到最新官方价，需先清空该字段再点「获取默认价格」。
          </p>
        </aside>
      </div>
      <template #footer>
        <button type="button" class="btn btn-secondary" @click="closeModelDialog">取消</button>
        <button type="button" class="btn btn-primary" @click="submitModelDialog">保存</button>
      </template>
    </BaseDialog>

    <!--
      表格「分组绑定」列的快捷入口。复用与编辑弹窗同一个选择控件 ——
      两处各写一套的话，改了一处忘另一处是最难发现的那类错。
    -->
    <BaseDialog
      :show="groupBindDialogVisible"
      :title="groupBindDialogTitle"
      width="wide"
      @close="closeGroupBindDialog"
    >
      <ModelGroupBindPicker
        v-model="groupBindForm"
        :groups="groupContext?.groups || []"
        :platform="selectedPlatform"
        :platform-overrides="groupContext?.platformOverrides"
        :context-state="groupContextState"
      />
      <template #footer>
        <button type="button" class="btn btn-secondary" @click="closeGroupBindDialog">取消</button>
        <button type="button" class="btn btn-primary" @click="submitGroupBindDialog">保存</button>
      </template>
    </BaseDialog>

    <!--
      批量绑定：复用与单个模型同一个选择控件（mode="batch" 只换文案，勾选逻辑一模一样）。
      另写一套选择列表的话，平台兼容过滤、排序去重口径迟早会分叉 ——
      而口径分叉的后果是保存后才暴露的假脏标记。
    -->
    <BaseDialog
      :show="batchBindDialogVisible"
      title="批量绑定分组"
      width="wide"
      @close="closeBatchBindDialog"
    >
      <!--
        必须写清作用对象：批量操作最怕的是「以为只改了勾中的几个」。
        这里点明平台 + 数量 + 追加语义，是完成本次操作不可缺少的信息。
      -->
      <p class="batch-bind-targets">
        为 {{ currentPlatformLabel }} 下勾选的 {{ selectedModels.length }} 个模型追加分组绑定。
      </p>
      <ModelGroupBindPicker
        v-model="batchBindForm"
        mode="batch"
        :groups="groupContext?.groups || []"
        :platform="selectedPlatform"
        :platform-overrides="groupContext?.platformOverrides"
        :context-state="groupContextState"
      />
      <template #footer>
        <button type="button" class="btn btn-secondary" @click="closeBatchBindDialog">取消</button>
        <button
          type="button"
          class="btn btn-primary"
          :disabled="batchBindForm.length === 0"
          @click="submitBatchBindDialog"
        >
          绑定
        </button>
      </template>
    </BaseDialog>

    <BaseDialog :show="batchDialogVisible" title="批量添加模型" width="wide" @close="closeBatchDialog">
      <div class="space-y-4">
        <TextArea
          v-model="batchText"
          label="模型列表"
          rows="10"
          placeholder="每行一个模型 ID，也支持用逗号或空格分隔"
          hint="重复模型会自动跳过，新增模型来源会标记为手动维护。"
        />
      </div>
      <template #footer>
        <button type="button" class="btn btn-secondary" @click="closeBatchDialog">取消</button>
        <button type="button" class="btn btn-primary" @click="submitBatchDialog">添加</button>
      </template>
    </BaseDialog>

    <BaseDialog :show="syncDialogVisible" title="同步上游账号模型" width="wide" @close="closeSyncDialog">
      <div class="space-y-4">
          <div class="rounded-xl border border-emerald-200 bg-emerald-50 px-4 py-3 text-sm text-emerald-800 dark:border-emerald-500/30 dark:bg-emerald-500/10 dark:text-emerald-100">
            将调用所选账号的上游模型列表接口，并把返回模型合并到当前平台配置中。已有模型不会重复添加。
          </div>
          <Select
            v-model="syncAccountId"
            :options="syncAccountOptions"
            :disabled="syncAccountsLoading"
            searchable
            clearable
            :placeholder="syncAccountsLoading ? '正在加载账号' : '选择同平台账号'"
            empty-text="当前平台暂无可选账号"
          aria-label="选择同步账号"
        />
        <div class="grid grid-cols-1 gap-3 sm:grid-cols-2 lg:grid-cols-4">
          <div class="sync-meta-card">
            <span>当前平台</span>
            <strong>{{ currentPlatformLabel }}</strong>
          </div>
          <div class="sync-meta-card">
            <span>可选账号</span>
            <strong>{{ syncAccountOptions.length }}</strong>
          </div>
          <div class="sync-meta-card">
            <span>上次同步</span>
            <strong>{{ formatTime(currentConfig.synced_at || null) }}</strong>
          </div>
          <!--
            synced_from_account_name 一直被写进配置（submitSyncDialog 里赋的值），但从来没显示过。
            「上次同步」只说了什么时候，没说用的是哪个账号 —— 而「上次是不是用了另一个账号」
            正是决定要不要重新同步的关键信息，所以单独给一格而不是塞进上面那格。
          -->
          <div class="sync-meta-card">
            <span>上次同步账号</span>
            <strong :title="currentConfig.synced_from_account_name || ''">
              {{ currentConfig.synced_from_account_name || '—' }}
            </strong>
          </div>
        </div>
      </div>
      <template #footer>
        <button type="button" class="btn btn-secondary" @click="closeSyncDialog">取消</button>
        <button type="button" class="btn btn-primary" :disabled="syncing || !syncAccountId" @click="submitSyncDialog">
          {{ syncing ? '同步中…' : '开始同步' }}
        </button>
      </template>
    </BaseDialog>

    <ConfirmDialog
      :show="Boolean(modelPendingRemove)"
      title="删除模型"
      :message="`确认从 ${currentPlatformLabel} 配置中删除模型 ${modelPendingRemove?.id || ''} 吗？`"
      confirm-text="删除"
      cancel-text="取消"
      danger
      @confirm="confirmRemoveModel"
      @cancel="modelPendingRemove = null"
    />

    <!--
      「刷新」是整块重新拉取 + 整体覆盖，未保存的编辑会一起没掉。
      用页面已有的 ConfirmDialog 而不是 window.confirm：它是页内动作，
      和上面的「删除模型」保持同一套确认样式。
    -->
    <ConfirmDialog
      :show="reloadConfirmVisible"
      title="刷新配置"
      message="当前有未保存的改动，刷新会丢弃这些改动并重新从服务端加载。确认刷新吗？"
      confirm-text="丢弃并刷新"
      cancel-text="取消"
      danger
      @confirm="confirmReload"
      @cancel="reloadConfirmVisible = false"
    />

    <!--
      批量清空要问一句：它一次抹掉的是多个模型的全部分组绑定，误触后逐个补回来很费事。
      改动本身在内存里、保存配置前不落盘，但正因为「反正没保存」才更容易顺手点保存。
    -->
    <ConfirmDialog
      :show="clearGroupsConfirmVisible"
      title="清空分组绑定"
      :message="`确认清空这 ${selectedModels.length} 个模型的全部已绑定分组吗？它们会从所有分组下消失，保存配置后生效。`"
      confirm-text="清空分组"
      cancel-text="取消"
      danger
      @confirm="confirmClearSelectedGroups"
      @cancel="clearGroupsConfirmVisible = false"
    />
    <!--
      保存前的冲突确认。后端 UpdateModelSquareConfig 完全不比对入参的 updated_at，
      两个人同时编辑时后保存的会静默覆盖前一个人的改动 —— 这个弹窗是唯一的拦截点。
      用 danger 是因为「覆盖保存」确实会丢掉对方的工作，不是普通确认。
    -->
    <ConfirmDialog
      :show="conflictConfirmVisible"
      title="配置已被他人修改"
      message="服务端的模型广场配置在你打开本页之后被改过。继续保存会用你当前的改动整体覆盖对方的改动，且无法撤销。"
      confirm-text="覆盖保存"
      cancel-text="取消"
      danger
      @confirm="confirmOverwriteConfig"
      @cancel="conflictConfirmVisible = false"
    />
  </AppLayout>
</template>

<script setup lang="ts">
import { computed, onMounted, onUnmounted, ref, watch } from 'vue'
import { onBeforeRouteLeave } from 'vue-router'
import { adminAPI } from '@/api/admin'
// 分组 ID 规范化与上下文类型直接从模型广场 API 模块取：规范化口径必须与保存载荷、
// 聚合函数三处完全一致，任一处形态不同都会让「未保存改动」比对误报。
import {
  normalizeModelGroupIDs,
  type ModelSquareGroupContext,
  type ModelSquareUserGroup,
} from '@/api/admin/modelSquare'
import type {
  ModelSquareConfigPayload,
  ModelSquareOfficialPricing,
  ModelSquarePlatformConfig,
  ModelSquarePlatformModelConfig,
  ModelSquareSyncAccountCandidate,
} from '@/api/admin/modelSquareConfig'
import type { CustomPlatform } from '@/api/admin/customPlatforms'
import type { Account } from '@/types'
import type { Column } from '@/components/common/types'
import type { SelectOption } from '@/components/common/Select.vue'
import { useAppStore } from '@/stores/app'
import { extractApiErrorMessage } from '@/utils/apiError'
import { resolvePlatformDisplayLabel, setCustomPlatformLabels } from '@/utils/customPlatformLabels'
import { platformAccentColor } from '@/utils/platformColors'
import AppLayout from '@/components/layout/AppLayout.vue'
import TablePageLayout from '@/components/layout/TablePageLayout.vue'
import DataTable from '@/components/common/DataTable.vue'
import EmptyState from '@/components/common/EmptyState.vue'
import BaseDialog from '@/components/common/BaseDialog.vue'
import ConfirmDialog from '@/components/common/ConfirmDialog.vue'
import Input from '@/components/common/Input.vue'
import SearchInput from '@/components/common/SearchInput.vue'
import Select from '@/components/common/Select.vue'
import TextArea from '@/components/common/TextArea.vue'
import PlatformIcon from '@/components/common/PlatformIcon.vue'
import Icon from '@/components/icons/Icon.vue'
import ModelGroupBindPicker from './ModelGroupBindPicker.vue'
import { groupPlatformBadgeClass, groupPlatformLabel } from './groupPlatformStyle'

const BUILTIN_PLATFORMS = [
  { platform: 'anthropic', name: 'Anthropic' },
  { platform: 'openai', name: 'OpenAI' },
  { platform: 'gemini', name: 'Gemini' },
  { platform: 'antigravity', name: 'Antigravity' },
  { platform: 'grok', name: 'Grok' },
]

const PRICE_FIELDS = [
  { key: 'input_price', label: '输入价格' },
  { key: 'output_price', label: '输出价格' },
  { key: 'cache_write_price', label: '缓存写入价格' },
  { key: 'cache_read_price', label: '缓存读取价格' },
] as const

const PRICE_PER_MILLION_TOKENS = 1_000_000

type PriceField = typeof PRICE_FIELDS[number]['key']
// per_request_price（按张一口价）单独拎出来，不进 PRICE_FIELDS：它是「每张图固定价」，
// 直接按 USD 录入、不按 token 换算，也没有官方参考价可跟随——塞进 token 价格位会连带
// 触发 1M 换算、基准价对照、「已自定义 N/4」计数，语义全错。
type ModelForm = { id: string; display_name: string; group_ids: number[]; per_request_price: string } & Record<PriceField, string>
type PriceSource = 'configured' | 'official'
// 价格位的色调。与展示页的价格卡片同源，两个页面的「输入是青、输出是橙」必须一致，
// 否则管理员在两个页面之间切换时会以为价格换了含义。
type PriceTone = 'teal' | 'orange' | 'violet' | 'blue'
type PriceSlot = {
  key: PriceField | 'per_request_price'
  label: string
  unit: string
  toneClass: string
  // null 表示这个价格位既没有自定义价、也没有官方参考价 —— 与「价格是 0」是两回事，
  // 所以不能拿 0 兜底：配置页里 $0 会被读成「这一项免费」，那就不是样式问题了。
  value: number | null
  source: PriceSource | null
  // 按张价格是绝对单价，展示时不能再 ×1M。raw 为真时走原样格式化。
  raw?: boolean
}
type OfficialPricingStatus = 'loading' | 'found' | 'not_found' | 'error'
// 弹窗里每个价格字段的来源状态。'unset' 必须和 'official' 分开：
// 没有官方参考价时「跟随官方」是误导 —— 那时并没有官方价可跟。
type ModelPriceFieldState = 'custom' | 'official' | 'unset'

const MODEL_PRICE_FIELD_STATE_LABELS: Record<ModelPriceFieldState, string> = {
  custom: '已自定义',
  official: '跟随官方',
  unset: '未设置',
}

const createEmptyModelForm = (): ModelForm => ({
  id: '',
  display_name: '',
  group_ids: [],
  input_price: '',
  output_price: '',
  cache_write_price: '',
  cache_read_price: '',
  per_request_price: '',
})

const columns: Column[] = [
  { key: 'id', label: '模型 ID', sortable: true },
  { key: 'display_name', label: '展示名称', sortable: true },
  { key: 'price_summary', label: '价格（每 1M Tokens）', sortable: false },
  // 分组归属是配置里存的字段（模型的 group_ids），不再由渠道反推。
  // 配置页是唯一能加模型的地方，如果这里不给出分组归属，管理员加完模型
  // 到展示页会发现按分组筛不到，却在配置页看不出任何异常。
  { key: 'group_binding', label: '分组绑定', sortable: false },
  { key: 'source', label: '来源', sortable: true },
  { key: 'actions', label: '操作', sortable: false },
]

const appStore = useAppStore()
const loading = ref(false)
const saving = ref(false)
const syncing = ref(false)
const loadError = ref('')
const selectedPlatform = ref('openai')
const searchQuery = ref('')
const configUpdatedAt = ref<string | null>(null)
/**
 * 上次与后端同步（加载成功或保存成功）时的载荷快照，用来判断有没有未保存改动。
 * 初值取「空配置」的序列化结果而不是空串：加载失败时 platformConfigs 是空的，
 * 拿空串去比会把「什么都没加载出来」误报成「有未保存改动」。
 */
const savedSnapshot = ref(stableSerialize({ platforms: [] }))
const reloadConfirmVisible = ref(false)
const conflictConfirmVisible = ref(false)
const platformConfigs = ref<ModelSquarePlatformConfig[]>([])
const customPlatforms = ref<CustomPlatform[]>([])
const accounts = ref<Account[]>([])
const syncAccounts = ref<ModelSquareSyncAccountCandidate[]>([])
const syncAccountsLoading = ref(false)

const modelDialogVisible = ref(false)
const editingModelId = ref<string | null>(null)
const modelForm = ref<ModelForm>(createEmptyModelForm())
// 计费类型：token=按量（默认），per_image=图/张。打开弹窗时按已配置的按张价推断，
// 有按张价即认定为图/张类型。控制弹窗里展示并保存哪一套价格字段。
type ModelBillingMode = 'token' | 'per_image'
const modelBillingMode = ref<ModelBillingMode>('token')
const defaultPricingLoading = ref(false)
const referencePricingLoadingCount = ref(0)
const referencePricingLoading = computed(() => referencePricingLoadingCount.value > 0)
const officialPricingMap = ref<Record<string, ModelSquareOfficialPricing>>({})
const officialPricingStatusMap = ref<Record<string, OfficialPricingStatus>>({})
const batchDialogVisible = ref(false)
const batchText = ref('')
const syncDialogVisible = ref(false)
const syncAccountId = ref<number | null>(null)
const modelPendingRemove = ref<ModelSquarePlatformModelConfig | null>(null)

const normalizePlatform = (platform?: string | null) => (platform || '').trim().toLowerCase()
const normalizeModelId = (id?: string | null) => (id || '').trim()
const modelKey = (id: string) => id.trim().toLowerCase()
const platformIconKey = (platform: string) => platform as any
const referencePricingInFlightKeys = new Set<string>()
const referencePricingPromises = new Map<string, Promise<void>>()

/**
 * 分组归属上下文：渠道、分组、分组平台覆盖。
 * 这三份数据与配置本身无关，进页面拉一次即可。拉不到时不让整页失败 ——
 * 分组归属只是辅助信息，缺了它管理员仍应能维护模型清单，该列退化为「—」。
 */
const groupContext = ref<ModelSquareGroupContext | null>(null)
const groupContextLoading = ref(false)

/**
 * 分组上下文对 UI 的三态。
 * 只传「有没有 groups」是不够的：加载中与加载失败都必须和「确实没有可绑定的分组」
 * 区分开，否则一次接口抖动会被读成配置缺失。
 */
const groupContextState = computed<'loading' | 'unavailable' | 'ready'>(() => {
  if (groupContext.value) return 'ready'
  return groupContextLoading.value ? 'loading' : 'unavailable'
})

const groupById = computed(() => new Map((groupContext.value?.groups || []).map(group => [String(group.id), group])))

/**
 * 模型绑定的分组名单。返回 null 表示「分组名解析不出来」（分组上下文没加载出来），
 * 必须和「确实没绑定」区分开：前者是一次接口抖动，重试即可；后者是配置缺失，
 * 需要管理员去绑定。两者的提示语与处理动作完全不同。
 */
function modelBoundGroupList(model: ModelSquarePlatformModelConfig): ModelSquareUserGroup[] | null {
  const ids = model.group_ids || []
  if (ids.length === 0) return []
  if (!groupContext.value) return null
  return ids
    .map(id => groupById.value.get(String(id)))
    .filter((group): group is ModelSquareUserGroup => Boolean(group))
}

/**
 * 列里最多展示两个分组名，剩下的折成 +N，完整名单挂在 title 上。
 *
 * 每个名字后面补上平台：chip 的平台信息只靠颜色传达，色觉障碍下读不出来，
 * 悬浮说明必须把它写全。
 */
function modelBoundGroupsTitle(groups: ModelSquareUserGroup[]): string {
  if (groups.length === 0) return ''
  const overrides = groupContext.value?.platformOverrides
  const names = groups.map(group => `${group.name}（${groupPlatformLabel(group, overrides)}）`)
  return `已绑定分组：${names.join('、')}`
}

/** 分组绑定单元格的悬浮说明。三种状态给三种说法，不能混用。 */
function groupBindingCellTitle(model: ModelSquarePlatformModelConfig): string {
  const groups = modelBoundGroupList(model)
  if (groups === null) return '分组数据加载失败，点「刷新」可重试'
  if (groups.length === 0) return '未绑定分组：展示页按任何分组都筛不到这个模型。点击选择分组。'
  return `${modelBoundGroupsTitle(groups)}。点击修改。`
}

/**
 * 没有绑定任何分组的模型数：这些模型在展示页按任何分组都筛不到。
 *
 * 不看分组上下文是否加载成功 —— group_ids 是配置里存的，不依赖外部接口，
 * 所以这个数字在接口抖动时依然准确，不需要像从前那样区分「未知」。
 */
function countUnboundModels(models: ModelSquarePlatformModelConfig[]): number {
  return models.filter(model => (model.group_ids || []).length === 0).length
}

const platformLabelMap = computed(() => {
  const labels = new Map<string, string>()
  for (const item of BUILTIN_PLATFORMS) labels.set(item.platform, item.name)
  for (const item of customPlatforms.value) {
    const platform = normalizePlatform(item.code)
    if (!platform || labels.has(platform)) continue
    labels.set(platform, item.name.trim() || resolvePlatformDisplayLabel(platform))
  }
  for (const account of accounts.value) labels.set(normalizePlatform(account.platform), resolvePlatformDisplayLabel(account.platform))
  for (const config of platformConfigs.value) {
    const platform = normalizePlatform(config.platform)
    if (!platform) continue
    labels.set(platform, config.name?.trim() || resolvePlatformDisplayLabel(platform))
  }
  return labels
})

const platformCards = computed(() => {
  const platforms = new Map<string, { platform: string; label: string; modelCount: number; uncoveredCount: number; rank: number; order: number }>()
  for (const [index, item] of BUILTIN_PLATFORMS.entries()) {
    platforms.set(item.platform, { platform: item.platform, label: item.name, modelCount: 0, uncoveredCount: 0, rank: 0, order: index })
  }
  for (const item of [...customPlatforms.value].sort((left, right) => {
    const sortOrderDiff = (left.sort_order ?? 0) - (right.sort_order ?? 0)
    if (sortOrderDiff !== 0) return sortOrderDiff
    return left.name.localeCompare(right.name, 'zh-Hans-CN')
  })) {
    const platform = normalizePlatform(item.code)
    if (!platform || platforms.has(platform)) continue
    platforms.set(platform, {
      platform,
      label: item.name.trim() || resolvePlatformDisplayLabel(platform),
      modelCount: 0,
      uncoveredCount: 0,
      rank: 1,
      order: item.sort_order ?? 0,
    })
  }
  for (const account of accounts.value) {
    const platform = normalizePlatform(account.platform)
    if (!platform || platforms.has(platform)) continue
    platforms.set(platform, { platform, label: resolvePlatformDisplayLabel(platform), modelCount: 0, uncoveredCount: 0, rank: 2, order: 0 })
  }
  for (const config of platformConfigs.value) {
    const platform = normalizePlatform(config.platform)
    if (!platform) continue
    const existing = platforms.get(platform)
    platforms.set(platform, {
      platform,
      label: config.name?.trim() || existing?.label || resolvePlatformDisplayLabel(platform),
      modelCount: config.models?.length || 0,
      uncoveredCount: countUnboundModels(config.models || []),
      rank: existing?.rank ?? 3,
      order: existing?.order ?? 0,
    })
  }
  return Array.from(platforms.values()).sort((left, right) => {
    if (left.rank !== right.rank) return left.rank - right.rank
    if (left.order !== right.order) return left.order - right.order
    return left.label.localeCompare(right.label, 'zh-Hans-CN')
  })
})

const currentPlatformLabel = computed(() => platformLabelMap.value.get(selectedPlatform.value) || resolvePlatformDisplayLabel(selectedPlatform.value))
const currentConfig = computed<ModelSquarePlatformConfig>(() => {
  return platformConfigs.value.find(item => normalizePlatform(item.platform) === selectedPlatform.value) || createPlatformConfig(selectedPlatform.value)
})
const currentModels = computed(() => currentConfig.value.models || [])
const filteredModels = computed(() => {
  const keyword = searchQuery.value.trim().toLowerCase()
  if (!keyword) return currentModels.value
  return currentModels.value.filter(model => `${model.id} ${model.display_name || ''}`.toLowerCase().includes(keyword))
})
const syncAccountOptions = computed<SelectOption[]>(() => syncAccounts.value
  .map(account => ({
    value: account.id,
    label: [
      account.name || `#${account.id}`,
      account.type,
      account.status,
      ...(account.group_names || []),
    ].filter(Boolean).join(' · '),
  })))

function createPlatformConfig(platform: string): ModelSquarePlatformConfig {
  return {
    platform,
    name: platformLabelMap.value.get(platform) || resolvePlatformDisplayLabel(platform),
    synced_from_account_id: null,
    synced_from_account_name: '',
    synced_at: null,
    models: [],
  }
}

function modelPriceValues(model: ModelSquarePlatformModelConfig): Pick<ModelSquarePlatformModelConfig, PriceField> {
  return {
    input_price: model.input_price ?? null,
    output_price: model.output_price ?? null,
    cache_write_price: model.cache_write_price ?? null,
    cache_read_price: model.cache_read_price ?? null,
  }
}

function storedPriceToDisplayPrice(value?: number | null): string {
  if (value == null || !Number.isFinite(value)) return ''
  return formatPlainPriceNumber(value * PRICE_PER_MILLION_TOKENS)
}

function displayPriceToStoredPrice(value: number): number {
  return value / PRICE_PER_MILLION_TOKENS
}

// 按张价格是「每张图的绝对单价」，存取都不做 1M 换算——展示页也是直接按张显示、不除以百万。
function storedRequestPriceToDisplay(value?: number | null): string {
  if (value == null || !Number.isFinite(value)) return ''
  return formatPlainPriceNumber(value)
}

function formatPlainPriceNumber(value: number): string {
  if (!Number.isFinite(value)) return ''
  return new Intl.NumberFormat('en-US', {
    minimumFractionDigits: Number.isInteger(value) ? 0 : 2,
    maximumFractionDigits: 6,
    useGrouping: false,
  }).format(value)
}

function ensureCurrentConfig(): ModelSquarePlatformConfig {
  const platform = normalizePlatform(selectedPlatform.value)
  let config = platformConfigs.value.find(item => normalizePlatform(item.platform) === platform)
  if (!config) {
    config = createPlatformConfig(platform)
    platformConfigs.value.push(config)
  }
  config.platform = platform
  if (!config.name) config.name = currentPlatformLabel.value
  if (!Array.isArray(config.models)) config.models = []
  return config
}

function normalizeConfig(input: ModelSquareConfigPayload): void {
  configUpdatedAt.value = input.updated_at || null
  platformConfigs.value = (input.platforms || []).map(item => ({
    platform: normalizePlatform(item.platform),
    name: item.name || resolvePlatformDisplayLabel(item.platform),
    synced_from_account_id: item.synced_from_account_id ?? null,
    synced_from_account_name: item.synced_from_account_name || '',
    synced_at: item.synced_at || null,
    models: dedupeModels(item.models || []),
  })).filter(item => item.platform)
  // 刚加载出来（或刚存回去）的这份配置就是新的基准，未保存状态随之清零。
  savedSnapshot.value = stableSerialize(buildSavePayload())
}

function dedupeModels(models: ModelSquarePlatformModelConfig[]): ModelSquarePlatformModelConfig[] {
  const seen = new Set<string>()
  const result: ModelSquarePlatformModelConfig[] = []
  for (const model of models) {
    const id = normalizeModelId(model.id)
    if (!id) continue
    const key = modelKey(id)
    if (seen.has(key)) continue
    seen.add(key)
    const groupIDs = normalizeModelGroupIDs(model.group_ids)
    // 先展开原对象，再规范化本页真正负责的那几个字段：
    // 保存走的是整块 PUT、后端整体覆盖配置，所以这里如果只挑字段重建对象，
    // 后端支持但本页没展示的那些价格位（优先级价、1 小时缓存写入价、图片价、按请求价）
    // 就会被静默写成 null —— 展示页对这 11 个 token 价格位会悄悄回退成官方参考价
    // （管理员以为自己的定价生效了）；按请求价没有官方兜底，会变成「未设置」不再显示。
    result.push({
      ...model,
      id,
      display_name: (model.display_name || id).trim(),
      source: model.source === 'sync' ? 'sync' : 'manual',
      /*
        group_ids 也要在这里规范化，理由同上：保存快照取自本函数的产出，而重新加载后
        拿到的是后端 normalize 过的形态，两边口径不同就会让「未保存改动」一直亮着。
        空绑定写成 undefined 而不是 []：后端 json 的 omitempty 会省略这个字段，
        前端留一个空数组就与后端返回的形态对不上，脏检测随即误报。
      */
      group_ids: groupIDs.length ? groupIDs : undefined,
      ...modelPriceValues(model),
    })
  }
  return result
}

/**
 * 保存用的规范化载荷。saveConfig 与「未保存检测」共用它 ——
 * 两边各写一份的话，改了一处忘另一处就会出现「明明改了却显示已保存」这种最坏的假阴性。
 */
function buildSavePayload(): ModelSquareConfigPayload {
  return {
    platforms: platformConfigs.value
      .map(config => ({
        ...config,
        platform: normalizePlatform(config.platform),
        name: config.name?.trim() || resolvePlatformDisplayLabel(config.platform),
        models: dedupeModels(config.models || []),
      }))
      .filter(config => config.platform),
  }
}

/**
 * 稳定序列化：先把每个对象的 key 排序再 stringify。
 * 直接 stringify 会被「键的插入顺序」影响 —— 编辑过的模型是
 * `{ ...原对象, id, display_name, source, ...价格 }` 拼出来的，键序可能与刚加载时不同，
 * 于是逻辑上没变也会被判成「有未保存改动」，提示就变成了狼来了。
 */
function stableSerialize(value: unknown): string {
  return JSON.stringify(value, (_key, item) => {
    if (item && typeof item === 'object' && !Array.isArray(item)) {
      const entries = Object.entries(item as Record<string, unknown>)
      entries.sort(([left], [right]) => (left < right ? -1 : left > right ? 1 : 0))
      return Object.fromEntries(entries)
    }
    return item
  })
}

/**
 * 有没有未保存改动。用「当前载荷 vs 上次同步快照」比对得出，而不是在每个改动点手动置脏 ——
 * 手动置脏一定会漏掉某个入口（批量添加、同步、删除……），而漏掉的那次就是一次静默的数据丢失。
 */
const isDirty = computed(() => stableSerialize(buildSavePayload()) !== savedSnapshot.value)

/**
 * 拉取分组归属上下文。失败不抛给调用方：分组归属是辅助信息，
 * 拉不到时该列退化为「—」，不能让模型监控接口的一次抖动把整页配置也带崩。
 */
async function loadGroupContext(): Promise<void> {
  groupContextLoading.value = true
  try {
    groupContext.value = await adminAPI.modelSquare.loadGroupContext()
  } catch {
    groupContext.value = null
  } finally {
    groupContextLoading.value = false
  }
}

async function reload(): Promise<void> {
  loading.value = true
  loadError.value = ''
  try {
    const [config, customPlatformList, accountPage] = await Promise.all([
      adminAPI.modelSquareConfig.get(),
      adminAPI.customPlatforms.list(false),
      adminAPI.accounts.list(1, 500, { lite: 'true' }),
    ])
    customPlatforms.value = Array.isArray(customPlatformList) ? customPlatformList : []
    setCustomPlatformLabels(customPlatformList)
    normalizeConfig(config)
    accounts.value = Array.isArray(accountPage.items) ? accountPage.items : []
    if (!platformCards.value.some(item => item.platform === selectedPlatform.value)) {
      selectedPlatform.value = platformCards.value[0]?.platform || 'openai'
    }
    void loadCurrentPlatformReferencePricing()
    // 分组归属单独拉、不阻塞首屏：配置清单先出来，分组列随后补齐。
    void loadGroupContext()
  } catch (err) {
    loadError.value = extractApiErrorMessage(err, '加载模型广场配置失败')
    appStore.showError(loadError.value)
  } finally {
    loading.value = false
  }
}

async function loadSyncAccountsForCurrentPlatform(): Promise<void> {
  const platform = normalizePlatform(selectedPlatform.value)
  syncAccounts.value = []
  if (!platform) return

  syncAccountsLoading.value = true
  try {
    const candidates = await adminAPI.modelSquareConfig.listSyncAccounts(platform)
    if (normalizePlatform(selectedPlatform.value) === platform) {
      syncAccounts.value = Array.isArray(candidates) ? candidates : []
    }
  } catch (err) {
    if (normalizePlatform(selectedPlatform.value) === platform) {
      appStore.showError(extractApiErrorMessage(err, '加载可同步账号失败'))
    }
  } finally {
    if (normalizePlatform(selectedPlatform.value) === platform) {
      syncAccountsLoading.value = false
    }
  }
}

/**
 * 保存前先问一句：服务端的配置有没有在「我打开这一页之后」被别人改过。
 *
 * 为什么需要：后端 `UpdateModelSquareConfig` 只做 validate → normalize → 盖上新的
 * `updated_at` → 整块覆盖，**完全不比对入参里的 `updated_at`**（已核对源码）。
 * 于是两个人同时编辑时，后保存的会静默覆盖前一个人的改动，双方都以为存成功了。
 *
 * 这里只做「提交前重拉一次做比对」，不是完整的乐观锁：重拉与提交之间仍有 TOCTOU 窗口。
 * 选它是因为足以拦住最常见的场景（编辑十几分钟再保存），且**不需要改后端契约**；
 * 真要做严，得让后端比对 `updated_at` 并返回 409，那是另一件事。
 */
async function hasRemoteConflict(): Promise<boolean> {
  if (!configUpdatedAt.value) return false
  try {
    const remote = await adminAPI.modelSquareConfig.get()
    const remoteUpdatedAt = remote?.updated_at || null
    if (!remoteUpdatedAt) return false
    return remoteUpdatedAt !== configUpdatedAt.value
  } catch {
    // 探测失败不阻断保存：这只是保护措施，不该因为一次网络抖动就让人存不了配置。
    return false
  }
}

/**
 * 真正落库的那一步。从 saveConfig 里拆出来，是为了让「发现冲突后用户确认覆盖」
 * 能直接调它，而不必再走一次冲突探测（否则会陷进同一个弹窗里出不来）。
 */
async function persistConfig(): Promise<void> {
  try {
    const updated = await adminAPI.modelSquareConfig.update(buildSavePayload())
    normalizeConfig(updated)
    appStore.showSuccess('模型广场配置已保存')
  } catch (err) {
    appStore.showError(extractApiErrorMessage(err, '保存模型广场配置失败'))
  }
}

async function saveConfig(): Promise<void> {
  if (saving.value) return
  saving.value = true
  try {
    if (await hasRemoteConflict()) {
      conflictConfirmVisible.value = true
      return
    }
    await persistConfig()
  } finally {
    saving.value = false
  }
}

/** 用户在「配置已被他人修改」弹窗里选了「覆盖保存」。 */
async function confirmOverwriteConfig(): Promise<void> {
  conflictConfirmVisible.value = false
  saving.value = true
  try {
    await persistConfig()
  } finally {
    saving.value = false
  }
}

function fillModelFormFromOfficialPricing(pricing: ModelSquareOfficialPricing): void {
  for (const price of PRICE_FIELDS) {
    if (modelForm.value[price.key].trim()) continue
    const officialValue = pricing[price.key as keyof ModelSquareOfficialPricing]
    if (isOfficialReferencePriceValue(officialValue)) {
      modelForm.value[price.key] = storedPriceToDisplayPrice(officialValue)
    }
  }
}

function openModelDialog(model?: ModelSquarePlatformModelConfig): void {
  editingModelId.value = model?.id || null
  const officialPricing = model ? officialPricingForModel(model) : null
  modelForm.value = {
    id: model?.id || '',
    display_name: model?.display_name && model.display_name !== model.id ? model.display_name : '',
    group_ids: normalizeModelGroupIDs(model?.group_ids),
    ...Object.fromEntries(PRICE_FIELDS.map(({ key }) => {
      const configuredValue = model?.[key]
      const officialValue = officialPricing?.[key as keyof ModelSquareOfficialPricing]
      const value = configuredValue != null && Number.isFinite(configuredValue)
        ? configuredValue
        : isOfficialReferencePriceValue(officialValue)
          ? officialValue
          : null
      return [key, storedPriceToDisplayPrice(value)]
    })) as Pick<ModelForm, PriceField>,
    // 按张价格没有官方兜底，只从已配置值回填；缺省即留空。
    per_request_price: storedRequestPriceToDisplay(model?.per_request_price),
  }
  // 有按张价的既有模型按图/张类型打开，其余（含新建）默认按量。
  modelBillingMode.value = model?.per_request_price != null ? 'per_image' : 'token'
  modelDialogVisible.value = true

  if (model && !officialPricing && hasMissingConfiguredPrice(model)) {
    void waitForOfficialPricing(model).then(() => {
      if (!modelDialogVisible.value || modelKey(editingModelId.value || '') !== modelKey(model.id)) return
      if (modelKey(normalizeModelId(modelForm.value.id)) !== modelKey(model.id)) return
      const loadedPricing = officialPricingForModel(model)
      if (loadedPricing) fillModelFormFromOfficialPricing(loadedPricing)
    })
  }
}

function closeModelDialog(): void {
  modelDialogVisible.value = false
  editingModelId.value = null
  modelForm.value = createEmptyModelForm()
  modelBillingMode.value = 'token'
}

/*
  表格里「分组绑定」列的快捷入口。

  只改绑定、不碰价格：管理员在表格上就能看出哪个模型绑错了，不该为了改它打开整个编辑弹窗
  —— 那里同时挂着全部价格字段，误改一个就是一次静默的定价变更。
*/
const groupBindDialogVisible = ref(false)
const groupBindModelID = ref<string | null>(null)
const groupBindForm = ref<number[]>([])

const groupBindDialogTitle = computed(() => `绑定分组 · ${groupBindModelID.value || '模型'}`)

function openGroupBindDialog(model: ModelSquarePlatformModelConfig): void {
  groupBindModelID.value = model.id
  groupBindForm.value = normalizeModelGroupIDs(model.group_ids)
  groupBindDialogVisible.value = true
}

function closeGroupBindDialog(): void {
  groupBindDialogVisible.value = false
  groupBindModelID.value = null
  groupBindForm.value = []
}

function submitGroupBindDialog(): void {
  const config = ensureCurrentConfig()
  const index = config.models.findIndex(model => modelKey(model.id) === modelKey(groupBindModelID.value || ''))
  if (index < 0) {
    // 弹窗开着的时候模型被别处删掉了。静默不保存会让管理员以为绑定生效了。
    appStore.showError('模型已不存在，请刷新后重试')
    closeGroupBindDialog()
    return
  }
  config.models.splice(index, 1, {
    ...config.models[index],
    group_ids: normalizeModelGroupIDs(groupBindForm.value),
  })
  closeGroupBindDialog()
}

/*
  批量绑定。

  入口用表格勾选，而不是「按条件批量」：管理员判断哪个模型该绑哪个分组，靠的是看表格里的
  价格与渠道状态，勾选是唯一能把「他此刻看到的这一批」原样传下来的方式。

  勾选键只在当前平台内唯一（模型 ID 不带平台前缀），所以换平台必须清空勾选 ——
  否则 openai 的 gpt-5.2-pro 会把自定义平台里的同名模型一起带上。
*/
const selectedModelKeys = ref<string[]>([])
const batchBindDialogVisible = ref(false)
const batchBindForm = ref<number[]>([])
const clearGroupsConfirmVisible = ref(false)

/** 勾选、且当前平台里确实还存在的模型。键可能是模型被删掉后留下的残留，必须按配置过滤。 */
const selectedModels = computed(() => {
  const keys = new Set(selectedModelKeys.value.map(key => String(key)))
  return currentModels.value.filter(model => keys.has(String(model.id)))
})

/** 勾选框的无障碍名称。默认文案是「请选择 <id>」，读屏时分不清选的是哪个模型。 */
function modelSelectionLabel(row: ModelSquarePlatformModelConfig): string {
  return `选择模型 ${row.display_name || row.id}`
}

function clearModelSelection(): void {
  selectedModelKeys.value = []
}

/*
  合并分组绑定：追加去重后走共享口径规范化。
  空结果写 undefined 而不是 [] —— 后端那个字段是 omitempty，空数组与缺字段序列化形态不同，
  会让配置页的「未保存改动」误报（「解绑全部」看起来像「改过但没生效」）。
*/
function mergeGroupIDs(current: number[] | undefined, additions: number[]): number[] | undefined {
  const merged = normalizeModelGroupIDs([...(current || []), ...additions])
  return merged.length ? merged : undefined
}

/** 把当前平台下所有被勾选的模型改写一遍分组绑定，返回实际改动的模型数。 */
function updateSelectedModelGroups(
  resolve: (model: ModelSquarePlatformModelConfig) => number[] | undefined,
): number {
  const targets = selectedModels.value
  if (targets.length === 0) return 0
  const targetKeys = new Set(targets.map(model => modelKey(model.id)))
  const config = ensureCurrentConfig()
  config.models = config.models.map(model =>
    targetKeys.has(modelKey(model.id)) ? { ...model, group_ids: resolve(model) } : model
  )
  return targets.length
}

function openBatchBindDialog(): void {
  if (selectedModels.value.length === 0) return
  // 每次打开都从空开始：批量操作里「上次勾了哪些分组」是最不该被继承的状态。
  batchBindForm.value = []
  batchBindDialogVisible.value = true
}

function closeBatchBindDialog(): void {
  batchBindDialogVisible.value = false
  batchBindForm.value = []
}

function submitBatchBindDialog(): void {
  const additions = normalizeModelGroupIDs(batchBindForm.value)
  if (additions.length === 0) {
    appStore.showError('请至少选择一个要绑定的分组')
    return
  }
  const count = updateSelectedModelGroups(model => mergeGroupIDs(model.group_ids, additions))
  if (count === 0) {
    // 勾选的模型在弹窗开着的时候被别处删掉了。静默关掉会让管理员以为绑定生效了。
    appStore.showError('勾选的模型已不存在，请刷新后重试')
    closeBatchBindDialog()
    return
  }
  appStore.showSuccess(`已为 ${count} 个模型追加 ${additions.length} 个分组绑定`)
  closeBatchBindDialog()
  clearModelSelection()
}

function requestClearSelectedGroups(): void {
  if (selectedModels.value.length === 0) return
  clearGroupsConfirmVisible.value = true
}

function confirmClearSelectedGroups(): void {
  clearGroupsConfirmVisible.value = false
  const count = updateSelectedModelGroups(() => undefined)
  if (count === 0) return
  appStore.showSuccess(`已清空 ${count} 个模型的分组绑定`)
  clearModelSelection()
}

function parseModelFormPrices(): Pick<ModelSquarePlatformModelConfig, PriceField> | null {
  const prices = {} as Pick<ModelSquarePlatformModelConfig, PriceField>
  for (const price of PRICE_FIELDS) {
    const raw = modelForm.value[price.key].trim()
    if (!raw) {
      prices[price.key] = null
      continue
    }
    const value = Number(raw)
    if (!Number.isFinite(value) || value < 0) {
      appStore.showError(`${price.label}必须是非负数字`)
      return null
    }
    prices[price.key] = displayPriceToStoredPrice(value)
  }
  return prices
}

// 解析按张价格：空 -> null（清空 / 不按张计费），非负数 -> 原样存（不换算 1M）。
// 非法输入返回 'invalid'，调用方据此中止保存，避免把坏值写进配置。
function parseModelFormRequestPrice(): number | null | 'invalid' {
  const raw = modelForm.value.per_request_price.trim()
  if (!raw) return null
  const value = Number(raw)
  if (!Number.isFinite(value) || value < 0) {
    appStore.showError('按张价格必须是非负数字')
    return 'invalid'
  }
  return value
}

function isOfficialReferencePriceValue(value: unknown): value is number {
  return typeof value === 'number' && Number.isFinite(value) && value > 0
}

function officialPricingKey(id: string): string {
  return modelKey(id)
}

function officialPricingLookupCandidates(id: string): string[] {
  const normalized = normalizeModelId(id)
  if (!normalized) return []

  const candidates = [normalized]
  const lastSegment = normalized.split('/').pop()?.trim() || ''
  if (lastSegment && modelKey(lastSegment) !== modelKey(normalized)) candidates.push(lastSegment)
  return candidates
}

function officialPricingForModel(model: ModelSquarePlatformModelConfig): ModelSquareOfficialPricing | null {
  return officialPricingMap.value[officialPricingKey(model.id)] || null
}

function officialPricingStatusForModel(model: ModelSquarePlatformModelConfig): OfficialPricingStatus | null {
  return officialPricingStatusMap.value[officialPricingKey(model.id)] || null
}

function hasMissingConfiguredPrice(model: ModelSquarePlatformModelConfig): boolean {
  return PRICE_FIELDS.some(price => model[price.key] == null)
}

function shouldLoadOfficialPricing(model: ModelSquarePlatformModelConfig): boolean {
  const id = normalizeModelId(model.id)
  if (!id || !hasMissingConfiguredPrice(model)) return false
  const key = officialPricingKey(id)
  return !officialPricingMap.value[key] && !referencePricingInFlightKeys.has(key)
}

function waitForOfficialPricing(model: ModelSquarePlatformModelConfig): Promise<void> {
  const key = officialPricingKey(model.id)
  if (officialPricingMap.value[key] || !shouldLoadOfficialPricing(model)) {
    return referencePricingPromises.get(key) || Promise.resolve()
  }
  return loadOfficialPricingForModels([model])
}

async function loadOfficialPricingForModels(models: ModelSquarePlatformModelConfig[]): Promise<void> {
  const requestModels = models
    .map(model => ({ ...model, id: normalizeModelId(model.id) }))
    .filter(model => shouldLoadOfficialPricing(model))
  if (requestModels.length === 0) return

  const requestKeys = requestModels.map(model => officialPricingKey(model.id))
  for (const key of requestKeys) referencePricingInFlightKeys.add(key)
  officialPricingStatusMap.value = {
    ...officialPricingStatusMap.value,
    ...Object.fromEntries(requestKeys.map(key => [key, 'loading' as OfficialPricingStatus])),
  }
  referencePricingLoadingCount.value += 1

  const pricingPromise = (async () => {
    try {
      const pricingResults = await Promise.all(requestModels.map(async model => {
        try {
          let lastPricing: ModelSquareOfficialPricing | null = null
          for (const candidate of officialPricingLookupCandidates(model.id)) {
            const pricing = await adminAPI.modelSquareConfig.getModelPricing(candidate)
            lastPricing = pricing
            if (pricing.found) return { id: model.id, pricing, failed: false }
          }
          return { id: model.id, pricing: lastPricing, failed: false }
        } catch {
          return { id: model.id, pricing: null, failed: true }
        }
      }))

      const nextPricingMap = { ...officialPricingMap.value }
      const nextStatusMap = { ...officialPricingStatusMap.value }
      for (const { id, pricing, failed } of pricingResults) {
        const key = officialPricingKey(id)
        if (pricing?.found) {
          nextPricingMap[key] = pricing
          nextStatusMap[key] = 'found'
        } else {
          delete nextPricingMap[key]
          nextStatusMap[key] = failed ? 'error' : 'not_found'
        }
      }
      officialPricingMap.value = nextPricingMap
      officialPricingStatusMap.value = nextStatusMap
    } finally {
      for (const key of requestKeys) {
        referencePricingInFlightKeys.delete(key)
        referencePricingPromises.delete(key)
      }
      referencePricingLoadingCount.value = Math.max(0, referencePricingLoadingCount.value - 1)
    }
  })()
  for (const key of requestKeys) referencePricingPromises.set(key, pricingPromise)
  await pricingPromise
}

async function loadCurrentPlatformReferencePricing(): Promise<void> {
  const config = platformConfigs.value.find(item => normalizePlatform(item.platform) === selectedPlatform.value)
  if (!config) return
  await loadOfficialPricingForModels(config.models)
}

async function applyDefaultPricing(): Promise<void> {
  const id = normalizeModelId(modelForm.value.id)
  if (!id) {
    appStore.showError('请先填写模型 ID')
    return
  }
  defaultPricingLoading.value = true
  try {
    const pricing = await adminAPI.modelSquareConfig.getModelPricing(id)
    if (!pricing.found) {
      appStore.showError('未找到该模型的默认价格')
      return
    }
    let filled = 0
    for (const price of PRICE_FIELDS) {
      const current = modelForm.value[price.key].trim()
      const defaultValue = pricing[price.key as keyof ModelSquareOfficialPricing]
      if (!current && isOfficialReferencePriceValue(defaultValue)) {
        modelForm.value[price.key] = storedPriceToDisplayPrice(defaultValue)
        filled += 1
      }
    }
    appStore.showSuccess(filled > 0 ? `已回填 ${filled} 项默认价格` : '当前价格字段已全部填写')
  } catch (err) {
    appStore.showError(extractApiErrorMessage(err, '获取默认价格失败'))
  } finally {
    defaultPricingLoading.value = false
  }
}

function submitModelDialog(): void {
  const id = normalizeModelId(modelForm.value.id)
  if (!id) {
    appStore.showError('请填写模型 ID')
    return
  }
  const config = ensureCurrentConfig()
  const duplicate = config.models.find(model => modelKey(model.id) === modelKey(id) && modelKey(model.id) !== modelKey(editingModelId.value || ''))
  if (duplicate) {
    appStore.showError('当前平台已存在相同模型')
    return
  }
  const displayName = normalizeModelId(modelForm.value.display_name) || id
  const isPerImage = modelBillingMode.value === 'per_image'
  // 只解析当前计费类型对应的那套价格，另一套在下面统一清掉。
  let tokenPrices: Pick<ModelSquarePlatformModelConfig, PriceField> | null = null
  let perRequestPrice: number | null = null
  if (isPerImage) {
    const parsed = parseModelFormRequestPrice()
    if (parsed === 'invalid') return
    perRequestPrice = parsed
  } else {
    tokenPrices = parseModelFormPrices()
    if (!tokenPrices) return
  }
  const index = config.models.findIndex(model => modelKey(model.id) === modelKey(editingModelId.value || ''))
  // 编辑已有模型时从原对象出发、只覆盖本次改过的字段（理由同 dedupeModels）。
  // 新建时没有原对象，展开 undefined 得到空对象，行为与从前一致。
  const nextModel: ModelSquarePlatformModelConfig = {
    ...(index >= 0 ? config.models[index] : undefined),
    id,
    display_name: displayName,
    source: 'manual',
    group_ids: normalizeModelGroupIDs(modelForm.value.group_ids),
  }
  // 两种计费类型互斥：写入当前类型的价格，并清掉另一套本页字段，避免模型同时带着 token 价与按张价。
  // 清空用 delete（而非写 null）：整块 PUT 后端按缺省处理，也不给模型平白塞一个 null 字段。
  if (isPerImage) {
    if (perRequestPrice !== null) nextModel.per_request_price = perRequestPrice
    else delete nextModel.per_request_price
    for (const { key } of PRICE_FIELDS) delete nextModel[key]
  } else {
    Object.assign(nextModel, tokenPrices)
    delete nextModel.per_request_price
  }
  if (index >= 0) config.models.splice(index, 1, nextModel)
  else config.models.push(nextModel)
  closeModelDialog()
  void loadCurrentPlatformReferencePricing()
}

function openBatchDialog(): void {
  batchText.value = ''
  batchDialogVisible.value = true
}

function closeBatchDialog(): void {
  batchDialogVisible.value = false
  batchText.value = ''
}

function submitBatchDialog(): void {
  const ids = Array.from(new Set(batchText.value.split(/[\s,，]+/).map(normalizeModelId).filter(Boolean)))
  if (ids.length === 0) {
    appStore.showError('请至少填写一个模型 ID')
    return
  }
  const config = ensureCurrentConfig()
  const existing = new Set(config.models.map(model => modelKey(model.id)))
  let added = 0
  for (const id of ids) {
    if (existing.has(modelKey(id))) continue
    config.models.push({ id, display_name: id, source: 'manual', ...modelPriceValues({ id }) })
    existing.add(modelKey(id))
    added += 1
  }
  appStore.showSuccess(`已添加 ${added} 个模型`)
  closeBatchDialog()
  void loadCurrentPlatformReferencePricing()
}

function openSyncDialog(): void {
  syncAccountId.value = null
  syncDialogVisible.value = true
  void loadSyncAccountsForCurrentPlatform()
}

function closeSyncDialog(): void {
  if (syncing.value) return
  syncDialogVisible.value = false
  syncAccountId.value = null
  syncAccounts.value = []
}

async function submitSyncDialog(): Promise<void> {
  if (!syncAccountId.value) {
    appStore.showError('请选择要同步的账号')
    return
  }
  const account = syncAccounts.value.find(item => item.id === syncAccountId.value)
  syncing.value = true
  try {
    const result = await adminAPI.accounts.syncUpstreamModels(syncAccountId.value)
    const models = Array.isArray(result.models) ? result.models : []
    const config = ensureCurrentConfig()
    const existing = new Set(config.models.map(model => modelKey(model.id)))
    const addedModelIds: string[] = []
    let added = 0
    for (const raw of models) {
      const id = normalizeModelId(raw)
      if (!id || existing.has(modelKey(id))) continue
      config.models.push({ id, display_name: id, source: 'sync', ...modelPriceValues({ id }) })
      existing.add(modelKey(id))
      addedModelIds.push(id)
      added += 1
    }
    await loadOfficialPricingForModels(addedModelIds.map(id => ({ id, source: 'sync', ...modelPriceValues({ id }) })))
    config.synced_from_account_id = syncAccountId.value
    config.synced_from_account_name = account?.name || ''
    config.synced_at = new Date().toISOString()
    appStore.showSuccess(`同步完成，新增 ${added} 个模型`)
    closeSyncDialog()
  } catch (err) {
    appStore.showError(extractApiErrorMessage(err, '同步上游模型失败'))
  } finally {
    syncing.value = false
  }
}

function askRemoveModel(model: ModelSquarePlatformModelConfig): void {
  modelPendingRemove.value = model
}

function confirmRemoveModel(): void {
  if (!modelPendingRemove.value) return
  const config = ensureCurrentConfig()
  config.models = config.models.filter(model => modelKey(model.id) !== modelKey(modelPendingRemove.value?.id || ''))
  modelPendingRemove.value = null
}

function formatPriceSlotValue(value: number): string {
  if (value == null || !Number.isFinite(value)) return ''
  return `$${formatPlainPriceNumber(value * PRICE_PER_MILLION_TOKENS)}`
}

// 按张价格原样展示，不做 1M 换算。
function formatRawPriceSlotValue(value: number): string {
  if (value == null || !Number.isFinite(value)) return ''
  return `$${formatPlainPriceNumber(value)}`
}

function pricingValue(model: ModelSquarePlatformModelConfig, officialPricing: ModelSquareOfficialPricing | null, key: PriceField): { value: number; source: PriceSource } | null {
  const configuredValue = model[key]
  if (configuredValue != null && Number.isFinite(configuredValue)) {
    return { value: configuredValue, source: 'configured' }
  }
  const officialValue = officialPricing?.[key as keyof ModelSquareOfficialPricing]
  if (isOfficialReferencePriceValue(officialValue)) {
    return { value: officialValue, source: 'official' }
  }
  return null
}

/*
  价格位描述表。顺序、标签、单位、色调全部照展示页列表视图（defaultPriceDescriptors）：
  两个页面都要显示这四个价格，顺序或文案差一点，管理员换页面比对时就会读错位。

  单位只给输入/输出：展示页就是这么定的（缓存价是输入价的折算，再写一遍单位是重复）。
  配置页的列头已经写了「每 1M Tokens」，但照抄展示页的卡片语言更重要 ——
  少一层「这个页面为什么和那个页面不一样」的困惑。
*/
const PRICE_SLOT_DESCRIPTORS: Array<{ key: PriceField; label: string; unit: string; tone: PriceTone }> = [
  { key: 'input_price', label: '输入', unit: '$/百万 tokens', tone: 'teal' },
  { key: 'output_price', label: '输出', unit: '$/百万 tokens', tone: 'orange' },
  { key: 'cache_read_price', label: '缓存读取', unit: '', tone: 'blue' },
  { key: 'cache_write_price', label: '缓存写入', unit: '', tone: 'violet' },
]

// 按张计费的模型（配了 per_request_price）在列表里只显示一张「按张」卡片，
// 不摆 token 价格位 —— 它没有 token 价，硬摆会回退成官方参考价，读成「按 token 计费」就错了。
function modelIsPerImage(model: ModelSquarePlatformModelConfig): boolean {
  return model.per_request_price != null && Number.isFinite(model.per_request_price)
}

function modelPriceSlots(model: ModelSquarePlatformModelConfig): PriceSlot[] {
  if (modelIsPerImage(model)) {
    return [{
      key: 'per_request_price',
      label: '按张',
      unit: '$/张',
      toneClass: 'price-card-teal',
      value: model.per_request_price as number,
      source: 'configured',
      raw: true,
    }]
  }
  const officialPricing = officialPricingForModel(model)
  return PRICE_SLOT_DESCRIPTORS.map(descriptor => {
    const price = pricingValue(model, officialPricing, descriptor.key)
    return {
      key: descriptor.key,
      label: descriptor.label,
      unit: descriptor.unit,
      // 缺价是虚线空框，与「有价」必须一眼分开。但色调一律保留：
      // 官方参考价由卡片里那枚「官方参考」标签标记，不再整张卡片换色 ——
      // 换色会让四个价格位失去各自的色调，这一列就又变回一片同色了。
      toneClass: price ? `price-card-${descriptor.tone}` : 'price-card-unset',
      value: price?.value ?? null,
      source: price?.source ?? null,
    }
  })
}

// 四个价格位全无价时不摆四张空卡片 —— 那时该说的是「为什么没有价」（查询中/目录无价/查询失败），
// 四张破折号会把一条可操作的状态信息压成装饰。
function modelHasAnyPrice(model: ModelSquarePlatformModelConfig): boolean {
  return modelPriceSlots(model).some(slot => slot.value != null)
}

function modelPriceEmptyText(model: ModelSquarePlatformModelConfig): string {
  switch (officialPricingStatusForModel(model)) {
    case 'loading':
      return '正在查询官方参考价'
    case 'not_found':
      return '官方目录无价格'
    case 'error':
      return '官方价格查询失败'
    default:
      return '未设置'
  }
}

// ---------------------------------------------------------------------------
// 弹窗内的基准价对照
//
// 弹窗里原本看不到官方基准价：表格有「官方参考」徽标，弹窗没有，填数字时既不知道
// 官方是多少、也不知道自己填的是高还是低。下面这组取数专门服务弹窗。
// 单位说明：官方价和表单值都按 USD / 1M Tokens 参与比较，比值因此可以直接算。
// ---------------------------------------------------------------------------

const modelDialogOfficialPricing = computed<ModelSquareOfficialPricing | null>(
  () => officialPricingMap.value[officialPricingKey(modelForm.value.id)] || null,
)
const modelDialogPricingStatus = computed<OfficialPricingStatus | null>(
  () => officialPricingStatusMap.value[officialPricingKey(modelForm.value.id)] || null,
)

/** 官方基准价（USD / 1M Tokens）。没有可用的官方参考价时返回 null。 */
function modelOfficialDisplayPrice(key: PriceField): number | null {
  const value = modelDialogOfficialPricing.value?.[key as keyof ModelSquareOfficialPricing]
  if (!isOfficialReferencePriceValue(value)) return null
  return value * PRICE_PER_MILLION_TOKENS
}

function formatBaselinePrice(value: number): string {
  return `$${formatPlainPriceNumber(value)}`
}

/** 正在编辑的模型在已保存配置里的原对象；新建模型时为 null。 */
const editingOriginalModel = computed<ModelSquarePlatformModelConfig | null>(() => {
  const key = modelKey(editingModelId.value || '')
  if (!key) return null
  const platform = normalizePlatform(selectedPlatform.value)
  const config = platformConfigs.value.find(item => normalizePlatform(item.platform) === platform)
  return config?.models.find(model => modelKey(model.id) === key) || null
})

/**
 * 字段状态。
 *
 * ⚠️ 不能只看表单值：openModelDialog 会把空字段预填成官方参考价
 * （`configuredValue ?? officialValue`），所以打开弹窗后「跟随官方」的字段
 * 表单里也是有值的 —— 只按表单非空判断会把它们全部误报成「已自定义」。
 *
 * 因此以「已保存配置」为准；只有表单值被改成和官方基准不一样时，才算本次自定义。
 */
function modelPriceFieldState(key: PriceField): ModelPriceFieldState {
  const saved = editingOriginalModel.value?.[key]
  if (saved != null && Number.isFinite(saved)) return 'custom'

  const officialDisplay = modelOfficialDisplayPrice(key)
  const raw = modelForm.value[key].trim()

  if (officialDisplay == null) {
    // 官方没有基准价可跟随：填了才算自定义，没填就是未设置。
    return raw ? 'custom' : 'unset'
  }
  if (!raw) return 'official'
  const value = Number(raw)
  if (!Number.isFinite(value)) return 'official'
  return Math.abs(value - officialDisplay) < 1e-9 ? 'official' : 'custom'
}

/** 当前填写的价格相对官方基准的倍数；任一侧缺失时返回 null。 */
function modelPriceRatio(key: PriceField): number | null {
  const raw = modelForm.value[key].trim()
  if (!raw) return null
  const value = Number(raw)
  if (!Number.isFinite(value) || value < 0) return null
  const base = modelOfficialDisplayPrice(key)
  if (base == null || base <= 0) return null
  return value / base
}

function modelPricePlaceholder(key: PriceField): string {
  const base = modelOfficialDisplayPrice(key)
  return base == null ? '例如：5' : `留空使用官方 ${formatPlainPriceNumber(base)}`
}

/** 字段下方的基准价说明。官方价取不到时要分清「查不到」和「还没查」，否则会误判成目录里没有。 */
function modelPriceBaselineText(key: PriceField): string {
  const base = modelOfficialDisplayPrice(key)
  if (base == null) {
    switch (modelDialogPricingStatus.value) {
      case 'loading':
        return '正在查询官方参考价'
      case 'error':
        return '官方价格查询失败，可点「获取默认价格」重试'
      case 'not_found':
        return '官方目录无参考价，留空则展示页不显示该价格'
      default:
        return '填写模型 ID 后可查询官方基准价'
    }
  }
  const baseText = formatBaselinePrice(base)
  // 只有「自己定的价」才谈得上倍数。跟随官方的字段表单里是预填的官方值，
  // 再显示一个 ×1.00 / 与官方一致 只会和旁边的徽标重复。
  if (modelPriceFieldState(key) !== 'custom') return `官方基准 ${baseText}`
  const ratio = modelPriceRatio(key)
  if (ratio == null) return `官方基准 ${baseText}`
  if (Math.abs(ratio - 1) < 1e-9) return `官方基准 ${baseText} · 与官方一致`
  return `官方基准 ${baseText} · 你的价 ×${ratio.toFixed(2)}`
}

function modelPriceUnsetReason(): string {
  switch (modelDialogPricingStatus.value) {
    case 'loading':
      return '官方参考价查询中'
    case 'error':
      return '官方价格查询失败'
    case 'not_found':
      return '官方目录无参考价'
    default:
      return '暂无官方参考价'
  }
}

const modelConfiguredPriceCount = computed(
  () => PRICE_FIELDS.filter(price => modelPriceFieldState(price.key) === 'custom').length,
)

const modelBaselineRows = computed(() => PRICE_FIELDS.map(price => {
  const key = price.key
  const base = modelOfficialDisplayPrice(key)
  const state = modelPriceFieldState(key)

  if (state === 'custom') {
    const ratio = modelPriceRatio(key)
    const tone = ratio == null || Math.abs(ratio - 1) < 1e-9 ? 'is-same' : ratio > 1 ? 'is-high' : 'is-low'
    return {
      key,
      label: price.label,
      tone,
      valueText: modelForm.value[key].trim(),
      valueTone: 'is-custom',
      meta: base == null ? modelPriceUnsetReason() : `官方 ${formatBaselinePrice(base)}`,
      badge: ratio == null ? '' : `×${ratio.toFixed(2)}`,
    }
  }

  if (state === 'official' && base != null) {
    return {
      key,
      label: price.label,
      tone: 'is-follow',
      valueText: formatBaselinePrice(base),
      valueTone: 'is-inherited',
      meta: '跟随官方 · 未自定义',
      badge: '',
    }
  }

  return {
    key,
    label: price.label,
    tone: 'is-unset',
    valueText: '未设置',
    valueTone: 'is-unset',
    meta: modelPriceUnsetReason(),
    badge: '',
  }
}))

function formatTime(value?: string | null): string {
  if (!value) return '未保存'
  const date = new Date(value)
  if (Number.isNaN(date.getTime())) return '未保存'
  return date.toLocaleString('zh-CN', { hour12: false })
}

watch(selectedPlatform, () => {
  searchQuery.value = ''
  syncAccountId.value = null
  syncAccounts.value = []
  // 勾选键只在平台内唯一，换平台后旧键会指向另一个平台的同名模型 —— 必须清空。
  clearModelSelection()
  if (syncDialogVisible.value) {
    void loadSyncAccountsForCurrentPlatform()
  }
  void loadCurrentPlatformReferencePricing()
})

/**
 * 「刷新」会重新拉取配置并整体覆盖 platformConfigs，未保存的编辑会被静默丢掉。
 * 有未保存改动时先问一句 —— 这是最容易误触的一条路径（刷新按钮就在保存按钮左边）。
 */
function requestReload(): void {
  if (isDirty.value) {
    reloadConfirmVisible.value = true
    return
  }
  void reload()
}

function confirmReload(): void {
  reloadConfirmVisible.value = false
  void reload()
}

/**
 * 浏览器刷新 / 关标签页。这里只能用原生提示（浏览器不允许自定义样式），
 * 但它是最后一道防线：未保存的改动只在内存里，页面一走就没了。
 */
function handleBeforeUnload(event: BeforeUnloadEvent): void {
  if (!isDirty.value) return
  event.preventDefault()
  // 部分浏览器只认 returnValue，不写就不会弹提示。
  event.returnValue = ''
}

/**
 * 站内路由跳转（点侧边栏）。这里用 window.confirm 而不是页面里的 ConfirmDialog：
 * 守卫在导航中途触发，没有组件上下文可以承载弹窗；项目里 BackupView / GroupsView /
 * PluginsView 处理同类确认用的也是 window.confirm。
 */
onBeforeRouteLeave(() => {
  if (!isDirty.value) return true
  return window.confirm('模型广场配置有未保存的改动，离开后这些改动会丢失。确认离开吗？')
})

onMounted(() => {
  window.addEventListener('beforeunload', handleBeforeUnload)
  void reload()
})

onUnmounted(() => {
  window.removeEventListener('beforeunload', handleBeforeUnload)
})
</script>

<style scoped>
/* 有未保存改动时给保存按钮加一圈提示环：管理员的视线本来就在按钮上，比在页面上另写一行更管用。 */
.btn.is-dirty {
  @apply ring-2 ring-amber-400/70 dark:ring-amber-400/60;
}

.save-dirty-dot {
  @apply ml-1 inline-block h-1.5 w-1.5 shrink-0 rounded-full bg-white/90;
}

/* .metric-card 与 .hero-* 随 hero 横栏一并删除。sync-meta-card 是同步弹窗的 meta 卡，与 hero 无关，保留。 */
.sync-meta-card span {
  @apply text-xs font-medium text-gray-500 dark:text-dark-400;
}

/* 「未绑定 N」告警：跟随平台平铺标签，一眼看出问题在哪个平台。 */
.platform-option-warn {
  @apply shrink-0 rounded-full bg-amber-100 px-2 py-0.5 text-[11px] font-medium text-amber-800 dark:bg-amber-500/20 dark:text-amber-100;
}

/* 平台平铺标签。基础排版走 @apply，配色交给下面的 chip-tinted / chip-tinted-active。 */
.platform-chip {
  @apply inline-flex min-w-0 items-center gap-1.5 rounded-lg px-3 py-1.5 text-sm font-medium transition;
}

/* 标签上的模型数徽标：非激活态用平台淡色调，激活态在实心底色上转成半透明白。 */
.platform-chip-count {
  @apply shrink-0 rounded-full px-1.5 text-[11px] font-semibold tabular-nums;
  background-color: color-mix(in srgb, var(--chip-accent) 16%, transparent);
}

.chip-tinted-active .platform-chip-count {
  @apply text-white;
  background-color: rgb(255 255 255 / 0.22);
}

/* chip 配色统一从 --chip-accent（平台主色）经 color-mix 派生，新增平台无需扩样式；
   激活态与非激活态在模板上互斥挂载，避免选择器优先级互相覆盖。
   与展示页 PlazaFilterBar 同一套取色口径。 */
.chip-tinted {
  color: color-mix(in srgb, var(--chip-accent) 78%, black);
  background-color: color-mix(in srgb, var(--chip-accent) 9%, transparent);
  box-shadow: inset 0 0 0 1px color-mix(in srgb, var(--chip-accent) 25%, transparent);
}

.chip-tinted:hover {
  background-color: color-mix(in srgb, var(--chip-accent) 16%, transparent);
}

.dark .chip-tinted {
  color: color-mix(in srgb, var(--chip-accent) 72%, white);
  background-color: color-mix(in srgb, var(--chip-accent) 12%, transparent);
  box-shadow: inset 0 0 0 1px color-mix(in srgb, var(--chip-accent) 30%, transparent);
}

.dark .chip-tinted:hover {
  background-color: color-mix(in srgb, var(--chip-accent) 18%, transparent);
}

.chip-tinted-active {
  color: #fff;
  background-color: color-mix(in srgb, var(--chip-accent) 85%, black);
  box-shadow: 0 1px 2px 0 color-mix(in srgb, var(--chip-accent) 35%, transparent);
}

.chip-tinted-active:hover {
  background-color: color-mix(in srgb, var(--chip-accent) 75%, black);
}

.dark .chip-tinted-active {
  background-color: color-mix(in srgb, var(--chip-accent) 80%, transparent);
}

.dark .chip-tinted-active:hover {
  background-color: var(--chip-accent);
}

.reference-pricing-status {
  @apply inline-flex items-center gap-2 rounded-xl border border-sky-200 bg-sky-50 px-3 py-2 text-xs font-medium text-sky-700 dark:border-sky-500/30 dark:bg-sky-500/10 dark:text-sky-200;
}

/*
  模型 ID 胶囊。

  用 indigo 是因为这一页的颜色都被占满了：sky = 官方 / 上游同步，emerald = 手动维护，
  amber = 自定义，teal·orange·blue·violet = 四个价格位。indigo 是唯一没被赋予语义的色，
  拿来做标识色不会和任何既有含义撞车 —— 换成其它色就会被读成「这个模型有某种状态」。

  加粗：这一列和展示名称是管理员找模型时第一眼扫的地方，弱化它等于让人先去看价格。
*/
.model-code {
  @apply inline-flex max-w-md rounded-lg border border-indigo-200 bg-indigo-50 px-2 py-1 font-mono text-xs font-semibold text-indigo-800 dark:border-indigo-500/30 dark:bg-indigo-500/15 dark:text-indigo-200;
}

/*
  价格列卡片组：配色、尺寸、间距全部照展示页（ModelSquareView.vue）列表视图的价格卡片，
  两页必须长得一样 —— 管理员在这两个页面之间来回切，价格不能换一副样子。

  色值写成变量 + color-mix，与展示页的 --ms-* 是同一组数值、同一个混色公式：
  底色 = 12% 色调 + 88% 面板色，边框 = 40% 色调 + 60% 分隔线色。
  这两个结果落在 Tailwind 两个色阶之间，拿现成色阶凑不出来，只能自己混。
  没提成公共文件是因为展示页那套变量是它整个页面的主题（卡片、chip、弹窗都在用），
  为四个价格卡片去动它不值得。
*/
.price-cards {
  --pc-panel: #ffffff;
  --pc-panel-2: #f9fafb;
  --pc-line: #e5e7eb;
  --pc-text: #111827;
  --pc-muted: #64748b;
  --pc-dim: #94a3b8;
  --pc-teal: #0f766e;
  --pc-teal-strong: #115e59;
  --pc-orange: #ea580c;
  --pc-blue: #1d4ed8;
  --pc-violet: #6d28d9;

  /*
    width: max-content 不能省。grid 是块级的，默认会撑满整个单元格，
    于是四格均分「列宽减间距」，列被别的列挤宽时卡片就跟着变胖 ——
    实测撑到 103px，而展示页那张是 83px，同一套样式两个页面却不一样大。
    收缩到内容宽度后卡片宽度只由内容决定，与展示页一致。
  */
  @apply grid grid-cols-4 gap-2;
  width: max-content;
}

.dark .price-cards {
  --pc-panel: #1e293b;
  --pc-panel-2: #334155;
  --pc-line: #334155;
  --pc-text: #f9fafb;
  --pc-muted: #9ca3af;
  --pc-dim: #6b7280;
  --pc-teal: #5eead4;
  --pc-teal-strong: #2dd4bf;
  --pc-orange: #fb923c;
  --pc-blue: #60a5fa;
  --pc-violet: #a78bfa;
}

.price-card {
  /*
    min-width 83px = 展示页那张卡片的宽度（列头 + 单位行撑出来的自然宽度）。
    必须写死：四个价格全吃官方价的行不显示单位，内容只有标签那 52px，
    卡片会缩到 70px —— 同一列里不同行大小不一，这一列本来就是用来横向比对的。
    超过 83px 的情况（比如某个价格位数特别长）照常撑开，min-width 只兜下限。
  */
  @apply flex min-w-[83px] flex-col rounded-md border px-2 py-1.5;
  border-color: var(--pc-line);
  background: var(--pc-panel-2);
}

.price-card-teal {
  border-color: color-mix(in srgb, var(--pc-teal) 40%, var(--pc-line));
  background: color-mix(in srgb, var(--pc-teal) 12%, var(--pc-panel));
}

.price-card-orange {
  border-color: color-mix(in srgb, var(--pc-orange) 40%, var(--pc-line));
  background: color-mix(in srgb, var(--pc-orange) 12%, var(--pc-panel));
}

.price-card-blue {
  border-color: color-mix(in srgb, var(--pc-blue) 40%, var(--pc-line));
  background: color-mix(in srgb, var(--pc-blue) 12%, var(--pc-panel));
}

.price-card-violet {
  border-color: color-mix(in srgb, var(--pc-violet) 40%, var(--pc-line));
  background: color-mix(in srgb, var(--pc-violet) 12%, var(--pc-panel));
}

/* 未设置：虚线空框。与「有价」靠边框样式区分而不是靠颜色 —— 颜色已经被四个价格位占满了。 */
.price-card-unset {
  border-style: dashed;
  border-color: var(--pc-line);
  background: var(--pc-panel-2);
}

.price-card-label {
  @apply whitespace-nowrap text-xs font-medium;
  color: var(--pc-muted);
}

.price-card-teal .price-card-label {
  color: var(--pc-teal);
}

.price-card-orange .price-card-label {
  color: var(--pc-orange);
}

.price-card-blue .price-card-label {
  color: var(--pc-blue);
}

.price-card-violet .price-card-label {
  color: var(--pc-violet);
}

.price-card-value {
  @apply mt-1 whitespace-nowrap font-mono text-[13px] font-bold;
  color: var(--pc-text);
}

.price-card-teal .price-card-value {
  color: var(--pc-teal-strong);
}

.price-card-orange .price-card-value {
  color: var(--pc-orange);
}

.price-card-blue .price-card-value {
  color: var(--pc-blue);
}

.price-card-violet .price-card-value {
  color: var(--pc-violet);
}

/* 破折号是占位，但仍要比标签深一档：四种卡片共用「标签 < 数字」这条层级规则。 */
.price-card-unset .price-card-value {
  color: var(--pc-dim);
}

.price-card-meta {
  @apply mt-0.5;
}

.price-card-meta small {
  @apply block whitespace-nowrap text-[10px] font-medium;
  color: var(--pc-dim);
}

/*
  「官方参考」标签：这一列里唯一标价格来源的地方。
  原先它挂在「基础 / 缓存」分组框的末尾，一个组里两个价格有一个是参考价时读不出是哪个，
  现在跟着卡片走。天蓝是这个页面里「跟随官方」的既定语言（编辑弹窗的 is-official 徽标、
  基准价对照的 is-follow 行），所以沿用。
*/
.price-card-tag {
  @apply whitespace-nowrap rounded-full bg-sky-100 px-1.5 text-[10px] font-medium leading-4 text-sky-700 dark:bg-sky-500/20 dark:text-sky-200;
}

.price-empty {
  @apply text-xs text-gray-400 dark:text-dark-500;
}

/*
  批量操作条：只在有勾选时出现，所以不用做成常驻工具条。

  为什么不复用 AccountBulkActionsBar：那个组件把 `admin.accounts.bulkActions.*` 文案与
  账号专属事件（refresh-token / toggle-schedulable / probe-upstream-billing）写死在里面，
  模型广场套不进去 —— 硬套的结果是出现一批用不上的按钮和错位的 i18n key。
*/
.batch-bar {
  @apply mx-4 mt-4 flex flex-wrap items-center justify-between gap-2 rounded-xl border border-primary-200 bg-primary-50/60 px-4 py-2 dark:border-primary-700/60 dark:bg-primary-900/20;
}

.batch-bar-count {
  @apply text-sm font-medium text-primary-900 dark:text-primary-100;
}

.batch-bar-actions {
  @apply flex flex-wrap items-center gap-2;
}

/* 批量绑定弹窗里的作用对象说明：不写清「改的是哪一批」，批量操作就是盲操作。 */
.batch-bind-targets {
  @apply mb-2 text-sm text-gray-600 dark:text-gray-300;
}

/*
  分组绑定列：整格是一个按钮（表格里的快捷编辑入口），但外观保持原来的纯文本 + chip，
  只在 hover 时才浮出边框和底色 —— 每行都挂一个可见按钮会把这一列变成视觉噪音。
*/
.group-binding {
  @apply flex max-w-md flex-wrap items-center gap-1 rounded-lg border border-transparent px-1 py-0.5 text-left transition-colors hover:border-gray-200 hover:bg-gray-50 dark:hover:border-dark-600 dark:hover:bg-dark-700;
}

/*
  chip 只给形状和边框宽度，背景/文字/边框色由 groupPlatformBadgeClass 按分组平台给出
  （与弹窗里的绑定控件同一套取色）。这里写死颜色会盖掉平台色。
*/
.group-chip {
  @apply inline-flex max-w-40 items-center truncate rounded-lg border px-2 py-0.5 text-xs;
}

.group-chip-more {
  @apply inline-flex items-center rounded-lg bg-gray-100 px-2 py-0.5 text-xs text-gray-600 dark:bg-dark-800 dark:text-dark-300;
}

/* 未绑定分组：琥珀是这一列唯一需要动作的状态 —— 展示页按任何分组都筛不到它。 */
.group-binding-empty {
  @apply inline-flex items-center rounded-lg bg-amber-100 px-2 py-0.5 text-xs font-medium text-amber-800 dark:bg-amber-500/20 dark:text-amber-100;
}

.group-binding-pending {
  @apply text-xs text-gray-400 dark:text-dark-500;
}

.source-badge {
  @apply inline-flex rounded-full px-2.5 py-1 text-xs font-medium;
}

.source-sync {
  @apply bg-sky-500/10 text-sky-700 dark:bg-sky-500/15 dark:text-sky-200;
}

.source-manual {
  @apply bg-emerald-500/10 text-emerald-700 dark:bg-emerald-500/15 dark:text-emerald-200;
}

.sync-meta-card {
  @apply rounded-xl border border-gray-200 bg-gray-50 px-4 py-3 dark:border-dark-700 dark:bg-dark-900/60;
}

.sync-meta-card strong {
  @apply mt-1 block truncate text-sm font-semibold text-gray-950 dark:text-white;
}

/* --- 编辑模型弹窗 ---
   弹窗内容会 Teleport 到 body，所以这里刻意只用 Tailwind 工具类（含 dark: 变体），
   不引用页面根节点上的 CSS 变量 —— 那些变量在弹窗里取不到，会表现为「改了颜色但没生效」。 */

.model-dialog-grid {
  @apply grid grid-cols-1 gap-5 lg:grid-cols-[minmax(0,1fr)_19rem];
}

/* 图/张类型没有基准价对照 aside，收成单列，别留一条 19rem 的空槽。 */
.model-dialog-grid.is-single {
  @apply lg:grid-cols-1;
}

.model-billing-mode {
  @apply flex flex-wrap items-center gap-3;
}

.model-billing-mode-label {
  @apply text-sm font-medium text-gray-700 dark:text-dark-200;
}

.model-billing-mode-options {
  @apply inline-flex rounded-lg border border-gray-200 bg-gray-50 p-0.5 dark:border-dark-700 dark:bg-dark-900/50;
}

.model-billing-mode-btn {
  @apply rounded-md px-3 py-1.5 text-sm font-medium text-gray-500 transition-colors hover:text-gray-800 dark:text-dark-400 dark:hover:text-dark-100;
}

.model-billing-mode-btn.is-active {
  @apply bg-white text-primary-600 shadow-sm dark:bg-dark-700 dark:text-primary-300;
}

.model-dialog-section-head {
  @apply flex flex-wrap items-start justify-between gap-3 border-t border-gray-100 pt-4 dark:border-dark-700;
}

.model-dialog-section-title {
  @apply text-sm font-semibold text-gray-950 dark:text-white;
}

.model-dialog-section-note {
  @apply mt-1 text-xs leading-relaxed text-gray-500 dark:text-dark-400;
}

.model-price-field {
  @apply rounded-xl border border-gray-200 bg-gray-50/60 px-3 py-3 transition-colors dark:border-dark-700 dark:bg-dark-900/50;
}

.model-price-field:focus-within {
  border-color: #6ee7b7;
}

.model-price-field-foot {
  @apply mt-2 flex flex-wrap items-center gap-x-2 gap-y-1;
}

.model-price-field-state {
  @apply shrink-0 rounded-full px-2 py-0.5 text-[11px] font-medium;
}

.model-price-field-baseline {
  @apply min-w-0 text-[11px] leading-relaxed text-gray-500 dark:text-dark-400;
}

.model-request-price {
  @apply mt-3 rounded-xl border border-gray-200 bg-gray-50/60 px-3 py-3 dark:border-dark-700 dark:bg-dark-900/50;
}

.model-request-price-note {
  @apply mt-1 text-[11px] leading-relaxed text-gray-500 dark:text-dark-400;
}

.model-price-field-state.is-custom {
  @apply bg-amber-100 text-amber-800 dark:bg-amber-500/15 dark:text-amber-200;
}

.model-price-field-state.is-official {
  @apply bg-sky-100 text-sky-700 dark:bg-sky-500/15 dark:text-sky-200;
}

.model-price-field-state.is-unset {
  @apply bg-gray-100 text-gray-500 dark:bg-dark-700 dark:text-dark-400;
}

.model-baseline-aside {
  @apply rounded-2xl border border-gray-200 bg-white p-4 dark:border-dark-700 dark:bg-dark-900;
}

@media (min-width: 1024px) {
  .model-baseline-aside {
    @apply sticky top-0 self-start;
  }
}

.model-baseline-head {
  @apply mb-3;
}

.model-baseline-title {
  @apply text-sm font-semibold text-gray-950 dark:text-white;
}

.model-baseline-note {
  @apply mt-1 text-xs text-gray-500 dark:text-dark-400;
}

.model-baseline-rows {
  @apply space-y-2;
}

.model-baseline-row {
  @apply rounded-xl border border-transparent bg-gray-50 px-3 py-2 dark:bg-dark-800/60;
}

.model-baseline-row-top {
  @apply flex items-baseline justify-between gap-2;
}

.model-baseline-row-name {
  @apply text-xs text-gray-600 dark:text-dark-300;
}

.model-baseline-row-value {
  @apply font-mono text-sm font-semibold text-gray-950 dark:text-white;
}

.model-baseline-row-value.is-inherited {
  @apply text-sky-700 dark:text-sky-300;
}

.model-baseline-row-value.is-unset {
  @apply text-xs font-normal text-gray-400 dark:text-dark-500;
}

.model-baseline-row-meta {
  @apply mt-1 flex items-center justify-between gap-2 text-[11px] text-gray-500 dark:text-dark-400;
}

.model-baseline-ratio {
  @apply font-mono font-medium;
}

.model-baseline-row.is-high {
  @apply bg-amber-50 dark:bg-amber-500/10;
}

.model-baseline-row.is-high .model-baseline-ratio {
  @apply text-amber-700 dark:text-amber-300;
}

.model-baseline-row.is-low {
  @apply bg-emerald-50 dark:bg-emerald-500/10;
}

.model-baseline-row.is-low .model-baseline-ratio {
  @apply text-emerald-700 dark:text-emerald-300;
}

.model-baseline-row.is-same .model-baseline-ratio {
  @apply text-gray-500 dark:text-dark-400;
}

.model-baseline-row.is-follow {
  @apply border-sky-200 bg-sky-50/70 dark:border-sky-500/25 dark:bg-sky-500/10;
}

.model-baseline-row.is-unset {
  @apply border-dashed border-gray-300 bg-transparent dark:border-dark-600;
}

.model-baseline-summary {
  @apply mt-3 flex items-center justify-between border-t border-gray-100 pt-3 dark:border-dark-700;
}

.model-baseline-summary-label {
  @apply text-xs text-gray-500 dark:text-dark-400;
}

.model-baseline-summary-value {
  @apply font-mono text-sm font-semibold text-gray-950 dark:text-white;
}

.model-baseline-foot {
  @apply mt-2 text-[11px] leading-relaxed text-gray-400 dark:text-dark-500;
}

/* 预填语义的提醒。用琥珀左边线而不是普通灰字：这是「不说就会做错决定」的一条，不是补充说明。 */
.model-baseline-hint {
  @apply mt-3 rounded-lg border-l-2 border-amber-400 bg-amber-50/70 px-3 py-2 text-[11px] leading-relaxed text-amber-900 dark:border-amber-500/60 dark:bg-amber-500/10 dark:text-amber-100;
}
</style>

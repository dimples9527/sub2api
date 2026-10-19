<template>
  <AppLayout>
    <div class="ma-page">
      <!-- 作品高度探针：隐藏的同源 iframe，只为读一次 scrollHeight（见 artworkViewHeight）。
           展示用的预览一律 sandbox=""，不给模型吐的 HTML 任何本站上下文；
           这里多给 allow-same-origin 才读得到 DOM，但脚本仍然禁跑（没有 allow-scripts）。 -->
      <iframe
        ref="artworkMeasureFrame"
        class="ma-art-measure"
        sandbox="allow-same-origin"
        aria-hidden="true"
        tabindex="-1"
      ></iframe>
      <!-- 页头：标题 + 当前配置 + 三项统计 -->
      <section class="ma-top">
        <div class="ma-top-head">
          <div class="ma-top-copy">
            <div class="ma-top-title-line">
              <h1 class="ma-top-title">{{ t('modelDegradation.title') }}</h1>
              <span class="ma-top-sub">{{ t('modelDegradation.subtitle') }}</span>
            </div>
            <div class="ma-top-chips">
              <span class="ma-chip">
                {{ t('modelDegradation.chips.model') }}
                <b>{{ form.modelId || '—' }}</b>
              </span>
              <span class="ma-chip">
                {{ t('modelDegradation.form.reasoningEffort') }}
                <b>{{ currentEffortLabel }}</b>
              </span>
            </div>
          </div>
          <dl class="ma-stats">
            <div class="ma-stat">
              <dt>{{ t('modelDegradation.stats.passRate') }}</dt>
              <dd>
                <strong>{{ passRate }}</strong>
              </dd>
            </div>
            <div class="ma-stat">
              <dt>{{ t('modelDegradation.stats.lastDuration') }}</dt>
              <dd>
                <strong>{{ lastDurationLabel }}</strong>
              </dd>
            </div>
            <div class="ma-stat">
              <dt>{{ t('modelDegradation.stats.runCount') }}</dt>
              <dd>
                <strong>{{ history.length }}</strong>
              </dd>
            </div>
          </dl>
        </div>

        <!-- 监控状态：最近若干次检测的色条 -->
        <div class="ma-monitor-head">
          <div class="ma-monitor-title">
            <span>{{ t('modelDegradation.monitor.title') }}</span>
            <span class="ma-monitor-cap">
              {{ t('modelDegradation.monitor.capacity', { count: BAR_COUNT }) }}
            </span>
          </div>
          <button
            type="button"
            class="ma-icon-btn"
            :disabled="loadingHistory"
            :aria-label="t('modelDegradation.history.title')"
            @click="loadHistory"
          >
            <Icon name="refresh" size="sm" :class="loadingHistory ? 'animate-spin' : ''" />
          </button>
        </div>
        <div class="ma-bars">
          <span v-for="(bar, index) in bars" :key="index" class="ma-bar" :class="bar"></span>
        </div>
        <div class="ma-caption">
          <span>{{ t('modelDegradation.monitor.caption', { count: BAR_COUNT }) }}</span>
          <div class="ma-legend">
            <span><i class="passed"></i>{{ t('modelDegradation.monitor.legend.passed') }}</span>
            <span><i class="error"></i>{{ t('modelDegradation.monitor.legend.error') }}</span>
            <span><i class="incorrect"></i>{{ t('modelDegradation.monitor.legend.incorrect') }}</span>
          </div>
          <span>{{ history.length ? '' : t('modelDegradation.monitor.empty') }}</span>
        </div>
      </section>

      <!-- 发起检测 -->
      <section class="ma-detect">
        <div class="ma-panel">
          <div class="ma-form-col">
            <div class="ma-form-head">
              <h2 class="ma-h2">{{ t('modelDegradation.form.title') }}</h2>
              <div class="ma-seg">
                <button
                  v-for="task in taskOptions"
                  :key="task.value"
                  type="button"
                  class="ma-seg-btn"
                  :class="{ 'is-active': form.tasks.includes(task.value) }"
                  :aria-pressed="form.tasks.includes(task.value)"
                  :disabled="busy"
                  @click="selectTask(task.value)"
                >
                  {{ t(task.label) }}
                </button>
              </div>
            </div>

            <!-- 题目选择：每个选中的检测项下面铺一排卡片。点卡片即选中，
                 题目正文不直接铺出来，靠卡片右下角的「详情」按需展开。 -->
            <div v-if="form.tasks.length" class="ma-questions">
              <div v-for="type in form.tasks" :key="type" class="ma-question">
                <div class="ma-question-head">
                  <span class="ma-label">{{ taskTypeLabel(type) }}</span>
                  <span class="ma-qcount">{{ presetSummary(type) }}</span>
                  <label class="ma-qcustom-toggle">
                    <input
                      type="checkbox"
                      :checked="customEnabled[type]"
                      :disabled="busy"
                      @change="toggleCustom(type, $event)"
                    />
                    <span>{{ t('modelDegradation.form.customQuestion') }}</span>
                  </label>
                </div>

                <div v-if="loadingTasks" class="ma-qempty">
                  {{ t('modelDegradation.form.taskBankLoading') }}
                </div>
                <div v-else-if="!taskBank[type].length" class="ma-qempty">
                  {{ t('modelDegradation.form.taskBankEmpty') }}
                </div>
                <div v-else class="ma-qcards" role="radiogroup" :aria-label="taskTypeLabel(type)">
                  <div
                    v-for="option in taskBank[type]"
                    :key="option.key"
                    class="ma-qcard"
                    :class="{
                      'is-selected': presetKeys[type] === option.key,
                      'is-disabled': cardLocked(type, option.key)
                    }"
                    role="radio"
                    :aria-checked="presetKeys[type] === option.key"
                    :aria-disabled="cardLocked(type, option.key)"
                    tabindex="0"
                    @click="togglePreset(type, option.key)"
                    @keydown.enter.prevent="togglePreset(type, option.key)"
                    @keydown.space.prevent="togglePreset(type, option.key)"
                  >
                    <span class="ma-qcard-check">
                      <Icon v-if="presetKeys[type] === option.key" name="check" size="sm" />
                    </span>
                    <span class="ma-qcard-title">{{ option.title }}</span>
                    <button
                      type="button"
                      class="ma-qcard-detail"
                      :class="{ 'is-open': isDetailOpen(type, option.key) }"
                      :disabled="busy"
                      @click.stop="toggleDetail(type, option.key)"
                    >
                      <Icon name="eye" size="sm" />
                      <span>{{
                        isDetailOpen(type, option.key)
                          ? t('modelDegradation.form.detailHide')
                          : t('modelDegradation.form.detailShow')
                      }}</span>
                    </button>
                    <p v-if="isDetailOpen(type, option.key)" class="ma-qcard-prompt">
                      {{ option.prompt }}
                    </p>
                  </div>
                </div>

                <div v-if="customEnabled[type]" class="ma-qcustom">
                  <textarea
                    v-model="customPrompt[type]"
                    class="ma-qtextarea"
                    rows="3"
                    :disabled="busy"
                    :placeholder="t('modelDegradation.form.customPromptPlaceholder')"
                  ></textarea>
                  <Input
                    v-if="type === 'logic'"
                    v-model="customExpected"
                    :label="t('modelDegradation.form.customExpected')"
                    :placeholder="t('modelDegradation.form.customExpectedPlaceholder')"
                    :hint="t('modelDegradation.form.customExpectedHint')"
                    :disabled="busy"
                  />
                </div>
              </div>
            </div>

            <div class="ma-fields">
              <Input
                v-model="form.baseUrl"
                :label="t('modelDegradation.form.baseUrl')"
                :placeholder="t('modelDegradation.form.baseUrlPlaceholder')"
                :hint="t('modelDegradation.form.baseUrlHint')"
                autocomplete="off"
                :input-attrs="credentialInputAttrs"
                :disabled="busy"
              />
              <Input
                v-model="form.apiKey"
                type="password"
                :label="t('modelDegradation.form.apiKey')"
                :placeholder="t('modelDegradation.form.apiKeyPlaceholder')"
                :hint="t('modelDegradation.form.apiKeyHint')"
                autocomplete="new-password"
                :input-attrs="credentialInputAttrs"
                :disabled="busy"
              />
            </div>

            <div class="ma-toolbar">
              <div class="ma-selects">
                <div class="ma-field">
                  <span class="ma-label">{{ t('modelDegradation.form.protocol') }}</span>
                  <Select
                    v-model="form.protocol"
                    :options="protocolOptions"
                    :aria-label="t('modelDegradation.form.protocol')"
                    :disabled="busy"
                    :searchable="false"
                  />
                </div>
                <div class="ma-field">
                  <span class="ma-label">{{ t('modelDegradation.form.model') }}</span>
                  <Select
                    v-model="form.modelId"
                    :options="modelOptions"
                    :placeholder="modelPlaceholder"
                    :aria-label="t('modelDegradation.form.model')"
                    :disabled="busy"
                    searchable
                    creatable
                    :creatable-prefix="t('modelDegradation.form.modelCreatablePrefix')"
                    :search-placeholder="t('modelDegradation.form.modelSearchPlaceholder')"
                    :empty-text="t('modelDegradation.form.modelEmpty')"
                  />
                </div>
                <div class="ma-field">
                  <span class="ma-label">{{ t('modelDegradation.form.reasoningEffort') }}</span>
                  <Select
                    v-model="form.reasoningEffort"
                    :options="effortOptions"
                    :placeholder="t('modelDegradation.form.reasoningEffortPlaceholder')"
                    :aria-label="t('modelDegradation.form.reasoningEffort')"
                    :disabled="reasoningEffortDisabled"
                    :searchable="false"
                  />
                </div>
                <div class="ma-field">
                  <span class="ma-label">{{ t('modelDegradation.form.renderTimeout') }}</span>
                  <Select
                    v-model="form.renderTimeoutMinutes"
                    :options="renderTimeoutOptions"
                    :placeholder="t('modelDegradation.form.renderTimeoutPlaceholder')"
                    :aria-label="t('modelDegradation.form.renderTimeout')"
                    :disabled="renderTimeoutDisabled"
                    searchable
                    creatable
                    :creatable-prefix="t('modelDegradation.form.renderTimeoutCreatablePrefix')"
                    :search-placeholder="t('modelDegradation.form.renderTimeoutSearchPlaceholder')"
                  />
                </div>
              </div>
              <div class="ma-submit">
                <button
                  type="button"
                  class="ma-ghost"
                  :disabled="fetchModelsDisabled"
                  @click="handleFetchModels"
                >
                  <Icon name="refresh" size="sm" :class="loadingModels ? 'animate-spin' : ''" />
                  {{
                    loadingModels
                      ? t('modelDegradation.form.modelFetching')
                      : t('modelDegradation.form.modelFetch')
                  }}
                </button>
                <button type="button" class="ma-btn" :disabled="!canSubmit" @click="handleStart">
                  <Icon
                    :name="busy ? 'refresh' : 'checkCircle'"
                    size="sm"
                    :class="busy ? 'animate-spin' : ''"
                  />
                  {{ busy ? t('modelDegradation.form.submitting') : t('modelDegradation.form.submit') }}
                </button>
              </div>
            </div>

            <p class="ma-hint">{{ protocolHint }}</p>
            <p class="ma-note">{{ t('modelDegradation.form.securityNote') }}</p>
          </div>

          <!-- 我的检测：当前/最近一次结果 -->
          <aside class="ma-result-col">
            <!-- 标题行对齐源站的「我的检测」：标题 + 账号名，右侧翻页器。
                 源站第二行是「访客 ID」，我们这边是登录用户，换成账号 ID。 -->
            <div class="ma-mine-head">
              <div class="ma-mine-id">
                <h3 class="ma-h3">{{ t('modelDegradation.resultPanel.title') }}</h3>
                <span v-if="accountName" class="ma-mine-user">{{ accountName }}</span>
              </div>
              <div v-if="mineRuns.length > 1" class="ma-mine-pager">
                <button
                  type="button"
                  class="ma-mine-step"
                  :disabled="running || mineIndex <= 0"
                  :aria-label="t('modelDegradation.resultPanel.prev')"
                  @click="stepMine(-1)"
                >
                  <Icon name="chevronLeft" size="sm" />
                </button>
                <span class="ma-mine-page">{{ mineIndex + 1 }} / {{ mineRuns.length }}</span>
                <button
                  type="button"
                  class="ma-mine-step"
                  :disabled="running || mineIndex >= mineRuns.length - 1"
                  :aria-label="t('modelDegradation.resultPanel.next')"
                  @click="stepMine(1)"
                >
                  <Icon name="chevronRight" size="sm" />
                </button>
              </div>
            </div>
            <p v-if="accountId !== null" class="ma-mine-sub">
              {{ t('modelDegradation.resultPanel.accountId', { id: accountId }) }}
            </p>
            <div class="ma-result-body">
              <!-- 运行中：一行「进行中 + 秒表 + 停止」，下面一行模型与思考等级。
                   刻意不做题目清单 —— 单轮最长 10 分钟，用户真正想知道的是
                   「还要等多久、能不能现在就走」，逐题进度反而把结论淹了。 -->
              <template v-if="running">
                <div class="ma-running">
                  <div class="ma-running-row">
                    <span class="ma-running-mascot">
                      <ModelDegradationMascot
                        :animal="mascotAnimal(String(runModelLabel))"
                        state="thinking"
                      />
                    </span>
                    <span class="ma-running-label" :class="{ 'is-retrying': currentRun?.retrying }">
                      {{
                        currentRun?.retrying
                          ? t('modelDegradation.resultPanel.retrying')
                          : t('modelDegradation.resultPanel.running')
                      }}
                    </span>
                    <span class="ma-running-count">{{ arrivedCount }}/{{ progressTotal }}</span>
                    <span class="ma-running-right">
                      <span class="ma-running-time">
                        <Icon name="clock" size="sm" />{{ elapsedLabel }}
                      </span>
                      <button
                        type="button"
                        class="ma-running-stop"
                        :disabled="cancelling"
                        :title="t('modelDegradation.resultPanel.stop')"
                        @click="handleCancel"
                      >
                        <span class="ma-running-stop-box"></span>
                      </button>
                    </span>
                  </div>
                  <p class="ma-running-model">
                    {{ runModelLabel }}<span class="ma-running-dot">·</span>{{ runEffortLabel }}
                  </p>
                </div>
              </template>
              <template v-else-if="currentRun">
                <!-- 单张大卡：预览铺满卡面，右上斜角勋章（源站「我的检测」同形制）。
                     逻辑题没有产物，退回判定概览封面，两种卡片形制保持一致。 -->
                <div class="ma-mine-card">
                  <div
                    class="ma-mine-stage"
                    :class="[thumbTone(currentRun), { 'is-render': !!artworkPreviewHtml }]"
                  >
                    <ModelDegradationWave
                      v-if="!artworkPreviewHtml"
                      :running="currentRun.status === 'running'"
                    />
                    <svg
                      v-if="artworkPreviewHtml"
                      class="ma-artview"
                      :viewBox="artworkViewBox(artworkPreviewHtml)"
                      preserveAspectRatio="xMidYMid meet"
                      aria-hidden="true"
                    >
                      <foreignObject
                        x="0"
                        y="0"
                        width="1280"
                        :height="artworkViewHeight(artworkPreviewHtml)"
                      >
                        <iframe
                          :srcdoc="artworkSrcdoc(artworkPreviewHtml)"
                          sandbox=""
                          class="ma-frame"
                          :title="t('modelDegradation.item.preview')"
                        ></iframe>
                      </foreignObject>
                    </svg>
                    <template v-else>
                      <span class="ma-mascot">
                        <ModelDegradationMascot
                          :animal="mascotAnimal(String(currentRun.model_id ?? currentRun.id))"
                          :state="mascotState(currentRun)"
                        />
                      </span>
                      <strong class="ma-logic-verdict">{{ logicVerdictLabel(currentRun) }}</strong>
                      <span class="ma-logic-score">{{ tileSummary(currentRun) }}</span>
                      <dl class="ma-logic-rows">
                        <div v-for="row in logicRows(currentRun)" :key="row.label">
                          <dt>{{ row.label }}</dt>
                          <dd>{{ row.value }}</dd>
                        </div>
                      </dl>
                    </template>
                  </div>
                  <span class="ma-ribbon" :class="ribbonClass(currentRun)">{{
                    ribbonLabel(currentRun)
                  }}</span>
                </div>
                <p class="ma-verdict-hint">
                  {{
                    currentRun.status === 'cancelled'
                      ? t('modelDegradation.resultPanel.cancelledHint')
                      : (currentRun.render_verdict === 'generated' && currentRun.logic_total === 0)
                        ? t('modelDegradation.verdictHint.renderOnly')
                        : t('modelDegradation.verdictHint.' + currentRun.verdict)
                  }}
                </p>
                <p v-if="currentRun.error_message" class="ma-error">
                  {{ currentRun.error_message }}
                </p>
                <dl class="ma-mini">
                  <div>
                    <dt>{{ t('modelDegradation.metrics.model') }}</dt>
                    <dd>{{ currentRun.model_id }}</dd>
                  </div>
                  <div>
                    <dt>{{ t('modelDegradation.metrics.logic') }}</dt>
                    <dd>{{ currentRun.logic_passed }}/{{ currentRun.logic_total }}</dd>
                  </div>
                  <div>
                    <dt>{{ t('modelDegradation.metrics.render') }}</dt>
                    <dd>{{ renderLabel }}</dd>
                  </div>
                  <div>
                    <dt>{{ t('modelDegradation.metrics.reasoningEffort') }}</dt>
                    <dd>{{ effortLabel(currentRun.reasoning_effort) }}</dd>
                  </div>
                  <div>
                    <dt>{{ t('modelDegradation.metrics.protocol') }}</dt>
                    <dd>{{ t('modelDegradation.form.protocolLabel.' + currentRun.protocol) }}</dd>
                  </div>
                  <div>
                    <dt>{{ t('modelDegradation.metrics.renderTimeout') }}</dt>
                    <dd>{{ renderTimeoutLabel(currentRun.render_timeout_seconds) }}</dd>
                  </div>
                  <div>
                    <dt>{{ t('modelDegradation.metrics.duration') }}</dt>
                    <dd>{{ formatDuration(currentRun.duration_ms) }}</dd>
                  </div>
                </dl>
              </template>
              <p v-else class="ma-pending">{{ t('modelDegradation.resultPanel.waiting') }}</p>
            </div>
          </aside>
        </div>
      </section>

      <!-- 检测结果：单张详情卡（头部 / 大预览 / 指标页脚），对齐源站的访客检测详情。
           逐题的题目与回答收进底部折叠区 —— 题目正文很长，铺出来就是一整面文字墙。
           生成题的 iframe 用 sandbox="" 且不执行脚本：模型吐的 HTML 不因此获得本站上下文。
           三个条件里 `currentRun &&` 必须在最前：vue-tsc 靠它把 currentRun 收窄成非空。 -->
      <section
        v-if="currentRun && !running && currentRun.status !== 'running'"
        class="ma-section"
      >
        <!-- 折叠下拉条：铺满整行的抬头条，整条可点，点哪里都收起 / 展开结果卡。
             开合按钮做成正中间的圆片箭头（不写字），收起时留在页面上的是这条抬头。 -->
        <h2 class="ma-result-heading">
          <button
            type="button"
            class="ma-result-bar"
            :aria-expanded="resultExpanded"
            aria-controls="ma-result-body"
            @click="resultExpanded = !resultExpanded"
          >
            <span class="ma-h2">{{ t('modelDegradation.result.title') }}</span>
            <span class="ma-result-bar-sub">{{ currentRun.base_url }}</span>
            <span
              class="ma-result-bar-toggle"
              :class="{ 'is-collapsed': !resultExpanded }"
              aria-hidden="true"
            >
              <Icon name="chevronDown" size="md" />
            </span>
          </button>
        </h2>
        <article id="ma-result-body" v-show="resultExpanded" class="ma-detail">
          <header class="ma-detail-head">
            <div class="ma-detail-id">
              <span class="ma-detail-name">{{ currentRun.model_id }}</span>
              <span class="ma-detail-time">{{ formatDateTime(currentRun.created_at) }}</span>
              <span class="ma-detail-owner">
                {{ ownerLabel(currentRun) }}
                <span v-if="currentRun.is_owner" class="ma-owner-tag">{{
                  t('modelDegradation.history.ownerSelf')
                }}</span>
              </span>
            </div>
            <div class="ma-detail-actions">
              <button
                type="button"
                :title="t('modelDegradation.history.share')"
                :aria-label="t('modelDegradation.history.share')"
                @click="copyRunSummary(currentRun)"
              >
                <Icon name="upload" size="sm" />
              </button>
              <button
                type="button"
                :disabled="!artworkPreviewHtml"
                :title="t('modelDegradation.result.copyArtwork')"
                :aria-label="t('modelDegradation.result.copyArtwork')"
                @click="copyArtwork(currentRun)"
              >
                <Icon name="copy" size="sm" />
              </button>
              <button
                type="button"
                :title="t('modelDegradation.history.download')"
                :aria-label="t('modelDegradation.history.download')"
                @click="downloadRunReport(currentRun)"
              >
                <Icon name="download" size="sm" />
              </button>
              <button
                type="button"
                :disabled="!artworkPreviewHtml"
                :title="t('modelDegradation.result.openArtwork')"
                :aria-label="t('modelDegradation.result.openArtwork')"
                @click="openArtwork(currentRun)"
              >
                <Icon name="externalLink" size="sm" />
              </button>
              <button
                type="button"
                :disabled="!artworkPreviewHtml"
                :title="t('modelDegradation.result.downloadArtwork')"
                :aria-label="t('modelDegradation.result.downloadArtwork')"
                @click="downloadArtwork(currentRun)"
              >
                <Icon name="document" size="sm" />
              </button>
            </div>
          </header>

          <div
            class="ma-detail-stage"
            :class="[thumbTone(currentRun), { 'is-render': !!artworkPreviewHtml }]"
          >
            <!-- 结果卡只在跑完之后出现，波形没必要进 is-running -->
            <ModelDegradationWave v-if="!artworkPreviewHtml" />
            <svg
              v-if="artworkPreviewHtml"
              class="ma-artview"
              :viewBox="artworkViewBox(artworkPreviewHtml)"
              preserveAspectRatio="xMidYMid meet"
              aria-hidden="true"
            >
              <foreignObject
                x="0"
                y="0"
                width="1280"
                :height="artworkViewHeight(artworkPreviewHtml)"
              >
                <iframe
                  :srcdoc="artworkSrcdoc(artworkPreviewHtml)"
                  sandbox=""
                  class="ma-frame"
                  :title="t('modelDegradation.item.preview')"
                ></iframe>
              </foreignObject>
            </svg>
            <template v-else>
              <span class="ma-mascot">
                <ModelDegradationMascot
                  :animal="mascotAnimal(String(currentRun.model_id ?? currentRun.id))"
                  :state="mascotState(currentRun)"
                />
              </span>
              <strong class="ma-logic-verdict">{{ logicVerdictLabel(currentRun) }}</strong>
              <span class="ma-logic-score">{{ tileSummary(currentRun) }}</span>
              <dl class="ma-logic-rows">
                <div v-for="row in logicRows(currentRun)" :key="row.label">
                  <dt>{{ row.label }}</dt>
                  <dd>{{ row.value }}</dd>
                </div>
              </dl>
            </template>
            <span class="ma-ribbon" :class="ribbonClass(currentRun)">{{
              ribbonLabel(currentRun)
            }}</span>
          </div>

          <footer class="ma-detail-foot">
            <div class="ma-detail-foot-main">
              <span class="ma-detail-model">{{ currentRun.model_id }}</span>
              <span class="ma-detail-chip">{{ effortLabel(currentRun.reasoning_effort) }}</span>
              <span class="ma-detail-dur">
                <Icon name="clock" size="sm" />{{ formatDuration(currentRun.duration_ms) }}
              </span>
            </div>
            <dl class="ma-detail-stats">
              <div>
                <dt>{{ t('modelDegradation.metrics.logic') }}</dt>
                <dd>{{ currentRun.logic_passed }}/{{ currentRun.logic_total }}</dd>
              </div>
              <div>
                <dt>{{ t('modelDegradation.metrics.render') }}</dt>
                <dd>{{ renderLabel }}</dd>
              </div>
              <div>
                <dt>{{ t('modelDegradation.metrics.protocol') }}</dt>
                <dd>{{ t('modelDegradation.form.protocolLabel.' + currentRun.protocol) }}</dd>
              </div>
            </dl>
            <p v-if="currentRun.error_message" class="ma-error">
              {{ currentRun.error_message }}
            </p>
          </footer>

          <details v-if="(currentRun.items || []).length" class="ma-detail-more">
            <summary>{{ t('modelDegradation.result.questionDetail') }}</summary>
            <ul class="ma-qlist">
              <li v-for="item in currentRun.items || []" :key="item.task_key">
                <div class="ma-item-meta">
                  <span class="ma-badge" :class="itemBadgeClass(item)">{{ itemBadgeLabel(item) }}</span>
                  <span class="ma-item-kind">
                    {{
                      t('modelDegradation.form.' + (item.task_type === 'logic' ? 'taskLogic' : 'taskRender'))
                    }}
                  </span>
                  <span class="ma-item-dur">{{ formatDuration(item.duration_ms) }}</span>
                </div>
                <p class="ma-item-prompt">{{ item.prompt }}</p>
                <p v-if="item.expected" class="ma-item-row">
                  <b>{{ t('modelDegradation.result.expected') }}</b>{{ item.expected }}
                </p>
                <!-- 生成题这里只出源码之外的那段话：源码本体在预览里看，
                     整份贴进来会把折叠区撑成一面代码墙。 -->
                <p v-if="itemAnswerText(item)" class="ma-item-row">
                  <b>{{ t('modelDegradation.result.answer') }}</b>{{ itemAnswerText(item) }}
                </p>
                <p class="ma-item-detail">{{ item.detail }}</p>
              </li>
            </ul>
          </details>
        </article>
      </section>

      <!-- 检测历史 -->
      <section class="ma-section">
        <div class="ma-section-head ma-history-head">
          <h2 class="ma-h2">{{ t('modelDegradation.history.title') }}</h2>
          <span class="ma-section-sub">
            {{ t('modelDegradation.monitor.caption', { count: BAR_COUNT }) }}
          </span>
          <div class="ma-gallery-filter">
            <span class="ma-label">{{ t('modelDegradation.history.typeFilterLabel') }}</span>
            <Select
              v-model="historyTypeFilter"
              :options="historyTypeOptions"
              :aria-label="t('modelDegradation.history.typeFilterLabel')"
              :searchable="false"
            />
            <span class="ma-label">{{ t('modelDegradation.history.filterLabel') }}</span>
            <Select
              v-model="historyFilter"
              :options="historyFilterOptions"
              :aria-label="t('modelDegradation.history.filterLabel')"
              :searchable="false"
            />
            <span class="ma-label">{{ t('modelDegradation.history.groupFilterLabel') }}</span>
            <Select
              v-model="historyGroupFilter"
              :options="historyGroupOptions"
              class="ma-group-select"
              :aria-label="t('modelDegradation.history.groupFilterLabel')"
            />
          </div>
        </div>
        <!-- 「一条记录都没有」和「筛掉了所有记录」是两件事，文案要分开：
             前者是还没跑过，后者点了筛选才出现，说「还没有记录」会让人以为没跑成功。 -->
        <p v-if="!history.length" class="ma-empty">
          {{ t('modelDegradation.history.empty') }}
        </p>
        <p v-else-if="!filteredHistory.length" class="ma-empty">
          {{ t('modelDegradation.history.filterEmpty') }}
        </p>
        <div v-else ref="historyWallRef" class="ma-groups">
          <!-- 分组归类：同一分组的记录聚成一节，节头是分组名 + 条数 + 细线。
               节头下方整组卡片铺成一行、不换行；一行放不下时由节头右端的
               ‹ › 按钮横向滚动（行本身是滚动容器）。没命中分组的按上游
               主机名归节（与缎带同一口径）。 -->
          <template v-for="group in historyGroups" :key="group.key">
            <div class="ma-group-head">
              <h3 class="ma-group-name" :class="{ 'ma-group-name-hidden': group.hidden }">{{ group.label }}</h3>
              <span
                v-if="group.hidden"
                class="ma-group-hidden"
                :title="t('modelDegradation.history.hiddenHint')"
              >
                {{ t('modelDegradation.history.hiddenBadge') }}
              </span>
              <span class="ma-group-count">{{ group.runs.length }}</span>
              <span class="ma-group-rule" aria-hidden="true"></span>
              <div v-show="rowScroll[group.key]?.overflow" class="ma-group-nav">
                <button
                  type="button"
                  class="ma-wall-nav"
                  :disabled="!rowScroll[group.key]?.left"
                  :aria-label="t('modelDegradation.history.scrollPrev')"
                  @click="scrollGroupRow(group.key, -1)"
                >
                  <Icon name="chevronLeft" size="sm" />
                </button>
                <button
                  type="button"
                  class="ma-wall-nav"
                  :disabled="!rowScroll[group.key]?.right"
                  :aria-label="t('modelDegradation.history.scrollNext')"
                  @click="scrollGroupRow(group.key, 1)"
                >
                  <Icon name="chevronRight" size="sm" />
                </button>
              </div>
            </div>
            <div class="ma-group-row" :data-group-row="group.key">
              <!-- 每张卡片是一个 .ma-tile：预览居中铺在卡面里，右上斜角勋章。
                   静止时卡面上只有预览与勋章 —— 模型 / 耗时 / 判定不铺在卡上，
                   它们留在展开层里（源站的墙就是这么干净的）。
                   卡片本体用 role=button 的 div 而不是 <button>：预览里要放 <iframe>，
                   按钮里嵌 iframe 不是合法 HTML。键盘走 enter / space，与题目卡片一致。
                   移入时另起一张「大卡」浮层（.ma-hover-card）：标题行 + 预览 + 指标页脚，
                   形制照源站那张访客详情卡。浮层 fixed 定位、比卡片高：行是滚动容器，
                   绝对定位会被裁掉，fixed 才能整个探出来；位置在悬停时按卡片量出。
                   预览是浮层自己的一份 iframe，只在悬停的那一张上创建。 -->
              <div
                v-for="(run, runIndex) in group.runs"
                :key="run.id"
                class="ma-tile"
                :class="{
                  'is-zoom': zoomId === run.id,
                  'is-bound': timelineHoverId === run.id,
                  'is-chain-left': runIndex > 0,
                  'is-chain-right': runIndex < group.runs.length - 1
                }"
                :data-run-id="run.id"
                @mouseenter="timelineHoverId = run.id"
                @mouseleave="hideZoom(); timelineHoverId = null"
                @focusout="handleCardFocusOut(run, $event)"
              >
                <!-- 卡片正上方的时间点：日期 + 时间，下面一条细线垂到卡片上沿。
                     点与卡一一对应、互相点亮；点节点直接打开那一轮详情。 -->
                <div class="ma-tile-cap">
                  <button
                    type="button"
                    class="ma-group-node"
                    :class="[
                      group.nodes[runIndex].tone,
                      {
                        'is-latest': group.nodes[runIndex].latest,
                        'is-bound': timelineHoverId === run.id
                      }
                    ]"
                    :title="group.nodes[runIndex].title"
                    :aria-label="group.nodes[runIndex].title"
                    @click="openRun(run.id)"
                    @mouseenter="timelineHoverId = run.id"
                    @mouseleave="timelineHoverId = null"
                    @focus="timelineHoverId = run.id"
                    @blur="timelineHoverId = null"
                  >
                    <span class="ma-group-node-time">{{ group.nodes[runIndex].label }}</span>
                    <span class="ma-group-node-dot"></span>
                  </button>
                </div>
                <div
                  class="ma-card"
                  role="button"
                  tabindex="0"
                  :aria-label="t('modelDegradation.history.open')"
                  @mouseenter="showZoom(run); timelineHoverId = run.id"
                  @click="openRun(run.id)"
                  @keydown.enter.prevent="openRun(run.id)"
                  @keydown.space.prevent="openRun(run.id)"
                  @keydown.esc="hideZoom"
                  @focus="showZoom(run)"
                >
                  <span
                    class="ma-card-thumb"
                    :class="[thumbTone(run), { 'is-render': !!renderHtmlOf(run) }]"
                  >
                    <!-- 跑过生成题就有产物：整页按 1280px 逻辑视口渲染，
                         外层 SVG 把作品等比缩进封面框（见 .ma-artview 注释）。 -->
                    <ModelDegradationWave v-if="!renderHtmlOf(run)" :running="run.status === 'running'" />
                    <span v-if="renderHtmlOf(run)" class="ma-card-cover">
                      <svg
                        class="ma-artview"
                        :viewBox="artworkViewBox(renderHtmlOf(run))"
                        preserveAspectRatio="xMidYMid meet"
                        aria-hidden="true"
                      >
                        <foreignObject
                          x="0"
                          y="0"
                          width="1280"
                          :height="artworkViewHeight(renderHtmlOf(run))"
                        >
                          <iframe
                            :srcdoc="artworkSrcdoc(renderHtmlOf(run))"
                            sandbox=""
                            class="ma-card-frame"
                            :title="t('modelDegradation.item.preview')"
                          ></iframe>
                        </foreignObject>
                      </svg>
                    </span>
                    <!-- 纯逻辑题没有可预览的产物，用判定概览顶一张同形制的封面 -->
                    <template v-else>
                      <Icon
                        :name="tileIcon(run)"
                        size="lg"
                        class="ma-card-glyph"
                        :class="run.status === 'running' ? 'animate-spin' : ''"
                      />
                      <span class="ma-card-glyph-text">{{ tileSummary(run) }}</span>
                      <span v-if="tileReason(run)" class="ma-card-glyph-note">{{ tileReason(run) }}</span>
                    </template>
                    <!-- 勋章压的是「预览」的角：源站那张详情卡里，标题行在上面、
                         缎带从作品框的右上角斜插下来，被作品框的圆角裁断。
                         放在 thumb 里两种形态（静止 / 展开）才对得上。 -->
                    <span class="ma-ribbon" :class="ribbonClass(run)">{{ ribbonLabel(run) }}</span>
                  </span>
                </div>

                <!-- 展开大卡：铺在卡片之上并向上探出标题行的高度，只在悬停/聚焦时显形。
                     整卡可点、点空白处与卡片本体一样打开详情；操作钮必须 .stop，
                     否则按钮的点击会跟着触发打开详情。 -->
                <div
                  class="ma-hover-card"
                  :style="hoverCardStyleOf(run.id)"
                  :aria-hidden="zoomId === run.id ? undefined : 'true'"
                  @click="openRun(run.id)"
                >
                  <div class="ma-more-head">
                    <div class="ma-more-title">
                      <span class="ma-more-name">{{ run.model_id }}</span>
                      <span class="ma-more-time">{{ formatDateTime(run.created_at) }}</span>
                      <span class="ma-more-owner">
                        {{ ownerLabel(run) }}
                        <span v-if="run.is_owner" class="ma-owner-tag">{{
                          t('modelDegradation.history.ownerSelf')
                        }}</span>
                      </span>
                    </div>
                    <div class="ma-more-actions">
                      <button
                        type="button"
                        :title="t('modelDegradation.history.detail')"
                        :aria-label="t('modelDegradation.history.detail')"
                        @click.stop="openRun(run.id)"
                      >
                        <Icon name="eye" size="sm" />
                      </button>
                      <button
                        type="button"
                        :title="t('modelDegradation.history.copy')"
                        :aria-label="t('modelDegradation.history.copy')"
                        @click.stop="copyRunSummary(run)"
                      >
                        <Icon name="copy" size="sm" />
                      </button>
                      <button
                        type="button"
                        :title="t('modelDegradation.history.download')"
                        :aria-label="t('modelDegradation.history.download')"
                        @click.stop="downloadRunReport(run)"
                      >
                        <Icon name="download" size="sm" />
                      </button>
                      <button
                        type="button"
                        :disabled="!renderHtmlOf(run)"
                        :title="t('modelDegradation.result.downloadArtwork')"
                        :aria-label="t('modelDegradation.result.downloadArtwork')"
                        @click.stop="downloadArtwork(run)"
                      >
                        <Icon name="document" size="sm" />
                      </button>
                    </div>
                  </div>

                  <span
                    class="ma-hover-stage"
                    :class="[thumbTone(run), { 'is-render': !!renderHtmlOf(run) }]"
                  >
                    <!-- 只在悬停这一张上建 iframe：铺满墙的话每张卡都要多一份文档 -->
                    <ModelDegradationWave v-if="!renderHtmlOf(run)" :running="run.status === 'running'" />
                    <span v-if="renderHtmlOf(run) && zoomId === run.id" class="ma-hover-cover">
                      <svg
                        class="ma-artview"
                        :viewBox="artworkViewBox(renderHtmlOf(run))"
                        preserveAspectRatio="xMidYMid meet"
                        aria-hidden="true"
                      >
                        <foreignObject
                          x="0"
                          y="0"
                          width="1280"
                          :height="artworkViewHeight(renderHtmlOf(run))"
                        >
                          <iframe
                            :srcdoc="artworkSrcdoc(renderHtmlOf(run))"
                            sandbox=""
                            class="ma-hover-frame"
                            :title="t('modelDegradation.item.preview')"
                          ></iframe>
                        </foreignObject>
                      </svg>
                    </span>
                    <template v-else>
                      <Icon :name="tileIcon(run)" size="lg" class="ma-card-glyph" />
                      <span class="ma-card-glyph-text">{{ tileSummary(run) }}</span>
                      <span v-if="tileReason(run)" class="ma-card-glyph-note">{{ tileReason(run) }}</span>
                    </template>
                    <span class="ma-ribbon" :class="ribbonClass(run)">{{ ribbonLabel(run) }}</span>
                  </span>

                  <div class="ma-more-foot">
                    <div class="ma-more-row">
                      <span class="ma-more-chip">{{ effortLabel(run.reasoning_effort) }}</span>
                      <span class="ma-more-dur">
                        <Icon name="clock" size="sm" />{{ formatDuration(run.duration_ms) }}
                      </span>
                    </div>
                    <dl class="ma-more-stats">
                      <div>
                        <dt>{{ t('modelDegradation.metrics.logic') }}</dt>
                        <dd>{{ run.logic_passed }}/{{ run.logic_total }}</dd>
                      </div>
                      <div>
                        <dt>{{ t('modelDegradation.metrics.render') }}</dt>
                        <dd>{{ renderVerdictLabel(run.render_verdict) }}</dd>
                      </div>
                    </dl>
                  </div>
                </div>
              </div>
            </div>
          </template>
        </div>
        <!-- 翻页器：历史一次最多 BAR_COUNT 条，分页是纯客户端的（列表接口只给
             limit，没有 offset / status 参数），不为此动后端。页码用夹紧后的
             historyPageSafe，筛掉大半记录时按钮的禁用态不会闪一下越界值。 -->
        <nav v-if="filteredHistory.length" class="ma-pager" :aria-label="t('modelDegradation.history.title')">
          <button
            type="button"
            class="ma-pager-btn"
            :disabled="historyPageSafe <= 1"
            @click="historyPage = historyPageSafe - 1"
          >
            {{ t('modelDegradation.history.prevPage') }}
          </button>
          <span class="ma-pager-status" role="status">
            {{
              t('modelDegradation.history.pageStatus', {
                page: historyPageSafe,
                pages: historyPageCount,
                total: filteredHistory.length
              })
            }}
          </span>
          <button
            type="button"
            class="ma-pager-btn"
            :disabled="historyPageSafe >= historyPageCount"
            @click="historyPage = historyPageSafe + 1"
          >
            {{ t('modelDegradation.history.nextPage') }}
          </button>
        </nav>
      </section>
    </div>
  </AppLayout>
</template>

<script setup lang="ts">
import { computed, nextTick, onMounted, onUnmounted, reactive, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import AppLayout from '@/components/layout/AppLayout.vue'
import Icon from '@/components/icons/Icon.vue'
import Input from '@/components/common/Input.vue'
import Select, { type SelectOption } from '@/components/common/Select.vue'
import modelDegradationAPI, {
  MODEL_AUDIT_PROTOCOLS,
  MODEL_AUDIT_REASONING_EFFORTS,
  MODEL_AUDIT_RENDER_TIMEOUT_DEFAULT_MINUTES,
  MODEL_AUDIT_RENDER_TIMEOUT_MAX_MINUTES,
  MODEL_AUDIT_RENDER_TIMEOUT_MINUTES,
  MODEL_AUDIT_RENDER_TIMEOUT_MIN_MINUTES,
  type ModelDegradationItem,
  type ModelDegradationItemVerdict,
  type ModelDegradationProtocol,
  type ModelDegradationQuestionPayload,
  type ModelDegradationReasoningEffort,
  type ModelDegradationRun,
  type ModelDegradationTaskBank,
  type ModelDegradationTaskType
} from '@/api/modelDegradation'
import { adminAPI } from '@/api/admin'
import { useAppStore } from '@/stores/app'
import { useAuthStore } from '@/stores/auth'
import { useClipboard } from '@/composables/useClipboard'
import { extractApiErrorMessage } from '@/utils/apiError'
import { formatDateTime, formatDateTimeToMinute, formatReasoningEffort } from '@/utils/format'
import ModelDegradationWave from './ModelDegradationWave.vue'
import ModelDegradationMascot from './ModelDegradationMascot.vue'

const { t } = useI18n()
const appStore = useAppStore()
const authStore = useAuthStore()
const { copyToClipboard } = useClipboard()

// 「我的检测」标题行右侧的账号标识。源站那里是匿名访客的「访客 ID」，
// 我们是登录用户，换成用户名 + 账号 ID 这套口径。
const accountName = computed(() => authStore.user?.username || authStore.user?.email || '')
const accountId = computed(() => authStore.user?.id ?? null)

// 监控状态条展示的检测次数（与源站的展示容量一致）。
const BAR_COUNT = 60
// 历史区每页铺这些条（翻页的页大小），避免一屏太多把页面拉长。
const HISTORY_CARD_COUNT = 20

// 不让浏览器把「别处存的账号密码」填进这两个输入框。
// 这两个框正好是一个 text + 一个 password，Chrome 会当成登录表单，
// 拿保存的邮箱/密码往里灌；autocomplete="new-password" 告诉它这不是登录，
// data-*-ignore 则让 1Password / LastPass / Bitwarden 跳过。
const credentialInputAttrs: Record<string, string> = {
  'data-1p-ignore': '',
  'data-lpignore': 'true',
  'data-bwignore': 'true'
}

/**
 * 上次选的模型、思考等级与生成题超时。
 *
 * 只记这几项：凭据（Base URL / API Key）绝不落盘 —— 它们是上游密钥，
 * 本功能的前提就是「只在本次调用里用一次、服务端不保存」。而模型、思考等级
 * 与超时是用户的偏好，每次重进页面都要重选一遍很烦。
 */
const LAST_SELECTION_KEY = 'model_degradation_last_selection'

// 超时档位必须是区间内的整数：旧版本存下的值、或存储被改坏时的怪值一律退回默认，
// 否则会把一个必然被后端拒绝的值填回输入框。
const readRenderTimeoutMinutes = (raw: unknown): string => {
  const fallback = String(MODEL_AUDIT_RENDER_TIMEOUT_DEFAULT_MINUTES)
  if (typeof raw !== 'string') return fallback
  const parsed = Number(raw.trim())
  if (!Number.isInteger(parsed)) return fallback
  if (
    parsed < MODEL_AUDIT_RENDER_TIMEOUT_MIN_MINUTES ||
    parsed > MODEL_AUDIT_RENDER_TIMEOUT_MAX_MINUTES
  ) {
    return fallback
  }
  return String(parsed)
}

const readLastSelection = (): {
  modelId: string
  reasoningEffort: ModelDegradationReasoningEffort
  renderTimeoutMinutes: string
} => {
  const empty = {
    modelId: '',
    // 默认 medium（与 manxue-ai 一致）：检测时恒定带档位发出；
    // 空串仅作为「不指定」选项保留，不再是默认值。
    reasoningEffort: 'medium' as ModelDegradationReasoningEffort,
    renderTimeoutMinutes: String(MODEL_AUDIT_RENDER_TIMEOUT_DEFAULT_MINUTES)
  }
  try {
    const raw = globalThis.localStorage?.getItem(LAST_SELECTION_KEY)
    if (!raw) return empty
    const parsed: unknown = JSON.parse(raw)
    if (typeof parsed !== 'object' || parsed === null) return empty
    const { modelId, reasoningEffort, renderTimeoutMinutes } = parsed as {
      modelId?: unknown
      reasoningEffort?: unknown
      renderTimeoutMinutes?: unknown
    }
    // 档位必须仍在本版枚举里：旧版本存下的值（含空串）退回默认 medium，
    // 否则会把一个上游不认识的参数发出去，白白换回一次 400。
    const effort =
      typeof reasoningEffort === 'string' &&
      (MODEL_AUDIT_REASONING_EFFORTS as readonly string[]).includes(reasoningEffort)
        ? (reasoningEffort as ModelDegradationReasoningEffort)
        : 'medium'
    return {
      modelId: typeof modelId === 'string' ? modelId : '',
      reasoningEffort: effort,
      renderTimeoutMinutes: readRenderTimeoutMinutes(renderTimeoutMinutes)
    }
  } catch {
    // 存储被禁用、或内容被改坏时静默退回默认值 —— 记不住偏好不该让页面打不开。
    return empty
  }
}

const lastSelection = readLastSelection()

const form = reactive({
  baseUrl: '',
  apiKey: '',
  // 默认 responses（与 manxue-ai 一致）：长生成走流式，连接全程有数据帧，
  // 不会被网关 idle 超时（如 Cloudflare 100s）掐断；chat_completions 非流式，
  // 生成期间零流量，慢路由上最容易被中间层截断而超时重试。
  protocol: 'responses' as ModelDegradationProtocol,
  modelId: lastSelection.modelId,
  // 默认 medium（与 manxue-ai 对齐）；空串 = 不指定，请求里不带 reasoning_effort。
  reasoningEffort: lastSelection.reasoningEffort,
  // 生成题单题超时（分钟）。存字符串而不是数字：下拉是 creatable 的，
  // 用户手输时输入框会经过 "1"、"15" 这类半成品，用数字存会把输入卡住。
  renderTimeoutMinutes: lastSelection.renderTimeoutMinutes,
  // 页签单选，一轮只跑一类。默认逻辑题：它最快、结果最直观，需要时切到生成题。
  tasks: ['logic'] as ModelDegradationTaskType[]
})

const persistLastSelection = () => {
  try {
    globalThis.localStorage?.setItem(
      LAST_SELECTION_KEY,
      JSON.stringify({
        modelId: form.modelId,
        reasoningEffort: form.reasoningEffort,
        renderTimeoutMinutes: form.renderTimeoutMinutes
      })
    )
  } catch {
    // 隐私模式 / 存储被禁用：静默跳过，不影响检测本身。
  }
}

const taskOptions = [
  { value: 'logic' as ModelDegradationTaskType, label: 'modelDegradation.form.taskLogic' },
  { value: 'render' as ModelDegradationTaskType, label: 'modelDegradation.form.taskRender' }
]

// 协议决定端点、认证头与响应体解析方式，选项文案按协议名取。
const protocolOptions = computed<SelectOption[]>(() =>
  MODEL_AUDIT_PROTOCOLS.map((protocol) => ({
    value: protocol,
    label: t(`modelDegradation.form.protocolLabel.${protocol}`)
  }))
)

// 切到不支持思考等级的协议时清掉已选档位，避免界面上显示一个不会被发送的值。
watch(
  () => form.protocol,
  (protocol) => {
    if (protocol === 'messages') form.reasoningEffort = ''
  }
)

// 选择一变就记下来，下次进页面直接填好。放在协议监听之后：被协议清掉档位那一下
// 也会落盘，正是想要的（否则下次会填回一个当前协议用不了的档位）。
watch(() => [form.modelId, form.reasoningEffort, form.renderTimeoutMinutes], persistLastSelection)

// 协议提示随协议变：追加的路径、认证头、以及哪些参数不生效都在这里说清楚。
const protocolHint = computed(() => t(`modelDegradation.form.protocolHint.${form.protocol}`))

// 思考等级固定枚举，档位名沿用后端与源站一致的英文原值，不翻译。
const effortOptions = computed<SelectOption[]>(() => [
  { value: '', label: t('modelDegradation.form.reasoningEffortPlaceholder') },
  ...MODEL_AUDIT_REASONING_EFFORTS.map((level) => ({
    value: level,
    label: formatReasoningEffort(level)
  }))
])

// 生成题超时：常用档位铺出来，同时允许手输区间内的任意整数（creatable）。
// 档位与区间跟后端 modelDegradationRenderTaskTimeout(Min/Max) 同源。
const renderTimeoutOptions = computed<SelectOption[]>(() =>
  MODEL_AUDIT_RENDER_TIMEOUT_MINUTES.map((minutes) => ({
    value: String(minutes),
    label: t('modelDegradation.form.renderTimeoutUnit', { minutes })
  }))
)

const running = ref(false)
const currentRun = ref<ModelDegradationRun | null>(null)
const history = ref<ModelDegradationRun[]>([])
const loadingHistory = ref(false)

// 检测结果卡的展开 / 折叠。默认展开；用户手动收起后保持收起，只有结果换成
// 另一轮（新跑完一次、或点开别的历史卡片）才自动回到展开态 —— 刚操作过的
// 那条记录一定是可见的，手动收起的偏好也不会被无关的刷新重置。
const resultExpanded = ref(true)

watch(
  () => currentRun.value?.id,
  (id, prev) => {
    if (id && id !== prev) resultExpanded.value = true
  }
)

// 本次提交了哪些题。只用来在运行中渲染「等待卡片」——
// run 记录里不存这个字段（轮询拿到的是库里的行），所以只能自己记。
const pendingQuestions = ref<{ key: string; taskType: ModelDegradationTaskType; label: string }[]>([])

// 内置题库。取自服务端，避免题目正文与判分用的期望答案在前端再抄一份。
const taskBank = ref<ModelDegradationTaskBank>({ logic: [], render: [] })
const loadingTasks = ref(false)

// 每个检测项下选了哪道预制题（存 key，空串 = 没选）。单选 —— 与后端
// modelDegradationMaxLogicQuestions / modelDegradationMaxRenderQuestions（都是 1）一致，
// 多选会被后端直接 400 掉。
const presetKeys = reactive<Record<ModelDegradationTaskType, string>>({ logic: '', render: '' })
// 是否改用自定义题。与预制题互斥 —— 同时开会让题目数量超出后端上限。
const customEnabled = reactive<Record<ModelDegradationTaskType, boolean>>({ logic: false, render: false })
const customPrompt = reactive<Record<ModelDegradationTaskType, string>>({ logic: '', render: '' })
// 自定义逻辑题的期望答案。逻辑题靠「回答里出现期望数字」判分，所以后端强制必填。
const customExpected = ref('')

// 哪些卡片的题目详情被展开了（存 `type:key"）。允许多个同时展开，
// 这样能横向对比几道题的全文；题目正文很长，全铺出来会把表单撑爆。
const openDetails = ref<string[]>([])

// 各类型单轮只跑一道题（与后端 modelDegradationMaxLogicQuestions /
// modelDegradationMaxRenderQuestions 都是 1 一致），多选会被后端直接 400。

const taskTypeLabel = (type: ModelDegradationTaskType) =>
  t(type === 'logic' ? 'modelDegradation.form.taskLogic' : 'modelDegradation.form.taskRender')

// 本次将要提交的题目。自定义题与预制题互斥，所以每个检测项只会贡献其中一种；
// 页签单选，form.tasks 里恒为当前这一类，循环就是取这一类。
const selectedQuestions = computed<ModelDegradationQuestionPayload[]>(() => {
  const questions: ModelDegradationQuestionPayload[] = []
  for (const type of form.tasks) {
    if (customEnabled[type]) {
      const prompt = customPrompt[type].trim()
      if (prompt) {
        questions.push({
          task_type: type,
          prompt,
          expected: type === 'logic' ? customExpected.value.trim() : undefined
        })
      }
      continue
    }
    // 预制题单选：有选中就提交那一道的 key，由后端取库里的正文与期望答案。
    const key = presetKeys[type]
    if (key) questions.push({ task_type: type, key })
  }
  return questions
})

// 等待卡片上显示的名字：预制题用短标题，自定义题用通用名。
const pendingLabelFor = (question: ModelDegradationQuestionPayload) => {
  if (!question.key) return t('modelDegradation.form.customQuestion')
  const option = taskBank.value[question.task_type].find((item) => item.key === question.key)
  return option?.title ?? question.key
}

// 题目区右上那行字：走自定义题时报「自定义题目」，否则报选中那道预制题的
// 短标题（单选，标题比「已选 1 道」更能说明选的是哪道），没选就报未选择。
const presetSummary = (type: ModelDegradationTaskType) => {
  if (customEnabled[type]) return t('modelDegradation.form.customQuestion')
  const key = presetKeys[type]
  if (!key) return t('modelDegradation.form.presetNone')
  return taskBank.value[type].find((option) => option.key === key)?.title ?? t('modelDegradation.form.presetNone')
}

// 卡片是否不可选：检测进行中，或这一项走自定义题。
// 单选下不存在「数量到上限」这种锁 —— 点另一张卡就是换选，不是加选。
// 与模板里的 aria-disabled 用的是同一个判据 —— 两处若写歪了会出现「能点但无效」。
const cardLocked = (type: ModelDegradationTaskType, _key: string) => busy.value || customEnabled[type]

const isDetailOpen = (type: ModelDegradationTaskType, key: string) =>
  openDetails.value.includes(`${type}:${key}`)

// 「详情」只切正文显隐；点它不该顺带把卡片选中/取消，所以模板里用了 @click.stop。
const toggleDetail = (type: ModelDegradationTaskType, key: string) => {
  const id = `${type}:${key}`
  const index = openDetails.value.indexOf(id)
  if (index >= 0) {
    openDetails.value.splice(index, 1)
    return
  }
  openDetails.value.push(id)
}

const togglePreset = (type: ModelDegradationTaskType, key: string) => {
  // 单选：点已选中的那张 = 取消选择，点别的 = 换选。
  presetKeys[type] = presetKeys[type] === key ? '' : key
}

const toggleCustom = (type: ModelDegradationTaskType, event: Event) => {
  customEnabled[type] = (event.target as HTMLInputElement).checked
  // 走自定义题时预制题的勾选状态没有意义了，正文也不需要再展开。
  if (customEnabled[type]) openDetails.value = []
}

const loadTaskBank = async () => {
  loadingTasks.value = true
  try {
    taskBank.value = await modelDegradationAPI.listModelDegradationTasks()
    // 两类各自默认选第一道题：页签可以切来切去，切过去就得是「有题可跑」的状态，
    // 否则用户切到生成题会看到一个没选中的空卡组。
    presetKeys.logic = taskBank.value.logic[0]?.key ?? ''
    presetKeys.render = taskBank.value.render[0]?.key ?? ''
  } catch (error) {
    appStore.showError(extractApiErrorMessage(error, t('modelDegradation.errors.loadTasksFailed')))
  } finally {
    loadingTasks.value = false
  }
}

// 模型候选取自「用户填的那个上游」的 /models：检测打的是任意上游，
// 拿本站模型广场的列表去填，会选到上游根本没有的模型。
// 请求由服务端代理（浏览器直连第三方上游会被 CORS 挡掉）。
const modelOptions = ref<SelectOption[]>([])
const loadingModels = ref(false)
// 凭据/协议变化后自动取列表的防抖时长，避免边输边打上游。
const MODEL_LOAD_DEBOUNCE_MS = 600
let modelLoadTimer: ReturnType<typeof setTimeout> | null = null
// 请求序号：只认最后一次的结果，防止慢响应盖掉新响应。
let modelLoadSeq = 0

// 轮询句柄与次数。检测在服务端异步执行，前端按固定间隔取回状态；
// 次数上限兜底，避免服务端异常时无限轮询。
// 上限必须盖过后端的整体超时（当前 8 分钟：1 道逻辑题 + 1 道生成题 + 收尾余量），
// 否则后端还没落最终结果，前端就先放弃轮询了。
const POLL_INTERVAL_MS = 3000
const POLL_MAX_ATTEMPTS = 240
let pollTimer: ReturnType<typeof setTimeout> | null = null
let pollAttempts = 0

const busy = computed(() => running.value)
const canSubmit = computed(
  () =>
    !busy.value &&
    form.baseUrl.trim() !== '' &&
    form.apiKey.trim() !== '' &&
    (form.modelId ?? '').trim() !== '' &&
    form.tasks.length > 0 &&
    selectedQuestions.value.length > 0
)

const hasUpstreamCredentials = computed(
  () => form.baseUrl.trim() !== '' && form.apiKey.trim() !== ''
)

// 凭据不全时上游无从查起，检测进行中也别去打扰上游。
const fetchModelsDisabled = computed(() => loadingModels.value || busy.value || !hasUpstreamCredentials.value)

// Anthropic Messages 不接受 reasoning_effort（它用 thinking.budget_tokens），
// 该档位对它是无效参数，因此置灰而不是让它看着能设。
const reasoningEffortDisabled = computed(() => busy.value || form.protocol === 'messages')

// 生成题超时只对生成题生效：逻辑题的回答只有一句话，单题预算固定 60s。
// 当前页签是逻辑题时置灰而不是隐藏 —— 藏在页签后面会让人以为这个设置不存在。
const renderTimeoutDisabled = computed(() => busy.value || !form.tasks.includes('render'))

const modelPlaceholder = computed(() => {
  if (loadingModels.value) return `${t('common.loading')}...`
  if (!hasUpstreamCredentials.value) return t('modelDegradation.form.modelNeedCredentials')
  return t('modelDegradation.form.modelPlaceholder')
})

const currentEffortLabel = computed(() => effortLabel(form.reasoningEffort))

// 生成题判定文案。历史卡片、悬停面板与结果面板共用一份口径，
// 所以抽成函数而不是各写一个 switch。
const renderVerdictLabel = (verdict: ModelDegradationItemVerdict | '') => {
  switch (verdict) {
    case 'ok':
      return t('modelDegradation.metrics.renderOk')
    case 'degraded':
      return t('modelDegradation.metrics.renderDegraded')
    case 'unknown':
      return t('modelDegradation.metrics.renderUnknown')
    case 'generated':
      return t('modelDegradation.metrics.renderGenerated')
    default:
      return t('modelDegradation.metrics.renderSkipped')
  }
}

const renderLabel = computed(() => renderVerdictLabel(currentRun.value?.render_verdict ?? ''))

/** 已完成的检测里判定正常的占比；没有已完成记录时显示占位符。 */
const passRate = computed(() => {
  const done = history.value.filter((run) => run.status === 'completed')
  if (done.length === 0) return '—'
  const normal = done.filter((run) => run.verdict === 'normal').length
  return `${Math.round((normal / done.length) * 100)}%`
})

const lastDurationLabel = computed(() => {
  const last = history.value.find((run) => run.status === 'completed')
  return last ? formatDuration(last.duration_ms) : '—'
})

/** 检测结果 → 监控条配色；历史倒序，因此这里也倒着取。 */
const barClassOf = (run: ModelDegradationRun) => {
  if (run.status === 'running') return 'empty'
  switch (run.verdict) {
    case 'normal':
      return 'passed'
    case 'degraded':
      return 'incorrect'
    default:
      return 'error'
  }
}

// 左侧补空条，保证最新一次始终在最右端。
const bars = computed(() => {
  const recent = history.value.slice(0, BAR_COUNT)
  const out: string[] = []
  for (let i = recent.length; i < BAR_COUNT; i++) out.push('empty')
  for (let i = recent.length - 1; i >= 0; i--) out.push(barClassOf(recent[i]))
  return out
})

/* ---- 检测历史：题型筛选 + 判定筛选 + 分组筛选 + 翻页 ---- */
// 四件事都在客户端做：列表接口只给了 limit，没有 offset / status / 题型 / 分组
// 参数，而历史一次最多 BAR_COUNT 条，本地过滤足够，不为此去动后端。

const HISTORY_FILTERS = ['all', 'passed', 'degraded', 'unknown', 'failed'] as const
type HistoryFilter = (typeof HISTORY_FILTERS)[number]

const historyFilter = ref<HistoryFilter>('all')

const historyFilterOptions = computed<SelectOption[]>(() =>
  HISTORY_FILTERS.map((value) => ({
    value,
    label: t(`modelDegradation.history.filter.${value}`)
  }))
)

// 分组桶：优先用命中的平台分组名；没命中分组的（第三方上游 Key）按上游主机名
// 归桶 —— 与卡片缎带同一口径，筛选列表与归类展示里每条记录都有明确归属，
// 不会把各个第三方上游全堆进同一个「未知」桶。
const GROUP_BUCKET_NONE = '__none__'

const groupBucketOf = (run: ModelDegradationRun): { key: string; label: string } => {
  if (run.group_name) return { key: 'g:' + run.group_name, label: run.group_name }
  const host = run.base_url.replace(/^https?:\/\//i, '').split('/')[0]
  if (host) return { key: 'u:' + host, label: host }
  return { key: GROUP_BUCKET_NONE, label: t('modelDegradation.history.groupFilter.none') }
}

// 与后端 hidden host 的匹配口径一致：小写，去 scheme、路径、端口。
const normalizedHostOf = (baseUrl: string): string => {
  let host = baseUrl.trim().toLowerCase()
  const scheme = host.indexOf('://')
  if (scheme >= 0) host = host.slice(scheme + 3)
  host = host.split(/[/?#]/)[0]
  const port = host.indexOf(':')
  if (port >= 0) host = host.slice(0, port)
  return host
}

// 管理员视角的「已隐藏」标记：普通用户的列表由后端过滤掉这些分组 / 上游主机，
// 管理员看到的仍是全量数据 —— 这里读一次管理端配置，在对应分节上标出来。
// 配置只用于标记，读取失败不影响历史墙本身。
const hiddenHistoryGroupIds = ref<Set<number>>(new Set())
const hiddenHistoryHosts = ref<Set<string>>(new Set())

const loadHiddenHistoryConfig = async () => {
  if (!authStore.isAdmin) return
  try {
    const config = await adminAPI.modelDegradation.getModelDegradationHistoryVisibility()
    hiddenHistoryGroupIds.value = new Set(config.hidden_group_ids ?? [])
    hiddenHistoryHosts.value = new Set((config.hidden_hosts ?? []).map(normalizedHostOf))
  } catch (error) {
    console.error('Failed to load model audit visibility config:', error)
  }
}

const isHiddenHistoryBucket = (runs: ModelDegradationRun[]) => {
  if (!authStore.isAdmin) return false
  if (!hiddenHistoryGroupIds.value.size && !hiddenHistoryHosts.value.size) return false
  return runs.some((run) =>
    run.group_id != null
      ? hiddenHistoryGroupIds.value.has(run.group_id)
      : hiddenHistoryHosts.value.has(normalizedHostOf(run.base_url))
  )
}

const historyGroupFilter = ref('all')

// 选项从已加载的历史现算：按记录条数倒序，最常测的分组排最上面。
const historyGroupOptions = computed<SelectOption[]>(() => {
  const buckets = new Map<string, { label: string; count: number }>()
  for (const run of history.value) {
    const bucket = groupBucketOf(run)
    const entry = buckets.get(bucket.key)
    if (entry) entry.count += 1
    else buckets.set(bucket.key, { label: bucket.label, count: 1 })
  }
  const options = [...buckets.entries()]
    .sort((a, b) => b[1].count - a[1].count || a[1].label.localeCompare(b[1].label))
    .map(([value, entry]) => ({ value, label: entry.label }))
  return [{ value: 'all', label: t('modelDegradation.history.groupFilter.all') }, ...options]
})

const matchHistoryGroup = (run: ModelDegradationRun, filter: string) =>
  filter === 'all' || groupBucketOf(run).key === filter

// 题型筛选：生成题认 render_verdict（跑过生成题才有这个判定），逻辑题认
// logic_total。还在跑的记录要到收尾才结算这两个字段、题型未知 —— 任何题型下
// 都保留它，否则刚发起的检测会立刻从墙上消失。默认生成题：近期的检测都以
// 作品墙为主，逻辑题记录切过去看。
const historyTypeFilter = ref<ModelDegradationTaskType>('render')

const historyTypeOptions = computed<SelectOption[]>(() =>
  (['render', 'logic'] as const).map((value) => ({ value, label: taskTypeLabel(value) }))
)

const matchHistoryType = (run: ModelDegradationRun, filter: ModelDegradationTaskType) =>
  run.status === 'running' || (filter === 'render' ? run.render_verdict !== '' : run.logic_total > 0)

// 「请求失败」在历史里不是一个独立的 status：整轮请求都打不通时，后端仍把这一轮
// 记成 completed，只是整体 verdict 落成 unknown、逐题明细带上游失败前缀。
// 所以按「completed + unknown + 生成题没有产出」认请求失败 —— 只跑生成题且跑通时
// render_verdict 是 generated（那种记录是「已生成、不做降智判定」，不算失败）。
// status=failed 只留给服务重启兜底与执行 panic，也一并收进这一档。
const isRequestFailed = (run: ModelDegradationRun) => {
  if (run.status === 'failed') return true
  if (run.status !== 'completed') return false
  return run.verdict === 'unknown' && run.render_verdict !== 'generated'
}

const matchHistoryFilter = (run: ModelDegradationRun, filter: HistoryFilter) => {
  switch (filter) {
    case 'passed':
      return run.verdict === 'normal'
    case 'degraded':
      return run.verdict === 'degraded'
    case 'failed':
      return isRequestFailed(run)
    case 'unknown':
      // 「无法判定」用排除法认：既不是通过/降智，也不是请求失败。这样它只装
      // 「只跑了生成题（已生成）」「用户中止」两类，不会和请求失败混为一谈，
      // 也就不会误报成「全是网络问题」。仍在进行的记录不进任何具体档。
      return (
        run.status !== 'running' &&
        !isRequestFailed(run) &&
        run.verdict !== 'normal' &&
        run.verdict !== 'degraded'
      )
    default:
      return true
  }
}

const filteredHistory = computed(() =>
  history.value.filter(
    (run) =>
      matchHistoryType(run, historyTypeFilter.value) &&
      matchHistoryFilter(run, historyFilter.value) &&
      matchHistoryGroup(run, historyGroupFilter.value)
  )
)

/* ---- 检测历史：卡片时间点与单行横向滚动 ---- */
// 时间戳 → 节点上的「月-日 时:分」：每个节点都带完整日期，隔天不用回看上下文。
// 日期走 i18n（zh 是 10-13，en 是 10/13），时间走 Intl 的 hour / minute 拆分。
const timelineDateTimeLabel = (value: string) => {
  const date = new Date(value)
  const day = t('modelDegradation.history.timeline.date', {
    month: date.getMonth() + 1,
    day: date.getDate()
  })
  return `${day} ${formatDateTime(value, { hour: '2-digit', minute: '2-digit', hour12: false })}`
}

// 时间线悬停联动：节点与卡片是同一份记录的两端，悬停任一端另一端点亮。
const timelineHoverId = ref<number | null>(null)

// 鼠标移入（或键盘聚焦）时展开的那张卡，只记 id；浮卡坐标见 hoverCardStyleOf。
const zoomId = ref<number | null>(null)

// 分组行是横向滚动容器：整组卡片铺成一行、不换行，铺不下就左右滚（按钮在节头）。
// 每行的滚动余量现量现用，按钮据此显隐、置灰。
const rowScroll = ref<Record<string, { left: boolean; right: boolean; overflow: boolean }>>({})
const historyWallRef = ref<HTMLElement | null>(null)

const updateRowScroll = () => {
  const wall = historyWallRef.value
  if (!wall) return
  // 卡墙会被 v-if 摘下又挂回，观察目标要跟着最新的元素走（对同一元素重复 observe 是幂等的）。
  wallResizeObserver.observe(wall)
  const next: typeof rowScroll.value = {}
  for (const row of wall.querySelectorAll<HTMLElement>('.ma-group-row')) {
    const key = row.dataset.groupRow
    if (!key) continue
    next[key] = {
      left: row.scrollLeft > 1,
      right: row.scrollLeft + row.clientWidth < row.scrollWidth - 1,
      overflow: row.scrollWidth > row.clientWidth + 1
    }
  }
  // 值没变就不写回：滚动事件与 ResizeObserver 都在调它，别引出多余的渲染回路。
  const keys = Object.keys(next)
  const unchanged =
    keys.length === Object.keys(rowScroll.value).length &&
    keys.every((key) => {
      const a = rowScroll.value[key]
      const b = next[key]
      return !!a && !!b && a.left === b.left && a.right === b.right && a.overflow === b.overflow
    })
  if (!unchanged) rowScroll.value = next
}

const scrollGroupRow = (key: string, dir: -1 | 1) => {
  const rows = historyWallRef.value?.querySelectorAll<HTMLElement>('.ma-group-row') ?? []
  const row = [...rows].find((el) => el.dataset.groupRow === key)
  if (!row) return
  row.scrollBy({ left: dir * Math.round(row.clientWidth * 0.8), behavior: 'smooth' })
}

// 悬停浮卡：所在行是滚动容器，绝对定位会被行裁掉，所以浮卡用 fixed 定位逃出裁剪。
// 每张卡片的浮层坐标按「它上次展开时」记在表里 —— 收起的淡出要停在原处，
// 不能弹回静态位置；页面 / 行滚动时由 scheduleScrollSync 跟着刷新。
const zoomPositions = reactive(new Map<number, { left: number; top: number; width: number }>())

const hoverCardStyleOf = (id: number) => {
  const pos = zoomPositions.get(id)
  if (!pos) return undefined
  return { left: `${pos.left}px`, top: `${pos.top}px`, width: `${pos.width}px` }
}

const positionZoomCard = () => {
  const id = zoomId.value
  const tile =
    id == null ? null : historyWallRef.value?.querySelector<HTMLElement>(`.ma-tile[data-run-id="${id}"]`)
  if (!tile || id == null) return
  const rect = tile.getBoundingClientRect()
  // 上沿沿用老校准值：浮卡顶端探到瓦片顶点之上 22px，正好盖住 cap 里的
  // 时间标签与圆点（老绝对定位是 top: cap 高 38 - 60 = -22）。
  zoomPositions.set(id, { left: rect.left, top: rect.top - 22, width: rect.width })
}

let scrollFrame = 0
const scheduleScrollSync = () => {
  cancelAnimationFrame(scrollFrame)
  scrollFrame = requestAnimationFrame(() => {
    updateRowScroll()
    if (zoomId.value != null) positionZoomCard()
  })
}

const wallResizeObserver = new ResizeObserver(scheduleScrollSync)

const historyPage = ref(1)
const historyPageCount = computed(() =>
  Math.max(1, Math.ceil(filteredHistory.value.length / HISTORY_CARD_COUNT))
)

// 页码夹在有效区间内：切到条数更少的筛选后，原来停在第 3 页就越界了。
// 用计算值而不是在 watch 里回写 ref —— 后者要多一轮渲染才收敛，
// 且翻页按钮的 disabled 会先按越界的旧值算一次。
const historyPageSafe = computed(() =>
  Math.min(Math.max(1, historyPage.value), historyPageCount.value)
)

// 当前这一页的卡片。
const historyCards = computed(() => {
  const start = (historyPageSafe.value - 1) * HISTORY_CARD_COUNT
  return filteredHistory.value.slice(start, start + HISTORY_CARD_COUNT)
})

// 当前这一页按分组聚成若干节：节内保持时间倒序，节与节按组在页内首次出现的
// 顺序排（最近有动静的组排最前）。分组只做归类展示，不参与分页计数。
// nodes 与 runs 同序（新的在前）：每张卡片正上方挂一个时间点，悬停节点给
// 「时间 · 模型 · 分组」整串信息，点节点打开那一轮详情。
const historyGroups = computed(() => {
  const sections: {
    key: string
    label: string
    runs: ModelDegradationRun[]
    hidden: boolean
    nodes: { tone: string; latest: boolean; label: string; title: string }[]
  }[] = []
  const index = new Map<string, (typeof sections)[number]>()
  for (const run of historyCards.value) {
    const bucket = groupBucketOf(run)
    let section = index.get(bucket.key)
    if (!section) {
      section = { key: bucket.key, label: bucket.label, runs: [], hidden: false, nodes: [] }
      index.set(bucket.key, section)
      sections.push(section)
    }
    section.runs.push(run)
    section.nodes.push({
      tone: barClassOf(run),
      latest: section.runs.length === 1,
      label: timelineDateTimeLabel(run.created_at),
      title: `${formatDateTimeToMinute(run.created_at)} · ${run.model_id} · ${groupBucketOf(run).label}`
    })
  }
  // 标记只看整组：配置按分组 ID / 主机名隐藏，同一节里若混入同名的其它分组也一并提示。
  for (const section of sections) section.hidden = isHiddenHistoryBucket(section.runs)
  return sections
})

// 渲染之后对一次账：换页 / 筛选 / 容器宽度变化都会改行的滚动余量，
// 浮卡开着的话坐标也要跟着新布局走。
watch(historyGroups, () => {
  nextTick(scheduleScrollSync)
})

onMounted(() => {
  if (historyWallRef.value) wallResizeObserver.observe(historyWallRef.value)
  // 滚动不冒泡：挂捕获态监听才能同时收到页面滚动与行内横向滚动。
  window.addEventListener('scroll', scheduleScrollSync, { capture: true, passive: true })
  scheduleScrollSync()
})

onUnmounted(() => {
  wallResizeObserver.disconnect()
  window.removeEventListener('scroll', scheduleScrollSync, { capture: true })
  cancelAnimationFrame(scrollFrame)
})

// 换筛选回到第 1 页：否则从「全部」第 3 页切到只有 1 页的筛选、再切回来，
// 会莫名停在原来的第 3 页。三个筛选都要监听。
watch([historyTypeFilter, historyFilter, historyGroupFilter], () => {
  historyPage.value = 1
})

/* ============ 检测历史：卡片封面与悬停详情 ============ */

// 列表接口不返回题目明细（Response 里可能是整份 HTML，塞进列表会把接口拖垮），
// 封面与悬停面板要用就现拉。reactive 的 Map 让模板里的 get 也被追踪：
// 明细一到手，卡片自己就从「判定概览」换成「生成题封面」。
const detailCache = reactive(new Map<number, ModelDegradationRun>())
// 同一个 id 并发只拉一次：鼠标划过卡片的速度远快于接口，
// 不合并的话每张卡都会被连打好几次。
const detailPending = new Map<number, Promise<ModelDegradationRun | null>>()

const loadRunDetail = (id: number): Promise<ModelDegradationRun | null> => {
  const cached = detailCache.get(id)
  if (cached) return Promise.resolve(cached)
  const pending = detailPending.get(id)
  if (pending) return pending
  const request = modelDegradationAPI
    .getModelDegradationRun(id)
    .then((detail) => {
      // 还在跑的那一轮不进缓存：它这会儿没有 items，缓存下来会让「我的检测」
      // 与结果卡在这一轮跑完之后仍停在「检测中」的旧状态上。
      if (detail.status !== 'running') detailCache.set(id, detail)
      return detail
    })
    .catch(() => null)
    .finally(() => {
      detailPending.delete(id)
    })
  detailPending.set(id, request)
  return request
}

// 「这一轮有没有可预览的产物」：生成题不再做降智判定，只要源码里出现了
// SVG / Canvas 或 <!doctype html> 就直接当作作品预览出来。
const ARTWORK_CODE_REGEX = /<(svg|canvas)\b/i
const ARTWORK_DOCTYPE_REGEX = /<!doctype\s+html\s*>/gi
const ARTWORK_HTML_TAG_REGEX = /<html[\s>]/gi

// 取 limit 之前最后一处匹配（`g` 正则走 matchAll，不会在模块级正则上留 lastIndex 状态）。
const lastMatchBefore = (regex: RegExp, text: string, limit: number) => {
  let found: RegExpMatchArray | null = null
  for (const match of text.matchAll(regex)) {
    if ((match.index ?? limit) >= limit) break
    found = match
  }
  return found
}

// 生成题的原文几乎都不是干净的源码，而是「先说一段话 → 围栏里贴源码 → 再补一句」。
// 整段塞进 srcdoc，预览里就是一面文字墙 —— 实测一条真实回答是 445 字前言
// 加 10 KB 源码，而且源码在 iframe 里还被当成纯文本显示。所以预览只截源码本体：
//   ① 有文档标记就截到 </html>，顺带把围栏与前后的话丢掉；
//   ② 只有片段（裸 <svg> / <canvas>）就切到最后一个闭合标签。
// 截不出源码就返回空串，模板据此退回判定概览封面 —— 纯文字的回答不在预览里露面。
const extractArtworkHtml = (raw: string) => {
  if (!raw || !ARTWORK_CODE_REGEX.test(raw)) return ''
  const close = /<\/html\s*>/i.exec(raw)
  // 文档标记只在 </html> 之前找：历史题面带「以 <!DOCTYPE html> 开头」的字面量，
  // 模型在前言里复述一遍的话，取第一处会把那段复述当成文档开头，前言又跟着进预览。
  const limit = close ? close.index : raw.length
  const doctype = lastMatchBefore(ARTWORK_DOCTYPE_REGEX, raw, limit)
  const htmlTag = lastMatchBefore(ARTWORK_HTML_TAG_REGEX, raw, limit)
  // doctype 与 <html> 必须分开找、且优先用 doctype：合成一条正则取「最后」的话，
  // <html lang=...> 会盖过它前面的 doctype，截出来的源码少一行 doctype
  // （iframe 落进 quirks 模式），而原文那份 doctype 会被当成「源码之外的话」留在详情里。
  // doctype 后面必须紧跟标签（只隔空白），否则那是前言里的一句复述，不算文档开头。
  const doctypeIsHead =
    !!doctype && /^\s*</.test(raw.slice((doctype.index ?? 0) + doctype[0].length))
  const start = doctypeIsHead ? (doctype?.index ?? -1) : (htmlTag?.index ?? -1)
  if (start >= 0) {
    return (close ? raw.slice(start, close.index + close[0].length) : raw.slice(start)).trim()
  }
  const codeStart = raw.search(ARTWORK_CODE_REGEX)
  const endSvg = raw.lastIndexOf('</svg>')
  const endCanvas = raw.lastIndexOf('</canvas>')
  const fragEnd = Math.max(endSvg < 0 ? -1 : endSvg + 6, endCanvas < 0 ? -1 : endCanvas + 9)
  return raw.slice(codeStart, fragEnd > codeStart ? fragEnd : undefined).trim()
}

// 源码之外的那段话（整条都是纯文字时就是全文）。预览不显示它，
// 但详情里要能读到 —— 否则模型「说了什么」就彻底看不到了。
const extractArtworkText = (raw: string) => {
  if (!raw) return ''
  const html = extractArtworkHtml(raw)
  const rest = html ? raw.replace(html, '') : raw
  return rest.replace(/```[a-zA-Z]*/g, '').trim()
}

// 生成题的产物：第一条生成题里截出来的源码。逻辑题没有产物，返回空串。
// 优先用 run 自带的 items —— 选中的那一轮（结果卡 / 我的检测）手里就有明细；
// 列表接口不带 items，卡片才回退到按 id 拉的缓存。
//
// 预览**不冻动画**，直接放模型原样的作品：这道题要判的就是「骑车动作动没动」，
// 冻成一帧静态图等于把唯一要看的东西藏起来了。复制 / 新标签页打开的也是这份。
const renderHtmlOf = (run: ModelDegradationRun | null) => {
  if (!run) return ''
  const items = run.items ?? detailCache.get(run.id)?.items ?? []
  const item = items.find((entry) => entry.task_type === 'render')
  return item ? extractArtworkHtml(item.response) : ''
}

// 预览 iframe 的 srcdoc 补一段「藏滚动条」的样式：预览视口高度按实测作品高度设置，
// 正常情况下整页正好放下、不该出现滚动条；但测量回来之前会先按 16:10 画一版，
// 那一下页面可能比视口高，带沙箱的 iframe 会把系统滚动条亮出来 —— 缩略图上尤其脏。
// 只藏视觉滚动条、不锁滚动：万一测量失败（回退 800），大预览里仍能用滚轮看到下半页。
//
// 追加在文档尾部而不是插进 <head>：srcdoc 的 doctype / head 形状由模型决定，
// 往最前面插内容会把缺 doctype 的文档推进 quirks 模式；而且源码里若在脚本
// 字符串中出现字面量 </head>，插进去还可能割裂脚本。追加则无此风险。
const SCROLLBAR_HIDE_STYLE =
  '<style>html,body{scrollbar-width:none!important;-ms-overflow-style:none!important}' +
  'html::-webkit-scrollbar,body::-webkit-scrollbar{width:0!important;height:0!important;display:none!important}</style>'

// —— 裸 SVG / Canvas 片段的自适应包装 ——
// 现版题面只要一个独立 SVG，截出来的是裸 <svg> 片段。直接当 srcdoc 时，浏览器
// 按它自己的 width/height 在视口左上角原尺寸画 —— 比 1280x800 小的作品右侧、
// 下方就是一大片白。包一层铺满逻辑视口的文档：
//   - 带 viewBox 的 SVG 按 preserveAspectRatio 等比居中缩进整屏，观感与
//     manxue-ai 用 <img object-fit:contain> 一致；
//   - 缺 viewBox 只有像素宽高时按宽高补一个 viewBox，否则拉大元素也不会缩放
//     （没有 viewBox 就没有缩放基准）。
// 整页 HTML（旧题面）保持原样，由作品自己的版式铺满视口。
const ARTWORK_FRAGMENT_DOC_STYLE =
  '<style>html,body{margin:0;width:100%;height:100%;overflow:hidden}' +
  'body>svg{display:block;width:100%!important;height:100%!important}' +
  'body>canvas{display:block;width:100%;height:100%}</style>'

const ARTWORK_SVG_OPEN_TAG_REGEX = /^<svg\b[^>]*>/i
const ARTWORK_SVG_VIEWBOX_REGEX = /\bviewBox\s*=/i

// 只认纯数字（可带 px）：百分数或带别的单位的尺寸补出的 viewBox 不合法，宁可不补。
const readArtworkSvgPixelSize = (openTag: string) => {
  const readAttr = (name: string) => {
    const match = new RegExp(`(?:^|\\s)${name}\\s*=\\s*(?:"([^"]*)"|'([^']*)')`, 'i').exec(openTag)
    const value = (match?.[1] ?? match?.[2] ?? '').trim()
    if (!/^\d+(?:\.\d+)?(?:px)?$/i.test(value)) return ''
    return value.replace(/px$/i, '')
  }
  const width = readAttr('width')
  const height = readAttr('height')
  return width && height ? `0 0 ${width} ${height}` : ''
}

const wrapArtworkFragment = (fragment: string) => {
  const openTag = ARTWORK_SVG_OPEN_TAG_REGEX.exec(fragment)?.[0] ?? ''
  let artwork = fragment
  if (openTag && !ARTWORK_SVG_VIEWBOX_REGEX.test(openTag)) {
    const viewBox = readArtworkSvgPixelSize(openTag)
    if (viewBox) {
      artwork = openTag.replace(/^(<svg)\b/i, `$1 viewBox="${viewBox}"`) + fragment.slice(openTag.length)
    }
  }
  return (
    '<!doctype html><html><head><meta charset="utf-8">' +
    `${ARTWORK_FRAGMENT_DOC_STYLE}</head><body>${artwork}</body></html>`
  )
}

// 裸片段包成铺满视口的文档；整页 HTML 原样。藏滚动条样式两者都追加。
const artworkSrcdoc = (html: string) => {
  if (!html) return ''
  const source = html.trimStart()
  const document = /^<(?:svg|canvas)\b/i.test(source) ? wrapArtworkFragment(source) : html
  return `${document}${SCROLLBAR_HIDE_STYLE}`
}

// —— 作品预览的等比缩放 ——
// 整页作品（旧题面的 HTML）几乎都是「桌面上一屏」的版式（min-height:100vh + 居中
// 场景，实测两个真实样例都是）。直接把 iframe 塞进小缩略图时，它的内部视口就是
// 框的大小（可能只有 240px 宽），页面按小视口重排、内容比视口高，结果只能看到一角。所以预览里的
// iframe 统一按 1280px 宽的逻辑视口排版，外层 SVG（viewBox + foreignObject）
// 把整页等比缩进预览框 —— 卡片封面、悬停大卡、详情预览三处一个形制。
//
// 高度多数就是 800（1280x800 正好 16:10，满框不留白）；个别作品比 800 高
// （实测鹈鹕页 830px，硬按 800 裁会切掉页脚）。所以量出真实高度后把 viewBox
// 跟着调高，超出 16:10 的部分由 preserveAspectRatio=meet 换成两侧留白。
// 测量放在隐藏的同源 iframe（sandbox 只多给 allow-same-origin，脚本照旧禁跑），
// 串行一个个量，避免一次铺开十几份文档渲染。
const ARTWORK_VIEW_WIDTH = 1280
const ARTWORK_VIEW_HEIGHT = 800

const artworkViewHeights = ref<Record<string, number>>({})
const artworkMeasureFrame = ref<HTMLIFrameElement | null>(null)
const artworkMeasureQueue: string[] = []
let artworkMeasuring = false

const drainArtworkMeasureQueue = async () => {
  if (artworkMeasuring) return
  artworkMeasuring = true
  try {
    while (artworkMeasureQueue.length) {
      const frame = artworkMeasureFrame.value
      if (!frame) break
      const html = artworkMeasureQueue.shift() as string
      if (html in artworkViewHeights.value) continue
      const height = await new Promise<number>((resolve) => {
        const onLoad = () => {
          frame.removeEventListener('load', onLoad)
          try {
            const doc = frame.contentDocument
            resolve(
              Math.max(
                ARTWORK_VIEW_HEIGHT,
                doc?.documentElement?.scrollHeight ?? 0,
                doc?.body?.scrollHeight ?? 0,
              ),
            )
          } catch {
            resolve(ARTWORK_VIEW_HEIGHT)
          }
        }
        frame.addEventListener('load', onLoad)
        frame.srcdoc = artworkSrcdoc(html)
      })
      artworkViewHeights.value[html] = height
    }
  } finally {
    artworkMeasuring = false
  }
}

const queueArtworkViewMeasure = (html: string) => {
  if (html in artworkViewHeights.value || artworkMeasureQueue.includes(html)) return
  artworkMeasureQueue.push(html)
  void drainArtworkMeasureQueue()
}

// viewBox 高度跟着实测走；测量回来之前先按 16:10 画，回来后自动校正。
const artworkViewHeight = (html: string) => {
  const height = artworkViewHeights.value[html]
  if (height !== undefined) return height
  if (html) queueArtworkViewMeasure(html)
  return ARTWORK_VIEW_HEIGHT
}

const artworkViewBox = (html: string) =>
  `0 0 ${ARTWORK_VIEW_WIDTH} ${artworkViewHeight(html)}`

// 详情里那一行「模型回答」：逻辑题给原样回答，生成题只给源码之外的话
// （源码本体在预览、复制源码、展开预览里，整份贴进详情就是一面代码墙）。
const itemAnswerText = (item: ModelDegradationItem) =>
  item.task_type === 'logic' ? item.response : extractArtworkText(item.response)

// 当前选中那一轮的产物预览。结果卡与「我的检测」都用它，抽出来免得两处各算一遍。
const artworkPreviewHtml = computed(() => renderHtmlOf(currentRun.value))

// 卡片封面要用明细：生成题取作品源码，纯逻辑题取「请求回了什么」的补充说明。
// 串行而不是并发：一次铺开二十个请求会把接口拖慢，也会把限流打出来。
const prefetchTileDetails = async () => {
  for (const run of historyCards.value) {
    if (detailCache.has(run.id)) continue
    await loadRunDetail(run.id)
  }
}

// 翻页 / 换筛选后要为这一页新露面的卡片补明细：封面（作品源码、判定概览的
// 补充说明）都要 items 才画得出来，而列表接口不带 items。
// 首屏那一次由 loadHistory 直接调，这里的 watch 只管后续翻页。
watch([historyPageSafe, historyTypeFilter, historyFilter, historyGroupFilter], () => {
  void prefetchTileDetails()
})

const thumbTone = (run: ModelDegradationRun) => {
  if (run.status === 'running') return 'is-running'
  if (run.status === 'cancelled') return 'is-cancelled'
  if (run.verdict === 'normal') return 'is-normal'
  if (run.verdict === 'degraded') return 'is-degraded'
  return 'is-unknown'
}

const tileIcon = (run: ModelDegradationRun) => {
  if (run.status === 'running') return 'sync'
  if (run.status === 'cancelled') return 'ban'
  if (run.verdict === 'normal') return 'checkCircle'
  if (run.verdict === 'degraded') return 'exclamationCircle'
  return 'infoCircle'
}

// 吉祥物按模型名分角色：同一个模型永远摊到同一只动物，历史里一眼能认出「又是这条狗」。
const mascotAnimal = (seed: string): 'dog' | 'monkey' => {
  let hash = 0
  for (let i = 0; i < seed.length; i += 1) hash = (hash * 31 + seed.charCodeAt(i)) >>> 0
  return hash % 2 === 0 ? 'dog' : 'monkey'
}

const mascotState = (run: ModelDegradationRun): 'thinking' | 'pass' | 'fail' | 'unknown' => {
  if (run.status === 'running') return 'thinking'
  if (run.verdict === 'normal') return 'pass'
  if (run.verdict === 'degraded') return 'fail'
  return 'unknown'
}

// 封面中间那行字：逻辑题的得分，没跑逻辑题就报生成题判定。
const tileSummary = (run: ModelDegradationRun) => {
  if (run.status === 'running') return t('modelDegradation.resultPanel.running')
  if (run.logic_total > 0) {
    return `${t('modelDegradation.metrics.logic')} ${run.logic_passed}/${run.logic_total}`
  }
  if (run.render_verdict) return renderVerdictLabel(run.render_verdict)
  return '—'
}

// 后端写死的上游失败前缀（model_degradation_service.go 的 runLogicTask /
// runRenderTask）。它是后端常量，不随前端语言变化，所以这里按字面量匹配。
const UPSTREAM_FAILURE_PREFIX = '上游请求失败：'

// 把上游报错收敛成一句固定诊断，卡面上不回显上游响应原文。
//
// 为什么不原文照登：一是响应体里可能夹带网关回显的请求凭据 —— 后端 logredact
// 只抹 api_key / authorization / x-api-key 三个键，网关把整段请求塞回 body 时抹不干净；
// 二是「Upstream HTTP/2 stream failed」这类原文对看历史的人毫无意义，它说不清该找谁，
// 而检测历史是共享的，所有人都会看到。因此只按错误性质给一句固定说法。
//
// 只收敛带上游失败前缀的文本：item.detail 同时承载判定说明（「命中答案 21」、
// 「未出现期望答案 …」），那些不是错误，必须原样展示，否则会把判分理由吞掉。
const upstreamDiagnosis = (detail: string): string => {
  const raw = detail.trim()
  if (!raw.startsWith(UPSTREAM_FAILURE_PREFIX)) return raw

  // 非 2xx：后端拼成「上游返回 HTTP <code>：<body 片段>」，只取状态码，body 一律丢弃。
  const status = raw.match(/上游返回 HTTP (\d{3})/)
  if (status) {
    const code = Number(status[1])
    if (code === 401) return t('modelDegradation.upstream.authFailed')
    if (code === 403) return t('modelDegradation.upstream.forbidden')
    if (code === 404) return t('modelDegradation.upstream.notFound')
    if (code === 408) return t('modelDegradation.upstream.timeout')
    if (code === 429) return t('modelDegradation.upstream.rateLimited')
    if (code === 504 || code === 524) return t('modelDegradation.upstream.gatewayTimeout')
    if (code >= 500) return t('modelDegradation.upstream.serverError')
    return t('modelDegradation.upstream.generic')
  }

  // 网络层失败。HTTP/2 stream closed、连接重置、EOF 的根因都在上游或中间网关，
  // 使用者唯一有效的动作就是重跑一次，所以统一说成「连接中断」。
  if (/http2|stream closed|stream error|connection reset|broken pipe|reset by peer|\bEOF\b/i.test(raw)) {
    return t('modelDegradation.upstream.streamBroken')
  }
  if (/读取上游响应失败|IncompleteRead/i.test(raw)) {
    return t('modelDegradation.upstream.responseIncomplete')
  }
  if (/超时|timeout|timed out/i.test(raw)) {
    return t('modelDegradation.upstream.timeout')
  }
  if (/额度不足|insufficient_quota/i.test(raw)) {
    return t('modelDegradation.upstream.quotaExceeded')
  }
  if (/限流|rate_limit/i.test(raw)) {
    return t('modelDegradation.upstream.rateLimited')
  }
  if (/上游返回空响应|\bempty\b/i.test(raw)) {
    return t('modelDegradation.upstream.emptyResponse')
  }
  return t('modelDegradation.upstream.generic')
}

// 没有产物可预览时（纯逻辑题、或上游直接报错），光靠一个图标加「逻辑题 0/1」
// 看不出这一轮到底发生了什么：上游 400、超时、模型答了什么，全都不在卡面上。
// 这里补一段「请求到底回了什么」—— 优先模型的回答，答不出来就给单题的失败原因
// （上游报错走 upstreamDiagnosis 收敛成固定诊断，判定说明原样展示），
// 再没有就退回 run 级的错误信息。
// 列表接口不返回 items，历史小卡回退到按 id 拉的明细缓存；还没拉到就先不显示。
const tileReason = (run: ModelDegradationRun | null) => {
  if (!run || run.status === 'running') return ''
  const items = run.items ?? detailCache.get(run.id)?.items ?? []
  for (const item of items) {
    const answer = (
      item.task_type === 'logic' ? item.response : extractArtworkText(item.response)
    ).trim()
    if (answer) return answer
    const reason = item.detail.trim()
    if (reason) return upstreamDiagnosis(reason)
  }
  return run.error_message.trim()
}

// 大卡（我的检测 / 结果卡）在「没有作品可预览」时铺的判定概览。
// 逻辑题没有产物，卡面如果只剩一个图标加得分就太单调了，而且看不出错在哪 ——
// 这里把判定、得分、期望、模型回答、失败原因摊在同一张卡面上。
//
// 优先挑没通过的那道题：用户最关心的是「哪道错了」，全对时才挑第一道。
const logicFocusItem = (run: ModelDegradationRun | null): ModelDegradationItem | null => {
  if (!run) return null
  const items = run.items ?? detailCache.get(run.id)?.items ?? []
  const logicItems = items.filter((item) => item.task_type === 'logic')
  if (logicItems.length === 0) return null
  return logicItems.find((item) => !item.passed) ?? logicItems[0]
}

// 卡面主标题：跑完才谈判定，进行中与中止各有说法。
const logicVerdictLabel = (run: ModelDegradationRun) => {
  if (run.status === 'running') return t('modelDegradation.resultPanel.running')
  if (run.status === 'cancelled') return t('modelDegradation.resultPanel.cancelled')
  return t('modelDegradation.verdict.' + run.verdict)
}

// 卡面上那几行。没有的行直接不出现：跑着的那一轮还没有题目，
// 上游直接失败时模型回答是空的（此时由「失败原因」那一行说明）。
const logicRows = (run: ModelDegradationRun | null) => {
  const item = logicFocusItem(run)
  if (!item) return []
  const rows: { label: string; value: string }[] = []
  if (item.expected) {
    rows.push({ label: t('modelDegradation.result.expected'), value: item.expected })
  }
  const answer = itemAnswerText(item)
  rows.push({ label: t('modelDegradation.result.answer'), value: answer || '—' })
  const reason = tileReason(run)
  if (reason) rows.push({ label: t('modelDegradation.result.reason'), value: reason })
  return rows
}

// 缎带配色：文字换成了分组，但颜色仍表判定结果 —— 缩略图上这是唯一的判定信号。
// 中止与进行中不是判定结果，配色也跟着中性化。
const ribbonClass = (run: ModelDegradationRun) => {
  if (run.status === 'running') return 'is-running'
  if (run.status === 'cancelled') return 'is-cancelled'
  if (run.status === 'failed') return 'is-failed'
  // 只跑了生成题：封面就是作品本身，角标标「已生成」而不是「无法判定」。
  if (run.render_verdict === 'generated' && run.logic_total === 0) return 'is-generated'
  if (run.verdict === 'normal') return 'is-normal'
  if (run.verdict === 'degraded') return 'is-degraded'
  return 'is-unknown'
}

// 右上角缎带改显示「分组」：本次检测填的 Key 命中的平台分组名；没命中（填的是第三方
// 上游 Key，或该 Key 未绑分组）时改显示 Base URL，便于直接看出这次测的是哪个上游。
// 缎带只有 ~136px，完整 URL 放不下：去掉协议头再显示，把空间让给域名
// （共享历史里别人的地址后端已脱敏成 https://主机/***，去头后就是 主机/***）。
// 判定不再占用这个位置 —— 它由 ribbonClass 的颜色和卡片下方的判定提示表达。
const ribbonLabel = (run: ModelDegradationRun) => {
  if (run.group_name) return run.group_name
  if (run.base_url) return run.base_url.replace(/^https?:\/\//i, '')
  return t('modelDegradation.history.groupUnknown')
}

// 共享历史：标出是谁跑的。邮箱后端已脱敏（只留首字符与域名），前端不再加工。
// 用户被软删后 LEFT JOIN 不到邮箱，此时给个占位而不是显示空。
const ownerLabel = (run: ModelDegradationRun) =>
  run.owner_email || t('modelDegradation.history.ownerUnknown')

const showZoom = (run: ModelDegradationRun) => {
  zoomId.value = run.id
  // 浮卡 fixed 定位，坐标按卡片当前的位置 / 宽度现量（量在展开之前）。
  positionZoomCard()
  // 展开层的作品框要从明细里取源码；命中缓存时同步返回，不用等这个 promise。
  void loadRunDetail(run.id)
}

const hideZoom = () => {
  zoomId.value = null
}

// 焦点在这一块里挪动（卡片 → 展开层里的操作钮）不算离开：button 获得焦点时
// 卡片自己也会 blur；展开层与卡片是 tile 下的兄弟节点，所以判据挂在 tile 上，
// 用它的 contains 把两者都圈进来 —— 只按卡片自己算的话，一点操作钮展开层就收掉。
const handleCardFocusOut = (run: ModelDegradationRun, event: FocusEvent) => {
  const tile = event.currentTarget
  const next = event.relatedTarget
  if (next instanceof Node && tile instanceof Node && tile.contains(next)) return
  if (zoomId.value === run.id) hideZoom()
}

// 分享 = 把这一轮结论复制成一段纯文本。库里的记录本来就不含 API Key，
// 这段文本也只带判定与用量，可以直接贴到别处。
const copyRunSummary = (run: ModelDegradationRun) => {
  const verdict =
    run.status === 'cancelled'
      ? t('modelDegradation.resultPanel.cancelled')
      : (run.render_verdict === 'generated' && run.logic_total === 0)
        ? t('modelDegradation.verdict.generated')
        : t('modelDegradation.verdict.' + run.verdict)
  const text = [
    `${t('modelDegradation.history.title')} #${run.id}`,
    `${t('modelDegradation.history.verdictLabel')}: ${verdict}`,
    `${t('modelDegradation.chips.model')}: ${run.model_id}`,
    `${t('modelDegradation.metrics.protocol')}: ${t('modelDegradation.form.protocolLabel.' + run.protocol)}`,
    `${t('modelDegradation.metrics.reasoningEffort')}: ${effortLabel(run.reasoning_effort)}`,
    `${t('modelDegradation.metrics.logic')}: ${run.logic_passed}/${run.logic_total}`,
    `${t('modelDegradation.metrics.render')}: ${renderVerdictLabel(run.render_verdict)}`,
    `${t('modelDegradation.metrics.duration')}: ${formatDuration(run.duration_ms)}`,
    formatDateTime(run.created_at),
    `${t('modelDegradation.history.upstream')}: ${run.base_url}`,
    // 共享历史：复制出去的摘要也带上发起者，否则转发给别人就分不清是谁跑的。
    `${t('modelDegradation.history.owner')}: ${ownerLabel(run)}`,
    // 只有识别出平台分组时才带这一行（填第三方上游 Key 时没有分组）。
    ...(run.group_name ? [`${t('modelDegradation.history.groupLabel')}: ${run.group_name}`] : [])
  ].join('\n')
  void copyToClipboard(text, t('modelDegradation.history.copyOk'))
}

// 下载 = 把这一轮（含各题明细）存成 JSON。明细可能还没拉过
// （列表接口不带 items），先补齐再落盘。
const downloadRunReport = async (run: ModelDegradationRun) => {
  const detail = (await loadRunDetail(run.id)) ?? run
  const blob = new Blob([JSON.stringify(detail, null, 2)], { type: 'application/json' })
  const url = URL.createObjectURL(blob)
  const link = document.createElement('a')
  link.href = url
  link.download = `model-degradation-${run.id}.json`
  link.click()
  URL.revokeObjectURL(url)
}

// 复制 = 把生成题的源码整份带走。没有产物（纯逻辑题）时按钮是禁用的。
const copyArtwork = (run: ModelDegradationRun) => {
  const html = renderHtmlOf(run)
  if (!html) return
  void copyToClipboard(html, t('modelDegradation.result.artworkCopied'))
}

// 展开 = 把作品单独开一页看。用 Blob URL 而不是把源码写进新窗口：
// 新标签页拿不到我们的 srcdoc，Blob 是它唯一读得到的东西。
const openArtwork = (run: ModelDegradationRun) => {
  const html = renderHtmlOf(run)
  if (!html) return
  const url = URL.createObjectURL(new Blob([html], { type: 'text/html' }))
  window.open(url, '_blank', 'noopener')
  // 立刻 revoke 会让新标签页读到空文档；交给浏览器在页面卸载后回收。
  window.setTimeout(() => URL.revokeObjectURL(url), 60_000)
}

// 下载作品 = 把生成题的页面源码存成 .html 落盘。
// 与「下载报告」是两件事：那个下的是整轮检测记录（JSON），这个是模型吐出来的页面本身。
// 没有产物（纯逻辑题、或生成题请求失败）时按钮禁用，跟复制源码 / 打开作品一致。
const downloadArtwork = (run: ModelDegradationRun) => {
  const html = renderHtmlOf(run)
  if (!html) return
  const blob = new Blob([html], { type: 'text/html;charset=utf-8' })
  const url = URL.createObjectURL(blob)
  const link = document.createElement('a')
  link.href = url
  link.download = `model-degradation-${run.id}.html`
  // 先挂进文档再点：游离节点上的 .click() 在部分浏览器不触发下载。
  document.body.appendChild(link)
  link.click()
  link.remove()
  // revoke 挪到下一帧：紧跟 click 同步撤销，下载还没读到 blob 就被掐掉了。
  window.requestAnimationFrame(() => URL.revokeObjectURL(url))
}

/* ============ 我的检测：结果轮播 ============ */

// 轮播的数据源就是检测历史（最近在前）。源站那一栏是「我的检测」的列表，
// 一页一张，所以这里翻的是历史本身，不再本地另存一份。
// 结果卡只在「这一轮已经有结果」时出现：还在跑的那一轮没有判定可展示，
// 从历史墙点进去时也不该冒出一张空结果卡。判据写在模板的 v-if 里而不是抽成
// computed —— vue-tsc 靠 v-if 里的 `currentRun &&` 把 currentRun 收窄成非空，
// 换成 `v-if="showResult"` 之后整段模板都会报 possibly null。

const mineRuns = computed(() => history.value)

// 当前停在第几页。选中的那一轮还不在历史里（刚开始跑、列表没刷新）时落回第一页，
// 免得翻页器显示一个越界的页码。
const mineIndex = computed(() => {
  const id = currentRun.value?.id
  const index = mineRuns.value.findIndex((run) => run.id === id)
  return index >= 0 ? index : 0
})

const stepMine = (delta: number) => {
  if (running.value) return
  const next = mineRuns.value[mineIndex.value + delta]
  if (next) void openRun(next.id, { scroll: false })
}

const effortLabel = (effort: ModelDegradationReasoningEffort) =>
  effort ? formatReasoningEffort(effort) : t('modelDegradation.metrics.reasoningEffortUnset')

const formatDuration = (ms: number) => {
  if (!ms) return '—'
  if (ms < 1000) return `${ms} ms`
  return `${(ms / 1000).toFixed(1)} s`
}

// 结果面板里的「生成题超时」。库里存的是秒，界面上按分钟说 ——
// 下拉本身就是分钟粒度，回显成 900 秒反而要用户自己换算。
const renderTimeoutLabel = (seconds: number) =>
  t('modelDegradation.form.renderTimeoutUnit', { minutes: Math.round(seconds / 60) })

const itemBadgeClass = (item: ModelDegradationItem) => {
  if (item.verdict === 'ok') return 'is-ok'
  if (item.verdict === 'degraded') return 'is-degraded'
  if (item.verdict === 'generated') return 'is-generated'
  return 'is-unknown'
}

const itemBadgeLabel = (item: ModelDegradationItem) => {
  if (item.verdict === 'ok') return t('modelDegradation.item.passed')
  if (item.verdict === 'degraded') return t('modelDegradation.item.failed')
  if (item.verdict === 'generated') return t('modelDegradation.item.generated')
  return t('modelDegradation.item.unknown')
}

// 运行中的进度：已到题目数 / 本次提交的总题数。
// 不再逐题铺卡片 —— 生成题单轮能跑十几分钟（超时可调），用户要看的是「还要等多久、能不能停」。
const progressTotal = computed(() => pendingQuestions.value.length)
const arrivedCount = computed(() => {
  const items = currentRun.value?.items ?? []
  return items.filter((item) => item.task_type === 'logic' || item.task_type === 'render').length
})

// 秒表：按服务端 created_at 算已耗时，而不是本地记一个起点。
// 这样刷新页面后计时不会归零，也和「后端什么时候开始的」保持一致。
const nowTick = ref(Date.now())
const startedAtMs = computed(() => {
  const raw = currentRun.value?.created_at
  if (!raw) return null
  const parsed = Date.parse(raw)
  return Number.isNaN(parsed) ? null : parsed
})
const elapsedSeconds = computed(() => {
  if (startedAtMs.value === null) return 0
  return Math.max(0, Math.floor((nowTick.value - startedAtMs.value) / 1000))
})
const elapsedLabel = computed(() => {
  const total = elapsedSeconds.value
  if (total < 60) {
    return t('modelDegradation.resultPanel.seconds', { count: total })
  }
  return t('modelDegradation.resultPanel.minutesSeconds', {
    minutes: Math.floor(total / 60),
    seconds: total % 60
  })
})

// 第二行的「模型 · 思考等级」。运行刚开始时 currentRun 还是 null，
// 用表单里的值顶上，避免那一行空一下。
const runModelLabel = computed(() => currentRun.value?.model_id || (form.modelId ?? '').trim() || '—')
const runEffortLabel = computed(() =>
  effortLabel(currentRun.value?.reasoning_effort || form.reasoningEffort)
)

let tickTimer: number | null = null
const startTick = () => {
  stopTick()
  nowTick.value = Date.now()
  tickTimer = window.setInterval(() => {
    nowTick.value = Date.now()
  }, 1000)
}
const stopTick = () => {
  if (tickTimer !== null) {
    window.clearInterval(tickTimer)
    tickTimer = null
  }
}

const cancelling = ref(false)

// 客户端拦截器把错误统一成 { status, code, message }，不是 AxiosError，
// 所以判 409 只能按形状取 status，不能当成 Axios 实例来用。
const isHttpStatus = (error: unknown, status: number): boolean =>
  typeof error === 'object' &&
  error !== null &&
  (error as { status?: number }).status === status

// 中止一轮检测。成功后主动拉一次最新状态，不等下一轮轮询 ——
// 点了按钮界面没变化，最容易被当成按钮坏了。
const handleCancel = async () => {
  const run = currentRun.value
  if (!run || cancelling.value) return

  cancelling.value = true
  try {
    await modelDegradationAPI.cancelModelDegradationRun(run.id)
  } catch (error) {
    // 409 说明这一轮刚好在两次请求之间跑完了：按结果态展示即可，
    // 不用把「已经结束了」当成错误弹给用户。
    if (!isHttpStatus(error, 409)) {
      appStore.showError(extractApiErrorMessage(error, t('modelDegradation.errors.cancelFailed')))
    }
  } finally {
    cancelling.value = false
  }

  clearPoll()
  stopTick()
  try {
    const latest = await modelDegradationAPI.getModelDegradationRun(run.id)
    currentRun.value = latest
  } catch {
    // 拉不到就保留 409 时服务端已经写下的中止状态，由下一轮历史刷新兜底。
  }
  running.value = false
  await loadHistory()
}

// 检测项页签是单选：一轮只跑一类（逻辑题或生成题），点哪个就切到哪个。
const selectTask = (value: ModelDegradationTaskType) => {
  if (busy.value) return
  form.tasks = [value]
}

const clearPoll = () => {
  if (pollTimer !== null) {
    clearTimeout(pollTimer)
    pollTimer = null
  }
}

const pollRun = async (runId: number) => {
  pollAttempts += 1
  try {
    const run = await modelDegradationAPI.getModelDegradationRun(runId)
    currentRun.value = run

    if (run.status === 'running' && pollAttempts < POLL_MAX_ATTEMPTS) {
      pollTimer = setTimeout(() => void pollRun(runId), POLL_INTERVAL_MS)
      return
    }

    stopTick()
    running.value = false
    await loadHistory()
  } catch (error) {
    stopTick()
    running.value = false
    appStore.showError(extractApiErrorMessage(error, t('modelDegradation.errors.loadFailed')))
  }
}

const handleStart = async () => {
  if (!form.baseUrl.trim()) {
    appStore.showError(t('modelDegradation.errors.needBaseUrl'))
    return
  }
  if (!form.apiKey.trim()) {
    appStore.showError(t('modelDegradation.errors.needApiKey'))
    return
  }
  if (!(form.modelId ?? '').trim()) {
    appStore.showError(t('modelDegradation.errors.needModel'))
    return
  }
  if (form.tasks.length === 0) {
    appStore.showError(t('modelDegradation.errors.needTask'))
    return
  }
  // 生成题超时是 creatable 下拉，用户能输入任意字符串：先在本地说清哪不对，
  // 别让它变成一个看起来像上游故障的 400。只跑逻辑题时这项不生效，也就不校验。
  const needsRender = form.tasks.includes('render')
  const renderTimeoutMinutes = Number(form.renderTimeoutMinutes)
  if (
    needsRender &&
    (!Number.isInteger(renderTimeoutMinutes) ||
      renderTimeoutMinutes < MODEL_AUDIT_RENDER_TIMEOUT_MIN_MINUTES ||
      renderTimeoutMinutes > MODEL_AUDIT_RENDER_TIMEOUT_MAX_MINUTES)
  ) {
    appStore.showError(
      t('modelDegradation.errors.needRenderTimeout', {
        min: MODEL_AUDIT_RENDER_TIMEOUT_MIN_MINUTES,
        max: MODEL_AUDIT_RENDER_TIMEOUT_MAX_MINUTES
      })
    )
    return
  }
  if (selectedQuestions.value.length === 0) {
    appStore.showError(t('modelDegradation.errors.needQuestion'))
    return
  }
  // 逻辑题靠「回答里出现期望数字」判分，自定义题漏填期望答案就没法判，先拦在本地。
  if (customEnabled.logic && customPrompt.logic.trim() && !customExpected.value.trim()) {
    appStore.showError(t('modelDegradation.errors.needCustomExpected'))
    return
  }

  const questions = selectedQuestions.value

  running.value = true
  currentRun.value = null
  // 记下这次提交了哪些题，运行中才能铺出对应的等待卡片。
  pendingQuestions.value = questions.map((question) => ({
    key: question.key ?? 'custom',
    taskType: question.task_type,
    label: pendingLabelFor(question)
  }))
  clearPoll()

  try {
    const run = await modelDegradationAPI.startModelDegradation({
      base_url: form.baseUrl.trim(),
      api_key: form.apiKey.trim(),
      model_id: (form.modelId ?? '').trim(),
      protocol: form.protocol,
      // 不指定时不带该字段，避免把「上游不支持」变成一次 400。
      reasoning_effort: form.reasoningEffort || undefined,
      // 只跑逻辑题时不带：这项对逻辑题不生效，写进记录里只会是噪音。
      render_timeout_seconds: needsRender ? renderTimeoutMinutes * 60 : undefined,
      tasks: form.tasks,
      questions
    })
    currentRun.value = run
    pollAttempts = 0
    startTick()
    pollTimer = setTimeout(() => void pollRun(run.id), POLL_INTERVAL_MS)
  } catch (error) {
    stopTick()
    running.value = false
    pendingQuestions.value = []
    appStore.showError(extractApiErrorMessage(error, t('modelDegradation.errors.startFailed')))
  }
}

// 自动取列表时静默失败（边输凭据边打上游，报错会刷屏）；
// 用户主动点「获取模型」时要把失败原因说出来，否则点了没反应最容易被当成坏了。
const loadUpstreamModels = async (options: { silent?: boolean } = {}) => {
  const { silent = true } = options
  const baseUrl = form.baseUrl.trim()
  const apiKey = form.apiKey.trim()

  // 凭据不全就没法问上游：清空候选，Select 仍可手输。
  if (!baseUrl || !apiKey) {
    modelLoadSeq += 1
    modelOptions.value = []
    loadingModels.value = false
    return
  }

  const seq = ++modelLoadSeq
  loadingModels.value = true
  try {
    const models = await modelDegradationAPI.listUpstreamModels({
      base_url: baseUrl,
      api_key: apiKey,
      protocol: form.protocol
    })
    if (seq !== modelLoadSeq) return
    modelOptions.value = models.map((id) => ({ value: id, label: id }))
  } catch (error) {
    if (seq !== modelLoadSeq) return
    // 拉不到就退化成「空列表 + 手输」，不阻断检测本身。
    modelOptions.value = []
    if (silent) {
      console.error('Failed to load upstream models:', error)
    } else {
      appStore.showError(extractApiErrorMessage(error, t('modelDegradation.errors.modelsFailed')))
    }
  } finally {
    if (seq === modelLoadSeq) loadingModels.value = false
  }
}

// 「获取模型」按钮：先掐掉待触发的自动加载，避免同一次点击打两次上游。
const handleFetchModels = () => {
  if (modelLoadTimer !== null) {
    clearTimeout(modelLoadTimer)
    modelLoadTimer = null
  }
  void loadUpstreamModels({ silent: false })
}

// 地址、密钥或协议改完后延迟取一次，避免边输边打上游。
// 协议也在监听里：/models 的认证头随协议变，换了协议候选列表就得重新取。
const scheduleUpstreamModelsLoad = () => {
  if (modelLoadTimer !== null) clearTimeout(modelLoadTimer)
  modelLoadTimer = setTimeout(() => {
    modelLoadTimer = null
    void loadUpstreamModels()
  }, MODEL_LOAD_DEBOUNCE_MS)
}

watch(() => [form.baseUrl, form.apiKey, form.protocol], scheduleUpstreamModelsLoad)

const loadHistory = async () => {
  loadingHistory.value = true
  try {
    history.value = await modelDegradationAPI.listModelDegradationRuns(BAR_COUNT)
    // 生成题的封面要明细才有得画，这里顺手补上；失败只是封面退回概览态。
    void prefetchTileDetails()
    // 「我的检测」得有内容可看：还没选中任何一轮时默认落在最近一次上，
    // 与源站打开页面就能看到自己上一次检测一致。跳过还在跑的那一轮 ——
    // 它没有结果可展示，选中它只会让结果卡空着。
    const latest = history.value.find((run) => run.status !== 'running')
    if (!currentRun.value && latest) void openRun(latest.id, { scroll: false })
  } catch (error) {
    console.error('Failed to load model audit history:', error)
  } finally {
    loadingHistory.value = false
  }
}

// 选中一轮：结果卡与「我的检测」都跟着它走。
// scroll 只在用户主动点卡片时才要 —— 进页面时自动选中还顺带把页面顶上去会很突兀。
const openRun = async (id: number, options: { scroll?: boolean } = {}) => {
  const { scroll = true } = options
  try {
    // 走 loadRunDetail 而不是直接打接口：能复用「同 id 并发只拉一次」和缓存，
    // 与卡片的封面预取不会各请求一遍。
    const detail = await loadRunDetail(id)
    if (!detail) throw new Error('empty detail')
    currentRun.value = detail
    // 点开的是正在展示的这一轮（id 没变）时，结果区的 watch 不会响 ——
    // 但「点节点 / 卡片就该看到结果」在任何情况下都成立，这里直接展开。
    resultExpanded.value = true
    if (scroll) window.scrollTo({ top: 0, behavior: 'smooth' })
  } catch (error) {
    appStore.showError(extractApiErrorMessage(error, t('modelDegradation.errors.loadFailed')))
  }
}

onMounted(() => {
  void loadHistory()
  void loadTaskBank()
  // 管理员才需要「已隐藏」标记；普通用户的列表后端已经过滤，读配置没有意义。
  void loadHiddenHistoryConfig()
})

onUnmounted(() => {
  clearPoll()
  stopTick()
  if (modelLoadTimer !== null) {
    clearTimeout(modelLoadTimer)
    modelLoadTimer = null
  }
})
</script>

<style scoped>
/* 配色与尺寸对齐源站 manxue.ai 的「糖果测试」页：鼠尾草绿系 + 珊瑚橙点缀。
   深色模式给一套等价映射，避免浅底浅字。 */
.ma-page {
  --ma-panel: rgba(255, 255, 255, 0.7);
  --ma-panel-soft: rgba(237, 243, 239, 0.5);
  --ma-tile-bg: #eef1ec;
  --ma-border: #dfe6e2;
  --ma-border-soft: #dfe7e1;
  --ma-line: #e7ece8;
  --ma-input-bg: #f7faf8;
  --ma-input-border: #d7dfda;
  --ma-text: #242d2b;
  --ma-title: #30493d;
  --ma-label: #61746a;
  --ma-muted: #8a938d;
  --ma-faint: #8a988f;
  --ma-control-fg: #486454;
  --ma-primary: #154e43;
  --ma-primary-fg: #ffffff;
  --ma-pass: #54a77b;
  --ma-fail: #d8736b;
  --ma-warn: #e4bc50;
  --ma-bar-empty: #e5eae7;
  --ma-seg-bg: #edf2ee;
  --ma-seg-on-bg: #dcebe2;
  --ma-seg-on-border: #a3c8b5;
  --ma-seg-on-fg: #154e43;
  --ma-seg-on-shadow: 0 1px 2px rgba(21, 78, 67, 0.16);

  display: flex;
  flex-direction: column;
  gap: 26px;
  color: var(--ma-text);
  font-size: 14px;
  line-height: 1.5;
}

.dark .ma-page {
  --ma-panel: rgba(35, 40, 39, 0.72);
  --ma-panel-soft: rgba(30, 38, 35, 0.6);
  --ma-tile-bg: #242b28;
  --ma-border: #2c3532;
  --ma-border-soft: #2f3935;
  --ma-line: #2a3330;
  --ma-input-bg: #1d2221;
  --ma-input-border: #37413d;
  --ma-text: #e6ebe8;
  --ma-title: #cfe0d6;
  --ma-label: #9aaba0;
  --ma-muted: #8d9a93;
  --ma-faint: #7f8d86;
  --ma-control-fg: #b9cdc2;
  --ma-primary: #2f7d69;
  --ma-bar-empty: #262e2b;
  --ma-seg-bg: #202724;
  --ma-seg-on-bg: #2d4c3f;
  --ma-seg-on-border: #45775f;
  --ma-seg-on-fg: #d6ece0;
  --ma-seg-on-shadow: 0 1px 2px rgba(0, 0, 0, 0.4);
}

/* ============ 页头 ============ */
.ma-top {
  padding-bottom: 30px;
}

.ma-top-head {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 16px;
}

.ma-top-copy {
  min-width: 0;
}

.ma-top-title-line {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 6px 14px;
}

.ma-top-title {
  margin: 0;
  color: var(--ma-title);
  font-size: 21px;
  font-weight: 600;
  line-height: 1.4;
  white-space: nowrap;
}

.ma-top-sub {
  color: var(--ma-faint);
  font-size: 12px;
}

.ma-top-chips {
  display: flex;
  flex-wrap: wrap;
  gap: 6px 18px;
  margin-top: 8px;
}

.ma-chip {
  color: var(--ma-muted);
  font-size: 11px;
}

.ma-chip b {
  margin-left: 6px;
  color: var(--ma-text);
  font-weight: 600;
}

.ma-stats {
  display: grid;
  flex: 0 0 288px;
  grid-template-columns: repeat(3, minmax(0, 1fr));
  gap: 24px;
  width: 288px;
  max-width: 100%;
  margin: 0;
}

.ma-stat {
  display: flex;
  flex-direction: column;
  gap: 5px;
  min-width: 0;
}

.ma-stat dt {
  order: 1;
  color: var(--ma-muted);
  font-size: 11px;
  white-space: nowrap;
}

.ma-stat dd {
  margin: 0;
}

.ma-stat strong {
  display: block;
  color: var(--ma-text);
  font-size: 22px;
  font-weight: 500;
  font-variant-numeric: tabular-nums;
  line-height: 1.3;
  white-space: nowrap;
}

/* ============ 监控状态 ============ */
.ma-monitor-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 16px;
  margin-top: 18px;
}

.ma-monitor-title {
  display: flex;
  align-items: center;
  gap: 12px;
  font-size: 13px;
  font-weight: 600;
}

.ma-monitor-cap {
  color: var(--ma-faint);
  font-size: 11px;
  font-weight: 400;
}

.ma-icon-btn {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 32px;
  height: 32px;
  border: 1px solid var(--ma-border-soft);
  border-radius: 6px;
  background: transparent;
  color: var(--ma-control-fg);
  cursor: pointer;
  transition: border-color 0.15s ease, background 0.15s ease;
}

.ma-icon-btn:hover:not(:disabled) {
  border-color: var(--ma-input-border);
  background: var(--ma-input-bg);
}

.ma-icon-btn:disabled {
  cursor: default;
  opacity: 0.5;
}

.ma-bars {
  display: grid;
  grid-template-columns: repeat(60, minmax(0, 1fr));
  gap: 5px;
  height: 40px;
  margin-top: 10px;
}

.ma-bar {
  display: block;
  width: 100%;
  min-width: 0;
  height: 40px;
  border: 1px solid transparent;
  border-radius: 4px;
}

.ma-bar.empty {
  background: var(--ma-bar-empty);
}

.ma-bar.passed {
  background: var(--ma-pass);
}

.ma-bar.incorrect {
  background: var(--ma-fail);
}

.ma-bar.error {
  background: var(--ma-warn);
}

.ma-caption {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 10px;
  margin-top: 11px;
  color: var(--ma-faint);
  font-size: 10px;
}

.ma-legend {
  display: flex;
  gap: 16px;
}

.ma-legend span {
  display: inline-flex;
  align-items: center;
  gap: 5px;
  white-space: nowrap;
}

.ma-legend i {
  width: 5px;
  height: 10px;
  border-radius: 2px;
}

.ma-legend i.passed {
  background: var(--ma-pass);
}

.ma-legend i.error {
  background: var(--ma-warn);
}

.ma-legend i.incorrect {
  background: var(--ma-fail);
}

/* ============ 发起检测 ============ */
.ma-detect {
  padding: 26px 0;
  border-top: 1px solid var(--ma-border);
  border-bottom: 1px solid var(--ma-border);
}

.ma-panel {
  display: grid;
  grid-template-columns: minmax(0, 1fr) 380px;
  border: 1px solid var(--ma-border-soft);
  border-radius: 12px;
  background: var(--ma-panel);
}

.ma-form-col {
  display: flex;
  flex-direction: column;
  gap: 16px;
  min-width: 0;
  padding: 18px 26px;
}

.ma-form-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 20px;
  min-height: 34px;
}

.ma-h2 {
  margin: 0;
  color: var(--ma-title);
  font-size: 16px;
  font-weight: 600;
  line-height: 1.5;
}

.ma-h3 {
  margin: 0;
  color: var(--ma-title);
  font-size: 16px;
  font-weight: 600;
  line-height: 1.5;
}

.ma-seg {
  display: flex;
  gap: 3px;
  padding: 3px;
  border-radius: 7px;
  background: var(--ma-seg-bg);
}

.ma-seg-btn {
  padding: 3.5px 12px;
  border: 1px solid transparent;
  border-radius: 5px;
  background: transparent;
  color: var(--ma-muted);
  font-size: 12px;
  font-weight: 500;
  white-space: nowrap;
  cursor: pointer;
  transition: background 0.15s ease, color 0.15s ease, border-color 0.15s ease, box-shadow 0.15s ease;
}

/* 选中态必须一眼可辨：原来的「浅底白胶囊」跟轨道只差 10 个灰阶，
   实际渲染出来几乎看不出哪个是开的。改成品牌绿染色 + 绿描边 + 加粗 + 微投影。 */
.ma-seg-btn.is-active {
  background: var(--ma-seg-on-bg);
  border-color: var(--ma-seg-on-border);
  color: var(--ma-seg-on-fg);
  font-weight: 600;
  box-shadow: var(--ma-seg-on-shadow);
}

.ma-seg-btn:disabled {
  cursor: default;
  opacity: 0.55;
}

/* 题目选择区：每个选中的检测项一块。 */
.ma-questions {
  display: flex;
  flex-direction: column;
  gap: 10px;
  min-width: 0;
}

.ma-question {
  display: flex;
  flex-direction: column;
  gap: 6px;
  padding: 10px 12px;
  border: 1px solid var(--ma-border-soft);
  border-radius: 10px;
  background: var(--ma-panel-soft);
  min-width: 0;
}

.ma-question-head {
  display: flex;
  align-items: center;
  gap: 10px;
  min-width: 0;
}

/* 已选数量的角标：卡片本身就是选择控件，这里再给一个可扫的计数。 */
.ma-qcount {
  color: var(--ma-muted);
  font-size: 12px;
  font-variant-numeric: tabular-nums;
  white-space: nowrap;
}

.ma-qcustom-toggle {
  display: inline-flex;
  align-items: center;
  gap: 5px;
  margin-left: auto;
  color: var(--ma-muted);
  font-size: 12px;
  white-space: nowrap;
  cursor: pointer;
}

/* 题目卡片：点整张卡即选中。左边一个方形指示器，标题居中，详情按钮贴右下。 */
/* 卡片按内容收缩、按容器自动换列：minmax(240px, 320px) 把单张卡限制在
   240~320px，剩下的空间不分配（不是 1fr），所以不会出现「为铺满而拉宽」。
   940px 容器下一行放 3 张，右侧留白；窄容器自动降成 2 列 / 1 列。 */
.ma-qcards {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(240px, 320px));
  align-items: start;
  gap: 6px;
  min-width: 0;
}

.ma-qcard {
  display: grid;
  grid-template-columns: auto minmax(0, 1fr) auto;
  align-items: center;
  gap: 4px 10px;
  padding: 8px 10px;
  border: 1px solid var(--ma-input-border);
  border-radius: 8px;
  background: var(--ma-input-bg);
  cursor: pointer;
  transition: background 0.15s ease, border-color 0.15s ease;
}

.ma-qcard:hover {
  border-color: var(--ma-seg-on-border);
}

.ma-qcard.is-selected {
  border-color: var(--ma-seg-on-border);
  background: var(--ma-seg-on-bg);
}

.ma-qcard:focus-visible {
  outline: 2px solid var(--ma-seg-on-border);
  outline-offset: 1px;
}

.ma-qcard.is-disabled {
  cursor: default;
  opacity: 0.55;
}

.ma-qcard.is-disabled:hover {
  border-color: var(--ma-input-border);
}

.ma-qcard.is-selected.is-disabled:hover {
  border-color: var(--ma-seg-on-border);
}

/* 未选中是空框，选中是实心绿底 + 白勾 —— 只靠颜色区分对色弱不友好，
   所以形状也要变（空框 vs 实心带勾）。 */
.ma-qcard-check {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 16px;
  height: 16px;
  border: 1px solid var(--ma-input-border);
  border-radius: 5px;
  background: var(--ma-panel);
  color: var(--ma-seg-on-fg);
}

.ma-qcard.is-selected .ma-qcard-check {
  border-color: var(--ma-seg-on-border);
  background: var(--ma-seg-on-border);
}

.ma-qcard-title {
  min-width: 0;
  color: var(--ma-control-fg);
  font-size: 13px;
  font-weight: 500;
}

.ma-qcard.is-selected .ma-qcard-title {
  color: var(--ma-seg-on-fg);
}

/* 详情按钮在卡片上是「附属动作」而不是选择控件，所以做成低调的文字按钮；
   点它不能顺带改选中态（模板里 @click.stop）。 */
.ma-qcard-detail {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  padding: 2px 6px;
  border: 1px solid transparent;
  border-radius: 6px;
  background: transparent;
  color: var(--ma-muted);
  font-size: 12px;
  cursor: pointer;
  transition: background 0.15s ease, color 0.15s ease, border-color 0.15s ease;
}

.ma-qcard-detail:hover:not(:disabled) {
  border-color: var(--ma-input-border);
  background: var(--ma-panel);
  color: var(--ma-control-fg);
}

.ma-qcard-detail.is-open {
  color: var(--ma-seg-on-fg);
}

.ma-qcard.is-selected .ma-qcard-detail.is-open {
  background: var(--ma-panel);
  color: var(--ma-title);
}

/* 展开的正文独占新的一行（grid 第三行横跨整张卡），不会把标题挤成一列。
   窄卡（240~320px）下正文会按常规规则在标点/空格处自动换行，不设固定宽度。 */
.ma-qcard-prompt {
  grid-column: 1 / -1;
  margin: 2px 0 0;
  padding-top: 6px;
  border-top: 1px dashed var(--ma-input-border);
  color: var(--ma-muted);
  font-size: 12px;
  line-height: 1.7;
  white-space: pre-wrap;
}

.ma-qcustom {
  display: flex;
  flex-direction: column;
  gap: 10px;
}

.ma-qtextarea {
  width: 100%;
  padding: 8px 10px;
  border: 1px solid var(--ma-input-border);
  border-radius: 8px;
  background: var(--ma-input-bg);
  color: var(--ma-text);
  font-size: 13px;
  line-height: 1.6;
  resize: vertical;
}

.ma-qtextarea:focus {
  outline: none;
  border-color: var(--ma-seg-on-border);
}

.ma-qempty {
  margin: 0;
  color: var(--ma-faint);
  font-size: 12px;
}

.ma-fields {
  display: grid;
  grid-template-columns: minmax(0, 1fr) minmax(0, 1.15fr);
  gap: 18px;
  min-width: 0;
}

.ma-toolbar {
  display: flex;
  align-items: flex-end;
  justify-content: space-between;
  gap: 14px 16px;
  /* 四个下拉加两个按钮在窄卡片里放不下：允许换行，别把按钮挤到卡片外面。 */
  flex-wrap: wrap;
}

.ma-selects {
  display: grid;
  /* 四个下拉的宽度上限。用 minmax 而不是固定值：卡片在 1101px 附近最窄，
     固定宽会整块溢出（flex-wrap 只能挪整块，救不了比一行还宽的单个元素）。
     上限整体收窄过一轮：多了「生成题超时」这一列后，四个下拉的总宽一旦超过
     卡片内容宽，flex 换行会把两个按钮整块挤到下一行（换行按「不收缩的假设宽度」
     判定，跟能不能挤下无关）。收窄后仍保持原有的换行时机。 */
  grid-template-columns: minmax(0, 158px) minmax(0, 152px) minmax(0, 104px) minmax(0, 96px);
  gap: 10px;
}

.ma-field {
  display: flex;
  flex-direction: column;
  gap: 6px;
  min-width: 0;
}

.ma-label {
  color: var(--ma-label);
  font-size: 11px;
  line-height: 16px;
}

.ma-submit {
  display: flex;
  gap: 12px;
  /* 换行后按钮独占一行，靠右对齐而不是留在左边。 */
  margin-left: auto;
}

.ma-btn {
  display: inline-flex;
  align-items: center;
  gap: 7px;
  height: 36px;
  padding: 0 14px;
  border: none;
  border-radius: 6px;
  background: var(--ma-primary);
  color: var(--ma-primary-fg);
  font-size: 12.25px;
  font-weight: 500;
  white-space: nowrap;
  cursor: pointer;
  transition: opacity 0.15s ease;
}

.ma-btn:disabled {
  cursor: default;
  opacity: 0.5;
}

/* 次级按钮：与主按钮等高同圆角，只靠描边区分（源站 .visitor-preset-action 的写法）。 */
.ma-ghost {
  display: inline-flex;
  align-items: center;
  gap: 7px;
  height: 36px;
  padding: 0 12px;
  border: 1px solid var(--ma-border-soft);
  border-radius: 6px;
  background: transparent;
  color: var(--ma-control-fg);
  font-size: 12.25px;
  font-weight: 500;
  white-space: nowrap;
  cursor: pointer;
  transition:
    background 0.15s ease,
    opacity 0.15s ease;
}

.ma-ghost:hover:not(:disabled) {
  background: var(--ma-seg-bg);
}

.ma-ghost:disabled {
  cursor: default;
  opacity: 0.5;
}

/* 协议提示：说明当前协议会追加哪条路径、用哪套认证头，以及哪些参数不生效。 */
.ma-hint {
  margin: 0;
  color: var(--ma-muted);
  font-size: 11px;
  line-height: 1.7;
}

.ma-note {
  margin: 0;
  color: var(--ma-faint);
  font-size: 11px;
}

/* ============ 我的检测 ============ */
.ma-result-col {
  display: flex;
  flex-direction: column;
  min-width: 0;
  padding: 18px 20px;
  border-left: 1px solid var(--ma-border-soft);
  border-radius: 0 12px 12px 0;
  background: var(--ma-panel-soft);
}

/* 「我的检测」标题行：标题 + 账号名，右侧翻页器（与源站同形制）。 */
.ma-mine-head {
  display: flex;
  align-items: baseline;
  justify-content: space-between;
  gap: 8px 12px;
  min-height: 24px;
}

.ma-mine-id {
  display: flex;
  align-items: baseline;
  gap: 8px;
  min-width: 0;
}

.ma-mine-user {
  overflow: hidden;
  color: var(--ma-faint);
  font-size: 13px;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.ma-mine-pager {
  display: inline-flex;
  flex: none;
  align-items: center;
  gap: 2px;
}

.ma-mine-step {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 22px;
  height: 22px;
  padding: 0;
  border: 0;
  border-radius: 6px;
  background: transparent;
  color: var(--ma-muted);
  cursor: pointer;
  transition: background 0.15s ease, color 0.15s ease;
}

.ma-mine-step:hover:not(:disabled) {
  background: var(--ma-seg-bg);
  color: var(--ma-control-fg);
}

.ma-mine-step:disabled {
  color: var(--ma-bar-empty);
  cursor: default;
}

.ma-mine-page {
  min-width: 40px;
  color: var(--ma-muted);
  font-size: 12px;
  font-variant-numeric: tabular-nums;
  text-align: center;
}

.ma-mine-sub {
  margin: 2px 0 12px;
  overflow: hidden;
  color: var(--ma-faint);
  font-size: 11px;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.ma-result-body {
  display: flex;
  flex: 1;
  flex-direction: column;
  gap: 10px;
  min-height: 128px;
}

.ma-pending {
  display: grid;
  flex: 1;
  place-items: center;
  margin: 0;
  padding: 24px 16px;
  color: var(--ma-faint);
  font-size: 13px;
  text-align: center;
}

/* 运行中：一行「吉祥物 + 进行中 + 秒表 + 停止」，下面一行模型与思考等级。
   不做题目清单 —— 单轮最长 1 小时，用户要看的是「还要等多久、能不能停」。 */
.ma-running {
  display: flex;
  flex-direction: column;
  justify-content: center;
  gap: 14px;
  min-height: 220px;
  padding: 22px;
  border: 1px solid var(--ma-border-soft);
  border-radius: 12px;
  background: var(--ma-panel);
}

.ma-running-row {
  display: flex;
  align-items: center;
  gap: 10px;
  min-width: 0;
}

.ma-running-spin {
  display: inline-flex;
  flex: none;
  color: var(--ma-seg-on-border);
}

/* 转圈：图标本身不带动画，用 CSS 让它一直转，比换图标省事。 */
.ma-running-spin :deep(svg) {
  animation: ma-spin 1.1s linear infinite;
}

@keyframes ma-spin {
  to {
    transform: rotate(360deg);
  }
}

.ma-running-label {
  flex: none;
  color: var(--ma-title);
  font-size: 16px;
  font-weight: 600;
}

/* 有题目正在自动重试（含退避等待）时状态字切成琥珀色「重试中」；
   重试成功或彻底结束后的下一次轮询就恢复「进行中」。 */
.ma-running-label.is-retrying {
  color: #947112;
}

.dark .ma-running-label.is-retrying {
  color: #e0c580;
}

.ma-running-count {
  flex: 1 1 auto;
  min-width: 0;
  color: var(--ma-faint);
  font-size: 12px;
  font-variant-numeric: tabular-nums;
}

.ma-running-right {
  display: inline-flex;
  flex: none;
  align-items: center;
  gap: 10px;
}

/* 秒表。tabular-nums 是必须的：等宽数字才不会每秒跳一下把旁边的东西顶歪。 */
.ma-running-time {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  color: var(--ma-muted);
  font-size: 13px;
  font-variant-numeric: tabular-nums;
}

/* 停止按钮：一个描边小方块（与源站一致）。方块本身就是全部图标，
   不再加文字，宽度才压得住 12px 一档。 */
.ma-running-stop {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 26px;
  height: 26px;
  padding: 0;
  border: 1px solid var(--ma-input-border);
  border-radius: 6px;
  background: transparent;
  cursor: pointer;
  transition: background 0.15s ease, border-color 0.15s ease;
}

.ma-running-stop:hover:not(:disabled) {
  border-color: var(--ma-fail);
  background: var(--ma-panel-soft);
}

.ma-running-stop:disabled {
  cursor: default;
  opacity: 0.55;
}

.ma-running-stop-box {
  width: 9px;
  height: 9px;
  border-radius: 1px;
  background: var(--ma-control-fg);
}

.ma-running-stop:hover:not(:disabled) .ma-running-stop-box {
  background: var(--ma-fail);
}

.ma-running-model {
  margin: 0;
  overflow: hidden;
  color: var(--ma-muted);
  font-size: 13px;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.ma-running-dot {
  margin: 0 6px;
  color: var(--ma-faint);
}

/* 单张结果大卡：卡面上只有预览与右上斜角勋章 —— 判定由勋章表达，
   不再另起一枚「正常 / 疑似降智」胶囊（两处说同一件事，反而显得吵）。 */
.ma-mine-card {
  position: relative;
  padding: 10px;
  overflow: hidden;
  border: 1px solid var(--ma-border);
  border-radius: 16px;
  background: var(--ma-tile-bg);
}

.ma-mine-stage {
  position: relative;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 8px;
  aspect-ratio: 16 / 10;
  overflow: hidden;
  border: 1px solid var(--ma-border-soft);
  border-radius: 10px;
  background: var(--ma-panel-soft);
  color: var(--ma-muted);
}

/* 生成题的产物是模型自己吐的页面，底色由它自己定，容器统一铺白 ——
   与结果卡、历史封面保持同一套处理。 */
.ma-mine-stage.is-render {
  background: #ffffff;
}

.ma-mine-stage.is-normal .ma-card-glyph {
  color: var(--ma-pass);
}

.ma-mine-stage.is-degraded .ma-card-glyph {
  color: var(--ma-fail);
}

.ma-mine-stage.is-unknown .ma-card-glyph {
  color: var(--ma-warn);
}

.ma-mine-stage.is-cancelled .ma-card-glyph,
.ma-mine-stage.is-running .ma-card-glyph {
  color: var(--ma-faint);
}

.ma-verdict-hint {
  margin: 0;
  color: var(--ma-muted);
  font-size: 12px;
  line-height: 1.6;
}

.ma-error {
  margin: 0;
  color: #a34d48;
  font-size: 12px;
  line-height: 1.7;
  overflow-wrap: anywhere;
}

.ma-mini {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 10px 14px;
  margin: auto 0 0;
  padding-top: 12px;
  border-top: 1px solid var(--ma-line);
}

.ma-mini div {
  min-width: 0;
}

.ma-mini dt {
  color: var(--ma-muted);
  font-size: 10px;
}

.ma-mini dd {
  margin: 2px 0 0;
  color: var(--ma-text);
  font-size: 13px;
  font-weight: 600;
  font-variant-numeric: tabular-nums;
}

/* ============ 通用分节 ============ */
.ma-section {
  display: flex;
  flex-direction: column;
  gap: 16px;
  padding-top: 2px;
}

.ma-section-head {
  display: flex;
  align-items: baseline;
  gap: 8px 20px;
}

.ma-section-sub {
  overflow: hidden;
  color: var(--ma-faint);
  font-size: 11px;
  text-overflow: ellipsis;
  white-space: nowrap;
}

/* ============ 检测结果：折叠下拉条 ============ */
/* 铺满整行的抬头条：标题、上游地址都在条上，点整条收起 / 展开结果卡。
   条本身就是这块区域的抬头 —— 收起时留下的是它，不是一枚小开关。 */
.ma-result-heading {
  margin: 0;
}

.ma-result-bar {
  position: relative;
  display: flex;
  align-items: center;
  gap: 12px;
  width: 100%;
  padding: 11px 18px;
  border: 1px solid var(--ma-border);
  border-radius: 12px;
  background: var(--ma-panel);
  text-align: left;
  cursor: pointer;
  transition: border-color 0.15s ease, background 0.15s ease;
}

.ma-result-bar:hover {
  border-color: var(--ma-input-border);
  background: var(--ma-seg-bg);
}

/* 地址紧跟标题：封顶 32% 宽，长了走省略号，不会伸到中间的圆片底下。 */
.ma-result-bar-sub {
  flex: 0 1 auto;
  max-width: 32%;
  min-width: 0;
  overflow: hidden;
  color: var(--ma-faint);
  font-size: 11px;
  text-overflow: ellipsis;
  white-space: nowrap;
}

/* 居中开合箭头：不描边、不填底，和条融成一体 —— 状态全交给箭头朝向（收起转 -90°），
   悬停时箭头加深、再淡淡透出一圈底色，点按感还在，但不再是贴在条上的一枚圆扣。 */
.ma-result-bar-toggle {
  position: absolute;
  left: 50%;
  top: 50%;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 30px;
  height: 30px;
  border-radius: 50%;
  background: transparent;
  color: var(--ma-muted);
  transform: translate(-50%, -50%);
  transition: transform 0.25s ease, background 0.2s ease, color 0.15s ease;
}

.ma-result-bar:hover .ma-result-bar-toggle {
  background: var(--ma-seg-on-bg);
  color: var(--ma-seg-on-fg);
}

.ma-result-bar-toggle.is-collapsed {
  transform: translate(-50%, -50%) rotate(-90deg);
}

.ma-empty {
  margin: 0;
  padding: 24px 16px;
  color: var(--ma-faint);
  font-size: 13px;
  text-align: center;
}

/* ============ 检测结果：单张详情卡 ============ */
/* 结构对齐源站的访客检测详情：头部（名称 / 时间 / 操作）→ 大预览 → 指标页脚。
   页脚只放我们真有的：模型、思考等级、耗时、逻辑/生成判定、协议 ——
   上游不回 token 用量，所以不做源站那一行「输入 / 输出 / 总计 / 预估费用」。 */
.ma-detail {
  overflow: hidden;
  border: 1px solid var(--ma-border);
  border-radius: 14px;
  background: var(--ma-panel);
}

.ma-detail-head {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 12px;
  padding: 14px 18px;
}

.ma-detail-id {
  display: flex;
  flex-direction: column;
  gap: 2px;
  min-width: 0;
}

.ma-detail-name {
  overflow: hidden;
  color: var(--ma-title);
  font-size: 15px;
  font-weight: 600;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.ma-detail-time {
  color: var(--ma-faint);
  font-size: 11.5px;
  font-variant-numeric: tabular-nums;
}

/* 共享历史：发起者（邮箱已由后端脱敏）。 */
.ma-detail-owner {
  display: flex;
  align-items: center;
  gap: 4px;
  color: var(--ma-faint);
  font-size: 11.5px;
}

.ma-detail-actions {
  display: flex;
  flex: none;
  gap: 4px;
}

.ma-detail-actions button {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 28px;
  height: 28px;
  border: 1px solid transparent;
  border-radius: 8px;
  background: transparent;
  color: var(--ma-muted);
  cursor: pointer;
  transition: background 0.15s ease, color 0.15s ease;
}

.ma-detail-actions button:hover:not(:disabled) {
  background: var(--ma-seg-bg);
  color: var(--ma-control-fg);
}

/* 纯逻辑题没有产物，「复制源码」与「开新页看」没有对象可指 —— 置灰而不是点了没反应。 */
.ma-detail-actions button:disabled {
  opacity: 0.4;
  cursor: default;
}

/* 舞台：固定 16:10 的视口，让模型吐的页面按它自己的版式渲染。
   ⚠️ 这里的 max-width 不能换成 max-height —— 实测（Chrome）给带 aspect-ratio 的
   块级盒子加 max-height，宽度会被反过来按「钳制后的高度 × 比例」算出来：
   结果是 672px 宽的一张图缩在 1518px 的卡片里。要限高就只能限宽。 */
.ma-detail-stage {
  position: relative;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 8px;
  width: 100%;
  max-width: 900px;
  aspect-ratio: 16 / 10;
  margin: 0 auto;
  overflow: hidden;
  border: 1px solid var(--ma-border-soft);
  border-radius: 12px;
  background: var(--ma-panel-soft);
  color: var(--ma-muted);
}

/* 生成题的产物是模型自己吐的页面，底色由它自己定，容器统一铺白。 */
.ma-detail-stage.is-render {
  background: #ffffff;
}

.ma-detail-stage.is-normal .ma-card-glyph {
  color: var(--ma-pass);
}

.ma-detail-stage.is-degraded .ma-card-glyph {
  color: var(--ma-fail);
}

.ma-detail-stage.is-unknown .ma-card-glyph {
  color: var(--ma-warn);
}

.ma-detail-stage.is-cancelled .ma-card-glyph,
.ma-detail-stage.is-running .ma-card-glyph {
  color: var(--ma-faint);
}

/* 预览 iframe：尺寸交给外层舞台，它只负责铺满。
   结果卡与「我的检测」共用这一条。 */
.ma-frame {
  display: block;
  width: 100%;
  height: 100%;
  border: 0;
  background: #ffffff;
}

/* 作品预览的等比缩放器：iframe 永远按 1280px 宽的逻辑视口排版（页面按桌面版式
   画好），外层 SVG 用 viewBox + preserveAspectRatio=meet 把整页缩进预览框。
   比 16:10 高一截的作品会被整体缩小、两侧留白，而不是切掉一角 ——
   viewBox 高度取实测作品高度，见 artworkViewHeight。 */
.ma-artview {
  display: block;
  width: 100%;
  height: 100%;
}

/* 量作品高度用的隐藏探针 iframe：固定 1280x800 逻辑视口、整体挪出屏幕。
   不能用 display:none 藏 —— 不参与布局的文档量不到 scrollHeight。 */
.ma-art-measure {
  position: fixed;
  top: 0;
  left: -10000px;
  width: 1280px;
  height: 800px;
  border: 0;
  pointer-events: none;
}

.ma-detail-foot {
  display: flex;
  flex-direction: column;
  gap: 10px;
  padding: 14px 18px;
}

.ma-detail-foot-main {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 8px;
}

.ma-detail-model {
  color: var(--ma-title);
  font-size: 15px;
  font-weight: 600;
}

/* 思考等级：源站是个描边小胶囊，这里沿用同一形制。 */
.ma-detail-chip {
  padding: 1px 7px;
  border: 1px solid var(--ma-input-border);
  border-radius: 6px;
  color: var(--ma-muted);
  font-size: 11.5px;
}

.ma-detail-dur {
  display: inline-flex;
  align-items: center;
  gap: 5px;
  margin-left: auto;
  color: var(--ma-muted);
  font-size: 12px;
  font-variant-numeric: tabular-nums;
}

.ma-detail-stats {
  display: grid;
  grid-template-columns: repeat(3, minmax(0, 1fr));
  gap: 10px 16px;
  margin: 0;
  padding-top: 12px;
  border-top: 1px solid var(--ma-line);
}

.ma-detail-stats dt {
  color: var(--ma-muted);
  font-size: 10.5px;
}

.ma-detail-stats dd {
  margin: 2px 0 0;
  color: var(--ma-text);
  font-size: 13px;
  font-weight: 600;
  font-variant-numeric: tabular-nums;
}

/* 题目与回答：默认收起 —— 逐题铺开正文会把这页压成一面文字墙。 */
.ma-detail-more {
  padding: 0 18px 14px;
}

.ma-detail-more summary {
  color: var(--ma-muted);
  font-size: 12px;
  cursor: pointer;
}

.ma-qlist {
  display: flex;
  flex-direction: column;
  gap: 14px;
  margin: 12px 0 0;
  padding: 0;
  list-style: none;
}

.ma-qlist > li {
  padding: 12px 14px;
  border: 1px solid var(--ma-border-soft);
  border-radius: 10px;
  background: var(--ma-panel-soft);
}

.ma-item-meta {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 8px;
}

.ma-badge {
  display: inline-flex;
  align-items: center;
  padding: 2px 8px;
  border-radius: 5px;
  font-size: 11px;
  font-weight: 600;
  white-space: nowrap;
}

.ma-badge.is-ok {
  background: rgba(84, 167, 123, 0.14);
  color: #3f8a62;
}

.ma-badge.is-degraded {
  background: rgba(216, 115, 107, 0.14);
  color: #b5564e;
}

.ma-badge.is-unknown {
  background: rgba(228, 188, 80, 0.18);
  color: #947112;
}

.ma-badge.is-generated {
  background: rgba(90, 150, 200, 0.16);
  color: #2f6f9f;
}

.dark .ma-badge.is-ok {
  background: rgba(84, 167, 123, 0.2);
  color: #8fd0ab;
}

.dark .ma-badge.is-degraded {
  background: rgba(216, 115, 107, 0.2);
  color: #e6a49d;
}

.dark .ma-badge.is-unknown {
  background: rgba(228, 188, 80, 0.2);
  color: #e0c580;
}

.dark .ma-badge.is-generated {
  background: rgba(90, 150, 200, 0.22);
  color: #8ec5ec;
}

.ma-item-kind {
  color: var(--ma-faint);
  font-size: 11px;
}

.ma-item-dur {
  margin-left: auto;
  color: var(--ma-faint);
  font-size: 11px;
  font-variant-numeric: tabular-nums;
}

.ma-item-prompt {
  margin: 10px 0 0;
  color: var(--ma-text);
  font-size: 12.5px;
  line-height: 1.6;
  /* 题面可能带 Markdown 表格（糖果题的统计表），保留换行免得表格被折成一行。 */
  white-space: pre-wrap;
}

/* 期望答案 / 模型回答：标签压暗，值保持正文色，一行一项。 */
.ma-item-row {
  margin: 6px 0 0;
  color: var(--ma-text);
  font-size: 12px;
  line-height: 1.6;
  overflow-wrap: anywhere;
}

.ma-item-row b {
  margin-right: 6px;
  color: var(--ma-muted);
  font-weight: 500;
}

.ma-item-detail {
  margin: 4px 0 0;
  color: var(--ma-muted);
  font-size: 11.5px;
  line-height: 1.6;
}

/* ============ 检测历史 ============ */
/* 卡片对齐参考图的作品墙：一张预览铺满卡面，右上皮带勋章，左上一枚分享。
   预览分两种 —— 跑过生成题的把模型产出的 HTML 缩进来当封面；纯逻辑题
   没有可预览的产物，用判定概览顶一张同形制的封面，两种卡片高度才一致。 */
/* 筛选挂在标题行右端：margin-left:auto 把它顶到行尾，与标题共用一行。
   Select 的下拉 Teleport 到 body，不受这行 overflow 影响。 */
.ma-gallery-filter {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 8px;
  margin-left: auto;
}

.ma-gallery-filter .ma-label {
  white-space: nowrap;
}

/* 筛选框本身窄一点就够：只有五个固定选项，不跟着 .ma-selects 的宽度走。 */
.ma-gallery-filter :deep(.select-trigger) {
  width: 148px;
}

/* 分组筛选要装上游主机名，比判定筛选宽一档，长主机名才不至于统统省略号。 */
.ma-gallery-filter :deep(.ma-group-select .select-trigger) {
  width: 190px;
}

/* 翻页器：居中一行，形制照源站的画廊分页。历史一次最多 BAR_COUNT 条，
   分页是纯客户端的（列表接口只给 limit），所以这里只管画状态与两个按钮。 */
.ma-pager {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  justify-content: center;
  gap: 12px 16px;
}

/* 次级控件：与 .ma-ghost 同色系（描边 + 透明底），但比它矮半头，
   免得在列表末尾抢眼。 */
.ma-pager-btn {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  height: 32px;
  padding: 0 14px;
  border: 1px solid var(--ma-border-soft);
  border-radius: 6px;
  background: transparent;
  color: var(--ma-control-fg);
  font-size: 12px;
  font-weight: 500;
  white-space: nowrap;
  cursor: pointer;
  transition:
    background 0.15s ease,
    border-color 0.15s ease,
    opacity 0.15s ease;
}

.ma-pager-btn:hover:not(:disabled) {
  border-color: var(--ma-input-border);
  background: var(--ma-seg-bg);
}

.ma-pager-btn:focus-visible {
  outline: 2px solid var(--ma-primary);
  outline-offset: 2px;
}

.ma-pager-btn:disabled {
  cursor: default;
  opacity: 0.45;
}

/* 页码状态：数字用等宽字形，翻页时宽度不跳。 */
.ma-pager-status {
  color: var(--ma-muted);
  font-size: 12px;
  font-variant-numeric: tabular-nums;
  text-align: center;
}

/* ============ 卡片时间点：每张卡片正上方一个节点，连线垂到卡片 ============ */
/* 节点定在卡片正上方（水平居中），一行内均匀分布；下面一条细线垂到卡片上沿，
   把「点」和「卡」扣在一起；同一行相邻节点之间再用横线连成一条轴。
   行不换行（超出由行横向滚动），行首 / 行尾直接按序号挂类名（is-chain-left / right），
   CSS 只负责画线。节点配色沿用顶部监控条的同一套（passed / incorrect / error / empty）。 */
.ma-tile-cap {
  display: flex;
  align-items: flex-start;
  justify-content: center;
  height: var(--ma-cap-height);
}

.ma-group-node {
  position: relative;
  z-index: 1;
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 5px;
  margin: 0;
  /* 顶部 2px + 标签 12px + 间距 5px + 圆点 10px = 盒高 29；下面的垂线从 29 起。 */
  padding: 2px 0 0;
  border: none;
  background: transparent;
  cursor: pointer;
}

/* 垂线：从圆点落到卡片上沿（cap 底部），长度 = cap 高 - 节点盒高。 */
.ma-group-node::after {
  content: '';
  position: absolute;
  top: 100%;
  left: 50%;
  width: 1px;
  height: calc(var(--ma-cap-height) - 29px);
  background: var(--ma-line);
  transform: translateX(-50%);
}

.ma-group-node-time {
  color: var(--ma-faint);
  font-size: 10px;
  line-height: 12px;
  white-space: nowrap;
  font-variant-numeric: tabular-nums;
  transition: color 0.15s ease;
}

.ma-group-node-dot {
  width: 10px;
  height: 10px;
  border-radius: 50%;
  background: var(--ma-bar-empty);
  transition: transform 0.15s ease;
}

.ma-group-node.passed .ma-group-node-dot {
  background: var(--ma-pass);
}

.ma-group-node.incorrect .ma-group-node-dot {
  background: var(--ma-fail);
}

.ma-group-node.error .ma-group-node-dot {
  background: var(--ma-warn);
}

/* 最新一次大一号；悬停 / 与卡片联动时再放大一档。 */
.ma-group-node.is-latest .ma-group-node-dot {
  transform: scale(1.3);
}

.ma-group-node:hover .ma-group-node-dot,
.ma-group-node.is-bound .ma-group-node-dot {
  transform: scale(1.45);
}

.ma-group-node:focus-visible {
  outline: none;
}

.ma-group-node:focus-visible .ma-group-node-dot {
  outline: 2px solid var(--ma-primary);
  outline-offset: 2px;
}

.ma-group-node:hover .ma-group-node-time,
.ma-group-node.is-bound .ma-group-node-time {
  color: var(--ma-seg-on-fg);
}

/* 行内横线：圆点中心在 cap 顶下 24px（2 + 12 + 5 + 5），线就画在这个高度；
   两端各探出半个槽距，和左右邻居的线段在槽正中严丝合缝地接上。 */
.ma-tile.is-chain-left::after,
.ma-tile.is-chain-right::before {
  content: '';
  position: absolute;
  top: 24px;
  height: 1px;
  background: var(--ma-line);
}

.ma-tile.is-chain-right::before {
  left: 50%;
  right: calc(var(--ma-wall-gap) / -2);
}

.ma-tile.is-chain-left::after {
  right: 50%;
  left: calc(var(--ma-wall-gap) / -2);
}

/* 卡片侧的光环：悬停时间线节点时，同一条记录在墙上亮起来。 */
.ma-tile.is-bound .ma-card {
  border-color: var(--ma-seg-on-border);
  box-shadow: 0 0 0 2px color-mix(in srgb, var(--ma-primary) 22%, transparent);
}

/* 分组墙：每组一节，节头下面整组卡片铺成一行（.ma-group-row 横向滚动），
   多组上下堆叠；节头里的细线把组名和右侧的滚动按钮隔开。
   --ma-cap-height / --ma-wall-gap 是量给时间点用的：cap 高决定垂线长度，
   槽距决定行内横线的探出量，几处计算共用一套值。 */
.ma-groups {
  --ma-cap-height: 38px;
  --ma-wall-gap: 16px;
  display: flex;
  flex-direction: column;
  gap: var(--ma-wall-gap);
}

.ma-group-head {
  display: flex;
  align-items: center;
  gap: 8px;
  /* 换组时多留一点：上一组的卡片行与这一组的节头之间要有「分节」的呼吸感。 */
  margin-top: 6px;
}

.ma-group-head:first-child {
  margin-top: 0;
}

/* 分组名：左侧一条竖标（与判定配色同系的绿），字号比页面级标题小半头。 */
.ma-group-name {
  position: relative;
  margin: 0;
  padding-left: 10px;
  color: var(--ma-title);
  font-size: 13px;
  font-weight: 600;
  letter-spacing: 0.2px;
}

.ma-group-name::before {
  content: '';
  position: absolute;
  left: 0;
  top: 50%;
  width: 3px;
  height: 12px;
  border-radius: 2px;
  background: var(--ma-pass);
  transform: translateY(-50%);
}

/* 条数：小圆角胶囊，数字用等宽字形，翻页时宽度不跳。 */
.ma-group-count {
  padding: 1px 7px;
  border-radius: 999px;
  background: var(--ma-seg-bg);
  color: var(--ma-muted);
  font-size: 11px;
  font-variant-numeric: tabular-nums;
}

/* 管理员专属标记：这一节已在「降智检测可见性」里隐藏，普通用户看不到；
   管理员仍能看到全量记录。实心琥珀胶囊 + 标题删除线，扫一眼就能认出来。 */
.ma-group-hidden {
  padding: 2px 9px;
  border: 1px solid #d3a52f;
  border-radius: 999px;
  background: #f3c94f;
  color: #392a04;
  font-size: 11px;
  font-weight: 700;
  line-height: 16px;
  letter-spacing: 0.2px;
  white-space: nowrap;
  box-shadow: 0 1px 3px rgba(160, 118, 12, 0.35);
}

.dark .ma-group-hidden {
  border-color: #f0d27a;
  background: #e4bc50;
  color: #241a02;
  box-shadow: 0 0 0 1px rgba(228, 188, 80, 0.25), 0 1px 4px rgba(0, 0, 0, 0.45);
}

/* 隐藏分节的标题直接划掉并降成灰色：与胶囊配合，但不会盖住记录本身。 */
.ma-group-name-hidden {
  color: var(--ma-muted);
  text-decoration: line-through;
  text-decoration-color: rgba(184, 134, 11, 0.8);
  text-decoration-thickness: 2px;
}

.dark .ma-group-name-hidden {
  color: var(--ma-faint);
  text-decoration-color: rgba(228, 188, 80, 0.75);
}

/* 细线：把节头剩余宽度补满，充当分节的分隔线。 */
.ma-group-rule {
  flex: 1;
  height: 1px;
  background: var(--ma-line);
}

/* 行滚动按钮：只在整组一行铺不下时显现（v-show），按行左右滚一屏的 8 成。 */
.ma-group-nav {
  display: flex;
  flex: none;
  gap: 4px;
}

.ma-wall-nav {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 26px;
  height: 26px;
  padding: 0;
  border: 1px solid var(--ma-border-soft);
  border-radius: 6px;
  background: transparent;
  color: var(--ma-muted);
  cursor: pointer;
  transition:
    background 0.15s ease,
    border-color 0.15s ease,
    color 0.15s ease,
    opacity 0.15s ease;
}

.ma-wall-nav:hover:not(:disabled) {
  border-color: var(--ma-input-border);
  background: var(--ma-seg-bg);
  color: var(--ma-primary);
}

.ma-wall-nav:focus-visible {
  outline: 2px solid var(--ma-primary);
  outline-offset: 2px;
}

.ma-wall-nav:disabled {
  cursor: default;
  opacity: 0.45;
}

/* 卡片行：整组单行铺开、不换行，铺不下就横向滚。滚动条藏掉 ——
   左右导航交给节头的按钮，触控板横滑与滚轮仍照常可用。 */
.ma-group-row {
  display: flex;
  gap: var(--ma-wall-gap);
  overflow-x: auto;
  overscroll-behavior-x: contain;
  scrollbar-width: none;
}

.ma-group-row::-webkit-scrollbar {
  display: none;
}

/* 行不换行后卡宽不再由网格列宽决定：给固定卡宽，超出留给行滚动。 */
.ma-tile {
  position: relative;
  flex: 0 0 auto;
  width: 260px;
}

/* 卡面 = 浅灰绿底 + 内缩的预览 + 右上斜角勋章（源站作品墙同形制）。
   padding 是必要的：源站的预览是「镶」在卡里的一张小图，不是铺满卡面的背景。 */
.ma-card {
  position: relative;
  display: flex;
  flex-direction: column;
  width: 100%;
  padding: 10px;
  overflow: hidden;
  border: 1px solid var(--ma-border);
  border-radius: 18px;
  background: var(--ma-tile-bg);
  text-align: left;
  cursor: pointer;
  transition: border-color 0.15s ease, box-shadow 0.15s ease;
}

.ma-card:hover {
  border-color: var(--ma-input-border);
  box-shadow: 0 6px 18px rgba(21, 78, 67, 0.1);
}

.ma-card:focus-visible {
  outline: 2px solid var(--ma-primary);
  outline-offset: 2px;
}

/* 预览框：16:10，与源站一致。比原来的 4:3 更扁，一排能多放一列，
   也更接近生成页自身的版式比例 —— 页面按 16:10 视口画，缩略图才不走形。
   展开大卡里的作品框与卡片封面同形制，两处共用一套尺寸与裁切。 */
.ma-card-thumb,
.ma-hover-stage {
  position: relative;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 8px;
  aspect-ratio: 16 / 10;
  overflow: hidden;
  border: 1px solid var(--ma-border-soft);
  border-radius: 10px;
  background: var(--ma-panel-soft);
  color: var(--ma-muted);
}

/* 生成题的封面是模型自己吐的页面，底色由它自己定，容器统一铺白 ——
   与结果卡里 .ma-detail-stage.is-render 保持同一套处理。 */
.ma-card-thumb.is-render,
.ma-hover-stage.is-render {
  background: #ffffff;
}

/* 封面层：铺满作品框，内层 SVG（.ma-artview）负责把 1280px 逻辑视口缩进框里。
   不用 transform scale —— 直接缩放 iframe 时 Chrome 会按缩放后的尺寸重算内部
   视口，缩略图只剩一块（实测）；viewBox 缩放不动 iframe 的视口。 */
.ma-card-cover,
.ma-hover-cover {
  position: absolute;
  inset: 0;
  pointer-events: none;
}

.ma-card-frame,
.ma-hover-frame {
  display: block;
  width: 100%;
  height: 100%;
  border: 0;
}

.ma-card-glyph-text {
  color: var(--ma-label);
  font-size: 12px;
  font-weight: 500;
}

/* 没有产物可预览时补的那段说明（模型答了什么 / 上游为什么失败）。
   卡面尺寸差得多 —— 历史小卡只有 240px 宽 —— 超过三行就截断，
   别把上面的图标和得分挤没了。上游报错原文可能是一长串不带空格的 JSON。
   它是纯逻辑卡上唯一的正文：分数只说对了几道，答的是什么全在这行，
   所以不能沿用灰色小字 —— 垫一张纸片底、加深加粗，让答案一眼可读。 */
.ma-card-glyph-note {
  display: -webkit-box;
  max-width: 88%;
  margin: 0;
  padding: 4px 10px;
  overflow: hidden;
  border: 1px solid var(--ma-border-soft);
  border-radius: 8px;
  background: var(--ma-panel);
  color: var(--ma-text);
  font-size: 12.5px;
  font-weight: 600;
  line-height: 1.5;
  text-align: center;
  overflow-wrap: anywhere;
  -webkit-box-orient: vertical;
  -webkit-line-clamp: 3;
}

/* 答案按判定着色：通过 → 绿，降智 → 红，请求失败 / 无法判定 → 琥珀警告色。
   色相跟着判定走（与上面的 .ma-card-glyph 同一套变量）；文字用变量与正文色
   调和，保证浅底上的对比度比纯色更稳；纸片底、描边也各带一点同色，
   让「答对了 / 答错了」在卡片墙上一眼可辨。 */
.ma-card-thumb.is-normal .ma-card-glyph-note,
.ma-hover-stage.is-normal .ma-card-glyph-note {
  border-color: color-mix(in srgb, var(--ma-pass) 36%, var(--ma-border-soft));
  background: color-mix(in srgb, var(--ma-pass) 9%, var(--ma-panel));
  color: color-mix(in srgb, var(--ma-pass) 68%, var(--ma-text));
}

.ma-card-thumb.is-degraded .ma-card-glyph-note,
.ma-hover-stage.is-degraded .ma-card-glyph-note {
  border-color: color-mix(in srgb, var(--ma-fail) 36%, var(--ma-border-soft));
  background: color-mix(in srgb, var(--ma-fail) 9%, var(--ma-panel));
  color: color-mix(in srgb, var(--ma-fail) 68%, var(--ma-text));
}

.ma-card-thumb.is-unknown .ma-card-glyph-note,
.ma-hover-stage.is-unknown .ma-card-glyph-note {
  border-color: color-mix(in srgb, var(--ma-warn) 40%, var(--ma-border-soft));
  background: color-mix(in srgb, var(--ma-warn) 10%, var(--ma-panel));
  color: color-mix(in srgb, var(--ma-warn) 58%, var(--ma-text));
}

/* 逻辑题没有作品可预览，两张大卡（我的检测 / 结果卡）改铺「判定概览」：
   判定 + 得分 + 期望 / 模型回答 / 失败原因，卡面自己就能看完一轮。
   两张卡宽度差得远（侧栏那张只有 ~280px），靠 max-width 与省略号自适应，
   行多了就被 .ma-mine-stage / .ma-detail-stage 的 overflow 裁掉而不是撑破卡面。 */
.ma-logic-verdict {
  color: var(--ma-text);
  font-size: 15px;
  font-weight: 700;
}

.ma-logic-score {
  color: var(--ma-label);
  font-size: 12px;
}

.ma-logic-rows {
  display: flex;
  flex-direction: column;
  gap: 3px;
  width: 92%;
  margin: 0;
  font-size: 11.5px;
}

.ma-logic-rows > div {
  display: flex;
  gap: 6px;
  min-width: 0;
}

.ma-logic-rows dt {
  flex: none;
  color: var(--ma-faint);
}

/* 期望/回答/原因都可能是长文本（上游报错原文尤其长），单行省略。 */
.ma-logic-rows dd {
  margin: 0;
  overflow: hidden;
  color: var(--ma-muted);
  text-overflow: ellipsis;
  white-space: nowrap;
}

.ma-card-thumb.is-normal .ma-card-glyph,
.ma-hover-stage.is-normal .ma-card-glyph {
  color: var(--ma-pass);
}

.ma-card-thumb.is-degraded .ma-card-glyph,
.ma-hover-stage.is-degraded .ma-card-glyph {
  color: var(--ma-fail);
}

.ma-card-thumb.is-unknown .ma-card-glyph,
.ma-hover-stage.is-unknown .ma-card-glyph {
  color: var(--ma-warn);
}

.ma-card-thumb.is-cancelled .ma-card-glyph,
.ma-card-thumb.is-running .ma-card-glyph,
.ma-hover-stage.is-cancelled .ma-card-glyph,
.ma-hover-stage.is-running .ma-card-glyph {
  color: var(--ma-faint);
}

/* 斜角勋章：压的是「作品框」的角 —— 源站那张详情卡里标题行在上面，
   缎带从作品框右上角斜插下来、被作品框的圆角裁断。
   rotate 45° 后探出作品框的部分由 .ma-card-thumb / .ma-hover-stage 的 overflow 裁掉，
   切口刚好落在圆角内侧。
   内容显示的是分组名（不再是判定），颜色仍按判定取，见 ribbonClass。 */
.ma-ribbon {
  position: absolute;
  /* 往内收：rotate(45deg) 后缎带的「左上端」和「右下端」都会探出作品框、被 overflow 裁掉，
     贴角越紧可见段越短。原来 top/right 都贴角时，136px 的缎带只剩约 75px 可见，
     「内部测试分组」这种六个字的名字就只露出中间一截。
     沿垂直于缎带的方向往卡内挪约 24px 后，右下端不再超出、只剩左上端被裁，
     可见段回到约 118px —— 六个字能完整放下。
     ⚠️ top 与 right 必须成对调：只挪 top，右下端仍在框外，可见段几乎不变。 */
  top: 24px;
  right: -18px;
  width: 136px;
  padding: 4px 0;
  overflow: hidden;
  transform: rotate(45deg);
  background: var(--ma-pass);
  color: #ffffff;
  font-size: 12px;
  font-weight: 700;
  text-align: center;
  letter-spacing: 0.5px;
  /* 分组名长度不可控：超宽就省略，别撑破角标。 */
  text-overflow: ellipsis;
  white-space: nowrap;
  box-shadow: 0 1px 3px rgba(0, 0, 0, 0.14);
}

.ma-ribbon.is-degraded {
  background: var(--ma-fail);
}

.ma-ribbon.is-unknown,
.ma-ribbon.is-failed {
  background: var(--ma-warn);
  color: #5c4712;
}

.ma-ribbon.is-cancelled,
.ma-ribbon.is-running {
  background: #97a29c;
}

.ma-ribbon.is-generated {
  background: #4a86c5;
  color: #fff;
}

/* ============ 悬停展开大卡 ============ */
/* 移入时另起一张比卡片更高的卡：标题行 / 作品框 / 指标页脚，形制照源站那张详情卡。
   行是横向滚动容器，会裁掉绝对定位的浮层，所以浮卡 fixed 定位逃出裁剪：
   top / left / width 由脚本按卡片的位置量出（见 hoverCardStyleOf），
   页面或行滚动时跟着刷新。上沿沿用老校准值：浮卡顶端探到瓦片顶点之上 22px
   （老绝对定位 top: cap 高 38 - 60 = -22），正好盖住 cap 里的时间标签与圆点。
   底色必须不透明：它要整个盖住下面的卡片，半透明会让封面 iframe 从字底下透出来。
   不展开时 visibility + pointer-events 一起收掉，操作钮不会变成
   「看不见但能 Tab 到」的暗桩。 */
.ma-hover-card {
  position: fixed;
  /* 必须压过 cap 里时间点的 z-index: 1：浮卡升起时要真正盖住
     它后面透出来的圆点 / 时间标签，而不是被它们穿过去。 */
  z-index: 30;
  display: flex;
  flex-direction: column;
  gap: 10px;
  padding: 12px 10px;
  border: 1px solid var(--ma-input-border);
  border-radius: 18px;
  background: var(--ma-tile-bg);
  box-shadow: 0 18px 44px rgba(18, 38, 32, 0.24);
  /* 展开层整卡可点（空白处点开详情，与卡片本体一致）：它盖住卡片后，
     不在这里声明 pointer 的话，光标会从小手掉回箭头，像突然不可点了。 */
  cursor: pointer;
  opacity: 0;
  visibility: hidden;
  /* 收起方向：比展开略快，鼠标划走要利落收掉，不拖残影。 */
  transform: translateY(10px) scale(0.97);
  pointer-events: none;
  transition: opacity 0.18s ease-in, transform 0.18s ease-in, visibility 0.18s;
}

.ma-tile.is-zoom .ma-hover-card {
  opacity: 1;
  visibility: visible;
  /* 展开方向：从缩略图上方 10px、97% 大小「长」到原位。曲线是加强版的
     ease-out（先快后慢），收尾不带机械的匀速感，放大过程更像浮起来。 */
  transform: translateY(0) scale(1);
  pointer-events: auto;
  transition:
    opacity 0.24s cubic-bezier(0.16, 1, 0.3, 1),
    transform 0.24s cubic-bezier(0.16, 1, 0.3, 1),
    visibility 0.24s;
}

/* 高度写死 48px：作品框要落在静止时缩略图的位置上，这个值直接进 .ma-hover-card 的
   top 计算，标题行一旦被长模型名撑高，悬停时预览就会跟着跳一下。 */
.ma-more-head {
  display: flex;
  flex: none;
  align-items: center;
  justify-content: space-between;
  gap: 8px;
  height: 48px;
}

.ma-more-title {
  display: flex;
  flex-direction: column;
  gap: 2px;
  min-width: 0;
}

.ma-more-name {
  overflow: hidden;
  color: var(--ma-title);
  font-size: 12.5px;
  font-weight: 600;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.ma-more-time {
  color: var(--ma-faint);
  font-size: 10.5px;
}

/* 共享历史：发起者（邮箱已由后端脱敏）。 */
.ma-more-owner {
  display: flex;
  align-items: center;
  gap: 4px;
  overflow: hidden;
  color: var(--ma-faint);
  font-size: 10.5px;
}

/* 「本人」标记：历史池里混着所有人的记录，用它区分自己跑的那几条。 */
.ma-owner-tag {
  flex: none;
  padding: 0 5px;
  border: 1px solid var(--ma-input-border);
  border-radius: 5px;
  color: var(--ma-muted);
  font-size: 10px;
  line-height: 15px;
}

.ma-more-actions {
  display: flex;
  flex: 0 0 auto;
  gap: 4px;
}

.ma-more-actions button {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 24px;
  height: 24px;
  border: 1px solid transparent;
  border-radius: 7px;
  background: transparent;
  color: var(--ma-muted);
  cursor: pointer;
  transition: background 0.15s ease, color 0.15s ease;
}

.ma-more-actions button:hover:not(:disabled) {
  background: var(--ma-seg-bg);
  color: var(--ma-control-fg);
}

/* 纯逻辑题没有产物，「下载作品」没有对象可指 —— 置灰而不是点了没反应。 */
.ma-more-actions button:disabled {
  opacity: 0.4;
  cursor: default;
}

/* 指标页脚：第一行是档位与耗时，第二行是两道题的判定。
   上游不回报 token 用量，所以源站那三行（输入/输出、总计、预估费用）这里没有。 */
.ma-more-foot {
  display: flex;
  flex-direction: column;
  gap: 8px;
  padding-top: 9px;
  border-top: 1px solid var(--ma-line);
}

.ma-more-row {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 8px;
}

.ma-more-chip {
  padding: 1px 6px;
  border-radius: 5px;
  background: var(--ma-seg-bg);
  color: var(--ma-control-fg);
  font-size: 11px;
  font-weight: 600;
}

.ma-more-dur {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  color: var(--ma-muted);
  font-size: 11.5px;
  font-variant-numeric: tabular-nums;
}

/* 底部两块统计：对齐参考图里「输入 / 输出」那一行。
   上游不回报 token 用量，这里只放平台自己算得出来的两项。 */
.ma-more-stats {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 10px;
  margin: 0;
}

.ma-more-stats dt {
  color: var(--ma-faint);
  font-size: 10px;
}

.ma-more-stats dd {
  margin: 2px 0 0;
  color: var(--ma-text);
  font-size: 12.5px;
  font-weight: 600;
  font-variant-numeric: tabular-nums;
}

/* ============ 组件覆写 ============ */
.ma-page :deep(.input-label) {
  margin-bottom: 8px;
  color: var(--ma-label);
  font-size: 11px;
  font-weight: 500;
  line-height: 16px;
}

.ma-page :deep(.input) {
  height: 38px;
  padding: 3.5px 10.5px;
  border-color: var(--ma-input-border);
  border-radius: 7px;
  background: var(--ma-input-bg);
  color: var(--ma-text);
  font-size: 12.25px;
}

.ma-page :deep(.input::placeholder) {
  color: var(--ma-faint);
}

.ma-page :deep(.input:focus) {
  border-color: var(--ma-primary);
  box-shadow: none;
}

.ma-page :deep(.input-hint) {
  margin-top: 6px;
  color: var(--ma-faint);
  font-size: 11px;
}

.ma-page :deep(.select-trigger) {
  height: 36px;
  padding: 0 10px;
  border-color: var(--ma-border-soft);
  border-radius: 6px;
  background: var(--ma-input-bg);
  color: var(--ma-control-fg);
  font-size: 12px;
}

.ma-page :deep(.select-trigger:hover) {
  border-color: var(--ma-input-border);
}

.ma-page :deep(.select-trigger.select-trigger-open) {
  border-color: var(--ma-primary);
  box-shadow: none;
}

.ma-page :deep(.select-value) {
  font-size: 12px;
}

/* 吉祥物是 100% 宽高的 SVG，直接丢进 flex 列会被拉满 —— 尺寸由外层 span 定死。 */
.ma-mascot {
  flex: none;
  width: 66px;
  height: 66px;
}

.ma-running-mascot {
  flex: none;
  width: 52px;
  height: 52px;
}

/* ============ 动态底纹 ============
   没有产物可预览的卡面只剩几行字，撑不住画面 —— 底下铺两层一直在动的底纹：
   一层判定色光晕缓慢漂移、一层网格匀速平移；跑着的那一轮网格换成自上而下的扫描带。
   颜色跟判定走：--ma-glow 由下面各 .is-* 规则给出，默认取灰。
   三层动画全走 transform，不碰 background-position —— 历史里可能有几十张卡，
   重绘动画会一起掉帧。 */

.ma-mine-stage,
.ma-detail-stage,
.ma-card-thumb,
.ma-hover-stage {
  /* 底纹层压在 z-index:-1，不建层就会掉到卡片自己的背景后面、彻底看不见。 */
  isolation: isolate;
  --ma-glow: var(--ma-faint);
}

.ma-mine-stage.is-normal,
.ma-detail-stage.is-normal,
.ma-card-thumb.is-normal,
.ma-hover-stage.is-normal {
  --ma-glow: var(--ma-pass);
}

.ma-mine-stage.is-degraded,
.ma-detail-stage.is-degraded,
.ma-card-thumb.is-degraded,
.ma-hover-stage.is-degraded {
  --ma-glow: var(--ma-fail);
}

.ma-mine-stage.is-unknown,
.ma-detail-stage.is-unknown,
.ma-card-thumb.is-unknown,
.ma-hover-stage.is-unknown {
  --ma-glow: var(--ma-warn);
}

/* 两层都比框大一圈（inset 负值），漂移 / 平移时才不会露出边。 */
.ma-mine-stage::before,
.ma-detail-stage::before,
.ma-card-thumb::before,
.ma-hover-stage::before,
.ma-mine-stage::after,
.ma-detail-stage::after,
.ma-card-thumb::after,
.ma-hover-stage::after {
  content: '';
  position: absolute;
  inset: -40%;
  z-index: -1;
  pointer-events: none;
}

.ma-mine-stage::before,
.ma-detail-stage::before,
.ma-card-thumb::before,
.ma-hover-stage::before {
  background-image:
    radial-gradient(
      34% 30% at 26% 28%,
      color-mix(in srgb, var(--ma-glow) 26%, transparent),
      transparent 70%
    ),
    radial-gradient(
      38% 34% at 74% 72%,
      color-mix(in srgb, var(--ma-glow) 16%, transparent),
      transparent 72%
    );
  animation: ma-glow-drift 17s ease-in-out infinite alternate;
}

.ma-mine-stage::after,
.ma-detail-stage::after,
.ma-card-thumb::after,
.ma-hover-stage::after {
  background-image:
    repeating-linear-gradient(
      0deg,
      color-mix(in srgb, var(--ma-glow) 13%, transparent) 0 1px,
      transparent 1px 26px
    ),
    repeating-linear-gradient(
      90deg,
      color-mix(in srgb, var(--ma-glow) 13%, transparent) 0 1px,
      transparent 1px 26px
    );
  animation: ma-grid-slide 26s linear infinite;
}

/* 跑着的那一轮：光晕快一倍，网格换成扫描带，一眼看得出还在动。 */
.ma-mine-stage.is-running::before,
.ma-detail-stage.is-running::before,
.ma-card-thumb.is-running::before,
.ma-hover-stage.is-running::before {
  animation-duration: 6s;
}

.ma-mine-stage.is-running::after,
.ma-detail-stage.is-running::after,
.ma-card-thumb.is-running::after,
.ma-hover-stage.is-running::after {
  background-image: linear-gradient(
    180deg,
    transparent,
    color-mix(in srgb, var(--ma-glow) 26%, transparent) 50%,
    transparent
  );
  background-repeat: no-repeat;
  background-size: 100% 38%;
  animation: ma-scan 2.8s ease-in-out infinite;
}

/* 生成题有产物时整层关掉：产物自己带底色，叠上去只会糊。 */
.ma-mine-stage.is-render::before,
.ma-detail-stage.is-render::before,
.ma-card-thumb.is-render::before,
.ma-hover-stage.is-render::before,
.ma-mine-stage.is-render::after,
.ma-detail-stage.is-render::after,
.ma-card-thumb.is-render::after,
.ma-hover-stage.is-render::after {
  display: none;
}

@keyframes ma-glow-drift {
  from {
    transform: translate3d(-3%, -2%, 0);
  }

  to {
    transform: translate3d(4%, 3%, 0);
  }
}

@keyframes ma-grid-slide {
  from {
    transform: translate3d(0, 0, 0);
  }

  to {
    transform: translate3d(26px, 26px, 0);
  }
}

@keyframes ma-scan {
  from {
    transform: translate3d(0, -120%, 0);
  }

  to {
    transform: translate3d(0, 320%, 0);
  }
}

/* 页头光晕：跟封面同一套做法，只是更淡、铺得更开，给整页一层会呼吸的底。 */
.ma-top {
  position: relative;
  isolation: isolate;
}

.ma-top::before {
  content: '';
  position: absolute;
  top: -8px;
  right: -6%;
  left: -6%;
  z-index: -1;
  height: 260px;
  pointer-events: none;
  background-image:
    radial-gradient(
      44% 58% at 18% 32%,
      color-mix(in srgb, var(--ma-primary) 15%, transparent),
      transparent 72%
    ),
    radial-gradient(
      38% 50% at 84% 8%,
      color-mix(in srgb, var(--ma-pass) 11%, transparent),
      transparent 70%
    );
  animation: ma-glow-drift 22s ease-in-out infinite alternate;
}

@media (prefers-reduced-motion: reduce) {
  .ma-top::before,
  .ma-mine-stage::before,
  .ma-detail-stage::before,
  .ma-card-thumb::before,
  .ma-hover-stage::before,
  .ma-mine-stage::after,
  .ma-detail-stage::after,
  .ma-card-thumb::after,
  .ma-hover-stage::after {
    animation: none;
  }
}

/* ============ 响应式 ============ */
@media (max-width: 1100px) {
  .ma-panel {
    grid-template-columns: minmax(0, 1fr);
  }

  .ma-result-col {
    border-top: 1px solid var(--ma-border-soft);
    border-left: none;
    border-radius: 0 0 12px 12px;
  }
}

@media (max-width: 860px) {
  .ma-top-head {
    flex-direction: column;
  }

  .ma-stats {
    flex-basis: auto;
    width: 100%;
    gap: 18px;
  }

  .ma-fields {
    grid-template-columns: minmax(0, 1fr);
  }

  .ma-toolbar {
    flex-direction: column;
    align-items: stretch;
  }

  .ma-selects {
    grid-template-columns: repeat(4, minmax(0, 1fr));
  }

  .ma-submit {
    justify-content: flex-end;
  }

  .ma-caption {
    flex-wrap: wrap;
  }

  /* 窄屏把筛选整条换到标题下面：标题 + 计数 + 筛选挤在一行会溢出。 */
  .ma-history-head {
    flex-wrap: wrap;
    row-gap: 10px;
  }

  .ma-gallery-filter {
    margin-left: 0;
  }
}
</style>

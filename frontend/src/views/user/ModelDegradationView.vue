<template>
  <AppLayout>
    <div class="ma-page">
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
                    <span class="ma-running-spin"><Icon name="sync" size="sm" /></span>
                    <span class="ma-running-label">{{ t('modelDegradation.resultPanel.running') }}</span>
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
                    <iframe
                      v-if="artworkPreviewHtml"
                      :srcdoc="artworkPreviewHtml"
                      sandbox=""
                      class="ma-frame"
                      :title="t('modelDegradation.item.preview')"
                    ></iframe>
                    <template v-else>
                      <Icon :name="tileIcon(currentRun)" size="lg" class="ma-card-glyph" />
                      <span class="ma-card-glyph-text">{{ tileSummary(currentRun) }}</span>
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
        <div class="ma-section-head">
          <h2 class="ma-h2">{{ t('modelDegradation.result.title') }}</h2>
          <span class="ma-section-sub">{{ currentRun.base_url }}</span>
        </div>
        <article class="ma-detail">
          <header class="ma-detail-head">
            <div class="ma-detail-id">
              <span class="ma-detail-name">{{ currentRun.model_id }}</span>
              <span class="ma-detail-time">{{ formatDateTime(currentRun.created_at) }}</span>
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
            </div>
          </header>

          <div
            class="ma-detail-stage"
            :class="[thumbTone(currentRun), { 'is-render': !!artworkPreviewHtml }]"
          >
            <iframe
              v-if="artworkPreviewHtml"
              :srcdoc="artworkPreviewHtml"
              sandbox=""
              class="ma-frame"
              :title="t('modelDegradation.item.preview')"
            ></iframe>
            <template v-else>
              <Icon :name="tileIcon(currentRun)" size="lg" class="ma-card-glyph" />
              <span class="ma-card-glyph-text">{{ tileSummary(currentRun) }}</span>
            </template>
            <span class="ma-ribbon" :class="ribbonClass(currentRun)">{{ ribbonLabel(currentRun) }}</span>
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
        <div class="ma-section-head">
          <h2 class="ma-h2">{{ t('modelDegradation.history.title') }}</h2>
          <span class="ma-section-sub">
            {{ t('modelDegradation.monitor.caption', { count: BAR_COUNT }) }}
          </span>
        </div>
        <p v-if="historyCards.length === 0" class="ma-empty">
          {{ t('modelDegradation.history.empty') }}
        </p>
        <div v-else class="ma-grid">
          <!-- 每张卡片是一个 .ma-tile：预览居中铺在卡面里，右上斜角勋章。
               静止时卡面上只有预览与勋章 —— 模型 / 耗时 / 判定不铺在卡上，
               它们留在展开层里（源站的墙就是这么干净的）。
               卡片本体用 role=button 的 div 而不是 <button>：预览里要放 <iframe>，
               按钮里嵌 iframe 不是合法 HTML。键盘走 enter / space，与题目卡片一致。
               移入时另起一张「大卡」浮层（.ma-hover-card）：标题行 + 预览 + 指标页脚，
               形制照源站那张访客详情卡。浮层 absolute 定位、比卡片高，所以网格不会重排；
               预览是浮层自己的一份 iframe，只在悬停的那一张上创建。 -->
          <div
            v-for="run in historyCards"
            :key="run.id"
            class="ma-tile"
            :class="{ 'is-zoom': zoomId === run.id }"
            @mouseenter="showZoom(run)"
            @mouseleave="hideZoom"
          >
            <div
              class="ma-card"
              role="button"
              tabindex="0"
              :aria-label="t('modelDegradation.history.open')"
              @click="openRun(run.id)"
              @keydown.enter.prevent="openRun(run.id)"
              @keydown.space.prevent="openRun(run.id)"
              @keydown.esc="hideZoom"
              @focus="showZoom(run)"
              @focusout="handleCardFocusOut(run, $event)"
            >
              <span
                class="ma-card-thumb"
                :class="[thumbTone(run), { 'is-render': !!renderHtmlOf(run) }]"
              >
                <!-- 跑过生成题就有产物：把模型吐的 HTML 等比缩半当封面（禁脚本）。
                     缩放加在包裹 div 上而不是 iframe 自己身上 —— 直接给 iframe
                     transform 时 Chrome 会把内部视口按缩放后的尺寸再算一遍，
                     实测缩略图只买到左上四分之一块。 -->
                <span v-if="renderHtmlOf(run)" class="ma-card-cover">
                  <iframe
                    :srcdoc="renderHtmlOf(run)"
                    sandbox=""
                    class="ma-card-frame"
                    :title="t('modelDegradation.item.preview')"
                  ></iframe>
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
                </template>
                <!-- 勋章压的是「预览」的角：源站那张详情卡里，标题行在上面、
                     缎带从作品框的右上角斜插下来，被作品框的圆角裁断。
                     放在 thumb 里两种形态（静止 / 展开）才对得上。 -->
                <span class="ma-ribbon" :class="ribbonClass(run)">{{ ribbonLabel(run) }}</span>
              </span>
            </div>

            <!-- 展开大卡：铺在卡片之上并向上探出标题行的高度，只在悬停/聚焦时显形。
                 操作钮必须 .stop，否则点击会冒泡到卡片本体、把详情又开一遍。 -->
            <div class="ma-hover-card" :aria-hidden="zoomId === run.id ? undefined : 'true'">
              <div class="ma-more-head">
                <div class="ma-more-title">
                  <span class="ma-more-name">{{ run.model_id }}</span>
                  <span class="ma-more-time">{{ formatDateTime(run.created_at) }}</span>
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
                </div>
              </div>

              <span
                class="ma-hover-stage"
                :class="[thumbTone(run), { 'is-render': !!renderHtmlOf(run) }]"
              >
                <!-- 只在悬停这一张上建 iframe：铺满墙的话每张卡都要多一份文档 -->
                <span v-if="renderHtmlOf(run) && zoomId === run.id" class="ma-hover-cover">
                  <iframe
                    :srcdoc="renderHtmlOf(run)"
                    sandbox=""
                    class="ma-hover-frame"
                    :title="t('modelDegradation.item.preview')"
                  ></iframe>
                </span>
                <template v-else>
                  <Icon :name="tileIcon(run)" size="lg" class="ma-card-glyph" />
                  <span class="ma-card-glyph-text">{{ tileSummary(run) }}</span>
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
      </section>
    </div>
  </AppLayout>
</template>

<script setup lang="ts">
import { computed, onMounted, onUnmounted, reactive, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import AppLayout from '@/components/layout/AppLayout.vue'
import Icon from '@/components/icons/Icon.vue'
import Input from '@/components/common/Input.vue'
import Select, { type SelectOption } from '@/components/common/Select.vue'
import modelDegradationAPI, {
  MODEL_AUDIT_PROTOCOLS,
  MODEL_AUDIT_REASONING_EFFORTS,
  type ModelDegradationItem,
  type ModelDegradationItemVerdict,
  type ModelDegradationProtocol,
  type ModelDegradationQuestionPayload,
  type ModelDegradationReasoningEffort,
  type ModelDegradationRun,
  type ModelDegradationTaskBank,
  type ModelDegradationTaskType
} from '@/api/modelDegradation'
import { useAppStore } from '@/stores/app'
import { useAuthStore } from '@/stores/auth'
import { useClipboard } from '@/composables/useClipboard'
import { extractApiErrorMessage } from '@/utils/apiError'
import { formatDateTime, formatReasoningEffort } from '@/utils/format'

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
// 历史区只铺最近这些条，避免卡片太多把页面拉长。
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
 * 上次选的模型与思考等级。
 *
 * 只记这两项：凭据（Base URL / API Key）绝不落盘 —— 它们是上游密钥，
 * 本功能的前提就是「只在本次调用里用一次、服务端不保存」。而模型与思考等级
 * 是用户的偏好，每次重进页面都要重选一遍很烦。
 */
const LAST_SELECTION_KEY = 'model_degradation_last_selection'

const readLastSelection = (): { modelId: string; reasoningEffort: ModelDegradationReasoningEffort } => {
  const empty = { modelId: '', reasoningEffort: '' as ModelDegradationReasoningEffort }
  try {
    const raw = globalThis.localStorage?.getItem(LAST_SELECTION_KEY)
    if (!raw) return empty
    const parsed: unknown = JSON.parse(raw)
    if (typeof parsed !== 'object' || parsed === null) return empty
    const { modelId, reasoningEffort } = parsed as { modelId?: unknown; reasoningEffort?: unknown }
    // 档位必须仍在本版枚举里：旧版本存下的值直接丢掉，
    // 否则会把一个上游不认识的参数发出去，白白换回一次 400。
    const effort =
      typeof reasoningEffort === 'string' &&
      (MODEL_AUDIT_REASONING_EFFORTS as readonly string[]).includes(reasoningEffort)
        ? (reasoningEffort as ModelDegradationReasoningEffort)
        : ''
    return { modelId: typeof modelId === 'string' ? modelId : '', reasoningEffort: effort }
  } catch {
    // 存储被禁用、或内容被改坏时静默退回默认值 —— 记不住偏好不该让页面打不开。
    return empty
  }
}

const lastSelection = readLastSelection()

const form = reactive({
  baseUrl: '',
  apiKey: '',
  // 默认 chat_completions：这是本功能在支持多协议之前的唯一行为。
  protocol: 'chat_completions' as ModelDegradationProtocol,
  modelId: lastSelection.modelId,
  // 空串 = 不指定，请求里不带 reasoning_effort。
  reasoningEffort: lastSelection.reasoningEffort,
  // 页签单选，一轮只跑一类。默认逻辑题：它最快、结果最直观，需要时切到生成题。
  tasks: ['logic'] as ModelDegradationTaskType[]
})

const persistLastSelection = () => {
  try {
    globalThis.localStorage?.setItem(
      LAST_SELECTION_KEY,
      JSON.stringify({ modelId: form.modelId, reasoningEffort: form.reasoningEffort })
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
watch(() => [form.modelId, form.reasoningEffort], persistLastSelection)

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

const running = ref(false)
const currentRun = ref<ModelDegradationRun | null>(null)
const history = ref<ModelDegradationRun[]>([])
const loadingHistory = ref(false)

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

// 历史卡片只铺最近若干条。
const historyCards = computed(() => history.value.slice(0, HISTORY_CARD_COUNT))

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
  // 文档标记只在 </html> 之前找：题面本身就带「以 <!DOCTYPE html> 开头」这个字面量，
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

// 详情里那一行「模型回答」：逻辑题给原样回答，生成题只给源码之外的话
// （源码本体在预览、复制源码、展开预览里，整份贴进详情就是一面代码墙）。
const itemAnswerText = (item: ModelDegradationItem) =>
  item.task_type === 'logic' ? item.response : extractArtworkText(item.response)

// 当前选中那一轮的产物预览。结果卡与「我的检测」都用它，抽出来免得两处各算一遍。
const artworkPreviewHtml = computed(() => renderHtmlOf(currentRun.value))

// 只补「跑过生成题」的卡片：纯逻辑题拉回来的明细没有可预览的产物，白占请求。
// 串行而不是并发：一次铺开二十个请求会把接口拖慢，也会把限流打出来。
const prefetchTileDetails = async () => {
  for (const run of historyCards.value) {
    if (!run.render_verdict || detailCache.has(run.id)) continue
    await loadRunDetail(run.id)
  }
}

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

// 封面中间那行字：逻辑题的得分，没跑逻辑题就报生成题判定。
const tileSummary = (run: ModelDegradationRun) => {
  if (run.status === 'running') return t('modelDegradation.resultPanel.running')
  if (run.logic_total > 0) {
    return `${t('modelDegradation.metrics.logic')} ${run.logic_passed}/${run.logic_total}`
  }
  if (run.render_verdict) return renderVerdictLabel(run.render_verdict)
  return '—'
}

// 斜皮带勋章：中止与进行中不是判定结果，配色也跟着中性化。
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

const ribbonLabel = (run: ModelDegradationRun) => {
  if (run.status === 'running') return t('modelDegradation.history.ribbon.running')
  if (run.status === 'cancelled') return t('modelDegradation.history.ribbon.cancelled')
  if (run.status === 'failed') return t('modelDegradation.history.ribbon.failed')
  // 只跑了生成题：封面就是作品本身，角标标「已生成」而不是「无法判定」。
  if (run.render_verdict === 'generated' && run.logic_total === 0) {
    return t('modelDegradation.history.ribbon.generated')
  }
  return t('modelDegradation.history.ribbon.' + run.verdict)
}

// 鼠标移入（或键盘聚焦）时展开的那张卡，只记 id。
// 展开态是 .ma-hover-card：相对 .ma-tile 绝对定位，比卡片高出一截，
// 所以既不用 fixed 浮层去算坐标 / 翻转 / 贴视口钳位，也不会让网格重排。
const zoomId = ref<number | null>(null)

const showZoom = (run: ModelDegradationRun) => {
  zoomId.value = run.id
  // 展开层的作品框要从明细里取源码；命中缓存时同步返回，不用等这个 promise。
  void loadRunDetail(run.id)
}

const hideZoom = () => {
  zoomId.value = null
}

// 焦点在卡片内部挪动（点到展开层里的操作钮）不算离开：
// button 获得焦点时卡片自己也会 blur，直接收掉的话钮就点不着了。
const handleCardFocusOut = (run: ModelDegradationRun, event: FocusEvent) => {
  const card = event.currentTarget
  const next = event.relatedTarget
  if (next instanceof Node && card instanceof Node && card.contains(next)) return
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
    `${t('modelDegradation.history.upstream')}: ${run.base_url}`
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
// 不再逐题铺卡片 —— 单轮最长 10 分钟，用户要看的是「还要等多久、能不能停」。
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
const elapsedLabel = computed(() =>
  elapsedSeconds.value < 60
    ? t('modelDegradation.resultPanel.seconds', { count: elapsedSeconds.value })
    : t('modelDegradation.resultPanel.minutes', { count: Math.floor(elapsedSeconds.value / 60) })
)

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
    if (scroll) window.scrollTo({ top: 0, behavior: 'smooth' })
  } catch (error) {
    appStore.showError(extractApiErrorMessage(error, t('modelDegradation.errors.loadFailed')))
  }
}

onMounted(() => {
  void loadHistory()
  void loadTaskBank()
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
  grid-template-columns: minmax(0, 1fr) 340px;
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
  /* 三个下拉加两个按钮在窄卡片里放不下：允许换行，别把按钮挤到卡片外面。 */
  flex-wrap: wrap;
}

.ma-selects {
  display: grid;
  /* 三个下拉的宽度上限。用 minmax 而不是固定值：卡片在 1101px 附近最窄，
     固定宽会整块溢出（flex-wrap 只能挪整块，救不了比一行还宽的单个元素）。 */
  grid-template-columns: minmax(0, 174px) minmax(0, 160px) minmax(0, 130px);
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

/* 运行中：一行「转圈 + 进行中 + 秒表 + 停止」，下面一行模型与思考等级。
   不做题目清单 —— 单轮最长 10 分钟，用户要看的是「还要等多久、能不能停」。 */
.ma-running {
  display: flex;
  flex-direction: column;
  gap: 8px;
  padding: 14px 16px;
  border: 1px solid var(--ma-border-soft);
  border-radius: 10px;
  background: var(--ma-panel);
}

.ma-running-row {
  display: flex;
  align-items: center;
  gap: 8px;
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
  font-size: 14px;
  font-weight: 600;
}

.ma-running-count {
  flex: 1 1 auto;
  min-width: 0;
  color: var(--ma-faint);
  font-size: 11px;
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
  gap: 5px;
  color: var(--ma-muted);
  font-size: 12px;
  font-variant-numeric: tabular-nums;
}

/* 停止按钮：一个描边小方块（与源站一致）。方块本身就是全部图标，
   不再加文字，宽度才压得住 12px 一档。 */
.ma-running-stop {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 22px;
  height: 22px;
  padding: 0;
  border: 1px solid var(--ma-input-border);
  border-radius: 5px;
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
  width: 8px;
  height: 8px;
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
  font-size: 12px;
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
.ma-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(min(100%, 260px), 1fr));
  gap: 16px;
}

.ma-tile {
  position: relative;
}

/* 放大中的卡片要盖住邻居：transform 不占布局，抬 z-index 才有层叠次序。
   阴影同步加重，看着像「从墙面上浮起来」。 */
.ma-tile.is-zoom {
  z-index: 40;
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

/* 封面 iframe：按 200% 尺寸渲染再缩半，等于把整页缩小成缩略图。
   两层都不能省：
   - 缩放加在包裹层上，而不是直接 transform iframe ——
     Chrome 会按缩放后的尺寸重算 iframe 内部视口，缩略图只剩一块（实测）；
   - 包裹层用 absolute 钉在左上角 —— 作品框是居中裁品的 flex 容器，
     留在流里会被 align/justify 居中，比框大的封面会往两边探，
     缩半之后看到的是页面的另一角。 */
.ma-card-cover,
.ma-hover-cover {
  position: absolute;
  top: 0;
  left: 0;
  width: 200%;
  height: 200%;
  transform: scale(0.5);
  transform-origin: top left;
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
   切口刚好落在圆角内侧。 */
.ma-ribbon {
  position: absolute;
  top: 2px;
  right: -50px;
  width: 136px;
  padding: 4px 0;
  transform: rotate(45deg);
  background: var(--ma-pass);
  color: #ffffff;
  font-size: 12px;
  font-weight: 700;
  text-align: center;
  letter-spacing: 0.5px;
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
   absolute 定位所以网格不重排；top 取负值把标题行探到卡片上方，
   让作品框正好落在静止时缩略图的位置上（卡片 padding 10 + 边框 1 = 11，
   所以 -11 - 12 - 48 + 11 = -60：展开层上内边距 12 + 标题行 48 - 回补 11）。
   底色必须不透明：它要整个盖住下面的卡片，半透明会让封面 iframe 从字底下透出来。
   不展开时 visibility + pointer-events 一起收掉，操作钮不会变成
   「看不见但能 Tab 到」的暗桩。 */
.ma-hover-card {
  position: absolute;
  top: -60px;
  right: 0;
  left: 0;
  display: flex;
  flex-direction: column;
  gap: 10px;
  padding: 12px 10px;
  border: 1px solid var(--ma-input-border);
  border-radius: 18px;
  background: var(--ma-tile-bg);
  box-shadow: 0 18px 44px rgba(18, 38, 32, 0.24);
  opacity: 0;
  visibility: hidden;
  transform: translateY(5px);
  pointer-events: none;
  transition: opacity 0.16s ease, transform 0.16s ease, visibility 0.16s;
}

.ma-tile.is-zoom .ma-hover-card {
  opacity: 1;
  visibility: visible;
  transform: translateY(0);
  pointer-events: auto;
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

.ma-more-actions button:hover {
  background: var(--ma-seg-bg);
  color: var(--ma-control-fg);
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
    grid-template-columns: repeat(3, minmax(0, 1fr));
  }

  .ma-submit {
    justify-content: flex-end;
  }

  .ma-caption {
    flex-wrap: wrap;
  }
}
</style>

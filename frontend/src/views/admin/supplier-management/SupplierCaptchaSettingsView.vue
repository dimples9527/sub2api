<template>
  <SupplierModuleLayout>
    <section class="sp-panel" aria-label="上游打码设置">
      <header class="sp-panel-head">
        <div class="sp-panel-title">
          <span class="sp-section-index">00</span>
          <div>
            <h2>上游打码设置</h2>
            <span>系统全局打码平台账号，供应商侧单独开关启用</span>
          </div>
        </div>
      </header>
      <p class="sp-intro">
        配置当前打码平台账号，用于自动求解 Sub2API / NewAPI 上游登录的 Cloudflare Turnstile。
        账号为系统全局一份；是否启用由各供应商的「Turnstile 人机校验」开关控制。失败时直接失败，不降级。
      </p>
    </section>

    <div v-if="error" class="sp-alert sp-error-line">{{ error }}</div>

    <section class="sp-panel" aria-label="打码调用统计">
      <header class="sp-panel-head">
        <div class="sp-panel-title">
          <span class="sp-section-index">01</span>
          <div>
            <h2>调用统计</h2>
            <span>仅统计真实调用打码平台的次数</span>
          </div>
        </div>
      </header>
      <div class="sp-stats-body">
        <div class="sp-metric-grid sp-captcha-metric-grid">
          <article class="sp-metric-card sp-stat-static" aria-label="累计调用">
            <div class="sp-metric-label">累计调用</div>
            <div class="sp-metric-value">{{ stats.call_total }}</div>
            <div class="sp-metric-foot">真实发起 SolveTurnstile 的次数</div>
          </article>
          <article class="sp-metric-card sp-stat-static" aria-label="成功次数">
            <div class="sp-metric-label">成功</div>
            <div class="sp-metric-value sp-green">{{ stats.call_success }}</div>
            <div class="sp-metric-foot">返回有效 token 的次数</div>
          </article>
          <article class="sp-metric-card sp-stat-static" aria-label="失败次数">
            <div class="sp-metric-label">失败</div>
            <div class="sp-metric-value sp-red">{{ stats.call_failed }}</div>
            <div class="sp-metric-foot">打码失败或空 token 的次数</div>
          </article>
          <article class="sp-metric-card sp-stat-static" aria-label="最近调用时间">
            <div class="sp-metric-label">最近调用</div>
            <div class="sp-metric-value sp-last-called">{{ lastCalledLabel }}</div>
            <div class="sp-metric-foot">UTC 时间，缓存命中不计入</div>
          </article>
        </div>
      </div>
    </section>

    <section class="sp-panel">
      <header class="sp-panel-head">
        <div class="sp-panel-title">
          <span class="sp-section-index">02</span>
          <div>
            <h2>{{ currentCaptchaProvider.label }} 账号</h2>
            <span>仅在供应商开启 Turnstile 时才会调用</span>
          </div>
        </div>
      </header>

      <div class="sp-captcha-form">
        <div class="sp-field">
          <label for="captcha-provider">打码平台</label>
          <select id="captcha-provider" v-model="form.provider" class="sp-select" @change="handleProviderChange">
            <option value="2captcha">2Captcha</option>
            <option value="yescaptcha">YesCaptcha</option>
          </select>
          <p class="sp-field-hint">
            当前平台：<strong>{{ currentCaptchaProvider.label }}</strong>；切换平台后需重新填写 API Key，自定义 Endpoint 不会跨平台复用。
          </p>
        </div>

        <div class="sp-field">
          <label for="captcha-api-key">
            API Key
            <span v-if="form.api_key_configured" class="sp-field-badge">已配置</span>
          </label>
          <div class="sp-input-row">
            <input
              id="captcha-api-key"
              v-model="form.api_key"
              class="sp-input mono"
              type="password"
              placeholder="********"
              autocomplete="new-password"
            />
            <button
              v-if="form.api_key_configured"
              class="sp-button small ghost"
              type="button"
              :disabled="saving"
              @click="clearApiKey"
            >
              清空
            </button>
          </div>
          <p class="sp-field-hint">
            {{
              form.api_key_configured
                ? 'API Key 已配置，留空以保留当前值。'
                : currentCaptchaProvider.keyHint
            }}
          </p>
        </div>

        <div class="sp-field">
          <label for="captcha-endpoint">自定义 Endpoint</label>
          <input
            id="captcha-endpoint"
            v-model="form.endpoint"
            class="sp-input mono"
            type="text"
            :placeholder="currentCaptchaProvider.endpoint"
          />
          <p class="sp-field-hint">
            可选。默认 <code>{{ currentCaptchaProvider.endpoint }}</code>；仅在使用代理或镜像时填写。
          </p>
        </div>

        <div class="sp-form-actions">
          <button class="sp-button" type="button" :disabled="loading || saving" @click="loadSettings">
            刷新
          </button>
          <button class="sp-button primary" type="button" :disabled="loading || saving" @click="saveSettings">
            {{ saving ? '保存中...' : '保存配置' }}
          </button>
        </div>
      </div>
    </section>

    <section class="sp-panel">
      <header class="sp-panel-head">
        <div class="sp-panel-title">
          <span class="sp-section-index">03</span>
          <div>
            <h2>使用说明</h2>
            <span>与供应商开关的配合方式</span>
          </div>
        </div>
      </header>
      <ul class="sp-help-list">
        <li>在本页配置全局打码平台账号（当前为 <strong>{{ currentCaptchaProvider.label }}</strong>）。</li>
        <li>在「供应商管理」中为需要绕过上游人机验证的供应商打开「Turnstile 人机校验」。</li>
        <li>仅在登录上游时打码；token / session 缓存命中时不会再次打码。</li>
        <li>打码失败会直接导致登录失败，不会用空 token 重试。</li>
        <li>上方统计只记录真实调用打码平台的次数，配置错误或未实际请求平台不计入。</li>
      </ul>
    </section>
  </SupplierModuleLayout>
</template>

<script setup lang="ts">
import { computed, onMounted, reactive, ref } from 'vue'
import { SupplierModuleLayout } from '@/components/admin/supplier-management'
import {
  supplierCaptchaSettingsAPI,
  type SupplierCaptchaSettings,
} from '@/api/admin/supplierCaptchaSettings'
import { extractApiErrorMessage } from '@/utils/apiError'
import { useAppStore } from '@/stores/app'

const loading = ref(false)
const saving = ref(false)
const error = ref('')
const appStore = useAppStore()

const captchaProviderMeta = {
  '2captcha': {
    label: '2Captcha',
    endpoint: 'https://api.2captcha.com',
    keyHint: '从 2Captcha 控制台获取；请保密。',
  },
  yescaptcha: {
    label: 'YesCaptcha',
    endpoint: 'https://api.yescaptcha.com',
    keyHint: '从 YesCaptcha 控制台获取；请保密。',
  },
} as const

type CaptchaProvider = keyof typeof captchaProviderMeta
const defaultCaptchaProvider: CaptchaProvider = '2captcha'

function normalizeCaptchaProvider(provider?: string): CaptchaProvider {
  const normalized = provider?.trim().toLowerCase()
  if (!normalized || !Object.prototype.hasOwnProperty.call(captchaProviderMeta, normalized)) {
    return defaultCaptchaProvider
  }
  return normalized as CaptchaProvider
}

const form = reactive<{
  provider: CaptchaProvider
  api_key: string
  api_key_configured: boolean
  endpoint: string
}>({
  provider: defaultCaptchaProvider,
  api_key: '',
  api_key_configured: false,
  endpoint: '',
})

const stats = reactive({
  call_total: 0,
  call_success: 0,
  call_failed: 0,
  last_called_at: '',
})

const currentCaptchaProvider = computed(() => captchaProviderMeta[normalizeCaptchaProvider(form.provider)])

const lastCalledLabel = computed(() => {
  if (!stats.last_called_at) return '暂无'
  const date = new Date(stats.last_called_at)
  if (Number.isNaN(date.getTime())) return stats.last_called_at
  return date.toLocaleString()
})

function handleProviderChange() {
  form.api_key = ''
  form.api_key_configured = false
  form.endpoint = ''
}

function applySettings(settings: SupplierCaptchaSettings) {
  form.provider = normalizeCaptchaProvider(settings.provider)
  form.api_key = ''
  form.api_key_configured = !!settings.api_key_configured
  form.endpoint = settings.endpoint || ''
  stats.call_total = Number(settings.call_total || 0)
  stats.call_success = Number(settings.call_success || 0)
  stats.call_failed = Number(settings.call_failed || 0)
  stats.last_called_at = settings.last_called_at || ''
}

async function loadSettings() {
  loading.value = true
  error.value = ''
  try {
    const settings = await supplierCaptchaSettingsAPI.get()
    applySettings(settings)
  } catch (e) {
    error.value = extractApiErrorMessage(e, '加载打码配置失败')
  } finally {
    loading.value = false
  }
}

async function saveSettings() {
  saving.value = true
  error.value = ''
  try {
    const settings = await supplierCaptchaSettingsAPI.update({
      provider: normalizeCaptchaProvider(form.provider),
      api_key: form.api_key.trim() || undefined,
      endpoint: form.endpoint.trim(),
    })
    applySettings(settings)
    appStore.showSuccess('打码配置已保存')
  } catch (e) {
    error.value = extractApiErrorMessage(e, '保存打码配置失败')
  } finally {
    saving.value = false
  }
}

async function clearApiKey() {
  if (!window.confirm(`确认清空已配置的 ${currentCaptchaProvider.value.label} API Key？`)) {
    return
  }
  saving.value = true
  error.value = ''
  try {
    const settings = await supplierCaptchaSettingsAPI.update({
      provider: normalizeCaptchaProvider(form.provider),
      endpoint: form.endpoint.trim(),
      clear_api_key: true,
    })
    applySettings(settings)
    appStore.showSuccess('API Key 已清空')
  } catch (e) {
    error.value = extractApiErrorMessage(e, '清空 API Key 失败')
  } finally {
    saving.value = false
  }
}

onMounted(() => {
  void loadSettings()
})
</script>

<style scoped>
/* ===== 面板间距 ===== */
.sp-panel + .sp-panel {
  margin-top: 1rem;
}

.sp-panel + .sp-alert {
  margin-top: 1rem;
}

/* ===== 说明文字 ===== */
.sp-intro {
  margin: 0;
  padding: 0 1.5rem 1.25rem;
  color: var(--sp-muted);
  font-size: 0.875rem;
  line-height: 1.65;
}

/* ===== 统计区域 ===== */
.sp-stats-body {
  padding: 1rem 1.25rem 1.25rem;
}

.sp-captcha-metric-grid {
  grid-template-columns: repeat(4, minmax(140px, 1fr));
  margin-bottom: 0;
}

.sp-stat-static {
  cursor: default;
}

.sp-stat-static:hover {
  border-color: var(--sp-line);
  box-shadow: var(--sp-shadow);
}

.sp-last-called {
  font-size: 1rem;
  line-height: 1.4;
  word-break: break-word;
}

/* ===== 表单 ===== */
.sp-captcha-form {
  display: grid;
  gap: 1.25rem;
  padding: 1.25rem 1.5rem 1.5rem;
}

.sp-field {
  display: grid;
  gap: 0.45rem;
}

.sp-field label {
  font-size: 0.875rem;
  font-weight: 600;
  color: var(--sp-text);
}

.sp-field-badge {
  display: inline-flex;
  align-items: center;
  margin-left: 0.4rem;
  padding: 0.1rem 0.45rem;
  border-radius: 9999px;
  background: color-mix(in srgb, var(--sp-green) 12%, transparent);
  color: var(--sp-green);
  font-size: 0.625rem;
  font-weight: 700;
  letter-spacing: 0.04em;
  text-transform: uppercase;
  vertical-align: middle;
}

.sp-field-hint {
  margin: 0;
  font-size: 0.75rem;
  color: var(--sp-muted);
  line-height: 1.5;
}

.sp-field-hint strong {
  color: var(--sp-text);
  font-weight: 600;
}

.sp-field-hint code {
  padding: 0.1rem 0.35rem;
  border: 1px solid var(--sp-line);
  border-radius: 0.25rem;
  background: var(--sp-panel-2);
  font-family: ui-monospace, SFMono-Regular, Consolas, "Liberation Mono", monospace;
  font-size: 0.75rem;
  color: var(--sp-text);
}

.sp-select {
  width: 100%;
  max-width: 24rem;
  min-height: 2.5rem;
  padding: 0.5rem 0.75rem;
  border: 1px solid var(--sp-line);
  border-radius: 0.5rem;
  outline: 0;
  background: var(--sp-panel);
  color: var(--sp-text);
  font-size: 0.875rem;
  transition: border-color 0.15s ease, box-shadow 0.15s ease;
}

.sp-select:focus {
  border-color: var(--sp-cyan);
  box-shadow: 0 0 0 2px color-mix(in srgb, var(--sp-cyan) 12%, transparent);
}

.sp-input {
  width: 100%;
  max-width: 36rem;
  min-height: 2.5rem;
  padding: 0.5rem 0.75rem;
  border: 1px solid var(--sp-line);
  border-radius: 0.5rem;
  outline: 0;
  background: var(--sp-panel);
  color: var(--sp-text);
  font-size: 0.875rem;
  transition: border-color 0.15s ease, box-shadow 0.15s ease;
}

.sp-input:focus {
  border-color: var(--sp-cyan);
  box-shadow: 0 0 0 2px color-mix(in srgb, var(--sp-cyan) 12%, transparent);
}

.sp-input.mono {
  font-family: ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, 'Liberation Mono', 'Courier New', monospace;
  font-size: 0.875rem;
}

.sp-input-row {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  max-width: 36rem;
}

.sp-input-row .sp-input {
  flex: 1;
  max-width: none;
}

.sp-form-actions {
  display: flex;
  gap: 0.75rem;
  flex-wrap: wrap;
}

/* ===== 使用说明列表 ===== */
.sp-help-list {
  margin: 0;
  padding: 0 1.5rem 1.5rem 2.75rem;
  display: grid;
  gap: 0.5rem;
  color: var(--sp-muted);
  font-size: 0.875rem;
  line-height: 1.6;
}

.sp-help-list li::marker {
  color: var(--sp-cyan);
}

.sp-help-list strong {
  color: var(--sp-text);
  font-weight: 600;
}

/* ===== 响应式 ===== */
@media (max-width: 960px) {
  .sp-captcha-metric-grid {
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }

  .sp-input-row {
    flex-direction: column;
    align-items: stretch;
  }
}

@media (max-width: 560px) {
  .sp-captcha-metric-grid {
    grid-template-columns: minmax(0, 1fr);
  }
}
</style>
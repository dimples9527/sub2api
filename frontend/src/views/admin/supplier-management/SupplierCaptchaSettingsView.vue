<template>
  <SupplierModuleLayout>
    <section class="sp-panel" aria-label="上游打码设置">
      <header class="sp-panel-head">
        <div class="sp-panel-title">
          <span class="sp-section-index">00</span>
          <div>
            <h2>上游打码设置</h2>
            <span>系统全局 2Captcha 账号，供应商侧单独开关启用</span>
          </div>
        </div>
      </header>
      <p class="sp-intro">
        配置 2Captcha 账号，用于自动求解 Sub2API / NewAPI 上游登录的 Cloudflare Turnstile。
        账号为系统全局一份；是否启用由各供应商的「Turnstile 人机校验」开关控制。失败时直接失败，不降级。
      </p>
    </section>

    <div v-if="error" class="sp-alert sp-error-line">{{ error }}</div>
    <div v-if="success" class="sp-alert sp-success-line">{{ success }}</div>

    <section class="sp-panel">
      <header class="sp-panel-head">
        <div class="sp-panel-title">
          <span class="sp-section-index">01</span>
          <div>
            <h2>2Captcha 账号</h2>
            <span>仅在供应商开启 Turnstile 时才会调用</span>
          </div>
        </div>
      </header>

      <div class="sp-captcha-form">
        <div class="sp-field">
          <label for="captcha-provider">打码平台</label>
          <select id="captcha-provider" v-model="form.provider" class="sp-input" disabled>
            <option value="2captcha">2Captcha</option>
          </select>
          <p class="sp-field-hint">当前仅支持 2Captcha</p>
        </div>

        <div class="sp-field">
          <label for="captcha-api-key">API Key</label>
          <input
            id="captcha-api-key"
            v-model="form.api_key"
            class="sp-input mono"
            type="password"
            placeholder="********"
            autocomplete="new-password"
          />
          <p class="sp-field-hint">
            {{
              form.api_key_configured
                ? 'API Key 已配置，留空以保留当前值。'
                : '从 2Captcha 控制台获取；请保密。'
            }}
          </p>
          <button
            v-if="form.api_key_configured"
            class="sp-button small ghost"
            type="button"
            :disabled="saving"
            @click="clearApiKey"
          >
            清空 API Key
          </button>
        </div>

        <div class="sp-field">
          <label for="captcha-endpoint">自定义 Endpoint</label>
          <input
            id="captcha-endpoint"
            v-model="form.endpoint"
            class="sp-input mono"
            type="text"
            placeholder="https://api.2captcha.com"
          />
          <p class="sp-field-hint">可选。默认 https://api.2captcha.com；仅在使用代理或镜像时填写。</p>
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
          <span class="sp-section-index">02</span>
          <div>
            <h2>使用说明</h2>
            <span>与供应商开关的配合方式</span>
          </div>
        </div>
      </header>
      <ul class="sp-help-list">
        <li>在本页配置全局 2Captcha 账号。</li>
        <li>在「供应商管理」中为需要绕过上游人机验证的供应商打开「Turnstile 人机校验」。</li>
        <li>仅在登录上游时打码；token / session 缓存命中时不会再次打码。</li>
        <li>打码失败会直接导致登录失败，不会用空 token 重试。</li>
      </ul>
    </section>
  </SupplierModuleLayout>
</template>

<script setup lang="ts">
import { onMounted, reactive, ref } from 'vue'
import { SupplierModuleLayout } from '@/components/admin/supplier-management'
import {
  supplierCaptchaSettingsAPI,
  type SupplierCaptchaSettings,
} from '@/api/admin/supplierCaptchaSettings'
import { extractApiErrorMessage } from '@/utils/apiError'

const loading = ref(false)
const saving = ref(false)
const error = ref('')
const success = ref('')

const form = reactive({
  provider: '2captcha',
  api_key: '',
  api_key_configured: false,
  endpoint: '',
})

function applySettings(settings: SupplierCaptchaSettings) {
  form.provider = settings.provider || '2captcha'
  form.api_key = ''
  form.api_key_configured = !!settings.api_key_configured
  form.endpoint = settings.endpoint || ''
}

async function loadSettings() {
  loading.value = true
  error.value = ''
  success.value = ''
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
  success.value = ''
  try {
    const settings = await supplierCaptchaSettingsAPI.update({
      provider: form.provider || '2captcha',
      api_key: form.api_key.trim() || undefined,
      endpoint: form.endpoint.trim(),
    })
    applySettings(settings)
    success.value = '打码配置已保存'
  } catch (e) {
    error.value = extractApiErrorMessage(e, '保存打码配置失败')
  } finally {
    saving.value = false
  }
}

async function clearApiKey() {
  if (!window.confirm('确认清空已配置的 2Captcha API Key？')) {
    return
  }
  saving.value = true
  error.value = ''
  success.value = ''
  try {
    const settings = await supplierCaptchaSettingsAPI.update({
      provider: form.provider || '2captcha',
      endpoint: form.endpoint.trim(),
      clear_api_key: true,
    })
    applySettings(settings)
    success.value = 'API Key 已清空'
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
.sp-intro {
  margin: 0;
  padding: 0 1.5rem 1.25rem;
  color: var(--sp-muted, #4b5563);
  font-size: 0.9rem;
  line-height: 1.6;
}

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
  color: var(--sp-text, #111827);
}

.sp-field-hint {
  margin: 0;
  font-size: 0.75rem;
  color: var(--sp-muted, #6b7280);
}

.sp-input {
  width: 100%;
  max-width: 36rem;
  border: 1px solid var(--sp-border, #d1d5db);
  border-radius: 0.5rem;
  padding: 0.55rem 0.75rem;
  background: var(--sp-surface, #fff);
  color: inherit;
}

.sp-input.mono {
  font-family: ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, 'Liberation Mono', 'Courier New', monospace;
  font-size: 0.875rem;
}

.sp-form-actions {
  display: flex;
  gap: 0.75rem;
  flex-wrap: wrap;
}

.sp-help-list {
  margin: 0;
  padding: 0 1.5rem 1.5rem 2.75rem;
  display: grid;
  gap: 0.5rem;
  color: var(--sp-muted, #4b5563);
  font-size: 0.9rem;
  line-height: 1.55;
}

.sp-success-line {
  color: #047857;
  background: #ecfdf5;
  border: 1px solid #a7f3d0;
  border-radius: 0.75rem;
  padding: 0.75rem 1rem;
  margin-bottom: 1rem;
}
</style>

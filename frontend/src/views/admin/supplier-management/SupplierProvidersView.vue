<template>
  <SupplierModuleLayout>
    <header class="sp-page-head">
      <div>
        <div class="sp-eyebrow">Provider Operations</div>
        <h1>供应商管理</h1>
        <p class="sp-subtitle">用运行风险、账号健康和成本效率决定优先处理顺序。</p>
      </div>
      <div class="sp-controls">
        <input v-model="search" class="sp-search" placeholder="搜索供应商" @keyup.enter="loadProviders" />
        <button class="sp-button" type="button" :disabled="loading" @click="loadProviders">刷新数据</button>
        <button class="sp-button" type="button" @click="openTypeManager">类型维护</button>
        <button class="sp-button primary" type="button" @click="openCreate">新增供应商</button>
      </div>
    </header>

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
          <div class="sp-tools">
            <button
              v-for="sort in sorts"
              :key="sort"
              class="sp-pill"
              :class="{ active: activeSort === sort }"
              type="button"
              @click="activeSort = sort"
            >
              {{ sort }}
            </button>
          </div>
        </header>

        <div class="sp-table-wrap">
          <table class="sp-table">
            <thead>
              <tr>
                <th>供应商</th>
                <th>运行状态</th>
                <th>有效 / 可调度账号</th>
                <th>成功率</th>
                <th>今日成本</th>
                <th>余额可用</th>
                <th>倍率风险</th>
                <th>凭据</th>
                <th>最近同步</th>
                <th>操作</th>
              </tr>
            </thead>
            <tbody>
              <tr v-if="loading">
                <td colspan="10">正在加载供应商数据...</td>
              </tr>
              <tr
                v-for="provider in sortedProviders"
                :key="provider.id"
                class="clickable"
                :class="{ selected: selectedProvider?.id === provider.id }"
                @click="selectedProvider = provider"
              >
                <td>
                  <div class="sp-entity">{{ provider.name }}</div>
                  <div class="sp-sub">{{ provider.code }} · {{ provider.provider_type }} · {{ provider.base_url }}</div>
                </td>
                <td><span class="sp-status" :class="statusTone(provider)">{{ statusText(provider) }}</span></td>
                <td><span class="sp-num">{{ provider.valid_account_count }} / {{ provider.schedulable_account_count }}</span></td>
                <td :class="{ 'sp-up': provider.success_rate > 0 && provider.success_rate < 95 }">{{ percent(provider.success_rate) }}</td>
                <td><span class="sp-num">{{ currency(provider.today_cost) }}</span></td>
                <td :class="{ 'sp-up': isLowBalance(provider) }">{{ balanceText(provider) }}</td>
                <td><span class="sp-status" :class="rateTone(provider)">{{ rateRiskText(provider) }}</span></td>
                <td><span class="sp-status" :class="provider.credential_configured ? 'good' : 'warn'">{{ provider.credential_configured ? '已配置' : '未配置' }}</span></td>
                <td>{{ syncText(provider) }}</td>
                <td>
                  <div class="sp-inline" @click.stop>
                    <button class="sp-button small" type="button" @click="openEdit(provider)">编辑</button>
                    <button class="sp-button small" type="button" :disabled="isSyncing(provider, 'all')" @click="syncProviderData(provider, 'all')">{{ isSyncing(provider, 'all') ? '同步中' : '同步全部' }}</button>
                    <button class="sp-button small" type="button" :disabled="provider.is_default" @click="makeDefault(provider)">默认</button>
                    <button class="sp-button small danger" type="button" @click="removeProvider(provider)">删除</button>
                  </div>
                </td>
              </tr>
              <tr v-if="!loading && !sortedProviders.length">
                <td colspan="10">没有符合条件的供应商</td>
              </tr>
            </tbody>
          </table>
        </div>
      </div>

      <aside class="sp-panel">
        <header class="sp-panel-head">
          <div class="sp-panel-title">
            <span class="sp-section-index">02</span>
            <div>
              <h2>供应商组合健康</h2>
              <span>来自供应商独立数据表</span>
            </div>
          </div>
          <span class="sp-status" :class="healthTone">{{ healthLabel }}</span>
        </header>
        <div class="sp-panel-body">
          <div class="sp-alert">{{ healthMessage }}</div>
          <div class="sp-stat-list" style="margin-top: 12px">
            <div class="sp-stat-box"><span>启用供应商</span><b>{{ summary.enabled_count }}</b></div>
            <div class="sp-stat-box"><span>高风险供应商</span><b>{{ summary.high_risk_count }}</b></div>
            <div class="sp-stat-box"><span>余额不足 3 天</span><b>{{ summary.low_balance_count }}</b></div>
            <div class="sp-stat-box"><span>同步异常</span><b>{{ summary.sync_failure_count }}</b></div>
          </div>
          <div class="sp-list">
            <div class="sp-list-item">
              <div><strong>默认供应商</strong><small>{{ defaultProvider ? `${defaultProvider.name} · ${defaultProvider.code}` : '尚未配置' }}</small></div>
              <span class="sp-status" :class="defaultProvider ? 'good' : 'warn'">{{ defaultProvider ? '已设置' : '缺失' }}</span>
            </div>
            <div class="sp-list-item">
              <div><strong>凭据覆盖</strong><small>{{ credentialCoverage }}</small></div>
              <span class="sp-status" :class="credentialMissingCount ? 'warn' : 'good'">{{ credentialMissingCount ? '需补充' : '完整' }}</span>
            </div>
            <div class="sp-list-item">
              <div><strong>倍率风险</strong><small>当前累计 {{ summary.rate_risk_count }} 个风险项</small></div>
              <span class="sp-status" :class="summary.rate_risk_count ? 'warn' : 'good'">{{ summary.rate_risk_count ? '关注' : '正常' }}</span>
            </div>
          </div>
        </div>
      </aside>
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

    <SupplierModal
      :show="modalVisible"
      :title="editingProvider ? '编辑供应商' : '新增供应商'"
      confirm-text="保存供应商"
      @close="closeModal"
      @confirm="submitProvider"
    >
      <form class="sp-form" @submit.prevent="submitProvider">
        <label><span>供应商名称</span><input v-model="form.name" required /></label>
        <label><span>供应商编码</span><input v-model="form.code" required :disabled="Boolean(editingProvider)" /></label>
        <label>
          <span>供应商类型</span>
          <select v-model="form.provider_type" required @change="applySelectedTypeTemplate(true)">
            <option value="" disabled>请选择供应商类型</option>
            <option v-for="type in enabledProviderTypes" :key="type.code" :value="type.code">{{ type.name }}（{{ type.code }}）</option>
          </select>
        </label>
        <label><span>基础地址</span><input v-model="form.base_url" required placeholder="https://supplier.example.com" /></label>
        <label><span>登录接口</span><input v-model="form.login_url" placeholder="https://supplier.example.com/api/v1/auth/login" /></label>
        <label><span>API Key 接口</span><input v-model="form.api_keys_url" placeholder="https://supplier.example.com/api/admin/keys" /></label>
        <label><span>分组接口</span><input v-model="form.groups_url" /></label>
        <label><span>余额接口</span><input v-model="form.balance_url" /></label>
        <label><span>成本接口</span><input v-model="form.usage_cost_url" /></label>
        <label v-if="form.provider_type === 'sub2api'"><span>登录邮箱</span><input v-model="form.email" /></label>
        <label v-else><span>登录用户名</span><input v-model="form.username" /></label>
        <label><span>登录密码</span><input v-model="form.password" type="password" :placeholder="editingProvider ? '留空则保留原密码' : ''" /></label>
        <label><span>账号名前缀</span><input v-model="form.account_name_prefix" /></label>
        <label><span>临时禁用分钟</span><input v-model.number="form.temp_disable_minutes" min="0" type="number" /></label>
        <label><span>倍率缩放</span><input v-model.number="form.account_rate_multiplier_scale" min="0.000001" step="0.000001" type="number" /></label>
        <label><span>排序</span><input v-model.number="form.sort_order" type="number" /></label>
        <label class="sp-check"><input v-model="form.enabled" type="checkbox" />启用供应商</label>
        <label class="sp-check"><input v-model="form.is_default" type="checkbox" />设为默认供应商</label>
        <div class="sp-form-note">切换类型会用类型模板覆盖接口字段；覆盖后仍可继续手动编辑。</div>
      </form>
    </SupplierModal>

    <SupplierModal
      :show="typeManagerVisible"
      title="供应商类型维护"
      confirm-text="保存类型"
      @close="closeTypeManager"
      @confirm="submitProviderType"
    >
      <div class="sp-type-manager">
        <div class="sp-type-list">
          <button
            v-for="type in providerTypes"
            :key="type.id"
            class="sp-type-row"
            :class="{ active: editingProviderType?.id === type.id }"
            type="button"
            @click="editProviderType(type)"
          >
            <span><b>{{ type.name }}</b><small>{{ type.code }}</small></span>
            <em :class="type.enabled ? 'good' : 'warn'">{{ type.enabled ? '启用' : '停用' }}</em>
          </button>
          <button class="sp-button" type="button" @click="newProviderType">新增类型</button>
        </div>
        <form class="sp-form" @submit.prevent="submitProviderType">
          <label><span>供应商类型</span><input v-model="typeForm.name" required placeholder="Sub2API" /></label>
          <label><span>类型编码</span><input v-model="typeForm.code" required placeholder="sub2api" /></label>
          <label><span>登录接口</span><input v-model="typeForm.login_url" placeholder="https://supplier.example.com/api/v1/auth/login" /></label>
          <label><span>API Key 接口</span><input v-model="typeForm.api_keys_url" /></label>
          <label><span>分组接口</span><input v-model="typeForm.groups_url" /></label>
          <label><span>余额接口</span><input v-model="typeForm.balance_url" /></label>
          <label><span>成本接口</span><input v-model="typeForm.usage_cost_url" /></label>
          <label><span>排序</span><input v-model.number="typeForm.sort_order" type="number" /></label>
          <label class="sp-check"><input v-model="typeForm.enabled" type="checkbox" />启用类型</label>
          <div class="sp-form-note">这些接口作为供应商模板使用；供应商自身字段为空时后台会使用这里的配置。</div>
          <button v-if="editingProviderType" class="sp-button danger" type="button" @click="removeProviderType(editingProviderType)">删除当前类型</button>
        </form>
      </div>
    </SupplierModal>

    <SupplierModal
      :show="testResultVisible"
      title="接口测试结果"
      confirm-text="关闭"
      @close="closeTestResult"
      @confirm="closeTestResult"
    >
      <div v-if="testResult" class="sp-test-result">
        <div class="sp-detail-grid">
          <div class="sp-detail-cell"><span>测试接口</span><b>{{ scopeLabel(testResult.scope) }}</b></div>
          <div class="sp-detail-cell"><span>HTTP 状态</span><b>{{ testResult.http_status || '无' }}</b></div>
          <div class="sp-detail-cell"><span>耗时</span><b>{{ testResult.duration_ms }} ms</b></div>
          <div class="sp-detail-cell"><span>响应大小</span><b>{{ testResult.response_bytes }} bytes</b></div>
        </div>
        <div v-if="testResult.error" class="sp-alert sp-error-line">请求错误：{{ testResult.error }}</div>
        <div v-if="testResult.parse_error" class="sp-alert sp-error-line">解析错误：{{ testResult.parse_error }}</div>
        <div class="sp-timeline">
          <h4>调用尝试</h4>
          <div v-for="(attempt, index) in testResult.attempts" :key="`${attempt.endpoint}:${index}`" class="sp-event">
            <b>{{ index + 1 }}. {{ attempt.endpoint }}</b>
            <p>HTTP {{ attempt.http_status || '无' }} · {{ attempt.duration_ms }} ms · {{ attempt.response_bytes }} bytes</p>
            <p v-if="attempt.error">请求错误：{{ attempt.error }}</p>
            <p v-if="attempt.parse_error">解析错误：{{ attempt.parse_error }}</p>
          </div>
        </div>
        <div class="sp-timeline">
          <h4>脱敏原始返回</h4>
          <pre class="sp-message-detail">{{ testResult.response_summary || '无返回内容' }}</pre>
        </div>
        <div class="sp-timeline">
          <h4>解析结果</h4>
          <pre class="sp-message-detail">{{ formatDiagnosticJSON(testResult.parsed_data) }}</pre>
        </div>
        <div class="sp-form-note">敏感字段已脱敏；该测试只调用接口，不会写入同步记录或本地数据表。</div>
      </div>
    </SupplierModal>

  </SupplierModuleLayout>
</template>

<script setup lang="ts">
import { computed, onMounted, reactive, ref, watch } from 'vue'
import { SupplierDrawer, SupplierModal, SupplierModuleLayout } from '@/components/admin/supplier-management'
import supplierProvidersAPI, { type SupplierProvider, type SupplierProviderSummary, type SupplierProviderUpsertPayload } from '@/api/admin/supplierProviders'
import supplierProviderTypesAPI, { type SupplierProviderType, type SupplierProviderTypeUpsertPayload } from '@/api/admin/supplierProviderTypes'
import { syncProvider, testProviderEndpoint, type SupplierProviderEndpointTestResult, type SupplierSyncScope } from '@/api/admin/supplierProviderData'
import { useAppStore } from '@/stores/app'

type Tone = 'good' | 'warn' | 'bad' | 'info' | ''
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

const providers = ref<SupplierProvider[]>([])
const providerTypes = ref<SupplierProviderType[]>([])
const summary = ref<SupplierProviderSummary>(emptySummary())
const search = ref('')
const filter = ref('all')
const activeSort = ref('风险优先')
const loading = ref(false)
const error = ref('')
const selectedProvider = ref<SupplierProvider | null>(null)
const editingProvider = ref<SupplierProvider | null>(null)
const editingProviderType = ref<SupplierProviderType | null>(null)
const modalVisible = ref(false)
const typeManagerVisible = ref(false)
const form = reactive<SupplierProviderUpsertPayload>(emptyForm())
const typeForm = reactive<SupplierProviderTypeUpsertPayload>(emptyTypeForm())
const syncingKeys = ref<Set<string>>(new Set())
const testingKeys = ref<Set<string>>(new Set())
const testResultVisible = ref(false)
const testResult = ref<SupplierProviderEndpointTestResult | null>(null)
let searchTimer: number | undefined
const appStore = useAppStore()

const sorts = ['风险优先', '成本效率', '最近同步']
const enabledProviderTypes = computed(() => providerTypes.value.filter(type => type.enabled))

const metrics = computed(() => [
  { key: 'all', tone: 'green', label: '启用供应商', value: String(summary.value.enabled_count), foot: `共管理 ${summary.value.total_count} 个供应商` },
  { key: 'risk', tone: 'red', label: '高风险供应商', value: String(summary.value.high_risk_count), foot: '风险等级为 high 或 critical' },
  { key: 'balance', tone: 'orange', label: '余额不足 3 天', value: String(summary.value.low_balance_count), foot: '按预计可用天数判断' },
  { key: 'sync', tone: 'blue', label: '同步异常', value: String(summary.value.sync_failure_count), foot: '最近同步状态失败' },
  { key: 'rate', tone: 'amber', label: '倍率风险项', value: String(summary.value.rate_risk_count), foot: '供应商账号倍率风险累计' },
])

const filteredProviders = computed(() => providers.value.filter(provider => {
  if (filter.value === 'risk' && !['high', 'critical'].includes(provider.risk_level)) return false
  if (filter.value === 'balance' && !isLowBalance(provider)) return false
  if (filter.value === 'sync' && provider.sync_status !== 'failed') return false
  if (filter.value === 'rate' && provider.rate_risk_count <= 0) return false
  return true
}))

const sortedProviders = computed(() => {
  const rows = [...filteredProviders.value]
  if (activeSort.value === '成本效率') return rows.sort((left, right) => left.today_cost - right.today_cost)
  if (activeSort.value === '最近同步') return rows.sort((left, right) => new Date(right.last_sync_at || 0).getTime() - new Date(left.last_sync_at || 0).getTime())
  return rows.sort((left, right) => riskWeight(right) - riskWeight(left) || left.sort_order - right.sort_order || left.id - right.id)
})

const defaultProvider = computed(() => providers.value.find(provider => provider.is_default) || null)
const credentialMissingCount = computed(() => providers.value.filter(provider => !provider.credential_configured).length)
const credentialCoverage = computed(() => `${providers.value.length - credentialMissingCount.value} / ${providers.value.length} 个供应商已配置凭据`)
const healthTone = computed<Tone>(() => summary.value.high_risk_count || summary.value.sync_failure_count ? 'warn' : 'good')
const healthLabel = computed(() => healthTone.value === 'good' ? '稳定' : '需关注')
const healthMessage = computed(() => {
  if (!providers.value.length) return '还没有供应商数据，请先新增供应商配置。'
  if (summary.value.high_risk_count) return `当前有 ${summary.value.high_risk_count} 个高风险供应商，应优先检查凭据、余额和同步结果。`
  if (summary.value.sync_failure_count) return `当前有 ${summary.value.sync_failure_count} 个供应商同步异常，需要查看同步日志。`
  return '当前供应商组合没有高风险或同步失败记录。'
})

watch(search, () => {
  window.clearTimeout(searchTimer)
  searchTimer = window.setTimeout(loadProviders, 350)
})

onMounted(async () => {
  await loadProviderTypes()
  await loadProviders()
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
    is_default: provider.is_default,
  })
  modalVisible.value = true
}

function closeModal() {
  modalVisible.value = false
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
  if (typeof provider.estimated_days === 'number') return `${provider.estimated_days.toFixed(1)} 天`
  return currency(provider.current_balance)
}

function isLowBalance(provider: SupplierProvider): boolean {
  return typeof provider.estimated_days === 'number' && provider.estimated_days < 3
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
  return `¥ ${Number(value || 0).toLocaleString('zh-CN', { maximumFractionDigits: 2 })}`
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

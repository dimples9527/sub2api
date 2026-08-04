<template>
  <SupplierModuleLayout>
    <header class="sp-page-head">
      <div>
        <div class="sp-eyebrow">Supplier Operations / Notification Desk</div>
        <h1>通知配置</h1>
        <p class="sp-subtitle">独立管理飞书与邮件渠道、余额事件订阅，以及每一次通知投递的重试记录。</p>
      </div>
      <div class="sp-controls">
        <span v-if="lastLoadedAt" class="sp-data-note sp-notification-loaded">更新于 {{ formatDateTime(lastLoadedAt) }}</span>
        <button class="sp-button" type="button" :disabled="loading" @click="loadAll">{{ loading ? '刷新中…' : '刷新数据' }}</button>
        <button class="sp-button primary" type="button" @click="openCreateChannelDialog">新增通知渠道</button>
      </div>
    </header>

    <div v-if="error" class="sp-alert sp-error-line" data-test="supplier-notification-error">{{ error }}</div>
    <div v-if="toast" class="sp-alert sp-success-line" data-test="supplier-notification-toast">{{ toast }}</div>

    <section class="sp-metric-grid sp-notification-metrics" aria-label="通知配置概览">
      <article class="sp-metric-card sp-blue"><div class="sp-metric-label">通知渠道</div><div class="sp-metric-value">{{ channels.length }}</div><div class="sp-metric-foot">飞书与邮件渠道统一管理</div></article>
      <article class="sp-metric-card sp-green"><div class="sp-metric-label">已启用渠道</div><div class="sp-metric-value">{{ enabledChannelCount }}</div><div class="sp-metric-foot">未配置完整的渠道不会成功投递</div></article>
      <article class="sp-metric-card sp-violet"><div class="sp-metric-label">有效订阅</div><div class="sp-metric-value">{{ enabledSubscriptionCount }}</div><div class="sp-metric-foot">按事件类型和供应商范围分发</div></article>
      <article class="sp-metric-card sp-amber"><div class="sp-metric-label">待处理投递</div><div class="sp-metric-value">{{ pendingDeliveryCount }}</div><div class="sp-metric-foot">失败记录会按策略自动重试</div></article>
    </section>

    <section class="sp-panel sp-notification-panel" data-test="notification-channels-section">
      <header class="sp-panel-head">
        <div class="sp-panel-title"><span class="sp-section-index">01</span><div><h2>通知渠道</h2><span>敏感配置只显示状态，不回显 Webhook、Secret 或密码</span></div></div>
        <span class="sp-status info">{{ channels.length }} 个渠道</span>
      </header>
      <DataTable :columns="channelColumns" :data="channels" :loading="loading" row-key="id" :virtualize-threshold="1000">
        <template #cell-name="{ row: channel }"><div class="sp-entity">{{ channel.name }}</div><div class="sp-sub">#{{ channel.id }} · {{ channelTypeLabel(channel.channel_type) }}</div></template>
        <template #cell-channel_type="{ row: channel }"><span class="sp-tag" :class="channel.channel_type === 'feishu' ? 'info' : 'good'">{{ channelTypeLabel(channel.channel_type) }}</span></template>
        <template #cell-configured="{ row: channel }">
          <div class="sp-config-stack">
            <span class="sp-status" :class="channel.configured ? 'good' : 'bad'">{{ channel.configured ? '已配置' : '待完善' }}</span>
            <span v-if="channel.channel_type === 'feishu'" class="sp-sub">Webhook {{ channel.feishu_webhook_configured ? '已配置' : '未配置' }} · Secret {{ secretConfigured(channel) ? '已配置' : '未配置' }}</span>
            <span v-else class="sp-sub">{{ channel.email_host || '未填写服务器' }}<template v-if="channel.email_port">:{{ channel.email_port }}</template> · {{ channel.email_to?.length || 0 }} 个收件人</span>
            <span v-if="channel.proxy_configured" class="sp-sub sp-proxy-note">已配置 HTTP 代理</span>
          </div>
        </template>
        <template #cell-enabled="{ row: channel }"><div class="sp-inline"><Toggle :model-value="channel.enabled" :aria-label="`${channel.name}通知渠道${channel.enabled ? '已启用' : '已停用'}`" @click.stop @update:model-value="toggleChannel(channel, $event)" /><span class="sp-status" :class="channel.enabled ? 'good' : 'info'">{{ channel.enabled ? '已启用' : '已停用' }}</span></div></template>
        <template #cell-updated_at="{ row: channel }">{{ formatDateTime(channel.updated_at) }}</template>
        <template #cell-actions="{ row: channel }"><div class="sp-table-actions"><button class="sp-button small ghost" type="button" :disabled="testingChannelId === channel.id" @click="testChannel(channel)">{{ testingChannelId === channel.id ? '测试中…' : '测试发送' }}</button><button class="sp-button small ghost" type="button" @click="openEditChannelDialog(channel)">编辑</button><button class="sp-button small danger" type="button" @click="removeChannel(channel)">删除</button></div></template>
        <template #empty><div class="sp-panel-body sp-empty-state">暂无通知渠道，请先新增飞书或邮件渠道。</div></template>
      </DataTable>
    </section>

    <section class="sp-panel sp-notification-panel" data-test="notification-subscriptions-section">
      <header class="sp-panel-head sp-notification-filter-head">
        <div class="sp-panel-title"><span class="sp-section-index">02</span><div><h2>事件订阅</h2><span>一个渠道可以订阅全部供应商，也可以只订阅指定供应商</span></div></div>
        <button class="sp-button small primary" type="button" :disabled="channels.length === 0" @click="openCreateSubscriptionDialog">新增订阅</button>
      </header>
      <DataTable :columns="subscriptionColumns" :data="subscriptions" :loading="loading" row-key="id" :virtualize-threshold="1000">
        <template #cell-channel_id="{ row: subscription }"><div class="sp-entity">{{ channelName(subscription.channel_id) }}</div><div class="sp-sub">渠道 #{{ subscription.channel_id }}</div></template>
        <template #cell-provider_id="{ row: subscription }">{{ providerName(subscription.provider_id) }}</template>
        <template #cell-event_type="{ row: subscription }"><span class="sp-tag" :class="subscription.event_type === 'balance_recovered' ? 'good' : 'warn'">{{ eventTypeLabel(subscription.event_type) }}</span></template>
        <template #cell-enabled="{ row: subscription }"><div class="sp-inline"><Toggle :model-value="subscription.enabled" :aria-label="`${channelName(subscription.channel_id)}订阅${subscription.enabled ? '已启用' : '已停用'}`" @click.stop @update:model-value="toggleSubscription(subscription, $event)" /><span class="sp-status" :class="subscription.enabled ? 'good' : 'info'">{{ subscription.enabled ? '已启用' : '已停用' }}</span></div></template>
        <template #cell-updated_at="{ row: subscription }">{{ formatDateTime(subscription.updated_at) }}</template>
        <template #cell-actions="{ row: subscription }"><div class="sp-table-actions"><button class="sp-button small ghost" type="button" @click="openEditSubscriptionDialog(subscription)">编辑</button><button class="sp-button small danger" type="button" @click="removeSubscription(subscription)">删除</button></div></template>
        <template #empty><div class="sp-panel-body sp-empty-state">暂无事件订阅。新增订阅后，余额不足和余额恢复事件才会进入渠道。</div></template>
      </DataTable>
    </section>

    <section class="sp-panel sp-notification-panel" data-test="notification-deliveries-section">
      <header class="sp-panel-head sp-notification-filter-head">
        <div class="sp-panel-title"><span class="sp-section-index">03</span><div><h2>投递记录</h2><span>查看通知状态、重试次数和最近一次失败原因</span></div></div>
        <div class="sp-controls sp-notification-filters">
          <Select v-model="deliveryChannelFilter" :options="channelOptions" clearable :searchable="false" aria-label="按渠道筛选" class="sp-notification-filter-control" @change="applyDeliveryFilters" />
          <Select v-model="deliveryEventFilter" :options="eventTypeOptions" clearable :searchable="false" aria-label="按事件筛选" class="sp-notification-filter-control" @change="applyDeliveryFilters" />
          <Select v-model="deliveryStatusFilter" :options="deliveryStatusOptions" clearable :searchable="false" aria-label="按状态筛选" class="sp-notification-filter-control" @change="applyDeliveryFilters" />
          <button class="sp-button small ghost" type="button" @click="resetDeliveryFilters">重置筛选</button>
        </div>
      </header>
      <DataTable :columns="deliveryColumns" :data="deliveries" :loading="deliveriesLoading" row-key="id" :virtualize-threshold="1000">
        <template #cell-channel_name="{ row: delivery }"><div class="sp-entity">{{ delivery.channel_name }}</div><div class="sp-sub">渠道 #{{ delivery.channel_id }}</div></template>
        <template #cell-provider_name="{ row: delivery }"><div class="sp-entity">{{ delivery.provider_name }}</div><div class="sp-sub">供应商 #{{ delivery.provider_id }}</div></template>
        <template #cell-event_type="{ row: delivery }">{{ eventTypeLabel(delivery.event_type) }}</template>
        <template #cell-status="{ row: delivery }"><span class="sp-status" :class="deliveryStatusTone(delivery.status)">{{ deliveryStatusLabel(delivery.status) }}</span><div v-if="delivery.last_error" class="sp-sub sp-delivery-error">{{ delivery.last_error }}</div></template>
        <template #cell-attempt_count="{ row: delivery }">{{ delivery.attempt_count }} 次</template>
        <template #cell-next_attempt_at="{ row: delivery }">{{ delivery.sent_at ? `已发送 · ${formatDateTime(delivery.sent_at)}` : formatDateTime(delivery.next_attempt_at) }}</template>
        <template #cell-actions="{ row: delivery }"><button class="sp-button small ghost" type="button" @click="openDeliveryDetail(delivery)">查看详情</button></template>
        <template #empty><div class="sp-panel-body sp-empty-state">暂无投递记录。</div></template>
      </DataTable>
      <div class="sp-pagination-row">
        <Pagination v-if="deliveryTotal > 0" v-model:page="deliveryPage" v-model:page-size="deliveryPageSize" :total="deliveryTotal" :show-jump="deliveryTotal > 100" @update:page="loadDeliveries" @update:page-size="onDeliveryPageSizeChange" />
      </div>
    </section>

    <BaseDialog :show="channelDialogVisible" :title="channelDialogTitle" width="wide" @close="closeChannelDialog">
      <form v-if="channelForm" class="sp-dialog-form" @submit.prevent="saveChannel">
        <section class="sp-form-section"><div class="sp-form-section-head"><span>01</span><div><h3>基础信息</h3><p>渠道名称用于订阅和投递记录中的识别。</p></div></div><div class="sp-form-grid"><Input v-model="channelForm.name" label="渠道名称" placeholder="例如：供应商余额飞书群" required /><div class="sp-form-control"><label class="sp-form-label" for="supplier-notification-channel-type">渠道类型</label><Select id="supplier-notification-channel-type" v-model="channelForm.channel_type" :options="channelTypeOptions" :disabled="channelForm.id !== null" :searchable="false" aria-label="渠道类型" /><p v-if="channelForm.id !== null" class="sp-form-hint">编辑已有渠道时不能切换类型，避免误覆盖另一类配置。</p></div></div><label class="sp-switch-field"><span>启用通知渠道</span><span class="sp-inline"><Toggle v-model="channelForm.enabled" /><em>{{ channelForm.enabled ? '已启用' : '已停用' }}</em></span></label></section>

        <section v-if="channelForm.channel_type === 'feishu'" class="sp-form-section"><div class="sp-form-section-head"><span>02</span><div><h3>飞书 Webhook</h3><p>Webhook 和签名 Secret 不会从后端回显；编辑时留空表示保留原值。</p></div></div><div class="sp-form-grid"><Input v-model="channelForm.feishu.webhook_url" label="Webhook 地址" placeholder="https://open.feishu.cn/open-apis/bot/v2/hook/..." hint="新增渠道必须填写；编辑时可以重新填写，留空保留原地址。" autocomplete="url" /><Input v-model="channelForm.feishu.secret" type="password" label="签名 Secret" placeholder="留空表示保留已配置 Secret" hint="Secret 只用于服务端签名，页面不会显示已保存的值。" autocomplete="new-password" /></div></section>

        <section v-else class="sp-form-section"><div class="sp-form-section-head"><span>02</span><div><h3>邮件 SMTP</h3><p>收件人可以使用逗号、分号或换行分隔；密码留空表示保留原值。</p></div></div><div class="sp-form-grid"><Input v-model="channelForm.email.host" label="SMTP 主机" placeholder="smtp.example.com" required /><Input v-model="channelForm.email.port" type="number" label="SMTP 端口" min="1" max="65535" placeholder="587" required /><Input v-model="channelForm.email.username" label="SMTP 用户名" placeholder="可选" autocomplete="username" /><Input v-model="channelForm.email.password" type="password" label="SMTP 密码" placeholder="留空表示保留已配置密码" autocomplete="new-password" /><Input v-model="channelForm.email.from" label="发件人地址" placeholder="alerts@example.com" required autocomplete="email" /><Input v-model="channelForm.email.to" label="收件人地址" placeholder="ops@example.com, owner@example.com" required /></div><label class="sp-switch-field"><span>启用 STARTTLS</span><span class="sp-inline"><Toggle v-model="channelForm.email.starttls" /><em>{{ channelForm.email.starttls ? '已启用' : '已停用' }}</em></span></label></section>

        <section class="sp-form-section"><div class="sp-form-section-head"><span>03</span><div><h3>HTTP 代理（可选）</h3><p>代理账号和密码不会回显；已有代理配置会在保存时保留。</p></div></div><div class="sp-form-grid"><Input v-model="channelForm.proxy.url" label="代理地址" placeholder="http://proxy.example.com:8080" autocomplete="url" /><Input v-model="channelForm.proxy.username" label="代理用户名" placeholder="可选" autocomplete="username" /><Input v-model="channelForm.proxy.password" type="password" label="代理密码" placeholder="留空表示保留已配置密码" autocomplete="new-password" /></div></section>
      </form>
      <template #footer><button class="sp-button" type="button" @click="closeChannelDialog">取消</button><button class="sp-button primary" type="button" :disabled="saving" @click="saveChannel">{{ saving ? '保存中…' : '保存渠道' }}</button></template>
    </BaseDialog>

    <BaseDialog :show="subscriptionDialogVisible" :title="subscriptionDialogTitle" width="normal" @close="closeSubscriptionDialog">
      <form v-if="subscriptionForm" class="sp-dialog-form" @submit.prevent="saveSubscription">
        <div class="sp-form-grid"><div class="sp-form-control"><label class="sp-form-label" for="supplier-notification-subscription-channel">通知渠道</label><Select id="supplier-notification-subscription-channel" v-model="subscriptionForm.channel_id" :options="channelOptions" :searchable="false" aria-label="通知渠道" /></div><div class="sp-form-control"><label class="sp-form-label" for="supplier-notification-subscription-event">事件类型</label><Select id="supplier-notification-subscription-event" v-model="subscriptionForm.event_type" :options="eventTypeOptions" :searchable="false" aria-label="事件类型" /></div><div class="sp-form-control sp-form-control-wide"><label class="sp-form-label" for="supplier-notification-subscription-provider">供应商范围</label><Select id="supplier-notification-subscription-provider" v-model="subscriptionForm.provider_id" :options="providerOptions" searchable clearable aria-label="供应商范围" /><p class="sp-form-hint">选择“全部供应商”时，任何供应商的对应事件都会投递到该渠道。</p></div></div>
        <label class="sp-switch-field"><span>启用事件订阅</span><span class="sp-inline"><Toggle v-model="subscriptionForm.enabled" /><em>{{ subscriptionForm.enabled ? '已启用' : '已停用' }}</em></span></label>
      </form>
      <template #footer><button class="sp-button" type="button" @click="closeSubscriptionDialog">取消</button><button class="sp-button primary" type="button" :disabled="savingSubscription" @click="saveSubscription">{{ savingSubscription ? '保存中…' : '保存订阅' }}</button></template>
    </BaseDialog>

    <BaseDialog :show="deliveryDetailVisible" title="通知投递详情" width="extra-wide" @close="closeDeliveryDetail">
      <div v-if="deliveryDetail" class="sp-delivery-detail">
        <section class="sp-detail-summary"><div><span>渠道</span><strong>{{ deliveryDetail.channel_name }}</strong></div><div><span>供应商</span><strong>{{ deliveryDetail.provider_name }}</strong></div><div><span>事件</span><strong>{{ eventTypeLabel(deliveryDetail.event_type) }}</strong></div><div><span>状态</span><strong class="sp-status" :class="deliveryStatusTone(deliveryDetail.status)">{{ deliveryStatusLabel(deliveryDetail.status) }}</strong></div><div><span>尝试次数</span><strong>{{ deliveryDetail.attempt_count }} 次</strong></div><div><span>创建时间</span><strong>{{ formatDateTime(deliveryDetail.created_at) }}</strong></div></section>
        <div v-if="deliveryDetail.last_error" class="sp-alert sp-error-line">最近失败：{{ deliveryDetail.last_error }}</div>
        <section class="sp-detail-section"><header class="sp-detail-section-head"><h3>投递载荷</h3><span>仅展示余额事件内容，不包含渠道凭据</span></header><pre class="sp-payload">{{ formatPayload(deliveryDetail.payload) }}</pre></section>
        <section class="sp-detail-section"><header class="sp-detail-section-head"><h3>投递尝试</h3><span>{{ deliveryAttempts.length }} 条记录</span></header><DataTable :columns="attemptColumns" :data="deliveryAttempts" :loading="deliveryAttemptsLoading" row-key="id"><template #cell-attempt_number="{ row: attempt }">第 {{ attempt.attempt_number }} 次</template><template #cell-status="{ row: attempt }"><span class="sp-status" :class="attempt.status === 'delivered' ? 'good' : attempt.status === 'failed' ? 'bad' : 'info'">{{ deliveryStatusLabel(attempt.status) }}</span></template><template #cell-http_status="{ row: attempt }">{{ attempt.http_status || '—' }}</template><template #cell-error_message="{ row: attempt }">{{ attempt.error_message || attempt.response_body || '—' }}</template><template #cell-attempted_at="{ row: attempt }">{{ formatDateTime(attempt.attempted_at) }}</template><template #empty><div class="sp-empty-state">暂无投递尝试记录。</div></template></DataTable></section>
      </div>
      <div v-else class="sp-empty-state">正在加载投递详情…</div>
    </BaseDialog>
  </SupplierModuleLayout>
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { SupplierModuleLayout } from '@/components/admin/supplier-management'
import BaseDialog from '@/components/common/BaseDialog.vue'
import DataTable from '@/components/common/DataTable.vue'
import Input from '@/components/common/Input.vue'
import Pagination from '@/components/common/Pagination.vue'
import Select, { type SelectOption } from '@/components/common/Select.vue'
import Toggle from '@/components/common/Toggle.vue'
import type { Column } from '@/components/common/types'
import supplierProvidersAPI, { type SupplierProvider } from '@/api/admin/supplierProviders'
import {
  createSupplierNotificationChannel,
  createSupplierNotificationSubscription,
  deleteSupplierNotificationChannel,
  deleteSupplierNotificationSubscription,
  getSupplierNotificationDelivery,
  listSupplierNotificationChannels,
  listSupplierNotificationDeliveryAttempts,
  listSupplierNotificationDeliveries,
  listSupplierNotificationSubscriptions,
  sendSupplierNotificationChannelTest,
  updateSupplierNotificationChannel,
  updateSupplierNotificationSubscription,
  type SupplierNotificationChannelInput,
  type SupplierNotificationChannelType,
  type SupplierNotificationChannelView,
  type SupplierNotificationDelivery,
  type SupplierNotificationDeliveryAttempt,
  type SupplierNotificationDeliveryDetail,
  type SupplierNotificationDeliveryListParams,
  type SupplierNotificationDeliveryStatus,
  type SupplierNotificationEventType,
  type SupplierNotificationSubscription,
  type SupplierNotificationSubscriptionInput,
} from '@/api/admin/supplierNotifications'
import { extractApiErrorMessage } from '@/utils/apiError'

interface ChannelForm {
  id: number | null
  name: string
  channel_type: SupplierNotificationChannelType
  enabled: boolean
  feishu: {
    webhook_url: string
    secret: string
  }
  email: {
    host: string
    port: string
    username: string
    password: string
    from: string
    to: string
    starttls: boolean
  }
  proxy: {
    url: string
    username: string
    password: string
  }
}

interface SubscriptionForm {
  id: number | null
  channel_id: number
  provider_id: number
  event_type: SupplierNotificationEventType
  enabled: boolean
}

const channels = ref<SupplierNotificationChannelView[]>([])
const subscriptions = ref<SupplierNotificationSubscription[]>([])
const providers = ref<SupplierProvider[]>([])
const deliveries = ref<SupplierNotificationDelivery[]>([])
const loading = ref(false)
const deliveriesLoading = ref(false)
const deliveryAttemptsLoading = ref(false)
const saving = ref(false)
const savingSubscription = ref(false)
const savingChannelId = ref<number | null>(null)
const savingSubscriptionId = ref<number | null>(null)
const testingChannelId = ref<number | null>(null)
const error = ref('')
const toast = ref('')
const lastLoadedAt = ref('')

const channelDialogVisible = ref(false)
const channelDialogTitle = ref('新增通知渠道')
const channelForm = ref<ChannelForm | null>(null)
const subscriptionDialogVisible = ref(false)
const subscriptionDialogTitle = ref('新增事件订阅')
const subscriptionForm = ref<SubscriptionForm | null>(null)

const deliveryChannelFilter = ref<string | number | boolean | null>(null)
const deliveryEventFilter = ref<string | number | boolean | null>(null)
const deliveryStatusFilter = ref<string | number | boolean | null>(null)
const deliveryPage = ref(1)
const deliveryPageSize = ref(10)
const deliveryTotal = ref(0)

const deliveryDetailVisible = ref(false)
const deliveryDetail = ref<SupplierNotificationDeliveryDetail | null>(null)
const deliveryAttempts = ref<SupplierNotificationDeliveryAttempt[]>([])

const channelColumns: Column[] = [
  { key: 'name', label: '渠道' },
  { key: 'channel_type', label: '类型' },
  { key: 'configured', label: '配置状态' },
  { key: 'enabled', label: '启用状态' },
  { key: 'updated_at', label: '最近更新' },
  { key: 'actions', label: '操作' },
]

const subscriptionColumns: Column[] = [
  { key: 'channel_id', label: '通知渠道' },
  { key: 'provider_id', label: '供应商范围' },
  { key: 'event_type', label: '事件类型' },
  { key: 'enabled', label: '订阅状态' },
  { key: 'updated_at', label: '最近更新' },
  { key: 'actions', label: '操作' },
]

const deliveryColumns: Column[] = [
  { key: 'channel_name', label: '通知渠道' },
  { key: 'provider_name', label: '供应商' },
  { key: 'event_type', label: '事件类型' },
  { key: 'status', label: '投递状态' },
  { key: 'attempt_count', label: '尝试次数' },
  { key: 'next_attempt_at', label: '下次尝试' },
  { key: 'actions', label: '操作' },
]

const attemptColumns: Column[] = [
  { key: 'attempt_number', label: '尝试' },
  { key: 'status', label: '状态' },
  { key: 'http_status', label: 'HTTP' },
  { key: 'error_message', label: '结果' },
  { key: 'attempted_at', label: '尝试时间' },
]

const channelTypeOptions: SelectOption[] = [
  { value: 'feishu', label: '飞书机器人' },
  { value: 'email', label: 'SMTP 邮件' },
]

const eventTypeOptions: SelectOption[] = [
  { value: 'balance_low', label: '余额不足' },
  { value: 'balance_recovered', label: '余额恢复' },
]

const deliveryStatusOptions: SelectOption[] = [
  { value: 'pending', label: '待处理' },
  { value: 'sending', label: '投递中' },
  { value: 'delivered', label: '已送达' },
  { value: 'failed', label: '失败' },
]

const channelOptions = computed<SelectOption[]>(() =>
  channels.value.map((channel) => ({ value: channel.id, label: channel.name }))
)

const providerOptions = computed<SelectOption[]>(() => [
  { value: 0, label: '全部供应商' },
  ...providers.value.map((provider) => ({ value: provider.id, label: provider.name })),
])

const enabledChannelCount = computed(() => channels.value.filter((channel) => channel.enabled).length)
const enabledSubscriptionCount = computed(() => subscriptions.value.filter((subscription) => subscription.enabled).length)
const pendingDeliveryCount = computed(() =>
  deliveries.value.filter((delivery) => delivery.status === 'pending' || delivery.status === 'sending').length
)

let toastTimer: ReturnType<typeof setTimeout> | undefined

async function loadAll(): Promise<void> {
  loading.value = true
  error.value = ''
  try {
    const [channelResult, subscriptionResult, providerResult] = await Promise.all([
      listSupplierNotificationChannels(),
      listSupplierNotificationSubscriptions(),
      supplierProvidersAPI.list({ page: 1, page_size: 1000 }),
    ])
    channels.value = channelResult.items ?? []
    subscriptions.value = subscriptionResult.items ?? []
    providers.value = providerResult.items ?? []
    await loadDeliveries()
    lastLoadedAt.value = new Date().toISOString()
  } catch (err) {
    error.value = extractApiErrorMessage(err, '加载通知配置失败')
  } finally {
    loading.value = false
  }
}

async function loadDeliveries(): Promise<void> {
  deliveriesLoading.value = true
  try {
    const params: SupplierNotificationDeliveryListParams = {
      channel_id: asPositiveNumber(deliveryChannelFilter.value),
      event_type: asEventType(deliveryEventFilter.value),
      status: asDeliveryStatus(deliveryStatusFilter.value),
      page: deliveryPage.value,
      page_size: deliveryPageSize.value,
    }
    const result = await listSupplierNotificationDeliveries(params)
    deliveries.value = result.items ?? []
    deliveryTotal.value = result.total ?? 0
  } catch (err) {
    error.value = extractApiErrorMessage(err, '加载通知投递记录失败')
  } finally {
    deliveriesLoading.value = false
  }
}

function openCreateChannelDialog(): void {
  channelDialogTitle.value = '新增通知渠道'
  channelForm.value = createChannelForm()
  channelDialogVisible.value = true
}

function openEditChannelDialog(channel: SupplierNotificationChannelView): void {
  channelDialogTitle.value = `编辑「${channel.name}」通知渠道`
  channelForm.value = {
    id: channel.id,
    name: channel.name,
    channel_type: channel.channel_type === 'email' ? 'email' : 'feishu',
    enabled: channel.enabled,
    feishu: { webhook_url: '', secret: '' },
    email: {
      host: channel.email_host ?? '',
      port: String(channel.email_port ?? 587),
      username: channel.email_username ?? '',
      password: '',
      from: channel.email_from ?? '',
      to: (channel.email_to ?? []).join(', '),
      starttls: channel.email_starttls ?? true,
    },
    proxy: { url: channel.proxy_url ?? '', username: '', password: '' },
  }
  channelDialogVisible.value = true
}

function closeChannelDialog(): void {
  if (saving.value) return
  channelDialogVisible.value = false
  channelForm.value = null
}

async function saveChannel(): Promise<void> {
  const form = channelForm.value
  if (!form) return
  const name = form.name.trim()
  if (!name) {
    error.value = '通知渠道名称不能为空'
    return
  }

  const input: SupplierNotificationChannelInput = {
    name,
    channel_type: form.channel_type,
    enabled: form.enabled,
  }

  if (form.channel_type === 'feishu') {
    const webhook = form.feishu.webhook_url.trim()
    const secret = form.feishu.secret
    if (form.id === null && !webhook) {
      error.value = '新增飞书渠道必须填写 Webhook 地址'
      return
    }
    if (webhook || secret) {
      input.feishu = {
        ...(webhook ? { webhook_url: webhook } : {}),
        ...(secret ? { secret } : {}),
      }
    }
  } else {
    const host = form.email.host.trim()
    const port = Number(form.email.port)
    const username = form.email.username.trim()
    const from = form.email.from.trim()
    const recipients = parseRecipients(form.email.to)
    if (!host || !Number.isInteger(port) || port < 1 || port > 65535) {
      error.value = 'SMTP 主机和端口填写不完整，端口范围为 1-65535'
      return
    }
    if (!from || recipients.length === 0) {
      error.value = '请填写发件人和至少一个收件人地址'
      return
    }
    if (form.email.password && !username) {
      error.value = '填写 SMTP 密码时必须同时填写用户名'
      return
    }
    input.email = {
      host,
      port,
      username,
      ...(form.email.password ? { password: form.email.password } : {}),
      from,
      to: recipients,
      starttls: form.email.starttls,
    }
  }

  const proxyURL = form.proxy.url.trim()
  if (proxyURL) {
    input.proxy = {
      url: proxyURL,
      ...(form.proxy.username ? { username: form.proxy.username } : {}),
      ...(form.proxy.password ? { password: form.proxy.password } : {}),
    }
  } else if (form.proxy.username || form.proxy.password) {
    error.value = '填写代理账号或密码时必须同时填写代理地址'
    return
  }

  saving.value = true
  error.value = ''
  try {
    const saved = form.id === null
      ? await createSupplierNotificationChannel(input)
      : await updateSupplierNotificationChannel(form.id, input)
    if (form.id === null) {
      channels.value = [saved, ...channels.value]
    } else {
      replaceById(channels.value, saved)
    }
    closeChannelDialog()
    showToast(form.id === null ? '通知渠道已新增' : '通知渠道已保存')
  } catch (err) {
    error.value = extractApiErrorMessage(err, '保存通知渠道失败')
  } finally {
    saving.value = false
  }
}

async function toggleChannel(channel: SupplierNotificationChannelView, enabled: boolean): Promise<void> {
  if (savingChannelId.value !== null) return
  savingChannelId.value = channel.id
  error.value = ''
  try {
    const saved = await updateSupplierNotificationChannel(channel.id, {
      name: channel.name,
      channel_type: channel.channel_type === 'email' ? 'email' : 'feishu',
      enabled,
    })
    replaceById(channels.value, saved)
    showToast(`${channel.name}已${enabled ? '启用' : '停用'}`)
  } catch (err) {
    error.value = extractApiErrorMessage(err, '更新通知渠道开关失败')
  } finally {
    savingChannelId.value = null
  }
}

async function testChannel(channel: SupplierNotificationChannelView): Promise<void> {
  if (testingChannelId.value !== null) return
  testingChannelId.value = channel.id
  error.value = ''
  try {
    const result = await sendSupplierNotificationChannelTest(channel.id)
    showToast(`${channel.name}测试发送成功（HTTP ${result.http_status || '—'}）`)
  } catch (err) {
    error.value = extractApiErrorMessage(err, `测试发送「${channel.name}」失败`)
  } finally {
    testingChannelId.value = null
  }
}

async function removeChannel(channel: SupplierNotificationChannelView): Promise<void> {
  if (!window.confirm(`确定删除通知渠道“${channel.name}”吗？关联订阅也将无法继续投递。`)) return
  error.value = ''
  try {
    await deleteSupplierNotificationChannel(channel.id)
    channels.value = channels.value.filter((item) => item.id !== channel.id)
    subscriptions.value = subscriptions.value.filter((item) => item.channel_id !== channel.id)
    showToast('通知渠道已删除')
    if (deliveryChannelFilter.value === channel.id) {
      deliveryChannelFilter.value = null
      deliveryPage.value = 1
      await loadDeliveries()
    }
  } catch (err) {
    error.value = extractApiErrorMessage(err, '删除通知渠道失败')
  }
}

function openCreateSubscriptionDialog(): void {
  if (channels.value.length === 0) return
  subscriptionDialogTitle.value = '新增事件订阅'
  subscriptionForm.value = {
    id: null,
    channel_id: channels.value[0].id,
    provider_id: 0,
    event_type: 'balance_low',
    enabled: true,
  }
  subscriptionDialogVisible.value = true
}

function openEditSubscriptionDialog(subscription: SupplierNotificationSubscription): void {
  subscriptionDialogTitle.value = `编辑事件订阅 #${subscription.id}`
  subscriptionForm.value = {
    id: subscription.id,
    channel_id: subscription.channel_id,
    provider_id: subscription.provider_id ?? 0,
    event_type: subscription.event_type === 'balance_recovered' ? 'balance_recovered' : 'balance_low',
    enabled: subscription.enabled,
  }
  subscriptionDialogVisible.value = true
}

function closeSubscriptionDialog(): void {
  if (savingSubscription.value) return
  subscriptionDialogVisible.value = false
  subscriptionForm.value = null
}

async function saveSubscription(): Promise<void> {
  const form = subscriptionForm.value
  if (!form) return
  if (form.channel_id <= 0) {
    error.value = '请选择通知渠道'
    return
  }
  const input: SupplierNotificationSubscriptionInput = {
    channel_id: form.channel_id,
    provider_id: form.provider_id > 0 ? form.provider_id : null,
    event_type: form.event_type,
    enabled: form.enabled,
  }
  savingSubscription.value = true
  error.value = ''
  try {
    const saved = form.id === null
      ? await createSupplierNotificationSubscription(input)
      : await updateSupplierNotificationSubscription(form.id, input)
    if (form.id === null) {
      subscriptions.value = [saved, ...subscriptions.value]
    } else {
      replaceById(subscriptions.value, saved)
    }
    closeSubscriptionDialog()
    showToast(form.id === null ? '事件订阅已新增' : '事件订阅已保存')
  } catch (err) {
    error.value = extractApiErrorMessage(err, '保存事件订阅失败')
  } finally {
    savingSubscription.value = false
  }
}

async function toggleSubscription(subscription: SupplierNotificationSubscription, enabled: boolean): Promise<void> {
  if (savingSubscriptionId.value !== null) return
  savingSubscriptionId.value = subscription.id
  error.value = ''
  try {
    const saved = await updateSupplierNotificationSubscription(subscription.id, {
      channel_id: subscription.channel_id,
      provider_id: subscription.provider_id ?? null,
      event_type: subscription.event_type === 'balance_recovered' ? 'balance_recovered' : 'balance_low',
      enabled,
    })
    replaceById(subscriptions.value, saved)
    showToast(`事件订阅已${enabled ? '启用' : '停用'}`)
  } catch (err) {
    error.value = extractApiErrorMessage(err, '更新事件订阅开关失败')
  } finally {
    savingSubscriptionId.value = null
  }
}

async function removeSubscription(subscription: SupplierNotificationSubscription): Promise<void> {
  if (!window.confirm('确定删除这条事件订阅吗？')) return
  error.value = ''
  try {
    await deleteSupplierNotificationSubscription(subscription.id)
    subscriptions.value = subscriptions.value.filter((item) => item.id !== subscription.id)
    showToast('事件订阅已删除')
  } catch (err) {
    error.value = extractApiErrorMessage(err, '删除事件订阅失败')
  }
}

function applyDeliveryFilters(): void {
  deliveryPage.value = 1
  void loadDeliveries()
}

function resetDeliveryFilters(): void {
  deliveryChannelFilter.value = null
  deliveryEventFilter.value = null
  deliveryStatusFilter.value = null
  deliveryPage.value = 1
  void loadDeliveries()
}

function onDeliveryPageSizeChange(): void {
  deliveryPage.value = 1
  void loadDeliveries()
}

async function openDeliveryDetail(delivery: SupplierNotificationDelivery): Promise<void> {
  deliveryDetailVisible.value = true
  deliveryDetail.value = null
  deliveryAttempts.value = []
  deliveryAttemptsLoading.value = true
  error.value = ''
  try {
    const [detail, attempts] = await Promise.all([
      getSupplierNotificationDelivery(delivery.id),
      listSupplierNotificationDeliveryAttempts(delivery.id),
    ])
    deliveryDetail.value = detail
    deliveryAttempts.value = attempts.items ?? []
  } catch (err) {
    error.value = extractApiErrorMessage(err, '加载通知投递详情失败')
  } finally {
    deliveryAttemptsLoading.value = false
  }
}

function closeDeliveryDetail(): void {
  deliveryDetailVisible.value = false
  deliveryDetail.value = null
  deliveryAttempts.value = []
}

function createChannelForm(): ChannelForm {
  return {
    id: null,
    name: '',
    channel_type: 'feishu',
    enabled: true,
    feishu: { webhook_url: '', secret: '' },
    email: {
      host: '',
      port: '587',
      username: '',
      password: '',
      from: '',
      to: '',
      starttls: true,
    },
    proxy: { url: '', username: '', password: '' },
  }
}

function parseRecipients(value: string): string[] {
  return value.split(/[,;，；\n]+/).map((item) => item.trim()).filter(Boolean)
}

function replaceById<T extends { id: number }>(items: T[], saved: T): void {
  const index = items.findIndex((item) => item.id === saved.id)
  if (index >= 0) items[index] = saved
}

function asPositiveNumber(value: string | number | boolean | null): number | undefined {
  return typeof value === 'number' && value > 0 ? value : undefined
}

function asEventType(value: string | number | boolean | null): SupplierNotificationEventType | undefined {
  return value === 'balance_low' || value === 'balance_recovered' ? value : undefined
}

function asDeliveryStatus(value: string | number | boolean | null): SupplierNotificationDeliveryStatus | undefined {
  return value === 'pending' || value === 'sending' || value === 'delivered' || value === 'failed' ? value : undefined
}

function channelName(id: number): string {
  return channels.value.find((channel) => channel.id === id)?.name ?? `渠道 #${id}`
}

function providerName(id?: number | null): string {
  if (!id) return '全部供应商'
  return providers.value.find((provider) => provider.id === id)?.name ?? `供应商 #${id}`
}

function secretConfigured(channel: SupplierNotificationChannelView): boolean {
  return channel.feishu_secret_configured === true
}

function channelTypeLabel(channelType: string): string {
  return channelType === 'email' ? 'SMTP 邮件' : '飞书机器人'
}

function eventTypeLabel(eventType: string): string {
  return eventType === 'balance_recovered' ? '余额恢复' : '余额不足'
}

function deliveryStatusLabel(status: string): string {
  if (status === 'pending') return '待处理'
  if (status === 'sending') return '投递中'
  if (status === 'delivered') return '已送达'
  if (status === 'failed') return '失败'
  return status
}

function deliveryStatusTone(status: string): string {
  if (status === 'delivered') return 'good'
  if (status === 'failed') return 'bad'
  if (status === 'pending') return 'warn'
  return 'info'
}

function formatDateTime(value?: string | null): string {
  if (!value) return '—'
  const date = new Date(value)
  return Number.isNaN(date.getTime()) ? value : date.toLocaleString('zh-CN', { hour12: false })
}

function formatPayload(payload?: Record<string, unknown>): string {
  if (!payload) return '暂无载荷'
  try {
    return JSON.stringify(payload, null, 2)
  } catch {
    return '载荷无法展示'
  }
}

function showToast(message: string): void {
  toast.value = message
  if (toastTimer) clearTimeout(toastTimer)
  toastTimer = setTimeout(() => {
    toast.value = ''
  }, 4500)
}

onMounted(() => {
  void loadAll()
})
</script>

<style scoped>
.sp-notification-metrics { grid-template-columns: repeat(4, minmax(145px, 1fr)); }
.sp-notification-panel { margin-bottom: 1rem; }
.sp-notification-loaded { margin: 0; padding: 0.45rem 0.65rem; border-left: 0; background: var(--sp-panel-2); }
.sp-notification-filter-head { align-items: flex-start; }
.sp-notification-filters { justify-content: flex-end; }
.sp-notification-filter-control { min-width: 9rem; }
.sp-config-stack { display: grid; gap: 0.2rem; min-width: 12rem; }
.sp-proxy-note { color: var(--sp-cyan); }
.sp-delivery-error { max-width: 18rem; overflow: hidden; color: var(--sp-red); text-overflow: ellipsis; white-space: nowrap; }
.sp-form-section { display: grid; gap: 1rem; padding: 0.25rem 0 1.15rem; border-bottom: 1px solid var(--sp-line); }
.sp-form-section + .sp-form-section { padding-top: 1.15rem; }
.sp-form-section:last-child { padding-bottom: 0; border-bottom: 0; }
.sp-form-section-head { display: flex; align-items: flex-start; gap: 0.75rem; }
.sp-form-section-head > span { display: inline-grid; width: 1.75rem; height: 1.75rem; flex: 0 0 1.75rem; place-items: center; border: 1px solid color-mix(in srgb, var(--sp-cyan) 35%, var(--sp-line)); border-radius: 0.6rem; background: color-mix(in srgb, var(--sp-cyan) 8%, var(--sp-panel)); color: var(--sp-cyan); font-size: 0.72rem; font-weight: 800; }
.sp-form-section-head h3 { margin: 0; color: var(--sp-text); font-size: 0.95rem; }
.sp-form-section-head p { margin: 0.25rem 0 0; color: var(--sp-muted); font-size: 0.78rem; line-height: 1.55; }
.sp-form-grid { display: grid; grid-template-columns: repeat(2, minmax(0, 1fr)); gap: 0.85rem; }
.sp-form-control-wide { grid-column: 1 / -1; }
.sp-form-control { display: grid; gap: 0.4rem; }
.sp-form-label { color: var(--sp-text); font-size: 0.8rem; font-weight: 700; }
.sp-form-hint { margin: 0; color: var(--sp-muted); font-size: 0.75rem; line-height: 1.5; }
.sp-switch-field { display: flex; align-items: center; justify-content: space-between; gap: 1rem; color: var(--sp-text); font-size: 0.82rem; font-weight: 700; }
.sp-switch-field em { color: var(--sp-muted); font-size: 0.76rem; font-style: normal; font-weight: 600; }
.sp-delivery-detail { display: grid; gap: 1rem; }
.sp-detail-summary { display: grid; grid-template-columns: repeat(3, minmax(0, 1fr)); gap: 0.75rem; }
.sp-detail-summary > div { display: grid; gap: 0.25rem; min-width: 0; padding: 0.7rem 0.8rem; border: 1px solid var(--sp-line); border-radius: 0.7rem; background: var(--sp-panel-2); }
.sp-detail-summary span { color: var(--sp-muted); font-size: 0.73rem; }
.sp-detail-summary strong { overflow: hidden; color: var(--sp-text); font-size: 0.86rem; text-overflow: ellipsis; white-space: nowrap; }
.sp-detail-section { display: grid; gap: 0.65rem; }
.sp-detail-section-head { display: flex; align-items: baseline; justify-content: space-between; gap: 0.75rem; }
.sp-detail-section-head h3 { margin: 0; color: var(--sp-text); font-size: 0.9rem; }
.sp-detail-section-head span { color: var(--sp-muted); font-size: 0.75rem; }
.sp-payload { max-height: 20rem; margin: 0; overflow: auto; padding: 0.85rem; border: 1px solid var(--sp-line); border-radius: 0.7rem; background: var(--sp-panel-2); color: var(--sp-text); font: 0.76rem/1.55 ui-monospace, SFMono-Regular, Consolas, monospace; white-space: pre-wrap; word-break: break-word; }
.sp-empty-state { color: var(--sp-muted); text-align: center; }
.sp-pagination-row { display: flex; justify-content: flex-end; padding: 0.75rem 1rem 1rem; }

@media (max-width: 900px) {
  .sp-detail-summary { grid-template-columns: repeat(2, minmax(0, 1fr)); }
}

@media (max-width: 760px) {
  .sp-notification-metrics { grid-template-columns: repeat(2, minmax(0, 1fr)); }
  .sp-notification-filter-head { align-items: stretch; }
  .sp-notification-filters { justify-content: stretch; }
  .sp-notification-filter-control { min-width: 0; flex: 1 1 8rem; }
  .sp-form-grid { grid-template-columns: 1fr; }
  .sp-form-control-wide { grid-column: auto; }
}

@media (max-width: 480px) {
  .sp-notification-metrics { gap: 0.5rem; }
  .sp-notification-metrics .sp-metric-card { min-height: 92px; padding: 0.75rem; }
  .sp-notification-metrics .sp-metric-value { font-size: 1.5rem; }
  .sp-detail-summary { grid-template-columns: 1fr 1fr; gap: 0.5rem; }
  .sp-detail-section-head { align-items: flex-start; flex-direction: column; }
}
</style>
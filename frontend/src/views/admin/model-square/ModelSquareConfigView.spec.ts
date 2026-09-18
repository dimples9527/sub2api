import { flushPromises, mount } from '@vue/test-utils'
import { readFileSync } from 'node:fs'
import { resolve } from 'node:path'
import { computed, defineComponent, h } from 'vue'
import { beforeEach, describe, expect, it, vi } from 'vitest'

import ModelSquareConfigView from './ModelSquareConfigView.vue'
import { platformBadgeClass, platformTextClass } from '@/utils/platformColors'

function createDeferred<T>() {
  let resolve!: (value: T | PromiseLike<T>) => void
  let reject!: (reason?: unknown) => void
  const promise = new Promise<T>((promiseResolve, promiseReject) => {
    resolve = promiseResolve
    reject = promiseReject
  })
  return { promise, resolve, reject }
}

/**
 * Select 替身：把 #selected 与 #option 两个插槽都真的渲染出来，并支持点击切换平台。
 *
 * 为什么不能再用 `<div />` 敷衍：平台 chip 横栏删除后，切平台只剩这一个入口，stub 必须真的能切。
 * 而「未绑定 N」告警同时挂在两处 —— #selected（当前平台，必须一直可见）
 * 与 #option（展开时一眼看出问题在哪个平台）。只渲染其中一个都会漏测，
 * 得到「测试通过但其实那个位置根本没渲染」的假绿。
 *
 * 插槽签名已对着 `src/components/common/Select.vue` 核过（替身只有在镜像真实 API 时才可信）：
 *   - `:option="selectedOption"`（`modelValue` 匹配不到任何选项时传 `null`）
 *   - `:option="option" :selected="isSelected(option)"`
 * 页面正是靠 `option?.value || selectedPlatform` 兜住 `null` 那一种，所以这里必须也传 `null`。
 *
 * ⚠️ 两处**刻意**偏离真实组件，不要据此推断真实行为：
 *   1. 真实下拉是 `Teleport` 到 `body` 的，且只在 `isOpen` 时才渲染；替身把选项**内联、无条件**渲染，
 *      于是「点选项切平台」不需要先点开 trigger。测试若依赖「下拉是打开的」这类状态，替身给不了。
 *   2. 真实 trigger 是 `<button>`，替身也是，但替身不做 `disabled` / `clearable` / 键盘导航。
 */
const selectStub = defineComponent({
  props: {
    options: { type: Array, default: () => [] },
    modelValue: { type: String, default: '' },
  },
  emits: ['update:modelValue'],
  setup(props, { emit, slots }) {
    type StubOption = { value: string; label?: string }
    const list = () => (props.options || []) as StubOption[]
    const selectedOption = computed(() => list().find(option => option.value === props.modelValue) || null)
    return () => h('div', [
      h(
        'button',
        { type: 'button', class: 'select-trigger-stub' },
        slots.selected
          ? slots.selected({ option: selectedOption.value })
          : String(selectedOption.value?.label ?? ''),
      ),
      ...list().map(option => h(
        'button',
        {
          type: 'button',
          class: 'select-option-stub',
          onClick: () => emit('update:modelValue', option.value),
        },
        slots.option
          ? slots.option({ option, selected: option.value === props.modelValue })
          : String(option.label ?? ''),
      )),
    ])
  },
})

/**
 * 共享替身工厂。
 *
 * 为什么需要它：这个 spec 原来有 12 份近乎重复的内联 `stubs: {...}`，改动很容易只落到
 * 其中一份上，剩下 11 份继续用旧行为静默跑。实测踩过：把 Select 替身换成能渲染插槽的
 * 版本时，以为改的是 A、实际改到了 B，结果新写的测试里一个选项都选不到。
 *
 * 差异收敛成 7 个显式选项，**每个都对应一种真实的差异**，不是随手给的默认值。
 * 默认值一律取「最贴近真实组件」的那档 —— 替身渲染得越少，测试越容易
 * 「通过但其实那个位置根本没渲染」；要降级必须显式写出来，一眼能看见。
 */
type StubOptions = {
  /** TablePageLayout 透出哪些插槽。 */
  layoutSlots?: Array<'actions' | 'filters' | 'table' | 'default'>
  /** DataTable 透出的单元格插槽；不给就是空壳，渲染不出任何行内容。 */
  cells?: Array<'price_summary' | 'actions' | 'group_binding'>
  /** DataTable 只渲染第一行还是全部行。 */
  cellRows?: 'first' | 'all'
  /** none 空壳 / content 只透出内容 / toggle 带 show 开关 / titled 带 show 与标题。 */
  dialog?: 'none' | 'content' | 'toggle' | 'titled'
  /** none 空壳 / interactive 可点击的确认框（能测「点了确认之后做没做那件事」）。 */
  confirm?: 'none' | 'interactive'
  /** plain 裸 input / labelled 带 label 关联（价格字段靠它定位）。 */
  input?: 'plain' | 'labelled'
  /** empty 空壳 / rich 渲染 #selected 与 #option 且可点击切换。 */
  select?: 'empty' | 'rich'
}

/** BaseDialog 替身：渲染内容与 footer，并带标题（标题可用来断言弹窗确实打开了）。 */
const baseDialogStub = {
  props: ['show', 'title'],
  template: '<section v-if="show"><h2>{{ title }}</h2><slot /><footer><slot name="footer" /></footer></section>',
}

const confirmDialogStub = {
  props: ['show', 'title', 'message', 'confirmText', 'cancelText', 'danger'],
  emits: ['confirm', 'cancel'],
  template: [
    '<div v-if="show" class="confirm-stub">',
    '<span class="confirm-stub-title">{{ title }}</span>',
    // message 也要渲染出来：确认框里真正让人决定点不点的是这句话
    // （「会波及几个模型」「覆盖对方的改动」），只测标题等于没测到那层信息。
    '<span class="confirm-stub-message">{{ message }}</span>',
    '<button type="button" class="confirm-stub-confirm" @click="$emit(\'confirm\')">{{ confirmText }}</button>',
    '<button type="button" class="confirm-stub-cancel" @click="$emit(\'cancel\')">{{ cancelText }}</button>',
    '</div>',
  ].join(''),
}

const inputStub = {
  props: ['modelValue', 'label', 'placeholder', 'type', 'required'],
  emits: ['update:modelValue'],
  template: '<label><span>{{ label }}</span><input :aria-label="label" :value="modelValue" @input="$emit(\'update:modelValue\', $event.target.value)" /></label>',
}

function makeStubs(options: StubOptions = {}) {
  const {
    layoutSlots = ['actions', 'filters', 'table'],
    cells = [],
    cellRows = 'first',
    dialog = 'titled',
    confirm = 'interactive',
    input = 'labelled',
    select = 'rich',
  } = options

  return {
    AppLayout: { template: '<div><slot /></div>' },
    TablePageLayout: {
      template: `<div>${layoutSlots
        .map(slot => (slot === 'default' ? '<slot />' : `<slot name="${slot}" />`))
        .join('')}</div>`,
    },
    DataTable: cells.length === 0 ? { template: '<div />' } : dataTableStub(cells, cellRows),
    EmptyState: { template: '<div />' },
    BaseDialog: dialog === 'none'
      ? { template: '<div />' }
      : dialog === 'content'
        ? { template: '<div><slot /></div>' }
        : dialog === 'toggle'
          ? { props: ['show'], template: '<div v-if="show"><slot /><slot name="footer" /></div>' }
          : baseDialogStub,
    ConfirmDialog: confirm === 'none' ? { template: '<div />' } : confirmDialogStub,
    Input: input === 'plain' ? { template: '<input />' } : inputStub,
    SearchInput: { template: '<input />' },
    Select: select === 'empty' ? { template: '<div />' } : selectStub,
    TextArea: { template: '<textarea />' },
    PlatformIcon: { template: '<span />' },
    Icon: { template: '<span />' },
  }
}

/**
 * DataTable 替身。
 *
 * 必须镜像真实组件的**受控选择**接口（`selectable` + `selectedKeys` + `update:selectedKeys`）：
 * 批量绑定与批量清空的入口就是行勾选，替身不渲染勾选框的话，那些用例只能绕过 UI
 * 直接改组件内部状态 —— 测的就不是管理员实际走的那条路径了。
 * 接口已对着 `src/components/common/DataTable.vue` 核过（`:checked="selectedKeySet.has(...)"`、
 * `emit('update:selectedKeys', keys)`）。
 */
function dataTableStub(cells: string[], cellRows: 'first' | 'all') {
  const cellMarkup = (row: string) => cells.map(name => `<slot name="cell-${name}" :row="${row}" />`).join('')
  const selectMarkup = (row: string) => [
    '<input',
    ' v-if="selectable"',
    ' type="checkbox"',
    ' class="row-select"',
    ` :checked="(selectedKeys || []).map(String).includes(String(${row}.id))"`,
    ` @change="$emit('update:selectedKeys', $event.target.checked ? [...(selectedKeys || []), ${row}.id] : (selectedKeys || []).filter(key => String(key) !== String(${row}.id)))"`,
    ' />',
  ].join('')
  const rows = cellRows === 'all'
    ? `<template v-for="row in data" :key="row.id">${selectMarkup('row')}${cellMarkup('row')}</template>`
    : `<template v-if="data[0]">${selectMarkup('data[0]')}${cellMarkup('data[0]')}</template>`

  return {
    /*
      `selectable` 必须声明成 Boolean：真实组件用的是 `selectable?: boolean`（编译后 type=Boolean），
      页面写的是裸属性 `selectable`。替身若用数组语法声明（type 推断为 null），
      裸属性会原样传成空字符串 —— `v-if="selectable"` 判假，勾选框一个都不渲染，
      批量相关的用例全都会在「找不到勾选框」上失败。
    */
    props: {
      data: { type: Array, default: () => [] },
      selectable: { type: Boolean, default: false },
      selectedKeys: { type: Array, default: () => [] },
    },
    emits: ['update:selectedKeys'],
    template: `<div>${rows}</div>`,
  }
}

function mountPriceSummaryView() {
  return mount(ModelSquareConfigView, {
    global: {
      stubs: makeStubs({ layoutSlots: ['table'], cells: ['price_summary'], dialog: 'none', confirm: 'none', input: 'plain', select: 'empty' }),
    },
  })
}

/** 渲染「编辑模型」弹窗：BaseDialog 摊开内容，Input 保留 aria-label 以便定位价格输入框。 */
function mountModelDialogView() {
  return mount(ModelSquareConfigView, {
    global: {
      stubs: makeStubs({ cells: ['actions'], confirm: 'none', select: 'empty' }),
    },
  })
}

/**
 * 渲染分组绑定列：只摊开 DataTable 的 cell-group_binding 插槽。
 *
 * 刻意不 stub ModelGroupBindPicker —— 这个 spec 里它没有外部依赖（不用 i18n、不用 store），
 * 真实渲染才能测到「勾选之后确实写回配置」；替身会让交互测试在什么都没做的情况下通过。
 */
function mountGroupBindingView() {
  return mount(ModelSquareConfigView, {
    global: {
      stubs: makeStubs({ cells: ['group_binding'], cellRows: 'all', dialog: 'none', confirm: 'none', input: 'plain' }),
    },
  })
}

/** 渲染未保存改动相关区域：工具条、表格、编辑弹窗，以及会真实反映 show 的 ConfirmDialog。 */
function mountDirtyTrackingView() {
  return mount(ModelSquareConfigView, {
    global: {
      stubs: makeStubs({ cells: ['actions'] }),
    },
  })
}

/** 走真实点击路径制造一次未保存改动：编辑模型 -> 改输入价 -> 保存弹窗。 */
async function makeUnsavedEdit(wrapper: ReturnType<typeof mountDirtyTrackingView>): Promise<void> {
  await wrapper.findAll('button').find(button => button.text() === '编辑')!.trigger('click')
  await wrapper.find('input[aria-label="输入价格（USD / 1M Tokens）"]').setValue('9')
  await wrapper.findAll('button').find(button => button.text() === '保存')!.trigger('click')
  await flushPromises()
}

const { adminApiMock, appStoreMock, onBeforeRouteLeaveMock } = vi.hoisted(() => ({
  adminApiMock: {
    modelSquareConfig: {
      get: vi.fn(),
      update: vi.fn(),
      getModelPricing: vi.fn(),
      listSyncAccounts: vi.fn(),
    },
    modelSquare: {
      loadGroupContext: vi.fn(),
    },
    customPlatforms: {
      list: vi.fn(),
    },
    accounts: {
      list: vi.fn(),
    },
  },
  appStoreMock: {
    showError: vi.fn(),
    showSuccess: vi.fn(),
  },
  onBeforeRouteLeaveMock: vi.fn(),
}))

vi.mock('@/api/admin', () => ({
  adminAPI: adminApiMock,
}))

vi.mock('@/stores/app', () => ({
  useAppStore: () => appStoreMock,
}))

// 单测里直接 mount 组件、没有 <router-view>，不 mock 的话 vue-router 会对每次挂载
// 打一条 "No active route record" 警告。项目里其它 spec 也是这么 mock 的。
vi.mock('vue-router', () => ({
  onBeforeRouteLeave: onBeforeRouteLeaveMock,
}))

describe('model square config wiring', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    adminApiMock.modelSquareConfig.get.mockResolvedValue({ platforms: [], updated_at: null })
    adminApiMock.modelSquareConfig.update.mockResolvedValue({ platforms: [], updated_at: null })
    adminApiMock.customPlatforms.list.mockResolvedValue([
      {
        id: 1,
        code: 'custom-alpha',
        name: '自定义平台 Alpha',
        enabled: true,
        sort_order: 0,
        created_at: '2026-08-11T00:00:00Z',
        updated_at: '2026-08-11T00:00:00Z',
      },
      {
        id: 2,
        code: 'custom-sigma',
        name: '自定义平台 Sigma',
        enabled: true,
        sort_order: 10,
        created_at: '2026-08-11T00:00:00Z',
        updated_at: '2026-08-11T00:00:00Z',
      },
    ])
    adminApiMock.accounts.list.mockResolvedValue({ items: [] })
    adminApiMock.modelSquareConfig.getModelPricing.mockResolvedValue({ found: false })
    adminApiMock.modelSquareConfig.listSyncAccounts.mockResolvedValue([])
    adminApiMock.modelSquare.loadGroupContext.mockResolvedValue({
      groups: [],
      platformOverrides: new Map(),
    })
  })

  it('registers the admin route between model monitor and announcements', () => {
    const routerSource = readFileSync(resolve(process.cwd(), 'src/router/index.ts'), 'utf8')
    const modelMonitorIndex = routerSource.indexOf("path: '/admin/model-monitor/custom-platforms'")
    const configIndex = routerSource.indexOf("path: '/admin/model-square/config'")
    const announcementsIndex = routerSource.indexOf("path: '/admin/announcements'")

    expect(configIndex).toBeGreaterThan(modelMonitorIndex)
    expect(configIndex).toBeLessThan(announcementsIndex)
    expect(routerSource).toContain("import('@/views/admin/model-square/ModelSquareConfigView.vue')")
  })

  it('registers the sidebar entry as an admin-only item between model monitor and announcements', () => {
    const sidebarSource = readFileSync(resolve(process.cwd(), 'src/components/layout/AppSidebar.vue'), 'utf8')
    const modelMonitorIndex = sidebarSource.indexOf("path: '/admin/model-monitor'")
    const configIndex = sidebarSource.indexOf("path: '/admin/model-square/config'")
    const announcementsIndex = sidebarSource.indexOf("path: '/admin/announcements'")

    expect(configIndex).toBeGreaterThan(modelMonitorIndex)
    expect(configIndex).toBeLessThan(announcementsIndex)
    expect(sidebarSource).toContain("label: '模型广场配置'")
    expect(sidebarSource).toContain('hideInSimpleMode: true')
  })

  it('wires the model square config API to the new backend endpoints', () => {
    const apiSource = readFileSync(resolve(process.cwd(), 'src/api/admin/modelSquareConfig.ts'), 'utf8')
    expect(apiSource).toContain("/admin/upstream-management/model-square/config")
    expect(apiSource).toContain('/admin/upstream-management/model-square/sync-accounts')
    expect(apiSource).toContain('export async function get()')
    expect(apiSource).toContain('export async function update(payload: ModelSquareConfigPayload)')
    expect(apiSource).toContain('export async function listSyncAccounts(platform: string)')
  })

  it('includes custom platforms in the model square platform lists', () => {
    const viewSource = readFileSync(resolve(process.cwd(), 'src/views/admin/model-square/ModelSquareConfigView.vue'), 'utf8')

    expect(viewSource).toContain('const customPlatforms = ref<CustomPlatform[]>([])')
    expect(viewSource).toContain('for (const item of customPlatforms.value)')
    expect(viewSource).toContain('rank: 1')
    expect(viewSource).toContain('setCustomPlatformLabels(customPlatformList)')
  })

  it('keeps only input, output and cache prices and uses the model square pricing source', () => {
    const viewSource = readFileSync(resolve(process.cwd(), 'src/views/admin/model-square/ModelSquareConfigView.vue'), 'utf8')
    const apiSource = readFileSync(resolve(process.cwd(), 'src/api/admin/modelSquareConfig.ts'), 'utf8')

    expect(viewSource).toContain("{ key: 'price_summary', label: '价格（每 1M Tokens）'")
    expect(viewSource).toContain("{ key: 'input_price', label: '输入价格' }")
    expect(viewSource).toContain("{ key: 'output_price', label: '输出价格' }")
    expect(viewSource).toContain("{ key: 'cache_write_price', label: '缓存写入价格' }")
    expect(viewSource).toContain("{ key: 'cache_read_price', label: '缓存读取价格' }")
    expect(viewSource).not.toContain('input_price_priority')
    expect(viewSource).not.toContain('output_price_priority')
    expect(viewSource).not.toContain('cache_write_price_priority')
    expect(viewSource).not.toContain('cache_read_price_priority')
    expect(viewSource).not.toContain('cache_write_1h_price')
    expect(viewSource).not.toContain('image_input_price')
    expect(viewSource).not.toContain('image_output_price')
    expect(viewSource).not.toContain('per_request_price')
    expect(viewSource).toContain('模型配置中心')
    expect(viewSource).not.toContain('Model Square Config')
    expect(viewSource).toContain('官方参考价格来自项目动态价格目录')
    expect(viewSource).toContain('PRICE_PER_MILLION_TOKENS')
    expect(viewSource).toContain('displayPriceToStoredPrice')
    expect(viewSource).toContain('modelPriceSlots(row)')
    // 价格列的唯一取数口径：四个价格位的键、标签、色调都从这张描述表出
    expect(viewSource).toContain('PRICE_SLOT_DESCRIPTORS')
    expect(viewSource).toContain("label: '缓存写入'")
    expect(viewSource).toContain('price-card')
    expect(viewSource).toContain('adminAPI.modelSquareConfig.getModelPricing')
    expect(viewSource).not.toContain('adminAPI.channels.getModelDefaultPricing')
    expect(viewSource).toContain('isOfficialReferencePriceValue')
    expect(viewSource).toContain('只会回填当前未填写的价格字段')
    expect(apiSource).toContain('/admin/upstream-management/model-square/model-pricing')
    expect(apiSource).toContain('export async function getModelPricing')
  })

  /*
    平台 chip 横栏删除后，「自定义平台要按 sort_order 排在内置平台之后」这条约束
    改由平台 Select 的选项承载 —— 它现在是唯一列出全部平台的地方。
  */
  it('renders custom platforms in the platform selector', async () => {
    const wrapper = mount(ModelSquareConfigView, {
      global: {
        stubs: makeStubs({ layoutSlots: ['actions', 'filters', 'default'], dialog: 'content', confirm: 'none', input: 'plain' }),
      },
    })

    await flushPromises()

    const text = wrapper.text()
    expect(text.indexOf('自定义平台 Alpha')).toBeGreaterThanOrEqual(0)
    expect(text.indexOf('自定义平台 Sigma')).toBeGreaterThanOrEqual(0)
    expect(text.indexOf('自定义平台 Alpha')).toBeLessThan(text.indexOf('自定义平台 Sigma'))
    expect(wrapper.text()).toContain('自定义平台 Sigma')
  })

  it('loads group-effective-platform accounts when opening the sync dialog', async () => {
    adminApiMock.modelSquareConfig.get.mockResolvedValue({
      updated_at: null,
      platforms: [{
        platform: 'glm',
        name: 'GLM',
        models: [],
      }],
    })
    adminApiMock.modelSquareConfig.listSyncAccounts.mockResolvedValue([{
      id: 11,
      name: 'glm-group-account',
      platform: 'openai',
      type: 'api_key',
      status: 'active',
      group_ids: [101],
      group_names: ['glm-group'],
      effective_platform: 'glm',
    }])

    const wrapper = mount(ModelSquareConfigView, {
      global: {
        stubs: makeStubs({ dialog: 'toggle', confirm: 'none', input: 'plain' }),
      },
    })

    await flushPromises()
    await wrapper.findAll('.select-option-stub').find(button => button.text().includes('GLM'))!.trigger('click')
    await wrapper.findAll('button.btn-secondary')[2].trigger('click')
    await flushPromises()

    expect(adminApiMock.modelSquareConfig.listSyncAccounts).toHaveBeenCalledWith('glm')
    expect(wrapper.text()).toContain('glm-group-account')
  })

  /*
    synced_from_account_name 一直被写进配置（submitSyncDialog 里赋的值），却从来没显示过。
    「上次同步」只说了什么时候，没说用的是哪个账号 —— 而「上次是不是用了另一个账号」
    正是决定要不要重新同步的关键信息。
  */
  it('shows which account the platform was last synced from', async () => {
    const openSyncDialog = async (platform: Record<string, unknown>) => {
      adminApiMock.modelSquareConfig.get.mockResolvedValue({ updated_at: null, platforms: [platform] })
      adminApiMock.modelSquareConfig.listSyncAccounts.mockResolvedValue([])
      const wrapper = mountDirtyTrackingView()
      await flushPromises()
      await flushPromises()
      // 先切到 GLM：currentConfig 跟着 selectedPlatform 走，默认落在第一个内置平台上。
      await wrapper.findAll('.select-option-stub').find(button => button.text().includes('GLM'))!.trigger('click')
      await flushPromises()
      // 按文字找按钮而不是按下标：替身渲染出的按钮数量会随 stub 变化，下标很脆。
      await wrapper.findAll('button').find(button => button.text().includes('同步账号模型'))!.trigger('click')
      await flushPromises()
      return wrapper
    }

    const synced = await openSyncDialog({
      platform: 'glm',
      name: 'GLM',
      synced_from_account_id: 11,
      synced_from_account_name: 'glm-group-account',
      synced_at: '2026-09-17T00:00:00Z',
      models: [],
    })
    const syncedCards = synced.findAll('.sync-meta-card')
    expect(syncedCards).toHaveLength(4)
    expect(syncedCards[3].find('span').text()).toBe('上次同步账号')
    expect(syncedCards[3].find('strong').text()).toBe('glm-group-account')

    // 从没同步过时要显示占位符而不是一片空白 —— 空白会让人以为页面坏了
    const never = await openSyncDialog({ platform: 'glm', name: 'GLM', models: [] })
    expect(never.findAll('.sync-meta-card')[3].find('strong').text()).toBe('—')
  })

  it('renders configured prices directly in the model list', async () => {
    adminApiMock.modelSquareConfig.get.mockResolvedValue({
      updated_at: null,
      platforms: [{
        platform: 'openai',
        name: 'OpenAI',
        models: [{
          id: 'gpt-5.5',
          display_name: 'GPT-5.5',
          source: 'manual',
          input_price: 0.000005,
          output_price: 0.00003,
          cache_write_price: 0.0000075,
          cache_read_price: 0.0000005,
        }],
      }],
    })

    const wrapper = mount(ModelSquareConfigView, {
      global: {
        stubs: makeStubs({ cells: ['price_summary'], dialog: 'content', confirm: 'none', input: 'plain', select: 'empty' }),
      },
    })

    await flushPromises()

    // 四个价格位各一张卡片，标签随卡片走 —— 不再有「基础 / 缓存」分组框：
    // 分组标题不携带信息（输入输出的关系读标签就懂），只是多一层视觉噪音。
    // 顺序与展示页列表视图一致：输入 → 输出 → 缓存读取 → 缓存写入。
    const cards = wrapper.findAll('.price-card')
    expect(cards.map(card => card.find('.price-card-label').text()))
      .toEqual(['输入', '输出', '缓存读取', '缓存写入'])
    expect(cards.map(card => card.find('.price-card-value').text()))
      .toEqual(['$5', '$30', '$0.50', '$7.50'])
    // 单位同样照展示页：只给输入/输出，缓存两项不写。
    expect(wrapper.findAll('.price-card-meta small').map(node => node.text()))
      .toEqual(['$/百万 tokens', '$/百万 tokens'])
    // 四个价格都自己配了，一张参考价标签都不该出现
    expect(wrapper.findAll('.price-card-tag').length).toBe(0)
    expect(wrapper.text()).not.toContain('优先级')
    expect(wrapper.text()).not.toContain('图像')
  })

  it('stores manually entered token prices as per-token values after showing per-million-token inputs', async () => {
    adminApiMock.modelSquareConfig.get.mockResolvedValue({ updated_at: null, platforms: [] })

    const wrapper = mount(ModelSquareConfigView, {
      global: {
        stubs: makeStubs({ confirm: 'none', select: 'empty' }),
      },
    })

    await flushPromises()

    const addButton = wrapper.findAll('button').find(button => button.text().includes('添加模型'))
    expect(addButton).toBeTruthy()
    await addButton!.trigger('click')

    await wrapper.find('input[aria-label="模型 ID"]').setValue('gpt-5.5')
    await wrapper.find('input[aria-label="输入价格（USD / 1M Tokens）"]').setValue('5')
    await wrapper.find('input[aria-label="输出价格（USD / 1M Tokens）"]').setValue('30')
    await wrapper.find('input[aria-label="缓存写入价格（USD / 1M Tokens）"]').setValue('6.25')
    await wrapper.find('input[aria-label="缓存读取价格（USD / 1M Tokens）"]').setValue('0.5')

    const submitButton = wrapper.findAll('button').find(button => button.text() === '保存')
    expect(submitButton).toBeTruthy()
    await submitButton!.trigger('click')

    const saveButton = wrapper.findAll('button').find(button => button.text().includes('保存配置'))
    expect(saveButton).toBeTruthy()
    await saveButton!.trigger('click')
    await flushPromises()

    const savedPayload = adminApiMock.modelSquareConfig.update.mock.calls[0][0]
    expect(savedPayload.platforms[0].models[0]).toEqual({
      id: 'gpt-5.5',
      display_name: 'gpt-5.5',
      source: 'manual',
      input_price: 0.000005,
      output_price: 0.00003,
      cache_write_price: 0.00000625,
      cache_read_price: 0.0000005,
    })
  })

  it('保存时原样带回本页未展示的价格字段，不会把它们抹成 null', async () => {
    // 后端支持 12 个价格位，本页只展示其中 4 个；保存走的是整块 PUT，后端整体覆盖配置。
    // 所以未展示的字段必须原样带回：有官方参考价兜底的那几个一旦被写成 null，
    // 展示页会静默回退成官方价（管理员以为自己的定价生效了）；
    // per_request_price 没有官方兜底，会变成「未设置」不再显示，而不是写成 0。
    adminApiMock.modelSquareConfig.get.mockResolvedValue({
      updated_at: null,
      platforms: [{
        platform: 'openai',
        name: 'OpenAI',
        models: [{
          id: 'gpt-5.5',
          display_name: 'GPT-5.5',
          source: 'sync',
          input_price: 0.000005,
          cache_write_1h_price: 0.000007,
          input_price_priority: 0.000008,
          image_input_price: 0.00001,
          per_request_price: 0.00000012,
        }],
      }],
    })

    const wrapper = mount(ModelSquareConfigView, {
      global: {
        stubs: makeStubs({ cells: ['actions'], confirm: 'none', select: 'empty' }),
      },
    })

    await flushPromises()
    await flushPromises()

    const editButton = wrapper.findAll('button').find(button => button.text() === '编辑')
    expect(editButton).toBeTruthy()
    await editButton!.trigger('click')

    await wrapper.find('input[aria-label="输入价格（USD / 1M Tokens）"]').setValue('9')
    const submitButton = wrapper.findAll('button').find(button => button.text() === '保存')
    expect(submitButton).toBeTruthy()
    await submitButton!.trigger('click')

    const saveButton = wrapper.findAll('button').find(button => button.text().includes('保存配置'))
    expect(saveButton).toBeTruthy()
    await saveButton!.trigger('click')
    await flushPromises()

    const savedModel = adminApiMock.modelSquareConfig.update.mock.calls[0][0].platforms[0].models[0]
    // 本次改过的字段按新值写入（9 USD / 1M tokens -> 每 token 9e-6）
    expect(savedModel.input_price).toBe(0.000009)
    // 本页没展示的字段必须一个不少地保留
    expect(savedModel.cache_write_1h_price).toBe(0.000007)
    expect(savedModel.input_price_priority).toBe(0.000008)
    expect(savedModel.image_input_price).toBe(0.00001)
    expect(savedModel.per_request_price).toBe(0.00000012)
  })

  it('shows official reference prices for existing models without saving them as configured prices', async () => {
    adminApiMock.modelSquareConfig.get.mockResolvedValue({
      updated_at: null,
      platforms: [{
        platform: 'openai',
        name: 'OpenAI',
        models: [{ id: 'gpt-5.5', display_name: 'GPT-5.5', source: 'sync' }],
      }],
    })
    adminApiMock.modelSquareConfig.getModelPricing.mockResolvedValue({
      found: true,
      input_price: 0.000005,
      output_price: 0.00003,
      cache_read_price: 0.0000005,
    })

    const wrapper = mount(ModelSquareConfigView, {
      global: {
        stubs: makeStubs({ cells: ['price_summary'], dialog: 'content', confirm: 'none', input: 'plain', select: 'empty' }),
      },
    })

    await flushPromises()
    await flushPromises()

    expect(adminApiMock.modelSquareConfig.getModelPricing).toHaveBeenCalledWith('gpt-5.5')
    const cards = wrapper.findAll('.price-card')
    expect(cards.map(card => card.find('.price-card-value').text()))
      .toEqual(['$5', '$30', '$0.50', '—'])
    // 官方参考价必须逐张卡片可辨：原先徽标挂在组尾，组里两个价格有一个是参考价时读不出是哪个
    expect(cards.map(card => card.find('.price-card-tag').exists()))
      .toEqual([true, true, true, false])
    expect(wrapper.findAll('.price-card-tag').map(tag => tag.text()))
      .toEqual(['官方参考', '官方参考', '官方参考'])
    expect(wrapper.findAll('.price-card-unset').length).toBe(1)
    // 缺价显示破折号而不是 $0：配置页里 $0 会被读成「这一项免费」
    expect(cards[3].find('.price-card-value').text()).toBe('—')

    const saveButton = wrapper.findAll('button').find(button => button.text().includes('保存配置'))
    expect(saveButton).toBeTruthy()
    await saveButton!.trigger('click')
    await flushPromises()

    const savedPayload = adminApiMock.modelSquareConfig.update.mock.calls[0][0]
    expect(savedPayload.platforms[0].models[0]).toEqual({
      id: 'gpt-5.5',
      display_name: 'GPT-5.5',
      source: 'sync',
      input_price: null,
      output_price: null,
      cache_write_price: null,
      cache_read_price: null,
    })
  })

  it('prefills edit dialog prices from configured values first and official references for empty fields', async () => {
    adminApiMock.modelSquareConfig.get.mockResolvedValue({
      updated_at: null,
      platforms: [{
        platform: 'openai',
        name: 'OpenAI',
        models: [{
          id: 'gpt-5.5',
          display_name: 'GPT-5.5',
          source: 'manual',
          input_price: 0.000007,
        }],
      }],
    })
    adminApiMock.modelSquareConfig.getModelPricing.mockResolvedValue({
      found: true,
      input_price: 0.000005,
      output_price: 0.00003,
      cache_read_price: 0.0000005,
    })

    const wrapper = mount(ModelSquareConfigView, {
      global: {
        stubs: makeStubs({ cells: ['price_summary', 'actions'], confirm: 'none', select: 'empty' }),
      },
    })

    await flushPromises()
    await flushPromises()

    const editButton = wrapper.findAll('button').find(button => button.text() === '编辑')
    expect(editButton).toBeTruthy()
    await editButton!.trigger('click')

    expect((wrapper.find('input[aria-label="输入价格（USD / 1M Tokens）"]').element as HTMLInputElement).value).toBe('7')
    expect((wrapper.find('input[aria-label="输出价格（USD / 1M Tokens）"]').element as HTMLInputElement).value).toBe('30')
    expect((wrapper.find('input[aria-label="缓存读取价格（USD / 1M Tokens）"]').element as HTMLInputElement).value).toBe('0.50')
  })

  it('fills still-empty edit dialog prices after an in-flight official lookup completes', async () => {
    const pricingDeferred = createDeferred<{
      found: boolean
      input_price?: number
      output_price?: number
    }>()
    adminApiMock.modelSquareConfig.get.mockResolvedValue({
      updated_at: null,
      platforms: [{
        platform: 'openai',
        name: 'OpenAI',
        models: [{ id: 'gpt-5.5', display_name: 'GPT-5.5', source: 'manual' }],
      }],
    })
    adminApiMock.modelSquareConfig.getModelPricing.mockReturnValue(pricingDeferred.promise)

    const wrapper = mount(ModelSquareConfigView, {
      global: {
        stubs: makeStubs({ cells: ['price_summary', 'actions'], confirm: 'none', select: 'empty' }),
      },
    })

    await flushPromises()

    const editButton = wrapper.findAll('button').find(button => button.text() === '编辑')
    expect(editButton).toBeTruthy()
    await editButton!.trigger('click')
    expect((wrapper.find('input[aria-label="输出价格（USD / 1M Tokens）"]').element as HTMLInputElement).value).toBe('')

    pricingDeferred.resolve({ found: true, input_price: 0.000005, output_price: 0.00003 })
    await flushPromises()
    await flushPromises()

    expect((wrapper.find('input[aria-label="输入价格（USD / 1M Tokens）"]').element as HTMLInputElement).value).toBe('5')
    expect((wrapper.find('input[aria-label="输出价格（USD / 1M Tokens）"]').element as HTMLInputElement).value).toBe('30')
  })

  it('falls back to the catalog model name when a synced model has a provider prefix', async () => {
    adminApiMock.modelSquareConfig.get.mockResolvedValue({
      updated_at: null,
      platforms: [{
        platform: 'openai',
        name: 'OpenAI',
        models: [{ id: 'openai/gpt-5.5', display_name: 'GPT-5.5', source: 'sync' }],
      }],
    })
    adminApiMock.modelSquareConfig.getModelPricing.mockImplementation(async (model: string) => {
      if (model === 'gpt-5.5') {
        return {
          found: true,
          input_price: 0.000005,
          output_price: 0.00003,
        }
      }
      return { found: false }
    })

    const wrapper = mountPriceSummaryView()

    await flushPromises()
    await flushPromises()

    expect(adminApiMock.modelSquareConfig.getModelPricing).toHaveBeenCalledWith('openai/gpt-5.5')
    expect(adminApiMock.modelSquareConfig.getModelPricing).toHaveBeenCalledWith('gpt-5.5')
    expect(wrapper.findAll('.price-card-value').map(card => card.text()))
      .toEqual(['$5', '$30', '—', '—'])
    expect(wrapper.findAll('.price-card-tag').map(tag => tag.text()))
      .toEqual(['官方参考', '官方参考'])
  })

  it('shows lookup status in the price column while official reference prices are loading', async () => {
    const pricingDeferred = createDeferred<{ found: boolean }>()
    adminApiMock.modelSquareConfig.get.mockResolvedValue({
      updated_at: null,
      platforms: [{
        platform: 'openai',
        name: 'OpenAI',
        models: [{ id: 'gpt-5.5', display_name: 'GPT-5.5', source: 'sync' }],
      }],
    })
    adminApiMock.modelSquareConfig.getModelPricing.mockReturnValue(pricingDeferred.promise)

    const wrapper = mountPriceSummaryView()

    await flushPromises()

    expect(adminApiMock.modelSquareConfig.getModelPricing).toHaveBeenCalledWith('gpt-5.5')
    expect(wrapper.text()).toContain('正在查询官方参考价')
    expect(wrapper.text()).not.toContain('未设置')

    pricingDeferred.resolve({ found: false })
    await flushPromises()
  })

  it('shows not-found status in the price column when the official catalog has no price', async () => {
    adminApiMock.modelSquareConfig.get.mockResolvedValue({
      updated_at: null,
      platforms: [{
        platform: 'openai',
        name: 'OpenAI',
        models: [{ id: 'gpt-unknown', display_name: 'gpt-unknown', source: 'manual' }],
      }],
    })
    adminApiMock.modelSquareConfig.getModelPricing.mockResolvedValue({ found: false })

    const wrapper = mountPriceSummaryView()

    await flushPromises()
    await flushPromises()

    expect(adminApiMock.modelSquareConfig.getModelPricing).toHaveBeenCalledWith('gpt-unknown')
    expect(wrapper.text()).toContain('官方目录无价格')
    expect(wrapper.text()).not.toContain('未设置')
  })

  it('shows failure status in the price column when official price lookup fails', async () => {
    adminApiMock.modelSquareConfig.get.mockResolvedValue({
      updated_at: null,
      platforms: [{
        platform: 'openai',
        name: 'OpenAI',
        models: [{ id: 'gpt-5.5', display_name: 'GPT-5.5', source: 'manual' }],
      }],
    })
    adminApiMock.modelSquareConfig.getModelPricing.mockRejectedValue(new Error('network failed'))

    const wrapper = mountPriceSummaryView()

    await flushPromises()
    await flushPromises()

    expect(adminApiMock.modelSquareConfig.getModelPricing).toHaveBeenCalledWith('gpt-5.5')
    expect(wrapper.text()).toContain('官方价格查询失败')
    expect(wrapper.text()).not.toContain('未设置')
  })

  it('shows the official baseline and markup ratio inside the edit dialog', async () => {
    adminApiMock.modelSquareConfig.get.mockResolvedValue({
      updated_at: null,
      platforms: [{
        platform: 'openai',
        name: 'OpenAI',
        models: [{ id: 'gpt-5.5', display_name: 'GPT-5.5', source: 'manual', input_price: 0.00001 }],
      }],
    })
    adminApiMock.modelSquareConfig.getModelPricing.mockResolvedValue({
      found: true,
      input_price: 0.000005,
      output_price: 0.00003,
    })

    const wrapper = mountModelDialogView()
    await flushPromises()
    await flushPromises()

    const editButton = wrapper.findAll('button').find(button => button.text() === '编辑')
    expect(editButton).toBeTruthy()
    await editButton!.trigger('click')

    // 字段名必须继续挂在 Input 的 label 上：Input 不透传 $attrs，
    // 一旦改成页面层自己拼标签，内层 input 就丢掉无障碍名称了。
    expect(wrapper.find('input[aria-label="输入价格（USD / 1M Tokens）"]').exists()).toBe(true)

    const cards = wrapper.findAll('.model-price-field')
    expect(cards).toHaveLength(4)

    // 输入价格：已保存配置里有值（10 / 1M），官方基准 5 / 1M → 已自定义，并给出 2 倍关系
    expect(cards[0].find('.model-price-field-state').text()).toBe('已自定义')
    expect(cards[0].find('.model-price-field-baseline').text()).toBe('官方基准 $5 · 你的价 ×2.00')

    // 输出价格：已保存配置里没有值，弹窗把它预填成了官方基准 30。
    // 这里必须仍报「跟随官方」—— 只按表单非空判断会误报成已自定义。
    expect(cards[1].find('.model-price-field-state').text()).toBe('跟随官方')
    expect(cards[1].find('.model-price-field-baseline').text()).toBe('官方基准 $30')
  })

  it('marks dialog fields as unset rather than following official when the catalog has no baseline', async () => {
    adminApiMock.modelSquareConfig.get.mockResolvedValue({
      updated_at: null,
      platforms: [{
        platform: 'openai',
        name: 'OpenAI',
        models: [{ id: 'internal-embed', display_name: '内部向量模型', source: 'manual' }],
      }],
    })
    adminApiMock.modelSquareConfig.getModelPricing.mockResolvedValue({ found: false })

    const wrapper = mountModelDialogView()
    await flushPromises()
    await flushPromises()

    await wrapper.findAll('button').find(button => button.text() === '编辑')!.trigger('click')
    // 打开「没有已配价格」的模型会再触发一次官方价查询，状态会先回到 loading，
    // 所以这里必须再 flush 一轮，否则断言到的是查询中的文案。
    await flushPromises()
    await flushPromises()

    const cards = wrapper.findAll('.model-price-field')
    expect(cards).toHaveLength(4)
    for (const card of cards) {
      expect(card.find('.model-price-field-state').text()).toBe('未设置')
      expect(card.find('.model-price-field-baseline').text()).toBe('官方目录无参考价，留空则展示页不显示该价格')
    }
    // 没有官方基准价可跟随时，绝不能标成「跟随官方」
    expect(wrapper.text()).not.toContain('跟随官方')
  })

  it('tones the baseline panel by above, following, and unset', async () => {
    adminApiMock.modelSquareConfig.get.mockResolvedValue({
      updated_at: null,
      platforms: [{
        platform: 'openai',
        name: 'OpenAI',
        models: [{ id: 'gpt-5.5', display_name: 'GPT-5.5', source: 'manual', input_price: 0.00001 }],
      }],
    })
    adminApiMock.modelSquareConfig.getModelPricing.mockResolvedValue({
      found: true,
      input_price: 0.000005,
      output_price: 0.00003,
    })

    const wrapper = mountModelDialogView()
    await flushPromises()
    await flushPromises()

    await wrapper.findAll('button').find(button => button.text() === '编辑')!.trigger('click')

    const rows = wrapper.findAll('.model-baseline-row')
    expect(rows).toHaveLength(4)

    // 输入价格：自己填的 10 高于官方 5 → 加价色调 + 倍数
    expect(rows[0].classes()).toContain('is-high')
    expect(rows[0].find('.model-baseline-ratio').text()).toBe('×2.00')

    // 输出价格：没配过、跟着官方 30 → 跟随色调，且值直接显示官方价而不是破折号
    expect(rows[1].classes()).toContain('is-follow')
    expect(rows[1].find('.model-baseline-row-value').text()).toBe('$30')

    // 缓存写入 / 读取：既没配也没有官方基准 → 未设置
    expect(rows[2].classes()).toContain('is-unset')
    expect(rows[3].classes()).toContain('is-unset')
    expect(rows[2].find('.model-baseline-row-value').text()).toBe('未设置')

    expect(wrapper.find('.model-baseline-summary-value').text()).toBe('1 / 4')
  })

  it('shows the groups each configured model is manually bound to', async () => {
    adminApiMock.modelSquareConfig.get.mockResolvedValue({
      updated_at: null,
      platforms: [{
        platform: 'openai',
        name: 'OpenAI',
        models: [
          { id: 'gpt-5.5', display_name: 'GPT-5.5', source: 'manual', group_ids: [7] },
          { id: 'orphan-model', display_name: '孤儿模型', source: 'manual' },
        ],
      }],
    })
    adminApiMock.modelSquare.loadGroupContext.mockResolvedValue({
      groups: [{ id: 7, name: '默认分组', platform: 'openai', rate_multiplier: 1 }],
      platformOverrides: new Map(),
    })

    const wrapper = mountGroupBindingView()
    await flushPromises()
    await flushPromises()
    await flushPromises()

    // 分组归属来自配置里存的 group_ids，不再由渠道反推。
    expect(wrapper.text()).toContain('默认分组')
    // 没绑定的模型必须显式说出来 —— 它在展示页按任何分组都筛不到。
    expect(wrapper.findAll('.group-binding-empty')).toHaveLength(1)
    expect(wrapper.find('.group-binding-empty').text()).toBe('未绑定分组')
  })

  it('colors bound group chips by the platform each group actually belongs to', async () => {
    /*
      chip 的平台色是区分「本平台分组」与 composite 分组的唯一线索 —— 两者光看名字看不出来，
      而绑错了会让模型在展示页整组消失。
      断言比的是 platformColors 的返回值而不是几个 Tailwind 字面量：要锁的是
      「取色来自全站色板、且不同平台确实不同色」，写死类名只会在换色板时变成噪音。
    */
    adminApiMock.modelSquareConfig.get.mockResolvedValue({
      updated_at: null,
      platforms: [{
        platform: 'openai',
        name: 'OpenAI',
        models: [{ id: 'gpt-5.5', display_name: 'GPT-5.5', source: 'manual', group_ids: [7, 8] }],
      }],
    })
    adminApiMock.modelSquare.loadGroupContext.mockResolvedValue({
      groups: [
        { id: 7, name: '默认分组', platform: 'openai', rate_multiplier: 1 },
        { id: 8, name: '通用分组', platform: 'composite', rate_multiplier: 1 },
      ],
      platformOverrides: new Map(),
    })

    const wrapper = mountGroupBindingView()
    await flushPromises()
    await flushPromises()
    await flushPromises()

    const chips = wrapper.findAll('.group-chip')
    expect(chips.map(chip => chip.text())).toEqual(['默认分组', '通用分组'])

    expect(chips[0].classes()).toEqual(expect.arrayContaining(platformBadgeClass('openai').split(' ')))
    expect(chips[1].classes()).toEqual(expect.arrayContaining(platformBadgeClass('composite').split(' ')))
    // 两个平台色必须真的不同，否则「按平台着色」等于没做。
    expect(platformBadgeClass('openai')).not.toBe(platformBadgeClass('composite'))
  })

  it('colors a group chip by its overridden platform rather than its raw platform', async () => {
    /*
      平台覆盖会把分组挂到另一个平台下。着色若取 group.platform，就会出现
      「分组能被 listBindableGroups 选进来（它走的是有效平台）却标着另一个平台的颜色」这种自相矛盾。
      这里让原始平台与有效平台刻意不同：原始 anthropic，覆盖成 openai。
    */
    adminApiMock.modelSquareConfig.get.mockResolvedValue({
      updated_at: null,
      platforms: [{
        platform: 'openai',
        name: 'OpenAI',
        models: [{ id: 'gpt-5.5', display_name: 'GPT-5.5', source: 'manual', group_ids: [7] }],
      }],
    })
    adminApiMock.modelSquare.loadGroupContext.mockResolvedValue({
      groups: [{ id: 7, name: '默认分组', platform: 'anthropic', rate_multiplier: 1 }],
      platformOverrides: new Map([['7', 'openai']]),
    })

    const wrapper = mountGroupBindingView()
    await flushPromises()
    await flushPromises()
    await flushPromises()

    const chip = wrapper.find('.group-chip')
    expect(chip.classes()).toEqual(expect.arrayContaining(platformBadgeClass('openai').split(' ')))
    expect(chip.classes()).not.toEqual(expect.arrayContaining(platformBadgeClass('anthropic').split(' ')))
    // 颜色不是唯一载体：悬浮说明里必须写全平台名，色觉障碍下也能读出来。
    expect(chip.attributes('title')).toBe('默认分组 · OpenAI')
  })

  it('labels every group option in the bind dialog with its platform', async () => {
    adminApiMock.modelSquareConfig.get.mockResolvedValue({
      updated_at: null,
      platforms: [{
        platform: 'openai',
        name: 'OpenAI',
        models: [{ id: 'gpt-5.5', display_name: 'GPT-5.5', source: 'manual', group_ids: [7] }],
      }],
    })
    adminApiMock.modelSquare.loadGroupContext.mockResolvedValue({
      groups: [
        { id: 7, name: '默认分组', platform: 'openai', rate_multiplier: 1 },
        { id: 8, name: '通用分组', platform: 'composite', rate_multiplier: 1 },
      ],
      platformOverrides: new Map(),
    })

    const wrapper = mount(ModelSquareConfigView, {
      global: {
        // 这里要真的把弹窗内容渲染出来（mountGroupBindingView 把 BaseDialog 换成了空 div）。
        stubs: makeStubs({ cells: ['group_binding'], cellRows: 'all', confirm: 'none', input: 'plain', select: 'empty' }),
      },
    })
    await flushPromises()
    await flushPromises()
    await flushPromises()

    await wrapper.find('.group-binding').trigger('click')
    await flushPromises()

    /*
      第一个徽标是「当前平台」图例，之后才是各分组的平台。
      图例不能省：没有它，管理员只能看到一堆颜色，不知道哪个颜色对应本平台。
    */
    expect(wrapper.findAll('.model-group-bind-platform').map(node => node.text()))
      .toEqual(['OpenAI', 'OpenAI', 'Composite'])

    /*
      分组名也要着色，不只是徽标：徽标只占一小块，视线扫一列名字时颜色比文字标签更快。
      同样比 platformTextClass 的返回值而不是字面量。
    */
    const names = wrapper.findAll('.model-group-bind-name')
    expect(names[0].classes()).toEqual(expect.arrayContaining(platformTextClass('openai').split(' ')))
    expect(names[1].classes()).toEqual(expect.arrayContaining(platformTextClass('composite').split(' ')))
    expect(platformTextClass('openai')).not.toBe(platformTextClass('composite'))
  })

  it('lists bound groups the current platform can no longer bind, and lets them be unbound', async () => {
    /*
      换过平台、或分组被平台覆盖改到别处之后，绑定不会自动清理，配置里就留着一条当前平台
      选不到的 group_id。它会计入「已选 N」，但不在候选列表里就既看不见也取消不掉 ——
      管理员看到「已选 2、只勾了 1 个」，却没有任何入口把多出来的那个去掉。
    */
    adminApiMock.modelSquareConfig.get.mockResolvedValue({
      updated_at: null,
      platforms: [{
        platform: 'openai',
        name: 'OpenAI',
        models: [{ id: 'gpt-5.5', display_name: 'GPT-5.5', source: 'manual', group_ids: [7, 8] }],
      }],
    })
    adminApiMock.modelSquare.loadGroupContext.mockResolvedValue({
      groups: [
        { id: 7, name: '默认分组', platform: 'openai', rate_multiplier: 1 },
        // 平台对不上：openai 的模型绑不了 anthropic 的分组
        { id: 8, name: '异构分组', platform: 'anthropic', rate_multiplier: 1 },
      ],
      platformOverrides: new Map(),
    })
    adminApiMock.modelSquareConfig.update.mockImplementation(async (payload: unknown) => payload as never)

    const wrapper = mount(ModelSquareConfigView, {
      global: {
        stubs: makeStubs({ cells: ['group_binding'], cellRows: 'all', confirm: 'none', input: 'plain', select: 'empty' }),
      },
    })
    await flushPromises()
    await flushPromises()
    await flushPromises()

    await wrapper.find('.group-binding').trigger('click')
    await flushPromises()

    // 候选里只有同平台的分组
    const options = wrapper.findAll('.model-group-bind-option')
    expect(options).toHaveLength(1)
    expect(options[0].text()).toContain('默认分组')

    // 不兼容的那条单独列出来，并给出唯一的取消入口
    const orphans = wrapper.findAll('.model-group-bind-orphan')
    expect(orphans).toHaveLength(1)
    expect(orphans[0].text()).toContain('异构分组')
    expect(orphans[0].text()).toContain('取消绑定')

    await orphans[0].find('button').trigger('click')
    await flushPromises()

    // 取消后计数必须跟着降下来，否则「已选 N」还是对不上可见的勾选框
    expect(wrapper.find('.model-group-bind-count').text()).toBe('已选 1')

    await wrapper.findAll('button').find(button => button.text() === '保存')!.trigger('click')
    await flushPromises()
    await wrapper.findAll('button').find(button => button.text().includes('保存配置'))!.trigger('click')
    await flushPromises()

    const payload = adminApiMock.modelSquareConfig.update.mock.calls[0][0] as {
      platforms: Array<{ models: Array<{ group_ids?: number[] }> }>
    }
    expect(payload.platforms[0].models[0].group_ids).toEqual([7])
  })

  it('batch-binds groups to every selected model and appends to existing bindings', async () => {
    adminApiMock.modelSquareConfig.get.mockResolvedValue({
      updated_at: null,
      platforms: [{
        platform: 'openai',
        name: 'OpenAI',
        models: [
          { id: 'gpt-5.5', display_name: 'GPT-5.5', source: 'manual', group_ids: [7] },
          { id: 'gpt-5.4', display_name: 'GPT-5.4', source: 'manual' },
          { id: 'gpt-5.3', display_name: 'GPT-5.3', source: 'manual' },
        ],
      }],
    })
    adminApiMock.modelSquare.loadGroupContext.mockResolvedValue({
      groups: [
        { id: 7, name: '默认分组', platform: 'openai', rate_multiplier: 1 },
        { id: 8, name: '备用分组', platform: 'openai', rate_multiplier: 2 },
      ],
      platformOverrides: new Map(),
    })
    adminApiMock.modelSquareConfig.update.mockImplementation(async (payload: unknown) => payload as never)

    const wrapper = mount(ModelSquareConfigView, {
      global: {
        stubs: makeStubs({ cells: ['group_binding'], cellRows: 'all', confirm: 'none', input: 'plain', select: 'empty' }),
      },
    })
    await flushPromises()
    await flushPromises()
    await flushPromises()

    // 没有勾选时不出现批量操作条
    expect(wrapper.find('.batch-bar').exists()).toBe(false)

    const checkboxes = wrapper.findAll('.row-select')
    expect(checkboxes).toHaveLength(3)
    await checkboxes[0].setValue(true)
    await checkboxes[1].setValue(true)
    await flushPromises()

    expect(wrapper.find('.batch-bar-count').text()).toBe('已选 2 个模型')

    await wrapper.findAll('button').find(button => button.text().includes('绑定分组'))!.trigger('click')
    await flushPromises()

    expect(wrapper.find('h2').text()).toBe('批量绑定分组')
    // 批量模式不能沿用「留空则不出现在任何分组下」——这里留空只是「本次不追加」。
    expect(wrapper.find('.model-group-bind-note').text()).toContain('追加')
    expect(wrapper.find('.model-group-bind-note').text()).not.toContain('不出现在任何分组下')

    const target = wrapper.findAll('.model-group-bind-option').find(node => node.text().includes('备用分组'))!
    await target.find('input').setValue(true)
    await flushPromises()

    await wrapper.findAll('button').find(button => button.text() === '绑定')!.trigger('click')
    await flushPromises()

    await wrapper.findAll('button').find(button => button.text().includes('保存配置'))!.trigger('click')
    await flushPromises()

    const payload = adminApiMock.modelSquareConfig.update.mock.calls[0][0] as {
      platforms: Array<{ models: Array<{ id: string; group_ids?: number[] }> }>
    }
    const byId = new Map(payload.platforms[0].models.map(model => [model.id, model.group_ids]))
    // 追加而不是覆盖：原有的 [7] 必须留着
    expect(byId.get('gpt-5.5')).toEqual([7, 8])
    expect(byId.get('gpt-5.4')).toEqual([8])
    // 没勾选的模型一个字段都不该被碰到
    expect(byId.get('gpt-5.3')).toBeUndefined()
  })

  it('clears group bindings for the selected models only, after a confirmation', async () => {
    adminApiMock.modelSquareConfig.get.mockResolvedValue({
      updated_at: null,
      platforms: [{
        platform: 'openai',
        name: 'OpenAI',
        models: [
          { id: 'gpt-5.5', display_name: 'GPT-5.5', source: 'manual', group_ids: [7, 8] },
          { id: 'gpt-5.4', display_name: 'GPT-5.4', source: 'manual', group_ids: [7] },
          { id: 'gpt-5.3', display_name: 'GPT-5.3', source: 'manual', group_ids: [7] },
        ],
      }],
    })
    adminApiMock.modelSquare.loadGroupContext.mockResolvedValue({
      groups: [{ id: 7, name: '默认分组', platform: 'openai', rate_multiplier: 1 }],
      platformOverrides: new Map(),
    })
    adminApiMock.modelSquareConfig.update.mockImplementation(async (payload: unknown) => payload as never)

    const wrapper = mount(ModelSquareConfigView, {
      global: {
        stubs: makeStubs({ cells: ['group_binding'], cellRows: 'all', input: 'plain', select: 'empty' }),
      },
    })
    await flushPromises()
    await flushPromises()
    await flushPromises()

    const checkboxes = wrapper.findAll('.row-select')
    await checkboxes[0].setValue(true)
    await checkboxes[1].setValue(true)
    await flushPromises()

    await wrapper.findAll('button').find(button => button.text() === '清空分组')!.trigger('click')
    await flushPromises()

    // 一次抹掉多个模型的全部分组，必须先问一句，且说明会波及几个模型
    expect(wrapper.find('.confirm-stub-title').text()).toBe('清空分组绑定')
    expect(wrapper.find('.confirm-stub-message').text()).toContain('2 个模型')

    await wrapper.find('.confirm-stub-confirm').trigger('click')
    await flushPromises()

    await wrapper.findAll('button').find(button => button.text().includes('保存配置'))!.trigger('click')
    await flushPromises()

    const payload = adminApiMock.modelSquareConfig.update.mock.calls[0][0] as {
      platforms: Array<{ models: Array<{ id: string; group_ids?: number[] }> }>
    }
    const byId = new Map(payload.platforms[0].models.map(model => [model.id, model.group_ids]))
    expect(byId.get('gpt-5.5')).toBeUndefined()
    expect(byId.get('gpt-5.4')).toBeUndefined()
    // 没勾选的模型必须原样保留
    expect(byId.get('gpt-5.3')).toEqual([7])
  })

  it('drops the row selection when the platform changes', async () => {
    /*
      勾选键是模型 ID，而模型 ID 只在平台内唯一 —— 两个平台都可能有 gpt-5.5。
      不清空的话，切平台后那批勾选会落到另一个平台里的同名模型上，
      管理员点「批量绑定」时改的是他根本没看过的模型。
    */
    adminApiMock.modelSquareConfig.get.mockResolvedValue({
      updated_at: null,
      platforms: [
        { platform: 'openai', name: 'OpenAI', models: [{ id: 'gpt-5.5', display_name: 'GPT-5.5', source: 'manual' }] },
        { platform: 'gemini', name: 'Gemini', models: [{ id: 'gpt-5.5', display_name: '同名模型', source: 'manual' }] },
      ],
    })
    adminApiMock.modelSquare.loadGroupContext.mockResolvedValue({
      groups: [],
      platformOverrides: new Map(),
    })

    const wrapper = mount(ModelSquareConfigView, {
      global: {
        stubs: makeStubs({ cells: ['group_binding'], cellRows: 'all', confirm: 'none', input: 'plain' }),
      },
    })
    await flushPromises()
    await flushPromises()
    await flushPromises()

    await wrapper.find('.row-select').setValue(true)
    await flushPromises()
    expect(wrapper.find('.batch-bar-count').text()).toBe('已选 1 个模型')

    await wrapper.findAll('.select-option-stub').find(node => node.text().includes('Gemini'))!.trigger('click')
    await flushPromises()

    expect(wrapper.find('.batch-bar').exists()).toBe(false)
  })

  it('keeps showing the bound group regardless of channel state', async () => {
    /*
      分组归属是配置里手动绑定的，与渠道无关：渠道一个都没有，这一列照样把分组名显示出来，
      管理员才看得出模型本该出现在哪些分组里。

      渠道也不再产生任何告警 —— 可用性已改为只看分组绑定（见 api/admin/modelSquare.ts）。
    */
    adminApiMock.modelSquareConfig.get.mockResolvedValue({
      updated_at: null,
      platforms: [{
        platform: 'openai',
        name: 'OpenAI',
        models: [{ id: 'gpt-5.5', display_name: 'GPT-5.5', source: 'manual', group_ids: [7] }],
      }],
    })
    adminApiMock.modelSquare.loadGroupContext.mockResolvedValue({
      groups: [{ id: 7, name: '默认分组', platform: 'openai', rate_multiplier: 1 }],
      platformOverrides: new Map(),
    })

    const wrapper = mountGroupBindingView()
    await flushPromises()
    await flushPromises()
    await flushPromises()

    expect(wrapper.text()).toContain('默认分组')
    expect(wrapper.find('.group-binding-warn').exists()).toBe(false)
  })

  it('reports unknown instead of unbound when the group context fails to load', async () => {
    adminApiMock.modelSquareConfig.get.mockResolvedValue({
      updated_at: null,
      platforms: [{
        platform: 'openai',
        name: 'OpenAI',
        models: [{ id: 'gpt-5.5', display_name: 'GPT-5.5', source: 'manual', group_ids: [7] }],
      }],
    })
    adminApiMock.modelSquare.loadGroupContext.mockRejectedValue(new Error('boom'))

    const wrapper = mountGroupBindingView()
    await flushPromises()
    await flushPromises()
    await flushPromises()

    /*
      绑定了分组、但分组列表没拉到时必须退化成「—」，不能显示成「未绑定分组」——
      后者会让管理员去重复绑定一个本来就绑好的模型。
    */
    expect(wrapper.find('.group-binding-empty').exists()).toBe(false)
    expect(wrapper.find('.group-binding-pending').text()).toBe('—')
  })

  it('counts models without any group binding on the platform selector', async () => {
    adminApiMock.modelSquareConfig.get.mockResolvedValue({
      updated_at: null,
      platforms: [{
        platform: 'openai',
        name: 'OpenAI',
        models: [
          { id: 'gpt-5.5', display_name: 'GPT-5.5', source: 'manual', group_ids: [7] },
          { id: 'orphan-model', display_name: '孤儿模型', source: 'manual' },
        ],
      }],
    })
    // 分组上下文拉失败也不影响这个计数：group_ids 存在配置里，不依赖这个接口。
    adminApiMock.modelSquare.loadGroupContext.mockRejectedValue(new Error('boom'))

    const wrapper = mountGroupBindingView()
    await flushPromises()
    await flushPromises()
    await flushPromises()

    /*
      告警要挂在两处：当前平台（一直可见）与下拉选项（展开时看全部平台）。
      只测其中一个，另一个位置漏渲染也发现不了。
    */
    expect(wrapper.find('.select-trigger-stub .platform-option-warn').text()).toBe('未绑定 1')
    expect(wrapper.find('.select-option-stub .platform-option-warn').text()).toBe('未绑定 1')
  })

  it('no longer flags a model because its channels are disabled', async () => {
    /*
      这里曾经断言「渠道全部停用 → 标记渠道未启用」。该判据已按产品决策移除：
      请求路由走分组→账号（account_groups），渠道只负责定价、模型映射与模型限制，
      渠道停用不等于模型跑不通 —— 拿它当可用性条件会把有账号支撑的模型误标成不可用。

      渠道上下文已不再进入页面，所以这里也不再提供 channels 数据。
    */
    adminApiMock.modelSquareConfig.get.mockResolvedValue({
      updated_at: null,
      platforms: [{
        platform: 'openai',
        name: 'OpenAI',
        models: [{ id: 'gpt-5.5', display_name: 'GPT-5.5', source: 'manual', group_ids: [7] }],
      }],
    })
    adminApiMock.modelSquare.loadGroupContext.mockResolvedValue({
      groups: [{ id: 7, name: '默认分组', platform: 'openai', rate_multiplier: 1 }],
      platformOverrides: new Map(),
    })

    const wrapper = mountGroupBindingView()
    await flushPromises()
    await flushPromises()
    await flushPromises()

    expect(wrapper.text()).toContain('默认分组')
    expect(wrapper.find('.group-binding-warn').exists()).toBe(false)
    // 有分组归属就不算未绑定，平台 Select 上不该报数
    expect(wrapper.find('.platform-option-warn').exists()).toBe(false)
  })

  it('binds groups inside the edit dialog and persists them', async () => {
    adminApiMock.modelSquareConfig.get.mockResolvedValue({
      updated_at: null,
      platforms: [{
        platform: 'openai',
        name: 'OpenAI',
        models: [{ id: 'gpt-5.5', display_name: 'GPT-5.5', source: 'manual' }],
      }],
    })
    adminApiMock.modelSquare.loadGroupContext.mockResolvedValue({
      groups: [
        { id: 7, name: '默认分组', platform: 'openai', rate_multiplier: 1 },
        { id: 9, name: '跨平台分组', platform: 'anthropic', rate_multiplier: 1 },
      ],
      platformOverrides: new Map(),
    })
    adminApiMock.modelSquareConfig.update.mockImplementation(async (payload: unknown) => payload as never)

    const wrapper = mountModelDialogView()
    await flushPromises()
    await flushPromises()

    await wrapper.findAll('button').find(button => button.text() === '编辑')!.trigger('click')

    // 只列同平台分组：anthropic 的分组不该出现在 openai 模型的绑定项里。
    expect(wrapper.findAll('.model-group-bind-name').map(node => node.text())).toEqual(['默认分组'])

    await wrapper.find('.model-group-bind-checkbox').setValue(true)
    await wrapper.findAll('button').find(button => button.text() === '保存')!.trigger('click')
    await flushPromises()

    await wrapper.findAll('button').find(button => button.text().includes('保存配置'))!.trigger('click')
    await flushPromises()

    const payload = adminApiMock.modelSquareConfig.update.mock.calls[0][0] as {
      platforms: Array<{ models: Array<{ group_ids?: number[] }> }>
    }
    expect(payload.platforms[0].models[0].group_ids).toEqual([7])
  })

  it('binds groups from the table cell shortcut without touching prices', async () => {
    adminApiMock.modelSquareConfig.get.mockResolvedValue({
      updated_at: null,
      platforms: [{
        platform: 'openai',
        name: 'OpenAI',
        models: [{ id: 'gpt-5.5', display_name: 'GPT-5.5', source: 'manual', group_ids: [7] }],
      }],
    })
    adminApiMock.modelSquare.loadGroupContext.mockResolvedValue({
      groups: [
        { id: 7, name: '默认分组', platform: 'openai', rate_multiplier: 1 },
        { id: 8, name: '备用分组', platform: 'openai', rate_multiplier: 2 },
      ],
      platformOverrides: new Map(),
    })
    adminApiMock.modelSquareConfig.update.mockImplementation(async (payload: unknown) => payload as never)

    const wrapper = mount(ModelSquareConfigView, {
      global: {
        stubs: makeStubs({ cells: ['group_binding'], cellRows: 'all', confirm: 'none', input: 'plain', select: 'empty' }),
      },
    })
    await flushPromises()
    await flushPromises()
    await flushPromises()

    // 整格是快捷入口：点开即可改绑定，不必进「编辑模型」弹窗（那里同时挂着全部价格字段）。
    await wrapper.find('.group-binding').trigger('click')
    await flushPromises()

    expect(wrapper.find('h2').text()).toBe('绑定分组 · gpt-5.5')

    const target = wrapper.findAll('.model-group-bind-option').find(node => node.text().includes('备用分组'))!
    await target.find('input').setValue(true)
    await wrapper.findAll('button').find(button => button.text() === '保存')!.trigger('click')
    await flushPromises()

    await wrapper.findAll('button').find(button => button.text().includes('保存配置'))!.trigger('click')
    await flushPromises()

    const payload = adminApiMock.modelSquareConfig.update.mock.calls[0][0] as {
      platforms: Array<{ models: Array<{ group_ids?: number[] }> }>
    }
    // 原有绑定必须保留，新增的是追加而不是替换。
    expect(payload.platforms[0].models[0].group_ids).toEqual([7, 8])
  })

  it('flags unsaved edits and clears the flag once saved', async () => {
    adminApiMock.modelSquareConfig.get.mockResolvedValue({
      updated_at: '2026-09-17T00:00:00Z',
      platforms: [{
        platform: 'openai',
        name: 'OpenAI',
        models: [{ id: 'gpt-5.5', display_name: 'GPT-5.5', source: 'manual' }],
      }],
    })
    adminApiMock.modelSquareConfig.update.mockImplementation(async (payload: unknown) => payload as never)

    const wrapper = mountDirtyTrackingView()
    await flushPromises()
    await flushPromises()

    // 刚加载完没有任何改动，不能一进页面就喊「有未保存改动」
    expect(wrapper.find('button.btn-primary.is-dirty').exists()).toBe(false)
    expect(wrapper.text()).not.toContain('有未保存改动')

    await makeUnsavedEdit(wrapper)

    // 提示要落在保存按钮上：管理员的视线本来就在那儿（琥珀色提示环 + 圆点）
    const dirtySaveButton = wrapper.find('button.btn-primary.is-dirty')
    expect(dirtySaveButton.exists()).toBe(true)
    /*
      「有未保存改动」文案原本在 hero 横栏上，hero 删除后挪进保存按钮的 sr-only 文案。
      这里断言 text() 而不是可见性，是因为要守住的就是「它还在可访问树里」这件事 ——
      圆点是 aria-hidden 的纯装饰，只剩它的话无障碍上完全不知道有未保存改动。
    */
    expect(dirtySaveButton.text()).toContain('有未保存改动')

    await dirtySaveButton.trigger('click')
    await flushPromises()

    expect(wrapper.find('button.btn-primary.is-dirty').exists()).toBe(false)
    expect(wrapper.text()).not.toContain('有未保存改动')
  })

  /*
    后端 UpdateModelSquareConfig 只做 validate → normalize → 盖新的 updated_at → 整块覆盖，
    完全不比对入参里的 updated_at（已核对源码）。两个人同时编辑时，后保存的会静默覆盖
    前一个人的改动，双方都以为存成功了。下面两条锁住「提交前先比对」这个唯一的拦截点。
  */
  it('asks before overwriting when the config changed on the server', async () => {
    const loaded = {
      updated_at: '2026-09-17T00:00:00Z',
      platforms: [{
        platform: 'openai',
        name: 'OpenAI',
        models: [{ id: 'gpt-5.5', display_name: 'GPT-5.5', source: 'manual' }],
      }],
    }
    adminApiMock.modelSquareConfig.get
      .mockResolvedValueOnce(loaded)
      // 保存前的探测拿到的是「别人改过之后」的时间戳
      .mockResolvedValueOnce({ ...loaded, updated_at: '2026-09-17T01:00:00Z' })
    adminApiMock.modelSquareConfig.update.mockImplementation(async (payload: unknown) => payload as never)

    const wrapper = mountDirtyTrackingView()
    await flushPromises()
    await flushPromises()

    await makeUnsavedEdit(wrapper)
    await wrapper.findAll('button').find(button => button.text().includes('保存配置'))!.trigger('click')
    await flushPromises()

    // 关键：先问，而且问之前一个字节都不能写上去
    expect(wrapper.find('.confirm-stub-title').text()).toBe('配置已被他人修改')
    expect(adminApiMock.modelSquareConfig.update).not.toHaveBeenCalled()

    // 用户选了「覆盖保存」之后才真的提交
    await wrapper.find('.confirm-stub-confirm').trigger('click')
    await flushPromises()
    expect(adminApiMock.modelSquareConfig.update).toHaveBeenCalledTimes(1)
    expect(wrapper.find('.confirm-stub').exists()).toBe(false)
  })

  it('does not ask when the server copy is unchanged', async () => {
    adminApiMock.modelSquareConfig.get.mockResolvedValue({
      updated_at: '2026-09-17T00:00:00Z',
      platforms: [{
        platform: 'openai',
        name: 'OpenAI',
        models: [{ id: 'gpt-5.5', display_name: 'GPT-5.5', source: 'manual' }],
      }],
    })
    adminApiMock.modelSquareConfig.update.mockImplementation(async (payload: unknown) => payload as never)

    const wrapper = mountDirtyTrackingView()
    await flushPromises()
    await flushPromises()

    await makeUnsavedEdit(wrapper)
    await wrapper.findAll('button').find(button => button.text().includes('保存配置'))!.trigger('click')
    await flushPromises()

    // 没冲突就别多弹一个框 —— 每次保存都拦一下比不拦还烦
    expect(wrapper.find('.confirm-stub').exists()).toBe(false)
    expect(adminApiMock.modelSquareConfig.update).toHaveBeenCalledTimes(1)
  })

  it('asks before reloading when there are unsaved edits', async () => {
    adminApiMock.modelSquareConfig.get.mockResolvedValue({
      updated_at: '2026-09-17T00:00:00Z',
      platforms: [{
        platform: 'openai',
        name: 'OpenAI',
        models: [{ id: 'gpt-5.5', display_name: 'GPT-5.5', source: 'manual' }],
      }],
    })

    const wrapper = mountDirtyTrackingView()
    await flushPromises()
    await flushPromises()
    expect(wrapper.find('.confirm-stub').exists()).toBe(false)

    await makeUnsavedEdit(wrapper)

    const getCallsBefore = adminApiMock.modelSquareConfig.get.mock.calls.length
    await wrapper.findAll('button').find(button => button.text().includes('刷新'))!.trigger('click')
    await flushPromises()

    // 先弹确认，而且不能已经悄悄把配置重新拉了一遍（那样改动就已经没了）
    expect(wrapper.find('.confirm-stub-title').text()).toBe('刷新配置')
    expect(adminApiMock.modelSquareConfig.get.mock.calls.length).toBe(getCallsBefore)
  })

  it('blocks route navigation while there are unsaved edits', async () => {
    adminApiMock.modelSquareConfig.get.mockResolvedValue({
      updated_at: '2026-09-17T00:00:00Z',
      platforms: [{
        platform: 'openai',
        name: 'OpenAI',
        models: [{ id: 'gpt-5.5', display_name: 'GPT-5.5', source: 'manual' }],
      }],
    })
    const confirmSpy = vi.spyOn(window, 'confirm').mockReturnValue(false)

    const wrapper = mountDirtyTrackingView()
    await flushPromises()
    await flushPromises()

    const guard = onBeforeRouteLeaveMock.mock.calls[0][0] as () => unknown
    expect(typeof guard).toBe('function')

    // 没有改动就安静放行，别每次都拦一下
    expect(guard()).toBe(true)
    expect(confirmSpy).not.toHaveBeenCalled()

    await makeUnsavedEdit(wrapper)

    expect(guard()).toBe(false)
    expect(confirmSpy).toHaveBeenCalledTimes(1)

    // 用户确认离开后才放行
    confirmSpy.mockReturnValue(true)
    expect(guard()).toBe(true)

    confirmSpy.mockRestore()
  })

  it('asks the browser to confirm before unloading while there are unsaved edits', async () => {
    adminApiMock.modelSquareConfig.get.mockResolvedValue({
      updated_at: '2026-09-17T00:00:00Z',
      platforms: [{
        platform: 'openai',
        name: 'OpenAI',
        models: [{ id: 'gpt-5.5', display_name: 'GPT-5.5', source: 'manual' }],
      }],
    })
    // 直接拿注册进 window 的处理器来调，不真的 dispatch：
    // 组件在 onMounted 里挂的是 window 级监听，dispatch 会被前面用例遗留的实例一起接住。
    const addListenerSpy = vi.spyOn(window, 'addEventListener')

    const wrapper = mountDirtyTrackingView()
    await flushPromises()
    await flushPromises()

    const registered = addListenerSpy.mock.calls.find(([type]) => type === 'beforeunload')
    expect(registered).toBeTruthy()
    const handler = registered![1] as (event: Event) => void

    const clean = new Event('beforeunload', { cancelable: true })
    handler(clean)
    expect(clean.defaultPrevented).toBe(false)

    await makeUnsavedEdit(wrapper)

    const dirty = new Event('beforeunload', { cancelable: true })
    handler(dirty)
    expect(dirty.defaultPrevented).toBe(true)

    addListenerSpy.mockRestore()
  })
})

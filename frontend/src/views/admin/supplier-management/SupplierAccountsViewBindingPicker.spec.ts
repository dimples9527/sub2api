import { readFileSync } from 'node:fs'
import { dirname, resolve } from 'node:path'
import { fileURLToPath } from 'node:url'
import { describe, expect, it } from 'vitest'

/**
 * 编辑账号绑定弹窗的「分组区域自适应放大」守卫。
 *
 * 背景：GroupSelector 是通用组件（按项目约定不得修改），它把列表钉在 max-h-32（128px），
 * 分组一多只能在 128px 里上下滚。这里改为在页面层接管高度，并用源码断言钉住几个
 * 「改错了会静默失效」的点：
 *   - 选择器必须整条包在 :global() 里（弹窗 Teleport 后祖先在 body 下）
 *   - 必须用 max-height 而不是 height（否则分组少时弹窗留一大片空白）
 *   - 必须显式解除 max-h-32（只写 flex:1 撑不开）
 *   - 弹窗必须自带 --sp-* 兜底（页面根节点的变量在弹窗里拿不到）
 */
const here = dirname(fileURLToPath(import.meta.url))
const viewSource = readFileSync(resolve(here, 'SupplierAccountsView.vue'), 'utf-8')
const groupSelectorSource = readFileSync(
  resolve(here, '../../../components/common/GroupSelector.vue'),
  'utf-8'
)

const bindingDialogBlock = () => {
  const start = viewSource.indexOf(':global(.modal-content:has(.sp-account-binding-dialog)) {')
  expect(start).toBeGreaterThan(-1)
  return viewSource.slice(start, viewSource.indexOf('}', start))
}

describe('编辑账号绑定弹窗 · 分组区域自适应放大', () => {
  it('把分组列表包进独立容器，让高度能沿 flex 链传下去', () => {
    expect(viewSource).toContain('<div class="sp-account-binding-picker">')
    expect(viewSource).toMatch(
      /<div class="sp-account-binding-picker">\s*<GroupSelector\s+v-model="selectedBindingGroupIDs"/
    )
  })

  it('用 max-height 撑高弹窗而不是固定 height，避免分组少时留白', () => {
    expect(viewSource).toContain('max-height: min(88dvh, 820px)')
    // 注意：max-height 里也含 "height: min(" 子串，这里要求前面是空白/分号/花括号
    expect(bindingDialogBlock()).not.toMatch(/[\s;{]height:\s*min\(/)
  })

  it('整条选择器都包在 :global() 里，Teleport 后不会把 data-v 挂到 .modal-content 上', () => {
    expect(viewSource).toContain(':global(.modal-content:has(.sp-account-binding-dialog)) {')
    expect(viewSource).toContain(
      ':global(.modal-content:has(.sp-account-binding-dialog) .modal-body) {'
    )
  })

  it('解除 GroupSelector 内部列表的高度上限，让它吃掉剩余高度', () => {
    expect(viewSource).toContain('.sp-account-binding-picker :deep(> div) {')
    expect(viewSource).toContain('.sp-account-binding-picker :deep(.grid.grid-cols-2) {')
    const start = viewSource.indexOf('.sp-account-binding-picker :deep(.grid.grid-cols-2) {')
    const rule = viewSource.slice(start, viewSource.indexOf('}', start))
    expect(rule).toContain('max-height: none;')
  })

  it('定位列表容器时绕开 max-h- 类名（本页有一条既有的产品决策守卫禁止它）', () => {
    // SupplierLocalDataViews.spec.ts 断言本页源码不含 'max-h-'：
    // 账号表的分组列不得用 max-h-* 截断。用 .grid.max-h-32 当选择器会撞红，
    // 所以这里改用 .grid.grid-cols-2 定位同一个元素。
    expect(viewSource).not.toContain('max-h-')
  })

  it('弹窗自带完整 --sp-* 兜底，深浅两套都要有', () => {
    const light = bindingDialogBlock()
    for (const token of ['--sp-panel:', '--sp-panel-2:', '--sp-line:', '--sp-text:', '--sp-muted:']) {
      expect(light).toContain(token)
    }
    const darkStart = viewSource.indexOf(
      ':global(.dark .modal-content:has(.sp-account-binding-dialog)) {'
    )
    expect(darkStart).toBeGreaterThan(-1)
    const dark = viewSource.slice(darkStart, viewSource.indexOf('}', darkStart))
    expect(dark).toContain('--sp-panel: #172033;')
  })

  it('不修改通用组件 GroupSelector，只在页面层覆盖', () => {
    expect(groupSelectorSource).not.toContain('sp-account-binding')
    // 它的默认上限仍是 128px —— 这正是需要页面层覆盖的原因
    expect(groupSelectorSource).toContain('max-h-32')
  })
})

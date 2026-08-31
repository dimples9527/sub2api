import { flushPromises, mount, type VueWrapper } from '@vue/test-utils'
import { afterEach, describe, expect, it } from 'vitest'
import SupplierDayPicker from './SupplierDayPicker.vue'

function toDateInputValue(date: Date) {
  return `${date.getFullYear()}-${String(date.getMonth() + 1).padStart(2, '0')}-${String(date.getDate()).padStart(2, '0')}`
}

const yesterday = toDateInputValue(new Date(Date.now() - 24 * 60 * 60 * 1000))

let wrapper: VueWrapper | null = null

function mountPicker(props: { modelValue?: string; max?: string; disabled?: boolean } = {}) {
  wrapper = mount(SupplierDayPicker, {
    attachTo: document.body,
    props: { modelValue: '2026-08-15', max: '2026-08-20', ...props },
  })
  return wrapper
}

function panel() {
  return document.querySelector('[data-test="day-panel"]')
}

function cell(value: string) {
  return document.querySelector<HTMLButtonElement>(`[data-test="day-${value}"]`)
}

async function openPanel(props: { modelValue?: string; max?: string; disabled?: boolean } = {}) {
  const view = mountPicker(props)
  await view.find('[data-test="stat-date"]').trigger('click')
  await flushPromises()
  return view
}

afterEach(() => {
  wrapper?.unmount()
  wrapper = null
  // teleport 出去的面板带离场过渡，jsdom 里不会自然结束，手动清空避免污染下一条用例的 document 查询
  document.body.innerHTML = ''
})

describe('SupplierDayPicker', () => {
  it('触发按钮展示日期与星期，点击后弹出自定义面板', async () => {
    const view = mountPicker()

    const trigger = view.find('[data-test="stat-date"]')
    expect(trigger.text()).toContain('2026-08-15')
    expect(trigger.text()).toContain('周六')
    expect(panel()).toBeNull()

    await trigger.trigger('click')
    await flushPromises()

    expect(panel()).not.toBeNull()
    expect(document.querySelector('[data-test="panel-month"]')?.textContent).toBe('2026 年 8 月')
    expect(document.querySelectorAll('.sp-day-cell')).toHaveLength(42)
  })

  it('按周一为首列补齐相邻月份并标出所选日期', async () => {
    await openPanel()

    const cells = document.querySelectorAll('.sp-day-cell')
    expect(cells[0]?.getAttribute('data-day')).toBe('2026-07-27')
    expect(cells[0]?.classList.contains('is-outside')).toBe(true)
    expect(cell('2026-08-15')?.classList.contains('is-selected')).toBe(true)
    expect(cell('2026-08-15')?.classList.contains('is-outside')).toBe(false)
  })

  it('超过上限的日期与下一月按钮都禁用', async () => {
    await openPanel()

    expect(cell('2026-08-20')?.disabled).toBe(false)
    expect(cell('2026-08-21')?.disabled).toBe(true)
    expect(document.querySelector<HTMLButtonElement>('[data-test="next-month"]')?.disabled).toBe(true)
  })

  it('选择日期后同时抛出 v-model 与 change 并关闭面板', async () => {
    const view = await openPanel()

    cell('2026-08-10')?.click()
    await flushPromises()

    expect(view.emitted('update:modelValue')).toEqual([['2026-08-10']])
    expect(view.emitted('change')).toEqual([['2026-08-10']])
    expect(view.find('[data-test="stat-date"]').attributes('aria-expanded')).toBe('false')
  })

  it('切换上一月后保留面板并更新标题', async () => {
    await openPanel()

    document.querySelector<HTMLButtonElement>('[data-test="prev-month"]')?.click()
    await flushPromises()

    expect(document.querySelector('[data-test="panel-month"]')?.textContent).toBe('2026 年 7 月')
    expect(cell('2026-07-15')?.classList.contains('is-outside')).toBe(false)
  })

  it('昨天快捷按钮直接选中昨天', async () => {
    const view = await openPanel({ modelValue: yesterday, max: '' })

    document.querySelector<HTMLButtonElement>('[data-test="panel-yesterday"]')?.click()
    await flushPromises()

    expect(view.emitted('change')).toEqual([[yesterday]])
  })

  it('方向键在日期格之间移动焦点并跨月翻页', async () => {
    await openPanel()

    const grid = document.querySelector('.sp-day-grid')
    grid?.dispatchEvent(new KeyboardEvent('keydown', { key: 'ArrowLeft', bubbles: true }))
    await flushPromises()
    expect(cell('2026-08-14')?.getAttribute('tabindex')).toBe('0')

    for (let index = 0; index < 2; index += 1) {
      document.querySelector('.sp-day-grid')?.dispatchEvent(new KeyboardEvent('keydown', { key: 'ArrowUp', bubbles: true }))
      await flushPromises()
    }
    expect(document.querySelector('[data-test="panel-month"]')?.textContent).toBe('2026 年 7 月')
    expect(cell('2026-07-31')?.getAttribute('tabindex')).toBe('0')
  })

  it('Esc 与点击外部都会关掉面板', async () => {
    const view = await openPanel()

    panel()?.dispatchEvent(new KeyboardEvent('keydown', { key: 'Escape', bubbles: true }))
    await flushPromises()
    expect(view.find('[data-test="stat-date"]').attributes('aria-expanded')).toBe('false')

    await view.find('[data-test="stat-date"]').trigger('click')
    await flushPromises()
    expect(view.find('[data-test="stat-date"]').attributes('aria-expanded')).toBe('true')

    document.dispatchEvent(new MouseEvent('mousedown', { bubbles: true }))
    await flushPromises()
    expect(view.find('[data-test="stat-date"]').attributes('aria-expanded')).toBe('false')
  })

  it('禁用时点击不弹出面板', async () => {
    const view = mountPicker({ disabled: true })

    await view.find('[data-test="stat-date"]').trigger('click')
    await flushPromises()

    expect(panel()).toBeNull()
  })
})

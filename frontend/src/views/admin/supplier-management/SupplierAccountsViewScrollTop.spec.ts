import { readFileSync } from 'node:fs'
import { resolve } from 'node:path'
import { describe, expect, it } from 'vitest'

const source = readFileSync(
  resolve(process.cwd(), 'src/views/admin/supplier-management/SupplierAccountsView.vue'),
  'utf8'
)

describe('SupplierAccountsView 回到顶部悬浮按钮', () => {
  it('提供可访问的回到顶部按钮', () => {
    expect(source).toContain('data-test="supplier-account-scroll-top"')
    expect(source).toContain('aria-label="回到顶部"')
    expect(source).toContain('class="sp-scroll-top"')
    // 按钮在页面顶部时不该留在 DOM 里，否则键盘 Tab 会落到看不见的控件上
    expect(source).toContain('v-if="showScrollTop"')
  })

  it('滚动到一定距离后才出现，且用滚动进度驱动圆环', () => {
    expect(source).toContain('SCROLL_TOP_VISIBLE_OFFSET')
    expect(source).toContain('showScrollTop.value = scrolled > SCROLL_TOP_VISIBLE_OFFSET')
    expect(source).toContain(':stroke-dashoffset="scrollTopRingOffset"')
  })

  it('监听 window 滚动并在卸载时解绑', () => {
    // 滚动发生在 window 上：AppLayout 的 <main> 没有自己的 overflow 容器
    expect(source).toContain("window.addEventListener('scroll', handleWindowScroll, { passive: true })")
    expect(source).toContain("window.removeEventListener('scroll', handleWindowScroll)")
  })

  it('尊重系统「减少动态效果」设置', () => {
    expect(source).toContain("window.matchMedia('(prefers-reduced-motion: reduce)').matches")
    expect(source).toContain("behavior: reduceMotion ? 'auto' : 'smooth'")
  })

  it('层级低于弹窗，避免盖住遮罩与对话框', () => {
    // BaseDialog 用 z-50，.sp-overlay 用 z-80，悬浮按钮必须更低
    expect(source).toMatch(/\.sp-scroll-top \{[^}]*z-index: 40;/s)
  })
})

import { readFileSync } from 'node:fs'
import { resolve } from 'node:path'
import { describe, expect, it } from 'vitest'

describe('供应商余额预警导航', () => {
  it('registers both supplier notification routes and sidebar entries', () => {
    const routerSource = readFileSync(resolve(process.cwd(), 'src/router/index.ts'), 'utf8')
    const sidebarSource = readFileSync(resolve(process.cwd(), 'src/components/layout/AppSidebar.vue'), 'utf8')

    expect(routerSource).toContain("path: '/admin/supplier-management/balance-alert'")
    expect(routerSource).toContain("path: '/admin/supplier-management/notifications'")
    expect(routerSource).toContain('SupplierBalanceAlertView.vue')
    expect(routerSource).toContain('SupplierNotificationView.vue')
    expect(sidebarSource).toContain("/admin/supplier-management/balance-alert")
    expect(sidebarSource).toContain("/admin/supplier-management/notifications")
    expect(sidebarSource).toContain('供应商余额预警')
    expect(sidebarSource).toContain('供应商通知配置')

    const cssSource = readFileSync(
      resolve(process.cwd(), 'src/components/admin/supplier-management/supplier-management.css'),
      'utf8'
    )
    expect(cssSource).toContain('.sp-button.primary')
    expect(cssSource).toContain('background: var(--sp-cyan, #3b82f6)')
    expect(cssSource).toContain('color: #fff')
  })
})

import { readFileSync } from 'node:fs'
import { resolve } from 'node:path'
import { describe, expect, it } from 'vitest'

describe('\u4f9b\u5e94\u5546\u8d26\u53f7\u5065\u5eb7\u8d8b\u52bf\u5bfc\u822a', () => {
  it('\u6ce8\u518c\u8d26\u53f7\u5065\u5eb7\u8d8b\u52bf\u8def\u7531\u5e76\u52a0\u5165\u4f9b\u5e94\u5546\u7ba1\u7406\u4fa7\u8fb9\u680f', () => {
    const routerSource = readFileSync(resolve(process.cwd(), 'src/router/index.ts'), 'utf8')
    const sidebarSource = readFileSync(resolve(process.cwd(), 'src/components/layout/AppSidebar.vue'), 'utf8')

    expect(routerSource).toContain("path: '/admin/supplier-management/account-health'")
    expect(routerSource).toContain('SupplierAccountHealthView.vue')
    expect(sidebarSource).toContain("path: '/admin/supplier-management/account-health'")
    expect(sidebarSource).toContain("label: '\u8d26\u53f7\u5065\u5eb7\u8d8b\u52bf'")
  })
})

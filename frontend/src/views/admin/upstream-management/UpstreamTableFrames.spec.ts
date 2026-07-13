import { readFileSync } from 'node:fs'
import { resolve } from 'node:path'
import { describe, expect, it } from 'vitest'

const viewPath = (fileName: string) => resolve(process.cwd(), 'src/views/admin/upstream-management', fileName)

const primaryFrameApply = (source: string, selector: string) => {
  const match = source.match(new RegExp(`\\.${selector}\\s*\\{\\s*\\n\\s*@apply\\s+([^;]+);`))
  return match?.[1] ?? ''
}

describe('upstream management table frames', () => {
  it.each([
    ['UpstreamGroupsView.vue', 'groups-table-primary'],
    ['UpstreamAccountsView.vue', 'accounts-table-primary'],
  ])('%s lets TablePageLayout own the outer table border', (fileName, selector) => {
    const source = readFileSync(viewPath(fileName), 'utf8')
    const applyClasses = primaryFrameApply(source, selector)

    expect(applyClasses).not.toMatch(/\bborder(?:-\w+|\b)/)
    expect(applyClasses).not.toContain('rounded-lg')
  })

  it('registers the upstream dashboard as the upstream management landing page', () => {
    const routerSource = readFileSync(resolve(process.cwd(), 'src/router/index.ts'), 'utf8')
    const sidebarSource = readFileSync(resolve(process.cwd(), 'src/components/layout/AppSidebar.vue'), 'utf8')

    expect(routerSource).toContain("path: '/admin/upstream-management'")
    expect(routerSource).toContain("import('@/views/admin/upstream-management/UpstreamDashboardView.vue')")
    expect(sidebarSource).toContain("{ path: '/admin/upstream-management', label: t('nav.upstreamOverview')")
  })

  it('registers the automation center route and sidebar entry', () => {
    const routerSource = readFileSync(resolve(process.cwd(), 'src/router/index.ts'), 'utf8')
    const sidebarSource = readFileSync(resolve(process.cwd(), 'src/components/layout/AppSidebar.vue'), 'utf8')

    expect(routerSource).toContain("path: '/admin/upstream-management/automations'")
    expect(routerSource).toContain("import('@/views/admin/upstream-management/UpstreamAutomationCenterView.vue')")
    expect(sidebarSource).toContain("{ path: '/admin/upstream-management/automations', label: t('nav.upstreamAutomations')")
  })
})

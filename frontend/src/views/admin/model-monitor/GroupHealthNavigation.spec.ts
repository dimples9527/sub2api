import { readFileSync } from 'node:fs'
import { resolve } from 'node:path'
import { describe, expect, it } from 'vitest'

describe('分组健康趋势导航接入', () => {
  it('注册管理端路由', () => {
    const source = readFileSync(resolve(process.cwd(), 'src/router/index.ts'), 'utf8')
    expect(source).toContain("path: '/admin/model-monitor/group-health'")
    expect(source).toContain("name: 'AdminModelMonitorGroupHealth'")
    expect(source).toContain("import('@/views/admin/model-monitor/GroupHealthTrendView.vue')")
  })

  it('把分组健康趋势放入模型监控子菜单', () => {
    const source = readFileSync(resolve(process.cwd(), 'src/components/layout/AppSidebar.vue'), 'utf8')
    expect(source).toContain("{ path: '/admin/model-monitor/group-health', label: '分组健康趋势'")
  })
})

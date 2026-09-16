import { readFileSync } from 'node:fs'
import { resolve } from 'node:path'
import { describe, expect, it } from 'vitest'

function readSource(path: string): string {
  return readFileSync(resolve(path), 'utf8')
}

/**
 * 平台目录只有一套来源：@/utils/platformOptions。
 * 它在启动时由 platformsAPI.list() 填充，接口返回前用 CORE_PLATFORM_CODES 兜底，
 * 因此能自动跟上后端新增的平台，也能带上自定义平台。
 *
 * 这里曾经断言各页面从 @/constants/platforms 静态目录导入 —— 那套目录已删除。
 * 两条并存时各页面用哪套全凭历史，平台清单还会漂移（静态目录一度比动态目录多两个平台）。
 * 这些断言用来挡住回退。
 */
describe('admin platform filters', () => {
  it('uses the shared platform catalog on the subscriptions page', () => {
    const source = readSource('src/views/admin/SubscriptionsView.vue')
    expect(source).toContain("import { CORE_PLATFORM_OPTIONS } from '@/utils/platformOptions'")
    expect(source).toMatch(/const platformFilterOptions[\s\S]*?\.\.\.CORE_PLATFORM_OPTIONS/)
  })

  it('uses the shared platform catalog on the groups page', () => {
    const source = readSource('src/views/admin/GroupsView.vue')
    // GroupsView 这个文件统一用双引号，断言不能写死引号风格
    expect(source).toMatch(/import\s*\{\s*CORE_PLATFORM_OPTIONS\s*\}\s*from\s*['"]@\/utils\/platformOptions['"]/)
    expect(source).toMatch(/const platformFilterOptions[\s\S]*?\.\.\.CORE_PLATFORM_OPTIONS/)
    // composite 是分组聚合类型、不是具体上游，不能作为复合路由的目标平台
    expect(source).toMatch(/compositeRoutePlatformOptions[\s\S]*?value !== "composite"/)
  })

  it('uses the shared platform catalog wherever concrete platforms are selected', () => {
    for (const path of [
      'src/components/admin/account/AccountTableFilters.vue',
      'src/components/admin/ErrorPassthroughRulesModal.vue',
      'src/views/admin/ops/components/OpsDashboardHeader.vue'
    ]) {
      const source = readSource(path)
      expect(source).toContain("from '@/utils/platformOptions'")
      expect(source).toMatch(/buildPlatformOptions\(|CORE_PLATFORM_OPTIONS/)
    }
  })

  it('no longer imports the removed static catalog', () => {
    for (const path of [
      'src/views/admin/SubscriptionsView.vue',
      'src/views/admin/GroupsView.vue',
      'src/components/admin/account/AccountTableFilters.vue',
      'src/components/admin/ErrorPassthroughRulesModal.vue',
      'src/views/admin/ops/components/OpsDashboardHeader.vue'
    ]) {
      expect(readSource(path)).not.toContain('@/constants/platforms')
    }
  })
})

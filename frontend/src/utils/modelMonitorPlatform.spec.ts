import { readFileSync } from 'node:fs'
import { resolve } from 'node:path'

const monitorPages = [
  resolve(process.cwd(), 'public/model-monitor.html'),
  resolve(process.cwd(), 'public/model-monitor-local.html')
]

describe('模型监控平台展示', () => {
  it.each(monitorPages)('页面 %s 使用分组平台字段展示平台', (pageUrl) => {
    const html = readFileSync(pageUrl, 'utf8')

    expect(html).toMatch(/data-sort="groupPlatform"[^>]*>平台/)
    expect(html).toContain("platform: 'Platform'")
    expect(html).toContain("platform: '平台'")
    expect(html).toContain('function platformBadge')
    expect(html).toContain('platformBadge(row.groupPlatform)')
    expect(html).toContain("platformBadge(row.groupPlatform, 'card-platform')")
    expect(html).toMatch(/data-filter="platform"/)
    expect(html).toContain("allPlatforms: 'All platforms'")
    expect(html).toContain("allPlatforms: '全部平台'")
    expect(html).toContain("platform: 'all'")
    expect(html).toContain("replaceOptions('platform'")
    expect(html).toContain('row.groupPlatform')
    expect(html).toContain("case 'platform': state.platform = value")
    expect(html).toContain("state.platform !== 'all' && row.groupPlatform !== state.platform")
    expect(html).not.toMatch(/<div class="custom-select" data-filter="service"/)
    expect(html).not.toContain('${serviceBadge(row.service)}')
  })
})

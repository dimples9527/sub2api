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

  it.each(monitorPages)('页面 %s 按 40% 阈值尽可能展示绿色', (pageUrl) => {
    const html = readFileSync(pageUrl, 'utf8')

    expect(html).toMatch(
      /function\s+pointTone\(point\)\s*\{[\s\S]*?if\s*\(status\s*===\s*0\s*\|\|\s*availability\s*<=\s*0\)\s*return\s*'red';[\s\S]*?if\s*\(status\s*===\s*1\s*\|\|\s*availability\s*>=\s*40\)\s*return\s*'green';[\s\S]*?return\s*'yellow';[\s\S]*?\n\s*\}/
    )
    expect(html).toContain("if (value >= 40) return 'high';")
    expect(html).toContain("if (value > 0) return 'mid';")
    expect(html).toContain("if (value >= 40) return 'green';")
    expect(html).toContain("if (value > 0) return 'yellow';")
  })
})

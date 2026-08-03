import { readFileSync } from 'node:fs'
import { resolve } from 'node:path'
import { JSDOM } from 'jsdom'

const monitorPages = [
  resolve(process.cwd(), 'public/model-monitor.html'),
  resolve(process.cwd(), 'public/model-monitor-local.html')
]

function mountMonitorPage(pageUrl: string) {
  const html = readFileSync(pageUrl, 'utf8')
  const dom = new JSDOM(html, {
    runScripts: 'dangerously',
    url: 'http://localhost/',
    beforeParse: (window) => {
      const runtimeWindow = window as typeof window & {
        __APP_CONFIG__?: Record<string, string>
        fetch: typeof fetch
      }
      runtimeWindow.__APP_CONFIG__ = {}
      runtimeWindow.matchMedia = () => ({
        matches: false,
        addEventListener: () => undefined,
        addListener: () => undefined,
        removeEventListener: () => undefined,
        removeListener: () => undefined,
        onchange: null,
        media: '',
        dispatchEvent: () => false
      }) as MediaQueryList
      runtimeWindow.fetch = () => new Promise<Response>(() => undefined)
    }
  })
  return dom
}

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

  it.each(monitorPages)('页面 %s 没有监控样本时使用灰色趋势占位', (pageUrl) => {
    const html = readFileSync(pageUrl, 'utf8')

    expect(html).toContain('function emptyTrendBars()')
    expect(html).toMatch(/function\s+normalizeTrend\(item, availability, index\)[\s\S]*?if\s*\(!hasMonitorData\(item\)\)\s*return emptyTrendBars\(\);/)
    expect(html).toContain("if (point.tone === 'empty')")
    expect(html).toContain('.trend-bar.empty { background: #3a465b; }')
  })

  it.each(monitorPages)('页面 %s 区分无数据和真实不可用趋势', (pageUrl) => {
    const dom = mountMonitorPage(pageUrl)
    try {
      const runtimeWindow = dom.window as typeof dom.window & {
        normalizeStatusItem: (item: unknown, index: number, group: unknown) => { trend: Array<{ tone: string }> }
        renderTrendBars: (row: unknown, index: number) => string
      }
      const normalizeStatusItem = runtimeWindow.normalizeStatusItem
      const renderTrendBars = runtimeWindow.renderTrendBars
      const emptyRow = normalizeStatusItem({
        provider: 'empty',
        service: 'CC',
        layers: [{ timeline: [] }]
      }, 0, { rateMultiplier: 1 })
      const redRow = normalizeStatusItem({
        provider: 'red',
        service: 'CC',
        layers: [{ timeline: [{ status: 0, availability: 0, timestamp: 1_784_604_000 }] }]
      }, 0, { rateMultiplier: 1 })

      expect(emptyRow.trend).toHaveLength(18)
      expect(emptyRow.trend.every((point) => point.tone === 'empty')).toBe(true)
      expect(renderTrendBars(emptyRow, 0)).toContain('trend-bar empty')
      expect(renderTrendBars(emptyRow, 0)).not.toContain('trend-bar red')
      expect(redRow.trend.every((point) => point.tone === 'red')).toBe(true)
      expect(renderTrendBars(redRow, 0)).toContain('trend-bar red')
    } finally {
      dom.window.close()
    }
  })
})

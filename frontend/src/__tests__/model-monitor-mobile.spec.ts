import { readFileSync } from 'node:fs'
import { resolve } from 'node:path'
import { describe, expect, it } from 'vitest'

const html = readFileSync(resolve(__dirname, '../../public/model-monitor.html'), 'utf8')

function mediaBlock(maxWidth: number) {
  const marker = `@media (max-width: ${maxWidth}px) {`
  const start = html.indexOf(marker)
  expect(start).toBeGreaterThanOrEqual(0)

  let depth = 0
  for (let i = start; i < html.length; i += 1) {
    if (html[i] === '{') depth += 1
    if (html[i] === '}') {
      depth -= 1
      if (depth === 0) {
        return html.slice(start, i + 1)
      }
    }
  }

  throw new Error(`Could not parse media block: ${marker}`)
}

describe('model monitor mobile layout', () => {
  it('lets the monitor content span the full viewport on phones', () => {
    const phoneStyles = mediaBlock(640)

    expect(phoneStyles).toContain('body { padding: 0 0 20px; }')
    expect(phoneStyles).toContain('.page { max-width: none; width: 100%; }')
    expect(phoneStyles).toContain('.card-grid {')
    expect(phoneStyles).toContain('grid-template-columns: minmax(0, 1fr);')
    expect(phoneStyles).toContain('gap: 0;')
    expect(phoneStyles).toContain('.monitor-card {')
    expect(phoneStyles).toContain('width: 100%;')
  })
})

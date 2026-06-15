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
})

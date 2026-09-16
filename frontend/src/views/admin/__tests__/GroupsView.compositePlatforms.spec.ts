import { readFileSync } from 'node:fs'
import { resolve } from 'node:path'
import { describe, expect, it } from 'vitest'

import { CORE_PLATFORM_OPTIONS } from '@/utils/platformOptions'

describe('GroupsView Composite route options', () => {
  it('offers Kimi, Zhipu GLM, DeepSeek, MiniMax and OpenCode as route targets', () => {
    expect(CORE_PLATFORM_OPTIONS.map((option) => option.value)).toEqual(
      expect.arrayContaining(['kimi', 'zhipu', 'deepseek', 'minimax', 'opencode_go'])
    )
  })

  it('derives the route targets from the shared catalog and excludes composite', () => {
    const source = readFileSync(resolve('src/views/admin/GroupsView.vue'), 'utf8')

    // GroupsView 这个文件统一用双引号，断言不能写死引号风格
    expect(source).toMatch(/import\s*\{\s*CORE_PLATFORM_OPTIONS\s*\}\s*from\s*['"]@\/utils\/platformOptions['"]/)
    // 复合路由的目标平台必须是具体上游，composite 只能被过滤掉
    expect(source).toMatch(
      /compositeRoutePlatformOptions[\s\S]*?CORE_PLATFORM_OPTIONS\.filter\([\s\S]*?value !== "composite"/
    )
  })
})

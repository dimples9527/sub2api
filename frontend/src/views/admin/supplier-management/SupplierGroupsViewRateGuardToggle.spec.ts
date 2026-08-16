import { readFileSync } from 'node:fs'
import { resolve } from 'node:path'
import { describe, expect, it } from 'vitest'

const source = readFileSync(
  resolve(process.cwd(), 'src/views/admin/supplier-management/SupplierGroupsView.vue'),
  'utf8',
)

describe('供应商分组倍率守护开关', () => {
  it('切换倍率守护开关不应触发行点击打开详情弹窗', () => {
    // 行点击会打开详情弹窗：selected 非空时展示分组详情
    expect(source).toContain('@row-click="selected = $event"')
    expect(source).toContain(':show="Boolean(selected)"')
    // 倍率守护开关必须阻止冒泡，避免点击一次同时打开详情弹窗
    const guardToggle = source.match(
      /cell-rate_guard_status[\s\S]*?<Toggle[\s\S]*?\/>/
    )
    expect(guardToggle).not.toBeNull()
    expect(guardToggle![0]).toContain('@click.stop')
    expect(guardToggle![0]).toContain('toggleRateGuardEnabled(group)')
  })
})

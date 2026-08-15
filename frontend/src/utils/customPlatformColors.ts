/**
 * 自定义平台颜色解析。
 *
 * 后端 custom_platforms 表已增加 color 字段（见 225_custom_platforms_color.sql），
 * 加载自定义平台后调用 updateCustomPlatformColors 写入缓存；
 * 分组平台配置等页面按平台区分配色时优先使用自定义颜色，其余回退核心平台品牌色。
 */

import { platformAccentColor } from '@/utils/platformColors'
import type { CustomPlatform } from '@/api/admin/customPlatforms'

const colorCache = new Map<string, string>()

function normalizePlatformCode(platform?: string): string {
  return (platform || '').trim().toLowerCase()
}

export function updateCustomPlatformColors(items: CustomPlatform[]): void {
  colorCache.clear()
  for (const item of items) {
    const code = normalizePlatformCode(item.code)
    if (!code || !item.color) continue
    colorCache.set(code, item.color.trim())
  }
}

/** 仅返回自定义平台配置的颜色；核心平台返回 undefined，交给 platformColors 处理。 */
export function resolveCustomPlatformColor(platform?: string): string | undefined {
  const code = normalizePlatformCode(platform)
  if (!code) return undefined
  return colorCache.get(code)
}

/** 解析平台实际展示颜色：自定义平台优先用其配置色，其余回退核心平台品牌色。 */
export function resolvePlatformColor(platform?: string): string {
  const custom = resolveCustomPlatformColor(platform)
  if (custom) return custom
  return platformAccentColor(platform || '')
}

/**
 * 自定义平台徽章的内联配色（仅自定义平台生效）。
 * Tailwind 无法为运行时颜色动态生成工具类，因此通过 color-mix 派生浅色底、边框与文字色。
 */
export function customPlatformBadgeStyle(platform?: string): Record<string, string> | undefined {
  const color = resolveCustomPlatformColor(platform)
  if (!color) return undefined
  return {
    color,
    backgroundColor: `color-mix(in srgb, ${color} 12%, transparent)`,
    borderColor: `color-mix(in srgb, ${color} 35%, transparent)`,
  }
}

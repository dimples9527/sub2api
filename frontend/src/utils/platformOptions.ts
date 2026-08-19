import type { CustomPlatform } from '@/api/admin/customPlatforms'
import type { PlatformDefinition } from '@/api/admin/platforms'
import { platformsAPI } from '@/api/admin/platforms'
import { reactive } from 'vue'
import { platformLabel, setPlatformLabels, type Platform } from '@/utils/platformColors'

export type PlatformOption = {
  value: string
  label: string
}

// 核心平台选项的唯一前端来源；新增框架平台时只需在这里登记一次。
export const CORE_PLATFORM_CODES: readonly Platform[] = [
  'anthropic',
  'openai',
  'gemini',
  'antigravity',
  'grok',
  'kimi',
  'zhipu',
  'deepseek',
  'composite',
]

const FALLBACK_PLATFORM_OPTIONS: PlatformOption[] = CORE_PLATFORM_CODES.map((value) => ({
  value,
  label: platformLabel(value),
}))

export const CORE_PLATFORM_OPTIONS = reactive<PlatformOption[]>(FALLBACK_PLATFORM_OPTIONS)

const CORE_PLATFORM_MODEL_PLACEHOLDERS: Partial<Record<Platform, string>> = {
  anthropic: 'claude-3-5-haiku-latest',
  openai: 'gpt-4o-mini',
  gemini: 'gemini-2.5-flash',
  antigravity: 'gemini-3-flash',
  grok: 'grok-3-mini',
  kimi: 'moonshot-v1-8k',
  zhipu: 'glm-4.5-air',
  deepseek: 'deepseek-chat',
}

export type HealthGuardPlatformOption = PlatformOption & { placeholder: string }

export const CORE_HEALTH_GUARD_PLATFORM_OPTIONS = reactive<HealthGuardPlatformOption[]>(CORE_PLATFORM_OPTIONS
  .filter((option) => option.value !== 'composite')
  .map((option) => ({
    ...option,
    placeholder: CORE_PLATFORM_MODEL_PLACEHOLDERS[option.value as Platform] || 'model-id',
  })))

export function applyPlatformCatalog(definitions: PlatformDefinition[]): void {
  const valid = definitions.filter((item) => item.code.trim())
  CORE_PLATFORM_OPTIONS.splice(0, CORE_PLATFORM_OPTIONS.length, ...valid.map((item) => ({
    value: item.code,
    label: item.name || platformLabel(item.code),
  })))
  CORE_HEALTH_GUARD_PLATFORM_OPTIONS.splice(
    0,
    CORE_HEALTH_GUARD_PLATFORM_OPTIONS.length,
    ...valid.filter((item) => item.health_guard).map((item) => ({
      value: item.code,
      label: item.name || platformLabel(item.code),
      placeholder: 'model-id',
    })),
  )
  setPlatformLabels(Object.fromEntries(valid.map((item) => [item.code, item.name])))
}

let catalogRequest: Promise<void> | null = null

export function loadPlatformCatalog(): Promise<void> {
  if (!catalogRequest) {
    catalogRequest = platformsAPI.list()
      .then(applyPlatformCatalog)
      .catch(() => undefined)
      .finally(() => { catalogRequest = null })
  }
  return catalogRequest
}

export function buildPlatformOptions(customPlatforms: CustomPlatform[] = []): PlatformOption[] {
  const options = CORE_PLATFORM_OPTIONS.map((option) => ({ ...option }))
  const seen = new Set(options.map((option) => String(option.value)))
  for (const platform of customPlatforms) {
    if (!platform.enabled || seen.has(platform.code)) continue
    seen.add(platform.code)
    options.push({ value: platform.code, label: platform.name })
  }
  return options
}

import { customPlatformsAPI, type CustomPlatform } from '@/api/admin/customPlatforms'
import { platformLabel as corePlatformLabel } from '@/utils/platformColors'

const labelCache = new Map<string, string>()
let loadingPromise: Promise<void> | null = null

function normalizePlatformCode(platform?: string): string {
  return (platform || '').trim().toLowerCase()
}

function updateLabelCache(items: CustomPlatform[]) {
  labelCache.clear()
  for (const item of items) {
    const code = normalizePlatformCode(item.code)
    if (!code) continue
    labelCache.set(code, item.name.trim() || item.code)
  }
}

export function resolvePlatformDisplayLabel(platform?: string): string {
  const code = normalizePlatformCode(platform)
  if (!code) return 'API'
  return labelCache.get(code) || corePlatformLabel(code)
}

export async function refreshCustomPlatformLabels(): Promise<void> {
  const items = await customPlatformsAPI.list(false)
  updateLabelCache(items)
}

export async function ensureCustomPlatformLabels(): Promise<void> {
  if (!loadingPromise) {
    loadingPromise = refreshCustomPlatformLabels().finally(() => {
      loadingPromise = null
    })
  }
  return loadingPromise
}

export function setCustomPlatformLabels(items: CustomPlatform[]): void {
  updateLabelCache(items)
}

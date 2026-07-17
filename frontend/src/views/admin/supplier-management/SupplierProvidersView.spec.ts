import { readFileSync } from 'node:fs'
import { dirname, resolve } from 'node:path'
import { fileURLToPath } from 'node:url'
import { describe, expect, it } from 'vitest'

const supplierProvidersSource = readFileSync(
  resolve(dirname(fileURLToPath(import.meta.url)), 'SupplierProvidersView.vue'),
  'utf-8'
)

describe('SupplierProvidersView payload normalization', () => {
  it('submits Sub2API credentials as email only and clears stale username', () => {
    expect(supplierProvidersSource).toContain('const normalizedProviderType = payload.provider_type.trim()')
    expect(supplierProvidersSource).toContain("email: normalizedProviderType === 'sub2api' ? payload.email?.trim() || '' : ''")
    expect(supplierProvidersSource).toContain("username: normalizedProviderType === 'sub2api' ? '' : payload.username?.trim() || ''")
  })

  it('provides per-scope test buttons and a frontend diagnostics dialog', () => {
    expect(supplierProvidersSource).toContain('testProviderEndpoint')
    expect(supplierProvidersSource).toContain('测试 API Key')
    expect(supplierProvidersSource).toContain('测试分组')
    expect(supplierProvidersSource).toContain('测试余额')
    expect(supplierProvidersSource).toContain('测试成本')
    expect(supplierProvidersSource).toContain('接口测试结果')
    expect(supplierProvidersSource).toContain('testResultVisible')
  })

  it('uses the global app toast store for provider operation feedback', () => {
    expect(supplierProvidersSource).toContain("import { useAppStore } from '@/stores/app'")
    expect(supplierProvidersSource).toContain('const appStore = useAppStore()')
    expect(supplierProvidersSource).toContain('appStore.showError(')
    expect(supplierProvidersSource).toContain('appStore.showSuccess(')
    expect(supplierProvidersSource).not.toContain('class="sp-toast"')
  })
})

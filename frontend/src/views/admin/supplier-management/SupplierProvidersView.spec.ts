import { readFileSync } from 'node:fs'
import { dirname, resolve } from 'node:path'
import { fileURLToPath } from 'node:url'
import { describe, expect, it } from 'vitest'

const supplierProvidersSource = readFileSync(
  resolve(dirname(fileURLToPath(import.meta.url)), 'SupplierProvidersView.vue'),
  'utf-8'
)

describe('SupplierProvidersView payload normalization', () => {
  it('separates the page heading from one unified provider filter card', () => {
    const pageHeadSource = supplierProvidersSource.match(/<header class="sp-page-head">[\s\S]*?<\/header>/)?.[0]

    expect(pageHeadSource).not.toContain('sp-controls')
    expect(pageHeadSource).not.toContain('v-model="search"')
    expect(supplierProvidersSource).toContain('class="sp-provider-filter-card"')
    expect(supplierProvidersSource).toContain('class="sp-filter-card-head"')
    expect(supplierProvidersSource).toContain('class="sp-provider-filter-body"')
    expect(supplierProvidersSource).toContain('class="sp-provider-filter-fields"')
    expect(supplierProvidersSource).toContain('class="sp-provider-filter-actions"')
    expect(supplierProvidersSource).toContain('筛选供应商')
    expect(supplierProvidersSource).toContain('@media (max-width: 900px)')
    expect(supplierProvidersSource).toContain('@media (max-width: 520px)')
  })

  it('provides a direct create-provider-type action and dedicated dialog', () => {
    expect(supplierProvidersSource).toContain('@click="openCreateProviderType"')
    expect(supplierProvidersSource).toContain('新增供应商类型')
    expect(supplierProvidersSource).toContain(':show="createTypeVisible"')
    expect(supplierProvidersSource).toContain('class="sp-type-create-dialog"')
    expect(supplierProvidersSource).toContain('@submit.prevent="submitNewProviderType"')
    expect(supplierProvidersSource).toContain('const createTypeVisible = ref(false)')
    expect(supplierProvidersSource).toContain('function openCreateProviderType()')
    expect(supplierProvidersSource).toContain('function closeCreateProviderType()')
    expect(supplierProvidersSource).toContain('async function submitNewProviderType()')
  })

  it('uses structured page-level styling for all supplier dialogs', () => {
    expect(supplierProvidersSource).toContain('class="sp-provider-dialog"')
    expect(supplierProvidersSource).toContain('class="sp-dialog-summary"')
    expect(supplierProvidersSource).toContain('class="sp-type-manager-dialog"')
    expect(supplierProvidersSource).toContain('class="sp-test-dialog"')
    expect(supplierProvidersSource).toContain('class="sp-dialog-section-head"')
    expect(supplierProvidersSource).toContain('.sp-dialog-section {')
    expect(supplierProvidersSource).toContain('.sp-type-manager-dialog {')
    expect(supplierProvidersSource).toContain('.sp-test-dialog {')
    expect(supplierProvidersSource).toContain('@media (max-width: 760px)')
    expect(supplierProvidersSource).toContain(':global(.dark .modal-content:has(.sp-provider-dialog))')
  })
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

  it('uses existing framework components instead of native table, modal, and form controls', () => {
    expect(supplierProvidersSource).toContain("import BaseDialog from '@/components/common/BaseDialog.vue'")
    expect(supplierProvidersSource).toContain("import DataTable from '@/components/common/DataTable.vue'")
    expect(supplierProvidersSource).toContain("import Input from '@/components/common/Input.vue'")
    expect(supplierProvidersSource).toContain("import Select, { type SelectOption } from '@/components/common/Select.vue'")
    expect(supplierProvidersSource).toContain("import Toggle from '@/components/common/Toggle.vue'")
    expect(supplierProvidersSource).toContain('<BaseDialog')
    expect(supplierProvidersSource).toContain('<DataTable')
    expect(supplierProvidersSource).toContain('<Input')
    expect(supplierProvidersSource).toContain('<Select')
    expect(supplierProvidersSource).toContain('<Toggle')
    expect(supplierProvidersSource).not.toContain('SupplierModal')
    expect(supplierProvidersSource).not.toContain('<table')
    expect(supplierProvidersSource).not.toContain('<select')
    expect(supplierProvidersSource).not.toContain('<input')
    expect(supplierProvidersSource).not.toContain('type="checkbox"')
  })
})

import { readFileSync } from 'node:fs'
import { resolve } from 'node:path'
import { describe, expect, it } from 'vitest'

const source = readFileSync(
  resolve(process.cwd(), 'src/views/admin/supplier-management/SupplierNotificationView.vue'),
  'utf8'
)

describe('SupplierNotificationView', () => {
  it('uses shared controls for channel, subscription, and delivery management', () => {
    expect(source).toContain('<SupplierModuleLayout>')
    expect(source).toContain('<DataTable')
    expect(source).toContain('<Select')
    expect(source).toContain('<Toggle')
    expect(source).toContain('<BaseDialog')
    expect(source).toContain('createSupplierNotificationChannel')
    expect(source).toContain('createSupplierNotificationSubscription')
    expect(source).toContain('listSupplierNotificationDeliveries')
  })

  it('does not render or bind returned channel secrets and passwords', () => {
    expect(source).toContain('已配置')
    expect(source).toContain('secretConfigured')
    expect(source).toContain('password:')
    expect(source).not.toContain('channel.secret')
    expect(source).not.toContain('channel.password')
    expect(source).not.toContain('channel.webhook_url')
  })
})

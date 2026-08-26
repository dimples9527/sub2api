import { describe, expect, it, vi } from 'vitest'
import { createKeyedRequestLoader } from './useKeyedRequestLoader'

describe('createKeyedRequestLoader', () => {
  it('同一个账号和趋势范围的并发请求只执行一次', async () => {
    let resolveRequest: ((value: string) => void) | undefined
    const request = vi.fn(() => new Promise<string>(resolve => {
      resolveRequest = resolve
    }))
    const load = createKeyedRequestLoader(request, (accountId: number, range: string) => `${accountId}:${range}`)

    const first = load(42, '24h')
    const second = load(42, '24h')

    expect(request).toHaveBeenCalledTimes(1)
    expect(first).toBe(second)

    resolveRequest?.('ok')
    await expect(first).resolves.toBe('ok')
  })
})

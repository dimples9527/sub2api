import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'

import { ADMIN_UI_REQUEST_HEADER } from '../adminUIRequest'
import { streamSupplierProviderSync } from './supplierProviderData'

function streamResponse(chunks: string[]) {
  const encoder = new TextEncoder()
  const body = new ReadableStream<Uint8Array>({
    start(controller) {
      for (const chunk of chunks) controller.enqueue(encoder.encode(chunk))
      controller.close()
    },
  })

  return {
    ok: true,
    status: 200,
    statusText: 'OK',
    body,
  }
}

describe('supplierProviderData SSE API', () => {
  const originalFetch = globalThis.fetch

  beforeEach(() => {
    globalThis.fetch = vi.fn()
    localStorage.clear()
  })

  afterEach(() => {
    globalThis.fetch = originalFetch
  })

  it('uses the admin auth headers and parses events split across response chunks', async () => {
    localStorage.setItem('auth_token', 'admin-token')
    vi.mocked(globalThis.fetch).mockResolvedValue(streamResponse([
      'data: {"stage":"prepare","message":"\u51c6\u5907\u540c\u6b65",',
      '"time":"2026-08-05T00:00:00Z"}\n\n',
      'data: {"stage":"done","message":"\u540c\u6b65\u5b8c\u6210","ok":true,"time":"2026-08-05T00:00:01Z"}\n\n',
    ]) as Response)

    const events: Array<{ stage: string; message: string; ok?: boolean }> = []
    await streamSupplierProviderSync(8, 'all', {
      onEvent: event => events.push(event),
    })

    expect(globalThis.fetch).toHaveBeenCalledWith(
      '/api/v1/admin/supplier-management/providers/8/sync/all/stream',
      expect.objectContaining({
        method: 'POST',
        headers: {
          Accept: 'text/event-stream',
          Authorization: 'Bearer admin-token',
          [ADMIN_UI_REQUEST_HEADER]: '1',
        },
      }),
    )
    expect(events).toEqual([
      { stage: 'prepare', message: '\u51c6\u5907\u540c\u6b65', time: '2026-08-05T00:00:00Z' },
      { stage: 'done', message: '\u540c\u6b65\u5b8c\u6210', ok: true, time: '2026-08-05T00:00:01Z' },
    ])
  })

  it('throws a readable error when the stream request fails', async () => {
    vi.mocked(globalThis.fetch).mockResolvedValue({
      ok: false,
      status: 502,
      statusText: 'Bad Gateway',
      text: vi.fn().mockResolvedValue('\u4e0a\u6e38\u540c\u6b65\u670d\u52a1\u4e0d\u53ef\u7528'),
    } as unknown as Response)

    await expect(streamSupplierProviderSync(8, 'accounts', { onEvent: vi.fn() }))
      .rejects.toThrow('\u540c\u6b65\u8fdb\u5ea6\u8bf7\u6c42\u5931\u8d25\uff08502\uff09\uff1a\u4e0a\u6e38\u540c\u6b65\u670d\u52a1\u4e0d\u53ef\u7528')
  })
})
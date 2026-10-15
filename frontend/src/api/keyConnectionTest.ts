/**
 * API 密钥连通性测试。
 *
 * 这里刻意不走面板接口（apiClient），而是像「图片生成」页那样由浏览器直接请求网关：
 * 只有真正走一遍网关，才能验证这把密钥当下的可用性，并量出用户视角的首字耗时
 * —— 它包含浏览器到网关的网络与网关内部的排队时间，这正是截图里那个数字的含义。
 *
 * 请求头里的密钥取自密钥列表接口，页面本来就能复制它，因此没有新增暴露面。
 */

/**
 * 测试发送的固定短消息。与后端账号测试保持同一套极短提示词，
 * 目的是让回复尽快出首字，而不是考察模型能力。
 */
const TEST_PROMPT = 'hi'

/** 只取少量 token：测试关心「能不能出字」和首字延迟，不关心回复长度。 */
const TEST_MAX_TOKENS = 64

/** 网关错误体形如 { error: { message } }；少数路径用 { message } 或 { detail }。 */
export function extractGatewayErrorMessage(payload: unknown, fallback: string): string {
  if (typeof payload !== 'object' || payload === null) return fallback
  const record = payload as Record<string, unknown>

  const error = record.error
  if (typeof error === 'string' && error) return error
  if (typeof error === 'object' && error !== null) {
    const message = (error as Record<string, unknown>).message
    if (typeof message === 'string' && message) return message
  }

  for (const key of ['message', 'detail'] as const) {
    const value = record[key]
    if (typeof value === 'string' && value) return value
  }
  return fallback
}

/**
 * 找出缓冲区里下一个 SSE 事件边界（空行）。
 *
 * 服务端可能用 LF 也可能用 CRLF 分隔事件，只找 `\n\n` 会在 CRLF 流上永远匹配不到，
 * 表现是「连接成功但一个字都出不来」，所以两种都要认。
 */
function findEventBoundary(buffer: string): { index: number; length: number } | null {
  const lf = buffer.indexOf('\n\n')
  const crlf = buffer.indexOf('\r\n\r\n')
  if (crlf !== -1 && (lf === -1 || crlf < lf)) return { index: crlf, length: 4 }
  if (lf !== -1) return { index: lf, length: 2 }
  return null
}

/**
 * 从一段 SSE 事件里取出增量文本，同时识别网关在 200 流里回吐的错误事件。
 * `error` 为 null 表示这一段里没有错误；非 null 时 `delta` 一定为空。
 */
function parseChatCompletionEvent(rawEvent: string): { delta: string; error: string | null } {
  for (const rawLine of rawEvent.split('\n')) {
    const line = rawLine.replace(/\r$/, '').trim()
    if (!line.startsWith('data:')) continue

    const data = line.slice(5).trim()
    if (!data || data === '[DONE]') continue

    let chunk: unknown
    try {
      chunk = JSON.parse(data)
    } catch {
      // 心跳或非 JSON 事件，跳过即可，不影响判定。
      continue
    }
    if (typeof chunk !== 'object' || chunk === null) continue

    const record = chunk as Record<string, unknown>
    if (record.error) {
      // 这里必须用 null 之外的值表示「有错误」：网关可能只给一个空对象，
      // 若以空串当作「无错误」会把上游报错静默吞掉。
      return { delta: '', error: extractGatewayErrorMessage(chunk, '') }
    }

    const choices = record.choices
    if (!Array.isArray(choices) || choices.length === 0) continue

    const delta = (choices[0] as Record<string, unknown> | null)?.delta
    const content =
      typeof delta === 'object' && delta !== null
        ? (delta as Record<string, unknown>).content
        : undefined
    if (typeof content === 'string' && content) {
      return { delta: content, error: null }
    }
  }
  return { delta: '', error: null }
}

/** 读取该密钥当前可用的模型列表（按密钥所属分组过滤，含分组模型白名单）。 */
export async function fetchKeyTestModels(
  apiKey: string,
  options?: { signal?: AbortSignal },
): Promise<string[]> {
  const response = await fetch('/v1/models', {
    headers: { Authorization: `Bearer ${apiKey}` },
    signal: options?.signal,
  })
  const payload: unknown = await response.json().catch(() => null)

  if (!response.ok) {
    throw new Error(extractGatewayErrorMessage(payload, `HTTP ${response.status}`))
  }

  const data = (payload as { data?: unknown } | null)?.data
  if (!Array.isArray(data)) return []

  const models: string[] = []
  const seen = new Set<string>()
  for (const item of data) {
    const id =
      typeof item === 'object' && item !== null ? (item as { id?: unknown }).id : undefined
    if (typeof id !== 'string') continue
    const model = id.trim()
    if (!model || seen.has(model)) continue
    seen.add(model)
    models.push(model)
  }
  return models
}

export interface KeyConnectionTestHandlers {
  /** 首个文字片段到达时回调，单位毫秒（相对请求发出时刻）。 */
  onFirstToken?: (elapsedMs: number) => void
  /** 每收到一段文字回调一次，参数是累计后的全文。 */
  onDelta?: (text: string) => void
}

export interface KeyConnectionTestResult {
  /** 累计回复全文；为空说明网关没吐出任何文字。 */
  text: string
  /** 首字耗时（毫秒）；整段流里没有任何文字片段时为 null。 */
  firstTokenMs: number | null
  durationMs: number
}

/**
 * 用给定密钥对指定模型发起一次流式测试请求。
 *
 * 统一走 /v1/chat/completions：后端该端点会按分组平台自动分流，
 * OpenAI 系平台原生处理，Anthropic 系有专门的协议转换实现
 * （见 backend/internal/handler/gateway_handler_chat_completions.go），
 * 因此一把密钥无论属于哪类分组都能用同一份前端代码测。
 */
export async function runKeyConnectionTest(
  apiKey: string,
  model: string,
  handlers: KeyConnectionTestHandlers = {},
  options?: { signal?: AbortSignal },
): Promise<KeyConnectionTestResult> {
  const startedAt = performance.now()
  const response = await fetch('/v1/chat/completions', {
    method: 'POST',
    headers: {
      Authorization: `Bearer ${apiKey}`,
      'Content-Type': 'application/json',
      Accept: 'text/event-stream',
    },
    body: JSON.stringify({
      model,
      messages: [{ role: 'user', content: TEST_PROMPT }],
      max_tokens: TEST_MAX_TOKENS,
      stream: true,
    }),
    signal: options?.signal,
  })

  if (!response.ok) {
    const payload: unknown = await response.json().catch(() => null)
    throw new Error(extractGatewayErrorMessage(payload, `HTTP ${response.status}`))
  }

  const durationMs = () => performance.now() - startedAt
  // 没有流可读等同于「一个文字片段都没收到」，交给调用方按空响应处理，
  // 免得这里抛出未本地化的英文文案。
  if (!response.body) {
    return { text: '', firstTokenMs: null, durationMs: durationMs() }
  }

  const reader = response.body.getReader()
  const decoder = new TextDecoder()
  let buffer = ''
  let text = ''
  let firstTokenMs: number | null = null

  try {
    for (;;) {
      const { done, value } = await reader.read()
      if (done) break

      buffer += decoder.decode(value, { stream: true })

      let boundary = findEventBoundary(buffer)
      while (boundary) {
        const rawEvent = buffer.slice(0, boundary.index)
        buffer = buffer.slice(boundary.index + boundary.length)

        const { delta, error } = parseChatCompletionEvent(rawEvent)
        if (error !== null) throw new Error(error)
        if (delta) {
          if (firstTokenMs === null) {
            firstTokenMs = performance.now() - startedAt
            handlers.onFirstToken?.(firstTokenMs)
          }
          text += delta
          handlers.onDelta?.(text)
        }

        boundary = findEventBoundary(buffer)
      }
    }
  } finally {
    reader.cancel().catch(() => undefined)
  }

  return { text, firstTokenMs, durationMs: durationMs() }
}

/** 测试发送的固定短消息，供弹窗回显「发了什么」时使用。 */
export const KEY_TEST_PROMPT = TEST_PROMPT

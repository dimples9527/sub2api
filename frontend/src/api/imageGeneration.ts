import type { ApiKey } from '@/types'

export type ImageQuality = 'auto' | 'low' | 'medium' | 'high'
export type ImageSize = '1024x1024' | '1024x1536' | '1536x1024'

export interface ImageGenerationPayloadInput {
  model: string
  prompt: string
  size: ImageSize
  count: number
  quality: ImageQuality
}

export interface ImageGenerationPayload {
  model: string
  prompt: string
  size: ImageSize
  n: number
  quality: ImageQuality
  response_format: 'b64_json'
}

export interface GeneratedImage {
  id: string
  src: string
  revisedPrompt: string
}

interface OpenAIImageResponseItem {
  b64_json?: string
  url?: string
  revised_prompt?: string
}

interface OpenAIImageResponse {
  data?: OpenAIImageResponseItem[]
}

export function buildImageGenerationPayload(input: ImageGenerationPayloadInput): ImageGenerationPayload {
  return {
    model: input.model.trim(),
    prompt: input.prompt.trim(),
    size: input.size,
    n: input.count,
    quality: input.quality,
    response_format: 'b64_json',
  }
}

export function parseImageGenerationResponse(response: OpenAIImageResponse): GeneratedImage[] {
  return (response.data ?? [])
    .map((item, index): GeneratedImage | null => {
      const rawSource = item.b64_json
        ? `data:image/png;base64,${item.b64_json}`
        : item.url?.trim()

      if (!rawSource) return null

      return {
        id: `${Date.now()}-${index}-${Math.random().toString(36).slice(2, 8)}`,
        src: rawSource,
        revisedPrompt: item.revised_prompt?.trim() ?? '',
      }
    })
    .filter((item): item is GeneratedImage => item !== null)
}

export function getImageCapableOpenAIKeys(keys: ApiKey[]): ApiKey[] {
  return keys.filter((key) => {
    if (key.status !== 'active') return false
    if (!key.group) return false
    if (key.group.platform !== 'openai') return false
    return key.group.allow_image_generation === true
  })
}

export async function generateImages(
  apiKey: string,
  payload: ImageGenerationPayload,
  options?: { signal?: AbortSignal },
): Promise<GeneratedImage[]> {
  const response = await fetch('/v1/images/generations', {
    method: 'POST',
    headers: {
      Authorization: `Bearer ${apiKey}`,
      'Content-Type': 'application/json',
    },
    body: JSON.stringify(payload),
    signal: options?.signal,
  })

  const data = await response.json().catch(() => ({}))
  if (!response.ok) {
    const message =
      data?.error?.message ||
      data?.message ||
      data?.detail ||
      `Image generation failed with status ${response.status}`
    throw new Error(message)
  }

  return parseImageGenerationResponse(data)
}

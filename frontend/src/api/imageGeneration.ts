import type { ApiKey } from '@/types'

export type ImageQuality = 'auto' | 'low' | 'medium' | 'high'
export type ImageSize = '1920x1088' | '2560x1440' | '3840x2160'

export interface ImageGenerationPayloadInput {
  model: string
  prompt: string
  size: ImageSize
  count: number
  quality: ImageQuality
}

export interface ImageEditPayloadInput extends ImageGenerationPayloadInput {
  image: File
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

interface OpenAIModelsResponseItem {
  id?: string
}

interface OpenAIModelsResponse {
  data?: OpenAIModelsResponseItem[]
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

export function buildImageEditFormData(input: ImageEditPayloadInput): FormData {
  const formData = new FormData()
  formData.append('model', input.model.trim())
  formData.append('prompt', input.prompt.trim())
  formData.append('size', input.size)
  formData.append('n', String(input.count))
  formData.append('quality', input.quality)
  formData.append('response_format', 'b64_json')
  formData.append('image', input.image)
  return formData
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

export async function fetchImageModelOptions(
  apiKey: string,
  options?: { signal?: AbortSignal },
): Promise<string[]> {
  const response = await fetch('/v1/models', {
    headers: {
      Authorization: `Bearer ${apiKey}`,
    },
    signal: options?.signal,
  })

  const data = await response.json().catch(() => ({}))
  if (!response.ok) {
    const message =
      data?.error?.message ||
      data?.message ||
      data?.detail ||
      `Model list failed with status ${response.status}`
    throw new Error(message)
  }

  return parseImageModelOptions(data)
}

export function parseImageModelOptions(response: OpenAIModelsResponse): string[] {
  const seen = new Set<string>()
  const models: string[] = []

  for (const item of response.data ?? []) {
    const model = item.id?.trim()
    if (!model || !model.toLowerCase().startsWith('gpt-image-') || seen.has(model)) {
      continue
    }
    seen.add(model)
    models.push(model)
  }

  return models
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

export async function generateImageEdit(
  apiKey: string,
  payload: ImageEditPayloadInput,
  options?: { signal?: AbortSignal },
): Promise<GeneratedImage[]> {
  const response = await fetch('/v1/images/edits', {
    method: 'POST',
    headers: {
      Authorization: `Bearer ${apiKey}`,
    },
    body: buildImageEditFormData(payload),
    signal: options?.signal,
  })

  const data = await response.json().catch(() => ({}))
  if (!response.ok) {
    const message =
      data?.error?.message ||
      data?.message ||
      data?.detail ||
      `Image edit failed with status ${response.status}`
    throw new Error(message)
  }

  return parseImageGenerationResponse(data)
}

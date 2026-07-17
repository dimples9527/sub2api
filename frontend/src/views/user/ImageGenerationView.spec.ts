import { flushPromises, mount } from '@vue/test-utils'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import ImageGenerationView from './ImageGenerationView.vue'

const mocks = vi.hoisted(() => ({
  showError: vi.fn(),
  showSuccess: vi.fn(),
  generateImages: vi.fn(),
  generateImageEdit: vi.fn(),
  fetchImageModelOptions: vi.fn(),
}))

const keyList = vi.hoisted(() => ({
  items: [
    {
      id: 7,
      key: 'sk-test',
      name: 'OpenAI key',
      status: 'active',
      group: {
        name: 'OpenAI',
        platform: 'openai',
        allow_image_generation: true,
      },
    },
  ],
}))

vi.mock('@/stores/app', () => ({
  useAppStore: () => ({
    showError: mocks.showError,
    showSuccess: mocks.showSuccess,
  }),
}))

vi.mock('@/composables/useClipboard', () => ({
  useClipboard: () => ({
    copyToClipboard: vi.fn(),
  }),
}))

vi.mock('@/api/keys', () => ({
  default: {
    list: vi.fn().mockImplementation(async () => ({ items: keyList.items })),
  },
}))

vi.mock('@/api/imageGeneration', async () => {
  const actual = await vi.importActual<typeof import('@/api/imageGeneration')>('@/api/imageGeneration')
  return {
    ...actual,
    generateImages: mocks.generateImages,
    generateImageEdit: mocks.generateImageEdit,
    fetchImageModelOptions: mocks.fetchImageModelOptions,
  }
})

vi.mock('vue-i18n', async () => {
  const actual = await vi.importActual<typeof import('vue-i18n')>('vue-i18n')

  return {
    ...actual,
    useI18n: () => ({
      t: (key: string, params?: Record<string, string>) => {
        const messages: Record<string, string> = {
          'imageGeneration.navCrumb': 'Image generation',
          'imageGeneration.title': 'Image generation',
          'imageGeneration.description': 'Generate images from prompts',
          'imageGeneration.serviceOnline': 'Online',
          'imageGeneration.generating': 'Generating',
          'imageGeneration.generatingHint': 'Please wait',
          'imageGeneration.elapsed': `Elapsed ${params?.seconds ?? '0.0'}s`,
          'imageGeneration.stage.preparing': 'Preparing',
          'imageGeneration.stage.queued': 'Queued',
          'imageGeneration.stage.generating': 'Rendering',
          'imageGeneration.stage.finishing': 'Finishing',
          'imageGeneration.reusePrompt': 'Reuse prompt',
          'imageGeneration.continueEdit': 'Continue edit',
          'imageGeneration.continueEditPrefix': 'Continue editing',
          'imageGeneration.downloadAll': 'Download all',
          'imageGeneration.copyImageUrl': 'Copy image URL',
          'imageGeneration.download': 'Download',
          'imageGeneration.viewImage': 'Preview',
          'imageGeneration.closePreview': 'Close preview',
          'imageGeneration.removeReferenceImage': 'Remove reference image',
          'imageGeneration.editPromptPlaceholder': 'Describe how to edit this image',
          'imageGeneration.previewDetails.model': 'Model',
          'imageGeneration.previewDetails.size': 'Size',
          'imageGeneration.previewDetails.quality': 'Quality',
          'imageGeneration.previewDetails.elapsed': 'Elapsed',
          'imageGeneration.previewDetails.createdAt': 'Created',
          'imageGeneration.retry': 'Retry',
          'imageGeneration.selectKey': 'Select key',
          'imageGeneration.model': 'Model',
          'imageGeneration.size': 'Size',
          'imageGeneration.count': 'Count',
          'imageGeneration.quality': 'Quality',
          'imageGeneration.generate': 'Generate',
          'imageGeneration.submitHint': 'Enter to send',
          'imageGeneration.generateSuccess': 'Generated',
          'imageGeneration.emptyResponse': 'No images returned',
          'imageGeneration.loadKeysFailed': 'Failed to load keys',
          'imageGeneration.loadingKeys': 'Loading keys',
          'imageGeneration.noKeys': 'No keys',
          'imageGeneration.emptyTitle': 'Start generating images',
          'imageGeneration.emptyDescription': 'Enter a prompt to generate images.',
          'imageGeneration.promptPlaceholder': 'Describe the image',
          'imageGeneration.samples.glass': 'Glass product poster',
          'imageGeneration.samples.portrait': 'Natural light portrait',
          'imageGeneration.samples.sunrise': 'Mountain sunrise',
          'imageGeneration.samples.logo': 'Brand logo',
          'imageGeneration.qualities.auto': 'Auto',
          'common.refresh': 'Refresh',
          'common.cancel': 'Cancel',
        }
        return messages[key] ?? key
      },
    }),
  }
})

describe('ImageGenerationView', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    localStorage.clear()
    mocks.generateImages.mockResolvedValue([
      {
        id: 'image-1',
        src: 'data:image/png;base64,abc123',
        revisedPrompt: '',
      },
    ])
    mocks.generateImageEdit.mockResolvedValue([
      {
        id: 'image-edit-1',
        src: 'data:image/png;base64,edited123',
        revisedPrompt: '',
      },
    ])
    mocks.fetchImageModelOptions.mockResolvedValue(['gpt-image-3', 'gpt-image-2'])
  })

  afterEach(() => {
    vi.useRealTimers()
    document.body.innerHTML = ''
    localStorage.clear()
  })

  it('uses SUNSHINE API as the image generation brand', async () => {
    const wrapper = mountView()
    await flushPromises()

    expect(wrapper.text()).toContain('SUNSHINE API')
    expect(wrapper.text()).not.toContain('MIAOAPI')
  })

  it('opens a large preview when a generated image is clicked', async () => {
    const wrapper = mountView()
    await flushPromises()

    await wrapper.find('textarea').setValue('draw a mountain sunrise')
    await wrapper.find('form').trigger('submit.prevent')
    await flushPromises()

    await wrapper.find('[data-testid="generated-image-button"]').trigger('click')
    await flushPromises()

    const dialog = document.body.querySelector('[data-testid="image-preview-dialog"]')
    expect(dialog).not.toBeNull()
    expect(dialog?.querySelector('img')?.getAttribute('src')).toBe('data:image/png;base64,abc123')
  })

  it('renders generated results as a chat-style exchange', async () => {
    const wrapper = mountView()
    await flushPromises()

    await wrapper.find('textarea').setValue('draw a mountain sunrise')
    await wrapper.find('form').trigger('submit.prevent')
    await flushPromises()

    expect(wrapper.find('.chat-turn.user-turn').text()).toContain('draw a mountain sunrise')
    expect(wrapper.find('.chat-turn.assistant-turn').exists()).toBe(true)
    expect(wrapper.find('.assistant-bubble [data-testid="generated-image-button"]').exists()).toBe(true)
  })

  it('loads image models for the selected image-capable key', async () => {
    const wrapper = mountView()
    await flushPromises()

    expect(mocks.fetchImageModelOptions).toHaveBeenCalledWith('sk-test')
    await flushPromises()
    const selects = wrapper.findAll('select')
    const imageModelSelect = selects[1].element as HTMLSelectElement
    expect(Array.from(imageModelSelect.options).map((option) => option.value)).toEqual(['gpt-image-3', 'gpt-image-2'])

    await wrapper.find('textarea').setValue('draw a mountain sunrise')
    await wrapper.find('form').trigger('submit.prevent')
    await flushPromises()

    expect(mocks.generateImages.mock.calls[0][1]).toMatchObject({
      model: 'gpt-image-2',
    })
  })

  it('falls back to default image models when the key is not image capable', async () => {
    keyList.items = [
      {
        id: 8,
        key: 'sk-no-image',
        name: 'Text key',
        status: 'active',
        group: {
          name: 'OpenAI',
          platform: 'openai',
          allow_image_generation: false,
        },
      },
    ]

    const wrapper = mountView()
    await flushPromises()

    expect(mocks.fetchImageModelOptions).not.toHaveBeenCalled()
    const selects = wrapper.findAll('select')
    const imageModelSelect = selects[1].element as HTMLSelectElement
    expect(Array.from(imageModelSelect.options).map((option) => option.value)).toEqual(['gpt-image-2', 'gpt-image-1'])

    keyList.items = [
      {
        id: 7,
        key: 'sk-test',
        name: 'OpenAI key',
        status: 'active',
        group: {
          name: 'OpenAI',
          platform: 'openai',
          allow_image_generation: true,
        },
      },
    ]
  })

  it('shows elapsed time while generation is still running', async () => {
    vi.useFakeTimers()
    mocks.generateImages.mockReturnValue(new Promise(() => {}))

    const wrapper = mountView()
    await flushPromises()

    await wrapper.find('textarea').setValue('draw a mountain sunrise')
    await wrapper.find('form').trigger('submit.prevent')
    await flushPromises()

    vi.advanceTimersByTime(5200)
    await wrapper.vm.$nextTick()

    expect(wrapper.text()).toContain('Elapsed 5.2s')
  })

  it('keeps the final elapsed time on the generated result', async () => {
    vi.useFakeTimers()
    const deferred = createDeferred()
    mocks.generateImages.mockReturnValue(deferred.promise)

    const wrapper = mountView()
    await flushPromises()

    await wrapper.find('textarea').setValue('draw a mountain sunrise')
    await wrapper.find('form').trigger('submit.prevent')
    await flushPromises()

    vi.advanceTimersByTime(2700)
    deferred.resolve([
      {
        id: 'image-1',
        src: 'data:image/png;base64,abc123',
        revisedPrompt: '',
      },
    ])
    await flushPromises()

    expect(wrapper.text()).toContain('Elapsed 2.7s')
  })

  it('supports keyboard navigation and escape in image preview', async () => {
    mocks.generateImages.mockResolvedValue([
      {
        id: 'image-1',
        src: 'data:image/png;base64,abc123',
        revisedPrompt: '',
      },
      {
        id: 'image-2',
        src: 'data:image/png;base64,def456',
        revisedPrompt: '',
      },
    ])

    const wrapper = mountView()
    await flushPromises()

    await wrapper.find('textarea').setValue('draw a mountain sunrise')
    await wrapper.find('form').trigger('submit.prevent')
    await flushPromises()

    await wrapper.find('[data-testid="generated-image-button"]').trigger('click')
    await flushPromises()

    window.dispatchEvent(new KeyboardEvent('keydown', { key: 'ArrowRight' }))
    await flushPromises()
    expect(document.body.querySelector('[data-testid="image-preview-dialog"] img')?.getAttribute('src')).toBe(
      'data:image/png;base64,def456',
    )

    window.dispatchEvent(new KeyboardEvent('keydown', { key: 'Escape' }))
    await flushPromises()
    expect(document.body.querySelector('[data-testid="image-preview-dialog"]')).toBeNull()
  })

  it('offers continue edit and download all actions for generated results', async () => {
    mocks.generateImages.mockResolvedValue([
      {
        id: 'image-1',
        src: 'data:image/png;base64,abc123',
        revisedPrompt: '',
      },
      {
        id: 'image-2',
        src: 'data:image/png;base64,def456',
        revisedPrompt: '',
      },
    ])

    const wrapper = mountView()
    await flushPromises()

    await wrapper.find('textarea').setValue('draw a mountain sunrise')
    await wrapper.find('form').trigger('submit.prevent')
    await flushPromises()

    expect(wrapper.find('[data-testid="download-all-images"]').exists()).toBe(true)

    await wrapper.find('[data-testid="continue-edit-image"]').trigger('click')
    expect(wrapper.find('[data-testid="reference-image-preview"]').exists()).toBe(true)
    expect(wrapper.find('[data-testid="reference-image-preview"] img').attributes('src')).toBe('data:image/png;base64,abc123')
    expect((wrapper.find('textarea').element as HTMLTextAreaElement).value).toBe('')
    expect(wrapper.find('textarea').attributes('placeholder')).toBe('Describe how to edit this image')
  })

  it('submits edits with the selected reference image and can remove it', async () => {
    const wrapper = mountView()
    await flushPromises()

    await wrapper.find('textarea').setValue('draw a mountain sunrise')
    await wrapper.find('form').trigger('submit.prevent')
    await flushPromises()

    await wrapper.find('[data-testid="continue-edit-image"]').trigger('click')
    await wrapper.find('textarea').setValue('make the sky rainy')
    await wrapper.find('form').trigger('submit.prevent')
    await flushPromises()

    expect(mocks.generateImageEdit).toHaveBeenCalledTimes(1)
    expect(mocks.generateImageEdit.mock.calls[0][1]).toMatchObject({
      model: 'gpt-image-2',
      prompt: 'make the sky rainy',
      size: '1920x1088',
      count: 1,
      quality: 'auto',
    })
    expect(mocks.generateImageEdit.mock.calls[0][1].image).toBeInstanceOf(File)
    expect(wrapper.text()).toContain('make the sky rainy')

    await wrapper.find('[data-testid="continue-edit-image"]').trigger('click')
    await wrapper.find('[data-testid="remove-reference-image"]').trigger('click')
    expect(wrapper.find('[data-testid="reference-image-preview"]').exists()).toBe(false)
    expect(wrapper.find('textarea').attributes('placeholder')).toBe('Describe the image')
  })

  it('persists generated history in localStorage and restores it on the next mount', async () => {
    const wrapper = mountView()
    await flushPromises()

    await wrapper.find('textarea').setValue('draw a mountain sunrise')
    await wrapper.find('form').trigger('submit.prevent')
    await flushPromises()

    expect(localStorage.getItem('sunshine:image-generation-history')).toContain('draw a mountain sunrise')

    wrapper.unmount()
    const restoredWrapper = mountView()
    await flushPromises()

    expect(restoredWrapper.text()).toContain('draw a mountain sunrise')
    expect(restoredWrapper.find('[data-testid="generated-image-button"]').exists()).toBe(true)
  })

  it('keeps failed generation settings and retries them from the error panel', async () => {
    mocks.generateImages
      .mockRejectedValueOnce(new Error('network timeout'))
      .mockResolvedValueOnce([
        {
          id: 'image-1',
          src: 'data:image/png;base64,abc123',
          revisedPrompt: '',
        },
      ])

    const wrapper = mountView()
    await flushPromises()

    await wrapper.find('textarea').setValue('draw a retryable mountain sunrise')
    await wrapper.find('form').trigger('submit.prevent')
    await flushPromises()

    await wrapper.find('[data-testid="retry-image-generation"]').trigger('click')
    await flushPromises()

    expect(mocks.generateImages).toHaveBeenCalledTimes(2)
    expect(wrapper.text()).toContain('draw a retryable mountain sunrise')
    expect(wrapper.find('[data-testid="generated-image-button"]').exists()).toBe(true)
  })

  it('shows generation details in the image preview', async () => {
    vi.useFakeTimers()

    const wrapper = mountView()
    await flushPromises()

    await wrapper.find('textarea').setValue('draw a detailed mountain sunrise')
    await wrapper.find('form').trigger('submit.prevent')
    await flushPromises()

    await wrapper.find('[data-testid="generated-image-button"]').trigger('click')
    await flushPromises()

    const dialogText = document.body.querySelector('[data-testid="image-preview-dialog"]')?.textContent ?? ''
    expect(dialogText).toContain('Model')
    expect(dialogText).toContain('gpt-image-2')
    expect(dialogText).toContain('Size')
    expect(dialogText).toContain('1920x1088')
    expect(dialogText).toContain('Quality')
    expect(dialogText).toContain('auto')
  })

  it('uses prompt, size, and timestamp in download filenames', async () => {
    const wrapper = mountView()
    await flushPromises()

    await wrapper.find('textarea').setValue('Draw: Mountain Sunrise!!')
    await wrapper.find('form').trigger('submit.prevent')
    await flushPromises()

    const downloadName = wrapper.find('.icon-action[download]').attributes('download')
    expect(downloadName).toMatch(/^draw-mountain-sunrise-1920x1088-\d{8}-\d{6}-image-1\.png$/)
  })
})

function mountView() {
  return mount(ImageGenerationView, {
    global: {
      stubs: {
        AppLayout: { template: '<div><slot /></div>' },
        Icon: { template: '<span />' },
      },
    },
  })
}

function createDeferred() {
  let resolve!: (value: Array<{ id: string; src: string; revisedPrompt: string }>) => void
  const promise = new Promise<Array<{ id: string; src: string; revisedPrompt: string }>>((innerResolve) => {
    resolve = innerResolve
  })
  return { promise, resolve }
}

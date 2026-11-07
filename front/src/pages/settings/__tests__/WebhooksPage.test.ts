import type { Webhook } from '@entities/webhook'
import { Code, ConnectError } from '@connectrpc/connect'
import { SCheckbox } from '@shared/ui'
import { flushPromises, mount } from '@vue/test-utils'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { ref } from 'vue'
import WebhooksPage from '../WebhooksPage.vue'

const mocks = vi.hoisted(() => ({
  create: vi.fn(),
}))

const webhooks = ref<Webhook[]>([])

vi.mock('@entities/auth', () => ({
  useAuth: () => ({
    currentWorkspaceId: ref('ws-1'),
    workspaces: ref([{ workspaceId: 'ws-1', role: 'admin' }]),
  }),
}))

vi.mock('@entities/webhook', async importOriginal => ({
  ...await importOriginal<typeof import('@entities/webhook')>(),
  useWebhooks: () => ({ data: webhooks, isLoading: ref(false), error: ref(null) }),
  useCreateWebhook: () => ({ mutateAsync: mocks.create, isPending: ref(false) }),
  useUpdateWebhook: () => ({ mutateAsync: vi.fn(), isPending: ref(false) }),
  useDeleteWebhook: () => ({ mutateAsync: vi.fn(), isPending: ref(false) }),
  useTestWebhook: () => ({ mutateAsync: vi.fn(), isPending: ref(false) }),
}))

vi.mock('@shared/ui', async importOriginal => ({
  ...await importOriginal<typeof import('@shared/ui')>(),
  useToast: () => ({ success: vi.fn(), error: vi.fn() }),
}))

beforeEach(() => {
  vi.clearAllMocks()
  webhooks.value = []
})

async function submitCreate(url: string) {
  const wrapper = mount(WebhooksPage, { global: { stubs: { teleport: true } } })
  const openButton = wrapper.findAll('button').find(b => b.text().includes('Create webhook'))
  await openButton!.trigger('click')

  await wrapper.find('input[type="url"]').setValue(url)
  wrapper.findAllComponents(SCheckbox)[0].vm.$emit('update:modelValue', true)
  await wrapper.find('form').trigger('submit')
  await flushPromises()

  return wrapper
}

describe('webhooksPage', () => {
  it.each([
    'http://localhost:8080/hook',
    'https://localhost/hook',
    'https://127.0.0.1/hook',
    'https://10.0.0.5/hook',
    'https://192.168.1.10/hook',
    'https://172.20.0.1/hook',
    'https://169.254.169.254/latest',
    'https://[::1]/hook',
  ])('rejects %s without calling the API', async (url) => {
    const wrapper = await submitCreate(url)

    expect(wrapper.text()).toContain('URL must be HTTPS and reach a public address')
    expect(mocks.create).not.toHaveBeenCalled()
  })

  it('sends a public HTTPS URL', async () => {
    mocks.create.mockResolvedValue(undefined)
    await submitCreate('https://hooks.example.com/hook')

    expect(mocks.create).toHaveBeenCalledWith(expect.objectContaining({ url: 'https://hooks.example.com/hook' }))
  })

  it('shows the API error when the server rejects the URL', async () => {
    mocks.create.mockRejectedValue(new ConnectError('URL must not resolve to a private or internal address', Code.InvalidArgument))
    const wrapper = await submitCreate('https://internal.example.com/hook')

    expect(wrapper.text()).toContain('URL must not resolve to a private or internal address')
  })

  it('says in the hint that private addresses are not allowed', async () => {
    const wrapper = await submitCreate('http://localhost/hook')

    expect(wrapper.text()).toContain('Must be HTTPS. Localhost and private network addresses are not allowed.')
  })
})

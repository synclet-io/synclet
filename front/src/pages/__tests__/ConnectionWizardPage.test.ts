import { Code, ConnectError } from '@connectrpc/connect'
import { flushPromises, mount } from '@vue/test-utils'
import { describe, expect, it, vi } from 'vitest'
import { defineComponent, ref } from 'vue'
import { createMemoryHistory, createRouter } from 'vue-router'
import ConnectionWizardPage from '../ConnectionWizardPage.vue'

const mocks = vi.hoisted(() => ({ create: vi.fn() }))

vi.mock('@entities/connection', () => ({
  getConnection: vi.fn(async () => ({
    id: 'c1',
    name: 'Orders',
    sourceId: 's1',
    destinationId: 'd1',
    schedule: '',
    schemaChangePolicy: 'propagate',
    maxAttempts: 3,
    namespaceDefinition: 'source',
    customNamespaceFormat: '',
    streamPrefix: '',
    tags: ['eu'],
  })),
  getConfiguredCatalog: vi.fn(async () => undefined),
  configuredCatalogToStreams: () => [],
  useCreateConnection: () => ({ mutateAsync: mocks.create }),
  useConfigureStreams: () => ({ mutateAsync: vi.fn() }),
  useConnectionTags: () => ({ data: ref([]) }),
}))
vi.mock('@entities/source/api', () => ({ getSource: vi.fn(async () => ({ id: 's1', name: 'Orders DB' })) }))
vi.mock('@entities/destination/api', () => ({ getDestination: vi.fn(async () => ({ id: 'd1', name: 'Warehouse' })) }))
vi.mock('@entities/connector', () => ({ useManagedConnector: () => ({ data: ref(undefined) }) }))

const Empty = defineComponent({ template: '<div />' })

async function mountAtReview() {
  const router = createRouter({
    history: createMemoryHistory(),
    routes: [
      { path: '/connections/new', component: ConnectionWizardPage },
      { path: '/:rest(.*)*', component: Empty },
    ],
  })
  await router.push('/connections/new?cloneFrom=c1')
  const wrapper = mount(ConnectionWizardPage, {
    global: { plugins: [router], stubs: { StepSelectSource: true, StepSelectDestination: true, StepSettings: true, StepStreamConfig: true, WizardStepper: true } },
    attachTo: document.body,
  })
  await flushPromises()
  const button = (text: string) => wrapper.findAll('button').find(b => b.text() === text)!
  await button('Next').trigger('click')
  await button('Skip').trigger('click')
  await flushPromises()
  return { wrapper, button }
}

describe('connection wizard create errors', () => {
  it('shows a tag validation error under the tag input, not in the banner', async () => {
    mocks.create.mockRejectedValue(new ConnectError('validation error on tags: too many tags: 21 (max 20)', Code.InvalidArgument))
    const { wrapper, button } = await mountAtReview()

    await button('Create Connection').trigger('click')
    await flushPromises()

    const tagRoot = wrapper.get('[data-tag-input]')
    expect(tagRoot.get('[role="alert"]').text()).toBe('too many tags: 21 (max 20)')
    expect(wrapper.findAll('[role="alert"]')).toHaveLength(1)
    expect(wrapper.text().match(/too many tags/g)).toHaveLength(1)
  })

  it('shows other create errors in the banner', async () => {
    mocks.create.mockRejectedValue(new ConnectError('validation error on name: required', Code.InvalidArgument))
    const { wrapper, button } = await mountAtReview()

    await button('Create Connection').trigger('click')
    await flushPromises()

    expect(wrapper.get('[data-tag-input]').find('[role="alert"]').exists()).toBe(false)
    expect(wrapper.text()).toContain('validation error on name: required')
  })
})

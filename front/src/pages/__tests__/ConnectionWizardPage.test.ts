import { Code, ConnectError } from '@connectrpc/connect'
import { flushPromises, mount } from '@vue/test-utils'
import { describe, expect, it, vi } from 'vitest'
import { defineComponent, onMounted, ref } from 'vue'
import { createMemoryHistory, createRouter } from 'vue-router'
import ConnectionWizardPage from '../ConnectionWizardPage.vue'

const mocks = vi.hoisted(() => ({ create: vi.fn(), configure: vi.fn() }))

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
  useConfigureStreams: () => ({ mutateAsync: mocks.configure }),
  useConnectionTags: () => ({ data: ref([]) }),
}))
vi.mock('@entities/source/api', () => ({ getSource: vi.fn(async () => ({ id: 's1', name: 'Orders DB' })) }))
vi.mock('@entities/destination/api', () => ({ getDestination: vi.fn(async (id: string) => ({ id, name: 'Warehouse', executorGroup: id === 'd-eu' ? 'eu' : 'default' })) }))
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

describe('connection wizard streams step', () => {
  function streamStep(valid: boolean) {
    return defineComponent({
      emits: ['update:streams'],
      setup(_, { emit, expose }) {
        expose({ validate: () => valid })
        onMounted(() => emit('update:streams', [{ streamName: 'orders', namespace: '', syncMode: 'incremental', destinationSyncMode: 'append', cursorField: [], primaryKey: [], selectedFields: [] }]))
        return {}
      },
      template: '<div />',
    })
  }

  async function nextFromStreams(valid: boolean) {
    const router = createRouter({
      history: createMemoryHistory(),
      routes: [
        { path: '/connections/new', component: ConnectionWizardPage },
        { path: '/:rest(.*)*', component: Empty },
      ],
    })
    await router.push('/connections/new?cloneFrom=c1')
    const wrapper = mount(ConnectionWizardPage, {
      global: { plugins: [router], stubs: { StepSelectSource: true, StepSelectDestination: true, StepSettings: true, StepStreamConfig: streamStep(valid), WizardStepper: true } },
    })
    await flushPromises()
    const button = (text: string) => wrapper.findAll('button').find(b => b.text() === text)
    await button('Next')!.trigger('click')
    await flushPromises()
    await button('Next')!.trigger('click')
    await flushPromises()
    return { wrapper, button }
  }

  it('stays on the streams step while its validation fails', async () => {
    const { button } = await nextFromStreams(false)
    expect(button('Create Connection')).toBeUndefined()
    expect(button('Skip')).toBeDefined()
  })

  it('goes on to the review once the streams are valid', async () => {
    const { button } = await nextFromStreams(true)
    expect(button('Create Connection')).toBeDefined()
  })
})

describe('connection wizard transforms', () => {
  const mappers = [{ set: { cents: 'int(r.amount * 100.0)' } }]
  const seen: { sourceId?: string, connectionName?: string } = {}
  const streamStep = defineComponent({
    props: { sourceId: { type: String, default: '' }, connectionName: { type: String, default: '' } },
    emits: ['update:streams'],
    setup(props, { emit, expose }) {
      expose({ validate: () => true })
      onMounted(() => {
        seen.sourceId = props.sourceId
        seen.connectionName = props.connectionName
        emit('update:streams', [{ streamName: 'orders', namespace: '', syncMode: 'full_refresh', destinationSyncMode: 'overwrite', cursorField: [], primaryKey: [], selectedFields: [], mappers }])
      })
      return {}
    },
    template: '<div />',
  })

  it('gives the streams step the source and connection name and sends the transform steps on create', async () => {
    mocks.create.mockResolvedValue({ id: 'new-1' })
    mocks.configure.mockResolvedValue(undefined)
    const router = createRouter({
      history: createMemoryHistory(),
      routes: [
        { path: '/connections/new', component: ConnectionWizardPage },
        { path: '/:rest(.*)*', component: Empty },
      ],
    })
    await router.push('/connections/new?cloneFrom=c1')
    const wrapper = mount(ConnectionWizardPage, {
      global: { plugins: [router], stubs: { StepSelectSource: true, StepSelectDestination: true, StepSettings: true, StepStreamConfig: streamStep, WizardStepper: true } },
    })
    await flushPromises()
    const button = (text: string) => wrapper.findAll('button').find(b => b.text() === text)!
    await button('Next').trigger('click')
    await flushPromises()

    expect(seen).toEqual({ sourceId: 's1', connectionName: 'Orders (copy)' })

    await button('Next').trigger('click')
    await flushPromises()
    await button('Create Connection').trigger('click')
    await flushPromises()

    expect(mocks.configure).toHaveBeenCalledWith({
      connectionId: 'new-1',
      streams: [expect.objectContaining({ streamName: 'orders', mappers })],
    })
  })
})

describe('connection wizard executor groups', () => {
  const seen: { sourceGroup?: string, destinationGroup?: string, destinationId?: string } = {}
  const picked = { group: 'default' }
  const sourceStep = defineComponent({
    props: { sourceId: { type: String, default: '' }, group: { type: String, default: undefined } },
    emits: ['update:sourceId', 'update:source', 'auto-advance'],
    setup(props, { emit }) {
      onMounted(() => {
        seen.sourceGroup = props.group
      })
      function pick() {
        emit('update:sourceId', `s-${picked.group}`)
        emit('update:source', { id: `s-${picked.group}`, name: 'Orders DB', executorGroup: picked.group })
        emit('auto-advance')
      }
      return { pick }
    },
    template: '<button type="button" @click="pick">pick source</button>',
  })
  const destinationStep = defineComponent({
    props: { destinationId: { type: String, default: '' }, group: { type: String, default: undefined } },
    setup(props) {
      onMounted(() => {
        seen.destinationGroup = props.group
        seen.destinationId = props.destinationId
      })
      return {}
    },
    template: '<div />',
  })

  async function mountWithDestination() {
    const router = createRouter({
      history: createMemoryHistory(),
      routes: [
        { path: '/connections/new', component: ConnectionWizardPage },
        { path: '/:rest(.*)*', component: Empty },
      ],
    })
    await router.push('/connections/new?destinationId=d-eu')
    const wrapper = mount(ConnectionWizardPage, {
      global: { plugins: [router], stubs: { StepSelectSource: sourceStep, StepSelectDestination: destinationStep, StepSettings: true, StepStreamConfig: true, WizardStepper: true } },
    })
    await flushPromises()
    return wrapper
  }

  it('keeps offering sources of the preselected destination group after going back', async () => {
    picked.group = 'eu'
    const wrapper = await mountWithDestination()
    await wrapper.findAll('button').find(b => b.text() === 'pick source')!.trigger('click')
    await flushPromises()
    expect(seen.destinationId).toBe('d-eu')

    seen.sourceGroup = undefined
    await wrapper.findAll('button').find(b => b.text() === 'Back')!.trigger('click')
    await flushPromises()
    expect(seen.sourceGroup).toBe('eu')
  })

  it('offers sources of a preselected destination group and drops the destination when a source of another group is picked', async () => {
    picked.group = 'default'
    const router = createRouter({
      history: createMemoryHistory(),
      routes: [
        { path: '/connections/new', component: ConnectionWizardPage },
        { path: '/:rest(.*)*', component: Empty },
      ],
    })
    await router.push('/connections/new?destinationId=d-eu')
    const wrapper = mount(ConnectionWizardPage, {
      global: { plugins: [router], stubs: { StepSelectSource: sourceStep, StepSelectDestination: destinationStep, StepSettings: true, StepStreamConfig: true, WizardStepper: true } },
    })
    await flushPromises()
    expect(seen.sourceGroup).toBe('eu')

    await wrapper.findAll('button').find(b => b.text() === 'pick source')!.trigger('click')
    await flushPromises()
    expect(seen).toEqual({ sourceGroup: 'eu', destinationGroup: 'default', destinationId: '' })
  })
})

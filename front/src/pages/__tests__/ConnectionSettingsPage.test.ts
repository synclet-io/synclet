/* eslint-disable no-template-curly-in-string -- ${SOURCE_NAMESPACE} is the literal placeholder under test */
import { Code, ConnectError } from '@connectrpc/connect'
import { flushPromises, mount } from '@vue/test-utils'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { ref } from 'vue'
import ConnectionSettingsPage from '../ConnectionSettingsPage.vue'

const mocks = vi.hoisted(() => ({
  role: 'editor',
  workflows: [] as { id: string, name: string }[],
  mutateAsync: vi.fn(),
  toastError: vi.fn(),
}))

vi.mock('vue-router', () => ({
  useRoute: () => ({ params: { id: 'c1' } }),
}))
vi.mock('@entities/auth', () => ({
  useAuth: () => ({
    currentWorkspaceId: ref('ws1'),
    workspaces: ref([{ workspaceId: 'ws1', role: mocks.role }]),
  }),
}))
vi.mock('@entities/connection', () => ({
  useConnection: () => ({
    data: ref({
      id: 'c1',
      name: 'Orders',
      schedule: '',
      schemaChangePolicy: 'propagate',
      maxAttempts: 3,
      namespaceDefinition: 'source',
      customNamespaceFormat: '',
      streamPrefix: '',
      tags: ['eu'],
      workflows: mocks.workflows,
    }),
  }),
  useUpdateConnection: () => ({ mutateAsync: mocks.mutateAsync, isPending: ref(false) }),
  useConnectionTags: () => ({ data: ref([{ tag: 'eu', connectionCount: 1 }, { tag: 'prod', connectionCount: 2 }]) }),
  useConfiguredCatalog: () => ({ data: ref({ streams: [{ stream: { name: 'orders', namespace: 'sales' } }, { stream: { name: 'users', namespace: 'public' } }] }) }),
  configuredCatalogToStreams: (c: { streams: { stream: { name: string, namespace: string } }[] }) => c.streams.map(s => ({ streamName: s.stream.name, namespace: s.stream.namespace })),
}))
vi.mock('@shared/ui', async importOriginal => ({
  ...(await importOriginal<typeof import('@shared/ui')>()),
  useToast: () => ({ success: vi.fn(), error: mocks.toastError }),
}))

function mountPage() {
  return mount(ConnectionSettingsPage, {
    global: { stubs: { SCronEditor: true, RouterLink: { props: ['to'], template: '<a :href="to"><slot /></a>' } } },
    attachTo: document.body,
  })
}

function tagInput(w: ReturnType<typeof mountPage>) {
  return w.get('input[role="combobox"]')
}

function saveButton(w: ReturnType<typeof mountPage>) {
  return w.findAll('button').find(b => b.text() === 'Save Settings')!
}

describe('connection settings tags', () => {
  beforeEach(() => {
    document.body.innerHTML = ''
    mocks.role = 'editor'
    mocks.workflows = []
    mocks.mutateAsync.mockReset()
    mocks.toastError.mockReset()
  })

  it('places the tag input under the name in the General card', async () => {
    const w = mountPage()
    await flushPromises()

    const titles = w.findAll('h3').map(h => h.text())
    expect(titles[0]).toBe('General')
    expect(titles).not.toContain('Tags')
    const card = w.findAll('h3')[0]!.element.closest('.rounded-xl')!
    const labels = Array.from(card.querySelectorAll('label'), l => l.textContent?.trim())
    expect(labels).toEqual(['Name', 'Tags'])
    expect(w.findAll('label').filter(l => l.text() === 'Tags')).toHaveLength(1)
  })

  it('sends the edited tags on save', async () => {
    mocks.mutateAsync.mockResolvedValue({})
    const w = mountPage()
    await flushPromises()

    const input = tagInput(w)
    await input.setValue('prod')
    await input.trigger('keydown', { key: 'Enter' })
    await saveButton(w).trigger('click')
    await flushPromises()

    expect(mocks.mutateAsync).toHaveBeenCalledWith(expect.objectContaining({ id: 'c1', tags: ['eu', 'prod'] }))
  })

  it('shows a tags validation error under the tag input', async () => {
    mocks.mutateAsync.mockRejectedValue(new ConnectError('validation error on tags: too many tags: 21 (max 20)', Code.InvalidArgument))
    const w = mountPage()
    await flushPromises()

    await saveButton(w).trigger('click')
    await flushPromises()

    expect(w.get('[role="alert"]').text()).toBe('too many tags: 21 (max 20)')
    expect(mocks.toastError).not.toHaveBeenCalled()

    const input = tagInput(w)
    await input.setValue('prod')
    await input.trigger('keydown', { key: 'Enter' })
    expect(w.find('[role="alert"]').exists()).toBe(false)
  })

  it('toasts errors on other fields', async () => {
    mocks.mutateAsync.mockRejectedValue(new ConnectError('validation error on schedule: bad cron', Code.InvalidArgument))
    const w = mountPage()
    await flushPromises()

    await saveButton(w).trigger('click')
    await flushPromises()

    expect(mocks.toastError).toHaveBeenCalled()
    expect(w.find('[role="alert"]').exists()).toBe(false)
  })

  it('shows a viewer the tags without an input or remove buttons', async () => {
    mocks.role = 'viewer'
    const w = mountPage()
    await flushPromises()

    expect(w.find('[data-tag="eu"]').exists()).toBe(true)
    expect(w.find('input[role="combobox"]').exists()).toBe(false)
    expect(w.find('button[aria-label^="Remove tag"]').exists()).toBe(false)
  })
})

describe('connection settings schedule', () => {
  beforeEach(() => {
    document.body.innerHTML = ''
    mocks.role = 'editor'
    mocks.workflows = []
    mocks.mutateAsync.mockReset()
    mocks.mutateAsync.mockResolvedValue({})
  })

  it('edits the schedule of a connection in no workflow', async () => {
    const w = mountPage()
    await flushPromises()

    expect(w.findComponent({ name: 'SCronEditor' }).exists()).toBe(true)
    expect(w.find('[data-testid="run-by-workflow"]').exists()).toBe(false)

    await saveButton(w).trigger('click')
    await flushPromises()
    expect(mocks.mutateAsync).toHaveBeenCalledWith(expect.objectContaining({ schedule: '' }))
  })

  it('says a step connection is run by its workflows and sends no schedule', async () => {
    mocks.workflows = [{ id: 'w1', name: 'Nightly' }, { id: 'w2', name: 'Hourly' }]
    const w = mountPage()
    await flushPromises()

    expect(w.findComponent({ name: 'SCronEditor' }).exists()).toBe(false)
    const note = w.get('[data-testid="run-by-workflow"]')
    expect(note.text()).toContain('run by')
    expect(note.findAll('a').map(a => [a.text(), a.attributes('href')])).toEqual([['Nightly', '/workflows/w1'], ['Hourly', '/workflows/w2']])

    await saveButton(w).trigger('click')
    await flushPromises()
    expect(mocks.mutateAsync).toHaveBeenCalledTimes(1)
    expect(mocks.mutateAsync.mock.calls[0]![0]).not.toHaveProperty('schedule')
  })
})

describe('connection settings destination naming', () => {
  beforeEach(() => {
    document.body.innerHTML = ''
    mocks.role = 'editor'
    mocks.workflows = []
    mocks.mutateAsync.mockReset()
  })

  it('separates the destination namespace from the table name prefix and shows the first stream as the example', async () => {
    const w = mountPage()
    await flushPromises()

    expect(w.get('fieldset legend').text()).toBe('Destination namespace')
    expect(w.findAll('label').some(l => l.text() === 'Table name prefix')).toBe(true)
    expect(w.findAll('label').some(l => l.text() === 'Stream Prefix')).toBe(false)
    expect(w.get('[data-testid="destination-name-example"]').text()).toContain('sales.orders → sales.orders')
  })

  it('saves "Source namespace with a prefix" as a custom format', async () => {
    mocks.mutateAsync.mockResolvedValue({})
    const w = mountPage()
    await flushPromises()

    await w.get('fieldset select').setValue('source_prefix')
    const label = w.findAll('label').find(l => l.text() === 'Namespace prefix')!
    await w.get(`#${label.attributes('for')}`).setValue('prod_')
    await saveButton(w).trigger('click')
    await flushPromises()

    expect(mocks.mutateAsync).toHaveBeenCalledWith(expect.objectContaining({ namespaceDefinition: 'custom', customNamespaceFormat: 'prod_${SOURCE_NAMESPACE}' }))
  })
})

import { Code, ConnectError } from '@connectrpc/connect'
import { flushPromises, mount } from '@vue/test-utils'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { defineComponent, h, ref } from 'vue'
import ConnectorsSettingsPage from '../ConnectorsSettingsPage.vue'

const mocks = vi.hoisted(() => ({
  role: 'editor',
  deleteConnector: vi.fn(),
  toastSuccess: vi.fn(),
  toastError: vi.fn(),
}))

vi.mock('@entities/auth', () => ({
  useAuth: () => ({
    currentWorkspaceId: ref('ws1'),
    workspaces: ref([{ workspaceId: 'ws1', role: mocks.role }]),
  }),
}))
vi.mock('@entities/connector', () => {
  const mutation = () => ({ mutate: vi.fn(), mutateAsync: vi.fn(), isPending: ref(false) })
  return {
    useManagedConnectors: () => ({
      data: ref([
        { id: 'c1', name: 'Faker', connectorType: 'source', dockerImage: 'airbyte/source-faker', dockerTag: '6.2', starred: true, used: false },
        { id: 'c2', name: 'Stripe', connectorType: 'source', dockerImage: 'airbyte/source-stripe', dockerTag: '5.0', starred: false, used: true },
      ]),
      isLoading: ref(false),
    }),
    useAddConnector: mutation,
    useUpdateManagedConnector: mutation,
    useBatchUpdateConnectors: mutation,
    useDeleteManagedConnector: () => ({ mutateAsync: mocks.deleteConnector, isPending: ref(false) }),
  }
})
vi.mock('@entities/repository', () => {
  const mutation = () => ({ mutate: vi.fn(), mutateAsync: vi.fn(), isPending: ref(false) })
  return {
    useRepositories: () => ({ data: ref([]), isLoading: ref(false) }),
    useAddRepository: mutation,
    useDeleteRepository: mutation,
    useSyncRepository: mutation,
  }
})
vi.mock('@shared/ui', async importOriginal => ({
  ...(await importOriginal<typeof import('@shared/ui')>()),
  useToast: () => ({ success: mocks.toastSuccess, error: mocks.toastError }),
}))
vi.mock('@features/connector-update', () => ({ BreakingChangeDialog: defineComponent(() => () => null) }))
vi.mock('@features/connector-star', () => ({
  ConnectorStarToggle: defineComponent({
    props: { connector: { type: Object, required: true } },
    setup: props => () => h('span', { 'data-testid': 'star-toggle' }, `${(props.connector as { id: string }).id}:${(props.connector as { starred: boolean }).starred}`),
  }),
}))

function mountPage() {
  return mount(ConnectorsSettingsPage, { global: { stubs: { RouterLink: true } }, attachTo: document.body })
}

function deleteButtons() {
  return Array.from(document.querySelectorAll<HTMLButtonElement>('button[aria-label^="Delete "]'))
}

function confirmButton() {
  return Array.from(document.querySelectorAll<HTMLButtonElement>('button')).find(b => b.textContent?.trim() === 'Delete Connector')
}

describe('connectorsSettingsPage', () => {
  beforeEach(() => {
    document.body.innerHTML = ''
    mocks.role = 'editor'
    mocks.deleteConnector.mockReset()
    mocks.toastSuccess.mockReset()
    mocks.toastError.mockReset()
  })

  it('has a star toggle on each installed connector', async () => {
    const wrapper = mountPage()
    await flushPromises()
    expect(wrapper.findAll('[data-testid="star-toggle"]').map(t => t.text())).toEqual(['c1:true', 'c2:false'])
  })

  it('has a delete action on each installed connector for editors', async () => {
    mountPage()
    await flushPromises()
    expect(deleteButtons().map(b => b.getAttribute('aria-label'))).toEqual(['Delete Faker', 'Delete Stripe'])
  })

  it('hides the delete action from viewers', async () => {
    mocks.role = 'viewer'
    mountPage()
    await flushPromises()
    expect(deleteButtons()).toHaveLength(0)
  })

  it('deletes the connector once the dialog is confirmed', async () => {
    mocks.deleteConnector.mockResolvedValue(undefined)
    mountPage()
    await flushPromises()

    deleteButtons()[0]!.click()
    await flushPromises()
    expect(document.body.textContent).toContain('Are you sure you want to delete Faker?')
    expect(mocks.deleteConnector).not.toHaveBeenCalled()

    confirmButton()!.click()
    await flushPromises()
    expect(mocks.deleteConnector).toHaveBeenCalledWith('c1')
    expect(mocks.toastSuccess).toHaveBeenCalledWith('Faker deleted')
    expect(confirmButton()).toBeUndefined()
  })

  it('shows the server reason when the connector is still in use', async () => {
    mocks.deleteConnector.mockRejectedValue(new ConnectError('cannot delete connector: used by 2 source(s)', Code.InvalidArgument))
    mountPage()
    await flushPromises()

    deleteButtons()[1]!.click()
    await flushPromises()
    confirmButton()!.click()
    await flushPromises()
    expect(mocks.deleteConnector).toHaveBeenCalledWith('c2')
    expect(mocks.toastError).toHaveBeenCalledWith('cannot delete connector: used by 2 source(s)')
    expect(mocks.toastSuccess).not.toHaveBeenCalled()
  })
})

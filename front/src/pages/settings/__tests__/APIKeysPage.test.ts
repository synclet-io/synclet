import type { APIKey } from '@entities/auth'
import { SConfirmDialog } from '@shared/ui'
import { flushPromises, mount } from '@vue/test-utils'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { ref } from 'vue'
import APIKeysPage from '../APIKeysPage.vue'

const mocks = vi.hoisted(() => ({
  revoke: vi.fn(),
}))

const keys = ref<APIKey[]>([])

vi.mock('@entities/auth', () => ({
  useAuth: () => ({ currentWorkspaceId: ref('ws-1') }),
}))

vi.mock('@entities/auth/apiKeys', () => ({
  useAPIKeys: () => ({ data: keys, isLoading: ref(false), error: ref(null) }),
  useCreateAPIKey: () => ({ mutateAsync: vi.fn(), isPending: ref(false) }),
  useRevokeAPIKey: () => ({ mutateAsync: mocks.revoke, isPending: ref(false) }),
}))

vi.mock('@shared/ui', async importOriginal => ({
  ...await importOriginal<typeof import('@shared/ui')>(),
  useToast: () => ({ success: vi.fn(), error: vi.fn() }),
}))

function apiKey(overrides: Partial<APIKey>): APIKey {
  return {
    id: 'key-1',
    workspaceId: 'ws-1',
    name: 'CI',
    createdAt: new Date(),
    expiresAt: undefined,
    lastUsedAt: undefined,
    oauth: null,
    ...overrides,
  }
}

beforeEach(() => {
  vi.clearAllMocks()
  keys.value = [
    apiKey({ id: 'key-1', name: 'CI' }),
    apiKey({ id: 'key-2', name: 'Claude (OAuth)', oauth: { clientName: 'Claude', level: 'actions' } }),
  ]
})

describe('aPIKeysPage', () => {
  it('shows the client name and access level of OAuth keys', () => {
    const wrapper = mount(APIKeysPage)
    const rows = wrapper.findAll('tbody tr')
    expect(rows[1].text()).toContain('Claude')
    expect(rows[1].text()).toContain('Read + actions')
    expect(rows[0].text()).not.toContain('Read')
  })

  it('revokes an OAuth key like any other key and warns that the client loses access', async () => {
    mocks.revoke.mockResolvedValue(undefined)
    const wrapper = mount(APIKeysPage)

    await wrapper.find('button[aria-label="Revoke Claude (OAuth)"]').trigger('click')
    const dialog = wrapper.findComponent(SConfirmDialog)
    expect(dialog.props('message')).toContain('refresh token stops working')
    dialog.vm.$emit('confirm')
    await flushPromises()

    expect(mocks.revoke).toHaveBeenCalledWith('key-2')
  })
})

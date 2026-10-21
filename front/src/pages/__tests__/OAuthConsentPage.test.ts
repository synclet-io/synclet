import { flushPromises, mount } from '@vue/test-utils'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { ref } from 'vue'
import OAuthConsentPage from '../OAuthConsentPage.vue'

const mocks = vi.hoisted(() => ({
  query: {} as Record<string, string>,
  getOAuthClient: vi.fn(),
  authorizeOAuth: vi.fn(),
  redirectTo: vi.fn(),
}))

vi.mock('vue-router', async importOriginal => ({
  ...await importOriginal<typeof import('vue-router')>(),
  useRoute: () => ({ query: mocks.query }),
}))

vi.mock('@entities/auth/oauth', async importOriginal => ({
  ...await importOriginal<typeof import('@entities/auth/oauth')>(),
  getOAuthClient: mocks.getOAuthClient,
  authorizeOAuth: mocks.authorizeOAuth,
}))

vi.mock('@shared/lib/navigation', () => ({
  redirectTo: mocks.redirectTo,
}))

vi.mock('@entities/auth', () => ({
  useAuth: () => ({
    currentWorkspaceId: ref('ws-2'),
    workspaces: ref([
      { workspaceId: 'ws-1', workspaceName: 'Acme', role: 'admin' },
      { workspaceId: 'ws-2', workspaceName: 'Beta', role: 'editor' },
    ]),
  }),
}))

const authorizeQuery = {
  response_type: 'code',
  client_id: 'client-1',
  redirect_uri: 'https://claude.ai/api/mcp/auth_callback',
  code_challenge: 'challenge',
  code_challenge_method: 'S256',
  state: 'xyz',
}

beforeEach(() => {
  vi.clearAllMocks()
  mocks.query = { ...authorizeQuery }
  mocks.getOAuthClient.mockResolvedValue({ clientId: 'client-1', clientName: 'Claude' })
})

describe('oAuthConsentPage', () => {
  it('shows the client name, the user\'s workspaces and the three access levels', async () => {
    const wrapper = mount(OAuthConsentPage)
    await flushPromises()

    expect(mocks.getOAuthClient).toHaveBeenCalledWith('client-1', authorizeQuery.redirect_uri)
    expect(wrapper.text()).toContain('Claude')
    const options = wrapper.findAll('select option').map(o => o.text())
    expect(options).toEqual(['Acme', 'Beta'])
    expect((wrapper.find('select').element as HTMLSelectElement).value).toBe('ws-2')
    expect(wrapper.findAll('input[type="radio"]')).toHaveLength(3)
    expect(wrapper.text()).toContain('Read')
    expect(wrapper.text()).toContain('Read + actions')
    expect(wrapper.text()).toContain('Read + actions + write')
  })

  it('on Allow issues a code for the chosen workspace and level and follows the returned URL', async () => {
    mocks.authorizeOAuth.mockResolvedValue('https://claude.ai/api/mcp/auth_callback?code=abc&state=xyz')
    const wrapper = mount(OAuthConsentPage)
    await flushPromises()

    await wrapper.find('select').setValue('ws-1')
    await wrapper.find('input[type="radio"][value="actions"]').setValue(true)
    await wrapper.find('[data-testid="oauth-allow"]').trigger('click')
    await flushPromises()

    expect(mocks.authorizeOAuth).toHaveBeenCalledWith('ws-1', {
      clientId: 'client-1',
      redirectUri: authorizeQuery.redirect_uri,
      codeChallenge: 'challenge',
      codeChallengeMethod: 'S256',
      level: 'actions',
      state: 'xyz',
    })
    expect(mocks.redirectTo).toHaveBeenCalledWith('https://claude.ai/api/mcp/auth_callback?code=abc&state=xyz')
  })

  it('on Deny redirects to the redirect URI with error=access_denied and the state', async () => {
    const wrapper = mount(OAuthConsentPage)
    await flushPromises()

    await wrapper.find('[data-testid="oauth-deny"]').trigger('click')

    expect(mocks.authorizeOAuth).not.toHaveBeenCalled()
    const target = new URL(mocks.redirectTo.mock.calls[0][0])
    expect(`${target.origin}${target.pathname}`).toBe(authorizeQuery.redirect_uri)
    expect(target.searchParams.get('error')).toBe('access_denied')
    expect(target.searchParams.get('state')).toBe('xyz')
  })

  it('does not redirect anywhere when the client or redirect URI is not registered', async () => {
    mocks.getOAuthClient.mockRejectedValue(new Error('redirect URI is not registered'))
    const wrapper = mount(OAuthConsentPage)
    await flushPromises()

    expect(wrapper.find('[data-testid="oauth-allow"]').exists()).toBe(false)
    expect(wrapper.find('[data-testid="oauth-deny"]').exists()).toBe(false)
    expect(wrapper.text()).toContain('redirect URI is not registered')
    expect(mocks.redirectTo).not.toHaveBeenCalled()
  })

  it('rejects a request without client_id or redirect_uri without calling the server', async () => {
    mocks.query = { response_type: 'code' }
    const wrapper = mount(OAuthConsentPage)
    await flushPromises()

    expect(mocks.getOAuthClient).not.toHaveBeenCalled()
    expect(wrapper.find('[data-testid="oauth-allow"]').exists()).toBe(false)
  })
})

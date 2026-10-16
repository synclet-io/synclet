import { flushPromises, mount } from '@vue/test-utils'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { ref } from 'vue'
import ForgotPasswordPage from '../ForgotPasswordPage.vue'
import LoginPage from '../LoginPage.vue'
import ResetPasswordPage from '../ResetPasswordPage.vue'

const mocks = vi.hoisted(() => ({
  push: vi.fn(),
  query: {} as Record<string, string>,
  requestPasswordReset: vi.fn(),
  resetPassword: vi.fn(),
  logout: vi.fn(),
}))

vi.mock('vue-router', async importOriginal => ({
  ...await importOriginal<typeof import('vue-router')>(),
  useRoute: () => ({ query: mocks.query }),
  useRouter: () => ({ push: mocks.push }),
}))

vi.mock('@entities/auth', () => ({
  requestPasswordReset: mocks.requestPasswordReset,
  resetPassword: mocks.resetPassword,
  getOIDCProviders: vi.fn().mockResolvedValue([]),
  useAuth: () => ({ isAuthenticated: ref(false), logout: mocks.logout, login: vi.fn() }),
}))

const systemInfo = ref<{ emailEnabled: boolean, registrationEnabled: boolean } | undefined>()
vi.mock('@entities/system', () => ({
  useSystemInfo: () => ({ data: systemInfo }),
}))

const stubs = { global: { stubs: { RouterLink: { template: '<a><slot /></a>' } } } }

beforeEach(() => {
  vi.clearAllMocks()
  mocks.query = {}
  systemInfo.value = { emailEnabled: true, registrationEnabled: true }
})

describe('loginPage', () => {
  it('links to the forgot-password page when email is enabled', () => {
    const wrapper = mount(LoginPage, stubs)
    expect(wrapper.text()).toContain('Forgot password?')
  })

  it('hides the forgot-password link when email is disabled', () => {
    systemInfo.value = { emailEnabled: false, registrationEnabled: true }
    const wrapper = mount(LoginPage, stubs)
    expect(wrapper.text()).not.toContain('Forgot password?')
  })
})

describe('forgotPasswordPage', () => {
  it('requests a reset and shows the neutral confirmation', async () => {
    mocks.requestPasswordReset.mockResolvedValue(undefined)
    const wrapper = mount(ForgotPasswordPage, stubs)
    await wrapper.find('input[type="email"]').setValue('jane@example.com')
    await wrapper.find('form').trigger('submit')
    await flushPromises()

    expect(mocks.requestPasswordReset).toHaveBeenCalledWith('jane@example.com')
    expect(wrapper.text()).toContain('If an account exists, we sent a link to reset your password.')
    expect(wrapper.find('form').exists()).toBe(false)
  })
})

describe('resetPasswordPage', () => {
  it('resets the password and goes to login', async () => {
    mocks.query = { token: 'raw-token' }
    mocks.resetPassword.mockResolvedValue(undefined)
    const wrapper = mount(ResetPasswordPage, stubs)
    const inputs = wrapper.findAll('input[type="password"]')
    await inputs[0].setValue('new-password-456')
    await inputs[1].setValue('new-password-456')
    await wrapper.find('form').trigger('submit')
    await flushPromises()

    expect(mocks.resetPassword).toHaveBeenCalledWith('raw-token', 'new-password-456')
    expect(mocks.push).toHaveBeenCalledWith({ name: 'login' })
  })

  it('rejects mismatched passwords without calling the API', async () => {
    mocks.query = { token: 'raw-token' }
    const wrapper = mount(ResetPasswordPage, stubs)
    const inputs = wrapper.findAll('input[type="password"]')
    await inputs[0].setValue('new-password-456')
    await inputs[1].setValue('other-password-789')
    await wrapper.find('form').trigger('submit')
    await flushPromises()

    expect(mocks.resetPassword).not.toHaveBeenCalled()
    expect(wrapper.text()).toContain('Passwords do not match')
  })

  it('shows an invalid-link message when the token is missing', () => {
    const wrapper = mount(ResetPasswordPage, stubs)
    expect(wrapper.text()).toContain('This reset link is invalid or has expired.')
    expect(wrapper.find('form').exists()).toBe(false)
  })
})

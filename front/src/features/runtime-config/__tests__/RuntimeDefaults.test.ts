import type { RuntimeDefaults } from '@entities/system'
import { mount } from '@vue/test-utils'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { ref } from 'vue'
import RuntimeConfigForm from '../RuntimeConfigForm.vue'
import RuntimeDefaultsCard from '../RuntimeDefaultsCard.vue'

const defaults: RuntimeDefaults = {
  executor: 'docker',
  cpuRequest: '',
  cpuLimit: '2',
  memoryRequest: '',
  memoryLimit: '2Gi',
  serviceAccountName: '',
  maxConcurrentJobs: 10,
  idleTimeoutSeconds: 600,
  logArchiveBackend: 's3',
  dockerMaxConcurrentJobs: 5,
}

const data = ref<RuntimeDefaults | undefined>()
const isLoading = ref(false)
const isError = ref(false)
const error = ref<Error | null>(null)
vi.mock('@entities/system', () => ({
  useRuntimeDefaults: () => ({ data, isLoading, isError, error, refetch: vi.fn() }),
}))

beforeEach(() => {
  data.value = defaults
  isLoading.value = false
  isError.value = false
  error.value = null
})

describe('runtimeDefaultsCard', () => {
  it('lists the effective defaults', () => {
    const text = mount(RuntimeDefaultsCard).text()
    expect(text).toContain('Docker')
    expect(text).toContain('2Gi')
    expect(text).toContain('10m')
    expect(text).toContain('s3')
    expect(text).toContain('Not set')
    expect(text).toContain('Docker executor concurrent syncs')
  })

  it('hides the Docker limit for other executors', () => {
    data.value = { ...defaults, executor: 'k8s' }
    expect(mount(RuntimeDefaultsCard).text()).not.toContain('Docker executor concurrent syncs')
  })

  it('shows a skeleton while loading', () => {
    data.value = undefined
    isLoading.value = true
    expect(mount(RuntimeDefaultsCard).find('[data-testid="runtime-defaults-loading"]').exists()).toBe(true)
  })

  it('shows the error', () => {
    data.value = undefined
    isError.value = true
    error.value = new Error('boom')
    expect(mount(RuntimeDefaultsCard).text()).toContain('boom')
  })
})

describe('runtimeConfigForm', () => {
  it('shows the global default as placeholder', () => {
    const wrapper = mount(RuntimeConfigForm, { props: { runtimeConfig: null, entityType: 'source', saving: false } })
    const placeholders = wrapper.findAll('input').map(i => i.attributes('placeholder'))
    expect(placeholders).toContain('Default: 2')
    expect(placeholders).toContain('Default: 2Gi')
  })
})

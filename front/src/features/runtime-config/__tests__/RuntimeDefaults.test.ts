import type { RuntimeDefaults } from '@entities/system'
import { mount } from '@vue/test-utils'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { ref } from 'vue'
import RuntimeConfigForm from '../RuntimeConfigForm.vue'
import RuntimeDefaultsCard from '../RuntimeDefaultsCard.vue'

const defaults: RuntimeDefaults = {
  cpuRequest: '',
  cpuLimit: '2',
  memoryRequest: '',
  memoryLimit: '2Gi',
  serviceAccountName: '',
  maxConcurrentJobs: 10,
  idleTimeoutSeconds: 600,
  logArchiveBackend: 's3',
  executors: [],
}

const noResources = { cpuRequest: '', cpuLimit: '', memoryRequest: '', memoryLimit: '', cpuExecutorDefault: false, memoryExecutorDefault: false }

const data = ref<RuntimeDefaults | undefined>()
const isLoading = ref(false)
const isError = ref(false)
const error = ref<Error | null>(null)
vi.mock('@entities/system', async importOriginal => ({
  ...await importOriginal<typeof import('@entities/system')>(),
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
    const wrapper = mount(RuntimeDefaultsCard)
    const text = wrapper.text()
    expect(text).toContain('2Gi')
    expect(text).toContain('10m')
    expect(text).toContain('s3')
    expect(text).toContain('Not set')
    const rows = wrapper.findAll('dl > div').map(r => r.text())
    expect(rows.filter(r => r.startsWith('Executor'))).toEqual([])
  })

  it('lists the defaults of each executor type', () => {
    data.value = {
      ...defaults,
      executors: [
        { type: 'docker', capacity: 5, resourceDefaults: { ...noResources, cpuLimit: '2', memoryLimit: '2Gi' } },
        {
          type: 'k8s',
          capacity: 0,
          resourceDefaults: { ...noResources, cpuRequest: '750m', cpuLimit: '750m', cpuExecutorDefault: true, memoryLimit: '1Gi' },
        },
      ],
    }
    const rows = mount(RuntimeDefaultsCard).findAll('dl > div').map(r => r.text())
    expect(rows).toContain('Docker · Capacity5')
    expect(rows).toContain('Docker · CPU limit2')
    expect(rows).toContain('Docker · Memory limit2Gi')
    expect(rows).toContain('Kubernetes · CapacityUnlimited')
    expect(rows).toContain('Kubernetes · CPU request750m (executor default)')
    expect(rows).toContain('Kubernetes · CPU limit750m (executor default)')
    expect(rows).toContain('Kubernetes · Memory limit1Gi')
    expect(rows).toContain('Kubernetes · Memory requestNot set')
  })

  it('says when no executor has reported', () => {
    expect(mount(RuntimeDefaultsCard).text()).toContain('No executor has reported yet')
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

import type { ExecutorInstance } from '@entities/system'
import { mount } from '@vue/test-utils'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { ref } from 'vue'
import SystemPage from '../SystemPage.vue'

const executors = ref<ExecutorInstance[] | undefined>([])

vi.mock('@entities/system', async importOriginal => ({
  ...await importOriginal<typeof import('@entities/system')>(),
  useExecutors: () => ({ data: executors, isLoading: ref(false), error: ref(null) }),
}))

const resourceDefaults = { cpuRequest: '', cpuLimit: '', memoryRequest: '', memoryLimit: '', cpuExecutorDefault: false, memoryExecutorDefault: false }

function executor(overrides: Partial<ExecutorInstance>): ExecutorInstance {
  return {
    id: 'executor-0',
    type: 'docker',
    version: 'v0.5.0',
    capacity: 5,
    nativeDestinations: [],
    resourceDefaults,
    firstSeenAt: new Date(),
    lastSeenAt: new Date(),
    ...overrides,
  }
}

beforeEach(() => {
  executors.value = []
})

describe('systemPage', () => {
  it('lists the executors known to the control plane', () => {
    executors.value = [
      executor({ id: 'synclet-docker', type: 'docker', version: 'v0.5.0', capacity: 5 }),
      executor({ id: 'synclet-executor-0', type: 'k8s', version: 'v0.5.1', capacity: 0, nativeDestinations: ['postgres', 'clickhouse'] }),
    ]

    const rows = mount(SystemPage).findAll('tbody tr').map(r => r.findAll('td').map(c => c.text()))

    expect(rows).toHaveLength(2)
    expect(rows[0]!.slice(0, 5)).toEqual(['synclet-docker', 'Docker', 'v0.5.0', '5', '—'])
    expect(rows[1]!.slice(0, 5)).toEqual(['synclet-executor-0', 'Kubernetes', 'v0.5.1', 'Unlimited', 'postgres, clickhouse'])
    expect(rows[0]![5]).toBe('just now')
  })

  it('shows an empty state when no executor has reported', () => {
    const text = mount(SystemPage).text()
    expect(text).toContain('No executors')
  })
})

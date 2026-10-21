import type { MapperTarget, ValidateMappersParams, ValidateMappersResult } from '@entities/connection'
import type { StreamRow } from '../../catalogStreams'
import { validateMappers } from '@entities/connection'
import { mount } from '@vue/test-utils'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import TransformTab from '../TransformTab.vue'

vi.mock('@entities/connection', () => ({ validateMappers: vi.fn() }))

const schema = { properties: { id: { type: 'integer' }, amount: { type: 'string' } } }

function row(mappers: Record<string, unknown>[]): StreamRow {
  return {
    name: 'orders',
    namespace: 'public',
    syncMode: 'full_refresh',
    destinationSyncMode: 'overwrite',
    enabled: true,
    supportedSyncModes: ['full_refresh'],
    cursorField: [],
    primaryKey: [['id']],
    selectedFields: [],
    jsonSchema: schema,
    sourceDefinedCursor: false,
    defaultCursorField: [],
    sourceDefinedPrimaryKey: [],
    mappers,
  }
}

function result(partial: Partial<ValidateMappersResult>): ValidateMappersResult {
  return { errors: [], warnings: [], outputSchema: undefined, outputPrimaryKey: [], sampleOutput: undefined, sampleKept: true, sampleError: '', ...partial }
}

describe('useMapperValidation', () => {
  beforeEach(() => {
    vi.useFakeTimers()
    vi.mocked(validateMappers).mockReset()
  })

  afterEach(() => {
    vi.useRealTimers()
  })

  it('checks each prefix of the steps once and shows server issues, result types and samples on their steps', async () => {
    vi.mocked(validateMappers).mockImplementation(async (_target: MapperTarget, req: ValidateMappersParams) => {
      if (req.mappers.length === 1) {
        return result({
          outputSchema: { properties: { id: { type: 'integer' }, amount: { type: 'number' }, cents: { type: 'integer' } } },
          sampleOutput: { id: 7, amount: 1.5, cents: 150 },
        })
      }
      return result({
        errors: [{ step: 1, field: '', message: 'undeclared reference to \'nope\'' }],
        sampleKept: false,
      })
    })

    const wrapper = mount(TransformTab, {
      props: { stream: row([{ set: { cents: 'int(r.amount * 100.0)' } }, { filter: 'r.nope > 0' }]), target: { connectionId: 'conn-1' } },
    })
    await wrapper.get('[data-testid="test-sample"]').setValue('{"id": 7, "amount": "1.5"}')

    await vi.advanceTimersByTimeAsync(399)
    expect(validateMappers).not.toHaveBeenCalled()
    await vi.advanceTimersByTimeAsync(1)

    expect(validateMappers).toHaveBeenCalledTimes(2)
    const requests = vi.mocked(validateMappers).mock.calls.map(c => c[1])
    expect(requests.map(r => r.mappers.length)).toEqual([1, 2])
    expect(requests[0].sampleRecord).toEqual({ id: 7, amount: '1.5' })

    const cards = wrapper.findAll('[data-testid="step-card"]')
    expect(cards[1].findAll('[data-testid="step-issue"]').map(i => i.text())).toEqual(['undeclared reference to \'nope\''])
    expect(cards[0].findAll('[data-testid="step-issue"]')).toHaveLength(0)
    expect(cards[0].get('[data-testid="result-type"]').text()).toBe('→ integer')

    const outputs = wrapper.findAll('[data-testid="test-step-output"]')
    expect(outputs).toHaveLength(2)
    expect(outputs[0].text()).toContain('"cents": 150')
    expect(outputs[1].text()).toContain('Not run: a step up to here has errors.')
  })

  it('drops a stale response', async () => {
    let resolveFirst: (r: ValidateMappersResult) => void = () => {}
    vi.mocked(validateMappers)
      .mockImplementationOnce(() => new Promise((resolve) => { resolveFirst = resolve }))
      .mockResolvedValue(result({}))

    const wrapper = mount(TransformTab, {
      props: { stream: row([{ filter: 'r.id > 0' }]), target: { connectionId: 'conn-1' } },
    })
    await vi.advanceTimersByTimeAsync(400)
    await wrapper.setProps({ stream: row([{ filter: 'r.id > 1' }]) })
    await vi.advanceTimersByTimeAsync(400)

    resolveFirst(result({ errors: [{ step: 0, field: '', message: 'stale' }] }))
    await vi.advanceTimersByTimeAsync(0)

    expect(wrapper.text()).not.toContain('stale')
  })

  it('shows a record a filter drops as filtered out', async () => {
    vi.mocked(validateMappers).mockResolvedValue(result({ sampleKept: false }))
    const wrapper = mount(TransformTab, {
      props: { stream: row([{ filter: 'r.id > 7' }]), target: { connectionId: 'conn-1' } },
    })
    await wrapper.get('[data-testid="test-sample"]').setValue('{"id": 7}')
    await vi.advanceTimersByTimeAsync(400)

    expect(wrapper.get('[data-testid="test-step-output"]').text()).toContain('Filtered out')
    expect(wrapper.get('[data-testid="result-type"]').text()).toBe('→ boolean')
  })

  it('gives an existing column the type a set step writes and marks it changed', async () => {
    vi.mocked(validateMappers).mockResolvedValue(result({
      outputSchema: { properties: { id: { type: 'integer' }, amount: { type: 'boolean' } } },
    }))
    const wrapper = mount(TransformTab, {
      props: { stream: row([{ set: { amount: 'true' } }]), target: { connectionId: 'conn-1' } },
    })
    await vi.advanceTimersByTimeAsync(400)

    expect(wrapper.get('[data-testid="result-type"]').text()).toBe('→ boolean')
    const amount = wrapper.findAll('[data-testid="output-column"]').find(c => c.text().startsWith('amount'))!
    expect(amount.text()).toContain('boolean')
    expect(amount.text()).toContain('type changed')
  })

  it('validates against a source and connection name before the connection exists', async () => {
    vi.mocked(validateMappers).mockResolvedValue(result({}))
    mount(TransformTab, {
      props: { stream: row([{ filter: 'r.id > 0' }]), target: { sourceId: 'src-1', connectionName: 'Orders' } },
    })
    await vi.advanceTimersByTimeAsync(400)

    expect(vi.mocked(validateMappers).mock.calls.map(c => c[0])).toEqual([{ sourceId: 'src-1', connectionName: 'Orders' }])
  })
})

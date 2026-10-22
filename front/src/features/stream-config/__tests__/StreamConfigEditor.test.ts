import type { StreamRow } from '../catalogStreams'
import { mount } from '@vue/test-utils'
import { describe, expect, it } from 'vitest'
import StreamConfigEditor from '../StreamConfigEditor.vue'

function row(overrides: Partial<StreamRow>): StreamRow {
  return {
    name: 'orders',
    namespace: '',
    syncMode: 'full_refresh',
    destinationSyncMode: 'overwrite',
    enabled: true,
    supportedSyncModes: ['full_refresh', 'incremental'],
    cursorField: [],
    primaryKey: [],
    selectedFields: [{ fieldPath: ['id'] }],
    jsonSchema: { properties: { id: { type: 'integer' } } },
    sourceDefinedCursor: false,
    defaultCursorField: [],
    sourceDefinedPrimaryKey: [],
    mappers: [],
    ...overrides,
  }
}

describe('stream config editor', () => {
  it('reports missing fields, cursor and primary key and marks the streams', async () => {
    const wrapper = mount(StreamConfigEditor, {
      props: {
        'streams': [
          row({ name: 'empty', selectedFields: [] }),
          row({ name: 'inc', syncMode: 'incremental', destinationSyncMode: 'append_dedup' }),
          row({ name: 'off', enabled: false, selectedFields: [] }),
        ],
        'connectionId': '',
        'onUpdate:streams': () => {},
      },
      global: { stubs: { StreamExpandedRow: true } },
    })

    const errors = (wrapper.vm as unknown as { validate: () => { streamKey: string, message: string }[] }).validate()
    await wrapper.vm.$nextTick()

    expect(errors.map(e => [e.streamKey, e.message])).toEqual([
      ['empty', 'Select at least one field for replication'],
      ['inc', 'Cursor field is required for Incremental sync'],
      ['inc', 'Primary key is required for Append + Dedup mode'],
    ])
    expect(wrapper.findAll('svg.text-danger')).toHaveLength(2)
  })
})

describe('turning streams on', () => {
  function mountEditor(streams: StreamRow[]) {
    const wrapper = mount(StreamConfigEditor, {
      props: {
        streams,
        'connectionId': '',
        'onUpdate:streams': (v: StreamRow[]) => wrapper.setProps({ streams: v }),
      },
      global: { stubs: { StreamExpandedRow: true } },
    })
    return wrapper
  }

  const known = { sourceDefinedPrimaryKey: [['id']], defaultCursorField: ['id'] }

  async function toggle(wrapper: ReturnType<typeof mountEditor>, name: string, on: boolean) {
    await wrapper.find(`input[aria-label="Enable stream ${name}"]`).setValue(on)
  }

  function current(wrapper: ReturnType<typeof mountEditor>): StreamRow {
    return (wrapper.props() as { streams: StreamRow[] }).streams[0]
  }

  it('applies the defaults to a stream that was never configured', async () => {
    const wrapper = mountEditor([row({ enabled: false, selectedFields: [], ...known })])
    await toggle(wrapper, 'orders', true)
    expect(current(wrapper)).toMatchObject({ enabled: true, syncMode: 'incremental', destinationSyncMode: 'append_dedup' })
  })

  it('keeps the saved configuration when a saved stream is turned off and on', async () => {
    const wrapper = mountEditor([row({ configured: true, ...known })])
    await toggle(wrapper, 'orders', false)
    await toggle(wrapper, 'orders', true)
    expect(current(wrapper)).toMatchObject({ enabled: true, syncMode: 'full_refresh', destinationSyncMode: 'overwrite', selectedFields: [{ fieldPath: ['id'] }] })
  })

  it('keeps the edits of this session when an edited stream is turned off and on', async () => {
    const wrapper = mountEditor([row({ enabled: false, selectedFields: [], ...known })])
    await toggle(wrapper, 'orders', true)
    await wrapper.find('select').setValue('full_refresh|append')
    await toggle(wrapper, 'orders', false)
    await toggle(wrapper, 'orders', true)
    expect(current(wrapper)).toMatchObject({ enabled: true, syncMode: 'full_refresh', destinationSyncMode: 'append' })
  })
})

describe('namespace sections', () => {
  function mountEditor(streams: StreamRow[]) {
    const wrapper = mount(StreamConfigEditor, {
      props: {
        streams,
        'connectionId': '',
        'onUpdate:streams': (v: StreamRow[]) => wrapper.setProps({ streams: v }),
      },
      global: { stubs: { StreamExpandedRow: true } },
    })
    return wrapper
  }

  function enabledNames(wrapper: ReturnType<typeof mountEditor>): string[] {
    return (wrapper.props() as { streams: StreamRow[] }).streams.filter(s => s.enabled).map(s => `${s.namespace}.${s.name}`)
  }

  function namespaceCheckbox(wrapper: ReturnType<typeof mountEditor>, label: string) {
    return wrapper.find(`input[aria-label="Enable all streams of namespace ${label}"]`)
  }

  it('shows a section with name, enabled count and a checkbox also for a single namespace', async () => {
    const wrapper = mountEditor([
      row({ name: 'a', namespace: 'public' }),
      row({ name: 'b', namespace: 'public', enabled: false }),
    ])
    const header = wrapper.find('[data-testid="namespace-public"]')
    expect(header.exists()).toBe(true)
    expect(header.text()).toContain('public')
    expect(header.text()).toContain('1/2')
    const box = namespaceCheckbox(wrapper, 'public').element as HTMLInputElement
    expect(box.checked).toBe(false)
    expect(box.indeterminate).toBe(true)

    await header.find('button[aria-expanded]').trigger('click')
    expect(wrapper.find('input[aria-label="Enable stream a"]').exists()).toBe(false)
  })

  it('turns every stream of a namespace on and off', async () => {
    const wrapper = mountEditor([
      row({ name: 'a', namespace: 'public', enabled: false }),
      row({ name: 'b', namespace: 'public', enabled: false }),
      row({ name: 'c', namespace: 'sales', enabled: false }),
    ])
    await namespaceCheckbox(wrapper, 'public').setValue(true)
    expect(enabledNames(wrapper)).toEqual(['public.a', 'public.b'])
    expect((namespaceCheckbox(wrapper, 'public').element as HTMLInputElement).checked).toBe(true)
    expect((namespaceCheckbox(wrapper, 'public').element as HTMLInputElement).indeterminate).toBe(false)
    await namespaceCheckbox(wrapper, 'public').setValue(false)
    expect(enabledNames(wrapper)).toEqual([])
  })

  it('acts only on the streams the search shows', async () => {
    const wrapper = mountEditor([
      row({ name: 'orders', namespace: 'public', enabled: false }),
      row({ name: 'users', namespace: 'public', enabled: false }),
      row({ name: 'orders_archive', namespace: 'sales', enabled: false }),
      row({ name: 'leads', namespace: 'sales', enabled: false }),
    ])
    await wrapper.find('input[placeholder="Search streams..."]').setValue('orders')

    await wrapper.find('input[aria-label="Enable all streams"]').setValue(true)
    expect(enabledNames(wrapper)).toEqual(['public.orders', 'sales.orders_archive'])

    await wrapper.find('input[aria-label="Enable all streams"]').setValue(false)
    expect(enabledNames(wrapper)).toEqual([])

    await namespaceCheckbox(wrapper, 'public').setValue(true)
    expect(enabledNames(wrapper)).toEqual(['public.orders'])
  })
})

describe('bulk sync mode', () => {
  function mountEditor(streams: StreamRow[]) {
    const wrapper = mount(StreamConfigEditor, {
      props: {
        streams,
        'connectionId': '',
        'onUpdate:streams': (v: StreamRow[]) => wrapper.setProps({ streams: v }),
      },
      global: { stubs: { StreamExpandedRow: true } },
    })
    return wrapper
  }

  it('sets one sync mode on the selected streams and reports the ones that do not support it', async () => {
    const wrapper = mountEditor([
      row({ name: 'a' }),
      row({ name: 'b', supportedSyncModes: ['full_refresh'] }),
      row({ name: 'c', enabled: false }),
      row({ name: 'd' }),
    ])
    await wrapper.find('select[aria-label="Sync mode of selected streams"]').setValue('incremental|append')

    const modes = (wrapper.props() as { streams: StreamRow[] }).streams.map(s => [s.name, `${s.syncMode}|${s.destinationSyncMode}`])
    expect(modes).toEqual([
      ['a', 'incremental|append'],
      ['b', 'full_refresh|overwrite'],
      ['c', 'full_refresh|overwrite'],
      ['d', 'incremental|append'],
    ])
    expect(wrapper.find('[role="status"]').text()).toContain('1 stream skipped')
  })
})

describe('expanding and collapsing namespaces', () => {
  function mountEditor() {
    return mount(StreamConfigEditor, {
      props: {
        'streams': [
          row({ name: 'orders', namespace: 'sales' }),
          row({ name: 'users', namespace: 'crm' }),
        ],
        'connectionId': '',
        'onUpdate:streams': () => {},
      },
      global: { stubs: { StreamExpandedRow: true } },
    })
  }

  const streamInput = (wrapper: ReturnType<typeof mountEditor>, name: string) =>
    wrapper.find(`input[aria-label="Enable stream ${name}"]`)
  const button = (wrapper: ReturnType<typeof mountEditor>, label: string) =>
    wrapper.find(`button[aria-label="${label}"]`)

  it('collapse all hides the streams of every namespace and closes the open stream', async () => {
    const wrapper = mountEditor()
    await button(wrapper, 'Expand stream orders').trigger('click')
    expect(wrapper.find('stream-expanded-row-stub').exists()).toBe(true)

    await button(wrapper, 'Collapse all').trigger('click')

    expect(streamInput(wrapper, 'orders').exists()).toBe(false)
    expect(streamInput(wrapper, 'users').exists()).toBe(false)
    expect(wrapper.find('stream-expanded-row-stub').exists()).toBe(false)
    expect(button(wrapper, 'Collapse all').attributes('disabled')).toBeDefined()
    expect(button(wrapper, 'Expand all').attributes('disabled')).toBeUndefined()
  })

  it('expand all shows the streams of every namespace', async () => {
    const wrapper = mountEditor()
    await button(wrapper, 'Collapse all').trigger('click')

    await button(wrapper, 'Expand all').trigger('click')

    expect(streamInput(wrapper, 'orders').exists()).toBe(true)
    expect(streamInput(wrapper, 'users').exists()).toBe(true)
    expect(button(wrapper, 'Expand all').attributes('disabled')).toBeDefined()
    expect(button(wrapper, 'Collapse all').attributes('disabled')).toBeUndefined()
  })
})

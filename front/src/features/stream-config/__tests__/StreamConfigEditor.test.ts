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
    return (wrapper.props('streams') as StreamRow[])[0]
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

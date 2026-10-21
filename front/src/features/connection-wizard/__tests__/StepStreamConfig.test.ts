import StreamConfigEditor from '@features/stream-config/StreamConfigEditor.vue'
import { flushPromises, mount } from '@vue/test-utils'
import { describe, expect, it, vi } from 'vitest'
import { ref } from 'vue'
import StepStreamConfig from '../StepStreamConfig.vue'

vi.mock('@entities/connector-task', () => ({
  useConnectorTaskResult: () => ({ data: ref(null), error: ref(null), timedOut: ref(false) }),
}))
vi.mock('@entities/source', () => ({
  discoverSourceSchema: vi.fn(),
  getSourceCatalog: vi.fn(),
}))

const userFields = ['id', 'name', 'email', 'title', 'age', 'created_at', 'updated_at']
const properties = Object.fromEntries(userFields.map(f => [f, { type: 'string' }]))

const catalog = {
  streams: [
    {
      stream: {
        name: 'users',
        json_schema: { type: 'object', properties },
        supported_sync_modes: ['full_refresh', 'incremental'],
        source_defined_primary_key: [['id']],
        default_cursor_field: ['updated_at'],
      },
    },
    {
      stream: {
        name: 'events',
        json_schema: { type: 'object', properties },
        supported_sync_modes: ['incremental'],
      },
    },
  ],
}

function mountStep(streams: unknown[] = []) {
  return mount(StepStreamConfig, {
    props: { sourceId: 'src-1', streams: streams as never, discoveredCatalog: catalog },
    global: { stubs: { StreamExpandedRow: true } },
  })
}

function rowOf(wrapper: ReturnType<typeof mountStep>, name: string) {
  return wrapper.find(`input[aria-label="Enable stream ${name}"]`).element.closest('tr')!
}

describe('wizard streams step', () => {
  it('edits streams with the shared stream editor, including the sync mode', async () => {
    const wrapper = mountStep()
    await flushPromises()

    expect(wrapper.findComponent(StreamConfigEditor).exists()).toBe(true)
    expect(rowOf(wrapper, 'users').querySelector('select')).not.toBeNull()
  })

  it('turns a stream on with all fields, the source key and cursor, and the best sync mode', async () => {
    const wrapper = mountStep()
    await flushPromises()

    await wrapper.find('input[aria-label="Enable stream users"]').setValue(true)

    const row = rowOf(wrapper, 'users')
    expect(row.textContent).toContain('7/7 fields')
    expect(row.querySelector('select')!.value).toBe('incremental|append_dedup')
    const emitted = wrapper.emitted('update:streams')!.at(-1)![0] as Array<Record<string, unknown>>
    expect(emitted).toEqual([expect.objectContaining({
      streamName: 'users',
      syncMode: 'incremental',
      destinationSyncMode: 'append_dedup',
      cursorField: ['updated_at'],
      primaryKey: [],
      selectedFields: [],
    })])
  })

  it('blocks going on while an enabled stream is invalid, with the editor messages', async () => {
    const wrapper = mountStep()
    await flushPromises()
    await wrapper.find('input[aria-label="Enable stream events"]').setValue(true)

    const valid = (wrapper.vm as unknown as { validate: () => boolean }).validate()
    await flushPromises()

    expect(valid).toBe(false)
    expect(wrapper.text()).toContain('Fix 1 issue(s) before continuing: events: Cursor field is required for Incremental sync')
  })

  it('lets a valid configuration go on', async () => {
    const wrapper = mountStep()
    await flushPromises()
    await wrapper.find('input[aria-label="Enable stream users"]').setValue(true)

    expect((wrapper.vm as unknown as { validate: () => boolean }).validate()).toBe(true)
  })

  it('shows a restored stream without a field selection as all fields selected', async () => {
    const wrapper = mountStep([{
      streamName: 'users',
      namespace: '',
      syncMode: 'full_refresh',
      destinationSyncMode: 'overwrite',
      cursorField: [],
      primaryKey: [],
      selectedFields: [],
    }])
    await flushPromises()

    expect(rowOf(wrapper, 'users').textContent).toContain('7/7 fields')
  })
})

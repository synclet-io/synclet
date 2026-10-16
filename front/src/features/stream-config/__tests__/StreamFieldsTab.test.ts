import type { StreamRow } from '../StreamFieldsTab.vue'
import { mount } from '@vue/test-utils'
import { describe, expect, it } from 'vitest'
import FieldTree from '../FieldTree.vue'
import StreamFieldsTab from '../StreamFieldsTab.vue'

const schema = {
  properties: {
    'id': { type: 'integer' },
    'user.email': { type: 'string' },
    'user': { type: 'object', properties: { email: { type: 'string' } } },
  },
}

function row(partial: Partial<StreamRow> = {}): StreamRow {
  return {
    name: 'users',
    namespace: '',
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
    ...partial,
  }
}

function mountTab(stream: StreamRow) {
  return mount(StreamFieldsTab, { props: { stream, schema, validationErrors: [] } })
}

function emittedPaths(wrapper: ReturnType<typeof mountTab>): string[][] {
  const events = wrapper.emitted('update:stream') as [StreamRow][]
  return events.at(-1)![0].selectedFields.map(f => f.fieldPath)
}

describe('streamFieldsTab', () => {
  it('keeps a dotted field name as a single path segment', async () => {
    const wrapper = mountTab(row())
    await wrapper.findComponent(FieldTree).vm.$emit('toggle', ['user.email'], true)
    expect(emittedPaths(wrapper)).toEqual(expect.arrayContaining([['user.email'], ['id']]))
    expect(emittedPaths(wrapper)).not.toContainEqual(['user', 'email'])
  })

  it('distinguishes a dotted name from the nested path it resembles', async () => {
    const wrapper = mountTab(row({ selectedFields: [{ fieldPath: ['user.email'] }] }))
    await wrapper.findComponent(FieldTree).vm.$emit('toggle', ['user', 'email'], true)
    expect(emittedPaths(wrapper)).toEqual(expect.arrayContaining([['user.email'], ['user', 'email'], ['id']]))
  })

  it('select all emits every leaf path unchanged', async () => {
    const wrapper = mountTab(row())
    await wrapper.findComponent(FieldTree).vm.$emit('selectAll')
    expect(emittedPaths(wrapper)).toHaveLength(3)
    expect(emittedPaths(wrapper)).toEqual(expect.arrayContaining([['id'], ['user.email'], ['user', 'email']]))
  })
})

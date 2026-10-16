import type { DiscoveredCatalog } from '../catalogStreams'
import type { StreamRow } from '../StreamFieldsTab.vue'
import { describe, expect, it } from 'vitest'
import { mergeDiscoveredStreams, parseStreams } from '../catalogStreams'

function discovered(...names: Array<string | [string, string]>): DiscoveredCatalog {
  return {
    streams: names.map((n) => {
      const [namespace, name] = Array.isArray(n) ? n : ['', n]
      return {
        stream: {
          name,
          namespace,
          json_schema: { properties: { id: { type: 'integer' }, updated_at: { type: 'string' } } },
          supported_sync_modes: ['full_refresh', 'incremental'],
        },
      }
    }),
  }
}

describe('parseStreams', () => {
  it('enables every stream when nothing is configured', () => {
    const rows = parseStreams(discovered('users', 'orders'))
    expect(rows.map(r => [r.name, r.enabled, r.syncMode])).toEqual([
      ['users', true, 'full_refresh'],
      ['orders', true, 'full_refresh'],
    ])
  })

  it('applies the configured catalog and disables unconfigured streams', () => {
    const rows = parseStreams(discovered('users', ['public', 'orders']), {
      streams: [{
        stream: { name: 'orders', namespace: 'public' },
        sync_mode: 'incremental',
        destination_sync_mode: 'append_dedup',
        cursor_field: ['updated_at'],
        primary_key: [['id']],
        selected_fields: [{ field_path: ['id'] }],
      }],
    })
    const orders = rows.find(r => r.name === 'orders')!
    expect(orders).toMatchObject({
      enabled: true,
      syncMode: 'incremental',
      destinationSyncMode: 'append_dedup',
      cursorField: ['updated_at'],
      primaryKey: [['id']],
      selectedFields: [{ fieldPath: ['id'] }],
    })
    expect(rows.find(r => r.name === 'users')!.enabled).toBe(false)
  })
})

describe('mergeDiscoveredStreams', () => {
  function configured(): StreamRow[] {
    return parseStreams(discovered('users', 'orders', 'legacy'), {
      streams: [
        { name: 'orders', syncMode: 'incremental', destinationSyncMode: 'append', cursorField: ['updated_at'], primaryKey: [{ fieldPath: ['id'] }], selectedFields: [{ fieldPath: ['id'] }] },
        { name: 'legacy', syncMode: 'full_refresh', destinationSyncMode: 'overwrite' },
      ],
    })
  }

  it('keeps sync modes, cursors, keys, field selection and enabled flags of existing streams', () => {
    const fresh = parseStreams(discovered('users', 'orders', 'legacy'))
    const { streams, added, removed } = mergeDiscoveredStreams(configured(), fresh)

    expect(added).toEqual([])
    expect(removed).toEqual([])
    expect(streams.find(s => s.name === 'users')!.enabled).toBe(false)
    expect(streams.find(s => s.name === 'orders')).toMatchObject({
      enabled: true,
      syncMode: 'incremental',
      destinationSyncMode: 'append',
      cursorField: ['updated_at'],
      primaryKey: [['id']],
      selectedFields: [{ fieldPath: ['id'] }],
    })
  })

  it('takes schema metadata from the new discovery', () => {
    const fresh = parseStreams({
      streams: [{
        stream: { name: 'orders', json_schema: { properties: { id: {}, total: {} } }, supported_sync_modes: ['incremental'], source_defined_cursor: true },
      }],
    })
    const { streams } = mergeDiscoveredStreams(configured(), fresh)
    expect(streams[0]).toMatchObject({
      jsonSchema: { properties: { id: {}, total: {} } },
      supportedSyncModes: ['incremental'],
      sourceDefinedCursor: true,
      syncMode: 'incremental',
      cursorField: ['updated_at'],
    })
  })

  it('reports added and removed streams and leaves new streams disabled', () => {
    const fresh = parseStreams(discovered('users', 'orders', ['crm', 'invoices']))
    const { streams, added, removed } = mergeDiscoveredStreams(configured(), fresh)

    expect(added).toEqual(['crm.invoices'])
    expect(removed).toEqual(['legacy'])
    expect(streams.find(s => s.name === 'invoices')!.enabled).toBe(false)
  })

  it('enables everything when there was nothing listed before', () => {
    const { streams, added } = mergeDiscoveredStreams([], parseStreams(discovered('users')))
    expect(added).toEqual(['users'])
    expect(streams[0].enabled).toBe(true)
  })
})

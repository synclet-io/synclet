import { describe, expect, it } from 'vitest'
import { buildConnectionListQuery, parseConnectionListQuery } from '../listQuery'

const SRC = '11111111-1111-4111-8111-111111111111'
const DST = '22222222-2222-4222-8222-222222222222'

describe('parseConnectionListQuery', () => {
  it('returns defaults for an empty query', () => {
    expect(parseConnectionListQuery({})).toEqual({ sourceId: '', destinationId: '', page: 1 })
  })

  it('reads filters and page', () => {
    expect(parseConnectionListQuery({ sourceId: SRC, destinationId: DST, page: '3' }))
      .toEqual({ sourceId: SRC, destinationId: DST, page: 3 })
  })

  it('ignores malformed ids and pages', () => {
    expect(parseConnectionListQuery({ sourceId: 'nope', destinationId: [DST, SRC], page: '-2' }))
      .toEqual({ sourceId: '', destinationId: DST, page: 1 })
  })
})

describe('buildConnectionListQuery', () => {
  it('resets the page when a filter changes and keeps unrelated params', () => {
    expect(buildConnectionListQuery({ page: '4', tab: 'x' }, { sourceId: SRC }))
      .toEqual({ tab: 'x', sourceId: SRC })
  })

  it('keeps filters when paging', () => {
    expect(buildConnectionListQuery({ sourceId: SRC, destinationId: DST }, { page: 2 }))
      .toEqual({ sourceId: SRC, destinationId: DST, page: '2' })
  })

  it('drops cleared filters', () => {
    expect(buildConnectionListQuery({ sourceId: SRC, destinationId: DST, page: '2' }, { sourceId: '', destinationId: '' }))
      .toEqual({})
  })
})

import { describe, expect, it } from 'vitest'
import { buildConnectionListQuery, parseConnectionListQuery } from '../listQuery'

const SRC = '11111111-1111-4111-8111-111111111111'
const DST = '22222222-2222-4222-8222-222222222222'

describe('parseConnectionListQuery', () => {
  it('returns defaults for an empty query', () => {
    expect(parseConnectionListQuery({})).toEqual({ sourceId: '', destinationId: '', tags: [], page: 1 })
  })

  it('reads filters and page', () => {
    expect(parseConnectionListQuery({ sourceId: SRC, destinationId: DST, page: '3' }))
      .toEqual({ sourceId: SRC, destinationId: DST, tags: [], page: 3 })
  })

  it('ignores malformed ids and pages', () => {
    expect(parseConnectionListQuery({ sourceId: 'nope', destinationId: [DST, SRC], page: '-2' }))
      .toEqual({ sourceId: '', destinationId: DST, tags: [], page: 1 })
  })
})

describe('parseConnectionListQuery tags', () => {
  it('lowercases, dedupes, sorts and drops invalid tags', () => {
    expect(parseConnectionListQuery({ tags: ['Prod', 'eu', 'bad!', 'prod'] }).tags).toEqual(['eu', 'prod'])
  })

  it('reads a single string value', () => {
    expect(parseConnectionListQuery({ tags: 'eu' }).tags).toEqual(['eu'])
  })
})

describe('buildConnectionListQuery', () => {
  it('writes tags as an array and resets the page', () => {
    expect(buildConnectionListQuery({ page: '3', sourceId: SRC }, { tags: ['prod', 'eu'] }))
      .toEqual({ sourceId: SRC, tags: ['eu', 'prod'] })
  })

  it('drops the tags key for an empty list', () => {
    expect(buildConnectionListQuery({ tags: ['prod'], page: '2' }, { tags: [] }))
      .toEqual({})
  })

  it('keeps tags when paging', () => {
    expect(buildConnectionListQuery({ tags: 'prod' }, { page: 2 }))
      .toEqual({ tags: ['prod'], page: '2' })
  })

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

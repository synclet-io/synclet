import { describe, expect, it } from 'vitest'
import { matchScore, rankMatches } from '../matcher'

describe('matchScore', () => {
  it('returns matched=true with score 0 for empty query (everything matches)', () => {
    const r = matchScore('Anything', '')
    expect(r.matched).toBe(true)
    expect(r.score).toBe(0)
  })

  it('is case-insensitive', () => {
    expect(matchScore('PostgreSQL Source', 'post').matched).toBe(true)
    expect(matchScore('PostgreSQL Source', 'POST').matched).toBe(true)
  })

  it('returns matched=false when query is not a substring', () => {
    expect(matchScore('users', 'orders').matched).toBe(false)
    expect(matchScore('', 'foo').matched).toBe(false)
  })

  it('scores full prefix higher than mid-string match', () => {
    const prefix = matchScore('postgres', 'post')
    const middle = matchScore('snappy postgres', 'post')
    expect(prefix.score).toBeGreaterThan(middle.score)
  })

  it('scores token-prefix higher than non-boundary mid-string match', () => {
    // "post" starts a token in "my-postgres-prod" (after the dash)
    const tokenPrefix = matchScore('my-postgres-prod', 'post')
    // "ost" is mid-string in "compose" with no token boundary
    const buried = matchScore('compose', 'ost')
    expect(tokenPrefix.score).toBeGreaterThan(buried.score)
  })
})

describe('rankMatches', () => {
  it('returns items in descending score order', () => {
    const items = [
      { value: 1, text: 'analytics db' },
      { value: 2, text: 'postgres prod' },
      { value: 3, text: 'compose pipeline' }, // contains "ost" mid-string
    ]
    const result = rankMatches(items, 'post')
    expect(result).toEqual([2])
  })

  it('respects the limit parameter', () => {
    const items = Array.from({ length: 10 }, (_, i) => ({ value: i, text: `node-${i}` }))
    expect(rankMatches(items, 'node', 3)).toHaveLength(3)
  })

  it('returns empty when nothing matches', () => {
    const items = [{ value: 1, text: 'foo' }]
    expect(rankMatches(items, 'xyz')).toEqual([])
  })
})

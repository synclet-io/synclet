import { describe, expect, it } from 'vitest'
import { isNavActive } from '../isNavActive'

describe('isNavActive', () => {
  it('highlights the dashboard only on the root path', () => {
    expect(isNavActive('/', '/')).toBe(true)
    expect(isNavActive('/sources', '/')).toBe(false)
  })

  it('highlights a section on its index and detail pages', () => {
    expect(isNavActive('/sources', '/sources')).toBe(true)
    expect(isNavActive('/sources/123', '/sources')).toBe(true)
    expect(isNavActive('/connections/abc/streams', '/connections')).toBe(true)
  })

  it('highlights settings on its sub-pages', () => {
    expect(isNavActive('/settings/general', '/settings')).toBe(true)
  })

  it('does not match a section that only shares a prefix', () => {
    expect(isNavActive('/sourcesx', '/sources')).toBe(false)
    expect(isNavActive('/destinations', '/sources')).toBe(false)
  })
})

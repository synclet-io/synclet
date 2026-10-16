import { Code, ConnectError } from '@connectrpc/connect'
import { describe, expect, it } from 'vitest'
import { getErrorMessage, requireField } from '../errorUtils'

describe('getErrorMessage', () => {
  it('returns the raw message of a ConnectError without the code prefix', () => {
    const err = new ConnectError('name is required', Code.InvalidArgument)
    expect(err.message).toBe('[invalid_argument] name is required')
    expect(getErrorMessage(err)).toBe('name is required')
  })

  it('maps PermissionDenied and Unavailable to friendly copy', () => {
    expect(getErrorMessage(new ConnectError('', Code.PermissionDenied))).toMatch(/permission/)
    expect(getErrorMessage(new ConnectError('dial tcp: refused', Code.Unavailable))).toMatch(/unreachable/)
  })

  it('falls back when a ConnectError has no message', () => {
    expect(getErrorMessage(new ConnectError('', Code.Internal))).toBe('An unexpected error occurred')
  })

  it('handles plain errors, strings and unknown values', () => {
    expect(getErrorMessage(new Error('boom'))).toBe('boom')
    expect(getErrorMessage('text')).toBe('text')
    expect(getErrorMessage({ message: 'obj' })).toBe('obj')
    expect(getErrorMessage(42)).toBe('An unexpected error occurred')
  })
})

describe('requireField', () => {
  it('returns present values and throws a clear error for missing ones', () => {
    expect(requireField(0, 'count')).toBe(0)
    expect(() => requireField(undefined, 'connection')).toThrow('The server response is missing connection')
  })
})

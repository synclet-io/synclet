import { Code, ConnectError } from '@connectrpc/connect'
import { describe, expect, it } from 'vitest'
import { tagFieldError } from '../tagError'

describe('tagFieldError', () => {
  it('returns the tag message of an InvalidArgument tags error', () => {
    expect(tagFieldError(new ConnectError('validation error on tags: too many tags: 21 (max 20)', Code.InvalidArgument)))
      .toBe('too many tags: 21 (max 20)')
  })

  it('ignores a validation error on another field', () => {
    expect(tagFieldError(new ConnectError('validation error on schedule: bad cron', Code.InvalidArgument))).toBe('')
  })

  it('ignores other codes and non-Connect errors', () => {
    expect(tagFieldError(new ConnectError('validation error on tags: x', Code.PermissionDenied))).toBe('')
    expect(tagFieldError(new Error('validation error on tags: x'))).toBe('')
  })
})

import { Code, ConnectError } from '@connectrpc/connect'

const TAGS_PREFIX = 'validation error on tags: '

/** Returns the server's message for an invalid tags field, or '' when the error is about anything else. */
export function tagFieldError(e: unknown): string {
  if (e instanceof ConnectError && e.code === Code.InvalidArgument && e.rawMessage.startsWith(TAGS_PREFIX))
    return e.rawMessage.slice(TAGS_PREFIX.length)
  return ''
}

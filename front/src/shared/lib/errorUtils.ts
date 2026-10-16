import { Code, ConnectError } from '@connectrpc/connect'

const FALLBACK_MESSAGE = 'An unexpected error occurred'

const friendlyByCode: Partial<Record<Code, string>> = {
  [Code.PermissionDenied]: 'You don\'t have permission to perform this action.',
  [Code.Unavailable]: 'The server is unreachable. Check your connection and try again.',
}

export function getErrorMessage(error: unknown): string {
  if (error instanceof ConnectError)
    return friendlyByCode[error.code] ?? (error.rawMessage || FALLBACK_MESSAGE)
  if (error instanceof Error)
    return error.message
  if (typeof error === 'string')
    return error
  if (error && typeof error === 'object' && 'message' in error
    && typeof (error as { message: unknown }).message === 'string') {
    return (error as { message: string }).message
  }
  return FALLBACK_MESSAGE
}

export function requireField<T>(value: T | undefined | null, name: string): T {
  if (value === undefined || value === null)
    throw new Error(`The server response is missing ${name}`)
  return value
}

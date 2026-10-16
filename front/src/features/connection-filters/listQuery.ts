import type { LocationQuery, LocationQueryRaw } from 'vue-router'

export interface ConnectionListQuery {
  sourceId: string
  destinationId: string
  page: number
}

const UUID_PATTERN = /^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$/i

function single(value: LocationQuery[string] | undefined): string {
  const first = Array.isArray(value) ? value[0] : value
  return typeof first === 'string' ? first : ''
}

function uuidOrEmpty(value: string): string {
  return UUID_PATTERN.test(value) ? value : ''
}

export function parseConnectionListQuery(query: LocationQuery): ConnectionListQuery {
  const page = Number.parseInt(single(query.page), 10)
  return {
    sourceId: uuidOrEmpty(single(query.sourceId)),
    destinationId: uuidOrEmpty(single(query.destinationId)),
    page: Number.isInteger(page) && page > 1 ? page : 1,
  }
}

/**
 * Merges a patch into the current route query. Changing a filter resets
 * pagination unless the patch sets the page itself; default values are
 * dropped so the URL stays minimal.
 */
export function buildConnectionListQuery(current: LocationQuery, patch: Partial<ConnectionListQuery>): LocationQueryRaw {
  const next = { ...parseConnectionListQuery(current), ...patch }
  if (patch.page === undefined && ('sourceId' in patch || 'destinationId' in patch))
    next.page = 1

  const query: LocationQueryRaw = { ...current }
  delete query.sourceId
  delete query.destinationId
  delete query.page
  if (next.sourceId)
    query.sourceId = next.sourceId
  if (next.destinationId)
    query.destinationId = next.destinationId
  if (next.page > 1)
    query.page = String(next.page)
  return query
}

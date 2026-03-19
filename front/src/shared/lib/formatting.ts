import type { Timestamp } from '@bufbuild/protobuf/wkt'

/** Convert a protobuf Timestamp to a JS Date. Returns undefined if input is nil. */
export function tsToDate(ts: Timestamp | undefined): Date | undefined {
  if (!ts)
    return undefined
  return new Date(Number(ts.seconds) * 1000 + Math.floor(ts.nanos / 1_000_000))
}

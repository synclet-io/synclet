/** Format a large number with K/M suffix. "1.2M", "45.6K", or raw number if under 1000. */
export function formatNumber(n: number): string {
  if (n >= 1_000_000) {
    const val = n / 1_000_000
    return `${val % 1 === 0 ? val.toFixed(0) : val.toFixed(1)}M`
  }
  if (n >= 1_000) {
    const val = n / 1_000
    return `${val % 1 === 0 ? val.toFixed(0) : val.toFixed(1)}K`
  }
  return String(n)
}

/** Format duration in ms. "-" for undefined, "Xms" for sub-second, "X.Xs" for < 60s, "Xm Ys" for >= 60s. */
export function formatDuration(ms: number | undefined): string {
  if (ms == null || ms === 0)
    return '-'
  if (ms < 1000)
    return `${ms}ms`
  if (ms < 60000)
    return `${(ms / 1000).toFixed(1)}s`
  return `${Math.floor(ms / 60000)}m ${Math.round((ms % 60000) / 1000)}s`
}

/** Format a percentage with 1 decimal place. "98.5%" */
export function formatPercent(n: number): string {
  return `${n.toFixed(1)}%`
}

/** Format a Date as relative time. "2 minutes ago", "1 hour ago", "3 days ago". */
export function formatRelativeTime(date: Date): string {
  const now = new Date()
  const diffMs = now.getTime() - date.getTime()
  const diffSeconds = Math.floor(diffMs / 1000)
  const diffMinutes = Math.floor(diffSeconds / 60)
  const diffHours = Math.floor(diffMinutes / 60)
  const diffDays = Math.floor(diffHours / 24)

  if (diffDays > 0)
    return diffDays === 1 ? '1 day ago' : `${diffDays} days ago`
  if (diffHours > 0)
    return diffHours === 1 ? '1 hour ago' : `${diffHours} hours ago`
  if (diffMinutes > 0)
    return diffMinutes === 1 ? '1 minute ago' : `${diffMinutes} minutes ago`
  return 'just now'
}

/** Map a status string to a badge variant. */
export function statusVariant(status: string): 'success' | 'danger' | 'warning' | 'gray' {
  const map: Record<string, 'success' | 'danger' | 'warning' | 'gray'> = {
    active: 'success',
    completed: 'success',
    running: 'success',
    failed: 'danger',
    error: 'danger',
    scheduled: 'warning',
    starting: 'warning',
    paused: 'warning',
  }
  return map[status.toLowerCase()] || 'gray'
}

/** Trend format: "^ 15% vs prev period" or "v 12% vs prev period" or "-- No change". */
export function formatTrend(delta: number): { text: string, color: 'green' | 'red' | 'gray' } {
  if (delta === 0)
    return { text: '-- No change', color: 'gray' }
  if (delta > 0)
    return { text: `^ ${Math.abs(delta).toFixed(1)}% vs prev period`, color: 'green' }
  return { text: `v ${Math.abs(delta).toFixed(1)}% vs prev period`, color: 'red' }
}

import cronstrue from 'cronstrue'

const WHITESPACE_RE = /\s+/

/**
 * Validates a 5-field cron expression (minute hour dom month dow).
 * Returns an error message or null if valid.
 */
export function validateCron(expr: string): string | null {
  if (!expr)
    return null

  const parts = expr.trim().split(WHITESPACE_RE)
  if (parts.length !== 5) {
    return 'Cron expression must have 5 fields: minute hour day-of-month month day-of-week'
  }

  try {
    cronstrue.toString(expr)
    return null
  }
  catch {
    return 'Invalid cron expression'
  }
}

/**
 * Returns a human-readable description of a cron expression, or null if invalid/empty.
 */
export function describeCron(expr: string): string | null {
  if (!expr)
    return null

  try {
    return cronstrue.toString(expr, { use24HourTimeFormat: true })
  }
  catch {
    return null
  }
}

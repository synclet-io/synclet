export interface MatchResult {
  matched: boolean
  /** Higher = better match. 0 when matched=false. */
  score: number
}

const TOKEN_PREFIX_BONUS = 20
const FULL_PREFIX_BONUS = 50
const TOKEN_RE = /[\s\-_./]+/

export function matchScore(text: string, query: string): MatchResult {
  if (!query)
    return { matched: true, score: 0 }
  if (!text)
    return { matched: false, score: 0 }

  const t = text.toLowerCase()
  const q = query.toLowerCase()

  const idx = t.indexOf(q)
  if (idx === -1)
    return { matched: false, score: 0 }

  let score = 100 - idx // earlier match wins

  if (idx === 0)
    score += FULL_PREFIX_BONUS

  // Token-prefix bonus: q starts a token in t (boundary or beginning).
  for (const token of t.split(TOKEN_RE)) {
    if (token.startsWith(q)) {
      score += TOKEN_PREFIX_BONUS
      break
    }
  }

  return { matched: true, score }
}

export interface Searchable<T> {
  value: T
  text: string
}

export function rankMatches<T>(items: Searchable<T>[], query: string, limit?: number): T[] {
  const matched: Array<{ value: T, score: number }> = []
  for (const item of items) {
    const r = matchScore(item.text, query)
    if (r.matched)
      matched.push({ value: item.value, score: r.score })
  }
  matched.sort((a, b) => b.score - a.score)
  const out = matched.map(m => m.value)
  return limit ? out.slice(0, limit) : out
}

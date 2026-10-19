import type { Completion, CompletionContext, CompletionResult, CompletionSource } from '@codemirror/autocomplete'
import type { EditorView } from '@codemirror/view'
import type { Column } from '../columns'
import functions from './functions.json'
import { celContextAt } from './language'

interface CelFunction {
  name: string
  signature: string
  detail: string
  method?: boolean
}

const celFunctions: CelFunction[] = functions

export const SYNC_FIELDS: Column[] = [
  { name: 'started_at', type: 'timestamp' },
  { name: 'job_id', type: 'string' },
  { name: 'connection', type: 'string' },
]

function insertCall(view: EditorView, completion: Completion, from: number, to: number) {
  const name = completion.label
  view.dispatch({
    changes: { from, to, insert: `${name}()` },
    selection: { anchor: from + name.length + 1 },
  })
}

function functionOption(fn: CelFunction): Completion {
  return { label: fn.name, type: fn.method ? 'method' : 'function', detail: fn.signature, info: fn.detail, apply: insertCall }
}

const allFunctionOptions = celFunctions.map(functionOption)
const methodOptions = celFunctions.filter(fn => fn.method).map(functionOption)
const rootOptions: Completion[] = [
  { label: 'r', type: 'variable', detail: 'record' },
  { label: 'sync', type: 'variable', detail: 'sync run' },
]

function fieldOptions(columns: Column[]): Completion[] {
  return columns.map(c => ({ label: c.name, type: 'property', detail: c.type }))
}

const word = /[A-Z_]\w*$/i
const trailingDot = /\.\s*$/
const numberEnd = /(?:^|\W)\d+$/
const digitEnd = /\d$/
const fieldChars = /^\w*$/
const memberPath = /(?:^|[^\w.])\s*([A-Z_]\w*(?:\s*\.\s*[A-Z_]\w*)*)\s*$/i

// The options after `<before>.`: record fields, sync fields or methods.
function memberOptions(before: string, columns: Column[]): Completion[] {
  const match = memberPath.exec(before)
  const path = match ? match[1].split('.').map(s => s.trim()) : []
  if (path[0] === 'sync' && path.length === 1)
    return fieldOptions(SYNC_FIELDS)
  if (path[0] !== 'r')
    return methodOptions
  let level: Column[] | undefined = columns
  for (const segment of path.slice(1))
    level = level?.find(c => c.name === segment)?.fields
  if (path.length === 1)
    return fieldOptions(columns)
  return level ? fieldOptions(level) : methodOptions
}

// Completes record fields after `r.`, sync fields after `sync.`, methods after any other
// expression and a dot, and functions and variables on a bare identifier.
export function celCompletions(columns: Column[]): CompletionSource {
  return (ctx: CompletionContext): CompletionResult | null => {
    const text = ctx.state.sliceDoc(0, ctx.pos)
    if (celContextAt(text) !== 'code')
      return null
    const typed = word.exec(text)?.[0] ?? ''
    const from = ctx.pos - typed.length
    const beforeWord = text.slice(0, from)
    const dot = trailingDot.exec(beforeWord)
    if (dot) {
      if (numberEnd.test(beforeWord.slice(0, dot.index).trimEnd()))
        return null
      return { from, options: memberOptions(beforeWord.slice(0, dot.index), columns), validFor: fieldChars }
    }
    if (typed === '' && !ctx.explicit)
      return null
    if (digitEnd.test(beforeWord))
      return null
    return { from, options: [...rootOptions, ...allFunctionOptions], validFor: fieldChars }
  }
}

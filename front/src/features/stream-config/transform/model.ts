export type MapperKind = 'rename' | 'cast' | 'set' | 'hash' | 'mask' | 'filter' | 'drop'

export type StepJson = Record<string, unknown>

export interface CastOptions {
  to: string
  format?: string
  timezone?: string
  decimalSeparator?: string
  thousandsSeparator?: string
  trim?: boolean
  stripPrefix?: string
  stripSuffix?: string
  trueValues?: string[]
  falseValues?: string[]
  source?: string
  onError?: string
}

export interface StepEntry {
  field: string
  target?: string
  expr?: string
  cast?: CastOptions
  keepFirst?: number
  keepLast?: number
}

export interface StepDraft {
  id: string
  kind: MapperKind
  entries: StepEntry[]
  filter?: string
  onError?: string
}

export const MAPPER_KINDS: MapperKind[] = ['rename', 'cast', 'set', 'hash', 'mask', 'filter', 'drop']

export const MAPPER_LABELS: Record<MapperKind, string> = {
  rename: 'Rename',
  cast: 'Cast',
  set: 'Set',
  hash: 'Hash',
  mask: 'Mask',
  filter: 'Filter',
  drop: 'Drop',
}

export const CAST_TARGETS = ['string', 'integer', 'number', 'boolean', 'timestamp', 'date', 'object', 'array']

const castKeys: Array<[keyof CastOptions, string]> = [
  ['to', 'to'],
  ['format', 'format'],
  ['timezone', 'timezone'],
  ['decimalSeparator', 'decimal_separator'],
  ['thousandsSeparator', 'thousands_separator'],
  ['trim', 'trim'],
  ['stripPrefix', 'strip_prefix'],
  ['stripSuffix', 'strip_suffix'],
  ['trueValues', 'true_values'],
  ['falseValues', 'false_values'],
  ['source', 'source'],
  ['onError', 'on_error'],
]

let nextId = 0

function draftId(): string {
  nextId++
  return `step-${nextId}`
}

function stepKind(json: StepJson): MapperKind {
  const kind = MAPPER_KINDS.find(k => k in json)
  return kind ?? 'set'
}

function castFromJson(raw: unknown): CastOptions {
  if (typeof raw === 'string')
    return { to: raw }
  const obj = (raw ?? {}) as Record<string, unknown>
  const out: Record<string, unknown> = { to: '' }
  for (const [key, jsonKey] of castKeys) {
    if (obj[jsonKey] !== undefined)
      out[key] = obj[jsonKey]
  }
  return out as unknown as CastOptions
}

function isEmpty(value: unknown): boolean {
  return value === undefined || value === '' || value === false || (Array.isArray(value) && value.length === 0)
}

function castToJson(cast: CastOptions): unknown {
  const out: Record<string, unknown> = { to: cast.to }
  for (const [key, jsonKey] of castKeys) {
    if (key !== 'to' && !isEmpty(cast[key]))
      out[jsonKey] = cast[key]
  }
  return Object.keys(out).length === 1 ? cast.to : out
}

function maskFromJson(raw: unknown): Pick<StepEntry, 'keepFirst' | 'keepLast'> {
  const obj = (raw ?? {}) as { keep_first?: number, keep_last?: number }
  return { keepFirst: obj.keep_first, keepLast: obj.keep_last }
}

export function fromStepJson(json: StepJson): StepDraft {
  const kind = stepKind(json)
  const value = json[kind]
  const step: StepDraft = { id: draftId(), kind, entries: [] }
  if (typeof json.on_error === 'string' && json.on_error)
    step.onError = json.on_error

  const map = (value ?? {}) as Record<string, unknown>
  switch (kind) {
    case 'rename':
      step.entries = Object.entries(map).map(([field, target]) => ({ field, target: String(target ?? '') }))
      break
    case 'cast':
      step.entries = Object.entries(map).map(([field, spec]) => ({ field, cast: castFromJson(spec) }))
      break
    case 'set':
      step.entries = Object.entries(map).map(([field, expr]) => ({ field, expr: String(expr ?? '') }))
      break
    case 'hash':
    case 'drop':
      step.entries = (Array.isArray(value) ? value : []).map(field => ({ field: String(field) }))
      break
    case 'mask':
      step.entries = Object.entries(map).map(([field, spec]) => ({ field, ...maskFromJson(spec) }))
      break
    case 'filter':
      step.filter = typeof value === 'string' ? value : ''
      break
  }
  return step
}

export function toStepJson(step: StepDraft): StepJson {
  let value: unknown
  switch (step.kind) {
    case 'rename':
      value = Object.fromEntries(step.entries.map(e => [e.field, e.target ?? '']))
      break
    case 'cast':
      value = Object.fromEntries(step.entries.map(e => [e.field, castToJson(e.cast ?? { to: 'string' })]))
      break
    case 'set':
      value = Object.fromEntries(step.entries.map(e => [e.field, e.expr ?? '']))
      break
    case 'hash':
    case 'drop':
      value = step.entries.map(e => e.field)
      break
    case 'mask':
      value = Object.fromEntries(step.entries.map((e) => {
        const spec: Record<string, number> = {}
        if (e.keepFirst)
          spec.keep_first = e.keepFirst
        if (e.keepLast)
          spec.keep_last = e.keepLast
        return [e.field, spec]
      }))
      break
    case 'filter':
      value = step.filter ?? ''
      break
  }
  const json: StepJson = { [step.kind]: value }
  if (step.onError && step.kind !== 'rename' && step.kind !== 'drop')
    json.on_error = step.onError
  return json
}

export function newEntry(kind: MapperKind): StepEntry {
  switch (kind) {
    case 'rename':
      return { field: '', target: '' }
    case 'cast':
      return { field: '', cast: { to: 'string' } }
    case 'set':
      return { field: '', expr: '' }
    case 'mask':
      return { field: '', keepLast: 4 }
    default:
      return { field: '' }
  }
}

export function newStep(kind: MapperKind): StepDraft {
  if (kind === 'filter')
    return { id: draftId(), kind, entries: [], filter: '' }
  return { id: draftId(), kind, entries: [newEntry(kind)] }
}

export function addStep(steps: StepDraft[], kind: MapperKind): StepDraft[] {
  return [...steps, newStep(kind)]
}

export function moveStep(steps: StepDraft[], index: number, delta: number): StepDraft[] {
  const target = index + delta
  if (index < 0 || index >= steps.length || target < 0 || target >= steps.length)
    return steps
  const out = [...steps]
  const [step] = out.splice(index, 1)
  out.splice(target, 0, step)
  return out
}

export function updateStep(steps: StepDraft[], index: number, step: StepDraft): StepDraft[] {
  return steps.map((s, i) => (i === index ? step : s))
}

export function removeStep(steps: StepDraft[], index: number): StepDraft[] {
  return steps.filter((_, i) => i !== index)
}

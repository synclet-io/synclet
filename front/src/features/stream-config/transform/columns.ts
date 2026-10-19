import type { DestinationSyncMode, SelectedField } from '@entities/connection'
import type { StepDraft } from './model'

export interface Column {
  name: string
  type: string
  fields?: Column[]
}

export type ColumnMark = 'new' | 'renamed' | 'retyped' | 'dropped'

export interface OutputColumn extends Column {
  marks: ColumnMark[]
  renamedFrom?: string
  primaryKey: boolean
}

export interface TransformIssue {
  step: number
  field: string
  message: string
  severity: 'error' | 'warning'
}

export interface ColumnWalk {
  stepInputs: Column[][]
  output: OutputColumn[]
  issues: TransformIssue[]
}

const maxFieldNameBytes = 255

export const FILTER_WARNING = 'Rows filtered out later stay in the destination'

// eslint-disable-next-line no-control-regex
const controlChars = /[\u0000-\u001F\u007F-\u009F]/

export function fieldNameProblem(name: string): string | undefined {
  if (name === '')
    return 'field name is empty'
  if (new TextEncoder().encode(name).length > maxFieldNameBytes || name.trim() !== name || controlChars.test(name))
    return `invalid field name "${name}"`
  return undefined
}

export function schemaType(prop: unknown): string {
  const p = (prop ?? {}) as { type?: string | string[], format?: string }
  const types = (Array.isArray(p.type) ? p.type : [p.type]).filter((t): t is string => !!t && t !== 'null')
  const type = types[0]
  if (!type)
    return 'any'
  if (type === 'string' && p.format === 'date-time')
    return 'timestamp'
  if (type === 'string' && p.format === 'date')
    return 'date'
  return type
}

export function schemaColumns(schema: Record<string, unknown>): Column[] {
  const props = (schema?.properties ?? {}) as Record<string, unknown>
  return Object.entries(props).map(([name, prop]) => {
    const col: Column = { name, type: schemaType(prop) }
    const nested = (prop as { properties?: unknown } | undefined)?.properties
    if (nested && typeof nested === 'object')
      col.fields = schemaColumns(prop as Record<string, unknown>)
    return col
  })
}

export function isCdcStream(columns: Column[]): boolean {
  return columns.some(c => c.name.startsWith('_ab_cdc_'))
}

const recordField = /\br\.([A-Z_]\w*)/gi
const stringLiteral = /[rb]{0,2}(?:'''[\s\S]*?'''|"""[\s\S]*?"""|'(?:\\.|[^'\\\n])*'|"(?:\\.|[^"\\\n])*")/gi

// Record fields an expression reads, outside its string literals.
export function expressionFields(expr: string): string[] {
  const code = expr.replace(stringLiteral, '""')
  return Array.from(new Set(Array.from(code.matchAll(recordField), m => m[1])))
}

interface ColState {
  type: string
  fields?: Column[]
  source?: string
  written: boolean
  retyped: boolean
}

class Walker {
  cols = new Map<string, ColState>()
  created = new Map<string, number>()
  renamed = new Map<string, number>()
  issues: TransformIssue[] = []

  constructor(input: Column[]) {
    for (const c of input)
      this.cols.set(c.name, { type: c.type, fields: c.fields, source: c.name, written: false, retyped: false })
  }

  fail(step: number, field: string, message: string) {
    this.issues.push({ step, field, message, severity: 'error' })
  }

  snapshot(): Column[] {
    return Array.from(this.cols, ([name, c]) => (c.fields ? { name, type: c.type, fields: c.fields } : { name, type: c.type }))
  }

  ref(idx: number, field: string): boolean {
    if (this.cols.has(field))
      return true
    const by = this.renamed.get(field)
    if (field === '')
      this.fail(idx, field, 'field name is empty')
    else if (by !== undefined)
      this.fail(idx, field, `field "${field}" was renamed by step ${by + 1}`)
    else
      this.fail(idx, field, `field "${field}" does not exist`)
    return false
  }

  create(idx: number, field: string): boolean {
    const problem = fieldNameProblem(field)
    if (problem) {
      this.fail(idx, field, problem)
      return false
    }
    const by = this.created.get(field)
    if (by !== undefined && by !== idx) {
      this.fail(idx, field, `field "${field}" already exists (created by step ${by + 1})`)
      return false
    }
    return true
  }

  write(idx: number, field: string, type: string) {
    const existing = this.cols.get(field)
    if (!existing) {
      this.created.set(field, idx)
      this.cols.set(field, { type, written: true, retyped: false })
    }
    else {
      existing.retyped = existing.retyped || (existing.source !== undefined && type !== 'any' && type !== existing.type)
      if (type !== 'any')
        existing.type = type
      existing.fields = undefined
      existing.written = true
    }
    this.renamed.delete(field)
  }

  expression(idx: number, expr: string) {
    for (const field of expressionFields(expr)) {
      const by = this.renamed.get(field)
      if (by !== undefined && !this.cols.has(field))
        this.fail(idx, field, `field "${field}" was renamed by step ${by + 1}`)
    }
  }

  duplicates(idx: number, step: StepDraft): Set<number> {
    const seen = new Set<string>()
    const skip = new Set<number>()
    step.entries.forEach((e, i) => {
      if (e.field !== '' && seen.has(e.field)) {
        this.fail(idx, e.field, `field "${e.field}" is listed twice in this step`)
        skip.add(i)
      }
      seen.add(e.field)
    })
    return skip
  }

  rename(idx: number, step: StepDraft, skip: Set<number>, pk: string[][]) {
    const sources = new Set(step.entries.map(e => e.field))
    const targets = new Set<string>()
    const pairs: Array<[string, string]> = []
    step.entries.forEach((e, i) => {
      if (skip.has(i))
        return
      const to = e.target ?? ''
      let ok = this.ref(idx, e.field)
      const problem = fieldNameProblem(to)
      if (problem) {
        this.fail(idx, to, problem)
        ok = false
      }
      else if (targets.has(to) || (this.cols.has(to) && !sources.has(to))) {
        this.fail(idx, to, `field "${to}" already exists`)
        ok = false
      }
      targets.add(to)
      if (ok)
        pairs.push([e.field, to])
    })
    if (pairs.length === 0)
      return

    const targetOf = new Map(pairs)
    const next = new Map<string, ColState>()
    for (const [name, col] of this.cols) {
      const to = targetOf.get(name)
      if (to === undefined) {
        if (!pairs.some(([, t]) => t === name))
          next.set(name, col)
        continue
      }
      next.set(to, { ...col, written: true })
    }
    this.cols = next
    for (const [from, to] of pairs) {
      this.created.delete(from)
      this.created.delete(to)
      this.renamed.set(from, idx)
      this.renamed.delete(to)
    }
    for (const path of pk) {
      const to = path.length > 0 ? targetOf.get(path[0]) : undefined
      if (to !== undefined)
        path[0] = to
    }
  }

  step(idx: number, step: StepDraft, pk: string[][], keyed: boolean) {
    const skip = this.duplicates(idx, step)
    const entries = step.entries.filter((_, i) => !skip.has(i))
    switch (step.kind) {
      case 'rename':
        this.rename(idx, step, skip, pk)
        break
      case 'cast':
        for (const e of entries) {
          const to = e.cast?.to || 'string'
          const ok = e.cast?.source
            ? (this.cols.has(e.field) || this.create(idx, e.field))
            : this.ref(idx, e.field)
          if (e.cast?.source)
            this.expression(idx, e.cast.source)
          if (ok)
            this.write(idx, e.field, to)
        }
        break
      case 'set':
        for (const e of entries) {
          this.expression(idx, e.expr ?? '')
          if (this.create(idx, e.field))
            this.write(idx, e.field, 'any')
        }
        break
      case 'hash':
        for (const e of entries) {
          if (this.ref(idx, e.field))
            this.write(idx, e.field, 'string')
        }
        break
      case 'mask':
        for (const e of entries) {
          if ((e.keepFirst ?? 0) < 0 || (e.keepLast ?? 0) < 0) {
            this.fail(idx, e.field, 'keep_first and keep_last must not be negative')
            continue
          }
          if (this.ref(idx, e.field))
            this.write(idx, e.field, 'string')
        }
        break
      case 'filter':
        this.expression(idx, step.filter ?? '')
        if (keyed)
          this.issues.push({ step: idx, field: '', message: FILTER_WARNING, severity: 'warning' })
        break
    }
  }
}

function isSelected(selectedFields: SelectedField[], name: string): boolean {
  return selectedFields.length === 0 || selectedFields.some(sf => sf.fieldPath[0] === name)
}

function selectionCovers(selectedFields: SelectedField[], path: string[]): boolean {
  return selectedFields.some(sf => sf.fieldPath.length > 0 && sf.fieldPath.length <= path.length && sf.fieldPath.every((seg, i) => seg === path[i]))
}

export function walkColumns(
  input: Column[],
  steps: StepDraft[],
  selectedFields: SelectedField[],
  primaryKey: string[][],
  destinationSyncMode: DestinationSyncMode,
): ColumnWalk {
  const walker = new Walker(input)
  const dedup = destinationSyncMode === 'append_dedup'
  const keyed = dedup || isCdcStream(input)
  const pk = primaryKey.map(p => [...p])
  const stepInputs: Column[][] = []

  steps.forEach((step, idx) => {
    stepInputs.push(walker.snapshot())
    walker.step(idx, step, pk, keyed)
  })

  if (dedup && selectedFields.length > 0) {
    primaryKey.forEach((path, i) => {
      if (path.length === 0)
        return
      const current = walker.cols.get(pk[i][0])
      if ((current?.written ?? false) || selectionCovers(selectedFields, path))
        return
      const field = path.join('.')
      walker.fail(-1, field, `primary key field "${field}" is not selected`)
    })
  }

  const pkNames = new Set(pk.filter(p => p.length > 0).map(p => p[0]))
  const output = Array.from(walker.cols, ([name, col]): OutputColumn => {
    const marks: ColumnMark[] = []
    if (col.source === undefined)
      marks.push('new')
    if (col.source !== undefined && col.source !== name)
      marks.push('renamed')
    if (col.retyped)
      marks.push('retyped')
    if (!col.written && col.source !== undefined && !isSelected(selectedFields, col.source))
      marks.push('dropped')
    return {
      name,
      type: col.type,
      marks,
      renamedFrom: col.source !== undefined && col.source !== name ? col.source : undefined,
      primaryKey: pkNames.has(name),
    }
  })

  return { stepInputs, output, issues: walker.issues }
}

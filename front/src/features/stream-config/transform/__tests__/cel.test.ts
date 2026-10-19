import type { Completion, CompletionResult } from '@codemirror/autocomplete'
import type { Column } from '../columns'
import { readFileSync } from 'node:fs'
import { resolve } from 'node:path'
import { CompletionContext, completionStatus, startCompletion } from '@codemirror/autocomplete'
import { EditorState } from '@codemirror/state'
import { runScopeHandlers } from '@codemirror/view'
import { mount } from '@vue/test-utils'
import { describe, expect, it, vi } from 'vitest'
import { celCompletions } from '../cel/complete'
import functions from '../cel/functions.json'
import { CEL_TOKEN_CLASSES, celTokens } from '../cel/language'
import CelInput from '../CelInput.vue'
import StepForm from '../StepForm.vue'
import { celText, celView, setCel } from './celInput'

const columns: Column[] = [
  { name: 'id', type: 'integer' },
  { name: 'address', type: 'object', fields: [{ name: 'city', type: 'string' }, { name: 'zip', type: 'string' }] },
]

function complete(doc: string, cols: Column[] = columns, explicit = false): CompletionResult | null {
  const state = EditorState.create({ doc, selection: { anchor: doc.length } })
  return celCompletions(cols)(new CompletionContext(state, doc.length, explicit)) as CompletionResult | null
}

function labels(options: readonly Completion[] | undefined, type?: string): string[] {
  return (options ?? []).filter(o => type === undefined || o.type === type).map(o => o.label)
}

function kinds(text: string): string[] {
  return celTokens(text).filter(t => t.kind !== '').map(t => t.kind)
}

describe('the CEL tokenizer', () => {
  it('tags fields, functions, operators, numbers and comments', () => {
    expect(celTokens('r.amount * 2 > 10 && has(r.x) // c').filter(t => t.kind !== '')).toEqual([
      { text: 'r', kind: 'variable' },
      { text: 'amount', kind: 'property' },
      { text: '*', kind: 'operator' },
      { text: '2', kind: 'number' },
      { text: '>', kind: 'operator' },
      { text: '10', kind: 'number' },
      { text: '&&', kind: 'operator' },
      { text: 'has', kind: 'function' },
      { text: 'r', kind: 'variable' },
      { text: 'x', kind: 'property' },
      { text: '// c', kind: 'comment' },
    ])
  })

  it('tags strings, keywords and number forms', () => {
    expect(kinds('\'a\' + "b"')).toEqual(['string', 'operator', 'string'])
    expect(kinds('true')).toEqual(['keyword'])
    expect(kinds('null')).toEqual(['keyword'])
    expect(kinds('1.5')).toEqual(['number'])
    expect(kinds('0x1F')).toEqual(['number'])
    expect(kinds('1u')).toEqual(['number'])
    expect(kinds('r\'raw\' + \'\'\'tri\'ple\'\'\'')).toEqual(['string', 'operator', 'string'])
    expect(kinds('x in [1] ? !y : z <= 3')).toEqual(['variable', 'keyword', 'number', 'operator', 'operator', 'variable', 'operator', 'variable', 'operator', 'number'])
  })

  it('tags a method call after a dot as a function', () => {
    expect(celTokens('r.s.lowerAscii()').filter(t => t.kind !== '').map(t => t.kind)).toEqual(['variable', 'property', 'function'])
  })

  it('colours every token class in the light and the dark theme', () => {
    const css = readFileSync(resolve(__dirname, '../../../../style.css'), 'utf8')
    expect(CEL_TOKEN_CLASSES.length).toBeGreaterThanOrEqual(8)
    for (const cls of CEL_TOKEN_CLASSES) {
      expect(css, cls).toMatch(new RegExp(`(^|[\\s,}])\\.${cls}\\s*[,{]`, 'm'))
      expect(css, `.dark ${cls}`).toMatch(new RegExp(`\\.dark \\.${cls}\\s*[,{]`))
    }
  })
})

describe('the CEL completion', () => {
  it('offers the step input columns with their types after r.', () => {
    const res = complete('r.')
    expect(res?.options.map(o => [o.label, o.detail])).toEqual([['id', 'integer'], ['address', 'object']])
  })

  it('offers nested fields after r.<object>.', () => {
    expect(labels(complete('r.address.')?.options, 'property')).toEqual(['city', 'zip'])
    expect(labels(complete('r.address.ci')?.options, 'property')).toEqual(['city', 'zip'])
    expect(complete('r.address.ci')?.from).toBe('r.address.'.length)
  })

  it('offers no fields after a scalar column', () => {
    expect(labels(complete('r.id.')?.options, 'property')).toEqual([])
  })

  it('offers exactly the sync fields after sync.', () => {
    expect(complete('sync.')?.options.map(o => [o.label, o.detail])).toEqual([
      ['started_at', 'timestamp'],
      ['job_id', 'string'],
      ['connection', 'string'],
    ])
  })

  it('offers methods after an expression and a dot', () => {
    const names = labels(complete('\'abc\'.')?.options)
    expect(names).toContain('lowerAscii')
    expect(names).not.toContain('parseTimestamp')
  })

  it('offers a function with its signature', () => {
    const res = complete('low')
    const lower = res?.options.find(o => o.label === 'lowerAscii')
    expect(lower?.detail).toBe('string.lowerAscii() -> string')
    expect(res?.from).toBe(0)
  })

  it('offers exactly the engine functions, each with a signature, plus r and sync', () => {
    const res = complete('', columns, true)
    const fns = (res?.options ?? []).filter(o => o.type === 'function' || o.type === 'method')
    const expected = ['lowerAscii', 'upperAscii', 'trim', 'substring', 'split', 'replace', 'matches', 'int', 'double', 'string', 'timestamp', 'duration', 'has', 'coalesce', 'now', 'sha256', 'parseTimestamp', 'parseNumber']
    expect(fns.map(o => o.label).sort()).toEqual([...expected].sort())
    expect(functions.map(f => f.name).sort()).toEqual([...expected].sort())
    for (const f of fns)
      expect(f.detail, f.label).toBeTruthy()
    expect(labels(res?.options, 'variable')).toEqual(['r', 'sync'])
  })

  it('offers the columns after r. that follows a word operator', () => {
    expect(labels(complete('x in r.')?.options)).toEqual(['id', 'address'])
    expect(labels(complete('\'a\' in r.address.')?.options)).toEqual(['city', 'zip'])
  })

  it('offers nothing inside a string or a comment', () => {
    expect(complete('\'r.')).toBeNull()
    expect(complete('"lo')).toBeNull()
    expect(complete('r.a // r.')).toBeNull()
  })
})

describe('the CelInput Enter key', () => {
  function pressEnter(multiline: boolean) {
    const wrapper = mount(CelInput, { props: { modelValue: 'ab', columns, multiline }, attachTo: document.body })
    const view = celView(wrapper)
    view.dispatch({ selection: { anchor: 2 } })
    runScopeHandlers(view, new KeyboardEvent('keydown', { key: 'Enter' }), 'editor')
    const result = { doc: view.state.doc.toString(), emitted: wrapper.emitted('update:modelValue') }
    wrapper.unmount()
    return result
  }

  it('leaves a single-line input unchanged', () => {
    expect(pressEnter(false)).toEqual({ doc: 'ab', emitted: undefined })
  })

  it('accepts an open completion in a single-line input', async () => {
    const wrapper = mount(CelInput, { props: { modelValue: 'r.', columns }, attachTo: document.body })
    const view = celView(wrapper)
    view.dispatch({ selection: { anchor: 2 } })
    startCompletion(view)
    await vi.waitFor(() => expect(completionStatus(view.state)).toBe('active'))
    await new Promise(resolve => setTimeout(resolve, 100))
    runScopeHandlers(view, new KeyboardEvent('keydown', { key: 'Enter' }), 'editor')
    expect(view.state.doc.toString()).toBe('r.address')
    wrapper.unmount()
  })

  it('inserts a newline in a multiline input', () => {
    expect(pressEnter(true).doc).toBe('ab\n')
  })
})

describe('the CelInput in StepForm', () => {
  it('renders highlighted tokens for a set expression and emits typed text', async () => {
    const wrapper = mount(StepForm, {
      props: {
        step: { id: 's1', kind: 'set', entries: [{ field: 'x', expr: 'r.id + 1' }] },
        columns,
        resultTypes: {},
      },
      attachTo: document.body,
    })
    const expr = wrapper.get('[data-testid="entry-expr"]')
    expect(celText(expr)).toBe('r.id + 1')
    expect(expr.findAll('.tok-variable').length).toBeGreaterThan(0)
    expect(expr.findAll('.tok-number').length).toBeGreaterThan(0)

    setCel(expr, 'sha256(r.id)')
    const updates = wrapper.emitted('update') as Array<[{ entries: Array<{ expr?: string }> }]>
    expect(updates.at(-1)?.[0].entries[0].expr).toBe('sha256(r.id)')
    wrapper.unmount()
  })
})

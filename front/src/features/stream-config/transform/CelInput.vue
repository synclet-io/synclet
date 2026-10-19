<script setup lang="ts">
import type { Column } from './columns'
import { autocompletion, completionKeymap } from '@codemirror/autocomplete'
import { defaultKeymap, history, historyKeymap } from '@codemirror/commands'
import { syntaxHighlighting } from '@codemirror/language'
import { Annotation, Compartment, EditorState, Prec } from '@codemirror/state'
import { EditorView, keymap, placeholder as placeholderExt } from '@codemirror/view'
import { onBeforeUnmount, onMounted, ref, watch } from 'vue'
import { celCompletions } from './cel/complete'
import { celHighlighter, celLanguage } from './cel/language'

const props = withDefaults(defineProps<{
  modelValue: string
  columns: Column[]
  multiline?: boolean
  placeholder?: string
  ariaLabel?: string
}>(), { multiline: false, placeholder: '', ariaLabel: undefined })

const emit = defineEmits<{
  'update:modelValue': [value: string]
}>()

const host = ref<HTMLDivElement>()
const completion = new Compartment()
const external = Annotation.define<boolean>()
let view: EditorView | undefined

function completionExtension(columns: Column[]) {
  return autocompletion({ override: [celCompletions(columns)], icons: false })
}

// A single-line input keeps its text on one line: Enter only accepts a completion, and pasted newlines become spaces.
const newline = /\r?\n/g
const singleLine = [
  Prec.highest(keymap.of([{ key: 'Enter', run: () => true }])),
  EditorState.transactionFilter.of((tr) => {
    if (!tr.docChanged || tr.newDoc.lines === 1)
      return tr
    const text = tr.newDoc.toString()
    return [tr, { changes: { from: 0, to: text.length, insert: text.replace(newline, ' ') }, sequential: true }]
  }),
]

const theme = EditorView.theme({
  '&': { fontSize: '0.875rem' },
  '&.cm-focused': { outline: 'none' },
  '.cm-content': { fontFamily: 'inherit', padding: '0', caretColor: 'var(--color-text-heading)' },
  '.cm-line': { padding: '0' },
  '.cm-scroller': { fontFamily: 'inherit', lineHeight: '1.5rem' },
  '.cm-placeholder': { color: 'var(--color-text-muted)' },
})

onMounted(() => {
  view = new EditorView({
    parent: host.value!,
    state: EditorState.create({
      doc: props.modelValue,
      extensions: [
        history(),
        celLanguage,
        syntaxHighlighting(celHighlighter),
        completion.of(completionExtension(props.columns)),
        keymap.of([...completionKeymap, ...historyKeymap, ...defaultKeymap]),
        placeholderExt(props.placeholder),
        theme,
        props.multiline ? EditorView.lineWrapping : singleLine,
        EditorView.contentAttributes.of(props.ariaLabel ? { 'aria-label': props.ariaLabel } : {}),
        EditorView.updateListener.of((u) => {
          if (u.docChanged && !u.transactions.some(tr => tr.annotation(external)))
            emit('update:modelValue', u.state.doc.toString())
        }),
      ],
    }),
  })
})

watch(() => props.modelValue, (value) => {
  if (view && value !== view.state.doc.toString())
    view.dispatch({ changes: { from: 0, to: view.state.doc.length, insert: value }, annotations: external.of(true) })
})

watch(() => props.columns, (columns) => {
  view?.dispatch({ effects: completion.reconfigure(completionExtension(columns)) })
})

onBeforeUnmount(() => {
  view?.destroy()
  view = undefined
})
</script>

<template>
  <div
    ref="host"
    class="cel-input min-w-0 font-mono px-2.5 border border-border rounded-md bg-surface text-heading focus-within:ring-2 focus-within:ring-primary/20 focus-within:border-primary"
    :class="multiline ? 'py-1 min-h-[3.25rem]' : 'py-[3px] min-h-8'"
  />
</template>

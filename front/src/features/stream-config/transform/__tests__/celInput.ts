import { EditorView } from '@codemirror/view'

export function celView(wrapper: { element: Element }): EditorView {
  const editor = wrapper.element.querySelector('.cm-editor')
  const found = editor ? EditorView.findFromDOM(editor as HTMLElement) : null
  if (!found)
    throw new Error('no CEL editor inside the element')
  return found
}

export function celText(wrapper: { element: Element }): string {
  return celView(wrapper).state.doc.toString()
}

// Replaces the editor's text the way typing does.
export function setCel(wrapper: { element: Element }, value: string) {
  const v = celView(wrapper)
  v.dispatch({ changes: { from: 0, to: v.state.doc.length, insert: value }, userEvent: 'input.type' })
}

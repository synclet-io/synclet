import { describe, expect, it } from 'vitest'
import { nextTick } from 'vue'
import { cleanConfig, useConfigEditor } from '../useConfigEditor'

describe('cleanConfig', () => {
  it('drops UI-only keys at every depth and keeps arrays intact', () => {
    expect(cleanConfig({ host: 'db', __tab: 1, ssl: { mode: 'require', __open: true }, tags: [{ __x: 1 }] }))
      .toEqual({ host: 'db', ssl: { mode: 'require' }, tags: [{ __x: 1 }] })
  })
})

describe('useConfigEditor', () => {
  it('mirrors form values into the JSON text while the form is shown', async () => {
    const editor = useConfigEditor()
    editor.values.value = { port: 5432 }
    await nextTick()
    expect(JSON.parse(editor.jsonText.value)).toEqual({ port: 5432 })
  })

  it('parses the JSON back into the form when switching to it', () => {
    const editor = useConfigEditor()
    expect(editor.toggleJson()).toBe(true)
    editor.jsonText.value = '{"port": 1}'
    expect(editor.toggleJson()).toBe(true)
    expect(editor.jsonMode.value).toBe(false)
    expect(editor.values.value).toEqual({ port: 1 })
  })

  it('reports invalid JSON on toggle and on resolve', () => {
    const editor = useConfigEditor()
    editor.toggleJson()
    editor.jsonText.value = '{'
    expect(() => editor.resolve()).toThrow('Invalid JSON')
    expect(editor.toggleJson()).toBe(false)
  })

  it('resolves the JSON editor contents without UI-only keys', () => {
    const editor = useConfigEditor()
    editor.reset({ a: 1 })
    editor.toggleJson()
    editor.jsonText.value = '{"a": 2, "__b": 3}'
    expect(editor.resolve()).toEqual({ a: 2 })
  })
})

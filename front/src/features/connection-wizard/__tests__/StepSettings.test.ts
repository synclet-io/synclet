/* eslint-disable no-template-curly-in-string -- ${SOURCE_NAMESPACE} is the literal placeholder under test */
import { mount } from '@vue/test-utils'
import { describe, expect, it } from 'vitest'
import StepSettings from '../StepSettings.vue'

function mountStep() {
  return mount(StepSettings, {
    props: {
      name: 'c',
      schedule: '',
      schemaChangePolicy: 'propagate',
      maxAttempts: 3,
      namespaceDefinition: 'source',
      customNamespaceFormat: '',
      streamPrefix: '',
    },
    global: { stubs: { SCronEditor: true } },
  })
}

describe('stepSettings destination naming', () => {
  it('shows the destination namespace group, the table name prefix and the example', async () => {
    const w = mountStep()
    expect(w.get('fieldset legend').text()).toBe('Destination namespace')
    expect(w.findAll('label').some(l => l.text() === 'Table name prefix')).toBe(true)
    expect(w.text()).not.toContain('Namespace & Prefix')
    expect(w.get('[data-testid="destination-name-example"]').text()).toContain('public.users → public.users')
  })

  it('emits the prefix mode as a custom format', async () => {
    const w = mountStep()
    await w.get('fieldset select').setValue('source_prefix')
    expect(w.emitted('update:namespaceDefinition')?.at(-1)).toEqual(['custom'])
    expect(w.emitted('update:customNamespaceFormat')?.at(-1)).toEqual(['${SOURCE_NAMESPACE}'])
  })

  it('emits the table name prefix', async () => {
    const w = mountStep()
    const label = w.findAll('label').find(l => l.text() === 'Table name prefix')!
    await w.get(`#${label.attributes('for')}`).setValue('production_')
    expect(w.emitted('update:streamPrefix')?.at(-1)).toEqual(['production_'])
  })
})

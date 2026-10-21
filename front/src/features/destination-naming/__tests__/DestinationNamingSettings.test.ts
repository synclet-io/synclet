/* eslint-disable no-template-curly-in-string -- ${SOURCE_NAMESPACE} is the literal placeholder under test */
import type { NamespaceDefinition } from '@entities/connection'
import { mount } from '@vue/test-utils'
import { describe, expect, it } from 'vitest'
import { defineComponent, h, ref } from 'vue'
import DestinationNamingSettings from '../DestinationNamingSettings.vue'

function mountHost(initial: { def?: NamespaceDefinition, format?: string, prefix?: string, sampleStream?: { namespace: string, name: string } } = {}) {
  const def = ref<NamespaceDefinition>(initial.def ?? 'source')
  const format = ref(initial.format ?? '')
  const prefix = ref(initial.prefix ?? '')
  const Host = defineComponent(() => () => h(DestinationNamingSettings, {
    'namespaceDefinition': def.value,
    'customNamespaceFormat': format.value,
    'streamPrefix': prefix.value,
    'sampleStream': initial.sampleStream,
    'onUpdate:namespaceDefinition': (v: NamespaceDefinition) => { def.value = v },
    'onUpdate:customNamespaceFormat': (v: string) => { format.value = v },
    'onUpdate:streamPrefix': (v: string) => { prefix.value = v },
  }))
  const w = mount(Host)
  return { w, def, format, prefix }
}

function inputByLabel(w: ReturnType<typeof mount>, text: string) {
  const label = w.findAll('label').find(l => l.text() === text)
  if (!label)
    throw new Error(`no label ${text}`)
  return w.get(`#${label.attributes('for')}`)
}

function example(w: ReturnType<typeof mount>) {
  return w.get('[data-testid="destination-name-example"]').text()
}

describe('destinationNamingSettings', () => {
  it('groups the namespace mode and format and keeps the table name prefix separate', async () => {
    const { w } = mountHost({ def: 'custom', format: 'analytics' })
    const group = w.get('fieldset')
    expect(group.get('legend').text()).toBe('Destination namespace')
    const groupLabels = group.findAll('label').map(l => l.text())
    expect(groupLabels).toEqual(['Namespace mode', 'Custom namespace format'])

    const prefixLabel = w.findAll('label').find(l => l.text() === 'Table name prefix')!
    expect(prefixLabel.exists()).toBe(true)
    expect(group.element.contains(prefixLabel.element)).toBe(false)
    expect(w.text()).toContain('Prepended to every destination table or stream name')
  })

  it('shows public.users and its destination name, updating as the fields change', async () => {
    const { w } = mountHost()
    expect(example(w)).toContain('public.users → public.users')

    await inputByLabel(w, 'Table name prefix').setValue('production_')
    expect(example(w)).toContain('public.users → public.production_users')

    await inputByLabel(w, 'Namespace mode').setValue('custom')
    await inputByLabel(w, 'Custom namespace format').setValue('analytics')
    expect(example(w)).toContain('public.users → analytics.production_users')

    await inputByLabel(w, 'Namespace mode').setValue('destination')
    expect(example(w)).toContain('public.users → production_users')
    expect(example(w)).toContain('destination\'s default namespace')
  })

  it('uses the given sample stream', () => {
    const { w } = mountHost({ def: 'custom', format: 'raw_${SOURCE_NAMESPACE}', sampleStream: { namespace: 'sales', name: 'orders' } })
    expect(example(w)).toContain('sales.orders → raw_sales.orders')
  })

  it('stores "Source namespace with a prefix" as a custom format', async () => {
    const { w, def, format } = mountHost()
    await inputByLabel(w, 'Namespace mode').setValue('source_prefix')
    await inputByLabel(w, 'Namespace prefix').setValue('prod_')
    expect(def.value).toBe('custom')
    expect(format.value).toBe('prod_${SOURCE_NAMESPACE}')
    expect(example(w)).toContain('public.users → prod_public.users')
  })

  it('opens a saved <prefix>${SOURCE_NAMESPACE} format in the prefix mode', () => {
    const { w } = mountHost({ def: 'custom', format: 'prod_${SOURCE_NAMESPACE}' })
    expect((inputByLabel(w, 'Namespace mode').element as HTMLSelectElement).value).toBe('source_prefix')
    expect((inputByLabel(w, 'Namespace prefix').element as HTMLInputElement).value).toBe('prod_')
    expect(w.findAll('label').some(l => l.text() === 'Custom namespace format')).toBe(false)
  })

  it('opens any other custom format as "Custom format"', () => {
    const { w } = mountHost({ def: 'custom', format: 'prod_${SOURCE_NAMESPACE}_raw' })
    expect((inputByLabel(w, 'Namespace mode').element as HTMLSelectElement).value).toBe('custom')
    expect((inputByLabel(w, 'Custom namespace format').element as HTMLInputElement).value).toBe('prod_${SOURCE_NAMESPACE}_raw')
  })

  it('switches to the prefix mode when a saved format arrives after mount', async () => {
    const { w, def, format } = mountHost()
    def.value = 'custom'
    format.value = 'stg_${SOURCE_NAMESPACE}'
    await w.vm.$nextTick()
    expect((inputByLabel(w, 'Namespace mode').element as HTMLSelectElement).value).toBe('source_prefix')
    expect((inputByLabel(w, 'Namespace prefix').element as HTMLInputElement).value).toBe('stg_')
  })

  it('stays in "Custom format" while a prefix-shaped format is typed', async () => {
    const { w } = mountHost({ def: 'custom', format: 'analytics' })
    await inputByLabel(w, 'Custom namespace format').setValue('prod_${SOURCE_NAMESPACE}')
    expect((inputByLabel(w, 'Namespace mode').element as HTMLSelectElement).value).toBe('custom')
  })

  it('keeps $ out of the namespace prefix', async () => {
    const { w, format } = mountHost()
    await inputByLabel(w, 'Namespace mode').setValue('source_prefix')
    await inputByLabel(w, 'Namespace prefix').setValue('p$x_')
    expect(format.value).toBe('px_${SOURCE_NAMESPACE}')
  })
})

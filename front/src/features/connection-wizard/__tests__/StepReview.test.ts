import type { WizardState } from '../useWizardState'
import { mount } from '@vue/test-utils'
import { describe, expect, it, vi } from 'vitest'
import { ref } from 'vue'
import StepReview from '../StepReview.vue'

vi.mock('@entities/connector', () => ({
  useManagedConnector: () => ({ data: ref(undefined) }),
}))
vi.mock('@entities/connection', () => ({
  useConnectionTags: () => ({ data: ref([{ tag: 'prod', connectionCount: 1 }]) }),
}))

function state(tags: string[]): WizardState {
  return {
    sourceId: 's',
    source: null,
    destinationId: 'd',
    destination: null,
    name: 'n',
    schedule: '',
    schemaChangePolicy: 'propagate',
    maxAttempts: 3,
    namespaceDefinition: 'source',
    customNamespaceFormat: '',
    streamPrefix: '',
    tags,
    discoveredCatalog: null,
    streams: [],
    streamsSkipped: true,
  }
}

describe('wizard review step tags', () => {
  it('emits the edited tags', async () => {
    const w = mount(StepReview, { props: { state: state(['eu']) }, attachTo: document.body })
    const input = w.get('input[role="combobox"]')
    await input.setValue('Prod')
    await input.trigger('keydown', { key: 'Enter' })
    expect(w.emitted('update:tags')).toEqual([[['eu', 'prod']]])
  })

  it('renders the tags error', () => {
    const w = mount(StepReview, { props: { state: state([]), tagsError: 'too many tags' } })
    expect(w.get('[role="alert"]').text()).toBe('too many tags')
  })
})

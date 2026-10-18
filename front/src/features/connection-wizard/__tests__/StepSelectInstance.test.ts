import { mount } from '@vue/test-utils'
import { describe, expect, it, vi } from 'vitest'
import { ref } from 'vue'
import StepSelectDestination from '../StepSelectDestination.vue'
import StepSelectSource from '../StepSelectSource.vue'

const SOURCE_CONNECTOR_ID = '0b9c5f8e-1d2a-4c3b-9e8f-7a6b5c4d3e2f'
const DESTINATION_CONNECTOR_ID = '5e4d3c2b-1a09-4f8e-8d7c-6b5a49382716'

vi.mock('@entities/source', () => ({
  useSources: () => ({
    data: ref({ items: [{ id: 'src-1', name: 'Storefront', managedConnectorId: SOURCE_CONNECTOR_ID }] }),
    isLoading: ref(false),
  }),
}))
vi.mock('@entities/destination', () => ({
  useDestinations: () => ({
    data: ref({ items: [{ id: 'dst-1', name: 'Analytics warehouse', managedConnectorId: DESTINATION_CONNECTOR_ID }] }),
    isLoading: ref(false),
  }),
}))
vi.mock('@entities/connector', () => ({
  useManagedConnectors: () => ({
    data: ref([
      { id: SOURCE_CONNECTOR_ID, name: 'Faker', connectorType: 'source', dockerImage: 'airbyte/source-faker', dockerTag: '6.2.38' },
      { id: DESTINATION_CONNECTOR_ID, name: 'Postgres', connectorType: 'destination', dockerImage: 'airbyte/destination-postgres', dockerTag: '3.0.23' },
    ]),
    isLoading: ref(false),
  }),
}))

describe('wizard instance cards', () => {
  it('show the source name and its connector, not the connector id', () => {
    const card = mount(StepSelectSource, { props: { sourceId: '' } }).find('button')
    expect(card.text()).toContain('Storefront')
    expect(card.text()).toContain('Faker (6.2.38)')
    expect(card.text()).not.toContain(SOURCE_CONNECTOR_ID)
  })

  it('show the destination name and its connector, not the connector id', () => {
    const card = mount(StepSelectDestination, { props: { destinationId: '' } }).find('button')
    expect(card.text()).toContain('Analytics warehouse')
    expect(card.text()).toContain('Postgres (3.0.23)')
    expect(card.text()).not.toContain(DESTINATION_CONNECTOR_ID)
  })
})

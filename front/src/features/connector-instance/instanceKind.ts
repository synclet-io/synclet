import type { Destination } from '@entities/destination'
import type { Source } from '@entities/source'
import type { MaybeRef, Ref } from 'vue'
import {
  useCreateDestination,
  useDeleteDestination,
  useDestination,
  useDestinations,
  useTestDestinationConnection,
  useUpdateDestination,
} from '@entities/destination'
import {
  useCreateSource,
  useDeleteSource,
  useSource,
  useSources,
  useTestSourceConnection,
  useUpdateSource,
} from '@entities/source'

export type InstanceKind = 'source' | 'destination'
export type ConnectorInstance = Source | Destination

export interface InstanceKindMeta {
  noun: string
  title: string
  pluralTitle: string
  listDescription: string
  emptyDescription: string
  setupDescription: string
  nameTooltip: string
  routes: { list: string, detail: string, create: string, edit: string }
}

export const INSTANCE_KINDS: Record<InstanceKind, InstanceKindMeta> = {
  source: {
    noun: 'source',
    title: 'Source',
    pluralTitle: 'Sources',
    listDescription: 'Manage your data source connections',
    emptyDescription: 'A source is a connector instance with credentials for a specific system. Sync a connector repository in Settings first, then add a source here.',
    setupDescription: 'Configure a new data source',
    nameTooltip: 'A friendly label for this source. Shown throughout the UI; doesn\'t have to match the underlying database or API name.',
    routes: { list: 'sources', detail: 'source-detail', create: 'source-new', edit: 'source-edit' },
  },
  destination: {
    noun: 'destination',
    title: 'Destination',
    pluralTitle: 'Destinations',
    listDescription: 'Manage your data destination connections',
    emptyDescription: 'A destination is a connector instance with credentials for the system that will receive synced data. Sync a connector repository in Settings first, then add a destination here.',
    setupDescription: 'Configure a new data destination',
    nameTooltip: 'A friendly label for this destination. Shown throughout the UI; doesn\'t have to match the underlying warehouse or bucket name.',
    routes: { list: 'destinations', detail: 'destination-detail', create: 'destination-new', edit: 'destination-edit' },
  },
}

export interface CreateInstanceParams {
  name: string
  managedConnectorId: string
  config: Record<string, unknown>
}

export function useInstances(kind: InstanceKind, options: { page: Ref<number>, pageSize: number }) {
  return kind === 'source' ? useSources(options) : useDestinations(options)
}

export function useInstance(kind: InstanceKind, id: MaybeRef<string>) {
  return kind === 'source' ? useSource(id) : useDestination(id)
}

export function useCreateInstance(kind: InstanceKind) {
  if (kind === 'source') {
    const mutation = useCreateSource()
    return {
      isPending: mutation.isPending,
      create: async (params: CreateInstanceParams): Promise<ConnectorInstance | undefined> =>
        (await mutation.mutateAsync(params)).source,
    }
  }
  const mutation = useCreateDestination()
  return {
    isPending: mutation.isPending,
    create: (params: CreateInstanceParams): Promise<ConnectorInstance | undefined> => mutation.mutateAsync(params),
  }
}

export function useUpdateInstance(kind: InstanceKind) {
  return kind === 'source' ? useUpdateSource() : useUpdateDestination()
}

export function useDeleteInstance(kind: InstanceKind) {
  return kind === 'source' ? useDeleteSource() : useDeleteDestination()
}

export function useTestInstanceConnection(kind: InstanceKind) {
  return kind === 'source' ? useTestSourceConnection() : useTestDestinationConnection()
}

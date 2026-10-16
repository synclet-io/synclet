import type { InstanceKind } from '../instanceKind'
import { vi } from 'vitest'
import { ref } from 'vue'

export const calls: Array<{ kind: InstanceKind, op: string, arg: unknown }> = []

export function fakeInstance(kind: InstanceKind) {
  return {
    id: `${kind}-1`,
    workspaceId: 'w1',
    name: `My ${kind}`,
    managedConnectorId: 'mc-1',
    config: { host: 'db' },
    runtimeConfig: null,
    createdAt: new Date('2026-01-01'),
    updatedAt: undefined,
  }
}

function mutation(kind: InstanceKind, op: string, result: () => unknown) {
  return {
    isPending: ref(false),
    mutateAsync: vi.fn(async (arg: unknown) => {
      calls.push({ kind, op, arg })
      return result()
    }),
  }
}

export function fakeEntity(kind: InstanceKind) {
  const title = kind === 'source' ? 'Source' : 'Destination'
  return {
    [`use${title}s`]: () => ({ data: ref({ items: [fakeInstance(kind)], total: 1 }), isLoading: ref(false), error: ref(null) }),
    [`use${title}`]: () => ({ data: ref(fakeInstance(kind)), isLoading: ref(false) }),
    [`useCreate${title}`]: () => mutation(kind, 'create', () => kind === 'source' ? { source: fakeInstance(kind) } : fakeInstance(kind)),
    [`useUpdate${title}`]: () => mutation(kind, 'update', () => fakeInstance(kind)),
    [`useDelete${title}`]: () => mutation(kind, 'delete', () => undefined),
    [`useTest${title}Connection`]: () => mutation(kind, 'test', () => ({ taskId: 't1' })),
  }
}

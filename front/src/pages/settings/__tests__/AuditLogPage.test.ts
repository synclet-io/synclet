import type { AuditEvent } from '@entities/audit'
import { mount } from '@vue/test-utils'
import { describe, expect, it, vi } from 'vitest'
import { ref } from 'vue'
import AuditLogPage from '../AuditLogPage.vue'

const events = ref<AuditEvent[]>([])

vi.mock('vue-router', async importOriginal => ({
  ...await importOriginal<typeof import('vue-router')>(),
  useRoute: () => ({ query: {} }),
  useRouter: () => ({ replace: vi.fn() }),
}))

vi.mock('@entities/audit', () => ({
  AUDIT_PAGE_SIZE: 50,
  useAuditEvents: () => ({
    data: ref({ events: events.value, hasMore: false }),
    isLoading: ref(false),
    isFetching: ref(false),
    error: ref(null),
  }),
  useAuditRetention: () => ({ data: ref(undefined) }),
  useUpdateAuditRetention: () => ({ mutateAsync: vi.fn(), isPending: ref(false) }),
}))

function auditEvent(overrides: Partial<AuditEvent>): AuditEvent {
  return {
    id: 'event-1',
    workspaceId: 'ws-1',
    actorType: 'user',
    actorId: 'user-1',
    actorLabel: 'alice@example.com',
    actorDeleted: false,
    action: 'update',
    resourceType: 'source',
    resourceId: 'source-1',
    resourceLabel: 'Postgres',
    changes: [],
    diffTruncated: false,
    createdAt: new Date('2026-10-16T12:00:00Z'),
    ...overrides,
  }
}

describe('auditLogPage', () => {
  it('labels events whose actor no longer exists', () => {
    events.value = [
      auditEvent({ id: 'live', actorId: 'user-live', actorLabel: 'alice@example.com' }),
      auditEvent({ id: 'gone', actorId: 'user-gone', actorLabel: 'bob@example.com', actorDeleted: true }),
    ]

    const wrapper = mount(AuditLogPage)
    const rows = wrapper.findAll('li')

    expect(rows).toHaveLength(2)
    expect(rows[0]!.text()).toContain('alice@example.com')
    expect(rows[0]!.text()).not.toContain('deleted user')
    expect(rows[1]!.text()).toContain('bob@example.com')
    expect(rows[1]!.text()).toContain('deleted user')
    expect(rows[1]!.text()).toContain('user-gone')
  })
})

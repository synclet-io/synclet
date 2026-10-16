<script setup lang="ts">
import type { AuditAction, AuditEvent, AuditResourceType, ListAuditEventsParams } from '@entities/audit'
import { AUDIT_PAGE_SIZE, useAuditEvents, useAuditRetention, useUpdateAuditRetention } from '@entities/audit'
import { getErrorMessage } from '@shared/lib/errorUtils'
import { PageHeader, SAlert, SBadge, SButton, SCard, SEmptyState, SInput, SSelect, SSkeleton, useToast } from '@shared/ui'
import { ChevronDown, ChevronRight, X } from 'lucide-vue-next'
import { computed, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'

const route = useRoute()
const router = useRouter()

function queryString(key: string): string {
  const value = route.query[key]
  return typeof value === 'string' ? value : ''
}

function setQuery(patch: Record<string, string | undefined>, resetPage = true) {
  const query: Record<string, string> = {}
  for (const [key, value] of Object.entries({ ...route.query, ...(resetPage ? { page: undefined } : {}), ...patch })) {
    if (typeof value === 'string' && value !== '')
      query[key] = value
  }
  router.replace({ query })
}

function queryModel<T extends string>(key: string) {
  return computed<T | ''>({
    get: () => queryString(key) as T | '',
    set: value => setQuery({ [key]: value || undefined }),
  })
}

const action = queryModel<AuditAction>('action')
const resourceType = queryModel<AuditResourceType>('resource')
const since = queryModel('since')
const until = queryModel('until')
const actorId = computed(() => queryString('actor'))
const actorLabel = computed(() => queryString('actorLabel'))
const resourceId = computed(() => queryString('resourceId'))
const resourceLabel = computed(() => queryString('resourceLabel'))

const page = computed(() => {
  const n = Number.parseInt(queryString('page'), 10)
  return Number.isInteger(n) && n > 1 ? n : 1
})

function setPage(n: number) {
  setQuery({ page: n > 1 ? String(n) : undefined }, false)
}

function startOfDay(date: string): Date | undefined {
  return date ? new Date(`${date}T00:00:00`) : undefined
}

function endOfDay(date: string): Date | undefined {
  if (!date)
    return undefined
  const d = new Date(`${date}T00:00:00`)
  d.setDate(d.getDate() + 1)
  return d
}

const params = computed<ListAuditEventsParams>(() => ({
  action: action.value || undefined,
  resourceType: resourceType.value || undefined,
  actorId: actorId.value || undefined,
  resourceId: resourceId.value || undefined,
  since: startOfDay(since.value),
  until: endOfDay(until.value),
  limit: AUDIT_PAGE_SIZE,
  offset: (page.value - 1) * AUDIT_PAGE_SIZE,
}))

const { data, isLoading, isFetching, error } = useAuditEvents(params)
const events = computed(() => data.value?.events ?? [])
const hasMore = computed(() => data.value?.hasMore ?? false)
const rangeStart = computed(() => (page.value - 1) * AUDIT_PAGE_SIZE + 1)
const rangeEnd = computed(() => rangeStart.value + events.value.length - 1)

const hasFilters = computed(() => !!(action.value || resourceType.value || actorId.value || resourceId.value || since.value || until.value))

function filterByActor(event: AuditEvent) {
  setQuery({ actor: event.actorId, actorLabel: event.actorLabel || event.actorType })
}

function filterByResource(event: AuditEvent) {
  setQuery({ resourceId: event.resourceId, resourceLabel: event.resourceLabel || event.resourceId })
}

function clearFilters() {
  router.replace({ query: {} })
}

const toast = useToast()
const { data: retention } = useAuditRetention()
const updateRetention = useUpdateAuditRetention()
const retentionDays = ref<string | number>('')
const retentionError = ref('')

watch(retention, (value) => {
  if (value)
    retentionDays.value = value.retentionDays
}, { immediate: true })

async function saveRetention() {
  retentionError.value = ''
  const days = Number(retentionDays.value)
  if (!Number.isInteger(days) || days < 1) {
    retentionError.value = 'Enter a whole number of days, at least 1'
    return
  }
  try {
    await updateRetention.mutateAsync(days)
    toast.success('Audit retention updated')
  }
  catch (e: unknown) {
    retentionError.value = getErrorMessage(e)
  }
}

const expanded = ref<Set<string>>(new Set())

function toggle(id: string) {
  const next = new Set(expanded.value)
  if (next.has(id))
    next.delete(id)
  else
    next.add(id)
  expanded.value = next
}

const actionOptions: Array<{ value: '' | AuditAction, label: string }> = [
  { value: '', label: 'All actions' },
  { value: 'create', label: 'Create' },
  { value: 'update', label: 'Update' },
  { value: 'delete', label: 'Delete' },
  { value: 'enable', label: 'Enable' },
  { value: 'disable', label: 'Disable' },
  { value: 'test', label: 'Test' },
  { value: 'sync', label: 'Sync' },
  { value: 'login', label: 'Login' },
  { value: 'logout', label: 'Logout' },
]

const resourceOptions: Array<{ value: '' | AuditResourceType, label: string }> = [
  { value: '', label: 'All resources' },
  { value: 'source', label: 'Sources' },
  { value: 'destination', label: 'Destinations' },
  { value: 'connection', label: 'Connections' },
  { value: 'connector', label: 'Connectors' },
  { value: 'repository', label: 'Repositories' },
  { value: 'notification_channel', label: 'Notification channels' },
  { value: 'notification_rule', label: 'Notification rules' },
  { value: 'webhook', label: 'Webhooks' },
  { value: 'workspace', label: 'Workspace' },
  { value: 'workspace_member', label: 'Workspace members' },
  { value: 'workspace_invite', label: 'Workspace invites' },
  { value: 'api_key', label: 'API keys' },
  { value: 'user', label: 'Users' },
]

function actionVariant(a: AuditAction | 'unknown'): 'success' | 'warning' | 'danger' | 'info' | 'gray' {
  switch (a) {
    case 'create': return 'success'
    case 'update': return 'info'
    case 'delete': return 'danger'
    case 'enable':
    case 'disable':
    case 'test':
    case 'sync':
      return 'warning'
    case 'login':
    case 'logout':
    case 'unknown':
      return 'gray'
  }
}

function formatValue(v: unknown): string {
  if (v === null || v === undefined)
    return '—'
  if (typeof v === 'string')
    return v
  return JSON.stringify(v)
}
</script>

<template>
  <PageHeader title="Audit log" description="Every configuration change in this workspace. Admin-only." :level="2">
    <template #actions>
      <div class="flex flex-col items-end gap-1">
        <form class="flex items-center gap-2" @submit.prevent="saveRetention">
          <span class="text-sm text-text-secondary whitespace-nowrap">Retention (days)</span>
          <SInput v-model="retentionDays" type="number" class="w-24" aria-label="Retention (days)" :disabled="!retention" />
          <SButton type="submit" variant="secondary" :loading="updateRetention.isPending.value" :disabled="!retention">
            Save
          </SButton>
        </form>
        <p v-if="retentionError" class="text-xs text-danger">
          {{ retentionError }}
        </p>
        <p v-else-if="retention" class="text-xs text-text-muted">
          Events older than this are deleted. Max {{ retention.maxRetentionDays }} days.
        </p>
      </div>
    </template>
  </PageHeader>

  <div class="flex flex-wrap items-end gap-3 mb-3">
    <SSelect v-model="action" :options="actionOptions" aria-label="Action" />
    <SSelect v-model="resourceType" :options="resourceOptions" aria-label="Resource type" />
    <SInput v-model="since" type="date" label="From" />
    <SInput v-model="until" type="date" label="To" />
    <SButton v-if="hasFilters" variant="ghost" @click="clearFilters">
      Clear filters
    </SButton>
  </div>

  <div v-if="actorId || resourceId" class="flex flex-wrap gap-2 mb-3">
    <SBadge v-if="actorId" variant="info">
      Actor: {{ actorLabel || actorId }}
      <button class="ml-1 rounded focus:outline-none focus-visible:ring-2 focus-visible:ring-primary/50" title="Remove actor filter" aria-label="Remove actor filter" @click="setQuery({ actor: undefined, actorLabel: undefined })">
        <X class="w-3 h-3" />
      </button>
    </SBadge>
    <SBadge v-if="resourceId" variant="info">
      Resource: {{ resourceLabel || resourceId }}
      <button class="ml-1 rounded focus:outline-none focus-visible:ring-2 focus-visible:ring-primary/50" title="Remove resource filter" aria-label="Remove resource filter" @click="setQuery({ resourceId: undefined, resourceLabel: undefined })">
        <X class="w-3 h-3" />
      </button>
    </SBadge>
  </div>

  <SAlert v-if="error" variant="danger" class="mb-3">
    {{ getErrorMessage(error) }}
  </SAlert>

  <div v-if="isLoading" class="space-y-2">
    <SSkeleton v-for="i in 5" :key="i" variant="rect" height="60px" />
  </div>

  <SEmptyState
    v-else-if="events.length === 0 && page > 1"
    title="No more audit events"
    description="This page is past the last event for the current filters."
  >
    <SButton variant="secondary" @click="setPage(1)">
      Back to first page
    </SButton>
  </SEmptyState>

  <SEmptyState
    v-else-if="events.length === 0 && hasFilters"
    title="No matching audit events"
    description="No events match the current filters."
  >
    <SButton variant="secondary" @click="clearFilters">
      Clear filters
    </SButton>
  </SEmptyState>

  <SEmptyState
    v-else-if="events.length === 0"
    title="No audit events"
    description="Configuration changes will appear here. Audit logging started when this feature was deployed; earlier changes are not recorded."
  />

  <SCard v-else :padding="false">
    <ul class="divide-y divide-border">
      <li v-for="event in events" :key="event.id" class="px-4 py-3">
        <button
          class="w-full flex items-start gap-3 text-left rounded focus:outline-none focus-visible:ring-2 focus-visible:ring-primary/50"
          :aria-expanded="expanded.has(event.id)"
          @click="toggle(event.id)"
        >
          <component :is="expanded.has(event.id) ? ChevronDown : ChevronRight" class="w-4 h-4 mt-0.5 text-text-muted shrink-0" />
          <div class="flex-1 min-w-0">
            <div class="flex flex-wrap items-center gap-2 mb-1">
              <SBadge :variant="actionVariant(event.action)">
                {{ event.action }}
              </SBadge>
              <span class="text-sm font-medium text-heading truncate">
                {{ event.resourceType }} — {{ event.resourceLabel || event.resourceId }}
              </span>
            </div>
            <p class="text-xs text-text-secondary">
              by {{ event.actorLabel || event.actorType }}
              <template v-if="event.actorDeleted">
                <SBadge variant="gray" :title="`User ${event.actorId} no longer exists`">
                  deleted user
                </SBadge>
                <span class="font-mono">{{ event.actorId }}</span>
              </template>
              · {{ event.createdAt?.toLocaleString() ?? 'unknown time' }}
            </p>
          </div>
        </button>
        <div v-if="expanded.has(event.id)" class="mt-3 pl-7">
          <div class="flex flex-wrap gap-2 mb-3">
            <SButton v-if="event.actorId && event.actorId !== actorId" size="sm" variant="secondary" @click="filterByActor(event)">
              Only events by {{ event.actorLabel || event.actorType }}
            </SButton>
            <SButton v-if="event.resourceId && event.resourceId !== resourceId" size="sm" variant="secondary" @click="filterByResource(event)">
              Only events for this {{ event.resourceType }}
            </SButton>
          </div>
          <div v-if="event.diffTruncated" class="text-xs text-warning mb-2">
            Diff truncated to 8 KB.
          </div>
          <table v-if="event.changes.length > 0" class="w-full text-xs">
            <thead class="text-text-muted uppercase text-xs tracking-wider">
              <tr>
                <th class="text-left py-1 pr-3">
                  Field
                </th>
                <th class="text-left py-1 pr-3">
                  Before
                </th>
                <th class="text-left py-1">
                  After
                </th>
              </tr>
            </thead>
            <tbody>
              <tr v-for="(change, idx) in event.changes" :key="`${event.id}-${idx}`" class="border-t border-border/50">
                <td class="py-1 pr-3 font-mono text-text-primary">
                  {{ change.path }}
                </td>
                <td class="py-1 pr-3 text-text-secondary font-mono">
                  {{ formatValue(change.before) }}
                </td>
                <td class="py-1 text-text-primary font-mono">
                  {{ formatValue(change.after) }}
                </td>
              </tr>
            </tbody>
          </table>
          <p v-else class="text-xs text-text-muted">
            No field-level diff captured.
          </p>
        </div>
      </li>
    </ul>
  </SCard>

  <div v-if="events.length > 0 && (page > 1 || hasMore)" class="flex items-center justify-between mt-4">
    <span class="text-xs text-text-muted">Showing {{ rangeStart }}–{{ rangeEnd }}</span>
    <div class="flex gap-2">
      <SButton variant="secondary" size="sm" :disabled="page <= 1 || isFetching" @click="setPage(page - 1)">
        Previous
      </SButton>
      <SButton variant="secondary" size="sm" :disabled="!hasMore || isFetching" @click="setPage(page + 1)">
        Next
      </SButton>
    </div>
  </div>
</template>

<script setup lang="ts">
import type { AuditAction, AuditResourceType, ListAuditEventsParams } from '@entities/audit'
import { useAuditEvents } from '@entities/audit'
import { PageHeader, SBadge, SCard, SEmptyState, SSelect, SSkeleton } from '@shared/ui'
import { ChevronDown, ChevronRight } from 'lucide-vue-next'
import { computed, ref } from 'vue'

const action = ref<AuditAction | ''>('')
const resourceType = ref<AuditResourceType | ''>('')

const params = computed<ListAuditEventsParams>(() => ({
  action: action.value || undefined,
  resourceType: resourceType.value || undefined,
  limit: 50,
  offset: 0,
}))

const { data: events, isLoading, error } = useAuditEvents(params)

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

function actionVariant(a: AuditAction): 'success' | 'warning' | 'danger' | 'info' | 'gray' {
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
  <PageHeader title="Audit log" description="Every configuration change in this workspace. Admin-only." />

  <div class="flex gap-3 mb-4">
    <SSelect v-model="action" :options="actionOptions" />
    <SSelect v-model="resourceType" :options="resourceOptions" />
  </div>

  <div v-if="error" class="text-sm text-danger mb-3">
    {{ error.message }}
  </div>

  <div v-if="isLoading" class="space-y-2">
    <SSkeleton v-for="i in 5" :key="i" variant="rect" height="60px" />
  </div>

  <SEmptyState
    v-else-if="!events || events.length === 0"
    title="No audit events"
    description="Configuration changes will appear here. Audit logging started when this feature was deployed; earlier changes are not recorded."
  />

  <SCard v-else :padding="false">
    <ul class="divide-y divide-border">
      <li v-for="event in events" :key="event.id" class="px-4 py-3">
        <button class="w-full flex items-start gap-3 text-left" @click="toggle(event.id)">
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
              · {{ event.createdAt?.toLocaleString() ?? 'unknown time' }}
            </p>
          </div>
        </button>
        <div v-if="expanded.has(event.id)" class="mt-3 pl-7">
          <div v-if="event.diffTruncated" class="text-xs text-amber-600 mb-2">
            Diff truncated to 8 KB.
          </div>
          <table v-if="event.changes.length > 0" class="w-full text-xs">
            <thead class="text-text-muted uppercase text-[10px] tracking-wider">
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
</template>

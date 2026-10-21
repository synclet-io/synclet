<script setup lang="ts">
import type { WorkflowRun, WorkflowStepInput } from '@entities/workflow'
import type { Column } from '@shared/ui'
import { useAuth } from '@entities/auth'
import { useConnections } from '@entities/connection'
import {
  useCreateWorkflow,
  useDeleteWorkflow,
  useRerunWorkflowRun,
  useRunWorkflow,
  useUpdateWorkflow,
  useWorkflow,
  useWorkflowRuns,
} from '@entities/workflow'
import { WorkflowStepsEditor } from '@features/workflow-editor'
import { runStatusVariant, WorkflowGraph } from '@features/workflow-graph'
import { describeCron } from '@shared/lib/cron'
import { getErrorMessage } from '@shared/lib/errorUtils'
import { PageHeader, SAlert, SBadge, SButton, SCard, SConfirmDialog, SCronEditor, SInput, SSelect, STable, useToast } from '@shared/ui'
import { computed, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'

const route = useRoute()
const router = useRouter()
const toast = useToast()

const id = computed(() => (route.params.id as string | undefined) ?? '')
const isNew = computed(() => !id.value)

const auth = useAuth()
const canEdit = computed(() => {
  const role = auth.workspaces.value.find(w => w.workspaceId === auth.currentWorkspaceId.value)?.role
  return role === 'editor' || role === 'admin'
})

const { data: workflow, error: workflowError } = useWorkflow(id)
const { data: runs } = useWorkflowRuns(id)
const { data: connectionPage } = useConnections()

const connections = computed(() => (connectionPage.value?.items ?? []).map(c => ({ id: c.id, name: c.name })))
const connectionNames = computed(() => new Map(connections.value.map(c => [c.id, c.name])))

function connectionName(connectionId: string): string {
  return connectionNames.value.get(connectionId) ?? connectionId
}

const form = ref({ name: '', schedule: '', steps: [] as WorkflowStepInput[] })

watch(workflow, (w) => {
  if (!w)
    return
  form.value = {
    name: w.name,
    schedule: w.schedule,
    steps: w.steps.map(s => ({ connectionId: s.connectionId, afterConnectionIds: [...s.afterConnectionIds] })),
  }
}, { immediate: true })

const selectedRunId = ref('')
const activeRun = computed(() => runs.value?.find(r => r.status === 'running'))

const createMutation = useCreateWorkflow()
const updateMutation = useUpdateWorkflow()
const deleteMutation = useDeleteWorkflow()
const runMutation = useRunWorkflow()
const rerunMutation = useRerunWorkflowRun()

async function save() {
  try {
    if (isNew.value) {
      const created = await createMutation.mutateAsync({ ...form.value })
      toast.success('Workflow created')
      router.push({ name: 'workflow-detail', params: { id: created.id } })
      return
    }
    await updateMutation.mutateAsync({ id: id.value, ...form.value })
    toast.success('Workflow saved')
  }
  catch (e: unknown) {
    toast.error(`Error: ${getErrorMessage(e)}`)
  }
}

async function runNow() {
  try {
    const run = await runMutation.mutateAsync(id.value)
    selectedRunId.value = run.id
    toast.success('Workflow run started')
  }
  catch (e: unknown) {
    toast.error(`Error: ${getErrorMessage(e)}`)
  }
}

async function rerun(run: WorkflowRun) {
  try {
    await rerunMutation.mutateAsync(run.id)
    selectedRunId.value = run.id
    toast.success('Rerunning the failed steps')
  }
  catch (e: unknown) {
    toast.error(`Error: ${getErrorMessage(e)}`)
  }
}

const confirmingDelete = ref(false)
const deleteError = ref('')

async function confirmDelete() {
  deleteError.value = ''
  try {
    await deleteMutation.mutateAsync(id.value)
    confirmingDelete.value = false
    toast.success('Workflow deleted')
    router.push({ name: 'workflows' })
  }
  catch (e: unknown) {
    deleteError.value = getErrorMessage(e)
  }
}

const shownRun = computed<WorkflowRun | undefined>(() => {
  const all = runs.value ?? []
  return all.find(r => r.id === selectedRunId.value) ?? all[0]
})

function runLabel(run: WorkflowRun): string {
  return `${run.startedAt?.toLocaleString() ?? 'Not started'} · ${run.status}`
}

const runOptions = computed(() => [
  { label: 'Latest run', value: '' },
  ...(runs.value ?? []).map(r => ({ label: runLabel(r), value: r.id })),
])

const runColumns: Column[] = [
  { key: 'status', label: 'Status' },
  { key: 'trigger', label: 'Trigger' },
  { key: 'started', label: 'Started' },
  { key: 'completed', label: 'Completed' },
  { key: 'steps', label: 'Steps' },
  { key: 'actions', label: '', align: 'right' },
]

function stepSummary(run: WorkflowRun): string {
  const done = run.stepRuns.filter(s => s.status === 'succeeded').length
  return `${done}/${run.stepRuns.length} succeeded`
}

const scheduleDescription = computed(() => (form.value.schedule ? describeCron(form.value.schedule) ?? form.value.schedule : 'Manual: runs only when started'))
const saving = computed(() => createMutation.isPending.value || updateMutation.isPending.value)
const title = computed(() => (isNew.value ? 'New workflow' : workflow.value?.name ?? 'Workflow'))
</script>

<template>
  <PageHeader :title="title" back-to="/workflows" back-label="Workflows">
    <template v-if="!isNew && canEdit" #actions>
      <SButton
        size="sm"
        :loading="runMutation.isPending.value"
        :disabled="!!activeRun"
        :title="activeRun ? 'A run is in progress' : undefined"
        @click="runNow"
      >
        Run now
      </SButton>
      <SButton variant="danger" size="sm" :disabled="!!activeRun" @click="confirmingDelete = true">
        Delete
      </SButton>
    </template>
  </PageHeader>

  <SAlert v-if="!isNew && workflowError" variant="danger" class="mb-4">
    {{ getErrorMessage(workflowError) }}
  </SAlert>

  <div class="space-y-4">
    <SCard title="Definition">
      <div class="space-y-4">
        <SInput v-model="form.name" label="Name" placeholder="e.g. Nightly warehouse load" :disabled="!canEdit" />
        <div>
          <SCronEditor v-if="canEdit" v-model="form.schedule" />
          <p v-else class="text-sm text-text-secondary" data-testid="schedule-readonly">
            Schedule: {{ scheduleDescription }}
          </p>
        </div>
        <div>
          <h4 class="text-sm font-medium text-heading mb-2">
            Steps
          </h4>
          <WorkflowStepsEditor v-model="form.steps" :connections="connections" :readonly="!canEdit" />
        </div>
        <SAlert v-if="canEdit && activeRun" variant="info">
          A run is in progress; the workflow can be changed once it ends.
        </SAlert>
        <div v-if="canEdit" class="flex justify-end">
          <SButton :loading="saving" :disabled="!!activeRun" @click="save">
            {{ isNew ? 'Create workflow' : 'Save workflow' }}
          </SButton>
        </div>
      </div>
    </SCard>

    <SCard v-if="!isNew && workflow" title="Graph">
      <div class="space-y-3">
        <div v-if="(runs ?? []).length > 0" class="w-full sm:w-80">
          <SSelect v-model="selectedRunId" :options="runOptions" size="sm" aria-label="Run shown in the graph" />
        </div>
        <p v-else class="text-sm text-text-secondary">
          No runs yet.
        </p>
        <WorkflowGraph :steps="workflow.steps" :run="shownRun" :connection-name="connectionName" />
      </div>
    </SCard>

    <SCard v-if="!isNew" title="Run history">
      <STable
        :columns="runColumns"
        :data="runs"
        empty-text="No runs yet"
        empty-description="Runs start on the schedule or with Run now."
      >
        <template #cell-status="{ row }">
          <SBadge :variant="runStatusVariant(row.status)" dot>
            {{ row.status }}
          </SBadge>
        </template>
        <template #cell-trigger="{ row }">
          <span class="text-sm text-text-secondary">{{ row.trigger }}</span>
        </template>
        <template #cell-started="{ row }">
          <span class="text-sm text-text-secondary">{{ row.startedAt?.toLocaleString() ?? '-' }}</span>
        </template>
        <template #cell-completed="{ row }">
          <span class="text-sm text-text-secondary">{{ row.completedAt?.toLocaleString() ?? '-' }}</span>
        </template>
        <template #cell-steps="{ row }">
          <span class="text-sm text-text-secondary">{{ stepSummary(row) }}</span>
        </template>
        <template #cell-actions="{ row }">
          <div class="flex justify-end gap-2">
            <SButton variant="ghost" size="sm" :aria-pressed="shownRun?.id === row.id" @click="selectedRunId = row.id">
              Show in graph
            </SButton>
            <SButton
              v-if="canEdit && row.status === 'failed'"
              variant="secondary"
              size="sm"
              :disabled="!!activeRun"
              :loading="rerunMutation.isPending.value"
              @click="rerun(row)"
            >
              Rerun failed steps
            </SButton>
          </div>
        </template>
      </STable>
    </SCard>
  </div>

  <SConfirmDialog
    :open="confirmingDelete"
    title="Delete workflow"
    :message="`Delete ${workflow?.name ?? 'this workflow'} and its run history? Its connections stay and keep no schedule until you set one.`"
    confirm-text="Delete"
    :loading="deleteMutation.isPending.value"
    :error="deleteError"
    @confirm="confirmDelete"
    @cancel="confirmingDelete = false"
  />
</template>

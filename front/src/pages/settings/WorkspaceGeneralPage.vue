<script setup lang="ts">
import { useAuth } from '@entities/auth'
import { deleteWorkspace, getWorkspace, updateWorkspace } from '@entities/workspace/api'
import { useExportConfig, useImportConfig } from '@entities/workspace/composables'
import { connectionClient } from '@shared/api/services'
import { getErrorMessage } from '@shared/lib/errorUtils'
import { workspaceKeys } from '@shared/lib/queryKeys'
import { PageHeader, SAlert, SButton, SCard, SConfirmDialog, SInput, SSkeleton, useToast } from '@shared/ui'
import { useQuery, useQueryClient } from '@tanstack/vue-query'
import { Download, Upload } from 'lucide-vue-next'
import { computed, inject, ref, watch } from 'vue'
import { useRouter } from 'vue-router'
import { workspaceAdminKey } from './SettingsLayout.vue'

const auth = useAuth()
const router = useRouter()
const toast = useToast()
const qc = useQueryClient()
const isAdmin = inject(workspaceAdminKey, computed(() => false))

const workspaceId = computed(() => auth.currentWorkspaceId.value ?? '')

const workspaceQuery = useQuery({
  queryKey: computed(() => workspaceKeys.detail(workspaceId.value)),
  queryFn: () => getWorkspace(workspaceId.value),
  enabled: computed(() => auth.isAuthenticated.value && !!workspaceId.value),
})
const settingsQuery = useQuery({
  queryKey: computed(() => [...workspaceKeys.detail(workspaceId.value), 'pipelineSettings'] as const),
  queryFn: async () => (await connectionClient.getPipelineSettings({})).settings?.maxJobsPerWorkspace ?? 0,
  enabled: computed(() => auth.isAuthenticated.value && !!workspaceId.value),
})

const workspace = computed(() => workspaceQuery.data.value ?? null)
const loading = computed(() => workspaceQuery.isLoading.value || settingsQuery.isLoading.value)

const name = ref('')
const maxJobs = ref(0)
const saving = ref(false)
const deleting = ref(false)
const error = ref('')
const showDeleteConfirm = ref(false)
const savingRetention = ref(false)

watch(() => workspaceQuery.data.value, (ws) => {
  name.value = ws?.name ?? ''
}, { immediate: true })
watch(() => settingsQuery.data.value, (value) => {
  maxJobs.value = value ?? 0
}, { immediate: true })
watch([workspaceQuery.error, settingsQuery.error], ([wsErr, settingsErr]) => {
  const e = wsErr ?? settingsErr
  if (e)
    error.value = getErrorMessage(e) || 'Failed to load workspace'
})

const exportMutation = useExportConfig()
const importMutation = useImportConfig()
const fileInput = ref<HTMLInputElement | null>(null)
const showImportConfirm = ref(false)
const pendingFile = ref<File | null>(null)
const importResult = ref<{ created: number, updated: number, errors: string[] } | null>(null)

watch(workspaceId, () => {
  error.value = ''
  importResult.value = null
})

async function handleSave() {
  if (!auth.currentWorkspaceId.value)
    return
  saving.value = true
  error.value = ''
  try {
    const ws = await updateWorkspace(auth.currentWorkspaceId.value, { name: name.value })
    qc.setQueryData(workspaceKeys.detail(auth.currentWorkspaceId.value), ws)
    toast.success('Workspace updated')
    await auth.fetchCurrentUser()
  }
  catch (e: unknown) {
    error.value = getErrorMessage(e) || 'Failed to update workspace'
  }
  finally {
    saving.value = false
  }
}

async function handleSaveRetention() {
  if (!auth.currentWorkspaceId.value)
    return
  savingRetention.value = true
  error.value = ''
  try {
    await connectionClient.updatePipelineSettings({
      maxJobsPerWorkspace: maxJobs.value,
    })
    await settingsQuery.refetch()
    toast.success('Retention settings saved')
  }
  catch (e: unknown) {
    error.value = getErrorMessage(e) || 'Failed to update retention settings'
  }
  finally {
    savingRetention.value = false
  }
}

async function handleDelete() {
  if (!auth.currentWorkspaceId.value)
    return
  deleting.value = true
  try {
    await deleteWorkspace(auth.currentWorkspaceId.value)
    await auth.fetchCurrentUser()
    router.push('/')
  }
  catch (e: unknown) {
    error.value = getErrorMessage(e) || 'Failed to delete workspace'
  }
  finally {
    deleting.value = false
    showDeleteConfirm.value = false
  }
}

async function handleExport() {
  try {
    const blob = await exportMutation.mutateAsync()
    const wsName = workspace.value?.name || 'workspace'
    const date = new Date().toISOString().slice(0, 10)
    const url = URL.createObjectURL(blob)
    const a = document.createElement('a')
    a.href = url
    a.download = `synclet-config-${wsName}-${date}.yaml`
    a.click()
    URL.revokeObjectURL(url)
    toast.success('Configuration exported')
  }
  catch (e: unknown) {
    error.value = getErrorMessage(e) || 'Failed to export configuration'
  }
}

function handleImportClick() {
  fileInput.value?.click()
}

function handleFileSelect(event: Event) {
  const input = event.target as HTMLInputElement
  const file = input.files?.[0]
  if (!file)
    return
  pendingFile.value = file
  showImportConfirm.value = true
  input.value = ''
}

async function handleImportConfirm() {
  if (!pendingFile.value)
    return
  showImportConfirm.value = false
  try {
    const result = await importMutation.mutateAsync(pendingFile.value)
    importResult.value = result
    if (result.errors.length === 0) {
      toast.success(`Imported: ${result.created} created, ${result.updated} updated`)
    }
  }
  catch (e: unknown) {
    error.value = getErrorMessage(e) || 'Failed to import configuration'
  }
  finally {
    pendingFile.value = null
  }
}
</script>

<template>
  <div class="space-y-6">
    <PageHeader title="General" :level="2" />
    <div v-if="loading" class="space-y-6">
      <SSkeleton variant="rect" height="160px" />
      <SSkeleton variant="rect" height="120px" />
    </div>
    <template v-else>
      <SAlert v-if="!isAdmin" variant="info">
        You need admin permissions to change workspace settings.
      </SAlert>

      <SCard title="Workspace Settings">
        <SAlert v-if="error" variant="danger" class="mb-4" dismissible @dismiss="error = ''">
          {{ error }}
        </SAlert>
        <form class="space-y-4" @submit.prevent="handleSave">
          <SInput v-model="name" label="Workspace name" required :disabled="!isAdmin" />
          <SButton v-if="isAdmin" type="submit" :loading="saving">
            Save
          </SButton>
        </form>
      </SCard>

      <SCard v-if="isAdmin" title="Configuration">
        <p class="text-sm text-text-secondary mb-4">
          Export or import workspace configuration as YAML. Secrets are replaced with placeholders on export.
        </p>
        <div class="flex flex-wrap gap-3">
          <SButton variant="secondary" :loading="exportMutation.isPending.value" @click="handleExport">
            <Download class="w-4 h-4 mr-2" />
            Export Configuration
          </SButton>
          <SButton variant="secondary" :loading="importMutation.isPending.value" @click="handleImportClick">
            <Upload class="w-4 h-4 mr-2" />
            Import Configuration
          </SButton>
          <input ref="fileInput" type="file" accept=".yaml,.yml" class="hidden" @change="handleFileSelect">
        </div>

        <SAlert v-if="importResult && importResult.errors.length === 0" variant="success" class="mt-4" dismissible @dismiss="importResult = null">
          Imported: {{ importResult.created }} created, {{ importResult.updated }} updated
        </SAlert>
        <SAlert v-if="importResult && importResult.errors.length > 0" variant="danger" class="mt-4" dismissible @dismiss="importResult = null">
          <p>Import completed with errors ({{ importResult.created }} created, {{ importResult.updated }} updated):</p>
          <ul class="list-disc list-inside mt-1">
            <li v-for="(err, i) in importResult.errors" :key="i">
              {{ err }}
            </li>
          </ul>
        </SAlert>
      </SCard>

      <SCard title="Job Retention">
        <p class="text-sm text-text-secondary mb-4">
          Configure how many completed, failed, and cancelled jobs to keep per workspace. Older jobs beyond this limit are automatically deleted. Set to 0 for unlimited.
        </p>
        <SInput
          v-model.number="maxJobs"
          label="Max jobs to keep"
          type="number"
          :min="0"
          placeholder="0 (unlimited)"
          :disabled="!isAdmin"
        />
        <SButton v-if="isAdmin" :loading="savingRetention" class="mt-4" @click="handleSaveRetention">
          Save
        </SButton>
      </SCard>

      <SCard v-if="isAdmin">
        <h3 class="text-lg font-semibold text-danger mb-2">
          Danger Zone
        </h3>
        <p class="text-sm text-text-secondary mb-4">
          Deleting a workspace is permanent and cannot be undone. All data will be lost.
        </p>
        <SButton variant="danger" @click="showDeleteConfirm = true">
          Delete workspace
        </SButton>
      </SCard>
    </template>

    <SConfirmDialog
      :open="showDeleteConfirm"
      title="Delete workspace"
      message="Are you sure? This action is permanent and cannot be undone. All data will be lost."
      confirm-text="Delete"
      :loading="deleting"
      @confirm="handleDelete"
      @cancel="showDeleteConfirm = false"
    />

    <SConfirmDialog
      :open="showImportConfirm"
      title="Import configuration"
      message="This will create or update sources, destinations, and connections. Existing items with matching names will be updated."
      confirm-text="Import"
      @confirm="handleImportConfirm"
      @cancel="showImportConfirm = false; pendingFile = null"
    />
  </div>
</template>

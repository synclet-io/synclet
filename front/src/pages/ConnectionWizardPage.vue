<script setup lang="ts">
import { configuredCatalogToStreams, getConfiguredCatalog, getConnection, useConfigureStreams, useCreateConnection } from '@entities/connection'
import { getDestination } from '@entities/destination/api'
import { getSource } from '@entities/source/api'
import { tagFieldError } from '@features/connection-tags/tagError'
import StepReview from '@features/connection-wizard/StepReview.vue'
import StepSelectDestination from '@features/connection-wizard/StepSelectDestination.vue'
import StepSelectSource from '@features/connection-wizard/StepSelectSource.vue'
import StepSettings from '@features/connection-wizard/StepSettings.vue'
import StepStreamConfig from '@features/connection-wizard/StepStreamConfig.vue'
import { useWizardState, WIZARD_STEPS } from '@features/connection-wizard/useWizardState'
import WizardStepper from '@features/connection-wizard/WizardStepper.vue'
import { getErrorMessage } from '@shared/lib/errorUtils'
import { PageHeader, SAlert, SButton, SCard, SConfirmDialog, SSkeleton } from '@shared/ui'
import { onMounted, onUnmounted, ref } from 'vue'
import { onBeforeRouteLeave, useRoute, useRouter } from 'vue-router'

const route = useRoute()
const router = useRouter()
const { currentStep, state, canProceed, hasUnsavedChanges, committedConnectionId, markCommitted, next, back, goTo, preselect } = useWizardState()
const createConnectionMutation = useCreateConnection()
const configureStreamsMutation = useConfigureStreams()
const submitting = ref(false)
const error = ref('')
const tagsError = ref('')
const redirectTimer = ref<ReturnType<typeof setTimeout> | null>(null)
const hydrating = ref(false)
const skipLeaveGuard = ref(false)
const resolveLeave = ref<((leave: boolean) => void) | null>(null)
const cloneFromId = ref<string>(typeof route.query.cloneFrom === 'string' ? route.query.cloneFrom : '')
const preselectSourceId = typeof route.query.sourceId === 'string' ? route.query.sourceId : ''
const preselectDestinationId = typeof route.query.destinationId === 'string' ? route.query.destinationId : ''

async function applyPreselection() {
  if (!preselectSourceId && !preselectDestinationId)
    return
  hydrating.value = true
  try {
    const [source, destination] = await Promise.all([
      preselectSourceId ? getSource(preselectSourceId).catch(() => undefined) : undefined,
      preselectDestinationId ? getDestination(preselectDestinationId).catch(() => undefined) : undefined,
    ])
    preselect({ source, destination })
  }
  finally {
    hydrating.value = false
  }
}

onMounted(async () => {
  if (!cloneFromId.value) {
    await applyPreselection()
    return
  }
  hydrating.value = true
  try {
    const [src, configuredCat] = await Promise.all([
      getConnection(cloneFromId.value),
      getConfiguredCatalog(cloneFromId.value).catch(() => undefined),
    ])
    if (!src) {
      error.value = 'Could not load the connection to duplicate.'
      return
    }
    const [source, destination] = await Promise.all([
      getSource(src.sourceId),
      getDestination(src.destinationId),
    ])
    state.value = {
      ...state.value,
      sourceId: src.sourceId,
      source: source ?? null,
      destinationId: src.destinationId,
      destination: destination ?? null,
      name: `${src.name} (copy)`,
      schedule: src.schedule,
      schemaChangePolicy: src.schemaChangePolicy,
      maxAttempts: src.maxAttempts,
      namespaceDefinition: src.namespaceDefinition,
      customNamespaceFormat: src.customNamespaceFormat,
      streamPrefix: src.streamPrefix,
      tags: [...src.tags],
      discoveredCatalog: configuredCat ?? null,
      streams: configuredCatalogToStreams(configuredCat),
      streamsSkipped: false,
    }
    // Skip ahead to Settings — source/destination are already chosen.
    goTo(3, { force: true })
  }
  catch (e: unknown) {
    error.value = getErrorMessage(e) || 'Failed to duplicate connection.'
  }
  finally {
    hydrating.value = false
  }
})

onUnmounted(() => {
  if (redirectTimer.value) {
    clearTimeout(redirectTimer.value)
  }
})

onBeforeRouteLeave(() => {
  if (skipLeaveGuard.value || submitting.value || !hasUnsavedChanges.value)
    return true
  return new Promise<boolean>((resolve) => {
    resolveLeave.value = resolve
  })
})

function settleLeave(leave: boolean) {
  resolveLeave.value?.(leave)
  resolveLeave.value = null
}

function handleAutoAdvance() {
  next()
}

function cancelWizard() {
  skipLeaveGuard.value = true
  router.push('/connections')
}

async function handleCreate() {
  if (committedConnectionId.value)
    return
  submitting.value = true
  error.value = ''
  tagsError.value = ''
  try {
    const connection = await createConnectionMutation.mutateAsync({
      name: state.value.name,
      sourceId: state.value.sourceId,
      destinationId: state.value.destinationId,
      schedule: state.value.schedule || undefined,
      schemaChangePolicy: state.value.schemaChangePolicy,
      maxAttempts: state.value.maxAttempts,
      namespaceDefinition: state.value.namespaceDefinition,
      customNamespaceFormat: state.value.customNamespaceFormat || undefined,
      streamPrefix: state.value.streamPrefix || undefined,
      tags: state.value.tags,
    })
    if (!connection)
      throw new Error('Failed to create connection')
    markCommitted(connection.id)

    // Configure streams if any were selected
    if (state.value.streams.length > 0 && !state.value.streamsSkipped) {
      try {
        await configureStreamsMutation.mutateAsync({
          connectionId: connection.id,
          streams: state.value.streams,
        })
      }
      catch {
        error.value = 'Connection created but stream configuration failed. Redirecting to stream settings...'
        submitting.value = false
        redirectTimer.value = setTimeout(() => router.push(`/connections/${connection.id}/streams`), 3000)
        return
      }
    }

    // Full success: redirect to connection detail
    router.push(`/connections/${connection.id}`)
  }
  catch (e: unknown) {
    tagsError.value = tagFieldError(e)
    if (!tagsError.value)
      error.value = getErrorMessage(e) || 'Failed to create connection. Please try again.'
    submitting.value = false
  }
}

function updateTags(tags: string[]) {
  state.value.tags = tags
  tagsError.value = ''
}

function handleSkipStreams() {
  state.value.streamsSkipped = true
  next()
}
</script>

<template>
  <div>
    <PageHeader
      :title="cloneFromId ? 'Duplicate Connection' : 'New Connection'"
      :description="cloneFromId ? 'Review settings, then save as a new connection' : 'Set up a data pipeline in a few steps'"
    />

    <SCard class="max-w-4xl mx-auto">
      <div v-if="hydrating" class="space-y-4">
        <SSkeleton variant="rect" height="40px" />
        <SSkeleton variant="rect" height="120px" />
        <SSkeleton variant="rect" height="80px" />
      </div>
      <template v-else>
        <WizardStepper :current-step="currentStep" :steps="WIZARD_STEPS" @go-to="goTo" />

        <div class="min-h-[300px]">
          <StepSelectSource
            v-if="currentStep === 1"
            :source-id="state.sourceId"
            @update:source-id="state.sourceId = $event"
            @update:source="state.source = $event"
            @auto-advance="handleAutoAdvance"
          />

          <StepSelectDestination
            v-if="currentStep === 2"
            :destination-id="state.destinationId"
            @update:destination-id="state.destinationId = $event"
            @update:destination="state.destination = $event"
            @auto-advance="handleAutoAdvance"
          />

          <StepSettings
            v-if="currentStep === 3"
            :name="state.name"
            :schedule="state.schedule"
            :schema-change-policy="state.schemaChangePolicy"
            :max-attempts="state.maxAttempts"
            :namespace-definition="state.namespaceDefinition"
            :custom-namespace-format="state.customNamespaceFormat"
            :stream-prefix="state.streamPrefix"
            @update:name="state.name = $event"
            @update:schedule="state.schedule = $event"
            @update:schema-change-policy="state.schemaChangePolicy = $event"
            @update:max-attempts="state.maxAttempts = $event"
            @update:namespace-definition="state.namespaceDefinition = $event"
            @update:custom-namespace-format="state.customNamespaceFormat = $event"
            @update:stream-prefix="state.streamPrefix = $event"
          />

          <StepStreamConfig
            v-if="currentStep === 4"
            :source-id="state.sourceId"
            :streams="state.streams"
            :discovered-catalog="state.discoveredCatalog"
            @update:streams="state.streams = $event"
            @update:discovered-catalog="state.discoveredCatalog = $event"
            @skip="handleSkipStreams"
          />

          <StepReview
            v-if="currentStep === 5"
            :state="state"
            :tags-error="tagsError"
            @go-to="goTo"
            @update:tags="updateTags"
          />
        </div>

        <SAlert v-if="error" variant="danger" class="mt-4">
          {{ error }}
        </SAlert>

        <!-- Footer -->
        <div class="flex items-center justify-between pt-6 border-t border-border mt-6">
          <div class="flex gap-3">
            <SButton v-if="currentStep > 1" variant="secondary" @click="back">
              Back
            </SButton>
            <SButton variant="secondary" @click="cancelWizard">
              Cancel
            </SButton>
          </div>
          <div class="flex gap-3">
            <SButton v-if="currentStep === 4" variant="secondary" @click="handleSkipStreams">
              Skip
            </SButton>
            <SButton v-if="currentStep === 3 || currentStep === 4" :disabled="!canProceed" @click="next">
              Next
            </SButton>
            <SButton v-if="currentStep === 5" :loading="submitting" :disabled="!!committedConnectionId" @click="handleCreate">
              Create Connection
            </SButton>
          </div>
        </div>
      </template>
    </SCard>

    <SConfirmDialog
      :open="!!resolveLeave"
      title="Leave without creating?"
      message="The connection has not been created yet. Leaving discards the source, destination and settings you picked."
      confirm-text="Leave"
      cancel-text="Stay"
      @confirm="settleLeave(true)"
      @cancel="settleLeave(false)"
    />
  </div>
</template>

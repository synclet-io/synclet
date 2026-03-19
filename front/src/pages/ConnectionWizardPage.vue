<script setup lang="ts">
import { useConfigureStreams, useCreateConnection } from '@entities/connection'
import StepReview from '@features/connection-wizard/StepReview.vue'
import StepSelectDestination from '@features/connection-wizard/StepSelectDestination.vue'
import StepSelectSource from '@features/connection-wizard/StepSelectSource.vue'
import StepSettings from '@features/connection-wizard/StepSettings.vue'
import StepStreamConfig from '@features/connection-wizard/StepStreamConfig.vue'
import { useWizardState, WIZARD_STEPS } from '@features/connection-wizard/useWizardState'
import WizardStepper from '@features/connection-wizard/WizardStepper.vue'
import { getErrorMessage } from '@shared/lib/errorUtils'
import { PageHeader, SAlert, SButton, SCard } from '@shared/ui'
import { onUnmounted, ref } from 'vue'
import { onBeforeRouteLeave, useRouter } from 'vue-router'

const router = useRouter()
const { currentStep, state, canProceed, hasAnyState, next, back, goTo } = useWizardState()
const createConnectionMutation = useCreateConnection()
const configureStreamsMutation = useConfigureStreams()
const submitting = ref(false)
const error = ref('')
const redirectTimer = ref<ReturnType<typeof setTimeout> | null>(null)

onUnmounted(() => {
  if (redirectTimer.value) {
    clearTimeout(redirectTimer.value)
  }
})

onBeforeRouteLeave((_to, _from, nextGuard) => {
  if (hasAnyState.value && !submitting.value) {
    // eslint-disable-next-line no-alert
    const answer = window.confirm('You have unsaved changes. Leave this page?')
    if (!answer)
      return nextGuard(false)
  }
  nextGuard()
})

function handleAutoAdvance() {
  next()
}

function discardDraft() {
  router.push('/connections')
}

async function handleCreate() {
  submitting.value = true
  error.value = ''
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
    })
    if (!connection)
      throw new Error('Failed to create connection')

    // Configure streams if any were selected
    if (state.value.streams.length > 0 && !state.value.streamsSkipped) {
      try {
        await configureStreamsMutation.mutateAsync({
          connectionId: connection.id,
          streams: state.value.streams,
        })
      }
      catch {
        // Partial failure: connection created but streams failed
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
    error.value = getErrorMessage(e) || 'Failed to create connection. Please try again.'
    submitting.value = false
  }
}

function handleSkipStreams() {
  state.value.streamsSkipped = true
  next()
}
</script>

<template>
  <div>
    <PageHeader title="New Connection" description="Set up a data pipeline in a few steps" />

    <SCard class="max-w-4xl mx-auto">
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
          @go-to="goTo"
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
          <SButton variant="ghost" @click="discardDraft">
            Discard Draft
          </SButton>
        </div>
        <div class="flex gap-3">
          <SButton v-if="currentStep === 4" variant="secondary" @click="handleSkipStreams">
            Skip
          </SButton>
          <SButton v-if="currentStep === 3 || currentStep === 4" :disabled="!canProceed" @click="next">
            Next
          </SButton>
          <SButton v-if="currentStep === 5" :loading="submitting" @click="handleCreate">
            Create Connection
          </SButton>
        </div>
      </div>
    </SCard>
  </div>
</template>

<script setup lang="ts">
import { requestPasswordReset } from '@entities/auth'
import { getErrorMessage } from '@shared/lib/errorUtils'
import { SAlert, SButton, SInput, SyncletLogo } from '@shared/ui'
import { ref } from 'vue'
import { useRoute } from 'vue-router'

const route = useRoute()
const email = ref((route.query.email as string) || '')
const error = ref('')
const loading = ref(false)
const sent = ref(false)

async function handleSubmit() {
  loading.value = true
  error.value = ''
  try {
    await requestPasswordReset(email.value)
    sent.value = true
  }
  catch (e: unknown) {
    error.value = getErrorMessage(e) || 'Could not request a password reset'
  }
  finally {
    loading.value = false
  }
}
</script>

<template>
  <div class="min-h-screen flex">
    <!-- Left panel - branding -->
    <div class="hidden lg:flex lg:w-[480px] bg-slate-900 flex-col justify-between p-10">
      <div class="flex items-center gap-2.5">
        <SyncletLogo size="w-8 h-8" />
        <span class="text-lg font-semibold text-white tracking-tight">Synclet</span>
      </div>
      <div>
        <h2 class="text-2xl font-semibold text-white leading-snug">
          Locked out?<br>It happens.
        </h2>
        <p class="mt-3 text-sm text-slate-400 leading-relaxed max-w-sm">
          We'll email you a link to choose a new password.
        </p>
      </div>
      <p class="text-xs text-slate-400">
        &copy; {{ new Date().getFullYear() }} Synclet
      </p>
    </div>

    <!-- Right panel - form -->
    <div class="flex-1 flex items-center justify-center bg-page p-6">
      <div class="w-full max-w-sm">
        <div class="flex items-center gap-2.5 mb-10 lg:hidden">
          <SyncletLogo size="w-8 h-8" />
          <span class="text-lg font-semibold text-heading tracking-tight">Synclet</span>
        </div>

        <div class="mb-8">
          <h1 class="text-xl font-semibold text-heading">
            Forgot your password?
          </h1>
          <p class="mt-1 text-sm text-text-secondary">
            Enter your email and we'll send you a reset link
          </p>
        </div>

        <SAlert v-if="sent" variant="success">
          If an account exists, we sent a link to reset your password.
        </SAlert>

        <form v-else class="space-y-5" @submit.prevent="handleSubmit">
          <SAlert v-if="error" variant="danger" dismissible @dismiss="error = ''">
            {{ error }}
          </SAlert>
          <SInput v-model="email" label="Email" type="email" placeholder="you@example.com" required />
          <SButton type="submit" :loading="loading" class="w-full">
            Send reset link
          </SButton>
        </form>

        <p class="mt-6 text-center text-sm text-text-secondary">
          Remembered it?
          <RouterLink :to="{ name: 'login' }" class="text-primary hover:text-primary-hover font-medium ml-1">
            Sign in
          </RouterLink>
        </p>
      </div>
    </div>
  </div>
</template>

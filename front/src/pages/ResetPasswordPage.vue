<script setup lang="ts">
import { resetPassword, useAuth } from '@entities/auth'
import { getErrorMessage } from '@shared/lib/errorUtils'
import { SAlert, SButton, SInput, SyncletLogo } from '@shared/ui'
import { computed, ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'

const route = useRoute()
const router = useRouter()
const auth = useAuth()
const token = computed(() => (typeof route.query.token === 'string' ? route.query.token : ''))
const password = ref('')
const confirmPassword = ref('')
const error = ref('')
const loading = ref(false)

async function handleSubmit() {
  error.value = ''
  if (password.value !== confirmPassword.value) {
    error.value = 'Passwords do not match'
    return
  }

  loading.value = true
  try {
    await resetPassword(token.value, password.value)
    if (auth.isAuthenticated.value)
      await auth.logout()
    router.push({ name: 'login' })
  }
  catch (e: unknown) {
    error.value = getErrorMessage(e) || 'Could not reset the password'
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
          Choose a new<br>password.
        </h2>
        <p class="mt-3 text-sm text-slate-400 leading-relaxed max-w-sm">
          Resetting your password signs you out of every session.
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
            Reset your password
          </h1>
        </div>

        <div v-if="!token" class="space-y-5">
          <SAlert variant="danger">
            This reset link is invalid or has expired.
          </SAlert>
          <p class="text-center text-sm text-text-secondary">
            <RouterLink :to="{ name: 'forgot-password' }" class="text-primary hover:text-primary-hover font-medium">
              Request a new link
            </RouterLink>
          </p>
        </div>

        <form v-else class="space-y-5" @submit.prevent="handleSubmit">
          <SAlert v-if="error" variant="danger" dismissible @dismiss="error = ''">
            {{ error }}
          </SAlert>
          <SInput v-model="password" label="New password" type="password" placeholder="Choose a new password" required />
          <SInput v-model="confirmPassword" label="Confirm password" type="password" placeholder="Repeat the new password" required />
          <SButton type="submit" :loading="loading" class="w-full">
            Reset password
          </SButton>
        </form>

        <p class="mt-6 text-center text-sm text-text-secondary">
          <RouterLink :to="{ name: 'login' }" class="text-primary hover:text-primary-hover font-medium">
            Back to sign in
          </RouterLink>
        </p>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { useAuth } from '@entities/auth'
import { useSystemInfo } from '@entities/system'
import { getErrorMessage } from '@shared/lib/errorUtils'
import { SAlert, SButton, SInput } from '@shared/ui'
import { Zap } from 'lucide-vue-next'
import { ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'

const route = useRoute()
const router = useRouter()
const auth = useAuth()
const { data: systemInfo } = useSystemInfo()
const name = ref('')
const email = ref((route.query.email as string) || '')
const password = ref('')
const error = ref('')
const loading = ref(false)

// Redirect to login if registration is disabled.
watch(systemInfo, (info) => {
  if (info && !info.registrationEnabled) {
    router.replace('/login')
  }
}, { immediate: true })

async function handleRegister() {
  loading.value = true
  error.value = ''
  try {
    await auth.register(email.value, password.value, name.value)
    const pendingInviteToken = localStorage.getItem('pendingInviteToken')
    localStorage.removeItem('pendingInviteToken')
    const uuidRegex = /^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$/i
    if (pendingInviteToken && uuidRegex.test(pendingInviteToken)) {
      router.push(`/invite/${pendingInviteToken}`)
    }
    else {
      router.push('/')
    }
  }
  catch (e: unknown) {
    error.value = getErrorMessage(e) || 'Registration failed'
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
        <div class="w-8 h-8 bg-primary rounded-lg flex items-center justify-center">
          <Zap class="w-4.5 h-4.5 text-white" />
        </div>
        <span class="text-lg font-semibold text-white tracking-tight">Synclet</span>
      </div>
      <div>
        <h2 class="text-2xl font-semibold text-white leading-snug">
          Get started<br>in minutes.
        </h2>
        <p class="mt-3 text-sm text-slate-400 leading-relaxed max-w-sm">
          Create an account to start building data pipelines. No credit card required.
        </p>
      </div>
      <p class="text-xs text-slate-500">
        &copy; {{ new Date().getFullYear() }} Synclet
      </p>
    </div>

    <!-- Right panel - form -->
    <div class="flex-1 flex items-center justify-center bg-page p-6">
      <div class="w-full max-w-sm">
        <div class="flex items-center gap-2.5 mb-10 lg:hidden">
          <div class="w-8 h-8 bg-primary rounded-lg flex items-center justify-center">
            <Zap class="w-4.5 h-4.5 text-white" />
          </div>
          <span class="text-lg font-semibold text-heading tracking-tight">Synclet</span>
        </div>

        <div class="mb-8">
          <h1 class="text-xl font-semibold text-heading">
            Create your account
          </h1>
          <p class="mt-1 text-sm text-text-secondary">
            Start syncing your data in minutes
          </p>
        </div>

        <form class="space-y-5" @submit.prevent="handleRegister">
          <SAlert v-if="error" variant="danger" dismissible @dismiss="error = ''">
            {{ error }}
          </SAlert>
          <SInput v-model="name" label="Name" type="text" placeholder="Your name" required />
          <SInput v-model="email" label="Email" type="email" placeholder="you@example.com" required />
          <SInput v-model="password" label="Password" type="password" placeholder="Choose a password" required />
          <SButton type="submit" :loading="loading" class="w-full">
            Create account
          </SButton>
        </form>

        <p class="mt-6 text-center text-sm text-text-secondary">
          Already have an account?
          <RouterLink to="/login" class="text-primary hover:text-primary-hover font-medium ml-1">
            Sign in
          </RouterLink>
        </p>
      </div>
    </div>
  </div>
</template>

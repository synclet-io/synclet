<script setup lang="ts">
import type { OIDCProvider } from '@entities/auth'
import { getOIDCProviders, useAuth } from '@entities/auth'
import { useSystemInfo } from '@entities/system'
import { getErrorMessage } from '@shared/lib/errorUtils'
import { SAlert, SButton, SInput } from '@shared/ui'
import { Zap } from 'lucide-vue-next'
import { onMounted, ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'

const route = useRoute()
const router = useRouter()
const auth = useAuth()
const email = ref((route.query.email as string) || '')
const password = ref('')
const error = ref('')
const loading = ref(false)
const oidcProviders = ref<OIDCProvider[]>([])
const { data: systemInfo } = useSystemInfo()

onMounted(async () => {
  try {
    oidcProviders.value = await getOIDCProviders()
  }
  catch {
    // OIDC not configured, ignore.
  }
})

async function handleLogin() {
  loading.value = true
  error.value = ''
  try {
    await auth.login(email.value, password.value)
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
    error.value = getErrorMessage(e) || 'Login failed'
  }
  finally {
    loading.value = false
  }
}

function startOIDCLogin(slug: string) {
  window.location.href = `/auth/oidc/${slug}/login`
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
          Data sync,<br>simplified.
        </h2>
        <p class="mt-3 text-sm text-slate-400 leading-relaxed max-w-sm">
          Connect your data sources and destinations with a few clicks. Manage pipelines, monitor jobs, and keep everything in sync.
        </p>
      </div>
      <p class="text-xs text-slate-500">
        &copy; {{ new Date().getFullYear() }} Synclet
      </p>
    </div>

    <!-- Right panel - form -->
    <div class="flex-1 flex items-center justify-center bg-page p-6">
      <div class="w-full max-w-sm">
        <!-- Mobile logo -->
        <div class="flex items-center gap-2.5 mb-10 lg:hidden">
          <div class="w-8 h-8 bg-primary rounded-lg flex items-center justify-center">
            <Zap class="w-4.5 h-4.5 text-white" />
          </div>
          <span class="text-lg font-semibold text-heading tracking-tight">Synclet</span>
        </div>

        <div class="mb-8">
          <h1 class="text-xl font-semibold text-heading">
            Welcome back
          </h1>
          <p class="mt-1 text-sm text-text-secondary">
            Sign in to your account to continue
          </p>
        </div>

        <form class="space-y-5" @submit.prevent="handleLogin">
          <SAlert v-if="error" variant="danger" dismissible @dismiss="error = ''">
            {{ error }}
          </SAlert>
          <SInput v-model="email" label="Email" type="email" placeholder="you@example.com" required />
          <SInput v-model="password" label="Password" type="password" placeholder="Enter your password" required />
          <SButton type="submit" :loading="loading" class="w-full">
            Sign in
          </SButton>
        </form>

        <div v-if="oidcProviders.length > 0" class="mt-6">
          <div class="relative my-6">
            <div class="absolute inset-0 flex items-center">
              <div class="w-full border-t border-border" />
            </div>
            <div class="relative flex justify-center text-xs">
              <span class="bg-page px-2 text-text-secondary">or continue with</span>
            </div>
          </div>
          <div class="space-y-2">
            <SButton
              v-for="provider in oidcProviders"
              :key="provider.slug"
              variant="secondary"
              class="w-full"
              @click="startOIDCLogin(provider.slug)"
            >
              Sign in with {{ provider.displayName }}
            </SButton>
          </div>
        </div>

        <p v-if="systemInfo?.registrationEnabled !== false" class="mt-6 text-center text-sm text-text-secondary">
          Don't have an account?
          <RouterLink to="/register" class="text-primary hover:text-primary-hover font-medium ml-1">
            Create one
          </RouterLink>
        </p>
      </div>
    </div>
  </div>
</template>

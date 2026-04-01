<script setup lang="ts">
import { useAuth } from '@entities/auth'
import { getAuthMeta } from '@entities/auth/token'
import { onMounted, ref } from 'vue'
import { useRouter } from 'vue-router'

const router = useRouter()
const auth = useAuth()
const error = ref('')
const uuidRegex = /^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$/i

onMounted(async () => {
  // Backend set cookies via the redirect — check they exist.
  if (!getAuthMeta()) {
    error.value = 'Authentication failed. No session cookies received.'
    return
  }

  try {
    await auth.fetchCurrentUser()
  }
  catch {
    // User loaded, proceed anyway.
  }

  const pendingInviteToken = localStorage.getItem('pendingInviteToken')
  localStorage.removeItem('pendingInviteToken')
  if (pendingInviteToken && uuidRegex.test(pendingInviteToken)) {
    router.replace(`/invite/${pendingInviteToken}`)
  }
  else {
    router.replace('/')
  }
})
</script>

<template>
  <div class="min-h-screen flex items-center justify-center bg-page">
    <div class="text-center">
      <div v-if="error" class="text-red-500 mb-4">
        {{ error }}
      </div>
      <div v-else class="text-text-secondary">
        Completing sign in...
      </div>
    </div>
  </div>
</template>

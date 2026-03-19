<script setup lang="ts">
import { changePassword, updateProfile, useAuth } from '@entities/auth'
import { getErrorMessage } from '@shared/lib/errorUtils'
import { SAlert, SButton, SCard, SInput, useToast } from '@shared/ui'
import { ref } from 'vue'

const auth = useAuth()
const toast = useToast()

const name = ref(auth.user.value?.name || '')
const savingProfile = ref(false)
const profileError = ref('')

const currentPassword = ref('')
const newPassword = ref('')
const confirmPasswordVal = ref('')
const changingPassword = ref(false)
const passwordError = ref('')

async function handleUpdateProfile() {
  savingProfile.value = true
  profileError.value = ''
  try {
    await updateProfile(name.value)
    await auth.fetchCurrentUser()
    toast.success('Profile updated')
  }
  catch (e: unknown) {
    profileError.value = getErrorMessage(e) || 'Failed to update profile'
  }
  finally {
    savingProfile.value = false
  }
}

async function handleChangePassword() {
  if (newPassword.value !== confirmPasswordVal.value) {
    passwordError.value = 'Passwords do not match'
    return
  }
  changingPassword.value = true
  passwordError.value = ''
  try {
    await changePassword(currentPassword.value, newPassword.value)
    currentPassword.value = ''
    newPassword.value = ''
    confirmPasswordVal.value = ''
    toast.success('Password changed')
  }
  catch (e: unknown) {
    passwordError.value = getErrorMessage(e) || 'Failed to change password'
  }
  finally {
    changingPassword.value = false
  }
}
</script>

<template>
  <div class="max-w-2xl space-y-8">
    <SCard title="Profile">
      <SAlert v-if="profileError" variant="danger" class="mb-4" dismissible @dismiss="profileError = ''">
        {{ profileError }}
      </SAlert>
      <form class="space-y-4" @submit.prevent="handleUpdateProfile">
        <SInput :model-value="auth.user.value?.email" label="Email" type="email" disabled />
        <SInput v-model="name" label="Name" required />
        <SButton type="submit" :loading="savingProfile">
          Save
        </SButton>
      </form>
    </SCard>

    <SCard title="Change Password">
      <SAlert v-if="passwordError" variant="danger" class="mb-4" dismissible @dismiss="passwordError = ''">
        {{ passwordError }}
      </SAlert>
      <form class="space-y-4" @submit.prevent="handleChangePassword">
        <SInput v-model="currentPassword" label="Current password" type="password" required />
        <SInput v-model="newPassword" label="New password" type="password" required />
        <SInput v-model="confirmPasswordVal" label="Confirm new password" type="password" required />
        <SButton type="submit" :loading="changingPassword">
          Change password
        </SButton>
      </form>
    </SCard>
  </div>
</template>

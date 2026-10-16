import type { User, WorkspaceMembership } from './types'
import { listWorkspaces } from '@entities/workspace/api'
import { computed, readonly, ref } from 'vue'
import * as authApi from './api'
import { clearWorkspaceId, getAuthMeta, getWorkspaceId, setWorkspaceId } from './token'

const user = ref<User | null>(null)
const workspaces = ref<WorkspaceMembership[]>([])
const workspacesLoaded = ref(false)
const currentWorkspaceId = ref<string | null>(getWorkspaceId())
const hasSession = ref(getAuthMeta() !== null)

const isAuthenticated = computed(() => hasSession.value)

async function login(email: string, password: string) {
  const result = await authApi.login(email, password)
  hasSession.value = true
  user.value = result.user ?? null
  setWorkspaces(result.workspaces)
}

async function register(email: string, password: string, name: string) {
  const result = await authApi.register(email, password, name)
  hasSession.value = true
  user.value = result.user ?? null
  setWorkspaces(result.workspaces)
}

async function logout() {
  try {
    await authApi.logout()
  }
  catch { /* best-effort */ }
  clearState()
}

async function refreshAccessToken(): Promise<boolean> {
  try {
    await authApi.refreshToken()
    return true
  }
  catch {
    clearState()
    return false
  }
}

async function fetchCurrentUser() {
  const result = await authApi.getCurrentUser()
  user.value = result.user ?? null
  setWorkspaces(result.workspaces)
}

async function reloadWorkspaces() {
  const ws = await listWorkspaces()
  setWorkspaces(ws.map(({ workspace, role }) => ({
    workspaceId: workspace.id,
    workspaceName: workspace.name,
    role,
  })))
}

function setWorkspaces(ws: WorkspaceMembership[]) {
  workspaces.value = ws
  workspacesLoaded.value = true
  const saved = getWorkspaceId()
  const validSaved = saved && ws.some(w => w.workspaceId === saved)
  if (validSaved) {
    currentWorkspaceId.value = saved
  }
  else if (ws.length > 0) {
    switchWorkspace(ws[0].workspaceId)
  }
  else {
    clearWorkspaceId()
    currentWorkspaceId.value = null
  }
}

function switchWorkspace(id: string) {
  currentWorkspaceId.value = id
  setWorkspaceId(id)
}

function clearState() {
  clearWorkspaceId()
  hasSession.value = false
  user.value = null
  workspaces.value = []
  workspacesLoaded.value = false
  currentWorkspaceId.value = null
}

async function init() {
  const meta = getAuthMeta()
  if (!meta)
    return

  try {
    await fetchCurrentUser()
  }
  catch {
    // Token invalid or expired — try refresh
    if (meta.refreshExpiresAt > Date.now() / 1000) {
      const refreshed = await refreshAccessToken()
      if (refreshed) {
        try {
          await fetchCurrentUser()
        }
        catch {
          clearState()
        }
        return
      }
    }
    clearState()
  }
}

export function useAuth() {
  return {
    user: readonly(user),
    workspaces: readonly(workspaces),
    workspacesLoaded: readonly(workspacesLoaded),
    isAuthenticated,
    currentWorkspaceId: readonly(currentWorkspaceId),
    login,
    register,
    logout,
    refreshAccessToken,
    fetchCurrentUser,
    reloadWorkspaces,
    setWorkspaces,
    switchWorkspace,
    init,
  }
}

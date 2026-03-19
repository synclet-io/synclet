import { useAuth } from '@entities/auth'
import { getSystemInfo } from '@entities/system/api'
import { listWorkspaces } from '@entities/workspace/api'
import { setOnAuthFailure } from '@shared/api/transport'
import { createRouter, createWebHistory } from 'vue-router'

export const router = createRouter({
  history: createWebHistory(),
  routes: [
    {
      path: '/login',
      name: 'login',
      component: () => import('@pages/LoginPage.vue'),
      meta: { public: true },
    },
    {
      path: '/register',
      name: 'register',
      component: () => import('@pages/RegisterPage.vue'),
      meta: { public: true },
    },
    {
      path: '/auth/oidc/callback',
      name: 'oidc-callback',
      component: () => import('@pages/OIDCCallbackPage.vue'),
      meta: { public: true },
    },
    {
      path: '/invite/:token',
      name: 'invite',
      component: () => import('@pages/InviteAcceptPage.vue'),
      meta: { public: true },
    },
    {
      path: '/create-workspace',
      name: 'create-workspace',
      component: () => import('@pages/CreateWorkspacePage.vue'),
      meta: { requiresAuth: true },
    },
    {
      path: '/',
      component: () => import('@app/layouts/AppLayout.vue'),
      children: [
        {
          path: '',
          name: 'dashboard',
          component: () => import('@pages/DashboardPage.vue'),
        },
        {
          path: 'sources',
          name: 'sources',
          component: () => import('@pages/SourcesPage.vue'),
        },
        {
          path: 'sources/new',
          name: 'source-new',
          component: () => import('@pages/SourceSetupPage.vue'),
        },
        {
          path: 'sources/:id',
          name: 'source-detail',
          component: () => import('@pages/SourceDetailPage.vue'),
        },
        {
          path: 'sources/:id/edit',
          name: 'source-edit',
          component: () => import('@pages/SourceEditPage.vue'),
        },
        {
          path: 'destinations',
          name: 'destinations',
          component: () => import('@pages/DestinationsPage.vue'),
        },
        {
          path: 'destinations/new',
          name: 'destination-new',
          component: () => import('@pages/DestinationSetupPage.vue'),
        },
        {
          path: 'destinations/:id',
          name: 'destination-detail',
          component: () => import('@pages/DestinationDetailPage.vue'),
        },
        {
          path: 'destinations/:id/edit',
          name: 'destination-edit',
          component: () => import('@pages/DestinationEditPage.vue'),
        },
        {
          path: 'connections',
          name: 'connections',
          component: () => import('@pages/ConnectionsPage.vue'),
        },
        {
          path: 'connections/new',
          name: 'connection-new',
          component: () => import('@pages/ConnectionWizardPage.vue'),
        },
        {
          path: 'connections/:id',
          component: () => import('@pages/ConnectionLayout.vue'),
          children: [
            {
              path: '',
              name: 'connection-detail',
              component: () => import('@pages/ConnectionDetailPage.vue'),
            },
            {
              path: 'settings',
              name: 'connection-settings',
              component: () => import('@pages/ConnectionSettingsPage.vue'),
            },
            {
              path: 'notifications',
              name: 'connection-notifications',
              component: () => import('@pages/ConnectionNotificationsPage.vue'),
            },
          ],
        },
        {
          path: 'connections/:id/streams',
          name: 'connection-streams',
          component: () => import('@pages/StreamConfigPage.vue'),
        },
        {
          path: 'jobs',
          name: 'jobs',
          component: () => import('@pages/JobsPage.vue'),
        },
        {
          path: 'jobs/:id',
          name: 'job-detail',
          component: () => import('@pages/JobDetailPage.vue'),
        },
        {
          path: 'settings',
          component: () => import('@pages/settings/SettingsLayout.vue'),
          children: [
            {
              path: '',
              redirect: { name: 'settings-general' },
            },
            {
              path: 'general',
              name: 'settings-general',
              component: () => import('@pages/settings/WorkspaceGeneralPage.vue'),
            },
            {
              path: 'members',
              name: 'settings-members',
              component: () => import('@pages/settings/WorkspaceMembersPage.vue'),
            },
            {
              path: 'connectors',
              name: 'settings-connectors',
              component: () => import('@pages/ConnectorBrowserPage.vue'),
            },
            {
              path: 'notifications',
              name: 'settings-notifications',
              component: () => import('@pages/settings/NotificationsPage.vue'),
            },
            {
              path: 'api-keys',
              name: 'settings-api-keys',
              component: () => import('@pages/settings/APIKeysPage.vue'),
            },
            {
              path: 'account',
              name: 'settings-account',
              component: () => import('@pages/settings/AccountPage.vue'),
            },
          ],
        },
      ],
    },
  ],
})

let initialized = false
let cachedSystemInfo: Awaited<ReturnType<typeof getSystemInfo>> | null = null

router.beforeEach(async (to) => {
  const auth = useAuth()

  // Initialize auth state on first navigation.
  if (!initialized) {
    initialized = true
    await auth.init()
    setOnAuthFailure(() => {
      auth.logout()
      router.push('/login')
    })
  }

  // Allow public routes without auth.
  if (to.meta.public) {
    // Redirect authenticated users away from login/register.
    if (auth.isAuthenticated.value && (to.name === 'login' || to.name === 'register')) {
      return { name: 'dashboard' }
    }
    return
  }

  // Require authentication for all other routes.
  if (!auth.isAuthenticated.value) {
    return { name: 'login', query: { redirect: to.fullPath } }
  }

  // Load system info if not cached.
  if (!cachedSystemInfo) {
    cachedSystemInfo = await getSystemInfo()
  }

  // Block workspace-management routes in single-workspace mode (D-09).
  if (cachedSystemInfo?.workspacesMode === 'single') {
    const blockedRoutes = ['create-workspace', 'invite', 'settings-members']
    if (blockedRoutes.includes(to.name as string)) {
      return { name: 'dashboard' }
    }
  }

  // Load workspaces if not yet loaded.
  if (auth.workspaces.value.length === 0) {
    const ws = await listWorkspaces()
    // TODO(W8): ListWorkspaces API does not return member role. Need to either
    // extend the proto to include role per workspace, or make a separate
    // ListMembers call per workspace. Defaulting to 'admin' so admin-only UI
    // is not incorrectly hidden. The RoleInterceptor on the backend enforces
    // actual permissions regardless of what the frontend displays.
    auth.setWorkspaces(ws.map(w => ({ workspaceId: w.id, workspaceName: w.name, role: 'admin' })))
  }

  // Redirect to workspace creation if user has no workspaces.
  if (auth.workspaces.value.length === 0 && to.name !== 'create-workspace') {
    // In single-workspace mode, default workspace always exists -- don't redirect.
    if (cachedSystemInfo?.workspacesMode !== 'single') {
      return { name: 'create-workspace' }
    }
  }
})

<script setup lang="ts">
import type { DropdownItem } from '@shared/ui'
import { useAuth } from '@entities/auth'
import { useSystemInfo } from '@entities/system'
import { DarkModeToggle } from '@shared/ui'
import SDropdown from '@shared/ui/SDropdown.vue'
import { ArrowRightLeft, ChevronDown, Database, History, LayoutDashboard, LogOut, Menu, Server, Settings, Zap } from 'lucide-vue-next'
import { computed, ref, watch } from 'vue'
import { RouterLink, RouterView, useRoute, useRouter } from 'vue-router'

const router = useRouter()
const route = useRoute()
const sidebarOpen = ref(false)

watch(() => route.path, () => {
  sidebarOpen.value = false
})
const auth = useAuth()
const { data: systemInfo } = useSystemInfo()
const isSingleWorkspace = computed(() => systemInfo.value?.workspacesMode === 'single')

const navItems = [
  { name: 'Dashboard', to: '/', icon: LayoutDashboard },
  { name: 'Sources', to: '/sources', icon: Database },
  { name: 'Destinations', to: '/destinations', icon: Server },
  { name: 'Connections', to: '/connections', icon: ArrowRightLeft },
  { name: 'Jobs', to: '/jobs', icon: History },
]

async function handleLogout() {
  await auth.logout()
  router.push('/login')
}

const workspaceItems = computed<DropdownItem[]>(() =>
  auth.workspaces.value.map(ws => ({
    label: ws.workspaceName,
    value: ws.workspaceId,
    active: ws.workspaceId === auth.currentWorkspaceId.value,
    onClick: () => auth.switchWorkspace(ws.workspaceId),
  })),
)

const currentWorkspaceName = computed(() =>
  auth.workspaces.value.find(w => w.workspaceId === auth.currentWorkspaceId.value)?.workspaceName || 'Select workspace',
)

const userInitial = computed(() => {
  const name = auth.user.value?.name || auth.user.value?.email || '?'
  return name.charAt(0).toUpperCase()
})
</script>

<template>
  <div class="flex h-screen bg-page">
    <!-- Mobile overlay backdrop -->
    <div
      v-if="sidebarOpen"
      class="fixed inset-0 bg-black/50 z-40 lg:hidden"
      @click="sidebarOpen = false"
    />

    <!-- Sidebar -->
    <aside
      class="fixed inset-y-0 left-0 z-50 w-[240px] bg-sidebar flex flex-col shrink-0 transform transition-transform duration-200 lg:static lg:translate-x-0"
      :class="sidebarOpen ? 'translate-x-0' : '-translate-x-full'"
    >
      <!-- Logo -->
      <div class="px-5 h-14 flex items-center border-b border-sidebar-border">
        <div class="flex items-center gap-2.5">
          <div class="w-7 h-7 bg-primary rounded-lg flex items-center justify-center">
            <Zap class="w-4 h-4 text-white" />
          </div>
          <span class="text-[15px] font-semibold text-sidebar-text-active tracking-tight">Synclet</span>
        </div>
      </div>

      <!-- Navigation -->
      <nav class="flex-1 px-3 py-3 space-y-0.5">
        <RouterLink
          v-for="item in navItems"
          :key="item.name"
          :to="item.to"
          class="flex items-center gap-3 px-3 py-2 rounded-lg text-[13px] font-medium text-sidebar-text hover:bg-sidebar-hover hover:text-sidebar-text-active transition-colors"
          exact-active-class="!bg-sidebar-active !text-sidebar-text-active"
          @click="sidebarOpen = false"
        >
          <component :is="item.icon" class="w-4 h-4 opacity-70" />
          {{ item.name }}
        </RouterLink>
      </nav>

      <!-- Bottom section -->
      <div class="mt-auto border-t border-sidebar-border px-3 py-3 space-y-0.5">
        <div class="flex justify-center py-1">
          <DarkModeToggle />
        </div>
        <RouterLink
          to="/settings"
          class="flex items-center gap-3 px-3 py-2 rounded-lg text-[13px] font-medium text-sidebar-text hover:bg-sidebar-hover hover:text-sidebar-text-active transition-colors"
          exact-active-class="!bg-sidebar-active !text-sidebar-text-active"
          @click="sidebarOpen = false"
        >
          <Settings class="w-4 h-4 opacity-70" />
          Settings
        </RouterLink>

        <!-- Workspace selector -->
        <SDropdown v-if="!isSingleWorkspace && auth.workspaces.value.length > 0" :items="workspaceItems" align="left">
          <template #trigger>
            <button class="w-full flex items-center justify-between px-3 py-2 rounded-lg text-[13px] text-sidebar-text hover:bg-sidebar-hover hover:text-sidebar-text-active transition-colors">
              <span class="truncate">{{ currentWorkspaceName }}</span>
              <ChevronDown class="w-3.5 h-3.5 shrink-0 opacity-50" />
            </button>
          </template>
        </SDropdown>

        <!-- User info + logout -->
        <div class="flex items-center gap-3 px-3 py-2">
          <div class="w-7 h-7 rounded-lg bg-sidebar-active flex items-center justify-center text-xs font-medium text-sidebar-text-active shrink-0">
            {{ userInitial }}
          </div>
          <span class="text-[13px] text-sidebar-text truncate flex-1">{{ auth.user.value?.name || auth.user.value?.email }}</span>
          <button
            class="p-1 text-sidebar-text/50 hover:text-sidebar-text-active transition-colors"
            title="Logout"
            @click="handleLogout"
          >
            <LogOut class="w-3.5 h-3.5" />
          </button>
        </div>
      </div>
    </aside>

    <!-- Main content -->
    <main class="flex-1 overflow-auto">
      <div class="sticky top-0 z-30 flex items-center h-14 px-4 bg-page border-b border-border lg:hidden">
        <button aria-label="Open menu" @click="sidebarOpen = true">
          <Menu class="w-5 h-5 text-heading" />
        </button>
        <div class="ml-3 flex items-center gap-2.5">
          <div class="w-7 h-7 bg-primary rounded-lg flex items-center justify-center">
            <Zap class="w-4 h-4 text-white" />
          </div>
          <span class="text-[15px] font-semibold text-heading tracking-tight">Synclet</span>
        </div>
      </div>
      <div class="px-4 py-4 sm:px-6 sm:py-6 lg:px-8 lg:py-8">
        <RouterView />
      </div>
    </main>
  </div>
</template>

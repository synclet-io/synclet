<script setup lang="ts">
import type { DropdownItem } from '@shared/ui'
import { useAuth } from '@entities/auth'
import { useSystemInfo } from '@entities/system'
import { GlobalSearchModal } from '@features/global-search'
import { DarkModeToggle, SButton, SyncletLogo } from '@shared/ui'
import SDropdown from '@shared/ui/SDropdown.vue'
import { useMediaQuery } from '@vueuse/core'
import { ArrowRightLeft, ChevronDown, Database, History, LayoutDashboard, LogOut, Menu, Network, Search, Server, Settings } from 'lucide-vue-next'
import { computed, nextTick, onMounted, onUnmounted, ref, watch } from 'vue'
import { RouterLink, RouterView, useRoute, useRouter } from 'vue-router'
import { isNavActive } from './isNavActive'

const router = useRouter()
const route = useRoute()
const sidebarOpen = ref(false)
const searchOpen = ref(false)
const isDesktop = useMediaQuery('(min-width: 1024px)')
const sidebarEl = ref<HTMLElement | null>(null)
const menuButton = ref<InstanceType<typeof SButton> | null>(null)

const drawerOpen = computed(() => !isDesktop.value && sidebarOpen.value)
const drawerHidden = computed(() => !isDesktop.value && !sidebarOpen.value)

async function openSidebar() {
  sidebarOpen.value = true
  await nextTick()
  sidebarEl.value?.querySelector<HTMLElement>('a[href], button:not([disabled])')?.focus()
}

async function closeSidebar() {
  sidebarOpen.value = false
  await nextTick()
  ;(menuButton.value?.$el as HTMLElement | undefined)?.focus()
}

watch(() => route.path, () => {
  sidebarOpen.value = false
})

watch(isDesktop, (desktop) => {
  if (desktop)
    sidebarOpen.value = false
})

function onGlobalKeydown(e: KeyboardEvent) {
  if (e.key === 'k' && (e.metaKey || e.ctrlKey)) {
    e.preventDefault()
    searchOpen.value = !searchOpen.value
  }
  else if (e.key === 'Escape' && !e.defaultPrevented && drawerOpen.value && !searchOpen.value) {
    e.preventDefault()
    closeSidebar()
  }
}

onMounted(() => {
  document.addEventListener('keydown', onGlobalKeydown)
})
onUnmounted(() => {
  document.removeEventListener('keydown', onGlobalKeydown)
})

const navLinkClass = 'flex items-center gap-3 px-3 py-2 rounded-lg text-[13px] font-medium text-sidebar-text hover:bg-sidebar-hover hover:text-sidebar-text-active transition-colors focus:outline-none focus-visible:ring-2 focus-visible:ring-primary/60'
const navActiveClass = '!bg-sidebar-active !text-sidebar-text-active'

const MAC_RE = /mac/i
const cmdKey = computed(() => (typeof navigator !== 'undefined' && MAC_RE.test(navigator.platform)) ? '⌘' : 'Ctrl')
const auth = useAuth()
const { data: systemInfo } = useSystemInfo()
const isSingleWorkspace = computed(() => systemInfo.value?.workspacesMode === 'single')

const navItems = [
  { name: 'Dashboard', to: '/', icon: LayoutDashboard },
  { name: 'Sources', to: '/sources', icon: Database },
  { name: 'Destinations', to: '/destinations', icon: Server },
  { name: 'Connections', to: '/connections', icon: ArrowRightLeft },
  { name: 'Topology', to: '/topology', icon: Network },
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
      aria-hidden="true"
      @click="closeSidebar"
    />

    <!-- Sidebar -->
    <aside
      id="app-sidebar"
      ref="sidebarEl"
      aria-label="Sidebar"
      :inert="drawerHidden"
      class="fixed inset-y-0 left-0 z-50 w-[240px] bg-sidebar flex flex-col shrink-0 transform transition-transform duration-200 lg:static lg:translate-x-0"
      :class="sidebarOpen ? 'translate-x-0' : '-translate-x-full'"
    >
      <!-- Logo -->
      <div class="px-5 h-14 flex items-center border-b border-sidebar-border">
        <div class="flex items-center gap-2.5">
          <SyncletLogo />
          <span class="text-[15px] font-semibold text-sidebar-text-active tracking-tight">Synclet</span>
        </div>
      </div>

      <!-- Navigation -->
      <nav aria-label="Main" class="flex-1 px-3 py-3 space-y-0.5">
        <button
          type="button"
          class="w-full text-left"
          :class="navLinkClass"
          :aria-keyshortcuts="cmdKey === '⌘' ? 'Meta+K' : 'Control+K'"
          @click="searchOpen = true"
        >
          <Search class="w-4 h-4 opacity-70" />
          <span class="flex-1">Search</span>
          <kbd class="text-[10px] border border-sidebar-border rounded px-1 py-0.5 opacity-70">{{ cmdKey }}K</kbd>
        </button>
        <RouterLink
          v-for="item in navItems"
          :key="item.name"
          :to="item.to"
          :class="[navLinkClass, isNavActive(route.path, item.to) && navActiveClass]"
          :aria-current="isNavActive(route.path, item.to) ? 'page' : undefined"
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
          :class="[navLinkClass, isNavActive(route.path, '/settings') && navActiveClass]"
          :aria-current="isNavActive(route.path, '/settings') ? 'page' : undefined"
          @click="sidebarOpen = false"
        >
          <Settings class="w-4 h-4 opacity-70" />
          Settings
        </RouterLink>

        <!-- Workspace selector -->
        <SDropdown v-if="!isSingleWorkspace && auth.workspaces.value.length > 0" :items="workspaceItems" align="left">
          <template #trigger="{ attrs }">
            <button v-bind="attrs" type="button" class="w-full flex items-center justify-between px-3 py-2 rounded-lg text-[13px] text-sidebar-text hover:bg-sidebar-hover hover:text-sidebar-text-active transition-colors focus:outline-none focus-visible:ring-2 focus-visible:ring-primary/60">
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
          <SButton
            variant="ghost"
            size="icon"
            class="!text-sidebar-text/50 hover:!text-sidebar-text-active hover:!bg-sidebar-hover focus-visible:ring-offset-sidebar"
            aria-label="Log out"
            title="Log out"
            @click="handleLogout"
          >
            <LogOut class="w-3.5 h-3.5" />
          </SButton>
        </div>
      </div>
    </aside>

    <!-- Main content -->
    <main class="flex-1 overflow-auto" :inert="drawerOpen">
      <div class="sticky top-0 z-30 flex items-center h-14 px-4 bg-page border-b border-border lg:hidden">
        <SButton
          ref="menuButton"
          variant="ghost"
          size="icon"
          aria-label="Open menu"
          aria-controls="app-sidebar"
          :aria-expanded="sidebarOpen"
          @click="openSidebar"
        >
          <Menu class="w-5 h-5 text-heading" />
        </SButton>
        <div class="ml-3 flex items-center gap-2.5">
          <SyncletLogo />
          <span class="text-[15px] font-semibold text-heading tracking-tight">Synclet</span>
        </div>
      </div>
      <div class="px-4 py-4 sm:px-6 sm:py-6 lg:px-8 lg:py-8">
        <RouterView />
      </div>
    </main>

    <GlobalSearchModal :open="searchOpen" @close="searchOpen = false" />
  </div>
</template>

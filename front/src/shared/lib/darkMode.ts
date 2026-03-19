import { ref, watch } from 'vue'

const STORAGE_KEY = 'synclet-theme'
export type Theme = 'light' | 'dark' | 'system'

const theme = ref<Theme>((localStorage.getItem(STORAGE_KEY) as Theme) || 'system')

function applyTheme(t: Theme) {
  const isDark = t === 'dark' || (t === 'system' && window.matchMedia('(prefers-color-scheme: dark)').matches)
  document.documentElement.classList.toggle('dark', isDark)
}

watch(theme, (val) => {
  localStorage.setItem(STORAGE_KEY, val)
  applyTheme(val)
}, { immediate: true })

window.matchMedia('(prefers-color-scheme: dark)').addEventListener('change', () => {
  if (theme.value === 'system')
    applyTheme('system')
})

export function useDarkMode() {
  return {
    theme,
    setTheme: (t: Theme) => {
      theme.value = t
    },
  }
}

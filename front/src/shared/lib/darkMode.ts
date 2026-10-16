import { ref, watch } from 'vue'

const STORAGE_KEY = 'synclet-theme'
const THEME_COLOR = { light: '#f8fafc', dark: '#020617' }
export type Theme = 'light' | 'dark' | 'system'

function isTheme(value: unknown): value is Theme {
  return value === 'light' || value === 'dark' || value === 'system'
}

function readStoredTheme(): Theme {
  try {
    const stored = localStorage.getItem(STORAGE_KEY)
    return isTheme(stored) ? stored : 'system'
  }
  catch {
    return 'system'
  }
}

function storeTheme(t: Theme) {
  try {
    localStorage.setItem(STORAGE_KEY, t)
  }
  catch {}
}

function darkQuery(): MediaQueryList | undefined {
  try {
    return window.matchMedia('(prefers-color-scheme: dark)')
  }
  catch {
    return undefined
  }
}

const query = darkQuery()
const theme = ref<Theme>(readStoredTheme())

function applyTheme(t: Theme) {
  const isDark = t === 'dark' || (t === 'system' && !!query?.matches)
  document.documentElement.classList.toggle('dark', isDark)
  for (const meta of document.querySelectorAll<HTMLMetaElement>('meta[name="theme-color"]')) {
    const metaDark = t === 'system' ? meta.media.includes('dark') : isDark
    meta.content = metaDark ? THEME_COLOR.dark : THEME_COLOR.light
  }
}

watch(theme, (val) => {
  storeTheme(val)
  applyTheme(val)
}, { immediate: true })

query?.addEventListener('change', () => {
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

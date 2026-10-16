import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { nextTick } from 'vue'

function addThemeColorMeta(media: string) {
  const meta = document.createElement('meta')
  meta.name = 'theme-color'
  meta.media = media
  meta.content = ''
  document.head.appendChild(meta)
  return meta
}

describe('darkMode', () => {
  beforeEach(() => {
    vi.resetModules()
    document.head.innerHTML = ''
    document.documentElement.className = ''
    vi.stubGlobal('matchMedia', vi.fn(() => ({ matches: false, addEventListener: vi.fn() })))
  })

  afterEach(() => {
    vi.unstubAllGlobals()
    vi.restoreAllMocks()
  })

  it('falls back to system when storage access throws', async () => {
    vi.spyOn(Storage.prototype, 'getItem').mockImplementation(() => {
      throw new DOMException('blocked', 'SecurityError')
    })
    vi.spyOn(Storage.prototype, 'setItem').mockImplementation(() => {
      throw new DOMException('blocked', 'SecurityError')
    })
    const { useDarkMode } = await import('../darkMode')
    const { theme, setTheme } = useDarkMode()
    expect(theme.value).toBe('system')
    setTheme('dark')
    await nextTick()
    expect(document.documentElement.classList.contains('dark')).toBe(true)
  })

  it('ignores an unrecognised stored value', async () => {
    vi.spyOn(Storage.prototype, 'getItem').mockReturnValue('purple')
    const { useDarkMode } = await import('../darkMode')
    expect(useDarkMode().theme.value).toBe('system')
  })

  it('survives a missing matchMedia', async () => {
    vi.stubGlobal('matchMedia', undefined)
    const { useDarkMode } = await import('../darkMode')
    expect(useDarkMode().theme.value).toBeDefined()
  })

  it('points every theme-color meta at the chosen theme', async () => {
    const light = addThemeColorMeta('(prefers-color-scheme: light)')
    const dark = addThemeColorMeta('(prefers-color-scheme: dark)')
    vi.spyOn(Storage.prototype, 'getItem').mockReturnValue('system')
    const { useDarkMode } = await import('../darkMode')
    expect([light.content, dark.content]).toEqual(['#f8fafc', '#020617'])

    useDarkMode().setTheme('dark')
    await nextTick()
    expect([light.content, dark.content]).toEqual(['#020617', '#020617'])

    useDarkMode().setTheme('light')
    await nextTick()
    expect([light.content, dark.content]).toEqual(['#f8fafc', '#f8fafc'])
  })
})

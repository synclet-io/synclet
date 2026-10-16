import { Chart as ChartJS } from 'chart.js'
import { describe, expect, it } from 'vitest'
import { readPalette, useChartTheme, withAlpha } from '../chartSetup'

const root = document.documentElement

function setVars(vars: Record<string, string>) {
  for (const [name, value] of Object.entries(vars))
    root.style.setProperty(name, value)
}

describe('withAlpha', () => {
  it('converts long and short hex', () => {
    expect(withAlpha('#22c55e', 0.1)).toBe('rgba(34, 197, 94, 0.1)')
    expect(withAlpha('#fff', 0.5)).toBe('rgba(255, 255, 255, 0.5)')
  })

  it('replaces the alpha of rgb and rgba colours', () => {
    expect(withAlpha('rgb(59 130 246 / 0.1)', 0.2)).toBe('rgba(59, 130, 246, 0.2)')
    expect(withAlpha('rgba(1, 2, 3, 0.9)', 0.4)).toBe('rgba(1, 2, 3, 0.4)')
  })

  it('passes unknown formats through', () => {
    expect(withAlpha('oklch(0.6 0.2 250)', 0.3)).toBe('oklch(0.6 0.2 250)')
  })
})

describe('readPalette', () => {
  it('reads trimmed values of the theme variables', () => {
    setVars({ '--color-primary': ' #2563eb', '--color-status-success': '#16a34a', '--color-text-secondary': '#64748b' })
    const p = readPalette()
    expect(p.primary).toBe('#2563eb')
    expect(p.success).toBe('#16a34a')
    expect(p.text).toBe('#64748b')
  })
})

describe('useChartTheme', () => {
  it('applies chart defaults and refreshes when the html class changes', async () => {
    setVars({ '--color-text-secondary': '#64748b', '--color-border-default': '#e2e8f0', '--color-primary': '#2563eb' })
    const { palette, version } = useChartTheme()
    const before = version.value
    expect(palette.value.primary).toBe('#2563eb')
    expect(ChartJS.defaults.color).toBe('#64748b')
    expect(ChartJS.defaults.borderColor).toBe('#e2e8f0')

    setVars({ '--color-text-secondary': '#94a3b8', '--color-border-default': '#334155', '--color-primary': '#3b82f6' })
    root.classList.add('dark')
    await new Promise(resolve => setTimeout(resolve, 0))

    expect(palette.value.primary).toBe('#3b82f6')
    expect(ChartJS.defaults.color).toBe('#94a3b8')
    expect(ChartJS.defaults.borderColor).toBe('#334155')
    expect(version.value).toBe(before + 1)

    root.classList.add('overflow-hidden')
    await new Promise(resolve => setTimeout(resolve, 0))
    expect(version.value).toBe(before + 1)
  })
})

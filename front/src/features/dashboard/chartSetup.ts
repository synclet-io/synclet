import {
  ArcElement,
  BarElement,
  CategoryScale,
  Chart as ChartJS,
  Filler,
  Legend,
  LinearScale,
  LineElement,
  PointElement,
  Title,
  Tooltip,
} from 'chart.js'
import { computed, ref, shallowRef } from 'vue'

ChartJS.register(
  CategoryScale,
  LinearScale,
  BarElement,
  LineElement,
  PointElement,
  ArcElement,
  Title,
  Tooltip,
  Legend,
  Filler,
)

export interface ChartPalette {
  heading: string
  text: string
  grid: string
  surface: string
  primary: string
  success: string
  danger: string
  warning: string
  accent: string
  neutral: string
  muted: string
}

const PALETTE_VARS: Record<keyof ChartPalette, string> = {
  heading: '--color-text-heading',
  text: '--color-text-secondary',
  grid: '--color-border-default',
  surface: '--color-bg-surface',
  primary: '--color-primary',
  success: '--color-status-success',
  danger: '--color-status-danger',
  warning: '--color-status-warning',
  accent: '--color-purple-400',
  neutral: '--color-text-secondary',
  muted: '--color-text-muted',
}

export function readPalette(el: Element = document.documentElement): ChartPalette {
  const style = getComputedStyle(el)
  const palette = {} as ChartPalette
  for (const [key, name] of Object.entries(PALETTE_VARS))
    palette[key as keyof ChartPalette] = style.getPropertyValue(name).trim()
  return palette
}

const HEX_COLOR = /^#([0-9a-f]{3}|[0-9a-f]{6})$/i
const RGB_COLOR = /^rgba?\(\s*([\d.]+)[\s,]+([\d.]+)[\s,]+([\d.]+)/i

export function withAlpha(color: string, alpha: number): string {
  const hex = HEX_COLOR.exec(color)
  if (hex) {
    const digits = hex[1].length === 3 ? Array.from(hex[1], c => c + c).join('') : hex[1]
    const [r, g, b] = [0, 2, 4].map(i => Number.parseInt(digits.slice(i, i + 2), 16))
    return `rgba(${r}, ${g}, ${b}, ${alpha})`
  }
  const rgb = RGB_COLOR.exec(color)
  if (rgb)
    return `rgba(${rgb[1]}, ${rgb[2]}, ${rgb[3]}, ${alpha})`
  return color
}

function applyDefaults(p: ChartPalette) {
  ChartJS.defaults.color = p.text
  ChartJS.defaults.borderColor = p.grid
  ChartJS.defaults.plugins.tooltip.backgroundColor = withAlpha(p.surface, 0.95)
  ChartJS.defaults.plugins.tooltip.titleColor = p.heading
  ChartJS.defaults.plugins.tooltip.bodyColor = p.heading
  ChartJS.defaults.plugins.tooltip.borderColor = p.grid
  ChartJS.defaults.plugins.tooltip.borderWidth = 1
}

const current = shallowRef<ChartPalette | null>(null)
const version = ref(0)

function refresh() {
  const next = readPalette()
  if (current.value && JSON.stringify(next) === JSON.stringify(current.value))
    return
  current.value = next
  applyDefaults(next)
  version.value++
}

function ensureObserver() {
  if (current.value)
    return
  refresh()
  new MutationObserver(refresh).observe(document.documentElement, {
    attributes: true,
    attributeFilter: ['class'],
  })
}

export function useChartTheme() {
  ensureObserver()
  return {
    palette: computed(() => current.value!),
    version: computed(() => version.value),
  }
}

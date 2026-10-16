import type { Page } from '@playwright/test'
import AxeBuilder from '@axe-core/playwright'
import { expect, test } from '@playwright/test'
import { FIXTURE_ID, mockApi, WORKSPACE_ID } from './mockApi'

interface RouteEntry {
  path: string
  public: boolean
}

const THEMES = ['light', 'dark'] as const
const FAILING_IMPACTS = new Set(['critical', 'serious'])

function routeUrl(path: string): string {
  return path
    .replace(/:pathMatch\([^)]*\)\*/, 'no-such-page')
    .replace(/:token/, 'invite-token')
    .replace(/:id/g, FIXTURE_ID)
}

async function loadRoutes(page: Page): Promise<RouteEntry[]> {
  await page.goto('/login')
  const routes = await page.evaluate(async (modulePath) => {
    const { router } = await import(/* @vite-ignore */ modulePath)
    return (router.getRoutes() as Array<{ path: string, meta: Record<string, unknown>, redirect?: unknown }>)
      .filter(r => !r.redirect)
      .map(r => ({ path: r.path, public: r.meta.public === true }))
  }, '/src/app/router.ts')
  const unique = new Map(routes.map(r => [r.path, r]))
  return [...unique.values()]
}

for (const theme of THEMES) {
  test(`every route in ${theme} theme has no critical or serious axe violations`, async ({ browser, baseURL }) => {
    test.setTimeout(5 * 60_000)

    const probe = await browser.newPage()
    await mockApi(probe)
    const routes = await loadRoutes(probe)
    await probe.close()
    expect(routes.length).toBeGreaterThan(20)

    const failures: string[] = []

    for (const route of routes) {
      const context = await browser.newContext({ colorScheme: theme })
      if (!route.public) {
        const expires = Math.floor(Date.now() / 1000) + 3600
        await context.addCookies([{
          name: 'synclet_auth',
          value: `access_expires=${expires}&refresh_expires=${expires}`,
          url: baseURL!,
        }])
      }
      await context.addInitScript(([t, ws]) => {
        localStorage.setItem('synclet-theme', t)
        localStorage.setItem('synclet_workspace_id', ws)
      }, [theme, WORKSPACE_ID])

      const page = await context.newPage()
      await mockApi(page)
      const url = routeUrl(route.path)
      await page.goto(url)
      await page.waitForLoadState('networkidle')
      await expect(page.locator('#app')).not.toBeEmpty()
      await expect(page.locator('html')).toHaveClass(theme === 'dark' ? /\bdark\b/ : /^(?!.*\bdark\b)/)

      const { violations } = await new AxeBuilder({ page }).analyze()
      for (const v of violations) {
        if (!FAILING_IMPACTS.has(v.impact ?? ''))
          continue
        const targets = v.nodes.map(n => n.target.join(' ')).join(', ')
        failures.push(`${url} (${new URL(page.url()).pathname}): ${v.id} (${v.impact}) ${targets}`)
      }
      await context.close()
    }

    expect(failures, failures.join('\n')).toEqual([])
  })
}

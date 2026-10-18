// Captures the README and docs images from a Synclet filled by demo-seed.
//
//   node scripts/demo-seed/capture.mjs screenshots <out-dir> <connection-id> <job-id>
//   node scripts/demo-seed/capture.mjs gif <out-dir>
//
// "screenshots" writes <page>-light.png and <page>-dark.png at 1440x900 for the
// dashboard, a connection, the topology, the audit log and a job's logs.
// "gif" records the "create a connection, run its first sync" flow in the light
// theme as a WebM in <out-dir>; README.md has the ffmpeg step that makes the GIF.
// The gif mode creates a connection named "Product catalog → Warehouse".
//
// Playwright comes from front/node_modules (cd front && bun install).
// SYNCLET_URL, DEMO_EMAIL and DEMO_PASSWORD override the demo-seed defaults.
import { createRequire } from 'node:module'
import { fileURLToPath } from 'node:url'

const require = createRequire(fileURLToPath(new URL('../../front/package.json', import.meta.url)))
const { chromium } = require('playwright')

const base = process.env.SYNCLET_URL || 'http://localhost:8080'
const email = process.env.DEMO_EMAIL || 'demo@example.com'
const password = process.env.DEMO_PASSWORD || 'demo-password'
const [mode, out, connectionId, jobId] = process.argv.slice(2)

if (!out || (mode === 'screenshots' && !jobId) || !['screenshots', 'gif'].includes(mode)) {
  console.error('usage: capture.mjs screenshots <out-dir> <connection-id> <job-id> | capture.mjs gif <out-dir>')
  process.exit(2)
}

async function login(page) {
  await page.goto(`${base}/login`)
  await page.getByLabel(/email/i).fill(email)
  await page.getByLabel(/password/i).fill(password)
  await page.getByRole('button', { name: /sign in|log in/i }).click()
  await page.waitForURL(u => !u.pathname.startsWith('/login'))
}

async function screenshots(browser) {
  const pages = [
    ['dashboard', '/'],
    ['connection', `/connections/${connectionId}`],
    ['topology', '/topology'],
    ['audit-log', '/settings/audit'],
    ['job-logs', `/jobs/${jobId}`],
  ]

  for (const theme of ['light', 'dark']) {
    const ctx = await browser.newContext({ viewport: { width: 1440, height: 900 }, colorScheme: theme, locale: 'en-US', timezoneId: 'UTC' })
    await ctx.addInitScript((t) => {
      try { localStorage.setItem('synclet-theme', t) }
      catch {}
    }, theme)
    const page = await ctx.newPage()
    await login(page)

    for (const [name, path] of pages) {
      await page.goto(base + path)
      await page.waitForLoadState('networkidle')
      await page.waitForTimeout(2000)

      const dismiss = page.getByRole('button', { name: 'Dismiss' })
      if (await dismiss.count()) {
        await dismiss.first().click()
        await page.waitForTimeout(500)
      }

      if (name === 'audit-log') {
        await page.getByRole('combobox', { name: 'Resource type' }).selectOption({ label: 'Connections' })
        await page.waitForTimeout(800)
        await page.getByRole('combobox', { name: 'Action' }).selectOption({ label: 'Update' })
        await page.waitForTimeout(1500)
        await page.locator('main button[aria-expanded]').nth(1).click()
        await page.waitForTimeout(1000)
      }

      if (name === 'job-logs') {
        await page.mouse.wheel(0, 330)
        await page.waitForTimeout(800)
      }

      await page.mouse.move(1430, 890)
      await page.screenshot({ path: `${out}/${name}-${theme}.png` })
      console.log(`${name}-${theme}.png`)
    }

    await ctx.close()
  }
}

async function gif(browser) {
  const W = 1280
  const H = 800

  const pre = await browser.newContext({ viewport: { width: W, height: H } })
  await login(await pre.newPage())
  const storageState = await pre.storageState()
  await pre.close()

  const ctx = await browser.newContext({
    viewport: { width: W, height: H },
    colorScheme: 'light',
    locale: 'en-US',
    timezoneId: 'UTC',
    storageState,
    recordVideo: { dir: out, size: { width: W, height: H } },
  })
  // A visible pointer, since recorded video has no cursor.
  await ctx.addInitScript(() => {
    try { localStorage.setItem('synclet-theme', 'light') }
    catch {}
    window.addEventListener('DOMContentLoaded', () => {
      const d = document.createElement('div')
      Object.assign(d.style, {
        position: 'fixed', zIndex: 2147483647, pointerEvents: 'none', width: '18px', height: '18px', borderRadius: '50%',
        background: 'rgba(37,99,235,.35)', border: '2px solid rgba(37,99,235,.9)', transform: 'translate(-50%,-50%)',
        left: '-40px', top: '-40px', transition: 'width .1s,height .1s',
      })
      document.body.appendChild(d)
      document.addEventListener('mousemove', (e) => { d.style.left = `${e.clientX}px`; d.style.top = `${e.clientY}px` }, true)
      document.addEventListener('mousedown', () => { d.style.width = d.style.height = '12px' }, true)
      document.addEventListener('mouseup', () => { d.style.width = d.style.height = '18px' }, true)
    })
  })

  const page = await ctx.newPage()
  const t0 = Date.now()
  const marks = []
  const mark = name => marks.push([name, (Date.now() - t0) / 1000])

  async function click(loc, pause = 500) {
    await loc.scrollIntoViewIfNeeded()
    const b = await loc.boundingBox()
    await page.mouse.move(b.x + Math.min(b.width / 2, 60), b.y + b.height / 2, { steps: 18 })
    await page.waitForTimeout(250)
    await page.mouse.down()
    await page.mouse.up()
    await page.waitForTimeout(pause)
  }

  async function type(loc, text) {
    await click(loc, 150)
    await page.keyboard.type(text, { delay: 45 })
    await page.waitForTimeout(300)
  }

  await page.goto(`${base}/connections`)
  await page.waitForTimeout(1800)
  mark('start')
  await click(page.getByRole('link', { name: /new connection/i }).first(), 900)
  await click(page.getByRole('button', { name: /^Storefront \(Faker\)/ }), 700)
  await click(page.getByRole('button', { name: /^Analytics warehouse/ }), 700)
  await type(page.getByLabel('Connection Name'), 'Product catalog → Warehouse')
  await type(page.getByLabel('Stream Prefix'), 'catalog_')
  await click(page.getByRole('button', { name: 'Next' }), 1000)
  await click(page.getByRole('checkbox', { name: 'Enable stream products' }), 400)
  await click(page.getByRole('checkbox', { name: 'Enable stream purchases' }), 600)
  await click(page.getByRole('button', { name: 'Next' }), 1500)
  await click(page.getByRole('button', { name: 'Create Connection' }), 200)
  await page.waitForURL(/\/connections\/[0-9a-f-]{36}/, { timeout: 20000 })
  await page.waitForTimeout(1500)
  await click(page.getByRole('button', { name: 'Sync Now' }), 300)
  await page.getByRole('button', { name: 'Sync Now' }).waitFor({ timeout: 180000 })
  mark('sync done')
  await page.waitForTimeout(2500)
  await click(page.locator('main table tbody tr a').first(), 2500)
  await page.waitForTimeout(2500)
  mark('end')

  await ctx.close()
  console.log(await page.video().path())
  console.log('marks (seconds after the page opened):', JSON.stringify(marks))
}

const browser = await chromium.launch()
try {
  await (mode === 'gif' ? gif(browser) : screenshots(browser))
}
finally {
  await browser.close()
}

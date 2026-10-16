// Synchronously applies the dark theme class before first paint so the page
// does not flash light → dark when the user has dark mode persisted. Loaded
// with a regular <script> in index.html (not module) so it runs before any
// of the bundled app code. Lives in /public so it stays out of the Vite
// bundle and remains a "self"-origin script under strict CSP.
(function () {
  let t = 'system'
  try {
    const stored = localStorage.getItem('synclet-theme')
    if (stored === 'light' || stored === 'dark')
      t = stored
  }
  catch {}
  let systemDark = false
  try {
    systemDark = window.matchMedia('(prefers-color-scheme: dark)').matches
  }
  catch {}
  const d = t === 'dark' || (t === 'system' && systemDark)
  if (d)
    document.documentElement.classList.add('dark')
  if (t !== 'system') {
    const color = d ? '#020617' : '#f8fafc'
    document.querySelectorAll('meta[name="theme-color"]').forEach((m) => {
      m.setAttribute('content', color)
    })
  }
})()

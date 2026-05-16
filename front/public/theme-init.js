// Synchronously applies the dark theme class before first paint so the page
// does not flash light → dark when the user has dark mode persisted. Loaded
// with a regular <script> in index.html (not module) so it runs before any
// of the bundled app code. Lives in /public so it stays out of the Vite
// bundle and remains a "self"-origin script under strict CSP.
(function () {
  const t = localStorage.getItem('synclet-theme') || 'system'
  const d = t === 'dark' || (t === 'system' && window.matchMedia('(prefers-color-scheme: dark)').matches)
  if (d) {
    document.documentElement.classList.add('dark')
  }
})()

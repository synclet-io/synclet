const PENDING_INVITE_KEY = 'pendingInviteToken'
const OIDC_REDIRECT_KEY = 'synclet_oidc_redirect'
const INVITE_TOKEN_RE = /^[0-9a-f]{64}$/i
const AUTH_PATHS = ['/login', '/register', '/auth/oidc/callback']

export function isInviteToken(value: unknown): value is string {
  return typeof value === 'string' && INVITE_TOKEN_RE.test(value)
}

export function storePendingInvite(token: string) {
  if (isInviteToken(token))
    localStorage.setItem(PENDING_INVITE_KEY, token)
}

export function takePendingInvite(): string | null {
  const token = localStorage.getItem(PENDING_INVITE_KEY)
  localStorage.removeItem(PENDING_INVITE_KEY)
  return isInviteToken(token) ? token : null
}

/** Returns `raw` when it is a same-origin app path that is not an auth page, otherwise null. */
export function safeRedirect(raw: unknown): string | null {
  if (typeof raw !== 'string' || !raw.startsWith('/') || raw.startsWith('//') || raw.startsWith('/\\'))
    return null
  let url: URL
  try {
    url = new URL(raw, window.location.origin)
  }
  catch {
    return null
  }
  if (url.origin !== window.location.origin || AUTH_PATHS.includes(url.pathname))
    return null
  return `${url.pathname}${url.search}${url.hash}`
}

export function rememberOIDCRedirect(raw: unknown) {
  const target = safeRedirect(raw)
  if (target)
    sessionStorage.setItem(OIDC_REDIRECT_KEY, target)
  else
    sessionStorage.removeItem(OIDC_REDIRECT_KEY)
}

export function takeOIDCRedirect(): string | null {
  const target = sessionStorage.getItem(OIDC_REDIRECT_KEY)
  sessionStorage.removeItem(OIDC_REDIRECT_KEY)
  return safeRedirect(target)
}

/** Where to send the user after a successful sign-in: a pending invite wins over `redirect`. */
export function postAuthPath(redirect: unknown): string {
  const invite = takePendingInvite()
  if (invite)
    return `/invite/${invite}`
  return safeRedirect(redirect) ?? '/'
}

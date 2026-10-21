/** Sends the browser to a URL outside the app. */
export function redirectTo(url: string) {
  window.location.assign(url)
}

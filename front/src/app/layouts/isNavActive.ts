export function isNavActive(currentPath: string, to: string): boolean {
  if (to === '/')
    return currentPath === '/'
  return currentPath === to || currentPath.startsWith(`${to}/`)
}

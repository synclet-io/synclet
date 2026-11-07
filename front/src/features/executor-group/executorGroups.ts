export const DEFAULT_EXECUTOR_GROUP = 'default'

export interface ExecutorGroupOption {
  name: string
  liveExecutors: number
  inUse: boolean
}

interface Grouped {
  executorGroup: string
}

export function groupOf(item: Grouped): string {
  return item.executorGroup || DEFAULT_EXECUTOR_GROUP
}

/** Groups with a live executor, groups used by a source or destination, and `default`; `default` first, the rest by name. */
export function executorGroupOptions(executors: Grouped[], sources: Grouped[], destinations: Grouped[]): ExecutorGroupOption[] {
  const byName = new Map<string, ExecutorGroupOption>()
  const option = (name: string) => {
    let o = byName.get(name)
    if (!o) {
      o = { name, liveExecutors: 0, inUse: false }
      byName.set(name, o)
    }
    return o
  }
  option(DEFAULT_EXECUTOR_GROUP)
  for (const e of executors)
    option(groupOf(e)).liveExecutors++
  for (const item of [...sources, ...destinations])
    option(groupOf(item)).inUse = true
  return [...byName.values()].sort((a, b) => {
    if (a.name === DEFAULT_EXECUTOR_GROUP)
      return -1
    if (b.name === DEFAULT_EXECUTOR_GROUP)
      return 1
    return a.name.localeCompare(b.name)
  })
}

export function isOnlyDefault(groups: ExecutorGroupOption[]): boolean {
  return groups.length === 1 && groups[0].name === DEFAULT_EXECUTOR_GROUP
}

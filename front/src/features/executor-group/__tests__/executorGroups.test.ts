import { describe, expect, it } from 'vitest'
import { executorGroupOptions, isOnlyDefault } from '../executorGroups'

const live = (...groups: string[]) => groups.map(executorGroup => ({ executorGroup }))
const used = live

describe('executorGroupOptions', () => {
  it('lists default alone when nothing exists', () => {
    const groups = executorGroupOptions([], [], [])
    expect(groups).toEqual([{ name: 'default', liveExecutors: 0, inUse: false }])
    expect(isOnlyDefault(groups)).toBe(true)
  })

  it('counts default executors and uses', () => {
    const groups = executorGroupOptions(live('default'), used('default'), used('default'))
    expect(groups).toEqual([{ name: 'default', liveExecutors: 1, inUse: true }])
    expect(isOnlyDefault(groups)).toBe(true)
  })

  it('lists groups with live executors', () => {
    const groups = executorGroupOptions(live('default', 'eu', 'eu'), used('default'), [])
    expect(groups).toEqual([
      { name: 'default', liveExecutors: 1, inUse: true },
      { name: 'eu', liveExecutors: 2, inUse: false },
    ])
    expect(isOnlyDefault(groups)).toBe(false)
  })

  it('lists groups still used without live executors and counts an empty group as default', () => {
    const groups = executorGroupOptions(live('default'), used('us'), used(''))
    expect(groups).toEqual([
      { name: 'default', liveExecutors: 1, inUse: true },
      { name: 'us', liveExecutors: 0, inUse: true },
    ])
  })

  it('puts default first and sorts the rest by name', () => {
    const groups = executorGroupOptions(live('zeta', 'Alpha'), used('beta'), [])
    expect(groups.map(g => g.name)).toEqual(['default', 'Alpha', 'beta', 'zeta'])
  })
})

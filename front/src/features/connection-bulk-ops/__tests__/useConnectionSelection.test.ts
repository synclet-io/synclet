import { describe, expect, it } from 'vitest'
import { useConnectionSelection } from '../useConnectionSelection'

describe('useConnectionSelection', () => {
  it('starts empty', () => {
    const s = useConnectionSelection()
    expect(s.count.value).toBe(0)
    expect(s.isEmpty.value).toBe(true)
    expect(s.ids()).toEqual([])
  })

  it('toggles individual ids', () => {
    const s = useConnectionSelection()
    s.toggle('a')
    expect(s.isSelected('a')).toBe(true)
    expect(s.count.value).toBe(1)
    s.toggle('a')
    expect(s.isSelected('a')).toBe(false)
    expect(s.isEmpty.value).toBe(true)
  })

  it('selectMany adds multiple ids without duplicates', () => {
    const s = useConnectionSelection()
    s.selectMany(['a', 'b', 'c'])
    s.selectMany(['b', 'd'])
    expect(s.ids().sort()).toEqual(['a', 'b', 'c', 'd'])
  })

  it('clear removes everything', () => {
    const s = useConnectionSelection()
    s.selectMany(['a', 'b'])
    s.clear()
    expect(s.isEmpty.value).toBe(true)
  })

  it('allSelected only true when every pool id is selected', () => {
    const s = useConnectionSelection()
    expect(s.allSelected(['a', 'b'])).toBe(false)
    s.selectMany(['a'])
    expect(s.allSelected(['a', 'b'])).toBe(false)
    s.selectMany(['b'])
    expect(s.allSelected(['a', 'b'])).toBe(true)
  })

  it('allSelected returns false on empty pool (nothing to select)', () => {
    const s = useConnectionSelection()
    expect(s.allSelected([])).toBe(false)
  })

  it('toggleAll selects all then deselects all on second call', () => {
    const s = useConnectionSelection()
    const pool = ['a', 'b', 'c']
    s.toggleAll(pool)
    expect(s.allSelected(pool)).toBe(true)
    s.toggleAll(pool)
    expect(s.isEmpty.value).toBe(true)
  })

  it('toggleAll only affects the given pool (preserves other selections)', () => {
    const s = useConnectionSelection()
    s.selectMany(['x'])
    s.toggleAll(['a', 'b'])
    expect(s.ids().sort()).toEqual(['a', 'b', 'x'])
    s.toggleAll(['a', 'b'])
    expect(s.ids()).toEqual(['x'])
  })

  it('setSelected explicitly sets membership', () => {
    const s = useConnectionSelection()
    s.setSelected('a', true)
    expect(s.isSelected('a')).toBe(true)
    s.setSelected('a', true)
    expect(s.count.value).toBe(1)
    s.setSelected('a', false)
    expect(s.isSelected('a')).toBe(false)
  })
})

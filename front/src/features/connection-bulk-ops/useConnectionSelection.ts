import { computed, ref } from 'vue'

export function useConnectionSelection() {
  const selected = ref<Set<string>>(new Set())

  const count = computed(() => selected.value.size)
  const isEmpty = computed(() => selected.value.size === 0)

  function isSelected(id: string): boolean {
    return selected.value.has(id)
  }

  function toggle(id: string) {
    const next = new Set(selected.value)
    if (next.has(id))
      next.delete(id)
    else
      next.add(id)
    selected.value = next
  }

  function setSelected(id: string, value: boolean) {
    const next = new Set(selected.value)
    if (value)
      next.add(id)
    else
      next.delete(id)
    selected.value = next
  }

  function selectMany(ids: string[]) {
    const next = new Set(selected.value)
    for (const id of ids) next.add(id)
    selected.value = next
  }

  function deselectMany(ids: string[]) {
    const next = new Set(selected.value)
    for (const id of ids) next.delete(id)
    selected.value = next
  }

  function clear() {
    selected.value = new Set()
  }

  function ids(): string[] {
    return Array.from(selected.value)
  }

  function allSelected(pool: string[]): boolean {
    if (pool.length === 0)
      return false
    return pool.every(id => selected.value.has(id))
  }

  function toggleAll(pool: string[]) {
    if (allSelected(pool))
      deselectMany(pool)
    else
      selectMany(pool)
  }

  return {
    selected,
    count,
    isEmpty,
    isSelected,
    toggle,
    setSelected,
    selectMany,
    deselectMany,
    clear,
    ids,
    allSelected,
    toggleAll,
  }
}

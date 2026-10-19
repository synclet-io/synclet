import type { CSSProperties, Ref } from 'vue'
import { onBeforeUnmount, ref } from 'vue'

/** A card's vertical box, relative to the top of the step list. */
export interface Box { top: number, height: number }

/** The index the dragged card lands at: the number of other cards whose centre is above the dragged card's centre. */
export function dropSlot(boxes: Box[], from: number, draggedCenter: number): number {
  let slot = 0
  boxes.forEach((b, i) => {
    if (i !== from && b.top + b.height / 2 < draggedCenter)
      slot++
  })
  return Math.min(Math.max(slot, 0), Math.max(boxes.length - 1, 0))
}

/** How far card `i` moves aside while the card at `from` is shown in `slot`. */
export function shiftFor(i: number, from: number, slot: number, draggedHeight: number, gap: number): number {
  if (from < i && i <= slot)
    return -(draggedHeight + gap)
  if (slot <= i && i < from)
    return draggedHeight + gap
  return 0
}

/** Top of the placeholder for the card at `from` landing in `slot`, with the other cards shifted aside. */
export function placeholderTop(boxes: Box[], from: number, slot: number): number {
  if (slot <= from)
    return boxes[slot].top
  return boxes[slot].top + boxes[slot].height - boxes[from].height
}

/** How far above or below the list the pointer may go and still drop into the shown slot. */
const OUTSIDE_MARGIN = 48

interface Options {
  list: Ref<HTMLElement | null>
  onDrop: (from: number, to: number) => void
}

/**
 * Pointer-driven reordering of the cards in `list`, each wrapped in a
 * `[data-testid="step-slot"]` element: the dragged card follows the pointer,
 * the others slide aside and `placeholder` marks the slot it will land in.
 */
export function useStepDrag({ list, onDrop }: Options) {
  const dragging = ref<number | null>(null)
  const slot = ref(0)
  const dy = ref(0)
  const placeholder = ref<{ top: number, height: number } | null>(null)

  let boxes: Box[] = []
  let gap = 0
  let startY = 0
  let startListTop = 0

  function slotElements(): HTMLElement[] {
    return list.value ? Array.from(list.value.querySelectorAll<HTMLElement>(':scope > [data-testid="step-slot"]')) : []
  }

  // Only the vertical extent counts, so drifting sideways out of a narrow column keeps the drop.
  function isOutside(clientY: number): boolean {
    const rect = list.value?.getBoundingClientRect()
    return rect === undefined || clientY < rect.top - OUTSIDE_MARGIN || clientY > rect.bottom + OUTSIDE_MARGIN
  }

  function update(outside: boolean) {
    const from = dragging.value
    if (from === null)
      return
    slot.value = dropSlot(boxes, from, boxes[from].top + boxes[from].height / 2 + dy.value)
    placeholder.value = outside ? null : { top: placeholderTop(boxes, from, slot.value), height: boxes[from].height }
  }

  function onMove(e: PointerEvent) {
    if (dragging.value === null || !list.value)
      return
    const listTop = list.value.getBoundingClientRect().top
    dy.value = e.clientY - startY + (startListTop - listTop)
    update(isOutside(e.clientY))
  }

  function end() {
    window.removeEventListener('pointermove', onMove)
    window.removeEventListener('pointerup', onUp)
    window.removeEventListener('pointercancel', end)
    window.removeEventListener('keydown', onKey)
    dragging.value = null
    placeholder.value = null
    dy.value = 0
  }

  function onUp(e: PointerEvent) {
    const from = dragging.value
    const to = slot.value
    const inside = !isOutside(e.clientY)
    end()
    if (from !== null && inside && to !== from)
      onDrop(from, to)
  }

  function onKey(e: KeyboardEvent) {
    if (e.key === 'Escape')
      end()
  }

  function start(index: number, e: PointerEvent) {
    if (!list.value)
      return
    end()
    const listRect = list.value.getBoundingClientRect()
    boxes = slotElements().map((el) => {
      const r = el.getBoundingClientRect()
      return { top: r.top - listRect.top, height: r.height }
    })
    if (index < 0 || index >= boxes.length)
      return
    gap = boxes.length > 1 ? Math.max(boxes[1].top - boxes[0].top - boxes[0].height, 0) : 0
    startY = e.clientY
    startListTop = listRect.top
    const target = e.target as Element | null
    if (target && 'setPointerCapture' in target && typeof target.setPointerCapture === 'function') {
      try {
        target.setPointerCapture(e.pointerId)
      }
      catch {}
    }
    dragging.value = index
    dy.value = 0
    update(false)
    window.addEventListener('pointermove', onMove)
    window.addEventListener('pointerup', onUp)
    window.addEventListener('pointercancel', end)
    window.addEventListener('keydown', onKey)
  }

  function styleFor(i: number): CSSProperties {
    const from = dragging.value
    if (from === null)
      return {}
    if (i === from)
      return { transform: `translateY(${dy.value}px)`, position: 'relative', zIndex: 10 }
    return { transform: `translateY(${shiftFor(i, from, slot.value, boxes[from].height, gap)}px)`, transition: 'transform 150ms ease' }
  }

  onBeforeUnmount(end)

  return { dragging, slot, dy, placeholder, start, styleFor }
}

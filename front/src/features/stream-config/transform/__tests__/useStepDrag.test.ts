import type { Box } from '../useStepDrag'
import { describe, expect, it } from 'vitest'
import { dropSlot, placeholderTop, shiftFor } from '../useStepDrag'

const H = 100
const GAP = 8
const equal: Box[] = [0, 1, 2].map(i => ({ top: i * (H + GAP), height: H }))

function center(boxes: Box[], from: number, dy: number) {
  return boxes[from].top + boxes[from].height / 2 + dy
}

describe('step drag slot math', () => {
  it('drops a card dragged down past the centre of the last card into the last slot', () => {
    const slot = dropSlot(equal, 0, center(equal, 0, 2 * (H + GAP) + 10))
    expect(slot).toBe(2)
    expect([0, 1, 2].map(i => shiftFor(i, 0, slot, H, GAP))).toEqual([0, -(H + GAP), -(H + GAP)])
    expect(placeholderTop(equal, 0, slot)).toBe(equal[2].top)
  })

  it('drops a card dragged up above the centre of the first card into the first slot', () => {
    const slot = dropSlot(equal, 2, center(equal, 2, -2 * (H + GAP) - 10))
    expect(slot).toBe(0)
    expect([0, 1, 2].map(i => shiftFor(i, 2, slot, H, GAP))).toEqual([H + GAP, H + GAP, 0])
    expect(placeholderTop(equal, 2, slot)).toBe(0)
  })

  it('keeps the slot and shifts nothing for a small move within the card', () => {
    const slot = dropSlot(equal, 1, center(equal, 1, 30))
    expect(slot).toBe(1)
    expect([0, 1, 2].map(i => shiftFor(i, 1, slot, H, GAP))).toEqual([0, 0, 0])
    expect(placeholderTop(equal, 1, slot)).toBe(equal[1].top)
  })

  it('clamps a pointer far below or above the list to the last or first slot', () => {
    expect(dropSlot(equal, 0, 10_000)).toBe(2)
    expect(dropSlot(equal, 2, -10_000)).toBe(0)
  })

  it('aligns the placeholder bottom with the target card bottom for unequal heights', () => {
    const boxes: Box[] = [{ top: 0, height: 60 }, { top: 68, height: 200 }, { top: 276, height: 100 }]
    const slot = dropSlot(boxes, 0, center(boxes, 0, 200))
    expect(slot).toBe(1)
    const top = placeholderTop(boxes, 0, slot)
    expect(top + boxes[0].height).toBe(boxes[1].top + boxes[1].height)
    expect(shiftFor(1, 0, slot, 60, GAP)).toBe(-(60 + GAP))
  })
})

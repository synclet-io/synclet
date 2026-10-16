import axe from 'axe-core'
import { expect } from 'vitest'

export async function expectNoAxeViolations(root: Element = document.body) {
  const { violations } = await axe.run(root, {
    rules: {
      'color-contrast': { enabled: false },
      'region': { enabled: false },
    },
  })
  const summary = violations.map(v => `${v.id} (${v.impact}): ${v.nodes.map(n => n.target.join(' ')).join(', ')}`)
  expect(summary).toEqual([])
}

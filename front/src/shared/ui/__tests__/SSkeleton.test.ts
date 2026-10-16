import { mount } from '@vue/test-utils'
import { describe, expect, it } from 'vitest'
import SSkeleton from '../SSkeleton.vue'

describe('sSkeleton', () => {
  it('puts caller classes on the block itself so utility sizing applies', () => {
    const w = mount(SSkeleton, { attrs: { class: 'h-64' } })
    expect(w.classes()).toContain('h-64')
    expect(w.classes()).toContain('animate-pulse')
    expect(w.attributes('style')).toBeUndefined()
  })

  it('applies explicit width and height props inline', () => {
    const w = mount(SSkeleton, { props: { variant: 'rect', width: '300px', height: '40px' } })
    expect(w.attributes('style')).toContain('width: 300px')
    expect(w.attributes('style')).toContain('height: 40px')
  })

  it('uses the width as height for circles', () => {
    const w = mount(SSkeleton, { props: { variant: 'circle', width: '24px' } })
    expect(w.attributes('style')).toContain('height: 24px')
    expect(w.classes()).toContain('rounded-full')
  })

  it('renders count blocks in a wrapper', () => {
    const w = mount(SSkeleton, { props: { count: 3 } })
    expect(w.findAll('.animate-pulse')).toHaveLength(3)
  })
})

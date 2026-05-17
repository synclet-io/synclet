import { describe, expect, it } from 'vitest'
import { useBulkConnectionActions } from '../useBulkConnectionActions'

function deferred<T>() {
  let resolve!: (v: T) => void
  let reject!: (e: unknown) => void
  const promise = new Promise<T>((res, rej) => {
    resolve = res
    reject = rej
  })
  return { promise, resolve, reject }
}

describe('useBulkConnectionActions', () => {
  it('starts idle with zero counters', () => {
    const b = useBulkConnectionActions()
    expect(b.running.value).toBe(false)
    expect(b.progress.value.total).toBe(0)
    expect(b.percent.value).toBe(0)
  })

  it('runs every id through the operation and records per-item success', async () => {
    const b = useBulkConnectionActions()
    const op = async (id: string) => `done:${id}`
    const result = await b.run(['a', 'b', 'c'], op)
    expect(result.completed).toBe(3)
    expect(result.succeeded).toBe(3)
    expect(result.failed).toBe(0)
    expect(result.results.map(r => r.id).sort()).toEqual(['a', 'b', 'c'])
    expect(result.results.every(r => r.ok)).toBe(true)
  })

  it('records failures without aborting the batch', async () => {
    const b = useBulkConnectionActions()
    const op = async (id: string) => {
      if (id === 'b')
        throw new Error('boom')
      return id
    }
    const result = await b.run(['a', 'b', 'c'], op)
    expect(result.succeeded).toBe(2)
    expect(result.failed).toBe(1)
    const failed = result.results.find(r => r.id === 'b')
    expect(failed?.ok).toBe(false)
    expect(failed?.error).toBe('boom')
  })

  it('caps concurrency at the given limit', async () => {
    const b = useBulkConnectionActions()
    let inflight = 0
    let peak = 0
    const gates: ReturnType<typeof deferred<void>>[] = []
    const op = async (id: string) => {
      inflight++
      peak = Math.max(peak, inflight)
      const idx = Number.parseInt(id, 10)
      const g = deferred<void>()
      gates[idx] = g
      await g.promise
      inflight--
    }
    const ids = Array.from({ length: 10 }, (_, i) => String(i))
    const runPromise = b.run(ids, op, { concurrency: 3 })
    // Yield to allow workers to start their first ops.
    await Promise.resolve()
    await Promise.resolve()
    // Resolve gates one by one so workers can pick up new ids.
    for (let i = 0; i < ids.length; i++) {
      // Wait until this index has actually started.
      while (!gates[i]) await new Promise(r => setTimeout(r, 0))
      gates[i].resolve()
      await Promise.resolve()
    }
    await runPromise
    expect(peak).toBeLessThanOrEqual(3)
    expect(b.progress.value.succeeded).toBe(10)
  })

  it('reports percent based on completed/total', async () => {
    const b = useBulkConnectionActions()
    await b.run(['a', 'b', 'c', 'd'], async () => undefined)
    expect(b.percent.value).toBe(100)
  })

  it('reset clears counters', async () => {
    const b = useBulkConnectionActions()
    await b.run(['a'], async () => undefined)
    b.reset()
    expect(b.progress.value.total).toBe(0)
    expect(b.progress.value.completed).toBe(0)
  })

  it('refuses to start a second run while one is in progress', async () => {
    const b = useBulkConnectionActions()
    const g = deferred<void>()
    const op = async () => {
      await g.promise
    }
    const first = b.run(['a'], op)
    await expect(b.run(['b'], async () => undefined)).rejects.toThrow(/already running/)
    g.resolve()
    await first
  })
})

import { computed, ref } from 'vue'

export interface BulkItemResult<T = unknown> {
  id: string
  ok: boolean
  error?: string
  value?: T
}

export interface BulkProgress<T = unknown> {
  total: number
  completed: number
  succeeded: number
  failed: number
  results: BulkItemResult<T>[]
}

interface RunOptions {
  concurrency?: number
}

export function useBulkConnectionActions() {
  const running = ref(false)
  const progress = ref<BulkProgress>({
    total: 0,
    completed: 0,
    succeeded: 0,
    failed: 0,
    results: [],
  })

  const percent = computed(() => {
    if (progress.value.total === 0)
      return 0
    return Math.round((progress.value.completed / progress.value.total) * 100)
  })

  function reset() {
    progress.value = { total: 0, completed: 0, succeeded: 0, failed: 0, results: [] }
  }

  async function run<T>(
    ids: string[],
    op: (id: string) => Promise<T>,
    options: RunOptions = {},
  ): Promise<BulkProgress<T>> {
    if (running.value)
      throw new Error('Bulk action already running')

    const concurrency = Math.max(1, options.concurrency ?? 8)
    running.value = true
    progress.value = {
      total: ids.length,
      completed: 0,
      succeeded: 0,
      failed: 0,
      results: [],
    }

    let cursor = 0
    async function worker() {
      while (cursor < ids.length) {
        const idx = cursor++
        const id = ids[idx]
        const result: BulkItemResult<T> = { id, ok: false }
        try {
          result.value = await op(id)
          result.ok = true
        }
        catch (e: unknown) {
          result.ok = false
          result.error = e instanceof Error ? e.message : String(e)
        }
        progress.value = {
          ...progress.value,
          completed: progress.value.completed + 1,
          succeeded: progress.value.succeeded + (result.ok ? 1 : 0),
          failed: progress.value.failed + (result.ok ? 0 : 1),
          results: [...progress.value.results, result],
        }
      }
    }

    try {
      const workers: Promise<void>[] = []
      for (let i = 0; i < Math.min(concurrency, ids.length); i++)
        workers.push(worker())
      await Promise.all(workers)
      return progress.value as BulkProgress<T>
    }
    finally {
      running.value = false
    }
  }

  return { running, progress, percent, run, reset }
}

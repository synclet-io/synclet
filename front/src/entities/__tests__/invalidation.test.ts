import * as queryKeys from '@shared/lib/queryKeys'
import { describe, expect, it } from 'vitest'

const entitySources = import.meta.glob<string>(['../**/*.ts', '!../**/__tests__/**'], {
  query: '?raw',
  import: 'default',
  eager: true,
})

const mutationsWithoutCacheEffects: Record<string, string> = {
  useTestSourceConnection: 'starts a check task; the result is read through the task query',
  useTestDestinationConnection: 'starts a check task; the result is read through the task query',
  useTestChannel: 'sends a test notification without changing stored data',
  useTestWebhook: 'sends a test delivery without changing stored data',
  useExportConfig: 'reads the workspace config',
  useDiscoverSchemaTask: 'invalidates schema changes and stream states once the discover task completes',
}

const keyFactoryNames = new Set(Object.keys(queryKeys))

interface MutationComposable {
  file: string
  name: string
  body: string
}

function mutationComposables(): MutationComposable[] {
  const result: MutationComposable[] = []
  for (const [file, source] of Object.entries(entitySources)) {
    const chunks = source.split(/^export function /m).slice(1)
    for (const chunk of chunks) {
      const name = chunk.match(/^(\w+)/)?.[1]
      if (name && chunk.includes('useMutation('))
        result.push({ file, name, body: chunk })
    }
  }
  return result
}

function onSuccessBody(body: string): string | undefined {
  const starts = ['onSuccess:', 'onSettled:'].map(hook => body.indexOf(hook)).filter(i => i !== -1)
  if (starts.length === 0)
    return undefined
  return body.slice(Math.min(...starts))
}

describe('mutation cache invalidation', () => {
  const composables = mutationComposables()

  it('finds the mutation composables', () => {
    expect(composables.length).toBeGreaterThan(20)
  })

  it.each(composables.map(c => [c.name, c] as const))('%s touches a key from queryKeys.ts on success', (name, composable) => {
    if (name in mutationsWithoutCacheEffects)
      return
    const onSuccess = onSuccessBody(composable.body)
    expect(onSuccess, `${name} in ${composable.file} has no onSuccess or onSettled`).toBeDefined()
    const referenced = Array.from(onSuccess!.matchAll(/\b(\w+Keys)\.\w+/g), m => m[1])
    expect(
      referenced.some(key => keyFactoryNames.has(key)),
      `${name} in ${composable.file} does not reference a key factory from queryKeys.ts in onSuccess`,
    ).toBe(true)
  })

  it('keeps the exemption list current', () => {
    const names = new Set(composables.map(c => c.name))
    for (const name of Object.keys(mutationsWithoutCacheEffects))
      expect(names, `${name} is exempt but no longer a mutation composable`).toContain(name)
  })
})

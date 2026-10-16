import { ref, watch } from 'vue'

export function cleanConfig(obj: Record<string, any>): Record<string, any> {
  const result: Record<string, any> = {}
  for (const [k, v] of Object.entries(obj)) {
    if (k.startsWith('__'))
      continue
    if (v && typeof v === 'object' && !Array.isArray(v)) {
      result[k] = cleanConfig(v)
    }
    else {
      result[k] = v
    }
  }
  return result
}

/**
 * Connector config edited either through the schema form or as raw JSON.
 * `values` is the form state; `jsonText` mirrors it while the form is shown.
 */
export function useConfigEditor() {
  const values = ref<Record<string, any>>({})
  const jsonMode = ref(false)
  const jsonText = ref('{}')

  watch(values, (val) => {
    if (!jsonMode.value)
      jsonText.value = JSON.stringify(val, null, 2)
  }, { deep: true })

  function reset(config: Record<string, unknown>) {
    values.value = { ...config }
    jsonText.value = JSON.stringify(config, null, 2)
  }

  function parseJson(): boolean {
    try {
      values.value = JSON.parse(jsonText.value)
      return true
    }
    catch {
      return false
    }
  }

  /** Toggles between form and JSON; returns false when the JSON left behind does not parse. */
  function toggleJson(): boolean {
    jsonMode.value = !jsonMode.value
    return jsonMode.value || parseJson()
  }

  /** The config to submit, without UI-only `__` keys. Throws when the JSON editor holds invalid JSON. */
  function resolve(): Record<string, any> {
    if (jsonMode.value && !parseJson())
      throw new Error('Invalid JSON')
    return cleanConfig(values.value)
  }

  return { values, jsonMode, jsonText, reset, toggleJson, resolve }
}

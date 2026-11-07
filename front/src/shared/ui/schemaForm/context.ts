import type { ValidationError } from '@shared/lib/jsonSchemaUtils'
import type { ComputedRef, InjectionKey, Ref } from 'vue'

/**
 * State shared by a root JsonSchemaForm and every form nested in it. Fields are
 * addressed by their dotted path from the root value.
 */
export interface SchemaFormContext {
  /** Set by `validate()`: every field shows its error, touched or not. */
  showAll: Ref<boolean>
  /** The `oneOf` branch the user selected, by field path. */
  selections: Record<string, number>
  /** JSON editors holding text that is not a JSON object, by field path. */
  invalidJson: Record<string, string>
  /** Errors of the whole root value. */
  errors: ComputedRef<ValidationError[]>
}

export const schemaFormContextKey: InjectionKey<SchemaFormContext> = Symbol('schemaFormContext')

export function joinFieldPath(path: string | undefined, key: string): string {
  return path ? `${path}.${key}` : key
}

export const inputBase = 'w-full px-3 py-2 border rounded-lg bg-surface text-heading text-sm placeholder:text-text-muted focus:outline-none focus:ring-2 focus:ring-primary/20 focus:border-primary'
export const inputClass = `${inputBase} border-border-strong`
export const inputErrorClass = `${inputBase} border-danger`
export const labelClass = 'block text-sm font-medium text-heading mb-1'
export const descriptionClass = 'text-xs text-text-muted mt-1 [&_a]:text-primary [&_a]:underline'

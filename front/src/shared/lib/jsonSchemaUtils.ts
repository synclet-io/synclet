export interface JsonSchema {
  type?: string
  properties?: Record<string, JsonSchema>
  required?: string[]
  title?: string
  description?: string
  default?: unknown
  enum?: unknown[]
  const?: unknown
  examples?: unknown[]
  pattern?: string
  minLength?: number
  maxLength?: number
  minimum?: number
  maximum?: number
  oneOf?: JsonSchema[]
  allOf?: JsonSchema[]
  if?: JsonSchema
  then?: JsonSchema
  else?: JsonSchema
  items?: JsonSchema
  order?: number
  airbyte_secret?: boolean
  multiline?: boolean
  [key: string]: unknown
}

export interface ValidationError {
  field: string
  message: string
}

export function getOrderedProperties(schema: JsonSchema): string[] {
  if (!schema.properties)
    return []
  const keys = Object.keys(schema.properties)

  return keys.sort((a, b) => {
    const orderA = schema.properties![a].order
    const orderB = schema.properties![b].order
    if (orderA != null && orderB != null)
      return (orderA as number) - (orderB as number)
    if (orderA != null)
      return -1
    if (orderB != null)
      return 1
    return a.localeCompare(b)
  })
}

export function resolveOneOf(schema: JsonSchema, currentValue: Record<string, unknown>): number {
  if (!schema.oneOf || schema.oneOf.length === 0)
    return -1

  for (let i = 0; i < schema.oneOf.length; i++) {
    const branch = schema.oneOf[i]
    if (!branch.properties)
      continue

    const constFields = Object.entries(branch.properties).filter(
      ([, propSchema]) => propSchema.const !== undefined,
    )

    if (constFields.length === 0)
      continue

    const allMatch = constFields.every(
      ([key, propSchema]) => currentValue[key] === propSchema.const,
    )
    if (allMatch)
      return i
  }

  return 0
}

export function getFieldDefaults(schema: JsonSchema): Record<string, unknown> {
  const defaults: Record<string, unknown> = {}
  if (!schema.properties)
    return defaults

  for (const [key, propSchema] of Object.entries(schema.properties)) {
    if (propSchema.const !== undefined) {
      // Const fields always get their fixed value.
      defaults[key] = propSchema.const
    }
    else if (propSchema.default !== undefined) {
      defaults[key] = propSchema.default
    }
    else if (propSchema.oneOf) {
      // Skip oneOf fields — defaults are handled by the sub-form when a branch is selected.
    }
    else if (propSchema.type === 'object' && propSchema.properties) {
      const nested = getFieldDefaults(propSchema)
      if (Object.keys(nested).length > 0) {
        defaults[key] = nested
      }
    }
  }

  return defaults
}

export function validateField(
  value: unknown,
  fieldSchema: JsonSchema,
  isRequired: boolean,
): ValidationError | null {
  const isEmpty = value === undefined || value === null || value === ''

  if (isRequired && isEmpty) {
    return { field: '', message: 'This field is required' }
  }

  if (isEmpty)
    return null

  if (typeof value === 'string') {
    if (fieldSchema.minLength != null && value.length < fieldSchema.minLength) {
      return { field: '', message: `Minimum length is ${fieldSchema.minLength}` }
    }
    if (fieldSchema.maxLength != null && value.length > fieldSchema.maxLength) {
      return { field: '', message: `Maximum length is ${fieldSchema.maxLength}` }
    }
    if (fieldSchema.pattern) {
      try {
        const re = new RegExp(fieldSchema.pattern)
        if (!re.test(value)) {
          return { field: '', message: `Must match pattern: ${fieldSchema.pattern}` }
        }
      }
      catch {
        // Invalid regex pattern from connector spec — skip validation
      }
    }
  }

  if (typeof value === 'number') {
    if (fieldSchema.minimum != null && value < fieldSchema.minimum) {
      return { field: '', message: `Minimum value is ${fieldSchema.minimum}` }
    }
    if (fieldSchema.maximum != null && value > fieldSchema.maximum) {
      return { field: '', message: `Maximum value is ${fieldSchema.maximum}` }
    }
  }

  return null
}

const UNDERSCORE_RE = /_/g
const CAMEL_CASE_RE = /([a-z])([A-Z])/g
const FIRST_CHAR_RE = /^./

export function humanizeKey(key: string): string {
  return key
    .replace(UNDERSCORE_RE, ' ')
    .replace(CAMEL_CASE_RE, '$1 $2')
    .replace(FIRST_CHAR_RE, s => s.toUpperCase())
}

export function mergeAllOf(schema: JsonSchema): JsonSchema {
  if (!schema.allOf)
    return schema
  const merged: JsonSchema = { ...schema }
  delete merged.allOf

  for (const sub of schema.allOf) {
    if (sub.properties) {
      merged.properties = { ...merged.properties, ...sub.properties }
    }
    if (sub.required) {
      merged.required = [...(merged.required || []), ...sub.required]
    }
    if (sub.title && !merged.title)
      merged.title = sub.title
    if (sub.description && !merged.description)
      merged.description = sub.description
  }

  return merged
}

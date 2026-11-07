import { isValidDateValue } from './dateFormats'

export interface JsonSchema {
  type?: string | string[]
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
  airbyte_hidden?: boolean
  format?: string
  pattern_descriptor?: string
  minItems?: number
  maxItems?: number
  uniqueItems?: boolean
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

/** The non-null type of a schema whose `type` may be a list such as `["string", "null"]`. */
export function schemaType(schema: JsonSchema): string | undefined {
  if (!Array.isArray(schema.type))
    return schema.type
  return schema.type.find(t => t !== 'null') ?? schema.type[0]
}

/** The schema with `allOf` merged and `type` reduced to its non-null type. */
export function normalizeSchema(schema: JsonSchema): JsonSchema {
  const merged = mergeAllOf(schema)
  const type = schemaType(merged)
  return type === merged.type ? merged : { ...merged, type }
}

/** The schema with `allOf` merged and each property normalized. */
export function normalizeProperties(schema: JsonSchema): JsonSchema {
  const merged = mergeAllOf(schema)
  if (!merged.properties)
    return merged
  const properties = Object.fromEntries(
    Object.entries(merged.properties).map(([key, prop]) => [key, normalizeSchema(prop)]),
  )
  return { ...merged, properties }
}

export function isPlainObject(value: unknown): value is Record<string, unknown> {
  return typeof value === 'object' && value !== null && !Array.isArray(value)
}

/** Whether a `oneOf` branch holds an object of fields rather than a single primitive value. */
export function isObjectBranch(branch: JsonSchema): boolean {
  return !!branch.properties || schemaType(branch) === 'object'
}

/**
 * The fields that tell an object branch apart: a `const`, or an `enum` with a
 * single value, as Airbyte specs write discriminators.
 */
export function branchDiscriminators(branch: JsonSchema): Array<[string, unknown]> {
  const result: Array<[string, unknown]> = []
  for (const [key, prop] of Object.entries(branch.properties ?? {})) {
    if (prop.const !== undefined)
      result.push([key, prop.const])
    else if (Array.isArray(prop.enum) && prop.enum.length === 1)
      result.push([key, prop.enum[0]])
  }
  return result
}

/** The value a `oneOf` field takes when its branch is selected. */
export function initialBranchValue(branch: JsonSchema): unknown {
  if (!isObjectBranch(branch))
    return branch.const ?? branch.default
  return Object.fromEntries(branchDiscriminators(branch))
}

function jsonType(value: unknown): string {
  if (Array.isArray(value))
    return 'array'
  if (value === null)
    return 'null'
  if (typeof value === 'number')
    return Number.isInteger(value) ? 'integer' : 'number'
  return typeof value
}

function typeAccepts(type: string | undefined, value: unknown): boolean {
  const actual = jsonType(value)
  return type === actual || (type === 'number' && actual === 'integer')
}

function branchFit(branch: JsonSchema, value: Record<string, unknown>): number {
  const props = branch.properties ?? {}
  const keys = Object.keys(value)
  const known = keys.filter(k => k in props).length
  const unknown = keys.length - known
  const missing = (branch.required ?? []).filter(k => !(k in value)).length
  return known * 4 - unknown * 8 - missing
}

/**
 * The index of the `oneOf` branch a stored value belongs to: the branch whose
 * discriminators it carries, or else the branch whose properties fit it best.
 * 0 when nothing matches; -1 when the schema has no `oneOf`.
 */
export function resolveOneOf(schema: JsonSchema, currentValue: unknown): number {
  const branches = schema.oneOf
  if (!branches || branches.length === 0)
    return -1

  if (!isPlainObject(currentValue)) {
    if (currentValue === undefined || currentValue === null || currentValue === '') {
      if (schema.default === undefined)
        return 0
      currentValue = schema.default
    }
    const byConst = branches.findIndex(b => !isObjectBranch(b) && b.const !== undefined && b.const === currentValue)
    if (byConst >= 0)
      return byConst
    const byType = branches.findIndex(b => !isObjectBranch(b) && b.const === undefined && typeAccepts(schemaType(b) ?? schemaType(schema), currentValue))
    return byType >= 0 ? byType : 0
  }

  const value = currentValue
  let best = 0
  let bestScore = Number.NEGATIVE_INFINITY
  for (let i = 0; i < branches.length; i++) {
    const branch = branches[i]
    if (!isObjectBranch(branch))
      continue
    const discriminators = branchDiscriminators(branch)
    if (discriminators.some(([key, expected]) => key in value && value[key] !== expected))
      continue
    if (discriminators.length > 0 && discriminators.every(([key, expected]) => value[key] === expected))
      return i
    if (Object.keys(value).length === 0)
      continue
    const score = branchFit(branch, value)
    if (score > bestScore) {
      best = i
      bestScore = score
    }
  }
  return best
}

/** Array text entries converted to the type of the array's items. */
export function coerceArrayItems(items: string[], itemSchema?: JsonSchema): unknown[] {
  const type = itemSchema ? schemaType(itemSchema) : undefined
  if (type !== 'integer' && type !== 'number')
    return items
  return items.map((item) => {
    const n = Number(item)
    return Number.isFinite(n) ? n : item
  })
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
    else if (schemaType(propSchema) === 'object' && propSchema.properties) {
      const nested = getFieldDefaults(propSchema)
      if (Object.keys(nested).length > 0) {
        defaults[key] = nested
      }
    }
  }

  return defaults
}

/** The value the API returns in place of a stored `airbyte_secret` value. */
export const SECRET_MASK = '********'

function error(message: string, field = ''): ValidationError {
  return { field, message }
}

function validateItems(items: unknown[], fieldSchema: JsonSchema): ValidationError | null {
  if (fieldSchema.minItems != null && items.length < fieldSchema.minItems)
    return error(`Select or enter at least ${fieldSchema.minItems} ${fieldSchema.minItems === 1 ? 'value' : 'values'}`)
  if (fieldSchema.maxItems != null && items.length > fieldSchema.maxItems)
    return error(`Select or enter at most ${fieldSchema.maxItems} ${fieldSchema.maxItems === 1 ? 'value' : 'values'}`)
  if (fieldSchema.uniqueItems) {
    const seen = new Set(items.map(item => JSON.stringify(item)))
    if (seen.size !== items.length)
      return error('Values must be unique')
  }
  if (fieldSchema.items) {
    const itemSchema: JsonSchema = {
      pattern_descriptor: fieldSchema.pattern_descriptor,
      airbyte_secret: fieldSchema.airbyte_secret,
      ...normalizeSchema(fieldSchema.items),
    }
    for (let i = 0; i < items.length; i++) {
      const err = validateField(items[i], itemSchema, false)
      if (err)
        return error(`Item ${i + 1}: ${err.message}`)
    }
  }
  return null
}

/** Validates one value against its own schema, without descending into nested objects or `oneOf` branches. */
export function validateField(
  value: unknown,
  fieldSchema: JsonSchema,
  isRequired: boolean,
): ValidationError | null {
  const isEmpty = value === undefined || value === null || value === ''

  if (isRequired && isEmpty)
    return error('This field is required')

  if (isEmpty)
    return null

  if (fieldSchema.airbyte_secret && value === SECRET_MASK)
    return null

  const type = schemaType(fieldSchema)

  if (fieldSchema.const !== undefined && value !== fieldSchema.const)
    return error(`Must be ${String(fieldSchema.const)}`)

  if (Array.isArray(fieldSchema.enum) && !fieldSchema.enum.includes(value))
    return error(`Must be one of: ${fieldSchema.enum.map(String).join(', ')}`)

  if ((type === 'integer' || type === 'number') && typeof value !== 'number')
    return error('Must be a number')
  if (type === 'integer' && typeof value === 'number' && !Number.isInteger(value))
    return error('Must be a whole number')
  if (type === 'array' && !Array.isArray(value))
    return error('Must be a list of values')
  if (type === 'object' && !isPlainObject(value))
    return error('Must be a JSON object')

  if (typeof value === 'string') {
    if (fieldSchema.minLength != null && value.length < fieldSchema.minLength)
      return error(`Minimum length is ${fieldSchema.minLength}`)
    if (fieldSchema.maxLength != null && value.length > fieldSchema.maxLength)
      return error(`Maximum length is ${fieldSchema.maxLength}`)
    if (fieldSchema.pattern) {
      let re: RegExp | null = null
      try {
        re = new RegExp(fieldSchema.pattern)
      }
      catch {
        // Invalid regex pattern from connector spec — skip validation
      }
      if (re && !re.test(value)) {
        return error(fieldSchema.pattern_descriptor
          ? `Must match the format ${fieldSchema.pattern_descriptor}`
          : `Must match pattern: ${fieldSchema.pattern}`)
      }
    }
    const strictDate = !fieldSchema.pattern
    if (fieldSchema.format === 'date' && !isValidDateValue(value, 'date', strictDate))
      return error('Must be a valid date')
    if (fieldSchema.format === 'date-time' && !isValidDateValue(value, 'date-time', strictDate))
      return error('Must be a valid date and time')
  }

  if (typeof value === 'number') {
    if (fieldSchema.minimum != null && value < fieldSchema.minimum)
      return error(`Minimum value is ${fieldSchema.minimum}`)
    if (fieldSchema.maximum != null && value > fieldSchema.maximum)
      return error(`Maximum value is ${fieldSchema.maximum}`)
  }

  if (Array.isArray(value))
    return validateItems(value, fieldSchema)

  return null
}

export interface ValidateOptions {
  /** The `oneOf` branch the user selected for the field at `path`, if any. */
  selectedBranch?: (path: string) => number | undefined
}

function joinPath(path: string, key: string): string {
  return path ? `${path}.${key}` : key
}

/**
 * Validates every field of an object value against an object schema, descending
 * into nested objects, the selected branch of each `oneOf` and array items.
 * Hidden and `const` fields are skipped: the form sets them itself.
 */
export function validateObjectValue(
  value: Record<string, unknown>,
  schema: JsonSchema,
  path = '',
  options: ValidateOptions = {},
): ValidationError[] {
  const resolved = mergeAllOf(schema)
  const required = new Set(resolved.required ?? [])
  const errors: ValidationError[] = []
  for (const [key, prop] of Object.entries(resolved.properties ?? {})) {
    if (prop.airbyte_hidden || prop.const !== undefined)
      continue
    errors.push(...validateValue(value[key], prop, required.has(key), joinPath(path, key), options))
  }
  return errors
}

/** Validates a value and everything nested in it; each error's `field` is the dotted path of the invalid field. */
export function validateValue(
  value: unknown,
  rawSchema: JsonSchema,
  isRequired: boolean,
  path: string,
  options: ValidateOptions = {},
): ValidationError[] {
  const schema = normalizeSchema(rawSchema)
  const isEmpty = value === undefined || value === null

  if (schema.oneOf && schema.oneOf.length > 0) {
    const index = options.selectedBranch?.(path) ?? resolveOneOf(schema, value)
    const branch = normalizeSchema(schema.oneOf[index] ?? schema.oneOf[0])
    if (!isObjectBranch(branch)) {
      const err = validateField(value, { ...branch, type: schemaType(branch) ?? schemaType(schema) }, isRequired)
      return err ? [{ ...err, field: path }] : []
    }
    if (isEmpty)
      return isRequired ? validateObjectValue({}, branch, path, options) : []
    if (!isPlainObject(value))
      return [error('Must be an object', path)]
    return validateObjectValue(value, branch, path, options)
  }

  if (schemaType(schema) === 'object' && schema.properties) {
    if (isEmpty)
      return isRequired ? validateObjectValue({}, schema, path, options) : []
    if (!isPlainObject(value))
      return [error('Must be an object', path)]
    return validateObjectValue(value, schema, path, options)
  }

  const err = validateField(value, schema, isRequired)
  return err ? [{ ...err, field: path }] : []
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

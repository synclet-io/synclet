import type { JsonSchema } from '../jsonSchemaUtils'
import { describe, expect, it } from 'vitest'
import {
  getFieldDefaults,
  getOrderedProperties,
  humanizeKey,
  mergeAllOf,
  resolveOneOf,
  validateField,
} from '../jsonSchemaUtils'

describe('getOrderedProperties', () => {
  it('returns properties in order field sequence', () => {
    const schema = {
      properties: {
        host: { type: 'string', order: 1 },
        port: { type: 'integer', order: 0 },
        database: { type: 'string', order: 2 },
      },
    }
    expect(getOrderedProperties(schema)).toEqual(['port', 'host', 'database'])
  })

  it('falls back to alphabetical order when no order field', () => {
    const schema = {
      properties: {
        zebra: { type: 'string' },
        alpha: { type: 'string' },
        middle: { type: 'string' },
      },
    }
    expect(getOrderedProperties(schema)).toEqual(['alpha', 'middle', 'zebra'])
  })

  it('handles mix of ordered and unordered properties', () => {
    const schema = {
      properties: {
        unordered_b: { type: 'string' },
        ordered_first: { type: 'string', order: 0 },
        unordered_a: { type: 'string' },
        ordered_second: { type: 'string', order: 1 },
      },
    }
    const result = getOrderedProperties(schema)
    expect(result[0]).toBe('ordered_first')
    expect(result[1]).toBe('ordered_second')
    expect(result.slice(2)).toEqual(['unordered_a', 'unordered_b'])
  })

  it('returns empty array for schema with no properties', () => {
    expect(getOrderedProperties({})).toEqual([])
    expect(getOrderedProperties({ type: 'object' })).toEqual([])
  })
})

describe('resolveOneOf', () => {
  const schema: JsonSchema = {
    oneOf: [
      {
        title: 'Option A',
        properties: {
          type: { const: 'a' },
          value_a: { type: 'string' },
        },
      },
      {
        title: 'Option B',
        properties: {
          type: { const: 'b' },
          value_b: { type: 'string' },
        },
      },
    ],
  }

  it('returns correct branch index based on const discriminator match', () => {
    expect(resolveOneOf(schema, { type: 'b' })).toBe(1)
    expect(resolveOneOf(schema, { type: 'a' })).toBe(0)
  })

  it('returns first branch (default) when no value matches', () => {
    expect(resolveOneOf(schema, { type: 'unknown' })).toBe(0)
    expect(resolveOneOf(schema, {})).toBe(0)
  })

  it('handles nested oneOf with multiple const fields', () => {
    const multiConst: JsonSchema = {
      oneOf: [
        {
          properties: {
            auth: { const: 'oauth' },
            version: { const: 'v2' },
          },
        },
        {
          properties: {
            auth: { const: 'token' },
            version: { const: 'v1' },
          },
        },
      ],
    }
    expect(resolveOneOf(multiConst, { auth: 'token', version: 'v1' })).toBe(1)
    expect(resolveOneOf(multiConst, { auth: 'oauth', version: 'v1' })).toBe(0) // partial match falls through
  })

  it('returns -1 when oneOf array is empty', () => {
    expect(resolveOneOf({ oneOf: [] }, {})).toBe(-1)
    expect(resolveOneOf({}, {})).toBe(-1)
  })
})

describe('getFieldDefaults', () => {
  it('extracts top-level default values', () => {
    const schema = {
      properties: {
        host: { type: 'string', default: 'localhost' },
        port: { type: 'integer', default: 5432 },
        name: { type: 'string' },
      },
    }
    expect(getFieldDefaults(schema)).toEqual({ host: 'localhost', port: 5432 })
  })

  it('extracts nested object defaults recursively', () => {
    const schema = {
      properties: {
        connection: {
          type: 'object',
          properties: {
            host: { type: 'string', default: '127.0.0.1' },
            ssl: { type: 'boolean', default: true },
          },
        },
      },
    }
    expect(getFieldDefaults(schema)).toEqual({
      connection: { host: '127.0.0.1', ssl: true },
    })
  })

  it('returns empty object when no defaults exist', () => {
    const schema = {
      properties: {
        host: { type: 'string' },
        port: { type: 'integer' },
      },
    }
    expect(getFieldDefaults(schema)).toEqual({})
  })

  it('handles boolean, number, string, and array defaults', () => {
    const schema = {
      properties: {
        enabled: { type: 'boolean', default: false },
        count: { type: 'number', default: 0 },
        label: { type: 'string', default: '' },
        tags: { type: 'array', default: ['a', 'b'] },
      },
    }
    expect(getFieldDefaults(schema)).toEqual({
      enabled: false,
      count: 0,
      label: '',
      tags: ['a', 'b'],
    })
  })
})

describe('validateField', () => {
  it('returns error for empty required field', () => {
    const err = validateField('', { type: 'string' }, true)
    expect(err).not.toBeNull()
    expect(err!.message).toContain('required')
  })

  it('returns error for undefined required field', () => {
    expect(validateField(undefined, { type: 'string' }, true)).not.toBeNull()
    expect(validateField(null, { type: 'string' }, true)).not.toBeNull()
  })

  it('returns no error for filled required field', () => {
    expect(validateField('hello', { type: 'string' }, true)).toBeNull()
  })

  it('validates pattern regex — valid', () => {
    expect(validateField('abc123', { type: 'string', pattern: '^[a-z0-9]+$' }, false)).toBeNull()
  })

  it('validates pattern regex — invalid', () => {
    const err = validateField('ABC!', { type: 'string', pattern: '^[a-z0-9]+$' }, false)
    expect(err).not.toBeNull()
    expect(err!.message).toContain('pattern')
  })

  it('validates minLength constraint', () => {
    expect(validateField('ab', { type: 'string', minLength: 3 }, false)).not.toBeNull()
    expect(validateField('abc', { type: 'string', minLength: 3 }, false)).toBeNull()
  })

  it('validates maxLength constraint', () => {
    expect(validateField('abcd', { type: 'string', maxLength: 3 }, false)).not.toBeNull()
    expect(validateField('abc', { type: 'string', maxLength: 3 }, false)).toBeNull()
  })

  it('validates minimum for numbers', () => {
    expect(validateField(0, { type: 'number', minimum: 1 }, false)).not.toBeNull()
    expect(validateField(1, { type: 'number', minimum: 1 }, false)).toBeNull()
  })

  it('validates maximum for numbers', () => {
    expect(validateField(100, { type: 'number', maximum: 99 }, false)).not.toBeNull()
    expect(validateField(99, { type: 'number', maximum: 99 }, false)).toBeNull()
  })

  it('returns no errors when no constraints exist', () => {
    expect(validateField('anything', { type: 'string' }, false)).toBeNull()
    expect(validateField(42, { type: 'number' }, false)).toBeNull()
  })

  it('returns null for empty non-required field', () => {
    expect(validateField('', { type: 'string', minLength: 3 }, false)).toBeNull()
  })
})

describe('humanizeKey', () => {
  it('converts snake_case to title case', () => {
    expect(humanizeKey('database_name')).toBe('Database name')
  })

  it('converts camelCase to title case', () => {
    expect(humanizeKey('databaseName')).toBe('Database Name')
  })
})

describe('mergeAllOf', () => {
  it('merges allOf schemas into one', () => {
    const schema: JsonSchema = {
      allOf: [
        { properties: { a: { type: 'string' } }, required: ['a'] },
        { properties: { b: { type: 'number' } }, required: ['b'] },
      ],
    }
    const merged = mergeAllOf(schema)
    expect(merged.properties).toEqual({ a: { type: 'string' }, b: { type: 'number' } })
    expect(merged.required).toEqual(['a', 'b'])
  })

  it('returns schema unchanged if no allOf', () => {
    const schema = { type: 'object', properties: { x: { type: 'string' } } }
    expect(mergeAllOf(schema)).toEqual(schema)
  })
})

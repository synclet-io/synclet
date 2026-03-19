export interface SchemaField {
  name: string
  path: string[]
  type: string // 'string' | 'integer' | 'number' | 'boolean' | 'object' | 'array'
  children?: SchemaField[]
}

export function parseJsonSchema(
  schema: Record<string, any>,
  parentPath: string[] = [],
  depth: number = 0,
): SchemaField[] {
  if (depth >= 10)
    return [] // Depth limit
  const properties = schema.properties || {}
  return Object.entries(properties).map(([name, prop]: [string, any]) => {
    const path = [...parentPath, name]
    const type = Array.isArray(prop.type)
      ? prop.type.find((t: string) => t !== 'null') || 'string'
      : prop.type || 'string'
    const field: SchemaField = { name, path, type }
    if (type === 'object' && prop.properties) {
      field.children = parseJsonSchema(prop, path, depth + 1)
    }
    if (type === 'array' && prop.items?.properties) {
      field.children = parseJsonSchema(prop.items, path, depth + 1)
    }
    return field
  })
}

// Returns all leaf field paths (no object/array types with children)
export function getLeafFields(fields: SchemaField[]): SchemaField[] {
  const leaves: SchemaField[] = []
  function walk(fs: SchemaField[]) {
    for (const f of fs) {
      if (f.children && f.children.length > 0)
        walk(f.children)
      else leaves.push(f)
    }
  }
  walk(fields)
  return leaves
}

// Converts field path to dot-joined string for Set keys
export function pathKey(path: string[]): string {
  return path.join('.')
}

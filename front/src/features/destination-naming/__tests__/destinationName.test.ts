/* eslint-disable no-template-curly-in-string -- ${SOURCE_NAMESPACE} is the literal placeholder under test */
import { describe, expect, it } from 'vitest'
import { destinationName, formatStreamName, namespaceModeFromSettings, namespaceSettingsFromMode } from '../destinationName'

const users = { namespace: 'public', name: 'users' }

describe('destinationName', () => {
  it.each([
    ['source keeps the source namespace', 'source', '', '', { namespace: 'public', name: 'users' }],
    ['source with a prefix prefixes the name only', 'source', '', 'production_', { namespace: 'public', name: 'production_users' }],
    ['destination clears the namespace', 'destination', 'ignored_${SOURCE_NAMESPACE}', 'p_', { namespace: '', name: 'p_users' }],
    ['custom replaces ${SOURCE_NAMESPACE}', 'custom', 'prod_${SOURCE_NAMESPACE}', '', { namespace: 'prod_public', name: 'users' }],
    ['custom replaces every ${SOURCE_NAMESPACE}', 'custom', '${SOURCE_NAMESPACE}_${SOURCE_NAMESPACE}', '', { namespace: 'public_public', name: 'users' }],
    ['custom without the placeholder is static', 'custom', 'analytics', 'production_', { namespace: 'analytics', name: 'production_users' }],
    ['custom with an empty format gives an empty namespace', 'custom', '', '', { namespace: '', name: 'users' }],
  ] as const)('%s', (_, def, format, prefix, expected) => {
    expect(destinationName(users, { namespaceDefinition: def, customNamespaceFormat: format, streamPrefix: prefix })).toEqual(expected)
  })

  it('keeps an empty source namespace empty in custom mode', () => {
    expect(destinationName({ namespace: '', name: 'users' }, { namespaceDefinition: 'custom', customNamespaceFormat: 'x_${SOURCE_NAMESPACE}', streamPrefix: '' }))
      .toEqual({ namespace: 'x_', name: 'users' })
  })
})

describe('formatStreamName', () => {
  it('joins namespace and name', () => {
    expect(formatStreamName({ namespace: 'analytics', name: 'production_users' })).toBe('analytics.production_users')
  })
  it('shows the name alone without a namespace', () => {
    expect(formatStreamName({ namespace: '', name: 'users' })).toBe('users')
  })
})

describe('namespaceModeFromSettings', () => {
  it.each([
    ['source', '', { mode: 'source', namespacePrefix: '' }],
    ['destination', 'prod_${SOURCE_NAMESPACE}', { mode: 'destination', namespacePrefix: '' }],
    ['custom', 'prod_${SOURCE_NAMESPACE}', { mode: 'source_prefix', namespacePrefix: 'prod_' }],
    ['custom', '${SOURCE_NAMESPACE}', { mode: 'source_prefix', namespacePrefix: '' }],
    ['custom', '$x_${SOURCE_NAMESPACE}', { mode: 'custom', namespacePrefix: '' }],
    ['custom', 'prod_${SOURCE_NAMESPACE}_raw', { mode: 'custom', namespacePrefix: '' }],
    ['custom', 'a_${SOURCE_NAMESPACE}${SOURCE_NAMESPACE}', { mode: 'custom', namespacePrefix: '' }],
    ['custom', 'analytics', { mode: 'custom', namespacePrefix: '' }],
    ['custom', '', { mode: 'custom', namespacePrefix: '' }],
  ] as const)('%s %s', (def, format, expected) => {
    expect(namespaceModeFromSettings(def, format)).toEqual(expected)
  })
})

describe('namespaceSettingsFromMode', () => {
  it('stores a source prefix as the custom format <prefix>${SOURCE_NAMESPACE}', () => {
    expect(namespaceSettingsFromMode('source_prefix', 'prod_', 'old')).toEqual({ namespaceDefinition: 'custom', customNamespaceFormat: 'prod_${SOURCE_NAMESPACE}' })
  })
  it('keeps the custom format for the other modes', () => {
    expect(namespaceSettingsFromMode('custom', '', 'analytics')).toEqual({ namespaceDefinition: 'custom', customNamespaceFormat: 'analytics' })
    expect(namespaceSettingsFromMode('source', '', 'analytics')).toEqual({ namespaceDefinition: 'source', customNamespaceFormat: 'analytics' })
    expect(namespaceSettingsFromMode('destination', '', '')).toEqual({ namespaceDefinition: 'destination', customNamespaceFormat: '' })
  })
})

import type { NamespaceDefinition } from '@entities/connection'

// eslint-disable-next-line no-template-curly-in-string -- the literal placeholder the backend replaces
export const SOURCE_NAMESPACE_PLACEHOLDER = '${SOURCE_NAMESPACE}'

export interface StreamName {
  namespace: string
  name: string
}

export interface NamingSettings {
  namespaceDefinition: NamespaceDefinition
  customNamespaceFormat: string
  streamPrefix: string
}

/** UI namespace mode; `source_prefix` is stored as the custom format `<prefix>` followed by the placeholder. */
export type NamespaceMode = 'source' | 'source_prefix' | 'destination' | 'custom'

const SOURCE_PREFIX_FORMAT = /^([^$]*)\$\{SOURCE_NAMESPACE\}$/

/** Mirrors ApplyNamespaceAndPrefix in modules/pipeline/pipelineservice/namespace.go. */
export function destinationName(stream: StreamName, settings: NamingSettings): StreamName {
  const name = settings.streamPrefix + stream.name
  switch (settings.namespaceDefinition) {
    case 'destination':
      return { namespace: '', name }
    case 'custom':
      return { namespace: settings.customNamespaceFormat.split(SOURCE_NAMESPACE_PLACEHOLDER).join(stream.namespace), name }
    default:
      return { namespace: stream.namespace, name }
  }
}

export function formatStreamName(stream: StreamName): string {
  return stream.namespace ? `${stream.namespace}.${stream.name}` : stream.name
}

export function namespaceModeFromSettings(def: NamespaceDefinition, customFormat: string): { mode: NamespaceMode, namespacePrefix: string } {
  if (def !== 'custom')
    return { mode: def, namespacePrefix: '' }
  const match = SOURCE_PREFIX_FORMAT.exec(customFormat)
  return match ? { mode: 'source_prefix', namespacePrefix: match[1]! } : { mode: 'custom', namespacePrefix: '' }
}

export function namespaceSettingsFromMode(mode: NamespaceMode, namespacePrefix: string, customFormat: string): { namespaceDefinition: NamespaceDefinition, customNamespaceFormat: string } {
  if (mode === 'source_prefix')
    return { namespaceDefinition: 'custom', customNamespaceFormat: namespacePrefix + SOURCE_NAMESPACE_PLACEHOLDER }
  return { namespaceDefinition: mode, customNamespaceFormat: customFormat }
}

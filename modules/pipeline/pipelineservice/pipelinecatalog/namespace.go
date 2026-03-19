package pipelinecatalog

import (
	"strings"

	"github.com/synclet-io/synclet/modules/pipeline/pipelineservice"
	"github.com/synclet-io/synclet/pkg/protocol"
)

// ApplyNamespaceAndPrefix modifies a catalog in-place to apply namespace
// rewriting and stream prefix per connection settings.
func ApplyNamespaceAndPrefix(
	catalog *protocol.ConfiguredAirbyteCatalog,
	nsDef pipelineservice.NamespaceDefinition,
	customFormat *string,
	prefix *string,
) {
	for i := range catalog.Streams {
		stream := &catalog.Streams[i]

		// Apply stream prefix to name.
		if prefix != nil && *prefix != "" {
			stream.Stream.Name = *prefix + stream.Stream.Name
		}

		// Apply namespace rewriting.
		switch nsDef {
		case pipelineservice.NamespaceDefinitionSource:
			// No-op: keep original namespace from source.
		case pipelineservice.NamespaceDefinitionDestination:
			// Clear namespace so destination uses its default.
			stream.Stream.Namespace = ""
		case pipelineservice.NamespaceDefinitionCustom:
			if customFormat != nil {
				stream.Stream.Namespace = strings.ReplaceAll(
					*customFormat,
					"${SOURCE_NAMESPACE}",
					stream.Stream.Namespace,
				)
			}
		}
	}
}

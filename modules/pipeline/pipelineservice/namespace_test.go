package pipelinecatalog

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/synclet-io/synclet/modules/pipeline/pipelineservice"
	"github.com/synclet-io/synclet/pkg/protocol"
)

func namespaceCatalog() *protocol.ConfiguredAirbyteCatalog {
	return &protocol.ConfiguredAirbyteCatalog{Streams: []protocol.ConfiguredAirbyteStream{
		{Stream: protocol.AirbyteStream{Namespace: "public", Name: "users"}},
		{Stream: protocol.AirbyteStream{Namespace: "", Name: "orders"}},
	}}
}

func streamIDs(catalog *protocol.ConfiguredAirbyteCatalog) [][2]string {
	out := make([][2]string, len(catalog.Streams))
	for i, s := range catalog.Streams {
		out[i] = [2]string{s.Stream.Namespace, s.Stream.Name}
	}

	return out
}

func TestApplyNamespaceAndPrefix(t *testing.T) {
	prefix := "raw_"
	empty := ""
	custom := "tenant_${SOURCE_NAMESPACE}"

	tests := []struct {
		name         string
		nsDef        pipelineservice.NamespaceDefinition
		customFormat *string
		prefix       *string
		want         [][2]string
	}{
		{
			name:   "source keeps namespace and applies prefix",
			nsDef:  pipelineservice.NamespaceDefinitionSource,
			prefix: &prefix,
			want:   [][2]string{{"public", "raw_users"}, {"", "raw_orders"}},
		},
		{
			name:  "destination clears namespace",
			nsDef: pipelineservice.NamespaceDefinitionDestination,
			want:  [][2]string{{"", "users"}, {"", "orders"}},
		},
		{
			name:         "custom format substitutes source namespace",
			nsDef:        pipelineservice.NamespaceDefinitionCustom,
			customFormat: &custom,
			prefix:       &empty,
			want:         [][2]string{{"tenant_public", "users"}, {"tenant_", "orders"}},
		},
		{
			name:  "custom without format leaves namespace untouched",
			nsDef: pipelineservice.NamespaceDefinitionCustom,
			want:  [][2]string{{"public", "users"}, {"", "orders"}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			catalog := namespaceCatalog()

			ApplyNamespaceAndPrefix(catalog, tt.nsDef, tt.customFormat, tt.prefix)

			assert.Equal(t, tt.want, streamIDs(catalog))
		})
	}
}

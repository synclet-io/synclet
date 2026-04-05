package pipelineexec

import (
	"encoding/json"

	"github.com/google/uuid"

	"github.com/synclet-io/synclet/modules/pipeline/pipelineservice"
	"github.com/synclet-io/synclet/pkg/protocol"
)

// SyncBundle contains all pre-loaded data needed to execute a sync.
// Populated by ClaimJobBundleResult from ExecutorBackend.
type SyncBundle struct {
	Job                   *pipelineservice.Job
	ConnectionID          uuid.UUID
	WorkspaceID           uuid.UUID
	SourceID              uuid.UUID
	DestinationID         uuid.UUID
	SourceImage           string
	SourceConfig          json.RawMessage // Already decrypted
	DestImage             string
	DestConfig            json.RawMessage                    // Already decrypted
	ConfiguredCatalog     *protocol.ConfiguredAirbyteCatalog // Already unmarshaled
	StateBlob             json.RawMessage                    // May be nil
	SourceRuntimeConfig   string                             // JSON
	DestRuntimeConfig     string                             // JSON
	NamespaceDefinition   pipelineservice.NamespaceDefinition
	CustomNamespaceFormat *string
	StreamPrefix          *string
}

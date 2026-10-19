package pipelineexec

import (
	"encoding/json"
	"errors"
	"fmt"

	"github.com/synclet-io/synclet/modules/pipeline/pipelineservice"
	"github.com/synclet-io/synclet/pkg/mapper"
	"github.com/synclet-io/synclet/pkg/protocol"
)

// RecordTransformer applies each configured stream's mappers and field
// selection to the RECORDs the source emits, keyed by the source's own stream
// names. Streams with mappers run their compiled chain, which ends with the
// field selection; streams with only selected fields are filtered to them;
// other streams pass unchanged.
type RecordTransformer struct {
	streams map[protocol.StreamDescriptor]*streamTransform
	sync    mapper.SyncInfo
}

type streamTransform struct {
	name     string
	chain    *mapper.Chain
	selected []protocol.SelectedField
}

// NewRecordTransformer compiles the mappers of every stream in catalog with
// salt as the hash salt; sync is what the mappers' expressions see as `sync`.
// A stream whose mappers do not compile is an error naming the stream.
func NewRecordTransformer(catalog *protocol.ConfiguredAirbyteCatalog, salt []byte, sync mapper.SyncInfo) (*RecordTransformer, error) {
	transformer := &RecordTransformer{
		streams: make(map[protocol.StreamDescriptor]*streamTransform),
		sync:    sync,
	}

	if catalog == nil {
		return transformer, nil
	}

	for i := range catalog.Streams {
		stream := &catalog.Streams[i]
		key := protocol.StreamDescriptor{Name: stream.Stream.Name, Namespace: stream.Stream.Namespace}
		name := streamDisplayName(key)

		if !pipelineservice.HasMappers(stream) {
			if len(stream.SelectedFields) > 0 {
				transformer.streams[key] = &streamTransform{name: name, selected: stream.SelectedFields}
			}

			continue
		}

		chain, result, err := pipelineservice.CompileStreamMappers(stream, stream.Stream.JSONSchema, salt)
		if err != nil {
			return nil, fmt.Errorf("stream %q: %w", name, err)
		}

		if len(result.Errors) > 0 {
			return nil, fmt.Errorf("stream %q: mappers do not compile: %s", name, pipelineservice.FormatMapperIssues(result.Errors))
		}

		transformer.streams[key] = &streamTransform{name: name, chain: chain}
	}

	return transformer, nil
}

// Apply transforms rec.Data in place. keep is false when a filter or an
// on_error skip drops the record. A step failing under on_error fail returns
// an error naming the stream and the step.
func (t *RecordTransformer) Apply(rec *protocol.AirbyteRecordMessage) (keep bool, err error) {
	stream, ok := t.streams[protocol.StreamDescriptor{Name: rec.Stream, Namespace: rec.Namespace}]
	if !ok {
		return true, nil
	}

	if stream.chain == nil {
		filtered, filterErr := pipelineservice.FilterRecordData(rec.Data, stream.selected)
		if filterErr == nil {
			rec.Data = filtered
		}

		return true, nil
	}

	data, err := mapper.DecodeRecord(rec.Data)
	if err != nil {
		return false, fmt.Errorf("stream %q: %w", stream.name, err)
	}

	out, keep, err := stream.chain.Run(data, t.sync)
	if err != nil {
		var stepErr *mapper.StepError
		if errors.As(err, &stepErr) {
			return false, fmt.Errorf("stream %q: %w", stream.name, stepErr)
		}

		return false, fmt.Errorf("stream %q: running mappers: %w", stream.name, err)
	}

	if !keep {
		return false, nil
	}

	encoded, err := json.Marshal(out)
	if err != nil {
		return false, fmt.Errorf("stream %q: encoding mapped record: %w", stream.name, err)
	}

	rec.Data = encoded

	return true, nil
}

func streamDisplayName(desc protocol.StreamDescriptor) string {
	if desc.Namespace == "" {
		return desc.Name
	}

	return desc.Namespace + "." + desc.Name
}

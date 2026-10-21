package pipelinecatalog

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/saturn4er/boilerplate-go/lib/filter"

	"github.com/synclet-io/synclet/modules/pipeline/pipelineservice"
	"github.com/synclet-io/synclet/pkg/mapper"
	"github.com/synclet-io/synclet/pkg/protocol"
)

// ValidateMappersParams holds an unsaved mapper list for one stream, with an
// optional sample record to run through it. The stream's source is the
// connection's when ConnectionID is set, otherwise SourceID, with
// ConnectionName standing for the name of a connection not created yet. Both
// the connection and the source must belong to WorkspaceID.
type ValidateMappersParams struct {
	WorkspaceID    uuid.UUID
	ConnectionID   uuid.UUID
	SourceID       uuid.UUID
	ConnectionName string
	Stream         protocol.ConfiguredAirbyteStream
	SampleRecord   json.RawMessage
}

// ValidateMappersResult is the compile result of the mapper list and, when a
// sample record was given and the list compiled, the sample's run.
type ValidateMappersResult struct {
	Errors           []mapper.Issue
	Warnings         []mapper.Issue
	OutputSchema     json.RawMessage
	OutputPrimaryKey [][]string
	SampleOutput     json.RawMessage
	SampleKept       bool
	SampleError      string
}

// ValidateMappers compiles a stream's mapper list against the stream's schema
// in the source's latest discovery, the way a save does, without storing it.
// The preview hashes without a salt.
type ValidateMappers struct {
	storage pipelineservice.Storage
}

// NewValidateMappers creates a new ValidateMappers use case.
func NewValidateMappers(storage pipelineservice.Storage) *ValidateMappers {
	return &ValidateMappers{storage: storage}
}

// Execute compiles the mapper list and runs the sample record through it.
func (uc *ValidateMappers) Execute(ctx context.Context, params ValidateMappersParams) (*ValidateMappersResult, error) {
	sourceID, connectionName, err := uc.resolveSource(ctx, params)
	if err != nil {
		return nil, err
	}

	availableStreams, err := latestDiscoveredStreams(ctx, uc.storage, sourceID)
	if err != nil {
		return nil, fmt.Errorf("getting latest catalog: %w", err)
	}

	stream := params.Stream
	key := streamKey(stream.Stream.Namespace, stream.Stream.Name)

	available, ok := availableStreams[key]
	if !ok {
		return nil, streamValidationError("stream %q not found in discovered catalog", key)
	}

	if len(available.SourceDefinedPrimaryKey) > 0 {
		stream.PrimaryKey = available.SourceDefinedPrimaryKey
	}

	if available.SourceDefinedCursor {
		stream.CursorField = available.DefaultCursorField
	}

	if len(stream.SelectedFields) > 0 {
		stream.SelectedFields = pipelineservice.ForceIncludeFields(stream.SelectedFields, stream.CursorField, stream.PrimaryKey)
	}

	chain, compiled, err := pipelineservice.CompileStreamMappers(&stream, available.JSONSchema, nil)
	if err != nil {
		return &ValidateMappersResult{ //nolint:nilerr // undecodable mappers are a result issue, not a call failure
			Errors: []mapper.Issue{{Step: mapper.SelectionStep, Message: err.Error()}},
		}, nil
	}

	result := &ValidateMappersResult{
		Errors:   compiled.Errors,
		Warnings: compiled.Warnings,
	}

	if chain == nil {
		return result, nil
	}

	result.OutputSchema = compiled.Schema
	result.OutputPrimaryKey = compiled.PrimaryKey

	if len(params.SampleRecord) == 0 {
		return result, nil
	}

	record, err := mapper.DecodeRecord(params.SampleRecord)
	if err != nil {
		result.SampleError = err.Error()

		return result, nil //nolint:nilerr // an undecodable sample is reported in the result
	}

	out, kept, err := chain.Run(record, mapper.SyncInfo{
		StartedAt:  time.Now(),
		JobID:      "preview",
		Connection: connectionName,
	})
	if err != nil {
		var stepErr *mapper.StepError
		if !errors.As(err, &stepErr) {
			return nil, fmt.Errorf("running sample record: %w", err)
		}

		result.SampleError = stepErr.Error()

		return result, nil
	}

	result.SampleKept = kept

	if kept {
		result.SampleOutput, err = json.Marshal(out)
		if err != nil {
			return nil, fmt.Errorf("marshaling sample output: %w", err)
		}
	}

	return result, nil
}

func (uc *ValidateMappers) resolveSource(ctx context.Context, params ValidateMappersParams) (uuid.UUID, string, error) {
	if params.ConnectionID != uuid.Nil {
		conn, err := uc.storage.Connections().First(ctx, &pipelineservice.ConnectionFilter{
			ID:          filter.Equals(params.ConnectionID),
			WorkspaceID: filter.Equals(params.WorkspaceID),
		})
		if err != nil {
			return uuid.Nil, "", fmt.Errorf("getting connection: %w", err)
		}

		return conn.SourceID, conn.Name, nil
	}

	src, err := uc.storage.Sources().First(ctx, &pipelineservice.SourceFilter{
		ID:          filter.Equals(params.SourceID),
		WorkspaceID: filter.Equals(params.WorkspaceID),
	})
	if err != nil {
		return uuid.Nil, "", fmt.Errorf("getting source: %w", err)
	}

	return src.ID, params.ConnectionName, nil
}

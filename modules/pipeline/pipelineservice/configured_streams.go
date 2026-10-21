package pipelineservice

import (
	"context"
	"fmt"
	"strings"

	"github.com/google/uuid"
)

// NoConfiguredStreamsError reports connections that cannot run because none
// of their streams is configured.
type NoConfiguredStreamsError struct {
	ConnectionNames []string
}

func (e *NoConfiguredStreamsError) Error() string {
	quoted := make([]string, len(e.ConnectionNames))
	for i, name := range e.ConnectionNames {
		quoted[i] = fmt.Sprintf("%q", name)
	}

	if len(quoted) == 1 {
		return fmt.Sprintf("connection %s has no configured streams; configure its streams before running it", quoted[0])
	}

	return fmt.Sprintf("connections %s have no configured streams; configure their streams before running them",
		strings.Join(quoted, ", "))
}

// ConnectionsWithoutStreams returns the connections among conns that have
// no configured stream, in the order given.
func ConnectionsWithoutStreams(ctx context.Context, storage Storage, conns []*Connection) ([]*Connection, error) {
	if len(conns) == 0 {
		return nil, nil
	}

	ids := make([]uuid.UUID, len(conns))
	for i, conn := range conns {
		ids[i] = conn.ID
	}

	configured, err := storage.Connections().WithConfiguredStreams(ctx, ids)
	if err != nil {
		return nil, fmt.Errorf("checking configured streams: %w", err)
	}

	var missing []*Connection

	for _, conn := range conns {
		if !configured[conn.ID] {
			missing = append(missing, conn)
		}
	}

	return missing, nil
}

// RequireConfiguredStreams returns a *NoConfiguredStreamsError naming each of
// conns that has no configured stream.
func RequireConfiguredStreams(ctx context.Context, storage Storage, conns ...*Connection) error {
	missing, err := ConnectionsWithoutStreams(ctx, storage, conns)
	if err != nil {
		return err
	}

	if len(missing) == 0 {
		return nil
	}

	names := make([]string, len(missing))
	for i, conn := range missing {
		names[i] = conn.Name
	}

	return &NoConfiguredStreamsError{ConnectionNames: names}
}

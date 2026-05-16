package workspacestorage

import (
	"fmt"

	"github.com/google/uuid"
	"github.com/saturn4er/boilerplate-go/lib/txoutbox"

	"github.com/synclet-io/synclet/modules/workspace/workspaceevent"
	workspacesvc "github.com/synclet-io/synclet/modules/workspace/workspaceservice"
)

// buildWorkspaceEventMessage is invoked by the generated WorkspaceEventsOutbox
// each time a use case sends a workspace event. It encodes the event into the
// JSON wire format defined in the workspaceevent package and wraps it in a
// txoutbox.Message tagged with the workspace topic.
func buildWorkspaceEventMessage(event *workspacesvc.WorkspaceEvent) (*txoutbox.Message, error) {
	metadata, body, err := workspaceevent.EncodeWorkspaceEvent(event)
	if err != nil {
		return nil, fmt.Errorf("encoding workspace event: %w", err)
	}

	orderingKey, err := workspaceevent.OrderingKey(event)
	if err != nil {
		return nil, fmt.Errorf("computing ordering key: %w", err)
	}

	return &txoutbox.Message{
		Topic:          workspaceevent.WorkspaceEventTopic,
		OrderingKey:    orderingKey,
		IdempotencyKey: uuid.NewString(),
		Data:           body,
		Metadata:       metadata,
	}, nil
}

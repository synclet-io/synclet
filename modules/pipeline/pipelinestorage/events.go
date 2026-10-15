package pipelinestorage

import (
	"fmt"

	"github.com/google/uuid"
	"github.com/saturn4er/boilerplate-go/lib/txoutbox"

	"github.com/synclet-io/synclet/modules/pipeline/pipelineevent"
	pipelinesvc "github.com/synclet-io/synclet/modules/pipeline/pipelineservice"
)

// buildPipelineEventMessage is invoked by the generated PipelineEventsOutbox
// each time a use case sends a pipeline event. It encodes the event into the
// JSON wire format defined in the pipelineevent package and wraps it in a
// txoutbox.Message tagged with the pipeline topic.
func buildPipelineEventMessage(event *pipelinesvc.PipelineEvent) (*txoutbox.Message, error) {
	metadata, body, err := pipelineevent.EncodePipelineEvent(event)
	if err != nil {
		return nil, fmt.Errorf("encoding pipeline event: %w", err)
	}

	orderingKey, err := pipelineevent.OrderingKey(event)
	if err != nil {
		return nil, fmt.Errorf("computing ordering key: %w", err)
	}

	return &txoutbox.Message{
		Topic:          pipelineevent.PipelineEventTopic,
		OrderingKey:    orderingKey,
		IdempotencyKey: uuid.NewString(),
		Data:           body,
		Metadata:       metadata,
	}, nil
}

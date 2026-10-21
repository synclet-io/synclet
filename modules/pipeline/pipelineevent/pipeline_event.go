// Package pipelineevent defines the JSON envelope for pipeline domain events
// that are published to subscribers via the transactional outbox. Producers
// (storage layer) call EncodePipelineEvent to build the txoutbox message;
// consumers call DecodePipelineEvent to reconstruct the in-process model.
package pipelineevent

import (
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/synclet-io/synclet/modules/pipeline/pipelineservice"
	"github.com/synclet-io/synclet/pkg/eventutil"
)

// PipelineEventTopic is the topic name written to txoutbox.Message.Topic for
// every pipeline event. Consumers subscribe by this name.
const PipelineEventTopic = "pipeline.pipeline_event"

// Wire-format strings. Kept distinct from the in-process enum so the wire
// format is stable across enum re-orderings.
const (
	pipelineEventTypeJobCreated          = "job.created"
	pipelineEventTypeJobUpdated          = "job.updated"
	pipelineEventTypeWorkflowRunStarted  = "workflow_run.started"
	pipelineEventTypeWorkflowRunFinished = "workflow_run.finished"
)

func eventTypeToString(eventType pipelineservice.PipelineEventType) (string, error) {
	switch eventType {
	case pipelineservice.PipelineEventTypeJobCreated:
		return pipelineEventTypeJobCreated, nil
	case pipelineservice.PipelineEventTypeJobUpdated:
		return pipelineEventTypeJobUpdated, nil
	case pipelineservice.PipelineEventTypeWorkflowRunStarted:
		return pipelineEventTypeWorkflowRunStarted, nil
	case pipelineservice.PipelineEventTypeWorkflowRunFinished:
		return pipelineEventTypeWorkflowRunFinished, nil
	default:
		return "", fmt.Errorf("unknown PipelineEventType: %d", eventType)
	}
}

func eventTypeFromString(value string) (pipelineservice.PipelineEventType, error) {
	switch value {
	case pipelineEventTypeJobCreated:
		return pipelineservice.PipelineEventTypeJobCreated, nil
	case pipelineEventTypeJobUpdated:
		return pipelineservice.PipelineEventTypeJobUpdated, nil
	case pipelineEventTypeWorkflowRunStarted:
		return pipelineservice.PipelineEventTypeWorkflowRunStarted, nil
	case pipelineEventTypeWorkflowRunFinished:
		return pipelineservice.PipelineEventTypeWorkflowRunFinished, nil
	default:
		return 0, fmt.Errorf("unknown PipelineEventType string: %q", value)
	}
}

// jobPayload is the wire format of a Job inside an event envelope.
type jobPayload struct {
	ID                uuid.UUID  `json:"id"`
	ConnectionID      uuid.UUID  `json:"connection_id"`
	JobType           string     `json:"job_type"`
	Status            string     `json:"status"`
	ScheduledAt       time.Time  `json:"scheduled_at"`
	StartedAt         *time.Time `json:"started_at,omitempty"`
	CompletedAt       *time.Time `json:"completed_at,omitempty"`
	Error             *string    `json:"error,omitempty"`
	Attempt           int        `json:"attempt"`
	MaxAttempts       int        `json:"max_attempts"`
	WorkerID          *string    `json:"worker_id,omitempty"`
	HeartbeatAt       *time.Time `json:"heartbeat_at,omitempty"`
	K8sJobName        *string    `json:"k8s_job_name,omitempty"`
	FailureReason     *string    `json:"failure_reason,omitempty"`
	WorkflowStepRunID *uuid.UUID `json:"workflow_step_run_id,omitempty"`
	CreatedAt         time.Time  `json:"created_at"`
}

func toJobPayload(job *pipelineservice.Job) jobPayload {
	return jobPayload{
		ID:                job.ID,
		ConnectionID:      job.ConnectionID,
		JobType:           job.JobType.String(),
		Status:            job.Status.String(),
		ScheduledAt:       job.ScheduledAt,
		StartedAt:         job.StartedAt,
		CompletedAt:       job.CompletedAt,
		Error:             job.Error,
		Attempt:           job.Attempt,
		MaxAttempts:       job.MaxAttempts,
		WorkerID:          job.WorkerID,
		HeartbeatAt:       job.HeartbeatAt,
		K8sJobName:        job.K8sJobName,
		FailureReason:     job.FailureReason,
		WorkflowStepRunID: job.WorkflowStepRunID,
		CreatedAt:         job.CreatedAt,
	}
}

func fromJobPayload(payload jobPayload) (*pipelineservice.Job, error) {
	jobType, err := jobTypeFromString(payload.JobType)
	if err != nil {
		return nil, fmt.Errorf("decoding job type: %w", err)
	}

	status, err := jobStatusFromString(payload.Status)
	if err != nil {
		return nil, fmt.Errorf("decoding job status: %w", err)
	}

	return &pipelineservice.Job{
		ID:                payload.ID,
		ConnectionID:      payload.ConnectionID,
		JobType:           jobType,
		Status:            status,
		ScheduledAt:       payload.ScheduledAt,
		StartedAt:         payload.StartedAt,
		CompletedAt:       payload.CompletedAt,
		Error:             payload.Error,
		Attempt:           payload.Attempt,
		MaxAttempts:       payload.MaxAttempts,
		WorkerID:          payload.WorkerID,
		HeartbeatAt:       payload.HeartbeatAt,
		K8sJobName:        payload.K8sJobName,
		FailureReason:     payload.FailureReason,
		WorkflowStepRunID: payload.WorkflowStepRunID,
		CreatedAt:         payload.CreatedAt,
	}, nil
}

type jobCreatedPayload struct {
	WorkspaceID uuid.UUID  `json:"workspace_id"`
	Job         jobPayload `json:"job"`
}

type jobUpdatedPayload struct {
	WorkspaceID uuid.UUID  `json:"workspace_id"`
	OldData     jobPayload `json:"old_data"`
	NewData     jobPayload `json:"new_data"`
}

type workflowRunStartedPayload struct {
	WorkspaceID uuid.UUID `json:"workspace_id"`
	WorkflowID  uuid.UUID `json:"workflow_id"`
	RunID       uuid.UUID `json:"run_id"`
}

type workflowRunFinishedPayload struct {
	WorkspaceID   uuid.UUID   `json:"workspace_id"`
	WorkflowID    uuid.UUID   `json:"workflow_id"`
	WorkflowName  string      `json:"workflow_name"`
	RunID         uuid.UUID   `json:"run_id"`
	Status        string      `json:"status"`
	Error         *string     `json:"error,omitempty"`
	ConnectionIDs []uuid.UUID `json:"connection_ids"`
}

// EncodePipelineEvent serializes a domain event into its (metadata, data)
// wire form. The returned metadata always includes
// eventutil.MetadataKeyEventType so consumers can dispatch without inspecting
// the body.
func EncodePipelineEvent(event *pipelineservice.PipelineEvent) (metadata map[string]string, body []byte, err error) {
	if event == nil {
		return nil, nil, errors.New("event is nil")
	}

	eventTypeStr, err := eventTypeToString(event.EventType)
	if err != nil {
		return nil, nil, fmt.Errorf("encoding event type: %w", err)
	}

	var payload any

	switch data := event.Data.(type) {
	case *pipelineservice.JobCreatedEventData:
		if data == nil || data.Job == nil {
			return nil, nil, errors.New("job.created event missing job")
		}

		payload = jobCreatedPayload{
			WorkspaceID: data.WorkspaceID,
			Job:         toJobPayload(data.Job),
		}
	case *pipelineservice.JobUpdatedEventData:
		if data == nil || data.OldData == nil || data.NewData == nil {
			return nil, nil, errors.New("job.updated event missing old/new job")
		}

		payload = jobUpdatedPayload{
			WorkspaceID: data.WorkspaceID,
			OldData:     toJobPayload(data.OldData),
			NewData:     toJobPayload(data.NewData),
		}
	case *pipelineservice.WorkflowRunStartedEventData:
		if data == nil {
			return nil, nil, errors.New("workflow_run.started event missing data")
		}

		payload = workflowRunStartedPayload{
			WorkspaceID: data.WorkspaceID,
			WorkflowID:  data.WorkflowID,
			RunID:       data.RunID,
		}
	case *pipelineservice.WorkflowRunFinishedEventData:
		if data == nil {
			return nil, nil, errors.New("workflow_run.finished event missing data")
		}

		payload = workflowRunFinishedPayload{
			WorkspaceID:   data.WorkspaceID,
			WorkflowID:    data.WorkflowID,
			WorkflowName:  data.WorkflowName,
			RunID:         data.RunID,
			Status:        data.Status.String(),
			Error:         data.Error,
			ConnectionIDs: data.ConnectionIDs,
		}
	default:
		return nil, nil, fmt.Errorf("unsupported PipelineEventData type: %T", event.Data)
	}

	body, err = json.Marshal(payload)
	if err != nil {
		return nil, nil, fmt.Errorf("marshalling event payload: %w", err)
	}

	metadata = map[string]string{
		eventutil.MetadataKeyEventType: eventTypeStr,
	}

	return metadata, body, nil
}

// DecodePipelineEvent rebuilds a domain event from its (metadata, data) wire
// form.
func DecodePipelineEvent(metadata map[string]string, body []byte) (*pipelineservice.PipelineEvent, error) {
	eventTypeStr, ok := metadata[eventutil.MetadataKeyEventType]
	if !ok {
		return nil, fmt.Errorf("metadata missing %s", eventutil.MetadataKeyEventType)
	}

	eventType, err := eventTypeFromString(eventTypeStr)
	if err != nil {
		return nil, fmt.Errorf("decoding event type: %w", err)
	}

	var data pipelineservice.PipelineEventData

	switch eventTypeStr {
	case pipelineEventTypeJobCreated:
		payload := jobCreatedPayload{}
		if err := json.Unmarshal(body, &payload); err != nil {
			return nil, fmt.Errorf("unmarshalling job.created payload: %w", err)
		}

		job, err := fromJobPayload(payload.Job)
		if err != nil {
			return nil, fmt.Errorf("decoding job: %w", err)
		}

		data = &pipelineservice.JobCreatedEventData{
			WorkspaceID: payload.WorkspaceID,
			Job:         job,
		}
	case pipelineEventTypeJobUpdated:
		payload := jobUpdatedPayload{}
		if err := json.Unmarshal(body, &payload); err != nil {
			return nil, fmt.Errorf("unmarshalling job.updated payload: %w", err)
		}

		oldJob, err := fromJobPayload(payload.OldData)
		if err != nil {
			return nil, fmt.Errorf("decoding old job: %w", err)
		}

		newJob, err := fromJobPayload(payload.NewData)
		if err != nil {
			return nil, fmt.Errorf("decoding new job: %w", err)
		}

		data = &pipelineservice.JobUpdatedEventData{
			WorkspaceID: payload.WorkspaceID,
			OldData:     oldJob,
			NewData:     newJob,
		}
	case pipelineEventTypeWorkflowRunStarted:
		payload := workflowRunStartedPayload{}
		if err := json.Unmarshal(body, &payload); err != nil {
			return nil, fmt.Errorf("unmarshalling workflow_run.started payload: %w", err)
		}

		data = &pipelineservice.WorkflowRunStartedEventData{
			WorkspaceID: payload.WorkspaceID,
			WorkflowID:  payload.WorkflowID,
			RunID:       payload.RunID,
		}
	case pipelineEventTypeWorkflowRunFinished:
		payload := workflowRunFinishedPayload{}
		if err := json.Unmarshal(body, &payload); err != nil {
			return nil, fmt.Errorf("unmarshalling workflow_run.finished payload: %w", err)
		}

		status, err := workflowRunStatusFromString(payload.Status)
		if err != nil {
			return nil, fmt.Errorf("decoding run status: %w", err)
		}

		data = &pipelineservice.WorkflowRunFinishedEventData{
			WorkspaceID:   payload.WorkspaceID,
			WorkflowID:    payload.WorkflowID,
			WorkflowName:  payload.WorkflowName,
			RunID:         payload.RunID,
			Status:        status,
			Error:         payload.Error,
			ConnectionIDs: payload.ConnectionIDs,
		}
	default:
		return nil, fmt.Errorf("unsupported event type: %s", eventTypeStr)
	}

	return &pipelineservice.PipelineEvent{
		EventType: eventType,
		Data:      data,
	}, nil
}

func workflowRunStatusFromString(value string) (pipelineservice.WorkflowRunStatus, error) {
	switch value {
	case pipelineservice.WorkflowRunStatusRunning.String():
		return pipelineservice.WorkflowRunStatusRunning, nil
	case pipelineservice.WorkflowRunStatusSucceeded.String():
		return pipelineservice.WorkflowRunStatusSucceeded, nil
	case pipelineservice.WorkflowRunStatusFailed.String():
		return pipelineservice.WorkflowRunStatusFailed, nil
	default:
		return 0, fmt.Errorf("unknown WorkflowRunStatus: %q", value)
	}
}

func jobTypeFromString(value string) (pipelineservice.JobType, error) {
	switch value {
	case pipelineservice.JobTypeSync.String():
		return pipelineservice.JobTypeSync, nil
	case pipelineservice.JobTypeDiscover.String():
		return pipelineservice.JobTypeDiscover, nil
	case pipelineservice.JobTypeCheck.String():
		return pipelineservice.JobTypeCheck, nil
	default:
		return 0, fmt.Errorf("unknown JobType: %q", value)
	}
}

func jobStatusFromString(value string) (pipelineservice.JobStatus, error) {
	switch value {
	case pipelineservice.JobStatusScheduled.String():
		return pipelineservice.JobStatusScheduled, nil
	case pipelineservice.JobStatusStarting.String():
		return pipelineservice.JobStatusStarting, nil
	case pipelineservice.JobStatusRunning.String():
		return pipelineservice.JobStatusRunning, nil
	case pipelineservice.JobStatusCompleted.String():
		return pipelineservice.JobStatusCompleted, nil
	case pipelineservice.JobStatusFailed.String():
		return pipelineservice.JobStatusFailed, nil
	case pipelineservice.JobStatusCancelled.String():
		return pipelineservice.JobStatusCancelled, nil
	default:
		return 0, fmt.Errorf("unknown JobStatus: %q", value)
	}
}

// OrderingKey returns the txoutbox ordering key of the event: the connection
// UUID for job events, the workflow UUID for workflow run events. Events with
// the same key stay in order.
func OrderingKey(event *pipelineservice.PipelineEvent) (string, error) {
	if event == nil {
		return "", errors.New("event is nil")
	}

	switch data := event.Data.(type) {
	case *pipelineservice.JobCreatedEventData:
		if data == nil || data.Job == nil {
			return "", errors.New("job.created event missing job")
		}

		return data.Job.ConnectionID.String(), nil
	case *pipelineservice.JobUpdatedEventData:
		if data == nil || data.NewData == nil {
			return "", errors.New("job.updated event missing new job")
		}

		return data.NewData.ConnectionID.String(), nil
	case *pipelineservice.WorkflowRunStartedEventData:
		if data == nil {
			return "", errors.New("workflow_run.started event missing data")
		}

		return data.WorkflowID.String(), nil
	case *pipelineservice.WorkflowRunFinishedEventData:
		if data == nil {
			return "", errors.New("workflow_run.finished event missing data")
		}

		return data.WorkflowID.String(), nil
	default:
		return "", fmt.Errorf("unsupported PipelineEventData type: %T", event.Data)
	}
}

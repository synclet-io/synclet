package pipelinesync

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/go-pnp/go-pnp/logging"

	"github.com/synclet-io/synclet/modules/pipeline/pipelineservice"
	"github.com/synclet-io/synclet/pkg/connector"
	"github.com/synclet-io/synclet/pkg/protocol"
)

// DockerConnectorTaskWorker polls for pending connector tasks and executes them
// using ConnectorClient (Docker). Follows the same semaphore pattern as DockerSyncWorker
// but without heartbeat (tasks are short-lived). Per D-16, uses a separate goroutine pool.
type DockerConnectorTaskWorker struct {
	backend   ExecutorBackend
	client    *connector.ConnectorClient
	manager   *SyncWorkerManager
	semaphore chan struct{}
	workerID  string
	logger    *logging.Logger
}

// DockerConnectorTaskWorkerParams holds all constructor dependencies.
type DockerConnectorTaskWorkerParams struct {
	Backend       ExecutorBackend
	Client        *connector.ConnectorClient
	Manager       *SyncWorkerManager
	MaxConcurrent int
	Logger        *logging.Logger
}

// NewDockerConnectorTaskWorker creates a new DockerConnectorTaskWorker with all dependencies.
func NewDockerConnectorTaskWorker(params DockerConnectorTaskWorkerParams) *DockerConnectorTaskWorker {
	workerID := getWorkerID()
	concurrency := params.MaxConcurrent
	if concurrency <= 0 {
		concurrency = 5
	}

	var logger *logging.Logger
	if params.Logger != nil {
		logger = params.Logger.Named("docker-task-worker")
	}

	return &DockerConnectorTaskWorker{
		backend:   params.Backend,
		client:    params.Client,
		manager:   params.Manager,
		semaphore: make(chan struct{}, concurrency),
		workerID:  workerID,
		logger:    logger,
	}
}

// Execute polls for and claims a pending connector task, then spawns a goroutine
// to execute it. Returns immediately so the jobber timer resets.
// Checks the concurrency semaphore BEFORE claiming to avoid claiming tasks
// that cannot be executed.
func (w *DockerConnectorTaskWorker) Execute(ctx context.Context) error {
	// Check concurrency limit before claiming.
	select {
	case w.semaphore <- struct{}{}:
	default:
		return nil
	}

	result, err := w.backend.ClaimConnectorTask(ctx, w.workerID)
	if err != nil {
		<-w.semaphore
		return err
	}

	if result == nil {
		<-w.semaphore
		return nil
	}

	if w.logger != nil {
		w.logger.WithFields(map[string]interface{}{"worker_id": w.workerID, "task_id": result.TaskID.String(), "task_type": result.TaskType}).Info(ctx, "claimed connector task")
	}

	w.manager.RunJob(func(taskCtx context.Context) {
		defer func() { <-w.semaphore }()
		w.executeTask(taskCtx, result)
	})

	return nil
}

// executeTask runs the appropriate connector operation and reports the result.
func (w *DockerConnectorTaskWorker) executeTask(ctx context.Context, task *ClaimConnectorTaskResult) {
	var resultBytes []byte
	var taskErr error

	switch task.TaskType {
	case pipelineservice.ConnectorTaskTypeCheck:
		taskErr = w.executeCheck(ctx, task, &resultBytes)
	case pipelineservice.ConnectorTaskTypeSpec:
		taskErr = w.executeSpec(ctx, task, &resultBytes)
	case pipelineservice.ConnectorTaskTypeDiscover:
		taskErr = w.executeDiscover(ctx, task, &resultBytes)
	default:
		taskErr = fmt.Errorf("unknown task type: %s", task.TaskType)
	}

	// Report result via backend.
	params := ReportConnectorTaskResultParams{
		TaskID:  task.TaskID,
		Success: taskErr == nil,
		Result:  resultBytes,
	}
	if taskErr != nil {
		params.ErrorMessage = taskErr.Error()
		if w.logger != nil {
			w.logger.WithError(taskErr).WithFields(map[string]interface{}{"task_id": task.TaskID.String(), "task_type": task.TaskType}).Error(ctx, "connector task failed")
		}
	}

	if err := w.backend.ReportConnectorTaskResult(ctx, params); err != nil {
		if w.logger != nil {
			w.logger.WithError(err).WithField("task_id", task.TaskID.String()).Error(ctx, "failed to report connector task result")
		}
	}

	if w.logger != nil && taskErr == nil {
		w.logger.WithFields(map[string]interface{}{"task_id": task.TaskID.String(), "task_type": task.TaskType}).Info(ctx, "connector task completed")
	}
}

// executeCheck runs a connection check and marshals the result.
func (w *DockerConnectorTaskWorker) executeCheck(ctx context.Context, task *ClaimConnectorTaskResult, resultBytes *[]byte) error {
	status, err := w.client.Check(ctx, task.Image, task.Config)
	if err != nil {
		return fmt.Errorf("check: %w", err)
	}

	result := &pipelineservice.CheckResult{
		Success: status.Status == protocol.ConnectionStatusSucceeded,
		Message: status.Message,
	}

	data, err := json.Marshal(result)
	if err != nil {
		return fmt.Errorf("marshaling check result: %w", err)
	}

	*resultBytes = data
	return nil
}

// executeSpec runs a connector spec and marshals the result.
func (w *DockerConnectorTaskWorker) executeSpec(ctx context.Context, task *ClaimConnectorTaskResult, resultBytes *[]byte) error {
	spec, err := w.client.Spec(ctx, task.Image)
	if err != nil {
		return fmt.Errorf("spec: %w", err)
	}

	result := &pipelineservice.SpecResult{
		DocumentationURL:        spec.DocumentationURL,
		ChangelogURL:            spec.ChangelogURL,
		ConnectionSpecification: string(spec.ConnectionSpecification),
		SupportsIncremental:     spec.SupportsIncremental,
		SupportsNormalization:   spec.SupportsNormalization,
		SupportsDBT:             spec.SupportsDBT,
		ProtocolVersion:         spec.ProtocolVersion,
	}

	// SupportedDestinationSyncModes: marshal []DestinationSyncMode to JSON string for jsonb field
	if len(spec.SupportedDestinationSyncModes) > 0 {
		modesJSON, err := json.Marshal(spec.SupportedDestinationSyncModes)
		if err != nil {
			return fmt.Errorf("marshaling supported_destination_sync_modes: %w", err)
		}
		result.SupportedDestinationSyncModes = string(modesJSON)
	}

	// AdvancedAuth: marshal *AdvancedAuth to JSON string for jsonb field
	if spec.AdvancedAuth != nil {
		authJSON, err := json.Marshal(spec.AdvancedAuth)
		if err != nil {
			return fmt.Errorf("marshaling advanced_auth: %w", err)
		}
		result.AdvancedAuth = string(authJSON)
	}

	data, err := json.Marshal(result)
	if err != nil {
		return fmt.Errorf("marshaling spec result: %w", err)
	}

	*resultBytes = data
	return nil
}

// executeDiscover runs catalog discovery and marshals the result.
func (w *DockerConnectorTaskWorker) executeDiscover(ctx context.Context, task *ClaimConnectorTaskResult, resultBytes *[]byte) error {
	catalog, err := w.client.Discover(ctx, task.Image, task.Config)
	if err != nil {
		return fmt.Errorf("discover: %w", err)
	}

	catalogJSON, err := json.Marshal(catalog)
	if err != nil {
		return fmt.Errorf("marshaling catalog: %w", err)
	}

	result := &pipelineservice.DiscoverResult{
		Catalog: string(catalogJSON),
	}

	data, err := json.Marshal(result)
	if err != nil {
		return fmt.Errorf("marshaling discover result: %w", err)
	}

	*resultBytes = data
	return nil
}

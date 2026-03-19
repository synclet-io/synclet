package coordinator

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"sync"
	"time"

	"connectrpc.com/connect"

	executorv1 "github.com/synclet-io/synclet/gen/proto/synclet/internalapi/executor/v1"
	"github.com/synclet-io/synclet/gen/proto/synclet/internalapi/executor/v1/executorv1connect"
	protocolv1 "github.com/synclet-io/synclet/gen/proto/synclet/protocol/v1"
	"github.com/synclet-io/synclet/pkg/protocol"
)

// Reporter sends metadata (heartbeats, state checkpoints, completion) to the Synclet server.
// It buffers notifications and retries with exponential backoff on failure.
type Reporter struct {
	client       executorv1connect.ExecutorServiceClient
	jobID        string
	connectionID string

	stateCh        chan *executorv1.ReportStateRequest
	completionCh   chan *executorv1.ReportCompletionRequest
	configUpdateCh chan *executorv1.ReportConfigUpdateRequest
	logCh          chan string

	wg   sync.WaitGroup
	done chan struct{}
}

// NewReporter creates a reporter that sends notifications to the given server address.
func NewReporter(serverAddr, jobID, connectionID string) *Reporter {
	client := executorv1connect.NewExecutorServiceClient(
		&http.Client{Timeout: 30 * time.Second},
		serverAddr,
	)

	return &Reporter{
		client:         client,
		jobID:          jobID,
		connectionID:   connectionID,
		stateCh:        make(chan *executorv1.ReportStateRequest, 100),
		completionCh:   make(chan *executorv1.ReportCompletionRequest, 1),
		configUpdateCh: make(chan *executorv1.ReportConfigUpdateRequest, 10),
		logCh:          make(chan string, 500),
		done:           make(chan struct{}),
	}
}

// Start begins the background goroutines for draining state and completion buffers.
func (r *Reporter) Start(ctx context.Context) {
	r.wg.Add(1)
	go func() {
		defer r.wg.Done()
		r.drainStates(ctx)
	}()

	r.wg.Add(1)
	go func() {
		defer r.wg.Done()
		r.drainCompletions(ctx)
	}()

	r.wg.Add(1)
	go func() {
		defer r.wg.Done()
		r.drainConfigUpdates(ctx)
	}()

	r.wg.Add(1)
	go func() {
		defer r.wg.Done()
		r.drainLogs(ctx)
	}()
}

// Stop waits for all pending notifications to be sent, with a 30s timeout.
func (r *Reporter) Stop() {
	close(r.done)

	ch := make(chan struct{})
	go func() {
		r.wg.Wait()
		close(ch)
	}()

	select {
	case <-ch:
	case <-time.After(30 * time.Second):
		slog.Warn("reporter: timed out waiting for flush")
	}
}

// SendHeartbeat sends a heartbeat to the server. Fire-and-forget: drops on failure.
func (r *Reporter) SendHeartbeat(ctx context.Context, recordsRead, bytesSynced int64) {
	req := connect.NewRequest(&executorv1.HeartbeatRequest{
		JobId:       r.jobID,
		RecordsRead: recordsRead,
		BytesSynced: bytesSynced,
	})

	if _, err := r.client.Heartbeat(ctx, req); err != nil {
		slog.Error("reporter: heartbeat failed", "error", err)
	}
}

// QueueState queues a state checkpoint for reliable delivery.
// The msg is serialized to JSON and sent as the state_data blob.
func (r *Reporter) QueueState(msg *protocol.AirbyteStateMessage) {
	stateData, err := json.Marshal(msg)
	if err != nil {
		slog.Error("reporter: failed to marshal state message", "error", err)
		return
	}

	req := &executorv1.ReportStateRequest{
		JobId:        r.jobID,
		ConnectionId: r.connectionID,
		StateData:    stateData,
		StateType:    airbyteStateTypeToProto(msg.Type),
	}

	select {
	case r.stateCh <- req:
	default:
		slog.Warn("reporter: state buffer full, dropping state")
	}
}

// QueueCompletion queues a completion report for reliable delivery.
func (r *Reporter) QueueCompletion(req *executorv1.ReportCompletionRequest) {
	req.JobId = r.jobID
	req.ConnectionId = r.connectionID

	select {
	case r.completionCh <- req:
	default:
		slog.Warn("reporter: completion buffer full")
	}
}

// QueueLog queues a log line for delivery. Non-blocking: drops on full buffer.
func (r *Reporter) QueueLog(line string) {
	select {
	case r.logCh <- line:
	default:
		slog.Warn("reporter: log buffer full, dropping log line")
	}
}

// QueueConfigUpdate queues a config update report for reliable delivery.
func (r *Reporter) QueueConfigUpdate(connectorType executorv1.ConnectorType, connectorID string, config []byte) {
	req := &executorv1.ReportConfigUpdateRequest{
		JobId:         r.jobID,
		ConnectionId:  r.connectionID,
		ConnectorType: connectorType,
		ConnectorId:   connectorID,
		Config:        config,
	}

	select {
	case r.configUpdateCh <- req:
	default:
		slog.Warn("reporter: config update buffer full, dropping update", "connector_type", connectorType, "connector_id", connectorID)
	}
}

func (r *Reporter) drainStates(ctx context.Context) {
	for {
		select {
		case req, ok := <-r.stateCh:
			if !ok {
				return
			}
			r.sendWithRetry(ctx, "state", 10, func(ctx context.Context) error {
				_, err := r.client.ReportState(ctx, connect.NewRequest(req))
				return err
			})
		case <-r.done:
			// Drain remaining states.
			for {
				select {
				case req := <-r.stateCh:
					flushCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
					r.sendWithRetry(flushCtx, "state", 10, func(ctx context.Context) error {
						_, err := r.client.ReportState(ctx, connect.NewRequest(req))
						return err
					})
					cancel()
				default:
					return
				}
			}
		}
	}
}

func (r *Reporter) drainCompletions(ctx context.Context) {
	for {
		select {
		case req, ok := <-r.completionCh:
			if !ok {
				return
			}
			r.sendWithRetry(ctx, "completion", 10, func(ctx context.Context) error {
				_, err := r.client.ReportCompletion(ctx, connect.NewRequest(req))
				return err
			})
		case <-r.done:
			// Drain remaining completions.
			for {
				select {
				case req := <-r.completionCh:
					flushCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
					r.sendWithRetry(flushCtx, "completion", 10, func(ctx context.Context) error {
						_, err := r.client.ReportCompletion(ctx, connect.NewRequest(req))
						return err
					})
					cancel()
				default:
					return
				}
			}
		}
	}
}

func (r *Reporter) drainConfigUpdates(ctx context.Context) {
	for {
		select {
		case req, ok := <-r.configUpdateCh:
			if !ok {
				return
			}
			r.sendWithRetry(ctx, "config_update", 10, func(ctx context.Context) error {
				_, err := r.client.ReportConfigUpdate(ctx, connect.NewRequest(req))
				return err
			})
		case <-r.done:
			// Drain remaining config updates.
			for {
				select {
				case req := <-r.configUpdateCh:
					flushCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
					r.sendWithRetry(flushCtx, "config_update", 10, func(ctx context.Context) error {
						_, err := r.client.ReportConfigUpdate(ctx, connect.NewRequest(req))
						return err
					})
					cancel()
				default:
					return
				}
			}
		}
	}
}

func (r *Reporter) drainLogs(ctx context.Context) {
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()
	var batch []string
	flush := func() {
		if len(batch) == 0 {
			return
		}
		_, err := r.client.ReportLog(ctx, connect.NewRequest(&executorv1.ReportLogRequest{
			JobId:    r.jobID,
			LogLines: batch,
		}))
		if err != nil {
			slog.Error("reporter: failed to report logs", "error", err)
		}
		batch = batch[:0]
	}
	for {
		select {
		case line := <-r.logCh:
			batch = append(batch, line)
			if len(batch) >= 50 {
				flush()
			}
		case <-ticker.C:
			flush()
		case <-r.done:
			// Drain remaining log lines.
			for {
				select {
				case line := <-r.logCh:
					batch = append(batch, line)
				default:
					// Use a background context for the final flush since the outer ctx may be cancelled.
					if len(batch) > 0 {
						flushCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
						_, err := r.client.ReportLog(flushCtx, connect.NewRequest(&executorv1.ReportLogRequest{
							JobId:    r.jobID,
							LogLines: batch,
						}))
						if err != nil {
							slog.Error("reporter: failed to report logs during drain", "error", err)
						}
						cancel()
					}
					return
				}
			}
		}
	}
}

func (r *Reporter) sendWithRetry(ctx context.Context, name string, maxRetries int, fn func(ctx context.Context) error) {
	backoff := time.Second
	maxBackoff := 30 * time.Second

	for attempt := 1; ; attempt++ {
		err := fn(ctx)
		if err == nil {
			return
		}

		if attempt >= maxRetries {
			slog.Error("reporter: send failed after max retries, giving up", "type", name, "error", err, "attempts", attempt)
			return
		}

		slog.Error("reporter: send failed, retrying", "type", name, "error", err, "backoff", backoff, "attempt", attempt)

		select {
		case <-ctx.Done():
			slog.Warn("reporter: send cancelled", "type", name)
			return
		case <-time.After(backoff):
		}

		backoff *= 2
		if backoff > maxBackoff {
			backoff = maxBackoff
		}
	}
}

func airbyteStateTypeToProto(t protocol.AirbyteStateType) protocolv1.StateType {
	switch t {
	case protocol.StateTypeStream:
		return protocolv1.StateType_STATE_TYPE_STREAM
	case protocol.StateTypeGlobal:
		return protocolv1.StateType_STATE_TYPE_GLOBAL
	case protocol.StateTypeLegacy:
		return protocolv1.StateType_STATE_TYPE_LEGACY
	default:
		return protocolv1.StateType_STATE_TYPE_STREAM
	}
}

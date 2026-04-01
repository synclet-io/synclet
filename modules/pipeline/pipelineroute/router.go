package pipelineroute

import (
	"context"
	"errors"
	"fmt"
	"io"
	"sync"
	"time"

	"github.com/go-pnp/go-pnp/logging"
	"github.com/synclet-io/synclet/pkg/protocol"
)

// RunConfig holds optional configuration for the routing session.
type RunConfig struct {
	IdleTimeout time.Duration      // Kill if no RECORD for this duration. 0 = disabled.
	Rewriter    *NamespaceRewriter // Rewrite namespace/prefix on RECORD and STATE messages. nil = no rewriting.
}

// Stats holds the final statistics from a routing session.
type Stats struct {
	RecordsRead int64
	BytesSynced int64
	Duration    time.Duration
}

// internalStats tracks counters during routing. Source loop writes RecordsRead
// and BytesSynced. No concurrent writes to the same field, so plain int64 is safe.
type internalStats struct {
	startTime   time.Time
	recordsRead int64
	bytesSynced int64
}

func (s *internalStats) export() *Stats {
	return &Stats{
		RecordsRead: s.recordsRead,
		BytesSynced: s.bytesSynced,
		Duration:    time.Since(s.startTime),
	}
}

// Run reads messages from sourceStdout, routes them to destStdin,
// reads dest output from destStdout, and dispatches side effects to handler.
// It blocks until both source and dest readers reach EOF or ctx is cancelled.
func Run(ctx context.Context, sourceStdout io.Reader, destStdin io.WriteCloser, destStdout io.ReadCloser, handler Handler, cfg RunConfig, logger *logging.Logger) (*Stats, error) {
	stats := &internalStats{startTime: time.Now()}

	srcReader := protocol.NewMessageReader(sourceStdout)
	destWriter := protocol.NewMessageWriter(destStdin)
	destReader := protocol.NewMessageReader(destStdout)

	// Read dest output in background.
	var destErr error
	var wg sync.WaitGroup
	wg.Add(1)

	go func() {
		defer wg.Done()

		destErr = readDestOutput(ctx, destReader, handler, stats, logger)
	}()

	// Read source output, route messages to dest.
	srcErr := routeSourceMessages(ctx, srcReader, destWriter, handler, stats, cfg, logger)

	// Signal EOF to destination.
	_ = destStdin.Close()

	// When source fails, close destStdout to unblock the dest reader goroutine.
	// Without this, the dest reader stays blocked on Read() because the destination
	// container is still running (it received stdin EOF but may take minutes to exit).
	// The defer destCleanup() in the caller cannot run until Run() returns, creating
	// a deadlock-like wait that only resolves on context timeout.
	if srcErr != nil {
		_ = destStdout.Close()
	}

	// Wait for dest output reader to finish.
	wg.Wait()

	// Prefer destination error over source error — a destination crash is the
	// root cause, while the source error is usually a secondary pipe failure.
	if destErr != nil {
		return stats.export(), destErr
	}

	return stats.export(), srcErr
}

// routeSourceMessages reads messages from the source and routes them according
// to the routing table defined in D-06.
func routeSourceMessages(ctx context.Context, reader *protocol.MessageReader, writer *protocol.MessageWriter, handler Handler, stats *internalStats, cfg RunConfig, logger *logging.Logger) error {
	var lastRecordTime time.Time
	firstRecordSeen := false

	for {
		msg, err := reader.Read()
		if err != nil {
			if errors.Is(err, io.EOF) {
				return nil
			}

			return err
		}

		// Check for context cancellation between reads.
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		// Check idle timeout: only after first RECORD to avoid killing slow initial loads.
		if cfg.IdleTimeout > 0 && firstRecordSeen && time.Since(lastRecordTime) > cfg.IdleTimeout {
			return fmt.Errorf("idle timeout: no RECORD messages for %s", cfg.IdleTimeout)
		}

		switch msg.Type {
		case protocol.MessageTypeRecord:
			if !firstRecordSeen {
				firstRecordSeen = true
			}

			lastRecordTime = time.Now()

			stats.recordsRead++
			if msg.Record != nil {
				stats.bytesSynced += int64(len(msg.Record.Data))
			}

			if cfg.Rewriter != nil {
				cfg.Rewriter.RewriteRecord(msg)
			}

			if err := writer.Write(msg); err != nil {
				return err
			}

		case protocol.MessageTypeState:
			// Rewrite stream descriptors before forwarding to destination.
			if cfg.Rewriter != nil {
				cfg.Rewriter.RewriteState(msg)
			}

			if err := writer.Write(msg); err != nil {
				return err
			}

		case protocol.MessageTypeLog:
			// Log messages stay local, not forwarded to destination.
			if msg.Log != nil {
				logger.WithFields(map[string]interface{}{"level": msg.Log.Level, "message": msg.Log.Message}).Info(ctx, "source log")

				if err := handler.OnLog(ctx, formatLogLine("[src]", msg.Log)); err != nil {
					logger.WithError(err).Error(ctx, "failed to handle log")
				}
			}

		case protocol.MessageTypeControl:
			// Dispatch to handler, do NOT forward to destination.
			if msg.Control != nil {
				if err := handler.OnSourceControl(ctx, msg.Control); err != nil {
					logger.WithError(err).Error(ctx, "handler OnSourceControl error")
				}
			}

		case protocol.MessageTypeTrace:
			if msg.Trace != nil {
				switch msg.Trace.Type {
				case protocol.TraceTypeAnalytics:
					// Analytics traces are logged but not forwarded.
					if msg.Trace.Analytics != nil {
						logger.WithFields(map[string]interface{}{"type": msg.Trace.Analytics.Type, "value": msg.Trace.Analytics.Value}).Info(ctx, "source analytics trace")
					}

					if err := handler.OnLog(ctx, formatTraceLine("[src]", msg.Trace)); err != nil {
						logger.WithError(err).Error(ctx, "failed to handle trace log")
					}
				case protocol.TraceTypeError:
					// Error traces are dispatched to handler AND forwarded to dest.
					if err := handler.OnSourceTrace(ctx, msg.Trace); err != nil {
						logger.WithError(err).Error(ctx, "handler OnSourceTrace error")
					}

					if err := handler.OnLog(ctx, formatTraceLine("[src]", msg.Trace)); err != nil {
						logger.WithError(err).Error(ctx, "failed to handle trace log")
					}

					if cfg.Rewriter != nil {
						cfg.Rewriter.RewriteTrace(msg)
					}

					if err := writer.Write(msg); err != nil {
						return err
					}
				default:
					// Other traces (estimate, stream_status) are forwarded to dest.
					if cfg.Rewriter != nil {
						cfg.Rewriter.RewriteTrace(msg)
					}

					if err := writer.Write(msg); err != nil {
						return err
					}
				}
			}

		default:
			// Unknown message types are forwarded to destination.
			if err := writer.Write(msg); err != nil {
				return err
			}
		}
	}
}

// readDestOutput reads messages from the destination's stdout and dispatches
// side effects to the handler.
func readDestOutput(ctx context.Context, reader *protocol.MessageReader, handler Handler, _ *internalStats, logger *logging.Logger) error {
	for {
		msg, err := reader.Read()
		if err != nil {
			if errors.Is(err, io.EOF) || errors.Is(err, io.ErrClosedPipe) {
				return nil
			}

			return err
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		switch msg.Type {
		case protocol.MessageTypeState:
			if msg.State != nil {
				if err := handler.OnStateConfirmed(ctx, msg.State); err != nil {
					logger.WithError(err).Error(ctx, "handler OnStateConfirmed error")
				}
			}

		case protocol.MessageTypeRecord:
			// Destination RECORD messages are ignored (destinations don't reliably report this).

		case protocol.MessageTypeLog:
			if msg.Log != nil {
				logger.WithFields(map[string]interface{}{"level": msg.Log.Level, "message": msg.Log.Message}).Info(ctx, "dest log")

				if err := handler.OnLog(ctx, formatLogLine("[dst]", msg.Log)); err != nil {
					logger.WithError(err).Error(ctx, "failed to handle log")
				}
			}

		case protocol.MessageTypeControl:
			if msg.Control != nil {
				if err := handler.OnDestControl(ctx, msg.Control); err != nil {
					logger.WithError(err).Error(ctx, "handler OnDestControl error")
				}
			}

		case protocol.MessageTypeTrace:
			if msg.Trace != nil {
				if msg.Trace.Type == protocol.TraceTypeAnalytics && msg.Trace.Analytics != nil {
					logger.WithFields(map[string]interface{}{"type": msg.Trace.Analytics.Type, "value": msg.Trace.Analytics.Value}).Info(ctx, "dest analytics trace")
				}

				if msg.Trace.Type == protocol.TraceTypeAnalytics || msg.Trace.Type == protocol.TraceTypeError {
					if err := handler.OnLog(ctx, formatTraceLine("[dst]", msg.Trace)); err != nil {
						logger.WithError(err).Error(ctx, "failed to handle trace log")
					}
				}
			}
		}
	}
}

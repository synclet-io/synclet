package pipelineexec

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
	Rewriter    *NamespaceRewriter // Rewrite namespace/prefix on messages to the destination and restore source names on confirmed STATE. nil = no rewriting.
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

		destErr = readDestOutput(ctx, destReader, handler, cfg.Rewriter, logger)
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

// sourceRead is one result of reading the source stream.
type sourceRead struct {
	msg *protocol.AirbyteMessage
	err error
}

// readSource reads the source stream on its own goroutine so the router can
// stop waiting on a silent source. The goroutine exits after delivering EOF or
// an error, or once done is closed; a Read blocked at that point returns when
// the caller tears down the source.
func readSource(reader *protocol.MessageReader, done <-chan struct{}) <-chan sourceRead {
	reads := make(chan sourceRead)

	go func() {
		for {
			msg, err := reader.Read()

			select {
			case reads <- sourceRead{msg: msg, err: err}:
			case <-done:
				return
			}

			if err != nil {
				return
			}
		}
	}()

	return reads
}

// routeSourceMessages reads messages from the source and routes them according
// to the routing table defined in D-06. The idle timeout is armed by the first
// RECORD, so slow initial loads are not killed, and fires even while the
// source produces no output at all.
func routeSourceMessages(ctx context.Context, reader *protocol.MessageReader, writer *protocol.MessageWriter, handler Handler, stats *internalStats, cfg RunConfig, logger *logging.Logger) error {
	done := make(chan struct{})
	defer close(done)

	reads := readSource(reader, done)

	idleTimer := time.NewTimer(cfg.IdleTimeout)
	idleTimer.Stop()

	defer idleTimer.Stop()

	var idle <-chan time.Time

	counter := newSourceStatsCounter()

	for {
		var read sourceRead

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-idle:
			return fmt.Errorf("idle timeout: no RECORD messages for %s", cfg.IdleTimeout)
		case read = <-reads:
		}

		if read.err != nil {
			if errors.Is(read.err, io.EOF) {
				return nil
			}

			return read.err
		}

		msg := read.msg

		switch msg.Type {
		case protocol.MessageTypeRecord:
			if cfg.IdleTimeout > 0 {
				idleTimer.Reset(cfg.IdleTimeout)
				idle = idleTimer.C
			}

			stats.recordsRead++
			if msg.Record != nil {
				stats.bytesSynced += int64(len(msg.Record.Data))
			}

			counter.countRecord(msg.Record)

			if cfg.Rewriter != nil {
				cfg.Rewriter.RewriteRecord(msg)
			}

			if err := writer.Write(msg); err != nil {
				return err
			}

		case protocol.MessageTypeState:
			counter.fillState(msg.State)

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
				case protocol.TraceTypeEstimate:
					// Estimate traces are logged and not forwarded to the destination.
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
					// Stream status traces and trace types not handled above are forwarded to dest.
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
// side effects to the handler. Confirmed STATE carries the source catalog's
// stream names when rewriter is set.
func readDestOutput(ctx context.Context, reader *protocol.MessageReader, handler Handler, rewriter *NamespaceRewriter, logger *logging.Logger) error {
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
				if rewriter != nil {
					rewriter.RestoreState(msg.State)
				}

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

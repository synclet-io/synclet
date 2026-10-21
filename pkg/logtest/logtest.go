// Package logtest records everything written to a logging.Logger so tests can
// assert on the process log output.
package logtest

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"sync"

	"github.com/go-pnp/go-pnp/logging"
)

// Recorder collects log entries from every logger derived from Logger.
type Recorder struct {
	mu      sync.Mutex
	entries []string
}

// Logger returns a logger writing to r.
func (r *Recorder) Logger() *logging.Logger {
	return &logging.Logger{Delegate: &delegate{recorder: r}}
}

// String returns every recorded entry, one per line, with its fields, error
// and arguments.
func (r *Recorder) String() string {
	r.mu.Lock()
	defer r.mu.Unlock()

	return strings.Join(r.entries, "\n")
}

type delegate struct {
	recorder *Recorder
	fields   []string
}

func (d *delegate) record(level, msg string, args []any) {
	entry := fmt.Sprintf("%s %s %s %v", level, msg, strings.Join(d.fields, " "), args)

	d.recorder.mu.Lock()
	d.recorder.entries = append(d.recorder.entries, entry)
	d.recorder.mu.Unlock()
}

func (d *delegate) with(fields ...string) *delegate {
	return &delegate{recorder: d.recorder, fields: append(append([]string(nil), d.fields...), fields...)}
}

func (d *delegate) Info(_ context.Context, msg string, args ...any)  { d.record("INFO", msg, args) }
func (d *delegate) Warn(_ context.Context, msg string, args ...any)  { d.record("WARN", msg, args) }
func (d *delegate) Debug(_ context.Context, msg string, args ...any) { d.record("DEBUG", msg, args) }
func (d *delegate) Error(_ context.Context, msg string, args ...any) { d.record("ERROR", msg, args) }

func (d *delegate) WithFields(fields map[string]any) logging.Delegate {
	pairs := make([]string, 0, len(fields))
	for key, value := range fields {
		pairs = append(pairs, fmt.Sprintf("%s=%v", key, value))
	}

	sort.Strings(pairs)

	return d.with(pairs...)
}

func (d *delegate) WithField(key string, value any) logging.Delegate {
	return d.with(fmt.Sprintf("%s=%v", key, value))
}

func (d *delegate) WithError(err error) logging.Delegate {
	return d.with(fmt.Sprintf("error=%v", err))
}

func (d *delegate) Named(component string) logging.Delegate {
	return d.with("logger=" + component)
}

func (d *delegate) SkipCallers(int) logging.Delegate { return d }

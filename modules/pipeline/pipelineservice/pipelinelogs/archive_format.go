package pipelinelogs

import (
	"bufio"
	"compress/gzip"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"time"

	"go.uber.org/multierr"

	"github.com/synclet-io/synclet/modules/pipeline/pipelineservice"
)

// archivedLine is one NDJSON record of an archived attempt. ID is the line's
// job_logs id, which keeps GetJobLogs cursors valid across archiving.
type archivedLine struct {
	ID        int64     `json:"id"`
	Timestamp time.Time `json:"timestamp"`
	Message   string    `json:"message"`
}

// archiveWriter gzips job log lines as NDJSON.
type archiveWriter struct {
	gz  *gzip.Writer
	enc *json.Encoder
}

func newArchiveWriter(w io.Writer) *archiveWriter {
	gz := gzip.NewWriter(w)

	enc := json.NewEncoder(gz)
	enc.SetEscapeHTML(false)

	return &archiveWriter{gz: gz, enc: enc}
}

func (w *archiveWriter) Write(line pipelineservice.JobLog) error {
	return w.enc.Encode(archivedLine{ID: line.ID, Timestamp: line.CreatedAt, Message: line.LogLine})
}

func (w *archiveWriter) Close() error {
	return w.gz.Close()
}

// readArchive returns up to limit lines with id > afterID from a gzipped NDJSON archive.
func readArchive(r io.Reader, afterID int64, limit int) (_ []archivedLine, rerr error) {
	unzipped, err := gzip.NewReader(r)
	if err != nil {
		return nil, fmt.Errorf("opening gzip stream: %w", err)
	}
	defer multierr.AppendInvoke(&rerr, multierr.Close(unzipped))

	var lines []archivedLine

	dec := json.NewDecoder(bufio.NewReader(unzipped))

	for len(lines) < limit {
		var line archivedLine
		if err := dec.Decode(&line); errors.Is(err, io.EOF) {
			break
		} else if err != nil {
			return nil, fmt.Errorf("decoding archived log line: %w", err)
		}

		if line.ID > afterID {
			lines = append(lines, line)
		}
	}

	return lines, nil
}

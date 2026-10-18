package pipelineexec

import (
	"context"
	"fmt"
	"io"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/synclet-io/synclet/modules/pipeline/pipelineservice"
	"github.com/synclet-io/synclet/pkg/protocol"
)

// statsDestination echoes every STATE it reads and records its sourceStats,
// with -1 standing for a STATE that carries none.
func statsDestination(stdin io.Reader, stdout *io.PipeWriter, counts *[]float64) {
	reader := protocol.NewMessageReader(stdin)
	writer := protocol.NewMessageWriter(stdout)

	defer func() { _ = stdout.Close() }()

	for {
		msg, err := reader.Read()
		if err != nil {
			return
		}

		if msg.Type != protocol.MessageTypeState {
			continue
		}

		count := float64(-1)
		if msg.State.SourceStats != nil {
			count = msg.State.SourceStats.RecordCount
		}

		*counts = append(*counts, count)

		if err := writer.Write(msg); err != nil {
			return
		}
	}
}

func statsRecords(n int, name, namespace string) []string {
	lines := make([]string, n)
	for i := range lines {
		lines[i] = fmt.Sprintf(`{"type":"RECORD","record":{"stream":%q,"namespace":%q,"data":{"id":1},"emitted_at":1}}`, name, namespace)
	}

	return lines
}

func statsStreamState(name string) []string {
	return []string{fmt.Sprintf(`{"type":"STATE","state":{"type":"STREAM","stream":{"stream_descriptor":{"name":%q,"namespace":"x"},"stream_state":{"cursor":1}}}}`, name)}
}

func statsStreamStateWithCount(name string, count int) []string {
	return []string{fmt.Sprintf(`{"type":"STATE","state":{"type":"STREAM","stream":{"stream_descriptor":{"name":%q,"namespace":"x"},"stream_state":{"cursor":1}},"sourceStats":{"recordCount":%d}}}`, name, count)}
}

var (
	statsGlobalState = []string{`{"type":"STATE","state":{"type":"GLOBAL","global":{"shared_state":{"lsn":"0/1"},"stream_states":[{"stream_descriptor":{"name":"a","namespace":"x"},"stream_state":{"cursor":1}},{"stream_descriptor":{"name":"b","namespace":"x"},"stream_state":{"cursor":1}}]}}}`}
	statsLegacyState = []string{`{"type":"STATE","state":{"type":"LEGACY","data":{"cursor":1}}}`}
)

func statsSource(groups ...[]string) string {
	var lines []string
	for _, g := range groups {
		lines = append(lines, g...)
	}

	return strings.Join(lines, "\n") + "\n"
}

func runSourceStats(t *testing.T, source string, rewriter *NamespaceRewriter) []float64 {
	t.Helper()

	destStdinReader, destStdinWriter := io.Pipe()
	destStdoutReader, destStdoutWriter := io.Pipe()

	var counts []float64

	destDone := make(chan struct{})

	go func() {
		statsDestination(destStdinReader, destStdoutWriter, &counts)
		close(destDone)
	}()

	_, err := Run(context.Background(), strings.NewReader(source), destStdinWriter, destStdoutReader, &DefaultHandler{}, RunConfig{Rewriter: rewriter}, nil)
	require.NoError(t, err)
	<-destDone

	return counts
}

func TestRun_SourceStats(t *testing.T) {
	tests := []struct {
		name     string
		source   string
		rewriter *NamespaceRewriter
		want     []float64
	}{
		{
			name:   "stream state counts its stream's records",
			source: statsSource(statsRecords(3, "a", "x"), statsStreamState("a")),
			want:   []float64{3},
		},
		{
			name: "each stream state counts only its stream since its previous state",
			source: statsSource(
				statsRecords(2, "a", "x"), statsRecords(1, "b", "x"),
				statsStreamState("a"), statsStreamState("b"),
				statsRecords(1, "a", "x"), statsStreamState("a"),
			),
			want: []float64{2, 1, 1},
		},
		{
			name:   "state with no records counts zero",
			source: statsSource(statsStreamState("a")),
			want:   []float64{0},
		},
		{
			name:   "streams with the same name in different namespaces count apart",
			source: statsSource(statsRecords(2, "a", "x"), statsRecords(1, "a", "y"), statsStreamState("a")),
			want:   []float64{2},
		},
		{
			name: "global state counts every stream since the previous state",
			source: statsSource(
				statsRecords(2, "a", "x"), statsRecords(3, "b", "x"), statsGlobalState,
				statsRecords(1, "a", "x"), statsGlobalState,
			),
			want: []float64{5, 1},
		},
		{
			name:   "legacy state counts like global",
			source: statsSource(statsRecords(2, "a", "x"), statsLegacyState),
			want:   []float64{2},
		},
		{
			name: "global state resets the per-stream counts",
			source: statsSource(
				statsRecords(2, "a", "x"), statsGlobalState,
				statsRecords(1, "a", "x"), statsStreamState("a"),
			),
			want: []float64{2, 1},
		},
		{
			name:   "source's own count passes through unchanged",
			source: statsSource(statsRecords(2, "a", "x"), statsStreamStateWithCount("a", 99)),
			want:   []float64{99},
		},
		{
			name: "source's own count resets the counter",
			source: statsSource(
				statsRecords(2, "a", "x"), statsStreamStateWithCount("a", 99),
				statsRecords(1, "a", "x"), statsStreamState("a"),
			),
			want: []float64{99, 1},
		},
		{
			name:     "streams renamed by the rewriter count under the source's names",
			source:   statsSource(statsRecords(3, "a", "x"), statsStreamState("a")),
			rewriter: NewNamespaceRewriter(makeCatalog(makeStream("a", "x")), pipelineservice.NamespaceDefinitionSource, nil, strPtr("pre_")),
			want:     []float64{3},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, runSourceStats(t, tt.source, tt.rewriter))
		})
	}
}

package pipelineexec

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/synclet-io/synclet/pkg/mapper"
	"github.com/synclet-io/synclet/pkg/protocol"
)

const mapperNamespace = "public"

const ordersMapperSchema = `{"type":"object","properties":{
	"id":{"type":"integer"},
	"email":{"type":"string"},
	"region":{"type":"string"},
	"secret":{"type":"string"},
	"status":{"type":"string"},
	"amount":{"type":"string"}
}}`

// mapperRun is what the destination received and what the router reported.
type mapperRun struct {
	records map[string][]string
	counts  []float64
	stats   *Stats
	err     error
}

// mapperDestination records every RECORD's data by "<namespace>.<stream>" and
// the sourceStats of every STATE, which it echoes back.
func mapperDestination(stdin io.Reader, stdout *io.PipeWriter, run *mapperRun) {
	reader := protocol.NewMessageReader(stdin)
	writer := protocol.NewMessageWriter(stdout)

	defer func() { _ = stdout.Close() }()

	for {
		msg, err := reader.Read()
		if err != nil {
			return
		}

		switch msg.Type {
		case protocol.MessageTypeRecord:
			key := msg.Record.Namespace + "." + msg.Record.Stream
			run.records[key] = append(run.records[key], string(msg.Record.Data))
		case protocol.MessageTypeState:
			count := float64(-1)
			if msg.State.SourceStats != nil {
				count = msg.State.SourceStats.RecordCount
			}

			run.counts = append(run.counts, count)

			if err := writer.Write(msg); err != nil {
				return
			}
		}
	}
}

func runWithMappers(t *testing.T, catalog *protocol.ConfiguredAirbyteCatalog, salt []byte, sourceLines ...string) *mapperRun {
	t.Helper()

	transformer, err := NewRecordTransformer(catalog, salt, mapper.SyncInfo{StartedAt: time.Unix(0, 0).UTC(), JobID: "job-1", Connection: "conn-1"})
	require.NoError(t, err)

	destStdinReader, destStdinWriter := io.Pipe()
	destStdoutReader, destStdoutWriter := io.Pipe()

	run := &mapperRun{records: map[string][]string{}}
	destDone := make(chan struct{})

	go func() {
		mapperDestination(destStdinReader, destStdoutWriter, run)
		close(destDone)
	}()

	source := strings.Join(sourceLines, "\n") + "\n"
	run.stats, run.err = Run(context.Background(), strings.NewReader(source), destStdinWriter, destStdoutReader, &DefaultHandler{}, RunConfig{Transformer: transformer}, nil)

	<-destDone

	return run
}

func mapperStream(name, mappers string, selected ...string) protocol.ConfiguredAirbyteStream {
	stream := makeStream(name, mapperNamespace)
	stream.Stream.JSONSchema = json.RawMessage(ordersMapperSchema)

	if mappers != "" {
		stream.Mappers = json.RawMessage(mappers)
	}

	for _, field := range selected {
		stream.SelectedFields = append(stream.SelectedFields, protocol.SelectedField{FieldPath: []string{field}})
	}

	return stream
}

func mapperRecord(name, data string) string {
	return fmt.Sprintf(`{"type":"RECORD","record":{"stream":%q,"namespace":%q,"data":%s,"emitted_at":1}}`, name, mapperNamespace, data)
}

func mapperStreamState(name string) string {
	return fmt.Sprintf(`{"type":"STATE","state":{"type":"STREAM","stream":{"stream_descriptor":{"name":%q,"namespace":%q},"stream_state":{"cursor":1}}}}`, name, mapperNamespace)
}

func saltedHash(salt, value string) string {
	sum := sha256.Sum256([]byte(salt + value))

	return hex.EncodeToString(sum[:])
}

func TestRouter_MappersTransformRecords(t *testing.T) {
	catalog := makeCatalog(
		mapperStream("orders",
			`[{"rename":{"email":"contact"}},{"set":{"secret_code":"r.secret.lowerAscii()"}},{"hash":["region"]}]`,
			"id", "email", "region"),
		mapperStream("plain", ""),
	)

	run := runWithMappers(t, catalog, []byte("salt-1"),
		mapperRecord("orders", `{"id":7,"email":"a@b.c","region":"eu","secret":"XYZ","status":"new"}`),
		mapperRecord("plain", `{"z":1,"a":{"y":2,"b":3}}`),
	)
	require.NoError(t, run.err)

	require.Len(t, run.records["public.orders"], 1)
	assert.JSONEq(t, fmt.Sprintf(`{"id":7,"contact":"a@b.c","region":%q,"secret_code":"xyz"}`, saltedHash("salt-1", "eu")), run.records["public.orders"][0])
	assert.Equal(t, []string{`{"z":1,"a":{"y":2,"b":3}}`}, run.records["public.plain"])
}

func TestRouter_FilteredRecordsNotCounted(t *testing.T) {
	catalog := makeCatalog(mapperStream("orders", `[{"filter":"r.status != 'test'"}]`))

	run := runWithMappers(t, catalog, nil,
		mapperRecord("orders", `{"id":1,"status":"new"}`),
		mapperRecord("orders", `{"id":2,"status":"test"}`),
		mapperRecord("orders", `{"id":3,"status":"new"}`),
		mapperStreamState("orders"),
	)
	require.NoError(t, run.err)

	assert.Equal(t, []float64{2}, run.counts)
	assert.Equal(t, int64(3), run.stats.RecordsRead)
	assert.Equal(t, int64(1), run.stats.RecordsFiltered)
	assert.Len(t, run.records["public.orders"], 2)
}

func TestRouter_MapperErrorFails(t *testing.T) {
	catalog := makeCatalog(mapperStream("orders", `[{"cast":{"amount":"integer"},"on_error":"fail"}]`))

	run := runWithMappers(t, catalog, nil,
		mapperRecord("orders", `{"id":1,"amount":"abc"}`),
	)

	require.Error(t, run.err)
	assert.Contains(t, run.err.Error(), `stream "public.orders"`)
	assert.Contains(t, run.err.Error(), "mapper step 0")
	assert.Empty(t, run.records["public.orders"])
}

func TestRouter_MapperErrorNullAndSkip(t *testing.T) {
	t.Run("null", func(t *testing.T) {
		catalog := makeCatalog(mapperStream("orders", `[{"cast":{"amount":"integer"},"on_error":"null"}]`))

		run := runWithMappers(t, catalog, nil,
			mapperRecord("orders", `{"id":1,"amount":"abc"}`),
		)
		require.NoError(t, run.err)

		require.Len(t, run.records["public.orders"], 1)
		assert.JSONEq(t, `{"id":1,"amount":null}`, run.records["public.orders"][0])
		assert.Zero(t, run.stats.RecordsFiltered)
	})

	t.Run("skip", func(t *testing.T) {
		catalog := makeCatalog(mapperStream("orders", `[{"cast":{"amount":"integer"},"on_error":"skip"}]`))

		run := runWithMappers(t, catalog, nil,
			mapperRecord("orders", `{"id":1,"amount":"abc"}`),
			mapperRecord("orders", `{"id":2,"amount":"5"}`),
		)
		require.NoError(t, run.err)

		require.Len(t, run.records["public.orders"], 1)
		assert.JSONEq(t, `{"id":2,"amount":5}`, run.records["public.orders"][0])
		assert.Equal(t, int64(1), run.stats.RecordsFiltered)
	})
}

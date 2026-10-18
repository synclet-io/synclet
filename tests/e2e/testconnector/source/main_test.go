package main

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"os"
	"testing"

	airbyte "github.com/synclet-io/synclet-go-sdk"
)

// The fixture is the state the platform's GetSyncState returns once a STREAM
// checkpoint follows a GLOBAL one; pipelinestate's tests assert that output.
const streamAfterGlobalState = "../../../../modules/pipeline/pipelineservice/pipelinestate/testdata/stream_after_global_state.json"

func TestReadResumesFromStreamStateAfterGlobal(t *testing.T) {
	raw, err := os.ReadFile(streamAfterGlobalState)
	if err != nil {
		t.Fatal(err)
	}

	state, err := airbyte.ParseInputState(raw)
	if err != nil {
		t.Fatalf("ParseInputState: %v", err)
	}

	cfg := sourceConfig{Streams: []streamConfig{
		{Name: "users", Namespace: "public", RecordCount: 1},
		{Name: "orders", RecordCount: 1},
	}}

	var out bytes.Buffer
	if err := (&testSource{}).Read(context.Background(), cfg, nil, state, airbyte.NewEmitter(&out)); err != nil {
		t.Fatalf("Read: %v", err)
	}

	firstIDs := map[string]int{}
	scanner := bufio.NewScanner(&out)
	for scanner.Scan() {
		var msg airbyte.Message
		if err := json.Unmarshal(scanner.Bytes(), &msg); err != nil {
			t.Fatal(err)
		}
		if msg.Type != airbyte.MessageTypeRecord {
			continue
		}
		if _, seen := firstIDs[msg.Record.Stream]; seen {
			continue
		}
		var rec struct {
			ID int `json:"id"`
		}
		if err := json.Unmarshal(msg.Record.Data, &rec); err != nil {
			t.Fatal(err)
		}
		firstIDs[msg.Record.Stream] = rec.ID
	}

	want := map[string]int{"users": 150, "orders": 7}
	for stream, id := range want {
		if firstIDs[stream] != id {
			t.Errorf("stream %s resumed at %d, want %d", stream, firstIDs[stream], id)
		}
	}
}

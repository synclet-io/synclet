package mapper

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/synclet-io/synclet/pkg/protocol"
)

// DefaultCostLimit is the CEL cost limit per expression evaluation when Input.CostLimit is zero.
const DefaultCostLimit uint64 = 100000

// SelectionStep is the Issue.Step of a problem with the field selection.
const SelectionStep = -1

// Input is everything Compile needs about one stream.
type Input struct {
	Schema              json.RawMessage
	Steps               []Step
	SelectedFields      []protocol.SelectedField
	PrimaryKey          [][]string
	CursorField         []string
	SyncMode            protocol.SyncMode
	DestinationSyncMode protocol.DestinationSyncMode
	Salt                []byte
	CostLimit           uint64
}

// Result is the output of Compile: the output schema, primary key and cursor
// field, or the issues found. CursorField is nil when a step dropped the cursor.
type Result struct {
	Schema      json.RawMessage
	PrimaryKey  [][]string
	CursorField []string
	Errors      []Issue
	Warnings    []Issue
}

// Issue is a problem found in one step, or in the field selection when Step is SelectionStep.
type Issue struct {
	Step    int    `json:"step"`
	Field   string `json:"field,omitempty"`
	Message string `json:"message"`
}

// SyncInfo is the job metadata expressions see as `sync`.
type SyncInfo struct {
	StartedAt  time.Time
	JobID      string
	Connection string
}

// StepError is a runtime failure of a step whose on_error is fail.
type StepError struct {
	Step  int
	Field string
	Err   error
}

func (e *StepError) Error() string {
	if e.Field != "" {
		return fmt.Sprintf("mapper step %d, field %q: %v", e.Step, e.Field, e.Err)
	}

	return fmt.Sprintf("mapper step %d: %v", e.Step, e.Err)
}

func (e *StepError) Unwrap() error { return e.Err }

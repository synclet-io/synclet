// Package mapper compiles a stream's mapper list against its JSON schema and
// runs the compiled chain on one record at a time.
package mapper

import (
	"encoding/json"
	"fmt"
)

// OnError values.
const (
	OnErrorFail = "fail"
	OnErrorNull = "null"
	OnErrorSkip = "skip"
)

// Cast targets.
const (
	CastString    = "string"
	CastInteger   = "integer"
	CastNumber    = "number"
	CastBoolean   = "boolean"
	CastTimestamp = "timestamp"
	CastDatetime  = "datetime"
	CastDate      = "date"
	CastObject    = "object"
	CastArray     = "array"
)

// Timestamp formats besides token patterns.
const (
	FormatISO8601 = "iso8601"
	FormatUnix    = "unix"
	FormatUnixMs  = "unix_ms"
)

// Step is one mapper of a stream's chain. Exactly one mapper field is set.
type Step struct {
	Rename  map[string]string   `json:"rename,omitempty"`
	Cast    map[string]CastSpec `json:"cast,omitempty"`
	Set     map[string]string   `json:"set,omitempty"`
	Hash    []string            `json:"hash,omitempty"`
	Mask    map[string]MaskSpec `json:"mask,omitempty"`
	Filter  *string             `json:"filter,omitempty"`
	Drop    []string            `json:"drop,omitempty"`
	OnError string              `json:"on_error,omitempty"`
}

// CastSpec converts one field to a target type.
type CastSpec struct {
	To                 string   `json:"to"`
	Format             string   `json:"format,omitempty"`
	Timezone           string   `json:"timezone,omitempty"`
	DecimalSeparator   string   `json:"decimal_separator,omitempty"`
	ThousandsSeparator string   `json:"thousands_separator,omitempty"`
	Trim               bool     `json:"trim,omitempty"`
	StripPrefix        string   `json:"strip_prefix,omitempty"`
	StripSuffix        string   `json:"strip_suffix,omitempty"`
	TrueValues         []string `json:"true_values,omitempty"`
	FalseValues        []string `json:"false_values,omitempty"`
	Source             string   `json:"source,omitempty"`
	OnError            string   `json:"on_error,omitempty"`
}

// UnmarshalJSON accepts either a full spec object or a bare target string.
func (c *CastSpec) UnmarshalJSON(data []byte) error {
	var target string
	if err := json.Unmarshal(data, &target); err == nil {
		*c = CastSpec{To: target}

		return nil
	}

	type plain CastSpec

	var spec plain
	if err := json.Unmarshal(data, &spec); err != nil {
		return fmt.Errorf("cast spec: %w", err)
	}

	*c = CastSpec(spec)

	return nil
}

// MaskSpec keeps the first and last characters of a value and stars the rest.
type MaskSpec struct {
	KeepFirst int `json:"keep_first,omitempty"`
	KeepLast  int `json:"keep_last,omitempty"`
}

// kind names the mapper a step holds, or "" when it holds none or several.
func (s *Step) kind() (name string, kinds int) {
	if s.Rename != nil {
		kinds++
		name = "rename"
	}

	if s.Cast != nil {
		kinds++
		name = "cast"
	}

	if s.Set != nil {
		kinds++
		name = "set"
	}

	if s.Hash != nil {
		kinds++
		name = "hash"
	}

	if s.Mask != nil {
		kinds++
		name = "mask"
	}

	if s.Filter != nil {
		kinds++
		name = "filter"
	}

	if s.Drop != nil {
		kinds++
		name = "drop"
	}

	return name, kinds
}

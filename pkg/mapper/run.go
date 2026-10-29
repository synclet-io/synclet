package mapper

import (
	"bytes"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"strconv"
	"strings"
	"time"

	"github.com/google/cel-go/cel"
	"github.com/google/cel-go/common/types"
	"github.com/google/cel-go/common/types/ref"
	"github.com/google/cel-go/common/types/traits"
)

var (
	errNotScalar   = errors.New("value is not a string, number or boolean")
	errCast        = errors.New("cannot cast")
	errFilterValue = errors.New("filter did not return a bool")
)

// datetimeLayout writes a wall-clock timestamp without a zone offset.
const datetimeLayout = "2006-01-02T15:04:05.999999999"

// DecodeRecord decodes a record's JSON data, keeping numbers as json.Number
// so Run reads them as the schema says.
func DecodeRecord(data []byte) (map[string]any, error) {
	dec := json.NewDecoder(bytes.NewReader(data))
	dec.UseNumber()

	var rec map[string]any
	if err := dec.Decode(&rec); err != nil {
		return nil, fmt.Errorf("decoding record: %w", err)
	}

	return rec, nil
}

// Run applies the chain to one record and projects it to the field selection.
// keep is false when a filter or an on_error skip drops the record; err is a
// *StepError when a step with on_error fail fails. The input map is not modified.
func (c *Chain) Run(rec map[string]any, sync SyncInfo) (out map[string]any, keep bool, err error) {
	cur, _ := normalize(rec, c.input).(map[string]any) //nolint:errcheck // normalize keeps a map a map
	if cur == nil {
		cur = map[string]any{}
	}

	syncVal := map[string]any{
		"started_at": sync.StartedAt,
		"job_id":     sync.JobID,
		"connection": sync.Connection,
	}

	for idx := range c.steps {
		next, kept, stepErr := c.steps[idx].run(cur, syncVal, c.salt)
		if stepErr != nil {
			return nil, false, withStep(stepErr, idx)
		}

		if !kept {
			return nil, false, nil
		}

		cur = next
	}

	if c.selection == nil {
		return cur, true, nil
	}

	out = c.selection.apply(cur)

	for name, sub := range c.written {
		val, ok := cur[name]
		if !ok {
			continue
		}

		if nested, isMap := val.(map[string]any); isMap && sub != nil {
			val = sub.apply(nested)
		}

		out[name] = val
	}

	return out, true, nil
}

func withStep(err error, idx int) error {
	var stepErr *StepError
	if errors.As(err, &stepErr) {
		stepErr.Step = idx

		return stepErr
	}

	return &StepError{Step: idx, Err: err}
}

// outcome applies on_error to a failure of one field: fail returns the error,
// null writes null, skip drops the record.
func outcome(onError, field string, out map[string]any, err error) (bool, error) {
	switch onError {
	case OnErrorNull:
		if field != "" {
			out[field] = nil
		}

		return true, nil
	case OnErrorSkip:
		return false, nil
	default:
		return false, &StepError{Field: field, Err: err}
	}
}

func (s *compiledStep) run(cur, syncVal map[string]any, salt []byte) (next map[string]any, keep bool, err error) {
	if s.kind == "filter" {
		keep, err = evalFilter(s.filter, cur, syncVal)
		if err != nil {
			keep, err = outcome(s.onError, "", nil, err)
		}

		return cur, keep, err
	}

	out := make(map[string]any, len(cur)+len(s.sets))
	for key, val := range cur {
		out[key] = val
	}

	for _, pair := range s.renames {
		delete(out, pair[0])
	}

	for _, field := range s.drops {
		delete(out, field)
	}

	for _, pair := range s.renames {
		if val, ok := cur[pair[0]]; ok {
			out[pair[1]] = val
		}
	}

	for _, set := range s.sets {
		val, err := eval(set.program, cur, syncVal)
		if err != nil {
			if keep, err := outcome(s.onError, set.field, out, err); !keep {
				return nil, false, err
			}

			continue
		}

		out[set.field] = val
	}

	for i := range s.casts {
		if keep, err := s.casts[i].run(cur, syncVal, out); !keep {
			return nil, false, err
		}
	}

	for _, field := range s.fields {
		if keep, err := apply(s.onError, field, cur, out, func(val any) (any, error) { return hashValue(salt, val) }); !keep {
			return nil, false, err
		}
	}

	for _, field := range sortedKeys(s.masks) {
		spec := s.masks[field]
		if keep, err := apply(s.onError, field, cur, out, func(val any) (any, error) { return maskValue(spec, val) }); !keep {
			return nil, false, err
		}
	}

	return out, true, nil
}

// apply writes fn of a field's value; a missing field stays missing and null stays null.
func apply(onError, field string, cur, out map[string]any, convert func(any) (any, error)) (bool, error) {
	val, ok := cur[field]
	if !ok || val == nil {
		return true, nil
	}

	res, err := convert(val)
	if err != nil {
		return outcome(onError, field, out, err)
	}

	out[field] = res

	return true, nil
}

func (cc *compiledCast) run(cur, syncVal, out map[string]any) (bool, error) {
	var (
		val any
		ok  bool
	)

	if cc.source != nil {
		res, err := eval(cc.source, cur, syncVal)
		if err != nil {
			return outcome(cc.onError, cc.field, out, err)
		}

		val, ok = res, true
	} else {
		val, ok = cur[cc.field]
	}

	if !ok {
		return true, nil
	}

	if val == nil {
		out[cc.field] = nil

		return true, nil
	}

	res, err := cc.convert(val)
	if err != nil {
		return outcome(cc.onError, cc.field, out, err)
	}

	out[cc.field] = res

	return true, nil
}

func eval(program cel.Program, cur, syncVal map[string]any) (any, error) {
	val, _, err := program.Eval(map[string]any{"r": cur, "sync": syncVal})
	if err != nil {
		return nil, fmt.Errorf("evaluating expression: %w", err)
	}

	return toNative(val)
}

func evalFilter(program cel.Program, cur, syncVal map[string]any) (bool, error) {
	val, _, err := program.Eval(map[string]any{"r": cur, "sync": syncVal})
	if err != nil {
		return false, fmt.Errorf("evaluating filter: %w", err)
	}

	keep, ok := val.(types.Bool)
	if !ok {
		return false, fmt.Errorf("%w: got %s", errFilterValue, val.Type())
	}

	return bool(keep), nil
}

// toNative converts a CEL result to the record's Go form; timestamps become RFC 3339 strings.
func toNative(val ref.Val) (any, error) {
	switch typed := val.(type) {
	case types.Null:
		return nil, nil
	case types.Int:
		return int64(typed), nil
	case types.Uint:
		if uint64(typed) > math.MaxInt64 {
			return float64(typed), nil
		}

		return int64(typed), nil
	case types.Double:
		return float64(typed), nil
	case types.String:
		return string(typed), nil
	case types.Bool:
		return bool(typed), nil
	case types.Bytes:
		return base64.StdEncoding.EncodeToString(typed), nil
	case types.Timestamp:
		return typed.Format(time.RFC3339Nano), nil
	case types.Duration:
		return typed.String(), nil
	case *types.Err:
		return nil, typed
	case traits.Lister:
		return listToNative(typed)
	case traits.Mapper:
		return mapToNative(typed)
	}

	return val.Value(), nil
}

func listToNative(list traits.Lister) ([]any, error) {
	out := []any{}

	for it := list.Iterator(); it.HasNext() == types.True; {
		item, err := toNative(it.Next())
		if err != nil {
			return nil, err
		}

		out = append(out, item)
	}

	return out, nil
}

func mapToNative(mapper traits.Mapper) (map[string]any, error) {
	out := map[string]any{}

	for it := mapper.Iterator(); it.HasNext() == types.True; {
		key := it.Next()

		name, ok := key.(types.String)
		if !ok {
			return nil, fmt.Errorf("%w: map key %v is not a string", errCast, key)
		}

		item, err := toNative(mapper.Get(key))
		if err != nil {
			return nil, err
		}

		out[string(name)] = item
	}

	return out, nil
}

// scalarText is a scalar's JSON text, a string's own text.
func scalarText(val any) (string, error) {
	switch typed := val.(type) {
	case string:
		return typed, nil
	case int64:
		return strconv.FormatInt(typed, 10), nil
	case float64:
		text, err := json.Marshal(typed)
		if err != nil {
			return "", fmt.Errorf("%w: %w", errNotScalar, err)
		}

		return string(text), nil
	case bool:
		return strconv.FormatBool(typed), nil
	case json.Number:
		return string(typed), nil
	}

	return "", fmt.Errorf("%w: %T", errNotScalar, val)
}

func hashValue(salt []byte, val any) (any, error) {
	text, err := scalarText(val)
	if err != nil {
		return nil, err
	}

	hasher := sha256.New()
	hasher.Write(salt)
	hasher.Write([]byte(text))

	return hex.EncodeToString(hasher.Sum(nil)), nil
}

func maskValue(spec MaskSpec, val any) (any, error) {
	text, err := scalarText(val)
	if err != nil {
		return nil, err
	}

	runes := []rune(text)
	if spec.KeepFirst+spec.KeepLast >= len(runes) {
		return text, nil
	}

	for i := spec.KeepFirst; i < len(runes)-spec.KeepLast; i++ {
		runes[i] = '*'
	}

	return string(runes), nil
}

func (cc *compiledCast) convert(val any) (any, error) {
	spec := cc.spec

	switch spec.To {
	case CastString:
		return cc.toString(val)
	case CastInteger, CastNumber:
		return cc.toNumber(val)
	case CastBoolean:
		return toBoolean(spec, val)
	case CastTimestamp, CastDatetime, CastDate:
		parsed, err := cc.format.parse(val, cc.loc)
		if err != nil {
			return nil, err
		}

		switch spec.To {
		case CastDate:
			return parsed.Format(time.DateOnly), nil
		case CastDatetime:
			return parsed.In(cc.loc).Format(datetimeLayout), nil
		}

		return parsed.Format(time.RFC3339Nano), nil
	case CastObject, CastArray:
		return toJSONValue(spec.To, val)
	}

	return nil, fmt.Errorf("%w to %q", errCast, spec.To)
}

func (cc *compiledCast) toString(val any) (any, error) {
	if cc.spec.Format != "" {
		parsed, err := timeFormat{name: FormatISO8601}.parse(val, cc.loc)
		if err != nil {
			return nil, err
		}

		return cc.format.format(parsed), nil
	}

	switch typed := val.(type) {
	case float64, int64:
		text, _ := scalarText(typed) //nolint:errcheck // numbers are scalars
		if cc.spec.DecimalSeparator != "" {
			text = strings.Replace(text, ".", cc.spec.DecimalSeparator, 1)
		}

		return text, nil
	case map[string]any, []any:
		out, err := json.Marshal(typed)
		if err != nil {
			return nil, fmt.Errorf("%w to string: %w", errCast, err)
		}

		return string(out), nil
	}

	return scalarText(val)
}

func (cc *compiledCast) toNumber(val any) (any, error) {
	spec := cc.spec

	var num float64

	switch typed := val.(type) {
	case int64:
		if spec.To == CastInteger {
			return typed, nil
		}

		return float64(typed), nil
	case float64:
		num = typed
	case string:
		text := cleanNumber(spec, typed)

		if spec.To == CastInteger {
			if n, err := strconv.ParseInt(text, 10, 64); err == nil {
				return n, nil
			}
		}

		parsed, err := strconv.ParseFloat(text, 64)
		if err != nil {
			return nil, fmt.Errorf("%w %q to %s", errCast, typed, spec.To)
		}

		num = parsed
	default:
		return nil, fmt.Errorf("%w %T to %s", errCast, val, spec.To)
	}

	if spec.To == CastNumber {
		return num, nil
	}

	if num != math.Trunc(num) || math.Abs(num) >= 1<<63 {
		return nil, fmt.Errorf("%w %v to integer: not a whole number", errCast, num)
	}

	return int64(num), nil
}

func cleanNumber(spec CastSpec, text string) string {
	if spec.Trim {
		text = strings.TrimSpace(text)
	}

	text = strings.TrimSuffix(strings.TrimPrefix(text, spec.StripPrefix), spec.StripSuffix)

	if spec.Trim {
		text = strings.TrimSpace(text)
	}

	if spec.ThousandsSeparator != "" {
		text = strings.ReplaceAll(text, spec.ThousandsSeparator, "")
	}

	if spec.DecimalSeparator != "" && spec.DecimalSeparator != "." {
		text = strings.ReplaceAll(text, spec.DecimalSeparator, ".")
	}

	return text
}

var (
	defaultTrue  = []string{"true", "1", "yes", "y", "t"}
	defaultFalse = []string{"false", "0", "no", "n", "f"}
)

func toBoolean(spec CastSpec, val any) (any, error) {
	switch typed := val.(type) {
	case bool:
		return typed, nil
	case int64:
		if typed == 0 || typed == 1 {
			return typed == 1, nil
		}
	case float64:
		if typed == 0 || typed == 1 {
			return typed == 1, nil
		}
	case string:
		trueValues, falseValues := spec.TrueValues, spec.FalseValues
		if len(trueValues) == 0 && len(falseValues) == 0 {
			trueValues, falseValues = defaultTrue, defaultFalse
		}

		text := strings.TrimSpace(typed)
		if containsFold(trueValues, text) {
			return true, nil
		}

		if containsFold(falseValues, text) {
			return false, nil
		}
	}

	return nil, fmt.Errorf("%w %v to boolean", errCast, val)
}

func containsFold(values []string, text string) bool {
	for _, value := range values {
		if strings.EqualFold(value, text) {
			return true
		}
	}

	return false
}

func toJSONValue(target string, val any) (any, error) {
	if text, ok := val.(string); ok {
		dec := json.NewDecoder(strings.NewReader(text))
		dec.UseNumber()

		var parsed any
		if err := dec.Decode(&parsed); err != nil {
			return nil, fmt.Errorf("%w %q to %s: %w", errCast, text, target, err)
		}

		val = normalize(parsed, nil)
	}

	switch val.(type) {
	case map[string]any:
		if target == CastObject {
			return val, nil
		}
	case []any:
		if target == CastArray {
			return val, nil
		}
	}

	return nil, fmt.Errorf("%w %T to %s", errCast, val, target)
}

package mapper

import (
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"strconv"
	"strings"
	"time"
	_ "time/tzdata" // timezones resolve on hosts without a zoneinfo database
)

var errNotTimestamp = errors.New("value is not a timestamp")

// patternTokens maps the pattern tokens to Go layout elements, longest first.
var patternTokens = []struct{ token, layout string }{
	{"yyyy", "2006"},
	{"yy", "06"},
	{"MM", "01"},
	{"dd", "02"},
	{"HH", "15"},
	{"hh", "03"},
	{"mm", "04"},
	{"ss", "05"},
	{"SSS", "000"},
	{"XXX", "Z07:00"},
	{"a", "PM"},
}

var isoLayouts = []string{
	time.RFC3339Nano,
	"2006-01-02T15:04:05.999999999",
	"2006-01-02 15:04:05.999999999Z07:00",
	"2006-01-02 15:04:05.999999999",
	"2006-01-02",
}

// timeFormat is a parsed cast format: iso8601, unix, unix_ms or a Go layout
// converted from a token pattern.
type timeFormat struct {
	name   string
	layout string
}

func parseTimeFormat(format string) (timeFormat, error) {
	switch format {
	case "", FormatISO8601:
		return timeFormat{name: FormatISO8601}, nil
	case FormatUnix, FormatUnixMs:
		return timeFormat{name: format}, nil
	}

	layout, err := patternLayout(format)
	if err != nil {
		return timeFormat{}, err
	}

	return timeFormat{name: format, layout: layout}, nil
}

// patternLayout converts a `dd.MM.yyyy HH:mm` style pattern to a Go layout.
// Letters outside the tokens, digits and Go layout words are rejected; text in
// single quotes is copied as is when it holds no letter but T.
func patternLayout(pattern string) (string, error) {
	errUnknown := fmt.Errorf("unknown cast format %q", pattern)

	var out strings.Builder

	rest := pattern

outer:
	for rest != "" {
		if rest[0] == '\'' {
			end := strings.IndexByte(rest[1:], '\'')
			if end < 0 {
				return "", errUnknown
			}

			literal := rest[1 : end+1]
			if !safeLiteral(literal) {
				return "", errUnknown
			}

			out.WriteString(literal)

			rest = rest[end+2:]

			continue
		}

		for _, tok := range patternTokens {
			if strings.HasPrefix(rest, tok.token) {
				out.WriteString(tok.layout)
				rest = rest[len(tok.token):]

				continue outer
			}
		}

		if !safeLiteral(rest[:1]) {
			return "", errUnknown
		}

		out.WriteByte(rest[0])
		rest = rest[1:]
	}

	return out.String(), nil
}

func safeLiteral(literal string) bool {
	for _, char := range literal {
		isLetter := (char >= 'a' && char <= 'z') || (char >= 'A' && char <= 'Z')
		isDigit := char >= '0' && char <= '9'

		if isDigit || (isLetter && char != 'T') {
			return false
		}
	}

	return true
}

// parse reads a timestamp from a string, or from a number for unix formats.
func (f timeFormat) parse(value any, loc *time.Location) (time.Time, error) {
	switch f.name {
	case FormatUnix, FormatUnixMs:
		num, err := unixNumber(value)
		if err != nil {
			return time.Time{}, err
		}

		if f.name == FormatUnixMs {
			return time.UnixMilli(int64(num)).UTC(), nil
		}

		sec, frac := math.Modf(num)

		return time.Unix(int64(sec), int64(frac*1e9)).UTC(), nil
	}

	str, ok := value.(string)
	if !ok {
		return time.Time{}, fmt.Errorf("%w: %v", errNotTimestamp, value)
	}

	if f.layout != "" {
		parsed, err := time.ParseInLocation(f.layout, str, loc)
		if err != nil {
			return time.Time{}, fmt.Errorf("parsing %q with %q: %w", str, f.name, err)
		}

		return parsed, nil
	}

	for _, layout := range isoLayouts {
		if parsed, err := time.ParseInLocation(layout, str, loc); err == nil {
			return parsed, nil
		}
	}

	return time.Time{}, fmt.Errorf("%w: %q is not ISO 8601", errNotTimestamp, str)
}

// format writes a timestamp in this format.
func (f timeFormat) format(stamp time.Time) string {
	switch f.name {
	case FormatUnix:
		return strconv.FormatInt(stamp.Unix(), 10)
	case FormatUnixMs:
		return strconv.FormatInt(stamp.UnixMilli(), 10)
	case FormatISO8601:
		return stamp.Format(time.RFC3339Nano)
	}

	return stamp.Format(f.layout)
}

func unixNumber(value any) (float64, error) {
	switch val := value.(type) {
	case int64:
		return float64(val), nil
	case float64:
		return val, nil
	case json.Number:
		return val.Float64() //nolint:wrapcheck // strconv error names the value
	case string:
		num, err := strconv.ParseFloat(strings.TrimSpace(val), 64)
		if err != nil {
			return 0, fmt.Errorf("%w: %q is not a unix time", errNotTimestamp, val)
		}

		return num, nil
	}

	return 0, fmt.Errorf("%w: %v", errNotTimestamp, value)
}

func loadLocation(name string) (*time.Location, error) {
	if name == "" {
		return time.UTC, nil
	}

	loc, err := time.LoadLocation(name)
	if err != nil {
		return nil, fmt.Errorf("unknown timezone %q", name) //nolint:err113 // message is the user-facing issue
	}

	return loc, nil
}

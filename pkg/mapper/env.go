package mapper

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/google/cel-go/cel"
	"github.com/google/cel-go/common"
	"github.com/google/cel-go/common/ast"
	"github.com/google/cel-go/common/operators"
	"github.com/google/cel-go/common/types"
	"github.com/google/cel-go/common/types/ref"
	"github.com/google/cel-go/ext"
)

const coalesceFunction = "__coalesce"

var coalesceParam = cel.TypeParamType("T")

var coalesceMismatch = regexp.MustCompile(`found no matching overload for '` + coalesceFunction + `' applied to '\((.*?)\)'`)

// coalesceMessage rewrites CEL's overload error for __coalesce into one that
// names coalesce and the two argument types that differ.
func coalesceMessage(msg string) string {
	return coalesceMismatch.ReplaceAllStringFunc(msg, func(match string) string {
		left, right := splitTypePair(coalesceMismatch.FindStringSubmatch(match)[1])

		return fmt.Sprintf("coalesce arguments have different types: %s and %s", left, right)
	})
}

// splitTypePair splits "A, B" at the comma outside any parentheses, so
// parameterized types such as map(string, int) stay whole.
func splitTypePair(pair string) (left, right string) {
	depth := 0

	for i, ch := range pair {
		switch ch {
		case '(':
			depth++
		case ')':
			depth--
		case ',':
			if depth == 0 {
				return pair[:i], strings.TrimSpace(pair[i+1:])
			}
		}
	}

	return pair, ""
}

// coalesceMacro expands coalesce(a, b, ...) right to left into
// has(r.a) ? __coalesce(r.a, rest) : rest, so a missing field falls back like
// a null one. A null literal argument is rejected: it would make the result null.
var coalesceMacro = cel.GlobalVarArgMacro("coalesce",
	func(helper cel.MacroExprFactory, _ ast.Expr, args []ast.Expr) (ast.Expr, *common.Error) {
		if len(args) < 2 {
			return nil, &common.Error{Message: "coalesce takes at least two arguments"}
		}

		for _, arg := range args {
			if arg.Kind() == ast.LiteralKind && arg.AsLiteral().Type() == types.NullType {
				return nil, helper.NewError(arg.ID(), "coalesce argument cannot be null")
			}
		}

		rest := args[len(args)-1]
		for i := len(args) - 2; i >= 0; i-- {
			rest = coalescePair(helper, args[i], rest)
		}

		return rest, nil
	})

func coalescePair(helper cel.MacroExprFactory, value, fallback ast.Expr) ast.Expr {
	call := helper.NewCall(coalesceFunction, value, fallback)

	if value.Kind() != ast.SelectKind || value.AsSelect().IsTestOnly() {
		return call
	}

	sel := value.AsSelect()
	present := helper.NewPresenceTest(helper.Copy(sel.Operand()), sel.FieldName())

	return helper.NewCall(operators.Conditional, present, call, helper.Copy(fallback))
}

func newEnv(provider *recordTypes, recordType *types.Type) (*cel.Env, error) {
	env, err := cel.NewEnv(
		cel.CustomTypeProvider(provider),
		ext.Strings(),
		cel.Variable("r", recordType),
		cel.Variable("sync", types.NewObjectType(syncTypeName)),
		cel.Macros(coalesceMacro),
		cel.Function("parseTimestamp",
			cel.Overload("parseTimestamp_string_string",
				[]*cel.Type{cel.StringType, cel.StringType}, cel.TimestampType,
				cel.BinaryBinding(func(value, format ref.Val) ref.Val {
					return parseTimestampVal(value, format, types.String(""))
				})),
			cel.Overload("parseTimestamp_string_string_string",
				[]*cel.Type{cel.StringType, cel.StringType, cel.StringType}, cel.TimestampType,
				cel.FunctionBinding(func(args ...ref.Val) ref.Val {
					return parseTimestampVal(args[0], args[1], args[2])
				})),
		),
		cel.Function("parseNumber",
			cel.Overload("parseNumber_string_string",
				[]*cel.Type{cel.StringType, cel.StringType}, cel.DoubleType,
				cel.BinaryBinding(func(value, sep ref.Val) ref.Val {
					num, err := parseNumber(string(value.(types.String)), string(sep.(types.String))) //nolint:forcetypeassert // overload guarantees strings
					if err != nil {
						return types.WrapErr(err)
					}

					return types.Double(num)
				})),
		),
		cel.Function("sha256",
			cel.Overload("sha256_string", []*cel.Type{cel.StringType}, cel.StringType,
				cel.UnaryBinding(func(value ref.Val) ref.Val {
					sum := sha256.Sum256([]byte(value.(types.String))) //nolint:forcetypeassert // overload guarantees a string

					return types.String(hex.EncodeToString(sum[:]))
				})),
		),
		cel.Function(coalesceFunction,
			cel.Overload(coalesceFunction+"_T_T", []*cel.Type{coalesceParam, coalesceParam}, coalesceParam,
				cel.BinaryBinding(func(value, fallback ref.Val) ref.Val {
					if value.Type() == types.NullType {
						return fallback
					}

					return value
				})),
		),
		cel.Function("catch",
			cel.Overload("catch_T_T", []*cel.Type{coalesceParam, coalesceParam}, coalesceParam,
				cel.OverloadIsNonStrict(),
				cel.BinaryBinding(catchVal)),
		),
		cel.Function("now",
			cel.Overload("now_timestamp", []*cel.Type{}, cel.TimestampType,
				cel.FunctionBinding(func(...ref.Val) ref.Val {
					return types.Timestamp{Time: time.Now().UTC()}
				})),
		),
	)
	if err != nil {
		return nil, fmt.Errorf("creating CEL environment: %w", err)
	}

	return env, nil
}

// catchVal returns value, or fallback when evaluating value failed; an
// unknown value stays unknown.
func catchVal(value, fallback ref.Val) ref.Val {
	if types.IsError(value) {
		return fallback
	}

	return value
}

func parseTimestampVal(value, format, timezone ref.Val) ref.Val {
	layout, err := parseTimeFormat(string(format.(types.String))) //nolint:forcetypeassert // overload guarantees strings
	if err != nil {
		return types.WrapErr(err)
	}

	loc, err := loadLocation(string(timezone.(types.String))) //nolint:forcetypeassert // overload guarantees strings
	if err != nil {
		return types.WrapErr(err)
	}

	parsed, err := layout.parse(string(value.(types.String)), loc) //nolint:forcetypeassert // overload guarantees strings
	if err != nil {
		return types.WrapErr(err)
	}

	return types.Timestamp{Time: parsed}
}

// parseNumber reads a number written with the given decimal separator; the
// other of `.` and `,` and spaces are taken as thousands separators.
func parseNumber(value, decimalSep string) (float64, error) {
	thousands := ","
	if decimalSep == "," {
		thousands = "."
	}

	clean := strings.NewReplacer(thousands, "", " ", "", " ", "").Replace(strings.TrimSpace(value))
	if decimalSep != "" && decimalSep != "." {
		clean = strings.ReplaceAll(clean, decimalSep, ".")
	}

	num, err := strconv.ParseFloat(clean, 64)
	if err != nil {
		return 0, fmt.Errorf("%q is not a number", value) //nolint:err113 // message names the value
	}

	return num, nil
}

// checkLiteralFormats reports a constant format or timezone argument of
// parseTimestamp that would fail on every record.
func checkLiteralFormats(checked *cel.Ast) error {
	calls := ast.MatchDescendants(ast.NavigateAST(checked.NativeRep()), ast.FunctionMatcher("parseTimestamp"))
	for _, call := range calls {
		args := call.AsCall().Args()

		if lit, ok := literalString(args, 1); ok {
			if _, err := parseTimeFormat(lit); err != nil {
				return err
			}
		}

		if lit, ok := literalString(args, 2); ok {
			if _, err := loadLocation(lit); err != nil {
				return err
			}
		}
	}

	return nil
}

func literalString(args []ast.Expr, idx int) (string, bool) {
	if idx >= len(args) || args[idx].Kind() != ast.LiteralKind {
		return "", false
	}

	str, ok := args[idx].AsLiteral().(types.String)

	return string(str), ok
}

// readsNullable reports whether an expression selects a record field, at any
// depth, whose schema admits null. Every argument of coalesce but the last is
// skipped: a null there falls back to the next one.
func readsNullable(checked *cel.Ast, props map[string]json.RawMessage) bool {
	root := ast.NavigateAST(checked.NativeRep())

	fallenBack := map[int64]bool{}

	for _, call := range ast.MatchDescendants(root, ast.FunctionMatcher(coalesceFunction)) {
		for _, sel := range ast.MatchDescendants(call.Children()[0], ast.KindMatcher(ast.SelectKind)) {
			fallenBack[sel.ID()] = true
		}
	}

	for _, sel := range ast.MatchDescendants(root, ast.KindMatcher(ast.SelectKind)) {
		if sel.AsSelect().IsTestOnly() || fallenBack[sel.ID()] {
			continue
		}

		path, ok := recordPath(sel)
		if ok && pathNullable(props, path) {
			return true
		}
	}

	return false
}

// recordPath returns the field path of a select chain rooted at `r`.
func recordPath(expr ast.Expr) ([]string, bool) {
	var path []string

	for expr.Kind() == ast.SelectKind {
		path = append([]string{expr.AsSelect().FieldName()}, path...)
		expr = expr.AsSelect().Operand()
	}

	return path, expr.Kind() == ast.IdentKind && expr.AsIdent() == "r"
}

func pathNullable(props map[string]json.RawMessage, path []string) bool {
	for _, field := range path {
		raw, ok := props[field]
		if !ok {
			return false
		}

		node := parseNode(raw)
		if node.nullable() {
			return true
		}

		props = node.Properties
	}

	return false
}

// referencedFields returns the top-level record fields an expression selects.
func referencedFields(parsed *cel.Ast) []string {
	var fields []string

	sels := ast.MatchDescendants(ast.NavigateAST(parsed.NativeRep()), ast.KindMatcher(ast.SelectKind))
	for _, sel := range sels {
		operand := sel.AsSelect().Operand()
		if operand.Kind() == ast.IdentKind && operand.AsIdent() == "r" {
			fields = append(fields, sel.AsSelect().FieldName())
		}
	}

	return fields
}

// dateConversionProblem reports int(), uint() or double() applied to a record
// field that holds a date, time or date-time string: CEL reads the string as a
// number, so the conversion fails on every record.
func dateConversionProblem(checked *cel.Ast, props map[string]json.RawMessage) string {
	root := ast.NavigateAST(checked.NativeRep())

	for _, conv := range []string{"int", "uint", "double"} {
		for _, call := range ast.MatchDescendants(root, ast.FunctionMatcher(conv)) {
			args := call.AsCall().Args()
			if len(args) != 1 {
				continue
			}

			path, ok := recordPath(args[0])
			if !ok || len(path) == 0 {
				continue
			}

			node, ok := pathNode(props, path)
			if !ok {
				continue
			}

			kind := node.dateKind()
			if kind == "" {
				continue
			}

			field := strings.Join(path, ".")
			msg := fmt.Sprintf("%q is a %s string, %s() cannot read it", field, kind, conv)

			if kind != "time" {
				msg += fmt.Sprintf(`; use %s(parseTimestamp(r.%s, "iso8601")) for epoch seconds`, conv, field)
			}

			return msg
		}
	}

	return ""
}

func pathNode(props map[string]json.RawMessage, path []string) (schemaNode, bool) {
	var node schemaNode

	for _, field := range path {
		raw, ok := props[field]
		if !ok {
			return schemaNode{}, false
		}

		node = parseNode(raw)
		props = node.Properties
	}

	return node, true
}

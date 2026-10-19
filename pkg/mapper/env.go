package mapper

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
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

// coalesceMacro expands coalesce(r.f, b) to has(r.f) ? __coalesce(r.f, b) : b,
// so a missing field falls back like a null one.
var coalesceMacro = cel.GlobalMacro("coalesce", 2,
	func(helper cel.MacroExprFactory, _ ast.Expr, args []ast.Expr) (ast.Expr, *common.Error) {
		call := helper.NewCall(coalesceFunction, args[0], args[1])

		if args[0].Kind() != ast.SelectKind || args[0].AsSelect().IsTestOnly() {
			return call, nil
		}

		sel := args[0].AsSelect()
		present := helper.NewPresenceTest(helper.Copy(sel.Operand()), sel.FieldName())

		return helper.NewCall(operators.Conditional, present, call, helper.Copy(args[1])), nil
	})

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
			cel.Overload(coalesceFunction+"_dyn_dyn", []*cel.Type{cel.DynType, cel.DynType}, cel.DynType,
				cel.BinaryBinding(func(value, fallback ref.Val) ref.Val {
					if value.Type() == types.NullType {
						return fallback
					}

					return value
				})),
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
// depth, whose schema admits null.
func readsNullable(checked *cel.Ast, props map[string]json.RawMessage) bool {
	sels := ast.MatchDescendants(ast.NavigateAST(checked.NativeRep()), ast.KindMatcher(ast.SelectKind))
	for _, sel := range sels {
		if sel.AsSelect().IsTestOnly() {
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

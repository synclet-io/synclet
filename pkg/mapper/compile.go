package mapper

import (
	"encoding/json"
	"fmt"
	"slices"
	"strings"
	"time"
	"unicode"

	"github.com/google/cel-go/cel"
	"github.com/google/cel-go/common/types"

	"github.com/synclet-io/synclet/pkg/protocol"
)

const maxFieldNameBytes = 255

// Chain is a compiled mapper list, safe for concurrent use.
type Chain struct {
	steps     []compiledStep
	input     *shape
	selection projection
	written   map[string]projection
	salt      []byte
}

type compiledStep struct {
	kind    string
	onError string
	renames [][2]string
	sets    []compiledSet
	casts   []compiledCast
	fields  []string
	masks   map[string]MaskSpec
	filter  cel.Program
}

type compiledSet struct {
	field   string
	program cel.Program
}

type compiledCast struct {
	field   string
	spec    CastSpec
	onError string
	format  timeFormat
	loc     *time.Location
	source  cel.Program

	readsNull bool
}

type compiler struct {
	in       Input
	result   *Result
	props    map[string]json.RawMessage
	created  map[string]int
	renamed  map[string]int
	written  map[string]bool
	origin   map[string]string
	required []string
	steps    []compiledStep
}

// Compile checks the steps against the schema and returns the runnable chain
// with the output schema and primary key, or a nil chain and the errors.
func Compile(input Input) (*Chain, *Result) {
	if input.CostLimit == 0 {
		input.CostLimit = DefaultCostLimit
	}

	cmp := &compiler{
		in:      input,
		result:  &Result{PrimaryKey: clonePaths(input.PrimaryKey)},
		created: map[string]int{},
		renamed: map[string]int{},
		written: map[string]bool{},
		origin:  map[string]string{},
	}

	root := map[string]json.RawMessage{}
	if err := json.Unmarshal(input.Schema, &root); err != nil {
		cmp.fail(SelectionStep, "", fmt.Sprintf("invalid stream schema: %v", err))

		return nil, cmp.result
	}

	if raw, ok := root["properties"]; ok {
		if err := json.Unmarshal(raw, &cmp.props); err != nil {
			cmp.fail(SelectionStep, "", fmt.Sprintf("invalid stream schema properties: %v", err))

			return nil, cmp.result
		}
	}

	if raw, ok := root["required"]; ok {
		_ = json.Unmarshal(raw, &cmp.required) //nolint:errcheck // a malformed required list is dropped
	}

	shp := shapeOf(cmp.props)
	isCDC := cmp.has("_ab_cdc_deleted_at") || cmp.has("_ab_cdc_lsn")

	for idx := range input.Steps {
		cmp.step(idx, &input.Steps[idx], isCDC)
	}

	selection := buildProjection(input.SelectedFields)

	cmp.checkPrimaryKey()

	if len(cmp.result.Errors) > 0 {
		return nil, cmp.result
	}

	written := cmp.writtenProjections(selection)

	cmp.result.Schema = outputSchema(root, cmp.props, cmp.required, selection, written)

	return &Chain{
		steps:     cmp.steps,
		input:     shp,
		selection: selection,
		written:   written,
		salt:      input.Salt,
	}, cmp.result
}

func (c *compiler) fail(step int, field, msg string) {
	c.result.Errors = append(c.result.Errors, Issue{Step: step, Field: field, Message: msg})
}

func (c *compiler) has(field string) bool {
	_, ok := c.props[field]

	return ok
}

func (c *compiler) step(idx int, step *Step, isCDC bool) {
	kind, count := step.kind()
	if count != 1 {
		c.fail(idx, "", "step must hold exactly one mapper")

		return
	}

	if !onErrorAllowed(kind, step.OnError) {
		c.fail(idx, "", fmt.Sprintf("on_error %q is not allowed on %s", step.OnError, kind))

		return
	}

	compiled := compiledStep{kind: kind, onError: orFail(step.OnError)}

	switch kind {
	case "rename":
		compiled.renames = c.rename(idx, step.Rename)
	case "cast":
		compiled.casts = c.cast(idx, step.Cast, step.OnError)
	case "set":
		compiled.sets = c.set(idx, step.Set, compiled.onError)
	case "hash":
		compiled.fields = c.scalars(idx, step.Hash, compiled.onError)
	case "mask":
		compiled.masks = c.mask(idx, step.Mask, compiled.onError)
	case "filter":
		compiled.filter = c.filter(idx, *step.Filter, isCDC)
	}

	c.steps = append(c.steps, compiled)
}

func onErrorAllowed(kind, onError string) bool {
	switch kind {
	case "rename":
		return onError == ""
	case "filter":
		return onError == "" || onError == OnErrorFail || onError == OnErrorSkip
	default:
		return onError == "" || onError == OnErrorFail || onError == OnErrorNull || onError == OnErrorSkip
	}
}

func orFail(onError string) string {
	if onError == "" {
		return OnErrorFail
	}

	return onError
}

// ref reports whether a field the step reads exists, with an issue when it does not.
func (c *compiler) ref(idx int, field string) bool {
	if c.has(field) {
		return true
	}

	if by, ok := c.renamed[field]; ok {
		c.fail(idx, field, fmt.Sprintf("field %q was renamed by step %d", field, by))
	} else {
		c.fail(idx, field, fmt.Sprintf("field %q does not exist", field))
	}

	return false
}

// create reports whether the step may write a field: a valid name not created by another step.
func (c *compiler) create(idx int, field string) bool {
	if !validFieldName(field) {
		c.fail(idx, field, fmt.Sprintf("invalid field name %q", field))

		return false
	}

	if by, ok := c.created[field]; ok && by != idx {
		c.fail(idx, field, fmt.Sprintf("field %q already exists", field))

		return false
	}

	return true
}

func (c *compiler) write(idx int, field string, schema json.RawMessage) {
	if !c.has(field) {
		c.created[field] = idx
	}

	c.props[field] = schema
	c.written[field] = true
	delete(c.renamed, field)
	delete(c.origin, field)
}

// writtenProjections maps each output field a step wrote to the part of it the
// output keeps: a field only moved by renames keeps its source's selected
// subtree, any other written field is kept whole (a nil projection).
func (c *compiler) writtenProjections(selection projection) map[string]projection {
	out := map[string]projection{}

	for name := range c.written {
		if !c.has(name) {
			continue
		}

		out[name] = nil

		if source, ok := c.origin[name]; ok && selection != nil {
			if sub, selected := selection[source]; selected {
				out[name] = sub
			}
		}
	}

	return out
}

func validFieldName(name string) bool {
	if name == "" || len(name) > maxFieldNameBytes || strings.TrimSpace(name) != name {
		return false
	}

	return !strings.ContainsFunc(name, unicode.IsControl)
}

func (c *compiler) rename(idx int, renames map[string]string) [][2]string {
	sources := sortedKeys(renames)
	targets := map[string]bool{}
	out := make([][2]string, 0, len(sources))

	for _, from := range sources {
		to := renames[from]
		ok := c.ref(idx, from)

		switch {
		case !validFieldName(to):
			c.fail(idx, to, fmt.Sprintf("invalid field name %q", to))

			ok = false
		case targets[to] || (c.has(to) && !isSource(renames, to)):
			c.fail(idx, to, fmt.Sprintf("field %q already exists", to))

			ok = false
		}

		targets[to] = true

		if ok {
			out = append(out, [2]string{from, to})
		}
	}

	moved := map[string]json.RawMessage{}
	origins := map[string]string{}

	for _, pair := range out {
		moved[pair[1]] = c.props[pair[0]]
		origins[pair[1]] = pair[0]

		if source, ok := c.origin[pair[0]]; ok {
			origins[pair[1]] = source
		} else if c.written[pair[0]] {
			delete(origins, pair[1])
		}

		delete(c.props, pair[0])
		delete(c.created, pair[0])
		delete(c.origin, pair[0])
		c.renamed[pair[0]] = idx
	}

	for _, pair := range out {
		c.write(idx, pair[1], moved[pair[1]])
		delete(c.created, pair[1])

		if source, ok := origins[pair[1]]; ok {
			c.origin[pair[1]] = source
		}
	}

	c.renameReferences(out)

	return out
}

func isSource(renames map[string]string, field string) bool {
	_, ok := renames[field]

	return ok
}

// renameReferences rewrites the primary key and required names of one rename
// step, each looked up once against the names before the step.
func (c *compiler) renameReferences(pairs [][2]string) {
	targets := make(map[string]string, len(pairs))
	for _, pair := range pairs {
		targets[pair[0]] = pair[1]
	}

	for _, path := range c.result.PrimaryKey {
		if len(path) == 0 {
			continue
		}

		if to, ok := targets[path[0]]; ok {
			path[0] = to
		}
	}

	for i, name := range c.required {
		if to, ok := targets[name]; ok {
			c.required[i] = to
		}
	}
}

func (c *compiler) set(idx int, sets map[string]string, onError string) []compiledSet {
	out := make([]compiledSet, 0, len(sets))
	results := map[string]json.RawMessage{}

	env, provider := c.env(idx)
	if env == nil {
		return nil
	}

	for _, field := range sortedKeys(sets) {
		if !c.create(idx, field) {
			continue
		}

		program, typ, readsNull := c.expression(idx, field, env, sets[field])
		if program == nil {
			continue
		}

		out = append(out, compiledSet{field: field, program: program})
		results[field] = withNull(provider.schemaOf(typ), onError == OnErrorNull || readsNull)
	}

	for _, field := range sortedKeys(results) {
		c.write(idx, field, results[field])
	}

	return out
}

func (c *compiler) cast(idx int, casts map[string]CastSpec, stepOnError string) []compiledCast {
	out := make([]compiledCast, 0, len(casts))
	results := map[string]json.RawMessage{}

	for _, field := range sortedKeys(casts) {
		spec := casts[field]
		compiled, ok := c.castField(idx, field, spec, stepOnError)

		if ok {
			out = append(out, compiled)
			canBeNull := compiled.onError == OnErrorNull || compiled.readsNull || (spec.Source == "" && parseNode(c.props[field]).nullable())
			results[field] = withNull(castSchema(spec.To), canBeNull)
		}
	}

	for _, field := range sortedKeys(results) {
		c.write(idx, field, results[field])
	}

	return out
}

func (c *compiler) castField(idx int, field string, spec CastSpec, stepOnError string) (compiledCast, bool) {
	compiled := compiledCast{field: field, spec: spec, onError: orFail(stepOnError)}

	if spec.OnError != "" {
		compiled.onError = spec.OnError
	}

	if !onErrorAllowed("cast", spec.OnError) {
		c.fail(idx, field, fmt.Sprintf("on_error %q is not allowed on cast", spec.OnError))

		return compiled, false
	}

	if castSchema(spec.To) == nil {
		c.fail(idx, field, fmt.Sprintf("unknown cast target %q", spec.To))

		return compiled, false
	}

	format, err := parseTimeFormat(spec.Format)
	if err != nil {
		c.fail(idx, field, err.Error())

		return compiled, false
	}

	compiled.format = format

	compiled.loc, err = loadLocation(spec.Timezone)
	if err != nil {
		c.fail(idx, field, err.Error())

		return compiled, false
	}

	if spec.Source == "" {
		return compiled, c.ref(idx, field)
	}

	if !c.has(field) && !c.create(idx, field) {
		return compiled, false
	}

	env, _ := c.env(idx)
	if env == nil {
		return compiled, false
	}

	compiled.source, _, compiled.readsNull = c.expression(idx, field, env, spec.Source)

	return compiled, compiled.source != nil
}

func (c *compiler) scalars(idx int, fields []string, onError string) []string {
	out := make([]string, 0, len(fields))

	for _, field := range fields {
		if c.ref(idx, field) {
			out = append(out, field)
		}
	}

	for _, field := range out {
		c.write(idx, field, stringSchemaLike(c.props[field], onError == OnErrorNull))
	}

	return out
}

func (c *compiler) mask(idx int, masks map[string]MaskSpec, onError string) map[string]MaskSpec {
	out := map[string]MaskSpec{}

	for _, field := range sortedKeys(masks) {
		spec := masks[field]
		if spec.KeepFirst < 0 || spec.KeepLast < 0 {
			c.fail(idx, field, "keep_first and keep_last must not be negative")

			continue
		}

		if c.ref(idx, field) {
			out[field] = spec
		}
	}

	for _, field := range sortedKeys(out) {
		c.write(idx, field, stringSchemaLike(c.props[field], onError == OnErrorNull))
	}

	return out
}

func (c *compiler) filter(idx int, expr string, isCDC bool) cel.Program {
	env, _ := c.env(idx)
	if env == nil {
		return nil
	}

	program, typ, _ := c.expression(idx, "", env, expr)
	if program == nil {
		return nil
	}

	if typ.Kind() != types.BoolKind && typ.Kind() != types.DynKind {
		c.fail(idx, "", fmt.Sprintf("filter must return bool, got %s", typ))

		return nil
	}

	if isCDC || keyedMode(c.in.DestinationSyncMode) {
		c.result.Warnings = append(c.result.Warnings, Issue{Step: idx, Message: "rows filtered out later stay in the destination"})
	}

	return program
}

func (c *compiler) env(idx int) (*cel.Env, *recordTypes) {
	provider, recordType, err := newRecordTypes(c.props)
	if err != nil {
		c.fail(idx, "", err.Error())

		return nil, nil
	}

	env, err := newEnv(provider, recordType)
	if err != nil {
		c.fail(idx, "", err.Error())

		return nil, nil
	}

	return env, provider
}

// expression compiles one CEL expression against the step input, reporting
// fields renamed away by earlier steps in place of CEL's undefined field error.
func (c *compiler) expression(idx int, field string, env *cel.Env, expr string) (cel.Program, *types.Type, bool) {
	checked, iss := env.Compile(expr)
	if iss.Err() != nil {
		if !c.renamedReference(idx, env, expr) {
			c.fail(idx, field, coalesceMessage(iss.Err().Error()))
		}

		return nil, nil, false
	}

	if err := checkLiteralFormats(checked); err != nil {
		c.fail(idx, field, err.Error())

		return nil, nil, false
	}

	program, err := env.Program(checked, cel.CostLimit(c.in.CostLimit))
	if err != nil {
		c.fail(idx, field, err.Error())

		return nil, nil, false
	}

	typ := checked.OutputType()

	return program, typ, typ.Kind() != types.BoolKind && readsNullable(checked, c.props)
}

func (c *compiler) renamedReference(idx int, env *cel.Env, expr string) bool {
	parsed, iss := env.Parse(expr)
	if iss.Err() != nil {
		return false
	}

	found := false

	for _, field := range referencedFields(parsed) {
		if by, ok := c.renamed[field]; ok && !c.has(field) {
			c.fail(idx, field, fmt.Sprintf("field %q was renamed by step %d", field, by))

			found = true
		}
	}

	return found
}

func (c *compiler) checkPrimaryKey() {
	if len(c.in.SelectedFields) == 0 || !keyedMode(c.in.DestinationSyncMode) {
		return
	}

	for i, path := range c.in.PrimaryKey {
		if len(path) == 0 {
			continue
		}

		current := c.result.PrimaryKey[i]
		if c.written[current[0]] || selectionCovers(c.in.SelectedFields, path) {
			continue
		}

		field := strings.Join(path, ".")
		c.fail(SelectionStep, field, fmt.Sprintf("primary key field %q is not selected", field))
	}
}

// keyedMode reports whether the destination sync mode writes by primary key.
func keyedMode(mode protocol.DestinationSyncMode) bool {
	switch mode {
	case protocol.DestinationSyncModeAppendDedup, protocol.DestinationSyncModeUpdate, protocol.DestinationSyncModeSoftDelete:
		return true
	default:
		return false
	}
}

func selectionCovers(selected []protocol.SelectedField, path []string) bool {
	for _, sel := range selected {
		if len(sel.FieldPath) > 0 && len(sel.FieldPath) <= len(path) && slices.Equal(sel.FieldPath, path[:len(sel.FieldPath)]) {
			return true
		}
	}

	return false
}

func outputSchema(
	root map[string]json.RawMessage,
	props map[string]json.RawMessage,
	required []string,
	selection projection,
	written map[string]projection,
) json.RawMessage {
	out := make(map[string]json.RawMessage, len(root))
	for key, val := range root {
		out[key] = val
	}

	final := props
	if selection != nil {
		final = map[string]json.RawMessage{}

		for name, raw := range props {
			if sub, ok := written[name]; ok {
				final[name] = projectSchema(raw, sub)
			} else if sub, ok := selection[name]; ok {
				final[name] = projectSchema(raw, sub)
			}
		}
	}

	if _, ok := root["properties"]; ok || len(final) > 0 {
		out["properties"] = mustMarshal(final)
	}

	if _, ok := root["required"]; ok {
		kept := []string{}

		for _, name := range required {
			if _, present := final[name]; present {
				kept = append(kept, name)
			}
		}

		out["required"] = mustMarshal(kept)
	}

	return mustMarshal(out)
}

func projectSchema(raw json.RawMessage, sub projection) json.RawMessage {
	if sub == nil {
		return raw
	}

	node := map[string]json.RawMessage{}
	if err := json.Unmarshal(raw, &node); err != nil {
		return raw
	}

	propsRaw, ok := node["properties"]
	if !ok {
		return raw
	}

	props := map[string]json.RawMessage{}
	if err := json.Unmarshal(propsRaw, &props); err != nil {
		return raw
	}

	kept := map[string]json.RawMessage{}

	for name, child := range sub {
		if childRaw, ok := props[name]; ok {
			kept[name] = projectSchema(childRaw, child)
		}
	}

	node["properties"] = mustMarshal(kept)

	return mustMarshal(node)
}

func mustMarshal(value any) json.RawMessage {
	out, err := json.Marshal(value)
	if err != nil {
		panic(fmt.Sprintf("mapper: marshaling schema: %v", err))
	}

	return out
}

func sortedKeys[V any](m map[string]V) []string {
	keys := make([]string, 0, len(m))
	for key := range m {
		keys = append(keys, key)
	}

	slices.Sort(keys)

	return keys
}

func clonePaths(paths [][]string) [][]string {
	if paths == nil {
		return nil
	}

	out := make([][]string, len(paths))
	for i, path := range paths {
		out[i] = slices.Clone(path)
	}

	return out
}

// projection is a tree of selected field paths; a nil subtree keeps the whole value.
type projection map[string]projection

func buildProjection(selected []protocol.SelectedField) projection {
	if len(selected) == 0 {
		return nil
	}

	root := projection{}

	for _, sel := range selected {
		node := root

		for depth, key := range sel.FieldPath {
			if depth == len(sel.FieldPath)-1 {
				node[key] = nil

				break
			}

			child, exists := node[key]
			if exists && child == nil {
				break
			}

			if !exists {
				child = projection{}
				node[key] = child
			}

			node = child
		}
	}

	return root
}

func (p projection) apply(rec map[string]any) map[string]any {
	out := make(map[string]any, len(p))

	for key, sub := range p {
		val, ok := rec[key]
		if !ok {
			continue
		}

		if nested, isMap := val.(map[string]any); isMap && sub != nil {
			out[key] = sub.apply(nested)
		} else {
			out[key] = val
		}
	}

	return out
}

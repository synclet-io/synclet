# Stream mappers

A stream's `mappers` change its records on their way from the source to the destination. `configure_stream`, `configure_streams` and `create_connection` take them per stream, `get_configured_catalog` returns them, and `validate_mappers` checks them without saving.

Before writing mappers:

1. Read the stream's current configuration with `get_configured_catalog` (or, for a connection not created yet, its schema with `get_source_catalog`). Field names and types come from the stream's JSON schema.
2. Build the list from the shapes below.
3. Check it with `validate_mappers`. It returns the errors a save would refuse with, the warnings and the output schema, and runs a `sample_record` through the steps.
4. Save with `configure_stream`, sending the whole stream with the new `mappers`.

## Shape

`mappers` is a JSON array of steps. The steps run top to bottom, once per record; each sees the record as the step before it left it. Every entry of one step reads the record as it was before that step, so a field set in a step can be read only by a later step.

Each step is an object with exactly one kind key, plus an optional `on_error`:

| Key | Value | `on_error` |
|---|---|---|
| `rename` | object: old name → new name | not allowed |
| `cast` | object: field → target type, or field → cast spec | `fail`, `null`, `skip` |
| `set` | object: field → CEL expression | `fail`, `null`, `skip` |
| `hash` | array of field names | `fail`, `null`, `skip` |
| `mask` | object: field → `{"keep_first": n, "keep_last": n}` | `fail`, `null`, `skip` |
| `filter` | one CEL expression returning a boolean | `fail`, `skip` |
| `drop` | array of field names | not allowed |

`on_error` decides what happens when the step fails on one record: `fail` (default) fails the sync, `null` writes null to the field that failed, `skip` drops the record.

After the last step, the stream's `selected_fields` are applied; fields a step writes are always kept. A rename of a primary key or cursor field carries over to the new name.

The examples below use a stream whose records look like this:

```json
{"id": 42, "email": "Ada@Example.COM", "amount": 12.5, "status": "active", "created_at": "2026-03-15T09:30:00Z"}
```

`id` is an integer, `email` a nullable string, `amount` a number, `status` a string and `created_at` a string with format `date-time`.

## rename

Gives fields new names. A new name cannot be an existing field, unless that field is renamed in the same step. Later steps must use the new name.

```json mappers
[{"rename": {"email": "contact_email", "created_at": "created"}}]
```

## set

Writes a field from a CEL expression. The field can be new or existing; it takes the type the expression returns.

```json mappers
[
  {"set": {"amount_cents": "int(r.amount * 100.0)", "label": "string(r.id) + \"-\" + r.status"}},
  {"set": {"email": "coalesce(r.email, \"unknown\").lowerAscii()"}, "on_error": "null"}
]
```

## cast

Converts a field to `string`, `integer`, `number`, `boolean`, `timestamp`, `datetime`, `date`, `object` or `array`. The value is either the target type alone or a spec:

| Key | Applies to | Meaning |
|---|---|---|
| `to` | all | Target type (required). |
| `format` | timestamp, datetime, date, string | `iso8601` (default), `unix`, `unix_ms` or a pattern of the tokens `yyyy yy MM dd HH hh a mm ss SSS XXX` (a literal letter goes in single quotes: `yyyy-MM-dd'T'HH:mm:ss`). For timestamp, datetime and date it is the input format; for string it is the output format of an ISO 8601 input. |
| `timezone` | timestamp, datetime, date | IANA name for values without an offset, such as `Europe/Kyiv`. Default UTC. |
| `decimal_separator`, `thousands_separator` | integer, number | For text such as `1.234,56`. |
| `trim`, `strip_prefix`, `strip_suffix` | integer, number | Remove spaces, a currency sign or a unit before parsing. |
| `true_values`, `false_values` | boolean | Text meaning true or false, case-insensitive. Default `true 1 yes y t` and `false 0 no n f`. |
| `source` | all | CEL expression to cast instead of the field; the field can be new. |
| `on_error` | all | `fail`, `null` or `skip` for this field, overriding the step's. |

A timestamp is written as an RFC 3339 string and a date as `yyyy-MM-dd`. A datetime is written as `yyyy-MM-ddTHH:mm:ss[.fraction]` without an offset, in the cast's timezone (UTC by default); destinations store it as a timestamp without time zone (BigQuery DATETIME). `integer` rejects a fractional value.

```json mappers
[
  {"cast": {
    "id": "string",
    "created_at": {"to": "timestamp", "format": "iso8601", "timezone": "UTC"},
    "created_day": {"to": "string", "format": "dd.MM.yyyy", "source": "r.created_at"},
    "status": {"to": "boolean", "true_values": ["active"], "false_values": ["inactive"], "on_error": "null"}
  }, "on_error": "fail"}
]
```

## hash

Replaces each value with a salted SHA-256 hex string (the salt is per connection). Strings, numbers and booleans are hashed as text; null stays null. The field becomes a string.

```json mappers
[{"hash": ["email"], "on_error": "skip"}]
```

## mask

Replaces characters with `*`, keeping `keep_first` characters at the start and `keep_last` at the end (both default 0, which masks every character). The field becomes a string.

```json mappers
[{"mask": {"email": {"keep_first": 2, "keep_last": 4}, "status": {}}}]
```

## filter

Keeps only the records for which the expression is true. In append + dedup mode, a row that passed the filter in an earlier sync stays in the destination.

```json mappers
[{"filter": "r.status != \"test\" && r.amount > 0.0", "on_error": "skip"}]
```

## drop

Removes fields from the record and from the destination schema. Later steps cannot read them. A primary key field cannot be dropped in append + dedup mode, nor the cursor field of an incremental stream.

Select a field, compute from it, then drop it:

```json mappers
[
  {"set": {"amount_cents": "int(r.amount * 100.0)"}},
  {"drop": ["amount"]}
]
```

## Expressions

`set`, `filter` and a cast's `source` are [CEL](https://cel.dev) expressions over the record `r` (and `sync`, with `sync.started_at`, `sync.job_id`, `sync.connection`).

- Fields: `r.email`, `r.address.city`, `r.tags[0]`. Names must be identifiers; rename others first.
- Types come from the schema: `integer` is int, `number` is double, strings (including date-time) are string. CEL never converts on its own: `r.id * 1.5` is rejected, write `double(r.id) * 1.5`; `r.amount / 2` is rejected, write `r.amount / 2.0`.
- Operators: `+ - * / %`, `== != < <= > >=`, `&& || !`, `in`, `? :`. `+` joins strings and lists.
- Conversions: `int()`, `uint()`, `double()`, `string()`, `bool()`. `int()` of a date-time field is rejected; use `int(parseTimestamp(r.created_at, "iso8601"))`.
- Null and missing: `has(r.f)`; comparing with `null` is rejected, use `coalesce(r.email, "")`. `coalesce(a, b, ...)` takes at least two arguments of one type.
- `catch(value, fallback)` returns `fallback` when `value` fails on a record.
- Strings: `lowerAscii`, `upperAscii`, `trim`, `substring`, `split`, `replace`, `charAt`, `indexOf`, `lastIndexOf`, `join`, `format`, `reverse`, `size`, `contains`, `startsWith`, `endsWith`, `matches` (RE2).
- Lists: `size`, `exists`, `all`, `exists_one`, `map`, `filter`.
- Time: `parseTimestamp(value, format[, timezone])` with the cast formats, `timestamp("...")`, `duration("24h")`, `now()`, getters such as `getFullYear()`, `getHours("Europe/Kyiv")`.
- `parseNumber(value, decimalSeparator)`, `sha256(value)` (unsalted).

## Checks

A save is refused when a step names a field that does not exist or was renamed or dropped by an earlier step, writes a name that already exists, has a syntax or type error, a filter that is not boolean, an unknown cast target, format or timezone, an `on_error` the step does not allow, or leaves a primary key field unselected or dropped in append + dedup mode. Value errors (a bad conversion, a null outside `coalesce`, division by zero) happen per record and follow `on_error`.

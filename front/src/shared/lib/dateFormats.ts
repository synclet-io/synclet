/**
 * The string layouts connector specs give `format: date` and `format: date-time`
 * fields, and their conversion to and from the value of a native date input.
 * Date-time values are read and written as UTC.
 */
export interface DateLayout {
  kind: 'date' | 'date-time'
  sample: string
  /** The value for `<input type="date">` or `<input type="datetime-local">`, or '' when `value` is not in this layout. */
  toInput: (value: string) => string
  fromInput: (input: string) => string
}

const ISO_DATE_RE = /^(\d{4})-(\d{2})-(\d{2})$/
const US_DATE_RE = /^(\d{2})-(\d{2})-(\d{4})$/
const INPUT_DATE_TIME_RE = /^(\d{4}-\d{2}-\d{2})T(\d{2}:\d{2})(?::(\d{2}))?/

function dateTimeLayout(fractionDigits: number): DateLayout {
  const fraction = fractionDigits > 0 ? `.${'0'.repeat(fractionDigits)}` : ''
  return {
    kind: 'date-time',
    sample: `2024-01-02T03:04:05${fraction}Z`,
    toInput: (value) => {
      const m = INPUT_DATE_TIME_RE.exec(value)
      return m ? `${m[1]}T${m[2]}:${m[3] ?? '00'}` : ''
    },
    fromInput: (input) => {
      const m = INPUT_DATE_TIME_RE.exec(input)
      return m ? `${m[1]}T${m[2]}:${m[3] ?? '00'}${fraction}Z` : ''
    },
  }
}

const DATE_LAYOUTS: DateLayout[] = [
  {
    kind: 'date',
    sample: '2024-01-02',
    toInput: value => ISO_DATE_RE.test(value) ? value : '',
    fromInput: input => input,
  },
  {
    kind: 'date',
    sample: '01-02-2024',
    toInput: (value) => {
      const m = US_DATE_RE.exec(value)
      return m ? `${m[3]}-${m[1]}-${m[2]}` : ''
    },
    fromInput: (input) => {
      const m = ISO_DATE_RE.exec(input)
      return m ? `${m[2]}-${m[3]}-${m[1]}` : ''
    },
  },
]

const DATE_TIME_LAYOUTS: DateLayout[] = [dateTimeLayout(0), dateTimeLayout(3), dateTimeLayout(6)]

/**
 * The layout for a field with `format: date` or `date-time`: the first known layout
 * its `pattern` accepts, or the ISO layout when it has no pattern. Undefined when
 * the pattern accepts none of them.
 */
export function dateLayoutFor(format: unknown, pattern?: string): DateLayout | undefined {
  const layouts = format === 'date' ? DATE_LAYOUTS : format === 'date-time' ? DATE_TIME_LAYOUTS : []
  if (!pattern)
    return layouts[0]
  let re: RegExp
  try {
    re = new RegExp(pattern)
  }
  catch {
    return layouts[0]
  }
  return layouts.find(l => re.test(l.sample))
}

/** Whether the native input shows `value` exactly, so writing the input back leaves `value` unchanged. */
export function fitsDateInput(value: string, layout: DateLayout): boolean {
  const input = layout.toInput(value)
  return input !== '' && layout.fromInput(input) === value
}

function daysInMonth(year: number, month: number): number {
  return new Date(Date.UTC(year, month, 0)).getUTCDate()
}

function isRealDate(year: number, month: number, day: number): boolean {
  return month >= 1 && month <= 12 && day >= 1 && day <= daysInMonth(year, month)
}

const LEADING_ISO_DATE_RE = /^(\d{4})-(\d{2})-(\d{2})/
const LEADING_US_DATE_RE = /^(\d{2})-(\d{2})-(\d{4})/
const TIME_RE = /^[T ](\d{2}):(\d{2})(?::(\d{2})(?:\.\d+)?)?(?:Z|[+-](\d{2}):?(\d{2}))?$/i

/**
 * Whether `value` is a real calendar date, optionally followed by a time, for
 * `format`. A `date-time` value may omit the time, as connectors parse a bare
 * date. With `strict`, the value must start with a date and anything after it
 * must be a time (only for `date-time`). Without it, as for a field whose
 * `pattern` has already accepted the value, only the parts recognised are checked.
 */
export function isValidDateValue(value: string, format: 'date' | 'date-time', strict = true): boolean {
  let rest: string
  const iso = LEADING_ISO_DATE_RE.exec(value)
  const us = iso ? null : LEADING_US_DATE_RE.exec(value)
  if (iso) {
    if (!isRealDate(Number(iso[1]), Number(iso[2]), Number(iso[3])))
      return false
    rest = value.slice(iso[0].length)
  }
  else if (us) {
    if (!isRealDate(Number(us[3]), Number(us[1]), Number(us[2])))
      return false
    rest = value.slice(us[0].length)
  }
  else {
    return !strict
  }

  if (rest === '')
    return true
  const time = TIME_RE.exec(rest)
  if (!time)
    return !strict
  if (format === 'date' && strict)
    return false
  return Number(time[1]) < 24 && Number(time[2]) < 60 && Number(time[3] ?? 0) < 60
    && Number(time[4] ?? 0) < 24 && Number(time[5] ?? 0) < 60
}

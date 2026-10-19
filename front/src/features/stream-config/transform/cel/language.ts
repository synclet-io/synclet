import type { StreamParser } from '@codemirror/language'
import { StreamLanguage, StringStream } from '@codemirror/language'
import { tagHighlighter, tags } from '@lezer/highlight'

interface CelState {
  quote: string | null
  raw: boolean
  afterDot: boolean
}

type CelToken = 'comment' | 'string' | 'number' | 'keyword' | 'operator' | 'function' | 'variable' | 'property'

const keywords = new Set(['true', 'false', 'null', 'in'])

const stringStart = /^([rb]{0,2})('''|"""|'|")/i
const hexNumber = /^0x[0-9a-f]+u?/i
const floatNumber = /^(?:\d+\.\d+|\d+(?=e)|\.\d+)(?:e[+-]?\d+)?/i
const intNumber = /^\d+u?/i
const operator = /^(?:==|!=|<=|>=|&&|\|\||[<>!?:+\-*/%])/
const identifier = /^[A-Z_]\w*/i
const callParen = /^\s*\(/
const rawPrefix = /r/i

function eat(stream: StringStream, pattern: RegExp): RegExpMatchArray | null {
  return stream.match(pattern) as RegExpMatchArray | null
}

function readString(stream: StringStream, state: CelState): CelToken {
  const quote = state.quote ?? '\''
  while (!stream.eol()) {
    if (stream.match(quote)) {
      state.quote = null
      return 'string'
    }
    if (stream.next() === '\\' && !state.raw)
      stream.next()
  }
  return 'string'
}

const parser: StreamParser<CelState> = {
  name: 'cel',
  startState: () => ({ quote: null, raw: false, afterDot: false }),
  copyState: s => ({ ...s }),
  token(stream, state): CelToken | null {
    if (state.quote?.length === 1 && stream.sol())
      state.quote = null
    if (state.quote)
      return readString(stream, state)
    if (stream.eatSpace())
      return null

    const wasDot = state.afterDot
    state.afterDot = false

    if (stream.match('//')) {
      stream.skipToEnd()
      return 'comment'
    }
    const str = eat(stream, stringStart)
    if (str) {
      state.quote = str[2]
      state.raw = rawPrefix.test(str[1])
      return readString(stream, state)
    }
    if (eat(stream, hexNumber) || eat(stream, floatNumber) || eat(stream, intNumber))
      return 'number'
    if (eat(stream, operator))
      return 'operator'
    if (eat(stream, identifier)) {
      const word = stream.current()
      if (!wasDot && keywords.has(word))
        return 'keyword'
      if (stream.match(callParen, false))
        return 'function'
      return wasDot ? 'property' : 'variable'
    }
    if (stream.next() === '.')
      state.afterDot = true
    return null
  },
  tokenTable: {
    comment: tags.comment,
    string: tags.string,
    number: tags.number,
    keyword: tags.keyword,
    operator: tags.operator,
    function: tags.function(tags.variableName),
    variable: tags.variableName,
    property: tags.propertyName,
  },
}

export const celLanguage = StreamLanguage.define(parser)

export const celHighlighter = tagHighlighter([
  { tag: tags.comment, class: 'tok-comment' },
  { tag: tags.string, class: 'tok-string' },
  { tag: tags.number, class: 'tok-number' },
  { tag: tags.keyword, class: 'tok-keyword' },
  { tag: tags.operator, class: 'tok-operator' },
  { tag: tags.function(tags.variableName), class: 'tok-function' },
  { tag: tags.variableName, class: 'tok-variable' },
  { tag: tags.propertyName, class: 'tok-property' },
])

export const CEL_TOKEN_CLASSES = ['tok-comment', 'tok-string', 'tok-number', 'tok-keyword', 'tok-operator', 'tok-function', 'tok-variable', 'tok-property']

interface CelTokenSpan {
  text: string
  kind: string
}

function scan(text: string): { tokens: CelTokenSpan[], state: CelState } {
  const state = parser.startState!(2)
  const tokens: CelTokenSpan[] = []
  for (const line of text.split('\n')) {
    const stream = new StringStream(line, 2, 2)
    while (!stream.eol()) {
      stream.start = stream.pos
      const kind = parser.token(stream, state) ?? ''
      tokens.push({ text: stream.current(), kind })
    }
  }
  return { tokens, state }
}

// Splits text into tokens with the tokenizer the editor highlights with; kind is '' for whitespace and punctuation.
export function celTokens(text: string): CelTokenSpan[] {
  return scan(text).tokens
}

// Whether the end of text lies in code, inside a string literal or inside a comment.
export function celContextAt(text: string): 'code' | 'string' | 'comment' {
  const { tokens, state } = scan(text)
  if (state.quote)
    return 'string'
  const lastLine = text.slice(text.lastIndexOf('\n') + 1)
  const last = tokens.at(-1)
  return lastLine !== '' && last?.kind === 'comment' ? 'comment' : 'code'
}

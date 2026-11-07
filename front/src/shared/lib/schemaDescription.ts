import DOMPurify from 'dompurify'

const MARKDOWN_LINK_RE = /\[([^\]]+)\]\(([^)\s]+)\)/g
const AMP_RE = /&/g
const QUOTE_RE = /"/g
const LT_RE = /</g

function escapeAttribute(value: string): string {
  return value.replace(AMP_RE, '&amp;').replace(QUOTE_RE, '&quot;').replace(LT_RE, '&lt;')
}

/**
 * A connector spec description as sanitized HTML. Descriptions mix HTML with
 * markdown links; the links become anchors that open in a new tab.
 */
export function renderDescription(description: string): string {
  const html = description.replace(
    MARKDOWN_LINK_RE,
    (_, label: string, url: string) => `<a href="${escapeAttribute(url)}" target="_blank" rel="noopener noreferrer">${label}</a>`,
  )
  return DOMPurify.sanitize(html, { ADD_ATTR: ['target'] })
}

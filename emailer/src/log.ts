// PII-safe logging helpers. Email addresses are PII; never log them raw.

const EMAIL_RE = /[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}/g

// "ada.lovelace@clave.dev" -> "a***@clave.dev"
export function maskEmail(email: string): string {
  const at = email.indexOf("@")
  if (at <= 0) return "***"
  return `${email[0]}***${email.slice(at)}`
}

// Replace any email addresses found anywhere in a string (e.g. SMTP errors).
export function scrub(input: string): string {
  return input.replace(EMAIL_RE, (m) => maskEmail(m))
}

// Pull a loggable, scrubbed message out of an unknown error.
export function errMessage(error: unknown): string {
  const msg = error instanceof Error ? error.message : String(error)
  return scrub(msg)
}

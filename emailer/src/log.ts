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

type Level = "info" | "warn" | "error"

// Structured, PII-safe logger. String fields are scrubbed of email addresses;
// pass already-masked values (e.g. maskEmail()) for recipients.
function emit(level: Level, msg: string, fields?: Record<string, unknown>) {
  const entry: Record<string, unknown> = {
    ts: new Date().toISOString(),
    level,
    msg,
  }
  if (fields) {
    for (const [k, v] of Object.entries(fields)) {
      entry[k] = typeof v === "string" ? scrub(v) : v
    }
  }
  const line = JSON.stringify(entry)
  if (level === "error") console.error(line)
  else console.log(line)
}

export const log = {
  info: (msg: string, fields?: Record<string, unknown>) => emit("info", msg, fields),
  warn: (msg: string, fields?: Record<string, unknown>) => emit("warn", msg, fields),
  error: (msg: string, fields?: Record<string, unknown>) => emit("error", msg, fields),
}

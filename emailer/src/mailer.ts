import nodemailer from "nodemailer"

const SMTP_URL = process.env.SMTP_URL
const FROM = process.env.EMAIL_FROM ?? "Clave <no-reply@clave.dev>"

if (!SMTP_URL) {
  throw new Error("SMTP_URL env var is required")
}

// SMTP_URL form: smtp://user:pass@host:587  (or smtps:// for implicit TLS)
// Explicit timeouts so an unreachable host fails fast and visibly instead of
// hanging the worker.
const transport = nodemailer.createTransport(SMTP_URL, {
  connectionTimeout: 10_000,
  greetingTimeout: 10_000,
  socketTimeout: 20_000,
})

export async function sendEmail(opts: {
  to: string
  subject: string
  html: string
  text: string
}): Promise<{ messageId: string }> {
  const info = await transport.sendMail({
    from: FROM,
    to: opts.to,
    subject: opts.subject,
    html: opts.html,
    text: opts.text,
  })
  return { messageId: info.messageId }
}

export async function verifyMailer() {
  await transport.verify()
}

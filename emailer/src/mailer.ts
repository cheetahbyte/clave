import nodemailer from "nodemailer"

const SMTP_URL = process.env.SMTP_URL
const FROM = process.env.EMAIL_FROM ?? "Clave <no-reply@clave.dev>"

if (!SMTP_URL) {
  throw new Error("SMTP_URL env var is required")
}

// SMTP_URL form: smtp://user:pass@host:587  (or smtps:// for implicit TLS)
const transport = nodemailer.createTransport(SMTP_URL)

export async function sendEmail(opts: {
  to: string
  subject: string
  html: string
  text: string
}) {
  await transport.sendMail({
    from: FROM,
    to: opts.to,
    subject: opts.subject,
    html: opts.html,
    text: opts.text,
  })
}

export async function verifyMailer() {
  await transport.verify()
}

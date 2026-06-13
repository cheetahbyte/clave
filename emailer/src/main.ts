import { consumeEmailEvents, closeQueue } from "./queue.ts"
import { sendEmail, verifyMailer } from "./mailer.ts"
import { renderEmail } from "./templates.ts"
import { maskEmail, errMessage, log } from "./log.ts"

async function main() {
  await verifyMailer()
  log.info("SMTP transport verified")

  await consumeEmailEvents(async (event) => {
    const { type, email } = event
    const recipient = maskEmail(email)
    const startedAt = Date.now()

    log.info("event received", { type, recipient })

    const rendered = await renderEmail(event)
    const { messageId } = await sendEmail({
      to: email,
      subject: rendered.subject,
      html: rendered.html,
      text: rendered.text,
    })

    log.info("email sent", {
      type,
      recipient,
      messageId,
      durationMs: Date.now() - startedAt,
    })
  })

  log.info("email worker started, waiting for events")
}

async function shutdown(signal: string) {
  log.info("shutting down", { signal })
  try {
    await closeQueue()
  } catch (error) {
    log.error("error during shutdown", { err: errMessage(error) })
  }
  process.exit(0)
}

process.on("SIGINT", () => shutdown("SIGINT"))
process.on("SIGTERM", () => shutdown("SIGTERM"))

main().catch((error) => {
  log.error("fatal error", { err: errMessage(error) })
  process.exit(1)
})

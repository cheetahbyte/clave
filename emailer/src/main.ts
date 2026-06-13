import { consumeEmailEvents, closeQueue } from "./queue.ts"
import { sendEmail, verifyMailer } from "./mailer.ts"
import { renderEmail } from "./templates.ts"
import { maskEmail, errMessage } from "./log.ts"

async function main() {
  await verifyMailer()
  console.log("SMTP transport verified")

  await consumeEmailEvents(async (event) => {
    const { type, email } = event
    console.log(`Processing "${type}" for ${maskEmail(email)}`)

    const rendered = await renderEmail(event)
    await sendEmail({
      to: email,
      subject: rendered.subject,
      html: rendered.html,
      text: rendered.text,
    })

    console.log(`Sent "${type}" to ${maskEmail(email)}`)
  })

  console.log("Email worker started, waiting for events")
}

async function shutdown(signal: string) {
  console.log(`${signal} received, shutting down`)
  try {
    await closeQueue()
  } catch (error) {
    console.error("Error during shutdown:", errMessage(error))
  }
  process.exit(0)
}

process.on("SIGINT", () => shutdown("SIGINT"))
process.on("SIGTERM", () => shutdown("SIGTERM"))

main().catch((error) => {
  console.error("Fatal error:", errMessage(error))
  process.exit(1)
})

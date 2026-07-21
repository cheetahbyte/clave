// One-shot test publisher. Sends a single email event then exits.
// Usage: bun run src/publish-test.ts [event-type] [recipient@example.com]
// Event types: admin.2fa_code, license.created, license.replaced, selfservice.magic_link, organization.invite
import { publishEmailEvent, closeQueue } from "./queue.ts"
import type { EmailEvent } from "./templates.ts"

const eventType = (process.argv[2] ?? "license.created") as EmailEvent["type"]
const to = process.argv[3] ?? "test@example.com"

const event: EmailEvent = (() => {
  switch (eventType) {
    case "admin.2fa_code":
      return {
        type: "admin.2fa_code",
        email: to,
        data: { code: "123456", ttlMinutes: 10 },
      }
    case "license.created":
      return {
        type: "license.created",
        email: to,
        data: {
          licenseKey: "CLAVE-XXXX-YYYY-ZZZZ",
          productName: "Acme Pro",
          portalLink: "https://app.clave.dev/selfservice/acme",
          isTrial: false,
        },
      }
    case "license.replaced":
      return {
        type: "license.replaced",
        email: to,
        data: {
          licenseKey: "CLAVE-AAAA-BBBB-CCCC",
          productName: "Acme Pro",
          portalLink: "https://app.clave.dev/selfservice/acme",
        },
      }
    case "selfservice.magic_link":
      return {
        type: "selfservice.magic_link",
        email: to,
        data: {
          link: "https://app.clave.dev/selfservice/acme/auth?token=abc123",
        },
      }
    case "organization.invite":
      return {
        type: "organization.invite",
        email: to,
        data: {
          orgName: "Acme Corp",
          link: "https://app.clave.dev/invite/abc123",
        },
      }
  }
})()

await publishEmailEvent(event)

console.log(`Published ${eventType} -> ${to}`)

await new Promise((r) => setTimeout(r, 200))
await closeQueue()
process.exit(0)

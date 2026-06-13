// One-shot test publisher. Sends a single license.created event then exits.
// Usage: bun run src/publish-test.ts [recipient@example.com]
import { publishEmailEvent, closeQueue } from "./queue.ts"

const to = process.argv[2] ?? "test@example.com"

await publishEmailEvent({
  type: "license.created",
  email: to,
  data: {
    name: "Ada",
    licenseKey: "",
    dashboardUrl: "https://app.clave.dev/dashboard",
  },
})

console.log(`Published license.created -> ${to}`)

// give the confirm channel a moment to flush, then close
await new Promise((r) => setTimeout(r, 200))
await closeQueue()
process.exit(0)

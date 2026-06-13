import { render, toPlainText } from "react-email"
import LicenseCreated from "./emails/license-created.js"

export type EmailEvent = {
  type: "license.created"
  email: string
  data: { licenseKey: string; productName?: string; portalLink?: string; isTrial?: boolean }
}

export type Rendered = { subject: string; html: string; text: string }

export async function renderEmail(event: EmailEvent): Promise<Rendered> {
  switch (event.type) {
    case "license.created": {
      const d = event.data
      if (!d.licenseKey) throw new Error("license.created requires licenseKey")

      const noun = d.isTrial ? "trial" : "license"
      const subject = d.isTrial ? "Your Clave trial key" : "Your Clave license key"

      const component = LicenseCreated({
        licenseKey: d.licenseKey,
        productName: d.productName,
        portalLink: d.portalLink,
        isTrial: d.isTrial,
      })

      const html = await render(component)
      const text = toPlainText(html)

      return { subject, html, text }
    }
  }
}
